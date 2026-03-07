package entity

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewConversation(t *testing.T) {
	conv := NewConversation("tenant1", "contact1", "channel1")
	assert.Equal(t, ConversationStatusOpen, conv.Status)
	assert.Equal(t, ConversationPriorityNormal, conv.Priority)
	assert.NotNil(t, conv.Tags)
	assert.NotNil(t, conv.Metadata)
	assert.Equal(t, "tenant1", conv.TenantID)
	assert.Equal(t, "contact1", conv.ContactID)
	assert.Equal(t, "channel1", conv.ChannelID)
}

func TestConversation_IsOpen(t *testing.T) {
	tests := []struct {
		name   string
		status ConversationStatus
		want   bool
	}{
		{"open", ConversationStatusOpen, true},
		{"pending", ConversationStatusPending, true},
		{"resolved", ConversationStatusResolved, false},
		{"closed", ConversationStatusClosed, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			conv := &Conversation{Status: tt.status}
			assert.Equal(t, tt.want, conv.IsOpen())
		})
	}
}

func TestConversation_Resolve(t *testing.T) {
	conv := NewConversation("t", "c", "ch")
	conv.Resolve()
	assert.Equal(t, ConversationStatusResolved, conv.Status)
	assert.NotNil(t, conv.ResolvedAt)
}

func TestConversation_Reopen(t *testing.T) {
	conv := NewConversation("t", "c", "ch")
	conv.Resolve()
	conv.Reopen()
	assert.Equal(t, ConversationStatusOpen, conv.Status)
	assert.Nil(t, conv.ResolvedAt)
}

func TestConversation_Assign(t *testing.T) {
	conv := NewConversation("t", "c", "ch")
	conv.Assign("user1")
	assert.NotNil(t, conv.AssignedUserID)
	assert.Equal(t, "user1", *conv.AssignedUserID)
}

func TestConversation_Unassign(t *testing.T) {
	conv := NewConversation("t", "c", "ch")
	conv.Assign("user1")
	conv.Unassign()
	assert.Nil(t, conv.AssignedUserID)
}
