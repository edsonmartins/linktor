package whatsapp_official

import (
	"context"
	"fmt"
	"mime"
	"path/filepath"
	"strings"

	"github.com/msgfy/linktor/pkg/plugin"
)

// MediaManager handles media operations for WhatsApp
type MediaManager struct {
	client *Client
}

// NewMediaManager creates a new media manager
func NewMediaManager(client *Client) *MediaManager {
	return &MediaManager{
		client: client,
	}
}

// MediaLimits defines the limits for different media types
var MediaLimits = map[string]int64{
	"audio":    16 * 1024 * 1024,  // 16 MB
	"document": 100 * 1024 * 1024, // 100 MB
	"image":    5 * 1024 * 1024,   // 5 MB
	"sticker":  500 * 1024,        // 500 KB
	"video":    16 * 1024 * 1024,  // 16 MB
}

// SupportedMediaTypes defines supported MIME types for each media type
var SupportedMediaTypes = map[string][]string{
	"audio": {
		"audio/aac",
		"audio/mp4",
		"audio/mpeg",
		"audio/amr",
		"audio/ogg",
		"audio/opus",
	},
	"document": {
		"text/plain",
		"application/pdf",
		"application/vnd.ms-powerpoint",
		"application/msword",
		"application/vnd.ms-excel",
		"application/vnd.openxmlformats-officedocument.wordprocessingml.document",
		"application/vnd.openxmlformats-officedocument.presentationml.presentation",
		"application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
	},
	"image": {
		"image/jpeg",
		"image/png",
	},
	"sticker": {
		"image/webp",
	},
	"video": {
		"video/mp4",
		"video/3gp",
	},
}

// UploadMedia uploads media to WhatsApp servers
func (m *MediaManager) UploadMedia(ctx context.Context, media *plugin.Media) (*plugin.MediaUpload, error) {
	// Determine media type from MIME type
	mediaType := m.getMediaType(media.MimeType)
	if mediaType == "" {
		return &plugin.MediaUpload{
			Success: false,
			Error:   fmt.Sprintf("unsupported media type: %s", media.MimeType),
		}, nil
	}

	// Check file size
	limit, ok := MediaLimits[mediaType]
	if ok && media.SizeBytes > limit {
		return &plugin.MediaUpload{
			Success: false,
			Error:   fmt.Sprintf("file size exceeds limit of %d bytes for %s", limit, mediaType),
		}, nil
	}

	// Upload to WhatsApp
	resp, err := m.client.UploadMedia(ctx, media.Filename, media.MimeType, media.Data)
	if err != nil {
		return &plugin.MediaUpload{
			Success: false,
			Error:   err.Error(),
		}, nil
	}

	return &plugin.MediaUpload{
		Success: true,
		MediaID: resp.ID,
	}, nil
}

// DownloadMedia downloads media from WhatsApp servers
func (m *MediaManager) DownloadMedia(ctx context.Context, mediaID string) (*plugin.Media, error) {
	// Get media info
	info, err := m.client.GetMediaInfo(ctx, mediaID)
	if err != nil {
		return nil, fmt.Errorf("failed to get media info: %w", err)
	}

	// Download the actual media content
	data, contentType, err := m.client.DownloadMedia(ctx, info.URL)
	if err != nil {
		return nil, fmt.Errorf("failed to download media: %w", err)
	}

	// Generate filename if not provided
	filename := m.generateFilename(mediaID, contentType)

	return &plugin.Media{
		ID:        mediaID,
		URL:       info.URL,
		Data:      data,
		MimeType:  contentType,
		SizeBytes: info.FileSize,
		Filename:  filename,
		Metadata: map[string]string{
			"sha256":   info.SHA256,
			"media_id": mediaID,
		},
	}, nil
}

// DeleteMedia deletes media from WhatsApp servers
func (m *MediaManager) DeleteMedia(ctx context.Context, mediaID string) error {
	return m.client.DeleteMedia(ctx, mediaID)
}

// GetMediaInfo retrieves media information without downloading
func (m *MediaManager) GetMediaInfo(ctx context.Context, mediaID string) (*MediaInfoResponse, error) {
	return m.client.GetMediaInfo(ctx, mediaID)
}

// ResolveMediaURL gets the download URL for a media ID
func (m *MediaManager) ResolveMediaURL(ctx context.Context, mediaID string) (string, error) {
	info, err := m.client.GetMediaInfo(ctx, mediaID)
	if err != nil {
		return "", err
	}
	return info.URL, nil
}

// getMediaType determines the media type from MIME type
func (m *MediaManager) getMediaType(mimeType string) string {
	for mediaType, mimeTypes := range SupportedMediaTypes {
		for _, mt := range mimeTypes {
			if strings.EqualFold(mimeType, mt) {
				return mediaType
			}
		}
	}

	// Try to guess from base type
	if strings.HasPrefix(mimeType, "image/") {
		return "image"
	}
	if strings.HasPrefix(mimeType, "video/") {
		return "video"
	}
	if strings.HasPrefix(mimeType, "audio/") {
		return "audio"
	}

	// Default to document for other types
	return "document"
}

