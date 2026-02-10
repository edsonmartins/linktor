# Linktor Visual Response Engine (VRE)

## Arquitetura: Template Rendering as a Service

### Conceito

O Linktor oferece um serviço de renderização visual que qualquer bot/agente consome via API REST.
O agente de SAC (ou qualquer outro) **não sabe** como a imagem é gerada — ele chama uma tool
que é um wrapper para o endpoint do Linktor, passando dados estruturados. O Linktor:

1. Recebe o request com `template_id` + `data` (JSON)
2. Carrega o template HTML associado ao tenant/projeto
3. Injeta os dados no template (Go `html/template` ou Handlebars)
4. Renderiza HTML → PNG via headless Chrome (`chromedp`)
5. Retorna a imagem (URL ou base64) + caption gerada
6. Opcionalmente, envia direto pelo canal (WhatsApp, Telegram, etc.)

---

## Fluxo Completo: SAC Rio Quality como Exemplo

```
┌─────────────────────────────────────────────────────────────────┐
│                    AGENTE LLM (SAC Rio Quality)                 │
│                                                                 │
│  System Prompt: "Você é o assistente da Rio Quality..."         │
│                                                                 │
│  Tools disponíveis:                                             │
│  ┌─────────────────┐ ┌──────────────────┐ ┌──────────────────┐ │
│  │ consultar_pedido │ │ buscar_produtos  │ │ verificar_estoque│ │
│  │ (dados)          │ │ (dados)          │ │ (dados)          │ │
│  └────────┬────────┘ └────────┬─────────┘ └────────┬─────────┘ │
│           │                   │                     │           │
│  ┌────────┴───────────────────┴─────────────────────┴────────┐  │
│  │              TOOLS DE APRESENTAÇÃO (wrappers)              │  │
│  │  ┌──────────────┐ ┌────────────────┐ ┌──────────────────┐ │  │
│  │  │ mostrar_menu │ │ card_produto   │ │ status_pedido    │ │  │
│  │  │ (terminal)   │ │ (terminal)     │ │ (terminal)       │ │  │
│  │  └──────┬───────┘ └───────┬────────┘ └────────┬─────────┘ │  │
│  └─────────┼─────────────────┼────────────────────┼──────────┘  │
└────────────┼─────────────────┼────────────────────┼─────────────┘
             │                 │                    │
             ▼                 ▼                    ▼
┌─────────────────────────────────────────────────────────────────┐
│              ORQUESTRADOR (Middleware do Bot)                    │
│                                                                 │
│  - Detecta tool_call com terminal: true                         │
│  - Monta request para Linktor VRE API                           │
│  - Recebe imagem + caption                                      │
│  - Envia pelo canal (ou delega ao Linktor)                      │
└──────────────────────────┬──────────────────────────────────────┘
                           │
                           ▼
┌─────────────────────────────────────────────────────────────────┐
│                 LINKTOR VRE SERVICE                              │
│                                                                 │
│  POST /api/v1/render                                            │
│  {                                                              │
│    "tenant_id": "rio-quality",                                  │
│    "template_id": "menu_principal",                             │
│    "data": { "nome": "João", "opcoes": [...] },                │
│    "output": "png",                                             │
│    "channel": "whatsapp",    // opcional: adapta dimensões      │
│    "send_to": "5543999...",  // opcional: envia direto           │
│    "caption": "Olá João!..." // opcional: LLM gera o caption    │
│  }                                                              │
│                                                                 │
│  ┌──────────┐   ┌──────────────┐   ┌────────────────────────┐  │
│  │ Template  │──▶│ Renderer     │──▶│ Delivery               │  │
│  │ Registry  │   │ (chromedp)   │   │ (canal adapter)        │  │
│  └──────────┘   └──────────────┘   └────────────────────────┘  │
│                                                                 │
│  Resposta:                                                      │
│  {                                                              │
│    "image_url": "https://cdn.linktor.io/renders/abc123.png",   │
│    "image_base64": "iVBOR...",                                  │
│    "caption": "Olá João! Escolha uma opção:\n1️⃣ ...",          │
│    "delivered": true                                            │
│  }                                                              │
└─────────────────────────────────────────────────────────────────┘
```

---

## Tool Definitions para o LLM (Function Calling)

