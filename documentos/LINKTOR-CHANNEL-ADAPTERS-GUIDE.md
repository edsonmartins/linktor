# LINKTOR Channel Adapters - Guia de Implementação e Referências

## 🎯 ESTRATÉGIA RECOMENDADA

**Resposta direta à sua pergunta:**

✅ **USAR BIBLIOTECAS EXISTENTES + REFERÊNCIA EM PROJETOS OPEN SOURCE**

**Razões:**
1. **Evitar reinventar a roda** - Bibliotecas maduras já resolveram edge cases complexos
2. **Acelerar desenvolvimento** - Foco em integração, não em protocolo
3. **Manutenção facilitada** - Comunidade mantém compatibilidade com mudanças nas APIs
4. **Código de referência** - Projetos como Chatwoot mostram padrões arquiteturais

**Abordagem híbrida ideal:**
- ✅ Usar bibliotecas Go maduras para cada canal
- ✅ Estudar código-fonte do Chatwoot (Ruby) para entender fluxos
- ✅ Consultar documentação oficial apenas para features específicas
- ✅ Contribuir de volta para as bibliotecas quando encontrar bugs

---

## 📚 PROJETOS OPEN SOURCE DE REFERÊNCIA

### 1. Chatwoot (⭐⭐⭐⭐⭐)

**URL:** https://github.com/chatwoot/chatwoot  
**Linguagem:** Ruby on Rails  
**Estrelas:** 27k+  
**Licença:** MIT (permissiva)

**Por que estudar:**
- Implementa 9+ canais (WhatsApp, FB, IG, Telegram, SMS, Email, Web Chat)
- Arquitetura madura de channel adapters
- Padrões de webhook handling
- Rate limiting strategies
- Message normalization entre canais

**Arquivos-chave para estudar:**

```ruby
# Channel base class
app/models/channel/base.rb

# WhatsApp Cloud API implementation
app/models/channel/whatsapp.rb
app/services/whatsapp/providers/whatsapp_cloud_service.rb

# Telegram
app/models/channel/telegram.rb
lib/integrations/telegram/bot.rb

# SMS (Twilio)
app/models/channel/sms.rb

# Message handling pattern
app/services/channel/inbound_message_handler.rb

# Webhook receiver
app/controllers/api/v1/webhooks/whatsapp_controller.rb
app/controllers/api/v1/webhooks/telegram_controller.rb
```

**Conceitos a extrair:**
- ✅ Padrão de normalização de mensagens
- ✅ Estratégia de polling vs webhook
- ✅ Gerenciamento de sessões
- ✅ Retry logic para falhas
- ✅ Media handling (upload/download)

**Limitação:**
- ❌ Não tem sistema de plugins dinâmico (channels são hard-coded)
- ❌ Ruby não ajuda diretamente (mas conceitos são universais)

---

### 2. go-whatsapp-web-multidevice (⭐⭐⭐⭐)

**URL:** https://github.com/aldinokemal/go-whatsapp-web-multidevice  
**Linguagem:** Go  
**Estrelas:** 2k+  
**Licença:** MIT

**Por que estudar:**
- ✅ **REST API wrapper** completo sobre whatsmeow
- ✅ Multi-account support
- ✅ Webhook integration
- ✅ Chatwoot integration PRONTA
- ✅ Admin UI incluído

**Arquitetura:**

```go
// Estrutura de como eles implementam
services/
├── message.go          // Send messages
├── group.go            // Group management  
├── user.go             // User info
└── webhook.go          // Webhook delivery

internal/
├── rest/               // REST API handlers
├── websocket/          // WebSocket events
└── whatsapp/           // whatsmeow wrapper
    ├── login.go
    ├── send.go
    └── receive.go
```

**Código de referência direto:**

