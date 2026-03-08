package facebook

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/msgfy/linktor/internal/adapters/meta"
	"github.com/msgfy/linktor/pkg/plugin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewWebhookHandler(t *testing.T) {
	h := NewWebhookHandler("secret", "verify-tok")
	assert.NotNil(t, h)
	assert.Equal(t, "secret", h.appSecret)
	assert.Equal(t, "verify-tok", h.verifyToken)
}

func TestWebhookHandler_VerifyWebhook(t *testing.T) {
	h := NewWebhookHandler("secret", "my-verify-token")

	t.Run("valid verification", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/webhook?hub.mode=subscribe&hub.verify_token=my-verify-token&hub.challenge=challenge123", nil)
		challenge, err := h.VerifyWebhook(req)
		require.NoError(t, err)
		assert.Equal(t, "challenge123", challenge)
	})

	t.Run("invalid token", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/webhook?hub.mode=subscribe&hub.verify_token=wrong-token&hub.challenge=challenge123", nil)
		_, err := h.VerifyWebhook(req)
		assert.Equal(t, ErrInvalidVerifyToken, err)
	})

	t.Run("wrong mode", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/webhook?hub.mode=unsubscribe&hub.verify_token=my-verify-token&hub.challenge=challenge123", nil)
		_, err := h.VerifyWebhook(req)
		assert.Equal(t, ErrInvalidVerifyToken, err)
	})

	t.Run("missing params", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/webhook", nil)
		_, err := h.VerifyWebhook(req)
		assert.Equal(t, ErrInvalidVerifyToken, err)
	})
}

func TestIsMessengerWebhook(t *testing.T) {
	t.Run("page object", func(t *testing.T) {
		payload := &meta.WebhookPayload{Object: "page"}
		assert.True(t, IsMessengerWebhook(payload))
	})

	t.Run("instagram object", func(t *testing.T) {
		payload := &meta.WebhookPayload{Object: "instagram"}
		assert.False(t, IsMessengerWebhook(payload))
	})

	t.Run("empty object", func(t *testing.T) {
		payload := &meta.WebhookPayload{}
		assert.False(t, IsMessengerWebhook(payload))
	})
}

func TestGetPageIDFromPayload(t *testing.T) {
	t.Run("with entries", func(t *testing.T) {
		payload := &meta.WebhookPayload{
			Entry: []meta.WebhookEntry{{ID: "page-123"}},
		}
		assert.Equal(t, "page-123", GetPageIDFromPayload(payload))
	})

	t.Run("empty entries", func(t *testing.T) {
		payload := &meta.WebhookPayload{}
		assert.Equal(t, "", GetPageIDFromPayload(payload))
	})
}

