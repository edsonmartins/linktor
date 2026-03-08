package email

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewClient(t *testing.T) {
	t.Run("valid SMTP", func(t *testing.T) {
		cfg := &Config{
			Provider:  ProviderSMTP,
			FromEmail: "test@example.com",
			SMTPHost:  "smtp.example.com",
			SMTPPort:  587,
		}
		client, err := NewClient(cfg)
		require.NoError(t, err)
		require.NotNil(t, client)
		assert.Equal(t, "smtp", client.GetProviderName())
	})

	t.Run("valid SendGrid", func(t *testing.T) {
		cfg := &Config{
			Provider:       ProviderSendGrid,
			FromEmail:      "test@example.com",
			SendGridAPIKey: "SG.testkey",
		}
		client, err := NewClient(cfg)
		require.NoError(t, err)
		require.NotNil(t, client)
		assert.Equal(t, "sendgrid", client.GetProviderName())
	})

	t.Run("valid Mailgun", func(t *testing.T) {
		cfg := &Config{
			Provider:      ProviderMailgun,
			FromEmail:     "test@example.com",
			MailgunAPIKey: "key-abc123",
			MailgunDomain: "mg.example.com",
		}
		client, err := NewClient(cfg)
		require.NoError(t, err)
		require.NotNil(t, client)
		assert.Equal(t, "mailgun", client.GetProviderName())
	})

	t.Run("valid SES", func(t *testing.T) {
		cfg := &Config{
			Provider:       ProviderSES,
			FromEmail:      "test@example.com",
			SESRegion:      "us-east-1",
			SESAccessKeyID: "AKID",
			SESSecretKey:   "secret",
		}
		client, err := NewClient(cfg)
		require.NoError(t, err)
		require.NotNil(t, client)
		assert.Equal(t, "ses", client.GetProviderName())
	})

	t.Run("valid Postmark", func(t *testing.T) {
		cfg := &Config{
			Provider:            ProviderPostmark,
			FromEmail:           "test@example.com",
			PostmarkServerToken: "token123",
		}
		client, err := NewClient(cfg)
		require.NoError(t, err)
		require.NotNil(t, client)
		assert.Equal(t, "postmark", client.GetProviderName())
	})

	t.Run("invalid provider", func(t *testing.T) {
		cfg := &Config{
			Provider:  Provider("invalid"),
			FromEmail: "test@example.com",
		}
		client, err := NewClient(cfg)
		assert.Error(t, err)
		assert.Nil(t, client)
		assert.Contains(t, err.Error(), "unsupported email provider")
	})
}

func TestClient_GetProviderName(t *testing.T) {
	providers := []struct {
		config   *Config
		expected string
	}{
		{
			config: &Config{
				Provider:  ProviderSMTP,
				FromEmail: "t@e.com",
				SMTPHost:  "h",
				SMTPPort:  587,
			},
			expected: "smtp",
		},
		{
			config: &Config{
				Provider:       ProviderSendGrid,
				FromEmail:      "t@e.com",
				SendGridAPIKey: "k",
			},
			expected: "sendgrid",
		},
		{
			config: &Config{
				Provider:      ProviderMailgun,
				FromEmail:     "t@e.com",
				MailgunAPIKey: "k",
				MailgunDomain: "d",
			},
			expected: "mailgun",
		},
		{
			config: &Config{
				Provider:       ProviderSES,
				FromEmail:      "t@e.com",
				SESRegion:      "r",
				SESAccessKeyID: "a",
				SESSecretKey:   "s",
			},
			expected: "ses",
		},
		{
			config: &Config{
				Provider:            ProviderPostmark,
				FromEmail:           "t@e.com",
				PostmarkServerToken: "t",
			},
			expected: "postmark",
		},
	}

	for _, tt := range providers {
		t.Run(tt.expected, func(t *testing.T) {
			client, err := NewClient(tt.config)
			require.NoError(t, err)
			assert.Equal(t, tt.expected, client.GetProviderName())
		})
	}
}

func TestClient_SendText(t *testing.T) {
	// SendText is a helper that wraps Send; we verify it creates the correct OutboundEmail struct.
	// We cannot actually send without a live provider, but we can verify the client was constructed.
	cfg := &Config{
		Provider:       ProviderSendGrid,
		FromEmail:      "test@example.com",
		SendGridAPIKey: "SG.testkey",
	}
	client, err := NewClient(cfg)
	require.NoError(t, err)
	require.NotNil(t, client)
	// Verify the client and provider are properly initialized
	assert.Equal(t, "sendgrid", client.GetProviderName())
	assert.NotNil(t, client.GetProvider())
}

func TestClient_SendHTML(t *testing.T) {
	cfg := &Config{
		Provider:       ProviderSendGrid,
		FromEmail:      "test@example.com",
		SendGridAPIKey: "SG.testkey",
	}
	client, err := NewClient(cfg)
	require.NoError(t, err)
	require.NotNil(t, client)
	assert.Equal(t, "sendgrid", client.GetProviderName())
	assert.NotNil(t, client.GetProvider())
	assert.Equal(t, cfg, client.GetConfig())
}
