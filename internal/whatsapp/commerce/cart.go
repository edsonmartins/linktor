package commerce

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"
)

// CartManager handles shopping cart operations
type CartManager struct {
	httpClient    *http.Client
	accessToken   string
	phoneNumberID string
	apiVersion    string
	baseURL       string

	mu               sync.RWMutex
	carts            map[string]*Cart // key: customer_phone
	abandonedCarts   map[string]*Cart // key: cart_id
	cartTTL          time.Duration
	abandonThreshold time.Duration
}

// CartManagerConfig represents configuration for the cart manager
type CartManagerConfig struct {
	AccessToken      string
	PhoneNumberID    string
	APIVersion       string
	CartTTL          time.Duration // How long carts live before expiring
	AbandonThreshold time.Duration // Time after which inactive carts are considered abandoned
}

// NewCartManager creates a new cart manager
func NewCartManager(config *CartManagerConfig) *CartManager {
	apiVersion := config.APIVersion
	if apiVersion == "" {
		apiVersion = "v21.0"
	}

	cartTTL := config.CartTTL
	if cartTTL == 0 {
		cartTTL = 24 * time.Hour
	}

	abandonThreshold := config.AbandonThreshold
	if abandonThreshold == 0 {
		abandonThreshold = 1 * time.Hour
	}

	return &CartManager{
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		accessToken:      config.AccessToken,
		phoneNumberID:    config.PhoneNumberID,
		apiVersion:       apiVersion,
		baseURL:          "https://graph.facebook.com",
		carts:            make(map[string]*Cart),
		abandonedCarts:   make(map[string]*Cart),
		cartTTL:          cartTTL,
		abandonThreshold: abandonThreshold,
	}
}

// buildURL builds the API URL
func (cm *CartManager) buildURL(path string) string {
	return fmt.Sprintf("%s/%s%s", cm.baseURL, cm.apiVersion, path)
}

// doRequest executes an HTTP request
func (cm *CartManager) doRequest(ctx context.Context, method, url string, body interface{}) ([]byte, error) {
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

	req.Header.Set("Authorization", "Bearer "+cm.accessToken)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := cm.httpClient.Do(req)
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
// Cart Operations
// =============================================================================

// GetOrCreateCart gets or creates a cart for a customer
func (cm *CartManager) GetOrCreateCart(customerPhone, catalogID string) *Cart {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	cart, ok := cm.carts[customerPhone]
	if ok && !cart.IsExpired() {
		cart.UpdatedAt = time.Now()
		return cart
	}

	// Create new cart
	cart = &Cart{
		ID:            generateCartID(),
		CustomerPhone: customerPhone,
		CatalogID:     catalogID,
		Items:         make([]CartItem, 0),
		Currency:      "BRL",
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
		ExpiresAt:     time.Now().Add(cm.cartTTL),
	}
	cm.carts[customerPhone] = cart

	return cart
}

// GetCart retrieves a cart for a customer
func (cm *CartManager) GetCart(customerPhone string) (*Cart, bool) {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	cart, ok := cm.carts[customerPhone]
	if !ok || cart.IsExpired() {
		return nil, false
	}
	return cart, true
}

// AddItem adds an item to the cart
func (cm *CartManager) AddItem(customerPhone string, item CartItem) error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	cart, ok := cm.carts[customerPhone]
	if !ok || cart.IsExpired() {
		return fmt.Errorf("cart not found for customer: %s", customerPhone)
	}

	// Check if item already exists
	for i, existingItem := range cart.Items {
		if existingItem.ProductID == item.ProductID {
			cart.Items[i].Quantity += item.Quantity
			cart.RecalculateSubtotal()
			cart.UpdatedAt = time.Now()
			cart.Abandoned = false
			return nil
		}
	}

	// Add new item
	cart.Items = append(cart.Items, item)
	cart.RecalculateSubtotal()
	cart.UpdatedAt = time.Now()
	cart.Abandoned = false

	return nil
}

// RemoveItem removes an item from the cart
func (cm *CartManager) RemoveItem(customerPhone, productID string) error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	cart, ok := cm.carts[customerPhone]
	if !ok || cart.IsExpired() {
		return fmt.Errorf("cart not found for customer: %s", customerPhone)
	}

	for i, item := range cart.Items {
		if item.ProductID == productID {
			cart.Items = append(cart.Items[:i], cart.Items[i+1:]...)
			cart.RecalculateSubtotal()
			cart.UpdatedAt = time.Now()
			return nil
		}
	}

	return fmt.Errorf("item not found in cart: %s", productID)
}

