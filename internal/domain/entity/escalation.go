package entity

import (
	"time"
)

// EscalationPriority represents the priority of an escalation
type EscalationPriority string

const (
	EscalationPriorityLow    EscalationPriority = "low"
	EscalationPriorityNormal EscalationPriority = "normal"
	EscalationPriorityHigh   EscalationPriority = "high"
	EscalationPriorityUrgent EscalationPriority = "urgent"
)

// EscalationReason represents why a conversation was escalated
type EscalationReason string

const (
	EscalationReasonLowConfidence   EscalationReason = "low_confidence"
	EscalationReasonNegativeSentiment EscalationReason = "negative_sentiment"
	EscalationReasonKeywordTrigger  EscalationReason = "keyword_trigger"
	EscalationReasonUserRequest     EscalationReason = "user_request"
	EscalationReasonFlowAction      EscalationReason = "flow_action"
	EscalationReasonBotFailure      EscalationReason = "bot_failure"
	EscalationReasonComplexQuery    EscalationReason = "complex_query"
	EscalationReasonManual          EscalationReason = "manual"
)

// EscalationContext represents the full context for an escalated conversation
// This is provided to human agents to give them complete context
type EscalationContext struct {
	// Identification
	ID             string `json:"id"`
	ConversationID string `json:"conversation_id"`
	TenantID       string `json:"tenant_id"`

	// Summary and Analysis
	Summary          string           `json:"summary"`           // AI-generated conversation summary
	DetectedIntent   string           `json:"detected_intent,omitempty"`
	Sentiment        string           `json:"sentiment,omitempty"`
	SentimentScore   float64          `json:"sentiment_score,omitempty"`
	EscalationReason EscalationReason `json:"escalation_reason"`
	ReasonDetail     string           `json:"reason_detail,omitempty"`

	// Priority and Assignment
	Priority         EscalationPriority `json:"priority"`
	SuggestedTeam    string             `json:"suggested_team,omitempty"`    // Based on intent/tags
	SuggestedUserID  string             `json:"suggested_user_id,omitempty"` // Based on skills

	// Collected Information
	CollectedEntities map[string]string `json:"collected_entities,omitempty"` // From flow or AI
	FlowData          map[string]string `json:"flow_data,omitempty"`          // Data collected during flows
	Tags              []string          `json:"tags,omitempty"`               // Conversation tags

	// Conversation History
	LastMessages     []EscalationMessage `json:"last_messages"`      // Last N messages
	MessageCount     int                 `json:"message_count"`      // Total messages in conversation
	BotAttempts      int                 `json:"bot_attempts"`       // How many bot responses
	BotSuccessRate   float64             `json:"bot_success_rate"`   // Bot confidence average

	// Customer Information
	Customer *EscalationCustomer `json:"customer,omitempty"`

	// Channel Information
	ChannelType string `json:"channel_type"`
	ChannelName string `json:"channel_name,omitempty"`

	// Bot Information
	BotID   string `json:"bot_id,omitempty"`
	BotName string `json:"bot_name,omitempty"`

	// Active Flow Information
	ActiveFlowID   string `json:"active_flow_id,omitempty"`
	ActiveFlowName string `json:"active_flow_name,omitempty"`
	FlowNodeID     string `json:"flow_node_id,omitempty"` // Where in the flow

	// Timing
	ConversationStartedAt time.Time  `json:"conversation_started_at"`
	EscalatedAt           time.Time  `json:"escalated_at"`
	FirstResponseAt       *time.Time `json:"first_response_at,omitempty"`
	WaitTimeSeconds       int64      `json:"wait_time_seconds"` // Time since escalation
}

// EscalationMessage represents a simplified message in escalation context
type EscalationMessage struct {
	ID         string    `json:"id"`
	SenderType string    `json:"sender_type"` // contact, user, bot
	SenderName string    `json:"sender_name,omitempty"`
	Content    string    `json:"content"`
	Timestamp  time.Time `json:"timestamp"`
	IsBot      bool      `json:"is_bot"`
}

// EscalationCustomer represents customer info in escalation context
type EscalationCustomer struct {
	ID          string            `json:"id"`
	Name        string            `json:"name"`
	Email       string            `json:"email,omitempty"`
	Phone       string            `json:"phone,omitempty"`
	AvatarURL   string            `json:"avatar_url,omitempty"`
	Tags        []string          `json:"tags,omitempty"`
	CustomFields map[string]string `json:"custom_fields,omitempty"`

	// History
	TotalConversations int    `json:"total_conversations"`
	LastConversation   string `json:"last_conversation,omitempty"`
	IsVIP              bool   `json:"is_vip"`
}

// NewEscalationContext creates a new escalation context
func NewEscalationContext(conversationID, tenantID string, reason EscalationReason) *EscalationContext {
	return &EscalationContext{
		ConversationID:    conversationID,
		TenantID:          tenantID,
		EscalationReason:  reason,
		Priority:          EscalationPriorityNormal,
		EscalatedAt:       time.Now(),
		LastMessages:      []EscalationMessage{},
		CollectedEntities: make(map[string]string),
		FlowData:          make(map[string]string),
		Tags:              []string{},
	}
}

// SetSummary sets the AI-generated summary
func (e *EscalationContext) SetSummary(summary string) {
	e.Summary = summary
}

// SetPriority sets the escalation priority
func (e *EscalationContext) SetPriority(priority EscalationPriority) {
	e.Priority = priority
}

// AddMessage adds a message to the context
func (e *EscalationContext) AddMessage(msg EscalationMessage) {
	e.LastMessages = append(e.LastMessages, msg)
	e.MessageCount++
	if msg.IsBot {
		e.BotAttempts++
	}
}

// SetCustomer sets the customer information
func (e *EscalationContext) SetCustomer(customer *EscalationCustomer) {
	e.Customer = customer
}

// AddEntity adds a collected entity
func (e *EscalationContext) AddEntity(key, value string) {
	if e.CollectedEntities == nil {
		e.CollectedEntities = make(map[string]string)
	}
	e.CollectedEntities[key] = value
}

// AddTag adds a tag
func (e *EscalationContext) AddTag(tag string) {
	for _, t := range e.Tags {
		if t == tag {
			return // Already exists
		}
	}
	e.Tags = append(e.Tags, tag)
}

// CalculateWaitTime calculates the wait time since escalation
func (e *EscalationContext) CalculateWaitTime() int64 {
	return int64(time.Since(e.EscalatedAt).Seconds())
}

// GetPriorityFromReason determines priority based on escalation reason
func GetPriorityFromReason(reason EscalationReason) EscalationPriority {
	switch reason {
	case EscalationReasonNegativeSentiment:
		return EscalationPriorityHigh
	case EscalationReasonUserRequest:
		return EscalationPriorityNormal
	case EscalationReasonBotFailure:
		return EscalationPriorityHigh
	case EscalationReasonComplexQuery:
		return EscalationPriorityNormal
	case EscalationReasonLowConfidence:
		return EscalationPriorityLow
	default:
		return EscalationPriorityNormal
	}
}

// EscalationRequest represents a request to escalate a conversation
type EscalationRequest struct {
	ConversationID string              `json:"conversation_id"`
	Reason         EscalationReason    `json:"reason"`
	ReasonDetail   string              `json:"reason_detail,omitempty"`
	Priority       *EscalationPriority `json:"priority,omitempty"`
	AssignToUserID *string             `json:"assign_to_user_id,omitempty"`
	AssignToTeam   *string             `json:"assign_to_team,omitempty"`
	Tags           []string            `json:"tags,omitempty"`
}
