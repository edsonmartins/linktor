package nats

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSubjectInbound(t *testing.T) {
	tests := []struct {
		channelType string
		expected    string
	}{
		{"whatsapp", "linktor.messages.inbound.whatsapp"},
		{"telegram", "linktor.messages.inbound.telegram"},
		{"sms", "linktor.messages.inbound.sms"},
		{"email", "linktor.messages.inbound.email"},
		{"webchat", "linktor.messages.inbound.webchat"},
	}
	for _, tt := range tests {
		t.Run(tt.channelType, func(t *testing.T) {
			assert.Equal(t, tt.expected, SubjectInbound(tt.channelType))
		})
	}
}

func TestSubjectOutbound(t *testing.T) {
	tests := []struct {
		channelType string
		expected    string
	}{
		{"whatsapp", "linktor.messages.outbound.whatsapp"},
		{"telegram", "linktor.messages.outbound.telegram"},
		{"sms", "linktor.messages.outbound.sms"},
		{"email", "linktor.messages.outbound.email"},
	}
	for _, tt := range tests {
		t.Run(tt.channelType, func(t *testing.T) {
			assert.Equal(t, tt.expected, SubjectOutbound(tt.channelType))
		})
	}
}

func TestSubjectStatus(t *testing.T) {
	tests := []struct {
		channelType string
		expected    string
	}{
		{"whatsapp", "linktor.messages.status.whatsapp"},
		{"telegram", "linktor.messages.status.telegram"},
		{"sms", "linktor.messages.status.sms"},
	}
	for _, tt := range tests {
		t.Run(tt.channelType, func(t *testing.T) {
			assert.Equal(t, tt.expected, SubjectStatus(tt.channelType))
		})
	}
}

func TestSubjectEvent(t *testing.T) {
	tests := []struct {
		eventType string
		expected  string
	}{
		{"message.received", "linktor.events.message.received"},
		{"message.sent", "linktor.events.message.sent"},
		{"conversation.created", "linktor.events.conversation.created"},
		{"contact.created", "linktor.events.contact.created"},
		{"channel.connected", "linktor.events.channel.connected"},
		{"bot.response", "linktor.events.bot.response"},
	}
	for _, tt := range tests {
		t.Run(tt.eventType, func(t *testing.T) {
			assert.Equal(t, tt.expected, SubjectEvent(tt.eventType))
		})
	}
}

func TestSubjectWebhook(t *testing.T) {
	tests := []struct {
		tenantID string
		expected string
	}{
		{"tenant-123", "linktor.webhooks.tenant-123"},
		{"abc-def-ghi", "linktor.webhooks.abc-def-ghi"},
		{"t1", "linktor.webhooks.t1"},
	}
	for _, tt := range tests {
		t.Run(tt.tenantID, func(t *testing.T) {
			assert.Equal(t, tt.expected, SubjectWebhook(tt.tenantID))
		})
	}
}

func TestConsumerOutbound(t *testing.T) {
	tests := []struct {
		channelType string
		expected    string
	}{
		{"whatsapp", "outbound-whatsapp"},
		{"telegram", "outbound-telegram"},
		{"sms", "outbound-sms"},
	}
	for _, tt := range tests {
		t.Run(tt.channelType, func(t *testing.T) {
			assert.Equal(t, tt.expected, ConsumerOutbound(tt.channelType))
		})
	}
}

func TestSubjectBotAnalyze(t *testing.T) {
	tests := []struct {
		tenantID string
		expected string
	}{
		{"tenant-1", "linktor.bot.analyze.tenant-1"},
		{"t1", "linktor.bot.analyze.t1"},
	}
	for _, tt := range tests {
		t.Run(tt.tenantID, func(t *testing.T) {
			assert.Equal(t, tt.expected, SubjectBotAnalyze(tt.tenantID))
		})
	}
}

func TestSubjectBotResponse(t *testing.T) {
	tests := []struct {
		tenantID string
		expected string
	}{
		{"tenant-1", "linktor.bot.response.tenant-1"},
		{"t1", "linktor.bot.response.t1"},
	}
	for _, tt := range tests {
		t.Run(tt.tenantID, func(t *testing.T) {
			assert.Equal(t, tt.expected, SubjectBotResponse(tt.tenantID))
		})
	}
}

func TestSubjectBotEscalate(t *testing.T) {
	tests := []struct {
		tenantID string
		expected string
	}{
		{"tenant-1", "linktor.bot.escalate.tenant-1"},
		{"t1", "linktor.bot.escalate.t1"},
	}
	for _, tt := range tests {
		t.Run(tt.tenantID, func(t *testing.T) {
			assert.Equal(t, tt.expected, SubjectBotEscalate(tt.tenantID))
		})
	}
}

func TestSubjectWhatsAppTemplateStatus(t *testing.T) {
	tests := []struct {
		tenantID string
		expected string
	}{
		{"t1", "linktor.whatsapp.template.status.t1"},
		{"tenant-abc", "linktor.whatsapp.template.status.tenant-abc"},
	}
	for _, tt := range tests {
		t.Run(tt.tenantID, func(t *testing.T) {
			assert.Equal(t, tt.expected, SubjectWhatsAppTemplateStatus(tt.tenantID))
		})
	}
}