As tools do agente SAC são **wrappers finos** que chamam o Linktor.
O LLM só conhece o schema — não sabe que por trás há renderização de imagem.

### Tool: `mostrar_menu`

```json
{
  "name": "mostrar_menu",
  "description": "Apresenta um menu visual de opções para o cliente escolher. Use quando precisar oferecer 2-8 opções de ação. O cliente responderá com o número ou texto da opção.",
  "parameters": {
    "type": "object",
    "properties": {
      "titulo": {
        "type": "string",
        "description": "Título do menu (ex: 'Como posso ajudar?')"
      },
      "subtitulo": {
        "type": "string",
        "description": "Subtítulo opcional com contexto"
      },
      "opcoes": {
        "type": "array",
        "items": {
          "type": "object",
          "properties": {
            "label": { "type": "string" },
            "descricao": { "type": "string" },
            "icone": {
              "type": "string",
              "enum": ["pedido", "catalogo", "entrega", "financeiro",
                       "atendente", "reclamacao", "devolucao", "outro"]
            }
          },
          "required": ["label"]
        },
        "maxItems": 8
      },
      "mensagem_antes": {
        "type": "string",
        "description": "Texto conversacional que o agente quer enviar ANTES do menu"
      }
    },
    "required": ["titulo", "opcoes"]
  },
  "_meta": {
    "terminal": true,
    "linktor_template": "menu_opcoes",
    "response_type": "visual"
  }
}
```

### Tool: `mostrar_card_produto`

```json
{
  "name": "mostrar_card_produto",
  "description": "Mostra um card visual de produto com imagem, preço e disponibilidade. Use para apresentar um produto específico ao cliente.",
  "parameters": {
    "type": "object",
    "properties": {
      "nome": { "type": "string" },
      "sku": { "type": "string" },
      "preco": { "type": "number" },
      "unidade": { "type": "string", "enum": ["kg", "un", "cx", "fd", "pc"] },
      "estoque": { "type": "integer" },
      "imagem_url": { "type": "string" },
      "destaque": { "type": "string", "description": "Selo opcional: 'promoção', 'novo', 'mais vendido'" },
      "mensagem": { "type": "string", "description": "Texto do agente sobre o produto" }
    },
    "required": ["nome", "preco", "unidade"]
  },
  "_meta": {
    "terminal": true,
    "linktor_template": "card_produto",
    "response_type": "visual"
  }
}
```

### Tool: `mostrar_status_pedido`

```json
{
  "name": "mostrar_status_pedido",
  "description": "Mostra um card visual com o status do pedido do cliente, incluindo timeline de etapas.",
  "parameters": {
    "type": "object",
    "properties": {
      "numero_pedido": { "type": "string" },
      "status_atual": {
        "type": "string",
        "enum": ["recebido", "separacao", "faturado", "transporte", "entregue"]
      },
      "itens_resumo": { "type": "string" },
      "valor_total": { "type": "number" },
      "previsao_entrega": { "type": "string" },
      "motorista": { "type": "string" },
      "mensagem": { "type": "string" }
    },
    "required": ["numero_pedido", "status_atual"]
  },
  "_meta": {
    "terminal": true,
    "linktor_template": "status_pedido",
    "response_type": "visual"
  }
}
```

### Tool: `mostrar_lista_produtos`

```json
{
  "name": "mostrar_lista_produtos",
  "description": "Mostra uma lista visual de produtos (até 6) para o cliente comparar. Use quando o cliente pedir sugestões ou buscar produtos.",
  "parameters": {
    "type": "object",
    "properties": {
      "titulo": { "type": "string" },
      "produtos": {
        "type": "array",
        "items": {
          "type": "object",
          "properties": {
            "nome": { "type": "string" },
            "preco": { "type": "number" },
            "unidade": { "type": "string" },
            "estoque_status": { "type": "string", "enum": ["disponivel", "baixo", "indisponivel"] }
          }
        },
        "maxItems": 6
      },
      "mensagem": { "type": "string" }
    },
    "required": ["titulo", "produtos"]
  },
  "_meta": {
    "terminal": true,
    "linktor_template": "lista_produtos",
    "response_type": "visual"
  }
}
```

---

## Templates HTML: Estrutura e Customização

### Organização no Linktor

