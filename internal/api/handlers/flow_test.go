package handlers

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/msgfy/linktor/internal/application/service"
	"github.com/msgfy/linktor/internal/domain/entity"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupFlowTest(t *testing.T) (*FlowHandler, *mockFlowRepository) {
	t.Helper()

	flowRepo := newMockFlowRepository()
	flowService := service.NewFlowService(flowRepo)
	handler := NewFlowHandler(flowService)

	return handler, flowRepo
}

func createTestFlow(id, tenantID string) *entity.Flow {
	return &entity.Flow{
		ID:           id,
		TenantID:     tenantID,
		Name:         "Welcome Flow",
		Description:  "Greets new users",
		Trigger:      entity.FlowTriggerWelcome,
		TriggerValue: "",
		StartNodeID:  "node-1",
		Nodes: []entity.FlowNode{
			{
				ID:      "node-1",
				Type:    entity.FlowNodeMessage,
				Content: "Welcome! How can I help you?",
				Transitions: []entity.FlowTransition{
					{ToNodeID: "node-2", Condition: entity.TransitionConditionDefault},
				},
			},
			{
				ID:      "node-2",
				Type:    entity.FlowNodeMessage,
				Content: "Thank you for contacting us.",
			},
		},
		IsActive: false,
		Priority: 1,
	}
}

func TestFlowHandler_List(t *testing.T) {
	handler, flowRepo := setupFlowTest(t)

	flowRepo.Flows["flow-1"] = createTestFlow("flow-1", "tenant-1")
	flowRepo.Flows["flow-2"] = &entity.Flow{
		ID:          "flow-2",
		TenantID:    "tenant-1",
		Name:        "FAQ Flow",
		Trigger:     entity.FlowTriggerKeyword,
		StartNodeID: "node-1",
		Nodes: []entity.FlowNode{
			{ID: "node-1", Type: entity.FlowNodeMessage, Content: "FAQ response"},
		},
	}

	w, c := newTestContext(http.MethodGet, "/flows", nil)
	c.Set("tenant_id", "tenant-1")
	c.Request.URL.RawQuery = "page=1&page_size=20"

	handler.List(c)

	assert.Equal(t, http.StatusOK, w.Code)

	var rawResp map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &rawResp)
	require.NoError(t, err)
	assert.NotNil(t, rawResp["data"])

	dataList, ok := rawResp["data"].([]interface{})
	require.True(t, ok)
	assert.Len(t, dataList, 2)
}

func TestFlowHandler_List_NoTenantID(t *testing.T) {
	handler, _ := setupFlowTest(t)

	w, c := newTestContext(http.MethodGet, "/flows", nil)

	handler.List(c)

	assert.NotEqual(t, http.StatusOK, w.Code)
}

func TestFlowHandler_Create(t *testing.T) {
	handler, _ := setupFlowTest(t)

	body := CreateFlowRequest{
		Name:        "New Flow",
		Description: "A test flow",
		Trigger:     "keyword",
		TriggerValue: "help",
		StartNodeID: "node-1",
		Nodes: []entity.FlowNode{
			{
				ID:      "node-1",
				Type:    entity.FlowNodeMessage,
				Content: "How can I help?",
			},
		},
		Priority: 5,
	}

	w, c := newTestContext(http.MethodPost, "/flows", body)
	c.Set("tenant_id", "tenant-1")

	handler.Create(c)

	assert.Equal(t, http.StatusCreated, w.Code)

	var resp Response
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.True(t, resp.Success)

	dataMap := resp.Data.(map[string]interface{})
	assert.Equal(t, "New Flow", dataMap["name"])
	assert.Equal(t, "keyword", dataMap["trigger"])
}

func TestFlowHandler_Create_InvalidBody(t *testing.T) {
	handler, _ := setupFlowTest(t)

	body := map[string]string{
		"name": "incomplete",
	}

	w, c := newTestContext(http.MethodPost, "/flows", body)
	c.Set("tenant_id", "tenant-1")

	handler.Create(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var resp Response
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.False(t, resp.Success)
	assert.Equal(t, "VALIDATION_ERROR", resp.Error.Code)
}

func TestFlowHandler_Create_NoTenantID(t *testing.T) {
	handler, _ := setupFlowTest(t)

	body := CreateFlowRequest{
		Name:        "New Flow",
		Trigger:     "keyword",
		StartNodeID: "node-1",
		Nodes: []entity.FlowNode{
			{ID: "node-1", Type: entity.FlowNodeMessage, Content: "Hi"},
		},
	}

	w, c := newTestContext(http.MethodPost, "/flows", body)

	handler.Create(c)

	assert.NotEqual(t, http.StatusCreated, w.Code)
}

func TestFlowHandler_Get(t *testing.T) {
	handler, flowRepo := setupFlowTest(t)

	flowRepo.Flows["flow-1"] = createTestFlow("flow-1", "tenant-1")

	w, c := newTestContext(http.MethodGet, "/flows/flow-1", nil)
	c.Set("tenant_id", "tenant-1")
	c.Params = gin.Params{{Key: "id", Value: "flow-1"}}

	handler.Get(c)

	assert.Equal(t, http.StatusOK, w.Code)

	var rawResp map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &rawResp)
	require.NoError(t, err)
	assert.Equal(t, "flow-1", rawResp["id"])
	assert.Equal(t, "Welcome Flow", rawResp["name"])
}

