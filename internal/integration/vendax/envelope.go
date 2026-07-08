package vendax

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
	return LinktorEnvelope{
		TenantID:       ev.TenantID,
		VendorID:       vendorID,
		CustomerID:     customerID,
		Channel:        ev.Payload.ChannelType,
		MessageType:    ev.Payload.ContentType,
		Content:        ev.Payload.Content,
		IdempotencyKey: ev.Payload.MessageID,
	}
}

// messageReceivedPayload são os campos do payload de message.received (buildMessageReceivedOutboxEvent).
type messageReceivedPayload struct {
	MessageID      string `json:"message_id"`
	ConversationID string `json:"conversation_id"`
	ContactID      string `json:"contact_id"`
	ChannelID      string `json:"channel_id"`
	ChannelType    string `json:"channel_type"`
	ContentType    string `json:"content_type"`
	Content        string `json:"content"`
	ExternalID     string `json:"external_id"`
	SenderID       string `json:"sender_id"`
	SenderName     string `json:"sender_name"`
}
