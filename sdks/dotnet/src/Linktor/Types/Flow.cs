using System.Text.Json.Serialization;

namespace Linktor.Types;

[JsonConverter(typeof(JsonStringEnumConverter))]
public enum FlowStatus { Active, Inactive, Draft }

[JsonConverter(typeof(JsonStringEnumConverter))]
public enum FlowExecutionStatus { Running, Waiting, Completed, Failed, Cancelled }

public class Flow
{
    [JsonPropertyName("id")] public string Id { get; set; } = string.Empty;
    [JsonPropertyName("tenantId")] public string TenantId { get; set; } = string.Empty;
    [JsonPropertyName("name")] public string Name { get; set; } = string.Empty;
    [JsonPropertyName("description")] public string? Description { get; set; }
    [JsonPropertyName("status")] public FlowStatus Status { get; set; }
    [JsonPropertyName("version")] public int Version { get; set; }
    [JsonPropertyName("nodes")] public List<FlowNode> Nodes { get; set; } = new();
    [JsonPropertyName("edges")] public List<FlowEdge> Edges { get; set; } = new();
    [JsonPropertyName("variables")] public List<FlowVariable> Variables { get; set; } = new();
    [JsonPropertyName("metadata")] public Dictionary<string, object>? Metadata { get; set; }
    [JsonPropertyName("createdAt")] public DateTime CreatedAt { get; set; }
    [JsonPropertyName("updatedAt")] public DateTime UpdatedAt { get; set; }
}

public class FlowNode
{
    [JsonPropertyName("id")] public string Id { get; set; } = string.Empty;
    [JsonPropertyName("type")] public string Type { get; set; } = string.Empty;
    [JsonPropertyName("position")] public Dictionary<string, double>? Position { get; set; }
    [JsonPropertyName("data")] public Dictionary<string, object>? Data { get; set; }
}

public class FlowEdge
{
    [JsonPropertyName("id")] public string Id { get; set; } = string.Empty;
    [JsonPropertyName("source")] public string Source { get; set; } = string.Empty;
    [JsonPropertyName("target")] public string Target { get; set; } = string.Empty;
    [JsonPropertyName("sourceHandle")] public string? SourceHandle { get; set; }
    [JsonPropertyName("targetHandle")] public string? TargetHandle { get; set; }
    [JsonPropertyName("label")] public string? Label { get; set; }
    [JsonPropertyName("condition")] public string? Condition { get; set; }
}

public class FlowVariable
{
    [JsonPropertyName("name")] public string Name { get; set; } = string.Empty;
    [JsonPropertyName("type")] public string Type { get; set; } = string.Empty;
    [JsonPropertyName("defaultValue")] public object? DefaultValue { get; set; }
    [JsonPropertyName("description")] public string? Description { get; set; }
}

public class FlowExecution
{
    [JsonPropertyName("id")] public string Id { get; set; } = string.Empty;
    [JsonPropertyName("flowId")] public string FlowId { get; set; } = string.Empty;
    [JsonPropertyName("conversationId")] public string ConversationId { get; set; } = string.Empty;
    [JsonPropertyName("status")] public FlowExecutionStatus Status { get; set; }
    [JsonPropertyName("currentNodeId")] public string? CurrentNodeId { get; set; }
    [JsonPropertyName("variables")] public Dictionary<string, object>? Variables { get; set; }
    [JsonPropertyName("history")] public List<FlowExecutionStep> History { get; set; } = new();
    [JsonPropertyName("startedAt")] public DateTime StartedAt { get; set; }
    [JsonPropertyName("completedAt")] public DateTime? CompletedAt { get; set; }
    [JsonPropertyName("error")] public string? Error { get; set; }
}

public class FlowExecutionStep
{
    [JsonPropertyName("nodeId")] public string NodeId { get; set; } = string.Empty;
    [JsonPropertyName("nodeType")] public string NodeType { get; set; } = string.Empty;
    [JsonPropertyName("startedAt")] public DateTime StartedAt { get; set; }
    [JsonPropertyName("completedAt")] public DateTime? CompletedAt { get; set; }
    [JsonPropertyName("input")] public Dictionary<string, object>? Input { get; set; }
    [JsonPropertyName("output")] public Dictionary<string, object>? Output { get; set; }
    [JsonPropertyName("error")] public string? Error { get; set; }
}

public class CreateFlowInput
{
    [JsonPropertyName("name")] public string Name { get; set; } = string.Empty;
    [JsonPropertyName("description")] public string? Description { get; set; }
    [JsonPropertyName("nodes")] public List<FlowNode>? Nodes { get; set; }
    [JsonPropertyName("edges")] public List<FlowEdge>? Edges { get; set; }
    [JsonPropertyName("variables")] public List<FlowVariable>? Variables { get; set; }
    [JsonPropertyName("metadata")] public Dictionary<string, object>? Metadata { get; set; }
}

public class UpdateFlowInput
{
    [JsonPropertyName("name")] public string? Name { get; set; }
    [JsonPropertyName("description")] public string? Description { get; set; }
    [JsonPropertyName("status")] public FlowStatus? Status { get; set; }
    [JsonPropertyName("nodes")] public List<FlowNode>? Nodes { get; set; }
    [JsonPropertyName("edges")] public List<FlowEdge>? Edges { get; set; }
    [JsonPropertyName("variables")] public List<FlowVariable>? Variables { get; set; }
    [JsonPropertyName("metadata")] public Dictionary<string, object>? Metadata { get; set; }
}

public class ExecuteFlowInput
{
    [JsonPropertyName("conversationId")] public string ConversationId { get; set; } = string.Empty;
    [JsonPropertyName("variables")] public Dictionary<string, object>? Variables { get; set; }
}
