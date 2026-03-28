package dedup

import (
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewCache(t *testing.T) {
	t.Run("default TTL", func(t *testing.T) {
		c := NewCache(0)
		require.NotNil(t, c)
		assert.Equal(t, 30*time.Minute, c.ttl)
		assert.NotNil(t, c.entries)
	})

	t.Run("custom TTL", func(t *testing.T) {
		c := NewCache(10 * time.Second)
		assert.Equal(t, 10*time.Second, c.ttl)
	})
}

func TestCache_MarkAndIsDuplicate(t *testing.T) {
	c := NewCache(time.Minute)

	key := "ch1:ext1:message"
	assert.False(t, c.IsDuplicate(key))

	c.Mark(key)
	assert.True(t, c.IsDuplicate(key))
}

func TestCache_NotDuplicate(t *testing.T) {
	c := NewCache(time.Minute)
	assert.False(t, c.IsDuplicate("never-marked"))
}

func TestCache_ExpiredEntry(t *testing.T) {
	c := NewCache(1 * time.Millisecond)
	c.Mark("key1")
	time.Sleep(5 * time.Millisecond)
	assert.False(t, c.IsDuplicate("key1"))
}

func TestCache_Cleanup(t *testing.T) {
	c := NewCache(5 * time.Millisecond)

	c.Mark("expire1")
	c.Mark("expire2")
	time.Sleep(10 * time.Millisecond)

	// These should still be present in the map (not cleaned up yet).
	assert.Equal(t, 2, c.Size())

	// Add a fresh entry.
	c.Mark("fresh")
	assert.Equal(t, 3, c.Size())

	c.Cleanup()
	assert.Equal(t, 1, c.Size())
	assert.True(t, c.IsDuplicate("fresh"))
	assert.False(t, c.IsDuplicate("expire1"))
}

func TestCache_Size(t *testing.T) {
	c := NewCache(time.Minute)
	assert.Equal(t, 0, c.Size())

	c.Mark("a")
	c.Mark("b")
	assert.Equal(t, 2, c.Size())

	// Re-marking the same key does not increase size.
	c.Mark("a")
	assert.Equal(t, 2, c.Size())
}

func TestCache_Clear(t *testing.T) {
	c := NewCache(time.Minute)
	c.Mark("a")
	c.Mark("b")
	assert.Equal(t, 2, c.Size())

	c.Clear()
	assert.Equal(t, 0, c.Size())
	assert.False(t, c.IsDuplicate("a"))
}

func TestBuildKey(t *testing.T) {
	key := BuildKey("ch1", "ext123", "message")
	assert.Equal(t, "ch1:ext123:message", key)
}

func TestCache_Concurrent(t *testing.T) {
	c := NewCache(time.Minute)
	const goroutines = 50
	const keysPerGoroutine = 100

	var wg sync.WaitGroup
	var duplicates atomic.Int64

	wg.Add(goroutines)
	for g := 0; g < goroutines; g++ {
		go func(id int) {
			defer wg.Done()
			for k := 0; k < keysPerGoroutine; k++ {
				key := BuildKey("ch", "ext", "msg")
				if c.IsDuplicate(key) {
					duplicates.Add(1)
				}
				c.Mark(key)
			}
		}(g)
	}
	wg.Wait()

	// The single shared key should be marked.
	assert.True(t, c.IsDuplicate(BuildKey("ch", "ext", "msg")))
	// Most calls after the first Mark should have been duplicates.
	assert.Greater(t, duplicates.Load(), int64(0))
}
