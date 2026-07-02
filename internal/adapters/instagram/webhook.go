package instagram

import (
	"crypto/subtle"
	"encoding/json"
	"io"
	"net/http"

	"github.com/msgfy/linktor/internal/adapters/meta"
)

// maxWebhookBodySize caps the number of bytes read from a webhook request body
// to protect against unbounded reads (DoS).
const maxWebhookBodySize = 1 << 20 // 1 MB

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

	if mode == "subscribe" && subtle.ConstantTimeCompare([]byte(token), []byte(h.verifyToken)) == 1 {
		return challenge, nil
	}

	return "", ErrInvalidVerifyToken
}

// ParseWebhook parses and validates an incoming webhook request (POST)
func (h *WebhookHandler) ParseWebhook(r *http.Request) (*WebhookPayload, error) {
	// Read body with a size limit to protect against unbounded reads (DoS)
	body, err := io.ReadAll(io.LimitReader(r.Body, maxWebhookBodySize))
	if err != nil {
		return nil, ErrReadBodyFailed
	}

	// Validate signature. Fail closed: without a configured app secret we cannot
	// authenticate the payload, so reject it rather than accept unsigned data.
	if h.appSecret == "" {
		return nil, ErrInvalidSignature
	}
	signature := r.Header.Get("X-Hub-Signature-256")
	if !meta.ValidateWebhookSignature(h.appSecret, body, signature) {
		return nil, ErrInvalidSignature
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

		// Process messaging events (DMs, story replies/mentions, reactions)
		for i := range entry.Messaging {
			messages = appendMessagingEvent(messages, &entry.Messaging[i], instagramID)
		}

		// Process standby events
		for i := range entry.Standby {
			messages = appendMessagingEvent(messages, &entry.Standby[i], instagramID)
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

// appendMessagingEvent converts a single messaging event into an IncomingMessage
// and appends it. It covers DMs and story replies/mentions (Message present) as
// well as reactions (message_reactions), which carry no Message and were
// previously dropped. Reaction text is set to the emoji so downstream consumers
// that treat IncomingMessage as a message do not produce empty content.
func appendMessagingEvent(messages []*IncomingMessage, event *meta.MessagingEvent, instagramID string) []*IncomingMessage {
	if msg := ConvertIncomingMessage(event, instagramID); msg != nil {
		return append(messages, msg)
	}
	if r := ConvertReaction(event, instagramID); r != nil {
		if r.Text == "" {
			r.Text = r.ReactionEmoji
		}
		return append(messages, r)
	}
	return messages
}

// ExtractReadReceipts extracts messaging_seen (read) events from the payload.
// These are kept separate from ExtractMessages so callers that treat every
// extracted item as a message are not flooded with empty read events.
func ExtractReadReceipts(payload *WebhookPayload) []*IncomingMessage {
	var reads []*IncomingMessage
	for _, entry := range payload.Entry {
		instagramID := entry.ID
		for i := range entry.Messaging {
			if r := ConvertRead(&entry.Messaging[i], instagramID); r != nil {
				reads = append(reads, r)
			}
		}
	}
	return reads
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

// IsInstagramViaPageWebhook reports whether the webhook is for Instagram DMs
// delivered via a Facebook Page. Such traffic arrives under the "page" object
// (indistinguishable at the object level from plain Messenger traffic), so the
// only reliable discriminator is the configured Instagram account ID: for a
// genuine Instagram-via-Page event the entry.ID (the IG account) or the
// recipient.ID matches the channel's instagram_id.
//
// Callers that know their Instagram account ID MUST pass it so pure Messenger
// traffic is not misclassified as Instagram. The variadic parameter preserves
// backwards compatibility for legacy callers that cannot supply an ID (in which
// case it falls back to the looser sender+message heuristic).
func IsInstagramViaPageWebhook(payload *WebhookPayload, instagramID ...string) bool {
	if payload.Object != "page" {
		return false
	}

	id := ""
	for _, v := range instagramID {
		if v != "" {
			id = v
			break
		}
	}

	// Without a configured Instagram ID we cannot reliably distinguish IG-via-Page
	// from plain Messenger; fall back to the legacy heuristic.
	if id == "" {
		for _, entry := range payload.Entry {
			for _, event := range entry.Messaging {
				if event.Sender.ID != "" && event.Message != nil {
					return true
				}
			}
		}
		return false
	}

	// Match the configured Instagram account against the entry (the account that
	// owns the event) or the recipient (the account receiving the DM).
	for _, entry := range payload.Entry {
		if entry.ID == id {
			return true
		}
		for _, event := range entry.Messaging {
			if event.Recipient.ID == id {
				return true
			}
		}
		for _, event := range entry.Standby {
			if event.Recipient.ID == id {
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
