# Plano de Execução – Projeto Linktor (v1.0)

> **Instrução:** Sempre que uma tarefa avançar de status, atualize esta tabela com a nova situação e registre a data no campo "Última atualização". Os status sugeridos são `TODO`, `IN_PROGRESS`, `BLOCKED` e `DONE`.

## Legend
- `TODO`: ainda não iniciado.
- `IN_PROGRESS`: em execução.
- `BLOCKED`: impedida por dependência externa.
- `DONE`: concluída e validada.

**IMPORTANTE:**

- Seguir arquitetura hexagonal (Ports & Adapters) com bounded contexts
- Iniciar como Modular Monolith, evoluir para Microservices
- Plugin System via HashiCorp go-plugin com gRPC isolation
- NATS JetStream para mensageria assíncrona
- PostgreSQL multi-tenant com Redis para cache/rate limiting
- **NÃO implementar testes** neste momento (foco em MVP funcional)
- **NÃO implementar observabilidade** neste momento

**CONTEXTO DO PROJETO:**
O Linktor é uma plataforma open-source de mensageria B2B multichannel powered by msgfy engine. Resolve o problema de fragmentação de canais de comunicação, oferecendo:
1. **Unificação de Canais:** WhatsApp, SMS, Telegram, RCS, Instagram, Facebook Messenger, Web Chat, Voice
2. **Normalização de Mensagens:** Formato canônico independente do canal
3. **Plugin System:** Adapters extensíveis para cada canal
4. **Multi-Tenancy:** Isolamento por tenant com planos de assinatura

**Branding:**
- **LINKTOR** (linktor.io): Plataforma completa, documentação, cloud hosting
- **msgfy** (GitHub org): Core engine (Apache 2.0), SDKs, CLI tools

---

## 📊 STATUS GERAL DO PROJETO

| Fase | Sprint | Progresso | Status | Descrição |
|------|--------|-----------|--------|-----------|
| **FASE 1: Foundation** | 1-2 | 100% | ✅ DONE | Monorepo, CI/CD, Database, API Gateway |
| **FASE 2: Core Messaging** | 3-4 | 100% | ✅ DONE | Messaging Service, Plugin Loader, Web Chat |
| **FASE 3: Admin Panel MVP** | 5-6 | 100% | ✅ DONE | React UI, Auth, Dashboard, WebSocket Real-time |
| **FASE 4: WhatsApp Official** | 7-8 | 0% | 🎯 NEXT | Meta Cloud API, Webhooks, Media |
| **FASE 5: SDKs & CLI** | 9-10 | 0% | TODO | Go, Java, Python, TypeScript, .NET SDKs |
| **FASE 6: More Channels** | 11-12 | 0% | TODO | SMS, Telegram, RCS, Instagram |
| **FASE 7: Analytics & Enterprise** | 13-14 | 0% | TODO | Analytics, Webhooks, Multi-tenant |
| **FASE 8: Beta Release** | 15-16 | 0% | TODO | Bug fixes, Performance, Security |

---

## 🏗️ FASE 1: FOUNDATION (Sprints 1-2, Semanas 1-4)

### Sprint 1 - Setup Inicial

| ID | Tarefa | Descrição | Deps | Status | Última Atualização |
|----|--------|-----------|------|--------|-------------------|
| 1.1.1 | Setup Monorepo | Criar estrutura de diretórios seguindo padrão Go modules | - | DONE | 2026-02-02 |
| 1.1.2 | Configurar Go Modules | go.mod, go.sum, workspace setup | 1.1.1 | DONE | 2026-02-02 |
| 1.1.3 | Setup Docker Compose | Postgres, Redis, NATS JetStream para dev | 1.1.1 | DONE | 2026-02-02 |
| 1.1.4 | Configurar CI/CD | GitHub Actions: lint, test, build, security scan | 1.1.2 | TODO | - |
| 1.1.5 | Setup Buf para Protobuf | buf.yaml, buf.gen.yaml, proto estrutura | 1.1.1 | DONE | 2026-02-02 |
| 1.1.6 | Definir Protobuf Core | message.proto, conversation.proto, contact.proto | 1.1.5 | DONE | 2026-02-02 |
| 1.1.7 | Configurar Linting | golangci-lint, .editorconfig, pre-commit hooks | 1.1.2 | DONE | 2026-02-02 |

### Sprint 2 - Database & API Gateway

