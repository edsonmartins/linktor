package webchat

import (
	"context"
	"testing"
	"time"

	"github.com/msgfy/linktor/pkg/plugin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewAdapter(t *testing.T) {
	a := NewAdapter()
	require.NotNil(t, a)

	info := a.GetChannelInfo()
	assert.Equal(t, plugin.ChannelTypeWebChat, info.Type)
	assert.Equal(t, "Web Chat", info.Name)

	caps := info.Capabilities
	require.NotNil(t, caps)
	assert.True(t, caps.SupportsMedia)
	assert.True(t, caps.SupportsInteractive)
	assert.True(t, caps.SupportsReadReceipts)
	assert.True(t, caps.SupportsTypingIndicator)
	assert.True(t, caps.SupportsReplies)
	assert.False(t, caps.SupportsLocation)
	assert.False(t, caps.SupportsTemplates)
	assert.Equal(t, 4096, caps.MaxMessageLength)
	assert.Equal(t, int64(10*1024*1024), caps.MaxMediaSize)
	assert.Equal(t, 5, caps.MaxAttachments)
	assert.Contains(t, caps.SupportedContentTypes, plugin.ContentTypeText)
	assert.Contains(t, caps.SupportedContentTypes, plugin.ContentTypeImage)
	assert.Contains(t, caps.SupportedContentTypes, plugin.ContentTypeDocument)
}

func TestAdapter_Initialize(t *testing.T) {
	t.Run("with defaults", func(t *testing.T) {
		a := NewAdapter()
		err := a.Initialize(map[string]string{})
		require.NoError(t, err)

		cfg := a.GetConfig()
		assert.Equal(t, "Chat with us", cfg.WidgetTitle)
		assert.Equal(t, "#007bff", cfg.WidgetColor)
		assert.Contains(t, cfg.WelcomeMessage, "Hello")
		assert.False(t, cfg.AllowAttachments)
		assert.False(t, cfg.RequireEmail)
	})

	t.Run("with custom config", func(t *testing.T) {
		a := NewAdapter()
		err := a.Initialize(map[string]string{
			"widget_title":      "Support",
			"widget_color":      "#ff0000",
			"welcome_message":   "Hi there!",
			"allow_attachments": "true",
			"require_email":     "true",
			"require_name":      "true",
			"avatar_url":        "https://example.com/avatar.png",
		})
		require.NoError(t, err)

		cfg := a.GetConfig()
		assert.Equal(t, "Support", cfg.WidgetTitle)
		assert.Equal(t, "#ff0000", cfg.WidgetColor)
		assert.Equal(t, "Hi there!", cfg.WelcomeMessage)
		assert.True(t, cfg.AllowAttachments)
		assert.True(t, cfg.RequireEmail)
		assert.True(t, cfg.RequireName)
		assert.Equal(t, "https://example.com/avatar.png", cfg.AvatarURL)
	})
}

func TestAdapter_ConnectDisconnect(t *testing.T) {
	a := NewAdapter()
	a.Initialize(map[string]string{})
	ctx := context.Background()

	assert.False(t, a.IsConnected())
	assert.Nil(t, a.GetHub())

	// Connect
	err := a.Connect(ctx)
	require.NoError(t, err)
	assert.True(t, a.IsConnected())
	assert.NotNil(t, a.GetHub())

	// Connect again is idempotent
	err = a.Connect(ctx)
	require.NoError(t, err)

	// Disconnect
	err = a.Disconnect(ctx)
	require.NoError(t, err)
	assert.False(t, a.IsConnected())
	assert.Nil(t, a.GetHub())

	// Disconnect again is idempotent
	err = a.Disconnect(ctx)
	require.NoError(t, err)
}

func TestAdapter_SendMessage_NotConnected(t *testing.T) {
	a := NewAdapter()
	a.Initialize(map[string]string{})
	ctx := context.Background()

	result, err := a.SendMessage(ctx, &plugin.OutboundMessage{
		RecipientID: "session-1",
		Content:     "hello",
		ContentType: plugin.ContentTypeText,
	})
	require.NoError(t, err)
	assert.False(t, result.Success)
	assert.Equal(t, plugin.MessageStatusFailed, result.Status)
	assert.Equal(t, "adapter not connected", result.Error)
}

func TestAdapter_SendMessage_ClientNotFound(t *testing.T) {
	a := NewAdapter()
	a.Initialize(map[string]string{})
	ctx := context.Background()

	a.Connect(ctx)
	defer a.Disconnect(ctx)

	// Give hub time to start
	time.Sleep(10 * time.Millisecond)

	result, err := a.SendMessage(ctx, &plugin.OutboundMessage{
		RecipientID: "nonexistent-session",
		Content:     "hello",
		ContentType: plugin.ContentTypeText,
	})
	require.NoError(t, err)
	assert.False(t, result.Success)
	assert.Equal(t, "client not connected", result.Error)
}

