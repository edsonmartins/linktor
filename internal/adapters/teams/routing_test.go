package teams

import (
	"encoding/base64"
	"testing"

	"github.com/msgfy/linktor/internal/domain/entity"
)

// makeJWT builds an unsigned-looking JWT whose payload is the given JSON.
func makeJWT(payloadJSON string) string {
	enc := func(s string) string { return base64.RawURLEncoding.EncodeToString([]byte(s)) }
	return enc(`{"alg":"RS256"}`) + "." + enc(payloadJSON) + ".sig"
}

func TestUnverifiedAudience(t *testing.T) {
	cases := []struct {
		name   string
		header string
		want   string
	}{
		{"string aud", "Bearer " + makeJWT(`{"aud":"app-123"}`), "app-123"},
		{"array aud", "Bearer " + makeJWT(`{"aud":["app-456","other"]}`), "app-456"},
		{"no bearer prefix", makeJWT(`{"aud":"app-789"}`), "app-789"},
		{"malformed", "Bearer not.a.jwt.token", ""},
		{"empty", "", ""},
	}
	for _, tc := range cases {
		if got := UnverifiedAudience(tc.header); got != tc.want {
			t.Errorf("%s: UnverifiedAudience=%q want %q", tc.name, got, tc.want)
		}
	}
}

func TestMatchSharedChannel(t *testing.T) {
	exact := &entity.Channel{ID: "exact", Credentials: map[string]string{CredAppID: "app", CredTenantID: "org-1"}}
	multi := &entity.Channel{ID: "multi", Credentials: map[string]string{CredAppID: "app", CredTenantID: "common"}}
	otherApp := &entity.Channel{ID: "other", Credentials: map[string]string{CredAppID: "different", CredTenantID: "org-1"}}

	// Exact tenant match wins over a multi-tenant fallback.
	if got := MatchSharedChannel([]*entity.Channel{multi, exact, otherApp}, "app", "org-1"); got == nil || got.ID != "exact" {
		t.Errorf("expected exact org match, got %v", got)
	}

	// Unknown org falls back to the multi-tenant channel of the same app.
	if got := MatchSharedChannel([]*entity.Channel{multi, otherApp}, "app", "org-999"); got == nil || got.ID != "multi" {
		t.Errorf("expected multi-tenant fallback, got %v", got)
	}

	// Wrong app id never matches.
	if got := MatchSharedChannel([]*entity.Channel{otherApp}, "app", "org-1"); got != nil {
		t.Errorf("expected no match for unknown app, got %v", got)
	}

	// No multi-tenant fallback and no exact match → nil.
	single := &entity.Channel{ID: "single", Credentials: map[string]string{CredAppID: "app", CredTenantID: "org-2"}}
	if got := MatchSharedChannel([]*entity.Channel{single}, "app", "org-1"); got != nil {
		t.Errorf("expected nil when only a different single-tenant channel exists, got %v", got)
	}
}

func TestIsTrustedServiceURL(t *testing.T) {
	trusted := []string{
		"https://smba.trafficmanager.net/amer/",
		"https://smba.trafficmanager.net/emea/",
		"https://api.botframework.com",
		"https://europe.botframework.com/v3/",
		"https://something.skype.com/",
	}
	for _, u := range trusted {
		if !IsTrustedServiceURL(u) {
			t.Errorf("expected %q to be trusted", u)
		}
	}

	untrusted := []string{
		"",
		"http://smba.trafficmanager.net/amer/",   // not https
		"https://evil.com/",                      // arbitrary host
		"https://botframework.com.evil.com/",     // suffix-spoof on a different host
		"https://attacker.net/v3/conversations/", // attacker-controlled sink
		"ftp://api.botframework.com",             // wrong scheme
	}
	for _, u := range untrusted {
		if IsTrustedServiceURL(u) {
			t.Errorf("expected %q to be UNtrusted", u)
		}
	}
}
