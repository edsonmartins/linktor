package facebook

import (
	"context"
	"testing"

	"github.com/msgfy/linktor/pkg/plugin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewAdapter(t *testing.T) {
	a := NewAdapter()
	require.NotNil(t, a)

	info := a.GetChannelInfo()
	assert.Equal(t, plugin.ChannelTypeFacebook, info.Type)
	assert.Equal(t, "Facebook Messenger", info.Name)

	caps := info.Capabilities
	require.NotNil(t, caps)
	assert.True(t, caps.SupportsMedia)
	assert.True(t, caps.SupportsLocation)
	assert.True(t, caps.SupportsTemplates)
	assert.True(t, caps.SupportsInteractive)
	assert.True(t, caps.SupportsReadReceipts)
	assert.True(t, caps.SupportsTypingIndicator)
	assert.False(t, caps.SupportsReactions)
	assert.False(t, caps.SupportsReplies)
	assert.False(t, caps.SupportsForwarding)
	assert.Equal(t, 2000, caps.MaxMessageLength)
	assert.Equal(t, int64(25*1024*1024), caps.MaxMediaSize)
	assert.Equal(t, 1, caps.MaxAttachments)
	assert.Contains(t, caps.SupportedContentTypes, plugin.ContentTypeText)
	assert.Contains(t, caps.SupportedContentTypes, plugin.ContentTypeImage)
	assert.Contains(t, caps.SupportedContentTypes, plugin.ContentTypeVideo)
	assert.Contains(t, caps.SupportedContentTypes, plugin.ContentTypeAudio)
	assert.Contains(t, caps.SupportedContentTypes, plugin.ContentTypeDocument)
	assert.Contains(t, caps.SupportedContentTypes, plugin.ContentTypeLocation)
}

func TestAdapter_Initialize(t *testing.T) {
	t.Run("valid config", func(t *testing.T) {
		a := NewAdapter()
		err := a.Initialize(map[string]string{
			"app_id":            "app-123",
			"app_secret":        "secret",
			"page_id":           "page-456",
			"page_access_token": "token-abc",
			"verify_token":      "verify-123",
		})
		assert.NoError(t, err)
		assert.Equal(t, "app-123", a.config.AppID)
		assert.Equal(t, "secret", a.config.AppSecret)
		assert.Equal(t, "page-456", a.config.PageID)
		assert.Equal(t, "token-abc", a.config.PageAccessToken)
		assert.Equal(t, "verify-123", a.config.VerifyToken)
	})

	t.Run("minimal config", func(t *testing.T) {
		a := NewAdapter()
		err := a.Initialize(map[string]string{
			"page_id":           "page-1",
			"page_access_token": "tok",
		})
		assert.NoError(t, err)
	})

	t.Run("with instagram_id", func(t *testing.T) {
		a := NewAdapter()
		err := a.Initialize(map[string]string{
			"instagram_id":      "ig-789",
			"page_id":           "page-1",
			"page_access_token": "tok",
		})
		assert.NoError(t, err)
		assert.Equal(t, "ig-789", a.config.InstagramID)
	})
}

func TestFacebookConfig_Validate(t *testing.T) {
	t.Run("valid", func(t *testing.T) {
		cfg := &FacebookConfig{PageID: "p1", PageAccessToken: "tok"}
		assert.NoError(t, cfg.Validate())
	})

	t.Run("missing page_id", func(t *testing.T) {
		cfg := &FacebookConfig{PageAccessToken: "tok"}
		err := cfg.Validate()
		require.Error(t, err)
		assert.Equal(t, ErrMissingPageID, err)
		assert.Contains(t, err.Error(), "page_id")
	})

	t.Run("missing page_access_token", func(t *testing.T) {
		cfg := &FacebookConfig{PageID: "p1"}
		err := cfg.Validate()
		require.Error(t, err)
		assert.Equal(t, ErrMissingPageAccessToken, err)
		assert.Contains(t, err.Error(), "page_access_token")
	})
}

func TestAdapter_SendMessage_NotConnected(t *testing.T) {
	a := NewAdapter()
	a.Initialize(map[string]string{})

	result, err := a.SendMessage(nil, &plugin.OutboundMessage{
		RecipientID: "user-1",
		Content:     "hello",
		ContentType: plugin.ContentTypeText,
	})
	require.NoError(t, err)
	assert.False(t, result.Success)
	assert.Equal(t, plugin.MessageStatusFailed, result.Status)
	assert.Equal(t, "adapter not connected", result.Error)
}

func TestAdapter_SendTypingIndicator_NotConnected(t *testing.T) {
	a := NewAdapter()
	a.Initialize(map[string]string{})

	err := a.SendTypingIndicator(nil, &plugin.TypingIndicator{
		RecipientID: "user-1",
		IsTyping:    true,
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not connected")
}

func TestAdapter_SendReadReceipt_NotConnected(t *testing.T) {
	a := NewAdapter()
	a.Initialize(map[string]string{})

	err := a.SendReadReceipt(nil, &plugin.ReadReceipt{
		RecipientID: "user-1",
		MessageID:   "msg-1",
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not connected")
}

func TestAdapter_GetWebhookPath(t *testing.T) {
	a := NewAdapter()
	assert.Equal(t, "/webhooks/facebook", a.GetWebhookPath())
}

func TestAdapter_ValidateWebhook(t *testing.T) {
	t.Run("no app secret skips validation", func(t *testing.T) {
		a := NewAdapter()
		a.config = &FacebookConfig{}
		assert.True(t, a.ValidateWebhook(map[string]string{}, nil))
	})
}

func TestAdapter_SetHandlers(t *testing.T) {
	a := NewAdapter()

	a.SetMessageHandler(func(ctx context.Context, msg *plugin.InboundMessage) error {
		return nil
	})
	a.SetStatusHandler(func(ctx context.Context, status *plugin.StatusCallback) error {
		return nil
	})
	// No panic means success — handlers are stored internally
}

func TestConfigError(t *testing.T) {
	err := &ConfigError{Field: "test", Message: "test error"}
	assert.Equal(t, "test error", err.Error())
}

func TestContentTypeFromFB(t *testing.T) {
	tests := []struct {
		input    string
		expected plugin.ContentType
	}{
		{"image", plugin.ContentTypeImage},
		{"video", plugin.ContentTypeVideo},
		{"audio", plugin.ContentTypeAudio},
		{"file", plugin.ContentTypeDocument},
		{"location", plugin.ContentTypeLocation},
		{"unknown", plugin.ContentTypeText},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			assert.Equal(t, tt.expected, contentTypeFromFB(tt.input))
		})
	}
}

func TestGetAttachmentType(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"image", "image"},
		{"video", "video"},
		{"audio", "audio"},
		{"file", "document"},
		{"location", "location"},
		{"fallback", "link"},
		{"other", "file"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			assert.Equal(t, tt.expected, GetAttachmentType(tt.input))
		})
	}
}
