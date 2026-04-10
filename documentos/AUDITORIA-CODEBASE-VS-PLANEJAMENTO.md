# Auditoria do codebase versus planejamento

Data da auditoria: 2026-04-07

Escopo: comparação do codebase em `/Users/edsonmartins/desenvolvimento/linktor` com os documentos em `documentos/`, cobrindo spec geral, adapters, WhatsApp Cloud API, coexistência, VRE e MCP.

## Atualização pós-correção

Data da atualização: 2026-04-08

Os achados abaixo representam o estado auditado em 2026-04-07. As correções aplicadas em seguida endereçaram os principais pontos:

- `RunMigrations()` passou a criar o schema usado em runtime para coexistência, templates, observabilidade, pagamentos e histórico.
- Endpoints de teste de canais chamados pelo admin passaram a existir no backend, e o fluxo RCS do admin passou a usar a API real do backend.
- Clients avançados de WhatsApp passaram a ser registrados no startup e em eventos de lifecycle de canal conectado/atualizado/desconectado.
- Webhooks `message_echoes` passaram a atualizar `last_echo_at` para canais de coexistência.
- VRE foi alinhado para `/api/v1/vre/*`, preview via `GET`, default JPEG e data URL escapada.
- MCP HTTP passou a registrar/exportar as tools VRE.
- Documentação de coexistência foi corrigida para não prometer importação automática de 6 meses via Cloud API; o caminho documentado agora é importação manual por arquivo/export.

## Veredito executivo

O projeto está além do MVP descrito no `LINKTOR-PROJECT-SPEC.md`: há backend Go funcional, admin Next, WebChat, múltiplos adapters, bot/AI, knowledge base, flows conversacionais, SDKs em várias linguagens, MCP e VRE. A suíte Go passa quando executada com permissão para `httptest` abrir listeners locais.

O risco principal não é falta de código: é desalinhamento entre código, migrações e integração real. Várias funcionalidades avançadas existem como handlers/clients/repositórios, mas dependem de tabelas que não são criadas pelo caminho de startup (`cmd/server/main.go` -> `db.RunMigrations`). Isso afeta canais/coexistência/templates/observabilidade/pagamentos/histórico e pode quebrar rotas em runtime mesmo com testes unitários verdes.

## Achados críticos

### 1. Migrações externas não são executadas no startup

Planejamento:
- O spec exige schema/migrations no foundation e PostgreSQL persistente.
- Os documentos avançados adicionam tabelas para templates, observabilidade, commerce, payments, calling, CTWA e coexistência.

Código:
- `cmd/server/main.go:145` chama `db.RunMigrations(context.Background())`.
- `internal/infrastructure/database/postgres.go:65-87` executa apenas SQL embutido em Go, até `refactorChannelStatus`.
- `deploy/docker/migrations/005_ai_tables.sql` a `010_whatsapp_coexistence.sql` existem, mas não são carregados pelo servidor.

Impacto:
- `ChannelRepository` já usa colunas `is_coexistence`, `waba_id`, `last_echo_at`, `coexistence_status` em `INSERT` e `SELECT` (`internal/infrastructure/database/channel_repo.go:38-43`, `73-79`, `107-115`). Se o banco nasce só por `RunMigrations`, essas colunas não existem.
- `TemplateRepository` usa tabela `templates`, criada apenas em `deploy/docker/migrations/007_templates_table.sql`.
- `ObservabilityRepository` usa `message_logs`, criada apenas em `deploy/docker/migrations/006_observability_tables.sql`.
- `PaymentRepository` usa `whatsapp_payments`, criada apenas em `deploy/docker/migrations/009_whatsapp_advanced_tables.sql`.
- `HistoryImportRepository` usa `whatsapp_history_imports`, criada apenas em `deploy/docker/migrations/010_whatsapp_coexistence.sql`.

Severidade: crítica. Antes de expandir features, unificar o mecanismo de migração ou incorporar essas migrações no runner real.

### 2. Handlers avançados de WhatsApp são expostos, mas clientes não são registrados

Planejamento:
- O blueprint pede analytics nativos, payments, calling e CTWA em fases 5 e 6.

Código:
- O servidor expõe rotas para analytics WhatsApp, payments, calling e CTWA (`cmd/server/main.go:893-940`).
- Os handlers são criados (`cmd/server/main.go:490-499`), mas a conexão real por canal está apenas comentada (`cmd/server/main.go:504-507`).
- `paymentRepo` é inicializado mas descartado com `_ = paymentRepo` (`cmd/server/main.go:184`, `512`).

Impacto:
- As rotas existem, mas tendem a retornar erro de client não encontrado até que o fluxo de conexão de canal registre `analytics.Client`, `payments.Client`, `calling.Client` e `ctwa.Client`.

Severidade: alta.

### 3. Coexistência WhatsApp está parcialmente implementada, mas o histórico planejado não é viável pelo próprio código

