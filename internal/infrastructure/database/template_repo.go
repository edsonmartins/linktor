package database

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/msgfy/linktor/internal/domain/entity"
	"github.com/msgfy/linktor/internal/domain/repository"
	"github.com/msgfy/linktor/pkg/errors"
)

// TemplateRepository implements repository.TemplateRepository with PostgreSQL
type TemplateRepository struct {
	db *PostgresDB
}

// NewTemplateRepository creates a new PostgreSQL template repository
func NewTemplateRepository(db *PostgresDB) *TemplateRepository {
	return &TemplateRepository{db: db}
}

// Create creates a new template
func (r *TemplateRepository) Create(ctx context.Context, template *entity.Template) error {
	components, err := json.Marshal(template.Components)
	if err != nil {
		return errors.Wrap(err, errors.ErrCodeInternal, "failed to marshal components")
	}

	query := `
		INSERT INTO templates (
			id, tenant_id, channel_id, external_id, name, language, category,
			status, quality_score, components, rejection_reason, last_synced_at,
			created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)
	`

	_, err = r.db.Pool.Exec(ctx, query,
		template.ID,
		template.TenantID,
		template.ChannelID,
		nullString(template.ExternalID),
		template.Name,
		template.Language,
		string(template.Category),
		string(template.Status),
		string(template.QualityScore),
		components,
		nullString(template.RejectionReason),
		template.LastSyncedAt,
		template.CreatedAt,
		template.UpdatedAt,
	)

	if err != nil {
		return errors.Wrap(err, errors.ErrCodeInternal, "failed to create template")
	}

	return nil
}

// FindByID finds a template by ID
func (r *TemplateRepository) FindByID(ctx context.Context, id string) (*entity.Template, error) {
	query := `
		SELECT id, tenant_id, channel_id, external_id, name, language, category,
		       status, quality_score, components, rejection_reason, last_synced_at,
		       created_at, updated_at
		FROM templates
		WHERE id = $1
	`

	template, err := r.scanTemplate(r.db.Pool.QueryRow(ctx, query, id))
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, errors.New(errors.ErrCodeNotFound, "template not found")
		}
		return nil, errors.Wrap(err, errors.ErrCodeInternal, "failed to find template")
	}

	return template, nil
}

// FindByExternalID finds a template by its external (Meta) ID
func (r *TemplateRepository) FindByExternalID(ctx context.Context, externalID string) (*entity.Template, error) {
	query := `
		SELECT id, tenant_id, channel_id, external_id, name, language, category,
		       status, quality_score, components, rejection_reason, last_synced_at,
		       created_at, updated_at
		FROM templates
		WHERE external_id = $1
	`

	template, err := r.scanTemplate(r.db.Pool.QueryRow(ctx, query, externalID))
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, errors.New(errors.ErrCodeNotFound, "template not found")
		}
		return nil, errors.Wrap(err, errors.ErrCodeInternal, "failed to find template by external ID")
	}

	return template, nil
}

// FindByName finds a template by name and language
func (r *TemplateRepository) FindByName(ctx context.Context, tenantID, channelID, name, language string) (*entity.Template, error) {
	query := `
		SELECT id, tenant_id, channel_id, external_id, name, language, category,
		       status, quality_score, components, rejection_reason, last_synced_at,
		       created_at, updated_at
		FROM templates
		WHERE tenant_id = $1 AND channel_id = $2 AND name = $3 AND language = $4
	`

	template, err := r.scanTemplate(r.db.Pool.QueryRow(ctx, query, tenantID, channelID, name, language))
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, errors.New(errors.ErrCodeNotFound, "template not found")
		}
		return nil, errors.Wrap(err, errors.ErrCodeInternal, "failed to find template by name")
	}

	return template, nil
}

