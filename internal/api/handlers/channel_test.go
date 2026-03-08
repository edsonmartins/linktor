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

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func setupChannelHandler() (*ChannelHandler, *testutil.MockChannelRepository, *testutil.MockProducer) {
	repo := testutil.NewMockChannelRepository()
	producer := testutil.NewMockProducer()
	svc := service.NewChannelService(repo, nil, producer)
	handler := NewChannelHandler(svc, producer)
	return handler, repo, producer
}

func newChannelAuthContext(method, path string, body []byte) (*gin.Context, *httptest.ResponseRecorder) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Set("tenant_id", "tenant-1")
	c.Set("user_id", "user-1")
	var req *http.Request
	if body != nil {
		req = httptest.NewRequest(method, path, bytes.NewReader(body))
	} else {
		req = httptest.NewRequest(method, path, nil)
	}
	req.Header.Set("Content-Type", "application/json")
	c.Request = req
	return c, w
}

func seedChannel(repo *testutil.MockChannelRepository, id, tenantID, name string, chType entity.ChannelType) *entity.Channel {
	now := time.Now()
	ch := &entity.Channel{
		ID:               id,
		TenantID:         tenantID,
		Type:             chType,
		Name:             name,
		Identifier:       "+5511999999999",
		Enabled:          true,
		ConnectionStatus: entity.ConnectionStatusDisconnected,
		Config:           map[string]string{},
		Credentials:      map[string]string{},
		CreatedAt:        now,
		UpdatedAt:        now,
	}
	repo.Channels[id] = ch
	return ch
}

func parseChannelResponse(t *testing.T, w *httptest.ResponseRecorder) Response {
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

func TestChannelList_Success(t *testing.T) {
	handler, repo, _ := setupChannelHandler()
	seedChannel(repo, "ch-1", "tenant-1", "WhatsApp 1", entity.ChannelTypeWhatsApp)
	seedChannel(repo, "ch-2", "tenant-1", "Telegram", entity.ChannelTypeTelegram)
	// Different tenant — should not appear
	seedChannel(repo, "ch-3", "tenant-2", "Other", entity.ChannelTypeSMS)

	c, w := newChannelAuthContext(http.MethodGet, "/channels", nil)
	handler.List(c)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	resp := parseChannelResponse(t, w)
	if !resp.Success {
		t.Fatal("expected success=true")
	}
	// Data should be a slice with 2 items
	items, ok := resp.Data.([]interface{})
	if !ok {
		t.Fatalf("expected data to be a slice, got %T", resp.Data)
	}
	if len(items) != 2 {
		t.Fatalf("expected 2 channels, got %d", len(items))
	}
}

func TestChannelList_Empty(t *testing.T) {
	handler, _, _ := setupChannelHandler()

	c, w := newChannelAuthContext(http.MethodGet, "/channels", nil)
	handler.List(c)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	resp := parseChannelResponse(t, w)
	if !resp.Success {
		t.Fatal("expected success=true")
	}
}

func TestChannelList_NoTenantID(t *testing.T) {
	handler, _, _ := setupChannelHandler()

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/channels", nil)
	// Do NOT set tenant_id

	handler.List(c)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
}

// ---------------------------------------------------------------------------
// Get
// ---------------------------------------------------------------------------

func TestChannelGet_Success(t *testing.T) {
	handler, repo, _ := setupChannelHandler()
	seedChannel(repo, "ch-1", "tenant-1", "WhatsApp 1", entity.ChannelTypeWhatsApp)

	c, w := newChannelAuthContext(http.MethodGet, "/channels/ch-1", nil)
	c.Params = gin.Params{{Key: "id", Value: "ch-1"}}

	handler.Get(c)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	resp := parseChannelResponse(t, w)
	if !resp.Success {
		t.Fatal("expected success=true")
	}
	data, ok := resp.Data.(map[string]interface{})
	if !ok {
		t.Fatalf("expected data to be a map, got %T", resp.Data)
	}
	if data["id"] != "ch-1" {
		t.Fatalf("expected id=ch-1, got %v", data["id"])
	}
}

func TestChannelGet_NotFound(t *testing.T) {
	handler, _, _ := setupChannelHandler()

	c, w := newChannelAuthContext(http.MethodGet, "/channels/nonexistent", nil)
	c.Params = gin.Params{{Key: "id", Value: "nonexistent"}}

	handler.Get(c)

	// The mock repo returns a plain error, which maps to 500 via RespondError
	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", w.Code)
	}
	resp := parseChannelResponse(t, w)
	if resp.Success {
		t.Fatal("expected success=false")
	}
}

