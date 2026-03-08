package service

import (
	"math"
	"testing"

	"github.com/msgfy/linktor/internal/domain/entity"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDefaultEmbeddingConfig(t *testing.T) {
	cfg := DefaultEmbeddingConfig()
	require.NotNil(t, cfg)
	assert.Equal(t, entity.AIProviderOpenAI, cfg.DefaultProvider)
	assert.Equal(t, "text-embedding-ada-002", cfg.DefaultModel)
	assert.Equal(t, 1536, cfg.Dimensions)
}

func TestNewEmbeddingService(t *testing.T) {
	factory := NewAIProviderFactory()

	t.Run("with nil config uses default", func(t *testing.T) {
		svc := NewEmbeddingService(factory, nil)
		require.NotNil(t, svc)
		assert.Equal(t, 1536, svc.config.Dimensions)
		assert.Equal(t, entity.AIProviderOpenAI, svc.config.DefaultProvider)
	})

	t.Run("with custom config", func(t *testing.T) {
		cfg := &EmbeddingConfig{
			DefaultProvider: entity.AIProviderAnthropic,
			DefaultModel:    "custom-model",
			Dimensions:      768,
		}
		svc := NewEmbeddingService(factory, cfg)
		require.NotNil(t, svc)
		assert.Equal(t, 768, svc.config.Dimensions)
		assert.Equal(t, entity.AIProviderAnthropic, svc.config.DefaultProvider)
		assert.Equal(t, "custom-model", svc.config.DefaultModel)
	})
}

func TestEmbeddingService_GetDimensions(t *testing.T) {
	factory := NewAIProviderFactory()

	t.Run("default dimensions", func(t *testing.T) {
		svc := NewEmbeddingService(factory, nil)
		assert.Equal(t, 1536, svc.GetDimensions())
	})

	t.Run("custom dimensions", func(t *testing.T) {
		cfg := &EmbeddingConfig{Dimensions: 512}
		svc := NewEmbeddingService(factory, cfg)
		assert.Equal(t, 512, svc.GetDimensions())
	})
}

func TestCosineSimilarity(t *testing.T) {
	tests := []struct {
		name     string
		a        []float64
		b        []float64
		expected float64
		delta    float64
	}{
		{
			name:     "identical vectors",
			a:        []float64{1, 0, 0},
			b:        []float64{1, 0, 0},
			expected: 1.0,
			delta:    0.01,
		},
		{
			name:     "orthogonal vectors",
			a:        []float64{1, 0, 0},
			b:        []float64{0, 1, 0},
			expected: 0.0,
			delta:    0.01,
		},
		{
			name:     "different lengths returns 0",
			a:        []float64{1, 0},
			b:        []float64{1, 0, 0},
			expected: 0.0,
			delta:    0.01,
		},
		{
			name:     "zero vectors",
			a:        []float64{0, 0, 0},
			b:        []float64{0, 0, 0},
			expected: 0.0,
			delta:    0.01,
		},
		{
			name:     "similar vectors",
			a:        []float64{1, 2, 3},
			b:        []float64{1, 2, 4},
			expected: 0.99,
			delta:    0.02,
		},
		{
			name:     "opposite vectors",
			a:        []float64{1, 0, 0},
			b:        []float64{-1, 0, 0},
			expected: -1.0,
			delta:    0.01,
		},
		{
			name:     "empty vectors",
			a:        []float64{},
			b:        []float64{},
			expected: 0.0,
			delta:    0.01,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := CosineSimilarity(tt.a, tt.b)
			assert.InDelta(t, tt.expected, result, tt.delta)
		})
	}
}

func TestEmbeddingService_IsAvailable_NoProvider(t *testing.T) {
	factory := NewAIProviderFactory()
	svc := NewEmbeddingService(factory, nil)
	assert.False(t, svc.IsAvailable())
}

// TestSqrt verifies the custom sqrt function used internally
func TestSqrt(t *testing.T) {
	tests := []struct {
		name  string
		input float64
	}{
		{"sqrt of 4", 4.0},
		{"sqrt of 9", 9.0},
		{"sqrt of 2", 2.0},
		{"sqrt of 0", 0.0},
		{"sqrt of negative", -1.0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := sqrt(tt.input)
			if tt.input <= 0 {
				assert.Equal(t, 0.0, result)
			} else {
				assert.InDelta(t, math.Sqrt(tt.input), result, 0.0001)
			}
		})
	}
}