// FindByTenant finds all templates for a tenant
func (r *TemplateRepository) FindByTenant(ctx context.Context, tenantID string, params *repository.ListParams) ([]*entity.Template, int64, error) {
	if params == nil {
		params = repository.NewListParams()
	}

	// Count total
	countQuery := `SELECT COUNT(*) FROM templates WHERE tenant_id = $1`
	var total int64
	if err := r.db.Pool.QueryRow(ctx, countQuery, tenantID).Scan(&total); err != nil {
		return nil, 0, errors.Wrap(err, errors.ErrCodeInternal, "failed to count templates")
	}

	// Get templates
	query := `
		SELECT id, tenant_id, channel_id, external_id, name, language, category,
		       status, quality_score, components, rejection_reason, last_synced_at,
		       created_at, updated_at
		FROM templates
		WHERE tenant_id = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3
	`

	rows, err := r.db.Pool.Query(ctx, query, tenantID, params.Limit, params.Offset)
	if err != nil {
		return nil, 0, errors.Wrap(err, errors.ErrCodeInternal, "failed to query templates")
	}
	defer rows.Close()

	templates, err := r.scanTemplates(rows)
	if err != nil {
		return nil, 0, err
	}

	return templates, total, nil
}

// FindByChannel finds all templates for a channel
func (r *TemplateRepository) FindByChannel(ctx context.Context, channelID string, params *repository.ListParams) ([]*entity.Template, int64, error) {
	if params == nil {
		params = repository.NewListParams()
	}

	// Count total
	countQuery := `SELECT COUNT(*) FROM templates WHERE channel_id = $1`
	var total int64
	if err := r.db.Pool.QueryRow(ctx, countQuery, channelID).Scan(&total); err != nil {
		return nil, 0, errors.Wrap(err, errors.ErrCodeInternal, "failed to count templates")
	}

	// Get templates
	query := `
		SELECT id, tenant_id, channel_id, external_id, name, language, category,
		       status, quality_score, components, rejection_reason, last_synced_at,
		       created_at, updated_at
		FROM templates
		WHERE channel_id = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3
	`

	rows, err := r.db.Pool.Query(ctx, query, channelID, params.Limit, params.Offset)
	if err != nil {
		return nil, 0, errors.Wrap(err, errors.ErrCodeInternal, "failed to query templates")
	}
	defer rows.Close()

	templates, err := r.scanTemplates(rows)
	if err != nil {
		return nil, 0, err
	}

	return templates, total, nil
}

// FindByStatus finds templates by status
func (r *TemplateRepository) FindByStatus(ctx context.Context, tenantID string, status entity.TemplateStatus, params *repository.ListParams) ([]*entity.Template, int64, error) {
	if params == nil {
		params = repository.NewListParams()
	}

	// Count total
	countQuery := `SELECT COUNT(*) FROM templates WHERE tenant_id = $1 AND status = $2`
	var total int64
	if err := r.db.Pool.QueryRow(ctx, countQuery, tenantID, string(status)).Scan(&total); err != nil {
		return nil, 0, errors.Wrap(err, errors.ErrCodeInternal, "failed to count templates")
	}

	// Get templates
	query := `
		SELECT id, tenant_id, channel_id, external_id, name, language, category,
		       status, quality_score, components, rejection_reason, last_synced_at,
		       created_at, updated_at
		FROM templates
		WHERE tenant_id = $1 AND status = $2
		ORDER BY created_at DESC
		LIMIT $3 OFFSET $4
	`

	rows, err := r.db.Pool.Query(ctx, query, tenantID, string(status), params.Limit, params.Offset)
	if err != nil {
		return nil, 0, errors.Wrap(err, errors.ErrCodeInternal, "failed to query templates")
	}
	defer rows.Close()

	templates, err := r.scanTemplates(rows)
	if err != nil {
		return nil, 0, err
	}

	return templates, total, nil
}

// FindNeedsSync finds templates that need to be synced with Meta
func (r *TemplateRepository) FindNeedsSync(ctx context.Context, channelID string, syncIntervalSeconds int64) ([]*entity.Template, error) {
	query := `
		SELECT id, tenant_id, channel_id, external_id, name, language, category,
		       status, quality_score, components, rejection_reason, last_synced_at,
		       created_at, updated_at
		FROM templates
		WHERE channel_id = $1
		  AND (last_synced_at IS NULL OR last_synced_at < NOW() - INTERVAL '1 second' * $2)
		ORDER BY last_synced_at ASC NULLS FIRST
		LIMIT 100
	`

	rows, err := r.db.Pool.Query(ctx, query, channelID, syncIntervalSeconds)
	if err != nil {
		return nil, errors.Wrap(err, errors.ErrCodeInternal, "failed to query templates needing sync")
	}
	defer rows.Close()

	return r.scanTemplates(rows)
}