func TestChannelGet_EmptyID(t *testing.T) {
	handler, _, _ := setupChannelHandler()

	c, w := newChannelAuthContext(http.MethodGet, "/channels/", nil)
	// No param set — simulates empty id

	handler.Get(c)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

// ---------------------------------------------------------------------------
// Create
// ---------------------------------------------------------------------------

func TestChannelCreate_Success(t *testing.T) {
	handler, repo, _ := setupChannelHandler()

	body, _ := json.Marshal(CreateChannelRequest{
		Type:       "telegram",
		Name:       "My Telegram",
		Identifier: "@mybot",
		Config:     map[string]string{"bot_token": "abc"},
	})

	c, w := newChannelAuthContext(http.MethodPost, "/channels", body)
	handler.Create(c)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d; body: %s", w.Code, w.Body.String())
	}
	resp := parseChannelResponse(t, w)
	if !resp.Success {
		t.Fatal("expected success=true")
	}
	// Verify channel was stored in the repo
	if len(repo.Channels) != 1 {
		t.Fatalf("expected 1 channel in repo, got %d", len(repo.Channels))
	}
	// Verify the returned data has the correct type and name
	data := resp.Data.(map[string]interface{})
	if data["type"] != "telegram" {
		t.Fatalf("expected type=telegram, got %v", data["type"])
	}
	if data["name"] != "My Telegram" {
		t.Fatalf("expected name=My Telegram, got %v", data["name"])
	}
	if data["tenant_id"] != "tenant-1" {
		t.Fatalf("expected tenant_id=tenant-1, got %v", data["tenant_id"])
	}
}

func TestChannelCreate_MissingType(t *testing.T) {
	handler, _, _ := setupChannelHandler()

	body, _ := json.Marshal(map[string]string{
		"name": "No Type Channel",
	})

	c, w := newChannelAuthContext(http.MethodPost, "/channels", body)
	handler.Create(c)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
	resp := parseChannelResponse(t, w)
	if resp.Success {
		t.Fatal("expected success=false")
	}
}

func TestChannelCreate_MissingName(t *testing.T) {
	handler, _, _ := setupChannelHandler()

	body, _ := json.Marshal(map[string]string{
		"type": "telegram",
	})

	c, w := newChannelAuthContext(http.MethodPost, "/channels", body)
	handler.Create(c)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestChannelCreate_EmptyBody(t *testing.T) {
	handler, _, _ := setupChannelHandler()

	c, w := newChannelAuthContext(http.MethodPost, "/channels", []byte(`{}`))
	handler.Create(c)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestChannelCreate_NoTenantID(t *testing.T) {
	handler, _, _ := setupChannelHandler()

	body, _ := json.Marshal(CreateChannelRequest{
		Type: "telegram",
		Name: "Bot",
	})

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/channels", bytes.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")
	// No tenant_id set

	handler.Create(c)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
}

// ---------------------------------------------------------------------------
// Update
// ---------------------------------------------------------------------------

func TestChannelUpdate_Success(t *testing.T) {
	handler, repo, _ := setupChannelHandler()
	seedChannel(repo, "ch-1", "tenant-1", "Old Name", entity.ChannelTypeTelegram)

	body, _ := json.Marshal(CreateChannelRequest{
		Type:       "telegram",
		Name:       "New Name",
		Identifier: "@newbot",
		Config:     map[string]string{"key": "val"},
	})

	c, w := newChannelAuthContext(http.MethodPut, "/channels/ch-1", body)
	c.Params = gin.Params{{Key: "id", Value: "ch-1"}}

	handler.Update(c)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d; body: %s", w.Code, w.Body.String())
	}
	resp := parseChannelResponse(t, w)
	if !resp.Success {
		t.Fatal("expected success=true")
	}
	data := resp.Data.(map[string]interface{})
	if data["name"] != "New Name" {
		t.Fatalf("expected name=New Name, got %v", data["name"])
	}
	// Verify repo was updated
	ch := repo.Channels["ch-1"]
	if ch.Name != "New Name" {
		t.Fatalf("expected repo name=New Name, got %s", ch.Name)
	}
	if ch.Identifier != "@newbot" {
		t.Fatalf("expected repo identifier=@newbot, got %s", ch.Identifier)
	}
}

