package database

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/msgfy/linktor/internal/domain/entity"
	"github.com/msgfy/linktor/internal/domain/repository"
	"github.com/msgfy/linktor/pkg/errors"
)

// UserRepository implements repository.UserRepository with PostgreSQL
type UserRepository struct {
	db *PostgresDB
}

// NewUserRepository creates a new PostgreSQL user repository
func NewUserRepository(db *PostgresDB) *UserRepository {
	return &UserRepository{db: db}
}

// Create creates a new user
func (r *UserRepository) Create(ctx context.Context, user *entity.User) error {
	query := `
		INSERT INTO users (
			id, tenant_id, email, password_hash, name, role, avatar_url,
			status, last_login_at, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
	`

	_, err := r.db.Pool.Exec(ctx, query,
		user.ID,
		user.TenantID,
		user.Email,
		user.PasswordHash,
		user.Name,
		string(user.Role),
		user.AvatarURL,
		string(user.Status),
		user.LastLoginAt,
		user.CreatedAt,
		user.UpdatedAt,
	)

	if err != nil {
		return errors.Wrap(err, errors.ErrCodeInternal, "failed to create user")
	}

	return nil
}

// FindByID finds a user by ID
func (r *UserRepository) FindByID(ctx context.Context, id string) (*entity.User, error) {
	query := `
		SELECT id, tenant_id, email, password_hash, name, role, avatar_url,
		       status, last_login_at, created_at, updated_at
		FROM users
		WHERE id = $1
	`

	user, err := r.scanUser(r.db.Pool.QueryRow(ctx, query, id))
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, errors.New(errors.ErrCodeUserNotFound, "user not found")
		}
		return nil, errors.Wrap(err, errors.ErrCodeInternal, "failed to find user")
	}

	return user, nil
}

// FindByEmail finds a user by email (across all tenants)
func (r *UserRepository) FindByEmail(ctx context.Context, email string) (*entity.User, error) {
	query := `
		SELECT id, tenant_id, email, password_hash, name, role, avatar_url,
		       status, last_login_at, created_at, updated_at
		FROM users
		WHERE email = $1
	`

	user, err := r.scanUser(r.db.Pool.QueryRow(ctx, query, email))
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, errors.New(errors.ErrCodeUserNotFound, "user not found")
		}
		return nil, errors.Wrap(err, errors.ErrCodeInternal, "failed to find user")
	}

	return user, nil
}

// FindByTenantAndEmail finds a user by tenant ID and email
func (r *UserRepository) FindByTenantAndEmail(ctx context.Context, tenantID, email string) (*entity.User, error) {
	query := `
		SELECT id, tenant_id, email, password_hash, name, role, avatar_url,
		       status, last_login_at, created_at, updated_at
		FROM users
		WHERE tenant_id = $1 AND email = $2
	`

	user, err := r.scanUser(r.db.Pool.QueryRow(ctx, query, tenantID, email))
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, errors.New(errors.ErrCodeUserNotFound, "user not found")
		}
		return nil, errors.Wrap(err, errors.ErrCodeInternal, "failed to find user")
	}

	return user, nil
}

// FindByTenant finds users for a tenant with pagination
func (r *UserRepository) FindByTenant(ctx context.Context, tenantID string, params *repository.ListParams) ([]*entity.User, int64, error) {
	// Count total
	var total int64
	if err := r.db.Pool.QueryRow(ctx,
		"SELECT COUNT(*) FROM users WHERE tenant_id = $1",
		tenantID,
	).Scan(&total); err != nil {
		return nil, 0, errors.Wrap(err, errors.ErrCodeInternal, "failed to count users")
	}

	// Get users
	query := fmt.Sprintf(`
		SELECT id, tenant_id, email, password_hash, name, role, avatar_url,
		       status, last_login_at, created_at, updated_at
		FROM users
		WHERE tenant_id = $1
		ORDER BY %s %s
		LIMIT $2 OFFSET $3
	`, sanitizeUserColumn(params.SortBy), sanitizeDirection(params.SortDir))

	rows, err := r.db.Pool.Query(ctx, query, tenantID, params.Limit(), params.Offset())
	if err != nil {
		return nil, 0, errors.Wrap(err, errors.ErrCodeInternal, "failed to query users")
	}
	defer rows.Close()

	var users []*entity.User
	for rows.Next() {
		user, err := r.scanUserFromRows(rows)
		if err != nil {
			return nil, 0, err
		}
		users = append(users, user)
	}

	return users, total, nil
}

