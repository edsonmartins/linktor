package valueobject

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAllEventTypes_NotEmpty(t *testing.T) {
	types := AllEventTypes()
	assert.NotEmpty(t, types)
	assert.GreaterOrEqual(t, len(types), 17)
}

func TestAllEventTypes_ReturnsCopy(t *testing.T) {
	a := AllEventTypes()
	b := AllEventTypes()
	a[0] = "modified"
	assert.NotEqual(t, a[0], b[0], "should return a copy, not the original")
}

func TestIsValidEventType_ValidTypes(t *testing.T) {
	validTypes := []string{
		EventMessageInbound, EventMessageOutbound, EventMessageStatus,
		EventMessageReaction, EventMessageEdit, EventMessageRevoke,
		EventConversationCreated, EventConversationClosed,
		EventContactCreated, EventContactUpdated,
		EventConnectionStatus, EventConnectionQRCode,
		EventPresenceTyping, EventPresenceOnline,
		EventSystemHealth, EventAll,
	}
	for _, et := range validTypes {
		assert.True(t, IsValidEventType(et), "expected %q to be valid", et)
	}
}

func TestIsValidEventType_InvalidType(t *testing.T) {
	assert.False(t, IsValidEventType("unknown.event"))
	assert.False(t, IsValidEventType(""))
	assert.False(t, IsValidEventType("message"))
}

func TestMatchesSubscription_EmptyList(t *testing.T) {
	assert.True(t, MatchesSubscription("message.inbound", nil))
	assert.True(t, MatchesSubscription("anything", []string{}))
}

func TestMatchesSubscription_MatchingEvent(t *testing.T) {
	subs := []string{"message.inbound", "message.outbound"}
	assert.True(t, MatchesSubscription("message.inbound", subs))
	assert.True(t, MatchesSubscription("message.outbound", subs))
}

func TestMatchesSubscription_NonMatchingEvent(t *testing.T) {
	subs := []string{"message.inbound"}
	assert.False(t, MatchesSubscription("conversation.created", subs))
}

func TestMatchesSubscription_WildcardInList(t *testing.T) {
	subs := []string{EventAll}
	assert.True(t, MatchesSubscription("message.inbound", subs))
	assert.True(t, MatchesSubscription("anything.else", subs))
}

func TestEventTypeConstants_Values(t *testing.T) {
	assert.Equal(t, "message.inbound", EventMessageInbound)
	assert.Equal(t, "message.outbound", EventMessageOutbound)
	assert.Equal(t, "conversation.created", EventConversationCreated)
	assert.Equal(t, "connection.qrcode", EventConnectionQRCode)
	assert.Equal(t, "*", EventAll)
}