func TestChannelUpdate_NotFound(t *testing.T) {
	handler, _, _ := setupChannelHandler()

	body, _ := json.Marshal(CreateChannelRequest{
		Type: "telegram",
		Name: "Name",
	})

	c, w := newChannelAuthContext(http.MethodPut, "/channels/nonexistent", body)
	c.Params = gin.Params{{Key: "id", Value: "nonexistent"}}

	handler.Update(c)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", w.Code)
	}
}

func TestChannelUpdate_EmptyID(t *testing.T) {
	handler, _, _ := setupChannelHandler()

	body, _ := json.Marshal(CreateChannelRequest{
		Type: "telegram",
		Name: "Name",
	})

	c, w := newChannelAuthContext(http.MethodPut, "/channels/", body)
	// No id param

	handler.Update(c)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestChannelUpdate_InvalidBody(t *testing.T) {
	handler, repo, _ := setupChannelHandler()
	seedChannel(repo, "ch-1", "tenant-1", "Name", entity.ChannelTypeTelegram)

	c, w := newChannelAuthContext(http.MethodPut, "/channels/ch-1", []byte(`{invalid`))
	c.Params = gin.Params{{Key: "id", Value: "ch-1"}}

	handler.Update(c)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

// ---------------------------------------------------------------------------
// Delete
// ---------------------------------------------------------------------------

func TestChannelDelete_Success(t *testing.T) {
	handler, repo, _ := setupChannelHandler()
	seedChannel(repo, "ch-1", "tenant-1", "To Delete", entity.ChannelTypeTelegram)

	c, _ := newChannelAuthContext(http.MethodDelete, "/channels/ch-1", nil)
	c.Params = gin.Params{{Key: "id", Value: "ch-1"}}

	handler.Delete(c)

	// gin's c.Status() does not flush to the recorder when no body is written,
	// so we check via the writer's Status() method instead.
	if c.Writer.Status() != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", c.Writer.Status())
	}
	if _, exists := repo.Channels["ch-1"]; exists {
		t.Fatal("expected channel to be deleted from repo")
	}
}

func TestChannelDelete_NotFound(t *testing.T) {
	handler, _, _ := setupChannelHandler()

	c, _ := newChannelAuthContext(http.MethodDelete, "/channels/nonexistent", nil)
	c.Params = gin.Params{{Key: "id", Value: "nonexistent"}}

	handler.Delete(c)

	// Mock repo Delete is a no-op on missing key, so handler returns 204.
	if c.Writer.Status() != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", c.Writer.Status())
	}
}

func TestChannelDelete_EmptyID(t *testing.T) {
	handler, _, _ := setupChannelHandler()

	c, w := newChannelAuthContext(http.MethodDelete, "/channels/", nil)

	handler.Delete(c)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

// ---------------------------------------------------------------------------
// UpdateStatus (active / inactive)
// ---------------------------------------------------------------------------

func TestChannelUpdateStatus_Active(t *testing.T) {
	handler, repo, _ := setupChannelHandler()
	ch := seedChannel(repo, "ch-1", "tenant-1", "Channel", entity.ChannelTypeTelegram)
	ch.Enabled = false

	body, _ := json.Marshal(UpdateStatusRequest{Status: "active"})
	c, w := newChannelAuthContext(http.MethodPut, "/channels/ch-1/status", body)
	c.Params = gin.Params{{Key: "id", Value: "ch-1"}}

	handler.UpdateStatus(c)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d; body: %s", w.Code, w.Body.String())
	}
	resp := parseChannelResponse(t, w)
	if !resp.Success {
		t.Fatal("expected success=true")
	}
	data := resp.Data.(map[string]interface{})
	if data["enabled"] != true {
		t.Fatalf("expected enabled=true, got %v", data["enabled"])
	}
}

func TestChannelUpdateStatus_Inactive(t *testing.T) {
	handler, repo, _ := setupChannelHandler()
	seedChannel(repo, "ch-1", "tenant-1", "Channel", entity.ChannelTypeTelegram)

	body, _ := json.Marshal(UpdateStatusRequest{Status: "inactive"})
	c, w := newChannelAuthContext(http.MethodPut, "/channels/ch-1/status", body)
	c.Params = gin.Params{{Key: "id", Value: "ch-1"}}

	handler.UpdateStatus(c)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d; body: %s", w.Code, w.Body.String())
	}
	resp := parseChannelResponse(t, w)
	data := resp.Data.(map[string]interface{})
	if data["enabled"] != false {
		t.Fatalf("expected enabled=false, got %v", data["enabled"])
	}
}

