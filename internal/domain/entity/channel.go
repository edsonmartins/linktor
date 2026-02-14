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
