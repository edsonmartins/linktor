package whatsapp

import (
	"testing"

	"go.mau.fi/whatsmeow/types"
)

func TestResolveRecipientJID(t *testing.T) {
	tests := []struct {
		name       string
		in         string
		wantUser   string
		wantServer string
	}{
		{
			name:       "bare phone number gets default user server",
			in:         "554488122990",
			wantUser:   "554488122990",
			wantServer: types.DefaultUserServer,
		},
		{
			name:       "full user JID is preserved",
			in:         "554488122990@s.whatsapp.net",
			wantUser:   "554488122990",
			wantServer: types.DefaultUserServer,
		},
		{
			name:       "group JID is preserved",
			in:         "123456789-987654321@g.us",
			wantUser:   "123456789-987654321",
			wantServer: types.GroupServer,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			jid, err := resolveRecipientJID(tt.in)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if jid.User != tt.wantUser {
				t.Errorf("User = %q, want %q", jid.User, tt.wantUser)
			}
			if jid.Server != tt.wantServer {
				t.Errorf("Server = %q, want %q", jid.Server, tt.wantServer)
			}
		})
	}
}

// TestResolveRecipientJIDRegression guards the exact production failure: a bare
// number must NOT end up with the number as the JID server (which yields
// whatsmeow's "can't send message to unknown server" error).
func TestResolveRecipientJIDRegression(t *testing.T) {
	jid, err := resolveRecipientJID("554488122990")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if jid.Server == "554488122990" {
		t.Fatalf("regression: phone number leaked into JID server field")
	}
	if jid.Server != types.DefaultUserServer {
		t.Fatalf("Server = %q, want %q", jid.Server, types.DefaultUserServer)
	}
}
