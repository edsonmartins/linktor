using System.Security.Cryptography;
using System.Text;
using System.Text.Json;
using Linktor.Types;

namespace Linktor.Utils;

public static class WebhookVerifier
{
    public static string ComputeSignature(byte[] payload, string secret)
    {
        using var hmac = new HMACSHA256(Encoding.UTF8.GetBytes(secret));
        var hash = hmac.ComputeHash(payload);
        return Convert.ToHexString(hash).ToLowerInvariant();
    }

    public static string ComputeSignature(string payload, string secret)
        => ComputeSignature(Encoding.UTF8.GetBytes(payload), secret);

    public static bool VerifySignature(byte[] payload, string signature, string secret)
    {
        if (string.IsNullOrEmpty(signature) || string.IsNullOrEmpty(secret))
            return false;

        var expected = ComputeSignature(payload, secret);
        return CryptographicOperations.FixedTimeEquals(
            Encoding.UTF8.GetBytes(signature.ToLowerInvariant()),
            Encoding.UTF8.GetBytes(expected)
        );
    }

    public static bool VerifySignature(string payload, string signature, string secret)
        => VerifySignature(Encoding.UTF8.GetBytes(payload), signature, secret);

    public static bool Verify(byte[] payload, Dictionary<string, string> headers, string secret, int? toleranceSeconds = null)
    {
        var tolerance = toleranceSeconds ?? WebhookConstants.DefaultToleranceSeconds;

        // Get signature (case-insensitive)
        var signature = headers.TryGetValue(WebhookConstants.SignatureHeader, out var sig) ? sig
            : headers.TryGetValue(WebhookConstants.SignatureHeader.ToLower(), out sig) ? sig
            : null;

        if (string.IsNullOrEmpty(signature))
            return false;

        // Verify timestamp if present
        var timestampStr = headers.TryGetValue(WebhookConstants.TimestampHeader, out var ts) ? ts
            : headers.TryGetValue(WebhookConstants.TimestampHeader.ToLower(), out ts) ? ts
            : null;

        if (!string.IsNullOrEmpty(timestampStr))
        {
            if (!long.TryParse(timestampStr, out var timestamp))
                return false;

            var now = DateTimeOffset.UtcNow.ToUnixTimeSeconds();
            if (Math.Abs(now - timestamp) > tolerance)
                return false;
        }

        return VerifySignature(payload, signature, secret);
    }

    public static WebhookEvent ConstructEvent(byte[] payload, Dictionary<string, string> headers, string secret, int? toleranceSeconds = null)
    {
        var tolerance = toleranceSeconds == 0 ? WebhookConstants.DefaultToleranceSeconds
            : toleranceSeconds ?? WebhookConstants.DefaultToleranceSeconds;

        if (!Verify(payload, headers, secret, tolerance))
            throw new WebhookVerificationException("Webhook signature verification failed");

        try
        {
            var json = Encoding.UTF8.GetString(payload);
            var ev = JsonSerializer.Deserialize<WebhookEvent>(json, new JsonSerializerOptions
            {
                PropertyNamingPolicy = JsonNamingPolicy.CamelCase
            });

            if (ev == null || string.IsNullOrEmpty(ev.Id) || string.IsNullOrEmpty(ev.Type))
                throw new WebhookVerificationException("Invalid webhook event structure");

            return ev;
        }
        catch (JsonException ex)
        {
            throw new WebhookVerificationException($"Failed to parse webhook event: {ex.Message}");
        }
    }

    public static WebhookEvent ConstructEvent(string payload, Dictionary<string, string> headers, string secret, int? toleranceSeconds = null)
        => ConstructEvent(Encoding.UTF8.GetBytes(payload), headers, secret, toleranceSeconds);
}
