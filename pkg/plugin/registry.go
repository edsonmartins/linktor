package plugin

import (
	"context"
	"fmt"
	"sync"
)

// Registry manages channel adapters
type Registry struct {
	mu       sync.RWMutex
	adapters map[ChannelType]ChannelAdapter
	configs  map[string]ChannelAdapter // channelID -> adapter
}

// NewRegistry creates a new adapter registry
func NewRegistry() *Registry {
	return &Registry{
		adapters: make(map[ChannelType]ChannelAdapter),
		configs:  make(map[string]ChannelAdapter),
	}
}

// RegisterAdapter registers a channel adapter type
func (r *Registry) RegisterAdapter(channelType ChannelType, adapter ChannelAdapter) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.adapters[channelType]; exists {
		return fmt.Errorf("adapter for channel type %s already registered", channelType)
	}

	r.adapters[channelType] = adapter
	return nil
}

// UnregisterAdapter removes a channel adapter type
func (r *Registry) UnregisterAdapter(channelType ChannelType) {
	r.mu.Lock()
	defer r.mu.Unlock()

	delete(r.adapters, channelType)
}

// GetAdapter returns the adapter for a channel type
func (r *Registry) GetAdapter(channelType ChannelType) (ChannelAdapter, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	adapter, exists := r.adapters[channelType]
	if !exists {
		return nil, fmt.Errorf("no adapter registered for channel type %s", channelType)
	}

	return adapter, nil
}

// GetAdapterByChannelID returns the adapter instance for a specific channel
func (r *Registry) GetAdapterByChannelID(channelID string) (ChannelAdapter, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	adapter, exists := r.configs[channelID]
	if !exists {
		return nil, fmt.Errorf("no adapter configured for channel %s", channelID)
	}

	return adapter, nil
}

// ConfigureChannel creates and configures an adapter instance for a specific channel
func (r *Registry) ConfigureChannel(ctx context.Context, channelID string, channelType ChannelType, config map[string]string) (ChannelAdapter, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Get the adapter template
	template, exists := r.adapters[channelType]
	if !exists {
		return nil, fmt.Errorf("no adapter registered for channel type %s", channelType)
	}

	// Create a new instance (adapters should implement cloning or we need a factory)
	// For now, we'll initialize the template directly
	// In production, you'd want to create a new instance
	if err := template.Initialize(config); err != nil {
		return nil, fmt.Errorf("failed to initialize adapter: %w", err)
	}

	if err := template.Connect(ctx); err != nil {
		return nil, fmt.Errorf("failed to connect adapter: %w", err)
	}

	r.configs[channelID] = template
	return template, nil
}

// DisconnectChannel disconnects and removes a channel adapter instance
func (r *Registry) DisconnectChannel(ctx context.Context, channelID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	adapter, exists := r.configs[channelID]
	if !exists {
		return nil
	}

	if err := adapter.Disconnect(ctx); err != nil {
		return fmt.Errorf("failed to disconnect adapter: %w", err)
	}

	delete(r.configs, channelID)
	return nil
}

// ListAdapters returns all registered adapter types
func (r *Registry) ListAdapters() []ChannelType {
	r.mu.RLock()
	defer r.mu.RUnlock()

	types := make([]ChannelType, 0, len(r.adapters))
	for channelType := range r.adapters {
		types = append(types, channelType)
	}

	return types
}

// ListConfiguredChannels returns all configured channel IDs
func (r *Registry) ListConfiguredChannels() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	ids := make([]string, 0, len(r.configs))
	for id := range r.configs {
		ids = append(ids, id)
	}

	return ids
}

// GetAdapterInfo returns info about all registered adapters
func (r *Registry) GetAdapterInfo() []*ChannelInfo {
	r.mu.RLock()
	defer r.mu.RUnlock()

	info := make([]*ChannelInfo, 0, len(r.adapters))
	for _, adapter := range r.adapters {
		info = append(info, adapter.GetChannelInfo())
	}

	return info
}

// Close disconnects all adapters and cleans up
func (r *Registry) Close(ctx context.Context) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	var lastErr error
	for id, adapter := range r.configs {
		if err := adapter.Disconnect(ctx); err != nil {
			lastErr = err
		}
		delete(r.configs, id)
	}

	return lastErr
}

// Global registry instance
var globalRegistry = NewRegistry()

// GetGlobalRegistry returns the global adapter registry
func GetGlobalRegistry() *Registry {
	return globalRegistry
}

// Register is a convenience function to register an adapter in the global registry
func Register(channelType ChannelType, adapter ChannelAdapter) error {
	return globalRegistry.RegisterAdapter(channelType, adapter)
}

// Get is a convenience function to get an adapter from the global registry
func Get(channelType ChannelType) (ChannelAdapter, error) {
	return globalRegistry.GetAdapter(channelType)
}
