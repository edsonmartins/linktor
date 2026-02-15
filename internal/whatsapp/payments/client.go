package payments

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"
)

// Client handles WhatsApp Payments API interactions
type Client struct {
	httpClient    *http.Client
	accessToken   string
	phoneNumberID string
	apiVersion    string
	baseURL       string

	gateway       Gateway
	gatewayConfig *GatewayConfig

	mu       sync.RWMutex
	payments map[string]*Payment // In-memory storage, should be replaced with DB
}

// ClientConfig represents configuration for the payments client
type ClientConfig struct {
	AccessToken   string
	PhoneNumberID string
	APIVersion    string
	GatewayConfig *GatewayConfig
}

// Gateway defines the interface for payment gateways
type Gateway interface {
	CreatePayment(ctx context.Context, req *PaymentRequest) (*PaymentResponse, error)
	GetPaymentStatus(ctx context.Context, paymentID string) (*Payment, error)
	ProcessRefund(ctx context.Context, req *RefundRequest) (*Refund, error)
	ValidateWebhook(payload []byte, signature string) bool
}

// NewClient creates a new payments client
func NewClient(config *ClientConfig) *Client {
	apiVersion := config.APIVersion
	if apiVersion == "" {
		apiVersion = "v21.0"
	}

	client := &Client{
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		accessToken:   config.AccessToken,
		phoneNumberID: config.PhoneNumberID,
		apiVersion:    apiVersion,
		baseURL:       "https://graph.facebook.com",
		gatewayConfig: config.GatewayConfig,
		payments:      make(map[string]*Payment),
	}

	// Initialize gateway based on config
	if config.GatewayConfig != nil {
		client.gateway = client.initGateway(config.GatewayConfig)
	}

	return client
}

// initGateway initializes the appropriate payment gateway
func (c *Client) initGateway(config *GatewayConfig) Gateway {
	switch config.Type {
	case GatewayRazorpay:
		return NewRazorpayGateway(config)
	case GatewayPagSeguro:
		return NewPagSeguroGateway(config)
	default:
		return NewMockGateway(config)
	}
}

// buildURL builds the API URL
func (c *Client) buildURL(path string) string {
	return fmt.Sprintf("%s/%s%s", c.baseURL, c.apiVersion, path)
}

// doRequest executes an HTTP request
func (c *Client) doRequest(ctx context.Context, method, url string, body interface{}) ([]byte, error) {
	var bodyReader io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request body: %w", err)
		}
		bodyReader = bytes.NewReader(data)
	}

	req, err := http.NewRequestWithContext(ctx, method, url, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.accessToken)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	if resp.StatusCode >= 400 {
		var errResp struct {
			Error struct {
				Message string `json:"message"`
				Code    int    `json:"code"`
			} `json:"error"`
		}
		json.Unmarshal(respBody, &errResp)
		return nil, fmt.Errorf("API error (status %d): %s", resp.StatusCode, errResp.Error.Message)
	}

	return respBody, nil
}

// =============================================================================
// Payment Operations
// =============================================================================

// CreatePayment creates a new payment request
func (c *Client) CreatePayment(ctx context.Context, req *PaymentRequest) (*PaymentResponse, error) {
	if c.gateway == nil {
		return nil, fmt.Errorf("payment gateway not configured")
	}

	// Create payment via gateway
	gatewayResp, err := c.gateway.CreatePayment(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("gateway error: %w", err)
	}

	// Store payment
	payment := &Payment{
		ID:               gatewayResp.PaymentID,
		ReferenceID:      req.ReferenceID,
		CustomerPhone:    req.To,
		Amount:           req.Amount,
		Currency:         req.Currency,
		Status:           PaymentStatusPending,
		Type:             req.Type,
		Description:      req.Description,
		GatewayPaymentID: gatewayResp.PaymentID,
		ExpiresAt:        gatewayResp.ExpiresAt,
		CreatedAt:        time.Now(),
		UpdatedAt:        time.Now(),
	}

	c.mu.Lock()
	c.payments[payment.ID] = payment
	c.mu.Unlock()

	// Send payment message via WhatsApp
	messageID, err := c.sendPaymentMessage(ctx, req, gatewayResp)
	if err != nil {
		// Payment was created but message failed - non-fatal error
		// The payment.MessageID will remain empty
		_ = err // Ignore error, payment was created successfully
	} else {
		payment.MessageID = messageID
		gatewayResp.MessageID = messageID
	}

	return gatewayResp, nil
}

