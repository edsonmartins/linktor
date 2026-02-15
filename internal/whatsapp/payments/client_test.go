package payments

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

// generateTestSignature generates HMAC-SHA256 signature for testing
func generateTestSignature(secret string, payload []byte) string {
	if secret == "" {
		return ""
	}
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(payload)
	return hex.EncodeToString(mac.Sum(nil))
}

// ClientTestSuite tests the payments client
type ClientTestSuite struct {
	suite.Suite
	client *Client
}

func TestClientTestSuite(t *testing.T) {
	suite.Run(t, new(ClientTestSuite))
}

func (suite *ClientTestSuite) SetupTest() {
	config := &ClientConfig{
		AccessToken:   "test-token",
		PhoneNumberID: "test-phone",
		APIVersion:    "v21.0",
		GatewayConfig: &GatewayConfig{
			Type:          GatewayRazorpay,
			APIKey:        "test-key",
			APISecret:     "test-secret",
			WebhookSecret: "webhook-secret",
			SandboxMode:   true,
		},
	}
	suite.client = NewClient(config)
}

// NewClient tests
func (suite *ClientTestSuite) TestNewClient_WithValidConfig() {
	config := &ClientConfig{
		AccessToken:   "test-token",
		PhoneNumberID: "test-phone",
	}

	client := NewClient(config)

	assert.NotNil(suite.T(), client)
	assert.NotNil(suite.T(), client.httpClient)
	assert.NotNil(suite.T(), client.payments)
}

func (suite *ClientTestSuite) TestNewClient_DefaultsAPIVersion() {
	config := &ClientConfig{
		AccessToken:   "test-token",
		PhoneNumberID: "test-phone",
		APIVersion:    "",
	}

	client := NewClient(config)

	assert.Equal(suite.T(), "v21.0", client.apiVersion)
}

func (suite *ClientTestSuite) TestNewClient_InitializesGateway() {
	config := &ClientConfig{
		AccessToken:   "test-token",
		PhoneNumberID: "test-phone",
		GatewayConfig: &GatewayConfig{
			Type:      GatewayRazorpay,
			APIKey:    "test-key",
			APISecret: "test-secret",
		},
	}

	client := NewClient(config)

	assert.NotNil(suite.T(), client.gateway)
}

func (suite *ClientTestSuite) TestNewClient_WithoutGateway() {
	config := &ClientConfig{
		AccessToken:   "test-token",
		PhoneNumberID: "test-phone",
	}

	client := NewClient(config)

	assert.Nil(suite.T(), client.gateway)
}

// Payment storage tests
func (suite *ClientTestSuite) TestGetPayment_NotFound() {
	payment, found := suite.client.GetPayment("non-existent")

	assert.False(suite.T(), found)
	assert.Nil(suite.T(), payment)
}

func (suite *ClientTestSuite) TestGetPaymentByReference_NotFound() {
	payment, found := suite.client.GetPaymentByReference("non-existent")

	assert.False(suite.T(), found)
	assert.Nil(suite.T(), payment)
}

func (suite *ClientTestSuite) TestUpdatePaymentStatus() {
	// Add a test payment
	payment := &Payment{
		ID:        "test-payment",
		Status:    PaymentStatusPending,
		UpdatedAt: time.Now(),
	}
	suite.client.mu.Lock()
	suite.client.payments[payment.ID] = payment
	suite.client.mu.Unlock()

	err := suite.client.UpdatePaymentStatus("test-payment", PaymentStatusSuccess)

	assert.NoError(suite.T(), err)

	updated, found := suite.client.GetPayment("test-payment")
	assert.True(suite.T(), found)
	assert.Equal(suite.T(), PaymentStatusSuccess, updated.Status)
	assert.NotNil(suite.T(), updated.PaidAt)
}

func (suite *ClientTestSuite) TestUpdatePaymentStatus_Failed() {
	payment := &Payment{
		ID:        "test-payment-failed",
		Status:    PaymentStatusPending,
		UpdatedAt: time.Now(),
	}
	suite.client.mu.Lock()
	suite.client.payments[payment.ID] = payment
	suite.client.mu.Unlock()

	err := suite.client.UpdatePaymentStatus("test-payment-failed", PaymentStatusFailed)

	assert.NoError(suite.T(), err)

	updated, found := suite.client.GetPayment("test-payment-failed")
	assert.True(suite.T(), found)
	assert.Equal(suite.T(), PaymentStatusFailed, updated.Status)
	assert.NotNil(suite.T(), updated.FailedAt)
}

