package types

import "encoding/json"

// ChannelStatus type
type ChannelStatus string

const (
	ChannelStatusActive     ChannelStatus = "active"
	ChannelStatusInactive   ChannelStatus = "inactive"
	ChannelStatusConnecting ChannelStatus = "connecting"
	ChannelStatusError      ChannelStatus = "error"
)

// Channel model
type Channel struct {
	ID         string                 `json:"id"`
	TenantID   string                 `json:"tenantId"`
	Name       string                 `json:"name"`
	Type       ChannelType            `json:"type"`
	Status     ChannelStatus          `json:"status"`
	Config     map[string]interface{} `json:"config"`
	WebhookURL string                 `json:"webhookUrl,omitempty"`
	Metadata   map[string]interface{} `json:"metadata,omitempty"`
	Timestamps
}

// CreateChannelInput for creating channels.
//
// Config holds non-secret settings (e.g. phone_number_id, waba_id). Credentials
// holds secrets (e.g. access_token, bot_token) — the server stores them
// encrypted and never returns them. WebhookURL is the external endpoint Linktor
// delivers signed inbound/status events to for this channel.
type CreateChannelInput struct {
	Name        string                 `json:"name"`
	Type        ChannelType            `json:"type"`
	Config      map[string]interface{} `json:"config,omitempty"`
	Credentials map[string]string      `json:"credentials,omitempty"`
	WebhookURL  string                 `json:"webhook_url,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// UpdateChannelInput for updating channels. Credentials, when present, replace
// the stored secrets; omit it to leave them untouched.
type UpdateChannelInput struct {
	Name        string                 `json:"name,omitempty"`
	Config      map[string]interface{} `json:"config,omitempty"`
	Credentials map[string]string      `json:"credentials,omitempty"`
	WebhookURL  string                 `json:"webhook_url,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// ConnectResult is the result of connecting a channel. For WhatsApp Web-style
// linking it carries the QR payload (QRCode) to render and its lifetime
// (ExpiresIn seconds); poll Connect again to refresh an expired code. PairCode
// is the phone-linking code when pairing by number. PasskeyRequired is true when
// the account is passkey-locked and must be linked by signing PasskeyChallenge
// (submit the assertion via SubmitPasskeyResponse), not by QR.
type ConnectResult struct {
	Channel          *Channel        `json:"channel"`
	QRCode           string          `json:"qr_code,omitempty"`
	ExpiresIn        int             `json:"expires_in,omitempty"`
	PairCode         string          `json:"pair_code,omitempty"`
	PasskeyRequired  bool            `json:"passkey_required,omitempty"`
	PasskeyChallenge json.RawMessage `json:"passkey_challenge,omitempty"`
}

// PairCodeInput requests a WhatsApp pairing code for a phone number.
type PairCodeInput struct {
	PhoneNumber string `json:"phone_number"`
}

// ListChannelsParams for list request
type ListChannelsParams struct {
	PaginationParams
	Type   ChannelType   `json:"type,omitempty"`
	Status ChannelStatus `json:"status,omitempty"`
	Search string        `json:"search,omitempty"`
}
