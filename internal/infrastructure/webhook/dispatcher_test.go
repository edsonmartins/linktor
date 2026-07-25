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

func TestDispatcherEventSubscriptionFilter(t *testing.T) {
	cases := []struct {
		name          string
		webhookEvents string
		eventType     string
		payload       map[string]interface{}
		wantEnqueued  bool
	}{
		{
			name:          "empty list delivers all (received)",
			webhookEvents: "",
			eventType:     nats.EventMessageReceived,
			payload:       map[string]interface{}{"channel_id": "ch-1", "message_id": "m-1"},
			wantEnqueued:  true,
		},
		{
			name:          "empty list delivers all (read status)",
			webhookEvents: "",
			eventType:     nats.EventMessageRead,
			payload:       map[string]interface{}{"channel_id": "ch-1", "message_id": "m-1", "status": "read"},
			wantEnqueued:  true,
		},
		{
			name:          "subscribed type is delivered",
			webhookEvents: "message.received,message.read",
			eventType:     nats.EventMessageReceived,
			payload:       map[string]interface{}{"channel_id": "ch-1", "message_id": "m-1"},
			wantEnqueued:  true,
		},
		{
			name:          "unsubscribed status type is dropped",
			webhookEvents: "message.received",
			eventType:     nats.EventMessageRead,
			payload:       map[string]interface{}{"channel_id": "ch-1", "message_id": "m-1", "status": "read"},
			wantEnqueued:  false,
		},
		{
			name:          "unsubscribed inbound is dropped",
			webhookEvents: "message.read,message.delivered",
			eventType:     nats.EventMessageReceived,
			payload:       map[string]interface{}{"channel_id": "ch-1", "message_id": "m-1"},
			wantEnqueued:  false,
		},
		{
			name:          "wildcard delivers everything",
			webhookEvents: "*",
			eventType:     nats.EventMessageFailed,
			payload:       map[string]interface{}{"channel_id": "ch-1", "message_id": "m-1", "status": "failed"},
			wantEnqueued:  true,
		},
		{
			name:          "whitespace and casing tolerated",
			webhookEvents: " message.received , message.read ",
			eventType:     nats.EventMessageRead,
			payload:       map[string]interface{}{"channel_id": "ch-1", "message_id": "m-1", "status": "read"},
			wantEnqueued:  true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			pub := &recordingPublisher{}
			ch := newTestChannel()
			ch.Config = map[string]string{cfgWebhookEvents: tc.webhookEvents}
			d := NewDispatcher(pub, &fakeChannels{channel: ch})

			err := d.handle(context.Background(), &nats.Event{
				Type:     tc.eventType,
				TenantID: "tenant-1",
				Payload:  tc.payload,
			})
			if err != nil {
				t.Fatalf("handle: %v", err)
			}

			got := len(pub.deliveries) == 1
			if got != tc.wantEnqueued {
				t.Errorf("enqueued=%v, want %v (deliveries=%d)", got, tc.wantEnqueued, len(pub.deliveries))
			}
		})
	}
}

func TestDispatcherEnqueuesContactCreated(t *testing.T) {
	pub := &recordingPublisher{}
	d := NewDispatcher(pub, &fakeChannels{channel: newTestChannel()})

	err := d.handle(context.Background(), &nats.Event{
		Type:     nats.EventContactCreated,
		TenantID: "tenant-1",
		Payload: map[string]interface{}{
			"contact_id":   "ct-9",
			"name":         "Ada",
			"phone":        "+15550001111",
			"email":        "ada@example.com",
			"channel_id":   "ch-1",
			"channel_type": "whatsapp",
		},
	})
	if err != nil {
		t.Fatalf("handle: %v", err)
	}
	if len(pub.deliveries) != 1 {
		t.Fatalf("expected 1 delivery, got %d", len(pub.deliveries))
	}

	var env Envelope
	if err := json.Unmarshal(pub.deliveries[0].Body, &env); err != nil {
		t.Fatalf("body not an envelope: %v", err)
	}
	if env.Type != TypeContactCreated {
		t.Errorf("type = %q, want %q", env.Type, TypeContactCreated)
	}
	// Deterministic dedup id for a one-shot contact.created.
	if pub.deliveries[0].ID != "evt_ct-9_contact.created" {
		t.Errorf("unexpected id %q", pub.deliveries[0].ID)
	}
	data, _ := env.Data.(map[string]interface{})
	if data["contactId"] != "ct-9" || data["channelId"] != "ch-1" || data["email"] != "ada@example.com" {
		t.Errorf("unexpected contact data: %+v", data)
	}
}

