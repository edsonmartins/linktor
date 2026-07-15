package handlers

import (
	"fmt"
	"io"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/msgfy/linktor/internal/api/middleware"
	storagelib "github.com/msgfy/linktor/internal/infrastructure/storage"
)

// maxAttachmentBytes caps a single uploaded attachment (25 MiB), matching the
// practical media limits of the downstream channels (e.g. WhatsApp).
const maxAttachmentBytes = 25 << 20

// AttachmentHandler handles uploading media that agents attach to outgoing
// messages. The uploaded object is stored in the shared media store and its
// public URL is returned; the client then references it when sending a message.
type AttachmentHandler struct {
	store storagelib.Client
}

// NewAttachmentHandler creates a new attachment handler. store may be nil when
// no object storage is configured, in which case uploads are rejected.
func NewAttachmentHandler(store storagelib.Client) *AttachmentHandler {
	return &AttachmentHandler{store: store}
}

// UploadAttachmentResponse is returned after a successful upload and mirrors the
// fields a subsequent send-message request expects under `attachments`.
type UploadAttachmentResponse struct {
	URL       string `json:"url"`
	Type      string `json:"type"`
	Filename  string `json:"filename"`
	MimeType  string `json:"mime_type"`
	SizeBytes int64  `json:"size_bytes"`
}

// Upload godoc
// @Summary      Upload a message attachment
// @Description  Stores an uploaded file and returns its public URL + metadata for use in a send-message request
// @Tags         messages
// @Accept       multipart/form-data
// @Produce      json
// @Security     BearerAuth
// @Param        id path string true "Conversation ID"
// @Param        file formData file true "File to upload"
// @Success      201 {object} Response{data=UploadAttachmentResponse}
// @Failure      400 {object} Response
// @Failure      401 {object} Response
// @Failure      413 {object} Response
// @Failure      503 {object} Response
// @Router       /conversations/{id}/attachments [post]
func (h *AttachmentHandler) Upload(c *gin.Context) {
	tenantID := middleware.MustGetTenantID(c)
	if tenantID == "" {
		return
	}

	if h.store == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "attachment storage is not configured"})
		return
	}

	fileHeader, err := c.FormFile("file")
	if err != nil {
		RespondValidationError(c, "A file is required", nil)
		return
	}
	if fileHeader.Size > maxAttachmentBytes {
		c.JSON(http.StatusRequestEntityTooLarge, gin.H{"error": "file exceeds the 25 MB limit"})
		return
	}

	file, err := fileHeader.Open()
	if err != nil {
		RespondValidationError(c, "Could not read the uploaded file", nil)
		return
	}
	defer file.Close()

	data, err := io.ReadAll(io.LimitReader(file, maxAttachmentBytes+1))
	if err != nil {
		RespondValidationError(c, "Could not read the uploaded file", nil)
		return
	}
	if int64(len(data)) > maxAttachmentBytes {
		c.JSON(http.StatusRequestEntityTooLarge, gin.H{"error": "file exceeds the 25 MB limit"})
		return
	}

	mimeType := fileHeader.Header.Get("Content-Type")
	if mimeType == "" {
		mimeType = http.DetectContentType(data)
	}

	ext := filepath.Ext(fileHeader.Filename)
	key := fmt.Sprintf("attachments/%s/%s%s", tenantID, uuid.New().String(), ext)

	url, err := h.store.Upload(c.Request.Context(), key, data, mimeType)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to store attachment"})
		return
	}

	RespondCreated(c, UploadAttachmentResponse{
		URL:       url,
		Type:      attachmentTypeFromMime(mimeType),
		Filename:  fileHeader.Filename,
		MimeType:  mimeType,
		SizeBytes: int64(len(data)),
	})
}

// attachmentTypeFromMime maps a MIME type to the coarse attachment type used by
// the message/entity model and the channel adapters.
func attachmentTypeFromMime(mimeType string) string {
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
