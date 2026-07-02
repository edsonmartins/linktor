package outbox

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/msgfy/linktor/internal/domain/entity"
	"github.com/msgfy/linktor/pkg/testutil"
)

// fakeOutboxRepo is an in-memory repository.OutboxRepository for relay tests.
type fakeOutboxRepo struct {
	events    []*entity.OutboxEvent
	failFetch error
}

func (r *fakeOutboxRepo) FetchUnpublished(ctx context.Context, limit int) ([]*entity.OutboxEvent, error) {
	if r.failFetch != nil {
		return nil, r.failFetch
	}
	var out []*entity.OutboxEvent
	for _, e := range r.events {
		if e.PublishedAt == nil {
			out = append(out, e)
		}
	}
	return out, nil
}

func (r *fakeOutboxRepo) MarkPublished(ctx context.Context, id string) error {
	for _, e := range r.events {
		if e.ID == id {
			now := time.Unix(1, 0)
			e.PublishedAt = &now
		}
	}
	return nil
}

func (r *fakeOutboxRepo) RecordFailure(ctx context.Context, id, lastErr string) error {
	for _, e := range r.events {
		if e.ID == id {
			e.Attempts++
			e.LastError = lastErr
		}
	}
	return nil
}

func newEvent(id string) *entity.OutboxEvent {
	return &entity.OutboxEvent{
		ID:             id,
		EventType:      "message.received",
		TenantID:       "tenant-1",
		Payload:        []byte(`{"message_id":"m1"}`),
		IdempotencyKey: "evt-" + id,
		CreatedAt:      time.Unix(0, 0),
	}
}

func TestRelay_DrainPublishesAndMarks(t *testing.T) {
	repo := &fakeOutboxRepo{events: []*entity.OutboxEvent{newEvent("a"), newEvent("b")}}
	producer := testutil.NewMockProducer()
	relay := NewRelay(repo, producer)

	require.NoError(t, relay.drain(context.Background()))

	// Both events published with their payload and idempotency key preserved.
	require.Len(t, producer.Events, 2)
	assert.Equal(t, "message.received", producer.Events[0].Type)
	assert.Equal(t, "evt-a", producer.Events[0].IdempotencyKey)
	assert.Equal(t, "m1", producer.Events[0].Payload["message_id"])

	// Both marked published → a second drain does nothing.
	for _, e := range repo.events {
		assert.NotNil(t, e.PublishedAt)
	}
	producer.Events = nil
	require.NoError(t, relay.drain(context.Background()))
	assert.Len(t, producer.Events, 0, "published events are not re-published")
}

func TestRelay_PublishFailureLeavesEventUnpublished(t *testing.T) {
	repo := &fakeOutboxRepo{events: []*entity.OutboxEvent{newEvent("a")}}
	producer := testutil.NewMockProducer()
	producer.ReturnError = fmt.Errorf("nats down")
	relay := NewRelay(repo, producer)

	require.NoError(t, relay.drain(context.Background()), "a publish failure is not fatal to the batch")

	// Still unpublished and retried later; failure recorded.
	assert.Nil(t, repo.events[0].PublishedAt)
	assert.Equal(t, 1, repo.events[0].Attempts)
	assert.Contains(t, repo.events[0].LastError, "nats down")

	// Recovery: next drain publishes it.
	producer.ReturnError = nil
	require.NoError(t, relay.drain(context.Background()))
	require.Len(t, producer.Events, 1)
	assert.NotNil(t, repo.events[0].PublishedAt)
}

func TestRelay_InvalidPayloadIsSkippedNotWedged(t *testing.T) {
	bad := newEvent("bad")
	bad.Payload = []byte(`{not json`)
	good := newEvent("good")
	repo := &fakeOutboxRepo{events: []*entity.OutboxEvent{bad, good}}
	producer := testutil.NewMockProducer()
	relay := NewRelay(repo, producer)

	require.NoError(t, relay.drain(context.Background()))

	// The good event still publishes; the bad one is recorded and left behind.
	require.Len(t, producer.Events, 1)
	assert.Equal(t, "evt-good", producer.Events[0].IdempotencyKey)
	assert.Nil(t, bad.PublishedAt)
	assert.Equal(t, 1, bad.Attempts)
}
