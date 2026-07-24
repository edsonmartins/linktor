package entity

import (
	"regexp"
	"strings"
	"time"
)

// SandboxAllowlistEntry authorizes one recipient to receive messages from a
// tenant's sandbox channels (INV-017). Entries are tenant-scoped; ChannelID
// optionally narrows an entry to a single sandbox channel ("" = valid for all
// of the tenant's sandbox channels). Recipient is always stored normalized
// (see NormalizeE164) so lookups compare canonical forms, never raw input.
type SandboxAllowlistEntry struct {
	ID        string    `json:"id"`
	TenantID  string    `json:"tenant_id"`
	ChannelID string    `json:"channel_id,omitempty"`
	Recipient string    `json:"recipient"`
	Note      string    `json:"note,omitempty"`
	CreatedBy string    `json:"created_by,omitempty"`
	CreatedAt time.Time `json:"created_at"`
}

// e164Digits matches the digits of an E.164 number after normalization:
// no leading zero, 8 to 15 digits total.
var e164Digits = regexp.MustCompile(`^[1-9]\d{7,14}$`)

// e164Separators are the formatting characters tolerated in raw phone input.
var e164Separators = strings.NewReplacer(" ", "", "-", "", ".", "", "(", "", ")", "")

// NormalizeE164 canonicalizes a phone number to "+<digits>" E.164 form,
// tolerating common formatting (spaces, dashes, dots, parentheses, optional
// leading +). It must be applied both when writing allowlist entries and when
// comparing an outbound recipient against them: providers hand recipients in
// different shapes ("+55 44 9..." vs "55449...") and a normalization mismatch
// either blocks a legitimate test recipient or — worse — lets one through.
func NormalizeE164(raw string) (string, bool) {
	s := e164Separators.Replace(strings.TrimSpace(raw))
	s = strings.TrimPrefix(s, "+")
	if !e164Digits.MatchString(s) {
		return "", false
	}
	return "+" + s, true
}

// MaskRecipient redacts a recipient identifier for logs, metrics and error
// messages, keeping only a short prefix and the last two characters. Full
// recipient numbers must never appear in any observable output (INV-002).
func MaskRecipient(raw string) string {
	s := strings.TrimSpace(raw)
	if len(s) <= 5 {
		return "****"
	}
	return s[:3] + strings.Repeat("*", len(s)-5) + s[len(s)-2:]
}
