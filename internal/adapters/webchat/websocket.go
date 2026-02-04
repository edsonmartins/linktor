package webchat

import (
	"encoding/json"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

const (
	// Time allowed to write a message to the peer
	writeWait = 10 * time.Second

	// Time allowed to read the next pong message from the peer
	pongWait = 60 * time.Second

	// Send pings to peer with this period (must be less than pongWait)
	pingPeriod = (pongWait * 9) / 10

	// Maximum message size allowed from peer
	maxMessageSize = 512 * 1024 // 512KB
)

// WebSocket message types
const (
	MessageTypeMessage  = "message"
	MessageTypeTyping   = "typing"
	MessageTypeRead     = "read"
	MessageTypeConnect  = "connect"
	MessageTypeError    = "error"
	MessageTypeAck      = "ack"
	MessageTypePresence = "presence"
)

// WebSocketMessage represents a WebSocket protocol message
type WebSocketMessage struct {
	Type    string         `json:"type"`
	Payload MessagePayload `json:"payload"`
}

// MessagePayload represents the message payload
type MessagePayload struct {
	ID          string              `json:"id,omitempty"`
	ContentType string              `json:"content_type,omitempty"`
	Content     string              `json:"content,omitempty"`
	SenderType  string              `json:"sender_type,omitempty"`
	SenderID    string              `json:"sender_id,omitempty"`
	SenderName  string              `json:"sender_name,omitempty"`
	Attachments []AttachmentPayload `json:"attachments,omitempty"`
	Metadata    map[string]string   `json:"metadata,omitempty"`
	Timestamp   string              `json:"timestamp,omitempty"`
	IsTyping    bool                `json:"is_typing,omitempty"`
	Error       string              `json:"error,omitempty"`
}

// AttachmentPayload represents an attachment in a message
type AttachmentPayload struct {
	Type      string `json:"type"`
	URL       string `json:"url"`
	Filename  string `json:"filename,omitempty"`
	MimeType  string `json:"mime_type,omitempty"`
	SizeBytes int64  `json:"size_bytes,omitempty"`
}

// Hub maintains the set of active clients and broadcasts messages
type Hub struct {
	// Registered clients
	clients map[string]*Client

	// Client sessions by conversation ID
	conversations map[string]map[string]*Client

	// Register requests from clients
	register chan *Client

	// Unregister requests from clients
	unregister chan *Client

	// Inbound messages from clients
	broadcast chan *BroadcastMessage

	// Mutex for thread safety
	mu sync.RWMutex

	// Done channel to stop the hub
	done chan struct{}
}

// BroadcastMessage represents a message to broadcast
type BroadcastMessage struct {
	SessionID string
	Message   *WebSocketMessage
}

// NewHub creates a new Hub
func NewHub() *Hub {
	return &Hub{
		clients:       make(map[string]*Client),
		conversations: make(map[string]map[string]*Client),
		register:      make(chan *Client),
		unregister:    make(chan *Client),
		broadcast:     make(chan *BroadcastMessage),
		done:          make(chan struct{}),
	}
}

// Run starts the hub's main loop
func (h *Hub) Run() {
	for {
		select {
		case client := <-h.register:
			h.mu.Lock()
			h.clients[client.SessionID] = client
			if client.ConversationID != "" {
				if h.conversations[client.ConversationID] == nil {
					h.conversations[client.ConversationID] = make(map[string]*Client)
				}
				h.conversations[client.ConversationID][client.SessionID] = client
			}
			h.mu.Unlock()

		case client := <-h.unregister:
			h.mu.Lock()
			if _, ok := h.clients[client.SessionID]; ok {
				delete(h.clients, client.SessionID)
				if client.ConversationID != "" {
					if convClients, ok := h.conversations[client.ConversationID]; ok {
						delete(convClients, client.SessionID)
						if len(convClients) == 0 {
							delete(h.conversations, client.ConversationID)
						}
					}
				}
				close(client.send)
			}
			h.mu.Unlock()

		case msg := <-h.broadcast:
			h.mu.RLock()
			if client, ok := h.clients[msg.SessionID]; ok {
				select {
				case client.send <- msg.Message:
				default:
					// Client buffer full, skip
				}
			}
			h.mu.RUnlock()

		case <-h.done:
			h.mu.Lock()
			for _, client := range h.clients {
				close(client.send)
			}
			h.clients = make(map[string]*Client)
			h.conversations = make(map[string]map[string]*Client)
			h.mu.Unlock()
			return
		}
	}
}

// Stop stops the hub
func (h *Hub) Stop() {
	close(h.done)
}

// GetClient returns a client by session ID
func (h *Hub) GetClient(sessionID string) *Client {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.clients[sessionID]
}

// GetConversationClients returns all clients for a conversation
func (h *Hub) GetConversationClients(conversationID string) []*Client {
	h.mu.RLock()
	defer h.mu.RUnlock()

	clients := h.conversations[conversationID]
	result := make([]*Client, 0, len(clients))
	for _, client := range clients {
		result = append(result, client)
	}
	return result
}

// BroadcastToConversation sends a message to all clients in a conversation
func (h *Hub) BroadcastToConversation(conversationID string, msg *WebSocketMessage) {
	h.mu.RLock()
	clients := h.conversations[conversationID]
	h.mu.RUnlock()

	for _, client := range clients {
		select {
		case client.send <- msg:
		default:
			// Client buffer full
		}
	}
}

// ClientCount returns the number of connected clients
func (h *Hub) ClientCount() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.clients)
}

