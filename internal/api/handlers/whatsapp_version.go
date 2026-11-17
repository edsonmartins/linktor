package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/msgfy/linktor/internal/adapters/whatsapp"
)

// GetWhatsAppClientVersion godoc
// @Summary      WhatsApp Web client version health
// @Description  Build announced by the unofficial WhatsApp channels, the latest build seen on web.whatsapp.com and whether WhatsApp refused ours (405 client outdated)
// @Tags         observability
// @Produce      json
// @Success      200 {object} whatsapp.VersionStatus
// @Router       /observability/whatsapp-version [get]
func GetWhatsAppClientVersion(c *gin.Context) {
	c.JSON(http.StatusOK, whatsapp.GetVersionStatus())
}

// CheckWhatsAppClientVersion godoc
// @Summary      Re-check the WhatsApp Web client version now
// @Tags         observability
// @Produce      json
// @Success      200 {object} whatsapp.VersionStatus
// @Router       /observability/whatsapp-version/check [post]
func CheckWhatsAppClientVersion(c *gin.Context) {
	c.JSON(http.StatusOK, whatsapp.RefreshVersion(c.Request.Context()))
}
