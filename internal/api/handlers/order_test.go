package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/msgfy/linktor/internal/domain/entity"
	"github.com/msgfy/linktor/internal/domain/repository"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ============================================================================
// mockOrderRepository
// ============================================================================

type mockOrderRepository struct {
	orders      map[string]*entity.Order
	items       map[string][]entity.OrderItem
	history     map[string][]entity.OrderStatusHistory
	ReturnError error
}

func newMockOrderRepository() *mockOrderRepository {
	return &mockOrderRepository{
		orders:  make(map[string]*entity.Order),
		items:   make(map[string][]entity.OrderItem),
		history: make(map[string][]entity.OrderStatusHistory),
	}
}

func (m *mockOrderRepository) Create(ctx context.Context, order *entity.Order) error {
	if m.ReturnError != nil {
		return m.ReturnError
	}
	m.orders[order.ID] = order
	return nil
}

func (m *mockOrderRepository) GetByID(ctx context.Context, orgID, orderID string) (*entity.Order, error) {
	if m.ReturnError != nil {
		return nil, m.ReturnError
	}
	order, ok := m.orders[orderID]
	if !ok {
		return nil, fmt.Errorf("order not found: %s", orderID)
	}
	return order, nil
}

func (m *mockOrderRepository) GetByMessageID(ctx context.Context, orgID, messageID string) (*entity.Order, error) {
	if m.ReturnError != nil {
		return nil, m.ReturnError
	}
	for _, order := range m.orders {
		if order.MessageID == messageID {
			return order, nil
		}
	}
	return nil, fmt.Errorf("order not found by message ID: %s", messageID)
}

func (m *mockOrderRepository) Update(ctx context.Context, order *entity.Order) error {
	if m.ReturnError != nil {
		return m.ReturnError
	}
	m.orders[order.ID] = order
	return nil
}

func (m *mockOrderRepository) UpdateStatus(ctx context.Context, orgID, orderID string, status entity.OrderStatus) error {
	if m.ReturnError != nil {
		return m.ReturnError
	}
	order, ok := m.orders[orderID]
	if !ok {
		return fmt.Errorf("order not found: %s", orderID)
	}
	order.Status = status
	return nil
}

func (m *mockOrderRepository) Delete(ctx context.Context, orgID, orderID string) error {
	if m.ReturnError != nil {
		return m.ReturnError
	}
	delete(m.orders, orderID)
	return nil
}

func (m *mockOrderRepository) List(ctx context.Context, orgID string, filters repository.OrderFilters, pagination repository.Pagination) ([]*entity.Order, int, error) {
	if m.ReturnError != nil {
		return nil, 0, m.ReturnError
	}
	var result []*entity.Order
	for _, o := range m.orders {
		if o.OrganizationID == orgID {
			result = append(result, o)
		}
	}
	return result, len(result), nil
}

func (m *mockOrderRepository) GetByCustomer(ctx context.Context, orgID, customerPhone string, pagination repository.Pagination) ([]*entity.Order, int, error) {
	if m.ReturnError != nil {
		return nil, 0, m.ReturnError
	}
	var result []*entity.Order
	for _, o := range m.orders {
		if o.OrganizationID == orgID && o.CustomerPhone == customerPhone {
			result = append(result, o)
		}
	}
	return result, len(result), nil
}

func (m *mockOrderRepository) GetByChannel(ctx context.Context, orgID, channelID string, pagination repository.Pagination) ([]*entity.Order, int, error) {
	if m.ReturnError != nil {
		return nil, 0, m.ReturnError
	}
	var result []*entity.Order
	for _, o := range m.orders {
		if o.OrganizationID == orgID && o.ChannelID == channelID {
			result = append(result, o)
		}
	}
	return result, len(result), nil
}

func (m *mockOrderRepository) GetOrderItems(ctx context.Context, orderID string) ([]entity.OrderItem, error) {
	if m.ReturnError != nil {
		return nil, m.ReturnError
	}
	items, ok := m.items[orderID]
	if !ok {
		return []entity.OrderItem{}, nil
	}
	return items, nil
}

