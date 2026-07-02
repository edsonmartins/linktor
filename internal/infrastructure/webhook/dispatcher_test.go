package webhook

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/msgfy/linktor/internal/domain/entity"
	"github.com/msgfy/linktor/internal/infrastructure/nats"
)

type fakeChannels struct {
	channel *entity.Channel
	err     error
}

func (f *fakeChannels) FindByID(_ context.Context, _ string) (*entity.Channel, error) {
	return f.channel, f.err
}

type recordingPublisher struct {
	deliveries []*nats.WebhookDelivery
	err        error
}

func (r *recordingPublisher) PublishWebhookDelivery(_ context.Context, wh *nats.WebhookDelivery) error {
	if r.err != nil {
		return r.err
	}
	r.deliveries = append(r.deliveries, wh)
	return nil
}

func newTestChannel() *entity.Channel {
	return &entity.Channel{
		ID:          "ch-1",
		Type:        "slack",
		WebhookURL:  "https://desklenz.example/webhook",
		Credentials: map[string]string{credWebhookSecret: "s3cr3t"},
	}
}

func TestDispatcherEnqueuesInboundDurably(t *testing.T) {
	pub := &recordingPublisher{}
	d := NewDispatcher(pub, &fakeChannels{channel: newTestChannel()})

	event := &nats.Event{
		Type:     nats.EventMessageReceived,
		TenantID: "tenant-1",
		Payload: map[string]interface{}{
			"channel_id":      "ch-1",
			"message_id":      "m-1",
			"conversation_id": "c-1",
			"contact_id":      "ct-1",
			"content_type":    "text",
			"content":         "hello",
			"sender_id":       "U123",
			"sender_name":     "Ada",
		},
	}

	if err := d.handle(context.Background(), event); err != nil {
		t.Fatalf("handle: %v", err)
	}
	if len(pub.deliveries) != 1 {
		t.Fatalf("expected 1 enqueued delivery, got %d", len(pub.deliveries))
	}

	got := pub.deliveries[0]
	if got.ChannelID != "ch-1" || got.EventType != TypeMessageReceived || got.URL != "https://desklenz.example/webhook" {
		t.Errorf("unexpected delivery envelope fields: %+v", got)
	}
	if len(got.Body) == 0 {
		t.Fatal("delivery must carry the exact signed body bytes")
	}

	// Secret must NOT travel on the stream — only channel id + body do.
	var env Envelope
	if err := json.Unmarshal(got.Body, &env); err != nil {
		t.Fatalf("body is not a valid envelope: %v", err)
	}
	if env.Type != TypeMessageReceived || env.TenantID != "tenant-1" {
		t.Errorf("unexpected envelope: %+v", env)
	}
}

func TestDispatcherSkipsChannelWithoutWebhook(t *testing.T) {
	pub := &recordingPublisher{}
	ch := newTestChannel()
	ch.WebhookURL = ""
	d := NewDispatcher(pub, &fakeChannels{channel: ch})

	err := d.handle(context.Background(), &nats.Event{
		Type:     nats.EventMessageReceived,
		TenantID: "t",
		Payload:  map[string]interface{}{"channel_id": "ch-1"},
	})
	if err != nil {
		t.Fatalf("handle: %v", err)
	}
	if len(pub.deliveries) != 0 {
		t.Errorf("no webhook configured → nothing should be enqueued, got %d", len(pub.deliveries))
	}
}

func TestDispatcherRedeliversOnEnqueueFailure(t *testing.T) {
	pub := &recordingPublisher{err: context.DeadlineExceeded}
	d := NewDispatcher(pub, &fakeChannels{channel: newTestChannel()})

	err := d.handle(context.Background(), &nats.Event{
		Type:     nats.EventMessageReceived,
		TenantID: "t",
		Payload:  map[string]interface{}{"channel_id": "ch-1"},
	})
	if err == nil {
		t.Error("enqueue failure must return an error so the source event is redelivered")
	}
}

func TestDispatcherDeliveryIDIsDeterministic(t *testing.T) {
	// A redelivered source event must produce the SAME delivery id (NATS dedup
	// key) so the external endpoint isn't POSTed twice.
	pub := &recordingPublisher{}
	d := NewDispatcher(pub, &fakeChannels{channel: newTestChannel()})

	event := &nats.Event{
		Type:     nats.EventMessageReceived,
		TenantID: "tenant-1",
		Payload: map[string]interface{}{
			"channel_id": "ch-1",
			"message_id": "m-42",
		},
	}

	for i := 0; i < 2; i++ {
		if err := d.handle(context.Background(), event); err != nil {
			t.Fatalf("handle: %v", err)
		}
	}
	if len(pub.deliveries) != 2 {
		t.Fatalf("expected 2 enqueues, got %d", len(pub.deliveries))
	}
	if a, b := pub.deliveries[0].ID, pub.deliveries[1].ID; a != b {
		t.Errorf("delivery id must be deterministic for dedup: %q != %q", a, b)
	}
	if pub.deliveries[0].ID != "evt_m-42_received" {
		t.Errorf("unexpected deterministic id: %q", pub.deliveries[0].ID)
	}
}
