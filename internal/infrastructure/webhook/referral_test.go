package webhook

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	whatsappofficial "github.com/msgfy/linktor/internal/adapters/whatsapp_official"
	"github.com/msgfy/linktor/internal/application/usecase"
	"github.com/msgfy/linktor/internal/infrastructure/nats"
)

// A Meta Click-to-WhatsApp webhook (fictitious ids) goes through the official adapter, rides the
// message.received event with the referral_* keys the outbox carries, and reaches the consumer as
// the structured `referral` — ctwa_clid included.
func TestReferralCtwaDaMetaChegaAoEnvelope(t *testing.T) {
	meta := []byte(`{"object":"whatsapp_business_account","entry":[{"id":"WABA-1","changes":[{"field":"messages","value":{
	  "messaging_product":"whatsapp",
	  "metadata":{"display_phone_number":"5511900000000","phone_number_id":"PN-1"},
	  "contacts":[{"profile":{"name":"Cliente Ficticio"},"wa_id":"5511999990000"}],
	  "messages":[{"from":"5511999990000","id":"wamid.ctwa1","timestamp":"1700000000","type":"text",
	    "text":{"body":"Quero saber do produto"},
	    "referral":{"source_url":"https://fb.me/anuncio-ficticio","source_id":"120200000000000","source_type":"ad",
	      "headline":"Promoção fictícia","body":"Fale com a gente","media_type":"video",
	      "video_url":"https://video.example.test/criativo.mp4","thumbnail_url":"https://video.example.test/t.jpg",
	      "ctwa_clid":"clid-ficticio-123"}}]}}]}]}`)
	proc := whatsappofficial.NewWebhookProcessor(nil)
	payload, err := proc.ParseWebhook(meta)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	msgs := proc.ExtractMessages(payload)
	if len(msgs) != 1 {
		t.Fatalf("messages = %d", len(msgs))
	}

	// What the outbox event carries (receive_message.go): the referral_* keys present on the message.
	event := map[string]interface{}{
		"channel_id": "ch-1", "message_id": "m-ctwa", "conversation_id": "c-1", "contact_id": "ct-1",
		"content_type": "text", "content": "Quero saber do produto",
	}
	for _, k := range usecase.ReferralMetadataKeys {
		if v := msgs[0].Metadata[k]; v != "" {
			event[k] = v
		}
	}

	pub := &recordingPublisher{}
	d := NewDispatcher(pub, &fakeChannels{channel: newTestChannel()})
	if err := d.handle(context.Background(), &nats.Event{Type: nats.EventMessageReceived, TenantID: "tenant-1", Payload: event}); err != nil {
		t.Fatalf("handle: %v", err)
	}
	got := inboundOf(t, pub).Message.Referral
	want := ReferralPayload{
		SourceType: "ad", SourceID: "120200000000000", SourceURL: "https://fb.me/anuncio-ficticio",
		Headline: "Promoção fictícia", Body: "Fale com a gente", CtwaClid: "clid-ficticio-123",
		MediaType: "video", MediaURL: "https://video.example.test/criativo.mp4",
	}
	if got == nil || *got != want {
		t.Fatalf("referral = %+v, want %+v", got, want)
	}
	if !strings.Contains(string(pub.deliveries[0].Body), `"ctwaClid":"clid-ficticio-123"`) {
		t.Errorf("wire JSON must carry ctwaClid: %s", pub.deliveries[0].Body)
	}
}

// Image wins over video for mediaUrl; a message without referral keys has no `referral` on the wire.
func TestDispatcherReferralImagemEAusencia(t *testing.T) {
	pub := &recordingPublisher{}
	d := NewDispatcher(pub, &fakeChannels{channel: newTestChannel()})
	_ = d.handle(context.Background(), &nats.Event{Type: nats.EventMessageReceived, TenantID: "tenant-1", Payload: map[string]interface{}{
		"channel_id": "ch-1", "message_id": "m-img", "content_type": "text", "content": "oi",
		"referral_source_type": "ad", "referral_image_url": "https://img.example.test/a.jpg", "referral_video_url": "https://v.example.test/a.mp4",
	}})
	if r := inboundOf(t, pub).Message.Referral; r == nil || r.MediaURL != "https://img.example.test/a.jpg" || r.SourceType != "ad" {
		t.Fatalf("referral = %+v", r)
	}

	pub2 := &recordingPublisher{}
	d2 := NewDispatcher(pub2, &fakeChannels{channel: newTestChannel()})
	_ = d2.handle(context.Background(), &nats.Event{Type: nats.EventMessageReceived, TenantID: "tenant-1", Payload: map[string]interface{}{
		"channel_id": "ch-1", "message_id": "m-sem", "content_type": "text", "content": "oi",
	}})
	if r := inboundOf(t, pub2).Message.Referral; r != nil {
		t.Fatalf("no referral keys must mean no referral, got %+v", r)
	}
	if strings.Contains(string(pub2.deliveries[0].Body), `"referral"`) {
		t.Errorf("absent referral must be omitted from the wire: %s", pub2.deliveries[0].Body)
	}
}

// message.failed carries the provider's error code next to the text; other statuses omit it.
func TestDispatcherStatusCarriesErrorCode(t *testing.T) {
	pub := &recordingPublisher{}
	d := NewDispatcher(pub, &fakeChannels{channel: newTestChannel()})
	_ = d.handle(context.Background(), &nats.Event{Type: nats.EventMessageFailed, TenantID: "tenant-1", Payload: map[string]interface{}{
		"message_id": "msg-f", "channel_id": "ch-1", "status": "failed", "error": "Outside 24h window", "error_code": "131047",
	}})
	var env struct {
		Data map[string]interface{} `json:"data"`
	}
	if err := json.Unmarshal(pub.deliveries[0].Body, &env); err != nil {
		t.Fatalf("envelope: %v", err)
	}
	if env.Data["errorCode"] != "131047" || env.Data["error"] != "Outside 24h window" {
		t.Fatalf("data = %+v", env.Data)
	}

	pub2 := &recordingPublisher{}
	d2 := NewDispatcher(pub2, &fakeChannels{channel: newTestChannel()})
	_ = d2.handle(context.Background(), &nats.Event{Type: nats.EventMessageDelivered, TenantID: "tenant-1", Payload: map[string]interface{}{
		"message_id": "msg-d", "channel_id": "ch-1", "status": "delivered",
	}})
	if strings.Contains(string(pub2.deliveries[0].Body), "errorCode") {
		t.Errorf("errorCode must be omitted when absent: %s", pub2.deliveries[0].Body)
	}
}
