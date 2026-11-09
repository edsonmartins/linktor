package handlers

import (
	stderrors "errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/msgfy/linktor/internal/api/middleware"
	"github.com/msgfy/linktor/pkg/errors"
	"github.com/msgfy/linktor/pkg/plugin"
)

// GetContactAvatar godoc
// @Summary      Profile picture of a WhatsApp contact
// @Description  Looks up the contact's profile picture on the channel's WhatsApp session (embedded or Linktor Bridge) and returns the image bytes. Nothing is stored: neither the CDN URL nor the image. Send the pictureId you already hold in If-None-Match to get 304 when it is still current. A contact without a picture and a contact hiding it from this number are both 404, told apart by "reason" (not_set | hidden).
// @Tags         channels
// @Produce      image/jpeg
// @Security     BearerAuth
// @Param        id             path   string true  "Channel ID"
// @Param        contact        path   string true  "Phone in digits, <number>@s.whatsapp.net or <id>@lid"
// @Param        If-None-Match  header string false "pictureId already held"
// @Success      200 {file} binary "image bytes; X-Picture-Id and ETag carry the pictureId"
// @Success      304 "the pictureId in If-None-Match is still current"
// @Failure      404 {object} map[string]interface{} "reason: not_set | hidden, or channel not found"
// @Failure      409 {object} Response "no live WhatsApp session"
// @Failure      429 {object} Response "per-channel rate limit; see Retry-After"
// @Failure      501 {object} Response "channel type has no contact avatar"
// @Router       /channels/{id}/contacts/{contact}/avatar [get]
func (h *ChannelHandler) GetContactAvatar(c *gin.Context) {
	id := c.Param("id")
	contact := c.Param("contact")
	if id == "" || contact == "" {
		RespondValidationError(c, "Channel ID and contact are required", nil)
		return
	}

	tenantID := middleware.MustGetTenantID(c)
	if tenantID == "" {
		return
	}

	existingID := pictureIDFromIfNoneMatch(c.GetHeader("If-None-Match"))

	avatar, err := h.channelService.GetContactAvatar(c.Request.Context(), tenantID, id, contact, existingID)
	if err != nil {
		respondContactAvatarError(c, err)
		return
	}

	c.Header("X-Picture-Id", avatar.PictureID)
	c.Header("ETag", `"`+avatar.PictureID+`"`)
	// A foto é dado pessoal e muda sem aviso: nenhum cache intermediário guarda.
	c.Header("Cache-Control", "private, no-store")
	if avatar.Unchanged {
		c.Status(http.StatusNotModified)
		return
	}
	c.Data(http.StatusOK, avatar.ContentType, avatar.Image)
}

// respondContactAvatarError põe o motivo do 404 no corpo e o Retry-After no 429;
// os demais erros seguem o envelope padrão.
func respondContactAvatarError(c *gin.Context, err error) {
	var reason string
	switch {
	case stderrors.Is(err, plugin.ErrAvatarHidden):
		reason = "hidden"
	case stderrors.Is(err, plugin.ErrAvatarNotSet):
		reason = "not_set"
	}
	if reason != "" {
		c.JSON(http.StatusNotFound, gin.H{
			"success": false,
			"reason":  reason,
			"error": &ErrorResponse{
				Code:    "AVATAR_" + strings.ToUpper(reason),
				Message: err.Error(),
			},
		})
		return
	}
	if appErr := errors.GetAppError(err); appErr != nil && appErr.Code == errors.ErrCodeRateLimited {
		if retry := appErr.Details["retryAfter"]; retry != "" {
			c.Header("Retry-After", retry)
		}
	}
	RespondError(c, err)
}

// pictureIDFromIfNoneMatch tira o pictureId do If-None-Match: aceita o valor
// entre aspas (ETag forte ou W/), sem aspas, ou o primeiro de uma lista. "*" não
// identifica foto nenhuma e é ignorado.
func pictureIDFromIfNoneMatch(header string) string {
	v := strings.TrimSpace(header)
	if i := strings.IndexByte(v, ','); i >= 0 {
		v = strings.TrimSpace(v[:i])
	}
	v = strings.TrimPrefix(v, "W/")
	v = strings.Trim(v, `"`)
	if v == "*" {
		return ""
	}
	return v
}