// Update updates a user
func (r *UserRepository) Update(ctx context.Context, user *entity.User) error {
	user.UpdatedAt = time.Now()

	query := `
		UPDATE users SET
			email = $1,
			password_hash = $2,
			name = $3,
			role = $4,
			avatar_url = $5,
			status = $6,
			last_login_at = $7,
			updated_at = $8
		WHERE id = $9
	`

	result, err := r.db.Pool.Exec(ctx, query,
		user.Email,
		user.PasswordHash,
		user.Name,
		string(user.Role),
		user.AvatarURL,
		string(user.Status),
		user.LastLoginAt,
		user.UpdatedAt,
		user.ID,
	)

	if err != nil {
		return errors.Wrap(err, errors.ErrCodeInternal, "failed to update user")
	}

	if result.RowsAffected() == 0 {
		return errors.New(errors.ErrCodeUserNotFound, "user not found")
	}

	return nil
}

// Delete deletes a user
func (r *UserRepository) Delete(ctx context.Context, id string) error {
	result, err := r.db.Pool.Exec(ctx, "DELETE FROM users WHERE id = $1", id)
	if err != nil {
		return errors.Wrap(err, errors.ErrCodeInternal, "failed to delete user")
	}

	if result.RowsAffected() == 0 {
		return errors.New(errors.ErrCodeUserNotFound, "user not found")
	}

	return nil
}

// CountByTenant counts users for a tenant
func (r *UserRepository) CountByTenant(ctx context.Context, tenantID string) (int64, error) {
	var count int64
	err := r.db.Pool.QueryRow(ctx,
		"SELECT COUNT(*) FROM users WHERE tenant_id = $1",
		tenantID,
	).Scan(&count)

	if err != nil {
		return 0, errors.Wrap(err, errors.ErrCodeInternal, "failed to count users")
	}

	return count, nil
}

// Helper methods

func (r *UserRepository) scanUser(row pgx.Row) (*entity.User, error) {
	var u entity.User
	var role, status string
	var avatarURL *string

	err := row.Scan(
		&u.ID, &u.TenantID, &u.Email, &u.PasswordHash, &u.Name, &role, &avatarURL,
		&status, &u.LastLoginAt, &u.CreatedAt, &u.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}

	u.Role = entity.UserRole(role)
	u.Status = entity.UserStatus(status)
	u.AvatarURL = avatarURL

	return &u, nil
}

func (r *UserRepository) scanUserFromRows(rows pgx.Rows) (*entity.User, error) {
	var u entity.User
	var role, status string
	var avatarURL *string

	err := rows.Scan(
		&u.ID, &u.TenantID, &u.Email, &u.PasswordHash, &u.Name, &role, &avatarURL,
		&status, &u.LastLoginAt, &u.CreatedAt, &u.UpdatedAt,
	)
	if err != nil {
		return nil, errors.Wrap(err, errors.ErrCodeInternal, "failed to scan user")
	}

	u.Role = entity.UserRole(role)
	u.Status = entity.UserStatus(status)
	u.AvatarURL = avatarURL

	return &u, nil
}

func sanitizeUserColumn(col string) string {
	allowed := map[string]bool{
		"created_at": true,
		"updated_at": true,
		"email":      true,
		"name":       true,
		"role":       true,
		"status":     true,
	}
	if allowed[col] {
		return col
	}
	return "created_at"
}

// FindAvailableAgents finds available agents for a channel
func (r *UserRepository) FindAvailableAgents(ctx context.Context, tenantID, channelID string) ([]*entity.User, error) {
	// Find agents that are active and part of this tenant
	// In a full implementation, this would also check:
	// - Agent's assigned channels
	// - Agent's working hours
	// - Agent's current workload
	// - Agent's online status
	query := `
		SELECT id, tenant_id, email, password_hash, name, role, avatar_url,
		       status, last_login_at, created_at, updated_at
		FROM users
		WHERE tenant_id = $1
		  AND status = 'active'
		  AND role IN ('agent', 'admin')
		ORDER BY created_at ASC
	`

	rows, err := r.db.Pool.Query(ctx, query, tenantID)
	if err != nil {
		return nil, errors.Wrap(err, errors.ErrCodeInternal, "failed to query available agents")
	}
	defer rows.Close()

	var users []*entity.User
	for rows.Next() {
		user, err := r.scanUserFromRows(rows)
		if err != nil {
			return nil, err
		}
		users = append(users, user)
	}

	return users, nil
}
