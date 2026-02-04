package database

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/msgfy/linktor/internal/application/service"
	"github.com/msgfy/linktor/internal/domain/entity"
	"github.com/msgfy/linktor/internal/domain/repository"
	"github.com/msgfy/linktor/pkg/errors"
)

// KnowledgeBaseRepository implements repository.KnowledgeBaseRepository with PostgreSQL
type KnowledgeBaseRepository struct {
	db *PostgresDB
}

// NewKnowledgeBaseRepository creates a new knowledge base repository
func NewKnowledgeBaseRepository(db *PostgresDB) *KnowledgeBaseRepository {
	return &KnowledgeBaseRepository{db: db}
}

// Create creates a new knowledge base
func (r *KnowledgeBaseRepository) Create(ctx context.Context, kb *entity.KnowledgeBase) error {
	configJSON, err := json.Marshal(kb.Config)
	if err != nil {
		return errors.Wrap(err, errors.ErrCodeInternal, "failed to marshal config")
	}

	query := `
		INSERT INTO knowledge_bases (
			id, tenant_id, name, description, type, config, status,
			item_count, last_sync_at, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
	`

	_, err = r.db.Pool.Exec(ctx, query,
		kb.ID,
		kb.TenantID,
		kb.Name,
		kb.Description,
		string(kb.Type),
		configJSON,
		string(kb.Status),
		kb.ItemCount,
		kb.LastSyncAt,
		kb.CreatedAt,
		kb.UpdatedAt,
	)

	if err != nil {
		return errors.Wrap(err, errors.ErrCodeInternal, "failed to create knowledge base")
	}

	return nil
}

// FindByID finds a knowledge base by ID
func (r *KnowledgeBaseRepository) FindByID(ctx context.Context, id string) (*entity.KnowledgeBase, error) {
	query := `
		SELECT id, tenant_id, name, description, type, config, status,
		       item_count, last_sync_at, created_at, updated_at
		FROM knowledge_bases
		WHERE id = $1
	`

	kb, err := r.scanKnowledgeBase(r.db.Pool.QueryRow(ctx, query, id))
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, errors.New(errors.ErrCodeNotFound, "knowledge base not found")
		}
		return nil, errors.Wrap(err, errors.ErrCodeInternal, "failed to find knowledge base")
	}

	return kb, nil
}

// FindByTenant finds knowledge bases for a tenant with pagination
func (r *KnowledgeBaseRepository) FindByTenant(ctx context.Context, tenantID string, params *repository.ListParams) ([]*entity.KnowledgeBase, int64, error) {
	// Count total
	var total int64
	if err := r.db.Pool.QueryRow(ctx,
		"SELECT COUNT(*) FROM knowledge_bases WHERE tenant_id = $1",
		tenantID,
	).Scan(&total); err != nil {
		return nil, 0, errors.Wrap(err, errors.ErrCodeInternal, "failed to count knowledge bases")
	}

	// Get knowledge bases
	query := fmt.Sprintf(`
		SELECT id, tenant_id, name, description, type, config, status,
		       item_count, last_sync_at, created_at, updated_at
		FROM knowledge_bases
		WHERE tenant_id = $1
		ORDER BY %s %s
		LIMIT $2 OFFSET $3
	`, sanitizeKBColumn(params.SortBy), sanitizeDirection(params.SortDir))

	rows, err := r.db.Pool.Query(ctx, query, tenantID, params.Limit(), params.Offset())
	if err != nil {
		return nil, 0, errors.Wrap(err, errors.ErrCodeInternal, "failed to query knowledge bases")
	}
	defer rows.Close()

	var kbs []*entity.KnowledgeBase
	for rows.Next() {
		kb, err := r.scanKnowledgeBaseFromRows(rows)
		if err != nil {
			return nil, 0, err
		}
		kbs = append(kbs, kb)
	}

	return kbs, total, nil
}

