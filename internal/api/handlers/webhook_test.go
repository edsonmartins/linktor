package handlers

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/msgfy/linktor/internal/application/service"
	"github.com/msgfy/linktor/internal/domain/entity"
	"github.com/msgfy/linktor/pkg/testutil"
)

func init() {
	gin.SetMode(gin.TestMode)
}

// setupWebhookTest creates a WebhookHandler with mock dependencies and a default WhatsApp channel.
func setupWebhookTest() (*WebhookHandler, *testutil.MockChannelRepository, *testutil.MockProducer, *mockTemplateRepository) {
	channelRepo := testutil.NewMockChannelRepository()
	producer := testutil.NewMockProducer()
	templateRepo := newMockTemplateRepository()
	templateSvc := service.NewTemplateService(templateRepo, channelRepo)
	handler := NewWebhookHandler(channelRepo, producer, templateSvc)

	channel := &entity.Channel{
		ID:               "ch-1",
		TenantID:         "tenant-1",
		Type:             entity.ChannelTypeWhatsAppOfficial,
		Name:             "Test WhatsApp",
		Enabled:          true,
		ConnectionStatus: entity.ConnectionStatusConnected,
		Credentials: map[string]string{
			"verify_token":   "test-verify-token",
			"webhook_secret": "test-secret",
		},
	}
	channelRepo.Channels["ch-1"] = channel

	return handler, channelRepo, producer, templateRepo
}

// buildWhatsAppPayload constructs a WhatsApp webhook payload with the given messages, contacts, and statuses.
func buildWhatsAppPayload(messages []WhatsAppMessage, contacts []WhatsAppContact, statuses []WhatsAppStatus) map[string]interface{} {
	value := map[string]interface{}{
		"messaging_product": "whatsapp",
		"metadata": map[string]string{
			"display_phone_number": "551199999999",
			"phone_number_id":      "phone-id-1",
		},
	}
	if messages != nil {
		value["messages"] = messages
	}
	if contacts != nil {
		value["contacts"] = contacts
	}
	if statuses != nil {
		value["statuses"] = statuses
	}

	return map[string]interface{}{
		"object": "whatsapp_business_account",
		"entry": []map[string]interface{}{
			{
				"id": "entry-1",
				"changes": []map[string]interface{}{
					{
						"field": "messages",
						"value": value,
					},
				},
			},
		},
	}
}

func buildWhatsAppFieldPayload(field string, value map[string]interface{}) map[string]interface{} {
	return map[string]interface{}{
		"object": "whatsapp_business_account",
		"entry": []map[string]interface{}{
			{
				"id": "entry-1",
				"changes": []map[string]interface{}{
					{
						"field": field,
						"value": value,
					},
				},
			},
		},
	}
}

// computeHMACSHA256 computes the HMAC-SHA256 signature for a body using the given secret.
func computeHMACSHA256(secret string, body []byte) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(body)
	return "sha256=" + hex.EncodeToString(mac.Sum(nil))
}

// postWhatsAppJSON sets up a POST request with JSON body, HMAC signature, and gin params on the context.
func postWhatsAppJSON(c *gin.Context, payload interface{}, secret string) []byte {
	body, _ := json.Marshal(payload)
	req := httptest.NewRequest(http.MethodPost, "/webhook/ch-1", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	if secret != "" {
		req.Header.Set("X-Hub-Signature-256", computeHMACSHA256(secret, body))
	}
	c.Request = req
	c.Params = []gin.Param{{Key: "channelId", Value: "ch-1"}}
	return body
}

// ---------------------------------------------------------------------------
// 1. WhatsApp GET Verification
// ---------------------------------------------------------------------------

func TestWebhookWhatsAppVerification_CorrectToken(t *testing.T) {
	handler, _, _, _ := setupWebhookTest()

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	req := httptest.NewRequest(http.MethodGet, "/webhook/ch-1", nil)
	req.URL.RawQuery = url.Values{
		"hub.mode":         {"subscribe"},
		"hub.verify_token": {"test-verify-token"},
		"hub.challenge":    {"test-challenge"},
	}.Encode()
	c.Request = req
	c.Params = []gin.Param{{Key: "channelId", Value: "ch-1"}}

	handler.WhatsAppWebhook(c)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "test-challenge", w.Body.String())
}

