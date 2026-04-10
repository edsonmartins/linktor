# Auditoria Frontend vs API e Ambiente

Data: 2026-04-08

## Conclusão

O ambiente compila, mas ainda não deve ser considerado funcional end-to-end.

O backend Go passa nos testes e compila. O admin Next.js passa em `next build` e em `tsc` depois que o build gera `.next/types`. Docs e MCP também passam em build. A landing passa em build após instalar dependências.

Os bloqueios principais para declarar o ambiente como funcional são:

1. Docker/OrbStack não está acessível no ambiente local atual, então não foi possível validar backend + Postgres + Redis + NATS + MinIO rodando juntos.
2. `docker-compose.yml` só define dependências; não sobe backend, admin, docs ou landing.
3. Admin e landing usam `next@15.1.0`, versão com vulnerabilidades críticas reportadas por `npm audit`.
4. Há inconsistências de shape de resposta entre o `api` client do admin e alguns handlers do backend.
5. URLs de webhook exibidas no admin usam o host do frontend em vários canais, não o host público/API do backend.
6. A documentação principal ainda tem instruções antigas para login e coexistência que divergem do código atual.

## Validação Executada

### Backend

Comando:

```bash
env GOCACHE=/tmp/linktor-go-build-cache go test ./...
```

Resultado: passou.

Comando:

```bash
env GOCACHE=/tmp/linktor-go-build-cache go build -o /tmp/linktor-server-check ./cmd/server
```

Resultado: passou. Houve aviso de escrita no cache de módulo fora do sandbox, mas o build terminou com código 0.

### Admin

Comando:

```bash
npm run build
```

Diretório: `web/admin`

Resultado: passou.

Comando:

```bash
npx tsc --noEmit
```

Resultado:

- Antes de `next build`: falhou porque `tsconfig.json` inclui `.next/types/**/*.ts` e esses arquivos não existiam.
- Depois de `next build`: passou.

Isso indica dependência de ordem de comandos: `tsc` isolado em workspace limpo não é confiável sem gerar `.next/types` antes.

### Docs

Comando:

```bash
npm run build
```

Diretório: `web/docs`

Resultado: passou.

Ruídos restantes:

- `DEP0169` de dependência usando `url.parse()`.
- Docusaurus não conseguiu escrever cache de update-check em `/Users/edsonmartins/.config`.

Nenhum dos dois quebrou o build.

### MCP

Comando:

```bash
npm run build
```

Diretório: `mcp/linktor-mcp-server`

Resultado: passou.

### Landing

Comandos:

```bash
npm ci
npx tsc --noEmit
npm run build
```

Diretório: `web/landing`

Resultado: passou após instalar dependências.

`npm ci` emitiu aviso explícito de vulnerabilidade na versão do Next.

### Docker e Portas

Comando:

```bash
docker compose config
```

Resultado: configuração válida, com aviso de que `version` é obsoleto.

Comando:

```bash
docker compose ps
```

Resultado: falhou porque o Docker/OrbStack daemon não está acessível:

```text
Cannot connect to the Docker daemon at unix:///Users/edsonmartins/.orbstack/run/docker.sock. Is the docker daemon running?
```

Checagem de portas:

- `8081`: sem listener.
- `3000`: sem listener.

Ou seja: nada estava efetivamente rodando no momento da auditoria.

## Contrato Frontend vs API

### Rotas Operacionais

As rotas estáticas consumidas pelo admin existem no backend para os módulos principais:

- `auth`: `/auth/login`, `/auth/refresh`
- `analytics`: `/analytics/overview`, `/analytics/conversations`, `/analytics/flows`, `/analytics/escalations`, `/analytics/channels`
- `channels`: CRUD, enable, connect, disconnect, coexistence status e testes de canal
- `contacts`: list/create/get/update/delete
- `conversations`: list/get/messages/send
- `bots`: CRUD, activate/deactivate, channels, config, test
- `flows`: CRUD, activate/deactivate, test
- `knowledge-bases`: CRUD, items, bulk import, search, regenerate embeddings
- `oauth`: Facebook, Instagram e WhatsApp Embedded Signup
- `observability`: logs, queue, reset consumer, stats
- `users`: CRUD
- `ws`: `/api/v1/ws`

Não encontrei no admin o mesmo tipo de falha anterior de chamada direta para endpoint operacional inexistente.

### Divergências de Resposta

O client do admin em `web/admin/src/lib/api.ts` desembrulha qualquer resposta no formato:

```json
{ "success": true, "data": ... }
```

Isso funciona para handlers que usam `RespondSuccess` quando o frontend espera diretamente o objeto/array.

O problema é que o backend tem dois padrões de paginação:

1. `RespondWithMeta`: retorna `{ success: true, data, meta }`.
2. `RespondPaginated`: retorna `{ data, pagination }`, sem `success`.

Como o client desembrulha o caso 1, ele descarta `meta` e retorna apenas o array. Algumas telas ainda esperam `{ data: [...] }`.

Casos afetados:

- `web/admin/src/app/(dashboard)/conversations/page.tsx`: chama `/conversations`, mas usa `data?.data`; o backend retorna `success/data/meta`, e o client converte para array. Resultado provável: lista vazia.
- `web/admin/src/app/(dashboard)/contacts/page.tsx`: mesmo problema com `/contacts`.
- `web/admin/src/app/(dashboard)/users/page.tsx`: mesmo problema com `/users`.
- `web/admin/src/app/(dashboard)/dashboard/page.tsx`:
  - `/conversations` é tratado como `{ data: Conversation[] }`, mas vira array.
  - `/channels` é tratado como `{ data: Channel[] }`, mas vira array.
  - `/contacts` perde `meta`, então `contactsData?.total` não existe.
