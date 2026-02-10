package entity

import (
	"fmt"
	"time"
)

// OutputFormat represents the image output format
type OutputFormat string

const (
	OutputFormatPNG  OutputFormat = "png"
	OutputFormatWebP OutputFormat = "webp"  // Recommended: smaller file size
	OutputFormatJPEG OutputFormat = "jpeg"
)

// ChannelType for VRE rendering
type VREChannelType string

const (
	VREChannelWhatsApp VREChannelType = "whatsapp"
	VREChannelTelegram VREChannelType = "telegram"
	VREChannelWeb      VREChannelType = "web"
	VREChannelEmail    VREChannelType = "email"
)

// ChannelDefaults defines default rendering settings per channel
type ChannelDefaults struct {
	Width     int          `json:"width"`
	MaxHeight int          `json:"max_height"`
	Format    OutputFormat `json:"format"`
	Quality   int          `json:"quality"` // 0-100, only for webp/jpeg
	Scale     float64      `json:"scale"`   // 1.0 = normal, 2.0 = retina
}

// DefaultChannelSettings provides optimized defaults per channel
var DefaultChannelSettings = map[VREChannelType]ChannelDefaults{
	VREChannelWhatsApp: {Width: 800, MaxHeight: 1200, Format: OutputFormatWebP, Quality: 85, Scale: 1.5},
	VREChannelTelegram: {Width: 800, MaxHeight: 1200, Format: OutputFormatWebP, Quality: 85, Scale: 1.5},
	VREChannelWeb:      {Width: 600, MaxHeight: 0, Format: OutputFormatWebP, Quality: 80, Scale: 1.0},
	VREChannelEmail:    {Width: 600, MaxHeight: 0, Format: OutputFormatPNG, Quality: 90, Scale: 1.0},
}

// RenderRequest represents a request to render a template
type RenderRequest struct {
	TenantID   string                 `json:"tenant_id" validate:"required"`
	TemplateID string                 `json:"template_id,omitempty"` // Optional: use predefined template
	HTML       string                 `json:"html,omitempty"`        // Optional: custom HTML (takes precedence)
	Data       map[string]interface{} `json:"data"`
	Channel    VREChannelType         `json:"channel,omitempty"`  // whatsapp, telegram, web
	Caption    string                 `json:"caption,omitempty"`  // Optional: LLM can provide
	SendTo     string                 `json:"send_to,omitempty"`  // Optional: send directly to this recipient
	SessionID  string                 `json:"session_id,omitempty"`

	// FollowUpText is an optional text message to send after the image
	// Example: "Temos sim! 🦐 Quer adicionar ao pedido? Quantos kg?"
	FollowUpText string `json:"follow_up_text,omitempty"`

	// Image optimization options (optional overrides)
	Width   int          `json:"width,omitempty"`   // Override default width
	Format  OutputFormat `json:"format,omitempty"`  // Override default format
	Quality int          `json:"quality,omitempty"` // Override quality (0-100)
	Scale   float64      `json:"scale,omitempty"`   // Override scale (1.0-2.0)
}

// IsCustomHTML returns true if the request uses custom HTML
func (r *RenderRequest) IsCustomHTML() bool {
	return r.HTML != ""
}

// Validate validates the render request
func (r *RenderRequest) Validate() error {
	if r.TenantID == "" {
		return fmt.Errorf("tenant_id is required")
	}
	if r.HTML == "" && r.TemplateID == "" {
		return fmt.Errorf("either html or template_id is required")
	}
	return nil
}

// GetDefaults returns the effective rendering defaults for this request
func (r *RenderRequest) GetDefaults() ChannelDefaults {
	defaults, ok := DefaultChannelSettings[r.Channel]
	if !ok {
		defaults = DefaultChannelSettings[VREChannelWhatsApp] // fallback
	}

	// Apply overrides
	if r.Width > 0 {
		defaults.Width = r.Width
	}
	if r.Format != "" {
		defaults.Format = r.Format
	}
	if r.Quality > 0 {
		defaults.Quality = r.Quality
	}
	if r.Scale > 0 {
		defaults.Scale = r.Scale
	}

	return defaults
}

