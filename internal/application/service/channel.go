package service

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/msgfy/linktor/internal/adapters/whatsapp"
	"github.com/msgfy/linktor/internal/domain/entity"
	"github.com/msgfy/linktor/internal/domain/repository"
	"github.com/msgfy/linktor/internal/infrastructure/nats"
	"github.com/msgfy/linktor/pkg/errors"
	"github.com/msgfy/linktor/pkg/logger"
	"github.com/msgfy/linktor/pkg/plugin"
	"go.uber.org/zap"
)

// DefaultListParams returns default pagination parameters
func DefaultListParams() *repository.ListParams {
	return &repository.ListParams{
		Page:     1,
		PageSize: 100,
		SortBy:   "created_at",
		SortDir:  "desc",
	}
}

// CreateChannelInput represents input for creating a channel
type CreateChannelInput struct {
	TenantID    string
	Type        string
	Name        string
	Identifier  string
	Config      map[string]string
	Credentials map[string]string
}

// UpdateChannelInput represents input for updating a channel
type UpdateChannelInput struct {
	Name        *string
	Identifier  *string
	Config      map[string]string
	Credentials map[string]string
}

// ConnectResult represents the result of connecting a channel
type ConnectResult struct {
	Channel   *entity.Channel `json:"channel"`
	QRCode    string          `json:"qr_code,omitempty"`
	ExpiresIn int             `json:"expires_in,omitempty"`
	PairCode  string          `json:"pair_code,omitempty"`
}

// ChannelService handles channel operations
type ChannelService struct {
	repo     repository.ChannelRepository
	registry *plugin.Registry
	producer *nats.Producer
}

// NewChannelService creates a new channel service
func NewChannelService(repo repository.ChannelRepository, registry *plugin.Registry, producer *nats.Producer) *ChannelService {
	return &ChannelService{
		repo:     repo,
		registry: registry,
		producer: producer,
	}
}

// List returns all channels for a tenant
func (s *ChannelService) List(ctx context.Context, tenantID string) ([]*entity.Channel, error) {
	if s.repo == nil {
		return nil, errors.New(errors.ErrCodeInternal, "channel repository not initialized")
	}
	// Use default pagination params
	params := repository.NewListParams()
	params.Page = 1
	params.PageSize = 100
	params.SortBy = "created_at"
	params.SortDir = "desc"

	channels, _, err := s.repo.FindByTenant(ctx, tenantID, params)
	return channels, err
}

// Create creates a new channel
func (s *ChannelService) Create(ctx context.Context, input *CreateChannelInput) (*entity.Channel, error) {
	now := time.Now()
	channel := &entity.Channel{
		ID:               uuid.New().String(),
		TenantID:         input.TenantID,
		Type:             entity.ChannelType(input.Type),
		Name:             input.Name,
		Identifier:       input.Identifier,
		Enabled:          true, // Enabled by default
		ConnectionStatus: entity.ConnectionStatusDisconnected,
		Config:           input.Config,
		Credentials:      input.Credentials,
		CreatedAt:        now,
		UpdatedAt:        now,
	}

	if err := s.repo.Create(ctx, channel); err != nil {
		return nil, err
	}

	return channel, nil
}

// GetByID returns a channel by ID
func (s *ChannelService) GetByID(ctx context.Context, id string) (*entity.Channel, error) {
	return s.repo.FindByID(ctx, id)
}

// Update updates a channel
func (s *ChannelService) Update(ctx context.Context, id string, input *UpdateChannelInput) (*entity.Channel, error) {
	channel, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if input.Name != nil {
		channel.Name = *input.Name
	}
	if input.Identifier != nil {
		channel.Identifier = *input.Identifier
	}
	if input.Config != nil {
		channel.Config = input.Config
	}
	if input.Credentials != nil {
		channel.Credentials = input.Credentials
	}
	channel.UpdatedAt = time.Now()

	if err := s.repo.Update(ctx, channel); err != nil {
		return nil, err
	}

	return channel, nil
}

// Delete deletes a channel
func (s *ChannelService) Delete(ctx context.Context, id string) error {
	return s.repo.Delete(ctx, id)
}

