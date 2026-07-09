# VendaX Core bridge (L0)

Bridge que integra o **Linktor** ao **VendaX Core** por NATS. Traduz entre os vocabulários de
subjects/envelopes dos dois lados. Habilitado por `LINKTOR_VENDAX_BRIDGE_ENABLED=true`.

Contexto e decisões: `docs/vendax-integration/PLANO-integracao-linktor-vendax.md`.

## Fronteira

No Linktor a conversa é **vendedor ↔ cliente**. O Agente de IA do VendaX **não** é participante do
canal — ele assiste/sugere ao vendedor apenas dentro do VendaX; o que chega ao cliente é sempre uma
mensagem do vendedor. Por isso o **vendedor é o usuário atribuído à conversa** (`AssignedUserID`) e
mapeia para o `vendorId` do Core, sem reforma de domínio.

## O que o L0 faz

- **Inbound (Linktor → Core):** consome o evento interno `linktor.events.message.received` (consumer
  JetStream durável `vendax-bridge-inbound` no stream `LINKTOR_EVENTS`), monta o `LinktorEnvelope`
  (vendorId = usuário atribuído; customerId = identifier do contato; idempotencyKey = `message.ID`) e
  publica em `tenant.{id}.linktor.inbound` via **NATS core publish** (o Core consome com core sub).
- **Outbound (Core → Linktor):** assina `tenant.*.core.outbound` via **NATS core subscribe**, resolve
  a conversa do Linktor por `(customerId, channel)` (sem estado) e entrega via `SendMessageUseCase`.
- **Channel config (Core → Linktor, L1):** assina `tenant.*.core.channel.config` (o Admin VendaX /
  CA-06 publica), resolve o canal existente no Linktor por `(type, identifier)` e aplica os metadados
  declarativos que o Core é autoridade: `status` → `Channel.Enabled`, e `settings.vendorId` →
  `Channel.Config["vendax_vendor_id"]` (o vendedor-dono do canal). **O Linktor continua dono** do
  canal — se a instância declarada não existe, apenas loga um aviso (não cria; pairing/credenciais
  são do Linktor). Idempotente por versão (só aplica versões crescentes).

O `vendax_vendor_id` do canal serve de **fallback do vendorId no inbound**: quando a conversa ainda
não tem vendedor atribuído, o bridge usa o vendedor-dono do canal (fecha o "1 canal = 1 vendedor").

## Robustez (L2)

- **Vocabulário de canal canônico:** o `channel` do envelope trocado com o Core usa sempre os tipos
  canônicos (`WHATSAPP`, `TELEGRAM`, `MESSENGER`, …); os subtipos do Linktor (`whatsapp_official`/
  `unofficial`/`whatsapp`) são detalhe interno. `coreChannelType` normaliza no inbound;
  `linktorChannelTypes` reexpande no outbound e no channel.config (ver `channeltypes.go`).
- **Idempotência do outbound:** o Core entrega o outbound at-least-once; um retry do outbox reemite a
  mesma `idempotencyKey`. O bridge deduplica por `idempotencyKey` (FIFO com limite, `dedupe.go`) para
  não entregar a mensagem 2× ao cliente. O inbound já é idempotente no Core (dedup por `message.ID`).
- **Isolamento de tenant:** o inbound recusa uma conversa cujo `TenantID` não bate com o do evento
  (defense-in-depth, além da validação de ownership do `SendMessageUseCase`).

## Tipos de mensagem (L3)

- **Mapeamento de content-type bidirecional** (`messagetypes.go`): inbound normaliza o `content_type`
  do Linktor → `messageType` do Core (ADR-010); outbound reexpande para o `ContentType` do Linktor.
- **Mídia (áudio/imagem/vídeo/documento):**
  - *Inbound:* mídia sem legenda vira `messageType` canônico + a **URL do anexo** no `content` (a
    mensagem não se perde). **A transcrição de áudio é caminho separado no Core** (HTTP multipart
    `POST /v1/messages/audio`), não o inbound NATS — integrá-la exigiria o bridge baixar o áudio e
    postar no Core; fica como evolução.
  - *Outbound:* mídia do Core é entregue como **anexo** (`SendMessageInput.Attachments`), não texto.
- **Rich objects** (quote/suggestion/boleto/tracking/credit) **não são entregues ao canal** — o
  outbound os **pula com aviso** (`deliverableToChannel`) em vez de vazar JSON cru ao cliente. Como
  renderizá-los no canal (texto formatado? template do WhatsApp?) é **decisão de produto pendente**.

Escopo até L3: **texto + mídia, canais canônicos, idempotente nas duas pontas.** Falta: rendering de
rich objects ao canal (produto), transcrição de áudio via Core, e multi-vendedor/roteamento (L4).

## Pré-requisito de infra

Linktor e Core precisam falar o **mesmo servidor NATS**. O Core usa NATS core pub/sub (plain); o
Linktor usa JetStream — os dois interoperam no mesmo servidor. No spike, aponte ambos para o mesmo
NATS (ex.: o `nats:2.10 -js` do `docker-compose.yml` do Linktor, em `:4222`, com o Core configurado
para essa URL).

> Durabilidade: o Core entrega o outbound via NATS core (at-most-once). Se o bridge estiver offline no
> instante da publicação, aquela mensagem se perde. Aceitável no L0; evoluir para um stream JetStream
> durável capturando `tenant.*.core.outbound` antes de produção.

## Gate (DoD) — teste de integração ponta-a-ponta

Com Core + Linktor no mesmo NATS, um canal ativo e um vendedor atribuído à conversa:

1. **Inbound:** uma mensagem de texto que chega ao canal aparece no app do vendedor (via Core), com o
   `vendorId` correto.
2. **Outbound:** o vendedor responde no app VendaX → a resposta chega ao cliente no canal.
3. **Idempotência:** replay não gera mensagem duplicada em nenhuma ponta.
4. **Isolamento:** com `LINKTOR_VENDAX_BRIDGE_ENABLED` ausente/false, o Linktor se comporta como hoje.

## Testes unitários

`translation_test.go` cobre o coração da tradução (mapeamento de campos inbound/outbound e o
unmarshal do evento). Rodar: `go test ./internal/integration/vendax/`.