func TestChannelUpdateStatus_InvalidStatus(t *testing.T) {
	handler, repo, _ := setupChannelHandler()
	seedChannel(repo, "ch-1", "tenant-1", "Channel", entity.ChannelTypeTelegram)

	body, _ := json.Marshal(map[string]string{"status": "unknown"})
	c, w := newChannelAuthContext(http.MethodPut, "/channels/ch-1/status", body)
	c.Params = gin.Params{{Key: "id", Value: "ch-1"}}

	handler.UpdateStatus(c)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d; body: %s", w.Code, w.Body.String())
	}
}

func TestChannelUpdateStatus_EmptyID(t *testing.T) {
	handler, _, _ := setupChannelHandler()

	body, _ := json.Marshal(UpdateStatusRequest{Status: "active"})
	c, w := newChannelAuthContext(http.MethodPut, "/channels//status", body)

	handler.UpdateStatus(c)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestChannelUpdateStatus_NotFound(t *testing.T) {
	handler, _, _ := setupChannelHandler()

	body, _ := json.Marshal(UpdateStatusRequest{Status: "active"})
	c, w := newChannelAuthContext(http.MethodPut, "/channels/nonexistent/status", body)
	c.Params = gin.Params{{Key: "id", Value: "nonexistent"}}

	handler.UpdateStatus(c)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", w.Code)
	}
}

// ---------------------------------------------------------------------------
// UpdateEnabled
// ---------------------------------------------------------------------------

func TestChannelUpdateEnabled_Enable(t *testing.T) {
	handler, repo, _ := setupChannelHandler()
	ch := seedChannel(repo, "ch-1", "tenant-1", "Channel", entity.ChannelTypeTelegram)
	ch.Enabled = false

	body, _ := json.Marshal(UpdateEnabledRequest{Enabled: true})
	c, w := newChannelAuthContext(http.MethodPut, "/channels/ch-1/enabled", body)
	c.Params = gin.Params{{Key: "id", Value: "ch-1"}}

	handler.UpdateEnabled(c)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d; body: %s", w.Code, w.Body.String())
	}
	resp := parseChannelResponse(t, w)
	if !resp.Success {
		t.Fatal("expected success=true")
	}
	data := resp.Data.(map[string]interface{})
	if data["enabled"] != true {
		t.Fatalf("expected enabled=true, got %v", data["enabled"])
	}
}

func TestChannelUpdateEnabled_Disable(t *testing.T) {
	handler, repo, _ := setupChannelHandler()
	seedChannel(repo, "ch-1", "tenant-1", "Channel", entity.ChannelTypeTelegram)

	body, _ := json.Marshal(UpdateEnabledRequest{Enabled: false})
	c, w := newChannelAuthContext(http.MethodPut, "/channels/ch-1/enabled", body)
	c.Params = gin.Params{{Key: "id", Value: "ch-1"}}

	handler.UpdateEnabled(c)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d; body: %s", w.Code, w.Body.String())
	}
	resp := parseChannelResponse(t, w)
	data := resp.Data.(map[string]interface{})
	if data["enabled"] != false {
		t.Fatalf("expected enabled=false, got %v", data["enabled"])
	}
}

func TestChannelUpdateEnabled_EmptyID(t *testing.T) {
	handler, _, _ := setupChannelHandler()

	body, _ := json.Marshal(UpdateEnabledRequest{Enabled: true})
	c, w := newChannelAuthContext(http.MethodPut, "/channels//enabled", body)

	handler.UpdateEnabled(c)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestChannelUpdateEnabled_NotFound(t *testing.T) {
	handler, _, _ := setupChannelHandler()

	body, _ := json.Marshal(UpdateEnabledRequest{Enabled: true})
	c, w := newChannelAuthContext(http.MethodPut, "/channels/nonexistent/enabled", body)
	c.Params = gin.Params{{Key: "id", Value: "nonexistent"}}

	handler.UpdateEnabled(c)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", w.Code)
	}
}

