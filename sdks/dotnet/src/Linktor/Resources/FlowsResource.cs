using Linktor.Types;

namespace Linktor.Resources;

public class FlowsResource
{
    private readonly LinktorClient _client;

    public FlowsResource(LinktorClient client) => _client = client;

    public Task<PaginatedResponse<Flow>> ListAsync(ListFlowsParams? parameters = null, CancellationToken ct = default)
    {
        var query = BuildQuery(parameters);
        return _client.GetAsync<PaginatedResponse<Flow>>($"/flows{query}", ct);
    }

    public Task<Flow> GetAsync(string id, CancellationToken ct = default)
        => _client.GetAsync<Flow>($"/flows/{id}", ct);

    public Task<Flow> CreateAsync(CreateFlowInput input, CancellationToken ct = default)
        => _client.PostAsync<Flow>("/flows", input, ct);

    public Task<Flow> UpdateAsync(string id, UpdateFlowInput input, CancellationToken ct = default)
        => _client.PatchAsync<Flow>($"/flows/{id}", input, ct);

    public Task DeleteAsync(string id, CancellationToken ct = default)
        => _client.DeleteAsync($"/flows/{id}", ct);

    public Task<Flow> PublishAsync(string id, CancellationToken ct = default)
        => _client.PostAsync<Flow>($"/flows/{id}/publish", new { }, ct);

    public Task<Flow> UnpublishAsync(string id, CancellationToken ct = default)
        => _client.PostAsync<Flow>($"/flows/{id}/unpublish", new { }, ct);

    public Task<FlowExecution> ExecuteAsync(string id, FlowExecuteInput input, CancellationToken ct = default)
        => _client.PostAsync<FlowExecution>($"/flows/{id}/execute", input, ct);

    public Task<FlowValidation> ValidateAsync(string id, CancellationToken ct = default)
        => _client.PostAsync<FlowValidation>($"/flows/{id}/validate", new { }, ct);

    public Task<Flow> DuplicateAsync(string id, DuplicateFlowInput? input = null, CancellationToken ct = default)
        => _client.PostAsync<Flow>($"/flows/{id}/duplicate", input ?? new DuplicateFlowInput(), ct);

    public Task<List<FlowNodeType>> GetNodeTypesAsync(CancellationToken ct = default)
        => _client.GetAsync<List<FlowNodeType>>("/flows/node-types", ct);

    public Task<PaginatedResponse<FlowExecution>> GetExecutionsAsync(string flowId, ListExecutionsParams? parameters = null, CancellationToken ct = default)
    {
        var query = BuildExecQuery(parameters);
        return _client.GetAsync<PaginatedResponse<FlowExecution>>($"/flows/{flowId}/executions{query}", ct);
    }

    private static string BuildQuery(ListFlowsParams? p)
    {
        if (p == null) return "";
        var parts = new List<string>();
        if (p.Limit.HasValue) parts.Add($"limit={p.Limit}");
        if (p.Offset.HasValue) parts.Add($"offset={p.Offset}");
        if (!string.IsNullOrEmpty(p.Status)) parts.Add($"status={Uri.EscapeDataString(p.Status)}");
        return parts.Count > 0 ? "?" + string.Join("&", parts) : "";
    }

    private static string BuildExecQuery(ListExecutionsParams? p)
    {
        if (p == null) return "";
        var parts = new List<string>();
        if (p.Limit.HasValue) parts.Add($"limit={p.Limit}");
        if (p.Offset.HasValue) parts.Add($"offset={p.Offset}");
        if (!string.IsNullOrEmpty(p.Status)) parts.Add($"status={Uri.EscapeDataString(p.Status)}");
        return parts.Count > 0 ? "?" + string.Join("&", parts) : "";
    }
}

public class ListFlowsParams
{
    public int? Limit { get; set; }
    public int? Offset { get; set; }
    public string? Status { get; set; }
}

public class CreateFlowInput
{
    public string Name { get; set; } = string.Empty;
    public string? Description { get; set; }
    public List<FlowNode>? Nodes { get; set; }
    public List<FlowEdge>? Edges { get; set; }
    public FlowSettings? Settings { get; set; }
    public Dictionary<string, object>? Metadata { get; set; }
}

public class UpdateFlowInput
{
    public string? Name { get; set; }
    public string? Description { get; set; }
    public List<FlowNode>? Nodes { get; set; }
    public List<FlowEdge>? Edges { get; set; }
    public FlowSettings? Settings { get; set; }
    public Dictionary<string, object>? Metadata { get; set; }
}

public class FlowExecuteInput
{
    public string? ConversationId { get; set; }
    public string? ContactId { get; set; }
    public Dictionary<string, object>? Variables { get; set; }
    public string? TriggerNodeId { get; set; }
}

public class DuplicateFlowInput
{
    public string? Name { get; set; }
}

public class ListExecutionsParams
{
    public int? Limit { get; set; }
    public int? Offset { get; set; }
    public string? Status { get; set; }
}

public class FlowValidation
{
    public bool IsValid { get; set; }
    public List<FlowValidationError>? Errors { get; set; }
    public List<FlowValidationWarning>? Warnings { get; set; }
}

public class FlowValidationError
{
    public string NodeId { get; set; } = string.Empty;
    public string Code { get; set; } = string.Empty;
    public string Message { get; set; } = string.Empty;
}

public class FlowValidationWarning
{
    public string NodeId { get; set; } = string.Empty;
    public string Code { get; set; } = string.Empty;
    public string Message { get; set; } = string.Empty;
}

public class FlowNodeType
{
    public string Type { get; set; } = string.Empty;
    public string Name { get; set; } = string.Empty;
    public string? Description { get; set; }
    public string Category { get; set; } = string.Empty;
    public List<FlowNodePort>? Inputs { get; set; }
    public List<FlowNodePort>? Outputs { get; set; }
    public Dictionary<string, object>? Schema { get; set; }
}

public class FlowNodePort
{
    public string Name { get; set; } = string.Empty;
    public string Type { get; set; } = string.Empty;
    public bool Required { get; set; }
    public string? Description { get; set; }
}