func TestFlowHandler_Get_NotFound(t *testing.T) {
	handler, _ := setupFlowTest(t)

	w, c := newTestContext(http.MethodGet, "/flows/nonexistent", nil)
	c.Set("tenant_id", "tenant-1")
	c.Params = gin.Params{{Key: "id", Value: "nonexistent"}}

	handler.Get(c)

	assert.NotEqual(t, http.StatusOK, w.Code)
}

func TestFlowHandler_Get_WrongTenant(t *testing.T) {
	handler, flowRepo := setupFlowTest(t)

	flowRepo.Flows["flow-1"] = createTestFlow("flow-1", "tenant-2")

	w, c := newTestContext(http.MethodGet, "/flows/flow-1", nil)
	c.Set("tenant_id", "tenant-1")
	c.Params = gin.Params{{Key: "id", Value: "flow-1"}}

	handler.Get(c)

	assert.Equal(t, http.StatusForbidden, w.Code)

	var resp Response
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.False(t, resp.Success)
	assert.Equal(t, "FORBIDDEN", resp.Error.Code)
}

func TestFlowHandler_Get_EmptyID(t *testing.T) {
	handler, _ := setupFlowTest(t)

	w, c := newTestContext(http.MethodGet, "/flows/", nil)
	c.Set("tenant_id", "tenant-1")

	handler.Get(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var resp Response
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.False(t, resp.Success)
	assert.Equal(t, "VALIDATION_ERROR", resp.Error.Code)
}

func TestFlowHandler_Update(t *testing.T) {
	handler, flowRepo := setupFlowTest(t)

	flowRepo.Flows["flow-1"] = createTestFlow("flow-1", "tenant-1")

	newName := "Updated Flow"
	body := UpdateFlowRequest{
		Name: &newName,
	}

	w, c := newTestContext(http.MethodPut, "/flows/flow-1", body)
	c.Set("tenant_id", "tenant-1")
	c.Params = gin.Params{{Key: "id", Value: "flow-1"}}

	handler.Update(c)

	assert.Equal(t, http.StatusOK, w.Code)

	var rawResp map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &rawResp)
	require.NoError(t, err)
	assert.Equal(t, "Updated Flow", rawResp["name"])
}

func TestFlowHandler_Update_WrongTenant(t *testing.T) {
	handler, flowRepo := setupFlowTest(t)

	flowRepo.Flows["flow-1"] = createTestFlow("flow-1", "tenant-2")

	newName := "Updated"
	body := UpdateFlowRequest{
		Name: &newName,
	}

	w, c := newTestContext(http.MethodPut, "/flows/flow-1", body)
	c.Set("tenant_id", "tenant-1")
	c.Params = gin.Params{{Key: "id", Value: "flow-1"}}

	handler.Update(c)

	assert.Equal(t, http.StatusForbidden, w.Code)
}

func TestFlowHandler_Delete(t *testing.T) {
	handler, flowRepo := setupFlowTest(t)

	flowRepo.Flows["flow-1"] = createTestFlow("flow-1", "tenant-1")

	w, c := newTestContext(http.MethodDelete, "/flows/flow-1", nil)
	c.Set("tenant_id", "tenant-1")
	c.Params = gin.Params{{Key: "id", Value: "flow-1"}}

	handler.Delete(c)

	assert.Equal(t, http.StatusOK, w.Code)

	var rawResp map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &rawResp)
	require.NoError(t, err)
	assert.Equal(t, "Flow deleted successfully", rawResp["message"])
	assert.Empty(t, flowRepo.Flows)
}

func TestFlowHandler_Delete_WrongTenant(t *testing.T) {
	handler, flowRepo := setupFlowTest(t)

	flowRepo.Flows["flow-1"] = createTestFlow("flow-1", "tenant-2")

	w, c := newTestContext(http.MethodDelete, "/flows/flow-1", nil)
	c.Set("tenant_id", "tenant-1")
	c.Params = gin.Params{{Key: "id", Value: "flow-1"}}

	handler.Delete(c)

	assert.Equal(t, http.StatusForbidden, w.Code)
}

