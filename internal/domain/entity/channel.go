package entity

import (
	"time"
)

// ChannelType represents the type of a channel
type ChannelType string

const (
	ChannelTypeWebChat             ChannelType = "webchat"
	ChannelTypeWhatsApp            ChannelType = "whatsapp"
	ChannelTypeWhatsAppOfficial    ChannelType = "whatsapp_official"
	ChannelTypeWhatsAppUnofficial  ChannelType = "whatsapp_unofficial"
	ChannelTypeTelegram            ChannelType = "telegram"
	ChannelTypeSMS                 ChannelType = "sms"
	ChannelTypeRCS                 ChannelType = "rcs"
	ChannelTypeInstagram           ChannelType = "instagram"
	ChannelTypeFacebook            ChannelType = "facebook"
	ChannelTypeEmail               ChannelType = "email"
	ChannelTypeVoice               ChannelType = "voice"
)

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
	ID               string            `json:"id"`
	TenantID         string            `json:"tenant_id"`
	Type             ChannelType       `json:"type"`
	Name             string            `json:"name"`
	Identifier       string            `json:"identifier,omitempty"`
	Enabled          bool              `json:"enabled"`                      // Whether channel is enabled in system
	ConnectionStatus ConnectionStatus  `json:"connection_status"`            // Connection state (connected/disconnected/etc)
	Config           map[string]string `json:"config,omitempty"`
	Credentials      map[string]string `json:"-"`                            // Never expose credentials
	WebhookURL       string            `json:"webhook_url,omitempty"`
	CreatedAt        time.Time         `json:"created_at"`
	UpdatedAt        time.Time         `json:"updated_at"`

	// WhatsApp Coexistence fields
	IsCoexistence      bool              `json:"is_coexistence,omitempty"`       // Whether channel uses Business App + Cloud API
	WABAID             string            `json:"waba_id,omitempty"`              // WhatsApp Business Account ID
	LastEchoAt         *time.Time        `json:"last_echo_at,omitempty"`         // Last message echo from Business App
	CoexistenceStatus  CoexistenceStatus `json:"coexistence_status,omitempty"`   // Current coexistence status
}

// NewChannel creates a new channel
func NewChannel(tenantID string, channelType ChannelType, name, identifier string) *Channel {
	now := time.Now()
	return &Channel{
		TenantID:         tenantID,
		Type:             channelType,
		Name:             name,
		Identifier:       identifier,
		Enabled:          true,                          // Enabled by default
		ConnectionStatus: ConnectionStatusDisconnected,  // Start disconnected
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
