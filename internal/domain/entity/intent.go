package entity

import (
	"time"
)

// Sentiment represents the sentiment of a message
type Sentiment string

const (
	SentimentPositive Sentiment = "positive"
	SentimentNeutral  Sentiment = "neutral"
	SentimentNegative Sentiment = "negative"
)

// Intent represents a classified intent from a message
type Intent struct {
	Name       string            `json:"name"`
	Confidence float64           `json:"confidence"`
	Entities   map[string]string `json:"entities,omitempty"`
}

// NewIntent creates a new intent
func NewIntent(name string, confidence float64) *Intent {
	return &Intent{
		Name:       name,
		Confidence: confidence,
		Entities:   make(map[string]string),
	}
}

// IsHighConfidence returns true if confidence is above threshold
func (i *Intent) IsHighConfidence(threshold float64) bool {
	return i.Confidence >= threshold
}

// AddEntity adds an entity to the intent
func (i *Intent) AddEntity(key, value string) {
	if i.Entities == nil {
		i.Entities = make(map[string]string)
	}
	i.Entities[key] = value
}

// ContextMessage represents a message in the context window
type ContextMessage struct {
	Role      string    `json:"role"` // user, assistant, system
	Content   string    `json:"content"`
	Timestamp time.Time `json:"timestamp"`
	MessageID string    `json:"message_id,omitempty"`
}

// NewContextMessage creates a new context message
func NewContextMessage(role, content string, messageID string) *ContextMessage {
	return &ContextMessage{
		Role:      role,
		Content:   content,
		Timestamp: time.Now(),
		MessageID: messageID,
	}
}

// ConversationContext holds the AI context for a conversation
type ConversationContext struct {
	ID             string                 `json:"id"`
	ConversationID string                 `json:"conversation_id"`
	BotID          *string                `json:"bot_id,omitempty"`
	Intent         *Intent                `json:"intent,omitempty"`
	Entities       map[string]interface{} `json:"entities,omitempty"`
	Sentiment      Sentiment              `json:"sentiment"`
	ContextWindow  []ContextMessage       `json:"context_window"`
	State          map[string]interface{} `json:"state,omitempty"` // Flow state variables
	LastAnalysisAt *time.Time             `json:"last_analysis_at,omitempty"`
	CreatedAt      time.Time              `json:"created_at"`
	UpdatedAt      time.Time              `json:"updated_at"`
}

// NewConversationContext creates a new conversation context
func NewConversationContext(conversationID string) *ConversationContext {
	now := time.Now()
	return &ConversationContext{
		ConversationID: conversationID,
		Entities:       make(map[string]interface{}),
		Sentiment:      SentimentNeutral,
		ContextWindow:  []ContextMessage{},
		State:          make(map[string]interface{}),
		CreatedAt:      now,
		UpdatedAt:      now,
	}
}

// SetBot assigns a bot to the context
func (c *ConversationContext) SetBot(botID string) {
	c.BotID = &botID
	c.UpdatedAt = time.Now()
}

// ClearBot removes the bot assignment
func (c *ConversationContext) ClearBot() {
	c.BotID = nil
	c.UpdatedAt = time.Now()
}

// SetIntent updates the detected intent
func (c *ConversationContext) SetIntent(intent *Intent) {
	c.Intent = intent
	now := time.Now()
	c.LastAnalysisAt = &now
	c.UpdatedAt = now
}

// SetSentiment updates the sentiment
func (c *ConversationContext) SetSentiment(sentiment Sentiment) {
	c.Sentiment = sentiment
	c.UpdatedAt = time.Now()
}

// AddMessage adds a message to the context window
func (c *ConversationContext) AddMessage(msg ContextMessage) {
	c.ContextWindow = append(c.ContextWindow, msg)
	c.UpdatedAt = time.Now()
}

// AddUserMessage adds a user message to the context window
func (c *ConversationContext) AddUserMessage(content, messageID string) {
	c.AddMessage(*NewContextMessage("user", content, messageID))
}

// AddAssistantMessage adds an assistant message to the context window
func (c *ConversationContext) AddAssistantMessage(content, messageID string) {
	c.AddMessage(*NewContextMessage("assistant", content, messageID))
}

// AddSystemMessage adds a system message to the context window
func (c *ConversationContext) AddSystemMessage(content string) {
	c.AddMessage(*NewContextMessage("system", content, ""))
}

// TrimContextWindow trims the context window to max size, keeping most recent
func (c *ConversationContext) TrimContextWindow(maxSize int) {
	if len(c.ContextWindow) > maxSize {
		c.ContextWindow = c.ContextWindow[len(c.ContextWindow)-maxSize:]
		c.UpdatedAt = time.Now()
	}
}

// GetContextMessages returns the context window as a slice of messages
func (c *ConversationContext) GetContextMessages() []ContextMessage {
	return c.ContextWindow
}

// SetEntity sets an entity value
func (c *ConversationContext) SetEntity(key string, value interface{}) {
	if c.Entities == nil {
		c.Entities = make(map[string]interface{})
	}
	c.Entities[key] = value
	c.UpdatedAt = time.Now()
}

// GetEntity gets an entity value
func (c *ConversationContext) GetEntity(key string) (interface{}, bool) {
	if c.Entities == nil {
		return nil, false
	}
	val, ok := c.Entities[key]
	return val, ok
}

// SetStateValue sets a state variable
func (c *ConversationContext) SetStateValue(key string, value interface{}) {
	if c.State == nil {
		c.State = make(map[string]interface{})
	}
	c.State[key] = value
	c.UpdatedAt = time.Now()
}

// GetStateValue gets a state variable
func (c *ConversationContext) GetStateValue(key string) (interface{}, bool) {
	if c.State == nil {
		return nil, false
	}
	val, ok := c.State[key]
	return val, ok
}

// ClearState clears all state variables
func (c *ConversationContext) ClearState() {
	c.State = make(map[string]interface{})
	c.UpdatedAt = time.Now()
}

// NeedsAnalysis checks if the context needs to be re-analyzed
func (c *ConversationContext) NeedsAnalysis(maxAge time.Duration) bool {
	if c.LastAnalysisAt == nil {
		return true
	}
	return time.Since(*c.LastAnalysisAt) > maxAge
}

// IntentResult represents the result of intent classification
type IntentResult struct {
	Intent     *Intent   `json:"intent"`
	Candidates []Intent  `json:"candidates,omitempty"`
	RawScore   float64   `json:"raw_score,omitempty"`
}

// SentimentResult represents the result of sentiment analysis
type SentimentResult struct {
	Sentiment  Sentiment `json:"sentiment"`
	Score      float64   `json:"score"`
	Confidence float64   `json:"confidence"`
}
