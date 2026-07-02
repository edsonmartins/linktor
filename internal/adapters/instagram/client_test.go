package instagram

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

func newTestClient(t *testing.T, srvURL string) *Client {
	t.Helper()
	client, err := NewClient(&InstagramConfig{InstagramID: "IGID", AccessToken: "tok"})
	require.NoError(t, err)
	client.api.SetBaseURL(srvURL)
	return client
}

func captureServer(t *testing.T, body *[]byte) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		*body, _ = io.ReadAll(r.Body)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"message_id":"mid.1","recipient_id":"IGSID"}`))
	}))
}

func TestSendTextMessageSetsMessagingTypeResponse(t *testing.T) {
	var body []byte
	srv := captureServer(t, &body)
	defer srv.Close()

	_, err := newTestClient(t, srv.URL).SendTextMessage(context.Background(), "IGSID", "hi")
	require.NoError(t, err)

	var sent map[string]any
	require.NoError(t, json.Unmarshal(body, &sent))
	require.Equal(t, "RESPONSE", sent["messaging_type"], "IG text replies must use messaging_type RESPONSE")
}

func TestSendAttachmentSetsMessagingTypeResponse(t *testing.T) {
	var body []byte
	srv := captureServer(t, &body)
	defer srv.Close()

	_, err := newTestClient(t, srv.URL).SendImage(context.Background(), "IGSID", "https://cdn/x.png")
	require.NoError(t, err)

	var sent map[string]any
	require.NoError(t, json.Unmarshal(body, &sent))
	require.Equal(t, "RESPONSE", sent["messaging_type"], "IG media replies must use messaging_type RESPONSE")
}
