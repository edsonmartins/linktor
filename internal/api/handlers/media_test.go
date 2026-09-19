package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/msgfy/linktor/internal/api/middleware"
	"github.com/msgfy/linktor/internal/domain/entity"
	"github.com/msgfy/linktor/internal/infrastructure/storage"
	"github.com/msgfy/linktor/pkg/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// memMediaStore is an in-memory storage.Client for the media proxy.
type memMediaStore struct{ objects map[string][]byte }

func (m *memMediaStore) Upload(_ context.Context, key string, data []byte, _ string) (string, error) {
	m.objects[key] = data
	return "https://api.test" + storage.MediaProxyPath + key, nil
}
func (m *memMediaStore) Delete(_ context.Context, key string) error {
	delete(m.objects, key)
	return nil
}
func (m *memMediaStore) GetURL(_ context.Context, key string) (string, error) {
	return "https://api.test" + storage.MediaProxyPath + key, nil
}
func (m *memMediaStore) Open(_ context.Context, key string) (io.ReadCloser, string, int64, error) {
	data, ok := m.objects[key]
	if !ok {
		return nil, "", 0, errors.New("not found")
	}
	return io.NopCloser(bytes.NewReader(data)), "image/webp", int64(len(data)), nil
}

// memRevocations mirrors storage.MediaRevocations' key-or-ancestor semantics.
type memRevocations struct{ set map[string]bool }

func (r *memRevocations) Revoke(_ context.Context, k string) error { r.set[k] = true; return nil }
func (r *memRevocations) IsRevoked(_ context.Context, key string) (bool, error) {
	if r.set[key] {
		return true, nil
	}
	for i := range key {
		if key[i] == '/' && r.set[key[:i+1]] {
			return true, nil
		}
	}
	return false, nil
}

const (
	attKey     = "attachments/tenant-1/0b7c.webp"
	inboundKey = "inbound/whatsapp/ch-1/9f2e-foto.webp"
)

type mediaFixture struct {
	router  *gin.Engine
	signer  *storage.MediaSigner
	revoked *memRevocations
	handler *MediaHandler
}

// newMediaFixture wires the routes as main.go does. The X-Test-Tenant header
// stands in for the session/API key OptionalAuthenticate would resolve.
func newMediaFixture(t *testing.T, signer *storage.MediaSigner, unsignedUntil time.Time) *mediaFixture {
	t.Helper()
	gin.SetMode(gin.TestMode)
	store := &memMediaStore{objects: map[string][]byte{attKey: []byte("img"), inboundKey: []byte("inb")}}
	channels := testutil.NewMockChannelRepository()
	channels.Channels["ch-1"] = &entity.Channel{ID: "ch-1", TenantID: "tenant-1"}
	rev := &memRevocations{set: map[string]bool{}}
	h := NewMediaHandler(store, MediaAccess{Signer: signer, UnsignedUntil: unsignedUntil, Revocations: rev, Channels: channels})

	asTenant := func(c *gin.Context) {
		if tenant := c.GetHeader("X-Test-Tenant"); tenant != "" {
			c.Set(middleware.TenantIDKey, tenant)
		}
	}
	r := gin.New()
	api := r.Group("/api/v1")
	api.GET("/media/*key", asTenant, h.Serve)
	api.POST("/media/sign", asTenant, h.Sign)
	api.POST("/media/revoke", asTenant, h.Revoke)
	return &mediaFixture{router: r, signer: signer, revoked: rev, handler: h}
}

func (f *mediaFixture) get(t *testing.T, rawURL, tenant string) *httptest.ResponseRecorder {
	t.Helper()
	u, err := url.Parse(rawURL)
	require.NoError(t, err)
	req := httptest.NewRequest(http.MethodGet, u.RequestURI(), nil)
	if tenant != "" {
		req.Header.Set("X-Test-Tenant", tenant)
	}
	w := httptest.NewRecorder()
	f.router.ServeHTTP(w, req)
	return w
}

func (f *mediaFixture) post(t *testing.T, path, tenant string, body any) *httptest.ResponseRecorder {
	t.Helper()
	b, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, path, bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Test-Tenant", tenant)
	w := httptest.NewRecorder()
	f.router.ServeHTTP(w, req)
	return w
}

func newTestSigner(ttl time.Duration) *storage.MediaSigner {
	return storage.NewMediaSigner("test-secret", "https://api.test", ttl)
}

func TestMediaServe_NoSignerKeepsLegacyCapabilityURL(t *testing.T) {
	f := newMediaFixture(t, nil, time.Time{})
	w := f.get(t, storage.MediaProxyPath+attKey, "")
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "img", w.Body.String())
}

