package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/msgfy/linktor/internal/application/service"
	"github.com/msgfy/linktor/internal/domain/entity"
	"github.com/msgfy/linktor/internal/infrastructure/config"
	"github.com/msgfy/linktor/pkg/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"
)

func setupAuthTest(t *testing.T) (*AuthHandler, *testutil.MockUserRepository, *testutil.MockTenantRepository) {
	t.Helper()

	userRepo := testutil.NewMockUserRepository()
	tenantRepo := testutil.NewMockTenantRepository()

	jwtCfg := &config.JWTConfig{
		Secret:          "test-secret",
		AccessTokenTTL:  15,
		RefreshTokenTTL: 24,
		Issuer:          "test",
	}

	authService := service.NewAuthService(userRepo, jwtCfg)
	userService := service.NewUserService(userRepo, tenantRepo)
	handler := NewAuthHandler(authService, userService)

	return handler, userRepo, tenantRepo
}

func createTestUser(t *testing.T) *entity.User {
	t.Helper()
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte("password123"), bcrypt.DefaultCost)
	require.NoError(t, err)

	return &entity.User{
		ID:           "user-1",
		TenantID:     "tenant-1",
		Email:        "test@example.com",
		Name:         "Test User",
		PasswordHash: string(hashedPassword),
		Role:         entity.UserRoleAdmin,
		Status:       entity.UserStatusActive,
	}
}

func newTestContext(method, path string, body interface{}) (*httptest.ResponseRecorder, *gin.Context) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	var reqBody *bytes.Buffer
	if body != nil {
		jsonBody, _ := json.Marshal(body)
		reqBody = bytes.NewBuffer(jsonBody)
	} else {
		reqBody = bytes.NewBuffer(nil)
	}

	c.Request = httptest.NewRequest(method, path, reqBody)
	c.Request.Header.Set("Content-Type", "application/json")
	return w, c
}

// --- Login tests ---

func TestLogin_ValidCredentials(t *testing.T) {
	handler, userRepo, _ := setupAuthTest(t)
	user := createTestUser(t)
	userRepo.Users[user.ID] = user

	body := LoginRequest{
		Email:    "test@example.com",
		Password: "password123",
	}
	w, c := newTestContext(http.MethodPost, "/auth/login", body)

	handler.Login(c)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp Response
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.True(t, resp.Success)
	assert.NotNil(t, resp.Data)

	dataMap, ok := resp.Data.(map[string]interface{})
	require.True(t, ok)
	assert.NotEmpty(t, dataMap["access_token"])
	assert.NotEmpty(t, dataMap["refresh_token"])
	assert.NotNil(t, dataMap["expires_in"])
	assert.NotNil(t, dataMap["user"])
}

