package main

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/msgfy/linktor/internal/domain/entity"
	"github.com/msgfy/linktor/internal/infrastructure/nats"
	"github.com/msgfy/linktor/internal/infrastructure/storage"
	"github.com/msgfy/linktor/pkg/testutil"
	"github.com/stretchr/testify/require"
)

type rehostStore struct{ objects map[string][]byte }

func (m *rehostStore) Upload(_ context.Context, key string, data []byte, _ string) (string, error) {
	m.objects[key] = data
	return "https://api.test" + storage.MediaProxyPath + key, nil
}
func (m *rehostStore) Delete(_ context.Context, key string) error { delete(m.objects, key); return nil }
func (m *rehostStore) GetURL(_ context.Context, key string) (string, error) {
	return "https://api.test" + storage.MediaProxyPath + key, nil
}
func (m *rehostStore) Open(_ context.Context, key string) (io.ReadCloser, string, int64, error) {
	data, ok := m.objects[key]
	if !ok {
		return nil, "", 0, errors.New("not found")
	}
	return io.NopCloser(bytes.NewReader(data)), "", int64(len(data)), nil
}

// A mídia do Direto fica atrás do proxy do runtime: o Linktor baixa com o
// api_token do canal, só no host da send_url, e guarda no media store.
func TestRehostInboundMedia_DiretoAuthenticated(t *testing.T) {
	var gotAuth []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = append(gotAuth, r.Header.Get("Authorization"))
		if r.Header.Get("Authorization") != "Bearer tok-1" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		w.Header().Set("Content-Type", "audio/mp4")
		_, _ = w.Write([]byte("m4a-bytes"))
	}))
	defer srv.Close()

	channels := testutil.NewMockChannelRepository()
	channels.Channels["ch-d"] = &entity.Channel{
		ID: "ch-d", Type: entity.ChannelTypeDireto,
		Config:      map[string]string{"send_url": srv.URL, "instance_id": "inst"},
		Credentials: map[string]string{"api_token": "tok-1"},
	}
	store := &rehostStore{objects: map[string][]byte{}}
	msg := &nats.InboundMessage{
		ChannelID: "ch-d", ChannelType: "direto",
		Attachments: []nats.AttachmentData{{Type: "audio", URL: srv.URL + "/channel/v1/inst/media/a.mp4", Filename: "audio.m4a", MimeType: "audio/mp4"}},
	}
	rehostInboundMedia(context.Background(), store, channels, msg)

	require.Equal(t, []string{"Bearer tok-1"}, gotAuth)
	require.Contains(t, msg.Attachments[0].URL, storage.MediaProxyPath)
	require.Equal(t, int64(len("m4a-bytes")), msg.Attachments[0].SizeBytes)
	require.Len(t, store.objects, 1)
}

// O token nunca vai para um host que não é o da send_url: a URL veio do
// payload e poderia apontar para qualquer lugar.
func TestRehostInboundMedia_DiretoOtherHostGetsNoToken(t *testing.T) {
	var gotAuth string
	other := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer other.Close()

	channels := testutil.NewMockChannelRepository()
	channels.Channels["ch-d"] = &entity.Channel{
		ID: "ch-d", Type: entity.ChannelTypeDireto,
		Config:      map[string]string{"send_url": "https://direto.example"},
		Credentials: map[string]string{"api_token": "tok-1"},
	}
	store := &rehostStore{objects: map[string][]byte{}}
	original := other.URL + "/channel/v1/inst/media/a.png"
	msg := &nats.InboundMessage{ChannelID: "ch-d", ChannelType: "direto",
		Attachments: []nats.AttachmentData{{Type: "image", URL: original}}}
	rehostInboundMedia(context.Background(), store, channels, msg)

	require.Empty(t, gotAuth)
	require.Equal(t, original, msg.Attachments[0].URL, "falha mantém a URL original")
	require.Empty(t, store.objects)
}

func TestSameHost(t *testing.T) {
	require.True(t, sameHost("https://direto.vendax.ai/channel/v1/i/media/x.png", "https://direto.vendax.ai"))
	require.True(t, sameHost("https://DIRETO.vendax.ai/x", "https://direto.vendax.ai/"))
	require.False(t, sameHost("http://direto.vendax.ai/x", "https://direto.vendax.ai"))
	require.False(t, sameHost("https://evil.example/x", "https://direto.vendax.ai"))
	require.False(t, sameHost("/v0/file/s/x.png", "https://direto.vendax.ai"))
	require.False(t, sameHost("https://direto.vendax.ai/x", ""))
}
