package repository

import (
	"context"

	"github.com/msgfy/linktor/internal/domain/entity"
)

// HistoryImportRepository defines the interface for history import persistence
type HistoryImportRepository interface {
	// Create creates a new history import job
	Create(ctx context.Context, importJob *entity.HistoryImport) error

	// FindByID finds a history import by ID
	FindByID(ctx context.Context, id string) (*entity.HistoryImport, error)

	// FindByChannelID finds all imports for a channel
	FindByChannelID(ctx context.Context, channelID string) ([]*entity.HistoryImport, error)

	// FindByTenantID finds all imports for a tenant with pagination
	FindByTenantID(ctx context.Context, tenantID string, params *ListParams) ([]*entity.HistoryImport, int64, error)

	// FindRunning finds all currently running imports
	FindRunning(ctx context.Context) ([]*entity.HistoryImport, error)

	// Update updates a history import job
	Update(ctx context.Context, importJob *entity.HistoryImport) error

	// Delete deletes a history import job
	Delete(ctx context.Context, id string) error
}