```
linktor/
├── templates/
│   ├── _base/                          # Templates base compartilhados
│   │   ├── whatsapp-card.html          # Layout base: 800x600, dark/light
│   │   ├── whatsapp-wide.html          # Layout wide: 800x400
│   │   └── whatsapp-tall.html          # Layout tall: 800x1000
│   │
│   ├── _components/                    # Componentes reutilizáveis
│   │   ├── header.html                 # Logo + título
│   │   ├── option-row.html             # Linha de opção com ícone
│   │   ├── product-card.html           # Card de produto individual
│   │   ├── timeline-step.html          # Step de timeline
│   │   ├── price-tag.html              # Etiqueta de preço
│   │   └── badge.html                  # Selo (promoção, novo, etc.)
│   │
│   ├── default/                        # Templates padrão do Linktor
│   │   ├── menu_opcoes.html
│   │   ├── card_produto.html
│   │   ├── status_pedido.html
│   │   ├── lista_produtos.html
│   │   ├── confirmacao.html
│   │   └── erro.html
│   │
│   └── tenants/                        # Templates customizados por tenant
│       ├── rio-quality/
│       │   ├── config.json             # Cores, logo, fontes
│       │   ├── menu_opcoes.html        # Override do default
│       │   └── card_produto.html       # Override do default
│       │
│       └── outro-cliente/
│           └── config.json
```

### config.json por Tenant

```json
{
  "tenant_id": "rio-quality",
  "brand": {
    "name": "Rio Quality",
    "logo_url": "https://cdn.linktor.io/tenants/rio-quality/logo.png",
    "primary_color": "#1B4F72",
    "secondary_color": "#F39C12",
    "accent_color": "#27AE60",
    "background": "#FFFFFF",
    "text_color": "#2C3E50",
    "font_family": "Inter, sans-serif",
    "border_radius": "12px"
  },
  "templates": {
    "menu_opcoes": "default",
    "card_produto": "custom",
    "status_pedido": "default"
  },
  "icons": {
    "pedido": "🛒",
    "catalogo": "📋",
    "entrega": "🚚",
    "financeiro": "💰",
    "atendente": "👤",
    "reclamacao": "📝",
    "devolucao": "↩️",
    "outro": "❓"
  }
}
```

### Template HTML: `menu_opcoes.html`

Usa Go `html/template` com marcações `{{ }}` que são substituídas pelos dados.

```html
<!DOCTYPE html>
<html>
<head>
<meta charset="UTF-8">
<style>
  * { margin: 0; padding: 0; box-sizing: border-box; }

  body {
    width: 800px;
    font-family: {{ .Brand.FontFamily }};
    background: {{ .Brand.Background }};
    color: {{ .Brand.TextColor }};
  }

  .card {
    padding: 32px;
    background: {{ .Brand.Background }};
  }

  .header {
    display: flex;
    align-items: center;
    gap: 16px;
    margin-bottom: 24px;
    padding-bottom: 16px;
    border-bottom: 2px solid {{ .Brand.PrimaryColor }}20;
  }

  .logo {
    width: 48px;
    height: 48px;
    border-radius: 50%;
    object-fit: cover;
  }

  .brand-name {
    font-size: 14px;
    color: {{ .Brand.PrimaryColor }};
    font-weight: 600;
  }

  .titulo {
    font-size: 22px;
    font-weight: 700;
    color: {{ .Brand.PrimaryColor }};
    margin-bottom: 8px;
  }

  .subtitulo {
    font-size: 14px;
    color: #7F8C8D;
    margin-bottom: 24px;
  }

  .opcoes {
    display: flex;
    flex-direction: column;
    gap: 12px;
  }

  .opcao {
    display: flex;
    align-items: center;
    gap: 16px;
    padding: 16px 20px;
    background: {{ .Brand.PrimaryColor }}08;
    border: 1px solid {{ .Brand.PrimaryColor }}20;
    border-radius: {{ .Brand.BorderRadius }};
    transition: all 0.2s;
  }

  .opcao-numero {
    width: 36px;
    height: 36px;
    background: {{ .Brand.PrimaryColor }};
    color: white;
    border-radius: 50%;
    display: flex;
    align-items: center;
    justify-content: center;
    font-weight: 700;
    font-size: 16px;
    flex-shrink: 0;
  }

  .opcao-icone {
    font-size: 24px;
    flex-shrink: 0;
  }

  .opcao-texto {
    flex: 1;
  }

  .opcao-label {
    font-size: 16px;
    font-weight: 600;
    color: {{ .Brand.TextColor }};
  }

  .opcao-descricao {
    font-size: 12px;
    color: #95A5A6;
    margin-top: 2px;
  }

  .footer {
    margin-top: 24px;
    padding-top: 16px;
    border-top: 1px solid #ECF0F1;
    font-size: 12px;
    color: #BDC3C7;
    text-align: center;
  }
</style>
</head>
<body>
<div class="card">

  <div class="header">
    {{ if .Brand.LogoURL }}
    <img src="{{ .Brand.LogoURL }}" class="logo" />
    {{ end }}
    <span class="brand-name">{{ .Brand.Name }}</span>
  </div>

  <div class="titulo">{{ .Data.Titulo }}</div>
  {{ if .Data.Subtitulo }}
  <div class="subtitulo">{{ .Data.Subtitulo }}</div>
  {{ end }}

  <div class="opcoes">
    {{ range $i, $opt := .Data.Opcoes }}
    <div class="opcao">
      <div class="opcao-numero">{{ add $i 1 }}</div>
      <div class="opcao-icone">{{ icon $opt.Icone }}</div>
      <div class="opcao-texto">
        <div class="opcao-label">{{ $opt.Label }}</div>
        {{ if $opt.Descricao }}
        <div class="opcao-descricao">{{ $opt.Descricao }}</div>
        {{ end }}
      </div>
    </div>
    {{ end }}
  </div>

  <div class="footer">
    Responda com o número da opção desejada
  </div>

</div>
</body>
</html>
```

