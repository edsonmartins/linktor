package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/msgfy/linktor/internal/application/service"
	"github.com/msgfy/linktor/internal/domain/entity"
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
	// APIKeyHeader is the header carrying a tenant API key for server-to-server
	// callers (e.g. the DeskLenz outbound path posting messages).
	APIKeyHeader = "X-API-Key"
	// APIKeyRole is the role assigned to requests authenticated via API key. It
	// passes the common protected routes but is intentionally NOT admin/owner, so
	// API keys cannot reach role-gated administrative endpoints.
	APIKeyRole = "api"
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

// AuthMiddleware handles JWT and API-key authentication
type AuthMiddleware struct {
	authService   *service.AuthService
	apiKeyService *service.APIKeyService // optional; enables X-API-Key auth
}

// NewAuthMiddleware creates a new auth middleware. apiKeyService may be nil to
// disable X-API-Key authentication.
func NewAuthMiddleware(authService *service.AuthService, apiKeyService *service.APIKeyService) *AuthMiddleware {
	return &AuthMiddleware{
		authService:   authService,
		apiKeyService: apiKeyService,
	}
}

// Authenticate returns a gin middleware that validates a JWT (Authorization
// Bearer header or access_token cookie) and, failing that, an X-API-Key header.
func (m *AuthMiddleware) Authenticate() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Token comes from the Authorization header (API/CLI) or the HttpOnly
		// access_token cookie (browser).
		if token := extractToken(c); token != "" {
			claims, err := m.authService.ValidateAccessToken(token)
			if err != nil {
				abortWithError(c, errors.New(errors.ErrCodeTokenInvalid, "invalid or expired token"))
				return
			}
			c.Set(TenantIDKey, claims.TenantID)
			c.Set(UserIDKey, claims.UserID)
			c.Set(UserRoleKey, claims.Role)
			c.Set(UserEmailKey, claims.Email)
			c.Next()
			return
		}

		// Server-to-server callers (e.g. DeskLenz) authenticate with a tenant API key.
		if apiKey := c.GetHeader(APIKeyHeader); apiKey != "" && m.apiKeyService != nil {
			key, err := m.apiKeyService.Authenticate(c.Request.Context(), apiKey)
			if err != nil {
				abortWithError(c, errors.Unauthorized("invalid API key"))
				return
			}
			c.Set(TenantIDKey, key.TenantID)
			c.Set(UserIDKey, apiKeyUserID(key))
			c.Set(UserRoleKey, APIKeyRole)
			c.Set(UserEmailKey, "apikey:"+key.Name)
			c.Next()
			return
		}

		abortWithError(c, errors.Unauthorized("missing authentication credentials"))
	}
}

// apiKeyUserID returns the acting user id for an API-key request: the key's bound
// user when present, otherwise a stable synthetic id derived from the key.
func apiKeyUserID(key *entity.APIKey) string {
	if key.UserID != nil && *key.UserID != "" {
		return *key.UserID
	}
	return "apikey:" + key.ID
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
