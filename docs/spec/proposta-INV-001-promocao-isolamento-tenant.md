# Proposta de emenda ao CONSTITUTION — INV-001 (isolamento de tenant)

- **Data:** 2026-07-24
- **Origem:** workstream de isolamento de tenant (branch `fix/tenant-isolation-idor`).
- **Status:** proposta para revisão humana. **Não editei o `CONSTITUTION.md`** (mantido pelo arquiteto, G-002). Este documento é o insumo da decisão. Nota: o `CONSTITUTION.md` vive na branch `feat/sandbox-channel-fase0` (PR #50, ainda não mesclado); esta proposta pressupõe que ele seja mesclado antes de ser aplicada.

## Contexto: reconciliação da auditoria 2026-07

INV-001 está hoje `PARCIAL` — **pré-condição de release** — porque a auditoria `docs/auditoria-homologacao-2026-07.md` classificou o IDOR de tenant como P0 aberto. Este workstream **reconciliou a auditoria contra o código atual** (três levantamentos independentes: canais/bots/usuários; knowledge/payments/carts/commerce; whatsapp-avançado/history) e encontrou uma realidade bem diferente da descrita:

**Os P0 de IDOR da auditoria já estavam remediados**, via o padrão `GetByTenantAndID` + família `*ForTenant` no service (erro sempre `NotFound` para não vazar existência), com testes cross-tenant. Estavam protegidos: canais, bots, usuários, refund de pagamento, calling/CTWA/analytics, history import (StartImport), knowledge bases + busca RAG, WhatsApp Flows. Commerce e carts nem têm rota HTTP.

**Sobrava um único IDOR ativo**, mais fragilidades — todos endereçados por este workstream:

| # | Item | Ação | Commit |
|---|---|---|---|
| 1 | `GET /channels/:id/payments/customer/:phone` vazava histórico financeiro por telefone entre tenants (`GetByCustomer` sem org) | Escopado por `organization_id` (padrão de orders/carts) + teste cross-org | `a3e1b9d` |
| 2 | bots/users protegidos só na camada de service (repos com `WHERE id` sem `tenant_id`) — frágil a um handler futuro | Defense-in-depth no SQL: `WHERE id AND tenant_id` em Update/Delete/UpdateStatus/Assign/Unassign; teste de integração cross-tenant | `38269aa` |
| 3 | history-import (progress/list/cancel), carts — repos tenant/org-unaware, hoje sem rota | history-import valida tenant; cart Update escopa por org; id-methods de cart documentados como "escopar antes de expor" | `bf140ba` |

## Texto proposto para INV-001

> ### INV-001 — Todo dado é escopado por tenant
> `IMPOSTO`
>
> Toda entidade persistida carrega `tenant_id`/`organization_id`, e todo acesso por id valida a posse pelo tenant do requisitante. **Ponto de imposição em duas camadas:**
>
> - **Service:** cada recurso expõe `GetByTenantAndID` + família `*ForTenant`, que compara `resource.TenantID != tenantID` e retorna `NotFound` (nunca revela existência cross-tenant). Todos os handlers que recebem `:id` chamam a variante `*ForTenant`.
> - **SQL (defense-in-depth, onde presente):** as mutações de bots e users escopam por `tenant_id` na cláusula `WHERE`, de modo que um caller que pule a checagem de service ainda não escreve entre tenants. Verificado por teste de integração cross-tenant contra banco real.
>
> **Regra derivada:** recurso novo com rota por `:id` nasce com validação de posse por tenant no service; recurso que exponha mutação destrutiva deve ter também o backstop no SQL. Handler que resolva credencial de canal por `:id` valida a posse do canal antes de usá-la (helper `channelBelongsToTenant`).
>
> **Testes de imposição:** cross-tenant em payments (refund e customer), analytics, history import, template, e defense-in-depth de bots/users no repositório.

## Ressalvas de honestidade (para a decisão de status)

G-002 exige, para `IMPOSTO`, ponto de imposição identificado **e** teste correspondente — ambos existem. Mas há assimetrias que o revisor deve pesar antes de mudar o status:

1. **Defense-in-depth é parcial.** O backstop no SQL foi aplicado a **bots e users** (os recursos que a auditoria destacou e que têm rota admin). Canais, knowledge, payments etc. continuam protegidos **só na camada de service** — robusto hoje (todos os handlers usam `*ForTenant`), mas sem backstop no SQL. Se o arquiteto exigir defense-in-depth uniforme para declarar `IMPOSTO`, então o status honesto é **`IMPOSTO` na camada de service (imposto e testado) com defense-in-depth SQL parcial** — um meio-termo que talvez mereça uma nota no próprio invariante, ou um item de backlog para estender o backstop aos demais repos.

2. **Latentes não expostos.** Cart repo (todos os métodos por id agora escopados por `organization_id`) e history-import (serviço desligado, mas com validação de tenant) foram endereçados; o commerce segue como código sem rota, já validando tenant em `createCatalogClient`. Nenhum deles tem barreira de rota, mas nenhum reabre IDOR ao ser religado.

3. **INV-001 ≠ todos os P0 da auditoria.** A auditoria 2026-07 tinha outros P0 **além** do IDOR: schema divergente (seção B), panics determinísticos (C), fail-open/RCE. **Este workstream fecha apenas o IDOR de tenant.** Promover INV-001 não significa que o release está apto — os demais bloqueadores da auditoria são invariantes/workstreams próprios e devem ser verificados separadamente. O texto de INV-001 diz "pré-condição de release" hoje; ao promover, convém deixar claro que a pré-condição de release **do isolamento** está satisfeita, não a de todos os bloqueadores.

## Recomendação

Promover INV-001 de `PARCIAL` para **`IMPOSTO`** para o **isolamento de tenant**, com o ponto de imposição na camada de service (backed por SQL em bots/users) e os testes listados. Registrar como item de backlog de prioridade Média: **estender o backstop SQL (`WHERE tenant_id`) aos demais repos com mutação por id** (channels, knowledge, templates), para uniformizar a defense-in-depth. Manter separada a verificação dos demais P0 da auditoria (schema, panics) antes de qualquer declaração de aptidão a release.

## Relação com a gestão de canais por API key (contexto do pedido)

Este fix é pré-requisito de segurança para expor gestão de canais a uma aplicação externa via API key (tenant-scoped): as rotas de canal (Get/Update/Delete/Connect/QR) já validavam tenant no service, e agora o eixo IDOR está confirmado fechado. Uma API key da org A não alcança recursos da org B. **Pendência ortogonal ainda aberta:** os *scopes* da API key não são aplicados (uma key tem acesso total às rotas não-admin do seu tenant) — se a app externa precisar de acesso restrito (só canais, por exemplo), isso é um trabalho à parte (enforcement de scopes), não coberto aqui.
