package sms

import (
	"crypto/hmac"
	"crypto/sha1"
	"encoding/base64"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseWebhook(t *testing.T) {
	t.Run("incoming message", func(t *testing.T) {
		body := []byte("MessageSid=SM123&AccountSid=AC456&From=%2B15551234567&To=%2B15559876543&Body=Hello+World&NumMedia=0&FromCity=New+York&FromState=NY&FromCountry=US&ApiVersion=2010-04-01")

		payload, wtype, err := ParseWebhook(body)
		require.NoError(t, err)
		assert.Equal(t, WebhookTypeIncoming, wtype)
		assert.Equal(t, "SM123", payload.MessageSID)
		assert.Equal(t, "AC456", payload.AccountSID)
		assert.Equal(t, "+15551234567", payload.From)
		assert.Equal(t, "+15559876543", payload.To)
		assert.Equal(t, "Hello World", payload.Body)
		assert.Equal(t, "0", payload.NumMedia)
		assert.Equal(t, "New York", payload.FromCity)
		assert.Equal(t, "NY", payload.FromState)
		assert.Equal(t, "US", payload.FromCountry)
	})

	t.Run("status callback", func(t *testing.T) {
		body := []byte("MessageSid=SM123&AccountSid=AC456&MessageStatus=delivered&From=%2B15551234567&To=%2B15559876543")

		payload, wtype, err := ParseWebhook(body)
		require.NoError(t, err)
		assert.Equal(t, WebhookTypeStatus, wtype)
		assert.Equal(t, "SM123", payload.MessageSID)
		assert.Equal(t, "delivered", payload.MessageStatus)
	})

	t.Run("status callback with SmsStatus", func(t *testing.T) {
		body := []byte("MessageSid=SM123&SmsStatus=failed&ErrorCode=30008&ErrorMessage=Unknown+error")

		payload, wtype, err := ParseWebhook(body)
		require.NoError(t, err)
		assert.Equal(t, WebhookTypeStatus, wtype)
		assert.Equal(t, "failed", payload.SmsStatus)
		assert.Equal(t, "30008", payload.ErrorCode)
		assert.Equal(t, "Unknown error", payload.ErrorMessage)
	})

	t.Run("empty body", func(t *testing.T) {
		payload, _, err := ParseWebhook([]byte(""))
		require.NoError(t, err) // url.ParseQuery handles empty string
		assert.Empty(t, payload.MessageSID)
	})
}

func TestValidateSignature(t *testing.T) {
	authToken := "12345"
	webhookURL := "https://mycompany.com/myapp.php?foo=1&bar=2"
	params := map[string]string{
		"CallSid":    "CA1234567890ABCDE",
		"Caller":     "+14158675310",
		"Digits":     "1234",
		"From":       "+14158675310",
		"To":         "+18005551212",
	}

	// Compute expected signature
	validationString := webhookURL
	validationString += "CallSid" + "CA1234567890ABCDE"
	validationString += "Caller" + "+14158675310"
	validationString += "Digits" + "1234"
	validationString += "From" + "+14158675310"
	validationString += "To" + "+18005551212"

	mac := hmac.New(sha1.New, []byte(authToken))
	mac.Write([]byte(validationString))
	validSig := base64.StdEncoding.EncodeToString(mac.Sum(nil))

	t.Run("valid signature", func(t *testing.T) {
		assert.True(t, ValidateSignature(authToken, webhookURL, params, validSig))
	})

	t.Run("invalid signature", func(t *testing.T) {
		assert.False(t, ValidateSignature(authToken, webhookURL, params, "invalid-signature"))
	})

	t.Run("wrong auth token", func(t *testing.T) {
		assert.False(t, ValidateSignature("wrong-token", webhookURL, params, validSig))
	})

	t.Run("empty params", func(t *testing.T) {
		mac2 := hmac.New(sha1.New, []byte(authToken))
		mac2.Write([]byte(webhookURL))
		emptySig := base64.StdEncoding.EncodeToString(mac2.Sum(nil))
		assert.True(t, ValidateSignature(authToken, webhookURL, map[string]string{}, emptySig))
	})
}

func TestExtractMediaURLs(t *testing.T) {
	t.Run("no media", func(t *testing.T) {
		body := []byte("MessageSid=SM123&Body=hello&NumMedia=0")
		urls, types, err := ExtractMediaURLs(body)
		require.NoError(t, err)
		assert.Empty(t, urls)
		assert.Empty(t, types)
	})

	t.Run("with media", func(t *testing.T) {
		body := []byte("NumMedia=2&MediaUrl0=https%3A%2F%2Fapi.twilio.com%2Fmedia1.jpg&MediaContentType0=image%2Fjpeg&MediaUrl1=https%3A%2F%2Fapi.twilio.com%2Fmedia2.png&MediaContentType1=image%2Fpng")
		urls, types, err := ExtractMediaURLs(body)
		require.NoError(t, err)
		require.Len(t, urls, 2)
		assert.Equal(t, "https://api.twilio.com/media1.jpg", urls[0])
		assert.Equal(t, "https://api.twilio.com/media2.png", urls[1])
		require.Len(t, types, 2)
		assert.Equal(t, "image/jpeg", types[0])
		assert.Equal(t, "image/png", types[1])
	})
}

func TestTwiMLResponse(t *testing.T) {
	t.Run("simple message", func(t *testing.T) {
		resp := NewTwiMLResponse().Message("Hello!").String()
		assert.Contains(t, resp, "<?xml version")
		assert.Contains(t, resp, "<Response>")
		assert.Contains(t, resp, "<Message>Hello!</Message>")
		assert.Contains(t, resp, "</Response>")
	})

	t.Run("message with media", func(t *testing.T) {
		resp := NewTwiMLResponse().MessageWithMedia("Check this", "https://example.com/img.jpg").String()
		assert.Contains(t, resp, "<Body>Check this</Body>")
		assert.Contains(t, resp, "<Media>https://example.com/img.jpg</Media>")
	})

	t.Run("redirect", func(t *testing.T) {
		resp := NewTwiMLResponse().Redirect("https://example.com/next").String()
		assert.Contains(t, resp, "<Redirect>https://example.com/next</Redirect>")
	})

	t.Run("XML escaping", func(t *testing.T) {
		resp := NewTwiMLResponse().Message("Hello <World> & \"Friends\"").String()
		assert.Contains(t, resp, "&lt;World&gt;")
		assert.Contains(t, resp, "&amp;")
		assert.Contains(t, resp, "&quot;Friends&quot;")
	})

	t.Run("empty response", func(t *testing.T) {
		resp := EmptyTwiMLResponse()
		assert.Equal(t, `<?xml version="1.0" encoding="UTF-8"?><Response></Response>`, resp)
	})
}
