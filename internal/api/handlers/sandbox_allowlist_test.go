package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/msgfy/linktor/internal/api/middleware"
	"github.com/msgfy/linktor/internal/application/service"
	"github.com/msgfy/linktor/pkg/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newSandboxAllowlistRouter mirrors the production registration in main.go:
// the allowlist group runs behind RequireRole("admin", "owner"). The fake auth
// middleware stands in for JWT parsing only; role enforcement is the real one.
func newSandboxAllowlistRouter(role string) (*gin.Engine, *testutil.MockSandboxAllowlistRepository) {
	gin.SetMode(gin.TestMode)
	repo := testutil.NewMockSandboxAllowlistRepository()
	svc := service.NewSandboxAllowlistService(repo, testutil.NewMockChannelRepository())
	audit := service.NewAuditService(nil) // nil repo: Record is a safe no-op
	h := NewSandboxAllowlistHandler(svc, audit)

	router := gin.New()
	group := router.Group("/sandbox/allowlist")
	group.Use(func(c *gin.Context) {
		c.Set(middleware.TenantIDKey, "tenant1")
		c.Set(middleware.UserIDKey, "user1")
		c.Set(middleware.UserRoleKey, role)
	})
	authMw := middleware.NewAuthMiddleware(nil, nil)
	group.Use(authMw.RequireRole("admin", "owner"))
	{
		group.GET("", h.List)
		group.POST("", h.Add)
		group.DELETE("/:id", h.Remove)
	}
	return router, repo
}

func TestSandboxAllowlistRoutes_AdminCanAdd(t *testing.T) {
	router, repo := newSandboxAllowlistRouter("admin")

	body, _ := json.Marshal(AddSandboxAllowlistRequest{Recipient: "+55 44 99999-9999"})
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/sandbox/allowlist", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusCreated, w.Code, w.Body.String())
	require.Len(t, repo.Entries, 1)
	for _, e := range repo.Entries {
		assert.Equal(t, "+5544999999999", e.Recipient)
		assert.Equal(t, "tenant1", e.TenantID)
		assert.Equal(t, "user1", e.CreatedBy)
	}
}

func TestSandboxAllowlistRoutes_NonAdminIsRejected(t *testing.T) {
	router, repo := newSandboxAllowlistRouter("agent")

	body, _ := json.Marshal(AddSandboxAllowlistRequest{Recipient: "+5544999999999"})
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/sandbox/allowlist", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusForbidden, w.Code)
	assert.Empty(t, repo.Entries, "nothing may be written on a rejected request")
}
