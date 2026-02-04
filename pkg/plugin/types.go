package plugin

import (
	"time"
)

// ChannelType represents the type of messaging channel
type ChannelType string

const (
	ChannelTypeWebChat          ChannelType = "webchat"
	ChannelTypeWhatsApp         ChannelType = "whatsapp"
	ChannelTypeWhatsAppOfficial ChannelType = "whatsapp_official"
	ChannelTypeTelegram         ChannelType = "telegram"
	ChannelTypeSMS              ChannelType = "sms"
	ChannelTypeRCS              ChannelType = "rcs"
	ChannelTypeInstagram        ChannelType = "instagram"
	ChannelTypeFacebook         ChannelType = "facebook"
	ChannelTypeEmail            ChannelType = "email"
	ChannelTypeVoice            ChannelType = "voice"
)

// MessageStatus represents the delivery status of a message
type MessageStatus string

const (
	MessageStatusPending   MessageStatus = "pending"
	MessageStatusSent      MessageStatus = "sent"
	MessageStatusDelivered MessageStatus = "delivered"
	MessageStatusRead      MessageStatus = "read"
	MessageStatusFailed    MessageStatus = "failed"
)

// ContentType represents the type of message content
type ContentType string

const (
	ContentTypeText        ContentType = "text"
	ContentTypeImage       ContentType = "image"
	ContentTypeVideo       ContentType = "video"
	ContentTypeAudio       ContentType = "audio"
	ContentTypeDocument    ContentType = "document"
	ContentTypeLocation    ContentType = "location"
	ContentTypeContact     ContentType = "contact"
	ContentTypeTemplate    ContentType = "template"
	ContentTypeInteractive ContentType = "interactive"
)

// OutboundMessage represents a message to be sent via a channel adapter
type OutboundMessage struct {
	ID             string            `json:"id"`
	ConversationID string            `json:"conversation_id"`
	RecipientID    string            `json:"recipient_id"`
	ContentType    ContentType       `json:"content_type"`
	Content        string            `json:"content"`
	Metadata       map[string]string `json:"metadata,omitempty"`
	Attachments    []*Attachment     `json:"attachments,omitempty"`
}

// Attachment represents a file attachment
type Attachment struct {
	Type         string            `json:"type"`
	URL          string            `json:"url"`
	Filename     string            `json:"filename,omitempty"`
	MimeType     string            `json:"mime_type,omitempty"`
	SizeBytes    int64             `json:"size_bytes,omitempty"`
	ThumbnailURL string            `json:"thumbnail_url,omitempty"`
	Metadata     map[string]string `json:"metadata,omitempty"`
}

// SendResult represents the result of sending a message
type SendResult struct {
	Success    bool          `json:"success"`
	ExternalID string        `json:"external_id,omitempty"`
	Status     MessageStatus `json:"status"`
	Error      string        `json:"error,omitempty"`
	Timestamp  time.Time     `json:"timestamp"`
}

// Media represents media content for upload/download
type Media struct {
	ID           string            `json:"id,omitempty"`
	URL          string            `json:"url,omitempty"`
	Data         []byte            `json:"data,omitempty"`
	Filename     string            `json:"filename,omitempty"`
	MimeType     string            `json:"mime_type,omitempty"`
	SizeBytes    int64             `json:"size_bytes,omitempty"`
	ThumbnailURL string            `json:"thumbnail_url,omitempty"`
	Metadata     map[string]string `json:"metadata,omitempty"`
}

// MediaUpload represents the result of uploading media
type MediaUpload struct {
	Success  bool      `json:"success"`
	MediaID  string    `json:"media_id,omitempty"`
	URL      string    `json:"url,omitempty"`
	Error    string    `json:"error,omitempty"`
	ExpireAt time.Time `json:"expire_at,omitempty"`
}

// ChannelCapabilities describes what a channel adapter can do
type ChannelCapabilities struct {
	// Supported content types
	SupportedContentTypes []ContentType `json:"supported_content_types"`

	// Features
	SupportsMedia        bool `json:"supports_media"`
	SupportsLocation     bool `json:"supports_location"`
	SupportsTemplates    bool `json:"supports_templates"`
	SupportsInteractive  bool `json:"supports_interactive"`
	SupportsReadReceipts bool `json:"supports_read_receipts"`
	SupportsTypingIndicator bool `json:"supports_typing_indicator"`
	SupportsReactions    bool `json:"supports_reactions"`
	SupportsReplies      bool `json:"supports_replies"`
	SupportsForwarding   bool `json:"supports_forwarding"`

	// Limits
	MaxMessageLength  int   `json:"max_message_length"`
	MaxMediaSize      int64 `json:"max_media_size"`
	MaxAttachments    int   `json:"max_attachments"`

	// Supported media types
	SupportedMediaTypes []string `json:"supported_media_types"`
}

// ChannelInfo provides information about a channel adapter
type ChannelInfo struct {
	Type        ChannelType          `json:"type"`
	Name        string               `json:"name"`
	Description string               `json:"description"`
	Version     string               `json:"version"`
	Author      string               `json:"author"`
	Capabilities *ChannelCapabilities `json:"capabilities"`
}

// ConnectionStatus represents the connection state of a channel
type ConnectionStatus struct {
	Connected   bool      `json:"connected"`
	Status      string    `json:"status"`
	Error       string    `json:"error,omitempty"`
	LastConnect time.Time `json:"last_connect,omitempty"`
	Metadata    map[string]string `json:"metadata,omitempty"`
}

// InboundMessage represents an incoming message from a channel
type InboundMessage struct {
	ID          string            `json:"id"`
	ExternalID  string            `json:"external_id"`
	SenderID    string            `json:"sender_id"`
	SenderName  string            `json:"sender_name,omitempty"`
	ContentType ContentType       `json:"content_type"`
	Content     string            `json:"content"`
	Metadata    map[string]string `json:"metadata,omitempty"`
	Attachments []*Attachment     `json:"attachments,omitempty"`
	Timestamp   time.Time         `json:"timestamp"`
}

// StatusCallback represents a message status update from a channel
type StatusCallback struct {
	MessageID    string        `json:"message_id"`
	ExternalID   string        `json:"external_id"`
	Status       MessageStatus `json:"status"`
	ErrorMessage string        `json:"error_message,omitempty"`
	Timestamp    time.Time     `json:"timestamp"`
}

// TypingIndicator represents typing status
type TypingIndicator struct {
	RecipientID string `json:"recipient_id"`
	IsTyping    bool   `json:"is_typing"`
}

// ReadReceipt represents a read receipt to send
type ReadReceipt struct {
	RecipientID string `json:"recipient_id"`
	MessageID   string `json:"message_id"`
}
