package calling

import (
	"time"
)

// =============================================================================
// Call Status Types
// =============================================================================

// CallStatus represents the status of a call
type CallStatus string

const (
	CallStatusInitiated  CallStatus = "initiated"
	CallStatusRinging    CallStatus = "ringing"
	CallStatusConnected  CallStatus = "connected"
	CallStatusCompleted  CallStatus = "completed"
	CallStatusFailed     CallStatus = "failed"
	CallStatusBusy       CallStatus = "busy"
	CallStatusNoAnswer   CallStatus = "no_answer"
	CallStatusCanceled   CallStatus = "canceled"
	CallStatusRejected   CallStatus = "rejected"
)

// CallDirection represents the direction of a call
type CallDirection string

const (
	CallDirectionInbound  CallDirection = "inbound"
	CallDirectionOutbound CallDirection = "outbound"
)

// CallType represents the type of call
type CallType string

const (
	CallTypeVoice CallType = "voice"
	CallTypeVideo CallType = "video"
)

// =============================================================================
// Call Entity
// =============================================================================

// Call represents a WhatsApp call
type Call struct {
	ID             string        `json:"id"`
	OrganizationID string        `json:"organization_id"`
	ChannelID      string        `json:"channel_id"`
	PhoneNumberID  string        `json:"phone_number_id"`
	From           string        `json:"from"`
	To             string        `json:"to"`
	Direction      CallDirection `json:"direction"`
	Type           CallType      `json:"type"`
	Status         CallStatus    `json:"status"`
	Duration       int           `json:"duration"` // In seconds
	StartedAt      *time.Time    `json:"started_at,omitempty"`
	ConnectedAt    *time.Time    `json:"connected_at,omitempty"`
	EndedAt        *time.Time    `json:"ended_at,omitempty"`
	FailureReason  string        `json:"failure_reason,omitempty"`
	RecordingURL   string        `json:"recording_url,omitempty"`
	Metadata       CallMetadata  `json:"metadata,omitempty"`
	CreatedAt      time.Time     `json:"created_at"`
	UpdatedAt      time.Time     `json:"updated_at"`
}

// CallMetadata contains additional call information
type CallMetadata struct {
	UserAgent     string `json:"user_agent,omitempty"`
	NetworkType   string `json:"network_type,omitempty"`
	QualityScore  int    `json:"quality_score,omitempty"`
	PacketLoss    float64 `json:"packet_loss,omitempty"`
	Jitter        float64 `json:"jitter,omitempty"`
	Latency       int    `json:"latency,omitempty"` // In milliseconds
}

// =============================================================================
// Call Request Types
// =============================================================================

// InitiateCallRequest represents a request to initiate a call
type InitiateCallRequest struct {
	To          string            `json:"to"`
	Type        CallType          `json:"type"`
	CallbackURL string            `json:"callback_url,omitempty"`
	Timeout     int               `json:"timeout,omitempty"` // Ring timeout in seconds
	Metadata    map[string]string `json:"metadata,omitempty"`
}

// CallActionRequest represents a request to perform an action on a call
type CallActionRequest struct {
	Action string `json:"action"` // "end", "hold", "unhold", "mute", "unmute"
}

// =============================================================================
// Call Response Types
// =============================================================================

// InitiateCallResponse represents the response from initiating a call
type InitiateCallResponse struct {
	CallID    string     `json:"call_id"`
	Status    CallStatus `json:"status"`
	CreatedAt time.Time  `json:"created_at"`
}

// CallStatusResponse represents a call status update
type CallStatusResponse struct {
	CallID      string     `json:"call_id"`
	Status      CallStatus `json:"status"`
	Duration    int        `json:"duration,omitempty"`
	ConnectedAt *time.Time `json:"connected_at,omitempty"`
	EndedAt     *time.Time `json:"ended_at,omitempty"`
}

// =============================================================================
// Call Webhook Types
// =============================================================================

