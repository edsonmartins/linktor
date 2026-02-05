package whatsapp

import (
	"errors"
	"time"

	"go.mau.fi/whatsmeow/types"
)

// Errors
var (
	ErrNotLoggedIn     = errors.New("device not logged in")
	ErrAlreadyLoggedIn = errors.New("device already logged in")
	ErrClientNotReady  = errors.New("client not ready")
	ErrQRTimeout       = errors.New("QR code timeout")
	ErrSessionExpired  = errors.New("session expired")
)

// DeviceState represents the current state of a WhatsApp device
type DeviceState string

const (
	DeviceStateDisconnected DeviceState = "disconnected"
	DeviceStateConnecting   DeviceState = "connecting"
	DeviceStateConnected    DeviceState = "connected"
	DeviceStateLoggedOut    DeviceState = "logged_out"
	DeviceStateQRPending    DeviceState = "qr_pending"
)

// Config holds the configuration for WhatsApp unofficial adapter
type Config struct {
	// ChannelID is the unique identifier for this channel
	ChannelID string

	// DatabasePath is the path to the SQLite database for storing session data
	DatabasePath string

	// AutoReconnect enables automatic reconnection
	AutoReconnect bool

	// AutoTrustIdentity automatically trusts new identity keys
	AutoTrustIdentity bool

	// HistorySync enables message history sync on connection
	HistorySync bool

	// LogLevel sets the logging level
	LogLevel string

	// DeviceName is the display name for this device
	DeviceName string

	// PlatformType identifies the platform (e.g., "chrome", "firefox", "safari")
	PlatformType string
}

// Validate validates the configuration
func (c *Config) Validate() error {
	if c.ChannelID == "" {
		return errors.New("channel_id is required")
	}
	return nil
}

// SetDefaults sets default values for the configuration
func (c *Config) SetDefaults() {
	if c.DatabasePath == "" {
		c.DatabasePath = "storages/whatsapp_" + c.ChannelID + ".db"
	}
	if c.DeviceName == "" {
		c.DeviceName = "Linktor"
	}
	if c.PlatformType == "" {
		c.PlatformType = "chrome"
	}
	c.AutoReconnect = true
	c.AutoTrustIdentity = true
}

// DeviceInfo contains information about the connected device
type DeviceInfo struct {
	ID          string      `json:"id"`
	JID         string      `json:"jid"`
	PhoneNumber string      `json:"phone_number"`
	DisplayName string      `json:"display_name"`
	State       DeviceState `json:"state"`
	Platform    string      `json:"platform"`
	BusinessID  string      `json:"business_id,omitempty"`
	CreatedAt   time.Time   `json:"created_at"`
	UpdatedAt   time.Time   `json:"updated_at"`
}

// QRCodeEvent represents a QR code event during login
type QRCodeEvent struct {
	Code      string    `json:"code"`
	ImagePath string    `json:"image_path,omitempty"`
	ImageData []byte    `json:"image_data,omitempty"`
	ExpiresAt time.Time `json:"expires_at"`
}

// PairCodeRequest represents a request for phone pairing code
type PairCodeRequest struct {
	PhoneNumber string `json:"phone_number"`
}

// PairCodeResponse represents the response for phone pairing
type PairCodeResponse struct {
	Code      string    `json:"code"`
	ExpiresAt time.Time `json:"expires_at"`
}

// IncomingMessage represents an incoming WhatsApp message
type IncomingMessage struct {
	ExternalID   string       `json:"external_id"`
	SenderJID    types.JID    `json:"sender_jid"`
	ChatJID      types.JID    `json:"chat_jid"`
	SenderName   string       `json:"sender_name"`
	Text         string       `json:"text"`
	Timestamp    time.Time    `json:"timestamp"`
	IsFromMe     bool         `json:"is_from_me"`
	IsGroup      bool         `json:"is_group"`
	IsForwarded  bool         `json:"is_forwarded"`
	QuotedID     string       `json:"quoted_id,omitempty"`
	Attachments  []Attachment `json:"attachments,omitempty"`
	Mentions     []string     `json:"mentions,omitempty"`
	ReplyTo      *ReplyInfo   `json:"reply_to,omitempty"`
	Reaction     *Reaction    `json:"reaction,omitempty"`
	MessageType  string       `json:"message_type"`
	RawMessage   any          `json:"raw_message,omitempty"`
}

// Attachment represents a media attachment
type Attachment struct {
	Type      string `json:"type"`
	URL       string `json:"url,omitempty"`
	MediaKey  []byte `json:"media_key,omitempty"`
	SHA256    []byte `json:"sha256,omitempty"`
	EncSHA256 []byte `json:"enc_sha256,omitempty"`
	MimeType  string `json:"mime_type"`
	FileSize  uint64 `json:"file_size"`
	Filename  string `json:"filename,omitempty"`
	Caption   string `json:"caption,omitempty"`
	Width     uint32 `json:"width,omitempty"`
	Height    uint32 `json:"height,omitempty"`
	Duration  uint32 `json:"duration,omitempty"`
	Thumbnail []byte `json:"thumbnail,omitempty"`
	LocalPath string `json:"local_path,omitempty"`
}

