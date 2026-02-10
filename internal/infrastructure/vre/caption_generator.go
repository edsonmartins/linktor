package vre

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/msgfy/linktor/internal/domain/entity"
)

// CaptionGenerator generates accessible text captions for rendered images
type CaptionGenerator struct {
	numberEmojis []string
}

// NewCaptionGenerator creates a new caption generator
func NewCaptionGenerator() *CaptionGenerator {
	return &CaptionGenerator{
		numberEmojis: []string{"1️⃣", "2️⃣", "3️⃣", "4️⃣", "5️⃣", "6️⃣", "7️⃣", "8️⃣"},
	}
}

// Generate generates a caption for a template
// If the LLM already provided a caption, it is returned as-is
// Otherwise, an automatic caption is generated based on the template type and data
func (g *CaptionGenerator) Generate(templateID string, data map[string]interface{}, llmCaption string) string {
	// If LLM already provided a caption, use it
	if llmCaption != "" {
		return llmCaption
	}

	// Generate automatic caption based on template type
	switch entity.TemplateType(templateID) {
	case entity.TemplateTypeMenuOpcoes:
		return g.generateMenuCaption(data)
	case entity.TemplateTypeCardProduto:
		return g.generateProductCaption(data)
	case entity.TemplateTypeStatusPedido:
		return g.generateStatusCaption(data)
	case entity.TemplateTypeListaProdutos:
		return g.generateListCaption(data)
	case entity.TemplateTypeConfirmacao:
		return g.generateConfirmationCaption(data)
	case entity.TemplateTypeCobrancaPix:
		return g.generatePixCaption(data)
	default:
		return ""
	}
}

// generateMenuCaption generates caption for menu_opcoes template
func (g *CaptionGenerator) generateMenuCaption(data map[string]interface{}) string {
	var b strings.Builder

	// Title
	if titulo, ok := data["titulo"].(string); ok {
		b.WriteString(titulo)
		b.WriteString("\n\n")
	}

	// Options
	if opcoes, ok := data["opcoes"].([]interface{}); ok {
		for i, opt := range opcoes {
			if i >= len(g.numberEmojis) {
				break
			}
			if optMap, ok := opt.(map[string]interface{}); ok {
				b.WriteString(g.numberEmojis[i])
				b.WriteString(" ")
				if label, ok := optMap["label"].(string); ok {
					b.WriteString(label)
				}
				b.WriteString("\n")
			}
		}
	}

	b.WriteString("\n_Responda com o número da opção_")
	return b.String()
}

// generateProductCaption generates caption for card_produto template
func (g *CaptionGenerator) generateProductCaption(data map[string]interface{}) string {
	var b strings.Builder

	// Name
	if nome, ok := data["nome"].(string); ok {
		b.WriteString("📦 ")
		b.WriteString(nome)
		b.WriteString("\n")
	}

	// Price
	if preco, ok := data["preco"].(float64); ok {
		unidade := "un"
		if u, ok := data["unidade"].(string); ok {
			unidade = u
		}
		b.WriteString(fmt.Sprintf("💰 R$ %.2f/%s\n", preco, unidade))
	}

	// Stock
	if estoque, ok := data["estoque"].(float64); ok && estoque > 0 {
		b.WriteString("✅ Em estoque\n")
	} else if estoque, ok := data["estoque"].(int); ok && estoque > 0 {
		b.WriteString("✅ Em estoque\n")
	}

	// Message
	if mensagem, ok := data["mensagem"].(string); ok {
		b.WriteString("\n")
		b.WriteString(mensagem)
	}

	return b.String()
}

// generateStatusCaption generates caption for status_pedido template
func (g *CaptionGenerator) generateStatusCaption(data map[string]interface{}) string {
	var b strings.Builder

	// Order number and status
	if numero, ok := data["numero_pedido"].(string); ok {
		statusLabels := map[string]string{
			"recebido":   "📥 Recebido",
			"separacao":  "📦 Em separação",
			"faturado":   "🧾 Faturado",
			"transporte": "🚚 Em transporte",
			"entregue":   "✅ Entregue",
		}

		b.WriteString(fmt.Sprintf("📦 Pedido #%s\n", numero))

		if status, ok := data["status_atual"].(string); ok {
			if label, exists := statusLabels[status]; exists {
				b.WriteString(fmt.Sprintf("📍 Status: %s\n", label))
			}
		}
	}

	// Summary
	if itens, ok := data["itens_resumo"].(string); ok {
		b.WriteString(fmt.Sprintf("📋 %s\n", itens))
	}

	// Value
	if valor, ok := data["valor_total"].(float64); ok && valor > 0 {
		b.WriteString(fmt.Sprintf("💰 R$ %.2f\n", valor))
	}

	// Delivery
	if previsao, ok := data["previsao_entrega"].(string); ok {
		b.WriteString(fmt.Sprintf("🕐 Previsão: %s\n", previsao))
	}

	// Driver
	if motorista, ok := data["motorista"].(string); ok {
		b.WriteString(fmt.Sprintf("🚛 Motorista: %s\n", motorista))
	}

	return b.String()
}

