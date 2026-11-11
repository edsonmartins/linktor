package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/msgfy/linktor/internal/domain/entity"
	"github.com/msgfy/linktor/pkg/graphapi"
)

// grafoDePagina responde a leitura da Página e anota o token recebido.
func grafoDePagina(t *testing.T, tokenVisto *string) {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		*tokenVisto = r.URL.Query().Get("access_token")
		w.Header().Set("Content-Type", "application/json")
		if *tokenVisto == "tok-ruim" {
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte(`{"error":{"message":"Invalid OAuth access token","type":"OAuthException","code":190}}`))
			return
		}
		_, _ = w.Write([]byte(`{"id":"PAGE","name":"Schematiq"}`))
	}))
	t.Cleanup(srv.Close)
	t.Setenv(graphapi.EnvVar, srv.URL)
}

func corpoDeTesteFacebook(token string) []byte {
	corpo, _ := json.Marshal(map[string]interface{}{
		"channel_id":  "ch-fb",
		"type":        "facebook",
		"config":      map[string]string{"app_id": "APP", "page_id": "PAGE"},
		"credentials": map[string]string{"app_secret": "", "page_access_token": token},
	})
	return corpo
}

// Ao editar, o formulário manda o token em branco: o teste tem de usar o
// guardado e falar com a Graph de verdade, devolvendo o nome da Página.
func TestTestFacebookConnection_CompletaTokenGuardado(t *testing.T) {
	var tokenVisto string
	grafoDePagina(t, &tokenVisto)

	handler, repo, _ := setupChannelHandler()
	canal := seedChannel(repo, "ch-fb", "tenant-1", "Messenger", entity.ChannelTypeFacebook)
	canal.Config = map[string]string{"app_id": "APP", "page_id": "PAGE"}
	canal.Credentials = map[string]string{"page_access_token": "tok-guardado"}

	c, w := newChannelAuthContext("POST", "/api/v1/channels/test-facebook", corpoDeTesteFacebook(""))
	handler.TestFacebookConnection(c)

	if w.Code != 200 {
		t.Fatalf("esperado 200, veio %d: %s", w.Code, w.Body.String())
	}
	if tokenVisto != "tok-guardado" {
		t.Fatalf("token usado na Graph = %q", tokenVisto)
	}
	if !strings.Contains(w.Body.String(), "Schematiq") {
		t.Fatalf("resposta sem o nome da Página: %s", w.Body.String())
	}
}

func TestTestFacebookConnection_TokenInvalidoMostraCodigoDaMeta(t *testing.T) {
	var tokenVisto string
	grafoDePagina(t, &tokenVisto)

	handler, repo, _ := setupChannelHandler()
	seedChannel(repo, "ch-fb", "tenant-1", "Messenger", entity.ChannelTypeFacebook)

	c, w := newChannelAuthContext("POST", "/api/v1/channels/test-facebook", corpoDeTesteFacebook("tok-ruim"))
	handler.TestFacebookConnection(c)

	if w.Code != 400 {
		t.Fatalf("esperado 400, veio %d: %s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "(#190)") {
		t.Fatalf("resposta sem o código da Meta: %s", w.Body.String())
	}
}

// Canal criado por OAuth guarda o token em access_token.
func TestTestFacebookConnection_AceitaAccessTokenDoOAuth(t *testing.T) {
	var tokenVisto string
	grafoDePagina(t, &tokenVisto)

	handler, repo, _ := setupChannelHandler()
	canal := seedChannel(repo, "ch-fb", "tenant-1", "Messenger", entity.ChannelTypeFacebook)
	canal.Config = map[string]string{"page_id": "PAGE"}
	canal.Credentials = map[string]string{"access_token": "tok-oauth"}

	c, w := newChannelAuthContext("POST", "/api/v1/channels/test-facebook", corpoDeTesteFacebook(""))
	handler.TestFacebookConnection(c)

	if w.Code != 200 || tokenVisto != "tok-oauth" {
		t.Fatalf("código %d, token %q: %s", w.Code, tokenVisto, w.Body.String())
	}
}
