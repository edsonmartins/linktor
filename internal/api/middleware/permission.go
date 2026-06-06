package middleware

import (
	"github.com/gin-gonic/gin"
	"github.com/msgfy/linktor/internal/application/service"
	"github.com/msgfy/linktor/pkg/errors"
)

// PermissionMiddleware enforces fine-grained (resource, action) permissions
// resolved from the authenticated user's role via RoleService.
type PermissionMiddleware struct {
	roleService *service.RoleService
}

// NewPermissionMiddleware creates a new PermissionMiddleware.
func NewPermissionMiddleware(roleService *service.RoleService) *PermissionMiddleware {
	return &PermissionMiddleware{roleService: roleService}
}

// Require returns a gin middleware that aborts with 403 unless the caller's role
// grants (resource, action).
func (m *PermissionMiddleware) Require(resource, action string) gin.HandlerFunc {
	return func(c *gin.Context) {
		tenantID := GetTenantID(c)
		role := GetUserRole(c)
		if tenantID == "" || role == "" {
			abortWithError(c, errors.Unauthorized("missing authentication context"))
			return
		}

		if !m.roleService.HasPermission(c.Request.Context(), tenantID, role, resource, action) {
			abortWithError(c, errors.Forbidden("insufficient permissions"))
			return
		}

		c.Next()
	}
}
