package service

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"syscall"
	"time"

	"github.com/google/uuid"
	"github.com/msgfy/linktor/internal/domain/entity"
	"github.com/msgfy/linktor/internal/infrastructure/storage"
)

const (
	defaultMaxDownloadTimeout = 5 * time.Minute
	maxDownloadSize           = 100 * 1024 * 1024 // 100MB
	defaultUserAgent          = "Linktor/1.0 MediaProcessor"
)

// MediaProcessor handles media download, processing, and storage.
type MediaProcessor struct {
	storage            storage.Client
	maxDownloadTimeout time.Duration
	// allowPrivateHosts disables the SSRF guard. It defaults to false and is only
	// meant to be enabled in tests that talk to loopback httptest servers.
	allowPrivateHosts bool
}

// NewMediaProcessor creates a new MediaProcessor with default settings.
func NewMediaProcessor(store storage.Client) *MediaProcessor {
	return &MediaProcessor{
		storage:            store,
		maxDownloadTimeout: defaultMaxDownloadTimeout,
	}
}

// WithDownloadTimeout sets a custom download timeout.
func (p *MediaProcessor) WithDownloadTimeout(d time.Duration) *MediaProcessor {
	p.maxDownloadTimeout = d
	return p
}

// WithAllowPrivateHosts toggles the SSRF guard that blocks private/loopback/
// link-local destinations. Intended for tests only.
func (p *MediaProcessor) WithAllowPrivateHosts(allow bool) *MediaProcessor {
	p.allowPrivateHosts = allow
	return p
}

// isBlockedIP reports whether ip points at a private, loopback, link-local,
// unspecified, or otherwise non-routable range that an SSRF attacker could use
// to reach internal services (e.g. 169.254.169.254 cloud metadata, 127.0.0.1,
// 10.0.0.0/8).
func isBlockedIP(ip net.IP) bool {
	return ip == nil ||
		ip.IsLoopback() ||
		ip.IsPrivate() ||
		ip.IsLinkLocalUnicast() ||
		ip.IsLinkLocalMulticast() ||
		ip.IsMulticast() ||
		ip.IsUnspecified()
}

// newHTTPClient builds an http.Client whose dialer rejects connections to
// blocked IPs. The check runs in Dialer.Control, i.e. after DNS resolution and
// against the concrete IP being dialed, which also defeats DNS-rebinding.
func (p *MediaProcessor) newHTTPClient() *http.Client {
	dialer := &net.Dialer{Timeout: 15 * time.Second}
	if !p.allowPrivateHosts {
		dialer.Control = func(_, address string, _ syscall.RawConn) error {
			host, _, err := net.SplitHostPort(address)
			if err != nil {
				return err
			}
			ip := net.ParseIP(host)
			if isBlockedIP(ip) {
				return fmt.Errorf("blocked non-routable address: %s", host)
			}
			return nil
		}
	}
	return &http.Client{
		Timeout: p.maxDownloadTimeout,
		Transport: &http.Transport{
			DialContext:           dialer.DialContext,
			TLSHandshakeTimeout:   15 * time.Second,
			ResponseHeaderTimeout: p.maxDownloadTimeout,
		},
	}
}

// DownloadAndStore downloads media from a URL, stores it, and returns the storage URL.
// Falls back gracefully if download fails (returns empty string + error, caller decides).
func (p *MediaProcessor) DownloadAndStore(ctx context.Context, sourceURL, key, contentType string) (string, error) {
	if sourceURL == "" {
		return "", fmt.Errorf("source URL is empty")
	}

	// Restrict to http/https. Rejects file://, gopher://, data:, etc. that could
	// be abused for SSRF/local-file reads.
	parsed, err := url.Parse(sourceURL)
	if err != nil {
		return "", fmt.Errorf("invalid source URL: %w", err)
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return "", fmt.Errorf("unsupported URL scheme %q (only http/https allowed)", parsed.Scheme)
	}

	ctx, cancel := context.WithTimeout(ctx, p.maxDownloadTimeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, sourceURL, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("User-Agent", defaultUserAgent)

	// Use an SSRF-guarded client (blocks private/loopback/link-local IPs) with
	// its own timeout, rather than http.DefaultClient.
	resp, err := p.newHTTPClient().Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to download media: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("download returned status %d", resp.StatusCode)
	}

	limitedReader := io.LimitReader(resp.Body, maxDownloadSize+1)
	data, err := io.ReadAll(limitedReader)
	if err != nil {
		return "", fmt.Errorf("failed to read response body: %w", err)
	}
	if int64(len(data)) > maxDownloadSize {
		return "", fmt.Errorf("download exceeds maximum size of %d bytes", maxDownloadSize)
	}

	if contentType == "" {
		contentType = resp.Header.Get("Content-Type")
	}
	if contentType == "" {
		contentType = p.DetectContentType(data)
	}

	url, err := p.storage.Upload(ctx, key, data, contentType)
	if err != nil {
		return "", fmt.Errorf("failed to upload media: %w", err)
	}

	return url, nil
}

// ProcessAttachment downloads and stores an attachment, returning the updated attachment with storage URL.
// This is the main entry point for processing inbound media.
func (p *MediaProcessor) ProcessAttachment(ctx context.Context, attachment *entity.MessageAttachment) (*entity.MessageAttachment, error) {
	if attachment == nil {
		return nil, fmt.Errorf("attachment is nil")
	}
	if attachment.URL == "" {
		return attachment, nil
	}

	filename := attachment.Filename
	if filename == "" {
		filename = uuid.New().String()
	}

	key := GenerateKey(attachment.Metadata["channel_id"], attachment.MessageID, filename)

	storedURL, err := p.DownloadAndStore(ctx, attachment.URL, key, attachment.MimeType)
	if err != nil {
		return attachment, fmt.Errorf("failed to process attachment: %w", err)
	}

	result := *attachment
	result.URL = storedURL

	return &result, nil
}

// DetectContentType determines the content type from data bytes.
func (p *MediaProcessor) DetectContentType(data []byte) string {
	if len(data) == 0 {
		return "application/octet-stream"
	}
	return http.DetectContentType(data)
}

// GenerateKey creates a unique storage key for a media file.
// Format: {channelID}/{messageID}/{filename} or UUID if no filename.
func GenerateKey(channelID, messageID, filename string) string {
	if filename == "" {
		filename = uuid.New().String()
	}
	if channelID == "" {
		channelID = "unknown"
	}
	if messageID == "" {
		messageID = "unknown"
	}
	return fmt.Sprintf("%s/%s/%s", channelID, messageID, filename)
}
