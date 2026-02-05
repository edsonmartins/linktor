package types

// ChannelStatus type
type ChannelStatus string

const (
	ChannelStatusActive     ChannelStatus = "active"
	ChannelStatusInactive   ChannelStatus = "inactive"
	ChannelStatusConnecting ChannelStatus = "connecting"
	ChannelStatusError      ChannelStatus = "error"
)

// Channel model
type Channel struct {
	ID         string                 `json:"id"`
	TenantID   string                 `json:"tenantId"`
	Name       string                 `json:"name"`
	Type       ChannelType            `json:"type"`
	Status     ChannelStatus          `json:"status"`
	Config     map[string]interface{} `json:"config"`
	WebhookURL string                 `json:"webhookUrl,omitempty"`
	Metadata   map[string]interface{} `json:"metadata,omitempty"`
	Timestamps
}

// CreateChannelInput for creating channels
type CreateChannelInput struct {
	Name     string                 `json:"name"`
	Type     ChannelType            `json:"type"`
	Config   map[string]interface{} `json:"config"`
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

// UpdateChannelInput for updating channels
type UpdateChannelInput struct {
	Name     string                 `json:"name,omitempty"`
	Config   map[string]interface{} `json:"config,omitempty"`
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

// ListChannelsParams for list request
type ListChannelsParams struct {
	PaginationParams
	Type   ChannelType   `json:"type,omitempty"`
	Status ChannelStatus `json:"status,omitempty"`
	Search string        `json:"search,omitempty"`
}
