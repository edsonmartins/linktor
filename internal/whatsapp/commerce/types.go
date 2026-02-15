package commerce

import (
	"time"

	"github.com/msgfy/linktor/internal/domain/entity"
)

// Re-export domain types for convenience
type (
	Order       = entity.Order
	OrderItem   = entity.OrderItem
	OrderStatus = entity.OrderStatus
	Cart        = entity.Cart
	CartItem    = entity.CartItem
	Address     = entity.Address
)

// Re-export domain constants
const (
	OrderStatusPending    = entity.OrderStatusPending
	OrderStatusConfirmed  = entity.OrderStatusConfirmed
	OrderStatusProcessing = entity.OrderStatusProcessing
	OrderStatusShipped    = entity.OrderStatusShipped
	OrderStatusDelivered  = entity.OrderStatusDelivered
	OrderStatusCompleted  = entity.OrderStatusCompleted
	OrderStatusCancelled  = entity.OrderStatusCancelled
	OrderStatusRefunded   = entity.OrderStatusRefunded
)

// =============================================================================
// Product Types (specific to WhatsApp Commerce API)
// =============================================================================

// Product represents a product in the Facebook Commerce catalog
type Product struct {
	ID               string            `json:"id"`
	RetailerID       string            `json:"retailer_id"`
	Name             string            `json:"name"`
	Description      string            `json:"description,omitempty"`
	Price            int64             `json:"price"` // Price in cents
	Currency         string            `json:"currency"`
	ImageURL         string            `json:"image_url,omitempty"`
	URL              string            `json:"url,omitempty"`
	Availability     string            `json:"availability,omitempty"` // in stock, out of stock
	Condition        string            `json:"condition,omitempty"`    // new, refurbished, used
	Brand            string            `json:"brand,omitempty"`
	Category         string            `json:"category,omitempty"`
	CustomLabels     map[string]string `json:"custom_labels,omitempty"`
	AdditionalImages []string          `json:"additional_image_urls,omitempty"`
}

// ProductVariant represents a variant of a product
type ProductVariant struct {
	ID         string            `json:"id"`
	ProductID  string            `json:"product_id"`
	RetailerID string            `json:"retailer_id"`
	Name       string            `json:"name"`
	Price      int64             `json:"price"`
	Currency   string            `json:"currency"`
	ImageURL   string            `json:"image_url,omitempty"`
	Options    map[string]string `json:"options,omitempty"` // e.g., {"size": "M", "color": "blue"}
}

// ProductSection represents a section of products for multi-product messages
type ProductSection struct {
	Title      string   `json:"title"`
	ProductIDs []string `json:"product_retailer_ids"`
}

// =============================================================================
// Catalog Types
// =============================================================================

// Catalog represents a WhatsApp Commerce catalog
type Catalog struct {
	ID           string    `json:"id"`
	Name         string    `json:"name"`
	BusinessID   string    `json:"business_id"`
	ProductCount int       `json:"product_count"`
	IsPublished  bool      `json:"is_published"`
	VerticalType string    `json:"vertical_type,omitempty"` // commerce, hotels, flights, etc.
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// CatalogSyncStatus represents the sync status of a catalog
type CatalogSyncStatus struct {
	CatalogID      string    `json:"catalog_id"`
	LastSyncAt     time.Time `json:"last_sync_at"`
	Status         string    `json:"status"` // synced, syncing, error
	ProductsSynced int       `json:"products_synced"`
	ErrorMessage   string    `json:"error_message,omitempty"`
}

// =============================================================================
// Commerce Message Types
// =============================================================================

// OrderConfirmation represents an order confirmation message
type OrderConfirmation struct {
	OrderID           string `json:"order_id"`
	Status            string `json:"status"`
	Description       string `json:"description,omitempty"`
	EstimatedDelivery string `json:"estimated_delivery,omitempty"`
	TrackingNumber    string `json:"tracking_number,omitempty"`
	TrackingURL       string `json:"tracking_url,omitempty"`
}

// CartUpdate represents a cart update operation
type CartUpdate struct {
	Action    string `json:"action"` // add, remove, update
	ProductID string `json:"product_retailer_id"`
	Quantity  int    `json:"quantity"`
}

// =============================================================================
// Message Types for Commerce
// =============================================================================

// SingleProductMessage represents a single product message
type SingleProductMessage struct {
	To          string `json:"to"`
	CatalogID   string `json:"catalog_id"`
	ProductID   string `json:"product_retailer_id"`
	BodyText    string `json:"body_text,omitempty"`
	FooterText  string `json:"footer_text,omitempty"`
}

// MultiProductMessage represents a multi-product message (up to 30 products)
type MultiProductMessage struct {
	To          string           `json:"to"`
	CatalogID   string           `json:"catalog_id"`
	HeaderText  string           `json:"header_text"`
	BodyText    string           `json:"body_text"`
	FooterText  string           `json:"footer_text,omitempty"`
	Sections    []ProductSection `json:"sections"`
}

// CatalogMessage represents a catalog message
type CatalogMessage struct {
	To          string `json:"to"`
	BodyText    string `json:"body_text"`
	FooterText  string `json:"footer_text,omitempty"`
	ThumbnailProductID string `json:"thumbnail_product_retailer_id,omitempty"`
}

// OrderDetailsMessage represents an order details/status message
type OrderDetailsMessage struct {
	To              string `json:"to"`
	ReferenceID     string `json:"reference_id"` // Order ID
	Status          string `json:"status"`
	PaymentStatus   string `json:"payment_status,omitempty"`
	OrderType       string `json:"order_type,omitempty"` // food_order, physical_goods, digital_goods
	Description     string `json:"description,omitempty"`
}

// =============================================================================
// Webhook Types
// =============================================================================

// OrderWebhookPayload represents an order received via webhook
type OrderWebhookPayload struct {
	CatalogID string      `json:"catalog_id"`
	Text      string      `json:"text,omitempty"`
	Order     OrderData   `json:"order"`
}

// OrderData represents order data from webhook
type OrderData struct {
	ProductItems []OrderProductItem `json:"product_items"`
	Text         string             `json:"text,omitempty"`
}

// OrderProductItem represents a product item in webhook order data
type OrderProductItem struct {
	ProductRetailerID string `json:"product_retailer_id"`
	Quantity          int    `json:"quantity"`
	ItemPrice         string `json:"item_price"` // Formatted price string
	Currency          string `json:"currency"`
}

// =============================================================================
// API Response Types
// =============================================================================

// ProductListResponse represents a paginated list of products
type ProductListResponse struct {
	Products []Product `json:"data"`
	Paging   *Paging   `json:"paging,omitempty"`
}

// Paging represents pagination info
type Paging struct {
	Cursors struct {
		Before string `json:"before"`
		After  string `json:"after"`
	} `json:"cursors"`
	Next     string `json:"next,omitempty"`
	Previous string `json:"previous,omitempty"`
}

// CatalogListResponse represents a list of catalogs
type CatalogListResponse struct {
	Catalogs []Catalog `json:"data"`
	Paging   *Paging   `json:"paging,omitempty"`
}
