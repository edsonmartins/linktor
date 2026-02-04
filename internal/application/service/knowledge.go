package service

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/msgfy/linktor/internal/domain/entity"
	"github.com/msgfy/linktor/internal/domain/repository"
	"github.com/msgfy/linktor/pkg/errors"
)

// VectorStore defines the interface for vector similarity search
// This abstraction allows swapping pgvector with FalkorDB or other backends
type VectorStore interface {
	// Store stores an embedding with its ID
	Store(ctx context.Context, id string, embedding []float64, metadata map[string]string) error

	// Search performs vector similarity search
	Search(ctx context.Context, embedding []float64, topK int, filter map[string]string) ([]VectorSearchResult, error)

	// Delete removes an embedding by ID
	Delete(ctx context.Context, id string) error

	// DeleteByFilter removes embeddings matching the filter
	DeleteByFilter(ctx context.Context, filter map[string]string) error
}

// VectorSearchResult represents a vector search result
type VectorSearchResult struct {
	ID       string            `json:"id"`
	Score    float64           `json:"score"`
	Metadata map[string]string `json:"metadata"`
}

// KnowledgeService handles knowledge base operations
type KnowledgeService struct {
	kbRepo           repository.KnowledgeBaseRepository
	itemRepo         repository.KnowledgeItemRepository
	embeddingService *EmbeddingService
	vectorStore      VectorStore
}

// NewKnowledgeService creates a new knowledge service
func NewKnowledgeService(
	kbRepo repository.KnowledgeBaseRepository,
	itemRepo repository.KnowledgeItemRepository,
	embeddingService *EmbeddingService,
	vectorStore VectorStore,
) *KnowledgeService {
	return &KnowledgeService{
		kbRepo:           kbRepo,
		itemRepo:         itemRepo,
		embeddingService: embeddingService,
		vectorStore:      vectorStore,
	}
}

// CreateKnowledgeBaseInput represents input for creating a knowledge base
type CreateKnowledgeBaseInput struct {
	TenantID    string
	Name        string
	Description string
	Type        entity.KnowledgeType
	Config      *entity.KnowledgeConfig
}

// CreateKnowledgeBase creates a new knowledge base
func (s *KnowledgeService) CreateKnowledgeBase(ctx context.Context, input *CreateKnowledgeBaseInput) (*entity.KnowledgeBase, error) {
	kb := entity.NewKnowledgeBase(input.TenantID, input.Name, input.Type)
	kb.ID = uuid.New().String()
	kb.Description = input.Description

	if input.Config != nil {
		kb.Config = *input.Config
	}

	if err := s.kbRepo.Create(ctx, kb); err != nil {
		return nil, errors.Wrap(err, errors.ErrCodeInternal, "failed to create knowledge base")
	}

	return kb, nil
}

// GetKnowledgeBase gets a knowledge base by ID
func (s *KnowledgeService) GetKnowledgeBase(ctx context.Context, id string) (*entity.KnowledgeBase, error) {
	return s.kbRepo.FindByID(ctx, id)
}

// ListKnowledgeBases lists knowledge bases for a tenant
func (s *KnowledgeService) ListKnowledgeBases(ctx context.Context, tenantID string, params *repository.ListParams) ([]*entity.KnowledgeBase, int64, error) {
	return s.kbRepo.FindByTenant(ctx, tenantID, params)
}

// UpdateKnowledgeBaseInput represents input for updating a knowledge base
type UpdateKnowledgeBaseInput struct {
	Name        *string
	Description *string
	Config      *entity.KnowledgeConfig
}

// UpdateKnowledgeBase updates a knowledge base
func (s *KnowledgeService) UpdateKnowledgeBase(ctx context.Context, id string, input *UpdateKnowledgeBaseInput) (*entity.KnowledgeBase, error) {
	kb, err := s.kbRepo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if input.Name != nil {
		kb.Name = *input.Name
	}
	if input.Description != nil {
		kb.Description = *input.Description
	}
	if input.Config != nil {
		kb.Config = *input.Config
	}

	kb.UpdatedAt = time.Now()

	if err := s.kbRepo.Update(ctx, kb); err != nil {
		return nil, errors.Wrap(err, errors.ErrCodeInternal, "failed to update knowledge base")
	}

	return kb, nil
}

// DeleteKnowledgeBase deletes a knowledge base and all its items
func (s *KnowledgeService) DeleteKnowledgeBase(ctx context.Context, id string) error {
	// Delete all items first
	items, _, err := s.itemRepo.FindByKnowledgeBase(ctx, id, &repository.ListParams{Page: 1, PageSize: 10000})
	if err != nil {
		return errors.Wrap(err, errors.ErrCodeInternal, "failed to get knowledge items")
	}

	// Delete embeddings from vector store
	if s.vectorStore != nil {
		for _, item := range items {
			s.vectorStore.Delete(ctx, item.ID)
		}
	}

	// Delete all items
	if err := s.itemRepo.DeleteByKnowledgeBase(ctx, id); err != nil {
		return errors.Wrap(err, errors.ErrCodeInternal, "failed to delete knowledge items")
	}

	// Delete knowledge base
	return s.kbRepo.Delete(ctx, id)
}