func TestLogin_InvalidJSON(t *testing.T) {
	handler, _, _ := setupAuthTest(t)

	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/auth/login", bytes.NewBufferString("{invalid"))
	c.Request.Header.Set("Content-Type", "application/json")

	handler.Login(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var resp Response
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.False(t, resp.Success)
	assert.NotNil(t, resp.Error)
	assert.Equal(t, "VALIDATION_ERROR", resp.Error.Code)
}

func TestLogin_WrongPassword(t *testing.T) {
	handler, userRepo, _ := setupAuthTest(t)
	user := createTestUser(t)
	userRepo.Users[user.ID] = user

	body := LoginRequest{
		Email:    "test@example.com",
		Password: "wrongpassword",
	}
	w, c := newTestContext(http.MethodPost, "/auth/login", body)

	handler.Login(c)

	assert.NotEqual(t, http.StatusOK, w.Code)

	var resp Response
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.False(t, resp.Success)
	assert.NotNil(t, resp.Error)
}

func TestLogin_NonexistentEmail(t *testing.T) {
	handler, _, _ := setupAuthTest(t)

	body := LoginRequest{
		Email:    "nobody@example.com",
		Password: "password123",
	}
	w, c := newTestContext(http.MethodPost, "/auth/login", body)

	handler.Login(c)

	assert.NotEqual(t, http.StatusOK, w.Code)

	var resp Response
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.False(t, resp.Success)
}

// --- RefreshToken tests ---

func TestRefreshToken_Valid(t *testing.T) {
	handler, userRepo, _ := setupAuthTest(t)
	user := createTestUser(t)
	userRepo.Users[user.ID] = user

	// First login to get a refresh token
	loginBody := LoginRequest{
		Email:    "test@example.com",
		Password: "password123",
	}
	loginW, loginC := newTestContext(http.MethodPost, "/auth/login", loginBody)
	handler.Login(loginC)
	require.Equal(t, http.StatusOK, loginW.Code)

	var loginResp Response
	err := json.Unmarshal(loginW.Body.Bytes(), &loginResp)
	require.NoError(t, err)

	dataMap := loginResp.Data.(map[string]interface{})
	refreshToken := dataMap["refresh_token"].(string)

	// Now use the refresh token
	body := RefreshRequest{RefreshToken: refreshToken}
	w, c := newTestContext(http.MethodPost, "/auth/refresh", body)

	handler.RefreshToken(c)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp Response
	err = json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.True(t, resp.Success)

	respData := resp.Data.(map[string]interface{})
	assert.NotEmpty(t, respData["access_token"])
	assert.NotEmpty(t, respData["refresh_token"])
}

func TestRefreshToken_InvalidBody(t *testing.T) {
	handler, _, _ := setupAuthTest(t)

	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/auth/refresh", bytes.NewBufferString("{bad"))
	c.Request.Header.Set("Content-Type", "application/json")

	handler.RefreshToken(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var resp Response
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.False(t, resp.Success)
	assert.Equal(t, "VALIDATION_ERROR", resp.Error.Code)
}

// --- Me tests ---

func TestMe_Authenticated(t *testing.T) {
	handler, userRepo, _ := setupAuthTest(t)
	user := createTestUser(t)
	userRepo.Users[user.ID] = user

	w, c := newTestContext(http.MethodGet, "/me", nil)
	c.Set("user_id", "user-1")
	c.Set("tenant_id", "tenant-1")

	handler.Me(c)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp Response
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.True(t, resp.Success)

	dataMap := resp.Data.(map[string]interface{})
	assert.Equal(t, "user-1", dataMap["id"])
	assert.Equal(t, "test@example.com", dataMap["email"])
	assert.Equal(t, "Test User", dataMap["name"])
}

func TestMe_NoUserID(t *testing.T) {
	handler, _, _ := setupAuthTest(t)

	w, c := newTestContext(http.MethodGet, "/me", nil)
	// Do not set user_id

	handler.Me(c)

	assert.Equal(t, http.StatusUnauthorized, w.Code)

	var resp Response
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.False(t, resp.Success)
	assert.Equal(t, "UNAUTHORIZED", resp.Error.Code)
}

// --- UpdateMe tests ---

func TestUpdateMe_Valid(t *testing.T) {
	handler, userRepo, _ := setupAuthTest(t)
	user := createTestUser(t)
	userRepo.Users[user.ID] = user

	newName := "Updated Name"
	body := UpdateMeRequest{
		Name: &newName,
	}
	w, c := newTestContext(http.MethodPut, "/me", body)
	c.Set("user_id", "user-1")
	c.Set("tenant_id", "tenant-1")

	handler.UpdateMe(c)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp Response
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.True(t, resp.Success)

	dataMap := resp.Data.(map[string]interface{})
	assert.Equal(t, "Updated Name", dataMap["name"])
}

func TestUpdateMe_NoUserID(t *testing.T) {
	handler, _, _ := setupAuthTest(t)

	newName := "Updated Name"
	body := UpdateMeRequest{
		Name: &newName,
	}
	w, c := newTestContext(http.MethodPut, "/me", body)
	// Do not set user_id

	handler.UpdateMe(c)

	assert.Equal(t, http.StatusUnauthorized, w.Code)

	var resp Response
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.False(t, resp.Success)
	assert.Equal(t, "UNAUTHORIZED", resp.Error.Code)
}

func TestUpdateMe_InvalidBody(t *testing.T) {
	handler, _, _ := setupAuthTest(t)

	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Set("user_id", "user-1")
	c.Set("tenant_id", "tenant-1")
	c.Request = httptest.NewRequest(http.MethodPut, "/me", bytes.NewBufferString("{bad"))
	c.Request.Header.Set("Content-Type", "application/json")

	handler.UpdateMe(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var resp Response
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.False(t, resp.Success)
	assert.Equal(t, "VALIDATION_ERROR", resp.Error.Code)
}

// --- ChangePassword tests ---

func TestChangePassword_Valid(t *testing.T) {
	handler, userRepo, _ := setupAuthTest(t)
	user := createTestUser(t)
	userRepo.Users[user.ID] = user

	body := ChangePasswordRequest{
		CurrentPassword: "password123",
		NewPassword:     "newpassword123",
	}
	w, c := newTestContext(http.MethodPut, "/me/password", body)
	c.Set("user_id", "user-1")
	c.Set("tenant_id", "tenant-1")

	handler.ChangePassword(c)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp Response
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.True(t, resp.Success)

	// Verify password was actually changed
	updatedUser := userRepo.Users["user-1"]
	err = bcrypt.CompareHashAndPassword([]byte(updatedUser.PasswordHash), []byte("newpassword123"))
	assert.NoError(t, err)
}

func TestChangePassword_NoUserID(t *testing.T) {
	handler, _, _ := setupAuthTest(t)

	body := ChangePasswordRequest{
		CurrentPassword: "password123",
		NewPassword:     "newpassword123",
	}
	w, c := newTestContext(http.MethodPut, "/me/password", body)
	// Do not set user_id

	handler.ChangePassword(c)

	assert.Equal(t, http.StatusUnauthorized, w.Code)

	var resp Response
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.False(t, resp.Success)
	assert.Equal(t, "UNAUTHORIZED", resp.Error.Code)
}

func TestChangePassword_InvalidBody(t *testing.T) {
	handler, _, _ := setupAuthTest(t)

	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Set("user_id", "user-1")
	c.Set("tenant_id", "tenant-1")
	c.Request = httptest.NewRequest(http.MethodPut, "/me/password", bytes.NewBufferString("{bad"))
	c.Request.Header.Set("Content-Type", "application/json")

	handler.ChangePassword(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var resp Response
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.False(t, resp.Success)
	assert.Equal(t, "VALIDATION_ERROR", resp.Error.Code)
}

func TestChangePassword_WrongCurrentPassword(t *testing.T) {
	handler, userRepo, _ := setupAuthTest(t)
	user := createTestUser(t)
	userRepo.Users[user.ID] = user

	body := ChangePasswordRequest{
		CurrentPassword: "wrongpassword",
		NewPassword:     "newpassword123",
	}
	w, c := newTestContext(http.MethodPut, "/me/password", body)
	c.Set("user_id", "user-1")
	c.Set("tenant_id", "tenant-1")

	handler.ChangePassword(c)

	assert.NotEqual(t, http.StatusOK, w.Code)

	var resp Response
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.False(t, resp.Success)
}
