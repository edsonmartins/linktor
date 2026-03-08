package email

import (
	"context"
	"testing"

	"github.com/msgfy/linktor/pkg/plugin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewAdapter(t *testing.T) {
	adapter := NewAdapter()
	require.NotNil(t, adapter)

	info := adapter.GetChannelInfo()
	require.NotNil(t, info)
	assert.Equal(t, plugin.ChannelTypeEmail, info.Type)
	assert.Equal(t, "Email", info.Name)
	assert.Equal(t, "1.0.0", info.Version)

	caps := adapter.GetCapabilities()
	require.NotNil(t, caps)
	assert.True(t, caps.SupportsMedia)
	assert.True(t, caps.SupportsTemplates)
	assert.True(t, caps.SupportsReplies)
	assert.False(t, caps.SupportsLocation)
	assert.False(t, caps.SupportsInteractive)
	assert.False(t, caps.SupportsReadReceipts)
	assert.False(t, caps.SupportsTypingIndicator)
	assert.False(t, caps.SupportsReactions)
	assert.False(t, caps.SupportsForwarding)
	assert.Equal(t, 10, caps.MaxAttachments)
	assert.Equal(t, int64(25*1024*1024), caps.MaxMediaSize)
	assert.Equal(t, 0, caps.MaxMessageLength)
	assert.Contains(t, caps.SupportedMediaTypes, "application/pdf")
	assert.Contains(t, caps.SupportedMediaTypes, "image/jpeg")
}

func TestAdapter_Initialize(t *testing.T) {
	tests := []struct {
		name   string
		config map[string]string
		check  func(t *testing.T, a *Adapter)
	}{
		{
			name: "valid SMTP config",
			config: map[string]string{
				"provider":   "smtp",
				"from_email": "test@example.com",
				"from_name":  "Test User",
				"smtp_host":  "smtp.example.com",
				"smtp_port":  "587",
				"smtp_username": "user",
				"smtp_password": "pass",
				"smtp_encryption": "tls",
			},
			check: func(t *testing.T, a *Adapter) {
				cfg := a.GetConfig()
				assert.Equal(t, ProviderSMTP, cfg.Provider)
				assert.Equal(t, "test@example.com", cfg.FromEmail)
				assert.Equal(t, "Test User", cfg.FromName)
				assert.Equal(t, "smtp.example.com", cfg.SMTPHost)
				assert.Equal(t, 587, cfg.SMTPPort)
				assert.Equal(t, "user", cfg.SMTPUsername)
				assert.Equal(t, "pass", cfg.SMTPPassword)
				assert.Equal(t, "tls", cfg.SMTPEncryption)
			},
		},
		{
			name: "valid SendGrid config",
			config: map[string]string{
				"provider":         "sendgrid",
				"from_email":       "test@example.com",
				"sendgrid_api_key": "SG.testkey",
			},
			check: func(t *testing.T, a *Adapter) {
				cfg := a.GetConfig()
				assert.Equal(t, ProviderSendGrid, cfg.Provider)
				assert.Equal(t, "SG.testkey", cfg.SendGridAPIKey)
				assert.Equal(t, "test@example.com", cfg.FromEmail)
			},
		},
		{
			name: "valid Mailgun config",
			config: map[string]string{
				"provider":        "mailgun",
				"from_email":      "test@example.com",
				"mailgun_api_key": "key-abc123",
				"mailgun_domain":  "mg.example.com",
				"mailgun_region":  "eu",
			},
			check: func(t *testing.T, a *Adapter) {
				cfg := a.GetConfig()
				assert.Equal(t, ProviderMailgun, cfg.Provider)
				assert.Equal(t, "key-abc123", cfg.MailgunAPIKey)
				assert.Equal(t, "mg.example.com", cfg.MailgunDomain)
				assert.Equal(t, "eu", cfg.MailgunRegion)
			},
		},
		{
			name: "valid SES config",
			config: map[string]string{
				"provider":          "ses",
				"from_email":        "test@example.com",
				"ses_region":        "us-east-1",
				"ses_access_key_id": "AKIAIOSFODNN7EXAMPLE",
				"ses_secret_key":    "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY",
			},
			check: func(t *testing.T, a *Adapter) {
				cfg := a.GetConfig()
				assert.Equal(t, ProviderSES, cfg.Provider)
				assert.Equal(t, "us-east-1", cfg.SESRegion)
				assert.Equal(t, "AKIAIOSFODNN7EXAMPLE", cfg.SESAccessKeyID)
				assert.Equal(t, "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY", cfg.SESSecretKey)
			},
		},
		{
			name: "valid Postmark config",
			config: map[string]string{
				"provider":               "postmark",
				"from_email":             "test@example.com",
				"postmark_server_token":  "token-abc123",
			},
			check: func(t *testing.T, a *Adapter) {
				cfg := a.GetConfig()
				assert.Equal(t, ProviderPostmark, cfg.Provider)
				assert.Equal(t, "token-abc123", cfg.PostmarkServerToken)
			},
		},
		{
			name: "all fields parsed correctly",
			config: map[string]string{
				"provider":            "smtp",
				"from_email":          "from@example.com",
				"from_name":           "From Name",
				"reply_to":            "reply@example.com",
				"smtp_host":           "smtp.example.com",
				"smtp_port":           "465",
				"smtp_username":       "smtpuser",
				"smtp_password":       "smtppass",
				"smtp_encryption":     "starttls",
				"imap_host":           "imap.example.com",
				"imap_port":           "993",
				"imap_username":       "imapuser",
				"imap_password":       "imappass",
				"imap_folder":         "Custom",
				"imap_poll_interval":  "60",
				"webhook_secret":      "secret123",
			},
			check: func(t *testing.T, a *Adapter) {
				cfg := a.GetConfig()
				assert.Equal(t, "from@example.com", cfg.FromEmail)
				assert.Equal(t, "From Name", cfg.FromName)
				assert.Equal(t, "reply@example.com", cfg.ReplyTo)
				assert.Equal(t, "smtp.example.com", cfg.SMTPHost)
				assert.Equal(t, 465, cfg.SMTPPort)
				assert.Equal(t, "smtpuser", cfg.SMTPUsername)
				assert.Equal(t, "smtppass", cfg.SMTPPassword)
				assert.Equal(t, "starttls", cfg.SMTPEncryption)
				assert.Equal(t, "imap.example.com", cfg.IMAPHost)
				assert.Equal(t, 993, cfg.IMAPPort)
				assert.Equal(t, "imapuser", cfg.IMAPUsername)
				assert.Equal(t, "imappass", cfg.IMAPPassword)
				assert.Equal(t, "Custom", cfg.IMAPFolder)
				assert.Equal(t, 60, cfg.IMAPPollInterval)
				assert.Equal(t, "secret123", cfg.WebhookSecret)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			adapter := NewAdapter()
			err := adapter.Initialize(tt.config)
			require.NoError(t, err)
			tt.check(t, adapter)
		})
	}
}

