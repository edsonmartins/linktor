package facebook

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/msgfy/linktor/internal/adapters/meta"
	"github.com/msgfy/linktor/internal/outbound"
)

// servidorQueGrava guarda o corpo de cada requisição, na ordem, e responde um
// envio aceito.
func servidorQueGrava(t *testing.T, corpos *[]map[string]any) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw, _ := io.ReadAll(r.Body)
		var corpo map[string]any
		_ = json.Unmarshal(raw, &corpo)
		*corpos = append(*corpos, corpo)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"message_id":"mid.` + string(rune('0'+len(*corpos))) + `","recipient_id":"PSID"}`))
	}))
}

// servidorQueRecusa responde sempre o erro da Graph informado.
func servidorQueRecusa(t *testing.T, status, code int, mensagem string) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		_ = json.NewEncoder(w).Encode(map[string]any{"error": map[string]any{
			"message": mensagem, "type": "OAuthException", "code": code,
		}})
	}))
}

// Recusa da Graph que não é limite nem 5xx é permanente: repetir só deixava a
// mensagem "pendente" até a fila desistir.
func TestClassificarErro(t *testing.T) {
	casos := []struct {
		nome       string
		err        error
		permanente bool
	}{
		{"token inválido (#190)", &meta.APIError{StatusCode: 400, Code: 190, Message: "Invalid OAuth access token"}, true},
		{"parâmetro inválido (#100)", &meta.APIError{StatusCode: 400, Code: 100, Message: "No matching user found"}, true},
		{"fora da janela (#10)", &meta.APIError{StatusCode: 400, Code: 10, ErrorSubcode: 2018278}, true},
		{"limite de taxa (#4)", &meta.APIError{StatusCode: 400, Code: 4}, false},
		{"429", &meta.APIError{StatusCode: 429, Code: 613}, false},
		{"5xx", &meta.APIError{StatusCode: 500, Code: 2}, false},
		{"rede", errors.New("request failed: dial tcp"), false},
		{"já permanente", outbound.Permanentf("media requires a URL"), true},
	}
	for _, c := range casos {
		t.Run(c.nome, func(t *testing.T) {
			assert.Equal(t, c.permanente, outbound.IsPermanent(classificarErro(c.err)))
		})
	}
}

func TestSendMarcaErroDaMetaComoPermanente(t *testing.T) {
	srv := servidorQueRecusa(t, http.StatusBadRequest, 190, "Invalid OAuth access token")
	defer srv.Close()
	s := &fbSender{client: newTestClient(t, srv.URL)}

	_, err := s.Send(context.Background(), &outbound.Message{To: "PSID", Content: outbound.Text{Body: "oi"}})

	require.Error(t, err)
	assert.True(t, outbound.IsPermanent(err), "%v", err)
	assert.Contains(t, err.Error(), "(#190)", "o código da Meta tem de chegar à linha do tempo")
}

func TestSendMantemLimiteDeTaxaComoTransitorio(t *testing.T) {
	srv := servidorQueRecusa(t, http.StatusTooManyRequests, 4, "Application request limit reached")
	defer srv.Close()
	s := &fbSender{client: newTestClient(t, srv.URL)}

	_, err := s.Send(context.Background(), &outbound.Message{To: "PSID", Content: outbound.Text{Body: "oi"}})

	require.Error(t, err)
	assert.False(t, outbound.IsPermanent(err))
}

// O anexo do Messenger não carrega texto: a legenda vai antes, como mensagem
// própria, com o mesmo messaging_type/tag, e o recibo é o do anexo.
func TestSendMediaEnviaLegendaAntesDoAnexo(t *testing.T) {
	var corpos []map[string]any
	srv := servidorQueGrava(t, &corpos)
	defer srv.Close()
	s := &fbSender{client: newTestClient(t, srv.URL)}

	rec, err := s.Send(context.Background(), &outbound.Message{
		To:       "PSID",
		Content:  outbound.Media{Type: outbound.MediaImage, URL: "https://cdn.example/foto.png", Caption: "segue a foto"},
		Metadata: map[string]string{"messaging_type": "MESSAGE_TAG", "message_tag": "ACCOUNT_UPDATE"},
	})
	require.NoError(t, err)
	require.Len(t, corpos, 2)

	texto := corpos[0]["message"].(map[string]any)
	assert.Equal(t, "segue a foto", texto["text"])
	anexo := corpos[1]["message"].(map[string]any)["attachment"].(map[string]any)
	assert.Equal(t, "image", anexo["type"])
	assert.Equal(t, "https://cdn.example/foto.png", anexo["payload"].(map[string]any)["url"])
	for _, c := range corpos {
		assert.Equal(t, "MESSAGE_TAG", c["messaging_type"])
		assert.Equal(t, "ACCOUNT_UPDATE", c["tag"])
	}
	assert.Equal(t, "mid.2", rec.ProviderMessageID, "o recibo é o do anexo")
}

func TestSendMediaSemLegendaFazUmEnvioSo(t *testing.T) {
	var corpos []map[string]any
	srv := servidorQueGrava(t, &corpos)
	defer srv.Close()
	s := &fbSender{client: newTestClient(t, srv.URL)}

	_, err := s.Send(context.Background(), &outbound.Message{
		To: "PSID", Content: outbound.Media{Type: outbound.MediaDocument, URL: "https://cdn.example/a.pdf"},
	})
	require.NoError(t, err)
	assert.Len(t, corpos, 1)
}
