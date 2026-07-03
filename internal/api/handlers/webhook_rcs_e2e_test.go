package handlers

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/msgfy/linktor/internal/domain/entity"
)

// postRCS builds an unsigned JSON POST to the RCS webhook (the test handler runs
// with requireWebhookSecrets=false and the channel has no webhook_secret).
func postRCS(channelID, body string) (*httptest.ResponseRecorder, *gin.Context) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	req := httptest.NewRequest(http.MethodPost, "http://example.com/webhooks/rcs/"+channelID, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	c.Request = req
	c.Params = []gin.Param{{Key: "channelId", Value: channelID}}
	return w, c
}

// A Zenvia RCS channel created without an explicit "provider" must still work:
// the handler defaults it to Zenvia. Before the fix, an empty provider failed
// Config.Validate and every inbound webhook returned HTTP 500.
func TestWebhookRCS_DefaultProviderInboundText(t *testing.T) {
	handler, channelRepo, producer, _ := setupWebhookTest()
	ch := addChannel(channelRepo, "ch-rcs", entity.ChannelTypeRCS, map[string]string{"api_key": "rcs-key"})
	ch.Config = map[string]string{"agent_id": "agent-1"} // deliberately no "provider"

	body := `{
		"id": "wh-1",
		"timestamp": "2026-03-08T10:00:00Z",
		"type": "MESSAGE",
		"message": {
			"id": "rcs-ext-1",
			"from": "+5511999999999",
			"to": "agent-1",
			"contents": [{"type": "text", "text": "hi over RCS"}]
		}
	}`

	w, c := postRCS("ch-rcs", body)
	handler.RCSWebhook(c)

	assert.Equal(t, http.StatusOK, w.Code, "missing provider must default to Zenvia, not 500")
	require.Len(t, producer.InboundMessages, 1)
	msg := producer.InboundMessages[0]
	assert.Equal(t, "rcs", msg.ChannelType)
	assert.Equal(t, "text", msg.ContentType)
	assert.Equal(t, "hi over RCS", msg.Content)
	assert.Equal(t, "rcs-ext-1", msg.ExternalID)
	assert.Equal(t, "+5511999999999", msg.Metadata["sender_phone"])
}

// Inbound media (file content) is surfaced as an attachment through the handler.
func TestWebhookRCS_InboundMediaBecomesAttachment(t *testing.T) {
	handler, channelRepo, producer, _ := setupWebhookTest()
	ch := addChannel(channelRepo, "ch-rcs", entity.ChannelTypeRCS, map[string]string{"api_key": "rcs-key"})
	ch.Config = map[string]string{"provider": "zenvia", "agent_id": "agent-1"}

	body := `{
		"id": "wh-2",
		"type": "MESSAGE",
		"message": {
			"id": "rcs-ext-2",
			"from": "+5511999999999",
			"to": "agent-1",
			"contents": [{"type": "file", "file": {"fileUrl": "https://cdn.zenvia/p.png", "fileMimeType": "image/png"}}]
		}
	}`

	w, c := postRCS("ch-rcs", body)
	handler.RCSWebhook(c)

	assert.Equal(t, http.StatusOK, w.Code)
	require.Len(t, producer.InboundMessages, 1)
	msg := producer.InboundMessages[0]
	assert.Equal(t, "image", msg.ContentType)
	require.Len(t, msg.Attachments, 1)
	assert.Equal(t, "https://cdn.zenvia/p.png", msg.Attachments[0].URL)
	assert.Equal(t, "image/png", msg.Attachments[0].MimeType)
}
