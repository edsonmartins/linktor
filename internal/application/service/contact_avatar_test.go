package service

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/msgfy/linktor/internal/domain/entity"
	"github.com/msgfy/linktor/pkg/errors"
	"github.com/msgfy/linktor/pkg/plugin"
	"github.com/msgfy/linktor/pkg/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fakeAvatarAdapter é um adapter registrado que só sabe responder a foto.
type fakeAvatarAdapter struct {
	plugin.ChannelAdapter
	avatar  *plugin.ContactAvatar
	err     error
	calls   int
	gotID   string
	gotUser string
}

func (f *fakeAvatarAdapter) GetContactAvatar(_ context.Context, contact, existingID string) (*plugin.ContactAvatar, error) {
	f.calls++
	f.gotUser, f.gotID = contact, existingID
	return f.avatar, f.err
}

// adapterSemFoto é um adapter registrado sem a capacidade de foto.
type adapterSemFoto struct{ plugin.ChannelAdapter }

var jpegBytes = []byte{0xff, 0xd8, 0xff, 0xe0, 0, 0x10, 'J', 'F', 'I', 'F', 0, 1, 2, 3}

func newAvatarFixture(t *testing.T, chType entity.ChannelType, adapter plugin.ChannelAdapter) *ChannelService {
	t.Helper()
	repo := testutil.NewMockChannelRepository()
	repo.Channels["ch-1"] = &entity.Channel{ID: "ch-1", TenantID: "t-1", Type: chType}
	registry := plugin.NewRegistry()
	if adapter != nil {
		registry.RegisterChannelAdapter("ch-1", adapter)
	}
	svc := NewChannelService(repo, registry, testutil.NewMockProducer())
	// Sem esperar entre as consultas do mesmo teste, a não ser no de ritmo.
	svc.SetContactAvatarInterval(time.Nanosecond)
	return svc
}

// cdn sobe um servidor TLS no papel do CDN do WhatsApp e libera o host dele.
func cdn(t *testing.T, svc *ChannelService, handler http.HandlerFunc) string {
	t.Helper()
	srv := httptest.NewTLSServer(handler)
	t.Cleanup(srv.Close)
	svc.avatar.http = srv.Client()
	orig := avatarURLCheck
	avatarURLCheck = func(u *url.URL) error {
		if u.Scheme == "https" && u.Host == srv.Listener.Addr().String() {
			return nil
		}
		return checkAvatarURL(u)
	}
	t.Cleanup(func() { avatarURLCheck = orig })
	return srv.URL + "/pp.jpg"
}

func appCode(t *testing.T, err error) errors.ErrorCode {
	t.Helper()
	appErr := errors.GetAppError(err)
	require.NotNil(t, appErr, "esperava AppError, veio %v", err)
	return appErr.Code
}

func TestGetContactAvatar_BaixaOsBytes(t *testing.T) {
	fake := &fakeAvatarAdapter{}
	svc := newAvatarFixture(t, entity.ChannelTypeWhatsApp, fake)
	fake.avatar = &plugin.ContactAvatar{PictureID: "pic-2", URL: cdn(t, svc, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "image/jpeg")
		_, _ = w.Write(jpegBytes)
	})}

	got, err := svc.GetContactAvatar(context.Background(), "t-1", "ch-1", "5544999990000", "pic-1")
	require.NoError(t, err)
	assert.Equal(t, "pic-2", got.PictureID)
	assert.False(t, got.Unchanged)
	assert.Equal(t, jpegBytes, got.Image)
	assert.Equal(t, "image/jpeg", got.ContentType)
	assert.Equal(t, "5544999990000", fake.gotUser)
	assert.Equal(t, "pic-1", fake.gotID)
}

func TestGetContactAvatar_ContentTypeDetectadoQuandoOCDNNaoDiz(t *testing.T) {
	fake := &fakeAvatarAdapter{}
	svc := newAvatarFixture(t, entity.ChannelTypeWhatsApp, fake)
	fake.avatar = &plugin.ContactAvatar{PictureID: "p", URL: cdn(t, svc, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/octet-stream")
		_, _ = w.Write(jpegBytes)
	})}

	got, err := svc.GetContactAvatar(context.Background(), "t-1", "ch-1", "5544999990000", "")
	require.NoError(t, err)
	assert.Equal(t, "image/jpeg", got.ContentType)
}

