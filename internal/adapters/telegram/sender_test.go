package telegram

import (
	"errors"
	"testing"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/msgfy/linktor/internal/outbound"
)

func TestClassifyError(t *testing.T) {
	cases := []struct {
		name      string
		err       error
		permanent bool
	}{
		{"400 bad request", &tgbotapi.Error{Code: 400, Message: "bad request"}, true},
		{"403 blocked", &tgbotapi.Error{Code: 403, Message: "bot blocked"}, true},
		{"429 rate limit", &tgbotapi.Error{Code: 429, Message: "too many"}, false},
		{"500 server", &tgbotapi.Error{Code: 500, Message: "internal"}, false},
		{"network", errors.New("dial tcp"), false},
		{"already permanent", outbound.Permanentf("bad chat"), true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := outbound.IsPermanent(classifyError(c.err)); got != c.permanent {
				t.Fatalf("classifyError(%v) permanent=%v, want %v", c.err, got, c.permanent)
			}
		})
	}
}

func TestSenderFactoryRequiresBotToken(t *testing.T) {
	f := NewSenderFactory()
	if f.ChannelType() != "telegram" {
		t.Fatalf("unexpected channel type %q", f.ChannelType())
	}
	// Missing bot_token must fail without any network call.
	if _, err := f.New(map[string]string{}); err == nil {
		t.Fatal("expected error when bot_token is missing")
	}
}
