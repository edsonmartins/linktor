package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/msgfy/linktor/internal/api/middleware"
	"github.com/msgfy/linktor/internal/application/service"
	"github.com/msgfy/linktor/internal/domain/entity"
	"github.com/msgfy/linktor/pkg/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newOwnerRouter mirrors the production registration in main.go: the owner
// routes run behind RequireRole("admin","owner",api) and RequireScopeByMethod
// (routing:read, routing:write). The fake auth sets tenant, role and the
// API-key scopes; role and scope enforcement are the real middlewares.
func newOwnerRouter(t *testing.T, tenantID string, scopes []string) (*gin.Engine, *testutil.MockConversationOwnerRuleRepository) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	repo := testutil.NewMockConversationOwnerRuleRepository()
	channels := testutil.NewMockChannelRepository()
	channels.Channels["ch-a"] = &entity.Channel{ID: "ch-a", TenantID: "tenant-a", Type: entity.ChannelTypeWhatsAppOfficial}
	channels.Channels["ch-b"] = &entity.Channel{ID: "ch-b", TenantID: "tenant-b", Type: entity.ChannelTypeWhatsAppOfficial}
	h := NewConversationOwnerHandler(service.NewConversationOwnerService(repo, channels), service.NewAuditService(nil))

	router := gin.New()
	group := router.Group("/channels/:id/contacts/:identifier/owner")
	group.Use(func(c *gin.Context) {
		c.Set(middleware.TenantIDKey, tenantID)
		c.Set(middleware.UserRoleKey, middleware.APIKeyRole)
		c.Set(middleware.ScopesKey, scopes)
	})
	authMw := middleware.NewAuthMiddleware(nil, nil)
	group.Use(authMw.RequireRole("admin", "owner", middleware.APIKeyRole))
	group.Use(authMw.RequireScopeByMethod(middleware.ScopeRoutingRead, middleware.ScopeRoutingWrite))
	group.GET("", h.Get)
	group.PUT("", h.Put)
	group.DELETE("", h.Delete)
	return router, repo
}

func do(router *gin.Engine, method, path string, body interface{}) *httptest.ResponseRecorder {
	var buf bytes.Buffer
	if body != nil {
		_ = json.NewEncoder(&buf).Encode(body)
	}
	req := httptest.NewRequest(method, path, &buf)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	return w
}

type envelope struct {
	Data json.RawMessage `json:"data"`
}

func dataOf(t *testing.T, w *httptest.ResponseRecorder, v interface{}) {
	t.Helper()
	var env envelope
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &env), w.Body.String())
	require.NoError(t, json.Unmarshal(env.Data, v), w.Body.String())
}

const ownerPath = "/channels/ch-a/contacts/+55%2011%2098765-4321/owner"

func declarar(consumidor, ref string, desde time.Time) map[string]interface{} {
	return map[string]interface{}{"consumidor": consumidor, "conversa_ref": ref, "desde": desde.Format(time.RFC3339)}
}

func TestConversationOwner_PutGetDelete(t *testing.T) {
	router, repo := newOwnerRouter(t, "tenant-a", []string{middleware.ScopeRoutingWrite})
	desde := time.Date(2026, 11, 4, 12, 0, 0, 0, time.UTC)

	w := do(router, http.MethodPut, ownerPath, declarar("copilot", "conv-1", desde))
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	var put DeclareOwnerResponse
	dataOf(t, w, &put)
	assert.True(t, put.Aplicada)
	assert.Equal(t, "+5511987654321", put.Regra.ContactIdentifier, "chave normalizada em E.164")
	assert.Equal(t, "copilot", put.Regra.Consumer)
	require.Len(t, repo.Rules, 1)

	// Idempotente: a mesma declaração de novo aplica e não duplica.
	w = do(router, http.MethodPut, ownerPath, declarar("copilot", "conv-1", desde))
	require.Equal(t, http.StatusOK, w.Code)
	assert.Len(t, repo.Rules, 1)

	w = do(router, http.MethodGet, "/channels/ch-a/contacts/5511987654321/owner", nil)
	require.Equal(t, http.StatusOK, w.Code, "o mesmo contato, em outro formato")
	var got entity.ConversationOwnerRule
	dataOf(t, w, &got)
	assert.Equal(t, "conv-1", got.ConversaRef)

	// Liberação de outra conversa não apaga a regra.
	w = do(router, http.MethodDelete, ownerPath+"?conversa_ref=conv-antiga", nil)
	require.Equal(t, http.StatusOK, w.Code)
	var rel ReleaseOwnerResponse
	dataOf(t, w, &rel)
	assert.False(t, rel.Liberada)
	assert.Len(t, repo.Rules, 1)

	// Com a conversa certa (no corpo), libera; de novo, nada a liberar (idempotente).
	w = do(router, http.MethodDelete, ownerPath, map[string]string{"conversa_ref": "conv-1"})
	dataOf(t, w, &rel)
	assert.True(t, rel.Liberada)
	assert.Empty(t, repo.Rules)
	w = do(router, http.MethodDelete, ownerPath+"?conversa_ref=conv-1", nil)
	dataOf(t, w, &rel)
	assert.False(t, rel.Liberada)
	assert.Equal(t, http.StatusNotFound, do(router, http.MethodGet, ownerPath, nil).Code)
}