Planejamento:
- O guia de coexistência prevê Embedded Signup, message echoes, importação de 6 meses de histórico, monitoramento de atividade e billing.

Código:
- Embedded Signup e rotas de coexistência existem (`cmd/server/main.go:797-803`, `786-808`).
- Message echoes são parseados no adapter oficial (`internal/adapters/whatsapp_official/webhook.go` tem suporte a `message_echoes`) e há UI para badges.
- Monitoramento de atividade existe (`internal/application/service/coexistence_monitor.go`).
- `HistoryImportService.importConversations` retorna erro explícito dizendo que a Cloud API não expõe exportação de histórico e recomenda import manual (`internal/application/service/history_import.go:220-224`).

Impacto:
- A fase “Chat History Import” do planejamento não está implementada no sentido prometido. O código corretamente reconhece a limitação da API, mas isso torna o plano original obsoleto.
- Não encontrei rotas HTTP registradas para iniciar/listar/cancelar o import de histórico.

Severidade: alta para alinhamento de produto/documentação.

### 4. Frontend chama endpoints de teste de canais que não existem no backend

Código frontend:
- `web/admin/src/app/(dashboard)/channels/whatsapp-config.tsx` chama `/channels/test-whatsapp`.
- `telegram-config.tsx` chama `/channels/test-telegram`.
- `sms-config.tsx` chama `/channels/test-twilio`.
- `facebook-config.tsx` chama `/channels/test-facebook`.
- `instagram-config.tsx` chama `/channels/test-instagram`.
- `voice-config.tsx` chama `/channels/test`.
- `rcs-config.tsx` chama rotas Next locais `/api/channels/rcs...`.

Código backend:
- As rotas reais de canais em `cmd/server/main.go` incluem CRUD, status, enabled, connect, pair, disconnect e coexistência, mas não os endpoints de teste citados.
- Não há rotas Next em `web/admin/src/app/api`.

Impacto:
- A UI aparenta suportar “test connection”, mas essas ações falharão.

Severidade: alta para admin UX.

### 5. VRE implementado, mas a API real diverge da documentação e há bug de formato WebP

Planejamento:
- `linktor-visual-response-engine.md` documenta `POST /api/v1/render`, `POST /api/v1/render-and-send`, `GET /api/v1/templates`, `POST /api/v1/templates` e preview.

Código:
- O servidor expõe VRE sob `/api/v1/vre/...` (`cmd/server/main.go:945-954`), não diretamente em `/api/v1/render`.
- O cliente MCP chama preview com `POST /vre/templates/:id/preview` (`mcp/linktor-mcp-server/src/api/client.ts:512-515`), mas o backend expõe `GET /vre/templates/:id/preview` (`cmd/server/main.go:949-950`).
- `renderer.go` declara WebP como default, mas no `case OutputFormatWebP` codifica JPEG como fallback (`internal/infrastructure/vre/renderer.go:208-218`). A resposta pode dizer `format=webp`/`data:image/webp`, mas os bytes são JPEG.
- `RenderHTML` monta data URL concatenando HTML bruto (`internal/infrastructure/vre/renderer.go:139-147`), o que pode quebrar com caracteres reservados; deveria usar URL encoding/base64.

Impacto:
- Integrações baseadas no documento ou no MCP tendem a quebrar.
- Clientes podem receber conteúdo com MIME/format inconsistente.

Severidade: alta.

### 6. MCP HTTP server não registra ferramentas VRE, embora o stdio server registre

Planejamento:
- O guia MCP prevê HTTP streamable com tools/resources/prompts, e o docs playground aponta para HTTP.

Código:
- `server.ts` inclui `registerVRETools` e `vreToolDefinitions` no servidor stdio (`mcp/linktor-mcp-server/src/server.ts:26`, `46-55`).
- `http-server.ts` só registra conversations/messages/contacts/channels/bots/analytics/knowledge; VRE está ausente dos imports, handlers e `allTools` (`mcp/linktor-mcp-server/src/http-server.ts:21-29`, `60-68`, `81-90`).

Impacto:
- O playground/documentação HTTP não verá nem executará ferramentas VRE, apesar de elas existirem para stdio.

Severidade: média-alta.

## Cobertura versus roadmap

### Spec geral / MVP

Status: majoritariamente implementado, com risco de migração.

Evidências:
- Docker Compose existe para PostgreSQL/Redis/NATS/MinIO.
- API e auth existem.
- Messaging/conversations/contacts/channels existem.
- WebChat existe com websocket.
- Admin panel existe com dashboard, conversas, canais, bots, knowledge, flows, analytics, observabilidade e usuários.
- Multi-tenant existe no modelo e middleware.

Lacunas:
- Não há `.github/workflows`, então CI/CD planejado não existe.
- O Docker Compose sobe infra, mas não backend/frontend.
- O caminho de migração real não aplica as migrações externas.

### Adapters de canais

Status: mais amplo do que o spec inicial.