func (m *mockOrderRepository) AddOrderItem(ctx context.Context, item *entity.OrderItem) error {
	if m.ReturnError != nil {
		return m.ReturnError
	}
	m.items[item.OrderID] = append(m.items[item.OrderID], *item)
	return nil
}

func (m *mockOrderRepository) UpdateOrderItem(ctx context.Context, item *entity.OrderItem) error {
	if m.ReturnError != nil {
		return m.ReturnError
	}
	return nil
}

func (m *mockOrderRepository) DeleteOrderItem(ctx context.Context, orderID, itemID string) error {
	if m.ReturnError != nil {
		return m.ReturnError
	}
	return nil
}

func (m *mockOrderRepository) GetStatusHistory(ctx context.Context, orderID string) ([]entity.OrderStatusHistory, error) {
	if m.ReturnError != nil {
		return nil, m.ReturnError
	}
	h, ok := m.history[orderID]
	if !ok {
		return []entity.OrderStatusHistory{}, nil
	}
	return h, nil
}

func (m *mockOrderRepository) AddStatusHistory(ctx context.Context, history *entity.OrderStatusHistory) error {
	if m.ReturnError != nil {
		return m.ReturnError
	}
	m.history[history.OrderID] = append(m.history[history.OrderID], *history)
	return nil
}

func (m *mockOrderRepository) GetStats(ctx context.Context, orgID string, filters repository.StatsFilters) (*repository.OrderStats, error) {
	if m.ReturnError != nil {
		return nil, m.ReturnError
	}
	return &repository.OrderStats{
		TotalOrders:  len(m.orders),
		TotalRevenue: 10000,
		Currency:     "BRL",
		ByStatus:     map[entity.OrderStatus]int{entity.OrderStatusPending: len(m.orders)},
	}, nil
}

// ============================================================================
// Tests
// ============================================================================

func TestNewOrderHandler(t *testing.T) {
	repo := newMockOrderRepository()
	h := NewOrderHandler(repo)
	require.NotNil(t, h)
	assert.NotNil(t, h.orderRepo)
}

func TestOrderHandler_ListOrders(t *testing.T) {
	repo := newMockOrderRepository()
	repo.orders["order-1"] = &entity.Order{
		ID:             "order-1",
		OrganizationID: "org-1",
		CustomerPhone:  "+5511999999999",
		Status:         entity.OrderStatusPending,
		Currency:       "BRL",
	}
	h := NewOrderHandler(repo)

	w, c := newTestContext(http.MethodGet, "/orders", nil)
	c.Set("organization_id", "org-1")

	h.ListOrders(c)

	assert.Equal(t, http.StatusOK, w.Code)
	var resp map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.NotNil(t, resp["orders"])
	assert.NotNil(t, resp["pagination"])
}

func TestOrderHandler_GetOrder_Found(t *testing.T) {
	repo := newMockOrderRepository()
	repo.orders["order-1"] = &entity.Order{
		ID:             "order-1",
		OrganizationID: "org-1",
		CustomerPhone:  "+5511999999999",
		Status:         entity.OrderStatusPending,
		Currency:       "BRL",
	}
	h := NewOrderHandler(repo)

	w, c := newTestContext(http.MethodGet, "/orders/order-1", nil)
	c.Set("organization_id", "org-1")
	c.Params = gin.Params{{Key: "orderId", Value: "order-1"}}

	h.GetOrder(c)

	assert.Equal(t, http.StatusOK, w.Code)
	var resp entity.Order
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Equal(t, "order-1", resp.ID)
}

