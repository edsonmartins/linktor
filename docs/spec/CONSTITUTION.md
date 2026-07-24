# CONSTITUTION — Linktor

- **Componente:** Linktor — camada de mensageria multicanal (Go)
- **Versão:** 0.2
- **Data:** 2026-07-23
- **Status:** proposta para revisão humana
- **Base de evidência:** `feat/sandbox-channel-fase0` (9 commits, `324379f`..`1697a02`)

## Natureza deste documento

Este é um CONSTITUTION **descritivo**. Ele não inaugura decisões: extrai e formaliza os invariantes que o código do Linktor **já** sustenta, mais aqueles que a auditoria mostrou que precisam valer.

Consequências práticas:

- Invariante `IMPOSTO` é restrição sobre mudanças futuras. Alterá-lo exige ADR.
- Invariante `PARCIAL` ou `DECLARADO` é dívida reconhecida, com item de backlog. Não é permitido escrever código que **amplie** a lacuna (G-004).
- **Status é atualizado a cada leva de entrega.** Um invariante marcado como imposto quando não está é o primeiro passo para o documento virar ficção — que é exatamente o que ele existe para evitar.

### Histórico

| Versão | Data | Mudança |
|---|---|---|
| 0.1 | 2026-07-23 | Extração inicial a partir da análise `canal-de-teste-homologacao.md` |
| 0.2 | 2026-07-23 | Pós fase 0 do canal de teste: INV-016/017/018 promovidos a `IMPOSTO`; INV-015 promovido a `PARCIAL` (implementado, inativo); INV-023 rebaixado a `PARCIAL`; INV-024 e G-005 introduzidos |

## Escopo

O Linktor é responsável por: normalizar transportes heterogêneos num modelo canônico de mensagem; entregar mensagens de saída ao provider correto; ingerir mensagens de entrada com garantias de idempotência; e expor o resultado a consumidores (VendaX Sales Copilot, console, bots) por contrato estável.

Fora de escopo do componente: lógica de negócio conversacional, orquestração de agentes, decisão comercial.

---

## Legenda de status

| Status | Significado |
|---|---|
| `IMPOSTO` | Verificável no código, com ponto de imposição identificado e teste correspondente |
| `PARCIAL` | Imposto em parte do fluxo, imposto por convenção sem barreira estrutural, ou implementado mas não ativo |
| `DECLARADO` | Invariante desejado, **sem** imposição no código. Dívida com item de backlog |

---

## Seção I — Isolamento e soberania de dados

### INV-001 — Todo dado é escopado por tenant
`PARCIAL` — **pré-condição de release**

Toda entidade persistida carrega `tenant_id`, e todo acesso é filtrado por ele. Unicidade de identidade de contato é tenant-scoped (migration `00007`); limites operacionais são por tenant.

**Lacuna:** a auditoria de 2026-07 identificou IDOR de tenant classificado como P0. Enquanto não fechado, **nenhuma garantia derivada deste invariante é válida** — incluindo a fronteira de ambiente (INV-016) e a allowlist (INV-017), que herdam integralmente o isolamento de tenant.

**Precedente correto (fase 0):** acesso cross-tenant à allowlist é rejeitado como *not-found*, não como *forbidden* — não confirma existência de recurso alheio.

**Regra derivada:** nenhum endpoint novo recebe identificador de recurso sem validação de posse pelo tenant do requisitante.

### INV-002 — Credenciais são cifradas at-rest e nunca serializadas
`IMPOSTO`

Credenciais de canal são cifradas com AES-256-GCM (`pkg/crypto/crypto.go`) e marcadas `json:"-"` em `entity.Channel`.

**Limite conhecido:** a validação `credential_environment` introduzida na fase 0 é **declarativa** — não é possível inferir de um token se ele é de produção. A garantia dura contra alcance indevido é INV-017, não esta validação. Não registre INV-002 como cobrindo separação de ambiente de credencial.

