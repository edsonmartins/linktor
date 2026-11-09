package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/msgfy/linktor/internal/application/service"
	"github.com/msgfy/linktor/internal/domain/entity"
	"github.com/msgfy/linktor/pkg/plugin"
	"github.com/msgfy/linktor/pkg/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type stubAvatarAdapter struct {
	plugin.ChannelAdapter
	avatar *plugin.ContactAvatar
	err    error
	gotID  string
}

func (s *stubAvatarAdapter) GetContactAvatar(_ context.Context, _, existingID string) (*plugin.ContactAvatar, error) {
	s.gotID = existingID
	return s.avatar, s.err
}

func avatarRouter(t *testing.T, chType entity.ChannelType, adapter plugin.ChannelAdapter, interval time.Duration) *gin.Engine {
	t.Helper()
	repo := testutil.NewMockChannelRepository()
	seedChannel(repo, "ch-1", "tenant-1", "wa", chType)
	registry := plugin.NewRegistry()
	if adapter != nil {
		registry.RegisterChannelAdapter("ch-1", adapter)
	}
	producer := testutil.NewMockProducer()
	svc := service.NewChannelService(repo, registry, producer)
	svc.SetContactAvatarInterval(interval)
	h := NewChannelHandler(svc, producer, service.NewAuditService(nil))

	r := gin.New()
	r.Use(func(c *gin.Context) { c.Set("tenant_id", "tenant-1"); c.Next() })
	r.GET("/channels/:id/contacts/:contact/avatar", h.GetContactAvatar)
	return r
}

func getAvatar(r *gin.Engine, contact, ifNoneMatch string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodGet, "/channels/ch-1/contacts/"+contact+"/avatar", nil)
	if ifNoneMatch != "" {
		req.Header.Set("If-None-Match", ifNoneMatch)
	}
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func TestGetContactAvatarHandler_304ComIfNoneMatch(t *testing.T) {
	stub := &stubAvatarAdapter{avatar: &plugin.ContactAvatar{PictureID: "pic-1", Unchanged: true}}
	r := avatarRouter(t, entity.ChannelTypeWhatsApp, stub, time.Nanosecond)

	w := getAvatar(r, "5544999990000@s.whatsapp.net", `"pic-1"`)
	assert.Equal(t, http.StatusNotModified, w.Code)
	assert.Empty(t, w.Body.Bytes())
	assert.Equal(t, "pic-1", w.Header().Get("X-Picture-Id"))
	assert.Equal(t, `"pic-1"`, w.Header().Get("ETag"))
	assert.Equal(t, "pic-1", stub.gotID, "o pictureId do If-None-Match vai ao WhatsApp")
}

func TestGetContactAvatarHandler_404ComMotivo(t *testing.T) {
	for reason, err := range map[string]error{"not_set": plugin.ErrAvatarNotSet, "hidden": plugin.ErrAvatarHidden} {
		r := avatarRouter(t, entity.ChannelTypeWhatsApp, &stubAvatarAdapter{err: err}, time.Nanosecond)
		w := getAvatar(r, "5544999990000", "")
		require.Equal(t, http.StatusNotFound, w.Code, reason)
		var body map[string]any
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
		assert.Equal(t, reason, body["reason"])
	}
}

func TestGetContactAvatarHandler_429ComRetryAfter(t *testing.T) {
	stub := &stubAvatarAdapter{avatar: &plugin.ContactAvatar{PictureID: "p", Unchanged: true}}
	r := avatarRouter(t, entity.ChannelTypeWhatsApp, stub, 2*time.Second)
	require.Equal(t, http.StatusNotModified, getAvatar(r, "5544999990000", "p").Code)

	w := getAvatar(r, "5544999990000", "p")
	assert.Equal(t, http.StatusTooManyRequests, w.Code)
	assert.Equal(t, "2", w.Header().Get("Retry-After"))
}

func TestGetContactAvatarHandler_SemSessaoEOutroTipo(t *testing.T) {
	w := getAvatar(avatarRouter(t, entity.ChannelTypeWhatsApp, nil, time.Nanosecond), "5544999990000", "")
	assert.Equal(t, http.StatusConflict, w.Code)

	w = getAvatar(avatarRouter(t, entity.ChannelTypeTelegram, &stubAvatarAdapter{}, time.Nanosecond), "5544999990000", "")
	assert.Equal(t, http.StatusNotImplemented, w.Code)

	w = getAvatar(avatarRouter(t, entity.ChannelTypeWhatsApp, &stubAvatarAdapter{}, time.Nanosecond), "abc", "")
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestPictureIDFromIfNoneMatch(t *testing.T) {
	for in, want := range map[string]string{
		"":                "",
		`"abc"`:           "abc",
		`W/"abc"`:         "abc",
		"abc":             "abc",
		` "abc" , "def" `: "abc",
		"*":               "",
		`"1723456789"`:    "1723456789",
	} {
		assert.Equal(t, want, pictureIDFromIfNoneMatch(in), in)
	}
}
