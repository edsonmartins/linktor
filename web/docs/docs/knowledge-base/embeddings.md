---
sidebar_position: 2
title: Embeddings & Semantic Search
---

# Embeddings & Semantic Search

This guide explains how embeddings power the Knowledge Base's semantic search capabilities, enabling your bots to find relevant content even when customers don't use exact keywords.

## What are Embeddings?

**Embeddings** are numerical representations (vectors) of text that capture semantic meaning. Similar concepts have similar vectors, enabling semantic search that understands meaning rather than just matching keywords.

### Traditional Search vs. Semantic Search

| Approach | Query: "cancel subscription" |
|----------|------------------------------|
| **Keyword Search** | Only finds docs containing "cancel" and "subscription" |
| **Semantic Search** | Also finds "end membership", "stop billing", "unsubscribe" |

### How Embeddings Work

```
Text Input                     Embedding Vector
─────────────────────────────────────────────────────────────
"How to reset password"   →   [0.12, -0.45, 0.78, ..., 0.33]
"Change my login"         →   [0.11, -0.42, 0.75, ..., 0.31]  ← Similar vectors
"What's the weather"      →   [-0.56, 0.23, -0.12, ..., 0.89] ← Different vector
```

Texts with similar meanings produce vectors that are close together in the embedding space.

## Embedding Models

### Supported Models

Linktor supports multiple embedding models:

| Model | Provider | Dimensions | Performance | Cost |
|-------|----------|------------|-------------|------|
| `text-embedding-3-small` | OpenAI | 1536 | Good | Low |
| `text-embedding-3-large` | OpenAI | 3072 | Excellent | Medium |
| `text-embedding-ada-002` | OpenAI | 1536 | Good | Low |
| `voyage-large-2` | Voyage AI | 1024 | Excellent | Medium |
| `voyage-code-2` | Voyage AI | 1024 | Best for code | Medium |
| `embed-english-v3.0` | Cohere | 1024 | Good | Low |
| `embed-multilingual-v3.0` | Cohere | 1024 | Best multilingual | Medium |

### Configuring Embedding Model

```typescript
await client.knowledgeBases.create({
  name: 'Product Docs',
  settings: {
    embeddingModel: 'text-embedding-3-large',
    embeddingProvider: 'openai',
    embeddingApiKey: process.env.OPENAI_API_KEY  // Optional: uses default if not set
  }
})
```

### Self-Hosted Embeddings

Use your own embedding models:

```typescript
{
  settings: {
    embeddingProvider: 'custom',
    embeddingEndpoint: 'http://localhost:8081/embeddings',
    embeddingModel: 'all-MiniLM-L6-v2',
    embeddingDimensions: 384
  }
}
```

Compatible with:
- Hugging Face Inference Endpoints
- Sentence Transformers served via FastAPI
- Ollama embeddings
- Any OpenAI-compatible API

## Document Chunking

Large documents must be split into smaller chunks for effective retrieval. Chunking strategy significantly impacts search quality.

### Why Chunking Matters

```
Original Document (5000 tokens)
├── Chunk 1 (500 tokens) → Embedding 1
├── Chunk 2 (500 tokens) → Embedding 2
├── Chunk 3 (500 tokens) → Embedding 3
├── ...
└── Chunk 10 (500 tokens) → Embedding 10
```

Each chunk gets its own embedding, allowing precise retrieval of relevant sections.

### Chunk Size

```typescript
{
  settings: {
    chunkSize: 500,      // Target tokens per chunk
    chunkSizeType: 'tokens'  // 'tokens' | 'characters'
  }
}
```

| Chunk Size | Pros | Cons |
|------------|------|------|
| **Small (200-300)** | Precise retrieval | May lose context |
| **Medium (400-600)** | Balanced | Good default |
| **Large (800-1000)** | More context | Less precise matching |

### Chunk Overlap

Overlap ensures context isn't lost at chunk boundaries:

```typescript
{
  settings: {
    chunkSize: 500,
    chunkOverlap: 50  // Tokens shared between adjacent chunks
  }
}
```

**Without overlap:**
```
[Chunk 1: "...payment methods we accept."]
[Chunk 2: "Credit cards include Visa..."]
```

**With overlap:**
```
[Chunk 1: "...payment methods we accept. Credit cards"]
[Chunk 2: "accept. Credit cards include Visa..."]
```

### Chunking Strategies

#### Fixed Size (Default)

Split at fixed token/character intervals:

```typescript
{
  settings: {
    chunkingStrategy: 'fixed',
    chunkSize: 500,
    chunkOverlap: 50
  }
}
```

#### Semantic Chunking

Split at natural boundaries (paragraphs, sections):

```typescript
{
  settings: {
    chunkingStrategy: 'semantic',
    maxChunkSize: 1000,
    splitOn: ['paragraph', 'sentence'],
    preserveHeaders: true
  }
}
```

#### Recursive Chunking

Hierarchical splitting for better structure preservation:

```typescript
{
  settings: {
    chunkingStrategy: 'recursive',
    separators: ['\n\n', '\n', '. ', ' '],
    chunkSize: 500,
    chunkOverlap: 50
  }
}
```

