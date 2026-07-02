package webhook

import (
	"context"
	"time"

	"github.com/msgfy/linktor/internal/infrastructure/nats"
	"github.com/msgfy/linktor/pkg/errors"
	"github.com/msgfy/linktor/pkg/logger"
)

// DeliveryWorker is the consumer side of the durable outbound-webhook pipeline.
// It reads WebhookDelivery items off the webhooks stream, re-resolves the target
// channel to obtain the current endpoint URL and signing secret, signs the
// carried body with a FRESH timestamp (so redeliveries stay within the
// `linktor-channel-v1` anti-replay window) and performs a single HTTP attempt.
//
// Returning an error makes the NATS consumer redeliver (and eventually
// dead-letter) the item — that is where durable retry lives. The HTTP producer
// therefore makes only one attempt per invocation; NATS owns backoff and DLQ.
type DeliveryWorker struct {
	producer *WebhookProducer
	channels ChannelLookup
	now      func() time.Time
}

// NewDeliveryWorker builds the webhook delivery worker.
func NewDeliveryWorker(producer *WebhookProducer, channels ChannelLookup) *DeliveryWorker {
	return &DeliveryWorker{
		producer: producer,
		channels: channels,
		now:      time.Now,
	}
}

// Handle implements nats.WebhookHandler.
func (w *DeliveryWorker) Handle(ctx context.Context, wh *nats.WebhookDelivery) error {
	if wh == nil || len(wh.Body) == 0 {
		return nil // nothing to deliver → ack
	}

	channel, err := w.channels.FindByID(ctx, wh.ChannelID)
	if err != nil {
		if errors.IsNotFound(err) {
			// The channel was deleted while deliveries were still queued. Retrying
			// can never succeed and would flood the DLQ, so drop (ack) this item.
			logger.Warn("webhook: dropping delivery for deleted channel " + wh.ChannelID)
			return nil
		}
		// Transient lookup failure (e.g. DB blip): return the error to retry.
		return err
	}
	if channel == nil || channel.WebhookURL == "" {
		// No endpoint configured anymore → drop (ack); nothing to deliver to.
		return nil
	}

	secret := channel.Credentials[credWebhookSecret]
	if secret == "" {
		// Signing with an empty key produces a forgeable signature. Fail closed:
		// drop (ack) rather than emit a delivery anyone could forge. A config error
		// won't fix on retry, so we don't redeliver/DLQ it.
		logger.Warn("webhook: channel " + wh.ChannelID + " has no webhook_secret; refusing to send a forgeable (unsigned-key) delivery")
		return nil
	}
	endpoint := EndpointConfig{
		URL:        channel.WebhookURL,
		Headers:    SignatureHeaders(wh.Body, secret, w.now()),
		MaxRetries: 1, // single attempt; NATS handles durable retry + DLQ
	}

	if _, err := w.producer.Deliver(ctx, endpoint, wh.EventType, wh.Body); err != nil {
		logger.Warn("webhook: delivery to " + endpoint.URL + " failed (will retry): " + err.Error())
		return err
	}
	return nil
}
