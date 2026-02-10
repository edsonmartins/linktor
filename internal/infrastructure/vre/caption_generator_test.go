package vre

import (
	"strings"
	"testing"

	"github.com/msgfy/linktor/internal/domain/entity"
)

func TestCaptionGenerator_Generate(t *testing.T) {
	gen := NewCaptionGenerator()

	tests := []struct {
		name       string
		templateID string
		data       map[string]interface{}
		contains   []string
	}{
		{
			name:       "menu_opcoes basic",
			templateID: "menu_opcoes",
			data: map[string]interface{}{
				"titulo": "Como posso ajudar?",
				"opcoes": []interface{}{
					map[string]interface{}{"label": "Fazer pedido"},
					map[string]interface{}{"label": "Ver catálogo"},
				},
			},
			contains: []string{"Como posso ajudar?", "1️⃣", "Fazer pedido", "2️⃣", "Ver catálogo"},
		},
		{
			name:       "card_produto with price",
			templateID: "card_produto",
			data: map[string]interface{}{
				"nome":    "Camarão Cinza",
				"preco":   62.90,
				"unidade": "kg",
			},
			contains: []string{"Camarão Cinza", "R$", "62.90", "/kg"},
		},
		{
			name:       "status_pedido",
			templateID: "status_pedido",
			data: map[string]interface{}{
				"numero_pedido": "1234",
				"status_atual":  "transporte",
			},
			contains: []string{"Pedido #1234", "Em transporte"},
		},
		{
			name:       "lista_produtos",
			templateID: "lista_produtos",
			data: map[string]interface{}{
				"titulo": "Produtos disponíveis",
				"produtos": []interface{}{
					map[string]interface{}{"nome": "Produto A", "preco": 29.90},
					map[string]interface{}{"nome": "Produto B", "preco": 49.90},
				},
			},
			contains: []string{"Produtos disponíveis", "Produto A", "Produto B"},
		},
		{
			name:       "confirmacao",
			templateID: "confirmacao",
			data: map[string]interface{}{
				"titulo":      "Confirmar pedido?",
				"valor_total": 150.00,
			},
			contains: []string{"Resumo do pedido", "R$", "150.00"},
		},
		{
			name:       "cobranca_pix",
			templateID: "cobranca_pix",
			data: map[string]interface{}{
				"valor":       150.00,
				"pix_payload": "00020126580014br...",
			},
			contains: []string{"PIX", "R$", "150.00", "00020126580014br..."},
		},
		{
			name:       "unknown template",
			templateID: "unknown",
			data:       map[string]interface{}{},
			contains:   []string{}, // Empty caption for unknown
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := gen.Generate(tt.templateID, tt.data, "")

			for _, expected := range tt.contains {
				if !strings.Contains(result, expected) {
					t.Errorf("caption for %q should contain %q, got: %q", tt.templateID, expected, result)
				}
			}
		})
	}
}

func TestCaptionGenerator_GenerateWithCustomCaption(t *testing.T) {
	gen := NewCaptionGenerator()

	customCaption := "Custom caption provided"
	result := gen.Generate("menu_opcoes", map[string]interface{}{}, customCaption)

	if result != customCaption {
		t.Errorf("expected custom caption %q, got %q", customCaption, result)
	}
}

func TestCaptionGenerator_MenuWithDescriptions(t *testing.T) {
	gen := NewCaptionGenerator()

	data := map[string]interface{}{
		"titulo": "Escolha uma opção",
		"opcoes": []interface{}{
			map[string]interface{}{"label": "Pedido", "descricao": "Fazer novo pedido"},
			map[string]interface{}{"label": "Rastrear", "descricao": "Ver status da entrega"},
		},
	}

	result := gen.Generate("menu_opcoes", data, "")

	// The current implementation includes labels but not descriptions in caption
	if !strings.Contains(result, "Pedido") {
		t.Errorf("caption should contain label, got: %q", result)
	}
}

func TestCaptionGenerator_ProductWithStock(t *testing.T) {
	gen := NewCaptionGenerator()

	tests := []struct {
		name     string
		estoque  interface{}
		contains string
	}{
		{"with stock number", float64(100), "Em estoque"}, // JSON numbers are float64
		{"with stock int", 100, "Em estoque"},
		{"no stock field", nil, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data := map[string]interface{}{
				"nome":  "Produto",
				"preco": 10.00,
			}
			if tt.estoque != nil {
				data["estoque"] = tt.estoque
			}

			result := gen.Generate("card_produto", data, "")

			if tt.contains != "" && !strings.Contains(result, tt.contains) {
				t.Errorf("caption should contain %q for estoque=%v, got: %q", tt.contains, tt.estoque, result)
			}
		})
	}
}

func TestCaptionGenerator_StatusPedidoWithDetails(t *testing.T) {
	gen := NewCaptionGenerator()

	data := map[string]interface{}{
		"numero_pedido":    "5678",
		"status_atual":     "entregue",
		"previsao_entrega": "Hoje às 18h",
		"motorista":        "João",
	}

	result := gen.Generate("status_pedido", data, "")

	expectedParts := []string{"5678", "Hoje às 18h", "João"}
	for _, part := range expectedParts {
		if !strings.Contains(result, part) {
			t.Errorf("caption should contain %q, got: %q", part, result)
		}
	}
}

func TestCaptionGenerator_ConfirmacaoWithItems(t *testing.T) {
	gen := NewCaptionGenerator()

	data := map[string]interface{}{
		"titulo": "Confirmar?",
		"itens": []interface{}{
			map[string]interface{}{"nome": "Item 1", "quantidade": "2x", "preco": 20.00},
			map[string]interface{}{"nome": "Item 2", "quantidade": "1x", "preco": 15.00},
		},
		"valor_total": 35.00,
	}

	result := gen.Generate("confirmacao", data, "")

	if !strings.Contains(result, "Item 1") || !strings.Contains(result, "Item 2") {
		t.Errorf("caption should contain item names, got: %q", result)
	}
	if !strings.Contains(result, "35.00") {
		t.Errorf("caption should contain total, got: %q", result)
	}
}

func TestStatusPedidoData_GetTimelineSteps(t *testing.T) {
	tests := []struct {
		status   string
		wantDone int
		wantWait int
	}{
		{"recebido", 0, 4},
		{"separacao", 1, 3},
		{"faturado", 2, 2},
		{"transporte", 3, 1},
		{"entregue", 4, 0},
	}

	for _, tt := range tests {
		t.Run(tt.status, func(t *testing.T) {
			data := &entity.StatusPedidoData{
				StatusAtual: tt.status,
			}

			steps := data.GetTimelineSteps()

			doneCount := 0
			waitCount := 0
			for _, step := range steps {
				if step.Status == "done" {
					doneCount++
				} else if step.Status == "wait" {
					waitCount++
				}
			}

			if doneCount != tt.wantDone {
				t.Errorf("status %q: got %d done steps, want %d", tt.status, doneCount, tt.wantDone)
			}
			if waitCount != tt.wantWait {
				t.Errorf("status %q: got %d wait steps, want %d", tt.status, waitCount, tt.wantWait)
			}
		})
	}
}