// Update updates a knowledge base
func (r *KnowledgeBaseRepository) Update(ctx context.Context, kb *entity.KnowledgeBase) error {
	configJSON, err := json.Marshal(kb.Config)
	if err != nil {
		return errors.Wrap(err, errors.ErrCodeInternal, "failed to marshal config")
	}

	kb.UpdatedAt = time.Now()

	query := `
		UPDATE knowledge_bases SET
			name = $1,
			description = $2,
			type = $3,
			config = $4,
			status = $5,
			item_count = $6,
			last_sync_at = $7,
			updated_at = $8
		WHERE id = $9
	`

	result, err := r.db.Pool.Exec(ctx, query,
		kb.Name,
		kb.Description,
		string(kb.Type),
		configJSON,
		string(kb.Status),
		kb.ItemCount,
		kb.LastSyncAt,
		kb.UpdatedAt,
		kb.ID,
	)

	if err != nil {
		return errors.Wrap(err, errors.ErrCodeInternal, "failed to update knowledge base")
	}

	if result.RowsAffected() == 0 {
		return errors.New(errors.ErrCodeNotFound, "knowledge base not found")
	}

	return nil
}

// Delete deletes a knowledge base
func (r *KnowledgeBaseRepository) Delete(ctx context.Context, id string) error {
	result, err := r.db.Pool.Exec(ctx, "DELETE FROM knowledge_bases WHERE id = $1", id)
	if err != nil {
		return errors.Wrap(err, errors.ErrCodeInternal, "failed to delete knowledge base")
	}

	if result.RowsAffected() == 0 {
		return errors.New(errors.ErrCodeNotFound, "knowledge base not found")
	}

	return nil
}

// CountByTenant counts knowledge bases for a tenant
func (r *KnowledgeBaseRepository) CountByTenant(ctx context.Context, tenantID string) (int64, error) {
	var count int64
	err := r.db.Pool.QueryRow(ctx,
		"SELECT COUNT(*) FROM knowledge_bases WHERE tenant_id = $1",
		tenantID,
	).Scan(&count)

	if err != nil {
		return 0, errors.Wrap(err, errors.ErrCodeInternal, "failed to count knowledge bases")
	}

	return count, nil
}

// Helper methods

func (r *KnowledgeBaseRepository) scanKnowledgeBase(row pgx.Row) (*entity.KnowledgeBase, error) {
	var kb entity.KnowledgeBase
	var kbType, status string
	var configJSON []byte
	var description *string

	err := row.Scan(
		&kb.ID, &kb.TenantID, &kb.Name, &description, &kbType, &configJSON, &status,
		&kb.ItemCount, &kb.LastSyncAt, &kb.CreatedAt, &kb.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}

	kb.Type = entity.KnowledgeType(kbType)
	kb.Status = entity.KnowledgeStatus(status)
	if description != nil {
		kb.Description = *description
	}

	if len(configJSON) > 0 {
		json.Unmarshal(configJSON, &kb.Config)
	}

	return &kb, nil
}

func (r *KnowledgeBaseRepository) scanKnowledgeBaseFromRows(rows pgx.Rows) (*entity.KnowledgeBase, error) {
	var kb entity.KnowledgeBase
	var kbType, status string
	var configJSON []byte
	var description *string

	err := rows.Scan(
		&kb.ID, &kb.TenantID, &kb.Name, &description, &kbType, &configJSON, &status,
		&kb.ItemCount, &kb.LastSyncAt, &kb.CreatedAt, &kb.UpdatedAt,
	)
	if err != nil {
		return nil, errors.Wrap(err, errors.ErrCodeInternal, "failed to scan knowledge base")
	}

	kb.Type = entity.KnowledgeType(kbType)
	kb.Status = entity.KnowledgeStatus(status)
	if description != nil {
		kb.Description = *description
	}

	if len(configJSON) > 0 {
		json.Unmarshal(configJSON, &kb.Config)
	}

	return &kb, nil
}

func sanitizeKBColumn(col string) string {
	allowed := map[string]bool{
		"created_at":   true,
		"updated_at":   true,
		"name":         true,
		"type":         true,
		"status":       true,
		"item_count":   true,
		"last_sync_at": true,
	}
	if allowed[col] {
		return col
	}
	return "created_at"
}

