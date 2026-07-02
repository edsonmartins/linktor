package webchat

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewHub(t *testing.T) {
	hub := NewHub()
	require.NotNil(t, hub)
	assert.Equal(t, 0, hub.ClientCount())
}

func TestHub_RegisterUnregister(t *testing.T) {
	hub := NewHub()
	go hub.Run()
	defer hub.Stop()

	// Give hub time to start
	time.Sleep(10 * time.Millisecond)

	// Create client without real websocket (nil conn)
	client := &Client{
		hub:       hub,
		SessionID: "session-1",
		send:      make(chan *WebSocketMessage, 256),
		Metadata:  make(map[string]string),
	}

	// Register
	hub.register <- client
	time.Sleep(10 * time.Millisecond)

	assert.Equal(t, 1, hub.ClientCount())
	assert.Equal(t, client, hub.GetClient("session-1"))
	assert.Nil(t, hub.GetClient("nonexistent"))

	// Unregister
	hub.unregister <- client
	time.Sleep(10 * time.Millisecond)

	assert.Equal(t, 0, hub.ClientCount())
	assert.Nil(t, hub.GetClient("session-1"))
}

func TestHub_ConversationTracking(t *testing.T) {
	hub := NewHub()
	go hub.Run()
	defer hub.Stop()
	time.Sleep(10 * time.Millisecond)

	client1 := &Client{
		hub:            hub,
		SessionID:      "session-1",
		ConversationID: "conv-1",
		send:           make(chan *WebSocketMessage, 256),
		Metadata:       make(map[string]string),
	}
	client2 := &Client{
		hub:            hub,
		SessionID:      "session-2",
		ConversationID: "conv-1",
		send:           make(chan *WebSocketMessage, 256),
		Metadata:       make(map[string]string),
	}

	hub.register <- client1
	hub.register <- client2
	time.Sleep(10 * time.Millisecond)

	clients := hub.GetConversationClients("conv-1")
	assert.Len(t, clients, 2)

	// Empty conversation
	clients = hub.GetConversationClients("conv-nonexistent")
	assert.Len(t, clients, 0)

	// Unregister one client
	hub.unregister <- client1
	time.Sleep(10 * time.Millisecond)

	clients = hub.GetConversationClients("conv-1")
	assert.Len(t, clients, 1)

	// Unregister last client - conversation cleaned up
	hub.unregister <- client2
	time.Sleep(10 * time.Millisecond)

	clients = hub.GetConversationClients("conv-1")
	assert.Len(t, clients, 0)
}

func TestHub_BroadcastToConversation(t *testing.T) {
	hub := NewHub()
	go hub.Run()
	defer hub.Stop()
	time.Sleep(10 * time.Millisecond)

	client1 := &Client{
		hub:            hub,
		SessionID:      "session-1",
		ConversationID: "conv-1",
		send:           make(chan *WebSocketMessage, 256),
		Metadata:       make(map[string]string),
	}
	client2 := &Client{
		hub:            hub,
		SessionID:      "session-2",
		ConversationID: "conv-1",
		send:           make(chan *WebSocketMessage, 256),
		Metadata:       make(map[string]string),
	}
	client3 := &Client{
		hub:            hub,
		SessionID:      "session-3",
		ConversationID: "conv-2", // Different conversation
		send:           make(chan *WebSocketMessage, 256),
		Metadata:       make(map[string]string),
	}

	hub.register <- client1
	hub.register <- client2
	hub.register <- client3
	time.Sleep(10 * time.Millisecond)

	// Broadcast to conv-1
	msg := &WebSocketMessage{
		Type: MessageTypeMessage,
		Payload: MessagePayload{
			Content: "hello conv-1",
		},
	}
	hub.BroadcastToConversation("conv-1", msg)
	time.Sleep(10 * time.Millisecond)

	// client1 and client2 should receive, client3 should not
	assert.Len(t, client1.send, 1)
	assert.Len(t, client2.send, 1)
	assert.Len(t, client3.send, 0)

	received := <-client1.send
	assert.Equal(t, "hello conv-1", received.Payload.Content)
}

func TestHub_Broadcast(t *testing.T) {
	hub := NewHub()
	go hub.Run()
	defer hub.Stop()
	time.Sleep(10 * time.Millisecond)

	client := &Client{
		hub:       hub,
		SessionID: "session-1",
		send:      make(chan *WebSocketMessage, 256),
		Metadata:  make(map[string]string),
	}
	hub.register <- client
	time.Sleep(10 * time.Millisecond)

	hub.broadcast <- &BroadcastMessage{
		SessionID: "session-1",
		Message: &WebSocketMessage{
			Type:    MessageTypeMessage,
			Payload: MessagePayload{Content: "direct"},
		},
	}
	time.Sleep(10 * time.Millisecond)

	assert.Len(t, client.send, 1)
	received := <-client.send
	assert.Equal(t, "direct", received.Payload.Content)
}

