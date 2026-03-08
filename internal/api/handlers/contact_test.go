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

// setupContactHandler creates a ContactHandler with a mock repo and returns both.
func setupContactHandler() (*ContactHandler, *testutil.MockContactRepository) {
	repo := testutil.NewMockContactRepository()
	svc := service.NewContactService(repo)
	handler := NewContactHandler(svc)
	return handler, repo
}

// newContactAuthContext creates a gin context with tenant_id and user_id set.
func newContactAuthContext() (*gin.Context, *httptest.ResponseRecorder) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Set("tenant_id", "tenant-1")
	c.Set("user_id", "user-1")
	c.Request = httptest.NewRequest(http.MethodGet, "/", nil)
	return c, w
}

// seedContact adds a contact to the mock repo and returns it.
func seedContact(repo *testutil.MockContactRepository, id, tenantID, name, email, phone string) *entity.Contact {
	now := time.Now()
	contact := &entity.Contact{
		ID:           id,
		TenantID:     tenantID,
		Name:         name,
		Email:        email,
		Phone:        phone,
		CustomFields: make(map[string]string),
		Tags:         []string{},
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	repo.Contacts[id] = contact
	return contact
}

// parseContactResponse unmarshals the recorder body into a Response struct.
func parseContactResponse(t *testing.T, w *httptest.ResponseRecorder) Response {
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

func TestContactList_ReturnsContactsForTenant(t *testing.T) {
	handler, repo := setupContactHandler()

	seedContact(repo, "c-1", "tenant-1", "Alice", "alice@example.com", "+1111")
	seedContact(repo, "c-2", "tenant-1", "Bob", "bob@example.com", "+2222")
	// Different tenant -- should not appear
	seedContact(repo, "c-3", "tenant-other", "Charlie", "charlie@example.com", "+3333")

	c, w := newContactAuthContext()
	c.Request = httptest.NewRequest(http.MethodGet, "/contacts", nil)

	handler.List(c)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}

	resp := parseContactResponse(t, w)
	if !resp.Success {
		t.Fatal("expected success to be true")
	}
	if resp.Meta == nil {
		t.Fatal("expected meta to be present")
	}
	if resp.Meta.TotalItems != 2 {
		t.Fatalf("expected 2 total items, got %d", resp.Meta.TotalItems)
	}

	dataSlice, ok := resp.Data.([]interface{})
	if !ok {
		t.Fatalf("expected data to be a slice, got %T", resp.Data)
	}
	if len(dataSlice) != 2 {
		t.Fatalf("expected 2 contacts, got %d", len(dataSlice))
	}
}

func TestContactList_EmptyResult(t *testing.T) {
	handler, _ := setupContactHandler()

	c, w := newContactAuthContext()
	c.Request = httptest.NewRequest(http.MethodGet, "/contacts", nil)

	handler.List(c)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}

	resp := parseContactResponse(t, w)
	if !resp.Success {
		t.Fatal("expected success to be true")
	}
	if resp.Meta.TotalItems != 0 {
		t.Fatalf("expected 0 total items, got %d", resp.Meta.TotalItems)
	}
}

func TestContactList_NoTenantID_Returns401(t *testing.T) {
	handler, _ := setupContactHandler()

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/contacts", nil)

	handler.List(c)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected status 401, got %d", w.Code)
	}
}

// ---------------------------------------------------------------------------
// Get
// ---------------------------------------------------------------------------

func TestContactGet_ReturnsContact(t *testing.T) {
	handler, repo := setupContactHandler()

	seedContact(repo, "c-1", "tenant-1", "Alice", "alice@example.com", "+1111")

	c, w := newContactAuthContext()
	c.Params = []gin.Param{{Key: "id", Value: "c-1"}}

	handler.Get(c)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}

	resp := parseContactResponse(t, w)
	if !resp.Success {
		t.Fatal("expected success to be true")
	}

	data, ok := resp.Data.(map[string]interface{})
	if !ok {
		t.Fatal("expected data to be an object")
	}
	if data["id"] != "c-1" {
		t.Fatalf("expected contact id c-1, got %v", data["id"])
	}
	if data["name"] != "Alice" {
		t.Fatalf("expected name Alice, got %v", data["name"])
	}
}

func TestContactGet_NotFound_Returns404(t *testing.T) {
	handler, _ := setupContactHandler()

	c, w := newContactAuthContext()
	c.Params = []gin.Param{{Key: "id", Value: "nonexistent"}}

	handler.Get(c)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected status 404, got %d", w.Code)
	}

	resp := parseContactResponse(t, w)
	if resp.Success {
		t.Fatal("expected success to be false")
	}
}

