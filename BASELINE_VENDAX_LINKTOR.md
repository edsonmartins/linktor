# Baseline — Linktor — 2026-07-02

> Auditoria read-only para o roadmap VendaX Sales Copilot (IntegrAllTech).
> Método: build + suíte de testes executados de verdade; inventário por leitura de código (não de marketing); cruzamento com as auditorias internas do próprio projeto (`docs/auditoria-homologacao-2026-07.md` e `docs/plano-correcoes-homologacao.md`).

## 1. Identificação

- **Nome:** Linktor (module `github.com/msgfy/linktor`, "powered by msgfy engine")
- **Propósito:** Plataforma B2B open-source de mensageria multicanal com IA — unifica WhatsApp, Telegram, SMS, WebChat, Instagram, Facebook (e mais) em uma interface única de atendimento, com bots/IA, campanhas, RBAC e trilha de auditoria.
- **Stack:** Go 1.25 (Gin, pgx/pgxpool, NATS JetStream, Redis, MinIO, goose, whatsmeow, chromedp), PostgreSQL 16 + pgvector, frontend Next.js 15.5/React 19 (admin) + Docusaurus (docs) + landing + widget embed. MCP server em TypeScript (`mcp/linktor-mcp-server`). Evidência: `go.mod`, `docker-compose.yml`, `web/admin/package.json`.
- **Último commit:** 2026-07-02 19:10 (`2002a4d`, "fix(email): preserve threading/HTML/attachments…") — projeto **ativo hoje**, ritmo intenso de correções pós-auditoria interna.
- **Contribuidores:** 1 pessoa (Edson Martins, 111 commits somando as duas grafias — `git shortlog -sn`). Bus factor = 1.
- **Governança/spec:** não há ADRs/RFCs/OpenSpec formais. A "spec" são blueprints em `documentos/` (`LINKTOR-PROJECT-SPEC.md` 70KB, blueprint WhatsApp 77KB, guia de Coexistence, spec do VRE) + planos em `docs/`. O projeto tem cultura recorrente de auto-auditoria (2 auditorias em abril em `documentos/`, 1 em julho em `docs/`).

## 2. Saúde de build e testes

| Verificação | Comando | Resultado real |
|---|---|---|
| Build | `go build ./...` | **OK** (exit 0, zero erros) |
| Testes | `go test ./...` | **OK** (exit 0) — 46 pacotes `ok`, 0 falhas |
| Contagem | `go test ./... -json` | **3.278 testes executados / 0 falhando / 8 pulados** (202 arquivos `*_test.go`) |
| Cobertura | — | Não medida nesta auditoria (não há relatório versionado) |
| Frontend admin | — | NÃO VERIFICADO nesta máquina (lockfile + `node_modules` presentes; auditoria interna afirma que `next build` passa limpo). 23 specs Playwright em `web/admin/e2e/` |
| Execução local | `docker ps` / portas 5432/6379/4222/8081 | **NÃO TESTADO** — Docker/podman/colima ausentes na máquina de auditoria e nenhum serviço nativo escutando. Stack completa definida em `docker-compose.yml` (pgvector:pg16, redis:7.4, nats:2.10 -js, minio, backend :8081, admin :3000). `GET /health` (liveness) e `GET /ready` (checa Postgres+Redis+NATS, 503 se falhar) existem em `cmd/server/main.go:978-979` |
| Testes de integração DB | — | Rodam em CI com `pgvector/pgvector:pg16` sob build tag `integration` (commit `1c38f65`); validação viva de schema contra DB limpo marcada como pendente no próprio plano |

## 3. Capabilities e maturidade

Escala: 0 inexistente · 1 spec · 2 protótipo · 3 alfa · 4 beta/homologável · 5 produção.