func TestClient_SendMessage(t *testing.T) {
	hub := NewHub()
	client := NewClient(hub, nil, "session-1")

	msg := &WebSocketMessage{
		Type:    MessageTypeMessage,
		Payload: MessagePayload{Content: "test"},
	}

	err := client.SendMessage(msg)
	assert.NoError(t, err)

	// Read from channel
	received := <-client.send
	assert.Equal(t, "test", received.Payload.Content)
}

func TestClient_SetHandlers(t *testing.T) {
	hub := NewHub()
	client := NewClient(hub, nil, "session-1")

	msgCalled := false
	client.SetMessageHandler(func(msg *MessagePayload) error {
		msgCalled = true
		return nil
	})

	disconnCalled := false
	client.SetDisconnectHandler(func() {
		disconnCalled = true
	})

	assert.NotNil(t, client.onMessage)
	assert.NotNil(t, client.onDisconnect)

	client.onMessage(&MessagePayload{})
	assert.True(t, msgCalled)

	client.onDisconnect()
	assert.True(t, disconnCalled)
}

func TestClient_SendMessage_BufferFull(t *testing.T) {
	hub := NewHub()

	// Client with a tiny buffer so we can fill it deterministically.
	client := &Client{
		hub:       hub,
		SessionID: "session-full",
		send:      make(chan *WebSocketMessage, 1),
		Metadata:  make(map[string]string),
	}

	msg := &WebSocketMessage{Type: MessageTypeMessage, Payload: MessagePayload{Content: "x"}}

	// First send fills the buffer.
	require.NoError(t, client.SendMessage(msg))

	// Second send must fail (buffer full) instead of silently dropping.
	err := client.SendMessage(msg)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "buffer full")
}

func TestHub_StopWithActiveClient_NoPanicNoLeak(t *testing.T) {
	hub := NewHub()
	go hub.Run()
	time.Sleep(10 * time.Millisecond)

	client := &Client{
		hub:       hub,
		SessionID: "session-active",
		send:      make(chan *WebSocketMessage, 1),
		Metadata:  make(map[string]string),
	}
	hub.register <- client
	time.Sleep(10 * time.Millisecond)
	require.Equal(t, 1, hub.ClientCount())

	// Stop the hub while a client is still registered. Run closes the client's
	// send channel on shutdown.
	hub.Stop()
	time.Sleep(10 * time.Millisecond)

	// A send after Stop must NOT panic (send on closed channel) and must
	// report an error rather than a false success.
	assert.NotPanics(t, func() {
		err := client.SendMessage(&WebSocketMessage{
			Type:    MessageTypeMessage,
			Payload: MessagePayload{Content: "after stop"},
		})
		assert.Error(t, err)
	})

	// Register / Unregister after Stop must return promptly (non-blocking)
	// rather than leak a goroutine blocked on the channel forever.
	done := make(chan struct{})
	go func() {
		hub.Register(client)
		hub.Unregister(client)
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("Register/Unregister blocked after Stop (goroutine leak)")
	}

	// BroadcastToConversation after Stop must also be panic-safe.
	assert.NotPanics(t, func() {
		hub.BroadcastToConversation("conv-x", &WebSocketMessage{Type: MessageTypeMessage})
	})
}

func TestClient_ReadPumpUnregister_AfterStop_NoLeak(t *testing.T) {
	hub := NewHub()
	go hub.Run()
	time.Sleep(10 * time.Millisecond)

	client := &Client{
		hub:       hub,
		SessionID: "session-pump",
		send:      make(chan *WebSocketMessage, 1),
		Metadata:  make(map[string]string),
	}

	hub.Stop()
	time.Sleep(10 * time.Millisecond)

	// The deferred cleanup in ReadPump calls Unregister; ensure it does not
	// block once the hub has stopped.
	done := make(chan struct{})
	go func() {
		hub.Unregister(client)
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("Unregister blocked after Stop (goroutine leak)")
	}
}

func TestNewClient(t *testing.T) {
	hub := NewHub()
	client := NewClient(hub, nil, "session-abc")

	assert.Equal(t, "session-abc", client.SessionID)
	assert.Equal(t, hub, client.hub)
	assert.NotNil(t, client.send)
	assert.NotNil(t, client.Metadata)
	assert.Empty(t, client.ConversationID)
	assert.Empty(t, client.TenantID)
	assert.Empty(t, client.ContactID)
}

func TestWebSocketMessageTypes(t *testing.T) {
	assert.Equal(t, "message", MessageTypeMessage)
	assert.Equal(t, "typing", MessageTypeTyping)
	assert.Equal(t, "read", MessageTypeRead)
	assert.Equal(t, "connect", MessageTypeConnect)
	assert.Equal(t, "error", MessageTypeError)
	assert.Equal(t, "ack", MessageTypeAck)
	assert.Equal(t, "presence", MessageTypePresence)
}