Implementado no código:
- WebChat, WhatsApp Official, WhatsApp unofficial/whatsmeow, RCS, Telegram, SMS, Facebook, Instagram, Email.
- Há adapters de voice no código, mas eles não são registrados no registry do servidor principal.

Lacunas:
- O spec lista voice como adapter planejado, mas `cmd/server/main.go` não faz `plugin.Register` para `ChannelTypeVoice`.
- Testes de conexão no frontend não batem com rotas backend.

### WhatsApp Cloud API 100%

Status: cobertura ampla, mas não “100% integrada”.

Implementado:
- Webhook parser robusto, status, reações, media, location, contacts, interactive, templates, carousel, authentication templates, LTO/coupon, flows client/builder/encryption/data exchange, commerce/catalog/cart/order, analytics, payments, calling, CTWA.

Lacunas:
- Muitos módulos avançados não estão conectados ao ciclo de vida de canal.
- As tabelas necessárias não entram no runner real.
- Payment/calling/CTWA/analytics dependem de client registry ainda não implementado.
- Commerce tem managers em memória e handlers de order, mas integração de ponta a ponta com persistência/rotas completas ainda parece parcial.

### Coexistência

Status: parcial.

Implementado:
- Embedded signup backend e frontend.
- Campos de domínio para coexistência.
- Parser de message echoes.
- Badge/Widget de status no admin.
- Monitor de atividade.

Lacunas:
- Migração necessária não aplicada pelo startup.
- Histórico de 6 meses do planejamento foi contradito pelo próprio serviço.
- Billing App-free/API-paid aparece no documento, mas não encontrei implementação efetiva.
- Falta rota HTTP para histórico de importação.

### VRE

Status: funcional em núcleo, com divergências de contrato.

Implementado:
- Templates HTML padrão existem.
- Serviço renderiza via Chrome/chromedp.
- Cache Redis opcional.
- Upload para storage opcional.
- API `/api/v1/vre/*`.
- MCP tools VRE existem no stdio server.

Lacunas:
- Endpoints diferem da documentação.
- Preview diverge entre MCP client e backend.
- WebP não é realmente WebP.
- Data URL sem escaping.
- Orquestração “tool terminal visual” no fluxo do bot não está integrada de forma evidente ao pipeline AI/bot.

### MCP

Status: parcial.

Implementado:
- Server stdio.
- Server HTTP JSON-RPC manual.
- Tools, resources e prompts.
- Playground Docusaurus.

Lacunas:
- HTTP server não usa o `StreamableHTTPServerTransport` descrito no guia; implementa JSON-RPC manual.
- HTTP server não registra ferramentas VRE.
- `tools/index.ts` também não exporta VRE.
- `mcp/linktor-mcp-server` não tem `node_modules` instalado no ambiente auditado; typecheck não pôde ser executado localmente.

### SDKs e CLI

Status: presente, não auditado profundamente contra API real.

Implementado:
- SDKs Go, Python, TypeScript, PHP, Java, .NET e Rust estão no repo.
- CLI Go existe.

Risco:
- Como a API real está evoluindo e há divergências de rotas, os SDKs precisam de verificação contrato-a-contrato contra `cmd/server/main.go`.

## Validação executada

- `env GOCACHE=/tmp/linktor-go-build-cache go test ./...`
  - Primeiro falhou no sandbox por `httptest` não conseguir abrir porta local.
  - Reexecutado com permissão para listeners locais: passou.

- `npx tsc --noEmit` em `web/admin`
  - Passou.

- `npm run typecheck` em `mcp/linktor-mcp-server`
  - Não executou: `tsc` não encontrado; `node_modules` ausente.

- `npm run typecheck` em `web/docs`
  - Não executou: `tsc` não encontrado; `node_modules` ausente.

- `npm exec tsc -- --noEmit` em `web/landing`
  - Não validou: `node_modules` ausente; npm tentou resolver pacote externo `tsc` e retornou a mensagem padrão de que o compilador TypeScript precisa estar instalado localmente.

## Próximas correções recomendadas

1. Unificar migrações: usar um runner real para `deploy/docker/migrations` ou mover essas migrações para `RunMigrations`.
2. Adicionar teste de integração que sobe banco limpo, roda `cmd/server`/`RunMigrations` e chama pelo menos `/channels`, `/templates`, `/observability/stats`, `/channels/:id/coexistence-status`.
3. Registrar clients por canal para WhatsApp analytics/payments/calling/CTWA no fluxo de conexão.
4. Corrigir endpoints frontend de “test connection” ou implementar as rotas no backend.
5. Alinhar VRE docs, backend e MCP: path base, método de preview e contrato de resposta.
6. Corrigir renderer WebP ou alterar default/formato anunciado para JPEG/PNG.
7. Registrar VRE tools no MCP HTTP server e exportar em `tools/index.ts`.
8. Atualizar o planejamento de coexistência removendo a promessa de importação automática de 6 meses via Cloud API, substituindo por import manual CSV/JSON se esse for o caminho desejado.
