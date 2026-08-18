package webhook

import (
	"context"
	"encoding/json"
	"errors"
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

// fakeMessages keys by "externalID@conversationID" so a test can hold the same external id on two
// conversations — exactly the shape that made an unscoped lookup return the wrong channel's message.
type fakeMessages struct {
	byExternalID map[string]*entity.Message
	err          error
}

func (f *fakeMessages) FindByExternalIDInConversation(
	_ context.Context, externalID, conversationID string,
) (*entity.Message, error) {
	if f.err != nil {
		return nil, f.err
	}
	return f.byExternalID[externalID+"@"+conversationID], nil
}

func reactionEvent() *nats.Event {
	return &nats.Event{
		Type:     nats.EventMessageReceived,
		TenantID: "tenant-1",
		Payload: map[string]interface{}{
			"message_id":          "msg-reaction",
			"channel_id":          "ch-1",
			"conversation_id":     "conv-ch1",
			"content_type":        "text",
			"content":             "👍",
			"is_reaction":         "true",
			"reaction_message_id": "WA-EXTERNAL-1",
		},
	}
}

func inboundOf(t *testing.T, pub *recordingPublisher) InboundData {
	t.Helper()
	if len(pub.deliveries) != 1 {
		t.Fatalf("expected 1 delivery, got %d", len(pub.deliveries))
	}
	var raw struct {
		Data InboundData `json:"data"`
	}
	if err := json.Unmarshal(pub.deliveries[0].Body, &raw); err != nil {
		t.Fatalf("body not an envelope: %v", err)
	}
	return raw.Data
}

// A reaction must reach the consumer as a reaction, with the target translated to our id — the
// consumer never saw the provider's. Before this it arrived as an empty text message.
func TestDispatcherInboundReactionResolvesTarget(t *testing.T) {
	pub := &recordingPublisher{}
	// The same provider message also exists on another channel's conversation. Resolving without
	// the conversation scope would hand the consumer an id it has never seen.
	d := NewDispatcher(pub, &fakeChannels{channel: newTestChannel()}).
		WithMessages(&fakeMessages{byExternalID: map[string]*entity.Message{
			"WA-EXTERNAL-1@conv-ch1":   {ID: "linktor-msg-1"},
			"WA-EXTERNAL-1@conv-outro": {ID: "msg-de-outro-canal"},
		}})

	if err := d.handle(context.Background(), reactionEvent()); err != nil {
		t.Fatalf("handle: %v", err)
	}

	got := inboundOf(t, pub).Message.Reaction
	if got == nil {
		t.Fatal("reaction block missing — consumer would see an empty message")
	}
	if got.Emoji != "👍" {
		t.Errorf("emoji = %q, want 👍", got.Emoji)
	}
	if got.TargetMessageID != "linktor-msg-1" {
		t.Errorf("targetMessageId = %q, want linktor-msg-1", got.TargetMessageID)
	}
	if got.TargetChannelMessageID != "WA-EXTERNAL-1" {
		t.Errorf("targetChannelMessageId = %q, want WA-EXTERNAL-1", got.TargetChannelMessageID)
	}
}

// Target unknown (older than the integration, or lookup unavailable): still ship the reaction with
// the provider id rather than dropping it or pretending it is a message.
func TestDispatcherInboundReactionWithUnresolvableTarget(t *testing.T) {
	for _, tc := range []struct {
		name     string
		messages MessageLookup
	}{
		{"sem lookup", nil},
		{"alvo desconhecido", &fakeMessages{byExternalID: map[string]*entity.Message{}}},
		{"lookup falha", &fakeMessages{err: errors.New("db down")}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			pub := &recordingPublisher{}
			d := NewDispatcher(pub, &fakeChannels{channel: newTestChannel()})
			if tc.messages != nil {
				d = d.WithMessages(tc.messages)
			}

			if err := d.handle(context.Background(), reactionEvent()); err != nil {
				t.Fatalf("handle: %v", err)
			}

			got := inboundOf(t, pub).Message.Reaction
			if got == nil {
				t.Fatal("reaction block missing")
			}
			if got.TargetMessageID != "" {
				t.Errorf("targetMessageId = %q, want empty", got.TargetMessageID)
			}
			if got.TargetChannelMessageID != "WA-EXTERNAL-1" {
				t.Errorf("targetChannelMessageId = %q", got.TargetChannelMessageID)
			}
		})
	}
}

