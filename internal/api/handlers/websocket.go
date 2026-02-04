package handlers

import (
	"encoding/json"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/gorilla/websocket"
)

const (
	writeWait      = 10 * time.Second
	pongWait       = 60 * time.Second
	pingPeriod     = (pongWait * 9) / 10
	maxMessageSize = 512 * 1024
)

// WebSocket event types
const (
	WSEventNewMessage          = "new_message"
	WSEventMessageUpdated      = "message_updated"
	WSEventConversationUpdated = "conversation_updated"
	WSEventConversationCreated = "conversation_created"
	WSEventTyping              = "typing"
	WSEventPresence            = "presence"
	WSEventError               = "error"
	WSEventConnected           = "connected"
)

// WSMessage represents a WebSocket message
type WSMessage struct {
	Type    string      `json:"type"`
	Payload interface{} `json:"payload"`
}

// WSNewMessagePayload represents a new message event
type WSNewMessagePayload struct {
	ConversationID string      `json:"conversation_id"`
	Message        interface{} `json:"message"`
}

// WSTypingPayload represents a typing event
type WSTypingPayload struct {
	ConversationID string `json:"conversation_id"`
	UserID         string `json:"user_id"`
	UserName       string `json:"user_name"`
	IsTyping       bool   `json:"is_typing"`
}

// WSPresencePayload represents a presence event
type WSPresencePayload struct {
	UserID   string `json:"user_id"`
	Status   string `json:"status"` // online, offline, away
	LastSeen string `json:"last_seen,omitempty"`
}

// AgentHub manages WebSocket connections for agents
type AgentHub struct {
	// Registered clients by user ID
	clients map[string]*AgentClient

	// Clients by tenant ID for broadcasting
	tenants map[string]map[string]*AgentClient

	// Register channel
	register chan *AgentClient

	// Unregister channel
	unregister chan *AgentClient

	// Broadcast to tenant
	broadcast chan *TenantBroadcast

	mu   sync.RWMutex
	done chan struct{}
}

// TenantBroadcast represents a message to broadcast to a tenant
type TenantBroadcast struct {
	TenantID string
	Message  *WSMessage
	ExcludeUserID string // Optional: exclude this user from broadcast
}

// AgentClient represents a connected agent
type AgentClient struct {
	hub      *AgentHub
	conn     *websocket.Conn
	UserID   string
	TenantID string
	Email    string
	send     chan *WSMessage
}

// NewAgentHub creates a new agent hub
func NewAgentHub() *AgentHub {
	return &AgentHub{
		clients:    make(map[string]*AgentClient),
		tenants:    make(map[string]map[string]*AgentClient),
		register:   make(chan *AgentClient),
		unregister: make(chan *AgentClient),
		broadcast:  make(chan *TenantBroadcast, 256),
		done:       make(chan struct{}),
	}
}

// Run starts the hub
func (h *AgentHub) Run() {
	for {
		select {
		case client := <-h.register:
			h.mu.Lock()
			h.clients[client.UserID] = client
			if h.tenants[client.TenantID] == nil {
				h.tenants[client.TenantID] = make(map[string]*AgentClient)
			}
			h.tenants[client.TenantID][client.UserID] = client
			h.mu.Unlock()

			// Broadcast presence
			h.BroadcastToTenant(client.TenantID, &WSMessage{
				Type: WSEventPresence,
				Payload: WSPresencePayload{
					UserID: client.UserID,
					Status: "online",
				},
			}, client.UserID)

		case client := <-h.unregister:
			h.mu.Lock()
			if _, ok := h.clients[client.UserID]; ok {
				delete(h.clients, client.UserID)
				if tenantClients, ok := h.tenants[client.TenantID]; ok {
					delete(tenantClients, client.UserID)
					if len(tenantClients) == 0 {
						delete(h.tenants, client.TenantID)
					}
				}
				close(client.send)
			}
			h.mu.Unlock()

			// Broadcast offline presence
			h.BroadcastToTenant(client.TenantID, &WSMessage{
				Type: WSEventPresence,
				Payload: WSPresencePayload{
					UserID:   client.UserID,
					Status:   "offline",
					LastSeen: time.Now().Format(time.RFC3339),
				},
			}, "")

		case broadcast := <-h.broadcast:
			h.mu.RLock()
			if tenantClients, ok := h.tenants[broadcast.TenantID]; ok {
				for userID, client := range tenantClients {
					if userID == broadcast.ExcludeUserID {
						continue
					}
					select {
					case client.send <- broadcast.Message:
					default:
						// Buffer full
					}
				}
			}
			h.mu.RUnlock()

		case <-h.done:
			h.mu.Lock()
			for _, client := range h.clients {
				close(client.send)
			}
			h.clients = make(map[string]*AgentClient)
			h.tenants = make(map[string]map[string]*AgentClient)
			h.mu.Unlock()
			return
		}
	}
}

// Stop stops the hub
func (h *AgentHub) Stop() {
	close(h.done)
}

// BroadcastToTenant sends a message to all agents in a tenant
func (h *AgentHub) BroadcastToTenant(tenantID string, msg *WSMessage, excludeUserID string) {
	select {
	case h.broadcast <- &TenantBroadcast{
		TenantID:      tenantID,
		Message:       msg,
		ExcludeUserID: excludeUserID,
	}:
	default:
		// Channel full
	}
}

