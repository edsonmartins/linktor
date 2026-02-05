using System.Text.Json.Serialization;

namespace Linktor.Types;

public static class WebhookConstants
{
    public const string SignatureHeader = "X-Linktor-Signature";
    public const string TimestampHeader = "X-Linktor-Timestamp";
    public const int DefaultToleranceSeconds = 300;
}

public enum EventType
{
    MessageReceived,
    MessageSent,
    MessageDelivered,
    MessageRead,
    MessageFailed,
    ConversationCreated,
    ConversationUpdated,
    ConversationResolved,
    ConversationAssigned,
    ContactCreated,
    ContactUpdated,
    ContactDeleted,
    ChannelConnected,
    ChannelDisconnected,
    ChannelError,
    BotStarted,
    BotStopped,
    FlowStarted,
    FlowCompleted,
    FlowFailed
}

public class WebhookEvent
{
    [JsonPropertyName("id")] public string Id { get; set; } = string.Empty;
    [JsonPropertyName("type")] public string Type { get; set; } = string.Empty;
    [JsonPropertyName("timestamp")] public DateTime Timestamp { get; set; }
    [JsonPropertyName("tenantId")] public string TenantId { get; set; } = string.Empty;
    [JsonPropertyName("data")] public Dictionary<string, object>? Data { get; set; }

    public EventType? GetEventType() => Type switch
    {
        "message.received" => EventType.MessageReceived,
        "message.sent" => EventType.MessageSent,
        "message.delivered" => EventType.MessageDelivered,
        "message.read" => EventType.MessageRead,
        "message.failed" => EventType.MessageFailed,
        "conversation.created" => EventType.ConversationCreated,
        "conversation.updated" => EventType.ConversationUpdated,
        "conversation.resolved" => EventType.ConversationResolved,
        "conversation.assigned" => EventType.ConversationAssigned,
        "contact.created" => EventType.ContactCreated,
        "contact.updated" => EventType.ContactUpdated,
        "contact.deleted" => EventType.ContactDeleted,
        "channel.connected" => EventType.ChannelConnected,
        "channel.disconnected" => EventType.ChannelDisconnected,
        "channel.error" => EventType.ChannelError,
        "bot.started" => EventType.BotStarted,
        "bot.stopped" => EventType.BotStopped,
        "flow.started" => EventType.FlowStarted,
        "flow.completed" => EventType.FlowCompleted,
        "flow.failed" => EventType.FlowFailed,
        _ => null
    };
}

public class WebhookConfig
{
    [JsonPropertyName("url")] public string Url { get; set; } = string.Empty;
    [JsonPropertyName("secret")] public string Secret { get; set; } = string.Empty;
    [JsonPropertyName("events")] public List<string> Events { get; set; } = new();
    [JsonPropertyName("enabled")] public bool Enabled { get; set; }
    [JsonPropertyName("headers")] public Dictionary<string, string>? Headers { get; set; }
}
