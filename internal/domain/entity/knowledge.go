package entity

import (
	"time"
)

// KnowledgeType represents the type of knowledge base
type KnowledgeType string

const (
	KnowledgeTypeFAQ       KnowledgeType = "faq"
	KnowledgeTypeDocuments KnowledgeType = "documents"
	KnowledgeTypeWebsite   KnowledgeType = "website"
)

// KnowledgeStatus represents the status of a knowledge base
type KnowledgeStatus string

const (
	KnowledgeStatusActive   KnowledgeStatus = "active"
	KnowledgeStatusInactive KnowledgeStatus = "inactive"
	KnowledgeStatusSyncing  KnowledgeStatus = "syncing"
)

// KnowledgeConfig holds configuration for the knowledge base
type KnowledgeConfig struct {
	// For document type
	AllowedFileTypes []string `json:"allowed_file_types,omitempty"`
	MaxFileSize      int64    `json:"max_file_size,omitempty"`

	// For website type
	CrawlURLs        []string `json:"crawl_urls,omitempty"`
	CrawlDepth       int      `json:"crawl_depth,omitempty"`
	CrawlFrequency   string   `json:"crawl_frequency,omitempty"` // daily, weekly, monthly

	// Common
	EmbeddingModel   string `json:"embedding_model,omitempty"`
	ChunkSize        int    `json:"chunk_size,omitempty"`
	ChunkOverlap     int    `json:"chunk_overlap,omitempty"`
}

// KnowledgeBase represents a knowledge base for RAG
type KnowledgeBase struct {
	ID          string          `json:"id"`
	TenantID    string          `json:"tenant_id"`
	Name        string          `json:"name"`
	Description string          `json:"description,omitempty"`
	Type        KnowledgeType   `json:"type"`   // faq, documents, website
	Config      KnowledgeConfig `json:"config"`
	Status      KnowledgeStatus `json:"status"`
	ItemCount   int             `json:"item_count"`
	LastSyncAt  *time.Time      `json:"last_sync_at,omitempty"`
	CreatedAt   time.Time       `json:"created_at"`
	UpdatedAt   time.Time       `json:"updated_at"`
}

// NewKnowledgeBase creates a new knowledge base
func NewKnowledgeBase(tenantID, name string, kbType KnowledgeType) *KnowledgeBase {
	now := time.Now()
	return &KnowledgeBase{
		TenantID:  tenantID,
		Name:      name,
		Type:      kbType,
		Config:    KnowledgeConfig{},
		Status:    KnowledgeStatusActive,
		ItemCount: 0,
		CreatedAt: now,
		UpdatedAt: now,
	}
}

// IsActive returns true if the knowledge base is active
func (kb *KnowledgeBase) IsActive() bool {
	return kb.Status == KnowledgeStatusActive
}

// MarkSyncing marks the knowledge base as syncing
func (kb *KnowledgeBase) MarkSyncing() {
	kb.Status = KnowledgeStatusSyncing
	kb.UpdatedAt = time.Now()
}

// MarkSynced marks the knowledge base as synced
func (kb *KnowledgeBase) MarkSynced() {
	now := time.Now()
	kb.Status = KnowledgeStatusActive
	kb.LastSyncAt = &now
	kb.UpdatedAt = now
}

// IncrementItemCount increments the item count
func (kb *KnowledgeBase) IncrementItemCount() {
	kb.ItemCount++
	kb.UpdatedAt = time.Now()
}

// DecrementItemCount decrements the item count
func (kb *KnowledgeBase) DecrementItemCount() {
	if kb.ItemCount > 0 {
		kb.ItemCount--
	}
	kb.UpdatedAt = time.Now()
}

// SetItemCount sets the item count
func (kb *KnowledgeBase) SetItemCount(count int) {
	kb.ItemCount = count
	kb.UpdatedAt = time.Now()
}

// KnowledgeItem represents an item in the knowledge base
type KnowledgeItem struct {
	ID              string    `json:"id"`
	KnowledgeBaseID string    `json:"knowledge_base_id"`
	Question        string    `json:"question"`
	Answer          string    `json:"answer"`
	Keywords        []string  `json:"keywords,omitempty"`
	Embedding       []float64 `json:"embedding,omitempty"` // Vector for RAG
	Source          string    `json:"source,omitempty"`    // Original source URL or file
	Metadata        map[string]string `json:"metadata,omitempty"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

// NewKnowledgeItem creates a new knowledge item
func NewKnowledgeItem(knowledgeBaseID, question, answer string) *KnowledgeItem {
	now := time.Now()
	return &KnowledgeItem{
		KnowledgeBaseID: knowledgeBaseID,
		Question:        question,
		Answer:          answer,
		Keywords:        []string{},
		Metadata:        make(map[string]string),
		CreatedAt:       now,
		UpdatedAt:       now,
	}
}

// SetEmbedding sets the embedding vector
func (ki *KnowledgeItem) SetEmbedding(embedding []float64) {
	ki.Embedding = embedding
	ki.UpdatedAt = time.Now()
}

// HasEmbedding returns true if the item has an embedding
func (ki *KnowledgeItem) HasEmbedding() bool {
	return len(ki.Embedding) > 0
}

// AddKeyword adds a keyword to the item
func (ki *KnowledgeItem) AddKeyword(keyword string) {
	// Check if already exists
	for _, kw := range ki.Keywords {
		if kw == keyword {
			return
		}
	}
	ki.Keywords = append(ki.Keywords, keyword)
	ki.UpdatedAt = time.Now()
}

// SetSource sets the source of the item
func (ki *KnowledgeItem) SetSource(source string) {
	ki.Source = source
	ki.UpdatedAt = time.Now()
}

// SetMetadata sets a metadata value
func (ki *KnowledgeItem) SetMetadata(key, value string) {
	if ki.Metadata == nil {
		ki.Metadata = make(map[string]string)
	}
	ki.Metadata[key] = value
	ki.UpdatedAt = time.Now()
}

// SearchResult represents a search result from the knowledge base
type SearchResult struct {
	Item       *KnowledgeItem `json:"item"`
	Score      float64        `json:"score"`
	Highlights []string       `json:"highlights,omitempty"`
}

// KnowledgeSearchRequest represents a search request
type KnowledgeSearchRequest struct {
	Query           string   `json:"query"`
	KnowledgeBaseID string   `json:"knowledge_base_id"`
	TopK            int      `json:"top_k,omitempty"`
	MinScore        float64  `json:"min_score,omitempty"`
	IncludeKeywords bool     `json:"include_keywords,omitempty"`
}

// KnowledgeSearchResponse represents a search response
type KnowledgeSearchResponse struct {
	Results      []SearchResult `json:"results"`
	TotalFound   int            `json:"total_found"`
	SearchTimeMs int64          `json:"search_time_ms"`
}
