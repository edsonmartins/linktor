package webchat

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/msgfy/linktor/internal/domain/entity"
	"github.com/msgfy/linktor/pkg/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWidgetToken_RoundTrip(t *testing.T) {
	now := time.Unix(1_700_000_000, 0)
	secret := "s3cr3t-per-channel"
	channelID := "chan-123"

	token, err := IssueWidgetToken(channelID, secret, 30*time.Minute, now)
	require.NoError(t, err)
	require.NotEmpty(t, token)

	// Valid within TTL.
	require.NoError(t, VerifyWidgetToken(token, channelID, secret, now.Add(10*time.Minute)))
}

func TestWidgetToken_Expired(t *testing.T) {
	now := time.Unix(1_700_000_000, 0)
	secret := "s3cr3t"
	channelID := "chan-abc"

	token, err := IssueWidgetToken(channelID, secret, 30*time.Minute, now)
	require.NoError(t, err)

	// Verify just after expiry.
	err = VerifyWidgetToken(token, channelID, secret, now.Add(31*time.Minute))
	assert.ErrorIs(t, err, ErrTokenExpired)
}

func TestWidgetToken_Tampered(t *testing.T) {
	now := time.Unix(1_700_000_000, 0)
	secret := "s3cr3t"
	channelID := "chan-xyz"

	token, err := IssueWidgetToken(channelID, secret, 30*time.Minute, now)
	require.NoError(t, err)

	// Flip a character in the signature segment.
	tampered := token[:len(token)-1]
	if token[len(token)-1] == 'A' {
		tampered += "B"
	} else {
		tampered += "A"
	}
	err = VerifyWidgetToken(tampered, channelID, secret, now)
	assert.Error(t, err)

	// Wrong secret must also fail (constant-time compare).
	err = VerifyWidgetToken(token, channelID, "other-secret", now)
	assert.ErrorIs(t, err, ErrTokenSignature)
}

func TestWidgetToken_ChannelMismatch(t *testing.T) {
	now := time.Unix(1_700_000_000, 0)
	secret := "s3cr3t"

	token, err := IssueWidgetToken("chan-A", secret, 30*time.Minute, now)
	require.NoError(t, err)

	err = VerifyWidgetToken(token, "chan-B", secret, now)
	assert.ErrorIs(t, err, ErrTokenChannelMismatch)
}

func TestWidgetToken_Malformed(t *testing.T) {
	assert.ErrorIs(t, VerifyWidgetToken("not-a-token", "c", "s", time.Now()), ErrTokenMalformed)
	assert.ErrorIs(t, VerifyWidgetToken("a.b.c", "c", "s", time.Now()), ErrTokenMalformed)
	assert.ErrorIs(t, VerifyWidgetToken("", "c", "s", time.Now()), ErrTokenMalformed)
}

func TestWidgetToken_MissingSecret(t *testing.T) {
	_, err := IssueWidgetToken("c", "", 0, time.Now())
	assert.ErrorIs(t, err, ErrTokenMissingSecret)
	assert.ErrorIs(t, VerifyWidgetToken("x.y", "c", "", time.Now()), ErrTokenMissingSecret)
}

func TestWidgetToken_WithNonce(t *testing.T) {
	now := time.Unix(1_700_000_000, 0)
	token, err := IssueWidgetTokenWithNonce("chan", "secret", "session-nonce", time.Hour, now)
	require.NoError(t, err)
	require.NoError(t, VerifyWidgetToken(token, "chan", "secret", now))
}

func TestIPRateLimiter(t *testing.T) {
	now := time.Unix(1_700_000_000, 0)
	rl := newIPRateLimiter(1, 3) // burst 3, 1 token/sec

	// First 3 attempts allowed within the same instant.
	assert.True(t, rl.Allow("1.2.3.4", now))
	assert.True(t, rl.Allow("1.2.3.4", now))
	assert.True(t, rl.Allow("1.2.3.4", now))
	// 4th is throttled.
	assert.False(t, rl.Allow("1.2.3.4", now))

	// A different IP has its own bucket.
	assert.True(t, rl.Allow("5.6.7.8", now))

	// After 2 seconds, ~2 tokens refilled.
	assert.True(t, rl.Allow("1.2.3.4", now.Add(2*time.Second)))
	assert.True(t, rl.Allow("1.2.3.4", now.Add(2*time.Second)))
	assert.False(t, rl.Allow("1.2.3.4", now.Add(2*time.Second)))
}

func TestWidgetSecret_Resolution(t *testing.T) {
	assert.Equal(t, "", widgetSecret(nil))

	ch := &entity.Channel{Config: map[string]string{"widget_secret": "cfg"}}
	assert.Equal(t, "cfg", widgetSecret(ch))

	ch = &entity.Channel{Credentials: map[string]string{"widget_secret": "cred"}}
	assert.Equal(t, "cred", widgetSecret(ch))

	ch = &entity.Channel{
		Config:      map[string]string{"widget_secret": "cfg"},
		Credentials: map[string]string{"widget_secret": "cred"},
	}
	assert.Equal(t, "cfg", widgetSecret(ch)) // Config wins.
}

// TestWebSocketHandler_RejectsMissingToken verifies that a channel configured
// with a widget_secret rejects a WS upgrade request that lacks a valid token,
// before any upgrade is attempted.
func TestWebSocketHandler_RejectsMissingToken(t *testing.T) {
	gin.SetMode(gin.TestMode)

	channelRepo := testutil.NewMockChannelRepository()
	channelRepo.Channels["chan-secured"] = &entity.Channel{
		ID:               "chan-secured",
		TenantID:         "tenant-1",
		Type:             entity.ChannelTypeWebChat,
		Enabled:          true,
		ConnectionStatus: entity.ConnectionStatusConnected,
		Config:           map[string]string{"widget_secret": "top-secret"},
	}

	h := NewHandler(NewAdapter(), channelRepo, nil, nil, nil)

	r := gin.New()
	r.GET("/ws/:channelId", h.WebSocketHandler)

	// No token -> 401.
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/ws/chan-secured", nil)
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusUnauthorized, w.Code)

	// Invalid token -> 401.
	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/ws/chan-secured?token=garbage", nil)
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusUnauthorized, w.Code)

	// Valid token in header passes auth (upgrade then fails on the recorder,
	// which cannot hijack, so we only assert it is NOT a 401).
	token, err := IssueWidgetToken("chan-secured", "top-secret", 30*time.Minute, time.Now())
	require.NoError(t, err)
	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/ws/chan-secured", nil)
	req.Header.Set("X-Widget-Token", token)
	r.ServeHTTP(w, req)
	assert.NotEqual(t, http.StatusUnauthorized, w.Code)
}
