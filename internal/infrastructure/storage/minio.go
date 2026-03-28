package storage

import (
	"bytes"
	"context"
	"fmt"
	"net/url"
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
}

// NewMinIOClient creates a MinIO storage client and ensures the bucket exists
func NewMinIOClient(endpoint, accessKey, secretKey, bucketName, region string, useSSL bool) (*MinIOClient, error) {
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
		client:     client,
		bucketName: bucketName,
		presignTTL: 7 * 24 * time.Hour, // 7 days
	}, nil
}

// Upload stores data in MinIO and returns a presigned URL
func (c *MinIOClient) Upload(ctx context.Context, key string, data []byte, contentType string) (string, error) {
	filePath := fmt.Sprintf("linktor-media/%s", key)
	reader := bytes.NewReader(data)

	_, err := c.client.PutObject(ctx, c.bucketName, filePath, reader, int64(len(data)), minio.PutObjectOptions{
		ContentType: contentType,
	})
	if err != nil {
		return "", fmt.Errorf("failed to upload object: %w", err)
	}

	return c.GetURL(ctx, key)
}

// Delete removes an object from MinIO
func (c *MinIOClient) Delete(ctx context.Context, key string) error {
	filePath := fmt.Sprintf("linktor-media/%s", key)
	err := c.client.RemoveObject(ctx, c.bucketName, filePath, minio.RemoveObjectOptions{})
	if err != nil {
		return fmt.Errorf("failed to delete object: %w", err)
	}
	return nil
}

// GetURL returns a presigned URL for the object (valid for 7 days)
func (c *MinIOClient) GetURL(ctx context.Context, key string) (string, error) {
	filePath := fmt.Sprintf("linktor-media/%s", key)
	reqParams := make(url.Values)

	presignedURL, err := c.client.PresignedGetObject(ctx, c.bucketName, filePath, c.presignTTL, reqParams)
	if err != nil {
		return "", fmt.Errorf("failed to generate presigned URL: %w", err)
	}

	return presignedURL.String(), nil
}