// ReplyInfo contains information about a replied message
type ReplyInfo struct {
	MessageID string    `json:"message_id"`
	SenderJID types.JID `json:"sender_jid"`
	Text      string    `json:"text,omitempty"`
}

// Reaction represents a message reaction
type Reaction struct {
	Emoji     string    `json:"emoji"`
	SenderJID types.JID `json:"sender_jid"`
	MessageID string    `json:"message_id"`
	Timestamp time.Time `json:"timestamp"`
}

// Receipt represents a message receipt (read/delivered)
type Receipt struct {
	MessageIDs []string    `json:"message_ids"`
	SenderJID  types.JID   `json:"sender_jid"`
	ChatJID    types.JID   `json:"chat_jid"`
	Type       ReceiptType `json:"type"`
	Timestamp  time.Time   `json:"timestamp"`
}

// ReceiptType represents the type of receipt
type ReceiptType string

const (
	ReceiptTypeDelivered ReceiptType = "delivered"
	ReceiptTypeRead      ReceiptType = "read"
	ReceiptTypePlayed    ReceiptType = "played"
)

// PresenceUpdate represents a presence update (online/offline)
type PresenceUpdate struct {
	JID         types.JID `json:"jid"`
	Available   bool      `json:"available"`
	LastSeenAt  time.Time `json:"last_seen_at,omitempty"`
	Unavailable bool      `json:"unavailable"`
}

// ChatPresence represents typing/recording status
type ChatPresence struct {
	JID   types.JID         `json:"jid"`
	Chat  types.JID         `json:"chat"`
	State ChatPresenceState `json:"state"`
}

// ChatPresenceState represents the presence state
type ChatPresenceState string

const (
	ChatPresenceComposing ChatPresenceState = "composing"
	ChatPresenceRecording ChatPresenceState = "recording"
	ChatPresencePaused    ChatPresenceState = "paused"
)

// SendMessageRequest represents a request to send a message
type SendMessageRequest struct {
	To          string       `json:"to"`
	Text        string       `json:"text,omitempty"`
	ReplyToID   string       `json:"reply_to_id,omitempty"`
	Mentions    []string     `json:"mentions,omitempty"`
	Attachments []Attachment `json:"attachments,omitempty"`
	IsForwarded bool         `json:"is_forwarded,omitempty"`
	Disappearing uint32      `json:"disappearing,omitempty"`
}

// SendMessageResponse represents the response from sending a message
type SendMessageResponse struct {
	MessageID string    `json:"message_id"`
	Timestamp time.Time `json:"timestamp"`
}

// GroupInfo represents information about a WhatsApp group
type GroupInfo struct {
	JID         types.JID        `json:"jid"`
	Name        string           `json:"name"`
	Topic       string           `json:"topic,omitempty"`
	Participants []GroupParticipant `json:"participants"`
	CreatedAt   time.Time        `json:"created_at"`
	CreatedBy   types.JID        `json:"created_by"`
}

// GroupParticipant represents a participant in a group
type GroupParticipant struct {
	JID     types.JID `json:"jid"`
	IsAdmin bool      `json:"is_admin"`
	IsSuperAdmin bool `json:"is_super_admin"`
}

// ContactInfo represents a contact
type ContactInfo struct {
	JID           types.JID `json:"jid"`
	FullName      string    `json:"full_name"`
	PushName      string    `json:"push_name"`
	BusinessName  string    `json:"business_name,omitempty"`
	ProfilePicURL string    `json:"profile_pic_url,omitempty"`
	Status        string    `json:"status,omitempty"`
	IsBlocked     bool      `json:"is_blocked"`
	IsBusiness    bool      `json:"is_business"`
}

// MediaType represents the type of media
type MediaType string

const (
	MediaTypeImage    MediaType = "image"
	MediaTypeVideo    MediaType = "video"
	MediaTypeAudio    MediaType = "audio"
	MediaTypeDocument MediaType = "document"
	MediaTypeSticker  MediaType = "sticker"
)

// AttachmentTypeFromMIME returns the attachment type based on MIME type
func AttachmentTypeFromMIME(mimeType string) string {
	switch {
	case mimeType == "image/webp":
		return "sticker"
	case len(mimeType) >= 6 && mimeType[:6] == "image/":
		return "image"
	case len(mimeType) >= 6 && mimeType[:6] == "video/":
		return "video"
	case len(mimeType) >= 6 && mimeType[:6] == "audio/":
		return "audio"
	default:
		return "document"
	}
}

// ConnectionEvent represents a connection state change
type ConnectionEvent struct {
	State   DeviceState `json:"state"`
	Reason  string      `json:"reason,omitempty"`
	Time    time.Time   `json:"time"`
}

// HistorySyncEvent represents a history sync notification
type HistorySyncEvent struct {
	Progress int       `json:"progress"`
	Complete bool      `json:"complete"`
	Time     time.Time `json:"time"`
}

// LogoutEvent represents a logout event
type LogoutEvent struct {
	Reason    string    `json:"reason"`
	Time      time.Time `json:"time"`
	FromPhone bool      `json:"from_phone"`
}
