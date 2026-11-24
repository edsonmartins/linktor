---
sidebar_position: 2
title: TypeScript
description: Referência do SDK TypeScript/JavaScript do Linktor (@linktor/sdk) — instalação a partir do repositório, cliente, canais, envio direto, webhook em Express, tempo real, erros e limitações conhecidas.
---

# SDK TypeScript

O pacote `@linktor/sdk` (pasta `sdks/typescript` do repositório) é um cliente HTTP sobre `axios` para
Node.js. Hoje ele está alinhado com a API em **canais** e **envio direto de mensagem**; os demais
recursos foram escritos contra um contrato antigo e funcionam só em parte. Esta página documenta o que
foi conferido contra o servidor e lista, no fim, o que não funciona.

:::warning Três comportamentos valem para o SDK inteiro
1. **Só `channels` desembrulha o envelope.** Todos os outros métodos devolvem o corpo cru da resposta,
   `{ success, data, meta }`, embora o tipo declarado seja o da entidade. `result.id` vem `undefined`;
   o valor está em `result.data.id`. Use o helper `data()` da seção [Criar o cliente](#criar-o-cliente).
2. **Só `channels` e `messages` têm tipos em snake_case**, como a API. Os outros modelos (`Conversation`,
   `Contact`, `Bot`…) declaram `contactId`, `createdAt`, mas o servidor devolve `contact_id`,
   `created_at`. Na leitura, use o nome snake_case; na escrita, campos camelCase são ignorados pelo servidor.
3. **A mensagem de erro do servidor se perde.** O SDK procura `message` na raiz do corpo, e a API manda
   em `error.message`. Veja [Tratamento de erros](#tratamento-de-erros).
:::

| Recurso | Cobertura | Observação |
|---|---|---|
| Canais (`client.channels`) | Sim | Criar, atualizar, conectar por QR, pair code, remover |
| Envio direto (`client.messages`) | Sim | `POST /messages/send`; resposta vem no envelope |
| Mensagem para uma conversa | Não | `sendMessage`/`sendText` mandam corpo que o servidor não entende; use `HttpClient` |
| Conversas | Parcial | `list`, `get`, `listMessages` |
| Contatos | Parcial | `list`, `search`, `get`, `create`, `delete` |
| Bots | Parcial | `list`, `get`, `delete` |
| IA (`client.ai`) | Não | Todas as rotas chamadas não existem |
| Base de conhecimento | Parcial | `list`, `get`, `delete` |
| Fluxos | Parcial | `list`, `get`, `delete` |
| Analytics | Parcial | `getConversationMetrics`, `getChannelMetrics` (sem filtro de data) |
| VRE | Parcial | `render` e `listTemplates` |
| Verificação de webhook | Não compatível | Assina só o corpo; o servidor assina `timestamp.corpo` |
| WebSocket (`client.ws`) | Não compatível | Autenticação e protocolo diferentes dos do servidor |
| Autenticação por API key | Sim | Header `X-API-Key` |

Visão geral dos SDKs: [SDKs](./visao-geral.md). Contrato HTTP completo: [API](../api/visao-geral.md).

## Instalação

Coordenadas no `package.json`: nome `@linktor/sdk`, versão `1.0.0`.

:::note
Não foi possível confirmar que `@linktor/sdk` está publicado no registry do npm. O caminho garantido é
instalar a partir do repositório `github.com/edsonmartins/linktor`, subpasta `sdks/typescript`. O npm não
instala pacote de subpasta de um repositório git, e a pasta `dist/` não é versionada: é preciso clonar e compilar.
:::

```bash
git clone https://github.com/edsonmartins/linktor.git
cd linktor/sdks/typescript
npm ci

# JavaScript (CommonJS + ESM)
npx tsup src/index.ts --format cjs,esm
# Declarações de tipo (.d.ts) — o tsc acusa erros, mas emite os arquivos
npx tsc --emitDeclarationOnly --declaration --declarationMap false --outDir dist || true

npm pack --ignore-scripts          # gera linktor-sdk-1.0.0.tgz
cd /caminho/do/seu-projeto
npm install /caminho/para/linktor/sdks/typescript/linktor-sdk-1.0.0.tgz
```

:::warning
`npm run build` (o script oficial, `tsup ... --dts`) hoje **falha na etapa de tipos** com erros de
`noUnusedLocals` e um erro de tipagem em `src/websocket/client.ts`. Os bundles JavaScript são gerados antes
da falha; por isso os dois comandos separados acima.
:::

## Requisitos

- Node.js 18 ou superior (`engines.node`).
- Dependências de runtime: `axios` ^1.6 e `ws` ^8.14. TypeScript 4.7+ é opcional.
- **Use no servidor, não no navegador.** O ponto de entrada único importa `crypto` (webhook) e `ws`
  (WebSocket), que são módulos de Node. Além disso, o CORS da API não libera o header `X-API-Key` em
  chamadas de outra origem, e uma chave de API não deve ir para o front-end.

## Criar o cliente {#criar-o-cliente}

O SDK não acrescenta `/api/v1`: ele concatena o caminho do recurso (`/channels`, `/messages/send`) direto
em `baseUrl`. Passe a base **com** `/api/v1`. O padrão `https://api.linktor.io` não serve para uma
instalação própria.

```typescript
import { LinktorClient } from '@linktor/sdk';

const client = new LinktorClient({
  baseUrl: `${process.env.LINKTOR_URL}/api/v1`, // https://linktor.suaempresa.com/api/v1
  apiKey: process.env.LINKTOR_API_KEY,          // lk_sua_chave
  timeout: 15000,
});

// Desembrulha o envelope { success, data } dos métodos que não o fazem.
export function data<T>(res: T): T {
  const env = res as unknown as { success?: boolean; data?: T };
  return env !== null && typeof env === 'object' && 'success' in env ? (env.data as T) : res;
}
```

| Opção | Tipo | Padrão | Efeito |
|---|---|---|---|
| `baseUrl` | `string` | `https://api.linktor.io` | Base das chamadas; inclua `/api/v1` |
| `apiKey` | `string` | — | Enviada em `X-API-Key` |
| `accessToken` | `string` | — | JWT de usuário, enviado em `Authorization: Bearer` |
| `timeout` | `number` (ms) | `30000` | Tempo limite por tentativa |
| `maxRetries` | `number` | `3` | Novas tentativas em 429, 5xx, falha de rede e timeout |
| `retryDelay` | `number` (ms) | `1000` | Base do backoff exponencial: `retryDelay * 2^tentativa` |
| `headers` | `Record<string, string>` | — | Headers extras em toda requisição |
| `onTokenRefresh` | `() => Promise<string>` | — | Chamado no primeiro 401; o retorno vira o novo Bearer e a chamada é repetida |
| `websocket` | objeto | — | Opções do cliente WebSocket (veja [Tempo real](#tempo-real)) |

`createClient(config)` equivale a `new LinktorClient(config)`. O cliente também tem `setApiKey(chave)`,
`setAccessToken(token)` e `close()`.

:::warning
`maxRetries: 0` e `retryDelay: 0` são tratados como "não informado" e voltam para `3` e `1000`. Não há como
desligar as novas tentativas; o mínimo efetivo é `maxRetries: 1`. As tentativas valem também para `POST` —
no envio de mensagem, use `idempotency_key` (veja [Enviar mensagem](#enviar-mensagem)).
:::

## Autenticação

| Configuração | Header enviado | Quando usar |
|---|---|---|
| `apiKey` | `X-API-Key: lk_...` | Integração servidor a servidor. É o caminho recomendado |
| `accessToken` | `Authorization: Bearer <jwt>` | Agir como um usuário do painel |

Se os dois forem informados, só a API key é enviada. A chave precisa dos scopes das rotas usadas
(`messages:send` para envio, `channels:write` para criar canal…); veja
[Chaves de API](../painel/chaves-de-api.md) e [Autenticação](../api/autenticacao.md).

`client.auth.login()` chama `POST /auth/login`, mas não instala o token: ele lê `accessToken` na raiz da
resposta e o servidor devolve `data.access_token`. Faça a instalação à mão:

```typescript
const jwtClient = new LinktorClient({ baseUrl: `${process.env.LINKTOR_URL}/api/v1` });

const res = await jwtClient.auth.login({ email: 'agente@exemplo.com', password: process.env.SENHA! });
const sessao = data(res) as unknown as { access_token: string; refresh_token: string; expires_in: number };
jwtClient.setAccessToken(sessao.access_token);
```

`client.auth.logout()` funciona (`POST /auth/logout`). Os demais métodos de `auth` estão em
[Limitações conhecidas](#limitacoes).

## Canais

Todos os métodos abaixo devolvem a entidade já desembrulhada, com campos em snake_case.

```typescript
list(params?: ListChannelsParams): Promise<Channel[]>
get(id: string): Promise<Channel>
create(data: CreateChannelInput): Promise<Channel>
update(id: string, data: UpdateChannelInput): Promise<Channel>
delete(id: string): Promise<void>
connect(id: string): Promise<ConnectResult>
requestPairCode(id: string, phoneNumber: string): Promise<ConnectResult>
disconnect(id: string): Promise<Channel>
```

`CreateChannelInput`: `name`, `type` (obrigatórios), `identifier`, `config`, `credentials`, `webhook_url`.
`config` e `credentials` são mapas de string para string; `credentials` nunca volta nas respostas.
`ConnectResult`: `channel`, `qr_code`, `expires_in` (segundos), `pair_code`, `passkey_required`, `passkey_challenge`.

### Criar e conectar por QR

O QR expira em `expires_in` segundos; chame `connect` de novo para obter um novo ou para descobrir que o
aparelho já foi vinculado. Detalhes do canal: [WhatsApp não oficial](../canais/whatsapp-nao-oficial.md).

```typescript
const canal = await client.channels.create({
  type: 'whatsapp', // não oficial
  name: 'Atendimento',
});

const sleep = (ms: number) => new Promise((r) => setTimeout(r, ms));

for (let tentativa = 0; tentativa < 10; tentativa++) {
  const res = await client.channels.connect(canal.id);

  if (res.channel?.connection_status === 'connected') {
    console.log('Canal conectado');
    break;
  }
  if (res.qr_code) {
    console.log('Escaneie este QR no WhatsApp:', res.qr_code); // renderize com uma lib de QR
  }
  await sleep((res.expires_in ?? 20) * 1000);
}
```

Alternativa ao QR, o código de pareamento:

```typescript
const par = await client.channels.requestPairCode(canal.id, '+5511999999999');
console.log('Digite no WhatsApp:', par.pair_code);
```

### Atualizar e configurar o webhook de saída

`update` usa `PUT`. A URL de entrega é `webhook_url`; o segredo do HMAC vai em
`credentials.webhook_secret`; o filtro de eventos, em `config.webhook_events` (lista separada por vírgula;
vazio entrega todos).

```typescript
await client.channels.update(canal.id, {
  name: 'Atendimento — matriz',
  webhook_url: 'https://app.suaempresa.com/webhooks/linktor',
  credentials: { webhook_secret: process.env.LINKTOR_WEBHOOK_SECRET! },
  config: { webhook_events: 'message.received,message.sent,message.failed' },
});
```

:::warning
`credentials` é mesclado com o que já está guardado, mas `config` **substitui** o mapa atual (só as chaves
sensíveis são preservadas). Ao mandar `config`, mande todas as chaves não secretas que o canal usa — leia
antes com `client.channels.get(id)`.
:::

### Desconectar e remover

```typescript
await client.channels.disconnect(canal.id); // o retorno real é { message }, não um Channel
await client.channels.delete(canal.id);     // 204, sem corpo
```

Os filtros de `list(params)` (`type`, `status`, `search`) são enviados, mas o servidor devolve sempre todos
os canais do tenant; filtre no cliente.

## Enviar mensagem {#enviar-mensagem}

`client.messages` chama `POST /messages/send`: você informa canal e destinatário, e o Linktor resolve ou
cria contato e conversa. Exige o scope `messages:send`. A resposta é `202` com `status: "queued"`; o
resultado da entrega chega depois pelo webhook (`message.sent`, `message.failed`). Contrato completo em
[Enviar mensagens](../api/enviar-mensagens.md).

```typescript
sendDirect(data: DirectSendInput): Promise<DirectSendResult>
sendText(channelId: string, to: string, text: string, metadata?: Record<string, string>): Promise<DirectSendResult>
```

`DirectSendInput`: `channel_id`, `to`, `text`, `content_type` (só `'text'`), `metadata`.

```typescript
const res = await client.messages.sendText(
  'CHANNEL_ID',
  '+5511999999999',
  'Olá! Seu pedido saiu para entrega.',
  { idempotency_key: 'pedido-1042-saiu' },
);

const envio = data(res); // { id, conversation_id, channel_id, status: 'queued' }
console.log(envio.id, envio.status);
```

:::warning
O retorno é o envelope inteiro; sem `data()`, `res.id` é `undefined`. Como o SDK repete `POST` em 5xx e
falha de rede, mande `metadata.idempotency_key` (único por tenant): uma repetição com a mesma chave
devolve a mensagem original em vez de enviar outra. Chaves internas em `metadata` (`tenant_id`,
`conversation_id`, `campaign_id`…) são recusadas com 400.
:::

O tipo `DirectSendInput` não declara `attachments`. Para mandar mídia, ou para enviar numa conversa já
conhecida, use o `HttpClient` da próxima seção.

## Chamar rotas que o SDK não cobre {#http-client}

O SDK exporta o `HttpClient` que os recursos usam por dentro. Ele manda o mesmo header de autenticação,
aplica as mesmas novas tentativas e lança os mesmos erros. Devolve sempre o corpo cru (envelope).

```typescript
import { HttpClient } from '@linktor/sdk';

const http = new HttpClient({
  baseUrl: `${process.env.LINKTOR_URL}/api/v1`,
  apiKey: process.env.LINKTOR_API_KEY,
});

type Envelope<T> = { success: boolean; data: T };

// Mensagem para uma conversa existente
const msg = await http.post<Envelope<{ id: string; status: string }>>(
  '/conversations/CONVERSATION_ID/messages',
  { content_type: 'text', content: 'Seu pedido foi confirmado.' },
);
console.log(msg.data.id);

await http.post('/conversations/CONVERSATION_ID/assign', { user_id: 'USER_ID' });
await http.post('/conversations/CONVERSATION_ID/resolve');
```

Métodos: `get`, `post`, `put`, `patch`, `delete`; use `config.params` para a query string.

## Conversas

Funcionam `list(params?)`, `get(id)` e `listMessages(conversationId, params?)`.

```typescript
const pagina = await client.conversations.list({ status: 'open', limit: 20 });

// pagina.data é a lista; os itens chegam em snake_case
type ConversaWire = { id: string; contact_id: string; channel_id: string; status: string; unread_count: number };
for (const c of pagina.data as unknown as ConversaWire[]) {
  console.log(c.id, c.contact_id, c.status);
}

const mensagens = await client.conversations.listMessages('CONVERSATION_ID', { limit: 50 });
```

:::warning
- `pagina.pagination` não existe: a paginação vem em `meta` (`has_next`, `next_cursor`). Para a próxima
  página, repita a chamada com `before: <next_cursor>` — em `list`, passe por `HttpClient`, porque
  `ListConversationsParams` não declara `before`.
- Dos filtros tipados, o servidor entende `status`, `search` e `limit`. `channelId`, `assignedTo`,
  `contactId`, `tags`, `sortBy` são ignorados (a API espera `channel_id`, `assigned_to`).
- `iterate()` entrega a primeira página e depois lança `TypeError`.
:::

## Contatos

Funcionam `list(params?)`, `search(query, params?)`, `get(id)`, `create(data)` e `delete(id)`.

```typescript
const novo = data(await client.contacts.create({
  name: 'Maria Souza',
  email: 'cliente@exemplo.com',
  phone: '+5511999999999',
  tags: ['vip'],
}));

const achados = await client.contacts.search('Maria');
console.log(achados.data.length);

await client.contacts.delete(novo.id);
```

:::warning
- Em `create`, `avatarUrl`, `customFields`, `externalId` e `identifiers` são ignorados pelo servidor, que
  espera `avatar_url` e `custom_fields`. Para esses campos, e para atualizar (`PUT /contacts/:id`), use o
  `HttpClient`. No `PUT`, campo omitido vira string vazia: mande o contato inteiro.
- A API pagina contatos por `page` e `page_size`; `limit`, `offset` e `cursor` são ignorados.
- **Não use `findByEmail` nem `findByPhone`.** O servidor ignora os filtros `email` e `phone`, e os métodos
  devolvem o primeiro contato do tenant, seja qual for. Use `search()`.
:::

## Bots, base de conhecimento e fluxos

Em `client.bots`, `client.knowledgeBases` e `client.flows` funcionam `list(params?)`, `get(id)` e
`delete(id)`. Todos devolvem o envelope, com itens em snake_case.

`create` e `update` desses recursos não funcionam: os tipos de entrada não têm os campos obrigatórios da
API (`provider` e `model` no bot; `type` na base; `trigger` e `start_node_id` no fluxo) e `update` usa
`PATCH`, que não existe. Crie e edite com o `HttpClient`; o contrato está em
[API](../api/visao-geral.md). `client.ai` não tem nenhum método funcional.

## Analytics

`getConversationMetrics(range?)` (`GET /analytics/conversations`) e `getChannelMetrics(range?)`
(`GET /analytics/channels`) respondem, exigem o scope `analytics:read` e devolvem o envelope. O `range` vai como
`startDate`/`endDate`, mas a API espera `start_date`/`end_date` e `period`: o filtro é ignorado e vale o
período padrão (semanal). Os tipos `ConversationMetrics` e `ChannelMetrics` não descrevem a resposta real.
Para filtrar, use `http.get('/analytics/conversations', { params: { period: 'monthly' } })`.

## VRE

As rotas `/vre/*` só existem quando o serviço VRE está habilitado no servidor e **não usam o envelope**:
a resposta é o objeto direto, então `data()` não é necessário.

Funcionam `render(request)` e `listTemplates()`.

```typescript
const img = await client.vre.render({
  tenantId: '', // deixe vazio: o tenant vem da chave; um valor diferente dá 403
  templateId: 'menu_opcoes',
  data: { titulo: 'Como posso ajudar?', opcoes: [{ label: 'Ver pedidos' }, { label: 'Falar com atendente' }] },
  channel: 'whatsapp',
});

const wire = img as unknown as { image_base64: string; render_time_ms: number; size_bytes: number };
console.log(img.caption, img.width, img.height, wire.image_base64.length);
```

:::warning
`caption`, `width`, `height` e `format` chegam como tipados. `imageBase64`, `renderTimeMs`, `sizeBytes` e
`cacheHit` vêm `undefined`: os nomes reais são `image_base64`, `render_time_ms`, `size_bytes`, `cache_hit`.
O mesmo vale para os atalhos `renderMenu`, `renderProductCard`, `renderOrderStatus`, `renderProductList`,
`renderConfirmation` e `renderPixPayment`, que chamam `render`.
:::

## Verificar a assinatura do webhook

O Linktor assina cada entrega com HMAC-SHA256 do texto `<timestamp>.<corpo cru>`, usando o
`credentials.webhook_secret` do canal. O resultado vai em hexadecimal, sem prefixo, no header
`X-Linktor-Signature`; o timestamp (epoch em segundos) vai em `X-Linktor-Timestamp`. Eventos e formato do
payload: [Receber eventos](../api/receber-eventos.md).

:::warning O verificador do SDK não é compatível
`verifyWebhook`, `verifyWebhookSignature`, `constructEvent` e `createWebhookHandler` (também em
`client.webhooks`) calculam o HMAC **só do corpo**, sem o timestamp. Contra uma entrega real do Linktor
eles devolvem sempre `false`. Faça a verificação manual abaixo.
:::

```typescript
import express from 'express';
import { createHmac, timingSafeEqual } from 'node:crypto';

const SECRET = process.env.LINKTOR_WEBHOOK_SECRET!;
const TOLERANCIA_S = 300;

function assinaturaValida(corpo: Buffer, assinatura: string, timestamp: string): boolean {
  const ts = Number.parseInt(timestamp, 10);
  if (!assinatura || Number.isNaN(ts)) return false;
  if (Math.abs(Math.floor(Date.now() / 1000) - ts) > TOLERANCIA_S) return false;

  const esperado = createHmac('sha256', SECRET).update(`${timestamp}.`).update(corpo).digest();
  const recebido = Buffer.from(assinatura, 'hex');
  return recebido.length === esperado.length && timingSafeEqual(recebido, esperado);
}

const app = express();

// express.raw entrega o corpo como Buffer, byte a byte como foi assinado.
app.post('/webhooks/linktor', express.raw({ type: 'application/json' }), (req, res) => {
  const assinatura = req.header('X-Linktor-Signature') ?? '';
  const timestamp = req.header('X-Linktor-Timestamp') ?? '';

  if (!assinaturaValida(req.body as Buffer, assinatura, timestamp)) {
    res.status(401).send('assinatura inválida');
    return;
  }

  const evento = JSON.parse((req.body as Buffer).toString('utf8')) as {
    id: string; type: string; timestamp: string; tenantId: string; environment: string; data: any;
  };

  if (evento.type === 'message.received') {
    console.log(evento.data.conversationId, evento.data.message.content?.text);
  }

  res.sendStatus(200);
});

app.listen(3000);
```

Não registre `express.json()` antes dessa rota: com o corpo já interpretado, a assinatura não confere. O
envelope do webhook é camelCase (`tenantId`, `conversationId`, `contentType`), diferente da API REST.

## Tempo real (WebSocket) {#tempo-real}

:::warning `client.ws` não funciona com o servidor atual
- Com `apiKey`, o cliente manda `?api_key=...`; o WebSocket só aceita **JWT de usuário**.
- Com `accessToken`, ele manda só `?token=...`. A rota `/api/v1/ws` passa antes pelo middleware de
  autenticação, que exige o header `Authorization` (ou cookie), e o cliente do SDK não envia headers.
- Ele espera eventos `message`, `message_status` e `conversation_update` com campos na raiz; o servidor
  emite `new_message`, `message_updated`, `conversation_updated`, `conversation_created`, `typing`,
  `presence` com os dados em `payload`. Os handlers nunca disparam.
- `subscribe`, `unsubscribe` e o ping em JSON não existem no servidor, que transmite tudo do tenant.
:::

Use o pacote `ws` direto, com o JWT no header e na query string:

```typescript
import WebSocket from 'ws';

const jwt = process.env.LINKTOR_ACCESS_TOKEN!; // obtido em POST /auth/login (data.access_token)
const url = `${process.env.LINKTOR_URL!.replace(/^http/, 'ws')}/api/v1/ws?token=${encodeURIComponent(jwt)}`;

const ws = new WebSocket(url, { headers: { Authorization: `Bearer ${jwt}` } });

ws.on('message', (raw) => {
  const { type, payload } = JSON.parse(raw.toString()) as { type: string; payload: any };
  if (type === 'new_message') console.log(payload.conversation_id, payload.message.content);
  if (type === 'typing') console.log(payload.user_id, payload.is_typing);
});

ws.on('open', () => {
  ws.send(JSON.stringify({ type: 'typing', payload: { conversation_id: 'CONVERSATION_ID', is_typing: true } }));
});
```

Para integração servidor a servidor, prefira o webhook: ele aceita API key e tem nova tentativa de
entrega. Protocolo completo em [Tempo real](../api/tempo-real.md).

## Tratamento de erros {#tratamento-de-erros}

Toda falha HTTP vira uma subclasse de `LinktorError`, escolhida pelo status:

| Classe | Quando | `code` | Nova tentativa automática |
|---|---|---|---|
| `ValidationError` | 400 | `VALIDATION_ERROR` | Não |
| `AuthenticationError` | 401 | `AUTHENTICATION_ERROR` | Não (só `onTokenRefresh`) |
| `AuthorizationError` | 403 (scope ou papel insuficiente) | `AUTHORIZATION_ERROR` | Não |
| `NotFoundError` | 404 | `NOT_FOUND` | Não |
| `ConflictError` | 409 | `CONFLICT_ERROR` | Não |
| `RateLimitError` | 429 | `RATE_LIMIT_ERROR` | Sim |
| `ServerError` | 500, 502, 503, 504 | `SERVER_ERROR` | Sim |
| `TimeoutError` | Estouro de `timeout` | `TIMEOUT_ERROR` | Sim |
| `NetworkError` | Sem resposta do servidor | `NETWORK_ERROR` | Sim |
| `LinktorError` | Outros status | `UNKNOWN_ERROR` | Não |

Campos: `message`, `code`, `statusCode`, `requestId` (header `X-Request-ID` da resposta), `details`.

```typescript
import { LinktorError, AuthorizationError } from '@linktor/sdk';

try {
  await client.messages.sendText('CHANNEL_ID', '+5511999999999', 'Olá!');
} catch (err) {
  if (err instanceof AuthorizationError) {
    console.error('A chave não tem o scope messages:send');
  } else if (err instanceof LinktorError) {
    console.error(err.name, err.statusCode, err.requestId);
  } else {
    throw err;
  }
}
```

:::warning
O SDK lê `code`, `message` e `details` na raiz do corpo, mas a API responde
`{ "success": false, "error": { "code", "message", "details" } }`. Na prática `err.message` é sempre
`"Unknown error"` (ou `"Resource not found"` no 404), `err.code` é o valor fixo da tabela e `err.details`
vem vazio; `RateLimitError.retryAfter` nunca é preenchido. Decida pelo tipo e por `statusCode`, e use
`requestId` para achar a causa no log do servidor. Códigos reais em
[Erros e paginação](../api/erros-e-paginacao.md).
:::

## Limitações conhecidas {#limitacoes}

| Método | O que faz | Por que não funciona | Alternativa |
|---|---|---|---|
| `conversations.sendMessage`, `sendText`, `sendImage`, `sendDocument`, `sendLocation` | `POST /conversations/:id/messages` com `{ text }` ou `{ media }` | A API espera `content_type`, `content`, `attachments` | [`HttpClient`](#http-client) ou `messages.sendText` |
| `conversations.update`, `resolve`, `reopen` | `PATCH /conversations/:id` | Não há `PATCH` | `http.put(...)`, `http.post('.../resolve')`, `.../reopen` |
| `conversations.assign` | `POST .../assign` com `{ agentId }` | A API espera `user_id` | `http.post('.../assign', { user_id })` |
| `conversations.unassign`, `snooze`, `addTags`, `removeTags`, `markAsRead`, `getCounts`, `getMessage` | Rotas `/unassign`, `/snooze`, `/tags`, `/read`, `/counts`, `/messages/:id` da conversa | Rotas inexistentes | Tags por `PUT /conversations/:id`; mensagem por `GET /messages/:id` |
| `conversations.iterate`, `contacts.iterate` | Percorre páginas por `pagination.nextCursor` | A resposta traz `meta`, não `pagination` | Paginar à mão |
| `contacts.update`, `updateCustomFields` | `PATCH /contacts/:id` | Não há `PATCH` | `http.put('/contacts/:id', ...)` |
| `contacts.findByEmail`, `findByPhone` | `GET /contacts?email=` / `?phone=` | Filtro ignorado; devolve um contato qualquer | `contacts.search()` |
| `contacts.merge`, `getConversations`, `addTags`, `removeTags` | `/contacts/merge`, `/:id/conversations`, `/:id/tags` | Rotas inexistentes | — |
| `channels.getCapabilities` | `GET /channels/:id/capabilities` | Rota inexistente | — |
| `bots.create`, `update`, `activate`, `deactivate`, `assignToChannels` | `POST`/`PATCH /bots`, `{ channelIds }` | Faltam `provider` e `model`; sem `PATCH`; a API espera `channel_id` | `HttpClient`: `PUT /bots/:id`, `POST /bots/:id/activate`, `POST /bots/:id/channels` |
| `bots.removeFromChannels`, `assignKnowledgeBases`, `removeKnowledgeBases`, `getStats`, `duplicate` | Rotas de bot | Rotas inexistentes | `DELETE /bots/:id/channels/:channelId` |
| `ai.agents.*`, `ai.completions.*`, `ai.embeddings.*` | `/ai/agents`, `/ai/completions`, `/ai/embeddings` | Rotas inexistentes; não há streaming | `http.post('/ai/complete', { provider, model, messages })` |
| `knowledgeBases.create`, `update` | `POST`/`PATCH /knowledge-bases` | Falta `type`; sem `PATCH` | `HttpClient` com `POST`/`PUT` |
| `knowledgeBases.query`, `search`, `reprocess`, `getStats` e todos os `*Document*` | `/query`, `/documents`, `/reprocess`, `/stats` | Rotas inexistentes | `POST /knowledge-bases/:id/search`, `/items` |
| `flows.create`, `update`, `activate`, `deactivate` | `POST`/`PATCH /flows` | Faltam `trigger` e `start_node_id`; sem `PATCH` | `HttpClient`: `PUT /flows/:id`, `POST /flows/:id/activate` |
| `flows.duplicate`, `execute`, `getExecutions`, `getExecution`, `cancelExecution`, `getNodeTypes`, `validate`, `export`, `import`, `getStats` | Rotas de fluxo | Rotas inexistentes | `POST /flows/:id/test` |
| `analytics.getDashboard`, `getMessageMetrics`, `getAgentMetrics`, `getBotMetrics`, `generateReport`, `export`, `getRealtime` | `/analytics/dashboard`, `/messages`, `/agents`, `/bots`, `/reports`, `/export`, `/realtime` | Rotas inexistentes | `GET /analytics/overview`, `/flows`, `/escalations` |
| `vre.renderAndSend` | `POST /vre/render-and-send` com `conversation_id` | A API exige `send_to`, `channel` e `channel_id` | `HttpClient` com esses campos |
| `vre.preview` | `POST /vre/templates/:id/preview` | A rota é `GET` | `http.get('/vre/templates/:id/preview')` |
| `auth.refreshToken` | `POST /auth/refresh` com `{ refreshToken }` | A API espera `refresh_token` | `http.post('/auth/refresh', { refresh_token })` |
| `auth.getCurrentUser`, `updateProfile`, `changePassword` | `/auth/me`, `/auth/change-password` | Rotas inexistentes | `GET /me`, `PUT /me`, `PUT /me/password` |
| `client.webhooks.*`, `verifyWebhook`, `constructEvent`, `createWebhookHandler` | HMAC só do corpo | O servidor assina `timestamp.corpo` | Verificação manual |
| `client.ws` | WebSocket com `subscribe` | Autenticação e protocolo diferentes | Pacote `ws` direto |

## Referência: método → rota HTTP

| Método | Rota | Retorno real |
|---|---|---|
| `channels.list()` | `GET /channels` | `Channel[]` |
| `channels.get(id)` | `GET /channels/:id` | `Channel` |
| `channels.create(data)` | `POST /channels` | `Channel` |
| `channels.update(id, data)` | `PUT /channels/:id` | `Channel` |
| `channels.delete(id)` | `DELETE /channels/:id` | nada (204) |
| `channels.connect(id)` | `POST /channels/:id/connect` | `ConnectResult` |
| `channels.requestPairCode(id, phone)` | `POST /channels/:id/pair` | `ConnectResult` |
| `channels.disconnect(id)` | `POST /channels/:id/disconnect` | `{ message }` |
| `messages.sendDirect(data)` / `sendText(...)` | `POST /messages/send` | envelope com `DirectSendResult` |
| `conversations.list(params)` | `GET /conversations` | envelope com lista + `meta` |
| `conversations.get(id)` | `GET /conversations/:id` | envelope |
| `conversations.listMessages(id, params)` | `GET /conversations/:id/messages` | envelope com lista + `meta` |
| `contacts.list(params)` / `search(q)` | `GET /contacts` | envelope com lista + `meta` |
| `contacts.get(id)` | `GET /contacts/:id` | envelope |
| `contacts.create(data)` | `POST /contacts` | envelope |
| `contacts.delete(id)` | `DELETE /contacts/:id` | nada (204) |
| `bots.list()` / `get(id)` / `delete(id)` | `GET /bots`, `GET`/`DELETE /bots/:id` | envelope (`delete`: ignore o retorno) |
| `knowledgeBases.list()` / `get(id)` / `delete(id)` | `GET /knowledge-bases`, `GET`/`DELETE /knowledge-bases/:id` | envelope (`delete`: ignore o retorno) |
| `flows.list()` / `get(id)` / `delete(id)` | `GET /flows`, `GET`/`DELETE /flows/:id` | envelope (`delete`: ignore o retorno) |
| `analytics.getConversationMetrics()` | `GET /analytics/conversations` | envelope |
| `analytics.getChannelMetrics()` | `GET /analytics/channels` | envelope |
| `vre.render(req)` | `POST /vre/render` | objeto cru, snake_case |
| `vre.listTemplates()` | `GET /vre/templates` | `{ templates }` |
| `auth.login(cred)` | `POST /auth/login` | envelope; token não é instalado |
| `auth.logout()` | `POST /auth/logout` | envelope |