// Status events must carry the channel like every other payload: a consumer serving many channels
// on one endpoint routes by it, and one that resolves the tenant from the channel cannot process
// the event without it.
func TestDispatcherStatusCarriesChannel(t *testing.T) {
	for _, tc := range []struct {
		eventType string
		wantType  string
	}{
		{nats.EventMessageSent, TypeMessageSent},
		{nats.EventMessageDelivered, TypeMessageDelivered},
		{nats.EventMessageRead, TypeMessageRead},
		{nats.EventMessageFailed, TypeMessageFailed},
	} {
		t.Run(tc.wantType, func(t *testing.T) {
			pub := &recordingPublisher{}
			d := NewDispatcher(pub, &fakeChannels{channel: newTestChannel()})

			err := d.handle(context.Background(), &nats.Event{
				Type:     tc.eventType,
				TenantID: "tenant-1",
				Payload: map[string]interface{}{
					"message_id": "msg-7",
					"channel_id": "ch-1",
					"status":     "delivered",
				},
			})
			if err != nil {
				t.Fatalf("handle: %v", err)
			}
			if len(pub.deliveries) != 1 {
				t.Fatalf("expected 1 delivery, got %d", len(pub.deliveries))
			}

			var env Envelope
			if err := json.Unmarshal(pub.deliveries[0].Body, &env); err != nil {
				t.Fatalf("body not an envelope: %v", err)
			}
			if env.Type != tc.wantType {
				t.Errorf("type = %q, want %q", env.Type, tc.wantType)
			}
			data, _ := env.Data.(map[string]interface{})
			if data["channelId"] != "ch-1" {
				t.Errorf("channelId = %v, want ch-1 (data: %+v)", data["channelId"], data)
			}
			if data["messageId"] != "msg-7" {
				t.Errorf("messageId = %v, want msg-7", data["messageId"])
			}
		})
	}
}

func TestDispatcherEnqueuesConversationLifecycle(t *testing.T) {
	cases := []struct {
		eventType     string
		wantType      string
		deterministic bool
	}{
		{nats.EventConversationCreated, TypeConversationCreated, true},
		{nats.EventConversationAssigned, TypeConversationAssigned, false},
		{nats.EventConversationResolved, TypeConversationResolved, false},
		{nats.EventConversationReopened, TypeConversationReopened, false},
		{nats.EventConversationEscalated, TypeConversationEscalated, false},
	}
	for _, tc := range cases {
		t.Run(tc.wantType, func(t *testing.T) {
			pub := &recordingPublisher{}
			d := NewDispatcher(pub, &fakeChannels{channel: newTestChannel()})

			err := d.handle(context.Background(), &nats.Event{
				Type:     tc.eventType,
				TenantID: "tenant-1",
				Payload: map[string]interface{}{
					"conversation_id":  "cv-1",
					"channel_id":       "ch-1",
					"contact_id":       "ct-1",
					"status":           "open",
					"assigned_user_id": "u-1",
				},
			})
			if err != nil {
				t.Fatalf("handle: %v", err)
			}
			if len(pub.deliveries) != 1 {
				t.Fatalf("expected 1 delivery, got %d", len(pub.deliveries))
			}

			var env Envelope
			if err := json.Unmarshal(pub.deliveries[0].Body, &env); err != nil {
				t.Fatalf("body not an envelope: %v", err)
			}
			if env.Type != tc.wantType {
				t.Errorf("type = %q, want %q", env.Type, tc.wantType)
			}
			gotDeterministic := pub.deliveries[0].ID == "evt_cv-1_"+tc.wantType
			if gotDeterministic != tc.deterministic {
				t.Errorf("deterministic id = %v, want %v (id=%q)", gotDeterministic, tc.deterministic, pub.deliveries[0].ID)
			}
			data, _ := env.Data.(map[string]interface{})
			if data["conversationId"] != "cv-1" || data["channelId"] != "ch-1" {
				t.Errorf("unexpected conversation data: %+v", data)
			}
		})
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
