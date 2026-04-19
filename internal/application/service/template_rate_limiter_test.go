package service

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWriteLimiter_AllowsUnderLimit(t *testing.T) {
	l := newTemplateWriteLimiter(3, time.Hour)
	for i := 0; i < 3; i++ {
		assert.NoError(t, l.Allow("waba-1"), "first %d calls must pass", i+1)
	}
}

func TestWriteLimiter_BlocksAtLimit(t *testing.T) {
	l := newTemplateWriteLimiter(2, time.Hour)
	require.NoError(t, l.Allow("waba-1"))
	require.NoError(t, l.Allow("waba-1"))

	err := l.Allow("waba-1")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "rate limit")
	assert.Contains(t, err.Error(), "waba-1")
}

func TestWriteLimiter_PerWABAIsolation(t *testing.T) {
	// Hitting the limit for waba-1 must not affect waba-2 — Meta counts
	// writes per-WABA, not globally.
	l := newTemplateWriteLimiter(1, time.Hour)
	require.NoError(t, l.Allow("waba-1"))
	require.Error(t, l.Allow("waba-1"))

	assert.NoError(t, l.Allow("waba-2"))
}

func TestWriteLimiter_ReleasesOldEntries(t *testing.T) {
	// Very short window so the first call falls out before the second.
	l := newTemplateWriteLimiter(1, 10*time.Millisecond)
	require.NoError(t, l.Allow("waba-1"))
	time.Sleep(15 * time.Millisecond)
	assert.NoError(t, l.Allow("waba-1"), "timestamp outside window must be pruned")
}

func TestWriteLimiter_ZeroLimitDisables(t *testing.T) {
	l := newTemplateWriteLimiter(0, time.Hour)
	for i := 0; i < 10000; i++ {
		require.NoError(t, l.Allow("waba-1"))
	}
}

func TestWriteLimiter_NilReceiver(t *testing.T) {
	// Defensive: calling Allow on a nil *templateWriteLimiter (e.g. in a
	// test that bypasses NewTemplateService) must be a no-op, not a panic.
	var l *templateWriteLimiter
	assert.NoError(t, l.Allow("waba-1"))
}
