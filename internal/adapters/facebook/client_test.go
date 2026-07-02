package facebook

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/msgfy/linktor/internal/outbound"
	"github.com/stretchr/testify/require"
)

// captureServer returns an httptest server that records the last request body
// and replies with a minimal Send API success response.
func captureServer(t *testing.T, body *[]byte) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		*body, _ = io.ReadAll(r.Body)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"message_id":"mid.1","recipient_id":"PSID"}`))
	}))
}

func newTestClient(t *testing.T, srvURL string) *Client {
	t.Helper()
	client, err := NewClient(&FacebookConfig{PageID: "PAGE", PageAccessToken: "tok"})
	require.NoError(t, err)
	client.api.SetBaseURL(srvURL)
	return client
}

func TestSendTextMessageSetsMessagingTypeResponse(t *testing.T) {
	var body []byte
	srv := captureServer(t, &body)
	defer srv.Close()

	_, err := newTestClient(t, srv.URL).SendTextMessage(context.Background(), "PSID", "hi")
	require.NoError(t, err)

	var sent map[string]any
	require.NoError(t, json.Unmarshal(body, &sent))
	require.Equal(t, "RESPONSE", sent["messaging_type"], "text replies must use messaging_type RESPONSE")
}

func TestSendAttachmentSetsMessagingTypeResponse(t *testing.T) {
	var body []byte
	srv := captureServer(t, &body)
	defer srv.Close()

	_, err := newTestClient(t, srv.URL).SendImage(context.Background(), "PSID", "https://cdn/x.png")
	require.NoError(t, err)

	var sent map[string]any
	require.NoError(t, json.Unmarshal(body, &sent))
	require.Equal(t, "RESPONSE", sent["messaging_type"], "media replies must use messaging_type RESPONSE")
}

func TestSenderHonorsMessagingTypeMetadata(t *testing.T) {
	var body []byte
	srv := captureServer(t, &body)
	defer srv.Close()

	s := &fbSender{client: newTestClient(t, srv.URL)}
	_, err := s.Send(context.Background(), &outbound.Message{
		To:       "PSID",
		Content:  outbound.Text{Body: "out of window"},
		Metadata: map[string]string{"messaging_type": "MESSAGE_TAG", "message_tag": "HUMAN_AGENT"},
	})
	require.NoError(t, err)

	var sent map[string]any
	require.NoError(t, json.Unmarshal(body, &sent))
	require.Equal(t, "MESSAGE_TAG", sent["messaging_type"], "metadata must override the default RESPONSE")
	require.Equal(t, "HUMAN_AGENT", sent["tag"], "message tag must be forwarded for out-of-window sends")
}
