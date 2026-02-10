/**
 * VRE (Visual Response Engine) Resource
 *
 * Render visual templates as images for messaging channels.
 */

import type { HttpClient } from '../utils/http';
import type {
  VRERenderRequest,
  VRERenderResponse,
  VRERenderAndSendRequest,
  VRERenderAndSendResponse,
  VREListTemplatesResponse,
  VREPreviewRequest,
  VREPreviewResponse,
  VRETemplateType,
} from '../types/vre';

export class VREResource {
  constructor(private http: HttpClient) {}

  /**
   * Render a VRE template to an image.
   * Returns base64-encoded image data that can be sent to messaging channels.
   *
   * @example
   * ```typescript
   * const result = await client.vre.render({
   *   tenantId: 'my-tenant',
   *   templateId: 'menu_opcoes',
   *   data: {
   *     titulo: 'Como posso ajudar?',
   *     opcoes: [
   *       { label: 'Ver pedidos', icone: 'pedido' },
   *       { label: 'Catálogo', icone: 'catalogo' }
   *     ]
   *   },
   *   channel: 'whatsapp'
   * });
   * console.log(result.imageBase64); // data:image/webp;base64,...
   * console.log(result.caption); // Como posso ajudar?\n\n1️⃣ Ver pedidos...
   * ```
   */
  async render(request: VRERenderRequest): Promise<VRERenderResponse> {
    return this.http.post<VRERenderResponse>('/vre/render', {
      tenant_id: request.tenantId,
      template_id: request.templateId,
      data: request.data,
      channel: request.channel,
      format: request.format,
      width: request.width,
      quality: request.quality,
      scale: request.scale,
    });
  }

  /**
   * Render a VRE template and send it directly to a conversation.
   * Combines rendering and sending in one operation.
   *
   * @example
   * ```typescript
   * const result = await client.vre.renderAndSend({
   *   conversationId: 'conv-123',
   *   templateId: 'card_produto',
   *   data: {
   *     nome: 'Camarão Rosa',
   *     preco: 89.90,
   *     unidade: 'kg',
   *     estoque: 150
   *   },
   *   followUpText: 'Temos sim! Quer adicionar ao pedido?'
   * });
   * console.log(result.messageId); // msg-456
   * ```
   */
  async renderAndSend(request: VRERenderAndSendRequest): Promise<VRERenderAndSendResponse> {
    return this.http.post<VRERenderAndSendResponse>('/vre/render-and-send', {
      conversation_id: request.conversationId,
      template_id: request.templateId,
      data: request.data,
      caption: request.caption,
      follow_up_text: request.followUpText,
    });
  }

  /**
   * List available VRE templates with their schemas and example data.
   *
   * @param tenantId - Optional tenant ID to include custom templates
   *
   * @example
   * ```typescript
   * const templates = await client.vre.listTemplates();
   * for (const template of templates.templates) {
   *   console.log(`${template.id}: ${template.description}`);
   * }
   * ```
   */
  async listTemplates(tenantId?: string): Promise<VREListTemplatesResponse> {
    return this.http.get<VREListTemplatesResponse>('/vre/templates', {
      params: tenantId ? { tenant_id: tenantId } : undefined,
    });
  }

  /**
   * Preview a VRE template with sample data.
   * Returns the rendered image for inspection without sending.
   *
   * @example
   * ```typescript
   * const preview = await client.vre.preview({
   *   templateId: 'status_pedido',
   *   data: {
   *     numeroPedido: '#12345',
   *     statusAtual: 'transporte',
   *     valorTotal: 299.90
   *   }
   * });
   * // Display preview.imageBase64 in UI
   * ```
   */
  async preview(request: VREPreviewRequest): Promise<VREPreviewResponse> {
    return this.http.post<VREPreviewResponse>(
      `/vre/templates/${request.templateId}/preview`,
      request.data ? { data: request.data } : undefined
    );
  }

  // ============================================
  // Convenience methods for common templates
  // ============================================

  /**
   * Render a menu with numbered options.
   * Shortcut for render with template_id='menu_opcoes'.
   *
   * @example
   * ```typescript
   * const result = await client.vre.renderMenu({
   *   tenantId: 'my-tenant',
   *   titulo: 'Como posso ajudar?',
   *   opcoes: [
   *     { label: 'Ver pedidos', icone: 'pedido' },
   *     { label: 'Catálogo', icone: 'catalogo' },
   *     { label: 'Falar com atendente', icone: 'atendente' }
   *   ]
   * });
   * ```
   */
  async renderMenu(params: {
    tenantId: string;
    titulo: string;
    subtitulo?: string;
    opcoes: Array<{ label: string; icone?: string; descricao?: string }>;
    channel?: 'whatsapp' | 'telegram' | 'web' | 'email';
  }): Promise<VRERenderResponse> {
    return this.render({
      tenantId: params.tenantId,
      templateId: 'menu_opcoes',
      data: {
        titulo: params.titulo,
        subtitulo: params.subtitulo,
        opcoes: params.opcoes,
      },
      channel: params.channel,
    });
  }

  /**
   * Render a product card.
   * Shortcut for render with template_id='card_produto'.
   *
   * @example
   * ```typescript
   * const result = await client.vre.renderProductCard({
   *   tenantId: 'my-tenant',
   *   nome: 'Camarão Rosa',
   *   preco: 89.90,
   *   unidade: 'kg',
   *   imagemUrl: 'https://...'
   * });
   * ```
   */
  async renderProductCard(params: {
    tenantId: string;
    nome: string;
    preco: number;
    unidade: string;
    sku?: string;
    estoque?: number;
    imagemUrl?: string;
    destaque?: string;
    mensagem?: string;
    channel?: 'whatsapp' | 'telegram' | 'web' | 'email';
  }): Promise<VRERenderResponse> {
    return this.render({
      tenantId: params.tenantId,
      templateId: 'card_produto',
      data: {
        nome: params.nome,
        preco: params.preco,
        unidade: params.unidade,
        sku: params.sku,
        estoque: params.estoque,
        imagem_url: params.imagemUrl,
        destaque: params.destaque,
        mensagem: params.mensagem,
      },
      channel: params.channel,
    });
  }

