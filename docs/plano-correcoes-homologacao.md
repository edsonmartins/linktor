# Plano de Correções para Homologação — LINKTOR

Base: `docs/auditoria-homologacao-2026-07.md` (auditoria de 2026-07-02).
Objetivo: levar o projeto de "NÃO apto" a "apto para homologação", com backlog completo
(dos bloqueadores aos itens menores) organizado em workstreams rastreáveis.

## Como ler este plano
- **ID** — identificador estável para commits/PRs (ex.: `WS1-CANAIS`).
- **Prioridade** — P0 (gate de homologação, bloqueador), P1 (obrigatório p/ produto funcionar),
  P2 (integridade de dados), P3 (polish/dívida técnica).
- **Aceite** — como validamos que ficou pronto (teste/checagem objetiva).
- Cada workstream deve virar um PR (ou poucos PRs coesos) com testes.

## Fases sugeridas (ordem de execução)
1. **Fase A — Fundação** (WS0): migrações/schema + harness de teste cross-tenant. Desbloqueia todo o resto.
2. **Fase B — Segurança crítica** (WS1 tenant, WS3 fail-open/RCE, WS2 panics).
3. **Fase C — Entrega** (WS4 pipeline, depois canais do escopo real WS5–WS8).
4. **Fase D — Integridade** (WS10) e **API/observabilidade** (WS9).
5. **Fase E — Periferia** (WS11 frontend/deploy, WS12 CLI, WS13 testes) e **P3**.

Regra de escopo: canais fora do escopo de homologação que estão quebrados devem ser
**desabilitados explicitamente** (retornar erro claro em `Connect`/registro) em vez de expostos — ver WS14.

---

## WS0 — Fundação: Schema e Migrações  `[P0]`  ✅ (validação viva pendente)

Descoberta: já existe runner goose (`RunMigrations`→`runGooseMigrations`); `baselineStatements()` é a
migração v1 e correções vão como `.sql` numerados (v2+). Feito nesta sessão:

- [x] **WS0-MIG-SETUP** — goose já estava ligado no boot (`main.go:153`). Confirmado; correções agora
  entram como migrações versionadas `.sql`.
- [x] **WS0-SCHEMA-CTX** — migração `00002_align_repo_schema.sql`: `conversation_contexts` ganhou
  `bot_id, intent_name, intent_confidence, entities, context_window, state, last_analysis_at,
  current_flow_id, metadata` (expandiu a tabela para o modelo do repo).
- [x] **WS0-SCHEMA-AIRESP** — `00002`: `ai_responses` ganhou `message_id` e `conversation_id` virou
  nullable (a entidade `AIResponse` não tem ConversationID; o INSERT do repo agora satisfaz o schema).
- [x] **WS0-SCHEMA-ANALYTICS** — `00002`: `conversations.metadata` + `conversation_contexts.current_flow_id/metadata`.
- [x] **WS0-SCHEMA-TPLSUB** — `00002`: `templates.sub_category`. (Persistir `entity.Template.SubCategory`
  no INSERT/UPDATE ainda pendente — a coluna existe, o repo lê no filtro; escrita fica em WS10.)
- [x] **WS0-SCHEMA-PGVECTOR** — migração `00003_pgvector_embedding.sql`: `CREATE EXTENSION vector` +
  `embedding` de TEXT→`vector` (imagem já é `pgvector/pgvector:pg16`). Coluna unbounded (dimensão
  configurável 1536/768); índice ANN fica para quando a dimensão for fixada por deploy.
- [ ] **WS0-SCHEMA-VALIDATE** — PENDENTE: sem Docker/pgvector local nesta máquina. Rodar em CI com a
  imagem pgvector: subir DB limpo, `goose up`, smoke de bot_repo/analytics_repo/knowledge_repo/template_repo.
  *Aceite:* job de CI verde "schema smoke".

---

## WS1 — Isolamento de Tenant (IDOR)  `[P0]`

> **STATUS (sessão 2026-07-02):** NÚCLEO CONCLUÍDO. Feitos com teste cross-tenant e build/vet/suíte
> VERDES: WS1-CANAIS, WS1-BOTS, WS1-KB, WS1-USERS, WS1-TEMPLATES, WS1-COMMERCE-FLOWS, WS1-WA-ADVANCED
> (incl. guard no refund + wiring de `channelRepo` no main.go), WS1-HISTIMPORT. Padrão usado: wrappers
> `GetByTenantAndID`/`...ForTenant` na camada de serviço (espelha `ContactService`), sem tocar interfaces/mocks.
> PENDENTES: WS1-CONFIG-LEAK (Config serializa segredos na resposta), WS1-PAYMENTS-REPO / WS1-CARTS
> (scoping `organization_id` como defesa-em-profundidade), WS1-DEFENSE (guards condicionais, contact FindByEmail/Phone),
> WS1-EMAILUNIQ (FindByEmail global).

Padrão de correção: `WHERE id=$1 AND tenant_id=$2` no repo **ou** checar `resource.TenantID != tenantID -> 404`
no service. Adicionar teste de isolamento cross-tenant para cada recurso.

- [ ] **WS1-CANAIS** — `channel_repo.go:80,262,323,339,361`; `service/channel.go:130,135,197,212`;
  `handlers/channel.go` (Get/Update/Delete/Connect/Disconnect/UpdateStatus/UpdateEnabled/RequestPairCode).
  Também garantir que `Config` não vaze segredos na resposta (aplicar `json:"-"` ou DTO de saída sem credenciais).
- [ ] **WS1-BOTS** — `bot_repo.go:62,168,213,229,258,274`; `service/bot.go:94,105,143,148,533`;
  `handlers/bot.go` (GetByID/Update/Delete/Activate/Deactivate/AssignChannel/UnassignChannel/UpdateConfig/AddEscalationRule/Test).
  `AssignChannel` deve validar que canal e bot pertencem ao mesmo tenant.
- [ ] **WS1-KB** — `knowledge_repo.go:63,122,167,318,377,426,447`; `service/knowledge.go`; handlers.
  `Search(kbID,...)` deve validar ownership da KB.
- [ ] **WS1-PAYMENTS** — `payment_repo.go:69,91,114,165`: adicionar `organizationID` à interface `PaymentStore` e a todas as queries.
- [ ] **WS1-USERS** — `user_repo.go:154,194` (Update/Delete). `service/user.go:104,133` deve exigir `user.TenantID == tenantID`.
- [ ] **WS1-CARTS** — `cart_repo.go:65,93,114,253,267,210,179`.
- [ ] **WS1-COMMERCE-FLOWS** — `commerce.go:33` e `whatsapp_flows.go:29,96,110,143,161`: validar `channel.TenantID`
  antes de usar `access_token`; validar `flow.TenantID` nas ops por-ID.