**Regra derivada:** nenhum campo de credencial em resposta de API, log, métrica ou evento. Inclui número de telefone completo em mensagem de erro — mascare.

**Dívida:** `logActivity` do worker registra `meta["to"]` com número completo em falhas de provider (pré-existente; bloqueios de guarda já mascaram).

**Dívida:** não há cofre externo (OpenBao) integrado. Migrar é decisão de ADR futuro, não regressão.

### INV-003 — Dado de um tenant não trafega em contrato compartilhado sem escopo explícito
`IMPOSTO`

Subjects por tenant (`tenant.{id}.linktor.inbound`); bridge VendaX endereça por tenant e canal.

---

## Seção II — Identidade e modelo canônico

### INV-004 — Identidade de conversa não depende do transporte
`IMPOSTO`

`entity.Conversation` chaveada por UUID interno, correlacionada por `(tenant_id, channel_id, contact_id)`. Identificadores do provider vivem em `Message.ExternalID` e `ContactIdentity.Identifier`.

**Regra derivada:** nenhum adapter introduz chaveamento de conversa por identificador de sessão ou de transporte.

### INV-005 — Roteamento de envio é por instância de canal
`IMPOSTO`

Todo envio resolve por `channel_id` concreto (`Resolver.For`). Não existe — e não deve passar a existir — roteamento por tipo de canal, por tenant ou por qualquer atributo de política.

**Confirmado na fase 0:** `environment` foi implementado como atributo consultado nos pontos de imposição, **não** como participante da resolução de destino.

### INV-006 — Todo caminho de envio converge no funil único
`IMPOSTO`

API, campanha, bot e retries convergem em `outbound.Worker` → `Resolver.For` → `Sender.Send`.

**Regra derivada:** proibido criar caminho de envio que chame `Sender` ou adapter fora deste funil. Guarda de segurança de envio é implementada aqui, não em handler HTTP.

**Validado na fase 0:** a guarda de INV-017 tem cobertura de teste nos três caminhos (API, campanha, retry) precisamente por estar neste ponto.

### INV-007 — Contrato público de saída é versionado
`IMPOSTO`

Envelope de saída com versão explícita (`linktor-channel-v1`). Campos novos são aditivos com `omitempty`; a fase 0 verificou por teste que o envelope de produção não muda byte a byte.

**Dívida:** o modelo canônico interno usa `Content string` livre + `Metadata map[string]string`, sem payload tipado por `ContentType` nem versionamento de schema.

---

## Seção III — Garantias de entrega

### INV-008 — Ingestão é idempotente em três camadas
`IMPOSTO`

(1) Dedup em Redis por `eventID`, TTL 6h; (2) unique parcial `(conversation_id, external_id)` — camada autoritativa; (3) MsgID no JetStream, janela de 5 min.

**Regra derivada:** nenhum adapter entrega mensagem inbound sem `ExternalID` estável.

### INV-009 — Persistência de inbound e publicação de evento são atômicas
`IMPOSTO`

`MessageRepository.CreateWithOutboxEvent` grava mensagem e evento na mesma transação; `outbox.Relay` publica com idempotency key.

### INV-010 — Envio não perde mensagem por falha de processo
`DECLARADO` — **dívida ativa, sem mitigação implantada**

O fluxo de saída é dual-write: grava `pending` e publica no NATS em passos separados, sem outbox. Queda entre os dois deixa mensagem órfã sem relay de recuperação.

**Estado:** o job de reconciliação previsto na v0.1 **não** foi implementado na fase 0. A lacuna permanece aberta em produção.

### INV-011 — Falhas são classificadas como permanentes ou transitórias
`IMPOSTO`

4xx do provider → `failed` + ACK; 5xx e rate-limit → NAK → `MaxDeliver` → DLQ.

**Regra derivada:** todo `Sender` novo classifica erro explicitamente. Erro não classificado é permanente — falhar visível é preferível a reentrega infinita.

