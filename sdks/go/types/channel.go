package types

import (
	"encoding/json"
	"time"
)

// ConnectionStatus is the live connection state of a channel (wire field
// "connection_status"). It is distinct from Enabled, which is the system-level
// enable flag.
type ConnectionStatus string

const (
	ConnectionStatusDisconnected ConnectionStatus = "disconnected"
	ConnectionStatusConnecting   ConnectionStatus = "connecting"
	ConnectionStatusConnected    ConnectionStatus = "connected"
	ConnectionStatusError        ConnectionStatus = "error"
)

// CoexistenceStatus is the WhatsApp Business App + Cloud API coexistence state.
type CoexistenceStatus string

const (
	CoexistenceStatusInactive     CoexistenceStatus = "inactive"
	CoexistenceStatusPending      CoexistenceStatus = "pending"
	CoexistenceStatusActive       CoexistenceStatus = "active"
	CoexistenceStatusWarning      CoexistenceStatus = "warning"
	CoexistenceStatusDisconnected CoexistenceStatus = "disconnected"
)

// Channel mirrors the backend wire shape exactly (snake_case). Credentials are
// write-only and never present on a response.
type Channel struct {
	ID                       string            `json:"id"`
	TenantID                 string            `json:"tenant_id"`
	Type                     ChannelType       `json:"type"`
	Name                     string            `json:"name"`
	Identifier               string            `json:"identifier,omitempty"`
	Enabled                  bool              `json:"enabled"`
	ConnectionStatus         ConnectionStatus  `json:"connection_status"`
	Config                   map[string]string `json:"config,omitempty"`
	WebhookURL               string            `json:"webhook_url,omitempty"`
	CreatedAt                time.Time         `json:"created_at"`
	UpdatedAt                time.Time         `json:"updated_at"`
	IsCoexistence            bool              `json:"is_coexistence,omitempty"`
	WABAID                   string            `json:"waba_id,omitempty"`
	LastEchoAt               *time.Time        `json:"last_echo_at,omitempty"`
	CoexistenceStatus        CoexistenceStatus `json:"coexistence_status,omitempty"`
	MessageTemplateNamespace string            `json:"message_template_namespace,omitempty"`
}

// CreateChannelInput for creating channels.
//
// Config holds non-secret settings (e.g. phone_number_id, waba_id). Credentials
// holds secrets (e.g. access_token, bot_token) — the server stores them
// encrypted and never returns them. WebhookURL is the external endpoint Linktor
// delivers signed inbound/status events to for this channel.
type CreateChannelInput struct {
	Name        string            `json:"name"`
	Type        ChannelType       `json:"type"`
	Identifier  string            `json:"identifier,omitempty"`
	Config      map[string]string `json:"config,omitempty"`
	Credentials map[string]string `json:"credentials,omitempty"`
	WebhookURL  string            `json:"webhook_url,omitempty"`
}

// UpdateChannelInput for updating channels. Credentials, when present, replace
// the stored secrets; omit it (or send the redacted placeholder) to leave them
// untouched. Update reuses the create body shape server-side.
type UpdateChannelInput struct {
	Name        string            `json:"name,omitempty"`
	Identifier  string            `json:"identifier,omitempty"`
	Config      map[string]string `json:"config,omitempty"`
	Credentials map[string]string `json:"credentials,omitempty"`
	WebhookURL  string            `json:"webhook_url,omitempty"`
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

// ListChannelsParams for list request. Status/Type/Search are query filters.
type ListChannelsParams struct {
	PaginationParams
	Type   ChannelType      `json:"type,omitempty"`
	Status ConnectionStatus `json:"status,omitempty"`
	Search string           `json:"search,omitempty"`
}
