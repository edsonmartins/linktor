package entity

import (
	"encoding/json"
	"strconv"
	"time"
)

// RedactedSecret is the placeholder emitted in a serialized Channel.Config for
// sensitive keys. It is intentionally not a plausible real value. Clients must
// not echo it back on update: ChannelService.Update treats an incoming
// RedactedSecret (or empty) sensitive value as "keep the stored secret".
const RedactedSecret = "__redacted__"

// SensitiveConfigKeys lists Channel.Config keys whose values are secrets. They
// are encrypted at rest (see database.encryptChannelSecrets) and redacted from
// API responses (see Channel.MarshalJSON). Non-listed keys (phone_number_id,
// waba_id, proxy host/port, widget display config, ...) stay plaintext so they
// remain queryable and readable. Credentials is encrypted/hidden in full.
var SensitiveConfigKeys = map[string]bool{
	"access_token":          true,
	"user_access_token":     true,
	"page_access_token":     true,
	"auth_token":            true,
	"bot_token":             true,
	"app_secret":            true,
	"api_secret":            true,
	"api_key":               true,
	"api_key_secret":        true,
	"api_key_sid":           true,
	"account_sid":           true,
	"messaging_service_sid": true,
	"webhook_secret":        true,
	"verify_token":          true,
	"widget_secret":         true,
	"smtp_password":         true,
	"imap_password":         true,
	"mailgun_api_key":       true,
	"sendgrid_api_key":      true,
	"postmark_server_token": true,
	"ses_access_key_id":     true,
	"ses_secret_key":        true,
	"private_key":           true,
	"secret":                true,
	"token":                 true,
	"password":              true,
}

// ChannelType represents the type of a channel
type ChannelType string

const (
	ChannelTypeWebChat            ChannelType = "webchat"
	ChannelTypeWhatsApp           ChannelType = "whatsapp"
	ChannelTypeWhatsAppOfficial   ChannelType = "whatsapp_official"
	ChannelTypeWhatsAppUnofficial ChannelType = "whatsapp_unofficial"
	ChannelTypeTelegram           ChannelType = "telegram"
	ChannelTypeSMS                ChannelType = "sms"
	ChannelTypeRCS                ChannelType = "rcs"
	ChannelTypeInstagram          ChannelType = "instagram"
	ChannelTypeFacebook           ChannelType = "facebook"
	ChannelTypeEmail              ChannelType = "email"
	ChannelTypeVoice              ChannelType = "voice"
	ChannelTypeTeams              ChannelType = "teams"
	ChannelTypeSlack              ChannelType = "slack"
	ChannelTypeMattermost         ChannelType = "mattermost"
	ChannelTypeDireto             ChannelType = "direto"
)

// ChannelEnvironment tells whether a channel instance carries real customer
// traffic or synthetic homologation traffic. It is a policy attribute consulted
// by enforcement points (sandbox delivery guard, retention, export blocking);
// it never participates in send routing, which stays keyed by channel ID.
// The value is immutable after creation: every guarantee derived from it
// (allowlist, marking, purge) assumes a channel cannot switch environments.
type ChannelEnvironment string

const (
	ChannelEnvironmentProduction ChannelEnvironment = "production"
	ChannelEnvironmentSandbox    ChannelEnvironment = "sandbox"
)

// ParseChannelEnvironment validates and normalizes an environment value.
// Empty input means the caller did not choose, which defaults to production so
// existing channels and callers keep their behavior.
func ParseChannelEnvironment(v string) (ChannelEnvironment, bool) {
	switch ChannelEnvironment(v) {
	case "":
		return ChannelEnvironmentProduction, true
	case ChannelEnvironmentProduction, ChannelEnvironmentSandbox:
		return ChannelEnvironment(v), true
	default:
		return "", false
	}
}

// ConnectionStatus represents the connection status of a channel
type ConnectionStatus string