#### Document-Aware Chunking

Respects document structure (headings, lists, tables):

```typescript
{
  settings: {
    chunkingStrategy: 'document_aware',
    preserveStructure: true,
    keepListsTogether: true,
    keepTablesTogether: true,
    maxChunkSize: 1000
  }
}
```

## Vector Storage

### How Vectors are Stored

```
┌─────────────────────────────────────────────────────────────────┐
│                        Vector Database                          │
├─────────────────────────────────────────────────────────────────┤
│  Document ID  │  Chunk ID  │  Vector [1536 dims]  │  Metadata   │
├───────────────┼────────────┼──────────────────────┼─────────────┤
│  doc_001      │  chunk_001 │  [0.12, -0.45, ...]  │  {page: 1}  │
│  doc_001      │  chunk_002 │  [0.15, -0.42, ...]  │  {page: 1}  │
│  doc_002      │  chunk_001 │  [-0.33, 0.21, ...]  │  {page: 1}  │
└─────────────────────────────────────────────────────────────────┘
```

### Indexing Options

```typescript
{
  settings: {
    vectorIndex: {
      type: 'hnsw',           // 'hnsw' | 'ivfflat' | 'flat'
      m: 16,                  // HNSW: connections per node
      efConstruction: 200,    // HNSW: build-time quality
      efSearch: 100,          // HNSW: search-time quality
      lists: 100              // IVFFlat: number of lists
    }
  }
}
```

| Index Type | Speed | Accuracy | Memory | Use Case |
|------------|-------|----------|--------|----------|
| `flat` | Slow | 100% | High | Small datasets (&lt;10k) |
| `ivfflat` | Fast | 95-99% | Medium | Medium datasets |
| `hnsw` | Very Fast | 98-99% | Medium | Large datasets (default) |

## Semantic Search Process

### Query Flow

```
Customer Query: "How do I get a refund?"
         │
         ▼
┌─────────────────────────┐
│  1. Generate Embedding  │
│  Query → [0.08, -0.52...│
└───────────┬─────────────┘
            │
            ▼
┌─────────────────────────┐
│  2. Vector Similarity   │
│  Find nearest neighbors │
└───────────┬─────────────┘
            │
            ▼
┌─────────────────────────┐
│  3. Re-ranking          │
│  Order by relevance     │
└───────────┬─────────────┘
            │
            ▼
┌─────────────────────────┐
│  4. Return Top Chunks   │
│  Include metadata       │
└─────────────────────────┘
```

### Similarity Metrics

```typescript
{
  settings: {
    similarityMetric: 'cosine'  // 'cosine' | 'euclidean' | 'dotProduct'
  }
}
```

| Metric | Formula | Range | Notes |
|--------|---------|-------|-------|
| `cosine` | cos(A,B) | -1 to 1 | Best for normalized vectors (default) |
| `euclidean` | ||A-B|| | 0 to inf | Better for magnitude-sensitive data |
| `dotProduct` | A · B | -inf to inf | Fast, good for normalized vectors |

### Search Parameters

```typescript
const results = await client.knowledgeBases.search(kbId, {
  query: 'refund policy',

  // Retrieval settings
  limit: 10,                    // Max results to return
  minScore: 0.7,                // Minimum similarity score

  // Filtering
  filters: {
    category: 'policies',
    language: 'en'
  },

  // Re-ranking
  rerank: true,
  rerankModel: 'rerank-english-v2.0',

  // Hybrid search
  hybridSearch: true,
  keywordWeight: 0.3,           // Weight for keyword matching
  semanticWeight: 0.7           // Weight for semantic matching
})
```

## Hybrid Search

Combine semantic search with traditional keyword search for better results.

### How Hybrid Search Works

```
Query: "order #12345 status"
         │
    ┌────┴────┐
    │         │
    ▼         ▼
┌────────┐ ┌────────┐
│Semantic│ │Keyword │
│Search  │ │Search  │
└────┬───┘ └───┬────┘
     │         │
     ▼         ▼
┌────────────────────┐
│  Reciprocal Rank   │
│  Fusion (RRF)      │
└─────────┬──────────┘
          │
          ▼
   Combined Results
```

### Configuration

```typescript
{
  settings: {
    searchMode: 'hybrid',
    hybridConfig: {
      semanticWeight: 0.7,
      keywordWeight: 0.3,
      fusionMethod: 'rrf',        // 'rrf' | 'weighted_sum'
      keywordMatchType: 'bm25',   // 'bm25' | 'tfidf' | 'exact'
      keywordBoost: {
        title: 2.0,               // Boost title matches
        headings: 1.5             // Boost heading matches
      }
    }
  }
}
```

### When to Use Hybrid Search

| Scenario | Recommendation |
|----------|----------------|
| Technical documentation | Hybrid (terms matter) |
| Product names/SKUs | Hybrid with keyword boost |
| General Q&A | Semantic only usually fine |
| Code documentation | Hybrid (exact terms important) |
| Multi-language content | Semantic (handles variations) |

