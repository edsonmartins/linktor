package handlers

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/msgfy/linktor/internal/whatsapp/ctwa"
)

// CTWAHandler handles Click-to-WhatsApp Ads HTTP requests
type CTWAHandler struct {
	clients map[string]*ctwa.Client // key: channel_id
}

// NewCTWAHandler creates a new CTWA handler
func NewCTWAHandler() *CTWAHandler {
	return &CTWAHandler{
		clients: make(map[string]*ctwa.Client),
	}
}

// RegisterClient registers a CTWA client for a channel
func (h *CTWAHandler) RegisterClient(channelID string, client *ctwa.Client) {
	h.clients[channelID] = client
}

// getClient retrieves the CTWA client for a channel
func (h *CTWAHandler) getClient(channelID string) (*ctwa.Client, bool) {
	client, ok := h.clients[channelID]
	return client, ok
}

// GetReferral handles GET /channels/:channelId/ctwa/referrals/:referralId
func (h *CTWAHandler) GetReferral(c *gin.Context) {
	channelID := c.Param("channelId")
	referralID := c.Param("referralId")

	client, ok := h.getClient(channelID)
	if !ok {
		c.JSON(http.StatusNotFound, gin.H{"error": "Channel not found or CTWA not configured"})
		return
	}

	referral, found := client.GetReferral(referralID)
	if !found {
		c.JSON(http.StatusNotFound, gin.H{"error": "Referral not found"})
		return
	}

	c.JSON(http.StatusOK, referral)
}

// GetReferralByPhone handles GET /channels/:channelId/ctwa/referrals/phone/:phone
func (h *CTWAHandler) GetReferralByPhone(c *gin.Context) {
	channelID := c.Param("channelId")
	phone := c.Param("phone")

	client, ok := h.getClient(channelID)
	if !ok {
		c.JSON(http.StatusNotFound, gin.H{"error": "Channel not found or CTWA not configured"})
		return
	}

	referral, found := client.GetReferralByPhone(phone)
	if !found {
		c.JSON(http.StatusNotFound, gin.H{"error": "No referral found for this phone"})
		return
	}

	c.JSON(http.StatusOK, referral)
}

// GetReferralsByCampaign handles GET /channels/:channelId/ctwa/campaigns/:campaignId/referrals
func (h *CTWAHandler) GetReferralsByCampaign(c *gin.Context) {
	channelID := c.Param("channelId")
	campaignID := c.Param("campaignId")

	client, ok := h.getClient(channelID)
	if !ok {
		c.JSON(http.StatusNotFound, gin.H{"error": "Channel not found or CTWA not configured"})
		return
	}

	referrals := client.GetReferralsByCampaign(campaignID)
	c.JSON(http.StatusOK, gin.H{"referrals": referrals})
}

