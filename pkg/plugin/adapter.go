package plugin

import (
	"context"
)

// ChannelAdapter is the interface that all channel adapters must implement
type ChannelAdapter interface {
	// Lifecycle methods

	// Initialize sets up the adapter with the given configuration
	Initialize(config map[string]string) error

	// Connect establishes connection to the channel provider
	Connect(ctx context.Context) error

	// Disconnect closes the connection to the channel provider
	Disconnect(ctx context.Context) error

	// IsConnected returns true if the adapter is connected
	IsConnected() bool

	// GetConnectionStatus returns detailed connection status
	GetConnectionStatus() *ConnectionStatus

	// Messaging methods

	// SendMessage sends a message through the channel
	SendMessage(ctx context.Context, msg *OutboundMessage) (*SendResult, error)

	// SendTypingIndicator sends a typing indicator
	SendTypingIndicator(ctx context.Context, indicator *TypingIndicator) error

	// SendReadReceipt marks a message as read
	SendReadReceipt(ctx context.Context, receipt *ReadReceipt) error

	// Media methods

	// UploadMedia uploads media to the channel provider
	UploadMedia(ctx context.Context, media *Media) (*MediaUpload, error)

	// DownloadMedia downloads media from the channel provider
	DownloadMedia(ctx context.Context, mediaID string) (*Media, error)

	// Channel info methods

	// GetChannelType returns the type of channel this adapter handles
	GetChannelType() ChannelType

	// GetChannelInfo returns detailed information about the channel adapter
	GetChannelInfo() *ChannelInfo

	// GetCapabilities returns the capabilities of this channel
	GetCapabilities() *ChannelCapabilities
}

// MessageHandler is called when an inbound message is received
type MessageHandler func(ctx context.Context, msg *InboundMessage) error

// StatusHandler is called when a message status update is received
type StatusHandler func(ctx context.Context, status *StatusCallback) error

// ChannelAdapterWithWebhook extends ChannelAdapter for adapters that receive webhooks
type ChannelAdapterWithWebhook interface {
	ChannelAdapter

	// SetMessageHandler sets the handler for inbound messages
	SetMessageHandler(handler MessageHandler)

	// SetStatusHandler sets the handler for status updates
	SetStatusHandler(handler StatusHandler)

	// GetWebhookPath returns the path for webhook registration
	GetWebhookPath() string

	// ValidateWebhook validates a webhook request
	ValidateWebhook(headers map[string]string, body []byte) bool
}

// ChannelAdapterWithPolling extends ChannelAdapter for adapters that poll for messages
type ChannelAdapterWithPolling interface {
	ChannelAdapter

	// StartPolling starts polling for new messages
	StartPolling(ctx context.Context, handler MessageHandler) error

	// StopPolling stops polling
	StopPolling() error

	// GetPollingInterval returns the polling interval
	GetPollingInterval() int // in seconds
}

// ChannelAdapterWithWebSocket extends ChannelAdapter for adapters that use WebSocket
type ChannelAdapterWithWebSocket interface {
	ChannelAdapter

	// SetMessageHandler sets the handler for inbound messages
	SetMessageHandler(handler MessageHandler)

	// SetStatusHandler sets the handler for status updates
	SetStatusHandler(handler StatusHandler)

	// StartWebSocket starts the WebSocket connection
	StartWebSocket(ctx context.Context) error

	// StopWebSocket stops the WebSocket connection
	StopWebSocket() error
}

// BaseAdapter provides a base implementation with common functionality
type BaseAdapter struct {
	channelType ChannelType
	config      map[string]string
	connected   bool
	info        *ChannelInfo
}

// NewBaseAdapter creates a new base adapter
func NewBaseAdapter(channelType ChannelType, info *ChannelInfo) *BaseAdapter {
	return &BaseAdapter{
		channelType: channelType,
		config:      make(map[string]string),
		connected:   false,
		info:        info,
	}
}

// Initialize sets up the base adapter
func (b *BaseAdapter) Initialize(config map[string]string) error {
	b.config = config
	return nil
}

// GetConfig returns a config value
func (b *BaseAdapter) GetConfig(key string) string {
	return b.config[key]
}

// SetConnected sets the connection status
func (b *BaseAdapter) SetConnected(connected bool) {
	b.connected = connected
}

// IsConnected returns the connection status
func (b *BaseAdapter) IsConnected() bool {
	return b.connected
}

// GetChannelType returns the channel type
func (b *BaseAdapter) GetChannelType() ChannelType {
	return b.channelType
}

// GetChannelInfo returns the channel info
func (b *BaseAdapter) GetChannelInfo() *ChannelInfo {
	return b.info
}

// GetCapabilities returns the channel capabilities
func (b *BaseAdapter) GetCapabilities() *ChannelCapabilities {
	if b.info != nil {
		return b.info.Capabilities
	}
	return nil
}

// GetConnectionStatus returns the connection status
func (b *BaseAdapter) GetConnectionStatus() *ConnectionStatus {
	return &ConnectionStatus{
		Connected: b.connected,
		Status:    "connected",
	}
}

// Default implementations that return errors (should be overridden)

// SendTypingIndicator default implementation
func (b *BaseAdapter) SendTypingIndicator(ctx context.Context, indicator *TypingIndicator) error {
	return nil // No-op by default
}

// SendReadReceipt default implementation
func (b *BaseAdapter) SendReadReceipt(ctx context.Context, receipt *ReadReceipt) error {
	return nil // No-op by default
}

// UploadMedia default implementation
func (b *BaseAdapter) UploadMedia(ctx context.Context, media *Media) (*MediaUpload, error) {
	return &MediaUpload{Success: false, Error: "not implemented"}, nil
}

// DownloadMedia default implementation
func (b *BaseAdapter) DownloadMedia(ctx context.Context, mediaID string) (*Media, error) {
	return nil, nil
}
