package database

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/msgfy/linktor/internal/whatsapp/payments"
)

// PaymentRepository implements payments.PaymentStore with PostgreSQL
type PaymentRepository struct {
	db *PostgresDB
}

// NewPaymentRepository creates a new PostgreSQL payment repository
func NewPaymentRepository(db *PostgresDB) *PaymentRepository {
	return &PaymentRepository{db: db}
}

// Create creates a new payment record
func (r *PaymentRepository) Create(ctx context.Context, payment *payments.Payment) error {
	query := `
		INSERT INTO whatsapp_payments (
			id, organization_id, channel_id, order_id, reference_id, customer_phone,
			amount, currency, status, type, method, gateway_payment_id, gateway_order_id,
			description, expires_at, paid_at, failed_at, refunded_at, failure_reason,
			message_id, metadata, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20, $21, $22, $23)
	`

	// Payment struct does not have a Metadata field; store as empty JSON object
	metadata := []byte("{}")

	_, err := r.db.Pool.Exec(ctx, query,
		payment.ID,
		payment.OrganizationID,
		payment.ChannelID,
		paymentNullString(payment.OrderID),
		payment.ReferenceID,
		payment.CustomerPhone,
		payment.Amount,
		payment.Currency,
		string(payment.Status),
		string(payment.Type),
		paymentNullString(string(payment.Method)),
		paymentNullString(payment.GatewayPaymentID),
		paymentNullString(payment.GatewayOrderID),
		paymentNullString(payment.Description),
		payment.ExpiresAt,
		payment.PaidAt,
		payment.FailedAt,
		payment.RefundedAt,
		paymentNullString(payment.FailureReason),
		paymentNullString(payment.MessageID),
		metadata,
		payment.CreatedAt,
		payment.UpdatedAt,
	)

	if err != nil {
		return fmt.Errorf("failed to create payment: %w", err)
	}

	return nil
}

// GetByID retrieves a payment by its ID
func (r *PaymentRepository) GetByID(ctx context.Context, id string) (*payments.Payment, error) {
	query := `
		SELECT id, organization_id, channel_id, order_id, reference_id, customer_phone,
		       amount, currency, status, type, method, gateway_payment_id, gateway_order_id,
		       description, expires_at, paid_at, failed_at, refunded_at, failure_reason,
		       message_id, metadata, created_at, updated_at
		FROM whatsapp_payments
		WHERE id = $1
	`

	payment, err := scanPayment(r.db.Pool.QueryRow(ctx, query, id))
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("payment not found: %s", id)
		}
		return nil, fmt.Errorf("failed to get payment: %w", err)
	}

	return payment, nil
}

// GetByReference retrieves a payment by its reference ID
func (r *PaymentRepository) GetByReference(ctx context.Context, referenceID string) (*payments.Payment, error) {
	query := `
		SELECT id, organization_id, channel_id, order_id, reference_id, customer_phone,
		       amount, currency, status, type, method, gateway_payment_id, gateway_order_id,
		       description, expires_at, paid_at, failed_at, refunded_at, failure_reason,
		       message_id, metadata, created_at, updated_at
		FROM whatsapp_payments
		WHERE reference_id = $1
		LIMIT 1
	`

	payment, err := scanPayment(r.db.Pool.QueryRow(ctx, query, referenceID))
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("payment not found for reference: %s", referenceID)
		}
		return nil, fmt.Errorf("failed to get payment by reference: %w", err)
	}

	return payment, nil
}

