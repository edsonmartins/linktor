package instagram

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
	assert.Equal(t, plugin.ChannelTypeInstagram, info.Type)
	assert.Equal(t, "Instagram Direct Messages", info.Name)

	caps := info.Capabilities
	require.NotNil(t, caps)
	assert.True(t, caps.SupportsMedia)
	assert.True(t, caps.SupportsReadReceipts)
	assert.True(t, caps.SupportsReactions)
	assert.False(t, caps.SupportsLocation)
	assert.False(t, caps.SupportsTemplates)
	assert.False(t, caps.SupportsTypingIndicator)
	assert.Equal(t, 1000, caps.MaxMessageLength)
	assert.Equal(t, int64(8*1024*1024), caps.MaxMediaSize)
	assert.Equal(t, 1, caps.MaxAttachments)
}

func TestAdapter_Initialize(t *testing.T) {
	t.Run("valid config", func(t *testing.T) {
		a := NewAdapter()
		err := a.Initialize(map[string]string{
			"instagram_id": "ig-123",
			"access_token": "token-abc",
			"app_secret":   "secret",
			"verify_token": "verify-123",
		})
		assert.NoError(t, err)
	})

	t.Run("with page access token", func(t *testing.T) {
		a := NewAdapter()
		err := a.Initialize(map[string]string{
			"instagram_id":      "ig-123",
			"page_access_token": "page-token",
			"page_id":           "page-456",
		})
		assert.NoError(t, err)
	})

	t.Run("with expires_at", func(t *testing.T) {
		a := NewAdapter()
		expiresAt := time.Now().Add(24 * time.Hour).Format(time.RFC3339)
		err := a.Initialize(map[string]string{
			"instagram_id": "ig-123",
			"access_token": "token",
			"expires_at":   expiresAt,
		})
		assert.NoError(t, err)
	})
}

func TestInstagramConfig_Validate(t *testing.T) {
	t.Run("valid", func(t *testing.T) {
		cfg := &InstagramConfig{InstagramID: "ig-1", AccessToken: "tok"}
		assert.NoError(t, cfg.Validate())
	})

	t.Run("valid with page token", func(t *testing.T) {
		cfg := &InstagramConfig{InstagramID: "ig-1", PageAccessToken: "page-tok"}
		assert.NoError(t, cfg.Validate())
	})

	t.Run("missing instagram_id", func(t *testing.T) {
		cfg := &InstagramConfig{AccessToken: "tok"}
		err := cfg.Validate()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "instagram_id")
	})

	t.Run("missing access tokens", func(t *testing.T) {
		cfg := &InstagramConfig{InstagramID: "ig-1"}
		err := cfg.Validate()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "access_token")
	})
}

func TestInstagramConfig_GetEffectiveAccessToken(t *testing.T) {
	t.Run("prefers access_token", func(t *testing.T) {
		cfg := &InstagramConfig{AccessToken: "main", PageAccessToken: "page"}
		assert.Equal(t, "main", cfg.GetEffectiveAccessToken())
	})

	t.Run("falls back to page_access_token", func(t *testing.T) {
		cfg := &InstagramConfig{PageAccessToken: "page"}
		assert.Equal(t, "page", cfg.GetEffectiveAccessToken())
	})
}

func TestInstagramConfig_IsExpired(t *testing.T) {
	t.Run("not expired", func(t *testing.T) {
		cfg := &InstagramConfig{ExpiresAt: time.Now().Add(time.Hour)}
		assert.False(t, cfg.IsExpired())
	})

	t.Run("expired", func(t *testing.T) {
		cfg := &InstagramConfig{ExpiresAt: time.Now().Add(-time.Hour)}
		assert.True(t, cfg.IsExpired())
	})

	t.Run("zero time (no expiry)", func(t *testing.T) {
		cfg := &InstagramConfig{}
		assert.False(t, cfg.IsExpired())
	})
}

func TestConfigError(t *testing.T) {
	err := &ConfigError{Field: "test", Message: "test error"}
	assert.Equal(t, "test error", err.Error())
}

func TestAdapter_ValidateWebhook(t *testing.T) {
	t.Run("no app secret fails closed", func(t *testing.T) {
		a := NewAdapter()
		a.config = &InstagramConfig{}
		assert.False(t, a.ValidateWebhook(map[string]string{}, nil))
	})

	t.Run("invalid signature rejected", func(t *testing.T) {
		a := NewAdapter()
		a.config = &InstagramConfig{AppSecret: "secret"}
		assert.False(t, a.ValidateWebhook(map[string]string{
			"X-Hub-Signature-256": "sha256=deadbeef",
		}, []byte("{}")))
	})
}

func TestAdapter_SendMessage_MediaWithoutAttachment(t *testing.T) {
	// A media send with no attachments must not panic; it must return a failed
	// SendResult instead of dereferencing a nil API response.
	a := NewAdapter()
	a.Initialize(map[string]string{})
	a.client = &Client{config: &InstagramConfig{InstagramID: "ig-1", AccessToken: "tok"}}

	for _, ct := range []plugin.ContentType{
		plugin.ContentTypeImage,
		plugin.ContentTypeVideo,
		plugin.ContentTypeAudio,
	} {
		t.Run(string(ct), func(t *testing.T) {
			result, err := a.SendMessage(context.Background(), &plugin.OutboundMessage{
				RecipientID: "user-1",
				ContentType: ct,
			})
			require.NoError(t, err)
			require.NotNil(t, result)
			assert.False(t, result.Success)
			assert.Equal(t, plugin.MessageStatusFailed, result.Status)
			assert.Equal(t, "attachment required", result.Error)
		})
	}
}
