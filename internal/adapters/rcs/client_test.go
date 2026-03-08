package rcs

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func validConfig() *Config {
	return &Config{
		Provider: ProviderZenvia,
		AgentID:  "agent-123",
		APIKey:   "key-abc",
	}
}

// ========== NewClient ==========

func TestNewClient_Valid(t *testing.T) {
	client, err := NewClient(validConfig())
	require.NoError(t, err)
	assert.NotNil(t, client)
}

func TestNewClient_InvalidConfig(t *testing.T) {
	cfg := &Config{Provider: "", AgentID: "", APIKey: ""}
	client, err := NewClient(cfg)
	assert.Error(t, err)
	assert.Nil(t, client)
}

func TestNewClient_MissingAPIKey(t *testing.T) {
	cfg := &Config{Provider: ProviderZenvia, AgentID: "agent-123", APIKey: ""}
	client, err := NewClient(cfg)
	assert.Error(t, err)
	assert.Nil(t, client)
}

// ========== ValidateWebhook ==========

func TestValidateWebhook_CorrectSignature(t *testing.T) {
	secret := "my-webhook-secret"
	cfg := validConfig()
	cfg.WebhookSecret = secret

	client, err := NewClient(cfg)
	require.NoError(t, err)

	body := []byte(`{"type":"message","text":"hello"}`)

	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(body)
	signature := hex.EncodeToString(mac.Sum(nil))

	assert.True(t, client.ValidateWebhook(signature, body))
}

func TestValidateWebhook_WrongSignature(t *testing.T) {
	cfg := validConfig()
	cfg.WebhookSecret = "my-webhook-secret"

	client, err := NewClient(cfg)
	require.NoError(t, err)

	body := []byte(`{"type":"message","text":"hello"}`)
	assert.False(t, client.ValidateWebhook("wrong-signature", body))
}

func TestValidateWebhook_EmptySecret_SkipsValidation(t *testing.T) {
	cfg := validConfig()
	cfg.WebhookSecret = ""

	client, err := NewClient(cfg)
	require.NoError(t, err)

	body := []byte(`{"type":"message"}`)
	assert.True(t, client.ValidateWebhook("any-signature", body))
}

func TestValidateWebhook_Sha256Prefix(t *testing.T) {
	secret := "test-secret"
	cfg := validConfig()
	cfg.WebhookSecret = secret

	client, err := NewClient(cfg)
	require.NoError(t, err)

	body := []byte(`{"data":"test"}`)

	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(body)
	expectedSig := hex.EncodeToString(mac.Sum(nil))

	// With "sha256=" prefix
	assert.True(t, client.ValidateWebhook("sha256="+expectedSig, body))
}

func TestValidateWebhook_Sha256Prefix_Wrong(t *testing.T) {
	cfg := validConfig()
	cfg.WebhookSecret = "test-secret"

	client, err := NewClient(cfg)
	require.NoError(t, err)

	body := []byte(`{"data":"test"}`)
	assert.False(t, client.ValidateWebhook("sha256=invalidhex", body))
}

// ========== SendMessage - provider routing with httptest ==========

func TestSendMessage_Zenvia(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/channels/rcs/messages", r.URL.Path)
		assert.Equal(t, "POST", r.Method)
		assert.Equal(t, "key-zenvia", r.Header.Get("X-API-TOKEN"))
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"id": "zenvia-msg-1"})
	}))
	defer server.Close()

	cfg := &Config{
		Provider: ProviderZenvia,
		AgentID:  "agent-zenvia",
		APIKey:   "key-zenvia",
		BaseURL:  server.URL,
	}

	client, err := NewClient(cfg)
	require.NoError(t, err)

	result, err := client.SendMessage(context.Background(), &OutboundMessage{
		To:   "+5511999999999",
		Text: "Hello via Zenvia",
	})
	require.NoError(t, err)
	assert.True(t, result.Success)
	assert.Equal(t, "zenvia-msg-1", result.MessageID)
}

func TestSendMessage_Infobip(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/rcs/1/messages", r.URL.Path)
		assert.Equal(t, "POST", r.Method)
		assert.Equal(t, "App key-infobip", r.Header.Get("Authorization"))

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"messages": []map[string]string{{"messageId": "infobip-msg-1"}},
		})
	}))
	defer server.Close()

	cfg := &Config{
		Provider: ProviderInfobip,
		AgentID:  "agent-infobip",
		APIKey:   "key-infobip",
		BaseURL:  server.URL,
	}

	client, err := NewClient(cfg)
	require.NoError(t, err)

	result, err := client.SendMessage(context.Background(), &OutboundMessage{
		To:   "+5511999999999",
		Text: "Hello via Infobip",
	})
	require.NoError(t, err)
	assert.True(t, result.Success)
	assert.Equal(t, "infobip-msg-1", result.MessageID)
}

