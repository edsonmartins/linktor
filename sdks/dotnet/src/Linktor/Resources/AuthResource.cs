using Linktor.Types;

namespace Linktor.Resources;

public class AuthResource
{
    private readonly LinktorClient _client;

    public AuthResource(LinktorClient client) => _client = client;

    public async Task<LoginResponse> LoginAsync(string email, string password, CancellationToken ct = default)
    {
        var input = new LoginInput(email, password);
        var response = await _client.PostAsync<LoginResponse>("/auth/login", input, ct);
        _client.SetAccessToken(response.AccessToken);
        return response;
    }

    public async Task LogoutAsync(CancellationToken ct = default)
    {
        await _client.PostAsync<object?>("/auth/logout", new { }, ct);
        _client.SetAccessToken(null);
    }

    public async Task<RefreshTokenResponse> RefreshTokenAsync(string refreshToken, CancellationToken ct = default)
    {
        var input = new RefreshTokenInput { RefreshToken = refreshToken };
        var response = await _client.PostAsync<RefreshTokenResponse>("/auth/refresh", input, ct);
        _client.SetAccessToken(response.AccessToken);
        return response;
    }

    public Task<User> GetCurrentUserAsync(CancellationToken ct = default)
        => _client.GetAsync<User>("/auth/me", ct);

    public Task<Tenant> GetCurrentTenantAsync(CancellationToken ct = default)
        => _client.GetAsync<Tenant>("/auth/tenant", ct);
}