// UpdateEnabled updates the channel enabled state
func (s *ChannelService) UpdateEnabled(ctx context.Context, id string, enabled bool) (*entity.Channel, error) {
	channel, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if err := s.repo.UpdateEnabled(ctx, id, enabled); err != nil {
		return nil, err
	}

	channel.Enabled = enabled
	channel.UpdatedAt = time.Now()
	return channel, nil
}

// UpdateStatus updates channel status (handles both enabled and connection_status for backwards compatibility)
func (s *ChannelService) UpdateStatus(ctx context.Context, id string, status string) (*entity.Channel, error) {
	channel, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}

	// Map string status to enabled state (for backwards compatibility with frontend)
	switch status {
	case "active":
		if err := s.repo.UpdateEnabled(ctx, id, true); err != nil {
			return nil, err
		}
		channel.Enabled = true
	case "inactive":
		if err := s.repo.UpdateEnabled(ctx, id, false); err != nil {
			return nil, err
		}
		channel.Enabled = false
	default:
		return nil, errors.New(errors.ErrCodeValidation, "invalid status: use 'active' or 'inactive'")
	}

	channel.UpdatedAt = time.Now()
	return channel, nil
}

// Connect connects a channel
func (s *ChannelService) Connect(ctx context.Context, id string) (*ConnectResult, error) {
	logger.Info("Connect called",
		zap.String("channel_id", id))

	channel, err := s.repo.FindByID(ctx, id)
	if err != nil {
		logger.Error("Failed to find channel", zap.String("channel_id", id), zap.Error(err))
		return nil, err
	}

	logger.Info("Channel found",
		zap.String("channel_id", channel.ID),
		zap.String("channel_type", string(channel.Type)),
		zap.String("expected_whatsapp", string(entity.ChannelTypeWhatsApp)),
		zap.String("expected_unofficial", string(entity.ChannelTypeWhatsAppUnofficial)))

	// Handle WhatsApp Unofficial - requires QR code authentication
	if channel.Type == entity.ChannelTypeWhatsApp || channel.Type == entity.ChannelTypeWhatsAppUnofficial {
		logger.Info("Routing to WhatsApp unofficial handler")
		return s.connectWhatsAppUnofficial(ctx, channel)
	}

	logger.Info("Using default connect handler (not WhatsApp)")
	// Default behavior for other channels - mark as connected
	channel.ConnectionStatus = entity.ConnectionStatusConnected
	channel.UpdatedAt = time.Now()

	if err := s.repo.UpdateConnectionStatus(ctx, channel.ID, entity.ConnectionStatusConnected); err != nil {
		return nil, err
	}

	return &ConnectResult{
		Channel: channel,
	}, nil
}

