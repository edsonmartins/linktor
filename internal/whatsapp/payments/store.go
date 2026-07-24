package payments

import "context"

// PaymentStore defines the interface for payment persistence
type PaymentStore interface {
	Create(ctx context.Context, payment *Payment) error
	GetByID(ctx context.Context, id string) (*Payment, error)
	GetByReference(ctx context.Context, referenceID string) (*Payment, error)
	Update(ctx context.Context, payment *Payment) error
	// GetByCustomer returns a customer's payments scoped to an organization.
	// organizationID is REQUIRED: filtering by phone alone leaks other tenants'
	// financial history for the same number (IDOR). Mirrors OrderRepository /
	// CartRepository, which already scope GetByCustomer by org.
	GetByCustomer(ctx context.Context, organizationID, customerPhone string) ([]*Payment, error)
	GetStats(ctx context.Context, orgID string) (*PaymentStats, error)
}