```go
// Como eles fazem send message
func (service messageService) SendText(request whatsapp.MessageRequest) (whatsapp.MessageResponse, error) {
    recipient, _ := whatsapp.FormatPhone(request.Phone)
    
    msg := &waProto.Message{
        Conversation: proto.String(request.Message),
    }
    
    result, err := service.WaCli.SendMessage(
        context.Background(),
        recipient,
        msg,
    )
    
    return whatsapp.MessageResponse{
        MessageID: result.ID,
        Status:    "success",
    }, err
}

// Como recebem mensagens
func (cli *Client) eventHandler(evt interface{}) {
    switch v := evt.(type) {
    case *events.Message:
        // Normalize message
        normalized := cli.normalizeMessage(v)
        
        // Send to webhook
        cli.webhookService.Dispatch(normalized)
        
        // Persist to DB
        cli.messageRepo.Save(normalized)
    }
}
```

**Use este projeto como base para:**
- ✅ Estrutura do WhatsApp Unofficial adapter
- ✅ Padrão de webhook delivery
- ✅ Multi-session management
- ✅ QR code generation/scanning

---

## 📖 BIBLIOTECAS GO RECOMENDADAS POR CANAL

### WhatsApp

#### 1. WhatsApp Oficial (Meta Cloud API)

**Biblioteca:** SDK HTTP padrão + Meta Graph API  
**Documentação:** https://developers.facebook.com/docs/whatsapp/cloud-api

**Não precisa de lib especial:**
```go
// Implementação direta via HTTP
type WhatsAppOfficialAdapter struct {
    httpClient   *http.Client
    accessToken  string
    phoneNumberID string
}

func (a *WhatsAppOfficialAdapter) SendMessage(ctx context.Context, msg *Message) error {
    url := fmt.Sprintf("https://graph.facebook.com/v18.0/%s/messages", a.phoneNumberID)
    
    payload := map[string]interface{}{
        "messaging_product": "whatsapp",
        "to": msg.To,
        "type": "text",
        "text": map[string]string{"body": msg.Text},
    }
    
    req, _ := http.NewRequestWithContext(ctx, "POST", url, jsonPayload(payload))
    req.Header.Set("Authorization", "Bearer " + a.accessToken)
    req.Header.Set("Content-Type", "application/json")
    
    resp, err := a.httpClient.Do(req)
    // ... handle response
}
```

**Webhook handling:**
```go
func (a *WhatsAppOfficialAdapter) HandleWebhook(w http.ResponseWriter, r *http.Request) {
    // Verify webhook (GET request)
    if r.Method == "GET" {
        mode := r.URL.Query().Get("hub.mode")
        token := r.URL.Query().Get("hub.verify_token")
        challenge := r.URL.Query().Get("hub.challenge")
        
        if mode == "subscribe" && token == a.verifyToken {
            w.Write([]byte(challenge))
            return
        }
    }
    
    // Handle webhook payload (POST request)
    var payload WebhookPayload
    json.NewDecoder(r.Body).Decode(&payload)
    
    for _, entry := range payload.Entry {
        for _, change := range entry.Changes {
            if change.Value.Messages != nil {
                a.processInboundMessage(change.Value.Messages[0])
            }
        }
    }
}
```

#### 2. WhatsApp Não Oficial

**Biblioteca:** `go.mau.fi/whatsmeow` (⭐⭐⭐⭐⭐)  
**GitHub:** https://github.com/tulir/whatsmeow  
**Estrelas:** 2k+  
**Manutenção:** Ativa (usado por Matrix bridge)  
**Licença:** MPL 2.0

**Por que usar:**
- ✅ Suporta Multi-device API (versão mais recente do WhatsApp)
- ✅ Manutenção ativa pela comunidade Matrix
- ✅ Documentação completa
- ✅ Exemplos práticos (mdtest)
- ✅ Suporte a todos os tipos de mensagem

**Instalação:**
```bash
go get go.mau.fi/whatsmeow
go get go.mau.fi/whatsmeow/store/sqlstore
go get go.mau.fi/whatsmeow/types/events
```