- `web/admin/src/app/(dashboard)/conversations/chat-view.tsx`: `/conversations/:id/messages` é `RespondWithMeta`; o client retorna array, mas a tela espera `messagesData?.data`.
- `web/admin/src/app/(dashboard)/knowledge-base/[id]/page.tsx`: `/knowledge-bases/:id/items` usa `RespondPaginated` com `{ data, pagination }`, mas a tela espera `itemsData?.total`; a tabela carrega itens, mas paginação tende a ficar errada.
- `web/admin/src/app/(dashboard)/bots/[id]/page.tsx`: `/channels` usa `RespondSuccess`; o client retorna array, mas a tela espera `channelsData?.data`.
- `web/admin/src/app/(dashboard)/bots/[id]/page.tsx`: `/knowledge-bases` usa `RespondPaginated`; aqui `kbData?.data` funciona, mas o tipo declarado não representa `pagination`.

Recomendação: padronizar o client para retornar `{ data, meta/pagination }` ou padronizar backend para um único envelope. O caminho mais seguro é preservar envelope no client e criar helpers para `data`/`meta`, porque hoje o unwrap automático perde metadados.

### URLs de Webhook no Admin

Várias telas montam URL de webhook com `window.location.origin`, ou seja, o host do admin. Em dev isso vira `http://localhost:3000/api/v1/...`, mas o backend está em `http://localhost:8081/api/v1`.

Casos:

- Telegram: `web/admin/src/app/(dashboard)/channels/telegram-config.tsx`
- SMS/Twilio: `web/admin/src/app/(dashboard)/channels/sms-config.tsx`
- WhatsApp: `web/admin/src/app/(dashboard)/channels/whatsapp-config.tsx`
- Facebook: `web/admin/src/app/(dashboard)/channels/facebook-config.tsx`
- Instagram: `web/admin/src/app/(dashboard)/channels/instagram-config.tsx`
- Email: `web/admin/src/app/(dashboard)/channels/email-config.tsx`

Além disso:

- Email UI mostra `/api/v1/webhooks/email/sendgrid/{channelId}`, mas o backend registra `/api/v1/webhooks/email/:channelId/sendgrid`.
- Email UI mostra `/api/v1/webhooks/email/sendgrid/{channelId}/events`, mas não há rota equivalente registrada.
- Voice UI mostra `/api/v1/webhooks/voice/:channelId`, mas não há rota `/webhooks/voice/:channelId` registrada no backend.
- Voice UI concatena `NEXT_PUBLIC_API_URL || 'http://localhost:8081'` com `/api/v1/...`; se `NEXT_PUBLIC_API_URL` seguir o padrão do client (`http://localhost:8081/api/v1`), a URL exibida vira `.../api/v1/api/v1/...`.

Recomendação: criar um helper único `publicApiBaseUrl`/`publicWebhookBaseUrl`, diferenciar base URL da API de base URL pública de webhooks, e corrigir ordem das rotas de email.

### WebSocket

O admin usa:

```ts
ws://localhost:8081/api/v1/ws
```

O backend registra:

```go
protected.GET("/ws", wsHandler.HandleConnection)
```

Esse contrato está alinhado. O WebSocket exige token via query string, e o hook envia `?token=...`.

## Segurança de Dependências

### Admin

`npm audit --audit-level=low` em `web/admin` falhou:

- `next@15.1.0`: vulnerabilidade crítica.
- `picomatch`: vulnerabilidade alta.

O audit informa correção para `next@15.5.14` via `npm audit fix --force`, porque está fora da versão fixa declarada atualmente.

### Landing

`npm audit --audit-level=low` em `web/landing` falhou com os mesmos grupos:

- `next@15.1.0`: vulnerabilidade crítica.
- `picomatch`: vulnerabilidade alta.

### Docs e MCP

`web/docs` e `mcp/linktor-mcp-server` passaram em audit com 0 vulnerabilidades.

## Ambiente Local

`docker-compose.yml` define apenas:

- Postgres
- Redis
- NATS
- MinIO

Não define:

- backend Go
- admin Next.js
- docs
- landing
- MCP HTTP server

Portanto, mesmo com Docker funcionando, `docker compose up -d` não sobe a aplicação completa. O ambiente funcional hoje depende de comandos manuais separados:

```bash
docker compose up -d
go run ./cmd/server
cd web/admin && npm run dev
```

E, para landing/docs/MCP:

```bash
cd web/landing && npm run dev
cd web/docs && npm run start
cd mcp/linktor-mcp-server && npm run build
```

Além disso, o Docker daemon não estava acessível no momento da validação, então não foi possível validar o backend contra os serviços reais.

## Documentação Divergente

O `README.md` ainda afirma:

- Coexistence importa até 6 meses automaticamente.
- Login demo como `admin@linktor.io / admin123`.
- Endpoints de histórico `/channels/:id/import-history`.

O código atual faz seed com:

- `admin@demo.com / admin123`
- `agent@demo.com / admin123`

E a análise anterior já concluiu que import automático de 6 meses via Cloud API não deve ser prometido.

## Status

Ambiente compilável: sim.

Ambiente funcional end-to-end validado: não.

Contrato de rotas admin/backend: majoritariamente alinhado.

Contrato de payload/response: ainda inconsistente em listas/paginação.

Pronto para produção: não, por vulnerabilidade crítica de Next no admin/landing e por ambiente Docker incompleto.
