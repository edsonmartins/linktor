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

// ComputeSignature computes HMAC-SHA256 signature
func ComputeSignature(payload []byte, secret string) string {
	h := hmac.New(sha256.New, []byte(secret))
	h.Write(payload)
	return hex.EncodeToString(h.Sum(nil))
}

// VerifyWebhookSignature verifies the webhook signature
func VerifyWebhookSignature(payload []byte, signature, secret string) bool {
	if signature == "" || secret == "" {
		return false
	}

	expected := ComputeSignature(payload, secret)
	return hmac.Equal([]byte(signature), []byte(expected))
}

// VerifyWebhook verifies webhook with timestamp validation
func VerifyWebhook(payload []byte, headers http.Header, secret string, tolerance int) bool {
	if tolerance == 0 {
		tolerance = DefaultTolerance
	}

	signature := headers.Get(SignatureHeader)
	if signature == "" {
		return false
	}

	// Verify timestamp if present
	timestampStr := headers.Get(TimestampHeader)
	if timestampStr != "" {
		ts, err := strconv.ParseInt(timestampStr, 10, 64)
		if err != nil {
			return false
		}

		now := time.Now().Unix()
		if math.Abs(float64(now-ts)) > float64(tolerance) {
			return false
		}
	}

	return VerifyWebhookSignature(payload, signature, secret)
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
