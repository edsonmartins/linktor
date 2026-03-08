package service

import (
	"context"
	"testing"

	"github.com/msgfy/linktor/internal/domain/entity"
	"github.com/msgfy/linktor/pkg/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupUserService() (*UserService, *testutil.MockUserRepository, *testutil.MockTenantRepository) {
	userRepo := testutil.NewMockUserRepository()
	tenantRepo := testutil.NewMockTenantRepository()

	// Add a default tenant
	tenantRepo.Tenants["tenant-1"] = &entity.Tenant{
		ID:       "tenant-1",
		Name:     "Test Tenant",
		Plan:     entity.PlanProfessional,
		Settings: map[string]string{},
	}

	svc := NewUserService(userRepo, tenantRepo)
	return svc, userRepo, tenantRepo
}

func TestUserService_Create(t *testing.T) {
	svc, _, _ := setupUserService()

	user, err := svc.Create(context.Background(), &CreateUserInput{
		TenantID: "tenant-1",
		Email:    "user@example.com",
		Password: "password123",
		Name:     "Test User",
		Role:     entity.UserRoleAgent,
	})

	require.NoError(t, err)
	require.NotNil(t, user)
	assert.NotEmpty(t, user.ID)
	assert.Equal(t, "tenant-1", user.TenantID)
	assert.Equal(t, "user@example.com", user.Email)
	assert.Equal(t, "Test User", user.Name)
	assert.Equal(t, entity.UserRoleAgent, user.Role)
	assert.Equal(t, entity.UserStatusActive, user.Status)
}

func TestUserService_Create_DuplicateEmail(t *testing.T) {
	svc, userRepo, _ := setupUserService()

	// Add existing user
	userRepo.Users["existing"] = &entity.User{
		ID:       "existing",
		TenantID: "tenant-1",
		Email:    "user@example.com",
	}

	_, err := svc.Create(context.Background(), &CreateUserInput{
		TenantID: "tenant-1",
		Email:    "user@example.com",
		Password: "password123",
		Name:     "Duplicate User",
		Role:     entity.UserRoleAgent,
	})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "Email already in use")
}

func TestUserService_Create_TenantNotFound(t *testing.T) {
	svc, _, _ := setupUserService()

	_, err := svc.Create(context.Background(), &CreateUserInput{
		TenantID: "nonexistent",
		Email:    "user@example.com",
		Password: "password123",
		Name:     "Test User",
		Role:     entity.UserRoleAgent,
	})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "Tenant not found")
}

func TestUserService_Create_QuotaExceeded(t *testing.T) {
	svc, userRepo, tenantRepo := setupUserService()

	// Set limits
	tenantRepo.Tenants["tenant-1"].Limits = &entity.TenantLimits{MaxUsers: 1}

	// Add existing user
	userRepo.Users["existing"] = &entity.User{
		ID:       "existing",
		TenantID: "tenant-1",
		Email:    "existing@example.com",
	}

	_, err := svc.Create(context.Background(), &CreateUserInput{
		TenantID: "tenant-1",
		Email:    "new@example.com",
		Password: "password123",
		Name:     "New User",
		Role:     entity.UserRoleAgent,
	})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "Maximum number of users reached")
}

func TestUserService_GetByID(t *testing.T) {
	svc, userRepo, _ := setupUserService()

	userRepo.Users["user-1"] = &entity.User{
		ID:       "user-1",
		TenantID: "tenant-1",
		Name:     "Test User",
	}

	user, err := svc.GetByID(context.Background(), "user-1")
	require.NoError(t, err)
	assert.Equal(t, "user-1", user.ID)
	assert.Equal(t, "Test User", user.Name)
}

func TestUserService_GetByID_NotFound(t *testing.T) {
	svc, _, _ := setupUserService()

	_, err := svc.GetByID(context.Background(), "nonexistent")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "User not found")
}

func TestUserService_List(t *testing.T) {
	svc, userRepo, _ := setupUserService()

	userRepo.Users["u1"] = &entity.User{ID: "u1", TenantID: "tenant-1", Name: "User 1"}
	userRepo.Users["u2"] = &entity.User{ID: "u2", TenantID: "tenant-1", Name: "User 2"}
	userRepo.Users["u3"] = &entity.User{ID: "u3", TenantID: "other", Name: "Other Tenant User"}

	users, count, err := svc.List(context.Background(), "tenant-1", nil)
	require.NoError(t, err)
	assert.Equal(t, int64(2), count)
	assert.Len(t, users, 2)
}

func TestUserService_Update(t *testing.T) {
	svc, userRepo, _ := setupUserService()

	userRepo.Users["user-1"] = &entity.User{
		ID:       "user-1",
		TenantID: "tenant-1",
		Name:     "Old Name",
		Role:     entity.UserRoleAgent,
		Status:   entity.UserStatusActive,
	}

	newName := "New Name"
	newRole := entity.UserRoleAdmin
	user, err := svc.Update(context.Background(), "user-1", &UpdateUserInput{
		Name: &newName,
		Role: &newRole,
	})

	require.NoError(t, err)
	assert.Equal(t, "New Name", user.Name)
	assert.Equal(t, entity.UserRoleAdmin, user.Role)
}

func TestUserService_Update_NotFound(t *testing.T) {
	svc, _, _ := setupUserService()

	name := "New"
	_, err := svc.Update(context.Background(), "nonexistent", &UpdateUserInput{Name: &name})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "User not found")
}

func TestUserService_Delete(t *testing.T) {
	svc, userRepo, _ := setupUserService()

	userRepo.Users["user-1"] = &entity.User{ID: "user-1", TenantID: "tenant-1"}

	err := svc.Delete(context.Background(), "user-1")
	require.NoError(t, err)
	assert.Empty(t, userRepo.Users)
}

func TestUserService_Delete_NotFound(t *testing.T) {
	svc, _, _ := setupUserService()

	err := svc.Delete(context.Background(), "nonexistent")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "User not found")
}