  /**
   * Render an order status timeline.
   * Shortcut for render with template_id='status_pedido'.
   *
   * @example
   * ```typescript
   * const result = await client.vre.renderOrderStatus({
   *   tenantId: 'my-tenant',
   *   numeroPedido: '#12345',
   *   statusAtual: 'transporte',
   *   valorTotal: 299.90,
   *   previsaoEntrega: 'Amanhã, 10-12h'
   * });
   * ```
   */
  async renderOrderStatus(params: {
    tenantId: string;
    numeroPedido: string;
    statusAtual: 'recebido' | 'separacao' | 'faturado' | 'transporte' | 'entregue';
    itensResumo?: string;
    valorTotal?: number;
    previsaoEntrega?: string;
    motorista?: string;
    mensagem?: string;
    channel?: 'whatsapp' | 'telegram' | 'web' | 'email';
  }): Promise<VRERenderResponse> {
    return this.render({
      tenantId: params.tenantId,
      templateId: 'status_pedido',
      data: {
        numero_pedido: params.numeroPedido,
        status_atual: params.statusAtual,
        itens_resumo: params.itensResumo,
        valor_total: params.valorTotal,
        previsao_entrega: params.previsaoEntrega,
        motorista: params.motorista,
        mensagem: params.mensagem,
      },
      channel: params.channel,
    });
  }

  /**
   * Render a product list for comparison.
   * Shortcut for render with template_id='lista_produtos'.
   *
   * @example
   * ```typescript
   * const result = await client.vre.renderProductList({
   *   tenantId: 'my-tenant',
   *   titulo: 'Pescados Disponíveis',
   *   produtos: [
   *     { nome: 'Camarão Rosa', preco: 89.90, unidade: 'kg' },
   *     { nome: 'Salmão', preco: 79.90, unidade: 'kg' }
   *   ]
   * });
   * ```
   */
  async renderProductList(params: {
    tenantId: string;
    titulo: string;
    produtos: Array<{
      nome: string;
      preco: number;
      unidade?: string;
      estoqueStatus?: 'disponivel' | 'baixo' | 'indisponivel';
      sku?: string;
      emoji?: string;
    }>;
    mensagem?: string;
    channel?: 'whatsapp' | 'telegram' | 'web' | 'email';
  }): Promise<VRERenderResponse> {
    return this.render({
      tenantId: params.tenantId,
      templateId: 'lista_produtos',
      data: {
        titulo: params.titulo,
        produtos: params.produtos.map((p) => ({
          nome: p.nome,
          preco: p.preco,
          unidade: p.unidade,
          estoque_status: p.estoqueStatus,
          sku: p.sku,
          emoji: p.emoji,
        })),
        mensagem: params.mensagem,
      },
      channel: params.channel,
    });
  }

  /**
   * Render a confirmation summary.
   * Shortcut for render with template_id='confirmacao'.
   *
   * @example
   * ```typescript
   * const result = await client.vre.renderConfirmation({
   *   tenantId: 'my-tenant',
   *   titulo: 'Confirme seu Pedido',
   *   itens: [
   *     { nome: 'Camarão Rosa', quantidade: '2kg', preco: 179.80 },
   *     { nome: 'Salmão', quantidade: '1kg', preco: 79.90 }
   *   ],
   *   valorTotal: 259.70,
   *   previsaoEntrega: 'Amanhã, 10-12h'
   * });
   * ```
   */
  async renderConfirmation(params: {
    tenantId: string;
    titulo?: string;
    subtitulo?: string;
    itens: Array<{
      nome: string;
      quantidade?: string;
      preco: number;
      emoji?: string;
    }>;
    valorTotal: number;
    previsaoEntrega?: string;
    mensagem?: string;
    channel?: 'whatsapp' | 'telegram' | 'web' | 'email';
  }): Promise<VRERenderResponse> {
    return this.render({
      tenantId: params.tenantId,
      templateId: 'confirmacao',
      data: {
        titulo: params.titulo,
        subtitulo: params.subtitulo,
        itens: params.itens,
        valor_total: params.valorTotal,
        previsao_entrega: params.previsaoEntrega,
        mensagem: params.mensagem,
      },
      channel: params.channel,
    });
  }

  /**
   * Render a PIX payment QR code.
   * Shortcut for render with template_id='cobranca_pix'.
   *
   * @example
   * ```typescript
   * const result = await client.vre.renderPixPayment({
   *   tenantId: 'my-tenant',
   *   valor: 259.70,
   *   pixPayload: '00020126580014br.gov.bcb.pix...',
   *   numeroPedido: '#12345',
   *   expiracao: '30 minutos'
   * });
   * ```
   */
  async renderPixPayment(params: {
    tenantId: string;
    valor: number;
    pixPayload: string;
    numeroPedido?: string;
    expiracao?: string;
    mensagem?: string;
    channel?: 'whatsapp' | 'telegram' | 'web' | 'email';
  }): Promise<VRERenderResponse> {
    return this.render({
      tenantId: params.tenantId,
      templateId: 'cobranca_pix',
      data: {
        valor: params.valor,
        pix_payload: params.pixPayload,
        numero_pedido: params.numeroPedido,
        expiracao: params.expiracao,
        mensagem: params.mensagem,
      },
      channel: params.channel,
    });
  }
}
