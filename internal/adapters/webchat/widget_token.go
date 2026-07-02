package webchat

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/msgfy/linktor/internal/domain/entity"
)

// Widget token errors.
var (
	// ErrTokenMalformed is returned when the token structure is invalid.
	ErrTokenMalformed = errors.New("widget token malformed")
	// ErrTokenSignature is returned when the HMAC signature does not match.
	ErrTokenSignature = errors.New("widget token signature invalid")
	// ErrTokenChannelMismatch is returned when the token was issued for a
	// different channel than the one being accessed.
	ErrTokenChannelMismatch = errors.New("widget token channel mismatch")
	// ErrTokenExpired is returned when the token TTL has elapsed.
	ErrTokenExpired = errors.New("widget token expired")
	// ErrTokenMissingSecret is returned when no per-channel secret is provided.
	ErrTokenMissingSecret = errors.New("widget token secret missing")
)

// DefaultWidgetTokenTTL is the recommended short TTL for browser widget tokens.
const DefaultWidgetTokenTTL = 30 * time.Minute

var b64 = base64.RawURLEncoding

// widgetSecret returns the per-channel widget signing secret, preferring
// Config["widget_secret"] and falling back to Credentials["widget_secret"].
// Returns "" when the channel does not opt in to authenticated widget access.
func widgetSecret(ch *entity.Channel) string {
	if ch == nil {
		return ""
	}
	if v := strings.TrimSpace(ch.Config["widget_secret"]); v != "" {
		return v
	}
	if v := strings.TrimSpace(ch.Credentials["widget_secret"]); v != "" {
		return v
	}
	return ""
}

// IssueWidgetToken mints a signed, short-lived widget token for a channel.
//
// The token is an HMAC-SHA256 over the payload "channelID.expiryUnix" (with an
// optional session nonce appended as "channelID.expiryUnix.nonce"), signed with
// the per-channel secret. It is encoded as base64url(payload).base64url(sig).
//
// `now` is injected for deterministic testing; production callers pass
// time.Now(). Use IssueWidgetTokenWithNonce to bind a token to a session.
func IssueWidgetToken(channelID, secret string, ttl time.Duration, now time.Time) (string, error) {
	return IssueWidgetTokenWithNonce(channelID, secret, "", ttl, now)
}

// IssueWidgetTokenWithNonce is like IssueWidgetToken but binds an optional
// session nonce into the signed payload.
func IssueWidgetTokenWithNonce(channelID, secret, nonce string, ttl time.Duration, now time.Time) (string, error) {
	if secret == "" {
		return "", ErrTokenMissingSecret
	}
	if channelID == "" {
		return "", ErrTokenMalformed
	}
	if ttl <= 0 {
		ttl = DefaultWidgetTokenTTL
	}

	expiry := now.Add(ttl).Unix()
	payload := channelID + "." + strconv.FormatInt(expiry, 10)
	if nonce != "" {
		payload += "." + nonce
	}

	sig := sign(secret, payload)
	return b64.EncodeToString([]byte(payload)) + "." + b64.EncodeToString(sig), nil
}

// VerifyWidgetToken validates a widget token for the given channel and secret.
// It performs a constant-time signature comparison and rejects expired or
// tampered tokens. `now` is injected for deterministic testing.
func VerifyWidgetToken(token, channelID, secret string, now time.Time) error {
	if secret == "" {
		return ErrTokenMissingSecret
	}

	parts := strings.Split(token, ".")
	if len(parts) != 2 {
		return ErrTokenMalformed
	}

	payloadBytes, err := b64.DecodeString(parts[0])
	if err != nil {
		return ErrTokenMalformed
	}
	sigBytes, err := b64.DecodeString(parts[1])
	if err != nil {
		return ErrTokenMalformed
	}

	// Constant-time signature verification over the exact decoded payload.
	expected := sign(secret, string(payloadBytes))
	if !hmac.Equal(sigBytes, expected) {
		return ErrTokenSignature
	}

	// Payload is channelID.expiryUnix[.nonce].
	fields := strings.Split(string(payloadBytes), ".")
	if len(fields) < 2 {
		return ErrTokenMalformed
	}
	if fields[0] != channelID {
		return ErrTokenChannelMismatch
	}
	expiry, err := strconv.ParseInt(fields[1], 10, 64)
	if err != nil {
		return ErrTokenMalformed
	}
	if now.Unix() >= expiry {
		return ErrTokenExpired
	}

	return nil
}

func sign(secret, payload string) []byte {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(payload))
	return mac.Sum(nil)
}

// ipRateLimiter is a minimal in-memory token-bucket rate limiter keyed by
// client IP. It blunts abusive connection storms without external deps.
type ipRateLimiter struct {
	mu       sync.Mutex
	buckets  map[string]*tokenBucket
	rate     float64 // tokens refilled per second
	capacity float64 // burst size
}

type tokenBucket struct {
	tokens float64
	last   time.Time
}

// newIPRateLimiter builds a limiter allowing `capacity` bursts, refilling at
// `rate` tokens/second.
func newIPRateLimiter(rate, capacity float64) *ipRateLimiter {
	return &ipRateLimiter{
		buckets:  make(map[string]*tokenBucket),
		rate:     rate,
		capacity: capacity,
	}
}

// Allow reports whether a connection attempt from ip is permitted at time now,
// consuming one token when it is. `now` is injected for deterministic testing.
func (l *ipRateLimiter) Allow(ip string, now time.Time) bool {
	l.mu.Lock()
	defer l.mu.Unlock()

	b, ok := l.buckets[ip]
	if !ok {
		l.buckets[ip] = &tokenBucket{tokens: l.capacity - 1, last: now}
		return true
	}

	// Refill based on elapsed time.
	elapsed := now.Sub(b.last).Seconds()
	if elapsed > 0 {
		b.tokens += elapsed * l.rate
		if b.tokens > l.capacity {
			b.tokens = l.capacity
		}
		b.last = now
	}

	if b.tokens < 1 {
		return false
	}
	b.tokens--
	return true
}
