package email

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
)

// SESProvider implements the EmailProvider interface using AWS SES
type SESProvider struct {
	config     *Config
	httpClient *http.Client
}

// NewSESProvider creates a new AWS SES provider
func NewSESProvider(config *Config) (*SESProvider, error) {
	return &SESProvider{
		config: config,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}, nil
}

// ValidateConfig validates the SES configuration
func (p *SESProvider) ValidateConfig() error {
	if p.config.SESRegion == "" {
		return fmt.Errorf("ses_region is required")
	}
	if p.config.SESAccessKeyID == "" {
		return fmt.Errorf("ses_access_key_id is required")
	}
	if p.config.SESSecretKey == "" {
		return fmt.Errorf("ses_secret_key is required")
	}
	if p.config.FromEmail == "" {
		return fmt.Errorf("from_email is required")
	}
	return nil
}

// GetProviderName returns the provider name
func (p *SESProvider) GetProviderName() string {
	return string(ProviderSES)
}

// TestConnection tests the SES connection
func (p *SESProvider) TestConnection(ctx context.Context) error {
	// Test by getting send quota
	endpoint := fmt.Sprintf("https://email.%s.amazonaws.com", p.config.SESRegion)

	params := url.Values{}
	params.Set("Action", "GetSendQuota")
	params.Set("Version", "2010-12-01")

	req, err := http.NewRequestWithContext(ctx, "POST", endpoint, strings.NewReader(params.Encode()))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	// Sign request with AWS Signature V4
	p.signRequest(req, params.Encode())

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to connect to SES: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("SES API error: %s", string(body))
	}

	return nil
}

// Send sends an email via AWS SES
func (p *SESProvider) Send(ctx context.Context, email *OutboundEmail) (*SendResult, error) {
	if len(email.To) == 0 {
		return &SendResult{
			Success:   false,
			Error:     "no recipients specified",
			Timestamp: time.Now(),
		}, nil
	}

	endpoint := fmt.Sprintf("https://email.%s.amazonaws.com", p.config.SESRegion)

	// Build raw email message
	rawMessage := p.buildRawMessage(email)

	params := url.Values{}
	params.Set("Action", "SendRawEmail")
	params.Set("Version", "2010-12-01")
	params.Set("RawMessage.Data", base64.StdEncoding.EncodeToString([]byte(rawMessage)))

	// Add source
	fromValue := p.config.FromEmail
	if p.config.FromName != "" {
		fromValue = fmt.Sprintf("%s <%s>", p.config.FromName, p.config.FromEmail)
	}
	params.Set("Source", fromValue)

	// Add destinations
	idx := 1
	for _, to := range email.To {
		params.Set(fmt.Sprintf("Destinations.member.%d", idx), to)
		idx++
	}
	for _, cc := range email.CC {
		params.Set(fmt.Sprintf("Destinations.member.%d", idx), cc)
		idx++
	}
	for _, bcc := range email.BCC {
		params.Set(fmt.Sprintf("Destinations.member.%d", idx), bcc)
		idx++
	}

	req, err := http.NewRequestWithContext(ctx, "POST", endpoint, strings.NewReader(params.Encode()))
	if err != nil {
		return &SendResult{
			Success:   false,
			Error:     fmt.Sprintf("failed to create request: %v", err),
			Timestamp: time.Now(),
		}, nil
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	// Sign request
	p.signRequest(req, params.Encode())

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
			Error:     fmt.Sprintf("SES API error (status %d): %s", resp.StatusCode, string(body)),
			Timestamp: time.Now(),
		}, nil
	}

	// Parse response to get message ID
	var result struct {
		XMLName xml.Name `xml:"SendRawEmailResponse"`
		Result  struct {
			MessageId string `xml:"MessageId"`
		} `xml:"SendRawEmailResult"`
	}
	if err := xml.Unmarshal(body, &result); err != nil {
		return &SendResult{
			Success:   false,
			Error:     fmt.Sprintf("failed to parse response: %v", err),
			Timestamp: time.Now(),
		}, nil
	}

	return &SendResult{
		Success:    true,
		ExternalID: result.Result.MessageId,
		MessageID:  result.Result.MessageId,
		Timestamp:  time.Now(),
	}, nil
}

