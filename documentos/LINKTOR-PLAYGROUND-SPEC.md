# Linktor Playground — Especificação Técnica

**Status:** Draft para implementação
**Data:** 2026-07-11
**Autor:** Time Linktor
**Tipo:** Aplicação desktop multiplataforma (macOS / Windows / Linux) — projeto separado

---

## 1. Objetivo

O **Linktor Playground** é um app desktop, independente do backend, para **testar canais e o pipeline de mensagens do Linktor de ponta a ponta sem conectar provedores reais** (WhatsApp/Meta, Twilio, Telegram, etc.).

Público-alvo:
- **Devs** — validar mudanças no pipeline (webhook → NATS → bot/agente → resposta) localmente.
- **QA** — rodar cenários de conversa repetíveis com asserções antes de subir para produção.
- **Vendas / Onboarding** — demonstrar o produto sem precisar de uma conta WhatsApp Business aprovada.

Ele conecta a **qualquer instância Linktor** (dev local, staging, produção) via a API HTTP/WS já existente. **Não** requer nenhuma feature nova no backend para o MVP — apenas uma flag de ambiente (ver §6.1).

### Não-objetivos
- Não é um cliente de atendimento (isso é o admin dashboard).
- Não substitui o mock Prism da Graph API (`deploy/mocks/`); é complementar (ver §8).
- Não hospeda o backend; sempre aponta para um Linktor rodando.

---

## 2. Capacidades (o que o app faz)

