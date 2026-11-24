---
sidebar_position: 1
title: Servidor MCP
description: O que o servidor MCP do Linktor expõe (tools, resources e prompts), como compilar, configurar em Claude Desktop, Claude Code e outros clientes, e as variáveis de ambiente.
---

# Servidor MCP

O servidor MCP (Model Context Protocol) do Linktor permite que um assistente de IA — Claude Desktop,
Claude Code ou qualquer cliente MCP — opere o Linktor: listar conversas, enviar mensagens, gerenciar
canais e contatos, testar bots. Ele é um processo Node.js separado, em `mcp/linktor-mcp-server`, que
traduz chamadas de tools em requisições à [API HTTP](../api/visao-geral.md).

```mermaid
flowchart LR
    A[Cliente MCP<br/>Claude Desktop, Claude Code…] -- stdio ou HTTP --> B[linktor-mcp-server<br/>Node.js]
    B -- "HTTPS + X-API-Key" --> C[API do Linktor<br/>/api/v1]
```

O servidor não tem estado nem banco próprio: tudo o que ele faz é limitado pelo que a credencial
configurada pode fazer na API.

## O que o servidor expõe

| Tipo | Quantidade | Onde está no código |
|---|---|---|
| Tools | 51 | `src/tools/*.ts` |
| Resources | 6 anunciados, mais 4 por id | `src/resources/handlers.ts` |
| Prompts | 4 | `src/prompts/templates.ts` |

:::warning Nem todas as tools funcionam hoje
O servidor MCP foi escrito contra uma versão anterior da API. Das 51 tools, **8 falham** contra o
backend atual (rota que não existe mais ou formato de resposta diferente), assim como 3 resources.
Elas estão marcadas nas tabelas abaixo. As demais 43 tools chamam rotas que existem.
:::

### Tools

#### Conversas (8)

| Tool | O que faz | Rota chamada | Situação |
|---|---|---|---|
| `list_conversations` | Lista conversas com filtros | `GET /conversations` | ok |
| `get_conversation` | Obtém uma conversa | `GET /conversations/:id` | ok |
| `create_conversation` | Cria conversa (`contact_id` + `channel_id`) | `POST /conversations` | ok |
| `assign_conversation` | Atribui a um usuário | `POST /conversations/:id/assign` | ok |
| `resolve_conversation` | Marca como resolvida | `POST /conversations/:id/resolve` | ok |
| `reopen_conversation` | Reabre | `POST /conversations/:id/reopen` | ok |
| `unassign_conversation` | Remove a atribuição | `POST /conversations/:id/unassign` | **falha** — rota inexistente |
| `close_conversation` | Fecha a conversa | `POST /conversations/:id/close` | **falha** — rota inexistente |

#### Mensagens (5)

| Tool | O que faz | Rota chamada | Situação |
|---|---|---|---|
| `list_messages` | Lista mensagens de uma conversa | `GET /conversations/:id/messages` | ok |
| `send_message` | Envia mensagem em uma conversa | `POST /conversations/:id/messages` | ok |
| `send_channel_message` | Envia por canal: usa a conversa informada ou cria uma (`channel_id` + `contact_id`) e envia | `POST /conversations` + `POST /conversations/:id/messages` | ok |
| `receive_channel_messages` | Lê as conversas recentes de um canal com suas mensagens | `GET /conversations` + `GET /conversations/:id/messages` | ok |
| `get_message` | Obtém uma mensagem | `GET /conversations/:id/messages/:messageId` | **falha** — rota inexistente (a API usa `GET /messages/:id`) |

`receive_channel_messages` é leitura do que já está gravado no Linktor (polling), não uma assinatura em
tempo real. Para eventos em tempo real, use [webhooks](../api/receber-eventos.md) ou
[WebSocket](../api/tempo-real.md).

:::note Paginação nas tools de listagem
`list_conversations` e `list_messages` aceitam `limit`. Os parâmetros `offset` e `after` que as tools
oferecem não são lidos pela API atual (que pagina por cursor `before`) e são ignorados. Em
`list_contacts`, `limit` e `offset` também são ignorados: a API pagina por `page` e `page_size`, que a
tool não envia — o resultado é sempre a primeira página de 20 contatos.
:::

