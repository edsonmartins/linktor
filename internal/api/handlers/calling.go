package handlers

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/msgfy/linktor/internal/whatsapp/calling"
)

// CallingHandler handles call-related HTTP requests
type CallingHandler struct {
	clients map[string]*calling.Client // key: channel_id
}

// NewCallingHandler creates a new calling handler
func NewCallingHandler() *CallingHandler {
	return &CallingHandler{
		clients: make(map[string]*calling.Client),
	}
}

// RegisterClient registers a calling client for a channel
func (h *CallingHandler) RegisterClient(channelID string, client *calling.Client) {
	h.clients[channelID] = client
}

// getClient retrieves the calling client for a channel
func (h *CallingHandler) getClient(channelID string) (*calling.Client, bool) {
	client, ok := h.clients[channelID]
	return client, ok
}

// InitiateCall handles POST /channels/:channelId/calls
func (h *CallingHandler) InitiateCall(c *gin.Context) {
	channelID := c.Param("channelId")

	client, ok := h.getClient(channelID)
	if !ok {
		c.JSON(http.StatusNotFound, gin.H{"error": "Channel not found or calling not configured"})
		return
	}

	var req calling.InitiateCallRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	// Validate required fields
	if req.To == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Recipient phone number is required"})
		return
	}
	if req.Type == "" {
		req.Type = calling.CallTypeVoice // Default to voice
	}

	result, err := client.InitiateCall(c.Request.Context(), &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, result)
}

// GetCall handles GET /channels/:channelId/calls/:callId
func (h *CallingHandler) GetCall(c *gin.Context) {
	channelID := c.Param("channelId")
	callID := c.Param("callId")

	client, ok := h.getClient(channelID)
	if !ok {
		c.JSON(http.StatusNotFound, gin.H{"error": "Channel not found or calling not configured"})
		return
	}

	call, found := client.GetCall(callID)
	if !found {
		c.JSON(http.StatusNotFound, gin.H{"error": "Call not found"})
		return
	}

	c.JSON(http.StatusOK, call)
}

// EndCall handles POST /channels/:channelId/calls/:callId/end
func (h *CallingHandler) EndCall(c *gin.Context) {
	channelID := c.Param("channelId")
	callID := c.Param("callId")

	client, ok := h.getClient(channelID)
	if !ok {
		c.JSON(http.StatusNotFound, gin.H{"error": "Channel not found or calling not configured"})
		return
	}

	if err := client.EndCall(c.Request.Context(), callID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "ended"})
}

// GetCallStats handles GET /channels/:channelId/calls/stats
func (h *CallingHandler) GetCallStats(c *gin.Context) {
	channelID := c.Param("channelId")

	client, ok := h.getClient(channelID)
	if !ok {
		c.JSON(http.StatusNotFound, gin.H{"error": "Channel not found or calling not configured"})
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
		stats := client.GetCallStatsByPeriod(startDate, endDate)
		c.JSON(http.StatusOK, stats)
		return
	}

	stats := client.GetCallStats()
	c.JSON(http.StatusOK, stats)
}

// GetRecentCalls handles GET /channels/:channelId/calls
func (h *CallingHandler) GetRecentCalls(c *gin.Context) {
	channelID := c.Param("channelId")

	client, ok := h.getClient(channelID)
	if !ok {
		c.JSON(http.StatusNotFound, gin.H{"error": "Channel not found or calling not configured"})
		return
	}

	// Parse pagination
	limit := 20
	offset := 0

	if limitStr := c.Query("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 && l <= 100 {
			limit = l
		}
	}
	if offsetStr := c.Query("offset"); offsetStr != "" {
		if o, err := strconv.Atoi(offsetStr); err == nil && o >= 0 {
			offset = o
		}
	}

	calls := client.GetRecentCalls(limit, offset)
	c.JSON(http.StatusOK, gin.H{
		"calls":  calls,
		"limit":  limit,
		"offset": offset,
	})
}

// GetCallsByPhone handles GET /channels/:channelId/calls/phone/:phone
func (h *CallingHandler) GetCallsByPhone(c *gin.Context) {
	channelID := c.Param("channelId")
	phone := c.Param("phone")

	client, ok := h.getClient(channelID)
	if !ok {
		c.JSON(http.StatusNotFound, gin.H{"error": "Channel not found or calling not configured"})
		return
	}

	calls := client.GetCallsByPhone(phone)
	c.JSON(http.StatusOK, gin.H{"calls": calls})
}

// GetCallQuality handles GET /channels/:channelId/calls/:callId/quality
func (h *CallingHandler) GetCallQuality(c *gin.Context) {
	channelID := c.Param("channelId")
	callID := c.Param("callId")

	client, ok := h.getClient(channelID)
	if !ok {
		c.JSON(http.StatusNotFound, gin.H{"error": "Channel not found or calling not configured"})
		return
	}

	quality, err := client.GetCallQuality(c.Request.Context(), callID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, quality)
}

// GetCallRecording handles GET /channels/:channelId/calls/:callId/recording
func (h *CallingHandler) GetCallRecording(c *gin.Context) {
	channelID := c.Param("channelId")
	callID := c.Param("callId")

	client, ok := h.getClient(channelID)
	if !ok {
		c.JSON(http.StatusNotFound, gin.H{"error": "Channel not found or calling not configured"})
		return
	}

	recordingURL, err := client.GetCallRecording(c.Request.Context(), callID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"recording_url": recordingURL})
}

// HandleWebhook handles POST /webhooks/calls/:channelId
func (h *CallingHandler) HandleWebhook(c *gin.Context) {
	channelID := c.Param("channelId")

	client, ok := h.getClient(channelID)
	if !ok {
		c.JSON(http.StatusNotFound, gin.H{"error": "Channel not found"})
		return
	}

	var payload calling.CallWebhookPayload
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid webhook payload"})
		return
	}

	if err := client.ProcessWebhook(&payload); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "processed"})
}
