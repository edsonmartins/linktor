<?php

declare(strict_types=1);

namespace Linktor\Resources;

use Linktor\LinktorClient;
use Linktor\Types\VRERenderResponse;
use Linktor\Types\VRERenderAndSendResponse;
use Linktor\Types\VREListTemplatesResponse;
use Linktor\Types\VREPreviewResponse;

/**
 * VRE (Visual Response Engine) Resource
 *
 * Render visual templates as images for messaging channels.
 */
class VREResource
{
    private LinktorClient $client;

    public function __construct(LinktorClient $client)
    {
        $this->client = $client;
    }

    /**
     * Render a VRE template to an image.
     * Returns base64-encoded image data that can be sent to messaging channels.
     *
     * @param string $tenantId Tenant ID
     * @param string $templateId Template to render (menu_opcoes, card_produto, etc.)
     * @param array $data Template data
     * @param string|null $channel Target channel (whatsapp, telegram, web, email)
     * @param string|null $format Output format (png, webp, jpeg)
     * @param int|null $width Override width
     * @param int|null $quality Quality 0-100 (for webp/jpeg)
     * @param float|null $scale Scale factor (1.0-2.0)
     */
    public function render(
        string $tenantId,
        string $templateId,
        array $data,
        ?string $channel = null,
        ?string $format = null,
        ?int $width = null,
        ?int $quality = null,
        ?float $scale = null
    ): VRERenderResponse {
        $payload = [
            'tenant_id' => $tenantId,
            'template_id' => $templateId,
            'data' => $data,
        ];

        if ($channel !== null) $payload['channel'] = $channel;
        if ($format !== null) $payload['format'] = $format;
        if ($width !== null) $payload['width'] = $width;
        if ($quality !== null) $payload['quality'] = $quality;
        if ($scale !== null) $payload['scale'] = $scale;

        $response = $this->client->post('/vre/render', $payload);
        return VRERenderResponse::fromArray($response);
    }

    /**
     * Render a VRE template and send it directly to a conversation.
     *
     * @param string $conversationId Conversation ID to send to
     * @param string $templateId Template to render
     * @param array $data Template data
     * @param string|null $caption Optional custom caption
     * @param string|null $followUpText Optional text to send after image
     */
    public function renderAndSend(
        string $conversationId,
        string $templateId,
        array $data,
        ?string $caption = null,
        ?string $followUpText = null
    ): VRERenderAndSendResponse {
        $payload = [
            'conversation_id' => $conversationId,
            'template_id' => $templateId,
            'data' => $data,
        ];

        if ($caption !== null) $payload['caption'] = $caption;
        if ($followUpText !== null) $payload['follow_up_text'] = $followUpText;

        $response = $this->client->post('/vre/render-and-send', $payload);
        return VRERenderAndSendResponse::fromArray($response);
    }

    /**
     * List available VRE templates.
     *
     * @param string|null $tenantId Optional tenant ID to include custom templates
     */
    public function listTemplates(?string $tenantId = null): VREListTemplatesResponse
    {
        $path = '/vre/templates';
        if ($tenantId !== null) {
            $path .= '?tenant_id=' . urlencode($tenantId);
        }

        $response = $this->client->get($path);
        return VREListTemplatesResponse::fromArray($response);
    }

    /**
     * Preview a VRE template with sample data.
     *
     * @param string $templateId Template to preview
     * @param array|null $data Optional custom data (uses defaults if not provided)
     */
    public function preview(string $templateId, ?array $data = null): VREPreviewResponse
    {
        $payload = $data !== null ? ['data' => $data] : [];
        $response = $this->client->post("/vre/templates/{$templateId}/preview", $payload);
        return VREPreviewResponse::fromArray($response);
    }

    // ============================================
    // Convenience methods for common templates
    // ============================================

    /**
     * Render a menu with numbered options.
     *
     * @param string $tenantId Tenant ID
     * @param string $titulo Menu title
     * @param array $opcoes List of options (with label, icone, descricao)
     * @param string $channel Target channel
     */
    public function renderMenu(
        string $tenantId,
        string $titulo,
        array $opcoes,
        string $channel = 'whatsapp'
    ): VRERenderResponse {
        return $this->render($tenantId, 'menu_opcoes', [
            'titulo' => $titulo,
            'opcoes' => $opcoes,
        ], $channel);
    }

    /**
     * Render a product card.
     *
     * @param string $tenantId Tenant ID
     * @param string $nome Product name
     * @param float $preco Product price
     * @param string $unidade Unit (kg, un, cx, etc.)
     * @param string $channel Target channel
     */
    public function renderProductCard(
        string $tenantId,
        string $nome,
        float $preco,
        string $unidade,
        string $channel = 'whatsapp'
    ): VRERenderResponse {
        return $this->render($tenantId, 'card_produto', [
            'nome' => $nome,
            'preco' => $preco,
            'unidade' => $unidade,
        ], $channel);
    }

    /**
     * Render an order status timeline.
     *
     * @param string $tenantId Tenant ID
     * @param string $numeroPedido Order number
     * @param string $statusAtual Current status (recebido, separacao, faturado, transporte, entregue)
     * @param string $channel Target channel
     */
    public function renderOrderStatus(
        string $tenantId,
        string $numeroPedido,
        string $statusAtual,
        string $channel = 'whatsapp'
    ): VRERenderResponse {
        return $this->render($tenantId, 'status_pedido', [
            'numero_pedido' => $numeroPedido,
            'status_atual' => $statusAtual,
        ], $channel);
    }

    /**
     * Render a product list for comparison.
     *
     * @param string $tenantId Tenant ID
     * @param string $titulo List title
     * @param array $produtos List of products
     * @param string $channel Target channel
     */
    public function renderProductList(
        string $tenantId,
        string $titulo,
        array $produtos,
        string $channel = 'whatsapp'
    ): VRERenderResponse {
        return $this->render($tenantId, 'lista_produtos', [
            'titulo' => $titulo,
            'produtos' => $produtos,
        ], $channel);
    }

    /**
     * Render a confirmation summary.
     *
     * @param string $tenantId Tenant ID
     * @param float $valorTotal Total value
     * @param array $itens List of items
     * @param string $channel Target channel
     */
    public function renderConfirmation(
        string $tenantId,
        float $valorTotal,
        array $itens,
        string $channel = 'whatsapp'
    ): VRERenderResponse {
        return $this->render($tenantId, 'confirmacao', [
            'valor_total' => $valorTotal,
            'itens' => $itens,
        ], $channel);
    }

    /**
     * Render a PIX payment QR code.
     *
     * @param string $tenantId Tenant ID
     * @param float $valor Payment amount
     * @param string $pixPayload PIX EMV/BRCode payload
     * @param string $channel Target channel
     */
    public function renderPixPayment(
        string $tenantId,
        float $valor,
        string $pixPayload,
        string $channel = 'whatsapp'
    ): VRERenderResponse {
        return $this->render($tenantId, 'cobranca_pix', [
            'valor' => $valor,
            'pix_payload' => $pixPayload,
        ], $channel);
    }
}
