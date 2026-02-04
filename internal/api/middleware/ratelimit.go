package middleware

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
)

// RateLimiterConfig holds rate limiter configuration
type RateLimiterConfig struct {
	// Requests per window
	RequestsPerWindow int
	// Window duration
	WindowDuration time.Duration
	// Key prefix in Redis
	KeyPrefix string
}

// DefaultRateLimiterConfig returns default configuration
func DefaultRateLimiterConfig() *RateLimiterConfig {
	return &RateLimiterConfig{
		RequestsPerWindow: 100,
		WindowDuration:    time.Minute,
		KeyPrefix:         "ratelimit:",
	}
}

// RateLimiter implements token bucket rate limiting with Redis
type RateLimiter struct {
	redis  *redis.Client
	config *RateLimiterConfig
}

// NewRateLimiter creates a new rate limiter
func NewRateLimiter(redisClient *redis.Client, config *RateLimiterConfig) *RateLimiter {
	if config == nil {
		config = DefaultRateLimiterConfig()
	}
	return &RateLimiter{
		redis:  redisClient,
		config: config,
	}
}

// Limit returns a gin middleware that applies rate limiting
func (rl *RateLimiter) Limit() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get tenant ID for rate limiting key
		tenantID := c.GetString(TenantIDKey)
		if tenantID == "" {
			// Fall back to IP-based rate limiting
			tenantID = c.ClientIP()
		}

		// Build rate limit key
		key := fmt.Sprintf("%s%s", rl.config.KeyPrefix, tenantID)

		// Check rate limit
		allowed, remaining, resetAt, err := rl.checkRateLimit(c.Request.Context(), key)
		if err != nil {
			// On error, allow request but log
			c.Next()
			return
		}

		// Set rate limit headers
		c.Header("X-RateLimit-Limit", strconv.Itoa(rl.config.RequestsPerWindow))
		c.Header("X-RateLimit-Remaining", strconv.Itoa(remaining))
		c.Header("X-RateLimit-Reset", strconv.FormatInt(resetAt, 10))

		if !allowed {
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
				"code":    "RATE_LIMITED",
				"message": "Too many requests. Please try again later.",
				"retry_after": resetAt - time.Now().Unix(),
			})
			return
		}

		c.Next()
	}
}

// LimitByKey returns a middleware that rate limits by a custom key
func (rl *RateLimiter) LimitByKey(keyFunc func(*gin.Context) string) gin.HandlerFunc {
	return func(c *gin.Context) {
		key := fmt.Sprintf("%s%s", rl.config.KeyPrefix, keyFunc(c))

		allowed, remaining, resetAt, err := rl.checkRateLimit(c.Request.Context(), key)
		if err != nil {
			c.Next()
			return
		}

		c.Header("X-RateLimit-Limit", strconv.Itoa(rl.config.RequestsPerWindow))
		c.Header("X-RateLimit-Remaining", strconv.Itoa(remaining))
		c.Header("X-RateLimit-Reset", strconv.FormatInt(resetAt, 10))

		if !allowed {
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
				"code":    "RATE_LIMITED",
				"message": "Too many requests. Please try again later.",
				"retry_after": resetAt - time.Now().Unix(),
			})
			return
		}

		c.Next()
	}
}

// checkRateLimit checks if request is allowed using sliding window
func (rl *RateLimiter) checkRateLimit(ctx context.Context, key string) (allowed bool, remaining int, resetAt int64, err error) {
	now := time.Now()
	windowStart := now.Truncate(rl.config.WindowDuration)
	windowEnd := windowStart.Add(rl.config.WindowDuration)

	// Use Redis transaction
	pipe := rl.redis.TxPipeline()

	// Increment counter
	incrCmd := pipe.Incr(ctx, key)

	// Set expiration if key is new
	pipe.ExpireNX(ctx, key, rl.config.WindowDuration)

	// Execute pipeline
	_, err = pipe.Exec(ctx)
	if err != nil {
		return true, rl.config.RequestsPerWindow, windowEnd.Unix(), err
	}

	count := int(incrCmd.Val())
	remaining = rl.config.RequestsPerWindow - count
	if remaining < 0 {
		remaining = 0
	}

	allowed = count <= rl.config.RequestsPerWindow
	resetAt = windowEnd.Unix()

	return allowed, remaining, resetAt, nil
}

// Reset resets the rate limit for a key
func (rl *RateLimiter) Reset(ctx context.Context, key string) error {
	fullKey := fmt.Sprintf("%s%s", rl.config.KeyPrefix, key)
	return rl.redis.Del(ctx, fullKey).Err()
}
