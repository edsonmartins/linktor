package whatsapp_official

import (
	"context"
	"testing"

	"github.com/msgfy/linktor/internal/outbound"
)

func TestSenderFactoryRequiresCredentials(t *testing.T) {
	f := NewSenderFactory()
	if f.ChannelType() != "whatsapp_official" {
		t.Fatalf("unexpected channel type %q", f.ChannelType())
	}

	if _, err := f.New(map[string]string{"phone_number_id": "1"}); err == nil {
		t.Fatal("expected error when access_token is missing")
	}
	if _, err := f.New(map[string]string{"access_token": "x"}); err == nil {
		t.Fatal("expected error when phone_number_id is missing")
	}

	s, err := f.New(map[string]string{"access_token": "x", "phone_number_id": "1"})
	if err != nil || s == nil {
		t.Fatalf("expected a sender, got %v / %v", s, err)
	}
}

func TestSenderPermanentOnInvalidPayload(t *testing.T) {
	f := NewSenderFactory()
	s, _ := f.New(map[string]string{"access_token": "x", "phone_number_id": "1"})

	// Empty recipient must fail permanently (no network call).
	if _, err := s.Send(context.Background(), &outbound.Message{Content: outbound.Text{Body: "hi"}}); !outbound.IsPermanent(err) {
		t.Fatalf("expected permanent error for empty recipient, got %v", err)
	}

	// Template without a name must fail permanently (no network call).
	_, err := s.Send(context.Background(), &outbound.Message{To: "+55", Content: outbound.Template{}})
	if !outbound.IsPermanent(err) {
		t.Fatalf("expected permanent error for nameless template, got %v", err)
	}
}
