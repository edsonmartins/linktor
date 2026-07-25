package webhook

import (
	"encoding/json"
	"strconv"
	"testing"
	"time"
)

// fixedTime keeps golden output deterministic.
var fixedTime = time.Date(2026, 6, 28, 10, 30, 0, 0, time.UTC)

func TestInboundEnvelopeGolden(t *testing.T) {
	env := &Envelope{
		ID:        "evt_123",
		Type:      TypeMessageReceived,
		Timestamp: fixedTime,
		TenantID:  "tenant_abc",
		Data: InboundData{
			Message: InboundMessagePayload{
				ID:          "msg_1",
				Direction:   "inbound",
				ContentType: "text",
				Content:     MessageContent{Text: "olá"},
				SenderID:    "U123",
				SenderType:  "contact",
				Metadata:    map[string]string{"senderName": "João"},
			},
			ConversationID: "conv_1",
			ContactID:      "contact_1",
			ChannelID:      "ch_1",
			ChannelType:    "slack",
		},
	}

	got, err := MarshalEnvelope(env)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	const want = `{"id":"evt_123","type":"message.received","timestamp":"2026-06-28T10:30:00Z","tenantId":"tenant_abc","data":{"message":{"id":"msg_1","direction":"inbound","contentType":"text","content":{"text":"olá"},"senderId":"U123","senderType":"contact","metadata":{"senderName":"João"}},"conversationId":"conv_1","contactId":"contact_1","channelId":"ch_1","channelType":"slack"}}`

	if string(got) != want {
		t.Errorf("inbound envelope mismatch:\n got: %s\nwant: %s", got, want)
	}
}

func TestInboundEnvelopeWithMedia(t *testing.T) {
	env := &Envelope{
		ID:        "evt_456",
		Type:      TypeMessageReceived,
		Timestamp: fixedTime,
		TenantID:  "tenant_abc",
		Data: InboundData{
			Message: InboundMessagePayload{
				ID:          "msg_2",
				Direction:   "inbound",
				ContentType: "image",
				Content: MessageContent{
					Media: &MediaPayload{
						URL:      "https://cdn/x.jpg",
						MimeType: "image/jpeg",
						Filename: "x.jpg",
						Size:     1024,
						Caption:  "look",
					},
				},
				SenderType: "contact",
			},
			ConversationID: "conv_2",
			ContactID:      "contact_2",
			ChannelID:      "ch_2",
			ChannelType:    "teams",
		},
	}

	got, err := MarshalEnvelope(env)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	// Verify the nested media object round-trips with the contract field names.
	var decoded map[string]interface{}
	if err := json.Unmarshal(got, &decoded); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	media := decoded["data"].(map[string]interface{})["message"].(map[string]interface{})["content"].(map[string]interface{})["media"].(map[string]interface{})
	if media["mimeType"] != "image/jpeg" || media["url"] != "https://cdn/x.jpg" {
		t.Errorf("unexpected media payload: %v", media)
	}
}

func TestStatusEnvelopeGolden(t *testing.T) {
	env := &Envelope{
		ID:        "evt_789",
		Type:      TypeMessageDelivered,
		Timestamp: fixedTime,
		TenantID:  "tenant_abc",
		Data: StatusData{
			MessageID:   "msg_3",
			Status:      "delivered",
			Direction:   "outbound",
			Timestamp:   fixedTime,
			ChannelID:   "ch_1",
			ChannelType: "whatsapp",
		},
	}

	got, err := MarshalEnvelope(env)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	const want = `{"id":"evt_789","type":"message.delivered","timestamp":"2026-06-28T10:30:00Z","tenantId":"tenant_abc","data":{"messageId":"msg_3","status":"delivered","direction":"outbound","timestamp":"2026-06-28T10:30:00Z","channelId":"ch_1","channelType":"whatsapp"}}`

	if string(got) != want {
		t.Errorf("status envelope mismatch:\n got: %s\nwant: %s", got, want)
	}
}

func TestComputeSignatureStable(t *testing.T) {
	payload := []byte(`{"hello":"world"}`)
	// Pre-computed HMAC-SHA256(payload, "secret") hex.
	const want = "2677ad3e7c090b2fa2c0fb13020d66d5420879b8316eb356a2d60fb9073bc778"
	if got := ComputeSignature(payload, "secret"); got != want {
		t.Errorf("signature mismatch:\n got: %s\nwant: %s", got, want)
	}
}

func TestSignatureHeaders(t *testing.T) {
	headers := SignatureHeaders([]byte("body"), "secret", fixedTime)
	if headers[SignatureHeader] == "" {
		t.Error("missing signature header")
	}
	if headers[TimestampHeader] != "1782642600" {
		t.Errorf("unexpected timestamp header: %s", headers[TimestampHeader])
	}
}

// The wire signature must bind the timestamp, not just the body: two different
// timestamps over the same body must yield different signatures.
func TestSignatureBindsTimestamp(t *testing.T) {
	body := []byte(`{"hello":"world"}`)
	sigA := ComputeTimestampedSignature("1000", body, "secret")
	sigB := ComputeTimestampedSignature("2000", body, "secret")
	if sigA == sigB {
		t.Error("signature must change when the timestamp changes")
	}
	// And it must differ from the legacy body-only HMAC (proving the header is
	// no longer forgeable by swapping the timestamp).
	if sigA == ComputeSignature(body, "secret") {
		t.Error("timestamped signature must differ from body-only signature")
	}
}

func TestVerifySignatureRoundTrip(t *testing.T) {
	body := []byte(`{"id":"evt_1"}`)
	secret := "s3cr3t"
	headers := SignatureHeaders(body, secret, fixedTime)

	// Verified within tolerance of the signing time → accepted.
	if !VerifySignature(body, headers[SignatureHeader], headers[TimestampHeader], secret, fixedTime.Add(10*time.Second), DefaultTolerance) {
		t.Error("fresh, correctly-signed payload must verify")
	}
}

func TestVerifySignatureRejectsStale(t *testing.T) {
	body := []byte(`{"id":"evt_1"}`)
	secret := "s3cr3t"
	headers := SignatureHeaders(body, secret, fixedTime)

	// Same signature replayed well past the tolerance window → rejected.
	stale := fixedTime.Add(time.Duration(DefaultTolerance+60) * time.Second)
	if VerifySignature(body, headers[SignatureHeader], headers[TimestampHeader], secret, stale, DefaultTolerance) {
		t.Error("a stale (replayed) timestamp must be rejected")
	}
}

func TestVerifySignatureRejectsTamperedTimestamp(t *testing.T) {
	body := []byte(`{"id":"evt_1"}`)
	secret := "s3cr3t"
	headers := SignatureHeaders(body, secret, fixedTime)

	// Attacker rewinds the timestamp header to keep it "fresh" but leaves the
	// original signature: because the signature covers the timestamp, verify fails.
	forgedTs := strconv.FormatInt(fixedTime.Add(2*time.Hour).Unix(), 10)
	if VerifySignature(body, headers[SignatureHeader], forgedTs, secret, fixedTime.Add(2*time.Hour), DefaultTolerance) {
		t.Error("mutating the timestamp header must invalidate the signature")
	}
}

func TestVerifySignatureRejectsTamperedBody(t *testing.T) {
	secret := "s3cr3t"
	headers := SignatureHeaders([]byte(`{"id":"evt_1"}`), secret, fixedTime)
	if VerifySignature([]byte(`{"id":"evil"}`), headers[SignatureHeader], headers[TimestampHeader], secret, fixedTime, DefaultTolerance) {
		t.Error("a tampered body must be rejected")
	}
}