func TestContactGet_EmptyID_Returns400(t *testing.T) {
	handler, _ := setupContactHandler()

	c, w := newContactAuthContext()
	c.Params = []gin.Param{}

	handler.Get(c)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", w.Code)
	}
}

func TestContactGet_LoadsIdentities(t *testing.T) {
	handler, repo := setupContactHandler()

	seedContact(repo, "c-1", "tenant-1", "Alice", "alice@example.com", "+1111")
	repo.Identities["c-1"] = []*entity.ContactIdentity{
		{
			ID:          "id-1",
			ContactID:   "c-1",
			ChannelType: "whatsapp",
			Identifier:  "+1111",
			CreatedAt:   time.Now(),
		},
	}

	c, w := newContactAuthContext()
	c.Params = []gin.Param{{Key: "id", Value: "c-1"}}

	handler.Get(c)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}

	resp := parseContactResponse(t, w)
	data, ok := resp.Data.(map[string]interface{})
	if !ok {
		t.Fatal("expected data to be an object")
	}

	identities, ok := data["identities"].([]interface{})
	if !ok {
		t.Fatalf("expected identities to be a slice, got %T", data["identities"])
	}
	if len(identities) != 1 {
		t.Fatalf("expected 1 identity, got %d", len(identities))
	}
}

// ---------------------------------------------------------------------------
// Create
// ---------------------------------------------------------------------------

func TestContactCreate_ValidData_Returns201(t *testing.T) {
	handler, _ := setupContactHandler()

	payload := CreateContactRequest{
		Name:  "Alice",
		Email: "alice@example.com",
		Phone: "+1111",
		Tags:  []string{"vip"},
		CustomFields: map[string]string{
			"company": "Acme",
		},
	}
	body, _ := json.Marshal(payload)

	c, w := newContactAuthContext()
	c.Request = httptest.NewRequest(http.MethodPost, "/contacts", bytes.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")

	handler.Create(c)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected status 201, got %d; body: %s", w.Code, w.Body.String())
	}

	resp := parseContactResponse(t, w)
	if !resp.Success {
		t.Fatal("expected success to be true")
	}

	data, ok := resp.Data.(map[string]interface{})
	if !ok {
		t.Fatal("expected data to be an object")
	}
	if data["name"] != "Alice" {
		t.Fatalf("expected name Alice, got %v", data["name"])
	}
	if data["email"] != "alice@example.com" {
		t.Fatalf("expected email alice@example.com, got %v", data["email"])
	}
	if data["tenant_id"] != "tenant-1" {
		t.Fatalf("expected tenant_id tenant-1, got %v", data["tenant_id"])
	}
}

func TestContactCreate_MissingName_Returns400(t *testing.T) {
	handler, _ := setupContactHandler()

	// Name is empty -- service.Create should reject it with a validation error
	payload := CreateContactRequest{
		Email: "alice@example.com",
		Phone: "+1111",
	}
	body, _ := json.Marshal(payload)

	c, w := newContactAuthContext()
	c.Request = httptest.NewRequest(http.MethodPost, "/contacts", bytes.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")

	handler.Create(c)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d; body: %s", w.Code, w.Body.String())
	}

	resp := parseContactResponse(t, w)
	if resp.Success {
		t.Fatal("expected success to be false")
	}
}

func TestContactCreate_InvalidBody_Returns400(t *testing.T) {
	handler, _ := setupContactHandler()

	c, w := newContactAuthContext()
	c.Request = httptest.NewRequest(http.MethodPost, "/contacts", bytes.NewReader([]byte("not json")))
	c.Request.Header.Set("Content-Type", "application/json")

	handler.Create(c)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", w.Code)
	}
}

func TestContactCreate_NoTenantID_Returns401(t *testing.T) {
	handler, _ := setupContactHandler()

	payload := CreateContactRequest{
		Name:  "Alice",
		Email: "alice@example.com",
	}
	body, _ := json.Marshal(payload)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/contacts", bytes.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")

	handler.Create(c)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected status 401, got %d", w.Code)
	}
}

