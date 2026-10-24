package nats

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/msgfy/linktor/internal/infrastructure/metrics"
	"github.com/nats-io/nats.go/jetstream"
	"github.com/prometheus/client_golang/prometheus/testutil"
)

// fakeMsg implements just what deadLetterVia touches.
type fakeMsg struct {
	jetstream.Msg
	acked, naked bool
}

func (m *fakeMsg) Data() []byte                       { return []byte(`{"id":"evt-1"}`) }
func (m *fakeMsg) Ack() error                         { m.acked = true; return nil }
func (m *fakeMsg) NakWithDelay(_ time.Duration) error { m.naked = true; return nil }

// A dead-lettered message must move the dlq_messages_total counter for its
// consumer — that is what an operator alerts on; before, it was only a log line.
func TestDeadLetterCountsTheMetric(t *testing.T) {
	before := testutil.ToFloat64(metrics.DLQMessages.WithLabelValues("webhooks-teste"))
	var subject string
	msg := &fakeMsg{}
	deadLetterVia(context.Background(), func(_ context.Context, s string, _ []byte) error { subject = s; return nil },
		"webhooks-teste", msg, errors.New("endpoint 503"))
	if got := testutil.ToFloat64(metrics.DLQMessages.WithLabelValues("webhooks-teste")); got != before+1 {
		t.Fatalf("dlq_messages_total = %v, want %v", got, before+1)
	}
	if subject != SubjectDLQ("webhooks-teste") || !msg.acked {
		t.Fatalf("subject=%q acked=%v", subject, msg.acked)
	}
}

// If the DLQ publish fails, the message is NAK'd (not lost) and not counted.
func TestDeadLetterPublishFailureNaksWithoutCounting(t *testing.T) {
	before := testutil.ToFloat64(metrics.DLQMessages.WithLabelValues("webhooks-falha"))
	msg := &fakeMsg{}
	deadLetterVia(context.Background(), func(context.Context, string, []byte) error { return errors.New("dlq fora") },
		"webhooks-falha", msg, errors.New("endpoint 503"))
	if testutil.ToFloat64(metrics.DLQMessages.WithLabelValues("webhooks-falha")) != before || !msg.naked || msg.acked {
		t.Fatalf("naked=%v acked=%v", msg.naked, msg.acked)
	}
}