func TestExtractMessages(t *testing.T) {
	t.Run("text message", func(t *testing.T) {
		payload := &meta.WebhookPayload{
			Object: "page",
			Entry: []meta.WebhookEntry{
				{
					ID: "page-1",
					Messaging: []meta.MessagingEvent{
						{
							Sender:    meta.MessagingParty{ID: "user-1"},
							Recipient: meta.MessagingParty{ID: "page-1"},
							Timestamp: 1700000000000,
							Message: &meta.InboundMessage{
								MID:  "mid.123",
								Text: "Hello!",
							},
						},
					},
				},
			},
		}

		messages := ExtractMessages(payload)
		require.Len(t, messages, 1)
		assert.Equal(t, "mid.123", messages[0].ExternalID)
		assert.Equal(t, "user-1", messages[0].SenderID)
		assert.Equal(t, "page-1", messages[0].RecipientID)
		assert.Equal(t, "page-1", messages[0].PageID)
		assert.Equal(t, "Hello!", messages[0].Text)
		assert.False(t, messages[0].IsEcho)
	})

	t.Run("echo message", func(t *testing.T) {
		payload := &meta.WebhookPayload{
			Object: "page",
			Entry: []meta.WebhookEntry{
				{
					ID: "page-1",
					Messaging: []meta.MessagingEvent{
						{
							Sender:    meta.MessagingParty{ID: "page-1"},
							Recipient: meta.MessagingParty{ID: "user-1"},
							Message: &meta.InboundMessage{
								MID:    "mid.echo",
								Text:   "Echo!",
								IsEcho: true,
							},
						},
					},
				},
			},
		}

		messages := ExtractMessages(payload)
		require.Len(t, messages, 1)
		assert.True(t, messages[0].IsEcho)
	})

	t.Run("message with attachment", func(t *testing.T) {
		payload := &meta.WebhookPayload{
			Object: "page",
			Entry: []meta.WebhookEntry{
				{
					ID: "page-1",
					Messaging: []meta.MessagingEvent{
						{
							Sender:    meta.MessagingParty{ID: "user-1"},
							Recipient: meta.MessagingParty{ID: "page-1"},
							Message: &meta.InboundMessage{
								MID: "mid.img",
								Attachments: []meta.InboundAttachment{
									{
										Type: "image",
										Payload: meta.AttachmentPayload{
											URL: "https://example.com/photo.jpg",
										},
									},
								},
							},
						},
					},
				},
			},
		}

		messages := ExtractMessages(payload)
		require.Len(t, messages, 1)
		require.Len(t, messages[0].Attachments, 1)
		assert.Equal(t, "image", messages[0].Attachments[0].Type)
		assert.Equal(t, "https://example.com/photo.jpg", messages[0].Attachments[0].URL)
	})

	t.Run("message with location", func(t *testing.T) {
		payload := &meta.WebhookPayload{
			Object: "page",
			Entry: []meta.WebhookEntry{
				{
					ID: "page-1",
					Messaging: []meta.MessagingEvent{
						{
							Sender:    meta.MessagingParty{ID: "user-1"},
							Recipient: meta.MessagingParty{ID: "page-1"},
							Message: &meta.InboundMessage{
								MID: "mid.loc",
								Attachments: []meta.InboundAttachment{
									{
										Type: "location",
										Payload: meta.AttachmentPayload{
											Coordinates: &meta.Coordinates{
												Lat:  40.7128,
												Long: -74.0060,
											},
										},
									},
								},
							},
						},
					},
				},
			},
		}

		messages := ExtractMessages(payload)
		require.Len(t, messages, 1)
		require.Len(t, messages[0].Attachments, 1)
		assert.Equal(t, "location", messages[0].Attachments[0].Type)
		assert.InDelta(t, 40.7128, messages[0].Attachments[0].Lat, 0.001)
		assert.InDelta(t, -74.006, messages[0].Attachments[0].Long, 0.001)
	})

	t.Run("message with quick reply", func(t *testing.T) {
		payload := &meta.WebhookPayload{
			Object: "page",
			Entry: []meta.WebhookEntry{
				{
					ID: "page-1",
					Messaging: []meta.MessagingEvent{
						{
							Sender:    meta.MessagingParty{ID: "user-1"},
							Recipient: meta.MessagingParty{ID: "page-1"},
							Message: &meta.InboundMessage{
								MID:  "mid.qr",
								Text: "Yes",
								QuickReply: &meta.QuickReplyPayload{
									Payload: "YES_PAYLOAD",
								},
							},
						},
					},
				},
			},
		}

		messages := ExtractMessages(payload)
		require.Len(t, messages, 1)
		assert.Equal(t, "YES_PAYLOAD", messages[0].QuickReply)
	})

	t.Run("standby messages", func(t *testing.T) {
		payload := &meta.WebhookPayload{
			Object: "page",
			Entry: []meta.WebhookEntry{
				{
					ID: "page-1",
					Standby: []meta.MessagingEvent{
						{
							Sender:    meta.MessagingParty{ID: "user-1"},
							Recipient: meta.MessagingParty{ID: "page-1"},
							Message: &meta.InboundMessage{
								MID:  "mid.standby",
								Text: "Standby message",
							},
						},
					},
				},
			},
		}

		messages := ExtractMessages(payload)
		require.Len(t, messages, 1)
		assert.Equal(t, "Standby message", messages[0].Text)
	})

	t.Run("no message event", func(t *testing.T) {
		payload := &meta.WebhookPayload{
			Object: "page",
			Entry: []meta.WebhookEntry{
				{
					ID: "page-1",
					Messaging: []meta.MessagingEvent{
						{
							Sender:    meta.MessagingParty{ID: "user-1"},
							Recipient: meta.MessagingParty{ID: "page-1"},
							Delivery: &meta.DeliveryStatus{
								MIDs:      []string{"mid.1"},
								Watermark: 1700000000000,
							},
						},
					},
				},
			},
		}

		messages := ExtractMessages(payload)
		assert.Empty(t, messages)
	})
}