const (
	ConnectionStatusDisconnected ConnectionStatus = "disconnected"
	ConnectionStatusConnecting   ConnectionStatus = "connecting"
	ConnectionStatusConnected    ConnectionStatus = "connected"
	ConnectionStatusError        ConnectionStatus = "error"
)

// CoexistenceStatus represents the coexistence status for WhatsApp channels
type CoexistenceStatus string

const (
	CoexistenceStatusInactive     CoexistenceStatus = "inactive"
	CoexistenceStatusPending      CoexistenceStatus = "pending"
	CoexistenceStatusActive       CoexistenceStatus = "active"
	CoexistenceStatusWarning      CoexistenceStatus = "warning"
	CoexistenceStatusDisconnected CoexistenceStatus = "disconnected"
)

// ChannelStatus is deprecated, use ConnectionStatus instead
// Kept for backwards compatibility during migration
type ChannelStatus = ConnectionStatus

// Backwards compatibility constants
const (
	ChannelStatusInactive     = ConnectionStatusDisconnected
	ChannelStatusActive       = ConnectionStatusConnected
	ChannelStatusConnecting   = ConnectionStatusConnecting
	ChannelStatusError        = ConnectionStatusError
	ChannelStatusDisconnected = ConnectionStatusDisconnected
)

// Channel represents a communication channel
type Channel struct {
	ID               string             `json:"id"`
	TenantID         string             `json:"tenant_id"`
	Type             ChannelType        `json:"type"`
	Name             string             `json:"name"`
	Identifier       string             `json:"identifier,omitempty"`
	Enabled          bool               `json:"enabled"`           // Whether channel is enabled in system
	ConnectionStatus ConnectionStatus   `json:"connection_status"` // Connection state (connected/disconnected/etc)
	Environment      ChannelEnvironment `json:"environment"`       // Immutable after creation; see ChannelEnvironment
	Config           map[string]string  `json:"config,omitempty"`
	Credentials      map[string]string  `json:"-"` // Never expose credentials
	WebhookURL       string             `json:"webhook_url,omitempty"`
	CreatedAt        time.Time          `json:"created_at"`
	UpdatedAt        time.Time          `json:"updated_at"`

	// WhatsApp Coexistence fields
	IsCoexistence     bool              `json:"is_coexistence,omitempty"`     // Whether channel uses Business App + Cloud API
	WABAID            string            `json:"waba_id,omitempty"`            // WhatsApp Business Account ID
	LastEchoAt        *time.Time        `json:"last_echo_at,omitempty"`       // Last message echo from Business App
	CoexistenceStatus CoexistenceStatus `json:"coexistence_status,omitempty"` // Current coexistence status

	// MessageTemplateNamespace is the WABA-level identifier Meta exposes on
	// GET /{waba-id}. The Cloud API template sending flow does not require
	// it (templates are addressed by name + language), but partners running
	// legacy HSM integrations or customers provisioning their own Cloud API
	// apps do. We fetch it lazily via TemplateService.FetchNamespace.
	MessageTemplateNamespace string `json:"message_template_namespace,omitempty"`
}

// MarshalJSON redacts sensitive Config values (access tokens, app secrets,
// widget secrets, ...) from any serialized Channel so they never leak through
// API responses or events. It operates on a copy: the in-memory struct keeps
// the real values, so Go-level access (channel.Config["access_token"]) and DB
// persistence (which marshals the Config map field, not the whole entity) are
// unaffected. Credentials is already hidden via `json:"-"`.
func (c *Channel) MarshalJSON() ([]byte, error) {
	type channelAlias Channel
	clone := *c
	if len(c.Config) > 0 {
		redacted := make(map[string]string, len(c.Config))
		for k, v := range c.Config {
			if v != "" && SensitiveConfigKeys[k] {
				redacted[k] = RedactedSecret
			} else {
				redacted[k] = v
			}
		}
		clone.Config = redacted
	}
	return json.Marshal((*channelAlias)(&clone))
}

