package database

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/msgfy/linktor/internal/domain/entity"
	"github.com/msgfy/linktor/internal/domain/repository"
)

// OrderRepository implements repository.OrderRepository with PostgreSQL
type OrderRepository struct {
	db *PostgresDB
}

// NewOrderRepository creates a new PostgreSQL order repository
func NewOrderRepository(db *PostgresDB) *OrderRepository {
	return &OrderRepository{db: db}
}

func (r *OrderRepository) Create(ctx context.Context, order *entity.Order) error {
	if order.ID == "" {
		order.ID = uuid.New().String()
	}
	if order.CreatedAt.IsZero() {
		order.CreatedAt = time.Now()
	}
	order.UpdatedAt = time.Now()

	// The order row and its items must persist atomically. Previously the order
	// INSERT and the per-item INSERT loop ran on separate (auto-commit) calls,
	// so a failure partway through the items left a persisted order whose stored
	// total no longer matched its (missing) items. Wrap both in one transaction
	// so a mid-loop failure rolls the whole order back. Mirrors the tx pattern in
	// CampaignRepository.AddRecipients (Begin + deferred Rollback + Commit).
	tx, err := r.db.Pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin order tx: %w", err)
	}
	defer tx.Rollback(ctx)

	orderQuery := `
		INSERT INTO orders (
			id, organization_id, channel_id, conversation_id, catalog_id,
			customer_phone, customer_name, status, subtotal, tax, shipping,
			discount, total, currency, notes, message_id, tracking_number,
			tracking_url, created_at, updated_at, confirmed_at, shipped_at,
			delivered_at, cancelled_at
		) VALUES (
			$1, $2, $3, NULLIF($4,'')::uuid, $5, $6, $7, $8, $9, $10, $11, $12,
			$13, $14, $15, $16, $17, $18, $19, $20, $21, $22, $23, $24
		)
	`
	if _, err := tx.Exec(ctx, orderQuery,
		order.ID, order.OrganizationID, order.ChannelID, order.ConversationID,
		order.CatalogID, order.CustomerPhone, order.CustomerName, string(order.Status),
		order.Subtotal, order.Tax, order.Shipping, order.Discount, order.Total,
		order.Currency, order.Notes, order.MessageID, order.TrackingNumber,
		order.TrackingURL, order.CreatedAt, order.UpdatedAt,
		order.ConfirmedAt, order.ShippedAt, order.DeliveredAt, order.CancelledAt,
	); err != nil {
		return fmt.Errorf("failed to create order: %w", err)
	}

	itemQuery := `
		INSERT INTO order_items (
			id, order_id, product_id, product_name, product_sku,
			quantity, unit_price, total_price, currency, image_url
		) VALUES ($1, $2, $3, $4, NULLIF($5,''), $6, $7, $8, $9, NULLIF($10,''))
	`
	for i := range order.Items {
		if order.Items[i].ID == "" {
			order.Items[i].ID = uuid.New().String()
		}
		order.Items[i].OrderID = order.ID
		it := &order.Items[i]
		if _, err := tx.Exec(ctx, itemQuery,
			it.ID, it.OrderID, it.ProductID, it.ProductName, it.ProductSKU,
			it.Quantity, it.UnitPrice, it.TotalPrice, it.Currency, it.ImageURL,
		); err != nil {
			return fmt.Errorf("failed to add order item: %w", err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("failed to commit order: %w", err)
	}
	return nil
}

func (r *OrderRepository) GetByID(ctx context.Context, orgID, orderID string) (*entity.Order, error) {
	query := orderSelectColumns + ` FROM orders WHERE organization_id = $1 AND id = $2`
	order, err := scanOrder(r.db.Pool.QueryRow(ctx, query, orgID, orderID))
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("order not found: %s", orderID)
		}
		return nil, fmt.Errorf("failed to get order: %w", err)
	}
	return order, nil
}

func (r *OrderRepository) GetByMessageID(ctx context.Context, orgID, messageID string) (*entity.Order, error) {
	query := orderSelectColumns + ` FROM orders WHERE organization_id = $1 AND message_id = $2 LIMIT 1`
	order, err := scanOrder(r.db.Pool.QueryRow(ctx, query, orgID, messageID))
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("order not found by message_id: %s", messageID)
		}
		return nil, fmt.Errorf("failed to get order: %w", err)
	}
	return order, nil
}

