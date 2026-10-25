package main

import (
	"testing"

	"github.com/msgfy/linktor/internal/infrastructure/nats"
)

// The status event keeps its keys for every channel and adds the provider's original event (email)
// only when there is one.
func TestStatusWebhookPayload(t *testing.T) {
	plain := statusWebhookPayload("m-1", "ch-1", &nats.StatusUpdate{ExternalID: "x", Status: "delivered"})
	if len(plain) != 5 || plain["status"] != "delivered" || plain["channel_id"] != "ch-1" {
		t.Errorf("payload without the original event changed: %+v", plain)
	}
	for _, k := range []string{"provider_event", "bounce_type", "url"} {
		if _, ok := plain[k]; ok {
			t.Errorf("%s must be absent: %+v", k, plain)
		}
	}
	email := statusWebhookPayload("m-1", "ch-1", &nats.StatusUpdate{Status: "failed", ErrorMessage: "550", ProviderEvent: "bounced",
		BounceType: "hard"})
	if email["provider_event"] != "bounced" || email["bounce_type"] != "hard" || email["error"] != "550" {
		t.Errorf("bounce not carried: %+v", email)
	}
	click := statusWebhookPayload("m-1", "ch-1", &nats.StatusUpdate{Status: "read", ProviderEvent: "clicked", URL: "https://loja.exemplo.test/o"})
	if click["url"] != "https://loja.exemplo.test/o" {
		t.Errorf("url not carried: %+v", click)
	}
}
