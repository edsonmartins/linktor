package service

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/msgfy/linktor/internal/domain/entity"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockStorage implements storage.Client for testing.
type mockStorage struct {
	uploadedKeys []string
	uploadErr    error
}

func (m *mockStorage) Upload(_ context.Context, key string, _ []byte, _ string) (string, error) {
	m.uploadedKeys = append(m.uploadedKeys, key)
	if m.uploadErr != nil {
		return "", m.uploadErr
	}
	return "https://cdn.example.com/" + key, nil
}

func (m *mockStorage) Delete(_ context.Context, _ string) error {
	return nil
}

func (m *mockStorage) GetURL(_ context.Context, key string) (string, error) {
	return "https://cdn.example.com/" + key, nil
}

func TestNewMediaProcessor(t *testing.T) {
	store := &mockStorage{}
	mp := NewMediaProcessor(store)

	require.NotNil(t, mp)
	assert.Equal(t, defaultMaxDownloadTimeout, mp.maxDownloadTimeout)
	assert.NotNil(t, mp.storage)
}

func TestDownloadAndStore_Success(t *testing.T) {
	// JPEG magic bytes
	jpegData := []byte{0xFF, 0xD8, 0xFF, 0xE0, 0x00, 0x10, 0x4A, 0x46, 0x49, 0x46}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "image/jpeg")
		w.WriteHeader(http.StatusOK)
		w.Write(jpegData)
	}))
	defer server.Close()

	store := &mockStorage{}
	mp := NewMediaProcessor(store)

	url, err := mp.DownloadAndStore(context.Background(), server.URL, "test/key/photo.jpg", "image/jpeg")

	require.NoError(t, err)
	assert.Equal(t, "https://cdn.example.com/test/key/photo.jpg", url)
	assert.Contains(t, store.uploadedKeys, "test/key/photo.jpg")
}

func TestDownloadAndStore_ServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	store := &mockStorage{}
	mp := NewMediaProcessor(store)

	url, err := mp.DownloadAndStore(context.Background(), server.URL, "test/key", "")

	assert.Error(t, err)
	assert.Empty(t, url)
	assert.Contains(t, err.Error(), "status 500")
}

func TestDownloadAndStore_Timeout(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(2 * time.Second)
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("data"))
	}))
	defer server.Close()

	store := &mockStorage{}
	mp := NewMediaProcessor(store).WithDownloadTimeout(100 * time.Millisecond)

	url, err := mp.DownloadAndStore(context.Background(), server.URL, "test/key", "")

	assert.Error(t, err)
	assert.Empty(t, url)
}

func TestDownloadAndStore_InvalidURL(t *testing.T) {
	store := &mockStorage{}
	mp := NewMediaProcessor(store)

	url, err := mp.DownloadAndStore(context.Background(), "not-a-valid-url://foo", "test/key", "")

	assert.Error(t, err)
	assert.Empty(t, url)
}

func TestDownloadAndStore_EmptyURL(t *testing.T) {
	store := &mockStorage{}
	mp := NewMediaProcessor(store)

	url, err := mp.DownloadAndStore(context.Background(), "", "test/key", "")

	assert.Error(t, err)
	assert.Empty(t, url)
	assert.Contains(t, err.Error(), "source URL is empty")
}

func TestProcessAttachment_Success(t *testing.T) {
	jpegData := []byte{0xFF, 0xD8, 0xFF, 0xE0, 0x00, 0x10}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "image/jpeg")
		w.WriteHeader(http.StatusOK)
		w.Write(jpegData)
	}))
	defer server.Close()

	store := &mockStorage{}
	mp := NewMediaProcessor(store)

	attachment := &entity.MessageAttachment{
		ID:        "att-1",
		MessageID: "msg-1",
		Type:      "image",
		Filename:  "photo.jpg",
		MimeType:  "image/jpeg",
		URL:       server.URL,
		Metadata:  map[string]string{"channel_id": "ch-1"},
		CreatedAt: time.Now(),
	}

	result, err := mp.ProcessAttachment(context.Background(), attachment)

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, "https://cdn.example.com/ch-1/msg-1/photo.jpg", result.URL)
	assert.Equal(t, attachment.ID, result.ID)
	assert.Equal(t, attachment.Filename, result.Filename)
}

func TestProcessAttachment_NoURL(t *testing.T) {
	store := &mockStorage{}
	mp := NewMediaProcessor(store)

	attachment := &entity.MessageAttachment{
		ID:        "att-1",
		MessageID: "msg-1",
		Type:      "image",
		Filename:  "photo.jpg",
		URL:       "",
		CreatedAt: time.Now(),
	}

	result, err := mp.ProcessAttachment(context.Background(), attachment)

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, "", result.URL)
	assert.Equal(t, attachment, result)
}

func TestProcessAttachment_NilAttachment(t *testing.T) {
	store := &mockStorage{}
	mp := NewMediaProcessor(store)

	result, err := mp.ProcessAttachment(context.Background(), nil)

	assert.Error(t, err)
	assert.Nil(t, result)
}

func TestDetectContentType_JPEG(t *testing.T) {
	mp := NewMediaProcessor(&mockStorage{})

	// JPEG starts with FF D8 FF
	data := []byte{0xFF, 0xD8, 0xFF, 0xE0, 0x00, 0x10, 0x4A, 0x46, 0x49, 0x46}
	ct := mp.DetectContentType(data)
	assert.Equal(t, "image/jpeg", ct)
}

func TestDetectContentType_PNG(t *testing.T) {
	mp := NewMediaProcessor(&mockStorage{})

	// PNG magic bytes
	data := []byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A}
	ct := mp.DetectContentType(data)
	assert.Equal(t, "image/png", ct)
}

func TestDetectContentType_PDF(t *testing.T) {
	mp := NewMediaProcessor(&mockStorage{})

	data := []byte("%PDF-1.4 some content here")
	ct := mp.DetectContentType(data)
	assert.Equal(t, "application/pdf", ct)
}

func TestDetectContentType_Empty(t *testing.T) {
	mp := NewMediaProcessor(&mockStorage{})

	ct := mp.DetectContentType([]byte{})
	assert.Equal(t, "application/octet-stream", ct)
}

func TestGenerateKey_WithFilename(t *testing.T) {
	key := GenerateKey("ch-1", "msg-1", "photo.jpg")
	assert.Equal(t, "ch-1/msg-1/photo.jpg", key)
}

func TestGenerateKey_WithoutFilename(t *testing.T) {
	key := GenerateKey("ch-1", "msg-1", "")

	parts := strings.Split(key, "/")
	require.Len(t, parts, 3)
	assert.Equal(t, "ch-1", parts[0])
	assert.Equal(t, "msg-1", parts[1])

	// Third part should be a valid UUID
	_, err := uuid.Parse(parts[2])
	assert.NoError(t, err)
}

func TestGenerateKey_EmptyChannelAndMessage(t *testing.T) {
	key := GenerateKey("", "", "photo.jpg")
	assert.Equal(t, "unknown/unknown/photo.jpg", key)
}
