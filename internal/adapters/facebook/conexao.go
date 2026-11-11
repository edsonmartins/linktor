package facebook

import "context"

// ConfigFromMap monta a configuração do canal a partir de um ou mais mapas
// (config, credentials); em chave repetida o último vence. Canais criados por
// /oauth/channels guardam o token da Página em access_token, aceito como
// alternativa a page_access_token. Esta é a única casa dessa regra: o sender,
// o Conectar, o Testar Conexão e o webhook usam todos esta função.
func ConfigFromMap(mapas ...map[string]string) *FacebookConfig {
	junto := map[string]string{}
	for _, m := range mapas {
		for k, v := range m {
			junto[k] = v
		}
	}
	cfg := &FacebookConfig{
		AppID:           junto["app_id"],
		AppSecret:       junto["app_secret"],
		PageID:          junto["page_id"],
		PageAccessToken: junto["page_access_token"],
		UserAccessToken: junto["user_access_token"],
		VerifyToken:     junto["verify_token"],
		InstagramID:     junto["instagram_id"],
	}
	if cfg.PageAccessToken == "" {
		cfg.PageAccessToken = junto["access_token"]
	}
	return cfg
}

// camposDoWebhook são os eventos que o handler consome. A Meta recusa a
// assinatura inteira se um campo não for aceito, então optins, optouts,
// handovers e standby, que ninguém lê, ficam de fora.
var camposDoWebhook = []string{"messages", "messaging_postbacks", "message_deliveries", "message_reads"}

// Verificar confirma que o token alcança a Página e devolve o nome dela. É o
// que o Testar Conexão usa: só leitura, sem efeito colateral.
func Verificar(ctx context.Context, cfg *FacebookConfig) (string, error) {
	_, nome, err := verificar(ctx, cfg)
	return nome, err
}

// Conectar verifica o token e assina a Página nos eventos do webhook. Sem a
// assinatura o canal fica "connected" no banco e nenhuma mensagem chega — era
// um passo manual no painel da Meta que ninguém sabia que faltava.
func Conectar(ctx context.Context, cfg *FacebookConfig) (string, error) {
	client, nome, err := verificar(ctx, cfg)
	if err != nil {
		return "", err
	}
	if err := client.SubscribeToWebhooks(ctx); err != nil {
		return "", err
	}
	return nome, nil
}

func verificar(ctx context.Context, cfg *FacebookConfig) (*Client, string, error) {
	client, err := NewClient(cfg)
	if err != nil {
		return nil, "", err
	}
	pagina, err := client.GetPageInfo(ctx)
	if err != nil {
		return nil, "", err
	}
	return client, pagina.Name, nil
}
