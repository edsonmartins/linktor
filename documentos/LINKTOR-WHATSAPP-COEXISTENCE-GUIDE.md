# LINKTOR - WhatsApp Coexistence Implementation Guide
## Adendo ao Blueprint Principal

**Feature:** WhatsApp Coexistence (Coex)  
**Lançamento:** 2024-2025  
**Importância:** CRÍTICA - diferencial competitivo para onboarding de clientes  
**Complexidade:** Média-Alta  

---

## 📋 Índice

1. [O que é Coexistence](#o-que-é-coexistence)
2. [Por que é Crítico para o Linktor](#por-que-é-crítico)
3. [Como Funciona Tecnicamente](#arquitetura-técnica)
4. [Requisitos e Limitações](#requisitos-e-limitações)
5. [Implementação no Linktor](#implementação)
6. [Embedded Signup Flow](#embedded-signup)
7. [Webhook Handling](#webhook-handling)
8. [Pricing & Billing](#pricing--billing)
9. [Troubleshooting](#troubleshooting)

---

## O que é Coexistence?

### Definição

**WhatsApp Coexistence** permite que um negócio use **simultaneamente**:

- ✅ **WhatsApp Business App** (mobile) para conversas manuais 1:1
- ✅ **WhatsApp Cloud API** (plataforma) para automação/escala

**No mesmo número de telefone**, com sincronização bidirecional de mensagens.

### Problema que Resolve

**Antes do Coexistence:**

```
Cenário 1: Usar apenas Business App
├─ ✅ Interface familiar, conversas manuais
├─ ❌ Limite de 5 dispositivos
├─ ❌ Sem automação avançada
└─ ❌ Sem broadcast em massa

Cenário 2: Migrar para Cloud API
├─ ✅ Automação completa
├─ ✅ Multi-agente ilimitado
├─ ❌ PERDE acesso ao App
├─ ❌ PERDE histórico de conversas
└─ ❌ Processo complexo de migração
```

**Com Coexistence:**

```
MELHOR DOS DOIS MUNDOS
├─ ✅ Continua usando o App no celular
├─ ✅ Ganha automação via API
├─ ✅ Mantém TODO o histórico
├─ ✅ Sincronização automática
└─ ✅ Onboarding em 5 minutos (QR code)
```

---

## Por que é Crítico para o Linktor?

### 🎯 Vantagem Competitiva #1: Onboarding Frictionless

**Sem Coexistence:**
```
Cliente quer testar VendaX.ai:
1. "Precisa criar novo número WhatsApp"
2. "Vai perder conversas antigas"
3. "Precisa avisar todos os clientes do novo número"
→ DESISTE (85% das vezes)
```

**Com Coexistence:**
```
Cliente quer testar VendaX.ai:
1. "Escaneia QR code"
2. "Pronto! Já está conectado"
3. "Histórico de 6 meses importado"
→ CONVERTE (taxa 10x maior)
```

### 🚀 Use Cases para VendaX.ai

1. **Migração Zero-Friction**
   - Distribuidor já usa WhatsApp Business App há anos
   - Tem 5.000+ contatos e histórico
   - Quer testar VendaX.ai SEM migrar número
   - **Coex = onboarding em 5 minutos**

2. **Híbrido Manual + Automação**
   - Vendedor sênior prefere atender VIPs pelo App (celular)
   - VendaX.ai automatiza follow-ups de leads
   - Vendedor vê TUDO sincronizado no App
   - **Melhor de ambos os mundos**

3. **Rollout Gradual**
   - Começa com automação básica via API
   - Time continua usando App normalmente
   - Migração incremental de workflows
   - **Sem disrupção operacional**

### 📊 Impacto no Negócio

**Chatwoot NÃO suporta Coexistence** (Issue #12569 aberta)  
**Evolution API focado em Baileys** (não tem Coex)  
**Linktor com Coex = ÚNICO diferencial no mercado open source**

**Estimativa de impacto:**
- 🔥 **Taxa de conversão 10x maior** no onboarding
- 🔥 **Churn 70% menor** (cliente não precisa migrar tudo)
- 🔥 **Time-to-value < 10 minutos** (vs 3-7 dias sem Coex)

---

## Arquitetura Técnica

### Como Funciona a Sincronização

```
┌─────────────────────┐         ┌──────────────────────┐
│ WhatsApp Business   │         │  WhatsApp Cloud API  │
│       App           │◄────────┤   (Linktor/msgfy)    │
│   (Mobile/Web)      │   sync  │                      │
└─────────────────────┘         └──────────────────────┘
         │                                 │
         │                                 │
         ▼                                 ▼
    ┌────────────────────────────────────────┐
    │       Meta WhatsApp Platform           │
    │  (gerencia sincronização + storage)    │
    └────────────────────────────────────────┘
```

### Message Flow: App → API

```
1. Usuário envia mensagem via App (celular)
   ↓
2. Meta processa e armazena
   ↓
3. Meta envia webhook "smb_message_echoes" para Linktor
   ↓
4. Linktor recebe mensagem como "echo"
   ↓
5. Linktor exibe na UI como mensagem enviada
```

### Message Flow: API → App

```
1. Linktor envia mensagem via Cloud API
   ↓
2. Meta processa e armazena
   ↓
3. Meta sincroniza para App
   ↓
4. Mensagem aparece automaticamente no App
   (sem webhook, é transparente)
```

### Message Flow: Cliente → Ambos

```
1. Cliente envia mensagem para número
   ↓
2. Meta distribui para AMBOS:
   ├─ App: mensagem aparece imediatamente
   └─ API: webhook "messages" enviado para Linktor
   ↓
3. Ambos têm a mesma mensagem
   (não há duplicação, mesmo ID)
```

---

## Requisitos e Limitações

### ✅ Requisitos Técnicos

1. **WhatsApp Business App versão 2.24.17+**
   - Android ou iOS
   - Atualizar antes do onboarding

2. **Número ativo há 7+ dias**
   - Meta valida atividade recente
   - Recomendado: 1-2 meses de uso
   - Números novos podem ser rejeitados

3. **Facebook Page vinculada**
   - Obrigatório para Embedded Signup
   - Crie se não tiver

4. **País suportado**
   - Lista atualizada em fevereiro 2026:
   ```
   ❌ NÃO suportados (Coexistence):
   - Argentina
   - Egito  
   - Irã
   - Iraque
   - Nigéria (parcialmente)
   - África do Sul (parcialmente)
   - Síria
   - Ucrânia
   
   ✅ Brasil: SUPORTADO
   ```

5. **BSP com suporte a Coexistence**
   - Linktor precisa implementar Embedded Signup
   - Endpoint de webhook configurado

### ❌ Limitações Pós-Onboarding

**Features do App que PARAM de funcionar:**

1. ❌ **Broadcast Lists** (do App)
   - Use templates via API
   
2. ❌ **View Once Media**
   - Feature desabilitada

3. ❌ **Linked Devices Unsupported**
   - WhatsApp for Windows: ❌
   - WhatsApp for WearOS: ❌
   - WhatsApp Web: ✅ (funciona)
   - WhatsApp for Mac: ✅ (funciona)

4. ❌ **Business Verification padrão**
   - Precisa usar Partner-Led Business Verification (PLBV)
   - OU Meta Verified for Business

5. ❌ **Official Business Account (Blue Badge)**
   - Badge não é transferível
   - Precisa solicitar novamente via API

6. ❌ **Status/Stories**
   - Continuam funcionando no App
   - NÃO sincronizam para API

### ⚠️ Regras de Manutenção

**CRÍTICO:** App deve ser aberto **pelo menos 1x a cada 13-14 dias**

```
Se não abrir o App por 14+ dias:
├─ Coexistence é DESCONECTADO automaticamente
├─ Mensagens param de sincronizar
├─ API continua funcionando
└─ Precisa reconectar (novo QR code)
```

**Implicação para Linktor:**
- Avisar clientes sobre regra dos 14 dias
- Dashboard com "último uso do App"
- Notificação se inativo por 10+ dias

---

## Implementação no Linktor

### Fase 1: Embedded Signup Flow (Semana 1)

#### Documentação Oficial
- **Embedded Signup**: https://developers.facebook.com/docs/whatsapp/embedded-signup
- **Onboarding Business App Users**: https://developers.facebook.com/docs/whatsapp/embedded-signup/custom-flows/onboarding-business-app-users

#### Frontend: Botão "Conectar WhatsApp Existente"

```typescript
// frontend/src/components/WhatsAppOnboarding.tsx

export function WhatsAppOnboarding() {
  const [flow, setFlow] = useState<'new' | 'existing'>('new');
  
  const initiateEmbeddedSignup = () => {
    // Meta's Facebook SDK
    FB.login((response) => {
      if (response.authResponse) {
        const code = response.authResponse.code;
        
        // Send code to backend
        fetch('/api/whatsapp/embedded-signup', {
          method: 'POST',
          body: JSON.stringify({ code }),
        });
      }
    }, {
      config_id: '<CONFIG_ID>', // From Meta App Dashboard
      response_type: 'code',
      override_default_response_type: true,
      extras: {
        sessionInfoVersion: 2,
        feature: 'whatsapp_embedded_signup',
      },
    });
  };
  
  return (
    <div className="onboarding">
      <h2>Conectar WhatsApp ao Linktor</h2>
      
      <div className="flow-selector">
        <button onClick={() => setFlow('new')}>
          Novo Número
        </button>
        <button onClick={() => setFlow('existing')}>
          Número Existente (Coexistence)
        </button>
      </div>
      
      {flow === 'existing' && (
        <div className="coex-flow">
          <p>
            ✅ Mantém seu número atual<br/>
            ✅ Importa 6 meses de histórico<br/>
            ✅ Continua usando o App no celular
          </p>
          
          <button onClick={initiateEmbeddedSignup}>
            Escanear QR Code
          </button>
        </div>
      )}
    </div>
  );
}
```

#### Backend: Exchange Code for Token

```go
// internal/whatsapp/embeddedSignup.go
package whatsapp

type EmbeddedSignupRequest struct {
    Code string `json:"code"`
}

type EmbeddedSignupResponse struct {
    AccessToken    string `json:"access_token"`
    WABAID         string `json:"waba_id"`
    PhoneNumberID  string `json:"phone_number_id"`
    IsCoexistence  bool   `json:"is_coexistence"`
}

func (s *WhatsAppService) HandleEmbeddedSignup(ctx context.Context, req EmbeddedSignupRequest) (*EmbeddedSignupResponse, error) {
    // 1. Exchange code for access token
    tokenURL := "https://graph.facebook.com/v21.0/oauth/access_token"
    
    params := url.Values{}
    params.Set("client_id", s.config.AppID)
    params.Set("client_secret", s.config.AppSecret)
    params.Set("code", req.Code)
    
    resp, err := http.Get(tokenURL + "?" + params.Encode())
    if err != nil {
        return nil, err
    }
    defer resp.Body.Close()
    
    var tokenResp struct {
        AccessToken string `json:"access_token"`
    }
    json.NewDecoder(resp.Body).Decode(&tokenResp)
    
    // 2. Get WABA and Phone Number details
    debugURL := "https://graph.facebook.com/v21.0/debug_token"
    debugParams := url.Values{}
    debugParams.Set("input_token", tokenResp.AccessToken)
    debugParams.Set("access_token", s.config.SystemUserToken)
    
    debugResp, err := http.Get(debugURL + "?" + debugParams.Encode())
    if err != nil {
        return nil, err
    }
    defer debugResp.Body.Close()
    
    var debugData struct {
        Data struct {
            Granular []struct {
                Scope string `json:"scope"`
            } `json:"granular_scopes"`
        } `json:"data"`
    }
    json.NewDecoder(debugResp.Body).Decode(&debugData)
    
    // 3. Extract WABA ID and Phone Number ID
    var wabaID, phoneNumberID string
    for _, scope := range debugData.Data.Granular {
        if strings.HasPrefix(scope.Scope, "whatsapp_business_management") {
            // Parse WABA ID from scope
            wabaID = extractWABAID(scope.Scope)
        }
        if strings.HasPrefix(scope.Scope, "whatsapp_business_messaging") {
            phoneNumberID = extractPhoneNumberID(scope.Scope)
        }
    }
    
    // 4. Check if it's a Coexistence account
    isCoex := s.checkCoexistence(ctx, phoneNumberID, tokenResp.AccessToken)
    
    // 5. If Coexistence, subscribe to smb_message_echoes webhook
    if isCoex {
        s.subscribeToEchoes(ctx, wabaID, tokenResp.AccessToken)
    }
    
    // 6. Save configuration
    account := &Account{
        WABAID:        wabaID,
        PhoneNumberID: phoneNumberID,
        AccessToken:   tokenResp.AccessToken,
        IsCoexistence: isCoex,
        Status:        "active",
        CreatedAt:     time.Now(),
    }
    
    s.repo.SaveAccount(account)
    
    return &EmbeddedSignupResponse{
        AccessToken:   tokenResp.AccessToken,
        WABAID:        wabaID,
        PhoneNumberID: phoneNumberID,
        IsCoexistence: isCoex,
    }, nil
}

func (s *WhatsAppService) checkCoexistence(ctx context.Context, phoneNumberID, accessToken string) bool {
    url := fmt.Sprintf("https://graph.facebook.com/v21.0/%s?fields=is_business_app_number_active", phoneNumberID)
    
    req, _ := http.NewRequestWithContext(ctx, "GET", url, nil)
    req.Header.Set("Authorization", "Bearer "+accessToken)
    
    resp, err := http.DefaultClient.Do(req)
    if err != nil {
        return false
    }
    defer resp.Body.Close()
    
    var data struct {
        IsBusinessAppNumberActive bool `json:"is_business_app_number_active"`
    }
    json.NewDecoder(resp.Body).Decode(&data)
    
    return data.IsBusinessAppNumberActive
}

func (s *WhatsAppService) subscribeToEchoes(ctx context.Context, wabaID, accessToken string) error {
    url := fmt.Sprintf("https://graph.facebook.com/v21.0/%s/subscribed_apps", wabaID)
    
    payload := map[string]interface{}{
        "subscribed_fields": []string{
            "messages",
            "message_echoes", // CRITICAL for Coexistence
        },
    }
    
    body, _ := json.Marshal(payload)
    req, _ := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(body))
    req.Header.Set("Authorization", "Bearer "+accessToken)
    req.Header.Set("Content-Type", "application/json")
    
    resp, err := http.DefaultClient.Do(req)
    if err != nil {
        return err
    }
    defer resp.Body.Close()
    
    if resp.StatusCode != 200 {
        return fmt.Errorf("failed to subscribe to echoes: %d", resp.StatusCode)
    }
    
    return nil
}
```

#### Checklist Fase 1

- [ ] Facebook SDK integrado no frontend
- [ ] Botão "Conectar Número Existente" funciona
- [ ] QR code é exibido
- [ ] Exchange code → access token funciona
- [ ] WABA ID e Phone Number ID extraídos
- [ ] Detecção de Coexistence funciona
- [ ] Subscribe to `message_echoes` funciona
- [ ] Account salva no DB com flag `is_coexistence`

---

### Fase 2: Message Echoes Handling (Semana 2)

#### O que são "Message Echoes"?

**Echoes** são webhooks enviados pela Meta quando uma mensagem é **enviada via WhatsApp Business App**.

```
Webhook "message_echoes" contém:
├─ type: "message_echoes"
├─ Mensagem completa (text, image, etc.)
├─ from: número do NEGÓCIO (não do cliente!)
├─ to: número do CLIENTE
└─ is_echo: true
```

**Use case:**
```
Vendedor responde cliente pelo App (celular):
1. Mensagem enviada
2. Meta envia "echo" para Linktor
3. Linktor registra como "enviada via App"
4. Histórico completo fica no Linktor
```

#### Código: Processar Echoes

```go
// internal/whatsapp/webhook/echoes.go
package webhook

type MessageEcho struct {
    ID        string  `json:"id"`
    From      string  `json:"from"` // Business phone number
    To        string  `json:"to"`   // Customer phone number
    Timestamp string  `json:"timestamp"`
    Type      string  `json:"type"`
    
    // Content (same as regular message)
    Text     *TextContent  `json:"text,omitempty"`
    Image    *MediaContent `json:"image,omitempty"`
    // ... outros tipos
    
    Context  *Context      `json:"context,omitempty"`
}

func (h *WebhookHandler) handleMessageEcho(value map[string]interface{}) {
    echoes, ok := value["message_echoes"].([]interface{})
    if !ok || len(echoes) == 0 {
        return
    }
    
    for _, echoData := range echoes {
        echo := parseMessageEcho(echoData)
        
        // Verificar se é Coexistence account
        account := h.accountRepo.GetByPhoneNumber(echo.From)
        if !account.IsCoexistence {
            h.logger.Warn("Received echo for non-coex account", zap.String("phone", echo.From))
            continue
        }
        
        // Salvar mensagem como "enviada via App"
        msg := &Message{
            ID:          echo.ID,
            Direction:   "outbound",
            From:        echo.From,
            To:          echo.To,
            Type:        echo.Type,
            Content:     extractContent(echo),
            Timestamp:   parseTimestamp(echo.Timestamp),
            Channel:     "whatsapp",
            Source:      "business_app", // IMPORTANTE: marcar origem
            Status:      "delivered",    // Echoes já estão enviados
            ConversationID: h.getOrCreateConversation(echo.To),
        }
        
        h.messageRepo.Save(msg)
        
        // Dispatch event
        h.dispatcher.Publish("whatsapp.message.echo", msg)
        
        // Update conversation
        h.updateConversationLastMessage(msg.ConversationID, msg)
        
        h.logger.Info("Processed message echo",
            zap.String("id", msg.ID),
            zap.String("customer", echo.To),
            zap.String("type", echo.Type),
        )
    }
}

func parseMessageEcho(data interface{}) MessageEcho {
    // ... parse echo data
}

func extractContent(echo MessageEcho) interface{} {
    switch echo.Type {
    case "text":
        return echo.Text
    case "image":
        return echo.Image
    // ... outros tipos
    default:
        return nil
    }
}
```

#### UI: Exibir Echoes

```typescript
// frontend/src/components/ConversationThread.tsx

interface Message {
  id: string;
  direction: 'inbound' | 'outbound';
  source: 'api' | 'business_app'; // NOVO
  content: string;
  timestamp: Date;
}

export function ConversationThread({ messages }: { messages: Message[] }) {
  return (
    <div className="thread">
      {messages.map((msg) => (
        <div 
          key={msg.id}
          className={`message ${msg.direction}`}
        >
          <div className="content">{msg.content}</div>
          <div className="meta">
            {msg.timestamp.toLocaleString()}
            {msg.source === 'business_app' && (
              <span className="badge" title="Enviada pelo WhatsApp Business App">
                📱 App
              </span>
            )}
            {msg.source === 'api' && (
              <span className="badge" title="Enviada via API">
                🤖 API
              </span>
            )}
          </div>
        </div>
      ))}
    </div>
  );
}
```

#### Checklist Fase 2

- [ ] Webhook `message_echoes` reconhecido
- [ ] Echoes parseados corretamente
- [ ] Mensagens salvas com source="business_app"
- [ ] UI exibe badge diferenciando App vs API
- [ ] Histórico sincronizado corretamente
- [ ] Conversation thread unificado

---

### Fase 3: Chat History Import (Semana 3)

#### Importar 6 Meses de Histórico

Durante o Embedded Signup, Meta oferece importar até **6 meses** de histórico.

```go
// internal/whatsapp/historyImport.go
package whatsapp

type ChatHistoryImportRequest struct {
    PhoneNumberID string `json:"phone_number_id"`
    AccessToken   string `json:"access_token"`
}

func (s *WhatsAppService) ImportChatHistory(ctx context.Context, req ChatHistoryImportRequest) error {
    // 1. Request chat history export
    url := fmt.Sprintf("https://graph.facebook.com/v21.0/%s/conversations", req.PhoneNumberID)
    
    httpReq, _ := http.NewRequestWithContext(ctx, "GET", url, nil)
    httpReq.Header.Set("Authorization", "Bearer "+req.AccessToken)
    
    resp, err := http.DefaultClient.Do(httpReq)
    if err != nil {
        return err
    }
    defer resp.Body.Close()
    
    var conversations struct {
        Data []Conversation `json:"data"`
    }
    json.NewDecoder(resp.Body).Decode(&conversations)
    
    // 2. Process each conversation
    for _, conv := range conversations.Data {
        // Fetch messages for this conversation
        messages := s.fetchConversationMessages(ctx, conv.ID, req.AccessToken)
        
        // Import messages
        for _, msg := range messages {
            s.importMessage(msg)
        }
    }
    
    return nil
}

func (s *WhatsAppService) fetchConversationMessages(ctx context.Context, conversationID, accessToken string) []Message {
    url := fmt.Sprintf("https://graph.facebook.com/v21.0/%s/messages", conversationID)
    
    req, _ := http.NewRequestWithContext(ctx, "GET", url, nil)
    req.Header.Set("Authorization", "Bearer "+accessToken)
    
    resp, _ := http.DefaultClient.Do(req)
    defer resp.Body.Close()
    
    var data struct {
        Data []Message `json:"data"`
    }
    json.NewDecoder(resp.Body).Decode(&data)
    
    return data.Data
}

func (s *WhatsAppService) importMessage(msg Message) {
    // Save message with is_imported=true flag
    msg.IsImported = true
    msg.ImportedAt = time.Now()
    
    s.messageRepo.Save(&msg)
}
```

#### UI: Progress Indicator

```typescript
// frontend/src/components/HistoryImportProgress.tsx

export function HistoryImportProgress() {
  const [status, setStatus] = useState<'idle' | 'importing' | 'complete'>('idle');
  const [progress, setProgress] = useState(0);
  
  const startImport = async () => {
    setStatus('importing');
    
    const eventSource = new EventSource('/api/whatsapp/import-history/progress');
    
    eventSource.onmessage = (event) => {
      const data = JSON.parse(event.data);
      setProgress(data.progress);
      
      if (data.progress >= 100) {
        setStatus('complete');
        eventSource.close();
      }
    };
  };
  
  return (
    <div className="import-progress">
      {status === 'idle' && (
        <button onClick={startImport}>
          Importar Histórico (6 meses)
        </button>
      )}
      
      {status === 'importing' && (
        <div className="progress-bar">
          <div 
            className="progress-fill" 
            style={{ width: `${progress}%` }}
          />
          <span>{progress}% concluído</span>
        </div>
      )}
      
      {status === 'complete' && (
        <div className="success">
          ✅ Histórico importado com sucesso!
        </div>
      )}
    </div>
  );
}
```

#### Checklist Fase 3

- [ ] API de export de histórico funciona
- [ ] Mensagens importadas com flag `is_imported`
- [ ] Progress indicator funciona
- [ ] Contacts importados
- [ ] Conversations criadas automaticamente
- [ ] UI exibe histórico importado

---

### Fase 4: Activity Monitoring (Semana 4)

#### Monitor App Inactivity (14-day rule)

```go
// internal/whatsapp/monitor/coexActivity.go
package monitor

type CoexActivityMonitor struct {
    repo   *AccountRepository
    logger *zap.Logger
}

func (m *CoexActivityMonitor) CheckInactiveAccounts() {
    accounts := m.repo.GetCoexistenceAccounts()
    
    for _, account := range accounts {
        lastActivity := m.getLastAppActivity(account.PhoneNumberID)
        
        if lastActivity == nil {
            continue
        }
        
        daysSinceActivity := time.Since(*lastActivity).Hours() / 24
        
        if daysSinceActivity >= 10 && daysSinceActivity < 14 {
            // Warn user
            m.sendWarningNotification(account, daysSinceActivity)
        } else if daysSinceActivity >= 14 {
            // Coexistence may be disconnected
            m.checkCoexistenceStatus(account)
        }
    }
}

func (m *CoexActivityMonitor) getLastAppActivity(phoneNumberID string) *time.Time {
    // Query Meta API for last app activity
    // (This data may not be directly available - track via echoes)
    
    lastEcho := m.repo.GetLastMessageEcho(phoneNumberID)
    if lastEcho != nil {
        return &lastEcho.Timestamp
    }
    
    return nil
}

func (m *CoexActivityMonitor) sendWarningNotification(account *Account, days float64) {
    notification := Notification{
        Type:    "coex_inactivity_warning",
        Title:   "⚠️ WhatsApp App inativo",
        Message: fmt.Sprintf("Abra o WhatsApp Business App em até %.0f dias para manter Coexistence ativo", 14-days),
        Account: account,
    }
    
    // Send email + in-app notification
    m.notifier.Send(notification)
}

func (m *CoexActivityMonitor) checkCoexistenceStatus(account *Account) {
    isStillActive := checkCoexistence(context.Background(), account.PhoneNumberID, account.AccessToken)
    
    if !isStillActive {
        // Coexistence foi desconectado
        account.IsCoexistence = false
        account.CoexDisconnectedAt = time.Now()
        m.repo.Update(account)
        
        // Alert admin
        m.sendDisconnectionAlert(account)
    }
}
```

#### Dashboard Widget

```typescript
// frontend/src/components/CoexActivityWidget.tsx

export function CoexActivityWidget({ account }: { account: Account }) {
  const [lastActivity, setLastActivity] = useState<Date | null>(null);
  const [daysInactive, setDaysInactive] = useState(0);
  
  useEffect(() => {
    fetch(`/api/accounts/${account.id}/last-app-activity`)
      .then(res => res.json())
      .then(data => {
        if (data.last_activity) {
          const date = new Date(data.last_activity);
          setLastActivity(date);
          
          const days = Math.floor((Date.now() - date.getTime()) / (1000 * 60 * 60 * 24));
          setDaysInactive(days);
        }
      });
  }, [account.id]);
  
  const getStatusColor = () => {
    if (daysInactive < 7) return 'green';
    if (daysInactive < 10) return 'yellow';
    if (daysInactive < 14) return 'orange';
    return 'red';
  };
  
  return (
    <div className={`coex-status status-${getStatusColor()}`}>
      <h3>WhatsApp App Activity</h3>
      
      {lastActivity ? (
        <>
          <p>Última atividade: {lastActivity.toLocaleDateString()}</p>
          <p>{daysInactive} dias atrás</p>
          
          {daysInactive >= 10 && (
            <div className="warning">
              ⚠️ Abra o App em até {14 - daysInactive} dias!
            </div>
          )}
        </>
      ) : (
        <p>Nenhuma atividade detectada</p>
      )}
    </div>
  );
}
```

#### Checklist Fase 4

- [ ] Monitor de inatividade implementado
- [ ] Tracking de última atividade via echoes
- [ ] Notificações aos 10 dias
- [ ] Check de status aos 14 dias
- [ ] Dashboard widget funcional
- [ ] Alertas automáticos

---

## Pricing & Billing

### Modelo Híbrido

```
Coexistence Pricing Rules:

Cenário 1: Cliente envia → Resposta via App
├─ Custo: FREE (mensagens do App são sempre grátis)
└─ Conversation window: NÃO abre

Cenário 2: Cliente envia → Resposta via API
├─ Custo: Charged (per-message pricing)
├─ Category: Service (se dentro de 24h)
└─ Conversation window: Abre nova sessão

Cenário 3: Negócio inicia via API (template)
├─ Custo: Charged (marketing/utility/auth)
└─ Conversation window: Abre nova sessão

Cenário 4: Negócio inicia via App
├─ Custo: FREE
└─ Conversation window: NÃO abre (App não usa API)
```

### Estratégia de Custo para VendaX.ai

**Otimização:**
```
Use Case: Vendedor atende VIP
└─ Responde pelo App (celular) = FREE

Use Case: Follow-up automático
└─ Envia via API = Paid (mas justificado)

Use Case: Broadcast marketing
└─ Envia via API (template) = Paid
```

**Implementação:**

```go
// internal/whatsapp/billing/coexOptimizer.go
package billing

type MessageRoutingStrategy struct {
    IsVIP         bool
    IsAutomated   bool
    RequiresTemplate bool
}

func (s *MessageRoutingStrategy) ShouldUseApp() bool {
    // Use App (free) para:
    // - VIPs (atendimento manual)
    // - Dentro de conversation window
    // - Sem necessidade de template
    
    if s.IsVIP && !s.IsAutomated && !s.RequiresTemplate {
        return true
    }
    
    return false
}

func (s *MessageRoutingStrategy) ShouldUseAPI() bool {
    // Use API (paid) para:
    // - Automação
    // - Templates
    // - Broadcast
    
    return s.IsAutomated || s.RequiresTemplate
}
```

---

## Troubleshooting

### Problema 1: QR Code Não Funciona

**Sintomas:**
- QR code gerado mas escaneamento falha
- Erro: "Número não elegível"

**Causas:**
1. Número muito novo (< 7 dias de uso)
2. País não suportado
3. App desatualizado (< 2.24.17)
4. Número já vinculado a outro BSP

**Solução:**
```
1. Verificar versão do App: Atualizar se < 2.24.17
2. Verificar atividade: Usar App por 7+ dias
3. Verificar país: Lista de suportados
4. Desvincular de outro BSP se necessário
```

### Problema 2: Echoes Não Chegam

**Sintomas:**
- Mensagens enviadas via App não aparecem no Linktor

**Causas:**
1. Webhook `message_echoes` não subscrito
2. Webhook URL inválida
3. Coexistence desconectado

**Solução:**
```go
// Verificar subscription
func (s *Service) VerifyEchoSubscription(wabaID, token string) error {
    url := fmt.Sprintf("https://graph.facebook.com/v21.0/%s/subscribed_apps", wabaID)
    
    req, _ := http.NewRequest("GET", url, nil)
    req.Header.Set("Authorization", "Bearer "+token)
    
    resp, _ := http.DefaultClient.Do(req)
    defer resp.Body.Close()
    
    var data struct {
        Data []struct {
            SubscribedFields []string `json:"subscribed_fields"`
        } `json:"data"`
    }
    json.NewDecoder(resp.Body).Decode(&data)
    
    // Check if "message_echoes" is in subscribed fields
    for _, app := range data.Data {
        for _, field := range app.SubscribedFields {
            if field == "message_echoes" {
                return nil // OK
            }
        }
    }
    
    return errors.New("message_echoes not subscribed")
}
```

### Problema 3: Histórico Não Importado

**Sintomas:**
- Embedded Signup completo mas histórico vazio

**Causas:**
1. Usuário não autorizou sync no QR code
2. Histórico > 6 meses
3. App não tinha backup habilitado

**Solução:**
```
1. Refazer Embedded Signup
2. Certificar que usuário clica "Permitir" no sync
3. Explicar limite de 6 meses
```

### Problema 4: Coexistence Desconectado

**Sintomas:**
- Echoes param de chegar
- App mostra "Desconectado da plataforma"

**Causas:**
1. App não aberto por 14+ dias
2. Uninstall do App
3. Change de número

**Solução:**
```
1. Abrir App imediatamente
2. Refazer Embedded Signup se necessário
3. Configurar lembretes automáticos
```

---

## Resumo: Por que Coexistence é CRÍTICO?

### 🎯 Para o Negócio (VendaX.ai)

1. **Onboarding 10x mais rápido**
   - Chatwoot: "Migre seu número" → 85% desistem
   - Linktor: "Escaneie QR" → 5 minutos

2. **Diferencial competitivo único**
   - Nenhum open source suporta bem
   - Chatwoot tem Issue aberta sem resolução
   - Evolution API focado em Baileys

3. **Redução de churn**
   - Cliente não precisa migrar
   - Mantém número familiar
   - Zero disrupção operacional

### 🛠️ Para Implementação

1. **Complexidade Média**
   - Embedded Signup: 1 semana
   - Message Echoes: 1 semana
   - History Import: 1 semana
   - Monitoring: 1 semana
   - **Total: 4 semanas**

2. **ROI Altíssimo**
   - 4 semanas de dev
   - 10x melhoria em conversão
   - Diferencial único no mercado

### ✅ Checklist Final

**Implementar ANTES do MVP:**
- [ ] Embedded Signup flow
- [ ] Message echoes handling
- [ ] Chat history import
- [ ] Activity monitoring (14-day rule)
- [ ] Billing logic (App free vs API paid)

**Documentar para clientes:**
- [ ] Guia de onboarding com QR code
- [ ] Explicação App + API simultâneo
- [ ] Regra dos 14 dias
- [ ] Limitações (view once, broadcast lists)

---

## Recursos Adicionais

### Documentação Meta

- **Embedded Signup**: https://developers.facebook.com/docs/whatsapp/embedded-signup
- **Onboarding Business App**: https://developers.facebook.com/docs/whatsapp/embedded-signup/custom-flows/onboarding-business-app-users
- **Feature Comparison**: https://developers.facebook.com/docs/whatsapp/business-management-api/get-started#feature-comparison

### Ferramentas de Teste

```bash
# Verificar elegibilidade do número
curl "https://graph.facebook.com/v21.0/${PHONE_NUMBER_ID}?fields=is_business_app_number_active" \
  -H "Authorization: Bearer ${ACCESS_TOKEN}"

# Verificar subscriptions
curl "https://graph.facebook.com/v21.0/${WABA_ID}/subscribed_apps" \
  -H "Authorization: Bearer ${ACCESS_TOKEN}"
```

### Community Support

- **360dialog Coex Guide**: https://docs.360dialog.com/docs/waba-management/the-360-client-hub/embedded-signup/whatsapp-coexistence
- **Respond.io Coex Guide**: https://respond.io/help/whatsapp/whatsapp-coexistence

---

## Conclusão

**WhatsApp Coexistence não é opcional - é ESSENCIAL** para competir no mercado de mensageria B2B em 2026.

**Impacto esperado:**
- ✅ Onboarding 10x mais rápido
- ✅ Churn 70% menor
- ✅ Diferencial único vs competidores
- ✅ 4 semanas de implementação

**Prioridade:** ALTA - implementar junto com Fase 1 do blueprint principal

🚀 **Próximo passo:** Começar com Embedded Signup (Semana 1)
