package payments

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

// mockPaymentStore is a mock implementation of PaymentStore for testing
type mockPaymentStore struct {
	payments map[string]*Payment
}

func newMockPaymentStore() *mockPaymentStore {
	return &mockPaymentStore{payments: make(map[string]*Payment)}
}

func (m *mockPaymentStore) Create(ctx context.Context, payment *Payment) error {
	m.payments[payment.ID] = payment
	return nil
}

func (m *mockPaymentStore) GetByID(ctx context.Context, id string) (*Payment, error) {
	p, ok := m.payments[id]
	if !ok {
		return nil, fmt.Errorf("payment not found: %s", id)
	}
	return p, nil
}

func (m *mockPaymentStore) GetByReference(ctx context.Context, refID string) (*Payment, error) {
	for _, p := range m.payments {
		if p.ReferenceID == refID {
			return p, nil
		}
	}
	return nil, fmt.Errorf("payment not found for reference: %s", refID)
}

func (m *mockPaymentStore) Update(ctx context.Context, payment *Payment) error {
	m.payments[payment.ID] = payment
	return nil
}

func (m *mockPaymentStore) GetByCustomer(ctx context.Context, phone string) ([]*Payment, error) {
	var result []*Payment
	for _, p := range m.payments {
		if p.CustomerPhone == phone {
			result = append(result, p)
		}
	}
	return result, nil
}

func (m *mockPaymentStore) GetStats(ctx context.Context, orgID string) (*PaymentStats, error) {
	stats := &PaymentStats{
		ByStatus: make(map[PaymentStatus]int),
		ByMethod: make(map[PaymentMethod]int),
	}
	for _, p := range m.payments {
		stats.TotalPayments++
		stats.ByStatus[p.Status]++
		if p.Method != "" {
			stats.ByMethod[p.Method]++
		}
		switch p.Status {
		case PaymentStatusSuccess:
			stats.SuccessfulPayments++
			stats.TotalAmount += p.Amount
			if stats.Currency == "" {
				stats.Currency = p.Currency
			}
		case PaymentStatusFailed:
			stats.FailedPayments++
		case PaymentStatusRefunded:
			stats.RefundedAmount += p.Amount
		}
	}
	if stats.TotalPayments > 0 {
		stats.SuccessRate = float64(stats.SuccessfulPayments) / float64(stats.TotalPayments) * 100
	}
	return stats, nil
}

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
		Store: newMockPaymentStore(),
	}
	suite.client = NewClient(config)
}

// NewClient tests
func (suite *ClientTestSuite) TestNewClient_WithValidConfig() {
	config := &ClientConfig{
		AccessToken:   "test-token",
		PhoneNumberID: "test-phone",
		Store:         newMockPaymentStore(),
	}

	client := NewClient(config)

	assert.NotNil(suite.T(), client)
	assert.NotNil(suite.T(), client.httpClient)
	assert.NotNil(suite.T(), client.store)
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
	payment, err := suite.client.GetPayment(context.Background(), "non-existent")

	assert.Error(suite.T(), err)
	assert.Nil(suite.T(), payment)
}

func (suite *ClientTestSuite) TestGetPaymentByReference_NotFound() {
	payment, err := suite.client.GetPaymentByReference(context.Background(), "non-existent")

	assert.Error(suite.T(), err)
	assert.Nil(suite.T(), payment)
}

func (suite *ClientTestSuite) TestUpdatePaymentStatus() {
	// Add a test payment
	payment := &Payment{
		ID:        "test-payment",
		Status:    PaymentStatusPending,
		UpdatedAt: time.Now(),
	}
	suite.client.store.Create(context.Background(), payment)

	err := suite.client.UpdatePaymentStatus(context.Background(), "test-payment", PaymentStatusSuccess)

	assert.NoError(suite.T(), err)

	updated, err := suite.client.GetPayment(context.Background(), "test-payment")
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), PaymentStatusSuccess, updated.Status)
	assert.NotNil(suite.T(), updated.PaidAt)
}