| Capability | Maturidade (0-5) | Spec/Código | Evidência |
|---|---|---|---|
| Core de mensageria (contacts/conversations/messages, dedup, get-or-create atômico) | 4 | código funcional | `internal/application/usecase/receive_message.go`, `send_message.go`; 36 test files em `service/`; commits `d884ed3`, `81d6764` |
| Outbox transacional + eventos NATS | 4 | código funcional | `internal/infrastructure/outbox/relay.go`; `nats/subjects.go`; escrita atômica msg+evento (`CreateWithOutboxEvent`); commits `c46958a`..`fda2ad4` |
| WhatsApp Cloud API (oficial) | 4 | código funcional | `internal/adapters/whatsapp_official/` (12 test files), `internal/adapters/meta/client.go`; webhook fail-closed |
| WhatsApp não-oficial (whatsmeow, QR/pair) | 3 | código funcional | `internal/adapters/whatsapp/adapter.go`; registrado em `main.go:488`; depende de sessão — NÃO VERIFICADO em runtime |
| WhatsApp Coexistence (Business App + Cloud API, echoes, import de histórico) | 3 | código funcional | `entity/channel.go:118-129`, `service/coexistence_monitor.go`, `whatsapp_embedded_signup.go`; NÃO VERIFICADO sem ambiente Meta |
| WhatsApp avançado: Flows, Commerce/orders/carts, Payments, Calling, CTWA, Analytics | 3 | código funcional | `internal/whatsapp/{flows,commerce,payments,calling,ctwa,analytics}/` com testes; rotas em `main.go:1299-1360`; NÃO VERIFICADO contra Meta real |
| Telegram | 4 | código funcional | `internal/adapters/telegram/` (6 test files); webhook `main.go:1020` |
| Facebook Messenger / Instagram DM | 4 | código funcional | adapters + webhooks `main.go:1028-1030`; postback corrigido em `119dcf1` |
| WebChat (widget + WebSocket público + upload de mídia) | 4 | código funcional | `internal/adapters/webchat/handler.go`; `/ws/:channelId` `main.go:985`; widget-token |
| SMS (Twilio apenas) | 3 | código parcial | reativado em `458ec43`; MMS inbound perde mídia (`sms/adapter.go:233` not implemented); Vonage/Plivo do README **não existem** |
| Email (SendGrid/Mailgun/Postmark ok; SES/SMTP/IMAP quebrados ou gated) | 3 | código parcial | `internal/adapters/email/` (13 test files); `2002a4d` corrigiu threading/HTML; SES parser é stub no-op (`ses_test.go:147,160`); IMAP inbound gated |
| Teams / Slack / Mattermost | 3 | código parcial | senders em `main.go:402-404`; Mattermost via listener WS (`main.go:628`); 1-2 test files cada; recomendados a desabilitar na homologação (WS14) |
| RCS | 2 | código parcial | esqueleto text-only; cards/carrosséis descartados; auth Google RBM inoperante (auditoria interna); webhook `return true` sem secret (`rcs/client.go:71`) |
| Voice (Twilio/Vonage/Asterisk/FreeSWITCH/Amazon Connect) | 2 | código morto no runtime | `internal/adapters/voice/` NÃO é importado em `main.go`; sem rota; contém RCE conhecido (FreeSWITCH `api system rm`) e fail-opens — perigoso se algum dia for ligado sem revisão |
| Bots + IA (OpenAI/Anthropic/Ollama; intent, sentimento, geração, escalation) | 3 | código funcional | `internal/adapters/ai/{openai,anthropic,ollama}/` com testes; pipeline reconectado em `e6283a0`; modelo de escalation hardcoded; quota por-tenant ausente (`handlers/ai.go:151` TODO) |
| Knowledge base / RAG (pgvector, embeddings) | 3 | código funcional | rotas `/knowledge-bases` `main.go:1219-1234`; migração `00003_pgvector_embedding.sql`; NÃO VERIFICADO com DB real nesta auditoria |
| Flow engine (árvores de decisão) | 3 | código funcional | rotas `/flows` `main.go:1249-1259`; stack overflow corrigido pós-auditoria (`215862f`) |
| Campanhas em massa (retry, DLQ, progresso) | 4 | código funcional | rotas `/campaigns` `main.go:1438-1447`; e2e Playwright `campaigns` |
| RBAC granular + audit trail + API keys | 4 | código funcional | `middleware/permission.go`, `role_repo.go`, `api_key_repo.go`, audit middleware; e2e `roles`/`rbac`/`audit-logs` |
| Criptografia de credenciais em repouso (AES-256-GCM + rotação) | 4 | código funcional | `pkg/crypto/crypto.go:42-133`, `channel_secrets.go`; ressalva: redação de segredos na resposta HTTP pendente (WS1-CONFIG-LEAK) |
| Atribuição automática + SLA + auto-close | 3 | código funcional | `service/assignment.go`; rotas `/settings` `main.go:1422-1428`; NÃO VERIFICADO em runtime |
| VRE — Visual Response Engine (render headless chromedp) | 3 | código funcional | `internal/infrastructure/vre/{pool,renderer}.go`, `main.go:296-332`, página admin + e2e; upload CDN pendente; janela 24h é código morto |
| Templates WhatsApp (sync Meta, library) | 4 | código funcional | rotas `main.go:1262-1275`; rejeição Meta não persiste mais (`9f089e0`) |
| Observabilidade (Prometheus /metrics, lag JetStream, DLQ, alertas) | 4 | código funcional | `2f1565c`, `3f74842`, `af2f99c`; rotas `/observability` admin-only |
| Admin dashboard (Next.js, 21 rotas de página) | 4 | código funcional | `web/admin/src/app/(dashboard)/` — paridade ampla com o backend; 23 specs e2e |
| MCP server (~59 tools, stdio + HTTP) | 3 | código funcional | `mcp/linktor-mcp-server/` TS, cliente REST do Linktor; buildável; NÃO VERIFICADO end-to-end nesta auditoria |
| SDKs (Go, Python, TS, Java, PHP, Rust, .NET) + CLI | 3 / CLI 2 | código parcial | `sdks/*` com clients, types e alguns testes (não esqueletos); **CLI `msgfy` inoperante** (auditoria interna: login com token vazio, rotas 404, `backup`/`plugin` fingem sucesso; zero commits de correção) |
| API gRPC pública | 1 | só spec | `proto/` (6 arquivos) + `buf.yaml`; **nenhum** `.pb.go` gerado, nenhum `grpc.NewServer` no servidor |
| Plugin system out-of-process (HashiCorp go-plugin) | 2 | código morto | `pkg/plugin/loader.go` sem nenhum caller; o registry usado é in-process. Promessa do `execucao.md` nunca conectada |
| Grupos (handler Group) | 2 | stub que mente | `handlers/group.go:60` retorna "Placeholder Group" hardcoded |

