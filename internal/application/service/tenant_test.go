package service

import (
	"context"
	"testing"

	"github.com/msgfy/linktor/internal/domain/entity"
	"github.com/msgfy/linktor/pkg/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupTenantService() (*TenantService, *testutil.MockTenantRepository, *testutil.MockUserRepository) {
	tenantRepo := testutil.NewMockTenantRepository()
	userRepo := testutil.NewMockUserRepository()
	channelRepo := testutil.NewMockChannelRepository()
	contactRepo := testutil.NewMockContactRepository()

	svc := NewTenantService(tenantRepo, userRepo, channelRepo, contactRepo)
	return svc, tenantRepo, userRepo
}

func TestTenantService_GetByID(t *testing.T) {
	svc, tenantRepo, _ := setupTenantService()

	tenantRepo.Tenants["t1"] = &entity.Tenant{
		ID:       "t1",
		Name:     "Acme Corp",
		Plan:     entity.PlanProfessional,
		Settings: map[string]string{},
	}

	tenant, err := svc.GetByID(context.Background(), "t1")
	require.NoError(t, err)
	assert.Equal(t, "t1", tenant.ID)
	assert.Equal(t, "Acme Corp", tenant.Name)
}

func TestTenantService_GetByID_NotFound(t *testing.T) {
	svc, _, _ := setupTenantService()

	_, err := svc.GetByID(context.Background(), "nonexistent")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "Tenant not found")
}

func TestTenantService_Update(t *testing.T) {
	svc, tenantRepo, _ := setupTenantService()

	tenantRepo.Tenants["t1"] = &entity.Tenant{
		ID:       "t1",
		Name:     "Old Name",
		Plan:     entity.PlanFree,
		Settings: map[string]string{"key": "value"},
	}

	newName := "New Name"
	newPlan := entity.PlanProfessional
	tenant, err := svc.Update(context.Background(), "t1", &UpdateTenantInput{
		Name:     &newName,
		Plan:     &newPlan,
		Settings: map[string]string{"extra": "data"},
	})

	require.NoError(t, err)
	assert.Equal(t, "New Name", tenant.Name)
	assert.Equal(t, entity.PlanProfessional, tenant.Plan)
	assert.NotNil(t, tenant.Limits) // Plan change sets limits
	assert.Equal(t, "value", tenant.Settings["key"])
	assert.Equal(t, "data", tenant.Settings["extra"])
}

func TestTenantService_Update_NotFound(t *testing.T) {
	svc, _, _ := setupTenantService()

	name := "New"
	_, err := svc.Update(context.Background(), "nonexistent", &UpdateTenantInput{Name: &name})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "Tenant not found")
}

func TestTenantService_GetUsage(t *testing.T) {
	svc, tenantRepo, userRepo := setupTenantService()

	tenantRepo.Tenants["t1"] = &entity.Tenant{
		ID:       "t1",
		Name:     "Test",
		Plan:     entity.PlanProfessional,
		Limits:   entity.GetPlanLimits(entity.PlanProfessional),
		Settings: map[string]string{},
	}

	userRepo.Users["u1"] = &entity.User{ID: "u1", TenantID: "t1"}
	userRepo.Users["u2"] = &entity.User{ID: "u2", TenantID: "t1"}

	usage, err := svc.GetUsage(context.Background(), "t1")
	require.NoError(t, err)
	assert.Equal(t, int64(2), usage.Users)
	assert.NotNil(t, usage.Limits)
}

func TestTenantService_GetUsage_NotFound(t *testing.T) {
	svc, _, _ := setupTenantService()

	_, err := svc.GetUsage(context.Background(), "nonexistent")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "Tenant not found")
}