func TestOrderHandler_GetOrder_NotFound(t *testing.T) {
	repo := newMockOrderRepository()
	h := NewOrderHandler(repo)

	w, c := newTestContext(http.MethodGet, "/orders/order-999", nil)
	c.Set("organization_id", "org-1")
	c.Params = gin.Params{{Key: "orderId", Value: "order-999"}}

	h.GetOrder(c)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestOrderHandler_UpdateOrderStatus_Valid(t *testing.T) {
	repo := newMockOrderRepository()
	repo.orders["order-1"] = &entity.Order{
		ID:             "order-1",
		OrganizationID: "org-1",
		Status:         entity.OrderStatusPending,
	}
	h := NewOrderHandler(repo)

	w, c := newTestContext(http.MethodPatch, "/orders/order-1/status", map[string]string{
		"status": "confirmed",
		"notes":  "Order confirmed by admin",
	})
	c.Set("organization_id", "org-1")
	c.Params = gin.Params{{Key: "orderId", Value: "order-1"}}

	h.UpdateOrderStatus(c)

	assert.Equal(t, http.StatusOK, w.Code)
	var resp map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Equal(t, "success", resp["status"])
}

func TestOrderHandler_UpdateOrderStatus_InvalidStatus(t *testing.T) {
	repo := newMockOrderRepository()
	h := NewOrderHandler(repo)

	w, c := newTestContext(http.MethodPatch, "/orders/order-1/status", map[string]string{
		"status": "invalid_status",
	})
	c.Set("organization_id", "org-1")
	c.Params = gin.Params{{Key: "orderId", Value: "order-1"}}

	h.UpdateOrderStatus(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	var resp map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Equal(t, "Invalid status", resp["error"])
}

func TestOrderHandler_UpdateOrderStatus_InvalidBody(t *testing.T) {
	repo := newMockOrderRepository()
	h := NewOrderHandler(repo)

	w, c := newTestContext(http.MethodPatch, "/orders/order-1/status", nil)
	c.Set("organization_id", "org-1")
	c.Params = gin.Params{{Key: "orderId", Value: "order-1"}}

	h.UpdateOrderStatus(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestOrderHandler_GetOrderHistory(t *testing.T) {
	repo := newMockOrderRepository()
	repo.history["order-1"] = []entity.OrderStatusHistory{
		{
			ID:      "hist-1",
			OrderID: "order-1",
			Status:  entity.OrderStatusPending,
			Notes:   "Order created",
		},
	}
	h := NewOrderHandler(repo)

	w, c := newTestContext(http.MethodGet, "/orders/order-1/history", nil)
	c.Params = gin.Params{{Key: "orderId", Value: "order-1"}}

	h.GetOrderHistory(c)

	assert.Equal(t, http.StatusOK, w.Code)
	var resp map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.NotNil(t, resp["history"])
}

func TestOrderHandler_GetOrderStats(t *testing.T) {
	repo := newMockOrderRepository()
	repo.orders["order-1"] = &entity.Order{
		ID:             "order-1",
		OrganizationID: "org-1",
		Status:         entity.OrderStatusPending,
	}
	h := NewOrderHandler(repo)

	w, c := newTestContext(http.MethodGet, "/orders/stats", nil)
	c.Set("organization_id", "org-1")

	h.GetOrderStats(c)

	assert.Equal(t, http.StatusOK, w.Code)
	var resp repository.OrderStats
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Equal(t, 1, resp.TotalOrders)
	assert.Equal(t, "BRL", resp.Currency)
}

func TestOrderHandler_CancelOrder_NotFound(t *testing.T) {
	repo := newMockOrderRepository()
	h := NewOrderHandler(repo)

	w, c := newTestContext(http.MethodPost, "/orders/order-999/cancel", map[string]string{
		"reason": "customer request",
	})
	c.Set("organization_id", "org-1")
	c.Params = gin.Params{{Key: "orderId", Value: "order-999"}}

	h.CancelOrder(c)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestOrderHandler_UpdateShipping_NotFound(t *testing.T) {
	repo := newMockOrderRepository()
	h := NewOrderHandler(repo)

	w, c := newTestContext(http.MethodPatch, "/orders/order-999/shipping", map[string]string{
		"tracking_number": "BR123456789",
	})
	c.Set("organization_id", "org-1")
	c.Params = gin.Params{{Key: "orderId", Value: "order-999"}}

	h.UpdateShipping(c)

	assert.Equal(t, http.StatusNotFound, w.Code)
}
