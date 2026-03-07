package payments

import "context"

// PaymentStore defines the interface for payment persistence
type PaymentStore interface {
	Create(ctx context.Context, payment *Payment) error
	GetByID(ctx context.Context, id string) (*Payment, error)
	GetByReference(ctx context.Context, referenceID string) (*Payment, error)
	Update(ctx context.Context, payment *Payment) error
	GetByCustomer(ctx context.Context, customerPhone string) ([]*Payment, error)
	GetStats(ctx context.Context, orgID string) (*PaymentStats, error)
}