// RenderResponse represents the result of rendering a template
type RenderResponse struct {
	ImageURL     string        `json:"image_url,omitempty"`
	ImageBase64  string        `json:"image_base64,omitempty"`
	Caption      string        `json:"caption"`
	FollowUpText string        `json:"follow_up_text,omitempty"` // Text to send after image
	Width        int           `json:"width"`
	Height       int           `json:"height"`
	Format       OutputFormat  `json:"format"`
	SizeBytes    int64         `json:"size_bytes"` // Image size for monitoring
	RenderTime   time.Duration `json:"render_time_ms"`
	CacheHit     bool          `json:"cache_hit"`
	Delivered    bool          `json:"delivered,omitempty"` // If send_to was used
}

// TenantBrandConfig defines the visual branding for a tenant
type TenantBrandConfig struct {
	TenantID       string            `json:"tenant_id"`
	Name           string            `json:"name"`
	LogoURL        string            `json:"logo_url,omitempty"`
	PrimaryColor   string            `json:"primary_color"`   // #1B4F72
	SecondaryColor string            `json:"secondary_color"` // #F39C12
	AccentColor    string            `json:"accent_color"`    // #27AE60
	Background     string            `json:"background"`      // #FFFFFF
	TextColor      string            `json:"text_color"`      // #2C3E50
	MutedColor     string            `json:"muted_color"`     // #8B95A2
	FontFamily     string            `json:"font_family"`     // DM Sans, sans-serif
	BorderRadius   string            `json:"border_radius"`   // 14px
	Icons          map[string]string `json:"icons"`           // pedido -> 🛒
}

// DefaultBrandConfig returns the default Linktor branding
func DefaultBrandConfig() *TenantBrandConfig {
	return &TenantBrandConfig{
		Name:           "Linktor",
		PrimaryColor:   "#0F3460",
		SecondaryColor: "#E94560",
		AccentColor:    "#16C79A",
		Background:     "#FFFFFF",
		TextColor:      "#1A1A2E",
		MutedColor:     "#8B95A2",
		FontFamily:     "'DM Sans', sans-serif",
		BorderRadius:   "14px",
		Icons: map[string]string{
			"pedido":     "🛒",
			"catalogo":   "📋",
			"entrega":    "🚚",
			"financeiro": "💰",
			"atendente":  "👤",
			"reclamacao": "📝",
			"devolucao":  "↩️",
			"outro":      "❓",
		},
	}
}

// TemplateType represents the type of VRE template
type TemplateType string

const (
	TemplateTypeMenuOpcoes     TemplateType = "menu_opcoes"
	TemplateTypeCardProduto    TemplateType = "card_produto"
	TemplateTypeStatusPedido   TemplateType = "status_pedido"
	TemplateTypeListaProdutos  TemplateType = "lista_produtos"
	TemplateTypeConfirmacao    TemplateType = "confirmacao"
	TemplateTypeCobrancaPix    TemplateType = "cobranca_pix"
)

// MenuOpcaoData represents a single menu option
type MenuOpcaoData struct {
	Label     string `json:"label"`
	Descricao string `json:"descricao,omitempty"`
	Icone     string `json:"icone,omitempty"` // pedido, catalogo, entrega, etc.
}

// MenuOpcoesData represents data for the menu_opcoes template
type MenuOpcoesData struct {
	Titulo        string          `json:"titulo"`
	Subtitulo     string          `json:"subtitulo,omitempty"`
	Opcoes        []MenuOpcaoData `json:"opcoes"` // max 8
	MensagemAntes string          `json:"mensagem_antes,omitempty"`
}

// CardProdutoData represents data for the card_produto template
type CardProdutoData struct {
	Nome      string  `json:"nome"`
	SKU       string  `json:"sku,omitempty"`
	Preco     float64 `json:"preco"`
	Unidade   string  `json:"unidade"` // kg, un, cx, fd, pc
	Estoque   int     `json:"estoque,omitempty"`
	ImagemURL string  `json:"imagem_url,omitempty"`
	Destaque  string  `json:"destaque,omitempty"` // promoção, novo, mais vendido
	Mensagem  string  `json:"mensagem,omitempty"`
}

