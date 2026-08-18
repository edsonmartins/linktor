# Changelog

Todas as mudanças notáveis deste projeto são documentadas aqui.

O formato é baseado em [Keep a Changelog](https://keepachangelog.com/pt-BR/1.1.0/)
e o projeto adere ao [Versionamento Semântico](https://semver.org/lang/pt-BR/).

## [Não lançado]

### Bridge de WhatsApp (gateway em device)

Novo subsistema `internal/gateway` que permite hospedar a sessão WhatsApp
Multi-Device **no device do usuário** (Android/PC) e conectar de volta ao
Linktor via WebSocket — resolvendo o bloqueio de localização/geolocalização de
APIs não-oficiais.

- **`Hub` de bridges**: mantém `channelID → bridge` conectada; roteia outbound
  com ack correlacionado (`correlation_id`, timeout 60s).
- **`RemoteAdapter`**: implementa `plugin.ChannelAdapter`; registrado por canal
  no registry global enquanto a bridge está online, então o `outbound.Worker`
  resolve o envio pelo device sem mudanças no pipeline de entrega.
- **Inbound reutiliza o pipeline**: `internal/gateway` publica mensagens
  recebidas no subject `linktor.messages.inbound.<tipo>` (consumidor wildcard já
  existente processa sem alterações).
- **Endpoints**:
  - `POST /api/v1/channels/:id/bridge-token` (admin/owner) — gera/rotaciona o
    token do canal, armazenado cifrado em `credentials.bridge_token`.
  - `GET /api/v1/gateways/ws?channel_id=..&token=..` — conexão da bridge
    (autenticada pelo token do canal, não por JWT).
- **Escopo atual**: texto outbound/inbound. Mídia, interativos e reações sobre a
  bridge ficam como próximos milestones (falha permanente até lá).
- **Heartbeat e saúde da bridge**: frame de aplicação `ping`/`pong` (a bridge
  envia status + timestamp a cada 20s; o servidor registra `last_seen`).
  `GET /api/v1/channels/:id/bridge-health` expõe `online`, `platform`, `version`,
  `connected_at`, `last_seen_ping`, `stale` (sem heartbeat > 90s) e o último
  status da sessão reportado pelo device.
- **Saúde embutida na API de canais**: `GET /api/v1/channels` e
  `GET /api/v1/channels/:id` retornam `bridge_health` inline (quando há bridge)
  para o admin ver online/stale/last_seen sem chamada extra.
- **Identidade de contato no inbound da bridge**: `sender_id` só com os dígitos
  (JID.User) + `sender_jid`/`chat_jid` em metadata, igual ao adapter embutido;
  `sender_name` (pushName) propagado para o contato não nascer como
  "Unknown Contact".

Protocolo documentado em `bridges/docs/PROTOCOL.md` (repo bridges).

### Próximos milestones (bridge)

- Mídia (imagem/documento/vídeo/áudio) e interativos (botões/listas) pelo protocolo;
- UX de registro: criar canal + gerar token direto no admin;
- Rehost de mídia inbound no device → object storage;
- Teste E2E ao vivo com device real.

### Canais — WhatsApp não-oficial (whatsmeow)

Enriquecimento do canal não-oficial (`internal/adapters/whatsapp`) e novo motor
de chamadas (`internal/voip`), aproveitando código do projeto wacalls-chat.

- **Mensagens interativas native-flow:** botões de resposta rápida e listas de
  seleção que renderizam no WhatsApp Web multi-device (com o AdditionalNode
  `<biz>`), fallback automático para texto e parsing das respostas para
  `selected_id`. `SupportsInteractive=true`.
- **Unwrap de envelopes no inbound:** desembrulho recursivo de mensagens
  efêmeras, view-once (v1/v2/v2ext), device-sent, editadas e protocol antes de
  classificar; fallback de texto para anúncios CTWA (matchedText), live-location
  e contacts-array; flag de edição.
- **Edit / revoke / forward** de mensagens. `SupportsForwarding=true`.
- **Resolução LID↔PN:** senders com identidade `@lid` resolvidos para número de
  telefone no inbound (cache TTL), com o LID preservado em metadata; avatar com
  `ExistingID` para pular download inalterado.
- **Chamadas de voz/vídeo nativas (VoIP):** motor portado (CallManager, codec
  **mlow**, media/SRTP, signaling, transport) atrás de uma interface
  `VoipSocket` sobre o whatsmeow; `PlaceCall`/`AcceptCall`/`RejectCall`/`EndCall`,
  eventos de ciclo de vida via `SetCallHandler` e hook de stream via
  `SetCallAudioSink`.
- **Gravação de chamadas em WAV**, ligável por config (`record_calls`,
  `recordings_dir`): estéreo (esquerda = interlocutor, direita = local) ou mono
  para escuta; a captura só copia PCM em memória e escreve no fim da ligação, sem
  interferir na chamada. O evento `ended` carrega `recording_path`.
- **SQLite pure-Go** (`modernc.org/sqlite`) no store de sessão em vez do driver
  cgo — o binário compila com `CGO_ENABLED=0`.
- Dependências: `go.mau.fi/whatsmeow` atualizado (2026-01 → 2026-06) e
  `github.com/pion/webrtc/v4` adicionado (apenas para o subsistema de VoIP).

### Canais — WhatsApp oficial (ligações via Business Calling API)

Ligações de voz no canal **oficial** da Meta
([WhatsApp Business Calling API](https://developers.facebook.com/documentation/business-messaging/whatsapp/calling)),
em novo pacote `internal/whatsapp/officialcalls` acoplado ao adapter oficial
(`internal/adapters/whatsapp_official`). Diferente do não-oficial, usa **WebRTC
padrão** (ICE/DTLS/SRTP, OPUS) via `github.com/pion/webrtc/v4`.

- **Sinalização (Graph API):** `POST /{phone_number_id}/calls` com as ações do
  protocolo — `connect`/`pre_accept`/`accept`/`reject`/`terminate` — carregando
  `session {sdp_type, sdp}`; habilitação do recurso via `POST /settings`
  (`EnableCalling`, best-effort no `Connect`).
- **Webhook `calls`:** `ParseWebhookCalls` + `Gateway` roteiam `event=connect`
  (SDP offer) e `event=terminate`; fases `ringing` → `connected` → `ended`.
  Com `auto_answer_calls` o gateway negocia o answer e aceita automaticamente
  (bot/IVR); senão a chamada fica pendente até `AcceptCall`/`RejectCall`.
- **Mídia e gravação:** `CallSession` (pion `PeerConnection` + track OPUS)
  negocia o SDP; com `call_recordings_dir` grava o RTP recebido **direto em
  Ogg/Opus sem decodificar** (pure-Go), sem interferir na chamada. O evento
  `ended` carrega `recording_path`.
- Config do canal oficial: `enable_calls`, `auto_answer_calls`,
  `call_recordings_dir`.
- Cobertura de testes na sinalização, parse de webhook e mídia (loopback pion);
  o caminho ponta-a-ponta contra a Meta exige número com calling habilitado e só
  se confirma em teste ao vivo.

### Segurança / Correções (hardening dos conectores Teams/Slack/Mattermost)

- **Teams — exfiltração de token bloqueada:** o `serviceUrl` recebido na Activity
  só é persistido/usado como destino de saída se apontar para um host confiável do
  Bot Framework/Teams (`*.botframework.com`, `smba.trafficmanager.net`, `*.skype.com`)
  via `teams.IsTrustedServiceURL`. Antes, um `serviceUrl` envenenado vazaria o bearer
  AAD do bot para um host arbitrário.
- **Teams — endpoint compartilhado:** documentado o limite de confiança do modelo
  multi-tenant (o JWT do Bot Connector prova só `app_id`, não o tenant AAD); para
  isolamento estrito use o endpoint por canal com app registration distinto.
- **Webhook de saída — assinatura forjável:** entregas com `webhook_secret` vazio
  não são mais enviadas (fail-closed) em vez de emitir HMAC com chave vazia.
- **Webhook de saída — dedup:** o id de entrega passa a ser determinístico
  (`message_id`+tipo), evitando POSTs duplicados ao DeskLenz em redelivery.
- **Webhook de saída — canal removido:** entrega para canal deletado é descartada
  (ack) em vez de inundar a DLQ; só erros transitórios redelivram.
- **Download de mídia inbound (Slack/Teams/Mattermost):** limitado a 64 MiB para
  evitar OOM com anexos hostis/gigantes.
- **Mattermost:** corrigido vazamento de goroutine por reconexão (watcher de
  cancelamento agora encerra com a conexão) e posts só-anexo não viram mais
  mensagem vazia quando não há media store (metadados do anexo são preservados).
- **Facebook/Instagram:** `messaging_type` agora sobreponível por metadata
  (`messaging_type`/`message_tag`), default `RESPONSE` — permite envios fora da
  janela de 24h (UPDATE/MESSAGE_TAG) que antes ficavam travados.
- **Webhook producer:** corrigido off-by-one no backoff de retry (clamp no índice;
  sem panic de índice fora do range para `MaxRetries` > nº de delays).

### Adicionado

#### Entrega outbound unificada
- Subsistema `internal/outbound`: modelo de conteúdo tipado (Text/Template/Media),
  `Sender`/`Factory` por canal, `Resolver` (channelID → Sender, com cache TTL e
  invalidação) e um **worker NATS genérico** (resolve → envia → ack/nak).
- **Senders stateless** (HTTP) para WhatsApp Cloud API, Telegram, SMS (Twilio),
  Facebook Messenger, Instagram DM, RCS e Email.
- **Caminho stateful** (`PluginSender`) para WebChat e WhatsApp não-oficial
  (whatsmeow), entregando pela sessão/conexão viva no mesmo worker.
- **Dead-letter queue** (`LINKTOR_DLQ`): mensagens que esgotam as tentativas em
  qualquer consumer são dead-lettered (não mais loop ou descarte silencioso) e
  ficam retidas 14 dias para inspeção/replay; visível no monitor de fila.
- Rate-limit por canal e classificação de erro permanent/transient (4xx falha
  rápido; 429/5xx → retry/DLQ).
- Assunto de email configurável via metadata (`subject`/`email_subject`).

#### Funcionalidades de produto
- **Campanhas em massa**: entidade + repositório + serviço (modelo de fila:
  enfileira → worker entrega/retry), correlação de status por destinatário,
  sweep de `queued` órfão, e UI (lista, criação, progresso ao vivo, filtro,
  paginação, retry/cancel).
- **RBAC granular**: papéis customizados por tenant (`resource × action`),
  papéis de sistema seedados, cache no Redis, middleware `RequirePermission`,
  e UI com matriz recurso×ação.
- **Atribuição automática** de conversas (manual/round-robin/load-balanced) e
  **SLA + auto-close** via ticker, com UI de Operações.
- **Respostas rápidas** (canned) com atalho `/comando`, placeholders e contador.
- **Trilha de auditoria** via middleware (deriva ação/recurso da rota) +
  registros explícitos nos handlers especializados, com UI read-only filtrável.

#### Conectores de canal Teams / Slack / Mattermost
- **Webhook de saída `linktor-channel-v1`** (`internal/infrastructure/webhook`):
  envelope assinado (HMAC-SHA256, headers `X-Linktor-Signature`/`X-Linktor-Timestamp`,
  tolerância anti-replay 300s) entregue por canal ao consumidor externo (DeskLenz).
  `Dispatcher` consome eventos NATS (`message.received` + `message.{sent,delivered,
  read,failed}`), resolve o endpoint do canal (`channel.WebhookURL` +
  `credentials.webhook_secret`) e entrega via `WebhookProducer` (retry/backoff).
- **Microsoft Teams** (`internal/adapters/teams`): Bot Framework / Bot Connector.
  Inbound via webhook (`POST /webhooks/teams/:channelId`) com validação de JWT do
  Bot Connector (JWKS + issuer/audience); outbound `POST {serviceUrl}/v3/
  conversations/{id}/activities` com bearer AAD (token de app cacheado/renovado).
  Persiste a *conversation reference* (`service_url`) no 1º inbound para
  mensageria proativa. Suporta os dois modelos de propriedade do bot
  (single-tenant e multi-tenant) — distinção por `tenant_id` da Activity.
- **Slack** (`internal/adapters/slack`): Events API (inbound) + Web API (outbound).
  Handshake `url_verification`, verificação `X-Slack-Signature`
  (`v0:timestamp:body`, anti-replay), filtro de eco do próprio bot;
  outbound via `chat.postMessage` (bot token `xoxb-`).
- **Mattermost** (`internal/adapters/mattermost`): self-hosted. Inbound via
  WebSocket persistente (`{base_url}/api/v4/websocket`, evento `posted`, anti-eco
  por `bot_user_id`, reconexão com backoff). `Manager` inicia os listeners no boot
  e reage a connect/disconnect em runtime (via `ChannelLifecycleHooks`), iniciando/
  parando o listener por canal sem reinício; outbound REST `POST /api/v4/posts` com PAT.
- Enum `ChannelType` estendido (`teams=11`, `slack=12`, `mattermost=13`) no proto,
  entidade e `pkg/plugin`; senders registrados no `outboundResolver`.
- **Admin (web/admin)**: tipo `ChannelType` + ícones/labels/descrições (en/pt-BR/es),
  cards na lista de canais e telas de configuração (`teams-config.tsx`,
  `slack-config.tsx`, `mattermost-config.tsx`) com os campos de credencial exatos do
  backend (tudo em `credentials`) + seção opcional de webhook de saída (DeskLenz:
  `webhook_url` top-level + `webhook_secret`); E2E (`e2e/channels.spec.ts`) cobrindo
  criação + envio por canal. Paridade visual nos demais mapas por canal (dashboard,
  tabela de analytics, variantes de badge, opções de identidade de contato).
- **API de canal**: `webhook_url` aceito no create/update (`CreateChannelRequest` +
  `CreateChannelInput`/`UpdateChannelInput`), persistido na coluna `webhook_url`.
- **Mídia inbound re-hospedada**: anexos atrás de URLs autenticadas — Slack
  (`url_private`, baixado com bearer do bot token) e Mattermost (`/api/v4/files/{id}`
  via PAT) — são baixados com autenticação e re-hospedados num *media store*
  (`storage.Client.Upload`) para o consumidor externo (DeskLenz) conseguir buscá-los.
  Opt-in por `MEDIA_UPLOAD_DIR`/`MEDIA_UPLOAD_BASE_URL`; sem store configurado, mantém
  a URL do provedor (best-effort, sem regressão). Ligado no `WebhookHandler` (Slack) e
  no `mattermost.Manager`/listener. Store de mídia agora também suporta **MinIO/S3**
  (preferido quando `MINIO_ENDPOINT` setado; senão dir local; senão desligado).

#### Conectores — endurecimento (contrato + robustez)
- **Auth `X-API-Key` (DeskLenz → Linktor)**: o endpoint de envio
  (`POST /conversations/:id/messages`) e demais rotas protegidas passam a aceitar
  uma chave de API de tenant via header `X-API-Key`, além do JWT Bearer/cookie.
  `APIKeyService.Authenticate` resolve a chave por prefixo público + comparação
  bcrypt em tempo constante (chaves expiradas excluídas no SQL), registra
  `last_used_at` (best effort) e injeta o contexto de tenant. Papel `api` —
  passa nas rotas comuns, mas barrado de rotas admin-only.
- **Webhook de saída durável**: a entrega ao DeskLenz deixou de ser
  fire-and-forget em goroutine e passou pelo **stream durável NATS
  `LINKTOR_WEBHOOKS`**. O `Dispatcher` serializa o envelope e **enfileira** (carregando
  os bytes exatos + `channel_id`); um `DeliveryWorker` consome, **resolve o canal,
  assina com timestamp fresco** (mantém o retry dentro da janela anti-replay de 300s,
  sem o segredo trafegar no stream) e faz 1 tentativa HTTP — NATS cuida de
  retry/backoff e DLQ. Crash entre evento e entrega não perde mais o webhook.
- **Teams multi-tenant (1 app Azure → N clientes)**: novo endpoint compartilhado
  `POST /webhooks/teams` (sem `channelId`). Resolve o canal pela audiência do token
  (app id, lida sem verificar) + `tenant_id` AAD da Activity (`MatchSharedChannel`:
  match exato de org vence; fallback p/ canal multi-tenant), e **só então valida o
  JWT (JWKS)** contra o `app_id` do canal resolvido. Novo
  `ChannelRepository.FindAllByType` (cross-tenant, credenciais descriptografadas).
- **Mídia inbound do Teams re-hospedada**: anexos do Bot Connector (atrás de URL com
  bearer) são baixados com o token AAD e re-hospedados no media store
  (`teams.Client.DownloadAttachment`), com fallback para a URL do provedor.
- **Mattermost — validação do handshake WebSocket**: a resposta ao
  `authentication_challenge` passa a ser validada (status `seq_reply`); PAT inválido
  retorna erro explícito (backoff/log) em vez de aguardar posts em silêncio, e posts
  só são processados após o handshake autenticado.
- **Mattermost — auto-detecção de `bot_user_id`**: como o anti-eco depende
  inteiramente do `bot_user_id`, se ele não for configurado o listener resolve o id
  do próprio bot via `/api/v4/users/me` no connect (best effort), evitando que o bot
  reprocesse os próprios posts (loop de eco).
- **Edição de canal — prefill de campos não-secretos**: a resposta de
  `GET/List /channels` passa a expor os campos **não-secretos** das credenciais
  (whitelist por tipo — ex.: Teams `app_id`/`tenant_id`/`service_url`, Slack `app_id`/
  `bot_user_id`, Mattermost `base_url`/`bot_user_id`) dentro de `config`, para o
  formulário de edição preencher. Segredos (`*_token`, `*_password`, `signing_secret`,
  `webhook_secret`) continuam `json:"-"` e nunca são retornados (cópia defensiva, sem
  mutar a entidade).
- **Edição de canal — merge de credenciais**: `ChannelService.Update` passou de
  *replace* para *merge* nas credenciais — um campo de segredo em branco na edição
  (não redigitado) **preserva** o valor armazenado em vez de apagá-lo. Valores não
  vazios sobrescrevem; chaves novas são adicionadas.
- **Upload nativo de mídia outbound**: Slack (fluxo external upload —
  `files.getUploadURLExternal` → POST bytes → `files.completeUploadExternal`) e
  Mattermost (`POST /api/v4/files` multipart → post com `file_ids`) passam a enviar
  mídia como **arquivo nativo** (caption como comentário/corpo) em vez de link. A
  mídia (hospedada no Linktor) é baixada via `outbound.FetchMedia` (cap de 64 MiB);
  qualquer falha faz **fallback para o link** (entrega nunca é perdida).
- **Test connection (ao vivo) dos 3 canais**: endpoints `POST /channels/test-{teams,
  slack,mattermost}` validam as credenciais contra o provedor de verdade — Teams
  (token AAD client-credentials), Slack (`auth.test`), Mattermost (`/api/v4/users/me`)
  — com timeout de 10s, além da validação de forma dos campos obrigatórios. Botão
  "Testar conexão" nas 3 telas de configuração do admin.

#### Segurança
- **Criptografia de credenciais em repouso** (AES-256-GCM, prefixo `enc:v1:`,
  retrocompatível com texto puro), com validação obrigatória em release.
- **Rotação de chave sem downtime**: chave primária + `crypto.previous_keys`
  (decrypt com fallback) e endpoint `POST /api/v1/channels/reencrypt`.

### Alterado
- `ChannelService.Delete` carrega o canal para o hook `OnDeleted` (invalidação
  de cache do sender), preservando a idempotência do delete.
- Email: config building extraído em `ConfigFromMap`, compartilhado pelo adapter
  e pelo novo sender.

### Testes
- Mocks dos novos repositórios em `pkg/testutil` e testes unitários para
  criptografia (round-trip + rotação), RBAC, tradução/resolver de outbound,
  classificação de erro, plugin sender, e os serviços de campanha, atribuição,
  SLA, papéis e canned.
- Conectores: golden do envelope `linktor-channel-v1` (inbound/status + assinatura
  HMAC), e por canal — Teams (build de Activity, classificação de erro, conversation
  reference), Slack (assinatura `v0`, anti-eco, render), Mattermost (URL WS, parse de
  post, frame de auth).

### Documentação
- README: seções de funcionalidades 9–15, diagramas/mockups SVG e config de
  `crypto`. Novo `CHANGELOG.md`.