// Update updates an existing payment record
func (r *PaymentRepository) Update(ctx context.Context, payment *payments.Payment) error {
	query := `
		UPDATE whatsapp_payments SET
			status = $1,
			method = $2,
			gateway_payment_id = $3,
			gateway_order_id = $4,
			description = $5,
			expires_at = $6,
			paid_at = $7,
			failed_at = $8,
			refunded_at = $9,
			failure_reason = $10,
			message_id = $11,
			metadata = $12,
			updated_at = $13
		WHERE id = $14
	`

	// Payment struct does not have a Metadata field; store as empty JSON object
	metadata := []byte("{}")

	result, err := r.db.Pool.Exec(ctx, query,
		string(payment.Status),
		paymentNullString(string(payment.Method)),
		paymentNullString(payment.GatewayPaymentID),
		paymentNullString(payment.GatewayOrderID),
		paymentNullString(payment.Description),
		payment.ExpiresAt,
		payment.PaidAt,
		payment.FailedAt,
		payment.RefundedAt,
		paymentNullString(payment.FailureReason),
		paymentNullString(payment.MessageID),
		metadata,
		payment.UpdatedAt,
		payment.ID,
	)

	if err != nil {
		return fmt.Errorf("failed to update payment: %w", err)
	}

	if result.RowsAffected() == 0 {
		return fmt.Errorf("payment not found: %s", payment.ID)
	}

	return nil
}

// GetByCustomer retrieves all payments for a customer phone number
func (r *PaymentRepository) GetByCustomer(ctx context.Context, customerPhone string) ([]*payments.Payment, error) {
	query := `
		SELECT id, organization_id, channel_id, order_id, reference_id, customer_phone,
		       amount, currency, status, type, method, gateway_payment_id, gateway_order_id,
		       description, expires_at, paid_at, failed_at, refunded_at, failure_reason,
		       message_id, metadata, created_at, updated_at
		FROM whatsapp_payments
		WHERE customer_phone = $1
		ORDER BY created_at DESC
	`

	rows, err := r.db.Pool.Query(ctx, query, customerPhone)
	if err != nil {
		return nil, fmt.Errorf("failed to query payments by customer: %w", err)
	}
	defer rows.Close()

	var result []*payments.Payment
	for rows.Next() {
		payment, err := scanPaymentFromRows(rows)
		if err != nil {
			return nil, err
		}
		result = append(result, payment)
	}

	return result, nil
}

// GetStats retrieves aggregated payment statistics for an organization
func (r *PaymentRepository) GetStats(ctx context.Context, orgID string) (*payments.PaymentStats, error) {
	stats := &payments.PaymentStats{
		ByStatus: make(map[payments.PaymentStatus]int),
		ByMethod: make(map[payments.PaymentMethod]int),
	}

	// Get overall stats
	overallQuery := `
		SELECT
			COUNT(*) AS total_payments,
			COALESCE(SUM(CASE WHEN status = 'success' THEN 1 ELSE 0 END), 0) AS successful_payments,
			COALESCE(SUM(CASE WHEN status = 'failed' THEN 1 ELSE 0 END), 0) AS failed_payments,
			COALESCE(SUM(CASE WHEN status = 'success' THEN amount ELSE 0 END), 0) AS total_amount,
			COALESCE(SUM(CASE WHEN status = 'refunded' THEN amount ELSE 0 END), 0) AS refunded_amount,
			COALESCE(MIN(CASE WHEN status = 'success' THEN currency END), '') AS currency
		FROM whatsapp_payments
		WHERE organization_id = $1
	`

	err := r.db.Pool.QueryRow(ctx, overallQuery, orgID).Scan(
		&stats.TotalPayments,
		&stats.SuccessfulPayments,
		&stats.FailedPayments,
		&stats.TotalAmount,
		&stats.RefundedAmount,
		&stats.Currency,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get payment stats: %w", err)
	}

	if stats.TotalPayments > 0 {
		stats.SuccessRate = float64(stats.SuccessfulPayments) / float64(stats.TotalPayments) * 100
	}

	// Get counts by status
	statusQuery := `
		SELECT status, COUNT(*) AS count
		FROM whatsapp_payments
		WHERE organization_id = $1
		GROUP BY status
	`

	statusRows, err := r.db.Pool.Query(ctx, statusQuery, orgID)
	if err != nil {
		return nil, fmt.Errorf("failed to get payment stats by status: %w", err)
	}
	defer statusRows.Close()

	for statusRows.Next() {
		var status string
		var count int
		if err := statusRows.Scan(&status, &count); err != nil {
			return nil, fmt.Errorf("failed to scan status stat: %w", err)
		}
		stats.ByStatus[payments.PaymentStatus(status)] = count
	}

	// Get counts by method
	methodQuery := `
		SELECT method, COUNT(*) AS count
		FROM whatsapp_payments
		WHERE organization_id = $1 AND method IS NOT NULL AND method != ''
		GROUP BY method
	`

	methodRows, err := r.db.Pool.Query(ctx, methodQuery, orgID)
	if err != nil {
		return nil, fmt.Errorf("failed to get payment stats by method: %w", err)
	}
	defer methodRows.Close()

	for methodRows.Next() {
		var method string
		var count int
		if err := methodRows.Scan(&method, &count); err != nil {
			return nil, fmt.Errorf("failed to scan method stat: %w", err)
		}
		stats.ByMethod[payments.PaymentMethod(method)] = count
	}

	return stats, nil
}

