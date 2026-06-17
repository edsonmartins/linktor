# Changelog

Todas as mudanças notáveis deste projeto são documentadas aqui.

O formato é baseado em [Keep a Changelog](https://keepachangelog.com/pt-BR/1.1.0/)
e o projeto adere ao [Versionamento Semântico](https://semver.org/lang/pt-BR/).

## [Não lançado]

Features de operação inspiradas na análise do whatomate + subsistema de entrega
outbound que as torna funcionais ponta-a-ponta. Tudo com backend, testes e UI.

### Adicionado

#### Entrega outbound unificada
- Subsistema `internal/outbound`: modelo de conteúdo tipado (Text/Template/Media),
  `Sender`/`Factory` por canal, `Resolver` (channelID → Sender, com cache TTL e
  invalidação) e um **worker NATS genérico** (resolve → envia → ack/nak).
- **Senders stateless** (HTTP) para WhatsApp Cloud API, Telegram, SMS (Twilio),
  Facebook Messenger, Instagram DM, RCS e Email.
- **Caminho stateful** (`PluginSender`) para WebChat e WhatsApp não-oficial
  (whatsmeow), entregando pela sessão/conexão viva no mesmo worker.
- **Dead-letter queue** (`LINKTOR_DLQ`): mensagens que esgotam as tentativas em
  qualquer consumer são dead-lettered (não mais loop ou descarte silencioso) e
  ficam retidas 14 dias para inspeção/replay; visível no monitor de fila.
- Rate-limit por canal e classificação de erro permanent/transient (4xx falha
  rápido; 429/5xx → retry/DLQ).
- Assunto de email configurável via metadata (`subject`/`email_subject`).

#### Funcionalidades de produto
- **Campanhas em massa**: entidade + repositório + serviço (modelo de fila:
  enfileira → worker entrega/retry), correlação de status por destinatário,
  sweep de `queued` órfão, e UI (lista, criação, progresso ao vivo, filtro,
  paginação, retry/cancel).
- **RBAC granular**: papéis customizados por tenant (`resource × action`),
  papéis de sistema seedados, cache no Redis, middleware `RequirePermission`,
  e UI com matriz recurso×ação.
- **Atribuição automática** de conversas (manual/round-robin/load-balanced) e
  **SLA + auto-close** via ticker, com UI de Operações.
- **Respostas rápidas** (canned) com atalho `/comando`, placeholders e contador.
- **Trilha de auditoria** via middleware (deriva ação/recurso da rota) +
  registros explícitos nos handlers especializados, com UI read-only filtrável.

#### Segurança
- **Criptografia de credenciais em repouso** (AES-256-GCM, prefixo `enc:v1:`,
  retrocompatível com texto puro), com validação obrigatória em release.
- **Rotação de chave sem downtime**: chave primária + `crypto.previous_keys`
  (decrypt com fallback) e endpoint `POST /api/v1/channels/reencrypt`.

### Alterado
- `ChannelService.Delete` carrega o canal para o hook `OnDeleted` (invalidação
  de cache do sender), preservando a idempotência do delete.
- Email: config building extraído em `ConfigFromMap`, compartilhado pelo adapter
  e pelo novo sender.

### Testes
- Mocks dos novos repositórios em `pkg/testutil` e testes unitários para
  criptografia (round-trip + rotação), RBAC, tradução/resolver de outbound,
  classificação de erro, plugin sender, e os serviços de campanha, atribuição,
  SLA, papéis e canned.

### Documentação
- README: seções de funcionalidades 9–15, diagramas/mockups SVG e config de
  `crypto`. Novo `CHANGELOG.md`.
