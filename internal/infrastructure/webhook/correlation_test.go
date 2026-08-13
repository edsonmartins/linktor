package webhook

import (
	"context"
	"errors"
	"testing"

	"github.com/msgfy/linktor/internal/domain/entity"
	"github.com/msgfy/linktor/internal/infrastructure/nats"
)

// replyEvent builds an inbound "message.received" event that quotes the outbound
// message identified by replyToExternalID (empty = not a reply).
func replyEvent(conversationID, replyToExternalID string) *nats.Event {
	payload := map[string]interface{}{
		"message_id":      "msg_in",
		"channel_id":      "ch-1",
		"conversation_id": conversationID,
		"contact_id":      "contact_1",
		"content_type":    "text",
		"content":         "Pode seguir",
		"sender_id":       "+5511999999999",
	}
	if replyToExternalID != "" {
		payload["reply_to_id"] = replyToExternalID
	}
	return &nats.Event{Type: nats.EventMessageReceived, TenantID: "tenant-1", Payload: payload}
}

// correlatedMessage is an outbound message sent through the direct-send route,
// carrying the integrator's own correlation in its metadata.
func correlatedMessage() *entity.Message {
	return &entity.Message{
		ID:         "msg_out",
		ExternalID: "wamid.OUT-1",
		Metadata: map[string]string{
			"source":             "alcada",
			"alcada_correlation": "token-opaco",
			// Not allowlisted: must never cross into the webhook.
			"idempotency_key": "chave-logica",
			"subject":         "Alçada — despacho",
		},
	}
}

func TestCorrelation_QuotedReplyCarriesTheCorrelation(t *testing.T) {
	pub := &recordingPublisher{}
	d := NewDispatcher(pub, &fakeChannels{channel: newTestChannel()}).
		WithMessages(&fakeMessages{byExternalID: map[string]*entity.Message{
			"wamid.OUT-1@conv-1": correlatedMessage(),
		}})

	if err := d.handle(context.Background(), replyEvent("conv-1", "wamid.OUT-1")); err != nil {
		t.Fatalf("handle: %v", err)
	}

	got := inboundOf(t, pub).Context
	if got == nil {
		t.Fatal("context ausente: a resposta citada perdeu a correlação")
	}
	if got["alcada_correlation"] != "token-opaco" {
		t.Errorf("alcada_correlation = %q, want token-opaco", got["alcada_correlation"])
	}
	if got["source"] != "alcada" {
		t.Errorf("source = %q, want alcada", got["source"])
	}
}

// Only allowlisted keys cross. Outbound metadata also holds operational detail
// (idempotency key, email subject) that must not be echoed back.
func TestCorrelation_OnlyAllowlistedKeysCross(t *testing.T) {
	pub := &recordingPublisher{}
	d := NewDispatcher(pub, &fakeChannels{channel: newTestChannel()}).
		WithMessages(&fakeMessages{byExternalID: map[string]*entity.Message{
			"wamid.OUT-1@conv-1": correlatedMessage(),
		}})

	if err := d.handle(context.Background(), replyEvent("conv-1", "wamid.OUT-1")); err != nil {
		t.Fatalf("handle: %v", err)
	}

	got := inboundOf(t, pub).Context
	if len(got) != 2 {
		t.Fatalf("context = %v, want exactly the 2 allowlisted keys", got)
	}
	for _, forbidden := range []string{"idempotency_key", "subject"} {
		if _, ok := got[forbidden]; ok {
			t.Errorf("chave %q atravessou a allowlist", forbidden)
		}
	}
}

// No citation → no correlation. Correlation is never inferred from the phone
// number, the conversation, timing or the text.
func TestCorrelation_ReplyWithoutCitationHasNoContext(t *testing.T) {
	pub := &recordingPublisher{}
	d := NewDispatcher(pub, &fakeChannels{channel: newTestChannel()}).
		WithMessages(&fakeMessages{byExternalID: map[string]*entity.Message{
			"wamid.OUT-1@conv-1": correlatedMessage(),
		}})

	if err := d.handle(context.Background(), replyEvent("conv-1", "")); err != nil {
		t.Fatalf("handle: %v", err)
	}

	if got := inboundOf(t, pub).Context; got != nil {
		t.Errorf("context = %v, want ausente para mensagem que não cita nada", got)
	}
}

func TestCorrelation_QuotedMessageWithoutCorrelationHasNoContext(t *testing.T) {
	pub := &recordingPublisher{}
	d := NewDispatcher(pub, &fakeChannels{channel: newTestChannel()}).
		WithMessages(&fakeMessages{byExternalID: map[string]*entity.Message{
			"wamid.OUT-1@conv-1": {
				ID:         "msg_out",
				ExternalID: "wamid.OUT-1",
				Metadata:   map[string]string{"preview_url": "true"},
			},
		}})

	if err := d.handle(context.Background(), replyEvent("conv-1", "wamid.OUT-1")); err != nil {
		t.Fatalf("handle: %v", err)
	}

	if got := inboundOf(t, pub).Context; got != nil {
		t.Errorf("context = %v, want ausente quando a citada não tem correlação", got)
	}
}

