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

// GetConversationAnalytics handles GET /channels/:channelId/analytics/conversations
func (h *WhatsAppAnalyticsHandler) GetConversationAnalytics(c *gin.Context) {
	channelID := c.Param("channelId")

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

// GetPhoneNumberAnalytics handles GET /channels/:channelId/analytics/phone
func (h *WhatsAppAnalyticsHandler) GetPhoneNumberAnalytics(c *gin.Context) {
	channelID := c.Param("channelId")

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

// GetTemplateAnalytics handles GET /channels/:channelId/analytics/templates/:templateId
func (h *WhatsAppAnalyticsHandler) GetTemplateAnalytics(c *gin.Context) {
	channelID := c.Param("channelId")
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

// GetAggregatedStats handles GET /channels/:channelId/analytics/stats
func (h *WhatsAppAnalyticsHandler) GetAggregatedStats(c *gin.Context) {
	channelID := c.Param("channelId")

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

// ExportAnalytics handles GET /channels/:channelId/analytics/export
func (h *WhatsAppAnalyticsHandler) ExportAnalytics(c *gin.Context) {
	channelID := c.Param("channelId")

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

// GetDashboardData handles GET /channels/:channelId/analytics/dashboard
func (h *WhatsAppAnalyticsHandler) GetDashboardData(c *gin.Context) {
	channelID := c.Param("channelId")

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