func TestMediaServe_ValidSignatureLoadsWithoutAnyHeader(t *testing.T) {
	signer := newTestSigner(time.Hour)
	f := newMediaFixture(t, signer, time.Now().Add(-time.Hour)) // transition over
	u, _ := signer.SignKey(attKey, 0)

	w := f.get(t, u, "")
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "img", w.Body.String())
	assert.NotContains(t, w.Header().Get("Cache-Control"), "immutable")
}

func TestMediaServe_ExpiredSignatureIs410(t *testing.T) {
	signer := newTestSigner(time.Second)
	f := newMediaFixture(t, signer, time.Time{})
	u, _ := signer.SignKey(attKey, 0)
	time.Sleep(1100 * time.Millisecond)

	w := f.get(t, u, "")
	assert.Equal(t, http.StatusGone, w.Code)
	assert.Empty(t, w.Body.String(), "an expired URL must not return the file")
}

func TestMediaServe_TamperedSignatureIs403(t *testing.T) {
	signer := newTestSigner(time.Hour)
	f := newMediaFixture(t, signer, time.Time{})
	u, _ := signer.SignKey(attKey, 0)
	// Same signature, another object.
	other := strings.Replace(u, attKey, inboundKey, 1)

	assert.Equal(t, http.StatusForbidden, f.get(t, other, "").Code)
}

func TestMediaServe_UnsignedDuringAndAfterTransition(t *testing.T) {
	signer := newTestSigner(time.Hour)
	bare := storage.MediaProxyPath + attKey

	open := newMediaFixture(t, signer, time.Time{}) // no cut-over date agreed yet
	w := open.get(t, bare, "")
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "true", w.Header().Get("Deprecation"))

	before := newMediaFixture(t, signer, time.Now().Add(24*time.Hour))
	assert.Equal(t, http.StatusOK, before.get(t, bare, "").Code)

	after := newMediaFixture(t, signer, time.Now().Add(-time.Minute))
	assert.Equal(t, http.StatusUnauthorized, after.get(t, bare, "").Code)
}

func TestMediaServe_OwningTenantSessionNeedsNoSignature(t *testing.T) {
	signer := newTestSigner(time.Hour)
	f := newMediaFixture(t, signer, time.Now().Add(-time.Minute))

	assert.Equal(t, http.StatusOK, f.get(t, storage.MediaProxyPath+attKey, "tenant-1").Code)
	assert.Equal(t, http.StatusOK, f.get(t, storage.MediaProxyPath+inboundKey, "tenant-1").Code, "inbound key owned via its channel")
	assert.Equal(t, http.StatusUnauthorized, f.get(t, storage.MediaProxyPath+attKey, "tenant-2").Code)
}

func TestMediaServe_RevokedBeatsValidSignature(t *testing.T) {
	signer := newTestSigner(time.Hour)
	f := newMediaFixture(t, signer, time.Time{})
	u, _ := signer.SignKey(inboundKey, 0)

	require.Equal(t, http.StatusOK, f.post(t, "/api/v1/media/revoke", "tenant-1", RevokeMediaRequest{Prefix: "inbound/whatsapp/ch-1"}).Code)

	assert.Equal(t, http.StatusForbidden, f.get(t, u, "").Code)
	assert.Equal(t, http.StatusForbidden, f.get(t, storage.MediaProxyPath+inboundKey, "tenant-1").Code)
	assert.Equal(t, http.StatusOK, f.get(t, storage.MediaProxyPath+attKey, "").Code, "other prefixes unaffected")
}

func TestMediaServe_RevocationWorksWithoutSigner(t *testing.T) {
	f := newMediaFixture(t, nil, time.Time{})
	require.Equal(t, http.StatusOK, f.post(t, "/api/v1/media/revoke", "tenant-1", RevokeMediaRequest{Key: attKey}).Code)
	assert.Equal(t, http.StatusForbidden, f.get(t, storage.MediaProxyPath+attKey, "").Code)
}

func TestMediaRevoke_OnlyOwnScope(t *testing.T) {
	f := newMediaFixture(t, newTestSigner(time.Hour), time.Time{})
	cases := []RevokeMediaRequest{
		{Key: attKey, Prefix: "attachments/tenant-1/"}, // both
		{},
	}
	for _, c := range cases {
		assert.Equal(t, http.StatusBadRequest, f.post(t, "/api/v1/media/revoke", "tenant-1", c).Code)
	}
	for _, c := range []RevokeMediaRequest{
		{Key: attKey},                              // someone else's key
		{Prefix: "attachments/"},                   // wider than a tenant
		{Prefix: "inbound/whatsapp/"},              // wider than a channel
		{Prefix: "inbound/whatsapp/ch-1/"},         // someone else's channel
		{Key: "attachments/tenant-2/../tenant-1/"}, // traversal
	} {
		assert.Equal(t, http.StatusNotFound, f.post(t, "/api/v1/media/revoke", "tenant-2", c).Code, "%+v", c)
	}
	assert.Empty(t, f.revoked.set)
}