// scanPayment scans a single payment row
func scanPayment(row pgx.Row) (*payments.Payment, error) {
	var p payments.Payment
	var orderID, method, gatewayPaymentID, gatewayOrderID *string
	var description, failureReason, messageID *string
	var metadata []byte
	var status, paymentType string

	err := row.Scan(
		&p.ID, &p.OrganizationID, &p.ChannelID, &orderID, &p.ReferenceID, &p.CustomerPhone,
		&p.Amount, &p.Currency, &status, &paymentType, &method, &gatewayPaymentID, &gatewayOrderID,
		&description, &p.ExpiresAt, &p.PaidAt, &p.FailedAt, &p.RefundedAt, &failureReason,
		&messageID, &metadata, &p.CreatedAt, &p.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}

	p.Status = payments.PaymentStatus(status)
	p.Type = payments.PaymentType(paymentType)

	if orderID != nil {
		p.OrderID = *orderID
	}
	if method != nil {
		p.Method = payments.PaymentMethod(*method)
	}
	if gatewayPaymentID != nil {
		p.GatewayPaymentID = *gatewayPaymentID
	}
	if gatewayOrderID != nil {
		p.GatewayOrderID = *gatewayOrderID
	}
	if description != nil {
		p.Description = *description
	}
	if failureReason != nil {
		p.FailureReason = *failureReason
	}
	if messageID != nil {
		p.MessageID = *messageID
	}

	return &p, nil
}

// scanPaymentFromRows scans a payment from rows
func scanPaymentFromRows(rows pgx.Rows) (*payments.Payment, error) {
	var p payments.Payment
	var orderID, method, gatewayPaymentID, gatewayOrderID *string
	var description, failureReason, messageID *string
	var metadata []byte
	var status, paymentType string

	err := rows.Scan(
		&p.ID, &p.OrganizationID, &p.ChannelID, &orderID, &p.ReferenceID, &p.CustomerPhone,
		&p.Amount, &p.Currency, &status, &paymentType, &method, &gatewayPaymentID, &gatewayOrderID,
		&description, &p.ExpiresAt, &p.PaidAt, &p.FailedAt, &p.RefundedAt, &failureReason,
		&messageID, &metadata, &p.CreatedAt, &p.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to scan payment: %w", err)
	}

	p.Status = payments.PaymentStatus(status)
	p.Type = payments.PaymentType(paymentType)

	if orderID != nil {
		p.OrderID = *orderID
	}
	if method != nil {
		p.Method = payments.PaymentMethod(*method)
	}
	if gatewayPaymentID != nil {
		p.GatewayPaymentID = *gatewayPaymentID
	}
	if gatewayOrderID != nil {
		p.GatewayOrderID = *gatewayOrderID
	}
	if description != nil {
		p.Description = *description
	}
	if failureReason != nil {
		p.FailureReason = *failureReason
	}
	if messageID != nil {
		p.MessageID = *messageID
	}

	return &p, nil
}

// paymentNullString converts an empty string to nil for nullable DB columns
func paymentNullString(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

// Ensure PaymentRepository implements payments.PaymentStore
var _ payments.PaymentStore = (*PaymentRepository)(nil)
