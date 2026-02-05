package instagram

import (
	"encoding/json"
	"io"
	"net/http"

	"github.com/msgfy/linktor/internal/adapters/meta"
)

// WebhookHandler handles Instagram DM webhooks
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
		instagramID := entry.ID

		// Process messaging events
		for _, event := range entry.Messaging {
			if msg := ConvertIncomingMessage(&event, instagramID); msg != nil {
				messages = append(messages, msg)
			}
		}

		// Process standby events
		for _, event := range entry.Standby {
			if msg := ConvertIncomingMessage(&event, instagramID); msg != nil {
				messages = append(messages, msg)
			}
		}

		// Handle test events (changes array format)
		for _, change := range entry.Changes {
			if change.Field == "messages" {
				if msg := extractMessageFromChange(&change, instagramID); msg != nil {
					messages = append(messages, msg)
				}
			}
		}
	}

	return messages
}

// extractMessageFromChange extracts a message from a webhook change event
// This handles the test event format from Instagram webhooks
func extractMessageFromChange(change *meta.WebhookChange, instagramID string) *IncomingMessage {
	// The value can be a messaging event in test mode
	valueBytes, err := json.Marshal(change.Value)
	if err != nil {
		return nil
	}

	var event meta.MessagingEvent
	if err := json.Unmarshal(valueBytes, &event); err != nil {
		return nil
	}

	return ConvertIncomingMessage(&event, instagramID)
}

// GetInstagramIDFromPayload extracts the Instagram account ID from the webhook payload
func GetInstagramIDFromPayload(payload *WebhookPayload) string {
	if len(payload.Entry) > 0 {
		return payload.Entry[0].ID
	}
	return ""
}

// IsInstagramWebhook checks if the webhook is for Instagram
func IsInstagramWebhook(payload *WebhookPayload) bool {
	return payload.Object == "instagram"
}

// IsInstagramViaPageWebhook checks if the webhook is for Instagram via Facebook Page
// Instagram DMs via FB Page come as "page" object but contain Instagram-specific data
func IsInstagramViaPageWebhook(payload *WebhookPayload) bool {
	if payload.Object != "page" {
		return false
	}

	// Check if any entry has Instagram-related messaging
	for _, entry := range payload.Entry {
		for _, event := range entry.Messaging {
			// Instagram messages via page have specific patterns
			// The sender/recipient IDs are Instagram-scoped IDs
			if event.Sender.ID != "" && event.Message != nil {
				return true
			}
		}
	}

	return false
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