func TestGetContactAvatar_NaoMudouNaoBaixa(t *testing.T) {
	fake := &fakeAvatarAdapter{avatar: &plugin.ContactAvatar{PictureID: "pic-1", Unchanged: true}}
	svc := newAvatarFixture(t, entity.ChannelTypeWhatsApp, fake)
	downloads := 0
	cdn(t, svc, func(w http.ResponseWriter, r *http.Request) { downloads++ })

	got, err := svc.GetContactAvatar(context.Background(), "t-1", "ch-1", "5544999990000@s.whatsapp.net", "pic-1")
	require.NoError(t, err)
	assert.True(t, got.Unchanged)
	assert.Equal(t, "pic-1", got.PictureID)
	assert.Nil(t, got.Image)
	assert.Zero(t, downloads)
}

func TestGetContactAvatar_SemFotoEOcultaSaoDistintas(t *testing.T) {
	for _, tc := range []struct {
		err  error
		want error
	}{
		{plugin.ErrAvatarNotSet, plugin.ErrAvatarNotSet},
		{plugin.ErrAvatarHidden, plugin.ErrAvatarHidden},
	} {
		fake := &fakeAvatarAdapter{err: tc.err}
		svc := newAvatarFixture(t, entity.ChannelTypeWhatsApp, fake)
		_, err := svc.GetContactAvatar(context.Background(), "t-1", "ch-1", "123456789012@lid", "")
		assert.ErrorIs(t, err, tc.want)
		assert.Equal(t, errors.ErrCodeNotFound, appCode(t, err))
	}
	// e um não é o outro
	fake := &fakeAvatarAdapter{err: plugin.ErrAvatarHidden}
	svc := newAvatarFixture(t, entity.ChannelTypeWhatsApp, fake)
	_, err := svc.GetContactAvatar(context.Background(), "t-1", "ch-1", "5544999990000", "")
	assert.NotErrorIs(t, err, plugin.ErrAvatarNotSet)
}

func TestGetContactAvatar_SemSessaoEhConflito(t *testing.T) {
	// nenhum adapter registrado
	svc := newAvatarFixture(t, entity.ChannelTypeWhatsApp, nil)
	_, err := svc.GetContactAvatar(context.Background(), "t-1", "ch-1", "5544999990000", "")
	assert.Equal(t, errors.ErrCodeConflict, appCode(t, err))

	// adapter registrado, sessão caída
	fake := &fakeAvatarAdapter{err: fmt.Errorf("%w: no bridge", plugin.ErrAvatarSessionUnavailable)}
	svc = newAvatarFixture(t, entity.ChannelType(plugin.ChannelTypeWhatsAppRemote), fake)
	_, err = svc.GetContactAvatar(context.Background(), "t-1", "ch-1", "5544999990000", "")
	assert.Equal(t, errors.ErrCodeConflict, appCode(t, err))

	// adapter sem a capacidade
	svc = newAvatarFixture(t, entity.ChannelTypeWhatsApp, adapterSemFoto{})
	_, err = svc.GetContactAvatar(context.Background(), "t-1", "ch-1", "5544999990000", "")
	assert.Equal(t, errors.ErrCodeConflict, appCode(t, err))
}

func TestGetContactAvatar_OutroTipoDeCanalNaoTemFoto(t *testing.T) {
	for _, chType := range []entity.ChannelType{entity.ChannelTypeWhatsAppOfficial, entity.ChannelTypeTelegram} {
		fake := &fakeAvatarAdapter{}
		svc := newAvatarFixture(t, chType, fake)
		_, err := svc.GetContactAvatar(context.Background(), "t-1", "ch-1", "5544999990000", "")
		assert.Equal(t, errors.ErrCodeNotImplemented, appCode(t, err), chType)
		assert.Zero(t, fake.calls)
	}
}

