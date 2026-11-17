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

// grafoDeNumero responde a leitura do número do WhatsApp e anota o token.
func grafoDeNumero(t *testing.T, tokenVisto *string) {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		*tokenVisto = strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
		w.Header().Set("Content-Type", "application/json")
		if *tokenVisto == "tok-ruim" {
			w.WriteHeader(http.StatusUnauthorized)
			_, _ = w.Write([]byte(`{"error":{"message":"Authentication Error","code":190}}`))
			return
		}
		_, _ = w.Write([]byte(`{"id":"PHONE","display_phone_number":"+1 555 141 9823","verified_name":"Test Number"}`))
	}))
	t.Cleanup(srv.Close)
	t.Setenv(graphapi.EnvVar, srv.URL)
}

func corpoDeTesteWhatsApp(token string) []byte {
	corpo, _ := json.Marshal(map[string]interface{}{
		"channel_id":  "ch-wa",
		"type":        "whatsapp_official",
		"config":      map[string]string{"phone_number_id": "PHONE", "api_version": "v23.0"},
		"credentials": map[string]string{"access_token": token, "webhook_secret": ""},
	})
	return corpo
}

// Ao editar o token fica em branco: o teste usa o guardado e lê o número na
// Meta. Antes respondia "configuration accepted" para qualquer texto.
func TestTestWhatsAppConnection_CompletaTokenGuardado(t *testing.T) {
	var tokenVisto string
	grafoDeNumero(t, &tokenVisto)

	handler, repo, _ := setupChannelHandler()
	canal := seedChannel(repo, "ch-wa", "tenant-1", "WhatsApp", entity.ChannelTypeWhatsAppOfficial)
	canal.Config = map[string]string{"phone_number_id": "PHONE"}
	canal.Credentials = map[string]string{"access_token": "tok-guardado"}

	c, w := newChannelAuthContext("POST", "/api/v1/channels/test-whatsapp", corpoDeTesteWhatsApp(""))
	handler.TestWhatsAppConnection(c)

	if w.Code != 200 || tokenVisto != "tok-guardado" || !strings.Contains(w.Body.String(), "555 141 9823") {
		t.Fatalf("código %d, token %q: %s", w.Code, tokenVisto, w.Body.String())
	}
}

func TestTestWhatsAppConnection_TokenInvalidoMostraCodigoDaMeta(t *testing.T) {
	var tokenVisto string
	grafoDeNumero(t, &tokenVisto)

	handler, repo, _ := setupChannelHandler()
	seedChannel(repo, "ch-wa", "tenant-1", "WhatsApp", entity.ChannelTypeWhatsAppOfficial)

	c, w := newChannelAuthContext("POST", "/api/v1/channels/test-whatsapp", corpoDeTesteWhatsApp("tok-ruim"))
	handler.TestWhatsAppConnection(c)

	if w.Code != 400 || !strings.Contains(w.Body.String(), "code 190") {
		t.Fatalf("código %d: %s", w.Code, w.Body.String())
	}
}
