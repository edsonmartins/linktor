using Linktor.Types;

namespace Linktor.Resources;

public class AIResource
{
    private readonly LinktorClient _client;

    public AgentsResource Agents { get; }
    public CompletionsResource Completions { get; }
    public EmbeddingsResource Embeddings { get; }

    public AIResource(LinktorClient client)
    {
        _client = client;
        Agents = new AgentsResource(client);
        Completions = new CompletionsResource(client);
        Embeddings = new EmbeddingsResource(client);
    }
}

public class AgentsResource
{
    private readonly LinktorClient _client;

    public AgentsResource(LinktorClient client) => _client = client;

    public Task<PaginatedResponse<Agent>> ListAsync(ListAgentsParams? parameters = null, CancellationToken ct = default)
    {
        var query = BuildQuery(parameters);
        return _client.GetAsync<PaginatedResponse<Agent>>($"/ai/agents{query}", ct);
    }

    public Task<Agent> GetAsync(string id, CancellationToken ct = default)
        => _client.GetAsync<Agent>($"/ai/agents/{id}", ct);

    public Task<Agent> CreateAsync(CreateAgentInput input, CancellationToken ct = default)
        => _client.PostAsync<Agent>("/ai/agents", input, ct);

    public Task<Agent> UpdateAsync(string id, UpdateAgentInput input, CancellationToken ct = default)
        => _client.PatchAsync<Agent>($"/ai/agents/{id}", input, ct);

    public Task DeleteAsync(string id, CancellationToken ct = default)
        => _client.DeleteAsync($"/ai/agents/{id}", ct);

    public Task<AgentInvokeResponse> InvokeAsync(string id, AgentInvokeInput input, CancellationToken ct = default)
        => _client.PostAsync<AgentInvokeResponse>($"/ai/agents/{id}/invoke", input, ct);

    private static string BuildQuery(ListAgentsParams? p)
    {
        if (p == null) return "";
        var parts = new List<string>();
        if (p.Limit.HasValue) parts.Add($"limit={p.Limit}");
        if (p.Offset.HasValue) parts.Add($"offset={p.Offset}");
        return parts.Count > 0 ? "?" + string.Join("&", parts) : "";
    }
}

public class CompletionsResource
{
    private readonly LinktorClient _client;

    public CompletionsResource(LinktorClient client) => _client = client;

    public Task<CompletionResponse> CreateAsync(CompletionInput input, CancellationToken ct = default)
        => _client.PostAsync<CompletionResponse>("/ai/completions", input, ct);

    public Task<ChatCompletionResponse> ChatAsync(ChatCompletionInput input, CancellationToken ct = default)
        => _client.PostAsync<ChatCompletionResponse>("/ai/completions/chat", input, ct);
}

public class EmbeddingsResource
{
    private readonly LinktorClient _client;

    public EmbeddingsResource(LinktorClient client) => _client = client;

    public Task<EmbeddingResponse> CreateAsync(EmbeddingInput input, CancellationToken ct = default)
        => _client.PostAsync<EmbeddingResponse>("/ai/embeddings", input, ct);

    public Task<SimilaritySearchResponse> SearchAsync(SimilaritySearchInput input, CancellationToken ct = default)
        => _client.PostAsync<SimilaritySearchResponse>("/ai/embeddings/search", input, ct);
}

public class ListAgentsParams
{
    public int? Limit { get; set; }
    public int? Offset { get; set; }
}

public class CreateAgentInput
{
    public string Name { get; set; } = string.Empty;
    public string? Description { get; set; }
    public string? SystemPrompt { get; set; }
    public string? Model { get; set; }
    public AgentConfig? Config { get; set; }
    public List<string>? KnowledgeBaseIds { get; set; }
    public List<AgentTool>? Tools { get; set; }
    public Dictionary<string, object>? Metadata { get; set; }
}

public class UpdateAgentInput
{
    public string? Name { get; set; }
    public string? Description { get; set; }
    public string? SystemPrompt { get; set; }
    public string? Model { get; set; }
    public AgentConfig? Config { get; set; }
    public List<string>? KnowledgeBaseIds { get; set; }
    public List<AgentTool>? Tools { get; set; }
    public Dictionary<string, object>? Metadata { get; set; }
}

public class AgentInvokeInput
{
    public string Message { get; set; } = string.Empty;
    public string? ConversationId { get; set; }
    public List<ChatMessage>? History { get; set; }
    public Dictionary<string, object>? Context { get; set; }
}

public class AgentInvokeResponse
{
    public string Response { get; set; } = string.Empty;
    public string? ConversationId { get; set; }
    public List<string>? Sources { get; set; }
    public Dictionary<string, object>? Metadata { get; set; }
}

public class CompletionInput
{
    public string Prompt { get; set; } = string.Empty;
    public string? Model { get; set; }
    public int? MaxTokens { get; set; }
    public double? Temperature { get; set; }
    public List<string>? StopSequences { get; set; }
}

public class CompletionResponse
{
    public string Text { get; set; } = string.Empty;
    public string Model { get; set; } = string.Empty;
    public TokenUsage? Usage { get; set; }
}

public class ChatCompletionInput
{
    public List<ChatMessage> Messages { get; set; } = new();
    public string? Model { get; set; }
    public int? MaxTokens { get; set; }
    public double? Temperature { get; set; }
    public List<string>? StopSequences { get; set; }
}

public class ChatCompletionResponse
{
    public ChatMessage Message { get; set; } = new();
    public string Model { get; set; } = string.Empty;
    public TokenUsage? Usage { get; set; }
}

public class EmbeddingInput
{
    public List<string> Texts { get; set; } = new();
    public string? Model { get; set; }
}

public class EmbeddingResponse
{
    public List<EmbeddingData> Embeddings { get; set; } = new();
    public string Model { get; set; } = string.Empty;
    public TokenUsage? Usage { get; set; }
}

public class EmbeddingData
{
    public int Index { get; set; }
    public List<float> Embedding { get; set; } = new();
}

public class SimilaritySearchInput
{
    public string Query { get; set; } = string.Empty;
    public string? KnowledgeBaseId { get; set; }
    public int? TopK { get; set; }
    public double? MinScore { get; set; }
}

public class SimilaritySearchResponse
{
    public List<SimilarityResult> Results { get; set; } = new();
}

public class SimilarityResult
{
    public string Id { get; set; } = string.Empty;
    public string Content { get; set; } = string.Empty;
    public double Score { get; set; }
    public Dictionary<string, object>? Metadata { get; set; }
}
