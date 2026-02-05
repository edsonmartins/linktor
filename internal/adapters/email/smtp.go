package email

import (
	"context"
	"crypto/tls"
	"fmt"
	"net/smtp"
	"strings"
	"time"

	"github.com/google/uuid"
)

// SMTPProvider implements the EmailProvider interface using SMTP
type SMTPProvider struct {
	config *Config
}

// NewSMTPProvider creates a new SMTP provider
func NewSMTPProvider(config *Config) (*SMTPProvider, error) {
	return &SMTPProvider{config: config}, nil
}

// ValidateConfig validates the SMTP configuration
func (p *SMTPProvider) ValidateConfig() error {
	if p.config.SMTPHost == "" {
		return fmt.Errorf("smtp_host is required")
	}
	if p.config.SMTPPort == 0 {
		return fmt.Errorf("smtp_port is required")
	}
	if p.config.FromEmail == "" {
		return fmt.Errorf("from_email is required")
	}
	return nil
}

// GetProviderName returns the provider name
func (p *SMTPProvider) GetProviderName() string {
	return string(ProviderSMTP)
}

// TestConnection tests the SMTP connection
func (p *SMTPProvider) TestConnection(ctx context.Context) error {
	addr := fmt.Sprintf("%s:%d", p.config.SMTPHost, p.config.SMTPPort)

	var client *smtp.Client
	var err error

	switch p.config.SMTPEncryption {
	case "tls":
		conn, err := tls.Dial("tcp", addr, &tls.Config{ServerName: p.config.SMTPHost})
		if err != nil {
			return fmt.Errorf("failed to connect with TLS: %w", err)
		}
		defer conn.Close()

		client, err = smtp.NewClient(conn, p.config.SMTPHost)
		if err != nil {
			return fmt.Errorf("failed to create SMTP client: %w", err)
		}
	default:
		client, err = smtp.Dial(addr)
		if err != nil {
			return fmt.Errorf("failed to connect to SMTP server: %w", err)
		}
	}

	defer client.Close()

	// Start TLS if using STARTTLS
	if p.config.SMTPEncryption == "starttls" {
		if err := client.StartTLS(&tls.Config{ServerName: p.config.SMTPHost}); err != nil {
			return fmt.Errorf("failed to start TLS: %w", err)
		}
	}

	// Authenticate if credentials are provided
	if p.config.SMTPUsername != "" && p.config.SMTPPassword != "" {
		auth := smtp.PlainAuth("", p.config.SMTPUsername, p.config.SMTPPassword, p.config.SMTPHost)
		if err := client.Auth(auth); err != nil {
			return fmt.Errorf("authentication failed: %w", err)
		}
	}

	return nil
}

// Send sends an email via SMTP
func (p *SMTPProvider) Send(ctx context.Context, email *OutboundEmail) (*SendResult, error) {
	if len(email.To) == 0 {
		return &SendResult{
			Success:   false,
			Error:     "no recipients specified",
			Timestamp: time.Now(),
		}, nil
	}

	// Generate message ID
	messageID := fmt.Sprintf("<%s@%s>", uuid.New().String(), extractDomain(p.config.FromEmail))

	// Build email message
	msg := p.buildMessage(email, messageID)

	// Get all recipients
	recipients := append([]string{}, email.To...)
	recipients = append(recipients, email.CC...)
	recipients = append(recipients, email.BCC...)

	// Send email
	addr := fmt.Sprintf("%s:%d", p.config.SMTPHost, p.config.SMTPPort)

	var auth smtp.Auth
	if p.config.SMTPUsername != "" && p.config.SMTPPassword != "" {
		auth = smtp.PlainAuth("", p.config.SMTPUsername, p.config.SMTPPassword, p.config.SMTPHost)
	}

	var err error
	switch p.config.SMTPEncryption {
	case "tls":
		err = p.sendWithTLS(addr, auth, p.config.FromEmail, recipients, msg)
	case "starttls":
		err = p.sendWithSTARTTLS(addr, auth, p.config.FromEmail, recipients, msg)
	default:
		err = smtp.SendMail(addr, auth, p.config.FromEmail, recipients, msg)
	}

	if err != nil {
		return &SendResult{
			Success:   false,
			Error:     err.Error(),
			Timestamp: time.Now(),
		}, nil
	}

	return &SendResult{
		Success:   true,
		MessageID: messageID,
		Timestamp: time.Now(),
	}, nil
}

// buildMessage builds the MIME message
func (p *SMTPProvider) buildMessage(email *OutboundEmail, messageID string) []byte {
	var sb strings.Builder

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
			// Base64 encode the content
			sb.WriteString(encodeBase64(att.Content))
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

	return []byte(sb.String())
}

