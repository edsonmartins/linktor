package service

import (
	"context"
	"fmt"
	"testing"

	"github.com/msgfy/linktor/internal/domain/entity"
	"github.com/msgfy/linktor/pkg/testutil"
)

func TestRoleHasPermissionOwnerShortcut(t *testing.T) {
	svc := NewRoleService(testutil.NewMockRoleRepository(), nil)
	if !svc.HasPermission(context.Background(), "t1", "owner", entity.ResourceUsers, entity.ActionDelete) {
		t.Fatal("owner must be allowed everything")
	}
}

func TestRoleHasPermissionFallbackToDefaults(t *testing.T) {
	// Empty repo: resolution must fall back to built-in defaults for the name.
	svc := NewRoleService(testutil.NewMockRoleRepository(), nil)
	ctx := context.Background()

	if !svc.HasPermission(ctx, "t1", "agent", entity.ResourceConversations, entity.ActionRead) {
		t.Fatal("agent should read conversations via defaults")
	}
	if svc.HasPermission(ctx, "t1", "agent", entity.ResourceUsers, entity.ActionCreate) {
		t.Fatal("agent must not create users")
	}
	if !svc.HasPermission(ctx, "t1", "admin", entity.ResourceCampaigns, entity.ActionCreate) {
		t.Fatal("admin should manage campaigns via defaults")
	}
}

func TestRoleEnsureSystemRolesSeeds(t *testing.T) {
	repo := testutil.NewMockRoleRepository()
	svc := NewRoleService(repo, nil)
	if err := svc.EnsureSystemRoles(context.Background(), "t1"); err != nil {
		t.Fatalf("EnsureSystemRoles: %v", err)
	}
	if len(repo.Roles) != len(entity.SystemRoleNames()) {
		t.Fatalf("expected %d system roles, got %d", len(entity.SystemRoleNames()), len(repo.Roles))
	}
	// Idempotent: a second call must not create duplicates.
	if err := svc.EnsureSystemRoles(context.Background(), "t1"); err != nil {
		t.Fatalf("second EnsureSystemRoles: %v", err)
	}
	if len(repo.Roles) != len(entity.SystemRoleNames()) {
		t.Fatalf("EnsureSystemRoles not idempotent: %d roles", len(repo.Roles))
	}
}

func TestRoleEnsureSystemRolesPropagatesTransientError(t *testing.T) {
	repo := testutil.NewMockRoleRepository()
	// A non-NotFound error from the probe must NOT be treated as "missing".
	repo.FindByNameErr = fmt.Errorf("db connection lost")
	svc := NewRoleService(repo, nil)

	if err := svc.EnsureSystemRoles(context.Background(), "t1"); err == nil {
		t.Fatal("expected transient error to propagate, not a duplicate Create")
	}
	if len(repo.Roles) != 0 {
		t.Fatalf("must not create roles on a transient probe error, got %d", len(repo.Roles))
	}
}
