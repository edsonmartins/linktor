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

// ChannelStatus represents the status of a channel
type ChannelStatus string

const (
	ChannelStatusInactive     ChannelStatus = "inactive"
	ChannelStatusActive       ChannelStatus = "active"
	ChannelStatusError        ChannelStatus = "error"
	ChannelStatusDisconnected ChannelStatus = "disconnected"
)

// Channel represents a communication channel
type Channel struct {
	ID          string            `json:"id"`
	TenantID    string            `json:"tenant_id"`
	Type        ChannelType       `json:"type"`
	Name        string            `json:"name"`
	Identifier  string            `json:"identifier,omitempty"`
	Status      ChannelStatus     `json:"status"`
	Config      map[string]string `json:"config,omitempty"`
	Credentials map[string]string `json:"-"` // Never expose credentials
	WebhookURL  string            `json:"webhook_url,omitempty"`
	CreatedAt   time.Time         `json:"created_at"`
	UpdatedAt   time.Time         `json:"updated_at"`
}

// NewChannel creates a new channel
func NewChannel(tenantID string, channelType ChannelType, name, identifier string) *Channel {
	now := time.Now()
	return &Channel{
		TenantID:    tenantID,
		Type:        channelType,
		Name:        name,
		Identifier:  identifier,
		Status:      ChannelStatusInactive,
		Config:      make(map[string]string),
		Credentials: make(map[string]string),
		CreatedAt:   now,
		UpdatedAt:   now,
	}
}

// IsActive returns true if the channel is active
func (c *Channel) IsActive() bool {
	return c.Status == ChannelStatusActive
}
