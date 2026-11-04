package entity

import (
	"regexp"
	"strings"
	"time"
)

// ConversationOwnerRule declares which consumer owns the open conversation with
// a contact on a channel (P5 of the CRM ↔ Sales Copilot agreement). While the
// rule is in force, contact-bound events for that (channel, contact) go only to
// the owning consumer; without a rule they go to the channel's default consumer,
// which is today's behavior. The rule is written and released by the products,
// independently of the conversation status inside Linktor.
type ConversationOwnerRule struct {
	TenantID          string     `json:"tenant_id"`
	ChannelID         string     `json:"channel_id"`
	ContactIdentifier string     `json:"contact_identifier"`
	Consumer          string     `json:"consumidor"`
	ConversaRef       string     `json:"conversa_ref"`
	Desde             time.Time  `json:"desde"`
	ExpiraEm          *time.Time `json:"expira_em,omitempty"`
	UpdatedAt         time.Time  `json:"updated_at"`
}

// VigenteEm reports whether the rule is in force at t: a rule without expira_em
// lasts until it is released; an expired rule counts as no rule.
func (r *ConversationOwnerRule) VigenteEm(t time.Time) bool {
	return r != nil && (r.ExpiraEm == nil || t.Before(*r.ExpiraEm))
}

// consumerName is the shape of a consumer name in channel configuration keys
// ("consumers.<name>.webhook_url").
var consumerName = regexp.MustCompile(`^[a-z][a-z0-9_-]{0,63}$`)

// ValidConsumerName reports whether name can identify a consumer.
func ValidConsumerName(name string) bool {
	return consumerName.MatchString(name)
}

// NormalizeContactIdentifier canonicalizes the contact identifier used as the
// rule key: E.164 for anything that parses as a phone number (what the CRM
// knows), the trimmed raw identifier otherwise — the mechanism covers every
// channel type, not only phone-based ones.
func NormalizeContactIdentifier(raw string) string {
	if e164, ok := NormalizeE164(raw); ok {
		return e164
	}
	return strings.TrimSpace(raw)
}
