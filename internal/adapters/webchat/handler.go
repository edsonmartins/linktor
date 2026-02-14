package webchat

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/msgfy/linktor/internal/domain/entity"
	"github.com/msgfy/linktor/internal/domain/repository"
	"github.com/msgfy/linktor/internal/infrastructure/nats"
)

// Handler handles WebChat HTTP endpoints
type Handler struct {
	adapter         *Adapter
	channelRepo     repository.ChannelRepository
	conversationRepo repository.ConversationRepository
	contactRepo     repository.ContactRepository
	producer        *nats.Producer
	upgrader        websocket.Upgrader
}

// NewHandler creates a new WebChat handler
func NewHandler(
	adapter *Adapter,
	channelRepo repository.ChannelRepository,
	conversationRepo repository.ConversationRepository,
	contactRepo repository.ContactRepository,
	producer *nats.Producer,
) *Handler {
	return &Handler{
		adapter:          adapter,
		channelRepo:      channelRepo,
		conversationRepo: conversationRepo,
		contactRepo:      contactRepo,
		producer:         producer,
		upgrader: websocket.Upgrader{
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
			CheckOrigin: func(r *http.Request) bool {
				// In production, validate origin against allowed domains
				return true
			},
		},
	}
}

// WebSocketHandler handles WebSocket upgrade requests
func (h *Handler) WebSocketHandler(c *gin.Context) {
	channelID := c.Param("channelId")

	// Verify channel exists and is active
	channel, err := h.channelRepo.FindByID(c.Request.Context(), channelID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "channel not found"})
		return
	}

	if channel.Type != entity.ChannelTypeWebChat {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid channel type"})
		return
	}

	if !channel.IsActive() {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "channel not active"})
		return
	}

	// Upgrade connection
	conn, err := h.upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		return
	}

	// Get or create session ID
	sessionID := c.Query("session_id")
	if sessionID == "" {
		sessionID = uuid.New().String()
	}

	// Create client
	hub := h.adapter.GetHub()
	if hub == nil {
		conn.Close()
		return
	}

	client := NewClient(hub, conn, sessionID)
	client.TenantID = channel.TenantID
	client.ChannelID = channel.ID

	// Extract visitor info from query params
	if name := c.Query("name"); name != "" {
		client.Metadata["name"] = name
	}
	if email := c.Query("email"); email != "" {
		client.Metadata["email"] = email
	}
	if phone := c.Query("phone"); phone != "" {
		client.Metadata["phone"] = phone
	}

	// Set message handler
	client.SetMessageHandler(func(msg *MessagePayload) error {
		return h.handleClientMessage(c.Request.Context(), client, channel, msg)
	})

	// Set disconnect handler
	client.SetDisconnectHandler(func() {
		h.adapter.HandleClientDisconnect(context.Background(), sessionID)
	})

	// Register client
	hub.register <- client

	// Send connection confirmation
	client.SendMessage(&WebSocketMessage{
		Type: MessageTypeConnect,
		Payload: MessagePayload{
			ID: sessionID,
			Metadata: map[string]string{
				"session_id":   sessionID,
				"widget_title": h.adapter.GetConfig().WidgetTitle,
				"widget_color": h.adapter.GetConfig().WidgetColor,
			},
			Timestamp: time.Now().Format(time.RFC3339),
		},
	})

	// Handle connect event
	h.adapter.HandleClientConnect(context.Background(), sessionID, client.Metadata)

	// Start pumps
	go client.WritePump()
	client.ReadPump()
}

// handleClientMessage processes messages from WebSocket clients
func (h *Handler) handleClientMessage(ctx context.Context, client *Client, channel *entity.Channel, msg *MessagePayload) error {
	// Get or create contact
	contact, err := h.getOrCreateContact(ctx, channel.TenantID, client)
	if err != nil {
		return err
	}
	client.ContactID = contact.ID

	// Get or create conversation
	conversation, err := h.getOrCreateConversation(ctx, channel.TenantID, channel.ID, contact.ID, client)
	if err != nil {
		return err
	}
	client.ConversationID = conversation.ID

	// Publish to NATS
	attachments := make([]nats.AttachmentData, 0, len(msg.Attachments))
	for _, att := range msg.Attachments {
		attachments = append(attachments, nats.AttachmentData{
			Type:      att.Type,
			URL:       att.URL,
			Filename:  att.Filename,
			MimeType:  att.MimeType,
			SizeBytes: att.SizeBytes,
		})
	}

	inbound := &nats.InboundMessage{
		ID:             uuid.New().String(),
		TenantID:       channel.TenantID,
		ChannelID:      channel.ID,
		ChannelType:    "webchat",
		ContactID:      contact.ID,
		ConversationID: conversation.ID,
		ExternalID:     msg.ID,
		ContentType:    msg.ContentType,
		Content:        msg.Content,
		Metadata: map[string]string{
			"session_id":  client.SessionID,
			"sender_name": client.Metadata["name"],
		},
		Attachments: attachments,
		Timestamp:   time.Now(),
	}

	return h.producer.PublishInbound(ctx, inbound)
}

