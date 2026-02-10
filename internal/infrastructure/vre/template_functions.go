package vre

import (
	"fmt"
	"html/template"
	"strings"
	"time"
)

// TemplateFuncs provides custom functions for VRE templates
var TemplateFuncs = template.FuncMap{
	// Math functions
	"add": func(a, b int) int {
		return a + b
	},
	"sub": func(a, b int) int {
		return a - b
	},
	"mul": func(a, b int) int {
		return a * b
	},

	// Icon mapping
	"icon": func(name string) string {
		icons := map[string]string{
			"pedido":     "🛒",
			"catalogo":   "📋",
			"entrega":    "🚚",
			"financeiro": "💰",
			"atendente":  "👤",
			"reclamacao": "📝",
			"devolucao":  "↩️",
			"outro":      "❓",
			"produto":    "📦",
			"status":     "📍",
			"pix":        "◆",
			"check":      "✓",
			"confirm":    "✅",
			"warning":    "⚠️",
			"error":      "❌",
			"info":       "ℹ️",
		}
		if emoji, ok := icons[name]; ok {
			return emoji
		}
		return "•"
	},

	// Currency formatting (Brazilian Real)
	"formatCurrency": func(v interface{}) string {
		var value float64
		switch n := v.(type) {
		case float64:
			value = n
		case float32:
			value = float64(n)
		case int:
			value = float64(n)
		case int64:
			value = float64(n)
		default:
			return "R$ --"
		}

		// Format with Brazilian locale (1.234,56)
		// Use math.Round to avoid floating point precision issues
		cents := int(value*100 + 0.5) // Round to nearest cent
		intPart := cents / 100
		decPart := cents % 100

		// Add thousand separators
		intStr := formatWithThousands(intPart)

		return fmt.Sprintf("R$ %s,%02d", intStr, decPart)
	},

	// Date formatting
	"formatDate": func(v interface{}) string {
		var t time.Time
		switch d := v.(type) {
		case time.Time:
			t = d
		case string:
			// Try to parse ISO format
			if parsed, err := time.Parse(time.RFC3339, d); err == nil {
				t = parsed
			} else if parsed, err := time.Parse("2006-01-02", d); err == nil {
				t = parsed
			} else {
				return d
			}
		default:
			return ""
		}
		return t.Format("02/01/2006")
	},

	// DateTime formatting
	"formatDateTime": func(v interface{}) string {
		var t time.Time
		switch d := v.(type) {
		case time.Time:
			t = d
		case string:
			if parsed, err := time.Parse(time.RFC3339, d); err == nil {
				t = parsed
			} else {
				return d
			}
		default:
			return ""
		}
		return t.Format("02/01/2006 15:04")
	},

	// Stock status formatting
	"stockStatus": func(status string) string {
		switch status {
		case "disponivel":
			return "✓ Em estoque"
		case "baixo":
			return "⚠ Estoque baixo"
		case "indisponivel":
			return "✗ Indisponível"
		default:
			return status
		}
	},

	// Stock status CSS class
	"stockClass": func(status string) string {
		switch status {
		case "disponivel":
			return "stock-ok"
		case "baixo":
			return "stock-low"
		case "indisponivel":
			return "stock-out"
		default:
			return ""
		}
	},

	// Timeline step status class
	"stepClass": func(status string) string {
		switch status {
		case "done":
			return "step-icon done"
		case "active":
			return "step-icon active"
		case "wait":
			return "step-icon wait"
		default:
			return "step-icon"
		}
	},

	// Safe HTML (for trusted content)
	"safeHTML": func(s string) template.HTML {
		return template.HTML(s)
	},

	// Safe CSS
	"safeCSS": func(s string) template.CSS {
		return template.CSS(s)
	},

	// String functions
	"upper": strings.ToUpper,
	"lower": strings.ToLower,
	"title": strings.Title,
	"trim":  strings.TrimSpace,

	// Truncate text
	"truncate": func(s string, length int) string {
		if len(s) <= length {
			return s
		}
		return s[:length-3] + "..."
	},

	// Number emoji (1-8)
	"numEmoji": func(n int) string {
		emojis := []string{"1️⃣", "2️⃣", "3️⃣", "4️⃣", "5️⃣", "6️⃣", "7️⃣", "8️⃣"}
		if n >= 1 && n <= 8 {
			return emojis[n-1]
		}
		return fmt.Sprintf("%d", n)
	},

	// Default value
	"default": func(defaultVal, val interface{}) interface{} {
		if val == nil || val == "" {
			return defaultVal
		}
		return val
	},

	// Conditional
	"ifelse": func(cond bool, trueVal, falseVal interface{}) interface{} {
		if cond {
			return trueVal
		}
		return falseVal
	},

	// Repeat string
	"repeat": strings.Repeat,

	// Join array
	"join": func(sep string, items []string) string {
		return strings.Join(items, sep)
	},

	// Substring
	"substr": func(s string, start, length int) string {
		if start >= len(s) {
			return ""
		}
		end := start + length
		if end > len(s) {
			end = len(s)
		}
		return s[start:end]
	},

	// Create a list
	"list": func(args ...interface{}) []interface{} {
		return args
	},

	// Create a dictionary
	"dict": func(args ...interface{}) map[string]interface{} {
		result := make(map[string]interface{})
		for i := 0; i+1 < len(args); i += 2 {
			if key, ok := args[i].(string); ok {
				result[key] = args[i+1]
			}
		}
		return result
	},

	// Get index from slice
	"index": func(collection interface{}, key interface{}) interface{} {
		switch c := collection.(type) {
		case []interface{}:
			if idx, ok := key.(int); ok && idx >= 0 && idx < len(c) {
				return c[idx]
			}
		case []string:
			if idx, ok := key.(int); ok && idx >= 0 && idx < len(c) {
				return c[idx]
			}
			// Also support string key lookup
			if k, ok := key.(string); ok {
				for i, v := range c {
					if v == k {
						return i
					}
				}
			}
		case map[string]interface{}:
			if k, ok := key.(string); ok {
				return c[k]
			}
		case map[string]string:
			if k, ok := key.(string); ok {
				return c[k]
			}
		}
		return nil
	},

	// Len function
	"len": func(v interface{}) int {
		switch c := v.(type) {
		case []interface{}:
			return len(c)
		case []string:
			return len(c)
		case string:
			return len(c)
		case map[string]interface{}:
			return len(c)
		default:
			return 0
		}
	},

	// Product unit formatting
	"formatUnit": func(unit string) string {
		units := map[string]string{
			"kg": "/kg",
			"un": "/un",
			"cx": "/cx",
			"fd": "/fd",
			"pc": "/pç",
		}
		if formatted, ok := units[unit]; ok {
			return formatted
		}
		return "/" + unit
	},

	// Alpha channel for colors (hex with alpha)
	"withAlpha": func(hexColor string, alpha int) string {
		// Convert alpha (0-100) to hex (00-FF)
		alphaHex := fmt.Sprintf("%02X", alpha*255/100)
		return hexColor + alphaHex
	},
}

// formatWithThousands adds thousand separators (Brazilian format: 1.234.567)
func formatWithThousands(n int) string {
	str := fmt.Sprintf("%d", n)
	if len(str) <= 3 {
		return str
	}

	var result strings.Builder
	remainder := len(str) % 3
	if remainder > 0 {
		result.WriteString(str[:remainder])
		if len(str) > remainder {
			result.WriteString(".")
		}
	}

	for i := remainder; i < len(str); i += 3 {
		result.WriteString(str[i : i+3])
		if i+3 < len(str) {
			result.WriteString(".")
		}
	}

	return result.String()
}