| ID | Tarefa | Descrição | Deps | Status | Última Atualização |
|----|--------|-----------|------|--------|-------------------|
| 1.2.1 | Schema Tenants | Tabela tenants com subscription plans | 1.1.3 | DONE | 2026-02-02 |
| 1.2.2 | Schema Users | Tabela users com roles e permissions | 1.2.1 | DONE | 2026-02-02 |
| 1.2.3 | Schema Channels | Tabela channels com config JSON | 1.2.1 | DONE | 2026-02-02 |
| 1.2.4 | Schema Contacts | Tabelas contacts + contact_identities | 1.2.1 | DONE | 2026-02-02 |
| 1.2.5 | Schema Conversations | Tabela conversations com status tracking | 1.2.4 | DONE | 2026-02-02 |
| 1.2.6 | Schema Messages | Tabelas messages + message_attachments | 1.2.5 | DONE | 2026-02-02 |
| 1.2.7 | API Gateway Skeleton | Gin router, middleware base, health check | 1.1.7 | DONE | 2026-02-02 |
| 1.2.8 | JWT Authentication | Login, refresh token, middleware auth | 1.2.7 | DONE | 2026-02-02 |
| 1.2.9 | Rate Limiting | Redis token bucket por tenant/channel | 1.2.8 | DONE | 2026-02-02 |
| 1.2.10 | Tenant Middleware | Extração tenant do JWT, context injection | 1.2.8 | DONE | 2026-02-02 |

---

## 📨 FASE 2: CORE MESSAGING (Sprints 3-4, Semanas 5-8)

### Sprint 3 - Messaging Service

| ID | Tarefa | Descrição | Deps | Status | Última Atualização |
|----|--------|-----------|------|--------|-------------------|
| 2.1.1 | Domain Message | Entidade Message com status tracking | 1.2.6 | DONE | 2026-02-03 |
| 2.1.2 | Message Repository | Interface + implementação PostgreSQL | 2.1.1 | DONE | 2026-02-03 |
| 2.1.3 | NATS Producer | Publicar mensagens outbound no JetStream | 2.1.2 | DONE | 2026-02-03 |
| 2.1.4 | NATS Consumer | Consumir mensagens inbound do JetStream | 2.1.3 | DONE | 2026-02-03 |
| 2.1.5 | Message Normalizer | Converter mensagens para formato canônico | 2.1.1 | DONE | 2026-02-03 |
| 2.1.6 | Send Message Use Case | Validar, normalizar, persistir, publicar | 2.1.5 | DONE | 2026-02-03 |
| 2.1.7 | Receive Message Use Case | Consumir, desnormalizar, persistir | 2.1.6 | DONE | 2026-02-03 |
| 2.1.8 | Message Status Webhook | Callback para atualização de status | 2.1.7 | DONE | 2026-02-03 |

### Sprint 4 - Plugin System & Web Chat

| ID | Tarefa | Descrição | Deps | Status | Última Atualização |
|----|--------|-----------|------|--------|-------------------|
| 2.2.1 | Plugin Interface | ChannelAdapter interface com gRPC | 2.1.8 | DONE | 2026-02-03 |
| 2.2.2 | Plugin Loader | HashiCorp go-plugin handshake | 2.2.1 | DONE | 2026-02-03 |
| 2.2.3 | Plugin Registry | Registro dinâmico de adapters | 2.2.2 | DONE | 2026-02-03 |
| 2.2.4 | Web Chat Adapter | Implementar adapter para WebSocket | 2.2.3 | DONE | 2026-02-03 |
| 2.2.5 | WebSocket Server | Conexões real-time para web chat | 2.2.4 | DONE | 2026-02-03 |
| 2.2.6 | Chat Widget Embed | Script embeddable para websites | 2.2.5 | DONE | 2026-02-03 |
| 2.2.7 | Conversation Service | CRUD de conversas, assignment | 2.1.8 | DONE | 2026-02-03 |
| 2.2.8 | Contact Service | CRUD de contatos, identidades | 2.2.7 | DONE | 2026-02-03 |

---

## 🖥️ FASE 3: ADMIN PANEL MVP (Sprints 5-6, Semanas 9-12) ✅

### Sprint 5 - Frontend Foundation

