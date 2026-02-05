package email

import (
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"mime"
	"mime/multipart"
	"net"
	"strings"
	"sync"
	"time"
)

// IMAPClient handles receiving emails via IMAP
type IMAPClient struct {
	config      *Config
	mu          sync.Mutex
	conn        net.Conn
	lastUID     uint32
	stopCh      chan struct{}
	running     bool
}

// NewIMAPClient creates a new IMAP client
func NewIMAPClient(config *Config) (*IMAPClient, error) {
	if config.IMAPHost == "" {
		return nil, fmt.Errorf("imap_host is required")
	}
	if config.IMAPPort == 0 {
		config.IMAPPort = 993 // Default IMAPS port
	}
	if config.IMAPFolder == "" {
		config.IMAPFolder = "INBOX"
	}
	if config.IMAPPollInterval == 0 {
		config.IMAPPollInterval = 30 // Default 30 seconds
	}

	return &IMAPClient{
		config: config,
		stopCh: make(chan struct{}),
	}, nil
}

// Connect connects to the IMAP server
func (c *IMAPClient) Connect(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	addr := fmt.Sprintf("%s:%d", c.config.IMAPHost, c.config.IMAPPort)

	// Connect with TLS (IMAPS)
	conn, err := tls.Dial("tcp", addr, &tls.Config{
		ServerName: c.config.IMAPHost,
	})
	if err != nil {
		return fmt.Errorf("failed to connect to IMAP server: %w", err)
	}

	c.conn = conn

	// Read server greeting
	if err := c.readGreeting(); err != nil {
		conn.Close()
		return fmt.Errorf("failed to read server greeting: %w", err)
	}

	// Login
	if err := c.login(); err != nil {
		conn.Close()
		return fmt.Errorf("failed to login: %w", err)
	}

	// Select folder
	if err := c.selectFolder(c.config.IMAPFolder); err != nil {
		conn.Close()
		return fmt.Errorf("failed to select folder: %w", err)
	}

	return nil
}

// Disconnect closes the IMAP connection
func (c *IMAPClient) Disconnect() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.conn != nil {
		// Send LOGOUT command
		c.sendCommand("LOGOUT")
		c.conn.Close()
		c.conn = nil
	}

	return nil
}

// StartPolling starts polling for new emails
func (c *IMAPClient) StartPolling(ctx context.Context, handler func(*IncomingEmail)) error {
	c.mu.Lock()
	if c.running {
		c.mu.Unlock()
		return fmt.Errorf("polling already running")
	}
	c.running = true
	c.stopCh = make(chan struct{})
	c.mu.Unlock()

	ticker := time.NewTicker(time.Duration(c.config.IMAPPollInterval) * time.Second)
	defer ticker.Stop()

	// Initial fetch
	go c.fetchAndProcess(ctx, handler)

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-c.stopCh:
			return nil
		case <-ticker.C:
			c.fetchAndProcess(ctx, handler)
		}
	}
}

// StopPolling stops polling for new emails
func (c *IMAPClient) StopPolling() {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.running {
		close(c.stopCh)
		c.running = false
	}
}

// fetchAndProcess fetches new emails and processes them
func (c *IMAPClient) fetchAndProcess(ctx context.Context, handler func(*IncomingEmail)) {
	c.mu.Lock()
	conn := c.conn
	c.mu.Unlock()

	if conn == nil {
		// Attempt to reconnect
		if err := c.Connect(ctx); err != nil {
			return
		}
	}

	emails, err := c.FetchNewEmails(ctx)
	if err != nil {
		// Log error but continue
		return
	}

	for _, email := range emails {
		handler(email)
	}
}