## Re-ranking

Re-ranking improves result quality by using a more sophisticated model to reorder initial results.

### How Re-ranking Works

```
Initial Semantic Search (fast, approximate)
         │
         ▼
Top 20 candidate results
         │
         ▼
┌─────────────────────────┐
│  Re-ranking Model       │
│  (cross-encoder)        │
│  Scores query+doc pairs │
└───────────┬─────────────┘
            │
            ▼
  Final Top 5 results (more accurate)
```

### Configuration

```typescript
{
  settings: {
    reranking: {
      enabled: true,
      model: 'rerank-english-v2.0',   // Cohere reranker
      topK: 20,                        // Candidates for reranking
      returnTopN: 5                    // Final results
    }
  }
}
```

### Supported Re-ranking Models

| Model | Provider | Languages | Quality |
|-------|----------|-----------|---------|
| `rerank-english-v2.0` | Cohere | English | Excellent |
| `rerank-multilingual-v2.0` | Cohere | 100+ languages | Excellent |
| `cross-encoder/ms-marco-MiniLM` | HuggingFace | English | Good |

## Metadata and Filtering

### Adding Metadata

```typescript
await client.knowledgeBases.uploadDocument(kbId, {
  file: './guide.pdf',
  metadata: {
    title: 'User Guide',
    category: 'documentation',
    product: 'app-v2',
    language: 'en',
    version: '2.1.0',
    author: 'docs-team',
    lastUpdated: '2024-01-15',
    audience: ['customers', 'partners'],
    tags: ['getting-started', 'configuration']
  }
})
```

### Filtering Searches

```typescript
const results = await client.knowledgeBases.search(kbId, {
  query: 'installation steps',
  filters: {
    // Exact match
    category: 'documentation',

    // Multiple values (OR)
    product: { in: ['app-v2', 'app-v3'] },

    // Range
    version: { gte: '2.0.0' },
    lastUpdated: { gte: '2024-01-01' },

    // Array contains
    audience: { contains: 'customers' },

    // Negation
    status: { ne: 'deprecated' },

    // Exists check
    reviewedBy: { exists: true }
  }
})
```

### Metadata Boosting

Boost results based on metadata:

```typescript
{
  settings: {
    metadataBoosts: {
      'category=faq': 1.5,         // Boost FAQ content
      'verified=true': 1.3,        // Boost verified content
      'lastUpdated>2024-01-01': 1.2 // Boost recent content
    }
  }
}
```

## Performance Optimization

### Indexing Performance

```typescript
// Batch processing for large imports
await client.knowledgeBases.batchUpload(kbId, {
  files: fileList,
  batchSize: 100,          // Process 100 files at a time
  parallelEmbeddings: 10,  // Concurrent embedding requests
  skipDuplicates: true     // Skip already indexed files
})
```

### Search Performance

```typescript
{
  settings: {
    // Caching
    queryCache: {
      enabled: true,
      ttl: 3600,           // 1 hour cache
      maxSize: 10000       // Max cached queries
    },

    // Pre-filtering
    preFilterEnabled: true,  // Apply filters before vector search

    // Result caching
    resultCache: {
      enabled: true,
      ttl: 300              // 5 minute cache
    }
  }
}
```

### Monitoring

```typescript
const metrics = await client.knowledgeBases.getMetrics(kbId, {
  period: '24h'
})

console.log(metrics.searchLatencyP50)    // Median search time
console.log(metrics.searchLatencyP99)    // 99th percentile
console.log(metrics.embeddingLatency)    // Embedding generation time
console.log(metrics.indexSize)           // Vector index size
console.log(metrics.cacheHitRate)        // Query cache effectiveness
```

## Troubleshooting

### Low Relevance Scores

| Cause | Solution |
|-------|----------|
| Poor chunking | Adjust chunk size/overlap |
| Wrong embedding model | Try a different model |
| Missing context | Enable hybrid search |
| Outdated content | Update documents |

### Slow Searches

| Cause | Solution |
|-------|----------|
| Large index | Enable HNSW indexing |
| Complex filters | Use pre-filtering |
| No caching | Enable query cache |
| Too many results | Reduce limit, increase minScore |

### Missing Results

| Cause | Solution |
|-------|----------|
| Score threshold too high | Lower minScore |
| Content not indexed | Check indexing status |
| Wrong filters | Verify metadata values |
| Language mismatch | Use multilingual model |

## Best Practices

1. **Choose the right embedding model**: Match model to your content type and languages

2. **Optimize chunk size**: Test different sizes with your actual queries

3. **Use meaningful metadata**: Enable powerful filtering and boosting

4. **Enable hybrid search for technical content**: Exact terms often matter

5. **Monitor and iterate**: Use analytics to improve over time

6. **Keep content fresh**: Outdated content hurts result quality

7. **Test with real queries**: Use actual customer questions for validation

## Next Steps

- [Knowledge Base Overview](/knowledge-base/overview) - Getting started guide
- [Bot Configuration](/bots/configuration) - Connect to bots
- [Testing Bots](/bots/testing) - Test knowledge retrieval
