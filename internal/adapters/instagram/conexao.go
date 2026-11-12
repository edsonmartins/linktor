package instagram

import (
	"context"
	"strings"
)

// ConfigFromMap monta a configuração do canal a partir de um ou mais mapas
// (config, credentials); em chave repetida o último vence. O token pode estar
// em access_token (formulário e OAuth) ou page_access_token; o host é decidido
// pelo próprio token, em NewClient. Única casa dessa regra: o sender, o
// Conectar, o Testar Conexão e o webhook usam todos esta função.
func ConfigFromMap(mapas ...map[string]string) *InstagramConfig {
	junto := map[string]string{}
	for _, m := range mapas {
		for k, v := range m {
			junto[k] = v
		}
	}
	return &InstagramConfig{
		InstagramID:     junto["instagram_id"],
		AccessToken:     junto["access_token"],
		AppID:           junto["app_id"],
		AppSecret:       junto["app_secret"],
		VerifyToken:     junto["verify_token"],
		PageID:          junto["page_id"],
		PageAccessToken: junto["page_access_token"],
	}
}

// camposDoWebhook são os eventos que o handler consome. A Meta recusa a
// assinatura inteira se um campo não for aceito, então só vai o que é lido.
var camposDoWebhook = []string{"messages", "messaging_seen", "message_reactions"}

// camposDoPerfil são os campos de perfil que o Instagram aceita. O padrão da
// Graph (first_name, last_name) é recusado com #100 em contas do Instagram.
var camposDoPerfil = []string{"name", "username", "profile_pic"}

// Verificar confirma que o token alcança a conta e devolve o @ dela. É o que o
// Testar Conexão usa: só leitura, sem efeito colateral.
func Verificar(ctx context.Context, cfg *InstagramConfig) (string, error) {
	_, usuario, err := verificar(ctx, cfg)
	return usuario, err
}

// Conectar verifica o token e assina a conta nos eventos do webhook. Sem a
// assinatura o canal fica "connected" no banco e nenhuma mensagem chega — era
// um botão escondido no painel da Meta.
func Conectar(ctx context.Context, cfg *InstagramConfig) (string, error) {
	client, usuario, err := verificar(ctx, cfg)
	if err != nil {
		return "", err
	}
	if err := client.SubscribeToWebhooks(ctx); err != nil {
		return "", err
	}
	return usuario, nil
}

func verificar(ctx context.Context, cfg *InstagramConfig) (*Client, string, error) {
	client, err := NewClient(cfg)
	if err != nil {
		return nil, "", err
	}
	conta, err := client.GetAccountInfo(ctx)
	if err != nil {
		return nil, "", err
	}
	usuario := conta.Username
	if usuario == "" {
		usuario = conta.Name
	}
	return client, usuario, nil
}

// NomeDoPerfil busca o nome de exibição de um IGSID; sem nome, devolve o @.
// Vazio quando o perfil não está disponível (em modo de desenvolvimento só
// quem tem papel no app expõe o perfil).
func (c *Client) NomeDoPerfil(ctx context.Context, igsid string) (string, error) {
	perfil, err := c.api.GetUserProfile(ctx, igsid, camposDoPerfil)
	if err != nil {
		return "", err
	}
	if nome := strings.TrimSpace(perfil.Name); nome != "" {
		return nome, nil
	}
	if perfil.Username != "" {
		return "@" + perfil.Username, nil
	}
	return "", nil
}
