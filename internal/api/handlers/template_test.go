package handlers

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/msgfy/linktor/internal/application/service"
	"github.com/msgfy/linktor/internal/domain/entity"
	"github.com/msgfy/linktor/pkg/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupTemplateTest(t *testing.T) (*TemplateHandler, *mockTemplateRepository) {
	t.Helper()

	templateRepo := newMockTemplateRepository()
	channelRepo := testutil.NewMockChannelRepository()

	templateService := service.NewTemplateService(templateRepo, channelRepo)
	handler := NewTemplateHandler(templateService)

	return handler, templateRepo
}

func TestTemplateHandler_List(t *testing.T) {
	handler, templateRepo := setupTemplateTest(t)

	templateRepo.Templates["tmpl-1"] = &entity.Template{
		ID:        "tmpl-1",
		TenantID:  "tenant-1",
		ChannelID: "channel-1",
		Name:      "welcome",
		Language:  "en",
		Category:  entity.TemplateCategoryUtility,
		Status:    entity.TemplateStatusApproved,
	}
	templateRepo.Templates["tmpl-2"] = &entity.Template{
		ID:        "tmpl-2",
		TenantID:  "tenant-1",
		ChannelID: "channel-1",
		Name:      "order_update",
		Language:  "en",
		Category:  entity.TemplateCategoryMarketing,
		Status:    entity.TemplateStatusPending,
	}

	w, c := newTestContext(http.MethodGet, "/templates", nil)
	c.Set("tenant_id", "tenant-1")

	handler.List(c)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp Response
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.True(t, resp.Success)
	assert.NotNil(t, resp.Data)

	dataList, ok := resp.Data.([]interface{})
	require.True(t, ok)
	assert.Len(t, dataList, 2)
}

func TestTemplateHandler_List_ByChannel(t *testing.T) {
	handler, templateRepo := setupTemplateTest(t)

	templateRepo.Templates["tmpl-1"] = &entity.Template{
		ID:        "tmpl-1",
		TenantID:  "tenant-1",
		ChannelID: "channel-1",
		Name:      "welcome",
		Language:  "en",
		Category:  entity.TemplateCategoryUtility,
		Status:    entity.TemplateStatusApproved,
	}
	templateRepo.Templates["tmpl-2"] = &entity.Template{
		ID:        "tmpl-2",
		TenantID:  "tenant-1",
		ChannelID: "channel-2",
		Name:      "order_update",
		Language:  "en",
		Category:  entity.TemplateCategoryMarketing,
		Status:    entity.TemplateStatusPending,
	}

	w, c := newTestContext(http.MethodGet, "/templates", nil)
	c.Set("tenant_id", "tenant-1")
	c.Request.URL.RawQuery = "channel_id=channel-1"

	handler.List(c)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp Response
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.True(t, resp.Success)

	dataList, ok := resp.Data.([]interface{})
	require.True(t, ok)
	assert.Len(t, dataList, 1)
}

func TestTemplateHandler_List_NoTenantID(t *testing.T) {
	handler, _ := setupTemplateTest(t)

	w, c := newTestContext(http.MethodGet, "/templates", nil)

	handler.List(c)

	assert.NotEqual(t, http.StatusOK, w.Code)
}

func TestTemplateHandler_Create(t *testing.T) {
	handler, _ := setupTemplateTest(t)

	body := CreateTemplateRequest{
		ChannelID: "channel-1",
		Name:      "new_template",
		Language:  "en",
		Category:  "UTILITY",
		Components: []entity.TemplateComponent{
			{
				Type: "BODY",
				Text: "Hello {{1}}, your order {{2}} is ready.",
			},
		},
	}

	w, c := newTestContext(http.MethodPost, "/templates", body)
	c.Set("tenant_id", "tenant-1")

	handler.Create(c)

	assert.Equal(t, http.StatusCreated, w.Code)

	var resp Response
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.True(t, resp.Success)

	dataMap := resp.Data.(map[string]interface{})
	assert.Equal(t, "new_template", dataMap["name"])
	assert.Equal(t, "en", dataMap["language"])
	assert.Equal(t, "UTILITY", dataMap["category"])
}