// Client represents a WebSocket client
type Client struct {
	// The WebSocket hub
	hub *Hub

	// The WebSocket connection
	conn *websocket.Conn

	// Unique session ID for this client
	SessionID string

	// Conversation ID (if associated with a conversation)
	ConversationID string

	// Tenant ID
	TenantID string

	// Channel ID
	ChannelID string

	// Contact ID (if identified)
	ContactID string

	// Buffered channel of outbound messages
	send chan *WebSocketMessage

	// Message handler callback
	onMessage func(msg *MessagePayload) error

	// Disconnect handler callback
	onDisconnect func()

	// Metadata
	Metadata map[string]string
}

// NewClient creates a new WebSocket client
func NewClient(hub *Hub, conn *websocket.Conn, sessionID string) *Client {
	return &Client{
		hub:       hub,
		conn:      conn,
		SessionID: sessionID,
		send:      make(chan *WebSocketMessage, 256),
		Metadata:  make(map[string]string),
	}
}

// SetMessageHandler sets the message handler callback
func (c *Client) SetMessageHandler(handler func(msg *MessagePayload) error) {
	c.onMessage = handler
}

// SetDisconnectHandler sets the disconnect handler callback
func (c *Client) SetDisconnectHandler(handler func()) {
	c.onDisconnect = handler
}

// SendMessage sends a message to the client
func (c *Client) SendMessage(msg *WebSocketMessage) error {
	select {
	case c.send <- msg:
		return nil
	default:
		return nil // Buffer full, message dropped
	}
}

// ReadPump pumps messages from the WebSocket connection to the hub
func (c *Client) ReadPump() {
	defer func() {
		c.hub.unregister <- c
		c.conn.Close()
		if c.onDisconnect != nil {
			c.onDisconnect()
		}
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
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				// Log error
			}
			break
		}

		var msg WebSocketMessage
		if err := json.Unmarshal(data, &msg); err != nil {
			c.sendError("invalid message format")
			continue
		}

		switch msg.Type {
		case MessageTypeMessage:
			if c.onMessage != nil {
				if err := c.onMessage(&msg.Payload); err != nil {
					c.sendError(err.Error())
				} else {
					// Send acknowledgment
					c.SendMessage(&WebSocketMessage{
						Type: MessageTypeAck,
						Payload: MessagePayload{
							ID: msg.Payload.ID,
						},
					})
				}
			}
		case MessageTypeTyping:
			// Could forward typing indicator to agents
		case MessageTypeRead:
			// Could update read receipts
		}
	}
}

// WritePump pumps messages from the hub to the WebSocket connection
func (c *Client) WritePump() {
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
				// Hub closed the channel
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

func (c *Client) sendError(message string) {
	c.SendMessage(&WebSocketMessage{
		Type: MessageTypeError,
		Payload: MessagePayload{
			Error: message,
		},
	})
}
