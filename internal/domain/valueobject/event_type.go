package valueobject

// Event type constants for the event distribution system
const (
	// Message events
	EventMessageInbound  = "message.inbound"
	EventMessageOutbound = "message.outbound"
	EventMessageStatus   = "message.status"
	EventMessageReaction = "message.reaction"
	EventMessageEdit     = "message.edit"
	EventMessageRevoke   = "message.revoke"

	// Conversation events
	EventConversationCreated   = "conversation.created"
	EventConversationUpdated   = "conversation.updated"
	EventConversationClosed    = "conversation.closed"
	EventConversationEscalated = "conversation.escalated"

	// Contact events
	EventContactCreated = "contact.created"
	EventContactUpdated = "contact.updated"

	// Channel/Connection events
	EventConnectionStatus = "connection.status"
	EventConnectionQRCode = "connection.qrcode"

	// Presence events
	EventPresenceTyping = "presence.typing"
	EventPresenceOnline = "presence.online"

	// System events
	EventSystemHealth = "system.health"

	// Wildcard — matches all event types
	EventAll = "*"
)

// allEventTypes is the registry of all known event types (excluding wildcard)
var allEventTypes = []string{
	EventMessageInbound, EventMessageOutbound, EventMessageStatus,
	EventMessageReaction, EventMessageEdit, EventMessageRevoke,
	EventConversationCreated, EventConversationUpdated,
	EventConversationClosed, EventConversationEscalated,
	EventContactCreated, EventContactUpdated,
	EventConnectionStatus, EventConnectionQRCode,
	EventPresenceTyping, EventPresenceOnline,
	EventSystemHealth,
}

// AllEventTypes returns all known event types (excluding wildcard)
func AllEventTypes() []string {
	out := make([]string, len(allEventTypes))
	copy(out, allEventTypes)
	return out
}

// IsValidEventType checks if the given type is a known event type
func IsValidEventType(eventType string) bool {
	if eventType == EventAll {
		return true
	}
	for _, t := range allEventTypes {
		if t == eventType {
			return true
		}
	}
	return false
}

// MatchesSubscription checks if eventType matches any of the subscribed events.
// An empty subscription list means subscribe to all events.
func MatchesSubscription(eventType string, subscribedEvents []string) bool {
	if len(subscribedEvents) == 0 {
		return true
	}
	for _, sub := range subscribedEvents {
		if sub == EventAll || sub == eventType {
			return true
		}
	}
	return false
}
