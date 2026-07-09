package vendax

import (
	"strings"

	"github.com/msgfy/linktor/internal/domain/entity"
)

// Vocabulário canônico de canal do VendaX Core (CA-06 ChannelConfigData.Types). O envelope trocado
// com o Core usa SEMPRE estes valores; os subtipos do Linktor (whatsapp_official/unofficial/…) são
// detalhe interno e não vazam para o Core.
const (
	coreChannelWhatsApp  = "WHATSAPP"
	coreChannelTelegram  = "TELEGRAM"
	coreChannelInstagram = "INSTAGRAM"
	coreChannelMessenger = "MESSENGER"
	coreChannelSMS       = "SMS"
)

// coreChannelType normaliza um channel_type do Linktor para o tipo canônico do Core. É o inverso de
// linktorChannelTypes. Usado no inbound para que o Core veja sempre o vocabulário canônico.
func coreChannelType(linktorType string) string {
	switch entity.ChannelType(linktorType) {
	case entity.ChannelTypeWhatsApp, entity.ChannelTypeWhatsAppOfficial, entity.ChannelTypeWhatsAppUnofficial:
		return coreChannelWhatsApp
	case entity.ChannelTypeTelegram:
		return coreChannelTelegram
	case entity.ChannelTypeInstagram:
		return coreChannelInstagram
	case entity.ChannelTypeFacebook:
		return coreChannelMessenger
	case entity.ChannelTypeSMS:
		return coreChannelSMS
	default:
		return strings.ToUpper(linktorType)
	}
}

// linktorChannelTypes mapeia o tipo canônico do Core para os tipos equivalentes no Linktor. WhatsApp
// do Core pode ser oficial, não-oficial ou o genérico no Linktor — todos são candidatos (por
// identifier). Usado no channel.config e no outbound para resolver o canal/conversa do Linktor.
func linktorChannelTypes(coreType string) []entity.ChannelType {
	switch strings.ToUpper(coreType) {
	case coreChannelWhatsApp:
		return []entity.ChannelType{
			entity.ChannelTypeWhatsAppOfficial,
			entity.ChannelTypeWhatsAppUnofficial,
			entity.ChannelTypeWhatsApp,
		}
	case coreChannelTelegram:
		return []entity.ChannelType{entity.ChannelTypeTelegram}
	case coreChannelInstagram:
		return []entity.ChannelType{entity.ChannelTypeInstagram}
	case coreChannelMessenger:
		return []entity.ChannelType{entity.ChannelTypeFacebook}
	case coreChannelSMS:
		return []entity.ChannelType{entity.ChannelTypeSMS}
	default:
		return []entity.ChannelType{entity.ChannelType(strings.ToLower(coreType))}
	}
}
