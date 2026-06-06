package middleware

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
)

// WebhookDedup provides Redis-backed deduplication for incoming webhook events.
//
// Rationale: providers (Meta, Twilio, Telegram, etc.) routinely retry webhooks
// when our endpoint is slow or returns 5xx, which can cause duplicate message
// ingestion if events are not idempotent at the consumer layer.
//
// Strategy: extract a stable event identifier from the request (provider-supplied
// header when available, otherwise a SHA-256 of the request body) and use Redis
// SETNX with a TTL to record "seen" identifiers. If the key already exists, we
// short-circuit with HTTP 200 — the original processing already ran (or is
// running in another worker) and retrying would duplicate side effects.
type WebhookDedup struct {
	redis  *redis.Client
	prefix string
	ttl    time.Duration
}

// NewWebhookDedup creates a new webhook deduplicator.
// ttl should comfortably exceed the longest provider retry window (Meta retries
// up to 24h; using a few hours strikes a balance between memory and protection).
func NewWebhookDedup(client *redis.Client, ttl time.Duration) *WebhookDedup {
	if ttl <= 0 {
		ttl = 6 * time.Hour
	}
	return &WebhookDedup{
		redis:  client,
		prefix: "webhook:dedup:",
		ttl:    ttl,
	}
}

// Middleware returns a gin handler that deduplicates webhook deliveries.
// channelKey identifies the source (e.g. "whatsapp", "twilio") so that event IDs
// from different providers do not collide.
func (d *WebhookDedup) Middleware(channelKey string) gin.HandlerFunc {
	return func(c *gin.Context) {
		if d == nil || d.redis == nil {
			c.Next()
			return
		}

		eventID := extractEventID(c)
		if eventID == "" {
			body, err := io.ReadAll(c.Request.Body)
			if err != nil {
				c.Next()
				return
			}
			c.Request.Body = io.NopCloser(bytes.NewReader(body))
			sum := sha256.Sum256(body)
			eventID = hex.EncodeToString(sum[:])
		}

		key := d.prefix + channelKey + ":" + eventID

		ctx, cancel := context.WithTimeout(c.Request.Context(), 250*time.Millisecond)
		defer cancel()

		ok, err := d.redis.SetNX(ctx, key, "1", d.ttl).Result()
		if err != nil {
			c.Next()
			return
		}
		if !ok {
			c.Header("X-Webhook-Dedup", "duplicate")
			c.AbortWithStatusJSON(http.StatusOK, gin.H{
				"status":   "duplicate",
				"event_id": eventID,
			})
			return
		}

		c.Header("X-Webhook-Dedup", "fresh")
		c.Next()
	}
}

// extractEventID returns the first non-empty value from a known set of headers
// commonly used by webhook providers to carry a unique event identifier.
func extractEventID(c *gin.Context) string {
	headers := []string{
		"X-Hub-Signature-256",
		"X-Webhook-Id",
		"X-Event-Id",
		"X-Twilio-Idempotency-Token",
		"X-Telegram-Bot-Api-Secret-Token",
		"Idempotency-Key",
	}
	for _, h := range headers {
		if v := c.GetHeader(h); v != "" {
			return v
		}
	}
	return ""
}
