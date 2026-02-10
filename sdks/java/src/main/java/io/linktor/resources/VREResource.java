package io.linktor.resources;

import io.linktor.types.VRE;
import io.linktor.utils.HttpClient;
import io.linktor.utils.LinktorException;

import java.util.HashMap;
import java.util.Map;

/**
 * VRE (Visual Response Engine) Resource
 *
 * Render visual templates as images for messaging channels.
 */
public class VREResource {
    private final HttpClient http;

    public VREResource(HttpClient http) {
        this.http = http;
    }

    /**
     * Render a VRE template to an image.
     * Returns base64-encoded image data that can be sent to messaging channels.
     */
    public VRE.RenderResponse render(VRE.RenderRequest request) throws LinktorException {
        return http.post("/vre/render", request, VRE.RenderResponse.class);
    }

    /**
     * Render a VRE template and send it directly to a conversation.
     * Combines rendering and sending in one operation.
     */
    public VRE.RenderAndSendResponse renderAndSend(VRE.RenderAndSendRequest request) throws LinktorException {
        return http.post("/vre/render-and-send", request, VRE.RenderAndSendResponse.class);
    }

    /**
     * List available VRE templates with their schemas and example data.
     */
    public VRE.ListTemplatesResponse listTemplates() throws LinktorException {
        return listTemplates(null);
    }

    /**
     * List available VRE templates with their schemas and example data.
     *
     * @param tenantId Optional tenant ID to include custom templates
     */
    public VRE.ListTemplatesResponse listTemplates(String tenantId) throws LinktorException {
        Map<String, String> queryParams = new HashMap<>();
        if (tenantId != null && !tenantId.isEmpty()) {
            queryParams.put("tenant_id", tenantId);
        }
        return http.get("/vre/templates", queryParams, VRE.ListTemplatesResponse.class);
    }

    /**
     * Preview a VRE template with sample data.
     *
     * @param templateId Template to preview
     * @param data Optional custom data (uses default sample data if not provided)
     */
    public VRE.PreviewResponse preview(String templateId, Map<String, Object> data) throws LinktorException {
        VRE.PreviewRequest request = new VRE.PreviewRequest(data);
        return http.post("/vre/templates/" + templateId + "/preview", request, VRE.PreviewResponse.class);
    }

    /**
     * Preview a VRE template with default sample data.
     */
    public VRE.PreviewResponse preview(String templateId) throws LinktorException {
        return preview(templateId, null);
    }

    // ============================================
    // Convenience methods for common templates
    // ============================================

    /**
     * Render a menu with numbered options.
     *
     * @param tenantId Tenant ID
     * @param titulo Menu title
     * @param opcoes List of options (label, icone, descricao)
     * @param channel Target channel
     */
    public VRE.RenderResponse renderMenu(String tenantId, String titulo, java.util.List<Map<String, Object>> opcoes, VRE.ChannelType channel) throws LinktorException {
        Map<String, Object> data = new HashMap<>();
        data.put("titulo", titulo);
        data.put("opcoes", opcoes);

        return render(VRE.RenderRequest.builder()
                .tenantId(tenantId)
                .templateId(VRE.TemplateType.MENU_OPCOES)
                .data(data)
                .channel(channel)
                .build());
    }

    /**
     * Render a product card.
     *
     * @param tenantId Tenant ID
     * @param nome Product name
     * @param preco Product price
     * @param unidade Unit (kg, un, cx, etc.)
     * @param channel Target channel
     */
    public VRE.RenderResponse renderProductCard(String tenantId, String nome, double preco, String unidade, VRE.ChannelType channel) throws LinktorException {
        Map<String, Object> data = new HashMap<>();
        data.put("nome", nome);
        data.put("preco", preco);
        data.put("unidade", unidade);

        return render(VRE.RenderRequest.builder()
                .tenantId(tenantId)
                .templateId(VRE.TemplateType.CARD_PRODUTO)
                .data(data)
                .channel(channel)
                .build());
    }

    /**
     * Render an order status timeline.
     *
     * @param tenantId Tenant ID
     * @param numeroPedido Order number
     * @param statusAtual Current status
     * @param channel Target channel
     */
    public VRE.RenderResponse renderOrderStatus(String tenantId, String numeroPedido, VRE.OrderStatus statusAtual, VRE.ChannelType channel) throws LinktorException {
        Map<String, Object> data = new HashMap<>();
        data.put("numero_pedido", numeroPedido);
        data.put("status_atual", statusAtual.name().toLowerCase());

        return render(VRE.RenderRequest.builder()
                .tenantId(tenantId)
                .templateId(VRE.TemplateType.STATUS_PEDIDO)
                .data(data)
                .channel(channel)
                .build());
    }

    /**
     * Render a product list for comparison.
     *
     * @param tenantId Tenant ID
     * @param titulo List title
     * @param produtos List of products
     * @param channel Target channel
     */
    public VRE.RenderResponse renderProductList(String tenantId, String titulo, java.util.List<Map<String, Object>> produtos, VRE.ChannelType channel) throws LinktorException {
        Map<String, Object> data = new HashMap<>();
        data.put("titulo", titulo);
        data.put("produtos", produtos);

        return render(VRE.RenderRequest.builder()
                .tenantId(tenantId)
                .templateId(VRE.TemplateType.LISTA_PRODUTOS)
                .data(data)
                .channel(channel)
                .build());
    }

    /**
     * Render a confirmation summary.
     *
     * @param tenantId Tenant ID
     * @param valorTotal Total value
     * @param itens List of items
     * @param channel Target channel
     */
    public VRE.RenderResponse renderConfirmation(String tenantId, double valorTotal, java.util.List<Map<String, Object>> itens, VRE.ChannelType channel) throws LinktorException {
        Map<String, Object> data = new HashMap<>();
        data.put("valor_total", valorTotal);
        data.put("itens", itens);

        return render(VRE.RenderRequest.builder()
                .tenantId(tenantId)
                .templateId(VRE.TemplateType.CONFIRMACAO)
                .data(data)
                .channel(channel)
                .build());
    }

    /**
     * Render a PIX payment QR code.
     *
     * @param tenantId Tenant ID
     * @param valor Payment amount
     * @param pixPayload PIX EMV/BRCode payload
     * @param channel Target channel
     */
    public VRE.RenderResponse renderPixPayment(String tenantId, double valor, String pixPayload, VRE.ChannelType channel) throws LinktorException {
        Map<String, Object> data = new HashMap<>();
        data.put("valor", valor);
        data.put("pix_payload", pixPayload);

        return render(VRE.RenderRequest.builder()
                .tenantId(tenantId)
                .templateId(VRE.TemplateType.COBRANCA_PIX)
                .data(data)
                .channel(channel)
                .build());
    }
}
