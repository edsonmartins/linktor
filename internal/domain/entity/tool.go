package entity

import "encoding/json"

// ToolResponseType defines how the tool result should be presented
type ToolResponseType string

const (
	ToolResponseTypeText   ToolResponseType = "text"   // Text response to the AI
	ToolResponseTypeVisual ToolResponseType = "visual" // Visual response via VRE
	ToolResponseTypeData   ToolResponseType = "data"   // Data response (JSON)
)

// ToolMeta contains metadata for tool behavior in Linktor
type ToolMeta struct {
	Terminal        bool             `json:"terminal"`                   // If true, don't continue to AI after tool execution
	ResponseType    ToolResponseType `json:"response_type,omitempty"`    // Type of response: text, visual, data
	LinktorTemplate string           `json:"linktor_template,omitempty"` // VRE template ID for visual responses
	RequiresAuth    bool             `json:"requires_auth,omitempty"`    // Requires user authentication
	RateLimit       int              `json:"rate_limit,omitempty"`       // Max calls per minute (0 = unlimited)
}

// ToolParameter defines a parameter for a tool
type ToolParameter struct {
	Name        string      `json:"name"`
	Type        string      `json:"type"` // string, number, boolean, array, object
	Description string      `json:"description"`
	Required    bool        `json:"required"`
	Enum        []string    `json:"enum,omitempty"`
	Default     interface{} `json:"default,omitempty"`
	Items       *ToolParameter `json:"items,omitempty"` // For array types
}

// Tool defines a tool/function that the AI can call
type Tool struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Parameters  map[string]interface{} `json:"parameters"` // JSON Schema
	Meta        *ToolMeta              `json:"_meta,omitempty"`
}

// ToolCall represents a tool call made by the AI
type ToolCall struct {
	ID        string                 `json:"id"`
	Name      string                 `json:"name"`
	Arguments map[string]interface{} `json:"arguments"`
}

// ToolResult represents the result of a tool execution
type ToolResult struct {
	ToolCallID string      `json:"tool_call_id"`
	Content    string      `json:"content"`
	IsError    bool        `json:"is_error,omitempty"`
	Data       interface{} `json:"data,omitempty"` // Structured data for visual responses
}

// VisualToolResult extends ToolResult for VRE responses
type VisualToolResult struct {
	ToolResult
	TemplateID   string `json:"template_id"`
	ImageBase64  string `json:"image_base64"`
	Caption      string `json:"caption"`
	FollowUpText string `json:"follow_up_text,omitempty"`
}

// NewTool creates a new tool with basic configuration
func NewTool(name, description string) *Tool {
	return &Tool{
		Name:        name,
		Description: description,
		Parameters: map[string]interface{}{
			"type":       "object",
			"properties": map[string]interface{}{},
			"required":   []string{},
		},
	}
}

// WithMeta adds metadata to the tool
func (t *Tool) WithMeta(meta *ToolMeta) *Tool {
	t.Meta = meta
	return t
}

// AddParameter adds a parameter to the tool
func (t *Tool) AddParameter(name, paramType, description string, required bool) *Tool {
	props := t.Parameters["properties"].(map[string]interface{})
	props[name] = map[string]interface{}{
		"type":        paramType,
		"description": description,
	}

	if required {
		requiredList := t.Parameters["required"].([]string)
		t.Parameters["required"] = append(requiredList, name)
	}

	return t
}

// IsTerminal returns true if the tool should end the AI turn
func (t *Tool) IsTerminal() bool {
	return t.Meta != nil && t.Meta.Terminal
}

// IsVisual returns true if the tool produces a visual response
func (t *Tool) IsVisual() bool {
	return t.Meta != nil && t.Meta.ResponseType == ToolResponseTypeVisual
}

// GetLinktorTemplate returns the VRE template ID for visual tools
func (t *Tool) GetLinktorTemplate() string {
	if t.Meta != nil {
		return t.Meta.LinktorTemplate
	}
	return ""
}

// ParseArguments parses tool call arguments from raw JSON
func (tc *ToolCall) ParseArguments(raw json.RawMessage) error {
	return json.Unmarshal(raw, &tc.Arguments)
}

// GetString gets a string argument by name
func (tc *ToolCall) GetString(name string) string {
	if v, ok := tc.Arguments[name].(string); ok {
		return v
	}
	return ""
}

// GetFloat gets a float argument by name
func (tc *ToolCall) GetFloat(name string) float64 {
	switch v := tc.Arguments[name].(type) {
	case float64:
		return v
	case int:
		return float64(v)
	case int64:
		return float64(v)
	}
	return 0
}

// GetBool gets a boolean argument by name
func (tc *ToolCall) GetBool(name string) bool {
	if v, ok := tc.Arguments[name].(bool); ok {
		return v
	}
	return false
}

// GetArray gets an array argument by name
func (tc *ToolCall) GetArray(name string) []interface{} {
	if v, ok := tc.Arguments[name].([]interface{}); ok {
		return v
	}
	return nil
}

// GetObject gets an object argument by name
func (tc *ToolCall) GetObject(name string) map[string]interface{} {
	if v, ok := tc.Arguments[name].(map[string]interface{}); ok {
		return v
	}
	return nil
}

