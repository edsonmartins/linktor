package database

import (
	"context"
	stderrors "errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/msgfy/linktor/internal/domain/entity"
	"github.com/msgfy/linktor/pkg/errors"
)

// isUniqueViolation reports whether err is a Postgres unique-constraint
// violation (SQLSTATE 23505).
func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return stderrors.As(err, &pgErr) && pgErr.Code == "23505"
}

// SandboxAllowlistRepository implements repository.SandboxAllowlistRepository
// with PostgreSQL. Every query filters by tenant_id so a cross-tenant lookup
// fails as not-found rather than returning another tenant's entries.
type SandboxAllowlistRepository struct {
	db *PostgresDB
}

// NewSandboxAllowlistRepository creates a new PostgreSQL sandbox allowlist repository.
func NewSandboxAllowlistRepository(db *PostgresDB) *SandboxAllowlistRepository {
	return &SandboxAllowlistRepository{db: db}
}

// Create inserts an allowlist entry. Duplicate (tenant, channel, recipient)
// combinations are rejected by the unique indexes and surfaced as conflicts.
func (r *SandboxAllowlistRepository) Create(ctx context.Context, entry *entity.SandboxAllowlistEntry) error {
	query := `
		INSERT INTO sandbox_allowlist_entries (id, tenant_id, channel_id, recipient, note, created_by, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`
	_, err := r.db.Pool.Exec(ctx, query,
		entry.ID,
		entry.TenantID,
		nullString(entry.ChannelID),
		entry.Recipient,
		nullString(entry.Note),
		nullString(entry.CreatedBy),
		entry.CreatedAt,
	)
	if err != nil {
		if isUniqueViolation(err) {
			return errors.New(errors.ErrCodeConflict, "recipient already in the sandbox allowlist")
		}
		return errors.Wrap(err, errors.ErrCodeInternal, "failed to create sandbox allowlist entry")
	}
	return nil
}

// FindByID returns an entry only if it belongs to the tenant.
func (r *SandboxAllowlistRepository) FindByID(ctx context.Context, tenantID, id string) (*entity.SandboxAllowlistEntry, error) {
	query := `
		SELECT id, tenant_id, channel_id, recipient, note, created_by, created_at
		FROM sandbox_allowlist_entries
		WHERE id = $1 AND tenant_id = $2
	`
	entry, err := scanSandboxAllowlistEntry(r.db.Pool.QueryRow(ctx, query, id, tenantID))
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, errors.New(errors.ErrCodeNotFound, "sandbox allowlist entry not found")
		}
		return nil, errors.Wrap(err, errors.ErrCodeInternal, "failed to find sandbox allowlist entry")
	}
	return entry, nil
}

// FindByTenant lists the tenant's allowlist entries, newest first.
func (r *SandboxAllowlistRepository) FindByTenant(ctx context.Context, tenantID string) ([]*entity.SandboxAllowlistEntry, error) {
	query := `
		SELECT id, tenant_id, channel_id, recipient, note, created_by, created_at
		FROM sandbox_allowlist_entries
		WHERE tenant_id = $1
		ORDER BY created_at DESC
	`
	rows, err := r.db.Pool.Query(ctx, query, tenantID)
	if err != nil {
		return nil, errors.Wrap(err, errors.ErrCodeInternal, "failed to query sandbox allowlist")
	}
	defer rows.Close()

	var entries []*entity.SandboxAllowlistEntry
	for rows.Next() {
		entry, err := scanSandboxAllowlistEntry(rows)
		if err != nil {
			return nil, errors.Wrap(err, errors.ErrCodeInternal, "failed to scan sandbox allowlist entry")
		}
		entries = append(entries, entry)
	}
	return entries, nil
}

// Delete removes an entry only if it belongs to the tenant. A cross-tenant id
// is indistinguishable from a missing one: both return not-found.
func (r *SandboxAllowlistRepository) Delete(ctx context.Context, tenantID, id string) error {
	result, err := r.db.Pool.Exec(ctx,
		`DELETE FROM sandbox_allowlist_entries WHERE id = $1 AND tenant_id = $2`, id, tenantID)
	if err != nil {
		return errors.Wrap(err, errors.ErrCodeInternal, "failed to delete sandbox allowlist entry")
	}
	if result.RowsAffected() == 0 {
		return errors.New(errors.ErrCodeNotFound, "sandbox allowlist entry not found")
	}
	return nil
}

// IsAllowed reports whether the normalized recipient is authorized for the
// given channel, via a tenant-wide entry or one scoped to that channel. It hits
// the database on every call by design (INV-017): the allowlist must be
// consulted at send time, never captured or cached by the guard.
func (r *SandboxAllowlistRepository) IsAllowed(ctx context.Context, tenantID, channelID, recipient string) (bool, error) {
	query := `
		SELECT EXISTS (
			SELECT 1 FROM sandbox_allowlist_entries
			WHERE tenant_id = $1
			  AND recipient = $2
			  AND (channel_id IS NULL OR channel_id = $3)
		)
	`
	var allowed bool
	if err := r.db.Pool.QueryRow(ctx, query, tenantID, recipient, channelID).Scan(&allowed); err != nil {
		return false, errors.Wrap(err, errors.ErrCodeInternal, "failed to check sandbox allowlist")
	}
	return allowed, nil
}

func scanSandboxAllowlistEntry(row pgx.Row) (*entity.SandboxAllowlistEntry, error) {
	var e entity.SandboxAllowlistEntry
	var channelID, note, createdBy *string
	if err := row.Scan(&e.ID, &e.TenantID, &channelID, &e.Recipient, &note, &createdBy, &e.CreatedAt); err != nil {
		return nil, err
	}
	if channelID != nil {
		e.ChannelID = *channelID
	}
	if note != nil {
		e.Note = *note
	}
	if createdBy != nil {
		e.CreatedBy = *createdBy
	}
	return &e, nil
}
