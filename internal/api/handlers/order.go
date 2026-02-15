package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/msgfy/linktor/internal/domain/entity"
	"github.com/msgfy/linktor/internal/domain/repository"
)

// OrderHandler handles order-related HTTP requests
type OrderHandler struct {
	orderRepo repository.OrderRepository
}

// NewOrderHandler creates a new order handler
func NewOrderHandler(orderRepo repository.OrderRepository) *OrderHandler {
	return &OrderHandler{
		orderRepo: orderRepo,
	}
}

// ListOrders handles GET /orders
func (h *OrderHandler) ListOrders(c *gin.Context) {
	ctx := c.Request.Context()
	orgID := c.GetString("organization_id")

	// Parse query parameters
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 20
	}

	pagination := repository.Pagination{
		Page:     page,
		PageSize: pageSize,
	}

	// Parse filters
	filters := repository.OrderFilters{
		CustomerPhone: c.Query("customer_phone"),
		ChannelID:     c.Query("channel_id"),
		CatalogID:     c.Query("catalog_id"),
		Search:        c.Query("search"),
	}

	if status := c.Query("status"); status != "" {
		s := entity.OrderStatus(status)
		filters.Status = &s
	}

	orders, total, err := h.orderRepo.List(ctx, orgID, filters, pagination)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"orders": orders,
		"pagination": gin.H{
			"page":        page,
			"page_size":   pageSize,
			"total":       total,
			"total_pages": (total + pageSize - 1) / pageSize,
		},
	})
}

// GetOrder handles GET /orders/:orderId
func (h *OrderHandler) GetOrder(c *gin.Context) {
	ctx := c.Request.Context()
	orgID := c.GetString("organization_id")
	orderID := c.Param("orderId")

	order, err := h.orderRepo.GetByID(ctx, orgID, orderID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Order not found"})
		return
	}

	// Get order items
	items, err := h.orderRepo.GetOrderItems(ctx, orderID)
	if err == nil {
		order.Items = items
	}

	c.JSON(http.StatusOK, order)
}

// UpdateOrderStatus handles PATCH /orders/:orderId/status
func (h *OrderHandler) UpdateOrderStatus(c *gin.Context) {
	ctx := c.Request.Context()
	orgID := c.GetString("organization_id")
	orderID := c.Param("orderId")

	var req struct {
		Status string `json:"status" binding:"required"`
		Notes  string `json:"notes,omitempty"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	status := entity.OrderStatus(req.Status)

	// Validate status
	validStatuses := map[entity.OrderStatus]bool{
		entity.OrderStatusPending:    true,
		entity.OrderStatusConfirmed:  true,
		entity.OrderStatusProcessing: true,
		entity.OrderStatusShipped:    true,
		entity.OrderStatusDelivered:  true,
		entity.OrderStatusCompleted:  true,
		entity.OrderStatusCancelled:  true,
		entity.OrderStatusRefunded:   true,
	}
	if !validStatuses[status] {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid status"})
		return
	}

	if err := h.orderRepo.UpdateStatus(ctx, orgID, orderID, status); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Add status history
	history := &entity.OrderStatusHistory{
		OrderID: orderID,
		Status:  status,
		Notes:   req.Notes,
	}
	h.orderRepo.AddStatusHistory(ctx, history)

	c.JSON(http.StatusOK, gin.H{
		"status":  "success",
		"message": "Order status updated",
	})
}

// GetOrderHistory handles GET /orders/:orderId/history
func (h *OrderHandler) GetOrderHistory(c *gin.Context) {
	ctx := c.Request.Context()
	orderID := c.Param("orderId")

	history, err := h.orderRepo.GetStatusHistory(ctx, orderID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"history": history})
}

// GetOrderStats handles GET /orders/stats
func (h *OrderHandler) GetOrderStats(c *gin.Context) {
	ctx := c.Request.Context()
	orgID := c.GetString("organization_id")

	filters := repository.StatsFilters{
		ChannelID: c.Query("channel_id"),
	}

	stats, err := h.orderRepo.GetStats(ctx, orgID, filters)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, stats)
}

// CancelOrder handles POST /orders/:orderId/cancel
func (h *OrderHandler) CancelOrder(c *gin.Context) {
	ctx := c.Request.Context()
	orgID := c.GetString("organization_id")
	orderID := c.Param("orderId")

	var req struct {
		Reason string `json:"reason,omitempty"`
	}
	c.ShouldBindJSON(&req)

	// Get order to check if it can be cancelled
	order, err := h.orderRepo.GetByID(ctx, orgID, orderID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Order not found"})
		return
	}

	if !order.CanCancel() {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Order cannot be cancelled"})
		return
	}

	if err := h.orderRepo.UpdateStatus(ctx, orgID, orderID, entity.OrderStatusCancelled); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Add status history
	history := &entity.OrderStatusHistory{
		OrderID: orderID,
		Status:  entity.OrderStatusCancelled,
		Notes:   req.Reason,
	}
	h.orderRepo.AddStatusHistory(ctx, history)

	c.JSON(http.StatusOK, gin.H{
		"status":  "success",
		"message": "Order cancelled",
	})
}

// UpdateShipping handles PATCH /orders/:orderId/shipping
func (h *OrderHandler) UpdateShipping(c *gin.Context) {
	ctx := c.Request.Context()
	orgID := c.GetString("organization_id")
	orderID := c.Param("orderId")

	var req struct {
		TrackingNumber string `json:"tracking_number" binding:"required"`
		TrackingURL    string `json:"tracking_url,omitempty"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	order, err := h.orderRepo.GetByID(ctx, orgID, orderID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Order not found"})
		return
	}

	order.TrackingNumber = req.TrackingNumber
	order.TrackingURL = req.TrackingURL

	if err := h.orderRepo.Update(ctx, order); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":  "success",
		"message": "Shipping info updated",
	})
}

// GetCustomerOrders handles GET /customers/:phone/orders
func (h *OrderHandler) GetCustomerOrders(c *gin.Context) {
	ctx := c.Request.Context()
	orgID := c.GetString("organization_id")
	phone := c.Param("phone")

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 20
	}

	pagination := repository.Pagination{
		Page:     page,
		PageSize: pageSize,
	}

	orders, total, err := h.orderRepo.GetByCustomer(ctx, orgID, phone, pagination)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"orders": orders,
		"pagination": gin.H{
			"page":        page,
			"page_size":   pageSize,
			"total":       total,
			"total_pages": (total + pageSize - 1) / pageSize,
		},
	})
}