func (suite *ClientTestSuite) TestUpdatePaymentStatus_Failed() {
	payment := &Payment{
		ID:        "test-payment-failed",
		Status:    PaymentStatusPending,
		UpdatedAt: time.Now(),
	}
	suite.client.store.Create(context.Background(), payment)

	err := suite.client.UpdatePaymentStatus(context.Background(), "test-payment-failed", PaymentStatusFailed)

	assert.NoError(suite.T(), err)

	updated, err := suite.client.GetPayment(context.Background(), "test-payment-failed")
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), PaymentStatusFailed, updated.Status)
	assert.NotNil(suite.T(), updated.FailedAt)
}

func (suite *ClientTestSuite) TestUpdatePaymentStatus_NotFound() {
	err := suite.client.UpdatePaymentStatus(context.Background(), "non-existent", PaymentStatusSuccess)

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
	suite.client.store.Create(context.Background(), payment)

	payload := &PaymentWebhookPayload{
		PaymentID: "test-payment-webhook",
		Status:    PaymentStatusSuccess,
		Method:    PaymentMethodPix,
	}

	err := suite.client.ProcessWebhook(context.Background(), payload)

	assert.NoError(suite.T(), err)

	updated, err := suite.client.GetPayment(context.Background(), "test-payment-webhook")
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), PaymentStatusSuccess, updated.Status)
	assert.Equal(suite.T(), PaymentMethodPix, updated.Method)
}

func (suite *ClientTestSuite) TestProcessWebhook_NotFound() {
	payload := &PaymentWebhookPayload{
		PaymentID: "non-existent",
		Status:    PaymentStatusSuccess,
	}

	err := suite.client.ProcessWebhook(context.Background(), payload)

	assert.Error(suite.T(), err)
	assert.Contains(suite.T(), err.Error(), "payment not found")
}

// Statistics tests
func (suite *ClientTestSuite) TestGetPaymentStats_Empty() {
	stats, err := suite.client.GetPaymentStats(context.Background())

	assert.NoError(suite.T(), err)
	assert.NotNil(suite.T(), stats)
	assert.Equal(suite.T(), 0, stats.TotalPayments)
	assert.Equal(suite.T(), 0, stats.SuccessfulPayments)
}

func (suite *ClientTestSuite) TestGetPaymentStats_WithPayments() {
	// Add test payments
	suite.client.store.Create(context.Background(), &Payment{
		ID:       "p1",
		Status:   PaymentStatusSuccess,
		Amount:   10000,
		Currency: "BRL",
		Method:   PaymentMethodPix,
	})
	suite.client.store.Create(context.Background(), &Payment{
		ID:       "p2",
		Status:   PaymentStatusFailed,
		Amount:   5000,
		Currency: "BRL",
	})
	suite.client.store.Create(context.Background(), &Payment{
		ID:       "p3",
		Status:   PaymentStatusSuccess,
		Amount:   15000,
		Currency: "BRL",
		Method:   PaymentMethodPix,
	})

	stats, err := suite.client.GetPaymentStats(context.Background())

	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), 3, stats.TotalPayments)
	assert.Equal(suite.T(), 2, stats.SuccessfulPayments)
	assert.Equal(suite.T(), 1, stats.FailedPayments)
	assert.Equal(suite.T(), int64(25000), stats.TotalAmount)
	assert.InDelta(suite.T(), 66.67, stats.SuccessRate, 0.01)
}

// Customer payments tests
func (suite *ClientTestSuite) TestGetPaymentsByCustomer() {
	suite.client.store.Create(context.Background(), &Payment{
		ID:            "p1",
		CustomerPhone: "5511999999999",
	})
	suite.client.store.Create(context.Background(), &Payment{
		ID:            "p2",
		CustomerPhone: "5511888888888",
	})
	suite.client.store.Create(context.Background(), &Payment{
		ID:            "p3",
		CustomerPhone: "5511999999999",
	})

	payments, err := suite.client.GetPaymentsByCustomer(context.Background(), "5511999999999")

	assert.NoError(suite.T(), err)
	assert.Len(suite.T(), payments, 2)
}

// CreatePayment tests
func (suite *ClientTestSuite) TestCreatePayment_NoGateway() {
	config := &ClientConfig{
		AccessToken:   "test-token",
		PhoneNumberID: "test-phone",
		Store:         newMockPaymentStore(),
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
	suite.client.store.Create(context.Background(), &Payment{
		ID:     "pending-payment",
		Status: PaymentStatusPending,
		Amount: 10000,
	})

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
