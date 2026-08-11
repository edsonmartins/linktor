package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestIsSameOrigin(t *testing.T) {
	cases := []struct {
		name   string
		origin string
		host   string
		want   bool
	}{
		{"same host and port", "http://linktor.local", "linktor.local", true},
		{"same host on a non-default port", "http://192.168.0.10:8080", "192.168.0.10:8080", true},
		// The proxy terminates TLS and reaches us over plain HTTP, so only the
		// host may be compared.
		{"https page behind a TLS-terminating proxy", "https://linktor.local", "linktor.local", true},
		{"different port is a different origin", "http://linktor.local:3000", "linktor.local", false},
		{"subdomain deploy stays on the allowlist", "https://app.linktor.dev", "api.linktor.dev", false},
		{"third-party page", "https://evil.example", "linktor.local", false},
		{"empty origin", "", "linktor.local", false},
		{"garbage origin", "://nope", "linktor.local", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			r := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", nil)
			r.Host = tc.host
			if got := IsSameOrigin(tc.origin, r); got != tc.want {
				t.Fatalf("IsSameOrigin(%q, host=%q) = %v, want %v", tc.origin, tc.host, got, tc.want)
			}
		})
	}
}

// runCORS sends a POST carrying the given Origin/Host through the middleware
// and reports the resulting status. The handler behind it always returns 200,
// so a 403 can only come from the origin check.
func runCORS(t *testing.T, origin, host string) int {
	t.Helper()

	gin.SetMode(gin.ReleaseMode)
	router := gin.New()
	router.Use(CORS())
	router.POST("/api/v1/auth/login", func(c *gin.Context) { c.Status(http.StatusOK) })

	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", nil)
	req.Host = host
	if origin != "" {
		req.Header.Set("Origin", origin)
	}
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	return w.Code
}

// A single-origin deploy serves the admin and the API from one host, so the
// browser attaches Origin to same-origin writes. Rejecting those would pin the
// install to whatever names happen to be in LINKTOR_CORS_ALLOWED_ORIGINS.
func TestCORSAllowsSameOriginWithoutAllowlist(t *testing.T) {
	t.Setenv("LINKTOR_CORS_ALLOWED_ORIGINS", "")

	if code := runCORS(t, "http://linktor.local", "linktor.local"); code != http.StatusOK {
		t.Fatalf("same-origin request = %d, want %d", code, http.StatusOK)
	}
}

func TestCORSDeniesCrossOriginWithoutAllowlist(t *testing.T) {
	t.Setenv("LINKTOR_CORS_ALLOWED_ORIGINS", "")

	if code := runCORS(t, "https://evil.example", "linktor.local"); code != http.StatusForbidden {
		t.Fatalf("cross-origin request = %d, want %d", code, http.StatusForbidden)
	}
}

func TestCORSStillHonoursAllowlist(t *testing.T) {
	t.Setenv("LINKTOR_CORS_ALLOWED_ORIGINS", "https://app.linktor.dev")

	if code := runCORS(t, "https://app.linktor.dev", "api.linktor.dev"); code != http.StatusOK {
		t.Fatalf("allowlisted origin = %d, want %d", code, http.StatusOK)
	}
	if code := runCORS(t, "https://other.linktor.dev", "api.linktor.dev"); code != http.StatusForbidden {
		t.Fatalf("unlisted origin = %d, want %d", code, http.StatusForbidden)
	}
}