- [ ] **WS1-HISTIMPORT** — `history_import.go:74` valida `channel.TenantID == input.TenantID`; repo escopado.
- [ ] **WS1-WA-ADVANCED** — `payments.go`, `calling.go`, `ctwa.go`, `whatsapp_analytics.go`: antes de
  `getClient(channelID)`, carregar canal e exigir `channel.TenantID == MustGetTenantID(c)` (404 caso contrário).
  Prioridade máxima em `refund` (ação de estado com efeito financeiro).
- [ ] **WS1-TEMPLATES** — `template.go:117,286` (GetByID/Delete) passar e validar `tenantID`.
- [ ] **WS1-DEFENSE** — Remover guards condicionais que "desligam" o filtro quando tenant vazio
  (`role_repo`, `template_repo` com `if input.TenantID != ""`). `contact_repo.FindByEmail/Phone` escopar por tenant.
- [ ] **WS1-EMAILUNIQ** — `user_repo.go:75` `FindByEmail` global vs `UNIQUE(tenant_id,email)`:
  escopar por tenant onde o fluxo tem tenant; para login, resolver por (tenant, email) ou documentar estratégia.
- [ ] **WS1-TESTS** — Suíte reutilizável de teste de isolamento: para cada recurso, tenant A não acessa recurso de B (404).

*Aceite geral WS1:* testes cross-tenant verdes para todos os recursos acima; nenhuma query endereçada por
entidade sem `tenant_id`/`organization_id`.

---

## WS2 — Estabilidade: Panics, Races e Leaks  `[P0/P1]`

> **STATUS (sessão 2026-07-02):** feitos com testes (build/vet/suíte + `-race` verdes):
> WS2-MEDIA-PANIC (whatsmeow/FB/IG), WS2-QR-CHAN, WS2-DISCONNECT (whatsmeow; email fica p/ WS14),
> WS2-WA-CONNECT-LEAK, WS2-WA-EVENT-DROP (buffer 100→1024 + log), WS2-HISTIMPORT-RACE (mutex),
> WS2-CTX-RACE (copy-on-read + TTL/tamanho), WS2-WEBCHAT-HUB (select/recover + register/unregister
> não-bloqueantes), WS2-FLOW-RECURSION (maxDepth=100). PENDENTES (fora do escopo de canais → WS14):
> WS2-AMI-RACE, WS2-ESL-RACE, WS2-MM-KEEPALIVE, WS2-MM-ECHO, WS2-AMI-RECONNECT. WS2-FLOW-IV-PANIC
> segue com WS5I (Flows). WS2-STOP-NOOP pendente (P2).


- [ ] **WS2-MEDIA-PANIC** `[P0]` — nil-pointer no envio de mídia por `err` sombreado:
  `whatsapp/adapter.go:245-307`, `facebook/adapter.go:157-197`, `instagram/adapter.go:166-201`.
  Declarar `var mediaData []byte; mediaData, err = ...` e retornar `SendResult{Success:false}` quando sem attachment.
- [ ] **WS2-QR-CHAN** `[P0]` — `whatsapp/client.go:140-162`: recriar `c.qrCh` a cada `Login()` (evita close of closed channel).
- [ ] **WS2-DISCONNECT** `[P0]` — Disconnect duplo: `whatsapp/adapter.go:137-149` e `email/adapter.go:179-181`:
  `stopCh = nil` após close; zerar client mesmo em erro de Close.
- [ ] **WS2-HISTIMPORT-RACE** `[P0]` — `history_import.go:26,111,116,272` (mapa `runningImports`) e `:24,174` (`waClient`):
  proteger com mutex/`sync.Map`. Race atual derruba o servidor (`concurrent map writes`).
- [ ] **WS2-CTX-RACE** `[P0]` — `conversation_context.go:31,51` + `flow_engine.go:89,146`: mutex protege o mapa mas não
  o `State map` interno; duas mensagens na mesma conversa → panic. Copiar/lock por conversa; adicionar eviction (TTL/LRU).
- [ ] **WS2-WEBCHAT-HUB** `[P0]` — `webchat/websocket.go:139-166,188-201,268-275`: proteger sends com `select`/`recover`
  sobre `h.done`; no `ReadPump`/handler usar `select { case h.register<-c: case <-h.done: }`. Evita panic + leak no shutdown.
- [ ] **WS2-AMI-RACE** `[P0]` — `asterisk.go:155,185-197,222-223`: um único `bufio.Reader` com demux por `ActionID`
  (canais separados resposta/evento).
- [ ] **WS2-ESL-RACE** `[P0]` — `freeswitch.go:103,208-222,362,422,448,469,859`: idem, demux por `Content-Type: command/reply`.
- [ ] **WS2-FLOW-IV-PANIC** `[P1]` — `flows/encryption.go:143-148`: `cipher.NewGCMWithNonceSize(block, len(iv))` + validar tamanho.
- [ ] **WS2-FLOW-RECURSION** `[P1]` — `flow_engine.go:180`: limite de profundidade/detecção de ciclo (evita stack overflow).
- [ ] **WS2-WA-CONNECT-LEAK** `[P1]` — `whatsapp/adapter.go:106-118`: `Connect` repetido vaza client/goroutine/SQLite; fechar anterior.
- [ ] **WS2-WA-EVENT-DROP** `[P1]` — `whatsapp/events.go:48-64`: buffer (100) cheio descarta receipts/mensagens sem log; aumentar/backpressure/log.
- [ ] **WS2-MM-KEEPALIVE** `[P1]` — `mattermost/listener.go:122-154`: `SetReadDeadline` + ping/pong para detectar conexão morta e reconectar.
- [ ] **WS2-MM-ECHO** `[P1]` — `mattermost/listener.go:97-104,163`: se `bot_user_id` não resolve, não processar posts (evita loop de eco).
- [ ] **WS2-AMI-RECONNECT** `[P1]` — `asterisk.go:185-197`: reconexão com backoff; zerar `amiConn` em erro.
- [ ] **WS2-STOP-NOOP** `[P2]` — `consumer.go:259`, `ai_consumer.go:221`: `cancelFunc` nunca atribuído; goroutines de polling vazam. Ligar cancel.

---

## WS3 — Segurança  `[P0]`

> **STATUS (sessão 2026-07-02):** feitos com testes (build/vet/suíte verdes):
> WS3-JWT-TYPE (claim `typ` access/refresh + assert HMAC), WS3-SEED (wipe destrutivo agora atrás de
> `SEED_RESET_DESTRUCTIVE=true`; senha aleatória/`SEED_ADMIN_PASSWORD`), WS3-SSRF (media_processor +
> whatsmeow fetchMediaFromURL: bloqueio IP privado/link-local + LimitReader + ctx), WS3-WA-SIG-FAILOPEN
> (whatsapp_official + FB/IG fail-closed sem secret), WS3-CONSTTIME (verify_token subtle.ConstantTimeCompare
> em wa_official/FB/IG), WS3-HMAC-TS (envelope assina `ts.body` + VerifySignature; SDK `sdks/go/webhook.go`
> alinhado), WS3-AI-QUOTA (cap max_tokens 4096 + allowlist provider/model), WS3-BODY-LIMIT (FB/IG 1MB).
> PENDENTES (fora do escopo de canais → WS14): WS3-FREESWITCH-RCE, WS3-WEBHOOK-FAILOPEN (voz/SES/SMS),
> WS3-SNS-VERIFY, WS3-CRLF-EMAIL, WS3-TWIML-XML, WS3-IMAP-INJECT. Ainda P2: WS3-OBS-SCOPE, WS3-ERRLEAK,
> WS3-DBFILE-PERMS, WS3-WS-JWT-METHOD. Nota: mudança de HMAC altera a assinatura no fio — consumidores
> externos (DeskLenz) precisam validar `timestamp + "." + body`.


