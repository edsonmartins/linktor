package facebook

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/msgfy/linktor/internal/adapters/meta"
	"github.com/msgfy/linktor/pkg/graphapi"
)

func TestConfigFromMap(t *testing.T) {
	cfg := ConfigFromMap(
		map[string]string{"page_id": "P1", "app_id": "A"},
		map[string]string{"access_token": "tok-oauth", "page_id": "P2"},
	)
	assert.Equal(t, "P2", cfg.PageID, "o último mapa vence")
	assert.Equal(t, "A", cfg.AppID)
	assert.Equal(t, "tok-oauth", cfg.PageAccessToken, "canal de OAuth guarda o token em access_token")

	cfg = ConfigFromMap(map[string]string{"page_access_token": "tok-pagina", "access_token": "tok-oauth"})
	assert.Equal(t, "tok-pagina", cfg.PageAccessToken, "page_access_token tem preferência")
}

// grafoFalso responde a leitura da Página e a assinatura do webhook, guardando
// o que recebeu.
type grafoFalso struct {
	srv         *httptest.Server
	assinados   string
	assinaturas int
	tokenVisto  string
}

func novoGrafoFalso(t *testing.T) *grafoFalso {
	t.Helper()
	g := &grafoFalso{}
	g.srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		g.tokenVisto = r.URL.Query().Get("access_token")
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/v22.0/PAGE/subscribed_apps":
			g.assinaturas++
			g.assinados = r.URL.Query().Get("subscribed_fields")
			_, _ = w.Write([]byte(`{"success":true}`))
		case r.Method == http.MethodGet && r.URL.Path == "/v22.0/PAGE":
			_, _ = w.Write([]byte(`{"id":"PAGE","name":"Schematiq"}`))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	t.Cleanup(g.srv.Close)
	t.Setenv(graphapi.EnvVar, g.srv.URL)
	return g
}

func TestConectarAssinaSoOsCamposConsumidos(t *testing.T) {
	g := novoGrafoFalso(t)

	nome, err := Conectar(context.Background(), &FacebookConfig{PageID: "PAGE", PageAccessToken: "tok"})

	require.NoError(t, err)
	assert.Equal(t, "Schematiq", nome)
	assert.Equal(t, "tok", g.tokenVisto)
	assert.Equal(t, 1, g.assinaturas)
	assert.Equal(t, "messages,messaging_postbacks,message_deliveries,message_reads", g.assinados,
		"um campo recusado pela Meta derruba a assinatura inteira; só vai o que o handler lê")
}

func TestVerificarNaoAssina(t *testing.T) {
	g := novoGrafoFalso(t)

	nome, err := Verificar(context.Background(), &FacebookConfig{PageID: "PAGE", PageAccessToken: "tok"})

	require.NoError(t, err)
	assert.Equal(t, "Schematiq", nome)
	assert.Equal(t, 0, g.assinaturas, "Testar Conexão não pode ter efeito colateral")
}

func TestVerificarDevolveErroDaMeta(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]any{"error": map[string]any{"message": "Invalid OAuth access token", "code": 190}})
	}))
	defer srv.Close()
	t.Setenv(graphapi.EnvVar, srv.URL)

	_, err := Verificar(context.Background(), &FacebookConfig{PageID: "PAGE", PageAccessToken: "ruim"})

	var apiErr *meta.APIError
	require.ErrorAs(t, err, &apiErr)
	assert.Equal(t, http.StatusBadRequest, apiErr.StatusCode)
	assert.Contains(t, err.Error(), "(#190)")
}

func TestConectarSemPageIDFalhaAntesDaRede(t *testing.T) {
	g := novoGrafoFalso(t)

	_, err := Conectar(context.Background(), &FacebookConfig{PageAccessToken: "tok"})

	assert.ErrorIs(t, err, ErrMissingPageID)
	assert.Empty(t, g.tokenVisto, "não pode nem tentar a rede")
}
