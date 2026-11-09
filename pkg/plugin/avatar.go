package plugin

import (
	"context"
	"errors"
)

// ContactAvatar é a resposta de quem consultou a foto de perfil de um contato.
//
// Unchanged diz que o PictureID que o chamador já tinha continua sendo a foto
// atual: o WhatsApp responde "não mudou" sem URL, e quem pergunta guarda a cópia
// que já tem. Fora desse caso, URL aponta para a imagem no CDN do WhatsApp — uma
// URL assinada e de validade não medida, para ser baixada no ato e esquecida.
type ContactAvatar struct {
	PictureID string
	URL       string
	Unchanged bool
}

// Os dois "sem foto" vêm separados porque significam coisas diferentes para quem
// guarda a foto: NotSet é o contato sem foto; Hidden é o titular escolhendo
// esconder a foto deste número. Os dois apagam a cópia guardada, mas só o Hidden
// é uma escolha do titular, e é ele que a retenção registra.
var (
	ErrAvatarNotSet = errors.New("contact has no profile picture")
	ErrAvatarHidden = errors.New("contact hides the profile picture from this number")
	// ErrAvatarSessionUnavailable diz que não há sessão viva para perguntar
	// (whatsmeow desconectado, bridge fora do ar ou sem resposta).
	ErrAvatarSessionUnavailable = errors.New("no live WhatsApp session to look up the avatar")
)

// ContactAvatarFetcher é capacidade OPCIONAL do adapter: só quem tem uma sessão
// de WhatsApp em device (whatsmeow embutido ou bridge) consegue ver a foto do
// contato. contact é um JID (número@s.whatsapp.net ou @lid) ou o telefone em
// dígitos; existingID é o PictureID que o chamador já tem (vazio se nenhum).
type ContactAvatarFetcher interface {
	GetContactAvatar(ctx context.Context, contact, existingID string) (*ContactAvatar, error)
}
