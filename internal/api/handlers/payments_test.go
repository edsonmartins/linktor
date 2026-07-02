package handlers

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/msgfy/linktor/internal/api/middleware"
	"github.com/msgfy/linktor/internal/domain/entity"
	"github.com/msgfy/linktor/internal/whatsapp/payments"
	"github.com/msgfy/linktor/pkg/testutil"
)

func TestNewPaymentsHandler(t *testing.T) {
	h := NewPaymentsHandler(testutil.NewMockChannelRepository())
	require.NotNil(t, h)
	assert.NotNil(t, h.clients)
}

func TestPaymentsHandler_RegisterClient(t *testing.T) {
	h := NewPaymentsHandler(testutil.NewMockChannelRepository())
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
	h := NewPaymentsHandler(testutil.NewMockChannelRepository())
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
	h := NewPaymentsHandler(testutil.NewMockChannelRepository())
	w, c := newTestContext(http.MethodGet, "/channels/channel-1/payments/pay-1", nil)
	c.Params = gin.Params{{Key: "id", Value: "channel-1"}, {Key: "paymentId", Value: "pay-1"}}

	h.GetPayment(c)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestPaymentsHandler_GetPaymentByReference_NoClient(t *testing.T) {
	h := NewPaymentsHandler(testutil.NewMockChannelRepository())
	w, c := newTestContext(http.MethodGet, "/channels/channel-1/payments/reference/ref-123", nil)
	c.Params = gin.Params{{Key: "id", Value: "channel-1"}, {Key: "referenceId", Value: "ref-123"}}

	h.GetPaymentByReference(c)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestPaymentsHandler_ProcessRefund_NoClient(t *testing.T) {
	h := NewPaymentsHandler(testutil.NewMockChannelRepository())
	w, c := newTestContext(http.MethodPost, "/channels/channel-1/payments/pay-1/refund", map[string]interface{}{
		"amount": 500,
		"reason": "customer request",
	})
	c.Params = gin.Params{{Key: "id", Value: "channel-1"}, {Key: "paymentId", Value: "pay-1"}}

	h.ProcessRefund(c)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestPaymentsHandler_GetPaymentStats_NoClient(t *testing.T) {
	h := NewPaymentsHandler(testutil.NewMockChannelRepository())
	w, c := newTestContext(http.MethodGet, "/channels/channel-1/payments/stats", nil)
	c.Params = gin.Params{{Key: "id", Value: "channel-1"}}

	h.GetPaymentStats(c)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestPaymentsHandler_GetCustomerPayments_NoClient(t *testing.T) {
	h := NewPaymentsHandler(testutil.NewMockChannelRepository())
	w, c := newTestContext(http.MethodGet, "/channels/channel-1/payments/customer/+5511999999999", nil)
	c.Params = gin.Params{{Key: "id", Value: "channel-1"}, {Key: "phone", Value: "+5511999999999"}}

	h.GetCustomerPayments(c)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

// TestPaymentsHandler_ProcessRefund_CrossTenantRejected proves the tenant-isolation
// guard: tenant-a cannot trigger a refund on a channel owned by tenant-b. A client
// is registered for the channel, so if the guard were missing the handler would
// dereference it and panic — reaching a 404 without panic proves the refund never
// executes cross-tenant.
func TestPaymentsHandler_ProcessRefund_CrossTenantRejected(t *testing.T) {
	repo := testutil.NewMockChannelRepository()
	repo.Channels["channel-b"] = &entity.Channel{ID: "channel-b", TenantID: "tenant-b"}

	h := NewPaymentsHandler(repo)
	h.RegisterClient("channel-b", (*payments.Client)(nil))

	w, c := newTestContext(http.MethodPost, "/channels/channel-b/payments/pay-1/refund", map[string]interface{}{
		"amount": 500,
		"reason": "attacker refund",
	})
	c.Set(middleware.TenantIDKey, "tenant-a")
	c.Params = gin.Params{{Key: "id", Value: "channel-b"}, {Key: "paymentId", Value: "pay-1"}}

	h.ProcessRefund(c)

	assert.Equal(t, http.StatusNotFound, w.Code)
	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	// Rejected by the tenant guard, not by client resolution.
	assert.NotContains(t, resp["error"], "not configured")
}

// TestPaymentsHandler_ProcessRefund_SameTenantPassesGuard proves the guard admits
// the owning tenant: the request passes tenant validation and only then fails at
// client resolution ("payments not configured"), confirming the guard is not a
// blanket denial.
func TestPaymentsHandler_ProcessRefund_SameTenantPassesGuard(t *testing.T) {
	repo := testutil.NewMockChannelRepository()
	repo.Channels["channel-a"] = &entity.Channel{ID: "channel-a", TenantID: "tenant-a"}

	h := NewPaymentsHandler(repo)

	w, c := newTestContext(http.MethodPost, "/channels/channel-a/payments/pay-1/refund", map[string]interface{}{
		"amount": 500,
	})
	c.Set(middleware.TenantIDKey, "tenant-a")
	c.Params = gin.Params{{Key: "id", Value: "channel-a"}, {Key: "paymentId", Value: "pay-1"}}

	h.ProcessRefund(c)

	assert.Equal(t, http.StatusNotFound, w.Code)
	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Contains(t, resp["error"], "not configured")
}

func TestPaymentsHandler_HandleWebhook_NoClient(t *testing.T) {
	h := NewPaymentsHandler(testutil.NewMockChannelRepository())
	w, c := newTestContext(http.MethodPost, "/webhooks/payments/channel-1", map[string]string{"event": "payment.completed"})
	c.Params = gin.Params{{Key: "id", Value: "channel-1"}}

	h.HandleWebhook(c)

	assert.Equal(t, http.StatusNotFound, w.Code)
}