## 4. Superfícies expostas (o que dá para consumir hoje)

- **REST `/api/v1`** — FUNCIONAL. ~35 grupos montados em `cmd/server/main.go:988-1447`: auth, conversations (+assign/resolve/escalate/messages), contacts, channels (+connect/pair/coexistence), bots, ai, knowledge-bases, flows, templates, analytics, whatsapp/flows, orders, payments, calls, ctwa, vre, users, api-keys, roles, canned-responses, settings, campaigns, audit-logs, observability, oauth (FB/IG/WA embedded signup). Auth por JWT (Bearer ou cookie HttpOnly) ou API key (`X-API-Key`), rate-limit global.
- **Health/ops** — FUNCIONAL: `GET /health` (liveness), `GET /ready` (fail-hard em Postgres/Redis/NATS), `GET /metrics` (Prometheus, público — restringir no ingress), `GET /swagger/*`.
- **Webhooks inbound** — FUNCIONAL: `/api/v1/webhooks/{whatsapp,telegram,teams,slack,twilio|sms,facebook|messenger,instagram,rcs,email(+sendgrid/mailgun/ses/postmark),generic,status,payments,calls,ctwa}/:channelId` com dedup Redis 6h. **Atenção:** validação de assinatura é fail-closed só para Meta; SendGrid/Postmark/RCS/voz são fail-open (ver §6).
- **Eventos NATS** — FUNCIONAL: streams `LINKTOR_{MESSAGES,EVENTS,WEBHOOKS,DLQ,AI,WHATSAPP}`; subjects `linktor.messages.{inbound,outbound,status}.<canal>`, `linktor.events.<tipo>` (message.received/sent/…, conversation.created/assigned/…, contact.created/updated), `linktor.bot.*`, `linktor.whatsapp.*`, `linktor.dlq.>`. Publicação via outbox transacional com idempotency key (efetivamente exactly-once).
- **WebSocket** — FUNCIONAL: `/ws/:channelId` (WebChat público, widget-token opcional) e `/api/v1/ws` (agente autenticado; eventos new_message, conversation_updated, typing, presence).
- **MCP** — `mcp/linktor-mcp-server` (TS, `@modelcontextprotocol/sdk`): ~59 tools (channels 17, vre 10, contacts 9, conversations 8, messages 5, bots 5, knowledge 3, analytics 2) + resources + prompts, transportes stdio e HTTP. Processo separado que consome a REST. Buildável; funcionamento end-to-end NÃO VERIFICADO.
- **SDKs** — 7 linguagens com clients/types (TS e Java com WebSocket); atualidade vs. API atual NÃO VERIFICADA. CLI Go `cmd/cli` existe porém **inoperante** (auditoria interna).
- **gRPC** — NADA consumível: só definições `proto/`, sem servidor.