// AddItemInput represents input for adding a knowledge item
type AddItemInput struct {
	KnowledgeBaseID string
	Question        string
	Answer          string
	Keywords        []string
	Source          string
	Metadata        map[string]string
}

// AddItem adds an item to a knowledge base
func (s *KnowledgeService) AddItem(ctx context.Context, input *AddItemInput) (*entity.KnowledgeItem, error) {
	// Verify knowledge base exists
	kb, err := s.kbRepo.FindByID(ctx, input.KnowledgeBaseID)
	if err != nil {
		return nil, err
	}

	// Create item
	item := entity.NewKnowledgeItem(input.KnowledgeBaseID, input.Question, input.Answer)
	item.ID = uuid.New().String()
	item.Keywords = input.Keywords
	item.Source = input.Source
	item.Metadata = input.Metadata

	// Generate embedding
	if s.embeddingService != nil && s.embeddingService.IsAvailable() {
		// Combine question and answer for embedding
		textToEmbed := input.Question + " " + input.Answer
		embedding, err := s.embeddingService.GenerateEmbedding(ctx, textToEmbed)
		if err != nil {
			// Log but continue - item can be re-embedded later
		} else {
			item.SetEmbedding(embedding)

			// Store in vector store
			if s.vectorStore != nil {
				metadata := map[string]string{
					"knowledge_base_id": input.KnowledgeBaseID,
					"item_id":           item.ID,
				}
				s.vectorStore.Store(ctx, item.ID, embedding, metadata)
			}
		}
	}

	if err := s.itemRepo.Create(ctx, item); err != nil {
		return nil, errors.Wrap(err, errors.ErrCodeInternal, "failed to create knowledge item")
	}

	// Update item count
	kb.IncrementItemCount()
	s.kbRepo.Update(ctx, kb)

	return item, nil
}

// UpdateItemInput represents input for updating a knowledge item
type UpdateItemInput struct {
	Question *string
	Answer   *string
	Keywords []string
	Source   *string
	Metadata map[string]string
}

// UpdateItem updates a knowledge item
func (s *KnowledgeService) UpdateItem(ctx context.Context, id string, input *UpdateItemInput) (*entity.KnowledgeItem, error) {
	item, err := s.itemRepo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}

	needsReembedding := false

	if input.Question != nil {
		item.Question = *input.Question
		needsReembedding = true
	}
	if input.Answer != nil {
		item.Answer = *input.Answer
		needsReembedding = true
	}
	if input.Keywords != nil {
		item.Keywords = input.Keywords
	}
	if input.Source != nil {
		item.Source = *input.Source
	}
	if input.Metadata != nil {
		item.Metadata = input.Metadata
	}

	item.UpdatedAt = time.Now()

	// Re-generate embedding if content changed
	if needsReembedding && s.embeddingService != nil && s.embeddingService.IsAvailable() {
		textToEmbed := item.Question + " " + item.Answer
		embedding, err := s.embeddingService.GenerateEmbedding(ctx, textToEmbed)
		if err == nil {
			item.SetEmbedding(embedding)

			// Update in vector store
			if s.vectorStore != nil {
				metadata := map[string]string{
					"knowledge_base_id": item.KnowledgeBaseID,
					"item_id":           item.ID,
				}
				s.vectorStore.Delete(ctx, item.ID)
				s.vectorStore.Store(ctx, item.ID, embedding, metadata)
			}
		}
	}

	if err := s.itemRepo.Update(ctx, item); err != nil {
		return nil, errors.Wrap(err, errors.ErrCodeInternal, "failed to update knowledge item")
	}

	return item, nil
}

// DeleteItem deletes a knowledge item
func (s *KnowledgeService) DeleteItem(ctx context.Context, id string) error {
	item, err := s.itemRepo.FindByID(ctx, id)
	if err != nil {
		return err
	}

	// Delete from vector store
	if s.vectorStore != nil {
		s.vectorStore.Delete(ctx, item.ID)
	}

	if err := s.itemRepo.Delete(ctx, id); err != nil {
		return errors.Wrap(err, errors.ErrCodeInternal, "failed to delete knowledge item")
	}

	// Update item count
	kb, err := s.kbRepo.FindByID(ctx, item.KnowledgeBaseID)
	if err == nil {
		kb.DecrementItemCount()
		s.kbRepo.Update(ctx, kb)
	}

	return nil
}

// GetItem gets a knowledge item by ID
func (s *KnowledgeService) GetItem(ctx context.Context, id string) (*entity.KnowledgeItem, error) {
	return s.itemRepo.FindByID(ctx, id)
}