func TestContactCreate_DuplicateEmail_Returns409(t *testing.T) {
	handler, repo := setupContactHandler()

	// Seed an existing contact with the same email
	seedContact(repo, "c-1", "tenant-1", "Alice", "alice@example.com", "+1111")

	payload := CreateContactRequest{
		Name:  "Alice Clone",
		Email: "alice@example.com",
		Phone: "+9999",
	}
	body, _ := json.Marshal(payload)

	c, w := newContactAuthContext()
	c.Request = httptest.NewRequest(http.MethodPost, "/contacts", bytes.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")

	handler.Create(c)

	if w.Code != http.StatusConflict {
		t.Fatalf("expected status 409, got %d; body: %s", w.Code, w.Body.String())
	}
}

func TestContactCreate_DuplicatePhone_Returns409(t *testing.T) {
	handler, repo := setupContactHandler()

	seedContact(repo, "c-1", "tenant-1", "Alice", "alice@example.com", "+1111")

	payload := CreateContactRequest{
		Name:  "Alice Clone",
		Email: "other@example.com",
		Phone: "+1111",
	}
	body, _ := json.Marshal(payload)

	c, w := newContactAuthContext()
	c.Request = httptest.NewRequest(http.MethodPost, "/contacts", bytes.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")

	handler.Create(c)

	if w.Code != http.StatusConflict {
		t.Fatalf("expected status 409, got %d; body: %s", w.Code, w.Body.String())
	}
}

// ---------------------------------------------------------------------------
// Update
// ---------------------------------------------------------------------------

func TestContactUpdate_ValidData_Returns200(t *testing.T) {
	handler, repo := setupContactHandler()

	seedContact(repo, "c-1", "tenant-1", "Alice", "alice@example.com", "+1111")

	payload := CreateContactRequest{
		Name:  "Alice Updated",
		Email: "newalice@example.com",
		Phone: "+2222",
	}
	body, _ := json.Marshal(payload)

	c, w := newContactAuthContext()
	c.Params = []gin.Param{{Key: "id", Value: "c-1"}}
	c.Request = httptest.NewRequest(http.MethodPut, "/contacts/c-1", bytes.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")

	handler.Update(c)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d; body: %s", w.Code, w.Body.String())
	}

	resp := parseContactResponse(t, w)
	if !resp.Success {
		t.Fatal("expected success to be true")
	}

	data, ok := resp.Data.(map[string]interface{})
	if !ok {
		t.Fatal("expected data to be an object")
	}
	if data["name"] != "Alice Updated" {
		t.Fatalf("expected name 'Alice Updated', got %v", data["name"])
	}
	if data["email"] != "newalice@example.com" {
		t.Fatalf("expected email 'newalice@example.com', got %v", data["email"])
	}
}

func TestContactUpdate_NotFound_Returns404(t *testing.T) {
	handler, _ := setupContactHandler()

	payload := CreateContactRequest{
		Name: "Alice Updated",
	}
	body, _ := json.Marshal(payload)

	c, w := newContactAuthContext()
	c.Params = []gin.Param{{Key: "id", Value: "nonexistent"}}
	c.Request = httptest.NewRequest(http.MethodPut, "/contacts/nonexistent", bytes.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")

	handler.Update(c)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected status 404, got %d", w.Code)
	}
}

func TestContactUpdate_EmptyID_Returns400(t *testing.T) {
	handler, _ := setupContactHandler()

	body := []byte(`{"name":"test"}`)

	c, w := newContactAuthContext()
	c.Params = []gin.Param{}
	c.Request = httptest.NewRequest(http.MethodPut, "/contacts/", bytes.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")

	handler.Update(c)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", w.Code)
	}
}

func TestContactUpdate_InvalidBody_Returns400(t *testing.T) {
	handler, repo := setupContactHandler()

	seedContact(repo, "c-1", "tenant-1", "Alice", "alice@example.com", "+1111")

	c, w := newContactAuthContext()
	c.Params = []gin.Param{{Key: "id", Value: "c-1"}}
	c.Request = httptest.NewRequest(http.MethodPut, "/contacts/c-1", bytes.NewReader([]byte("not json")))
	c.Request.Header.Set("Content-Type", "application/json")

	handler.Update(c)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", w.Code)
	}
}

func TestContactUpdate_WithTags_Returns200(t *testing.T) {
	handler, repo := setupContactHandler()

	seedContact(repo, "c-1", "tenant-1", "Alice", "alice@example.com", "+1111")

	payload := CreateContactRequest{
		Name: "Alice",
		Tags: []string{"vip", "premium"},
	}
	body, _ := json.Marshal(payload)

	c, w := newContactAuthContext()
	c.Params = []gin.Param{{Key: "id", Value: "c-1"}}
	c.Request = httptest.NewRequest(http.MethodPut, "/contacts/c-1", bytes.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")

	handler.Update(c)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d; body: %s", w.Code, w.Body.String())
	}

	resp := parseContactResponse(t, w)
	data, ok := resp.Data.(map[string]interface{})
	if !ok {
		t.Fatal("expected data to be an object")
	}

	tags, ok := data["tags"].([]interface{})
	if !ok {
		t.Fatalf("expected tags to be a slice, got %T", data["tags"])
	}
	if len(tags) != 2 {
		t.Fatalf("expected 2 tags, got %d", len(tags))
	}
}

// ---------------------------------------------------------------------------
// Delete
// ---------------------------------------------------------------------------

func TestContactDelete_Returns204(t *testing.T) {
	handler, repo := setupContactHandler()

	seedContact(repo, "c-1", "tenant-1", "Alice", "alice@example.com", "+1111")

	c, w := newContactAuthContext()
	c.Params = []gin.Param{{Key: "id", Value: "c-1"}}
	c.Request = httptest.NewRequest(http.MethodDelete, "/contacts/c-1", nil)

	handler.Delete(c)

	// gin's c.Status() sets the code on the writer; check both the writer
	// and the recorder to cover different gin versions.
	gotCode := w.Code
	if gotCode == http.StatusOK {
		// Fallback: gin may only set the status on its internal writer
		gotCode = c.Writer.Status()
	}
	if gotCode != http.StatusNoContent {
		t.Fatalf("expected status 204, got %d; body: %s", gotCode, w.Body.String())
	}

	// Body must be empty for 204
	if w.Body.Len() != 0 {
		t.Fatalf("expected empty body, got %s", w.Body.String())
	}

	// Verify contact is removed from repo
	if _, exists := repo.Contacts["c-1"]; exists {
		t.Fatal("expected contact to be deleted from repo")
	}
}

func TestContactDelete_NotFound_Returns404(t *testing.T) {
	handler, _ := setupContactHandler()

	c, w := newContactAuthContext()
	c.Params = []gin.Param{{Key: "id", Value: "nonexistent"}}
	c.Request = httptest.NewRequest(http.MethodDelete, "/contacts/nonexistent", nil)

	handler.Delete(c)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected status 404, got %d", w.Code)
	}
}

