package vendax

import (
	"encoding/json"
	"testing"

	"github.com/msgfy/linktor/internal/domain/entity"
)

// TestBuildInboundEnvelope prova o mapeamento inbound (Linktor -> Core): o vendedor é o usuário
// atribuído; o cliente é o identifier resolvido; tipo/conteúdo/idempotência batem com o evento.
func TestBuildInboundEnvelope(t *testing.T) {
	ev := messageReceivedEvent{
		Type:     "message.received",
		TenantID: "acme",
		Payload: messageReceivedPayload{
			MessageID:      "msg-123",
			ConversationID: "conv-1",
			ContactID:      "contact-1",
			ChannelType:    "whatsapp",
			ContentType:    "text",
			Content:        "Olá, quero fazer um pedido",
		},
	}

	env := buildInboundEnvelope(ev, "vendor-42", "+5511999990001")

	if env.TenantID != "acme" {
		t.Errorf("TenantID = %q, quero acme", env.TenantID)
	}
	if env.VendorID != "vendor-42" {
		t.Errorf("VendorID = %q, quero vendor-42 (o usuário atribuído)", env.VendorID)
	}
	if env.CustomerID != "+5511999990001" {
		t.Errorf("CustomerID = %q, quero o identifier do contato", env.CustomerID)
	}
	if env.Channel != "whatsapp" {
		t.Errorf("Channel = %q, quero whatsapp", env.Channel)
	}
	if env.MessageType != "text" {
		t.Errorf("MessageType = %q, quero text", env.MessageType)
	}
	if env.Content != "Olá, quero fazer um pedido" {
		t.Errorf("Content = %q", env.Content)
	}
	if env.IdempotencyKey != "msg-123" {
		t.Errorf("IdempotencyKey = %q, quero o message.ID (dedup no Core)", env.IdempotencyKey)
	}
}

// TestBuildSendInput prova o mapeamento outbound (Core -> Linktor): a mensagem ao cliente é do
// vendedor (SenderTypeUser); conversa/tenant/conteúdo batem; a idempotencyKey do Core é preservada.
func TestBuildSendInput(t *testing.T) {
	out := LinktorOutbound{
		TenantID:       "acme",
		VendorID:       "vendor-42",
		CustomerID:     "+5511999990001",
		Channel:        "whatsapp",
		ConversationID: "core-conv-abc",
		MessageType:    "text",
		Content:        "Segue sua cotação",
		IdempotencyKey: "out-key-9",
	}

	in := buildSendInput(out, "linktor-conv-1")

	if in.TenantID != "acme" {
		t.Errorf("TenantID = %q", in.TenantID)
	}
	if in.ConversationID != "linktor-conv-1" {
		t.Errorf("ConversationID = %q, quero o id da conversa do Linktor (não a do Core)", in.ConversationID)
	}
	if in.SenderID != "vendor-42" {
		t.Errorf("SenderID = %q, quero o vendedor", in.SenderID)
	}
	if in.SenderType != entity.SenderTypeUser {
		t.Errorf("SenderType = %q, quero user (mensagem do vendedor)", in.SenderType)
	}
	if in.ContentType != entity.ContentTypeText {
		t.Errorf("ContentType = %q, quero text", in.ContentType)
	}
	if in.Content != "Segue sua cotação" {
		t.Errorf("Content = %q", in.Content)
	}
	if in.Metadata["idempotency_key"] != "out-key-9" {
		t.Errorf("idempotency_key = %q, quero preservar a do Core", in.Metadata["idempotency_key"])
	}
	if in.Metadata["source"] != "vendax" {
		t.Errorf("source = %q, quero vendax", in.Metadata["source"])
	}
}

// TestMessageReceivedUnmarshal garante que o envelope real do Linktor (nats.Event com payload) é
// desserializado nos campos que o bridge usa.
func TestMessageReceivedUnmarshal(t *testing.T) {
	raw := []byte(`{
		"type": "message.received",
		"tenant_id": "acme",
		"payload": {
			"message_id": "m1",
			"conversation_id": "c1",
			"contact_id": "ct1",
			"channel_id": "ch1",
			"channel_type": "telegram",
			"content_type": "text",
			"content": "oi"
		},
		"timestamp": "2026-07-08T12:00:00Z"
	}`)

	var ev messageReceivedEvent
	if err := json.Unmarshal(raw, &ev); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if ev.TenantID != "acme" || ev.Payload.MessageID != "m1" || ev.Payload.ChannelType != "telegram" {
		t.Errorf("campos desserializados incorretos: %+v", ev)
	}
}
