package handlers

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/msgfy/linktor/internal/api/middleware"
	"github.com/msgfy/linktor/internal/application/service"
	"github.com/msgfy/linktor/internal/domain/entity"
)

// ObservabilityHandler handles observability endpoints
type ObservabilityHandler struct {
	observabilityService *service.ObservabilityService
}

// NewObservabilityHandler creates a new observability handler
func NewObservabilityHandler(observabilityService *service.ObservabilityService) *ObservabilityHandler {
	return &ObservabilityHandler{
		observabilityService: observabilityService,
	}
}

// GetLogs godoc
// @Summary      Get logs
// @Description  Returns system logs with filtering and pagination
// @Tags         observability
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        channel_id query string false "Filter by channel ID"
// @Param        level query string false "Filter by log level (info, warn, error)"
// @Param        source query string false "Filter by source (channel, queue, system, webhook)"
// @Param        start_date query string false "Start date (RFC3339)"
// @Param        end_date query string false "End date (RFC3339)"
// @Param        limit query int false "Limit results" default(50)
// @Param        offset query int false "Offset for pagination" default(0)
// @Success      200 {object} Response{data=entity.LogsResponse}
// @Failure      401 {object} Response
// @Failure      500 {object} Response
// @Router       /observability/logs [get]
func (h *ObservabilityHandler) GetLogs(c *gin.Context) {
	tenantID := middleware.GetTenantID(c)

	// Parse query parameters
	channelID := c.Query("channel_id")
	level := entity.LogLevel(c.Query("level"))
	source := entity.LogSource(c.Query("source"))

	// Parse dates
	var startDate, endDate time.Time
	if startStr := c.Query("start_date"); startStr != "" {
		if t, err := time.Parse(time.RFC3339, startStr); err == nil {
			startDate = t
		}
	}
	if endStr := c.Query("end_date"); endStr != "" {
		if t, err := time.Parse(time.RFC3339, endStr); err == nil {
			endDate = t
		}
	}

	// If no dates provided, default to last 24 hours
	if startDate.IsZero() {
		startDate = time.Now().Add(-24 * time.Hour)
	}
	if endDate.IsZero() {
		endDate = time.Now()
	}

	// Parse pagination
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))

	logs, err := h.observabilityService.GetLogs(c.Request.Context(), tenantID, channelID, level, source, startDate, endDate, limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get logs"})
		return
	}

	c.JSON(http.StatusOK, logs)
}

// GetQueueStats godoc
// @Summary      Get queue statistics
// @Description  Returns NATS message queue statistics including stream and consumer information
// @Tags         observability
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Success      200 {object} Response{data=entity.QueueStats}
// @Failure      401 {object} Response
// @Failure      500 {object} Response
// @Router       /observability/queue [get]
func (h *ObservabilityHandler) GetQueueStats(c *gin.Context) {
	stats, err := h.observabilityService.GetQueueStats(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get queue stats"})
		return
	}

	c.JSON(http.StatusOK, stats)
}

// GetStreamInfo godoc
// @Summary      Get stream info
// @Description  Returns detailed information about a specific NATS stream
// @Tags         observability
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        stream path string true "Stream name"
// @Success      200 {object} Response{data=entity.StreamInfo}
// @Failure      401 {object} Response
// @Failure      404 {object} Response
// @Failure      500 {object} Response
// @Router       /observability/queue/{stream} [get]
func (h *ObservabilityHandler) GetStreamInfo(c *gin.Context) {
	streamName := c.Param("stream")

	info, err := h.observabilityService.GetStreamInfo(c.Request.Context(), streamName)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get stream info"})
		return
	}

	c.JSON(http.StatusOK, info)
}

// ResetConsumer godoc
// @Summary      Reset consumer
// @Description  Reset a NATS consumer to reprocess messages from the beginning
// @Tags         observability
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        request body entity.ResetConsumerRequest true "Reset consumer request"
// @Success      200 {object} Response{data=object{success=bool,message=string}}
// @Failure      400 {object} Response
// @Failure      401 {object} Response
// @Failure      500 {object} Response
// @Router       /observability/queue/reset-consumer [post]
func (h *ObservabilityHandler) ResetConsumer(c *gin.Context) {
	var req entity.ResetConsumerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	if err := h.observabilityService.ResetConsumer(c.Request.Context(), req.Stream, req.Consumer); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to reset consumer: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Consumer reset successfully",
	})
}

// GetSystemStats godoc
// @Summary      Get system statistics
// @Description  Returns system statistics including message counts, response times, and error rates
// @Tags         observability
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        period query string false "Time period (hour, day, week)" default(day)
// @Success      200 {object} Response{data=entity.SystemStats}
// @Failure      401 {object} Response
// @Failure      500 {object} Response
// @Router       /observability/stats [get]
func (h *ObservabilityHandler) GetSystemStats(c *gin.Context) {
	tenantID := middleware.GetTenantID(c)

	periodStr := c.DefaultQuery("period", "day")
	period := entity.StatsPeriod(periodStr)

	stats, err := h.observabilityService.GetSystemStats(c.Request.Context(), tenantID, period)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get system stats"})
		return
	}

	c.JSON(http.StatusOK, stats)
}

// CleanupLogs godoc
// @Summary      Cleanup old logs
// @Description  Remove logs older than the specified retention period
// @Tags         observability
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        request body object{retention_days=int} true "Cleanup parameters"
// @Success      200 {object} Response{data=object{success=bool,deleted_count=int}}
// @Failure      401 {object} Response
// @Failure      500 {object} Response
// @Router       /observability/logs/cleanup [post]
func (h *ObservabilityHandler) CleanupLogs(c *gin.Context) {
	tenantID := middleware.GetTenantID(c)

	var req struct {
		RetentionDays int `json:"retention_days"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		req.RetentionDays = 30 // Default
	}

	count, err := h.observabilityService.CleanupOldLogs(c.Request.Context(), tenantID, req.RetentionDays)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to cleanup logs"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success":       true,
		"deleted_count": count,
	})
}
