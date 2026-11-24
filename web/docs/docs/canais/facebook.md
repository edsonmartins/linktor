---
sidebar_position: 5
title: Facebook Messenger
description: Conectar uma Página do Facebook para receber e responder mensagens do Messenger — app na Meta, token de Página, permissões, webhook, janela de 24h e tipos suportados.
---

# Facebook Messenger

## Resumo

O canal Facebook recebe e responde mensagens enviadas a uma **Página do Facebook** pelo Messenger,
usando a Graph API da Meta. O contato inicia a conversa; você responde dentro da janela de 24 horas.

- Valor de `type` na API: **`facebook`**
- Nome no painel: **Facebook Messenger** ("Plataforma Meta Messenger")

| Recurso | Suportado | Observação |
|---|---|---|
| Texto | Sim | Até 2000 caracteres |
| Imagem, vídeo, áudio, arquivo | Sim | Por URL; uma mídia por mensagem |
| Legenda em mídia | Como mensagem separada | O texto vai antes, em outra mensagem |
| Localização | Recebe | |
| Cliques em botão (postback) | Recebe | Chegam como mensagem de texto com o `payload` do botão |
| Botões e respostas rápidas (envio) | Não | Uma mensagem interativa com opções falha neste canal |
| Modelos de mensagem (templates) | Não | Não há modelos aprovados como no WhatsApp |
| Grupos | Não | |
| Entregue | Sim | `message.delivered` |
| Lida | Não | O evento de leitura da Meta não identifica a mensagem |

## Antes de começar

