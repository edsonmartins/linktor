package vendax

import "github.com/msgfy/linktor/internal/domain/entity"

// LinktorEnvelope é o envelope normalizado que o bridge publica para o VendaX Core em
// tenant.{id}.linktor.inbound. Espelha br.com.vendax.core...dto.LinktorEnvelope (7 campos String).
// O Core resolve a conversa por (vendorId, customerId, channel) e deduplica por idempotencyKey.
type LinktorEnvelope struct {
	TenantID       string `json:"tenantId"`
	VendorID       string `json:"vendorId"`
	CustomerID     string `json:"customerId"`
	Channel        string `json:"channel"`
	MessageType    string `json:"messageType"`
	Content        string `json:"content"`
	IdempotencyKey string `json:"idempotencyKey"`
	// Environment ("production"|"sandbox") marca origem sintética (INV-018).
	// Campo ADITIVO ao contrato — precisa entrar no DTO Java do Core antes do
	// freeze do plano de integração (status PROPOSTA em 2026-07-23); ausência
	// significa production para consumidores antigos.
	Environment string `json:"environment,omitempty"`
}

// LinktorOutbound é o envelope que o Core publica em tenant.{id}.core.outbound e o bridge consome
// para entregar no canal. Espelha br.com.vendax.core...dto.LinktorOutbound (inclui conversationId,
// que é o id da conversa no Core — o bridge re-resolve a conversa do Linktor por (customer, channel)).
type LinktorOutbound struct {
	TenantID       string `json:"tenantId"`
	VendorID       string `json:"vendorId"`
	CustomerID     string `json:"customerId"`
	Channel        string `json:"channel"`
	ConversationID string `json:"conversationId"`
	MessageType    string `json:"messageType"`
	Content        string `json:"content"`
	IdempotencyKey string `json:"idempotencyKey"`
}

// messageReceivedEvent é o envelope do evento interno do Linktor (nats.Event) publicado em
// linktor.events.message.received. O produtor não expõe struct tipada; declaramos a nossa.
type messageReceivedEvent struct {
	Type     string                 `json:"type"`
	TenantID string                 `json:"tenant_id"`
	Payload  messageReceivedPayload `json:"payload"`
}

// buildInboundEnvelope monta o envelope do Core a partir do evento interno + a identidade resolvida
// (vendedor = usuário atribuído; customer = identifier do contato). Pura, para ser testável.
func buildInboundEnvelope(ev messageReceivedEvent, vendorID, customerID string) LinktorEnvelope {
	p := ev.Payload
	// Para mídia sem legenda, o texto vem vazio; carrega a URL do anexo como referência (o Core não
	// perde a mensagem; a transcrição de áudio é caminho separado no Core — ver README §L3).
	content := p.Content
	if content == "" && len(p.Attachments) > 0 {
		content = p.Attachments[0].URL
	}
	return LinktorEnvelope{
		TenantID:       ev.TenantID,
		VendorID:       vendorID,
		CustomerID:     customerID,
		Channel:        coreChannelType(p.ChannelType),                     // vocabulário canônico do Core
		MessageType:    coreMessageType(entity.ContentType(p.ContentType)), // ADR-010
		Content:        content,
		IdempotencyKey: p.MessageID,
		Environment:    p.Environment,
	}
}

// messageReceivedPayload são os campos do payload de message.received (buildMessageReceivedOutboxEvent).
type messageReceivedPayload struct {
	MessageID      string       `json:"message_id"`
	ConversationID string       `json:"conversation_id"`
	ContactID      string       `json:"contact_id"`
	ChannelID      string       `json:"channel_id"`
	ChannelType    string       `json:"channel_type"`
	ContentType    string       `json:"content_type"`
	Content        string       `json:"content"`
	ExternalID     string       `json:"external_id"`
	SenderID       string       `json:"sender_id"`
	SenderName     string       `json:"sender_name"`
	Environment    string       `json:"environment"`
	Attachments    []attachment `json:"attachments"`
}

// attachment é um anexo de mídia do evento message.received (subconjunto usado pelo bridge).
type attachment struct {
	URL      string `json:"url"`
	MimeType string `json:"mime_type"`
	Filename string `json:"filename"`
}