func TestSendMessage_Pontaltech(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/rcs/send", r.URL.Path)
		assert.Equal(t, "POST", r.Method)
		assert.Equal(t, "Bearer key-pontal", r.Header.Get("Authorization"))

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"id": "pontal-msg-1"})
	}))
	defer server.Close()

	cfg := &Config{
		Provider: ProviderPontaltech,
		AgentID:  "agent-pontal",
		APIKey:   "key-pontal",
		BaseURL:  server.URL,
	}

	client, err := NewClient(cfg)
	require.NoError(t, err)

	result, err := client.SendMessage(context.Background(), &OutboundMessage{
		To:   "+5511999999999",
		Text: "Hello via Pontaltech",
	})
	require.NoError(t, err)
	assert.True(t, result.Success)
	assert.Equal(t, "pontal-msg-1", result.MessageID)
}

func TestSendMessage_Google(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Contains(t, r.URL.Path, "/phones/")
		assert.Equal(t, "POST", r.Method)
		assert.Equal(t, "Bearer key-google", r.Header.Get("Authorization"))

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"name": "google-msg-1"})
	}))
	defer server.Close()

	cfg := &Config{
		Provider: ProviderGoogle,
		AgentID:  "agent-google",
		APIKey:   "key-google",
		BaseURL:  server.URL,
	}

	client, err := NewClient(cfg)
	require.NoError(t, err)

	result, err := client.SendMessage(context.Background(), &OutboundMessage{
		To:   "+5511999999999",
		Text: "Hello via Google",
	})
	require.NoError(t, err)
	assert.True(t, result.Success)
	assert.Equal(t, "google-msg-1", result.MessageID)
}

func TestSendMessage_InvalidProvider(t *testing.T) {
	cfg := &Config{
		Provider: Provider("unknown"),
		AgentID:  "agent",
		APIKey:   "key",
	}

	client, err := NewClient(cfg)
	require.NoError(t, err)

	result, err := client.SendMessage(context.Background(), &OutboundMessage{
		To:   "+5511999999999",
		Text: "Hello",
	})
	assert.Error(t, err)
	assert.Equal(t, ErrInvalidProvider, err)
	assert.Nil(t, result)
}

