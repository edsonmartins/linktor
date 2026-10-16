package whatsapp

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// O WhatsApp exige o tipo na mensagem de mídia: imagem com Mimetype vazio é
// aceita pelo servidor e não chega ao destinatário.
func TestMediaMimeType(t *testing.T) {
	png := []byte{0x89, 'P', 'N', 'G', '\r', '\n', 0x1a, '\n', 0, 0, 0, 0}
	jpeg := []byte{0xFF, 0xD8, 0xFF, 0xE0, 0, 0x10, 'J', 'F', 'I', 'F', 0}
	pdf := []byte("%PDF-1.4\n%")

	assert.Equal(t, "image/jpeg", mediaMimeType("image/jpeg", png), "o tipo declarado vale")
	assert.Equal(t, "image/png", mediaMimeType("", png), "sem tipo, detecta pelo conteúdo")
	assert.Equal(t, "image/jpeg", mediaMimeType("  ", jpeg))
	assert.Equal(t, "application/pdf", mediaMimeType("", pdf))
}
