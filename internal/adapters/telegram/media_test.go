package telegram

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetMimeTypeFromFilename(t *testing.T) {
	tests := []struct {
		filename string
		expected string
	}{
		{"photo.jpg", "image/jpeg"},
		{"photo.jpeg", "image/jpeg"},
		{"image.png", "image/png"},
		{"anim.gif", "image/gif"},
		{"sticker.webp", "image/webp"},
		{"video.mp4", "video/mp4"},
		{"clip.webm", "video/webm"},
		{"song.mp3", "audio/mpeg"},
		{"voice.ogg", "audio/ogg"},
		{"audio.m4a", "audio/mp4"},
		{"report.pdf", "application/pdf"},
		{"letter.doc", "application/msword"},
		{"letter.docx", "application/vnd.openxmlformats-officedocument.wordprocessingml.document"},
		{"data.xls", "application/vnd.ms-excel"},
		{"data.xlsx", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet"},
		{"archive.zip", "application/zip"},
		{"notes.txt", "text/plain"},
		// Case insensitivity
		{"PHOTO.JPG", "image/jpeg"},
		{"Video.MP4", "video/mp4"},
		// Unknown extension
		{"file.xyz", "application/octet-stream"},
		{"noext", "application/octet-stream"},
		{"", "application/octet-stream"},
	}

	for _, tc := range tests {
		t.Run(tc.filename, func(t *testing.T) {
			assert.Equal(t, tc.expected, GetMimeTypeFromFilename(tc.filename))
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
		// Photo limits (10MB)
		{"photo within limit", "photo", 5 * 1024 * 1024, false},
		{"photo at limit", "photo", 10 * 1024 * 1024, false},
		{"photo over limit", "photo", 10*1024*1024 + 1, true},

		// Video limits (50MB)
		{"video within limit", "video", 30 * 1024 * 1024, false},
		{"video at limit", "video", 50 * 1024 * 1024, false},
		{"video over limit", "video", 50*1024*1024 + 1, true},

		// Audio limits (50MB)
		{"audio within limit", "audio", 10 * 1024 * 1024, false},
		{"audio at limit", "audio", 50 * 1024 * 1024, false},
		{"audio over limit", "audio", 50*1024*1024 + 1, true},

		// Document limits (50MB)
		{"document within limit", "document", 25 * 1024 * 1024, false},
		{"document at limit", "document", 50 * 1024 * 1024, false},
		{"document over limit", "document", 50*1024*1024 + 1, true},

		// Sticker limits (512KB)
		{"sticker within limit", "sticker", 256 * 1024, false},
		{"sticker at limit", "sticker", 512 * 1024, false},
		{"sticker over limit", "sticker", 512*1024 + 1, true},

		// Unknown type defaults to document limit
		{"unknown within limit", "unknown_type", 40 * 1024 * 1024, false},
		{"unknown over limit", "unknown_type", 50*1024*1024 + 1, true},

		// Zero size
		{"zero size", "photo", 0, false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := ValidateMediaSize(tc.mediaType, tc.size)
			if tc.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), "exceeds limit")
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestMediaTypeFromContentType(t *testing.T) {
	tests := []struct {
		contentType string
		expected    MessageType
	}{
		{"image/jpeg", MessageTypePhoto},
		{"image/png", MessageTypePhoto},
		{"image/gif", MessageTypePhoto},
		{"image/webp", MessageTypePhoto},
		{"video/mp4", MessageTypeVideo},
		{"video/webm", MessageTypeVideo},
		{"audio/mpeg", MessageTypeAudio},
		{"audio/ogg", MessageTypeAudio},
		{"audio/mp4", MessageTypeAudio},
		{"application/pdf", MessageTypeDocument},
		{"application/zip", MessageTypeDocument},
		{"text/plain", MessageTypeDocument},
		{"", MessageTypeDocument},
	}

	for _, tc := range tests {
		t.Run(tc.contentType, func(t *testing.T) {
			assert.Equal(t, tc.expected, MediaTypeFromContentType(tc.contentType))
		})
	}
}