| ID | Tarefa | Descrição | Deps | Status | Última Atualização |
|----|--------|-----------|------|--------|-------------------|
| 3.1.1 | Setup Next.js 15 + React 19 | TypeScript 5.3+, estrutura de pastas | 2.2.8 | DONE | 2026-02-03 |
| 3.1.2 | Configurar TailwindCSS | Terminal theme, Radix UI setup | 3.1.1 | DONE | 2026-02-03 |
| 3.1.3 | Setup React Query + Zustand | QueryClient, stores, hooks base | 3.1.2 | DONE | 2026-02-03 |
| 3.1.4 | Auth Context | Login, logout, token refresh, protected routes | 3.1.3 | DONE | 2026-02-03 |
| 3.1.5 | Layout Principal | Sidebar colapsável, header, main content | 3.1.4 | DONE | 2026-02-03 |
| 3.1.6 | Tela de Login | Form login com validação, terminal theme | 3.1.5 | DONE | 2026-02-03 |
| 3.1.7 | Dashboard Home | Cards de métricas, activity log | 3.1.6 | DONE | 2026-02-03 |

### Sprint 6 - Real-time Messaging UI

| ID | Tarefa | Descrição | Deps | Status | Última Atualização |
|----|--------|-----------|------|--------|-------------------|
| 3.2.1 | Lista de Conversas | Sidebar com conversas, filtros, busca | 3.1.7 | DONE | 2026-02-03 |
| 3.2.2 | Chat View | Área de mensagens, message bubbles | 3.2.1 | DONE | 2026-02-03 |
| 3.2.3 | Composer | Input de texto com send button | 3.2.2 | DONE | 2026-02-03 |
| 3.2.4 | WebSocket Client | Conexão real-time, reconnect, typing indicators | 3.2.3 | DONE | 2026-02-03 |
| 3.2.5 | Notificações | Toast system implementado | 3.2.4 | DONE | 2026-02-03 |
| 3.2.6 | Lista de Contatos | CRUD de contatos, tags | 3.2.5 | DONE | 2026-02-03 |
| 3.2.7 | Configuração de Canais | UI para visualizar canais | 3.2.6 | DONE | 2026-02-03 |
| 3.2.8 | Perfil de Usuário | Settings completo (6 seções) | 3.2.7 | DONE | 2026-02-03 |

---

## 📱 FASE 4: WHATSAPP OFFICIAL (Sprints 7-8, Semanas 13-16)

### Sprint 7 - Meta Cloud API

| ID | Tarefa | Descrição | Deps | Status | Última Atualização |
|----|--------|-----------|------|--------|-------------------|
| 4.1.1 | Meta App Setup | Documentar setup Meta Developer Portal | 3.2.8 | TODO | - |
| 4.1.2 | WhatsApp Adapter | Implementar ChannelAdapter para Meta API | 4.1.1 | TODO | - |
| 4.1.3 | Webhook Receiver | Endpoint para webhooks do Meta | 4.1.2 | TODO | - |
| 4.1.4 | Message Templates | Suporte a templates HSM | 4.1.3 | TODO | - |
| 4.1.5 | Media Upload | Upload de imagens, vídeos, documentos | 4.1.4 | TODO | - |
| 4.1.6 | Media Download | Download de mídia recebida | 4.1.5 | TODO | - |
| 4.1.7 | Status Callbacks | Processar callbacks de status | 4.1.6 | TODO | - |

### Sprint 8 - WhatsApp Features Avançadas

| ID | Tarefa | Descrição | Deps | Status | Última Atualização |
|----|--------|-----------|------|--------|-------------------|
| 4.2.1 | Interactive Messages | Buttons, lists, reply buttons | 4.1.7 | TODO | - |
| 4.2.2 | Catalog Messages | Product messages, multi-product | 4.2.1 | TODO | - |
| 4.2.3 | Location Messages | Envio/recebimento de localização | 4.2.2 | TODO | - |
| 4.2.4 | Contacts Messages | Compartilhamento de contatos | 4.2.3 | TODO | - |
| 4.2.5 | Read Receipts | Marcar mensagens como lidas | 4.2.4 | TODO | - |
| 4.2.6 | Typing Indicators | Mostrar "digitando..." | 4.2.5 | TODO | - |
| 4.2.7 | Rate Limit Handling | Backoff exponencial, queuing | 4.2.6 | TODO | - |

---

## 🔧 FASE 5: SDKs & CLI (Sprints 9-10, Semanas 17-20)

### Sprint 9 - SDK Core & Go

| ID | Tarefa | Descrição | Deps | Status | Última Atualização |
|----|--------|-----------|------|--------|-------------------|
| 5.1.1 | SDK Spec | Definir API pública dos SDKs | 4.2.7 | TODO | - |
| 5.1.2 | SDK-Go Base | Client HTTP, auth, error handling | 5.1.1 | TODO | - |
| 5.1.3 | SDK-Go Messages | Send, receive, list messages | 5.1.2 | TODO | - |
| 5.1.4 | SDK-Go Conversations | CRUD conversas | 5.1.3 | TODO | - |
| 5.1.5 | SDK-Go Contacts | CRUD contatos | 5.1.4 | TODO | - |
| 5.1.6 | SDK-Go Webhooks | Webhook receiver helper | 5.1.5 | TODO | - |
| 5.1.7 | SDK-Go Docs | README, examples, godoc | 5.1.6 | TODO | - |

