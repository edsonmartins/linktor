package email

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		config  Config
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid SMTP",
			config: Config{
				Provider:  ProviderSMTP,
				FromEmail: "test@example.com",
				SMTPHost:  "smtp.example.com",
				SMTPPort:  587,
			},
			wantErr: false,
		},
		{
			name: "valid SendGrid",
			config: Config{
				Provider:       ProviderSendGrid,
				FromEmail:      "test@example.com",
				SendGridAPIKey: "SG.testkey",
			},
			wantErr: false,
		},
		{
			name: "valid Mailgun",
			config: Config{
				Provider:      ProviderMailgun,
				FromEmail:     "test@example.com",
				MailgunAPIKey: "key-abc123",
				MailgunDomain: "mg.example.com",
			},
			wantErr: false,
		},
		{
			name: "valid SES",
			config: Config{
				Provider:       ProviderSES,
				FromEmail:      "test@example.com",
				SESRegion:      "us-east-1",
				SESAccessKeyID: "AKIAIOSFODNN7EXAMPLE",
				SESSecretKey:   "secretkey",
			},
			wantErr: false,
		},
		{
			name: "valid Postmark",
			config: Config{
				Provider:            ProviderPostmark,
				FromEmail:           "test@example.com",
				PostmarkServerToken: "token123",
			},
			wantErr: false,
		},
		{
			name: "missing provider",
			config: Config{
				FromEmail: "test@example.com",
			},
			wantErr: true,
			errMsg:  "provider is required",
		},
		{
			name: "missing from_email",
			config: Config{
				Provider: ProviderSMTP,
				SMTPHost: "smtp.example.com",
				SMTPPort: 587,
			},
			wantErr: true,
			errMsg:  "from_email is required",
		},
		{
			name: "SMTP missing host",
			config: Config{
				Provider:  ProviderSMTP,
				FromEmail: "test@example.com",
				SMTPPort:  587,
			},
			wantErr: true,
			errMsg:  "smtp_host is required",
		},
		{
			name: "SMTP missing port",
			config: Config{
				Provider:  ProviderSMTP,
				FromEmail: "test@example.com",
				SMTPHost:  "smtp.example.com",
			},
			wantErr: true,
			errMsg:  "smtp_port is required",
		},
		{
			name: "SendGrid missing api_key",
			config: Config{
				Provider:  ProviderSendGrid,
				FromEmail: "test@example.com",
			},
			wantErr: true,
			errMsg:  "sendgrid_api_key is required",
		},
		{
			name: "Mailgun missing api_key",
			config: Config{
				Provider:      ProviderMailgun,
				FromEmail:     "test@example.com",
				MailgunDomain: "mg.example.com",
			},
			wantErr: true,
			errMsg:  "mailgun_api_key is required",
		},
		{
			name: "Mailgun missing domain",
			config: Config{
				Provider:      ProviderMailgun,
				FromEmail:     "test@example.com",
				MailgunAPIKey: "key-abc123",
			},
			wantErr: true,
			errMsg:  "mailgun_domain is required",
		},
		{
			name: "SES missing region",
			config: Config{
				Provider:       ProviderSES,
				FromEmail:      "test@example.com",
				SESAccessKeyID: "AKID",
				SESSecretKey:   "secret",
			},
			wantErr: true,
			errMsg:  "ses_region is required",
		},
		{
			name: "SES missing access_key",
			config: Config{
				Provider:     ProviderSES,
				FromEmail:    "test@example.com",
				SESRegion:    "us-east-1",
				SESSecretKey: "secret",
			},
			wantErr: true,
			errMsg:  "ses_access_key_id is required",
		},
		{
			name: "SES missing secret_key",
			config: Config{
				Provider:       ProviderSES,
				FromEmail:      "test@example.com",
				SESRegion:      "us-east-1",
				SESAccessKeyID: "AKID",
			},
			wantErr: true,
			errMsg:  "ses_secret_key is required",
		},
		{
			name: "Postmark missing server_token",
			config: Config{
				Provider:  ProviderPostmark,
				FromEmail: "test@example.com",
			},
			wantErr: true,
			errMsg:  "postmark_server_token is required",
		},
		{
			name: "unsupported provider",
			config: Config{
				Provider:  Provider("unknown"),
				FromEmail: "test@example.com",
			},
			wantErr: true,
			errMsg:  "unsupported provider: unknown",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if tt.wantErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestEmailStatus_Constants(t *testing.T) {
	assert.Equal(t, EmailStatus("queued"), StatusQueued)
	assert.Equal(t, EmailStatus("sent"), StatusSent)
	assert.Equal(t, EmailStatus("delivered"), StatusDelivered)
	assert.Equal(t, EmailStatus("opened"), StatusOpened)
	assert.Equal(t, EmailStatus("clicked"), StatusClicked)
	assert.Equal(t, EmailStatus("bounced"), StatusBounced)
	assert.Equal(t, EmailStatus("spam"), StatusSpam)
	assert.Equal(t, EmailStatus("unsubscribed"), StatusUnsubscribed)
	assert.Equal(t, EmailStatus("failed"), StatusFailed)
}