#### Contatos (5)

| Tool | O que faz | Rota chamada | Situação |
|---|---|---|---|
| `list_contacts` | Lista contatos, com busca | `GET /contacts` | ok (só a primeira página) |
| `get_contact` | Obtém um contato | `GET /contacts/:id` | ok |
| `create_contact` | Cria um contato | `POST /contacts` | ok |
| `update_contact` | Atualiza um contato | `PUT /contacts/:id` | ok |
| `delete_contact` | Remove um contato | `DELETE /contacts/:id` | ok |

#### Canais (13)

| Tool | O que faz | Rota chamada | Situação |
|---|---|---|---|
| `list_channels` | Lista canais | `GET /channels` | ok |
| `get_channel` | Obtém um canal | `GET /channels/:id` | ok |
| `create_channel` | Cria um canal | `POST /channels` | ok |
| `update_channel` | Atualiza nome, identificador, `config` ou `credentials` | `PUT /channels/:id` | ok |
| `delete_channel` | Remove um canal | `DELETE /channels/:id` | ok |
| `connect_channel` | Conecta (devolve QR no WhatsApp não oficial) | `POST /channels/:id/connect` | ok |
| `disconnect_channel` | Desconecta | `POST /channels/:id/disconnect` | ok |
| `set_channel_enabled` | Habilita ou desabilita | `PUT /channels/:id/enabled` | ok |
| `set_channel_status` | Define `active`/`inactive` (compatibilidade) | `PUT /channels/:id/status` | ok |
| `request_whatsapp_pair_code` | Pede código de pareamento | `POST /channels/:id/pair` | ok |
| `test_channel_config` | Valida credenciais sem criar o canal | `POST /channels/test` | ok |
| `get_whatsapp_coexistence_status` | Estado da coexistência (WhatsApp oficial) | `GET /channels/:id/coexistence-status` | ok |
| `subscribe_whatsapp_echoes` | Assina os ecos de mensagem (coexistência) | `POST /channels/:id/subscribe-echoes` | ok |

O `type` aceito por `create_channel` é limitado, no schema da tool, a `webchat`, `whatsapp`,
`whatsapp_official`, `whatsapp_unofficial`, `telegram`, `sms`, `rcs`, `instagram`, `facebook`, `email` e
`voice`. Os tipos `teams`, `slack`, `mattermost` e `direto` existem na API, mas não estão na lista da
tool. Os campos de cada canal estão em [Canais](../canais/visao-geral.md) e
[Gestão de canais pela API](../api/canais.md).

#### Bots (5)

| Tool | O que faz | Rota chamada | Situação |
|---|---|---|---|
| `get_bot` | Obtém um bot | `GET /bots/:id` | ok |
| `activate_bot` | Ativa | `POST /bots/:id/activate` | ok |
| `deactivate_bot` | Desativa | `POST /bots/:id/deactivate` | ok |
| `test_bot` | Envia uma mensagem de teste | `POST /bots/:id/test` | ok |
| `list_bots` | Lista os bots | `GET /bots` | **falha** — a rota responde sem o campo `success`, que o servidor MCP exige |

#### Base de conhecimento (3)

| Tool | O que faz | Rota chamada | Situação |
|---|---|---|---|
| `search_knowledge` | Busca em uma base | `POST /knowledge-bases/:id/search` | **falha** — resposta sem o campo `success` |
| `list_knowledge_documents` | Lista os itens da base | `GET /knowledge-bases/:id/documents` | **falha** — rota inexistente (a API usa `/items`) |
| `get_knowledge_document` | Obtém um item | `GET /knowledge-bases/:id/documents/:documentId` | **falha** — rota inexistente |

#### Analytics (2)

| Tool | O que faz | Rota chamada | Situação |
|---|---|---|---|
| `get_conversation_stats` | Estatísticas de conversas | `GET /analytics/conversations` | ok, com ressalva |
| `get_analytics_summary` | Resumo do período | `GET /analytics/summary` | **falha** — rota inexistente (a API usa `/analytics/overview`) |