// CallWebhookPayload represents a call webhook event
type CallWebhookPayload struct {
	Type          string                 `json:"type"`
	CallID        string                 `json:"call_id"`
	From          string                 `json:"from"`
	To            string                 `json:"to"`
	Direction     CallDirection          `json:"direction"`
	Status        CallStatus             `json:"status"`
	Duration      int                    `json:"duration,omitempty"`
	FailureReason string                 `json:"failure_reason,omitempty"`
	Timestamp     time.Time              `json:"timestamp"`
	Metadata      map[string]interface{} `json:"metadata,omitempty"`
}

// Call webhook type constants
const (
	CallWebhookTypeInitiated  = "call.initiated"
	CallWebhookTypeRinging    = "call.ringing"
	CallWebhookTypeConnected  = "call.connected"
	CallWebhookTypeCompleted  = "call.completed"
	CallWebhookTypeFailed     = "call.failed"
	CallWebhookTypeMissed     = "call.missed"
)

// =============================================================================
// Call Statistics Types
// =============================================================================

// CallStats represents call statistics
type CallStats struct {
	TotalCalls       int                    `json:"total_calls"`
	InboundCalls     int                    `json:"inbound_calls"`
	OutboundCalls    int                    `json:"outbound_calls"`
	CompletedCalls   int                    `json:"completed_calls"`
	MissedCalls      int                    `json:"missed_calls"`
	FailedCalls      int                    `json:"failed_calls"`
	TotalDuration    int                    `json:"total_duration"` // In seconds
	AverageDuration  float64                `json:"average_duration"`
	ByStatus         map[CallStatus]int     `json:"by_status"`
	ByDirection      map[CallDirection]int  `json:"by_direction"`
	DailyStats       []DailyCallStats       `json:"daily_stats,omitempty"`
}

// DailyCallStats represents daily call statistics
type DailyCallStats struct {
	Date            string  `json:"date"`
	TotalCalls      int     `json:"total_calls"`
	InboundCalls    int     `json:"inbound_calls"`
	OutboundCalls   int     `json:"outbound_calls"`
	CompletedCalls  int     `json:"completed_calls"`
	MissedCalls     int     `json:"missed_calls"`
	TotalDuration   int     `json:"total_duration"`
	AverageDuration float64 `json:"average_duration"`
}

// =============================================================================
// Call Event Types
// =============================================================================

// CallEvent represents a call event for pub/sub
type CallEvent struct {
	Type      string    `json:"type"`
	CallID    string    `json:"call_id"`
	Status    string    `json:"status"`
	Timestamp time.Time `json:"timestamp"`
	Data      *Call     `json:"data,omitempty"`
}

// Call event type constants
const (
	CallEventInitiated = "call.initiated"
	CallEventRinging   = "call.ringing"
	CallEventConnected = "call.connected"
	CallEventCompleted = "call.completed"
	CallEventFailed    = "call.failed"
)

// NewCallEvent creates a new call event
func NewCallEvent(eventType string, call *Call) *CallEvent {
	return &CallEvent{
		Type:      eventType,
		CallID:    call.ID,
		Status:    string(call.Status),
		Timestamp: time.Now(),
		Data:      call,
	}
}

// =============================================================================
// Call Quality Types
// =============================================================================

// CallQuality represents call quality metrics
type CallQuality struct {
	CallID       string    `json:"call_id"`
	Score        int       `json:"score"` // 1-5
	PacketLoss   float64   `json:"packet_loss"`
	Jitter       float64   `json:"jitter"`
	Latency      int       `json:"latency"`
	Bitrate      int       `json:"bitrate"`
	MeasuredAt   time.Time `json:"measured_at"`
}

// QualityThresholds defines quality thresholds
type QualityThresholds struct {
	MaxPacketLoss float64 `json:"max_packet_loss"` // Percentage
	MaxJitter     float64 `json:"max_jitter"`      // Milliseconds
	MaxLatency    int     `json:"max_latency"`     // Milliseconds
	MinBitrate    int     `json:"min_bitrate"`     // kbps
}

// DefaultQualityThresholds returns default quality thresholds
func DefaultQualityThresholds() *QualityThresholds {
	return &QualityThresholds{
		MaxPacketLoss: 3.0,   // 3%
		MaxJitter:     30.0,  // 30ms
		MaxLatency:    150,   // 150ms
		MinBitrate:    64,    // 64kbps
	}
}
