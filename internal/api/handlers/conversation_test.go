package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/msgfy/linktor/internal/application/service"
	"github.com/msgfy/linktor/internal/domain/entity"
	"github.com/msgfy/linktor/pkg/testutil"
)

func init() {
	gin.SetMode(gin.TestMode)
}

// setupConversationHandler creates a ConversationHandler with mock repos and returns
// the handler plus the mock repos for test data setup.
func setupConversationHandler() (
	*ConversationHandler,
	*testutil.MockConversationRepository,
	*testutil.MockContactRepository,
	*testutil.MockChannelRepository,
) {
	convRepo := testutil.NewMockConversationRepository()
	contactRepo := testutil.NewMockContactRepository()
	channelRepo := testutil.NewMockChannelRepository()

	svc := service.NewConversationService(convRepo, contactRepo, channelRepo)
	handler := NewConversationHandler(svc, nil)

	return handler, convRepo, contactRepo, channelRepo
}

// newAuthContext creates a gin context with tenant_id and user_id already set,
// along with a recorder for inspecting responses.
func newAuthContext() (*gin.Context, *httptest.ResponseRecorder) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Set("tenant_id", "tenant-1")
	c.Set("user_id", "user-1")
	c.Request = httptest.NewRequest(http.MethodGet, "/", nil)
	return c, w
}

// seedConversation adds a conversation to the mock repo and returns it.
func seedConversation(repo *testutil.MockConversationRepository, id, tenantID string, status entity.ConversationStatus) *entity.Conversation {
	now := time.Now()
	conv := &entity.Conversation{
		ID:        id,
		TenantID:  tenantID,
		ContactID: "contact-1",
		ChannelID: "channel-1",
		Status:    status,
		Priority:  entity.ConversationPriorityNormal,
		Tags:      []string{},
		Metadata:  make(map[string]string),
		CreatedAt: now,
		UpdatedAt: now,
	}
	repo.Conversations[id] = conv
	return conv
}

// parseResponse unmarshals the recorder body into a Response struct.
func parseResponse(t *testing.T, w *httptest.ResponseRecorder) Response {
	t.Helper()
	var resp Response
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response body: %v", err)
	}
	return resp
}

// ---------------------------------------------------------------------------
// List
// ---------------------------------------------------------------------------

func TestList_ReturnsConversationsForTenant(t *testing.T) {
	handler, convRepo, _, _ := setupConversationHandler()

	seedConversation(convRepo, "conv-1", "tenant-1", entity.ConversationStatusOpen)
	seedConversation(convRepo, "conv-2", "tenant-1", entity.ConversationStatusPending)
	// conversation belonging to another tenant, should not appear
	seedConversation(convRepo, "conv-3", "tenant-other", entity.ConversationStatusOpen)

	c, w := newAuthContext()
	c.Request = httptest.NewRequest(http.MethodGet, "/conversations", nil)

	handler.List(c)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}

	resp := parseResponse(t, w)
	if !resp.Success {
		t.Fatal("expected success to be true")
	}
	if resp.Meta == nil {
		t.Fatal("expected meta to be present")
	}
	if resp.Meta.TotalItems != 2 {
		t.Fatalf("expected 2 total items, got %d", resp.Meta.TotalItems)
	}

	// Data should be a slice
	dataSlice, ok := resp.Data.([]interface{})
	if !ok {
		t.Fatalf("expected data to be a slice, got %T", resp.Data)
	}
	if len(dataSlice) != 2 {
		t.Fatalf("expected 2 conversations, got %d", len(dataSlice))
	}
}

func TestList_WithStatusFilter(t *testing.T) {
	handler, convRepo, _, _ := setupConversationHandler()

	seedConversation(convRepo, "conv-1", "tenant-1", entity.ConversationStatusOpen)
	seedConversation(convRepo, "conv-2", "tenant-1", entity.ConversationStatusResolved)

	c, w := newAuthContext()
	c.Request = httptest.NewRequest(http.MethodGet, "/conversations?status=open", nil)

	handler.List(c)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}

	resp := parseResponse(t, w)
	if !resp.Success {
		t.Fatal("expected success to be true")
	}
}

func TestList_NoTenantID_Returns401(t *testing.T) {
	handler, _, _, _ := setupConversationHandler()

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	// Deliberately do NOT set tenant_id
	c.Request = httptest.NewRequest(http.MethodGet, "/conversations", nil)

	handler.List(c)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected status 401, got %d", w.Code)
	}
}

