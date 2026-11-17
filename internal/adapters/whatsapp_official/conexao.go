package whatsapp_official

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

// ConfigFromMap monta a configuração do canal a partir de um ou mais mapas
// (config, credentials); em chave repetida o último vence. O ID da conta do
// WhatsApp (WABA) fica em business_id no formulário manual e em waba_id nos
// canais de Embedded Signup. Única casa dessas chaves: o sender, o Conectar e
// o Testar Conexão usam todos esta função.
func ConfigFromMap(mapas ...map[string]string) *Config {
	junto := map[string]string{}
	for _, m := range mapas {
		for k, v := range m {
			junto[k] = v
		}
	}
	cfg := &Config{
		AccessToken:   junto["access_token"],
		PhoneNumberID: junto["phone_number_id"],
		BusinessID:    junto["business_id"],
		VerifyToken:   junto["verify_token"],
		WebhookSecret: junto["webhook_secret"],
		APIVersion:    getOrDefault(junto, "api_version", DefaultAPIVersion),
	}
	if cfg.BusinessID == "" {
		cfg.BusinessID = junto["waba_id"]
	}
	return cfg
}

// Validate confere o mínimo para falar com a Cloud API.
func (c *Config) Validate() error {
	if strings.TrimSpace(c.AccessToken) == "" {
		return fmt.Errorf("access_token is required")
	}
	if strings.TrimSpace(c.PhoneNumberID) == "" {
		return fmt.Errorf("phone_number_id is required")
	}
	return nil
}

// Verificar confirma que o token alcança o número e devolve o número exibido
// e o nome verificado. É o que o Testar Conexão usa: só leitura.
func Verificar(ctx context.Context, cfg *Config) (*PhoneNumberInfo, error) {
	_, info, err := verificar(ctx, cfg)
	return info, err
}

// Conectar verifica o token e liga o app à conta do WhatsApp (WABA). Sem essa
// ligação a Meta aceita o webhook mas não entrega nenhum evento real: o canal
// ficava "connected" e mudo. Canal sem o ID da conta só é verificado, porque
// não há o que assinar.
func Conectar(ctx context.Context, cfg *Config) (*PhoneNumberInfo, error) {
	client, info, err := verificar(ctx, cfg)
	if err != nil {
		return nil, err
	}
	if cfg.BusinessID != "" {
		if err := client.SubscribeApp(ctx); err != nil {
			return nil, err
		}
	}
	return info, nil
}

func verificar(ctx context.Context, cfg *Config) (*Client, *PhoneNumberInfo, error) {
	if err := cfg.Validate(); err != nil {
		return nil, nil, err
	}
	client := NewClient(cfg)
	info, err := client.GetPhoneNumberInfo(ctx)
	if err != nil {
		return nil, nil, err
	}
	return client, info, nil
}

// SubscribeApp liga este app à conta do WhatsApp, nos campos que o app assinou
// no painel da Meta.
func (c *Client) SubscribeApp(ctx context.Context) error {
	endpoint := c.buildURL(fmt.Sprintf("/%s/subscribed_apps", c.cfg().BusinessID))
	respBody, err := c.doRequest(ctx, http.MethodPost, endpoint, nil, nil)
	if err != nil {
		return err
	}
	var resp struct {
		Success bool `json:"success"`
	}
	if err := json.Unmarshal(respBody, &resp); err != nil || !resp.Success {
		return fmt.Errorf("failed to subscribe app to the WhatsApp account")
	}
	return nil
}

// destinoBR devolve o celular brasileiro com o nono dígito. A Meta entrega o
// wa_id de muitos números do Brasil sem ele (55 + DDD + 8 dígitos), e responder
// para essa forma é recusado quando o cadastro está com 13 dígitos (#131030).
// Com o 9 a Meta resolve as duas formas. Fixo (começa em 2 a 5) e número de
// outro país passam intactos.
func destinoBR(numero string) string {
	digitos := strings.TrimPrefix(numero, "+")
	if len(digitos) != 12 || !strings.HasPrefix(digitos, "55") {
		return numero
	}
	if primeiro := digitos[4]; primeiro < '6' || primeiro > '9' {
		return numero
	}
	return numero[:len(numero)-8] + "9" + numero[len(numero)-8:]
}
