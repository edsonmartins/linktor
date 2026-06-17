package whatsapp_official

import (
	"context"
	"errors"
	"testing"

	"github.com/msgfy/linktor/internal/outbound"
)

func TestClassifyError(t *testing.T) {
	cases := []struct {
		name      string
		err       error
		permanent bool
	}{
		{"4xx invalid template", &APIRequestError{StatusCode: 400}, true},
		{"401 auth", &APIRequestError{StatusCode: 401}, true},
		{"429 rate limit", &APIRequestError{StatusCode: 429}, false},
		{"500 server", &APIRequestError{StatusCode: 500}, false},
		{"network", errors.New("request failed"), false},
		{"already permanent", outbound.Permanentf("bad payload"), true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := outbound.IsPermanent(classifyError(c.err)); got != c.permanent {
				t.Fatalf("classifyError(%v) permanent=%v, want %v", c.err, got, c.permanent)
			}
		})
	}
}

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