func TestTemplateHandler_Create_InvalidBody(t *testing.T) {
	handler, _ := setupTemplateTest(t)

	// Missing required fields
	body := map[string]string{
		"name": "incomplete",
	}

	w, c := newTestContext(http.MethodPost, "/templates", body)
	c.Set("tenant_id", "tenant-1")

	handler.Create(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var resp Response
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.False(t, resp.Success)
	assert.Equal(t, "VALIDATION_ERROR", resp.Error.Code)
}

func TestTemplateHandler_Create_NoTenantID(t *testing.T) {
	handler, _ := setupTemplateTest(t)

	body := CreateTemplateRequest{
		ChannelID: "channel-1",
		Name:      "new_template",
		Language:  "en",
		Category:  "UTILITY",
		Components: []entity.TemplateComponent{
			{Type: "BODY", Text: "Hello"},
		},
	}

	w, c := newTestContext(http.MethodPost, "/templates", body)

	handler.Create(c)

	assert.NotEqual(t, http.StatusCreated, w.Code)
}

func TestTemplateHandler_Get(t *testing.T) {
	handler, templateRepo := setupTemplateTest(t)

	templateRepo.Templates["tmpl-1"] = &entity.Template{
		ID:        "tmpl-1",
		TenantID:  "tenant-1",
		ChannelID: "channel-1",
		Name:      "welcome",
		Language:  "en",
		Category:  entity.TemplateCategoryUtility,
		Status:    entity.TemplateStatusApproved,
	}

	w, c := newTestContext(http.MethodGet, "/templates/tmpl-1", nil)
	c.Set("tenant_id", "tenant-1")
	c.Params = gin.Params{{Key: "id", Value: "tmpl-1"}}

	handler.Get(c)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp Response
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.True(t, resp.Success)

	dataMap := resp.Data.(map[string]interface{})
	assert.Equal(t, "tmpl-1", dataMap["id"])
	assert.Equal(t, "welcome", dataMap["name"])
}

func TestTemplateHandler_Get_NotFound(t *testing.T) {
	handler, _ := setupTemplateTest(t)

	w, c := newTestContext(http.MethodGet, "/templates/nonexistent", nil)
	c.Set("tenant_id", "tenant-1")
	c.Params = gin.Params{{Key: "id", Value: "nonexistent"}}

	handler.Get(c)

	assert.NotEqual(t, http.StatusOK, w.Code)

	var resp Response
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.False(t, resp.Success)
}

func TestTemplateHandler_Get_EmptyID(t *testing.T) {
	handler, _ := setupTemplateTest(t)

	w, c := newTestContext(http.MethodGet, "/templates/", nil)
	c.Set("tenant_id", "tenant-1")

	handler.Get(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var resp Response
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.False(t, resp.Success)
	assert.Equal(t, "VALIDATION_ERROR", resp.Error.Code)
}

func TestTemplateHandler_Delete(t *testing.T) {
	handler, templateRepo := setupTemplateTest(t)

	templateRepo.Templates["tmpl-1"] = &entity.Template{
		ID:        "tmpl-1",
		TenantID:  "tenant-1",
		ChannelID: "channel-1",
		Name:      "welcome",
		Language:  "en",
		Category:  entity.TemplateCategoryUtility,
		Status:    entity.TemplateStatusApproved,
	}

	w, c := newTestContext(http.MethodDelete, "/templates/tmpl-1", nil)
	c.Set("tenant_id", "tenant-1")
	c.Params = gin.Params{{Key: "id", Value: "tmpl-1"}}

	handler.Delete(c)

	gotCode := w.Code
	if gotCode == http.StatusOK {
		gotCode = c.Writer.Status()
	}
	assert.Equal(t, http.StatusNoContent, gotCode)
	assert.Empty(t, templateRepo.Templates)
}

func TestTemplateHandler_Delete_NotFound(t *testing.T) {
	handler, _ := setupTemplateTest(t)

	w, c := newTestContext(http.MethodDelete, "/templates/nonexistent", nil)
	c.Set("tenant_id", "tenant-1")
	c.Params = gin.Params{{Key: "id", Value: "nonexistent"}}

	handler.Delete(c)

	gotCode := w.Code
	if gotCode == http.StatusOK {
		gotCode = c.Writer.Status()
	}
	assert.NotEqual(t, http.StatusNoContent, gotCode)

	var resp Response
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.False(t, resp.Success)
}
