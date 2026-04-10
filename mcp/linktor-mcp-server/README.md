# Linktor MCP Server

MCP (Model Context Protocol) Server for Linktor - enabling AI assistants like Claude to interact with the Linktor omnichannel conversation platform.

## Overview

The Linktor MCP Server allows AI assistants to:
- Manage customer conversations across multiple channels
- Send and receive messages
- Access contact information
- Configure and test AI bots
- Query analytics and knowledge bases

## Architecture

```
┌─────────────────┐     ┌─────────────────┐     ┌─────────────────┐
│   Claude/AI     │────▶│  Linktor MCP    │────▶│  Linktor API    │
│   Assistant     │◀────│     Server      │◀────│  (Go Backend)   │
└─────────────────┘     └─────────────────┘     └─────────────────┘
                              │
                              ▼
                        stdio transport
```

## Features

- **34 Tools** for managing conversations, messages, contacts, channels, bots, analytics, and visual responses
- **6 Resources** for reading platform data
- **4 Prompts** for common customer support tasks
- **Visual Response Engine (VRE)** for rich visual messages on channels without native buttons
- Full TypeScript support with type definitions
- HTTP client with retry logic and error handling
- Works with Claude Desktop and other MCP-compatible clients

## Visual Response Engine (VRE)

The VRE enables sending rich visual responses on channels that don't support native buttons (like unofficial WhatsApp). SVG templates are rendered as images and sent with accessible captions.

```
┌─────────────────┐     ┌─────────────────┐     ┌─────────────────┐
│   AI decides    │────▶│  VRE renders    │────▶│  Image sent to  │
│   visual tool   │     │  SVG → PNG      │     │  WhatsApp/etc   │
└─────────────────┘     └─────────────────┘     └─────────────────┘
```

### VRE Tools (8 tools)

| Tool | Description |
|------|-------------|
| `render_template` | Render template and return base64 image |
| `render_and_send` | Render and send directly to conversation |
| `mostrar_menu` | Visual menu with up to 8 numbered options |
| `mostrar_card_produto` | Product card with price and stock |
| `mostrar_status_pedido` | Order status visual timeline |
| `mostrar_lista_produtos` | Comparative product list |
| `mostrar_confirmacao` | Order confirmation summary |
| `mostrar_cobranca_pix` | PIX QR code with copy-paste code |

### Visual Templates

#### 1. Menu Options (`menu_opcoes`)

Interactive menu with numbered options and icons. The AI shows this when presenting choices to the customer.

<p align="center">
<img src="https://via.placeholder.com/320x400/0F3460/FFFFFF?text=Menu+Visual" alt="Menu Template" />
</p>

```
┌─────────────────────────────────────┐
│  🏢 ACME CORP                       │
│  ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━│
│  Como posso ajudar?                 │
├─────────────────────────────────────┤
│                                     │
│  ┌─────────────────────────────┐   │
│  │ ① 🛒 Fazer pedido           │   │
│  │   Monte seu pedido          │   │
│  └─────────────────────────────┘   │
│                                     │
│  ┌─────────────────────────────┐   │
│  │ ② 📦 Status do pedido       │   │
│  │   Rastreie sua entrega      │   │
│  └─────────────────────────────┘   │
│                                     │
│  ┌─────────────────────────────┐   │
│  │ ③ 📋 Ver catálogo           │   │
│  │   Conheça nossos produtos   │   │
│  └─────────────────────────────┘   │
│                                     │
│  ┌─────────────────────────────┐   │
│  │ ④ 💰 Financeiro             │   │
│  │   Boletos e pagamentos      │   │
│  └─────────────────────────────┘   │
│                                     │
│  ┌─────────────────────────────┐   │
│  │ ⑤ 👤 Falar com atendente    │   │
│  │   Atendimento humano        │   │
│  └─────────────────────────────┘   │
│                                     │
│  ─────────────────────────────────  │
│  Responda com o número da opção     │
└─────────────────────────────────────┘
```