## 5. Integrações consumidas

| Dependência | Uso | Configurável por tenant? |
|---|---|---|
| Meta Graph API (WhatsApp Cloud, FB, IG) | canais, templates, flows, payments, commerce | **Sim** — credenciais por canal em `channel.Config`/`Credentials` (cifradas); embedded signup por tenant |
| whatsmeow (protocolo WA multi-device) | canal WA não-oficial | Sim — sessão por canal (QR/pair) |
| Telegram Bot API | canal | Sim — token por canal |
| Twilio | SMS (e voz, morto) | Sim — creds por canal |
| SendGrid / Mailgun / Postmark / SES / SMTP / IMAP | email | Sim — por canal (SES/SMTP/IMAP quebrados/gated) |
| Microsoft Teams (Bot Framework/AAD), Slack, Mattermost | canais | Sim — por canal; rota `/teams` compartilhada resolve tenant por audience |
| OpenAI / Anthropic / **Ollama (local)** | IA (intent, sentimento, geração, embeddings) | Provider configurável; **modelo de escalation hardcoded `gpt-4o-mini`** (pendência WS10-ESC-MODEL); quota por-tenant ausente |
| PostgreSQL+pgvector, Redis, NATS JetStream, MinIO | infraestrutura | Por instância (env/config), não por tenant |
| Google RBM (RCS) | canal RCS | Por canal, mas auth inoperante |
| Outros produtos IntegrAllTech | — | **Nenhuma integração encontrada** (sem referência a ArchFlow/Mentors/TaxEngine/RecomX/RouteX/Brain Sentry no código) |

## 6. Multi-tenancy, segurança e dados

