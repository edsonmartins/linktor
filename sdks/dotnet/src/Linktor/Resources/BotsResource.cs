using Linktor.Types;

namespace Linktor.Resources;

public class BotsResource
{
    private readonly LinktorClient _client;

    public BotsResource(LinktorClient client) => _client = client;

    public Task<PaginatedResponse<Bot>> ListAsync(ListBotsParams? parameters = null, CancellationToken ct = default)
    {
        var query = BuildQuery(parameters);
        return _client.GetAsync<PaginatedResponse<Bot>>($"/bots{query}", ct);
    }

    public Task<Bot> GetAsync(string id, CancellationToken ct = default)
        => _client.GetAsync<Bot>($"/bots/{id}", ct);

    public Task<Bot> CreateAsync(CreateBotInput input, CancellationToken ct = default)
        => _client.PostAsync<Bot>("/bots", input, ct);

    public Task<Bot> UpdateAsync(string id, UpdateBotInput input, CancellationToken ct = default)
        => _client.PatchAsync<Bot>($"/bots/{id}", input, ct);

    public Task DeleteAsync(string id, CancellationToken ct = default)
        => _client.DeleteAsync($"/bots/{id}", ct);

    public Task<Bot> StartAsync(string id, CancellationToken ct = default)
        => _client.PostAsync<Bot>($"/bots/{id}/start", new { }, ct);

    public Task<Bot> StopAsync(string id, CancellationToken ct = default)
        => _client.PostAsync<Bot>($"/bots/{id}/stop", new { }, ct);

    public Task<BotStatus> GetStatusAsync(string id, CancellationToken ct = default)
        => _client.GetAsync<BotStatus>($"/bots/{id}/status", ct);

    private static string BuildQuery(ListBotsParams? p)
    {
        if (p == null) return "";
        var parts = new List<string>();
        if (p.Limit.HasValue) parts.Add($"limit={p.Limit}");
        if (p.Offset.HasValue) parts.Add($"offset={p.Offset}");
        if (!string.IsNullOrEmpty(p.Status)) parts.Add($"status={Uri.EscapeDataString(p.Status)}");
        return parts.Count > 0 ? "?" + string.Join("&", parts) : "";
    }
}

public class ListBotsParams
{
    public int? Limit { get; set; }
    public int? Offset { get; set; }
    public string? Status { get; set; }
}

public class CreateBotInput
{
    public string Name { get; set; } = string.Empty;
    public string? Description { get; set; }
    public string? AgentId { get; set; }
    public string? FlowId { get; set; }
    public List<string>? ChannelIds { get; set; }
    public BotConfig? Config { get; set; }
    public bool Enabled { get; set; } = true;
    public Dictionary<string, object>? Metadata { get; set; }
}

public class UpdateBotInput
{
    public string? Name { get; set; }
    public string? Description { get; set; }
    public string? AgentId { get; set; }
    public string? FlowId { get; set; }
    public List<string>? ChannelIds { get; set; }
    public BotConfig? Config { get; set; }
    public bool? Enabled { get; set; }
    public Dictionary<string, object>? Metadata { get; set; }
}

public class BotStatus
{
    public string Id { get; set; } = string.Empty;
    public string Status { get; set; } = string.Empty;
    public bool IsRunning { get; set; }
    public int ActiveConversations { get; set; }
    public DateTime? StartedAt { get; set; }
    public string? ErrorMessage { get; set; }
}
