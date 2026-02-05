package types

import "time"

// ConversationStatus type
type ConversationStatus string

const (
	ConversationStatusOpen     ConversationStatus = "open"
	ConversationStatusPending  ConversationStatus = "pending"
	ConversationStatusResolved ConversationStatus = "resolved"
	ConversationStatusSnoozed  ConversationStatus = "snoozed"
)

// SenderType type
type SenderType string

const (
	SenderTypeContact SenderType = "contact"
	SenderTypeAgent   SenderType = "agent"
	SenderTypeBot     SenderType = "bot"
	SenderTypeSystem  SenderType = "system"
)

// MediaContent in messages
type MediaContent struct {
	URL          string `json:"url"`
	MimeType     string `json:"mimeType"`
	Filename     string `json:"filename,omitempty"`
	Size         int64  `json:"size,omitempty"`
	Caption      string `json:"caption,omitempty"`
	ThumbnailURL string `json:"thumbnailUrl,omitempty"`
}

// LocationContent in messages
type LocationContent struct {
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
	Name      string  `json:"name,omitempty"`
	Address   string  `json:"address,omitempty"`
}

// ButtonContent in messages
type ButtonContent struct {
	Type    string `json:"type"` // reply, url, call
	Text    string `json:"text"`
	Payload string `json:"payload,omitempty"`
	URL     string `json:"url,omitempty"`
	Phone   string `json:"phone,omitempty"`
}

// MessageContent structure
type MessageContent struct {
	Text     string           `json:"text,omitempty"`
	Media    *MediaContent    `json:"media,omitempty"`
	Location *LocationContent `json:"location,omitempty"`
	Buttons  []ButtonContent  `json:"buttons,omitempty"`
}

// Message model
type Message struct {
	ID             string                 `json:"id"`
	ConversationID string                 `json:"conversationId"`
	Direction      MessageDirection       `json:"direction"`
	ContentType    ContentType            `json:"contentType"`
	Content        MessageContent         `json:"content"`
	Status         MessageStatus          `json:"status"`
	ExternalID     string                 `json:"externalId,omitempty"`
	SenderID       string                 `json:"senderId,omitempty"`
	SenderType     SenderType             `json:"senderType"`
	DeliveredAt    *time.Time             `json:"deliveredAt,omitempty"`
	ReadAt         *time.Time             `json:"readAt,omitempty"`
	FailedReason   string                 `json:"failedReason,omitempty"`
	Metadata       map[string]interface{} `json:"metadata,omitempty"`
	Timestamps
}

// Conversation model
type Conversation struct {
	ID            string                 `json:"id"`
	TenantID      string                 `json:"tenantId"`
	ChannelID     string                 `json:"channelId"`
	ChannelType   ChannelType            `json:"channelType"`
	ContactID     string                 `json:"contactId"`
	Status        ConversationStatus     `json:"status"`
	AssignedTo    string                 `json:"assignedTo,omitempty"`
	AssignedAt    *time.Time             `json:"assignedAt,omitempty"`
	LastMessage   *Message               `json:"lastMessage,omitempty"`
	LastMessageAt *time.Time             `json:"lastMessageAt,omitempty"`
	UnreadCount   int                    `json:"unreadCount"`
	Metadata      map[string]interface{} `json:"metadata,omitempty"`
	Tags          []string               `json:"tags,omitempty"`
	Timestamps
}

// ListConversationsParams for list request
type ListConversationsParams struct {
	PaginationParams
	Status      ConversationStatus `json:"status,omitempty"`
	ChannelID   string             `json:"channelId,omitempty"`
	ChannelType ChannelType        `json:"channelType,omitempty"`
	AssignedTo  string             `json:"assignedTo,omitempty"`
	ContactID   string             `json:"contactId,omitempty"`
	Tags        []string           `json:"tags,omitempty"`
	Search      string             `json:"search,omitempty"`
	SortBy      string             `json:"sortBy,omitempty"`
	SortOrder   string             `json:"sortOrder,omitempty"`
}

// SendMessageInput for sending messages
type SendMessageInput struct {
	Text     string                 `json:"text,omitempty"`
	Media    *MediaContent          `json:"media,omitempty"`
	Location *LocationContent       `json:"location,omitempty"`
	Buttons  []ButtonContent        `json:"buttons,omitempty"`
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

// UpdateConversationInput for updating conversations
type UpdateConversationInput struct {
	Status     ConversationStatus     `json:"status,omitempty"`
	AssignedTo *string                `json:"assignedTo,omitempty"`
	Tags       []string               `json:"tags,omitempty"`
	Metadata   map[string]interface{} `json:"metadata,omitempty"`
}
