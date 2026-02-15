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

// =============================================================================
// WhatsApp-specific subjects (channel-specific, not generic)
// =============================================================================

// Stream for WhatsApp-specific events
const StreamWhatsApp = "LINKTOR_WHATSAPP"

// Subject patterns for WhatsApp-specific events
const (
	// Template events
	SubjectWhatsAppTemplateStatusAll     = "linktor.whatsapp.template.status.>"
	SubjectWhatsAppTemplateStatusPattern = "linktor.whatsapp.template.status.%s" // %s = tenant_id
	SubjectWhatsAppTemplateQualityAll     = "linktor.whatsapp.template.quality.>"
	SubjectWhatsAppTemplateQualityPattern = "linktor.whatsapp.template.quality.%s"
	SubjectWhatsAppTemplateCategoryAll    = "linktor.whatsapp.template.category.>"
	SubjectWhatsAppTemplateCategoryPattern = "linktor.whatsapp.template.category.%s"

	// Account events
	SubjectWhatsAppAccountAlertAll     = "linktor.whatsapp.account.alert.>"
	SubjectWhatsAppAccountAlertPattern = "linktor.whatsapp.account.alert.%s"
	SubjectWhatsAppAccountUpdateAll    = "linktor.whatsapp.account.update.>"
	SubjectWhatsAppAccountUpdatePattern = "linktor.whatsapp.account.update.%s"
	SubjectWhatsAppAccountReviewAll    = "linktor.whatsapp.account.review.>"
	SubjectWhatsAppAccountReviewPattern = "linktor.whatsapp.account.review.%s"

	// Phone number events
	SubjectWhatsAppPhoneNameAll      = "linktor.whatsapp.phone.name.>"
	SubjectWhatsAppPhoneNamePattern  = "linktor.whatsapp.phone.name.%s"
	SubjectWhatsAppPhoneQualityAll   = "linktor.whatsapp.phone.quality.>"
	SubjectWhatsAppPhoneQualityPattern = "linktor.whatsapp.phone.quality.%s"

	// Flow events
	SubjectWhatsAppFlowAll     = "linktor.whatsapp.flow.>"
	SubjectWhatsAppFlowPattern = "linktor.whatsapp.flow.%s"

	// Security events
	SubjectWhatsAppSecurityAll     = "linktor.whatsapp.security.>"
	SubjectWhatsAppSecurityPattern = "linktor.whatsapp.security.%s"

	// Business capability events
	SubjectWhatsAppCapabilityAll     = "linktor.whatsapp.capability.>"
	SubjectWhatsAppCapabilityPattern = "linktor.whatsapp.capability.%s"

	// Message echo events (messages sent via Business app)
	SubjectWhatsAppEchoAll     = "linktor.whatsapp.echo.>"
	SubjectWhatsAppEchoPattern = "linktor.whatsapp.echo.%s"
)

// WhatsApp-specific event types
const (
	EventWhatsAppTemplateApproved     = "whatsapp.template.approved"
	EventWhatsAppTemplateRejected     = "whatsapp.template.rejected"
	EventWhatsAppTemplatePaused       = "whatsapp.template.paused"
	EventWhatsAppTemplateDisabled     = "whatsapp.template.disabled"
	EventWhatsAppTemplateQualityGreen = "whatsapp.template.quality.green"
	EventWhatsAppTemplateQualityYellow = "whatsapp.template.quality.yellow"
	EventWhatsAppTemplateQualityRed   = "whatsapp.template.quality.red"

	EventWhatsAppAccountAlert   = "whatsapp.account.alert"
	EventWhatsAppAccountBanned  = "whatsapp.account.banned"
	EventWhatsAppAccountReview  = "whatsapp.account.review"

	EventWhatsAppPhoneFlagged   = "whatsapp.phone.flagged"
	EventWhatsAppPhoneUnflagged = "whatsapp.phone.unflagged"
	EventWhatsAppPhoneNameApproved = "whatsapp.phone.name.approved"
	EventWhatsAppPhoneNameRejected = "whatsapp.phone.name.rejected"

	EventWhatsAppFlowStatusChange = "whatsapp.flow.status"
	EventWhatsAppFlowError        = "whatsapp.flow.error"

	EventWhatsAppSecurity = "whatsapp.security"

	EventWhatsAppCapabilityUpdate = "whatsapp.capability.update"

	EventWhatsAppEcho = "whatsapp.echo"
)

// Consumer names for WhatsApp-specific events
const (
	ConsumerWhatsAppTemplates  = "whatsapp-templates"
	ConsumerWhatsAppAccount    = "whatsapp-account"
	ConsumerWhatsAppPhone      = "whatsapp-phone"
	ConsumerWhatsAppFlows      = "whatsapp-flows"
	ConsumerWhatsAppSecurity   = "whatsapp-security"
	ConsumerWhatsAppCapability = "whatsapp-capability"
	ConsumerWhatsAppEcho       = "whatsapp-echo"
)

// Helper functions for WhatsApp-specific subjects

// SubjectWhatsAppTemplateStatus returns the subject for template status updates
func SubjectWhatsAppTemplateStatus(tenantID string) string {
	return fmt.Sprintf(SubjectWhatsAppTemplateStatusPattern, tenantID)
}

// SubjectWhatsAppTemplateQuality returns the subject for template quality updates
func SubjectWhatsAppTemplateQuality(tenantID string) string {
	return fmt.Sprintf(SubjectWhatsAppTemplateQualityPattern, tenantID)
}

// SubjectWhatsAppAccountAlert returns the subject for account alerts
func SubjectWhatsAppAccountAlert(tenantID string) string {
	return fmt.Sprintf(SubjectWhatsAppAccountAlertPattern, tenantID)
}

// SubjectWhatsAppPhoneQuality returns the subject for phone quality updates
func SubjectWhatsAppPhoneQuality(tenantID string) string {
	return fmt.Sprintf(SubjectWhatsAppPhoneQualityPattern, tenantID)
}

// SubjectWhatsAppFlow returns the subject for flow events
func SubjectWhatsAppFlow(tenantID string) string {
	return fmt.Sprintf(SubjectWhatsAppFlowPattern, tenantID)
}

// SubjectWhatsAppSecurity returns the subject for security events
func SubjectWhatsAppSecurity(tenantID string) string {
	return fmt.Sprintf(SubjectWhatsAppSecurityPattern, tenantID)
}
