package facebook

import (
	"encoding/json"
	"io"
	"net/http"

	"github.com/msgfy/linktor/internal/adapters/meta"
)

// WebhookHandler handles Facebook Messenger webhooks
type WebhookHandler struct {
	appSecret   string
	verifyToken string
}

// NewWebhookHandler creates a new webhook handler
func NewWebhookHandler(appSecret, verifyToken string) *WebhookHandler {
	return &WebhookHandler{
		appSecret:   appSecret,
		verifyToken: verifyToken,
	}
}

// VerifyWebhook handles webhook verification requests (GET)
func (h *WebhookHandler) VerifyWebhook(r *http.Request) (string, error) {
	mode := r.URL.Query().Get("hub.mode")
	token := r.URL.Query().Get("hub.verify_token")
	challenge := r.URL.Query().Get("hub.challenge")

	if mode == "subscribe" && token == h.verifyToken {
		return challenge, nil
	}

	return "", ErrInvalidVerifyToken
}

// ParseWebhook parses and validates an incoming webhook request (POST)
func (h *WebhookHandler) ParseWebhook(r *http.Request) (*WebhookPayload, error) {
	// Read body
	body, err := io.ReadAll(r.Body)
	if err != nil {
		return nil, ErrReadBodyFailed
	}

	// Validate signature if app secret is configured
	if h.appSecret != "" {
		signature := r.Header.Get("X-Hub-Signature-256")
		if !meta.ValidateWebhookSignature(h.appSecret, body, signature) {
			return nil, ErrInvalidSignature
		}
	}

	// Parse payload
	var payload meta.WebhookPayload
	if err := json.Unmarshal(body, &payload); err != nil {
		return nil, ErrParsePayloadFailed
	}

	return &payload, nil
}

// ExtractMessages extracts incoming messages from the webhook payload
func ExtractMessages(payload *WebhookPayload) []*IncomingMessage {
	var messages []*IncomingMessage

	for _, entry := range payload.Entry {
		pageID := entry.ID

		// Process messaging events
		for _, event := range entry.Messaging {
			if msg := ConvertIncomingMessage(&event, pageID); msg != nil {
				messages = append(messages, msg)
			}
		}

		// Process standby events (messages from other apps)
		for _, event := range entry.Standby {
			if msg := ConvertIncomingMessage(&event, pageID); msg != nil {
				messages = append(messages, msg)
			}
		}
	}

	return messages
}

// ExtractDeliveryStatuses extracts delivery status updates from the webhook payload
func ExtractDeliveryStatuses(payload *WebhookPayload) []*DeliveryStatus {
	var statuses []*DeliveryStatus

	for _, entry := range payload.Entry {
		for _, event := range entry.Messaging {
			if status := ConvertDeliveryStatus(&event); status != nil {
				statuses = append(statuses, status)
			}
		}
	}

	return statuses
}

// ExtractReadStatuses extracts read status updates from the webhook payload
func ExtractReadStatuses(payload *WebhookPayload) []*ReadStatus {
	var statuses []*ReadStatus

	for _, entry := range payload.Entry {
		for _, event := range entry.Messaging {
			if status := ConvertReadStatus(&event); status != nil {
				statuses = append(statuses, status)
			}
		}
	}

	return statuses
}

// ExtractPostbacks extracts postback events from the webhook payload
func ExtractPostbacks(payload *WebhookPayload) []*Postback {
	var postbacks []*Postback

	for _, entry := range payload.Entry {
		for _, event := range entry.Messaging {
			if pb := ConvertPostback(&event); pb != nil {
				postbacks = append(postbacks, pb)
			}
		}
	}

	return postbacks
}

// GetPageIDFromPayload extracts the page ID from the webhook payload
func GetPageIDFromPayload(payload *WebhookPayload) string {
	if len(payload.Entry) > 0 {
		return payload.Entry[0].ID
	}
	return ""
}

// IsMessengerWebhook checks if the webhook is for Messenger
func IsMessengerWebhook(payload *WebhookPayload) bool {
	return payload.Object == "page"
}

// Webhook errors
var (
	ErrInvalidVerifyToken = &WebhookError{Code: "invalid_verify_token", Message: "Invalid verify token"}
	ErrInvalidSignature   = &WebhookError{Code: "invalid_signature", Message: "Invalid webhook signature"}
	ErrReadBodyFailed     = &WebhookError{Code: "read_body_failed", Message: "Failed to read request body"}
	ErrParsePayloadFailed = &WebhookError{Code: "parse_payload_failed", Message: "Failed to parse webhook payload"}
)

// WebhookError represents a webhook processing error
type WebhookError struct {
	Code    string
	Message string
}

func (e *WebhookError) Error() string {
	return e.Message
}