// ---------------------------------------------------------------------------
// Get
// ---------------------------------------------------------------------------

func TestGet_ReturnsConversation(t *testing.T) {
	handler, convRepo, _, _ := setupConversationHandler()

	seedConversation(convRepo, "conv-1", "tenant-1", entity.ConversationStatusOpen)

	c, w := newAuthContext()
	c.Params = []gin.Param{{Key: "id", Value: "conv-1"}}

	handler.Get(c)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}

	resp := parseResponse(t, w)
	if !resp.Success {
		t.Fatal("expected success to be true")
	}

	data, ok := resp.Data.(map[string]interface{})
	if !ok {
		t.Fatal("expected data to be an object")
	}
	if data["id"] != "conv-1" {
		t.Fatalf("expected conversation id conv-1, got %v", data["id"])
	}
}

func TestGet_NotFound_Returns404(t *testing.T) {
	handler, _, _, _ := setupConversationHandler()

	c, w := newAuthContext()
	c.Params = []gin.Param{{Key: "id", Value: "nonexistent"}}

	handler.Get(c)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected status 404, got %d", w.Code)
	}

	resp := parseResponse(t, w)
	if resp.Success {
		t.Fatal("expected success to be false")
	}
}

func TestGet_EmptyID_Returns400(t *testing.T) {
	handler, _, _, _ := setupConversationHandler()

	c, w := newAuthContext()
	// No id param set
	c.Params = []gin.Param{}

	handler.Get(c)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", w.Code)
	}
}

// ---------------------------------------------------------------------------
// Create
// ---------------------------------------------------------------------------

func TestCreate_ValidData_Returns201(t *testing.T) {
	handler, _, contactRepo, channelRepo := setupConversationHandler()

	// Seed dependencies: contact and channel must exist
	contactRepo.Contacts["contact-1"] = &entity.Contact{
		ID:       "contact-1",
		TenantID: "tenant-1",
		Name:     "Test Contact",
	}
	channelRepo.Channels["channel-1"] = &entity.Channel{
		ID:       "channel-1",
		TenantID: "tenant-1",
		Name:     "Test Channel",
	}

	payload := CreateConversationRequest{
		ContactID: "contact-1",
		ChannelID: "channel-1",
		Subject:   "Test Subject",
		Priority:  "high",
		Tags:      []string{"tag1"},
	}
	body, _ := json.Marshal(payload)

	c, w := newAuthContext()
	c.Request = httptest.NewRequest(http.MethodPost, "/conversations", bytes.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")

	handler.Create(c)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected status 201, got %d; body: %s", w.Code, w.Body.String())
	}

	resp := parseResponse(t, w)
	if !resp.Success {
		t.Fatal("expected success to be true")
	}

	data, ok := resp.Data.(map[string]interface{})
	if !ok {
		t.Fatal("expected data to be an object")
	}
	if data["contact_id"] != "contact-1" {
		t.Fatalf("expected contact_id contact-1, got %v", data["contact_id"])
	}
	if data["subject"] != "Test Subject" {
		t.Fatalf("expected subject 'Test Subject', got %v", data["subject"])
	}
}

func TestCreate_InvalidBody_Returns400(t *testing.T) {
	handler, _, _, _ := setupConversationHandler()

	// Missing required fields
	body := []byte(`{"subject":"no contact or channel"}`)

	c, w := newAuthContext()
	c.Request = httptest.NewRequest(http.MethodPost, "/conversations", bytes.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")

	handler.Create(c)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", w.Code)
	}

	resp := parseResponse(t, w)
	if resp.Success {
		t.Fatal("expected success to be false")
	}
}

func TestCreate_NoTenantID_Returns401(t *testing.T) {
	handler, _, _, _ := setupConversationHandler()

	payload := CreateConversationRequest{
		ContactID: "contact-1",
		ChannelID: "channel-1",
	}
	body, _ := json.Marshal(payload)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	// No tenant_id set
	c.Request = httptest.NewRequest(http.MethodPost, "/conversations", bytes.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")

	handler.Create(c)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected status 401, got %d", w.Code)
	}
}