func TestMediaSign_IssuesURLsForOwnKeysOnly(t *testing.T) {
	signer := newTestSigner(time.Hour)
	f := newMediaFixture(t, signer, time.Now().Add(-time.Minute))

	w := f.post(t, "/api/v1/media/sign", "tenant-1", SignMediaRequest{
		Keys: []string{attKey, "attachments/tenant-2/x.png"},
		URLs: []string{"https://api.linktor.dev" + storage.MediaProxyPath + inboundKey, "https://cdn.other/x.png"},
	})
	require.Equal(t, http.StatusOK, w.Code)
	var resp struct {
		Data []SignedMediaURL `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	require.Len(t, resp.Data, 4)

	assert.Empty(t, resp.Data[0].Error)
	assert.NotNil(t, resp.Data[0].ExpiresAt)
	assert.Equal(t, "not found", resp.Data[1].Error)
	assert.Empty(t, resp.Data[1].URL)
	assert.Empty(t, resp.Data[2].Error)
	assert.Equal(t, inboundKey, resp.Data[2].Key)
	assert.NotEmpty(t, resp.Data[3].Error)

	// The issued URLs actually load, with no header, after the transition.
	assert.Equal(t, http.StatusOK, f.get(t, resp.Data[0].URL, "").Code)
	assert.Equal(t, http.StatusOK, f.get(t, resp.Data[2].URL, "").Code)
}

func TestMediaSign_TTLCappedAtSignerDefault(t *testing.T) {
	f := newMediaFixture(t, newTestSigner(time.Hour), time.Time{})
	w := f.post(t, "/api/v1/media/sign", "tenant-1", SignMediaRequest{Keys: []string{attKey}, TTLSeconds: 365 * 86400})
	var resp struct {
		Data []SignedMediaURL `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	require.NotNil(t, resp.Data[0].ExpiresAt)
	assert.WithinDuration(t, time.Now().Add(time.Hour), *resp.Data[0].ExpiresAt, 5*time.Second)
}

func TestMediaSign_RevokedKeyIsRefused(t *testing.T) {
	f := newMediaFixture(t, newTestSigner(time.Hour), time.Time{})
	f.revoked.set[attKey] = true
	w := f.post(t, "/api/v1/media/sign", "tenant-1", SignMediaRequest{Keys: []string{attKey}})
	assert.Contains(t, w.Body.String(), `"error":"revoked"`)
}

func TestMediaSign_WithoutSignerReturnsDurableURL(t *testing.T) {
	f := newMediaFixture(t, nil, time.Time{})
	w := f.post(t, "/api/v1/media/sign", "tenant-1", SignMediaRequest{Keys: []string{attKey}})
	var resp struct {
		Data []SignedMediaURL `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, "https://api.test"+storage.MediaProxyPath+attKey, resp.Data[0].URL)
	assert.Nil(t, resp.Data[0].ExpiresAt)
}

func TestMediaSign_BatchLimits(t *testing.T) {
	f := newMediaFixture(t, newTestSigner(time.Hour), time.Time{})
	assert.Equal(t, http.StatusBadRequest, f.post(t, "/api/v1/media/sign", "tenant-1", SignMediaRequest{}).Code)
	many := make([]string, maxMediaSignBatch+1)
	for i := range many {
		many[i] = attKey
	}
	assert.Equal(t, http.StatusBadRequest, f.post(t, "/api/v1/media/sign", "tenant-1", SignMediaRequest{Keys: many}).Code)
}

func TestWithSignedMedia_SignsCopyNotStoredMessage(t *testing.T) {
	SetMediaURLSigner(newTestSigner(time.Hour))
	defer SetMediaURLSigner(nil)

	stored := "https://api.test" + storage.MediaProxyPath + attKey
	msg := &entity.Message{
		ID:          "m1",
		Attachments: []*entity.MessageAttachment{{URL: stored, ThumbnailURL: "https://cdn.other/t.png"}},
		Metadata:    map[string]string{"media_url": stored},
	}
	out := withSignedMedia(msg)

	assert.Contains(t, out.Attachments[0].URL, "sig=")
	assert.Equal(t, "https://cdn.other/t.png", out.Attachments[0].ThumbnailURL)
	assert.Contains(t, out.Metadata["media_url"], "sig=")
	assert.Equal(t, stored, msg.Attachments[0].URL, "stored reference untouched")
	assert.Equal(t, stored, msg.Metadata["media_url"])
}
