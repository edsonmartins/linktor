package handlers

import (
	"context"
	"testing"
	"time"

	"github.com/msgfy/linktor/internal/adapters/email"
	"github.com/msgfy/linktor/internal/domain/entity"
	"github.com/msgfy/linktor/internal/infrastructure/nats"
)

// statusRecorder captures the status updates the handler publishes.
type statusRecorder struct {
	nats.Publisher
	updates []*nats.StatusUpdate
}

func (r *statusRecorder) PublishStatusUpdate(_ context.Context, s *nats.StatusUpdate) error {
	r.updates = append(r.updates, s)
	return nil
}

// The email status keeps folding the provider's event into the contract's status values, and now
// carries the original event, the bounce type and the clicked link alongside (additive).
func TestProcessEmailStatusCarriesProviderEvent(t *testing.T) {
	cases := []struct {
		name   string
		in     email.StatusCallback
		status string
	}{
		{"click", email.StatusCallback{Status: email.StatusClicked, ProviderEvent: email.ProviderEventClicked, URL: "https://loja.exemplo.test/o"}, "read"},
		{"open", email.StatusCallback{Status: email.StatusOpened, ProviderEvent: email.ProviderEventOpened}, "read"},
		{"unsubscribe", email.StatusCallback{Status: email.StatusUnsubscribed, ProviderEvent: email.ProviderEventUnsubscribed}, "read"},
		{"hard bounce", email.StatusCallback{Status: email.StatusBounced, ProviderEvent: email.ProviderEventBounced, BounceType: email.BounceHard, ErrorMessage: "550"}, "failed"},
		{"spam", email.StatusCallback{Status: email.StatusSpam, ProviderEvent: email.ProviderEventSpamReport}, "failed"},
		{"delivered", email.StatusCallback{Status: email.StatusDelivered, ProviderEvent: email.ProviderEventDelivered}, "delivered"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			rec := &statusRecorder{}
			h := &WebhookHandler{producer: rec}
			in := c.in
			in.ExternalID = "ext-1"
			in.Timestamp = time.Unix(1, 0)
			h.processEmailStatus(context.Background(), &entity.Channel{ID: "ch-1"}, &in)
			if len(rec.updates) != 1 {
				t.Fatalf("expected 1 update, got %d", len(rec.updates))
			}
			u := rec.updates[0]
			if u.Status != c.status {
				t.Errorf("status = %q, want %q", u.Status, c.status)
			}
			if u.ProviderEvent != c.in.ProviderEvent || u.BounceType != c.in.BounceType || u.URL != c.in.URL {
				t.Errorf("original event lost: %+v", u)
			}
			if u.ChannelType != "email" || u.ExternalID != "ext-1" {
				t.Errorf("unexpected update: %+v", u)
			}
		})
	}
}