// KnowledgeItemRepository implements repository.KnowledgeItemRepository with PostgreSQL
type KnowledgeItemRepository struct {
	db *PostgresDB
}

// NewKnowledgeItemRepository creates a new knowledge item repository
func NewKnowledgeItemRepository(db *PostgresDB) *KnowledgeItemRepository {
	return &KnowledgeItemRepository{db: db}
}

// Create creates a new knowledge item
func (r *KnowledgeItemRepository) Create(ctx context.Context, item *entity.KnowledgeItem) error {
	metadataJSON, err := json.Marshal(item.Metadata)
	if err != nil {
		return errors.Wrap(err, errors.ErrCodeInternal, "failed to marshal metadata")
	}

	query := `
		INSERT INTO knowledge_items (
			id, knowledge_base_id, question, answer, keywords,
			embedding, source, metadata, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
	`

	var embeddingStr *string
	if len(item.Embedding) > 0 {
		s := vectorToString(item.Embedding)
		embeddingStr = &s
	}

	_, err = r.db.Pool.Exec(ctx, query,
		item.ID,
		item.KnowledgeBaseID,
		item.Question,
		item.Answer,
		item.Keywords,
		embeddingStr,
		item.Source,
		metadataJSON,
		item.CreatedAt,
		item.UpdatedAt,
	)

	if err != nil {
		return errors.Wrap(err, errors.ErrCodeInternal, "failed to create knowledge item")
	}

	return nil
}

// FindByID finds a knowledge item by ID
func (r *KnowledgeItemRepository) FindByID(ctx context.Context, id string) (*entity.KnowledgeItem, error) {
	query := `
		SELECT id, knowledge_base_id, question, answer, keywords,
		       embedding::text, source, metadata, created_at, updated_at
		FROM knowledge_items
		WHERE id = $1
	`

	item, err := r.scanKnowledgeItem(r.db.Pool.QueryRow(ctx, query, id))
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, errors.New(errors.ErrCodeNotFound, "knowledge item not found")
		}
		return nil, errors.Wrap(err, errors.ErrCodeInternal, "failed to find knowledge item")
	}

	return item, nil
}

// FindByKnowledgeBase finds items for a knowledge base with pagination
func (r *KnowledgeItemRepository) FindByKnowledgeBase(ctx context.Context, kbID string, params *repository.ListParams) ([]*entity.KnowledgeItem, int64, error) {
	// Count total
	var total int64
	if err := r.db.Pool.QueryRow(ctx,
		"SELECT COUNT(*) FROM knowledge_items WHERE knowledge_base_id = $1",
		kbID,
	).Scan(&total); err != nil {
		return nil, 0, errors.Wrap(err, errors.ErrCodeInternal, "failed to count knowledge items")
	}

	// Get items
	query := fmt.Sprintf(`
		SELECT id, knowledge_base_id, question, answer, keywords,
		       embedding::text, source, metadata, created_at, updated_at
		FROM knowledge_items
		WHERE knowledge_base_id = $1
		ORDER BY %s %s
		LIMIT $2 OFFSET $3
	`, sanitizeKIColumn(params.SortBy), sanitizeDirection(params.SortDir))

	rows, err := r.db.Pool.Query(ctx, query, kbID, params.Limit(), params.Offset())
	if err != nil {
		return nil, 0, errors.Wrap(err, errors.ErrCodeInternal, "failed to query knowledge items")
	}
	defer rows.Close()

	var items []*entity.KnowledgeItem
	for rows.Next() {
		item, err := r.scanKnowledgeItemFromRows(rows)
		if err != nil {
			return nil, 0, err
		}
		items = append(items, item)
	}

	return items, total, nil
}

