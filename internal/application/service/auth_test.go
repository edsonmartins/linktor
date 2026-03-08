package service

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/msgfy/linktor/internal/domain/entity"
	"github.com/msgfy/linktor/internal/infrastructure/config"
	"github.com/msgfy/linktor/pkg/errors"
	"github.com/msgfy/linktor/pkg/testutil"
	"golang.org/x/crypto/bcrypt"
)

func newTestAuthService() (*AuthService, *testutil.MockUserRepository) {
	userRepo := testutil.NewMockUserRepository()
	cfg := &config.JWTConfig{
		Secret:          "test-secret-key-for-jwt-signing",
		AccessTokenTTL:  15,  // 15 minutes
		RefreshTokenTTL: 168, // 7 days in hours
		Issuer:          "linktor-test",
	}
	return NewAuthService(userRepo, cfg), userRepo
}

func createTestUser(t *testing.T, tenantID, email, password string) *entity.User {
	t.Helper()
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.MinCost)
	require.NoError(t, err)
	now := time.Now()
	return &entity.User{
		ID:           "user-1",
		TenantID:     tenantID,
		Email:        email,
		PasswordHash: string(hash),
		Name:         "Test User",
		Role:         entity.UserRoleAdmin,
		Status:       entity.UserStatusActive,
		CreatedAt:    now,
		UpdatedAt:    now,
	}
}

func TestAuthService_Login(t *testing.T) {
	t.Run("successful login", func(t *testing.T) {
		svc, userRepo := newTestAuthService()
		user := createTestUser(t, "tenant-1", "admin@test.com", "password123")
		userRepo.Users[user.ID] = user

		result, err := svc.Login(context.Background(), "admin@test.com", "password123")

		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Equal(t, user.ID, result.User.ID)
		assert.NotEmpty(t, result.AccessToken)
		assert.NotEmpty(t, result.RefreshToken)
		assert.Equal(t, int64(15*60), result.ExpiresIn)
		assert.NotNil(t, userRepo.Users[user.ID].LastLoginAt)
	})

	t.Run("wrong email", func(t *testing.T) {
		svc, _ := newTestAuthService()

		result, err := svc.Login(context.Background(), "nonexistent@test.com", "password123")

		assert.Nil(t, result)
		require.Error(t, err)
		appErr := errors.GetAppError(err)
		require.NotNil(t, appErr)
		assert.Equal(t, errors.ErrCodeInvalidCredentials, appErr.Code)
	})

	t.Run("wrong password", func(t *testing.T) {
		svc, userRepo := newTestAuthService()
		user := createTestUser(t, "tenant-1", "admin@test.com", "password123")
		userRepo.Users[user.ID] = user

		result, err := svc.Login(context.Background(), "admin@test.com", "wrongpassword")

		assert.Nil(t, result)
		require.Error(t, err)
		appErr := errors.GetAppError(err)
		require.NotNil(t, appErr)
		assert.Equal(t, errors.ErrCodeInvalidCredentials, appErr.Code)
	})

	t.Run("inactive user", func(t *testing.T) {
		svc, userRepo := newTestAuthService()
		user := createTestUser(t, "tenant-1", "admin@test.com", "password123")
		user.Status = entity.UserStatusInactive
		userRepo.Users[user.ID] = user

		result, err := svc.Login(context.Background(), "admin@test.com", "password123")

		assert.Nil(t, result)
		require.Error(t, err)
		appErr := errors.GetAppError(err)
		require.NotNil(t, appErr)
		assert.Equal(t, errors.ErrCodeInvalidCredentials, appErr.Code)
	})
}

func TestAuthService_RefreshToken(t *testing.T) {
	t.Run("successful refresh", func(t *testing.T) {
		svc, userRepo := newTestAuthService()
		user := createTestUser(t, "tenant-1", "admin@test.com", "password123")
		userRepo.Users[user.ID] = user

		// Login to get a valid refresh token
		loginResult, err := svc.Login(context.Background(), "admin@test.com", "password123")
		require.NoError(t, err)

		// Refresh
		result, err := svc.RefreshToken(context.Background(), loginResult.RefreshToken)

		require.NoError(t, err)
		require.NotNil(t, result)
		assert.NotEmpty(t, result.AccessToken)
		assert.NotEmpty(t, result.RefreshToken)
		assert.Equal(t, int64(15*60), result.ExpiresIn)
	})

	t.Run("invalid token", func(t *testing.T) {
		svc, _ := newTestAuthService()

		result, err := svc.RefreshToken(context.Background(), "invalid-token")

		assert.Nil(t, result)
		require.Error(t, err)
	})

	t.Run("user not found", func(t *testing.T) {
		svc, userRepo := newTestAuthService()
		user := createTestUser(t, "tenant-1", "admin@test.com", "password123")
		userRepo.Users[user.ID] = user

		loginResult, err := svc.Login(context.Background(), "admin@test.com", "password123")
		require.NoError(t, err)

		// Delete user after getting token
		delete(userRepo.Users, user.ID)

		result, err := svc.RefreshToken(context.Background(), loginResult.RefreshToken)

		assert.Nil(t, result)
		require.Error(t, err)
	})

	t.Run("inactive user on refresh", func(t *testing.T) {
		svc, userRepo := newTestAuthService()
		user := createTestUser(t, "tenant-1", "admin@test.com", "password123")
		userRepo.Users[user.ID] = user

		loginResult, err := svc.Login(context.Background(), "admin@test.com", "password123")
		require.NoError(t, err)

		// Deactivate user after getting token
		user.Status = entity.UserStatusInactive

		result, err := svc.RefreshToken(context.Background(), loginResult.RefreshToken)

		assert.Nil(t, result)
		require.Error(t, err)
	})
}

