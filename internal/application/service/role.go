package service

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/google/uuid"
	"github.com/msgfy/linktor/internal/domain/entity"
	"github.com/msgfy/linktor/internal/domain/repository"
	"github.com/msgfy/linktor/pkg/errors"
)

const permCacheTTL = 5 * time.Minute

// RoleService manages custom RBAC roles and resolves permissions, with an
// optional Redis cache. It is backward compatible: a user whose role string has
// no matching DB role falls back to the built-in defaults for that name.
type RoleService struct {
	repo  repository.RoleRepository
	redis *redis.Client // optional
}

// NewRoleService creates a new RoleService. redis may be nil (no caching).
func NewRoleService(repo repository.RoleRepository, redisClient *redis.Client) *RoleService {
	return &RoleService{repo: repo, redis: redisClient}
}

// EnsureSystemRoles seeds the built-in roles for a tenant if they are missing.
// Idempotent and safe to call on every login/list.
func (s *RoleService) EnsureSystemRoles(ctx context.Context, tenantID string) error {
	for _, name := range entity.SystemRoleNames() {
		_, err := s.repo.FindByName(ctx, tenantID, name)
		if err == nil {
			continue // already present
		}
		if !errors.IsNotFound(err) {
			// Transient/other DB error: don't risk a duplicate Create that would
			// violate UNIQUE(tenant_id, name). Surface it instead.
			return err
		}
		role := entity.NewRole(tenantID, name, "Built-in "+name+" role")
		role.ID = uuid.New().String()
		role.IsSystem = true
		role.IsDefault = name == entity.SystemRoleAgent
		role.Permissions = entity.SystemRolePermissions(name)
		if err := s.repo.Create(ctx, role); err != nil {
			return err
		}
	}
	return nil
}

// List returns all roles for a tenant, seeding system roles first.
func (s *RoleService) List(ctx context.Context, tenantID string) ([]*entity.Role, error) {
	if err := s.EnsureSystemRoles(ctx, tenantID); err != nil {
		return nil, err
	}
	return s.repo.FindByTenant(ctx, tenantID)
}

// Get returns a role scoped to the tenant.
func (s *RoleService) Get(ctx context.Context, tenantID, id string) (*entity.Role, error) {
	role, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if role.TenantID != tenantID {
		return nil, errors.New(errors.ErrCodeNotFound, "role not found")
	}
	return role, nil
}

// Create adds a custom role.
func (s *RoleService) Create(ctx context.Context, tenantID, name, description string, perms []entity.Permission) (*entity.Role, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, errors.Validation("name is required")
	}
	if _, err := s.repo.FindByName(ctx, tenantID, name); err == nil {
		return nil, errors.New(errors.ErrCodeConflict, "a role with this name already exists")
	}
	role := entity.NewRole(tenantID, name, description)
	role.ID = uuid.New().String()
	role.Permissions = perms
	if err := s.repo.Create(ctx, role); err != nil {
		return nil, err
	}
	return role, nil
}

// Update replaces a custom role's name/description/permissions. System roles
// cannot be edited.
func (s *RoleService) Update(ctx context.Context, tenantID, id, name, description string, perms []entity.Permission) (*entity.Role, error) {
	role, err := s.Get(ctx, tenantID, id)
	if err != nil {
		return nil, err
	}
	if role.IsSystem {
		return nil, errors.Forbidden("system roles cannot be modified")
	}
	if name != "" {
		role.Name = name
	}
	role.Description = description
	if perms != nil {
		role.Permissions = perms
	}
	if err := s.repo.Update(ctx, role); err != nil {
		return nil, err
	}
	s.invalidate(ctx, tenantID, role.Name)
	return role, nil
}

// Delete removes a custom role.
func (s *RoleService) Delete(ctx context.Context, tenantID, id string) error {
	role, err := s.Get(ctx, tenantID, id)
	if err != nil {
		return err
	}
	if err := s.repo.Delete(ctx, id); err != nil {
		return err
	}
	s.invalidate(ctx, tenantID, role.Name)
	return nil
}

// HasPermission reports whether a role (by name) grants (resource, action)
// within a tenant. Resolution order: owner shortcut → cache → DB role → built-in
// defaults for the name.
func (s *RoleService) HasPermission(ctx context.Context, tenantID, roleName, resource, action string) bool {
	if roleName == entity.SystemRoleOwner {
		return true
	}

	perms, ok := s.cachedPermissions(ctx, tenantID, roleName)
	if !ok {
		perms = s.resolvePermissions(ctx, tenantID, roleName)
		s.cachePermissions(ctx, tenantID, roleName, perms)
	}

	probe := entity.Role{Permissions: perms}
	return probe.Has(resource, action)
}

func (s *RoleService) resolvePermissions(ctx context.Context, tenantID, roleName string) []entity.Permission {
	if role, err := s.repo.FindByName(ctx, tenantID, roleName); err == nil {
		return role.Permissions
	}
	// Fallback for tenants not yet seeded, or legacy role strings.
	return entity.SystemRolePermissions(roleName)
}

func (s *RoleService) cacheKey(tenantID, roleName string) string {
	return "perm:" + tenantID + ":" + roleName
}

func (s *RoleService) cachedPermissions(ctx context.Context, tenantID, roleName string) ([]entity.Permission, bool) {
	if s.redis == nil {
		return nil, false
	}
	raw, err := s.redis.Get(ctx, s.cacheKey(tenantID, roleName)).Result()
	if err != nil {
		return nil, false
	}
	var perms []entity.Permission
	if json.Unmarshal([]byte(raw), &perms) != nil {
		return nil, false
	}
	return perms, true
}

func (s *RoleService) cachePermissions(ctx context.Context, tenantID, roleName string, perms []entity.Permission) {
	if s.redis == nil {
		return
	}
	raw, err := json.Marshal(perms)
	if err != nil {
		return
	}
	s.redis.Set(ctx, s.cacheKey(tenantID, roleName), raw, permCacheTTL)
}

func (s *RoleService) invalidate(ctx context.Context, tenantID, roleName string) {
	if s.redis == nil {
		return
	}
	s.redis.Del(ctx, s.cacheKey(tenantID, roleName))
}
