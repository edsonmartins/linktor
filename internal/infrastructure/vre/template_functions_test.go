package vre

import (
	"testing"
	"time"
)

func TestFormatCurrency(t *testing.T) {
	formatCurrency := TemplateFuncs["formatCurrency"].(func(interface{}) string)

	tests := []struct {
		name     string
		input    interface{}
		expected string
	}{
		{"float64 simple", 99.90, "R$ 99,90"},
		{"float64 with thousands", 1234.56, "R$ 1.234,56"},
		{"float64 zero decimals", 100.00, "R$ 100,00"},
		{"int", 50, "R$ 50,00"},
		{"int64", int64(1000), "R$ 1.000,00"},
		{"float32", float32(25.50), "R$ 25,50"},
		{"invalid type", "not a number", "R$ --"},
		{"large number", 1234567.89, "R$ 1.234.567,89"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatCurrency(tt.input)
			if result != tt.expected {
				t.Errorf("formatCurrency(%v) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestIcon(t *testing.T) {
	icon := TemplateFuncs["icon"].(func(string) string)

	tests := []struct {
		name     string
		expected string
	}{
		{"pedido", "🛒"},
		{"catalogo", "📋"},
		{"entrega", "🚚"},
		{"financeiro", "💰"},
		{"atendente", "👤"},
		{"unknown", "•"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := icon(tt.name)
			if result != tt.expected {
				t.Errorf("icon(%q) = %q, want %q", tt.name, result, tt.expected)
			}
		})
	}
}

func TestFormatDate(t *testing.T) {
	formatDate := TemplateFuncs["formatDate"].(func(interface{}) string)

	tests := []struct {
		name     string
		input    interface{}
		expected string
	}{
		{
			"time.Time",
			time.Date(2024, 12, 25, 10, 30, 0, 0, time.UTC),
			"25/12/2024",
		},
		{
			"ISO string",
			"2024-12-25T10:30:00Z",
			"25/12/2024",
		},
		{
			"date string",
			"2024-12-25",
			"25/12/2024",
		},
		{
			"invalid string",
			"not a date",
			"not a date",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatDate(tt.input)
			if result != tt.expected {
				t.Errorf("formatDate(%v) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestStockStatus(t *testing.T) {
	stockStatus := TemplateFuncs["stockStatus"].(func(string) string)

	tests := []struct {
		input    string
		expected string
	}{
		{"disponivel", "✓ Em estoque"},
		{"baixo", "⚠ Estoque baixo"},
		{"indisponivel", "✗ Indisponível"},
		{"unknown", "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := stockStatus(tt.input)
			if result != tt.expected {
				t.Errorf("stockStatus(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestNumEmoji(t *testing.T) {
	numEmoji := TemplateFuncs["numEmoji"].(func(int) string)

	tests := []struct {
		input    int
		expected string
	}{
		{1, "1️⃣"},
		{2, "2️⃣"},
		{3, "3️⃣"},
		{8, "8️⃣"},
		{0, "0"},
		{9, "9"},
		{-1, "-1"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			result := numEmoji(tt.input)
			if result != tt.expected {
				t.Errorf("numEmoji(%d) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestTruncate(t *testing.T) {
	truncate := TemplateFuncs["truncate"].(func(string, int) string)

	tests := []struct {
		input    string
		length   int
		expected string
	}{
		{"short", 10, "short"},
		{"this is a long text", 10, "this is..."},
		{"exactly10c", 10, "exactly10c"},
		{"", 5, ""},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := truncate(tt.input, tt.length)
			if result != tt.expected {
				t.Errorf("truncate(%q, %d) = %q, want %q", tt.input, tt.length, result, tt.expected)
			}
		})
	}
}

func TestFormatUnit(t *testing.T) {
	formatUnit := TemplateFuncs["formatUnit"].(func(string) string)

	tests := []struct {
		input    string
		expected string
	}{
		{"kg", "/kg"},
		{"un", "/un"},
		{"cx", "/cx"},
		{"pc", "/pç"},
		{"custom", "/custom"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := formatUnit(tt.input)
			if result != tt.expected {
				t.Errorf("formatUnit(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestMathFunctions(t *testing.T) {
	add := TemplateFuncs["add"].(func(int, int) int)
	sub := TemplateFuncs["sub"].(func(int, int) int)
	mul := TemplateFuncs["mul"].(func(int, int) int)

	if result := add(2, 3); result != 5 {
		t.Errorf("add(2, 3) = %d, want 5", result)
	}

	if result := sub(5, 2); result != 3 {
		t.Errorf("sub(5, 2) = %d, want 3", result)
	}

	if result := mul(4, 3); result != 12 {
		t.Errorf("mul(4, 3) = %d, want 12", result)
	}
}

func TestDict(t *testing.T) {
	dict := TemplateFuncs["dict"].(func(...interface{}) map[string]interface{})

	result := dict("key1", "value1", "key2", 42)

	if result["key1"] != "value1" {
		t.Errorf("dict key1 = %v, want 'value1'", result["key1"])
	}
	if result["key2"] != 42 {
		t.Errorf("dict key2 = %v, want 42", result["key2"])
	}
}

func TestList(t *testing.T) {
	list := TemplateFuncs["list"].(func(...interface{}) []interface{})

	result := list("a", "b", "c")

	if len(result) != 3 {
		t.Errorf("list length = %d, want 3", len(result))
	}
	if result[0] != "a" || result[1] != "b" || result[2] != "c" {
		t.Errorf("list = %v, want [a, b, c]", result)
	}
}

func TestFormatWithThousands(t *testing.T) {
	tests := []struct {
		input    int
		expected string
	}{
		{0, "0"},
		{123, "123"},
		{1234, "1.234"},
		{12345, "12.345"},
		{123456, "123.456"},
		{1234567, "1.234.567"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			result := formatWithThousands(tt.input)
			if result != tt.expected {
				t.Errorf("formatWithThousands(%d) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}