### Sprint 10 - SDKs Python, TypeScript, Java & CLI

| ID | Tarefa | Descrição | Deps | Status | Última Atualização |
|----|--------|-----------|------|--------|-------------------|
| 5.2.1 | SDK-Python | asyncio client completo | 5.1.7 | TODO | - |
| 5.2.2 | SDK-TypeScript | fetch-based client | 5.1.7 | TODO | - |
| 5.2.3 | SDK-Java | HttpClient + OkHttp based | 5.1.7 | TODO | - |
| 5.2.4 | SDK-.NET | HttpClient-based | 5.1.7 | TODO | - |
| 5.2.5 | CLI Setup | Cobra CLI, config management | 5.1.7 | TODO | - |
| 5.2.6 | CLI Channels | msgfy channel list/add/remove | 5.2.5 | TODO | - |
| 5.2.7 | CLI Messages | msgfy send/receive commands | 5.2.6 | TODO | - |
| 5.2.8 | CLI Contacts | msgfy contact CRUD | 5.2.7 | TODO | - |

---

## 📡 FASE 6: MORE CHANNELS (Sprints 11-12, Semanas 21-24)

### Sprint 11 - SMS & Telegram

| ID | Tarefa | Descrição | Deps | Status | Última Atualização |
|----|--------|-----------|------|--------|-------------------|
| 6.1.1 | Twilio Setup | Documentar setup Twilio account | 5.2.7 | TODO | - |
| 6.1.2 | SMS Adapter | Twilio SDK integration | 6.1.1 | TODO | - |
| 6.1.3 | MMS Support | Envio/recebimento de MMS | 6.1.2 | TODO | - |
| 6.1.4 | SMS Webhooks | Status callbacks Twilio | 6.1.3 | TODO | - |
| 6.1.5 | Telegram Adapter | go-telegram-bot-api integration | 6.1.4 | TODO | - |
| 6.1.6 | Telegram Media | Fotos, vídeos, documentos | 6.1.5 | TODO | - |
| 6.1.7 | Telegram Keyboards | Inline e reply keyboards | 6.1.6 | TODO | - |

### Sprint 12 - RCS & Instagram

| ID | Tarefa | Descrição | Deps | Status | Última Atualização |
|----|--------|-----------|------|--------|-------------------|
| 6.2.1 | RCS Provider Setup | Zenvia/Infobip setup | 6.1.7 | TODO | - |
| 6.2.2 | RCS Adapter | Rich messaging, carousels | 6.2.1 | TODO | - |
| 6.2.3 | RCS Suggestions | Suggested replies/actions | 6.2.2 | TODO | - |
| 6.2.4 | Instagram Adapter | Meta Graph API Instagram | 6.2.3 | TODO | - |
| 6.2.5 | Instagram Stories | Reply to stories | 6.2.4 | TODO | - |
| 6.2.6 | Facebook Messenger | Meta Messenger Platform | 6.2.5 | TODO | - |
| 6.2.7 | Channel Auto-detection | Detectar canal pelo contact identity | 6.2.6 | TODO | - |

---

## 📊 FASE 7: ANALYTICS & ENTERPRISE (Sprints 13-14, Semanas 25-28)

### Sprint 13 - Analytics Service

| ID | Tarefa | Descrição | Deps | Status | Última Atualização |
|----|--------|-----------|------|--------|-------------------|
| 7.1.1 | Events Schema | Tabela analytics_events | 6.2.7 | TODO | - |
| 7.1.2 | Event Tracking | Eventos de mensagem, conversa, user | 7.1.1 | TODO | - |
| 7.1.3 | Metrics Aggregation | Métricas por período, canal, agente | 7.1.2 | TODO | - |
| 7.1.4 | Dashboard Analytics | Gráficos, KPIs, exportação | 7.1.3 | TODO | - |
| 7.1.5 | Reports API | Endpoints para relatórios | 7.1.4 | TODO | - |
| 7.1.6 | Scheduled Reports | Relatórios automáticos por email | 7.1.5 | TODO | - |

### Sprint 14 - Multi-tenant & Webhooks

