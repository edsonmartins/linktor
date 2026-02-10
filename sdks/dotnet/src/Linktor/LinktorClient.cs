using System.Net.Http.Json;
using System.Text.Json;
using Linktor.Types;
using Linktor.Utils;
using Linktor.Resources;

namespace Linktor;

public class LinktorClient
{
    private readonly HttpClient _http;
    private readonly string _baseUrl;
    private string? _apiKey;
    private string? _accessToken;
    private readonly int _maxRetries;
    private readonly JsonSerializerOptions _jsonOptions;

    public AuthResource Auth { get; }
    public ConversationsResource Conversations { get; }
    public ContactsResource Contacts { get; }
    public ChannelsResource Channels { get; }
    public BotsResource Bots { get; }
    public AIResource AI { get; }
    public KnowledgeBasesResource KnowledgeBases { get; }
    public FlowsResource Flows { get; }
    public VREResource VRE { get; }

    public LinktorClient(LinktorClientOptions options)
    {
        _baseUrl = options.BaseUrl.TrimEnd('/');
        _apiKey = options.ApiKey;
        _accessToken = options.AccessToken;
        _maxRetries = options.MaxRetries;

        _http = new HttpClient { Timeout = TimeSpan.FromSeconds(options.TimeoutSeconds) };
        _jsonOptions = new JsonSerializerOptions
        {
            PropertyNamingPolicy = JsonNamingPolicy.CamelCase,
            DefaultIgnoreCondition = System.Text.Json.Serialization.JsonIgnoreCondition.WhenWritingNull
        };

        Auth = new AuthResource(this);
        Conversations = new ConversationsResource(this);
        Contacts = new ContactsResource(this);
        Channels = new ChannelsResource(this);
        Bots = new BotsResource(this);
        AI = new AIResource(this);
        KnowledgeBases = new KnowledgeBasesResource(this);
        Flows = new FlowsResource(this);
        VRE = new VREResource(this);
    }

    internal void SetAccessToken(string? token) => _accessToken = token;

    internal async Task<T> GetAsync<T>(string path, CancellationToken ct = default)
    {
        return await RequestAsync<T>(HttpMethod.Get, path, null, ct);
    }

    internal async Task<T> PostAsync<T>(string path, object? body, CancellationToken ct = default)
    {
        return await RequestAsync<T>(HttpMethod.Post, path, body, ct);
    }

    internal async Task<T> PatchAsync<T>(string path, object body, CancellationToken ct = default)
    {
        return await RequestAsync<T>(HttpMethod.Patch, path, body, ct);
    }

    internal async Task DeleteAsync(string path, CancellationToken ct = default)
    {
        await RequestAsync<object?>(HttpMethod.Delete, path, null, ct);
    }

    private async Task<T> RequestAsync<T>(HttpMethod method, string path, object? body, CancellationToken ct)
    {
        var url = $"{_baseUrl}{path}";
        var attempts = 0;

        while (true)
        {
            attempts++;

            var request = new HttpRequestMessage(method, url);

            if (!string.IsNullOrEmpty(_apiKey))
                request.Headers.Add("X-API-Key", _apiKey);
            else if (!string.IsNullOrEmpty(_accessToken))
                request.Headers.Add("Authorization", $"Bearer {_accessToken}");

            if (body != null)
                request.Content = JsonContent.Create(body, options: _jsonOptions);

            var response = await _http.SendAsync(request, ct);
            var requestId = response.Headers.TryGetValues("X-Request-ID", out var ids)
                ? ids.FirstOrDefault()
                : null;

            if (response.IsSuccessStatusCode)
            {
                var content = await response.Content.ReadAsStringAsync(ct);
                if (string.IsNullOrEmpty(content) || content == "null")
                    return default!;

                // Try to parse as ApiResponse first
                try
                {
                    var apiResponse = JsonSerializer.Deserialize<ApiResponse<T>>(content, _jsonOptions);
                    if (apiResponse?.Success == true && apiResponse.Data != null)
                        return apiResponse.Data;
                }
                catch { }

                return JsonSerializer.Deserialize<T>(content, _jsonOptions)!;
            }

            // Handle rate limiting
            if ((int)response.StatusCode == 429 && attempts < _maxRetries)
            {
                var retryAfter = response.Headers.RetryAfter?.Delta?.TotalSeconds ?? 60;
                await Task.Delay(TimeSpan.FromSeconds(retryAfter), ct);
                continue;
            }

            // Handle server errors with retry
            if ((int)response.StatusCode >= 500 && attempts < _maxRetries)
            {
                await Task.Delay(TimeSpan.FromSeconds(Math.Pow(2, attempts)), ct);
                continue;
            }

            var errorContent = await response.Content.ReadAsStringAsync(ct);
            var message = "Request failed";
            try
            {
                var error = JsonSerializer.Deserialize<ApiError>(errorContent, _jsonOptions);
                if (error?.Message != null) message = error.Message;
            }
            catch { }

            throw LinktorException.FromStatus((int)response.StatusCode, message, requestId);
        }
    }

    // Static webhook utilities
    public static bool VerifyWebhookSignature(byte[] payload, string signature, string secret)
        => WebhookVerifier.VerifySignature(payload, signature, secret);

    public static string ComputeWebhookSignature(byte[] payload, string secret)
        => WebhookVerifier.ComputeSignature(payload, secret);

    public static WebhookEvent ConstructWebhookEvent(byte[] payload, Dictionary<string, string> headers, string secret, int? tolerance = null)
        => WebhookVerifier.ConstructEvent(payload, headers, secret, tolerance);
}

public class LinktorClientOptions
{
    public string BaseUrl { get; set; } = "https://api.linktor.io";
    public string? ApiKey { get; set; }
    public string? AccessToken { get; set; }
    public int TimeoutSeconds { get; set; } = 30;
    public int MaxRetries { get; set; } = 3;
}
