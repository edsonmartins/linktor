package vendax

import (
	"context"
	"encoding/json"
	"fmt"

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

// buildSendInput monta o input do SendMessageUseCase a partir do envelope de saída do Core. A
// mensagem que chega ao cliente é sempre do vendedor (SenderTypeUser). Pura, para ser testável.
func buildSendInput(out LinktorOutbound, conversationID string) *usecase.SendMessageInput {
	return &usecase.SendMessageInput{
		TenantID:       out.TenantID,
		ConversationID: conversationID,
		SenderID:       out.VendorID,
		SenderType:     entity.SenderTypeUser,
		ContentType:    entity.ContentType(out.MessageType),
		Content:        out.Content,
		Metadata: map[string]string{
			"source":          "vendax",
			"idempotency_key": out.IdempotencyKey,
		},
	}
}

// resolveConversation reencontra a conversa do Linktor a partir de (customerId, channel) do envelope
// do Core, sem manter tabela de correlação (decisão de arquitetura #5). No L0 há 1 canal por tipo.
func (b *Bridge) resolveConversation(ctx context.Context, out LinktorOutbound) (*entity.Conversation, error) {
	contact, err := b.contactRepo.FindByIdentity(ctx, out.TenantID, out.Channel, out.CustomerID)
	if err != nil {
		return nil, fmt.Errorf("contato por identidade (%s/%s): %w", out.Channel, out.CustomerID, err)
	}
	if contact == nil {
		return nil, fmt.Errorf("contato não encontrado (channel=%s id=%s)", out.Channel, out.CustomerID)
	}

	channels, err := b.channelRepo.FindByType(ctx, out.TenantID, entity.ChannelType(out.Channel))
	if err != nil {
		return nil, fmt.Errorf("canais do tipo %s: %w", out.Channel, err)
	}
	if len(channels) == 0 {
		return nil, fmt.Errorf("nenhum canal do tipo %s no tenant %s", out.Channel, out.TenantID)
	}

	// L0: 1 canal por tipo. Procura a conversa aberta do contato em qualquer canal desse tipo.
	for _, ch := range channels {
		conv, err := b.conversationRepo.FindOpenByContactAndChannel(ctx, contact.ID, ch.ID)
		if err != nil {
			return nil, fmt.Errorf("conversa aberta (contato=%s canal=%s): %w", contact.ID, ch.ID, err)
		}
		if conv != nil {
			return conv, nil
		}
	}
	return nil, nil
}