func (r *OrderRepository) Update(ctx context.Context, order *entity.Order) error {
	order.UpdatedAt = time.Now()
	query := `
		UPDATE orders SET
			status = $1, subtotal = $2, tax = $3, shipping = $4,
			discount = $5, total = $6, currency = $7, notes = $8,
			message_id = $9, tracking_number = $10, tracking_url = $11,
			updated_at = $12, confirmed_at = $13, shipped_at = $14,
			delivered_at = $15, cancelled_at = $16
		WHERE id = $17 AND organization_id = $18
	`
	res, err := r.db.Pool.Exec(ctx, query,
		string(order.Status), order.Subtotal, order.Tax, order.Shipping,
		order.Discount, order.Total, order.Currency, order.Notes,
		order.MessageID, order.TrackingNumber, order.TrackingURL,
		order.UpdatedAt, order.ConfirmedAt, order.ShippedAt,
		order.DeliveredAt, order.CancelledAt,
		order.ID, order.OrganizationID,
	)
	if err != nil {
		return fmt.Errorf("failed to update order: %w", err)
	}
	if res.RowsAffected() == 0 {
		return fmt.Errorf("order not found: %s", order.ID)
	}
	return nil
}

func (r *OrderRepository) UpdateStatus(ctx context.Context, orgID, orderID string, status entity.OrderStatus) error {
	now := time.Now()
	tsCol := ""
	switch status {
	case entity.OrderStatusConfirmed:
		tsCol = "confirmed_at"
	case entity.OrderStatusShipped:
		tsCol = "shipped_at"
	case entity.OrderStatusDelivered:
		tsCol = "delivered_at"
	case entity.OrderStatusCancelled:
		tsCol = "cancelled_at"
	}

	var query string
	var args []interface{}
	if tsCol != "" {
		query = fmt.Sprintf(`UPDATE orders SET status = $1, %s = $2, updated_at = $3 WHERE id = $4 AND organization_id = $5`, tsCol)
		args = []interface{}{string(status), now, now, orderID, orgID}
	} else {
		query = `UPDATE orders SET status = $1, updated_at = $2 WHERE id = $3 AND organization_id = $4`
		args = []interface{}{string(status), now, orderID, orgID}
	}

	res, err := r.db.Pool.Exec(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("failed to update order status: %w", err)
	}
	if res.RowsAffected() == 0 {
		return fmt.Errorf("order not found: %s", orderID)
	}
	return nil
}

func (r *OrderRepository) Delete(ctx context.Context, orgID, orderID string) error {
	res, err := r.db.Pool.Exec(ctx, `DELETE FROM orders WHERE id = $1 AND organization_id = $2`, orderID, orgID)
	if err != nil {
		return fmt.Errorf("failed to delete order: %w", err)
	}
	if res.RowsAffected() == 0 {
		return fmt.Errorf("order not found: %s", orderID)
	}
	return nil
}

