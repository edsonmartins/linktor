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
	if env.Channel != "WHATSAPP" {
		t.Errorf("Channel = %q, quero WHATSAPP (vocabulário canônico do Core)", env.Channel)
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

// TestChannelConfigUnmarshal garante que o evento channel.config do Core é desserializado.
func TestChannelConfigUnmarshal(t *testing.T) {
	raw := []byte(`{
		"tenantId": "acme",
		"version": 3,
		"channels": [
			{"id":"wpp-01","type":"WHATSAPP","identifier":"+5511999990001","displayName":"WhatsApp Vendas","status":"ATIVO","settings":{"vendorId":"vendor-42"}}
		]
	}`)
	var cfg channelConfigChanged
	if err := json.Unmarshal(raw, &cfg); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if cfg.TenantID != "acme" || cfg.Version != 3 || len(cfg.Channels) != 1 {
		t.Fatalf("cabeçalho incorreto: %+v", cfg)
	}
	ch := cfg.Channels[0]
	if ch.Type != "WHATSAPP" || ch.Identifier != "+5511999990001" || ch.Status != "ATIVO" {
		t.Errorf("canal incorreto: %+v", ch)
	}
	if ch.Settings["vendorId"] != "vendor-42" {
		t.Errorf("vendorId = %q, quero vendor-42", ch.Settings["vendorId"])
	}
}

// TestShouldApplyIsIdempotent prova o versionamento: só aplica versões estritamente crescentes.
func TestShouldApplyIsIdempotent(t *testing.T) {
	b := &Bridge{appliedVersion: make(map[string]int)}
	if !b.shouldApply("acme", 1) {
		t.Error("v1 deveria aplicar (primeira vez)")
	}
	if b.shouldApply("acme", 1) {
		t.Error("v1 repetida NÃO deveria reaplicar")
	}
	if b.shouldApply("acme", 1) { // replay do NATS
		t.Error("replay de v1 NÃO deveria reaplicar")
	}
	if !b.shouldApply("acme", 2) {
		t.Error("v2 deveria aplicar")
	}
	if b.shouldApply("acme", 1) {
		t.Error("v1 tardia (fora de ordem) NÃO deveria reaplicar")
	}
	if !b.shouldApply("other", 1) {
		t.Error("outro tenant é independente")
	}
}

// TestLinktorChannelTypes prova o mapeamento de tipo Core→Linktor (WhatsApp tem 3 candidatos).
func TestLinktorChannelTypes(t *testing.T) {
	wa := linktorChannelTypes("WHATSAPP")
	if len(wa) != 3 || wa[0] != entity.ChannelTypeWhatsAppOfficial {
		t.Errorf("WHATSAPP = %v, quero [official, unofficial, whatsapp]", wa)
	}
	if got := linktorChannelTypes("MESSENGER"); len(got) != 1 || got[0] != entity.ChannelTypeFacebook {
		t.Errorf("MESSENGER = %v, quero [facebook]", got)
	}
	if got := linktorChannelTypes("TELEGRAM"); got[0] != entity.ChannelTypeTelegram {
		t.Errorf("TELEGRAM = %v", got)
	}
}

// TestCoreChannelType prova a normalização Linktor→Core (os subtipos de WhatsApp colapsam em WHATSAPP)
// e a simetria com linktorChannelTypes.
func TestCoreChannelType(t *testing.T) {
	cases := map[string]string{
		"whatsapp":            "WHATSAPP",
		"whatsapp_official":   "WHATSAPP",
		"whatsapp_unofficial": "WHATSAPP",
		"telegram":            "TELEGRAM",
		"facebook":            "MESSENGER",
		"instagram":           "INSTAGRAM",
		"sms":                 "SMS",
	}
	for linktorType, want := range cases {
		if got := coreChannelType(linktorType); got != want {
			t.Errorf("coreChannelType(%q) = %q, quero %q", linktorType, got, want)
		}
	}
	// simetria: cada subtipo do Linktor volta a estar entre os candidatos do seu tipo canônico
	if types := linktorChannelTypes("WHATSAPP"); len(types) != 3 {
		t.Errorf("WHATSAPP deveria ter 3 subtipos candidatos, tem %d", len(types))
	}
}

// TestDedupe prova a guarda de idempotência do outbound (FIFO com limite).
func TestDedupe(t *testing.T) {
	d := newDedupe(2)
	if d.seenBefore("k1") {
		t.Error("k1 é novo")
	}
	if !d.seenBefore("k1") {
		t.Error("k1 repetido deveria ser visto")
	}
	if d.seenBefore("") {
		t.Error("chave vazia nunca deduplica")
	}
	if d.seenBefore("") {
		t.Error("chave vazia nunca deduplica (2)")
	}
	// enche além do limite (2): k1 já está; adiciona k2, k3 -> k1 é despejado
	d.seenBefore("k2")
	d.seenBefore("k3")
	if d.seenBefore("k1") {
		t.Error("k1 deveria ter sido despejado (FIFO, max=2) e tratado como novo")
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
