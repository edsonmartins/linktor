package rcs

import (
	"context"
	"testing"

	"github.com/msgfy/linktor/pkg/plugin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ========== NewAdapter ==========

func TestNewAdapter_NotNil(t *testing.T) {
	adapter := NewAdapter()
	assert.NotNil(t, adapter)
}

func TestNewAdapter_ChannelType(t *testing.T) {
	adapter := NewAdapter()
	assert.Equal(t, plugin.ChannelTypeRCS, adapter.GetChannelType())
}

func TestNewAdapter_ChannelInfo(t *testing.T) {
	adapter := NewAdapter()
	info := adapter.GetChannelInfo()
	require.NotNil(t, info)
	assert.Equal(t, plugin.ChannelTypeRCS, info.Type)
	assert.Equal(t, "RCS Business Messaging", info.Name)
	assert.Equal(t, "1.0.0", info.Version)
	assert.Equal(t, "Linktor Team", info.Author)
	assert.NotEmpty(t, info.Description)
}

// ========== Capabilities ==========

func TestNewAdapter_Capabilities_ContentTypes(t *testing.T) {
	adapter := NewAdapter()
	info := adapter.GetChannelInfo()
	require.NotNil(t, info.Capabilities)

	expected := []plugin.ContentType{
		plugin.ContentTypeText,
		plugin.ContentTypeImage,
		plugin.ContentTypeVideo,
		plugin.ContentTypeDocument,
		plugin.ContentTypeLocation,
		plugin.ContentTypeInteractive,
	}
	assert.Equal(t, expected, info.Capabilities.SupportedContentTypes)
}

func TestNewAdapter_Capabilities_MaxMessageLength(t *testing.T) {
	adapter := NewAdapter()
	info := adapter.GetChannelInfo()
	require.NotNil(t, info.Capabilities)
	assert.Equal(t, 3072, info.Capabilities.MaxMessageLength)
}

func TestNewAdapter_Capabilities_MaxMediaSize(t *testing.T) {
	adapter := NewAdapter()
	info := adapter.GetChannelInfo()
	require.NotNil(t, info.Capabilities)
	assert.Equal(t, int64(10*1024*1024), info.Capabilities.MaxMediaSize) // 10MB
}

func TestNewAdapter_Capabilities_Flags(t *testing.T) {
	adapter := NewAdapter()
	caps := adapter.GetChannelInfo().Capabilities
	require.NotNil(t, caps)

	assert.True(t, caps.SupportsMedia)
	assert.True(t, caps.SupportsLocation)
	assert.False(t, caps.SupportsTemplates)
	assert.True(t, caps.SupportsInteractive)
	assert.True(t, caps.SupportsReadReceipts)
	assert.True(t, caps.SupportsTypingIndicator)
	assert.False(t, caps.SupportsReactions)
	assert.True(t, caps.SupportsReplies)
	assert.False(t, caps.SupportsForwarding)
	assert.Equal(t, 1, caps.MaxAttachments)
}

func TestNewAdapter_Capabilities_MediaTypes(t *testing.T) {
	adapter := NewAdapter()
	caps := adapter.GetChannelInfo().Capabilities
	require.NotNil(t, caps)

	expected := []string{
		"image/jpeg", "image/png", "image/gif",
		"video/mp4", "video/3gpp",
		"audio/mp3", "audio/aac",
		"application/pdf",
	}
	assert.Equal(t, expected, caps.SupportedMediaTypes)
}

// ========== Initialize ==========

func TestInitialize_ValidConfig(t *testing.T) {
	adapter := NewAdapter()
	err := adapter.Initialize(map[string]string{
		"provider":       "zenvia",
		"agent_id":       "agent-123",
		"api_key":        "key-abc",
		"api_secret":     "secret-xyz",
		"webhook_url":    "https://example.com/webhook",
		"webhook_secret": "wh-secret",
		"base_url":       "https://custom.api.com",
		"sender_id":      "MyBrand",
	})
	assert.NoError(t, err)

	assert.Equal(t, Provider("zenvia"), adapter.config.Provider)
	assert.Equal(t, "agent-123", adapter.config.AgentID)
	assert.Equal(t, "key-abc", adapter.config.APIKey)
	assert.Equal(t, "secret-xyz", adapter.config.APISecret)
	assert.Equal(t, "https://example.com/webhook", adapter.config.WebhookURL)
	assert.Equal(t, "wh-secret", adapter.config.WebhookSecret)
	assert.Equal(t, "https://custom.api.com", adapter.config.BaseURL)
	assert.Equal(t, "MyBrand", adapter.config.SenderID)
}

func TestInitialize_MissingProvider_DefaultsToZenvia(t *testing.T) {
	adapter := NewAdapter()
	err := adapter.Initialize(map[string]string{
		"agent_id": "agent-123",
		"api_key":  "key-abc",
	})
	assert.NoError(t, err)
	assert.Equal(t, ProviderZenvia, adapter.config.Provider)
}

func TestInitialize_EmptyConfig(t *testing.T) {
	adapter := NewAdapter()
	err := adapter.Initialize(map[string]string{})
	assert.NoError(t, err)
	// Provider defaults to zenvia when empty
	assert.Equal(t, ProviderZenvia, adapter.config.Provider)
}

func TestInitialize_AllProviders(t *testing.T) {
	providers := []string{"zenvia", "infobip", "pontaltech", "google"}
	for _, p := range providers {
		t.Run(p, func(t *testing.T) {
			adapter := NewAdapter()
			err := adapter.Initialize(map[string]string{
				"provider": p,
				"agent_id": "agent",
				"api_key":  "key",
			})
			assert.NoError(t, err)
			assert.Equal(t, Provider(p), adapter.config.Provider)
		})
	}
}

// ========== Interface compliance ==========

func TestAdapter_ImplementsChannelAdapter(t *testing.T) {
	var _ plugin.ChannelAdapter = (*Adapter)(nil)
}

func TestAdapter_ImplementsChannelAdapterWithWebhook(t *testing.T) {
	var _ plugin.ChannelAdapterWithWebhook = (*Adapter)(nil)
}

// ========== GetWebhookPath ==========

func TestAdapter_GetWebhookPath(t *testing.T) {
	adapter := NewAdapter()
	assert.Equal(t, "/webhooks/rcs", adapter.GetWebhookPath())
}

// ========== GetConnectionStatus ==========

func TestAdapter_GetConnectionStatus_Disconnected(t *testing.T) {
	adapter := NewAdapter()
	status := adapter.GetConnectionStatus()
	assert.False(t, status.Connected)
	assert.Equal(t, "disconnected", status.Status)
}

// ========== ValidateWebhook (adapter level) ==========

func TestAdapter_ValidateWebhook_NoClient(t *testing.T) {
	adapter := NewAdapter()
	// No client connected, should return false
	result := adapter.ValidateWebhook(map[string]string{
		"X-Signature": "some-sig",
	}, []byte("body"))
	assert.False(t, result)
}

// ========== GetClient ==========

func TestAdapter_GetClient_NilWhenNotConnected(t *testing.T) {
	adapter := NewAdapter()
	assert.Nil(t, adapter.GetClient())
}

// ========== SetMessageHandler / SetStatusHandler ==========

func TestAdapter_SetMessageHandler(t *testing.T) {
	adapter := NewAdapter()
	called := false
	adapter.SetMessageHandler(func(ctx context.Context, msg *plugin.InboundMessage) error {
		called = true
		return nil
	})
	// Just verify no panic; handler stored internally
	assert.False(t, called) // not called yet
}

func TestAdapter_SetStatusHandler(t *testing.T) {
	adapter := NewAdapter()
	called := false
	adapter.SetStatusHandler(func(ctx context.Context, cb *plugin.StatusCallback) error {
		called = true
		return nil
	})
	assert.False(t, called)
}