**Tool Call:**
```json
{
  "name": "mostrar_menu",
  "arguments": {
    "titulo": "Como posso ajudar?",
    "opcoes": [
      { "label": "Fazer pedido", "descricao": "Monte seu pedido", "icone": "pedido" },
      { "label": "Status do pedido", "icone": "entrega" },
      { "label": "Ver catálogo", "icone": "catalogo" },
      { "label": "Financeiro", "icone": "financeiro" },
      { "label": "Falar com atendente", "icone": "atendente" }
    ]
  }
}
```

#### 2. Product Card (`card_produto`)

Product card with image, price, and stock status. Used when presenting a specific product.

```
┌─────────────────────────────────────┐
│                                     │
│           🦐                        │
│                    ┌──────────────┐ │
│                    │ ⭐ MAIS      │ │
│                    │   VENDIDO   │ │
│                    └──────────────┘ │
├─────────────────────────────────────┤
│                                     │
│  Camarão Cinza Limpo GG             │
│  SKU: FRT-CM-2840                   │
│                                     │
│  ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━  │
│                                     │
│  R$ 62,90/kg        ✓ Em estoque    │
│                                     │
└─────────────────────────────────────┘
```

**Tool Call:**
```json
{
  "name": "mostrar_card_produto",
  "arguments": {
    "nome": "Camarão Cinza Limpo GG",
    "sku": "FRT-CM-2840",
    "preco": 62.90,
    "unidade": "kg",
    "estoque": 230,
    "destaque": "mais vendido",
    "mensagem": "Temos sim! 🦐 Esse é o nosso campeão de vendas."
  }
}
```

#### 3. Order Status (`status_pedido`)

Visual timeline showing order progress with delivery information.

```
┌─────────────────────────────────────┐
│  Pedido #4521                   🏢  │
├─────────────────────────────────────┤
│                                     │
│   ✓ ─────── ✓ ─────── ✓ ─────── 🚚 ─────── ○   │
│   │         │         │         │         │   │
│ Recebido Separação Faturado Transporte Entregue│
│                                     │
├──────────────────┬──────────────────┤
│ Itens            │ Valor            │
│ 12 produtos      │ R$ 3.847,50      │
├──────────────────┼──────────────────┤
│ Previsão         │ Motorista        │
│ Hoje, 16h        │ Carlos R.        │
└──────────────────┴──────────────────┘
```

**Tool Call:**
```json
{
  "name": "mostrar_status_pedido",
  "arguments": {
    "numero_pedido": "4521",
    "status_atual": "transporte",
    "itens_resumo": "12 produtos",
    "valor_total": 3847.50,
    "previsao_entrega": "Hoje, 16h",
    "motorista": "Carlos R."
  }
}
```

#### 4. Product List (`lista_produtos`)

Comparative list for multiple products. Used when customer asks for options.

```
┌─────────────────────────────────────┐
│  🐟 Pescados Disponíveis            │
│  4 produtos encontrados             │
├─────────────────────────────────────┤
│                                     │
│  ① 🦐 Camarão Cinza GG              │
│     FRT-CM-2840 · Em estoque        │
│                       R$ 62,90/kg   │
│                                     │
│  ② 🐟 Filé de Tilápia               │
│     FRT-TL-1205 · Em estoque        │
│                       R$ 34,50/kg   │
│                                     │
│  ③ 🐙 Polvo Limpo                   │
│     FRT-PV-0892 · Estoque baixo     │
│                       R$ 89,00/kg   │
│                                     │
│  ④ 🦑 Lula Limpa em Anéis           │
│     FRT-LL-0445 · Em estoque        │
│                       R$ 45,90/kg   │
│                                     │
│  ─────────────────────────────────  │
│  Responda com o número para detalhes│
└─────────────────────────────────────┘
```

