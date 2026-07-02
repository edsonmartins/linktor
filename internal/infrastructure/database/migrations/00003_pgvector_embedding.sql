-- +goose Up
-- KnowledgeItemRepository.SearchByEmbedding uses pgvector's cosine-distance
-- operator (<=>), which only exists for the `vector` type. The baseline stored
-- `embedding` as TEXT, so RAG search failed with "operator does not exist".
-- The runtime image is pgvector/pgvector:pg16, so the extension is available.
--
-- The column is left unbounded (`vector`, no fixed dimension) because the
-- embedding dimension is configurable (1536 for ada-002, 768 for others).
-- Unbounded vectors support <=> via sequential scan; add a typed column + ANN
-- index (ivfflat/hnsw) later once a single dimension is pinned per deployment.
CREATE EXTENSION IF NOT EXISTS vector;

ALTER TABLE knowledge_items
    ALTER COLUMN embedding TYPE vector
    USING (NULLIF(embedding, '')::vector);

-- +goose Down
ALTER TABLE knowledge_items
    ALTER COLUMN embedding TYPE text
    USING (embedding::text);
