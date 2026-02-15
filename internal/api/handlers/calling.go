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

// InitiateCall godoc
// @Summary      Initiate a call
// @Description  Starts a new WhatsApp voice or video call to a customer
// @Tags         whatsapp-calling
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        channelId path string true "Channel ID"
// @Param        request   body calling.InitiateCallRequest true "Call request details"
// @Success      201 {object} calling.Call
// @Failure      400 {object} Response
// @Failure      404 {object} Response
// @Failure      500 {object} Response
// @Router       /channels/{channelId}/calls [post]
func (h *CallingHandler) InitiateCall(c *gin.Context) {
	channelID := c.Param("id")

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

// GetCall godoc
// @Summary      Get call by ID
// @Description  Returns details of a specific call
// @Tags         whatsapp-calling
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        channelId path string true "Channel ID"
// @Param        callId    path string true "Call ID"
// @Success      200 {object} calling.Call
// @Failure      404 {object} Response
// @Router       /channels/{channelId}/calls/{callId} [get]
func (h *CallingHandler) GetCall(c *gin.Context) {
	channelID := c.Param("id")
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

// EndCall godoc
// @Summary      End a call
// @Description  Terminates an active call
// @Tags         whatsapp-calling
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        channelId path string true "Channel ID"
// @Param        callId    path string true "Call ID"
// @Success      200 {object} object{status=string}
// @Failure      404 {object} Response
// @Failure      500 {object} Response
// @Router       /channels/{channelId}/calls/{callId}/end [post]
func (h *CallingHandler) EndCall(c *gin.Context) {
	channelID := c.Param("id")
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

// GetCallStats godoc
// @Summary      Get call statistics
// @Description  Returns aggregated call statistics for the channel
// @Tags         whatsapp-calling
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        channelId  path  string true  "Channel ID"
// @Param        start_date query string false "Start date (YYYY-MM-DD)"
// @Param        end_date   query string false "End date (YYYY-MM-DD)"
// @Success      200 {object} calling.CallStats
// @Failure      400 {object} Response
// @Failure      404 {object} Response
// @Router       /channels/{channelId}/calls/stats [get]
func (h *CallingHandler) GetCallStats(c *gin.Context) {
	channelID := c.Param("id")

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

// GetRecentCalls godoc
// @Summary      List recent calls
// @Description  Returns a paginated list of recent calls
// @Tags         whatsapp-calling
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        channelId path  string  true  "Channel ID"
// @Param        limit     query integer false "Limit (1-100)" default(20)
// @Param        offset    query integer false "Offset" default(0)
// @Success      200 {object} object{calls=[]calling.Call,limit=int,offset=int}
// @Failure      404 {object} Response
// @Router       /channels/{channelId}/calls [get]
func (h *CallingHandler) GetRecentCalls(c *gin.Context) {
	channelID := c.Param("id")

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

// GetCallsByPhone godoc
// @Summary      Get calls by phone number
// @Description  Returns all calls involving a specific phone number
// @Tags         whatsapp-calling
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        channelId path string true "Channel ID"
// @Param        phone     path string true "Phone number"
// @Success      200 {object} object{calls=[]calling.Call}
// @Failure      404 {object} Response
// @Router       /channels/{channelId}/calls/phone/{phone} [get]
func (h *CallingHandler) GetCallsByPhone(c *gin.Context) {
	channelID := c.Param("id")
	phone := c.Param("phone")

	client, ok := h.getClient(channelID)
	if !ok {
		c.JSON(http.StatusNotFound, gin.H{"error": "Channel not found or calling not configured"})
		return
	}

	calls := client.GetCallsByPhone(phone)
	c.JSON(http.StatusOK, gin.H{"calls": calls})
}

// GetCallQuality godoc
// @Summary      Get call quality metrics
// @Description  Returns quality metrics for a specific call (score, packet loss, jitter, latency)
// @Tags         whatsapp-calling
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        channelId path string true "Channel ID"
// @Param        callId    path string true "Call ID"
// @Success      200 {object} calling.CallQuality
// @Failure      404 {object} Response
// @Router       /channels/{channelId}/calls/{callId}/quality [get]
func (h *CallingHandler) GetCallQuality(c *gin.Context) {
	channelID := c.Param("id")
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

// GetCallRecording godoc
// @Summary      Get call recording
// @Description  Returns the recording URL for a specific call (if available)
// @Tags         whatsapp-calling
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        channelId path string true "Channel ID"
// @Param        callId    path string true "Call ID"
// @Success      200 {object} object{recording_url=string}
// @Failure      404 {object} Response
// @Router       /channels/{channelId}/calls/{callId}/recording [get]
func (h *CallingHandler) GetCallRecording(c *gin.Context) {
	channelID := c.Param("id")
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

// HandleWebhook godoc
// @Summary      Call webhook endpoint
// @Description  Receives call status updates from WhatsApp Business API
// @Tags         webhooks
// @Accept       json
// @Produce      json
// @Param        channelId path string true "Channel ID"
// @Param        payload   body calling.CallWebhookPayload true "Webhook payload"
// @Success      200 {object} object{status=string}
// @Failure      400 {object} Response
// @Failure      404 {object} Response
// @Failure      500 {object} Response
// @Router       /webhooks/calls/{channelId} [post]
func (h *CallingHandler) HandleWebhook(c *gin.Context) {
	channelID := c.Param("id")

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
