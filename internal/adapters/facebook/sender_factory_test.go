package facebook

import "testing"

func TestSenderFactoryType(t *testing.T) {
	if NewSenderFactory().ChannelType() != "facebook" {
		t.Fatalf("unexpected channel type")
	}
}

func TestSenderFactoryRequiresCreds(t *testing.T) {
	// Empty creds must fail before any network call.
	if _, err := NewSenderFactory().New(map[string]string{}); err == nil {
		t.Fatal("expected error for missing credentials")
	}
}

// Channels created through /oauth/channels store the page token as access_token.
func TestSenderFactoryAcceptsOAuthAccessToken(t *testing.T) {
	if _, err := NewSenderFactory().New(map[string]string{"access_token": "page-token", "page_id": "123"}); err != nil {
		t.Fatalf("expected access_token to be accepted, got %v", err)
	}
}
