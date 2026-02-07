package entity

import "time"

// LogLevel represents the severity of a log entry
type LogLevel string

const (
	LogLevelInfo  LogLevel = "info"
	LogLevelWarn  LogLevel = "warn"
	LogLevelError LogLevel = "error"
)

// LogSource represents the origin of a log entry
type LogSource string

const (
	LogSourceChannel LogSource = "channel"
	LogSourceQueue   LogSource = "queue"
	LogSourceSystem  LogSource = "system"
	LogSourceWebhook LogSource = "webhook"
)

// MessageLog represents a log entry in the observability system
type MessageLog struct {
	ID          string                 `json:"id"`
	TenantID    string                 `json:"tenant_id"`
	ChannelID   *string                `json:"channel_id,omitempty"`
	ChannelName string                 `json:"channel_name,omitempty"` // Populated via join
	Level       LogLevel               `json:"level"`
	Source      LogSource              `json:"source"`
	Message     string                 `json:"message"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
	CreatedAt   time.Time              `json:"created_at"`
}

// MessageLogFilter contains filter parameters for log queries
type MessageLogFilter struct {
	TenantID  string    `json:"tenant_id"`
	ChannelID string    `json:"channel_id,omitempty"`
	Level     LogLevel  `json:"level,omitempty"`
	Source    LogSource `json:"source,omitempty"`
	StartDate time.Time `json:"start_date"`
	EndDate   time.Time `json:"end_date"`
	Limit     int       `json:"limit"`
	Offset    int       `json:"offset"`
}

// LogsResponse represents a paginated response of logs
type LogsResponse struct {
	Logs    []MessageLog `json:"logs"`
	Total   int64        `json:"total"`
	HasMore bool         `json:"has_more"`
}

// StreamInfo contains information about a NATS JetStream stream
type StreamInfo struct {
	Name        string         `json:"name"`
	Description string         `json:"description,omitempty"`
	Messages    uint64         `json:"messages"`
	Bytes       uint64         `json:"bytes"`
	Consumers   []ConsumerInfo `json:"consumers"`
	FirstSeq    uint64         `json:"first_seq"`
	LastSeq     uint64         `json:"last_seq"`
	Created     time.Time      `json:"created"`
}

// ConsumerInfo contains information about a NATS consumer
type ConsumerInfo struct {
	Name          string    `json:"name"`
	Pending       int       `json:"pending"`
	AckPending    int       `json:"ack_pending"`
	Redelivered   int       `json:"redelivered"`
	LastDelivered time.Time `json:"last_delivered,omitempty"`
}

// QueueStats represents the overall queue statistics
type QueueStats struct {
	Streams       []StreamInfo `json:"streams"`
	TotalMessages uint64       `json:"total_messages"`
	TotalPending  int          `json:"total_pending"`
}

// SystemStats contains system-wide statistics
type SystemStats struct {
	Messages      MessageStats      `json:"messages"`
	ResponseTime  ResponseTimeStats `json:"response_time"`
	Channels      ChannelStats      `json:"channels"`
	Conversations ConversationStats `json:"conversations"`
	Errors        ErrorStats        `json:"errors"`
}

// MessageStats contains message-related statistics
type MessageStats struct {
	Total     int64           `json:"total"`
	PerHour   []HourlyCount   `json:"per_hour"`
	ByChannel []ChannelCount  `json:"by_channel"`
}

// HourlyCount represents message count for a specific hour
type HourlyCount struct {
	Hour  time.Time `json:"hour"`
	Count int64     `json:"count"`
}

// ChannelCount represents message count for a specific channel
type ChannelCount struct {
	ChannelID   string `json:"channel_id"`
	ChannelName string `json:"channel_name"`
	ChannelType string `json:"channel_type"`
	Count       int64  `json:"count"`
}

// ResponseTimeStats contains response time statistics
type ResponseTimeStats struct {
	AvgMs int64 `json:"avg_ms"`
	P95Ms int64 `json:"p95_ms"`
	P99Ms int64 `json:"p99_ms"`
}

// ChannelStats contains channel-related statistics
type ChannelStats struct {
	Total        int `json:"total"`
	Connected    int `json:"connected"`
	Disconnected int `json:"disconnected"`
}

// ConversationStats contains conversation-related statistics
type ConversationStats struct {
	Active        int64 `json:"active"`
	ResolvedToday int64 `json:"resolved_today"`
	Pending       int64 `json:"pending"`
}

// ErrorStats contains error-related statistics
type ErrorStats struct {
	LastHour  int64 `json:"last_hour"`
	Last24h   int64 `json:"last_24h"`
	BySource  []SourceErrorCount `json:"by_source"`
}

// SourceErrorCount represents error count by source
type SourceErrorCount struct {
	Source LogSource `json:"source"`
	Count  int64     `json:"count"`
}

// StatsPeriod represents the time period for statistics
type StatsPeriod string

const (
	StatsPeriodHour  StatsPeriod = "hour"
	StatsPeriodDay   StatsPeriod = "day"
	StatsPeriodWeek  StatsPeriod = "week"
)

// StatsFilter contains filter parameters for statistics queries
type StatsFilter struct {
	TenantID string      `json:"tenant_id"`
	Period   StatsPeriod `json:"period"`
}

// ResetConsumerRequest represents a request to reset a consumer
type ResetConsumerRequest struct {
	Stream   string `json:"stream" binding:"required"`
	Consumer string `json:"consumer" binding:"required"`
}
