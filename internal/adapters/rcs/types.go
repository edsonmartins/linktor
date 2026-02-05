package rcs

import (
	"errors"
	"time"
)

// Errors
var (
	ErrInvalidProvider     = errors.New("invalid RCS provider")
	ErrMissingCredentials  = errors.New("missing RCS credentials")
	ErrProviderUnavailable = errors.New("RCS provider unavailable")
	ErrMessageTooLong      = errors.New("message exceeds maximum length")
	ErrUnsupportedMedia    = errors.New("unsupported media type")
)

// Provider represents an RCS service provider
type Provider string

const (
	ProviderZenvia    Provider = "zenvia"
	ProviderInfobip   Provider = "infobip"
	ProviderPontaltech Provider = "pontaltech"
	ProviderGoogle    Provider = "google"
)

// Config holds the configuration for RCS adapter
type Config struct {
	// Provider is the RCS service provider
	Provider Provider `json:"provider"`

	// AgentID is the RCS Business Messaging agent ID
	AgentID string `json:"agent_id"`

	// APIKey is the API key for authentication
	APIKey string `json:"api_key"`

	// APISecret is the API secret (if required)
	APISecret string `json:"api_secret,omitempty"`

	// WebhookURL is the URL for receiving delivery reports
	WebhookURL string `json:"webhook_url,omitempty"`

	// WebhookSecret is the secret for webhook signature validation
	WebhookSecret string `json:"webhook_secret,omitempty"`

	// BaseURL is the provider's API base URL (optional, for custom endpoints)
	BaseURL string `json:"base_url,omitempty"`

	// SenderID is the sender identifier (phone number or brand name)
	SenderID string `json:"sender_id,omitempty"`
}

// Validate validates the configuration
func (c *Config) Validate() error {
	if c.Provider == "" {
		return errors.New("provider is required")
	}
	if c.AgentID == "" {
		return errors.New("agent_id is required")
	}
	if c.APIKey == "" {
		return errors.New("api_key is required")
	}
	return nil
}

// GetBaseURL returns the base URL for the provider
func (c *Config) GetBaseURL() string {
	if c.BaseURL != "" {
		return c.BaseURL
	}

	switch c.Provider {
	case ProviderZenvia:
		return "https://api.zenvia.com/v2"
	case ProviderInfobip:
		return "https://api.infobip.com"
	case ProviderPontaltech:
		return "https://api.pontaltech.com.br/v1"
	case ProviderGoogle:
		return "https://rcsbusinessmessaging.googleapis.com/v1"
	default:
		return ""
	}
}

// IncomingMessage represents an incoming RCS message
type IncomingMessage struct {
	ExternalID  string       `json:"external_id"`
	SenderPhone string       `json:"sender_phone"`
	Text        string       `json:"text,omitempty"`
	MediaURL    string       `json:"media_url,omitempty"`
	MediaType   string       `json:"media_type,omitempty"`
	Location    *Location    `json:"location,omitempty"`
	Suggestion  *Suggestion  `json:"suggestion,omitempty"`
	Timestamp   time.Time    `json:"timestamp"`
	AgentID     string       `json:"agent_id"`
	RawPayload  interface{}  `json:"raw_payload,omitempty"`
}

// OutboundMessage represents an outbound RCS message
type OutboundMessage struct {
	To          string            `json:"to"`
	Text        string            `json:"text,omitempty"`
	MediaURL    string            `json:"media_url,omitempty"`
	MediaType   string            `json:"media_type,omitempty"`
	Card        *RichCard         `json:"card,omitempty"`
	Carousel    *Carousel         `json:"carousel,omitempty"`
	Suggestions []Suggestion      `json:"suggestions,omitempty"`
	Metadata    map[string]string `json:"metadata,omitempty"`
}

// SendResult represents the result of sending a message
type SendResult struct {
	Success    bool      `json:"success"`
	MessageID  string    `json:"message_id,omitempty"`
	Error      string    `json:"error,omitempty"`
	Timestamp  time.Time `json:"timestamp"`
}

// RichCard represents an RCS rich card
type RichCard struct {
	Title       string       `json:"title,omitempty"`
	Description string       `json:"description,omitempty"`
	MediaURL    string       `json:"media_url,omitempty"`
	MediaType   string       `json:"media_type,omitempty"`
	Suggestions []Suggestion `json:"suggestions,omitempty"`
}

// Carousel represents an RCS carousel
type Carousel struct {
	Width string     `json:"width,omitempty"` // "SMALL", "MEDIUM"
	Cards []RichCard `json:"cards"`
}

// Suggestion represents an RCS suggestion (action/reply)
type Suggestion struct {
	Type        SuggestionType `json:"type"`
	Text        string         `json:"text"`
	PostbackData string        `json:"postback_data,omitempty"`
	// For dial action
	PhoneNumber string `json:"phone_number,omitempty"`
	// For URL action
	URL string `json:"url,omitempty"`
	// For location action
	Location *Location `json:"location,omitempty"`
	// For calendar action
	CalendarEvent *CalendarEvent `json:"calendar_event,omitempty"`
}

// SuggestionType represents the type of suggestion
type SuggestionType string

const (
	SuggestionTypeReply       SuggestionType = "reply"
	SuggestionTypeAction      SuggestionType = "action"
	SuggestionTypeDial        SuggestionType = "dial"
	SuggestionTypeURL         SuggestionType = "url"
	SuggestionTypeLocation    SuggestionType = "location"
	SuggestionTypeCalendar    SuggestionType = "calendar"
	SuggestionTypeShare       SuggestionType = "share"
)

// Location represents a geographic location
type Location struct {
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
	Label     string  `json:"label,omitempty"`
	Query     string  `json:"query,omitempty"`
}

// CalendarEvent represents a calendar event suggestion
type CalendarEvent struct {
	Title       string    `json:"title"`
	Description string    `json:"description,omitempty"`
	StartTime   time.Time `json:"start_time"`
	EndTime     time.Time `json:"end_time"`
}

// DeliveryReport represents an RCS delivery report
type DeliveryReport struct {
	MessageID string        `json:"message_id"`
	Status    DeliveryStatus `json:"status"`
	Timestamp time.Time     `json:"timestamp"`
	Error     string        `json:"error,omitempty"`
}

// DeliveryStatus represents the delivery status
type DeliveryStatus string

const (
	StatusPending   DeliveryStatus = "pending"
	StatusSent      DeliveryStatus = "sent"
	StatusDelivered DeliveryStatus = "delivered"
	StatusRead      DeliveryStatus = "read"
	StatusFailed    DeliveryStatus = "failed"
)

// WebhookPayload represents an incoming webhook payload
type WebhookPayload struct {
	Provider    Provider          `json:"provider"`
	Type        string            `json:"type"` // "message", "status", "event"
	Message     *IncomingMessage  `json:"message,omitempty"`
	Status      *DeliveryReport   `json:"status,omitempty"`
	RawPayload  interface{}       `json:"raw_payload,omitempty"`
}

// ProviderClient is the interface for RCS provider clients
type ProviderClient interface {
	// SendMessage sends a message
	SendMessage(msg *OutboundMessage) (*SendResult, error)

	// ParseWebhook parses an incoming webhook
	ParseWebhook(body []byte) (*WebhookPayload, error)

	// ValidateWebhook validates a webhook signature
	ValidateWebhook(signature string, body []byte) bool

	// GetAgentInfo retrieves agent information
	GetAgentInfo() (map[string]interface{}, error)
}
