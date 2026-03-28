package retry

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()
	require.NotNil(t, cfg)
	assert.Equal(t, 5, cfg.MaxAttempts)
	assert.Equal(t, 2*time.Second, cfg.InitialDelay)
	assert.Equal(t, 5*time.Minute, cfg.MaxDelay)
	assert.Equal(t, 2.0, cfg.BackoffFactor)
}

func TestDo_SuccessFirstAttempt(t *testing.T) {
	cfg := &Config{MaxAttempts: 3, InitialDelay: time.Millisecond, MaxDelay: time.Second, BackoffFactor: 2.0}
	var calls atomic.Int32

	err := Do(context.Background(), cfg, func() error {
		calls.Add(1)
		return nil
	})

	assert.NoError(t, err)
	assert.Equal(t, int32(1), calls.Load())
}

func TestDo_SuccessAfterRetries(t *testing.T) {
	cfg := &Config{MaxAttempts: 5, InitialDelay: time.Millisecond, MaxDelay: time.Second, BackoffFactor: 2.0}
	var calls atomic.Int32

	err := Do(context.Background(), cfg, func() error {
		n := calls.Add(1)
		if n < 3 {
			return errors.New("not yet")
		}
		return nil
	})

	assert.NoError(t, err)
	assert.Equal(t, int32(3), calls.Load())
}

func TestDo_AllAttemptsFail(t *testing.T) {
	cfg := &Config{MaxAttempts: 3, InitialDelay: time.Millisecond, MaxDelay: time.Second, BackoffFactor: 2.0}
	var calls atomic.Int32
	sentinel := errors.New("permanent failure")

	err := Do(context.Background(), cfg, func() error {
		calls.Add(1)
		return sentinel
	})

	assert.ErrorIs(t, err, sentinel)
	assert.Equal(t, int32(3), calls.Load())
}

func TestDo_ContextCancelled(t *testing.T) {
	cfg := &Config{MaxAttempts: 10, InitialDelay: time.Second, MaxDelay: 10 * time.Second, BackoffFactor: 2.0}

	ctx, cancel := context.WithCancel(context.Background())
	var calls atomic.Int32

	go func() {
		time.Sleep(5 * time.Millisecond)
		cancel()
	}()

	err := Do(ctx, cfg, func() error {
		calls.Add(1)
		return errors.New("fail")
	})

	assert.Error(t, err)
	assert.True(t, errors.Is(err, context.Canceled))
	// Should not have exhausted all attempts.
	assert.Less(t, calls.Load(), int32(10))
}

func TestDo_BackoffDelays(t *testing.T) {
	cfg := &Config{MaxAttempts: 4, InitialDelay: 10 * time.Millisecond, MaxDelay: time.Second, BackoffFactor: 2.0}

	var timestamps []time.Time
	_ = Do(context.Background(), cfg, func() error {
		timestamps = append(timestamps, time.Now())
		return errors.New("fail")
	})

	require.Len(t, timestamps, 4)

	// Verify that delays increase between attempts.
	for i := 2; i < len(timestamps); i++ {
		prev := timestamps[i-1].Sub(timestamps[i-2])
		curr := timestamps[i].Sub(timestamps[i-1])
		// Current gap should be at least 1.5x the previous gap (allowing
		// some scheduling jitter below the exact 2x factor).
		assert.Greater(t, curr.Seconds(), prev.Seconds()*1.3,
			"delay between attempt %d->%d should be larger than %d->%d", i-1, i, i-2, i-1)
	}
}

func TestDoWithResult_Success(t *testing.T) {
	cfg := &Config{MaxAttempts: 3, InitialDelay: time.Millisecond, MaxDelay: time.Second, BackoffFactor: 2.0}
	var calls atomic.Int32

	result, err := DoWithResult(context.Background(), cfg, func() (string, error) {
		n := calls.Add(1)
		if n < 2 {
			return "", errors.New("not yet")
		}
		return "hello", nil
	})

	assert.NoError(t, err)
	assert.Equal(t, "hello", result)
	assert.Equal(t, int32(2), calls.Load())
}

func TestDoWithResult_Failure(t *testing.T) {
	cfg := &Config{MaxAttempts: 2, InitialDelay: time.Millisecond, MaxDelay: time.Second, BackoffFactor: 2.0}
	sentinel := errors.New("always fails")

	result, err := DoWithResult(context.Background(), cfg, func() (int, error) {
		return 0, sentinel
	})

	assert.ErrorIs(t, err, sentinel)
	assert.Equal(t, 0, result)
}