// UpdateItemQuantity updates the quantity of an item in the cart
func (cm *CartManager) UpdateItemQuantity(customerPhone, productID string, quantity int) error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	cart, ok := cm.carts[customerPhone]
	if !ok || cart.IsExpired() {
		return fmt.Errorf("cart not found for customer: %s", customerPhone)
	}

	if quantity <= 0 {
		// Remove item if quantity is 0 or negative
		for i, item := range cart.Items {
			if item.ProductID == productID {
				cart.Items = append(cart.Items[:i], cart.Items[i+1:]...)
				cart.RecalculateSubtotal()
				cart.UpdatedAt = time.Now()
				return nil
			}
		}
		return fmt.Errorf("item not found in cart: %s", productID)
	}

	for i, item := range cart.Items {
		if item.ProductID == productID {
			cart.Items[i].Quantity = quantity
			cart.RecalculateSubtotal()
			cart.UpdatedAt = time.Now()
			cart.Abandoned = false
			return nil
		}
	}

	return fmt.Errorf("item not found in cart: %s", productID)
}

// ClearCart removes all items from the cart
func (cm *CartManager) ClearCart(customerPhone string) error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	cart, ok := cm.carts[customerPhone]
	if !ok {
		return fmt.Errorf("cart not found for customer: %s", customerPhone)
	}

	cart.Items = make([]CartItem, 0)
	cart.Subtotal = 0
	cart.UpdatedAt = time.Now()
	cart.Abandoned = false

	return nil
}

// DeleteCart deletes a cart completely
func (cm *CartManager) DeleteCart(customerPhone string) {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	delete(cm.carts, customerPhone)
}

// =============================================================================
// Cart Helper Functions
// =============================================================================

// generateCartID generates a unique cart ID
func generateCartID() string {
	return fmt.Sprintf("CART-%d", time.Now().UnixNano())
}

// Note: IsExpired, RecalculateSubtotal, GetItemCount, IsEmpty methods
// are defined in internal/domain/entity/cart.go

// =============================================================================
// Cart Abandonment
// =============================================================================

// CheckAbandonedCarts checks for abandoned carts and moves them
func (cm *CartManager) CheckAbandonedCarts() []*Cart {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	now := time.Now()
	threshold := now.Add(-cm.abandonThreshold)
	abandoned := make([]*Cart, 0)

	for phone, cart := range cm.carts {
		if cart.UpdatedAt.Before(threshold) && !cart.IsEmpty() && !cart.Abandoned {
			cart.Abandoned = true
			cm.abandonedCarts[cart.ID] = cart
			abandoned = append(abandoned, cart)
			delete(cm.carts, phone)
		}
	}

	return abandoned
}

// GetAbandonedCarts returns all abandoned carts
func (cm *CartManager) GetAbandonedCarts() []*Cart {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	carts := make([]*Cart, 0, len(cm.abandonedCarts))
	for _, cart := range cm.abandonedCarts {
		carts = append(carts, cart)
	}
	return carts
}

// RecoverAbandonedCart recovers an abandoned cart for a customer
func (cm *CartManager) RecoverAbandonedCart(cartID, customerPhone string) (*Cart, error) {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	cart, ok := cm.abandonedCarts[cartID]
	if !ok {
		return nil, fmt.Errorf("abandoned cart not found: %s", cartID)
	}

	// Recover the cart
	cart.Abandoned = false
	cart.UpdatedAt = time.Now()
	cart.ExpiresAt = time.Now().Add(cm.cartTTL)
	cart.CustomerPhone = customerPhone

	cm.carts[customerPhone] = cart
	delete(cm.abandonedCarts, cartID)

	return cart, nil
}

// =============================================================================
// Cart Messages
// =============================================================================

// SendCartSummary sends a cart summary to the customer
func (cm *CartManager) SendCartSummary(ctx context.Context, customerPhone string) (string, error) {
	cart, ok := cm.GetCart(customerPhone)
	if !ok {
		return "", fmt.Errorf("cart not found for customer: %s", customerPhone)
	}

	summary := cm.FormatCartSummary(cart)
	return cm.sendTextMessage(ctx, customerPhone, summary)
}

// FormatCartSummary formats a cart as a text summary
func (cm *CartManager) FormatCartSummary(cart *Cart) string {
	if cart.IsEmpty() {
		return "Your cart is empty."
	}

	summary := "*Your Shopping Cart*\n\n"

	for i, item := range cart.Items {
		itemTotal := float64(item.UnitPrice*int64(item.Quantity)) / 100
		summary += fmt.Sprintf("%d. %s\n   Qty: %d x %.2f = %.2f %s\n\n",
			i+1, item.ProductName, item.Quantity,
			float64(item.UnitPrice)/100, itemTotal, item.Currency)
	}

	subtotal := float64(cart.Subtotal) / 100
	summary += fmt.Sprintf("*Subtotal: %.2f %s*\n", subtotal, cart.Currency)
	summary += fmt.Sprintf("\nItems in cart: %d", cart.GetItemCount())

	return summary
}

