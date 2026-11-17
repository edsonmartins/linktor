package main

import (
	"context"

	whatsappofficial "github.com/msgfy/linktor/internal/adapters/whatsapp_official"
	"github.com/msgfy/linktor/internal/domain/entity"
)

// conectorWhatsAppOficial é a parte de whatsappofficial.Conectar que o
// preparador usa; em teste entra um substituto.
type conectorWhatsAppOficial func(ctx context.Context, cfg *whatsappofficial.Config) (*whatsappofficial.PhoneNumberInfo, error)

// prepareWhatsAppOfficialChannel valida o token lendo o número e liga o app à
// conta do WhatsApp ao conectar o canal, como prepareFacebookChannel. Antes,
// Conectar só gravava "connected": token errado passava, e sem a ligação a Meta
// não entregava nenhuma mensagem. O número exibido fica em config para o card.
func prepareWhatsAppOfficialChannel(ctx context.Context, conectar conectorWhatsAppOficial, ch *entity.Channel) error {
	info, err := conectar(ctx, whatsappofficial.ConfigFromMap(ch.Config, ch.Credentials))
	if err != nil {
		return err
	}
	if ch.Config == nil {
		ch.Config = map[string]string{}
	}
	if info != nil && info.DisplayPhoneNumber != "" && ch.Config["phone_number"] == "" {
		ch.Config["phone_number"] = info.DisplayPhoneNumber
	}
	return nil
}