- [ ] **WS3-FREESWITCH-RCE** `[P0]` — `freeswitch.go:518`: eliminar `api system rm`; validar `recordingID` como UUID
  (whitelist) e apagar via API de arquivos/S3. Sanitizar `\n`/`\r` em todos os `uuid_*`/ações AMI.
- [ ] **WS3-WEBHOOK-FAILOPEN` [P0]` — Fechar fail-open de validação:
  - Voz: `vonage.go:647` (validar JWT), `asterisk.go:654`/`amazon_connect.go:520` (HMAC de segredo compartilhado como FreeSWITCH),
    `twilio.go:580-584` (fail-closed; usar URL real do request).
  - Email: `email/webhook.go:499` SendGrid (ECDSA `X-Twilio-Email-Event-Webhook-Signature`).
  - SMS: `sms/adapter.go:277-281` fail-closed e assinar com URL do request.
  - `requireWebhookSecrets`: exigir segredo por padrão; falhar boot se canal externo sem segredo (ou garantir `Mode=release` em prod).
- [ ] **WS3-SNS-VERIFY** `[P0]` — `email/webhook.go:277-338`: validar assinatura SNS (cert AWS) e confirmar subscription (GET no `SubscribeURL`).
- [ ] **WS3-CRLF-EMAIL** `[P0]` — `smtp.go:155-181`, `ses.go:207-232`: rejeitar/sanitizar `\r`/`\n` em Subject/To/ReplyTo/headers custom; `mime.QEncoding` no subject.
- [ ] **WS3-TWIML-XML** `[P0]` — `voice/twilio.go` (IVR): `xml.EscapeText` em todo texto/atributo (evita toll-fraud por injeção de `<Dial>`).
- [ ] **WS3-JWT-TYPE** `[P0]` — `auth.go:98,135,168,189`: claim `typ` (access/refresh) validado no lugar certo; store de jti p/ rotação/revogação; invalidar em ChangePassword/desativação.
- [ ] **WS3-SEED** `[P0]` — `seed.go:28` remover o `DELETE FROM ...` destrutivo (ou restringir a flag explícita + confirmação);
  `seed.go:50` gerar senha admin aleatória e exibir uma vez (nunca `admin123` fixo).
- [ ] **WS3-SSRF** `[P0]` — allowlist de esquema/host + bloqueio de IP privado/link-local + `io.LimitReader`:
  `media_processor.go:43`, `whatsapp/adapter.go:700` (`fetchMediaFromURL`), downloads de mídia dos adapters.
- [ ] **WS3-WA-SIG-FAILOPEN** `[P1]` — `whatsapp_official/adapter.go:553`, FB/IG/RCS: fail-closed quando secret vazio (ou log de alerta forte).
- [ ] **WS3-CONSTTIME** `[P1]` — comparações não constant-time: `verify_token` (whatsapp_official/webhook.go:46, fb/ig webhook.go:31),
  Mailgun duplicado (`mailgun.go:253`), Postmark (`webhook.go:519`). Usar `hmac.Equal`/`subtle.ConstantTimeCompare` e remover duplicatas.
- [ ] **WS3-HMAC-TS** `[P1]` — assinar `timestamp + "." + body` e validar frescor:
  webhook externo (`webhook/envelope.go:90`) e replay do Mailgun (sem checagem de timestamp hoje).
- [ ] **WS3-AI-QUOTA** `[P1]` — `handlers/ai.go:91` `/ai/complete`: teto de `max_tokens`, allowlist de modelo, rate-limit/quota por tenant.
- [ ] **WS3-BODY-LIMIT** `[P2]` — `http.MaxBytesReader` nos `io.ReadAll(r.Body)` de webhooks (fb/ig/others) para evitar DoS por payload.
- [ ] **WS3-WS-JWT-METHOD** `[P2]` — `websocket.go:272`: assertar `*jwt.SigningMethodHMAC` e distinguir access/refresh.
- [ ] **WS3-OBS-SCOPE** `[P2]` — `observability.go` queue/stream/reset-consumer: restringir a superadmin de plataforma (não admin de tenant).
- [ ] **WS3-ERRLEAK** `[P2]` — payments/calling/ctwa/analytics retornam `err.Error()` cru da Meta ao cliente; padronizar via `RespondError`.
- [ ] **WS3-DBFILE-PERMS** `[P2]` — `whatsapp/types.go:67-69`: sanitizar `ChannelID` no path/DSN do SQLite; permissão 0600; remover `storages/whatsapp_test-channel.db` do git + `.gitignore`.
- [ ] **WS3-IMAP-INJECT** `[P2]` — `imap.go:217`: `LOGIN` com quoting/literais IMAP (senha com espaço/aspas quebra/injeta).

---

## WS4 — Pipeline Central de Entrega  `[P1]`  ✅

> **STATUS (sessão 2026-07-02):** CONCLUÍDO com testes (build/vet/suíte verdes).
> - WS4-INTERACTIVE: novo `Kind` `Interactive` (botões channel-agnostic) no modelo outbound; `translate()`
>   reconstrói de `metadata["quick_replies"]`+`interactive_body`; `send_message.go` emite esses campos;
>   senders WhatsApp oficial (reply buttons ≤3 / list >3) e Telegram (inline keyboard) renderizam nativamente;
>   fallback para texto quando não há botões. Testes em `outbound_test.go`.
> - WS4-ATTACHMENTS: `translate()` agora faz source de `raw.Attachments[0]` para Media (inclui inferência de
>   tipo por MIME quando o ContentType é texto). Testes cobrindo anexo→media.
> - WS4-RATELIMIT-CTX: `channelLimiter.wait(ctx, ...)` interrompível por ctx (timer + select) — não estoura AckWait.
> - WS4-DEDUP-ACK: consumer inbound classifica erros (`isRetryableInboundError`) — conflict/validation/forbidden/
>   not-found/unauthorized viram ACK (log), transitórios NAK. Evita poluição da DLQ por duplicata.
> - WS4-BOT-PIPELINE: `maybeTriggerBot` no consumer inbound publica `BotResponseRequest` quando a mensagem é de
>   contato, a conversa não tem agente humano e o canal tem bot ativo (`botRepo.FindByChannel`). Fecha a lacuna
>   em que os subscribers de bot estavam ligados mas nunca recebiam tráfego (auto-resposta não funcionava).
>   Gating anti-loop: só `SenderTypeContact`.
> PENDENTE (P2): WS4-PUBLISH-ERR (falha de PublishEvent/PublishStatusUpdate ainda engolida — outbox/retry).

