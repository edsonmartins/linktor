package telegram

import (
	"fmt"
	"path/filepath"
	"strings"
)

// MediaInfo contains information about a media file
type MediaInfo struct {
	FileID       string
	FileUniqueID string
	FileSize     int64
	FilePath     string
	Width        int
	Height       int
	Duration     int
	MimeType     string
	Thumbnail    *PhotoSize
}

// PhotoSize represents a photo size
type PhotoSize struct {
	FileID       string
	FileUniqueID string
	Width        int
	Height       int
	FileSize     int64
}

// GetMimeTypeFromFilename guesses MIME type from filename
func GetMimeTypeFromFilename(filename string) string {
	ext := strings.ToLower(filepath.Ext(filename))

	mimeTypes := map[string]string{
		".jpg":  "image/jpeg",
		".jpeg": "image/jpeg",
		".png":  "image/png",
		".gif":  "image/gif",
		".webp": "image/webp",
		".mp4":  "video/mp4",
		".webm": "video/webm",
		".mp3":  "audio/mpeg",
		".ogg":  "audio/ogg",
		".m4a":  "audio/mp4",
		".pdf":  "application/pdf",
		".doc":  "application/msword",
		".docx": "application/vnd.openxmlformats-officedocument.wordprocessingml.document",
		".xls":  "application/vnd.ms-excel",
		".xlsx": "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
		".zip":  "application/zip",
		".txt":  "text/plain",
	}

	if mime, ok := mimeTypes[ext]; ok {
		return mime
	}

	return "application/octet-stream"
}

// ValidateMediaSize checks if media size is within Telegram limits
func ValidateMediaSize(mediaType string, sizeBytes int64) error {
	// Telegram file size limits (in bytes)
	const (
		maxPhotoSize    = 10 * 1024 * 1024  // 10 MB
		maxVideoSize    = 50 * 1024 * 1024  // 50 MB
		maxAudioSize    = 50 * 1024 * 1024  // 50 MB
		maxDocumentSize = 50 * 1024 * 1024  // 50 MB
		maxStickerSize  = 512 * 1024        // 512 KB
	)

	limits := map[string]int64{
		"photo":    maxPhotoSize,
		"video":    maxVideoSize,
		"audio":    maxAudioSize,
		"document": maxDocumentSize,
		"sticker":  maxStickerSize,
	}

	limit, ok := limits[mediaType]
	if !ok {
		limit = maxDocumentSize // Default to document limit
	}

	if sizeBytes > limit {
		return fmt.Errorf("%s size %d exceeds limit of %d bytes", mediaType, sizeBytes, limit)
	}

	return nil
}

// MediaDownloader handles downloading media from Telegram
type MediaDownloader struct {
	client *Client
}

// NewMediaDownloader creates a new media downloader
func NewMediaDownloader(client *Client) *MediaDownloader {
	return &MediaDownloader{
		client: client,
	}
}

// DownloadMedia downloads a media file by its file_id
func (d *MediaDownloader) DownloadMedia(fileID string) ([]byte, string, error) {
	return d.client.DownloadFile(fileID)
}

// GetMediaURL gets the download URL for a media file
func (d *MediaDownloader) GetMediaURL(fileID string) (string, error) {
	file, err := d.client.GetFile(fileID)
	if err != nil {
		return "", fmt.Errorf("failed to get file info: %w", err)
	}

	return d.client.GetFileURL(file.FilePath), nil
}

// MediaTypeFromContentType converts content type to Telegram media type
func MediaTypeFromContentType(contentType string) MessageType {
	switch {
	case strings.HasPrefix(contentType, "image/"):
		return MessageTypePhoto
	case strings.HasPrefix(contentType, "video/"):
		return MessageTypeVideo
	case strings.HasPrefix(contentType, "audio/"):
		return MessageTypeAudio
	default:
		return MessageTypeDocument
	}
}
