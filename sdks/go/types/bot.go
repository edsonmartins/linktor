package types

// BotStatus type
type BotStatus string

const (
	BotStatusActive   BotStatus = "active"
	BotStatusInactive BotStatus = "inactive"
	BotStatusDraft    BotStatus = "draft"
)

// BotType type
type BotType string

const (
	BotTypeFlow   BotType = "flow"
	BotTypeAI     BotType = "ai"
	BotTypeHybrid BotType = "hybrid"
)

// Bot model
type Bot struct {
	ID               string                 `json:"id"`
	TenantID         string                 `json:"tenantId"`
	Name             string                 `json:"name"`
	Description      string                 `json:"description,omitempty"`
	Status           BotStatus              `json:"status"`
	Type             BotType                `json:"type"`
	Config           map[string]interface{} `json:"config"`
	ChannelIDs       []string               `json:"channelIds"`
	FlowID           string                 `json:"flowId,omitempty"`
	AgentID          string                 `json:"agentId,omitempty"`
	KnowledgeBaseIDs []string               `json:"knowledgeBaseIds,omitempty"`
	Metadata         map[string]interface{} `json:"metadata,omitempty"`
	Timestamps
}

// CreateBotInput for creating bots
type CreateBotInput struct {
	Name             string                 `json:"name"`
	Description      string                 `json:"description,omitempty"`
	Type             BotType                `json:"type"`
	Config           map[string]interface{} `json:"config,omitempty"`
	ChannelIDs       []string               `json:"channelIds,omitempty"`
	FlowID           string                 `json:"flowId,omitempty"`
	AgentID          string                 `json:"agentId,omitempty"`
	KnowledgeBaseIDs []string               `json:"knowledgeBaseIds,omitempty"`
	Metadata         map[string]interface{} `json:"metadata,omitempty"`
}

// UpdateBotInput for updating bots
type UpdateBotInput struct {
	Name             string                 `json:"name,omitempty"`
	Description      string                 `json:"description,omitempty"`
	Status           BotStatus              `json:"status,omitempty"`
	Config           map[string]interface{} `json:"config,omitempty"`
	ChannelIDs       []string               `json:"channelIds,omitempty"`
	FlowID           string                 `json:"flowId,omitempty"`
	AgentID          string                 `json:"agentId,omitempty"`
	KnowledgeBaseIDs []string               `json:"knowledgeBaseIds,omitempty"`
	Metadata         map[string]interface{} `json:"metadata,omitempty"`
}

// ListBotsParams for list request
type ListBotsParams struct {
	PaginationParams
	Status    BotStatus `json:"status,omitempty"`
	Type      BotType   `json:"type,omitempty"`
	ChannelID string    `json:"channelId,omitempty"`
	Search    string    `json:"search,omitempty"`
}
