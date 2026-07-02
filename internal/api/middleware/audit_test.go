package middleware

import "testing"

func TestDeriveAuditFromPath(t *testing.T) {
	cases := []struct {
		method, path, id string
		wantRes, wantAct string
		wantID           string
	}{
		{"POST", "/api/v1/channels", "", "channel", "channel.create", ""},
		{"PUT", "/api/v1/channels/:id", "c1", "channel", "channel.update", "c1"},
		{"PATCH", "/api/v1/channels/:id", "c1", "channel", "channel.update", "c1"},
		{"DELETE", "/api/v1/contacts/:id", "x9", "contact", "contact.delete", "x9"},
		{"POST", "/api/v1/channels/:id/connect", "c1", "channel", "channel.connect", "c1"},
		{"POST", "/api/v1/channels/:id/disconnect", "c1", "channel", "channel.disconnect", "c1"},
		{"PUT", "/api/v1/tenant", "", "tenant", "tenant.update", ""},
		{"POST", "/api/v1/users", "", "user", "user.create", ""},
	}
	for _, c := range cases {
		res, id, act := deriveAuditFromPath(c.method, c.path, c.id)
		if res != c.wantRes || act != c.wantAct || id != c.wantID {
			t.Errorf("%s %s => (%q,%q,%q), want (%q,%q,%q)",
				c.method, c.path, res, id, act, c.wantRes, c.wantID, c.wantAct)
		}
	}
}

func TestSingular(t *testing.T) {
	cases := map[string]string{
		"channels": "channel",
		"contacts": "contact",
		"users":    "user",
		"entities": "entity",
		"tenant":   "tenant",
	}
	for in, want := range cases {
		if got := singular(in); got != want {
			t.Errorf("singular(%q)=%q, want %q", in, got, want)
		}
	}
}
