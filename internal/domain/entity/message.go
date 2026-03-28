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
	ContentTypeSticker     ContentType = "sticker"
	ContentTypePoll        ContentType = "poll"
	ContentTypeReaction    ContentType = "reaction"
)

// SenderType represents who sent the message
type SenderType string

const (
	SenderTypeContact SenderType = "contact"
	SenderTypeUser    SenderType = "user"
	SenderTypeSystem  SenderType = "system"
	SenderTypeBot     SenderType = "bot"
)

// MessageSource represents the source/origin of a message
type MessageSource string

const (
	MessageSourceAPI         MessageSource = "api"          // Message sent via Cloud API
	MessageSourceBusinessApp MessageSource = "business_app" // Message sent via WhatsApp Business App (echo)
	MessageSourceImported    MessageSource = "imported"     // Message imported from chat history
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

	// Source tracking for WhatsApp Coexistence
	Source     MessageSource `json:"source,omitempty"`      // Where the message originated (api, business_app, imported)
	IsImported bool          `json:"is_imported,omitempty"` // Whether this message was imported from history
	ImportedAt *time.Time    `json:"imported_at,omitempty"` // When this message was imported

	// Edit/Delete tracking
	EditedAt  *time.Time `json:"edited_at,omitempty"`  // When the message was last edited
	DeletedAt *time.Time `json:"deleted_at,omitempty"` // When the message was deleted/revoked
	IsEdited  bool       `json:"is_edited,omitempty"`  // Whether the message has been edited
	IsDeleted bool       `json:"is_deleted,omitempty"` // Whether the message has been deleted/revoked

	// Reactions
	Reactions []Reaction `json:"reactions,omitempty"` // Emoji reactions on this message

	// Reply context
	ReplyToID string `json:"reply_to_id,omitempty"` // ID of the message being replied to
}

// Reaction represents an emoji reaction on a message
type Reaction struct {
	UserID    string    `json:"user_id"`
	Emoji     string    `json:"emoji"`
	Timestamp time.Time `json:"timestamp"`
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
		Source:         MessageSourceAPI, // Default to API source
		IsImported:     false,
	}
}

// NewEchoMessage creates a new message from a WhatsApp Business App echo
func NewEchoMessage(conversationID string, senderType SenderType, senderID string, contentType ContentType, content string) *Message {
	msg := NewMessage(conversationID, senderType, senderID, contentType, content)
	msg.Source = MessageSourceBusinessApp
	msg.Status = MessageStatusDelivered // Echoes are already delivered
	return msg
}

// NewImportedMessage creates a new message from chat history import
func NewImportedMessage(conversationID string, senderType SenderType, senderID string, contentType ContentType, content string, originalCreatedAt time.Time) *Message {
	now := time.Now()
	return &Message{
		ConversationID: conversationID,
		SenderType:     senderType,
		SenderID:       senderID,
		ContentType:    contentType,
		Content:        content,
		Metadata:       make(map[string]string),
		Status:         MessageStatusDelivered, // Imported messages are historical
		Attachments:    []*MessageAttachment{},
		CreatedAt:      originalCreatedAt,
		Source:         MessageSourceImported,
		IsImported:     true,
		ImportedAt:     &now,
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

// Edit updates the message content and marks it as edited
func (m *Message) Edit(newContent string) {
	now := time.Now()
	m.Content = newContent
	m.EditedAt = &now
	m.IsEdited = true
}

// Revoke marks the message as deleted/revoked
func (m *Message) Revoke() {
	now := time.Now()
	m.DeletedAt = &now
	m.IsDeleted = true
	m.Content = "" // Clear content on revoke
}

// AddReaction adds or updates a reaction from a user
func (m *Message) AddReaction(userID, emoji string) {
	// Remove existing reaction from this user
	m.RemoveReaction(userID)
	if emoji != "" {
		m.Reactions = append(m.Reactions, Reaction{
			UserID:    userID,
			Emoji:     emoji,
			Timestamp: time.Now(),
		})
	}
}

// RemoveReaction removes a reaction from a user
func (m *Message) RemoveReaction(userID string) {
	filtered := make([]Reaction, 0, len(m.Reactions))
	for _, r := range m.Reactions {
		if r.UserID != userID {
			filtered = append(filtered, r)
		}
	}
	m.Reactions = filtered
}

// IsFromContact returns true if the message is from a contact
func (m *Message) IsFromContact() bool {
	return m.SenderType == SenderTypeContact
}

// IsFromUser returns true if the message is from a user/agent
func (m *Message) IsFromUser() bool {
	return m.SenderType == SenderTypeUser
}

// IsFromAPI returns true if the message was sent via Cloud API
func (m *Message) IsFromAPI() bool {
	return m.Source == MessageSourceAPI || m.Source == ""
}

// IsFromBusinessApp returns true if the message was sent via WhatsApp Business App
func (m *Message) IsFromBusinessApp() bool {
	return m.Source == MessageSourceBusinessApp
}

// IsImportedMessage returns true if the message was imported from chat history
func (m *Message) IsImportedMessage() bool {
	return m.IsImported || m.Source == MessageSourceImported
}
