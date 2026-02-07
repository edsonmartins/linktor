package handlers

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/msgfy/linktor/internal/api/middleware"
	"github.com/msgfy/linktor/internal/application/service"
	"github.com/msgfy/linktor/internal/domain/entity"
)

// AnalyticsHandler handles analytics endpoints
type AnalyticsHandler struct {
	analyticsService *service.AnalyticsService
}

// NewAnalyticsHandler creates a new analytics handler
func NewAnalyticsHandler(analyticsService *service.AnalyticsService) *AnalyticsHandler {
	return &AnalyticsHandler{
		analyticsService: analyticsService,
	}
}

// parseAnalyticsParams extracts common analytics parameters from the request
func (h *AnalyticsHandler) parseAnalyticsParams(c *gin.Context) (entity.AnalyticsPeriod, time.Time, time.Time) {
	periodStr := c.DefaultQuery("period", "weekly")
	period := entity.AnalyticsPeriod(periodStr)

	var startDate, endDate time.Time
	var customStart, customEnd *time.Time

	if startStr := c.Query("start_date"); startStr != "" {
		if t, err := time.Parse("2006-01-02", startStr); err == nil {
			customStart = &t
		}
	}

	if endStr := c.Query("end_date"); endStr != "" {
		if t, err := time.Parse("2006-01-02", endStr); err == nil {
			// Set end date to end of day
			t = t.Add(24*time.Hour - time.Second)
			customEnd = &t
		}
	}

	startDate, endDate = h.analyticsService.GetDateRange(period, customStart, customEnd)
	return period, startDate, endDate
}

// GetOverview godoc
// @Summary      Get analytics overview
// @Description  Returns high-level analytics metrics including message counts, response times, and conversation statistics
// @Tags         analytics
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        period query string false "Time period (daily, weekly, monthly)" default(weekly)
// @Param        start_date query string false "Custom start date (YYYY-MM-DD)"
// @Param        end_date query string false "Custom end date (YYYY-MM-DD)"
// @Success      200 {object} Response{data=entity.AnalyticsOverview}
// @Failure      401 {object} Response
// @Failure      500 {object} Response
// @Router       /analytics/overview [get]
func (h *AnalyticsHandler) GetOverview(c *gin.Context) {
	tenantID := middleware.GetTenantID(c)
	period, startDate, endDate := h.parseAnalyticsParams(c)

	overview, err := h.analyticsService.GetOverview(c.Request.Context(), tenantID, period, startDate, endDate)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get overview analytics"})
		return
	}

	c.JSON(http.StatusOK, overview)
}

// GetConversations godoc
// @Summary      Get conversation analytics
// @Description  Returns conversation analytics grouped by day
// @Tags         analytics
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        period query string false "Time period (daily, weekly, monthly)" default(weekly)
// @Param        start_date query string false "Custom start date (YYYY-MM-DD)"
// @Param        end_date query string false "Custom end date (YYYY-MM-DD)"
// @Success      200 {object} Response{data=[]entity.ConversationsByDay}
// @Failure      401 {object} Response
// @Failure      500 {object} Response
// @Router       /analytics/conversations [get]
func (h *AnalyticsHandler) GetConversations(c *gin.Context) {
	tenantID := middleware.GetTenantID(c)
	_, startDate, endDate := h.parseAnalyticsParams(c)

	conversations, err := h.analyticsService.GetConversationsByDay(c.Request.Context(), tenantID, startDate, endDate)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get conversation analytics"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": conversations})
}

// GetFlows godoc
// @Summary      Get flow analytics
// @Description  Returns analytics for conversation flows including execution counts and success rates
// @Tags         analytics
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        period query string false "Time period (daily, weekly, monthly)" default(weekly)
// @Param        start_date query string false "Custom start date (YYYY-MM-DD)"
// @Param        end_date query string false "Custom end date (YYYY-MM-DD)"
// @Success      200 {object} Response{data=[]entity.FlowAnalytics}
// @Failure      401 {object} Response
// @Failure      500 {object} Response
// @Router       /analytics/flows [get]
func (h *AnalyticsHandler) GetFlows(c *gin.Context) {
	tenantID := middleware.GetTenantID(c)
	_, startDate, endDate := h.parseAnalyticsParams(c)

	flows, err := h.analyticsService.GetFlowAnalytics(c.Request.Context(), tenantID, startDate, endDate)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get flow analytics"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": flows})
}

// GetEscalations godoc
// @Summary      Get escalation analytics
// @Description  Returns escalation analytics grouped by reason
// @Tags         analytics
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        period query string false "Time period (daily, weekly, monthly)" default(weekly)
// @Param        start_date query string false "Custom start date (YYYY-MM-DD)"
// @Param        end_date query string false "Custom end date (YYYY-MM-DD)"
// @Success      200 {object} Response{data=[]entity.EscalationsByReason}
// @Failure      401 {object} Response
// @Failure      500 {object} Response
// @Router       /analytics/escalations [get]
func (h *AnalyticsHandler) GetEscalations(c *gin.Context) {
	tenantID := middleware.GetTenantID(c)
	_, startDate, endDate := h.parseAnalyticsParams(c)

	escalations, err := h.analyticsService.GetEscalationsByReason(c.Request.Context(), tenantID, startDate, endDate)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get escalation analytics"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": escalations})
}

// GetChannels godoc
// @Summary      Get channel analytics
// @Description  Returns analytics for each channel including message counts and engagement metrics
// @Tags         analytics
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        period query string false "Time period (daily, weekly, monthly)" default(weekly)
// @Param        start_date query string false "Custom start date (YYYY-MM-DD)"
// @Param        end_date query string false "Custom end date (YYYY-MM-DD)"
// @Success      200 {object} Response{data=[]entity.ChannelAnalytics}
// @Failure      401 {object} Response
// @Failure      500 {object} Response
// @Router       /analytics/channels [get]
func (h *AnalyticsHandler) GetChannels(c *gin.Context) {
	tenantID := middleware.GetTenantID(c)
	_, startDate, endDate := h.parseAnalyticsParams(c)

	channels, err := h.analyticsService.GetChannelAnalytics(c.Request.Context(), tenantID, startDate, endDate)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get channel analytics"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": channels})
}
