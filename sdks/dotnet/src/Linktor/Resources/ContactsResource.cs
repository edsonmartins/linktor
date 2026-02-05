using Linktor.Types;

namespace Linktor.Resources;

public class ContactsResource
{
    private readonly LinktorClient _client;

    public ContactsResource(LinktorClient client) => _client = client;

    public Task<PaginatedResponse<Contact>> ListAsync(ListContactsParams? parameters = null, CancellationToken ct = default)
    {
        var query = BuildQuery(parameters);
        return _client.GetAsync<PaginatedResponse<Contact>>($"/contacts{query}", ct);
    }

    public Task<Contact> GetAsync(string id, CancellationToken ct = default)
        => _client.GetAsync<Contact>($"/contacts/{id}", ct);

    public Task<Contact> CreateAsync(CreateContactInput input, CancellationToken ct = default)
        => _client.PostAsync<Contact>("/contacts", input, ct);

    public Task<Contact> UpdateAsync(string id, UpdateContactInput input, CancellationToken ct = default)
        => _client.PatchAsync<Contact>($"/contacts/{id}", input, ct);

    public Task DeleteAsync(string id, CancellationToken ct = default)
        => _client.DeleteAsync($"/contacts/{id}", ct);

    public Task<PaginatedResponse<Contact>> SearchAsync(SearchContactsInput input, CancellationToken ct = default)
        => _client.PostAsync<PaginatedResponse<Contact>>("/contacts/search", input, ct);

    public Task<Contact> MergeAsync(MergeContactsInput input, CancellationToken ct = default)
        => _client.PostAsync<Contact>("/contacts/merge", input, ct);

    public Task<Contact> GetByExternalIdAsync(string externalId, CancellationToken ct = default)
        => _client.GetAsync<Contact>($"/contacts/external/{Uri.EscapeDataString(externalId)}", ct);

    private static string BuildQuery(ListContactsParams? p)
    {
        if (p == null) return "";
        var parts = new List<string>();
        if (p.Limit.HasValue) parts.Add($"limit={p.Limit}");
        if (p.Offset.HasValue) parts.Add($"offset={p.Offset}");
        if (!string.IsNullOrEmpty(p.Cursor)) parts.Add($"cursor={Uri.EscapeDataString(p.Cursor)}");
        if (!string.IsNullOrEmpty(p.Search)) parts.Add($"search={Uri.EscapeDataString(p.Search)}");
        if (!string.IsNullOrEmpty(p.Tag)) parts.Add($"tag={Uri.EscapeDataString(p.Tag)}");
        return parts.Count > 0 ? "?" + string.Join("&", parts) : "";
    }
}

public class ListContactsParams
{
    public int? Limit { get; set; }
    public int? Offset { get; set; }
    public string? Cursor { get; set; }
    public string? Search { get; set; }
    public string? Tag { get; set; }
}

public class CreateContactInput
{
    public string? Name { get; set; }
    public string? Email { get; set; }
    public string? Phone { get; set; }
    public string? ExternalId { get; set; }
    public string? AvatarUrl { get; set; }
    public List<string>? Tags { get; set; }
    public Dictionary<string, object>? CustomFields { get; set; }
    public Dictionary<string, object>? Metadata { get; set; }
}

public class UpdateContactInput
{
    public string? Name { get; set; }
    public string? Email { get; set; }
    public string? Phone { get; set; }
    public string? ExternalId { get; set; }
    public string? AvatarUrl { get; set; }
    public List<string>? Tags { get; set; }
    public Dictionary<string, object>? CustomFields { get; set; }
    public Dictionary<string, object>? Metadata { get; set; }
}

public class SearchContactsInput
{
    public string? Query { get; set; }
    public Dictionary<string, object>? Filters { get; set; }
    public int? Limit { get; set; }
    public int? Offset { get; set; }
}

public class MergeContactsInput
{
    public string PrimaryId { get; set; } = string.Empty;
    public List<string> SecondaryIds { get; set; } = new();
}
