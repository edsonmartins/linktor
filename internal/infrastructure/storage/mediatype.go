package storage

import "strings"

// AttachmentTypeFromMime reduz um MIME ao tipo grosso usado em attachments[]
// (image, video, audio, document). É a regra única para uploads do atendente,
// do widget e das mídias que chegam pelos canais.
func AttachmentTypeFromMime(mimeType string) string {
	mimeType = strings.ToLower(strings.TrimSpace(mimeType))
	switch {
	case strings.HasPrefix(mimeType, "image/"):
		return "image"
	case strings.HasPrefix(mimeType, "video/"):
		return "video"
	case strings.HasPrefix(mimeType, "audio/"):
		return "audio"
	default:
		return "document"
	}
}
