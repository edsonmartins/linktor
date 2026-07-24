using System.Text.Json;
using System.Text.Json.Serialization;

namespace Linktor.Types;

/// <summary>
/// Channel type identifiers as sent on the wire (snake_case string values).
/// Modeled as string constants to mirror the backend contract exactly.
/// </summary>
public static class ChannelType
{
    public const string Webchat = "webchat";
    public const string Whatsapp = "whatsapp";
    public const string WhatsappOfficial = "whatsapp_official";
    public const string WhatsappUnofficial = "whatsapp_unofficial";
    public const string Telegram = "telegram";
    public const string Sms = "sms";
    public const string Rcs = "rcs";
    public const string Instagram = "instagram";
    public const string Facebook = "facebook";
    public const string Email = "email";
    public const string Voice = "voice";
    public const string Teams = "teams";
    public const string Slack = "slack";
    public const string Mattermost = "mattermost";
    public const string Direto = "direto";
}

/// <summary>
/// Live connection state of a channel (wire field <c>connection_status</c>),
/// distinct from <see cref="Channel.Enabled"/> which is the system-level enable flag.
/// </summary>
public static class ConnectionStatus
{
    public const string Disconnected = "disconnected";
    public const string Connecting = "connecting";
    public const string Connected = "connected";
    public const string Error = "error";
}

/// <summary>
/// WhatsApp Business App + Cloud API coexistence state (wire field <c>coexistence_status</c>).
/// </summary>
public static class CoexistenceStatus
{
    public const string Inactive = "inactive";
    public const string Pending = "pending";
    public const string Active = "active";
    public const string Warning = "warning";
    public const string Disconnected = "disconnected";
}

/// <summary>
/// Channel mirrors the backend wire shape exactly (snake_case). Credentials are
/// write-only and never present on a response. <c>config</c> is a flat string map
/// whose secret values are redacted to <c>"__redacted__"</c>.
/// </summary>
public class Channel
{
    [JsonPropertyName("id")]
    public string Id { get; set; } = string.Empty;

    [JsonPropertyName("tenant_id")]
    public string TenantId { get; set; } = string.Empty;

    [JsonPropertyName("type")]
    public string Type { get; set; } = string.Empty;

    [JsonPropertyName("name")]
    public string Name { get; set; } = string.Empty;

    [JsonPropertyName("identifier")]
    public string? Identifier { get; set; }

    /// <summary>System-level enable flag (distinct from <see cref="ConnectionStatus"/>).</summary>
    [JsonPropertyName("enabled")]
    public bool Enabled { get; set; }

    /// <summary>Live connection state. See <see cref="ConnectionStatus"/> for known values.</summary>
    [JsonPropertyName("connection_status")]
    public string ConnectionStatus { get; set; } = string.Empty;

    /// <summary>Flat string map of non-secret settings; secret values are redacted to "__redacted__".</summary>
    [JsonPropertyName("config")]
    public Dictionary<string, string>? Config { get; set; }

    [JsonPropertyName("webhook_url")]
    public string? WebhookUrl { get; set; }

    [JsonPropertyName("created_at")]
    public DateTime CreatedAt { get; set; }

    [JsonPropertyName("updated_at")]
    public DateTime UpdatedAt { get; set; }

    // WhatsApp coexistence

    [JsonPropertyName("is_coexistence")]
    public bool? IsCoexistence { get; set; }

    [JsonPropertyName("waba_id")]
    public string? WabaId { get; set; }

    [JsonPropertyName("last_echo_at")]
    public DateTime? LastEchoAt { get; set; }

    /// <summary>Coexistence state. See <see cref="CoexistenceStatus"/> for known values.</summary>
    [JsonPropertyName("coexistence_status")]
    public string? CoexistenceStatus { get; set; }

    [JsonPropertyName("message_template_namespace")]
    public string? MessageTemplateNamespace { get; set; }
}

/// <summary>
/// Body for creating a channel. <c>Config</c> holds non-secret settings
/// (e.g. phone_number_id, waba_id). <c>Credentials</c> holds secrets
/// (e.g. access_token, bot_token) — stored encrypted, never returned.
/// <c>WebhookUrl</c> is the external endpoint Linktor delivers signed
/// inbound/status events to.
/// </summary>
public class CreateChannelInput
{
    [JsonPropertyName("type")]
    public string Type { get; set; } = string.Empty;

    [JsonPropertyName("name")]
    public string Name { get; set; } = string.Empty;

    [JsonPropertyName("identifier")]
    public string? Identifier { get; set; }

    [JsonPropertyName("config")]
    public Dictionary<string, string>? Config { get; set; }

    [JsonPropertyName("credentials")]
    public Dictionary<string, string>? Credentials { get; set; }

    [JsonPropertyName("webhook_url")]
    public string? WebhookUrl { get; set; }
}

/// <summary>
/// Body for updating a channel; reuses the create shape server-side.
/// <c>Credentials</c>, when present, replace the stored secrets; omit it
/// (or send the redacted placeholder) to leave them untouched.
/// </summary>
public class UpdateChannelInput
{
    [JsonPropertyName("name")]
    public string? Name { get; set; }

    [JsonPropertyName("identifier")]
    public string? Identifier { get; set; }

    [JsonPropertyName("config")]
    public Dictionary<string, string>? Config { get; set; }

    [JsonPropertyName("credentials")]
    public Dictionary<string, string>? Credentials { get; set; }

    [JsonPropertyName("webhook_url")]
    public string? WebhookUrl { get; set; }
}

/// <summary>
/// Result of connecting a channel. For WhatsApp Web-style linking, <c>QrCode</c>
/// carries the payload to render and <c>ExpiresIn</c> its lifetime in seconds —
/// call connect again to refresh an expired code. <c>PairCode</c> is the
/// phone-linking code. When <c>PasskeyRequired</c> is true the account is
/// passkey-locked and must be linked by signing <c>PasskeyChallenge</c>.
/// </summary>
public class ConnectResult
{
    [JsonPropertyName("channel")]
    public Channel Channel { get; set; } = new();

    [JsonPropertyName("qr_code")]
    public string? QrCode { get; set; }

    [JsonPropertyName("expires_in")]
    public int? ExpiresIn { get; set; }

    [JsonPropertyName("pair_code")]
    public string? PairCode { get; set; }

    [JsonPropertyName("passkey_required")]
    public bool? PasskeyRequired { get; set; }

    /// <summary>Raw passkey challenge JSON; sign and submit via the passkey endpoint.</summary>
    [JsonPropertyName("passkey_challenge")]
    public JsonElement? PasskeyChallenge { get; set; }
}

/// <summary>Body for requesting a WhatsApp pairing code.</summary>
public class PairCodeInput
{
    [JsonPropertyName("phone_number")]
    public string PhoneNumber { get; set; } = string.Empty;
}

/// <summary>Query filters for listing channels.</summary>
public class ListChannelsParams
{
    /// <summary>Filter by channel type. See <see cref="ChannelType"/> for known values.</summary>
    public string? Type { get; set; }

    /// <summary>Filter by connection status. See <see cref="ConnectionStatus"/> for known values.</summary>
    public string? Status { get; set; }

    public string? Search { get; set; }
}
