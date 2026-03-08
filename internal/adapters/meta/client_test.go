package meta

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestClient(t *testing.T, handler http.HandlerFunc) (*Client, *httptest.Server) {
	t.Helper()
	server := httptest.NewServer(handler)
	c := NewClient("test-access-token", "test-app-secret")
	c.baseURL = server.URL
	c.apiVersion = "v22.0"
	return c, server
}

func TestNewClient(t *testing.T) {
	c := NewClient("token123", "secret456")
	assert.Equal(t, "token123", c.accessToken)
	assert.Equal(t, "secret456", c.appSecret)
	assert.Equal(t, DefaultAPIVersion, c.apiVersion)
	assert.Equal(t, GraphAPIBaseURL, c.baseURL)
}

func TestNewInstagramClient(t *testing.T) {
	c := NewInstagramClient("token", "secret")
	assert.Equal(t, InstagramAPIBaseURL, c.baseURL)
}

func TestClient_SetAPIVersion(t *testing.T) {
	c := NewClient("t", "s")
	c.SetAPIVersion("v19.0")
	assert.Equal(t, "v19.0", c.apiVersion)
}

func TestClient_SetAccessToken(t *testing.T) {
	c := NewClient("old", "s")
	c.SetAccessToken("new")
	assert.Equal(t, "new", c.accessToken)
}

func TestClient_SendMessage(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		c, server := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, http.MethodPost, r.Method)
			assert.Contains(t, r.URL.Path, "/page-123/messages")
			assert.Equal(t, "Bearer test-access-token", r.Header.Get("Authorization"))

			json.NewEncoder(w).Encode(SendMessageResponse{
				RecipientID: "user-456",
				MessageID:   "mid.123",
			})
		})
		defer server.Close()

		resp, err := c.SendMessage(context.Background(), "page-123", &OutboundMessage{
			Recipient: MessageRecipient{ID: "user-456"},
			Message:   MessageContent{Text: "hello"},
		})
		require.NoError(t, err)
		assert.Equal(t, "mid.123", resp.MessageID)
		assert.Equal(t, "user-456", resp.RecipientID)
	})

	t.Run("with me endpoint", func(t *testing.T) {
		c, server := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
			assert.Contains(t, r.URL.Path, "/me/messages")
			json.NewEncoder(w).Encode(SendMessageResponse{MessageID: "mid.1"})
		})
		defer server.Close()

		resp, err := c.SendMessage(context.Background(), "me", &OutboundMessage{})
		require.NoError(t, err)
		assert.Equal(t, "mid.1", resp.MessageID)
	})

	t.Run("API error", func(t *testing.T) {
		c, server := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"error": map[string]interface{}{
					"message": "Invalid recipient",
					"type":    "OAuthException",
					"code":    100,
				},
			})
		})
		defer server.Close()

		_, err := c.SendMessage(context.Background(), "page-1", &OutboundMessage{})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "Invalid recipient")
	})
}

func TestClient_GetUserProfile(t *testing.T) {
	c, server := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodGet, r.Method)
		assert.Contains(t, r.URL.Path, "/user-789")
		json.NewEncoder(w).Encode(UserProfile{
			ID:        "789",
			Name:      "John Doe",
			FirstName: "John",
			LastName:  "Doe",
		})
	})
	defer server.Close()

	profile, err := c.GetUserProfile(context.Background(), "user-789", nil)
	require.NoError(t, err)
	assert.Equal(t, "789", profile.ID)
	assert.Equal(t, "John Doe", profile.Name)
}

func TestClient_GetMyPages(t *testing.T) {
	c, server := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Contains(t, r.URL.Path, "/me/accounts")
		json.NewEncoder(w).Encode(PagesResponse{
			Data: []PageInfo{
				{ID: "page-1", Name: "My Page", Category: "Business"},
			},
		})
	})
	defer server.Close()

	pages, err := c.GetMyPages(context.Background())
	require.NoError(t, err)
	require.Len(t, pages.Data, 1)
	assert.Equal(t, "page-1", pages.Data[0].ID)
	assert.Equal(t, "My Page", pages.Data[0].Name)
}

func TestClient_GetInstagramAccount(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		c, server := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
			json.NewEncoder(w).Encode(map[string]interface{}{
				"instagram_business_account": map[string]interface{}{
					"id":       "ig-123",
					"username": "mybrand",
					"name":     "My Brand",
				},
			})
		})
		defer server.Close()

		account, err := c.GetInstagramAccount(context.Background(), "page-1")
		require.NoError(t, err)
		assert.Equal(t, "ig-123", account.ID)
		assert.Equal(t, "mybrand", account.Username)
	})

	t.Run("no account", func(t *testing.T) {
		c, server := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
			json.NewEncoder(w).Encode(map[string]interface{}{})
		})
		defer server.Close()

		_, err := c.GetInstagramAccount(context.Background(), "page-1")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "no Instagram business account")
	})
}