| # | Módulo | Descrição | Depende de |
|---|--------|-----------|-----------|
| C1 | **Perfis de conexão** | Cadastra N backends (base URL + credenciais), login, troca rápida entre eles. Rótulo colorido por ambiente (dev/staging/**prod**). | `POST /auth/login`, `/auth/refresh`, `GET /me` |
| C2 | **Simulador WebChat** ⭐ | Age como *visitante*. Abre sessão WebSocket num canal `webchat`, envia mensagens e recebe respostas de bot/agente em uma UI de chat. Múltiplas sessões/personas simultâneas. | `GET /ws/:channelId`, `webchat/:id/config`, `widget-token` |
| C3 | **Injetor de webhook de entrada** | Para qualquer canal, monta e envia um payload `generic` (assinado com HMAC do `webhook_secret` do canal) simulando um cliente escrevendo naquele canal. | `POST /webhooks/generic/:channelId` |
| C4 | **Testador de Bot** | Envia uma mensagem para `POST /bots/:id/test`, mostra resposta, confiança, intent e se escalaria. | `POST /bots/:id/test` |
| C5 | **Testador de Flow** | Roda um flow com inputs simulados. | `POST /flows/:id/test` |
| C6 | **Inspector de pipeline** | Mostra o que o backend produziu a partir da mensagem: conversa criada, mensagem persistida, resposta do bot, escalonamento. Via polling da API (MVP) ou WS. | `GET /conversations`, `/messages`, ... |
| C7 | **Runner de cenários** | Sequências multi-passo roteirizadas (enviar msg → aguardar resposta → asserção "contém X"), salvas em YAML/JSON, com relatório pass/fail. | Combina C2–C6 |

**MVP = C1 + C2** (perfil/login + simulador WebChat). É o menor conjunto que entrega valor real: conversa credential-free ponta a ponta.

---

## 3. Stack técnica

### 3.1 Framework: Tauri v2
- **Core Rust** + **frontend web** (WebView nativo do SO). Binários pequenos, cross-platform (macOS, Windows, Linux).
- **Frontend:** React 18 + TypeScript + Vite. Tailwind + (opcional) reuso de componentes shadcn/ui do `web/admin` para consistência visual.
- **Rust:** onde vive a lógica de rede sensível (ver §3.2).

### 3.2 Por que fazer HTTP/WebSocket no lado Rust (decisão-chave)
O endpoint WebSocket do WebChat faz **checagem de `Origin`** (`internal/adapters/webchat/handler.go:228` `isOriginAllowed`). De dentro do WebView do Tauri, o `Origin` é `tauri://localhost` (macOS/Linux) ou `https://tauri.localhost` (Windows) — **não** está na allowlist do servidor, então o upgrade seria rejeitado (a menos que o servidor esteja em modo dev não-enforce e o origin seja localhost).

**Solução:** o cliente WebSocket roda em **Rust** (`tokio-tungstenite`), que nos deixa **definir o header `Origin` explicitamente**. Assim o app envia um `Origin` que o servidor aceita (ex.: `http://localhost:3000` no dev, ou um valor configurável no perfil de conexão) sem depender de mudança no servidor. Como bônus, evita CORS no WebView para as chamadas HTTP também.

> Regra do servidor (`isOriginAllowed`): allowlist explícita via `LINKTOR_WS_ALLOWED_ORIGINS` sempre vence; em modo não-enforce, origins `localhost`/`127.0.0.1` são aceitos; com `LINKTOR_WS_ENFORCE_ORIGIN=1` só a allowlist passa. O Playground deve expor no perfil de conexão um campo **"WebSocket Origin"** (default `http://localhost:3000`) para casar com o que o servidor-alvo aceita.

### 3.3 Bibliotecas Rust
- `reqwest` — HTTP (login, canais, injeção de webhook, testes de bot/flow).
- `tokio-tungstenite` — cliente WebSocket com header `Origin` customizável.
- `hmac` + `sha2` — assinatura HMAC do injetor de webhook (C3).
- `keyring` (ou `tauri-plugin-stronghold`) — armazenamento seguro de tokens/refresh no keychain do SO.
- `serde` / `serde_json` — payloads.

### 3.4 Persistência local
- **Config não-sensível** (perfis sem segredo, personas, cenários, templates de webhook): JSON no diretório de config do app (`tauri::api::path::app_config_dir`).
- **Segredos** (senha? não — só refresh token e `webhook_secret` que o usuário colar): **keychain do SO** via `keyring`. Nunca em texto plano no disco.

---

## 4. Integração com o backend Linktor (contratos reais)

> Todos os paths abaixo foram verificados no código atual. Base: `${API_URL}` = `https://<host>/api/v1`. WebSocket é fora do prefixo `/api/v1`.

### 4.1 Autenticação
`POST ${API_URL}/auth/login`
```json
{ "email": "user@tenant.com", "password": "..." }
```
Resposta (`data`): `{ user, access_token, refresh_token, expires_in }`. O backend **também** seta cookies HttpOnly, mas o app desktop usa o **`access_token` do corpo** como `Authorization: Bearer <token>` (mais simples que gerenciar cookies).

- `POST ${API_URL}/auth/refresh` `{ "refresh_token": "..." }` → novo par de tokens.
- `GET ${API_URL}/me` (Bearer) → valida sessão / mostra tenant e usuário.

**Auth por API Key (confirmado — `middleware/auth.go:88`):** header **`X-API-Key`** (habilitado quando o `apiKeyService` está wired no servidor). Ordem de precedência no backend: `Authorization: Bearer` → cookie `access_token` → `X-API-Key`. O Playground suporta os **dois** métodos por perfil (§9): senha (uso interativo) e API Key (automação/CI/cenários sem senha).

### 4.2 Canais
- `GET ${API_URL}/channels` (Bearer) — listar; filtrar `type == "webchat"` para o simulador.
- `POST ${API_URL}/channels` (Bearer) — criar canal webchat direto do app (conveniência).

### 4.3 WebChat (módulo C2)
1. **(opcional) Config do widget:** `GET ${API_URL}/webchat/:channelId/config` (sem auth) → título, cor, welcome message.
2. **(se o canal tiver `widget_secret`) Token:** `POST ${API_URL}/webchat/:channelId/widget-token` (Bearer) → token curto. Se o canal **não** tiver `widget_secret`, o WS é aberto sem token (dev).
3. **WebSocket:** `GET ${WS_URL}/ws/:channelId?session_id=<uuid>&name=<nome>&email=<email>&token=<widget-token>`
   - `session_id` — persistente por persona (mantém o mesmo contato/conversa entre reconexões).
   - Header `Origin` — setado pelo cliente Rust (§3.2).

**Protocolo WebSocket** (`internal/adapters/webchat/websocket.go`) — mensagens JSON `{ "type": ..., "payload": {...} }`:

| Direção | `type` | Uso |
|---------|--------|-----|
| App → servidor | `message` | Enviar mensagem do visitante. Payload: `{ id, content_type: "text", content, attachments?, metadata? }` |
| App → servidor | `typing` | (opcional) indicador de digitação |
| App → servidor | `read` | (opcional) recibo de leitura |
| Servidor → app | `connect` | Confirmação de conexão; payload traz `session_id`, `widget_title`, `widget_color` |
| Servidor → app | `ack` | ACK de uma mensagem enviada (ecoa o `id`) |
| Servidor → app | `message` | **Resposta do bot/agente** — renderizar no chat |
| Servidor → app | `typing` / `presence` | (opcional) estados |
| Servidor → app | `error` | payload `{ error }` |

Fluxo: enviar `message` → receber `ack` → (backend processa: contato/conversa criados, publica inbound no NATS, bot/agente responde) → chega `message` de volta. Nenhuma credencial de provedor envolvida.

### 4.4 Injetor de webhook genérico (módulo C3)
`POST ${API_URL}/webhooks/generic/:channelId`

Body (`GenericWebhookPayload`):
```json
{
  "message_id": "ext-123",
  "sender_id": "+5511999999999",
  "sender_name": "Cliente Teste",
  "content_type": "text",
  "content": "Olá, quero testar",
  "metadata": { "qualquer": "coisa" }
}
```
**Assinatura HMAC (confirmado — `webhook.go:658-688`):**
- Header: **`X-Linktor-Signature`** (fallback aceito: `X-Hub-Signature-256`).
- Valor: **`sha256=` + `hex(HMAC_SHA256(raw_body, webhook_secret))`** — hex minúsculo, calculado sobre o **corpo cru** (os mesmos bytes enviados).
- Comparação constant-time no servidor (`hmac.Equal`).
- Se o canal **não** tiver `webhook_secret`, passa sem assinatura — **exceto** se o servidor rodar com `requireWebhookSecrets` (retorna 401). Para dev, um canal sem secret dispensa a assinatura.

Implementação Rust (C3): assinar com `hmac`+`sha2`, montar `format!("sha256={}", hex::encode(mac))` e enviar em `X-Linktor-Signature`. O `webhook_secret` é colado pelo usuário no app (segredo do servidor, não exposto por API) e guardado no keychain.

Resposta: `{ "status": "ok", "message_id": "<uuid interno>" }`.

Também disponível: `POST ${API_URL}/webhooks/status/:channelId` para simular callbacks de status de entrega (delivered/read).

### 4.5 Testadores diretos (módulos C4/C5)
- `POST ${API_URL}/bots/:id/test` (Bearer) `{ "message": "..." }` → `{ response, confidence, intent, should_escalate, tokens_used, latency_ms }`. **Single-turn** (sem estado de conversa).
- `POST ${API_URL}/flows/:id/test` (Bearer) `{ "inputs": {...} }` → resultado da execução do flow.

---

## 5. Arquitetura do app

```
┌─────────────────────────────────────────────┐
│  Frontend (React + TS, WebView Tauri)        │
│  - Profile manager      - Chat panels (C2)   │
│  - Channel picker       - Webhook injector   │
│  - Bot/Flow testers     - Inspector          │
│  - Scenario editor/runner                    │
└───────────────▲───────────────┬──────────────┘
        eventos │ (tauri emit)   │ invoke (comandos)
┌───────────────┴───────────────▼──────────────┐
│  Core Rust (Tauri commands)                   │
│  auth: login/refresh/me                       │
│  channels: list/create                        │
│  ws: connect / send / (emite msgs recebidas)  │
│  webhook: inject (+ HMAC sign)                │
│  test: bot / flow                             │
│  secrets: keychain (tokens, webhook_secret)   │
│  reqwest (HTTP) · tokio-tungstenite (WS)      │
└───────────────┬───────────────────────────────┘
                │ HTTP + WS (Origin controlado)
        ┌───────▼────────┐
        │ Linktor backend │  (dev / staging / prod)
        └─────────────────┘
```

### Comandos Tauri (Rust → exposto ao front)
```
login(profile_id, email, password) -> Session
refresh(profile_id) -> Session
me(profile_id) -> User
list_channels(profile_id) -> Channel[]
create_webchat_channel(profile_id, name) -> Channel
ws_connect(profile_id, channel_id, persona) -> session_handle   // emite eventos "ws://<handle>/message"
ws_send(session_handle, content) -> ack_id
ws_disconnect(session_handle)
inject_webhook(profile_id, channel_id, payload, webhook_secret) -> result
test_bot(profile_id, bot_id, message) -> BotTestResult
test_flow(profile_id, flow_id, inputs) -> FlowTestResult
get_conversation(profile_id, conversation_id) -> Conversation   // inspector
run_scenario(profile_id, scenario) -> ScenarioReport
```
Mensagens WebSocket recebidas são empurradas ao front via `app.emit()` (evento por sessão), não por polling.

---

## 6. Requisitos no servidor-alvo

### 6.1 Ambiente de dev (recomendado para o Playground)
Uma destas opções para o WebSocket aceitar o app:
- **(preferida)** App envia `Origin: http://localhost:3000` (configurável no perfil) → aceito pelo modo dev não-enforce ou por allowlist.
- ou servidor com `LINKTOR_WS_ALLOWED_ORIGINS` incluindo o origin que o app usa.
- ou servidor com `LINKTOR_WS_ALLOW_EMPTY_ORIGIN=1` (se o app enviar Origin vazio).

Canal WebChat **sem** `widget_secret` → WS abre sem token (mais simples para dev). Com `widget_secret`, o app minta o token via API (§4.3).

### 6.2 Contra produção
- Modo enforce (`LINKTOR_WS_ENFORCE_ORIGIN=1`) provavelmente ativo → o `Origin` do app precisa estar na allowlist de produção. Documentar o valor a usar.
- Canais reais têm `widget_secret` → token obrigatório.
- **Guard de segurança no app:** perfil marcado como `prod` exige confirmação explícita antes de injetar webhooks ou criar canais (evita poluir produção).

---

## 7. Modelo de dados local

```jsonc
// profile
{ "id", "label", "env": "dev|staging|prod", "api_url", "ws_url",
  "ws_origin", "auth_method": "password|api_key",
  "user_email", /* refresh_token → keychain */ }

