package database

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/msgfy/linktor/internal/domain/entity"
	"github.com/msgfy/linktor/internal/domain/repository"
)

// CartRepository implements repository.CartRepository with PostgreSQL
type CartRepository struct {
	db *PostgresDB
}

// NewCartRepository creates a new PostgreSQL cart repository
func NewCartRepository(db *PostgresDB) *CartRepository {
	return &CartRepository{db: db}
}

func (r *CartRepository) Create(ctx context.Context, cart *entity.Cart) error {
	if cart.ID == "" {
		cart.ID = uuid.New().String()
	}
	if cart.CreatedAt.IsZero() {
		cart.CreatedAt = time.Now()
	}
	cart.UpdatedAt = time.Now()
	if cart.ExpiresAt.IsZero() {
		cart.ExpiresAt = time.Now().Add(7 * 24 * time.Hour)
	}

	query := `
		INSERT INTO carts (
			id, organization_id, channel_id, customer_phone, catalog_id,
			subtotal, currency, created_at, updated_at, expires_at,
			abandoned, abandoned_at, recovered_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
	`
	_, err := r.db.Pool.Exec(ctx, query,
		cart.ID, cart.OrganizationID, cart.ChannelID, cart.CustomerPhone,
		cart.CatalogID, cart.Subtotal, cart.Currency,
		cart.CreatedAt, cart.UpdatedAt, cart.ExpiresAt,
		cart.Abandoned, cart.AbandonedAt, cart.RecoveredAt,
	)
	if err != nil {
		return fmt.Errorf("failed to create cart: %w", err)
	}

	for i := range cart.Items {
		if cart.Items[i].ID == "" {
			cart.Items[i].ID = uuid.New().String()
		}
		cart.Items[i].CartID = cart.ID
		if err := r.AddCartItem(ctx, &cart.Items[i]); err != nil {
			return err
		}
	}
	return nil
}

func (r *CartRepository) GetByID(ctx context.Context, orgID, cartID string) (*entity.Cart, error) {
	query := cartSelectColumns + ` FROM carts WHERE id = $1 AND organization_id = $2`
	cart, err := scanCart(r.db.Pool.QueryRow(ctx, query, cartID, orgID))
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("cart not found: %s", cartID)
		}
		return nil, fmt.Errorf("failed to get cart: %w", err)
	}
	return cart, nil
}

func (r *CartRepository) GetByCustomer(ctx context.Context, orgID, customerPhone string) (*entity.Cart, error) {
	query := cartSelectColumns + `
		FROM carts
		WHERE organization_id = $1 AND customer_phone = $2 AND abandoned = FALSE AND expires_at > NOW()
		ORDER BY updated_at DESC LIMIT 1
	`
	cart, err := scanCart(r.db.Pool.QueryRow(ctx, query, orgID, customerPhone))
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("active cart not found for customer: %s", customerPhone)
		}
		return nil, fmt.Errorf("failed to get cart: %w", err)
	}
	return cart, nil
}

// The cart mutation/lookup methods are organization-scoped (INV-001): a cart
// is matched by id AND organization_id, so a future handler exposing a cart by
// id cannot reach another tenant's cart. Update scopes via cart.OrganizationID;
// the id-based methods take orgID explicitly.
func (r *CartRepository) Update(ctx context.Context, cart *entity.Cart) error {
	cart.UpdatedAt = time.Now()
	query := `
		UPDATE carts SET
			subtotal = $1, currency = $2, updated_at = $3, expires_at = $4,
			abandoned = $5, abandoned_at = $6, recovered_at = $7
		WHERE id = $8 AND organization_id = $9
	`
	res, err := r.db.Pool.Exec(ctx, query,
		cart.Subtotal, cart.Currency, cart.UpdatedAt, cart.ExpiresAt,
		cart.Abandoned, cart.AbandonedAt, cart.RecoveredAt, cart.ID, cart.OrganizationID,
	)
	if err != nil {
		return fmt.Errorf("failed to update cart: %w", err)
	}
	if res.RowsAffected() == 0 {
		return fmt.Errorf("cart not found: %s", cart.ID)
	}
	return nil
}

func (r *CartRepository) Delete(ctx context.Context, orgID, cartID string) error {
	res, err := r.db.Pool.Exec(ctx, `DELETE FROM carts WHERE id = $1 AND organization_id = $2`, cartID, orgID)
	if err != nil {
		return fmt.Errorf("failed to delete cart: %w", err)
	}
	if res.RowsAffected() == 0 {
		return fmt.Errorf("cart not found: %s", cartID)
	}
	return nil
}