// connectWhatsAppUnofficial handles WhatsApp unofficial connection with QR code
func (s *ChannelService) connectWhatsAppUnofficial(ctx context.Context, channel *entity.Channel) (*ConnectResult, error) {
	logger.Info("Connecting WhatsApp channel",
		zap.String("channel_id", channel.ID),
		zap.String("tenant_id", channel.TenantID))

	// Check if already connected via registry
	if s.registry != nil {
		if existingAdapter, err := s.registry.GetAdapterByChannelID(channel.ID); err == nil {
			waAdapter, ok := existingAdapter.(*whatsapp.Adapter)
			if ok && waAdapter.IsLoggedIn() {
				logger.Info("WhatsApp channel already connected via registry",
					zap.String("channel_id", channel.ID))
				channel.ConnectionStatus = entity.ConnectionStatusConnected
				channel.UpdatedAt = time.Now()
				s.repo.UpdateConnectionStatus(ctx, channel.ID, entity.ConnectionStatusConnected)
				return &ConnectResult{
					Channel: channel,
				}, nil
			}
		}
	}

	// Create new adapter instance
	adapter := whatsapp.NewAdapter()

	// Configure adapter
	config := map[string]string{
		"channel_id":    channel.ID,
		"database_path": fmt.Sprintf("storages/whatsapp_%s.db", channel.ID),
	}

	// Add device name from channel config if available
	if deviceName, ok := channel.Config["device_name"]; ok && deviceName != "" {
		config["device_name"] = deviceName
	}

	// Initialize adapter
	if err := adapter.Initialize(config); err != nil {
		logger.Error("Failed to initialize WhatsApp adapter",
			zap.String("channel_id", channel.ID),
			zap.Error(err))
		return nil, errors.Wrap(err, errors.ErrCodeInternal, "failed to initialize WhatsApp adapter")
	}

	// Use a background context for WhatsApp connection to keep it alive
	// beyond the HTTP request lifecycle
	bgCtx := context.Background()

	// Connect to WhatsApp servers using background context
	if err := adapter.Connect(bgCtx); err != nil {
		logger.Error("Failed to connect WhatsApp adapter",
			zap.String("channel_id", channel.ID),
			zap.Error(err))
		return nil, errors.Wrap(err, errors.ErrCodeInternal, "failed to connect WhatsApp adapter")
	}

	// Set up message handler to publish inbound messages to NATS
	s.setupWhatsAppHandlers(adapter, channel)

	// Check if already logged in (has stored session)
	isLoggedIn := adapter.IsLoggedIn()
	logger.Info("Checking WhatsApp login status",
		zap.String("channel_id", channel.ID),
		zap.Bool("is_logged_in", isLoggedIn))

	if isLoggedIn {
		logger.Info("WhatsApp channel reconnected with stored session",
			zap.String("channel_id", channel.ID))
		channel.ConnectionStatus = entity.ConnectionStatusConnected
		channel.UpdatedAt = time.Now()
		s.repo.UpdateConnectionStatus(ctx, channel.ID, entity.ConnectionStatusConnected)

		// Store adapter in registry for future use
		if s.registry != nil {
			s.registry.RegisterChannelAdapter(channel.ID, adapter)
		}

		return &ConnectResult{
			Channel: channel,
		}, nil
	}

	// Start QR code login process using background context
	// This ensures the connection stays alive after the HTTP request completes
	logger.Info("Starting WhatsApp QR code login",
		zap.String("channel_id", channel.ID))
	qrChan, err := adapter.Login(bgCtx)
	if err != nil {
		logger.Error("Failed to start WhatsApp login",
			zap.String("channel_id", channel.ID),
			zap.Error(err))
		adapter.Disconnect(bgCtx)
		return nil, errors.Wrap(err, errors.ErrCodeInternal, "failed to start WhatsApp login")
	}

	// Wait for first QR code with timeout
	select {
	case qrEvent, ok := <-qrChan:
		if !ok {
			logger.Error("QR code channel closed unexpectedly",
				zap.String("channel_id", channel.ID))
			return nil, errors.New(errors.ErrCodeInternal, "QR code channel closed unexpectedly")
		}

		logger.Info("QR code generated for WhatsApp login",
			zap.String("channel_id", channel.ID),
			zap.Time("expires_at", qrEvent.ExpiresAt))

		// Update channel status to connecting
		channel.ConnectionStatus = entity.ConnectionStatusConnecting
		channel.UpdatedAt = time.Now()
		s.repo.UpdateConnectionStatus(ctx, channel.ID, entity.ConnectionStatusConnecting)

		// Store adapter in registry for future QR code refresh and connection handling
		if s.registry != nil {
			s.registry.RegisterChannelAdapter(channel.ID, adapter)
		}

		// Start goroutine to monitor login success and update channel status
		go s.monitorWhatsAppLoginStatus(channel.ID, adapter, qrChan)

		// Calculate expiry time in seconds
		expiresIn := int(time.Until(qrEvent.ExpiresAt).Seconds())
		if expiresIn < 0 {
			expiresIn = 60 // Default 60 seconds if already expired
		}

		return &ConnectResult{
			Channel:   channel,
			QRCode:    qrEvent.Code,
			ExpiresIn: expiresIn,
		}, nil

	case <-time.After(30 * time.Second):
		logger.Warn("Timeout waiting for WhatsApp QR code",
			zap.String("channel_id", channel.ID))
		adapter.Disconnect(bgCtx)
		return nil, errors.New(errors.ErrCodeTimeout, "timeout waiting for QR code")

	case <-ctx.Done():
		logger.Warn("Context cancelled during WhatsApp connection",
			zap.String("channel_id", channel.ID))
		// Don't disconnect - the background context keeps the connection alive
		// Just return the error for this request
		return nil, ctx.Err()
	}
}

