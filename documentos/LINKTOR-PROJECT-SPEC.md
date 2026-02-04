# LINKTOR - Plataforma Multichannel Messaging Open Source

**Powered by msgfy engine**

> Plataforma multichannel messaging B2B, open source, extensível via plugins, com SDKs multiplataforma e admin self-hosted.

---

## 📋 ÍNDICE

1. [Visão Geral](#visão-geral)
2. [Arquitetura Técnica](#arquitetura-técnica)
3. [Monorepo Structure](#monorepo-structure)
4. [Core Services](#core-services)
5. [msgfy Engine](#msgfy-engine)
6. [SDKs Multiplataforma](#sdks-multiplataforma)
7. [CLI Tools](#cli-tools)
8. [Admin Panel](#admin-panel)
9. [Sistema de Plugins](#sistema-de-plugins)
10. [Channel Adapters](#channel-adapters)
11. [Database Schema](#database-schema)
12. [CI/CD Pipeline](#cicd-pipeline)
13. [Deployment](#deployment)
14. [Contribution Guidelines](#contribution-guidelines)
15. [Roadmap de Implementação](#roadmap-de-implementação)

---

## 🎯 VISÃO GERAL

### Stack Tecnológico

**Backend:**
- Go 1.22+ (core services)
- NATS JetStream (message broker)
- Redis (cache, sessions, rate limiting)
- PostgreSQL 16 (primary database)
- gRPC + REST APIs

**Frontend:**
- React 19
- TypeScript 5.3+
- Vite
- TanStack Query v5
- shadcn/ui
- Tailwind CSS

**DevOps:**
- Docker + Docker Compose
- Kubernetes (production)
- GitHub Actions
- Buf.build (Protobuf)
- Trivy (security scanning)

### Branding Strategy

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

### Licenciamento

**Core (Apache 2.0):**
- msgfy engine
- Channel adapters básicos (WhatsApp oficial, SMS, Web Chat, Telegram)
- SDKs e CLI
- API Gateway

**Enterprise (Proprietário):**
- Advanced analytics
- Multi-tenant cell management
- SSO/SAML
- SLA monitoring
- Suporte prioritário

---

## 🏗️ ARQUITETURA TÉCNICA

### Diagrama de Alto Nível

```
┌─────────────────────────────────────────────────────────────────┐
│                         EXTERNAL CLIENTS                        │
│  Web Dashboard │ Mobile Apps │ Third-party APIs │ CLI Tools    │
└────────────┬────────────────────────────────────────────────────┘
             │
             ▼
┌─────────────────────────────────────────────────────────────────┐
│                        API GATEWAY (Go)                         │
│  REST + gRPC │ Auth │ Rate Limiting │ Request Routing          │
└────────────┬────────────────────────────────────────────────────┘
             │
             ▼
┌─────────────────────────────────────────────────────────────────┐
│                      CORE SERVICES (Go)                         │
├─────────────────────────────────────────────────────────────────┤
│  Messaging Service  │  Conversation Service │  Contact Service  │
│  Channel Service    │  Tenant Service       │  Analytics Service│
└────────────┬────────────────────────────────────────────────────┘
             │
             ▼
┌─────────────────────────────────────────────────────────────────┐
│                    MESSAGE BROKER (NATS JetStream)              │
│  Subjects: msg.inbound.{channel} │ msg.outbound.{tenant}       │
└────────────┬────────────────────────────────────────────────────┘
             │
             ▼
┌─────────────────────────────────────────────────────────────────┐
│                      CHANNEL ADAPTERS (Plugins)                 │
│  WhatsApp Official │ WhatsApp Unofficial │ SMS │ RCS │ Telegram │
│  Web Chat │ Voice │ Instagram │ Facebook Messenger             │
└─────────────────────────────────────────────────────────────────┘
             │
             ▼
┌─────────────────────────────────────────────────────────────────┐
│                    EXTERNAL CHANNEL APIS                        │
│  Meta API │ Twilio │ Telegram │ Zenvia │ Custom                │
└─────────────────────────────────────────────────────────────────┘

┌─────────────────────────────────────────────────────────────────┐
│                     DATA LAYER                                  │
├─────────────────────────────────────────────────────────────────┤
│  PostgreSQL 16     │  Redis Cluster    │  Object Storage (S3)  │
│  (Primary DB)      │  (Cache/Sessions) │  (Media files)        │
└─────────────────────────────────────────────────────────────────┘
```

### Modular Monolith → Microservices Path

**Fase 1 (MVP): Modular Monolith**
```
linktor-server/
  ├── cmd/server/          # Single binary
  ├── internal/
  │   ├── messaging/       # Bounded context
  │   ├── conversation/    # Bounded context
  │   ├── contacts/        # Bounded context
  │   ├── channels/        # Bounded context
  │   ├── tenants/         # Bounded context
  │   └── analytics/       # Bounded context
  └── pkg/                 # Shared utilities
```

**Fase 2 (Scale): Microservices**
Cada bounded context vira serviço independente quando necessário.

---

## 📁 MONOREPO STRUCTURE

```
linktor/
├── .github/
│   └── workflows/
│       ├── ci.yml
│       ├── release.yml
│       ├── docker.yml
│       └── deploy.yml
│
├── services/                    # Core services em Go
│   ├── api-gateway/
│   │   ├── cmd/
│   │   ├── internal/
│   │   ├── proto/
│   │   ├── Dockerfile
│   │   └── go.mod
│   │
│   ├── messaging/
│   │   ├── cmd/
│   │   ├── internal/
│   │   │   ├── domain/         # Entities
│   │   │   ├── repository/     # Data access
│   │   │   ├── service/        # Business logic
│   │   │   └── handlers/       # gRPC/REST handlers
│   │   ├── proto/
│   │   ├── migrations/
│   │   ├── Dockerfile
│   │   └── go.mod
│   │
│   ├── conversation/
│   ├── contacts/
│   ├── channels/
│   ├── tenants/
│   └── analytics/
│
├── plugins/                     # Channel adapters
│   ├── whatsapp-official/
│   │   ├── main.go
│   │   ├── adapter.go
│   │   └── go.mod
│   │
│   ├── whatsapp-baileys/       # WhatsApp não oficial
│   ├── sms-twilio/
│   ├── rcs/
│   ├── telegram/
│   ├── webchat/
│   ├── voice/
│   ├── instagram/
│   └── facebook-messenger/
│
├── msgfy/                       # Core engine/SDK
│   ├── core/                    # Core library Go
│   │   ├── client/
│   │   ├── adapters/
│   │   ├── types/
│   │   └── go.mod
│   │
│   ├── sdk-go/
│   ├── sdk-java/
│   ├── sdk-python/
│   ├── sdk-typescript/
│   ├── sdk-dotnet/
│   └── cli/                     # msgfy CLI tool
│       ├── cmd/
│       ├── internal/
│       └── go.mod
│
├── web/                         # Frontend apps
│   ├── admin/                   # Admin Panel React 19
│   │   ├── src/
│   │   │   ├── components/
│   │   │   ├── pages/
│   │   │   ├── hooks/
│   │   │   ├── lib/
│   │   │   └── main.tsx
│   │   ├── package.json
│   │   ├── vite.config.ts
│   │   └── tsconfig.json
│   │
│   └── docs/                    # Documentation site
│       └── docusaurus/
│
├── proto/                       # Shared Protobuf definitions
│   ├── messaging/
│   │   └── v1/
│   │       ├── messaging.proto
│   │       └── types.proto
│   ├── conversation/
│   ├── contacts/
│   ├── channels/
│   └── buf.yaml
│
├── infra/                       # Infrastructure as Code
│   ├── docker/
│   │   ├── docker-compose.yml
│   │   ├── docker-compose.dev.yml
│   │   └── docker-compose.prod.yml
│   │
│   ├── k8s/                     # Kubernetes manifests
│   │   ├── base/
│   │   ├── overlays/
│   │   │   ├── dev/
│   │   │   ├── staging/
│   │   │   └── prod/
│   │   └── kustomization.yaml
│   │
│   └── terraform/               # Cloud infrastructure
│       ├── aws/
│       ├── gcp/
│       └── modules/
│
├── scripts/                     # Development scripts
│   ├── setup.sh
│   ├── build-all.sh
│   ├── test-all.sh
│   ├── generate-proto.sh
│   └── migrate.sh
│
├── docs/                        # Documentation
│   ├── architecture/
│   ├── api/
│   ├── plugins/
│   ├── deployment/
│   └── contributing/
│
├── .gitignore
├── LICENSE
├── README.md
├── CONTRIBUTING.md
└── SECURITY.md
```

---

## 🔧 CORE SERVICES

### 1. API Gateway

**Responsabilidades:**
- Autenticação e autorização (JWT)
- Rate limiting (Redis token bucket)
- Request routing para services
- Protocol translation (REST ↔ gRPC)
- CORS handling
- API versioning

**Stack:**
```go
// services/api-gateway/internal/server/server.go
package server

import (
    "github.com/gin-gonic/gin"
    "github.com/linktor/msgfy/core/auth"
    "github.com/linktor/msgfy/core/ratelimit"
)

type Server struct {
    router        *gin.Engine
    authService   *auth.Service
    rateLimiter   *ratelimit.RateLimiter
    messagingConn *grpc.ClientConn
    // ... outros clients gRPC
}

func (s *Server) setupRoutes() {
    v1 := s.router.Group("/api/v1")
    v1.Use(s.authMiddleware())
    v1.Use(s.rateLimitMiddleware())
    
    // Messaging endpoints
    messages := v1.Group("/messages")
    messages.POST("", s.sendMessage)
    messages.GET("/:id", s.getMessage)
    
    // Conversations endpoints
    conversations := v1.Group("/conversations")
    conversations.GET("", s.listConversations)
    conversations.GET("/:id", s.getConversation)
    
    // Channels endpoints
    channels := v1.Group("/channels")
    channels.GET("", s.listChannels)
    channels.POST("", s.createChannel)
}
```

**Configuração:**
```yaml
# services/api-gateway/config.yaml
server:
  port: 8080
  host: 0.0.0.0
  read_timeout: 30s
  write_timeout: 30s

auth:
  jwt_secret: ${JWT_SECRET}
  jwt_expiry: 24h

rate_limit:
  requests_per_second: 100
  burst: 200

grpc:
  messaging_service: messaging:50051
  conversation_service: conversation:50052
  contact_service: contacts:50053
```

### 2. Messaging Service

**Responsabilidades:**
- Envio/recebimento de mensagens
- Normalização de mensagens entre canais
- Publicação no NATS
- Persistência de mensagens
- Retry logic e DLQ (Dead Letter Queue)

**Domain Model:**
```go
// services/messaging/internal/domain/message.go
package domain

import (
    "time"
    "github.com/google/uuid"
)

type MessageStatus string

const (
    StatusPending   MessageStatus = "pending"
    StatusSent      MessageStatus = "sent"
    StatusDelivered MessageStatus = "delivered"
    StatusRead      MessageStatus = "read"
    StatusFailed    MessageStatus = "failed"
)

type MessageType string

const (
    TypeText     MessageType = "text"
    TypeImage    MessageType = "image"
    TypeVideo    MessageType = "video"
    TypeAudio    MessageType = "audio"
    TypeDocument MessageType = "document"
    TypeLocation MessageType = "location"
    TypeContact  MessageType = "contact"
)

type Message struct {
    ID             uuid.UUID       `json:"id"`
    ConversationID uuid.UUID       `json:"conversation_id"`
    TenantID       uuid.UUID       `json:"tenant_id"`
    ChannelID      uuid.UUID       `json:"channel_id"`
    Direction      string          `json:"direction"` // inbound, outbound
    Type           MessageType     `json:"type"`
    Status         MessageStatus   `json:"status"`
    
    // Sender/Receiver
    FromID         string          `json:"from_id"`
    ToID           string          `json:"to_id"`
    
    // Content
    Content        MessageContent  `json:"content"`
    
    // Metadata
    Metadata       map[string]any  `json:"metadata"`
    ExternalID     string          `json:"external_id"` // ID no canal externo
    
    // Timestamps
    CreatedAt      time.Time       `json:"created_at"`
    SentAt         *time.Time      `json:"sent_at"`
    DeliveredAt    *time.Time      `json:"delivered_at"`
    ReadAt         *time.Time      `json:"read_at"`
}

type MessageContent struct {
    Text        string            `json:"text,omitempty"`
    MediaURL    string            `json:"media_url,omitempty"`
    MediaType   string            `json:"media_type,omitempty"`
    Caption     string            `json:"caption,omitempty"`
    Location    *Location         `json:"location,omitempty"`
    Contact     *Contact          `json:"contact,omitempty"`
    Buttons     []Button          `json:"buttons,omitempty"`
}

type Location struct {
    Latitude  float64 `json:"latitude"`
    Longitude float64 `json:"longitude"`
    Address   string  `json:"address,omitempty"`
}

type Button struct {
    ID    string `json:"id"`
    Title string `json:"title"`
    Type  string `json:"type"` // reply, url, phone
    Value string `json:"value"`
}
```

**Service Layer:**
```go
// services/messaging/internal/service/messaging_service.go
package service

import (
    "context"
    "github.com/nats-io/nats.go"
    "github.com/linktor/messaging/internal/domain"
    "github.com/linktor/messaging/internal/repository"
)

type MessagingService struct {
    repo        repository.MessageRepository
    nats        *nats.Conn
    jetstream   nats.JetStreamContext
}

func (s *MessagingService) SendMessage(ctx context.Context, msg *domain.Message) error {
    // 1. Validate message
    if err := s.validateMessage(msg); err != nil {
        return err
    }
    
    // 2. Save to database
    if err := s.repo.Create(ctx, msg); err != nil {
        return err
    }
    
    // 3. Publish to NATS
    subject := fmt.Sprintf("msg.outbound.%s.%s", msg.TenantID, msg.ChannelID)
    payload, _ := json.Marshal(msg)
    
    _, err := s.jetstream.Publish(subject, payload)
    if err != nil {
        // Mark as failed and retry later
        s.markAsFailed(ctx, msg.ID, err)
        return err
    }
    
    return nil
}

func (s *MessagingService) HandleInbound(ctx context.Context, msg *domain.Message) error {
    // 1. Normalize message (já vem normalizado do adapter)
    
    // 2. Find or create conversation
    conv, err := s.findOrCreateConversation(ctx, msg)
    if err != nil {
        return err
    }
    msg.ConversationID = conv.ID
    
    // 3. Save message
    if err := s.repo.Create(ctx, msg); err != nil {
        return err
    }
    
    // 4. Notify via WebSocket (publish to NATS for websocket service)
    s.publishWebSocketNotification(msg)
    
    return nil
}
```

### 3. Conversation Service

**Domain Model:**
```go
// services/conversation/internal/domain/conversation.go
package domain

type Conversation struct {
    ID             uuid.UUID       `json:"id"`
    TenantID       uuid.UUID       `json:"tenant_id"`
    ChannelID      uuid.UUID       `json:"channel_id"`
    ContactID      uuid.UUID       `json:"contact_id"`
    AssignedTo     *uuid.UUID      `json:"assigned_to"` // User ID
    Status         string          `json:"status"` // open, closed, archived
    
    // Metadata
    Subject        string          `json:"subject"`
    Tags           []string        `json:"tags"`
    Priority       string          `json:"priority"`
    LastMessageAt  time.Time       `json:"last_message_at"`
    
    // Counts
    MessageCount   int             `json:"message_count"`
    UnreadCount    int             `json:"unread_count"`
    
    CreatedAt      time.Time       `json:"created_at"`
    UpdatedAt      time.Time       `json:"updated_at"`
    ClosedAt       *time.Time      `json:"closed_at"`
}
```

### 4. Contact Service

**Domain Model:**
```go
// services/contacts/internal/domain/contact.go
package domain

type Contact struct {
    ID         uuid.UUID         `json:"id"`
    TenantID   uuid.UUID         `json:"tenant_id"`
    
    // Identity across channels
    Identities []ContactIdentity `json:"identities"`
    
    // Profile
    Name       string            `json:"name"`
    Email      string            `json:"email"`
    Phone      string            `json:"phone"`
    Avatar     string            `json:"avatar"`
    
    // Custom fields
    CustomFields map[string]any  `json:"custom_fields"`
    
    // Metadata
    Tags       []string          `json:"tags"`
    Segments   []string          `json:"segments"`
    
    CreatedAt  time.Time         `json:"created_at"`
    UpdatedAt  time.Time         `json:"updated_at"`
}

type ContactIdentity struct {
    ChannelID    uuid.UUID `json:"channel_id"`
    ChannelType  string    `json:"channel_type"` // whatsapp, telegram, etc
    ExternalID   string    `json:"external_id"`  // Phone number, username, etc
}
```

### 5. Channel Service

**Domain Model:**
```go
// services/channels/internal/domain/channel.go
package domain

type ChannelType string

const (
    ChannelWhatsAppOfficial   ChannelType = "whatsapp_official"
    ChannelWhatsAppUnofficial ChannelType = "whatsapp_unofficial"
    ChannelSMS                ChannelType = "sms"
    ChannelRCS                ChannelType = "rcs"
    ChannelTelegram           ChannelType = "telegram"
    ChannelWebChat            ChannelType = "webchat"
    ChannelVoice              ChannelType = "voice"
    ChannelInstagram          ChannelType = "instagram"
    ChannelFacebookMessenger  ChannelType = "facebook_messenger"
)

type Channel struct {
    ID          uuid.UUID         `json:"id"`
    TenantID    uuid.UUID         `json:"tenant_id"`
    Type        ChannelType       `json:"type"`
    Name        string            `json:"name"`
    Status      string            `json:"status"` // active, inactive, error
    
    // Plugin info
    PluginID    string            `json:"plugin_id"`
    PluginPath  string            `json:"plugin_path"`
    
    // Configuration (encrypted)
    Config      map[string]any    `json:"config"`
    
    // Credentials (encrypted)
    Credentials map[string]string `json:"credentials"`
    
    // Rate limits
    RateLimits  RateLimits        `json:"rate_limits"`
    
    CreatedAt   time.Time         `json:"created_at"`
    UpdatedAt   time.Time         `json:"updated_at"`
}

type RateLimits struct {
    MessagesPerSecond int `json:"messages_per_second"`
    MessagesPerMinute int `json:"messages_per_minute"`
    MessagesPerHour   int `json:"messages_per_hour"`
    MessagesPerDay    int `json:"messages_per_day"`
}
```

### 6. Tenant Service

**Domain Model:**
```go
// services/tenants/internal/domain/tenant.go
package domain

type Tenant struct {
    ID          uuid.UUID      `json:"id"`
    Name        string         `json:"name"`
    Slug        string         `json:"slug"` // URL-friendly
    Status      string         `json:"status"`
    
    // Subscription
    Plan        string         `json:"plan"`
    Limits      TenantLimits   `json:"limits"`
    
    // Settings
    Settings    TenantSettings `json:"settings"`
    
    CreatedAt   time.Time      `json:"created_at"`
    UpdatedAt   time.Time      `json:"updated_at"`
}

type TenantLimits struct {
    MaxChannels      int `json:"max_channels"`
    MaxUsers         int `json:"max_users"`
    MaxMessagesMonth int `json:"max_messages_month"`
    MaxContacts      int `json:"max_contacts"`
}

type TenantSettings struct {
    Timezone        string         `json:"timezone"`
    Locale          string         `json:"locale"`
    BusinessHours   BusinessHours  `json:"business_hours"`
    Webhooks        []Webhook      `json:"webhooks"`
}
```

---

## ⚙️ msgfy ENGINE

Core library compartilhada entre todos os componentes.

### Estrutura

```
msgfy/core/
├── adapters/
│   ├── interface.go         # Interface do adapter
│   └── registry.go          # Registry de adapters
├── auth/
│   ├── jwt.go
│   └── middleware.go
├── config/
│   └── loader.go
├── db/
│   ├── postgres.go
│   └── migrations.go
├── events/
│   ├── nats.go
│   └── types.go
├── ratelimit/
│   ├── redis_limiter.go
│   └── token_bucket.go
├── storage/
│   └── s3.go
└── types/
    ├── message.go
    ├── conversation.go
    └── errors.go
```

### Adapter Interface

```go
// msgfy/core/adapters/interface.go
package adapters

import (
    "context"
    "github.com/linktor/msgfy/core/types"
)

// ChannelAdapter é a interface que todos os plugins devem implementar
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
    
    // Webhook/Polling
    StartReceiving(ctx context.Context, handler MessageHandler) error
    StopReceiving(ctx context.Context) error
    
    // Capabilities
    Capabilities() Capabilities
}

type MessageHandler func(ctx context.Context, msg *types.Message) error

type HealthStatus struct {
    Status      string    `json:"status"` // healthy, degraded, unhealthy
    Message     string    `json:"message"`
    LastChecked time.Time `json:"last_checked"`
}

type SendResult struct {
    ExternalID string    `json:"external_id"`
    Status     string    `json:"status"`
    SentAt     time.Time `json:"sent_at"`
}

type Capabilities struct {
    SupportsText     bool `json:"supports_text"`
    SupportsImages   bool `json:"supports_images"`
    SupportsVideos   bool `json:"supports_videos"`
    SupportsAudio    bool `json:"supports_audio"`
    SupportsFiles    bool `json:"supports_files"`
    SupportsButtons  bool `json:"supports_buttons"`
    SupportsLocation bool `json:"supports_location"`
    SupportsContacts bool `json:"supports_contacts"`
    MaxMediaSizeMB   int  `json:"max_media_size_mb"`
}
```

### Adapter Registry

```go
// msgfy/core/adapters/registry.go
package adapters

import (
    "fmt"
    "sync"
)

var (
    registry = make(map[string]AdapterFactory)
    mu       sync.RWMutex
)

type AdapterFactory func() ChannelAdapter

// Register registra um novo adapter
func Register(name string, factory AdapterFactory) {
    mu.Lock()
    defer mu.Unlock()
    registry[name] = factory
}

// Get retorna um adapter factory
func Get(name string) (AdapterFactory, error) {
    mu.RLock()
    defer mu.RUnlock()
    
    factory, ok := registry[name]
    if !ok {
        return nil, fmt.Errorf("adapter %s not found", name)
    }
    
    return factory, nil
}

// List retorna todos os adapters registrados
func List() []string {
    mu.RLock()
    defer mu.RUnlock()
    
    names := make([]string, 0, len(registry))
    for name := range registry {
        names = append(names, name)
    }
    return names
}
```

---

## 📦 SDKs MULTIPLATAFORMA

### SDK Go

```go
// msgfy/sdk-go/client.go
package msgfy

import (
    "context"
    "google.golang.org/grpc"
)

type Client struct {
    apiKey  string
    baseURL string
    
    messaging    MessagingClient
    conversation ConversationClient
    contacts     ContactsClient
    channels     ChannelsClient
}

func New(apiKey string, opts ...Option) (*Client, error) {
    c := &Client{
        apiKey:  apiKey,
        baseURL: "https://api.linktor.io",
    }
    
    for _, opt := range opts {
        opt(c)
    }
    
    // Initialize gRPC connections
    conn, err := grpc.Dial(c.baseURL, grpc.WithTransportCredentials(...))
    if err != nil {
        return nil, err
    }
    
    c.messaging = NewMessagingClient(conn)
    c.conversation = NewConversationClient(conn)
    c.contacts = NewContactsClient(conn)
    c.channels = NewChannelsClient(conn)
    
    return c, nil
}

// SendMessage envia uma mensagem
func (c *Client) SendMessage(ctx context.Context, req *SendMessageRequest) (*Message, error) {
    return c.messaging.Send(ctx, req)
}

// Example usage:
// client, _ := msgfy.New("sk_live_...")
// msg, _ := client.SendMessage(ctx, &msgfy.SendMessageRequest{
//     ChannelID: "ch_123",
//     To:        "+5544999999999",
//     Text:      "Hello from Linktor!",
// })
```

### SDK Java

```java
// msgfy/sdk-java/src/main/java/io/linktor/msgfy/MsgfyClient.java
package io.linktor.msgfy;

import io.grpc.ManagedChannel;
import io.grpc.ManagedChannelBuilder;
import io.linktor.msgfy.v1.*;

public class MsgfyClient {
    private final String apiKey;
    private final ManagedChannel channel;
    
    private final MessagingServiceGrpc.MessagingServiceBlockingStub messagingStub;
    private final ConversationServiceGrpc.ConversationServiceBlockingStub conversationStub;
    
    public MsgfyClient(String apiKey) {
        this(apiKey, "api.linktor.io:443");
    }
    
    public MsgfyClient(String apiKey, String target) {
        this.apiKey = apiKey;
        this.channel = ManagedChannelBuilder
            .forTarget(target)
            .useTransportSecurity()
            .build();
        
        this.messagingStub = MessagingServiceGrpc.newBlockingStub(channel);
        this.conversationStub = ConversationServiceGrpc.newBlockingStub(channel);
    }
    
    public Message sendMessage(SendMessageRequest request) {
        return messagingStub.sendMessage(request);
    }
    
    // Usage:
    // MsgfyClient client = new MsgfyClient("sk_live_...");
    // Message msg = client.sendMessage(SendMessageRequest.newBuilder()
    //     .setChannelId("ch_123")
    //     .setTo("+5544999999999")
    //     .setText("Hello from Java!")
    //     .build());
}
```

### SDK Python

```python
# msgfy/sdk-python/msgfy/client.py
from typing import Optional
import grpc
from .generated import messaging_pb2, messaging_pb2_grpc

class MsgfyClient:
    def __init__(self, api_key: str, base_url: str = "api.linktor.io:443"):
        self.api_key = api_key
        self.base_url = base_url
        
        credentials = grpc.ssl_channel_credentials()
        self.channel = grpc.secure_channel(base_url, credentials)
        
        self.messaging = messaging_pb2_grpc.MessagingServiceStub(self.channel)
        self.conversation = messaging_pb2_grpc.ConversationServiceStub(self.channel)
    
    def send_message(
        self,
        channel_id: str,
        to: str,
        text: str,
        media_url: Optional[str] = None
    ) -> dict:
        request = messaging_pb2.SendMessageRequest(
            channel_id=channel_id,
            to=to,
            text=text,
            media_url=media_url or ""
        )
        
        response = self.messaging.SendMessage(request)
        return {
            "id": response.id,
            "status": response.status,
            "sent_at": response.sent_at
        }

# Usage:
# client = MsgfyClient("sk_live_...")
# msg = client.send_message(
#     channel_id="ch_123",
#     to="+5544999999999",
#     text="Hello from Python!"
# )
```

### SDK TypeScript

```typescript
// msgfy/sdk-typescript/src/client.ts
import { MessagingServiceClient } from './generated/messaging_grpc_pb';
import { SendMessageRequest, Message } from './generated/messaging_pb';
import * as grpc from '@grpc/grpc-js';

export class MsgfyClient {
  private messagingClient: MessagingServiceClient;
  
  constructor(
    private apiKey: string,
    private baseUrl: string = 'api.linktor.io:443'
  ) {
    const credentials = grpc.credentials.createSsl();
    this.messagingClient = new MessagingServiceClient(baseUrl, credentials);
  }
  
  async sendMessage(params: {
    channelId: string;
    to: string;
    text: string;
    mediaUrl?: string;
  }): Promise<Message.AsObject> {
    const request = new SendMessageRequest();
    request.setChannelId(params.channelId);
    request.setTo(params.to);
    request.setText(params.text);
    if (params.mediaUrl) request.setMediaUrl(params.mediaUrl);
    
    return new Promise((resolve, reject) => {
      this.messagingClient.sendMessage(request, (err, response) => {
        if (err) reject(err);
        else resolve(response!.toObject());
      });
    });
  }
}

// Usage:
// const client = new MsgfyClient('sk_live_...');
// const msg = await client.sendMessage({
//   channelId: 'ch_123',
//   to: '+5544999999999',
//   text: 'Hello from TypeScript!'
// });
```

### SDK .NET

```csharp
// msgfy/sdk-dotnet/Msgfy/MsgfyClient.cs
using Grpc.Net.Client;
using Linktor.Msgfy.V1;

namespace Msgfy
{
    public class MsgfyClient
    {
        private readonly string _apiKey;
        private readonly GrpcChannel _channel;
        private readonly MessagingService.MessagingServiceClient _messagingClient;
        
        public MsgfyClient(string apiKey, string baseUrl = "https://api.linktor.io")
        {
            _apiKey = apiKey;
            _channel = GrpcChannel.ForAddress(baseUrl);
            _messagingClient = new MessagingService.MessagingServiceClient(_channel);
        }
        
        public async Task<Message> SendMessageAsync(SendMessageRequest request)
        {
            var headers = new Metadata
            {
                { "Authorization", $"Bearer {_apiKey}" }
            };
            
            return await _messagingClient.SendMessageAsync(request, headers);
        }
    }
}

// Usage:
// var client = new MsgfyClient("sk_live_...");
// var msg = await client.SendMessageAsync(new SendMessageRequest
// {
//     ChannelId = "ch_123",
//     To = "+5544999999999",
//     Text = "Hello from C#!"
// });
```

---

## 🖥️ CLI TOOLS

### msgfy CLI

CLI tool para gerenciar channels, enviar mensagens e administrar plataforma.

```go
// msgfy/cli/cmd/root.go
package cmd

import (
    "github.com/spf13/cobra"
    "github.com/spf13/viper"
)

var rootCmd = &cobra.Command{
    Use:   "msgfy",
    Short: "Linktor CLI - Manage your multichannel messaging platform",
    Long:  `msgfy is a command-line interface for Linktor platform`,
}

func Execute() error {
    return rootCmd.Execute()
}

func init() {
    cobra.OnInitialize(initConfig)
    
    rootCmd.PersistentFlags().String("api-key", "", "API key for authentication")
    rootCmd.PersistentFlags().String("base-url", "https://api.linktor.io", "Base URL")
    
    viper.BindPFlag("api_key", rootCmd.PersistentFlags().Lookup("api-key"))
    viper.BindPFlag("base_url", rootCmd.PersistentFlags().Lookup("base-url"))
}
```

**Comandos principais:**

```bash
# Channel management
msgfy channel list
msgfy channel create --type whatsapp_official --name "Main WhatsApp"
msgfy channel config set <channel-id> --key api_key --value "xxx"
msgfy channel test <channel-id>
msgfy channel delete <channel-id>

# Messaging
msgfy send --channel <id> --to "+5544999999999" --text "Hello!"
msgfy send --channel <id> --to "+5544999999999" --image "https://example.com/image.jpg"

# Conversations
msgfy conversation list --status open
msgfy conversation show <conversation-id>
msgfy conversation close <conversation-id>

# Contacts
msgfy contact list
msgfy contact create --name "João" --phone "+5544999999999"
msgfy contact import --file contacts.csv

# Config
msgfy config set api-key <key>
msgfy config get api-key
msgfy config list

# Server (para self-hosted)
msgfy server start
msgfy server migrate
msgfy server plugin install whatsapp-unofficial
msgfy server plugin list
```

**Implementação de exemplo:**

```go
// msgfy/cli/cmd/send.go
package cmd

import (
    "context"
    "fmt"
    "github.com/spf13/cobra"
    "github.com/linktor/msgfy/sdk-go"
)

var sendCmd = &cobra.Command{
    Use:   "send",
    Short: "Send a message",
    RunE:  runSend,
}

var (
    channelID string
    to        string
    text      string
    imageURL  string
)

func init() {
    rootCmd.AddCommand(sendCmd)
    
    sendCmd.Flags().StringVar(&channelID, "channel", "", "Channel ID")
    sendCmd.Flags().StringVar(&to, "to", "", "Recipient")
    sendCmd.Flags().StringVar(&text, "text", "", "Message text")
    sendCmd.Flags().StringVar(&imageURL, "image", "", "Image URL")
    
    sendCmd.MarkFlagRequired("channel")
    sendCmd.MarkFlagRequired("to")
}

func runSend(cmd *cobra.Command, args []string) error {
    apiKey := viper.GetString("api_key")
    if apiKey == "" {
        return fmt.Errorf("API key not configured. Run: msgfy config set api-key <key>")
    }
    
    client, err := msgfy.New(apiKey)
    if err != nil {
        return err
    }
    
    req := &msgfy.SendMessageRequest{
        ChannelID: channelID,
        To:        to,
        Text:      text,
        MediaURL:  imageURL,
    }
    
    msg, err := client.SendMessage(context.Background(), req)
    if err != nil {
        return err
    }
    
    fmt.Printf("✓ Message sent successfully\n")
    fmt.Printf("  ID: %s\n", msg.ID)
    fmt.Printf("  Status: %s\n", msg.Status)
    
    return nil
}
```

---

## 🎨 ADMIN PANEL

Admin panel em React 19 para gerenciar toda a plataforma.

### Stack

```json
{
  "dependencies": {
    "react": "^19.0.0",
    "react-dom": "^19.0.0",
    "react-router-dom": "^6.21.0",
    "@tanstack/react-query": "^5.17.0",
    "@tanstack/react-table": "^8.11.0",
    "zustand": "^4.4.7",
    "zod": "^3.22.4",
    "react-hook-form": "^7.49.2",
    "@hookform/resolvers": "^3.3.3",
    "date-fns": "^3.0.6",
    "recharts": "^2.10.3",
    "lucide-react": "^0.303.0"
  },
  "devDependencies": {
    "vite": "^5.0.10",
    "typescript": "^5.3.3",
    "tailwindcss": "^3.4.0",
    "@types/react": "^18.2.47",
    "@types/react-dom": "^18.2.18",
    "eslint": "^8.56.0",
    "prettier": "^3.1.1"
  }
}
```

### Estrutura de Pastas

```
web/admin/
├── src/
│   ├── app/
│   │   ├── App.tsx
│   │   ├── routes.tsx
│   │   └── providers.tsx
│   │
│   ├── pages/
│   │   ├── Dashboard/
│   │   ├── Conversations/
│   │   ├── Contacts/
│   │   ├── Channels/
│   │   ├── Analytics/
│   │   ├── Settings/
│   │   └── Login/
│   │
│   ├── components/
│   │   ├── ui/              # shadcn/ui components
│   │   ├── layout/
│   │   │   ├── Sidebar.tsx
│   │   │   ├── Header.tsx
│   │   │   └── Layout.tsx
│   │   ├── conversation/
│   │   │   ├── ConversationList.tsx
│   │   │   ├── MessageThread.tsx
│   │   │   └── MessageInput.tsx
│   │   └── channel/
│   │       ├── ChannelCard.tsx
│   │       └── ChannelForm.tsx
│   │
│   ├── hooks/
│   │   ├── useAuth.ts
│   │   ├── useConversations.ts
│   │   ├── useMessages.ts
│   │   └── useWebSocket.ts
│   │
│   ├── lib/
│   │   ├── api/
│   │   │   ├── client.ts
│   │   │   ├── messages.ts
│   │   │   ├── conversations.ts
│   │   │   └── channels.ts
│   │   ├── websocket.ts
│   │   └── utils.ts
│   │
│   ├── stores/
│   │   ├── auth.ts
│   │   ├── conversations.ts
│   │   └── ui.ts
│   │
│   ├── types/
│   │   ├── message.ts
│   │   ├── conversation.ts
│   │   └── channel.ts
│   │
│   └── main.tsx
│
├── public/
├── index.html
├── vite.config.ts
├── tailwind.config.js
└── package.json
```

### Componentes Principais

**Layout:**

```tsx
// web/admin/src/components/layout/Layout.tsx
import { Outlet } from 'react-router-dom';
import { Sidebar } from './Sidebar';
import { Header } from './Header';

export function Layout() {
  return (
    <div className="flex h-screen bg-gray-50">
      <Sidebar />
      
      <div className="flex flex-1 flex-col overflow-hidden">
        <Header />
        
        <main className="flex-1 overflow-y-auto p-6">
          <Outlet />
        </main>
      </div>
    </div>
  );
}
```

**Conversas:**

```tsx
// web/admin/src/pages/Conversations/ConversationsPage.tsx
import { useState } from 'react';
import { useConversations } from '@/hooks/useConversations';
import { ConversationList } from '@/components/conversation/ConversationList';
import { MessageThread } from '@/components/conversation/MessageThread';

export function ConversationsPage() {
  const [selectedId, setSelectedId] = useState<string | null>(null);
  const { data: conversations, isLoading } = useConversations();
  
  return (
    <div className="flex h-full gap-4">
      {/* Lista de conversas */}
      <div className="w-80 flex-shrink-0">
        <ConversationList
          conversations={conversations}
          selectedId={selectedId}
          onSelect={setSelectedId}
          loading={isLoading}
        />
      </div>
      
      {/* Thread de mensagens */}
      <div className="flex-1">
        {selectedId ? (
          <MessageThread conversationId={selectedId} />
        ) : (
          <div className="flex h-full items-center justify-center text-gray-400">
            Selecione uma conversa
          </div>
        )}
      </div>
    </div>
  );
}
```

**WebSocket Hook:**

```typescript
// web/admin/src/hooks/useWebSocket.ts
import { useEffect, useCallback } from 'react';
import { useQueryClient } from '@tanstack/react-query';

export function useWebSocket() {
  const queryClient = useQueryClient();
  
  useEffect(() => {
    const ws = new WebSocket('wss://api.linktor.io/ws');
    
    ws.onopen = () => {
      console.log('WebSocket connected');
    };
    
    ws.onmessage = (event) => {
      const data = JSON.parse(event.data);
      
      // Invalidate queries based on event type
      if (data.type === 'message.created') {
        queryClient.invalidateQueries({ queryKey: ['conversations'] });
        queryClient.invalidateQueries({ 
          queryKey: ['messages', data.conversation_id] 
        });
      }
    };
    
    ws.onerror = (error) => {
      console.error('WebSocket error:', error);
    };
    
    return () => {
      ws.close();
    };
  }, [queryClient]);
}
```

**API Client:**

```typescript
// web/admin/src/lib/api/client.ts
import axios from 'axios';

const apiClient = axios.create({
  baseURL: import.meta.env.VITE_API_URL || 'https://api.linktor.io',
  headers: {
    'Content-Type': 'application/json',
  },
});

apiClient.interceptors.request.use((config) => {
  const token = localStorage.getItem('auth_token');
  if (token) {
    config.headers.Authorization = `Bearer ${token}`;
  }
  return config;
});

apiClient.interceptors.response.use(
  (response) => response,
  (error) => {
    if (error.response?.status === 401) {
      localStorage.removeItem('auth_token');
      window.location.href = '/login';
    }
    return Promise.reject(error);
  }
);

export { apiClient };
```

---

## 🔌 SISTEMA DE PLUGINS

Baseado em HashiCorp go-plugin para isolamento e extensibilidade.

### Plugin Interface

```go
// msgfy/core/adapters/plugin.go
package adapters

import (
    "context"
    "github.com/hashicorp/go-plugin"
    "google.golang.org/grpc"
)

// ChannelAdapterPlugin implements plugin.Plugin
type ChannelAdapterPlugin struct {
    plugin.Plugin
    Impl ChannelAdapter
}

func (p *ChannelAdapterPlugin) GRPCServer(broker *plugin.GRPCBroker, s *grpc.Server) error {
    RegisterChannelAdapterServer(s, &GRPCServer{Impl: p.Impl})
    return nil
}

func (p *ChannelAdapterPlugin) GRPCClient(ctx context.Context, broker *plugin.GRPCBroker, c *grpc.ClientConn) (interface{}, error) {
    return &GRPCClient{client: NewChannelAdapterClient(c)}, nil
}
```

### Exemplo: WhatsApp Official Adapter

```go
// plugins/whatsapp-official/main.go
package main

import (
    "context"
    "fmt"
    "github.com/hashicorp/go-plugin"
    "github.com/linktor/msgfy/core/adapters"
    "github.com/linktor/msgfy/core/types"
)

type WhatsAppOfficialAdapter struct {
    accessToken string
    phoneNumberID string
    webhookVerifyToken string
}

func (a *WhatsAppOfficialAdapter) Name() string {
    return "WhatsApp Official"
}

func (a *WhatsAppOfficialAdapter) Type() string {
    return "whatsapp_official"
}

func (a *WhatsAppOfficialAdapter) Version() string {
    return "1.0.0"
}

func (a *WhatsAppOfficialAdapter) Initialize(ctx context.Context, config map[string]any) error {
    // Parse config
    return nil
}

func (a *WhatsAppOfficialAdapter) Connect(ctx context.Context, credentials map[string]string) error {
    a.accessToken = credentials["access_token"]
    a.phoneNumberID = credentials["phone_number_id"]
    a.webhookVerifyToken = credentials["webhook_verify_token"]
    
    // Validate credentials
    return a.validateCredentials(ctx)
}

func (a *WhatsAppOfficialAdapter) SendMessage(ctx context.Context, msg *types.Message) (*adapters.SendResult, error) {
    // Convert to WhatsApp format
    whatsappMsg := a.convertToWhatsAppFormat(msg)
    
    // Send via Meta Cloud API
    resp, err := a.sendToMetaAPI(ctx, whatsappMsg)
    if err != nil {
        return nil, err
    }
    
    return &adapters.SendResult{
        ExternalID: resp.Messages[0].ID,
        Status:     "sent",
        SentAt:     time.Now(),
    }, nil
}

func (a *WhatsAppOfficialAdapter) Capabilities() adapters.Capabilities {
    return adapters.Capabilities{
        SupportsText:     true,
        SupportsImages:   true,
        SupportsVideos:   true,
        SupportsAudio:    true,
        SupportsFiles:    true,
        SupportsButtons:  true,
        SupportsLocation: true,
        SupportsContacts: true,
        MaxMediaSizeMB:   16,
    }
}

func main() {
    plugin.Serve(&plugin.ServeConfig{
        HandshakeConfig: adapters.Handshake,
        Plugins: map[string]plugin.Plugin{
            "channel_adapter": &adapters.ChannelAdapterPlugin{
                Impl: &WhatsAppOfficialAdapter{},
            },
        },
        GRPCServer: plugin.DefaultGRPCServer,
    })
}
```

### Plugin Loader

```go
// services/channels/internal/plugins/loader.go
package plugins

import (
    "context"
    "os/exec"
    "github.com/hashicorp/go-plugin"
    "github.com/linktor/msgfy/core/adapters"
)

type PluginLoader struct {
    pluginsDir string
    loaded     map[string]*plugin.Client
}

func NewPluginLoader(pluginsDir string) *PluginLoader {
    return &PluginLoader{
        pluginsDir: pluginsDir,
        loaded:     make(map[string]*plugin.Client),
    }
}

func (l *PluginLoader) Load(ctx context.Context, pluginPath string) (adapters.ChannelAdapter, error) {
    // Check if already loaded
    if client, ok := l.loaded[pluginPath]; ok {
        raw, err := client.Client()
        if err != nil {
            return nil, err
        }
        return raw.(adapters.ChannelAdapter), nil
    }
    
    // Start plugin process
    client := plugin.NewClient(&plugin.ClientConfig{
        HandshakeConfig: adapters.Handshake,
        Plugins: map[string]plugin.Plugin{
            "channel_adapter": &adapters.ChannelAdapterPlugin{},
        },
        Cmd: exec.Command(pluginPath),
        AllowedProtocols: []plugin.Protocol{
            plugin.ProtocolGRPC,
        },
    })
    
    // Connect via gRPC
    rpcClient, err := client.Client()
    if err != nil {
        return nil, err
    }
    
    // Get plugin
    raw, err := rpcClient.Dispense("channel_adapter")
    if err != nil {
        return nil, err
    }
    
    adapter := raw.(adapters.ChannelAdapter)
    
    // Store loaded client
    l.loaded[pluginPath] = client
    
    return adapter, nil
}

func (l *PluginLoader) Unload(pluginPath string) {
    if client, ok := l.loaded[pluginPath]; ok {
        client.Kill()
        delete(l.loaded, pluginPath)
    }
}
```

---

## 📡 CHANNEL ADAPTERS

### Lista de Adapters

1. **whatsapp-official** - WhatsApp Business Cloud API (Meta)
2. **whatsapp-baileys** - WhatsApp via Baileys (não oficial)
3. **sms-twilio** - SMS via Twilio
4. **rcs** - RCS Business Messaging
5. **telegram** - Telegram Bot API
6. **webchat** - Web Chat (widget embeddable)
7. **voice** - Telefonia VoIP
8. **instagram** - Instagram Messaging API
9. **facebook-messenger** - Facebook Messenger Platform

### Adapter: WhatsApp Unofficial (Baileys)

```go
// plugins/whatsapp-baileys/adapter.go
package main

import (
    "context"
    "encoding/json"
    "github.com/linktor/msgfy/core/adapters"
    "github.com/linktor/msgfy/core/types"
    "os/exec"
)

type BaileysAdapter struct {
    sessionID string
    nodeProcess *exec.Cmd
    messageHandler adapters.MessageHandler
}

func (a *BaileysAdapter) Name() string {
    return "WhatsApp Baileys"
}

func (a *BaileysAdapter) Connect(ctx context.Context, credentials map[string]string) error {
    a.sessionID = credentials["session_id"]
    
    // Start Node.js process with Baileys
    a.nodeProcess = exec.Command("node", "baileys-bridge.js")
    
    stdin, _ := a.nodeProcess.StdinPipe()
    stdout, _ := a.nodeProcess.StdoutPipe()
    
    if err := a.nodeProcess.Start(); err != nil {
        return err
    }
    
    // Send connect command
    connectCmd := map[string]any{
        "command": "connect",
        "session_id": a.sessionID,
    }
    json.NewEncoder(stdin).Encode(connectCmd)
    
    // Start reading messages from Node process
    go a.readMessages(stdout)
    
    return nil
}

func (a *BaileysAdapter) SendMessage(ctx context.Context, msg *types.Message) (*adapters.SendResult, error) {
    // Convert to Baileys format and send via Node process
    // Implementation details...
    return &adapters.SendResult{}, nil
}
```

**Node.js bridge:**

```javascript
// plugins/whatsapp-baileys/baileys-bridge.js
const { default: makeWASocket, DisconnectReason } = require('@whiskeysockets/baileys');
const readline = require('readline');

const rl = readline.createInterface({
  input: process.stdin,
  output: process.stdout,
  terminal: false
});

let sock;

rl.on('line', async (line) => {
  const cmd = JSON.parse(line);
  
  switch (cmd.command) {
    case 'connect':
      sock = makeWASocket({
        auth: {
          sessionId: cmd.session_id
        }
      });
      
      sock.ev.on('messages.upsert', async (m) => {
        const msg = m.messages[0];
        
        // Send to Go via stdout
        console.log(JSON.stringify({
          type: 'message',
          data: {
            from: msg.key.remoteJid,
            text: msg.message?.conversation || '',
            timestamp: msg.messageTimestamp
          }
        }));
      });
      break;
      
    case 'send':
      await sock.sendMessage(cmd.to, { text: cmd.text });
      console.log(JSON.stringify({
        type: 'sent',
        message_id: cmd.id
      }));
      break;
  }
});
```

---

## 🗄️ DATABASE SCHEMA

### PostgreSQL Schema

```sql
-- migrations/001_initial_schema.sql

-- Tenants
CREATE TABLE tenants (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255) NOT NULL,
    slug VARCHAR(100) UNIQUE NOT NULL,
    status VARCHAR(50) DEFAULT 'active',
    plan VARCHAR(50) DEFAULT 'free',
    settings JSONB DEFAULT '{}',
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

-- Users
CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID REFERENCES tenants(id) ON DELETE CASCADE,
    email VARCHAR(255) UNIQUE NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    name VARCHAR(255),
    role VARCHAR(50) DEFAULT 'agent',
    status VARCHAR(50) DEFAULT 'active',
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

-- Channels
CREATE TABLE channels (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID REFERENCES tenants(id) ON DELETE CASCADE,
    type VARCHAR(50) NOT NULL,
    name VARCHAR(255) NOT NULL,
    status VARCHAR(50) DEFAULT 'active',
    plugin_id VARCHAR(100),
    plugin_path VARCHAR(500),
    config JSONB DEFAULT '{}',
    credentials JSONB DEFAULT '{}', -- Encrypted
    rate_limits JSONB DEFAULT '{}',
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

-- Contacts
CREATE TABLE contacts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID REFERENCES tenants(id) ON DELETE CASCADE,
    name VARCHAR(255),
    email VARCHAR(255),
    phone VARCHAR(50),
    avatar TEXT,
    custom_fields JSONB DEFAULT '{}',
    tags TEXT[] DEFAULT '{}',
    segments TEXT[] DEFAULT '{}',
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

CREATE INDEX idx_contacts_tenant ON contacts(tenant_id);
CREATE INDEX idx_contacts_phone ON contacts(phone);
CREATE INDEX idx_contacts_email ON contacts(email);

-- Contact Identities (múltiplos canais)
CREATE TABLE contact_identities (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    contact_id UUID REFERENCES contacts(id) ON DELETE CASCADE,
    channel_id UUID REFERENCES channels(id) ON DELETE CASCADE,
    channel_type VARCHAR(50) NOT NULL,
    external_id VARCHAR(255) NOT NULL,
    created_at TIMESTAMP DEFAULT NOW(),
    UNIQUE(channel_id, external_id)
);

CREATE INDEX idx_identities_contact ON contact_identities(contact_id);
CREATE INDEX idx_identities_channel ON contact_identities(channel_id);

-- Conversations
CREATE TABLE conversations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID REFERENCES tenants(id) ON DELETE CASCADE,
    channel_id UUID REFERENCES channels(id) ON DELETE SET NULL,
    contact_id UUID REFERENCES contacts(id) ON DELETE CASCADE,
    assigned_to UUID REFERENCES users(id) ON DELETE SET NULL,
    status VARCHAR(50) DEFAULT 'open',
    subject VARCHAR(500),
    tags TEXT[] DEFAULT '{}',
    priority VARCHAR(50) DEFAULT 'normal',
    last_message_at TIMESTAMP,
    message_count INT DEFAULT 0,
    unread_count INT DEFAULT 0,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW(),
    closed_at TIMESTAMP
);

CREATE INDEX idx_conversations_tenant ON conversations(tenant_id);
CREATE INDEX idx_conversations_contact ON conversations(contact_id);
CREATE INDEX idx_conversations_assigned ON conversations(assigned_to);
CREATE INDEX idx_conversations_status ON conversations(status);

-- Messages
CREATE TABLE messages (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    conversation_id UUID REFERENCES conversations(id) ON DELETE CASCADE,
    tenant_id UUID REFERENCES tenants(id) ON DELETE CASCADE,
    channel_id UUID REFERENCES channels(id) ON DELETE SET NULL,
    direction VARCHAR(20) NOT NULL, -- inbound, outbound
    type VARCHAR(50) NOT NULL,
    status VARCHAR(50) DEFAULT 'pending',
    from_id VARCHAR(255),
    to_id VARCHAR(255),
    content JSONB NOT NULL,
    metadata JSONB DEFAULT '{}',
    external_id VARCHAR(255),
    created_at TIMESTAMP DEFAULT NOW(),
    sent_at TIMESTAMP,
    delivered_at TIMESTAMP,
    read_at TIMESTAMP
);

CREATE INDEX idx_messages_conversation ON messages(conversation_id);
CREATE INDEX idx_messages_tenant ON messages(tenant_id);
CREATE INDEX idx_messages_created ON messages(created_at DESC);
CREATE INDEX idx_messages_external ON messages(external_id);

-- Message Attachments
CREATE TABLE message_attachments (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    message_id UUID REFERENCES messages(id) ON DELETE CASCADE,
    type VARCHAR(50) NOT NULL, -- image, video, audio, document
    url TEXT NOT NULL,
    size_bytes BIGINT,
    mime_type VARCHAR(100),
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMP DEFAULT NOW()
);

-- Analytics Events
CREATE TABLE analytics_events (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID REFERENCES tenants(id) ON DELETE CASCADE,
    event_type VARCHAR(100) NOT NULL,
    event_data JSONB NOT NULL,
    created_at TIMESTAMP DEFAULT NOW()
);

CREATE INDEX idx_analytics_tenant ON analytics_events(tenant_id);
CREATE INDEX idx_analytics_type ON analytics_events(event_type);
CREATE INDEX idx_analytics_created ON analytics_events(created_at);

-- Webhooks
CREATE TABLE webhooks (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID REFERENCES tenants(id) ON DELETE CASCADE,
    url TEXT NOT NULL,
    events TEXT[] NOT NULL,
    secret VARCHAR(255),
    status VARCHAR(50) DEFAULT 'active',
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

-- Webhook Logs
CREATE TABLE webhook_logs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    webhook_id UUID REFERENCES webhooks(id) ON DELETE CASCADE,
    event_type VARCHAR(100),
    payload JSONB,
    response_status INT,
    response_body TEXT,
    error TEXT,
    created_at TIMESTAMP DEFAULT NOW()
);

-- API Keys
CREATE TABLE api_keys (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID REFERENCES tenants(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    key_hash VARCHAR(255) NOT NULL,
    prefix VARCHAR(20) NOT NULL,
    permissions TEXT[] DEFAULT '{}',
    last_used_at TIMESTAMP,
    expires_at TIMESTAMP,
    created_at TIMESTAMP DEFAULT NOW()
);

CREATE UNIQUE INDEX idx_api_keys_hash ON api_keys(key_hash);
```

### Redis Keys

```
# Rate limiting
ratelimit:{tenant_id}:{channel_id}:{time_window}

# Sessions
session:{session_id}

# Message deduplication
msg:dedup:{channel_id}:{external_id}

# WebSocket connections
ws:conn:{user_id}

# Cache
cache:channel:{channel_id}
cache:conversation:{conversation_id}
cache:contact:{contact_id}
```

---

## 🚀 CI/CD PIPELINE

### GitHub Actions Workflow

```yaml
# .github/workflows/ci.yml
name: CI

on:
  push:
    branches: [ main, develop ]
  pull_request:
    branches: [ main ]

jobs:
  test-go:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      
      - uses: actions/setup-go@v5
        with:
          go-version: '1.22'
      
      - name: Run tests
        run: |
          cd services
          go test -v -race -coverprofile=coverage.out ./...
      
      - name: Upload coverage
        uses: codecov/codecov-action@v3
        with:
          files: ./coverage.out

  lint-go:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      
      - uses: golangci/golangci-lint-action@v3
        with:
          version: latest

  test-frontend:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      
      - uses: actions/setup-node@v4
        with:
          node-version: '20'
      
      - name: Install dependencies
        run: |
          cd web/admin
          npm ci
      
      - name: Run tests
        run: |
          cd web/admin
          npm test
      
      - name: Build
        run: |
          cd web/admin
          npm run build

  security-scan:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      
      - name: Run Trivy vulnerability scanner
        uses: aquasecurity/trivy-action@master
        with:
          scan-type: 'fs'
          scan-ref: '.'
          format: 'sarif'
          output: 'trivy-results.sarif'
      
      - name: Upload Trivy results to GitHub Security
        uses: github/codeql-action/upload-sarif@v2
        with:
          sarif_file: 'trivy-results.sarif'

  build-docker:
    needs: [test-go, test-frontend]
    runs-on: ubuntu-latest
    if: github.event_name == 'push' && github.ref == 'refs/heads/main'
    
    steps:
      - uses: actions/checkout@v4
      
      - name: Set up Docker Buildx
        uses: docker/setup-buildx-action@v3
      
      - name: Login to GitHub Container Registry
        uses: docker/login-action@v3
        with:
          registry: ghcr.io
          username: ${{ github.actor }}
          password: ${{ secrets.GITHUB_TOKEN }}
      
      - name: Build and push API Gateway
        uses: docker/build-push-action@v5
        with:
          context: ./services/api-gateway
          push: true
          tags: |
            ghcr.io/${{ github.repository }}/api-gateway:latest
            ghcr.io/${{ github.repository }}/api-gateway:${{ github.sha }}
      
      # Repeat for other services...
```

### Release Workflow

```yaml
# .github/workflows/release.yml
name: Release

on:
  push:
    tags:
      - 'v*'

jobs:
  release:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      
      - name: Create Release
        uses: actions/create-release@v1
        env:
          GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}
        with:
          tag_name: ${{ github.ref }}
          release_name: Release ${{ github.ref }}
          draft: false
          prerelease: false
      
      - name: Build CLI binaries
        run: |
          cd msgfy/cli
          ./scripts/build-all.sh
      
      - name: Upload CLI binaries
        uses: actions/upload-release-asset@v1
        # ... upload para cada plataforma
```

---

## 📦 DEPLOYMENT

### Docker Compose (Development)

```yaml
# infra/docker/docker-compose.dev.yml
version: '3.8'

services:
  postgres:
    image: postgres:16-alpine
    environment:
      POSTGRES_DB: linktor
      POSTGRES_USER: linktor
      POSTGRES_PASSWORD: linktor_dev
    ports:
      - "5432:5432"
    volumes:
      - postgres_data:/var/lib/postgresql/data

  redis:
    image: redis:7-alpine
    ports:
      - "6379:6379"
    volumes:
      - redis_data:/data

  nats:
    image: nats:2.10-alpine
    command: 
      - "-js"
      - "-m"
      - "8222"
    ports:
      - "4222:4222"
      - "8222:8222"
    volumes:
      - nats_data:/data

  api-gateway:
    build:
      context: ../../services/api-gateway
      dockerfile: Dockerfile
    ports:
      - "8080:8080"
    environment:
      DATABASE_URL: postgres://linktor:linktor_dev@postgres:5432/linktor
      REDIS_URL: redis://redis:6379
      NATS_URL: nats://nats:4222
      JWT_SECRET: dev_secret_change_in_prod
    depends_on:
      - postgres
      - redis
      - nats

  messaging:
    build:
      context: ../../services/messaging
      dockerfile: Dockerfile
    environment:
      DATABASE_URL: postgres://linktor:linktor_dev@postgres:5432/linktor
      NATS_URL: nats://nats:4222
    depends_on:
      - postgres
      - nats

  admin:
    build:
      context: ../../web/admin
      dockerfile: Dockerfile
    ports:
      - "3000:80"
    depends_on:
      - api-gateway

volumes:
  postgres_data:
  redis_data:
  nats_data:
```

### Kubernetes (Production)

```yaml
# infra/k8s/base/api-gateway-deployment.yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: api-gateway
  labels:
    app: api-gateway
spec:
  replicas: 3
  selector:
    matchLabels:
      app: api-gateway
  template:
    metadata:
      labels:
        app: api-gateway
    spec:
      containers:
      - name: api-gateway
        image: ghcr.io/linktor/api-gateway:latest
        ports:
        - containerPort: 8080
        env:
        - name: DATABASE_URL
          valueFrom:
            secretKeyRef:
              name: linktor-secrets
              key: database-url
        - name: REDIS_URL
          valueFrom:
            configMapKeyRef:
              name: linktor-config
              key: redis-url
        - name: NATS_URL
          valueFrom:
            configMapKeyRef:
              name: linktor-config
              key: nats-url
        resources:
          requests:
            memory: "128Mi"
            cpu: "100m"
          limits:
            memory: "256Mi"
            cpu: "500m"
        livenessProbe:
          httpGet:
            path: /health
            port: 8080
          initialDelaySeconds: 30
          periodSeconds: 10
        readinessProbe:
          httpGet:
            path: /ready
            port: 8080
          initialDelaySeconds: 5
          periodSeconds: 5
---
apiVersion: v1
kind: Service
metadata:
  name: api-gateway
spec:
  selector:
    app: api-gateway
  ports:
  - port: 80
    targetPort: 8080
  type: LoadBalancer
```

---

## 🤝 CONTRIBUTION GUIDELINES

### CONTRIBUTING.md

```markdown
# Contribuindo para Linktor

Obrigado por considerar contribuir! Aqui está como você pode ajudar:

## Como Contribuir

### Reportar Bugs

1. Verifique se o bug já foi reportado em [Issues](https://github.com/linktor/linktor/issues)
2. Abra um novo issue com:
   - Descrição clara do problema
   - Passos para reproduzir
   - Comportamento esperado vs atual
   - Screenshots se aplicável
   - Versão do Linktor e ambiente

### Sugerir Features

1. Abra um issue com tag `enhancement`
2. Descreva claramente:
   - Problema que a feature resolve
   - Solução proposta
   - Alternativas consideradas
   - Impacto na API/compatibilidade

### Pull Requests

1. Fork o repositório
2. Crie uma branch: `git checkout -b feature/minha-feature`
3. Commit suas mudanças: `git commit -am 'Add: nova feature'`
4. Push para a branch: `git push origin feature/minha-feature`
5. Abra um Pull Request

#### Convenções de Commit

Usamos [Conventional Commits](https://www.conventionalcommits.org/):

```
feat: adiciona suporte a RCS
fix: corrige rate limiting no WhatsApp
docs: atualiza README com exemplos
style: formata código Go
refactor: simplifica adapter registry
test: adiciona testes para messaging service
chore: atualiza dependências
```

#### Checklist do PR

- [ ] Código segue o style guide
- [ ] Testes adicionados/atualizados
- [ ] Documentação atualizada
- [ ] CHANGELOG.md atualizado
- [ ] Todos os testes passam
- [ ] Sem warnings de linter

### Desenvolvimento Local

```bash
# Clone o repositório
git clone https://github.com/linktor/linktor.git
cd linktor

# Setup ambiente
./scripts/setup.sh

# Start services
docker-compose -f infra/docker/docker-compose.dev.yml up

# Run tests
./scripts/test-all.sh
```

### Criando um Novo Channel Adapter

1. Copie template: `cp -r plugins/template plugins/meu-canal`
2. Implemente `ChannelAdapter` interface
3. Adicione testes
4. Adicione documentação em `docs/plugins/meu-canal.md`
5. Adicione exemplo em `examples/meu-canal/`

### Code Review

Todos os PRs passam por code review. Procuramos:

- Qualidade do código
- Cobertura de testes
- Documentação adequada
- Performance
- Segurança
- Compatibilidade

## Código de Conduta

Leia nosso [CODE_OF_CONDUCT.md](CODE_OF_CONDUCT.md).

## Licença

Ao contribuir, você concorda que suas contribuições serão licenciadas sob Apache 2.0.
```

---

## 📅 ROADMAP DE IMPLEMENTAÇÃO

### Sprint 1-2 (Semanas 1-4): Foundation

**Objetivos:**
- Setup básico do monorepo
- Configuração de CI/CD
- Database schema e migrations
- Core services skeleton

**Deliverables:**
- [ ] Estrutura de monorepo completa
- [ ] Docker Compose funcionando
- [ ] GitHub Actions configurado
- [ ] PostgreSQL schema criado
- [ ] NATS JetStream configurado
- [ ] API Gateway básico (health check, auth)

**Equipe:**
- 1 Backend (Go)
- 1 DevOps

---

### Sprint 3-4 (Semanas 5-8): Core Messaging

**Objetivos:**
- Messaging service completo
- Conversation service
- Plugin system base
- Primeiro adapter (Web Chat)

**Deliverables:**
- [ ] Messaging service com send/receive
- [ ] Conversation management
- [ ] Contact service básico
- [ ] Plugin loader funcionando
- [ ] Web Chat adapter (websocket)
- [ ] msgfy core library

**Equipe:**
- 2 Backend (Go)
- 1 DevOps

---

### Sprint 5-6 (Semanas 9-12): Admin Panel MVP

**Objetivos:**
- Admin panel funcional
- Autenticação
- Visualização de conversas
- Envio de mensagens

**Deliverables:**
- [ ] React 19 app base
- [ ] Autenticação JWT
- [ ] Dashboard
- [ ] Página de conversas
- [ ] Thread de mensagens
- [ ] WebSocket real-time
- [ ] Gerenciamento de canais

**Equipe:**
- 2 Frontend (React)
- 1 Backend (Go) para APIs

---

### Sprint 7-8 (Semanas 13-16): WhatsApp Official

**Objetivos:**
- WhatsApp Official adapter
- Webhook handling
- Media upload/download
- Rate limiting robusto

**Deliverables:**
- [ ] WhatsApp Official adapter completo
- [ ] Webhook receiver
- [ ] Media handling (S3)
- [ ] Template messages
- [ ] Rate limiting com Redis
- [ ] Testes end-to-end

**Equipe:**
- 2 Backend (Go)
- 1 QA

---

### Sprint 9-10 (Semanas 17-20): SDKs

**Objetivos:**
- SDKs para Go, Python, TypeScript
- CLI tool
- Documentação

**Deliverables:**
- [ ] SDK Go
- [ ] SDK Python
- [ ] SDK TypeScript
- [ ] msgfy CLI
- [ ] Exemplos de uso
- [ ] Documentação API

**Equipe:**
- 2 Backend (diferentes linguagens)
- 1 Tech Writer

---

### Sprint 11-12 (Semanas 21-24): Mais Channels

**Objetivos:**
- SMS adapter
- Telegram adapter
- RCS adapter (se viável)

**Deliverables:**
- [ ] SMS via Twilio adapter
- [ ] Telegram Bot API adapter
- [ ] RCS adapter (POC)
- [ ] Instagram adapter (POC)
- [ ] Documentação de cada canal

**Equipe:**
- 3 Backend (Go)
- 1 QA

---

### Sprint 13-14 (Semanas 25-28): Analytics & Enterprise

**Objetivos:**
- Analytics básico
- Webhooks
- Multi-tenant completo

**Deliverables:**
- [ ] Analytics service
- [ ] Métricas básicas
- [ ] Webhooks outbound
- [ ] Multi-tenant isolation
- [ ] Admin de tenants

**Equipe:**
- 2 Backend (Go)
- 1 Frontend (React)

---

### Sprint 15-16 (Semanas 29-32): Beta Release

**Objetivos:**
- Bug fixes
- Performance optimization
- Security audit
- Documentation

**Deliverables:**
- [ ] Testes de carga
- [ ] Security scan
- [ ] Documentation site
- [ ] Beta release
- [ ] Community setup (Discord, GitHub Discussions)

**Equipe:**
- Full team

---

## 🎯 MVP DEFINITION (Sprint 6)

**Must Have:**
- ✅ Messaging service (send/receive)
- ✅ Conversation management
- ✅ Contact management
- ✅ Web Chat adapter
- ✅ Admin panel (conversas, envio)
- ✅ Authentication
- ✅ PostgreSQL persistence
- ✅ Docker Compose deploy

**Nice to Have:**
- ⚠️ WhatsApp Official (pode vir em Sprint 7-8)
- ⚠️ Analytics básico
- ⚠️ Multi-tenant

**Out of Scope (Post-MVP):**
- ❌ SDKs (vem depois)
- ❌ Advanced analytics
- ❌ Cell-based architecture
- ❌ Kubernetes deployment

---

## 📚 PRÓXIMOS PASSOS

1. **Setup inicial:**
   ```bash
   # Criar repositório
   gh repo create linktor/linktor --public --clone
   cd linktor
   
   # Copiar estrutura
   # (usar a estrutura de pastas acima)
   
   # Initial commit
   git add .
   git commit -m "feat: initial project structure"
   git push origin main
   ```

2. **Configurar CI/CD:**
   - GitHub Actions
   - Docker Hub / GHCR
   - Codecov

3. **Setup ambiente dev:**
   - Docker Compose
   - Migrations
   - Seed data

4. **Começar Sprint 1!**

---

## 📞 SUPORTE E COMUNIDADE

- **GitHub Discussions:** https://github.com/linktor/linktor/discussions
- **Discord:** discord.gg/linktor (criar)
- **Docs:** https://docs.linktor.io (criar)
- **Email:** hello@linktor.io

---

**Linktor** - Link all your channels. Powered by msgfy.
Licensed under Apache 2.0
