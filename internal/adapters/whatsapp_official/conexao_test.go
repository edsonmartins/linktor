package whatsapp_official

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/msgfy/linktor/internal/outbound"
	"github.com/msgfy/linktor/pkg/graphapi"
)

func TestConfigFromMap(t *testing.T) {
	cfg := ConfigFromMap(
		map[string]string{"phone_number_id": "P1", "business_id": "WABA"},
		map[string]string{"access_token": "tok", "phone_number_id": "P2"},
	)
	assert.Equal(t, "P2", cfg.PhoneNumberID, "o último mapa vence")
	assert.Equal(t, "WABA", cfg.BusinessID)
	assert.Equal(t, "tok", cfg.AccessToken)
	assert.Equal(t, DefaultAPIVersion, cfg.APIVersion)

	cfg = ConfigFromMap(map[string]string{"waba_id": "WABA-ES"})
	assert.Equal(t, "WABA-ES", cfg.BusinessID, "canal de Embedded Signup guarda a conta em waba_id")
	assert.Error(t, cfg.Validate())
}

// A Meta entrega o wa_id de muitos celulares do Brasil sem o nono dígito;
// responder para essa forma era recusado com #131030.
func TestDestinoBR(t *testing.T) {
	casos := map[string]string{
		"554497239232":   "5544997239232",  // celular sem o 9
		"+554497239232":  "+5544997239232", // com +
		"5544997239232":  "5544997239232",  // já com o 9
		"554432221234":   "554432221234",   // fixo: começa em 3
		"16315551181":    "16315551181",    // outro país
		"5511":           "5511",           // curto demais
		"351912345678":   "351912345678",   // 12 dígitos, não é 55
		"+5511987654321": "+5511987654321", // 13 dígitos intacto
	}
	for entrada, esperado := range casos {
		assert.Equal(t, esperado, destinoBR(entrada), entrada)
	}
}

// grafoFalso responde a leitura do número e a ligação do app à conta.
type grafoFalso struct {
	assinaturas int
	caminhos    []string
	autorizacao string
}

func novoGrafoFalso(t *testing.T) *grafoFalso {
	t.Helper()
	g := &grafoFalso{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		g.caminhos = append(g.caminhos, r.Method+" "+r.URL.Path)
		g.autorizacao = r.Header.Get("Authorization")
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/v23.0/WABA/subscribed_apps":
			g.assinaturas++
			_, _ = w.Write([]byte(`{"success":true}`))
		case r.Method == http.MethodGet && r.URL.Path == "/v23.0/PHONE":
			_, _ = w.Write([]byte(`{"id":"PHONE","display_phone_number":"+1 555 141 9823","verified_name":"Test Number"}`))
		default:
			w.WriteHeader(http.StatusNotFound)
			_, _ = w.Write([]byte(`{"error":{"message":"not found","code":100}}`))
		}
	}))
	t.Cleanup(srv.Close)
	t.Setenv(graphapi.EnvVar, srv.URL)
	return g
}

func TestConectarLigaOAppAConta(t *testing.T) {
	g := novoGrafoFalso(t)

	info, err := Conectar(context.Background(), &Config{AccessToken: "tok", PhoneNumberID: "PHONE", BusinessID: "WABA", APIVersion: "v23.0"})

	require.NoError(t, err)
	assert.Equal(t, "+1 555 141 9823", info.DisplayPhoneNumber)
	assert.Equal(t, "Bearer tok", g.autorizacao)
	assert.Equal(t, 1, g.assinaturas, "sem a ligação a Meta não entrega nenhum evento real")
}

func TestConectarSemContaSoVerifica(t *testing.T) {
	g := novoGrafoFalso(t)

	_, err := Conectar(context.Background(), &Config{AccessToken: "tok", PhoneNumberID: "PHONE", APIVersion: "v23.0"})

	require.NoError(t, err)
	assert.Equal(t, 0, g.assinaturas)
}

func TestVerificarNaoAssina(t *testing.T) {
	g := novoGrafoFalso(t)

	info, err := Verificar(context.Background(), &Config{AccessToken: "tok", PhoneNumberID: "PHONE", BusinessID: "WABA", APIVersion: "v23.0"})

	require.NoError(t, err)
	assert.Equal(t, "Test Number", info.VerifiedName)
	assert.Equal(t, 0, g.assinaturas, "Testar Conexão não pode ter efeito colateral")
}

func TestVerificarDevolveErroDaMeta(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"error":{"message":"Authentication Error","type":"OAuthException","code":190}}`))
	}))
	defer srv.Close()
	t.Setenv(graphapi.EnvVar, srv.URL)

	_, err := Verificar(context.Background(), &Config{AccessToken: "ruim", PhoneNumberID: "PHONE", APIVersion: "v23.0"})

	var apiErr *APIRequestError
	require.ErrorAs(t, err, &apiErr)
	assert.Equal(t, 190, apiErr.APIError.Code)

	_, err = Verificar(context.Background(), &Config{PhoneNumberID: "PHONE"})
	assert.ErrorContains(t, err, "access_token is required", "sem token nem tenta a rede")
}

// O sender responde para o número com o nono dígito, mesmo que o contato tenha
// sido gravado sem ele.
func TestSendNormalizaCelularBrasileiro(t *testing.T) {
	var corpo map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(raw, &corpo)
		_, _ = w.Write([]byte(`{"messaging_product":"whatsapp","messages":[{"id":"wamid.1"}]}`))
	}))
	defer srv.Close()
	t.Setenv(graphapi.EnvVar, srv.URL)

	s, err := NewSenderFactory().New(map[string]string{"access_token": "tok", "phone_number_id": "PHONE"})
	require.NoError(t, err)
	msg := &outbound.Message{To: "554497239232", Content: outbound.Text{Body: "oi"}}

	rec, err := s.Send(context.Background(), msg)

	require.NoError(t, err)
	assert.Equal(t, "wamid.1", rec.ProviderMessageID)
	assert.Equal(t, "5544997239232", corpo["to"])
	assert.Equal(t, "554497239232", msg.To, "a mensagem original não é alterada")
}
