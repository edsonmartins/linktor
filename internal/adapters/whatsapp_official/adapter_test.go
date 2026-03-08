package whatsapp_official

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/msgfy/linktor/pkg/plugin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func validConfig() map[string]string {
	return map[string]string{
		"access_token":    "test-access-token",
		"phone_number_id": "123456789",
		"verify_token":    "my-verify-token",
	}
}

func initializedAdapter(t *testing.T) *Adapter {
	t.Helper()
	a := NewAdapter()
	require.NoError(t, a.Initialize(validConfig()))
	return a
}

func computeHMAC(secret string, body []byte) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(body)
	return "sha256=" + hex.EncodeToString(mac.Sum(nil))
}

// ---------------------------------------------------------------------------
// 1. TestNewAdapter
// ---------------------------------------------------------------------------

func TestNewAdapter(t *testing.T) {
	a := NewAdapter()

	t.Run("ChannelType", func(t *testing.T) {
		assert.Equal(t, plugin.ChannelTypeWhatsAppOfficial, a.GetChannelType())
	})

	t.Run("ChannelInfo", func(t *testing.T) {
		info := a.GetChannelInfo()
		require.NotNil(t, info)
		assert.Equal(t, plugin.ChannelTypeWhatsAppOfficial, info.Type)
		assert.Equal(t, "WhatsApp Business Cloud API", info.Name)
		assert.NotEmpty(t, info.Version)
	})

	t.Run("Capabilities", func(t *testing.T) {
		caps := a.GetCapabilities()
		require.NotNil(t, caps)

		assert.True(t, caps.SupportsTemplates)
		assert.True(t, caps.SupportsInteractive)
		assert.True(t, caps.SupportsReadReceipts)
		assert.True(t, caps.SupportsReactions)
		assert.True(t, caps.SupportsReplies)
		assert.False(t, caps.SupportsTypingIndicator)
		assert.Equal(t, 4096, caps.MaxMessageLength)
		assert.Equal(t, int64(16*1024*1024), caps.MaxMediaSize)
		assert.Equal(t, 1, caps.MaxAttachments)
	})

	t.Run("SupportedContentTypes", func(t *testing.T) {
		caps := a.GetCapabilities()
		expected := []plugin.ContentType{
			plugin.ContentTypeText,
			plugin.ContentTypeImage,
			plugin.ContentTypeVideo,
			plugin.ContentTypeAudio,
			plugin.ContentTypeDocument,
			plugin.ContentTypeLocation,
			plugin.ContentTypeContact,
			plugin.ContentTypeTemplate,
			plugin.ContentTypeInteractive,
		}
		assert.Equal(t, expected, caps.SupportedContentTypes)
	})

	t.Run("SessionsMapInitialized", func(t *testing.T) {
		assert.NotNil(t, a.sessions)
	})
}

// ---------------------------------------------------------------------------
// 2. TestAdapter_Initialize
// ---------------------------------------------------------------------------

