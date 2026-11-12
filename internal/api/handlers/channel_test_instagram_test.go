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

// grafoDeConta responde a leitura da conta do Instagram e anota o token.
func grafoDeConta(t *testing.T, tokenVisto *string) {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		*tokenVisto = r.URL.Query().Get("access_token")
		w.Header().Set("Content-Type", "application/json")
		if *tokenVisto == "EAAruim" {
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte(`{"error":{"message":"Invalid OAuth access token","code":190}}`))
			return
		}
		_, _ = w.Write([]byte(`{"id":"IG","username":"aleak1ra"}`))
	}))
	t.Cleanup(srv.Close)
	t.Setenv(graphapi.EnvVar, srv.URL)
}

func corpoDeTesteInstagram(token string) []byte {
	corpo, _ := json.Marshal(map[string]interface{}{
		"channel_id":  "ch-ig",
		"type":        "instagram",
		"config":      map[string]string{"instagram_id": "IG", "app_id": "APP"},
		"credentials": map[string]string{"access_token": token, "app_secret": ""},
	})
	return corpo
}

// Ao editar o token fica em branco: o teste usa o guardado e lê a conta na Graph.
func TestTestInstagramConnection_CompletaTokenGuardado(t *testing.T) {
	var tokenVisto string
	grafoDeConta(t, &tokenVisto)

	handler, repo, _ := setupChannelHandler()
	canal := seedChannel(repo, "ch-ig", "tenant-1", "Instagram", entity.ChannelTypeInstagram)
	canal.Config = map[string]string{"instagram_id": "IG"}
	canal.Credentials = map[string]string{"access_token": "EAAguardado"}

	c, w := newChannelAuthContext("POST", "/api/v1/channels/test-instagram", corpoDeTesteInstagram(""))
	handler.TestInstagramConnection(c)

	if w.Code != 200 || tokenVisto != "EAAguardado" || !strings.Contains(w.Body.String(), "@aleak1ra") {
		t.Fatalf("código %d, token %q: %s", w.Code, tokenVisto, w.Body.String())
	}
}

func TestTestInstagramConnection_TokenInvalidoMostraCodigoDaMeta(t *testing.T) {
	var tokenVisto string
	grafoDeConta(t, &tokenVisto)

	handler, repo, _ := setupChannelHandler()
	seedChannel(repo, "ch-ig", "tenant-1", "Instagram", entity.ChannelTypeInstagram)

	c, w := newChannelAuthContext("POST", "/api/v1/channels/test-instagram", corpoDeTesteInstagram("EAAruim"))
	handler.TestInstagramConnection(c)

	if w.Code != 400 || !strings.Contains(w.Body.String(), "(#190)") {
		t.Fatalf("código %d: %s", w.Code, w.Body.String())
	}
}
