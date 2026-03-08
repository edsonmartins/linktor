package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/msgfy/linktor/internal/application/service"
	"github.com/msgfy/linktor/internal/domain/entity"
	"github.com/msgfy/linktor/internal/domain/repository"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ============================================================================
// mockKnowledgeBaseRepository
// ============================================================================

type mockKnowledgeBaseRepository struct {
	KBs         map[string]*entity.KnowledgeBase
	ReturnError error
}

func newMockKnowledgeBaseRepository() *mockKnowledgeBaseRepository {
	return &mockKnowledgeBaseRepository{
		KBs: make(map[string]*entity.KnowledgeBase),
	}
}

func (m *mockKnowledgeBaseRepository) Create(ctx context.Context, kb *entity.KnowledgeBase) error {
	if m.ReturnError != nil {
		return m.ReturnError
	}
	m.KBs[kb.ID] = kb
	return nil
}

func (m *mockKnowledgeBaseRepository) FindByID(ctx context.Context, id string) (*entity.KnowledgeBase, error) {
	if m.ReturnError != nil {
		return nil, m.ReturnError
	}
	kb, ok := m.KBs[id]
	if !ok {
		return nil, fmt.Errorf("knowledge base not found: %s", id)
	}
	return kb, nil
}

func (m *mockKnowledgeBaseRepository) FindByTenant(ctx context.Context, tenantID string, params *repository.ListParams) ([]*entity.KnowledgeBase, int64, error) {
	if m.ReturnError != nil {
		return nil, 0, m.ReturnError
	}
	var result []*entity.KnowledgeBase
	for _, kb := range m.KBs {
		if kb.TenantID == tenantID {
			result = append(result, kb)
		}
	}
	return result, int64(len(result)), nil
}

func (m *mockKnowledgeBaseRepository) Update(ctx context.Context, kb *entity.KnowledgeBase) error {
	if m.ReturnError != nil {
		return m.ReturnError
	}
	m.KBs[kb.ID] = kb
	return nil
}

func (m *mockKnowledgeBaseRepository) Delete(ctx context.Context, id string) error {
	if m.ReturnError != nil {
		return m.ReturnError
	}
	delete(m.KBs, id)
	return nil
}

func (m *mockKnowledgeBaseRepository) CountByTenant(ctx context.Context, tenantID string) (int64, error) {
	if m.ReturnError != nil {
		return 0, m.ReturnError
	}
	var count int64
	for _, kb := range m.KBs {
		if kb.TenantID == tenantID {
			count++
		}
	}
	return count, nil
}

// ============================================================================
// mockKnowledgeItemRepository
// ============================================================================

type mockKnowledgeItemRepository struct {
	Items       map[string]*entity.KnowledgeItem
	ReturnError error
}

func newMockKnowledgeItemRepository() *mockKnowledgeItemRepository {
	return &mockKnowledgeItemRepository{
		Items: make(map[string]*entity.KnowledgeItem),
	}
}

func (m *mockKnowledgeItemRepository) Create(ctx context.Context, item *entity.KnowledgeItem) error {
	if m.ReturnError != nil {
		return m.ReturnError
	}
	m.Items[item.ID] = item
	return nil
}

func (m *mockKnowledgeItemRepository) FindByID(ctx context.Context, id string) (*entity.KnowledgeItem, error) {
	if m.ReturnError != nil {
		return nil, m.ReturnError
	}
	item, ok := m.Items[id]
	if !ok {
		return nil, fmt.Errorf("knowledge item not found: %s", id)
	}
	return item, nil
}

func (m *mockKnowledgeItemRepository) FindByKnowledgeBase(ctx context.Context, kbID string, params *repository.ListParams) ([]*entity.KnowledgeItem, int64, error) {
	if m.ReturnError != nil {
		return nil, 0, m.ReturnError
	}
	var result []*entity.KnowledgeItem
	for _, item := range m.Items {
		if item.KnowledgeBaseID == kbID {
			result = append(result, item)
		}
	}
	return result, int64(len(result)), nil
}

func (m *mockKnowledgeItemRepository) Update(ctx context.Context, item *entity.KnowledgeItem) error {
	if m.ReturnError != nil {
		return m.ReturnError
	}
	m.Items[item.ID] = item
	return nil
}

func (m *mockKnowledgeItemRepository) UpdateEmbedding(ctx context.Context, id string, embedding []float64) error {
	if m.ReturnError != nil {
		return m.ReturnError
	}
	return nil
}

