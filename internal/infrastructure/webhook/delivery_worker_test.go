package webhook

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/msgfy/linktor/internal/infrastructure/nats"
	"github.com/msgfy/linktor/pkg/errors"
)

func TestDeliveryWorkerSignsFreshAndDelivers(t *testing.T) {
	body := []byte(`{"id":"evt_1","type":"message.received"}`)
	secret := "s3cr3t"

	var gotSig, gotTs string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotSig = r.Header.Get(SignatureHeader)
		gotTs = r.Header.Get(TimestampHeader)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	ch := newTestChannel()
	ch.WebhookURL = srv.URL
	ch.Credentials[credWebhookSecret] = secret

	worker := NewDeliveryWorker(NewWebhookProducer(), &fakeChannels{channel: ch})
	err := worker.Handle(context.Background(), &nats.WebhookDelivery{
		ChannelID: "ch-1",
		EventType: TypeMessageReceived,
		Body:      body,
	})
	if err != nil {
		t.Fatalf("Handle: %v", err)
	}
	if gotTs == "" {
		t.Error("timestamp header must be set fresh at delivery time")
	}
	// The wire signature binds the fresh timestamp to the body, so recompute it
	// with the timestamp the worker actually sent.
	wantSig := ComputeTimestampedSignature(gotTs, body, secret)
	if gotSig != wantSig {
		t.Errorf("signature header mismatch: got %q want %q", gotSig, wantSig)
	}
}

func TestDeliveryWorkerReturnsErrorOn5xxForRetry(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	ch := newTestChannel()
	ch.WebhookURL = srv.URL
	worker := NewDeliveryWorker(NewWebhookProducer(), &fakeChannels{channel: ch})

	err := worker.Handle(context.Background(), &nats.WebhookDelivery{
		ChannelID: "ch-1",
		EventType: TypeMessageReceived,
		Body:      []byte(`{}`),
	})
	if err == nil {
		t.Error("a 5xx must return an error so NATS redelivers (and eventually DLQs)")
	}
}

func TestDeliveryWorkerDropsWhenChannelGone(t *testing.T) {
	worker := NewDeliveryWorker(NewWebhookProducer(), &fakeChannels{channel: nil})
	err := worker.Handle(context.Background(), &nats.WebhookDelivery{
		ChannelID: "ch-1",
		EventType: TypeMessageReceived,
		Body:      []byte(`{}`),
	})
	if err != nil {
		t.Errorf("a gone channel should be dropped (ack), got error: %v", err)
	}
}

func TestDeliveryWorkerIgnoresEmptyBody(t *testing.T) {
	worker := NewDeliveryWorker(NewWebhookProducer(), &fakeChannels{channel: newTestChannel()})
	if err := worker.Handle(context.Background(), &nats.WebhookDelivery{ChannelID: "ch-1"}); err != nil {
		t.Errorf("empty body should ack, got: %v", err)
	}
}

func TestDeliveryWorkerDropsDeletedChannel(t *testing.T) {
	// A since-deleted channel surfaces as a not-found error, not (nil,nil). It must
	// be dropped (ack), not retried, so it doesn't flood the DLQ.
	worker := NewDeliveryWorker(NewWebhookProducer(), &fakeChannels{
		err: errors.New(errors.ErrCodeChannelNotFound, "channel not found"),
	})
	err := worker.Handle(context.Background(), &nats.WebhookDelivery{
		ChannelID: "ch-1",
		EventType: TypeMessageReceived,
		Body:      []byte(`{}`),
	})
	if err != nil {
		t.Errorf("a deleted channel should be dropped (ack), got error: %v", err)
	}
}

func TestDeliveryWorkerRetriesOnTransientLookupError(t *testing.T) {
	worker := NewDeliveryWorker(NewWebhookProducer(), &fakeChannels{
		err: errors.New(errors.ErrCodeInternal, "db unavailable"),
	})
	err := worker.Handle(context.Background(), &nats.WebhookDelivery{
		ChannelID: "ch-1",
		EventType: TypeMessageReceived,
		Body:      []byte(`{}`),
	})
	if err == nil {
		t.Error("a transient lookup error must return an error so NATS retries")
	}
}

func TestDeliveryWorkerDropsWhenNoSecret(t *testing.T) {
	// Signing with an empty key would produce a forgeable signature: the worker
	// must NOT deliver, and must ack (no retry).
	called := false
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	ch := newTestChannel()
	ch.WebhookURL = srv.URL
	ch.Credentials = map[string]string{} // no webhook_secret

	worker := NewDeliveryWorker(NewWebhookProducer(), &fakeChannels{channel: ch})
	err := worker.Handle(context.Background(), &nats.WebhookDelivery{
		ChannelID: "ch-1",
		EventType: TypeMessageReceived,
		Body:      []byte(`{}`),
	})
	if err != nil {
		t.Errorf("missing secret should ack (drop), got error: %v", err)
	}
	if called {
		t.Error("must not POST a forgeable (empty-key) signed delivery")
	}
}
