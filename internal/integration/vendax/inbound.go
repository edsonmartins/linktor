package vendax

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/nats-io/nats.go/jetstream"
	"go.uber.org/zap"

	linktornats "github.com/msgfy/linktor/internal/infrastructure/nats"
	"github.com/msgfy/linktor/pkg/logger"
)

// startInbound cria um consumer JetStream durável em LINKTOR_EVENTS filtrando message.received e
// republica cada mensagem do cliente como LinktorEnvelope no subject de inbound do Core.
func (b *Bridge) startInbound(ctx context.Context) error {
	js := b.nats.JetStream()
	stream, err := js.Stream(ctx, linktornats.StreamEvents)
	if err != nil {
		return fmt.Errorf("stream %s: %w", linktornats.StreamEvents, err)
	}
	consumer, err := stream.CreateOrUpdateConsumer(ctx, jetstream.ConsumerConfig{
		Name:          durableInbound,
		Durable:       durableInbound,
		FilterSubject: linktornats.SubjectEvent(linktornats.EventMessageReceived),
		AckPolicy:     jetstream.AckExplicitPolicy,
		MaxDeliver:    5,
		AckWait:       30 * time.Second,
		DeliverPolicy: jetstream.DeliverAllPolicy,
	})
	if err != nil {
		return fmt.Errorf("consumer %s: %w", durableInbound, err)
	}

	b.wg.Add(1)
	go func() {
		defer b.wg.Done()
		for {
			select {
			case <-ctx.Done():
				return
			default:
				msgs, err := consumer.Fetch(10, jetstream.FetchMaxWait(5*time.Second))
				if err != nil {
					if ctx.Err() != nil {
						return
					}
					continue
				}
				for msg := range msgs.Messages() {
					if err := b.handleMessageReceived(ctx, msg.Data()); err != nil {
						logger.Error("bridge inbound: falha ao processar message.received: " + err.Error())
						_ = msg.Nak()
						continue
					}
					_ = msg.Ack()
				}
			}
		}
	}()
	return nil
}

// handleMessageReceived traduz o evento interno em LinktorEnvelope e o publica (core pub/sub) para
// o Core. O vendedor é o usuário atribuído à conversa; sem vendedor, a mensagem é ignorada (no L0,
// 1 canal = 1 vendedor, então a atribuição existe).
func (b *Bridge) handleMessageReceived(ctx context.Context, data []byte) error {
	var ev messageReceivedEvent
	if err := json.Unmarshal(data, &ev); err != nil {
		return fmt.Errorf("unmarshal evento: %w", err)
	}
	p := ev.Payload

	conv, err := b.conversationRepo.FindByID(ctx, p.ConversationID)
	if err != nil {
		return fmt.Errorf("conversa %s: %w", p.ConversationID, err)
	}
	if conv == nil {
		return fmt.Errorf("conversa %s não encontrada", p.ConversationID)
	}
	// Defense-in-depth: a conversa tem de pertencer ao tenant do evento (nunca cruzar tenants).
	if conv.TenantID != ev.TenantID {
		return fmt.Errorf("conversa %s do tenant %s não bate com o evento (%s)",
			p.ConversationID, conv.TenantID, ev.TenantID)
	}

	vendorID := ""
	if conv.AssignedUserID != nil {
		vendorID = *conv.AssignedUserID
	}
	if vendorID == "" {
		// Fallback "1 canal = 1 vendedor": usa o vendedor-dono do canal (channel.config, L1).
		vendorID = b.channelVendorID(ctx, p.ChannelID)
	}
	if vendorID == "" {
		// Sem vendedor não há a quem entregar no VendaX; não vazamos a mensagem ao Core.
		logger.Warn("bridge inbound: conversa sem vendedor (nem atribuído nem por canal); ignorando",
			zap.String("conversation_id", p.ConversationID))
		return nil
	}

	// Reação de canal do cliente (RFC-010 Fatia 2): caminho próprio, não é mensagem de conteúdo.
	if p.IsReaction == "true" {
		return b.publishInboundReaction(ctx, ev, vendorID)
	}

	env := buildInboundEnvelope(ev, vendorID, b.resolveCustomerID(ctx, p.ContactID, p.ChannelType))
	payload, err := json.Marshal(env)
	if err != nil {
		return fmt.Errorf("marshal envelope: %w", err)
	}
	// core publish -> o dispatcher core do Core (CoreNatsSubscriptions) recebe.
	if err := b.nats.Conn().Publish(coreInboundSubject(ev.TenantID), payload); err != nil {
		return fmt.Errorf("publish inbound: %w", err)
	}
	return nil
}

