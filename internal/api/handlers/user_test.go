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

func setupUserTest(t *testing.T) (*UserHandler, *testutil.MockUserRepository, *testutil.MockTenantRepository) {
	t.Helper()

	userRepo := testutil.NewMockUserRepository()
	tenantRepo := testutil.NewMockTenantRepository()

	userService := service.NewUserService(userRepo, tenantRepo)
	handler := NewUserHandler(userService)

	return handler, userRepo, tenantRepo
}

func TestUserHandler_List(t *testing.T) {
	handler, userRepo, _ := setupUserTest(t)

	userRepo.Users["user-1"] = &entity.User{
		ID:       "user-1",
		TenantID: "tenant-1",
		Email:    "alice@example.com",
		Name:     "Alice",
		Role:     entity.UserRoleAdmin,
		Status:   entity.UserStatusActive,
	}
	userRepo.Users["user-2"] = &entity.User{
		ID:       "user-2",
		TenantID: "tenant-1",
		Email:    "bob@example.com",
		Name:     "Bob",
		Role:     entity.UserRoleAgent,
		Status:   entity.UserStatusActive,
	}

	w, c := newTestContext(http.MethodGet, "/users", nil)
	c.Set("tenant_id", "tenant-1")

	handler.List(c)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp Response
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.True(t, resp.Success)
	assert.NotNil(t, resp.Data)
	assert.NotNil(t, resp.Meta)

	dataList, ok := resp.Data.([]interface{})
	require.True(t, ok)
	assert.Len(t, dataList, 2)
}

func TestUserHandler_List_NoTenantID(t *testing.T) {
	handler, _, _ := setupUserTest(t)

	w, c := newTestContext(http.MethodGet, "/users", nil)
	// Do not set tenant_id

	handler.List(c)

	assert.NotEqual(t, http.StatusOK, w.Code)
}

func TestUserHandler_Get(t *testing.T) {
	handler, userRepo, _ := setupUserTest(t)

	userRepo.Users["user-1"] = &entity.User{
		ID:       "user-1",
		TenantID: "tenant-1",
		Email:    "alice@example.com",
		Name:     "Alice",
		Role:     entity.UserRoleAdmin,
		Status:   entity.UserStatusActive,
	}

	w, c := newTestContext(http.MethodGet, "/users/user-1", nil)
	c.Set("tenant_id", "tenant-1")
	c.Params = gin.Params{{Key: "id", Value: "user-1"}}

	handler.Get(c)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp Response
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.True(t, resp.Success)

	dataMap := resp.Data.(map[string]interface{})
	assert.Equal(t, "user-1", dataMap["id"])
	assert.Equal(t, "alice@example.com", dataMap["email"])
	assert.Equal(t, "Alice", dataMap["name"])
}

func TestUserHandler_Get_NotFound(t *testing.T) {
	handler, _, _ := setupUserTest(t)

	w, c := newTestContext(http.MethodGet, "/users/nonexistent", nil)
	c.Set("tenant_id", "tenant-1")
	c.Params = gin.Params{{Key: "id", Value: "nonexistent"}}

	handler.Get(c)

	assert.NotEqual(t, http.StatusOK, w.Code)

	var resp Response
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.False(t, resp.Success)
}

func TestUserHandler_Get_WrongTenant(t *testing.T) {
	handler, userRepo, _ := setupUserTest(t)

	userRepo.Users["user-1"] = &entity.User{
		ID:       "user-1",
		TenantID: "tenant-2",
		Email:    "alice@example.com",
		Name:     "Alice",
		Role:     entity.UserRoleAdmin,
		Status:   entity.UserStatusActive,
	}

	w, c := newTestContext(http.MethodGet, "/users/user-1", nil)
	c.Set("tenant_id", "tenant-1")
	c.Params = gin.Params{{Key: "id", Value: "user-1"}}

	handler.Get(c)

	assert.Equal(t, http.StatusNotFound, w.Code)

	var resp Response
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.False(t, resp.Success)
	assert.Equal(t, "NOT_FOUND", resp.Error.Code)
}

