package middleware

import (
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

func newTestAuthMiddleware(t *testing.T) (*AuthMiddleware, *service.LoginResult) {
	t.Helper()
	gin.SetMode(gin.TestMode)

	userRepo := testutil.NewMockUserRepository()
	hash, err := bcrypt.GenerateFromPassword([]byte("password123"), bcrypt.MinCost)
	require.NoError(t, err)
	user := &entity.User{
		ID:           "user-1",
		TenantID:     "tenant-1",
		Email:        "admin@test.com",
		PasswordHash: string(hash),
		Name:         "Admin",
		Role:         entity.UserRoleAdmin,
		Status:       entity.UserStatusActive,
	}
	userRepo.Users[user.ID] = user

	authService := service.NewAuthService(userRepo, &config.JWTConfig{
		Secret:          "test-secret-key-for-jwt-signing",
		AccessTokenTTL:  15,
		RefreshTokenTTL: 168,
		Issuer:          "linktor-test",
	})

	loginResult, err := authService.Login(t.Context(), "admin@test.com", "password123")
	require.NoError(t, err)

	return NewAuthMiddleware(authService, nil), loginResult
}

func doAuthRequest(m *AuthMiddleware, bearer string) *httptest.ResponseRecorder {
	r := gin.New()
	r.GET("/protected", m.Authenticate(), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})
	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set(AuthorizationHeader, BearerPrefix+bearer)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func TestAuthenticate_AcceptsAccessToken(t *testing.T) {
	m, login := newTestAuthMiddleware(t)
	w := doAuthRequest(m, login.AccessToken)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestAuthenticate_RejectsRefreshTokenAsCredential(t *testing.T) {
	m, login := newTestAuthMiddleware(t)
	// A refresh token must NOT be accepted as an API access credential.
	w := doAuthRequest(m, login.RefreshToken)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}
