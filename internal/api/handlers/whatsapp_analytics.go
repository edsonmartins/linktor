package handlers

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/msgfy/linktor/internal/whatsapp/analytics"
)

// WhatsAppAnalyticsHandler handles WhatsApp analytics HTTP requests
type WhatsAppAnalyticsHandler struct {
	clients map[string]*analytics.Client // key: channel_id
}

// NewWhatsAppAnalyticsHandler creates a new analytics handler
func NewWhatsAppAnalyticsHandler() *WhatsAppAnalyticsHandler {
	return &WhatsAppAnalyticsHandler{
		clients: make(map[string]*analytics.Client),
	}
}

// RegisterClient registers an analytics client for a channel
func (h *WhatsAppAnalyticsHandler) RegisterClient(channelID string, client *analytics.Client) {
	h.clients[channelID] = client
}

// getClient retrieves the analytics client for a channel
func (h *WhatsAppAnalyticsHandler) getClient(channelID string) (*analytics.Client, bool) {
	client, ok := h.clients[channelID]
	return client, ok
}

// GetConversationAnalytics godoc
// @Summary      Get conversation analytics
// @Description  Returns WhatsApp conversation analytics for a channel
// @Tags         whatsapp-analytics
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        channelId   path     string  true  "Channel ID"
// @Param        start_date  query    string  false "Start date (YYYY-MM-DD)"
// @Param        end_date    query    string  false "End date (YYYY-MM-DD)"
// @Param        granularity query    string  false "Granularity (DAILY, MONTHLY)" default(DAILY)
// @Success      200 {object} analytics.ConversationAnalytics
// @Failure      404 {object} Response
// @Failure      500 {object} Response
// @Router       /channels/{channelId}/analytics/conversations [get]
func (h *WhatsAppAnalyticsHandler) GetConversationAnalytics(c *gin.Context) {
	channelID := c.Param("id")

	client, ok := h.getClient(channelID)
	if !ok {
		c.JSON(http.StatusNotFound, gin.H{"error": "Channel not found or analytics not configured"})
		return
	}

	// Parse query parameters
	startDate, err := parseDate(c.Query("start_date"))
	if err != nil {
		startDate = time.Now().AddDate(0, 0, -30) // Default to last 30 days
	}

	endDate, err := parseDate(c.Query("end_date"))
	if err != nil {
		endDate = time.Now()
	}

	granularity := c.DefaultQuery("granularity", "DAILY")

	req := &analytics.AnalyticsRequest{
		PhoneNumberID: channelID, // Assuming channelID maps to phoneNumberID
		StartDate:     startDate,
		EndDate:       endDate,
		Granularity:   granularity,
	}

	result, err := client.GetConversationAnalytics(c.Request.Context(), req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, result)
}

// GetPhoneNumberAnalytics godoc
// @Summary      Get phone number analytics
// @Description  Returns WhatsApp phone number quality and limit information
// @Tags         whatsapp-analytics
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        channelId path string true "Channel ID"
// @Success      200 {object} analytics.PhoneNumberAnalytics
// @Failure      404 {object} Response
// @Failure      500 {object} Response
// @Router       /channels/{channelId}/analytics/phone [get]
func (h *WhatsAppAnalyticsHandler) GetPhoneNumberAnalytics(c *gin.Context) {
	channelID := c.Param("id")

	client, ok := h.getClient(channelID)
	if !ok {
		c.JSON(http.StatusNotFound, gin.H{"error": "Channel not found or analytics not configured"})
		return
	}

	result, err := client.GetPhoneNumberAnalytics(c.Request.Context(), channelID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, result)
}

// GetTemplateAnalytics godoc
// @Summary      Get template analytics
// @Description  Returns performance metrics for a specific WhatsApp template
// @Tags         whatsapp-analytics
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        channelId   path  string true  "Channel ID"
// @Param        templateId  path  string true  "Template ID"
// @Param        start_date  query string false "Start date (YYYY-MM-DD)"
// @Param        end_date    query string false "End date (YYYY-MM-DD)"
// @Success      200 {object} analytics.TemplateAnalytics
// @Failure      404 {object} Response
// @Failure      500 {object} Response
// @Router       /channels/{channelId}/analytics/templates/{templateId} [get]
func (h *WhatsAppAnalyticsHandler) GetTemplateAnalytics(c *gin.Context) {
	channelID := c.Param("id")
	templateID := c.Param("templateId")

	client, ok := h.getClient(channelID)
	if !ok {
		c.JSON(http.StatusNotFound, gin.H{"error": "Channel not found or analytics not configured"})
		return
	}

	// Parse query parameters
	startDate, err := parseDate(c.Query("start_date"))
	if err != nil {
		startDate = time.Now().AddDate(0, 0, -30) // Default to last 30 days
	}

	endDate, err := parseDate(c.Query("end_date"))
	if err != nil {
		endDate = time.Now()
	}

	result, err := client.GetTemplateAnalytics(c.Request.Context(), templateID, startDate, endDate)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, result)
}

// GetAggregatedStats godoc
// @Summary      Get aggregated statistics
// @Description  Returns aggregated WhatsApp conversation statistics with trends
// @Tags         whatsapp-analytics
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        channelId  path  string true  "Channel ID"
// @Param        start_date query string false "Start date (YYYY-MM-DD)"
// @Param        end_date   query string false "End date (YYYY-MM-DD)"
// @Success      200 {object} analytics.AggregatedStats
// @Failure      404 {object} Response
// @Failure      500 {object} Response
// @Router       /channels/{channelId}/analytics/stats [get]
func (h *WhatsAppAnalyticsHandler) GetAggregatedStats(c *gin.Context) {
	channelID := c.Param("id")

	client, ok := h.getClient(channelID)
	if !ok {
		c.JSON(http.StatusNotFound, gin.H{"error": "Channel not found or analytics not configured"})
		return
	}

	// Parse query parameters
	startDate, err := parseDate(c.Query("start_date"))
	if err != nil {
		startDate = time.Now().AddDate(0, 0, -30) // Default to last 30 days
	}

	endDate, err := parseDate(c.Query("end_date"))
	if err != nil {
		endDate = time.Now()
	}

	req := &analytics.AnalyticsRequest{
		PhoneNumberID: channelID,
		StartDate:     startDate,
		EndDate:       endDate,
		Granularity:   "DAILY",
	}

	result, err := client.GetAggregatedStats(c.Request.Context(), req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, result)
}

