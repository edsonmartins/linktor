package instagram

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/msgfy/linktor/internal/adapters/meta"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIsInstagramViaPageWebhook(t *testing.T) {
	messengerPayload := &meta.WebhookPayload{
		Object: "page",
		Entry: []meta.WebhookEntry{
			{
				ID: "page-999",
				Messaging: []meta.MessagingEvent{
					{
						Sender:    meta.MessagingParty{ID: "user-1"},
						Recipient: meta.MessagingParty{ID: "page-999"},
						Message:   &meta.InboundMessage{MID: "mid.1", Text: "hi"},
					},
				},
			},
		},
	}

	igViaPagePayload := &meta.WebhookPayload{
		Object: "page",
		Entry: []meta.WebhookEntry{
			{
				ID: "ig-123",
				Messaging: []meta.MessagingEvent{
					{
						Sender:    meta.MessagingParty{ID: "user-1"},
						Recipient: meta.MessagingParty{ID: "ig-123"},
						Message:   &meta.InboundMessage{MID: "mid.2", Text: "hi ig"},
					},
				},
			},
		},
	}

	t.Run("rejects pure Messenger payload when instagram_id given", func(t *testing.T) {
		assert.False(t, IsInstagramViaPageWebhook(messengerPayload, "ig-123"))
	})

	t.Run("accepts genuine IG-via-page payload", func(t *testing.T) {
		assert.True(t, IsInstagramViaPageWebhook(igViaPagePayload, "ig-123"))
	})

	t.Run("non-page object is never IG-via-page", func(t *testing.T) {
		assert.False(t, IsInstagramViaPageWebhook(&meta.WebhookPayload{Object: "instagram"}, "ig-123"))
	})

	t.Run("legacy no-id call keeps heuristic", func(t *testing.T) {
		assert.True(t, IsInstagramViaPageWebhook(messengerPayload))
	})
}

func TestExtractMessages_ReactionsAndStories(t *testing.T) {
	t.Run("reaction is surfaced", func(t *testing.T) {
		payload := &meta.WebhookPayload{
			Object: "instagram",
			Entry: []meta.WebhookEntry{
				{
					ID: "ig-1",
					Messaging: []meta.MessagingEvent{
						{
							Sender:    meta.MessagingParty{ID: "user-1"},
							Recipient: meta.MessagingParty{ID: "ig-1"},
							Reaction: &meta.ReactionEvent{
								MID:    "mid.reacted",
								Action: "react",
								Emoji:  "❤️",
							},
						},
					},
				},
			},
		}

		msgs := ExtractMessages(payload)
		require.Len(t, msgs, 1)
		assert.Equal(t, EventTypeReaction, msgs[0].EventType)
		assert.Equal(t, "❤️", msgs[0].ReactionEmoji)
		assert.Equal(t, "❤️", msgs[0].Text)
	})

	t.Run("story reply carries story ref and text", func(t *testing.T) {
		payload := &meta.WebhookPayload{
			Object: "instagram",
			Entry: []meta.WebhookEntry{
				{
					ID: "ig-1",
					Messaging: []meta.MessagingEvent{
						{
							Sender:    meta.MessagingParty{ID: "user-1"},
							Recipient: meta.MessagingParty{ID: "ig-1"},
							Message: &meta.InboundMessage{
								MID:  "mid.storyreply",
								Text: "nice story!",
								ReplyTo: &meta.ReplyTo{
									Story: &meta.StoryRef{ID: "story-1", URL: "https://cdn/story.jpg"},
								},
							},
						},
					},
				},
			},
		}

		msgs := ExtractMessages(payload)
		require.Len(t, msgs, 1)
		assert.Equal(t, EventTypeStoryReply, msgs[0].EventType)
		assert.Equal(t, "story-1", msgs[0].StoryID)
		assert.Equal(t, "nice story!", msgs[0].Text)
	})
}

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

func TestWebhookHandler_ParseWebhook_FailClosed(t *testing.T) {
	t.Run("no app secret rejects payload", func(t *testing.T) {
		h := NewWebhookHandler("", "verify-tok")
		req := httptest.NewRequest(http.MethodPost, "/webhook", strings.NewReader(`{"object":"instagram"}`))
		_, err := h.ParseWebhook(req)
		assert.Equal(t, ErrInvalidSignature, err)
	})

	t.Run("invalid signature rejected", func(t *testing.T) {
		h := NewWebhookHandler("secret", "verify-tok")
		req := httptest.NewRequest(http.MethodPost, "/webhook", strings.NewReader(`{"object":"instagram"}`))
		req.Header.Set("X-Hub-Signature-256", "sha256=deadbeef")
		_, err := h.ParseWebhook(req)
		assert.Equal(t, ErrInvalidSignature, err)
	})
}

func TestWebhookError(t *testing.T) {
	err := &WebhookError{Code: "test_code", Message: "test message"}
	assert.Equal(t, "test message", err.Error())
}
