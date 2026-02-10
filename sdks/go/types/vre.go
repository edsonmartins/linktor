package types

// VREOutputFormat for rendered images
type VREOutputFormat string

const (
	VREOutputFormatPNG  VREOutputFormat = "png"
	VREOutputFormatWebP VREOutputFormat = "webp"
	VREOutputFormatJPEG VREOutputFormat = "jpeg"
)

// VREChannelType for VRE rendering
type VREChannelType string

const (
	VREChannelWhatsApp VREChannelType = "whatsapp"
	VREChannelTelegram VREChannelType = "telegram"
	VREChannelWeb      VREChannelType = "web"
	VREChannelEmail    VREChannelType = "email"
)

// VRETemplateType available templates
type VRETemplateType string

const (
	VRETemplateMenuOpcoes    VRETemplateType = "menu_opcoes"
	VRETemplateCardProduto   VRETemplateType = "card_produto"
	VRETemplateStatusPedido  VRETemplateType = "status_pedido"
	VRETemplateListaProdutos VRETemplateType = "lista_produtos"
	VRETemplateConfirmacao   VRETemplateType = "confirmacao"
	VRETemplateCobrancaPix   VRETemplateType = "cobranca_pix"
)

// OrderStatus for status_pedido template
type OrderStatus string

const (
	OrderStatusRecebido   OrderStatus = "recebido"
	OrderStatusSeparacao  OrderStatus = "separacao"
	OrderStatusFaturado   OrderStatus = "faturado"
	OrderStatusTransporte OrderStatus = "transporte"
	OrderStatusEntregue   OrderStatus = "entregue"
)

// StockStatus for products
type StockStatus string

const (
	StockStatusDisponivel   StockStatus = "disponivel"
	StockStatusBaixo        StockStatus = "baixo"
	StockStatusIndisponivel StockStatus = "indisponivel"
)

// VRERenderRequest for rendering a template
type VRERenderRequest struct {
	TenantID   string                 `json:"tenant_id"`
	TemplateID VRETemplateType        `json:"template_id"`
	Data       map[string]interface{} `json:"data"`
	Channel    VREChannelType         `json:"channel,omitempty"`
	Format     VREOutputFormat        `json:"format,omitempty"`
	Width      int                    `json:"width,omitempty"`
	Quality    int                    `json:"quality,omitempty"`
	Scale      float64                `json:"scale,omitempty"`
}

// VRERenderResponse from rendering a template
type VRERenderResponse struct {
	ImageBase64  string          `json:"image_base64"`
	Caption      string          `json:"caption"`
	Width        int             `json:"width"`
	Height       int             `json:"height"`
	Format       VREOutputFormat `json:"format"`
	RenderTimeMs int             `json:"render_time_ms"`
	SizeBytes    int64           `json:"size_bytes,omitempty"`
	CacheHit     bool            `json:"cache_hit,omitempty"`
}

// VRERenderAndSendRequest for render and send
type VRERenderAndSendRequest struct {
	ConversationID string                 `json:"conversation_id"`
	TemplateID     VRETemplateType        `json:"template_id"`
	Data           map[string]interface{} `json:"data"`
	Caption        string                 `json:"caption,omitempty"`
	FollowUpText   string                 `json:"follow_up_text,omitempty"`
}

// VRERenderAndSendResponse from render and send
type VRERenderAndSendResponse struct {
	MessageID string `json:"message_id"`
	ImageURL  string `json:"image_url"`
	Caption   string `json:"caption"`
}

// VRETemplate definition
type VRETemplate struct {
	ID          VRETemplateType        `json:"id"`
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Schema      map[string]interface{} `json:"schema"`
}

// VREListTemplatesResponse from listing templates
type VREListTemplatesResponse struct {
	Templates []VRETemplate `json:"templates"`
}

// VREPreviewRequest for previewing a template
type VREPreviewRequest struct {
	TemplateID VRETemplateType        `json:"template_id"`
	Data       map[string]interface{} `json:"data,omitempty"`
}