// persona (visitante do WebChat)
{ "id", "name", "email?", "phone?", "session_id" /* estável */ }

// webhook_template
{ "id", "channel_type", "payload": {...GenericWebhookPayload} }

// scenario
{ "id", "name", "channel_id",
  "steps": [ { "send": "texto", "expect": { "contains": ["..."], "timeout_ms": 5000 } } ] }
```

---

## 8. Relação com ferramentas existentes

| Ferramenta | Papel | Playground usa? |
|-----------|-------|-----------------|
| Canal **WebChat** | Canal real credential-free | ✅ base do C2 |
| **GenericWebhook** | Injeção de entrada em qualquer canal | ✅ base do C3 |
| **Prism mock** (`deploy/mocks/`) | Mocka a Graph API de **saída** (WhatsApp sem Meta) | ⛓️ complementar: rode o backend com `LINKTOR_GRAPH_API_URL=:4010` e use o Playground para injetar a entrada; a saída WhatsApp cai no mock |
| **Bot Test** dialog (admin) | Teste single-turn de bot | ➡️ reimplementado como C4 |
| `web/embed/widget.js` | Widget web de referência | 📖 referência do protocolo WS |

Cenário WhatsApp completo credential-free: **Playground (injeta inbound) → backend → bot → outbound → Prism mock**.

---

## 9. Decisões (fechadas — 2026-07-11)

1. **Localização do código:** ✅ **Repo separado** `linktor-playground`. Release, versionamento e assinatura de binário independentes do backend.
2. **SOs do primeiro release:** ✅ **macOS + Linux**. Windows fica para uma fase posterior (exige assinatura de código / SmartScreen — ver §10 M7).
3. **Auth:** ✅ **Senha + API Key**. Login usuário/senha para uso interativo e `X-API-Key` para automação/CI (§4.1).
4. **Design system:** ✅ **Reusar o do `web/admin`** (shadcn/ui + Tailwind). Portar os componentes necessários para consistência visual com o produto.
5. **Distribuição/assinatura:** notarização macOS + empacotamento Linux (AppImage/deb) no M7; auto-update via `tauri-plugin-updater` opcional no M7. Assinatura Windows entra junto com o suporte a Windows (pós-M7).

### Implicações do escopo fechado
- **Scaffold (M0):** targets Tauri `aarch64/x86_64-apple-darwin` + `x86_64-unknown-linux-gnu`. Não configurar target Windows por ora.
- **Design:** criar um pacote/pasta de UI portando tokens Tailwind + componentes shadcn/ui do `web/admin` (botão, input, dialog, card, tabs). Evitar dependências específicas de Next.js — o Playground é Vite/React puro.
- **Perfil de conexão:** campo `auth_method` = `password | api_key`; quando `api_key`, guardar a chave no keychain e enviar `X-API-Key` em vez de `Bearer`.

---

## 10. Roadmap sugerido

| Marco | Entrega | Módulos |
|-------|---------|---------|
| **M0** | Scaffold Tauri v2 + React + Vite; perfil de conexão; login + `/me`; keychain | C1 |
| **M1** ⭐ | Simulador WebChat: cliente WS em Rust (Origin configurável), UI de chat 1 sessão | C2 |
| **M2** | Gerência de canais (listar/criar webchat); múltiplas sessões/personas | C2+ |
| **M3** | Injetor de webhook genérico com HMAC (após confirmar header de assinatura) | C3 |
| **M4** | Testadores de Bot e Flow | C4, C5 |
| **M5** | Inspector de pipeline (conversa/mensagens) | C6 |
| **M6** | Runner de cenários com asserções + relatório | C7 |
| **M7** | Empacotamento macOS (notarização) + Linux (AppImage/deb); auto-update opcional | — |
| **M8** (pós) | Suporte a Windows + assinatura de código (SmartScreen) | — |

**Definição de "pronto para uso interno":** fim do M3 (conversa WebChat + injeção de webhook, contra dev local). M1 já é demonstrável.

---

## 11. Riscos / pontos de atenção

- **Origin do WebSocket** (§3.2/§6) — a principal pegadinha; resolvido fazendo o WS em Rust com Origin configurável. Validar cedo, no M1.
- ~~**Header/formato do HMAC** do GenericWebhook~~ — **confirmado** (§4.4): `X-Linktor-Signature: sha256=hex(HMAC_SHA256(body, secret))`.
- **`widget_secret` em produção** — token obrigatório; garantir o fluxo de mintagem (§4.3) antes de testar contra prod.
- **Segurança contra prod** — guard de confirmação e rótulo de ambiente são obrigatórios, não opcionais (o app pode escrever em qualquer backend).
- **Nomes internos** — o backend usa o módulo Go `msgfy`/`linktor`; conferir base URL e versionamento de API (`/api/v1`) do alvo.
