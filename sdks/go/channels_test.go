package linktor

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/linktor/linktor-go/types"
)

// TestChannelsConnectReturnsQR proves Connect unwraps the {success,data} envelope
// and surfaces the QR payload — the gap that previously made connect() drop the
// QR by deserializing into a bare Channel.
func TestChannelsConnectReturnsQR(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/channels/ch1/connect" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		if got := r.Header.Get("X-API-Key"); got != "lk_test" {
			t.Errorf("expected API key header, got %q", got)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"success":true,"data":{
			"channel":{"id":"ch1","name":"wa","type":"whatsapp","connection_status":"connecting"},
			"qr_code":"QR-PAYLOAD-123","expires_in":60}}`)
	}))
	defer srv.Close()

	c := NewClient(WithBaseURL(srv.URL), WithAPIKey("lk_test"))
	res, err := c.Channels.Connect(context.Background(), "ch1")
	if err != nil {
		t.Fatalf("Connect: %v", err)
	}
	if res.QRCode != "QR-PAYLOAD-123" {
		t.Errorf("QRCode = %q, want QR-PAYLOAD-123", res.QRCode)
	}
	if res.ExpiresIn != 60 {
		t.Errorf("ExpiresIn = %d, want 60", res.ExpiresIn)
	}
	if res.Channel == nil || res.Channel.ID != "ch1" {
		t.Errorf("Channel not populated: %+v", res.Channel)
	}
	if res.Channel != nil && res.Channel.ConnectionStatus != types.ConnectionStatusConnecting {
		t.Errorf("ConnectionStatus = %q, want connecting", res.Channel.ConnectionStatus)
	}
}

// TestChannelsCreateSendsCredentials proves Create serializes the credentials
// map (previously absent from CreateChannelInput) and unwraps the created channel.
func TestChannelsCreateSendsCredentials(t *testing.T) {
	var body map[string]json.RawMessage
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewDecoder(r.Body).Decode(&body)
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"success":true,"data":{"id":"ch9","name":"wa","type":"whatsapp","status":"inactive"}}`)
	}))
	defer srv.Close()

	c := NewClient(WithBaseURL(srv.URL), WithAPIKey("lk_test"))
	ch, err := c.Channels.Create(context.Background(), &types.CreateChannelInput{
		Name:        "wa",
		Type:        types.ChannelTypeWhatsApp,
		Credentials: map[string]string{"access_token": "secret"},
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if ch.ID != "ch9" {
		t.Errorf("ID = %q, want ch9", ch.ID)
	}
	if _, ok := body["credentials"]; !ok {
		t.Errorf("credentials not sent in request body: %v", body)
	}
}

// TestChannelsRequestPairCode proves pairing posts the phone number and returns
// the pair code.
func TestChannelsRequestPairCode(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/channels/ch1/pair" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		var in types.PairCodeInput
		_ = json.NewDecoder(r.Body).Decode(&in)
		if in.PhoneNumber != "+5511999999999" {
			t.Errorf("phone = %q", in.PhoneNumber)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"success":true,"data":{"pair_code":"ABCD-1234"}}`)
	}))
	defer srv.Close()

	c := NewClient(WithBaseURL(srv.URL), WithAPIKey("lk_test"))
	res, err := c.Channels.RequestPairCode(context.Background(), "ch1", "+5511999999999")
	if err != nil {
		t.Fatalf("RequestPairCode: %v", err)
	}
	if res.PairCode != "ABCD-1234" {
		t.Errorf("PairCode = %q, want ABCD-1234", res.PairCode)
	}
}