A ressalva de `get_conversation_stats`: a tool envia `period` como `day`, `week` ou `month`; a API
documenta `daily`, `weekly` e `monthly`. O período efetivamente aplicado pode não ser o pedido.

#### Respostas visuais — VRE (10)

O VRE (Visual Response Engine) renderiza modelos visuais como imagem, para canais sem botões nativos.

| Tool | O que faz | Rota chamada |
|---|---|---|
| `render_template` | Renderiza um modelo e devolve a imagem em base64 (exige `tenant_id`) | `POST /vre/render` |
| `render_and_send` | Renderiza e envia para uma conversa | `POST /vre/render-and-send` |
| `list_templates` | Lista os modelos visuais | `GET /vre/templates` |
| `preview_template` | Pré-visualiza um modelo | `GET /vre/templates/:id/preview` |
| `mostrar_menu` | Menu com até 8 opções numeradas | `POST /vre/render-and-send` |
| `mostrar_card_produto` | Cartão de produto com preço e estoque | `POST /vre/render-and-send` |
| `mostrar_status_pedido` | Linha do tempo do pedido | `POST /vre/render-and-send` |
| `mostrar_lista_produtos` | Lista comparativa de produtos | `POST /vre/render-and-send` |
| `mostrar_confirmacao` | Resumo para confirmação | `POST /vre/render-and-send` |
| `mostrar_cobranca_pix` | QR code Pix com copia-e-cola | `POST /vre/render-and-send` |

:::note
As rotas `/vre/*` só existem quando o VRE está habilitado na instalação do Linktor. Sem ele, as 10
tools falham (a rota não existe). O comportamento de cada tool de VRE não foi validado ponta a ponta para
esta documentação.
:::

### Resources

Anunciados em `resources/list`:

| URI | Conteúdo | Situação |
|---|---|---|
| `linktor://conversations` | Conversas abertas (até 50) | ok |
| `linktor://contacts` | Contatos | ok |
| `linktor://channels` | Canais | ok |
| `linktor://bots` | Bots | **falha** — mesmo motivo de `list_bots` |
| `linktor://analytics/summary` | Resumo dos últimos 30 dias | **falha** — rota inexistente |
| `linktor://users` | Usuários do tenant | **falha com API key** (`403`): a rota `/users` é restrita aos papéis `admin` e `owner` |

Também é possível ler um item pelo id — `linktor://conversations/{id}`, `linktor://contacts/{id}`,
`linktor://channels/{id}` e `linktor://bots/{id}`. Esses quatro modelos de URI funcionam em
`resources/read`, mas o servidor não os anuncia (não implementa `resources/templates/list`), então o
cliente precisa conhecer o formato.

### Prompts

| Prompt | Argumentos | Para quê |
|---|---|---|
| `customer_support` | `company_name` (obrigatório), `tone` | Instruções de atendente de suporte |
| `conversation_summary` | `conversation_id` (obrigatório) | Resumir uma conversa, com pontos-chave e pendências |
| `draft_response` | `message` (obrigatório), `context` | Rascunhar a resposta a uma mensagem do cliente |
| `analyze_sentiment` | `text` (obrigatório) | Analisar o sentimento de um texto |

Os textos dos prompts estão em inglês.

## Instalar

Requisitos: Node.js 18 ou superior e uma [chave de API](../painel/chaves-de-api.md) do Linktor.

```bash
cd mcp/linktor-mcp-server
npm install
npm run build
```

O build gera `dist/index.js` (transporte stdio) e `dist/http-server.js` (transporte HTTP).

:::warning Aponte para `dist/index.js`, não para o binário `linktor-mcp`
O `package.json` declara o pacote `@linktor/mcp-server` com o binário `linktor-mcp`, e o exemplo que
acompanha o código usa `npx -y @linktor/mcp-server`. Dois cuidados:

