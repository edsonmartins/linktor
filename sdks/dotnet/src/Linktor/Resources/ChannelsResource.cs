using Linktor.Types;

namespace Linktor.Resources;

public class ChannelsResource
{
    private readonly LinktorClient _client;

    public ChannelsResource(LinktorClient client) => _client = client;

    public Task<PaginatedResponse<Channel>> ListAsync(ListChannelsParams? parameters = null, CancellationToken ct = default)
    {
        var query = BuildQuery(parameters);
        return _client.GetAsync<PaginatedResponse<Channel>>($"/channels{query}", ct);
    }

    public Task<Channel> GetAsync(string id, CancellationToken ct = default)
        => _client.GetAsync<Channel>($"/channels/{id}", ct);

    public Task<Channel> CreateAsync(CreateChannelInput input, CancellationToken ct = default)
        => _client.PostAsync<Channel>("/channels", input, ct);

    public Task<Channel> UpdateAsync(string id, UpdateChannelInput input, CancellationToken ct = default)
        => _client.PatchAsync<Channel>($"/channels/{id}", input, ct);

    public Task DeleteAsync(string id, CancellationToken ct = default)
        => _client.DeleteAsync($"/channels/{id}", ct);

    public Task<Channel> ConnectAsync(string id, CancellationToken ct = default)
        => _client.PostAsync<Channel>($"/channels/{id}/connect", new { }, ct);

    public Task<Channel> DisconnectAsync(string id, CancellationToken ct = default)
        => _client.PostAsync<Channel>($"/channels/{id}/disconnect", new { }, ct);

    public Task<ChannelStatus> GetStatusAsync(string id, CancellationToken ct = default)
        => _client.GetAsync<ChannelStatus>($"/channels/{id}/status", ct);

    public Task<Channel> TestAsync(string id, CancellationToken ct = default)
        => _client.PostAsync<Channel>($"/channels/{id}/test", new { }, ct);

    private static string BuildQuery(ListChannelsParams? p)
    {
        if (p == null) return "";
        var parts = new List<string>();
        if (p.Limit.HasValue) parts.Add($"limit={p.Limit}");
        if (p.Offset.HasValue) parts.Add($"offset={p.Offset}");
        if (!string.IsNullOrEmpty(p.Type)) parts.Add($"type={Uri.EscapeDataString(p.Type)}");
        if (!string.IsNullOrEmpty(p.Status)) parts.Add($"status={Uri.EscapeDataString(p.Status)}");
        return parts.Count > 0 ? "?" + string.Join("&", parts) : "";
    }
}

public class ListChannelsParams
{
    public int? Limit { get; set; }
    public int? Offset { get; set; }
    public string? Type { get; set; }
    public string? Status { get; set; }
}

public class CreateChannelInput
{
    public string Name { get; set; } = string.Empty;
    public string Type { get; set; } = string.Empty;
    public Dictionary<string, object> Config { get; set; } = new();
    public bool Enabled { get; set; } = true;
    public Dictionary<string, object>? Metadata { get; set; }
}

public class UpdateChannelInput
{
    public string? Name { get; set; }
    public Dictionary<string, object>? Config { get; set; }
    public bool? Enabled { get; set; }
    public Dictionary<string, object>? Metadata { get; set; }
}

public class ChannelStatus
{
    public string Id { get; set; } = string.Empty;
    public string Status { get; set; } = string.Empty;
    public bool IsConnected { get; set; }
    public DateTime? LastActivityAt { get; set; }
    public string? ErrorMessage { get; set; }
    public Dictionary<string, object>? Details { get; set; }
}
