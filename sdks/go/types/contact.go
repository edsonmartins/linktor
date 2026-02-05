package types

import "time"

// ContactIdentifier for a channel
type ContactIdentifier struct {
	ChannelType    ChannelType            `json:"channelType"`
	ChannelID      string                 `json:"channelId"`
	Identifier     string                 `json:"identifier"`
	DisplayName    string                 `json:"displayName,omitempty"`
	ProfilePicture string                 `json:"profilePicture,omitempty"`
	Metadata       map[string]interface{} `json:"metadata,omitempty"`
}

// Contact model
type Contact struct {
	ID                string                 `json:"id"`
	TenantID          string                 `json:"tenantId"`
	ExternalID        string                 `json:"externalId,omitempty"`
	Name              string                 `json:"name"`
	Email             string                 `json:"email,omitempty"`
	Phone             string                 `json:"phone,omitempty"`
	AvatarURL         string                 `json:"avatarUrl,omitempty"`
	Identifiers       []ContactIdentifier    `json:"identifiers"`
	CustomFields      map[string]interface{} `json:"customFields,omitempty"`
	Tags              []string               `json:"tags,omitempty"`
	Metadata          map[string]interface{} `json:"metadata,omitempty"`
	LastSeenAt        *time.Time             `json:"lastSeenAt,omitempty"`
	ConversationCount int                    `json:"conversationCount"`
	Timestamps
}

// CreateContactInput for creating contacts
type CreateContactInput struct {
	Name         string                 `json:"name"`
	Email        string                 `json:"email,omitempty"`
	Phone        string                 `json:"phone,omitempty"`
	ExternalID   string                 `json:"externalId,omitempty"`
	AvatarURL    string                 `json:"avatarUrl,omitempty"`
	CustomFields map[string]interface{} `json:"customFields,omitempty"`
	Tags         []string               `json:"tags,omitempty"`
}

// UpdateContactInput for updating contacts
type UpdateContactInput struct {
	Name         string                 `json:"name,omitempty"`
	Email        string                 `json:"email,omitempty"`
	Phone        string                 `json:"phone,omitempty"`
	ExternalID   string                 `json:"externalId,omitempty"`
	AvatarURL    string                 `json:"avatarUrl,omitempty"`
	CustomFields map[string]interface{} `json:"customFields,omitempty"`
	Tags         []string               `json:"tags,omitempty"`
}

// ListContactsParams for list request
type ListContactsParams struct {
	PaginationParams
	Search      string      `json:"search,omitempty"`
	Email       string      `json:"email,omitempty"`
	Phone       string      `json:"phone,omitempty"`
	Tags        []string    `json:"tags,omitempty"`
	ChannelType ChannelType `json:"channelType,omitempty"`
	SortBy      string      `json:"sortBy,omitempty"`
	SortOrder   string      `json:"sortOrder,omitempty"`
}