Sem isto o produto não entrega recursos-chave, independentemente do canal.

- [ ] **WS4-INTERACTIVE** — `worker.go:154` sem case `interactive`; adicionar tradução de `metadata["interactive"]`
  para o payload de cada sender (whatsapp oficial/whatsmeow/telegram/rcs/fb). `send_message.go:137` já popula o metadata.
- [ ] **WS4-ATTACHMENTS** — `worker.go:138` `translate()` só olha `meta["media_url"]/["media_id"]`; ler `Attachments`
  (populado em `send_message.go:185`) e propagar a cada sender.
- [ ] **WS4-BOT-PIPELINE** — `ai_consumer.go:230-275` (`PublishBotAnalysis/Response/Escalation`) nunca chamados;
  conectar o fluxo para que os consumers (`main.go:835`) recebam tráfego. Sem isso a auto-resposta de bot não funciona.
- [ ] **WS4-PUBLISH-ERR** — `receive_message.go:314,349,365` e `message.go:293,346,398`: falha de `PublishEvent`
  não pode ser engolida; propagar/retry ou outbox transacional. Idem `PublishStatusUpdate` (`worker.go:104,115,122`).
- [ ] **WS4-DEDUP-ACK** — `main.go:780`: conflito de dedup e erros de validação/forbidden devem dar ACK (nil), não NAK→DLQ.
- [ ] **WS4-RATELIMIT-CTX** — `worker.go:82,201`: rate limiter `wait()` deve respeitar `ctx` (não `time.Sleep` fixo)
  para não estourar AckWait (redelivery → envio duplicado) e não travar shutdown.
- [ ] **WS4-STATUS-STUCK** — garantir que envio bem-sucedido atualize status (hoje pode ficar `pending` para sempre — ver WS4-PUBLISH-ERR).

---

## WS5 — WhatsApp (Oficial + whatsmeow + internal)  `[P0/P1]`

> **STATUS (sessão 2026-07-02):** feitos com testes (build/vet/suíte + `-race` verdes):
> - Oficial: WS5O-WEBHOOK-FAILED (error_data/DisableInfo/OtherInfo como objeto — parse não descarta mais o batch),
>   WS5O-QUICKREPLY (campo `Payload`), WS5O-COUPON (`coupon_code`), WS5O-OTP (`sub_type:url`, sem package/signature no send),
>   WS5O-RATELIMIT (130429/131048/131056 → retryable), WS5O-CONFIG-RACE (RLock em `c.config`), WS5O-UTF8 + WS5O-LIST-LIMITS
>   (truncar por runes; 10 rows total). Testes agora asseram o JSON serializado.
> - whatsmeow: WS5W-MEDIA-INBOUND (download+descriptografia eager via whatsmeow; bytes em Metadata["data"] base64;
>   URL cifrada limpa) e WS5W-LOCATION (lat/long extraídos).
> - Flows: WS5I-FLOW-CRYPTO (IV invertido + base64 cru), WS5I-FLOW-BUILDER (duplicate screen ID), WS2-FLOW-IV-PANIC
>   (NewGCMWithNonceSize), WS5I-FLOW-ENDPOINT-ERR (status 421/427/432 sem vazar). Round-trip testado.
> PENDENTES: WS5I-FLOW-METAID (persistir Meta flow ID em service/whatsapp_flows.go — não feito), WS5I-PAY-*/
> WS5I-COMMERCE-*/WS5I-CATALOG-*/WS5I-ANALYTICS-*/WS5I-CALLING/WS5I-STATE-PERSIST (payments/commerce/analytics/calling/ctwa
> — P1, não tocados nesta sessão), WS5O-WEBHOOK-META/WS5O-SESSION-WINDOW (P2), WS5W-AUTOTRUST/WS5W-RECEIPTS (P2),
> WS5I-APIVER (P2).

### WhatsApp Cloud API (`whatsapp_official`)
- [ ] **WS5O-WEBHOOK-FAILED** `[P0]` — `types.go:111-116`: `error_data` deve ser `*ErrorData` (objeto), idem `DisableInfo`/`OtherInfo`.
  Hoje qualquer status `failed` quebra o parse e descarta o batch inteiro (inclui inbound). **Reproduzido.**
- [ ] **WS5O-QUICKREPLY** `[P0]` — quick_reply serializa `text` em vez de `payload` (`template.go:207-209`,
  `template_convert.go:299-302`, `carousel.go:144-221`): adicionar campo `Payload` em `TemplateParameter`.
- [ ] **WS5O-COUPON** `[P1]` — copy_code serializa `text` em vez de `coupon_code` (`template.go:250-253`, `template_convert.go:318-322`).
- [ ] **WS5O-OTP** `[P1]` — `auth_templates.go:110-117`: `sub_type` inválido + `package_name`/`signature_hash` no payload; corrigir `SendOTP`.
- [ ] **WS5O-RATELIMIT** `[P1]` — `client.go:464-466`: 130429/131048/131056 (HTTP 400) são transitórios; reclassificar (evitar DLQ em picos) + backoff.
- [ ] **WS5O-CONFIG-RACE** `[P1]` — `client.go:483-492`: lock em `c.config` durante rotação de token.
- [ ] **WS5O-LIST-LIMITS** `[P2]` — `interactive.go:199-202`: limite de 10 rows é total (não por seção) + limite de seções.
- [ ] **WS5O-UTF8** `[P2]` — `interactive.go:178-180`: truncar por caracteres, não bytes (corrompe acento/emoji pt-BR).
- [ ] **WS5O-WEBHOOK-META** `[P2]` — `webhook.go:398-402`: não descartar `conversation`/`pricing`/error code (billing/janela).
- [ ] **WS5O-SESSION-WINDOW** `[P2]` — `consumer.go:234-243`: `SessionAwareConsumer` nunca instanciado (janela 24h é código morto); instanciar + expiração de sessão.

### whatsmeow (`whatsapp`)
- [ ] **WS5W-MEDIA-INBOUND** `[P1]` — `adapter.go:596-604,409`: propagar `MediaKey`/`SHA256`; implementar `DownloadMedia` (hoje stub) — nenhuma mídia recebida chega à app.
- [ ] **WS5W-LOCATION** `[P1]` — `events.go:247-255`: extrair lat/long de localização inbound.
- [ ] **WS5W-AUTOTRUST** `[P2]` — `types.go:76-77`: tornar `AutoTrustIdentity` opt-in (evitar MITM Signal silencioso).
- [ ] **WS5W-RECEIPTS** `[P2]` — `adapter.go:634`: receipts em lote perdem N-1 IDs; `ReceiptTypePlayed` rebaixado; revogação/edição/poll/view-once viram texto vazio (`events.go`).