func TestWebhookWhatsAppVerification_WrongToken(t *testing.T) {
	handler, _, _, _ := setupWebhookTest()

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	req := httptest.NewRequest(http.MethodGet, "/webhook/ch-1", nil)
	req.URL.RawQuery = url.Values{
		"hub.mode":         {"subscribe"},
		"hub.verify_token": {"wrong-token"},
		"hub.challenge":    {"test-challenge"},
	}.Encode()
	c.Request = req
	c.Params = []gin.Param{{Key: "channelId", Value: "ch-1"}}

	handler.WhatsAppWebhook(c)

	assert.Equal(t, http.StatusForbidden, w.Code)
}

func TestWebhookWhatsAppVerification_WrongMode(t *testing.T) {
	handler, _, _, _ := setupWebhookTest()

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	req := httptest.NewRequest(http.MethodGet, "/webhook/ch-1", nil)
	req.URL.RawQuery = url.Values{
		"hub.mode":         {"unsubscribe"},
		"hub.verify_token": {"test-verify-token"},
		"hub.challenge":    {"test-challenge"},
	}.Encode()
	c.Request = req
	c.Params = []gin.Param{{Key: "channelId", Value: "ch-1"}}

	handler.WhatsAppWebhook(c)

	assert.Equal(t, http.StatusForbidden, w.Code)
}