func TestGetContactAvatar_CanalDeOutroTenant(t *testing.T) {
	fake := &fakeAvatarAdapter{}
	svc := newAvatarFixture(t, entity.ChannelTypeWhatsApp, fake)
	_, err := svc.GetContactAvatar(context.Background(), "t-2", "ch-1", "5544999990000", "")
	assert.Equal(t, errors.ErrCodeNotFound, appCode(t, err))
	assert.Zero(t, fake.calls)
}

func TestGetContactAvatar_ContatoInvalido(t *testing.T) {
	fake := &fakeAvatarAdapter{}
	svc := newAvatarFixture(t, entity.ChannelTypeWhatsApp, fake)
	for _, contact := range []string{"", "+5544999990000", "abc", "123", "120363@g.us", "5544@broadcast", "1@s.whatsapp.net:2"} {
		_, err := svc.GetContactAvatar(context.Background(), "t-1", "ch-1", contact, "")
		assert.Equal(t, errors.ErrCodeValidation, appCode(t, err), contact)
	}
	assert.Zero(t, fake.calls)
}

func TestGetContactAvatar_RitmoPorCanal(t *testing.T) {
	fake := &fakeAvatarAdapter{avatar: &plugin.ContactAvatar{PictureID: "p", Unchanged: true}}
	svc := newAvatarFixture(t, entity.ChannelTypeWhatsApp, fake)
	svc.SetContactAvatarInterval(time.Hour)

	_, err := svc.GetContactAvatar(context.Background(), "t-1", "ch-1", "5544999990000", "p")
	require.NoError(t, err)

	_, err = svc.GetContactAvatar(context.Background(), "t-1", "ch-1", "5544999990001", "p")
	appErr := errors.GetAppError(err)
	require.NotNil(t, appErr)
	assert.Equal(t, errors.ErrCodeRateLimited, appErr.Code)
	assert.Equal(t, "3600", appErr.Details["retryAfter"])
	assert.Equal(t, 1, fake.calls, "a consulta barrada não chega ao WhatsApp")
}

func TestGetContactAvatar_URLForaDoCDNNaoEhBaixada(t *testing.T) {
	for _, raw := range []string{
		"http://pps.whatsapp.net/v/x.jpg",     // sem TLS
		"https://169.254.169.254/latest/meta", // rede interna
		"https://pps.whatsapp.net.evil.com/x", // sufixo falso
		"https://evilwhatsapp.net/x",          // sem o ponto
	} {
		fake := &fakeAvatarAdapter{avatar: &plugin.ContactAvatar{PictureID: "p", URL: raw}}
		svc := newAvatarFixture(t, entity.ChannelTypeWhatsApp, fake)
		_, err := svc.GetContactAvatar(context.Background(), "t-1", "ch-1", "5544999990000", "")
		assert.Equal(t, errors.ErrCodeBadGateway, appCode(t, err), raw)
	}
	u, _ := url.Parse("https://pps.whatsapp.net/v/t61/x.jpg?oh=1")
	assert.NoError(t, checkAvatarURL(u))
	u, _ = url.Parse("https://mmg.fbcdn.net/x.jpg")
	assert.NoError(t, checkAvatarURL(u))
}

func TestGetContactAvatar_CDNRecusaOuNaoEhImagem(t *testing.T) {
	for name, h := range map[string]http.HandlerFunc{
		"403": func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusForbidden) },
		"html": func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "text/html")
			_, _ = w.Write([]byte("<html>expired</html>"))
		},
		"vazio": func(w http.ResponseWriter, r *http.Request) {},
	} {
		fake := &fakeAvatarAdapter{}
		svc := newAvatarFixture(t, entity.ChannelTypeWhatsApp, fake)
		fake.avatar = &plugin.ContactAvatar{PictureID: "p", URL: cdn(t, svc, h)}
		_, err := svc.GetContactAvatar(context.Background(), "t-1", "ch-1", "5544999990000", "")
		assert.Equal(t, errors.ErrCodeBadGateway, appCode(t, err), name)
	}
}
