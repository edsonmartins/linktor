package repository

import (
	"context"
	"time"

	"github.com/msgfy/linktor/internal/domain/entity"
)

// OrderRepository defines the interface for order persistence
type OrderRepository interface {
	// Create creates a new order
	Create(ctx context.Context, order *entity.Order) error

	// GetByID retrieves an order by ID
	GetByID(ctx context.Context, orgID, orderID string) (*entity.Order, error)

	// GetByMessageID retrieves an order by WhatsApp message ID
	GetByMessageID(ctx context.Context, orgID, messageID string) (*entity.Order, error)

	// Update updates an existing order
	Update(ctx context.Context, order *entity.Order) error

	// UpdateStatus updates the status of an order
	UpdateStatus(ctx context.Context, orgID, orderID string, status entity.OrderStatus) error

	// Delete deletes an order
	Delete(ctx context.Context, orgID, orderID string) error

	// List lists orders with filters and pagination
	List(ctx context.Context, orgID string, filters OrderFilters, pagination Pagination) ([]*entity.Order, int, error)

	// GetByCustomer retrieves orders for a specific customer
	GetByCustomer(ctx context.Context, orgID, customerPhone string, pagination Pagination) ([]*entity.Order, int, error)

	// GetByChannel retrieves orders for a specific channel
	GetByChannel(ctx context.Context, orgID, channelID string, pagination Pagination) ([]*entity.Order, int, error)

	// GetOrderItems retrieves items for an order
	GetOrderItems(ctx context.Context, orderID string) ([]entity.OrderItem, error)

	// AddOrderItem adds an item to an order
	AddOrderItem(ctx context.Context, item *entity.OrderItem) error

	// UpdateOrderItem updates an order item
	UpdateOrderItem(ctx context.Context, item *entity.OrderItem) error

	// DeleteOrderItem deletes an order item
	DeleteOrderItem(ctx context.Context, orderID, itemID string) error

	// GetStatusHistory retrieves status history for an order
	GetStatusHistory(ctx context.Context, orderID string) ([]entity.OrderStatusHistory, error)

	// AddStatusHistory adds a status history entry
	AddStatusHistory(ctx context.Context, history *entity.OrderStatusHistory) error

	// GetStats retrieves order statistics
	GetStats(ctx context.Context, orgID string, filters StatsFilters) (*OrderStats, error)
}

// OrderFilters represents filters for listing orders
type OrderFilters struct {
	Status        *entity.OrderStatus
	CustomerPhone string
	ChannelID     string
	CatalogID     string
	FromDate      *time.Time
	ToDate        *time.Time
	MinTotal      *int64
	MaxTotal      *int64
	Search        string
}

// StatsFilters represents filters for order statistics
type StatsFilters struct {
	ChannelID string
	FromDate  *time.Time
	ToDate    *time.Time
}

// OrderStats represents order statistics
type OrderStats struct {
	TotalOrders     int                         `json:"total_orders"`
	TotalRevenue    int64                       `json:"total_revenue"`
	AverageOrderValue int64                     `json:"average_order_value"`
	Currency        string                      `json:"currency"`
	ByStatus        map[entity.OrderStatus]int  `json:"by_status"`
	ByChannel       map[string]int              `json:"by_channel"`
	TopProducts     []ProductStat               `json:"top_products"`
	DailyOrders     []DailyStat                 `json:"daily_orders"`
}

// ProductStat represents statistics for a product
type ProductStat struct {
	ProductID   string `json:"product_id"`
	ProductName string `json:"product_name"`
	Quantity    int    `json:"quantity"`
	Revenue     int64  `json:"revenue"`
}

// DailyStat represents daily statistics
type DailyStat struct {
	Date     string `json:"date"`
	Orders   int    `json:"orders"`
	Revenue  int64  `json:"revenue"`
}

// Pagination represents pagination parameters
type Pagination struct {
	Page     int `json:"page"`
	PageSize int `json:"page_size"`
}

// GetOffset returns the offset for SQL queries
func (p Pagination) GetOffset() int {
	if p.Page < 1 {
		p.Page = 1
	}
	return (p.Page - 1) * p.PageSize
}

// GetLimit returns the limit for SQL queries
func (p Pagination) GetLimit() int {
	if p.PageSize < 1 {
		return 20
	}
	if p.PageSize > 100 {
		return 100
	}
	return p.PageSize
}