### Template HTML: `status_pedido.html`

```html
<!DOCTYPE html>
<html>
<head>
<meta charset="UTF-8">
<style>
  * { margin: 0; padding: 0; box-sizing: border-box; }

  body {
    width: 800px;
    font-family: {{ .Brand.FontFamily }};
    background: {{ .Brand.Background }};
  }

  .card { padding: 32px; }

  .header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    margin-bottom: 24px;
  }

  .pedido-num {
    font-size: 14px;
    color: #7F8C8D;
  }

  .pedido-num strong {
    color: {{ .Brand.PrimaryColor }};
    font-size: 18px;
  }

  .timeline {
    display: flex;
    align-items: center;
    justify-content: space-between;
    margin: 32px 0;
    position: relative;
  }

  .timeline::before {
    content: '';
    position: absolute;
    top: 20px;
    left: 40px;
    right: 40px;
    height: 3px;
    background: #ECF0F1;
  }

  .timeline-step {
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: 8px;
    z-index: 1;
    flex: 1;
  }

  .step-dot {
    width: 40px;
    height: 40px;
    border-radius: 50%;
    display: flex;
    align-items: center;
    justify-content: center;
    font-size: 18px;
  }

  .step-dot.completed {
    background: {{ .Brand.AccentColor }};
    color: white;
  }

  .step-dot.current {
    background: {{ .Brand.SecondaryColor }};
    color: white;
    box-shadow: 0 0 0 4px {{ .Brand.SecondaryColor }}40;
  }

  .step-dot.pending {
    background: #ECF0F1;
    color: #BDC3C7;
  }

  .step-label {
    font-size: 11px;
    color: #7F8C8D;
    text-align: center;
    max-width: 80px;
  }

  .step-label.active {
    color: {{ .Brand.PrimaryColor }};
    font-weight: 600;
  }

  .info-grid {
    display: grid;
    grid-template-columns: 1fr 1fr;
    gap: 16px;
    margin-top: 24px;
  }

  .info-item {
    padding: 12px 16px;
    background: #F8F9FA;
    border-radius: 8px;
  }

  .info-label {
    font-size: 11px;
    color: #95A5A6;
    text-transform: uppercase;
    letter-spacing: 0.5px;
  }

  .info-value {
    font-size: 16px;
    font-weight: 600;
    color: {{ .Brand.TextColor }};
    margin-top: 4px;
  }
</style>
</head>
<body>
<div class="card">

  <div class="header">
    <div>
      <div class="pedido-num">Pedido <strong>#{{ .Data.NumeroPedido }}</strong></div>
    </div>
    <img src="{{ .Brand.LogoURL }}" style="height:32px;" />
  </div>

  <div class="timeline">
    {{ range $i, $step := .Steps }}
    <div class="timeline-step">
      <div class="step-dot {{ $step.Status }}">{{ $step.Icon }}</div>
      <div class="step-label {{ if eq $step.Status "current" }}active{{ end }}">
        {{ $step.Label }}
      </div>
    </div>
    {{ end }}
  </div>

  <div class="info-grid">
    {{ if .Data.ItensResumo }}
    <div class="info-item">
      <div class="info-label">Itens</div>
      <div class="info-value">{{ .Data.ItensResumo }}</div>
    </div>
    {{ end }}
    {{ if .Data.ValorTotal }}
    <div class="info-item">
      <div class="info-label">Valor Total</div>
      <div class="info-value">R$ {{ formatCurrency .Data.ValorTotal }}</div>
    </div>
    {{ end }}
    {{ if .Data.PrevisaoEntrega }}
    <div class="info-item">
      <div class="info-label">Previsão</div>
      <div class="info-value">{{ .Data.PrevisaoEntrega }}</div>
    </div>
    {{ end }}
    {{ if .Data.Motorista }}
    <div class="info-item">
      <div class="info-label">Motorista</div>
      <div class="info-value">{{ .Data.Motorista }}</div>
    </div>
    {{ end }}
  </div>

</div>
</body>
</html>
```