**Código básico:**
```go
package main

import (
    "context"
    "fmt"
    "os"
    
    "github.com/mdp/qrterminal"
    _ "github.com/mattn/go-sqlite3"
    "go.mau.fi/whatsmeow"
    "go.mau.fi/whatsmeow/store/sqlstore"
    "go.mau.fi/whatsmeow/types/events"
    waLog "go.mau.fi/whatsmeow/util/log"
)

func main() {
    // Setup database store
    container, _ := sqlstore.New("sqlite3", "file:whatsapp.db?_foreign_keys=on", waLog.Noop)
    deviceStore, _ := container.GetFirstDevice()
    
    // Create client
    client := whatsmeow.NewClient(deviceStore, waLog.Noop)
    
    // Register event handler
    client.AddEventHandler(func(evt interface{}) {
        switch v := evt.(type) {
        case *events.Message:
            fmt.Printf("Received message from %s: %s\n", v.Info.Sender, v.Message.GetConversation())
        }
    })
    
    // Connect
    if client.Store.ID == nil {
        // First time - need QR code
        qrChan, _ := client.GetQRChannel(context.Background())
        client.Connect()
        
        for evt := range qrChan {
            if evt.Event == "code" {
                qrterminal.GenerateHalfBlock(evt.Code, qrterminal.L, os.Stdout)
            }
        }
    } else {
        client.Connect()
    }
    
    // Send message
    recipient := types.NewJID("5544999999999", types.DefaultUserServer)
    client.SendMessage(context.Background(), recipient, &waProto.Message{
        Conversation: proto.String("Hello from whatsmeow!"),
    })
}
```

**Features importantes:**
- ✅ Multi-device sessions
- ✅ Media upload/download
- ✅ Groups
- ✅ Reactions
- ✅ Polls
- ✅ Status/Stories

---

### Telegram

**Biblioteca:** `github.com/go-telegram-bot-api/telegram-bot-api/v5` (⭐⭐⭐⭐⭐)  
**GitHub:** https://github.com/go-telegram-bot-api/telegram-bot-api  
**Estrelas:** 6k+  
**Licença:** MIT

**Alternativa:** `gopkg.in/telebot.v4` (mais features, menos downloads)

**Recomendação:** `go-telegram-bot-api` pela maturidade e simplicidade

**Instalação:**
```bash
go get -u github.com/go-telegram-bot-api/telegram-bot-api/v5
```

**Código básico:**
```go
package main

import (
    "log"
    tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type TelegramAdapter struct {
    bot *tgbotapi.BotAPI
}

func (a *TelegramAdapter) Connect(token string) error {
    bot, err := tgbotapi.NewBotAPI(token)
    if err != nil {
        return err
    }
    a.bot = bot
    return nil
}

func (a *TelegramAdapter) SendMessage(chatID int64, text string) error {
    msg := tgbotapi.NewMessage(chatID, text)
    _, err := a.bot.Send(msg)
    return err
}

func (a *TelegramAdapter) StartReceiving(handler func(*tgbotapi.Message)) {
    updateConfig := tgbotapi.NewUpdate(0)
    updateConfig.Timeout = 60
    
    updates := a.bot.GetUpdatesChan(updateConfig)
    
    for update := range updates {
        if update.Message != nil {
            handler(update.Message)
        }
    }
}
```

**Features:**
- ✅ Long polling built-in
- ✅ Webhooks support
- ✅ Inline keyboards
- ✅ Media handling
- ✅ File upload/download
- ✅ Bot commands

---

### SMS / Voice (Twilio)

**Biblioteca:** `github.com/twilio/twilio-go` (⭐⭐⭐⭐⭐)  
**GitHub:** https://github.com/twilio/twilio-go  
**Licença:** MIT  
**Oficial:** Sim (mantido pela Twilio)

**Instalação:**
```bash
go get github.com/twilio/twilio-go
```

**Código básico:**
```go
package main

import (
    "os"
    twilio "github.com/twilio/twilio-go"
    openapi "github.com/twilio/twilio-go/rest/api/v2010"
)

type TwilioSMSAdapter struct {
    client *twilio.RestClient
    from   string
}

func (a *TwilioSMSAdapter) Connect(accountSid, authToken, from string) {
    os.Setenv("TWILIO_ACCOUNT_SID", accountSid)
    os.Setenv("TWILIO_AUTH_TOKEN", authToken)
    
    a.client = twilio.NewRestClient()
    a.from = from
}

func (a *TwilioSMSAdapter) SendSMS(to, body string) (*openapi.ApiV2010Message, error) {
    params := &openapi.CreateMessageParams{}
    params.SetTo(to)
    params.SetFrom(a.from)
    params.SetBody(body)
    
    return a.client.Api.CreateMessage(params)
}

// Webhook handler
func (a *TwilioSMSAdapter) HandleInbound(w http.ResponseWriter, r *http.Request) {
    from := r.FormValue("From")
    body := r.FormValue("Body")
    messageSid := r.FormValue("MessageSid")
    
    // Process inbound message
    a.onMessageReceived(from, body, messageSid)
    
    // Optional: Send TwiML response
    w.Header().Set("Content-Type", "text/xml")
    w.Write([]byte(`<?xml version="1.0" encoding="UTF-8"?><Response></Response>`))
}
```