// publishInboundReaction traduz uma reação do cliente em envelope de reação para o Core. Resolve a
// mensagem-alvo pelo id do canal (reaction_message_id → external id) e envia a chave que o Core já
// conhece (idempotencyKey do envio, ou o id interno para msg do cliente) — assim o Core correlaciona
// sem precisar guardar channel_message_id das mensagens de saída (RFC-010 Fatia 2, sem L-C).
func (b *Bridge) publishInboundReaction(ctx context.Context, ev messageReceivedEvent, vendorID string) error {
	p := ev.Payload
	if p.ReactionMessageID == "" {
		logger.Warn("bridge inbound: reação sem reaction_message_id; ignorando")
		return nil
	}
	target, err := b.messageRepo.FindByExternalID(ctx, p.ReactionMessageID)
	if err != nil {
		return fmt.Errorf("alvo da reação por external id %s: %w", p.ReactionMessageID, err)
	}
	if target == nil {
		logger.Warn("bridge inbound: alvo da reação não encontrado (externalId=" + p.ReactionMessageID + ")")
		return nil
	}
	targetKey := target.Metadata["idempotency_key"] // msg do vendedor enviada via Core
	if targetKey == "" {
		targetKey = target.ID // msg do cliente: no Core, idempotencyKey = message.ID do Linktor
	}

	env := LinktorEnvelope{
		TenantID:             ev.TenantID,
		VendorID:             vendorID,
		CustomerID:           b.resolveCustomerID(ctx, p.ContactID, p.ChannelType),
		Channel:              coreChannelType(p.ChannelType),
		MessageType:          "reaction",
		Content:              p.Content, // emoji (vazio = remoção)
		IdempotencyKey:       p.MessageID,
		Environment:          p.Environment,
		TargetIdempotencyKey: targetKey,
	}
	payload, err := json.Marshal(env)
	if err != nil {
		return fmt.Errorf("marshal envelope reação: %w", err)
	}
	if err := b.nats.Conn().Publish(coreInboundSubject(ev.TenantID), payload); err != nil {
		return fmt.Errorf("publish reação inbound: %w", err)
	}
	return nil
}

// channelVendorID lê o vendedor-dono do canal gravado pelo channel.config (L1). "" se não houver.
func (b *Bridge) channelVendorID(ctx context.Context, channelID string) string {
	ch, err := b.channelRepo.FindByID(ctx, channelID)
	if err != nil || ch == nil {
		return ""
	}
	return ch.Config[channelVendorConfigKey]
}

// resolveCustomerID devolve o identifier do contato para o canal (telefone/handle) — a chave que o
// Core usa para o auto-link com o cliente do ERP (CC-09). Fallback: telefone do contato, depois o id.
func (b *Bridge) resolveCustomerID(ctx context.Context, contactID, channelType string) string {
	if identities, err := b.contactRepo.FindIdentitiesByContact(ctx, contactID); err == nil {
		for _, id := range identities {
			if id.ChannelType == channelType && id.Identifier != "" {
				return id.Identifier
			}
		}
	}
	if c, err := b.contactRepo.FindByID(ctx, contactID); err == nil && c != nil && c.Phone != "" {
		return c.Phone
	}
	return contactID
}