func (m *mockKnowledgeItemRepository) Delete(ctx context.Context, id string) error {
	if m.ReturnError != nil {
		return m.ReturnError
	}
	delete(m.Items, id)
	return nil
}

func (m *mockKnowledgeItemRepository) CountByKnowledgeBase(ctx context.Context, kbID string) (int64, error) {
	if m.ReturnError != nil {
		return 0, m.ReturnError
	}
	var count int64
	for _, item := range m.Items {
		if item.KnowledgeBaseID == kbID {
			count++
		}
	}
	return count, nil
}

func (m *mockKnowledgeItemRepository) SearchByKeywords(ctx context.Context, kbID string, keywords []string, limit int) ([]*entity.KnowledgeItem, error) {
	if m.ReturnError != nil {
		return nil, m.ReturnError
	}
	var result []*entity.KnowledgeItem
	for _, item := range m.Items {
		if item.KnowledgeBaseID == kbID {
			result = append(result, item)
		}
	}
	if len(result) > limit {
		result = result[:limit]
	}
	return result, nil
}

func (m *mockKnowledgeItemRepository) SearchByEmbedding(ctx context.Context, kbID string, embedding []float64, limit int, minScore float64) ([]*entity.SearchResult, error) {
	if m.ReturnError != nil {
		return nil, m.ReturnError
	}
	return nil, nil
}

func (m *mockKnowledgeItemRepository) DeleteByKnowledgeBase(ctx context.Context, kbID string) error {
	if m.ReturnError != nil {
		return m.ReturnError
	}
	for id, item := range m.Items {
		if item.KnowledgeBaseID == kbID {
			delete(m.Items, id)
		}
	}
	return nil
}

// ============================================================================
// Setup helper
// ============================================================================

func setupKnowledgeTest(t *testing.T) (*KnowledgeHandler, *mockKnowledgeBaseRepository, *mockKnowledgeItemRepository) {
	t.Helper()
	gin.SetMode(gin.TestMode)

	kbRepo := newMockKnowledgeBaseRepository()
	itemRepo := newMockKnowledgeItemRepository()
	svc := service.NewKnowledgeService(kbRepo, itemRepo, nil, nil)
	handler := NewKnowledgeHandler(svc)

	return handler, kbRepo, itemRepo
}

// ============================================================================
// Tests
// ============================================================================

func TestKnowledgeHandler_ListKnowledgeBases(t *testing.T) {
	handler, kbRepo, _ := setupKnowledgeTest(t)

	kb := entity.NewKnowledgeBase("tenant-1", "Test KB", entity.KnowledgeTypeFAQ)
	kb.ID = "kb-1"
	kbRepo.KBs[kb.ID] = kb

	w, c := newTestContext(http.MethodGet, "/knowledge-bases", nil)
	c.Set("tenant_id", "tenant-1")

	handler.ListKnowledgeBases(c)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.NotNil(t, resp["data"])
}

func TestKnowledgeHandler_CreateKnowledgeBase(t *testing.T) {
	handler, _, _ := setupKnowledgeTest(t)

	body := CreateKnowledgeBaseRequest{
		Name: "My KB",
		Type: "faq",
	}
	w, c := newTestContext(http.MethodPost, "/knowledge-bases", body)
	c.Set("tenant_id", "tenant-1")

	handler.CreateKnowledgeBase(c)

	assert.Equal(t, http.StatusCreated, w.Code)

	var resp Response
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.True(t, resp.Success)
}

