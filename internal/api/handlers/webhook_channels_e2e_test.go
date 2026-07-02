package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/msgfy/linktor/internal/adapters/facebook"
	whatsappofficial "github.com/msgfy/linktor/internal/adapters/whatsapp_official"
	"github.com/msgfy/linktor/internal/domain/entity"
	"github.com/msgfy/linktor/pkg/testutil"
)

// These end-to-end tests lock in the homologation fixes for the in-scope Meta
// channels (Facebook Messenger, Instagram DM) and the WhatsApp `failed`-status
// batch-parsing regression. They drive the real gin handlers with realistic,
// signed provider payloads and assert on the normalized InboundMessage(s) the
// handler publishes to NATS. TEST-ONLY: no production code is modified.

// addChannel registers a channel in the mock repo used by setupWebhookTest.
func addChannel(repo *testutil.MockChannelRepository, id string, ctype entity.ChannelType, creds map[string]string) *entity.Channel {
	ch := &entity.Channel{
		ID:               id,
		TenantID:         "tenant-1",
		Type:             ctype,
		Name:             string(ctype),
		Enabled:          true,
		ConnectionStatus: entity.ConnectionStatusConnected,
		Credentials:      creds,
	}
	repo.Channels[id] = ch
	return ch
}

// postSignedTo builds a signed POST context targeting the given channel id. When
// signature is non-empty it is sent verbatim as X-Hub-Signature-256; otherwise a
// valid HMAC-SHA256 of the body is computed with secret.
func postSignedTo(channelID string, payload interface{}, secret, signature string) (*httptest.ResponseRecorder, *gin.Context) {
	body, _ := json.Marshal(payload)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	req := httptest.NewRequest(http.MethodPost, "/webhook/"+channelID, bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	if signature != "" {
		req.Header.Set("X-Hub-Signature-256", signature)
	} else if secret != "" {
		req.Header.Set("X-Hub-Signature-256", computeHMACSHA256(secret, body))
	}
	c.Request = req
	c.Params = []gin.Param{{Key: "channelId", Value: channelID}}
	return w, c
}

// ---------------------------------------------------------------------------
// 1. Facebook Messenger — inbound TEXT + signature enforcement
// ---------------------------------------------------------------------------

func TestWebhookFacebook_InboundTextMessage(t *testing.T) {
	handler, channelRepo, producer, _ := setupWebhookTest()
	addChannel(channelRepo, "ch-fb", entity.ChannelTypeFacebook, map[string]string{
		"app_secret":   "fb-app-secret",
		"verify_token": "fb-verify",
	})

	payload := map[string]interface{}{
		"object": "page",
		"entry": []map[string]interface{}{
			{
				"id":   "page-123",
				"time": 1,
				"messaging": []map[string]interface{}{
					{
						"sender":    map[string]string{"id": "user-999"},
						"recipient": map[string]string{"id": "page-123"},
						"timestamp": int64(1700000000000),
						"message": map[string]interface{}{
							"mid":  "m_abc123",
							"text": "Hello from Messenger",
						},
					},
				},
			},
		},
	}

	w, c := postSignedTo("ch-fb", payload, "fb-app-secret", "")
	handler.FacebookWebhook(c)

	assert.Equal(t, http.StatusOK, w.Code)
	require.Len(t, producer.InboundMessages, 1)
	msg := producer.InboundMessages[0]
	assert.Equal(t, "facebook", msg.ChannelType)
	assert.Equal(t, "tenant-1", msg.TenantID)
	assert.Equal(t, "ch-fb", msg.ChannelID)
	assert.Equal(t, "text", msg.ContentType)
	assert.Equal(t, "Hello from Messenger", msg.Content)
	assert.Equal(t, "m_abc123", msg.ExternalID)
	assert.Equal(t, "user-999", msg.Metadata["sender_id"])
	assert.Equal(t, "page-123", msg.Metadata["page_id"])
}

func TestWebhookFacebook_InvalidSignatureRejected(t *testing.T) {
	handler, channelRepo, producer, _ := setupWebhookTest()
	addChannel(channelRepo, "ch-fb", entity.ChannelTypeFacebook, map[string]string{
		"app_secret":   "fb-app-secret",
		"verify_token": "fb-verify",
	})

	payload := map[string]interface{}{
		"object": "page",
		"entry": []map[string]interface{}{
			{
				"id": "page-123",
				"messaging": []map[string]interface{}{
					{
						"sender":    map[string]string{"id": "user-999"},
						"recipient": map[string]string{"id": "page-123"},
						"timestamp": int64(1700000000000),
						"message":   map[string]interface{}{"mid": "m_abc123", "text": "should be rejected"},
					},
				},
			},
		},
	}

	// Deliberately wrong signature: the handler must reject and publish nothing.
	w, c := postSignedTo("ch-fb", payload, "", "sha256=deadbeef")
	handler.FacebookWebhook(c)

	assert.NotEqual(t, http.StatusOK, w.Code)
	assert.Empty(t, producer.InboundMessages)
}

// ---------------------------------------------------------------------------
// 2. Facebook POSTBACK (button click) — WS6 fix. Button taps arrive as their own
// messaging event (postback, not message). The fix surfaces them as inbound
// events carrying the postback payload as content.
// ---------------------------------------------------------------------------

func TestWebhookFacebook_PostbackBecomesInboundMessage(t *testing.T) {
	// WS6: button taps arrive as their own messaging event carrying a `postback`
	// (not a `message`), and must be surfaced as inbound messages so the flow
	// engine can react to the click. This drives the real HTTP handler
	// (WebhookHandler.FacebookWebhook) end-to-end: a postback-only event must be
	// published as an inbound message with the payload as content.
	handler, channelRepo, producer, _ := setupWebhookTest()
	addChannel(channelRepo, "ch-fb", entity.ChannelTypeFacebook, map[string]string{
		"app_secret":   "fb-app-secret",
		"verify_token": "fb-verify",
	})

	payload := map[string]interface{}{
		"object": "page",
		"entry": []map[string]interface{}{
			{
				"id": "page-123",
				"messaging": []map[string]interface{}{
					{
						"sender":    map[string]string{"id": "user-999"},
						"recipient": map[string]string{"id": "page-123"},
						"timestamp": int64(1700000000000),
						"postback":  map[string]string{"title": "Get Started", "payload": "GET_STARTED_PAYLOAD"},
					},
				},
			},
		},
	}

	w, c := postSignedTo("ch-fb", payload, "fb-app-secret", "")
	handler.FacebookWebhook(c)

	assert.Equal(t, http.StatusOK, w.Code)
	require.Len(t, producer.InboundMessages, 1, "postback must be surfaced as an inbound message by the HTTP handler")
	msg := producer.InboundMessages[0]
	assert.Equal(t, "facebook", msg.ChannelType)
	assert.Equal(t, "GET_STARTED_PAYLOAD", msg.Content)
	assert.Equal(t, "user-999", msg.Metadata["sender_id"])
	assert.Equal(t, "postback", msg.Metadata["event_type"])
	assert.Equal(t, "GET_STARTED_PAYLOAD", msg.Metadata["postback_payload"])
	assert.Equal(t, "Get Started", msg.Metadata["postback_title"])

	// Sanity: the adapter-level extraction agrees with what the handler ingested.
	var wp facebook.WebhookPayload
	body, _ := json.Marshal(payload)
	require.NoError(t, json.Unmarshal(body, &wp))
	require.Len(t, facebook.ExtractPostbacks(&wp), 1)
}

// ---------------------------------------------------------------------------
// 3. Instagram DM — native ("instagram") and via-Page ("page") classification.
// The WS6 classification fix makes the InstagramWebhook accept BOTH shapes.
// ---------------------------------------------------------------------------

func TestWebhookInstagram_NativeDMText(t *testing.T) {
	handler, channelRepo, producer, _ := setupWebhookTest()
	addChannel(channelRepo, "ch-ig", entity.ChannelTypeInstagram, map[string]string{
		"app_secret":   "ig-app-secret",
		"verify_token": "ig-verify",
	})

	payload := map[string]interface{}{
		"object": "instagram",
		"entry": []map[string]interface{}{
			{
				"id":   "ig-account-1",
				"time": 1,
				"messaging": []map[string]interface{}{
					{
						"sender":    map[string]string{"id": "ig-user-1"},
						"recipient": map[string]string{"id": "ig-account-1"},
						"timestamp": int64(1700000000000),
						"message":   map[string]interface{}{"mid": "ig_mid_1", "text": "Hi via Instagram"},
					},
				},
			},
		},
	}

	w, c := postSignedTo("ch-ig", payload, "ig-app-secret", "")
	handler.InstagramWebhook(c)

	assert.Equal(t, http.StatusOK, w.Code)
	require.Len(t, producer.InboundMessages, 1)
	msg := producer.InboundMessages[0]
	assert.Equal(t, "instagram", msg.ChannelType)
	assert.Equal(t, "text", msg.ContentType)
	assert.Equal(t, "Hi via Instagram", msg.Content)
	assert.Equal(t, "ig_mid_1", msg.ExternalID)
	assert.Equal(t, "ig-user-1", msg.Metadata["sender_id"])
	assert.Equal(t, "ig-account-1", msg.Metadata["instagram_id"])
}

func TestWebhookInstagram_ViaPageDMText(t *testing.T) {
	handler, channelRepo, producer, _ := setupWebhookTest()
	addChannel(channelRepo, "ch-ig", entity.ChannelTypeInstagram, map[string]string{
		"app_secret":   "ig-app-secret",
		"verify_token": "ig-verify",
	})

	// Instagram DMs delivered via a Facebook Page arrive under object "page"
	// (indistinguishable at the object level from Messenger). The handler must
	// still classify this as Instagram traffic (IsInstagramViaPageWebhook) and
	// normalize it as an instagram inbound message.
	payload := map[string]interface{}{
		"object": "page",
		"entry": []map[string]interface{}{
			{
				"id": "ig-account-1",
				"messaging": []map[string]interface{}{
					{
						"sender":    map[string]string{"id": "ig-user-2"},
						"recipient": map[string]string{"id": "ig-account-1"},
						"timestamp": int64(1700000000000),
						"message":   map[string]interface{}{"mid": "ig_mid_2", "text": "DM via Page"},
					},
				},
			},
		},
	}

	w, c := postSignedTo("ch-ig", payload, "ig-app-secret", "")
	handler.InstagramWebhook(c)

	assert.Equal(t, http.StatusOK, w.Code)
	require.Len(t, producer.InboundMessages, 1)
	msg := producer.InboundMessages[0]
	assert.Equal(t, "instagram", msg.ChannelType)
	assert.Equal(t, "DM via Page", msg.Content)
	assert.Equal(t, "ig_mid_2", msg.ExternalID)
}

// ---------------------------------------------------------------------------
// 4. WhatsApp `failed` status regression (WS5). Meta sends error_data as an
// OBJECT. A batch that carries a `failed` status alongside a sibling inbound
// text message must still parse: the sibling message must be published and the
// handler must return 200. Locks the *ErrorData object typing in
// whatsapp_official/types.go.
// ---------------------------------------------------------------------------

func TestWebhookWhatsApp_FailedStatusWithObjectErrorDataKeepsSiblingMessage(t *testing.T) {
	handler, _, producer, _ := setupWebhookTest()

	value := map[string]interface{}{
		"messaging_product": "whatsapp",
		"metadata": map[string]string{
			"display_phone_number": "551199999999",
			"phone_number_id":      "phone-id-1",
		},
		"messages": []map[string]interface{}{
			{
				"id":   "wamid.sibling",
				"from": "5511999990000",
				"type": "text",
				"text": map[string]string{"body": "sibling inbound text"},
			},
		},
		"statuses": []map[string]interface{}{
			{
				"id":           "wamid.failed",
				"recipient_id": "5511999990000",
				"status":       "failed",
				"timestamp":    "1700000000",
				"errors": []map[string]interface{}{
					{
						"code":    131026,
						"title":   "Message undeliverable",
						"message": "Message failed to send",
						// error_data as an OBJECT (the real Meta shape) — the
						// regression this test guards.
						"error_data": map[string]interface{}{
							"messaging_product": "whatsapp",
							"details":           "Message failed to send because more than 24 hours have passed since the customer last replied.",
						},
					},
				},
			},
		},
	}

	payload := buildWhatsAppFieldPayload("messages", value)

	// Prove directly that the whatsapp_official payload type parses the object
	// error_data (this is the exact type the handler unmarshals alongside its own
	// payload). If ErrorData regressed to a string this Unmarshal would fail.
	rawBody, _ := json.Marshal(payload)
	var official whatsappofficial.WebhookPayload
	require.NoError(t, json.Unmarshal(rawBody, &official))

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	postWhatsAppJSON(c, payload, "test-secret")

	handler.WhatsAppWebhook(c)

	assert.Equal(t, http.StatusOK, w.Code)

	// The sibling inbound text must survive the presence of the object-shaped
	// failed-status error and be published.
	var siblingFound bool
	for _, m := range producer.InboundMessages {
		if m.ExternalID == "wamid.sibling" {
			siblingFound = true
			assert.Equal(t, "sibling inbound text", m.Content)
			assert.Equal(t, string(entity.ChannelTypeWhatsAppOfficial), m.ChannelType)
		}
	}
	assert.True(t, siblingFound, "sibling inbound text must be published despite the object-shaped failed status")

	// The failed status is still surfaced as a status update.
	var failedFound bool
	for _, s := range producer.StatusUpdates {
		if s.ExternalID == "wamid.failed" && s.Status == "failed" {
			failedFound = true
		}
	}
	assert.True(t, failedFound, "failed status must still be published")
}