- **Isolamento:** row-level por coluna `tenant_id` em schema compartilhado. **Sem RLS no Postgres** (zero `CREATE POLICY`). Tenant vem sempre do JWT/API-key (`middleware/auth.go:79,94`), nunca do request.
- **IDOR (bloqueador da auditoria de julho):** fechado nos endpoints HTTP conhecidos via wrappers `*ForTenant` na camada de serviço (`service/channel.go:172-239`, `service/user.go:96-168`, guards em payments/calling/ctwa/commerce — incl. o vetor de **refund cross-tenant**). **Porém os repositórios continuam tenant-cegos** (`channel_repo.go:80` `WHERE id=$1`; `payment_repo.go:69,114` e `cart_repo.go` sem filtro de organização) — qualquer novo caller do repo cru reintroduz o IDOR. Pendências abertas no próprio plano: WS1-PAYMENTS-REPO, WS1-CARTS, WS1-CONFIG-LEAK (segredos na resposta HTTP), WS1-DEFENSE, WS1-EMAILUNIQ (`user_repo.FindByEmail` global).
- **Auth:** JWT com claim `typ` (access/refresh — type confusion corrigido, `service/auth.go:20-32`), cookie HttpOnly, API keys com papel restrito, RBAC por (recurso×ação) com cache Redis, audit trail.
- **Webhooks:** Meta fail-closed; **SendGrid é stub `return true`** (`email/webhook.go:499-506`), Postmark/RCS/Twilio-Voice/Vonage/Asterisk/Connect/SNS fail-open — WS3-WEBHOOK-FAILOPEN segue **P0 pendente**.
- **Segredos:** placeholders de dev em `config.yaml:14,33,42` e valores embutidos em `docker-compose.yml` (existência apontada, valores não transcritos). Mitigação real: em modo release o boot **recusa** secrets vazios/placeholder/<32 chars (`config/config.go:141-164`). Seed destrutivo e senha default `admin123` removidos (`seed.go` — senha por env ou aleatória; reset atrás de `SEED_RESET_DESTRUCTIVE`). `.env` é symlink para fora do repo.
- **Cripto em repouso:** AES-256-GCM real com rotação de chave (`pkg/crypto/crypto.go`), aplicada a credenciais/config sensível de canais.
- **Migrations/dados:** goose em uso — baseline idempotente (versão 1, `postgres.go:70`) + 8 migrações versionadas 00002–00009 (`internal/infrastructure/database/migrations/`: align_repo_schema, pgvector, dedup index, flow_meta_id, composite_indexes, contact_identity_unique, conversation_tags, outbox_events). Banco sobe do zero por desenho; CI valida com pgvector:pg16; validação local viva NÃO TESTADA aqui (sem Docker). Entidades centrais: tenant, contact, conversation, message, channel, campaign, user, role, bot, flow, knowledge, template, order/cart, escalation, outbox (~25 entidades em `internal/domain/entity/`).

## 7. Dívidas, bloqueios e spec drift

- **Veredito da auditoria interna (2026-07-02): "NÃO APTO para homologação".** O git log mostra que a maioria dos P0 foi corrigida DEPOIS, com commits rastreáveis e testes (schema/migrações, IDOR HTTP, panics/races `215862f`, JWT/seed/SSRF `d8513e4`, pipeline de entrega + outbox, readiness fail-hard `a131a8a`). Restam P0: fail-open de webhooks não-Meta e pendências WS1 de tenant.
- **Spec drift severo no README:** tabela de canais anuncia **10 canais "✅ Completo"** — falso para Voice (código morto, não integrado, com RCE conhecido), SMS (só Twilio; Vonage/Plivo inexistentes), RCS (text-only), Email (SES/SMTP/IMAP). Estratégia adotada: **desabilitar canais quebrados por allowlist `LINKTOR_ENABLED_CHANNEL_TYPES`** em vez de corrigi-los — deploy sem essa var expõe canais quebrados (vazio = todos permitidos).
- **`execucao.md` obsoleto e enganoso** (última atualização 2026-02-03; diz que WhatsApp oficial está 0% e proíbe testes/observabilidade — tudo contradito pela realidade). Fonte de verdade atual: `docs/plano-correcoes-homologacao.md`.
- **Código morto:** HashiCorp go-plugin/gRPC out-of-process (zero callers), adapter Voice inteiro, `SessionAwareConsumer` (janela 24h WA), pacote `dedup` com TOCTOU, `Adapter.ProcessWebhook` de vários canais.
- **Stubs que mentem:** `Group.Get` retorna placeholder hardcoded; `tenant.MessagesThisMonth` sempre 0; parsers SES no-op; recording de voz fabricado.
- **CLI `msgfy`:** reprovado na auditoria, zero commits de correção (WS12 inteiro em aberto).
- **TODOs inline:** apenas 1 real (`handlers/ai.go:151`, quota por-tenant) — a dívida vive nos docs de auditoria, não no código.
- **Frontend:** 4 vulnerabilidades npm sem resolver (1 high); `retry:1` em mutations pode duplicar POSTs.
- **Bus factor 1** e ausência de ADRs formais.

