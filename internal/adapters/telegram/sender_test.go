package telegram

import "testing"

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
