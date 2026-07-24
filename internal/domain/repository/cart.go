package repository

import (
	"context"
	"time"

	"github.com/msgfy/linktor/internal/domain/entity"
)

// CartRepository defines the interface for cart persistence
type CartRepository interface {
	// Create creates a new cart
	Create(ctx context.Context, cart *entity.Cart) error

	// GetByID retrieves a cart by ID, scoped to the organization (INV-001)
	GetByID(ctx context.Context, orgID, cartID string) (*entity.Cart, error)

	// GetByCustomer retrieves an active cart for a customer
	GetByCustomer(ctx context.Context, orgID, customerPhone string) (*entity.Cart, error)

	// Update updates an existing cart
	Update(ctx context.Context, cart *entity.Cart) error

	// Delete deletes a cart, scoped to the organization (INV-001)
	Delete(ctx context.Context, orgID, cartID string) error

	// GetCartItems retrieves items for a cart
	GetCartItems(ctx context.Context, cartID string) ([]entity.CartItem, error)

	// AddCartItem adds an item to a cart
	AddCartItem(ctx context.Context, item *entity.CartItem) error

	// UpdateCartItem updates a cart item
	UpdateCartItem(ctx context.Context, item *entity.CartItem) error

	// DeleteCartItem deletes a cart item, scoped to the organization (INV-001)
	DeleteCartItem(ctx context.Context, orgID, cartID, itemID string) error

	// ClearCart removes all items from a cart
	ClearCart(ctx context.Context, cartID string) error

	// GetAbandonedCarts retrieves abandoned carts
	GetAbandonedCarts(ctx context.Context, orgID string, threshold time.Duration, pagination Pagination) ([]*entity.Cart, int, error)

	// MarkAsAbandoned marks a cart as abandoned, scoped to the org (INV-001)
	MarkAsAbandoned(ctx context.Context, orgID, cartID string) error

	// RecoverCart recovers an abandoned cart, scoped to the org (INV-001)
	RecoverCart(ctx context.Context, orgID, cartID string) error

	// GetExpiredCarts retrieves expired carts for cleanup
	GetExpiredCarts(ctx context.Context, limit int) ([]*entity.Cart, error)

	// DeleteExpiredCarts deletes expired carts
	DeleteExpiredCarts(ctx context.Context) (int, error)

	// GetStats retrieves cart statistics
	GetStats(ctx context.Context, orgID string) (*CartStats, error)
}

// CartStats represents cart statistics
type CartStats struct {
	ActiveCarts      int     `json:"active_carts"`
	AbandonedCarts   int     `json:"abandoned_carts"`
	TotalItems       int     `json:"total_items"`
	TotalValue       int64   `json:"total_value"`
	AverageCartValue int64   `json:"average_cart_value"`
	Currency         string  `json:"currency"`
	AbandonmentRate  float64 `json:"abandonment_rate"`
}
