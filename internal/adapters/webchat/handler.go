package webchat

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/msgfy/linktor/internal/domain/entity"
	"github.com/msgfy/linktor/internal/domain/repository"
	"github.com/msgfy/linktor/internal/infrastructure/nats"
	"github.com/msgfy/linktor/internal/infrastructure/storage"
)

// Handler handles WebChat HTTP endpoints
type Handler struct {
	adapter          *Adapter
	channelRepo      repository.ChannelRepository
	conversationRepo repository.ConversationRepository
	contactRepo      repository.ContactRepository
	producer         nats.Publisher
	// store guarda os anexos enviados pelo widget. nil quando não há storage
	// configurado; o upload responde 503 nesse caso.
	store         storage.Client
	upgrader      websocket.Upgrader
	rateLimiter   *ipRateLimiter
	uploadLimiter *ipRateLimiter
}

// NewHandler creates a new WebChat handler
func NewHandler(
	adapter *Adapter,
	channelRepo repository.ChannelRepository,
	conversationRepo repository.ConversationRepository,
	contactRepo repository.ContactRepository,
	producer nats.Publisher,
	store storage.Client,
) *Handler {
	return &Handler{
		adapter:          adapter,
		channelRepo:      channelRepo,
		conversationRepo: conversationRepo,
		contactRepo:      contactRepo,
		producer:         producer,
		store:            store,
		// Allow bursts of 10 connection attempts per IP, refilling ~1/sec.
		rateLimiter: newIPRateLimiter(1, 10),
		// Uploads são mais caros que conexões: 10 de rajada, repondo 1 a cada 2s.
		uploadLimiter: newIPRateLimiter(0.5, 10),
		upgrader: websocket.Upgrader{
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
			CheckOrigin: func(r *http.Request) bool {
				return isOriginAllowed(r)
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

	// Lightweight per-IP connection rate limiting to blunt abuse/DoS.
	if h.rateLimiter != nil && !h.rateLimiter.Allow(c.ClientIP(), time.Now()) {
		c.JSON(http.StatusTooManyRequests, gin.H{"error": "too many connection attempts"})
		return
	}

	// Opt-in per-channel widget authentication. When a widget_secret is
	// configured on the channel, a valid signed token is REQUIRED before the
	// upgrade. When absent, we preserve the current (dev/back-compat) behavior
	// but warn that the endpoint is unauthenticated.
	if secret := widgetSecret(channel); secret != "" {
		token := c.Query("token")
		if token == "" {
			token = c.GetHeader("X-Widget-Token")
		}
		if err := VerifyWidgetToken(token, channel.ID, secret, time.Now()); err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid or missing widget token"})
			return
		}
	} else {
		log.Printf("[webchat] WARNING: channel %s has no widget_secret configured; "+
			"WebSocket endpoint /ws/%s is UNAUTHENTICATED", channel.ID, channel.ID)
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

	// Set disconnect handler. The request ctx is already cancelled when the
	// connection drops, so use a fresh bounded ctx for the cleanup work.
	client.SetDisconnectHandler(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		h.adapter.HandleClientDisconnect(ctx, sessionID)
	})

	// Register client (non-blocking against a stopped hub)
	hub.Register(client)

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
	h.adapter.HandleClientConnect(c.Request.Context(), sessionID, client.Metadata)

	// Start pumps
	go client.WritePump()
	client.ReadPump()
}

// handleClientMessage processes messages from WebSocket clients
func (h *Handler) handleClientMessage(ctx context.Context, client *Client, channel *entity.Channel, msg *MessagePayload) error {
	if h.producer == nil {
		return fmt.Errorf("message queue unavailable")
	}

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
	if h.adapter != nil {
		if hub := h.adapter.GetHub(); hub != nil {
			hub.BindConversation(client, conversation.ID)
		}
	}

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
		// sender_id é o identificador que o ReceiveMessageUseCase usa para achar o contato; sem
		// ele o use case cai no ExternalID, que é o id da mensagem, e cada mensagem vira um
		// contato e uma conversa novos. É também o RecipientID da resposta, e o adapter
		// entrega pela sessão do WebSocket.
		Metadata: map[string]string{
			"sender_id":   client.SessionID,
			"session_id":  client.SessionID,
			"sender_name": client.Metadata["name"],
		},
		Attachments: attachments,
		Timestamp:   time.Now(),
	}

	return h.producer.PublishInbound(ctx, inbound)
}

func isOriginAllowed(r *http.Request) bool {
	origin := r.Header.Get("Origin")

	// An empty Origin (non-browser / native clients or same-origin without the
	// header) is NOT trusted by default. Opt in explicitly for dev/native use.
	if origin == "" {
		return boolEnv("LINKTOR_WS_ALLOW_EMPTY_ORIGIN")
	}

	// Explicit allowlist (existing config) always wins.
	for _, allowed := range strings.Split(os.Getenv("LINKTOR_WS_ALLOWED_ORIGINS"), ",") {
		if a := strings.TrimSpace(allowed); a != "" && a == origin {
			return true
		}
	}

	// In enforced mode only the allowlist is accepted; unknown origins (incl.
	// localhost) are rejected. This is the intended homologation posture.
	if boolEnv("LINKTOR_WS_ENFORCE_ORIGIN") {
		return false
	}

	// Dev back-compat: permit loopback origins when not enforcing.
	return strings.HasPrefix(origin, "http://localhost:") ||
		strings.HasPrefix(origin, "http://127.0.0.1:") ||
		strings.HasPrefix(origin, "https://localhost:") ||
		strings.HasPrefix(origin, "https://127.0.0.1:")
}

// boolEnv reports whether an env var is set to a truthy value.
func boolEnv(key string) bool {
	switch strings.ToLower(strings.TrimSpace(os.Getenv(key))) {
	case "1", "true", "yes", "on":
		return true
	default:
		return false
	}
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
				TenantID:    tenantID,
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
		TenantID:    tenantID,
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

	// Create new conversation, denormalizing the channel environment at birth
	// (INV-018). Best-effort lookup: an unmarked row reads as production, which
	// only matters if webchat ever gains sandbox semantics.
	environment := entity.ChannelEnvironmentProduction
	if ch, chErr := h.channelRepo.FindByID(ctx, channelID); chErr == nil {
		environment = ch.Environment
	}
	now := time.Now()
	conversation = &entity.Conversation{
		ID:          uuid.New().String(),
		TenantID:    tenantID,
		ChannelID:   channelID,
		ContactID:   contactID,
		Environment: environment,
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

	// Render this channel's own settings, falling back to defaults for any
	// unset key. Previously this returned the adapter's global config, so every
	// channel showed identical widget branding.
	config := ParseConfig(channel.Config)

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

// maxUploadBytes limita o anexo enviado pelo widget. É menor que o limite do
// atendente (25 MB) porque o endpoint é público.
const maxUploadBytes = 10 << 20

// uploadExtByMime é a allowlist de tipos aceitos do widget e a extensão gravada
// para cada um. A extensão vem daqui, nunca do nome enviado pelo cliente, para
// que um .html disfarçado nunca seja servido como tal.
var uploadExtByMime = map[string]string{
	"image/jpeg":  ".jpg",
	"image/png":   ".png",
	"image/gif":   ".gif",
	"image/webp":  ".webp",
	"audio/webm":  ".webm",
	"audio/ogg":   ".ogg",
	"audio/mpeg":  ".mp3",
	"audio/mp4":   ".m4a",
	"audio/x-m4a": ".m4a",
	"audio/wav":   ".wav",
	"audio/x-wav": ".wav",
}

// UploadResponse espelha handlers.UploadAttachmentResponse: os mesmos campos
// que o widget depois manda em attachments[] da mensagem.
type UploadResponse struct {
	URL       string `json:"url"`
	Type      string `json:"type"`
	Filename  string `json:"filename"`
	MimeType  string `json:"mime_type"`
	SizeBytes int64  `json:"size_bytes"`
}

// UploadMedia recebe um anexo do widget (multipart, campo "file"), guarda no
// media store e devolve a URL para o widget referenciar na mensagem. É público
// como o WebSocket: as mesmas checagens de canal e de widget token se aplicam,
// mais a flag allow_attachments do canal e um rate limit próprio por IP.
func (h *Handler) UploadMedia(c *gin.Context) {
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
	if !channel.IsActive() {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "channel not active"})
		return
	}
	if !ParseConfig(channel.Config).AllowAttachments {
		c.JSON(http.StatusForbidden, gin.H{"error": "attachments are disabled for this channel"})
		return
	}
	if secret := widgetSecret(channel); secret != "" {
		token := c.Query("token")
		if token == "" {
			token = c.GetHeader("X-Widget-Token")
		}
		if err := VerifyWidgetToken(token, channel.ID, secret, time.Now()); err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid or missing widget token"})
			return
		}
	}
	if h.uploadLimiter != nil && !h.uploadLimiter.Allow(c.ClientIP(), time.Now()) {
		c.JSON(http.StatusTooManyRequests, gin.H{"error": "too many uploads"})
		return
	}
	if h.store == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "attachment storage is not configured"})
		return
	}

	fileHeader, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "file is required"})
		return
	}
	if fileHeader.Size > maxUploadBytes {
		c.JSON(http.StatusRequestEntityTooLarge, gin.H{"error": "file exceeds the 10 MB limit"})
		return
	}

	// O MIME declarado decide o tipo; só os da allowlist passam.
	mimeType := strings.ToLower(strings.TrimSpace(strings.Split(fileHeader.Header.Get("Content-Type"), ";")[0]))
	ext, ok := uploadExtByMime[mimeType]
	if !ok {
		c.JSON(http.StatusUnsupportedMediaType, gin.H{"error": "unsupported file type: only images and audio are accepted"})
		return
	}

	file, err := fileHeader.Open()
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "could not read the uploaded file"})
		return
	}
	defer file.Close()

	data, err := io.ReadAll(io.LimitReader(file, maxUploadBytes+1))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "could not read the uploaded file"})
		return
	}
	if int64(len(data)) > maxUploadBytes {
		c.JSON(http.StatusRequestEntityTooLarge, gin.H{"error": "file exceeds the 10 MB limit"})
		return
	}

	key := fmt.Sprintf("webchat/%s/%s%s", channel.ID, uuid.New().String(), ext)
	url, err := h.store.Upload(c.Request.Context(), key, data, mimeType)
	if err != nil {
		log.Printf("[webchat] upload: falha ao gravar %s: %v", key, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to store attachment"})
		return
	}

	c.JSON(http.StatusOK, UploadResponse{
		URL:       url,
		Type:      attachmentTypeFromMime(mimeType),
		Filename:  fileHeader.Filename,
		MimeType:  mimeType,
		SizeBytes: int64(len(data)),
	})
}

// attachmentTypeFromMime reduz o MIME ao tipo grosso usado em attachments[]
// (mesma regra do upload do atendente em handlers.attachmentTypeFromMime).
func attachmentTypeFromMime(mimeType string) string {
	switch {
	case strings.HasPrefix(mimeType, "image/"):
		return "image"
	case strings.HasPrefix(mimeType, "video/"):
		return "video"
	case strings.HasPrefix(mimeType, "audio/"):
		return "audio"
	default:
		return "document"
	}
}
