package nats

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/nats-io/nats.go/jetstream"
)

// Producer publishes messages to NATS JetStream
type Producer struct {
	client *Client
}

// NewProducer creates a new message producer
func NewProducer(client *Client) *Producer {
	return &Producer{client: client}
}

// InboundMessage represents a message received from an external channel
type InboundMessage struct {
	ID             string            `json:"id"`
	TenantID       string            `json:"tenant_id"`
	ChannelID      string            `json:"channel_id"`
	ChannelType    string            `json:"channel_type"`
	ContactID      string            `json:"contact_id,omitempty"`
	ConversationID string            `json:"conversation_id,omitempty"`
	ExternalID     string            `json:"external_id"`
	ContentType    string            `json:"content_type"`
	Content        string            `json:"content"`
	Metadata       map[string]string `json:"metadata,omitempty"`
	Attachments    []AttachmentData  `json:"attachments,omitempty"`
	Timestamp      time.Time         `json:"timestamp"`
}

// OutboundMessage represents a message to be sent to an external channel
type OutboundMessage struct {
	ID             string            `json:"id"`
	TenantID       string            `json:"tenant_id"`
	ChannelID      string            `json:"channel_id"`
	ChannelType    string            `json:"channel_type"`
	ConversationID string            `json:"conversation_id"`
	ContactID      string            `json:"contact_id"`
	RecipientID    string            `json:"recipient_id"` // External identifier (phone, email, etc.)
	ContentType    string            `json:"content_type"`
	Content        string            `json:"content"`
	Metadata       map[string]string `json:"metadata,omitempty"`
	Attachments    []AttachmentData  `json:"attachments,omitempty"`
	Timestamp      time.Time         `json:"timestamp"`
}

// StatusUpdate represents a message status update
type StatusUpdate struct {
	MessageID    string    `json:"message_id"`
	ExternalID   string    `json:"external_id,omitempty"`
	ChannelType  string    `json:"channel_type"`
	Status       string    `json:"status"`
	ErrorMessage string    `json:"error_message,omitempty"`
	Timestamp    time.Time `json:"timestamp"`
}

// AttachmentData represents attachment information
type AttachmentData struct {
	Type         string            `json:"type"`
	URL          string            `json:"url"`
	Filename     string            `json:"filename,omitempty"`
	MimeType     string            `json:"mime_type,omitempty"`
	SizeBytes    int64             `json:"size_bytes,omitempty"`
	ThumbnailURL string            `json:"thumbnail_url,omitempty"`
	Metadata     map[string]string `json:"metadata,omitempty"`
}

// Event represents a system event
type Event struct {
	Type      string                 `json:"type"`
	TenantID  string                 `json:"tenant_id"`
	Payload   map[string]interface{} `json:"payload"`
	Timestamp time.Time              `json:"timestamp"`
}

// WebhookDelivery represents a webhook to be delivered
type WebhookDelivery struct {
	ID          string                 `json:"id"`
	TenantID    string                 `json:"tenant_id"`
	URL         string                 `json:"url"`
	EventType   string                 `json:"event_type"`
	Payload     map[string]interface{} `json:"payload"`
	Headers     map[string]string      `json:"headers,omitempty"`
	RetryCount  int                    `json:"retry_count"`
	MaxRetries  int                    `json:"max_retries"`
	Timestamp   time.Time              `json:"timestamp"`
}

// PublishInbound publishes an inbound message to the stream
func (p *Producer) PublishInbound(ctx context.Context, msg *InboundMessage) error {
	data, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("failed to marshal inbound message: %w", err)
	}

	subject := SubjectInbound(msg.ChannelType)
	_, err = p.client.js.Publish(ctx, subject, data,
		jetstream.WithMsgID(msg.ID),
	)
	if err != nil {
		return fmt.Errorf("failed to publish inbound message: %w", err)
	}

	return nil
}

// PublishOutbound publishes an outbound message to the stream
func (p *Producer) PublishOutbound(ctx context.Context, msg *OutboundMessage) error {
	data, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("failed to marshal outbound message: %w", err)
	}

	subject := SubjectOutbound(msg.ChannelType)
	_, err = p.client.js.Publish(ctx, subject, data,
		jetstream.WithMsgID(msg.ID),
	)
	if err != nil {
		return fmt.Errorf("failed to publish outbound message: %w", err)
	}

	return nil
}

// PublishStatusUpdate publishes a message status update
func (p *Producer) PublishStatusUpdate(ctx context.Context, status *StatusUpdate) error {
	data, err := json.Marshal(status)
	if err != nil {
		return fmt.Errorf("failed to marshal status update: %w", err)
	}

	subject := SubjectStatus(status.ChannelType)
	msgID := fmt.Sprintf("%s-%s-%d", status.MessageID, status.Status, status.Timestamp.UnixNano())
	_, err = p.client.js.Publish(ctx, subject, data,
		jetstream.WithMsgID(msgID),
	)
	if err != nil {
		return fmt.Errorf("failed to publish status update: %w", err)
	}

	return nil
}

// PublishEvent publishes a system event
func (p *Producer) PublishEvent(ctx context.Context, event *Event) error {
	data, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal event: %w", err)
	}

	subject := SubjectEvent(event.Type)
	msgID := fmt.Sprintf("%s-%s-%d", event.TenantID, event.Type, event.Timestamp.UnixNano())
	_, err = p.client.js.Publish(ctx, subject, data,
		jetstream.WithMsgID(msgID),
	)
	if err != nil {
		return fmt.Errorf("failed to publish event: %w", err)
	}

	return nil
}

// PublishWebhookDelivery publishes a webhook delivery request
func (p *Producer) PublishWebhookDelivery(ctx context.Context, webhook *WebhookDelivery) error {
	data, err := json.Marshal(webhook)
	if err != nil {
		return fmt.Errorf("failed to marshal webhook delivery: %w", err)
	}

	subject := SubjectWebhook(webhook.TenantID)
	_, err = p.client.js.Publish(ctx, subject, data,
		jetstream.WithMsgID(webhook.ID),
	)
	if err != nil {
		return fmt.Errorf("failed to publish webhook delivery: %w", err)
	}

	return nil
}