func TestContactDelete_EmptyID_Returns400(t *testing.T) {
	handler, _ := setupContactHandler()

	c, w := newContactAuthContext()
	c.Params = []gin.Param{}
	c.Request = httptest.NewRequest(http.MethodDelete, "/contacts/", nil)

	handler.Delete(c)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", w.Code)
	}
}

// ---------------------------------------------------------------------------
// AddIdentity
// ---------------------------------------------------------------------------

func TestContactAddIdentity_ValidData_Returns200(t *testing.T) {
	handler, repo := setupContactHandler()

	seedContact(repo, "c-1", "tenant-1", "Alice", "alice@example.com", "+1111")

	payload := AddIdentityRequest{
		ChannelType: "whatsapp",
		Identifier:  "+1111",
		Metadata:    map[string]string{"display_name": "Alice WA"},
	}
	body, _ := json.Marshal(payload)

	c, w := newContactAuthContext()
	c.Params = []gin.Param{{Key: "id", Value: "c-1"}}
	c.Request = httptest.NewRequest(http.MethodPost, "/contacts/c-1/identities", bytes.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")

	handler.AddIdentity(c)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d; body: %s", w.Code, w.Body.String())
	}

	resp := parseContactResponse(t, w)
	if !resp.Success {
		t.Fatal("expected success to be true")
	}

	// Verify identity was added to repo
	identities := repo.Identities["c-1"]
	if len(identities) != 1 {
		t.Fatalf("expected 1 identity in repo, got %d", len(identities))
	}
	if identities[0].ChannelType != "whatsapp" {
		t.Fatalf("expected channel_type whatsapp, got %s", identities[0].ChannelType)
	}
}

func TestContactAddIdentity_ContactNotFound_Returns404(t *testing.T) {
	handler, _ := setupContactHandler()

	payload := AddIdentityRequest{
		ChannelType: "whatsapp",
		Identifier:  "+1111",
	}
	body, _ := json.Marshal(payload)

	c, w := newContactAuthContext()
	c.Params = []gin.Param{{Key: "id", Value: "nonexistent"}}
	c.Request = httptest.NewRequest(http.MethodPost, "/contacts/nonexistent/identities", bytes.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")

	handler.AddIdentity(c)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected status 404, got %d", w.Code)
	}
}

