package storage

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewLocalClient(t *testing.T) {
	client := NewLocalClient("/tmp/uploads", "http://localhost:8080/files")
	assert.NotNil(t, client)
	assert.Equal(t, "/tmp/uploads", client.uploadDir)
	assert.Equal(t, "http://localhost:8080/files", client.baseURL)
}

func TestLocalClient_ImplementsInterface(t *testing.T) {
	var _ Client = (*LocalClient)(nil)
}

func TestLocalClient_Upload_Simple(t *testing.T) {
	tmpDir := t.TempDir()
	client := NewLocalClient(tmpDir, "http://cdn.example.com")

	url, err := client.Upload(context.Background(), "test.txt", []byte("hello world"), "text/plain")
	require.NoError(t, err)
	assert.Equal(t, "http://cdn.example.com/test.txt", url)

	// Verify file was written
	data, err := os.ReadFile(filepath.Join(tmpDir, "test.txt"))
	require.NoError(t, err)
	assert.Equal(t, "hello world", string(data))
}

func TestLocalClient_Upload_NestedPath(t *testing.T) {
	tmpDir := t.TempDir()
	client := NewLocalClient(tmpDir, "http://cdn.example.com")

	url, err := client.Upload(context.Background(), "images/avatar/photo.png", []byte("png-data"), "image/png")
	require.NoError(t, err)
	assert.Equal(t, "http://cdn.example.com/images/avatar/photo.png", url)

	// Verify nested dirs were created and file exists
	data, err := os.ReadFile(filepath.Join(tmpDir, "images", "avatar", "photo.png"))
	require.NoError(t, err)
	assert.Equal(t, "png-data", string(data))
}

func TestLocalClient_Upload_EmptyKey_GeneratesUUID(t *testing.T) {
	tmpDir := t.TempDir()
	client := NewLocalClient(tmpDir, "http://cdn.example.com")

	url, err := client.Upload(context.Background(), "", []byte("auto-named"), "application/octet-stream")
	require.NoError(t, err)
	assert.Contains(t, url, "http://cdn.example.com/")

	// UUID should be 36 chars (8-4-4-4-12)
	key := url[len("http://cdn.example.com/"):]
	assert.Len(t, key, 36)

	// Verify file exists
	data, err := os.ReadFile(filepath.Join(tmpDir, key))
	require.NoError(t, err)
	assert.Equal(t, "auto-named", string(data))
}

func TestLocalClient_Upload_EmptyData(t *testing.T) {
	tmpDir := t.TempDir()
	client := NewLocalClient(tmpDir, "http://cdn.example.com")

	url, err := client.Upload(context.Background(), "empty.txt", []byte{}, "text/plain")
	require.NoError(t, err)
	assert.Equal(t, "http://cdn.example.com/empty.txt", url)

	data, err := os.ReadFile(filepath.Join(tmpDir, "empty.txt"))
	require.NoError(t, err)
	assert.Empty(t, data)
}

func TestLocalClient_Upload_LargeData(t *testing.T) {
	tmpDir := t.TempDir()
	client := NewLocalClient(tmpDir, "http://cdn.example.com")

	largeData := make([]byte, 1024*1024) // 1MB
	for i := range largeData {
		largeData[i] = byte(i % 256)
	}

	url, err := client.Upload(context.Background(), "large.bin", largeData, "application/octet-stream")
	require.NoError(t, err)
	assert.Equal(t, "http://cdn.example.com/large.bin", url)

	data, err := os.ReadFile(filepath.Join(tmpDir, "large.bin"))
	require.NoError(t, err)
	assert.Equal(t, largeData, data)
}

func TestLocalClient_Upload_OverwriteExisting(t *testing.T) {
	tmpDir := t.TempDir()
	client := NewLocalClient(tmpDir, "http://cdn.example.com")

	_, err := client.Upload(context.Background(), "file.txt", []byte("first"), "text/plain")
	require.NoError(t, err)

	_, err = client.Upload(context.Background(), "file.txt", []byte("second"), "text/plain")
	require.NoError(t, err)

	data, err := os.ReadFile(filepath.Join(tmpDir, "file.txt"))
	require.NoError(t, err)
	assert.Equal(t, "second", string(data))
}

func TestLocalClient_Upload_InvalidDirectory(t *testing.T) {
	// Use a path that cannot be created (on macOS/Linux, /proc is read-only)
	client := NewLocalClient("/dev/null/impossible", "http://cdn.example.com")

	_, err := client.Upload(context.Background(), "test.txt", []byte("data"), "text/plain")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to create directory")
}

func TestLocalClient_Upload_MultipleFiles(t *testing.T) {
	tmpDir := t.TempDir()
	client := NewLocalClient(tmpDir, "http://cdn.example.com")

	files := map[string]string{
		"a.txt":          "content-a",
		"b.txt":          "content-b",
		"dir/c.txt":      "content-c",
		"dir/sub/d.txt":  "content-d",
	}

	for key, content := range files {
		url, err := client.Upload(context.Background(), key, []byte(content), "text/plain")
		require.NoError(t, err)
		assert.Equal(t, "http://cdn.example.com/"+key, url)
	}

	// Verify all files
	for key, content := range files {
		data, err := os.ReadFile(filepath.Join(tmpDir, key))
		require.NoError(t, err)
		assert.Equal(t, content, string(data))
	}
}

func TestLocalClient_Delete_Success(t *testing.T) {
	tmpDir := t.TempDir()
	client := NewLocalClient(tmpDir, "http://cdn.example.com")

	// Upload first
	_, err := client.Upload(context.Background(), "to-delete.txt", []byte("data"), "text/plain")
	require.NoError(t, err)

	// Delete
	err = client.Delete(context.Background(), "to-delete.txt")
	require.NoError(t, err)

	// Verify file is gone
	_, err = os.ReadFile(filepath.Join(tmpDir, "to-delete.txt"))
	assert.True(t, os.IsNotExist(err))
}

func TestLocalClient_Delete_NotFound(t *testing.T) {
	tmpDir := t.TempDir()
	client := NewLocalClient(tmpDir, "http://cdn.example.com")

	err := client.Delete(context.Background(), "nonexistent.txt")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "file not found")
}

func TestLocalClient_GetURL(t *testing.T) {
	client := NewLocalClient("/tmp", "http://cdn.example.com")

	url, err := client.GetURL(context.Background(), "test.txt")
	require.NoError(t, err)
	assert.Equal(t, "http://cdn.example.com/test.txt", url)
}

func TestLocalClient_Upload_DifferentBaseURLs(t *testing.T) {
	tmpDir := t.TempDir()

	tests := []struct {
		baseURL     string
		key         string
		expectedURL string
	}{
		{"http://localhost:8080", "file.txt", "http://localhost:8080/file.txt"},
		{"https://cdn.example.com", "img.png", "https://cdn.example.com/img.png"},
		{"http://localhost:8080/uploads", "doc.pdf", "http://localhost:8080/uploads/doc.pdf"},
	}

	for _, tt := range tests {
		client := NewLocalClient(tmpDir, tt.baseURL)
		url, err := client.Upload(context.Background(), tt.key, []byte("data"), "text/plain")
		require.NoError(t, err)
		assert.Equal(t, tt.expectedURL, url)
	}
}
