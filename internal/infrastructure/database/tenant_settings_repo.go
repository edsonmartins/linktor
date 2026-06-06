package database

import (
	"context"
	"encoding/json"

	"github.com/jackc/pgx/v5"
	"github.com/msgfy/linktor/internal/domain/entity"
	"github.com/msgfy/linktor/pkg/errors"
)

// TenantSettingsRepository implements repository.TenantSettingsRepository.
type TenantSettingsRepository struct {
	db *PostgresDB
}

// NewTenantSettingsRepository creates a new PostgreSQL tenant settings repository.
func NewTenantSettingsRepository(db *PostgresDB) *TenantSettingsRepository {
	return &TenantSettingsRepository{db: db}
}

const tenantSettingsColumns = `tenant_id, assignment_strategy, auto_assign_enabled, sla_first_response_minutes, sla_resolution_minutes, auto_close_after_minutes, business_hours, created_at, updated_at`

func (r *TenantSettingsRepository) Get(ctx context.Context, tenantID string) (*entity.TenantSettings, error) {
	query := `SELECT ` + tenantSettingsColumns + ` FROM tenant_settings WHERE tenant_id = $1`
	settings, err := scanTenantSettings(r.db.Pool.QueryRow(ctx, query, tenantID))
	if err != nil {
		if err == pgx.ErrNoRows {
			return entity.DefaultTenantSettings(tenantID), nil
		}
		return nil, errors.Wrap(err, errors.ErrCodeInternal, "failed to get tenant settings")
	}
	return settings, nil
}

func (r *TenantSettingsRepository) Upsert(ctx context.Context, s *entity.TenantSettings) error {
	hours, _ := json.Marshal(s.BusinessHours)
	query := `
		INSERT INTO tenant_settings (tenant_id, assignment_strategy, auto_assign_enabled, sla_first_response_minutes, sla_resolution_minutes, auto_close_after_minutes, business_hours)
		VALUES ($1,$2,$3,$4,$5,$6,$7)
		ON CONFLICT (tenant_id) DO UPDATE SET
			assignment_strategy = EXCLUDED.assignment_strategy,
			auto_assign_enabled = EXCLUDED.auto_assign_enabled,
			sla_first_response_minutes = EXCLUDED.sla_first_response_minutes,
			sla_resolution_minutes = EXCLUDED.sla_resolution_minutes,
			auto_close_after_minutes = EXCLUDED.auto_close_after_minutes,
			business_hours = EXCLUDED.business_hours
	`
	_, err := r.db.Pool.Exec(ctx, query,
		s.TenantID, string(s.AssignmentStrategy), s.AutoAssignEnabled,
		s.SLAFirstResponseMinutes, s.SLAResolutionMinutes, s.AutoCloseAfterMinutes, hours,
	)
	if err != nil {
		return errors.Wrap(err, errors.ErrCodeInternal, "failed to upsert tenant settings")
	}
	return nil
}

func (r *TenantSettingsRepository) ListWithSLA(ctx context.Context) ([]*entity.TenantSettings, error) {
	query := `SELECT ` + tenantSettingsColumns + ` FROM tenant_settings
		WHERE sla_first_response_minutes > 0 OR sla_resolution_minutes > 0 OR auto_close_after_minutes > 0`
	rows, err := r.db.Pool.Query(ctx, query)
	if err != nil {
		return nil, errors.Wrap(err, errors.ErrCodeInternal, "failed to list tenant settings")
	}
	defer rows.Close()

	var list []*entity.TenantSettings
	for rows.Next() {
		s, err := scanTenantSettings(rows)
		if err != nil {
			return nil, err
		}
		list = append(list, s)
	}
	return list, nil
}

func scanTenantSettings(row pgx.Row) (*entity.TenantSettings, error) {
	var s entity.TenantSettings
	var strategy string
	var hours []byte
	err := row.Scan(
		&s.TenantID, &strategy, &s.AutoAssignEnabled,
		&s.SLAFirstResponseMinutes, &s.SLAResolutionMinutes, &s.AutoCloseAfterMinutes,
		&hours, &s.CreatedAt, &s.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	s.AssignmentStrategy = entity.AssignmentStrategy(strategy)
	if len(hours) > 0 {
		_ = json.Unmarshal(hours, &s.BusinessHours)
	}
	if s.BusinessHours == nil {
		s.BusinessHours = map[string]interface{}{}
	}
	return &s, nil
}