func TestUserHandler_Create(t *testing.T) {
	handler, _, tenantRepo := setupUserTest(t)

	tenantRepo.Tenants["tenant-1"] = &entity.Tenant{
		ID:   "tenant-1",
		Name: "Test Tenant",
		Slug: "test-tenant",
	}

	body := CreateUserRequest{
		Email:    "new@example.com",
		Password: "password123",
		Name:     "New User",
		Role:     "agent",
	}

	w, c := newTestContext(http.MethodPost, "/users", body)
	c.Set("tenant_id", "tenant-1")

	handler.Create(c)

	assert.Equal(t, http.StatusCreated, w.Code)

	var resp Response
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.True(t, resp.Success)

	dataMap := resp.Data.(map[string]interface{})
	assert.Equal(t, "new@example.com", dataMap["email"])
	assert.Equal(t, "New User", dataMap["name"])
	assert.Equal(t, "agent", dataMap["role"])
}

func TestUserHandler_Create_InvalidBody(t *testing.T) {
	handler, _, _ := setupUserTest(t)

	// Missing required fields
	body := map[string]string{
		"email": "bad",
	}

	w, c := newTestContext(http.MethodPost, "/users", body)
	c.Set("tenant_id", "tenant-1")

	handler.Create(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var resp Response
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.False(t, resp.Success)
	assert.Equal(t, "VALIDATION_ERROR", resp.Error.Code)
}

func TestUserHandler_Update(t *testing.T) {
	handler, userRepo, _ := setupUserTest(t)

	userRepo.Users["user-1"] = &entity.User{
		ID:       "user-1",
		TenantID: "tenant-1",
		Email:    "alice@example.com",
		Name:     "Alice",
		Role:     entity.UserRoleAgent,
		Status:   entity.UserStatusActive,
	}

	newName := "Alice Updated"
	body := UpdateUserRequest{
		Name: &newName,
	}

	w, c := newTestContext(http.MethodPut, "/users/user-1", body)
	c.Set("tenant_id", "tenant-1")
	c.Params = gin.Params{{Key: "id", Value: "user-1"}}

	handler.Update(c)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp Response
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.True(t, resp.Success)

	dataMap := resp.Data.(map[string]interface{})
	assert.Equal(t, "Alice Updated", dataMap["name"])
}

func TestUserHandler_Update_NotFound(t *testing.T) {
	handler, _, _ := setupUserTest(t)

	newName := "Updated"
	body := UpdateUserRequest{
		Name: &newName,
	}

	w, c := newTestContext(http.MethodPut, "/users/nonexistent", body)
	c.Set("tenant_id", "tenant-1")
	c.Params = gin.Params{{Key: "id", Value: "nonexistent"}}

	handler.Update(c)

	assert.NotEqual(t, http.StatusOK, w.Code)

	var resp Response
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.False(t, resp.Success)
}

func TestUserHandler_Delete(t *testing.T) {
	handler, userRepo, _ := setupUserTest(t)

	userRepo.Users["user-2"] = &entity.User{
		ID:       "user-2",
		TenantID: "tenant-1",
		Email:    "bob@example.com",
		Name:     "Bob",
		Role:     entity.UserRoleAgent,
		Status:   entity.UserStatusActive,
	}

	w, c := newTestContext(http.MethodDelete, "/users/user-2", nil)
	c.Set("tenant_id", "tenant-1")
	c.Set("user_id", "user-1") // Different from the user being deleted
	c.Params = gin.Params{{Key: "id", Value: "user-2"}}

	handler.Delete(c)

	gotCode := w.Code
	if gotCode == http.StatusOK {
		gotCode = c.Writer.Status()
	}
	assert.Equal(t, http.StatusNoContent, gotCode)
}

func TestUserHandler_Delete_SelfDeletion(t *testing.T) {
	handler, userRepo, _ := setupUserTest(t)

	userRepo.Users["user-1"] = &entity.User{
		ID:       "user-1",
		TenantID: "tenant-1",
		Email:    "alice@example.com",
		Name:     "Alice",
		Role:     entity.UserRoleAdmin,
		Status:   entity.UserStatusActive,
	}

	w, c := newTestContext(http.MethodDelete, "/users/user-1", nil)
	c.Set("tenant_id", "tenant-1")
	c.Set("user_id", "user-1") // Same as the user being deleted
	c.Params = gin.Params{{Key: "id", Value: "user-1"}}

	handler.Delete(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var resp Response
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.False(t, resp.Success)
	assert.Equal(t, "VALIDATION_ERROR", resp.Error.Code)
}