func TestClient_SubscribeToWebhook(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		c, server := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, http.MethodPost, r.Method)
			assert.Contains(t, r.URL.Path, "/page-1/subscribed_apps")
			json.NewEncoder(w).Encode(SubscribedAppsResponse{Success: true})
		})
		defer server.Close()

		err := c.SubscribeToWebhook(context.Background(), "page-1", []string{"messages"})
		assert.NoError(t, err)
	})

	t.Run("failure", func(t *testing.T) {
		c, server := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
			json.NewEncoder(w).Encode(SubscribedAppsResponse{Success: false})
		})
		defer server.Close()

		err := c.SubscribeToWebhook(context.Background(), "page-1", nil)
		assert.Error(t, err)
	})
}

func TestClient_ExchangeCodeForToken(t *testing.T) {
	c, server := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Contains(t, r.URL.Path, "/oauth/access_token")
		assert.Equal(t, "auth-code-123", r.URL.Query().Get("code"))
		json.NewEncoder(w).Encode(OAuthTokenResponse{
			AccessToken: "new-token",
			TokenType:   "bearer",
			ExpiresIn:   3600,
		})
	})
	defer server.Close()

	resp, err := c.ExchangeCodeForToken(context.Background(), "app-id", "app-secret", "https://example.com/callback", "auth-code-123")
	require.NoError(t, err)
	assert.Equal(t, "new-token", resp.AccessToken)
	assert.Equal(t, "bearer", resp.TokenType)
}

func TestClient_GetLongLivedToken(t *testing.T) {
	c, server := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "fb_exchange_token", r.URL.Query().Get("grant_type"))
		json.NewEncoder(w).Encode(LongLivedTokenResponse{
			AccessToken: "long-lived-token",
			TokenType:   "bearer",
			ExpiresIn:   5184000,
		})
	})
	defer server.Close()

	resp, err := c.GetLongLivedToken(context.Background(), "app-id", "app-secret", "short-token")
	require.NoError(t, err)
	assert.Equal(t, "long-lived-token", resp.AccessToken)
	assert.Equal(t, int64(5184000), resp.ExpiresIn)
}

func TestValidateWebhookSignature(t *testing.T) {
	secret := "my-app-secret"
	payload := []byte(`{"object":"page","entry":[]}`)

	// Generate valid signature
	h := hmac.New(sha256.New, []byte(secret))
	h.Write(payload)
	validSig := "sha256=" + hex.EncodeToString(h.Sum(nil))

	tests := []struct {
		name      string
		secret    string
		payload   []byte
		signature string
		valid     bool
	}{
		{"valid signature", secret, payload, validSig, true},
		{"wrong signature", secret, payload, "sha256=0000000000000000000000000000000000000000000000000000000000000000", false},
		{"wrong secret", "wrong-secret", payload, validSig, false},
		{"modified payload", secret, []byte(`modified`), validSig, false},
		{"missing sha256 prefix", secret, payload, "invalid", false},
		{"empty signature", secret, payload, "", false},
		{"short signature", secret, payload, "sha25", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.valid, ValidateWebhookSignature(tt.secret, tt.payload, tt.signature))
		})
	}
}

func TestParseWebhookPayload(t *testing.T) {
	t.Run("valid payload", func(t *testing.T) {
		body := []byte(`{"object":"page","entry":[{"id":"123","time":1234567890,"messaging":[{"sender":{"id":"user-1"},"recipient":{"id":"page-1"},"timestamp":1234567890000,"message":{"mid":"mid.1","text":"hello"}}]}]}`)
		payload, err := ParseWebhookPayload(body)
		require.NoError(t, err)
		assert.Equal(t, "page", payload.Object)
		require.Len(t, payload.Entry, 1)
		require.Len(t, payload.Entry[0].Messaging, 1)
		assert.Equal(t, "user-1", payload.Entry[0].Messaging[0].Sender.ID)
	})

	t.Run("invalid JSON", func(t *testing.T) {
		_, err := ParseWebhookPayload([]byte(`{invalid`))
		assert.Error(t, err)
	})
}

func TestExtractMessagingEvents(t *testing.T) {
	entries := []WebhookEntry{
		{
			Messaging: []MessagingEvent{
				{Sender: MessagingParty{ID: "1"}},
				{Sender: MessagingParty{ID: "2"}},
			},
			Standby: []MessagingEvent{
				{Sender: MessagingParty{ID: "3"}},
			},
		},
		{
			Messaging: []MessagingEvent{
				{Sender: MessagingParty{ID: "4"}},
			},
		},
	}

	events := ExtractMessagingEvents(entries)
	assert.Len(t, events, 4)
}

