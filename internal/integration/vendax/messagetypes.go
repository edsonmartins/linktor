package vendax

import (
	"strings"

	"github.com/msgfy/linktor/internal/domain/entity"
)

// Mapeamento de tipos de mensagem entre o content_type do Linktor e o messageType do Core (ADR-010).
// No L3 tratamos texto e mídia (áudio/imagem/vídeo/documento). Rich objects do Core (quote/suggestion/
// boleto/tracking/credit) não são entregues ao canal como tal — ver deliverableToChannel.

// coreMessageType normaliza o content_type do Linktor para o messageType do Core (inbound).
func coreMessageType(ct entity.ContentType) string {
	switch ct {
	case entity.ContentTypeText:
		return "text"
	case entity.ContentTypeImage:
		return "image"
	case entity.ContentTypeAudio:
		return "audio"
	case entity.ContentTypeVideo:
		return "video"
	case entity.ContentTypeDocument:
		return "document"
	default:
		return string(ct) // location/contact/sticker/etc — passa como está
	}
}

// linktorContentType mapeia o messageType do Core para o ContentType do Linktor (outbound). Rich
// objects e tipos desconhecidos caem em text (mas só chegam aqui se deliverableToChannel permitir).
func linktorContentType(coreType string) entity.ContentType {
	switch strings.ToLower(coreType) {
	case "image":
		return entity.ContentTypeImage
	case "audio":
		return entity.ContentTypeAudio
	case "video":
		return entity.ContentTypeVideo
	case "document":
		return entity.ContentTypeDocument
	default:
		return entity.ContentTypeText
	}
}

// isMediaType diz se o ContentType do Linktor é mídia (entregue via anexo, não texto).
func isMediaType(ct entity.ContentType) bool {
	switch ct {
	case entity.ContentTypeImage, entity.ContentTypeAudio, entity.ContentTypeVideo, entity.ContentTypeDocument:
		return true
	default:
		return false
	}
}

// deliverableToChannel diz se o messageType do Core pode ser entregue ao cliente no canal. Texto e
// mídia sim; rich objects (quote/suggestion/…) não — como renderizá-los ao cliente é decisão de
// produto (pendência L3+), e não queremos vazar JSON cru ao canal.
func deliverableToChannel(coreMessageType string) bool {
	switch strings.ToLower(coreMessageType) {
	case "text", "image", "audio", "video", "document":
		return true
	default:
		return false
	}
}