---

## Implementação Go do VRE Service

### Estrutura do serviço

```
linktor/
└── services/
    └── vre/                         # Visual Response Engine
        ├── cmd/
        │   └── main.go
        ├── internal/
        │   ├── api/
        │   │   ├── handler.go       # HTTP handlers
        │   │   └── routes.go
        │   ├── renderer/
        │   │   ├── renderer.go      # Interface
        │   │   ├── chromedp.go      # Chrome headless renderer
        │   │   ├── pool.go          # Chrome instance pool
        │   │   └── cache.go         # Rendered image cache
        │   ├── template/
        │   │   ├── registry.go      # Template discovery & loading
        │   │   ├── functions.go     # Custom template functions
        │   │   └── resolver.go      # Tenant template resolution
        │   ├── caption/
        │   │   └── generator.go     # Gera caption acessível do template
        │   └── domain/
        │       ├── render_request.go
        │       └── render_response.go
        ├── templates/               # HTML templates
        │   ├── _base/
        │   ├── _components/
        │   ├── default/
        │   └── tenants/
        ├── Dockerfile
        └── go.mod
```

### Core: `render_request.go`

```go
package domain

type RenderRequest struct {
    TenantID   string          `json:"tenant_id" validate:"required"`
    TemplateID string          `json:"template_id" validate:"required"`
    Data       json.RawMessage `json:"data"`               // Dados dinâmicos do template
    Output     OutputFormat    `json:"output,omitempty"`    // png (default), webp, jpeg
    Channel    string          `json:"channel,omitempty"`   // whatsapp, telegram, web
    Width      int             `json:"width,omitempty"`     // Override de largura
    Caption    string          `json:"caption,omitempty"`   // Caption fornecido pelo LLM
    SendTo     string          `json:"send_to,omitempty"`   // Enviar direto pelo canal
    SessionID  string          `json:"session_id,omitempty"`
}

type RenderResponse struct {
    ImageURL    string `json:"image_url,omitempty"`
    ImageBase64 string `json:"image_base64,omitempty"`
    Caption     string `json:"caption"`
    Width       int    `json:"width"`
    Height      int    `json:"height"`
    Delivered   bool   `json:"delivered,omitempty"`
    RenderTime  int64  `json:"render_time_ms"`
}

// ChannelDefaults define dimensões padrão por canal
var ChannelDefaults = map[string]ChannelConfig{
    "whatsapp": {Width: 800, MaxHeight: 1200, Format: "png", Quality: 90},
    "telegram": {Width: 800, MaxHeight: 1200, Format: "png", Quality: 90},
    "web":      {Width: 600, MaxHeight: 0, Format: "webp", Quality: 85},
    "email":    {Width: 600, MaxHeight: 0, Format: "png", Quality: 95},
}
```