### internal/whatsapp (Flows, Payments, Commerce, Analytics, Calling, CTWA)
- [ ] **WS5I-FLOW-CRYPTO** `[P0]` — `flows/encryption.go:115-133`: IV da resposta = `requestIV[i]^0xFF`; retornar Base64 puro
  (não JSON) em `endpoint.go:101-103`. Sem isso 100% das interações de Flow falham no aparelho.
- [ ] **WS5I-FLOW-BUILDER** `[P1]` — `flows/builder.go:68-140`: `Build()` conta telas 2x → "duplicate screen ID"; corrigir merge/validação.
- [ ] **WS5I-FLOW-METAID** `[P1]` — `whatsapp_flows.go:80,208`: persistir `metaFlow.ID` em campo próprio (não Description); Update/Delete/Publish usam o Meta ID; sync não duplica.
- [ ] **WS5I-PAY-STATUS** `[P1]` — `payments/client.go:626-643,722-729,922-929`: checar `StatusCode`; refund reflete status real do gateway.
- [ ] **WS5I-PAY-ORDER** `[P1]` — `payments/client.go:283-297`: usar preço unitário (não `TotalPrice`) e `payment_settings` como array.
- [ ] **WS5I-COMMERCE-PRICE** `[P1]` — `commerce/order.go:197-203`: `parsePrice` trunca centavos e zera preço inválido sem erro.
- [ ] **WS5I-CATALOG-PAGE** `[P1]` — `commerce/catalog.go:604-621`: paginação pode entrar em loop infinito (mistura `Paging.Next` e `Cursors.After`).
- [ ] **WS5I-ANALYTICS-NODE** `[P1]` — `analytics/client.go:130-153`: usar nó WABA (não phone), ler `data_points`, mapear `conversation` (hoje sempre zero).
- [ ] **WS5I-CALLING** `[P1]` — `calling/client.go:110-181`: alinhar payloads à spec (action `connect` + sessão SDP; terminate correto).
- [ ] **WS5I-STATE-PERSIST** `[P1]` — `ctwa/client.go:11-17`, `calling`, `commerce/order.go`/`cart.go`: estado crítico só em memória sem eviction;
  usar `OrderRepository`/`CartRepository` existentes; TTL na janela 72h do CTWA.
- [ ] **WS5I-RETRY-429** `[P2]` — retry/backoff + tratamento de 429 nos 7 clientes; `json.Unmarshal` de erro da Graph não ignorado.
- [ ] **WS5I-FLOW-ENDPOINT-ERR** `[P2]` — `flows/endpoint.go:74-99`: usar códigos 421/427/432 e não vazar `err.Error()` em claro.
- [ ] **WS5I-APIVER** `[P2]` — versão `v21.0` hardcoded em 8 pontos (sai de suporte em 2026); centralizar/config.

---

## WS6 — Meta Family + RCS  `[P1/P2]`

> **STATUS (sessão 2026-07-02):** Meta family feito (build/vet/suíte verdes): WS6-FB-IG-PANIC (via WS2),
> WS6-OAUTH-ERR (fail-closed + campo Error), WS6-FB-POSTBACK (ExtractPostbacks ligado ao ProcessWebhook),
> WS6-IG-EVENTS (reactions/story reply/mention + reads), WS6-IG-CLASSIFY (match instagram_id), WS6-IG-BASEURL
> (graph.facebook.com no fluxo page-token), WS6-META-PAGE-PAGINATION (segue Paging.Next), WS6-OAUTH-URLENC (url.Values).
> PENDENTE: RCS inteiro (WS6-RCS-*) — fora do escopo de homologação, tratar via WS14 (desabilitar).


- [ ] **WS6-FB-IG-PANIC** `[P0]` — (coberto em WS2-MEDIA-PANIC) nil-pointer em envio de mídia FB/IG.
- [ ] **WS6-OAUTH-ERR** `[P1]` — `meta/client.go:362-428`: checar StatusCode e `error` do corpo (adicionar `Error` a `LongLivedTokenResponse`);
  handler (`oauth.go:289`) não persiste token vazio.
- [ ] **WS6-FB-POSTBACK** `[P1]` — `facebook/webhook.go:119`: `ExtractPostbacks` não é chamado; clique em botão perdido. Ligar no processamento.
- [ ] **WS6-IG-EVENTS** `[P1]` — reactions/story replies/seen do Instagram não consumidos (`instagram/client.go:122-127`); tratar `event.Reaction`/`event.Read`.
- [ ] **WS6-IG-CLASSIFY** `[P1]` — `instagram/webhook.go:129-146`: `IsInstagramViaPageWebhook` classifica qualquer webhook de Messenger como IG;
  comparar `entry.ID`/`recipient.ID` com o `instagram_id` do canal.
- [ ] **WS6-IG-BASEURL** `[P1]` — `instagram/client.go:24`: no fluxo IG-via-FB-Page usar `graph.facebook.com` (page token não funciona em graph.instagram.com).
- [ ] **WS6-MEDIA-STUB** `[P1]` — mensagens de mídia FB/IG sem attachment caem no `default` → retornar erro claro (parte do WS2-MEDIA-PANIC).
- [ ] **WS6-RCS-RICH** `[P1]` — Infobip/Pontaltech/Google descartam rich card/carousel/suggestions/mídia
  (`rcs/client.go:290-565`): implementar por provider ou remover capabilities + rejeitar com erro.
- [ ] **WS6-RCS-GOOGLE** `[P1]` — `rcs/client.go:512-608`: OAuth2 service account + `messageId` UUID; tratar delivery/read/Pub/Sub. Ou marcar não suportado.
- [ ] **WS6-RCS-INFOBIP-BATCH** `[P1]` — `rcs/client.go:370-391`: iterar todos `results[]` (hoje só `[0]`); `ParseWebhook` retorna slice.
- [ ] **WS6-RCS-MIME** `[P2]` — `rcs/sender.go:58-59`: enviar MIME real (`image/jpeg`) em vez de tipo lógico (`image`).
- [ ] **WS6-RCS-STATUSCORR** `[P2]` — `rcs/client.go:236-243`: correlacionar status pelo `messageId` da mensagem (não id do evento).
- [ ] **WS6-META-PAGE-PAGINATION** `[P2]` — `meta/client.go:261-276`: seguir `Paging.Next` em `GetMyPages` (>25 páginas).
- [ ] **WS6-OAUTH-URLENC** `[P2]` — `facebook/client.go:315`, `instagram/client.go:212`: `url.Values` (encode de redirect/state).
- [ ] **WS6-META-UNMARSHAL** `[P2]` — `rcs/client.go:181,335,458,558`: não ignorar erro de parse (evita `Success:true, MessageID:""`).
- [ ] **WS6-META-LOG** `[P3]` — trocar `fmt.Printf` por logger estruturado (fb/ig adapters).

