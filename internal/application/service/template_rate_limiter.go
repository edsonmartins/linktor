package service

import (
	"fmt"
	"sync"
	"time"
)

// templateWriteLimiter caps the rate at which we send mutating requests
// (create + edit) to Meta's template API. Meta rejects WABA-level bursts
// with error code 80008 after roughly 100 operations per hour; we enforce
// a slightly more conservative bucket (default 80/h) so a flapping admin
// loop never pushes us past Meta's ceiling.
//
// The limiter is scoped per WABA: each WhatsApp Business Account gets its
// own token bucket, since Meta counts the limit per-WABA. Multi-tenant
// instances sharing a single binary therefore can't knock each other out.
type templateWriteLimiter struct {
	mu      sync.Mutex
	buckets map[string]*templateBucket
	limit   int           // max writes allowed in window
	window  time.Duration // rolling window length
}

type templateBucket struct {
	timestamps []time.Time
}

// newTemplateWriteLimiter returns a limiter that allows `limit` writes
// per `window`. Passing limit=0 disables the limiter (all requests pass).
func newTemplateWriteLimiter(limit int, window time.Duration) *templateWriteLimiter {
	return &templateWriteLimiter{
		buckets: make(map[string]*templateBucket),
		limit:   limit,
		window:  window,
	}
}

// Allow reports whether a write against `wabaID` may proceed. On true,
// the bucket records the timestamp; on false, the caller must bubble up
// the rate-limit error instead of risking an 80008 from Meta.
func (l *templateWriteLimiter) Allow(wabaID string) error {
	if l == nil || l.limit == 0 {
		return nil
	}
	l.mu.Lock()
	defer l.mu.Unlock()

	now := time.Now()
	cutoff := now.Add(-l.window)

	b := l.buckets[wabaID]
	if b == nil {
		b = &templateBucket{}
		l.buckets[wabaID] = b
	}

	// Drop timestamps outside the window. Slice stays in chrono order so
	// a simple prefix-skip is enough.
	pruned := b.timestamps[:0]
	for _, ts := range b.timestamps {
		if ts.After(cutoff) {
			pruned = append(pruned, ts)
		}
	}
	b.timestamps = pruned

	if len(b.timestamps) >= l.limit {
		// Tell the caller when the next slot opens so the UI can surface it.
		oldest := b.timestamps[0]
		retry := oldest.Add(l.window).Sub(now)
		return fmt.Errorf("template write rate limit reached for WABA %s: %d writes per %s; retry in %s",
			wabaID, l.limit, l.window, retry.Round(time.Second))
	}

	b.timestamps = append(b.timestamps, now)
	return nil
}
