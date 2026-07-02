# Auditoria de Pré-Homologação — LINKTOR (2026-07-02)

Auditoria rigorosa de todos os canais, camada de API/segurança, serviços/persistência,
frontend/deploy e CLI. Build Go, `go vet`, testes Go e `next build` do admin passam limpos —
os defeitos abaixo são de **lógica/segurança/schema**, não de compilação.

**Veredito global: NÃO APTO para homologação.** Bloqueadores sistêmicos em três eixos:
1. Isolamento de tenant (IDOR cross-org) — verificado ponta a ponta (handler→service→repo).
2. Schema divergente do código — features que nunca rodaram contra banco novo.
3. Panics determinísticos e fail-open de segurança em vários adapters de canal.

Canais **prontos**: Slack, Teams. Canais **quase prontos** (correções pontuais): Meta shared,
SMS/Twilio, Email via handler central (SendGrid/Mailgun/Postmark envio), Mattermost.
Frontend admin e infraestrutura: aptos após 3 correções pequenas. CLI `msgfy`: reprovado.

---

## P0 — BLOQUEADORES OBRIGATÓRIOS

### A. Isolamento de tenant (IDOR entre organizações)
Padrão correto existe no projeto (contatos/conversas/orders comparam `TenantID`), mas não foi
aplicado a estes recursos. Correção uniforme: `WHERE id=$1 AND tenant_id=$2` no repo (ou checar
`resource.TenantID != tenantID → 404` no service) + testes de isolamento cross-tenant.

- **Canais** — `channel_repo.go:80,262,323,339,361`; `service/channel.go:130,135,197,212`;
  `handlers/channel.go` Get/Update/Delete/Connect/Disconnect/UpdateStatus/UpdateEnabled/RequestPairCode.
  Vaza `Config` com `access_token`/`app_secret` descriptografados; permite takeover/delete de canal alheio.
- **Bots** — `bot_repo.go:62,168,213,229,258,274`; CRUD + activate + AssignChannel + Test cross-tenant.
- **Knowledge bases e itens** — `knowledge_repo.go:63,122,167,318,377,426,447`; leitura/reescrita/delete + Search vaza RAG.
- **Pagamentos** — `payment_repo.go:69,91,114,165` (interface `PaymentStore` sem org); histórico financeiro por telefone vaza entre tenants.
- **Usuários** — `user_repo.go:154,194` (Update/Delete sem tenant; Get já valida). Admin de A altera role/deleta usuário de B.
- **Carrinhos** — `cart_repo.go:65,93,114,253,267,210,179`.
- **Commerce / WhatsApp Flows** — `commerce.go:33`, `whatsapp_flows.go:29,96,110,143,161`; usam `access_token` do canal alheio.
- **History import** — `history_import_repo.go`; `history_import.go:74` não valida `channel.TenantID`.
- **Família WhatsApp avançada** — `payments.go`, `calling.go`, `ctwa.go`, `whatsapp_analytics.go`
  resolvem `clients[channelID]` sem checar tenant. `POST /channels/{id-de-B}/payments/{p}/refund`
  dispara **reembolso real** via Meta no canal de outro tenant.

### B. Schema divergente do código (verificado contra `postgres.go`)
Diretório goose `migrations/` só tem `.gitkeep`; schema real é o CREATE TABLE embutido em `postgres.go`.

- **`conversation_contexts`** — repo (`bot_repo.go:392-579`) usa `bot_id, intent_name, intent_confidence,
  entities, context_window, state, last_analysis_at`; tabela (`postgres.go:411-421`) só tem
  `context_data, flow_state, intent, sentiment`. Todo Create/Find/Update falha com `42703`. **Verificado.**
- **`ai_responses`** — INSERT (`bot_repo.go:664`) usa `message_id` (coluna inexistente) e omite
  `conversation_id NOT NULL`. Persistência de resposta de IA sempre falha. **Verificado.**
- **Analytics de fluxo/escalação** — `analytics_repo.go:167-169,205` lê `conversations.metadata`,
  `cc.current_flow_id`, `cc.metadata` (inexistentes) → 500.
- **`templates.sub_category`** — `template_repo.go:149` filtra coluna inexistente; `SubCategory` nunca persistido.
- **Busca vetorial** — `knowledge_items.embedding` é `TEXT` sem extensão pgvector, mas `SearchByEmbedding`
  usa operador `<=>` → `operator does not exist`. RAG falha em banco novo.

Ação: criar migrações de alinhamento reais e validar cada caminho contra DB limpo.

### C. Panics determinísticos e crashes
- **Panic no envio de mídia** (nil-pointer por `err` sombreado) — `whatsapp/adapter.go:245-253+`,
  `facebook/adapter.go:157-197`, `instagram/adapter.go:166-201`.