// Update updates a template
func (r *TemplateRepository) Update(ctx context.Context, template *entity.Template) error {
	components, err := json.Marshal(template.Components)
	if err != nil {
		return errors.Wrap(err, errors.ErrCodeInternal, "failed to marshal components")
	}

	query := `
		UPDATE templates SET
			external_id = $2,
			name = $3,
			language = $4,
			category = $5,
			status = $6,
			quality_score = $7,
			components = $8,
			rejection_reason = $9,
			last_synced_at = $10,
			updated_at = $11
		WHERE id = $1
	`

	result, err := r.db.Pool.Exec(ctx, query,
		template.ID,
		nullString(template.ExternalID),
		template.Name,
		template.Language,
		string(template.Category),
		string(template.Status),
		string(template.QualityScore),
		components,
		nullString(template.RejectionReason),
		template.LastSyncedAt,
		time.Now(),
	)

	if err != nil {
		return errors.Wrap(err, errors.ErrCodeInternal, "failed to update template")
	}

	if result.RowsAffected() == 0 {
		return errors.New(errors.ErrCodeNotFound, "template not found")
	}

	return nil
}

// UpdateStatus updates a template's status
func (r *TemplateRepository) UpdateStatus(ctx context.Context, id string, status entity.TemplateStatus, reason string) error {
	query := `
		UPDATE templates SET
			status = $2,
			rejection_reason = $3,
			updated_at = $4
		WHERE id = $1
	`

	result, err := r.db.Pool.Exec(ctx, query, id, string(status), nullString(reason), time.Now())
	if err != nil {
		return errors.Wrap(err, errors.ErrCodeInternal, "failed to update template status")
	}

	if result.RowsAffected() == 0 {
		return errors.New(errors.ErrCodeNotFound, "template not found")
	}

	return nil
}

// UpdateQuality updates a template's quality score
func (r *TemplateRepository) UpdateQuality(ctx context.Context, id string, quality entity.TemplateQuality) error {
	query := `
		UPDATE templates SET
			quality_score = $2,
			updated_at = $3
		WHERE id = $1
	`

	result, err := r.db.Pool.Exec(ctx, query, id, string(quality), time.Now())
	if err != nil {
		return errors.Wrap(err, errors.ErrCodeInternal, "failed to update template quality")
	}

	if result.RowsAffected() == 0 {
		return errors.New(errors.ErrCodeNotFound, "template not found")
	}

	return nil
}

// Delete deletes a template
func (r *TemplateRepository) Delete(ctx context.Context, id string) error {
	query := `DELETE FROM templates WHERE id = $1`

	result, err := r.db.Pool.Exec(ctx, query, id)
	if err != nil {
		return errors.Wrap(err, errors.ErrCodeInternal, "failed to delete template")
	}

	if result.RowsAffected() == 0 {
		return errors.New(errors.ErrCodeNotFound, "template not found")
	}

	return nil
}

// DeleteByExternalID deletes a template by its external (Meta) ID
func (r *TemplateRepository) DeleteByExternalID(ctx context.Context, externalID string) error {
	query := `DELETE FROM templates WHERE external_id = $1`

	result, err := r.db.Pool.Exec(ctx, query, externalID)
	if err != nil {
		return errors.Wrap(err, errors.ErrCodeInternal, "failed to delete template")
	}

	if result.RowsAffected() == 0 {
		return errors.New(errors.ErrCodeNotFound, "template not found")
	}

	return nil
}

// CountByChannel counts templates for a channel
func (r *TemplateRepository) CountByChannel(ctx context.Context, channelID string) (int64, error) {
	query := `SELECT COUNT(*) FROM templates WHERE channel_id = $1`

	var count int64
	if err := r.db.Pool.QueryRow(ctx, query, channelID).Scan(&count); err != nil {
		return 0, errors.Wrap(err, errors.ErrCodeInternal, "failed to count templates")
	}

	return count, nil
}

