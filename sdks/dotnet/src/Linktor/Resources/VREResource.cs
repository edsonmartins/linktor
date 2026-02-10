using System.Text.Json.Serialization;
using Linktor.Types;

namespace Linktor.Resources;

/// <summary>
/// VRE (Visual Response Engine) Resource
/// Render visual templates as images for messaging channels.
/// </summary>
public class VREResource
{
    private readonly LinktorClient _client;

    public VREResource(LinktorClient client) => _client = client;

    /// <summary>
    /// Render a VRE template to an image.
    /// Returns base64-encoded image data that can be sent to messaging channels.
    /// </summary>
    public Task<VRERenderResponse> RenderAsync(VRERenderRequest request, CancellationToken ct = default)
        => _client.PostAsync<VRERenderResponse>("/vre/render", request, ct);

    /// <summary>
    /// Render a VRE template and send it directly to a conversation.
    /// </summary>
    public Task<VRERenderAndSendResponse> RenderAndSendAsync(VRERenderAndSendRequest request, CancellationToken ct = default)
        => _client.PostAsync<VRERenderAndSendResponse>("/vre/render-and-send", request, ct);

    /// <summary>
    /// List available VRE templates.
    /// </summary>
    public Task<VREListTemplatesResponse> ListTemplatesAsync(string? tenantId = null, CancellationToken ct = default)
    {
        var path = tenantId != null
            ? $"/vre/templates?tenant_id={Uri.EscapeDataString(tenantId)}"
            : "/vre/templates";
        return _client.GetAsync<VREListTemplatesResponse>(path, ct);
    }

    /// <summary>
    /// Preview a VRE template with sample data.
    /// </summary>
    public Task<VREPreviewResponse> PreviewAsync(string templateId, Dictionary<string, object>? data = null, CancellationToken ct = default)
        => _client.PostAsync<VREPreviewResponse>($"/vre/templates/{templateId}/preview", data != null ? new { data } : new { }, ct);

    // ============================================
    // Convenience methods for common templates
    // ============================================

    /// <summary>
    /// Render a menu with numbered options.
    /// </summary>
    public Task<VRERenderResponse> RenderMenuAsync(
        string tenantId,
        string titulo,
        List<MenuOpcaoData> opcoes,
        string channel = "whatsapp",
        CancellationToken ct = default)
    {
        var request = new VRERenderRequest
        {
            TenantId = tenantId,
            TemplateId = "menu_opcoes",
            Data = new Dictionary<string, object>
            {
                ["titulo"] = titulo,
                ["opcoes"] = opcoes
            },
            Channel = channel
        };
        return RenderAsync(request, ct);
    }

    /// <summary>
    /// Render a product card.
    /// </summary>
    public Task<VRERenderResponse> RenderProductCardAsync(
        string tenantId,
        string nome,
        double preco,
        string unidade,
        string channel = "whatsapp",
        CancellationToken ct = default)
    {
        var request = new VRERenderRequest
        {
            TenantId = tenantId,
            TemplateId = "card_produto",
            Data = new Dictionary<string, object>
            {
                ["nome"] = nome,
                ["preco"] = preco,
                ["unidade"] = unidade
            },
            Channel = channel
        };
        return RenderAsync(request, ct);
    }

    /// <summary>
    /// Render an order status timeline.
    /// </summary>
    public Task<VRERenderResponse> RenderOrderStatusAsync(
        string tenantId,
        string numeroPedido,
        string statusAtual,
        string channel = "whatsapp",
        CancellationToken ct = default)
    {
        var request = new VRERenderRequest
        {
            TenantId = tenantId,
            TemplateId = "status_pedido",
            Data = new Dictionary<string, object>
            {
                ["numero_pedido"] = numeroPedido,
                ["status_atual"] = statusAtual
            },
            Channel = channel
        };
        return RenderAsync(request, ct);
    }