### INV-012 — Não há garantia de ordenação por conversa
`DECLARADO` — **limitação assumida**

Stream inbound é `WorkQueuePolicy`, sem particionamento por conversa. Mensagens de uma conversa podem ser processadas fora de ordem.

**Regra derivada:** consumidores não podem assumir ordenação, e a limitação é declarada no contrato de integração — não silenciada.

### INV-013 — Autenticidade de inbound é verificada por provider
`IMPOSTO`

Validação de assinatura antes de qualquer processamento: HMAC-SHA256 (WhatsApp), HMAC-SHA1 (Twilio), token (Telegram), HMAC compartilhado (webhook genérico).

---

## Seção IV — Capacidade e política de provider

### INV-014 — Degradação de capacidade não pode ser silenciosa
`DECLARADO`

`plugin.ChannelCapabilities` é estática, não exposta a consumidores; o envio faz fallback silencioso (interactive → texto) e `MaxMediaSize` não é aplicado.

**Regra derivada (a partir da adoção):** toda degradação registra evento observável e o conjunto de capacidades é consultável em runtime.

### INV-015 — Política de mensageria do provider é aplicada no Linktor
`PARCIAL` — **implementado, não ativo**

Janela de 24h e status de template passaram a ser avaliados no fluxo real (fase 0, WP6/WP7): a política de janela foi extraída do consumer órfão para unidade pura com fonte durável (`MessageRepository.LastInboundAt`); o status de template é consultado no envio, sem cache, porque a Meta recategoriza por conta própria.

**Por que `PARCIAL` e não `IMPOSTO`:** o default de `LINKTOR_WA_WINDOW_ENFORCEMENT` e `LINKTOR_WA_TEMPLATE_ENFORCEMENT` é `dry_run`. A avaliação ocorre, o bloqueio não. Até a virada para `enforce`, a rejeição continua sendo terceirizada ao provider, com o custo correspondente em quality rating.

**Promoção a `IMPOSTO` exige:** critério de saída do dry-run definido e cumprido, e `enforce` ativo em produção.

### INV-024 — Política que falha aberta é contabilizada
`DECLARADO` — **introduzido na v0.2**

INV-015 falha **aberta** em incerteza (erro na consulta de `LastInboundAt`, status de template indisponível). A escolha é correta: é política de provider, não fronteira de segurança, e falso positivo bloquearia cliente real.

**Mas a falha aberta não é observável hoje.** A métrica existente conta bloqueios; não conta avaliações que não puderam ser feitas. Consequência: falha sistemática na consulta desliga o enforcement na prática, e o painel mostra queda de bloqueios — indistinguível de melhoria.

**Regra derivada:** todo ponto de decisão que falha aberto emite contador próprio, distinto do contador de bloqueios. Fail-open sem sinal é indistinguível de ausência de risco.

---

## Seção V — Ambiente e dado sintético

### INV-016 — Ambiente é atributo imutável da instância de canal
`IMPOSTO`

`ChannelEnvironment ∈ {sandbox, production}` em `entity.Channel`, default `production` (migration goose `00012`), persistido com round-trip verificado contra banco real.

Imutabilidade em **duas camadas**: o `UPDATE` do repositório omite a coluna deliberadamente (estrutural), e o service rejeita explicitamente update que altere o valor. Uma camada seria convenção; duas é estrutura.

**Regra derivada:** atributo com função de segurança recebe imposição estrutural além da validação de aplicação.

### INV-017 — Canal sandbox só entrega a destinatários em allowlist explícita
`IMPOSTO` — **núcleo de segurança da capacidade**

Decorator de `Sender` no funil único (INV-006), cobrindo API, campanha e retry. Allowlist tenant-scoped com opt-in por canal (migration `00013`), E.164 normalizado na escrita **e** na comparação.

