// Package outbox contains the relay that drains the transactional outbox and
// publishes its events to NATS. The write side (enqueue in the same tx as the
// aggregate) lives on the owning repository; this side guarantees the event is
// eventually delivered, and — because it republishes with the event's
// idempotency key — a relay retry after a partial failure is deduplicated by
// JetStream, giving effectively exactly-once delivery within the dedup window.
package outbox

import (
	"context"
	"encoding/json"
	"time"

	"github.com/msgfy/linktor/internal/domain/repository"
	"github.com/msgfy/linktor/internal/infrastructure/nats"
	"github.com/msgfy/linktor/pkg/logger"
)

// Relay periodically publishes unpublished outbox events.
type Relay struct {
	repo      repository.OutboxRepository
	publisher nats.Publisher
	batchSize int
}

// NewRelay creates an outbox relay.
func NewRelay(repo repository.OutboxRepository, publisher nats.Publisher) *Relay {
	return &Relay{repo: repo, publisher: publisher, batchSize: 100}
}

// Start runs the relay loop until ctx is cancelled. Run it in its own goroutine.
// A drain error is logged and retried on the next tick.
func (r *Relay) Start(ctx context.Context, interval time.Duration) {
	if interval <= 0 {
		interval = 2 * time.Second
	}
	t := time.NewTicker(interval)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			if err := r.drain(ctx); err != nil {
				logger.Warn("outbox relay drain failed: " + err.Error())
			}
		}
	}
}

// drain publishes one batch of pending events. Exported behavior is idempotent:
// an event published but not yet marked (relay crash) is re-published on the next
// pass and deduplicated by JetStream via its idempotency key.
func (r *Relay) drain(ctx context.Context) error {
	events, err := r.repo.FetchUnpublished(ctx, r.batchSize)
	if err != nil {
		return err
	}
	for _, e := range events {
		var payload map[string]interface{}
		if len(e.Payload) > 0 {
			if err := json.Unmarshal(e.Payload, &payload); err != nil {
				// A malformed payload will never publish; record and skip so it
				// does not wedge the batch. It stays unpublished for inspection.
				_ = r.repo.RecordFailure(ctx, e.ID, "invalid payload: "+err.Error())
				continue
			}
		}
		evt := &nats.Event{
			Type:           e.EventType,
			TenantID:       e.TenantID,
			Payload:        payload,
			Timestamp:      e.CreatedAt,
			IdempotencyKey: e.IdempotencyKey,
		}
		if err := r.publisher.PublishEvent(ctx, evt); err != nil {
			_ = r.repo.RecordFailure(ctx, e.ID, err.Error())
			continue
		}
		if err := r.repo.MarkPublished(ctx, e.ID); err != nil {
			// Published but not marked: the next pass re-publishes; JetStream
			// dedups by idempotency key, so this is safe.
			return err
		}
	}
	return nil
}
