package entity

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNormalizeContactIdentifier(t *testing.T) {
	assert.Equal(t, "+5511987654321", NormalizeContactIdentifier("+55 (11) 98765-4321"))
	assert.Equal(t, "+5511987654321", NormalizeContactIdentifier("5511987654321"))
	assert.Equal(t, "cliente@exemplo.test", NormalizeContactIdentifier("  cliente@exemplo.test "), "canal não telefônico")
	assert.Equal(t, "", NormalizeContactIdentifier("   "))
}

func TestConversationOwnerRule_VigenteEm(t *testing.T) {
	agora := time.Date(2026, 11, 4, 12, 0, 0, 0, time.UTC)
	semPrazo := &ConversationOwnerRule{Desde: agora.Add(-time.Hour)}
	assert.True(t, semPrazo.VigenteEm(agora))
	depois := agora.Add(time.Minute)
	assert.True(t, (&ConversationOwnerRule{ExpiraEm: &depois}).VigenteEm(agora))
	assert.False(t, (&ConversationOwnerRule{ExpiraEm: &agora}).VigenteEm(agora), "expira_em é o fim exclusivo")
	var nenhuma *ConversationOwnerRule
	assert.False(t, nenhuma.VigenteEm(agora))
}

func TestValidConsumerName(t *testing.T) {
	assert.True(t, ValidConsumerName("crm"))
	assert.True(t, ValidConsumerName("copilot"))
	assert.True(t, ValidConsumerName("sales_copilot-2"))
	assert.False(t, ValidConsumerName("CRM"))
	assert.False(t, ValidConsumerName(""))
	assert.False(t, ValidConsumerName("a.b"), "o ponto separa a chave da configuração do canal")
}
