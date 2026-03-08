package email

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewMailgunProvider(t *testing.T) {
	t.Run("US region", func(t *testing.T) {
		cfg := &Config{
			Provider:      ProviderMailgun,
			FromEmail:     "test@example.com",
			MailgunAPIKey: "key-abc123",
			MailgunDomain: "mg.example.com",
			MailgunRegion: "us",
		}
		p, err := NewMailgunProvider(cfg)
		require.NoError(t, err)
		assert.Equal(t, "https://api.mailgun.net/v3", p.baseURL)
	})

	t.Run("EU region", func(t *testing.T) {
		cfg := &Config{
			Provider:      ProviderMailgun,
			FromEmail:     "test@example.com",
			MailgunAPIKey: "key-abc123",
			MailgunDomain: "mg.example.com",
			MailgunRegion: "eu",
		}
		p, err := NewMailgunProvider(cfg)
		require.NoError(t, err)
		assert.Equal(t, "https://api.eu.mailgun.net/v3", p.baseURL)
	})

	t.Run("default region is US", func(t *testing.T) {
		cfg := &Config{
			Provider:      ProviderMailgun,
			FromEmail:     "test@example.com",
			MailgunAPIKey: "key-abc123",
			MailgunDomain: "mg.example.com",
		}
		p, err := NewMailgunProvider(cfg)
		require.NoError(t, err)
		assert.Equal(t, "https://api.mailgun.net/v3", p.baseURL)
	})
}

func TestMailgunProvider_ValidateConfig(t *testing.T) {
	t.Run("valid", func(t *testing.T) {
		p, _ := NewMailgunProvider(&Config{
			Provider:      ProviderMailgun,
			FromEmail:     "test@example.com",
			MailgunAPIKey: "key-abc123",
			MailgunDomain: "mg.example.com",
		})
		assert.NoError(t, p.ValidateConfig())
	})

	t.Run("missing api_key", func(t *testing.T) {
		p, _ := NewMailgunProvider(&Config{
			Provider:      ProviderMailgun,
			FromEmail:     "test@example.com",
			MailgunDomain: "mg.example.com",
		})
		err := p.ValidateConfig()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "mailgun_api_key is required")
	})

	t.Run("missing domain", func(t *testing.T) {
		p, _ := NewMailgunProvider(&Config{
			Provider:      ProviderMailgun,
			FromEmail:     "test@example.com",
			MailgunAPIKey: "key-abc123",
		})
		err := p.ValidateConfig()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "mailgun_domain is required")
	})

	t.Run("missing from_email", func(t *testing.T) {
		p, _ := NewMailgunProvider(&Config{
			Provider:      ProviderMailgun,
			MailgunAPIKey: "key-abc123",
			MailgunDomain: "mg.example.com",
		})
		err := p.ValidateConfig()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "from_email is required")
	})
}

func TestMailgunProvider_GetProviderName(t *testing.T) {
	p, _ := NewMailgunProvider(&Config{})
	assert.Equal(t, "mailgun", p.GetProviderName())
}

func TestValidateMailgunWebhookSignature(t *testing.T) {
	signingKey := "my-signing-key"
	timestamp := "1609459200"
	token := "random-token-value"

	h := hmac.New(sha256.New, []byte(signingKey))
	h.Write([]byte(timestamp + token))
	validSignature := hex.EncodeToString(h.Sum(nil))

	t.Run("valid signature", func(t *testing.T) {
		assert.True(t, ValidateMailgunWebhookSignature(signingKey, token, timestamp, validSignature))
	})

	t.Run("invalid signature", func(t *testing.T) {
		assert.False(t, ValidateMailgunWebhookSignature(signingKey, token, timestamp, "badsig"))
	})
}

func TestComputeHMACSHA256(t *testing.T) {
	key := "secret-key"
	data := "hello world"

	result := computeHMACSHA256(key, data)

	// Verify against standard library
	h := hmac.New(sha256.New, []byte(key))
	h.Write([]byte(data))
	expected := hex.EncodeToString(h.Sum(nil))

	assert.Equal(t, expected, result)
	assert.Len(t, result, 64) // SHA256 hex is 64 chars
}
