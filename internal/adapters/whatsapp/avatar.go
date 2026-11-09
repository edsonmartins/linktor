package whatsapp

import (
	"context"
	"errors"
	"fmt"

	"github.com/msgfy/linktor/pkg/plugin"
	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/types"
)

var _ plugin.ContactAvatarFetcher = (*Adapter)(nil)

// GetContactAvatar implementa plugin.ContactAvatarFetcher sobre a sessão
// whatsmeow do canal.
func (a *Adapter) GetContactAvatar(ctx context.Context, contact, existingID string) (*plugin.ContactAvatar, error) {
	jid, err := avatarTargetJID(contact)
	if err != nil {
		return nil, err
	}

	a.mu.RLock()
	client := a.client
	a.mu.RUnlock()
	if client == nil {
		return nil, plugin.ErrAvatarSessionUnavailable
	}

	info, err := client.GetAvatar(ctx, jid, existingID)
	return contactAvatarFrom(info, err)
}

// contactAvatarFrom traduz a resposta do whatsmeow para o contrato do plugin.
// Separado do adapter para o teste cobrir o mapeamento sem sessão de verdade.
func contactAvatarFrom(info *AvatarInfo, err error) (*plugin.ContactAvatar, error) {
	switch {
	case errors.Is(err, whatsmeow.ErrProfilePictureUnauthorized):
		return nil, plugin.ErrAvatarHidden
	case errors.Is(err, whatsmeow.ErrProfilePictureNotSet):
		return nil, plugin.ErrAvatarNotSet
	case errors.Is(err, ErrClientNotReady):
		return nil, plugin.ErrAvatarSessionUnavailable
	case err != nil:
		return nil, err
	}
	if info == nil {
		return nil, plugin.ErrAvatarNotSet
	}
	if info.Unchanged {
		return &plugin.ContactAvatar{PictureID: info.PictureID, Unchanged: true}, nil
	}
	if info.URL == "" {
		return nil, plugin.ErrAvatarNotSet
	}
	return &plugin.ContactAvatar{PictureID: info.PictureID, URL: info.URL}, nil
}

// avatarTargetJID aceita só contato de pessoa: número em dígitos, JID de
// usuário (@s.whatsapp.net) ou LID. Grupo, broadcast e newsletter ficam de fora.
func avatarTargetJID(contact string) (types.JID, error) {
	jid, err := resolveRecipientJID(contact)
	if err != nil {
		return types.EmptyJID, fmt.Errorf("invalid contact: %w", err)
	}
	if jid.User == "" || (jid.Server != types.DefaultUserServer && jid.Server != types.HiddenUserServer) {
		return types.EmptyJID, fmt.Errorf("invalid contact %q: use digits, a @s.whatsapp.net JID or a @lid", contact)
	}
	return jid.ToNonAD(), nil
}