func TestWebhookWhatsAppVerification_ChannelNotFound(t *testing.T) {
	handler, _, _, _ := setupWebhookTest()

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	req := httptest.NewRequest(http.MethodGet, "/webhook/ch-unknown", nil)
	req.URL.RawQuery = url.Values{
		"hub.mode":         {"subscribe"},
		"hub.verify_token": {"test-verify-token"},
		"hub.challenge":    {"test-challenge"},
	}.Encode()
	c.Request = req
	c.Params = []gin.Param{{Key: "channelId", Value: "ch-unknown"}}

	handler.WhatsAppWebhook(c)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

// ---------------------------------------------------------------------------
// 2. WhatsApp POST - Text Message
// ---------------------------------------------------------------------------

func TestWebhookWhatsAppPost_TextMessage(t *testing.T) {
	handler, _, producer, _ := setupWebhookTest()

	payload := buildWhatsAppPayload(
		[]WhatsAppMessage{
			{
				ID:   "wamid.123",
				From: "5511999990000",
				Type: "text",
				Text: struct {
					Body string `json:"body"`
				}{Body: "Hello from WhatsApp"},
			},
		},
		[]WhatsAppContact{
			{
				WaID: "5511999990000",
				Profile: struct {
					Name string `json:"name"`
				}{Name: "John Doe"},
			},
		},
		nil,
	)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	postWhatsAppJSON(c, payload, "test-secret")

	handler.WhatsAppWebhook(c)

	assert.Equal(t, http.StatusOK, w.Code)

	require.Len(t, producer.InboundMessages, 1)
	msg := producer.InboundMessages[0]
	assert.Equal(t, "tenant-1", msg.TenantID)
	assert.Equal(t, "ch-1", msg.ChannelID)
	assert.Equal(t, string(entity.ChannelTypeWhatsAppOfficial), msg.ChannelType)
	assert.Equal(t, "wamid.123", msg.ExternalID)
	assert.Equal(t, "text", msg.ContentType)
	assert.Equal(t, "Hello from WhatsApp", msg.Content)
	assert.Equal(t, "5511999990000", msg.Metadata["phone"])
	assert.Equal(t, "John Doe", msg.Metadata["sender_name"])
	assert.Equal(t, "5511999990000", msg.Metadata["sender_id"])
}

// ---------------------------------------------------------------------------
// 3. WhatsApp POST - Image Message
// ---------------------------------------------------------------------------

func TestWebhookWhatsAppPost_ImageMessage(t *testing.T) {
	handler, _, producer, _ := setupWebhookTest()

	payload := buildWhatsAppPayload(
		[]WhatsAppMessage{
			{
				ID:   "wamid.456",
				From: "5511999990000",
				Type: "image",
				Image: struct {
					ID       string `json:"id"`
					Caption  string `json:"caption"`
					MimeType string `json:"mime_type"`
				}{
					ID:       "media-id-abc",
					Caption:  "A photo",
					MimeType: "image/jpeg",
				},
			},
		},
		[]WhatsAppContact{
			{
				WaID: "5511999990000",
				Profile: struct {
					Name string `json:"name"`
				}{Name: "Jane"},
			},
		},
		nil,
	)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	postWhatsAppJSON(c, payload, "test-secret")

	handler.WhatsAppWebhook(c)

	assert.Equal(t, http.StatusOK, w.Code)
	require.Len(t, producer.InboundMessages, 1)
	msg := producer.InboundMessages[0]
	assert.Equal(t, "image", msg.ContentType)
	assert.Equal(t, "A photo", msg.Content)
	require.Len(t, msg.Attachments, 1)
	assert.Equal(t, "image", msg.Attachments[0].Type)
	assert.Equal(t, "media-id-abc", msg.Attachments[0].URL)
}

// ---------------------------------------------------------------------------
// 4. WhatsApp POST - Invalid JSON
// ---------------------------------------------------------------------------

func TestWebhookWhatsAppPost_InvalidJSON(t *testing.T) {
	handler, _, _, _ := setupWebhookTest()

	// Remove webhook_secret so signature check is bypassed and we reach JSON parsing
	handler.channelRepo.(*testutil.MockChannelRepository).Channels["ch-1"].Credentials = map[string]string{
		"verify_token": "test-verify-token",
	}

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	body := []byte("not json at all")
	req := httptest.NewRequest(http.MethodPost, "/webhook/ch-1", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	c.Request = req
	c.Params = []gin.Param{{Key: "channelId", Value: "ch-1"}}

	handler.WhatsAppWebhook(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// ---------------------------------------------------------------------------
// 5. WhatsApp POST - Channel Not Found
// ---------------------------------------------------------------------------

func TestWebhookWhatsAppPost_ChannelNotFound(t *testing.T) {
	handler, _, _, _ := setupWebhookTest()

	payload := buildWhatsAppPayload(nil, nil, nil)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	body, _ := json.Marshal(payload)
	req := httptest.NewRequest(http.MethodPost, "/webhook/ch-unknown", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	c.Request = req
	c.Params = []gin.Param{{Key: "channelId", Value: "ch-unknown"}}

	handler.WhatsAppWebhook(c)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

// ---------------------------------------------------------------------------
// 6. WhatsApp POST - Signature Validation
// ---------------------------------------------------------------------------

func TestWebhookWhatsAppPost_ValidSignature(t *testing.T) {
	handler, _, producer, _ := setupWebhookTest()

	payload := buildWhatsAppPayload(
		[]WhatsAppMessage{
			{
				ID:   "wamid.sig1",
				From: "5511999990000",
				Type: "text",
				Text: struct {
					Body string `json:"body"`
				}{Body: "signed message"},
			},
		},
		nil,
		nil,
	)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	postWhatsAppJSON(c, payload, "test-secret")

	handler.WhatsAppWebhook(c)

	assert.Equal(t, http.StatusOK, w.Code)
	require.Len(t, producer.InboundMessages, 1)
}

func TestWebhookWhatsAppPost_InvalidSignature(t *testing.T) {
	handler, _, _, _ := setupWebhookTest()

	payload := buildWhatsAppPayload(nil, nil, nil)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	body, _ := json.Marshal(payload)
	req := httptest.NewRequest(http.MethodPost, "/webhook/ch-1", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Hub-Signature-256", "sha256=0000000000000000000000000000000000000000000000000000000000000000")
	c.Request = req
	c.Params = []gin.Param{{Key: "channelId", Value: "ch-1"}}

	handler.WhatsAppWebhook(c)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestWebhookWhatsAppPost_TemplateStatusWebhook(t *testing.T) {
	handler, _, _, templateRepo := setupWebhookTest()
	templateRepo.Templates["tpl-1"] = &entity.Template{
		ID:         "tpl-1",
		ExternalID: "12345",
		Status:     entity.TemplateStatusPending,
	}

	payload := buildWhatsAppFieldPayload("message_template_status_update", map[string]interface{}{
		"message_template_id":       12345,
		"message_template_name":     "welcome_template",
		"message_template_language": "pt_BR",
		"event":                     "APPROVED",
	})

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	postWhatsAppJSON(c, payload, "test-secret")

	handler.WhatsAppWebhook(c)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, entity.TemplateStatusApproved, templateRepo.Templates["tpl-1"].Status)
}

func TestWebhookWhatsAppPost_TemplateQualityWebhook(t *testing.T) {
	handler, _, _, templateRepo := setupWebhookTest()
	templateRepo.Templates["tpl-1"] = &entity.Template{
		ID:           "tpl-1",
		ExternalID:   "12345",
		QualityScore: entity.TemplateQualityUnknown,
	}

	payload := buildWhatsAppFieldPayload("message_template_quality_update", map[string]interface{}{
		"message_template_id":       12345,
		"message_template_name":     "welcome_template",
		"message_template_language": "pt_BR",
		"previous_quality_score":    "UNKNOWN",
		"new_quality_score":         "GREEN",
	})

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	postWhatsAppJSON(c, payload, "test-secret")

	handler.WhatsAppWebhook(c)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, entity.TemplateQualityGreen, templateRepo.Templates["tpl-1"].QualityScore)
}

func TestWebhookWhatsAppPost_TemplateCategoryWebhook(t *testing.T) {
	handler, _, _, templateRepo := setupWebhookTest()
	templateRepo.Templates["tpl-1"] = &entity.Template{
		ID:         "tpl-1",
		ExternalID: "12345",
		Category:   entity.TemplateCategoryUtility,
	}

	payload := buildWhatsAppFieldPayload("template_category_update", map[string]interface{}{
		"message_template_id":       12345,
		"message_template_name":     "welcome_template",
		"message_template_language": "pt_BR",
		"previous_category":         "UTILITY",
		"new_category":              "MARKETING",
	})

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	postWhatsAppJSON(c, payload, "test-secret")

	handler.WhatsAppWebhook(c)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, entity.TemplateCategoryMarketing, templateRepo.Templates["tpl-1"].Category)
}

func TestWebhookWhatsAppPost_PhoneNumberQualityWebhook(t *testing.T) {
	handler, channelRepo, _, _ := setupWebhookTest()

	payload := buildWhatsAppFieldPayload("phone_number_quality_update", map[string]interface{}{
		"display_phone_number": "+5511999999999",
		"event":                "FLAGGED",
		"current_limit":        "TIER_1K",
	})

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	postWhatsAppJSON(c, payload, "test-secret")

	handler.WhatsAppWebhook(c)

	assert.Equal(t, http.StatusOK, w.Code)

	updatedChannel := channelRepo.Channels["ch-1"]
	require.NotNil(t, updatedChannel)
	assert.Equal(t, "RED", updatedChannel.Config["quality_rating"])
	assert.Equal(t, "FLAGGED", updatedChannel.Config["quality_rating_event"])
	assert.Equal(t, "TIER_1K", updatedChannel.Config["messaging_limit_tier"])
	assert.Equal(t, "+5511999999999", updatedChannel.Config["phone_number"])
}

func TestWebhookWhatsAppPost_NoSecretConfigured(t *testing.T) {
	handler, channelRepo, producer, _ := setupWebhookTest()

	// Remove webhook_secret from credentials
	delete(channelRepo.Channels["ch-1"].Credentials, "webhook_secret")

	payload := buildWhatsAppPayload(
		[]WhatsAppMessage{
			{
				ID:   "wamid.nosec",
				From: "5511999990000",
				Type: "text",
				Text: struct {
					Body string `json:"body"`
				}{Body: "no secret"},
			},
		},
		nil,
		nil,
	)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	body, _ := json.Marshal(payload)
	req := httptest.NewRequest(http.MethodPost, "/webhook/ch-1", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	c.Request = req
	c.Params = []gin.Param{{Key: "channelId", Value: "ch-1"}}

	handler.WhatsAppWebhook(c)

	assert.Equal(t, http.StatusOK, w.Code)
	require.Len(t, producer.InboundMessages, 1)
}

// ---------------------------------------------------------------------------
// 7. WhatsApp POST - Status Updates
// ---------------------------------------------------------------------------

func TestWebhookWhatsAppPost_StatusUpdates(t *testing.T) {
	handler, _, producer, _ := setupWebhookTest()

	payload := buildWhatsAppPayload(
		nil,
		nil,
		[]WhatsAppStatus{
			{ID: "wamid.s1", RecipientID: "5511999990000", Status: "sent", Timestamp: "1700000001"},
			{ID: "wamid.s2", RecipientID: "5511999990000", Status: "delivered", Timestamp: "1700000002"},
			{ID: "wamid.s3", RecipientID: "5511999990000", Status: "read", Timestamp: "1700000003"},
			{ID: "wamid.s4", RecipientID: "5511999990000", Status: "failed", Timestamp: "1700000004"},
		},
	)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	postWhatsAppJSON(c, payload, "test-secret")

	handler.WhatsAppWebhook(c)

	assert.Equal(t, http.StatusOK, w.Code)
	require.Len(t, producer.StatusUpdates, 4)

	assert.Equal(t, "sent", producer.StatusUpdates[0].Status)
	assert.Equal(t, "wamid.s1", producer.StatusUpdates[0].ExternalID)

	assert.Equal(t, "delivered", producer.StatusUpdates[1].Status)
	assert.Equal(t, "read", producer.StatusUpdates[2].Status)
	assert.Equal(t, "failed", producer.StatusUpdates[3].Status)
}

// ---------------------------------------------------------------------------
// 8. WhatsApp POST - Interactive Message
// ---------------------------------------------------------------------------

func TestWebhookWhatsAppPost_InteractiveMessage(t *testing.T) {
	handler, _, producer, _ := setupWebhookTest()

	messages := []map[string]interface{}{
		{
			"id":   "wamid.inter1",
			"from": "5511999990000",
			"type": "interactive",
			"interactive": map[string]interface{}{
				"type": "button_reply",
				"button_reply": map[string]interface{}{
					"id":    "btn-yes",
					"title": "Yes",
				},
			},
		},
	}

	payload := map[string]interface{}{
		"object": "whatsapp_business_account",
		"entry": []map[string]interface{}{
			{
				"id": "entry-1",
				"changes": []map[string]interface{}{
					{
						"field": "messages",
						"value": map[string]interface{}{
							"messaging_product": "whatsapp",
							"metadata":          map[string]string{},
							"messages":          messages,
						},
					},
				},
			},
		},
	}

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	postWhatsAppJSON(c, payload, "test-secret")

	handler.WhatsAppWebhook(c)

	assert.Equal(t, http.StatusOK, w.Code)
	require.Len(t, producer.InboundMessages, 1)
	msg := producer.InboundMessages[0]
	assert.Equal(t, "interactive", msg.ContentType)
	assert.Equal(t, "button_reply", msg.Metadata["interactive_type"])
	assert.Equal(t, "btn-yes", msg.Metadata["button_id"])
}

// ---------------------------------------------------------------------------
// 9. WhatsApp POST - Location Message
// ---------------------------------------------------------------------------

func TestWebhookWhatsAppPost_LocationMessage(t *testing.T) {
	handler, _, producer, _ := setupWebhookTest()

	messages := []map[string]interface{}{
		{
			"id":   "wamid.loc1",
			"from": "5511999990000",
			"type": "location",
			"location": map[string]interface{}{
				"latitude":  -23.5505,
				"longitude": -46.6333,
				"name":      "Sao Paulo",
				"address":   "SP, Brazil",
			},
		},
	}

	payload := map[string]interface{}{
		"object": "whatsapp_business_account",
		"entry": []map[string]interface{}{
			{
				"id": "entry-1",
				"changes": []map[string]interface{}{
					{
						"field": "messages",
						"value": map[string]interface{}{
							"messaging_product": "whatsapp",
							"metadata":          map[string]string{},
							"messages":          messages,
						},
					},
				},
			},
		},
	}

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	postWhatsAppJSON(c, payload, "test-secret")

	handler.WhatsAppWebhook(c)

	assert.Equal(t, http.StatusOK, w.Code)
	require.Len(t, producer.InboundMessages, 1)
	msg := producer.InboundMessages[0]
	assert.Equal(t, "location", msg.ContentType)
	assert.Contains(t, msg.Metadata["latitude"], "-23.55")
	assert.Contains(t, msg.Metadata["longitude"], "-46.63")
}

// ---------------------------------------------------------------------------
// 10. WhatsApp POST - Reply-to Context
// ---------------------------------------------------------------------------

func TestWebhookWhatsAppPost_ReplyToContext(t *testing.T) {
	handler, _, producer, _ := setupWebhookTest()

	messages := []map[string]interface{}{
		{
			"id":   "wamid.reply1",
			"from": "5511999990000",
			"type": "text",
			"text": map[string]string{"body": "This is a reply"},
			"context": map[string]string{
				"id":   "wamid.original",
				"from": "5511888880000",
			},
		},
	}

	payload := map[string]interface{}{
		"object": "whatsapp_business_account",
		"entry": []map[string]interface{}{
			{
				"id": "entry-1",
				"changes": []map[string]interface{}{
					{
						"field": "messages",
						"value": map[string]interface{}{
							"messaging_product": "whatsapp",
							"metadata":          map[string]string{},
							"messages":          messages,
						},
					},
				},
			},
		},
	}

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	postWhatsAppJSON(c, payload, "test-secret")

	handler.WhatsAppWebhook(c)

	assert.Equal(t, http.StatusOK, w.Code)
	require.Len(t, producer.InboundMessages, 1)
	msg := producer.InboundMessages[0]
	assert.Equal(t, "wamid.original", msg.Metadata["reply_to_id"])
	assert.Equal(t, "5511888880000", msg.Metadata["reply_to_from"])
}

// ---------------------------------------------------------------------------
// 11. Generic Webhook
// ---------------------------------------------------------------------------

func TestWebhookGeneric_ValidPayload(t *testing.T) {
	handler, channelRepo, producer, _ := setupWebhookTest()

	// Add a generic channel
	channelRepo.Channels["ch-gen"] = &entity.Channel{
		ID:               "ch-gen",
		TenantID:         "tenant-1",
		Type:             entity.ChannelTypeWebChat,
		Name:             "Generic Channel",
		Enabled:          true,
		ConnectionStatus: entity.ConnectionStatusConnected,
		Credentials:      map[string]string{},
	}

	payload := GenericWebhookPayload{
		MessageID:   "ext-msg-1",
		SenderID:    "user-42",
		SenderName:  "Alice",
		ContentType: "text",
		Content:     "Hello via generic webhook",
		Metadata:    map[string]string{"source": "api"},
	}

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	body, _ := json.Marshal(payload)
	req := httptest.NewRequest(http.MethodPost, "/webhook/generic/ch-gen", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	c.Request = req
	c.Params = []gin.Param{{Key: "channelId", Value: "ch-gen"}}

	handler.GenericWebhook(c)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.NotEmpty(t, resp["message_id"])

	require.Len(t, producer.InboundMessages, 1)
	msg := producer.InboundMessages[0]
	assert.Equal(t, "tenant-1", msg.TenantID)
	assert.Equal(t, "ch-gen", msg.ChannelID)
	assert.Equal(t, string(entity.ChannelTypeWebChat), msg.ChannelType)
	assert.Equal(t, "ext-msg-1", msg.ExternalID)
	assert.Equal(t, "text", msg.ContentType)
	assert.Equal(t, "Hello via generic webhook", msg.Content)
	assert.Equal(t, "user-42", msg.Metadata["sender_id"])
	assert.Equal(t, "Alice", msg.Metadata["sender_name"])
	assert.Equal(t, "api", msg.Metadata["source"])
}

// ---------------------------------------------------------------------------
// 12. Generic Webhook - Channel Not Found
// ---------------------------------------------------------------------------

func TestWebhookGeneric_ChannelNotFound(t *testing.T) {
	handler, _, _, _ := setupWebhookTest()

	payload := GenericWebhookPayload{
		MessageID:   "ext-1",
		ContentType: "text",
		Content:     "test",
		Metadata:    map[string]string{},
	}

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	body, _ := json.Marshal(payload)
	req := httptest.NewRequest(http.MethodPost, "/webhook/generic/ch-missing", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	c.Request = req
	c.Params = []gin.Param{{Key: "channelId", Value: "ch-missing"}}

	handler.GenericWebhook(c)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

// ---------------------------------------------------------------------------
// 13. Generic Webhook - Invalid JSON
// ---------------------------------------------------------------------------

func TestWebhookGeneric_InvalidJSON(t *testing.T) {
	handler, channelRepo, _, _ := setupWebhookTest()

	channelRepo.Channels["ch-gen"] = &entity.Channel{
		ID:               "ch-gen",
		TenantID:         "tenant-1",
		Type:             entity.ChannelTypeWebChat,
		Enabled:          true,
		ConnectionStatus: entity.ConnectionStatusConnected,
		Credentials:      map[string]string{},
	}

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	req := httptest.NewRequest(http.MethodPost, "/webhook/generic/ch-gen", bytes.NewReader([]byte("{bad")))
	req.Header.Set("Content-Type", "application/json")
	c.Request = req
	c.Params = []gin.Param{{Key: "channelId", Value: "ch-gen"}}

	handler.GenericWebhook(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// ---------------------------------------------------------------------------
// 14. Status Callback
// ---------------------------------------------------------------------------

func TestWebhookStatusCallback_ValidPayload(t *testing.T) {
	handler, _, producer, _ := setupWebhookTest()

	payload := StatusCallbackPayload{
		MessageID:    "msg-100",
		ExternalID:   "ext-100",
		Status:       "delivered",
		ErrorMessage: "",
	}

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	body, _ := json.Marshal(payload)
	req := httptest.NewRequest(http.MethodPost, "/webhook/status/ch-1", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	c.Request = req
	c.Params = []gin.Param{{Key: "channelId", Value: "ch-1"}}

	handler.StatusCallback(c)

	assert.Equal(t, http.StatusOK, w.Code)
	require.Len(t, producer.StatusUpdates, 1)
	su := producer.StatusUpdates[0]
	assert.Equal(t, "msg-100", su.MessageID)
	assert.Equal(t, "ext-100", su.ExternalID)
	assert.Equal(t, "delivered", su.Status)
	assert.Equal(t, string(entity.ChannelTypeWhatsAppOfficial), su.ChannelType)
}

// ---------------------------------------------------------------------------
// 15. Status Callback - Channel Not Found
// ---------------------------------------------------------------------------

func TestWebhookStatusCallback_ChannelNotFound(t *testing.T) {
	handler, _, _, _ := setupWebhookTest()

	payload := StatusCallbackPayload{
		MessageID: "msg-100",
		Status:    "delivered",
	}

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	body, _ := json.Marshal(payload)
	req := httptest.NewRequest(http.MethodPost, "/webhook/status/ch-missing", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	c.Request = req
	c.Params = []gin.Param{{Key: "channelId", Value: "ch-missing"}}

	handler.StatusCallback(c)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

// ---------------------------------------------------------------------------
// 16. Telegram Webhook
// ---------------------------------------------------------------------------

func TestWebhookTelegram_TextMessage(t *testing.T) {
	handler, channelRepo, producer, _ := setupWebhookTest()

	channelRepo.Channels["ch-tg"] = &entity.Channel{
		ID:               "ch-tg",
		TenantID:         "tenant-1",
		Type:             entity.ChannelTypeTelegram,
		Name:             "Telegram Bot",
		Enabled:          true,
		ConnectionStatus: entity.ConnectionStatusConnected,
		Credentials:      map[string]string{},
	}

	payload := map[string]interface{}{
		"update_id": 12345,
		"message": map[string]interface{}{
			"message_id": 678,
			"from": map[string]interface{}{
				"id":         int64(99001),
				"first_name": "Bob",
				"last_name":  "Smith",
				"username":   "bobsmith",
			},
			"chat": map[string]interface{}{
				"id":   int64(99001),
				"type": "private",
			},
			"text": "Hello from Telegram",
		},
	}

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	body, _ := json.Marshal(payload)
	req := httptest.NewRequest(http.MethodPost, "/webhook/telegram/ch-tg", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	c.Request = req
	c.Params = []gin.Param{{Key: "channelId", Value: "ch-tg"}}

	handler.TelegramWebhook(c)

	assert.Equal(t, http.StatusOK, w.Code)
	require.Len(t, producer.InboundMessages, 1)
	msg := producer.InboundMessages[0]
	assert.Equal(t, "telegram", msg.ChannelType)
	assert.Equal(t, "tenant-1", msg.TenantID)
	assert.Equal(t, "ch-tg", msg.ChannelID)
	assert.Equal(t, "text", msg.ContentType)
	assert.Equal(t, "Hello from Telegram", msg.Content)
}

// ---------------------------------------------------------------------------
// 17. Telegram Webhook - Channel Not Found
// ---------------------------------------------------------------------------

func TestWebhookTelegram_ChannelNotFound(t *testing.T) {
	handler, _, _, _ := setupWebhookTest()

	payload := map[string]interface{}{
		"update_id": 12345,
		"message": map[string]interface{}{
			"message_id": 678,
			"chat": map[string]interface{}{
				"id":   int64(99001),
				"type": "private",
			},
			"text": "Hello",
		},
	}

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	body, _ := json.Marshal(payload)
	req := httptest.NewRequest(http.MethodPost, "/webhook/telegram/ch-missing", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	c.Request = req
	c.Params = []gin.Param{{Key: "channelId", Value: "ch-missing"}}

	handler.TelegramWebhook(c)

	assert.Equal(t, http.StatusNotFound, w.Code)
}
