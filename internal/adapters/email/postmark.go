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

const postmarkAPIURL = "https://api.postmarkapp.com/email"

// PostmarkProvider implements the EmailProvider interface using Postmark
type PostmarkProvider struct {
	config     *Config
	httpClient *http.Client
}

// NewPostmarkProvider creates a new Postmark provider
func NewPostmarkProvider(config *Config) (*PostmarkProvider, error) {
	return &PostmarkProvider{
		config: config,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}, nil
}

// ValidateConfig validates the Postmark configuration
func (p *PostmarkProvider) ValidateConfig() error {
	if p.config.PostmarkServerToken == "" {
		return fmt.Errorf("postmark_server_token is required")
	}
	if p.config.FromEmail == "" {
		return fmt.Errorf("from_email is required")
	}
	return nil
}

// GetProviderName returns the provider name
func (p *PostmarkProvider) GetProviderName() string {
	return string(ProviderPostmark)
}

// TestConnection tests the Postmark connection
func (p *PostmarkProvider) TestConnection(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, "GET", "https://api.postmarkapp.com/server", nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Accept", "application/json")
	req.Header.Set("X-Postmark-Server-Token", p.config.PostmarkServerToken)

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to connect to Postmark: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == 401 {
		return fmt.Errorf("invalid server token")
	}

	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("Postmark API error: %s", string(body))
	}

	return nil
}

// Send sends an email via Postmark
func (p *PostmarkProvider) Send(ctx context.Context, email *OutboundEmail) (*SendResult, error) {
	if len(email.To) == 0 {
		return &SendResult{
			Success:   false,
			Error:     "no recipients specified",
			Timestamp: time.Now(),
		}, nil
	}

	// Build Postmark request
	payload := p.buildPostmarkPayload(email)

	jsonPayload, err := json.Marshal(payload)
	if err != nil {
		return &SendResult{
			Success:   false,
			Error:     fmt.Sprintf("failed to marshal payload: %v", err),
			Timestamp: time.Now(),
		}, nil
	}

	req, err := http.NewRequestWithContext(ctx, "POST", postmarkAPIURL, bytes.NewReader(jsonPayload))
	if err != nil {
		return &SendResult{
			Success:   false,
			Error:     fmt.Sprintf("failed to create request: %v", err),
			Timestamp: time.Now(),
		}, nil
	}

	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Postmark-Server-Token", p.config.PostmarkServerToken)

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
			Error:     fmt.Sprintf("Postmark API error (status %d): %s", resp.StatusCode, string(body)),
			Timestamp: time.Now(),
		}, nil
	}

	// Parse response
	var result struct {
		To          string `json:"To"`
		SubmittedAt string `json:"SubmittedAt"`
		MessageID   string `json:"MessageID"`
		ErrorCode   int    `json:"ErrorCode"`
		Message     string `json:"Message"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return &SendResult{
			Success:   false,
			Error:     fmt.Sprintf("failed to parse response: %v", err),
			Timestamp: time.Now(),
		}, nil
	}

	if result.ErrorCode != 0 {
		return &SendResult{
			Success:   false,
			Error:     fmt.Sprintf("Postmark error %d: %s", result.ErrorCode, result.Message),
			Timestamp: time.Now(),
		}, nil
	}

	return &SendResult{
		Success:    true,
		ExternalID: result.MessageID,
		MessageID:  result.MessageID,
		Timestamp:  time.Now(),
	}, nil
}

// buildPostmarkPayload builds the Postmark API payload
func (p *PostmarkProvider) buildPostmarkPayload(email *OutboundEmail) map[string]interface{} {
	payload := map[string]interface{}{
		"From":    p.buildFromAddress(),
		"To":      joinAddresses(email.To),
		"Subject": email.Subject,
	}

	if len(email.CC) > 0 {
		payload["Cc"] = joinAddresses(email.CC)
	}

	if len(email.BCC) > 0 {
		payload["Bcc"] = joinAddresses(email.BCC)
	}

	if email.TextBody != "" {
		payload["TextBody"] = email.TextBody
	}

	if email.HTMLBody != "" {
		payload["HtmlBody"] = email.HTMLBody
	}

	if email.ReplyTo != "" {
		payload["ReplyTo"] = email.ReplyTo
	}

	// Headers
	headers := []map[string]string{}
	if email.InReplyTo != "" {
		headers = append(headers, map[string]string{
			"Name":  "In-Reply-To",
			"Value": email.InReplyTo,
		})
	}
	if email.References != "" {
		headers = append(headers, map[string]string{
			"Name":  "References",
			"Value": email.References,
		})
	}
	for k, v := range email.Headers {
		headers = append(headers, map[string]string{
			"Name":  k,
			"Value": v,
		})
	}
	if len(headers) > 0 {
		payload["Headers"] = headers
	}

	// Attachments
	if len(email.Attachments) > 0 {
		attachments := make([]map[string]interface{}, len(email.Attachments))
		for i, att := range email.Attachments {
			attachment := map[string]interface{}{
				"Name":        att.Filename,
				"Content":     base64.StdEncoding.EncodeToString(att.Content),
				"ContentType": att.ContentType,
			}
			if att.ContentID != "" {
				attachment["ContentID"] = "cid:" + att.ContentID
			}
			attachments[i] = attachment
		}
		payload["Attachments"] = attachments
	}

	// Tracking
	if email.Metadata != nil {
		if trackOpens, ok := email.Metadata["track_opens"]; ok && trackOpens == "true" {
			payload["TrackOpens"] = true
		}
		if trackLinks, ok := email.Metadata["track_links"]; ok {
			payload["TrackLinks"] = trackLinks // "None", "HtmlAndText", "HtmlOnly", "TextOnly"
		}
		if tag, ok := email.Metadata["tag"]; ok {
			payload["Tag"] = tag
		}
	}

	return payload
}

// buildFromAddress builds the From address
func (p *PostmarkProvider) buildFromAddress() string {
	if p.config.FromName != "" {
		return fmt.Sprintf("%s <%s>", p.config.FromName, p.config.FromEmail)
	}
	return p.config.FromEmail
}

// joinAddresses joins email addresses with comma
func joinAddresses(addresses []string) string {
	result := ""
	for i, addr := range addresses {
		if i > 0 {
			result += ", "
		}
		result += addr
	}
	return result
}
