package service

import (
	"testing"

	"github.com/msgfy/linktor/internal/domain/entity"
	"github.com/stretchr/testify/assert"
)

func TestEmailReplyMetadata(t *testing.T) {
	recebida := &entity.Message{
		ExternalID: "abc@cliente.dev",
		Metadata:   map[string]string{"subject": "Orçamento nº 1", "references": "<anterior@cliente.dev>"},
	}

	t.Run("preenche assunto e thread a partir da última recebida", func(t *testing.T) {
		meta := map[string]string{}
		emailReplyMetadata(recebida, meta)
		assert.Equal(t, "Re: Orçamento nº 1", meta["subject"])
		assert.Equal(t, "<abc@cliente.dev>", meta["in_reply_to"])
		assert.Equal(t, "<anterior@cliente.dev> <abc@cliente.dev>", meta["references"])
	})

	t.Run("não duplica o Re:", func(t *testing.T) {
		meta := map[string]string{}
		emailReplyMetadata(&entity.Message{ExternalID: "x@y", Metadata: map[string]string{"subject": "RE: Pedido"}}, meta)
		assert.Equal(t, "RE: Pedido", meta["subject"])
	})

	t.Run("o caller prevalece", func(t *testing.T) {
		meta := map[string]string{"subject": "Outro assunto", "in_reply_to": "<meu@id>"}
		emailReplyMetadata(recebida, meta)
		assert.Equal(t, "Outro assunto", meta["subject"])
		assert.Equal(t, "<meu@id>", meta["in_reply_to"])
		assert.Equal(t, "<anterior@cliente.dev> <abc@cliente.dev>", meta["references"])
	})

	t.Run("sem id externo não inventa thread", func(t *testing.T) {
		meta := map[string]string{}
		emailReplyMetadata(&entity.Message{Metadata: map[string]string{"subject": "Oi"}}, meta)
		assert.Equal(t, "Re: Oi", meta["subject"])
		assert.Empty(t, meta["in_reply_to"])
		assert.Empty(t, meta["references"])
	})

	t.Run("id guardado com colchetes não ganha dois", func(t *testing.T) {
		meta := map[string]string{}
		emailReplyMetadata(&entity.Message{ExternalID: "<x@y>", Metadata: map[string]string{}}, meta)
		assert.Equal(t, "<x@y>", meta["in_reply_to"])
		assert.Empty(t, meta["subject"])
	})
}