// sendPaymentMessage sends a payment request message via WhatsApp
func (c *Client) sendPaymentMessage(ctx context.Context, req *PaymentRequest, resp *PaymentResponse) (string, error) {
	apiURL := c.buildURL(fmt.Sprintf("/%s/messages", c.phoneNumberID))

	// Build interactive order message
	body := map[string]interface{}{
		"messaging_product": "whatsapp",
		"recipient_type":    "individual",
		"to":                req.To,
		"type":              "interactive",
		"interactive": map[string]interface{}{
			"type": "order_details",
			"body": map[string]string{
				"text": req.Description,
			},
			"action": map[string]interface{}{
				"name": "review_and_pay",
				"parameters": map[string]interface{}{
					"reference_id":   req.ReferenceID,
					"type":           "digital-goods",
					"payment_status": "pending",
					"order": map[string]interface{}{
						"status": "pending",
						"items":  c.buildItemsPayload(req.Items),
						"subtotal": map[string]interface{}{
							"value":  req.Amount,
							"offset": 100,
						},
						"total": map[string]interface{}{
							"value":  req.Amount,
							"offset": 100,
						},
					},
					"payment_settings": c.buildPaymentSettings(req),
				},
			},
		},
	}

	respBody, err := c.doRequest(ctx, http.MethodPost, apiURL, body)
	if err != nil {
		return "", err
	}

	var result struct {
		Messages []struct {
			ID string `json:"id"`
		} `json:"messages"`
	}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return "", fmt.Errorf("failed to parse response: %w", err)
	}

	if len(result.Messages) == 0 {
		return "", fmt.Errorf("no message ID in response")
	}

	return result.Messages[0].ID, nil
}

// buildItemsPayload builds the items payload for WhatsApp message
func (c *Client) buildItemsPayload(items []PaymentItem) []map[string]interface{} {
	result := make([]map[string]interface{}, len(items))
	for i, item := range items {
		result[i] = map[string]interface{}{
			"retailer_id": fmt.Sprintf("item_%d", i),
			"name":        item.Name,
			"amount": map[string]interface{}{
				"value":  item.TotalPrice,
				"offset": 100,
			},
			"quantity": item.Quantity,
		}
	}
	return result
}

// buildPaymentSettings builds payment settings for WhatsApp message
func (c *Client) buildPaymentSettings(req *PaymentRequest) map[string]interface{} {
	settings := make(map[string]interface{})

	if c.gatewayConfig != nil {
		settings["payment_gateway"] = c.gatewayConfig.Type
	}

	if req.PaymentSettings != nil {
		if req.PaymentSettings.PixDetails != nil {
			settings["pix"] = map[string]interface{}{
				"key":      req.PaymentSettings.PixDetails.Key,
				"key_type": req.PaymentSettings.PixDetails.KeyType,
			}
		}
		if req.PaymentSettings.UPIDetails != nil {
			settings["upi"] = map[string]interface{}{
				"vpa": req.PaymentSettings.UPIDetails.VPA,
			}
		}
	}

	return settings
}

// GetPayment retrieves a payment by ID
func (c *Client) GetPayment(paymentID string) (*Payment, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	payment, ok := c.payments[paymentID]
	return payment, ok
}

// GetPaymentByReference retrieves a payment by reference ID
func (c *Client) GetPaymentByReference(referenceID string) (*Payment, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	for _, payment := range c.payments {
		if payment.ReferenceID == referenceID {
			return payment, true
		}
	}
	return nil, false
}