// VREPreviewResponse from preview
type VREPreviewResponse struct {
	ImageBase64 string `json:"image_base64"`
	Width       int    `json:"width"`
	Height      int    `json:"height"`
}

// MenuOpcaoData for menu_opcoes template
type MenuOpcaoData struct {
	Label     string `json:"label"`
	Descricao string `json:"descricao,omitempty"`
	Icone     string `json:"icone,omitempty"`
}

// MenuOpcoesData for menu_opcoes template
type MenuOpcoesData struct {
	Titulo        string          `json:"titulo"`
	Subtitulo     string          `json:"subtitulo,omitempty"`
	Opcoes        []MenuOpcaoData `json:"opcoes"`
	MensagemAntes string          `json:"mensagem_antes,omitempty"`
}

// CardProdutoData for card_produto template
type CardProdutoData struct {
	Nome      string  `json:"nome"`
	SKU       string  `json:"sku,omitempty"`
	Preco     float64 `json:"preco"`
	Unidade   string  `json:"unidade"`
	Estoque   int     `json:"estoque,omitempty"`
	ImagemURL string  `json:"imagem_url,omitempty"`
	Destaque  string  `json:"destaque,omitempty"`
	Mensagem  string  `json:"mensagem,omitempty"`
}

// StatusPedidoStep for status_pedido template
type StatusPedidoStep struct {
	Status string `json:"status"`
	Icon   string `json:"icon"`
	Label  string `json:"label"`
}

// StatusPedidoData for status_pedido template
type StatusPedidoData struct {
	NumeroPedido    string      `json:"numero_pedido"`
	StatusAtual     OrderStatus `json:"status_atual"`
	ItensResumo     string      `json:"itens_resumo,omitempty"`
	ValorTotal      float64     `json:"valor_total,omitempty"`
	PrevisaoEntrega string      `json:"previsao_entrega,omitempty"`
	Motorista       string      `json:"motorista,omitempty"`
	Mensagem        string      `json:"mensagem,omitempty"`
}

// ListaProdutoItem for lista_produtos template
type ListaProdutoItem struct {
	Nome          string      `json:"nome"`
	Preco         float64     `json:"preco"`
	Unidade       string      `json:"unidade,omitempty"`
	EstoqueStatus StockStatus `json:"estoque_status,omitempty"`
	SKU           string      `json:"sku,omitempty"`
	Emoji         string      `json:"emoji,omitempty"`
}

// ListaProdutosData for lista_produtos template
type ListaProdutosData struct {
	Titulo   string             `json:"titulo"`
	Produtos []ListaProdutoItem `json:"produtos"`
	Mensagem string             `json:"mensagem,omitempty"`
}

// ConfirmacaoItem for confirmacao template
type ConfirmacaoItem struct {
	Nome       string  `json:"nome"`
	Quantidade string  `json:"quantidade,omitempty"`
	Preco      float64 `json:"preco"`
	Emoji      string  `json:"emoji,omitempty"`
}

// ConfirmacaoData for confirmacao template
type ConfirmacaoData struct {
	Titulo          string            `json:"titulo,omitempty"`
	Subtitulo       string            `json:"subtitulo,omitempty"`
	Itens           []ConfirmacaoItem `json:"itens"`
	ValorTotal      float64           `json:"valor_total"`
	PrevisaoEntrega string            `json:"previsao_entrega,omitempty"`
	Mensagem        string            `json:"mensagem,omitempty"`
}

// CobrancaPixData for cobranca_pix template
type CobrancaPixData struct {
	Valor        float64 `json:"valor"`
	NumeroPedido string  `json:"numero_pedido,omitempty"`
	PixPayload   string  `json:"pix_payload"`
	Expiracao    string  `json:"expiracao,omitempty"`
	Mensagem     string  `json:"mensagem,omitempty"`
}
