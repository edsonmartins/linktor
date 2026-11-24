---
sidebar_position: 4
title: Go
description: Referência do SDK Go do Linktor (sdks/go) — instalação com replace a partir do repositório, cliente, canais, envio direto, webhook em net/http, erros e limitações conhecidas.
---

# SDK Go

O SDK Go (pasta `sdks/go` do repositório, pacote `linktor`) é um cliente HTTP sobre `net/http`, sem
dependências externas em uso. Ele está alinhado com a API em **canais**, **envio direto de mensagem** e
**verificação de webhook** — o verificador é o espelho do código que o servidor usa para assinar. Os
demais recursos foram escritos contra um contrato antigo e funcionam só em parte. Esta página documenta
o que foi conferido contra o servidor e lista, no fim, o que não funciona.

:::warning Três comportamentos valem para o SDK inteiro
1. **Só `Channels` desembrulha o envelope.** A API responde `{"success", "data", "error", "meta"}`; os
   outros métodos decodificam esse corpo direto no tipo da entidade. Resultado: `Messages.SendDirect`
   envia a mensagem, mas devolve a struct **vazia** (sem `ID`); `Conversations.Get`, `Contacts.Get` e
   `Contacts.Create` também voltam vazios, com `err == nil`. O contorno é o
   [adaptador de envelope](#adaptador-de-envelope), que entra como uma `Option` do cliente.
2. **O código e a mensagem de erro se perdem.** O SDK procura `code` e `message` na raiz do corpo e a
   API manda em `error.code` / `error.message`. Sem o adaptador, `*linktor.Error` só traz `Status`.
3. **Só `Channels`, `Messages` e `VRE` têm tipos em snake_case**, como a API. Os outros modelos usam
   tags camelCase (`contactId`, `createdAt`), então campos compostos chegam zerados na leitura e são
   ignorados pelo servidor na escrita. Campos de uma palavra só (`id`, `name`, `status`, `tags`) funcionam.
:::

| Recurso | Cobertura | Observação |
|---|---|---|
| Canais | sim | Criar, listar, obter, atualizar, remover, conectar (QR), pair code, desconectar |
| Mensagens — envio direto | sim | Só texto; o retorno precisa do adaptador de envelope. Mídia: HTTP nativo |
| Mensagens — para uma conversa | não | `Conversations.SendMessage` manda um corpo que o servidor recusa |
| Conversas | parcial | Só `List` (sem filtros) e `Get` (com adaptador); `id`, `status`, `tags` |
| Contatos | parcial | `List`, `Get`, `Create`, `Delete`; `Update` usa `PATCH`, que não existe |
| Bots | parcial | Só `List` e `Get`, com `id`, `name`, `type`, `status` |
| IA | não | Chama `/ai/completions` e `/ai/embeddings`, que não existem |
| Base de conhecimento | parcial | Só `List` e `Get`; `Query` chama rota inexistente |
| Fluxos | parcial | Só `List` e `Get`; `Execute` chama rota inexistente |
| Analytics | não | Chama `/analytics/dashboard` e `/analytics/realtime`, que não existem |
| VRE | parcial | `Render` e os atalhos `Render*` funcionam; o resto não |
| Verificação de webhook | sim | Compatível com o contrato do servidor |
| Tempo real (WebSocket) | não | O SDK não tem cliente WebSocket |
| Autenticação por API key | sim | Envia `X-API-Key` |
| Login (JWT) | não | `Auth.Login` não lê a resposta; use `WithAccessToken` com um token obtido por fora |

## Instalação

O `go.mod` do SDK declara `module github.com/linktor/linktor-go`, sem versão (não há tag de release),
mas o código vive em `github.com/edsonmartins/linktor`, na subpasta `sdks/go`. Um `go get` pelo caminho
do repositório é recusado pelo Go, porque o módulo encontrado declara outro caminho. O caminho garantido
é clonar o repositório e apontar o module path declarado para o clone com `replace`:

```bash
git clone https://github.com/edsonmartins/linktor.git ../linktor

go mod edit -require=github.com/linktor/linktor-go@v0.0.0
go mod edit -replace=github.com/linktor/linktor-go=../linktor/sdks/go
go mod tidy
```

O `go.mod` da sua aplicação passa a ter as duas primeiras linhas abaixo; o import usa sempre o module
path declarado, não o caminho do repositório:

```go
require github.com/linktor/linktor-go v0.0.0
replace github.com/linktor/linktor-go => ../linktor/sdks/go

import (
	linktor "github.com/linktor/linktor-go"
	"github.com/linktor/linktor-go/types"
)
```

:::note
Não foi possível confirmar se `github.com/linktor/linktor-go` resolve direto por `go get`; trate o
`replace` como o caminho suportado. O clone precisa existir onde o build roda (CI, imagem Docker) — ou
rode `go mod vendor` e versione a pasta `vendor/`. Fixe o clone num commit para ter builds reprodutíveis.
:::

## Requisitos

- Go 1.21 ou mais novo (`go 1.21` no `go.mod`; o SDK usa generics).
- O `go.mod` lista `github.com/gorilla/websocket v1.5.1`, mas nenhum arquivo do SDK o importa: nada é
  baixado no build.

## Criar o cliente

```go
client := linktor.NewClient(
	linktor.WithBaseURL(os.Getenv("LINKTOR_URL")+"/api/v1"),
	linktor.WithAPIKey(os.Getenv("LINKTOR_API_KEY")),
	linktor.WithTimeout(15*time.Second),
	linktor.WithMaxRetries(2),
	func(c *linktor.ClientConfig) { c.RetryDelay = 500 * time.Millisecond },
)
```

**A base URL precisa incluir `/api/v1`.** O SDK concatena `BaseURL + caminho` (`/channels`,
`/messages/send`) sem acrescentar prefixo. O padrão, `https://api.linktor.io`, não serve para uma
instalação própria: passe sempre `WithBaseURL`.

| Opção | Campo em `ClientConfig` | Tipo | Padrão |
|---|---|---|---|
| `WithBaseURL(url)` | `BaseURL` | `string` | `https://api.linktor.io` |
| `WithAPIKey(key)` | `APIKey` | `string` | vazio |
| `WithAccessToken(token)` | `AccessToken` | `string` | vazio |
| `WithTimeout(d)` | `Timeout` | `time.Duration` | `30s` |
| `WithMaxRetries(n)` | `MaxRetries` | `int` | `3` |
| sem função pronta | `RetryDelay` | `time.Duration` | `1s` |
| sem função pronta | `HTTPClient` | `*http.Client` | cliente novo com `Timeout` |

`Option` é `func(*ClientConfig)`: para os dois campos sem função pronta, passe uma função literal, como
no exemplo. Quando `HTTPClient` é informado, `Timeout` é ignorado — vale o timeout do cliente que você
passou. Não há opção de headers customizados.

**Retentativas.** Erro de rede e resposta `5xx` são repetidos até `MaxRetries` vezes, com espera de
`RetryDelay × 2^tentativa` (1 s, 2 s, 4 s). `4xx` (inclusive `429`) não é repetido. A repetição vale
também para `POST`: num envio de mensagem, use `metadata.idempotency_key` para não duplicar.

Todo método recebe `context.Context` como primeiro argumento; use-o para prazo e cancelamento.

### Adaptador de envelope {#adaptador-de-envelope}

Este `http.RoundTripper` desfaz o envelope antes de o SDK ler a resposta. Com ele, `SendDirect` devolve
o id da mensagem, os métodos `Get`/`Create` devolvem a entidade e `*linktor.Error` passa a ter `Code`,
`Message` e `Details`. Respostas em lista e as rotas de canal passam intactas, porque o SDK já as lê
no formato original.

```go
type envelopeTransport struct{ base http.RoundTripper }

func (t envelopeTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	resp, err := t.base.RoundTrip(req)
	if err != nil {
		return nil, err
	}
	raw, err := io.ReadAll(resp.Body)
	resp.Body.Close()
	if err != nil {
		return nil, err
	}
	var env struct {
		Success *bool           `json:"success"`
		Data    json.RawMessage `json:"data"`
		Error   json.RawMessage `json:"error"`
	}
	out := raw
	if json.Unmarshal(raw, &env) == nil && env.Success != nil {
		switch {
		case !*env.Success && len(env.Error) > 0:
			out = env.Error // {code, message, details} -> *linktor.Error
		case *env.Success && len(env.Data) > 0 && env.Data[0] == '{' &&
			!strings.Contains(req.URL.Path, "/channels"): // Channels já trata o envelope
			out = env.Data
		}
	}
	resp.Body = io.NopCloser(bytes.NewReader(out))
	resp.ContentLength = int64(len(out))
	return resp, nil
}

// WithEnvelope é uma linktor.Option: injeta o http.Client com o adaptador.
func WithEnvelope(timeout time.Duration) linktor.Option {
	return func(c *linktor.ClientConfig) {
		c.HTTPClient = &http.Client{
			Timeout:   timeout,
			Transport: envelopeTransport{base: http.DefaultTransport},
		}
	}
}
```

Uso: `linktor.NewClient(linktor.WithBaseURL(...), linktor.WithAPIKey(...), WithEnvelope(30*time.Second))`.
Imports: `bytes`, `encoding/json`, `io`, `net/http`, `strings`, `time`.

## Autenticação

| Configuração | Header enviado | Quando usar |
|---|---|---|
| `WithAPIKey("lk_sua_chave")` | `X-API-Key: lk_sua_chave` | Integração servidor a servidor |
| `WithAccessToken(jwt)` | `Authorization: Bearer <jwt>` | Agir como um usuário do painel |

Se os dois estiverem configurados, **só a API key é enviada**. `client.SetAPIKey(key)` e
`client.SetAccessToken(token)` trocam a credencial depois de criado o cliente (para passar a usar o
JWT, zere a chave com `SetAPIKey("")`). A API key é criada no painel — veja
[Chaves de API](../painel/chaves-de-api.md) — e cada operação exige o scope correspondente
(`messages:send` para envio). Detalhes em [Autenticação](../api/autenticacao.md).

`Auth.Login` não serve para obter o JWT: ele chama `POST /auth/login` corretamente, mas não lê
`data.access_token` e grava um token vazio no cliente. Faça o login com `net/http` e passe o
`access_token` para `WithAccessToken`.

## Canais

Todos os métodos de `client.Channels` funcionam sem o adaptador.

| Método | Retorno |
|---|---|
| `List(ctx, params *types.ListChannelsParams)` | `[]types.Channel` |
| `Get(ctx, id string)` | `*types.Channel` |
| `Create(ctx, input *types.CreateChannelInput)` | `*types.Channel` |
| `Update(ctx, id string, input *types.UpdateChannelInput)` | `*types.Channel` |
| `Delete(ctx, id string)` | só `error` (a API responde 204) |
| `Connect(ctx, id string)` | `*types.ConnectResult` |
| `RequestPairCode(ctx, id, phoneNumber string)` | `*types.ConnectResult` |
| `Disconnect(ctx, id string)` | `*types.Channel` **vazio** — confira só o `error` |

`types.Channel` traz `ID`, `TenantID`, `Type`, `Name`, `Identifier`, `Enabled`, `ConnectionStatus`,
`Config`, `WebhookURL`, `CreatedAt`, `UpdatedAt`. Credenciais nunca voltam na resposta.

### Criar um canal e conectar por QR

`Connect` devolve o conteúdo do QR em `QRCode` (texto para gerar a imagem com a biblioteca de QR de sua
preferência) e a validade em segundos em `ExpiresIn`. O laço abaixo consulta o canal a cada 3 s e,
quando o QR expira, chama `Connect` de novo para receber outro.

```go
ch, err := client.Channels.Create(ctx, &types.CreateChannelInput{
	Type: types.ChannelTypeWhatsApp, // "whatsapp" (não oficial)
	Name: "WhatsApp Vendas",
})
if err != nil {
	log.Fatal(err)
}
if err := conectarPorQR(ctx, client, ch.ID); err != nil {
	log.Fatal(err)
}

func conectarPorQR(ctx context.Context, client *linktor.Client, channelID string) error {
	for {
		res, err := client.Channels.Connect(ctx, channelID)
		if err != nil {
			return err
		}
		if res.Channel != nil && res.Channel.ConnectionStatus == types.ConnectionStatusConnected {
			return nil
		}
		if res.PasskeyRequired {
			return fmt.Errorf("conta protegida por passkey: conecte pelo painel")
		}
		if res.QRCode != "" {
			fmt.Println("Escaneie este QR no WhatsApp:", res.QRCode)
		}

		validade := time.Duration(res.ExpiresIn) * time.Second
		if validade <= 0 {
			validade = 20 * time.Second
		}
		expira := time.Now().Add(validade)
		for time.Now().Before(expira) {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(3 * time.Second):
			}
			ch, err := client.Channels.Get(ctx, channelID)
			if err != nil {
				return err
			}
			if ch.ConnectionStatus == types.ConnectionStatusConnected {
				return nil
			}
		}
		// QR expirou: a próxima volta chama Connect de novo e recebe um QR novo.
	}
}
```

### Conectar por código de pareamento

```go
pair, err := client.Channels.RequestPairCode(ctx, ch.ID, "+5511999999999")
if err != nil {
	log.Fatal(err)
}
fmt.Println("Digite no celular:", pair.PairCode)
```

O passo a passo no aparelho está em [WhatsApp não-oficial](../canais/whatsapp-nao-oficial.md).

### Atualizar, listar e remover

O webhook de saída do canal é configurado no próprio canal: `WebhookURL` é o destino,
`credentials.webhook_secret` é o segredo da assinatura e `config.webhook_events` filtra os tipos de
evento (lista separada por vírgula; vazio entrega todos).

```go
_, err = client.Channels.Update(ctx, ch.ID, &types.UpdateChannelInput{
	WebhookURL:  "https://app.suaempresa.com/webhooks/linktor",
	Credentials: map[string]string{"webhook_secret": os.Getenv("LINKTOR_WEBHOOK_SECRET")},
	Config:      map[string]string{"webhook_events": "message.received,message.sent,message.failed"},
})

canais, err := client.Channels.List(ctx, nil)
for _, c := range canais {
	fmt.Println(c.ID, c.Type, c.Name, c.Enabled, c.ConnectionStatus)
}

err = client.Channels.Delete(ctx, ch.ID)
```

:::note
- Os filtros de `ListChannelsParams` (`Type`, `Status`, `Search`, `Limit`…) vão na query string, mas o
  servidor não filtra a listagem de canais: vêm todos os canais do tenant. Filtre no seu código.
- Os campos de `UpdateChannelInput` usam `omitempty`: campo vazio significa "não alterar", então não dá
  para limpar o `WebhookURL` pelo SDK.
- `CreateChannelInput` não tem `environment`, e o SDK não cobre `PUT /channels/:id/status` nem
  `PUT /channels/:id/enabled`. Para isso, use HTTP nativo.
:::

## Enviar mensagem

`client.Messages` chama `POST /messages/send`: você informa canal e destinatário e o Linktor resolve (ou
cria) contato e conversa. A API responde `202` com a mensagem na fila; o resultado da entrega chega
depois pelo webhook (`message.sent`, `message.failed`). Requer o scope `messages:send`.

| Método | Retorno |
|---|---|
| `SendDirect(ctx, input *types.DirectSendInput)` | `*types.DirectSendResult` |
| `SendDirectText(ctx, channelID, to, text string)` | `*types.DirectSendResult` |

```go
_, err := client.Messages.SendDirectText(ctx, "CHANNEL_ID", "+5511999999999",
	"Olá! Seu pedido saiu para entrega.")
if err != nil {
	log.Fatalf("envio recusado: %v", err)
}
```

Sem o adaptador, `err == nil` confirma que a mensagem entrou na fila, mas o `DirectSendResult` vem com
todos os campos vazios. Para guardar o id (e correlacionar com o webhook), crie o cliente com
[`WithEnvelope`](#adaptador-de-envelope):

```go
res, err := client.Messages.SendDirect(ctx, &types.DirectSendInput{
	ChannelID: "CHANNEL_ID",
	To:        "+5511999999999",
	Text:      "Olá! Seu pedido saiu para entrega.",
	Metadata:  map[string]string{"idempotency_key": "pedido-1042-saiu-para-entrega"},
})
if err != nil {
	log.Fatal(err)
}
fmt.Println(res.ID, res.ConversationID, res.Status) // Status: "queued"
```

`Metadata` é `map[string]string`. `idempotency_key` (única por tenant, até 255 caracteres) faz uma
chamada repetida devolver a mensagem original em vez de enviar outra. Chaves internas do Linktor
(`conversation_id`, `channel_id`, `tenant_id`…) são recusadas com 400.

### Mídia: HTTP nativo

`DirectSendInput` não tem `attachments`, então o SDK envia só texto. Para mídia, chame a rota com
`net/http`:

```go
body, _ := json.Marshal(map[string]any{
	"channel_id":   "CHANNEL_ID",
	"to":           "+5511999999999",
	"content_type": "image",
	"text":         "Comprovante de entrega",
	"attachments": []map[string]any{{
		"url":       "https://cdn.suaempresa.com/comprovante.jpg",
		"type":      "image",
		"mime_type": "image/jpeg",
		"filename":  "comprovante.jpg",
	}},
})

req, _ := http.NewRequestWithContext(ctx, http.MethodPost,
	os.Getenv("LINKTOR_URL")+"/api/v1/messages/send", bytes.NewReader(body))
req.Header.Set("Content-Type", "application/json")
req.Header.Set("X-API-Key", os.Getenv("LINKTOR_API_KEY"))

resp, err := http.DefaultClient.Do(req)
if err != nil {
	log.Fatal(err)
}
defer resp.Body.Close()

var env struct {
	Data  struct{ ID, Status string }     `json:"data"`
	Error *struct{ Code, Message string } `json:"error"`
}
if err := json.NewDecoder(resp.Body).Decode(&env); err != nil {
	log.Fatalf("HTTP %d: %v", resp.StatusCode, err)
}
if env.Error != nil {
	log.Fatalf("HTTP %d %s: %s", resp.StatusCode, env.Error.Code, env.Error.Message)
}
fmt.Println(env.Data.ID, env.Data.Status) // 202, "queued"
```

Todos os campos da rota estão em [Enviar mensagens](../api/enviar-mensagens.md).

## Conversas, contatos, bots, base de conhecimento e fluxos

Nestes recursos o SDK serve para leitura simples. As listagens funcionam sem o adaptador; `Get` e
`Create` exigem o adaptador.

| Método | Campos que chegam preenchidos |
|---|---|
| `Conversations.List(ctx, params)` / `Conversations.Get(ctx, id)` | `ID`, `Status`, `Tags`, `Metadata` |
| `Contacts.List(ctx, params)` / `Contacts.Get(ctx, id)` | `ID`, `Name`, `Email`, `Phone`, `Tags` |
| `Contacts.Create(ctx, input)` | idem; o servidor grava `Name`, `Email`, `Phone`, `Tags` |
| `Contacts.Delete(ctx, id)` | só `error` |
| `Bots.List(ctx, params)` / `Bots.Get(ctx, id)` | `ID`, `Name`, `Type`, `Status`, `Config` |
| `KnowledgeBases.List(ctx)` / `KnowledgeBases.Get(ctx, id)` | `ID`, `Name`, `Description`, `Status` |
| `Flows.List(ctx)` / `Flows.Get(ctx, id)` | `ID`, `Name`, `Description` |

```go
convs, err := client.Conversations.List(ctx, nil)
if err != nil {
	log.Fatal(err)
}
for _, c := range convs.Data {
	fmt.Println(c.ID, c.Status, c.Tags)
}

_, err = client.Contacts.Create(ctx, &types.CreateContactInput{
	Name: "Maria Souza", Email: "cliente@exemplo.com", Phone: "+5511999999999", Tags: []string{"vip"},
})
```

:::warning O que se perde
- **Os parâmetros de listagem são ignorados**: `List` nunca monta query string. Vem sempre a primeira
  página com os padrões do servidor, sem filtro, e `Pagination` volta zerado (o SDK não lê `meta`).
- **Campos compostos voltam zerados**: `ContactID`, `ChannelID`, `UnreadCount`, `LastMessageAt`,
  `AvatarURL`, `CustomFields`, `CreatedAt`, `UpdatedAt` e semelhantes.
- **Em `Contacts.Create`**, `AvatarURL`, `CustomFields` e `ExternalID` são enviados em camelCase e o
  servidor os ignora.

Se você precisa de filtro, paginação ou desses campos, chame a rota com `net/http` — veja
[Erros e paginação](../api/erros-e-paginacao.md).
:::

## VRE

`client.VRE.Render` funciona sem o adaptador: as rotas `/vre/*` respondem JSON sem envelope. Elas só
existem quando o serviço VRE está habilitado no servidor (senão, 404).

| Método | Modelo (`template_id`) |
|---|---|
| `Render(ctx, input *types.VRERenderRequest)` | o de `input.TemplateID` |
| `RenderMenu(ctx, tenantID, titulo, opcoes, channel)` | `menu_opcoes` |
| `RenderProductCard(ctx, tenantID, produto, channel)` | `card_produto` |
| `RenderOrderStatus(ctx, tenantID, status, channel)` | `status_pedido` |
| `RenderProductList(ctx, tenantID, titulo, produtos, channel)` | `lista_produtos` |
| `RenderConfirmation(ctx, tenantID, confirmacao, channel)` | `confirmacao` |
| `RenderPixPayment(ctx, tenantID, pix, channel)` | `cobranca_pix` |

Todos devolvem `*types.VRERenderResponse`. Passe `tenantID` como `""`: o servidor usa o tenant da
credencial e responde 403 se o valor informado for de outro tenant.

```go
img, err := client.VRE.RenderMenu(ctx, "", "Como podemos ajudar?", []types.MenuOpcaoData{
	{Label: "Acompanhar pedido"},
	{Label: "Falar com atendente"},
}, types.VREChannelWhatsApp)
if err != nil {
	log.Fatal(err)
}

b64 := img.ImageBase64 // vem como data URL: "data:image/png;base64,...."
if i := strings.Index(b64, ","); strings.HasPrefix(b64, "data:") && i >= 0 {
	b64 = b64[i+1:]
}
png, _ := base64.StdEncoding.DecodeString(b64)
_ = os.WriteFile("menu.png", png, 0o644)
```

`RenderTimeMs` chega em nanossegundos, apesar do nome. Erros do VRE vêm como `{"error": "..."}`, então
`*linktor.Error` traz só `Status`, com ou sem adaptador.

## Verificar a assinatura do webhook

O verificador do SDK é compatível com o servidor: lê `X-Linktor-Signature` (HMAC-SHA256 em hexadecimal,
sem prefixo) e `X-Linktor-Timestamp` (epoch em segundos), assina `timestamp + "." + corpo cru`, compara
em tempo constante e recusa timestamps fora da tolerância (`linktor.DefaultTolerance`, 300 s). O segredo
é o `credentials.webhook_secret` do canal.

| Função | O que faz |
|---|---|
| `ConstructEvent(payload []byte, headers http.Header, secret string, tolerance int)` | Valida e devolve `*WebhookEvent` |
| `VerifyWebhook(payload, headers, secret, tolerance)` | Valida assinatura e tolerância; devolve `bool` |
| `VerifyWebhookSignature(payload, signature, timestamp, secret)` | Só a assinatura, sem checar a idade |
| `ComputeTimestampedSignature(timestamp, payload, secret)` | Calcula a assinatura esperada |
| `WebhookHandler(secret, handlers)` | `http.HandlerFunc` pronto, com um callback por tipo de evento |

`tolerance` é em segundos; `0` usa o padrão. `ComputeSignature(payload, secret)` assina só o corpo e
**não** corresponde ao que o servidor envia — não a use para validar.

Receptor mínimo com `net/http`, lendo o corpo cru antes de qualquer parse:

```go
package main

import (
	"io"
	"log"
	"net/http"
	"os"

	linktor "github.com/linktor/linktor-go"
)

func main() {
	secret := os.Getenv("LINKTOR_WEBHOOK_SECRET") // o mesmo valor de credentials.webhook_secret

	http.HandleFunc("/webhooks/linktor", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		raw, err := io.ReadAll(http.MaxBytesReader(w, r.Body, 1<<20))
		if err != nil {
			http.Error(w, "corpo ilegível", http.StatusBadRequest)
			return
		}

		event, err := linktor.ConstructEvent(raw, r.Header, secret, linktor.DefaultTolerance)
		if err != nil {
			http.Error(w, "assinatura inválida", http.StatusUnauthorized)
			return
		}

		switch event.Type {
		case "message.received":
			msg, _ := event.Data["message"].(map[string]interface{})
			content, _ := msg["content"].(map[string]interface{})
			log.Printf("evento %s: conversa %v, texto %q",
				event.ID, event.Data["conversationId"], content["text"])
		case "message.sent", "message.delivered", "message.read", "message.failed":
			log.Printf("mensagem %v agora está %v", event.Data["messageId"], event.Data["status"])
		}
		w.WriteHeader(http.StatusOK)
	})

	log.Fatal(http.ListenAndServe(":8080", nil))
}
```

`WebhookEvent` tem `ID`, `Type`, `Timestamp` (`time.Time`), `TenantID` e `Data`
(`map[string]interface{}`). O envelope do webhook é camelCase (`conversationId`, `messageId`), diferente
da API REST; o campo `environment` do envelope não é exposto pela struct — se precisar dele, decodifique
o corpo cru. A lista de eventos e o formato de `data` estão em
[Receber eventos](../api/receber-eventos.md).

O atalho `linktor.WebhookHandler(secret, handlers)` devolve um handler pronto que faz a mesma validação
e chama o callback do mapa `handlers` cuja chave é o tipo do evento; ele responde 400 quando a
assinatura não confere e 200 nos demais casos.

## Tratamento de erros

Há um único tipo de erro de API, `*linktor.Error`, devolvido para qualquer resposta com status 400 ou
maior. Falha de rede, timeout e cancelamento do `context` voltam como o erro original do `net/http`;
resposta que não decodifica volta como erro de `encoding/json`.

| Campo | Tipo | Conteúdo |
|---|---|---|
| `Status` | `int` | Status HTTP — sempre preenchido |
| `Code` | `string` | `error.code` da API (`VALIDATION_ERROR`, `NOT_FOUND`, `UNAUTHORIZED`…) — só com o adaptador |
| `Message` | `string` | `error.message` da API — só com o adaptador |
| `Details` | `map[string]interface{}` | `error.details` — só com o adaptador |
| `RequestID` | `string` | Header `X-Request-ID` da resposta, quando presente |

```go
res, err := client.Messages.SendDirectText(ctx, "CHANNEL_ID", "+5511999999999", "Olá!")
if err != nil {
	var apiErr *linktor.Error
	switch {
	case errors.As(err, &apiErr) && apiErr.Status == http.StatusUnauthorized:
		log.Fatal("credencial inválida ou sem o scope messages:send")
	case errors.As(err, &apiErr):
		log.Fatalf("HTTP %d %s: %s %v", apiErr.Status, apiErr.Code, apiErr.Message, apiErr.Details)
	default:
		log.Fatalf("falha de rede: %v", err) // timeout, DNS, contexto cancelado
	}
}
_ = res
```

Sem o adaptador, `err.Error()` imprime `[] ` (código e mensagem vazios): decida pelo `Status`. Se o
corpo do erro não for JSON (um proxy respondendo HTML, por exemplo), `Code` vira `UNKNOWN_ERROR` e
`Message` recebe o corpo cru. Os códigos da API estão em [Erros e paginação](../api/erros-e-paginacao.md).

## Limitações conhecidas

| Método | O que ele faz | Por que não funciona | Alternativa |
|---|---|---|---|
| `Auth.Login` | `POST /auth/login` | Não lê `data.access_token`; grava token vazio | Login por `net/http` + `WithAccessToken` |
| `Auth.GetCurrentUser` | `GET /auth/me` | Rota não existe (a real é `GET /me`) | `GET /me` por `net/http` |
| `Conversations.SendMessage`, `SendText` | `POST /conversations/:id/messages` com `{"text"}` | O servidor espera `content_type` e `content`; responde 400 | `Messages.SendDirect` ou HTTP nativo |
| `Conversations.Update`, `Resolve` | `PATCH /conversations/:id` | Não há `PATCH`; a API usa `PUT` e `POST /conversations/:id/resolve` | HTTP nativo |
| `Conversations.Assign` | `POST /conversations/:id/assign` com `agentId` | O servidor exige `user_id`; responde 400 | HTTP nativo |
| `Contacts.Update` | `PATCH /contacts/:id` | Não há `PATCH`; a API usa `PUT` | HTTP nativo |
| `Bots.Create` | `POST /bots` | `CreateBotInput` não tem `provider` nem `model`, obrigatórios | HTTP nativo |
| `AI.Completions.Complete` | `POST /ai/completions` | Rota não existe (a real é `POST /ai/complete`) | HTTP nativo |
| `AI.Embeddings.Embed` | `POST /ai/embeddings` | Rota não existe | Sem equivalente na API |
| `KnowledgeBases.Query` | `POST /knowledge-bases/:id/query` | Rota não existe (a real é `/search`, com `query` e `limit`) | HTTP nativo |
| `Flows.Execute` | `POST /flows/:id/execute` | Rota não existe | `POST /flows/:id/test` por HTTP nativo |
| `Analytics.GetDashboard`, `GetRealtime` | `GET /analytics/dashboard`, `/realtime` | Rotas não existem | `GET /analytics/overview` por HTTP nativo |
| `VRE.RenderAndSend` | `POST /vre/render-and-send` com `conversation_id` | O servidor exige `send_to`, `channel` e `channel_id` | HTTP nativo |
| `VRE.ListTemplates` | `GET /vre/templates` | A API devolve lista de strings; o SDK espera objetos e falha ao decodificar | HTTP nativo |
| `VRE.Preview` | `POST /vre/templates/:id/preview` | A rota é `GET` | HTTP nativo |

Além disso, `AI.Agents`, `AI.Completions` e `AI.Embeddings` são ponteiros nunca inicializados pelo
`NewClient` (chamar um método deles causa panic), e não há cliente WebSocket: para eventos em tempo real, use o webhook do canal ou veja
[Tempo real](../api/tempo-real.md). As rotas corretas estão na [visão geral da API](../api/visao-geral.md);
a comparação entre linguagens, na [visão geral dos SDKs](./visao-geral.md).

## Referência: método → rota HTTP

Caminhos relativos a `/api/v1`.

| Método | Rota |
|---|---|
| `Channels.List` | `GET /channels` |
| `Channels.Get` | `GET /channels/:id` |
| `Channels.Create` | `POST /channels` |
| `Channels.Update` | `PUT /channels/:id` |
| `Channels.Delete` | `DELETE /channels/:id` |
| `Channels.Connect` | `POST /channels/:id/connect` |
| `Channels.RequestPairCode` | `POST /channels/:id/pair` |
| `Channels.Disconnect` | `POST /channels/:id/disconnect` |
| `Messages.SendDirect`, `Messages.SendDirectText` | `POST /messages/send` |
| `Conversations.List` | `GET /conversations` |
| `Conversations.Get` | `GET /conversations/:id` |
| `Contacts.List` | `GET /contacts` |
| `Contacts.Get` | `GET /contacts/:id` |
| `Contacts.Create` | `POST /contacts` |
| `Contacts.Delete` | `DELETE /contacts/:id` |
| `Bots.List` | `GET /bots` |
| `Bots.Get` | `GET /bots/:id` |
| `KnowledgeBases.List` | `GET /knowledge-bases` |
| `KnowledgeBases.Get` | `GET /knowledge-bases/:id` |
| `Flows.List` | `GET /flows` |
| `Flows.Get` | `GET /flows/:id` |
| `VRE.Render` e atalhos `Render*` | `POST /vre/render` |
| `Auth.Logout` | `POST /auth/logout` |