// UpdatePaymentStatus updates the status of a payment
func (c *Client) UpdatePaymentStatus(paymentID string, status PaymentStatus) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	payment, ok := c.payments[paymentID]
	if !ok {
		return fmt.Errorf("payment not found: %s", paymentID)
	}

	payment.Status = status
	payment.UpdatedAt = time.Now()

	now := time.Now()
	switch status {
	case PaymentStatusSuccess:
		payment.PaidAt = &now
	case PaymentStatusFailed:
		payment.FailedAt = &now
	case PaymentStatusRefunded:
		payment.RefundedAt = &now
	}

	return nil
}

// =============================================================================
// Refund Operations
// =============================================================================

// ProcessRefund processes a refund for a payment
func (c *Client) ProcessRefund(ctx context.Context, req *RefundRequest) (*Refund, error) {
	if c.gateway == nil {
		return nil, fmt.Errorf("payment gateway not configured")
	}

	// Get payment
	payment, ok := c.GetPayment(req.PaymentID)
	if !ok {
		return nil, fmt.Errorf("payment not found: %s", req.PaymentID)
	}

	// Validate refund
	if payment.Status != PaymentStatusSuccess {
		return nil, fmt.Errorf("payment cannot be refunded: status is %s", payment.Status)
	}

	// Set amount to full payment if not specified
	if req.Amount == 0 {
		req.Amount = payment.Amount
	}

	// Process refund via gateway
	refund, err := c.gateway.ProcessRefund(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("gateway error: %w", err)
	}

	// Update payment status
	if req.Amount == payment.Amount {
		c.UpdatePaymentStatus(req.PaymentID, PaymentStatusRefunded)
	}

	// Send refund notification
	c.sendRefundNotification(ctx, payment, refund)

	return refund, nil
}

// sendRefundNotification sends a refund notification message
func (c *Client) sendRefundNotification(ctx context.Context, payment *Payment, refund *Refund) {
	message := fmt.Sprintf("Your refund of %.2f %s has been processed for payment %s.",
		float64(refund.Amount)/100, refund.Currency, payment.ReferenceID)

	apiURL := c.buildURL(fmt.Sprintf("/%s/messages", c.phoneNumberID))

	body := map[string]interface{}{
		"messaging_product": "whatsapp",
		"recipient_type":    "individual",
		"to":                payment.CustomerPhone,
		"type":              "text",
		"text": map[string]string{
			"body": message,
		},
	}

	c.doRequest(ctx, http.MethodPost, apiURL, body)
}

// =============================================================================
// Webhook Handling
// =============================================================================

// ProcessWebhook processes a payment webhook
func (c *Client) ProcessWebhook(payload *PaymentWebhookPayload) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Find payment by ID or reference
	var payment *Payment
	if p, ok := c.payments[payload.PaymentID]; ok {
		payment = p
	} else {
		for _, p := range c.payments {
			if p.ReferenceID == payload.ReferenceID {
				payment = p
				break
			}
		}
	}

	if payment == nil {
		return fmt.Errorf("payment not found for webhook: %s", payload.PaymentID)
	}

	// Update payment status
	payment.Status = payload.Status
	payment.Method = payload.Method
	payment.UpdatedAt = time.Now()

	now := time.Now()
	switch payload.Status {
	case PaymentStatusSuccess:
		payment.PaidAt = payload.PaidAt
		if payment.PaidAt == nil {
			payment.PaidAt = &now
		}
	case PaymentStatusFailed:
		payment.FailedAt = &now
		payment.FailureReason = payload.FailureReason
	case PaymentStatusRefunded:
		payment.RefundedAt = &now
	}

	return nil
}

// ValidateWebhookSignature validates a webhook signature
func (c *Client) ValidateWebhookSignature(payload []byte, signature string) bool {
	if c.gatewayConfig == nil || c.gatewayConfig.WebhookSecret == "" {
		return false
	}

	mac := hmac.New(sha256.New, []byte(c.gatewayConfig.WebhookSecret))
	mac.Write(payload)
	expected := hex.EncodeToString(mac.Sum(nil))

	return hmac.Equal([]byte(expected), []byte(signature))
}

// =============================================================================
// Statistics
// =============================================================================