// The lookup is (conversation_id, external_id) and never external_id alone: the
// same provider id exists once per channel that received it, so an unscoped
// lookup would correlate against another conversation — and so, potentially,
// another tenant.
func TestCorrelation_CitationFromAnotherConversationDoesNotCorrelate(t *testing.T) {
	pub := &recordingPublisher{}
	d := NewDispatcher(pub, &fakeChannels{channel: newTestChannel()}).
		WithMessages(&fakeMessages{byExternalID: map[string]*entity.Message{
			// The correlated message lives on a different conversation.
			"wamid.OUT-1@conv-outra": correlatedMessage(),
		}})

	if err := d.handle(context.Background(), replyEvent("conv-1", "wamid.OUT-1")); err != nil {
		t.Fatalf("handle: %v", err)
	}

	if got := inboundOf(t, pub).Context; got != nil {
		t.Errorf("context = %v, want ausente: a citada é de outra conversa", got)
	}
}

func TestCorrelation_CitationFromAnotherTenantDoesNotCorrelate(t *testing.T) {
	pub := &recordingPublisher{}
	// The other tenant's message is stored under its own conversation, which this
	// event never names — the conversation scope is what keeps tenants apart.
	d := NewDispatcher(pub, &fakeChannels{channel: newTestChannel()}).
		WithMessages(&fakeMessages{byExternalID: map[string]*entity.Message{
			"wamid.OUT-1@conv-do-outro-tenant": correlatedMessage(),
		}})

	if err := d.handle(context.Background(), replyEvent("conv-1", "wamid.OUT-1")); err != nil {
		t.Fatalf("handle: %v", err)
	}

	if got := inboundOf(t, pub).Context; got != nil {
		t.Errorf("context = %v, want ausente entre tenants", got)
	}
}

// A lookup failure must not drop the message: the webhook still ships, just
// without correlation.
func TestCorrelation_LookupErrorStillDeliversTheMessage(t *testing.T) {
	pub := &recordingPublisher{}
	d := NewDispatcher(pub, &fakeChannels{channel: newTestChannel()}).
		WithMessages(&fakeMessages{err: errors.New("db down")})

	if err := d.handle(context.Background(), replyEvent("conv-1", "wamid.OUT-1")); err != nil {
		t.Fatalf("handle: %v", err)
	}

	data := inboundOf(t, pub)
	if data.Message.ID != "msg_in" {
		t.Errorf("mensagem = %q, want msg_in", data.Message.ID)
	}
	if data.Context != nil {
		t.Errorf("context = %v, want ausente quando o lookup falha", data.Context)
	}
}

// Without a message lookup the dispatcher still ships the event — `context` is
// simply never populated.
func TestCorrelation_WithoutMessageLookupThereIsNoContext(t *testing.T) {
	pub := &recordingPublisher{}
	d := NewDispatcher(pub, &fakeChannels{channel: newTestChannel()})

	if err := d.handle(context.Background(), replyEvent("conv-1", "wamid.OUT-1")); err != nil {
		t.Fatalf("handle: %v", err)
	}

	if got := inboundOf(t, pub).Context; got != nil {
		t.Errorf("context = %v, want ausente sem MessageLookup", got)
	}
}

// A redelivery of the same inbound event collapses at the stream on the same
// dedup id, correlation included — the id must not depend on the correlation.
func TestCorrelation_RedeliveryKeepsTheSameDedupID(t *testing.T) {
	pub := &recordingPublisher{}
	d := NewDispatcher(pub, &fakeChannels{channel: newTestChannel()}).
		WithMessages(&fakeMessages{byExternalID: map[string]*entity.Message{
			"wamid.OUT-1@conv-1": correlatedMessage(),
		}})

	for i := 0; i < 2; i++ {
		if err := d.handle(context.Background(), replyEvent("conv-1", "wamid.OUT-1")); err != nil {
			t.Fatalf("handle #%d: %v", i, err)
		}
	}

	if len(pub.deliveries) != 2 {
		t.Fatalf("expected 2 enqueues, got %d", len(pub.deliveries))
	}
	if pub.deliveries[0].ID != pub.deliveries[1].ID {
		t.Errorf("dedup id mudou entre reentregas: %q vs %q", pub.deliveries[0].ID, pub.deliveries[1].ID)
	}
	if pub.deliveries[0].ID != "evt_msg_in_received" {
		t.Errorf("dedup id = %q, want evt_msg_in_received", pub.deliveries[0].ID)
	}
}

// "source" alone is not a correlation. Linktor stamps it on WhatsApp Business
// App echoes, so a customer replying to a message an operator typed in the
// WhatsApp app must not come back looking correlated.
func TestCorrelation_QualifierAloneDoesNotCreateContext(t *testing.T) {
	pub := &recordingPublisher{}
	d := NewDispatcher(pub, &fakeChannels{channel: newTestChannel()}).
		WithMessages(&fakeMessages{byExternalID: map[string]*entity.Message{
			"wamid.OUT-1@conv-1": {
				ID:         "msg_echo",
				ExternalID: "wamid.OUT-1",
				Metadata:   map[string]string{"source": "business_app", "is_echo": "true"},
			},
		}})

	if err := d.handle(context.Background(), replyEvent("conv-1", "wamid.OUT-1")); err != nil {
		t.Fatalf("handle: %v", err)
	}

	if got := inboundOf(t, pub).Context; got != nil {
		t.Errorf("context = %v, want ausente: \"source\" sozinho não é correlação", got)
	}
}
