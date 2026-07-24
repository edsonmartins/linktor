# Proposta de emenda ao CONSTITUTION — INV-025

- **Data:** 2026-07-23
- **Origem:** fase 0.2 do canal de teste (superfície de operação: console + observabilidade).
- **Status desta proposta:** proposta para revisão humana. **Não editei o `CONSTITUTION.md`** — o arquivo é mantido por decisão do arquiteto (G-002). Este documento é o insumo dessa decisão.
- **Depende de:** INV-018 (marcação propagada até a persistência e contratos) e INV-014 (degradação/estado não pode ser inferido no cliente).

## Lacuna que esta fase revelou

INV-018 garante que a origem sintética é **marcada e propagada até a persistência e os contratos de saída** (`conversations.environment`, envelopes NATS, `linktor-channel-v1`, `LinktorEnvelope`). Mas ele **termina na persistência**. Nada, hoje, obriga essa marcação a **alcançar a superfície de apresentação** onde um humano lê a conversa.

Antes desta fase, o dado sintético era identificável por query e invisível ao olho: cumpria a letra de INV-018 e não o propósito. O modo de falha correspondente é o **inverso** do que INV-017 fecha — aquele impede tráfego sintético de alcançar pessoa real; este impede **dado sintético de ser lido como real por um operador**. Um invariante que garante a marcação no banco mas não na tela deixa esse modo de falha aberto.

## Texto proposto

> ### INV-025 — A marcação de ambiente alcança o operador
> `IMPOSTO`
>
> Toda superfície onde uma conversa ou mensagem seja legível por humano apresenta a origem sintética (`environment = sandbox`) de forma visível. A distinção **não pode depender exclusivamente de cor**: há rótulo textual, e a informação é disponível a leitor de tela. A marcação é **derivada do backend** (o campo `environment` da conversa/canal), nunca inferida no cliente (INV-014).
>
> **Pontos de apresentação cobertos nesta fase:** listagem e detalhe de canal; listagem, detalhe e sessão aberta de conversa (banner persistente e não dispensável na conversa sandbox); filtro de ambiente na listagem de conversas, aplicado no backend.
>
> **Regra derivada:** toda superfície nova de leitura de conversa ou mensagem por humano nasce coberta — apresentar a marcação é requisito de aceite da superfície, não item posterior. Uma superfície que exiba conversa sem poder distinguir sintético de real não está pronta.
>
> **Correlato de diagnóstico (INV-014):** quando um envio sandbox é bloqueado, a superfície distingue bloqueio local (guarda) de rejeição de provedor, com o motivo derivado do backend (`metadata.blocked_by`) — nunca por inferência sobre o texto do erro. Motivo desconhecido é apresentado bruto, não adivinhado.

## Justificativa do status `IMPOSTO` (honesta, conforme G-002)

G-002 exige, para `IMPOSTO`, **ponto de imposição identificado e teste correspondente**. Esta fase entregou ambos:

- **Ponto de apresentação:** `EnvironmentBadge` / `SandboxConversationBanner` (`web/admin/src/components/environment-badge.tsx`), ligados em canal (`channels/page.tsx`), conversa (lista e `chat-view.tsx`) e filtro backend (`applyConversationFilters`). Diagnóstico em `message-failure-detail.tsx` a partir de `metadata.blocked_by`.
- **Teste correspondente:** specs Playwright `sandbox-environment.spec.ts` (rótulo textual via `role`, filtro chegando ao backend, produção inalterada) e `message-failure-detail.spec.ts` (bloqueio local vs. provedor, motivo desconhecido bruto).

**Ressalva de honestidade (a decidir na revisão):** `IMPOSTO` aqui vale para a **superfície entregue** (console admin web). A regra derivada ("superfície nova nasce coberta") é uma **restrição sobre mudanças futuras** que só um humano no processo de revisão consegue impor — não há barreira automática que impeça alguém de criar uma nova tela de conversa sem o badge. Se o revisor considerar que a ausência dessa barreira automática torna o invariante `PARCIAL` em vez de `IMPOSTO`, a classificação honesta é **`PARCIAL` — imposto na superfície atual, por convenção de revisão nas futuras**, com item de backlog para um teste que varra superfícies de conversa exigindo a marcação. Recomendo `IMPOSTO` para a superfície atual com a regra derivada registrada como convenção de revisão; deixo a escolha ao arquiteto.

## Efeito sobre o backlog derivado

Acrescentar ao backlog do CONSTITUTION (se `PARCIAL` for a escolha) um item: "Teste de cobertura que enumere superfícies de leitura de conversa e exija a marcação de ambiente" — prioridade Média.

## Fora do escopo desta proposta

Não propõe alteração a INV-014, INV-017 ou INV-018; INV-025 é aditivo e depende deles. Não cobre superfícies fora do console admin (ex.: eventual app do vendedor no VendaX) — essas herdam a marcação pelo contrato (INV-018) mas sua apresentação é responsabilidade do consumidor, o que deve ser dito explicitamente se/quando o contrato for consumido por UI de terceiro.
