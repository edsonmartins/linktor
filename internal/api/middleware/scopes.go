package middleware

import (
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/msgfy/linktor/pkg/errors"
)

// ScopesKey is the context key holding the granted scopes of an API-key request.
// It is set ONLY when the caller authenticated with an X-API-Key; JWT (human)
// requests never set it, so RequireScope is a no-op for them — humans are gated
// by role, not by scope.
const ScopesKey = "api_scopes"

// Scope vocabulary (resource:action). "*" grants everything (the default for
// keys created before scopes existed, so enforcement is not a breaking change).
// A "resource:*" grants every action on that resource, and "resource:write"
// implies "resource:read".
const (
	ScopeAll = "*"

	ScopeChannelsRead  = "channels:read"
	ScopeChannelsWrite = "channels:write"

	ScopeMessagesSend = "messages:send"

	ScopeContactsRead  = "contacts:read"
	ScopeContactsWrite = "contacts:write"

	ScopeConversationsRead  = "conversations:read"
	ScopeConversationsWrite = "conversations:write"

	// Demais recursos da API. Sem eles, uma chave restrita a "channels:*" + "messages:send"
	// continuava lendo bots, fluxos, templates, analytics, base de conhecimento e pedidos: o painel
	// oferecia a restrição, mas o gate só existia em canais e no envio de mensagens.
	ScopeBotsRead  = "bots:read"
	ScopeBotsWrite = "bots:write"

	ScopeFlowsRead  = "flows:read"
	ScopeFlowsWrite = "flows:write"

	ScopeTemplatesRead  = "templates:read"
	ScopeTemplatesWrite = "templates:write"

	ScopeKnowledgeRead  = "knowledge:read"
	ScopeKnowledgeWrite = "knowledge:write"

	ScopeOrdersRead  = "orders:read"
	ScopeOrdersWrite = "orders:write"

	ScopeAnalyticsRead = "analytics:read"

	ScopeAiUse = "ai:use"
)

// HasScope reports whether the granted scopes satisfy the required scope.
// Matching rules (most permissive first):
//   - "*"                → satisfies anything
//   - exact match        → "channels:write" satisfies "channels:write"
//   - "resource:*"       → satisfies any action on that resource
//   - write implies read → "channels:write" satisfies "channels:read"
func HasScope(granted []string, required string) bool {
	resource, action, _ := strings.Cut(required, ":")
	for _, g := range granted {
		switch {
		case g == ScopeAll, g == required, g == resource+":*":
			return true
		case action == "read" && g == resource+":write":
			return true
		}
	}
	return false
}

// GrantedScopes returns the scopes of the current request, and whether the
// request was authenticated by an API key at all. When ok is false the caller
// is a human (JWT) and scope checks do not apply.
func GrantedScopes(c *gin.Context) (scopes []string, ok bool) {
	raw, exists := c.Get(ScopesKey)
	if !exists {
		return nil, false
	}
	s, _ := raw.([]string)
	return s, true
}

// RequireScope returns middleware enforcing that an API-key request carries the
// given scope. JWT (human) requests pass through untouched — they are governed
// by RequireRole, not scopes.
func (m *AuthMiddleware) RequireScope(required string) gin.HandlerFunc {
	return func(c *gin.Context) {
		scopes, isAPIKey := GrantedScopes(c)
		if !isAPIKey {
			c.Next()
			return
		}
		if !HasScope(scopes, required) {
			abortWithError(c, errors.Forbidden("API key is missing the required scope: "+required))
			return
		}
		c.Next()
	}
}

// RequireScopeByMethod enforces readScope on safe (GET/HEAD) requests and
// writeScope on mutating ones (POST/PUT/PATCH/DELETE). Applied once at a route
// group, it scopes every current and future route in that group, so a new
// endpoint cannot silently escape scope enforcement.
func (m *AuthMiddleware) RequireScopeByMethod(readScope, writeScope string) gin.HandlerFunc {
	return func(c *gin.Context) {
		required := writeScope
		if c.Request.Method == "GET" || c.Request.Method == "HEAD" {
			required = readScope
		}
		m.RequireScope(required)(c)
	}
}
