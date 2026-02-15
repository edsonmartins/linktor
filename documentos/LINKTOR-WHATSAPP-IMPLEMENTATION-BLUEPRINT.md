# LINKTOR - WhatsApp Cloud API Implementation Blueprint
## Planejamento Completo para Cobertura 100% da API Oficial

**Objetivo:** Implementar TODAS as capabilities da WhatsApp Cloud API no Linktor/msgfy
**Stack:** Go 1.22+, NATS JetStream, PostgreSQL, React 19
**Timeline:** 30 semanas organizadas em 6 fases
**Diferencial:** Primeiro open source com cobertura COMPLETA da API oficial

---

## 📋 Índice

1. [Pré-requisitos e Setup Inicial](#fase-0-pré-requisitos)
2. [Fase 1: Fundação Robusta](#fase-1-fundação-robusta-semanas-1-6)
3. [Fase 2: Templates Avançados](#fase-2-templates-avançados-semanas-7-12)
4. [Fase 3: WhatsApp Flows Engine](#fase-3-whatsapp-flows-engine-semanas-13-18)
5. [Fase 4: Commerce Suite](#fase-4-commerce-suite-semanas-19-24)
6. [Fase 5: Analytics & Payments](#fase-5-analytics--payments-semanas-25-28)
7. [Fase 6: Features Premium](#fase-6-features-premium-semanas-29-30)

---

## FASE 0: Pré-requisitos (Antes de começar)

### Checklist de Preparação

- [ ] Conta Meta Business Manager criada
- [ ] WhatsApp Business Account (WABA) configurado
- [ ] Número de teste configurado
- [ ] Access token gerado (desenvolvimento)
- [ ] Webhook URL configurado (ngrok para dev)
- [ ] Postman/Insomnia com Meta Graph API collection

### Setup Inicial

```bash
# 1. Clone Linktor
git clone https://github.com/seu-org/linktor.git
cd linktor

# 2. Configure variáveis de ambiente
cat > .env.whatsapp << 'EOF'
WHATSAPP_CLOUD_API_VERSION=v21.0
WHATSAPP_PHONE_NUMBER_ID=your_phone_id
WHATSAPP_WABA_ID=your_waba_id
WHATSAPP_ACCESS_TOKEN=your_token
WHATSAPP_WEBHOOK_VERIFY_TOKEN=your_verify_token
WHATSAPP_BUSINESS_ID=your_business_id
EOF

# 3. Estrutura de diretórios
mkdir -p plugins/whatsapp-official/{handlers,types,utils,flows,commerce}
mkdir -p internal/whatsapp/{client,normalizer,webhook}
```

### Documentação Oficial Essencial

| Recurso | URL | Uso |
|---------|-----|-----|
| **Graph API Reference** | https://developers.facebook.com/docs/graph-api | API endpoints completa |
| **WhatsApp Cloud API Docs** | https://developers.facebook.com/docs/whatsapp/cloud-api | Guia principal |
| **Message Types** | https://developers.facebook.com/docs/whatsapp/cloud-api/reference/messages | Todos os tipos de mensagem |
| **Webhooks Reference** | https://developers.facebook.com/docs/whatsapp/cloud-api/webhooks | Eventos disponíveis |
| **Error Codes** | https://developers.facebook.com/docs/whatsapp/cloud-api/support/error-codes | Troubleshooting |

### Validação Inicial

```bash
# Teste 1: Verificar acesso à API
curl -X GET "https://graph.facebook.com/v21.0/${PHONE_NUMBER_ID}?fields=verified_name,code_verification_status,quality_rating" \
  -H "Authorization: Bearer ${ACCESS_TOKEN}"

# Teste 2: Enviar mensagem básica
curl -X POST "https://graph.facebook.com/v21.0/${PHONE_NUMBER_ID}/messages" \
  -H "Authorization: Bearer ${ACCESS_TOKEN}" \
  -H "Content-Type: application/json" \
  -d '{
    "messaging_product": "whatsapp",
    "to": "5511999999999",
    "type": "text",
    "text": {"body": "Hello from Linktor!"}
  }'
```

**✅ Se ambos retornarem 200 OK → Pode começar!**

---

## FASE 1: Fundação Robusta (Semanas 1-6)

### Objetivos

- ✅ Webhook handler completo (13 campos de subscrição)
- ✅ Reaction messages (enviar + receber)
- ✅ Interactive message builder (reply buttons, list messages, CTA URL)
- ✅ Location e Contact messages
- ✅ Template manager com sync bidirecional

---

### SPRINT 1.1: Webhook Handler Completo (Semana 1)

#### Documentação Oficial
- **Webhooks**: https://developers.facebook.com/docs/whatsapp/cloud-api/webhooks/components
- **Webhook Fields**: https://developers.facebook.com/docs/graph-api/webhooks/reference/whatsapp-business-account

#### Funcionalidades a Implementar

**13 Campos de Subscrição:**
1. `messages` - Mensagens recebidas + status updates
2. `message_template_status_update` - Template aprovado/rejeitado
3. `message_template_quality_update` - Quality score changes
4. `account_alerts` - Alertas de segurança/limite
5. `account_update` - Mudanças na conta
6. `account_review_update` - Status de revisão
7. `phone_number_name_update` - Display name changes
8. `phone_number_quality_update` - Quality rating changes
9. `template_category_update` - Categoria de template
10. `security` - Eventos de segurança
11. `flows` - Flow lifecycle events
12. `business_capability_update` - Capabilities changes
13. `message_echoes` (SMB) - Mensagens enviadas via Business App

#### Estrutura de Código

```go
// internal/whatsapp/webhook/handler.go
package webhook

type WebhookHandler struct {
    logger     *zap.Logger
    dispatcher *EventDispatcher
    verifyToken string
}

// Webhook verification (GET)
func (h *WebhookHandler) Verify(w http.ResponseWriter, r *http.Request) {
    mode := r.URL.Query().Get("hub.mode")
    token := r.URL.Query().Get("hub.verify_token")
    challenge := r.URL.Query().Get("hub.challenge")
    
    if mode == "subscribe" && token == h.verifyToken {
        w.WriteHeader(http.StatusOK)
        w.Write([]byte(challenge))
        return
    }
    
    w.WriteHeader(http.StatusForbidden)
}

// Webhook payload handler (POST)
func (h *WebhookHandler) Handle(w http.ResponseWriter, r *http.Request) {
    var payload WebhookPayload
    if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
        h.logger.Error("Failed to decode webhook", zap.Error(err))
        w.WriteHeader(http.StatusBadRequest)
        return
    }
    
    // SEMPRE retornar 200 OK imediatamente
    w.WriteHeader(http.StatusOK)
    
    // Processar de forma assíncrona
    go h.processPayload(payload)
}

func (h *WebhookHandler) processPayload(payload WebhookPayload) {
    for _, entry := range payload.Entry {
        for _, change := range entry.Changes {
            switch change.Field {
            case "messages":
                h.handleMessages(change.Value)
            case "message_template_status_update":
                h.handleTemplateStatusUpdate(change.Value)
            case "message_template_quality_update":
                h.handleTemplateQualityUpdate(change.Value)
            case "account_alerts":
                h.handleAccountAlerts(change.Value)
            case "phone_number_quality_update":
                h.handlePhoneQualityUpdate(change.Value)
            case "flows":
                h.handleFlowEvents(change.Value)
            // ... handle all 13 fields
            default:
                h.logger.Warn("Unknown webhook field", zap.String("field", change.Field))
            }
        }
    }
}
```

#### Types Completos

```go
// internal/whatsapp/webhook/types.go
package webhook

type WebhookPayload struct {
    Object string  `json:"object"`
    Entry  []Entry `json:"entry"`
}

type Entry struct {
    ID      string   `json:"id"`
    Changes []Change `json:"changes"`
}

type Change struct {
    Field string      `json:"field"`
    Value interface{} `json:"value"` // Will be type-asserted based on Field
}

// Message webhook value
type MessageValue struct {
    MessagingProduct string    `json:"messaging_product"`
    Metadata         Metadata  `json:"metadata"`
    Contacts         []Contact `json:"contacts,omitempty"`
    Messages         []Message `json:"messages,omitempty"`
    Statuses         []Status  `json:"statuses,omitempty"`
    Errors           []Error   `json:"errors,omitempty"`
}

type Message struct {
    ID        string     `json:"id"`
    From      string     `json:"from"`
    Timestamp string     `json:"timestamp"`
    Type      string     `json:"type"` // text, image, video, audio, document, sticker, location, contacts, reaction, interactive, button, order
    
    // Message content (only one will be populated based on Type)
    Text        *TextContent        `json:"text,omitempty"`
    Image       *MediaContent       `json:"image,omitempty"`
    Video       *MediaContent       `json:"video,omitempty"`
    Audio       *MediaContent       `json:"audio,omitempty"`
    Document    *MediaContent       `json:"document,omitempty"`
    Sticker     *MediaContent       `json:"sticker,omitempty"`
    Location    *LocationContent    `json:"location,omitempty"`
    Contacts    []ContactContent    `json:"contacts,omitempty"`
    Reaction    *ReactionContent    `json:"reaction,omitempty"`
    Interactive *InteractiveContent `json:"interactive,omitempty"`
    Button      *ButtonContent      `json:"button,omitempty"`
    Order       *OrderContent       `json:"order,omitempty"`
    System      *SystemContent      `json:"system,omitempty"`
    
    // Context (for replies)
    Context *Context `json:"context,omitempty"`
    
    // Referral (Click-to-WhatsApp Ads)
    Referral *Referral `json:"referral,omitempty"`
}

type ReactionContent struct {
    MessageID string `json:"message_id"`
    Emoji     string `json:"emoji"` // Unicode emoji or empty string to remove
}

type InteractiveContent struct {
    Type string `json:"type"` // button_reply, list_reply, nfm_reply (Flow)
    
    ButtonReply *ButtonReply `json:"button_reply,omitempty"`
    ListReply   *ListReply   `json:"list_reply,omitempty"`
    NfmReply    *NfmReply    `json:"nfm_reply,omitempty"` // Flow response
}

type NfmReply struct {
    Name          string                 `json:"name"`
    Body          string                 `json:"body"`
    ResponseJSON  string                 `json:"response_json"` // Flow data as JSON string
    FlowToken     string                 `json:"flow_token,omitempty"`
}

// Status updates
type Status struct {
    ID           string        `json:"id"`
    Status       string        `json:"status"` // sent, delivered, read, failed
    Timestamp    string        `json:"timestamp"`
    RecipientID  string        `json:"recipient_id"`
    Conversation *Conversation `json:"conversation,omitempty"`
    Pricing      *Pricing      `json:"pricing,omitempty"`
    Errors       []Error       `json:"errors,omitempty"`
}

type Conversation struct {
    ID     string          `json:"id"`
    Origin ConversationOrigin `json:"origin"`
    ExpirationTimestamp string `json:"expiration_timestamp,omitempty"`
}

type ConversationOrigin struct {
    Type string `json:"type"` // user_initiated, business_initiated, referral_conversion
}

type Pricing struct {
    Billable     bool   `json:"billable"`
    PricingModel string `json:"pricing_model"` // CBP (Conversation-Based Pricing)
    Category     string `json:"category"` // marketing, utility, authentication, service
}

// Template status update
type TemplateStatusUpdate struct {
    Event           string `json:"event"` // APPROVED, REJECTED, PENDING, PAUSED, DISABLED
    MessageTemplateID   int64  `json:"message_template_id"`
    MessageTemplateName string `json:"message_template_name"`
    MessageTemplateLanguage string `json:"message_template_language"`
    Reason          string `json:"reason,omitempty"`
    DisableInfo     string `json:"disable_info,omitempty"`
}

// Template quality update
type TemplateQualityUpdate struct {
    MessageTemplateID       int64  `json:"message_template_id"`
    MessageTemplateName     string `json:"message_template_name"`
    MessageTemplateLanguage string `json:"message_template_language"`
    PreviousQualityScore    string `json:"previous_quality_score"` // GREEN, YELLOW, RED
    NewQualityScore         string `json:"new_quality_score"`
}

// Phone number quality update
type PhoneQualityUpdate struct {
    DisplayPhoneNumber string `json:"display_phone_number"`
    Event              string `json:"event"` // FLAGGED, NOT_FLAGGED
    CurrentLimit       string `json:"current_limit"` // TIER_50, TIER_250, TIER_1K, TIER_10K, TIER_100K, TIER_UNLIMITED
}

// Account alerts
type AccountAlert struct {
    Title   string `json:"title"`
    Message string `json:"message"`
}
```

#### Prompt para IA (Implementação)

```markdown
# Prompt: Implementar WhatsApp Webhook Handler Completo em Go

Estou construindo o Linktor, uma plataforma de mensagens multichannel em Go 1.22+.

**Tarefa:** Implementar webhook handler COMPLETO para WhatsApp Cloud API v21.0

**Requisitos:**

1. **Handler HTTP**
   - GET /webhooks/whatsapp → Verificação do webhook (hub.verify_token)
   - POST /webhooks/whatsapp → Receber eventos
   - SEMPRE retornar 200 OK imediatamente (processar async)

2. **Processar 13 campos de webhook:**
   - messages (incluir todos os tipos: text, image, reaction, interactive, nfm_reply, order)
   - message_template_status_update
   - message_template_quality_update
   - account_alerts
   - account_update
   - account_review_update
   - phone_number_name_update
   - phone_number_quality_update
   - template_category_update
   - security
   - flows
   - business_capability_update
   - message_echoes

3. **Estrutura:**
   ```
   internal/whatsapp/webhook/
   ├── handler.go       # HTTP handler
   ├── types.go         # Todos os types do payload
   ├── processor.go     # Lógica de processamento
   └── dispatcher.go    # Event dispatcher para NATS
   ```

4. **Detalhe CRÍTICO:**
   - nfm_reply (Flow responses) NÃO pode ser descartado
   - reaction messages devem ser preservadas
   - Incluir conversation + pricing em status updates

5. **Event Dispatcher:**
   - Publicar eventos processados no NATS JetStream
   - Tópicos: `whatsapp.message.received`, `whatsapp.template.status`, etc.

**Documentação de referência:**
- https://developers.facebook.com/docs/whatsapp/cloud-api/webhooks/components
- https://developers.facebook.com/docs/graph-api/webhooks/reference/whatsapp-business-account

**Stack:** Go 1.22+, NATS JetStream, zap logger

Por favor, gere código completo e production-ready com:
- Error handling robusto
- Logging estruturado
- Testes unitários
- Comments explicativos nos tipos complexos
```

#### Checklist de Validação

- [ ] Webhook verification funciona (GET)
- [ ] Todos os 13 campos são reconhecidos
- [ ] nfm_reply (Flow responses) são processados corretamente
- [ ] reaction messages são capturadas
- [ ] Status updates incluem conversation + pricing
- [ ] Template status changes disparam eventos
- [ ] Phone quality changes são logadas
- [ ] Account alerts são processados
- [ ] Eventos são publicados no NATS
- [ ] Testes cobrem todos os tipos de webhook

---

### SPRINT 1.2: Reaction Messages (Semana 2)

#### Documentação Oficial
- **Send Reactions**: https://developers.facebook.com/docs/whatsapp/cloud-api/guides/send-messages#reaction-messages
- **Receive Reactions**: https://developers.facebook.com/docs/whatsapp/cloud-api/webhooks/payload-examples#reaction-messages

#### Funcionalidades

1. **Receber reações** (já implementado no webhook)
2. **Enviar reações**
3. **Remover reações** (emoji vazio)
4. **UI para agentes reagirem**

#### Código

```go
// internal/whatsapp/client/reactions.go
package client

type ReactionRequest struct {
    MessageID string `json:"message_id"` // ID da mensagem para reagir
    Emoji     string `json:"emoji"`      // Unicode emoji (ou vazio para remover)
}

func (c *Client) SendReaction(ctx context.Context, to string, req ReactionRequest) (*MessageResponse, error) {
    payload := map[string]interface{}{
        "messaging_product": "whatsapp",
        "recipient_type":    "individual",
        "to":                to,
        "type":              "reaction",
        "reaction": map[string]string{
            "message_id": req.MessageID,
            "emoji":      req.Emoji,
        },
    }
    
    return c.sendMessage(ctx, payload)
}

// Helper para remover reação
func (c *Client) RemoveReaction(ctx context.Context, to string, messageID string) (*MessageResponse, error) {
    return c.SendReaction(ctx, to, ReactionRequest{
        MessageID: messageID,
        Emoji:     "", // Empty = remove
    })
}
```

#### Teste Manual

```bash
# Enviar reação
curl -X POST "https://graph.facebook.com/v21.0/${PHONE_NUMBER_ID}/messages" \
  -H "Authorization: Bearer ${ACCESS_TOKEN}" \
  -H "Content-Type: application/json" \
  -d '{
    "messaging_product": "whatsapp",
    "to": "5511999999999",
    "type": "reaction",
    "reaction": {
      "message_id": "wamid.XXX",
      "emoji": "👍"
    }
  }'

# Remover reação (emoji vazio)
curl -X POST "https://graph.facebook.com/v21.0/${PHONE_NUMBER_ID}/messages" \
  -H "Authorization: Bearer ${ACCESS_TOKEN}" \
  -H "Content-Type: application/json" \
  -d '{
    "messaging_product": "whatsapp",
    "to": "5511999999999",
    "type": "reaction",
    "reaction": {
      "message_id": "wamid.XXX",
      "emoji": ""
    }
  }'
```

#### Checklist

- [ ] Enviar reação funciona
- [ ] Remover reação funciona
- [ ] UI mostra reações recebidas
- [ ] Agentes podem reagir via UI
- [ ] Reactions são persistidas no DB

---

### SPRINT 1.3: Interactive Messages (Semanas 3-4)

#### Documentação Oficial
- **Interactive Messages**: https://developers.facebook.com/docs/whatsapp/cloud-api/guides/send-messages#interactive-messages
- **Reply Buttons**: https://developers.facebook.com/docs/whatsapp/cloud-api/guides/send-messages#interactive-reply-buttons-messages
- **List Messages**: https://developers.facebook.com/docs/whatsapp/cloud-api/guides/send-messages#interactive-list-messages
- **CTA URL Buttons**: https://developers.facebook.com/docs/whatsapp/cloud-api/guides/send-messages#interactive-cta-url-messages

#### Tipos Suportados

1. **Reply Buttons** (até 3 botões)
2. **List Messages** (até 10 itens em seções)
3. **CTA URL Buttons**
4. **Location Request** (solicitar localização do usuário)

#### Código

```go
// internal/whatsapp/client/interactive.go
package client

type InteractiveType string

const (
    InteractiveButton   InteractiveType = "button"
    InteractiveList     InteractiveType = "list"
    InteractiveCTAURL   InteractiveType = "cta_url"
    InteractiveLocation InteractiveType = "location_request_message"
)

type InteractiveMessage struct {
    Type   InteractiveType `json:"type"`
    Header *Header         `json:"header,omitempty"`
    Body   Body            `json:"body"`
    Footer *Footer         `json:"footer,omitempty"`
    Action Action          `json:"action"`
}

type Header struct {
    Type     string      `json:"type"` // text, video, image, document
    Text     string      `json:"text,omitempty"`
    Video    *MediaParam `json:"video,omitempty"`
    Image    *MediaParam `json:"image,omitempty"`
    Document *MediaParam `json:"document,omitempty"`
}

type Body struct {
    Text string `json:"text"`
}

type Footer struct {
    Text string `json:"text"`
}

// Action for Reply Buttons
type ButtonAction struct {
    Buttons []Button `json:"buttons"` // Max 3
}

type Button struct {
    Type  string      `json:"type"` // reply
    Reply ButtonReply `json:"reply"`
}

type ButtonReply struct {
    ID    string `json:"id"`    // Unique ID (max 256 chars)
    Title string `json:"title"` // Button text (max 20 chars)
}

// Action for List Messages
type ListAction struct {
    Button   string    `json:"button"`   // Button text (max 20 chars)
    Sections []Section `json:"sections"` // Max 10 sections
}

type Section struct {
    Title string `json:"title,omitempty"` // Max 24 chars
    Rows  []Row  `json:"rows"`            // Max 10 rows per section
}

type Row struct {
    ID          string `json:"id"`          // Unique ID (max 200 chars)
    Title       string `json:"title"`       // Max 24 chars
    Description string `json:"description,omitempty"` // Max 72 chars
}

// Action for CTA URL
type CTAURLAction struct {
    Name       string             `json:"name"` // "cta_url"
    Parameters CTAURLParameters   `json:"parameters"`
}

type CTAURLParameters struct {
    DisplayText string `json:"display_text"` // Button text (max 20 chars)
    URL         string `json:"url"`          // URL with variable {{1}}
}

func (c *Client) SendInteractiveButtons(ctx context.Context, to string, msg InteractiveMessage) (*MessageResponse, error) {
    payload := map[string]interface{}{
        "messaging_product": "whatsapp",
        "recipient_type":    "individual",
        "to":                to,
        "type":              "interactive",
        "interactive":       msg,
    }
    
    return c.sendMessage(ctx, payload)
}

// Helper: Send 3-button quick reply
func (c *Client) SendQuickReply(ctx context.Context, to, bodyText string, buttons []ButtonReply) (*MessageResponse, error) {
    if len(buttons) > 3 {
        return nil, errors.New("maximum 3 buttons allowed")
    }
    
    btnActions := make([]Button, len(buttons))
    for i, btn := range buttons {
        btnActions[i] = Button{
            Type:  "reply",
            Reply: btn,
        }
    }
    
    msg := InteractiveMessage{
        Type: InteractiveButton,
        Body: Body{Text: bodyText},
        Action: ButtonAction{Buttons: btnActions},
    }
    
    return c.SendInteractiveButtons(ctx, to, msg)
}

// Helper: Send list message
func (c *Client) SendList(ctx context.Context, to, bodyText, buttonText string, sections []Section) (*MessageResponse, error) {
    msg := InteractiveMessage{
        Type: InteractiveList,
        Body: Body{Text: bodyText},
        Action: ListAction{
            Button:   buttonText,
            Sections: sections,
        },
    }
    
    return c.SendInteractiveButtons(ctx, to, msg)
}

// Helper: Send CTA URL button
func (c *Client) SendCTAURL(ctx context.Context, to, bodyText, displayText, url string) (*MessageResponse, error) {
    msg := InteractiveMessage{
        Type: InteractiveCTAURL,
        Body: Body{Text: bodyText},
        Action: CTAURLAction{
            Name: "cta_url",
            Parameters: CTAURLParameters{
                DisplayText: displayText,
                URL:         url,
            },
        },
    }
    
    return c.SendInteractiveButtons(ctx, to, msg)
}

// Helper: Request user's location
func (c *Client) RequestLocation(ctx context.Context, to, bodyText string) (*MessageResponse, error) {
    msg := InteractiveMessage{
        Type: InteractiveLocation,
        Body: Body{Text: bodyText},
        Action: map[string]string{
            "name": "send_location",
        },
    }
    
    return c.SendInteractiveButtons(ctx, to, msg)
}
```

#### UI Component (React)

```typescript
// frontend/src/components/InteractiveMessageBuilder.tsx

interface InteractiveMessageBuilderProps {
  onSend: (message: InteractiveMessage) => void;
}

export function InteractiveMessageBuilder({ onSend }: InteractiveMessageBuilderProps) {
  const [type, setType] = useState<'button' | 'list' | 'cta_url'>('button');
  const [bodyText, setBodyText] = useState('');
  const [buttons, setButtons] = useState<ButtonReply[]>([
    { id: '1', title: '' },
  ]);
  
  const addButton = () => {
    if (buttons.length < 3) {
      setButtons([...buttons, { id: String(buttons.length + 1), title: '' }]);
    }
  };
  
  const handleSend = () => {
    const message: InteractiveMessage = {
      type,
      body: { text: bodyText },
      action: type === 'button' 
        ? { buttons: buttons.map(b => ({ type: 'reply', reply: b })) }
        : // ... handle list/cta_url
    };
    
    onSend(message);
  };
  
  return (
    <div className="interactive-builder">
      <select value={type} onChange={(e) => setType(e.target.value)}>
        <option value="button">Reply Buttons</option>
        <option value="list">List Message</option>
        <option value="cta_url">CTA URL</option>
      </select>
      
      <textarea
        placeholder="Message text"
        value={bodyText}
        onChange={(e) => setBodyText(e.target.value)}
      />
      
      {type === 'button' && (
        <div className="buttons-config">
          {buttons.map((btn, i) => (
            <input
              key={i}
              placeholder={`Button ${i + 1} (max 20 chars)`}
              maxLength={20}
              value={btn.title}
              onChange={(e) => {
                const newButtons = [...buttons];
                newButtons[i].title = e.target.value;
                setButtons(newButtons);
              }}
            />
          ))}
          {buttons.length < 3 && (
            <button onClick={addButton}>+ Add Button</button>
          )}
        </div>
      )}
      
      <button onClick={handleSend}>Send Interactive Message</button>
    </div>
  );
}
```

#### Prompt para IA

```markdown
# Prompt: Implementar Interactive Messages Builder UI

Estou construindo a UI do Linktor em React 19 + TypeScript.

**Tarefa:** Criar componente visual para agentes construírem mensagens interativas WhatsApp

**Requisitos:**

1. **3 tipos de mensagens:**
   - Reply Buttons (até 3 botões de 20 chars cada)
   - List Messages (até 10 seções com até 10 itens cada)
   - CTA URL Buttons (1 botão com URL)

2. **Interface drag-and-drop style:**
   - Preview em tempo real (lado direito)
   - Builder (lado esquerdo)
   - Validação de limites (chars, quantidade)

3. **Features:**
   - Salvar como template
   - Usar variáveis {{1}}, {{2}} no texto
   - Preview mobile-like do WhatsApp
   - Export JSON da mensagem

**Stack:** React 19, TypeScript, Tailwind CSS, shadcn/ui

**Referência visual:** Imitar WhatsApp Business Manager message builder

Por favor, gere componente completo e production-ready.
```

#### Checklist

- [ ] Enviar reply buttons funciona (até 3)
- [ ] Enviar list messages funciona (até 10 seções)
- [ ] Enviar CTA URL funciona
- [ ] Location request funciona
- [ ] UI builder implementada
- [ ] Preview em tempo real
- [ ] Validação de limites de caracteres
- [ ] Responses de interactive messages são capturadas no webhook

---

### SPRINT 1.4: Location & Contact Messages (Semana 5)

#### Documentação Oficial
- **Send Location**: https://developers.facebook.com/docs/whatsapp/cloud-api/guides/send-messages#location-messages
- **Send Contacts**: https://developers.facebook.com/docs/whatsapp/cloud-api/guides/send-messages#contacts-messages
- **Receive Location**: https://developers.facebook.com/docs/whatsapp/cloud-api/webhooks/payload-examples#location-messages

#### Código

```go
// internal/whatsapp/client/location.go
package client

type LocationMessage struct {
    Latitude  float64 `json:"latitude"`
    Longitude float64 `json:"longitude"`
    Name      string  `json:"name,omitempty"`
    Address   string  `json:"address,omitempty"`
}

func (c *Client) SendLocation(ctx context.Context, to string, loc LocationMessage) (*MessageResponse, error) {
    payload := map[string]interface{}{
        "messaging_product": "whatsapp",
        "to":                to,
        "type":              "location",
        "location":          loc,
    }
    
    return c.sendMessage(ctx, payload)
}

// internal/whatsapp/client/contacts.go
package client

type ContactMessage struct {
    Contacts []Contact `json:"contacts"`
}

type Contact struct {
    Name        Name          `json:"name"`
    Birthday    string        `json:"birthday,omitempty"` // YYYY-MM-DD
    Phones      []Phone       `json:"phones,omitempty"`
    Emails      []Email       `json:"emails,omitempty"`
    Urls        []URL         `json:"urls,omitempty"`
    Addresses   []Address     `json:"addresses,omitempty"`
    Org         *Organization `json:"org,omitempty"`
}

type Name struct {
    FormattedName string `json:"formatted_name"`
    FirstName     string `json:"first_name,omitempty"`
    LastName      string `json:"last_name,omitempty"`
    MiddleName    string `json:"middle_name,omitempty"`
    Suffix        string `json:"suffix,omitempty"`
    Prefix        string `json:"prefix,omitempty"`
}

type Phone struct {
    Phone string `json:"phone"`
    Type  string `json:"type,omitempty"` // CELL, MAIN, IPHONE, HOME, WORK
    WaID  string `json:"wa_id,omitempty"`
}

type Email struct {
    Email string `json:"email"`
    Type  string `json:"type,omitempty"` // HOME, WORK
}

type URL struct {
    URL  string `json:"url"`
    Type string `json:"type,omitempty"` // HOME, WORK
}

type Address struct {
    Street      string `json:"street,omitempty"`
    City        string `json:"city,omitempty"`
    State       string `json:"state,omitempty"`
    Zip         string `json:"zip,omitempty"`
    Country     string `json:"country,omitempty"`
    CountryCode string `json:"country_code,omitempty"`
    Type        string `json:"type,omitempty"` // HOME, WORK
}

type Organization struct {
    Company    string `json:"company,omitempty"`
    Department string `json:"department,omitempty"`
    Title      string `json:"title,omitempty"`
}

func (c *Client) SendContacts(ctx context.Context, to string, contacts []Contact) (*MessageResponse, error) {
    payload := map[string]interface{}{
        "messaging_product": "whatsapp",
        "to":                to,
        "type":              "contacts",
        "contacts":          contacts,
    }
    
    return c.sendMessage(ctx, payload)
}
```

#### Checklist

- [ ] Enviar localização funciona
- [ ] Receber localização funciona
- [ ] Enviar contatos funciona
- [ ] Receber contatos funciona
- [ ] UI para agentes enviarem localização
- [ ] UI para enviar contatos (vCard builder)

---

### SPRINT 1.5: Template Manager (Semana 6)

#### Documentação Oficial
- **Message Templates**: https://developers.facebook.com/docs/whatsapp/business-management-api/message-templates
- **Create Template**: https://developers.facebook.com/docs/whatsapp/business-management-api/message-templates/create
- **Get Templates**: https://developers.facebook.com/docs/whatsapp/business-management-api/message-templates#list-templates

#### Funcionalidades

1. **Listar templates** (com cursor pagination)
2. **Criar template**
3. **Atualizar template**
4. **Deletar template**
5. **Sync bidirecional** (cache local + invalidação via webhook)

#### Código

```go
// internal/whatsapp/client/templates.go
package client

type Template struct {
    ID       string              `json:"id"`
    Name     string              `json:"name"`
    Category string              `json:"category"` // MARKETING, UTILITY, AUTHENTICATION
    Language string              `json:"language"`
    Status   string              `json:"status"`   // APPROVED, PENDING, REJECTED
    Components []TemplateComponent `json:"components"`
    QualityScore *QualityScore    `json:"quality_score,omitempty"`
}

type TemplateComponent struct {
    Type       string                   `json:"type"` // HEADER, BODY, FOOTER, BUTTONS
    Format     string                   `json:"format,omitempty"` // TEXT, IMAGE, VIDEO, DOCUMENT
    Text       string                   `json:"text,omitempty"`
    Buttons    []TemplateButton         `json:"buttons,omitempty"`
    Example    *TemplateExample         `json:"example,omitempty"`
}

type TemplateButton struct {
    Type        string `json:"type"` // QUICK_REPLY, URL, PHONE_NUMBER, COPY_CODE, FLOW
    Text        string `json:"text"`
    PhoneNumber string `json:"phone_number,omitempty"`
    URL         string `json:"url,omitempty"`
    Example     []string `json:"example,omitempty"`
    FlowID      string `json:"flow_id,omitempty"`
    FlowAction  string `json:"flow_action,omitempty"` // navigate, data_exchange
}

type QualityScore struct {
    Score  string    `json:"score"` // GREEN, YELLOW, RED, UNKNOWN
    Date   time.Time `json:"date"`
}

func (c *Client) ListTemplates(ctx context.Context, limit int, after string) (*TemplateList, error) {
    url := fmt.Sprintf("%s/%s/message_templates", c.baseURL, c.wabaID)
    
    params := map[string]string{
        "limit": strconv.Itoa(limit),
    }
    if after != "" {
        params["after"] = after
    }
    
    // ... make request
}

func (c *Client) CreateTemplate(ctx context.Context, tpl Template) (*Template, error) {
    url := fmt.Sprintf("%s/%s/message_templates", c.baseURL, c.wabaID)
    
    // ... make POST request
}

func (c *Client) DeleteTemplate(ctx context.Context, templateName string) error {
    url := fmt.Sprintf("%s/%s/message_templates", c.baseURL, c.wabaID)
    
    // ... make DELETE request with name parameter
}

// Template cache with invalidation
type TemplateCache struct {
    sync.RWMutex
    templates map[string]*Template
    lastSync  time.Time
}

func (tc *TemplateCache) InvalidateTemplate(name string) {
    tc.Lock()
    defer tc.Unlock()
    delete(tc.templates, name)
}
```

#### Prompt para IA

```markdown
# Prompt: Implementar Template Manager com Cache Inteligente

**Tarefa:** Criar gerenciador de templates WhatsApp com sync bidirecional

**Requisitos:**

1. **CRUD completo:**
   - List (com cursor pagination)
   - Create
   - Update (delete + create)
   - Delete

2. **Cache inteligente:**
   - Cache local em memória (map[string]*Template)
   - Invalidação via webhook (message_template_status_update)
   - Refresh automático a cada 1 hora
   - Thread-safe (sync.RWMutex)

3. **Webhook integration:**
   - Quando template_status_update chegar, invalidar cache
   - Quando quality_update chegar, atualizar score

4. **UI Requirements:**
   - Lista de templates com status/quality
   - Template builder visual
   - Preview em tempo real

**Stack:** Go 1.22+, Meta Graph API v21.0

Por favor, gere código production-ready com testes.
```

#### Checklist

- [ ] Listar templates funciona (com pagination)
- [ ] Criar template funciona
- [ ] Deletar template funciona
- [ ] Cache local implementado
- [ ] Invalidação via webhook funciona
- [ ] UI para gerenciar templates
- [ ] Template builder visual
- [ ] Quality score exibido

---

## FASE 2: Templates Avançados (Semanas 7-12)

### SPRINT 2.1: Carousel Templates (Semanas 7-8)

#### Documentação Oficial
- **Carousel Templates**: https://developers.facebook.com/docs/whatsapp/business-management-api/message-templates/components#carousel
- **Send Carousel**: https://developers.facebook.com/docs/whatsapp/cloud-api/guides/send-message-templates#carousel

#### Código

```go
// internal/whatsapp/client/carousel.go
package client

type CarouselTemplate struct {
    Name     string           `json:"name"`
    Language string           `json:"language"`
    Category string           `json:"category"` // MARKETING only
    Components []CarouselComponent `json:"components"`
}

type CarouselComponent struct {
    Type   string         `json:"type"` // "CAROUSEL"
    Cards  []CarouselCard `json:"cards"` // Min 2, Max 10
}

type CarouselCard struct {
    Components []CardComponent `json:"components"`
}

type CardComponent struct {
    Type    string                `json:"type"` // HEADER, BODY, BUTTONS
    Format  string                `json:"format,omitempty"` // IMAGE, VIDEO
    Example *CardExample          `json:"example,omitempty"`
    Text    string                `json:"text,omitempty"`
    Buttons []TemplateButton      `json:"buttons,omitempty"` // Max 2 per card
}

type CardExample struct {
    HeaderHandle []string `json:"header_handle,omitempty"` // Media URLs
}

func (c *Client) SendCarouselTemplate(ctx context.Context, to string, template CarouselTemplate, cards []CardData) (*MessageResponse, error) {
    components := []map[string]interface{}{
        {
            "type": "carousel",
            "cards": buildCards(cards),
        },
    }
    
    payload := map[string]interface{}{
        "messaging_product": "whatsapp",
        "to":                to,
        "type":              "template",
        "template": map[string]interface{}{
            "name":       template.Name,
            "language":   map[string]string{"code": template.Language},
            "components": components,
        },
    }
    
    return c.sendMessage(ctx, payload)
}
```

#### Prompt para IA

```markdown
# Prompt: Implementar Carousel Template Builder UI

**Tarefa:** Criar builder visual de Carousel Templates para WhatsApp

**Requisitos:**

1. **Drag-and-drop cards (2-10 cards)**
   - Cada card: Image/Video header + Body text + até 2 botões
   - Reordenar cards via drag

2. **Constraints:**
   - Todos os cards DEVEM ter mesmo formato de header (IMAGE ou VIDEO)
   - Todos os cards DEVEM ter mesmos tipos de botão
   - Body text: max 160 chars
   - Button text: max 20 chars

3. **Preview:**
   - Horizontal scrollable carousel
   - Mobile-like preview

4. **Export:**
   - Gerar JSON do template
   - Enviar para backend

**Stack:** React 19, dnd-kit, Tailwind

Por favor, gere componente completo com validação em tempo real.
```

#### Checklist

- [ ] Criar carousel template funciona
- [ ] Enviar carousel funciona
- [ ] Builder UI implementado
- [ ] Validação de constraints
- [ ] Preview funciona
- [ ] Min 2 / Max 10 cards enforced

---

### SPRINT 2.2: Authentication Templates (Semanas 9-10)

#### Documentação Oficial
- **Authentication Templates**: https://developers.facebook.com/docs/whatsapp/business-management-api/authentication-templates
- **Zero-tap & One-tap**: https://developers.facebook.com/docs/whatsapp/business-management-api/authentication-templates/autofill-button-authentication-templates

#### Tipos

1. **Copy Code** (manual)
2. **One-tap** (autofill button)
3. **Zero-tap** (Android only, automatic)

#### Código

```go
// internal/whatsapp/client/auth_templates.go
package client

type AuthTemplateType string

const (
    AuthCopyCode AuthTemplateType = "COPY_CODE"
    AuthOneTap   AuthTemplateType = "ONE_TAP"
    AuthZeroTap  AuthTemplateType = "ZERO_TAP"
)

type AuthTemplate struct {
    Name         string           `json:"name"`
    Language     string           `json:"language"`
    Category     string           `json:"category"` // Always AUTHENTICATION
    AuthType     AuthTemplateType `json:"-"`
    CodeExpiry   int              `json:"code_expiration_minutes"` // For COPY_CODE
    BodyText     string           `json:"body_text"`
    ButtonText   string           `json:"button_text,omitempty"` // For ONE_TAP
    PackageName  string           `json:"package_name,omitempty"` // For ZERO_TAP (Android)
    SignatureHash string          `json:"signature_hash,omitempty"` // For ZERO_TAP
}

func (c *Client) SendAuthTemplate(ctx context.Context, to, templateName, code string) (*MessageResponse, error) {
    components := []map[string]interface{}{
        {
            "type": "body",
            "parameters": []map[string]string{
                {"type": "text", "text": code}, // OTP code as {{1}}
            },
        },
        {
            "type": "button",
            "sub_type": "url",
            "index": "0",
            "parameters": []map[string]string{
                {"type": "text", "text": code}, // Code for autofill
            },
        },
    }
    
    payload := map[string]interface{}{
        "messaging_product": "whatsapp",
        "to":                to,
        "type":              "template",
        "template": map[string]interface{}{
            "name":     templateName,
            "language": map[string]string{"code": "en_US"},
            "components": components,
        },
    }
    
    return c.sendMessage(ctx, payload)
}
```

#### Checklist

- [ ] Copy Code template funciona
- [ ] One-tap template funciona
- [ ] Zero-tap template funciona (Android)
- [ ] OTP code gerado e enviado
- [ ] Template builder para auth
- [ ] Expiry time configurável

---

### SPRINT 2.3: LTO & Coupon Templates (Semanas 11-12)

#### Documentação Oficial
- **Limited-Time Offer**: https://developers.facebook.com/docs/whatsapp/business-management-api/message-templates/limited-time-offer
- **Coupon Code**: https://developers.facebook.com/docs/whatsapp/business-management-api/message-templates/coupon-code

#### Código

```go
// internal/whatsapp/client/lto_templates.go
package client

type LTOTemplate struct {
    Name         string `json:"name"`
    Language     string `json:"language"`
    OfferExpiration time.Time `json:"offer_expiration"`
    BodyText     string `json:"body_text"`
}

type CouponTemplate struct {
    Name       string `json:"name"`
    Language   string `json:"language"`
    CouponCode string `json:"coupon_code"`
    BodyText   string `json:"body_text"`
}

func (c *Client) SendLTOTemplate(ctx context.Context, to, templateName string, expiration time.Time) (*MessageResponse, error) {
    components := []map[string]interface{}{
        {
            "type": "limited_time_offer",
            "parameters": []map[string]interface{}{
                {
                    "type": "limited_time_offer",
                    "limited_time_offer": map[string]int64{
                        "expiration_time_ms": expiration.UnixMilli(),
                    },
                },
            },
        },
    }
    
    // ... send template
}

func (c *Client) SendCouponTemplate(ctx context.Context, to, templateName, couponCode string) (*MessageResponse, error) {
    components := []map[string]interface{}{
        {
            "type": "button",
            "sub_type": "copy_code",
            "index": "0",
            "parameters": []map[string]string{
                {"type": "coupon_code", "coupon_code": couponCode},
            },
        },
    }
    
    // ... send template
}
```

#### Checklist

- [ ] LTO template com countdown funciona
- [ ] Coupon template com copy-to-clipboard funciona
- [ ] Expiration time dinâmico
- [ ] Coupon code gerado dinamicamente
- [ ] UI builder para ambos

---

## FASE 3: WhatsApp Flows Engine (Semanas 13-18)

**Esta é a feature MAIS complexa e de MAIOR valor!**

### SPRINT 3.1: Flows Infrastructure (Semanas 13-14)

#### Documentação Oficial CRÍTICA
- **Flows Overview**: https://developers.facebook.com/docs/whatsapp/flows
- **Flow JSON Schema**: https://developers.facebook.com/docs/whatsapp/flows/reference/flowjson
- **Data Exchange**: https://developers.facebook.com/docs/whatsapp/flows/reference/flowjson#data-exchange
- **Encryption**: https://developers.facebook.com/docs/whatsapp/flows/guides/implementingyourflowendpoint

#### Pré-requisitos

1. **Gerar par de chaves RSA 2048-bit**
2. **Assinar public key para cada número**
3. **Criar endpoint para data exchange**
4. **Implementar encrypt/decrypt**

#### Código: RSA Key Generation

```go
// internal/whatsapp/flows/crypto.go
package flows

import (
    "crypto/rand"
    "crypto/rsa"
    "crypto/x509"
    "encoding/pem"
)

func GenerateRSAKeyPair() (*rsa.PrivateKey, error) {
    return rsa.GenerateKey(rand.Reader, 2048)
}

func ExportPublicKey(pub *rsa.PublicKey) (string, error) {
    pubBytes, err := x509.MarshalPKIXPublicKey(pub)
    if err != nil {
        return "", err
    }
    
    pemBlock := &pem.Block{
        Type:  "PUBLIC KEY",
        Bytes: pubBytes,
    }
    
    return string(pem.EncodeToMemory(pemBlock)), nil
}

func ExportPrivateKey(priv *rsa.PrivateKey) (string, error) {
    privBytes := x509.MarshalPKCS1PrivateKey(priv)
    
    pemBlock := &pem.Block{
        Type:  "RSA PRIVATE KEY",
        Bytes: privBytes,
    }
    
    return string(pem.EncodeToMemory(pemBlock)), nil
}

// Sign public key with WhatsApp
func (c *Client) RegisterFlowPublicKey(ctx context.Context, phoneNumberID, publicKey string) error {
    url := fmt.Sprintf("%s/%s/whatsapp_business_encryption", c.baseURL, phoneNumberID)
    
    payload := url.Values{}
    payload.Set("business_public_key", publicKey)
    
    req, err := http.NewRequestWithContext(ctx, "POST", url, strings.NewReader(payload.Encode()))
    if err != nil {
        return err
    }
    
    req.Header.Set("Authorization", "Bearer "+c.accessToken)
    req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
    
    resp, err := c.httpClient.Do(req)
    if err != nil {
        return err
    }
    defer resp.Body.Close()
    
    if resp.StatusCode != 200 {
        return fmt.Errorf("failed to register key: %d", resp.StatusCode)
    }
    
    return nil
}
```

#### Código: Flow Data Exchange Endpoint

```go
// internal/whatsapp/flows/endpoint.go
package flows

import (
    "crypto/aes"
    "crypto/cipher"
    "crypto/rsa"
    "crypto/sha256"
    "encoding/base64"
    "encoding/json"
)

type FlowDataExchangeRequest struct {
    EncryptedFlowData string `json:"encrypted_flow_data"`
    EncryptedAESKey   string `json:"encrypted_aes_key"`
    InitialVector     string `json:"initial_vector"`
}

type FlowDataExchangeResponse struct {
    Version          string                 `json:"version"` // "3.0"
    Screen           string                 `json:"screen"`
    Data             map[string]interface{} `json:"data"`
    ErrorMessages    []ErrorMessage         `json:"error_messages,omitempty"`
}

type ErrorMessage struct {
    ScreenID     string `json:"screen_id"`
    ComponentID  string `json:"component_id"`
    ErrorMessage string `json:"error_message"`
}

func (h *FlowEndpointHandler) HandleDataExchange(w http.ResponseWriter, r *http.Request) {
    var req FlowDataExchangeRequest
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        http.Error(w, "Invalid request", http.StatusBadRequest)
        return
    }
    
    // 1. Decrypt AES key using RSA private key
    encryptedAESKey, _ := base64.StdEncoding.DecodeString(req.EncryptedAESKey)
    aesKey, err := rsa.DecryptOAEP(
        sha256.New(),
        rand.Reader,
        h.privateKey,
        encryptedAESKey,
        nil,
    )
    if err != nil {
        http.Error(w, "Decryption failed", http.StatusBadRequest)
        return
    }
    
    // 2. Decrypt flow data using AES key
    encryptedData, _ := base64.StdEncoding.DecodeString(req.EncryptedFlowData)
    iv, _ := base64.StdEncoding.DecodeString(req.InitialVector)
    
    block, _ := aes.NewCipher(aesKey)
    mode := cipher.NewCBCDecrypter(block, iv)
    
    decrypted := make([]byte, len(encryptedData))
    mode.CryptBlocks(decrypted, encryptedData)
    
    // 3. Remove PKCS7 padding
    decrypted = removePKCS7Padding(decrypted)
    
    // 4. Parse flow data
    var flowData map[string]interface{}
    json.Unmarshal(decrypted, &flowData)
    
    // 5. Process flow logic
    response := h.processFlowData(flowData)
    
    // 6. Encrypt response
    encryptedResponse := h.encryptResponse(response, aesKey, iv)
    
    // 7. Return encrypted response
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(encryptedResponse)
}

func (h *FlowEndpointHandler) encryptResponse(data FlowDataExchangeResponse, aesKey, iv []byte) map[string]string {
    jsonData, _ := json.Marshal(data)
    
    // Add PKCS7 padding
    padded := addPKCS7Padding(jsonData, aes.BlockSize)
    
    // Encrypt
    block, _ := aes.NewCipher(aesKey)
    mode := cipher.NewCBCEncrypter(block, iv)
    
    encrypted := make([]byte, len(padded))
    mode.CryptBlocks(encrypted, padded)
    
    return map[string]string{
        "encrypted_flow_data": base64.StdEncoding.EncodeToString(encrypted),
    }
}

func addPKCS7Padding(data []byte, blockSize int) []byte {
    padding := blockSize - len(data)%blockSize
    padText := bytes.Repeat([]byte{byte(padding)}, padding)
    return append(data, padText...)
}

func removePKCS7Padding(data []byte) []byte {
    length := len(data)
    padding := int(data[length-1])
    return data[:length-padding]
}
```

#### Checklist

- [ ] RSA key pair gerado
- [ ] Public key registrado com WhatsApp
- [ ] Endpoint de data exchange funcional
- [ ] Decrypt request funciona
- [ ] Encrypt response funciona
- [ ] PKCS7 padding correto
- [ ] Testes com payload real da Meta

---

### SPRINT 3.2: Flow JSON Builder (Semanas 15-16)

#### Documentação
- **Flow JSON v7.0**: https://developers.facebook.com/docs/whatsapp/flows/reference/flowjson
- **Components**: https://developers.facebook.com/docs/whatsapp/flows/reference/flowjson#components
- **Screens**: https://developers.facebook.com/docs/whatsapp/flows/reference/flowjson#screens

#### Flow JSON Structure

```json
{
  "version": "7.0",
  "data_api_version": "3.0",
  "routing_model": {},
  "screens": [
    {
      "id": "SCREEN_ID",
      "title": "Screen Title",
      "terminal": false,
      "data": {},
      "layout": {
        "type": "SingleColumnLayout",
        "children": [
          {
            "type": "TextHeading",
            "text": "Welcome!"
          },
          {
            "type": "TextInput",
            "name": "user_name",
            "label": "Your Name",
            "required": true,
            "input-type": "text"
          },
          {
            "type": "Footer",
            "label": "Continue",
            "on-click-action": {
              "name": "navigate",
              "next": { "name": "NEXT_SCREEN" },
              "payload": {}
            }
          }
        ]
      }
    }
  ]
}
```

#### Component Types Suportados

1. **Layout**: SingleColumnLayout
2. **Text**: TextHeading, TextSubheading, TextBody, TextCaption
3. **Input**: TextInput, TextArea, DatePicker, Dropdown, RadioButtonsGroup, CheckboxGroup, OptIn
4. **Media**: Image, EmbeddedLink
5. **Actions**: Footer (navigation button)
6. **Display**: If (conditional rendering), PhotoPicker

#### Código: Flow Builder

```go
// internal/whatsapp/flows/builder.go
package flows

type FlowJSON struct {
    Version        string         `json:"version"`
    DataAPIVersion string         `json:"data_api_version"`
    RoutingModel   map[string]any `json:"routing_model"`
    Screens        []Screen       `json:"screens"`
}

type Screen struct {
    ID       string         `json:"id"`
    Title    string         `json:"title"`
    Terminal bool           `json:"terminal,omitempty"`
    Data     map[string]any `json:"data,omitempty"`
    Layout   Layout         `json:"layout"`
}

type Layout struct {
    Type     string      `json:"type"` // "SingleColumnLayout"
    Children []Component `json:"children"`
}

type Component struct {
    Type string `json:"type"`
    // ... dynamic fields based on type
}

// Builder pattern
type FlowBuilder struct {
    flow FlowJSON
}

func NewFlow() *FlowBuilder {
    return &FlowBuilder{
        flow: FlowJSON{
            Version:        "7.0",
            DataAPIVersion: "3.0",
            RoutingModel:   map[string]any{},
            Screens:        []Screen{},
        },
    }
}

func (b *FlowBuilder) AddScreen(id, title string) *ScreenBuilder {
    screen := Screen{
        ID:    id,
        Title: title,
        Layout: Layout{
            Type:     "SingleColumnLayout",
            Children: []Component{},
        },
    }
    
    b.flow.Screens = append(b.flow.Screens, screen)
    return &ScreenBuilder{screen: &b.flow.Screens[len(b.flow.Screens)-1]}
}

func (b *FlowBuilder) Build() FlowJSON {
    return b.flow
}

type ScreenBuilder struct {
    screen *Screen
}

func (sb *ScreenBuilder) AddHeading(text string) *ScreenBuilder {
    sb.screen.Layout.Children = append(sb.screen.Layout.Children, Component{
        Type: "TextHeading",
        // ... additional fields
    })
    return sb
}

func (sb *ScreenBuilder) AddTextInput(name, label string, required bool) *ScreenBuilder {
    sb.screen.Layout.Children = append(sb.screen.Layout.Children, Component{
        Type: "TextInput",
        // ... additional fields
    })
    return sb
}

func (sb *ScreenBuilder) AddFooter(label, nextScreen string) *ScreenBuilder {
    sb.screen.Layout.Children = append(sb.screen.Layout.Children, Component{
        Type: "Footer",
        // ... navigation action
    })
    return sb
}

// Usage example
func BuildLeadGenFlow() FlowJSON {
    return NewFlow().
        AddScreen("WELCOME", "Welcome").
            AddHeading("Get Started").
            AddTextInput("name", "Your Name", true).
            AddTextInput("email", "Email Address", true).
            AddFooter("Continue", "THANK_YOU").
        AddScreen("THANK_YOU", "Thank You").
            AddHeading("Thanks for signing up!").
            AddFooter("Close", "").
        Build()
}
```

#### Prompt para IA (Flow Builder UI)

```markdown
# Prompt: Visual Flow Builder Interface

**Tarefa:** Criar interface drag-and-drop para construir WhatsApp Flows visualmente

**Requisitos:**

1. **Canvas de Screens:**
   - Drag screens na ordem
   - Conectar screens com setas (navigation flow)
   - Click screen para editar

2. **Component Palette:**
   - Arrastar componentes para screen
   - TextHeading, TextInput, DatePicker, Dropdown, CheckboxGroup, Footer, etc.
   - Preview de cada componente

3. **Property Panel:**
   - Editar propriedades do componente selecionado
   - Validação em tempo real
   - Conditional rendering (If component)

4. **Preview:**
   - Mobile-like preview do Flow
   - Testar navegação
   - Simular data input

5. **Export:**
   - Gerar Flow JSON v7.0
   - Validar contra schema
   - Deploy direto para WhatsApp

**Stack:** React 19, react-flow (para diagrama), TypeScript, Tailwind

**Referência:** Imitar WhatsApp Manager Flow Builder

Por favor, gere aplicação completa com state management (Zustand/Jotai).
```

#### Checklist

- [ ] Flow JSON builder programático funciona
- [ ] UI visual builder implementada
- [ ] Todos os 14+ component types suportados
- [ ] Navegação entre screens funciona
- [ ] Conditional rendering (If) funciona
- [ ] Export JSON validado contra schema
- [ ] Preview mobile-like funciona

---

### SPRINT 3.3: Flow Lifecycle Management (Semanas 17-18)

#### Documentação
- **Create Flow**: https://developers.facebook.com/docs/whatsapp/flows/reference/flow-api#creating-a-flow
- **Publish Flow**: https://developers.facebook.com/docs/whatsapp/flows/reference/flow-api#publishing-a-flow
- **Flow States**: Draft → Published → Deprecated

#### Código

```go
// internal/whatsapp/client/flows.go
package client

type FlowStatus string

const (
    FlowDraft      FlowStatus = "DRAFT"
    FlowPublished  FlowStatus = "PUBLISHED"
    FlowDeprecated FlowStatus = "DEPRECATED"
)

type Flow struct {
    ID         string     `json:"id"`
    Name       string     `json:"name"`
    Status     FlowStatus `json:"status"`
    Categories []string   `json:"categories"` // SIGN_UP, SIGN_IN, APPOINTMENT_BOOKING, etc.
    JSON       FlowJSON   `json:"json_version,omitempty"`
    EndpointURI string    `json:"endpoint_uri,omitempty"`
}

func (c *Client) CreateFlow(ctx context.Context, name string, categories []string) (*Flow, error) {
    url := fmt.Sprintf("%s/%s/flows", c.baseURL, c.wabaID)
    
    payload := map[string]interface{}{
        "name":       name,
        "categories": categories,
    }
    
    // ... make POST request
}

func (c *Client) UpdateFlowJSON(ctx context.Context, flowID string, flowJSON FlowJSON) error {
    url := fmt.Sprintf("%s/%s/assets", c.baseURL, flowID)
    
    payload := map[string]interface{}{
        "name":                "flow.json",
        "asset_type":          "FLOW_JSON",
        "messaging_product":   "whatsapp",
        "flow_json":           flowJSON,
    }
    
    // ... make POST request
}

func (c *Client) PublishFlow(ctx context.Context, flowID string) error {
    url := fmt.Sprintf("%s/%s/publish", c.baseURL, flowID)
    
    // ... make POST request
}

func (c *Client) DeprecateFlow(ctx context.Context, flowID string) error {
    url := fmt.Sprintf("%s/%s/deprecate", c.baseURL, flowID)
    
    // ... make POST request
}

func (c *Client) SendFlowMessage(ctx context.Context, to, flowID, screenID, ctaText string, flowToken string, data map[string]interface{}) (*MessageResponse, error) {
    payload := map[string]interface{}{
        "messaging_product": "whatsapp",
        "to":                to,
        "type":              "interactive",
        "interactive": map[string]interface{}{
            "type": "flow",
            "header": map[string]string{
                "type": "text",
                "text": "Complete the form",
            },
            "body": map[string]string{
                "text": "Please fill in your details",
            },
            "action": map[string]interface{}{
                "name": "flow",
                "parameters": map[string]interface{}{
                    "flow_message_version": "3",
                    "flow_token":           flowToken,
                    "flow_id":              flowID,
                    "flow_cta":             ctaText,
                    "flow_action":          "navigate",
                    "flow_action_payload": map[string]interface{}{
                        "screen": screenID,
                        "data":   data,
                    },
                },
            },
        },
    }
    
    return c.sendMessage(ctx, payload)
}
```

#### Checklist

- [ ] Create flow funciona
- [ ] Update flow JSON funciona
- [ ] Publish flow funciona
- [ ] Deprecate flow funciona
- [ ] Send flow message funciona
- [ ] Flow responses (nfm_reply) são processadas
- [ ] UI para gerenciar lifecycle

---

## FASE 4: Commerce Suite (Semanas 19-24)

### SPRINT 4.1: Catalog Integration (Semanas 19-20)

#### Documentação
- **Commerce Manager**: https://www.facebook.com/business/help/commercemanager
- **Catalog API**: https://developers.facebook.com/docs/marketing-api/catalog
- **Send Product Messages**: https://developers.facebook.com/docs/whatsapp/cloud-api/guides/sell-products-and-services

#### Código

```go
// internal/whatsapp/client/catalog.go
package client

type CatalogProduct struct {
    ID                string  `json:"id"`
    Name              string  `json:"name"`
    Description       string  `json:"description"`
    Price             float64 `json:"price"`
    Currency          string  `json:"currency"`
    ImageURL          string  `json:"image_url"`
    Availability      string  `json:"availability"` // in stock, out of stock
    RetailerID        string  `json:"retailer_id"`
}

func (c *Client) GetCatalog(ctx context.Context) (*Catalog, error) {
    url := fmt.Sprintf("%s/%s/owned_product_catalogs", c.baseURL, c.businessID)
    
    // ... make GET request
}

func (c *Client) SendSingleProduct(ctx context.Context, to, catalogID, productRetailerID string) (*MessageResponse, error) {
    payload := map[string]interface{}{
        "messaging_product": "whatsapp",
        "to":                to,
        "type":              "interactive",
        "interactive": map[string]interface{}{
            "type": "product",
            "body": map[string]string{
                "text": "Check out this product",
            },
            "action": map[string]interface{}{
                "catalog_id":           catalogID,
                "product_retailer_id":  productRetailerID,
            },
        },
    }
    
    return c.sendMessage(ctx, payload)
}

func (c *Client) SendMultiProduct(ctx context.Context, to, catalogID, headerText, bodyText string, sections []ProductSection) (*MessageResponse, error) {
    payload := map[string]interface{}{
        "messaging_product": "whatsapp",
        "to":                to,
        "type":              "interactive",
        "interactive": map[string]interface{}{
            "type": "product_list",
            "header": map[string]string{
                "type": "text",
                "text": headerText,
            },
            "body": map[string]string{
                "text": bodyText,
            },
            "action": map[string]interface{}{
                "catalog_id": catalogID,
                "sections":   sections,
            },
        },
    }
    
    return c.sendMessage(ctx, payload)
}

type ProductSection struct {
    Title       string   `json:"title"`
    ProductItems []ProductItem `json:"product_items"` // Max 30 products
}

type ProductItem struct {
    ProductRetailerID string `json:"product_retailer_id"`
}

func (c *Client) SendCatalog(ctx context.Context, to, catalogID, bodyText string) (*MessageResponse, error) {
    payload := map[string]interface{}{
        "messaging_product": "whatsapp",
        "to":                to,
        "type":              "interactive",
        "interactive": map[string]interface{}{
            "type": "catalog_message",
            "body": map[string]string{
                "text": bodyText,
            },
            "action": map[string]string{
                "name":       "catalog_message",
                "catalog_id": catalogID,
            },
        },
    }
    
    return c.sendMessage(ctx, payload)
}
```

#### Checklist

- [ ] Integração com Facebook Commerce Manager
- [ ] Sync de catálogo funciona
- [ ] Enviar single product funciona
- [ ] Enviar multi-product funciona (até 30)
- [ ] Enviar catalog message funciona
- [ ] Order webhooks são processados
- [ ] UI para selecionar produtos do catálogo

---

### SPRINT 4.2: Order Management (Semanas 21-22)

#### Código

```go
// internal/whatsapp/webhook/orders.go
package webhook

type OrderMessage struct {
    CatalogID    string        `json:"catalog_id"`
    Text         string        `json:"text"`
    ProductItems []OrderItem   `json:"product_items"`
}

type OrderItem struct {
    ProductRetailerID string  `json:"product_retailer_id"`
    Quantity          int     `json:"quantity"`
    ItemPrice         float64 `json:"item_price"`
    Currency          string  `json:"currency"`
}

func (h *WebhookHandler) handleOrderMessage(msg Message) {
    if msg.Order == nil {
        return
    }
    
    // Extract order details
    order := Order{
        CatalogID:    msg.Order.CatalogID,
        CustomerID:   msg.From,
        Items:        msg.Order.ProductItems,
        TotalAmount:  calculateTotal(msg.Order.ProductItems),
        Status:       "pending",
        CreatedAt:    time.Now(),
    }
    
    // Save to database
    h.orderRepo.Create(order)
    
    // Dispatch event
    h.dispatcher.Publish("whatsapp.order.created", order)
    
    // Send confirmation
    h.sendOrderConfirmation(msg.From, order)
}

func calculateTotal(items []OrderItem) float64 {
    var total float64
    for _, item := range items {
        total += item.ItemPrice * float64(item.Quantity)
    }
    return total
}
```

#### Checklist

- [ ] Order messages são capturadas
- [ ] Order details parseados corretamente
- [ ] Order persistido no DB
- [ ] Confirmação de pedido enviada
- [ ] UI para visualizar pedidos
- [ ] Status tracking implementado

---

### SPRINT 4.3: Cart Handling (Semanas 23-24)

#### Código

```go
// internal/whatsapp/commerce/cart.go
package commerce

type Cart struct {
    ID         string      `json:"id"`
    CustomerID string      `json:"customer_id"`
    Items      []CartItem  `json:"items"`
    CreatedAt  time.Time   `json:"created_at"`
    UpdatedAt  time.Time   `json:"updated_at"`
}

type CartItem struct {
    ProductRetailerID string  `json:"product_retailer_id"`
    ProductName       string  `json:"product_name"`
    Quantity          int     `json:"quantity"`
    Price             float64 `json:"price"`
    Currency          string  `json:"currency"`
}

type CartService struct {
    repo *CartRepository
}

func (s *CartService) AddToCart(customerID, productID string, quantity int) error {
    cart, _ := s.repo.GetByCustomer(customerID)
    if cart == nil {
        cart = &Cart{
            ID:         generateID(),
            CustomerID: customerID,
            Items:      []CartItem{},
            CreatedAt:  time.Now(),
        }
    }
    
    // Add or update item
    found := false
    for i, item := range cart.Items {
        if item.ProductRetailerID == productID {
            cart.Items[i].Quantity += quantity
            found = true
            break
        }
    }
    
    if !found {
        product := s.getProduct(productID)
        cart.Items = append(cart.Items, CartItem{
            ProductRetailerID: productID,
            ProductName:       product.Name,
            Quantity:          quantity,
            Price:             product.Price,
            Currency:          product.Currency,
        })
    }
    
    cart.UpdatedAt = time.Now()
    return s.repo.Save(cart)
}

func (s *CartService) SendCartSummary(ctx context.Context, customerID string) error {
    cart, _ := s.repo.GetByCustomer(customerID)
    if cart == nil || len(cart.Items) == 0 {
        return errors.New("cart is empty")
    }
    
    // Build multi-product message
    sections := []ProductSection{
        {
            Title: "Your Cart",
            ProductItems: make([]ProductItem, len(cart.Items)),
        },
    }
    
    for i, item := range cart.Items {
        sections[0].ProductItems[i] = ProductItem{
            ProductRetailerID: item.ProductRetailerID,
        }
    }
    
    total := s.calculateTotal(cart)
    bodyText := fmt.Sprintf("Cart total: %s %.2f", cart.Items[0].Currency, total)
    
    return s.whatsappClient.SendMultiProduct(ctx, customerID, catalogID, "Your Cart", bodyText, sections)
}
```

#### Checklist

- [ ] Add to cart funciona
- [ ] Remove from cart funciona
- [ ] View cart funciona
- [ ] Cart summary enviado via WhatsApp
- [ ] Checkout flow implementado
- [ ] Cart abandonment tracking

---

## FASE 5: Analytics & Payments (Semanas 25-28)

### SPRINT 5.1: Analytics Nativos (Semanas 25-26)

#### Documentação
- **Conversation Analytics**: https://developers.facebook.com/docs/graph-api/reference/whatsapp-business-account/conversation_analytics

#### Código

```go
// internal/whatsapp/client/analytics.go
package client

type ConversationAnalytics struct {
    DataPoints []DataPoint `json:"data"`
    Paging     *Paging     `json:"paging,omitempty"`
}

type DataPoint struct {
    Start               string            `json:"start"`
    End                 string            `json:"end"`
    ConversationType    string            `json:"conversation_type"` // regular, free_entry_point, free_tier
    ConversationDirection string          `json:"conversation_direction"` // business_initiated, user_initiated
    Country             string            `json:"country,omitempty"`
    PhoneNumber         string            `json:"phone,omitempty"`
    Cost                float64           `json:"cost"`
    Conversations       int               `json:"conversations"`
}

func (c *Client) GetAnalytics(ctx context.Context, start, end time.Time, granularity string, dimensions []string) (*ConversationAnalytics, error) {
    url := fmt.Sprintf("%s/%s/conversation_analytics", c.baseURL, c.wabaID)
    
    params := map[string]string{
        "start":       strconv.FormatInt(start.Unix(), 10),
        "end":         strconv.FormatInt(end.Unix(), 10),
        "granularity": granularity, // HALF_HOUR, DAY, MONTH
        "dimensions":  strings.Join(dimensions, ","), // conversation_type, conversation_direction, country, phone
    }
    
    // ... make GET request with params
}

// Analytics Dashboard Data
type AnalyticsDashboard struct {
    TotalConversations int                           `json:"total_conversations"`
    TotalCost          float64                       `json:"total_cost"`
    ByType             map[string]int                `json:"by_type"`
    ByDirection        map[string]int                `json:"by_direction"`
    ByCountry          map[string]int                `json:"by_country"`
    Timeline           []TimelinePoint               `json:"timeline"`
    CostBreakdown      map[string]float64            `json:"cost_breakdown"`
}

type TimelinePoint struct {
    Timestamp      time.Time `json:"timestamp"`
    Conversations  int       `json:"conversations"`
    Cost           float64   `json:"cost"`
}

func (s *AnalyticsService) BuildDashboard(ctx context.Context, start, end time.Time) (*AnalyticsDashboard, error) {
    // Fetch analytics with multiple dimension combinations
    data, err := s.client.GetAnalytics(ctx, start, end, "DAY", []string{"conversation_type", "conversation_direction", "country"})
    if err != nil {
        return nil, err
    }
    
    dashboard := &AnalyticsDashboard{
        ByType:      make(map[string]int),
        ByDirection: make(map[string]int),
        ByCountry:   make(map[string]int),
        CostBreakdown: make(map[string]float64),
        Timeline:    []TimelinePoint{},
    }
    
    for _, dp := range data.DataPoints {
        dashboard.TotalConversations += dp.Conversations
        dashboard.TotalCost += dp.Cost
        
        dashboard.ByType[dp.ConversationType] += dp.Conversations
        dashboard.ByDirection[dp.ConversationDirection] += dp.Conversations
        dashboard.ByCountry[dp.Country] += dp.Conversations
        
        dashboard.CostBreakdown[dp.ConversationType] += dp.Cost
        
        // ... build timeline
    }
    
    return dashboard, nil
}
```

#### Checklist

- [ ] Fetch analytics via API funciona
- [ ] Analytics dashboard implementado
- [ ] Breakdown por tipo/direção/país
- [ ] Timeline de conversas
- [ ] Cost tracking
- [ ] Export para CSV
- [ ] Gráficos interativos (Chart.js/Recharts)

---

### SPRINT 5.2: Payments Integration (Semanas 27-28)

**NOTA:** Payments estão disponíveis apenas em India e Brasil

#### Documentação
- **Payments Overview**: https://developers.facebook.com/docs/whatsapp/cloud-api/guides/set-up-payment-method
- **Send Payment Request**: https://developers.facebook.com/docs/whatsapp/cloud-api/reference/messages#payment

#### Código

```go
// internal/whatsapp/client/payments.go
package client

type PaymentConfiguration struct {
    Provider       string `json:"provider"` // razorpay, payu (India)
    MerchantID     string `json:"merchant_id"`
    Currency       string `json:"currency"` // INR, BRL
}

type PaymentOrder struct {
    ReferenceID    string  `json:"reference_id"`
    Type           string  `json:"type"` // digital-goods, physical-goods
    TotalAmount    float64 `json:"total_amount"`
    Currency       string  `json:"currency"`
    OrderDetails   *OrderDetails `json:"order,omitempty"`
}

type OrderDetails struct {
    Status      string      `json:"status"` // pending, processing, partially_shipped, shipped, completed, canceled
    Subtotal    float64     `json:"subtotal"`
    Tax         float64     `json:"tax,omitempty"`
    Shipping    float64     `json:"shipping,omitempty"`
    Discount    float64     `json:"discount,omitempty"`
    Items       []OrderItem `json:"items"`
}

func (c *Client) SendPaymentRequest(ctx context.Context, to string, order PaymentOrder) (*MessageResponse, error) {
    payload := map[string]interface{}{
        "messaging_product": "whatsapp",
        "to":                to,
        "type":              "interactive",
        "interactive": map[string]interface{}{
            "type": "order_details",
            "header": map[string]string{
                "type": "text",
                "text": "Order Summary",
            },
            "body": map[string]interface{}{
                "text": fmt.Sprintf("Please complete payment for order #%s", order.ReferenceID),
            },
            "action": map[string]interface{}{
                "name": "review_and_pay",
                "parameters": map[string]interface{}{
                    "reference_id": order.ReferenceID,
                    "type":         order.Type,
                    "payment_settings": []map[string]interface{}{
                        {
                            "type": "payment_gateway",
                            "payment_gateway": map[string]string{
                                "type":         "razorpay", // or "payu"
                                "configuration_name": c.paymentConfig.MerchantID,
                            },
                        },
                    },
                    "total_amount": map[string]interface{}{
                        "value":  int64(order.TotalAmount * 100), // in paise/centavos
                        "offset": 100,
                    },
                    "order": order.OrderDetails,
                },
            },
        },
    }
    
    return c.sendMessage(ctx, payload)
}

func (h *WebhookHandler) handlePaymentUpdate(msg Message) {
    if msg.Order == nil {
        return
    }
    
    payment := Payment{
        ReferenceID: msg.Order.CatalogID, // Actually reference_id in payment context
        CustomerID:  msg.From,
        Status:      msg.Order.Status,
        Amount:      msg.Order.TotalAmount,
        UpdatedAt:   time.Now(),
    }
    
    h.paymentRepo.Update(payment)
    h.dispatcher.Publish("whatsapp.payment.updated", payment)
}
```

#### Checklist

- [ ] Payment configuration setup (India/Brazil)
- [ ] Send payment request funciona
- [ ] Payment status updates capturados
- [ ] Payment confirmation enviada
- [ ] Refund handling
- [ ] Payment dashboard

---

## FASE 6: Features Premium (Semanas 29-30)

### SPRINT 6.1: Business Calling API (Semana 29)

#### Documentação
- **Voice Calling**: https://developers.facebook.com/docs/whatsapp/business-management-api/calling

#### Código

```go
// internal/whatsapp/client/calling.go
package client

type CallType string

const (
    CallInbound  CallType = "inbound"
    CallOutbound CallType = "outbound"
)

func (c *Client) InitiateCall(ctx context.Context, to string) (*CallResponse, error) {
    url := fmt.Sprintf("%s/%s/calls", c.baseURL, c.phoneNumberID)
    
    payload := map[string]string{
        "to": to,
    }
    
    // ... make POST request
}

type CallEvent struct {
    CallID      string    `json:"call_id"`
    From        string    `json:"from"`
    To          string    `json:"to"`
    Type        CallType  `json:"type"`
    Status      string    `json:"status"` // ringing, answered, rejected, missed, ended
    StartTime   time.Time `json:"start_time,omitempty"`
    EndTime     time.Time `json:"end_time,omitempty"`
    Duration    int       `json:"duration,omitempty"` // seconds
}

func (h *WebhookHandler) handleCallEvent(value map[string]interface{}) {
    // Process call events
    event := parseCallEvent(value)
    
    h.callRepo.Save(event)
    h.dispatcher.Publish("whatsapp.call.event", event)
}
```

#### Checklist

- [ ] Outbound call funciona
- [ ] Inbound call handling
- [ ] Call status tracking
- [ ] Call logs persistidos
- [ ] UI para chamadas

---

### SPRINT 6.2: Click-to-WhatsApp Ads & Advanced Features (Semana 30)

#### Documentação
- **CTWA Ads**: https://developers.facebook.com/docs/whatsapp/click-to-whatsapp-ads
- **Referral Messages**: https://developers.facebook.com/docs/whatsapp/cloud-api/webhooks/payload-examples#referral-messages

#### Código

```go
// internal/whatsapp/webhook/referral.go
package webhook

type ReferralMessage struct {
    SourceURL   string `json:"source_url"`
    SourceID    string `json:"source_id"`
    SourceType  string `json:"source_type"` // ad, post
    Headline    string `json:"headline,omitempty"`
    Body        string `json:"body,omitempty"`
    MediaType   string `json:"media_type,omitempty"` // image, video
    ImageURL    string `json:"image_url,omitempty"`
    VideoURL    string `json:"video_url,omitempty"`
    ThumbnailURL string `json:"thumbnail_url,omitempty"`
}

func (h *WebhookHandler) handleReferral(msg Message) {
    if msg.Referral == nil {
        return
    }
    
    // Track CTWA ad conversion
    conversion := AdConversion{
        CustomerID:  msg.From,
        SourceURL:   msg.Referral.SourceURL,
        SourceID:    msg.Referral.SourceID,
        SourceType:  msg.Referral.SourceType,
        AdHeadline:  msg.Referral.Headline,
        ConvertedAt: time.Now(),
    }
    
    h.conversionRepo.Save(conversion)
    
    // 72-hour free messaging window starts
    h.openFreeWindow(msg.From, 72*time.Hour)
    
    h.dispatcher.Publish("whatsapp.ctwa.conversion", conversion)
}
```

#### Checklist

- [ ] Referral messages capturados
- [ ] CTWA tracking implementado
- [ ] 72h free window aplicada
- [ ] Attribution reporting
- [ ] ROI dashboard para ads

---

## Recursos Adicionais e Ferramentas

### Postman Collection

Importe a collection oficial da Meta:
https://www.postman.com/meta/workspace/whatsapp-business-platform

### Ferramentas de Teste

```bash
# WhatsApp Business API Test Number
# Configure em: https://developers.facebook.com/apps/

# Webhook Tester Local
ngrok http 3000

# Flow Playground
# https://developers.facebook.com/docs/whatsapp/flows/playground
```

### Libraries Go Úteis

```go
// go.mod additions
require (
    github.com/nats-io/nats.go v1.31.0
    go.uber.org/zap v1.26.0
    github.com/golang-jwt/jwt/v5 v5.2.0
    github.com/google/uuid v1.5.0
)
```

---

## Conclusão e Próximos Passos

Este blueprint fornece um roteiro COMPLETO e EXECUTÁVEL para implementar 100% da WhatsApp Cloud API no Linktor.

### Timeline Resumido

- **Semanas 1-6:** Fundação (webhooks, reactions, interactive, templates)
- **Semanas 7-12:** Templates avançados (carousel, auth, LTO, coupon)
- **Semanas 13-18:** WhatsApp Flows (encryption, builder, lifecycle)
- **Semanas 19-24:** Commerce (catalog, orders, cart)
- **Semanas 25-28:** Analytics & Payments
- **Semanas 29-30:** Calling & CTWA

### Resultado Esperado

Ao final das 30 semanas:

✅ **Único projeto open source** com cobertura COMPLETA da API oficial  
✅ **14 tipos de mensagens** suportados  
✅ **WhatsApp Flows Engine** funcional  
✅ **Commerce Suite** completo  
✅ **Analytics nativos**  
✅ **Payments** (India/Brasil)  
✅ **Business Calling**  

### Diferencial Competitivo

🎯 VendaX.ai será o **primeiro CRM AFV** com:
- WhatsApp Flows para lead gen
- Catálogo de produtos integrado
- Analytics ROI nativos
- Cobertura 100% da API oficial

**O mercado está esperando por isso. É hora de construir! 🚀**