**Features:**
- ✅ SMS
- ✅ MMS
- ✅ Voice calls
- ✅ TwiML generation
- ✅ Webhook validation
- ✅ Status callbacks

---

### RCS (Rich Communication Services)

**Biblioteca:** HTTP client padrão + API providers  
**Providers no Brasil:**
- Zenvia
- Infobip
- Pontaltech (parceira Google)

**Implementação:**
```go
type RCSAdapter struct {
    httpClient *http.Client
    apiKey     string
    provider   string // "zenvia", "infobip"
}

// Zenvia RCS implementation
func (a *RCSAdapter) SendRCSZenvia(to, text string) error {
    url := "https://api.zenvia.com/v2/channels/rcs/messages"
    
    payload := map[string]interface{}{
        "from": a.from,
        "to": to,
        "contents": []map[string]interface{}{
            {
                "type": "text",
                "text": text,
            },
        },
    }
    
    req, _ := http.NewRequest("POST", url, jsonPayload(payload))
    req.Header.Set("X-API-TOKEN", a.apiKey)
    req.Header.Set("Content-Type", "application/json")
    
    resp, err := a.httpClient.Do(req)
    // ... handle response
}
```

**Não existe lib Go específica, mas é simples REST:**
- Zenvia API: https://developers.zenvia.com/
- Infobip API: https://www.infobip.com/docs/api

---

### Instagram / Facebook Messenger

**Biblioteca:** HTTP client padrão + Meta Graph API  
**Documentação:**
- Instagram: https://developers.facebook.com/docs/messenger-platform/instagram
- Messenger: https://developers.facebook.com/docs/messenger-platform

**Implementação similar ao WhatsApp Official:**
```go
type InstagramAdapter struct {
    httpClient   *http.Client
    pageAccessToken string
}

func (a *InstagramAdapter) SendMessage(recipientID, text string) error {
    url := "https://graph.facebook.com/v18.0/me/messages"
    
    payload := map[string]interface{}{
        "recipient": map[string]string{"id": recipientID},
        "message": map[string]string{"text": text},
    }
    
    req, _ := http.NewRequest("POST", url, jsonPayload(payload))
    req.Header.Set("Authorization", "Bearer " + a.pageAccessToken)
    req.Header.Set("Content-Type", "application/json")
    
    // ... handle
}
```

---

## 🏗️ PADRÃO DE IMPLEMENTAÇÃO RECOMENDADO

### 1. Estrutura de cada adapter

```
plugins/whatsapp-official/
├── main.go                 # Plugin entrypoint (HashiCorp go-plugin)
├── adapter.go              # Implementa ChannelAdapter interface
├── client.go               # Wrapper da biblioteca
├── normalizer.go           # Converte para formato canônico
├── webhook.go              # Webhook handler (se aplicável)
├── types.go                # Tipos específicos do canal
├── config.go               # Configuração
└── adapter_test.go         # Testes
```

### 2. Template de adapter

