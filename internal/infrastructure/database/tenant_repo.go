package database

import (
	"context"
	"encoding/json"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/msgfy/linktor/internal/domain/entity"
	"github.com/msgfy/linktor/pkg/errors"
)

// TenantRepository implements repository.TenantRepository with PostgreSQL
type TenantRepository struct {
	db *PostgresDB
}

// NewTenantRepository creates a new PostgreSQL tenant repository
func NewTenantRepository(db *PostgresDB) *TenantRepository {
	return &TenantRepository{db: db}
}

// Create creates a new tenant
func (r *TenantRepository) Create(ctx context.Context, tenant *entity.Tenant) error {
	settings, err := json.Marshal(tenant.Settings)
	if err != nil {
		return errors.Wrap(err, errors.ErrCodeInternal, "failed to marshal settings")
	}

	limits, err := json.Marshal(tenant.Limits)
	if err != nil {
		return errors.Wrap(err, errors.ErrCodeInternal, "failed to marshal limits")
	}

	query := `
		INSERT INTO tenants (
			id, name, slug, plan, status, settings, limits, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`

	_, err = r.db.Pool.Exec(ctx, query,
		tenant.ID,
		tenant.Name,
		tenant.Slug,
		string(tenant.Plan),
		string(tenant.Status),
		settings,
		limits,
		tenant.CreatedAt,
		tenant.UpdatedAt,
	)

	if err != nil {
		return errors.Wrap(err, errors.ErrCodeInternal, "failed to create tenant")
	}

	return nil
}

// FindByID finds a tenant by ID
func (r *TenantRepository) FindByID(ctx context.Context, id string) (*entity.Tenant, error) {
	query := `
		SELECT id, name, slug, plan, status, settings, limits, created_at, updated_at
		FROM tenants
		WHERE id = $1
	`

	tenant, err := r.scanTenant(r.db.Pool.QueryRow(ctx, query, id))
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, errors.New(errors.ErrCodeTenantNotFound, "tenant not found")
		}
		return nil, errors.Wrap(err, errors.ErrCodeInternal, "failed to find tenant")
	}

	return tenant, nil
}

// FindBySlug finds a tenant by slug
func (r *TenantRepository) FindBySlug(ctx context.Context, slug string) (*entity.Tenant, error) {
	query := `
		SELECT id, name, slug, plan, status, settings, limits, created_at, updated_at
		FROM tenants
		WHERE slug = $1
	`

	tenant, err := r.scanTenant(r.db.Pool.QueryRow(ctx, query, slug))
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, errors.New(errors.ErrCodeTenantNotFound, "tenant not found")
		}
		return nil, errors.Wrap(err, errors.ErrCodeInternal, "failed to find tenant")
	}

	return tenant, nil
}

// Update updates a tenant
func (r *TenantRepository) Update(ctx context.Context, tenant *entity.Tenant) error {
	tenant.UpdatedAt = time.Now()

	settings, err := json.Marshal(tenant.Settings)
	if err != nil {
		return errors.Wrap(err, errors.ErrCodeInternal, "failed to marshal settings")
	}

	limits, err := json.Marshal(tenant.Limits)
	if err != nil {
		return errors.Wrap(err, errors.ErrCodeInternal, "failed to marshal limits")
	}

	query := `
		UPDATE tenants SET
			name = $1,
			slug = $2,
			plan = $3,
			status = $4,
			settings = $5,
			limits = $6,
			updated_at = $7
		WHERE id = $8
	`

	result, err := r.db.Pool.Exec(ctx, query,
		tenant.Name,
		tenant.Slug,
		string(tenant.Plan),
		string(tenant.Status),
		settings,
		limits,
		tenant.UpdatedAt,
		tenant.ID,
	)

	if err != nil {
		return errors.Wrap(err, errors.ErrCodeInternal, "failed to update tenant")
	}

	if result.RowsAffected() == 0 {
		return errors.New(errors.ErrCodeTenantNotFound, "tenant not found")
	}

	return nil
}

// Delete deletes a tenant
func (r *TenantRepository) Delete(ctx context.Context, id string) error {
	result, err := r.db.Pool.Exec(ctx, "DELETE FROM tenants WHERE id = $1", id)
	if err != nil {
		return errors.Wrap(err, errors.ErrCodeInternal, "failed to delete tenant")
	}

	if result.RowsAffected() == 0 {
		return errors.New(errors.ErrCodeTenantNotFound, "tenant not found")
	}

	return nil
}

// Helper methods

func (r *TenantRepository) scanTenant(row pgx.Row) (*entity.Tenant, error) {
	var t entity.Tenant
	var plan, status string
	var settingsJSON, limitsJSON []byte

	err := row.Scan(
		&t.ID, &t.Name, &t.Slug, &plan, &status, &settingsJSON, &limitsJSON,
		&t.CreatedAt, &t.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}

	t.Plan = entity.Plan(plan)
	t.Status = entity.TenantStatus(status)

	if len(settingsJSON) > 0 {
		if err := json.Unmarshal(settingsJSON, &t.Settings); err != nil {
			return nil, errors.Wrap(err, errors.ErrCodeInternal, "failed to unmarshal settings")
		}
	} else {
		t.Settings = make(map[string]string)
	}

	if len(limitsJSON) > 0 {
		if err := json.Unmarshal(limitsJSON, &t.Limits); err != nil {
			return nil, errors.Wrap(err, errors.ErrCodeInternal, "failed to unmarshal limits")
		}
	}

	return &t, nil
}
