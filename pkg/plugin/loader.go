package plugin

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sync"

	"github.com/hashicorp/go-plugin"
	"google.golang.org/grpc"
)

// HandshakeConfig is used to identify plugins
var HandshakeConfig = plugin.HandshakeConfig{
	ProtocolVersion:  1,
	MagicCookieKey:   "LINKTOR_PLUGIN",
	MagicCookieValue: "channel_adapter",
}

// PluginMap is the map of available plugin types
var PluginMap = map[string]plugin.Plugin{
	"channel_adapter": &ChannelAdapterPlugin{},
}

// ChannelAdapterPlugin is the plugin wrapper for channel adapters
type ChannelAdapterPlugin struct {
	plugin.Plugin
	Impl ChannelAdapter
}

// GRPCServer registers the plugin with gRPC
func (p *ChannelAdapterPlugin) GRPCServer(broker *plugin.GRPCBroker, s *grpc.Server) error {
	RegisterChannelAdapterServer(s, &GRPCServer{Impl: p.Impl})
	return nil
}

// GRPCClient returns the plugin client
func (p *ChannelAdapterPlugin) GRPCClient(ctx context.Context, broker *plugin.GRPCBroker, c *grpc.ClientConn) (interface{}, error) {
	return &GRPCClient{client: NewChannelAdapterClient(c)}, nil
}

// Loader manages external plugins
type Loader struct {
	mu      sync.RWMutex
	plugins map[string]*plugin.Client
	dir     string
}

// NewLoader creates a new plugin loader
func NewLoader(pluginDir string) *Loader {
	return &Loader{
		plugins: make(map[string]*plugin.Client),
		dir:     pluginDir,
	}
}

// LoadPlugin loads a plugin from a file
func (l *Loader) LoadPlugin(ctx context.Context, name string) (ChannelAdapter, error) {
	l.mu.Lock()
	defer l.mu.Unlock()

	// Check if already loaded
	if client, exists := l.plugins[name]; exists {
		raw, err := client.Client()
		if err != nil {
			return nil, fmt.Errorf("failed to get plugin client: %w", err)
		}
		adapter, err := raw.Dispense("channel_adapter")
		if err != nil {
			return nil, fmt.Errorf("failed to dispense plugin: %w", err)
		}
		return adapter.(ChannelAdapter), nil
	}

	// Find plugin binary
	pluginPath := filepath.Join(l.dir, name)
	if _, err := os.Stat(pluginPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("plugin not found: %s", pluginPath)
	}

	// Create plugin client
	client := plugin.NewClient(&plugin.ClientConfig{
		HandshakeConfig:  HandshakeConfig,
		Plugins:          PluginMap,
		Cmd:              exec.Command(pluginPath),
		AllowedProtocols: []plugin.Protocol{plugin.ProtocolGRPC},
	})

	// Connect via gRPC
	rpcClient, err := client.Client()
	if err != nil {
		client.Kill()
		return nil, fmt.Errorf("failed to connect to plugin: %w", err)
	}

	// Request the channel adapter
	raw, err := rpcClient.Dispense("channel_adapter")
	if err != nil {
		client.Kill()
		return nil, fmt.Errorf("failed to dispense plugin: %w", err)
	}

	adapter, ok := raw.(ChannelAdapter)
	if !ok {
		client.Kill()
		return nil, fmt.Errorf("plugin does not implement ChannelAdapter interface")
	}

	l.plugins[name] = client
	return adapter, nil
}

// UnloadPlugin unloads a plugin
func (l *Loader) UnloadPlugin(name string) error {
	l.mu.Lock()
	defer l.mu.Unlock()

	if client, exists := l.plugins[name]; exists {
		client.Kill()
		delete(l.plugins, name)
	}

	return nil
}

// DiscoverPlugins finds all plugins in the plugin directory
func (l *Loader) DiscoverPlugins() ([]string, error) {
	if l.dir == "" {
		return nil, nil
	}

	entries, err := os.ReadDir(l.dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to read plugin directory: %w", err)
	}

	var plugins []string
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		// Check if executable
		info, err := entry.Info()
		if err != nil {
			continue
		}
		if info.Mode()&0111 != 0 {
			plugins = append(plugins, entry.Name())
		}
	}

	return plugins, nil
}

// Close terminates all loaded plugins
func (l *Loader) Close() {
	l.mu.Lock()
	defer l.mu.Unlock()

	for name, client := range l.plugins {
		client.Kill()
		delete(l.plugins, name)
	}
}

// LoadedPlugins returns the names of all loaded plugins
func (l *Loader) LoadedPlugins() []string {
	l.mu.RLock()
	defer l.mu.RUnlock()

	names := make([]string, 0, len(l.plugins))
	for name := range l.plugins {
		names = append(names, name)
	}

	return names
}
