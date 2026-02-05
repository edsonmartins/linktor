package sms

import (
	"crypto/hmac"
	"crypto/sha1"
	"encoding/base64"
	"fmt"
	"net/url"
	"sort"
	"strings"
)

// WebhookType represents the type of Twilio webhook
type WebhookType string

const (
	WebhookTypeIncoming WebhookType = "incoming"
	WebhookTypeStatus   WebhookType = "status"
)

// ParseWebhook parses a Twilio webhook payload
func ParseWebhook(body []byte) (*WebhookPayload, WebhookType, error) {
	// Parse form-encoded body
	values, err := url.ParseQuery(string(body))
	if err != nil {
		return nil, "", fmt.Errorf("failed to parse webhook body: %w", err)
	}

	payload := &WebhookPayload{
		MessageSID:          values.Get("MessageSid"),
		SmsSID:              values.Get("SmsSid"),
		AccountSID:          values.Get("AccountSid"),
		MessagingServiceSID: values.Get("MessagingServiceSid"),
		From:                values.Get("From"),
		To:                  values.Get("To"),
		Body:                values.Get("Body"),
		NumMedia:            values.Get("NumMedia"),
		MessageStatus:       values.Get("MessageStatus"),
		SmsStatus:           values.Get("SmsStatus"),
		ErrorCode:           values.Get("ErrorCode"),
		ErrorMessage:        values.Get("ErrorMessage"),
		FromCity:            values.Get("FromCity"),
		FromState:           values.Get("FromState"),
		FromZip:             values.Get("FromZip"),
		FromCountry:         values.Get("FromCountry"),
		ToCity:              values.Get("ToCity"),
		ToState:             values.Get("ToState"),
		ToZip:               values.Get("ToZip"),
		ToCountry:           values.Get("ToCountry"),
		APIVersion:          values.Get("ApiVersion"),
	}

	// Determine webhook type
	webhookType := WebhookTypeIncoming
	if payload.MessageStatus != "" || payload.SmsStatus != "" {
		webhookType = WebhookTypeStatus
	}

	return payload, webhookType, nil
}

// ValidateSignature validates a Twilio webhook signature
// Reference: https://www.twilio.com/docs/usage/security#validating-requests
func ValidateSignature(authToken, webhookURL string, params map[string]string, signature string) bool {
	// Build the validation string: URL + sorted params
	validationString := webhookURL

	// Get sorted keys
	keys := make([]string, 0, len(params))
	for k := range params {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	// Append params in sorted order
	for _, k := range keys {
		validationString += k + params[k]
	}

	// Compute HMAC-SHA1
	mac := hmac.New(sha1.New, []byte(authToken))
	mac.Write([]byte(validationString))
	computedSignature := base64.StdEncoding.EncodeToString(mac.Sum(nil))

	// Compare signatures
	return hmac.Equal([]byte(computedSignature), []byte(signature))
}

// ExtractMediaURLs extracts media URLs from a webhook payload
func ExtractMediaURLs(body []byte) ([]string, []string, error) {
	values, err := url.ParseQuery(string(body))
	if err != nil {
		return nil, nil, err
	}

	var urls []string
	var contentTypes []string

	// Media URLs are in MediaUrl0, MediaUrl1, etc.
	for i := 0; i < 10; i++ {
		urlKey := fmt.Sprintf("MediaUrl%d", i)
		typeKey := fmt.Sprintf("MediaContentType%d", i)

		if mediaURL := values.Get(urlKey); mediaURL != "" {
			urls = append(urls, mediaURL)
			contentTypes = append(contentTypes, values.Get(typeKey))
		} else {
			break
		}
	}

	return urls, contentTypes, nil
}

// TwiMLResponse helps build TwiML responses
type TwiMLResponse struct {
	body strings.Builder
}

// NewTwiMLResponse creates a new TwiML response builder
func NewTwiMLResponse() *TwiMLResponse {
	return &TwiMLResponse{}
}

// Message adds a Message element to the response
func (t *TwiMLResponse) Message(text string) *TwiMLResponse {
	t.body.WriteString(fmt.Sprintf("<Message>%s</Message>", escapeXML(text)))
	return t
}

// MessageWithMedia adds a Message element with media
func (t *TwiMLResponse) MessageWithMedia(text string, mediaURLs ...string) *TwiMLResponse {
	t.body.WriteString("<Message>")
	if text != "" {
		t.body.WriteString(fmt.Sprintf("<Body>%s</Body>", escapeXML(text)))
	}
	for _, url := range mediaURLs {
		t.body.WriteString(fmt.Sprintf("<Media>%s</Media>", escapeXML(url)))
	}
	t.body.WriteString("</Message>")
	return t
}

// Redirect adds a Redirect element
func (t *TwiMLResponse) Redirect(url string) *TwiMLResponse {
	t.body.WriteString(fmt.Sprintf("<Redirect>%s</Redirect>", escapeXML(url)))
	return t
}

// String returns the complete TwiML response
func (t *TwiMLResponse) String() string {
	return fmt.Sprintf("<?xml version=\"1.0\" encoding=\"UTF-8\"?><Response>%s</Response>", t.body.String())
}

// Empty returns an empty TwiML response
func EmptyTwiMLResponse() string {
	return "<?xml version=\"1.0\" encoding=\"UTF-8\"?><Response></Response>"
}

// escapeXML escapes special XML characters
func escapeXML(s string) string {
	s = strings.ReplaceAll(s, "&", "&amp;")
	s = strings.ReplaceAll(s, "<", "&lt;")
	s = strings.ReplaceAll(s, ">", "&gt;")
	s = strings.ReplaceAll(s, "'", "&apos;")
	s = strings.ReplaceAll(s, "\"", "&quot;")
	return s
}