- **`close of closed channel`** — whatsmeow QR relogin (`whatsapp/client.go:140-162`) e Disconnect duplo
  (`whatsapp/adapter.go:137-149`; `email/adapter.go:179-181`).
- **`concurrent map writes`** — `history_import.go:26,111,116,272` (mapa sem mutex, derruba o servidor);
  `conversation_context.go` (State map mutado por mensagens concorrentes na mesma conversa).
- **WebChat Hub pós-`Stop()`** — `webchat/websocket.go:139-166,268-275`: send em canal fechado (panic) +
  goroutines/conexões vazadas.
- **Corridas de socket em voz** — dois `bufio.Reader` no mesmo socket AMI (`asterisk.go`) e ESL (`freeswitch.go`)
  roubam bytes entre resposta e event loop.
- **DoS por IV** — Flows `flows/encryption.go:143-148`: IV de 16 bytes + `NewGCM` (exige 12) → panic.
- **Stack overflow** — `flow_engine.go:180` recursão sem limite; ciclo A→B→A derruba o processo.

### D. Fail-open / bypass de segurança
- **Validação de webhook sempre `true`** — Voice Vonage/Asterisk/Amazon Connect
  (`vonage.go:647`, `asterisk.go:654`, `amazon_connect.go:520`) e Twilio Voice fail-open quando faltam
  headers (`twilio.go:580-584`); SendGrid (`email/webhook.go:499`).
- **Fail-open quando secret vazio** — FB/IG/RCS/WhatsApp oficial validam assinatura mas retornam `true`
  sem secret configurado; só protegido no caminho HTTP em modo `release`.
- **Injeção de shell (RCE)** — `freeswitch.go:518` monta `api system rm %s` com input não sanitizado.
- **Injeção CRLF/XML** — headers de e-mail (`smtp.go`, `ses.go`) e TwiML de voz (`twilio.go` IVR)
  sem escape → header injection / toll-fraud.
- **JWT type confusion** — `auth.go`: access e refresh intercambiáveis, sem claim `typ`, sem revogação.
- **Seed destrutivo** — `seed.go:28` faz `DELETE FROM messages/conversations/.../tenants` quando `users`
  está vazia, mesmo fora de release; credencial fixa `admin123` (`seed.go:50`).

### E. CLI `msgfy` inoperante contra o backend
- Tags JSON camelCase vs snake_case do backend → token salvo vazio (`client.go:35-40`, `auth.go:200`).
- Rotas/verbos errados (maioria dos comandos → 404): `/auth/me` vs `/me`, PATCH vs PUT,
  start/stop vs activate/deactivate, publish/execute inexistentes, upload como JSON em vez de multipart.
- baseURL default sem `/api/v1` (`root.go:103`).
- `server backup` imprime "sucesso" sem criar backup; `plugin *` retornam dados fictícios.

---

## P1 — ALTOS (corrigir antes ou junto da homologação)

### Entrega de mensagens (pipeline central)
- **Interativos/quick replies viram texto puro** — `worker.go:154` sem case `interactive`;
  nenhum sender lê `metadata["interactive"]`. Vale para whatsmeow, WhatsApp oficial, Telegram, RCS, Facebook.
- **Anexos de saída nunca entregues** — `translate()` só olha `meta["media_url"]/["media_id"]` que ninguém seta.
- **Pipeline bot/IA desconectado** — `PublishBotAnalysis/Response/Escalation` nunca chamados;
  consumers recebem tráfego zero. Auto-resposta de bot não funciona.
- **`PublishEvent`/`PublishStatusUpdate` ignorados** — evento perdido em falha, mensagem fica `pending` para sempre.
- **Dedup NAK→DLQ** — duplicata esperada e erros de validação viram NAK 5x → poluição de DLQ e alerta por webhook duplicado.
- **Rate limiter dorme sem ctx dentro do handler NATS** — `worker.go:82,201` estoura AckWait → redelivery → envio duplicado.

### WhatsApp (Meta Cloud API)
- **Parse de webhook quebra em qualquer `failed`** — `types.go:111-116` tipa `error_data` como string; Meta envia objeto → batch inteiro descartado (inclui inbound). **Reproduzido.**
- **quick_reply serializado como `text`** (não `payload`) e **copy_code como `text`** (não `coupon_code`) → rejeição pela Meta.
- **OTP templates com `sub_type` inválido** → `SendOTP` falha.
- **Rate limits reais (130429/131048/131056) classificados como permanente** → mensagens vão pra DLQ em picos.
- **Flows Data Exchange criptografia quebrada** — IV da resposta não é `requestIV ^ 0xFF`; corpo em JSON em vez de Base64 puro; builder sempre falha "duplicate screen ID". 100% das interações de Flow falham.

