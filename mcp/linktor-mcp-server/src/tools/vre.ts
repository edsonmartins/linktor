// ============================================
// VRE (Visual Response Engine) Tools
// ============================================

import type { Tool } from '@modelcontextprotocol/sdk/types.js';
import type { LinktorClient } from '../api/client.js';

export const vreToolDefinitions: Tool[] = [
  {
    name: 'render_template',
    description: 'Render a VRE template to an image. Returns base64-encoded image data that can be sent to messaging channels.',
    inputSchema: {
      type: 'object',
      properties: {
        tenant_id: {
          type: 'string',
          description: 'The tenant ID',
        },
        template_id: {
          type: 'string',
          enum: ['menu_opcoes', 'card_produto', 'status_pedido', 'lista_produtos', 'confirmacao', 'cobranca_pix'],
          description: 'The VRE template to render',
        },
        data: {
          type: 'object',
          description: 'Template data (varies by template type)',
          additionalProperties: true,
        },
        channel: {
          type: 'string',
          enum: ['whatsapp', 'telegram', 'email', 'web'],
          description: 'Target channel for optimized rendering (default: whatsapp)',
          default: 'whatsapp',
        },
        format: {
          type: 'string',
          enum: ['webp', 'png', 'jpeg'],
          description: 'Output image format (default: webp)',
          default: 'webp',
        },
      },
      required: ['tenant_id', 'template_id', 'data'],
    },
  },
  {
    name: 'render_and_send',
    description: 'Render a VRE template and send it directly to a conversation. Combines rendering and sending in one operation.',
    inputSchema: {
      type: 'object',
      properties: {
        conversation_id: {
          type: 'string',
          description: 'The conversation ID to send the rendered image to',
        },
        template_id: {
          type: 'string',
          enum: ['menu_opcoes', 'card_produto', 'status_pedido', 'lista_produtos', 'confirmacao', 'cobranca_pix'],
          description: 'The VRE template to render',
        },
        data: {
          type: 'object',
          description: 'Template data (varies by template type)',
          additionalProperties: true,
        },
        caption: {
          type: 'string',
          description: 'Optional custom caption for the image (auto-generated if not provided)',
        },
        follow_up_text: {
          type: 'string',
          description: 'Optional text message to send after the image',
        },
      },
      required: ['conversation_id', 'template_id', 'data'],
    },
  },
  {
    name: 'list_templates',
    description: 'List available VRE templates with their schemas and example data.',
    inputSchema: {
      type: 'object',
      properties: {
        tenant_id: {
          type: 'string',
          description: 'Optional tenant ID to include custom templates',
        },
      },
    },
  },
  {
    name: 'preview_template',
    description: 'Preview a VRE template with sample data. Returns the rendered image for inspection.',
    inputSchema: {
      type: 'object',
      properties: {
        template_id: {
          type: 'string',
          enum: ['menu_opcoes', 'card_produto', 'status_pedido', 'lista_produtos', 'confirmacao', 'cobranca_pix'],
          description: 'The VRE template to preview',
        },
        data: {
          type: 'object',
          description: 'Optional custom data for preview (uses default sample data if not provided)',
          additionalProperties: true,
        },
      },
      required: ['template_id'],
    },
  },
  // Visual response tools for bot integration
  {
    name: 'mostrar_menu',
    description: 'Display a visual menu with numbered options for the user to choose from. Use this when you need to present multiple options clearly. Returns an image with caption.',
    inputSchema: {
      type: 'object',
      properties: {
        titulo: {
          type: 'string',
          description: 'Menu title',
        },
        opcoes: {
          type: 'array',
          description: 'List of options (max 8)',
          items: {
            type: 'object',
            properties: {
              label: { type: 'string', description: 'Option text' },
              icone: { type: 'string', description: 'Icon name (pedido, catalogo, entrega, financeiro, atendente, etc)' },
              descricao: { type: 'string', description: 'Optional description' },
            },
            required: ['label'],
          },
          maxItems: 8,
        },
        conversation_id: {
          type: 'string',
          description: 'Conversation ID to send the menu to',
        },
      },
      required: ['titulo', 'opcoes', 'conversation_id'],
    },
  },
  {
    name: 'mostrar_card_produto',
    description: 'Display a visual product card with image, price, and stock information. Use for showcasing a single product.',
    inputSchema: {
      type: 'object',
      properties: {
        nome: { type: 'string', description: 'Product name' },
        preco: { type: 'number', description: 'Product price' },
        unidade: { type: 'string', description: 'Unit (kg, un, cx, etc)', default: 'un' },
        imagem_url: { type: 'string', description: 'Product image URL' },
        estoque: { type: 'string', enum: ['disponivel', 'baixo', 'indisponivel'], description: 'Stock status' },
        descricao: { type: 'string', description: 'Product description' },
        conversation_id: { type: 'string', description: 'Conversation ID' },
      },
      required: ['nome', 'preco', 'conversation_id'],
    },
  },
  {
    name: 'mostrar_status_pedido',
    description: 'Display a visual order status with progress timeline. Use to show order tracking information.',
    inputSchema: {
      type: 'object',
      properties: {
        numero_pedido: { type: 'string', description: 'Order number' },
        status_atual: {
          type: 'string',
          enum: ['recebido', 'separacao', 'faturado', 'transporte', 'entregue'],
          description: 'Current order status'
        },
        itens_resumo: { type: 'string', description: 'Items summary' },
        valor_total: { type: 'number', description: 'Total order value' },
        previsao_entrega: { type: 'string', description: 'Delivery estimate' },
        motorista: { type: 'string', description: 'Driver name' },
        conversation_id: { type: 'string', description: 'Conversation ID' },
      },
      required: ['numero_pedido', 'status_atual', 'conversation_id'],
    },
  },
  {
    name: 'mostrar_lista_produtos',
    description: 'Display a visual product list for comparison. Use when showing multiple products side by side.',
    inputSchema: {
      type: 'object',
      properties: {
        titulo: { type: 'string', description: 'List title' },
        produtos: {
          type: 'array',
          description: 'List of products (max 6)',
          items: {
            type: 'object',
            properties: {
              nome: { type: 'string' },
              preco: { type: 'number' },
              unidade: { type: 'string' },
              estoque: { type: 'string' },
            },
            required: ['nome', 'preco'],
          },
          maxItems: 6,
        },
        conversation_id: { type: 'string', description: 'Conversation ID' },
      },
      required: ['titulo', 'produtos', 'conversation_id'],
    },
  },
  {
    name: 'mostrar_confirmacao',
    description: 'Display a visual order/action confirmation with item summary and total. Use for final confirmation before processing.',
    inputSchema: {
      type: 'object',
      properties: {
        titulo: { type: 'string', description: 'Confirmation title' },
        itens: {
          type: 'array',
          description: 'List of items',
          items: {
            type: 'object',
            properties: {
              nome: { type: 'string' },
              quantidade: { type: 'string' },
              preco: { type: 'number' },
            },
          },
        },
        valor_total: { type: 'number', description: 'Total value' },
        previsao_entrega: { type: 'string', description: 'Delivery estimate' },
        conversation_id: { type: 'string', description: 'Conversation ID' },
      },
      required: ['titulo', 'valor_total', 'conversation_id'],
    },
  },
  {
    name: 'mostrar_cobranca_pix',
    description: 'Display a PIX payment QR code with copy-paste code. Use for payment collection.',
    inputSchema: {
      type: 'object',
      properties: {
        valor: { type: 'number', description: 'Payment amount' },
        pix_payload: { type: 'string', description: 'PIX copy-paste code' },
        qr_code_url: { type: 'string', description: 'QR code image URL' },
        numero_pedido: { type: 'string', description: 'Order number' },
        expiracao: { type: 'string', description: 'Expiration time' },
        conversation_id: { type: 'string', description: 'Conversation ID' },
      },
      required: ['valor', 'pix_payload', 'conversation_id'],
    },
  },
];