func TestSubjectWhatsAppTemplateQuality(t *testing.T) {
	tests := []struct {
		tenantID string
		expected string
	}{
		{"t1", "linktor.whatsapp.template.quality.t1"},
		{"tenant-abc", "linktor.whatsapp.template.quality.tenant-abc"},
	}
	for _, tt := range tests {
		t.Run(tt.tenantID, func(t *testing.T) {
			assert.Equal(t, tt.expected, SubjectWhatsAppTemplateQuality(tt.tenantID))
		})
	}
}

func TestSubjectWhatsAppAccountAlert(t *testing.T) {
	tests := []struct {
		tenantID string
		expected string
	}{
		{"t1", "linktor.whatsapp.account.alert.t1"},
		{"tenant-abc", "linktor.whatsapp.account.alert.tenant-abc"},
	}
	for _, tt := range tests {
		t.Run(tt.tenantID, func(t *testing.T) {
			assert.Equal(t, tt.expected, SubjectWhatsAppAccountAlert(tt.tenantID))
		})
	}
}

func TestSubjectWhatsAppPhoneQuality(t *testing.T) {
	tests := []struct {
		tenantID string
		expected string
	}{
		{"t1", "linktor.whatsapp.phone.quality.t1"},
		{"tenant-abc", "linktor.whatsapp.phone.quality.tenant-abc"},
	}
	for _, tt := range tests {
		t.Run(tt.tenantID, func(t *testing.T) {
			assert.Equal(t, tt.expected, SubjectWhatsAppPhoneQuality(tt.tenantID))
		})
	}
}

func TestSubjectWhatsAppFlow(t *testing.T) {
	tests := []struct {
		tenantID string
		expected string
	}{
		{"t1", "linktor.whatsapp.flow.t1"},
		{"tenant-abc", "linktor.whatsapp.flow.tenant-abc"},
	}
	for _, tt := range tests {
		t.Run(tt.tenantID, func(t *testing.T) {
			assert.Equal(t, tt.expected, SubjectWhatsAppFlow(tt.tenantID))
		})
	}
}

func TestSubjectWhatsAppSecurity(t *testing.T) {
	tests := []struct {
		tenantID string
		expected string
	}{
		{"t1", "linktor.whatsapp.security.t1"},
		{"tenant-abc", "linktor.whatsapp.security.tenant-abc"},
	}
	for _, tt := range tests {
		t.Run(tt.tenantID, func(t *testing.T) {
			assert.Equal(t, tt.expected, SubjectWhatsAppSecurity(tt.tenantID))
		})
	}
}

func TestStreamNameConstants(t *testing.T) {
	streams := map[string]string{
		"StreamMessages": StreamMessages,
		"StreamEvents":   StreamEvents,
		"StreamWebhooks": StreamWebhooks,
		"StreamAI":       StreamAI,
		"StreamWhatsApp": StreamWhatsApp,
	}
	for name, value := range streams {
		t.Run(name, func(t *testing.T) {
			assert.NotEmpty(t, value, "%s should not be empty", name)
		})
	}
}

func TestEventTypeConstants(t *testing.T) {
	events := map[string]string{
		"EventMessageReceived":       EventMessageReceived,
		"EventMessageSent":           EventMessageSent,
		"EventMessageDelivered":      EventMessageDelivered,
		"EventMessageRead":           EventMessageRead,
		"EventMessageFailed":         EventMessageFailed,
		"EventMessageAnalyzed":       EventMessageAnalyzed,
		"EventConversationCreated":   EventConversationCreated,
		"EventConversationAssigned":  EventConversationAssigned,
		"EventConversationResolved":  EventConversationResolved,
		"EventConversationReopened":  EventConversationReopened,
		"EventConversationEscalated": EventConversationEscalated,
		"EventContactCreated":        EventContactCreated,
		"EventContactUpdated":        EventContactUpdated,
		"EventChannelConnected":      EventChannelConnected,
		"EventChannelDisconnected":   EventChannelDisconnected,
		"EventChannelError":          EventChannelError,
		"EventBotResponse":           EventBotResponse,
		"EventBotEscalation":         EventBotEscalation,
		"EventBotAnalysis":           EventBotAnalysis,
	}
	for name, value := range events {
		t.Run(name, func(t *testing.T) {
			assert.NotEmpty(t, value, "%s should not be empty", name)
		})
	}
}