// StatusPedidoStep represents a single step in the order timeline
type StatusPedidoStep struct {
	Status string `json:"status"` // done, active, wait
	Icon   string `json:"icon"`
	Label  string `json:"label"`
}

// StatusPedidoData represents data for the status_pedido template
type StatusPedidoData struct {
	NumeroPedido     string `json:"numero_pedido"`
	StatusAtual      string `json:"status_atual"` // recebido, separacao, faturado, transporte, entregue
	ItensResumo      string `json:"itens_resumo,omitempty"`
	ValorTotal       float64 `json:"valor_total,omitempty"`
	PrevisaoEntrega  string `json:"previsao_entrega,omitempty"`
	Motorista        string `json:"motorista,omitempty"`
	Mensagem         string `json:"mensagem,omitempty"`
}

// GetTimelineSteps returns the timeline steps based on current status
func (s *StatusPedidoData) GetTimelineSteps() []StatusPedidoStep {
	statusOrder := []string{"recebido", "separacao", "faturado", "transporte", "entregue"}
	icons := map[string]string{
		"recebido":   "✓",
		"separacao":  "✓",
		"faturado":   "✓",
		"transporte": "🚚",
		"entregue":   "📍",
	}
	labels := map[string]string{
		"recebido":   "Recebido",
		"separacao":  "Separação",
		"faturado":   "Faturado",
		"transporte": "Em transporte",
		"entregue":   "Entregue",
	}

	var steps []StatusPedidoStep
	currentIndex := -1
	for i, status := range statusOrder {
		if status == s.StatusAtual {
			currentIndex = i
			break
		}
	}

	for i, status := range statusOrder {
		step := StatusPedidoStep{
			Icon:  icons[status],
			Label: labels[status],
		}
		if i < currentIndex {
			step.Status = "done"
		} else if i == currentIndex {
			step.Status = "active"
		} else {
			step.Status = "wait"
		}
		steps = append(steps, step)
	}

	return steps
}

// ListaProdutoItem represents a single product in the list
type ListaProdutoItem struct {
	Nome          string  `json:"nome"`
	Preco         float64 `json:"preco"`
	Unidade       string  `json:"unidade,omitempty"`
	EstoqueStatus string  `json:"estoque_status,omitempty"` // disponivel, baixo, indisponivel
	SKU           string  `json:"sku,omitempty"`
	Emoji         string  `json:"emoji,omitempty"`
}

// ListaProdutosData represents data for the lista_produtos template
type ListaProdutosData struct {
	Titulo   string             `json:"titulo"`
	Produtos []ListaProdutoItem `json:"produtos"` // max 6
	Mensagem string             `json:"mensagem,omitempty"`
}

// ConfirmacaoItem represents a single item in confirmation
type ConfirmacaoItem struct {
	Nome       string  `json:"nome"`
	Quantidade string  `json:"quantidade,omitempty"`
	Preco      float64 `json:"preco"`
	Emoji      string  `json:"emoji,omitempty"`
}

// ConfirmacaoData represents data for the confirmacao template
type ConfirmacaoData struct {
	Titulo          string            `json:"titulo,omitempty"`
	Subtitulo       string            `json:"subtitulo,omitempty"`
	Itens           []ConfirmacaoItem `json:"itens"`
	ValorTotal      float64           `json:"valor_total"`
	PrevisaoEntrega string            `json:"previsao_entrega,omitempty"`
	Mensagem        string            `json:"mensagem,omitempty"`
}

// CobrancaPixData represents data for the cobranca_pix template
type CobrancaPixData struct {
	Valor        float64 `json:"valor"`
	NumeroPedido string  `json:"numero_pedido,omitempty"`
	PixPayload   string  `json:"pix_payload"` // EMV/BRCode payload
	Expiracao    string  `json:"expiracao,omitempty"` // "30 minutos"
	Mensagem     string  `json:"mensagem,omitempty"`
}

// TemplateData represents the data passed to a template for rendering
type TemplateData struct {
	Brand *TenantBrandConfig     `json:"brand"`
	Data  map[string]interface{} `json:"data"`
}
