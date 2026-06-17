package middleware

import (
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/msgfy/linktor/internal/application/service"
)

// AuditMiddleware records an audit entry for every successful mutating request
// on the routes it wraps. Action and resource are derived from the route, so
// new endpoints are covered automatically without per-handler wiring.
type AuditMiddleware struct {
	audit *service.AuditService
}

// NewAuditMiddleware creates a new AuditMiddleware.
func NewAuditMiddleware(audit *service.AuditService) *AuditMiddleware {
	return &AuditMiddleware{audit: audit}
}

// Record returns a gin middleware that, after the handler runs, persists an
// audit entry when the request was a successful (2xx) mutation.
func (m *AuditMiddleware) Record() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()

		switch c.Request.Method {
		case "GET", "HEAD", "OPTIONS":
			return
		}
		if status := c.Writer.Status(); status < 200 || status >= 300 {
			return
		}
		tenantID := GetTenantID(c)
		if tenantID == "" {
			return
		}

		resource, resourceID, action := deriveAudit(c)
		actor := service.Actor{
			ID:    GetUserID(c),
			Email: GetUserEmail(c),
			IP:    c.ClientIP(),
			Agent: c.Request.UserAgent(),
		}
		m.audit.Record(c.Request.Context(), tenantID, actor, action, resource, resourceID, nil)
	}
}

// deriveAudit infers (resource, resourceID, action) from the matched route.
// Examples:
//
//	POST   /api/v1/channels            -> ("channel", "",   "channel.create")
//	PUT    /api/v1/channels/:id        -> ("channel", "<id>", "channel.update")
//	DELETE /api/v1/contacts/:id        -> ("contact", "<id>", "contact.delete")
//	POST   /api/v1/channels/:id/connect-> ("channel", "<id>", "channel.connect")
func deriveAudit(c *gin.Context) (resource, resourceID, action string) {
	return deriveAuditFromPath(c.Request.Method, c.FullPath(), firstParam(c))
}

// deriveAuditFromPath is the pure core of deriveAudit, split out for testing.
func deriveAuditFromPath(method, fullPath, id string) (resource, resourceID, action string) {
	parts := make([]string, 0, 4)
	for _, p := range strings.Split(fullPath, "/") {
		if p == "" || p == "api" || p == "v1" {
			continue
		}
		parts = append(parts, p)
	}
	if len(parts) == 0 {
		return "request", "", "request." + verbFor(method)
	}

	resource = singular(parts[0])
	resourceID = id

	last := parts[len(parts)-1]
	if len(parts) > 1 && !strings.HasPrefix(last, ":") {
		// Trailing literal segment is the explicit action (connect, disconnect…).
		action = resource + "." + last
	} else {
		action = resource + "." + verbFor(method)
	}
	return resource, resourceID, action
}

func verbFor(method string) string {
	switch method {
	case "POST":
		return "create"
	case "PUT", "PATCH":
		return "update"
	case "DELETE":
		return "delete"
	default:
		return strings.ToLower(method)
	}
}

func singular(s string) string {
	if strings.HasSuffix(s, "ies") {
		return strings.TrimSuffix(s, "ies") + "y"
	}
	if strings.HasSuffix(s, "s") {
		return strings.TrimSuffix(s, "s")
	}
	return s
}

func firstParam(c *gin.Context) string {
	if id := c.Param("id"); id != "" {
		return id
	}
	if len(c.Params) > 0 {
		return c.Params[0].Value
	}
	return ""
}