// TrackConversion handles POST /channels/:channelId/ctwa/conversions
func (h *CTWAHandler) TrackConversion(c *gin.Context) {
	channelID := c.Param("channelId")

	client, ok := h.getClient(channelID)
	if !ok {
		c.JSON(http.StatusNotFound, gin.H{"error": "Channel not found or CTWA not configured"})
		return
	}

	var req struct {
		ReferralID     string  `json:"referral_id" binding:"required"`
		ConversionType string  `json:"conversion_type" binding:"required"`
		Value          float64 `json:"value"`
		Currency       string  `json:"currency"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	if req.Currency == "" {
		req.Currency = "BRL"
	}

	conversion, err := client.TrackConversion(req.ReferralID, req.ConversionType, req.Value, req.Currency)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, conversion)
}

// GetConversion handles GET /channels/:channelId/ctwa/conversions/:conversionId
func (h *CTWAHandler) GetConversion(c *gin.Context) {
	channelID := c.Param("channelId")
	conversionID := c.Param("conversionId")

	client, ok := h.getClient(channelID)
	if !ok {
		c.JSON(http.StatusNotFound, gin.H{"error": "Channel not found or CTWA not configured"})
		return
	}

	conversion, found := client.GetConversion(conversionID)
	if !found {
		c.JSON(http.StatusNotFound, gin.H{"error": "Conversion not found"})
		return
	}

	c.JSON(http.StatusOK, conversion)
}

// GetConversionsByReferral handles GET /channels/:channelId/ctwa/referrals/:referralId/conversions
func (h *CTWAHandler) GetConversionsByReferral(c *gin.Context) {
	channelID := c.Param("channelId")
	referralID := c.Param("referralId")

	client, ok := h.getClient(channelID)
	if !ok {
		c.JSON(http.StatusNotFound, gin.H{"error": "Channel not found or CTWA not configured"})
		return
	}

	conversions := client.GetConversionsByReferral(referralID)
	c.JSON(http.StatusOK, gin.H{"conversions": conversions})
}

// GetFreeWindow handles GET /channels/:channelId/ctwa/free-window/:phone
func (h *CTWAHandler) GetFreeWindow(c *gin.Context) {
	channelID := c.Param("channelId")
	phone := c.Param("phone")

	client, ok := h.getClient(channelID)
	if !ok {
		c.JSON(http.StatusNotFound, gin.H{"error": "Channel not found or CTWA not configured"})
		return
	}

	window, found := client.GetFreeWindow(phone)
	if !found {
		c.JSON(http.StatusNotFound, gin.H{"error": "No active free messaging window"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"free_window":    window,
		"time_remaining": window.TimeRemaining().String(),
		"is_valid":       window.IsValid(),
	})
}

// GetStats handles GET /channels/:channelId/ctwa/stats
func (h *CTWAHandler) GetStats(c *gin.Context) {
	channelID := c.Param("channelId")

	client, ok := h.getClient(channelID)
	if !ok {
		c.JSON(http.StatusNotFound, gin.H{"error": "Channel not found or CTWA not configured"})
		return
	}

	// Check for period filter
	startDateStr := c.Query("start_date")
	endDateStr := c.Query("end_date")

	if startDateStr != "" && endDateStr != "" {
		startDate, err := time.Parse("2006-01-02", startDateStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid start_date format"})
			return
		}
		endDate, err := time.Parse("2006-01-02", endDateStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid end_date format"})
			return
		}
		stats := client.GetStatsByPeriod(startDate, endDate)
		c.JSON(http.StatusOK, stats)
		return
	}

	stats := client.GetStats()
	c.JSON(http.StatusOK, stats)
}

// GetTopAds handles GET /channels/:channelId/ctwa/top-ads
func (h *CTWAHandler) GetTopAds(c *gin.Context) {
	channelID := c.Param("channelId")

	client, ok := h.getClient(channelID)
	if !ok {
		c.JSON(http.StatusNotFound, gin.H{"error": "Channel not found or CTWA not configured"})
		return
	}

	limit := 10
	if limitStr := c.Query("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 && l <= 50 {
			limit = l
		}
	}

	topAds := client.GetTopPerformingAds(limit)
	c.JSON(http.StatusOK, gin.H{"top_ads": topAds})
}

// GenerateReport handles GET /channels/:channelId/ctwa/report
func (h *CTWAHandler) GenerateReport(c *gin.Context) {
	channelID := c.Param("channelId")

	client, ok := h.getClient(channelID)
	if !ok {
		c.JSON(http.StatusNotFound, gin.H{"error": "Channel not found or CTWA not configured"})
		return
	}

	// Default to last 30 days
	endDate := time.Now()
	startDate := endDate.AddDate(0, 0, -30)

	startDateStr := c.Query("start_date")
	endDateStr := c.Query("end_date")

	if startDateStr != "" {
		if sd, err := time.Parse("2006-01-02", startDateStr); err == nil {
			startDate = sd
		}
	}
	if endDateStr != "" {
		if ed, err := time.Parse("2006-01-02", endDateStr); err == nil {
			endDate = ed
		}
	}

	report := client.GenerateReport(startDate, endDate)
	c.JSON(http.StatusOK, report)
}

// GetDashboard handles GET /channels/:channelId/ctwa/dashboard
func (h *CTWAHandler) GetDashboard(c *gin.Context) {
	channelID := c.Param("channelId")

	client, ok := h.getClient(channelID)
	if !ok {
		c.JSON(http.StatusNotFound, gin.H{"error": "Channel not found or CTWA not configured"})
		return
	}

	// Last 30 days
	endDate := time.Now()
	startDate := endDate.AddDate(0, 0, -30)

	stats := client.GetStatsByPeriod(startDate, endDate)
	topAds := client.GetTopPerformingAds(5)

	dashboard := gin.H{
		"period": gin.H{
			"start": startDate.Format("2006-01-02"),
			"end":   endDate.Format("2006-01-02"),
		},
		"summary": gin.H{
			"total_referrals":   stats.TotalReferrals,
			"total_conversions": stats.TotalConversions,
			"conversion_rate":   stats.ConversionRate,
			"total_value":       stats.TotalValue,
			"currency":          stats.Currency,
			"average_value":     stats.AverageValue,
		},
		"by_source":   stats.BySource,
		"daily_stats": stats.DailyStats,
		"top_ads":     topAds,
	}

	c.JSON(http.StatusOK, dashboard)
}

// ProcessReferralWebhook handles POST /webhooks/ctwa/:channelId
func (h *CTWAHandler) ProcessReferralWebhook(c *gin.Context) {
	channelID := c.Param("channelId")

	client, ok := h.getClient(channelID)
	if !ok {
		c.JSON(http.StatusNotFound, gin.H{"error": "Channel not found"})
		return
	}

	var message ctwa.ReferralMessage
	if err := c.ShouldBindJSON(&message); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid webhook payload"})
		return
	}

	referral, err := client.ProcessReferral(channelID, &message)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":   "processed",
		"referral": referral,
	})
}