// Update updates a knowledge item
func (r *KnowledgeItemRepository) Update(ctx context.Context, item *entity.KnowledgeItem) error {
	metadataJSON, err := json.Marshal(item.Metadata)
	if err != nil {
		return errors.Wrap(err, errors.ErrCodeInternal, "failed to marshal metadata")
	}

	item.UpdatedAt = time.Now()

	var embeddingStr *string
	if len(item.Embedding) > 0 {
		s := vectorToString(item.Embedding)
		embeddingStr = &s
	}

	query := `
		UPDATE knowledge_items SET
			question = $1,
			answer = $2,
			keywords = $3,
			embedding = $4,
			source = $5,
			metadata = $6,
			updated_at = $7
		WHERE id = $8
	`

	result, err := r.db.Pool.Exec(ctx, query,
		item.Question,
		item.Answer,
		item.Keywords,
		embeddingStr,
		item.Source,
		metadataJSON,
		item.UpdatedAt,
		item.ID,
	)

	if err != nil {
		return errors.Wrap(err, errors.ErrCodeInternal, "failed to update knowledge item")
	}

	if result.RowsAffected() == 0 {
		return errors.New(errors.ErrCodeNotFound, "knowledge item not found")
	}

	return nil
}

// UpdateEmbedding updates only the embedding
func (r *KnowledgeItemRepository) UpdateEmbedding(ctx context.Context, id string, embedding []float64) error {
	var embeddingStr *string
	if len(embedding) > 0 {
		s := vectorToString(embedding)
		embeddingStr = &s
	}

	query := `UPDATE knowledge_items SET embedding = $1, updated_at = $2 WHERE id = $3`
	result, err := r.db.Pool.Exec(ctx, query, embeddingStr, time.Now(), id)
	if err != nil {
		return errors.Wrap(err, errors.ErrCodeInternal, "failed to update embedding")
	}

	if result.RowsAffected() == 0 {
		return errors.New(errors.ErrCodeNotFound, "knowledge item not found")
	}

	return nil
}

// Delete deletes a knowledge item
func (r *KnowledgeItemRepository) Delete(ctx context.Context, id string) error {
	result, err := r.db.Pool.Exec(ctx, "DELETE FROM knowledge_items WHERE id = $1", id)
	if err != nil {
		return errors.Wrap(err, errors.ErrCodeInternal, "failed to delete knowledge item")
	}

	if result.RowsAffected() == 0 {
		return errors.New(errors.ErrCodeNotFound, "knowledge item not found")
	}

	return nil
}

// CountByKnowledgeBase counts items in a knowledge base
func (r *KnowledgeItemRepository) CountByKnowledgeBase(ctx context.Context, kbID string) (int64, error) {
	var count int64
	err := r.db.Pool.QueryRow(ctx,
		"SELECT COUNT(*) FROM knowledge_items WHERE knowledge_base_id = $1",
		kbID,
	).Scan(&count)

	if err != nil {
		return 0, errors.Wrap(err, errors.ErrCodeInternal, "failed to count knowledge items")
	}

	return count, nil
}

