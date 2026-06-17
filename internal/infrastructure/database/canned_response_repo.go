package database

import (
	"context"
	"encoding/json"

	"github.com/jackc/pgx/v5"
	"github.com/msgfy/linktor/internal/domain/entity"
	"github.com/msgfy/linktor/internal/domain/repository"
	"github.com/msgfy/linktor/pkg/errors"
)

// CannedResponseRepository implements repository.CannedResponseRepository.
type CannedResponseRepository struct {
	db *PostgresDB
}

// NewCannedResponseRepository creates a new PostgreSQL canned response repository.
func NewCannedResponseRepository(db *PostgresDB) *CannedResponseRepository {
	return &CannedResponseRepository{db: db}
}

const cannedColumns = `id, tenant_id, shortcut, title, content, tags, usage_count, created_by, created_at, updated_at`

func (r *CannedResponseRepository) Create(ctx context.Context, cr *entity.CannedResponse) error {
	tags, _ := json.Marshal(cr.Tags)
	query := `INSERT INTO canned_responses (` + cannedColumns + `) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)`
	_, err := r.db.Pool.Exec(ctx, query,
		cr.ID, cr.TenantID, cr.Shortcut, cr.Title, cr.Content, tags,
		cr.UsageCount, nullString(cr.CreatedBy), cr.CreatedAt, cr.UpdatedAt,
	)
	if err != nil {
		return errors.Wrap(err, errors.ErrCodeInternal, "failed to create canned response")
	}
	return nil
}

func (r *CannedResponseRepository) FindByID(ctx context.Context, id string) (*entity.CannedResponse, error) {
	query := `SELECT ` + cannedColumns + ` FROM canned_responses WHERE id = $1`
	cr, err := scanCanned(r.db.Pool.QueryRow(ctx, query, id))
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, errors.New(errors.ErrCodeNotFound, "canned response not found")
		}
		return nil, errors.Wrap(err, errors.ErrCodeInternal, "failed to find canned response")
	}
	return cr, nil
}

func (r *CannedResponseRepository) FindByShortcut(ctx context.Context, tenantID, shortcut string) (*entity.CannedResponse, error) {
	query := `SELECT ` + cannedColumns + ` FROM canned_responses WHERE tenant_id = $1 AND shortcut = $2`
	cr, err := scanCanned(r.db.Pool.QueryRow(ctx, query, tenantID, shortcut))
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, errors.New(errors.ErrCodeNotFound, "canned response not found")
		}
		return nil, errors.Wrap(err, errors.ErrCodeInternal, "failed to find canned response")
	}
	return cr, nil
}

func (r *CannedResponseRepository) FindByTenant(ctx context.Context, tenantID, search string, params *repository.ListParams) ([]*entity.CannedResponse, int64, error) {
	if params == nil {
		params = repository.NewListParams()
	}

	where := "WHERE tenant_id = $1"
	args := []interface{}{tenantID}
	if search != "" {
		where += " AND (shortcut ILIKE $2 OR title ILIKE $2 OR content ILIKE $2)"
		args = append(args, "%"+search+"%")
	}

	var total int64
	if err := r.db.Pool.QueryRow(ctx, "SELECT COUNT(*) FROM canned_responses "+where, args...).Scan(&total); err != nil {
		return nil, 0, errors.Wrap(err, errors.ErrCodeInternal, "failed to count canned responses")
	}

	limitIdx := len(args) + 1
	query := "SELECT " + cannedColumns + " FROM canned_responses " + where +
		" ORDER BY usage_count DESC, shortcut ASC LIMIT $" + itoa(limitIdx) + " OFFSET $" + itoa(limitIdx+1)
	args = append(args, params.Limit(), params.Offset())

	rows, err := r.db.Pool.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, errors.Wrap(err, errors.ErrCodeInternal, "failed to query canned responses")
	}
	defer rows.Close()

	var list []*entity.CannedResponse
	for rows.Next() {
		cr, err := scanCanned(rows)
		if err != nil {
			return nil, 0, err
		}
		list = append(list, cr)
	}
	return list, total, nil
}

func (r *CannedResponseRepository) Update(ctx context.Context, cr *entity.CannedResponse) error {
	tags, _ := json.Marshal(cr.Tags)
	query := `UPDATE canned_responses SET shortcut=$1, title=$2, content=$3, tags=$4 WHERE id=$5`
	result, err := r.db.Pool.Exec(ctx, query, cr.Shortcut, cr.Title, cr.Content, tags, cr.ID)
	if err != nil {
		return errors.Wrap(err, errors.ErrCodeInternal, "failed to update canned response")
	}
	if result.RowsAffected() == 0 {
		return errors.New(errors.ErrCodeNotFound, "canned response not found")
	}
	return nil
}

func (r *CannedResponseRepository) Delete(ctx context.Context, id string) error {
	result, err := r.db.Pool.Exec(ctx, "DELETE FROM canned_responses WHERE id = $1", id)
	if err != nil {
		return errors.Wrap(err, errors.ErrCodeInternal, "failed to delete canned response")
	}
	if result.RowsAffected() == 0 {
		return errors.New(errors.ErrCodeNotFound, "canned response not found")
	}
	return nil
}

func (r *CannedResponseRepository) IncrementUsage(ctx context.Context, id string) error {
	_, err := r.db.Pool.Exec(ctx, "UPDATE canned_responses SET usage_count = usage_count + 1 WHERE id = $1", id)
	if err != nil {
		return errors.Wrap(err, errors.ErrCodeInternal, "failed to increment usage")
	}
	return nil
}

func scanCanned(row pgx.Row) (*entity.CannedResponse, error) {
	var cr entity.CannedResponse
	var tags []byte
	var createdBy *string
	err := row.Scan(&cr.ID, &cr.TenantID, &cr.Shortcut, &cr.Title, &cr.Content, &tags, &cr.UsageCount, &createdBy, &cr.CreatedAt, &cr.UpdatedAt)
	if err != nil {
		return nil, err
	}
	cr.CreatedBy = derefString(createdBy)
	if len(tags) > 0 {
		_ = json.Unmarshal(tags, &cr.Tags)
	}
	if cr.Tags == nil {
		cr.Tags = []string{}
	}
	return &cr, nil
}