- A publicação do pacote no npm não foi confirmada para esta documentação. Compile a partir do
  repositório.
- O binário `bin/linktor-mcp.js` importa `dist/index.js`, mas o servidor só inicia quando `index.js` é o
  arquivo executado diretamente. Pelo binário, o processo termina sem subir o servidor. Use sempre
  `node /caminho/para/linktor-mcp-server/dist/index.js`.

Pelo mesmo motivo, instale em um caminho **sem espaços nem caracteres especiais**: a detecção de
"executado diretamente" compara o caminho do arquivo literalmente.
:::

## Variáveis de ambiente

### Conexão com a API do Linktor (os dois transportes)

| Variável | Padrão | Descrição |
|---|---|---|
| `LINKTOR_API_URL` | `http://localhost:8081/api/v1` | Base da API, **incluindo** `/api/v1` |
| `LINKTOR_API_KEY` | — | Chave de API (`lk_…`), enviada como `X-API-Key` |
| `LINKTOR_ACCESS_TOKEN` | — | Token JWT de usuário, enviado como `Authorization: Bearer`. Se definido, **tem precedência** sobre a chave de API |
| `LINKTOR_TIMEOUT` | `30000` | Tempo limite por requisição, em ms |
| `LINKTOR_MAX_RETRIES` | `3` | Tentativas extras em erro de rede ou 5xx (erros 4xx não são repetidos) |
| `LINKTOR_RETRY_DELAY` | `1000` | Espera base entre tentativas, em ms (multiplicada pelo número da tentativa) |

Sem `LINKTOR_API_KEY` nem `LINKTOR_ACCESS_TOKEN`, o servidor sobe com um aviso e toda tool responde
`401`.

Prefira a chave de API: o token JWT expira e o servidor MCP não o renova. A chave só alcança o que seus
scopes permitem — veja [Autenticação](../api/autenticacao.md). Para usar todas as tools que funcionam,
a chave precisa de `conversations:write`, `messages:send`, `contacts:write`, `channels:write`,
`bots:write` e `analytics:read`; para um assistente só de leitura, conceda apenas os scopes `:read`.

### Transporte HTTP

| Variável | Padrão | Descrição |
|---|---|---|
| `MCP_HTTP_PORT` | `3001` | Porta |
| `MCP_HTTP_HOST` | `127.0.0.1` | Interface de escuta |
| `MCP_HTTP_AUTH_TOKEN` | — | Se definido, `POST /mcp` exige `Authorization: Bearer <token>` |
| `MCP_HTTP_ALLOWED_ORIGINS` | — | Origens de navegador permitidas (CORS), separadas por vírgula. Vazio: só `localhost` e `127.0.0.1` |

## Configurar o cliente

### Claude Desktop

Edite `claude_desktop_config.json` (em **Settings → Developer → Edit Config**; o nome do menu pode
mudar entre versões) e reinicie o aplicativo:

```json
{
  "mcpServers": {
    "linktor": {
      "command": "node",
      "args": ["/caminho/para/linktor/mcp/linktor-mcp-server/dist/index.js"],
      "env": {
        "LINKTOR_API_URL": "https://linktor.suaempresa.com/api/v1",
        "LINKTOR_API_KEY": "lk_sua_chave"
      }
    }
  }
}
```

### Claude Code

```bash
claude mcp add linktor \
  --env LINKTOR_API_URL=https://linktor.suaempresa.com/api/v1 \
  --env LINKTOR_API_KEY=lk_sua_chave \
  -- node /caminho/para/linktor/mcp/linktor-mcp-server/dist/index.js
```

Confira com `claude mcp list` e, dentro de uma sessão, com `/mcp`.

### Outros clientes (stdio)

Qualquer cliente MCP com transporte stdio serve. Configure:

| Item | Valor |
|---|---|
| Comando | `node` |
| Argumentos | `/caminho/para/linktor/mcp/linktor-mcp-server/dist/index.js` |
| Ambiente | `LINKTOR_API_URL`, `LINKTOR_API_KEY` |

Para testar sem cliente nenhum, use o MCP Inspector:

