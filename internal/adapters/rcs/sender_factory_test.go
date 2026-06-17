package rcs

import "testing"

func TestSenderFactoryType(t *testing.T) {
	if NewSenderFactory().ChannelType() != "rcs" {
		t.Fatalf("unexpected channel type")
	}
}

func TestSenderFactoryRequiresCreds(t *testing.T) {
	// Empty creds must fail before any network call.
	if _, err := NewSenderFactory().New(map[string]string{}); err == nil {
		t.Fatal("expected error for missing credentials")
	}
}
