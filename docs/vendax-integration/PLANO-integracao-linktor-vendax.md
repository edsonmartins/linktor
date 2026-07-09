# Plano de Integração — Linktor ↔ VendaX Core

> Status: **PROPOSTA** (planejamento). Data: 2026-07-08.
> Autor: sessão de planejamento (Claude) — a validar com Edson.
> Escopo: como o **Linktor** (hub de mensageria multicanal, Go) passa a ser o canal do
> **VendaX Sales Copilot** (Core em Java/Spring), integrados por **NATS**. Nenhum código foi
> alterado ao escrever este plano.

---

## 0. TL;DR — a decisão central

**No Linktor a conversa é sempre _vendedor ↔ cliente_ (dois participantes reais). O Agente de IA do
VendaX NUNCA entra na conversa do canal — ele assiste e sugere ao vendedor _apenas dentro do VendaX_,
e o que chega ao cliente é sempre uma mensagem do vendedor. Não existe "conversa a três" no Linktor.**

Consequência direta: **o modelo de domínio atual do Linktor já basta.** `Conversation.AssignedUserID`
(um único responsável) modela exatamente o par vendedor↔cliente — **o vendedor do VendaX é o usuário/
agente atribuído à conversa no Linktor.** Logo:

- O **`vendorId`** do Core mapeia para o **usuário atribuído** no Linktor (`AssignedUserID`) — não é um
  conceito novo no domínio, é um mapeamento vendedor↔user.
- O Linktor entra como um **bridge/adapter fino** que traduz contrato (subjects/envelopes) e transporta
  as mensagens vendedor↔cliente. **Nenhuma reforma de modelo** (participantes múltiplos, copilot, etc.).

Consequência de custo: a estimativa do baseline (4–6 semanas-pessoa para "conversa a três + vendorId"
como mudança de modelo) **não se aplica** — ela partiu do pressuposto de que a IA seria um participante
do canal. Como a IA vive só no VendaX, o gap de domínio do Linktor é **praticamente zero**.

---

## 1. Contexto

- **VendaX Core** (Java) = domínio + orquestração. NÃO integra canais direto — isso é o **Linktor**.
  O contrato de borda com o Linktor **já existe no Core** (envelopes + subjects NATS), mas o lado
  Linktor **ainda não fala esses subjects** (grep por `vendax`/`core.outbound`/`linktor.inbound` no
  código Go = zero).
- **Linktor** (Go) = hub multicanal maduro (WhatsApp oficial/whatsmeow/Coexistence, Telegram, Meta,
  WebChat, SMS; REST ampla, eventos NATS com outbox transacional, WebSocket de agente, MCP). É produto
  próprio ("msgfy engine"), multi-tenant, com credenciais de canal criptografadas.
- Ambos usam **NATS + transactional outbox + idempotência** → arquiteturas compatíveis; a integração é
  de **tradução de contrato**, não de infraestrutura.

---

## 2. Estado atual — os dois contratos

### 2.1 Borda que o **Core** espera (fonte: `vendax.ai/core`)

| Direção | Subject NATS | Envelope (record Java) | Campos |
|---|---|---|---|
| **Inbound** Linktor→Core | `tenant.{id}.linktor.inbound` | `LinktorEnvelope` | `tenantId, vendorId, customerId, channel, messageType, content, idempotencyKey` |
| **Outbound** Core→Linktor | `tenant.{id}.core.outbound` | `LinktorOutbound` | idem + `conversationId` |
| **Channel config** Core→Linktor | `tenant.{id}.core.channel.config` | `ChannelConfigChanged` | `tenantId, version, channels[]` |

`channels[]` = `Channel{id, type, identifier, displayName, status, settings}` — só config **declarativa
e não-secreta** (ADR-012); credenciais NÃO trafegam aqui.

Regras do Core:
- Consome o wildcard `tenant.*.linktor.inbound` (`CoreNatsSubscriptions`), resolve a conversa por
  **`(vendorId, customerId, channel)`**, deduplica por `(tenant, conversation, idempotencyKey)`,
  persiste a mensagem e aciona o Nexus/ArchFlow. **O Core NÃO descobre o vendedor pelo número — ele
  confia no `vendorId` que o Linktor enviar.**
- Publica outbound/channel.config via **outbox at-least-once** (dedup por `idempotencyKey`).
- Empurra ao app do vendedor por WebSocket `/v1/stream` (indexado por `tenant:vendor`) — é por aqui que
  a IA **assiste/sugere ao vendedor, dentro do VendaX**. Isso **não** passa pelo canal nem pelo Linktor;
  quando o vendedor aceita uma sugestão, o Core a envia ao cliente como mensagem **do vendedor**.