### Core: `renderer.go` (interface + chromedp)

```go
package renderer

import (
    "context"
    "github.com/chromedp/chromedp"
)

type Renderer interface {
    RenderHTML(ctx context.Context, html string, opts RenderOpts) ([]byte, error)
}

type ChromeRenderer struct {
    pool *BrowserPool
}

type RenderOpts struct {
    Width   int
    Format  string // "png", "webp", "jpeg"
    Quality int
}

func (r *ChromeRenderer) RenderHTML(ctx context.Context, html string, opts RenderOpts) ([]byte, error) {
    // Pega instância do pool
    allocCtx, cancel := r.pool.Acquire(ctx)
    defer cancel()

    taskCtx, taskCancel := chromedp.NewContext(allocCtx)
    defer taskCancel()

    var buf []byte

    // Navega para o HTML (via data URL ou temp file)
    // e captura screenshot
    err := chromedp.Run(taskCtx,
        chromedp.EmulateViewport(int64(opts.Width), 1),
        chromedp.Navigate("data:text/html,"+url.QueryEscape(html)),
        chromedp.WaitReady("body"),
        // Captura com altura automática baseada no conteúdo
        chromedp.ActionFunc(func(ctx context.Context) error {
            // Calcula altura real do conteúdo
            var height int64
            chromedp.Evaluate(`document.body.scrollHeight`, &height).Do(ctx)
            // Screenshot com dimensões exatas
            buf, err = page.CaptureScreenshot().
                WithClip(&page.Viewport{
                    X: 0, Y: 0,
                    Width:  float64(opts.Width),
                    Height: float64(height),
                    Scale:  2, // Retina quality
                }).Do(ctx)
            return err
        }),
    )

    return buf, err
}
```

### Core: `resolver.go` (resolução de templates por tenant)

```go
package template

// Resolve retorna o template HTML final para um tenant + template_id.
// Estratégia de fallback:
//   1. tenants/{tenant_id}/{template_id}.html  (custom do tenant)
//   2. default/{template_id}.html               (template padrão)
//   3. erro: template não encontrado

func (r *Resolver) Resolve(tenantID, templateID string) (*Template, error) {
    // 1. Tenta custom do tenant
    path := filepath.Join(r.basePath, "tenants", tenantID, templateID+".html")
    if exists(path) {
        return r.loadAndParse(path, tenantID)
    }

    // 2. Fallback para default
    path = filepath.Join(r.basePath, "default", templateID+".html")
    if exists(path) {
        return r.loadAndParse(path, tenantID)
    }

    return nil, ErrTemplateNotFound
}

// loadAndParse carrega o HTML e injeta o config.json do tenant
func (r *Resolver) loadAndParse(path, tenantID string) (*Template, error) {
    htmlBytes, _ := os.ReadFile(path)
    config := r.loadTenantConfig(tenantID) // Carrega cores, logo, fontes

    tmpl, err := template.New("").
        Funcs(r.customFuncs()). // add, icon, formatCurrency, etc.
        Parse(string(htmlBytes))

    return &Template{
        GoTemplate: tmpl,
        Config:     config,
    }, err
}
```

### Core: `caption/generator.go`

```go
package caption

// GenerateCaption cria o texto acessível da imagem.
// Se o LLM já forneceu caption, usa ele.
// Senão, gera automaticamente baseado no template + dados.

func Generate(req *domain.RenderRequest, templateID string) string {
    // LLM já forneceu caption? Usa.
    if req.Caption != "" {
        return req.Caption
    }

    // Geração automática baseada no tipo de template
    switch templateID {
    case "menu_opcoes":
        return generateMenuCaption(req.Data)
    case "card_produto":
        return generateProductCaption(req.Data)
    case "status_pedido":
        return generateStatusCaption(req.Data)
    default:
        return ""
    }
}

// generateMenuCaption cria texto como:
// "Escolha uma opção:
//  1️⃣ Fazer pedido
//  2️⃣ Status do pedido
//  3️⃣ Ver catálogo"
func generateMenuCaption(data json.RawMessage) string {
    var menu MenuData
    json.Unmarshal(data, &menu)

    emojis := []string{"1️⃣", "2️⃣", "3️⃣", "4️⃣", "5️⃣", "6️⃣", "7️⃣", "8️⃣"}
    var b strings.Builder
    b.WriteString(menu.Titulo + "\n\n")
    for i, opt := range menu.Opcoes {
        b.WriteString(emojis[i] + " " + opt.Label + "\n")
    }
    b.WriteString("\n_Responda com o número da opção_")
    return b.String()
}
```

