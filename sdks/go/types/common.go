package types

import "time"

// PaginationParams for list requests
type PaginationParams struct {
	Limit  int    `json:"limit,omitempty"`
	Offset int    `json:"offset,omitempty"`
	Cursor string `json:"cursor,omitempty"`
}

// Pagination info in responses
type Pagination struct {
	Total      int    `json:"total"`
	Limit      int    `json:"limit"`
	Offset     int    `json:"offset"`
	HasMore    bool   `json:"hasMore"`
	NextCursor string `json:"nextCursor,omitempty"`
}

// PaginatedResponse wrapper
type PaginatedResponse[T any] struct {
	Data       []T        `json:"data"`
	Pagination Pagination `json:"pagination"`
}

// APIError details
type APIError struct {
	Code    string                 `json:"code"`
	Message string                 `json:"message"`
	Details map[string]interface{} `json:"details,omitempty"`
}

// MessageDirection type
type MessageDirection string

const (
	MessageDirectionInbound  MessageDirection = "inbound"
	MessageDirectionOutbound MessageDirection = "outbound"
)

// MessageStatus type
type MessageStatus string

const (
	MessageStatusPending   MessageStatus = "pending"
	MessageStatusSent      MessageStatus = "sent"
	MessageStatusDelivered MessageStatus = "delivered"
	MessageStatusRead      MessageStatus = "read"
	MessageStatusFailed    MessageStatus = "failed"
)

// ChannelType type
type ChannelType string

const (
	ChannelTypeWhatsApp  ChannelType = "whatsapp"
	ChannelTypeTelegram  ChannelType = "telegram"
	ChannelTypeFacebook  ChannelType = "facebook"
	ChannelTypeInstagram ChannelType = "instagram"
	ChannelTypeWebchat   ChannelType = "webchat"
	ChannelTypeSMS       ChannelType = "sms"
	ChannelTypeEmail     ChannelType = "email"
	ChannelTypeRCS       ChannelType = "rcs"
)

// ContentType type
type ContentType string

const (
	ContentTypeText     ContentType = "text"
	ContentTypeImage    ContentType = "image"
	ContentTypeVideo    ContentType = "video"
	ContentTypeAudio    ContentType = "audio"
	ContentTypeDocument ContentType = "document"
	ContentTypeLocation ContentType = "location"
	ContentTypeContact  ContentType = "contact"
	ContentTypeSticker  ContentType = "sticker"
)

// Timestamps mixin
type Timestamps struct {
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}
