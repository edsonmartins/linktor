package entity

import (
	"time"
)

// BotType represents the type of bot
type BotType string

const (
	BotTypeAI        BotType = "ai"
	BotTypeRuleBased BotType = "rule_based"
	BotTypeHybrid    BotType = "hybrid"
)

// AIProviderType represents the AI provider
type AIProviderType string

const (
	AIProviderOpenAI    AIProviderType = "openai"
	AIProviderAnthropic AIProviderType = "anthropic"
	AIProviderOllama    AIProviderType = "ollama"
)

// BotStatus represents the status of a bot
type BotStatus string

const (
	BotStatusActive   BotStatus = "active"
	BotStatusInactive BotStatus = "inactive"
	BotStatusTraining BotStatus = "training"
)

// EscalationCondition represents when to escalate
type EscalationCondition string

const (
	EscalationConditionLowConfidence EscalationCondition = "low_confidence"
	EscalationConditionKeyword       EscalationCondition = "keyword"
	EscalationConditionSentiment     EscalationCondition = "sentiment"
	EscalationConditionIntent        EscalationCondition = "intent"
	EscalationConditionUserRequest   EscalationCondition = "user_request"
)

// EscalationAction represents what to do on escalation
type EscalationAction string

const (
	EscalationActionEscalate EscalationAction = "escalate"
	EscalationActionNotify   EscalationAction = "notify"
	EscalationActionTag      EscalationAction = "tag"
)

// EscalationRule defines when and how to escalate a conversation
type EscalationRule struct {
	Condition EscalationCondition `json:"condition"` // low_confidence, keyword, sentiment, intent
	Value     string              `json:"value"`     // threshold value, keyword, sentiment type
	Action    EscalationAction    `json:"action"`    // escalate, notify, tag
	Priority  string              `json:"priority"`  // high, urgent, normal
}

// WorkingHours defines when the bot should be active
type WorkingHours struct {
	Enabled  bool             `json:"enabled"`
	Timezone string           `json:"timezone"`
	Schedule []DaySchedule    `json:"schedule"`
}

// DaySchedule defines working hours for a specific day
type DaySchedule struct {
	Day       int    `json:"day"` // 0 = Sunday, 6 = Saturday
	StartTime string `json:"start_time"` // "09:00"
	EndTime   string `json:"end_time"`   // "18:00"
}

// BotConfig holds the bot configuration
type BotConfig struct {
	SystemPrompt        string           `json:"system_prompt"`
	Temperature         float64          `json:"temperature"`
	MaxTokens           int              `json:"max_tokens"`
	ConfidenceThreshold float64          `json:"confidence_threshold"` // Min confidence for auto-response
	EscalationRules     []EscalationRule `json:"escalation_rules"`
	KnowledgeBaseID     *string          `json:"knowledge_base_id"`
	WelcomeMessage      *string          `json:"welcome_message"`
	FallbackMessage     string           `json:"fallback_message"`
	WorkingHours        *WorkingHours    `json:"working_hours"`
	ContextWindowSize   int              `json:"context_window_size"` // Number of messages to include
	EnabledIntents      []string         `json:"enabled_intents"`     // Intents the bot can handle
	MaxResponseLength   int              `json:"max_response_length"`
	Tools               []*Tool          `json:"tools,omitempty"`       // Custom tools available to the bot
	EnableVRETools      bool             `json:"enable_vre_tools"`      // Enable built-in VRE visual tools
	ToolChoice          string           `json:"tool_choice,omitempty"` // auto, none, required
}

// Bot represents an AI chatbot configuration
type Bot struct {
	ID          string         `json:"id"`
	TenantID    string         `json:"tenant_id"`
	Name        string         `json:"name"`
	Type        BotType        `json:"type"`     // ai, rule_based, hybrid
	Provider    AIProviderType `json:"provider"` // openai, anthropic, ollama
	Model       string         `json:"model"`    // gpt-4, claude-3, llama3
	Config      BotConfig      `json:"config"`
	Status      BotStatus      `json:"status"`   // active, inactive, training
	Channels    []string       `json:"channels"` // channel IDs assigned to this bot
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
}

// NewBot creates a new bot with default configuration
func NewBot(tenantID, name string, botType BotType, provider AIProviderType, model string) *Bot {
	now := time.Now()
	return &Bot{
		TenantID: tenantID,
		Name:     name,
		Type:     botType,
		Provider: provider,
		Model:    model,
		Config: BotConfig{
			SystemPrompt:        "You are a helpful customer service assistant.",
			Temperature:         0.7,
			MaxTokens:           1024,
			ConfidenceThreshold: 0.7,
			EscalationRules:     []EscalationRule{},
			FallbackMessage:     "I'm sorry, I couldn't understand your request. Let me connect you with a human agent.",
			ContextWindowSize:   10,
			MaxResponseLength:   500,
		},
		Status:    BotStatusInactive,
		Channels:  []string{},
		CreatedAt: now,
		UpdatedAt: now,
	}
}