// An ordinary message must not grow a reaction block.
func TestDispatcherInboundMessageHasNoReaction(t *testing.T) {
	pub := &recordingPublisher{}
	d := NewDispatcher(pub, &fakeChannels{channel: newTestChannel()}).
		WithMessages(&fakeMessages{byExternalID: map[string]*entity.Message{}})

	err := d.handle(context.Background(), &nats.Event{
		Type:     nats.EventMessageReceived,
		TenantID: "tenant-1",
		Payload: map[string]interface{}{
			"message_id":   "msg-1",
			"channel_id":   "ch-1",
			"content_type": "text",
			"content":      "oi",
		},
	})
	if err != nil {
		t.Fatalf("handle: %v", err)
	}

	if got := inboundOf(t, pub).Message.Reaction; got != nil {
		t.Errorf("reaction should be absent, got %+v", got)
	}
}

// A group message must carry the group block so the consumer can thread by group;
// the individual who spoke stays in Message.SenderID.
func TestDispatcherInboundGroupCarriesGroupId(t *testing.T) {
	pub := &recordingPublisher{}
	d := NewDispatcher(pub, &fakeChannels{channel: newTestChannel()})

	err := d.handle(context.Background(), &nats.Event{
		Type:     nats.EventMessageReceived,
		TenantID: "tenant-1",
		Payload: map[string]interface{}{
			"message_id":   "m-g1",
			"channel_id":   "ch-1",
			"content_type": "text",
			"content":      "manda o invite pf",
			"sender_id":    "5512999999999",
			"is_group":     "true",
			"chat_jid":     "120363000000000000@g.us",
		},
	})
	if err != nil {
		t.Fatalf("handle: %v", err)
	}

	data := inboundOf(t, pub)
	if data.Group == nil {
		t.Fatal("group message must carry a group block")
	}
	if data.Group.ID != "120363000000000000@g.us" {
		t.Errorf("group.id = %q, want the chat_jid", data.Group.ID)
	}
	if data.Message.SenderID != "5512999999999" {
		t.Errorf("senderId = %q, want the individual who spoke", data.Message.SenderID)
	}
}

// A 1:1 message must not carry a group block (consumers key those by contact).
func TestDispatcherInbound1x1HasNoGroup(t *testing.T) {
	pub := &recordingPublisher{}
	d := NewDispatcher(pub, &fakeChannels{channel: newTestChannel()})

	err := d.handle(context.Background(), &nats.Event{
		Type:     nats.EventMessageReceived,
		TenantID: "tenant-1",
		Payload: map[string]interface{}{
			"message_id":   "m-1x1",
			"channel_id":   "ch-1",
			"content_type": "text",
			"content":      "oi",
			"sender_id":    "5511888888888",
		},
	})
	if err != nil {
		t.Fatalf("handle: %v", err)
	}
	if g := inboundOf(t, pub).Group; g != nil {
		t.Errorf("1:1 message must have no group block, got %+v", g)
	}
}

// A mention (comma-joined JIDs in the payload) is split into Message.Mentions so
// the consumer can tell the manager was called out and skip the debounce.
func TestDispatcherInboundCarriesMentions(t *testing.T) {
	pub := &recordingPublisher{}
	d := NewDispatcher(pub, &fakeChannels{channel: newTestChannel()})

	err := d.handle(context.Background(), &nats.Event{
		Type:     nats.EventMessageReceived,
		TenantID: "tenant-1",
		Payload: map[string]interface{}{
			"message_id":   "m-men",
			"channel_id":   "ch-1",
			"content_type": "text",
			"content":      "@gestor decide isso pf",
			"sender_id":    "5512999999999",
			"is_group":     "true",
			"chat_jid":     "120363000000000000@g.us",
			"mentions":     "5511777777777@s.whatsapp.net,5512999999999@s.whatsapp.net",
		},
	})
	if err != nil {
		t.Fatalf("handle: %v", err)
	}

	got := inboundOf(t, pub).Message.Mentions
	want := []string{"5511777777777@s.whatsapp.net", "5512999999999@s.whatsapp.net"}
	if len(got) != len(want) {
		t.Fatalf("mentions = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("mentions[%d] = %q, want %q", i, got[i], want[i])
		}
	}
}

