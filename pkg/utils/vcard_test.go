package utils

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGenerateVCard_AllFields(t *testing.T) {
	result := GenerateVCard("John Doe", "+5511999999999", "john@example.com", "Acme Inc")
	assert.Contains(t, result, "BEGIN:VCARD")
	assert.Contains(t, result, "VERSION:3.0")
	assert.Contains(t, result, "FN:John Doe")
	assert.Contains(t, result, "ORG:Acme Inc;")
	assert.Contains(t, result, "TEL;type=CELL;type=VOICE;waid=+5511999999999:+5511999999999")
	assert.Contains(t, result, "EMAIL:john@example.com")
	assert.Contains(t, result, "END:VCARD")
}

func TestGenerateVCard_OnlyName(t *testing.T) {
	result := GenerateVCard("Jane Doe", "", "", "")
	assert.Contains(t, result, "FN:Jane Doe")
	assert.NotContains(t, result, "TEL")
	assert.NotContains(t, result, "EMAIL")
	assert.NotContains(t, result, "ORG")
}

func TestGenerateVCard_NameAndPhone(t *testing.T) {
	result := GenerateVCard("Bob", "+14155551234", "", "")
	assert.Contains(t, result, "FN:Bob")
	assert.Contains(t, result, "TEL;type=CELL;type=VOICE;waid=+14155551234:+14155551234")
	assert.NotContains(t, result, "EMAIL")
	assert.NotContains(t, result, "ORG")
}

func TestGenerateVCard_NoOrg(t *testing.T) {
	result := GenerateVCard("Alice", "+5511888888888", "alice@test.com", "")
	assert.Contains(t, result, "FN:Alice")
	assert.Contains(t, result, "TEL")
	assert.Contains(t, result, "EMAIL:alice@test.com")
	assert.NotContains(t, result, "ORG")
}

func TestGenerateVCard_EmptyName(t *testing.T) {
	result := GenerateVCard("", "+123456789", "", "")
	assert.True(t, strings.HasPrefix(result, "BEGIN:VCARD"))
	assert.Contains(t, result, "FN:")
	assert.Contains(t, result, "END:VCARD")
}