// ExportAnalytics godoc
// @Summary      Export analytics data
// @Description  Export WhatsApp analytics to CSV or JSON format
// @Tags         whatsapp-analytics
// @Accept       json
// @Produce      text/csv,application/json
// @Security     BearerAuth
// @Param        channelId  path  string true  "Channel ID"
// @Param        start_date query string false "Start date (YYYY-MM-DD)"
// @Param        end_date   query string false "End date (YYYY-MM-DD)"
// @Param        format     query string false "Export format (csv, json)" default(csv)
// @Success      200 {file} file
// @Failure      400 {object} Response
// @Failure      404 {object} Response
// @Failure      500 {object} Response
// @Router       /channels/{channelId}/analytics/export [get]
func (h *WhatsAppAnalyticsHandler) ExportAnalytics(c *gin.Context) {
	channelID := c.Param("id")

	client, ok := h.getClient(channelID)
	if !ok {
		c.JSON(http.StatusNotFound, gin.H{"error": "Channel not found or analytics not configured"})
		return
	}

	// Parse query parameters
	startDate, err := parseDate(c.Query("start_date"))
	if err != nil {
		startDate = time.Now().AddDate(0, 0, -30)
	}

	endDate, err := parseDate(c.Query("end_date"))
	if err != nil {
		endDate = time.Now()
	}

	format := c.DefaultQuery("format", "csv")

	req := &analytics.AnalyticsRequest{
		PhoneNumberID: channelID,
		StartDate:     startDate,
		EndDate:       endDate,
		Granularity:   "DAILY",
	}

	analyticsData, err := client.GetConversationAnalytics(c.Request.Context(), req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	switch format {
	case "csv":
		data, err := client.ExportToCSV(analyticsData)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.Header("Content-Disposition", "attachment; filename=analytics.csv")
		c.Data(http.StatusOK, "text/csv", data)

	case "json":
		data, err := client.ExportToJSON(analyticsData)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.Header("Content-Disposition", "attachment; filename=analytics.json")
		c.Data(http.StatusOK, "application/json", data)

	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "Unsupported format. Use 'csv' or 'json'"})
	}
}

// GetDashboardData godoc
// @Summary      Get dashboard data
// @Description  Returns comprehensive dashboard data for WhatsApp analytics (last 30 days)
// @Tags         whatsapp-analytics
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        channelId path string true "Channel ID"
// @Success      200 {object} map[string]interface{}
// @Failure      404 {object} Response
// @Failure      500 {object} Response
// @Router       /channels/{channelId}/analytics/dashboard [get]
func (h *WhatsAppAnalyticsHandler) GetDashboardData(c *gin.Context) {
	channelID := c.Param("id")

	client, ok := h.getClient(channelID)
	if !ok {
		c.JSON(http.StatusNotFound, gin.H{"error": "Channel not found or analytics not configured"})
		return
	}

	ctx := c.Request.Context()

	// Get last 30 days data
	endDate := time.Now()
	startDate := endDate.AddDate(0, 0, -30)

	req := &analytics.AnalyticsRequest{
		PhoneNumberID: channelID,
		StartDate:     startDate,
		EndDate:       endDate,
		Granularity:   "DAILY",
	}

	// Fetch data in parallel would be better, but for simplicity:
	convAnalytics, err := client.GetConversationAnalytics(ctx, req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	phoneAnalytics, err := client.GetPhoneNumberAnalytics(ctx, channelID)
	if err != nil {
		// Non-fatal, continue without phone analytics
		phoneAnalytics = nil
	}

	aggStats, err := client.GetAggregatedStats(ctx, req)
	if err != nil {
		// Non-fatal, continue without aggregated stats
		aggStats = nil
	}

	// Build dashboard response
	dashboard := gin.H{
		"period": gin.H{
			"start": startDate.Format("2006-01-02"),
			"end":   endDate.Format("2006-01-02"),
		},
		"conversations": gin.H{
			"total":       convAnalytics.TotalConversations,
			"total_cost":  convAnalytics.TotalCost,
			"currency":    convAnalytics.Currency,
			"by_type":     convAnalytics.ByType,
			"by_category": convAnalytics.ByCategory,
			"by_country":  convAnalytics.ByCountry,
			"timeline":    convAnalytics.Timeline,
		},
	}

	if phoneAnalytics != nil {
		dashboard["phone_number"] = gin.H{
			"quality_rating":   phoneAnalytics.QualityRating,
			"messaging_limit":  phoneAnalytics.MessagingLimit,
			"status":           phoneAnalytics.Status,
			"throughput":       phoneAnalytics.CurrentThroughput,
		}
	}

	if aggStats != nil {
		dashboard["summary"] = gin.H{
			"average_daily_cost": aggStats.AverageDailyCost,
			"top_countries":      aggStats.TopCountries,
			"trends":             aggStats.Trends,
		}
	}

	c.JSON(http.StatusOK, dashboard)
}

// parseDate parses a date string in YYYY-MM-DD format
func parseDate(dateStr string) (time.Time, error) {
	return time.Parse("2006-01-02", dateStr)
}
