package retry

import (
	"context"
	"fmt"
	"math"
	"time"
)

// Config configures retry behavior.
type Config struct {
	MaxAttempts   int
	InitialDelay  time.Duration
	MaxDelay      time.Duration
	BackoffFactor float64
}

// DefaultConfig returns a Config with sensible defaults.
func DefaultConfig() *Config {
	return &Config{
		MaxAttempts:   5,
		InitialDelay:  2 * time.Second,
		MaxDelay:      5 * time.Minute,
		BackoffFactor: 2.0,
	}
}

// Do executes fn with retry logic and exponential backoff. It returns nil on
// the first successful call, or the last error after all attempts are
// exhausted.
func Do(ctx context.Context, cfg *Config, fn func() error) error {
	if cfg == nil {
		cfg = DefaultConfig()
	}

	var lastErr error
	for attempt := 0; attempt < cfg.MaxAttempts; attempt++ {
		if err := ctx.Err(); err != nil {
			return err
		}

		lastErr = fn()
		if lastErr == nil {
			return nil
		}

		fmt.Printf("retry: attempt %d/%d failed: %v\n", attempt+1, cfg.MaxAttempts, lastErr)

		if attempt < cfg.MaxAttempts-1 {
			delay := cfg.delay(attempt)
			select {
			case <-time.After(delay):
			case <-ctx.Done():
				return ctx.Err()
			}
		}
	}

	return lastErr
}

// DoWithResult executes fn and returns the result on success, or retries with
// exponential backoff.
func DoWithResult[T any](ctx context.Context, cfg *Config, fn func() (T, error)) (T, error) {
	if cfg == nil {
		cfg = DefaultConfig()
	}

	var (
		zero    T
		lastErr error
	)
	for attempt := 0; attempt < cfg.MaxAttempts; attempt++ {
		if err := ctx.Err(); err != nil {
			return zero, err
		}

		result, err := fn()
		if err == nil {
			return result, nil
		}
		lastErr = err

		fmt.Printf("retry: attempt %d/%d failed: %v\n", attempt+1, cfg.MaxAttempts, lastErr)

		if attempt < cfg.MaxAttempts-1 {
			delay := cfg.delay(attempt)
			select {
			case <-time.After(delay):
			case <-ctx.Done():
				return zero, ctx.Err()
			}
		}
	}

	return zero, lastErr
}

// delay calculates the backoff duration for the given attempt index.
func (c *Config) delay(attempt int) time.Duration {
	d := time.Duration(float64(c.InitialDelay) * math.Pow(c.BackoffFactor, float64(attempt)))
	if d > c.MaxDelay {
		d = c.MaxDelay
	}
	return d
}