// GetPaymentStats returns payment statistics
func (c *Client) GetPaymentStats() *PaymentStats {
	c.mu.RLock()
	defer c.mu.RUnlock()

	stats := &PaymentStats{
		ByStatus: make(map[PaymentStatus]int),
		ByMethod: make(map[PaymentMethod]int),
	}

	for _, payment := range c.payments {
		stats.TotalPayments++
		stats.ByStatus[payment.Status]++

		if payment.Method != "" {
			stats.ByMethod[payment.Method]++
		}

		switch payment.Status {
		case PaymentStatusSuccess:
			stats.SuccessfulPayments++
			stats.TotalAmount += payment.Amount
			if stats.Currency == "" {
				stats.Currency = payment.Currency
			}
		case PaymentStatusFailed:
			stats.FailedPayments++
		case PaymentStatusRefunded:
			stats.RefundedAmount += payment.Amount
		}
	}

	if stats.TotalPayments > 0 {
		stats.SuccessRate = float64(stats.SuccessfulPayments) / float64(stats.TotalPayments) * 100
	}

	return stats
}

// GetPaymentsByCustomer returns payments for a customer
func (c *Client) GetPaymentsByCustomer(customerPhone string) []*Payment {
	c.mu.RLock()
	defer c.mu.RUnlock()

	var result []*Payment
	for _, payment := range c.payments {
		if payment.CustomerPhone == customerPhone {
			result = append(result, payment)
		}
	}
	return result
}

// =============================================================================
// Gateway Implementations
// =============================================================================

// MockGateway is a mock payment gateway for testing
type MockGateway struct {
	config *GatewayConfig
}

// NewMockGateway creates a new mock gateway
func NewMockGateway(config *GatewayConfig) *MockGateway {
	return &MockGateway{config: config}
}

// CreatePayment creates a mock payment
func (g *MockGateway) CreatePayment(ctx context.Context, req *PaymentRequest) (*PaymentResponse, error) {
	expiresAt := time.Now().Add(30 * time.Minute)
	if req.ExpiresIn > 0 {
		expiresAt = time.Now().Add(req.ExpiresIn)
	}

	return &PaymentResponse{
		PaymentID:  fmt.Sprintf("mock_pay_%d", time.Now().UnixNano()),
		Status:     PaymentStatusPending,
		PaymentURL: fmt.Sprintf("https://mock-payment.example.com/pay/%s", req.ReferenceID),
		ExpiresAt:  &expiresAt,
	}, nil
}

// GetPaymentStatus gets mock payment status
func (g *MockGateway) GetPaymentStatus(ctx context.Context, paymentID string) (*Payment, error) {
	return &Payment{
		ID:     paymentID,
		Status: PaymentStatusPending,
	}, nil
}

// ProcessRefund processes a mock refund
func (g *MockGateway) ProcessRefund(ctx context.Context, req *RefundRequest) (*Refund, error) {
	now := time.Now()
	return &Refund{
		ID:          fmt.Sprintf("mock_ref_%d", time.Now().UnixNano()),
		PaymentID:   req.PaymentID,
		Amount:      req.Amount,
		Status:      RefundStatusSuccess,
		Reason:      req.Reason,
		CreatedAt:   now,
		ProcessedAt: &now,
	}, nil
}

// ValidateWebhook validates mock webhook
func (g *MockGateway) ValidateWebhook(payload []byte, signature string) bool {
	return true
}

// =============================================================================
// Razorpay Gateway (India)
// =============================================================================

// RazorpayGateway implements Gateway for Razorpay
type RazorpayGateway struct {
	config     *GatewayConfig
	httpClient *http.Client
	baseURL    string
}

// NewRazorpayGateway creates a new Razorpay gateway
func NewRazorpayGateway(config *GatewayConfig) *RazorpayGateway {
	baseURL := "https://api.razorpay.com/v1"
	if config.SandboxMode {
		baseURL = "https://api.razorpay.com/v1" // Razorpay uses same URL with test keys
	}

	return &RazorpayGateway{
		config: config,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		baseURL: baseURL,
	}
}

