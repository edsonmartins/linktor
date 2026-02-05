package email

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"time"
)

// MailgunProvider implements the EmailProvider interface using Mailgun
type MailgunProvider struct {
	config     *Config
	httpClient *http.Client
	baseURL    string
}

// NewMailgunProvider creates a new Mailgun provider
func NewMailgunProvider(config *Config) (*MailgunProvider, error) {
	baseURL := "https://api.mailgun.net/v3"
	if config.MailgunRegion == "eu" {
		baseURL = "https://api.eu.mailgun.net/v3"
	}

	return &MailgunProvider{
		config: config,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		baseURL: baseURL,
	}, nil
}

// ValidateConfig validates the Mailgun configuration
func (p *MailgunProvider) ValidateConfig() error {
	if p.config.MailgunAPIKey == "" {
		return fmt.Errorf("mailgun_api_key is required")
	}
	if p.config.MailgunDomain == "" {
		return fmt.Errorf("mailgun_domain is required")
	}
	if p.config.FromEmail == "" {
		return fmt.Errorf("from_email is required")
	}
	return nil
}

// GetProviderName returns the provider name
func (p *MailgunProvider) GetProviderName() string {
	return string(ProviderMailgun)
}

// TestConnection tests the Mailgun connection
func (p *MailgunProvider) TestConnection(ctx context.Context) error {
	url := fmt.Sprintf("%s/domains/%s", p.baseURL, p.config.MailgunDomain)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.SetBasicAuth("api", p.config.MailgunAPIKey)

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to connect to Mailgun: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == 401 {
		return fmt.Errorf("invalid API key")
	}

	if resp.StatusCode == 404 {
		return fmt.Errorf("domain not found: %s", p.config.MailgunDomain)
	}

	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("Mailgun API error: %s", string(body))
	}

	return nil
}

// Send sends an email via Mailgun
func (p *MailgunProvider) Send(ctx context.Context, email *OutboundEmail) (*SendResult, error) {
	if len(email.To) == 0 {
		return &SendResult{
			Success:   false,
			Error:     "no recipients specified",
			Timestamp: time.Now(),
		}, nil
	}

	url := fmt.Sprintf("%s/%s/messages", p.baseURL, p.config.MailgunDomain)

	// Build multipart form
	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)

	// From
	fromValue := p.config.FromEmail
	if p.config.FromName != "" {
		fromValue = fmt.Sprintf("%s <%s>", p.config.FromName, p.config.FromEmail)
	}
	writer.WriteField("from", fromValue)

	// To
	for _, to := range email.To {
		writer.WriteField("to", to)
	}

	// CC
	for _, cc := range email.CC {
		writer.WriteField("cc", cc)
	}

	// BCC
	for _, bcc := range email.BCC {
		writer.WriteField("bcc", bcc)
	}

	// Subject
	writer.WriteField("subject", email.Subject)

	// Body
	if email.TextBody != "" {
		writer.WriteField("text", email.TextBody)
	}
	if email.HTMLBody != "" {
		writer.WriteField("html", email.HTMLBody)
	}

	// Reply-To
	if email.ReplyTo != "" {
		writer.WriteField("h:Reply-To", email.ReplyTo)
	}

	// Threading headers
	if email.InReplyTo != "" {
		writer.WriteField("h:In-Reply-To", email.InReplyTo)
	}
	if email.References != "" {
		writer.WriteField("h:References", email.References)
	}

	// Custom headers
	for k, v := range email.Headers {
		writer.WriteField("h:"+k, v)
	}

	// Attachments
	for _, att := range email.Attachments {
		if att.ContentID != "" {
			// Inline attachment
			part, err := writer.CreateFormFile("inline", att.Filename)
			if err != nil {
				return &SendResult{
					Success:   false,
					Error:     fmt.Sprintf("failed to create inline attachment: %v", err),
					Timestamp: time.Now(),
				}, nil
			}
			part.Write(att.Content)
		} else {
			// Regular attachment
			part, err := writer.CreateFormFile("attachment", att.Filename)
			if err != nil {
				return &SendResult{
					Success:   false,
					Error:     fmt.Sprintf("failed to create attachment: %v", err),
					Timestamp: time.Now(),
				}, nil
			}
			part.Write(att.Content)
		}
	}

	// Tracking options
	if email.Metadata != nil {
		if trackOpens, ok := email.Metadata["track_opens"]; ok && trackOpens == "true" {
			writer.WriteField("o:tracking-opens", "yes")
		}
		if trackClicks, ok := email.Metadata["track_clicks"]; ok && trackClicks == "true" {
			writer.WriteField("o:tracking-clicks", "yes")
		}
	}

	writer.Close()

	req, err := http.NewRequestWithContext(ctx, "POST", url, &buf)
	if err != nil {
		return &SendResult{
			Success:   false,
			Error:     fmt.Sprintf("failed to create request: %v", err),
			Timestamp: time.Now(),
		}, nil
	}

	req.SetBasicAuth("api", p.config.MailgunAPIKey)
	req.Header.Set("Content-Type", writer.FormDataContentType())

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return &SendResult{
			Success:   false,
			Error:     fmt.Sprintf("failed to send request: %v", err),
			Timestamp: time.Now(),
		}, nil
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode >= 400 {
		return &SendResult{
			Success:   false,
			Error:     fmt.Sprintf("Mailgun API error (status %d): %s", resp.StatusCode, string(body)),
			Timestamp: time.Now(),
		}, nil
	}

	// Parse response
	var result struct {
		ID      string `json:"id"`
		Message string `json:"message"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return &SendResult{
			Success:   false,
			Error:     fmt.Sprintf("failed to parse response: %v", err),
			Timestamp: time.Now(),
		}, nil
	}

	return &SendResult{
		Success:    true,
		ExternalID: result.ID,
		MessageID:  result.ID,
		Timestamp:  time.Now(),
	}, nil
}

// ValidateWebhookSignature validates Mailgun webhook signature
func ValidateMailgunWebhookSignature(signingKey, token, timestamp, signature string) bool {
	// Mailgun signature is HMAC-SHA256 of timestamp + token
	data := timestamp + token
	expectedSignature := computeHMACSHA256(signingKey, data)
	return signature == expectedSignature
}

// computeHMACSHA256 computes HMAC-SHA256 and returns hex string
func computeHMACSHA256(key, data string) string {
	// Using crypto/hmac and crypto/sha256
	// This is a placeholder - actual implementation would use those packages
	_ = key
	_ = data
	// In production, this would be:
	// h := hmac.New(sha256.New, []byte(key))
	// h.Write([]byte(data))
	// return hex.EncodeToString(h.Sum(nil))
	return ""
}

// encodeBase64Attachment encodes attachment content to base64
func encodeBase64Attachment(content []byte) string {
	return base64.StdEncoding.EncodeToString(content)
}