// No mention → the field stays absent (1:1 and quiet group messages unaffected).
func TestDispatcherInboundNoMentions(t *testing.T) {
	pub := &recordingPublisher{}
	d := NewDispatcher(pub, &fakeChannels{channel: newTestChannel()})

	err := d.handle(context.Background(), &nats.Event{
		Type:     nats.EventMessageReceived,
		TenantID: "tenant-1",
		Payload: map[string]interface{}{
			"message_id":   "m-nomen",
			"channel_id":   "ch-1",
			"content_type": "text",
			"content":      "bom dia",
			"sender_id":    "5511888888888",
		},
	})
	if err != nil {
		t.Fatalf("handle: %v", err)
	}
	if m := inboundOf(t, pub).Message.Mentions; m != nil {
		t.Errorf("message without mention must have no mentions, got %+v", m)
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

// Authorship used to be a constant pair — "contact"/"inbound" — which held only
// while everything arriving was the customer talking. On a channel whose device
// stays in the operator's hand it stops holding, and the failure is silent: a
// plausible transcript with the sides swapped.

func TestDispatcherInboundFromOperatorIsNotTheContact(t *testing.T) {
	pub := &recordingPublisher{}
	d := NewDispatcher(pub, &fakeChannels{channel: newTestChannel()})

	err := d.handle(context.Background(), &nats.Event{
		Type:     nats.EventMessageReceived,
		TenantID: "tenant-1",
		Payload: map[string]interface{}{
			"message_id":   "m-op",
			"channel_id":   "ch-1",
			"content_type": "text",
			"content":      "consigo dividir em duas vezes",
			"sender_id":    "5521999998888",
			"sender_type":  "user",
		},
	})
	if err != nil {
		t.Fatalf("handle: %v", err)
	}

	data := inboundOf(t, pub)
	if data.Message.SenderType != "user" {
		t.Errorf("senderType = %q, want the operator's own type", data.Message.SenderType)
	}
	if data.Message.Direction != "outbound" {
		t.Errorf("direction = %q, want outbound — it left us toward the customer", data.Message.Direction)
	}
}

func TestDispatcherInboundFromContactStaysInbound(t *testing.T) {
	pub := &recordingPublisher{}
	d := NewDispatcher(pub, &fakeChannels{channel: newTestChannel()})

	err := d.handle(context.Background(), &nats.Event{
		Type:     nats.EventMessageReceived,
		TenantID: "tenant-1",
		Payload: map[string]interface{}{
			"message_id":   "m-c",
			"channel_id":   "ch-1",
			"content_type": "text",
			"content":      "já te envio o comprovante",
			"sender_id":    "5524999194577",
			"sender_type":  "contact",
		},
	})
	if err != nil {
		t.Fatalf("handle: %v", err)
	}

	data := inboundOf(t, pub)
	if data.Message.SenderType != "contact" || data.Message.Direction != "inbound" {
		t.Errorf("got %q/%q, want contact/inbound",
			data.Message.SenderType, data.Message.Direction)
	}
}

func TestDispatcherInboundWithoutSenderTypeKeepsOldBehaviour(t *testing.T) {
	// An event queued before this shipped, or a channel with no notion of an
	// operator device. Both are the customer talking, and both must keep
	// producing exactly the envelope integrators already parse.
	pub := &recordingPublisher{}
	d := NewDispatcher(pub, &fakeChannels{channel: newTestChannel()})

	err := d.handle(context.Background(), &nats.Event{
		Type:     nats.EventMessageReceived,
		TenantID: "tenant-1",
		Payload: map[string]interface{}{
			"message_id":   "m-legado",
			"channel_id":   "ch-1",
			"content_type": "text",
			"content":      "bom dia",
			"sender_id":    "5521988887777",
		},
	})
	if err != nil {
		t.Fatalf("handle: %v", err)
	}

	data := inboundOf(t, pub)
	if data.Message.SenderType != "contact" || data.Message.Direction != "inbound" {
		t.Errorf("got %q/%q, want contact/inbound for an event without sender_type",
			data.Message.SenderType, data.Message.Direction)
	}
}
