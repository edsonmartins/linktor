package linktor

import (
	"encoding/json"
	"testing"
)

func TestDecodeDataReferralEErrorCode(t *testing.T) {
	var recebido WebhookEvent
	if err := json.Unmarshal([]byte(`{"id":"e1","type":"message.received","timestamp":"2026-10-23T12:00:00Z","tenantId":"t1",
	  "environment":"sandbox","data":{"message":{"id":"m1","referral":{"sourceType":"ad","sourceId":"120","sourceUrl":"https://fb.me/x",
	  "headline":"H","body":"B","ctwaClid":"clid-1","mediaType":"image","mediaUrl":"https://img.test/a.jpg"}}}}`), &recebido); err != nil {
		t.Fatal(err)
	}
	if recebido.Environment != "sandbox" {
		t.Errorf("environment = %q", recebido.Environment)
	}
	var d struct {
		Message struct {
			Referral *ReferralData `json:"referral"`
		} `json:"message"`
	}
	if err := recebido.DecodeData(&d); err != nil {
		t.Fatal(err)
	}
	want := ReferralData{SourceType: "ad", SourceID: "120", SourceURL: "https://fb.me/x", Headline: "H", Body: "B",
		CtwaClid: "clid-1", MediaType: "image", MediaURL: "https://img.test/a.jpg"}
	if d.Message.Referral == nil || *d.Message.Referral != want {
		t.Fatalf("referral = %+v", d.Message.Referral)
	}

	var falhou WebhookEvent
	_ = json.Unmarshal([]byte(`{"id":"e2","type":"message.failed","timestamp":"2026-10-23T12:00:00Z","tenantId":"t1",
	  "data":{"messageId":"m2","status":"failed","direction":"outbound","timestamp":"2026-10-23T12:00:00Z","error":"Outside 24h window",
	  "errorCode":"131047","channelId":"ch-1","channelType":"whatsapp_official"}}`), &falhou)
	var s StatusData
	if err := falhou.DecodeData(&s); err != nil {
		t.Fatal(err)
	}
	if s.ErrorCode != "131047" || s.Error != "Outside 24h window" || s.MessageID != "m2" || s.ChannelID != "ch-1" {
		t.Fatalf("status = %+v", s)
	}
}