// BuiltInTools returns the default VRE tools for visual responses
func BuiltInVRETools() []*Tool {
	return []*Tool{
		{
			Name:        "mostrar_menu",
			Description: "Exibe um menu visual com opções numeradas para o usuário escolher. Use quando precisar apresentar múltiplas opções de forma clara.",
			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"titulo": map[string]interface{}{
						"type":        "string",
						"description": "Título do menu",
					},
					"opcoes": map[string]interface{}{
						"type":        "array",
						"description": "Lista de opções (máximo 8)",
						"items": map[string]interface{}{
							"type": "object",
							"properties": map[string]interface{}{
								"label":     map[string]interface{}{"type": "string", "description": "Texto da opção"},
								"icone":     map[string]interface{}{"type": "string", "description": "Nome do ícone (pedido, catalogo, entrega, etc)"},
								"descricao": map[string]interface{}{"type": "string", "description": "Descrição opcional"},
							},
							"required": []string{"label"},
						},
					},
				},
				"required": []string{"titulo", "opcoes"},
			},
			Meta: &ToolMeta{
				Terminal:        true,
				ResponseType:    ToolResponseTypeVisual,
				LinktorTemplate: "menu_opcoes",
			},
		},
		{
			Name:        "mostrar_card_produto",
			Description: "Exibe um card visual de produto com imagem, preço e informações de estoque.",
			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"nome":       map[string]interface{}{"type": "string", "description": "Nome do produto"},
					"preco":      map[string]interface{}{"type": "number", "description": "Preço do produto"},
					"unidade":    map[string]interface{}{"type": "string", "description": "Unidade (kg, un, cx, etc)"},
					"imagem_url": map[string]interface{}{"type": "string", "description": "URL da imagem do produto"},
					"estoque":    map[string]interface{}{"type": "string", "description": "Status do estoque (disponivel, baixo, indisponivel)"},
					"descricao":  map[string]interface{}{"type": "string", "description": "Descrição do produto"},
				},
				"required": []string{"nome", "preco"},
			},
			Meta: &ToolMeta{
				Terminal:        true,
				ResponseType:    ToolResponseTypeVisual,
				LinktorTemplate: "card_produto",
			},
		},
		{
			Name:        "mostrar_status_pedido",
			Description: "Exibe o status visual de um pedido com timeline de progresso.",
			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"numero_pedido":    map[string]interface{}{"type": "string", "description": "Número do pedido"},
					"status_atual":     map[string]interface{}{"type": "string", "description": "Status atual (recebido, separacao, faturado, transporte, entregue)"},
					"itens_resumo":     map[string]interface{}{"type": "string", "description": "Resumo dos itens"},
					"valor_total":      map[string]interface{}{"type": "number", "description": "Valor total do pedido"},
					"previsao_entrega": map[string]interface{}{"type": "string", "description": "Previsão de entrega"},
					"motorista":        map[string]interface{}{"type": "string", "description": "Nome do motorista"},
				},
				"required": []string{"numero_pedido", "status_atual"},
			},
			Meta: &ToolMeta{
				Terminal:        true,
				ResponseType:    ToolResponseTypeVisual,
				LinktorTemplate: "status_pedido",
			},
		},
		{
			Name:        "mostrar_lista_produtos",
			Description: "Exibe uma lista visual de produtos para comparação.",
			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"titulo": map[string]interface{}{"type": "string", "description": "Título da lista"},
					"produtos": map[string]interface{}{
						"type":        "array",
						"description": "Lista de produtos (máximo 6)",
						"items": map[string]interface{}{
							"type": "object",
							"properties": map[string]interface{}{
								"nome":    map[string]interface{}{"type": "string"},
								"preco":   map[string]interface{}{"type": "number"},
								"unidade": map[string]interface{}{"type": "string"},
								"estoque": map[string]interface{}{"type": "string"},
							},
							"required": []string{"nome", "preco"},
						},
					},
				},
				"required": []string{"titulo", "produtos"},
			},
			Meta: &ToolMeta{
				Terminal:        true,
				ResponseType:    ToolResponseTypeVisual,
				LinktorTemplate: "lista_produtos",
			},
		},
		{
			Name:        "mostrar_confirmacao",
			Description: "Exibe um resumo visual para confirmação de pedido ou ação.",
			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"titulo": map[string]interface{}{"type": "string", "description": "Título da confirmação"},
					"itens": map[string]interface{}{
						"type":        "array",
						"description": "Lista de itens",
						"items": map[string]interface{}{
							"type": "object",
							"properties": map[string]interface{}{
								"nome":       map[string]interface{}{"type": "string"},
								"quantidade": map[string]interface{}{"type": "string"},
								"preco":      map[string]interface{}{"type": "number"},
							},
						},
					},
					"valor_total":      map[string]interface{}{"type": "number", "description": "Valor total"},
					"previsao_entrega": map[string]interface{}{"type": "string", "description": "Previsão de entrega"},
				},
				"required": []string{"titulo", "valor_total"},
			},
			Meta: &ToolMeta{
				Terminal:        true,
				ResponseType:    ToolResponseTypeVisual,
				LinktorTemplate: "confirmacao",
			},
		},
		{
			Name:        "mostrar_cobranca_pix",
			Description: "Exibe um QR code PIX para pagamento com código copia e cola.",
			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"valor":         map[string]interface{}{"type": "number", "description": "Valor do pagamento"},
					"pix_payload":   map[string]interface{}{"type": "string", "description": "Código PIX copia e cola"},
					"qr_code_url":   map[string]interface{}{"type": "string", "description": "URL do QR code"},
					"numero_pedido": map[string]interface{}{"type": "string", "description": "Número do pedido"},
					"expiracao":     map[string]interface{}{"type": "string", "description": "Tempo de expiração"},
				},
				"required": []string{"valor", "pix_payload"},
			},
			Meta: &ToolMeta{
				Terminal:        true,
				ResponseType:    ToolResponseTypeVisual,
				LinktorTemplate: "cobranca_pix",
			},
		},
	}
}
