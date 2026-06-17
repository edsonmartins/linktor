package outbound

import (
	"context"
	"fmt"
	"strings"

	"github.com/msgfy/linktor/pkg/plugin"
)

// AdapterRegistry is the subset of the plugin registry the plugin sender needs.
type AdapterRegistry interface {
	GetAdapterByChannelID(channelID string) (plugin.ChannelAdapter, error)
	GetAdapter(channelType plugin.ChannelType) (plugin.ChannelAdapter, error)
}

// PluginSenderFactory builds a Sender that delivers through a live plugin
// adapter, for STATEFUL channels (webchat, WhatsApp-unofficial/whatsmeow) whose
// delivery rides an existing session/WebSocket connection rather than a
// stateless HTTP client. It bridges those channels onto the same unified
// outbound worker (retry, DLQ, status correlation) as the stateless senders.
type PluginSenderFactory struct {
	channelType string
	registry    AdapterRegistry
}

// NewPluginSenderFactory creates a factory for a stateful channel type backed
// by the plugin registry.
func NewPluginSenderFactory(channelType string, registry AdapterRegistry) *PluginSenderFactory {
	return &PluginSenderFactory{channelType: channelType, registry: registry}
}

// ChannelType implements outbound.Factory.
func (f *PluginSenderFactory) ChannelType() string { return f.channelType }

// New returns the plugin-backed sender. Credentials are unused: a stateful
// channel's session already holds them.
func (f *PluginSenderFactory) New(_ map[string]string) (Sender, error) {
	return &pluginSender{channelType: f.channelType, registry: f.registry}, nil
}

type pluginSender struct {
	channelType string
	registry    AdapterRegistry
}

func (s *pluginSender) Send(ctx context.Context, msg *Message) (*Receipt, error) {
	if msg.To == "" {
		return nil, Permanentf("recipient is required")
	}

	// Prefer the per-channel adapter (e.g. a whatsmeow session bound to this
	// channel); fall back to the type-level adapter (e.g. the shared webchat hub
	// keyed by recipient).
	adapter, err := s.registry.GetAdapterByChannelID(msg.ChannelID)
	if err != nil || adapter == nil {
		adapter, err = s.registry.GetAdapter(plugin.ChannelType(s.channelType))
		if err != nil || adapter == nil {
			// No live session/connection right now: retry (transient).
			return nil, fmt.Errorf("no live %s adapter for channel %s", s.channelType, msg.ChannelID)
		}
	}

	res, err := adapter.SendMessage(ctx, toPluginMessage(msg))
	if err != nil {
		return nil, err
	}
	if res == nil || !res.Success {
		reason := "delivery failed"
		if res != nil && res.Error != "" {
			reason = res.Error
		}
		return nil, fmt.Errorf("%s", reason) // transient -> retried by the worker
	}
	return &Receipt{ProviderMessageID: res.ExternalID}, nil
}

// toPluginMessage maps a typed outbound.Message onto the plugin transport shape
// the stateful adapters consume.
func toPluginMessage(msg *Message) *plugin.OutboundMessage {
	meta := map[string]string{}
	for k, v := range msg.Metadata {
		meta[k] = v
	}

	out := &plugin.OutboundMessage{
		ID:          msg.ID,
		RecipientID: msg.To,
		Metadata:    meta,
	}

	switch c := msg.Content.(type) {
	case Text:
		out.ContentType = plugin.ContentTypeText
		out.Content = c.Body
	case Template:
		out.ContentType = plugin.ContentTypeText
		out.Content = strings.TrimSpace(strings.Join(c.BodyParams, " "))
	case Media:
		out.ContentType = plugin.ContentType(c.Type)
		out.Content = c.Caption
		if c.URL != "" {
			out.Metadata["media_url"] = c.URL
		}
		if c.MediaID != "" {
			out.Metadata["media_id"] = c.MediaID
		}
		if c.Filename != "" {
			out.Metadata["filename"] = c.Filename
		}
	default:
		out.ContentType = plugin.ContentTypeText
	}
	return out
}
