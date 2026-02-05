package email

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

const sendGridAPIURL = "https://api.sendgrid.com/v3/mail/send"

// SendGridProvider implements the EmailProvider interface using SendGrid
type SendGridProvider struct {
	config     *Config
	httpClient *http.Client
}

// NewSendGridProvider creates a new SendGrid provider
func NewSendGridProvider(config *Config) (*SendGridProvider, error) {
	return &SendGridProvider{
		config: config,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}, nil
}

// ValidateConfig validates the SendGrid configuration
func (p *SendGridProvider) ValidateConfig() error {
	if p.config.SendGridAPIKey == "" {
		return fmt.Errorf("sendgrid_api_key is required")
	}
	if p.config.FromEmail == "" {
		return fmt.Errorf("from_email is required")
	}
	return nil
}

// GetProviderName returns the provider name
func (p *SendGridProvider) GetProviderName() string {
	return string(ProviderSendGrid)
}

// TestConnection tests the SendGrid connection by making an API call
func (p *SendGridProvider) TestConnection(ctx context.Context) error {
	// Use the API key to validate by checking scopes
	req, err := http.NewRequestWithContext(ctx, "GET", "https://api.sendgrid.com/v3/scopes", nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+p.config.SendGridAPIKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to connect to SendGrid: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == 401 {
		return fmt.Errorf("invalid API key")
	}

	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("SendGrid API error: %s", string(body))
	}

	return nil
}

// Send sends an email via SendGrid
func (p *SendGridProvider) Send(ctx context.Context, email *OutboundEmail) (*SendResult, error) {
	if len(email.To) == 0 {
		return &SendResult{
			Success:   false,
			Error:     "no recipients specified",
			Timestamp: time.Now(),
		}, nil
	}

	// Build SendGrid request
	payload := p.buildSendGridPayload(email)

	jsonPayload, err := json.Marshal(payload)
	if err != nil {
		return &SendResult{
			Success:   false,
			Error:     fmt.Sprintf("failed to marshal payload: %v", err),
			Timestamp: time.Now(),
		}, nil
	}

	req, err := http.NewRequestWithContext(ctx, "POST", sendGridAPIURL, bytes.NewReader(jsonPayload))
	if err != nil {
		return &SendResult{
			Success:   false,
			Error:     fmt.Sprintf("failed to create request: %v", err),
			Timestamp: time.Now(),
		}, nil
	}

	req.Header.Set("Authorization", "Bearer "+p.config.SendGridAPIKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return &SendResult{
			Success:   false,
			Error:     fmt.Sprintf("failed to send request: %v", err),
			Timestamp: time.Now(),
		}, nil
	}
	defer resp.Body.Close()

	// SendGrid returns 202 Accepted for successful sends
	if resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return &SendResult{
			Success:   false,
			Error:     fmt.Sprintf("SendGrid API error (status %d): %s", resp.StatusCode, string(body)),
			Timestamp: time.Now(),
		}, nil
	}

	// Get message ID from response header
	messageID := resp.Header.Get("X-Message-Id")

	return &SendResult{
		Success:    true,
		ExternalID: messageID,
		Timestamp:  time.Now(),
	}, nil
}

// buildSendGridPayload builds the SendGrid API payload
func (p *SendGridProvider) buildSendGridPayload(email *OutboundEmail) map[string]interface{} {
	// Build personalizations (recipients)
	personalizations := []map[string]interface{}{}

	toList := make([]map[string]string, len(email.To))
	for i, addr := range email.To {
		toList[i] = map[string]string{"email": addr}
	}

	personalization := map[string]interface{}{
		"to": toList,
	}

	if len(email.CC) > 0 {
		ccList := make([]map[string]string, len(email.CC))
		for i, addr := range email.CC {
			ccList[i] = map[string]string{"email": addr}
		}
		personalization["cc"] = ccList
	}

	if len(email.BCC) > 0 {
		bccList := make([]map[string]string, len(email.BCC))
		for i, addr := range email.BCC {
			bccList[i] = map[string]string{"email": addr}
		}
		personalization["bcc"] = bccList
	}

	personalizations = append(personalizations, personalization)

	// Build from
	from := map[string]string{
		"email": p.config.FromEmail,
	}
	if p.config.FromName != "" {
		from["name"] = p.config.FromName
	}

	// Build content
	content := []map[string]string{}
	if email.TextBody != "" {
		content = append(content, map[string]string{
			"type":  "text/plain",
			"value": email.TextBody,
		})
	}
	if email.HTMLBody != "" {
		content = append(content, map[string]string{
			"type":  "text/html",
			"value": email.HTMLBody,
		})
	}

	// Build payload
	payload := map[string]interface{}{
		"personalizations": personalizations,
		"from":             from,
		"subject":          email.Subject,
		"content":          content,
	}

	// Add reply-to
	if email.ReplyTo != "" {
		payload["reply_to"] = map[string]string{"email": email.ReplyTo}
	}

	// Add headers
	if len(email.Headers) > 0 || email.InReplyTo != "" || email.References != "" {
		headers := make(map[string]string)
		for k, v := range email.Headers {
			headers[k] = v
		}
		if email.InReplyTo != "" {
			headers["In-Reply-To"] = email.InReplyTo
		}
		if email.References != "" {
			headers["References"] = email.References
		}
		payload["headers"] = headers
	}

	// Add attachments
	if len(email.Attachments) > 0 {
		attachments := make([]map[string]interface{}, len(email.Attachments))
		for i, att := range email.Attachments {
			attachment := map[string]interface{}{
				"content":  base64.StdEncoding.EncodeToString(att.Content),
				"filename": att.Filename,
				"type":     att.ContentType,
			}
			if att.ContentID != "" {
				attachment["content_id"] = att.ContentID
				attachment["disposition"] = "inline"
			} else {
				attachment["disposition"] = "attachment"
			}
			attachments[i] = attachment
		}
		payload["attachments"] = attachments
	}

	// Add custom tracking settings
	if email.Metadata != nil {
		if trackOpens, ok := email.Metadata["track_opens"]; ok && trackOpens == "true" {
			payload["tracking_settings"] = map[string]interface{}{
				"open_tracking": map[string]bool{"enable": true},
			}
		}
		if trackClicks, ok := email.Metadata["track_clicks"]; ok && trackClicks == "true" {
			if ts, ok := payload["tracking_settings"].(map[string]interface{}); ok {
				ts["click_tracking"] = map[string]bool{"enable": true}
			} else {
				payload["tracking_settings"] = map[string]interface{}{
					"click_tracking": map[string]bool{"enable": true},
				}
			}
		}
	}

	return payload
}
