using System.Text.Json.Serialization;

namespace Linktor.Types;

[JsonConverter(typeof(JsonStringEnumConverter))]
public enum BotStatus { Active, Inactive, Draft }

[JsonConverter(typeof(JsonStringEnumConverter))]
public enum BotType { Flow, Ai, Hybrid }

public class Bot
{
    [JsonPropertyName("id")] public string Id { get; set; } = string.Empty;
    [JsonPropertyName("tenantId")] public string TenantId { get; set; } = string.Empty;
    [JsonPropertyName("name")] public string Name { get; set; } = string.Empty;
    [JsonPropertyName("description")] public string? Description { get; set; }
    [JsonPropertyName("status")] public BotStatus Status { get; set; }
    [JsonPropertyName("type")] public BotType Type { get; set; }
    [JsonPropertyName("config")] public Dictionary<string, object>? Config { get; set; }
    [JsonPropertyName("channelIds")] public List<string> ChannelIds { get; set; } = new();
    [JsonPropertyName("flowId")] public string? FlowId { get; set; }
    [JsonPropertyName("agentId")] public string? AgentId { get; set; }
    [JsonPropertyName("knowledgeBaseIds")] public List<string> KnowledgeBaseIds { get; set; } = new();
    [JsonPropertyName("metadata")] public Dictionary<string, object>? Metadata { get; set; }
    [JsonPropertyName("createdAt")] public DateTime CreatedAt { get; set; }
    [JsonPropertyName("updatedAt")] public DateTime UpdatedAt { get; set; }
}

public class CreateBotInput
{
    [JsonPropertyName("name")] public string Name { get; set; } = string.Empty;
    [JsonPropertyName("description")] public string? Description { get; set; }
    [JsonPropertyName("type")] public BotType Type { get; set; }
    [JsonPropertyName("config")] public Dictionary<string, object>? Config { get; set; }
    [JsonPropertyName("channelIds")] public List<string>? ChannelIds { get; set; }
    [JsonPropertyName("flowId")] public string? FlowId { get; set; }
    [JsonPropertyName("agentId")] public string? AgentId { get; set; }
    [JsonPropertyName("knowledgeBaseIds")] public List<string>? KnowledgeBaseIds { get; set; }
    [JsonPropertyName("metadata")] public Dictionary<string, object>? Metadata { get; set; }
}

public class UpdateBotInput
{
    [JsonPropertyName("name")] public string? Name { get; set; }
    [JsonPropertyName("description")] public string? Description { get; set; }
    [JsonPropertyName("status")] public BotStatus? Status { get; set; }
    [JsonPropertyName("config")] public Dictionary<string, object>? Config { get; set; }
    [JsonPropertyName("channelIds")] public List<string>? ChannelIds { get; set; }
    [JsonPropertyName("flowId")] public string? FlowId { get; set; }
    [JsonPropertyName("agentId")] public string? AgentId { get; set; }
    [JsonPropertyName("knowledgeBaseIds")] public List<string>? KnowledgeBaseIds { get; set; }
    [JsonPropertyName("metadata")] public Dictionary<string, object>? Metadata { get; set; }
}

public class ListBotsParams : PaginationParams
{
    [JsonPropertyName("status")] public BotStatus? Status { get; set; }
    [JsonPropertyName("type")] public BotType? Type { get; set; }
    [JsonPropertyName("channelId")] public string? ChannelId { get; set; }
    [JsonPropertyName("search")] public string? Search { get; set; }
}
