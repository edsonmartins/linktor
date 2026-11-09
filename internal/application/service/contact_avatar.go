package service

import (
	"context"
	stderrors "errors"
	"fmt"
	"io"
	"math"
	"mime"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/msgfy/linktor/internal/domain/entity"
	"github.com/msgfy/linktor/pkg/errors"
	"github.com/msgfy/linktor/pkg/logger"
	"github.com/msgfy/linktor/pkg/plugin"
	"go.uber.org/zap"
	"golang.org/x/time/rate"
)

// DefaultContactAvatarInterval é o ritmo padrão de consulta de foto por canal:
// uma por segundo. Número não oficial consultando centenas de fotos de uma vez
// é o tipo de volume que chama o antibloqueio do WhatsApp; este limite protege
// o número de qualquer chamador, não só do que já se limita do lado dele.
const DefaultContactAvatarInterval = time.Second

const (
	// maxContactAvatarBytes limita a imagem baixada do CDN. A foto em tamanho
	// cheio do WhatsApp fica na casa das centenas de KB.
	maxContactAvatarBytes = 5 << 20
	avatarDownloadTimeout = 20 * time.Second
)

// avatarHostSuffixes são os domínios de onde a foto pode ser baixada. A URL vem
// do WhatsApp — ou, num canal whatsapp_remote, do bridge no aparelho do
// usuário; sem esta lista, um bridge adulterado faria o servidor buscar
// qualquer endereço da rede interna.
var avatarHostSuffixes = []string{".whatsapp.net", ".fbcdn.net"}

// contactAvatarChannelTypes são os tipos com sessão do WhatsApp em device.
// O WhatsApp oficial (Cloud API) não dá a foto do cliente.
var contactAvatarChannelTypes = map[entity.ChannelType]bool{
	entity.ChannelTypeWhatsApp:                           true,
	entity.ChannelTypeWhatsAppUnofficial:                 true,
	entity.ChannelType(plugin.ChannelTypeWhatsAppRemote): true,
}

var (
	avatarPhonePattern = regexp.MustCompile(`^[0-9]{8,15}$`)
	avatarJIDPattern   = regexp.MustCompile(`^[0-9]{1,20}@(s\.whatsapp\.net|lid)$`)
)

// ContactAvatar é a foto do contato pronta para a resposta: os bytes da imagem
// e o PictureID que a identifica, ou Unchanged quando o PictureID que o
// chamador já tem continua valendo (sem bytes).
type ContactAvatar struct {
	PictureID   string
	Unchanged   bool
	Image       []byte
	ContentType string
}

// contactAvatarState é o estado da consulta de foto, que vive no ChannelService
// mas só serve a GetContactAvatar.
type contactAvatarState struct {
	mu       sync.Mutex
	interval time.Duration
	limiters map[string]*rate.Limiter
	http     *http.Client
}

// SetContactAvatarInterval define o intervalo mínimo entre consultas de foto
// num mesmo canal. Zero ou negativo volta ao padrão.
func (s *ChannelService) SetContactAvatarInterval(d time.Duration) {
	s.avatar.mu.Lock()
	defer s.avatar.mu.Unlock()
	s.avatar.interval = d
	s.avatar.limiters = nil
}

// allowAvatarLookup consome uma vaga do ritmo do canal; quando não há vaga,
// devolve quanto esperar.
func (s *ChannelService) allowAvatarLookup(channelID string) (bool, time.Duration) {
	s.avatar.mu.Lock()
	defer s.avatar.mu.Unlock()
	interval := s.avatar.interval
	if interval <= 0 {
		interval = DefaultContactAvatarInterval
	}
	if s.avatar.limiters == nil {
		s.avatar.limiters = map[string]*rate.Limiter{}
	}
	lim, ok := s.avatar.limiters[channelID]
	if !ok {
		lim = rate.NewLimiter(rate.Every(interval), 1)
		s.avatar.limiters[channelID] = lim
	}
	r := lim.Reserve()
	if d := r.Delay(); d > 0 {
		r.Cancel()
		return false, d
	}
	return true, 0
}

func (s *ChannelService) avatarHTTPClient() *http.Client {
	s.avatar.mu.Lock()
	defer s.avatar.mu.Unlock()
	if s.avatar.http == nil {
		s.avatar.http = &http.Client{
			Timeout: avatarDownloadTimeout,
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				if len(via) >= 3 {
					return fmt.Errorf("too many redirects")
				}
				return avatarURLCheck(req.URL)
			},
		}
	}
	return s.avatar.http
}

