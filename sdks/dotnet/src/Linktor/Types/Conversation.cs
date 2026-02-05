using System.Text.Json.Serialization;

namespace Linktor.Types;

[JsonConverter(typeof(JsonStringEnumConverter))]
public enum ConversationStatus
{
    Open,
    Pending,
    Resolved,
    Closed
}

[JsonConverter(typeof(JsonStringEnumConverter))]
public enum ConversationPriority
{
    Low,
    Medium,
    High,
    Urgent
}

[JsonConverter(typeof(JsonStringEnumConverter))]
public enum MessageType
{
    Text,
    Image,
    Video,
    Audio,
    Document,
    Location,
    Contact,
    Sticker,
    Template,
    Interactive,
    System
}

[JsonConverter(typeof(JsonStringEnumConverter))]
public enum MessageStatus
{
    Pending,
    Sent,
    Delivered,
    Read,
    Failed
}

[JsonConverter(typeof(JsonStringEnumConverter))]
public enum MessageDirection
{
    Inbound,
    Outbound
}

public class Conversation
{
    [JsonPropertyName("id")]
    public string Id { get; set; } = string.Empty;

    [JsonPropertyName("tenantId")]
    public string TenantId { get; set; } = string.Empty;

    [JsonPropertyName("channelId")]
    public string ChannelId { get; set; } = string.Empty;

    [JsonPropertyName("contactId")]
    public string ContactId { get; set; } = string.Empty;

    [JsonPropertyName("assignedAgentId")]
    public string? AssignedAgentId { get; set; }

    [JsonPropertyName("botId")]
    public string? BotId { get; set; }

    [JsonPropertyName("status")]
    public ConversationStatus Status { get; set; }

    [JsonPropertyName("priority")]
    public ConversationPriority? Priority { get; set; }

    [JsonPropertyName("subject")]
    public string? Subject { get; set; }

    [JsonPropertyName("lastMessage")]
    public Message? LastMessage { get; set; }

    [JsonPropertyName("unreadCount")]
    public int UnreadCount { get; set; }

    [JsonPropertyName("tags")]
    public List<string> Tags { get; set; } = new();

    [JsonPropertyName("metadata")]
    public Dictionary<string, object>? Metadata { get; set; }

    [JsonPropertyName("firstMessageAt")]
    public DateTime? FirstMessageAt { get; set; }

    [JsonPropertyName("lastMessageAt")]
    public DateTime? LastMessageAt { get; set; }

    [JsonPropertyName("resolvedAt")]
    public DateTime? ResolvedAt { get; set; }

    [JsonPropertyName("createdAt")]
    public DateTime CreatedAt { get; set; }

    [JsonPropertyName("updatedAt")]
    public DateTime UpdatedAt { get; set; }
}

public class Message
{
    [JsonPropertyName("id")]
    public string Id { get; set; } = string.Empty;

    [JsonPropertyName("conversationId")]
    public string ConversationId { get; set; } = string.Empty;

    [JsonPropertyName("type")]
    public MessageType Type { get; set; }

    [JsonPropertyName("direction")]
    public MessageDirection Direction { get; set; }

    [JsonPropertyName("status")]
    public MessageStatus Status { get; set; }

    [JsonPropertyName("text")]
    public string? Text { get; set; }

    [JsonPropertyName("media")]
    public MediaContent? Media { get; set; }

    [JsonPropertyName("location")]
    public LocationContent? Location { get; set; }

    [JsonPropertyName("contact")]
    public ContactContent? Contact { get; set; }

    [JsonPropertyName("template")]
    public TemplateContent? Template { get; set; }

    [JsonPropertyName("interactive")]
    public InteractiveContent? Interactive { get; set; }

    [JsonPropertyName("senderId")]
    public string? SenderId { get; set; }

    [JsonPropertyName("senderType")]
    public string? SenderType { get; set; }

    [JsonPropertyName("externalId")]
    public string? ExternalId { get; set; }

    [JsonPropertyName("metadata")]
    public Dictionary<string, object>? Metadata { get; set; }

    [JsonPropertyName("createdAt")]
    public DateTime CreatedAt { get; set; }

    [JsonPropertyName("updatedAt")]
    public DateTime UpdatedAt { get; set; }
}

public class MediaContent
{
    [JsonPropertyName("url")]
    public string Url { get; set; } = string.Empty;

    [JsonPropertyName("mimeType")]
    public string? MimeType { get; set; }

    [JsonPropertyName("filename")]
    public string? Filename { get; set; }

    [JsonPropertyName("size")]
    public long? Size { get; set; }

    [JsonPropertyName("caption")]
    public string? Caption { get; set; }
}

public class LocationContent
{
    [JsonPropertyName("latitude")]
    public double Latitude { get; set; }

    [JsonPropertyName("longitude")]
    public double Longitude { get; set; }

    [JsonPropertyName("name")]
    public string? Name { get; set; }

    [JsonPropertyName("address")]
    public string? Address { get; set; }
}

public class ContactContent
{
    [JsonPropertyName("name")]
    public string Name { get; set; } = string.Empty;

    [JsonPropertyName("phones")]
    public List<PhoneNumber> Phones { get; set; } = new();

    [JsonPropertyName("emails")]
    public List<string> Emails { get; set; } = new();
}