---

## WS7 — Email + SMS + Voz  `[P1/P2]`

### Email
- [ ] **WS7-IMAP-PARSE** `[P1]` — `imap.go:278-283`: implementar parsing real do `SEARCH` (ou migrar p/ `emersion/go-imap`); hoje nunca entrega inbound.
- [ ] **WS7-IMAP-RECONNECT** `[P1]` — `imap.go:141-162`: reconectar em conexão morta (zerar `conn` em erro de rede); logar erros de polling.
- [ ] **WS7-SES-SIGV4** `[P1]` — `ses.go:299-306`: `canonicalURI="/"` (path vazio quebra assinatura) ou migrar p/ aws-sdk-go-v2; hoje 403 sempre.
- [ ] **WS7-SES-DECODER** `[P2]` — `ses.go:386-395`: `ParseSESNotification` é no-op; usar `encoding/json` real.
- [ ] **WS7-EMAIL-QP** `[P2]` — `smtp.go:192-199`, `ses.go:244-251`: corpo declarado quoted-printable mas escrito cru; usar `mime/quotedprintable`.
- [ ] **WS7-EMAIL-ATTACH-URL** `[P2]` — todos providers só usam `att.Content`; baixar `Attachment.URL` (com limite) antes de enviar. `email/sender.go:48` só faz SendText (HTML/anexos não trafegam).
- [ ] **WS7-EMAIL-MULTIPART** `[P2]` — `email/webhook.go:37-41`: SendGrid Inbound Parse/Mailgun multipart parseados como querystring; usar `multipart.NewReader`.
- [ ] **WS7-EMAIL-BATCH** `[P2]` — `email/webhook.go:104-109`: SendGrid envia N eventos; processar todos.
- [ ] **WS7-EMAIL-DISCONNECT** `[P2]` — (coberto em WS2-DISCONNECT) double close em `email/adapter.go`.
- [ ] **WS7-EMAIL-SETUP-ERR** `[P3]` — `email/adapter.go:160-168`: erros de setup IMAP engolidos; propagar ao status do canal.

### SMS
- [ ] **WS7-SMS-MMS** `[P1]` — `sms/adapter.go:382-386`: chamar `ExtractMediaURLs` e popular `inbound.Attachments` (MMS inbound perde mídia).
- [ ] **WS7-SMS-TIMEOUT** `[P2]` — `sms/client.go:63-98`: `http.Client` com timeout no twilio-go; propagar `context`.
- [ ] **WS7-SMS-PHONEVALIDATE** `[P3]` — `sms/client.go:139`: aceitar E.164 curtos válidos.

### Voz
- [ ] **WS7-CONNECT-GETCALL** `[P1]` — `amazon_connect.go:212,249`: corrigir path `DescribeContact` (com instanceID); mapear status real.
- [ ] **WS7-FS-ENDCONF** `[P1]` — `freeswitch.go:632-634`: flag `endconf` invertida.
- [ ] **WS7-FS-SHORTREAD** `[P1]` — `freeswitch.go:196`: `io.ReadFull` no body ESL (short read desalinha eventos).
- [ ] **WS7-FS-CALLLEAK** `[P1]` — `freeswitch.go:30,247`: remover entradas do mapa `calls` em `CHANNEL_HANGUP_COMPLETE` (memory leak).
- [ ] **WS7-VOICE-RECORDING** `[P2]` — stubs de gravação: `amazon_connect.go:354-369`, `vonage.go:350-368` (S3 presigned / download autenticado).
- [ ] **WS7-SMTP-TIMEOUT** `[P2]` — `smtp.go:63,125,234,281`: `net.Dialer{Timeout}` + deadlines; honrar `ctx`.
- [ ] **WS7-FS-GETCALL** `[P2]` — `freeswitch.go:431-438`: parsear `uuid_getvar`/`uuid_dump` real (hoje retorna dados fabricados).
- [ ] **WS7-VOICE-HANDLER** `[P3]` — `voice/handler.go:53,145,193,241`: assinatura inválida → 401 (não 500); logar erros ignorados.

---

## WS8 — Chat Platforms (Telegram, WebChat)  `[P0/P1]`

> **STATUS (sessão 2026-07-02):** feitos (build/vet/suíte verdes): WS8-TG-SECRET (secret_token registrado no
> setWebhook via chamada HTTP direta — a lib fixada v5.5.1 não tem o campo), WS8-TG-IDS (`strconv.FormatInt`
> no handler), WS8-TG-KEYBOARD (buildKeyboardFromMetadata implementado + CallbackQuery/EditedMessage tratados no
> handler inbound), WS8-TG-DOWNLOAD (timeout+ctx+LimitReader 25MB), WS8-WEBCHAT-DROP (via WS2). PENDENTES:
> WS8-WEBCHAT-AUTH (token de widget assinado — FEITO: HMAC opt-in por `widget_secret`, rate-limit por IP, origin
> endurecido via LINKTOR_WS_ENFORCE_ORIGIN; rota POST /api/v1/webchat/{id}/widget-token no main.go; docs no .env.example),
> WS8-WEBCHAT-UPLOAD/REQFIELDS (P2), WS8-MM-* (mattermost fora de escopo → WS14).


- [ ] **WS8-TG-SECRET** `[P0]` — `telegram/client.go:36-48`: registrar `secret_token` no `SetWebhook` (`WebhookConfig{SecretToken}`);
  sem isso o webhook seguro rejeita 100% dos updates.
- [ ] **WS8-TG-IDS** `[P0]` — `webhook.go:1068-1086`: `strconv.FormatInt` em vez de `string(rune(int64))` (corrompe sender_id/chat_id).
- [ ] **WS8-WEBCHAT-AUTH** `[P1]` — `webchat/handler.go:44-52,200-216` + `main.go:920`: token de widget assinado por canal, rate-limit por IP/sessão, `CheckOrigin` restritivo por config.
- [ ] **WS8-WEBCHAT-DROP** `[P1]` — `websocket.go:268-275` + `adapter.go:169-183`: buffer cheio deve retornar erro/`Failed` (hoje descarta e reporta `Delivered`).
- [ ] **WS8-TG-DOWNLOAD** `[P1]` — `telegram/client.go:172-196`: `http.NewRequestWithContext` + timeout + `io.LimitReader`.
- [ ] **WS8-TG-KEYBOARD** `[P1]` — `telegram/adapter.go:290-314`: implementar `buildKeyboardFromMetadata` (quick_replies → InlineKeyboard); consumir `CallbackQuery` no inbound (`webhook.go:239`).
- [ ] **WS8-WEBCHAT-UPLOAD** `[P2]` — `handler.go:342-388`: rota de upload não registrada; expor com auth + validação de tipo/tamanho, ou remover e ajustar `AllowAttachments`.
- [ ] **WS8-WEBCHAT-REQFIELDS** `[P2]` — `handler.go:148-198`: aplicar `RequireEmail`/`RequireName`.
- [ ] **WS8-TG-VALIDATE** `[P3]` — `telegram/adapter.go:397-402`: remover `ValidateWebhook` sempre-true (código enganoso).
- [ ] **WS8-MM-URLSCHEME** `[P3]` — `mattermost/listener.go:278-287`: validar esquema de `base_url` no `StartChannel`.
- [ ] **WS8-TEAMS-ID** `[P3]` — `teams/client.go:146`: não ignorar erro de unmarshal do id.

