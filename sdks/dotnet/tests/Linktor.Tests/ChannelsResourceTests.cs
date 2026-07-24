using System.Net;
using System.Net.Sockets;
using System.Text;
using System.Text.Json;
using Linktor;
using Linktor.Types;
using Xunit;

namespace Linktor.Tests;

/// <summary>
/// End-to-end tests for <see cref="Linktor.Resources.ChannelsResource"/> that exercise the real
/// HTTP client (envelope unwrapping, snake_case binding) against a tiny in-process HTTP server.
/// The SDK builds its own <see cref="System.Net.Http.HttpClient"/>, so we drive it via BaseUrl
/// rather than by injecting a handler.
/// </summary>
public class ChannelsResourceTests
{
    /// <summary>A minimal single-shot HTTP server that records the request and returns a canned response.</summary>
    private sealed class StubServer : IDisposable
    {
        private readonly HttpListener _listener = new();
        public string BaseUrl { get; }
        public string? CapturedBody { get; private set; }
        public string? CapturedMethod { get; private set; }
        public string? CapturedPath { get; private set; }

        public StubServer(int statusCode, string responseJson)
        {
            var port = GetFreePort();
            BaseUrl = $"http://127.0.0.1:{port}";
            _listener.Prefixes.Add($"{BaseUrl}/");
            _listener.Start();
            _ = Task.Run(async () =>
            {
                var ctx = await _listener.GetContextAsync();
                CapturedMethod = ctx.Request.HttpMethod;
                CapturedPath = ctx.Request.Url?.AbsolutePath;
                using (var reader = new StreamReader(ctx.Request.InputStream, ctx.Request.ContentEncoding))
                    CapturedBody = await reader.ReadToEndAsync();

                var bytes = Encoding.UTF8.GetBytes(responseJson);
                ctx.Response.StatusCode = statusCode;
                ctx.Response.ContentType = "application/json";
                ctx.Response.ContentLength64 = bytes.Length;
                await ctx.Response.OutputStream.WriteAsync(bytes);
                ctx.Response.Close();
            });
        }

        private static int GetFreePort()
        {
            var l = new TcpListener(IPAddress.Loopback, 0);
            l.Start();
            var port = ((IPEndPoint)l.LocalEndpoint).Port;
            l.Stop();
            return port;
        }

        public void Dispose() => _listener.Close();
    }

    private static LinktorClient NewClient(string baseUrl) =>
        new(new LinktorClientOptions { BaseUrl = baseUrl, ApiKey = "lk_test" });

    [Fact]
    public async Task Connect_SurfacesQrCodeFromEnvelope()
    {
        const string response = """
        {"success":true,"data":{"channel":{"id":"chan_1","tenant_id":"t_1","type":"whatsapp_unofficial","name":"WA","enabled":true,"connection_status":"connecting"},"qr_code":"Q","expires_in":60}}
        """;
        using var server = new StubServer(200, response);
        var client = NewClient(server.BaseUrl);

        var result = await client.Channels.ConnectAsync("chan_1");

        Assert.Equal("Q", result.QrCode);
        Assert.Equal(60, result.ExpiresIn);
        Assert.Equal("chan_1", result.Channel.Id);
        Assert.Equal(ConnectionStatus.Connecting, result.Channel.ConnectionStatus);
        Assert.Equal("/channels/chan_1/connect", server.CapturedPath);
        Assert.Equal("POST", server.CapturedMethod);
    }

    [Fact]
    public async Task Create_SendsCredentialsAndSnakeCaseBody()
    {
        const string response = """
        {"success":true,"data":{"id":"chan_2","tenant_id":"t_1","type":"whatsapp","name":"Sales","enabled":true,"connection_status":"disconnected"}}
        """;
        using var server = new StubServer(201, response);
        var client = NewClient(server.BaseUrl);

        var created = await client.Channels.CreateAsync(new CreateChannelInput
        {
            Type = ChannelType.Whatsapp,
            Name = "Sales",
            Config = new Dictionary<string, string> { ["phone_number_id"] = "123" },
            Credentials = new Dictionary<string, string> { ["access_token"] = "secret-token" },
            WebhookUrl = "https://example.test/hook",
        });

        Assert.Equal("chan_2", created.Id);

        using var doc = JsonDocument.Parse(server.CapturedBody!);
        var root = doc.RootElement;
        Assert.Equal("whatsapp", root.GetProperty("type").GetString());
        Assert.Equal("Sales", root.GetProperty("name").GetString());
        Assert.Equal("secret-token", root.GetProperty("credentials").GetProperty("access_token").GetString());
        Assert.Equal("123", root.GetProperty("config").GetProperty("phone_number_id").GetString());
        Assert.Equal("https://example.test/hook", root.GetProperty("webhook_url").GetString());
    }

    [Fact]
    public async Task RequestPairCode_PostsPhoneNumberAndReturnsPairCode()
    {
        const string response = """
        {"success":true,"data":{"channel":{"id":"chan_3","tenant_id":"t_1","type":"whatsapp_unofficial","name":"WA","enabled":true,"connection_status":"connecting"},"pair_code":"AB12-CD34","expires_in":120}}
        """;
        using var server = new StubServer(200, response);
        var client = NewClient(server.BaseUrl);

        var result = await client.Channels.RequestPairCodeAsync("chan_3", "+5511999999999");

        Assert.Equal("AB12-CD34", result.PairCode);
        Assert.Equal("/channels/chan_3/pair", server.CapturedPath);

        using var doc = JsonDocument.Parse(server.CapturedBody!);
        Assert.Equal("+5511999999999", doc.RootElement.GetProperty("phone_number").GetString());
    }
}