// generateListCaption generates caption for lista_produtos template
func (g *CaptionGenerator) generateListCaption(data map[string]interface{}) string {
	var b strings.Builder

	// Title
	if titulo, ok := data["titulo"].(string); ok {
		b.WriteString(titulo)
		b.WriteString("\n\n")
	}

	// Products
	if produtos, ok := data["produtos"].([]interface{}); ok {
		for i, prod := range produtos {
			if i >= len(g.numberEmojis) {
				break
			}
			if prodMap, ok := prod.(map[string]interface{}); ok {
				b.WriteString(g.numberEmojis[i])
				b.WriteString(" ")
				if nome, ok := prodMap["nome"].(string); ok {
					b.WriteString(nome)
				}
				if preco, ok := prodMap["preco"].(float64); ok {
					unidade := "un"
					if u, ok := prodMap["unidade"].(string); ok {
						unidade = u
					}
					b.WriteString(fmt.Sprintf(" — R$ %.2f/%s", preco, unidade))
				}
				b.WriteString("\n")
			}
		}
	}

	// Message
	if mensagem, ok := data["mensagem"].(string); ok {
		b.WriteString("\n")
		b.WriteString(mensagem)
	}

	return b.String()
}

// generateConfirmationCaption generates caption for confirmacao template
func (g *CaptionGenerator) generateConfirmationCaption(data map[string]interface{}) string {
	var b strings.Builder

	b.WriteString("📋 Resumo do pedido:\n\n")

	// Items
	if itens, ok := data["itens"].([]interface{}); ok {
		for _, item := range itens {
			if itemMap, ok := item.(map[string]interface{}); ok {
				emoji := "•"
				if e, ok := itemMap["emoji"].(string); ok {
					emoji = e
				}
				nome := ""
				if n, ok := itemMap["nome"].(string); ok {
					nome = n
				}
				qtd := ""
				if q, ok := itemMap["quantidade"].(string); ok {
					qtd = q
				}
				preco := 0.0
				if p, ok := itemMap["preco"].(float64); ok {
					preco = p
				}

				if qtd != "" {
					b.WriteString(fmt.Sprintf("%s %s — %s — R$ %.2f\n", emoji, nome, qtd, preco))
				} else {
					b.WriteString(fmt.Sprintf("%s %s — R$ %.2f\n", emoji, nome, preco))
				}
			}
		}
	}

	// Total
	if total, ok := data["valor_total"].(float64); ok {
		b.WriteString(fmt.Sprintf("\n💰 Total: R$ %.2f\n", total))
	}

	// Delivery
	if previsao, ok := data["previsao_entrega"].(string); ok {
		b.WriteString(fmt.Sprintf("📅 Entrega: %s\n", previsao))
	}

	b.WriteString("\n_Responda SIM para confirmar ou peça alterações_")
	return b.String()
}

// generatePixCaption generates caption for cobranca_pix template
func (g *CaptionGenerator) generatePixCaption(data map[string]interface{}) string {
	var b strings.Builder

	b.WriteString("◆ Pagamento via PIX\n\n")

	// Order
	if numero, ok := data["numero_pedido"].(string); ok {
		b.WriteString(fmt.Sprintf("📋 Pedido #%s\n", numero))
	}

	// Value
	if valor, ok := data["valor"].(float64); ok {
		b.WriteString(fmt.Sprintf("💰 Valor: R$ %.2f\n", valor))
	}

	// Expiration
	if expiracao, ok := data["expiracao"].(string); ok {
		b.WriteString(fmt.Sprintf("⏱ Válido por %s\n", expiracao))
	}

	// PIX code (truncated for caption)
	if payload, ok := data["pix_payload"].(string); ok && len(payload) > 0 {
		b.WriteString("\n📱 Código PIX copia e cola:\n")
		if len(payload) > 50 {
			b.WriteString(payload[:50])
			b.WriteString("...")
		} else {
			b.WriteString(payload)
		}
		b.WriteString("\n")
	}

	b.WriteString("\n_Assim que identificarmos o pagamento, te aviso aqui!_")
	return b.String()
}

// GenerateFromJSON generates caption from JSON data
func (g *CaptionGenerator) GenerateFromJSON(templateID string, jsonData []byte, llmCaption string) (string, error) {
	var data map[string]interface{}
	if err := json.Unmarshal(jsonData, &data); err != nil {
		return "", fmt.Errorf("failed to parse JSON: %w", err)
	}
	return g.Generate(templateID, data, llmCaption), nil
}