func TestConversationOwner_DeclaracaoAtrasadaNaoSobrescreve(t *testing.T) {
	router, _ := newOwnerRouter(t, "tenant-a", []string{middleware.ScopeRoutingWrite})
	nova := time.Date(2026, 11, 4, 12, 0, 0, 0, time.UTC)
	require.Equal(t, http.StatusOK, do(router, http.MethodPut, ownerPath, declarar("crm", "conv-nova", nova)).Code)

	w := do(router, http.MethodPut, ownerPath, declarar("copilot", "conv-velha", nova.Add(-time.Hour)))
	require.Equal(t, http.StatusOK, w.Code)
	var put DeclareOwnerResponse
	dataOf(t, w, &put)
	assert.False(t, put.Aplicada)
	assert.Equal(t, "conv-nova", put.Regra.ConversaRef)
	assert.Equal(t, "crm", put.Regra.Consumer)

	// Uma conversa mais nova, de outro dono, aplica.
	w = do(router, http.MethodPut, ownerPath, declarar("copilot", "conv-seguinte", nova.Add(time.Hour)))
	dataOf(t, w, &put)
	assert.True(t, put.Aplicada)
	assert.Equal(t, "copilot", put.Regra.Consumer)
}

func TestConversationOwner_RegraExpiradaNaoVale(t *testing.T) {
	router, _ := newOwnerRouter(t, "tenant-a", []string{middleware.ScopeRoutingWrite})
	desde := time.Now().Add(-2 * time.Hour).UTC()
	body := declarar("copilot", "conv-1", desde)
	body["expira_em"] = time.Now().Add(-time.Hour).UTC().Format(time.RFC3339)
	require.Equal(t, http.StatusOK, do(router, http.MethodPut, ownerPath, body).Code)
	assert.Equal(t, http.StatusNotFound, do(router, http.MethodGet, ownerPath, nil).Code)
}

func TestConversationOwner_Escopo(t *testing.T) {
	desde := time.Now().UTC()
	leitura, _ := newOwnerRouter(t, "tenant-a", []string{middleware.ScopeRoutingRead})
	assert.Equal(t, http.StatusForbidden, do(leitura, http.MethodPut, ownerPath, declarar("crm", "c", desde)).Code)
	assert.Equal(t, http.StatusForbidden, do(leitura, http.MethodDelete, ownerPath+"?conversa_ref=c", nil).Code)
	assert.Equal(t, http.StatusNotFound, do(leitura, http.MethodGet, ownerPath, nil).Code, "leitura passa pelo escopo")

	canais, _ := newOwnerRouter(t, "tenant-a", []string{middleware.ScopeChannelsWrite, middleware.ScopeMessagesSend})
	assert.Equal(t, http.StatusForbidden, do(canais, http.MethodPut, ownerPath, declarar("crm", "c", desde)).Code,
		"channels:write não dá routing:write")
	assert.Equal(t, http.StatusForbidden, do(canais, http.MethodGet, ownerPath, nil).Code)
}

func TestConversationOwner_CanalDeOutroTenant(t *testing.T) {
	router, repo := newOwnerRouter(t, "tenant-a", []string{middleware.ScopeRoutingWrite})
	outro := "/channels/ch-b/contacts/+5511987654321/owner"
	assert.Equal(t, http.StatusNotFound, do(router, http.MethodPut, outro, declarar("crm", "c", time.Now())).Code)
	assert.Equal(t, http.StatusNotFound, do(router, http.MethodGet, outro, nil).Code)
	assert.Equal(t, http.StatusNotFound, do(router, http.MethodDelete, outro+"?conversa_ref=c", nil).Code)
	assert.Equal(t, http.StatusNotFound, do(router, http.MethodPut, "/channels/nao-existe/contacts/x/owner", declarar("crm", "c", time.Now())).Code)
	assert.Empty(t, repo.Rules)
}

func TestConversationOwner_Validacao(t *testing.T) {
	router, repo := newOwnerRouter(t, "tenant-a", []string{middleware.ScopeRoutingWrite})
	desde := time.Now().UTC()
	assert.Equal(t, http.StatusBadRequest, do(router, http.MethodPut, ownerPath,
		map[string]string{"consumidor": "crm", "conversa_ref": "c"}).Code, "sem desde")
	assert.Equal(t, http.StatusBadRequest, do(router, http.MethodPut, ownerPath, declarar("CRM!", "c", desde)).Code)
	body := declarar("crm", "c", desde)
	body["expira_em"] = desde.Add(-time.Minute).Format(time.RFC3339)
	assert.Equal(t, http.StatusBadRequest, do(router, http.MethodPut, ownerPath, body).Code, "expira antes de começar")
	assert.Equal(t, http.StatusBadRequest, do(router, http.MethodDelete, ownerPath, nil).Code, "sem conversa_ref")
	assert.Empty(t, repo.Rules)
}

func TestConversationOwner_IdentificadorNaoTelefone(t *testing.T) {
	router, repo := newOwnerRouter(t, "tenant-a", []string{middleware.ScopeRoutingWrite})
	w := do(router, http.MethodPut, "/channels/ch-a/contacts/cliente@exemplo.test/owner", declarar("crm", "c", time.Now()))
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	for _, r := range repo.Rules {
		assert.Equal(t, "cliente@exemplo.test", r.ContactIdentifier, "todos os tipos de canal (decisão iv)")
	}
}