func TestAuthService_ValidateAccessToken(t *testing.T) {
	t.Run("valid token", func(t *testing.T) {
		svc, userRepo := newTestAuthService()
		user := createTestUser(t, "tenant-1", "admin@test.com", "password123")
		userRepo.Users[user.ID] = user

		loginResult, err := svc.Login(context.Background(), "admin@test.com", "password123")
		require.NoError(t, err)

		claims, err := svc.ValidateAccessToken(loginResult.AccessToken)

		require.NoError(t, err)
		assert.Equal(t, "tenant-1", claims.TenantID)
		assert.Equal(t, "user-1", claims.UserID)
		assert.Equal(t, "admin@test.com", claims.Email)
		assert.Equal(t, "admin", claims.Role)
	})

	t.Run("invalid token", func(t *testing.T) {
		svc, _ := newTestAuthService()

		claims, err := svc.ValidateAccessToken("invalid-token")

		assert.Nil(t, claims)
		require.Error(t, err)
	})

	t.Run("token with wrong secret", func(t *testing.T) {
		svc1, userRepo1 := newTestAuthService()
		user := createTestUser(t, "tenant-1", "admin@test.com", "password123")
		userRepo1.Users[user.ID] = user

		loginResult, err := svc1.Login(context.Background(), "admin@test.com", "password123")
		require.NoError(t, err)

		// Create another service with different secret
		svc2 := NewAuthService(testutil.NewMockUserRepository(), &config.JWTConfig{
			Secret:          "different-secret",
			AccessTokenTTL:  15,
			RefreshTokenTTL: 168,
			Issuer:          "linktor-test",
		})

		claims, err := svc2.ValidateAccessToken(loginResult.AccessToken)

		assert.Nil(t, claims)
		require.Error(t, err)
	})
}

func TestAuthService_ChangePassword(t *testing.T) {
	t.Run("successful change", func(t *testing.T) {
		svc, userRepo := newTestAuthService()
		user := createTestUser(t, "tenant-1", "admin@test.com", "password123")
		userRepo.Users[user.ID] = user

		err := svc.ChangePassword(context.Background(), "user-1", "password123", "newpassword456")

		require.NoError(t, err)
		// Verify new password works
		updatedUser := userRepo.Users["user-1"]
		err = bcrypt.CompareHashAndPassword([]byte(updatedUser.PasswordHash), []byte("newpassword456"))
		assert.NoError(t, err)
	})

	t.Run("wrong current password", func(t *testing.T) {
		svc, userRepo := newTestAuthService()
		user := createTestUser(t, "tenant-1", "admin@test.com", "password123")
		userRepo.Users[user.ID] = user

		err := svc.ChangePassword(context.Background(), "user-1", "wrongpassword", "newpassword456")

		require.Error(t, err)
		appErr := errors.GetAppError(err)
		require.NotNil(t, appErr)
		assert.Equal(t, errors.ErrCodeInvalidCredentials, appErr.Code)
	})

	t.Run("user not found", func(t *testing.T) {
		svc, _ := newTestAuthService()

		err := svc.ChangePassword(context.Background(), "nonexistent", "password123", "newpassword456")

		require.Error(t, err)
	})
}

func TestHashPassword(t *testing.T) {
	hash, err := HashPassword("testpassword")

	require.NoError(t, err)
	assert.NotEmpty(t, hash)
	assert.NotEqual(t, "testpassword", hash)

	// Verify hash
	err = bcrypt.CompareHashAndPassword([]byte(hash), []byte("testpassword"))
	assert.NoError(t, err)

	// Wrong password doesn't verify
	err = bcrypt.CompareHashAndPassword([]byte(hash), []byte("wrong"))
	assert.Error(t, err)
}
