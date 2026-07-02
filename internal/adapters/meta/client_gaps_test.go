package meta

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestClient_UnsubscribeFromWebhook_Success(t *testing.T) {
	var capturedPath, capturedMethod string
	c, server := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		capturedPath = r.URL.Path
		capturedMethod = r.Method
		_, _ = w.Write([]byte(`{"success":true}`))
	})
	defer server.Close()

	require.NoError(t, c.UnsubscribeFromWebhook(context.Background(), "page-123"))
	assert.Equal(t, http.MethodDelete, capturedMethod)
	assert.Contains(t, capturedPath, "/page-123/subscribed_apps")
}

func TestClient_UnsubscribeFromWebhook_APIError(t *testing.T) {
	c, server := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte(`{"error":{"message":"insufficient permission","code":200}}`))
	})
	defer server.Close()

	err := c.UnsubscribeFromWebhook(context.Background(), "page-123")
	require.Error(t, err)
}

// -----------------------------------------------------------------------------
// Error-path coverage gaps: OAuth token exchange + long-lived token
// -----------------------------------------------------------------------------

func TestClient_ExchangeCodeForToken_ErrorSurfacedInPayload(t *testing.T) {
	// Meta signals OAuth failures via an `error` body (often with an HTTP error
	// status). The client must fail closed: return a Go-level error instead of a
	// (response, nil) carrying an empty access token that would then be persisted.
	c, server := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"error":{"message":"invalid code","code":100,"type":"OAuthException"}}`))
	})
	defer server.Close()

	resp, err := c.ExchangeCodeForToken(context.Background(), "app", "secret", "https://cb", "bad-code")
	require.Error(t, err)
	assert.Nil(t, resp, "failed exchange must not return a usable (empty-token) response")
	assert.Contains(t, err.Error(), "invalid code")
}

func TestClient_GetLongLivedToken_ReturnsEmptyOnError(t *testing.T) {
	// Mirror of the ExchangeCodeForToken fix: an error response must surface as a
	// Go-level error rather than an empty token.
	c, server := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"error":{"message":"internal"}}`))
	})
	defer server.Close()

	resp, err := c.GetLongLivedToken(context.Background(), "app", "secret", "short-lived-token")
	require.Error(t, err)
	assert.Nil(t, resp)
}

// -----------------------------------------------------------------------------
// Page/Instagram discovery error paths
// -----------------------------------------------------------------------------

func TestClient_GetPages_APIError(t *testing.T) {
	c, server := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"error":{"message":"token expired"}}`))
	})
	defer server.Close()

	_, err := c.GetMyPages(context.Background())
	require.Error(t, err)
}

func TestClient_GetUserProfile_APIError(t *testing.T) {
	c, server := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"error":{"message":"user not found"}}`))
	})
	defer server.Close()

	_, err := c.GetUserProfile(context.Background(), "u-404", []string{"id", "name"})
	require.Error(t, err)
}

// Placeholder to ensure the rewrite transport helper compiles across the
// test file suite (doesn't actually exercise anything but prevents stale
// imports if a future refactor drops httptest from this file).
var _ = httptest.NewServer