```go
// plugins/whatsapp-official/adapter.go
package main

import (
    "context"
    "github.com/linktor/msgfy/core/adapters"
    "github.com/linktor/msgfy/core/types"
)

type WhatsAppOfficialAdapter struct {
    // Client wrapper
    client *WhatsAppClient
    
    // Config
    accessToken   string
    phoneNumberID string
    verifyToken   string
    
    // Message handler callback
    messageHandler adapters.MessageHandler
}

// Metadata
func (a *WhatsAppOfficialAdapter) Name() string {
    return "WhatsApp Official (Meta Cloud API)"
}

func (a *WhatsAppOfficialAdapter) Type() string {
    return "whatsapp_official"
}

func (a *WhatsAppOfficialAdapter) Version() string {
    return "1.0.0"
}

// Lifecycle
func (a *WhatsAppOfficialAdapter) Initialize(ctx context.Context, config map[string]any) error {
    // Parse config
    return nil
}

func (a *WhatsAppOfficialAdapter) Connect(ctx context.Context, credentials map[string]string) error {
    a.accessToken = credentials["access_token"]
    a.phoneNumberID = credentials["phone_number_id"]
    a.verifyToken = credentials["verify_token"]
    
    // Initialize HTTP client
    a.client = NewWhatsAppClient(a.accessToken, a.phoneNumberID)
    
    // Test connection
    return a.client.Ping(ctx)
}

func (a *WhatsAppOfficialAdapter) Disconnect(ctx context.Context) error {
    // Cleanup
    return nil
}

// Health
func (a *WhatsAppOfficialAdapter) HealthCheck(ctx context.Context) (*adapters.HealthStatus, error) {
    err := a.client.Ping(ctx)
    if err != nil {
        return &adapters.HealthStatus{
            Status:  "unhealthy",
            Message: err.Error(),
        }, nil
    }
    
    return &adapters.HealthStatus{
        Status:  "healthy",
        Message: "Connected to Meta Cloud API",
    }, nil
}

// Messaging
func (a *WhatsAppOfficialAdapter) SendMessage(ctx context.Context, msg *types.Message) (*adapters.SendResult, error) {
    // Convert canonical message to WhatsApp format
    waMsg := a.toWhatsAppMessage(msg)
    
    // Send via client
    resp, err := a.client.SendMessage(ctx, waMsg)
    if err != nil {
        return nil, err
    }
    
    return &adapters.SendResult{
        ExternalID: resp.Messages[0].ID,
        Status:     "sent",
        SentAt:     time.Now(),
    }, nil
}

// Webhook/Receiving
func (a *WhatsAppOfficialAdapter) StartReceiving(ctx context.Context, handler adapters.MessageHandler) error {
    a.messageHandler = handler
    
    // Note: WhatsApp Official usa webhooks, não polling
    // O webhook HTTP server é iniciado externamente e chama HandleWebhook()
    
    return nil
}

func (a *WhatsAppOfficialAdapter) StopReceiving(ctx context.Context) error {
    a.messageHandler = nil
    return nil
}

// Webhook handler (chamado pelo HTTP server)
func (a *WhatsAppOfficialAdapter) HandleWebhook(payload []byte) error {
    // Parse webhook payload
    var webhook WhatsAppWebhook
    json.Unmarshal(payload, &webhook)
    
    // Convert to canonical format
    for _, entry := range webhook.Entry {
        for _, change := range entry.Changes {
            if change.Value.Messages != nil {
                for _, waMsg := range change.Value.Messages {
                    canonicalMsg := a.fromWhatsAppMessage(waMsg)
                    
                    // Call handler
                    if a.messageHandler != nil {
                        a.messageHandler(context.Background(), canonicalMsg)
                    }
                }
            }
        }
    }
    
    return nil
}

// Capabilities
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

// Conversion helpers
func (a *WhatsAppOfficialAdapter) toWhatsAppMessage(msg *types.Message) *WhatsAppMessage {
    // Convert canonical to WhatsApp format
    return &WhatsAppMessage{
        To:   msg.To,
        Type: "text",
        Text: map[string]string{"body": msg.Content.Text},
    }
}

func (a *WhatsAppOfficialAdapter) fromWhatsAppMessage(waMsg *WhatsAppInboundMessage) *types.Message {
    // Convert WhatsApp to canonical format
    return &types.Message{
        Direction: "inbound",
        Type:      types.MessageTypeText,
        From:      waMsg.From,
        Content: types.MessageContent{
            Text: waMsg.Text.Body,
        },
        Metadata: map[string]any{
            "wa_message_id": waMsg.ID,
            "timestamp":     waMsg.Timestamp,
        },
    }
}
```

---

