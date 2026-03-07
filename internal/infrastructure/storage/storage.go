package storage

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/google/uuid"
)

// Client defines the interface for object storage
type Client interface {
	// Upload uploads data and returns the public URL
	Upload(ctx context.Context, key string, data []byte, contentType string) (string, error)
}

// LocalClient stores files on the local filesystem and returns a URL
type LocalClient struct {
	uploadDir string
	baseURL   string
}

// NewLocalClient creates a local filesystem storage client
func NewLocalClient(uploadDir, baseURL string) *LocalClient {
	return &LocalClient{
		uploadDir: uploadDir,
		baseURL:   baseURL,
	}
}

// Upload saves the file locally and returns a URL
func (c *LocalClient) Upload(ctx context.Context, key string, data []byte, contentType string) (string, error) {
	if key == "" {
		key = uuid.New().String()
	}

	dir := filepath.Dir(filepath.Join(c.uploadDir, key))
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", fmt.Errorf("failed to create directory: %w", err)
	}

	path := filepath.Join(c.uploadDir, key)
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return "", fmt.Errorf("failed to write file: %w", err)
	}

	url := fmt.Sprintf("%s/%s", c.baseURL, key)
	return url, nil
}