func TestSendMessage_Zenvia_ServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"error":"server error"}`))
	}))
	defer server.Close()

	cfg := &Config{
		Provider: ProviderZenvia,
		AgentID:  "agent",
		APIKey:   "key",
		BaseURL:  server.URL,
	}

	client, err := NewClient(cfg)
	require.NoError(t, err)

	result, err := client.SendMessage(context.Background(), &OutboundMessage{
		To:   "+5511999999999",
		Text: "Hello",
	})
	require.NoError(t, err)
	assert.False(t, result.Success)
	assert.Contains(t, result.Error, "server error")
}

func TestSendMessage_Zenvia_WithMedia(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body ZenviaMessage
		json.NewDecoder(r.Body).Decode(&body)
		// Expect 2 contents: text + file
		assert.Len(t, body.Contents, 2)
		assert.Equal(t, "text", body.Contents[0].Type)
		assert.Equal(t, "file", body.Contents[1].Type)
		assert.Equal(t, "https://example.com/img.jpg", body.Contents[1].File.FileURL)

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"id": "msg-media"})
	}))
	defer server.Close()

	cfg := &Config{
		Provider: ProviderZenvia,
		AgentID:  "agent",
		APIKey:   "key",
		BaseURL:  server.URL,
	}

	client, err := NewClient(cfg)
	require.NoError(t, err)

	result, err := client.SendMessage(context.Background(), &OutboundMessage{
		To:        "+5511999999999",
		Text:      "Check this image",
		MediaURL:  "https://example.com/img.jpg",
		MediaType: "image/jpeg",
	})
	require.NoError(t, err)
	assert.True(t, result.Success)
	assert.Equal(t, "msg-media", result.MessageID)
}

// ========== ParseWebhook ==========

func TestParseWebhook_Zenvia_Message(t *testing.T) {
	cfg := validConfig()
	cfg.Provider = ProviderZenvia
	client, err := NewClient(cfg)
	require.NoError(t, err)

	body := []byte(`{
		"id": "wh-1",
		"timestamp": "2026-03-08T10:00:00Z",
		"type": "MESSAGE",
		"channel": "rcs",
		"direction": "IN",
		"message": {
			"id": "msg-ext-1",
			"from": "+5511999999999",
			"to": "agent-123",
			"contents": [{"type": "text", "text": "Hello agent"}]
		}
	}`)

	payload, err := client.ParseWebhook(body)
	require.NoError(t, err)
	assert.Equal(t, ProviderZenvia, payload.Provider)
	assert.Equal(t, "message", payload.Type)
	require.NotNil(t, payload.Message)
	assert.Equal(t, "msg-ext-1", payload.Message.ExternalID)
	assert.Equal(t, "+5511999999999", payload.Message.SenderPhone)
	assert.Equal(t, "Hello agent", payload.Message.Text)
	assert.Equal(t, "agent-123", payload.Message.AgentID)
}

func TestParseWebhook_Zenvia_Status(t *testing.T) {
	cfg := validConfig()
	cfg.Provider = ProviderZenvia
	client, err := NewClient(cfg)
	require.NoError(t, err)

	body := []byte(`{
		"id": "msg-1",
		"timestamp": "2026-03-08T10:00:00Z",
		"type": "MESSAGE_STATUS",
		"messageStatus": {
			"timestamp": "2026-03-08T10:00:01Z",
			"code": "DELIVERED"
		}
	}`)

	payload, err := client.ParseWebhook(body)
	require.NoError(t, err)
	assert.Equal(t, "status", payload.Type)
	require.NotNil(t, payload.Status)
	assert.Equal(t, "msg-1", payload.Status.MessageID)
	assert.Equal(t, StatusDelivered, payload.Status.Status)
}

func TestParseWebhook_Infobip_Message(t *testing.T) {
	cfg := validConfig()
	cfg.Provider = ProviderInfobip
	client, err := NewClient(cfg)
	require.NoError(t, err)

	body := []byte(`{
		"results": [{
			"messageId": "ib-msg-1",
			"from": "+5511888888888",
			"to": "agent-ib",
			"receivedAt": "2026-03-08T12:00:00Z",
			"text": "Hello Infobip"
		}]
	}`)

	payload, err := client.ParseWebhook(body)
	require.NoError(t, err)
	assert.Equal(t, ProviderInfobip, payload.Provider)
	assert.Equal(t, "message", payload.Type)
	require.NotNil(t, payload.Message)
	assert.Equal(t, "ib-msg-1", payload.Message.ExternalID)
	assert.Equal(t, "Hello Infobip", payload.Message.Text)
}

func TestParseWebhook_Infobip_Status(t *testing.T) {
	cfg := validConfig()
	cfg.Provider = ProviderInfobip
	client, err := NewClient(cfg)
	require.NoError(t, err)

	body := []byte(`{
		"results": [{
			"messageId": "ib-msg-2",
			"from": "+5511888888888",
			"to": "agent-ib",
			"receivedAt": "2026-03-08T12:00:00Z",
			"status": "DELIVERED"
		}]
	}`)

	payload, err := client.ParseWebhook(body)
	require.NoError(t, err)
	assert.Equal(t, "status", payload.Type)
	require.NotNil(t, payload.Status)
	assert.Equal(t, StatusDelivered, payload.Status.Status)
}

func TestParseWebhook_Pontaltech_Message(t *testing.T) {
	cfg := validConfig()
	cfg.Provider = ProviderPontaltech
	client, err := NewClient(cfg)
	require.NoError(t, err)

	body := []byte(`{
		"id": "pt-msg-1",
		"type": "message",
		"from": "+5511777777777",
		"to": "agent-pt",
		"content": "Hello Pontaltech",
		"timestamp": "2026-03-08T14:00:00Z"
	}`)

	payload, err := client.ParseWebhook(body)
	require.NoError(t, err)
	assert.Equal(t, ProviderPontaltech, payload.Provider)
	assert.Equal(t, "message", payload.Type)
	require.NotNil(t, payload.Message)
	assert.Equal(t, "pt-msg-1", payload.Message.ExternalID)
	assert.Equal(t, "Hello Pontaltech", payload.Message.Text)
}

func TestParseWebhook_Pontaltech_Status(t *testing.T) {
	cfg := validConfig()
	cfg.Provider = ProviderPontaltech
	client, err := NewClient(cfg)
	require.NoError(t, err)

	body := []byte(`{
		"id": "pt-msg-2",
		"type": "status",
		"from": "+5511777777777",
		"to": "agent-pt",
		"content": "",
		"status": "delivered",
		"timestamp": "2026-03-08T14:01:00Z"
	}`)

	payload, err := client.ParseWebhook(body)
	require.NoError(t, err)
	assert.Equal(t, "status", payload.Type)
	require.NotNil(t, payload.Status)
	assert.Equal(t, StatusDelivered, payload.Status.Status)
}

func TestParseWebhook_Google_Message(t *testing.T) {
	cfg := validConfig()
	cfg.Provider = ProviderGoogle
	client, err := NewClient(cfg)
	require.NoError(t, err)

	body := []byte(`{
		"senderPhoneNumber": "+5511666666666",
		"messageId": "goog-msg-1",
		"sendTime": "2026-03-08T16:00:00Z",
		"text": "Hello Google RBM"
	}`)

	payload, err := client.ParseWebhook(body)
	require.NoError(t, err)
	assert.Equal(t, ProviderGoogle, payload.Provider)
	assert.Equal(t, "message", payload.Type)
	require.NotNil(t, payload.Message)
	assert.Equal(t, "goog-msg-1", payload.Message.ExternalID)
	assert.Equal(t, "+5511666666666", payload.Message.SenderPhone)
	assert.Equal(t, "Hello Google RBM", payload.Message.Text)
	assert.Equal(t, "agent-123", payload.Message.AgentID) // Uses config AgentID
}

func TestParseWebhook_InvalidProvider(t *testing.T) {
	cfg := &Config{
		Provider: Provider("unknown"),
		AgentID:  "agent",
		APIKey:   "key",
	}
	client, err := NewClient(cfg)
	require.NoError(t, err)

	_, err = client.ParseWebhook([]byte(`{}`))
	assert.Error(t, err)
	assert.Equal(t, ErrInvalidProvider, err)
}

func TestParseWebhook_InvalidJSON(t *testing.T) {
	cfg := validConfig()
	client, err := NewClient(cfg)
	require.NoError(t, err)

	_, err = client.ParseWebhook([]byte(`not-json`))
	assert.Error(t, err)
}

// ========== GetAgentInfo ==========

func TestGetAgentInfo_Zenvia(t *testing.T) {
	cfg := validConfig()
	cfg.Provider = ProviderZenvia
	client, err := NewClient(cfg)
	require.NoError(t, err)

	info, err := client.GetAgentInfo(context.Background())
	require.NoError(t, err)
	assert.Equal(t, "zenvia", info["provider"])
	assert.Equal(t, "agent-123", info["agent_id"])
}

func TestGetAgentInfo_Infobip(t *testing.T) {
	cfg := validConfig()
	cfg.Provider = ProviderInfobip
	client, err := NewClient(cfg)
	require.NoError(t, err)

	info, err := client.GetAgentInfo(context.Background())
	require.NoError(t, err)
	assert.Equal(t, "infobip", info["provider"])
	assert.Equal(t, "agent-123", info["agent_id"])
}

func TestGetAgentInfo_DefaultProvider(t *testing.T) {
	cfg := &Config{
		Provider: ProviderPontaltech,
		AgentID:  "agent-pt",
		APIKey:   "key",
	}
	client, err := NewClient(cfg)
	require.NoError(t, err)

	info, err := client.GetAgentInfo(context.Background())
	require.NoError(t, err)
	assert.Equal(t, "pontaltech", info["provider"])
	assert.Equal(t, "agent-pt", info["agent_id"])
}

// ========== Status mapping helpers ==========

func TestMapZenviaStatus(t *testing.T) {
	tests := []struct {
		code     string
		expected DeliveryStatus
	}{
		{"SENT", StatusSent},
		{"DELIVERED", StatusDelivered},
		{"READ", StatusRead},
		{"FAILED", StatusFailed},
		{"REJECTED", StatusFailed},
		{"UNKNOWN", StatusPending},
	}
	for _, tt := range tests {
		t.Run(tt.code, func(t *testing.T) {
			assert.Equal(t, tt.expected, mapZenviaStatus(tt.code))
		})
	}
}

func TestMapInfobipStatus(t *testing.T) {
	tests := []struct {
		status   string
		expected DeliveryStatus
	}{
		{"PENDING", StatusPending},
		{"DELIVERED", StatusDelivered},
		{"SEEN", StatusRead},
		{"READ", StatusRead},
		{"FAILED", StatusFailed},
		{"REJECTED", StatusFailed},
		{"OTHER", StatusSent},
	}
	for _, tt := range tests {
		t.Run(tt.status, func(t *testing.T) {
			assert.Equal(t, tt.expected, mapInfobipStatus(tt.status))
		})
	}
}