## 8. Prontidão para o VendaX

**O que o VendaX consegue usar HOJE:**
- O **hub de mensageria multicanal** como serviço: REST `/api/v1` completa (conversas, mensagens, contatos, canais), WebSocket de agente, eventos NATS com outbox (integração assíncrona confiável — ideal para o copiloto reagir a `message.received`), e o **MCP server com ~59 tools** — superfície pronta para um agente de vendas operar conversas (ler, responder, consultar contato/canal/analytics) sem escrever integração nova.
- Canais **prontos**: WhatsApp Cloud API (o mais maduro, com templates/flows/commerce/payments/CTWA), Telegram, WebChat, Facebook/Instagram. WhatsApp não-oficial via **whatsmeow** (não Evolution — ver nota abaixo).
- Multi-tenancy, RBAC, API keys, audit e criptografia de credenciais — fundação de plataforma que o VendaX não precisa reconstruir.

**O que precisa de trabalho para o caso de uso VendaX:**
1. **Conversa a três / copiloto NÃO existe** (ver nota específica) — o modelo é 1 contato × 1 agente-ou-bot com handoff sequencial. Introduzir `vendorId`/participantes múltiplos/sugestões de copiloto é mudança de modelo de domínio (entidade Conversation, envelope, WS de agente, admin). Estimativa: **4-6 semanas-pessoa**.
2. **Envelope:** falta `vendorId` no envelope NATS e na entidade; `Message` não carrega `tenantId` direto (ancora em Conversation). Extensão do envelope: **1-2 semanas-pessoa** (junto do item 1).
3. **Fechar P0 residuais de segurança** (webhooks fail-open, WS1 de tenant, config-leak): **2-3 semanas-pessoa**.
4. **Validação de homologação real** (subir stack, testar canais contra provedores reais, validar Coexistence/pagamentos com conta Meta): **2-3 semanas-pessoa**.

**Estimativa para "homologável" (núcleo multicanal, canais quebrados desabilitados por allowlist):** **4-6 semanas-pessoa**. Com as extensões que o VendaX exige (conversa a três + vendorId): **10-14 semanas-pessoa**. Reativar RCS/Voice/Email-completo: +8-12 semanas-pessoa (provavelmente descopar).

## 9. Resumo executivo em 5 linhas

1. Linktor é um hub de mensageria multicanal Go **real e ativo** (commit hoje, 1 dev): build limpo, 3.278 testes passando, 8 canais funcionais — WhatsApp Cloud API é o mais maduro; **Evolution API não existe** (via não-oficial é whatsmeow).
2. A própria auditoria interna de 2026-07 reprovou a homologação; o git log comprova que a maioria dos P0 (IDOR HTTP, schema, panics, pipeline) foi corrigida depois — mas webhooks fail-open (SendGrid/voz/RCS) e defense-in-depth de tenant nos repositórios **seguem abertos**.
3. Superfícies consumíveis hoje: REST ampla, eventos NATS com outbox transacional, WebSocket, MCP server com ~59 tools — o VendaX consegue plugar um agente de vendas nisso **hoje**.
4. Spec drift severo: README anuncia 10 canais "completos" quando Voice é código morto com RCE conhecido e SMS/RCS/Email são parciais; a mitigação real é allowlist de canais, não correção.
5. O gap decisivo para o VendaX é conceitual, não de infraestrutura: **não há conversa a três nem vendorId** — o modelo é cliente×(bot OU agente) com handoff; estimar 10-14 semanas-pessoa para homologável já com as extensões do copiloto.