// NewChannel creates a new channel
func NewChannel(tenantID string, channelType ChannelType, name, identifier string) *Channel {
	now := time.Now()
	return &Channel{
		TenantID:         tenantID,
		Type:             channelType,
		Name:             name,
		Identifier:       identifier,
		Enabled:          true,                         // Enabled by default
		ConnectionStatus: ConnectionStatusDisconnected, // Start disconnected
		Environment:      ChannelEnvironmentProduction,
		Config:           make(map[string]string),
		Credentials:      make(map[string]string),
		CreatedAt:        now,
		UpdatedAt:        now,
	}
}

// IsEnabled returns true if the channel is enabled
func (c *Channel) IsEnabled() bool {
	return c.Enabled
}

// IsSandbox returns true if the channel carries synthetic homologation traffic.
// A missing environment (legacy row or zero-value struct) is production.
func (c *Channel) IsSandbox() bool {
	return c.Environment == ChannelEnvironmentSandbox
}

// IsConnected returns true if the channel is connected
func (c *Channel) IsConnected() bool {
	return c.ConnectionStatus == ConnectionStatusConnected
}

// IsActive returns true if the channel is both enabled and connected
// Deprecated: Use IsEnabled() and IsConnected() separately
func (c *Channel) IsActive() bool {
	return c.Enabled && c.ConnectionStatus == ConnectionStatusConnected
}

// IsCoexistenceChannel returns true if the channel supports WhatsApp Coexistence
func (c *Channel) IsCoexistenceChannel() bool {
	return c.IsCoexistence && c.Type == ChannelTypeWhatsAppOfficial
}

// IsCoexistenceActive returns true if coexistence is active and working
func (c *Channel) IsCoexistenceActive() bool {
	return c.IsCoexistence && c.CoexistenceStatus == CoexistenceStatusActive
}

// DaysSinceLastEcho returns the number of days since the last message echo
// Returns -1 if no echo has been received
func (c *Channel) DaysSinceLastEcho() int {
	if c.LastEchoAt == nil {
		return -1
	}
	return int(time.Since(*c.LastEchoAt).Hours() / 24)
}

// UpdateLastEchoAt updates the last echo timestamp to current time
func (c *Channel) UpdateLastEchoAt() {
	now := time.Now()
	c.LastEchoAt = &now
	c.UpdatedAt = now

	// Update coexistence status based on echo
	if c.IsCoexistence {
		c.CoexistenceStatus = CoexistenceStatusActive
	}
}

// SetLastEchoAt sets the last echo timestamp to a specific time
func (c *Channel) SetLastEchoAt(t time.Time) {
	c.LastEchoAt = &t
	c.UpdatedAt = time.Now()

	// Update coexistence status based on echo
	if c.IsCoexistence {
		c.CoexistenceStatus = CoexistenceStatusActive
	}
}

// CheckCoexistenceStatus evaluates and returns the current coexistence status
// based on the last echo timestamp
func (c *Channel) CheckCoexistenceStatus() CoexistenceStatus {
	if !c.IsCoexistence {
		return CoexistenceStatusInactive
	}

	days := c.DaysSinceLastEcho()
	if days < 0 {
		return CoexistenceStatusPending
	}

	switch {
	case days >= 14:
		return CoexistenceStatusDisconnected
	case days >= 10:
		return CoexistenceStatusWarning
	default:
		return CoexistenceStatusActive
	}
}

