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

// GetReferral godoc
// @Summary      Get referral by ID
// @Description  Returns a specific Click-to-WhatsApp Ads referral
// @Tags         whatsapp-ctwa
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        channelId  path string true "Channel ID"
// @Param        referralId path string true "Referral ID"
// @Success      200 {object} ctwa.Referral
// @Failure      404 {object} Response
// @Router       /channels/{channelId}/ctwa/referrals/{referralId} [get]
func (h *CTWAHandler) GetReferral(c *gin.Context) {
	channelID := c.Param("id")
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

// GetReferralByPhone godoc
// @Summary      Get referral by phone number
// @Description  Returns the most recent CTWA referral for a phone number
// @Tags         whatsapp-ctwa
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        channelId path string true "Channel ID"
// @Param        phone     path string true "Customer phone number"
// @Success      200 {object} ctwa.Referral
// @Failure      404 {object} Response
// @Router       /channels/{channelId}/ctwa/referrals/phone/{phone} [get]
func (h *CTWAHandler) GetReferralByPhone(c *gin.Context) {
	channelID := c.Param("id")
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

// GetReferralsByCampaign godoc
// @Summary      Get referrals by campaign
// @Description  Returns all CTWA referrals for a specific campaign
// @Tags         whatsapp-ctwa
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        channelId  path string true "Channel ID"
// @Param        campaignId path string true "Campaign ID"
// @Success      200 {object} object{referrals=[]ctwa.Referral}
// @Failure      404 {object} Response
// @Router       /channels/{channelId}/ctwa/campaigns/{campaignId}/referrals [get]
func (h *CTWAHandler) GetReferralsByCampaign(c *gin.Context) {
	channelID := c.Param("id")
	campaignID := c.Param("campaignId")

	client, ok := h.getClient(channelID)
	if !ok {
		c.JSON(http.StatusNotFound, gin.H{"error": "Channel not found or CTWA not configured"})
		return
	}

	referrals := client.GetReferralsByCampaign(campaignID)
	c.JSON(http.StatusOK, gin.H{"referrals": referrals})
}

// TrackConversion godoc
// @Summary      Track a conversion
// @Description  Records a conversion event for a CTWA referral (purchase, signup, etc.)
// @Tags         whatsapp-ctwa
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        channelId path string true "Channel ID"
// @Param        request   body object{referral_id=string,conversion_type=string,value=number,currency=string} true "Conversion details"
// @Success      201 {object} ctwa.AdConversion
// @Failure      400 {object} Response
// @Failure      404 {object} Response
// @Failure      500 {object} Response
// @Router       /channels/{channelId}/ctwa/conversions [post]
func (h *CTWAHandler) TrackConversion(c *gin.Context) {
	channelID := c.Param("id")

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

// GetConversion godoc
// @Summary      Get conversion by ID
// @Description  Returns details of a specific conversion event
// @Tags         whatsapp-ctwa
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        channelId    path string true "Channel ID"
// @Param        conversionId path string true "Conversion ID"
// @Success      200 {object} ctwa.AdConversion
// @Failure      404 {object} Response
// @Router       /channels/{channelId}/ctwa/conversions/{conversionId} [get]
func (h *CTWAHandler) GetConversion(c *gin.Context) {
	channelID := c.Param("id")
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

// GetConversionsByReferral godoc
// @Summary      Get conversions by referral
// @Description  Returns all conversions for a specific referral
// @Tags         whatsapp-ctwa
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        channelId  path string true "Channel ID"
// @Param        referralId path string true "Referral ID"
// @Success      200 {object} object{conversions=[]ctwa.AdConversion}
// @Failure      404 {object} Response
// @Router       /channels/{channelId}/ctwa/referrals/{referralId}/conversions [get]
func (h *CTWAHandler) GetConversionsByReferral(c *gin.Context) {
	channelID := c.Param("id")
	referralID := c.Param("referralId")

	client, ok := h.getClient(channelID)
	if !ok {
		c.JSON(http.StatusNotFound, gin.H{"error": "Channel not found or CTWA not configured"})
		return
	}

	conversions := client.GetConversionsByReferral(referralID)
	c.JSON(http.StatusOK, gin.H{"conversions": conversions})
}

// GetFreeWindow godoc
// @Summary      Get free messaging window
// @Description  Returns the 72-hour free messaging window status for a customer (CTWA benefit)
// @Tags         whatsapp-ctwa
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        channelId path string true "Channel ID"
// @Param        phone     path string true "Customer phone number"
// @Success      200 {object} object{free_window=ctwa.FreeMessagingWindow,time_remaining=string,is_valid=bool}
// @Failure      404 {object} Response
// @Router       /channels/{channelId}/ctwa/free-window/{phone} [get]
func (h *CTWAHandler) GetFreeWindow(c *gin.Context) {
	channelID := c.Param("id")
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

// GetStats godoc
// @Summary      Get CTWA statistics
// @Description  Returns Click-to-WhatsApp Ads statistics and conversion metrics
// @Tags         whatsapp-ctwa
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        channelId  path  string true  "Channel ID"
// @Param        start_date query string false "Start date (YYYY-MM-DD)"
// @Param        end_date   query string false "End date (YYYY-MM-DD)"
// @Success      200 {object} ctwa.CTWAStats
// @Failure      400 {object} Response
// @Failure      404 {object} Response
// @Router       /channels/{channelId}/ctwa/stats [get]
func (h *CTWAHandler) GetStats(c *gin.Context) {
	channelID := c.Param("id")

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

// GetTopAds godoc
// @Summary      Get top performing ads
// @Description  Returns the top performing Click-to-WhatsApp ads by conversions
// @Tags         whatsapp-ctwa
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        channelId path  string  true  "Channel ID"
// @Param        limit     query integer false "Limit (1-50)" default(10)
// @Success      200 {object} object{top_ads=[]ctwa.AdPerformance}
// @Failure      404 {object} Response
// @Router       /channels/{channelId}/ctwa/top-ads [get]
func (h *CTWAHandler) GetTopAds(c *gin.Context) {
	channelID := c.Param("id")

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

// GenerateReport godoc
// @Summary      Generate CTWA report
// @Description  Generates a comprehensive Click-to-WhatsApp Ads performance report
// @Tags         whatsapp-ctwa
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        channelId  path  string true  "Channel ID"
// @Param        start_date query string false "Start date (YYYY-MM-DD)" default(last 30 days)
// @Param        end_date   query string false "End date (YYYY-MM-DD)" default(today)
// @Success      200 {object} ctwa.CTWAReport
// @Failure      404 {object} Response
// @Router       /channels/{channelId}/ctwa/report [get]
func (h *CTWAHandler) GenerateReport(c *gin.Context) {
	channelID := c.Param("id")

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

// GetDashboard godoc
// @Summary      Get CTWA dashboard
// @Description  Returns comprehensive dashboard data for Click-to-WhatsApp Ads (last 30 days)
// @Tags         whatsapp-ctwa
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        channelId path string true "Channel ID"
// @Success      200 {object} map[string]interface{}
// @Failure      404 {object} Response
// @Router       /channels/{channelId}/ctwa/dashboard [get]
func (h *CTWAHandler) GetDashboard(c *gin.Context) {
	channelID := c.Param("id")

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

// ProcessReferralWebhook godoc
// @Summary      CTWA referral webhook endpoint
// @Description  Receives referral messages from Click-to-WhatsApp Ads
// @Tags         webhooks
// @Accept       json
// @Produce      json
// @Param        channelId path string true "Channel ID"
// @Param        payload   body ctwa.ReferralMessage true "Referral message payload"
// @Success      200 {object} object{status=string,referral=ctwa.Referral}
// @Failure      400 {object} Response
// @Failure      404 {object} Response
// @Failure      500 {object} Response
// @Router       /webhooks/ctwa/{channelId} [post]
func (h *CTWAHandler) ProcessReferralWebhook(c *gin.Context) {
	channelID := c.Param("id")

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
