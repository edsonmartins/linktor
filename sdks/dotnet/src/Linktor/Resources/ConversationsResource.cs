using Linktor.Types;

namespace Linktor.Resources;

public class ConversationsResource
{
    private readonly LinktorClient _client;

    public ConversationsResource(LinktorClient client) => _client = client;

    public Task<PaginatedResponse<Conversation>> ListAsync(ListConversationsParams? parameters = null, CancellationToken ct = default)
    {
        var query = BuildQuery(parameters);
        return _client.GetAsync<PaginatedResponse<Conversation>>($"/conversations{query}", ct);
    }

    public Task<Conversation> GetAsync(string id, CancellationToken ct = default)
        => _client.GetAsync<Conversation>($"/conversations/{id}", ct);

    public Task<Conversation> CreateAsync(CreateConversationInput input, CancellationToken ct = default)
        => _client.PostAsync<Conversation>("/conversations", input, ct);

    public Task<Conversation> UpdateAsync(string id, UpdateConversationInput input, CancellationToken ct = default)
        => _client.PatchAsync<Conversation>($"/conversations/{id}", input, ct);

    public Task DeleteAsync(string id, CancellationToken ct = default)
        => _client.DeleteAsync($"/conversations/{id}", ct);

    public Task<PaginatedResponse<Message>> GetMessagesAsync(string conversationId, ListMessagesParams? parameters = null, CancellationToken ct = default)
    {
        var query = BuildQuery(parameters);
        return _client.GetAsync<PaginatedResponse<Message>>($"/conversations/{conversationId}/messages{query}", ct);
    }

    public Task<Message> SendMessageAsync(string conversationId, SendMessageInput input, CancellationToken ct = default)
        => _client.PostAsync<Message>($"/conversations/{conversationId}/messages", input, ct);

    public Task<Conversation> AssignAsync(string id, AssignConversationInput input, CancellationToken ct = default)
        => _client.PostAsync<Conversation>($"/conversations/{id}/assign", input, ct);

    public Task<Conversation> ResolveAsync(string id, CancellationToken ct = default)
        => _client.PostAsync<Conversation>($"/conversations/{id}/resolve", new { }, ct);

    public Task<Conversation> ReopenAsync(string id, CancellationToken ct = default)
        => _client.PostAsync<Conversation>($"/conversations/{id}/reopen", new { }, ct);

    private static string BuildQuery(ListConversationsParams? p)
    {
        if (p == null) return "";
        var parts = new List<string>();
        if (p.Limit.HasValue) parts.Add($"limit={p.Limit}");
        if (p.Offset.HasValue) parts.Add($"offset={p.Offset}");
        if (!string.IsNullOrEmpty(p.Cursor)) parts.Add($"cursor={Uri.EscapeDataString(p.Cursor)}");
        if (!string.IsNullOrEmpty(p.Status)) parts.Add($"status={Uri.EscapeDataString(p.Status)}");
        if (!string.IsNullOrEmpty(p.ChannelId)) parts.Add($"channelId={Uri.EscapeDataString(p.ChannelId)}");
        if (!string.IsNullOrEmpty(p.ContactId)) parts.Add($"contactId={Uri.EscapeDataString(p.ContactId)}");
        if (!string.IsNullOrEmpty(p.AssignedTo)) parts.Add($"assignedTo={Uri.EscapeDataString(p.AssignedTo)}");
        return parts.Count > 0 ? "?" + string.Join("&", parts) : "";
    }

    private static string BuildQuery(ListMessagesParams? p)
    {
        if (p == null) return "";
        var parts = new List<string>();
        if (p.Limit.HasValue) parts.Add($"limit={p.Limit}");
        if (p.Offset.HasValue) parts.Add($"offset={p.Offset}");
        if (!string.IsNullOrEmpty(p.Cursor)) parts.Add($"cursor={Uri.EscapeDataString(p.Cursor)}");
        if (!string.IsNullOrEmpty(p.Direction)) parts.Add($"direction={Uri.EscapeDataString(p.Direction)}");
        return parts.Count > 0 ? "?" + string.Join("&", parts) : "";
    }
}

public class ListConversationsParams
{
    public int? Limit { get; set; }
    public int? Offset { get; set; }
    public string? Cursor { get; set; }
    public string? Status { get; set; }
    public string? ChannelId { get; set; }
    public string? ContactId { get; set; }
    public string? AssignedTo { get; set; }
}

public class ListMessagesParams
{
    public int? Limit { get; set; }
    public int? Offset { get; set; }
    public string? Cursor { get; set; }
    public string? Direction { get; set; }
}

public class CreateConversationInput
{
    public string ChannelId { get; set; } = string.Empty;
    public string ContactId { get; set; } = string.Empty;
    public Dictionary<string, object>? Metadata { get; set; }
}

public class UpdateConversationInput
{
    public string? Status { get; set; }
    public string? AssignedTo { get; set; }
    public Dictionary<string, object>? Metadata { get; set; }
}

public class SendMessageInput
{
    public string? Text { get; set; }
    public string? ContentType { get; set; }
    public MessageMedia? Media { get; set; }
    public List<QuickReply>? QuickReplies { get; set; }
    public Dictionary<string, object>? Metadata { get; set; }
}

public class AssignConversationInput
{
    public string UserId { get; set; } = string.Empty;
}
