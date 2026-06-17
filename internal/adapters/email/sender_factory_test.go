package email

import "testing"

func TestSenderFactoryType(t *testing.T) {
	if NewSenderFactory().ChannelType() != "email" {
		t.Fatalf("unexpected channel type")
	}
}

func TestSenderFactoryRequiresFromEmail(t *testing.T) {
	if _, err := NewSenderFactory().New(map[string]string{}); err == nil {
		t.Fatal("expected error for missing from_email")
	}
}
