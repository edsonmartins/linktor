package handlers

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewPaymentsHandler(t *testing.T) {
	h := NewPaymentsHandler()
	require.NotNil(t, h)
	assert.NotNil(t, h.clients)
}

func TestPaymentsHandler_RegisterClient(t *testing.T) {
	h := NewPaymentsHandler()
	// Register a nil client just to verify the map works
	h.RegisterClient("channel-1", nil)
	client, ok := h.getClient("channel-1")
	assert.True(t, ok)
	assert.Nil(t, client)

	// Non-existent channel
	_, ok = h.getClient("channel-999")
	assert.False(t, ok)
}

func TestPaymentsHandler_CreatePayment_NoClient(t *testing.T) {
	h := NewPaymentsHandler()
	w, c := newTestContext(http.MethodPost, "/channels/channel-1/payments", map[string]interface{}{
		"to":           "+5511999999999",
		"amount":       1000,
		"currency":     "BRL",
		"reference_id": "ref-123",
	})
	c.Params = gin.Params{{Key: "id", Value: "channel-1"}}

	h.CreatePayment(c)

	assert.Equal(t, http.StatusNotFound, w.Code)
	var resp map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Contains(t, resp["error"], "not found")
}

func TestPaymentsHandler_GetPayment_NoClient(t *testing.T) {
	h := NewPaymentsHandler()
	w, c := newTestContext(http.MethodGet, "/channels/channel-1/payments/pay-1", nil)
	c.Params = gin.Params{{Key: "id", Value: "channel-1"}, {Key: "paymentId", Value: "pay-1"}}

	h.GetPayment(c)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestPaymentsHandler_GetPaymentByReference_NoClient(t *testing.T) {
	h := NewPaymentsHandler()
	w, c := newTestContext(http.MethodGet, "/channels/channel-1/payments/reference/ref-123", nil)
	c.Params = gin.Params{{Key: "id", Value: "channel-1"}, {Key: "referenceId", Value: "ref-123"}}

	h.GetPaymentByReference(c)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestPaymentsHandler_ProcessRefund_NoClient(t *testing.T) {
	h := NewPaymentsHandler()
	w, c := newTestContext(http.MethodPost, "/channels/channel-1/payments/pay-1/refund", map[string]interface{}{
		"amount": 500,
		"reason": "customer request",
	})
	c.Params = gin.Params{{Key: "id", Value: "channel-1"}, {Key: "paymentId", Value: "pay-1"}}

	h.ProcessRefund(c)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestPaymentsHandler_GetPaymentStats_NoClient(t *testing.T) {
	h := NewPaymentsHandler()
	w, c := newTestContext(http.MethodGet, "/channels/channel-1/payments/stats", nil)
	c.Params = gin.Params{{Key: "id", Value: "channel-1"}}

	h.GetPaymentStats(c)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestPaymentsHandler_GetCustomerPayments_NoClient(t *testing.T) {
	h := NewPaymentsHandler()
	w, c := newTestContext(http.MethodGet, "/channels/channel-1/payments/customer/+5511999999999", nil)
	c.Params = gin.Params{{Key: "id", Value: "channel-1"}, {Key: "phone", Value: "+5511999999999"}}

	h.GetCustomerPayments(c)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestPaymentsHandler_HandleWebhook_NoClient(t *testing.T) {
	h := NewPaymentsHandler()
	w, c := newTestContext(http.MethodPost, "/webhooks/payments/channel-1", map[string]string{"event": "payment.completed"})
	c.Params = gin.Params{{Key: "id", Value: "channel-1"}}

	h.HandleWebhook(c)

	assert.Equal(t, http.StatusNotFound, w.Code)
}