// ListItems lists items in a knowledge base
func (s *KnowledgeService) ListItems(ctx context.Context, knowledgeBaseID string, params *repository.ListParams) ([]*entity.KnowledgeItem, int64, error) {
	return s.itemRepo.FindByKnowledgeBase(ctx, knowledgeBaseID, params)
}

// Search performs semantic search on a knowledge base
func (s *KnowledgeService) Search(ctx context.Context, knowledgeBaseID, query string, limit int) ([]entity.SearchResult, error) {
	// Use vector search if available
	if s.embeddingService != nil && s.embeddingService.IsAvailable() && s.vectorStore != nil {
		return s.vectorSearch(ctx, knowledgeBaseID, query, limit)
	}

	// Fallback to keyword search
	return s.keywordSearch(ctx, knowledgeBaseID, query, limit)
}

// vectorSearch performs vector similarity search
func (s *KnowledgeService) vectorSearch(ctx context.Context, knowledgeBaseID, query string, limit int) ([]entity.SearchResult, error) {
	// Generate query embedding
	queryEmbedding, err := s.embeddingService.GenerateEmbedding(ctx, query)
	if err != nil {
		// Fallback to keyword search
		return s.keywordSearch(ctx, knowledgeBaseID, query, limit)
	}

	// Search in vector store
	filter := map[string]string{
		"knowledge_base_id": knowledgeBaseID,
	}

	vectorResults, err := s.vectorStore.Search(ctx, queryEmbedding, limit, filter)
	if err != nil {
		return nil, errors.Wrap(err, errors.ErrCodeInternal, "vector search failed")
	}

	// Convert to SearchResult
	results := make([]entity.SearchResult, 0, len(vectorResults))
	for _, vr := range vectorResults {
		item, err := s.itemRepo.FindByID(ctx, vr.ID)
		if err != nil {
			continue
		}
		results = append(results, entity.SearchResult{
			Item:  item,
			Score: vr.Score,
		})
	}

	return results, nil
}

// keywordSearch performs keyword-based search
func (s *KnowledgeService) keywordSearch(ctx context.Context, knowledgeBaseID, query string, limit int) ([]entity.SearchResult, error) {
	// Split query into keywords
	keywords := splitKeywords(query)

	items, err := s.itemRepo.SearchByKeywords(ctx, knowledgeBaseID, keywords, limit)
	if err != nil {
		return nil, errors.Wrap(err, errors.ErrCodeInternal, "keyword search failed")
	}

	results := make([]entity.SearchResult, len(items))
	for i, item := range items {
		results[i] = entity.SearchResult{
			Item:  item,
			Score: 1.0, // No score for keyword search
		}
	}

	return results, nil
}

// splitKeywords splits a query into keywords
func splitKeywords(query string) []string {
	// Simple split by spaces - in production this would be more sophisticated
	var keywords []string
	var current string
	for _, c := range query {
		if c == ' ' || c == ',' || c == ';' {
			if current != "" {
				keywords = append(keywords, current)
				current = ""
			}
		} else {
			current += string(c)
		}
	}
	if current != "" {
		keywords = append(keywords, current)
	}
	return keywords
}

// RegenerateEmbeddings regenerates embeddings for all items in a knowledge base
func (s *KnowledgeService) RegenerateEmbeddings(ctx context.Context, knowledgeBaseID string) error {
	if s.embeddingService == nil || !s.embeddingService.IsAvailable() {
		return errors.New(errors.ErrCodeBadRequest, "embedding service not available")
	}

	// Get all items
	items, _, err := s.itemRepo.FindByKnowledgeBase(ctx, knowledgeBaseID, &repository.ListParams{Page: 1, PageSize: 10000})
	if err != nil {
		return err
	}

	kb, err := s.kbRepo.FindByID(ctx, knowledgeBaseID)
	if err != nil {
		return err
	}

	kb.MarkSyncing()
	s.kbRepo.Update(ctx, kb)

	// Regenerate embeddings
	for _, item := range items {
		textToEmbed := item.Question + " " + item.Answer
		embedding, err := s.embeddingService.GenerateEmbedding(ctx, textToEmbed)
		if err != nil {
			continue
		}

		item.SetEmbedding(embedding)
		s.itemRepo.Update(ctx, item)

		// Update vector store
		if s.vectorStore != nil {
			metadata := map[string]string{
				"knowledge_base_id": knowledgeBaseID,
				"item_id":           item.ID,
			}
			s.vectorStore.Delete(ctx, item.ID)
			s.vectorStore.Store(ctx, item.ID, embedding, metadata)
		}
	}

	kb.MarkSynced()
	s.kbRepo.Update(ctx, kb)

	return nil
}

// Note: KnowledgeService implements usecase.KnowledgeSearchService interface
// The interface is defined in generate_ai_response.go