func TestAdapter_SendTypingIndicator_NotConnected(t *testing.T) {
	a := NewAdapter()
	a.Initialize(map[string]string{})
	ctx := context.Background()

	err := a.SendTypingIndicator(ctx, &plugin.TypingIndicator{
		RecipientID: "session-1",
		IsTyping:    true,
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not connected")
}

func TestAdapter_SendReadReceipt_NotConnected(t *testing.T) {
	a := NewAdapter()
	a.Initialize(map[string]string{})
	ctx := context.Background()

	err := a.SendReadReceipt(ctx, &plugin.ReadReceipt{
		RecipientID: "session-1",
		MessageID:   "msg-1",
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not connected")
}

func TestAdapter_HandleInboundMessage(t *testing.T) {
	t.Run("no handler configured", func(t *testing.T) {
		a := NewAdapter()
		a.Initialize(map[string]string{})
		ctx := context.Background()

		err := a.HandleInboundMessage(ctx, "session-1", &MessagePayload{
			Content:     "hello",
			ContentType: "text",
		})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "no message handler")
	})

	t.Run("with handler", func(t *testing.T) {
		a := NewAdapter()
		a.Initialize(map[string]string{})
		ctx := context.Background()

		var received *plugin.InboundMessage
		a.SetMessageHandler(func(ctx context.Context, msg *plugin.InboundMessage) error {
			received = msg
			return nil
		})

		err := a.HandleInboundMessage(ctx, "session-1", &MessagePayload{
			ID:          "msg-1",
			Content:     "hello world",
			ContentType: "text",
			SenderName:  "Alice",
			Attachments: []AttachmentPayload{
				{Type: "image", URL: "https://example.com/img.png", MimeType: "image/png"},
			},
		})
		require.NoError(t, err)

		require.NotNil(t, received)
		assert.Equal(t, "session-1", received.SenderID)
		assert.Equal(t, "Alice", received.SenderName)
		assert.Equal(t, "hello world", received.Content)
		assert.Equal(t, plugin.ContentType("text"), received.ContentType)
		assert.Equal(t, "msg-1", received.ExternalID)
		assert.Equal(t, "session-1", received.Metadata["session_id"])
		require.Len(t, received.Attachments, 1)
		assert.Equal(t, "image", received.Attachments[0].Type)
		assert.Equal(t, "https://example.com/img.png", received.Attachments[0].URL)
	})
}

func TestAdapter_HandleClientConnect(t *testing.T) {
	a := NewAdapter()
	a.Initialize(map[string]string{"welcome_message": ""})
	ctx := context.Background()

	// Should not error even when no hub
	err := a.HandleClientConnect(ctx, "session-1", map[string]string{})
	assert.NoError(t, err)
}

func TestAdapter_HandleClientDisconnect(t *testing.T) {
	a := NewAdapter()
	ctx := context.Background()

	err := a.HandleClientDisconnect(ctx, "session-1")
	assert.NoError(t, err)
}

func TestAdapter_SetHandlers(t *testing.T) {
	a := NewAdapter()

	called := false
	a.SetMessageHandler(func(ctx context.Context, msg *plugin.InboundMessage) error {
		called = true
		return nil
	})

	a.SetStatusHandler(func(ctx context.Context, status *plugin.StatusCallback) error {
		return nil
	})

	// Verify handler is set by calling HandleInboundMessage
	a.HandleInboundMessage(context.Background(), "s1", &MessagePayload{Content: "test"})
	assert.True(t, called)
}

func TestConvertAttachments(t *testing.T) {
	t.Run("nil input", func(t *testing.T) {
		result := convertAttachments(nil)
		assert.Empty(t, result)
	})

	t.Run("with attachments", func(t *testing.T) {
		input := []*plugin.Attachment{
			{Type: "image", URL: "https://example.com/a.jpg", Filename: "a.jpg", MimeType: "image/jpeg", SizeBytes: 1024},
			{Type: "document", URL: "https://example.com/b.pdf", Filename: "b.pdf", MimeType: "application/pdf", SizeBytes: 2048},
		}
		result := convertAttachments(input)
		require.Len(t, result, 2)
		assert.Equal(t, "image", result[0].Type)
		assert.Equal(t, "https://example.com/a.jpg", result[0].URL)
		assert.Equal(t, "a.jpg", result[0].Filename)
		assert.Equal(t, int64(1024), result[0].SizeBytes)
		assert.Equal(t, "document", result[1].Type)
	})
}

func TestGetOrDefault(t *testing.T) {
	config := map[string]string{"key1": "value1", "key2": ""}

	assert.Equal(t, "value1", getOrDefault(config, "key1", "default"))
	assert.Equal(t, "default", getOrDefault(config, "key2", "default"))
	assert.Equal(t, "default", getOrDefault(config, "missing", "default"))
}
