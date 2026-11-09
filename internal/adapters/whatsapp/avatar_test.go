package whatsapp

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/msgfy/linktor/pkg/plugin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/types"
)

// Os dois erros do whatsmeow viram os dois motivos distintos do contrato: quem
// guarda a foto apaga nos dois, mas só o "oculta" é escolha do titular.
func TestContactAvatarFrom_ErrosDoWhatsmeow(t *testing.T) {
	// O whatsmeow embrulha o erro da IQ (wrapIQError); o mapeamento usa errors.Is.
	hidden := fmt.Errorf("%w: %w", whatsmeow.ErrProfilePictureUnauthorized, whatsmeow.ErrIQNotAuthorized)
	_, err := contactAvatarFrom(nil, hidden)
	assert.ErrorIs(t, err, plugin.ErrAvatarHidden)
	assert.NotErrorIs(t, err, plugin.ErrAvatarNotSet)

	notSet := fmt.Errorf("%w: %w", whatsmeow.ErrProfilePictureNotSet, whatsmeow.ErrIQNotFound)
	_, err = contactAvatarFrom(nil, notSet)
	assert.ErrorIs(t, err, plugin.ErrAvatarNotSet)
	assert.NotErrorIs(t, err, plugin.ErrAvatarHidden)

	// status 204 do servidor: o whatsmeow devolve o erro sem embrulho
	_, err = contactAvatarFrom(nil, whatsmeow.ErrProfilePictureNotSet)
	assert.ErrorIs(t, err, plugin.ErrAvatarNotSet)
}

func TestContactAvatarFrom_Respostas(t *testing.T) {
	_, err := contactAvatarFrom(nil, ErrClientNotReady)
	assert.ErrorIs(t, err, plugin.ErrAvatarSessionUnavailable)

	other := errors.New("iq timed out")
	_, err = contactAvatarFrom(nil, other)
	assert.ErrorIs(t, err, other)

	// AvatarInfo vazio = sem foto
	_, err = contactAvatarFrom(&AvatarInfo{}, nil)
	assert.ErrorIs(t, err, plugin.ErrAvatarNotSet)
	_, err = contactAvatarFrom(nil, nil)
	assert.ErrorIs(t, err, plugin.ErrAvatarNotSet)

	got, err := contactAvatarFrom(&AvatarInfo{PictureID: "p1", Unchanged: true}, nil)
	require.NoError(t, err)
	assert.Equal(t, &plugin.ContactAvatar{PictureID: "p1", Unchanged: true}, got)

	got, err = contactAvatarFrom(&AvatarInfo{PictureID: "p2", URL: "https://pps.whatsapp.net/x"}, nil)
	require.NoError(t, err)
	assert.Equal(t, &plugin.ContactAvatar{PictureID: "p2", URL: "https://pps.whatsapp.net/x"}, got)
}

func TestAvatarTargetJID(t *testing.T) {
	jid, err := avatarTargetJID("5544999990000")
	require.NoError(t, err)
	assert.Equal(t, types.NewJID("5544999990000", types.DefaultUserServer), jid)

	jid, err = avatarTargetJID("123456789@lid")
	require.NoError(t, err)
	assert.Equal(t, types.HiddenUserServer, jid.Server)

	jid, err = avatarTargetJID("5544999990000:12@s.whatsapp.net")
	require.NoError(t, err)
	assert.Equal(t, uint16(0), jid.Device, "a foto é da pessoa, não do aparelho")

	for _, bad := range []string{"120363000@g.us", "status@broadcast"} {
		_, err := avatarTargetJID(bad)
		assert.Error(t, err, bad)
	}
}

func TestGetContactAvatar_SemSessao(t *testing.T) {
	a := NewAdapter()
	_, err := a.GetContactAvatar(context.Background(), "5544999990000", "")
	assert.ErrorIs(t, err, plugin.ErrAvatarSessionUnavailable)
}