func (r *OrderRepository) List(ctx context.Context, orgID string, filters repository.OrderFilters, pagination repository.Pagination) ([]*entity.Order, int, error) {
	conditions := []string{"organization_id = $1"}
	args := []interface{}{orgID}
	idx := 2

	if filters.Status != nil {
		conditions = append(conditions, fmt.Sprintf("status = $%d", idx))
		args = append(args, string(*filters.Status))
		idx++
	}
	if filters.CustomerPhone != "" {
		conditions = append(conditions, fmt.Sprintf("customer_phone = $%d", idx))
		args = append(args, filters.CustomerPhone)
		idx++
	}
	if filters.ChannelID != "" {
		conditions = append(conditions, fmt.Sprintf("channel_id = $%d", idx))
		args = append(args, filters.ChannelID)
		idx++
	}
	if filters.CatalogID != "" {
		conditions = append(conditions, fmt.Sprintf("catalog_id = $%d", idx))
		args = append(args, filters.CatalogID)
		idx++
	}
	if filters.FromDate != nil {
		conditions = append(conditions, fmt.Sprintf("created_at >= $%d", idx))
		args = append(args, *filters.FromDate)
		idx++
	}
	if filters.ToDate != nil {
		conditions = append(conditions, fmt.Sprintf("created_at <= $%d", idx))
		args = append(args, *filters.ToDate)
		idx++
	}
	if filters.MinTotal != nil {
		conditions = append(conditions, fmt.Sprintf("total >= $%d", idx))
		args = append(args, *filters.MinTotal)
		idx++
	}
	if filters.MaxTotal != nil {
		conditions = append(conditions, fmt.Sprintf("total <= $%d", idx))
		args = append(args, *filters.MaxTotal)
		idx++
	}
	if filters.Search != "" {
		conditions = append(conditions, fmt.Sprintf("(customer_name ILIKE $%d OR customer_phone ILIKE $%d OR id::text ILIKE $%d)", idx, idx, idx))
		args = append(args, "%"+filters.Search+"%")
		idx++
	}

	where := strings.Join(conditions, " AND ")

	var total int
	if err := r.db.Pool.QueryRow(ctx, `SELECT COUNT(*) FROM orders WHERE `+where, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("failed to count orders: %w", err)
	}

	query := fmt.Sprintf(`%s FROM orders WHERE %s ORDER BY created_at DESC LIMIT $%d OFFSET $%d`,
		orderSelectColumns, where, idx, idx+1)
	args = append(args, pagination.GetLimit(), pagination.GetOffset())

	rows, err := r.db.Pool.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to query orders: %w", err)
	}
	defer rows.Close()

	var result []*entity.Order
	for rows.Next() {
		o, err := scanOrderFromRows(rows)
		if err != nil {
			return nil, 0, err
		}
		result = append(result, o)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("failed to iterate orders: %w", err)
	}
	return result, total, nil
}

func (r *OrderRepository) GetByCustomer(ctx context.Context, orgID, customerPhone string, pagination repository.Pagination) ([]*entity.Order, int, error) {
	filters := repository.OrderFilters{CustomerPhone: customerPhone}
	return r.List(ctx, orgID, filters, pagination)
}

func (r *OrderRepository) GetByChannel(ctx context.Context, orgID, channelID string, pagination repository.Pagination) ([]*entity.Order, int, error) {
	filters := repository.OrderFilters{ChannelID: channelID}
	return r.List(ctx, orgID, filters, pagination)
}

func (r *OrderRepository) GetOrderItems(ctx context.Context, orderID string) ([]entity.OrderItem, error) {
	query := `
		SELECT id, order_id, product_id, product_name, product_sku, quantity,
		       unit_price, total_price, currency, image_url
		FROM order_items WHERE order_id = $1 ORDER BY id
	`
	rows, err := r.db.Pool.Query(ctx, query, orderID)
	if err != nil {
		return nil, fmt.Errorf("failed to query order items: %w", err)
	}
	defer rows.Close()

	var items []entity.OrderItem
	for rows.Next() {
		var it entity.OrderItem
		var sku, img *string
		if err := rows.Scan(&it.ID, &it.OrderID, &it.ProductID, &it.ProductName,
			&sku, &it.Quantity, &it.UnitPrice, &it.TotalPrice, &it.Currency, &img); err != nil {
			return nil, fmt.Errorf("failed to scan order item: %w", err)
		}
		if sku != nil {
			it.ProductSKU = *sku
		}
		if img != nil {
			it.ImageURL = *img
		}
		items = append(items, it)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to iterate order items: %w", err)
	}
	return items, nil
}

func (r *OrderRepository) AddOrderItem(ctx context.Context, item *entity.OrderItem) error {
	if item.ID == "" {
		item.ID = uuid.New().String()
	}
	query := `
		INSERT INTO order_items (
			id, order_id, product_id, product_name, product_sku,
			quantity, unit_price, total_price, currency, image_url
		) VALUES ($1, $2, $3, $4, NULLIF($5,''), $6, $7, $8, $9, NULLIF($10,''))
	`
	_, err := r.db.Pool.Exec(ctx, query,
		item.ID, item.OrderID, item.ProductID, item.ProductName, item.ProductSKU,
		item.Quantity, item.UnitPrice, item.TotalPrice, item.Currency, item.ImageURL,
	)
	if err != nil {
		return fmt.Errorf("failed to add order item: %w", err)
	}
	return nil
}

