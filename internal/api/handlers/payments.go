package handlers

import (
	"encoding/json"
	"io"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/msgfy/linktor/internal/whatsapp/payments"
)

// PaymentsHandler handles payment-related HTTP requests
type PaymentsHandler struct {
	clients map[string]*payments.Client // key: channel_id
}

// NewPaymentsHandler creates a new payments handler
func NewPaymentsHandler() *PaymentsHandler {
	return &PaymentsHandler{
		clients: make(map[string]*payments.Client),
	}
}

// RegisterClient registers a payments client for a channel
func (h *PaymentsHandler) RegisterClient(channelID string, client *payments.Client) {
	h.clients[channelID] = client
}

// getClient retrieves the payments client for a channel
func (h *PaymentsHandler) getClient(channelID string) (*payments.Client, bool) {
	client, ok := h.clients[channelID]
	return client, ok
}

// CreatePayment handles POST /channels/:channelId/payments
func (h *PaymentsHandler) CreatePayment(c *gin.Context) {
	channelID := c.Param("channelId")

	client, ok := h.getClient(channelID)
	if !ok {
		c.JSON(http.StatusNotFound, gin.H{"error": "Channel not found or payments not configured"})
		return
	}

	var req payments.PaymentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	// Validate required fields
	if req.To == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Recipient phone number is required"})
		return
	}
	if req.Amount <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Amount must be greater than 0"})
		return
	}
	if req.Currency == "" {
		req.Currency = "BRL" // Default to BRL for Brazil
	}
	if req.ReferenceID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Reference ID is required"})
		return
	}

	result, err := client.CreatePayment(c.Request.Context(), &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, result)
}

// GetPayment handles GET /channels/:channelId/payments/:paymentId
func (h *PaymentsHandler) GetPayment(c *gin.Context) {
	channelID := c.Param("channelId")
	paymentID := c.Param("paymentId")

	client, ok := h.getClient(channelID)
	if !ok {
		c.JSON(http.StatusNotFound, gin.H{"error": "Channel not found or payments not configured"})
		return
	}

	payment, found := client.GetPayment(paymentID)
	if !found {
		c.JSON(http.StatusNotFound, gin.H{"error": "Payment not found"})
		return
	}

	c.JSON(http.StatusOK, payment)
}

// GetPaymentByReference handles GET /channels/:channelId/payments/reference/:referenceId
func (h *PaymentsHandler) GetPaymentByReference(c *gin.Context) {
	channelID := c.Param("channelId")
	referenceID := c.Param("referenceId")

	client, ok := h.getClient(channelID)
	if !ok {
		c.JSON(http.StatusNotFound, gin.H{"error": "Channel not found or payments not configured"})
		return
	}

	payment, found := client.GetPaymentByReference(referenceID)
	if !found {
		c.JSON(http.StatusNotFound, gin.H{"error": "Payment not found"})
		return
	}

	c.JSON(http.StatusOK, payment)
}

// ProcessRefund handles POST /channels/:channelId/payments/:paymentId/refund
func (h *PaymentsHandler) ProcessRefund(c *gin.Context) {
	channelID := c.Param("channelId")
	paymentID := c.Param("paymentId")

	client, ok := h.getClient(channelID)
	if !ok {
		c.JSON(http.StatusNotFound, gin.H{"error": "Channel not found or payments not configured"})
		return
	}

	var req struct {
		Amount int64  `json:"amount"`
		Reason string `json:"reason"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		// Non-fatal: amount defaults to 0 (full refund), reason is optional
		req.Amount = 0
		req.Reason = ""
	}

	refundReq := &payments.RefundRequest{
		PaymentID: paymentID,
		Amount:    req.Amount,
		Reason:    req.Reason,
	}

	refund, err := client.ProcessRefund(c.Request.Context(), refundReq)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, refund)
}

// GetPaymentStats handles GET /channels/:channelId/payments/stats
func (h *PaymentsHandler) GetPaymentStats(c *gin.Context) {
	channelID := c.Param("channelId")

	client, ok := h.getClient(channelID)
	if !ok {
		c.JSON(http.StatusNotFound, gin.H{"error": "Channel not found or payments not configured"})
		return
	}

	stats := client.GetPaymentStats()
	c.JSON(http.StatusOK, stats)
}

// GetCustomerPayments handles GET /channels/:channelId/payments/customer/:phone
func (h *PaymentsHandler) GetCustomerPayments(c *gin.Context) {
	channelID := c.Param("channelId")
	phone := c.Param("phone")

	client, ok := h.getClient(channelID)
	if !ok {
		c.JSON(http.StatusNotFound, gin.H{"error": "Channel not found or payments not configured"})
		return
	}

	paymentsList := client.GetPaymentsByCustomer(phone)
	c.JSON(http.StatusOK, gin.H{"payments": paymentsList})
}

// HandleWebhook handles POST /webhooks/payments/:channelId
func (h *PaymentsHandler) HandleWebhook(c *gin.Context) {
	channelID := c.Param("channelId")

	client, ok := h.getClient(channelID)
	if !ok {
		c.JSON(http.StatusNotFound, gin.H{"error": "Channel not found"})
		return
	}

	// Read raw body for signature validation
	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Failed to read request body"})
		return
	}

	// Validate signature
	signature := c.GetHeader("X-Webhook-Signature")
	if signature == "" {
		signature = c.GetHeader("X-Razorpay-Signature")
	}
	if signature == "" {
		signature = c.GetHeader("X-PagSeguro-Signature")
	}

	if !client.ValidateWebhookSignature(body, signature) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid webhook signature"})
		return
	}

	// Parse webhook payload
	var payload payments.PaymentWebhookPayload
	if err := json.Unmarshal(body, &payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid webhook payload"})
		return
	}

	// Process webhook
	if err := client.ProcessWebhook(&payload); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "processed"})
}
