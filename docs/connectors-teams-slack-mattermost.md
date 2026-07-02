# Conectores Teams / Slack / Mattermost — Setup & Operação

Runbook de configuração dos três conectores e do pipeline de webhook de saída
(`linktor-channel-v1`) consumido pelo DeskLenz. Todas as credenciais ficam em
`channel.credentials` (cifradas em repouso, AES-256-GCM, prefixo `enc:v1:`) —
nunca commitadas. As chaves de credencial abaixo batem exatamente com o backend.

## Contrato de integração com o DeskLenz

### Inbound (Linktor → DeskLenz)
Eventos `message.received` e `message.{sent,delivered,read,failed}` são entregues
como envelopes `linktor-channel-v1` assinados:

- Headers: `X-Linktor-Signature` (HMAC-SHA256 hex do corpo, chave =
  `credentials.webhook_secret`) e `X-Linktor-Timestamp` (epoch s; tolerância 300 s).
- Endpoint por canal: `channel.webhook_url` (campo top-level, configurável no admin).
- **Entrega durável**: o evento é enfileirado no stream NATS `LINKTOR_WEBHOOKS`; um
  worker assina com timestamp fresco a cada tentativa e entrega com retry/DLQ. Um
  crash entre evento e entrega não perde o webhook. A assinatura é recalculada na
  hora do envio, então redeliveries permanecem dentro da janela anti-replay.

### Outbound (DeskLenz → Linktor)
`POST /api/v1/conversations/{conversationId}/messages` autenticado com **uma de**:

- `Authorization: Bearer <jwt>` (sessão/serviço), ou
- `X-API-Key: <chave do canal/tenant>` — crie a chave no admin
  (`/api/v1/api-keys`, papel admin/owner). A chave é mostrada **uma única vez**.
  Requests com `X-API-Key` agem com papel `api` (rotas comuns; não admin-only).

## Microsoft Teams (Bot Framework)

Credenciais (`channel.credentials`): `app_id`, `app_password`, `tenant_id`,
`service_url` (descoberto no 1º inbound; pode ser semeado).

1. Registre um Azure Bot / App Registration. Anote App ID + secret.
2. Single-tenant: `tenant_id` = GUID do tenant do cliente. Endpoint de mensageria
   do bot = `https://<linktor>/api/v1/webhooks/teams/<channelId>`.
3. Multi-tenant (1 app → N clientes): `tenant_id` = `common` (ou vazio) no canal
   "multi"; cada org pode ainda ter um canal com `tenant_id` = GUID exato (vence o
   match). Endpoint **único** do bot = `https://<linktor>/api/v1/webhooks/teams`
   (sem channelId) — o canal é resolvido por `app_id` (audiência do token) +
   `tenant_id` AAD da Activity, e o JWT é validado (JWKS) contra o `app_id` resolvido.
4. Mídia inbound atrás de URL autenticada é baixada com o token AAD e re-hospedada
   no media store (ver abaixo).

## Slack (Events API + Web API)

Credenciais: `bot_token` (`xoxb-`), `signing_secret`, `app_id`, `bot_user_id`.

1. Crie um Slack App, habilite Events API e assine `message.*`.
2. Request URL: `https://<linktor>/api/v1/webhooks/slack/<channelId>` (responde ao
   handshake `url_verification`). Assinatura `X-Slack-Signature` (v0) verificada.
3. `url_private` é baixado com o bot token e re-hospedado (consumidor externo não
   acessa `url_private` diretamente).
4. Socket Mode não é usado (Events API público); é follow-up, não bloqueia.

## Mattermost (self-hosted, WebSocket)

Credenciais: `base_url` (por canal!), `bot_token` (PAT), `bot_user_id`.

1. Crie um bot account e um Personal Access Token.
2. Linktor mantém um WebSocket de saída para `{base_url}/api/v4/websocket`
   (evento `posted`, anti-eco por `bot_user_id`, reconexão com backoff). O
   handshake de auth é validado: PAT inválido falha explicitamente. Se
   `bot_user_id` não for informado, é resolvido automaticamente via
   `/api/v4/users/me` no connect (anti-eco continua funcionando).
3. Arquivos são baixados via `/api/v4/files/{id}` (PAT) e re-hospedados.
4. Reachability de rede Linktor → servidor Mattermost é pré-requisito de deploy.

## Media store (re-hospedagem de mídia inbound)

Anexos atrás de URLs autenticadas (Slack `url_private`, Mattermost file API, Teams
Bot Connector) são baixados e re-hospedados para o DeskLenz conseguir buscá-los.

- **MinIO/S3** (preferido): `MINIO_ENDPOINT`, `MINIO_ACCESS_KEY`, `MINIO_SECRET_KEY`,
  `MINIO_BUCKET` (default `linktor-media`), `MINIO_USE_SSL`. URLs pré-assinadas
  válidas ~7 dias — o `MINIO_ENDPOINT` precisa ser **acessível pelo consumidor**
  (use endpoint público quando o DeskLenz roda fora da rede do compose).
- **Local** (fallback): `MEDIA_UPLOAD_DIR` (+ `MEDIA_UPLOAD_BASE_URL`).
- Sem nenhum: re-hospedagem desligada, mantém a URL do provedor.

## Testar conexão

Cada tela de configuração tem um botão **Testar conexão** (e endpoints
`POST /channels/test-{teams,slack,mattermost}`) que valida as credenciais contra o
provedor de verdade (timeout 10 s): Teams adquire um token AAD (client-credentials),
Slack chama `auth.test`, Mattermost chama `/api/v4/users/me`.

## Edição de canal

Campos **não-secretos** (ex.: `base_url`, `app_id`, `tenant_id`, `bot_user_id`)
são pré-preenchidos no formulário de edição. **Segredos** (`*_token`,
`*_password`, `signing_secret`, `webhook_secret`) nunca voltam ao cliente e ficam
em branco no formulário — deixá-los em branco ao salvar **preserva** o valor
armazenado (não apaga); para trocar um segredo, basta digitar o novo valor.

## Limitações conhecidas (v1)

- **Conteúdo rico** = passthrough: Teams Adaptive Cards, Slack Blocks e quick
  replies viram texto/anexos. Interatividade (botões, slash commands) não tratada.
- **Mídia outbound**: Slack e Mattermost enviam upload **nativo** (arquivo real);
  Teams ainda entrega como link. Falha no upload nativo cai para link.
- **Recibos delivered/read inbound** dependem do provedor; só `message.sent` é
  emitido no envio.