func TestCreate_ContactNotFound_ReturnsError(t *testing.T) {
	handler, _, _, channelRepo := setupConversationHandler()

	// Only channel exists, contact does not
	channelRepo.Channels["channel-1"] = &entity.Channel{
		ID:       "channel-1",
		TenantID: "tenant-1",
	}

	payload := CreateConversationRequest{
		ContactID: "nonexistent-contact",
		ChannelID: "channel-1",
	}
	body, _ := json.Marshal(payload)

	c, w := newAuthContext()
	c.Request = httptest.NewRequest(http.MethodPost, "/conversations", bytes.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")

	handler.Create(c)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected status 404, got %d; body: %s", w.Code, w.Body.String())
	}
}

// ---------------------------------------------------------------------------
// Update
// ---------------------------------------------------------------------------

func TestUpdate_ValidData_Returns200(t *testing.T) {
	handler, convRepo, _, _ := setupConversationHandler()

	seedConversation(convRepo, "conv-1", "tenant-1", entity.ConversationStatusOpen)

	newSubject := "Updated Subject"
	newPriority := "high"
	payload := UpdateConversationRequest{
		Subject:  &newSubject,
		Priority: &newPriority,
	}
	body, _ := json.Marshal(payload)

	c, w := newAuthContext()
	c.Params = []gin.Param{{Key: "id", Value: "conv-1"}}
	c.Request = httptest.NewRequest(http.MethodPut, "/conversations/conv-1", bytes.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")

	handler.Update(c)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d; body: %s", w.Code, w.Body.String())
	}

	resp := parseResponse(t, w)
	if !resp.Success {
		t.Fatal("expected success to be true")
	}

	data, ok := resp.Data.(map[string]interface{})
	if !ok {
		t.Fatal("expected data to be an object")
	}
	if data["subject"] != "Updated Subject" {
		t.Fatalf("expected subject 'Updated Subject', got %v", data["subject"])
	}
	if data["priority"] != "high" {
		t.Fatalf("expected priority 'high', got %v", data["priority"])
	}
}

func TestUpdate_NotFound_Returns404(t *testing.T) {
	handler, _, _, _ := setupConversationHandler()

	newSubject := "Updated Subject"
	payload := UpdateConversationRequest{
		Subject: &newSubject,
	}
	body, _ := json.Marshal(payload)

	c, w := newAuthContext()
	c.Params = []gin.Param{{Key: "id", Value: "nonexistent"}}
	c.Request = httptest.NewRequest(http.MethodPut, "/conversations/nonexistent", bytes.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")

	handler.Update(c)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected status 404, got %d", w.Code)
	}
}

func TestUpdate_EmptyID_Returns400(t *testing.T) {
	handler, _, _, _ := setupConversationHandler()

	body := []byte(`{}`)

	c, w := newAuthContext()
	c.Params = []gin.Param{}
	c.Request = httptest.NewRequest(http.MethodPut, "/conversations/", bytes.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")

	handler.Update(c)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", w.Code)
	}
}

func TestUpdate_InvalidBody_Returns400(t *testing.T) {
	handler, convRepo, _, _ := setupConversationHandler()

	seedConversation(convRepo, "conv-1", "tenant-1", entity.ConversationStatusOpen)

	c, w := newAuthContext()
	c.Params = []gin.Param{{Key: "id", Value: "conv-1"}}
	c.Request = httptest.NewRequest(http.MethodPut, "/conversations/conv-1", bytes.NewReader([]byte("not json")))
	c.Request.Header.Set("Content-Type", "application/json")

	handler.Update(c)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", w.Code)
	}
}

// ---------------------------------------------------------------------------
// Assign
// ---------------------------------------------------------------------------

func TestAssign_ValidData_Returns200(t *testing.T) {
	handler, convRepo, _, _ := setupConversationHandler()

	seedConversation(convRepo, "conv-1", "tenant-1", entity.ConversationStatusOpen)

	payload := AssignRequest{UserID: "user-42"}
	body, _ := json.Marshal(payload)

	c, w := newAuthContext()
	c.Params = []gin.Param{{Key: "id", Value: "conv-1"}}
	c.Request = httptest.NewRequest(http.MethodPost, "/conversations/conv-1/assign", bytes.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")

	handler.Assign(c)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d; body: %s", w.Code, w.Body.String())
	}

	resp := parseResponse(t, w)
	if !resp.Success {
		t.Fatal("expected success to be true")
	}

	data, ok := resp.Data.(map[string]interface{})
	if !ok {
		t.Fatal("expected data to be an object")
	}
	if data["assigned_user_id"] != "user-42" {
		t.Fatalf("expected assigned_user_id 'user-42', got %v", data["assigned_user_id"])
	}
}

