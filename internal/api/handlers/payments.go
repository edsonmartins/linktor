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

// CreatePayment godoc
// @Summary      Create a payment request
// @Description  Creates a new WhatsApp payment request to send to a customer
// @Tags         whatsapp-payments
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        channelId path string true "Channel ID"
// @Param        request   body payments.PaymentRequest true "Payment request details"
// @Success      201 {object} payments.PaymentResponse
// @Failure      400 {object} Response
// @Failure      404 {object} Response
// @Failure      500 {object} Response
// @Router       /channels/{channelId}/payments [post]
func (h *PaymentsHandler) CreatePayment(c *gin.Context) {
	channelID := c.Param("id")

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

// GetPayment godoc
// @Summary      Get payment by ID
// @Description  Returns a specific payment by its ID
// @Tags         whatsapp-payments
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        channelId path string true "Channel ID"
// @Param        paymentId path string true "Payment ID"
// @Success      200 {object} payments.Payment
// @Failure      404 {object} Response
// @Router       /channels/{channelId}/payments/{paymentId} [get]
func (h *PaymentsHandler) GetPayment(c *gin.Context) {
	channelID := c.Param("id")
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

// GetPaymentByReference godoc
// @Summary      Get payment by reference ID
// @Description  Returns a payment by its external reference ID
// @Tags         whatsapp-payments
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        channelId   path string true "Channel ID"
// @Param        referenceId path string true "Reference ID"
// @Success      200 {object} payments.Payment
// @Failure      404 {object} Response
// @Router       /channels/{channelId}/payments/reference/{referenceId} [get]
func (h *PaymentsHandler) GetPaymentByReference(c *gin.Context) {
	channelID := c.Param("id")
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

// ProcessRefund godoc
// @Summary      Process a refund
// @Description  Initiates a refund for a completed payment
// @Tags         whatsapp-payments
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        channelId path string true "Channel ID"
// @Param        paymentId path string true "Payment ID"
// @Param        request   body object{amount=int64,reason=string} false "Refund details (empty for full refund)"
// @Success      200 {object} payments.Refund
// @Failure      400 {object} Response
// @Failure      404 {object} Response
// @Failure      500 {object} Response
// @Router       /channels/{channelId}/payments/{paymentId}/refund [post]
func (h *PaymentsHandler) ProcessRefund(c *gin.Context) {
	channelID := c.Param("id")
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

// GetPaymentStats godoc
// @Summary      Get payment statistics
// @Description  Returns aggregated payment statistics for the channel
// @Tags         whatsapp-payments
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        channelId path string true "Channel ID"
// @Success      200 {object} payments.PaymentStats
// @Failure      404 {object} Response
// @Router       /channels/{channelId}/payments/stats [get]
func (h *PaymentsHandler) GetPaymentStats(c *gin.Context) {
	channelID := c.Param("id")

	client, ok := h.getClient(channelID)
	if !ok {
		c.JSON(http.StatusNotFound, gin.H{"error": "Channel not found or payments not configured"})
		return
	}

	stats := client.GetPaymentStats()
	c.JSON(http.StatusOK, stats)
}

// GetCustomerPayments godoc
// @Summary      Get customer payments
// @Description  Returns all payments for a specific customer phone number
// @Tags         whatsapp-payments
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        channelId path string true "Channel ID"
// @Param        phone     path string true "Customer phone number"
// @Success      200 {object} object{payments=[]payments.Payment}
// @Failure      404 {object} Response
// @Router       /channels/{channelId}/payments/customer/{phone} [get]
func (h *PaymentsHandler) GetCustomerPayments(c *gin.Context) {
	channelID := c.Param("id")
	phone := c.Param("phone")

	client, ok := h.getClient(channelID)
	if !ok {
		c.JSON(http.StatusNotFound, gin.H{"error": "Channel not found or payments not configured"})
		return
	}

	paymentsList := client.GetPaymentsByCustomer(phone)
	c.JSON(http.StatusOK, gin.H{"payments": paymentsList})
}

// HandleWebhook godoc
// @Summary      Payment webhook endpoint
// @Description  Receives payment status updates from payment gateways (Razorpay, PagSeguro, etc.)
// @Tags         webhooks
// @Accept       json
// @Produce      json
// @Param        channelId             path   string true "Channel ID"
// @Param        X-Webhook-Signature   header string true "Webhook signature"
// @Param        payload               body   payments.PaymentWebhookPayload true "Webhook payload"
// @Success      200 {object} object{status=string}
// @Failure      400 {object} Response
// @Failure      401 {object} Response
// @Failure      404 {object} Response
// @Failure      500 {object} Response
// @Router       /webhooks/payments/{channelId} [post]
func (h *PaymentsHandler) HandleWebhook(c *gin.Context) {
	channelID := c.Param("id")

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
