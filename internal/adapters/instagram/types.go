package instagram

import (
	"time"

	"github.com/msgfy/linktor/internal/adapters/meta"
)

// InstagramConfig holds Instagram DM channel configuration
type InstagramConfig struct {
	// Instagram account ID
	InstagramID string `json:"instagram_id"`

	// Access token for Instagram API
	AccessToken string `json:"access_token"`

	// Token expiration
	ExpiresAt time.Time `json:"expires_at,omitempty"`

	// App credentials (for webhook validation)
	AppID     string `json:"app_id,omitempty"`
	AppSecret string `json:"app_secret,omitempty"`

	// Webhook verification token
	VerifyToken string `json:"verify_token"`

	// Optional: Connected Facebook Page ID (for IG via FB Page flow)
	PageID          string `json:"page_id,omitempty"`
	PageAccessToken string `json:"page_access_token,omitempty"`
}

// Validate validates the configuration
func (c *InstagramConfig) Validate() error {
	if c.InstagramID == "" {
		return ErrMissingInstagramID
	}
	if c.AccessToken == "" && c.PageAccessToken == "" {
		return ErrMissingAccessToken
	}
	return nil
}

// GetEffectiveAccessToken returns the access token to use
func (c *InstagramConfig) GetEffectiveAccessToken() string {
	if c.AccessToken != "" {
		return c.AccessToken
	}
	return c.PageAccessToken
}

// IsExpired checks if the token is expired
func (c *InstagramConfig) IsExpired() bool {
	if c.ExpiresAt.IsZero() {
		return false
	}
	return time.Now().After(c.ExpiresAt)
}

// IncomingMessage represents a parsed incoming Instagram message
type IncomingMessage struct {
	ID          string
	ExternalID  string
	SenderID    string
	RecipientID string
	InstagramID string
	Text        string
	Attachments []Attachment
	IsEcho      bool
	IsDeleted   bool
	Timestamp   time.Time
}

// Attachment represents a message attachment
type Attachment struct {
	Type string
	URL  string
}

// OutgoingMessage represents a message to be sent
type OutgoingMessage struct {
	RecipientID string
	Text        string
	Attachments []OutgoingAttachment
}

// OutgoingAttachment represents an attachment to send
type OutgoingAttachment struct {
	Type string
	URL  string
}

// WebhookPayload is an alias for the meta package type
type WebhookPayload = meta.WebhookPayload

// MessagingEvent is an alias for the meta package type
type MessagingEvent = meta.MessagingEvent

// Errors
var (
	ErrMissingInstagramID  = &ConfigError{Field: "instagram_id", Message: "instagram_id is required"}
	ErrMissingAccessToken  = &ConfigError{Field: "access_token", Message: "access_token or page_access_token is required"}
	ErrTokenExpired        = &ConfigError{Field: "access_token", Message: "access token has expired"}
)

// ConfigError represents a configuration error
type ConfigError struct {
	Field   string
	Message string
}

func (e *ConfigError) Error() string {
	return e.Message
}