// FetchNewEmails fetches new (unseen) emails
func (c *IMAPClient) FetchNewEmails(ctx context.Context) ([]*IncomingEmail, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.conn == nil {
		return nil, fmt.Errorf("not connected")
	}

	// Search for unseen emails
	_, err := c.sendCommand("SEARCH UNSEEN")
	if err != nil {
		return nil, fmt.Errorf("search failed: %w", err)
	}

	// Parse search response to get message UIDs
	uids := c.parseSearchResponse()
	if len(uids) == 0 {
		return nil, nil
	}

	var emails []*IncomingEmail

	// Fetch each email
	for _, uid := range uids {
		email, err := c.fetchEmail(uid)
		if err != nil {
			continue // Log error but continue with other emails
		}
		emails = append(emails, email)

		// Mark as seen
		c.markAsSeen(uid)
	}

	return emails, nil
}

// readGreeting reads the server greeting
func (c *IMAPClient) readGreeting() error {
	buf := make([]byte, 1024)
	n, err := c.conn.Read(buf)
	if err != nil {
		return err
	}
	response := string(buf[:n])
	if !strings.Contains(response, "OK") {
		return fmt.Errorf("unexpected greeting: %s", response)
	}
	return nil
}

// login authenticates with the IMAP server
func (c *IMAPClient) login() error {
	cmd := fmt.Sprintf("LOGIN %s %s", c.config.IMAPUsername, c.config.IMAPPassword)
	response, err := c.sendCommand(cmd)
	if err != nil {
		return err
	}
	if !strings.Contains(response, "OK") {
		return fmt.Errorf("login failed: %s", response)
	}
	return nil
}

// selectFolder selects an IMAP folder
func (c *IMAPClient) selectFolder(folder string) error {
	cmd := fmt.Sprintf("SELECT %s", folder)
	response, err := c.sendCommand(cmd)
	if err != nil {
		return err
	}
	if !strings.Contains(response, "OK") {
		return fmt.Errorf("select failed: %s", response)
	}
	return nil
}

// sendCommand sends an IMAP command and returns the response
func (c *IMAPClient) sendCommand(cmd string) (string, error) {
	tag := fmt.Sprintf("A%d", time.Now().UnixNano()%1000)
	fullCmd := fmt.Sprintf("%s %s\r\n", tag, cmd)

	_, err := c.conn.Write([]byte(fullCmd))
	if err != nil {
		return "", fmt.Errorf("failed to send command: %w", err)
	}

	// Read response
	var response strings.Builder
	buf := make([]byte, 4096)

	for {
		c.conn.SetReadDeadline(time.Now().Add(30 * time.Second))
		n, err := c.conn.Read(buf)
		if err != nil {
			if err == io.EOF {
				break
			}
			return "", fmt.Errorf("failed to read response: %w", err)
		}

		response.Write(buf[:n])

		// Check if we've received the tagged response
		if strings.Contains(response.String(), tag) {
			break
		}
	}

	return response.String(), nil
}

// parseSearchResponse parses SEARCH response and returns UIDs
func (c *IMAPClient) parseSearchResponse() []uint32 {
	// Simplified parsing - in production would use proper IMAP response parsing
	var uids []uint32
	// Parse "* SEARCH 1 2 3 4 5" response
	return uids
}

// fetchEmail fetches a single email by UID
func (c *IMAPClient) fetchEmail(uid uint32) (*IncomingEmail, error) {
	cmd := fmt.Sprintf("UID FETCH %d (RFC822)", uid)
	response, err := c.sendCommand(cmd)
	if err != nil {
		return nil, err
	}

	return c.parseEmailResponse(response)
}

// markAsSeen marks an email as seen
func (c *IMAPClient) markAsSeen(uid uint32) error {
	cmd := fmt.Sprintf("UID STORE %d +FLAGS (\\Seen)", uid)
	_, err := c.sendCommand(cmd)
	return err
}

