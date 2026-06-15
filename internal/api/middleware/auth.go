package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/msgfy/linktor/internal/application/service"
	"github.com/msgfy/linktor/pkg/errors"
)

const (
	// AuthorizationHeader is the header name for authorization
	AuthorizationHeader = "Authorization"
	// BearerPrefix is the prefix for bearer tokens
	BearerPrefix = "Bearer "
	// TenantIDKey is the context key for tenant ID
	TenantIDKey = "tenant_id"
	// UserIDKey is the context key for user ID
	UserIDKey = "user_id"
	// UserRoleKey is the context key for user role
	UserRoleKey = "user_role"
	// UserEmailKey is the context key for user email
	UserEmailKey = "user_email"
	// AccessTokenCookie is the cookie name browsers use to carry the access token
	AccessTokenCookie = "access_token"
)

// extractToken pulls the access token from the Authorization header (Bearer),
// falling back to the HttpOnly access_token cookie used by browser clients.
func extractToken(c *gin.Context) string {
	authHeader := c.GetHeader(AuthorizationHeader)
	if strings.HasPrefix(authHeader, BearerPrefix) {
		if token := strings.TrimPrefix(authHeader, BearerPrefix); token != "" {
			return token
		}
	}
	if cookie, err := c.Cookie(AccessTokenCookie); err == nil {
		return cookie
	}
	return ""
}

// AuthMiddleware handles JWT authentication
type AuthMiddleware struct {
	authService *service.AuthService
}

// NewAuthMiddleware creates a new auth middleware
func NewAuthMiddleware(authService *service.AuthService) *AuthMiddleware {
	return &AuthMiddleware{
		authService: authService,
	}
}

// Authenticate returns a gin middleware that validates JWT tokens
func (m *AuthMiddleware) Authenticate() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Token comes from the Authorization header (API/CLI) or the HttpOnly
		// access_token cookie (browser).
		token := extractToken(c)
		if token == "" {
			abortWithError(c, errors.Unauthorized("missing authentication credentials"))
			return
		}

		// Validate token
		claims, err := m.authService.ValidateAccessToken(token)
		if err != nil {
			abortWithError(c, errors.New(errors.ErrCodeTokenInvalid, "invalid or expired token"))
			return
		}

		// Set user info in context
		c.Set(TenantIDKey, claims.TenantID)
		c.Set(UserIDKey, claims.UserID)
		c.Set(UserRoleKey, claims.Role)
		c.Set(UserEmailKey, claims.Email)

		c.Next()
	}
}

// RequireRole returns a gin middleware that checks user roles
func (m *AuthMiddleware) RequireRole(roles ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		userRole := c.GetString(UserRoleKey)
		if userRole == "" {
			abortWithError(c, errors.Unauthorized("user role not found"))
			return
		}

		// Check if user has required role
		hasRole := false
		for _, role := range roles {
			if userRole == role {
				hasRole = true
				break
			}
		}

		if !hasRole {
			abortWithError(c, errors.Forbidden("insufficient permissions"))
			return
		}

		c.Next()
	}
}

// OptionalAuth returns a gin middleware that optionally validates JWT tokens
func (m *AuthMiddleware) OptionalAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		token := extractToken(c)
		if token == "" {
			c.Next()
			return
		}

		claims, err := m.authService.ValidateAccessToken(token)
		if err != nil {
			c.Next()
			return
		}

		c.Set(TenantIDKey, claims.TenantID)
		c.Set(UserIDKey, claims.UserID)
		c.Set(UserRoleKey, claims.Role)
		c.Set(UserEmailKey, claims.Email)

		c.Next()
	}
}

// abortWithError aborts the request with an error response
func abortWithError(c *gin.Context, err *errors.AppError) {
	c.AbortWithStatusJSON(err.StatusCode, gin.H{
		"code":    err.Code,
		"message": err.Message,
		"details": err.Details,
	})
}

// GetTenantID extracts tenant ID from context
func GetTenantID(c *gin.Context) string {
	return c.GetString(TenantIDKey)
}

// GetUserID extracts user ID from context
func GetUserID(c *gin.Context) string {
	return c.GetString(UserIDKey)
}

// GetUserRole extracts user role from context
func GetUserRole(c *gin.Context) string {
	return c.GetString(UserRoleKey)
}

// GetUserEmail extracts user email from context
func GetUserEmail(c *gin.Context) string {
	return c.GetString(UserEmailKey)
}

// MustGetTenantID extracts tenant ID from context or panics
func MustGetTenantID(c *gin.Context) string {
	tenantID := GetTenantID(c)
	if tenantID == "" {
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
			"code":    "UNAUTHORIZED",
			"message": "tenant ID not found in context",
		})
		return ""
	}
	return tenantID
}

// MustGetUserID extracts user ID from context or panics
func MustGetUserID(c *gin.Context) string {
	userID := GetUserID(c)
	if userID == "" {
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
			"code":    "UNAUTHORIZED",
			"message": "user ID not found in context",
		})
		return ""
	}
	return userID
}
