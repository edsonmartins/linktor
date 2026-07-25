package webhook

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"strconv"
	"time"
)

// signedMessage builds the Stripe-style signing input `timestamp + "." + body`.
// Signing the timestamp together with the body makes the X-Linktor-Timestamp
// header tamper-evident, closing the replay window that exists when only the
// body is authenticated.
func signedMessage(timestamp string, body []byte) []byte {
	msg := make([]byte, 0, len(timestamp)+1+len(body))
	msg = append(msg, timestamp...)
	msg = append(msg, '.')
	msg = append(msg, body...)
	return msg
}

// This file implements the canonical `linktor-channel-v1` outbound webhook
// contract consumed by external integrators (e.g. DeskLenz). The envelope is
// channel-agnostic: every supported channel rides the same shape and only
// `channelType` differs. Keep it in sync with `sdks/go/webhook.go` and the
// contract reference in desklenz-api (contracts/linktor-whatsapp-v1.md).

// Header names and anti-replay tolerance (mirrors sdks/go).
const (
	SignatureHeader  = "X-Linktor-Signature"
	TimestampHeader  = "X-Linktor-Timestamp"
	DefaultTolerance = 300 // seconds
)

// Event types emitted on the outbound webhook.
const (
	TypeMessageReceived  = "message.received"
	TypeMessageSent      = "message.sent"
	TypeMessageDelivered = "message.delivered"
	TypeMessageRead      = "message.read"
	TypeMessageFailed    = "message.failed"

	TypeContactCreated = "contact.created"

	TypeConversationCreated   = "conversation.created"
	TypeConversationAssigned  = "conversation.assigned"
	TypeConversationResolved  = "conversation.resolved"
	TypeConversationReopened  = "conversation.reopened"
	TypeConversationEscalated = "conversation.escalated"
)

// Envelope is the top-level `linktor-channel-v1` event.
type Envelope struct {
	ID        string    `json:"id"`
	Type      string    `json:"type"`
	Timestamp time.Time `json:"timestamp"` // RFC3339
	TenantID  string    `json:"tenantId"`
	// Environment is the channel's environment ("production"|"sandbox").
	// Additive, backward-compatible field (INV-018): consumers that predate it
	// keep parsing; absence means production.
	Environment string      `json:"environment,omitempty"`
	Data        interface{} `json:"data"`
}

// InboundData is the `data` payload for `message.received`.
type InboundData struct {
	Message        InboundMessagePayload `json:"message"`
	ConversationID string                `json:"conversationId"`
	ContactID      string                `json:"contactId"`
	ChannelID      string                `json:"channelId"`
	ChannelType    string                `json:"channelType"`
}

// InboundMessagePayload is the normalized message inside InboundData.
type InboundMessagePayload struct {
	ID          string            `json:"id"`
	Direction   string            `json:"direction"` // always "inbound"
	ContentType string            `json:"contentType"`
	Content     MessageContent    `json:"content"`
	SenderID    string            `json:"senderId,omitempty"`
	SenderType  string            `json:"senderType"` // "contact"
	Metadata    map[string]string `json:"metadata,omitempty"`
}

// MessageContent carries text and/or a single media attachment.
type MessageContent struct {
	Text  string        `json:"text,omitempty"`
	Media *MediaPayload `json:"media,omitempty"`
}

// MediaPayload describes an attachment by URL.
type MediaPayload struct {
	URL      string `json:"url"`
	MimeType string `json:"mimeType,omitempty"`
	Filename string `json:"filename,omitempty"`
	Size     int64  `json:"size,omitempty"`
	Caption  string `json:"caption,omitempty"`
}

// ContactData is the `data` payload for `contact.created`. ChannelID is the
// channel the contact first appeared on (the delivery is routed to that
// channel's webhook), so it is always set.
type ContactData struct {
	ContactID   string `json:"contactId"`
	Name        string `json:"name,omitempty"`
	Phone       string `json:"phone,omitempty"`
	Email       string `json:"email,omitempty"`
	ChannelID   string `json:"channelId"`
	ChannelType string `json:"channelType,omitempty"`
}

