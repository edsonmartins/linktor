package nats

import (
	"context"
	"testing"
)

func TestDeliveryAttempt(t *testing.T) {
	if _, _, ok := DeliveryAttemptFrom(context.Background()); ok {
		t.Fatal("sem tentativa no contexto")
	}
	n, max, ok := DeliveryAttemptFrom(WithDeliveryAttempt(context.Background(), 3, 10))
	if !ok || n != 3 || max != 10 {
		t.Fatalf("tentativa = %d/%d ok=%v, esperado 3/10", n, max, ok)
	}
}
