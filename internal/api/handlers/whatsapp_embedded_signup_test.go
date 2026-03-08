package handlers

import (
	"encoding/hex"
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/msgfy/linktor/pkg/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupEmbeddedSignupTest(t *testing.T) (*WhatsAppEmbeddedSignupHandler, *testutil.MockChannelRepository) {
	t.Helper()
	gin.SetMode(gin.TestMode)

	channelRepo := testutil.NewMockChannelRepository()
	handler := NewWhatsAppEmbeddedSignupHandler(channelRepo, "https://api.example.com/")

	return handler, channelRepo
}

func TestNewWhatsAppEmbeddedSignupHandler(t *testing.T) {
	channelRepo := testutil.NewMockChannelRepository()

	t.Run("https URL trims trailing slash", func(t *testing.T) {
		handler := NewWhatsAppEmbeddedSignupHandler(channelRepo, "https://api.example.com/")
		assert.Equal(t, "https://api.example.com", handler.baseURL)
		assert.NotEmpty(t, handler.stateSecret)
		assert.NotNil(t, handler.httpClient)
		assert.Equal(t, "https://graph.facebook.com/v21.0", handler.graphAPIURL)
	})

	t.Run("http URL with localhost allowed", func(t *testing.T) {
		handler := NewWhatsAppEmbeddedSignupHandler(channelRepo, "http://localhost:8080")
		assert.Equal(t, "http://localhost:8080", handler.baseURL)
	})

	t.Run("http URL non-localhost still works", func(t *testing.T) {
		handler := NewWhatsAppEmbeddedSignupHandler(channelRepo, "http://api.example.com")
		assert.Equal(t, "http://api.example.com", handler.baseURL)
	})
}

func TestGenerateNonce(t *testing.T) {
	nonce, err := generateNonce()
	require.NoError(t, err)
	assert.NotEmpty(t, nonce)
	assert.Len(t, nonce, 32) // 16 bytes = 32 hex chars
}

func TestEncodeDecodeEmbeddedState(t *testing.T) {
	handler, _ := setupEmbeddedSignupTest(t)

	original := &EmbeddedSignupState{
		TenantID:    "tenant-1",
		UserID:      "user-1",
		RedirectURL: "https://example.com/callback",
		Timestamp:   time.Now().Unix(),
		Nonce:       "abc123",
	}

	encoded, err := handler.encodeEmbeddedState(original)
	require.NoError(t, err)
	assert.NotEmpty(t, encoded)

	decoded, err := handler.decodeEmbeddedState(encoded)
	require.NoError(t, err)
	assert.Equal(t, original.TenantID, decoded.TenantID)
	assert.Equal(t, original.UserID, decoded.UserID)
	assert.Equal(t, original.RedirectURL, decoded.RedirectURL)
	assert.Equal(t, original.Timestamp, decoded.Timestamp)
	assert.Equal(t, original.Nonce, decoded.Nonce)
}

func TestDecodeEmbeddedState_Invalid(t *testing.T) {
	handler, _ := setupEmbeddedSignupTest(t)

	t.Run("invalid hex", func(t *testing.T) {
		_, err := handler.decodeEmbeddedState("not-valid-hex!!!")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid state encoding")
	})

	t.Run("too short", func(t *testing.T) {
		// Less than 32 bytes (64 hex chars for signature alone)
		shortData := hex.EncodeToString([]byte("short"))
		_, err := handler.decodeEmbeddedState(shortData)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "state too short")
	})

	t.Run("wrong signature", func(t *testing.T) {
		// Create valid-length data with wrong signature
		fakeData := []byte(`{"tenant_id":"t1","user_id":"u1","timestamp":1234567890,"nonce":"abc"}`)
		fakeSignature := make([]byte, 32)
		combined := append(fakeData, fakeSignature...)
		encoded := hex.EncodeToString(combined)

		_, err := handler.decodeEmbeddedState(encoded)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid state signature")
	})
}

func TestStartEmbeddedSignup_InvalidBody(t *testing.T) {
	handler, _ := setupEmbeddedSignupTest(t)

	// Missing required app_id
	body := map[string]string{"redirect_url": "https://example.com"}
	w, c := newTestContext(http.MethodPost, "/oauth/whatsapp/embedded-signup/start", body)
	c.Set("tenant_id", "tenant-1")
	c.Set("user_id", "user-1")

	handler.StartEmbeddedSignup(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var resp map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.NotEmpty(t, resp["error"])
}

func TestCompleteEmbeddedSignup_InvalidBody(t *testing.T) {
	handler, _ := setupEmbeddedSignupTest(t)

	// Missing required fields
	body := map[string]string{"extra": "field"}
	w, c := newTestContext(http.MethodPost, "/oauth/whatsapp/embedded-signup/callback", body)
	c.Set("tenant_id", "tenant-1")
	c.Set("user_id", "user-1")

	handler.CompleteEmbeddedSignup(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var resp map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.NotEmpty(t, resp["error"])
}

func TestCompleteEmbeddedSignup_InvalidState(t *testing.T) {
	handler, _ := setupEmbeddedSignupTest(t)

	body := EmbeddedSignupCallbackRequest{
		Code:      "auth-code-123",
		State:     "invalidhexstate",
		AppID:     "app-1",
		AppSecret: "secret-1",
	}
	w, c := newTestContext(http.MethodPost, "/oauth/whatsapp/embedded-signup/callback", body)
	c.Set("tenant_id", "tenant-1")
	c.Set("user_id", "user-1")

	handler.CompleteEmbeddedSignup(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var resp map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Contains(t, resp["error"], "invalid state")
}

func TestCompleteEmbeddedSignup_ExpiredState(t *testing.T) {
	handler, _ := setupEmbeddedSignupTest(t)

	// Create a state that expired (timestamp > 10 minutes ago)
	state := &EmbeddedSignupState{
		TenantID:  "tenant-1",
		UserID:    "user-1",
		Timestamp: time.Now().Add(-15 * time.Minute).Unix(), // 15 minutes ago
		Nonce:     "nonce123",
	}
	encodedState, err := handler.encodeEmbeddedState(state)
	require.NoError(t, err)

	body := EmbeddedSignupCallbackRequest{
		Code:      "auth-code-123",
		State:     encodedState,
		AppID:     "app-1",
		AppSecret: "secret-1",
	}
	w, c := newTestContext(http.MethodPost, "/oauth/whatsapp/embedded-signup/callback", body)
	c.Set("tenant_id", "tenant-1")
	c.Set("user_id", "user-1")

	handler.CompleteEmbeddedSignup(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var resp map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Equal(t, "state expired", resp["error"])
}

func TestCreateCoexistenceChannel_InvalidBody(t *testing.T) {
	handler, _ := setupEmbeddedSignupTest(t)

	// Missing required fields
	body := map[string]string{"name": "test"}
	w, c := newTestContext(http.MethodPost, "/oauth/whatsapp/embedded-signup/create-channel", body)
	c.Set("tenant_id", "tenant-1")
	c.Set("user_id", "user-1")

	handler.CreateCoexistenceChannel(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var resp map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.NotEmpty(t, resp["error"])
}