// ---------------------------------------------------------------------------
// Connect (non-WhatsApp channel, uses default path)
// ---------------------------------------------------------------------------

func TestChannelConnect_NonWhatsApp(t *testing.T) {
	handler, repo, _ := setupChannelHandler()
	seedChannel(repo, "ch-1", "tenant-1", "Telegram", entity.ChannelTypeTelegram)

	c, w := newChannelAuthContext(http.MethodPost, "/channels/ch-1/connect", nil)
	c.Params = gin.Params{{Key: "id", Value: "ch-1"}}

	handler.Connect(c)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d; body: %s", w.Code, w.Body.String())
	}
	resp := parseChannelResponse(t, w)
	if !resp.Success {
		t.Fatal("expected success=true")
	}
	// Verify channel is now connected in the repo
	ch := repo.Channels["ch-1"]
	if ch.ConnectionStatus != entity.ConnectionStatusConnected {
		t.Fatalf("expected connected status, got %s", ch.ConnectionStatus)
	}
}

func TestChannelConnect_NotFound(t *testing.T) {
	handler, _, _ := setupChannelHandler()

	c, w := newChannelAuthContext(http.MethodPost, "/channels/nonexistent/connect", nil)
	c.Params = gin.Params{{Key: "id", Value: "nonexistent"}}

	handler.Connect(c)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", w.Code)
	}
}

func TestChannelConnect_EmptyID(t *testing.T) {
	handler, _, _ := setupChannelHandler()

	c, w := newChannelAuthContext(http.MethodPost, "/channels//connect", nil)

	handler.Connect(c)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

// ---------------------------------------------------------------------------
// Disconnect
// ---------------------------------------------------------------------------

func TestChannelDisconnect_Success(t *testing.T) {
	handler, repo, _ := setupChannelHandler()
	ch := seedChannel(repo, "ch-1", "tenant-1", "Telegram", entity.ChannelTypeTelegram)
	ch.ConnectionStatus = entity.ConnectionStatusConnected

	c, w := newChannelAuthContext(http.MethodPost, "/channels/ch-1/disconnect", nil)
	c.Params = gin.Params{{Key: "id", Value: "ch-1"}}

	handler.Disconnect(c)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d; body: %s", w.Code, w.Body.String())
	}
	resp := parseChannelResponse(t, w)
	if !resp.Success {
		t.Fatal("expected success=true")
	}
	// Verify channel is now disconnected
	repoChannel := repo.Channels["ch-1"]
	if repoChannel.ConnectionStatus != entity.ConnectionStatusDisconnected {
		t.Fatalf("expected disconnected status, got %s", repoChannel.ConnectionStatus)
	}
}

func TestChannelDisconnect_NotFound(t *testing.T) {
	handler, _, _ := setupChannelHandler()

	c, w := newChannelAuthContext(http.MethodPost, "/channels/nonexistent/disconnect", nil)
	c.Params = gin.Params{{Key: "id", Value: "nonexistent"}}

	handler.Disconnect(c)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", w.Code)
	}
}

func TestChannelDisconnect_EmptyID(t *testing.T) {
	handler, _, _ := setupChannelHandler()

	c, w := newChannelAuthContext(http.MethodPost, "/channels//disconnect", nil)

	handler.Disconnect(c)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

// ---------------------------------------------------------------------------
// WhatsAppVerify
// ---------------------------------------------------------------------------

func TestChannelWhatsAppVerify(t *testing.T) {
	handler, _, _ := setupChannelHandler()

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/webhooks/whatsapp?hub.challenge=test-challenge-123", nil)

	handler.WhatsAppVerify(c)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	if w.Body.String() != "test-challenge-123" {
		t.Fatalf("expected body=test-challenge-123, got %s", w.Body.String())
	}
}

// ---------------------------------------------------------------------------
// WhatsAppWebhook
// ---------------------------------------------------------------------------

func TestChannelWhatsAppWebhook(t *testing.T) {
	handler, _, _ := setupChannelHandler()

	c, w := newChannelAuthContext(http.MethodPost, "/webhooks/whatsapp/ch-1", nil)
	c.Params = gin.Params{{Key: "channelId", Value: "ch-1"}}

	handler.WhatsAppWebhook(c)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	resp := parseChannelResponse(t, w)
	if !resp.Success {
		t.Fatal("expected success=true")
	}
}
