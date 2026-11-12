package instagram

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
		map[string]string{"instagram_id": "IG1", "app_id": "A"},
		map[string]string{"access_token": "EAAxyz", "instagram_id": "IG2", "app_secret": "s"},
	)
	assert.Equal(t, "IG2", cfg.InstagramID, "o último mapa vence")
	assert.Equal(t, "A", cfg.AppID)
	assert.Equal(t, "EAAxyz", cfg.AccessToken)
	assert.Equal(t, "s", cfg.AppSecret)
	assert.Equal(t, "EAAxyz", cfg.GetEffectiveAccessToken())

	cfg = ConfigFromMap(map[string]string{"instagram_id": "IG", "page_access_token": "EAApagina"})
	assert.Equal(t, "EAApagina", cfg.GetEffectiveAccessToken(), "page_access_token vale quando access_token está vazio")
}

// O host é decidido pelo token, não pela chave: apps novos só emitem "EAA…",
// que graph.instagram.com recusa com #190.
func TestHostPeloPrefixoDoToken(t *testing.T) {
	ig, err := NewClient(&InstagramConfig{InstagramID: "IG", AccessToken: "IGAAabc"})
	require.NoError(t, err)
	assert.Equal(t, meta.InstagramAPIBaseURL, ig.api.BaseURL())

	fb, err := NewClient(&InstagramConfig{InstagramID: "IG", AccessToken: "EAAabc"})
	require.NoError(t, err)
	assert.Equal(t, graphapi.BaseURL(), fb.api.BaseURL())

	pagina, err := NewClient(&InstagramConfig{InstagramID: "IG", PageAccessToken: "EAApagina"})
	require.NoError(t, err)
	assert.Equal(t, graphapi.BaseURL(), pagina.api.BaseURL())
}

// grafoFalso responde a leitura da conta e a assinatura do webhook.
type grafoFalso struct {
	assinados   string
	assinaturas int
	tokenVisto  string
	campos      string
}

func novoGrafoFalso(t *testing.T) *grafoFalso {
	t.Helper()
	g := &grafoFalso{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		g.tokenVisto = r.URL.Query().Get("access_token")
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/v22.0/IG/subscribed_apps":
			g.assinaturas++
			g.assinados = r.URL.Query().Get("subscribed_fields")
			_, _ = w.Write([]byte(`{"success":true}`))
		case r.Method == http.MethodGet && r.URL.Path == "/v22.0/IG":
			g.campos = r.URL.Query().Get("fields")
			_, _ = w.Write([]byte(`{"id":"IG","username":"aleak1ra","name":"Alexandre"}`))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	t.Cleanup(srv.Close)
	t.Setenv(graphapi.EnvVar, srv.URL)
	return g
}

func TestConectarAssinaSoOsCamposConsumidos(t *testing.T) {
	g := novoGrafoFalso(t)

	usuario, err := Conectar(context.Background(), &InstagramConfig{InstagramID: "IG", AccessToken: "EAAtok"})

	require.NoError(t, err)
	assert.Equal(t, "aleak1ra", usuario)
	assert.Equal(t, "EAAtok", g.tokenVisto)
	assert.Equal(t, 1, g.assinaturas)
	assert.Equal(t, "messages,messaging_seen,message_reactions", g.assinados)
}

func TestVerificarNaoAssina(t *testing.T) {
	g := novoGrafoFalso(t)

	usuario, err := Verificar(context.Background(), &InstagramConfig{InstagramID: "IG", AccessToken: "EAAtok"})

	require.NoError(t, err)
	assert.Equal(t, "aleak1ra", usuario)
	assert.Equal(t, 0, g.assinaturas, "Testar Conexão não pode ter efeito colateral")
}

func TestVerificarDevolveErroDaMeta(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]any{"error": map[string]any{"message": "Invalid OAuth access token", "code": 190}})
	}))
	defer srv.Close()
	t.Setenv(graphapi.EnvVar, srv.URL)

	_, err := Verificar(context.Background(), &InstagramConfig{InstagramID: "IG", AccessToken: "EAAruim"})

	var apiErr *meta.APIError
	require.ErrorAs(t, err, &apiErr)
	assert.Equal(t, http.StatusBadRequest, apiErr.StatusCode)
	assert.Contains(t, err.Error(), "(#190)")
}

// O Instagram recusa first_name/last_name: o perfil tem de pedir só o que a
// conta expõe, e sem nome o @ serve.
func TestNomeDoPerfilPedeCamposDoInstagram(t *testing.T) {
	var campos string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		campos = r.URL.Query().Get("fields")
		_, _ = w.Write([]byte(`{"id":"U1","username":"cliente"}`))
	}))
	defer srv.Close()
	t.Setenv(graphapi.EnvVar, srv.URL)
	client, err := NewClient(&InstagramConfig{InstagramID: "IG", AccessToken: "EAAtok"})
	require.NoError(t, err)

	nome, err := client.NomeDoPerfil(context.Background(), "U1")

	require.NoError(t, err)
	assert.Equal(t, "name,username,profile_pic", campos)
	assert.Equal(t, "@cliente", nome)
}