---

## Nota específica — Linktor (pergunta do projeto)

**1. O envelope normalizado (tenantId, vendorId, customerId, messageType, content) está implementado de ponta a ponta?**
**Parcialmente — 4 dos 5 campos, e a simetria outbound é incompleta.** O envelope de transporte NATS (`internal/infrastructure/nats/producer.go:44-73`, structs `InboundMessage`/`OutboundMessage`) carrega `TenantID`, `ChannelID`, `ContactID` (=customerId), `ConversationID`, `ContentType` (=messageType, normalizado para 12+ tipos: text/image/video/audio/document/location/contact/template/interactive/sticker/poll/reaction), `Content`, `Metadata` e `Attachments`. O fluxo inbound é de ponta a ponta e sólido: webhook → NATS → `receive_message.go` (canal como fonte de verdade do tenant, dedup por ExternalID) → `MessageNormalizer` → get-or-create atômico de contato/conversa → persistência + evento na mesma transação (outbox). **Faltas:** (a) **`vendorId` não existe** em nenhuma camada — não há conceito de vendedor no domínio; (b) a entidade `Message` persistida não carrega `tenantId`/`contactId` diretos (ancora em `Conversation`); (c) no outbound, a denormalização centralizada (`DenormalizeForChannel`) só cobre whatsapp/telegram/webchat/sms — os demais canais formatam a saída dentro de cada adapter, ou seja, o envelope é unificado mas a saída é parcialmente duplicada.

**2. Quais canais funcionam HOJE? WhatsApp via Evolution?**
**Evolution API: NÃO existe nenhuma integração** — `grep -ri evolution` só encontra menções em documentos de comparação, onde a Evolution é citada como *concorrente* (baseada em Baileys). Os modos WhatsApp reais são: **Meta Cloud API oficial** (o adapter mais completo, com templates/Flows/Commerce/Payments/Calling/CTWA), **whatsmeow** (não-oficial, Go, QR/pair — não Baileys) e **Coexistence** (Business App + Cloud API com echoes e import de histórico). Canais funcionais hoje (código + webhook + testes + registrados no runtime): **WhatsApp oficial, WhatsApp whatsmeow, Telegram, WebChat, Facebook Messenger, Instagram, Email (SendGrid/Mailgun/Postmark), SMS (só Twilio)**; parciais: Teams, Slack, Mattermost, RCS (text-only); morto: Voice (não integrado ao runtime). Em produção a exposição real depende da allowlist `LINKTOR_ENABLED_CHANNEL_TYPES`.

**3. Estado da conversa a três?**
**Inexistente (maturidade 0 — nem spec).** `Conversation` tem um único `AssignedUserID *string` (`entity/conversation.go:55`); não há lista de participantes, nem conceitos de handoff simultâneo, takeover ou copilot (greps por participants/handoff/takeover/copilot/vendor/seller só encontram grupos WhatsApp e chamadas de voz, irrelevantes). O que existe é **escalation de duas partes**: o bot atende e, ao escalar (`usecase/escalate_conversation.go`, rota `POST /conversations/:id/escalate`), sai e um humano entra — sequencial, nunca simultâneo. Para o VendaX (cliente + vendedor + copiloto IA), será preciso evoluir o modelo de domínio: participantes múltiplos na conversa, `vendorId` no envelope, canal de sugestões do copiloto para o vendedor (o WebSocket de agente existente é a base natural para isso).

---