// generateFilename generates a filename for downloaded media
func (m *MediaManager) generateFilename(mediaID, mimeType string) string {
	// Get extension from MIME type
	exts, err := mime.ExtensionsByType(mimeType)
	ext := ""
	if err == nil && len(exts) > 0 {
		ext = exts[0]
	} else {
		// Fallback extensions
		switch {
		case strings.Contains(mimeType, "jpeg"):
			ext = ".jpg"
		case strings.Contains(mimeType, "png"):
			ext = ".png"
		case strings.Contains(mimeType, "webp"):
			ext = ".webp"
		case strings.Contains(mimeType, "mp4"):
			ext = ".mp4"
		case strings.Contains(mimeType, "ogg"):
			ext = ".ogg"
		case strings.Contains(mimeType, "pdf"):
			ext = ".pdf"
		default:
			ext = ".bin"
		}
	}

	// Use last part of media ID as filename
	shortID := mediaID
	if len(mediaID) > 12 {
		shortID = mediaID[len(mediaID)-12:]
	}

	return fmt.Sprintf("whatsapp_%s%s", shortID, ext)
}

// IsSupportedMediaType checks if a MIME type is supported
func IsSupportedMediaType(mimeType string) bool {
	for _, mimeTypes := range SupportedMediaTypes {
		for _, mt := range mimeTypes {
			if strings.EqualFold(mimeType, mt) {
				return true
			}
		}
	}
	return false
}

// GetMediaTypeFromMimeType returns the WhatsApp media type for a MIME type
func GetMediaTypeFromMimeType(mimeType string) MessageType {
	for mediaType, mimeTypes := range SupportedMediaTypes {
		for _, mt := range mimeTypes {
			if strings.EqualFold(mimeType, mt) {
				return MessageType(mediaType)
			}
		}
	}
	return MessageTypeDocument
}

// GetMaxSize returns the maximum file size for a media type
func GetMaxSize(mediaType string) int64 {
	if limit, ok := MediaLimits[mediaType]; ok {
		return limit
	}
	return MediaLimits["document"]
}

// ValidateMediaSize validates if the file size is within limits
func ValidateMediaSize(mediaType string, size int64) error {
	limit := GetMaxSize(mediaType)
	if size > limit {
		return fmt.Errorf("file size %d exceeds limit of %d bytes for %s", size, limit, mediaType)
	}
	return nil
}

// InferMimeType infers MIME type from filename extension
func InferMimeType(filename string) string {
	ext := strings.ToLower(filepath.Ext(filename))
	mimeType := mime.TypeByExtension(ext)
	if mimeType != "" {
		return mimeType
	}

	// Fallback for common extensions
	switch ext {
	case ".jpg", ".jpeg":
		return "image/jpeg"
	case ".png":
		return "image/png"
	case ".gif":
		return "image/gif"
	case ".webp":
		return "image/webp"
	case ".mp4":
		return "video/mp4"
	case ".3gp":
		return "video/3gp"
	case ".ogg":
		return "audio/ogg"
	case ".mp3":
		return "audio/mpeg"
	case ".aac":
		return "audio/aac"
	case ".amr":
		return "audio/amr"
	case ".pdf":
		return "application/pdf"
	case ".doc":
		return "application/msword"
	case ".docx":
		return "application/vnd.openxmlformats-officedocument.wordprocessingml.document"
	case ".xls":
		return "application/vnd.ms-excel"
	case ".xlsx":
		return "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet"
	case ".ppt":
		return "application/vnd.ms-powerpoint"
	case ".pptx":
		return "application/vnd.openxmlformats-officedocument.presentationml.presentation"
	case ".txt":
		return "text/plain"
	default:
		return "application/octet-stream"
	}
}

// MediaUploadResult contains the result of a media upload
type MediaUploadResult struct {
	MediaID   string
	MediaType string
	MimeType  string
	Filename  string
	Size      int64
	Error     error
}

// BatchUploadMedia uploads multiple media files
func (m *MediaManager) BatchUploadMedia(ctx context.Context, media []*plugin.Media) []MediaUploadResult {
	results := make([]MediaUploadResult, len(media))

	for i, med := range media {
		result := MediaUploadResult{
			MimeType: med.MimeType,
			Filename: med.Filename,
			Size:     med.SizeBytes,
		}

		upload, err := m.UploadMedia(ctx, med)
		if err != nil {
			result.Error = err
		} else if !upload.Success {
			result.Error = fmt.Errorf("%s", upload.Error)
		} else {
			result.MediaID = upload.MediaID
			result.MediaType = m.getMediaType(med.MimeType)
		}

		results[i] = result
	}

	return results
}