func TestAssign_NotFound_Returns404(t *testing.T) {
	handler, _, _, _ := setupConversationHandler()

	payload := AssignRequest{UserID: "user-42"}
	body, _ := json.Marshal(payload)

	c, w := newAuthContext()
	c.Params = []gin.Param{{Key: "id", Value: "nonexistent"}}
	c.Request = httptest.NewRequest(http.MethodPost, "/conversations/nonexistent/assign", bytes.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")

	handler.Assign(c)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected status 404, got %d", w.Code)
	}
}

func TestAssign_InvalidBody_Returns400(t *testing.T) {
	handler, convRepo, _, _ := setupConversationHandler()

	seedConversation(convRepo, "conv-1", "tenant-1", entity.ConversationStatusOpen)

	// Missing required user_id
	body := []byte(`{}`)

	c, w := newAuthContext()
	c.Params = []gin.Param{{Key: "id", Value: "conv-1"}}
	c.Request = httptest.NewRequest(http.MethodPost, "/conversations/conv-1/assign", bytes.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")

	handler.Assign(c)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", w.Code)
	}
}

func TestAssign_EmptyID_Returns400(t *testing.T) {
	handler, _, _, _ := setupConversationHandler()

	payload := AssignRequest{UserID: "user-42"}
	body, _ := json.Marshal(payload)

	c, w := newAuthContext()
	c.Params = []gin.Param{}
	c.Request = httptest.NewRequest(http.MethodPost, "/conversations//assign", bytes.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")

	handler.Assign(c)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", w.Code)
	}
}

// ---------------------------------------------------------------------------
// Resolve
// ---------------------------------------------------------------------------

func TestResolve_OpenConversation_Returns200(t *testing.T) {
	handler, convRepo, _, _ := setupConversationHandler()

	seedConversation(convRepo, "conv-1", "tenant-1", entity.ConversationStatusOpen)

	c, w := newAuthContext()
	c.Params = []gin.Param{{Key: "id", Value: "conv-1"}}
	c.Request = httptest.NewRequest(http.MethodPost, "/conversations/conv-1/resolve", nil)

	handler.Resolve(c)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d; body: %s", w.Code, w.Body.String())
	}

	resp := parseResponse(t, w)
	if !resp.Success {
		t.Fatal("expected success to be true")
	}

	data, ok := resp.Data.(map[string]interface{})
	if !ok {
		t.Fatal("expected data to be an object")
	}
	if data["status"] != "resolved" {
		t.Fatalf("expected status 'resolved', got %v", data["status"])
	}
}

func TestResolve_AlreadyResolved_Returns400(t *testing.T) {
	handler, convRepo, _, _ := setupConversationHandler()

	seedConversation(convRepo, "conv-1", "tenant-1", entity.ConversationStatusResolved)

	c, w := newAuthContext()
	c.Params = []gin.Param{{Key: "id", Value: "conv-1"}}
	c.Request = httptest.NewRequest(http.MethodPost, "/conversations/conv-1/resolve", nil)

	handler.Resolve(c)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d; body: %s", w.Code, w.Body.String())
	}
}

func TestResolve_NotFound_Returns404(t *testing.T) {
	handler, _, _, _ := setupConversationHandler()

	c, w := newAuthContext()
	c.Params = []gin.Param{{Key: "id", Value: "nonexistent"}}
	c.Request = httptest.NewRequest(http.MethodPost, "/conversations/nonexistent/resolve", nil)

	handler.Resolve(c)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected status 404, got %d", w.Code)
	}
}

func TestResolve_EmptyID_Returns400(t *testing.T) {
	handler, _, _, _ := setupConversationHandler()

	c, w := newAuthContext()
	c.Params = []gin.Param{}
	c.Request = httptest.NewRequest(http.MethodPost, "/conversations//resolve", nil)

	handler.Resolve(c)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", w.Code)
	}
}

// ---------------------------------------------------------------------------
// Reopen
// ---------------------------------------------------------------------------

func TestReopen_ResolvedConversation_Returns200(t *testing.T) {
	handler, convRepo, _, _ := setupConversationHandler()

	seedConversation(convRepo, "conv-1", "tenant-1", entity.ConversationStatusResolved)

	c, w := newAuthContext()
	c.Params = []gin.Param{{Key: "id", Value: "conv-1"}}
	c.Request = httptest.NewRequest(http.MethodPost, "/conversations/conv-1/reopen", nil)

	handler.Reopen(c)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d; body: %s", w.Code, w.Body.String())
	}

	resp := parseResponse(t, w)
	if !resp.Success {
		t.Fatal("expected success to be true")
	}

	data, ok := resp.Data.(map[string]interface{})
	if !ok {
		t.Fatal("expected data to be an object")
	}
	if data["status"] != "open" {
		t.Fatalf("expected status 'open', got %v", data["status"])
	}
}

