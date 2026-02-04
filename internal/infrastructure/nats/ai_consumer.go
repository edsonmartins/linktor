package nats

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/nats-io/nats.go/jetstream"
)

// BotAnalysisRequest represents a request for bot analysis
type BotAnalysisRequest struct {
	MessageID      string            `json:"message_id"`
	ConversationID string            `json:"conversation_id"`
	TenantID       string            `json:"tenant_id"`
	ChannelID      string            `json:"channel_id"`
	Content        string            `json:"content"`
	ContentType    string            `json:"content_type"`
	Metadata       map[string]string `json:"metadata,omitempty"`
	Timestamp      time.Time         `json:"timestamp"`
}

// BotResponseRequest represents a request for bot response generation
type BotResponseRequest struct {
	MessageID      string            `json:"message_id"`
	ConversationID string            `json:"conversation_id"`
	TenantID       string            `json:"tenant_id"`
	ChannelID      string            `json:"channel_id"`
	ContactID      string            `json:"contact_id"`
	BotID          string            `json:"bot_id"`
	Content        string            `json:"content"`
	Metadata       map[string]string `json:"metadata,omitempty"`
	Timestamp      time.Time         `json:"timestamp"`
}

// BotEscalationRequest represents a request to escalate to human agent
type BotEscalationRequest struct {
	ConversationID string    `json:"conversation_id"`
	TenantID       string    `json:"tenant_id"`
	ChannelID      string    `json:"channel_id"`
	ContactID      string    `json:"contact_id"`
	BotID          string    `json:"bot_id"`
	Reason         string    `json:"reason"`
	Priority       string    `json:"priority"`
	Timestamp      time.Time `json:"timestamp"`
}

// BotAnalysisHandler handles bot analysis requests
type BotAnalysisHandler func(ctx context.Context, req *BotAnalysisRequest) error

// BotResponseHandler handles bot response requests
type BotResponseHandler func(ctx context.Context, req *BotResponseRequest) error

// BotEscalationHandler handles bot escalation requests
type BotEscalationHandler func(ctx context.Context, req *BotEscalationRequest) error

// AIConsumer consumes AI-related messages from NATS JetStream
type AIConsumer struct {
	client     *Client
	consumers  []jetstream.Consumer
	cancelFunc context.CancelFunc
}

// NewAIConsumer creates a new AI consumer
func NewAIConsumer(client *Client) *AIConsumer {
	return &AIConsumer{
		client:    client,
		consumers: make([]jetstream.Consumer, 0),
	}
}

// EnsureStream ensures the AI stream exists
func (c *AIConsumer) EnsureStream(ctx context.Context) error {
	streamCfg := jetstream.StreamConfig{
		Name:        StreamAI,
		Description: "Linktor AI/Bot processing stream",
		Subjects: []string{
			"linktor.bot.>",
		},
		Retention:    jetstream.WorkQueuePolicy,
		MaxConsumers: -1,
		MaxMsgs:      -1,
		MaxBytes:     -1,
		MaxAge:       24 * time.Hour,
		Storage:      jetstream.FileStorage,
		Replicas:     1,
		Duplicates:   5 * time.Minute,
	}

	_, err := c.client.js.CreateOrUpdateStream(ctx, streamCfg)
	if err != nil {
		return fmt.Errorf("failed to create AI stream: %w", err)
	}

	return nil
}

// SubscribeBotAnalysis subscribes to bot analysis requests
func (c *AIConsumer) SubscribeBotAnalysis(ctx context.Context, handler BotAnalysisHandler) error {
	cfg := ConsumerConfig{
		Stream:        StreamAI,
		Name:          ConsumerAIAnalyzer,
		FilterSubject: SubjectBotAnalyzeAll,
		MaxDeliver:    3,
		AckWait:       30 * time.Second,
		MaxAckPending: 100,
	}

	return c.subscribe(ctx, cfg, func(msg jetstream.Msg) error {
		var req BotAnalysisRequest
		if err := json.Unmarshal(msg.Data(), &req); err != nil {
			return fmt.Errorf("failed to unmarshal bot analysis request: %w", err)
		}
		return handler(ctx, &req)
	})
}

