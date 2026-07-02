package nats

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/msgfy/linktor/pkg/logger"
	"github.com/nats-io/nats.go/jetstream"
	"go.uber.org/zap"
)

// DLQMessage describes a dead-lettered message observed on the DLQ stream.
type DLQMessage struct {
	// Consumer is the name of the consumer that exhausted its retries.
	Consumer string
	Subject  string
	Data     []byte
	Time     time.Time
}

// DLQHandler handles dead-lettered messages. Returning an error NAKs the
// message so it is observed again later.
type DLQHandler func(ctx context.Context, msg *DLQMessage) error

// DLQConsumer observes the dead-letter stream. The DLQ stream uses
// LimitsPolicy, so observed messages remain on the stream for manual
// inspection and replay; this consumer only surfaces them.
type DLQConsumer struct {
	client *Client
}

// NewDLQConsumer creates a new DLQ observer.
func NewDLQConsumer(client *Client) *DLQConsumer {
	return &DLQConsumer{client: client}
}

// Subscribe starts a durable observer on the DLQ stream. Pass nil to use the
// default handler, which logs each dead-lettered message at error level so
// log-based alerting can pick it up.
func (c *DLQConsumer) Subscribe(ctx context.Context, handler DLQHandler) error {
	if handler == nil {
		handler = logDLQMessage
	}

	stream, err := c.client.js.Stream(ctx, StreamDLQ)
	if err != nil {
		return fmt.Errorf("failed to get stream %s: %w", StreamDLQ, err)
	}

	consumerCfg := jetstream.ConsumerConfig{
		Name:          "dlq-observer",
		Durable:       "dlq-observer",
		FilterSubject: SubjectDLQAll,
		AckPolicy:     jetstream.AckExplicitPolicy,
		MaxDeliver:    5,
		AckWait:       30 * time.Second,
		MaxAckPending: 100,
		DeliverPolicy: jetstream.DeliverAllPolicy,
	}

	consumer, err := stream.CreateOrUpdateConsumer(ctx, consumerCfg)
	if err != nil {
		return fmt.Errorf("failed to create consumer dlq-observer: %w", err)
	}

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
					dlqMsg := &DLQMessage{
						Consumer: strings.TrimPrefix(msg.Subject(), strings.TrimSuffix(SubjectDLQAll, ">")),
						Subject:  msg.Subject(),
						Data:     msg.Data(),
						Time:     time.Now(),
					}
					if meta, metaErr := msg.Metadata(); metaErr == nil {
						dlqMsg.Time = meta.Timestamp
					}

					if err := handler(ctx, dlqMsg); err != nil {
						msg.NakWithDelay(30 * time.Second)
					} else {
						msg.Ack()
					}
				}
			}
		}
	}()

	return nil
}

// logDLQMessage is the default DLQ handler: it surfaces dead-lettered
// messages in the logs at error level for alerting and triage.
func logDLQMessage(_ context.Context, msg *DLQMessage) error {
	const maxPreview = 512
	preview := string(msg.Data)
	if len(preview) > maxPreview {
		preview = preview[:maxPreview] + "…"
	}
	logger.Error("dead-lettered message requires attention",
		zap.String("consumer", msg.Consumer),
		zap.String("subject", msg.Subject),
		zap.Time("dead_lettered_at", msg.Time),
		zap.Int("size_bytes", len(msg.Data)),
		zap.String("payload_preview", preview),
	)
	return nil
}
