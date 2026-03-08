package handlers

import (
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewAgentHub(t *testing.T) {
	gin.SetMode(gin.TestMode)

	hub := NewAgentHub()

	require.NotNil(t, hub)
	assert.NotNil(t, hub.clients)
	assert.NotNil(t, hub.tenants)
	assert.NotNil(t, hub.register)
	assert.NotNil(t, hub.unregister)
	assert.NotNil(t, hub.broadcast)
	assert.NotNil(t, hub.done)
}

func TestAgentHub_GetOnlineUsers_Empty(t *testing.T) {
	gin.SetMode(gin.TestMode)

	hub := NewAgentHub()
	users := hub.GetOnlineUsers("tenant-1")

	assert.Empty(t, users)
}

func TestAgentHub_SendToUser_NoClient(t *testing.T) {
	gin.SetMode(gin.TestMode)

	hub := NewAgentHub()

	// Should not panic when sending to a non-existent user
	msg := &WSMessage{
		Type:    WSEventNewMessage,
		Payload: map[string]string{"test": "data"},
	}
	hub.SendToUser("nonexistent-user", msg)
	// No panic = success
}

func TestWSMessage_Types(t *testing.T) {
	assert.Equal(t, "new_message", WSEventNewMessage)
	assert.Equal(t, "message_updated", WSEventMessageUpdated)
	assert.Equal(t, "conversation_updated", WSEventConversationUpdated)
	assert.Equal(t, "conversation_created", WSEventConversationCreated)
	assert.Equal(t, "typing", WSEventTyping)
	assert.Equal(t, "presence", WSEventPresence)
	assert.Equal(t, "error", WSEventError)
	assert.Equal(t, "connected", WSEventConnected)
}

func TestNewWebSocketHandler(t *testing.T) {
	gin.SetMode(gin.TestMode)

	hub := NewAgentHub()
	handler := NewWebSocketHandler(hub, "test-jwt-secret")

	require.NotNil(t, handler)
	assert.Equal(t, hub, handler.hub)
	assert.Equal(t, "test-jwt-secret", handler.jwtSecret)
}
