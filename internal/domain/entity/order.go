package entity

import (
	"time"
)

// OrderStatus represents the status of an order
type OrderStatus string

const (
	OrderStatusPending    OrderStatus = "pending"
	OrderStatusConfirmed  OrderStatus = "confirmed"
	OrderStatusProcessing OrderStatus = "processing"
	OrderStatusShipped    OrderStatus = "shipped"
	OrderStatusDelivered  OrderStatus = "delivered"
	OrderStatusCompleted  OrderStatus = "completed"
	OrderStatusCancelled  OrderStatus = "cancelled"
	OrderStatusRefunded   OrderStatus = "refunded"
)

// Order represents a commerce order in the domain
type Order struct {
	ID              string      `json:"id" db:"id"`
	OrganizationID  string      `json:"organization_id" db:"organization_id"`
	ChannelID       string      `json:"channel_id" db:"channel_id"`
	ConversationID  string      `json:"conversation_id,omitempty" db:"conversation_id"`
	CatalogID       string      `json:"catalog_id" db:"catalog_id"`
	CustomerPhone   string      `json:"customer_phone" db:"customer_phone"`
	CustomerName    string      `json:"customer_name,omitempty" db:"customer_name"`
	Status          OrderStatus `json:"status" db:"status"`
	Items           []OrderItem `json:"items" db:"-"`
	Subtotal        int64       `json:"subtotal" db:"subtotal"`
	Tax             int64       `json:"tax" db:"tax"`
	Shipping        int64       `json:"shipping" db:"shipping"`
	Discount        int64       `json:"discount" db:"discount"`
	Total           int64       `json:"total" db:"total"`
	Currency        string      `json:"currency" db:"currency"`
	ShippingAddress *Address    `json:"shipping_address,omitempty" db:"-"`
	BillingAddress  *Address    `json:"billing_address,omitempty" db:"-"`
	Notes           string      `json:"notes,omitempty" db:"notes"`
	MessageID       string      `json:"message_id,omitempty" db:"message_id"`
	TrackingNumber  string      `json:"tracking_number,omitempty" db:"tracking_number"`
	TrackingURL     string      `json:"tracking_url,omitempty" db:"tracking_url"`
	CreatedAt       time.Time   `json:"created_at" db:"created_at"`
	UpdatedAt       time.Time   `json:"updated_at" db:"updated_at"`
	ConfirmedAt     *time.Time  `json:"confirmed_at,omitempty" db:"confirmed_at"`
	ShippedAt       *time.Time  `json:"shipped_at,omitempty" db:"shipped_at"`
	DeliveredAt     *time.Time  `json:"delivered_at,omitempty" db:"delivered_at"`
	CancelledAt     *time.Time  `json:"cancelled_at,omitempty" db:"cancelled_at"`
}

// OrderItem represents an item in an order
type OrderItem struct {
	ID          string `json:"id" db:"id"`
	OrderID     string `json:"order_id" db:"order_id"`
	ProductID   string `json:"product_id" db:"product_id"`
	ProductName string `json:"product_name" db:"product_name"`
	ProductSKU  string `json:"product_sku,omitempty" db:"product_sku"`
	Quantity    int    `json:"quantity" db:"quantity"`
	UnitPrice   int64  `json:"unit_price" db:"unit_price"`
	TotalPrice  int64  `json:"total_price" db:"total_price"`
	Currency    string `json:"currency" db:"currency"`
	ImageURL    string `json:"image_url,omitempty" db:"image_url"`
}