**A allowlist é consultada a cada envio, nunca capturada na construção do sender.** Remoção de destinatário vale no envio seguinte, sem depender do TTL de 5 min do cache do `Resolver` — verificado por teste que não usa expiração.

**Fail-closed em toda incerteza:** checker ausente, erro de consulta, destinatário inválido, ou tipo de canal sandbox sem semântica definida. Bloqueio é erro permanente, sem retry, com registro em `message_logs` e métrica.

**Modo de falha inaceitável:** tráfego sintético alcançar destinatário real de cliente. Requisito de segurança, não validação de negócio.

### INV-018 — Origem sintética é marcada e propagada até a persistência
`IMPOSTO`

`environment` nos envelopes NATS, no contrato `linktor-channel-v1` (aditivo, `omitempty`) e no `LinktorEnvelope` do VendaX. Denormalizado em `conversations` no nascimento da conversa (migration `00014`), com round-trip contra banco.

**Dívida de propagação:** o campo ainda não existe no DTO Java do Core VendaX. O contrato está em PROPOSTA — enquanto não congelar, a inclusão é aditiva e barata. Depois do freeze, vira versionamento entre dois produtos.

### INV-019 — Dado de sandbox tem retenção própria e não é exportado
`DECLARADO`

Retenção curta com expurgo automático; bloqueado para pipelines analíticos e para o bridge VendaX.

**Habilitado, não implementado:** o dado já nasce marcado (INV-018), então o expurgo e o bloqueio de export são construíveis sem retrabalho. Como analytics lê tabelas operacionais diretamente, o bloqueio deve ser estrutural (views filtradas), não convenção de filtro em cada query.

**Pendente de decisão humana:** prazo de retenção.

### INV-020 — Promoção de sandbox para produção é troca de configuração
`PARCIAL`

Sustentado por construção: roteamento é por `channel_id` (INV-005), e o contrato é idêntico entre ambientes — o campo novo é aditivo com `omitempty` e o envelope de produção não mudou. Promover é apontar o consumidor para um `channel_id` de canal `production`.

**Por que `PARCIAL`:** não há teste que verifique a propriedade de ponta a ponta. Vale por consequência dos outros invariantes, não por verificação própria.

Histórico de sandbox não migra na promoção — ele é expurgado (INV-019).

---

## Seção VI — Observabilidade e auditoria

### INV-021 — Métricas não carregam label de tenant
`IMPOSTO` — decisão deliberada de cardinalidade

`linktor_outbound_guard_blocked_total{channel_type,reason,mode}` respeita a regra. Granularidade por tenant sai de `message_logs`, não de série temporal.

### INV-022 — Timeline de conversa é reconstruível
`PARCIAL`

Reconstruível pelos **dados** (Postgres). Não pelos logs: sem tracing distribuído (OTel ausente), sem correlation-id unificado, e o log de inbound não carrega `message_id`/`conversation_id`.

**Regra derivada:** enquanto a lacuna existir, funcionalidades que dependem de correlação por mensagem (ex.: shadow channel) permanecem inviáveis e não são prometidas.

### INV-023 — Ações administrativas são auditadas
`PARCIAL` — **rebaixado na v0.2**

A allowlist de sandbox é auditada (autor, ação, valor). **Criação e alteração de canal não são.**

A assimetria é relevante: `environment` é o atributo que define toda a fronteira de sandbox, e não há trilha sobre quem criou o canal nem com qual credencial declarada. A parte que mais precisa ser prestável a contas é justamente a que não tem registro.

---

## Seção VII — Governança

### G-001 — Hierarquia normativa
CONSTITUTION (invariantes) > ADR (decisão de arquitetura) > RFC (desenho técnico) > OpenSpec (unidade de execução).

### G-002 — Emenda
Alterar invariante `IMPOSTO` exige ADR aceito. Promover de `DECLARADO`/`PARCIAL` para `IMPOSTO` exige ponto de imposição identificado **e** teste correspondente. Implementado sem estar ativo é `PARCIAL`, não `IMPOSTO`.