func (r *OrderRepository) UpdateOrderItem(ctx context.Context, item *entity.OrderItem) error {
	query := `
		UPDATE order_items SET
			product_name = $1, product_sku = NULLIF($2,''), quantity = $3,
			unit_price = $4, total_price = $5, currency = $6, image_url = NULLIF($7,'')
		WHERE id = $8 AND order_id = $9
	`
	res, err := r.db.Pool.Exec(ctx, query,
		item.ProductName, item.ProductSKU, item.Quantity, item.UnitPrice,
		item.TotalPrice, item.Currency, item.ImageURL, item.ID, item.OrderID,
	)
	if err != nil {
		return fmt.Errorf("failed to update order item: %w", err)
	}
	if res.RowsAffected() == 0 {
		return fmt.Errorf("order item not found: %s", item.ID)
	}
	return nil
}

func (r *OrderRepository) DeleteOrderItem(ctx context.Context, orderID, itemID string) error {
	res, err := r.db.Pool.Exec(ctx, `DELETE FROM order_items WHERE id = $1 AND order_id = $2`, itemID, orderID)
	if err != nil {
		return fmt.Errorf("failed to delete order item: %w", err)
	}
	if res.RowsAffected() == 0 {
		return fmt.Errorf("order item not found: %s", itemID)
	}
	return nil
}

func (r *OrderRepository) GetStatusHistory(ctx context.Context, orderID string) ([]entity.OrderStatusHistory, error) {
	query := `
		SELECT id, order_id, status, notes, created_by, created_at
		FROM order_status_history WHERE order_id = $1 ORDER BY created_at ASC
	`
	rows, err := r.db.Pool.Query(ctx, query, orderID)
	if err != nil {
		return nil, fmt.Errorf("failed to query status history: %w", err)
	}
	defer rows.Close()

	var history []entity.OrderStatusHistory
	for rows.Next() {
		var h entity.OrderStatusHistory
		var status string
		var notes, createdBy *string
		if err := rows.Scan(&h.ID, &h.OrderID, &status, &notes, &createdBy, &h.CreatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan history: %w", err)
		}
		h.Status = entity.OrderStatus(status)
		if notes != nil {
			h.Notes = *notes
		}
		if createdBy != nil {
			h.CreatedBy = *createdBy
		}
		history = append(history, h)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to iterate status history: %w", err)
	}
	return history, nil
}

func (r *OrderRepository) AddStatusHistory(ctx context.Context, history *entity.OrderStatusHistory) error {
	if history.ID == "" {
		history.ID = uuid.New().String()
	}
	if history.CreatedAt.IsZero() {
		history.CreatedAt = time.Now()
	}
	query := `
		INSERT INTO order_status_history (id, order_id, status, notes, created_by, created_at)
		VALUES ($1, $2, $3, NULLIF($4,''), NULLIF($5,''), $6)
	`
	_, err := r.db.Pool.Exec(ctx, query,
		history.ID, history.OrderID, string(history.Status),
		history.Notes, history.CreatedBy, history.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to add status history: %w", err)
	}
	return nil
}

func (r *OrderRepository) GetStats(ctx context.Context, orgID string, filters repository.StatsFilters) (*repository.OrderStats, error) {
	stats := &repository.OrderStats{
		ByStatus:  make(map[entity.OrderStatus]int),
		ByChannel: make(map[string]int),
		Currency:  "BRL",
	}

	conditions := []string{"organization_id = $1"}
	args := []interface{}{orgID}
	idx := 2
	if filters.ChannelID != "" {
		conditions = append(conditions, fmt.Sprintf("channel_id = $%d", idx))
		args = append(args, filters.ChannelID)
		idx++
	}
	if filters.FromDate != nil {
		conditions = append(conditions, fmt.Sprintf("created_at >= $%d", idx))
		args = append(args, *filters.FromDate)
		idx++
	}
	if filters.ToDate != nil {
		conditions = append(conditions, fmt.Sprintf("created_at <= $%d", idx))
		args = append(args, *filters.ToDate)
		idx++
	}
	where := strings.Join(conditions, " AND ")

	overall := fmt.Sprintf(`
		SELECT COUNT(*) AS total_orders,
		       COALESCE(SUM(total), 0) AS total_revenue,
		       COALESCE(MIN(currency), 'BRL') AS currency
		FROM orders WHERE %s
	`, where)

	if err := r.db.Pool.QueryRow(ctx, overall, args...).Scan(
		&stats.TotalOrders, &stats.TotalRevenue, &stats.Currency,
	); err != nil {
		return nil, fmt.Errorf("failed to get order stats: %w", err)
	}
	if stats.TotalOrders > 0 {
		stats.AverageOrderValue = stats.TotalRevenue / int64(stats.TotalOrders)
	}

	statusRows, err := r.db.Pool.Query(ctx,
		fmt.Sprintf(`SELECT status, COUNT(*) FROM orders WHERE %s GROUP BY status`, where), args...)
	if err == nil {
		defer statusRows.Close()
		for statusRows.Next() {
			var s string
			var c int
			if err := statusRows.Scan(&s, &c); err == nil {
				stats.ByStatus[entity.OrderStatus(s)] = c
			}
		}
		if err := statusRows.Err(); err != nil {
			return nil, fmt.Errorf("failed to iterate order status stats: %w", err)
		}
	}

	channelRows, err := r.db.Pool.Query(ctx,
		fmt.Sprintf(`SELECT channel_id, COUNT(*) FROM orders WHERE %s GROUP BY channel_id`, where), args...)
	if err == nil {
		defer channelRows.Close()
		for channelRows.Next() {
			var ch string
			var c int
			if err := channelRows.Scan(&ch, &c); err == nil {
				stats.ByChannel[ch] = c
			}
		}
		if err := channelRows.Err(); err != nil {
			return nil, fmt.Errorf("failed to iterate order channel stats: %w", err)
		}
	}

	return stats, nil
}

