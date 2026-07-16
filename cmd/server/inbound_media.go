package main

import (
	"context"
	"encoding/base64"
	"fmt"
	"mime"
	"os"
	"strings"

	"github.com/google/uuid"
	"github.com/msgfy/linktor/internal/infrastructure/nats"
	"github.com/msgfy/linktor/internal/infrastructure/storage"
	"github.com/msgfy/linktor/pkg/logger"
)

// mediaPublicBaseURL is the public origin the backend is reachable at (e.g.
// "https://api.linktor.dev"), used to build stable media-proxy URLs. Prefers an
// explicit MEDIA_PUBLIC_BASE_URL, then the existing BASE_URL (already set to the
// public API origin in deploys). When empty the store falls back to presigned
// S3 URLs.
func mediaPublicBaseURL() string {
	if v := strings.TrimRight(os.Getenv("MEDIA_PUBLIC_BASE_URL"), "/"); v != "" {
		return v
	}
	return strings.TrimRight(os.Getenv("BASE_URL"), "/")
}

// rehostInboundMedia turns inbound attachments that carry their bytes inline as
// base64 (metadata["data"]) into a browser-usable object-store URL.
//
// Some channel adapters (notably whatsmeow) download+decrypt media eagerly and
// hand it to us as base64 in the attachment metadata, with an empty URL, because
// the provider's CDN URL is ciphertext. Persisting that verbatim leaves the chat
// with an empty <img src> (broken image) and bloats every message row with the
// full image. Here we decode the bytes once, upload them to the media store, set
// the attachment URL, and drop the inline blob from the metadata.
//
// If no store is configured or an upload fails, the attachment is left untouched
// so behaviour degrades to the previous (base64-in-metadata) state rather than
// dropping the media.
func rehostInboundMedia(ctx context.Context, store storage.Client, msg *nats.InboundMessage) {
	if store == nil || msg == nil {
		return
	}

	for i := range msg.Attachments {
		att := &msg.Attachments[i]
		if att.URL != "" || att.Metadata == nil {
			continue
		}
		if att.Metadata["data_encoding"] != "base64" {
			continue
		}
		b64 := att.Metadata["data"]
		if b64 == "" {
			continue
		}

		data, err := base64.StdEncoding.DecodeString(b64)
		if err != nil {
			continue
		}

		contentType := att.MimeType
		if contentType == "" {
			contentType = "application/octet-stream"
		}

		key := fmt.Sprintf("inbound/%s/%s/%s%s",
			msg.ChannelType, msg.ChannelID, uuid.New().String(), extensionForMime(contentType, att.Filename))

		url, err := store.Upload(ctx, key, data, contentType)
		if err != nil {
			logger.Warn("inbound media re-host failed: " + err.Error())
			continue
		}

		att.URL = url
		if att.SizeBytes == 0 {
			att.SizeBytes = int64(len(data))
		}
		// Drop the inline blob now that a URL exists so the message row and API
		// responses don't carry the whole file as base64.
		delete(att.Metadata, "data")
		delete(att.Metadata, "data_encoding")
	}
}

// extensionForMime picks a file extension for the stored object, preferring the
// original filename's extension and falling back to the MIME type.
func extensionForMime(mimeType, filename string) string {
	if dot := strings.LastIndex(filename, "."); dot >= 0 && dot < len(filename)-1 {
		return filename[dot:]
	}
	if exts, err := mime.ExtensionsByType(mimeType); err == nil && len(exts) > 0 {
		return exts[0]
	}
	return ""
}