**Tool Call:**
```json
{
  "name": "mostrar_lista_produtos",
  "arguments": {
    "titulo": "Pescados Disponíveis",
    "produtos": [
      { "nome": "Camarão Cinza GG", "sku": "FRT-CM-2840", "preco": 62.90, "estoque": "ok" },
      { "nome": "Filé de Tilápia", "sku": "FRT-TL-1205", "preco": 34.50, "estoque": "ok" },
      { "nome": "Polvo Limpo", "sku": "FRT-PV-0892", "preco": 89.00, "estoque": "baixo" },
      { "nome": "Lula em Anéis", "sku": "FRT-LL-0445", "preco": 45.90, "estoque": "ok" }
    ],
    "mensagem": "Separei os pescados disponíveis pra você! 🐟"
  }
}
```

#### 5. Order Confirmation (`confirmacao`)

Summary card for order confirmation. Critical conversion moment.

```
┌─────────────────────────────────────┐
│          ┌───┐                      │
│          │ ✓ │                      │
│          └───┘                      │
│     Confirmar Pedido?               │
│  Entrega prevista: Quinta, 12/02    │
├─────────────────────────────────────┤
│                                     │
│  🦐 Camarão Cinza GG · 15kg         │
│                       R$ 943,50     │
│  ─────────────────────────────────  │
│  🦑 Lula em Anéis · 8kg             │
│                       R$ 367,20     │
│  ─────────────────────────────────  │
│  🐟 Filé de Tilápia · 20kg          │
│                       R$ 690,00     │
│                                     │
├─────────────────────────────────────┤
│  Total              R$ 2.000,70     │
├─────────────────────────────────────┤
│  Responda SIM para confirmar        │
└─────────────────────────────────────┘
```

**Tool Call:**
```json
{
  "name": "mostrar_confirmacao",
  "arguments": {
    "titulo": "Confirmar Pedido?",
    "itens": [
      { "nome": "Camarão Cinza GG", "quantidade": "15kg", "valor": 943.50 },
      { "nome": "Lula em Anéis", "quantidade": "8kg", "valor": 367.20 },
      { "nome": "Filé de Tilápia", "quantidade": "20kg", "valor": 690.00 }
    ],
    "valor_total": 2000.70,
    "previsao_entrega": "Quinta, 12/02"
  }
}
```

#### 6. PIX Payment (`cobranca_pix`)

Payment card with QR code and copy-paste code. Reusable across verticals.

```
┌─────────────────────────────────────┐
│            ◆ PIX                    │
│      Pagamento via PIX              │
├─────────────────────────────────────┤
│                                     │
│         ┌───────────────┐           │
│         │ ▓▓░░▓▓░░▓▓░░ │           │
│         │ ░░▓▓░░▓▓░░▓▓ │           │
│         │ ▓▓░░▓▓░░▓▓░░ │           │
│         │ ░░▓▓░░▓▓░░▓▓ │           │
│         │   QR CODE    │           │
│         └───────────────┘           │
│                                     │
│        R$ 2.000,70                  │
│   Pedido #4587 · Válido 30min       │
│                                     │
│  ┌─────────────────────────────┐   │
│  │ 00020126580014br.gov.bcb... │   │
│  └─────────────────────────────┘   │
│                                     │
│  Escaneie ou copie o código acima   │
└─────────────────────────────────────┘
```

**Tool Call:**
```json
{
  "name": "mostrar_cobranca_pix",
  "arguments": {
    "valor": 2000.70,
    "pedido_id": "4587",
    "pix_payload": "00020126580014br.gov.bcb.pix...",
    "validade_minutos": 30
  }
}
```

### How VRE Works

1. **AI Decision**: The LLM decides when a visual response is appropriate
2. **Tool Call**: AI calls a VRE tool (e.g., `mostrar_menu`) with structured data
3. **Template Resolution**: VRE loads the SVG template with tenant branding
4. **Rendering**: Template is rendered to PNG via headless Chrome
5. **Caption Generation**: Accessible text caption is generated
6. **Delivery**: Image + caption sent to the customer's channel

### Tenant Customization

Each tenant can customize templates via `config.json`:

```json
{
  "tenant_id": "acme-corp",
  "name": "Acme Corp",
  "logo_url": "https://...",
  "primary_color": "#0F3460",
  "secondary_color": "#E94560",
  "accent_color": "#16C79A"
}
```