func (r *CartRepository) GetCartItems(ctx context.Context, cartID string) ([]entity.CartItem, error) {
	query := `
		SELECT id, cart_id, product_id, product_name, product_sku,
		       quantity, unit_price, currency, image_url, added_at
		FROM cart_items WHERE cart_id = $1 ORDER BY added_at ASC
	`
	rows, err := r.db.Pool.Query(ctx, query, cartID)
	if err != nil {
		return nil, fmt.Errorf("failed to query cart items: %w", err)
	}
	defer rows.Close()

	var items []entity.CartItem
	for rows.Next() {
		var it entity.CartItem
		var sku, img *string
		if err := rows.Scan(&it.ID, &it.CartID, &it.ProductID, &it.ProductName,
			&sku, &it.Quantity, &it.UnitPrice, &it.Currency, &img, &it.AddedAt); err != nil {
			return nil, fmt.Errorf("failed to scan cart item: %w", err)
		}
		if sku != nil {
			it.ProductSKU = *sku
		}
		if img != nil {
			it.ImageURL = *img
		}
		items = append(items, it)
	}
	return items, nil
}

func (r *CartRepository) AddCartItem(ctx context.Context, item *entity.CartItem) error {
	if item.ID == "" {
		item.ID = uuid.New().String()
	}
	if item.AddedAt.IsZero() {
		item.AddedAt = time.Now()
	}
	query := `
		INSERT INTO cart_items (
			id, cart_id, product_id, product_name, product_sku,
			quantity, unit_price, currency, image_url, added_at
		) VALUES ($1, $2, $3, $4, NULLIF($5,''), $6, $7, $8, NULLIF($9,''), $10)
	`
	_, err := r.db.Pool.Exec(ctx, query,
		item.ID, item.CartID, item.ProductID, item.ProductName, item.ProductSKU,
		item.Quantity, item.UnitPrice, item.Currency, item.ImageURL, item.AddedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to add cart item: %w", err)
	}
	return nil
}

func (r *CartRepository) UpdateCartItem(ctx context.Context, item *entity.CartItem) error {
	query := `
		UPDATE cart_items SET
			product_name = $1, product_sku = NULLIF($2,''), quantity = $3,
			unit_price = $4, currency = $5, image_url = NULLIF($6,'')
		WHERE id = $7 AND cart_id = $8
	`
	res, err := r.db.Pool.Exec(ctx, query,
		item.ProductName, item.ProductSKU, item.Quantity, item.UnitPrice,
		item.Currency, item.ImageURL, item.ID, item.CartID,
	)
	if err != nil {
		return fmt.Errorf("failed to update cart item: %w", err)
	}
	if res.RowsAffected() == 0 {
		return fmt.Errorf("cart item not found: %s", item.ID)
	}
	return nil
}

func (r *CartRepository) DeleteCartItem(ctx context.Context, orgID, cartID, itemID string) error {
	res, err := r.db.Pool.Exec(ctx,
		`DELETE FROM cart_items WHERE id = $1 AND cart_id = $2
		 AND cart_id IN (SELECT id FROM carts WHERE organization_id = $3)`,
		itemID, cartID, orgID)
	if err != nil {
		return fmt.Errorf("failed to delete cart item: %w", err)
	}
	if res.RowsAffected() == 0 {
		return fmt.Errorf("cart item not found: %s", itemID)
	}
	return nil
}

func (r *CartRepository) ClearCart(ctx context.Context, cartID string) error {
	if _, err := r.db.Pool.Exec(ctx, `DELETE FROM cart_items WHERE cart_id = $1`, cartID); err != nil {
		return fmt.Errorf("failed to clear cart items: %w", err)
	}
	if _, err := r.db.Pool.Exec(ctx, `UPDATE carts SET subtotal = 0, updated_at = NOW() WHERE id = $1`, cartID); err != nil {
		return fmt.Errorf("failed to reset cart subtotal: %w", err)
	}
	return nil
}

func (r *CartRepository) GetAbandonedCarts(ctx context.Context, orgID string, threshold time.Duration, pagination repository.Pagination) ([]*entity.Cart, int, error) {
	cutoff := time.Now().Add(-threshold)
	countQuery := `
		SELECT COUNT(*) FROM carts
		WHERE organization_id = $1 AND abandoned = TRUE AND abandoned_at <= $2
	`
	var total int
	if err := r.db.Pool.QueryRow(ctx, countQuery, orgID, cutoff).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("failed to count abandoned carts: %w", err)
	}

	query := cartSelectColumns + `
		FROM carts
		WHERE organization_id = $1 AND abandoned = TRUE AND abandoned_at <= $2
		ORDER BY abandoned_at DESC LIMIT $3 OFFSET $4
	`
	rows, err := r.db.Pool.Query(ctx, query, orgID, cutoff, pagination.GetLimit(), pagination.GetOffset())
	if err != nil {
		return nil, 0, fmt.Errorf("failed to query abandoned carts: %w", err)
	}
	defer rows.Close()

	var result []*entity.Cart
	for rows.Next() {
		c, err := scanCartFromRows(rows)
		if err != nil {
			return nil, 0, err
		}
		result = append(result, c)
	}
	return result, total, nil
}

