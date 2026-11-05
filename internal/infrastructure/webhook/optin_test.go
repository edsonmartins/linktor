package webhook

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/msgfy/linktor/internal/infrastructure/nats"
)

// Os três tipos opt-in do linktor-channel-v1 juntos (template.status_changed, template.quality_changed e
// conversation.owner_changed): o canal que lista os três recebe os três; o que não lista nenhum — sem filtro, com "*"
// ou só com tipos antigos — não recebe nenhum.
func TestOptIn_TresTiposJuntos(t *testing.T) {
	eventos := []*nats.Event{
		templateStatusEvent(),
		{Type: nats.EventTemplateQualityChanged, TenantID: "tenant-1", Payload: map[string]interface{}{
			"template_id": "tpl-1", "external_id": "987", "channel_id": "ch-1", "name": "boas_vindas",
			"language": "pt_BR", "category": "UTILITY", "previous_quality": "GREEN", "quality": "RED"}},
		eventoOwnerChanged(),
	}
	tipos := []string{TypeTemplateStatusChanged, TypeTemplateQualityChanged, TypeConversationOwnerChanged}

	entregues := func(filtro string) []string {
		ch := newTestChannel()
		ch.Config = map[string]string{cfgWebhookEvents: filtro}
		pub := &recordingPublisher{}
		d := NewDispatcher(pub, &fakeChannels{channel: ch})
		for _, e := range eventos {
			if err := d.handle(context.Background(), e); err != nil {
				t.Fatalf("handle %s: %v", e.Type, err)
			}
		}
		var got []string
		for _, wh := range pub.deliveries {
			var env Envelope
			if err := json.Unmarshal(wh.Body, &env); err != nil {
				t.Fatalf("body not an envelope: %v", err)
			}
			got = append(got, env.Type)
		}
		return got
	}

	got := entregues("message.received, template.status_changed,template.quality_changed , conversation.owner_changed")
	if len(got) != 3 {
		t.Fatalf("canal com os três opt-in: entregues %v, want %v", got, tipos)
	}
	for i, tipo := range tipos {
		if got[i] != tipo {
			t.Errorf("entrega %d = %q, want %q", i, got[i], tipo)
		}
	}
	for _, filtro := range []string{"", "*", "message.received,message.read,conversation.created"} {
		if got := entregues(filtro); len(got) != 0 {
			t.Errorf("webhook_events=%q: entregues %v, want nenhum opt-in", filtro, got)
		}
	}
	for _, tipo := range tipos {
		if !optInEventTypes[tipo] {
			t.Errorf("%s fora de optInEventTypes", tipo)
		}
	}
}