// monitorWhatsAppLoginStatus monitors the QR channel for login success
// and updates the channel status accordingly
func (s *ChannelService) monitorWhatsAppLoginStatus(channelID string, adapter *whatsapp.Adapter, qrChan <-chan whatsapp.QRCodeEvent) {
	logger.Info("Started monitoring WhatsApp login status",
		zap.String("channel_id", channelID))

	// Monitor for up to 5 minutes (multiple QR codes may be generated)
	timeout := time.After(5 * time.Minute)

	for {
		select {
		case _, ok := <-qrChan:
			if !ok {
				// Channel closed - check if login was successful
				if adapter.IsLoggedIn() {
					logger.Info("WhatsApp login successful",
						zap.String("channel_id", channelID))
					s.updateChannelConnectionStatus(channelID, entity.ConnectionStatusConnected)
				} else {
					logger.Warn("WhatsApp login failed or timed out",
						zap.String("channel_id", channelID))
					s.updateChannelConnectionStatus(channelID, entity.ConnectionStatusDisconnected)
				}
				return
			}
			// New QR code event - just log it, frontend should poll for updates
			logger.Debug("New QR code event received",
				zap.String("channel_id", channelID))

		case <-timeout:
			logger.Warn("WhatsApp login monitoring timed out",
				zap.String("channel_id", channelID))
			if !adapter.IsLoggedIn() {
				s.updateChannelConnectionStatus(channelID, entity.ConnectionStatusDisconnected)
				adapter.Disconnect(context.Background())
			}
			return
		}
	}
}

// updateChannelConnectionStatus updates only the connection status in the database
func (s *ChannelService) updateChannelConnectionStatus(channelID string, status entity.ConnectionStatus) {
	ctx := context.Background()
	if err := s.repo.UpdateConnectionStatus(ctx, channelID, status); err != nil {
		logger.Error("Failed to update channel connection status",
			zap.String("channel_id", channelID),
			zap.String("status", string(status)),
			zap.Error(err))
	} else {
		logger.Info("Channel connection status updated",
			zap.String("channel_id", channelID),
			zap.String("connection_status", string(status)))
	}
}

