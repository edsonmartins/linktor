package dedup

import (
	"fmt"
	"sync"
	"time"
)

// Cache provides in-memory message deduplication with TTL.
type Cache struct {
	mu      sync.RWMutex
	entries map[string]time.Time
	ttl     time.Duration
}

// NewCache creates a new deduplication cache. If ttl is zero, a default of 30
// minutes is used.
func NewCache(ttl time.Duration) *Cache {
	if ttl == 0 {
		ttl = 30 * time.Minute
	}
	return &Cache{
		entries: make(map[string]time.Time),
		ttl:     ttl,
	}
}

// IsDuplicate returns true if the key has already been marked and has not yet
// expired.
func (c *Cache) IsDuplicate(key string) bool {
	c.mu.RLock()
	defer c.mu.RUnlock()

	ts, ok := c.entries[key]
	if !ok {
		return false
	}
	return time.Since(ts) < c.ttl
}

// Mark records the key with the current timestamp.
func (c *Cache) Mark(key string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.entries[key] = time.Now()
}

// Cleanup removes all expired entries from the cache.
func (c *Cache) Cleanup() {
	c.mu.Lock()
	defer c.mu.Unlock()

	now := time.Now()
	for k, ts := range c.entries {
		if now.Sub(ts) >= c.ttl {
			delete(c.entries, k)
		}
	}
}

// Size returns the number of entries currently in the cache (including
// expired but not yet cleaned up).
func (c *Cache) Size() int {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return len(c.entries)
}

// Clear removes all entries from the cache.
func (c *Cache) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.entries = make(map[string]time.Time)
}

// BuildKey creates a composite deduplication key from the given components.
func BuildKey(channelID, externalID, eventType string) string {
	return fmt.Sprintf("%s:%s:%s", channelID, externalID, eventType)
}
