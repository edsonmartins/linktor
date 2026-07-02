package linktor

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"strconv"
	"time"
)

const (
	SignatureHeader = "X-Linktor-Signature"
	TimestampHeader = "X-Linktor-Timestamp"
	DefaultTolerance = 300 // 5 minutes
)

// WebhookEvent represents a webhook event
type WebhookEvent struct {
	ID        string                 `json:"id"`
	Type      string                 `json:"type"`
	Timestamp time.Time              `json:"timestamp"`
	TenantID  string                 `json:"tenantId"`
	Data      map[string]interface{} `json:"data"`
}

// ComputeSignature computes HMAC-SHA256 over the raw payload only. It is the
// low-level primitive; production signatures bind the timestamp too — see
// ComputeTimestampedSignature.
func ComputeSignature(payload []byte, secret string) string {
	h := hmac.New(sha256.New, []byte(secret))
	h.Write(payload)
	return hex.EncodeToString(h.Sum(nil))
}

// ComputeTimestampedSignature computes the HMAC-SHA256 over the Stripe-style
// signing input `timestamp + "." + payload`. This matches what the Linktor
// server places in X-Linktor-Signature, so the unix-seconds X-Linktor-Timestamp
// header is authenticated and cannot be replayed with a fresh timestamp.
func ComputeTimestampedSignature(timestamp string, payload []byte, secret string) string {
	signed := make([]byte, 0, len(timestamp)+1+len(payload))
	signed = append(signed, timestamp...)
	signed = append(signed, '.')
	signed = append(signed, payload...)
	return ComputeSignature(signed, secret)
}

// VerifyWebhookSignature verifies a signature that binds timestamp and payload.
func VerifyWebhookSignature(payload []byte, signature, timestamp, secret string) bool {
	if signature == "" || timestamp == "" || secret == "" {
		return false
	}

	expected := ComputeTimestampedSignature(timestamp, payload, secret)
	return hmac.Equal([]byte(signature), []byte(expected))
}

// VerifyWebhook verifies webhook signature and timestamp freshness. The
// timestamp is mandatory (it is part of the signed material) and stale
// timestamps outside the tolerance window are rejected (anti-replay).
func VerifyWebhook(payload []byte, headers http.Header, secret string, tolerance int) bool {
	if tolerance == 0 {
		tolerance = DefaultTolerance
	}

	signature := headers.Get(SignatureHeader)
	timestampStr := headers.Get(TimestampHeader)
	if signature == "" || timestampStr == "" {
		return false
	}

	ts, err := strconv.ParseInt(timestampStr, 10, 64)
	if err != nil {
		return false
	}
	now := time.Now().Unix()
	if math.Abs(float64(now-ts)) > float64(tolerance) {
		return false
	}

	return VerifyWebhookSignature(payload, signature, timestampStr, secret)
}

// ConstructEvent parses and validates a webhook event
func ConstructEvent(payload []byte, headers http.Header, secret string, tolerance int) (*WebhookEvent, error) {
	if !VerifyWebhook(payload, headers, secret, tolerance) {
		return nil, fmt.Errorf("webhook signature verification failed")
	}

	var event WebhookEvent
	if err := json.Unmarshal(payload, &event); err != nil {
		return nil, fmt.Errorf("invalid JSON payload: %w", err)
	}

	if event.ID == "" || event.Type == "" {
		return nil, fmt.Errorf("invalid webhook event structure")
	}

	return &event, nil
}

// WebhookHandler creates an HTTP handler for webhooks
func WebhookHandler(secret string, handlers map[string]func(*WebhookEvent)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		// Read body
		var payload []byte
		if r.Body != nil {
			var err error
			payload, err = io.ReadAll(r.Body)
			if err != nil {
				http.Error(w, "Failed to read body", http.StatusBadRequest)
				return
			}
		}

		event, err := ConstructEvent(payload, r.Header, secret, 0)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		if handler, ok := handlers[event.Type]; ok {
			handler(event)
		}

		w.WriteHeader(http.StatusOK)
	}
}