## Installation

```bash
# Using npm
npm install @linktor/mcp-server

# Using npx (no installation required)
npx @linktor/mcp-server
```

## Configuration

### Environment Variables

| Variable | Description | Required |
|----------|-------------|----------|
| `LINKTOR_API_URL` | Linktor API URL (default: `http://localhost:8080/api/v1`) | No |
| `LINKTOR_API_KEY` | API Key for authentication | Yes* |
| `LINKTOR_ACCESS_TOKEN` | JWT Access Token (alternative to API Key) | Yes* |
| `LINKTOR_TIMEOUT` | Request timeout in ms (default: 30000) | No |
| `LINKTOR_MAX_RETRIES` | Max retry attempts (default: 3) | No |
| `LINKTOR_RETRY_DELAY` | Delay between retries in ms (default: 1000) | No |

*Either `LINKTOR_API_KEY` or `LINKTOR_ACCESS_TOKEN` is required.

### Claude Desktop Configuration

Add to your Claude Desktop config file:

**macOS:** `~/Library/Application Support/Claude/claude_desktop_config.json`
**Windows:** `%APPDATA%\Claude\claude_desktop_config.json`

```json
{
  "mcpServers": {
    "linktor": {
      "command": "npx",
      "args": ["-y", "@linktor/mcp-server"],
      "env": {
        "LINKTOR_API_URL": "https://api.linktor.io",
        "LINKTOR_API_KEY": "your-api-key"
      }
    }
  }
}
```

## Project Structure

```
linktor-mcp-server/
├── package.json
├── tsconfig.json
├── README.md
├── bin/
│   └── linktor-mcp.js          # CLI executable
├── examples/
│   └── claude-desktop.json     # Example Claude Desktop config
└── src/
    ├── index.ts                # Entry point
    ├── server.ts               # MCP Server setup
    ├── config.ts               # Configuration with Zod validation
    ├── api/
    │   ├── client.ts           # HTTP client for Linktor API
    │   └── types.ts            # TypeScript type definitions
    ├── tools/
    │   ├── index.ts            # Tool exports
    │   ├── conversations.ts    # Conversation management
    │   ├── messages.ts         # Message operations
    │   ├── contacts.ts         # Contact management
    │   ├── channels.ts         # Channel operations
    │   ├── bots.ts             # Bot management
    │   ├── analytics.ts        # Analytics queries
    │   ├── knowledge.ts        # Knowledge base search
    │   └── vre.ts              # Visual Response Engine tools
    ├── resources/
    │   ├── index.ts            # Resource exports
    │   └── handlers.ts         # Resource handlers
    └── prompts/
        ├── index.ts            # Prompt exports
        └── templates.ts        # Prompt templates
```

## Available Tools

### Conversations (8 tools)

| Tool | Description |
|------|-------------|
| `list_conversations` | List conversations with filters (status, channel, assigned user) |
| `get_conversation` | Get detailed conversation information |
| `create_conversation` | Create a new conversation with a contact |
| `assign_conversation` | Assign conversation to a user/agent |
| `unassign_conversation` | Remove assignment from conversation |
| `resolve_conversation` | Mark conversation as resolved |
| `reopen_conversation` | Reopen a resolved/closed conversation |
| `close_conversation` | Close conversation permanently |

### Messages (3 tools)

| Tool | Description |
|------|-------------|
| `list_messages` | List messages in a conversation with pagination |
| `get_message` | Get specific message details |
| `send_message` | Send message (text, image, document, etc.) |

### Contacts (5 tools)

| Tool | Description |
|------|-------------|
| `list_contacts` | List contacts with search and tag filters |
| `get_contact` | Get contact details with channel identities |
| `create_contact` | Create new contact with custom fields |
| `update_contact` | Update contact information |
| `delete_contact` | Delete a contact |

### Channels (4 tools)

