package entity

import (
	"time"
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

// SenderType represents who sent the message
type SenderType string

const (
	SenderTypeContact SenderType = "contact"
	SenderTypeUser    SenderType = "user"
	SenderTypeSystem  SenderType = "system"
	SenderTypeBot     SenderType = "bot"
)

// MessageAttachment represents a file attached to a message
type MessageAttachment struct {
	ID           string            `json:"id"`
	MessageID    string            `json:"message_id"`
	Type         string            `json:"type"`
	Filename     string            `json:"filename,omitempty"`
	MimeType     string            `json:"mime_type,omitempty"`
	SizeBytes    int64             `json:"size_bytes,omitempty"`
	URL          string            `json:"url"`
	ThumbnailURL string            `json:"thumbnail_url,omitempty"`
	Metadata     map[string]string `json:"metadata,omitempty"`
	CreatedAt    time.Time         `json:"created_at"`
}

// Message represents a chat message
type Message struct {
	ID             string               `json:"id"`
	ConversationID string               `json:"conversation_id"`
	SenderType     SenderType           `json:"sender_type"`
	SenderID       string               `json:"sender_id,omitempty"`
	ContentType    ContentType          `json:"content_type"`
	Content        string               `json:"content,omitempty"`
	Metadata       map[string]string    `json:"metadata,omitempty"`
	Status         MessageStatus        `json:"status"`
	ExternalID     string               `json:"external_id,omitempty"`
	ErrorMessage   string               `json:"error_message,omitempty"`
	Attachments    []*MessageAttachment `json:"attachments,omitempty"`
	SentAt         *time.Time           `json:"sent_at,omitempty"`
	DeliveredAt    *time.Time           `json:"delivered_at,omitempty"`
	ReadAt         *time.Time           `json:"read_at,omitempty"`
	CreatedAt      time.Time            `json:"created_at"`
}

// NewMessage creates a new message
func NewMessage(conversationID string, senderType SenderType, senderID string, contentType ContentType, content string) *Message {
	now := time.Now()
	return &Message{
		ConversationID: conversationID,
		SenderType:     senderType,
		SenderID:       senderID,
		ContentType:    contentType,
		Content:        content,
		Metadata:       make(map[string]string),
		Status:         MessageStatusPending,
		Attachments:    []*MessageAttachment{},
		CreatedAt:      now,
	}
}

// MarkAsSent marks the message as sent
func (m *Message) MarkAsSent() {
	now := time.Now()
	m.Status = MessageStatusSent
	m.SentAt = &now
}

// MarkAsDelivered marks the message as delivered
func (m *Message) MarkAsDelivered() {
	now := time.Now()
	m.Status = MessageStatusDelivered
	m.DeliveredAt = &now
}

// MarkAsRead marks the message as read
func (m *Message) MarkAsRead() {
	now := time.Now()
	m.Status = MessageStatusRead
	m.ReadAt = &now
}

// MarkAsFailed marks the message as failed
func (m *Message) MarkAsFailed(errorMessage string) {
	m.Status = MessageStatusFailed
	m.ErrorMessage = errorMessage
}

// IsFromContact returns true if the message is from a contact
func (m *Message) IsFromContact() bool {
	return m.SenderType == SenderTypeContact
}

// IsFromUser returns true if the message is from a user/agent
func (m *Message) IsFromUser() bool {
	return m.SenderType == SenderTypeUser
}
