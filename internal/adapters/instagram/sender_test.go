package instagram

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

func servidorQueGrava(t *testing.T, corpos *[]map[string]any) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw, _ := io.ReadAll(r.Body)
		var corpo map[string]any
		_ = json.Unmarshal(raw, &corpo)
		*corpos = append(*corpos, corpo)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"message_id":"mid.` + string(rune('0'+len(*corpos))) + `","recipient_id":"IGSID"}`))
	}))
}

func servidorQueRecusa(t *testing.T, status, code int, mensagem string) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		_ = json.NewEncoder(w).Encode(map[string]any{"error": map[string]any{"message": mensagem, "type": "OAuthException", "code": code}})
	}))
}

func TestClassificarErro(t *testing.T) {
	casos := []struct {
		nome       string
		err        error
		permanente bool
	}{
		{"token inválido (#190)", &meta.APIError{StatusCode: 400, Code: 190}, true},
		{"fora da janela (#10)", &meta.APIError{StatusCode: 400, Code: 10, ErrorSubcode: 2534022}, true},
		{"limite de taxa (#4)", &meta.APIError{StatusCode: 400, Code: 4}, false},
		{"5xx", &meta.APIError{StatusCode: 500, Code: 2}, false},
		{"rede", errors.New("request failed"), false},
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
	s := &igSender{client: newTestClient(t, srv.URL)}

	_, err := s.Send(context.Background(), &outbound.Message{To: "IGSID", Content: outbound.Text{Body: "oi"}})

	require.Error(t, err)
	assert.True(t, outbound.IsPermanent(err), "%v", err)
	assert.Contains(t, err.Error(), "(#190)")
}

func TestSendMediaEnviaLegendaAntesDoAnexo(t *testing.T) {
	var corpos []map[string]any
	srv := servidorQueGrava(t, &corpos)
	defer srv.Close()
	s := &igSender{client: newTestClient(t, srv.URL)}

	rec, err := s.Send(context.Background(), &outbound.Message{
		To:      "IGSID",
		Content: outbound.Media{Type: outbound.MediaImage, URL: "https://cdn.example/foto.png", Caption: "segue a foto"},
	})
	require.NoError(t, err)
	require.Len(t, corpos, 2)
	assert.Equal(t, "segue a foto", corpos[0]["message"].(map[string]any)["text"])
	anexo := corpos[1]["message"].(map[string]any)["attachment"].(map[string]any)
	assert.Equal(t, "image", anexo["type"])
	assert.Equal(t, "mid.2", rec.ProviderMessageID, "o recibo é o do anexo")

	corpos = nil
	_, err = s.Send(context.Background(), &outbound.Message{To: "IGSID", Content: outbound.Media{Type: outbound.MediaDocument, URL: "https://cdn.example/a.pdf"}})
	require.NoError(t, err)
	assert.Len(t, corpos, 1, "sem legenda, um envio só")
}