1. Uma **Página do Facebook** da qual você é administrador.
2. Um app em [developers.facebook.com](https://developers.facebook.com), com o **ID do app** e a
   **Chave secreta do app**.
3. O servidor Linktor em **HTTPS** com certificado válido.
4. No Linktor, papel de administrador ou [chave de API](../painel/chaves-de-api.md) com
   `channels:write`.

:::note
Os nomes de menu da Meta mudam com frequência. Se um rótulo estiver diferente, procure a função
equivalente.
:::

## Configurar na origem

### 1. Criar o app e adicionar o Messenger

1. Em **Meus apps** → **Criar app**, escolha o tipo **Empresa** e vincule ao seu portfólio empresarial.
2. No painel do app, adicione o produto **Messenger**.
3. Em **Configurações do app** → **Básico**, copie o **ID do app** e a **Chave secreta do app**.

### 2. Permissões

| Permissão | Para quê |
|---|---|
| `pages_messaging` | Enviar e receber mensagens da Página |
| `pages_manage_metadata` | Assinar a Página nos eventos do webhook |
| `pages_read_engagement` | Ler dados da Página |
| `pages_show_list` | Listar as Páginas no login |

Em modo de desenvolvimento, o app só troca mensagens com pessoas que têm papel no app. Para atender o
público, peça **acesso avançado** a `pages_messaging` na **Análise do app** e publique-o.

### 3. Gerar o token da Página

- **Para testar:** em **Messenger** → **Configurações da API do Messenger**, conecte a Página e clique
  em **Gerar token**. Esse token depende da sessão do usuário que o gerou.
- **Para produção:** crie um **usuário do sistema** em **Configurações do negócio** → **Usuários** →
  **Usuários do sistema**, atribua a Página e o app a ele e gere um token com as permissões acima e
  validade **Nunca**.

O **ID da Página** aparece em **Sobre** → **Transparência da Página**, ou em:

```bash
curl "https://graph.facebook.com/v23.0/me?fields=id,name&access_token=EAAxxxxxxxx"
```

### 4. Webhook

Feito depois de criar o canal no Linktor — veja [Webhook](#webhook).

## Configurar no Linktor

1. No painel, abra **Canais** e clique em **Facebook Messenger**.
2. A tela tem as abas **Conectar** (login pelo Facebook, só na criação), **Manual**, **Webhook** e
   **Guia**.
3. Na aba **Manual**, preencha os campos abaixo e clique em **Testar Conexão**. O teste lê a Página de
   verdade na Meta e responde com o nome dela (`connected to Page "Sua Página"`).
4. Clique em **Criar Canal**.
5. Na lista de canais, abra o menu do canal e clique em **Conectar**.

| Rótulo na tela | Chave | Onde fica | Obrigatório |
|---|---|---|---|
| **Nome do Canal** | `name` | raiz | Sim |
| **App ID** | `app_id` | `config` | Sim na tela; a API não exige |
| **Segredo do App** | `app_secret` | `credentials` | Sim (sem ele o webhook recusa tudo) |
| **ID da Página** | `page_id` | `config` | Sim |
| **Token de Acesso da Página** | `page_access_token` | `credentials` | Sim na criação; em branco na edição mantém o guardado |
| **Token de Verificação** | `verify_token` | `config` | Sim (vem **Gerado automaticamente**) |

Chaves aceitas sem campo na tela (defina pela API):

| Chave | Onde fica | Efeito |
|---|---|---|
| `access_token` | `credentials` | Aceito como token da Página quando `page_access_token` está vazio (é a chave que o fluxo **Conectar** grava) |
| `webhook_url` | raiz | Endereço da sua aplicação que recebe os eventos |
| `webhook_secret` | `credentials` | Chave que assina os eventos enviados à sua aplicação |
| `webhook_events` | `config` | Eventos a entregar, separados por vírgula |

### O que o botão Conectar faz

**Conectar** (ou `POST /channels/CHANNEL_ID/connect`) executa dois passos na Meta:

1. Lê a Página com o token — se falhar, o canal não conecta e o erro da Meta é devolvido
   (`failed to connect channel: ...`).
2. **Assina a Página no app** para os eventos `messages`, `messaging_postbacks`, `message_deliveries`
   e `message_reads`.

Sem o segundo passo o canal ficaria marcado como conectado e nenhuma mensagem chegaria. O nome da
Página é gravado em `config.page_name`.

### Conectar pelo Facebook (aba Conectar)

A aba **Conectar** faz o login pela Meta e lista as suas Páginas:

1. Informe **App ID** e **Segredo do App** e clique em **Conectar com Facebook**.
2. No popup, autorize `pages_messaging`, `pages_read_engagement`, `pages_manage_metadata` e
   `pages_show_list`.
3. Escolha a Página em **Selecione uma Página** e clique em **Criar Canal**.

O canal é criado já conectado, com o token de longa duração da Página, o `app_id`, o `app_secret` e um
token de verificação, e a Página é assinada nos eventos.

:::warning Limitações deste fluxo
- A URL de retorno é `<BASE_URL>/api/v1/oauth/facebook/callback` e precisa estar em **Login do
  Facebook** → **URIs de redirecionamento OAuth válidos**. O painel lê o código de autorização direto
  do popup, o que só funciona quando o endereço público da API (`BASE_URL`) tem a **mesma origem** do
  painel. Em instalações com domínios diferentes, use a aba **Manual**.
- `POST /oauth/refresh` com `{"channel_id": "CHANNEL_ID"}` troca o token guardado por um novo de longa
  duração. Ele lê o token em `credentials.access_token`, então atende os canais criados por este
  fluxo; para um canal criado pela aba **Manual**, gere um token novo e atualize o campo.
:::

## Webhook {#webhook}

URL que a Meta deve chamar:

```text
https://linktor.suaempresa.com/api/v1/webhooks/facebook/CHANNEL_ID
```

O caminho `/api/v1/webhooks/messenger/CHANNEL_ID` é um apelido para a mesma rota.

No app da Meta, em **Messenger** → **Configurações da API do Messenger** → **Webhooks**:

1. **URL de retorno** — a URL acima (copie da aba **Webhook** do canal).
2. **Token de verificação** — o valor de **Token de Verificação**.
3. **Verificar e salvar**. A Meta faz um `GET` com `hub.mode=subscribe`, `hub.verify_token` e
   `hub.challenge`; o Linktor devolve o desafio se o token conferir, ou 403 `verification failed`.
4. Na Página, assine os campos **`messages`**, **`messaging_postbacks`**, **`message_deliveries`** e
   **`message_reads`**. O botão **Conectar** do Linktor faz essa assinatura pela API.

Validação da assinatura:

- A Meta envia `X-Hub-Signature-256: sha256=<hex>`, o HMAC-SHA256 do corpo com o **Segredo do App**.
- O Linktor recalcula com `credentials.app_secret`. Assinatura inválida, ou canal sem `app_secret`,
  responde 400 com `Invalid webhook signature` (em modo `release`, um canal sem segredo responde 401
  `webhook secret not configured`).
- Eventos cujo objeto não é `page` respondem `{"status": "ignored"}`.

Os webhooks passam por deduplicação (janela de 6 horas). Mensagens de eco (as que a própria Página
enviou) são descartadas.

## Criar o canal pela API

`POST /api/v1/channels` com o scope `channels:write`, seguido de `connect` para validar o token e
assinar a Página nos eventos.

<Tabs groupId="linguagem">
<TabItem value="curl" label="cURL">

```bash
curl -X POST "$LINKTOR_URL/api/v1/channels" \
  -H "X-API-Key: $LINKTOR_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "type": "facebook",
    "name": "Messenger Loja",
    "config": {
      "page_id": "100000000000000",
      "app_id": "1234567890123456",
      "verify_token": "um-token-aleatorio-longo"
    },
    "credentials": {
      "page_access_token": "EAAxxxxxxxx",
      "app_secret": "APP_SECRET_DA_META",
      "webhook_secret": "um-segredo-longo"
    },
    "webhook_url": "https://app.suaempresa.com/webhooks/linktor"
  }'

# Valida o token e assina a conta nos eventos do webhook
curl -X POST "$LINKTOR_URL/api/v1/channels/CHANNEL_ID/connect" \
  -H "X-API-Key: $LINKTOR_API_KEY"
```

</TabItem>
<TabItem value="typescript" label="TypeScript">

```typescript
import { LinktorClient } from '@linktor/sdk';

const client = new LinktorClient({
  baseUrl: `${process.env.LINKTOR_URL}/api/v1`,
  apiKey: process.env.LINKTOR_API_KEY,
});

const canal = await client.channels.create({
  type: 'facebook',
  name: 'Messenger Loja',
  config: {
    page_id: '100000000000000',
    app_id: '1234567890123456',
    verify_token: 'um-token-aleatorio-longo',
  },
  credentials: {
    page_access_token: 'EAAxxxxxxxx',
    app_secret: 'APP_SECRET_DA_META',
    webhook_secret: 'um-segredo-longo',
  },
  webhook_url: 'https://app.suaempresa.com/webhooks/linktor',
});

await client.channels.connect(canal.id);
console.log(`${process.env.LINKTOR_URL}/api/v1/webhooks/facebook/${canal.id}`);
```

</TabItem>
<TabItem value="python" label="Python">

```python
import os
import requests

base = os.environ["LINKTOR_URL"].rstrip("/") + "/api/v1"
headers = {"X-API-Key": os.environ["LINKTOR_API_KEY"]}

r = requests.post(f"{base}/channels", headers=headers, timeout=30, json={
    "type": "facebook",
    "name": "Messenger Loja",
    "config": {
        "page_id": "100000000000000",
        "app_id": "1234567890123456",
        "verify_token": "um-token-aleatorio-longo",
    },
    "credentials": {
        "page_access_token": "EAAxxxxxxxx",
        "app_secret": "APP_SECRET_DA_META",
        "webhook_secret": "um-segredo-longo",
    },
    "webhook_url": "https://app.suaempresa.com/webhooks/linktor",
})
r.raise_for_status()
canal = r.json()["data"]

requests.post(f"{base}/channels/{canal['id']}/connect", headers=headers, timeout=30).raise_for_status()
print(f"{base}/webhooks/facebook/{canal['id']}")
```

</TabItem>
<TabItem value="go" label="Go">

```go
package main

import (
	"context"
	"fmt"
	"log"
	"os"

	linktor "github.com/linktor/linktor-go"
	"github.com/linktor/linktor-go/types"
)

func main() {
	ctx := context.Background()
	client := linktor.NewClient(
		linktor.WithBaseURL(os.Getenv("LINKTOR_URL")+"/api/v1"),
		linktor.WithAPIKey(os.Getenv("LINKTOR_API_KEY")),
	)

	canal, err := client.Channels.Create(ctx, &types.CreateChannelInput{
		Type: types.ChannelTypeFacebook,
		Name: "Messenger Loja",
		Config: map[string]string{
			"page_id":      "100000000000000",
			"app_id":       "1234567890123456",
			"verify_token": "um-token-aleatorio-longo",
		},
		Credentials: map[string]string{
			"page_access_token": "EAAxxxxxxxx",
			"app_secret":        "APP_SECRET_DA_META",
			"webhook_secret":    "um-segredo-longo",
		},
		WebhookURL: "https://app.suaempresa.com/webhooks/linktor",
	})
	if err != nil {
		log.Fatal(err)
	}
	if _, err := client.Channels.Connect(ctx, canal.ID); err != nil {
		log.Fatal(err)
	}
	fmt.Printf("%s/api/v1/webhooks/facebook/%s\n", os.Getenv("LINKTOR_URL"), canal.ID)
}
```

</TabItem>
<TabItem value="java" label="Java">

```java
import io.linktor.LinktorClient;
import io.linktor.types.Channel;
import java.util.Map;

LinktorClient client = LinktorClient.builder()
        .baseUrl(System.getenv("LINKTOR_URL") + "/api/v1")
        .apiKey(System.getenv("LINKTOR_API_KEY"))
        .build();

Channel.ChannelModel canal = client.channels.create(
        Channel.CreateChannelInput.builder()
                .type(Channel.ChannelType.FACEBOOK)
                .name("Messenger Loja")
                .config(Map.of(
                        "page_id", "100000000000000",
                        "app_id", "1234567890123456",
                        "verify_token", "um-token-aleatorio-longo"))
                .credentials(Map.of(
                        "page_access_token", "EAAxxxxxxxx",
                        "app_secret", "APP_SECRET_DA_META",
                        "webhook_secret", "um-segredo-longo"))
                .webhookUrl("https://app.suaempresa.com/webhooks/linktor")
                .build());

client.channels.connect(canal.getId());
System.out.println(System.getenv("LINKTOR_URL") + "/api/v1/webhooks/facebook/" + canal.getId());
```

</TabItem>
<TabItem value="csharp" label="C#">

```csharp
using System.Net.Http.Json;
using System.Text.Json;

var http = new HttpClient
{
    BaseAddress = new Uri(Environment.GetEnvironmentVariable("LINKTOR_URL")!.TrimEnd('/') + "/api/v1/"),
};
http.DefaultRequestHeaders.Add("X-API-Key", Environment.GetEnvironmentVariable("LINKTOR_API_KEY"));

var resposta = await http.PostAsJsonAsync("channels", new
{
    type = "facebook",
    name = "Messenger Loja",
    config = new Dictionary<string, string>
    {
        ["page_id"] = "100000000000000",
        ["app_id"] = "1234567890123456",
        ["verify_token"] = "um-token-aleatorio-longo",
    },
    credentials = new Dictionary<string, string>
    {
        ["page_access_token"] = "EAAxxxxxxxx",
        ["app_secret"] = "APP_SECRET_DA_META",
        ["webhook_secret"] = "um-segredo-longo",
    },
    webhook_url = "https://app.suaempresa.com/webhooks/linktor",
});
resposta.EnsureSuccessStatusCode();

var corpo = await resposta.Content.ReadFromJsonAsync<JsonElement>();
var id = corpo.GetProperty("data").GetProperty("id").GetString();

(await http.PostAsync($"channels/{id}/connect", null)).EnsureSuccessStatusCode();
Console.WriteLine($"{http.BaseAddress}webhooks/facebook/{id}");
```

</TabItem>
<TabItem value="php" label="PHP">

```php
<?php
require __DIR__ . '/vendor/autoload.php';

$http = new GuzzleHttp\Client([
    'base_uri' => rtrim(getenv('LINKTOR_URL'), '/') . '/api/v1/',
    'headers'  => ['X-API-Key' => getenv('LINKTOR_API_KEY')],
    'timeout'  => 30,
]);

$resposta = $http->post('channels', ['json' => [
    'type' => 'facebook',
    'name' => 'Messenger Loja',
    'config' => [
        'page_id'      => '100000000000000',
        'app_id'       => '1234567890123456',
        'verify_token' => 'um-token-aleatorio-longo',
    ],
    'credentials' => [
        'page_access_token' => 'EAAxxxxxxxx',
        'app_secret'        => 'APP_SECRET_DA_META',
        'webhook_secret'    => 'um-segredo-longo',
    ],
    'webhook_url' => 'https://app.suaempresa.com/webhooks/linktor',
]]);

$canal = json_decode((string) $resposta->getBody(), true)['data'];
$http->post("channels/{$canal['id']}/connect");

echo rtrim(getenv('LINKTOR_URL'), '/') . "/api/v1/webhooks/facebook/{$canal['id']}", PHP_EOL;
```

</TabItem>
<TabItem value="rust" label="Rust">

```rust
use linktor::{ChannelType, CreateChannelInput, LinktorClient};
use std::collections::HashMap;

#[tokio::main]
async fn main() -> Result<(), Box<dyn std::error::Error>> {
    let servidor = std::env::var("LINKTOR_URL")?;
    let client = LinktorClient::builder()
        .base_url(format!("{}/api/v1", servidor.trim_end_matches('/')))
        .api_key(std::env::var("LINKTOR_API_KEY")?)
        .build()?;

    let config = HashMap::from([
        ("page_id".to_string(), "100000000000000".to_string()),
        ("app_id".to_string(), "1234567890123456".to_string()),
        ("verify_token".to_string(), "um-token-aleatorio-longo".to_string()),
    ]);
    let credentials = HashMap::from([
        ("page_access_token".to_string(), "EAAxxxxxxxx".to_string()),
        ("app_secret".to_string(), "APP_SECRET_DA_META".to_string()),
        ("webhook_secret".to_string(), "um-segredo-longo".to_string()),
    ]);

    let canal = client
        .channels()
        .create(
            CreateChannelInput::new("Messenger Loja", ChannelType::Facebook)
                .config(config)
                .credentials(credentials)
                .webhook_url("https://app.suaempresa.com/webhooks/linktor"),
        )
        .await?;

    client.channels().connect(&canal.id).await?;
    println!("{}/api/v1/webhooks/facebook/{}", servidor, canal.id);
    Ok(())
}
```

</TabItem>
</Tabs>

:::warning Ao atualizar, reenvie o `config` inteiro
`PUT /channels/CHANNEL_ID` **substitui** `config` por completo; só `credentials` é mesclado (valor
vazio ou `__redacted__` mantém o segredo guardado). Um update que mande só parte do `config` apaga
`page_id` e `verify_token`.
:::

Para testar credenciais sem criar o canal: `POST /channels/test-facebook` com `config` e
`credentials` no corpo. Informando `channel_id`, os campos em branco são completados com o que está
guardado. Mais em [API de canais](../api/canais.md).

## Enviar e receber

### Enviar uma mensagem

`POST /api/v1/messages/send` (scope `messages:send`). O destinatário (`to`) é o **PSID** — o id da
pessoa no escopo da sua Página. Você o obtém em `data.message.senderId` de uma mensagem recebida; não
é possível iniciar conversa com quem nunca escreveu para a Página.

<Tabs groupId="linguagem">
<TabItem value="curl" label="cURL">

```bash
curl -X POST "$LINKTOR_URL/api/v1/messages/send" \
  -H "X-API-Key: $LINKTOR_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "channel_id": "CHANNEL_ID",
    "to": "7654321098765432",
    "content_type": "text",
    "text": "Entregamos, sim. Qual é o seu bairro?"
  }'
```

</TabItem>
<TabItem value="typescript" label="TypeScript">

```typescript
import { LinktorClient } from '@linktor/sdk';

const client = new LinktorClient({
  baseUrl: `${process.env.LINKTOR_URL}/api/v1`,
  apiKey: process.env.LINKTOR_API_KEY,
});

const res: any = await client.messages.sendText('CHANNEL_ID', '7654321098765432', 'Entregamos, sim. Qual é o seu bairro?');

// O SDK devolve o envelope inteiro: o resultado está em res.data
console.log(res.data.id, res.data.status); // ... queued
```

</TabItem>
<TabItem value="python" label="Python">

```python
import os
import requests

base = os.environ["LINKTOR_URL"].rstrip("/") + "/api/v1"

r = requests.post(
    f"{base}/messages/send",
    headers={"X-API-Key": os.environ["LINKTOR_API_KEY"]},
    timeout=30,
    json={
        "channel_id": "CHANNEL_ID",
        "to": "7654321098765432",
        "content_type": "text",
        "text": "Entregamos, sim. Qual é o seu bairro?",
    },
)
r.raise_for_status()
envio = r.json()["data"]
print(envio["id"], envio["status"])  # ... queued
```

</TabItem>
<TabItem value="go" label="Go">

```go
package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
)

func main() {
	corpo, _ := json.Marshal(map[string]any{
		"channel_id":   "CHANNEL_ID",
		"to":           "7654321098765432",
		"content_type": "text",
		"text":         "Entregamos, sim. Qual é o seu bairro?",
	})

	req, _ := http.NewRequest(http.MethodPost,
		os.Getenv("LINKTOR_URL")+"/api/v1/messages/send", bytes.NewReader(corpo))
	req.Header.Set("X-API-Key", os.Getenv("LINKTOR_API_KEY"))
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusAccepted {
		log.Fatalf("envio recusado: %s", resp.Status)
	}

	var saida struct {
		Data struct {
			ID     string `json:"id"`
			Status string `json:"status"`
		} `json:"data"`
	}
	json.NewDecoder(resp.Body).Decode(&saida)
	fmt.Println(saida.Data.ID, saida.Data.Status) // ... queued
}
```

</TabItem>
<TabItem value="java" label="Java">

```java
import com.google.gson.Gson;
import com.google.gson.JsonObject;
import java.net.URI;
import java.net.http.HttpClient;
import java.net.http.HttpRequest;
import java.net.http.HttpResponse;
import java.util.Map;

Gson gson = new Gson();
String corpo = gson.toJson(Map.of(
        "channel_id", "CHANNEL_ID",
        "to", "7654321098765432",
        "content_type", "text",
        "text", "Entregamos, sim. Qual é o seu bairro?"));

HttpRequest request = HttpRequest.newBuilder(
                URI.create(System.getenv("LINKTOR_URL") + "/api/v1/messages/send"))
        .header("Content-Type", "application/json")
        .header("X-API-Key", System.getenv("LINKTOR_API_KEY"))
        .POST(HttpRequest.BodyPublishers.ofString(corpo))
        .build();

HttpResponse<String> response = HttpClient.newHttpClient()
        .send(request, HttpResponse.BodyHandlers.ofString());
if (response.statusCode() != 202) {
    throw new IllegalStateException("Envio recusado: " + response.body());
}
JsonObject data = gson.fromJson(response.body(), JsonObject.class).getAsJsonObject("data");
System.out.println(data.get("id").getAsString() + " " + data.get("status").getAsString());
```

</TabItem>
<TabItem value="csharp" label="C#">

```csharp
using System.Net.Http.Json;
using System.Text.Json;

var http = new HttpClient
{
    BaseAddress = new Uri(Environment.GetEnvironmentVariable("LINKTOR_URL")!.TrimEnd('/') + "/api/v1/"),
};
http.DefaultRequestHeaders.Add("X-API-Key", Environment.GetEnvironmentVariable("LINKTOR_API_KEY"));

var resposta = await http.PostAsJsonAsync("messages/send", new
{
    channel_id = "CHANNEL_ID",
    to = "7654321098765432",
    content_type = "text",
    text = "Entregamos, sim. Qual é o seu bairro?",
});
resposta.EnsureSuccessStatusCode();

var data = (await resposta.Content.ReadFromJsonAsync<JsonElement>()).GetProperty("data");
Console.WriteLine($"{data.GetProperty("id")} {data.GetProperty("status")}"); // ... queued
```

</TabItem>
<TabItem value="php" label="PHP">

```php
<?php
require __DIR__ . '/vendor/autoload.php';

$http = new GuzzleHttp\Client([
    'base_uri' => rtrim(getenv('LINKTOR_URL'), '/') . '/api/v1/',
    'headers'  => ['X-API-Key' => getenv('LINKTOR_API_KEY')],
    'timeout'  => 30,
]);

$resposta = $http->post('messages/send', ['json' => [
    'channel_id'   => 'CHANNEL_ID',
    'to'           => '7654321098765432',
    'content_type' => 'text',
    'text'         => 'Entregamos, sim. Qual é o seu bairro?',
]]);

$envio = json_decode((string) $resposta->getBody(), true)['data'];
echo $envio['id'], ' ', $envio['status'], PHP_EOL; // ... queued
```

</TabItem>
<TabItem value="rust" label="Rust">

```rust
use serde_json::{json, Value};

#[tokio::main]
async fn main() -> Result<(), Box<dyn std::error::Error>> {
    let base = format!("{}/api/v1", std::env::var("LINKTOR_URL")?.trim_end_matches('/'));

    let resposta = reqwest::Client::new()
        .post(format!("{base}/messages/send"))
        .header("X-API-Key", std::env::var("LINKTOR_API_KEY")?)
        .json(&json!({
            "channel_id": "CHANNEL_ID",
            "to": "7654321098765432",
            "content_type": "text",
            "text": "Entregamos, sim. Qual é o seu bairro?"
        }))
        .send()
        .await?
        .error_for_status()?;

    let corpo: Value = resposta.json().await?;
    println!("{} {}", corpo["data"]["id"], corpo["data"]["status"]); // ... "queued"
    Ok(())
}
```

</TabItem>
</Tabs>

A resposta `202` traz `data.id`, `data.conversation_id`, `data.channel_id` e `data.status: "queued"`.

Mídia vai por URL pública, em `attachments`:

```json
{
  "channel_id": "CHANNEL_ID",
  "to": "7654321098765432",
  "content_type": "document",
  "text": "Segue o orçamento",
  "attachments": [
    {
      "url": "https://app.suaempresa.com/arquivos/orcamento-1042.pdf",
      "type": "document",
      "filename": "orcamento-1042.pdf",
      "mime_type": "application/pdf"
    }
  ]
}
```

O texto não vira legenda: ele é enviado como uma mensagem de texto e a mídia vem em seguida. Corpo
completo e idempotência em [Enviar mensagens](../api/enviar-mensagens.md).

### O que sua aplicação recebe

Com `webhook_url` e `credentials.webhook_secret` definidos no canal, cada mensagem recebida gera:

```json
{
  "id": "evt_2f4a6c80-received",
  "type": "message.received",
  "timestamp": "2026-11-24T14:03:11Z",
  "tenantId": "3f6f0c1e-9a51-4f0e-b0a5-6b6f6d2c1a11",
  "environment": "production",
  "data": {
    "message": {
      "id": "2f4a6c80-1b3d-4e5f-8a90-b1c2d3e4f506",
      "direction": "inbound",
      "contentType": "text",
      "content": { "text": "Boa tarde, vocês entregam no centro?" },
      "senderId": "7654321098765432",
      "senderType": "contact",
      "metadata": {
        "senderName": "Maria Souza",
        "externalId": "m_AbCdEfGhIjKlMnOpQrStUvWxYz0123456789"
      }
    },
    "conversationId": "b3a9d2c4-55e1-4f0a-8a44-0f6f1f1f7a21",
    "contactId": "0c2b7f7e-3f1d-4d57-a0ad-0f9e5f1a2b33",
    "channelId": "CHANNEL_ID",
    "channelType": "facebook"
  }
}
```

- `senderName` é buscado no perfil na primeira mensagem do contato. Em modo de desenvolvimento a Meta
  só expõe o perfil de quem tem papel no app; nesse caso o contato nasce sem nome.
- Mídia chega em `content.media.url`, apontando para o endereço da Meta. Só o **primeiro** anexo de
  cada mensagem é considerado.
- Um clique em botão (**Começar**, menu persistente, botão de um modelo criado fora do Linktor) chega
  como `message.received` de texto, com o `payload` do botão em `content.text`. Esse evento vem sem
  `externalId`.

Status de envio: `message.sent`, `message.delivered` e `message.failed`. Não há `message.read`.
Assinatura dos eventos (`X-Linktor-Signature`, `X-Linktor-Timestamp`) e reentrega em
[Receber eventos](../api/receber-eventos.md).

## Limites e particularidades

- **Janela de 24 horas.** A Meta só aceita mensagens até 24h depois da última mensagem da pessoa. O
  Linktor **não bloqueia** o envio fora da janela neste canal: a mensagem vai e a Meta recusa, gerando
  `message.failed`.
- **Fora da janela.** Por padrão toda mensagem sai como resposta (`messaging_type: RESPONSE`). Para
  usar uma tag da Meta, envie em `metadata` as chaves `messaging_type` (`MESSAGE_TAG`) e `message_tag`
  — por exemplo `HUMAN_AGENT`, que estende a resposta humana para até 7 dias e exige permissão
  aprovada no app. As demais tags têm uso restrito pela política da Meta.
- **Quem inicia é o cliente.** Não há como abrir conversa por telefone ou e-mail.
- **Tamanhos declarados pelo canal.** Texto de até 2000 caracteres; mídia de até 25 MB; JPEG, PNG,
  GIF, MP4, MP3 e PDF.
- **Sem botões no envio.** Mensagens interativas com opções (de bots e fluxos) falham neste canal com
  `unsupported content kind`. Em fluxos que atendem o Messenger, apresente as opções em texto.
- **Erros.** Erro definitivo da Meta não é reenviado; limite de taxa (429), erro 5xx e falha de rede
  são reenviados.
- **Uma Página por canal.** Para atender várias Páginas, crie um canal para cada uma; o mesmo app
  serve a todas.

## Solução de problemas

| Sintoma | Causa | O que fazer |
|---|---|---|
| **Testar Conexão** falha com erro `190` | Token inválido, expirado ou de outro app | Gere um token de Página novo |
| **Testar Conexão** falha com erro `100` ou `10` | **ID da Página** errado ou token sem acesso à Página | Confira o id e as permissões do token |
| **Conectar** falha com `failed to connect channel` | A Meta recusou a assinatura da Página: falta `pages_manage_metadata` ou `pages_messaging` | Gere o token com as quatro permissões |
| A Meta recusa **Verificar e salvar** | Token de verificação diferente ou URL com id errado | Copie os dois valores da aba **Webhook** |
| Webhook responde 400 `Invalid webhook signature` | **Segredo do App** vazio ou de outro app | Preencha com o segredo do app que entrega o webhook |
| Canal conectado, mas nenhuma mensagem chega | Página não assinada nos eventos | Clique em **Conectar** de novo |
| Só chegam mensagens de quem tem papel no app | App em modo de desenvolvimento | Peça acesso avançado a `pages_messaging` e publique o app |
| `message.failed` com erro `10` e subcódigo `2018278` | Fora da janela de 24h | Responda dentro da janela, ou use uma tag permitida |
| `message.failed` com erro `551` | A pessoa bloqueou a Página ou não está disponível | Não há ação |
| `message.failed` com `unsupported content kind` | Mensagem com botões | Envie as opções como texto |
| Contato aparece sem nome | A Meta não expôs o perfil | Normal em modo de desenvolvimento; o nome é buscado de novo enquanto o contato estiver sem nome |
| Popup de login abre e a tela não avança | `BASE_URL` em origem diferente da do painel | Use a aba **Manual** |
| `POST /messages/send` responde `channel is not connected` | Canal criado e não conectado | Clique em **Conectar** |

Logs por canal em [Observabilidade](../painel/observabilidade.md). O mesmo app e a mesma Página
atendem também o [Instagram Direct](./instagram.md).