// setupWhatsAppHandlers configures message and status handlers for WhatsApp adapter
func (s *ChannelService) setupWhatsAppHandlers(adapter *whatsapp.Adapter, channel *entity.Channel) {
	// Set message handler - called when a message is received from WhatsApp
	adapter.SetMessageHandler(func(ctx context.Context, msg *plugin.InboundMessage) error {
		logger.Debug("WhatsApp message received",
			zap.String("channel_id", channel.ID),
			zap.String("message_id", msg.ID),
			zap.String("sender_id", msg.SenderID),
			zap.String("content_type", string(msg.ContentType)))

		if s.producer == nil {
			logger.Warn("No NATS producer configured, skipping message",
				zap.String("channel_id", channel.ID))
			return nil // No producer configured, skip
		}

		// Convert attachments
		var attachments []nats.AttachmentData
		for _, att := range msg.Attachments {
			attachments = append(attachments, nats.AttachmentData{
				Type:     att.Type,
				URL:      att.URL,
				Filename: att.Filename,
				MimeType: att.MimeType,
				Metadata: att.Metadata,
			})
		}

		// Build inbound message for NATS
		inbound := &nats.InboundMessage{
			ID:          msg.ID,
			TenantID:    channel.TenantID,
			ChannelID:   channel.ID,
			ChannelType: string(channel.Type),
			ExternalID:  msg.ExternalID,
			ContentType: string(msg.ContentType),
			Content:     msg.Content,
			Metadata:    msg.Metadata,
			Attachments: attachments,
			Timestamp:   msg.Timestamp,
		}

		// Add sender info to metadata
		if inbound.Metadata == nil {
			inbound.Metadata = make(map[string]string)
		}
		inbound.Metadata["sender_id"] = msg.SenderID
		inbound.Metadata["sender_name"] = msg.SenderName

		// Publish to NATS
		if err := s.producer.PublishInbound(ctx, inbound); err != nil {
			logger.Error("Failed to publish WhatsApp message to NATS",
				zap.String("channel_id", channel.ID),
				zap.String("message_id", msg.ID),
				zap.Error(err))
			return err
		}
		return nil
	})

	// Set status handler - called when message status updates (delivered, read, etc.)
	adapter.SetStatusHandler(func(ctx context.Context, status *plugin.StatusCallback) error {
		logger.Debug("WhatsApp message status update",
			zap.String("channel_id", channel.ID),
			zap.String("message_id", status.MessageID),
			zap.String("status", string(status.Status)))

		if s.producer == nil {
			return nil
		}

		// Build status update for NATS
		statusUpdate := &nats.StatusUpdate{
			MessageID:   status.MessageID,
			ChannelType: string(channel.Type),
			Status:      string(status.Status),
			Timestamp:   status.Timestamp,
		}

		if status.ErrorMessage != "" {
			statusUpdate.ErrorMessage = status.ErrorMessage
			logger.Warn("WhatsApp message delivery failed",
				zap.String("channel_id", channel.ID),
				zap.String("message_id", status.MessageID),
				zap.String("error", status.ErrorMessage))
		}

		if err := s.producer.PublishStatusUpdate(ctx, statusUpdate); err != nil {
			logger.Error("Failed to publish status update to NATS",
				zap.String("channel_id", channel.ID),
				zap.String("message_id", status.MessageID),
				zap.Error(err))
			return err
		}
		return nil
	})

	// Set connection handler - called when connection state changes (login success, disconnect, etc.)
	adapter.SetConnectionHandler(func(ctx context.Context, connected bool, reason string) error {
		if connected {
			logger.Info("WhatsApp channel connected",
				zap.String("channel_id", channel.ID),
				zap.String("reason", reason))
		} else {
			logger.Warn("WhatsApp channel disconnected",
				zap.String("channel_id", channel.ID),
				zap.String("reason", reason))
		}

		// Update channel connection status in database
		var newStatus entity.ConnectionStatus
		if connected {
			newStatus = entity.ConnectionStatusConnected
		} else {
			newStatus = entity.ConnectionStatusDisconnected
		}

		if err := s.repo.UpdateConnectionStatus(ctx, channel.ID, newStatus); err != nil {
			logger.Error("Failed to update channel connection status",
				zap.String("channel_id", channel.ID),
				zap.Error(err))
			return err
		}

		// Optionally publish connection event to NATS
		if s.producer != nil {
			event := &nats.Event{
				Type:      "channel.connection",
				TenantID:  channel.TenantID,
				Timestamp: time.Now(),
				Payload: map[string]interface{}{
					"channel_id": channel.ID,
					"connected":  connected,
					"reason":     reason,
				},
			}
			if err := s.producer.PublishEvent(ctx, event); err != nil {
				logger.Error("Failed to publish connection event to NATS",
					zap.String("channel_id", channel.ID),
					zap.Error(err))
				return err
			}
		}

		return nil
	})
}

// Disconnect disconnects a channel
func (s *ChannelService) Disconnect(ctx context.Context, id string) error {
	logger.Info("Disconnecting channel", zap.String("channel_id", id))

	channel, err := s.repo.FindByID(ctx, id)
	if err != nil {
		logger.Error("Failed to find channel for disconnect",
			zap.String("channel_id", id),
			zap.Error(err))
		return err
	}

	// Disconnect adapter from registry if exists
	if s.registry != nil {
		if err := s.registry.DisconnectChannel(ctx, id); err != nil {
			logger.Warn("Failed to disconnect adapter from registry",
				zap.String("channel_id", id),
				zap.Error(err))
		}
	}

	// Update connection status to disconnected
	if err := s.repo.UpdateConnectionStatus(ctx, id, entity.ConnectionStatusDisconnected); err != nil {
		logger.Error("Failed to update channel connection status after disconnect",
			zap.String("channel_id", id),
			zap.Error(err))
		return err
	}

	channel.ConnectionStatus = entity.ConnectionStatusDisconnected
	channel.UpdatedAt = time.Now()

	logger.Info("Channel disconnected successfully", zap.String("channel_id", id))
	return nil
}