### 2.2 Como o **Linktor** funciona hoje (fonte: `desenvolvimento/linktor`)

- **Subjects próprios** (`internal/infrastructure/nats/subjects.go`), convenção
  `linktor.<domínio>.<...>.<discriminador>`:
  - `linktor.messages.inbound.{channel_type}` / `...outbound.{channel_type}` (discriminador = **tipo de
    canal**, tenant vai no _payload_).
  - `linktor.events.{event_type}` (eventos de domínio: `message.received`, `conversation.created`,
    `contact.created`, `channel.*`… — tenant no _payload_).
  - `linktor.webhooks.{tenant_id}`, `linktor.bot.*.{tenant_id}`, `linktor.whatsapp.*.{tenant_id}`
    (tenant **no subject**).
- **Outbox transacional** (`outbox/relay.go`, `CreateWithOutboxEvent`): grava agregado+evento na mesma
  tx pgx; relay publica `linktor.events.{type}` com envelope `{type, tenant_id, payload, timestamp}`;
  dedup por `IdempotencyKey` via JetStream MsgID.
- **Inbound**: adapter do canal → `linktor.messages.inbound.{ch}` → `ReceiveMessageUseCase` →
  get-or-create de Contact/Conversation/Message → evento `message.received` (payload rico:
  `message_id, conversation_id, contact_id, channel_id, channel_type, content_type, content,
  external_id, sender_id, sender_name, attachments[]`). Há menção a um envelope **`linktor-channel-v1`**
  montado pelo dispatcher de webhook.
- **Outbound**: REST `POST /api/v1/conversations/:id/messages` → `SendMessageUseCase` (valida
  tenant/canal/destinatário) → publica `linktor.messages.outbound.{ch}` → consumer `outbound-{ch}` →
  adapter entrega. Seleção do adapter é **por subject** (channel_type).
- **Modelo**: `Conversation{ID, TenantID, ContactID, ChannelID, AssignedUserID *string, Status, …}` —
  **um único responsável**, **sem `vendorId`/participantes/copilot**. `Contact`+`ContactIdentity`
  chaveados por `(tenant, channel_type, identifier)`. `Channel` rico, credenciais criptografadas em
  repouso, provisionado por `connect`/`pair` (QR/pairing WhatsApp).
- **Multi-tenant**: `TenantID` em todas as raízes; tenant vem do JWT/API-key no runtime.

---

## 3. Princípios de arquitetura (fronteiras a preservar)

1. **O Core não fala com canais; o Linktor é o canal.** A tradução mora no Linktor (um adapter
   "VendaX"), não no Core. O Core só conhece seus próprios subjects/envelopes.
2. **No Linktor a conversa é vendedor ↔ cliente (dois).** A IA do VendaX não é participante do canal —
   ela assiste/sugere ao vendedor apenas no VendaX (app + WebSocket `/v1/stream`, já implementado). O
   Linktor **não** precisa de participantes múltiplos nem modelo de copilot; o `AssignedUserID` atual
   já modela o par.
3. **`vendorId` = o usuário atribuído no Linktor.** O vendedor do VendaX é o `AssignedUserID` da
   conversa no Linktor — o bridge mapeia `vendorId (Core) ↔ userId (Linktor)`. A _atribuição_ do
   vendedor à conversa é a decisão aberta #3 (por canal, por assignment, ou por carteira/CRM).
4. **At-least-once nos dois sentidos.** Ambos os lados entregam com outbox at-least-once → **idempotência
   obrigatória**; `idempotencyKey` estável e derivável deterministicamente em cada direção.

---

## 4. Mapeamento de identidade (o coração do plano)

