package telegram

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func fakeBotAPI(t *testing.T, calls *[]string, forms map[string]map[string]string) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasPrefix(r.URL.Path, "/bot123:good/") {
			w.WriteHeader(http.StatusUnauthorized)
			_, _ = w.Write([]byte(`{"ok":false,"error_code":401,"description":"Unauthorized"}`))
			return
		}
		method := strings.TrimPrefix(r.URL.Path, "/bot123:good/")
		*calls = append(*calls, method)
		_ = r.ParseForm()
		forms[method] = map[string]string{"url": r.PostForm.Get("url"), "secret_token": r.PostForm.Get("secret_token")}
		switch method {
		case "getMe":
			_, _ = w.Write([]byte(`{"ok":true,"result":{"id":1,"is_bot":true,"username":"linktor_homolog_bot"}}`))
		default:
			_, _ = w.Write([]byte(`{"ok":true,"result":true}`))
		}
	}))
}

func TestWebhookRegistrar_RegisterValidatesThenSetsWebhookWithSecret(t *testing.T) {
	var calls []string
	forms := map[string]map[string]string{}
	srv := fakeBotAPI(t, &calls, forms)
	defer srv.Close()

	r := &WebhookRegistrar{APIBaseURL: srv.URL}
	username, err := r.Register(context.Background(), "123:good", "s3cr3t", "https://api.test/api/v1/webhooks/telegram/ch-1")
	require.NoError(t, err)

	assert.Equal(t, "linktor_homolog_bot", username)
	assert.Equal(t, []string{"getMe", "setWebhook"}, calls)
	assert.Equal(t, "https://api.test/api/v1/webhooks/telegram/ch-1", forms["setWebhook"]["url"])
	assert.Equal(t, "s3cr3t", forms["setWebhook"]["secret_token"])
}

func TestWebhookRegistrar_BadTokenFailsBeforeSetWebhookWithoutLeakingToken(t *testing.T) {
	var calls []string
	srv := fakeBotAPI(t, &calls, map[string]map[string]string{})
	defer srv.Close()

	r := &WebhookRegistrar{APIBaseURL: srv.URL}
	_, err := r.Register(context.Background(), "999:bad", "s", "https://api.test/x")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "Unauthorized")
	assert.NotContains(t, err.Error(), "999:bad")
	assert.Empty(t, calls, "setWebhook must not run with an invalid token")
}

func TestWebhookRegistrar_UnreachableAPIDoesNotLeakToken(t *testing.T) {
	r := &WebhookRegistrar{APIBaseURL: "http://127.0.0.1:1"}
	_, err := r.Register(context.Background(), "123:secret-token", "s", "https://api.test/x")
	require.Error(t, err)
	assert.NotContains(t, err.Error(), "123:secret-token")
}

func TestWebhookRegistrar_Unregister(t *testing.T) {
	var calls []string
	srv := fakeBotAPI(t, &calls, map[string]map[string]string{})
	defer srv.Close()

	require.NoError(t, (&WebhookRegistrar{APIBaseURL: srv.URL}).Unregister(context.Background(), "123:good"))
	assert.Equal(t, []string{"deleteWebhook"}, calls)
}

func TestNewSecretToken(t *testing.T) {
	a, err := NewSecretToken()
	require.NoError(t, err)
	b, _ := NewSecretToken()
	assert.Len(t, a, 64)
	assert.NotEqual(t, a, b)
}
