package storage

import (
	"context"
	"fmt"
	"io"
	"mime"
	"os"
	"path/filepath"

	"github.com/google/uuid"
)

// Client defines the interface for object storage
type Client interface {
	// Upload uploads data and returns the public URL
	Upload(ctx context.Context, key string, data []byte, contentType string) (string, error)
	// Delete removes an object by key
	Delete(ctx context.Context, key string) error
	// GetURL returns the public URL for a given key
	GetURL(ctx context.Context, key string) (string, error)
	// Open opens the object for reading, returning its content type and size.
	// The caller must Close the returned reader. Used by the media proxy so a
	// stored URL can be served without a (expiring) presigned link.
	Open(ctx context.Context, key string) (io.ReadCloser, string, int64, error)
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

// Delete removes a file from the local filesystem
func (c *LocalClient) Delete(ctx context.Context, key string) error {
	path := filepath.Join(c.uploadDir, key)
	if err := os.Remove(path); err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("file not found: %s", key)
		}
		return fmt.Errorf("failed to delete file: %w", err)
	}
	return nil
}

// GetURL returns the public URL for a given key
func (c *LocalClient) GetURL(ctx context.Context, key string) (string, error) {
	return fmt.Sprintf("%s/%s", c.baseURL, key), nil
}

// Open opens a stored file for reading.
func (c *LocalClient) Open(ctx context.Context, key string) (io.ReadCloser, string, int64, error) {
	f, err := os.Open(filepath.Join(c.uploadDir, key))
	if err != nil {
		return nil, "", 0, err
	}
	info, err := f.Stat()
	if err != nil {
		f.Close()
		return nil, "", 0, err
	}
	contentType := mime.TypeByExtension(filepath.Ext(key))
	if contentType == "" {
		contentType = "application/octet-stream"
	}
	return f, contentType, info.Size(), nil
}