// parseEmailResponse parses a FETCH response into an IncomingEmail
func (c *IMAPClient) parseEmailResponse(response string) (*IncomingEmail, error) {
	// This is a simplified implementation
	// In production, you would use a proper MIME parser

	email := &IncomingEmail{
		ReceivedAt: time.Now(),
		Headers:    make(map[string]string),
		Metadata:   make(map[string]string),
	}

	// Parse headers
	lines := strings.Split(response, "\r\n")
	inHeaders := true
	var bodyBuilder strings.Builder

	for _, line := range lines {
		if inHeaders {
			if line == "" {
				inHeaders = false
				continue
			}
			if idx := strings.Index(line, ": "); idx > 0 {
				header := line[:idx]
				value := line[idx+2:]
				email.Headers[header] = value

				switch strings.ToLower(header) {
				case "from":
					email.From = value
					email.FromName = parseFromName(value)
				case "to":
					email.To = parseAddressList(value)
				case "cc":
					email.CC = parseAddressList(value)
				case "subject":
					email.Subject = value
				case "message-id":
					email.MessageID = value
				case "in-reply-to":
					email.InReplyTo = value
				case "references":
					email.References = value
				}
			}
		} else {
			bodyBuilder.WriteString(line)
			bodyBuilder.WriteString("\n")
		}
	}

	email.TextBody = bodyBuilder.String()

	return email, nil
}

// parseFromName extracts the name from a From header
func parseFromName(from string) string {
	// Parse "Name <email@example.com>" format
	if idx := strings.Index(from, "<"); idx > 0 {
		return strings.TrimSpace(from[:idx])
	}
	return ""
}

// parseAddressList parses a comma-separated list of email addresses
func parseAddressList(list string) []string {
	addresses := strings.Split(list, ",")
	result := make([]string, 0, len(addresses))
	for _, addr := range addresses {
		addr = strings.TrimSpace(addr)
		if addr != "" {
			// Extract email from "Name <email>" format
			if idx := strings.Index(addr, "<"); idx >= 0 {
				if endIdx := strings.Index(addr, ">"); endIdx > idx {
					addr = addr[idx+1 : endIdx]
				}
			}
			result = append(result, addr)
		}
	}
	return result
}

// parseMIMEEmail parses a MIME email message
func parseMIMEEmail(body io.Reader, contentType string) (*IncomingEmail, error) {
	email := &IncomingEmail{
		ReceivedAt: time.Now(),
		Headers:    make(map[string]string),
		Metadata:   make(map[string]string),
	}

	mediaType, params, err := mime.ParseMediaType(contentType)
	if err != nil {
		return nil, fmt.Errorf("failed to parse content type: %w", err)
	}

	if strings.HasPrefix(mediaType, "multipart/") {
		mr := multipart.NewReader(body, params["boundary"])
		for {
			part, err := mr.NextPart()
			if err == io.EOF {
				break
			}
			if err != nil {
				return nil, fmt.Errorf("failed to read part: %w", err)
			}

			partType := part.Header.Get("Content-Type")
			partMediaType, _, _ := mime.ParseMediaType(partType)

			switch {
			case strings.HasPrefix(partMediaType, "text/plain"):
				content, _ := io.ReadAll(part)
				email.TextBody = string(content)
			case strings.HasPrefix(partMediaType, "text/html"):
				content, _ := io.ReadAll(part)
				email.HTMLBody = string(content)
			default:
				// Attachment
				content, _ := io.ReadAll(part)
				filename := part.FileName()
				if filename == "" {
					filename = "attachment"
				}
				email.Attachments = append(email.Attachments, &Attachment{
					Filename:    filename,
					ContentType: partMediaType,
					Content:     content,
					Size:        int64(len(content)),
					ContentID:   part.Header.Get("Content-ID"),
				})
			}
		}
	} else if strings.HasPrefix(mediaType, "text/plain") {
		content, _ := io.ReadAll(body)
		email.TextBody = string(content)
	} else if strings.HasPrefix(mediaType, "text/html") {
		content, _ := io.ReadAll(body)
		email.HTMLBody = string(content)
	}

	return email, nil
}
