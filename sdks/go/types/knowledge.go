package types

// KnowledgeBaseStatus type
type KnowledgeBaseStatus string

const (
	KnowledgeBaseStatusActive     KnowledgeBaseStatus = "active"
	KnowledgeBaseStatusProcessing KnowledgeBaseStatus = "processing"
	KnowledgeBaseStatusError      KnowledgeBaseStatus = "error"
	KnowledgeBaseStatusEmpty      KnowledgeBaseStatus = "empty"
)

// KnowledgeBase model
type KnowledgeBase struct {
	ID             string                 `json:"id"`
	TenantID       string                 `json:"tenantId"`
	Name           string                 `json:"name"`
	Description    string                 `json:"description,omitempty"`
	Status         KnowledgeBaseStatus    `json:"status"`
	EmbeddingModel string                 `json:"embeddingModel"`
	ChunkSize      int                    `json:"chunkSize"`
	ChunkOverlap   int                    `json:"chunkOverlap"`
	DocumentCount  int                    `json:"documentCount"`
	TotalChunks    int                    `json:"totalChunks"`
	Metadata       map[string]interface{} `json:"metadata,omitempty"`
	Timestamps
}

// DocumentStatus type
type DocumentStatus string

const (
	DocumentStatusPending    DocumentStatus = "pending"
	DocumentStatusProcessing DocumentStatus = "processing"
	DocumentStatusCompleted  DocumentStatus = "completed"
	DocumentStatusFailed     DocumentStatus = "failed"
)

// Document model
type Document struct {
	ID              string                 `json:"id"`
	KnowledgeBaseID string                 `json:"knowledgeBaseId"`
	Name            string                 `json:"name"`
	Type            string                 `json:"type"`
	SourceURL       string                 `json:"sourceUrl,omitempty"`
	Status          DocumentStatus         `json:"status"`
	Size            int64                  `json:"size"`
	ChunkCount      int                    `json:"chunkCount"`
	Metadata        map[string]interface{} `json:"metadata,omitempty"`
	Error           string                 `json:"error,omitempty"`
	Timestamps
}

// ScoredChunk with similarity score
type ScoredChunk struct {
	ID         string                 `json:"id"`
	DocumentID string                 `json:"documentId"`
	Content    string                 `json:"content"`
	ChunkIndex int                    `json:"chunkIndex"`
	TokenCount int                    `json:"tokenCount"`
	Score      float64                `json:"score"`
	Metadata   map[string]interface{} `json:"metadata,omitempty"`
	Document   *Document              `json:"document,omitempty"`
}

// QueryResult from knowledge base query
type QueryResult struct {
	Chunks []ScoredChunk `json:"chunks"`
	Query  string        `json:"query"`
	Model  string        `json:"model"`
}

// CreateKnowledgeBaseInput for creating knowledge bases
type CreateKnowledgeBaseInput struct {
	Name           string                 `json:"name"`
	Description    string                 `json:"description,omitempty"`
	EmbeddingModel string                 `json:"embeddingModel,omitempty"`
	ChunkSize      int                    `json:"chunkSize,omitempty"`
	ChunkOverlap   int                    `json:"chunkOverlap,omitempty"`
	Metadata       map[string]interface{} `json:"metadata,omitempty"`
}