// CreatePayment creates a Razorpay payment
func (g *RazorpayGateway) CreatePayment(ctx context.Context, req *PaymentRequest) (*PaymentResponse, error) {
	// Create Razorpay order
	orderReq := map[string]interface{}{
		"amount":   req.Amount, // Already in paise
		"currency": req.Currency,
		"receipt":  req.ReferenceID,
		"notes": map[string]string{
			"description": req.Description,
		},
	}

	body, _ := json.Marshal(orderReq)
	httpReq, _ := http.NewRequestWithContext(ctx, "POST", g.baseURL+"/orders", bytes.NewReader(body))
	httpReq.SetBasicAuth(g.config.APIKey, g.config.APISecret)
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := g.httpClient.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result struct {
		ID     string `json:"id"`
		Status string `json:"status"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	return &PaymentResponse{
		PaymentID: result.ID,
		Status:    PaymentStatusPending,
	}, nil
}

// GetPaymentStatus gets Razorpay payment status
func (g *RazorpayGateway) GetPaymentStatus(ctx context.Context, paymentID string) (*Payment, error) {
	httpReq, _ := http.NewRequestWithContext(ctx, "GET", g.baseURL+"/payments/"+paymentID, nil)
	httpReq.SetBasicAuth(g.config.APIKey, g.config.APISecret)

	resp, err := g.httpClient.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result struct {
		ID     string `json:"id"`
		Status string `json:"status"`
		Amount int64  `json:"amount"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	status := PaymentStatusPending
	switch result.Status {
	case "captured":
		status = PaymentStatusSuccess
	case "failed":
		status = PaymentStatusFailed
	case "refunded":
		status = PaymentStatusRefunded
	}

	return &Payment{
		ID:     result.ID,
		Amount: result.Amount,
		Status: status,
	}, nil
}

// ProcessRefund processes a Razorpay refund
func (g *RazorpayGateway) ProcessRefund(ctx context.Context, req *RefundRequest) (*Refund, error) {
	refundReq := map[string]interface{}{
		"amount": req.Amount,
	}
	if req.Reason != "" {
		refundReq["notes"] = map[string]string{"reason": req.Reason}
	}

	body, _ := json.Marshal(refundReq)
	httpReq, _ := http.NewRequestWithContext(ctx, "POST",
		g.baseURL+"/payments/"+req.PaymentID+"/refund", bytes.NewReader(body))
	httpReq.SetBasicAuth(g.config.APIKey, g.config.APISecret)
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := g.httpClient.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result struct {
		ID     string `json:"id"`
		Amount int64  `json:"amount"`
		Status string `json:"status"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	return &Refund{
		ID:              result.ID,
		PaymentID:       req.PaymentID,
		Amount:          result.Amount,
		Status:          RefundStatusSuccess,
		GatewayRefundID: result.ID,
		CreatedAt:       time.Now(),
	}, nil
}

// ValidateWebhook validates Razorpay webhook signature
func (g *RazorpayGateway) ValidateWebhook(payload []byte, signature string) bool {
	mac := hmac.New(sha256.New, []byte(g.config.WebhookSecret))
	mac.Write(payload)
	expected := hex.EncodeToString(mac.Sum(nil))
	return hmac.Equal([]byte(expected), []byte(signature))
}

// =============================================================================
// PagSeguro Gateway (Brazil)
// =============================================================================

// PagSeguroGateway implements Gateway for PagSeguro
type PagSeguroGateway struct {
	config     *GatewayConfig
	httpClient *http.Client
	baseURL    string
}

// NewPagSeguroGateway creates a new PagSeguro gateway
func NewPagSeguroGateway(config *GatewayConfig) *PagSeguroGateway {
	baseURL := "https://api.pagseguro.com"
	if config.SandboxMode {
		baseURL = "https://sandbox.api.pagseguro.com"
	}

	return &PagSeguroGateway{
		config: config,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		baseURL: baseURL,
	}
}

// CreatePayment creates a PagSeguro payment (Pix)
func (g *PagSeguroGateway) CreatePayment(ctx context.Context, req *PaymentRequest) (*PaymentResponse, error) {
	// Create Pix charge
	chargeReq := map[string]interface{}{
		"reference_id": req.ReferenceID,
		"description":  req.Description,
		"amount": map[string]interface{}{
			"value":    req.Amount,
			"currency": req.Currency,
		},
		"payment_method": map[string]interface{}{
			"type": "PIX",
		},
	}

	if req.ExpiresIn > 0 {
		chargeReq["expiration_date"] = time.Now().Add(req.ExpiresIn).Format(time.RFC3339)
	}

	body, _ := json.Marshal(chargeReq)
	httpReq, _ := http.NewRequestWithContext(ctx, "POST", g.baseURL+"/charges", bytes.NewReader(body))
	httpReq.Header.Set("Authorization", "Bearer "+g.config.APIKey)
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := g.httpClient.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result struct {
		ID     string `json:"id"`
		Status string `json:"status"`
		QRCodes []struct {
			Text   string `json:"text"`
			Images []struct {
				URL string `json:"url"`
			} `json:"images"`
		} `json:"qr_codes"`
		Links []struct {
			Href string `json:"href"`
			Rel  string `json:"rel"`
		} `json:"links"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	response := &PaymentResponse{
		PaymentID: result.ID,
		Status:    PaymentStatusPending,
	}

	// Extract QR code if available
	if len(result.QRCodes) > 0 {
		response.QRCode = result.QRCodes[0].Text
		if len(result.QRCodes[0].Images) > 0 {
			response.QRCodeBase64 = result.QRCodes[0].Images[0].URL
		}
	}

	// Extract payment URL
	for _, link := range result.Links {
		if link.Rel == "PAY" {
			response.PaymentURL = link.Href
			break
		}
	}

	return response, nil
}

// GetPaymentStatus gets PagSeguro payment status
func (g *PagSeguroGateway) GetPaymentStatus(ctx context.Context, paymentID string) (*Payment, error) {
	httpReq, _ := http.NewRequestWithContext(ctx, "GET", g.baseURL+"/charges/"+paymentID, nil)
	httpReq.Header.Set("Authorization", "Bearer "+g.config.APIKey)

	resp, err := g.httpClient.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result struct {
		ID     string `json:"id"`
		Status string `json:"status"`
		Amount struct {
			Value int64 `json:"value"`
		} `json:"amount"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	status := PaymentStatusPending
	switch result.Status {
	case "PAID":
		status = PaymentStatusSuccess
	case "CANCELED", "DECLINED":
		status = PaymentStatusFailed
	}

	return &Payment{
		ID:     result.ID,
		Amount: result.Amount.Value,
		Status: status,
	}, nil
}

// ProcessRefund processes a PagSeguro refund
func (g *PagSeguroGateway) ProcessRefund(ctx context.Context, req *RefundRequest) (*Refund, error) {
	refundReq := map[string]interface{}{
		"amount": map[string]interface{}{
			"value": req.Amount,
		},
	}

	body, _ := json.Marshal(refundReq)
	httpReq, _ := http.NewRequestWithContext(ctx, "POST",
		g.baseURL+"/charges/"+req.PaymentID+"/cancel", bytes.NewReader(body))
	httpReq.Header.Set("Authorization", "Bearer "+g.config.APIKey)
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := g.httpClient.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result struct {
		ID     string `json:"id"`
		Status string `json:"status"`
		Amount struct {
			Value int64 `json:"value"`
		} `json:"amount"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	return &Refund{
		ID:              result.ID,
		PaymentID:       req.PaymentID,
		Amount:          result.Amount.Value,
		Status:          RefundStatusSuccess,
		GatewayRefundID: result.ID,
		CreatedAt:       time.Now(),
	}, nil
}

// ValidateWebhook validates PagSeguro webhook signature
func (g *PagSeguroGateway) ValidateWebhook(payload []byte, signature string) bool {
	// PagSeguro uses a different validation method
	// For now, basic validation
	return signature != ""
}