// getOrCreateContact finds or creates a contact
func (h *Handler) getOrCreateContact(ctx context.Context, tenantID string, client *Client) (*entity.Contact, error) {
	// Try to find by session identity
	contact, err := h.contactRepo.FindByIdentity(ctx, tenantID, "webchat", client.SessionID)
	if err == nil && contact != nil {
		return contact, nil
	}

	// Try to find by email if provided
	if email, ok := client.Metadata["email"]; ok && email != "" {
		contact, err = h.contactRepo.FindByEmail(ctx, tenantID, email)
		if err == nil && contact != nil {
			// Add webchat identity
			identity := &entity.ContactIdentity{
				ID:          uuid.New().String(),
				ContactID:   contact.ID,
				ChannelType: "webchat",
				Identifier:  client.SessionID,
				Metadata:    client.Metadata,
				CreatedAt:   time.Now(),
			}
			h.contactRepo.AddIdentity(ctx, identity)
			return contact, nil
		}
	}

	// Create new contact
	now := time.Now()
	name := client.Metadata["name"]
	if name == "" {
		name = "Visitor"
	}

	contact = &entity.Contact{
		ID:           uuid.New().String(),
		TenantID:     tenantID,
		Name:         name,
		Email:        client.Metadata["email"],
		Phone:        client.Metadata["phone"],
		CustomFields: make(map[string]string),
		Tags:         []string{"webchat"},
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	if err := h.contactRepo.Create(ctx, contact); err != nil {
		return nil, err
	}

	// Add identity
	identity := &entity.ContactIdentity{
		ID:          uuid.New().String(),
		ContactID:   contact.ID,
		ChannelType: "webchat",
		Identifier:  client.SessionID,
		Metadata:    client.Metadata,
		CreatedAt:   now,
	}
	h.contactRepo.AddIdentity(ctx, identity)

	return contact, nil
}

// getOrCreateConversation finds or creates a conversation
func (h *Handler) getOrCreateConversation(ctx context.Context, tenantID, channelID, contactID string, client *Client) (*entity.Conversation, error) {
	// Try to find open conversation
	conversation, err := h.conversationRepo.FindOpenByContactAndChannel(ctx, contactID, channelID)
	if err == nil && conversation != nil {
		return conversation, nil
	}

	// Create new conversation
	now := time.Now()
	conversation = &entity.Conversation{
		ID:          uuid.New().String(),
		TenantID:    tenantID,
		ChannelID:   channelID,
		ContactID:   contactID,
		Status:      entity.ConversationStatusOpen,
		Priority:    entity.ConversationPriorityNormal,
		UnreadCount: 0,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	if err := h.conversationRepo.Create(ctx, conversation); err != nil {
		return nil, err
	}

	return conversation, nil
}

// GetWidgetConfig returns the widget configuration
func (h *Handler) GetWidgetConfig(c *gin.Context) {
	channelID := c.Param("channelId")

	channel, err := h.channelRepo.FindByID(c.Request.Context(), channelID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "channel not found"})
		return
	}

	if channel.Type != entity.ChannelTypeWebChat {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid channel type"})
		return
	}

	config := h.adapter.GetConfig()

	c.JSON(http.StatusOK, gin.H{
		"channel_id":        channel.ID,
		"widget_title":      config.WidgetTitle,
		"widget_color":      config.WidgetColor,
		"welcome_message":   config.WelcomeMessage,
		"offline_message":   config.OfflineMessage,
		"avatar_url":        config.AvatarURL,
		"allow_attachments": config.AllowAttachments,
		"require_email":     config.RequireEmail,
		"require_name":      config.RequireName,
		"enabled":           channel.Enabled,
		"connection_status": channel.ConnectionStatus,
	})
}

// UploadMedia handles media uploads from the widget
func (h *Handler) UploadMedia(c *gin.Context) {
	// TODO: Implement media upload handling
	// This would save the file to storage (MinIO/S3) and return the URL
	c.JSON(http.StatusNotImplemented, gin.H{"error": "not implemented"})
}