// ReconnectWhatsAppChannels reconnects all WhatsApp channels that have stored sessions
// This should be called on server startup to restore connections
func (s *ChannelService) ReconnectWhatsAppChannels(ctx context.Context) (int, error) {
	logger.Info("Starting WhatsApp channels reconnection")

	// Find all WhatsApp channels across all tenants
	channels, err := s.repo.FindByTypes(ctx, []entity.ChannelType{
		entity.ChannelTypeWhatsApp,
		entity.ChannelTypeWhatsAppUnofficial,
	})
	if err != nil {
		logger.Error("Failed to find WhatsApp channels for reconnection", zap.Error(err))
		return 0, err
	}

	logger.Info("Found WhatsApp channels to reconnect", zap.Int("count", len(channels)))

	reconnected := 0
	for _, channel := range channels {
		// Try to reconnect
		if err := s.reconnectWhatsAppChannel(ctx, channel); err != nil {
			logger.Warn("Failed to reconnect WhatsApp channel",
				zap.String("channel_id", channel.ID),
				zap.String("tenant_id", channel.TenantID),
				zap.Error(err))
			continue
		}
		reconnected++
	}

	return reconnected, nil
}

// RequestPairCode requests a pair code for WhatsApp authentication (alternative to QR code)
func (s *ChannelService) RequestPairCode(ctx context.Context, id string, phoneNumber string) (*ConnectResult, error) {
	logger.Info("Requesting WhatsApp pair code",
		zap.String("channel_id", id),
		zap.String("phone_number", phoneNumber))

	channel, err := s.repo.FindByID(ctx, id)
	if err != nil {
		logger.Error("Failed to find channel for pair code",
			zap.String("channel_id", id),
			zap.Error(err))
		return nil, err
	}

	// Only WhatsApp unofficial supports pair code
	if channel.Type != entity.ChannelTypeWhatsApp && channel.Type != entity.ChannelTypeWhatsAppUnofficial {
		logger.Warn("Pair code requested for non-WhatsApp channel",
			zap.String("channel_id", id),
			zap.String("channel_type", string(channel.Type)))
		return nil, errors.New(errors.ErrCodeValidation, "pair code is only available for WhatsApp channels")
	}

	// Check if adapter already exists in registry
	if s.registry != nil {
		if existingAdapter, err := s.registry.GetAdapterByChannelID(channel.ID); err == nil {
			waAdapter, ok := existingAdapter.(*whatsapp.Adapter)
			if ok {
				logger.Info("Using existing adapter for pair code",
					zap.String("channel_id", channel.ID))
				// Use existing adapter for pair code
				pairResp, err := waAdapter.LoginWithPairCode(ctx, phoneNumber)
				if err != nil {
					logger.Error("Failed to request pair code with existing adapter",
						zap.String("channel_id", channel.ID),
						zap.Error(err))
					return nil, errors.Wrap(err, errors.ErrCodeInternal, "failed to request pair code")
				}

				expiresIn := int(time.Until(pairResp.ExpiresAt).Seconds())
				if expiresIn < 0 {
					expiresIn = 300 // Default 5 minutes if already expired
				}

				logger.Info("Pair code generated successfully",
					zap.String("channel_id", channel.ID),
					zap.Int("expires_in", expiresIn))

				channel.ConnectionStatus = entity.ConnectionStatusConnecting
				channel.UpdatedAt = time.Now()
				s.repo.UpdateConnectionStatus(ctx, channel.ID, entity.ConnectionStatusConnecting)

				return &ConnectResult{
					Channel:   channel,
					PairCode:  pairResp.Code,
					ExpiresIn: expiresIn,
				}, nil
			}
		}
	}

	// Create new adapter instance
	adapter := whatsapp.NewAdapter()

	config := map[string]string{
		"channel_id":    channel.ID,
		"database_path": fmt.Sprintf("storages/whatsapp_%s.db", channel.ID),
	}

	if deviceName, ok := channel.Config["device_name"]; ok && deviceName != "" {
		config["device_name"] = deviceName
	}

	// Initialize adapter
	if err := adapter.Initialize(config); err != nil {
		return nil, errors.Wrap(err, errors.ErrCodeInternal, "failed to initialize WhatsApp adapter")
	}

	// Connect to WhatsApp servers
	if err := adapter.Connect(ctx); err != nil {
		return nil, errors.Wrap(err, errors.ErrCodeInternal, "failed to connect WhatsApp adapter")
	}

	// Set up message handlers
	s.setupWhatsAppHandlers(adapter, channel)

	// Request pair code
	pairResp, err := adapter.LoginWithPairCode(ctx, phoneNumber)
	if err != nil {
		adapter.Disconnect(ctx)
		return nil, errors.Wrap(err, errors.ErrCodeInternal, "failed to request pair code")
	}

	// Update channel connection status
	channel.ConnectionStatus = entity.ConnectionStatusConnecting
	channel.UpdatedAt = time.Now()
	s.repo.UpdateConnectionStatus(ctx, channel.ID, entity.ConnectionStatusConnecting)

	// Register adapter in registry
	if s.registry != nil {
		s.registry.RegisterChannelAdapter(channel.ID, adapter)
	}

	// Calculate expires in seconds
	expiresIn := int(time.Until(pairResp.ExpiresAt).Seconds())
	if expiresIn < 0 {
		expiresIn = 300 // Default 5 minutes if already expired
	}

	logger.Info("Pair code generated successfully",
		zap.String("channel_id", channel.ID),
		zap.Int("expires_in", expiresIn))

	return &ConnectResult{
		Channel:   channel,
		PairCode:  pairResp.Code,
		ExpiresIn: expiresIn,
	}, nil
}

