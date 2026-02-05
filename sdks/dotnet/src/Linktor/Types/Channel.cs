using System.Text.Json.Serialization;

namespace Linktor.Types;

[JsonConverter(typeof(JsonStringEnumConverter))]
public enum ChannelType
{
    Whatsapp,
    WhatsappUnofficial,
    Telegram,
    Facebook,
    Instagram,
    Webchat,
    Sms,
    Email,
    Rcs
}

[JsonConverter(typeof(JsonStringEnumConverter))]
public enum ChannelStatus
{
    Connected,
    Disconnected,
    Connecting,
    Error
}

public class Channel
{
    [JsonPropertyName("id")]
    public string Id { get; set; } = string.Empty;

    [JsonPropertyName("tenantId")]
    public string TenantId { get; set; } = string.Empty;

    [JsonPropertyName("name")]
    public string Name { get; set; } = string.Empty;

    [JsonPropertyName("type")]
    public ChannelType Type { get; set; }

    [JsonPropertyName("status")]
    public ChannelStatus Status { get; set; }

    [JsonPropertyName("config")]
    public Dictionary<string, object>? Config { get; set; }

    [JsonPropertyName("metadata")]
    public Dictionary<string, object>? Metadata { get; set; }

    [JsonPropertyName("errorMessage")]
    public string? ErrorMessage { get; set; }

    [JsonPropertyName("connectedAt")]
    public DateTime? ConnectedAt { get; set; }

    [JsonPropertyName("lastActivityAt")]
    public DateTime? LastActivityAt { get; set; }

    [JsonPropertyName("createdAt")]
    public DateTime CreatedAt { get; set; }

    [JsonPropertyName("updatedAt")]
    public DateTime UpdatedAt { get; set; }
}

public class CreateChannelInput
{
    [JsonPropertyName("name")]
    public string Name { get; set; } = string.Empty;

    [JsonPropertyName("type")]
    public ChannelType Type { get; set; }

    [JsonPropertyName("config")]
    public Dictionary<string, object>? Config { get; set; }

    [JsonPropertyName("metadata")]
    public Dictionary<string, object>? Metadata { get; set; }
}

public class UpdateChannelInput
{
    [JsonPropertyName("name")]
    public string? Name { get; set; }

    [JsonPropertyName("config")]
    public Dictionary<string, object>? Config { get; set; }

    [JsonPropertyName("metadata")]
    public Dictionary<string, object>? Metadata { get; set; }
}

public class ListChannelsParams : PaginationParams
{
    [JsonPropertyName("type")]
    public ChannelType? Type { get; set; }

    [JsonPropertyName("status")]
    public ChannelStatus? Status { get; set; }

    [JsonPropertyName("search")]
    public string? Search { get; set; }
}

public class ChannelStatusResponse
{
    [JsonPropertyName("status")]
    public ChannelStatus Status { get; set; }

    [JsonPropertyName("errorMessage")]
    public string? ErrorMessage { get; set; }

    [JsonPropertyName("connectedAt")]
    public DateTime? ConnectedAt { get; set; }

    [JsonPropertyName("lastActivityAt")]
    public DateTime? LastActivityAt { get; set; }
}