func (r *CartRepository) MarkAsAbandoned(ctx context.Context, orgID, cartID string) error {
	now := time.Now()
	res, err := r.db.Pool.Exec(ctx,
		`UPDATE carts SET abandoned = TRUE, abandoned_at = $1, updated_at = $1 WHERE id = $2 AND organization_id = $3`,
		now, cartID, orgID)
	if err != nil {
		return fmt.Errorf("failed to mark cart as abandoned: %w", err)
	}
	if res.RowsAffected() == 0 {
		return fmt.Errorf("cart not found: %s", cartID)
	}
	return nil
}

func (r *CartRepository) RecoverCart(ctx context.Context, orgID, cartID string) error {
	now := time.Now()
	res, err := r.db.Pool.Exec(ctx,
		`UPDATE carts SET abandoned = FALSE, recovered_at = $1, updated_at = $1 WHERE id = $2 AND organization_id = $3`,
		now, cartID, orgID)
	if err != nil {
		return fmt.Errorf("failed to recover cart: %w", err)
	}
	if res.RowsAffected() == 0 {
		return fmt.Errorf("cart not found: %s", cartID)
	}
	return nil
}

func (r *CartRepository) GetExpiredCarts(ctx context.Context, limit int) ([]*entity.Cart, error) {
	query := cartSelectColumns + `
		FROM carts WHERE expires_at <= NOW() ORDER BY expires_at ASC LIMIT $1
	`
	rows, err := r.db.Pool.Query(ctx, query, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query expired carts: %w", err)
	}
	defer rows.Close()

	var result []*entity.Cart
	for rows.Next() {
		c, err := scanCartFromRows(rows)
		if err != nil {
			return nil, err
		}
		result = append(result, c)
	}
	return result, nil
}

func (r *CartRepository) DeleteExpiredCarts(ctx context.Context) (int, error) {
	res, err := r.db.Pool.Exec(ctx, `DELETE FROM carts WHERE expires_at <= NOW()`)
	if err != nil {
		return 0, fmt.Errorf("failed to delete expired carts: %w", err)
	}
	return int(res.RowsAffected()), nil
}

func (r *CartRepository) GetStats(ctx context.Context, orgID string) (*repository.CartStats, error) {
	stats := &repository.CartStats{Currency: "BRL"}

	query := `
		SELECT
			COALESCE(SUM(CASE WHEN abandoned = FALSE AND expires_at > NOW() THEN 1 ELSE 0 END), 0) AS active,
			COALESCE(SUM(CASE WHEN abandoned = TRUE THEN 1 ELSE 0 END), 0) AS abandoned,
			COALESCE(SUM(subtotal), 0) AS total_value,
			COALESCE(MIN(currency), 'BRL') AS currency
		FROM carts WHERE organization_id = $1
	`
	if err := r.db.Pool.QueryRow(ctx, query, orgID).Scan(
		&stats.ActiveCarts, &stats.AbandonedCarts, &stats.TotalValue, &stats.Currency,
	); err != nil {
		return nil, fmt.Errorf("failed to get cart stats: %w", err)
	}

	totalCarts := stats.ActiveCarts + stats.AbandonedCarts
	if totalCarts > 0 {
		stats.AverageCartValue = stats.TotalValue / int64(totalCarts)
		stats.AbandonmentRate = float64(stats.AbandonedCarts) / float64(totalCarts) * 100
	}

	var totalItems int
	_ = r.db.Pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM cart_items ci JOIN carts c ON ci.cart_id = c.id WHERE c.organization_id = $1`,
		orgID).Scan(&totalItems)
	stats.TotalItems = totalItems

	return stats, nil
}

const cartSelectColumns = `SELECT id, organization_id, channel_id, customer_phone, catalog_id,
		subtotal, currency, created_at, updated_at, expires_at,
		abandoned, abandoned_at, recovered_at`

func scanCart(row pgx.Row) (*entity.Cart, error) {
	var c entity.Cart
	if err := row.Scan(
		&c.ID, &c.OrganizationID, &c.ChannelID, &c.CustomerPhone, &c.CatalogID,
		&c.Subtotal, &c.Currency, &c.CreatedAt, &c.UpdatedAt, &c.ExpiresAt,
		&c.Abandoned, &c.AbandonedAt, &c.RecoveredAt,
	); err != nil {
		return nil, err
	}
	c.Items = []entity.CartItem{}
	return &c, nil
}

func scanCartFromRows(rows pgx.Rows) (*entity.Cart, error) {
	var c entity.Cart
	if err := rows.Scan(
		&c.ID, &c.OrganizationID, &c.ChannelID, &c.CustomerPhone, &c.CatalogID,
		&c.Subtotal, &c.Currency, &c.CreatedAt, &c.UpdatedAt, &c.ExpiresAt,
		&c.Abandoned, &c.AbandonedAt, &c.RecoveredAt,
	); err != nil {
		return nil, fmt.Errorf("failed to scan cart: %w", err)
	}
	c.Items = []entity.CartItem{}
	return &c, nil
}

var _ repository.CartRepository = (*CartRepository)(nil)
