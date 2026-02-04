package entity

import "time"

// AnalyticsPeriod represents the time period for analytics aggregation
type AnalyticsPeriod string

const (
	PeriodDaily   AnalyticsPeriod = "daily"
	PeriodWeekly  AnalyticsPeriod = "weekly"
	PeriodMonthly AnalyticsPeriod = "monthly"
)

// OverviewAnalytics contains high-level metrics for the analytics dashboard
type OverviewAnalytics struct {
	Period    AnalyticsPeriod `json:"period"`
	StartDate time.Time       `json:"start_date"`
	EndDate   time.Time       `json:"end_date"`

	// Conversation Metrics
	TotalConversations int64   `json:"total_conversations"`
	ResolvedByBot      int64   `json:"resolved_by_bot"`
	EscalatedToHuman   int64   `json:"escalated_to_human"`
	ResolutionRate     float64 `json:"resolution_rate"` // percentage

	// Response Times (in milliseconds)
	AvgFirstResponseMs  int64 `json:"avg_first_response_ms"`
	AvgResolutionTimeMs int64 `json:"avg_resolution_time_ms"`

	// Bot Performance
	TotalBotMessages int64   `json:"total_bot_messages"`
	AvgConfidence    float64 `json:"avg_confidence"`

	// Trends (percentage change compared to previous period)
	ConversationsTrend float64 `json:"conversations_trend"`
	ResolutionTrend    float64 `json:"resolution_trend"`
}

// ConversationAnalytics contains daily conversation metrics
type ConversationAnalytics struct {
	Date               string `json:"date"`
	TotalConversations int64  `json:"total_conversations"`
	ResolvedByBot      int64  `json:"resolved_by_bot"`
	Escalated          int64  `json:"escalated"`
	Pending            int64  `json:"pending"`
}

// FlowAnalytics contains metrics for individual flows
type FlowAnalytics struct {
	FlowID         string  `json:"flow_id"`
	FlowName       string  `json:"flow_name"`
	TimesTriggered int64   `json:"times_triggered"`
	TimesCompleted int64   `json:"times_completed"`
	CompletionRate float64 `json:"completion_rate"`
	AvgStepsToEnd  float64 `json:"avg_steps_to_end"`
}

// EscalationAnalytics contains metrics about escalations by reason
type EscalationAnalytics struct {
	Reason         string  `json:"reason"`
	Count          int64   `json:"count"`
	Percentage     float64 `json:"percentage"`
	AvgTimeToEscMs int64   `json:"avg_time_to_escalation_ms"`
}

// ChannelAnalytics contains metrics per channel
type ChannelAnalytics struct {
	ChannelID          string  `json:"channel_id"`
	ChannelName        string  `json:"channel_name"`
	ChannelType        string  `json:"channel_type"`
	TotalConversations int64   `json:"total_conversations"`
	ResolvedByBot      int64   `json:"resolved_by_bot"`
	ResolutionRate     float64 `json:"resolution_rate"`
}

// AnalyticsFilter contains filter parameters for analytics queries
type AnalyticsFilter struct {
	TenantID  string          `json:"tenant_id"`
	Period    AnalyticsPeriod `json:"period"`
	StartDate time.Time       `json:"start_date"`
	EndDate   time.Time       `json:"end_date"`
	BotID     string          `json:"bot_id,omitempty"`
	ChannelID string          `json:"channel_id,omitempty"`
}
