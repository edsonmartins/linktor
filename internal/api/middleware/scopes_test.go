package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestHasScope(t *testing.T) {
	cases := []struct {
		name     string
		granted  []string
		required string
		want     bool
	}{
		{"star grants anything", []string{"*"}, "channels:write", true},
		{"exact match", []string{"channels:write"}, "channels:write", true},
		{"resource wildcard grants any action", []string{"channels:*"}, "channels:write", true},
		{"write implies read", []string{"channels:write"}, "channels:read", true},
		{"read does NOT imply write", []string{"channels:read"}, "channels:write", false},
		{"unrelated resource", []string{"messages:send"}, "channels:read", false},
		{"empty grants nothing", nil, "channels:read", false},
		{"one of several matches", []string{"messages:send", "channels:read"}, "channels:read", true},
		{"star among others", []string{"messages:send", "*"}, "contacts:write", true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := HasScope(tc.granted, tc.required); got != tc.want {
				t.Fatalf("HasScope(%v, %q) = %v, want %v", tc.granted, tc.required, got, tc.want)
			}
		})
	}
}

// runScope builds a request through RequireScope with the given granted scopes,
// or none set at all when setScopes is false (simulating a human/JWT caller).
func runScope(t *testing.T, setScopes bool, granted []string, required string) int {
	t.Helper()
	gin.SetMode(gin.TestMode)
	m := &AuthMiddleware{}
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/channels", nil)
	if setScopes {
		c.Set(ScopesKey, granted)
	}
	reached := false
	handlers := []gin.HandlerFunc{
		m.RequireScope(required),
		func(c *gin.Context) { reached = true; c.Status(http.StatusOK) },
	}
	for _, h := range handlers {
		if c.IsAborted() {
			break
		}
		h(c)
	}
	if reached && w.Code == 0 {
		w.WriteHeader(http.StatusOK)
	}
	return w.Code
}

func TestRequireScope_JWTIsNoOp(t *testing.T) {
	// No ScopesKey set → human/JWT caller → scope check must not fire.
	if code := runScope(t, false, nil, "channels:write"); code != http.StatusOK {
		t.Fatalf("JWT caller should pass RequireScope, got %d", code)
	}
}

func TestRequireScope_APIKeyGranted(t *testing.T) {
	if code := runScope(t, true, []string{"channels:write"}, "channels:write"); code != http.StatusOK {
		t.Fatalf("granted key should pass, got %d", code)
	}
}

func TestRequireScope_APIKeyDenied(t *testing.T) {
	if code := runScope(t, true, []string{"channels:read"}, "channels:write"); code != http.StatusForbidden {
		t.Fatalf("key missing scope should get 403, got %d", code)
	}
}

func TestRequireScopeByMethod(t *testing.T) {
	gin.SetMode(gin.TestMode)
	m := &AuthMiddleware{}
	cases := []struct {
		method  string
		granted []string
		want    int
	}{
		{http.MethodGet, []string{"channels:read"}, http.StatusOK},
		{http.MethodGet, []string{"channels:write"}, http.StatusOK}, // write implies read
		{http.MethodPost, []string{"channels:read"}, http.StatusForbidden},
		{http.MethodPost, []string{"channels:write"}, http.StatusOK},
		{http.MethodDelete, []string{"channels:read"}, http.StatusForbidden},
	}
	for _, tc := range cases {
		t.Run(tc.method+"_"+tc.granted[0], func(t *testing.T) {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request = httptest.NewRequest(tc.method, "/channels", nil)
			c.Set(ScopesKey, tc.granted)
			m.RequireScopeByMethod(ScopeChannelsRead, ScopeChannelsWrite)(c)
			if !c.IsAborted() {
				c.Status(http.StatusOK)
			}
			if w.Code != tc.want {
				t.Fatalf("%s with %v: got %d, want %d", tc.method, tc.granted, w.Code, tc.want)
			}
		})
	}
}