const orderSelectColumns = `SELECT id, organization_id, channel_id, COALESCE(conversation_id::text, ''),
		catalog_id, customer_phone, customer_name, status, subtotal, tax, shipping,
		discount, total, currency, notes, message_id, tracking_number, tracking_url,
		created_at, updated_at, confirmed_at, shipped_at, delivered_at, cancelled_at`

func scanOrder(row pgx.Row) (*entity.Order, error) {
	var o entity.Order
	var status string
	var customerName, notes, messageID, trackingNumber, trackingURL *string
	if err := row.Scan(
		&o.ID, &o.OrganizationID, &o.ChannelID, &o.ConversationID, &o.CatalogID,
		&o.CustomerPhone, &customerName, &status, &o.Subtotal, &o.Tax, &o.Shipping,
		&o.Discount, &o.Total, &o.Currency, &notes, &messageID, &trackingNumber,
		&trackingURL, &o.CreatedAt, &o.UpdatedAt, &o.ConfirmedAt, &o.ShippedAt,
		&o.DeliveredAt, &o.CancelledAt,
	); err != nil {
		return nil, err
	}
	o.Status = entity.OrderStatus(status)
	if customerName != nil {
		o.CustomerName = *customerName
	}
	if notes != nil {
		o.Notes = *notes
	}
	if messageID != nil {
		o.MessageID = *messageID
	}
	if trackingNumber != nil {
		o.TrackingNumber = *trackingNumber
	}
	if trackingURL != nil {
		o.TrackingURL = *trackingURL
	}
	o.Items = []entity.OrderItem{}
	return &o, nil
}

func scanOrderFromRows(rows pgx.Rows) (*entity.Order, error) {
	var o entity.Order
	var status string
	var customerName, notes, messageID, trackingNumber, trackingURL *string
	if err := rows.Scan(
		&o.ID, &o.OrganizationID, &o.ChannelID, &o.ConversationID, &o.CatalogID,
		&o.CustomerPhone, &customerName, &status, &o.Subtotal, &o.Tax, &o.Shipping,
		&o.Discount, &o.Total, &o.Currency, &notes, &messageID, &trackingNumber,
		&trackingURL, &o.CreatedAt, &o.UpdatedAt, &o.ConfirmedAt, &o.ShippedAt,
		&o.DeliveredAt, &o.CancelledAt,
	); err != nil {
		return nil, fmt.Errorf("failed to scan order: %w", err)
	}
	o.Status = entity.OrderStatus(status)
	if customerName != nil {
		o.CustomerName = *customerName
	}
	if notes != nil {
		o.Notes = *notes
	}
	if messageID != nil {
		o.MessageID = *messageID
	}
	if trackingNumber != nil {
		o.TrackingNumber = *trackingNumber
	}
	if trackingURL != nil {
		o.TrackingURL = *trackingURL
	}
	o.Items = []entity.OrderItem{}
	return &o, nil
}

var _ repository.OrderRepository = (*OrderRepository)(nil)
