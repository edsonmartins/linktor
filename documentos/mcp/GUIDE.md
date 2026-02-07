# MCP Playground para Docusaurus

## Visão Geral

Componente React interativo que embarca dentro da documentação Docusaurus, permitindo:

- **Autodescoberta**: lista automaticamente tools, resources e prompts do MCP server
- **Playground**: formulários dinâmicos gerados a partir do `inputSchema` de cada tool
- **Execução ao vivo**: invoca tools via Streamable HTTP e mostra responses em tempo real
- **Configurável**: URL do server, API key, e personalização visual

---

## Instalação no Docusaurus

### 1. Copiar o componente

```bash
cp McpPlayground.jsx seu-docusaurus/src/components/McpPlayground.jsx
```

### 2. Usar em páginas MDX

Crie ou edite qualquer `.mdx` no Docusaurus:

```mdx
---
title: API Playground
---

import McpPlayground from '@site/src/components/McpPlayground';

# VendaX.ai — MCP Playground

Teste as tools da API diretamente no browser.

<McpPlayground
  serverUrl="https://api.vendax.ai/mcp"
  serverName="VendaX.ai Tools"
  showConfig={false}
/>
```

### 3. Múltiplos servers na mesma página

```mdx
## Ferramentas de Vendas

<McpPlayground
  serverUrl="https://api.vendax.ai/mcp/sales"
  serverName="Sales Agent Tools"
  showConfig={false}
/>

## Ferramentas de Integração ERP

<McpPlayground
  serverUrl="https://api.vendax.ai/mcp/erp"
  serverName="ERP Integration Tools"
  showConfig={false}
/>
```

### 4. Modo aberto (para desenvolvedores)

```mdx
## Sandbox de Desenvolvimento

Conecte a qualquer MCP server:

<McpPlayground showConfig={true} />
```

---

## Requisitos do MCP Server

O server precisa expor um endpoint **Streamable HTTP** com CORS habilitado:

```typescript
// server.ts
import { McpServer } from "@modelcontextprotocol/sdk/server/mcp.js";
import { StreamableHTTPServerTransport } from "@modelcontextprotocol/sdk/server/streamableHttp.js";
import express from "express";
import cors from "cors";
import { z } from "zod";

const app = express();
app.use(cors());                  // ← CORS obrigatório para browser
app.use(express.json());

const server = new McpServer({
  name: "vendax-tools",
  version: "1.0.0",
});

// Registrar tools
server.tool(
  "buscar_cliente",
  "Busca cliente por CNPJ ou razão social",
  {
    cnpj: z.string().optional().describe("CNPJ do cliente"),
    nome: z.string().optional().describe("Razão social parcial"),
  },
  async ({ cnpj, nome }) => {
    // implementação...
    return {
      content: [{ type: "text", text: JSON.stringify(resultado) }],
    };
  }
);

// Endpoint MCP
const transports = new Map();

app.post("/mcp", async (req, res) => {
  const sessionId = req.headers["mcp-session-id"];
  
  let transport;
  if (sessionId && transports.has(sessionId)) {
    transport = transports.get(sessionId);
  } else {
    transport = new StreamableHTTPServerTransport("/mcp", res);
    transports.set(transport.sessionId, transport);
    await server.connect(transport);
  }
  
  await transport.handleRequest(req, res);
});

app.listen(3001, () => console.log("MCP server on :3001"));
```

---

## Segurança para Produção

### Autenticação via API Key

```typescript
// Middleware de auth no server
app.use("/mcp", (req, res, next) => {
  const key = req.headers.authorization?.replace("Bearer ", "");
  if (!key || !validApiKeys.has(key)) {
    return res.status(401).json({ error: "Unauthorized" });
  }
  next();
});
```

No playground, o campo "API Key" envia o header `Authorization: Bearer <key>`.

### Rate Limiting

```typescript
import rateLimit from "express-rate-limit";

app.use("/mcp", rateLimit({
  windowMs: 60_000,       // 1 minuto
  max: 30,                // 30 requests por minuto
  message: { error: "Rate limit exceeded" },
}));
```

### Read-Only Mode (Playground público)

Crie um subset de tools que são **safe para demonstração**:

```typescript
// Tools seguras para playground público
const PLAYGROUND_TOOLS = ["buscar_cliente", "listar_produtos", "calcular_frete"];

// Filtrar no middleware
app.use("/mcp/playground", (req, res, next) => {
  const body = req.body;
  if (body.method === "tools/call" && !PLAYGROUND_TOOLS.includes(body.params?.name)) {
    return res.json({
      jsonrpc: "2.0",
      id: body.id,
      error: { code: -32600, message: "Tool not available in playground mode" },
    });
  }
  next();
});
```

---

## Customização Visual

O componente aceita as props:

| Prop | Tipo | Default | Descrição |
|------|------|---------|-----------|
| `serverUrl` | `string` | `http://localhost:3001/mcp` | URL do MCP server |
| `serverName` | `string` | `"MCP Server"` | Nome exibido no header |
| `apiKey` | `string` | `""` | API key pré-configurada |
| `showConfig` | `boolean` | `true` | Mostrar/ocultar painel de configuração |

Para adaptar ao tema do Docusaurus, você pode wrappear com CSS variables:

```css
/* src/css/custom.css */
[data-theme='light'] .mcp-playground {
  --mcp-bg: #f8f9fb;
  --mcp-border: #e2e4e9;
  --mcp-text: #1a1c2e;
}
```

---

## Arquitetura

```
┌──────────────────────────────────────┐
│  Docusaurus (Static Site)            │
│                                      │
│  ┌────────────────────────────────┐  │
│  │  <McpPlayground />             │  │
│  │                                │  │
│  │  ┌──────────┐ ┌─────────────┐ │  │
│  │  │ Tool List│ │ Schema Form │ │  │
│  │  └──────────┘ └──────┬──────┘ │  │
│  │                      │        │  │
│  │  JSON-RPC 2.0 ───────┘        │  │
│  └──────────────┬─────────────────┘  │
└─────────────────┼────────────────────┘
                  │ POST /mcp
                  ▼
┌──────────────────────────────────────┐
│  MCP Server (Streamable HTTP)        │
│  ┌──────────┐ ┌──────┐ ┌─────────┐  │
│  │  Tools   │ │ Res. │ │ Prompts │  │
│  └──────────┘ └──────┘ └─────────┘  │
│                                      │
│  Auth · Rate Limit · CORS            │
└──────────────────────────────────────┘
```
