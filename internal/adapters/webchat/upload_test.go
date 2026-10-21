package webchat

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/textproto"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/msgfy/linktor/internal/domain/entity"
	"github.com/msgfy/linktor/pkg/testutil"
)

// memStore é um storage.Client em memória: guarda o que recebeu para o teste
// conferir a chave, o MIME e o conteúdo.
type memStore struct {
	objects map[string][]byte
	mimes   map[string]string
	fail    bool
}

func newMemStore() *memStore {
	return &memStore{objects: map[string][]byte{}, mimes: map[string]string{}}
}

func (s *memStore) Upload(_ context.Context, key string, data []byte, contentType string) (string, error) {
	if s.fail {
		return "", errors.New("boom")
	}
	s.objects[key] = data
	s.mimes[key] = contentType
	return "http://media.test/api/v1/media/" + key, nil
}
func (s *memStore) Delete(context.Context, string) error                 { return nil }
func (s *memStore) GetURL(_ context.Context, key string) (string, error) { return key, nil }
func (s *memStore) Open(context.Context, string) (io.ReadCloser, string, int64, error) {
	return nil, "", 0, errors.New("not implemented")
}

func uploadChannel(config map[string]string) *entity.Channel {
	return &entity.Channel{
		ID:               "ch-webchat",
		TenantID:         "tenant-1",
		Type:             entity.ChannelTypeWebChat,
		Enabled:          true,
		ConnectionStatus: entity.ConnectionStatusConnected,
		Config:           config,
	}
}

func uploadRouter(h *Handler) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.POST("/api/v1/webchat/:channelId/upload", h.UploadMedia)
	return r
}

// multipartFile monta o corpo multipart com o campo "file" e o Content-Type
// da parte, que é o que o handler usa para decidir o tipo.
func multipartFile(t *testing.T, filename, mimeType string, data []byte) (*bytes.Buffer, string) {
	t.Helper()
	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	hdr := textproto.MIMEHeader{}
	hdr.Set("Content-Disposition", `form-data; name="file"; filename="`+filename+`"`)
	hdr.Set("Content-Type", mimeType)
	part, err := w.CreatePart(hdr)
	require.NoError(t, err)
	_, err = part.Write(data)
	require.NoError(t, err)
	require.NoError(t, w.Close())
	return &buf, w.FormDataContentType()
}

func postUpload(r *gin.Engine, query string, body *bytes.Buffer, contentType string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodPost, "/api/v1/webchat/ch-webchat/upload"+query, body)
	req.Header.Set("Content-Type", contentType)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	return rec
}

func TestUploadMedia_StoresImageAndAudio(t *testing.T) {
	channelRepo := testutil.NewMockChannelRepository()
	channelRepo.Channels["ch-webchat"] = uploadChannel(map[string]string{"allow_attachments": "true"})
	store := newMemStore()
	r := uploadRouter(NewHandler(nil, channelRepo, nil, nil, nil, store))

	cases := []struct {
		name, filename, mime, wantType, wantExt string
	}{
		{"png", "foto.png", "image/png", "image", ".png"},
		{"webm gravado no widget", "audio-1.webm", "audio/webm;codecs=opus", "audio", ".webm"},
		{"mp3", "musica.mp3", "audio/mpeg", "audio", ".mp3"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			body, ct := multipartFile(t, tc.filename, tc.mime, []byte("conteudo"))
			rec := postUpload(r, "", body, ct)
			require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())

			var resp UploadResponse
			require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
			assert.Equal(t, tc.wantType, resp.Type)
			assert.Equal(t, tc.filename, resp.Filename)
			assert.Equal(t, int64(len("conteudo")), resp.SizeBytes)
			assert.True(t, strings.HasPrefix(resp.URL, "http://media.test/api/v1/media/webchat/ch-webchat/"), resp.URL)
			// A extensão gravada vem do MIME, e o MIME devolvido é o base, sem parâmetros.
			assert.True(t, strings.HasSuffix(resp.URL, tc.wantExt), resp.URL)
			assert.Equal(t, strings.Split(tc.mime, ";")[0], resp.MimeType)
		})
	}
	assert.Len(t, store.objects, 3)
}

