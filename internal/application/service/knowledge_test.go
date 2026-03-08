package service

import (
	"context"
	"testing"

	"github.com/msgfy/linktor/internal/domain/entity"
	"github.com/msgfy/linktor/internal/domain/repository"
	"github.com/msgfy/linktor/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ============================================================================
// Mock Repositories for Knowledge Tests
// ============================================================================

type mockKnowledgeBaseRepo struct {
	bases       map[string]*entity.KnowledgeBase
	ReturnError error
}

func newMockKnowledgeBaseRepo() *mockKnowledgeBaseRepo {
	return &mockKnowledgeBaseRepo{
		bases: make(map[string]*entity.KnowledgeBase),
	}
}

func (m *mockKnowledgeBaseRepo) Create(ctx context.Context, kb *entity.KnowledgeBase) error {
	if m.ReturnError != nil {
		return m.ReturnError
	}
	m.bases[kb.ID] = kb
	return nil
}

func (m *mockKnowledgeBaseRepo) FindByID(ctx context.Context, id string) (*entity.KnowledgeBase, error) {
	if m.ReturnError != nil {
		return nil, m.ReturnError
	}
	kb, ok := m.bases[id]
	if !ok {
		return nil, errors.New(errors.ErrCodeNotFound, "knowledge base not found")
	}
	return kb, nil
}

func (m *mockKnowledgeBaseRepo) FindByTenant(ctx context.Context, tenantID string, params *repository.ListParams) ([]*entity.KnowledgeBase, int64, error) {
	if m.ReturnError != nil {
		return nil, 0, m.ReturnError
	}
	var result []*entity.KnowledgeBase
	for _, kb := range m.bases {
		if kb.TenantID == tenantID {
			result = append(result, kb)
		}
	}
	return result, int64(len(result)), nil
}

func (m *mockKnowledgeBaseRepo) Update(ctx context.Context, kb *entity.KnowledgeBase) error {
	if m.ReturnError != nil {
		return m.ReturnError
	}
	m.bases[kb.ID] = kb
	return nil
}

func (m *mockKnowledgeBaseRepo) Delete(ctx context.Context, id string) error {
	if m.ReturnError != nil {
		return m.ReturnError
	}
	delete(m.bases, id)
	return nil
}

func (m *mockKnowledgeBaseRepo) CountByTenant(ctx context.Context, tenantID string) (int64, error) {
	if m.ReturnError != nil {
		return 0, m.ReturnError
	}
	var count int64
	for _, kb := range m.bases {
		if kb.TenantID == tenantID {
			count++
		}
	}
	return count, nil
}

// ---

type mockKnowledgeItemRepo struct {
	items       map[string]*entity.KnowledgeItem
	ReturnError error
}

func newMockKnowledgeItemRepo() *mockKnowledgeItemRepo {
	return &mockKnowledgeItemRepo{
		items: make(map[string]*entity.KnowledgeItem),
	}
}

func (m *mockKnowledgeItemRepo) Create(ctx context.Context, item *entity.KnowledgeItem) error {
	if m.ReturnError != nil {
		return m.ReturnError
	}
	m.items[item.ID] = item
	return nil
}

func (m *mockKnowledgeItemRepo) FindByID(ctx context.Context, id string) (*entity.KnowledgeItem, error) {
	if m.ReturnError != nil {
		return nil, m.ReturnError
	}
	item, ok := m.items[id]
	if !ok {
		return nil, errors.New(errors.ErrCodeNotFound, "knowledge item not found")
	}
	return item, nil
}

func (m *mockKnowledgeItemRepo) FindByKnowledgeBase(ctx context.Context, kbID string, params *repository.ListParams) ([]*entity.KnowledgeItem, int64, error) {
	if m.ReturnError != nil {
		return nil, 0, m.ReturnError
	}
	var result []*entity.KnowledgeItem
	for _, item := range m.items {
		if item.KnowledgeBaseID == kbID {
			result = append(result, item)
		}
	}
	return result, int64(len(result)), nil
}

func (m *mockKnowledgeItemRepo) Update(ctx context.Context, item *entity.KnowledgeItem) error {
	if m.ReturnError != nil {
		return m.ReturnError
	}
	m.items[item.ID] = item
	return nil
}

func (m *mockKnowledgeItemRepo) UpdateEmbedding(ctx context.Context, id string, embedding []float64) error {
	if m.ReturnError != nil {
		return m.ReturnError
	}
	item, ok := m.items[id]
	if !ok {
		return errors.New(errors.ErrCodeNotFound, "not found")
	}
	item.Embedding = embedding
	return nil
}

func (m *mockKnowledgeItemRepo) Delete(ctx context.Context, id string) error {
	if m.ReturnError != nil {
		return m.ReturnError
	}
	delete(m.items, id)
	return nil
}

func (m *mockKnowledgeItemRepo) CountByKnowledgeBase(ctx context.Context, kbID string) (int64, error) {
	if m.ReturnError != nil {
		return 0, m.ReturnError
	}
	var count int64
	for _, item := range m.items {
		if item.KnowledgeBaseID == kbID {
			count++
		}
	}
	return count, nil
}

func (m *mockKnowledgeItemRepo) SearchByKeywords(ctx context.Context, kbID string, keywords []string, limit int) ([]*entity.KnowledgeItem, error) {
	if m.ReturnError != nil {
		return nil, m.ReturnError
	}
	// Simple mock: return all items in the KB up to limit
	var result []*entity.KnowledgeItem
	for _, item := range m.items {
		if item.KnowledgeBaseID == kbID {
			result = append(result, item)
			if limit > 0 && len(result) >= limit {
				break
			}
		}
	}
	return result, nil
}

func (m *mockKnowledgeItemRepo) SearchByEmbedding(ctx context.Context, kbID string, embedding []float64, limit int, minScore float64) ([]*entity.SearchResult, error) {
	if m.ReturnError != nil {
		return nil, m.ReturnError
	}
	return []*entity.SearchResult{}, nil
}

func (m *mockKnowledgeItemRepo) DeleteByKnowledgeBase(ctx context.Context, kbID string) error {
	if m.ReturnError != nil {
		return m.ReturnError
	}
	for id, item := range m.items {
		if item.KnowledgeBaseID == kbID {
			delete(m.items, id)
		}
	}
	return nil
}

// ============================================================================
// Tests
// ============================================================================

func TestNewKnowledgeService(t *testing.T) {
	kbRepo := newMockKnowledgeBaseRepo()
	itemRepo := newMockKnowledgeItemRepo()

	svc := NewKnowledgeService(kbRepo, itemRepo, nil, nil)
	require.NotNil(t, svc)
	assert.Equal(t, kbRepo, svc.kbRepo)
	assert.Equal(t, itemRepo, svc.itemRepo)
	assert.Nil(t, svc.embeddingService)
	assert.Nil(t, svc.vectorStore)
}

func TestKnowledgeService_CreateKnowledgeBase(t *testing.T) {
	kbRepo := newMockKnowledgeBaseRepo()
	itemRepo := newMockKnowledgeItemRepo()
	svc := NewKnowledgeService(kbRepo, itemRepo, nil, nil)
	ctx := context.Background()

	input := &CreateKnowledgeBaseInput{
		TenantID:    "tenant-1",
		Name:        "FAQ Base",
		Description: "Frequently asked questions",
		Type:        entity.KnowledgeTypeFAQ,
	}

	kb, err := svc.CreateKnowledgeBase(ctx, input)
	require.NoError(t, err)
	require.NotNil(t, kb)
	assert.NotEmpty(t, kb.ID)
	assert.Equal(t, "tenant-1", kb.TenantID)
	assert.Equal(t, "FAQ Base", kb.Name)
	assert.Equal(t, "Frequently asked questions", kb.Description)
	assert.Equal(t, entity.KnowledgeTypeFAQ, kb.Type)

	// Verify persisted
	stored, ok := kbRepo.bases[kb.ID]
	require.True(t, ok)
	assert.Equal(t, kb.ID, stored.ID)
}

func TestKnowledgeService_GetKnowledgeBase(t *testing.T) {
	kbRepo := newMockKnowledgeBaseRepo()
	itemRepo := newMockKnowledgeItemRepo()
	svc := NewKnowledgeService(kbRepo, itemRepo, nil, nil)
	ctx := context.Background()

	t.Run("found", func(t *testing.T) {
		input := &CreateKnowledgeBaseInput{
			TenantID: "tenant-1",
			Name:     "Test KB",
			Type:     entity.KnowledgeTypeFAQ,
		}
		created, err := svc.CreateKnowledgeBase(ctx, input)
		require.NoError(t, err)

		found, err := svc.GetKnowledgeBase(ctx, created.ID)
		require.NoError(t, err)
		assert.Equal(t, created.ID, found.ID)
	})

	t.Run("not found", func(t *testing.T) {
		_, err := svc.GetKnowledgeBase(ctx, "nonexistent-id")
		require.Error(t, err)
	})
}

func TestKnowledgeService_UpdateKnowledgeBase(t *testing.T) {
	kbRepo := newMockKnowledgeBaseRepo()
	itemRepo := newMockKnowledgeItemRepo()
	svc := NewKnowledgeService(kbRepo, itemRepo, nil, nil)
	ctx := context.Background()

	// Create KB first
	input := &CreateKnowledgeBaseInput{
		TenantID: "tenant-1",
		Name:     "Original Name",
		Type:     entity.KnowledgeTypeFAQ,
	}
	created, err := svc.CreateKnowledgeBase(ctx, input)
	require.NoError(t, err)

	newName := "Updated Name"
	newDesc := "Updated description"
	updateInput := &UpdateKnowledgeBaseInput{
		Name:        &newName,
		Description: &newDesc,
	}

	updated, err := svc.UpdateKnowledgeBase(ctx, created.ID, updateInput)
	require.NoError(t, err)
	assert.Equal(t, "Updated Name", updated.Name)
	assert.Equal(t, "Updated description", updated.Description)
}

func TestKnowledgeService_DeleteKnowledgeBase(t *testing.T) {
	kbRepo := newMockKnowledgeBaseRepo()
	itemRepo := newMockKnowledgeItemRepo()
	svc := NewKnowledgeService(kbRepo, itemRepo, nil, nil)
	ctx := context.Background()

	input := &CreateKnowledgeBaseInput{
		TenantID: "tenant-1",
		Name:     "To Delete",
		Type:     entity.KnowledgeTypeFAQ,
	}
	created, err := svc.CreateKnowledgeBase(ctx, input)
	require.NoError(t, err)

	err = svc.DeleteKnowledgeBase(ctx, created.ID)
	require.NoError(t, err)

	_, ok := kbRepo.bases[created.ID]
	assert.False(t, ok)
}

func TestKnowledgeService_AddItem(t *testing.T) {
	kbRepo := newMockKnowledgeBaseRepo()
	itemRepo := newMockKnowledgeItemRepo()
	svc := NewKnowledgeService(kbRepo, itemRepo, nil, nil) // no embedding service
	ctx := context.Background()

	// Create KB first
	kbInput := &CreateKnowledgeBaseInput{
		TenantID: "tenant-1",
		Name:     "Test KB",
		Type:     entity.KnowledgeTypeFAQ,
	}
	kb, err := svc.CreateKnowledgeBase(ctx, kbInput)
	require.NoError(t, err)

	itemInput := &AddItemInput{
		KnowledgeBaseID: kb.ID,
		Question:        "What is your return policy?",
		Answer:          "You can return items within 30 days.",
		Keywords:        []string{"return", "policy", "refund"},
		Source:          "faq-page",
	}

	item, err := svc.AddItem(ctx, itemInput)
	require.NoError(t, err)
	require.NotNil(t, item)
	assert.NotEmpty(t, item.ID)
	assert.Equal(t, kb.ID, item.KnowledgeBaseID)
	assert.Equal(t, "What is your return policy?", item.Question)
	assert.Equal(t, "You can return items within 30 days.", item.Answer)
	assert.Equal(t, []string{"return", "policy", "refund"}, item.Keywords)
	assert.Equal(t, "faq-page", item.Source)

	// No embedding since service is nil
	assert.Empty(t, item.Embedding)

	// Item count should be incremented
	updatedKB := kbRepo.bases[kb.ID]
	assert.Equal(t, 1, updatedKB.ItemCount)
}

func TestKnowledgeService_GetItem(t *testing.T) {
	kbRepo := newMockKnowledgeBaseRepo()
	itemRepo := newMockKnowledgeItemRepo()
	svc := NewKnowledgeService(kbRepo, itemRepo, nil, nil)
	ctx := context.Background()

	kbInput := &CreateKnowledgeBaseInput{
		TenantID: "tenant-1",
		Name:     "Test KB",
		Type:     entity.KnowledgeTypeFAQ,
	}
	kb, err := svc.CreateKnowledgeBase(ctx, kbInput)
	require.NoError(t, err)

	itemInput := &AddItemInput{
		KnowledgeBaseID: kb.ID,
		Question:        "Test Q",
		Answer:          "Test A",
	}
	created, err := svc.AddItem(ctx, itemInput)
	require.NoError(t, err)

	found, err := svc.GetItem(ctx, created.ID)
	require.NoError(t, err)
	assert.Equal(t, created.ID, found.ID)
	assert.Equal(t, "Test Q", found.Question)
}

func TestKnowledgeService_UpdateItem(t *testing.T) {
	kbRepo := newMockKnowledgeBaseRepo()
	itemRepo := newMockKnowledgeItemRepo()
	svc := NewKnowledgeService(kbRepo, itemRepo, nil, nil)
	ctx := context.Background()

	kbInput := &CreateKnowledgeBaseInput{
		TenantID: "tenant-1",
		Name:     "Test KB",
		Type:     entity.KnowledgeTypeFAQ,
	}
	kb, err := svc.CreateKnowledgeBase(ctx, kbInput)
	require.NoError(t, err)

	itemInput := &AddItemInput{
		KnowledgeBaseID: kb.ID,
		Question:        "Original Q",
		Answer:          "Original A",
	}
	created, err := svc.AddItem(ctx, itemInput)
	require.NoError(t, err)

	newQuestion := "Updated Q"
	newAnswer := "Updated A"
	updateInput := &UpdateItemInput{
		Question: &newQuestion,
		Answer:   &newAnswer,
	}

	updated, err := svc.UpdateItem(ctx, created.ID, updateInput)
	require.NoError(t, err)
	assert.Equal(t, "Updated Q", updated.Question)
	assert.Equal(t, "Updated A", updated.Answer)
}

func TestKnowledgeService_DeleteItem(t *testing.T) {
	kbRepo := newMockKnowledgeBaseRepo()
	itemRepo := newMockKnowledgeItemRepo()
	svc := NewKnowledgeService(kbRepo, itemRepo, nil, nil)
	ctx := context.Background()

	kbInput := &CreateKnowledgeBaseInput{
		TenantID: "tenant-1",
		Name:     "Test KB",
		Type:     entity.KnowledgeTypeFAQ,
	}
	kb, err := svc.CreateKnowledgeBase(ctx, kbInput)
	require.NoError(t, err)

	itemInput := &AddItemInput{
		KnowledgeBaseID: kb.ID,
		Question:        "To delete",
		Answer:          "Will be deleted",
	}
	created, err := svc.AddItem(ctx, itemInput)
	require.NoError(t, err)

	err = svc.DeleteItem(ctx, created.ID)
	require.NoError(t, err)

	_, ok := itemRepo.items[created.ID]
	assert.False(t, ok)

	// Item count should be decremented
	updatedKB := kbRepo.bases[kb.ID]
	assert.Equal(t, 0, updatedKB.ItemCount)
}

func TestKnowledgeService_Search_KeywordFallback(t *testing.T) {
	kbRepo := newMockKnowledgeBaseRepo()
	itemRepo := newMockKnowledgeItemRepo()
	svc := NewKnowledgeService(kbRepo, itemRepo, nil, nil) // no embedding, no vector store
	ctx := context.Background()

	kbInput := &CreateKnowledgeBaseInput{
		TenantID: "tenant-1",
		Name:     "Test KB",
		Type:     entity.KnowledgeTypeFAQ,
	}
	kb, err := svc.CreateKnowledgeBase(ctx, kbInput)
	require.NoError(t, err)

	// Add items
	for _, q := range []string{"return policy", "shipping info", "payment methods"} {
		_, err := svc.AddItem(ctx, &AddItemInput{
			KnowledgeBaseID: kb.ID,
			Question:        q,
			Answer:          "Answer for " + q,
		})
		require.NoError(t, err)
	}

	results, err := svc.Search(ctx, kb.ID, "return policy", 10)
	require.NoError(t, err)
	assert.NotEmpty(t, results)
	// All results should have score 1.0 for keyword search
	for _, r := range results {
		assert.Equal(t, 1.0, r.Score)
	}
}

func TestSplitKeywords(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []string
	}{
		{"space separated", "hello world", []string{"hello", "world"}},
		{"comma separated", "hello,world", []string{"hello", "world"}},
		{"semicolon separated", "hello;world", []string{"hello", "world"}},
		{"mixed separators", "hello world,foo;bar", []string{"hello", "world", "foo", "bar"}},
		{"single word", "hello", []string{"hello"}},
		{"empty string", "", nil},
		{"multiple spaces", "hello  world", []string{"hello", "world"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := splitKeywords(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}