func TestExtractDeliveryStatuses(t *testing.T) {
	t.Run("with delivery", func(t *testing.T) {
		payload := &meta.WebhookPayload{
			Entry: []meta.WebhookEntry{
				{
					Messaging: []meta.MessagingEvent{
						{
							Delivery: &meta.DeliveryStatus{
								MIDs:      []string{"mid.1", "mid.2"},
								Watermark: 1700000000000,
							},
						},
					},
				},
			},
		}

		statuses := ExtractDeliveryStatuses(payload)
		require.Len(t, statuses, 1)
		assert.Equal(t, []string{"mid.1", "mid.2"}, statuses[0].MessageIDs)
		assert.Equal(t, time.UnixMilli(1700000000000), statuses[0].Watermark)
	})

	t.Run("no delivery", func(t *testing.T) {
		payload := &meta.WebhookPayload{
			Entry: []meta.WebhookEntry{
				{
					Messaging: []meta.MessagingEvent{
						{
							Message: &meta.InboundMessage{MID: "mid.1"},
						},
					},
				},
			},
		}

		statuses := ExtractDeliveryStatuses(payload)
		assert.Empty(t, statuses)
	})
}

func TestExtractReadStatuses(t *testing.T) {
	t.Run("with read", func(t *testing.T) {
		payload := &meta.WebhookPayload{
			Entry: []meta.WebhookEntry{
				{
					Messaging: []meta.MessagingEvent{
						{
							Read: &meta.ReadStatus{
								Watermark: 1700000000000,
							},
						},
					},
				},
			},
		}

		statuses := ExtractReadStatuses(payload)
		require.Len(t, statuses, 1)
		assert.Equal(t, time.UnixMilli(1700000000000), statuses[0].Watermark)
	})

	t.Run("no read", func(t *testing.T) {
		payload := &meta.WebhookPayload{
			Entry: []meta.WebhookEntry{
				{
					Messaging: []meta.MessagingEvent{
						{Message: &meta.InboundMessage{MID: "mid.1"}},
					},
				},
			},
		}

		statuses := ExtractReadStatuses(payload)
		assert.Empty(t, statuses)
	})
}

func TestExtractPostbacks(t *testing.T) {
	t.Run("with postback", func(t *testing.T) {
		payload := &meta.WebhookPayload{
			Entry: []meta.WebhookEntry{
				{
					Messaging: []meta.MessagingEvent{
						{
							Postback: &meta.Postback{
								Title:   "Get Started",
								Payload: "GET_STARTED",
							},
						},
					},
				},
			},
		}

		postbacks := ExtractPostbacks(payload)
		require.Len(t, postbacks, 1)
		assert.Equal(t, "Get Started", postbacks[0].Title)
		assert.Equal(t, "GET_STARTED", postbacks[0].Payload)
	})

	t.Run("no postback", func(t *testing.T) {
		payload := &meta.WebhookPayload{
			Entry: []meta.WebhookEntry{
				{
					Messaging: []meta.MessagingEvent{
						{Message: &meta.InboundMessage{MID: "mid.1"}},
					},
				},
			},
		}

		postbacks := ExtractPostbacks(payload)
		assert.Empty(t, postbacks)
	})
}

func TestConvertIncomingMessage(t *testing.T) {
	t.Run("nil message", func(t *testing.T) {
		event := &meta.MessagingEvent{
			Sender: meta.MessagingParty{ID: "user-1"},
		}
		assert.Nil(t, ConvertIncomingMessage(event, "page-1"))
	})

	t.Run("text message with timestamp", func(t *testing.T) {
		event := &meta.MessagingEvent{
			Sender:    meta.MessagingParty{ID: "user-1"},
			Recipient: meta.MessagingParty{ID: "page-1"},
			Timestamp: 1700000000000,
			Message: &meta.InboundMessage{
				MID:  "mid.123",
				Text: "Hi there",
			},
		}

		msg := ConvertIncomingMessage(event, "page-1")
		require.NotNil(t, msg)
		assert.Equal(t, "mid.123", msg.ExternalID)
		assert.Equal(t, "user-1", msg.SenderID)
		assert.Equal(t, "page-1", msg.RecipientID)
		assert.Equal(t, "page-1", msg.PageID)
		assert.Equal(t, "Hi there", msg.Text)
		assert.Equal(t, time.UnixMilli(1700000000000), msg.Timestamp)
	})
}