// AdvancedSettings represents configurable per-channel behavior settings
type AdvancedSettings struct {
	AlwaysOnline     bool   `json:"always_online"`      // Show online status always
	RejectCall       bool   `json:"reject_call"`        // Auto-reject incoming calls
	RejectCallMsg    string `json:"reject_call_msg"`    // Message sent when call is rejected
	AutoReadMessages bool   `json:"auto_read_messages"` // Auto-mark incoming messages as read
	IgnoreGroups     bool   `json:"ignore_groups"`      // Ignore group messages
	IgnoreStatus     bool   `json:"ignore_status"`      // Ignore status/story broadcasts
	QRCodeMaxCount   int    `json:"qrcode_max_count"`   // Max QR code generation attempts (0=unlimited)
	ProxyHost        string `json:"proxy_host"`         // SOCKS5 proxy host
	ProxyPort        int    `json:"proxy_port"`         // Proxy port
	ProxyUser        string `json:"proxy_user"`         // Proxy username
	ProxyPass        string `json:"proxy_pass"`         // Proxy password
}

// DefaultAdvancedSettings returns settings with sensible defaults
func DefaultAdvancedSettings() *AdvancedSettings {
	return &AdvancedSettings{
		QRCodeMaxCount: 5,
	}
}

// GetAdvancedSettings parses AdvancedSettings from Channel.Config
func (c *Channel) GetAdvancedSettings() *AdvancedSettings {
	if c.Config == nil {
		return DefaultAdvancedSettings()
	}

	s := DefaultAdvancedSettings()

	if v, ok := c.Config["always_online"]; ok {
		s.AlwaysOnline, _ = strconv.ParseBool(v)
	}
	if v, ok := c.Config["reject_call"]; ok {
		s.RejectCall, _ = strconv.ParseBool(v)
	}
	if v, ok := c.Config["reject_call_msg"]; ok {
		s.RejectCallMsg = v
	}
	if v, ok := c.Config["auto_read_messages"]; ok {
		s.AutoReadMessages, _ = strconv.ParseBool(v)
	}
	if v, ok := c.Config["ignore_groups"]; ok {
		s.IgnoreGroups, _ = strconv.ParseBool(v)
	}
	if v, ok := c.Config["ignore_status"]; ok {
		s.IgnoreStatus, _ = strconv.ParseBool(v)
	}
	if v, ok := c.Config["qrcode_max_count"]; ok {
		if n, err := strconv.Atoi(v); err == nil {
			s.QRCodeMaxCount = n
		}
	}
	if v, ok := c.Config["proxy_host"]; ok {
		s.ProxyHost = v
	}
	if v, ok := c.Config["proxy_port"]; ok {
		if n, err := strconv.Atoi(v); err == nil {
			s.ProxyPort = n
		}
	}
	if v, ok := c.Config["proxy_user"]; ok {
		s.ProxyUser = v
	}
	if v, ok := c.Config["proxy_pass"]; ok {
		s.ProxyPass = v
	}

	return s
}

// SetAdvancedSettings serializes AdvancedSettings into Channel.Config
func (c *Channel) SetAdvancedSettings(settings *AdvancedSettings) {
	if c.Config == nil {
		c.Config = make(map[string]string)
	}

	c.Config["always_online"] = strconv.FormatBool(settings.AlwaysOnline)
	c.Config["reject_call"] = strconv.FormatBool(settings.RejectCall)
	c.Config["reject_call_msg"] = settings.RejectCallMsg
	c.Config["auto_read_messages"] = strconv.FormatBool(settings.AutoReadMessages)
	c.Config["ignore_groups"] = strconv.FormatBool(settings.IgnoreGroups)
	c.Config["ignore_status"] = strconv.FormatBool(settings.IgnoreStatus)
	c.Config["qrcode_max_count"] = strconv.Itoa(settings.QRCodeMaxCount)
	c.Config["proxy_host"] = settings.ProxyHost
	c.Config["proxy_port"] = strconv.Itoa(settings.ProxyPort)
	c.Config["proxy_user"] = settings.ProxyUser
	c.Config["proxy_pass"] = settings.ProxyPass
}

// HasProxy returns true if a proxy is configured
func (s *AdvancedSettings) HasProxy() bool {
	return s.ProxyHost != ""
}