// ConversationData is the `data` payload for the
// `conversation.{created,assigned,resolved,reopened,escalated}` events.
type ConversationData struct {
	ConversationID string `json:"conversationId"`
	ChannelID      string `json:"channelId"`
	ContactID      string `json:"contactId"`
	Status         string `json:"status,omitempty"`
	AssignedUserID string `json:"assignedUserId,omitempty"`
	ChannelType    string `json:"channelType,omitempty"`
}

// StatusData is the `data` payload for `message.{sent,delivered,read,failed}`.
//
// ChannelID/ChannelType mirror every other data payload. Without them a consumer serving several
// channels on one endpoint cannot tell whose event this is, and one that resolves the tenant from
// the channel — rather than trusting the envelope's tenantId, which is the Linktor tenant — cannot
// process the event at all. VendaX was rejecting these with 400, which became retries and DLQ on
// our side. Additive and backward-compatible (INV-018), like `environment`.
type StatusData struct {
	MessageID   string    `json:"messageId"`
	Status      string    `json:"status"`
	Direction   string    `json:"direction"` // always "outbound"
	Timestamp   time.Time `json:"timestamp"`
	Error       string    `json:"error,omitempty"`
	ChannelID   string    `json:"channelId"`
	ChannelType string    `json:"channelType,omitempty"`
}

// ComputeSignature returns the hex-encoded HMAC-SHA256 of payload keyed by
// secret. It is the low-level HMAC primitive; the wire signature is produced by
// ComputeTimestampedSignature (which binds the timestamp too).
func ComputeSignature(payload []byte, secret string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(payload)
	return hex.EncodeToString(mac.Sum(nil))
}

// ComputeTimestampedSignature returns the hex-encoded HMAC-SHA256 over
// `timestamp + "." + payload`. This is the value placed in X-Linktor-Signature
// and the exact computation the consumer (DeskLenz / sdks/go) must verify
// against, so the X-Linktor-Timestamp header is covered by the signature.
func ComputeTimestampedSignature(timestamp string, payload []byte, secret string) string {
	return ComputeSignature(signedMessage(timestamp, payload), secret)
}

// SignatureHeaders builds the signing headers for a raw webhook body. `now` is
// injected so callers (and tests) control the timestamp deterministically. The
// signature covers both the timestamp and the body.
func SignatureHeaders(payload []byte, secret string, now time.Time) map[string]string {
	ts := strconv.FormatInt(now.Unix(), 10)
	return map[string]string{
		SignatureHeader: ComputeTimestampedSignature(ts, payload, secret),
		TimestampHeader: ts,
	}
}

// VerifySignature is the canonical verifier for the `linktor-channel-v1`
// outbound webhook contract. It recomputes the timestamped HMAC and rejects the
// request if the signature does not match (constant-time) or if the timestamp is
// outside toleranceSeconds of now (anti-replay). Pass tolerance <= 0 to use
// DefaultTolerance.
func VerifySignature(payload []byte, signature, timestamp, secret string, now time.Time, toleranceSeconds int) bool {
	if signature == "" || timestamp == "" || secret == "" {
		return false
	}
	if toleranceSeconds <= 0 {
		toleranceSeconds = DefaultTolerance
	}

	ts, err := strconv.ParseInt(timestamp, 10, 64)
	if err != nil {
		return false
	}
	delta := now.Unix() - ts
	if delta < 0 {
		delta = -delta
	}
	if delta > int64(toleranceSeconds) {
		return false // stale (or future-dated) → replay protection
	}

	expected := ComputeTimestampedSignature(timestamp, payload, secret)
	return hmac.Equal([]byte(expected), []byte(signature))
}

// MarshalEnvelope serializes an envelope to the exact JSON bytes that are both
// signed and delivered (sign-what-you-send).
func MarshalEnvelope(env *Envelope) ([]byte, error) {
	return json.Marshal(env)
}