| Conceito no Core | Origem no Linktor | Resolução proposta |
|---|---|---|
| `tenantId` | `TenantID` | **Alinhar o mesmo id nos dois sistemas** (decisão aberta #2). Se divergirem, tabela de mapeamento no bridge. |
| `vendorId` | `Conversation.AssignedUserID` (o vendedor **é** o usuário atribuído) | Mapear `vendorId (Core) ↔ userId (Linktor)`. Como o vendedor é atribuído à conversa: decisão aberta #3. |
| `customerId` | `Contact`/`ContactIdentity.identifier` (telefone) | Enviar o **telefone/identifier** como `customerId`; o Core faz auto-link telefone→cliente ERP (CC-09). |
| `channel` (string) | `Channel.id` + `channel_type` | `channel.config.channels[].id` = a instância do Linktor. `type` do Core (`WHATSAPP/…`) mapeia para o `channel_type` do Linktor (ver risco §9 — `WHATSAPP` → `whatsapp_official` **ou** `whatsapp_unofficial`; `MESSENGER` → `facebook`). |
| `conversationId` | `Conversation.ID` | Core resolve por `(vendor,customer,channel)` e devolve **seu** id no outbound. O bridge **re-resolve a conversa do Linktor por `(customerId→contact, channel)`** no outbound (mesma chave do Linktor) — evita tabela de correlação. (Alternativa: manter tabela `core_conv ↔ linktor_conv` — decisão aberta #5.) |
| `messageType`/`content` | `content_type`/`content` | Normalizar para os tipos ADR-010 (texto no MVP; áudio/quote/rich objects em L3). |
| `idempotencyKey` | `message.ID`/`external_id` | Inbound: derivar de `message.ID` do Linktor. Outbound: usar o `idempotencyKey` do Core como MsgID ao enviar. |

---

## 5. Fluxos

### 5.1 Inbound — cliente → Core → app do vendedor
```
WhatsApp/Telegram → adapter Linktor → linktor.messages.inbound.{ch}
   → ReceiveMessageUseCase (Contact/Conversation/Message + evento message.received)
   → [BRIDGE VendaX] consome message.received, resolve vendorId (config do canal),
     monta LinktorEnvelope, publica tenant.{id}.linktor.inbound
   → Core InboundMessageService (resolve conversa, dedup, Nexus→ArchFlow)
   → app do vendedor recebe via WebSocket /v1/stream
```

### 5.2 Outbound — vendedor/IA → cliente
```
Core (resposta do vendedor ou rich object da IA) → outbox → tenant.{id}.core.outbound (LinktorOutbound)
   → [BRIDGE VendaX] consome, resolve a conversa do Linktor por (customerId→contact, channel)
   → chama SendMessageUseCase (validações de tenant/canal/destinatário)
   → linktor.messages.outbound.{ch} → adapter entrega no canal
```

### 5.3 Channel config — Admin VendaX → Linktor
```
Admin VendaX (CA-06) publica → outbox → tenant.{id}.core.channel.config (ChannelConfigChanged)
   → [BRIDGE VendaX] consome, materializa/atualiza os Channels do tenant no Linktor
     (id, type, identifier, displayName, status). Credenciais e pairing (QR/pair) continuam no Linktor.
```
> **Autoridade dos canais** é a decisão aberta #1: o Admin VendaX declara a _intenção_ (quais
> instâncias/números existem e a qual vendedor pertencem) e o Linktor materializa + faz o pairing e
> guarda as credenciais. O `settings.vendorId` é o elo que injeta o vendedor no inbound.

---

## 6. Onde mora o bridge

- **Recomendado: dentro do Linktor**, como módulo `internal/integration/vendax/` (ou
  `internal/adapters/vendax/`), **opt-in por flag** (`LINKTOR_VENDAX_BRIDGE_ENABLED`). Precisa dos
  usecases (para reusar as validações de tenant/canal do `SendMessageUseCase`) e do provisionamento de
  canais. Assina os eventos internos (`message.received`) e os subjects do Core (`core.outbound`,
  `core.channel.config`); publica `linktor.inbound`.
- **Alternativa: processo ponte separado** (consome REST/NATS dos dois lados). Mais desacoplado, porém
  duplica validações e re-expõe credenciais/tenant. **Rejeitada para o MVP.**

---

## 7. Faseamento com gates (espelha o estilo OPENSPEC/CC-*/CV-* do VendaX)

| Fase | Escopo | Gate (DoD verificável) |
|---|---|---|
| **L0 · De-risk (spike)** | 1 mensagem de **texto** ponta-a-ponta nas duas direções, 1 canal, `vendorId` fixo por config. O bridge traduz `message.received`→`linktor.inbound` e `core.outbound`→`SendMessageUseCase`. | Mensagem real do canal aparece no app do vendedor (via Core); resposta do vendedor chega ao canal. Replay não duplica (idempotência nas duas pontas). Espelha o CC-00 do Core. |
| **L1 · Channel config** ✅ | Consumir `core.channel.config` → aplicar status/`vendorId` aos Channels existentes (Linktor é dono; não cria). `vendax_vendor_id` vira fallback do vendorId no inbound. | **Código feito (2026-07-08):** `channelconfig.go`, idempotente por versão, resolve canal por identifier, mapeia type Core→Linktor. Gate ponta-a-ponta pendente (mesmo ambiente do L0). |
| **L2 · Identidade & robustez** ✅ | Vocabulário de canal canônico nas duas pontas, idempotência do outbound (dedup por idempotencyKey), isolamento de tenant no inbound, correlação de conversa por (customer, channel) iterando subtipos. | **Código feito (2026-07-08):** `channeltypes.go` (coreChannelType↔linktorChannelTypes), `dedupe.go`, defense-in-depth de tenant. 8 testes verdes. Gate ponta-a-ponta pendente. |
| **L3 · Tipos & mídia** ✅ (parcial) | Mapeamento de content-type bidirecional; mídia (áudio/imagem/vídeo/documento) inbound (URL do anexo) e outbound (via anexo). Rich objects pulados com aviso (rendering ao canal = produto). | **Código feito (2026-07-08):** `messagetypes.go`, mídia no envelope. **Pendências:** rendering de rich objects ao canal (produto), transcrição de áudio (caminho HTTP do Core). |
| **L4 · Multi-vendedor / roteamento** _(se necessário)_ | Canal compartilhado por vários vendedores; roteamento (território/rodízio) resolve o `vendorId`. | Regra de roteamento resolve o vendedor correto por conversa. |

---

## 8. Decisões (fechadas com Edson, 2026-07-08)

1. **Autoridade dos canais:** o **Linktor continua dono** do canal (provisiona, pareia, guarda as
   credenciais). O CA-06 do VendaX é a camada **declarativa** que aponta para instâncias já existentes
   no Linktor e associa o vendedor. O fluxo §5.3 é, portanto, "materializar/atualizar apontando para o
   que o Linktor já tem", não criar do zero.
2. **`tenantId`:** **mesmo identificador nos dois sistemas** (ambos já multi-tenant). Sem tabela de
   mapeamento de tenant.
3. **Vendedor↔conversa:** o **vendedor é o usuário do Linktor** (`AssignedUserID`); `vendorId (Core) =
   userId (Linktor)`. MVP com atribuição **por canal** (1 número = 1 vendedor). Assignment do
   Linktor / carteira do CRM ficam para L4.
4. **Local do bridge:** **dentro do Linktor** — módulo `internal/integration/vendax/`, opt-in por flag
   `LINKTOR_VENDAX_BRIDGE_ENABLED`.
5. **Correlação de conversa:** **sem estado** — resolver por `(customerId→contato, channel)` no outbound
   (mesma chave que o Linktor já usa). Sem tabela `core_conv ↔ linktor_conv`.

---

## 9. Riscos e pré-requisitos

- **P0s de segurança do Linktor** (baseline §9): webhooks _fail-open_ (SendGrid/voz/RCS) e
  _defense-in-depth_ de tenant nos repositórios seguem abertos. São **higiene do Linktor**, não da
  integração, mas devem ser fechados antes de produção multi-tenant.
- **Mapeamento `channel_type`**: o Core usa `WHATSAPP/TELEGRAM/INSTAGRAM/MESSENGER/SMS`; o Linktor
  distingue `whatsapp_official` vs `whatsapp_unofficial` (whatsmeow) vs Coexistence, e usa `facebook`
  para Messenger. O `type` do CA-06 precisa de um sub-tipo (em `settings`) ou de uma tabela de-para.
- **Ordenação best-effort** dos dois outboxes → nunca assumir ordem; idempotência é a rede de proteção.
- **Allowlist de canais** (`LINKTOR_ENABLED_CHANNEL_TYPES`): a exposição real de canais em produção é
  limitada por ela; alinhar com os canais que o VendaX vai usar (começar por WhatsApp).
- **Revisão do baseline**: a estimativa de 4–6 semanas para "conversa a três + participantes múltiplos"
  **no Linktor** partiu de um pressuposto incorreto — que a IA seria participante do canal. Ela não é:
  no Linktor a conversa é vendedor↔cliente e a IA vive só no VendaX. O `AssignedUserID` atual já modela
  o par; o gap de domínio do Linktor é ~zero. Estimativa revisada abaixo.

## 10. Estimativa revisada (bridge fino, sem reforma de domínio)

- **L0 (de-risk)**: ~3–5 dias.
- **L1 (channel.config)**: ~3–5 dias.
- **L2 (identidade & robustez)**: ~1–1.5 semana.
- **L3 (rich objects/tipos)**: ~1–1.5 semana.
- **L4 (multi-vendedor)**: sob demanda.
- **Total até "loop completo texto+rich objects"**: **~3–4 semanas-pessoa** — versus 10–14 do baseline,
  porque não há reforma de domínio: a IA não é participante do canal e o `AssignedUserID` já modela o par
  vendedor↔cliente.

---

## 11. L0 — Spike de-risk detalhado (o próximo passo)

**Objetivo:** provar o loop Linktor↔Core por NATS com o bridge, para **1 canal (WhatsApp), 1 tenant, 1
vendedor fixo, só texto**. Espelha o CC-00 do Core. Nada de rich objects, áudio ou multi-vendedor ainda.

### 11.1 Pré-requisito de infra — alinhar o NATS
Os dois lados precisam falar o **mesmo servidor NATS JetStream** (o Core usa o da VPS devops, `:14222`).
Verificar/garantir:
- Linktor aponta para o mesmo NATS do Core (URL/creds).
- Existe stream que captura `tenant.*.core.outbound` (o bridge cria um **consumer durável** nele) e que
  o `tenant.*.linktor.inbound` publicado pelo bridge é capturado pela subscription do Core
  (`CoreNatsSubscriptions`). Confirmar se o inbound do Core é JetStream ou core NATS pub/sub e publicar
  no modo compatível.

### 11.2 Componentes a construir — `internal/integration/vendax/` (Go, opt-in por flag)
- **`config.go`** — lê `LINKTOR_VENDAX_BRIDGE_ENABLED` e os helpers de subject do Core
  (`tenant.{id}.linktor.inbound`, `tenant.{id}.core.outbound`). `tenantId` é o mesmo dos dois lados.
- **`inbound.go`** — assina o evento interno `linktor.events.message.received`, filtra INBOUND
  (`sender_type=contact`), e para cada um monta e publica o `LinktorEnvelope` em
  `tenant.{id}.linktor.inbound`:

  | Campo do envelope (Core) | Origem no Linktor |
  |---|---|
  | `tenantId` | `event.tenant_id` |
  | `vendorId` | `Conversation.AssignedUserID` (o vendedor atribuído) |
  | `customerId` | `ContactIdentity.identifier` (telefone) — o Core faz auto-link (CC-09) |
  | `channel` | `channel_type`/`channel_id` do evento |
  | `messageType` | `"text"` (L0) |
  | `content` | texto da mensagem |
  | `idempotencyKey` | `message.ID` do Linktor (estável) |

- **`outbound.go`** — consumer durável de `tenant.{id}.core.outbound`; parse `LinktorOutbound`; resolve
  a conversa do Linktor por `(customerId→ContactIdentity, channel)`; chama **`SendMessageUseCase`**
  (reusa validações de tenant/canal/destinatário) usando o `idempotencyKey` do Core como dedup MsgID.
- **Wiring** em `cmd/server/main.go` — registrar as duas subscriptions **apenas se a flag estiver
  ligada** (não altera o comportamento atual quando desligada).

### 11.3 Gate (DoD verificável)
1. **Inbound:** mensagem de texto que chega a um canal (real ou simulada por webhook de teste) **aparece
   no app do vendedor no VendaX** (via Core `/v1/stream`), com o `vendorId` correto.
2. **Outbound:** o vendedor responde no app VendaX → a resposta **chega ao cliente no canal**.
3. **Idempotência:** replay/duplicata **não gera mensagem dupla** em nenhuma ponta (inbound dedup por
   `message.ID`; outbound dedup por `idempotencyKey` do Core).
4. **Isolamento:** com a flag desligada, o Linktor se comporta exatamente como hoje.

### 11.4 Fora do escopo do L0 (fases seguintes)
channel.config (L1), correlação/robustez e mapeamento fino de `channel_type` (L2), rich objects/áudio
(L3), multi-vendedor (L4). No L0 o `channel_type` é fixo (WhatsApp) e o vendedor é fixo/por canal.

---

### Referências de código

- **Core**: `LinktorEnvelope`, `LinktorOutbound` (`infrastructure/messaging/dto/`); `CoreSubjects`;
  `CoreNatsSubscriptions`; `InboundMessageService`; `OutboundService`; `ChannelConfigService`
  (`application/channel/`, `record ChannelConfigChanged`); `OutboxEntity`/`OutboxPublisher`;
  WebSocket `/v1/stream` (`WebSocketConfig`, `AppWebSocketHandler`, `AppEventBroadcaster`).
- **Linktor**: `internal/infrastructure/nats/subjects.go` e `producer.go`; `outbox/relay.go`,
  `database/message_repo.go` (`CreateWithOutboxEvent`); `application/usecase/receive_message.go`,
  `send_message.go`; `domain/entity/{conversation,message,contact,channel}.go`;
  `database/channel_secrets.go` (criptografia de credenciais); `api/handlers/websocket.go`;
  `mcp/linktor-mcp-server/`.
