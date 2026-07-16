package storage

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/url"
	"strings"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

// Ensure MinIOClient implements Client
var _ Client = (*MinIOClient)(nil)

// MinIOClient stores files in MinIO/S3-compatible object storage
type MinIOClient struct {
	client     *minio.Client
	bucketName string
	presignTTL time.Duration
	// publicBaseURL, when set, makes Upload/GetURL return a stable media-proxy
	// URL (<publicBaseURL>/api/v1/media/<key>) served by our own backend instead
	// of a presigned S3 URL. Presigned URLs expire (max 7 days) and, once stored
	// in the DB, break; the proxy URL never expires and streams via Open.
	publicBaseURL string
}

// NewMinIOClient creates a MinIO storage client and ensures the bucket exists.
// publicBaseURL (e.g. "https://api.linktor.dev") selects the stable media-proxy
// URL scheme; pass "" to fall back to presigned URLs.
func NewMinIOClient(endpoint, accessKey, secretKey, bucketName, region string, useSSL bool, publicBaseURL string) (*MinIOClient, error) {
	client, err := minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(accessKey, secretKey, ""),
		Secure: useSSL,
		Region: region,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create MinIO client: %w", err)
	}

	// Ensure bucket exists
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	exists, err := client.BucketExists(ctx, bucketName)
	if err != nil {
		return nil, fmt.Errorf("failed to check bucket: %w", err)
	}
	if !exists {
		if err := client.MakeBucket(ctx, bucketName, minio.MakeBucketOptions{Region: region}); err != nil {
			return nil, fmt.Errorf("failed to create bucket: %w", err)
		}
	}

	return &MinIOClient{
		client:        client,
		bucketName:    bucketName,
		presignTTL:    7 * 24 * time.Hour, // 7 days (presigned fallback only)
		publicBaseURL: strings.TrimRight(publicBaseURL, "/"),
	}, nil
}

// Upload stores data in MinIO and returns a durable URL for it.
func (c *MinIOClient) Upload(ctx context.Context, key string, data []byte, contentType string) (string, error) {
	reader := bytes.NewReader(data)

	_, err := c.client.PutObject(ctx, c.bucketName, key, reader, int64(len(data)), minio.PutObjectOptions{
		ContentType: contentType,
	})
	if err != nil {
		return "", fmt.Errorf("failed to upload object: %w", err)
	}

	return c.GetURL(ctx, key)
}

// Delete removes an object from MinIO
func (c *MinIOClient) Delete(ctx context.Context, key string) error {
	err := c.client.RemoveObject(ctx, c.bucketName, key, minio.RemoveObjectOptions{})
	if err != nil {
		return fmt.Errorf("failed to delete object: %w", err)
	}
	return nil
}

// GetURL returns a durable URL for the object. When a public base URL is
// configured it points at our media proxy (never expires); otherwise it falls
// back to a presigned S3 URL (valid for presignTTL).
func (c *MinIOClient) GetURL(ctx context.Context, key string) (string, error) {
	if c.publicBaseURL != "" {
		return c.publicBaseURL + "/api/v1/media/" + key, nil
	}

	reqParams := make(url.Values)
	presignedURL, err := c.client.PresignedGetObject(ctx, c.bucketName, key, c.presignTTL, reqParams)
	if err != nil {
		return "", fmt.Errorf("failed to generate presigned URL: %w", err)
	}
	return presignedURL.String(), nil
}

// Open streams an object from MinIO for the media proxy.
func (c *MinIOClient) Open(ctx context.Context, key string) (io.ReadCloser, string, int64, error) {
	obj, err := c.client.GetObject(ctx, c.bucketName, key, minio.GetObjectOptions{})
	if err != nil {
		return nil, "", 0, fmt.Errorf("failed to get object: %w", err)
	}
	// Stat forces the request so a missing object surfaces here rather than as a
	// mid-stream error, and yields the stored content type + size.
	info, err := obj.Stat()
	if err != nil {
		obj.Close()
		return nil, "", 0, fmt.Errorf("failed to stat object: %w", err)
	}
	return obj, info.ContentType, info.Size, nil
}
