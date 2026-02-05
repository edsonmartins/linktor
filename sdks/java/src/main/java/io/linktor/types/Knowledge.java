package io.linktor.types;

import com.google.gson.annotations.SerializedName;
import java.time.Instant;
import java.util.List;
import java.util.Map;

public class Knowledge {

    public enum KnowledgeBaseStatus {
        @SerializedName("active") ACTIVE,
        @SerializedName("processing") PROCESSING,
        @SerializedName("error") ERROR,
        @SerializedName("empty") EMPTY
    }

    public enum DocumentStatus {
        @SerializedName("pending") PENDING,
        @SerializedName("processing") PROCESSING,
        @SerializedName("completed") COMPLETED,
        @SerializedName("failed") FAILED
    }

    public static class KnowledgeBase {
        private String id;
        private String tenantId;
        private String name;
        private String description;
        private KnowledgeBaseStatus status;
        private String embeddingModel;
        private int chunkSize;
        private int chunkOverlap;
        private int documentCount;
        private int totalChunks;
        private Map<String, Object> metadata;
        private Instant createdAt;
        private Instant updatedAt;

        public String getId() { return id; }
        public void setId(String id) { this.id = id; }

        public String getTenantId() { return tenantId; }
        public void setTenantId(String tenantId) { this.tenantId = tenantId; }

        public String getName() { return name; }
        public void setName(String name) { this.name = name; }

        public String getDescription() { return description; }
        public void setDescription(String description) { this.description = description; }

        public KnowledgeBaseStatus getStatus() { return status; }
        public void setStatus(KnowledgeBaseStatus status) { this.status = status; }

        public String getEmbeddingModel() { return embeddingModel; }
        public void setEmbeddingModel(String embeddingModel) { this.embeddingModel = embeddingModel; }

        public int getChunkSize() { return chunkSize; }
        public void setChunkSize(int chunkSize) { this.chunkSize = chunkSize; }

        public int getChunkOverlap() { return chunkOverlap; }
        public void setChunkOverlap(int chunkOverlap) { this.chunkOverlap = chunkOverlap; }

        public int getDocumentCount() { return documentCount; }
        public void setDocumentCount(int documentCount) { this.documentCount = documentCount; }

        public int getTotalChunks() { return totalChunks; }
        public void setTotalChunks(int totalChunks) { this.totalChunks = totalChunks; }

        public Map<String, Object> getMetadata() { return metadata; }
        public void setMetadata(Map<String, Object> metadata) { this.metadata = metadata; }

        public Instant getCreatedAt() { return createdAt; }
        public void setCreatedAt(Instant createdAt) { this.createdAt = createdAt; }

        public Instant getUpdatedAt() { return updatedAt; }
        public void setUpdatedAt(Instant updatedAt) { this.updatedAt = updatedAt; }
    }

    public static class Document {
        private String id;
        private String knowledgeBaseId;
        private String name;
        private String type;
        private String sourceUrl;
        private DocumentStatus status;
        private long size;
        private int chunkCount;
        private Map<String, Object> metadata;
        private String error;
        private Instant createdAt;
        private Instant updatedAt;

        public String getId() { return id; }
        public void setId(String id) { this.id = id; }

        public String getKnowledgeBaseId() { return knowledgeBaseId; }
        public void setKnowledgeBaseId(String knowledgeBaseId) { this.knowledgeBaseId = knowledgeBaseId; }

        public String getName() { return name; }
        public void setName(String name) { this.name = name; }

        public String getType() { return type; }
        public void setType(String type) { this.type = type; }

        public String getSourceUrl() { return sourceUrl; }
        public void setSourceUrl(String sourceUrl) { this.sourceUrl = sourceUrl; }

        public DocumentStatus getStatus() { return status; }
        public void setStatus(DocumentStatus status) { this.status = status; }

        public long getSize() { return size; }
        public void setSize(long size) { this.size = size; }

        public int getChunkCount() { return chunkCount; }
        public void setChunkCount(int chunkCount) { this.chunkCount = chunkCount; }

        public Map<String, Object> getMetadata() { return metadata; }
        public void setMetadata(Map<String, Object> metadata) { this.metadata = metadata; }

        public String getError() { return error; }
        public void setError(String error) { this.error = error; }

        public Instant getCreatedAt() { return createdAt; }
        public void setCreatedAt(Instant createdAt) { this.createdAt = createdAt; }

        public Instant getUpdatedAt() { return updatedAt; }
        public void setUpdatedAt(Instant updatedAt) { this.updatedAt = updatedAt; }
    }

    public static class ScoredChunk {
        private String id;
        private String documentId;
        private String content;
        private int chunkIndex;
        private int tokenCount;
        private double score;
        private Map<String, Object> metadata;
        private Document document;

        public String getId() { return id; }
        public void setId(String id) { this.id = id; }

        public String getDocumentId() { return documentId; }
        public void setDocumentId(String documentId) { this.documentId = documentId; }

        public String getContent() { return content; }
        public void setContent(String content) { this.content = content; }

        public int getChunkIndex() { return chunkIndex; }
        public void setChunkIndex(int chunkIndex) { this.chunkIndex = chunkIndex; }

        public int getTokenCount() { return tokenCount; }
        public void setTokenCount(int tokenCount) { this.tokenCount = tokenCount; }

        public double getScore() { return score; }
        public void setScore(double score) { this.score = score; }