| ID | Tarefa | Descrição | Deps | Status | Última Atualização |
|----|--------|-----------|------|--------|-------------------|
| 7.2.1 | Subscription Plans | Limites por plano, feature flags | 7.1.6 | TODO | - |
| 7.2.2 | Usage Metering | Contagem de mensagens, storage | 7.2.1 | TODO | - |
| 7.2.3 | Billing Integration | Stripe/PagSeguro webhooks | 7.2.2 | TODO | - |
| 7.2.4 | Outbound Webhooks | Configurar webhooks por tenant | 7.2.3 | TODO | - |
| 7.2.5 | Webhook Retry | Retry com exponential backoff | 7.2.4 | TODO | - |
| 7.2.6 | Webhook Logs | Log de entregas, debugging | 7.2.5 | TODO | - |
| 7.2.7 | API Keys | Geração, rotação, scopes | 7.2.6 | TODO | - |

---

## 🚀 FASE 8: BETA RELEASE (Sprints 15-16, Semanas 29-32)

### Sprint 15 - Stabilization

| ID | Tarefa | Descrição | Deps | Status | Última Atualização |
|----|--------|-----------|------|--------|-------------------|
| 8.1.1 | Bug Fixes | Correção de bugs reportados | 7.2.7 | TODO | - |
| 8.1.2 | Performance Tuning | Otimização de queries, caching | 8.1.1 | TODO | - |
| 8.1.3 | Load Testing | k6/Locust scripts, benchmarks | 8.1.2 | TODO | - |
| 8.1.4 | Security Audit | OWASP Top 10, dependency scan | 8.1.3 | TODO | - |
| 8.1.5 | Penetration Testing | Testes de intrusão básicos | 8.1.4 | TODO | - |
| 8.1.6 | Error Handling | Mensagens de erro user-friendly | 8.1.5 | TODO | - |

### Sprint 16 - Documentation & Launch

| ID | Tarefa | Descrição | Deps | Status | Última Atualização |
|----|--------|-----------|------|--------|-------------------|
| 8.2.1 | API Documentation | OpenAPI spec completo | 8.1.6 | TODO | - |
| 8.2.2 | User Guide | Guia de usuário do admin panel | 8.2.1 | TODO | - |
| 8.2.3 | Developer Guide | Guia de integração com SDKs | 8.2.2 | TODO | - |
| 8.2.4 | Deployment Guide | Docker, Kubernetes, cloud providers | 8.2.3 | TODO | - |
| 8.2.5 | Changelog | CHANGELOG.md completo | 8.2.4 | TODO | - |
| 8.2.6 | Release Notes | v1.0.0-beta release | 8.2.5 | TODO | - |
| 8.2.7 | Website Launch | linktor.io com docs | 8.2.6 | TODO | - |

---

## 📁 ESTRUTURA DE DIRETÓRIOS PROPOSTA

```
linktor/
├── cmd/
│   ├── server/          # API Gateway + Services
│   └── cli/             # msgfy CLI
├── internal/
│   ├── api/             # REST handlers
│   ├── grpc/            # gRPC services
│   ├── domain/          # Entities, Value Objects
│   ├── application/     # Use Cases
│   ├── infrastructure/  # DB, NATS, Redis
│   └── adapters/        # Channel Adapters
├── pkg/                 # Shared libraries
├── plugins/             # External adapter plugins
├── proto/               # Protobuf definitions
├── web/                 # React Admin Panel
│   ├── src/
│   │   ├── components/
│   │   ├── pages/
│   │   ├── hooks/
│   │   └── services/
│   └── public/
├── sdk/
│   ├── go/
│   ├── java/
│   ├── python/
│   ├── typescript/
│   └── dotnet/
├── deploy/
│   ├── docker/
│   └── kubernetes/
└── docs/
```

---

## 🔑 DEFINIÇÃO DO MVP (Sprint 6)

- ✅ Messaging service (send/receive)
- ✅ Conversation & Contact management
- ✅ Web Chat adapter (primeiro canal)
- ✅ Admin panel com real-time messaging
- ✅ JWT authentication
- ✅ PostgreSQL persistence
- ✅ Docker Compose deployment

---

## 📚 REFERÊNCIAS

- [LINKTOR-PROJECT-SPEC.md](documentos/LINKTOR-PROJECT-SPEC.md)
- [LINKTOR-CHANNEL-ADAPTERS-GUIDE.md](documentos/LINKTOR-CHANNEL-ADAPTERS-GUIDE.md)
- [Chatwoot](https://github.com/chatwoot/chatwoot) - Referência de arquitetura
- [go-whatsapp-web-multidevice](https://github.com/aldinokemal/go-whatsapp-web-multidevice) - Template WhatsApp