// IsActive returns true if the bot is active
func (b *Bot) IsActive() bool {
	return b.Status == BotStatusActive
}

// Activate activates the bot
func (b *Bot) Activate() {
	b.Status = BotStatusActive
	b.UpdatedAt = time.Now()
}

// Deactivate deactivates the bot
func (b *Bot) Deactivate() {
	b.Status = BotStatusInactive
	b.UpdatedAt = time.Now()
}

// AssignChannel assigns a channel to the bot
func (b *Bot) AssignChannel(channelID string) {
	// Check if already assigned
	for _, ch := range b.Channels {
		if ch == channelID {
			return
		}
	}
	b.Channels = append(b.Channels, channelID)
	b.UpdatedAt = time.Now()
}

// UnassignChannel removes a channel from the bot
func (b *Bot) UnassignChannel(channelID string) {
	for i, ch := range b.Channels {
		if ch == channelID {
			b.Channels = append(b.Channels[:i], b.Channels[i+1:]...)
			b.UpdatedAt = time.Now()
			return
		}
	}
}

// HasChannel returns true if the bot is assigned to the channel
func (b *Bot) HasChannel(channelID string) bool {
	for _, ch := range b.Channels {
		if ch == channelID {
			return true
		}
	}
	return false
}

// SetSystemPrompt updates the system prompt
func (b *Bot) SetSystemPrompt(prompt string) {
	b.Config.SystemPrompt = prompt
	b.UpdatedAt = time.Now()
}

// AddEscalationRule adds an escalation rule
func (b *Bot) AddEscalationRule(rule EscalationRule) {
	b.Config.EscalationRules = append(b.Config.EscalationRules, rule)
	b.UpdatedAt = time.Now()
}

// GetTools returns all tools available to the bot (custom + VRE if enabled)
func (b *Bot) GetTools() []*Tool {
	var tools []*Tool

	// Add custom tools
	if len(b.Config.Tools) > 0 {
		tools = append(tools, b.Config.Tools...)
	}

	// Add VRE tools if enabled
	if b.Config.EnableVRETools {
		tools = append(tools, BuiltInVRETools()...)
	}

	return tools
}

// GetToolByName returns a tool by name
func (b *Bot) GetToolByName(name string) *Tool {
	for _, tool := range b.GetTools() {
		if tool.Name == name {
			return tool
		}
	}
	return nil
}

// ShouldEscalate checks if a conversation should be escalated based on rules
func (b *Bot) ShouldEscalate(confidence float64, sentiment string, keywords []string) (bool, *EscalationRule) {
	for _, rule := range b.Config.EscalationRules {
		switch rule.Condition {
		case EscalationConditionLowConfidence:
			// Value should be a threshold like "0.5"
			// If confidence is below threshold, escalate
			// This is a simplified check - actual implementation would parse the value
			if confidence < b.Config.ConfidenceThreshold {
				return true, &rule
			}
		case EscalationConditionSentiment:
			if sentiment == rule.Value {
				return true, &rule
			}
		case EscalationConditionKeyword:
			for _, kw := range keywords {
				if kw == rule.Value {
					return true, &rule
				}
			}
		}
	}
	return false, nil
}

// AIResponse represents a response generated by the AI
type AIResponse struct {
	ID           string                 `json:"id"`
	MessageID    string                 `json:"message_id"`
	BotID        string                 `json:"bot_id"`
	Prompt       map[string]interface{} `json:"prompt"`
	Response     string                 `json:"response"`
	Confidence   float64                `json:"confidence"`
	TokensUsed   int                    `json:"tokens_used"`
	LatencyMs    int                    `json:"latency_ms"`
	Model        string                 `json:"model"`
	CreatedAt    time.Time              `json:"created_at"`
}

// NewAIResponse creates a new AI response record
func NewAIResponse(messageID, botID string, prompt map[string]interface{}, response string, confidence float64, tokensUsed, latencyMs int, model string) *AIResponse {
	return &AIResponse{
		MessageID:  messageID,
		BotID:      botID,
		Prompt:     prompt,
		Response:   response,
		Confidence: confidence,
		TokensUsed: tokensUsed,
		LatencyMs:  latencyMs,
		Model:      model,
		CreatedAt:  time.Now(),
	}
}