func TestConvertDeliveryStatus(t *testing.T) {
	t.Run("nil delivery", func(t *testing.T) {
		event := &meta.MessagingEvent{}
		assert.Nil(t, ConvertDeliveryStatus(event))
	})

	t.Run("valid delivery", func(t *testing.T) {
		event := &meta.MessagingEvent{
			Delivery: &meta.DeliveryStatus{
				MIDs:      []string{"mid.1"},
				Watermark: 1700000000000,
			},
		}
		status := ConvertDeliveryStatus(event)
		require.NotNil(t, status)
		assert.Equal(t, []string{"mid.1"}, status.MessageIDs)
	})
}

func TestConvertReadStatus(t *testing.T) {
	t.Run("nil read", func(t *testing.T) {
		event := &meta.MessagingEvent{}
		assert.Nil(t, ConvertReadStatus(event))
	})

	t.Run("valid read", func(t *testing.T) {
		event := &meta.MessagingEvent{
			Read: &meta.ReadStatus{Watermark: 1700000000000},
		}
		status := ConvertReadStatus(event)
		require.NotNil(t, status)
		assert.Equal(t, time.UnixMilli(1700000000000), status.Watermark)
	})
}

func TestConvertPostback(t *testing.T) {
	t.Run("nil postback", func(t *testing.T) {
		event := &meta.MessagingEvent{}
		assert.Nil(t, ConvertPostback(event))
	})

	t.Run("valid postback", func(t *testing.T) {
		event := &meta.MessagingEvent{
			Postback: &meta.Postback{
				Title:   "Help",
				Payload: "HELP_PAYLOAD",
			},
		}
		pb := ConvertPostback(event)
		require.NotNil(t, pb)
		assert.Equal(t, "Help", pb.Title)
		assert.Equal(t, "HELP_PAYLOAD", pb.Payload)
	})
}

func TestMsToTime(t *testing.T) {
	ts := msToTime(1700000000000)
	assert.Equal(t, time.UnixMilli(1700000000000), ts)
}

func TestWebhookError(t *testing.T) {
	err := &WebhookError{Code: "test_code", Message: "test message"}
	assert.Equal(t, "test message", err.Error())
}

