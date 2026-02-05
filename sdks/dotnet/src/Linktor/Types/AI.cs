using System.Text.Json.Serialization;

namespace Linktor.Types;

[JsonConverter(typeof(JsonStringEnumConverter))]
public enum AgentStatus { Active, Inactive, Draft }

public class Agent
{
    [JsonPropertyName("id")] public string Id { get; set; } = string.Empty;
    [JsonPropertyName("tenantId")] public string TenantId { get; set; } = string.Empty;
    [JsonPropertyName("name")] public string Name { get; set; } = string.Empty;
    [JsonPropertyName("description")] public string? Description { get; set; }
    [JsonPropertyName("status")] public AgentStatus Status { get; set; }
    [JsonPropertyName("model")] public string Model { get; set; } = string.Empty;
    [JsonPropertyName("systemPrompt")] public string? SystemPrompt { get; set; }
    [JsonPropertyName("temperature")] public double Temperature { get; set; }
    [JsonPropertyName("maxTokens")] public int MaxTokens { get; set; }
    [JsonPropertyName("knowledgeBaseIds")] public List<string> KnowledgeBaseIds { get; set; } = new();
    [JsonPropertyName("tools")] public List<Tool> Tools { get; set; } = new();
    [JsonPropertyName("metadata")] public Dictionary<string, object>? Metadata { get; set; }
    [JsonPropertyName("createdAt")] public DateTime CreatedAt { get; set; }
    [JsonPropertyName("updatedAt")] public DateTime UpdatedAt { get; set; }
}

public class Tool
{
    [JsonPropertyName("name")] public string Name { get; set; } = string.Empty;
    [JsonPropertyName("description")] public string? Description { get; set; }
    [JsonPropertyName("parameters")] public Dictionary<string, object>? Parameters { get; set; }
}

public class CreateAgentInput
{
    [JsonPropertyName("name")] public string Name { get; set; } = string.Empty;
    [JsonPropertyName("description")] public string? Description { get; set; }
    [JsonPropertyName("model")] public string? Model { get; set; }
    [JsonPropertyName("systemPrompt")] public string? SystemPrompt { get; set; }
    [JsonPropertyName("temperature")] public double? Temperature { get; set; }
    [JsonPropertyName("maxTokens")] public int? MaxTokens { get; set; }
    [JsonPropertyName("knowledgeBaseIds")] public List<string>? KnowledgeBaseIds { get; set; }
    [JsonPropertyName("tools")] public List<Tool>? Tools { get; set; }
    [JsonPropertyName("metadata")] public Dictionary<string, object>? Metadata { get; set; }
}

public class ChatMessage
{
    [JsonPropertyName("role")] public string Role { get; set; } = string.Empty;
    [JsonPropertyName("content")] public string Content { get; set; } = string.Empty;

    public static ChatMessage User(string content) => new() { Role = "user", Content = content };
    public static ChatMessage Assistant(string content) => new() { Role = "assistant", Content = content };
    public static ChatMessage System(string content) => new() { Role = "system", Content = content };
}

public class CompletionInput
{
    [JsonPropertyName("messages")] public List<ChatMessage> Messages { get; set; } = new();
    [JsonPropertyName("model")] public string? Model { get; set; }
    [JsonPropertyName("temperature")] public double? Temperature { get; set; }
    [JsonPropertyName("maxTokens")] public int? MaxTokens { get; set; }
    [JsonPropertyName("stream")] public bool Stream { get; set; }
    [JsonPropertyName("tools")] public List<Tool>? Tools { get; set; }
    [JsonPropertyName("metadata")] public Dictionary<string, object>? Metadata { get; set; }
}

public class CompletionResponse
{
    [JsonPropertyName("id")] public string Id { get; set; } = string.Empty;
    [JsonPropertyName("object")] public string Object { get; set; } = string.Empty;
    [JsonPropertyName("created")] public long Created { get; set; }
    [JsonPropertyName("model")] public string Model { get; set; } = string.Empty;
    [JsonPropertyName("choices")] public List<Choice> Choices { get; set; } = new();
    [JsonPropertyName("usage")] public Usage? Usage { get; set; }

    public string? Content => Choices.FirstOrDefault()?.Message?.Content;
}

public class Choice
{
    [JsonPropertyName("index")] public int Index { get; set; }
    [JsonPropertyName("message")] public ChatMessage? Message { get; set; }
    [JsonPropertyName("finishReason")] public string? FinishReason { get; set; }
}

public class Usage
{
    [JsonPropertyName("promptTokens")] public int PromptTokens { get; set; }
    [JsonPropertyName("completionTokens")] public int CompletionTokens { get; set; }
    [JsonPropertyName("totalTokens")] public int TotalTokens { get; set; }
}

public class EmbeddingInput
{
    [JsonPropertyName("input")] public string? Input { get; set; }
    [JsonPropertyName("inputs")] public List<string>? Inputs { get; set; }
    [JsonPropertyName("model")] public string? Model { get; set; }

    public static EmbeddingInput Single(string text) => new() { Input = text };
    public static EmbeddingInput Batch(List<string> texts) => new() { Inputs = texts };
}

public class EmbeddingResponse
{
    [JsonPropertyName("object")] public string Object { get; set; } = string.Empty;
    [JsonPropertyName("data")] public List<EmbeddingData> Data { get; set; } = new();
    [JsonPropertyName("model")] public string Model { get; set; } = string.Empty;
    [JsonPropertyName("usage")] public Usage? Usage { get; set; }

    public double[]? Embedding => Data.FirstOrDefault()?.Embedding;
}

public class EmbeddingData
{
    [JsonPropertyName("object")] public string Object { get; set; } = string.Empty;
    [JsonPropertyName("index")] public int Index { get; set; }
    [JsonPropertyName("embedding")] public double[] Embedding { get; set; } = Array.Empty<double>();
}
