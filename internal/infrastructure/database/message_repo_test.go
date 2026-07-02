package database

import (
	"testing"

	"github.com/msgfy/linktor/internal/domain/entity"
	"github.com/stretchr/testify/assert"
)

// TestMessageStatusRank_Monotonic verifies the delivery-status ranking used by
// UpdateStatus to prevent an out-of-order event from regressing a message.
func TestMessageStatusRank_Monotonic(t *testing.T) {
	// pending < sent < delivered < read
	assert.Less(t, messageStatusRank(entity.MessageStatusPending), messageStatusRank(entity.MessageStatusSent))
	assert.Less(t, messageStatusRank(entity.MessageStatusSent), messageStatusRank(entity.MessageStatusDelivered))
	assert.Less(t, messageStatusRank(entity.MessageStatusDelivered), messageStatusRank(entity.MessageStatusRead))

	// A redelivered "sent" ranks below "delivered"/"read", so the SQL guard
	// (currentRank <= newRank) blocks it and the status cannot regress.
	assert.Greater(t, messageStatusRank(entity.MessageStatusDelivered), messageStatusRank(entity.MessageStatusSent))
	assert.Greater(t, messageStatusRank(entity.MessageStatusRead), messageStatusRank(entity.MessageStatusSent))

	// "failed" sits at the "sent" tier so a later successful delivery receipt can
	// still advance the message, while "failed" itself is always writable in code.
	assert.Equal(t, messageStatusRank(entity.MessageStatusSent), messageStatusRank(entity.MessageStatusFailed))
}