func TestProcessWebhook(t *testing.T) {
	a := NewAdapter()
	a.config = &FacebookConfig{PageID: "page-1"}

	t.Run("messenger webhook", func(t *testing.T) {
		payload := &meta.WebhookPayload{
			Object: "page",
			Entry: []meta.WebhookEntry{
				{
					ID: "page-1",
					Messaging: []meta.MessagingEvent{
						{
							Sender:    meta.MessagingParty{ID: "user-1"},
							Recipient: meta.MessagingParty{ID: "page-1"},
							Message: &meta.InboundMessage{
								MID:  "mid.1",
								Text: "Hello",
							},
						},
					},
				},
			},
		}

		messages := a.ProcessWebhook(payload)
		require.Len(t, messages, 1)
		assert.Equal(t, "user-1", messages[0].SenderID)
		assert.Equal(t, "Hello", messages[0].Content)
		assert.Equal(t, plugin.ContentTypeText, messages[0].ContentType)
		assert.Equal(t, "page-1", messages[0].Metadata["page_id"])
	})

	t.Run("non-messenger webhook", func(t *testing.T) {
		payload := &meta.WebhookPayload{Object: "instagram"}
		messages := a.ProcessWebhook(payload)
		assert.Nil(t, messages)
	})

	t.Run("echo messages skipped", func(t *testing.T) {
		payload := &meta.WebhookPayload{
			Object: "page",
			Entry: []meta.WebhookEntry{
				{
					ID: "page-1",
					Messaging: []meta.MessagingEvent{
						{
							Sender:    meta.MessagingParty{ID: "page-1"},
							Recipient: meta.MessagingParty{ID: "user-1"},
							Message: &meta.InboundMessage{
								MID:    "mid.echo",
								Text:   "Echo",
								IsEcho: true,
							},
						},
					},
				},
			},
		}

		messages := a.ProcessWebhook(payload)
		assert.Empty(t, messages)
	})

	t.Run("with quick reply", func(t *testing.T) {
		payload := &meta.WebhookPayload{
			Object: "page",
			Entry: []meta.WebhookEntry{
				{
					ID: "page-1",
					Messaging: []meta.MessagingEvent{
						{
							Sender:    meta.MessagingParty{ID: "user-1"},
							Recipient: meta.MessagingParty{ID: "page-1"},
							Message: &meta.InboundMessage{
								MID:        "mid.qr",
								Text:       "Option A",
								QuickReply: &meta.QuickReplyPayload{Payload: "OPTION_A"},
							},
						},
					},
				},
			},
		}

		messages := a.ProcessWebhook(payload)
		require.Len(t, messages, 1)
		assert.Equal(t, "OPTION_A", messages[0].Metadata["quick_reply"])
	})

	t.Run("with image attachment", func(t *testing.T) {
		payload := &meta.WebhookPayload{
			Object: "page",
			Entry: []meta.WebhookEntry{
				{
					ID: "page-1",
					Messaging: []meta.MessagingEvent{
						{
							Sender:    meta.MessagingParty{ID: "user-1"},
							Recipient: meta.MessagingParty{ID: "page-1"},
							Message: &meta.InboundMessage{
								MID: "mid.img",
								Attachments: []meta.InboundAttachment{
									{
										Type:    "image",
										Payload: meta.AttachmentPayload{URL: "https://example.com/img.jpg"},
									},
								},
							},
						},
					},
				},
			},
		}

		messages := a.ProcessWebhook(payload)
		require.Len(t, messages, 1)
		assert.Equal(t, plugin.ContentTypeImage, messages[0].ContentType)
		require.Len(t, messages[0].Attachments, 1)
		assert.Equal(t, "image", messages[0].Attachments[0].Type)
		assert.Equal(t, "https://example.com/img.jpg", messages[0].Attachments[0].URL)
	})

	t.Run("with location attachment", func(t *testing.T) {
		payload := &meta.WebhookPayload{
			Object: "page",
			Entry: []meta.WebhookEntry{
				{
					ID: "page-1",
					Messaging: []meta.MessagingEvent{
						{
							Sender:    meta.MessagingParty{ID: "user-1"},
							Recipient: meta.MessagingParty{ID: "page-1"},
							Message: &meta.InboundMessage{
								MID: "mid.loc",
								Attachments: []meta.InboundAttachment{
									{
										Type: "location",
										Payload: meta.AttachmentPayload{
											Coordinates: &meta.Coordinates{
												Lat:  40.7128,
												Long: -74.0060,
											},
										},
									},
								},
							},
						},
					},
				},
			},
		}

		messages := a.ProcessWebhook(payload)
		require.Len(t, messages, 1)
		assert.Equal(t, plugin.ContentTypeLocation, messages[0].ContentType)
		assert.Contains(t, messages[0].Metadata["lat"], "40.71")
		assert.Contains(t, messages[0].Metadata["long"], "-74.00")
	})
}

func TestOAuthHelper(t *testing.T) {
	t.Run("new helper", func(t *testing.T) {
		h := NewOAuthHelper("app-123", "secret-456")
		assert.Equal(t, "app-123", h.AppID)
		assert.Equal(t, "secret-456", h.AppSecret)
	})

	t.Run("login URL with default scopes", func(t *testing.T) {
		h := NewOAuthHelper("app-123", "secret")
		url := h.GetLoginURL("https://example.com/callback", "state123", nil)
		assert.Contains(t, url, "client_id=app-123")
		assert.Contains(t, url, "redirect_uri=https://example.com/callback")
		assert.Contains(t, url, "state=state123")
		assert.Contains(t, url, "pages_messaging")
		assert.Contains(t, url, "pages_read_engagement")
		assert.Contains(t, url, "pages_manage_metadata")
	})

	t.Run("login URL with custom scopes", func(t *testing.T) {
		h := NewOAuthHelper("app-123", "secret")
		url := h.GetLoginURL("https://example.com/callback", "state123", []string{"email", "public_profile"})
		assert.Contains(t, url, "scope=email,public_profile")
		assert.NotContains(t, url, "pages_messaging")
	})
}