// sendWithTLS sends email using direct TLS connection
func (p *SMTPProvider) sendWithTLS(addr string, auth smtp.Auth, from string, to []string, msg []byte) error {
	conn, err := tls.Dial("tcp", addr, &tls.Config{ServerName: p.config.SMTPHost})
	if err != nil {
		return fmt.Errorf("failed to connect with TLS: %w", err)
	}
	defer conn.Close()

	client, err := smtp.NewClient(conn, p.config.SMTPHost)
	if err != nil {
		return fmt.Errorf("failed to create SMTP client: %w", err)
	}
	defer client.Close()

	if auth != nil {
		if err := client.Auth(auth); err != nil {
			return fmt.Errorf("authentication failed: %w", err)
		}
	}

	if err := client.Mail(from); err != nil {
		return fmt.Errorf("MAIL command failed: %w", err)
	}

	for _, recipient := range to {
		if err := client.Rcpt(recipient); err != nil {
			return fmt.Errorf("RCPT command failed for %s: %w", recipient, err)
		}
	}

	w, err := client.Data()
	if err != nil {
		return fmt.Errorf("DATA command failed: %w", err)
	}

	_, err = w.Write(msg)
	if err != nil {
		return fmt.Errorf("failed to write message: %w", err)
	}

	if err := w.Close(); err != nil {
		return fmt.Errorf("failed to close message: %w", err)
	}

	return client.Quit()
}

// sendWithSTARTTLS sends email using STARTTLS
func (p *SMTPProvider) sendWithSTARTTLS(addr string, auth smtp.Auth, from string, to []string, msg []byte) error {
	client, err := smtp.Dial(addr)
	if err != nil {
		return fmt.Errorf("failed to connect: %w", err)
	}
	defer client.Close()

	if err := client.StartTLS(&tls.Config{ServerName: p.config.SMTPHost}); err != nil {
		return fmt.Errorf("STARTTLS failed: %w", err)
	}

	if auth != nil {
		if err := client.Auth(auth); err != nil {
			return fmt.Errorf("authentication failed: %w", err)
		}
	}

	if err := client.Mail(from); err != nil {
		return fmt.Errorf("MAIL command failed: %w", err)
	}

	for _, recipient := range to {
		if err := client.Rcpt(recipient); err != nil {
			return fmt.Errorf("RCPT command failed for %s: %w", recipient, err)
		}
	}

	w, err := client.Data()
	if err != nil {
		return fmt.Errorf("DATA command failed: %w", err)
	}

	_, err = w.Write(msg)
	if err != nil {
		return fmt.Errorf("failed to write message: %w", err)
	}

	if err := w.Close(); err != nil {
		return fmt.Errorf("failed to close message: %w", err)
	}

	return client.Quit()
}

// extractDomain extracts the domain from an email address
func extractDomain(email string) string {
	parts := strings.Split(email, "@")
	if len(parts) == 2 {
		return parts[1]
	}
	return "localhost"
}

// encodeBase64 encodes bytes to base64 with line wrapping
func encodeBase64(data []byte) string {
	encoded := make([]byte, base64EncodedLen(len(data)))
	base64Encode(encoded, data)

	// Wrap lines at 76 characters
	var sb strings.Builder
	for i := 0; i < len(encoded); i += 76 {
		end := i + 76
		if end > len(encoded) {
			end = len(encoded)
		}
		sb.Write(encoded[i:end])
		sb.WriteString("\r\n")
	}
	return sb.String()
}

// base64EncodedLen returns the length of the base64 encoding of an input of length n
func base64EncodedLen(n int) int {
	return (n + 2) / 3 * 4
}

// base64Encode encodes src using standard base64 encoding
func base64Encode(dst, src []byte) {
	const encodeStd = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789+/"

	di, si := 0, 0
	n := (len(src) / 3) * 3
	for si < n {
		val := uint(src[si+0])<<16 | uint(src[si+1])<<8 | uint(src[si+2])
		dst[di+0] = encodeStd[val>>18&0x3F]
		dst[di+1] = encodeStd[val>>12&0x3F]
		dst[di+2] = encodeStd[val>>6&0x3F]
		dst[di+3] = encodeStd[val&0x3F]
		si += 3
		di += 4
	}

	remain := len(src) - si
	if remain == 0 {
		return
	}

	val := uint(src[si+0]) << 16
	if remain == 2 {
		val |= uint(src[si+1]) << 8
	}

	dst[di+0] = encodeStd[val>>18&0x3F]
	dst[di+1] = encodeStd[val>>12&0x3F]

	switch remain {
	case 2:
		dst[di+2] = encodeStd[val>>6&0x3F]
		dst[di+3] = '='
	case 1:
		dst[di+2] = '='
		dst[di+3] = '='
	}
}
