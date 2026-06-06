package database

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/msgfy/linktor/internal/domain/entity"
	"github.com/msgfy/linktor/pkg/errors"
)

// RoleRepository implements repository.RoleRepository with PostgreSQL.
type RoleRepository struct {
	db *PostgresDB
}

// NewRoleRepository creates a new PostgreSQL role repository.
func NewRoleRepository(db *PostgresDB) *RoleRepository {
	return &RoleRepository{db: db}
}

func (r *RoleRepository) Create(ctx context.Context, role *entity.Role) error {
	tx, err := r.db.Pool.Begin(ctx)
	if err != nil {
		return errors.Wrap(err, errors.ErrCodeInternal, "failed to begin tx")
	}
	defer tx.Rollback(ctx)

	_, err = tx.Exec(ctx,
		`INSERT INTO roles (id, tenant_id, name, description, is_system, is_default, created_at, updated_at)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8)`,
		role.ID, role.TenantID, role.Name, nullString(role.Description),
		role.IsSystem, role.IsDefault, role.CreatedAt, role.UpdatedAt,
	)
	if err != nil {
		return errors.Wrap(err, errors.ErrCodeInternal, "failed to insert role")
	}

	if err := insertRolePermissions(ctx, tx, role.ID, role.Permissions); err != nil {
		return err
	}

	if err := tx.Commit(ctx); err != nil {
		return errors.Wrap(err, errors.ErrCodeInternal, "failed to commit role")
	}
	return nil
}

func (r *RoleRepository) FindByID(ctx context.Context, id string) (*entity.Role, error) {
	role, err := scanRole(r.db.Pool.QueryRow(ctx, roleSelect+` WHERE id = $1`, id))
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, errors.New(errors.ErrCodeNotFound, "role not found")
		}
		return nil, errors.Wrap(err, errors.ErrCodeInternal, "failed to find role")
	}
	if err := r.loadPermissions(ctx, role); err != nil {
		return nil, err
	}
	return role, nil
}

func (r *RoleRepository) FindByName(ctx context.Context, tenantID, name string) (*entity.Role, error) {
	role, err := scanRole(r.db.Pool.QueryRow(ctx, roleSelect+` WHERE tenant_id = $1 AND name = $2`, tenantID, name))
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, errors.New(errors.ErrCodeNotFound, "role not found")
		}
		return nil, errors.Wrap(err, errors.ErrCodeInternal, "failed to find role")
	}
	if err := r.loadPermissions(ctx, role); err != nil {
		return nil, err
	}
	return role, nil
}

func (r *RoleRepository) FindByTenant(ctx context.Context, tenantID string) ([]*entity.Role, error) {
	rows, err := r.db.Pool.Query(ctx, roleSelect+` WHERE tenant_id = $1 ORDER BY is_system DESC, name ASC`, tenantID)
	if err != nil {
		return nil, errors.Wrap(err, errors.ErrCodeInternal, "failed to query roles")
	}
	defer rows.Close()

	var roles []*entity.Role
	for rows.Next() {
		role, err := scanRole(rows)
		if err != nil {
			return nil, err
		}
		roles = append(roles, role)
	}
	rows.Close()

	for _, role := range roles {
		if err := r.loadPermissions(ctx, role); err != nil {
			return nil, err
		}
	}
	return roles, nil
}

func (r *RoleRepository) Update(ctx context.Context, role *entity.Role) error {
	tx, err := r.db.Pool.Begin(ctx)
	if err != nil {
		return errors.Wrap(err, errors.ErrCodeInternal, "failed to begin tx")
	}
	defer tx.Rollback(ctx)

	result, err := tx.Exec(ctx,
		`UPDATE roles SET name=$1, description=$2, is_default=$3 WHERE id=$4`,
		role.Name, nullString(role.Description), role.IsDefault, role.ID,
	)
	if err != nil {
		return errors.Wrap(err, errors.ErrCodeInternal, "failed to update role")
	}
	if result.RowsAffected() == 0 {
		return errors.New(errors.ErrCodeNotFound, "role not found")
	}

	if _, err := tx.Exec(ctx, `DELETE FROM role_permissions WHERE role_id = $1`, role.ID); err != nil {
		return errors.Wrap(err, errors.ErrCodeInternal, "failed to clear permissions")
	}
	if err := insertRolePermissions(ctx, tx, role.ID, role.Permissions); err != nil {
		return err
	}

	if err := tx.Commit(ctx); err != nil {
		return errors.Wrap(err, errors.ErrCodeInternal, "failed to commit role update")
	}
	return nil
}

func (r *RoleRepository) Delete(ctx context.Context, id string) error {
	result, err := r.db.Pool.Exec(ctx, `DELETE FROM roles WHERE id = $1 AND is_system = false`, id)
	if err != nil {
		return errors.Wrap(err, errors.ErrCodeInternal, "failed to delete role")
	}
	if result.RowsAffected() == 0 {
		return errors.New(errors.ErrCodeNotFound, "role not found or is a system role")
	}
	return nil
}

const roleSelect = `SELECT id, tenant_id, name, description, is_system, is_default, created_at, updated_at FROM roles`

func scanRole(row pgx.Row) (*entity.Role, error) {
	var role entity.Role
	var description *string
	if err := row.Scan(&role.ID, &role.TenantID, &role.Name, &description, &role.IsSystem, &role.IsDefault, &role.CreatedAt, &role.UpdatedAt); err != nil {
		return nil, err
	}
	role.Description = derefString(description)
	role.Permissions = []entity.Permission{}
	return &role, nil
}

func (r *RoleRepository) loadPermissions(ctx context.Context, role *entity.Role) error {
	rows, err := r.db.Pool.Query(ctx, `SELECT resource, action FROM role_permissions WHERE role_id = $1`, role.ID)
	if err != nil {
		return errors.Wrap(err, errors.ErrCodeInternal, "failed to load permissions")
	}
	defer rows.Close()

	for rows.Next() {
		var p entity.Permission
		if err := rows.Scan(&p.Resource, &p.Action); err != nil {
			return errors.Wrap(err, errors.ErrCodeInternal, "failed to scan permission")
		}
		role.Permissions = append(role.Permissions, p)
	}
	return nil
}

func insertRolePermissions(ctx context.Context, tx pgx.Tx, roleID string, perms []entity.Permission) error {
	for _, p := range perms {
		if _, err := tx.Exec(ctx,
			`INSERT INTO role_permissions (role_id, resource, action) VALUES ($1,$2,$3)
			 ON CONFLICT DO NOTHING`,
			roleID, p.Resource, p.Action,
		); err != nil {
			return errors.Wrap(err, errors.ErrCodeInternal, "failed to insert permission")
		}
	}
	return nil
}
