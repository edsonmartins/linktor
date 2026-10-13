# Linktor

<p align="center">
  <img src="imagens/logo_linktor_texto.png" alt="Linktor Logo" width="280"/>
</p>

<p align="center">
  <strong>Plataforma Open-Source de Mensageria Multicanal com Inteligência Artificial</strong>
</p>

<p align="center">
  <em>Powered by msgfy engine</em>
</p>

<p align="center">
  <a href="#sobre">Sobre</a> •
  <a href="#funcionalidades">Funcionalidades</a> •
  <a href="#arquitetura">Arquitetura</a> •
  <a href="#tecnologias">Tecnologias</a> •
  <a href="#instalação">Instalação</a> •
  <a href="#uso">Uso</a> •
  <a href="#api">API</a> •
  <a href="#sdks">SDKs</a> •
  <a href="#mcp-server">MCP</a> •
  <a href="#documentação">Docs</a> •
  <a href="#plugins">Plugins</a> •
  <a href="#testes">Testes</a> •
  <a href="#contribuição">Contribuição</a>
</p>

---

### ✨ Destaques desta versão

| Recurso | O que é |
|---------|---------|
| ☎️ **Ligações WhatsApp (oficial)** | WhatsApp Business Calling API: sinalização Graph (connect/accept/reject/terminate + SDP) + mídia WebRTC (pion, OPUS) com **gravação em Ogg/Opus** |
| 📞 **Chamadas WhatsApp (não-oficial)** | Voz/vídeo nativas via whatsmeow (codec mlow próprio) + **gravação em WAV** ligável por config |
| 🔘 **Botões & listas no canal não-oficial** | Mensagens interativas native-flow (quick-reply + single-select) que renderizam no WhatsApp Web multi-device |
| 📣 **Campanhas em massa** | Disparo de templates com entrega assíncrona, progresso ao vivo, retry e DLQ |
| 🔐 **RBAC granular** | Papéis customizados por tenant (recurso × ação) com cache no Redis |
| 🎯 **Atribuição + SLA** | Roteamento automático (round-robin/balanceado) + breach de SLA e auto-close |
| ⚡ **Respostas rápidas** | Atalhos `/comando` com placeholders e contador de uso |
| 📜 **Trilha de auditoria** | Registro automático de toda mutação relevante de segurança |
| 🔑 **Criptografia em repouso** | Credenciais cifradas (AES-256-GCM) com rotação de chave sem downtime |
| 🚚 **Entrega unificada** | Um worker para todos os canais — stateless e stateful — com DLQ e rate-limit |

> Histórico completo em [CHANGELOG.md](CHANGELOG.md).

---

## Sobre

**Linktor** é uma plataforma B2B de mensageria multicanal, open source e extensível via plugins, que unifica a comunicação de empresas com seus clientes através de múltiplos canais em uma única interface. O projeto resolve o problema da fragmentação de canais, permitindo que equipes de atendimento gerenciem conversas de WhatsApp, Telegram, SMS, WebChat, Instagram e Facebook Messenger em um só lugar.

### O Problema

Empresas enfrentam dificuldades ao gerenciar múltiplos canais de comunicação:

- **Fragmentação**: Cada canal tem sua própria interface e API
- **Perda de contexto**: Histórico de conversas espalhado em diferentes sistemas
- **Escalabilidade**: Difícil escalar atendimento humano para demandas crescentes
- **Custos**: Manter integrações separadas é caro e complexo
- **Tempo de resposta**: Clientes esperam respostas instantâneas 24/7

### A Solução

Linktor oferece:

- **Inbox Unificada**: Todas as conversas em uma única interface
- **Bots com IA**: Atendimento automatizado com GPT-4o, Claude 4 e modelos locais (Ollama)
- **Knowledge Base**: RAG (Retrieval Augmented Generation) para respostas precisas baseadas em documentos
- **Flow Builder**: Editor visual para criar fluxos conversacionais sem código
- **Escalação Inteligente**: Transição suave de bot para humano quando necessário
- **Analytics**: Métricas de performance, resolução e satisfação
- **Multi-tenant**: Isolamento completo de dados por organização
- **Event Distribution**: Webhook outbound, NATS JetStream e WebSocket em paralelo
- **Object Storage**: MinIO/S3 com presigned URLs para mídia
- **Extensível**: Sistema de plugins para adicionar novos canais

### Branding

```
LINKTOR (linktor.io)
  ├── Plataforma completa
  ├── Documentação oficial
  ├── Cloud hosting (futuro)
  └── Branding corporativo

msgfy (GitHub org: msgfy)
  ├── Core engine open source
  ├── SDKs e bibliotecas
  ├── CLI tools
  └── Community packages
```

---

## Funcionalidades

### Canais Suportados

| Canal | Status | Descrição |
|-------|--------|-----------|
| WhatsApp Business API | ✅ Completo | Integração oficial Meta Cloud API + **Coexistence (SMB)** + **ligações (WhatsApp Business Calling API)** |
| WhatsApp Unofficial | ✅ Completo | whatsmeow (WhatsApp Web Multi-device) — mídia, interativos native-flow, edit/revoke/forward, resolução LID↔PN e **chamadas de voz/vídeo com gravação** |
| WebChat | ✅ Completo | Widget embeddable para websites |
| Telegram | ✅ Completo | Bot API com suporte a mídia |
| SMS | ✅ Completo | Twilio, Vonage, Plivo |
| Email | ✅ Completo | SMTP, SendGrid, SES, Mailgun, Postmark |
| Instagram DM | ✅ Completo | Meta Graph API |
| Facebook Messenger | ✅ Completo | Meta Graph API |
| RCS | ✅ Completo | Google RCS Business Messaging |
| Voice | ✅ Completo | Twilio Voice, Vonage, Amazon Connect, Asterisk, FreeSWITCH |

> **Entrega unificada (outbound):** todos os canais entregam mensagens pelo mesmo
> _worker_ sobre NATS JetStream — com **retry**, **dead-letter queue (DLQ)** e
> **rate-limit por canal**. Canais stateless (WhatsApp Cloud API, Telegram, SMS,
> Facebook, Instagram, RCS, Email) usam um `Sender` por canal construído a partir
> das credenciais; canais stateful (WebChat, WhatsApp não-oficial/whatsmeow)
> entregam pela sessão/conexão viva. Adicionar um canal novo é implementar uma
> única interface `outbound.Factory`.

<p align="center">
  <img src="imagens/outbound-architecture.svg" alt="Arquitetura de entrega outbound unificada" width="720"/>
</p>

### WhatsApp Coexistence (SMB)

O Linktor suporta **WhatsApp Coexistence**, uma feature crítica que permite uso simultâneo do WhatsApp Business App (celular) + Cloud API (plataforma) **no mesmo número de telefone**.

#### Vantagens

- **Onboarding em 5 minutos**: Escaneie o QR code e está conectado, sem migração de número
- **Histórico assistido**: Permite importar arquivos/exportações de conversas quando disponíveis
- **Híbrido App + API**: Vendedor responde VIPs pelo celular, bot automatiza follow-ups
- **Sincronização bidirecional**: Mensagens enviadas pelo App aparecem no Linktor (via Message Echoes)

#### Features Implementadas

| Feature | Status | Descrição |
|---------|--------|-----------|
| Embedded Signup Flow | ✅ | Conexão via QR code do Facebook SDK |
| OAuth Token Exchange | ✅ | Troca código por access token seguro |
| Detecção Coexistence | ✅ | Verifica se número usa Business App |
| Message Echoes | ✅ | Recebe mensagens enviadas pelo App |
| History Import | Parcial | Importação manual por arquivo/export, sem importação automática de 6 meses pela Cloud API |
| Activity Monitoring | ✅ | Monitora regra dos 14 dias |
| Coexistence Status | ✅ | Dashboard com status de sincronização |

#### API Endpoints

```bash
# Iniciar Embedded Signup (OAuth)
POST /api/v1/oauth/whatsapp/embedded-signup/start

# Callback do OAuth
GET /api/v1/oauth/whatsapp/embedded-signup/callback

# Criar canal após OAuth
POST /api/v1/oauth/whatsapp/embedded-signup/create-channel

# Status de Coexistence
GET /api/v1/channels/:id/coexistence-status

# Histórico legado
# A Cloud API não expõe importação automática de 6 meses; use importação manual quando disponível.
```

#### Regra dos 14 Dias

O WhatsApp Business App deve ser aberto **pelo menos 1x a cada 14 dias** para manter o Coexistence ativo. O Linktor monitora automaticamente:

- Alerta quando inativo por 10+ dias
- Verifica status quando chega a 14 dias
- Dashboard widget mostra última atividade do App

#### Limitações do Coexistence

Após ativar Coexistence, algumas features do App são desabilitadas:

- ❌ Broadcast Lists (use templates via API)
- ❌ View Once Media
- ❌ WhatsApp for Windows/WearOS
- ✅ WhatsApp Web e Mac funcionam normalmente

### WhatsApp Não-Oficial (whatsmeow)

