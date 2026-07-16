package handlers

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/msgfy/linktor/internal/infrastructure/storage"
)

// MediaHandler streams stored media objects through the API so message
// attachments can reference a stable, non-expiring URL instead of a presigned
// S3 link (which expires and then breaks the stored reference).
type MediaHandler struct {
	store storage.Client
}

// NewMediaHandler creates a media proxy handler. store may be nil to disable it.
func NewMediaHandler(store storage.Client) *MediaHandler {
	return &MediaHandler{store: store}
}

// Serve streams a stored object by its key.
//
// The key is an opaque, UUID-bearing path, so this endpoint is a capability URL
// with the same trust model as the presigned URLs it replaces: possession of
// the unguessable path is the authorization. That lets an <img>/<video> tag load
// it directly without carrying a session across origins.
func (h *MediaHandler) Serve(c *gin.Context) {
	if h.store == nil {
		c.Status(http.StatusServiceUnavailable)
		return
	}

	key := strings.TrimPrefix(c.Param("key"), "/")
	// Guard against path traversal for filesystem-backed stores.
	if key == "" || strings.Contains(key, "..") {
		c.Status(http.StatusBadRequest)
		return
	}

	reader, contentType, size, err := h.store.Open(c.Request.Context(), key)
	if err != nil {
		c.Status(http.StatusNotFound)
		return
	}
	defer reader.Close()

	if contentType == "" {
		contentType = "application/octet-stream"
	}
	// Content is addressed by an opaque immutable key — safe to cache hard.
	c.Header("Cache-Control", "private, max-age=31536000, immutable")
	c.DataFromReader(http.StatusOK, size, contentType, reader, nil)
}