// SubscribeBotResponse subscribes to bot response requests
func (c *AIConsumer) SubscribeBotResponse(ctx context.Context, handler BotResponseHandler) error {
	cfg := ConsumerConfig{
		Stream:        StreamAI,
		Name:          ConsumerAIResponder,
		FilterSubject: SubjectBotResponseAll,
		MaxDeliver:    3,
		AckWait:       60 * time.Second, // Longer timeout for AI calls
		MaxAckPending: 50,
	}

	return c.subscribe(ctx, cfg, func(msg jetstream.Msg) error {
		var req BotResponseRequest
		if err := json.Unmarshal(msg.Data(), &req); err != nil {
			return fmt.Errorf("failed to unmarshal bot response request: %w", err)
		}
		return handler(ctx, &req)
	})
}

// SubscribeBotEscalation subscribes to bot escalation requests
func (c *AIConsumer) SubscribeBotEscalation(ctx context.Context, handler BotEscalationHandler) error {
	cfg := ConsumerConfig{
		Stream:        StreamAI,
		Name:          ConsumerAIEscalation,
		FilterSubject: SubjectBotEscalateAll,
		MaxDeliver:    5,
		AckWait:       30 * time.Second,
		MaxAckPending: 100,
	}

	return c.subscribe(ctx, cfg, func(msg jetstream.Msg) error {
		var req BotEscalationRequest
		if err := json.Unmarshal(msg.Data(), &req); err != nil {
			return fmt.Errorf("failed to unmarshal bot escalation request: %w", err)
		}
		return handler(ctx, &req)
	})
}

// subscribe creates a consumer and starts consuming messages
func (c *AIConsumer) subscribe(ctx context.Context, cfg ConsumerConfig, handler func(jetstream.Msg) error) error {
	stream, err := c.client.js.Stream(ctx, cfg.Stream)
	if err != nil {
		return fmt.Errorf("failed to get stream %s: %w", cfg.Stream, err)
	}

	consumerCfg := jetstream.ConsumerConfig{
		Name:          cfg.Name,
		Durable:       cfg.Name,
		FilterSubject: cfg.FilterSubject,
		AckPolicy:     jetstream.AckExplicitPolicy,
		MaxDeliver:    cfg.MaxDeliver,
		AckWait:       cfg.AckWait,
		MaxAckPending: cfg.MaxAckPending,
		DeliverPolicy: jetstream.DeliverAllPolicy,
	}

	consumer, err := stream.CreateOrUpdateConsumer(ctx, consumerCfg)
	if err != nil {
		return fmt.Errorf("failed to create consumer %s: %w", cfg.Name, err)
	}

	c.consumers = append(c.consumers, consumer)

	// Start consuming in a goroutine
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			default:
				msgs, err := consumer.Fetch(10, jetstream.FetchMaxWait(5*time.Second))
				if err != nil {
					if err == context.Canceled || err == context.DeadlineExceeded {
						continue
					}
					time.Sleep(1 * time.Second)
					continue
				}

				for msg := range msgs.Messages() {
					if err := handler(msg); err != nil {
						// NAK with delay for retry
						msg.NakWithDelay(5 * time.Second)
					} else {
						msg.Ack()
					}
				}
			}
		}
	}()

	return nil
}

// Stop stops all consumers
func (c *AIConsumer) Stop() {
	if c.cancelFunc != nil {
		c.cancelFunc()
	}
}

// AIProducer extension methods for Producer

// PublishBotAnalysis publishes a bot analysis request
func (p *Producer) PublishBotAnalysis(ctx context.Context, req *BotAnalysisRequest) error {
	data, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("failed to marshal bot analysis request: %w", err)
	}

	subject := SubjectBotAnalyze(req.TenantID)
	_, err = p.client.js.Publish(ctx, subject, data)
	if err != nil {
		return fmt.Errorf("failed to publish bot analysis request: %w", err)
	}

	return nil
}

// PublishBotResponse publishes a bot response request
func (p *Producer) PublishBotResponse(ctx context.Context, req *BotResponseRequest) error {
	data, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("failed to marshal bot response request: %w", err)
	}

	subject := SubjectBotResponse(req.TenantID)
	_, err = p.client.js.Publish(ctx, subject, data)
	if err != nil {
		return fmt.Errorf("failed to publish bot response request: %w", err)
	}

	return nil
}

// PublishBotEscalation publishes a bot escalation request
func (p *Producer) PublishBotEscalation(ctx context.Context, req *BotEscalationRequest) error {
	data, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("failed to marshal bot escalation request: %w", err)
	}

	subject := SubjectBotEscalate(req.TenantID)
	_, err = p.client.js.Publish(ctx, subject, data)
	if err != nil {
		return fmt.Errorf("failed to publish bot escalation request: %w", err)
	}

	return nil
}
