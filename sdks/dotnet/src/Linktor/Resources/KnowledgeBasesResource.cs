using Linktor.Types;

namespace Linktor.Resources;

public class KnowledgeBasesResource
{
    private readonly LinktorClient _client;

    public KnowledgeBasesResource(LinktorClient client) => _client = client;

    public Task<PaginatedResponse<KnowledgeBase>> ListAsync(ListKnowledgeBasesParams? parameters = null, CancellationToken ct = default)
    {
        var query = BuildQuery(parameters);
        return _client.GetAsync<PaginatedResponse<KnowledgeBase>>($"/knowledge-bases{query}", ct);
    }

    public Task<KnowledgeBase> GetAsync(string id, CancellationToken ct = default)
        => _client.GetAsync<KnowledgeBase>($"/knowledge-bases/{id}", ct);

    public Task<KnowledgeBase> CreateAsync(CreateKnowledgeBaseInput input, CancellationToken ct = default)
        => _client.PostAsync<KnowledgeBase>("/knowledge-bases", input, ct);

    public Task<KnowledgeBase> UpdateAsync(string id, UpdateKnowledgeBaseInput input, CancellationToken ct = default)
        => _client.PatchAsync<KnowledgeBase>($"/knowledge-bases/{id}", input, ct);

    public Task DeleteAsync(string id, CancellationToken ct = default)
        => _client.DeleteAsync($"/knowledge-bases/{id}", ct);

    // Document operations
    public Task<PaginatedResponse<Document>> ListDocumentsAsync(string knowledgeBaseId, ListDocumentsParams? parameters = null, CancellationToken ct = default)
    {
        var query = BuildDocQuery(parameters);
        return _client.GetAsync<PaginatedResponse<Document>>($"/knowledge-bases/{knowledgeBaseId}/documents{query}", ct);
    }

    public Task<Document> GetDocumentAsync(string knowledgeBaseId, string documentId, CancellationToken ct = default)
        => _client.GetAsync<Document>($"/knowledge-bases/{knowledgeBaseId}/documents/{documentId}", ct);

    public Task<Document> AddDocumentAsync(string knowledgeBaseId, AddDocumentInput input, CancellationToken ct = default)
        => _client.PostAsync<Document>($"/knowledge-bases/{knowledgeBaseId}/documents", input, ct);

    public Task<Document> UpdateDocumentAsync(string knowledgeBaseId, string documentId, UpdateDocumentInput input, CancellationToken ct = default)
        => _client.PatchAsync<Document>($"/knowledge-bases/{knowledgeBaseId}/documents/{documentId}", input, ct);

    public Task DeleteDocumentAsync(string knowledgeBaseId, string documentId, CancellationToken ct = default)
        => _client.DeleteAsync($"/knowledge-bases/{knowledgeBaseId}/documents/{documentId}", ct);

    public Task<Document> ReprocessDocumentAsync(string knowledgeBaseId, string documentId, CancellationToken ct = default)
        => _client.PostAsync<Document>($"/knowledge-bases/{knowledgeBaseId}/documents/{documentId}/reprocess", new { }, ct);

    // Query operations
    public Task<QueryResponse> QueryAsync(string knowledgeBaseId, QueryInput input, CancellationToken ct = default)
        => _client.PostAsync<QueryResponse>($"/knowledge-bases/{knowledgeBaseId}/query", input, ct);

    private static string BuildQuery(ListKnowledgeBasesParams? p)
    {
        if (p == null) return "";
        var parts = new List<string>();
        if (p.Limit.HasValue) parts.Add($"limit={p.Limit}");
        if (p.Offset.HasValue) parts.Add($"offset={p.Offset}");
        return parts.Count > 0 ? "?" + string.Join("&", parts) : "";
    }

    private static string BuildDocQuery(ListDocumentsParams? p)
    {
        if (p == null) return "";
        var parts = new List<string>();
        if (p.Limit.HasValue) parts.Add($"limit={p.Limit}");
        if (p.Offset.HasValue) parts.Add($"offset={p.Offset}");
        if (!string.IsNullOrEmpty(p.Status)) parts.Add($"status={Uri.EscapeDataString(p.Status)}");
        return parts.Count > 0 ? "?" + string.Join("&", parts) : "";
    }
}

public class ListKnowledgeBasesParams
{
    public int? Limit { get; set; }
    public int? Offset { get; set; }
}

public class CreateKnowledgeBaseInput
{
    public string Name { get; set; } = string.Empty;
    public string? Description { get; set; }
    public string? EmbeddingModel { get; set; }
    public KnowledgeBaseConfig? Config { get; set; }
    public Dictionary<string, object>? Metadata { get; set; }
}

public class UpdateKnowledgeBaseInput
{
    public string? Name { get; set; }
    public string? Description { get; set; }
    public KnowledgeBaseConfig? Config { get; set; }
    public Dictionary<string, object>? Metadata { get; set; }
}

public class ListDocumentsParams
{
    public int? Limit { get; set; }
    public int? Offset { get; set; }
    public string? Status { get; set; }
}

public class AddDocumentInput
{
    public string? Title { get; set; }
    public string? Content { get; set; }
    public string? Url { get; set; }
    public string? FileType { get; set; }
    public string? Base64Content { get; set; }
    public Dictionary<string, object>? Metadata { get; set; }
}

public class UpdateDocumentInput
{
    public string? Title { get; set; }
    public string? Content { get; set; }
    public Dictionary<string, object>? Metadata { get; set; }
}

public class QueryInput
{
    public string Query { get; set; } = string.Empty;
    public int? TopK { get; set; }
    public double? MinScore { get; set; }
    public Dictionary<string, object>? Filter { get; set; }
    public bool? IncludeContent { get; set; }
}

public class QueryResponse
{
    public List<QueryResult> Results { get; set; } = new();
    public TokenUsage? Usage { get; set; }
}

public class QueryResult
{
    public string DocumentId { get; set; } = string.Empty;
    public string ChunkId { get; set; } = string.Empty;
    public string Content { get; set; } = string.Empty;
    public double Score { get; set; }
    public Dictionary<string, object>? Metadata { get; set; }
}
