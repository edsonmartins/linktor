package database

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/msgfy/linktor/internal/domain/entity"
	"github.com/msgfy/linktor/pkg/errors"
)

// APIKeyRepository implements API key persistence with PostgreSQL.
type APIKeyRepository struct {
	db *PostgresDB
}

// NewAPIKeyRepository creates a new PostgreSQL API key repository.
func NewAPIKeyRepository(db *PostgresDB) *APIKeyRepository {
	return &APIKeyRepository{db: db}
}

// Create stores a new API key.
func (r *APIKeyRepository) Create(ctx context.Context, apiKey *entity.APIKey) error {
	query := `
		INSERT INTO api_keys (
			id, tenant_id, user_id, name, key_hash, key_prefix, scopes,
			last_used_at, expires_at, created_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
	`

	_, err := r.db.Pool.Exec(ctx, query,
		apiKey.ID,
		apiKey.TenantID,
		apiKey.UserID,
		apiKey.Name,
		apiKey.KeyHash,
		apiKey.KeyPrefix,
		apiKey.Scopes,
		apiKey.LastUsedAt,
		apiKey.ExpiresAt,
		apiKey.CreatedAt,
	)
	if err != nil {
		return errors.Wrap(err, errors.ErrCodeInternal, "failed to create API key")
	}
	return nil
}

// ListByTenant returns API key metadata for a tenant.
func (r *APIKeyRepository) ListByTenant(ctx context.Context, tenantID string) ([]*entity.APIKey, error) {
	query := `
		SELECT id, tenant_id, user_id, name, key_hash, key_prefix, scopes,
		       last_used_at, expires_at, created_at
		FROM api_keys
		WHERE tenant_id = $1
		ORDER BY created_at DESC
	`

	rows, err := r.db.Pool.Query(ctx, query, tenantID)
	if err != nil {
		return nil, errors.Wrap(err, errors.ErrCodeInternal, "failed to list API keys")
	}
	defer rows.Close()

	var apiKeys []*entity.APIKey
	for rows.Next() {
		apiKey, err := scanAPIKey(rows)
		if err != nil {
			return nil, errors.Wrap(err, errors.ErrCodeInternal, "failed to scan API key")
		}
		apiKey.KeyHash = ""
		apiKeys = append(apiKeys, apiKey)
	}
	if err := rows.Err(); err != nil {
		return nil, errors.Wrap(err, errors.ErrCodeInternal, "failed to iterate API keys")
	}

	return apiKeys, nil
}

// Delete removes an API key by tenant and ID.
func (r *APIKeyRepository) Delete(ctx context.Context, tenantID, id string) error {
	result, err := r.db.Pool.Exec(ctx, "DELETE FROM api_keys WHERE tenant_id = $1 AND id = $2", tenantID, id)
	if err != nil {
		return errors.Wrap(err, errors.ErrCodeInternal, "failed to delete API key")
	}
	if result.RowsAffected() == 0 {
		return errors.New(errors.ErrCodeNotFound, "API key not found")
	}
	return nil
}

func scanAPIKey(rows pgx.Rows) (*entity.APIKey, error) {
	var apiKey entity.APIKey
	err := rows.Scan(
		&apiKey.ID,
		&apiKey.TenantID,
		&apiKey.UserID,
		&apiKey.Name,
		&apiKey.KeyHash,
		&apiKey.KeyPrefix,
		&apiKey.Scopes,
		&apiKey.LastUsedAt,
		&apiKey.ExpiresAt,
		&apiKey.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &apiKey, nil
}