func TestFlowHandler_Activate(t *testing.T) {
	handler, flowRepo := setupFlowTest(t)

	flowRepo.Flows["flow-1"] = createTestFlow("flow-1", "tenant-1")

	w, c := newTestContext(http.MethodPost, "/flows/flow-1/activate", nil)
	c.Set("tenant_id", "tenant-1")
	c.Params = gin.Params{{Key: "id", Value: "flow-1"}}

	handler.Activate(c)

	assert.Equal(t, http.StatusOK, w.Code)

	var rawResp map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &rawResp)
	require.NoError(t, err)
	assert.Equal(t, "Flow activated successfully", rawResp["message"])
	assert.True(t, flowRepo.Flows["flow-1"].IsActive)
}

func TestFlowHandler_Activate_WrongTenant(t *testing.T) {
	handler, flowRepo := setupFlowTest(t)

	flowRepo.Flows["flow-1"] = createTestFlow("flow-1", "tenant-2")

	w, c := newTestContext(http.MethodPost, "/flows/flow-1/activate", nil)
	c.Set("tenant_id", "tenant-1")
	c.Params = gin.Params{{Key: "id", Value: "flow-1"}}

	handler.Activate(c)

	assert.Equal(t, http.StatusForbidden, w.Code)
}

func TestFlowHandler_Deactivate(t *testing.T) {
	handler, flowRepo := setupFlowTest(t)

	flow := createTestFlow("flow-1", "tenant-1")
	flow.IsActive = true
	flowRepo.Flows["flow-1"] = flow

	w, c := newTestContext(http.MethodPost, "/flows/flow-1/deactivate", nil)
	c.Set("tenant_id", "tenant-1")
	c.Params = gin.Params{{Key: "id", Value: "flow-1"}}

	handler.Deactivate(c)

	assert.Equal(t, http.StatusOK, w.Code)

	var rawResp map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &rawResp)
	require.NoError(t, err)
	assert.Equal(t, "Flow deactivated successfully", rawResp["message"])
	assert.False(t, flowRepo.Flows["flow-1"].IsActive)
}

func TestFlowHandler_Deactivate_WrongTenant(t *testing.T) {
	handler, flowRepo := setupFlowTest(t)

	flow := createTestFlow("flow-1", "tenant-2")
	flow.IsActive = true
	flowRepo.Flows["flow-1"] = flow

	w, c := newTestContext(http.MethodPost, "/flows/flow-1/deactivate", nil)
	c.Set("tenant_id", "tenant-1")
	c.Params = gin.Params{{Key: "id", Value: "flow-1"}}

	handler.Deactivate(c)

	assert.Equal(t, http.StatusForbidden, w.Code)
}

func TestFlowHandler_Test(t *testing.T) {
	handler, flowRepo := setupFlowTest(t)

	flowRepo.Flows["flow-1"] = createTestFlow("flow-1", "tenant-1")

	body := TestFlowRequest{
		Inputs: []string{"hello", "help"},
	}

	w, c := newTestContext(http.MethodPost, "/flows/flow-1/test", body)
	c.Set("tenant_id", "tenant-1")
	c.Params = gin.Params{{Key: "id", Value: "flow-1"}}

	handler.Test(c)

	assert.Equal(t, http.StatusOK, w.Code)

	var rawResp map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &rawResp)
	require.NoError(t, err)
	assert.Equal(t, "flow-1", rawResp["flow_id"])
	assert.NotNil(t, rawResp["results"])
}

func TestFlowHandler_Test_WrongTenant(t *testing.T) {
	handler, flowRepo := setupFlowTest(t)

	flowRepo.Flows["flow-1"] = createTestFlow("flow-1", "tenant-2")

	body := TestFlowRequest{
		Inputs: []string{"hello"},
	}

	w, c := newTestContext(http.MethodPost, "/flows/flow-1/test", body)
	c.Set("tenant_id", "tenant-1")
	c.Params = gin.Params{{Key: "id", Value: "flow-1"}}

	handler.Test(c)

	assert.Equal(t, http.StatusForbidden, w.Code)
}

func TestFlowHandler_Test_InvalidBody(t *testing.T) {
	handler, flowRepo := setupFlowTest(t)

	flowRepo.Flows["flow-1"] = createTestFlow("flow-1", "tenant-1")

	w, c := newTestContext(http.MethodPost, "/flows/flow-1/test", nil)
	c.Set("tenant_id", "tenant-1")
	c.Params = gin.Params{{Key: "id", Value: "flow-1"}}
	c.Request.Body = http.NoBody

	handler.Test(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var resp Response
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.False(t, resp.Success)
	assert.Equal(t, "VALIDATION_ERROR", resp.Error.Code)
}
