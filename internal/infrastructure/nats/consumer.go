package nats

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/nats-io/nats.go/jetstream"
)

// MessageHandler is a function that handles incoming messages
type MessageHandler func(ctx context.Context, msg *InboundMessage) error

// OutboundHandler is a function that handles outbound messages
type OutboundHandler func(ctx context.Context, msg *OutboundMessage) error

// StatusHandler is a function that handles status updates
type StatusHandler func(ctx context.Context, status *StatusUpdate) error

// EventHandler is a function that handles events
type EventHandler func(ctx context.Context, event *Event) error

// WebhookHandler is a function that handles webhook deliveries
type WebhookHandler func(ctx context.Context, webhook *WebhookDelivery) error

// Consumer consumes messages from NATS JetStream
type Consumer struct {
	client     *Client
	consumers  []jetstream.Consumer
	cancelFunc context.CancelFunc
}

// NewConsumer creates a new message consumer
func NewConsumer(client *Client) *Consumer {
	return &Consumer{
		client:    client,
		consumers: make([]jetstream.Consumer, 0),
	}
}

// ConsumerConfig holds configuration for a consumer
type ConsumerConfig struct {
	Stream       string
	Name         string
	FilterSubject string
	MaxDeliver   int
	AckWait      time.Duration
	MaxAckPending int
}

// SubscribeInbound subscribes to inbound messages for a specific channel type
func (c *Consumer) SubscribeInbound(ctx context.Context, channelType string, handler MessageHandler) error {
	cfg := ConsumerConfig{
		Stream:        StreamMessages,
		Name:          fmt.Sprintf("%s-%s", ConsumerInbound, channelType),
		FilterSubject: SubjectInbound(channelType),
		MaxDeliver:    5,
		AckWait:       30 * time.Second,
		MaxAckPending: 100,
	}

	return c.subscribe(ctx, cfg, func(msg jetstream.Msg) error {
		var inbound InboundMessage
		if err := json.Unmarshal(msg.Data(), &inbound); err != nil {
			return fmt.Errorf("failed to unmarshal inbound message: %w", err)
		}
		return handler(ctx, &inbound)
	})
}

// SubscribeAllInbound subscribes to all inbound messages
func (c *Consumer) SubscribeAllInbound(ctx context.Context, handler MessageHandler) error {
	cfg := ConsumerConfig{
		Stream:        StreamMessages,
		Name:          ConsumerInbound,
		FilterSubject: SubjectInboundAll,
		MaxDeliver:    5,
		AckWait:       30 * time.Second,
		MaxAckPending: 100,
	}

	return c.subscribe(ctx, cfg, func(msg jetstream.Msg) error {
		var inbound InboundMessage
		if err := json.Unmarshal(msg.Data(), &inbound); err != nil {
			return fmt.Errorf("failed to unmarshal inbound message: %w", err)
		}
		return handler(ctx, &inbound)
	})
}

// SubscribeOutbound subscribes to outbound messages for a specific channel type
func (c *Consumer) SubscribeOutbound(ctx context.Context, channelType string, handler OutboundHandler) error {
	cfg := ConsumerConfig{
		Stream:        StreamMessages,
		Name:          ConsumerOutbound(channelType),
		FilterSubject: SubjectOutbound(channelType),
		MaxDeliver:    5,
		AckWait:       60 * time.Second,
		MaxAckPending: 50,
	}

	return c.subscribe(ctx, cfg, func(msg jetstream.Msg) error {
		var outbound OutboundMessage
		if err := json.Unmarshal(msg.Data(), &outbound); err != nil {
			return fmt.Errorf("failed to unmarshal outbound message: %w", err)
		}
		return handler(ctx, &outbound)
	})
}

// SubscribeStatus subscribes to message status updates
func (c *Consumer) SubscribeStatus(ctx context.Context, handler StatusHandler) error {
	cfg := ConsumerConfig{
		Stream:        StreamMessages,
		Name:          ConsumerStatus,
		FilterSubject: SubjectStatusAll,
		MaxDeliver:    3,
		AckWait:       10 * time.Second,
		MaxAckPending: 200,
	}

	return c.subscribe(ctx, cfg, func(msg jetstream.Msg) error {
		var status StatusUpdate
		if err := json.Unmarshal(msg.Data(), &status); err != nil {
			return fmt.Errorf("failed to unmarshal status update: %w", err)
		}
		return handler(ctx, &status)
	})
}

// SubscribeEvents subscribes to system events
func (c *Consumer) SubscribeEvents(ctx context.Context, handler EventHandler) error {
	cfg := ConsumerConfig{
		Stream:        StreamEvents,
		Name:          ConsumerEvents,
		FilterSubject: SubjectEventsAll,
		MaxDeliver:    3,
		AckWait:       10 * time.Second,
		MaxAckPending: 200,
	}

	return c.subscribe(ctx, cfg, func(msg jetstream.Msg) error {
		var event Event
		if err := json.Unmarshal(msg.Data(), &event); err != nil {
			return fmt.Errorf("failed to unmarshal event: %w", err)
		}
		return handler(ctx, &event)
	})
}

// SubscribeWebhooks subscribes to webhook delivery requests
func (c *Consumer) SubscribeWebhooks(ctx context.Context, handler WebhookHandler) error {
	cfg := ConsumerConfig{
		Stream:        StreamWebhooks,
		Name:          ConsumerWebhooks,
		FilterSubject: SubjectWebhooksAll,
		MaxDeliver:    10,
		AckWait:       30 * time.Second,
		MaxAckPending: 50,
	}

	return c.subscribe(ctx, cfg, func(msg jetstream.Msg) error {
		var webhook WebhookDelivery
		if err := json.Unmarshal(msg.Data(), &webhook); err != nil {
			return fmt.Errorf("failed to unmarshal webhook delivery: %w", err)
		}
		return handler(ctx, &webhook)
	})
}

// subscribe creates a consumer and starts consuming messages
func (c *Consumer) subscribe(ctx context.Context, cfg ConsumerConfig, handler func(jetstream.Msg) error) error {
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
func (c *Consumer) Stop() {
	if c.cancelFunc != nil {
		c.cancelFunc()
	}
}