// UpsertByExternalID creates or updates a template by its external ID
func (r *TemplateRepository) UpsertByExternalID(ctx context.Context, template *entity.Template) error {
	components, err := json.Marshal(template.Components)
	if err != nil {
		return errors.Wrap(err, errors.ErrCodeInternal, "failed to marshal components")
	}

	query := `
		INSERT INTO templates (
			id, tenant_id, channel_id, external_id, name, language, category,
			status, quality_score, components, rejection_reason, last_synced_at,
			created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)
		ON CONFLICT (external_id) WHERE external_id IS NOT NULL
		DO UPDATE SET
			name = EXCLUDED.name,
			language = EXCLUDED.language,
			category = EXCLUDED.category,
			status = EXCLUDED.status,
			quality_score = EXCLUDED.quality_score,
			components = EXCLUDED.components,
			rejection_reason = EXCLUDED.rejection_reason,
			last_synced_at = EXCLUDED.last_synced_at,
			updated_at = EXCLUDED.updated_at
	`

	_, err = r.db.Pool.Exec(ctx, query,
		template.ID,
		template.TenantID,
		template.ChannelID,
		nullString(template.ExternalID),
		template.Name,
		template.Language,
		string(template.Category),
		string(template.Status),
		string(template.QualityScore),
		components,
		nullString(template.RejectionReason),
		template.LastSyncedAt,
		template.CreatedAt,
		template.UpdatedAt,
	)

	if err != nil {
		return errors.Wrap(err, errors.ErrCodeInternal, "failed to upsert template")
	}

	return nil
}

// scanTemplate scans a single template from a row
func (r *TemplateRepository) scanTemplate(row pgx.Row) (*entity.Template, error) {
	var template entity.Template
	var externalID, rejectionReason *string
	var componentsJSON []byte
	var category, status, qualityScore string
	var lastSyncedAt *time.Time

	err := row.Scan(
		&template.ID,
		&template.TenantID,
		&template.ChannelID,
		&externalID,
		&template.Name,
		&template.Language,
		&category,
		&status,
		&qualityScore,
		&componentsJSON,
		&rejectionReason,
		&lastSyncedAt,
		&template.CreatedAt,
		&template.UpdatedAt,
	)

	if err != nil {
		return nil, err
	}

	// Parse nullable fields
	if externalID != nil {
		template.ExternalID = *externalID
	}
	if rejectionReason != nil {
		template.RejectionReason = *rejectionReason
	}
	template.LastSyncedAt = lastSyncedAt

	// Parse enums
	template.Category = entity.TemplateCategory(category)
	template.Status = entity.TemplateStatus(status)
	template.QualityScore = entity.TemplateQuality(qualityScore)

	// Parse components JSON
	if len(componentsJSON) > 0 {
		if err := json.Unmarshal(componentsJSON, &template.Components); err != nil {
			return nil, fmt.Errorf("failed to unmarshal components: %w", err)
		}
	}

	return &template, nil
}

// scanTemplates scans multiple templates from rows
func (r *TemplateRepository) scanTemplates(rows pgx.Rows) ([]*entity.Template, error) {
	var templates []*entity.Template

	for rows.Next() {
		var template entity.Template
		var externalID, rejectionReason *string
		var componentsJSON []byte
		var category, status, qualityScore string
		var lastSyncedAt *time.Time

		err := rows.Scan(
			&template.ID,
			&template.TenantID,
			&template.ChannelID,
			&externalID,
			&template.Name,
			&template.Language,
			&category,
			&status,
			&qualityScore,
			&componentsJSON,
			&rejectionReason,
			&lastSyncedAt,
			&template.CreatedAt,
			&template.UpdatedAt,
		)

		if err != nil {
			return nil, errors.Wrap(err, errors.ErrCodeInternal, "failed to scan template")
		}

		// Parse nullable fields
		if externalID != nil {
			template.ExternalID = *externalID
		}
		if rejectionReason != nil {
			template.RejectionReason = *rejectionReason
		}
		template.LastSyncedAt = lastSyncedAt

		// Parse enums
		template.Category = entity.TemplateCategory(category)
		template.Status = entity.TemplateStatus(status)
		template.QualityScore = entity.TemplateQuality(qualityScore)

		// Parse components JSON
		if len(componentsJSON) > 0 {
			if err := json.Unmarshal(componentsJSON, &template.Components); err != nil {
				return nil, fmt.Errorf("failed to unmarshal components: %w", err)
			}
		}

		templates = append(templates, &template)
	}

	if err := rows.Err(); err != nil {
		return nil, errors.Wrap(err, errors.ErrCodeInternal, "error iterating templates")
	}

	return templates, nil
}