func TestReopen_AlreadyOpen_Returns400(t *testing.T) {
	handler, convRepo, _, _ := setupConversationHandler()

	seedConversation(convRepo, "conv-1", "tenant-1", entity.ConversationStatusOpen)

	c, w := newAuthContext()
	c.Params = []gin.Param{{Key: "id", Value: "conv-1"}}
	c.Request = httptest.NewRequest(http.MethodPost, "/conversations/conv-1/reopen", nil)

	handler.Reopen(c)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d; body: %s", w.Code, w.Body.String())
	}
}

func TestReopen_NotFound_Returns404(t *testing.T) {
	handler, _, _, _ := setupConversationHandler()

	c, w := newAuthContext()
	c.Params = []gin.Param{{Key: "id", Value: "nonexistent"}}
	c.Request = httptest.NewRequest(http.MethodPost, "/conversations/nonexistent/reopen", nil)

	handler.Reopen(c)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected status 404, got %d", w.Code)
	}
}

func TestReopen_EmptyID_Returns400(t *testing.T) {
	handler, _, _, _ := setupConversationHandler()

	c, w := newAuthContext()
	c.Params = []gin.Param{}
	c.Request = httptest.NewRequest(http.MethodPost, "/conversations//reopen", nil)

	handler.Reopen(c)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", w.Code)
	}
}

// ---------------------------------------------------------------------------
// GetEscalationContext (nil escalateUC)
// ---------------------------------------------------------------------------

func TestGetEscalationContext_NilUseCase_ReturnsError(t *testing.T) {
	handler, _, _, _ := setupConversationHandler()
	// handler.escalateUC is nil by default from setupConversationHandler

	c, w := newAuthContext()
	c.Params = []gin.Param{{Key: "id", Value: "conv-1"}}
	c.Request = httptest.NewRequest(http.MethodGet, "/conversations/conv-1/escalation-context", nil)

	handler.GetEscalationContext(c)

	// When escalateUC is nil, RespondError(c, nil) is called which maps to 500
	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected status 500, got %d; body: %s", w.Code, w.Body.String())
	}
}

func TestGetEscalationContext_EmptyID_Returns400(t *testing.T) {
	handler, _, _, _ := setupConversationHandler()

	c, w := newAuthContext()
	c.Params = []gin.Param{}
	c.Request = httptest.NewRequest(http.MethodGet, "/conversations//escalation-context", nil)

	handler.GetEscalationContext(c)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", w.Code)
	}
}

func TestGetEscalationContext_NoTenantID_Returns401(t *testing.T) {
	handler, _, _, _ := setupConversationHandler()

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Params = []gin.Param{{Key: "id", Value: "conv-1"}}
	c.Request = httptest.NewRequest(http.MethodGet, "/conversations/conv-1/escalation-context", nil)
	// No tenant_id set

	handler.GetEscalationContext(c)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected status 401, got %d", w.Code)
	}
}

// ---------------------------------------------------------------------------
// Escalate (nil escalateUC)
// ---------------------------------------------------------------------------

func TestEscalate_NilUseCase_AfterConversationFound_ReturnsError(t *testing.T) {
	handler, convRepo, _, _ := setupConversationHandler()

	seedConversation(convRepo, "conv-1", "tenant-1", entity.ConversationStatusOpen)

	payload := ConversationEscalateRequest{
		Reason:   "customer angry",
		Priority: "high",
	}
	body, _ := json.Marshal(payload)

	c, w := newAuthContext()
	c.Params = []gin.Param{{Key: "id", Value: "conv-1"}}
	c.Request = httptest.NewRequest(http.MethodPost, "/conversations/conv-1/escalate", bytes.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")

	handler.Escalate(c)

	// escalateUC is nil, so RespondError(c, nil) => 500
	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected status 500, got %d; body: %s", w.Code, w.Body.String())
	}
}

func TestEscalate_EmptyID_Returns400(t *testing.T) {
	handler, _, _, _ := setupConversationHandler()

	body := []byte(`{"reason":"test"}`)

	c, w := newAuthContext()
	c.Params = []gin.Param{}
	c.Request = httptest.NewRequest(http.MethodPost, "/conversations//escalate", bytes.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")

	handler.Escalate(c)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", w.Code)
	}
}

