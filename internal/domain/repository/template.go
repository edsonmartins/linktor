package repository

import (
	"context"

	"github.com/msgfy/linktor/internal/domain/entity"
)

// TemplateRepository defines the interface for template data access
type TemplateRepository interface {
	// Create creates a new template
	Create(ctx context.Context, template *entity.Template) error

	// FindByID finds a template by ID
	FindByID(ctx context.Context, id string) (*entity.Template, error)

	// FindByExternalID finds a template by its external (Meta) ID
	FindByExternalID(ctx context.Context, externalID string) (*entity.Template, error)

	// FindByName finds a template by name and language
	FindByName(ctx context.Context, tenantID, channelID, name, language string) (*entity.Template, error)

	// FindByTenant finds all templates for a tenant
	FindByTenant(ctx context.Context, tenantID string, params *ListParams) ([]*entity.Template, int64, error)

	// FindByChannel finds all templates for a channel
	FindByChannel(ctx context.Context, channelID string, params *ListParams) ([]*entity.Template, int64, error)

	// FindByStatus finds templates by status
	FindByStatus(ctx context.Context, tenantID string, status entity.TemplateStatus, params *ListParams) ([]*entity.Template, int64, error)

	// FindNeedsSync finds templates that need to be synced with Meta
	FindNeedsSync(ctx context.Context, channelID string, syncInterval int64) ([]*entity.Template, error)

	// Update updates a template
	Update(ctx context.Context, template *entity.Template) error

	// UpdateStatus updates a template's status
	UpdateStatus(ctx context.Context, id string, status entity.TemplateStatus, reason string) error

	// UpdateQuality updates a template's quality score
	UpdateQuality(ctx context.Context, id string, quality entity.TemplateQuality) error

	// Delete deletes a template
	Delete(ctx context.Context, id string) error

	// DeleteByExternalID deletes a template by its external (Meta) ID
	DeleteByExternalID(ctx context.Context, externalID string) error

	// CountByChannel counts templates for a channel
	CountByChannel(ctx context.Context, channelID string) (int64, error)

	// UpsertByExternalID creates or updates a template by its external ID
	UpsertByExternalID(ctx context.Context, template *entity.Template) error
}