        public Map<String, Object> getMetadata() { return metadata; }
        public void setMetadata(Map<String, Object> metadata) { this.metadata = metadata; }

        public Document getDocument() { return document; }
        public void setDocument(Document document) { this.document = document; }
    }

    public static class QueryResult {
        private List<ScoredChunk> chunks;
        private String query;
        private String model;

        public List<ScoredChunk> getChunks() { return chunks; }
        public void setChunks(List<ScoredChunk> chunks) { this.chunks = chunks; }

        public String getQuery() { return query; }
        public void setQuery(String query) { this.query = query; }

        public String getModel() { return model; }
        public void setModel(String model) { this.model = model; }
    }

    public static class CreateKnowledgeBaseInput {
        private String name;
        private String description;
        private String embeddingModel;
        private Integer chunkSize;
        private Integer chunkOverlap;
        private Map<String, Object> metadata;

        public String getName() { return name; }
        public void setName(String name) { this.name = name; }

        public String getDescription() { return description; }
        public void setDescription(String description) { this.description = description; }

        public String getEmbeddingModel() { return embeddingModel; }
        public void setEmbeddingModel(String embeddingModel) { this.embeddingModel = embeddingModel; }

        public Integer getChunkSize() { return chunkSize; }
        public void setChunkSize(Integer chunkSize) { this.chunkSize = chunkSize; }

        public Integer getChunkOverlap() { return chunkOverlap; }
        public void setChunkOverlap(Integer chunkOverlap) { this.chunkOverlap = chunkOverlap; }

        public Map<String, Object> getMetadata() { return metadata; }
        public void setMetadata(Map<String, Object> metadata) { this.metadata = metadata; }

        public static Builder builder() { return new Builder(); }

        public static class Builder {
            private final CreateKnowledgeBaseInput input = new CreateKnowledgeBaseInput();

            public Builder name(String name) { input.name = name; return this; }
            public Builder description(String description) { input.description = description; return this; }
            public Builder embeddingModel(String model) { input.embeddingModel = model; return this; }
            public Builder chunkSize(Integer size) { input.chunkSize = size; return this; }
            public Builder chunkOverlap(Integer overlap) { input.chunkOverlap = overlap; return this; }
            public Builder metadata(Map<String, Object> metadata) { input.metadata = metadata; return this; }
            public CreateKnowledgeBaseInput build() { return input; }
        }
    }

    public static class UpdateKnowledgeBaseInput {
        private String name;
        private String description;
        private Map<String, Object> metadata;

        public String getName() { return name; }
        public void setName(String name) { this.name = name; }

        public String getDescription() { return description; }
        public void setDescription(String description) { this.description = description; }

        public Map<String, Object> getMetadata() { return metadata; }
        public void setMetadata(Map<String, Object> metadata) { this.metadata = metadata; }

        public static Builder builder() { return new Builder(); }

        public static class Builder {
            private final UpdateKnowledgeBaseInput input = new UpdateKnowledgeBaseInput();

            public Builder name(String name) { input.name = name; return this; }
            public Builder description(String description) { input.description = description; return this; }
            public Builder metadata(Map<String, Object> metadata) { input.metadata = metadata; return this; }
            public UpdateKnowledgeBaseInput build() { return input; }
        }
    }

    public static class AddDocumentInput {
        private String name;
        private String content;
        private String sourceUrl;
        private Map<String, Object> metadata;

        public String getName() { return name; }
        public void setName(String name) { this.name = name; }

        public String getContent() { return content; }
        public void setContent(String content) { this.content = content; }

        public String getSourceUrl() { return sourceUrl; }
        public void setSourceUrl(String sourceUrl) { this.sourceUrl = sourceUrl; }

        public Map<String, Object> getMetadata() { return metadata; }
        public void setMetadata(Map<String, Object> metadata) { this.metadata = metadata; }

        public static Builder builder() { return new Builder(); }

        public static class Builder {
            private final AddDocumentInput input = new AddDocumentInput();

            public Builder name(String name) { input.name = name; return this; }
            public Builder content(String content) { input.content = content; return this; }
            public Builder sourceUrl(String url) { input.sourceUrl = url; return this; }
            public Builder metadata(Map<String, Object> metadata) { input.metadata = metadata; return this; }
            public AddDocumentInput build() { return input; }
        }
    }

    public static class QueryKnowledgeBaseInput {
        private String query;
        private Integer topK;
        private Double minScore;
        private Map<String, Object> filter;

        public String getQuery() { return query; }
        public void setQuery(String query) { this.query = query; }

        public Integer getTopK() { return topK; }
        public void setTopK(Integer topK) { this.topK = topK; }

        public Double getMinScore() { return minScore; }
        public void setMinScore(Double minScore) { this.minScore = minScore; }

        public Map<String, Object> getFilter() { return filter; }
        public void setFilter(Map<String, Object> filter) { this.filter = filter; }

        public static Builder builder() { return new Builder(); }

        public static class Builder {
            private final QueryKnowledgeBaseInput input = new QueryKnowledgeBaseInput();

            public Builder query(String query) { input.query = query; return this; }
            public Builder topK(Integer topK) { input.topK = topK; return this; }
            public Builder minScore(Double minScore) { input.minScore = minScore; return this; }
            public Builder filter(Map<String, Object> filter) { input.filter = filter; return this; }
            public QueryKnowledgeBaseInput build() { return input; }
        }
    }
}