// SendToUser sends a message to a specific user
func (h *AgentHub) SendToUser(userID string, msg *WSMessage) {
	h.mu.RLock()
	client, ok := h.clients[userID]
	h.mu.RUnlock()

	if ok {
		select {
		case client.send <- msg:
		default:
		}
	}
}

// GetOnlineUsers returns online users for a tenant
func (h *AgentHub) GetOnlineUsers(tenantID string) []string {
	h.mu.RLock()
	defer h.mu.RUnlock()

	var users []string
	if tenantClients, ok := h.tenants[tenantID]; ok {
		for userID := range tenantClients {
			users = append(users, userID)
		}
	}
	return users
}

// WebSocketHandler handles agent WebSocket connections
type WebSocketHandler struct {
	hub       *AgentHub
	jwtSecret string
	upgrader  websocket.Upgrader
}

// NewWebSocketHandler creates a new WebSocket handler
func NewWebSocketHandler(hub *AgentHub, jwtSecret string) *WebSocketHandler {
	return &WebSocketHandler{
		hub:       hub,
		jwtSecret: jwtSecret,
		upgrader: websocket.Upgrader{
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
			CheckOrigin: func(r *http.Request) bool {
				return true // In production, validate origin
			},
		},
	}
}

// HandleConnection handles WebSocket upgrade and connection
func (h *WebSocketHandler) HandleConnection(c *gin.Context) {
	// Get token from query param
	token := c.Query("token")
	if token == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "missing token"})
		return
	}

	// Parse and validate JWT
	claims := jwt.MapClaims{}
	_, err := jwt.ParseWithClaims(token, claims, func(t *jwt.Token) (interface{}, error) {
		return []byte(h.jwtSecret), nil
	})
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
		return
	}

	userID, _ := claims["user_id"].(string)
	tenantID, _ := claims["tenant_id"].(string)
	email, _ := claims["email"].(string)

	if userID == "" || tenantID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid token claims"})
		return
	}

	// Upgrade connection
	conn, err := h.upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		return
	}

	client := &AgentClient{
		hub:      h.hub,
		conn:     conn,
		UserID:   userID,
		TenantID: tenantID,
		Email:    email,
		send:     make(chan *WSMessage, 256),
	}

	// Register client
	h.hub.register <- client

	// Send connected message
	client.send <- &WSMessage{
		Type: WSEventConnected,
		Payload: map[string]interface{}{
			"user_id":      userID,
			"tenant_id":    tenantID,
			"online_users": h.hub.GetOnlineUsers(tenantID),
		},
	}

	// Start pumps
	go client.writePump()
	go client.readPump()
}

// readPump reads messages from the WebSocket connection
func (c *AgentClient) readPump() {
	defer func() {
		c.hub.unregister <- c
		c.conn.Close()
	}()

	c.conn.SetReadLimit(maxMessageSize)
	c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetPongHandler(func(string) error {
		c.conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})

	for {
		_, data, err := c.conn.ReadMessage()
		if err != nil {
			break
		}

		var msg WSMessage
		if err := json.Unmarshal(data, &msg); err != nil {
			continue
		}

		// Handle client messages
		switch msg.Type {
		case WSEventTyping:
			// Broadcast typing indicator
			if payload, ok := msg.Payload.(map[string]interface{}); ok {
				convID, _ := payload["conversation_id"].(string)
				isTyping, _ := payload["is_typing"].(bool)
				c.hub.BroadcastToTenant(c.TenantID, &WSMessage{
					Type: WSEventTyping,
					Payload: WSTypingPayload{
						ConversationID: convID,
						UserID:         c.UserID,
						IsTyping:       isTyping,
					},
				}, c.UserID)
			}
		}
	}
}

// writePump writes messages to the WebSocket connection
func (c *AgentClient) writePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()

	for {
		select {
		case msg, ok := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			data, err := json.Marshal(msg)
			if err != nil {
				continue
			}

			if err := c.conn.WriteMessage(websocket.TextMessage, data); err != nil {
				return
			}

		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

// Global hub instance
var globalAgentHub *AgentHub
var hubOnce sync.Once

// GetAgentHub returns the global agent hub instance
func GetAgentHub() *AgentHub {
	hubOnce.Do(func() {
		globalAgentHub = NewAgentHub()
		go globalAgentHub.Run()
	})
	return globalAgentHub
}

// BroadcastNewMessage broadcasts a new message to all agents in a tenant
func BroadcastNewMessage(tenantID, conversationID string, message interface{}) {
	hub := GetAgentHub()
	hub.BroadcastToTenant(tenantID, &WSMessage{
		Type: WSEventNewMessage,
		Payload: WSNewMessagePayload{
			ConversationID: conversationID,
			Message:        message,
		},
	}, "")
}

// BroadcastConversationUpdate broadcasts a conversation update
func BroadcastConversationUpdate(tenantID string, conversation interface{}) {
	hub := GetAgentHub()
	hub.BroadcastToTenant(tenantID, &WSMessage{
		Type:    WSEventConversationUpdated,
		Payload: conversation,
	}, "")
}