// SendAbandonedCartReminder sends a reminder for an abandoned cart
func (cm *CartManager) SendAbandonedCartReminder(ctx context.Context, cart *Cart) (string, error) {
	message := fmt.Sprintf("*Don't forget your items!*\n\n"+
		"You have %d item(s) in your cart worth %.2f %s.\n\n"+
		"Complete your purchase now!",
		cart.GetItemCount(),
		float64(cart.Subtotal)/100,
		cart.Currency)

	return cm.sendTextMessage(ctx, cart.CustomerPhone, message)
}

// sendTextMessage sends a text message
func (cm *CartManager) sendTextMessage(ctx context.Context, to, text string) (string, error) {
	apiURL := cm.buildURL(fmt.Sprintf("/%s/messages", cm.phoneNumberID))

	body := map[string]interface{}{
		"messaging_product": "whatsapp",
		"recipient_type":    "individual",
		"to":                to,
		"type":              "text",
		"text": map[string]string{
			"body": text,
		},
	}

	respBody, err := cm.doRequest(ctx, http.MethodPost, apiURL, body)
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
// Cart to Order Conversion
// =============================================================================

// ConvertToOrder converts a cart to an order
func (cm *CartManager) ConvertToOrder(customerPhone string) (*Order, error) {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	cart, ok := cm.carts[customerPhone]
	if !ok || cart.IsExpired() {
		return nil, fmt.Errorf("cart not found for customer: %s", customerPhone)
	}

	if cart.IsEmpty() {
		return nil, fmt.Errorf("cart is empty")
	}

	// Create order from cart
	order := &Order{
		ID:            generateOrderID(),
		CatalogID:     cart.CatalogID,
		CustomerPhone: cart.CustomerPhone,
		Status:        OrderStatusPending,
		Items:         make([]OrderItem, len(cart.Items)),
		Subtotal:      cart.Subtotal,
		Total:         cart.Subtotal,
		Currency:      cart.Currency,
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}

	for i, cartItem := range cart.Items {
		order.Items[i] = OrderItem{
			ProductID:   cartItem.ProductID,
			ProductName: cartItem.ProductName,
			Quantity:    cartItem.Quantity,
			UnitPrice:   cartItem.UnitPrice,
			Currency:    cartItem.Currency,
			TotalPrice:  cartItem.UnitPrice * int64(cartItem.Quantity),
		}
	}

	// Clear the cart after conversion
	cart.Items = make([]CartItem, 0)
	cart.Subtotal = 0
	cart.UpdatedAt = time.Now()

	return order, nil
}

// =============================================================================
// Cart Statistics
// =============================================================================

// CartStats represents cart statistics
type CartStats struct {
	ActiveCarts    int   `json:"active_carts"`
	AbandonedCarts int   `json:"abandoned_carts"`
	TotalItems     int   `json:"total_items"`
	TotalValue     int64 `json:"total_value"`
	Currency       string `json:"currency"`
}

// GetCartStats returns cart statistics
func (cm *CartManager) GetCartStats() *CartStats {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	stats := &CartStats{}

	for _, cart := range cm.carts {
		if !cart.IsExpired() && !cart.IsEmpty() {
			stats.ActiveCarts++
			stats.TotalItems += cart.GetItemCount()
			stats.TotalValue += cart.Subtotal
			if stats.Currency == "" {
				stats.Currency = cart.Currency
			}
		}
	}

	stats.AbandonedCarts = len(cm.abandonedCarts)

	return stats
}

// =============================================================================
// Cart Cleanup
// =============================================================================

// CleanupExpiredCarts removes expired carts
func (cm *CartManager) CleanupExpiredCarts() int {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	count := 0
	for phone, cart := range cm.carts {
		if cart.IsExpired() {
			delete(cm.carts, phone)
			count++
		}
	}

	// Also cleanup old abandoned carts (older than 7 days)
	threshold := time.Now().Add(-7 * 24 * time.Hour)
	for id, cart := range cm.abandonedCarts {
		if cart.UpdatedAt.Before(threshold) {
			delete(cm.abandonedCarts, id)
			count++
		}
	}

	return count
}

// StartCleanupRoutine starts a background routine to cleanup expired carts
func (cm *CartManager) StartCleanupRoutine(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	go func() {
		for {
			select {
			case <-ctx.Done():
				ticker.Stop()
				return
			case <-ticker.C:
				cm.CleanupExpiredCarts()
				cm.CheckAbandonedCarts()
			}
		}
	}()
}