```yaml
baseline:
  projeto: "Linktor"
  data: "2026-07-02"
  ultimo_commit: "2026-07-02"
  build_ok: true
  testes: "3278 passando / 0 falhando / 8 pulados"
  sobe_localmente: nao_testado   # docker-compose completo existe; sem runtime Docker na máquina de auditoria
  maturidade_geral: 3
  capabilities:
    - nome: "core mensageria (contatos/conversas/mensagens)"
      maturidade: 4
      estado: "funcional"
    - nome: "eventos NATS + outbox transacional"
      maturidade: 4
      estado: "funcional"
    - nome: "WhatsApp Cloud API oficial (+templates/flows/commerce/payments/ctwa)"
      maturidade: 4
      estado: "funcional"
    - nome: "WhatsApp whatsmeow (nao-oficial) + Coexistence"
      maturidade: 3
      estado: "funcional"
    - nome: "Telegram / Facebook / Instagram / WebChat"
      maturidade: 4
      estado: "funcional"
    - nome: "SMS (so Twilio) / Email (parcial) / Teams / Slack / Mattermost"
      maturidade: 3
      estado: "parcial"
    - nome: "RCS (text-only)"
      maturidade: 2
      estado: "parcial"
    - nome: "Voice (nao integrado ao runtime, RCE conhecido)"
      maturidade: 2
      estado: "parcial"
    - nome: "bots + IA (OpenAI/Anthropic/Ollama) + RAG pgvector"
      maturidade: 3
      estado: "funcional"
    - nome: "campanhas em massa"
      maturidade: 4
      estado: "funcional"
    - nome: "RBAC + audit + API keys + cripto AES-256-GCM"
      maturidade: 4
      estado: "funcional"
    - nome: "admin Next.js (21 paginas, 23 e2e)"
      maturidade: 4
      estado: "funcional"
    - nome: "MCP server (~59 tools)"
      maturidade: 3
      estado: "funcional"
    - nome: "SDKs 7 linguagens"
      maturidade: 3
      estado: "parcial"
    - nome: "CLI msgfy"
      maturidade: 2
      estado: "parcial"
    - nome: "gRPC publico"
      maturidade: 1
      estado: "spec"
    - nome: "conversa a tres / copiloto vendedor"
      maturidade: 0
      estado: "spec"
  superficies_consumiveis_hoje:
    - "REST /api/v1 (~35 grupos: conversations, messages, contacts, channels, bots, ai, knowledge-bases, campaigns, templates, analytics...)"
    - "Webhooks inbound 12+ provedores em /api/v1/webhooks/*"
    - "Eventos NATS linktor.messages.*/linktor.events.* via outbox transacional"
    - "WebSocket /api/v1/ws (agente) e /ws/:channelId (webchat)"
    - "MCP server @linktor/mcp-server (~59 tools, stdio+HTTP)"
    - "SDKs go/python/typescript/java/php/rust/dotnet"
    - "GET /health, /ready, /metrics (Prometheus), /swagger"
  bloqueadores_criticos:
    - "Webhooks fail-open: SendGrid stub 'return true'; Postmark/RCS/voz/SNS sem validacao (P0 pendente WS3)"
    - "Isolamento de tenant so na camada de servico; repositorios tenant-cegos, sem RLS (WS1 pendencias: payments/carts/config-leak/FindByEmail global)"
    - "Segredos de canal podem vazar na resposta HTTP (WS1-CONFIG-LEAK pendente)"
    - "Canais quebrados dependem da allowlist LINKTOR_ENABLED_CHANNEL_TYPES (vazio = todos expostos, incl. codigo perigoso)"
    - "CLI msgfy inoperante; codigo morto perigoso (Voice com RCE FreeSWITCH) no repo"
    - "Validacao de homologacao em ambiente real nunca executada (sobe_localmente nao testado)"
  esforco_para_homologavel: "4-6 semanas-pessoa (nucleo, canais quebrados desabilitados); 10-14 semanas-pessoa incluindo extensoes VendaX (conversa a tres + vendorId)"
  pronto_para_vendax: "parcial"
```
