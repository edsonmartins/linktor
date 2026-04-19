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
	// Meta's OAuth endpoint returns the error shape in the response body
	// rather than an HTTP error, and the client currently reflects that —
	// it returns (response, nil) with tokenResp.Error populated. Verifying
	// this shape pins the current behaviour so a future refactor doesn't
	// silently swallow failed code exchanges.
	c, server := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"error":{"message":"invalid code","code":100,"type":"OAuthException"}}`))
	})
	defer server.Close()

	resp, err := c.ExchangeCodeForToken(context.Background(), "app", "secret", "https://cb", "bad-code")
	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.Empty(t, resp.AccessToken, "failed exchange must not return a usable token")
	require.NotNil(t, resp.Error)
	assert.Equal(t, "invalid code", resp.Error.Message)
	assert.Equal(t, 100, resp.Error.Code)
}

func TestClient_GetLongLivedToken_ReturnsEmptyOnError(t *testing.T) {
	// Mirror of the ExchangeCodeForToken gap: Meta signals errors in the
	// body; the client returns an empty token rather than a Go-level error.
	c, server := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"error":{"message":"internal"}}`))
	})
	defer server.Close()

	resp, err := c.GetLongLivedToken(context.Background(), "app", "secret", "short-lived-token")
	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.Empty(t, resp.AccessToken)
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