Além da Cloud API oficial, o Linktor traz um canal **não-oficial** sobre o
protocolo multi-device do WhatsApp (biblioteca [whatsmeow](https://github.com/tulir/whatsmeow)),
pareado por **QR code** ou **código de telefone**. O adapter vive em
`internal/adapters/whatsapp` e o motor de chamadas em `internal/voip`.

#### Mensageria

| Recurso | Descrição |
|---------|-----------|
| Mídia | Imagem, vídeo, áudio/PTT, documento, sticker, localização — download+decifra automáticos no inbound (com proteção SSRF no fetch por URL) |
| **Interativos native-flow** | Botões de resposta rápida e listas de seleção que **renderizam no WhatsApp Web multi-device**, com fallback automático para texto; respostas mapeadas para `selected_id` |
| **Unwrap de envelopes** | Desembrulho recursivo de mensagens efêmeras, view-once (v1/v2/v2ext), device-sent, editadas e protocol antes de classificar |
| **Edit / revoke / forward** | Edição, apagar-para-todos e encaminhamento de mensagens |
| Reactions, reply, presença | Envio de reações, respostas com citação, indicador de digitação e read-receipts |
| **Resolução LID↔PN** | Contatos com identidade `@lid` (privacidade) são resolvidos para o número de telefone e casam com o histórico, com cache TTL |

#### Chamadas de voz/vídeo (VoIP nativo)

Chamadas nativas do WhatsApp diretamente no canal não-oficial, com implementação
própria do codec de voz do WhatsApp (**mlow**), SRTP e travessia por relay:

- Iniciar/receber chamadas de **áudio e vídeo** (`PlaceCall` / `AcceptCall` / `RejectCall` / `EndCall`)
- Eventos de ciclo de vida (`incoming` / `state` / `ended`) via `SetCallHandler`
- Hook de stream (`SetCallAudioSink`) para processar o áudio do peer **sem interferir na ligação**

#### Gravação de chamadas

Ligável por configuração do canal. A gravação **não interfere na chamada**: o
callback de áudio apenas copia o PCM em memória; o arquivo é escrito no fim da
ligação. Grava em **WAV estéreo** (esquerda = interlocutor, direita = local) e
cai para mono quando a chamada é só de escuta.

| Parâmetro de canal | Valores | Efeito |
|--------------------|---------|--------|
| `record_calls` | `"true"` / `"false"` (padrão desligado) | Liga a gravação das chamadas |
| `recordings_dir` | caminho (padrão `media/recordings`) | Onde os WAVs são gravados |

O evento `ended` carrega o `recording_path` do arquivo gerado.

> **Nota de deploy:** o store de sessão do canal usa SQLite **pure-Go**
> (`modernc.org/sqlite`, sem cgo) — o binário compila com `CGO_ENABLED=0`.

### Ligações no canal oficial (WhatsApp Business Calling API)

Além do canal não-oficial, o Linktor implementa **ligações de voz no canal
oficial** da Meta usando a [WhatsApp Business Calling API](https://developers.facebook.com/documentation/business-messaging/whatsapp/calling).
O pacote vive em `internal/whatsapp/officialcalls` e é acoplado ao adapter
oficial (`internal/adapters/whatsapp_official`).

Diferente do canal não-oficial (codec mlow próprio + relay), a via oficial usa
**WebRTC padrão** (ICE/DTLS/SRTP, OPUS) — a sinalização troca SDP pela Graph API
e a mídia flui direto por [pion/webrtc](https://github.com/pion/webrtc).

#### Sinalização (Graph API)

`POST /{phone_number_id}/calls` com as ações do protocolo — cada uma carregando
o `session {sdp_type, sdp}` quando aplicável:

| Ação | Método | Efeito |
|------|--------|--------|
| `connect` | `Connect` | Inicia chamada de saída (envia SDP offer) |
| `pre_accept` | `PreAccept` | Pré-aceita para acelerar a conexão de mídia |
| `accept` | `Accept` | Atende (envia SDP answer) |
| `reject` | `Reject` | Recusa a chamada |
| `terminate` | `Terminate` | Encerra a chamada |

O recurso de calling é habilitado por `POST /{phone_number_id}/settings`
(`EnableCalling`, best-effort no `Connect` do adapter).

#### Webhook e ciclo de vida

O webhook `calls` é parseado por `ParseWebhookCalls`; o `Gateway` roteia:

- `event=connect` → chega o **SDP offer**; com `auto_answer_calls` ligado o
  gateway negocia o answer e chama `accept` automaticamente (típico de bot/IVR),
  senão a chamada fica pendente até `AcceptCall` / `RejectCall`
- `event=terminate` → encerra a mídia e emite o evento com o `recording_path`

Fases expostas ao host via handler: `ringing` → `connected` → `ended`.

#### Mídia e gravação (WebRTC / Ogg-Opus)

O `CallSession` (pion `PeerConnection` + track OPUS local) negocia o SDP e, quando
`call_recordings_dir` está configurado, grava o RTP recebido **direto em Ogg/Opus
sem decodificar** (pure-Go, sem interferir na chamada). Áudio de saída é injetado
por `WriteAudio`.

| Parâmetro de canal | Valores | Efeito |
|--------------------|---------|--------|
| `enable_calls` | `"true"` / `"false"` (padrão desligado) | Habilita calling no canal oficial |
| `auto_answer_calls` | `"true"` / `"false"` (padrão desligado) | Atende automaticamente chamadas de entrada |
| `call_recordings_dir` | caminho (vazio = sem gravação) | Onde os arquivos Ogg/Opus são gravados |

> **Validação:** a sinalização, o parse de webhook e a mídia WebRTC (loopback
> pion) têm cobertura de testes. O caminho de ponta-a-ponta contra a Meta exige
> um número com calling habilitado e só pode ser confirmado em **teste ao vivo**.

### Core Features

#### 1. Inbox Unificada
- Visualização de todas as conversas em tempo real
- Filtros por canal, status, prioridade e agente
- Busca full-text em mensagens
- Atribuição de conversas para agentes
- Indicadores de mensagens não lidas
- WebSocket para atualizações instantâneas

#### 2. Bots com Inteligência Artificial
- **Múltiplos Provedores**: OpenAI (GPT-4o, GPT-4), Anthropic (Claude 4 Opus/Sonnet), Ollama (modelos locais)
- **Configuração por Bot**: Temperatura, max tokens, system prompt personalizado
- **Regras de Escalação**: Por confiança baixa, sentimento negativo, keywords ou intenção
- **Horário de Funcionamento**: Bots ativos apenas em horários configurados
- **Contexto de Conversa**: Mantém histórico para respostas contextualizadas

#### 3. Knowledge Base (RAG)
- **Tipos**: FAQ, Documentos, Website
- **Embeddings**: Busca semântica com pgvector (OpenAI ada-002)
- **Chunking Inteligente**: Divisão automática de documentos longos
- **Similaridade**: Respostas baseadas em conteúdo mais relevante
- **Regeneração**: Atualização de embeddings sob demanda

#### 4. Flow Builder
- **Editor Visual**: Arrastar e soltar nós no canvas
- **Tipos de Nó**:
  - Message: Envia mensagem ao usuário
  - Question: Faz pergunta com quick replies
  - Condition: Bifurcação baseada em condições
  - Action: Executa ações (tags, HTTP calls)
  - End: Finaliza o fluxo
- **Quick Replies**: Botões de resposta rápida
- **Ações**: Tags, atribuição, escalação, chamadas HTTP
- **Triggers**: Por intent detectado, keyword ou manualmente
- **Teste Integrado**: Simule fluxos antes de publicar

#### 5. Analytics Dashboard
- **Métricas Principais**:
  - Total de conversas
  - Taxa de resolução por bot
  - Tempo médio de resposta
  - Confiança média das respostas
- **Gráficos**:
  - Conversas ao longo do tempo (area chart)
  - Motivos de escalação (pie chart)
- **Performance de Fluxos**: Taxa de conclusão por fluxo
- **Por Canal**: Métricas segmentadas por canal
- **Trends**: Comparação com período anterior

#### 6. Multi-tenancy
- Isolamento completo de dados por tenant
- Planos de assinatura (Free, Starter, Professional, Enterprise)
- Limites configuráveis:
  - Máximo de canais
  - Máximo de usuários
  - Máximo de contatos
  - Mensagens por mês

#### 7. Escalação Inteligente
- Detecção automática de necessidade de humano
- Transferência com contexto completo da conversa
- Histórico de interações com o bot
- Análise de sentimento e intenção
- Fila de atendimento por prioridade

#### 8. LVRE - Linktor Visual Response Engine

O **LVRE** (Linktor Visual Response Engine) é um sistema de renderização que converte HTML em imagens otimizadas para canais que não suportam elementos interativos (como botões ou templates estruturados) - como WhatsApp não-oficial, SMS, ou canais legados.

<p align="center">
  <img src="imagens/lvre-menu.svg" alt="LVRE Menu" width="280"/>
  <img src="imagens/lvre-produto.svg" alt="LVRE Produto" width="280"/>
</p>

<p align="center">
  <img src="imagens/lvre-status.svg" alt="LVRE Status" width="280"/>
  <img src="imagens/lvre-pix.svg" alt="LVRE PIX" width="280"/>
</p>

**Características:**

- **Renderização HTML → Imagem**: Usa headless Chrome (chromedp) para renderizar qualquer HTML
- **Templates Predefinidos**: 6 templates otimizados para WhatsApp (menu, produto, status, lista, confirmação, PIX)
- **Custom HTML**: Desenvolvedores podem passar qualquer HTML para renderização
- **Otimização de Imagem**: WebP por padrão (30% menor), compressão PNG via pngquant
- **Multi-canal**: Configurações otimizadas por canal (WhatsApp, Telegram, Web, Email)
- **Cache Redis**: Imagens renderizadas são cacheadas para performance
- **Caption + Follow-up**: Suporte a texto junto com a imagem e mensagem de acompanhamento
- **Multi-tenant**: Branding customizado por tenant (cores, logo, fontes)

**API Endpoints:**

```bash
# Renderizar HTML customizado
POST /api/v1/vre/render
{
  "html": "<div style='padding:20px'>...</div>",
  "channel": "whatsapp",
  "caption": "Texto junto com a imagem",
  "follow_up_text": "Mensagem enviada após a imagem"
}

# Renderizar template predefinido
POST /api/v1/vre/render
{
  "template_id": "menu_opcoes",
  "data": {
    "titulo": "Como posso ajudar?",
    "opcoes": [
      {"label": "Fazer pedido", "icone": "pedido"},
      {"label": "Rastrear pedido", "icone": "entrega"}
    ]
  },
  "channel": "whatsapp"
}

# Listar templates disponíveis
GET /api/v1/vre/templates

# Preview de template
GET /api/v1/vre/templates/:id/preview

# Configurar branding do tenant
PUT /api/v1/vre/config
```

**Templates Disponíveis:**

| Template | Descrição | Uso |
|----------|-----------|-----|
| `menu_opcoes` | Menu com até 8 opções numeradas | Menus interativos |
| `card_produto` | Card de produto com preço e estoque | Catálogo |
| `status_pedido` | Timeline visual de status | Rastreamento |
| `lista_produtos` | Lista comparativa (até 6 itens) | Busca de produtos |
| `confirmacao` | Resumo para confirmação | Checkout |
| `cobranca_pix` | QR code PIX + código copia-cola | Pagamentos |

#### 9. Campanhas em Massa (Bulk)

Disparo de templates para muitos destinatários, com entrega assíncrona sobre NATS.

<p align="center">
  <img src="imagens/campaign-progress.svg" alt="Tela de progresso de campanha" width="380"/>
</p>

- **Modelo de fila**: a campanha apenas **enfileira** os destinatários; o _worker_ de outbound entrega, faz retry e atualiza o status por destinatário
- **Acompanhamento ao vivo**: contadores de enviadas/entregues/leituras/falhas como projeção do status real (correlacionado pelos webhooks de status)
- **Retry de falhas** e **cancelamento**; _sweep_ periódico reconcilia destinatários presos em `queued`
- **UI**: lista, criação (canal + template + destinatários), tela de progresso ao vivo, filtro por status e paginação

```bash
POST   /api/v1/campaigns                 # criar (rascunho) + destinatários
POST   /api/v1/campaigns/:id/start       # iniciar (enfileira)
POST   /api/v1/campaigns/:id/retry       # reenfileirar falhas
POST   /api/v1/campaigns/:id/cancel      # cancelar
GET    /api/v1/campaigns/:id             # status + contadores
GET    /api/v1/campaigns/:id/recipients  # destinatários (filtro ?status=)
```

#### 10. RBAC Granular (Papéis Customizados)

Controle de acesso por **recurso × ação**, com papéis customizáveis por tenant.

<p align="center">
  <img src="imagens/rbac-matrix.svg" alt="Matriz de permissões recurso×ação" width="380"/>
</p>

- Papéis de sistema (`owner`/`admin`/`supervisor`/`agent`) seedados por tenant + papéis customizados
- Permissões `resource:action` (ex.: `campaigns:create`, `analytics:read`), com `manage` implicando todas as ações
- Resolução com cache no Redis; middleware `RequirePermission`
- **Retrocompatível**: papéis legados (string) resolvem pelos padrões internos
- **UI**: lista de papéis + matriz recurso×ação (papéis de sistema read-only)

```bash
GET    /api/v1/roles/catalog   # recursos e ações disponíveis
GET    /api/v1/roles           # listar (semeia papéis de sistema)
POST   /api/v1/roles           # criar papel customizado
PUT    /api/v1/roles/:id       # editar (permissões)
DELETE /api/v1/roles/:id       # excluir (apenas customizados)
```

#### 11. Atribuição Automática + SLA

Roteamento de conversas e enforcement de SLA por tenant.

<p align="center">
  <img src="imagens/operations-sla.svg" alt="Página de Operações: atribuição e SLA" width="380"/>
</p>

- **Estratégias de atribuição**: `manual`, `round_robin`, `load_balanced` (menos conversas abertas) — aplicado na entrada de mensagens
- **SLA & auto-close**: _ticker_ marca _breach_ de primeira resposta e auto-fecha conversas ociosas conforme os limites configurados
- **UI**: página de Operações (estratégia, auto-assign, limites de SLA/auto-close)

```bash
GET  /api/v1/settings                          # configurações operacionais
PUT  /api/v1/settings                          # atualizar (admin)
POST /api/v1/conversations/:id/auto-assign     # rotear conversa agora
```

#### 12. Respostas Rápidas (Canned Responses)

Respostas reutilizáveis que o agente insere por atalho (`/atalho`), com placeholders e contador de uso.

```bash
GET    /api/v1/canned-responses                # listar (busca ?search=)
POST   /api/v1/canned-responses                # criar
GET    /api/v1/canned-responses/use/:shortcut  # resolver atalho (+1 uso)
PUT    /api/v1/canned-responses/:id            # editar
DELETE /api/v1/canned-responses/:id            # excluir
```

#### 13. Trilha de Auditoria

Registro de toda mutação relevante de segurança.

<p align="center">
  <img src="imagens/audit-log.svg" alt="Trilha de auditoria" width="380"/>
</p>

- Middleware deriva **ação/recurso** da rota (ex.: `channel.create`, `contact.delete`), cobrindo channels/contacts/users/tenant automaticamente
- Handlers especializados (campanhas, papéis, settings, canned) gravam registros mais ricos
- **UI**: trilha read-only paginada com filtros por ação/recurso

```bash
GET /api/v1/audit-logs   # ?action= &resource_type= &page=
```

#### 14. Criptografia de Credenciais em Repouso

Tokens e segredos dos canais são cifrados no banco.

- **AES-256-GCM**, prefixo `enc:v1:` — **retrocompatível** com dados em texto puro
- Chaves não-sensíveis (ex.: `phone_number_id`) permanecem em claro e consultáveis
- **Rotação de chave sem downtime**: chave primária + `crypto.previous_keys` (decrypt com fallback); `POST /api/v1/channels/reencrypt` re-encripta com a chave nova
- Em release, a chave é **obrigatória**, ≥32 chars e distinta do `jwt.secret`

#### 15. Observabilidade da Fila + DLQ

O monitor de fila (Observability) lista os streams JetStream, incluindo a
**`LINKTOR_DLQ`**: mensagens que esgotam as tentativas em qualquer _consumer_
são _dead-lettered_ (em vez de loop ou descarte silencioso) e ficam retidas
para inspeção/replay.

<p align="center">
  <img src="imagens/dlq-monitor.svg" alt="Monitor de fila com a DLQ" width="380"/>
</p>

---

## Arquitetura


### Visão Geral

<p align="center">
  <img src="imagens/infografico_linktor.png" alt="Linktor Architecture" width="100%"/>
</p>

### Padrão Arquitetural: Hexagonal (Ports & Adapters)

O projeto segue a arquitetura hexagonal (também conhecida como Ports and Adapters), separando claramente:

- **Domain Layer**: Entidades, repositórios (interfaces) e regras de negócio puras
- **Application Layer**: Casos de uso e serviços de aplicação
- **Adapter Layer**: Implementações concretas (banco de dados, APIs externas, AI providers)
- **API Layer**: Controllers HTTP, WebSocket e gRPC

### Modular Monolith → Microservices Path

**Fase Atual: Modular Monolith**
- Bounded contexts bem definidos (messaging, conversation, contacts, channels, bots, analytics)
- Comunicação via interfaces internas
- Deploy como binário único

**Futuro: Microservices**
- Cada bounded context pode virar serviço independente
- Comunicação via NATS JetStream
- Escalabilidade horizontal por serviço

### Estrutura de Diretórios

```
linktor/
├── cmd/                          # Entry points
│   ├── server/                   # Servidor principal
│   │   └── main.go               # Bootstrap da aplicação
│   └── cli/                      # CLI tool (msgfy)
│       ├── main.go               # Entry point do CLI
│       ├── cmd/                  # Comandos Cobra
│       │   ├── root.go           # Comando raiz
│       │   ├── auth.go           # Autenticação
│       │   ├── channel.go        # Canais
│       │   ├── send.go           # Enviar mensagens
│       │   ├── conversation.go   # Conversas
│       │   ├── contact.go        # Contatos
│       │   ├── bot.go            # Bots
│       │   ├── flow.go           # Flows
│       │   ├── knowledge.go      # Knowledge bases
│       │   ├── webhook.go        # Webhooks
│       │   └── server.go         # Self-hosted
│       └── internal/client/      # HTTP client
│
├── internal/                     # Código interno (não exportado)
│   ├── domain/                   # Camada de domínio (DDD)
│   │   ├── entity/               # Entidades de negócio
│   │   │   ├── analytics.go      # Métricas e analytics
│   │   │   ├── bot.go            # Configuração de bots
│   │   │   ├── channel.go        # Canais de comunicação
│   │   │   ├── contact.go        # Contatos
│   │   │   ├── conversation.go   # Conversas
│   │   │   ├── escalation.go     # Regras de escalação
│   │   │   ├── flow.go           # Fluxos conversacionais
│   │   │   ├── intent.go         # Intenções detectadas
│   │   │   ├── knowledge.go      # Knowledge base
│   │   │   ├── message.go        # Mensagens
│   │   │   ├── tenant.go         # Multi-tenancy
│   │   │   └── user.go           # Usuários
│   │   ├── repository/           # Interfaces de repositório
│   │   └── valueobject/          # Value objects (event types, etc.)
│   │
│   ├── application/              # Camada de aplicação
│   │   ├── service/              # Serviços de aplicação
│   │   │   ├── ai_provider.go    # Factory de AI providers
│   │   │   ├── analytics.go      # Agregação de métricas
│   │   │   ├── auth.go           # Autenticação JWT
│   │   │   ├── bot.go            # Gerenciamento de bots
│   │   │   ├── conversation_context.go # Contexto de conversa
│   │   │   ├── embedding.go      # Geração de embeddings
│   │   │   ├── flow_engine.go    # Motor de execução de fluxos
│   │   │   ├── intent.go         # Classificação de intenção
│   │   │   ├── knowledge.go      # Busca semântica
│   │   │   └── ...
│   │   └── usecase/              # Casos de uso
│   │       ├── analyze_message.go
│   │       ├── escalate_conversation.go
│   │       ├── generate_ai_response.go
│   │       ├── receive_message.go
│   │       └── send_message.go
│   │
│   ├── api/                      # Camada de API
│   │   ├── handlers/             # HTTP handlers
│   │   │   ├── ai.go             # Endpoints de AI
│   │   │   ├── analytics.go      # Endpoints de analytics
│   │   │   ├── auth.go           # Login, refresh token
│   │   │   ├── bot.go            # CRUD de bots
│   │   │   ├── channel.go        # CRUD de canais
│   │   │   ├── conversation.go   # Gerenciamento de conversas
│   │   │   ├── flow.go           # CRUD de fluxos
│   │   │   ├── knowledge.go      # Knowledge base
│   │   │   ├── webhook.go        # Webhooks de canais
│   │   │   └── websocket.go      # Real-time updates
│   │   └── middleware/           # Middlewares
│   │       ├── auth.go           # JWT validation
│   │       ├── cors.go           # CORS handling
│   │       └── ratelimit.go      # Rate limiting
│   │
│   ├── adapters/                 # Adaptadores externos
│   │   ├── ai/                   # Provedores de IA
│   │   │   ├── openai/           # OpenAI GPT models
│   │   │   ├── anthropic/        # Anthropic Claude
│   │   │   └── ollama/           # Modelos locais
│   │   ├── whatsapp_official/    # Meta Cloud API
│   │   ├── whatsapp/             # WhatsApp Unofficial (Baileys)
│   │   ├── telegram/             # Telegram Bot API
│   │   ├── webchat/              # WebSocket chat
│   │   ├── email/                # Email (SMTP, SendGrid, SES, Mailgun, Postmark)
│   │   ├── sms/                  # SMS providers (Twilio)
│   │   ├── instagram/            # Instagram DM (Meta Graph API)
│   │   ├── facebook/             # Facebook Messenger (Meta Graph API)
│   │   ├── meta/                 # Meta shared client
│   │   ├── rcs/                  # RCS Business Messaging
│   │   └── voice/                # Voice (Twilio, Vonage, Amazon Connect, Asterisk, FreeSWITCH)
│   │   # cada adapter expõe um sender.go (outbound.Factory)
│   │
│   ├── outbound/                 # Entrega unificada (Sender/Factory/Resolver/Worker, DLQ, rate-limit)
│   │
│   └── infrastructure/           # Infraestrutura
│       ├── database/             # PostgreSQL + pgvector
│       │   ├── postgres.go       # Connection pool
│       │   ├── migrations/       # SQL migrations
│       │   └── *_repo.go         # Repository implementations
│       ├── nats/                 # Message broker
│       │   ├── client.go
│       │   ├── producer.go
│       │   └── consumer.go
│       ├── webhook/              # Webhook outbound delivery
│       │   └── producer.go       # HTTP POST with exponential backoff retry
│       ├── events/               # Event distribution system
│       │   └── dispatcher.go     # Multi-producer event dispatcher
│       ├── storage/              # Object storage
│       │   ├── storage.go        # Client interface + LocalClient
│       │   └── minio.go          # MinIO/S3 with presigned URLs
│       ├── redis/                # Cache
│       ├── vre/                  # Visual Response Engine
│       │   ├── renderer.go       # Chrome renderer (chromedp)
│       │   ├── pool.go           # Browser instance pool
│       │   ├── template_registry.go # Template discovery
│       │   ├── template_functions.go # Custom template funcs
│       │   └── caption_generator.go # Accessible captions
│       └── config/               # Configurações
│
├── web/                          # Frontend
│   ├── admin/                    # Dashboard (Next.js 15)
│   │   ├── src/
│   │   │   ├── app/              # App router pages
│   │   │   │   ├── (dashboard)/  # Protected routes
│   │   │   │   │   ├── analytics/
│   │   │   │   │   ├── channels/
│   │   │   │   │   ├── contacts/
│   │   │   │   │   ├── conversations/
│   │   │   │   │   ├── dashboard/
│   │   │   │   │   ├── flows/
│   │   │   │   │   ├── knowledge-base/
│   │   │   │   │   └── settings/
│   │   │   │   └── login/
│   │   │   ├── components/       # React components
│   │   │   │   ├── ui/           # shadcn/ui base
│   │   │   │   └── layout/       # Sidebar, Header
│   │   │   ├── hooks/            # Custom hooks
│   │   │   ├── lib/              # Utilities
│   │   │   ├── stores/           # Zustand stores
│   │   │   └── types/            # TypeScript types
│   │   └── package.json
│   └── embed/                    # Widget embeddable
│
├── templates/                    # LVRE HTML Templates
│   ├── default/                  # Templates padrão
│   │   ├── menu_opcoes.html      # Menu interativo
│   │   ├── card_produto.html     # Card de produto
│   │   ├── status_pedido.html    # Timeline de status
│   │   ├── lista_produtos.html   # Lista de produtos
│   │   ├── confirmacao.html      # Confirmação de pedido
│   │   └── cobranca_pix.html     # Cobrança PIX
│   └── tenants/                  # Templates por tenant
│       └── {tenant_id}/
│           └── config.json       # Branding config
│
├── deploy/                       # Deploy configs
│   ├── docker/                   # Docker & migrations
│   │   ├── migrations/           # SQL migration files
│   │   └── init.sql              # Initial schema
│   └── kubernetes/               # K8s manifests
│
├── proto/                        # Protocol Buffers
│   ├── channel/
│   ├── conversation/
│   ├── message/
│   └── tenant/
│
├── sdks/                         # SDKs multiplataforma
│   ├── go/                       # SDK Go
│   ├── typescript/               # SDK TypeScript/JavaScript
│   ├── python/                   # SDK Python
│   ├── java/                     # SDK Java
│   ├── dotnet/                   # SDK .NET/C#
│   ├── rust/                     # SDK Rust
│   └── php/                      # SDK PHP
│
├── pkg/                          # Packages públicos
│   ├── errors/                   # Error handling
│   ├── logger/                   # Structured logging
│   └── plugin/                   # Plugin system
│
├── documentos/                   # Documentação técnica
│   ├── LINKTOR-PROJECT-SPEC.md   # Especificação completa
│   └── LINKTOR-CHANNEL-ADAPTERS-GUIDE.md
│
├── docker-compose.yml            # Ambiente local
├── Makefile                      # Build automation
├── config.yaml                   # Configurações
├── go.mod                        # Go modules
└── README.md
```

---

## Tecnologias

### Backend

| Tecnologia | Versão | Uso |
|------------|--------|-----|
| Go | 1.24+ | Linguagem principal |
| Gin | 1.9+ | Framework HTTP |
| PostgreSQL | 16 | Banco de dados principal |
| pgvector | 0.5+ | Busca vetorial para RAG |
| Redis | 7 | Cache e rate limiting |
| NATS JetStream | 2.10 | Message broker assíncrono |
| MinIO | - | Object storage (S3-compatible) |
| JWT | - | Autenticação stateless |
| gRPC | - | Comunicação inter-serviços |
| HashiCorp go-plugin | - | Sistema de plugins |
| chromedp | - | LVRE: Renderização HTML → Imagem |
| pngquant | - | LVRE: Compressão de PNG (opcional) |

### Frontend

| Tecnologia | Versão | Uso |
|------------|--------|-----|
| Next.js | 15.1 | Framework React |
| React | 19 | UI Library |
| TypeScript | 5.7 | Tipagem estática |
| Tailwind CSS | 3.4 | Estilização utility-first |
| Radix UI | - | Componentes acessíveis |
| Zustand | 5.0 | State management |
| React Query | 5.62 | Data fetching e cache |
| Recharts | 3.7 | Gráficos e visualizações |
| React Flow | - | Flow builder visual |
| React Hook Form | 7.54 | Formulários |
| Zod | 3.24 | Validação de schemas |

### AI Providers

| Provider | Modelos | Uso |
|----------|---------|-----|
| OpenAI | GPT-4o, GPT-4 | Geração de respostas |
| Anthropic | Claude 4 Opus/Sonnet | Geração de respostas |
| Ollama | Llama, Mistral, etc | Modelos locais |
| OpenAI | text-embedding-ada-002 | Embeddings para RAG |

### DevOps

| Tecnologia | Uso |
|------------|-----|
| Docker | Containerização |
| Docker Compose | Ambiente local |
| Kubernetes | Orquestração (produção) |
| GitHub Actions | CI/CD |
| golangci-lint | Linting Go |
| Buf | Gerenciamento de Protobuf |
| Trivy | Security scanning |

---

## Instalação

Há dois jeitos de rodar: tudo em Docker (mais rápido para conhecer o sistema) ou infraestrutura em Docker com backend e admin rodando na máquina (para desenvolver).

### Pré-requisitos

- Docker e Docker Compose
- Go 1.25+ e Node.js 20+ (só para o modo de desenvolvimento)
- Make (opcional)

### 1. Clone o repositório

```bash
git clone https://github.com/msgfy/linktor.git
cd linktor
```

### 2. Crie o arquivo `.env`

```bash
cp .env.example .env
```

O `.env.example` já funciona como está para desenvolvimento local. O backend lê o `.env` da raiz ao iniciar, e o `docker compose` usa o mesmo arquivo para portas e URLs. A lista completa de variáveis está em [Exemplo de `.env`](#exemplo-de-env).

### 3A. Rodar tudo em Docker

```bash
docker compose up --build
```

Isso sobe:

- **PostgreSQL** (`LINKTOR_POSTGRES_PORT`, padrão `5432`): banco de dados principal
- **Redis** (`LINKTOR_REDIS_PORT`, padrão `6379`): rate limit e deduplicação de webhooks
- **NATS** (`LINKTOR_NATS_PORT`, padrão `4222`; monitoramento em `LINKTOR_NATS_MONITOR_PORT`, padrão `8222`): fila de mensagens
- **MinIO** (`LINKTOR_MINIO_PORT` e `LINKTOR_MINIO_CONSOLE_PORT`, padrões `9000` e `9001`): armazenamento de mídia
- **API** (`LINKTOR_API_PORT`, padrão `8081`): backend Go
- **Admin** (`LINKTOR_ADMIN_PORT`, padrão `3000`): frontend Next.js

Dentro do Compose, banco, Redis, NATS e JWT do backend são fixos no `docker-compose.yml`; do `.env` ele usa as portas, `LINKTOR_BASE_URL`, `LINKTOR_MINIO_*`, `LINKTOR_MEDIA_*`, `SEED_ADMIN_PASSWORD` e `NEXT_PUBLIC_*`.

Se alguma porta já estiver ocupada, ajuste no `.env` antes de subir, mantendo as URLs coerentes com a porta da API:

```bash
LINKTOR_API_PORT=18081
LINKTOR_ADMIN_PORT=13000
LINKTOR_POSTGRES_PORT=15432
LINKTOR_REDIS_PORT=16379
LINKTOR_MINIO_PORT=19000
LINKTOR_MINIO_CONSOLE_PORT=19001
LINKTOR_BASE_URL=http://localhost:18081
NEXT_PUBLIC_API_URL=http://localhost:18081/api/v1
NEXT_PUBLIC_WS_URL=ws://localhost:18081/api/v1/ws
NEXT_PUBLIC_WEBHOOK_BASE_URL=http://localhost:18081
```

### 3B. Rodar em modo de desenvolvimento

Suba só a infraestrutura no Docker:

```bash
docker compose up -d postgres redis nats minio
```

Inicie o backend em um terminal. As migrations rodam sozinhas na inicialização, e fora do modo `release` o primeiro boot cria os dados de demonstração:

```bash
go run ./cmd/server
# ou: make run
```

Inicie o admin em outro terminal:

```bash
cd web/admin
cp .env.example .env.local
npm install
npm run dev
```

O `web/admin/.env.local` precisa de `NEXT_PUBLIC_API_URL` e `NEXT_PUBLIC_WS_URL` apontando para o backend; os valores do exemplo já apontam para `localhost:8081`.

Para rodar os testes do backend: `make test`.

### 4. Acesse o sistema

- **Admin**: http://localhost:3000
- **API**: http://localhost:8081
- **NATS Monitoring**: http://localhost:8222
- **MinIO Console**: http://localhost:9001 (usuário `linktor`, senha `linktor123`)

Se você alterou as portas no `.env`, use esses valores aqui também.

### Credenciais padrão

```
Email: admin@demo.com
Senha: o valor de SEED_ADMIN_PASSWORD (admin123 no .env.example)
```

A senha é definida no primeiro boot, quando o banco ainda está vazio. Sem `SEED_ADMIN_PASSWORD` ela é aleatória e aparece uma única vez no log do backend (`docker compose logs backend | grep "Password"`). Para trocar depois, apague o volume do banco (`docker compose down -v`) e suba de novo.

### Recebendo webhooks dos canais

Provedores como Meta, Telegram, Twilio e Slack só chamam URLs HTTPS públicas. Em desenvolvimento, abra um túnel para a porta da API (ngrok, Cloudflare Tunnel) e coloque a URL do túnel em `BASE_URL`, `LINKTOR_BASE_URL` e `NEXT_PUBLIC_WEBHOOK_BASE_URL`. Depois reinicie o backend e o admin.

### Exemplo de `.env`

Este é o conteúdo do `.env.example`, com todas as variáveis lidas pelo backend, pelo `docker compose` e pelo admin. Para desenvolvimento local basta copiar; as linhas comentadas são opcionais.

```bash
# Linktor — variáveis de ambiente
# Copie para .env (cp .env.example .env). O backend lê este arquivo ao iniciar
# (go run ./cmd/server) e o docker compose o usa para portas e URLs.
# Os valores abaixo funcionam como estão para desenvolvimento local.
# Linhas comentadas são opcionais.

# ── Servidor ────────────────────────────────────────────────────────────────
LINKTOR_SERVER_PORT=8081
LINKTOR_SERVER_HOST=0.0.0.0
# debug | release. Em release: JWT secret e chave de criptografia fortes são
# obrigatórios, webhooks sem segredo são recusados e o seed demo não roda.
LINKTOR_SERVER_MODE=debug
#LINKTOR_SERVER_SHUTDOWN_TIMEOUT=30
LINKTOR_LOG_LEVEL=debug
LINKTOR_LOG_FORMAT=console
# Versão exibida em /health
#LINKTOR_VERSION=dev

# ── URL pública ─────────────────────────────────────────────────────────────
# Origem pública da API. Usada no registro do webhook do Telegram, no OAuth da
# Meta e nas URLs de mídia. Para receber webhooks de provedores, troque pela URL
# HTTPS do túnel (ngrok, Cloudflare Tunnel) ou do servidor.
BASE_URL=http://localhost:8081
# Mesmo valor, lido pelo docker compose e repassado ao container como BASE_URL.
LINKTOR_BASE_URL=http://localhost:8081

# ── Portas publicadas pelo docker compose ───────────────────────────────────
LINKTOR_API_PORT=8081
LINKTOR_ADMIN_PORT=3000
LINKTOR_POSTGRES_PORT=5432
LINKTOR_REDIS_PORT=6379
LINKTOR_NATS_PORT=4222
LINKTOR_NATS_MONITOR_PORT=8222
LINKTOR_MINIO_PORT=9000
LINKTOR_MINIO_CONSOLE_PORT=9001

# ── PostgreSQL ──────────────────────────────────────────────────────────────
LINKTOR_DATABASE_HOST=localhost
LINKTOR_DATABASE_PORT=5432
LINKTOR_DATABASE_USER=linktor
LINKTOR_DATABASE_PASSWORD=linktor
LINKTOR_DATABASE_DATABASE=linktor
LINKTOR_DATABASE_SSL_MODE=disable
#LINKTOR_DATABASE_MAX_OPEN_CONNS=50
#LINKTOR_DATABASE_MAX_IDLE_CONNS=10
#LINKTOR_DATABASE_MAX_LIFETIME=5

# ── Redis ───────────────────────────────────────────────────────────────────
# Rate limit e deduplicação de webhooks.
LINKTOR_REDIS_HOST=localhost
LINKTOR_REDIS_PORT=6379
LINKTOR_REDIS_PASSWORD=
#LINKTOR_REDIS_DB=0
# String de conexão; tem precedência sobre host/porta acima.
#REDIS_URL=redis://localhost:6379

# ── NATS JetStream ──────────────────────────────────────────────────────────
LINKTOR_NATS_URL=nats://localhost:4222
# Réplicas dos streams. Mantenha 1 em servidor único; 3 só em cluster NATS.
#LINKTOR_NATS_STREAM_REPLICAS=1

# ── Autenticação e criptografia ─────────────────────────────────────────────
# Em release: mínimo de 32 caracteres. Gere com: openssl rand -base64 48
LINKTOR_JWT_SECRET=change-me-in-development
#LINKTOR_JWT_ACCESS_TOKEN_TTL=15
#LINKTOR_JWT_REFRESH_TOKEN_TTL=168
# Os tokens vão em cookies HttpOnly. Em http local precisa ser false, senão o
# navegador descarta o cookie; em produção (HTTPS) use true.
LINKTOR_JWT_COOKIE_SECURE=false
#LINKTOR_JWT_COOKIE_DOMAIN=
#LINKTOR_JWT_COOKIE_SAME_SITE=lax
# Chave que cifra as credenciais dos canais em repouso (AES-256-GCM).
# Em release: mínimo de 32 caracteres e diferente do JWT secret.
LINKTOR_CRYPTO_ENCRYPTION_KEY=change-me-in-development-crypto-key

# ── Dados de demonstração ───────────────────────────────────────────────────
# Fora do modo release o primeiro boot cria admin@demo.com e agent@demo.com.
# Sem esta variável a senha é aleatória e aparece uma única vez no log do backend.
SEED_ADMIN_PASSWORD=admin123
# Força o seed demo também em release.
#LINKTOR_SEED_DEMO=true
# Apaga os dados existentes antes de recriar o seed. Destrutivo.
#SEED_RESET_DESTRUCTIVE=true

# ── Armazenamento de mídia ──────────────────────────────────────────────────
# MinIO/S3 para a mídia recebida dos canais. Sem MINIO_ENDPOINT a mídia fica com
# a URL do provedor (que costuma exigir autenticação) ou usa o diretório local.
MINIO_ENDPOINT=localhost:9000
MINIO_ACCESS_KEY=linktor
MINIO_SECRET_KEY=linktor123
MINIO_BUCKET=linktor-media
MINIO_USE_SSL=false
#MINIO_REGION=
# Os mesmos valores para o docker compose, que os repassa ao container.
#LINKTOR_MINIO_ENDPOINT=minio:9000
#LINKTOR_MINIO_ACCESS_KEY=linktor
#LINKTOR_MINIO_SECRET_KEY=linktor123
#LINKTOR_MINIO_BUCKET=linktor-media
#LINKTOR_MINIO_USE_SSL=false
# Alternativa sem MinIO: diretório local.
#MEDIA_UPLOAD_DIR=./uploads/media
#MEDIA_UPLOAD_BASE_URL=
# Base pública das URLs de mídia (padrão: BASE_URL).
#MEDIA_PUBLIC_BASE_URL=
# URLs de mídia assinadas (?exp=&sig=). Sem segredo, saem sem assinatura.
#MEDIA_SIGNING_SECRET=
#MEDIA_URL_TTL=168h
# Data limite para URLs antigas sem assinatura (RFC 3339 ou AAAA-MM-DD).
# Combine com os integradores (VendaX) antes de definir.
#MEDIA_UNSIGNED_UNTIL=
# Equivalentes lidos pelo docker compose.
#LINKTOR_MEDIA_SIGNING_SECRET=
#LINKTOR_MEDIA_URL_TTL=168h
#LINKTOR_MEDIA_UNSIGNED_UNTIL=

# ── Segurança de webhooks, CORS e WebSocket ─────────────────────────────────
# Exige assinatura nos webhooks também em debug (release sempre exige).
#LINKTOR_REQUIRE_WEBHOOK_SECRETS=true
# Origens CORS permitidas, separadas por vírgula. Vazio aceita só localhost.
#LINKTOR_CORS_ALLOWED_ORIGINS=https://app.example.com
# Origens que podem abrir WebSocket (admin e widget do WebChat).
#LINKTOR_WS_ALLOWED_ORIGINS=https://app.example.com
# Aceita somente a lista acima (recusa localhost) e recusa Origin vazio.
#LINKTOR_WS_ENFORCE_ORIGIN=true
#LINKTOR_WS_ALLOW_EMPTY_ORIGIN=false
# Permite webhook de saída para consumidor local com certificado autoassinado.
#LINKTOR_WEBHOOK_INSECURE_LOOPBACK=true

# ── Canais ──────────────────────────────────────────────────────────────────
# Tipos de canal que podem ser criados/conectados, separados por vírgula.
# Sem a variável, todos são permitidos. Valores: webchat, whatsapp,
# whatsapp_official, whatsapp_unofficial, telegram, instagram, facebook, sms,
# rcs, email, voice, teams, slack, mattermost, direto.
#LINKTOR_ENABLED_CHANNEL_TYPES=whatsapp_official,whatsapp_unofficial,telegram,webchat,facebook,instagram,sms,email
# WhatsApp oficial: janela de 24 h e status do template. off | dry_run | enforce
# (padrão dry_run: só registra em log).
#LINKTOR_WA_WINDOW_ENFORCEMENT=dry_run
#LINKTOR_WA_TEMPLATE_ENFORCEMENT=dry_run
# App Secret da Meta usado no Embedded Signup. Para vários apps, use o sufixo
# com o App ID: WHATSAPP_EMBEDDED_SIGNUP_APP_SECRET_<APP_ID>.
#WHATSAPP_EMBEDDED_SIGNUP_APP_SECRET=
# Sobrescreve a URL da Graph API (mock local: make mock-whatsapp-up).
#LINKTOR_GRAPH_API_URL=http://localhost:4010
# Ponte com o VendaX Core. Exige Linktor e Core no mesmo servidor NATS.
#LINKTOR_VENDAX_BRIDGE_ENABLED=true

# ── Provedores de IA (opcionais) ────────────────────────────────────────────
OPENAI_API_KEY=
#OPENAI_ORG_ID=
#OPENAI_DEFAULT_MODEL=
ANTHROPIC_API_KEY=
#ANTHROPIC_DEFAULT_MODEL=
OLLAMA_BASE_URL=
#OLLAMA_DEFAULT_MODEL=

# ── VRE (Visual Response Engine) ────────────────────────────────────────────
#VRE_UPLOAD_DIR=./uploads/vre
#VRE_UPLOAD_BASE_URL=
#VRE_TEMPLATES_PATH=./templates

# ── Admin (Next.js) ─────────────────────────────────────────────────────────
# Lidas pelo docker compose no build da imagem do admin. Rodando o admin fora do
# Docker (npm run dev), coloque estas três em web/admin/.env.local.
NEXT_PUBLIC_API_URL=http://localhost:8081/api/v1
NEXT_PUBLIC_WS_URL=ws://localhost:8081/api/v1/ws
# Base das URLs de webhook mostradas nas telas de canal e do snippet do WebChat.
# Deve ser a origem pública da API (a mesma de BASE_URL).
NEXT_PUBLIC_WEBHOOK_BASE_URL=http://localhost:8081
```

Em vez de variáveis `LINKTOR_*`, o backend também aceita um `config.yaml` na raiz, em `./config` ou em `/etc/linktor`, com as mesmas chaves em formato aninhado (`LINKTOR_DATABASE_HOST` vira `database.host`). Variáveis de ambiente têm precedência sobre o arquivo.

---

## Uso

### Configurando um Bot

1. Acesse **Bots** no menu lateral
2. Clique em **Create Bot**
3. Configure:
   - **Nome**: Nome identificador do bot
   - **Tipo**: customer_service, sales, faq
   - **Provider**: openai, anthropic, ollama
   - **Modelo**: gpt-4, claude-3-opus, llama2, etc.
   - **System Prompt**: Instruções de comportamento
   - **Temperatura**: 0.0 (determinístico) a 1.0 (criativo)
4. Configure regras de escalação:
   - Confiança baixa (< 0.6)
   - Sentimento negativo
   - Keywords específicas
   - Intenções detectadas
5. Atribua canais ao bot
6. Ative o bot

### Criando uma Knowledge Base

1. Acesse **Knowledge Base**
2. Clique em **Create Knowledge Base**
3. Escolha o tipo:
   - **FAQ**: Perguntas e respostas estruturadas
   - **Documents**: Documentos em texto
   - **Website**: Conteúdo de URLs
4. Adicione itens:
   - Para FAQ: pergunta + resposta + keywords
   - Para Documents: título + conteúdo
5. Os embeddings são gerados automaticamente
6. Vincule a knowledge base a um bot

### Criando um Fluxo Conversacional

1. Acesse **Flows**
2. Clique em **Create Flow**
3. Configure:
   - **Nome** e **descrição**
   - **Trigger**:
     - `intent` - Ativa quando detecta intenção específica
     - `keyword` - Ativa com palavras-chave
     - `welcome` - Mensagem inicial
     - `manual` - Ativado manualmente
4. Use o editor visual:
   - Arraste nós da paleta para o canvas
   - Conecte os nós para criar o fluxo
   - Configure cada nó (conteúdo, condições, ações)
5. Teste o fluxo no painel lateral
6. Ative o fluxo

### Visualizando Analytics

1. Acesse **Analytics**
2. Selecione o período:
   - Daily (últimas 24h)
   - Weekly (últimos 7 dias)
   - Monthly (últimos 30 dias)
3. Visualize:
   - **Cards**: Total de conversas, taxa de resolução, tempo de resposta, confiança
   - **Gráfico de área**: Conversas ao longo do tempo
   - **Gráfico de pizza**: Motivos de escalação
   - **Tabela de fluxos**: Performance de cada fluxo
   - **Tabela de canais**: Métricas por canal

---

## API

### Autenticação

Todas as rotas protegidas requerem um JWT no header:

```
Authorization: Bearer <access_token>
```

Para obter um token:

```bash
curl -X POST http://localhost:8081/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email": "admin@demo.com", "password": "admin123"}'
```

Resposta:
```json
{
  "access_token": "eyJhbG...",
  "refresh_token": "eyJhbG...",
  "user": {
    "id": "...",
    "email": "admin@demo.com",
    "name": "Admin",
    "role": "admin"
  }
}
```

### Endpoints Principais

#### Auth
```
POST /api/v1/auth/login          # Login
POST /api/v1/auth/refresh        # Refresh token
GET  /api/v1/me                  # Usuário atual
```

#### Conversations
```
GET  /api/v1/conversations              # Listar conversas
GET  /api/v1/conversations/:id          # Detalhes da conversa
GET  /api/v1/conversations/:id/messages # Mensagens da conversa
POST /api/v1/conversations/:id/messages # Enviar mensagem
```

#### Bots
```
GET    /api/v1/bots              # Listar bots
POST   /api/v1/bots              # Criar bot
GET    /api/v1/bots/:id          # Detalhes do bot
PUT    /api/v1/bots/:id          # Atualizar bot
DELETE /api/v1/bots/:id          # Deletar bot
POST   /api/v1/bots/:id/activate # Ativar bot
POST   /api/v1/bots/:id/deactivate # Desativar bot
POST   /api/v1/bots/:id/channels # Atribuir canal
DELETE /api/v1/bots/:id/channels/:channelId # Remover canal
PUT    /api/v1/bots/:id/config   # Atualizar configuração
POST   /api/v1/bots/:id/test     # Testar bot
```

#### Knowledge Bases
```
GET    /api/v1/knowledge-bases              # Listar KBs
POST   /api/v1/knowledge-bases              # Criar KB
GET    /api/v1/knowledge-bases/:id          # Detalhes
PUT    /api/v1/knowledge-bases/:id          # Atualizar
DELETE /api/v1/knowledge-bases/:id          # Deletar
GET    /api/v1/knowledge-bases/:id/items    # Listar itens
POST   /api/v1/knowledge-bases/:id/items    # Adicionar item
POST   /api/v1/knowledge-bases/:id/items/bulk # Adicionar em lote
GET    /api/v1/knowledge-bases/:id/items/:itemId # Detalhes do item
PUT    /api/v1/knowledge-bases/:id/items/:itemId # Atualizar item
DELETE /api/v1/knowledge-bases/:id/items/:itemId # Deletar item
POST   /api/v1/knowledge-bases/:id/search   # Busca semântica
POST   /api/v1/knowledge-bases/:id/regenerate-embeddings # Regenerar embeddings
```

#### Flows
```
GET    /api/v1/flows                 # Listar fluxos
POST   /api/v1/flows                 # Criar fluxo
GET    /api/v1/flows/:id             # Detalhes
PUT    /api/v1/flows/:id             # Atualizar
DELETE /api/v1/flows/:id             # Deletar
POST   /api/v1/flows/:id/activate    # Ativar
POST   /api/v1/flows/:id/deactivate  # Desativar
POST   /api/v1/flows/:id/test        # Testar
```

#### Analytics
```
GET /api/v1/analytics/overview       # Métricas gerais
GET /api/v1/analytics/conversations  # Conversas por dia
GET /api/v1/analytics/flows          # Métricas de fluxos
GET /api/v1/analytics/escalations    # Motivos de escalação
GET /api/v1/analytics/channels       # Métricas por canal
```

Query parameters para analytics:
```
?period=daily|weekly|monthly
?start_date=2024-01-01
?end_date=2024-01-31
?bot_id=xxx
?channel_id=xxx
```

#### AI
```
GET  /api/v1/ai/providers                  # Listar provedores
GET  /api/v1/ai/providers/:provider/models # Listar modelos
POST /api/v1/ai/complete                   # Completion
POST /api/v1/ai/classify-intent            # Classificar intent
POST /api/v1/ai/analyze-sentiment          # Análise de sentimento
POST /api/v1/ai/generate-response          # Gerar resposta
POST /api/v1/ai/escalate                   # Escalar conversa
```

#### VRE (Visual Response Engine)
```
POST   /api/v1/vre/render                  # Renderizar HTML para imagem
POST   /api/v1/vre/render-and-send         # Renderizar e enviar
GET    /api/v1/vre/templates               # Listar templates
GET    /api/v1/vre/templates/:id/preview   # Preview de template
POST   /api/v1/vre/templates/:id           # Upload de template
GET    /api/v1/vre/config                  # Configuração de branding
PUT    /api/v1/vre/config                  # Atualizar branding
DELETE /api/v1/vre/cache                   # Invalidar cache
```

#### Webhooks
```
POST /api/v1/webhooks/whatsapp/:channelId  # WhatsApp webhook
POST /api/v1/webhooks/telegram/:channelId  # Telegram webhook
POST /api/v1/webhooks/generic/:channelId   # Webhook genérico
```

### WebSocket

Conexão para atualizações em tempo real:

```javascript
const token = 'eyJhbG...'; // JWT token
const ws = new WebSocket(`ws://localhost:8081/api/v1/ws?token=${token}`);

ws.onmessage = (event) => {
  const data = JSON.parse(event.data);

  switch (data.type) {
    case 'new_message':
      // Nova mensagem recebida
      console.log('New message:', data.payload.message);
      break;
    case 'conversation_updated':
      // Conversa atualizada
      console.log('Conversation updated:', data.payload.conversation);
      break;
    case 'typing':
      // Indicador de digitação
      console.log('Typing:', data.payload);
      break;
  }
};
```

---

## SDKs

SDKs oficiais estão disponíveis em 7 linguagens. Todos seguem a mesma estrutura e API.

| SDK | Package | Instalação |
|-----|---------|------------|
| Go | `github.com/linktor/linktor-go` | `go get github.com/linktor/linktor-go` |
| TypeScript | `@linktor/sdk` | `npm install @linktor/sdk` |
| Python | `linktor` | `pip install linktor` |
| Java | `io.linktor:linktor-sdk` | Maven/Gradle |
| .NET | `Linktor.SDK` | `dotnet add package Linktor.SDK` |
| Rust | `linktor` | `cargo add linktor` |
| PHP | `linktor/linktor-php` | `composer require linktor/linktor-php` |

### SDK Go

```go
import linktor "github.com/linktor/linktor-go"

client := linktor.NewClient("lk_live_...")

// Enviar mensagem
msg, err := client.Conversations.SendMessage(ctx, "cv_123", &linktor.SendMessageInput{
    Text: "Hello from Go!",
})

// Listar conversas
convs, err := client.Conversations.List(ctx, &linktor.ListConversationsParams{
    Status: linktor.String("open"),
    Limit:  linktor.Int(20),
})

// WebSocket para eventos em tempo real
ws := client.WebSocket()
ws.OnMessage(func(msg *linktor.Message) {
    fmt.Printf("New message: %s\n", msg.Text)
})
ws.Connect(ctx)
```

### SDK TypeScript

```typescript
import { LinktorClient } from '@linktor/sdk';

const client = new LinktorClient('lk_live_...');

// Enviar mensagem
const msg = await client.conversations.sendMessage('cv_123', {
  text: 'Hello from TypeScript!',
});

// Listar conversas
const convs = await client.conversations.list({
  status: 'open',
  limit: 20,
});

// WebSocket para eventos em tempo real
const ws = client.websocket();
ws.on('message', (msg) => {
  console.log('New message:', msg.text);
});
await ws.connect();
```

### SDK Python

```python
from linktor import LinktorClient

client = LinktorClient("lk_live_...")

# Enviar mensagem
msg = client.conversations.send_message(
    conversation_id="cv_123",
    text="Hello from Python!"
)

# Listar conversas
convs = client.conversations.list(status="open", limit=20)

# WebSocket para eventos em tempo real
ws = client.websocket()

@ws.on_message
def handle_message(msg):
    print(f"New message: {msg.text}")

ws.connect()
```

### SDK Java

```java
import io.linktor.LinktorClient;

LinktorClient client = new LinktorClient("lk_live_...");

// Enviar mensagem
Message msg = client.conversations().sendMessage("cv_123",
    new SendMessageInput().text("Hello from Java!"));

// Listar conversas
PaginatedResponse<Conversation> convs = client.conversations().list(
    new ListConversationsParams().status("open").limit(20));

// WebSocket para eventos em tempo real
LinktorWebSocket ws = client.websocket();
ws.onMessage(msg -> System.out.println("New message: " + msg.getText()));
ws.connect();
```

### SDK .NET

```csharp
using Linktor;

var client = new LinktorClient("lk_live_...");

// Enviar mensagem
var msg = await client.Conversations.SendMessageAsync("cv_123", new SendMessageInput
{
    Text = "Hello from .NET!"
});

// Listar conversas
var convs = await client.Conversations.ListAsync(new ListConversationsParams
{
    Status = "open",
    Limit = 20
});

// WebSocket para eventos em tempo real
var ws = client.WebSocket();
ws.OnMessage += (sender, msg) => Console.WriteLine($"New message: {msg.Text}");
await ws.ConnectAsync();
```

### SDK Rust

```rust
use linktor::LinktorClient;

let client = LinktorClient::new("lk_live_...");

// Enviar mensagem
let msg = client.conversations()
    .send_message("cv_123", SendMessageInput {
        text: Some("Hello from Rust!".to_string()),
        ..Default::default()
    })
    .await?;

// Listar conversas
let convs = client.conversations()
    .list(ListConversationsParams {
        status: Some("open".to_string()),
        limit: Some(20),
        ..Default::default()
    })
    .await?;
```

### SDK PHP

```php
use Linktor\LinktorClient;

$client = new LinktorClient('lk_live_...');

// Enviar mensagem
$msg = $client->conversations->sendMessage('cv_123', [
    'text' => 'Hello from PHP!'
]);

// Listar conversas
$convs = $client->conversations->list([
    'status' => 'open',
    'limit' => 20
]);
```

### MCP Server

O Linktor oferece um servidor MCP (Model Context Protocol) que permite que assistentes de IA como Claude interajam diretamente com a plataforma.

```bash
# Instalar
npm install @linktor/mcp-server

# Executar (modo stdio - para Claude Desktop)
npx @linktor/mcp-server

# Executar (modo HTTP - para playground web)
npm run start:http
```

#### Configuração no Claude Desktop

Adicione ao `claude_desktop_config.json`:

```json
{
  "mcpServers": {
    "linktor": {
      "command": "npx",
      "args": ["-y", "@linktor/mcp-server"],
      "env": {
        "LINKTOR_API_KEY": "sua-api-key",
        "LINKTOR_API_URL": "https://api.linktor.io/api/v1"
      }
    }
  }
}
```

#### Capabilities

| Tipo | Quantidade | Descrição |
|------|------------|-----------|
| Tools | 51 | Conversas, mensagens, contatos, canais, bots, analytics, knowledge, VRE |
| Resources | 6 | Dados estáticos e templates parametrizados |
| Prompts | 4 | customer_support, conversation_summary, draft_response, analyze_sentiment |

#### MCP Playground

Teste as tools MCP interativamente no browser:

```bash
# Iniciar o servidor HTTP
cd mcp/linktor-mcp-server
npm run start:http

# Acessar o playground na documentação
open http://localhost:3002/mcp/playground
```

---

## Documentação

A documentação oficial está disponível em [docs.linktor.io](https://docs.linktor.io) e inclui:

| Seção | Descrição |
|-------|-----------|
| **Getting Started** | Instalação, quick start, autenticação |
| **Channels** | WhatsApp Official, WhatsApp Unofficial, Telegram, SMS, Email, Voice, WebChat, Instagram, Facebook, RCS |
| **AI & Bots** | Configuração de bots, providers, escalação |
| **Flows** | Flow Builder visual, tipos de nós, triggers |
| **Knowledge Base** | RAG, embeddings, busca semântica |
| **SDKs** | TypeScript, Python, Go, Java, Rust, .NET, PHP |
| **MCP** | Model Context Protocol, Playground interativo |
| **API Reference** | Documentação OpenAPI interativa |
| **Self-Hosting** | Docker, Kubernetes |

### Playground Interativo

A documentação inclui um **MCP Playground** que permite testar as tools da API diretamente no browser, sem necessidade de código. O playground:

- Lista automaticamente todas as tools, resources e prompts
- Gera formulários dinâmicos a partir do schema de cada tool
- Executa chamadas em tempo real via JSON-RPC
- Exibe respostas formatadas com tempo de execução

---

### CLI (msgfy)

O CLI oficial permite gerenciar a plataforma via linha de comando:

```bash
# Instalar
go install github.com/linktor/msgfy@latest

# Autenticar
msgfy auth login

# Listar canais
msgfy channel list

# Enviar mensagem
msgfy send --channel ch_abc123 --to "+5544999999999" --text "Hello!"

# Listar conversas
msgfy conv list --status open

# Gerenciar bots
msgfy bot list
msgfy bot start bt_abc123

# Consultar knowledge base
msgfy kb query kb_abc123 "Como resetar senha?"

# Webhook debugging
msgfy webhook listen --port 3000
```

Documentação completa do CLI: [cmd/cli/README.md](cmd/cli/README.md)

---

## Plugins

### Sistema de Plugins

Linktor usa HashiCorp go-plugin para extensibilidade. Cada canal é um plugin que implementa a interface `ChannelAdapter`.

### Interface do Adapter

```go
type ChannelAdapter interface {
    // Metadata
    Name() string
    Type() string
    Version() string

    // Lifecycle
    Initialize(ctx context.Context, config map[string]any) error
    Connect(ctx context.Context, credentials map[string]string) error
    Disconnect(ctx context.Context) error

    // Health
    HealthCheck(ctx context.Context) (*HealthStatus, error)

    // Messaging
    SendMessage(ctx context.Context, msg *types.Message) (*SendResult, error)

    // Receiving
    StartReceiving(ctx context.Context, handler MessageHandler) error
    StopReceiving(ctx context.Context) error

    // Capabilities
    Capabilities() Capabilities
}

type Capabilities struct {
    SupportsText     bool
    SupportsImages   bool
    SupportsVideos   bool
    SupportsAudio    bool
    SupportsFiles    bool
    SupportsButtons  bool
    SupportsLocation bool
    SupportsContacts bool
    MaxMediaSizeMB   int
}
```

### Criando um Novo Adapter

1. Copie o template: `cp -r plugins/template plugins/meu-canal`
2. Implemente a interface `ChannelAdapter`
3. Compile como plugin: `go build -buildmode=plugin`
4. Registre no sistema
5. Adicione documentação

Exemplo simplificado:

```go
package main

import (
    "context"
    "github.com/hashicorp/go-plugin"
    "github.com/msgfy/linktor/pkg/adapters"
)

type MeuCanalAdapter struct {
    config map[string]any
}

func (a *MeuCanalAdapter) Name() string { return "Meu Canal" }
func (a *MeuCanalAdapter) Type() string { return "meu_canal" }
func (a *MeuCanalAdapter) Version() string { return "1.0.0" }

func (a *MeuCanalAdapter) SendMessage(ctx context.Context, msg *types.Message) (*adapters.SendResult, error) {
    // Implementar envio
    return &adapters.SendResult{
        ExternalID: "msg_123",
        Status:     "sent",
    }, nil
}

// ... implementar outros métodos

func main() {
    plugin.Serve(&plugin.ServeConfig{
        HandshakeConfig: adapters.Handshake,
        Plugins: map[string]plugin.Plugin{
            "channel_adapter": &adapters.ChannelAdapterPlugin{
                Impl: &MeuCanalAdapter{},
            },
        },
        GRPCServer: plugin.DefaultGRPCServer,
    })
}
```

---

## Database Schema

### Tabelas Principais

```sql
-- Tenants (Multi-tenancy)
CREATE TABLE tenants (
    id UUID PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    slug VARCHAR(100) UNIQUE,
    plan VARCHAR(50),
    status VARCHAR(50),
    settings JSONB,
    limits JSONB,
    created_at TIMESTAMP,
    updated_at TIMESTAMP
);

-- Users
CREATE TABLE users (
    id UUID PRIMARY KEY,
    tenant_id UUID REFERENCES tenants(id),
    email VARCHAR(255) UNIQUE,
    password_hash VARCHAR(255),
    name VARCHAR(255),
    role VARCHAR(50),
    status VARCHAR(50),
    created_at TIMESTAMP,
    updated_at TIMESTAMP
);

-- Channels
CREATE TABLE channels (
    id UUID PRIMARY KEY,
    tenant_id UUID REFERENCES tenants(id),
    type VARCHAR(50),
    name VARCHAR(255),
    status VARCHAR(50),
    config JSONB,
    created_at TIMESTAMP,
    updated_at TIMESTAMP
);

-- Contacts
CREATE TABLE contacts (
    id UUID PRIMARY KEY,
    tenant_id UUID REFERENCES tenants(id),
    name VARCHAR(255),
    email VARCHAR(255),
    phone VARCHAR(50),
    metadata JSONB,
    tags TEXT[],
    created_at TIMESTAMP,
    updated_at TIMESTAMP
);

-- Conversations
CREATE TABLE conversations (
    id UUID PRIMARY KEY,
    tenant_id UUID REFERENCES tenants(id),
    channel_id UUID REFERENCES channels(id),
    contact_id UUID REFERENCES contacts(id),
    assigned_user_id UUID REFERENCES users(id),
    status VARCHAR(50),
    priority VARCHAR(50),
    metadata JSONB,
    escalated_at TIMESTAMP,
    resolved_at TIMESTAMP,
    first_reply_at TIMESTAMP,
    created_at TIMESTAMP,
    updated_at TIMESTAMP
);

-- Messages
CREATE TABLE messages (
    id UUID PRIMARY KEY,
    conversation_id UUID REFERENCES conversations(id),
    sender_type VARCHAR(50),
    sender_id VARCHAR(255),
    content_type VARCHAR(50),
    content TEXT,
    metadata JSONB,
    status VARCHAR(50),
    created_at TIMESTAMP,
    updated_at TIMESTAMP
);

-- Bots
CREATE TABLE bots (
    id UUID PRIMARY KEY,
    tenant_id UUID REFERENCES tenants(id),
    name VARCHAR(255),
    type VARCHAR(50),
    provider VARCHAR(50),
    model VARCHAR(100),
    config JSONB,
    is_active BOOLEAN,
    created_at TIMESTAMP,
    updated_at TIMESTAMP
);

-- Knowledge Bases
CREATE TABLE knowledge_bases (
    id UUID PRIMARY KEY,
    tenant_id UUID REFERENCES tenants(id),
    name VARCHAR(255),
    description TEXT,
    type VARCHAR(50),
    status VARCHAR(50),
    config JSONB,
    created_at TIMESTAMP,
    updated_at TIMESTAMP
);

-- Knowledge Items (com embeddings pgvector)
CREATE TABLE knowledge_items (
    id UUID PRIMARY KEY,
    knowledge_base_id UUID REFERENCES knowledge_bases(id),
    question TEXT,
    answer TEXT,
    keywords TEXT[],
    embedding vector(1536),
    metadata JSONB,
    created_at TIMESTAMP,
    updated_at TIMESTAMP
);

-- Flows
CREATE TABLE flows (
    id UUID PRIMARY KEY,
    tenant_id UUID REFERENCES tenants(id),
    bot_id UUID REFERENCES bots(id),
    name VARCHAR(255),
    description TEXT,
    trigger VARCHAR(50),
    trigger_value VARCHAR(255),
    start_node_id VARCHAR(100),
    nodes JSONB,
    is_active BOOLEAN,
    priority INTEGER,
    created_at TIMESTAMP,
    updated_at TIMESTAMP
);
```

---

## Roadmap

### Fase 1: Foundation ✅
- [x] Estrutura do monorepo
- [x] CI/CD básico
- [x] Schema do banco de dados
- [x] API REST base
- [x] Autenticação JWT

### Fase 2: Core Messaging ✅
- [x] Serviço de mensagens
- [x] Sistema de plugins
- [x] Adapter WebChat
- [x] Normalização de mensagens
- [x] WebSocket real-time

### Fase 3: Admin Panel ✅
- [x] Dashboard React/Next.js
- [x] Autenticação JWT
- [x] Inbox de conversas
- [x] Gerenciamento de canais
- [x] Gerenciamento de contatos

### Fase 4: WhatsApp Official ✅
- [x] Meta Cloud API integration
- [x] Webhooks
- [x] Templates de mensagem
- [x] Mídia (imagens, documentos)
- [x] **Coexistence (SMB)** - Uso simultâneo App + API no mesmo número
- [x] Embedded Signup Flow (OAuth via QR code)
- [x] Message Echoes (sincronização do App)
- [x] History Import manual por arquivo/export
- [x] Coexistence Activity Monitoring (regra dos 14 dias)

### Fase 5: AI/Chatbot ✅
- [x] Múltiplos provedores de IA (OpenAI, Anthropic, Ollama)
- [x] Gerenciamento de bots
- [x] Knowledge Base com RAG
- [x] Flow Builder visual
- [x] Analytics Dashboard
- [x] Escalação inteligente

### Fase 6: SDKs & CLI ✅
- [x] SDK Go
- [x] SDK TypeScript
- [x] SDK Python
- [x] SDK Java
- [x] SDK .NET
- [x] SDK Rust
- [x] SDK PHP
- [x] CLI tool (msgfy)

### Fase 7: More Channels ✅
- [x] Telegram completo
- [x] SMS (Twilio, Vonage, Plivo)
- [x] Email (SMTP, SendGrid, SES, Mailgun, Postmark)
- [x] Instagram DM
- [x] Facebook Messenger
- [x] RCS
- [x] WhatsApp Unofficial (Baileys)
- [x] Voice (Twilio, Vonage, Amazon Connect, Asterisk, FreeSWITCH)

### Fase 8: Event Distribution & Storage ✅
- [x] Webhook outbound producer com retry exponencial
- [x] MinIO/S3 object storage com presigned URLs
- [x] Event dispatcher multi-producer (NATS, Webhook, WebSocket)
- [x] Event type constants com subscription filtering

### Fase 9: Enterprise 📋
- [ ] Audit logs
- [ ] SSO/SAML
- [ ] Advanced analytics
- [ ] White-label
- [ ] SLA monitoring

---

## Testes

O projeto possui cobertura abrangente de testes automatizados:

| Tipo | Quantidade | Descrição |
|------|------------|-----------|
| **Go unit/integration** | 2,646 | Testes unitários e de integração |
| **Frontend E2E** | 44 | Testes end-to-end com Playwright |
| **Total** | **2,690** | Across 134 test files, 29 packages |

### Executando os testes

```bash
# Todos os testes Go
go test ./... -count=1

# Testes com verbose
go test ./... -v

# Testes por pacote
go test ./internal/adapters/telegram/... -v
go test ./internal/api/handlers/... -v
go test ./internal/application/service/... -v

# Testes do CLI
cd cmd/cli && go test ./... -v

# Testes E2E (frontend)
cd web/admin && npx playwright test

# Testes com coverage
go test ./... -coverprofile=coverage.out
go tool cover -html=coverage.out
```

### Cobertura por camada

| Camada | Pacotes testados | Arquivos de teste |
|--------|-----------------|-------------------|
| Adapters (10 canais + 3 AI) | whatsapp, whatsapp_official, telegram, webchat, email, sms, instagram, facebook, meta, rcs, voice, openai, anthropic, ollama | 40 |
| Handlers (26) | auth, channel, contact, conversation, message, webhook, bot, flow, template, analytics, knowledge, user, tenant, etc. | 20 |
| Services (23) | auth, bot, channel, contact, conversation, message, ai_provider, analytics, knowledge, template, tenant, user, etc. | 16 |
| Use Cases (5) | receive_message, send_message, escalate_conversation, analyze_message, generate_ai_response | 5 |
| Domain entities | contact, conversation, message | 3 |
| Infrastructure | database, nats, storage, vre, webhook, events | 8 |
| Domain | valueobject (event types) | 1 |
| CLI | client, commands, helpers | 5 |
| SDKs | Go, TypeScript | 2 |
| Frontend E2E | auth, dashboard, conversations, channels, contacts, bots, flows, users, knowledge-base, settings, analytics, observability | 12 |

---

## Contribuição

Contribuições são bem-vindas! Por favor, leia nosso guia de contribuição antes de submeter PRs.

### Desenvolvimento Local

```bash
# Instalar ferramentas de desenvolvimento
make install-tools

# Rodar linter
make lint

# Rodar testes
make test

# Rodar testes com coverage
make test-coverage

# Gerar código protobuf
make proto

# Resetar banco de dados
make db-reset
```

### Estrutura de Commits

Seguimos o padrão [Conventional Commits](https://www.conventionalcommits.org/):

```
feat: adiciona novo provider de AI
fix: corrige bug no flow engine
docs: atualiza documentação da API
refactor: reorganiza estrutura de handlers
test: adiciona testes para analytics
chore: atualiza dependências
```

### Checklist do PR

- [ ] Código segue o style guide
- [ ] Testes adicionados/atualizados
- [ ] Documentação atualizada
- [ ] Todos os testes passam
- [ ] Sem warnings de linter

### Criando um Novo Channel Adapter

1. Copie template: `cp -r plugins/template plugins/meu-canal`
2. Implemente `ChannelAdapter` interface
3. Adicione testes
4. Adicione documentação em `docs/plugins/`
5. Adicione exemplo em `examples/`

---

## Licenciamento

### Core (Apache 2.0)
- msgfy engine
- Channel adapters básicos
- SDKs e CLI
- API Gateway
- Admin Panel

### Enterprise (Proprietário)
- Advanced analytics
- Multi-tenant cell management
- SSO/SAML
- SLA monitoring
- Suporte prioritário

---

## Suporte e Comunidade

- **GitHub Issues**: [github.com/msgfy/linktor/issues](https://github.com/msgfy/linktor/issues)
- **GitHub Discussions**: [github.com/msgfy/linktor/discussions](https://github.com/msgfy/linktor/discussions)
- **Discord**: discord.gg/linktor
- **Documentação**: [docs.linktor.io](https://docs.linktor.io)
- **Email**: hello@linktor.io

---

## Agradecimentos

Este projeto foi inspirado e utiliza conceitos de projetos open source como:

- [Chatwoot](https://github.com/chatwoot/chatwoot) - Padrões de channel adapters
- [go-whatsapp-web-multidevice](https://github.com/aldinokemal/go-whatsapp-web-multidevice) - WhatsApp integration

---

<p align="center">
  <strong>Linktor</strong> - Link all your channels.<br/>
  Powered by <strong>msgfy</strong>
</p>

<p align="center">
  Licensed under Apache 2.0
</p>
