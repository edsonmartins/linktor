package nats

import "fmt"

// Stream names
const (
	StreamMessages = "LINKTOR_MESSAGES"
	StreamEvents   = "LINKTOR_EVENTS"
	StreamWebhooks = "LINKTOR_WEBHOOKS"
)

// Subject patterns for messages
const (
	// Inbound messages (from external channels to Linktor)
	SubjectInboundAll     = "linktor.messages.inbound.>"
	SubjectInboundPattern = "linktor.messages.inbound.%s" // %s = channel_type

	// Outbound messages (from Linktor to external channels)
	SubjectOutboundAll     = "linktor.messages.outbound.>"
	SubjectOutboundPattern = "linktor.messages.outbound.%s" // %s = channel_type

	// Message status updates
	SubjectStatusAll     = "linktor.messages.status.>"
	SubjectStatusPattern = "linktor.messages.status.%s" // %s = channel_type
)

// Subject patterns for events
const (
	SubjectEventsAll     = "linktor.events.>"
	SubjectEventsPattern = "linktor.events.%s" // %s = event_type
)

// Subject patterns for webhooks
const (
	SubjectWebhooksAll     = "linktor.webhooks.>"
	SubjectWebhooksPattern = "linktor.webhooks.%s" // %s = tenant_id
)

// Consumer names
const (
	ConsumerInbound        = "inbound-processor"
	ConsumerOutboundPrefix = "outbound-" // + channel_type
	ConsumerStatus         = "status-processor"
	ConsumerWebhooks       = "webhook-delivery"
	ConsumerEvents         = "event-processor"
	ConsumerAIAnalyzer     = "ai-analyzer"
	ConsumerAIResponder    = "ai-responder"
	ConsumerAIEscalation   = "ai-escalation"
)

// Stream for AI/Bot processing
const (
	StreamAI = "LINKTOR_AI"
)

// Subject patterns for AI/Bot processing
const (
	// Bot message analysis
	SubjectBotAnalyzeAll     = "linktor.bot.analyze.>"
	SubjectBotAnalyzePattern = "linktor.bot.analyze.%s" // %s = tenant_id

	// Bot responses
	SubjectBotResponseAll     = "linktor.bot.response.>"
	SubjectBotResponsePattern = "linktor.bot.response.%s" // %s = tenant_id

	// Escalation
	SubjectBotEscalateAll     = "linktor.bot.escalate.>"
	SubjectBotEscalatePattern = "linktor.bot.escalate.%s" // %s = tenant_id

	// Intent classification
	SubjectBotIntent = "linktor.bot.intent"

	// Context updates
	SubjectBotContext = "linktor.bot.context"
)

// Event types
const (
	EventMessageReceived  = "message.received"
	EventMessageSent      = "message.sent"
	EventMessageDelivered = "message.delivered"
	EventMessageRead      = "message.read"
	EventMessageFailed    = "message.failed"
	EventMessageAnalyzed  = "message.analyzed"

	EventConversationCreated   = "conversation.created"
	EventConversationAssigned  = "conversation.assigned"
	EventConversationResolved  = "conversation.resolved"
	EventConversationReopened  = "conversation.reopened"
	EventConversationEscalated = "conversation.escalated"

	EventContactCreated = "contact.created"
	EventContactUpdated = "contact.updated"

	EventChannelConnected    = "channel.connected"
	EventChannelDisconnected = "channel.disconnected"
	EventChannelError        = "channel.error"

	// AI/Bot events
	EventBotResponse   = "bot.response"
	EventBotEscalation = "bot.escalation"
	EventBotAnalysis   = "bot.analysis"
)

// SubjectInbound returns the subject for inbound messages of a channel type
func SubjectInbound(channelType string) string {
	return fmt.Sprintf(SubjectInboundPattern, channelType)
}

// SubjectOutbound returns the subject for outbound messages of a channel type
func SubjectOutbound(channelType string) string {
	return fmt.Sprintf(SubjectOutboundPattern, channelType)
}

// SubjectStatus returns the subject for status updates of a channel type
func SubjectStatus(channelType string) string {
	return fmt.Sprintf(SubjectStatusPattern, channelType)
}

// SubjectEvent returns the subject for a specific event type
func SubjectEvent(eventType string) string {
	return fmt.Sprintf(SubjectEventsPattern, eventType)
}

// SubjectWebhook returns the subject for webhook delivery to a tenant
func SubjectWebhook(tenantID string) string {
	return fmt.Sprintf(SubjectWebhooksPattern, tenantID)
}

// ConsumerOutbound returns the consumer name for a channel type
func ConsumerOutbound(channelType string) string {
	return ConsumerOutboundPrefix + channelType
}

// SubjectBotAnalyze returns the subject for bot analysis for a tenant
func SubjectBotAnalyze(tenantID string) string {
	return fmt.Sprintf(SubjectBotAnalyzePattern, tenantID)
}

// SubjectBotResponse returns the subject for bot responses for a tenant
func SubjectBotResponse(tenantID string) string {
	return fmt.Sprintf(SubjectBotResponsePattern, tenantID)
}

// SubjectBotEscalate returns the subject for bot escalation for a tenant
func SubjectBotEscalate(tenantID string) string {
	return fmt.Sprintf(SubjectBotEscalatePattern, tenantID)
}
