/**
 * VRE (Visual Response Engine) Types
 *
 * Types for rendering visual templates that can be sent to messaging channels.
 */

/**
 * Output format for rendered images
 */
export type VREOutputFormat = 'png' | 'webp' | 'jpeg';

/**
 * Channel type for VRE rendering (affects image optimization)
 */
export type VREChannelType = 'whatsapp' | 'telegram' | 'web' | 'email';

/**
 * Available template types
 */
export type VRETemplateType =
  | 'menu_opcoes'
  | 'card_produto'
  | 'status_pedido'
  | 'lista_produtos'
  | 'confirmacao'
  | 'cobranca_pix';

/**
 * Order status for status_pedido template
 */
export type OrderStatus =
  | 'recebido'
  | 'separacao'
  | 'faturado'
  | 'transporte'
  | 'entregue';

/**
 * Stock status for products
 */
export type StockStatus = 'disponivel' | 'baixo' | 'indisponivel';

// ============================================
// Render Request/Response
// ============================================

/**
 * Request to render a VRE template
 */
export interface VRERenderRequest {
  /**
   * Tenant ID
   */
  tenantId: string;

  /**
   * Template ID to render
   */
  templateId: VRETemplateType;

  /**
   * Template data (varies by template type)
   */
  data: Record<string, unknown>;

  /**
   * Target channel for optimized rendering
   * @default 'whatsapp'
   */
  channel?: VREChannelType;

  /**
   * Output image format
   * @default 'webp'
   */
  format?: VREOutputFormat;

  /**
   * Override default width
   */
  width?: number;

  /**
   * Quality (0-100, only for webp/jpeg)
   */
  quality?: number;

  /**
   * Scale factor (1.0-2.0)
   */
  scale?: number;
}

/**
 * Response from rendering a template
 */
export interface VRERenderResponse {
  /**
   * Base64-encoded image data
   */
  imageBase64: string;

  /**
   * Auto-generated caption for the image
   */
  caption: string;

  /**
   * Image width in pixels
   */
  width: number;

  /**
   * Image height in pixels
   */
  height: number;

  /**
   * Output format used
   */
  format: VREOutputFormat;

  /**
   * Render time in milliseconds
   */
  renderTimeMs: number;

  /**
   * Image size in bytes
   */
  sizeBytes?: number;

  /**
   * Whether the result was served from cache
   */
  cacheHit?: boolean;
}

// ============================================
// Render and Send
// ============================================

/**
 * Request to render a template and send it directly to a conversation
 */
export interface VRERenderAndSendRequest {
  /**
   * Conversation ID to send the rendered image to
   */
  conversationId: string;

  /**
   * Template ID to render
   */
  templateId: VRETemplateType;

  /**
   * Template data
   */
  data: Record<string, unknown>;

  /**
   * Optional custom caption (auto-generated if not provided)
   */
  caption?: string;

  /**
   * Optional text message to send after the image
   */
  followUpText?: string;
}

/**
 * Response from render and send
 */
export interface VRERenderAndSendResponse {
  /**
   * ID of the sent message
   */
  messageId: string;

  /**
   * URL of the uploaded image
   */
  imageUrl: string;

  /**
   * Caption used
   */
  caption: string;
}

// ============================================
// Templates
// ============================================

/**
 * VRE template definition
 */
export interface VRETemplate {
  /**
   * Template ID
   */
  id: VRETemplateType;

  /**
   * Human-readable name
   */
  name: string;

  /**
   * Template description
   */
  description: string;

  /**
   * JSON schema for template data
   */
  schema: Record<string, unknown>;
}

/**
 * Response from listing templates
 */
export interface VREListTemplatesResponse {
  templates: VRETemplate[];
}

/**
 * Request to preview a template
 */
export interface VREPreviewRequest {
  /**
   * Template ID to preview
   */
  templateId: VRETemplateType;

  /**
   * Optional custom data (uses default sample data if not provided)
   */
  data?: Record<string, unknown>;
}

/**
 * Response from template preview
 */
export interface VREPreviewResponse {
  /**
   * Base64-encoded image data
   */
  imageBase64: string;

  /**
   * Image width
   */
  width: number;

  /**
   * Image height
   */
  height: number;
}

