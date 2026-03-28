package service

import (
	"context"
	"fmt"
	"io"
	"net/http"
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

// DownloadAndStore downloads media from a URL, stores it, and returns the storage URL.
// Falls back gracefully if download fails (returns empty string + error, caller decides).
func (p *MediaProcessor) DownloadAndStore(ctx context.Context, sourceURL, key, contentType string) (string, error) {
	if sourceURL == "" {
		return "", fmt.Errorf("source URL is empty")
	}

	ctx, cancel := context.WithTimeout(ctx, p.maxDownloadTimeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, sourceURL, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("User-Agent", defaultUserAgent)

	resp, err := http.DefaultClient.Do(req)
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
