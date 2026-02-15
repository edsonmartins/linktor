package entity

import (
	"time"
)

// Cart represents a shopping cart in the domain
type Cart struct {
	ID             string     `json:"id" db:"id"`
	OrganizationID string     `json:"organization_id" db:"organization_id"`
	ChannelID      string     `json:"channel_id" db:"channel_id"`
	CustomerPhone  string     `json:"customer_phone" db:"customer_phone"`
	CatalogID      string     `json:"catalog_id" db:"catalog_id"`
	Items          []CartItem `json:"items" db:"-"`
	Subtotal       int64      `json:"subtotal" db:"subtotal"`
	Currency       string     `json:"currency" db:"currency"`
	CreatedAt      time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at" db:"updated_at"`
	ExpiresAt      time.Time  `json:"expires_at" db:"expires_at"`
	Abandoned      bool       `json:"abandoned" db:"abandoned"`
	AbandonedAt    *time.Time `json:"abandoned_at,omitempty" db:"abandoned_at"`
	RecoveredAt    *time.Time `json:"recovered_at,omitempty" db:"recovered_at"`
}

// CartItem represents an item in a cart
type CartItem struct {
	ID          string `json:"id" db:"id"`
	CartID      string `json:"cart_id" db:"cart_id"`
	ProductID   string `json:"product_id" db:"product_id"`
	ProductName string `json:"product_name" db:"product_name"`
	ProductSKU  string `json:"product_sku,omitempty" db:"product_sku"`
	Quantity    int    `json:"quantity" db:"quantity"`
	UnitPrice   int64  `json:"unit_price" db:"unit_price"`
	Currency    string `json:"currency" db:"currency"`
	ImageURL    string `json:"image_url,omitempty" db:"image_url"`
	AddedAt     time.Time `json:"added_at" db:"added_at"`
}

// NewCart creates a new cart
func NewCart(orgID, channelID, customerPhone, catalogID string, ttl time.Duration) *Cart {
	now := time.Now()
	return &Cart{
		OrganizationID: orgID,
		ChannelID:      channelID,
		CustomerPhone:  customerPhone,
		CatalogID:      catalogID,
		Items:          make([]CartItem, 0),
		Currency:       "BRL",
		CreatedAt:      now,
		UpdatedAt:      now,
		ExpiresAt:      now.Add(ttl),
	}
}

// AddItem adds an item to the cart
func (c *Cart) AddItem(item CartItem) {
	// Check if item already exists
	for i, existingItem := range c.Items {
		if existingItem.ProductID == item.ProductID {
			c.Items[i].Quantity += item.Quantity
			c.RecalculateSubtotal()
			c.touch()
			return
		}
	}

	// Add new item
	item.CartID = c.ID
	item.AddedAt = time.Now()
	c.Items = append(c.Items, item)
	c.RecalculateSubtotal()
	c.touch()
}

// RemoveItem removes an item from the cart
func (c *Cart) RemoveItem(productID string) bool {
	for i, item := range c.Items {
		if item.ProductID == productID {
			c.Items = append(c.Items[:i], c.Items[i+1:]...)
			c.RecalculateSubtotal()
			c.touch()
			return true
		}
	}
	return false
}

// UpdateItemQuantity updates the quantity of an item
func (c *Cart) UpdateItemQuantity(productID string, quantity int) bool {
	if quantity <= 0 {
		return c.RemoveItem(productID)
	}

	for i, item := range c.Items {
		if item.ProductID == productID {
			c.Items[i].Quantity = quantity
			c.RecalculateSubtotal()
			c.touch()
			return true
		}
	}
	return false
}

// Clear removes all items from the cart
func (c *Cart) Clear() {
	c.Items = make([]CartItem, 0)
	c.Subtotal = 0
	c.touch()
}

// RecalculateSubtotal recalculates the cart subtotal
func (c *Cart) RecalculateSubtotal() {
	var subtotal int64
	for _, item := range c.Items {
		subtotal += item.UnitPrice * int64(item.Quantity)
	}
	c.Subtotal = subtotal
}

// touch updates the UpdatedAt timestamp and resets abandoned state
func (c *Cart) touch() {
	c.UpdatedAt = time.Now()
	if c.Abandoned {
		c.Abandoned = false
		now := time.Now()
		c.RecoveredAt = &now
	}
}

// MarkAbandoned marks the cart as abandoned
func (c *Cart) MarkAbandoned() {
	c.Abandoned = true
	now := time.Now()
	c.AbandonedAt = &now
}

// IsExpired returns true if the cart has expired
func (c *Cart) IsExpired() bool {
	return time.Now().After(c.ExpiresAt)
}

// IsEmpty returns true if the cart has no items
func (c *Cart) IsEmpty() bool {
	return len(c.Items) == 0
}

// GetItemCount returns the total number of items
func (c *Cart) GetItemCount() int {
	count := 0
	for _, item := range c.Items {
		count += item.Quantity
	}
	return count
}

// GetItem returns an item by product ID
func (c *Cart) GetItem(productID string) (*CartItem, bool) {
	for _, item := range c.Items {
		if item.ProductID == productID {
			return &item, true
		}
	}
	return nil, false
}

// Extend extends the cart expiration
func (c *Cart) Extend(ttl time.Duration) {
	c.ExpiresAt = time.Now().Add(ttl)
	c.touch()
}

// ToOrder converts the cart to an order
func (c *Cart) ToOrder() *Order {
	order := &Order{
		OrganizationID: c.OrganizationID,
		ChannelID:      c.ChannelID,
		CatalogID:      c.CatalogID,
		CustomerPhone:  c.CustomerPhone,
		Status:         OrderStatusPending,
		Items:          make([]OrderItem, len(c.Items)),
		Subtotal:       c.Subtotal,
		Total:          c.Subtotal,
		Currency:       c.Currency,
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}

	for i, cartItem := range c.Items {
		order.Items[i] = OrderItem{
			ProductID:   cartItem.ProductID,
			ProductName: cartItem.ProductName,
			ProductSKU:  cartItem.ProductSKU,
			Quantity:    cartItem.Quantity,
			UnitPrice:   cartItem.UnitPrice,
			TotalPrice:  cartItem.UnitPrice * int64(cartItem.Quantity),
			Currency:    cartItem.Currency,
			ImageURL:    cartItem.ImageURL,
		}
	}

	return order
}

// CartSummary represents a cart summary
type CartSummary struct {
	CartID        string `json:"cart_id"`
	CustomerPhone string `json:"customer_phone"`
	ItemCount     int    `json:"item_count"`
	TotalItems    int    `json:"total_items"`
	Subtotal      int64  `json:"subtotal"`
	Currency      string `json:"currency"`
	Abandoned     bool   `json:"abandoned"`
	ExpiresAt     time.Time `json:"expires_at"`
}

// GetSummary returns a summary of the cart
func (c *Cart) GetSummary() *CartSummary {
	return &CartSummary{
		CartID:        c.ID,
		CustomerPhone: c.CustomerPhone,
		ItemCount:     len(c.Items),
		TotalItems:    c.GetItemCount(),
		Subtotal:      c.Subtotal,
		Currency:      c.Currency,
		Abandoned:     c.Abandoned,
		ExpiresAt:     c.ExpiresAt,
	}
}
