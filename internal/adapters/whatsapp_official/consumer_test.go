package whatsapp_official

import (
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMessageDeduplicator_IsDuplicate(t *testing.T) {
	t.Run("new message ID returns false", func(t *testing.T) {
		dedup := NewMessageDeduplicator(1 * time.Minute)

		result := dedup.IsDuplicate("msg-001")
		assert.False(t, result, "new message ID should not be duplicate")
	})

	t.Run("after MarkProcessed returns true", func(t *testing.T) {
		dedup := NewMessageDeduplicator(1 * time.Minute)

		dedup.MarkProcessed("msg-002")
		result := dedup.IsDuplicate("msg-002")
		assert.True(t, result, "processed message ID should be duplicate")
	})

	t.Run("different message ID returns false", func(t *testing.T) {
		dedup := NewMessageDeduplicator(1 * time.Minute)

		dedup.MarkProcessed("msg-003")

		result := dedup.IsDuplicate("msg-004")
		assert.False(t, result, "different message ID should not be duplicate")
	})
}

func TestMessageDeduplicator_MarkProcessed(t *testing.T) {
	dedup := NewMessageDeduplicator(1 * time.Minute)

	require.False(t, dedup.IsDuplicate("msg-100"), "should not be duplicate before marking")

	dedup.MarkProcessed("msg-100")

	assert.True(t, dedup.IsDuplicate("msg-100"), "should be duplicate after marking")
}

func TestMessageDeduplicator_Concurrent(t *testing.T) {
	dedup := NewMessageDeduplicator(1 * time.Minute)

	const goroutines = 50
	const messagesPerGoroutine = 100

	var wg sync.WaitGroup
	wg.Add(goroutines)

	for g := 0; g < goroutines; g++ {
		go func(id int) {
			defer wg.Done()
			for i := 0; i < messagesPerGoroutine; i++ {
				msgID := fmt.Sprintf("msg-%d-%d", id, i)
				_ = dedup.IsDuplicate(msgID)
				dedup.MarkProcessed(msgID)
				_ = dedup.IsDuplicate(msgID)
			}
		}(g)
	}

	wg.Wait()

	// Verify a sample of processed messages are detected as duplicates
	assert.True(t, dedup.IsDuplicate("msg-0-0"), "previously processed message should be duplicate")
	assert.True(t, dedup.IsDuplicate("msg-49-99"), "previously processed message should be duplicate")
	assert.False(t, dedup.IsDuplicate("msg-never-seen"), "unseen message should not be duplicate")
}