// SearchByKeywords searches items by keywords
func (r *KnowledgeItemRepository) SearchByKeywords(ctx context.Context, kbID string, keywords []string, limit int) ([]*entity.KnowledgeItem, error) {
	if len(keywords) == 0 {
		return []*entity.KnowledgeItem{}, nil
	}

	// Build search query - search in question, answer, and keywords array
	// Using full-text search or ILIKE for simplicity
	searchConditions := make([]string, 0)
	args := []interface{}{kbID}
	argIndex := 2

	for _, kw := range keywords {
		searchConditions = append(searchConditions,
			fmt.Sprintf("(question ILIKE $%d OR answer ILIKE $%d OR $%d = ANY(keywords))",
				argIndex, argIndex, argIndex+1))
		args = append(args, "%"+kw+"%", kw)
		argIndex += 2
	}

	query := fmt.Sprintf(`
		SELECT id, knowledge_base_id, question, answer, keywords,
		       embedding::text, source, metadata, created_at, updated_at
		FROM knowledge_items
		WHERE knowledge_base_id = $1 AND (%s)
		ORDER BY created_at DESC
		LIMIT %d
	`, strings.Join(searchConditions, " OR "), limit)

	rows, err := r.db.Pool.Query(ctx, query, args...)
	if err != nil {
		return nil, errors.Wrap(err, errors.ErrCodeInternal, "failed to search knowledge items")
	}
	defer rows.Close()

	var items []*entity.KnowledgeItem
	for rows.Next() {
		item, err := r.scanKnowledgeItemFromRows(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}

	return items, nil
}

// SearchByEmbedding searches items by vector similarity (RAG)
func (r *KnowledgeItemRepository) SearchByEmbedding(ctx context.Context, kbID string, embedding []float64, limit int, minScore float64) ([]*entity.SearchResult, error) {
	if len(embedding) == 0 {
		return []*entity.SearchResult{}, nil
	}

	embeddingStr := vectorToString(embedding)

	// Use pgvector's cosine distance operator <=>
	// The operator returns distance, so we convert to similarity: 1 - distance
	query := fmt.Sprintf(`
		SELECT id, knowledge_base_id, question, answer, keywords,
		       embedding::text, source, metadata, created_at, updated_at,
		       1 - (embedding <=> '%s') as similarity
		FROM knowledge_items
		WHERE knowledge_base_id = $1
		  AND embedding IS NOT NULL
		  AND 1 - (embedding <=> '%s') >= $2
		ORDER BY embedding <=> '%s'
		LIMIT $3
	`, embeddingStr, embeddingStr, embeddingStr)

	rows, err := r.db.Pool.Query(ctx, query, kbID, minScore, limit)
	if err != nil {
		return nil, errors.Wrap(err, errors.ErrCodeInternal, "failed to search by embedding")
	}
	defer rows.Close()

	var results []*entity.SearchResult
	for rows.Next() {
		var item entity.KnowledgeItem
		var embeddingText *string
		var metadataJSON []byte
		var similarity float64

		err := rows.Scan(
			&item.ID, &item.KnowledgeBaseID, &item.Question, &item.Answer, &item.Keywords,
			&embeddingText, &item.Source, &metadataJSON, &item.CreatedAt, &item.UpdatedAt,
			&similarity,
		)
		if err != nil {
			return nil, errors.Wrap(err, errors.ErrCodeInternal, "failed to scan search result")
		}

		if embeddingText != nil {
			item.Embedding = stringToVector(*embeddingText)
		}
		if len(metadataJSON) > 0 {
			json.Unmarshal(metadataJSON, &item.Metadata)
		}

		results = append(results, &entity.SearchResult{
			Item:  &item,
			Score: similarity,
		})
	}

	return results, nil
}

// DeleteByKnowledgeBase deletes all items in a knowledge base
func (r *KnowledgeItemRepository) DeleteByKnowledgeBase(ctx context.Context, kbID string) error {
	_, err := r.db.Pool.Exec(ctx, "DELETE FROM knowledge_items WHERE knowledge_base_id = $1", kbID)
	if err != nil {
		return errors.Wrap(err, errors.ErrCodeInternal, "failed to delete knowledge items")
	}
	return nil
}

// Helper methods

func (r *KnowledgeItemRepository) scanKnowledgeItem(row pgx.Row) (*entity.KnowledgeItem, error) {
	var item entity.KnowledgeItem
	var embeddingText *string
	var metadataJSON []byte

	err := row.Scan(
		&item.ID, &item.KnowledgeBaseID, &item.Question, &item.Answer, &item.Keywords,
		&embeddingText, &item.Source, &metadataJSON, &item.CreatedAt, &item.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}

	if embeddingText != nil {
		item.Embedding = stringToVector(*embeddingText)
	}
	if len(metadataJSON) > 0 {
		json.Unmarshal(metadataJSON, &item.Metadata)
	}

	return &item, nil
}

func (r *KnowledgeItemRepository) scanKnowledgeItemFromRows(rows pgx.Rows) (*entity.KnowledgeItem, error) {
	var item entity.KnowledgeItem
	var embeddingText *string
	var metadataJSON []byte

	err := rows.Scan(
		&item.ID, &item.KnowledgeBaseID, &item.Question, &item.Answer, &item.Keywords,
		&embeddingText, &item.Source, &metadataJSON, &item.CreatedAt, &item.UpdatedAt,
	)
	if err != nil {
		return nil, errors.Wrap(err, errors.ErrCodeInternal, "failed to scan knowledge item")
	}

	if embeddingText != nil {
		item.Embedding = stringToVector(*embeddingText)
	}
	if len(metadataJSON) > 0 {
		json.Unmarshal(metadataJSON, &item.Metadata)
	}

	return &item, nil
}

func sanitizeKIColumn(col string) string {
	allowed := map[string]bool{
		"created_at": true,
		"updated_at": true,
		"question":   true,
		"source":     true,
	}
	if allowed[col] {
		return col
	}
	return "created_at"
}

// vectorToString converts a float64 slice to pgvector string format
func vectorToString(v []float64) string {
	if len(v) == 0 {
		return ""
	}
	parts := make([]string, len(v))
	for i, f := range v {
		parts[i] = fmt.Sprintf("%f", f)
	}
	return "[" + strings.Join(parts, ",") + "]"
}

// stringToVector converts a pgvector string to float64 slice
func stringToVector(s string) []float64 {
	if s == "" {
		return nil
	}
	// Remove brackets
	s = strings.TrimPrefix(s, "[")
	s = strings.TrimSuffix(s, "]")

	parts := strings.Split(s, ",")
	result := make([]float64, len(parts))
	for i, p := range parts {
		var f float64
		fmt.Sscanf(strings.TrimSpace(p), "%f", &f)
		result[i] = f
	}
	return result
}

// PgVectorStore implements VectorStore using pgvector
// This is a simple implementation; for FalkorDB, create a similar struct
type PgVectorStore struct {
	db *PostgresDB
}

// NewPgVectorStore creates a new pgvector store
func NewPgVectorStore(db *PostgresDB) *PgVectorStore {
	return &PgVectorStore{db: db}
}

// Store stores an embedding with its ID
func (s *PgVectorStore) Store(ctx context.Context, id string, embedding []float64, metadata map[string]string) error {
	// For pgvector, the embedding is stored in the knowledge_items table
	// This is a no-op since we store embeddings directly in the item
	return nil
}

// Search performs vector similarity search
func (s *PgVectorStore) Search(ctx context.Context, embedding []float64, topK int, filter map[string]string) ([]service.VectorSearchResult, error) {
	kbID := filter["knowledge_base_id"]
	if kbID == "" {
		return nil, errors.New(errors.ErrCodeBadRequest, "knowledge_base_id filter required")
	}

	embeddingStr := vectorToString(embedding)

	query := fmt.Sprintf(`
		SELECT id, 1 - (embedding <=> '%s') as similarity
		FROM knowledge_items
		WHERE knowledge_base_id = $1
		  AND embedding IS NOT NULL
		ORDER BY embedding <=> '%s'
		LIMIT $2
	`, embeddingStr, embeddingStr)

	rows, err := s.db.Pool.Query(ctx, query, kbID, topK)
	if err != nil {
		return nil, errors.Wrap(err, errors.ErrCodeInternal, "vector search failed")
	}
	defer rows.Close()

	var results []service.VectorSearchResult
	for rows.Next() {
		var r service.VectorSearchResult
		var similarity float64
		if err := rows.Scan(&r.ID, &similarity); err != nil {
			return nil, errors.Wrap(err, errors.ErrCodeInternal, "failed to scan result")
		}
		r.Score = similarity
		results = append(results, r)
	}

	return results, nil
}

// Delete removes an embedding by ID
func (s *PgVectorStore) Delete(ctx context.Context, id string) error {
	// Embedding is deleted with the item
	return nil
}

// DeleteByFilter removes embeddings matching the filter
func (s *PgVectorStore) DeleteByFilter(ctx context.Context, filter map[string]string) error {
	// Embeddings are deleted with items
	return nil
}