func (suite *ClientTestSuite) TestUpdatePaymentStatus_NotFound() {
	err := suite.client.UpdatePaymentStatus("non-existent", PaymentStatusSuccess)

	assert.Error(suite.T(), err)
	assert.Contains(suite.T(), err.Error(), "payment not found")
}

// Webhook validation tests
func (suite *ClientTestSuite) TestValidateWebhookSignature_Valid() {
	payload := []byte(`{"payment_id":"test"}`)
	// Generate expected signature using the test helper
	expectedSig := generateTestSignature("webhook-secret", payload)

	valid := suite.client.ValidateWebhookSignature(payload, expectedSig)

	assert.True(suite.T(), valid)
}

func (suite *ClientTestSuite) TestValidateWebhookSignature_Invalid() {
	payload := []byte(`{"payment_id":"test"}`)

	valid := suite.client.ValidateWebhookSignature(payload, "invalid-signature")

	assert.False(suite.T(), valid)
}

func (suite *ClientTestSuite) TestValidateWebhookSignature_NoSecret() {
	config := &ClientConfig{
		AccessToken:   "test-token",
		PhoneNumberID: "test-phone",
	}
	client := NewClient(config)

	payload := []byte(`{"payment_id":"test"}`)

	valid := client.ValidateWebhookSignature(payload, "any-signature")

	assert.False(suite.T(), valid)
}

// ProcessWebhook tests
func (suite *ClientTestSuite) TestProcessWebhook_ExistingPayment() {
	payment := &Payment{
		ID:        "test-payment-webhook",
		Status:    PaymentStatusPending,
		UpdatedAt: time.Now(),
	}
	suite.client.mu.Lock()
	suite.client.payments[payment.ID] = payment
	suite.client.mu.Unlock()

	payload := &PaymentWebhookPayload{
		PaymentID: "test-payment-webhook",
		Status:    PaymentStatusSuccess,
		Method:    PaymentMethodPix,
	}

	err := suite.client.ProcessWebhook(payload)

	assert.NoError(suite.T(), err)

	updated, found := suite.client.GetPayment("test-payment-webhook")
	assert.True(suite.T(), found)
	assert.Equal(suite.T(), PaymentStatusSuccess, updated.Status)
	assert.Equal(suite.T(), PaymentMethodPix, updated.Method)
}

func (suite *ClientTestSuite) TestProcessWebhook_NotFound() {
	payload := &PaymentWebhookPayload{
		PaymentID: "non-existent",
		Status:    PaymentStatusSuccess,
	}

	err := suite.client.ProcessWebhook(payload)

	assert.Error(suite.T(), err)
	assert.Contains(suite.T(), err.Error(), "payment not found")
}

// Statistics tests
func (suite *ClientTestSuite) TestGetPaymentStats_Empty() {
	stats := suite.client.GetPaymentStats()

	assert.NotNil(suite.T(), stats)
	assert.Equal(suite.T(), 0, stats.TotalPayments)
	assert.Equal(suite.T(), 0, stats.SuccessfulPayments)
}

func (suite *ClientTestSuite) TestGetPaymentStats_WithPayments() {
	// Add test payments
	suite.client.mu.Lock()
	suite.client.payments["p1"] = &Payment{
		ID:       "p1",
		Status:   PaymentStatusSuccess,
		Amount:   10000,
		Currency: "BRL",
		Method:   PaymentMethodPix,
	}
	suite.client.payments["p2"] = &Payment{
		ID:       "p2",
		Status:   PaymentStatusFailed,
		Amount:   5000,
		Currency: "BRL",
	}
	suite.client.payments["p3"] = &Payment{
		ID:       "p3",
		Status:   PaymentStatusSuccess,
		Amount:   15000,
		Currency: "BRL",
		Method:   PaymentMethodPix,
	}
	suite.client.mu.Unlock()

	stats := suite.client.GetPaymentStats()

	assert.Equal(suite.T(), 3, stats.TotalPayments)
	assert.Equal(suite.T(), 2, stats.SuccessfulPayments)
	assert.Equal(suite.T(), 1, stats.FailedPayments)
	assert.Equal(suite.T(), int64(25000), stats.TotalAmount)
	assert.InDelta(suite.T(), 66.67, stats.SuccessRate, 0.01)
}