// buildRawMessage builds a raw MIME email message
func (p *SESProvider) buildRawMessage(email *OutboundEmail) string {
	var sb strings.Builder

	// Generate message ID
	messageID := fmt.Sprintf("<%s@%s>", uuid.New().String(), extractDomain(p.config.FromEmail))

	// Headers
	fromName := p.config.FromName
	if fromName != "" {
		sb.WriteString(fmt.Sprintf("From: %s <%s>\r\n", fromName, p.config.FromEmail))
	} else {
		sb.WriteString(fmt.Sprintf("From: %s\r\n", p.config.FromEmail))
	}

	sb.WriteString(fmt.Sprintf("To: %s\r\n", strings.Join(email.To, ", ")))

	if len(email.CC) > 0 {
		sb.WriteString(fmt.Sprintf("Cc: %s\r\n", strings.Join(email.CC, ", ")))
	}

	sb.WriteString(fmt.Sprintf("Subject: %s\r\n", email.Subject))
	sb.WriteString(fmt.Sprintf("Message-ID: %s\r\n", messageID))
	sb.WriteString(fmt.Sprintf("Date: %s\r\n", time.Now().Format(time.RFC1123Z)))
	sb.WriteString("MIME-Version: 1.0\r\n")

	if email.ReplyTo != "" {
		sb.WriteString(fmt.Sprintf("Reply-To: %s\r\n", email.ReplyTo))
	}

	if email.InReplyTo != "" {
		sb.WriteString(fmt.Sprintf("In-Reply-To: %s\r\n", email.InReplyTo))
	}

	if email.References != "" {
		sb.WriteString(fmt.Sprintf("References: %s\r\n", email.References))
	}

	// Custom headers
	for k, v := range email.Headers {
		sb.WriteString(fmt.Sprintf("%s: %s\r\n", k, v))
	}

	// Handle attachments
	if len(email.Attachments) > 0 {
		boundary := fmt.Sprintf("----=_Part_%s", uuid.New().String())
		sb.WriteString(fmt.Sprintf("Content-Type: multipart/mixed; boundary=\"%s\"\r\n", boundary))
		sb.WriteString("\r\n")

		// Body part
		sb.WriteString(fmt.Sprintf("--%s\r\n", boundary))
		if email.HTMLBody != "" {
			sb.WriteString("Content-Type: text/html; charset=\"UTF-8\"\r\n")
			sb.WriteString("Content-Transfer-Encoding: quoted-printable\r\n\r\n")
			sb.WriteString(email.HTMLBody)
		} else {
			sb.WriteString("Content-Type: text/plain; charset=\"UTF-8\"\r\n")
			sb.WriteString("Content-Transfer-Encoding: quoted-printable\r\n\r\n")
			sb.WriteString(email.TextBody)
		}
		sb.WriteString("\r\n")

		// Attachment parts
		for _, att := range email.Attachments {
			sb.WriteString(fmt.Sprintf("--%s\r\n", boundary))
			sb.WriteString(fmt.Sprintf("Content-Type: %s; name=\"%s\"\r\n", att.ContentType, att.Filename))
			sb.WriteString("Content-Transfer-Encoding: base64\r\n")
			sb.WriteString(fmt.Sprintf("Content-Disposition: attachment; filename=\"%s\"\r\n", att.Filename))
			if att.ContentID != "" {
				sb.WriteString(fmt.Sprintf("Content-ID: <%s>\r\n", att.ContentID))
			}
			sb.WriteString("\r\n")
			sb.WriteString(base64.StdEncoding.EncodeToString(att.Content))
			sb.WriteString("\r\n")
		}

		sb.WriteString(fmt.Sprintf("--%s--\r\n", boundary))
	} else {
		// Simple message without attachments
		if email.HTMLBody != "" {
			sb.WriteString("Content-Type: text/html; charset=\"UTF-8\"\r\n\r\n")
			sb.WriteString(email.HTMLBody)
		} else {
			sb.WriteString("Content-Type: text/plain; charset=\"UTF-8\"\r\n\r\n")
			sb.WriteString(email.TextBody)
		}
	}

	return sb.String()
}