func TestUploadMedia_Rejections(t *testing.T) {
	t.Run("tipo fora da allowlist", func(t *testing.T) {
		channelRepo := testutil.NewMockChannelRepository()
		channelRepo.Channels["ch-webchat"] = uploadChannel(map[string]string{"allow_attachments": "true"})
		store := newMemStore()
		r := uploadRouter(NewHandler(nil, channelRepo, nil, nil, nil, store))

		for _, mime := range []string{"text/plain", "text/html", "application/pdf", "video/mp4", ""} {
			body, ct := multipartFile(t, "x.bin", mime, []byte("x"))
			rec := postUpload(r, "", body, ct)
			assert.Equal(t, http.StatusUnsupportedMediaType, rec.Code, "mime=%q", mime)
		}
		assert.Empty(t, store.objects, "nada pode ter sido gravado")
	})

	t.Run("acima de 10 MB", func(t *testing.T) {
		channelRepo := testutil.NewMockChannelRepository()
		channelRepo.Channels["ch-webchat"] = uploadChannel(map[string]string{"allow_attachments": "true"})
		store := newMemStore()
		r := uploadRouter(NewHandler(nil, channelRepo, nil, nil, nil, store))

		body, ct := multipartFile(t, "grande.png", "image/png", bytes.Repeat([]byte("a"), maxUploadBytes+1))
		rec := postUpload(r, "", body, ct)
		assert.Equal(t, http.StatusRequestEntityTooLarge, rec.Code)
		assert.Empty(t, store.objects)
	})

	t.Run("allow_attachments desligado", func(t *testing.T) {
		channelRepo := testutil.NewMockChannelRepository()
		channelRepo.Channels["ch-webchat"] = uploadChannel(map[string]string{})
		r := uploadRouter(NewHandler(nil, channelRepo, nil, nil, nil, newMemStore()))

		body, ct := multipartFile(t, "foto.png", "image/png", []byte("x"))
		rec := postUpload(r, "", body, ct)
		assert.Equal(t, http.StatusForbidden, rec.Code)
	})

	t.Run("widget token obrigatório quando há secret", func(t *testing.T) {
		channelRepo := testutil.NewMockChannelRepository()
		channelRepo.Channels["ch-webchat"] = uploadChannel(map[string]string{
			"allow_attachments": "true",
			"widget_secret":     "top-secret",
		})
		r := uploadRouter(NewHandler(nil, channelRepo, nil, nil, nil, newMemStore()))

		body, ct := multipartFile(t, "foto.png", "image/png", []byte("x"))
		rec := postUpload(r, "", body, ct)
		assert.Equal(t, http.StatusUnauthorized, rec.Code, "sem token")

		body, ct = multipartFile(t, "foto.png", "image/png", []byte("x"))
		rec = postUpload(r, "?token=invalido", body, ct)
		assert.Equal(t, http.StatusUnauthorized, rec.Code, "token inválido")

		token, err := IssueWidgetToken("ch-webchat", "top-secret", time.Hour, time.Now())
		require.NoError(t, err)
		body, ct = multipartFile(t, "foto.png", "image/png", []byte("x"))
		rec = postUpload(r, "?token="+token, body, ct)
		assert.Equal(t, http.StatusOK, rec.Code, "token válido: %s", rec.Body.String())
	})

	t.Run("sem storage configurado", func(t *testing.T) {
		channelRepo := testutil.NewMockChannelRepository()
		channelRepo.Channels["ch-webchat"] = uploadChannel(map[string]string{"allow_attachments": "true"})
		r := uploadRouter(NewHandler(nil, channelRepo, nil, nil, nil, nil))

		body, ct := multipartFile(t, "foto.png", "image/png", []byte("x"))
		rec := postUpload(r, "", body, ct)
		assert.Equal(t, http.StatusServiceUnavailable, rec.Code)
	})

	t.Run("canal inexistente", func(t *testing.T) {
		r := uploadRouter(NewHandler(nil, testutil.NewMockChannelRepository(), nil, nil, nil, newMemStore()))
		body, ct := multipartFile(t, "foto.png", "image/png", []byte("x"))
		rec := postUpload(r, "", body, ct)
		assert.Equal(t, http.StatusNotFound, rec.Code)
	})

	t.Run("falha no storage", func(t *testing.T) {
		channelRepo := testutil.NewMockChannelRepository()
		channelRepo.Channels["ch-webchat"] = uploadChannel(map[string]string{"allow_attachments": "true"})
		store := newMemStore()
		store.fail = true
		r := uploadRouter(NewHandler(nil, channelRepo, nil, nil, nil, store))

		body, ct := multipartFile(t, "foto.png", "image/png", []byte("x"))
		rec := postUpload(r, "", body, ct)
		assert.Equal(t, http.StatusInternalServerError, rec.Code)
	})
}
