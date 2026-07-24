package vendax

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	natsgo "github.com/nats-io/nats.go"

	"github.com/msgfy/linktor/internal/application/usecase"
	"github.com/msgfy/linktor/internal/domain/entity"
	"github.com/msgfy/linktor/pkg/logger"
)

// startOutbound assina (core sub) tenant.*.core.outbound. O Core publica a saída em NATS core
// (plain), então usamos core subscribe.
func (b *Bridge) startOutbound(ctx context.Context) error {
	sub, err := b.nats.Conn().Subscribe(coreOutboundWildcard, func(m *natsgo.Msg) {
		if err := b.handleOutbound(ctx, m.Data); err != nil {
			logger.Error("bridge outbound: falha ao entregar core.outbound: " + err.Error())
		}
	})
	if err != nil {
		return fmt.Errorf("subscribe %s: %w", coreOutboundWildcard, err)
	}
	b.outboundSub = sub
	return nil
}

// handleOutbound entrega a mensagem do vendedor (ou rich object da IA já aceito no VendaX) ao cliente
// no canal. Resolve a conversa do Linktor por (customer, channel) — sem estado — e chama o
// SendMessageUseCase, que revalida tenant/canal/destinatário. A mensagem é sempre do vendedor (user).
func (b *Bridge) handleOutbound(ctx context.Context, data []byte) error {
	var out LinktorOutbound
	if err := json.Unmarshal(data, &out); err != nil {
		return fmt.Errorf("unmarshal outbound: %w", err)
	}

	// Reação (RFC-010): não é uma mensagem de conteúdo — trata em caminho próprio.
	if strings.EqualFold(out.MessageType, "reaction") {
		return b.handleOutboundReaction(ctx, out)
	}

	// Rich objects (quote/suggestion/…) não são entregues ao canal como tal — decisão de produto
	// pendente (ver README §L3). Não vazamos JSON cru ao cliente.
	if !deliverableToChannel(out.MessageType) {
		logger.Warn("bridge outbound: tipo não entregável ao canal ainda; ignorando: " + out.MessageType)
		return nil
	}

	// Idempotência: um retry do outbox do Core reemite a mesma idempotencyKey — não entregar 2×.
	if b.outboundDedupe.seenBefore(out.IdempotencyKey) {
		return nil
	}

	conv, err := b.resolveConversation(ctx, out)
	if err != nil {
		return err
	}
	if conv == nil {
		return fmt.Errorf("conversa Linktor não encontrada (customer=%s channel=%s)", out.CustomerID, out.Channel)
	}

	if _, err = b.sendMessageUC.Execute(ctx, buildSendInput(out, conv.ID)); err != nil {
		return fmt.Errorf("send message: %w", err)
	}
	return nil
}

// handleOutboundReaction entrega uma reação do vendedor a uma mensagem do cliente (RFC-010, Fatia 1):
// resolve a conversa por (customer, channel), acha a mensagem-alvo pelo id do canal
// (TargetChannelMessageId → external id) e chama o SendReaction que já existe. Content vazio = remover.
func (b *Bridge) handleOutboundReaction(ctx context.Context, out LinktorOutbound) error {
	if out.TargetChannelMessageId == "" {
		return fmt.Errorf("reação sem targetChannelMessageId (idem=%s)", out.IdempotencyKey)
	}
	// Idempotência: (idempotencyKey) cobre retries do outbox do Core.
	if b.outboundDedupe.seenBefore(out.IdempotencyKey) {
		return nil
	}

	conv, err := b.resolveConversation(ctx, out)
	if err != nil {
		return err
	}
	if conv == nil {
		return fmt.Errorf("conversa Linktor não encontrada p/ reação (customer=%s channel=%s)", out.CustomerID, out.Channel)
	}

	target, err := b.messageRepo.FindByExternalID(ctx, out.TargetChannelMessageId)
	if err != nil {
		return fmt.Errorf("mensagem-alvo por external id %s: %w", out.TargetChannelMessageId, err)
	}
	if target == nil {
		return fmt.Errorf("mensagem-alvo não encontrada (externalId=%s)", out.TargetChannelMessageId)
	}

	// SendReactionForTenant valida que a conversa é do tenant; content = emoji (vazio remove).
	if err := b.messageService.SendReactionForTenant(
		ctx, out.TenantID, conv.ID, target.ID, out.Content, out.VendorID); err != nil {
		return fmt.Errorf("enviar reação: %w", err)
	}
	return nil
}

// buildSendInput monta o input do SendMessageUseCase a partir do envelope de saída do Core. A
// mensagem que chega ao cliente é sempre do vendedor (SenderTypeUser). Para mídia, o content do Core
// carrega a URL e é entregue como anexo. Pura, para ser testável.
func buildSendInput(out LinktorOutbound, conversationID string) *usecase.SendMessageInput {
	ct := linktorContentType(out.MessageType)
	in := &usecase.SendMessageInput{
		TenantID:       out.TenantID,
		ConversationID: conversationID,
		SenderID:       out.VendorID,
		SenderType:     entity.SenderTypeUser,
		ContentType:    ct,
		Content:        out.Content,
		Metadata: map[string]string{
			"source":          "vendax",
			"idempotency_key": out.IdempotencyKey,
		},
	}
	if isMediaType(ct) {
		in.Attachments = []*usecase.AttachmentInput{{URL: out.Content, Type: string(ct)}}
		in.Content = ""
	}
	return in
}

// resolveConversation reencontra a conversa do Linktor a partir de (customerId, channel) do envelope
// do Core, sem manter tabela de correlação (decisão de arquitetura #5). O channel vem no vocabulário
// canônico do Core (ex.: WHATSAPP); iteramos os subtipos equivalentes do Linktor (official/unofficial/…)
// tanto para o contato quanto para o canal, casando por identifier.
func (b *Bridge) resolveConversation(ctx context.Context, out LinktorOutbound) (*entity.Conversation, error) {
	for _, lt := range linktorChannelTypes(out.Channel) {
		contact, err := b.contactRepo.FindByIdentity(ctx, out.TenantID, string(lt), out.CustomerID)
		if err != nil {
			return nil, fmt.Errorf("contato por identidade (%s/%s): %w", lt, out.CustomerID, err)
		}
		if contact == nil {
			continue
		}
		channels, err := b.channelRepo.FindByType(ctx, out.TenantID, lt)
		if err != nil {
			continue
		}
		for _, ch := range channels {
			conv, err := b.conversationRepo.FindOpenByContactAndChannel(ctx, contact.ID, ch.ID)
			if err != nil {
				return nil, fmt.Errorf("conversa aberta (contato=%s canal=%s): %w", contact.ID, ch.ID, err)
			}
			if conv != nil {
				return conv, nil
			}
		}
	}
	return nil, nil
}