// ============================================
// Template Data Types
// ============================================

/**
 * Menu option for menu_opcoes template
 */
export interface MenuOpcaoData {
  /**
   * Option label text
   */
  label: string;

  /**
   * Optional description
   */
  descricao?: string;

  /**
   * Icon name (pedido, catalogo, entrega, financeiro, atendente, etc.)
   */
  icone?: string;
}

/**
 * Data for menu_opcoes template
 */
export interface MenuOpcoesData {
  /**
   * Menu title
   */
  titulo: string;

  /**
   * Optional subtitle
   */
  subtitulo?: string;

  /**
   * Menu options (max 8)
   */
  opcoes: MenuOpcaoData[];

  /**
   * Optional message to show before options
   */
  mensagemAntes?: string;
}

/**
 * Data for card_produto template
 */
export interface CardProdutoData {
  /**
   * Product name
   */
  nome: string;

  /**
   * Product SKU
   */
  sku?: string;

  /**
   * Product price
   */
  preco: number;

  /**
   * Unit (kg, un, cx, fd, pc)
   */
  unidade: string;

  /**
   * Stock quantity
   */
  estoque?: number;

  /**
   * Product image URL
   */
  imagemUrl?: string;

  /**
   * Highlight badge (promocao, novo, mais vendido)
   */
  destaque?: string;

  /**
   * Additional message
   */
  mensagem?: string;
}

/**
 * Timeline step for status_pedido template
 */
export interface StatusPedidoStep {
  /**
   * Step status (done, active, wait)
   */
  status: 'done' | 'active' | 'wait';

  /**
   * Step icon
   */
  icon: string;

  /**
   * Step label
   */
  label: string;
}

/**
 * Data for status_pedido template
 */
export interface StatusPedidoData {
  /**
   * Order number
   */
  numeroPedido: string;

  /**
   * Current order status
   */
  statusAtual: OrderStatus;

  /**
   * Items summary
   */
  itensResumo?: string;

  /**
   * Total order value
   */
  valorTotal?: number;

  /**
   * Delivery estimate
   */
  previsaoEntrega?: string;

  /**
   * Driver name
   */
  motorista?: string;

  /**
   * Additional message
   */
  mensagem?: string;
}

/**
 * Product item for lista_produtos template
 */
export interface ListaProdutoItem {
  /**
   * Product name
   */
  nome: string;

  /**
   * Product price
   */
  preco: number;

  /**
   * Unit
   */
  unidade?: string;

  /**
   * Stock status
   */
  estoqueStatus?: StockStatus;

  /**
   * Product SKU
   */
  sku?: string;

  /**
   * Emoji for the product
   */
  emoji?: string;
}

/**
 * Data for lista_produtos template
 */
export interface ListaProdutosData {
  /**
   * List title
   */
  titulo: string;

  /**
   * Products (max 6)
   */
  produtos: ListaProdutoItem[];

  /**
   * Additional message
   */
  mensagem?: string;
}

/**
 * Confirmation item for confirmacao template
 */
export interface ConfirmacaoItem {
  /**
   * Item name
   */
  nome: string;

  /**
   * Quantity description
   */
  quantidade?: string;

  /**
   * Item price
   */
  preco: number;

  /**
   * Emoji
   */
  emoji?: string;
}

/**
 * Data for confirmacao template
 */
export interface ConfirmacaoData {
  /**
   * Confirmation title
   */
  titulo?: string;

  /**
   * Subtitle
   */
  subtitulo?: string;

  /**
   * Items
   */
  itens: ConfirmacaoItem[];

  /**
   * Total value
   */
  valorTotal: number;

  /**
   * Delivery estimate
   */
  previsaoEntrega?: string;

  /**
   * Additional message
   */
  mensagem?: string;
}

/**
 * Data for cobranca_pix template
 */
export interface CobrancaPixData {
  /**
   * Payment amount
   */
  valor: number;

  /**
   * Order number
   */
  numeroPedido?: string;

  /**
   * PIX EMV/BRCode payload
   */
  pixPayload: string;

  /**
   * Expiration time (e.g., "30 minutos")
   */
  expiracao?: string;

  /**
   * Additional message
   */
  mensagem?: string;
}
