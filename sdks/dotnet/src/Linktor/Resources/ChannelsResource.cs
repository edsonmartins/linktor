using Linktor.Types;

namespace Linktor.Resources;

public class ChannelsResource
{
    private readonly LinktorClient _client;

    public ChannelsResource(LinktorClient client) => _client = client;

    /// <summary>Lists channels. The channels list is not paginated on the wire.</summary>
    public Task<List<Channel>> ListAsync(ListChannelsParams? parameters = null, CancellationToken ct = default)
    {
        var query = BuildQuery(parameters);
        return _client.GetAsync<List<Channel>>($"/channels{query}", ct);
    }

    public Task<Channel> GetAsync(string id, CancellationToken ct = default)
        => _client.GetAsync<Channel>($"/channels/{id}", ct);

    public Task<Channel> CreateAsync(CreateChannelInput input, CancellationToken ct = default)
        => _client.PostAsync<Channel>("/channels", input, ct);

    /// <summary>Updates a channel. Uses PUT per the backend contract.</summary>
    public Task<Channel> UpdateAsync(string id, UpdateChannelInput input, CancellationToken ct = default)
        => _client.PutAsync<Channel>($"/channels/{id}", input, ct);

    public Task DeleteAsync(string id, CancellationToken ct = default)
        => _client.DeleteAsync($"/channels/{id}", ct);

    /// <summary>
    /// Starts (or refreshes) a channel connection. For WhatsApp Web-style linking
    /// the returned <see cref="ConnectResult"/> carries the QR payload (<c>QrCode</c>)
    /// to render and its lifetime (<c>ExpiresIn</c>); call connect again to refresh
    /// an expired code.
    /// </summary>
    public Task<ConnectResult> ConnectAsync(string id, CancellationToken ct = default)
        => _client.PostAsync<ConnectResult>($"/channels/{id}/connect", new { }, ct);

    /// <summary>
    /// Requests a WhatsApp pairing code for the given phone number as an
    /// alternative to QR linking.
    /// </summary>
    public Task<ConnectResult> RequestPairCodeAsync(string id, string phoneNumber, CancellationToken ct = default)
        => _client.PostAsync<ConnectResult>($"/channels/{id}/pair", new PairCodeInput { PhoneNumber = phoneNumber }, ct);

    public Task<Channel> DisconnectAsync(string id, CancellationToken ct = default)
        => _client.PostAsync<Channel>($"/channels/{id}/disconnect", new { }, ct);

    private static string BuildQuery(ListChannelsParams? p)
    {
        if (p == null) return "";
        var parts = new List<string>();
        if (!string.IsNullOrEmpty(p.Type)) parts.Add($"type={Uri.EscapeDataString(p.Type)}");
        if (!string.IsNullOrEmpty(p.Status)) parts.Add($"status={Uri.EscapeDataString(p.Status)}");
        if (!string.IsNullOrEmpty(p.Search)) parts.Add($"search={Uri.EscapeDataString(p.Search)}");
        return parts.Count > 0 ? "?" + string.Join("&", parts) : "";
    }
}
