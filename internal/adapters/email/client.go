package email

import (
	"context"
	"fmt"
)

// EmailProvider defines the interface for email providers
type EmailProvider interface {
	// Send sends an email
	Send(ctx context.Context, email *OutboundEmail) (*SendResult, error)

	// ValidateConfig validates the provider configuration
	ValidateConfig() error

	// GetProviderName returns the provider name
	GetProviderName() string

	// TestConnection tests the connection to the provider
	TestConnection(ctx context.Context) error
}

// Client wraps an email provider with common functionality
type Client struct {
	provider EmailProvider
	config   *Config
}

// NewClient creates a new email client with the appropriate provider
func NewClient(config *Config) (*Client, error) {
	var provider EmailProvider
	var err error

	switch config.Provider {
	case ProviderSMTP:
		provider, err = NewSMTPProvider(config)
	case ProviderSendGrid:
		provider, err = NewSendGridProvider(config)
	case ProviderMailgun:
		provider, err = NewMailgunProvider(config)
	case ProviderSES:
		provider, err = NewSESProvider(config)
	case ProviderPostmark:
		provider, err = NewPostmarkProvider(config)
	default:
		return nil, fmt.Errorf("unsupported email provider: %s", config.Provider)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to create provider %s: %w", config.Provider, err)
	}

	if err := provider.ValidateConfig(); err != nil {
		return nil, fmt.Errorf("invalid configuration for %s: %w", config.Provider, err)
	}

	return &Client{
		provider: provider,
		config:   config,
	}, nil
}

// Send sends an email through the configured provider
func (c *Client) Send(ctx context.Context, email *OutboundEmail) (*SendResult, error) {
	// Set defaults from config
	if email.ReplyTo == "" && c.config.ReplyTo != "" {
		email.ReplyTo = c.config.ReplyTo
	}

	return c.provider.Send(ctx, email)
}

// SendText sends a simple text email
func (c *Client) SendText(ctx context.Context, to, subject, body string) (*SendResult, error) {
	return c.Send(ctx, &OutboundEmail{
		To:       []string{to},
		Subject:  subject,
		TextBody: body,
	})
}

// SendHTML sends an HTML email
func (c *Client) SendHTML(ctx context.Context, to, subject, htmlBody string) (*SendResult, error) {
	return c.Send(ctx, &OutboundEmail{
		To:       []string{to},
		Subject:  subject,
		HTMLBody: htmlBody,
	})
}

// SendWithAttachments sends an email with attachments
func (c *Client) SendWithAttachments(ctx context.Context, to, subject, body string, attachments []*Attachment) (*SendResult, error) {
	return c.Send(ctx, &OutboundEmail{
		To:          []string{to},
		Subject:     subject,
		TextBody:    body,
		Attachments: attachments,
	})
}

// TestConnection tests the connection to the email provider
func (c *Client) TestConnection(ctx context.Context) error {
	return c.provider.TestConnection(ctx)
}

// GetProvider returns the underlying provider
func (c *Client) GetProvider() EmailProvider {
	return c.provider
}

// GetProviderName returns the name of the current provider
func (c *Client) GetProviderName() string {
	return c.provider.GetProviderName()
}

// GetConfig returns the client configuration
func (c *Client) GetConfig() *Config {
	return c.config
}