func TestWhatsAppEventTypeConstants(t *testing.T) {
	events := map[string]string{
		"EventWhatsAppTemplateApproved":      EventWhatsAppTemplateApproved,
		"EventWhatsAppTemplateRejected":      EventWhatsAppTemplateRejected,
		"EventWhatsAppTemplatePaused":        EventWhatsAppTemplatePaused,
		"EventWhatsAppTemplateDisabled":      EventWhatsAppTemplateDisabled,
		"EventWhatsAppTemplateQualityGreen":  EventWhatsAppTemplateQualityGreen,
		"EventWhatsAppTemplateQualityYellow": EventWhatsAppTemplateQualityYellow,
		"EventWhatsAppTemplateQualityRed":    EventWhatsAppTemplateQualityRed,
		"EventWhatsAppAccountAlert":          EventWhatsAppAccountAlert,
		"EventWhatsAppAccountBanned":         EventWhatsAppAccountBanned,
		"EventWhatsAppAccountReview":         EventWhatsAppAccountReview,
		"EventWhatsAppPhoneFlagged":          EventWhatsAppPhoneFlagged,
		"EventWhatsAppPhoneUnflagged":        EventWhatsAppPhoneUnflagged,
		"EventWhatsAppPhoneNameApproved":     EventWhatsAppPhoneNameApproved,
		"EventWhatsAppPhoneNameRejected":     EventWhatsAppPhoneNameRejected,
		"EventWhatsAppFlowStatusChange":      EventWhatsAppFlowStatusChange,
		"EventWhatsAppFlowError":             EventWhatsAppFlowError,
		"EventWhatsAppSecurity":              EventWhatsAppSecurity,
		"EventWhatsAppCapabilityUpdate":      EventWhatsAppCapabilityUpdate,
		"EventWhatsAppEcho":                  EventWhatsAppEcho,
	}
	for name, value := range events {
		t.Run(name, func(t *testing.T) {
			assert.NotEmpty(t, value, "%s should not be empty", name)
		})
	}
}

func TestConsumerNameConstants(t *testing.T) {
	consumers := map[string]string{
		"ConsumerInbound":            ConsumerInbound,
		"ConsumerOutboundPrefix":     ConsumerOutboundPrefix,
		"ConsumerStatus":             ConsumerStatus,
		"ConsumerWebhooks":           ConsumerWebhooks,
		"ConsumerEvents":             ConsumerEvents,
		"ConsumerAIAnalyzer":         ConsumerAIAnalyzer,
		"ConsumerAIResponder":        ConsumerAIResponder,
		"ConsumerAIEscalation":       ConsumerAIEscalation,
		"ConsumerWhatsAppTemplates":  ConsumerWhatsAppTemplates,
		"ConsumerWhatsAppAccount":    ConsumerWhatsAppAccount,
		"ConsumerWhatsAppPhone":      ConsumerWhatsAppPhone,
		"ConsumerWhatsAppFlows":      ConsumerWhatsAppFlows,
		"ConsumerWhatsAppSecurity":   ConsumerWhatsAppSecurity,
		"ConsumerWhatsAppCapability": ConsumerWhatsAppCapability,
		"ConsumerWhatsAppEcho":       ConsumerWhatsAppEcho,
	}
	for name, value := range consumers {
		t.Run(name, func(t *testing.T) {
			assert.NotEmpty(t, value, "%s should not be empty", name)
		})
	}
}

func TestWildcardSubjectsEndWithWildcard(t *testing.T) {
	wildcardSubjects := map[string]string{
		"SubjectInboundAll":                    SubjectInboundAll,
		"SubjectOutboundAll":                   SubjectOutboundAll,
		"SubjectStatusAll":                     SubjectStatusAll,
		"SubjectEventsAll":                     SubjectEventsAll,
		"SubjectWebhooksAll":                   SubjectWebhooksAll,
		"SubjectBotAnalyzeAll":                 SubjectBotAnalyzeAll,
		"SubjectBotResponseAll":                SubjectBotResponseAll,
		"SubjectBotEscalateAll":                SubjectBotEscalateAll,
		"SubjectWhatsAppTemplateStatusAll":     SubjectWhatsAppTemplateStatusAll,
		"SubjectWhatsAppTemplateQualityAll":    SubjectWhatsAppTemplateQualityAll,
		"SubjectWhatsAppTemplateCategoryAll":   SubjectWhatsAppTemplateCategoryAll,
		"SubjectWhatsAppAccountAlertAll":       SubjectWhatsAppAccountAlertAll,
		"SubjectWhatsAppAccountUpdateAll":      SubjectWhatsAppAccountUpdateAll,
		"SubjectWhatsAppAccountReviewAll":      SubjectWhatsAppAccountReviewAll,
		"SubjectWhatsAppPhoneNameAll":          SubjectWhatsAppPhoneNameAll,
		"SubjectWhatsAppPhoneQualityAll":       SubjectWhatsAppPhoneQualityAll,
		"SubjectWhatsAppFlowAll":               SubjectWhatsAppFlowAll,
		"SubjectWhatsAppSecurityAll":           SubjectWhatsAppSecurityAll,
		"SubjectWhatsAppCapabilityAll":         SubjectWhatsAppCapabilityAll,
		"SubjectWhatsAppEchoAll":               SubjectWhatsAppEchoAll,
	}
	for name, value := range wildcardSubjects {
		t.Run(name, func(t *testing.T) {
			assert.Contains(t, value, ".>", "%s should end with '.>'", name)
		})
	}
}