## 🎯 RECOMENDAÇÕES FINAIS

### ✅ O que USAR de cada projeto

**Chatwoot (Ruby) - Conceitos arquiteturais:**
- Padrão de normalização de mensagens
- Estrutura de webhook handling
- Rate limiting strategies
- Retry logic para falhas

**go-whatsapp-web-multidevice (Go) - Código direto:**
- Estrutura completa do adapter WhatsApp não oficial
- Multi-session management
- QR code handling
- Webhook delivery pattern

**Bibliotecas Go específicas:**
- whatsmeow para WhatsApp não oficial
- go-telegram-bot-api para Telegram
- twilio-go para SMS/Voice
- HTTP padrão para Meta APIs (WhatsApp/IG/FB)

### ❌ O que EVITAR

- ❌ Implementar protocolo WhatsApp do zero (use whatsmeow)
- ❌ Parse manual de Telegram updates (use go-telegram-bot-api)
- ❌ Criar wrapper Twilio próprio (SDK oficial é excelente)
- ❌ Copiar código Ruby do Chatwoot (use apenas conceitos)

### 📝 ORDEM RECOMENDADA DE IMPLEMENTAÇÃO

**Sprint 1: Web Chat (mais simples)**
- Não precisa de biblioteca externa
- Apenas WebSocket server em Go
- Perfeito para testar plugin system

**Sprint 2: Telegram (segunda mais simples)**
- go-telegram-bot-api é trivial de usar
- Webhooks ou long polling built-in
- Bom para validar normalização de mensagens

**Sprint 3: SMS Twilio (terceira)**
- SDK oficial simplifica tudo
- Webhook handling bem documentado
- Prepara terreno para voz

**Sprint 4: WhatsApp Official (complexa mas estável)**
- HTTP puro, sem libs Go necessárias
- Meta Graph API bem documentada
- Webhooks robustos

**Sprint 5: WhatsApp Unofficial (mais complexa)**
- whatsmeow é poderoso mas tem curva de aprendizado
- Multi-device, sessions, QR codes
- Estudar go-whatsapp-web-multidevice antes

**Sprint 6+: Instagram, Facebook, RCS**
- Após dominar Meta Graph API com WhatsApp
- RCS depende de providers (Zenvia/Infobip)

---

## 📞 RECURSOS ADICIONAIS

### Documentações Oficiais
- **WhatsApp Cloud API:** https://developers.facebook.com/docs/whatsapp/cloud-api
- **Telegram Bot API:** https://core.telegram.org/bots/api
- **Twilio:** https://www.twilio.com/docs
- **Instagram Messaging:** https://developers.facebook.com/docs/messenger-platform/instagram
- **RCS (Google):** https://developers.google.com/business-communications/rcs-business-messaging

### Comunidades
- **whatsmeow:** GitHub Discussions
- **Chatwoot:** GitHub + Discord
- **Go Telegram Bot API:** GitHub Issues

### Tools úteis
- **Postman Collections:** Para testar APIs Meta/Twilio
- **ngrok:** Para testar webhooks localmente
- **Mockoon:** Mock servers para desenvolvimento

---

## 🚀 PRÓXIMO PASSO PRÁTICO

Comece criando o **Web Chat adapter** como POC do plugin system:

```bash
# Estrutura inicial
mkdir -p plugins/webchat
cd plugins/webchat

# Criar arquivos
touch main.go adapter.go websocket.go

# Implementar adapter básico
# Testar plugin loader
# Validar msgfy interface
```

Depois parta para Telegram com go-telegram-bot-api, que é bem mais simples que WhatsApp mas já força você a lidar com webhooks reais.

---

**RESUMO EXECUTIVO:**

✅ **Use bibliotecas Go maduras** (não reimplemente protocolos)  
✅ **Estude Chatwoot** para padrões arquiteturais  
✅ **Clone go-whatsapp-web-multidevice** como base para WhatsApp não oficial  
✅ **Consulte docs oficiais** apenas para features específicas  
✅ **Comece simples** (Web Chat → Telegram → SMS → WhatsApp)  

Essa abordagem vai acelerar MUITO o desenvolvimento mantendo qualidade enterprise! 🎯