func TestKnowledgeHandler_CreateKnowledgeBase_InvalidBody(t *testing.T) {
	handler, _, _ := setupKnowledgeTest(t)

	// Missing required fields
	body := map[string]string{"description": "no name or type"}
	w, c := newTestContext(http.MethodPost, "/knowledge-bases", body)
	c.Set("tenant_id", "tenant-1")

	handler.CreateKnowledgeBase(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var resp Response
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.False(t, resp.Success)
	assert.Equal(t, "VALIDATION_ERROR", resp.Error.Code)
}

func TestKnowledgeHandler_GetKnowledgeBase(t *testing.T) {
	t.Run("found", func(t *testing.T) {
		handler, kbRepo, _ := setupKnowledgeTest(t)

		kb := entity.NewKnowledgeBase("tenant-1", "Test KB", entity.KnowledgeTypeFAQ)
		kb.ID = "kb-1"
		kbRepo.KBs[kb.ID] = kb

		w, c := newTestContext(http.MethodGet, "/knowledge-bases/kb-1", nil)
		c.Params = gin.Params{{Key: "id", Value: "kb-1"}}

		handler.GetKnowledgeBase(c)

		assert.Equal(t, http.StatusOK, w.Code)

		var resp Response
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.True(t, resp.Success)
	})

	t.Run("not found", func(t *testing.T) {
		handler, _, _ := setupKnowledgeTest(t)

		w, c := newTestContext(http.MethodGet, "/knowledge-bases/nonexistent", nil)
		c.Params = gin.Params{{Key: "id", Value: "nonexistent"}}

		handler.GetKnowledgeBase(c)

		assert.NotEqual(t, http.StatusOK, w.Code)
	})
}

func TestKnowledgeHandler_UpdateKnowledgeBase(t *testing.T) {
	handler, kbRepo, _ := setupKnowledgeTest(t)

	kb := entity.NewKnowledgeBase("tenant-1", "Old Name", entity.KnowledgeTypeFAQ)
	kb.ID = "kb-1"
	kbRepo.KBs[kb.ID] = kb

	newName := "New Name"
	body := UpdateKnowledgeBaseRequest{
		Name: &newName,
	}
	w, c := newTestContext(http.MethodPut, "/knowledge-bases/kb-1", body)
	c.Params = gin.Params{{Key: "id", Value: "kb-1"}}

	handler.UpdateKnowledgeBase(c)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp Response
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.True(t, resp.Success)
}

func TestKnowledgeHandler_DeleteKnowledgeBase(t *testing.T) {
	handler, kbRepo, _ := setupKnowledgeTest(t)

	kb := entity.NewKnowledgeBase("tenant-1", "Test KB", entity.KnowledgeTypeFAQ)
	kb.ID = "kb-1"
	kbRepo.KBs[kb.ID] = kb

	w, c := newTestContext(http.MethodDelete, "/knowledge-bases/kb-1", nil)
	c.Params = gin.Params{{Key: "id", Value: "kb-1"}}

	handler.DeleteKnowledgeBase(c)

	// RespondNoContent sets 204; in gin test mode this may present as 200
	assert.Contains(t, []int{http.StatusNoContent, http.StatusOK}, w.Code)
	// Verify the KB was actually deleted
	_, exists := kbRepo.KBs["kb-1"]
	assert.False(t, exists)
}

func TestKnowledgeHandler_AddItem(t *testing.T) {
	handler, kbRepo, _ := setupKnowledgeTest(t)

	kb := entity.NewKnowledgeBase("tenant-1", "Test KB", entity.KnowledgeTypeFAQ)
	kb.ID = "kb-1"
	kbRepo.KBs[kb.ID] = kb

	body := AddItemRequest{
		Question: "What is Linktor?",
		Answer:   "A messaging platform.",
		Keywords: []string{"linktor", "platform"},
	}
	w, c := newTestContext(http.MethodPost, "/knowledge-bases/kb-1/items", body)
	c.Params = gin.Params{{Key: "id", Value: "kb-1"}}

	handler.AddItem(c)

	assert.Equal(t, http.StatusCreated, w.Code)

	var resp Response
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.True(t, resp.Success)
}

func TestKnowledgeHandler_AddItem_InvalidBody(t *testing.T) {
	handler, _, _ := setupKnowledgeTest(t)

	// Missing required fields
	body := map[string]string{"source": "test"}
	w, c := newTestContext(http.MethodPost, "/knowledge-bases/kb-1/items", body)
	c.Params = gin.Params{{Key: "id", Value: "kb-1"}}

	handler.AddItem(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var resp Response
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.False(t, resp.Success)
	assert.Equal(t, "VALIDATION_ERROR", resp.Error.Code)
}

func TestKnowledgeHandler_GetItem(t *testing.T) {
	handler, _, itemRepo := setupKnowledgeTest(t)

	item := entity.NewKnowledgeItem("kb-1", "Question?", "Answer.")
	item.ID = "item-1"
	itemRepo.Items[item.ID] = item

	w, c := newTestContext(http.MethodGet, "/knowledge-bases/kb-1/items/item-1", nil)
	c.Params = gin.Params{
		{Key: "id", Value: "kb-1"},
		{Key: "itemId", Value: "item-1"},
	}

	handler.GetItem(c)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp Response
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.True(t, resp.Success)
}

func TestKnowledgeHandler_DeleteItem(t *testing.T) {
	handler, kbRepo, itemRepo := setupKnowledgeTest(t)

	kb := entity.NewKnowledgeBase("tenant-1", "Test KB", entity.KnowledgeTypeFAQ)
	kb.ID = "kb-1"
	kbRepo.KBs[kb.ID] = kb

	item := entity.NewKnowledgeItem("kb-1", "Question?", "Answer.")
	item.ID = "item-1"
	itemRepo.Items[item.ID] = item

	w, c := newTestContext(http.MethodDelete, "/knowledge-bases/kb-1/items/item-1", nil)
	c.Params = gin.Params{
		{Key: "id", Value: "kb-1"},
		{Key: "itemId", Value: "item-1"},
	}

	handler.DeleteItem(c)

	// RespondNoContent sets 204; in gin test mode this may present as 200
	assert.Contains(t, []int{http.StatusNoContent, http.StatusOK}, w.Code)
	// Verify the item was actually deleted
	_, exists := itemRepo.Items["item-1"]
	assert.False(t, exists)
}

func TestKnowledgeHandler_Search(t *testing.T) {
	handler, kbRepo, itemRepo := setupKnowledgeTest(t)

	kb := entity.NewKnowledgeBase("tenant-1", "Test KB", entity.KnowledgeTypeFAQ)
	kb.ID = "kb-1"
	kbRepo.KBs[kb.ID] = kb

	item := entity.NewKnowledgeItem("kb-1", "What is Linktor?", "A messaging platform.")
	item.ID = "item-1"
	item.Keywords = []string{"linktor", "messaging"}
	itemRepo.Items[item.ID] = item

	body := SearchRequest{
		Query: "linktor",
		Limit: 5,
	}
	w, c := newTestContext(http.MethodPost, "/knowledge-bases/kb-1/search", body)
	c.Params = gin.Params{{Key: "id", Value: "kb-1"}}

	handler.Search(c)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.NotNil(t, resp["results"])
	assert.Equal(t, "linktor", resp["query"])
}

func TestKnowledgeHandler_Search_InvalidBody(t *testing.T) {
	handler, _, _ := setupKnowledgeTest(t)

	// Missing required "query" field
	body := map[string]int{"limit": 5}
	w, c := newTestContext(http.MethodPost, "/knowledge-bases/kb-1/search", body)
	c.Params = gin.Params{{Key: "id", Value: "kb-1"}}

	handler.Search(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var resp Response
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.False(t, resp.Success)
	assert.Equal(t, "VALIDATION_ERROR", resp.Error.Code)
}

func TestKnowledgeHandler_BulkAddItems(t *testing.T) {
	handler, kbRepo, _ := setupKnowledgeTest(t)

	kb := entity.NewKnowledgeBase("tenant-1", "Test KB", entity.KnowledgeTypeFAQ)
	kb.ID = "kb-1"
	kbRepo.KBs[kb.ID] = kb

	body := map[string]interface{}{
		"items": []AddItemRequest{
			{Question: "Q1", Answer: "A1"},
			{Question: "Q2", Answer: "A2"},
		},
	}
	w, c := newTestContext(http.MethodPost, "/knowledge-bases/kb-1/items/bulk", body)
	c.Params = gin.Params{{Key: "id", Value: "kb-1"}}

	handler.BulkAddItems(c)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Equal(t, float64(2), resp["created"])
}

func TestKnowledgeHandler_BulkAddItems_TooMany(t *testing.T) {
	handler, _, _ := setupKnowledgeTest(t)

	// Create 101 items
	items := make([]AddItemRequest, 101)
	for i := range items {
		items[i] = AddItemRequest{
			Question: fmt.Sprintf("Q%d", i),
			Answer:   fmt.Sprintf("A%d", i),
		}
	}
	body := map[string]interface{}{
		"items": items,
	}
	w, c := newTestContext(http.MethodPost, "/knowledge-bases/kb-1/items/bulk", body)
	c.Params = gin.Params{{Key: "id", Value: "kb-1"}}

	handler.BulkAddItems(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var resp Response
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.False(t, resp.Success)
	assert.Equal(t, "VALIDATION_ERROR", resp.Error.Code)
}