---

## WS9 — API, IA, Rate-limit, Observabilidade  `[P2/P3]`

- [ ] **WS9-RL-LOGIN** `[P2]` — `main.go:927`: limite dedicado mais restritivo para `/auth/*` (hoje 100/min/IP).
- [ ] **WS9-RL-FAILOPEN** `[P3]` — `ratelimit.go:65-68`, `webhook_dedup.go:75-78`: fail-open em erro de Redis — manter mas alertar/monitorar.
- [ ] **WS9-FB-READ** `[P3]` — `webhook.go:1575`: read-status do Facebook é no-op (lacuna funcional conhecida).
- [ ] **WS9-OBS-PLACEHOLDER** `[P2]` — `observability_repo.go:58`: `fmt.Sprintf("$%d", i)` (quebra com >9 args).

---

## WS10 — Persistência e Integridade de Dados  `[P2]`

> **STATUS (sessão 2026-07-02):** feitos com testes (build/vet/suíte verdes): WS10-STATUS-MONO (rank + WHERE guard),
> WS10-MARKREAD (sender_type='contact'), WS10-ROWSERR (message/conversation/order repos), WS10-CONV-PAGE (count
> respeita filtros), WS10-CONV-STATUS (IsValid + validação em Update/Create/Assign), WS10-CAMPAIGN-CANCEL (relê
> status entre lotes e a cada 50; Cancel rejeita terminal; CAVEAT multi-réplica documentado — follow-up lock Redis),
> WS10-ORDER-TX (transação order+items). Testes de repo com DB real ficam sob tag `integration`.
> **UPDATE:** WS10-DEDUP-ATOMIC FEITO — migração `00004_message_dedup_index.sql` (índice único parcial
> `(conversation_id, external_id) WHERE external_id IS NOT NULL`) + `Create` com `ON CONFLICT DO NOTHING`
> retornando `ErrCodeConflict` em duplicata (consumer inbound já faz ACK). Teste de integração adicionado.
> WS0-SCHEMA-VALIDATE FEITO — job `migrations` no ci.yml (Postgres pgvector/pgvector:pg16 + `go test -tags=integration`).
> WS1-CONFIG-LEAK FEITO — `entity.Channel.MarshalJSON` redige chaves sensíveis do Config (lista única
> `entity.SensitiveConfigKeys`, +widget_secret); `ChannelService.Update` faz merge preservando segredo mascarado.
> PENDENTES: WS10-GETORCREATE (upsert de contato — não feito),
> WS10-PERSIST-FIELDS, WS10-TPL-EXTID, WS10-INDEXES (compostos), WS10-TPL-META-ERR, WS10-FLOW-NULLSCAN,
> WS10-TENANT-USAGE, WS10-MINIO-PRESIGN, WS10-ESC-MODEL, e o tail P3.


- [ ] **WS10-ROWSERR** — checar `rows.Err()` após iteração em todos os repos (knowledge, message, order, payment, user, role, sla, tenant_settings, bot, campaign, cart, channel, contact, flow, audit).
- [ ] **WS10-STATUS-MONO** — `message_repo.go:202`: status não-regressivo (`WHERE status_rank(status) < status_rank($1)`).
- [ ] **WS10-MARKREAD** — `message_repo.go:283`: filtrar `sender_type='contact'` no `MarkAsRead`.
- [ ] **WS10-CONV-STATUS** — `conversation.go:180,206`: validar transições de status/priority; validar usuário/tenant em `Assign`.
- [ ] **WS10-DEDUP-ATOMIC** — `receive_message.go:89` + `message_repo.go:98`: índice único parcial + `ON CONFLICT DO NOTHING`; escopar `FindByExternalID` por canal.
- [ ] **WS10-GETORCREATE** — `receive_message.go:173,246`: índice único parcial + upsert/advisory lock para conversa/contato (evita duplicatas em rajada).
- [ ] **WS10-ORDER-TX** — `order_repo.go:25-67`: envolver INSERT + itens em transação com rollback.
- [ ] **WS10-CONV-PAGE** — `conversation_repo.go:280-289`: `total` deve respeitar filtros (paginação da UI).
- [ ] **WS10-PERSIST-FIELDS** — persistir campos hoje descartados: `Conversation.Tags/Metadata`, `conversations.escalated_at`
  (KPI de escalação — `escalate_conversation.go:111`), `Channel.Identifier`/`MessageTemplateNamespace`.
- [ ] **WS10-TPL-EXTID** — `postgres.go:621`: índice/upsert de `templates.external_id` escopado a `(tenant_id, channel_id, external_id)`.
- [ ] **WS10-INDEXES** — índices compostos: `messages(conversation_id, created_at)`, `conversations(tenant_id, status)`.
- [ ] **WS10-CAMPAIGN-CANCEL** `[P1]` — `campaign.go:236,260`: reler status no loop de dispatch; lock distribuído (Redis) para múltiplas réplicas; bloquear cancelar `completed`.
- [ ] **WS10-TPL-META-ERR** — `template.go:103,299,332,364`: não persistir template sem `ExternalID` em rejeição da Meta; `SyncToMeta` não seta `LastSyncedAt` em falha; `SyncFromMeta` paginar (cursor).
- [ ] **WS10-FLOW-NULLSCAN** — `flow_repo.go:346`: `COALESCE`/ponteiros para colunas nuláveis.
- [ ] **WS10-TENANT-USAGE** — `tenant.go:114`: `MessagesThisMonth` real (hoje 0 hardcoded); `tenant_repo.go:183` guard p/ settings `null`.
- [ ] **WS10-ILIKE-ESCAPE** `[P3]` — escapar `%`/`_` em buscas ILIKE (knowledge/order/template/contact/canned_response).
- [ ] **WS10-UTC** `[P3]` — usar UTC em Updates; revisar `TO_CHAR(created_at,...)` em analytics (shift de fuso).
- [ ] **WS10-DEDUP-DEAD** `[P3]` — pacote `dedup` é código morto com TOCTOU; deletar ou tornar atômico.
- [ ] **WS10-MINIO-PRESIGN** `[P2]` — `minio.go:53`: guardar a key e presignar na leitura (URL de 7 dias expira → 404).
- [ ] **WS10-ESC-MODEL** `[P2]` — `escalate_conversation.go:482`: modelo `gpt-4o-mini` hardcoded quebra instalações Anthropic/Ollama.

---

## WS11 — Frontend, Deploy e Config  `[P0/P2]`

