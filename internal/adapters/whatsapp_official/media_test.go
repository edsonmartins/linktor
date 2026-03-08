package whatsapp_official

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMediaManager_getMediaType(t *testing.T) {
	mm := &MediaManager{client: nil}

	tests := []struct {
		name     string
		mimeType string
		expected string
	}{
		// Exact matches from SupportedMediaTypes
		{name: "audio/aac", mimeType: "audio/aac", expected: "audio"},
		{name: "audio/mp4", mimeType: "audio/mp4", expected: "audio"},
		{name: "audio/mpeg", mimeType: "audio/mpeg", expected: "audio"},
		{name: "image/jpeg", mimeType: "image/jpeg", expected: "image"},
		{name: "image/png", mimeType: "image/png", expected: "image"},
		{name: "image/webp is sticker", mimeType: "image/webp", expected: "sticker"},
		{name: "video/mp4", mimeType: "video/mp4", expected: "video"},
		{name: "video/3gp", mimeType: "video/3gp", expected: "video"},
		{name: "application/pdf", mimeType: "application/pdf", expected: "document"},
		{name: "text/plain", mimeType: "text/plain", expected: "document"},

		// Fallback by prefix (not in SupportedMediaTypes but matched by prefix)
		{name: "image/gif falls back to image by prefix", mimeType: "image/gif", expected: "image"},
		{name: "video/webm falls back to video by prefix", mimeType: "video/webm", expected: "video"},
		{name: "audio/wav falls back to audio by prefix", mimeType: "audio/wav", expected: "audio"},

		// Unknown type defaults to document
		{name: "application/octet-stream defaults to document", mimeType: "application/octet-stream", expected: "document"},

		// Case insensitive matching
		{name: "IMAGE/JPEG case insensitive", mimeType: "IMAGE/JPEG", expected: "image"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := mm.getMediaType(tt.mimeType)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestValidateMediaSize(t *testing.T) {
	tests := []struct {
		name      string
		mediaType string
		size      int64
		wantErr   bool
	}{
		{name: "image within limit", mediaType: "image", size: 4 * 1024 * 1024, wantErr: false},
		{name: "image exceeds limit", mediaType: "image", size: 6 * 1024 * 1024, wantErr: true},
		{name: "audio within limit", mediaType: "audio", size: 15 * 1024 * 1024, wantErr: false},
		{name: "audio exceeds limit", mediaType: "audio", size: 17 * 1024 * 1024, wantErr: true},
		{name: "document within limit", mediaType: "document", size: 99 * 1024 * 1024, wantErr: false},
		{name: "document exceeds limit", mediaType: "document", size: 101 * 1024 * 1024, wantErr: true},
		{name: "sticker within limit", mediaType: "sticker", size: 400 * 1024, wantErr: false},
		{name: "sticker exceeds limit", mediaType: "sticker", size: 600 * 1024, wantErr: true},
		{name: "video within limit", mediaType: "video", size: 15 * 1024 * 1024, wantErr: false},
		{name: "unknown type uses document limit", mediaType: "unknown", size: 99 * 1024 * 1024, wantErr: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateMediaSize(tt.mediaType, tt.size)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestIsSupportedMediaType(t *testing.T) {
	tests := []struct {
		name     string
		mimeType string
		expected bool
	}{
		{name: "audio/aac supported", mimeType: "audio/aac", expected: true},
		{name: "image/jpeg supported", mimeType: "image/jpeg", expected: true},
		{name: "image/png supported", mimeType: "image/png", expected: true},
		{name: "image/webp supported", mimeType: "image/webp", expected: true},
		{name: "video/mp4 supported", mimeType: "video/mp4", expected: true},
		{name: "application/pdf supported", mimeType: "application/pdf", expected: true},
		{name: "image/gif not supported", mimeType: "image/gif", expected: false},
		{name: "video/webm not supported", mimeType: "video/webm", expected: false},
		{name: "application/zip not supported", mimeType: "application/zip", expected: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsSupportedMediaType(tt.mimeType)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestInferMimeType(t *testing.T) {
	// Some extensions may resolve via the system's mime.TypeByExtension before
	// hitting the fallback switch. We therefore allow multiple accepted values
	// for types that vary across platforms.
	tests := []struct {
		name     string
		filename string
		accepted []string // any of these base MIME types is acceptable
	}{
		{name: ".jpg extension", filename: "photo.jpg", accepted: []string{"image/jpeg"}},
		{name: ".jpeg extension", filename: "photo.jpeg", accepted: []string{"image/jpeg"}},
		{name: ".png extension", filename: "photo.png", accepted: []string{"image/png"}},
		{name: ".gif extension", filename: "anim.gif", accepted: []string{"image/gif"}},
		{name: ".webp extension", filename: "sticker.webp", accepted: []string{"image/webp"}},
		{name: ".mp4 extension", filename: "video.mp4", accepted: []string{"video/mp4"}},
		{name: ".3gp extension", filename: "video.3gp", accepted: []string{"video/3gp", "video/3gpp"}},
		{name: ".ogg extension", filename: "audio.ogg", accepted: []string{"audio/ogg", "audio/x-ogg", "application/ogg"}},
		{name: ".mp3 extension", filename: "song.mp3", accepted: []string{"audio/mpeg", "audio/mp3"}},
		{name: ".aac extension", filename: "audio.aac", accepted: []string{"audio/aac", "audio/x-aac"}},
		{name: ".amr extension", filename: "voice.amr", accepted: []string{"audio/amr", "audio/x-amr"}},
		{name: ".pdf extension", filename: "doc.pdf", accepted: []string{"application/pdf"}},
		{name: ".doc extension", filename: "file.doc", accepted: []string{"application/msword"}},
		{name: ".docx extension", filename: "file.docx", accepted: []string{"application/vnd.openxmlformats-officedocument.wordprocessingml.document"}},
		{name: ".txt extension", filename: "readme.txt", accepted: []string{"text/plain"}},
		{name: ".unknown extension", filename: "file.unknown", accepted: []string{"application/octet-stream"}},
		{name: "no extension", filename: "noext", accepted: []string{"application/octet-stream"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := InferMimeType(tt.filename)
			// Strip parameters like "; charset=utf-8"
			resultBase := strings.TrimSpace(strings.Split(result, ";")[0])
			assert.Contains(t, tt.accepted, resultBase,
				"InferMimeType(%q) = %q, expected one of %v", tt.filename, resultBase, tt.accepted)
		})
	}
}

func TestMediaManager_generateFilename(t *testing.T) {
	mm := &MediaManager{client: nil}

	t.Run("basic image/jpeg", func(t *testing.T) {
		result := mm.generateFilename("abc123", "image/jpeg")
		assert.True(t, strings.HasPrefix(result, "whatsapp_"), "should start with whatsapp_")
		assert.Contains(t, result, "abc123", "should contain the media ID")
		// Extension could be .jpg, .jpeg, .jpe, .jfif depending on platform
		ext := result[strings.LastIndex(result, "."):]
		assert.True(t, ext == ".jpg" || ext == ".jpeg" || ext == ".jpe" || ext == ".jfif",
			"expected a jpeg-related extension, got %s", ext)
	})

	t.Run("mediaID longer than 12 chars is truncated", func(t *testing.T) {
		longID := "abcdefghijklmnopqrstuvwxyz"
		result := mm.generateFilename(longID, "image/png")
		assert.True(t, strings.HasPrefix(result, "whatsapp_"), "should start with whatsapp_")
		// The last 12 chars of longID are "opqrstuvwxyz"
		assert.Contains(t, result, "opqrstuvwxyz", "should contain the last 12 chars of the ID")
		assert.NotContains(t, result, "abcdefghijklmn", "should not contain the beginning of the long ID")
	})

	t.Run("mediaID exactly 12 chars is not truncated", func(t *testing.T) {
		exactID := "abcdefghijkl" // exactly 12 chars
		result := mm.generateFilename(exactID, "image/png")
		assert.True(t, strings.HasPrefix(result, "whatsapp_"), "should start with whatsapp_")
		assert.Contains(t, result, "abcdefghijkl", "should contain the full 12-char ID")
	})

	t.Run("unknown mimeType gets .bin extension", func(t *testing.T) {
		result := mm.generateFilename("test123", "application/x-unknown-type")
		assert.True(t, strings.HasPrefix(result, "whatsapp_"), "should start with whatsapp_")
		assert.True(t, strings.HasSuffix(result, ".bin"), "should end with .bin for unknown type")
	})
}

func TestGetMediaTypeFromMimeType(t *testing.T) {
	tests := []struct {
		name     string
		mimeType string
		expected MessageType
	}{
		{name: "image/jpeg", mimeType: "image/jpeg", expected: MessageType("image")},
		{name: "video/mp4", mimeType: "video/mp4", expected: MessageType("video")},
		{name: "audio/aac", mimeType: "audio/aac", expected: MessageType("audio")},
		{name: "application/pdf", mimeType: "application/pdf", expected: MessageType("document")},
		{name: "unknown falls back to document", mimeType: "application/x-foo", expected: MessageTypeDocument},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetMediaTypeFromMimeType(tt.mimeType)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGetMaxSize(t *testing.T) {
	tests := []struct {
		name      string
		mediaType string
		expected  int64
	}{
		{name: "image", mediaType: "image", expected: 5 * 1024 * 1024},
		{name: "audio", mediaType: "audio", expected: 16 * 1024 * 1024},
		{name: "document", mediaType: "document", expected: 100 * 1024 * 1024},
		{name: "sticker", mediaType: "sticker", expected: 500 * 1024},
		{name: "video", mediaType: "video", expected: 16 * 1024 * 1024},
		{name: "unknown defaults to document limit", mediaType: "unknown", expected: 100 * 1024 * 1024},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetMaxSize(tt.mediaType)
			assert.Equal(t, tt.expected, result)
		})
	}
}