public class PhoneNumber
{
    [JsonPropertyName("type")]
    public string Type { get; set; } = string.Empty;

    [JsonPropertyName("number")]
    public string Number { get; set; } = string.Empty;
}

public class TemplateContent
{
    [JsonPropertyName("name")]
    public string Name { get; set; } = string.Empty;

    [JsonPropertyName("language")]
    public string Language { get; set; } = string.Empty;

    [JsonPropertyName("components")]
    public List<TemplateComponent> Components { get; set; } = new();
}

public class TemplateComponent
{
    [JsonPropertyName("type")]
    public string Type { get; set; } = string.Empty;

    [JsonPropertyName("parameters")]
    public List<TemplateParameter> Parameters { get; set; } = new();
}

public class TemplateParameter
{
    [JsonPropertyName("type")]
    public string Type { get; set; } = string.Empty;

    [JsonPropertyName("text")]
    public string? Text { get; set; }

    [JsonPropertyName("image")]
    public MediaContent? Image { get; set; }

    [JsonPropertyName("document")]
    public MediaContent? Document { get; set; }

    [JsonPropertyName("video")]
    public MediaContent? Video { get; set; }
}

public class InteractiveContent
{
    [JsonPropertyName("type")]
    public string Type { get; set; } = string.Empty;

    [JsonPropertyName("header")]
    public InteractiveHeader? Header { get; set; }

    [JsonPropertyName("body")]
    public InteractiveBody? Body { get; set; }

    [JsonPropertyName("footer")]
    public InteractiveFooter? Footer { get; set; }

    [JsonPropertyName("action")]
    public InteractiveAction? Action { get; set; }
}

public class InteractiveHeader
{
    [JsonPropertyName("type")]
    public string Type { get; set; } = string.Empty;

    [JsonPropertyName("text")]
    public string? Text { get; set; }

    [JsonPropertyName("image")]
    public MediaContent? Image { get; set; }

    [JsonPropertyName("video")]
    public MediaContent? Video { get; set; }

    [JsonPropertyName("document")]
    public MediaContent? Document { get; set; }
}

public class InteractiveBody
{
    [JsonPropertyName("text")]
    public string Text { get; set; } = string.Empty;
}

public class InteractiveFooter
{
    [JsonPropertyName("text")]
    public string Text { get; set; } = string.Empty;
}

public class InteractiveAction
{
    [JsonPropertyName("buttons")]
    public List<Button> Buttons { get; set; } = new();

    [JsonPropertyName("sections")]
    public List<Section> Sections { get; set; } = new();
}

public class Button
{
    [JsonPropertyName("type")]
    public string Type { get; set; } = string.Empty;

    [JsonPropertyName("id")]
    public string Id { get; set; } = string.Empty;

    [JsonPropertyName("title")]
    public string Title { get; set; } = string.Empty;
}

public class Section
{
    [JsonPropertyName("title")]
    public string Title { get; set; } = string.Empty;

    [JsonPropertyName("rows")]
    public List<SectionRow> Rows { get; set; } = new();
}

public class SectionRow
{
    [JsonPropertyName("id")]
    public string Id { get; set; } = string.Empty;

    [JsonPropertyName("title")]
    public string Title { get; set; } = string.Empty;

    [JsonPropertyName("description")]
    public string? Description { get; set; }
}

// Input classes

public class ListConversationsParams : PaginationParams
{
    [JsonPropertyName("status")]
    public ConversationStatus? Status { get; set; }

    [JsonPropertyName("priority")]
    public ConversationPriority? Priority { get; set; }

    [JsonPropertyName("channelId")]
    public string? ChannelId { get; set; }

    [JsonPropertyName("contactId")]
    public string? ContactId { get; set; }

    [JsonPropertyName("assignedAgentId")]
    public string? AssignedAgentId { get; set; }

    [JsonPropertyName("tag")]
    public string? Tag { get; set; }

    [JsonPropertyName("search")]
    public string? Search { get; set; }
}

public class SendMessageInput
{
    [JsonPropertyName("text")]
    public string? Text { get; set; }

    [JsonPropertyName("type")]
    public MessageType? Type { get; set; }

    [JsonPropertyName("media")]
    public MediaContent? Media { get; set; }

    [JsonPropertyName("location")]
    public LocationContent? Location { get; set; }

    [JsonPropertyName("contact")]
    public ContactContent? Contact { get; set; }

    [JsonPropertyName("template")]
    public TemplateContent? Template { get; set; }

    [JsonPropertyName("interactive")]
    public InteractiveContent? Interactive { get; set; }

    [JsonPropertyName("metadata")]
    public Dictionary<string, object>? Metadata { get; set; }

    public static SendMessageInput TextMessage(string text) => new()
    {
        Text = text,
        Type = MessageType.Text
    };
}

public class UpdateConversationInput
{
    [JsonPropertyName("status")]
    public ConversationStatus? Status { get; set; }

    [JsonPropertyName("priority")]
    public ConversationPriority? Priority { get; set; }

    [JsonPropertyName("assignedAgentId")]
    public string? AssignedAgentId { get; set; }

    [JsonPropertyName("tags")]
    public List<string>? Tags { get; set; }

    [JsonPropertyName("metadata")]
    public Dictionary<string, object>? Metadata { get; set; }
}