- [ ] **WS11-CRYPTOKEY** `[P0]` — adicionar `CRYPTO_ENCRYPTION_KEY` a `deploy/.env.example` (boot falha em release sem ela).
- [ ] **WS11-S3ENV** `[P0]` — `deploy/docker-compose.prod.yml:149-154`: renomear `LINKTOR_S3_*` → `MINIO_*` (código lê `MINIO_*`; mídia desativada em silêncio).
- [ ] **WS11-I18N** `[P1]` — `web/admin/src/i18n/request.ts`: merge com `defaultLocale` como fallback; sincronizar chaves órfãs (es/en/pt-BR).
- [ ] **WS11-ENV-DOC** `[P2]` — documentar em `.env.example` (raiz) vars lidas e não documentadas: `OPENAI_ORG_ID`, `OPENAI_DEFAULT_MODEL`,
  `ANTHROPIC_DEFAULT_MODEL`, `OLLAMA_DEFAULT_MODEL`, `MINIO_REGION`, `LINKTOR_GRAPH_API_URL`, `NEXT_PROXY_API_ORIGIN`,
  `NEXT_PUBLIC_SHOW_DEMO_CREDENTIALS`, `MEDIA_UPLOAD_DIR/BASE_URL`.
- [ ] **WS11-CONFIG-PLACEHOLDER** `[P2]` — `config.yaml`: `jwt.secret` ≠ `crypto.encryption_key` (placeholders distintos).
- [ ] **WS11-RQ-RETRY** `[P2]` — `web/admin/src/lib/query.tsx:48-50`: `retry:0` em mutations (evita POST duplicado).
- [ ] **WS11-NPM-AUDIT** `[P2]` — resolver 4 vulnerabilidades npm (1 high, 2 moderate, 1 low): `npm audit fix` + verificar breaking.
- [ ] **WS11-API-PARSE** `[P3]` — `web/admin/src/lib/api.ts:212`: `JSON.parse` com try/catch (resposta não-JSON vira `ApiError` amigável).
- [ ] **WS11-CI** `[P3]` — `.github/workflows/ci.yml`: rodar `golangci-lint` e `next build` isolado do admin.
- [ ] **WS11-DEV-SECRETS** `[P3]` — `docker-compose.yml`: permitir override de `LINKTOR_JWT_SECRET` dev via `${...}`.

---

## WS12 — CLI `msgfy`  `[P1]`

Tratar como bloco separado; correções mecânicas.

- [ ] **WS12-JSONTAGS** — `cmd/cli/internal/client/client.go`: tags JSON snake_case (backend usa snake_case); corrige token vazio no login (`auth.go:200`).
- [ ] **WS12-ROUTES** — alinhar paths/verbos a `main.go`: `/me`, PUT (não PATCH), activate/deactivate, remover publish/execute inexistentes; upload multipart real.
- [ ] **WS12-BASEURL** — `root.go:103`: incluir `/api/v1` no baseURL; remover `SendDirectMessage` morto.
- [ ] **WS12-BACKUP** — `server.go:194-212`: `backup` retornar erro explícito ou implementar (hoje finge sucesso); idem `plugin *`.
- [ ] **WS12-QUERYENC** — `client.go:744-752`: `url.Values.Encode()`.
- [ ] **WS12-SIG-GUARD** — `webhook.go:300`: guard de tamanho antes de `Signature[:20]`.

---

## WS13 — Testes e Cobertura  `[P1/P2]`

Fechar as lacunas que mascararam os bugs (testes assertavam struct em memória, não JSON serializado/parse real).

- [ ] **WS13-TENANT** — suíte de isolamento cross-tenant (WS1-TESTS) — obrigatória no gate.
- [ ] **WS13-WA-JSON** — `ParseWebhook` com `failed`+`error_data` objeto; serialização de quick_reply/coupon (JSON, não struct); consumer.
- [ ] **WS13-FLOWS** — round-trip de criptografia de Flows (IV-flip/base64/tamanho de IV) + builder; hoje zero testes.
- [ ] **WS13-SCHEMA** — smoke de repos contra DB limpo (WS0-SCHEMA-VALIDATE).
- [ ] **WS13-EMAIL-VOICE** — ciclo IMAP (busca/parse/reconexão), `SMTPProvider.Send`/MIME, SES SigV4; Asterisk/FreeSWITCH/Connect providers.
- [ ] **WS13-WEBCHAT** — `Stop()` com clientes ativos, buffer cheio, auth/origin.
- [ ] **WS13-PIPELINE** — envio de interativo/anexo end-to-end (WS4).

---

## WS14 — Desabilitar canais fora de escopo  `[P0]`  ✅ (mecanismo pronto)

> **STATUS (sessão 2026-07-02):** IMPLEMENTADO. `ChannelService.SetEnabledChannelTypes` + checagem em
> `Create` e `Connect` (retorna erro de validação para tipo fora da allowlist). Ligado no `main.go` via
> `LINKTOR_ENABLED_CHANNEL_TYPES` (CSV; vazio = todos permitidos, para não quebrar dev/testes),
> documentado no `.env.example`. Testes verdes. **Para homologação, definir** no ambiente:
> `LINKTOR_ENABLED_CHANNEL_TYPES=whatsapp_official,whatsapp_unofficial,telegram,webchat,facebook,instagram`
> — isso bloqueia sms/rcs/email/voice/teams/slack/mattermost (que têm bugs abertos e ficaram fora do escopo).

Para os não prontos que não entrarem no escopo de homologação, retornar erro claro no registro/`Connect`
(em vez de expor quebrado). Reavaliar caso a caso:
- Email SES, Email SMTP/IMAP inbound.
- RCS providers não-Zenvia (Infobip/Pontaltech/Google) — homologar só Zenvia texto+mídia.
- Voz Asterisk / FreeSWITCH / Amazon Connect.
- WhatsApp Flows (até WS5I-FLOW-* fechados).

*Aceite:* tentar ativar um canal fora de escopo retorna erro explícito e logado; nada silenciosamente quebrado.

---

## Definição de "Pronto para Homologação" (gate)
1. WS0 (schema) + WS1 (tenant) + WS2 P0 (panics/races) + WS3 P0 (fail-open/RCE/JWT/seed/SSRF) completos e testados.
2. WS4 (pipeline: interativos, anexos, bot/IA, status) funcionando end-to-end para os canais em escopo.
3. Canais em escopo (definir lista) com seus P0/P1 fechados; demais desabilitados via WS14.
4. WS11 P0 (deploy) aplicado; smoke de deploy em release passando.
5. Suíte de testes (WS13-TENANT, WS13-SCHEMA, WS13-PIPELINE) verde no CI.

## Próximo passo
Definir a **lista de canais no escopo de homologação** para priorizar WS5–WS8 e decidir o WS14.
Sugestão de arranque: WS0 (fundação) em paralelo com WS1 (tenant) — são independentes e destravam o resto.