func TestAdapter_GetWebhookPath(t *testing.T) {
	adapter := NewAdapter()
	assert.Equal(t, "/webhooks/email", adapter.GetWebhookPath())
}

func TestAdapter_ValidateWebhook(t *testing.T) {
	adapter := NewAdapter()
	assert.True(t, adapter.ValidateWebhook(map[string]string{}, []byte("body")))
	assert.True(t, adapter.ValidateWebhook(nil, nil))
}

func TestAdapter_SendMessage_NotConnected(t *testing.T) {
	adapter := NewAdapter()
	err := adapter.Initialize(map[string]string{
		"provider":   "smtp",
		"from_email": "test@example.com",
		"smtp_host":  "smtp.example.com",
		"smtp_port":  "587",
	})
	require.NoError(t, err)

	msg := &plugin.OutboundMessage{
		RecipientID: "recipient@example.com",
		ContentType: plugin.ContentTypeText,
		Content:     "Hello",
	}

	result, err := adapter.SendMessage(context.Background(), msg)
	require.NoError(t, err)
	assert.False(t, result.Success)
	assert.Equal(t, plugin.MessageStatusFailed, result.Status)
	assert.Equal(t, "adapter not connected", result.Error)
}

func TestBuildOutboundEmail(t *testing.T) {
	adapter := NewAdapter()
	_ = adapter.Initialize(map[string]string{
		"provider":   "smtp",
		"from_email": "test@example.com",
		"smtp_host":  "smtp.example.com",
		"smtp_port":  "587",
	})

	t.Run("basic text message with subject in metadata", func(t *testing.T) {
		msg := &plugin.OutboundMessage{
			RecipientID: "recipient@example.com",
			ContentType: plugin.ContentTypeText,
			Content:     "Hello World",
			Metadata: map[string]string{
				"subject": "Test Subject",
			},
		}

		email := adapter.buildOutboundEmail(msg)
		assert.Equal(t, []string{"recipient@example.com"}, email.To)
		assert.Equal(t, "Test Subject", email.Subject)
		assert.Equal(t, "Hello World", email.TextBody)
		assert.Empty(t, email.HTMLBody)
	})

	t.Run("default subject when not in metadata", func(t *testing.T) {
		msg := &plugin.OutboundMessage{
			RecipientID: "recipient@example.com",
			ContentType: plugin.ContentTypeText,
			Content:     "Hello",
			Metadata:    map[string]string{},
		}

		email := adapter.buildOutboundEmail(msg)
		assert.Equal(t, "Message from Linktor", email.Subject)
	})

	t.Run("CC and BCC from metadata", func(t *testing.T) {
		msg := &plugin.OutboundMessage{
			RecipientID: "to@example.com",
			ContentType: plugin.ContentTypeText,
			Content:     "Hello",
			Metadata: map[string]string{
				"subject": "Test",
				"cc":      "cc1@example.com,cc2@example.com",
				"bcc":     "bcc@example.com",
			},
		}

		email := adapter.buildOutboundEmail(msg)
		assert.Equal(t, []string{"cc1@example.com", "cc2@example.com"}, email.CC)
		assert.Equal(t, []string{"bcc@example.com"}, email.BCC)
	})

	t.Run("threading with in_reply_to and references", func(t *testing.T) {
		msg := &plugin.OutboundMessage{
			RecipientID: "to@example.com",
			ContentType: plugin.ContentTypeText,
			Content:     "Reply",
			Metadata: map[string]string{
				"subject":    "Re: Thread",
				"in_reply_to": "<abc123@example.com>",
				"references":  "<abc123@example.com> <def456@example.com>",
			},
		}

		email := adapter.buildOutboundEmail(msg)
		assert.Equal(t, "<abc123@example.com>", email.InReplyTo)
		assert.Equal(t, "<abc123@example.com> <def456@example.com>", email.References)
	})

	t.Run("with attachments", func(t *testing.T) {
		msg := &plugin.OutboundMessage{
			RecipientID: "to@example.com",
			ContentType: plugin.ContentTypeText,
			Content:     "See attached",
			Metadata: map[string]string{
				"subject": "Files",
			},
			Attachments: []*plugin.Attachment{
				{
					Filename: "doc.pdf",
					MimeType: "application/pdf",
					URL:      "https://example.com/doc.pdf",
					Metadata: map[string]string{
						"content": "base64data",
					},
				},
			},
		}

		email := adapter.buildOutboundEmail(msg)
		require.Len(t, email.Attachments, 1)
		assert.Equal(t, "doc.pdf", email.Attachments[0].Filename)
		assert.Equal(t, "application/pdf", email.Attachments[0].ContentType)
		assert.Equal(t, "https://example.com/doc.pdf", email.Attachments[0].URL)
		assert.Equal(t, []byte("base64data"), email.Attachments[0].Content)
	})

	t.Run("HTML body from metadata", func(t *testing.T) {
		msg := &plugin.OutboundMessage{
			RecipientID: "to@example.com",
			ContentType: plugin.ContentTypeText,
			Content:     "Plain text fallback",
			Metadata: map[string]string{
				"subject":   "HTML Email",
				"html_body": "<h1>Hello</h1>",
			},
		}

		email := adapter.buildOutboundEmail(msg)
		assert.Equal(t, "<h1>Hello</h1>", email.HTMLBody)
		assert.Equal(t, "Plain text fallback", email.TextBody)
	})

	t.Run("reply_to in metadata", func(t *testing.T) {
		msg := &plugin.OutboundMessage{
			RecipientID: "to@example.com",
			ContentType: plugin.ContentTypeText,
			Content:     "Hello",
			Metadata: map[string]string{
				"subject":  "Test",
				"reply_to": "noreply@example.com",
			},
		}

		email := adapter.buildOutboundEmail(msg)
		assert.Equal(t, "noreply@example.com", email.ReplyTo)
	})

	t.Run("tracking metadata is copied", func(t *testing.T) {
		msg := &plugin.OutboundMessage{
			RecipientID: "to@example.com",
			ContentType: plugin.ContentTypeText,
			Content:     "Hello",
			Metadata: map[string]string{
				"subject":      "Test",
				"track_opens":  "true",
				"track_clicks": "true",
				"tag":          "campaign1",
				"other_key":    "ignored",
			},
		}

		email := adapter.buildOutboundEmail(msg)
		assert.Equal(t, "true", email.Metadata["track_opens"])
		assert.Equal(t, "true", email.Metadata["track_clicks"])
		assert.Equal(t, "campaign1", email.Metadata["tag"])
		_, exists := email.Metadata["other_key"]
		assert.False(t, exists)
	})
}