func TestAdapter_Initialize(t *testing.T) {
	tests := []struct {
		name      string
		config    map[string]string
		wantErr   string
		checkFunc func(t *testing.T, a *Adapter)
	}{
		{
			name:   "valid config",
			config: validConfig(),
			checkFunc: func(t *testing.T, a *Adapter) {
				assert.Equal(t, "test-access-token", a.config.AccessToken)
				assert.Equal(t, "123456789", a.config.PhoneNumberID)
				assert.Equal(t, "my-verify-token", a.config.VerifyToken)
			},
		},
		{
			name: "missing access_token",
			config: map[string]string{
				"phone_number_id": "123",
				"verify_token":    "tok",
			},
			wantErr: "access_token is required",
		},
		{
			name: "missing phone_number_id",
			config: map[string]string{
				"access_token": "tok",
				"verify_token": "tok",
			},
			wantErr: "phone_number_id is required",
		},
		{
			name: "missing verify_token",
			config: map[string]string{
				"access_token":    "tok",
				"phone_number_id": "123",
			},
			wantErr: "verify_token is required",
		},
		{
			name: "default api_version",
			config: map[string]string{
				"access_token":    "tok",
				"phone_number_id": "123",
				"verify_token":    "tok",
			},
			checkFunc: func(t *testing.T, a *Adapter) {
				assert.Equal(t, DefaultAPIVersion, a.config.APIVersion)
			},
		},
		{
			name: "custom api_version",
			config: map[string]string{
				"access_token":    "tok",
				"phone_number_id": "123",
				"verify_token":    "tok",
				"api_version":     "v19.0",
			},
			checkFunc: func(t *testing.T, a *Adapter) {
				assert.Equal(t, "v19.0", a.config.APIVersion)
			},
		},
		{
			name: "optional business_id and webhook_secret",
			config: map[string]string{
				"access_token":    "tok",
				"phone_number_id": "123",
				"verify_token":    "tok",
				"business_id":     "biz-999",
				"webhook_secret":  "whsec-abc",
			},
			checkFunc: func(t *testing.T, a *Adapter) {
				assert.Equal(t, "biz-999", a.config.BusinessID)
				assert.Equal(t, "whsec-abc", a.config.WebhookSecret)
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			a := NewAdapter()
			err := a.Initialize(tc.config)

			if tc.wantErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tc.wantErr)
				return
			}

			require.NoError(t, err)
			if tc.checkFunc != nil {
				tc.checkFunc(t, a)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// 3. TestAdapter_ValidateWebhook
// ---------------------------------------------------------------------------

func TestAdapter_ValidateWebhook(t *testing.T) {
	body := []byte(`{"object":"whatsapp_business_account"}`)

	t.Run("valid signature", func(t *testing.T) {
		a := initializedAdapter(t)
		a.config.WebhookSecret = "my-secret"

		sig := computeHMAC("my-secret", body)
		headers := map[string]string{"X-Hub-Signature-256": sig}
		assert.True(t, a.ValidateWebhook(headers, body))
	})

	t.Run("invalid signature", func(t *testing.T) {
		a := initializedAdapter(t)
		a.config.WebhookSecret = "my-secret"

		headers := map[string]string{"X-Hub-Signature-256": "sha256=badhash"}
		assert.False(t, a.ValidateWebhook(headers, body))
	})

	t.Run("no webhook secret configured skips validation", func(t *testing.T) {
		a := initializedAdapter(t)
		a.config.WebhookSecret = ""

		headers := map[string]string{}
		assert.True(t, a.ValidateWebhook(headers, body))
	})

	t.Run("case insensitive header key", func(t *testing.T) {
		a := initializedAdapter(t)
		a.config.WebhookSecret = "my-secret"

		sig := computeHMAC("my-secret", body)
		headers := map[string]string{"x-hub-signature-256": sig}
		assert.True(t, a.ValidateWebhook(headers, body))
	})
}

// ---------------------------------------------------------------------------
// 4. TestAdapter_buildSendRequest
// ---------------------------------------------------------------------------

func TestAdapter_buildSendRequest(t *testing.T) {
	a := initializedAdapter(t)

	t.Run("text message", func(t *testing.T) {
		msg := &plugin.OutboundMessage{
			RecipientID: "5511999999999",
			ContentType: plugin.ContentTypeText,
			Content:     "Hello, world!",
			Metadata:    map[string]string{},
		}
		req, err := a.buildSendRequest(msg)
		require.NoError(t, err)
		assert.Equal(t, MessageTypeText, req.Type)
		assert.Equal(t, "5511999999999", req.To)
		require.NotNil(t, req.Text)
		assert.Equal(t, "Hello, world!", req.Text.Body)
		assert.False(t, req.Text.PreviewURL)
	})

	t.Run("text with preview_url", func(t *testing.T) {
		msg := &plugin.OutboundMessage{
			RecipientID: "5511999999999",
			ContentType: plugin.ContentTypeText,
			Content:     "Check https://example.com",
			Metadata:    map[string]string{"preview_url": "true"},
		}
		req, err := a.buildSendRequest(msg)
		require.NoError(t, err)
		assert.True(t, req.Text.PreviewURL)
	})

	t.Run("text with reply_to", func(t *testing.T) {
		msg := &plugin.OutboundMessage{
			RecipientID: "5511999999999",
			ContentType: plugin.ContentTypeText,
			Content:     "replying",
			Metadata:    map[string]string{"reply_to_id": "wamid.abc123"},
		}
		req, err := a.buildSendRequest(msg)
		require.NoError(t, err)
		require.NotNil(t, req.Context)
		assert.Equal(t, "wamid.abc123", req.Context.MessageID)
	})

	t.Run("image message with attachment URL", func(t *testing.T) {
		msg := &plugin.OutboundMessage{
			RecipientID: "5511999999999",
			ContentType: plugin.ContentTypeImage,
			Content:     "Nice photo",
			Attachments: []*plugin.Attachment{
				{URL: "https://example.com/photo.jpg"},
			},
			Metadata: map[string]string{},
		}
		req, err := a.buildSendRequest(msg)
		require.NoError(t, err)
		assert.Equal(t, MessageTypeImage, req.Type)
		require.NotNil(t, req.Image)
		assert.Equal(t, "https://example.com/photo.jpg", req.Image.Link)
		assert.Equal(t, "Nice photo", req.Image.Caption)
	})

	t.Run("image message with media_id in attachment metadata", func(t *testing.T) {
		msg := &plugin.OutboundMessage{
			RecipientID: "5511999999999",
			ContentType: plugin.ContentTypeImage,
			Content:     "caption",
			Attachments: []*plugin.Attachment{
				{Metadata: map[string]string{"media_id": "mid-123"}},
			},
			Metadata: map[string]string{},
		}
		req, err := a.buildSendRequest(msg)
		require.NoError(t, err)
		assert.Equal(t, "mid-123", req.Image.ID)
	})

	t.Run("video message", func(t *testing.T) {
		msg := &plugin.OutboundMessage{
			RecipientID: "5511999999999",
			ContentType: plugin.ContentTypeVideo,
			Content:     "video caption",
			Attachments: []*plugin.Attachment{
				{URL: "https://example.com/video.mp4"},
			},
			Metadata: map[string]string{},
		}
		req, err := a.buildSendRequest(msg)
		require.NoError(t, err)
		assert.Equal(t, MessageTypeVideo, req.Type)
		require.NotNil(t, req.Video)
		assert.Equal(t, "https://example.com/video.mp4", req.Video.Link)
	})

	t.Run("audio message", func(t *testing.T) {
		msg := &plugin.OutboundMessage{
			RecipientID: "5511999999999",
			ContentType: plugin.ContentTypeAudio,
			Content:     "",
			Attachments: []*plugin.Attachment{
				{URL: "https://example.com/audio.ogg"},
			},
			Metadata: map[string]string{},
		}
		req, err := a.buildSendRequest(msg)
		require.NoError(t, err)
		assert.Equal(t, MessageTypeAudio, req.Type)
		require.NotNil(t, req.Audio)
		assert.Equal(t, "https://example.com/audio.ogg", req.Audio.Link)
	})

	t.Run("document message with attachment", func(t *testing.T) {
		msg := &plugin.OutboundMessage{
			RecipientID: "5511999999999",
			ContentType: plugin.ContentTypeDocument,
			Content:     "please review",
			Attachments: []*plugin.Attachment{
				{
					URL:      "https://example.com/file.pdf",
					Filename: "report.pdf",
				},
			},
			Metadata: map[string]string{},
		}
		req, err := a.buildSendRequest(msg)
		require.NoError(t, err)
		assert.Equal(t, MessageTypeDocument, req.Type)
		require.NotNil(t, req.Document)
		assert.Equal(t, "https://example.com/file.pdf", req.Document.Link)
		assert.Equal(t, "report.pdf", req.Document.Filename)
		assert.Equal(t, "please review", req.Document.Caption)
	})

	t.Run("location from JSON content", func(t *testing.T) {
		loc := LocationObject{
			Latitude:  -23.5505,
			Longitude: -46.6333,
			Name:      "Sao Paulo",
			Address:   "Sao Paulo, BR",
		}
		locJSON, _ := json.Marshal(loc)

		msg := &plugin.OutboundMessage{
			RecipientID: "5511999999999",
			ContentType: plugin.ContentTypeLocation,
			Content:     string(locJSON),
			Metadata:    map[string]string{},
		}
		req, err := a.buildSendRequest(msg)
		require.NoError(t, err)
		assert.Equal(t, MessageTypeLocation, req.Type)
		require.NotNil(t, req.Location)
		assert.InDelta(t, -23.5505, req.Location.Latitude, 0.0001)
		assert.InDelta(t, -46.6333, req.Location.Longitude, 0.0001)
		assert.Equal(t, "Sao Paulo", req.Location.Name)
	})

	t.Run("location from metadata", func(t *testing.T) {
		msg := &plugin.OutboundMessage{
			RecipientID: "5511999999999",
			ContentType: plugin.ContentTypeLocation,
			Content:     "not json",
			Metadata: map[string]string{
				"latitude":  "-23.5505",
				"longitude": "-46.6333",
				"name":      "Sao Paulo",
				"address":   "Sao Paulo, BR",
			},
		}
		req, err := a.buildSendRequest(msg)
		require.NoError(t, err)
		assert.Equal(t, MessageTypeLocation, req.Type)
		require.NotNil(t, req.Location)
		assert.InDelta(t, -23.5505, req.Location.Latitude, 0.0001)
		assert.InDelta(t, -46.6333, req.Location.Longitude, 0.0001)
		assert.Equal(t, "Sao Paulo", req.Location.Name)
		assert.Equal(t, "Sao Paulo, BR", req.Location.Address)
	})

	t.Run("contacts from JSON content", func(t *testing.T) {
		contacts := []ContactContent{
			{
				Name: ContactName{
					FormattedName: "John Doe",
					FirstName:     "John",
					LastName:      "Doe",
				},
				Phones: []ContactPhone{
					{Phone: "+5511999999999", Type: "CELL"},
				},
			},
		}
		contactsJSON, _ := json.Marshal(contacts)

		msg := &plugin.OutboundMessage{
			RecipientID: "5511999999999",
			ContentType: plugin.ContentTypeContact,
			Content:     string(contactsJSON),
			Metadata:    map[string]string{},
		}
		req, err := a.buildSendRequest(msg)
		require.NoError(t, err)
		assert.Equal(t, MessageTypeContacts, req.Type)
		require.Len(t, req.Contacts, 1)
		assert.Equal(t, "John Doe", req.Contacts[0].Name.FormattedName)
	})

	t.Run("template from metadata", func(t *testing.T) {
		components := []TemplateComponent{
			{Type: "body", Parameters: []TemplateParameter{{Type: "text", Text: "World"}}},
		}
		compJSON, _ := json.Marshal(components)

		msg := &plugin.OutboundMessage{
			RecipientID: "5511999999999",
			ContentType: plugin.ContentTypeTemplate,
			Content:     "",
			Metadata: map[string]string{
				"template_name":       "hello_world",
				"template_language":   "pt_BR",
				"template_components": string(compJSON),
			},
		}
		req, err := a.buildSendRequest(msg)
		require.NoError(t, err)
		assert.Equal(t, MessageType("template"), req.Type)
		require.NotNil(t, req.Template)
		assert.Equal(t, "hello_world", req.Template.Name)
		assert.Equal(t, "pt_BR", req.Template.Language.Code)
		require.Len(t, req.Template.Components, 1)
	})

	t.Run("template default language", func(t *testing.T) {
		msg := &plugin.OutboundMessage{
			RecipientID: "5511999999999",
			ContentType: plugin.ContentTypeTemplate,
			Content:     "",
			Metadata: map[string]string{
				"template_name": "hello_world",
			},
		}
		req, err := a.buildSendRequest(msg)
		require.NoError(t, err)
		assert.Equal(t, "en", req.Template.Language.Code)
	})

	t.Run("template missing name", func(t *testing.T) {
		msg := &plugin.OutboundMessage{
			RecipientID: "5511999999999",
			ContentType: plugin.ContentTypeTemplate,
			Content:     "",
			Metadata:    map[string]string{},
		}
		_, err := a.buildSendRequest(msg)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "template_name is required")
	})

	t.Run("interactive from metadata", func(t *testing.T) {
		interactive := InteractiveObject{
			Type: "button",
			Body: &InteractiveBody{Text: "Choose an option"},
			Action: &InteractiveAction{
				Buttons: []InteractiveButton{
					{Type: "reply", Reply: &ButtonReply{ID: "btn1", Title: "Option 1"}},
				},
			},
		}
		interactiveJSON, _ := json.Marshal(interactive)

		msg := &plugin.OutboundMessage{
			RecipientID: "5511999999999",
			ContentType: plugin.ContentTypeInteractive,
			Content:     string(interactiveJSON),
			Metadata: map[string]string{
				"interactive_type": "button",
			},
		}
		req, err := a.buildSendRequest(msg)
		require.NoError(t, err)
		assert.Equal(t, MessageTypeInteractive, req.Type)
		require.NotNil(t, req.Interactive)
		assert.Equal(t, "button", req.Interactive.Type)
	})

	t.Run("interactive missing type", func(t *testing.T) {
		msg := &plugin.OutboundMessage{
			RecipientID: "5511999999999",
			ContentType: plugin.ContentTypeInteractive,
			Content:     "{}",
			Metadata:    map[string]string{},
		}
		_, err := a.buildSendRequest(msg)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "interactive_type is required")
	})

	t.Run("unknown content type defaults to text", func(t *testing.T) {
		msg := &plugin.OutboundMessage{
			RecipientID: "5511999999999",
			ContentType: plugin.ContentType("unknown_type"),
			Content:     "fallback text",
			Metadata:    map[string]string{},
		}
		req, err := a.buildSendRequest(msg)
		require.NoError(t, err)
		assert.Equal(t, MessageTypeText, req.Type)
		require.NotNil(t, req.Text)
		assert.Equal(t, "fallback text", req.Text.Body)
	})
}

// ---------------------------------------------------------------------------
// 5. TestAdapter_IsSessionValid
// ---------------------------------------------------------------------------

func TestAdapter_IsSessionValid(t *testing.T) {
	t.Run("no session", func(t *testing.T) {
		a := initializedAdapter(t)
		assert.False(t, a.IsSessionValid("+5511999999999"))
	})

	t.Run("valid session", func(t *testing.T) {
		a := initializedAdapter(t)
		a.sessions["+5511999999999"] = &SessionInfo{
			ContactID:             "+5511999999999",
			CanSendSessionMessage: true,
			SessionExpiresAt:      time.Now().Add(12 * time.Hour),
		}
		assert.True(t, a.IsSessionValid("+5511999999999"))
	})

	t.Run("expired session", func(t *testing.T) {
		a := initializedAdapter(t)
		a.sessions["+5511999999999"] = &SessionInfo{
			ContactID:             "+5511999999999",
			CanSendSessionMessage: true,
			SessionExpiresAt:      time.Now().Add(-1 * time.Hour),
		}
		assert.False(t, a.IsSessionValid("+5511999999999"))
	})
}

// ---------------------------------------------------------------------------
// 6. TestAdapter_updateSession
// ---------------------------------------------------------------------------

func TestAdapter_updateSession(t *testing.T) {
	t.Run("new phone creates session", func(t *testing.T) {
		a := initializedAdapter(t)
		before := time.Now()

		a.updateSession("+5511999999999")

		session, ok := a.sessions["+5511999999999"]
		require.True(t, ok)
		assert.True(t, session.CanSendSessionMessage)
		assert.Equal(t, "+5511999999999", session.ContactID)
		assert.WithinDuration(t, before.Add(24*time.Hour), session.SessionExpiresAt, 2*time.Second)
		assert.WithinDuration(t, before, session.LastCustomerMessageAt, 2*time.Second)
	})

	t.Run("existing phone updates session", func(t *testing.T) {
		a := initializedAdapter(t)
		oldTime := time.Now().Add(-12 * time.Hour)
		a.sessions["+5511999999999"] = &SessionInfo{
			ContactID:             "+5511999999999",
			LastCustomerMessageAt: oldTime,
			SessionExpiresAt:      oldTime.Add(24 * time.Hour),
			CanSendSessionMessage: true,
		}

		before := time.Now()
		a.updateSession("+5511999999999")

		session := a.sessions["+5511999999999"]
		assert.WithinDuration(t, before, session.LastCustomerMessageAt, 2*time.Second)
		assert.WithinDuration(t, before.Add(24*time.Hour), session.SessionExpiresAt, 2*time.Second)
	})
}

// ---------------------------------------------------------------------------
// 7. TestAdapter_VerifyWebhookChallenge
// ---------------------------------------------------------------------------

func TestAdapter_VerifyWebhookChallenge(t *testing.T) {
	a := initializedAdapter(t)

	t.Run("correct mode and token", func(t *testing.T) {
		challenge, ok := a.VerifyWebhookChallenge("subscribe", "my-verify-token", "challenge-xyz")
		assert.True(t, ok)
		assert.Equal(t, "challenge-xyz", challenge)
	})

	t.Run("wrong token", func(t *testing.T) {
		challenge, ok := a.VerifyWebhookChallenge("subscribe", "wrong-token", "challenge-xyz")
		assert.False(t, ok)
		assert.Equal(t, "", challenge)
	})

	t.Run("wrong mode", func(t *testing.T) {
		challenge, ok := a.VerifyWebhookChallenge("unsubscribe", "my-verify-token", "challenge-xyz")
		assert.False(t, ok)
		assert.Equal(t, "", challenge)
	})
}

// ---------------------------------------------------------------------------
// 8. TestAdapter_HandleWebhook
// ---------------------------------------------------------------------------

func TestAdapter_HandleWebhook(t *testing.T) {
	makeWebhookBody := func(messages []IncomingMessage, statuses []StatusUpdate) []byte {
		payload := WebhookPayload{
			Object: "whatsapp_business_account",
			Entry: []WebhookEntry{
				{
					ID: "entry1",
					Changes: []WebhookChange{
						{
							Field: "messages",
							Value: WebhookChangeValue{
								MessagingProduct: "whatsapp",
								Metadata: WebhookMetadata{
									DisplayPhoneNumber: "+5511000000000",
									PhoneNumberID:      "123456789",
								},
								Contacts: []ContactInfo{
									{WaID: "5511999999999", Profile: ContactProfile{Name: "Test User"}},
								},
								Messages: messages,
								Statuses: statuses,
							},
						},
					},
				},
			},
		}
		b, _ := json.Marshal(payload)
		return b
	}

	t.Run("without processor returns error", func(t *testing.T) {
		a := initializedAdapter(t)
		// webhookProcessor is nil since we did not call Connect
		err := a.HandleWebhook(context.Background(), []byte(`{}`))
		require.Error(t, err)
		assert.Contains(t, err.Error(), "webhook processor not initialized")
	})

	t.Run("message handler called", func(t *testing.T) {
		a := initializedAdapter(t)
		a.webhookProcessor = NewWebhookProcessor(a.config)

		var received []*plugin.InboundMessage
		a.SetMessageHandler(func(ctx context.Context, msg *plugin.InboundMessage) error {
			received = append(received, msg)
			return nil
		})

		body := makeWebhookBody(
			[]IncomingMessage{
				{
					ID:        "wamid.abc",
					From:      "5511999999999",
					Timestamp: fmt.Sprintf("%d", time.Now().Unix()),
					Type:      MessageTypeText,
					Text:      &TextContent{Body: "Hello from test"},
				},
			},
			nil,
		)

		err := a.HandleWebhook(context.Background(), body)
		require.NoError(t, err)
		require.Len(t, received, 1)
		assert.Equal(t, "Hello from test", received[0].Content)
		assert.Equal(t, "5511999999999", received[0].SenderID)
		assert.Equal(t, "Test User", received[0].SenderName)

		// Verify session was updated
		assert.True(t, a.IsSessionValid("5511999999999"))
	})

	t.Run("status handler called", func(t *testing.T) {
		a := initializedAdapter(t)
		a.webhookProcessor = NewWebhookProcessor(a.config)

		var received []*plugin.StatusCallback
		a.SetStatusHandler(func(ctx context.Context, status *plugin.StatusCallback) error {
			received = append(received, status)
			return nil
		})

		body := makeWebhookBody(
			nil,
			[]StatusUpdate{
				{
					ID:          "wamid.sent1",
					RecipientID: "5511999999999",
					Status:      StatusDelivered,
					Timestamp:   fmt.Sprintf("%d", time.Now().Unix()),
				},
			},
		)

		err := a.HandleWebhook(context.Background(), body)
		require.NoError(t, err)
		require.Len(t, received, 1)
		assert.Equal(t, plugin.MessageStatusDelivered, received[0].Status)
		assert.Equal(t, "wamid.sent1", received[0].ExternalID)
	})

	t.Run("handler error continues processing", func(t *testing.T) {
		a := initializedAdapter(t)
		a.webhookProcessor = NewWebhookProcessor(a.config)

		callCount := 0
		a.SetMessageHandler(func(ctx context.Context, msg *plugin.InboundMessage) error {
			callCount++
			if callCount == 1 {
				return fmt.Errorf("handler error")
			}
			return nil
		})

		body := makeWebhookBody(
			[]IncomingMessage{
				{
					ID:        "wamid.1",
					From:      "5511999999999",
					Timestamp: fmt.Sprintf("%d", time.Now().Unix()),
					Type:      MessageTypeText,
					Text:      &TextContent{Body: "msg1"},
				},
				{
					ID:        "wamid.2",
					From:      "5511888888888",
					Timestamp: fmt.Sprintf("%d", time.Now().Unix()),
					Type:      MessageTypeText,
					Text:      &TextContent{Body: "msg2"},
				},
			},
			nil,
		)

		err := a.HandleWebhook(context.Background(), body)
		require.NoError(t, err)
		assert.Equal(t, 2, callCount, "handler should have been called for both messages")
	})

	t.Run("invalid payload returns error", func(t *testing.T) {
		a := initializedAdapter(t)
		a.webhookProcessor = NewWebhookProcessor(a.config)

		err := a.HandleWebhook(context.Background(), []byte(`{invalid`))
		require.Error(t, err)
	})

	t.Run("wrong object returns error", func(t *testing.T) {
		a := initializedAdapter(t)
		a.webhookProcessor = NewWebhookProcessor(a.config)

		body := []byte(`{"object":"page","entry":[]}`)
		err := a.HandleWebhook(context.Background(), body)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid webhook object")
	})
}

// ---------------------------------------------------------------------------
// 9. TestAdapter_GetConnectionStatus
// ---------------------------------------------------------------------------

func TestAdapter_GetConnectionStatus(t *testing.T) {
	t.Run("connected", func(t *testing.T) {
		a := initializedAdapter(t)
		a.SetConnected(true)

		status := a.GetConnectionStatus()
		assert.True(t, status.Connected)
		assert.Equal(t, "connected", status.Status)
		assert.Equal(t, "123456789", status.Metadata["phone_number_id"])
		assert.Equal(t, DefaultAPIVersion, status.Metadata["api_version"])
	})

	t.Run("disconnected", func(t *testing.T) {
		a := initializedAdapter(t)
		// Not connected by default

		status := a.GetConnectionStatus()
		assert.False(t, status.Connected)
		assert.Equal(t, "disconnected", status.Status)
	})
}

// ---------------------------------------------------------------------------
// 10. TestAdapter_SetMessageHandler / TestAdapter_SetStatusHandler
// ---------------------------------------------------------------------------

func TestAdapter_SetMessageHandler(t *testing.T) {
	a := initializedAdapter(t)

	called := false
	a.SetMessageHandler(func(ctx context.Context, msg *plugin.InboundMessage) error {
		called = true
		return nil
	})

	require.NotNil(t, a.messageHandler)
	err := a.messageHandler(context.Background(), &plugin.InboundMessage{})
	require.NoError(t, err)
	assert.True(t, called)
}

func TestAdapter_SetStatusHandler(t *testing.T) {
	a := initializedAdapter(t)

	called := false
	a.SetStatusHandler(func(ctx context.Context, status *plugin.StatusCallback) error {
		called = true
		return nil
	})

	require.NotNil(t, a.statusHandler)
	err := a.statusHandler(context.Background(), &plugin.StatusCallback{})
	require.NoError(t, err)
	assert.True(t, called)
}

// ---------------------------------------------------------------------------
// 11. TestAdapter_GetWebhookPath
// ---------------------------------------------------------------------------

func TestAdapter_GetWebhookPath(t *testing.T) {
	a := NewAdapter()
	assert.Equal(t, "/api/v1/webhooks/whatsapp", a.GetWebhookPath())
}

// ---------------------------------------------------------------------------
// 12. TestSessionInfo
// ---------------------------------------------------------------------------

func TestSessionInfo(t *testing.T) {
	t.Run("valid session", func(t *testing.T) {
		s := &SessionInfo{
			CanSendSessionMessage: true,
			SessionExpiresAt:      time.Now().Add(1 * time.Hour),
		}
		assert.True(t, s.IsSessionValid())
	})

	t.Run("expired session", func(t *testing.T) {
		s := &SessionInfo{
			CanSendSessionMessage: true,
			SessionExpiresAt:      time.Now().Add(-1 * time.Hour),
		}
		assert.False(t, s.IsSessionValid())
	})

	t.Run("CanSendSessionMessage false", func(t *testing.T) {
		s := &SessionInfo{
			CanSendSessionMessage: false,
			SessionExpiresAt:      time.Now().Add(1 * time.Hour),
		}
		assert.False(t, s.IsSessionValid())
	})

	t.Run("UpdateSession sets fields", func(t *testing.T) {
		s := &SessionInfo{ContactID: "phone1"}
		before := time.Now()
		s.UpdateSession()

		assert.True(t, s.CanSendSessionMessage)
		assert.WithinDuration(t, before, s.LastCustomerMessageAt, 2*time.Second)
		assert.WithinDuration(t, before.Add(24*time.Hour), s.SessionExpiresAt, 2*time.Second)
	})
}

// ---------------------------------------------------------------------------
// 13. Test getOrDefault helper
// ---------------------------------------------------------------------------

func Test_getOrDefault(t *testing.T) {
	tests := []struct {
		name         string
		config       map[string]string
		key          string
		defaultValue string
		want         string
	}{
		{
			name:         "key exists with value",
			config:       map[string]string{"key": "value"},
			key:          "key",
			defaultValue: "default",
			want:         "value",
		},
		{
			name:         "key exists but empty",
			config:       map[string]string{"key": ""},
			key:          "key",
			defaultValue: "default",
			want:         "default",
		},
		{
			name:         "key missing",
			config:       map[string]string{},
			key:          "key",
			defaultValue: "default",
			want:         "default",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := getOrDefault(tc.config, tc.key, tc.defaultValue)
			assert.Equal(t, tc.want, got)
		})
	}
}