// Customer payments tests
func (suite *ClientTestSuite) TestGetPaymentsByCustomer() {
	suite.client.mu.Lock()
	suite.client.payments["p1"] = &Payment{
		ID:            "p1",
		CustomerPhone: "5511999999999",
	}
	suite.client.payments["p2"] = &Payment{
		ID:            "p2",
		CustomerPhone: "5511888888888",
	}
	suite.client.payments["p3"] = &Payment{
		ID:            "p3",
		CustomerPhone: "5511999999999",
	}
	suite.client.mu.Unlock()

	payments := suite.client.GetPaymentsByCustomer("5511999999999")

	assert.Len(suite.T(), payments, 2)
}

// CreatePayment tests
func (suite *ClientTestSuite) TestCreatePayment_NoGateway() {
	config := &ClientConfig{
		AccessToken:   "test-token",
		PhoneNumberID: "test-phone",
	}
	client := NewClient(config)

	req := &PaymentRequest{
		To:          "5511999999999",
		Amount:      10000,
		Currency:    "BRL",
		ReferenceID: "ref-123",
	}

	_, err := client.CreatePayment(context.Background(), req)

	assert.Error(suite.T(), err)
	assert.Contains(suite.T(), err.Error(), "gateway not configured")
}

// ProcessRefund tests
func (suite *ClientTestSuite) TestProcessRefund_PaymentNotFound() {
	req := &RefundRequest{
		PaymentID: "non-existent",
		Amount:    5000,
	}

	_, err := suite.client.ProcessRefund(context.Background(), req)

	assert.Error(suite.T(), err)
	assert.Contains(suite.T(), err.Error(), "payment not found")
}

func (suite *ClientTestSuite) TestProcessRefund_InvalidStatus() {
	// Add payment with pending status
	suite.client.mu.Lock()
	suite.client.payments["pending-payment"] = &Payment{
		ID:     "pending-payment",
		Status: PaymentStatusPending,
		Amount: 10000,
	}
	suite.client.mu.Unlock()

	req := &RefundRequest{
		PaymentID: "pending-payment",
		Amount:    5000,
	}

	_, err := suite.client.ProcessRefund(context.Background(), req)

	assert.Error(suite.T(), err)
	assert.Contains(suite.T(), err.Error(), "cannot be refunded")
}

// Helper to generate test signature - moved to avoid import inside function
// Note: This is a test helper that duplicates the signature logic for testing

// Mock Gateway tests
func TestMockGateway_CreatePayment(t *testing.T) {
	config := &GatewayConfig{
		Type:   GatewayRazorpay,
		APIKey: "test-key",
	}
	gateway := NewMockGateway(config)

	req := &PaymentRequest{
		To:          "5511999999999",
		Amount:      10000,
		Currency:    "BRL",
		ReferenceID: "ref-123",
	}

	resp, err := gateway.CreatePayment(context.Background(), req)

	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Contains(t, resp.PaymentID, "mock_pay_")
	assert.Equal(t, PaymentStatusPending, resp.Status)
}

func TestMockGateway_ProcessRefund(t *testing.T) {
	config := &GatewayConfig{
		Type:   GatewayRazorpay,
		APIKey: "test-key",
	}
	gateway := NewMockGateway(config)

	req := &RefundRequest{
		PaymentID: "test-payment",
		Amount:    5000,
		Reason:    "Customer request",
	}

	refund, err := gateway.ProcessRefund(context.Background(), req)

	assert.NoError(t, err)
	assert.NotNil(t, refund)
	assert.Contains(t, refund.ID, "mock_ref_")
	assert.Equal(t, RefundStatusSuccess, refund.Status)
}

func TestMockGateway_ValidateWebhook(t *testing.T) {
	config := &GatewayConfig{
		Type:   GatewayRazorpay,
		APIKey: "test-key",
	}
	gateway := NewMockGateway(config)

	// Mock gateway always returns true
	valid := gateway.ValidateWebhook([]byte("test"), "any-signature")

	assert.True(t, valid)
}