// Address represents a shipping or billing address
type Address struct {
	ID          string `json:"id,omitempty" db:"id"`
	Name        string `json:"name,omitempty" db:"name"`
	PhoneNumber string `json:"phone_number,omitempty" db:"phone_number"`
	Street      string `json:"street,omitempty" db:"street"`
	Number      string `json:"number,omitempty" db:"number"`
	Complement  string `json:"complement,omitempty" db:"complement"`
	Neighborhood string `json:"neighborhood,omitempty" db:"neighborhood"`
	City        string `json:"city,omitempty" db:"city"`
	State       string `json:"state,omitempty" db:"state"`
	ZipCode     string `json:"zip_code,omitempty" db:"zip_code"`
	Country     string `json:"country,omitempty" db:"country"`
	CountryCode string `json:"country_code,omitempty" db:"country_code"`
}

// OrderStatusHistory represents a status change in order history
type OrderStatusHistory struct {
	ID        string      `json:"id" db:"id"`
	OrderID   string      `json:"order_id" db:"order_id"`
	Status    OrderStatus `json:"status" db:"status"`
	Notes     string      `json:"notes,omitempty" db:"notes"`
	CreatedBy string      `json:"created_by,omitempty" db:"created_by"`
	CreatedAt time.Time   `json:"created_at" db:"created_at"`
}

// NewOrder creates a new order
func NewOrder(orgID, channelID, catalogID, customerPhone string) *Order {
	now := time.Now()
	return &Order{
		OrganizationID: orgID,
		ChannelID:      channelID,
		CatalogID:      catalogID,
		CustomerPhone:  customerPhone,
		Status:         OrderStatusPending,
		Items:          make([]OrderItem, 0),
		Currency:       "BRL",
		CreatedAt:      now,
		UpdatedAt:      now,
	}
}

// AddItem adds an item to the order
func (o *Order) AddItem(item OrderItem) {
	item.OrderID = o.ID
	item.TotalPrice = item.UnitPrice * int64(item.Quantity)
	o.Items = append(o.Items, item)
	o.recalculateTotals()
}

// RemoveItem removes an item from the order
func (o *Order) RemoveItem(productID string) {
	for i, item := range o.Items {
		if item.ProductID == productID {
			o.Items = append(o.Items[:i], o.Items[i+1:]...)
			o.recalculateTotals()
			return
		}
	}
}

// recalculateTotals recalculates order totals
func (o *Order) recalculateTotals() {
	var subtotal int64
	for _, item := range o.Items {
		subtotal += item.TotalPrice
	}
	o.Subtotal = subtotal
	o.Total = subtotal + o.Tax + o.Shipping - o.Discount
	o.UpdatedAt = time.Now()
}

// SetTaxAndShipping sets tax and shipping values
func (o *Order) SetTaxAndShipping(tax, shipping int64) {
	o.Tax = tax
	o.Shipping = shipping
	o.recalculateTotals()
}

// ApplyDiscount applies a discount to the order
func (o *Order) ApplyDiscount(discount int64) {
	o.Discount = discount
	o.recalculateTotals()
}

// UpdateStatus updates the order status
func (o *Order) UpdateStatus(status OrderStatus) {
	o.Status = status
	o.UpdatedAt = time.Now()

	now := time.Now()
	switch status {
	case OrderStatusConfirmed:
		o.ConfirmedAt = &now
	case OrderStatusShipped:
		o.ShippedAt = &now
	case OrderStatusDelivered:
		o.DeliveredAt = &now
	case OrderStatusCancelled:
		o.CancelledAt = &now
	}
}

// CanCancel returns true if the order can be cancelled
func (o *Order) CanCancel() bool {
	return o.Status == OrderStatusPending ||
		o.Status == OrderStatusConfirmed ||
		o.Status == OrderStatusProcessing
}

// CanRefund returns true if the order can be refunded
func (o *Order) CanRefund() bool {
	return o.Status == OrderStatusCompleted ||
		o.Status == OrderStatusDelivered
}

// GetItemCount returns the total number of items
func (o *Order) GetItemCount() int {
	count := 0
	for _, item := range o.Items {
		count += item.Quantity
	}
	return count
}

// IsEmpty returns true if the order has no items
func (o *Order) IsEmpty() bool {
	return len(o.Items) == 0
}