// GetContactAvatar devolve a foto de perfil de um contato, consultada na
// sessão do WhatsApp do canal (embutida ou no bridge) e baixada no ato.
//
// Nada é guardado: nem a URL do CDN, cuja validade não foi medida, nem a
// imagem. Quem guarda a foto é o chamador, pelo PictureID.
//
// Erros que o chamador distingue: plugin.ErrAvatarNotSet e
// plugin.ErrAvatarHidden (embrulhados em AppError NotFound); conflito quando
// não há sessão viva; RateLimited com Details["retryAfter"] em segundos.
func (s *ChannelService) GetContactAvatar(
	ctx context.Context, tenantID, channelID, contact, existingID string,
) (*ContactAvatar, error) {
	contact = strings.TrimSpace(contact)
	if !avatarPhonePattern.MatchString(contact) && !avatarJIDPattern.MatchString(contact) {
		return nil, errors.New(errors.ErrCodeValidation,
			"contact must be a phone in digits with country code, a <number>@s.whatsapp.net JID or a <id>@lid")
	}

	channel, err := s.repo.FindByID(ctx, channelID)
	if err != nil {
		return nil, err
	}
	if channel == nil || channel.TenantID != tenantID {
		return nil, errors.New(errors.ErrCodeNotFound, "channel not found")
	}
	if !contactAvatarChannelTypes[channel.Type] {
		return nil, errors.New(errors.ErrCodeNotImplemented,
			fmt.Sprintf("channel type %q has no contact avatar; only WhatsApp sessions on a device do", channel.Type))
	}
	if s.registry == nil {
		return nil, errors.New(errors.ErrCodeInternal, "channel registry not configured")
	}

	pluginAdapter, err := s.registry.GetAdapterByChannelID(channelID)
	if err != nil || pluginAdapter == nil {
		return nil, errors.New(errors.ErrCodeConflict,
			"no live WhatsApp session for this channel; start the connection again")
	}
	fetcher, ok := pluginAdapter.(plugin.ContactAvatarFetcher)
	if !ok {
		return nil, errors.New(errors.ErrCodeConflict, "channel session cannot look up contact avatars")
	}

	if ok, wait := s.allowAvatarLookup(channelID); !ok {
		secs := int(math.Ceil(wait.Seconds()))
		if secs < 1 {
			secs = 1
		}
		return nil, errors.New(errors.ErrCodeRateLimited, "contact avatar rate limit for this channel").
			WithDetails(map[string]string{"retryAfter": strconv.Itoa(secs)})
	}

	info, err := fetcher.GetContactAvatar(ctx, contact, existingID)
	switch {
	case stderrors.Is(err, plugin.ErrAvatarNotSet), stderrors.Is(err, plugin.ErrAvatarHidden):
		return nil, errors.Wrap(err, errors.ErrCodeNotFound, err.Error())
	case stderrors.Is(err, plugin.ErrAvatarSessionUnavailable):
		return nil, errors.Wrap(err, errors.ErrCodeConflict,
			"no live WhatsApp session for this channel; start the connection again")
	case err != nil:
		logger.Warn("Contact avatar lookup failed",
			zap.String("channel_id", channelID), zap.Error(err))
		return nil, errors.Wrap(err, errors.ErrCodeBadGateway, "WhatsApp did not answer the avatar lookup")
	}

	if info.Unchanged {
		return &ContactAvatar{PictureID: info.PictureID, Unchanged: true}, nil
	}

	image, contentType, err := s.downloadAvatar(ctx, info.URL)
	if err != nil {
		logger.Warn("Contact avatar download failed",
			zap.String("channel_id", channelID), zap.Error(err))
		return nil, errors.Wrap(err, errors.ErrCodeBadGateway, "could not download the contact avatar from WhatsApp")
	}
	return &ContactAvatar{PictureID: info.PictureID, Image: image, ContentType: contentType}, nil
}

// avatarURLCheck é trocável nos testes, que servem a imagem de um httptest.
var avatarURLCheck = checkAvatarURL

// checkAvatarURL só deixa passar HTTPS nos domínios do CDN do WhatsApp.
func checkAvatarURL(u *url.URL) error {
	if u == nil || u.Scheme != "https" {
		return fmt.Errorf("avatar URL must be https")
	}
	host := strings.ToLower(u.Hostname())
	for _, suffix := range avatarHostSuffixes {
		if strings.HasSuffix(host, suffix) {
			return nil
		}
	}
	return fmt.Errorf("avatar URL host %q is not a WhatsApp CDN", host)
}

func (s *ChannelService) downloadAvatar(ctx context.Context, rawURL string) ([]byte, string, error) {
	u, err := url.Parse(rawURL)
	if err != nil {
		return nil, "", fmt.Errorf("parse avatar URL: %w", err)
	}
	if err := avatarURLCheck(u); err != nil {
		return nil, "", err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, "", err
	}
	resp, err := s.avatarHTTPClient().Do(req)
	if err != nil {
		return nil, "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, "", fmt.Errorf("avatar CDN answered %d", resp.StatusCode)
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, maxContactAvatarBytes+1))
	if err != nil {
		return nil, "", err
	}
	if len(body) == 0 {
		return nil, "", fmt.Errorf("avatar CDN returned an empty body")
	}
	if len(body) > maxContactAvatarBytes {
		return nil, "", fmt.Errorf("avatar larger than %d bytes", maxContactAvatarBytes)
	}

	contentType, _, _ := mime.ParseMediaType(resp.Header.Get("Content-Type"))
	if !strings.HasPrefix(contentType, "image/") {
		contentType, _, _ = mime.ParseMediaType(http.DetectContentType(body))
	}
	if !strings.HasPrefix(contentType, "image/") {
		return nil, "", fmt.Errorf("avatar CDN returned %q, not an image", contentType)
	}
	return body, contentType, nil
}
