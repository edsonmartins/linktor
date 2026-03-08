package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/msgfy/linktor/pkg/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupOAuthTest(t *testing.T) (*OAuthHandler, *testutil.MockChannelRepository) {
	t.Helper()
	gin.SetMode(gin.TestMode)

	channelRepo := testutil.NewMockChannelRepository()
	handler := NewOAuthHandler(channelRepo, "https://api.example.com/")

	return handler, channelRepo
}

func TestGenerateState(t *testing.T) {
	state := generateState()
	assert.NotEmpty(t, state)
	assert.Len(t, state, 64) // 32 bytes = 64 hex chars
}

func TestEncodeDecodeState(t *testing.T) {
	original := &OAuthState{
		TenantID:    "tenant-1",
		UserID:      "user-1",
		ChannelType: "facebook",
		RedirectURL: "https://example.com/callback",
		Timestamp:   1234567890,
	}

	encoded, err := encodeState(original)
	require.NoError(t, err)
	assert.NotEmpty(t, encoded)

	decoded, err := decodeState(encoded)
	require.NoError(t, err)
	assert.Equal(t, original.TenantID, decoded.TenantID)
	assert.Equal(t, original.UserID, decoded.UserID)
	assert.Equal(t, original.ChannelType, decoded.ChannelType)
	assert.Equal(t, original.RedirectURL, decoded.RedirectURL)
	assert.Equal(t, original.Timestamp, decoded.Timestamp)
}

func TestDecodeState_Invalid(t *testing.T) {
	// Invalid hex string
	_, err := decodeState("not-valid-hex!!!")
	assert.Error(t, err)

	// Valid hex but not valid JSON
	_, err = decodeState("deadbeef")
	assert.Error(t, err)
}

func TestNewOAuthHandler(t *testing.T) {
	channelRepo := testutil.NewMockChannelRepository()

	t.Run("trims trailing slash", func(t *testing.T) {
		handler := NewOAuthHandler(channelRepo, "https://api.example.com/")
		assert.Equal(t, "https://api.example.com", handler.baseURL)
	})

	t.Run("no trailing slash unchanged", func(t *testing.T) {
		handler := NewOAuthHandler(channelRepo, "https://api.example.com")
		assert.Equal(t, "https://api.example.com", handler.baseURL)
	})
}

func TestOAuthHandler_FacebookWebhookCallback(t *testing.T) {
	handler, _ := setupOAuthTest(t)

	t.Run("verify token matches", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest(http.MethodGet,
			"/oauth/facebook/webhook-callback?hub.mode=subscribe&hub.verify_token=mytoken&hub.challenge=challenge123&expected_token=mytoken",
			nil)

		handler.FacebookWebhookCallback(c)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Equal(t, "challenge123", w.Body.String())
	})

	t.Run("verify token does not match", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest(http.MethodGet,
			"/oauth/facebook/webhook-callback?hub.mode=subscribe&hub.verify_token=wrong&hub.challenge=challenge123&expected_token=mytoken",
			nil)

		handler.FacebookWebhookCallback(c)

		assert.Equal(t, http.StatusForbidden, w.Code)
	})

	t.Run("wrong mode", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest(http.MethodGet,
			"/oauth/facebook/webhook-callback?hub.mode=unsubscribe&hub.verify_token=mytoken&hub.challenge=challenge123&expected_token=mytoken",
			nil)

		handler.FacebookWebhookCallback(c)

		assert.Equal(t, http.StatusForbidden, w.Code)
	})
}

func TestOAuthHandler_FacebookLogin_InvalidBody(t *testing.T) {
	handler, _ := setupOAuthTest(t)

	// Missing required fields
	body := map[string]string{"redirect_url": "https://example.com"}
	w, c := newTestContext(http.MethodPost, "/oauth/facebook/login", body)
	c.Set("tenant_id", "tenant-1")
	c.Set("user_id", "user-1")

	handler.FacebookLogin(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var resp map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.NotEmpty(t, resp["error"])
}

func TestOAuthHandler_InstagramLogin_InvalidBody(t *testing.T) {
	handler, _ := setupOAuthTest(t)

	// Missing required fields
	body := map[string]string{"redirect_url": "https://example.com"}
	w, c := newTestContext(http.MethodPost, "/oauth/instagram/login", body)
	c.Set("tenant_id", "tenant-1")
	c.Set("user_id", "user-1")

	handler.InstagramLogin(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var resp map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.NotEmpty(t, resp["error"])
}

func TestOAuthHandler_CreateChannel_InvalidBody(t *testing.T) {
	handler, _ := setupOAuthTest(t)

	// Missing required fields
	body := map[string]string{"page_id": "123"}
	w, c := newTestContext(http.MethodPost, "/oauth/channels", body)
	c.Set("tenant_id", "tenant-1")

	handler.CreateChannel(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var resp map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.NotEmpty(t, resp["error"])
}

func TestOAuthHandler_CreateChannel_InvalidType(t *testing.T) {
	handler, _ := setupOAuthTest(t)

	body := OAuthCreateChannelRequest{
		Name:        "Test Channel",
		Type:        "telegram", // invalid - only facebook/instagram supported
		AccessToken: "token123",
		AppID:       "app-1",
		AppSecret:   "secret-1",
	}
	w, c := newTestContext(http.MethodPost, "/oauth/channels", body)
	c.Set("tenant_id", "tenant-1")

	handler.CreateChannel(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var resp map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Equal(t, "invalid channel type", resp["error"])
}

func TestOAuthHandler_RefreshToken_InvalidBody(t *testing.T) {
	handler, _ := setupOAuthTest(t)

	// Missing required fields
	body := map[string]string{"extra": "field"}
	w, c := newTestContext(http.MethodPost, "/oauth/refresh", body)
	c.Set("tenant_id", "tenant-1")

	handler.RefreshToken(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var resp map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.NotEmpty(t, resp["error"])
}