```bash
LINKTOR_API_URL=https://linktor.suaempresa.com/api/v1 LINKTOR_API_KEY=lk_sua_chave \
  npx @modelcontextprotocol/inspector node dist/index.js
```

### Transporte HTTP

```bash
cd mcp/linktor-mcp-server
LINKTOR_API_URL=https://linktor.suaempresa.com/api/v1 \
LINKTOR_API_KEY=lk_sua_chave \
MCP_HTTP_AUTH_TOKEN=um-token-longo-e-aleatorio \
npm run start:http
```

| Rota | Descrição |
|---|---|
| `POST /mcp` | Endpoint JSON-RPC 2.0 |
| `GET /health` | `{"status":"ok","service":"linktor-mcp-server"}` |

Métodos atendidos: `initialize`, `tools/list`, `tools/call`, `resources/list`, `resources/read`,
`prompts/list`, `prompts/get` e a notificação `notifications/initialized`. A sessão é identificada pelo
cabeçalho `Mcp-Session-Id`, devolvido na primeira resposta e que o cliente reenvia nas seguintes;
sessões expiram 30 minutos depois de criadas.

```bash
# 1) initialize — guarde o Mcp-Session-Id da resposta
curl -i -X POST http://127.0.0.1:3001/mcp \
  -H "Authorization: Bearer um-token-longo-e-aleatorio" \
  -H "Content-Type: application/json" \
  -d '{"jsonrpc":"2.0","id":1,"method":"initialize","params":{}}'

# 2) chama uma tool
curl -X POST http://127.0.0.1:3001/mcp \
  -H "Authorization: Bearer um-token-longo-e-aleatorio" \
  -H "Mcp-Session-Id: SESSION_ID" \
  -H "Content-Type: application/json" \
  -d '{"jsonrpc":"2.0","id":2,"method":"tools/call","params":{"name":"list_channels","arguments":{}}}'
```

:::warning Limites do transporte HTTP
- É uma implementação própria de JSON-RPC sobre `POST`, sem stream SSE nem `GET /mcp`. Ela atende o
  [Playground](playground.mdx) e chamadas diretas; a compatibilidade com clientes que esperam o
  transporte *Streamable HTTP* completo do MCP não foi verificada. Para Claude Desktop e Claude Code,
  use stdio.
- Todas as sessões usam a **mesma** credencial do Linktor (a do ambiente do processo). Quem alcança
  `POST /mcp` opera o seu tenant. Não exponha a porta sem `MCP_HTTP_AUTH_TOKEN`, e mantenha
  `MCP_HTTP_HOST=127.0.0.1` ou coloque o serviço atrás de um proxy com TLS.
:::

## Como os erros aparecem

Quando a API do Linktor recusa a chamada, a tool devolve `isError: true` com o erro em JSON:

```json
{
  "error": true,
  "code": "FORBIDDEN",
  "message": "API key is missing the required scope: channels:write"
}
```

| Sintoma | Causa provável |
|---|---|
| `UNAUTHORIZED` em toda tool | `LINKTOR_API_KEY` ausente, incorreta ou revogada |
| `FORBIDDEN` com `missing the required scope` | A chave não tem o scope da rota |
| Erro interno com `Unexpected token` (falha ao interpretar JSON), depois de alguns segundos | Tool que chama rota inexistente (veja as tabelas) ou `LINKTOR_API_URL` sem `/api/v1`. A API responde `404` em texto puro, o servidor MCP não consegue interpretar e ainda repete a chamada `LINKTOR_MAX_RETRIES` vezes antes de desistir |
| `UNKNOWN_ERROR` / `Request failed with status 200` | Tool cuja rota responde sem o campo `success` (`list_bots`, `search_knowledge`) |
| `TIMEOUT` | A API não respondeu em `LINKTOR_TIMEOUT` ms |
| O cliente não lista o servidor | Caminho de `dist/index.js` errado, build não executado, ou uso do binário `linktor-mcp` |

Os códigos de erro da API estão em [Erros e paginação](../api/erros-e-paginacao.md).
