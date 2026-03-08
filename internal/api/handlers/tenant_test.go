package handlers

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/msgfy/linktor/internal/application/service"
	"github.com/msgfy/linktor/internal/domain/entity"
	"github.com/msgfy/linktor/pkg/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupTenantTest(t *testing.T) (*TenantHandler, *testutil.MockTenantRepository, *testutil.MockUserRepository, *testutil.MockChannelRepository, *testutil.MockContactRepository) {
	t.Helper()

	tenantRepo := testutil.NewMockTenantRepository()
	userRepo := testutil.NewMockUserRepository()
	channelRepo := testutil.NewMockChannelRepository()
	contactRepo := testutil.NewMockContactRepository()

	tenantService := service.NewTenantService(tenantRepo, userRepo, channelRepo, contactRepo)
	handler := NewTenantHandler(tenantService)

	return handler, tenantRepo, userRepo, channelRepo, contactRepo
}

func TestTenantHandler_Get(t *testing.T) {
	handler, tenantRepo, _, _, _ := setupTenantTest(t)

	tenantRepo.Tenants["tenant-1"] = &entity.Tenant{
		ID:       "tenant-1",
		Name:     "Test Tenant",
		Slug:     "test-tenant",
		Plan:     entity.PlanFree,
		Status:   entity.TenantStatusActive,
		Settings: map[string]string{"key": "value"},
	}

	w, c := newTestContext(http.MethodGet, "/tenant", nil)
	c.Set("tenant_id", "tenant-1")

	handler.Get(c)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp Response
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.True(t, resp.Success)

	dataMap := resp.Data.(map[string]interface{})
	assert.Equal(t, "tenant-1", dataMap["id"])
	assert.Equal(t, "Test Tenant", dataMap["name"])
}

func TestTenantHandler_Get_NoTenantID(t *testing.T) {
	handler, _, _, _, _ := setupTenantTest(t)

	w, c := newTestContext(http.MethodGet, "/tenant", nil)
	// Do not set tenant_id

	handler.Get(c)

	assert.NotEqual(t, http.StatusOK, w.Code)
}

func TestTenantHandler_Get_NotFound(t *testing.T) {
	handler, _, _, _, _ := setupTenantTest(t)

	w, c := newTestContext(http.MethodGet, "/tenant", nil)
	c.Set("tenant_id", "nonexistent")

	handler.Get(c)

	assert.NotEqual(t, http.StatusOK, w.Code)

	var resp Response
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.False(t, resp.Success)
}

func TestTenantHandler_Update(t *testing.T) {
	handler, tenantRepo, _, _, _ := setupTenantTest(t)

	tenantRepo.Tenants["tenant-1"] = &entity.Tenant{
		ID:       "tenant-1",
		Name:     "Test Tenant",
		Slug:     "test-tenant",
		Plan:     entity.PlanFree,
		Status:   entity.TenantStatusActive,
		Settings: map[string]string{},
	}

	newName := "Updated Tenant"
	body := UpdateTenantRequest{
		Name:     &newName,
		Settings: map[string]string{"theme": "dark"},
	}

	w, c := newTestContext(http.MethodPut, "/tenant", body)
	c.Set("tenant_id", "tenant-1")

	handler.Update(c)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp Response
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.True(t, resp.Success)

	dataMap := resp.Data.(map[string]interface{})
	assert.Equal(t, "Updated Tenant", dataMap["name"])
}

func TestTenantHandler_Update_NoTenantID(t *testing.T) {
	handler, _, _, _, _ := setupTenantTest(t)

	newName := "Updated"
	body := UpdateTenantRequest{
		Name: &newName,
	}

	w, c := newTestContext(http.MethodPut, "/tenant", body)
	// Do not set tenant_id

	handler.Update(c)

	assert.NotEqual(t, http.StatusOK, w.Code)
}

func TestTenantHandler_Update_InvalidBody(t *testing.T) {
	handler, tenantRepo, _, _, _ := setupTenantTest(t)

	tenantRepo.Tenants["tenant-1"] = &entity.Tenant{
		ID:   "tenant-1",
		Name: "Test Tenant",
		Slug: "test-tenant",
	}

	gin.SetMode(gin.TestMode)
	w, c := newTestContext(http.MethodPut, "/tenant", nil)
	c.Set("tenant_id", "tenant-1")
	// Send a request with empty body which should still bind (empty JSON object is ok)
	// Instead send invalid JSON
	c.Request.Body = http.NoBody

	handler.Update(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var resp Response
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.False(t, resp.Success)
	assert.Equal(t, "VALIDATION_ERROR", resp.Error.Code)
}

func TestTenantHandler_GetUsage(t *testing.T) {
	handler, tenantRepo, _, _, _ := setupTenantTest(t)

	tenantRepo.Tenants["tenant-1"] = &entity.Tenant{
		ID:   "tenant-1",
		Name: "Test Tenant",
		Slug: "test-tenant",
		Plan: entity.PlanFree,
	}

	w, c := newTestContext(http.MethodGet, "/tenant/usage", nil)
	c.Set("tenant_id", "tenant-1")

	handler.GetUsage(c)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp Response
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.True(t, resp.Success)
	assert.NotNil(t, resp.Data)
}
