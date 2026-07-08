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

Escopo do L0: **só texto, 1 canal, 1 vendedor por canal.** channel.config (L1), robustez/identidade
(L2), rich objects/áudio (L3) e multi-vendedor (L4) são fases seguintes.

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