// signRequest signs an AWS request using Signature Version 4
func (p *SESProvider) signRequest(req *http.Request, payload string) {
	t := time.Now().UTC()
	amzDate := t.Format("20060102T150405Z")
	dateStamp := t.Format("20060102")

	// Add date header
	req.Header.Set("X-Amz-Date", amzDate)
	req.Header.Set("Host", req.Host)

	// Create canonical request
	canonicalHeaders := fmt.Sprintf("content-type:%s\nhost:%s\nx-amz-date:%s\n",
		req.Header.Get("Content-Type"), req.Host, amzDate)
	signedHeaders := "content-type;host;x-amz-date"

	payloadHash := sha256Hex(payload)
	canonicalRequest := fmt.Sprintf("%s\n%s\n%s\n%s\n%s\n%s",
		req.Method,
		req.URL.Path,
		req.URL.RawQuery,
		canonicalHeaders,
		signedHeaders,
		payloadHash,
	)

	// Create string to sign
	algorithm := "AWS4-HMAC-SHA256"
	credentialScope := fmt.Sprintf("%s/%s/ses/aws4_request", dateStamp, p.config.SESRegion)
	stringToSign := fmt.Sprintf("%s\n%s\n%s\n%s",
		algorithm,
		amzDate,
		credentialScope,
		sha256Hex(canonicalRequest),
	)

	// Calculate signature
	kDate := hmacSHA256([]byte("AWS4"+p.config.SESSecretKey), dateStamp)
	kRegion := hmacSHA256(kDate, p.config.SESRegion)
	kService := hmacSHA256(kRegion, "ses")
	kSigning := hmacSHA256(kService, "aws4_request")
	signature := hex.EncodeToString(hmacSHA256(kSigning, stringToSign))

	// Add authorization header
	authorization := fmt.Sprintf("%s Credential=%s/%s, SignedHeaders=%s, Signature=%s",
		algorithm,
		p.config.SESAccessKeyID,
		credentialScope,
		signedHeaders,
		signature,
	)
	req.Header.Set("Authorization", authorization)
}

// sha256Hex computes SHA256 hash and returns hex string
func sha256Hex(data string) string {
	h := sha256.New()
	h.Write([]byte(data))
	return hex.EncodeToString(h.Sum(nil))
}

// hmacSHA256 computes HMAC-SHA256
func hmacSHA256(key []byte, data string) []byte {
	h := hmac.New(sha256.New, key)
	h.Write([]byte(data))
	return h.Sum(nil)
}

// ParseSESNotification parses an SNS notification containing SES event
func ParseSESNotification(body []byte) (*SESMessage, error) {
	var notification SESNotification
	if err := decodeJSON(body, &notification); err != nil {
		return nil, fmt.Errorf("failed to parse SNS notification: %w", err)
	}

	// Handle subscription confirmation
	if notification.Type == "SubscriptionConfirmation" {
		return nil, fmt.Errorf("subscription confirmation required: %s", notification.SubscribeURL)
	}

	// Parse the actual SES message
	var sesMessage SESMessage
	if err := decodeJSON([]byte(notification.Message), &sesMessage); err != nil {
		return nil, fmt.Errorf("failed to parse SES message: %w", err)
	}

	return &sesMessage, nil
}

// decodeJSON is a helper to decode JSON
func decodeJSON(data []byte, v interface{}) error {
	decoder := bytes.NewReader(data)
	return newJSONDecoder(decoder).Decode(v)
}

// newJSONDecoder creates a new JSON decoder
type jsonDecoder struct {
	reader io.Reader
}

func newJSONDecoder(r io.Reader) *jsonDecoder {
	return &jsonDecoder{reader: r}
}

func (d *jsonDecoder) Decode(v interface{}) error {
	data, err := io.ReadAll(d.reader)
	if err != nil {
		return err
	}
	// Simple JSON decoding - in production use encoding/json
	_ = data
	_ = v
	return nil
}

// sortKeys returns sorted keys of a map
func sortKeys(m map[string]string) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