### G-003 — Não retroatividade
Não se escreve ADR para decisão já em produção e não contestada. O registro dessas decisões é este documento.

### G-004 — Regra do não-agravamento
Código novo não pode ampliar a lacuna de um invariante `PARCIAL` ou `DECLARADO`. Pode conviver com ela; não pode aprofundá-la.

### G-005 — Campo declarado é campo persistido
`introduzido na v0.2`

Ocorreu três vezes o mesmo defeito: campo presente na entidade que o repositório não persiste — `source`/`is_imported` em `messages`, e `Channel.Identifier`/`MessageTemplateNamespace` em `channels`. Não são três defeitos, é um defeito de processo.

**Regra:** a cobertura contra esta classe é um teste reflexivo genérico de round-trip por entidade, não correção caso a caso. Nenhum campo novo com função de segurança entra sem round-trip verificado contra banco real (precedente: INV-016, INV-018).

### G-006 — Assimetria deliberada de modo de falha
`introduzido na v0.2`

Fronteira de segurança falha **fechada** (INV-017). Conformidade com política externa falha **aberta** (INV-015, INV-024). A assimetria é intencional e deve ser preservada: uma protege terceiro contra o sistema, a outra protege o cliente contra falso positivo do sistema.

**Regra:** toda guarda nova declara explicitamente seu modo de falha e a qual das duas categorias pertence.

---

## Backlog derivado

| # | Item | Invariante | Estado |
|---|---|---|---|
| 1 | Fechar IDOR de tenant (P0 da auditoria) | INV-001 | **Bloqueante, aberto** |
| 2 | Contador de fail-open das políticas de provider | INV-024 | **Alta, aberto** |
| 3 | Critério de saída do dry-run e virada para `enforce` | INV-015 | **Alta, aberto** |
| 4 | Auditoria de criação/alteração de canal | INV-023 | Alta, aberto |
| 5 | Teste reflexivo de round-trip por entidade | G-005 | Alta, aberto |
| 6 | Persistir `Channel.Identifier` e `MessageTemplateNamespace` | G-005 | Média, aberto |
| 7 | Mascarar número em `logActivity` de falha de provider | INV-002 | Média, aberto |
| 8 | Campo `environment` no DTO Java do Core VendaX | INV-018 | Média, com prazo (antes do freeze) |
| 9 | Job de reconciliação de `pending` órfão | INV-010 | Média, aberto |
| 10 | Expurgo de sandbox + views filtradas para analytics/bridge | INV-019 | Média, aberto |
| 11 | Enriquecer log de inbound com `message_id`/`conversation_id` | INV-022 | Média, aberto |
| 12 | Endpoint de descoberta de capabilities + evento de degradação | INV-014 | Média, aberto |
| 13 | Aplicar `MaxMediaSize` das capacidades | INV-014 | Média, aberto |
| 14 | Outbox transacional no envio (substitui o item 9) | INV-010 | Backlog |
| 15 | Particionamento por conversa ou declaração formal | INV-012 | Backlog |
| 16 | Payload tipado e versionado no modelo canônico | INV-007 | Backlog |
| 17 | Integração com cofre (OpenBao) | INV-002 | Backlog |

**Concluídos na fase 0:** `environment` no domínio e schema (INV-016), guarda de allowlist (INV-017), propagação da marcação (INV-018), avaliação de janela de 24h e status de template (INV-015, em dry-run), métrica de bloqueio (INV-021).

---

## Pendências de decisão humana

1. Prazo de retenção de dado sandbox (INV-019).
2. Critério objetivo de saída do dry-run: janela de observação, taxa esperada de bloqueio, e o que caracteriza anomalia que impede a virada (INV-015).
3. Se o espelho não oficial entra no release atrás de flag e termo de aceite.
4. Se a criação de canal passa a exigir aprovação de segundo administrador quando `environment = sandbox` com credencial declarada de produção.
