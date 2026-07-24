package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/msgfy/linktor/internal/api/middleware"
	"github.com/msgfy/linktor/internal/application/service"
	"github.com/msgfy/linktor/internal/domain/entity"
	"github.com/msgfy/linktor/internal/domain/repository"
	"github.com/msgfy/linktor/pkg/plugin"
	"github.com/msgfy/linktor/pkg/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// captureAuditRepo records audit entries for assertions.
type captureAuditRepo struct{ logs []*entity.AuditLog }

func (c *captureAuditRepo) Create(ctx context.Context, log *entity.AuditLog) error {
	c.logs = append(c.logs, log)
	return nil
}
func (c *captureAuditRepo) FindByTenant(ctx context.Context, tenantID string, filters *repository.AuditLogFilters, params *repository.ListParams) ([]*entity.AuditLog, int64, error) {
	return c.logs, int64(len(c.logs)), nil
}

func newChannelAuditRouter() (*gin.Engine, *captureAuditRepo, *testutil.MockChannelRepository) {
	gin.SetMode(gin.TestMode)
	channelRepo := testutil.NewMockChannelRepository()
	channelSvc := service.NewChannelService(channelRepo, plugin.NewRegistry(), testutil.NewMockProducer())
	auditRepo := &captureAuditRepo{}
	h := NewChannelHandler(channelSvc, testutil.NewMockProducer(), service.NewAuditService(auditRepo))

	router := gin.New()
	group := router.Group("/channels")
	group.Use(func(c *gin.Context) {
		c.Set(middleware.TenantIDKey, "tenant1")
		c.Set(middleware.UserIDKey, "user1")
	})
	group.POST("", h.Create)
	group.PUT("/:id", h.Update)
	return router, auditRepo, channelRepo
}

func postChannel(router *gin.Engine, body map[string]any) *httptest.ResponseRecorder {
	raw, _ := json.Marshal(body)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/channels", bytes.NewReader(raw))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)
	return w
}

func TestChannelAudit_CreateRecordsDeclarationsWithoutSecrets(t *testing.T) {
	router, audit, _ := newChannelAuditRouter()

	w := postChannel(router, map[string]any{
		"type": "whatsapp_official", "name": "sb", "environment": "sandbox",
		"config": map[string]string{
			"phone_number_id":               "111",
			"sandbox_test_phone_number_ids": "111",
		},
		"credentials": map[string]string{
			"access_token":           "SUPER-SECRET-TOKEN",
			"credential_environment": "sandbox",
		},
	})

	require.Equal(t, http.StatusCreated, w.Code, w.Body.String())
	require.Len(t, audit.logs, 1)
	log := audit.logs[0]
	assert.Equal(t, "channel.create", log.Action)
	assert.Equal(t, "user1", log.ActorID)
	assert.Equal(t, "sandbox", log.Changes["environment"])
	assert.Equal(t, "sandbox", log.Changes["credential_environment"])
	assert.Equal(t, "111", log.Changes["phone_number_id"])

	serialized := fmt.Sprintf("%v", log.Changes)
	assert.NotContains(t, serialized, "SUPER-SECRET-TOKEN",
		"credential values must never enter the audit trail (INV-002)")
}

func TestChannelAudit_EnvironmentRejectionIsAudited(t *testing.T) {
	router, audit, repo := newChannelAuditRouter()

	// Sandbox without credential declaration → environment-binding rejection.
	w := postChannel(router, map[string]any{
		"type": "whatsapp_official", "name": "sb", "environment": "sandbox",
		"credentials": map[string]string{"access_token": "SUPER-SECRET-TOKEN"},
	})

	require.Equal(t, http.StatusBadRequest, w.Code, w.Body.String())
	assert.Empty(t, repo.Channels, "nothing persisted")
	require.Len(t, audit.logs, 1)
	log := audit.logs[0]
	assert.Equal(t, "channel.create_rejected", log.Action)
	assert.Contains(t, log.Changes["rejection_reason"], service.CredentialEnvironmentKey)
	assert.NotContains(t, fmt.Sprintf("%v", log.Changes), "SUPER-SECRET-TOKEN")
}

func TestChannelAudit_OrdinaryValidationIsNotAudited(t *testing.T) {
	router, audit, _ := newChannelAuditRouter()

	// Missing required name → plain binding rejection, no trail entry.
	w := postChannel(router, map[string]any{"type": "webchat"})

	require.Equal(t, http.StatusBadRequest, w.Code)
	assert.Empty(t, audit.logs, "ordinary validation noise must not pollute the trail")
}

func TestChannelAudit_UpdateAndImmutabilityRejection(t *testing.T) {
	router, audit, repo := newChannelAuditRouter()

	w := postChannel(router, map[string]any{
		"type": "webchat", "name": "sb", "environment": "sandbox",
		"credentials": map[string]string{"credential_environment": "sandbox"},
	})
	require.Equal(t, http.StatusCreated, w.Code, w.Body.String())
	var chID string
	for id := range repo.Channels {
		chID = id
	}
	audit.logs = nil

	// Successful update → channel.update with the delta.
	raw, _ := json.Marshal(map[string]any{"type": "webchat", "name": "renamed"})
	w = httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPut, "/channels/"+chID, bytes.NewReader(raw))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	require.Len(t, audit.logs, 1)
	assert.Equal(t, "channel.update", audit.logs[0].Action)
	assert.Equal(t, "renamed", audit.logs[0].Changes["name"])

	// Attempt to flip environment → channel.update_rejected.
	audit.logs = nil
	raw, _ = json.Marshal(map[string]any{"type": "webchat", "name": "sb", "environment": "production"})
	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPut, "/channels/"+chID, bytes.NewReader(raw))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)
	require.Equal(t, http.StatusBadRequest, w.Code, w.Body.String())
	require.Len(t, audit.logs, 1)
	assert.Equal(t, "channel.update_rejected", audit.logs[0].Action)
	assert.Contains(t, audit.logs[0].Changes["rejection_reason"], "immutable")
	assert.Equal(t, entity.ChannelEnvironmentSandbox, repo.Channels[chID].Environment,
		"persisted environment unchanged after rejected update")
}
