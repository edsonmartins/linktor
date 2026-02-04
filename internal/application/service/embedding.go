package service

import (
	"context"
	"fmt"

	"github.com/msgfy/linktor/internal/domain/entity"
)

// EmbeddingService handles embedding generation for RAG
type EmbeddingService struct {
	aiFactory *AIProviderFactory
	config    *EmbeddingConfig
}

// EmbeddingConfig holds configuration for the embedding service
type EmbeddingConfig struct {
	DefaultProvider entity.AIProviderType
	DefaultModel    string
	Dimensions      int
}

// DefaultEmbeddingConfig returns default configuration
func DefaultEmbeddingConfig() *EmbeddingConfig {
	return &EmbeddingConfig{
		DefaultProvider: entity.AIProviderOpenAI,
		DefaultModel:    "text-embedding-ada-002",
		Dimensions:      1536, // OpenAI ada-002 dimensions
	}
}

// NewEmbeddingService creates a new embedding service
func NewEmbeddingService(aiFactory *AIProviderFactory, config *EmbeddingConfig) *EmbeddingService {
	if config == nil {
		config = DefaultEmbeddingConfig()
	}
	return &EmbeddingService{
		aiFactory: aiFactory,
		config:    config,
	}
}

// GenerateEmbedding generates an embedding for the given text
func (s *EmbeddingService) GenerateEmbedding(ctx context.Context, text string) ([]float64, error) {
	return s.GenerateEmbeddingWithProvider(ctx, text, s.config.DefaultProvider, s.config.DefaultModel)
}

// GenerateEmbeddingWithProvider generates an embedding using a specific provider
func (s *EmbeddingService) GenerateEmbeddingWithProvider(
	ctx context.Context,
	text string,
	providerType entity.AIProviderType,
	model string,
) ([]float64, error) {
	provider, err := s.aiFactory.Get(providerType)
	if err != nil {
		return nil, fmt.Errorf("failed to get AI provider: %w", err)
	}

	req := &EmbeddingRequest{
		Text:  text,
		Model: model,
	}

	resp, err := provider.Embed(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to generate embedding: %w", err)
	}

	return resp.Embedding, nil
}

// GenerateBatchEmbeddings generates embeddings for multiple texts
func (s *EmbeddingService) GenerateBatchEmbeddings(ctx context.Context, texts []string) ([][]float64, error) {
	embeddings := make([][]float64, len(texts))

	// Generate embeddings one by one (could be optimized with batch API if provider supports it)
	for i, text := range texts {
		embedding, err := s.GenerateEmbedding(ctx, text)
		if err != nil {
			return nil, fmt.Errorf("failed to generate embedding for text %d: %w", i, err)
		}
		embeddings[i] = embedding
	}

	return embeddings, nil
}

// GetDimensions returns the embedding dimensions for the current provider/model
func (s *EmbeddingService) GetDimensions() int {
	return s.config.Dimensions
}

// IsAvailable checks if embedding service is available
func (s *EmbeddingService) IsAvailable() bool {
	provider, err := s.aiFactory.Get(s.config.DefaultProvider)
	if err != nil {
		return false
	}
	return provider.IsAvailable()
}

// CosineSimilarity calculates cosine similarity between two embeddings
func CosineSimilarity(a, b []float64) float64 {
	if len(a) != len(b) || len(a) == 0 {
		return 0
	}

	var dotProduct, normA, normB float64
	for i := range a {
		dotProduct += a[i] * b[i]
		normA += a[i] * a[i]
		normB += b[i] * b[i]
	}

	if normA == 0 || normB == 0 {
		return 0
	}

	return dotProduct / (sqrt(normA) * sqrt(normB))
}

// sqrt is a simple square root implementation
func sqrt(x float64) float64 {
	if x <= 0 {
		return 0
	}
	z := x / 2
	for i := 0; i < 10; i++ {
		z = z - (z*z-x)/(2*z)
	}
	return z
}
