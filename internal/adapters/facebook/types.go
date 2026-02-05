package facebook

import (
	"time"

	"github.com/msgfy/linktor/internal/adapters/meta"
)

// FacebookConfig holds Facebook Messenger channel configuration
type FacebookConfig struct {
	// Facebook App credentials
	AppID     string `json:"app_id"`
	AppSecret string `json:"app_secret"`

	// Page credentials
	PageID          string `json:"page_id"`
	PageAccessToken string `json:"page_access_token"`

	// User access token (for OAuth flow)
	UserAccessToken string `json:"user_access_token,omitempty"`

	// Webhook verification
	VerifyToken string `json:"verify_token"`

	// Optional: Connected Instagram account ID
	InstagramID string `json:"instagram_id,omitempty"`
}

// Validate validates the configuration
func (c *FacebookConfig) Validate() error {
	if c.PageID == "" {
		return ErrMissingPageID
	}
	if c.PageAccessToken == "" {
		return ErrMissingPageAccessToken
	}
	return nil
}

// IncomingMessage represents a parsed incoming Facebook message
type IncomingMessage struct {
	ID          string
	ExternalID  string
	SenderID    string
	RecipientID string
	PageID      string
	Text        string
	Attachments []Attachment
	IsEcho      bool
	QuickReply  string
	Timestamp   time.Time
}

// Attachment represents a message attachment
type Attachment struct {
	Type     string
	URL      string
	Title    string
	Lat      float64
	Long     float64
}

// OutgoingMessage represents a message to be sent
type OutgoingMessage struct {
	RecipientID string
	Text        string
	Attachments []OutgoingAttachment
	QuickReplies []QuickReply
}

// OutgoingAttachment represents an attachment to send
type OutgoingAttachment struct {
	Type string
	URL  string
}

// QuickReply represents a quick reply button
type QuickReply struct {
	Title   string
	Payload string
}

// DeliveryStatus represents message delivery status
type DeliveryStatus struct {
	MessageIDs []string
	Watermark  time.Time
}

// ReadStatus represents message read status
type ReadStatus struct {
	Watermark time.Time
}

// Postback represents a button postback
type Postback struct {
	Title   string
	Payload string
}

// WebhookPayload is an alias for the meta package type
type WebhookPayload = meta.WebhookPayload

// MessagingEvent is an alias for the meta package type
type MessagingEvent = meta.MessagingEvent

// Errors
var (
	ErrMissingPageID          = &ConfigError{Field: "page_id", Message: "page_id is required"}
	ErrMissingPageAccessToken = &ConfigError{Field: "page_access_token", Message: "page_access_token is required"}
	ErrMissingAppSecret       = &ConfigError{Field: "app_secret", Message: "app_secret is required for webhook validation"}
)

// ConfigError represents a configuration error
type ConfigError struct {
	Field   string
	Message string
}

func (e *ConfigError) Error() string {
	return e.Message
}