// reconnectWhatsAppChannel attempts to reconnect a single WhatsApp channel
func (s *ChannelService) reconnectWhatsAppChannel(ctx context.Context, channel *entity.Channel) error {
	logger.Info("Attempting to reconnect WhatsApp channel",
		zap.String("channel_id", channel.ID),
		zap.String("tenant_id", channel.TenantID))

	// Create adapter instance
	adapter := whatsapp.NewAdapter()

	config := map[string]string{
		"channel_id":    channel.ID,
		"database_path": fmt.Sprintf("storages/whatsapp_%s.db", channel.ID),
	}

	if deviceName, ok := channel.Config["device_name"]; ok && deviceName != "" {
		config["device_name"] = deviceName
	}

	// Initialize adapter
	if err := adapter.Initialize(config); err != nil {
		logger.Error("Failed to initialize adapter during reconnect",
			zap.String("channel_id", channel.ID),
			zap.Error(err))
		return err
	}

	// Connect to WhatsApp servers
	if err := adapter.Connect(ctx); err != nil {
		logger.Error("Failed to connect adapter during reconnect",
			zap.String("channel_id", channel.ID),
			zap.Error(err))
		return err
	}

	// Set up handlers
	s.setupWhatsAppHandlers(adapter, channel)

	// Check if session is still valid
	if !adapter.IsLoggedIn() {
		// No valid session, disconnect and mark as disconnected
		logger.Info("No valid WhatsApp session found, marking as disconnected",
			zap.String("channel_id", channel.ID))
		adapter.Disconnect(ctx)
		channel.ConnectionStatus = entity.ConnectionStatusDisconnected
		channel.UpdatedAt = time.Now()
		s.repo.UpdateConnectionStatus(ctx, channel.ID, entity.ConnectionStatusDisconnected)
		return nil
	}

	// Session is valid, mark as connected
	logger.Info("WhatsApp channel reconnected successfully with stored session",
		zap.String("channel_id", channel.ID))
	channel.ConnectionStatus = entity.ConnectionStatusConnected
	channel.UpdatedAt = time.Now()
	s.repo.UpdateConnectionStatus(ctx, channel.ID, entity.ConnectionStatusConnected)

	// Register adapter in registry
	if s.registry != nil {
		s.registry.RegisterChannelAdapter(channel.ID, adapter)
	}

	return nil
}