export function registerVRETools(
  handlers: Map<string, (args: Record<string, unknown>) => Promise<unknown>>,
  client: LinktorClient
): void {
  handlers.set('render_template', async (args) => {
    return client.vre.render({
      tenant_id: args.tenant_id as string,
      template_id: args.template_id as string,
      data: args.data as Record<string, unknown>,
      channel: args.channel as string | undefined,
      format: args.format as string | undefined,
    });
  });

  handlers.set('render_and_send', async (args) => {
    return client.vre.renderAndSend({
      conversation_id: args.conversation_id as string,
      template_id: args.template_id as string,
      data: args.data as Record<string, unknown>,
      caption: args.caption as string | undefined,
      follow_up_text: args.follow_up_text as string | undefined,
    });
  });

  handlers.set('list_templates', async (args) => {
    return client.vre.listTemplates(args.tenant_id as string | undefined);
  });

  handlers.set('preview_template', async (args) => {
    return client.vre.preview({
      template_id: args.template_id as string,
      data: args.data as Record<string, unknown> | undefined,
    });
  });

  // Visual bot tools (render and send in one step)
  const visualTools = [
    'mostrar_menu',
    'mostrar_card_produto',
    'mostrar_status_pedido',
    'mostrar_lista_produtos',
    'mostrar_confirmacao',
    'mostrar_cobranca_pix',
  ];

  for (const toolName of visualTools) {
    handlers.set(toolName, async (args) => {
      const conversationId = args.conversation_id as string;
      const templateId = toolName.replace('mostrar_', '').replace('card_', '').replace('lista_', '');

      // Map tool name to template ID
      const templateMap: Record<string, string> = {
        'mostrar_menu': 'menu_opcoes',
        'mostrar_card_produto': 'card_produto',
        'mostrar_status_pedido': 'status_pedido',
        'mostrar_lista_produtos': 'lista_produtos',
        'mostrar_confirmacao': 'confirmacao',
        'mostrar_cobranca_pix': 'cobranca_pix',
      };

      // Remove conversation_id from data
      const { conversation_id, ...data } = args;

      return client.vre.renderAndSend({
        conversation_id: conversationId,
        template_id: templateMap[toolName] || templateId,
        data: data as Record<string, unknown>,
      });
    });
  }
}