### Meta family / RCS
- **OAuth engole erro da Meta** — `meta/client.go:362-428` não checa StatusCode; canal salvo com `access_token` vazio.
- **Postbacks (FB) e reactions/story replies (IG) perdidos**; `IsInstagramViaPageWebhook` classifica qualquer webhook de Messenger como IG.
- **RCS**: rich cards/carousels/suggestions/mídia descartados por Infobip/Pontaltech/Google; Google RBM inoperante (auth/UUID errados); Infobip processa só `results[0]`.

### Email / Voz
- **IMAP polling é stub** — `imap.go:278-283` nunca entrega e-mail inbound; sem reconexão em conexão morta.
- **SES SigV4 inválido** (path vazio) → 403 sempre; provider inutilizável.
- **MMS inbound perde mídias**; **anexos por URL nunca enviados** (todos os providers de e-mail).
- **Amazon Connect `GetCall`** usa endpoint errado (sem instanceID) + status hardcoded.

### Segurança/dados
- **Cancel de campanha não interrompe dispatch em voo** — `campaign.go:236,260`; com >1 réplica, dispatch duplicado cobrado pela Meta.
- **SSRF** — `media_processor.go:43`, `whatsapp/adapter.go:700` (`fetchMediaFromURL`) sem allowlist (aceita `169.254.169.254`/localhost) nem `io.LimitReader`.
- **Telegram**: `secret_token` não registrado no setWebhook → webhook seguro rejeita 100% dos updates legítimos; IDs convertidos com `string(rune(int64))` corrompem `sender_id`/`chat_id` (`webhook.go:1068-1086`).
- **`FindByEmail` global** com uniqueness por-tenant → login não-determinístico com email duplicado entre tenants.
- **Meta flow ID nunca persistido** — `whatsapp_flows.go:80` guarda na Description; Update/Delete/Publish falham; sync duplica.
- **Deploy**: `CRYPTO_ENCRYPTION_KEY` ausente do `deploy/.env.example` (boot falha em release);
  compose prod passa `LINKTOR_S3_*` mas código lê `MINIO_*` (mídia silenciosamente desativada).

---

## P2 — MÉDIOS (integridade de dados — tickets em sequência)

- `rows.Err()` nunca checado na maioria dos repos → listas truncadas retornadas como sucesso.
- Status de mensagem não monotônico (redelivery regride delivered→sent); `MarkAsRead` sem filtro `sender_type`.
- Status/priority de conversa aceitos como string crua sem validação.
- Dedup de mensagem inbound não atômico + sem índice único (redelivery/retry duplica); get-or-create de conversa/contato racy.
- Order não-transacional (`order_repo.go:25-67`); paginação de conversas conta antes dos filtros.
- Campos nunca persistidos: `Conversation.Tags/Metadata`, `conversations.escalated_at` (KPI de escalação sempre zero), `Channel.Identifier`.
- HMAC de webhook externo não cobre timestamp (replay); `templates.external_id` único global (sobrescreve entre tenants).
- Índices compostos ausentes: `messages(conversation_id, created_at)`, `conversations(tenant_id, status)`.
- i18n sem fallback entre locales + chaves faltantes (es/en/pt-BR dessincronizados).
- React Query `retry:1` em mutations (POST não-idempotente duplicado); rate limit de login permissivo (100/min/IP).
- `/ai/complete` é proxy LLM genérico sem teto de `max_tokens` nem allowlist de modelo (custo ilimitado).
- 4 vulnerabilidades npm (1 high, 2 moderate, 1 low) no admin — rodar `npm audit fix`.

---

## Ordem de correção sugerida (gate de homologação)

1. **Isolamento de tenant (A)** — mesma classe de bug, correção replicável; adicionar testes cross-tenant.
2. **Schema (B)** — migrações de alinhamento + validação contra DB limpo; sem isso features de bot/IA/RAG/analytics não sobem.
3. **Panics e fail-open (C, D)** — cheap fixes de alto impacto; fechar a superfície pública de webhook.
4. **Entrega de mensagens (P1 pipeline)** — sem isso o produto não entrega interativos/anexos/auto-resposta.
5. **Por-canal**: priorizar os canais do escopo real de homologação; desabilitar explicitamente os não prontos
   (SES, SMTP/IMAP, RCS não-Zenvia, voz Asterisk/FreeSWITCH/Connect, Flows) em vez de expô-los quebrados.
6. **CLI** — tratar como bloco separado ou corrigir integralmente (correções mecânicas).
7. **P2** — tickets de integridade em sequência.