func TestContactAddIdentity_InvalidBody_Returns400(t *testing.T) {
	handler, repo := setupContactHandler()

	seedContact(repo, "c-1", "tenant-1", "Alice", "alice@example.com", "+1111")

	c, w := newContactAuthContext()
	c.Params = []gin.Param{{Key: "id", Value: "c-1"}}
	c.Request = httptest.NewRequest(http.MethodPost, "/contacts/c-1/identities", bytes.NewReader([]byte("not json")))
	c.Request.Header.Set("Content-Type", "application/json")

	handler.AddIdentity(c)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", w.Code)
	}
}

func TestContactAddIdentity_MissingRequiredFields_Returns400(t *testing.T) {
	handler, repo := setupContactHandler()

	seedContact(repo, "c-1", "tenant-1", "Alice", "alice@example.com", "+1111")

	// Missing channel_type and identifier (both required via binding:"required")
	payload := map[string]string{}
	body, _ := json.Marshal(payload)

	c, w := newContactAuthContext()
	c.Params = []gin.Param{{Key: "id", Value: "c-1"}}
	c.Request = httptest.NewRequest(http.MethodPost, "/contacts/c-1/identities", bytes.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")

	handler.AddIdentity(c)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", w.Code)
	}
}

func TestContactAddIdentity_EmptyID_Returns400(t *testing.T) {
	handler, _ := setupContactHandler()

	payload := AddIdentityRequest{
		ChannelType: "whatsapp",
		Identifier:  "+1111",
	}
	body, _ := json.Marshal(payload)

	c, w := newContactAuthContext()
	c.Params = []gin.Param{}
	c.Request = httptest.NewRequest(http.MethodPost, "/contacts//identities", bytes.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")

	handler.AddIdentity(c)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", w.Code)
	}
}

// ---------------------------------------------------------------------------
// RemoveIdentity
// ---------------------------------------------------------------------------

func TestContactRemoveIdentity_ValidData_Returns200(t *testing.T) {
	handler, repo := setupContactHandler()

	seedContact(repo, "c-1", "tenant-1", "Alice", "alice@example.com", "+1111")
	repo.Identities["c-1"] = []*entity.ContactIdentity{
		{
			ID:          "id-1",
			ContactID:   "c-1",
			ChannelType: "whatsapp",
			Identifier:  "+1111",
			CreatedAt:   time.Now(),
		},
	}

	c, w := newContactAuthContext()
	c.Params = []gin.Param{
		{Key: "id", Value: "c-1"},
		{Key: "identityId", Value: "id-1"},
	}
	c.Request = httptest.NewRequest(http.MethodDelete, "/contacts/c-1/identities/id-1", nil)

	handler.RemoveIdentity(c)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d; body: %s", w.Code, w.Body.String())
	}

	resp := parseContactResponse(t, w)
	if !resp.Success {
		t.Fatal("expected success to be true")
	}

	// Verify identity was removed from repo
	identities := repo.Identities["c-1"]
	if len(identities) != 0 {
		t.Fatalf("expected 0 identities in repo, got %d", len(identities))
	}
}

func TestContactRemoveIdentity_ContactNotFound_Returns404(t *testing.T) {
	handler, _ := setupContactHandler()

	c, w := newContactAuthContext()
	c.Params = []gin.Param{
		{Key: "id", Value: "nonexistent"},
		{Key: "identityId", Value: "id-1"},
	}
	c.Request = httptest.NewRequest(http.MethodDelete, "/contacts/nonexistent/identities/id-1", nil)

	handler.RemoveIdentity(c)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected status 404, got %d", w.Code)
	}
}

func TestContactRemoveIdentity_EmptyContactID_Returns400(t *testing.T) {
	handler, _ := setupContactHandler()

	c, w := newContactAuthContext()
	c.Params = []gin.Param{
		{Key: "identityId", Value: "id-1"},
	}
	c.Request = httptest.NewRequest(http.MethodDelete, "/contacts//identities/id-1", nil)

	handler.RemoveIdentity(c)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", w.Code)
	}
}

func TestContactRemoveIdentity_EmptyIdentityID_Returns400(t *testing.T) {
	handler, _ := setupContactHandler()

	c, w := newContactAuthContext()
	c.Params = []gin.Param{
		{Key: "id", Value: "c-1"},
	}
	c.Request = httptest.NewRequest(http.MethodDelete, "/contacts/c-1/identities/", nil)

	handler.RemoveIdentity(c)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", w.Code)
	}
}