func TestParseInboundMessage(t *testing.T) {
	t.Run("text message", func(t *testing.T) {
		event := &MessagingEvent{
			Sender:    MessagingParty{ID: "user-1"},
			Recipient: MessagingParty{ID: "page-1"},
			Timestamp: 1704067200000, // 2024-01-01
			Message: &InboundMessage{
				MID:  "mid.123",
				Text: "Hello!",
			},
		}

		msg := ParseInboundMessage(event)
		require.NotNil(t, msg)
		assert.Equal(t, "mid.123", msg.ExternalID)
		assert.Equal(t, "user-1", msg.SenderID)
		assert.Equal(t, "page-1", msg.RecipientID)
		assert.Equal(t, "Hello!", msg.Text)
		assert.False(t, msg.IsEcho)
	})

	t.Run("echo message", func(t *testing.T) {
		event := &MessagingEvent{
			Sender:    MessagingParty{ID: "page-1"},
			Recipient: MessagingParty{ID: "user-1"},
			Timestamp: 1704067200000,
			Message: &InboundMessage{
				MID:    "mid.456",
				Text:   "Reply",
				IsEcho: true,
			},
		}

		msg := ParseInboundMessage(event)
		require.NotNil(t, msg)
		assert.True(t, msg.IsEcho)
	})

	t.Run("message with attachments", func(t *testing.T) {
		event := &MessagingEvent{
			Sender:    MessagingParty{ID: "user-1"},
			Recipient: MessagingParty{ID: "page-1"},
			Timestamp: 1704067200000,
			Message: &InboundMessage{
				MID: "mid.789",
				Attachments: []InboundAttachment{
					{Type: "image", Payload: AttachmentPayload{URL: "https://example.com/img.jpg"}},
				},
			},
		}

		msg := ParseInboundMessage(event)
		require.NotNil(t, msg)
		require.Len(t, msg.Attachments, 1)
		assert.Equal(t, "image", msg.Attachments[0].Type)
		assert.Equal(t, "https://example.com/img.jpg", msg.Attachments[0].URL)
	})

	t.Run("message with location", func(t *testing.T) {
		event := &MessagingEvent{
			Sender:    MessagingParty{ID: "user-1"},
			Recipient: MessagingParty{ID: "page-1"},
			Timestamp: 1704067200000,
			Message: &InboundMessage{
				MID: "mid.loc",
				Attachments: []InboundAttachment{
					{
						Type: "location",
						Payload: AttachmentPayload{
							Coordinates: &Coordinates{Lat: -23.5505, Long: -46.6333},
						},
					},
				},
			},
		}

		msg := ParseInboundMessage(event)
		require.NotNil(t, msg)
		require.Len(t, msg.Attachments, 1)
		assert.InDelta(t, -23.5505, msg.Attachments[0].Lat, 0.001)
		assert.InDelta(t, -46.6333, msg.Attachments[0].Long, 0.001)
	})

	t.Run("message with quick reply", func(t *testing.T) {
		event := &MessagingEvent{
			Sender:    MessagingParty{ID: "user-1"},
			Recipient: MessagingParty{ID: "page-1"},
			Timestamp: 1704067200000,
			Message: &InboundMessage{
				MID:        "mid.qr",
				Text:       "Yes",
				QuickReply: &QuickReplyPayload{Payload: "YES_PAYLOAD"},
			},
		}

		msg := ParseInboundMessage(event)
		require.NotNil(t, msg)
		assert.Equal(t, "YES_PAYLOAD", msg.QuickReply)
	})

	t.Run("message with reply_to", func(t *testing.T) {
		event := &MessagingEvent{
			Sender:    MessagingParty{ID: "user-1"},
			Recipient: MessagingParty{ID: "page-1"},
			Timestamp: 1704067200000,
			Message: &InboundMessage{
				MID:     "mid.reply",
				Text:    "replying",
				ReplyTo: &ReplyTo{MID: "mid.original"},
			},
		}

		msg := ParseInboundMessage(event)
		require.NotNil(t, msg)
		assert.Equal(t, "mid.original", msg.ReplyToMID)
	})

	t.Run("nil message", func(t *testing.T) {
		event := &MessagingEvent{
			Sender:    MessagingParty{ID: "user-1"},
			Recipient: MessagingParty{ID: "page-1"},
			Timestamp: 1704067200000,
		}

		msg := ParseInboundMessage(event)
		assert.Nil(t, msg)
	})

	t.Run("timestamp conversion", func(t *testing.T) {
		event := &MessagingEvent{
			Sender:    MessagingParty{ID: "user-1"},
			Recipient: MessagingParty{ID: "page-1"},
			Timestamp: 1704067200000,
			Message:   &InboundMessage{MID: "mid.1", Text: "test"},
		}

		msg := ParseInboundMessage(event)
		require.NotNil(t, msg)
		expected := time.Unix(1704067200, 0)
		assert.Equal(t, expected, msg.Timestamp)
	})
}

func TestAPIError(t *testing.T) {
	err := &APIError{
		Message: "Invalid token",
		Type:    "OAuthException",
		Code:    190,
	}
	assert.Equal(t, "Invalid token", err.Error())
}

func TestBuildURL(t *testing.T) {
	c := NewClient("token", "secret")
	url := c.buildURL("me/messages")
	assert.Equal(t, fmt.Sprintf("%s/%s/me/messages", GraphAPIBaseURL, DefaultAPIVersion), url)
}
