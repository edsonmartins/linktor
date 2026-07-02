package handlers

import (
	"crypto/hmac"
	"crypto/sha1"
	"encoding/base64"
	"net/http"
	"net/http/httptest"
	"net/url"
	"sort"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/msgfy/linktor/internal/domain/entity"
)

// twilioSignature computes the X-Twilio-Signature the handler expects: base64 of
// HMAC-SHA1(authToken, requestURL + each sorted key+value), matching
// sms.ValidateSignature and firstValues() in the production handler.
func twilioSignature(authToken, requestURL string, form url.Values) string {
	keys := make([]string, 0, len(form))
	for k := range form {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	s := requestURL
	for _, k := range keys {
		s += k + form.Get(k)
	}
	mac := hmac.New(sha1.New, []byte(authToken))
	mac.Write([]byte(s))
	return base64.StdEncoding.EncodeToString(mac.Sum(nil))
}

// postTwilioForm builds a form-encoded POST to /webhooks/sms/:id with a valid (or
// overridden) X-Twilio-Signature. The request host/path fix requestURL so the
// signature is reproducible.
func postTwilioForm(channelID, authToken string, form url.Values, overrideSig string) (*httptest.ResponseRecorder, *gin.Context) {
	body := form.Encode()
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	req := httptest.NewRequest(http.MethodPost, "http://example.com/webhooks/sms/"+channelID, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	sig := overrideSig
	if sig == "" {
		sig = twilioSignature(authToken, "http://example.com/webhooks/sms/"+channelID, form)
	}
	req.Header.Set("X-Twilio-Signature", sig)
	c.Request = req
	c.Params = []gin.Param{{Key: "channelId", Value: channelID}}
	return w, c
}

// Regression + acceptance: a real Twilio inbound SMS (SmsStatus=received) must be
// published as an inbound message through the production handler. Before the
// classification fix it was routed to the status branch and silently dropped.
func TestWebhookTwilio_InboundSMSBecomesInboundMessage(t *testing.T) {
	handler, channelRepo, producer, _ := setupWebhookTest()
	addChannel(channelRepo, "ch-sms", entity.ChannelTypeSMS, map[string]string{
		"auth_token": "twilio-token",
	})

	form := url.Values{
		"MessageSid": {"SM_inbound_1"},
		"AccountSid": {"AC123"},
		"From":       {"+15551112222"},
		"To":         {"+15553334444"},
		"Body":       {"hello from a phone"},
		"SmsStatus":  {"received"},
		"NumMedia":   {"0"},
	}

	w, c := postTwilioForm("ch-sms", "twilio-token", form, "")
	handler.TwilioWebhook(c)

	assert.Equal(t, http.StatusOK, w.Code)
	require.Len(t, producer.InboundMessages, 1, "inbound SMS must be published, not swallowed as a status callback")
	msg := producer.InboundMessages[0]
	assert.Equal(t, "sms", msg.ChannelType)
	assert.Equal(t, "ch-sms", msg.ChannelID)
	assert.Equal(t, "text", msg.ContentType)
	assert.Equal(t, "hello from a phone", msg.Content)
	assert.Equal(t, "SM_inbound_1", msg.ExternalID)
	assert.Equal(t, "+15551112222", msg.Metadata["sender_id"])
}

func TestWebhookTwilio_InvalidSignatureRejected(t *testing.T) {
	handler, channelRepo, producer, _ := setupWebhookTest()
	addChannel(channelRepo, "ch-sms", entity.ChannelTypeSMS, map[string]string{
		"auth_token": "twilio-token",
	})

	form := url.Values{
		"MessageSid": {"SM_inbound_2"},
		"From":       {"+15551112222"},
		"To":         {"+15553334444"},
		"Body":       {"spoofed"},
		"SmsStatus":  {"received"},
	}

	w, c := postTwilioForm("ch-sms", "twilio-token", form, "not-the-right-signature")
	handler.TwilioWebhook(c)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
	assert.Empty(t, producer.InboundMessages, "a request with a bad signature must not be processed")
}
