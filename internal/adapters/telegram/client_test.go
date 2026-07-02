package telegram

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newTestClient builds a Client without hitting the network (bypassing
// NewClient, which calls getMe). Tests point apiBaseURL/httpClient at an
// httptest server.
func newTestClient(baseURL string) *Client {
	return &Client{
		botToken:         "TESTTOKEN",
		httpClient:       &http.Client{},
		apiBaseURL:       baseURL,
		maxDownloadBytes: defaultMaxDownloadBytes,
	}
}

// --- Bug 1: secret_token must be registered with setWebhook ---

func TestBuildSetWebhookRequest_IncludesSecretToken(t *testing.T) {
	c := newTestClient("https://api.telegram.org")
	c.SetSecretToken("my-secret")

	req, err := c.buildSetWebhookRequest(context.Background(), "https://example.com/hook")
	require.NoError(t, err)

	body, err := readAll(req)
	require.NoError(t, err)
	assert.Contains(t, body, "secret_token=my-secret")
	assert.Contains(t, body, "url=https%3A%2F%2Fexample.com%2Fhook")
}

func TestBuildSetWebhookRequest_NoSecretTokenWhenUnset(t *testing.T) {
	c := newTestClient("https://api.telegram.org")

	req, err := c.buildSetWebhookRequest(context.Background(), "https://example.com/hook")
	require.NoError(t, err)

	body, err := readAll(req)
	require.NoError(t, err)
	assert.NotContains(t, body, "secret_token")
}

func TestSetWebhook_SendsSecretTokenToTelegram(t *testing.T) {
	var gotSecret, gotURL string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.True(t, strings.HasSuffix(r.URL.Path, "/setWebhook"))
		require.NoError(t, r.ParseForm())
		gotSecret = r.FormValue("secret_token")
		gotURL = r.FormValue("url")
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"ok":true,"result":true}`))
	}))
	defer srv.Close()

	c := newTestClient(srv.URL)
	c.httpClient = srv.Client()
	c.SetSecretToken("s3cr3t")

	err := c.SetWebhook("https://example.com/telegram")
	require.NoError(t, err)
	assert.Equal(t, "s3cr3t", gotSecret)
	assert.Equal(t, "https://example.com/telegram", gotURL)
}

func TestSetWebhook_ReturnsErrorOnAPIFailure(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"ok":false,"description":"bad token"}`))
	}))
	defer srv.Close()

	c := newTestClient(srv.URL)
	c.httpClient = srv.Client()

	err := c.SetWebhook("https://example.com/telegram")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "bad token")
}

// --- Bug 2: DownloadFile must be bounded and honor context ---

func TestDownloadURL_RespectsSizeLimit(t *testing.T) {
	// Server returns 100 bytes but the client cap is 10 -> must error.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(strings.Repeat("x", 100)))
	}))
	defer srv.Close()

	c := newTestClient(srv.URL)
	c.httpClient = srv.Client()
	c.maxDownloadBytes = 10

	_, err := c.downloadURL(context.Background(), srv.URL+"/file")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "maximum download size")
}

func TestDownloadURL_WithinLimitSucceeds(t *testing.T) {
	payload := strings.Repeat("y", 50)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(payload))
	}))
	defer srv.Close()

	c := newTestClient(srv.URL)
	c.httpClient = srv.Client()
	c.maxDownloadBytes = 1000

	data, err := c.downloadURL(context.Background(), srv.URL+"/file")
	require.NoError(t, err)
	assert.Equal(t, payload, string(data))
}

func TestDownloadURL_HonorsCanceledContext(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("data"))
	}))
	defer srv.Close()

	c := newTestClient(srv.URL)
	c.httpClient = srv.Client()

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // canceled before use

	_, err := c.downloadURL(ctx, srv.URL+"/file")
	require.Error(t, err)
	assert.ErrorIs(t, err, context.Canceled)
}

func TestDownloadURL_Non200Status(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	c := newTestClient(srv.URL)
	c.httpClient = srv.Client()

	_, err := c.downloadURL(context.Background(), srv.URL+"/file")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "status 404")
}

// readAll drains a request body into a string.
func readAll(req *http.Request) (string, error) {
	if req.Body == nil {
		return "", nil
	}
	defer req.Body.Close()
	b, err := io.ReadAll(req.Body)
	return string(b), err
}
