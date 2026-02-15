package commerce

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"
)

// OrderManager handles order operations
type OrderManager struct {
	httpClient    *http.Client
	accessToken   string
	phoneNumberID string
	apiVersion    string
	baseURL       string

	mu     sync.RWMutex
	orders map[string]*Order // In-memory storage, should be replaced with DB
}

// OrderManagerConfig represents configuration for the order manager
type OrderManagerConfig struct {
	AccessToken   string
	PhoneNumberID string
	APIVersion    string
}

// NewOrderManager creates a new order manager
func NewOrderManager(config *OrderManagerConfig) *OrderManager {
	apiVersion := config.APIVersion
	if apiVersion == "" {
		apiVersion = "v21.0"
	}

	return &OrderManager{
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		accessToken:   config.AccessToken,
		phoneNumberID: config.PhoneNumberID,
		apiVersion:    apiVersion,
		baseURL:       "https://graph.facebook.com",
		orders:        make(map[string]*Order),
	}
}

// buildURL builds the API URL
func (om *OrderManager) buildURL(path string) string {
	return fmt.Sprintf("%s/%s%s", om.baseURL, om.apiVersion, path)
}

// doRequest executes an HTTP request
func (om *OrderManager) doRequest(ctx context.Context, method, url string, body interface{}) ([]byte, error) {
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

	req.Header.Set("Authorization", "Bearer "+om.accessToken)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := om.httpClient.Do(req)
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
// Order Processing from Webhook
// =============================================================================

// ProcessOrderWebhook processes an order received from webhook
func (om *OrderManager) ProcessOrderWebhook(payload *OrderWebhookPayload, customerPhone, messageID string) (*Order, error) {
	order := &Order{
		ID:            generateOrderID(),
		CatalogID:     payload.CatalogID,
		CustomerPhone: customerPhone,
		Status:        OrderStatusPending,
		Items:         make([]OrderItem, 0, len(payload.Order.ProductItems)),
		Currency:      "BRL", // Default, will be overridden
		Notes:         payload.Order.Text,
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
		MessageID:     messageID,
	}

	var subtotal int64
	for _, item := range payload.Order.ProductItems {
		price := parsePrice(item.ItemPrice)
		totalPrice := price * int64(item.Quantity)

		orderItem := OrderItem{
			ProductID:  item.ProductRetailerID,
			Quantity:   item.Quantity,
			UnitPrice:  price,
			Currency:   item.Currency,
			TotalPrice: totalPrice,
		}
		order.Items = append(order.Items, orderItem)
		subtotal += totalPrice

		if item.Currency != "" {
			order.Currency = item.Currency
		}
	}

	order.Subtotal = subtotal
	order.Total = subtotal // Will be updated with tax/shipping/discount

	om.mu.Lock()
	om.orders[order.ID] = order
	om.mu.Unlock()

	return order, nil
}

// parsePrice parses a price string like "10.99", "R$ 10,99", "10,99 BRL" to cents (1099)
func parsePrice(priceStr string) int64 {
	// Remove currency symbols, codes, and spaces
	cleaned := strings.TrimSpace(priceStr)

	// Remove common currency symbols and codes
	replacements := []string{
		"R$", "BRL", "USD", "$", "€", "EUR", "£", "GBP",
	}
	for _, r := range replacements {
		cleaned = strings.ReplaceAll(cleaned, r, "")
	}

	// Remove spaces
	cleaned = strings.TrimSpace(cleaned)
	cleaned = strings.ReplaceAll(cleaned, " ", "")

	// Handle Brazilian format (1.234,56) vs US format (1,234.56)
	// If there's both comma and dot, determine which is decimal separator
	hasComma := strings.Contains(cleaned, ",")
	hasDot := strings.Contains(cleaned, ".")

	if hasComma && hasDot {
		// Check which comes last - that's the decimal separator
		lastComma := strings.LastIndex(cleaned, ",")
		lastDot := strings.LastIndex(cleaned, ".")

		if lastComma > lastDot {
			// Brazilian format: 1.234,56
			cleaned = strings.ReplaceAll(cleaned, ".", "")
			cleaned = strings.ReplaceAll(cleaned, ",", ".")
		} else {
			// US format: 1,234.56
			cleaned = strings.ReplaceAll(cleaned, ",", "")
		}
	} else if hasComma {
		// Assume comma is decimal separator (Brazilian format)
		cleaned = strings.ReplaceAll(cleaned, ",", ".")
	}
	// If only dot, it's already in correct format

	// Parse as float and convert to cents
	price, err := strconv.ParseFloat(cleaned, 64)
	if err != nil {
		return 0
	}
	return int64(price * 100)
}

// generateOrderID generates a unique order ID
func generateOrderID() string {
	return fmt.Sprintf("ORD-%d", time.Now().UnixNano())
}

// =============================================================================
// Order Management
// =============================================================================

// GetOrder retrieves an order by ID
func (om *OrderManager) GetOrder(orderID string) (*Order, bool) {
	om.mu.RLock()
	defer om.mu.RUnlock()

	order, ok := om.orders[orderID]
	return order, ok
}

// GetOrdersByCustomer retrieves orders for a customer
func (om *OrderManager) GetOrdersByCustomer(customerPhone string) []*Order {
	om.mu.RLock()
	defer om.mu.RUnlock()

	var orders []*Order
	for _, order := range om.orders {
		if order.CustomerPhone == customerPhone {
			orders = append(orders, order)
		}
	}
	return orders
}

// UpdateOrderStatus updates the status of an order
func (om *OrderManager) UpdateOrderStatus(orderID string, status OrderStatus) error {
	om.mu.Lock()
	defer om.mu.Unlock()

	order, ok := om.orders[orderID]
	if !ok {
		return fmt.Errorf("order not found: %s", orderID)
	}

	order.Status = status
	order.UpdatedAt = time.Now()
	return nil
}

// AddShippingAddress adds shipping address to an order
func (om *OrderManager) AddShippingAddress(orderID string, address *Address) error {
	om.mu.Lock()
	defer om.mu.Unlock()

	order, ok := om.orders[orderID]
	if !ok {
		return fmt.Errorf("order not found: %s", orderID)
	}

	order.ShippingAddress = address
	order.UpdatedAt = time.Now()
	return nil
}

// UpdateOrderTotals updates the order totals
func (om *OrderManager) UpdateOrderTotals(orderID string, tax, shipping, discount int64) error {
	om.mu.Lock()
	defer om.mu.Unlock()

	order, ok := om.orders[orderID]
	if !ok {
		return fmt.Errorf("order not found: %s", orderID)
	}

	order.Tax = tax
	order.Shipping = shipping
	order.Discount = discount
	order.Total = order.Subtotal + tax + shipping - discount
	order.UpdatedAt = time.Now()
	return nil
}

// CancelOrder cancels an order
func (om *OrderManager) CancelOrder(orderID string) error {
	return om.UpdateOrderStatus(orderID, OrderStatusCancelled)
}

// =============================================================================
// Order Status Messages
// =============================================================================

// SendOrderConfirmation sends an order confirmation message
func (om *OrderManager) SendOrderConfirmation(ctx context.Context, to string, confirmation *OrderConfirmation) (string, error) {
	apiURL := om.buildURL(fmt.Sprintf("/%s/messages", om.phoneNumberID))

	// Build order status message
	text := fmt.Sprintf("*Order Confirmation*\n\nOrder ID: %s\nStatus: %s",
		confirmation.OrderID, confirmation.Status)

	if confirmation.Description != "" {
		text += fmt.Sprintf("\n\n%s", confirmation.Description)
	}

	if confirmation.EstimatedDelivery != "" {
		text += fmt.Sprintf("\n\nEstimated Delivery: %s", confirmation.EstimatedDelivery)
	}

	if confirmation.TrackingNumber != "" {
		text += fmt.Sprintf("\n\nTracking Number: %s", confirmation.TrackingNumber)
	}

	body := map[string]interface{}{
		"messaging_product": "whatsapp",
		"recipient_type":    "individual",
		"to":                to,
		"type":              "text",
		"text": map[string]string{
			"body": text,
		},
	}

	respBody, err := om.doRequest(ctx, http.MethodPost, apiURL, body)
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

// SendOrderStatusUpdate sends an order status update using interactive message
func (om *OrderManager) SendOrderStatusUpdate(ctx context.Context, msg *OrderDetailsMessage) (string, error) {
	apiURL := om.buildURL(fmt.Sprintf("/%s/messages", om.phoneNumberID))

	body := map[string]interface{}{
		"messaging_product": "whatsapp",
		"recipient_type":    "individual",
		"to":                msg.To,
		"type":              "interactive",
		"interactive": map[string]interface{}{
			"type": "order_status",
			"body": map[string]string{
				"text": msg.Description,
			},
			"action": map[string]interface{}{
				"name": "review_order",
				"parameters": map[string]interface{}{
					"reference_id":   msg.ReferenceID,
					"order_status":   msg.Status,
					"payment_status": msg.PaymentStatus,
				},
			},
		},
	}

	respBody, err := om.doRequest(ctx, http.MethodPost, apiURL, body)
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

// =============================================================================
// Order Summary
// =============================================================================

// GetOrderSummary returns a formatted order summary
func (om *OrderManager) GetOrderSummary(orderID string) (string, error) {
	om.mu.RLock()
	order, ok := om.orders[orderID]
	om.mu.RUnlock()

	if !ok {
		return "", fmt.Errorf("order not found: %s", orderID)
	}

	summary := fmt.Sprintf("*Order Summary*\n\nOrder ID: %s\nStatus: %s\n\n*Items:*\n",
		order.ID, order.Status)

	for _, item := range order.Items {
		itemTotal := float64(item.TotalPrice) / 100
		summary += fmt.Sprintf("- %s x%d: %.2f %s\n",
			item.ProductID, item.Quantity, itemTotal, item.Currency)
	}

	subtotal := float64(order.Subtotal) / 100
	total := float64(order.Total) / 100

	summary += fmt.Sprintf("\nSubtotal: %.2f %s", subtotal, order.Currency)

	if order.Tax > 0 {
		tax := float64(order.Tax) / 100
		summary += fmt.Sprintf("\nTax: %.2f %s", tax, order.Currency)
	}

	if order.Shipping > 0 {
		shipping := float64(order.Shipping) / 100
		summary += fmt.Sprintf("\nShipping: %.2f %s", shipping, order.Currency)
	}

	if order.Discount > 0 {
		discount := float64(order.Discount) / 100
		summary += fmt.Sprintf("\nDiscount: -%.2f %s", discount, order.Currency)
	}

	summary += fmt.Sprintf("\n\n*Total: %.2f %s*", total, order.Currency)

	return summary, nil
}

// =============================================================================
// Order Statistics
// =============================================================================

// OrderStats represents order statistics
type OrderStats struct {
	TotalOrders     int                    `json:"total_orders"`
	PendingOrders   int                    `json:"pending_orders"`
	CompletedOrders int                    `json:"completed_orders"`
	CancelledOrders int                    `json:"cancelled_orders"`
	TotalRevenue    int64                  `json:"total_revenue"`
	Currency        string                 `json:"currency"`
	ByStatus        map[OrderStatus]int    `json:"by_status"`
}

// GetOrderStats returns order statistics
func (om *OrderManager) GetOrderStats() *OrderStats {
	om.mu.RLock()
	defer om.mu.RUnlock()

	stats := &OrderStats{
		ByStatus: make(map[OrderStatus]int),
	}

	for _, order := range om.orders {
		stats.TotalOrders++
		stats.ByStatus[order.Status]++

		switch order.Status {
		case OrderStatusPending:
			stats.PendingOrders++
		case OrderStatusCompleted:
			stats.CompletedOrders++
			stats.TotalRevenue += order.Total
			if stats.Currency == "" {
				stats.Currency = order.Currency
			}
		case OrderStatusCancelled:
			stats.CancelledOrders++
		}
	}

	return stats
}

// Note: OrderRepository interface is defined in internal/domain/repository/order.go
// Use that interface for persistence operations

// =============================================================================
// Order Events
// =============================================================================

// OrderEvent represents an order event for pub/sub
type OrderEvent struct {
	Type      string    `json:"type"`
	OrderID   string    `json:"order_id"`
	Status    string    `json:"status,omitempty"`
	Timestamp time.Time `json:"timestamp"`
	Data      *Order    `json:"data,omitempty"`
}

// OrderEventType constants
const (
	OrderEventCreated   = "order.created"
	OrderEventUpdated   = "order.updated"
	OrderEventConfirmed = "order.confirmed"
	OrderEventShipped   = "order.shipped"
	OrderEventDelivered = "order.delivered"
	OrderEventCancelled = "order.cancelled"
)

// NewOrderEvent creates a new order event
func NewOrderEvent(eventType string, order *Order) *OrderEvent {
	return &OrderEvent{
		Type:      eventType,
		OrderID:   order.ID,
		Status:    string(order.Status),
		Timestamp: time.Now(),
		Data:      order,
	}
}