| Tool | Description |
|------|-------------|
| `list_channels` | List channels (WhatsApp, Telegram, etc.) |
| `get_channel` | Get channel configuration and status |
| `connect_channel` | Connect/activate a channel |
| `disconnect_channel` | Disconnect a channel |

### Bots (5 tools)

| Tool | Description |
|------|-------------|
| `list_bots` | List AI bots with status filter |
| `get_bot` | Get bot configuration and rules |
| `activate_bot` | Activate a bot |
| `deactivate_bot` | Deactivate a bot |
| `test_bot` | Test bot with a sample message |

### Analytics & Knowledge Base (5 tools)

| Tool | Description |
|------|-------------|
| `get_analytics_summary` | Get analytics for date range |
| `get_conversation_stats` | Get conversation statistics |
| `search_knowledge` | Semantic search in knowledge base |
| `list_knowledge_documents` | List KB documents |
| `get_knowledge_document` | Get document content |

### Visual Response Engine (8 tools)

| Tool | Description |
|------|-------------|
| `render_template` | Render any template and return base64 image |
| `render_and_send` | Render template and send directly to conversation |
| `mostrar_menu` | Display visual menu with numbered options |
| `mostrar_card_produto` | Display product card with price/stock |
| `mostrar_status_pedido` | Display order status timeline |
| `mostrar_lista_produtos` | Display comparative product list |
| `mostrar_confirmacao` | Display order confirmation summary |
| `mostrar_cobranca_pix` | Display PIX payment QR code |

## Resources

| URI | Description |
|-----|-------------|
| `linktor://conversations` | Active conversations list |
| `linktor://conversations/{id}` | Specific conversation details |
| `linktor://contacts` | All contacts |
| `linktor://contacts/{id}` | Specific contact details |
| `linktor://channels` | Configured channels |
| `linktor://channels/{id}` | Specific channel details |
| `linktor://bots` | AI bots list |
| `linktor://bots/{id}` | Specific bot details |
| `linktor://users` | Team members/agents |
| `linktor://analytics/summary` | Analytics summary (last 30 days) |

## Prompts

| Prompt | Description | Arguments |
|--------|-------------|-----------|
| `customer_support` | Customer support assistant | `company_name`, `tone?` |
| `conversation_summary` | Summarize a conversation | `conversation_id` |
| `draft_response` | Draft a customer response | `message`, `context?` |
| `analyze_sentiment` | Analyze text sentiment | `text` |

## Usage Examples

### List Open Conversations
```
"Show me all open conversations"
"List conversations assigned to user123"
"Get conversations from the WhatsApp channel"
```

### Send a Message
```
"Send 'Hello, how can I help you?' to conversation abc123"
"Reply to the customer with our refund policy"
```

### Manage Contacts
```
"Create a contact for John Doe with email john@example.com"
"Find contacts tagged as 'VIP'"
"Update the phone number for contact xyz789"
```

### Bot Operations
```
"List all active bots"
"Test the support bot with 'How do I reset my password?'"
"Deactivate bot abc123"
```

### Analytics
```
"Show me conversation statistics for the last week"
"Get analytics summary from January 1st to January 31st"
```

### Knowledge Base
```
"Search the knowledge base for refund policy"
"List all documents in knowledge base kb123"
```

## Development

```bash
# Install dependencies
npm install

# Build
npm run build

# Run in development mode (watch)
npm run dev

# Type check
npm run typecheck

# Start the server
npm run start
```

## Supported Channels

The Linktor platform supports the following channels:
- WebChat
- WhatsApp (Business & Official API)
- Telegram
- SMS
- RCS
- Instagram
- Facebook Messenger
- Email
- Voice

## Error Handling

The MCP Server handles errors gracefully and returns structured error responses:

```json
{
  "error": true,
  "code": "NOT_FOUND",
  "message": "Conversation not found",
  "details": {}
}
```

## License

MIT

## Links

- [Linktor Documentation](https://docs.linktor.io)
- [MCP Protocol Specification](https://modelcontextprotocol.io)
- [Claude Desktop](https://claude.ai/download)