---

## API REST do VRE

### `POST /api/v1/render`
Renderiza template e retorna imagem.

### `POST /api/v1/render-and-send`
Renderiza e envia direto pelo canal configurado.

### `GET /api/v1/templates`
Lista templates disponíveis para um tenant.

### `POST /api/v1/templates`
Faz upload de template HTML customizado para um tenant.

### `GET /api/v1/templates/:id/preview`
Renderiza preview com dados de exemplo.

---

## Como o Orquestrador do Bot Integra

### Pseudocódigo do middleware

```go
func ProcessLLMResponse(response LLMResponse) {
    for _, toolCall := range response.ToolCalls {
        toolDef := registry.GetTool(toolCall.Name)

        if toolDef.Meta.Terminal && toolDef.Meta.ResponseType == "visual" {
            // === TOOL DE APRESENTAÇÃO ===

            // 1. Texto antes (se o LLM gerou)
            if msg := toolCall.Args["mensagem_antes"]; msg != "" {
                channel.SendText(session.ChatID, msg)
            }

            // 2. Chama o Linktor VRE
            renderReq := &vre.RenderRequest{
                TenantID:   session.TenantID,
                TemplateID: toolDef.Meta.LinktorTemplate,
                Data:       toolCall.Args,                    // JSON dos argumentos
                Channel:    session.Channel,                  // "whatsapp"
                Caption:    toolCall.Args["mensagem"],        // Texto do LLM como caption
            }

            renderResp := linktorClient.Render(renderReq)

            // 3. Envia imagem + caption pelo canal
            channel.SendImage(session.ChatID, SendImageRequest{
                ImageURL: renderResp.ImageURL,
                Caption:  renderResp.Caption,
            })

            // 4. NÃO devolve resultado para o LLM (é terminal)
            return

        } else {
            // === TOOL DE DADOS ===
            // Executa e devolve resultado para o LLM continuar
            result := executeDataTool(toolCall)
            appendToConversation(result)
        }
    }
}
```

---

## Cache e Performance

### Estratégia de cache

```
Cache Key = hash(tenant_id + template_id + sha256(data_json) + channel)

Camadas:
1. Redis (TTL: 5 min)   → Mesma imagem pedida múltiplas vezes
2. S3/MinIO              → CDN URL para entrega
3. Chrome Pool           → 3-5 instâncias reutilizáveis

Invalidação:
- Quando template HTML é alterado → invalida todas as keys do template
- Quando config.json do tenant muda → invalida todas as keys do tenant
```

### Performance esperada (RTX 3060 / 16GB RAM)

```
Renderização HTML → PNG:  150-300ms (primeira vez)
Cache hit (Redis):        5-10ms
Chrome pool warmup:       2-3 segundos (startup)
Memória por instância:    ~100-150MB Chrome headless
Pool de 3 instâncias:     ~450MB total
Throughput:               ~10-15 renders/segundo
```

---

## Customização por Tenant: Fluxo Completo

### 1. Tenant usa templates padrão (zero config)
```
POST /api/v1/render
{ "tenant_id": "novo-cliente", "template_id": "menu_opcoes", "data": {...} }

→ Usa default/menu_opcoes.html com cores/logo padrão do Linktor
```

### 2. Tenant customiza cores e logo (config.json apenas)
```
PUT /api/v1/tenants/rio-quality/config
{ "brand": { "primary_color": "#1B4F72", "logo_url": "...", ... } }

→ Usa default/menu_opcoes.html mas com visual da Rio Quality
```

### 3. Tenant cria template custom (HTML completo)
```
POST /api/v1/tenants/rio-quality/templates/menu_opcoes
Content-Type: text/html
<html>... template customizado ...</html>

→ Usa template exclusivo do tenant com todas as marcações {{ }}
```

### Cada nível herda do anterior. O tenant pode customizar apenas o que precisa.