func TestEscalate_NoTenantID_Returns401(t *testing.T) {
	handler, _, _, _ := setupConversationHandler()

	body := []byte(`{"reason":"test"}`)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Params = []gin.Param{{Key: "id", Value: "conv-1"}}
	c.Request = httptest.NewRequest(http.MethodPost, "/conversations/conv-1/escalate", bytes.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")

	handler.Escalate(c)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected status 401, got %d", w.Code)
	}
}

func TestEscalate_InvalidBody_Returns400(t *testing.T) {
	handler, _, _, _ := setupConversationHandler()

	c, w := newAuthContext()
	c.Params = []gin.Param{{Key: "id", Value: "conv-1"}}
	c.Request = httptest.NewRequest(http.MethodPost, "/conversations/conv-1/escalate", bytes.NewReader([]byte("bad json")))
	c.Request.Header.Set("Content-Type", "application/json")

	handler.Escalate(c)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", w.Code)
	}
}

func TestEscalate_ConversationNotFound_Returns404(t *testing.T) {
	handler, _, _, _ := setupConversationHandler()

	payload := ConversationEscalateRequest{
		Reason:   "test",
		Priority: "normal",
	}
	body, _ := json.Marshal(payload)

	c, w := newAuthContext()
	c.Params = []gin.Param{{Key: "id", Value: "nonexistent"}}
	c.Request = httptest.NewRequest(http.MethodPost, "/conversations/nonexistent/escalate", bytes.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")

	handler.Escalate(c)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected status 404, got %d; body: %s", w.Code, w.Body.String())
	}
}

// ---------------------------------------------------------------------------
// Create - returns existing open conversation
// ---------------------------------------------------------------------------

func TestCreate_ExistingOpenConversation_ReturnsExisting(t *testing.T) {
	handler, convRepo, contactRepo, channelRepo := setupConversationHandler()

	contactRepo.Contacts["contact-1"] = &entity.Contact{
		ID:       "contact-1",
		TenantID: "tenant-1",
		Name:     "Test Contact",
	}
	channelRepo.Channels["channel-1"] = &entity.Channel{
		ID:       "channel-1",
		TenantID: "tenant-1",
		Name:     "Test Channel",
	}

	// Seed an existing open conversation for the same contact+channel
	existing := seedConversation(convRepo, "existing-conv", "tenant-1", entity.ConversationStatusOpen)
	existing.ContactID = "contact-1"
	existing.ChannelID = "channel-1"

	payload := CreateConversationRequest{
		ContactID: "contact-1",
		ChannelID: "channel-1",
		Subject:   "New Subject",
	}
	body, _ := json.Marshal(payload)

	c, w := newAuthContext()
	c.Request = httptest.NewRequest(http.MethodPost, "/conversations", bytes.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")

	handler.Create(c)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected status 201, got %d; body: %s", w.Code, w.Body.String())
	}

	resp := parseResponse(t, w)
	data, ok := resp.Data.(map[string]interface{})
	if !ok {
		t.Fatal("expected data to be an object")
	}
	// Should return the existing conversation ID
	if data["id"] != "existing-conv" {
		t.Fatalf("expected existing conversation id 'existing-conv', got %v", data["id"])
	}
}

// ---------------------------------------------------------------------------
// Update - status change
// ---------------------------------------------------------------------------

func TestUpdate_ChangeStatus_Returns200(t *testing.T) {
	handler, convRepo, _, _ := setupConversationHandler()

	seedConversation(convRepo, "conv-1", "tenant-1", entity.ConversationStatusOpen)

	newStatus := "pending"
	payload := UpdateConversationRequest{
		Status: &newStatus,
	}
	body, _ := json.Marshal(payload)

	c, w := newAuthContext()
	c.Params = []gin.Param{{Key: "id", Value: "conv-1"}}
	c.Request = httptest.NewRequest(http.MethodPut, "/conversations/conv-1", bytes.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")

	handler.Update(c)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d; body: %s", w.Code, w.Body.String())
	}

	resp := parseResponse(t, w)
	data, ok := resp.Data.(map[string]interface{})
	if !ok {
		t.Fatal("expected data to be an object")
	}
	if data["status"] != "pending" {
		t.Fatalf("expected status 'pending', got %v", data["status"])
	}
}
