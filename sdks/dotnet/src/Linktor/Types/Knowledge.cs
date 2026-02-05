using System.Text.Json.Serialization;

namespace Linktor.Types;

[JsonConverter(typeof(JsonStringEnumConverter))]
public enum KnowledgeBaseStatus { Active, Processing, Error, Empty }

[JsonConverter(typeof(JsonStringEnumConverter))]
public enum DocumentStatus { Pending, Processing, Completed, Failed }

public class KnowledgeBase
{
    [JsonPropertyName("id")] public string Id { get; set; } = string.Empty;
    [JsonPropertyName("tenantId")] public string TenantId { get; set; } = string.Empty;
    [JsonPropertyName("name")] public string Name { get; set; } = string.Empty;
    [JsonPropertyName("description")] public string? Description { get; set; }
    [JsonPropertyName("status")] public KnowledgeBaseStatus Status { get; set; }
    [JsonPropertyName("embeddingModel")] public string EmbeddingModel { get; set; } = string.Empty;
    [JsonPropertyName("chunkSize")] public int ChunkSize { get; set; }
    [JsonPropertyName("chunkOverlap")] public int ChunkOverlap { get; set; }
    [JsonPropertyName("documentCount")] public int DocumentCount { get; set; }
    [JsonPropertyName("totalChunks")] public int TotalChunks { get; set; }
    [JsonPropertyName("metadata")] public Dictionary<string, object>? Metadata { get; set; }
    [JsonPropertyName("createdAt")] public DateTime CreatedAt { get; set; }
    [JsonPropertyName("updatedAt")] public DateTime UpdatedAt { get; set; }
}

public class Document
{
    [JsonPropertyName("id")] public string Id { get; set; } = string.Empty;
    [JsonPropertyName("knowledgeBaseId")] public string KnowledgeBaseId { get; set; } = string.Empty;
    [JsonPropertyName("name")] public string Name { get; set; } = string.Empty;
    [JsonPropertyName("type")] public string Type { get; set; } = string.Empty;
    [JsonPropertyName("sourceUrl")] public string? SourceUrl { get; set; }
    [JsonPropertyName("status")] public DocumentStatus Status { get; set; }
    [JsonPropertyName("size")] public long Size { get; set; }
    [JsonPropertyName("chunkCount")] public int ChunkCount { get; set; }
    [JsonPropertyName("metadata")] public Dictionary<string, object>? Metadata { get; set; }
    [JsonPropertyName("error")] public string? Error { get; set; }
    [JsonPropertyName("createdAt")] public DateTime CreatedAt { get; set; }
    [JsonPropertyName("updatedAt")] public DateTime UpdatedAt { get; set; }
}

public class ScoredChunk
{
    [JsonPropertyName("id")] public string Id { get; set; } = string.Empty;
    [JsonPropertyName("documentId")] public string DocumentId { get; set; } = string.Empty;
    [JsonPropertyName("content")] public string Content { get; set; } = string.Empty;
    [JsonPropertyName("chunkIndex")] public int ChunkIndex { get; set; }
    [JsonPropertyName("tokenCount")] public int TokenCount { get; set; }
    [JsonPropertyName("score")] public double Score { get; set; }
    [JsonPropertyName("metadata")] public Dictionary<string, object>? Metadata { get; set; }
    [JsonPropertyName("document")] public Document? Document { get; set; }
}

public class QueryResult
{
    [JsonPropertyName("chunks")] public List<ScoredChunk> Chunks { get; set; } = new();
    [JsonPropertyName("query")] public string Query { get; set; } = string.Empty;
    [JsonPropertyName("model")] public string Model { get; set; } = string.Empty;
}

public class CreateKnowledgeBaseInput
{
    [JsonPropertyName("name")] public string Name { get; set; } = string.Empty;
    [JsonPropertyName("description")] public string? Description { get; set; }
    [JsonPropertyName("embeddingModel")] public string? EmbeddingModel { get; set; }
    [JsonPropertyName("chunkSize")] public int? ChunkSize { get; set; }
    [JsonPropertyName("chunkOverlap")] public int? ChunkOverlap { get; set; }
    [JsonPropertyName("metadata")] public Dictionary<string, object>? Metadata { get; set; }
}

public class AddDocumentInput
{
    [JsonPropertyName("name")] public string Name { get; set; } = string.Empty;
    [JsonPropertyName("content")] public string? Content { get; set; }
    [JsonPropertyName("sourceUrl")] public string? SourceUrl { get; set; }
    [JsonPropertyName("metadata")] public Dictionary<string, object>? Metadata { get; set; }
}

public class QueryKnowledgeBaseInput
{
    [JsonPropertyName("query")] public string Query { get; set; } = string.Empty;
    [JsonPropertyName("topK")] public int? TopK { get; set; }
    [JsonPropertyName("minScore")] public double? MinScore { get; set; }
    [JsonPropertyName("filter")] public Dictionary<string, object>? Filter { get; set; }
}