    /// <summary>
    /// Render a product list for comparison.
    /// </summary>
    public Task<VRERenderResponse> RenderProductListAsync(
        string tenantId,
        string titulo,
        List<ListaProdutoItem> produtos,
        string channel = "whatsapp",
        CancellationToken ct = default)
    {
        var request = new VRERenderRequest
        {
            TenantId = tenantId,
            TemplateId = "lista_produtos",
            Data = new Dictionary<string, object>
            {
                ["titulo"] = titulo,
                ["produtos"] = produtos
            },
            Channel = channel
        };
        return RenderAsync(request, ct);
    }

    /// <summary>
    /// Render a confirmation summary.
    /// </summary>
    public Task<VRERenderResponse> RenderConfirmationAsync(
        string tenantId,
        double valorTotal,
        List<ConfirmacaoItem> itens,
        string channel = "whatsapp",
        CancellationToken ct = default)
    {
        var request = new VRERenderRequest
        {
            TenantId = tenantId,
            TemplateId = "confirmacao",
            Data = new Dictionary<string, object>
            {
                ["valor_total"] = valorTotal,
                ["itens"] = itens
            },
            Channel = channel
        };
        return RenderAsync(request, ct);
    }

    /// <summary>
    /// Render a PIX payment QR code.
    /// </summary>
    public Task<VRERenderResponse> RenderPixPaymentAsync(
        string tenantId,
        double valor,
        string pixPayload,
        string channel = "whatsapp",
        CancellationToken ct = default)
    {
        var request = new VRERenderRequest
        {
            TenantId = tenantId,
            TemplateId = "cobranca_pix",
            Data = new Dictionary<string, object>
            {
                ["valor"] = valor,
                ["pix_payload"] = pixPayload
            },
            Channel = channel
        };
        return RenderAsync(request, ct);
    }
}

// ============================================
// Request/Input types
// ============================================

public class VRERenderRequest
{
    [JsonPropertyName("tenant_id")]
    public string TenantId { get; set; } = string.Empty;

    [JsonPropertyName("template_id")]
    public string TemplateId { get; set; } = string.Empty;

    [JsonPropertyName("data")]
    public Dictionary<string, object> Data { get; set; } = new();

    [JsonPropertyName("channel")]
    public string? Channel { get; set; }

    [JsonPropertyName("format")]
    public string? Format { get; set; }

    [JsonPropertyName("width")]
    public int? Width { get; set; }

    [JsonPropertyName("quality")]
    public int? Quality { get; set; }

    [JsonPropertyName("scale")]
    public double? Scale { get; set; }
}

public class VRERenderAndSendRequest
{
    [JsonPropertyName("conversation_id")]
    public string ConversationId { get; set; } = string.Empty;

    [JsonPropertyName("template_id")]
    public string TemplateId { get; set; } = string.Empty;

    [JsonPropertyName("data")]
    public Dictionary<string, object> Data { get; set; } = new();

    [JsonPropertyName("caption")]
    public string? Caption { get; set; }

    [JsonPropertyName("follow_up_text")]
    public string? FollowUpText { get; set; }
}

// ============================================
// Template data types
// ============================================

public class MenuOpcaoData
{
    [JsonPropertyName("label")]
    public string Label { get; set; } = string.Empty;

    [JsonPropertyName("descricao")]
    public string? Descricao { get; set; }

    [JsonPropertyName("icone")]
    public string? Icone { get; set; }
}

public class ListaProdutoItem
{
    [JsonPropertyName("nome")]
    public string Nome { get; set; } = string.Empty;

    [JsonPropertyName("preco")]
    public double Preco { get; set; }

    [JsonPropertyName("unidade")]
    public string? Unidade { get; set; }

    [JsonPropertyName("estoque_status")]
    public string? EstoqueStatus { get; set; }

    [JsonPropertyName("sku")]
    public string? Sku { get; set; }

    [JsonPropertyName("emoji")]
    public string? Emoji { get; set; }
}

public class ConfirmacaoItem
{
    [JsonPropertyName("nome")]
    public string Nome { get; set; } = string.Empty;

    [JsonPropertyName("quantidade")]
    public string? Quantidade { get; set; }

    [JsonPropertyName("preco")]
    public double Preco { get; set; }

    [JsonPropertyName("emoji")]
    public string? Emoji { get; set; }
}
