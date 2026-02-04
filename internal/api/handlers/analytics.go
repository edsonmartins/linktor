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

// GetOverview returns high-level analytics metrics
// GET /api/v1/analytics/overview
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

// GetConversations returns conversation analytics by day
// GET /api/v1/analytics/conversations
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

// GetFlows returns flow analytics
// GET /api/v1/analytics/flows
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

// GetEscalations returns escalation analytics by reason
// GET /api/v1/analytics/escalations
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

// GetChannels returns channel analytics
// GET /api/v1/analytics/channels
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
