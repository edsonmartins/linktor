---
sidebar_position: 4
title: Instagram Direct
description: Conectar uma conta profissional do Instagram para receber e responder mensagens diretas — app na Meta, permissões, token, webhook, janela de 24h e tipos suportados.
---

# Instagram Direct

## Resumo

O canal Instagram recebe e responde **mensagens diretas (DMs)** de uma conta profissional do Instagram
pela Graph API da Meta. O contato sempre inicia a conversa; você responde dentro da janela de 24 horas.

- Valor de `type` na API: **`instagram`**
- Nome no painel: **Instagram** ("Integração com Meta Graph API")

| Recurso | Suportado | Observação |
|---|---|---|
| Texto | Sim | Até 1000 caracteres |
| Imagem, vídeo, áudio | Sim | Por URL; uma mídia por mensagem |
| Arquivo | Envia como `file` | Sujeito à aceitação da Meta para a conta |
| Legenda em mídia | Como mensagem separada | O texto vai antes, em outra mensagem |
| Botões e listas | Não | Uma mensagem interativa com opções falha neste canal |
| Modelos de mensagem (templates) | Não | Não existem no Instagram |
| Resposta a story e menção em story | Recebe | Chegam como mensagem de texto |
| Reações | Recebe, como texto | O emoji chega como uma mensagem de texto |
| Grupos | Não | |
| Entregue / lida | Não | Só `message.sent` e `message.failed` |

## Antes de começar

1. Uma conta **profissional** do Instagram (Comercial ou Criador de conteúdo). Conta pessoal não tem
   acesso à API de mensagens.
2. A conta vinculada a uma **Página do Facebook** que você administra (necessário no caminho com token
   de Página).
3. No aplicativo do Instagram, em **Configurações** → **Mensagens e respostas a stories** →
   **Controles de mensagens** → **Ferramentas conectadas**, a opção **Permitir acesso às mensagens**
   ligada. Sem ela a Meta não entrega nada à API.
4. Um app em [developers.facebook.com](https://developers.facebook.com), com o **Segredo do App** em
   mãos.
5. O servidor Linktor em **HTTPS** com certificado válido.
6. No Linktor, papel de administrador ou [chave de API](../painel/chaves-de-api.md) com
   `channels:write`.

:::note
Os nomes de menu da Meta mudam com frequência. Se um rótulo estiver diferente, procure a função
equivalente.
:::

## Configurar na origem

### 1. Criar o app e adicionar o Instagram

1. Em **Meus apps** → **Criar app**, escolha o tipo **Empresa** e vincule ao seu portfólio empresarial.
2. Adicione o produto **Instagram** (ou **Messenger**, que traz as configurações do Instagram junto).
3. Em **Configurações do app** → **Básico**, copie o **ID do app** e a **Chave secreta do app**.

### 2. Permissões

| Caminho de login | Permissões |
|---|---|
| Login do Facebook (token de Página, `EAA…`) | `instagram_basic`, `instagram_manage_messages`, `pages_manage_metadata`, `pages_show_list`, `pages_read_engagement` |
| Login do Instagram (token `IGAA…`) | `instagram_business_basic`, `instagram_business_manage_messages` |

Em modo de desenvolvimento, o app só troca mensagens com contas que têm papel no app (administrador,
desenvolvedor, testador). Para atender o público, peça **acesso avançado** às permissões na **Análise
do app** e publique-o.

### 3. Obter o token e o id da conta

O Linktor aceita os dois tipos de token e **escolhe o servidor da Meta pelo prefixo**:

| Token começa com | Origem | O Linktor chama |
|---|---|---|
| `IGAA` | Login do Instagram (gerado na configuração da API do Instagram) | `graph.instagram.com` |
| qualquer outro (em geral `EAA`) | Token de Página ou de usuário do sistema, via login do Facebook | `graph.facebook.com` |

Para um token `EAA` duradouro, o caminho recomendado é um **usuário do sistema** no portfólio
empresarial, com acesso à Página e à conta do Instagram e as permissões da tabela acima, com validade
**Nunca**.

O **ID da Conta Instagram Business** não é o nome de usuário: é um número, em geral começando por
`1784`. Com um token de Página:

```bash
curl "https://graph.facebook.com/v23.0/PAGE_ID?fields=instagram_business_account&access_token=EAAxxxxxxxx"
```

```json
{ "instagram_business_account": { "id": "17841400000000000" }, "id": "100000000000000" }
```

### 4. Webhook

Feito depois de criar o canal no Linktor — veja [Webhook](#webhook).

## Configurar no Linktor

1. No painel, abra **Canais** e clique em **Instagram**.
2. A tela tem as abas **Conectar** (login pelo Facebook, só na criação), **Manual**, **Webhook** e
   **Guia**.
3. Na aba **Manual**, preencha os campos abaixo e clique em **Testar Conexão**. O teste lê a conta de
   verdade na Meta e responde com o nome de usuário (`connected to Instagram @suaconta`).
4. Clique em **Criar Canal**.
5. Na lista de canais, abra o menu do canal e clique em **Conectar**.

| Rótulo na tela | Chave | Onde fica | Obrigatório |
|---|---|---|---|
| **Nome do Canal** | `name` | raiz | Sim |
| **ID da Conta Instagram Business** | `instagram_id` | `config` | Sim |
| **Token de Acesso** | `access_token` | `credentials` | Sim na criação; em branco na edição mantém o guardado |
| **App ID** | `app_id` | `config` | Não |
| **Segredo do App** | `app_secret` | `credentials` | **Sim, na prática** (ver aviso) |
| **Token de Verificação** | `verify_token` | `config` | Sim (vem **Gerado automaticamente**) |

Chaves aceitas sem campo na tela (defina pela API):

| Chave | Onde fica | Efeito |
|---|---|---|
| `page_access_token` | `credentials` | Usado quando `access_token` está vazio |
| `page_id` | `config` | Página vinculada (gravado pelo fluxo **Conectar**) |
| `webhook_url` | raiz | Endereço da sua aplicação que recebe os eventos |
| `webhook_secret` | `credentials` | Chave que assina os eventos enviados à sua aplicação |
| `webhook_events` | `config` | Eventos a entregar, separados por vírgula |

:::warning O Segredo do App aparece como "Opcional", mas sem ele nada chega
A tela marca **Segredo do App** como opcional. O webhook, porém, **sempre** valida a assinatura da
Meta: um canal sem `app_secret` recusa toda entrega e nenhuma mensagem entra. Preencha o campo.
:::

### O que o botão Conectar faz

**Conectar** (ou `POST /channels/CHANNEL_ID/connect`) executa dois passos na Meta:

1. Lê a conta com o token — se falhar, o canal não conecta e o erro da Meta é devolvido
   (`failed to connect channel: ...`).
2. **Assina a conta nos eventos** `messages`, `messaging_seen` e `message_reactions`.

Sem o segundo passo o canal ficaria marcado como conectado e nenhuma mensagem chegaria. O nome de
usuário é gravado em `config.username`.

### Conectar pelo Facebook (aba Conectar)

A aba **Conectar** faz o login pela Meta e lista as contas do Instagram ligadas às suas Páginas:

1. Informe **Facebook App ID** e **Facebook App Secret** e clique em **Conectar via Facebook**.
2. No popup, autorize as permissões `instagram_basic`, `instagram_manage_messages`,
   `pages_manage_metadata`, `pages_read_engagement` e `pages_show_list`.
3. Escolha a conta em **Selecione uma Conta** e clique em **Criar Canal**.

O canal é criado já conectado, com o token de longa duração da Página, o `app_id`, o `app_secret` e um
token de verificação.

:::warning Limitações deste fluxo
- A URL de retorno é `<BASE_URL>/api/v1/oauth/instagram/callback` e precisa estar em **Login do
  Facebook** → **URIs de redirecionamento OAuth válidos**. O painel lê o código de autorização direto
  do popup, o que só funciona quando o endereço público da API (`BASE_URL`) tem a **mesma origem** do
  painel. Em instalações com domínios diferentes, use a aba **Manual**.
- O canal criado por este fluxo **não é assinado** nos eventos do webhook. Chame
  `POST /channels/CHANNEL_ID/connect` depois de criar, para executar a assinatura.
- Um token obtido por login expira (cerca de 60 dias). `POST /oauth/refresh` com `{"channel_id": "CHANNEL_ID"}`
  troca o token guardado por um novo de longa duração; para um canal criado pela aba **Manual**,
  envie também `app_id` e `app_secret` no corpo.
:::

## Webhook {#webhook}

URL que a Meta deve chamar:

```text
https://linktor.suaempresa.com/api/v1/webhooks/instagram/CHANNEL_ID
```

No app da Meta, na configuração de webhooks do produto **Instagram**:

1. **URL de retorno** — a URL acima (copie da aba **Webhook** do canal).
2. **Token de verificação** — o valor de **Token de Verificação**.
3. **Verificar e salvar**. A Meta faz um `GET` com `hub.mode=subscribe`, `hub.verify_token` e
   `hub.challenge`; o Linktor devolve o desafio se o token conferir, ou 403 `verification failed`.
4. Assine os campos **`messages`**, **`messaging_seen`** e **`message_reactions`**.

Validação da assinatura:

- A Meta envia `X-Hub-Signature-256: sha256=<hex>`, o HMAC-SHA256 do corpo com o **Segredo do App**.
- O Linktor recalcula com `credentials.app_secret`. Assinatura inválida, ou canal sem `app_secret`,
  responde 400 com `Invalid webhook signature` (em modo `release`, um canal sem segredo responde 401
  `webhook secret not configured`).
- Eventos que não são do Instagram respondem `{"status": "ignored"}`.

Os webhooks passam por deduplicação (janela de 6 horas). Mensagens de eco (as que a própria conta
enviou) e mensagens apagadas são descartadas.

## Criar o canal pela API

`POST /api/v1/channels` com o scope `channels:write`, seguido de `connect` para validar o token e
assinar a conta nos eventos.

<Tabs groupId="linguagem">
<TabItem value="curl" label="cURL">

```bash
curl -X POST "$LINKTOR_URL/api/v1/channels" \
  -H "X-API-Key: $LINKTOR_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "type": "instagram",
    "name": "Instagram Loja",
    "config": {
      "instagram_id": "17841400000000000",
      "app_id": "1234567890123456",
      "verify_token": "um-token-aleatorio-longo"
    },
    "credentials": {
      "access_token": "EAAxxxxxxxx",
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
  type: 'instagram',
  name: 'Instagram Loja',
  config: {
    instagram_id: '17841400000000000',
    app_id: '1234567890123456',
    verify_token: 'um-token-aleatorio-longo',
  },
  credentials: {
    access_token: 'EAAxxxxxxxx',
    app_secret: 'APP_SECRET_DA_META',
    webhook_secret: 'um-segredo-longo',
  },
  webhook_url: 'https://app.suaempresa.com/webhooks/linktor',
});

await client.channels.connect(canal.id);
console.log(`${process.env.LINKTOR_URL}/api/v1/webhooks/instagram/${canal.id}`);
```

</TabItem>
<TabItem value="python" label="Python">

```python
import os
import requests

base = os.environ["LINKTOR_URL"].rstrip("/") + "/api/v1"
headers = {"X-API-Key": os.environ["LINKTOR_API_KEY"]}

r = requests.post(f"{base}/channels", headers=headers, timeout=30, json={
    "type": "instagram",
    "name": "Instagram Loja",
    "config": {
        "instagram_id": "17841400000000000",
        "app_id": "1234567890123456",
        "verify_token": "um-token-aleatorio-longo",
    },
    "credentials": {
        "access_token": "EAAxxxxxxxx",
        "app_secret": "APP_SECRET_DA_META",
        "webhook_secret": "um-segredo-longo",
    },
    "webhook_url": "https://app.suaempresa.com/webhooks/linktor",
})
r.raise_for_status()
canal = r.json()["data"]

requests.post(f"{base}/channels/{canal['id']}/connect", headers=headers, timeout=30).raise_for_status()
print(f"{base}/webhooks/instagram/{canal['id']}")
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
		Type: types.ChannelTypeInstagram,
		Name: "Instagram Loja",
		Config: map[string]string{
			"instagram_id": "17841400000000000",
			"app_id":       "1234567890123456",
			"verify_token": "um-token-aleatorio-longo",
		},
		Credentials: map[string]string{
			"access_token":   "EAAxxxxxxxx",
			"app_secret":     "APP_SECRET_DA_META",
			"webhook_secret": "um-segredo-longo",
		},
		WebhookURL: "https://app.suaempresa.com/webhooks/linktor",
	})
	if err != nil {
		log.Fatal(err)
	}
	if _, err := client.Channels.Connect(ctx, canal.ID); err != nil {
		log.Fatal(err)
	}
	fmt.Printf("%s/api/v1/webhooks/instagram/%s\n", os.Getenv("LINKTOR_URL"), canal.ID)
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
                .type(Channel.ChannelType.INSTAGRAM)
                .name("Instagram Loja")
                .config(Map.of(
                        "instagram_id", "17841400000000000",
                        "app_id", "1234567890123456",
                        "verify_token", "um-token-aleatorio-longo"))
                .credentials(Map.of(
                        "access_token", "EAAxxxxxxxx",
                        "app_secret", "APP_SECRET_DA_META",
                        "webhook_secret", "um-segredo-longo"))
                .webhookUrl("https://app.suaempresa.com/webhooks/linktor")
                .build());

client.channels.connect(canal.getId());
System.out.println(System.getenv("LINKTOR_URL") + "/api/v1/webhooks/instagram/" + canal.getId());
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
    type = "instagram",
    name = "Instagram Loja",
    config = new Dictionary<string, string>
    {
        ["instagram_id"] = "17841400000000000",
        ["app_id"] = "1234567890123456",
        ["verify_token"] = "um-token-aleatorio-longo",
    },
    credentials = new Dictionary<string, string>
    {
        ["access_token"] = "EAAxxxxxxxx",
        ["app_secret"] = "APP_SECRET_DA_META",
        ["webhook_secret"] = "um-segredo-longo",
    },
    webhook_url = "https://app.suaempresa.com/webhooks/linktor",
});
resposta.EnsureSuccessStatusCode();

var corpo = await resposta.Content.ReadFromJsonAsync<JsonElement>();
var id = corpo.GetProperty("data").GetProperty("id").GetString();

(await http.PostAsync($"channels/{id}/connect", null)).EnsureSuccessStatusCode();
Console.WriteLine($"{http.BaseAddress}webhooks/instagram/{id}");
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
    'type' => 'instagram',
    'name' => 'Instagram Loja',
    'config' => [
        'instagram_id' => '17841400000000000',
        'app_id'       => '1234567890123456',
        'verify_token' => 'um-token-aleatorio-longo',
    ],
    'credentials' => [
        'access_token'   => 'EAAxxxxxxxx',
        'app_secret'     => 'APP_SECRET_DA_META',
        'webhook_secret' => 'um-segredo-longo',
    ],
    'webhook_url' => 'https://app.suaempresa.com/webhooks/linktor',
]]);

$canal = json_decode((string) $resposta->getBody(), true)['data'];
$http->post("channels/{$canal['id']}/connect");

echo rtrim(getenv('LINKTOR_URL'), '/') . "/api/v1/webhooks/instagram/{$canal['id']}", PHP_EOL;
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
        ("instagram_id".to_string(), "17841400000000000".to_string()),
        ("app_id".to_string(), "1234567890123456".to_string()),
        ("verify_token".to_string(), "um-token-aleatorio-longo".to_string()),
    ]);
    let credentials = HashMap::from([
        ("access_token".to_string(), "EAAxxxxxxxx".to_string()),
        ("app_secret".to_string(), "APP_SECRET_DA_META".to_string()),
        ("webhook_secret".to_string(), "um-segredo-longo".to_string()),
    ]);

    let canal = client
        .channels()
        .create(
            CreateChannelInput::new("Instagram Loja", ChannelType::Instagram)
                .config(config)
                .credentials(credentials)
                .webhook_url("https://app.suaempresa.com/webhooks/linktor"),
        )
        .await?;

    client.channels().connect(&canal.id).await?;
    println!("{}/api/v1/webhooks/instagram/{}", servidor, canal.id);
    Ok(())
}
```

</TabItem>
</Tabs>

:::warning Ao atualizar, reenvie o `config` inteiro
`PUT /channels/CHANNEL_ID` **substitui** `config` por completo; só `credentials` é mesclado (valor
vazio ou `__redacted__` mantém o segredo guardado). Um update que mande só parte do `config` apaga
`instagram_id` e `verify_token`.
:::

Para testar credenciais sem criar o canal: `POST /channels/test-instagram` com `config` e
`credentials` no corpo. Informando `channel_id`, os campos em branco são completados com o que está
guardado. Mais em [API de canais](../api/canais.md).

## Enviar e receber

### Enviar uma mensagem

`POST /api/v1/messages/send` (scope `messages:send`). O destinatário (`to`) é o **IGSID** — o id do
usuário no escopo da sua conta do Instagram. Você o obtém em `data.message.senderId` de uma mensagem
recebida; não é possível enviar para um nome de usuário nem iniciar conversa com quem nunca escreveu.

<Tabs groupId="linguagem">
<TabItem value="curl" label="cURL">

```bash
curl -X POST "$LINKTOR_URL/api/v1/messages/send" \
  -H "X-API-Key: $LINKTOR_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "channel_id": "CHANNEL_ID",
    "to": "6543210987654321",
    "content_type": "text",
    "text": "Sim, temos em estoque. Posso reservar para você?"
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

const res: any = await client.messages.sendText('CHANNEL_ID', '6543210987654321', 'Sim, temos em estoque. Posso reservar para você?');

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
        "to": "6543210987654321",
        "content_type": "text",
        "text": "Sim, temos em estoque. Posso reservar para você?",
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
		"to":           "6543210987654321",
		"content_type": "text",
		"text":         "Sim, temos em estoque. Posso reservar para você?",
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
        "to", "6543210987654321",
        "content_type", "text",
        "text", "Sim, temos em estoque. Posso reservar para você?"));

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
    to = "6543210987654321",
    content_type = "text",
    text = "Sim, temos em estoque. Posso reservar para você?",
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
    'to'           => '6543210987654321',
    'content_type' => 'text',
    'text'         => 'Sim, temos em estoque. Posso reservar para você?',
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
            "to": "6543210987654321",
            "content_type": "text",
            "text": "Sim, temos em estoque. Posso reservar para você?"
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
  "to": "6543210987654321",
  "content_type": "image",
  "text": "Este é o modelo que você pediu",
  "attachments": [
    { "url": "https://app.suaempresa.com/arquivos/produto.jpg", "type": "image", "mime_type": "image/jpeg" }
  ]
}
```

O texto não vira legenda: ele é enviado como uma mensagem de texto e a mídia vem em seguida. Corpo
completo e idempotência em [Enviar mensagens](../api/enviar-mensagens.md).

### O que sua aplicação recebe

Com `webhook_url` e `credentials.webhook_secret` definidos no canal, cada mensagem recebida gera:

```json
{
  "id": "evt_8e3f1a20-received",
  "type": "message.received",
  "timestamp": "2026-11-24T14:03:11Z",
  "tenantId": "3f6f0c1e-9a51-4f0e-b0a5-6b6f6d2c1a11",
  "environment": "production",
  "data": {
    "message": {
      "id": "8e3f1a20-4b5c-4d6e-9f70-1a2b3c4d5e6f",
      "direction": "inbound",
      "contentType": "text",
      "content": { "text": "Oi! Esse produto ainda está disponível?" },
      "senderId": "6543210987654321",
      "senderType": "contact",
      "metadata": {
        "senderName": "Maria Souza",
        "externalId": "aWdfZAG1faXRlbToxOklHTWVzc2FnZAUlEOjE3ODQx"
      }
    },
    "conversationId": "b3a9d2c4-55e1-4f0a-8a44-0f6f1f1f7a21",
    "contactId": "0c2b7f7e-3f1d-4d57-a0ad-0f9e5f1a2b33",
    "channelId": "CHANNEL_ID",
    "channelType": "instagram"
  }
}
```

- `senderName` é buscado no perfil na primeira mensagem do contato: o nome de exibição ou, na falta
  dele, `@usuario`. Em modo de desenvolvimento a Meta só expõe o perfil de quem tem papel no app; nesse
  caso o contato nasce sem nome.
- Mídia chega em `content.media.url`, apontando para o endereço da Meta. Só o **primeiro** anexo de
  cada mensagem é considerado.
- Resposta a story chega com o texto digitado (ou `[story reply]` se vier vazia); menção em story chega
  como `[mentioned you in a story]`, com o story como anexo.
- Uma reação chega como uma mensagem de texto cujo conteúdo é o emoji.

Para o status de envio só existem `message.sent` e `message.failed`. O evento de leitura da Meta
(`messaging_seen`) é assinado, mas hoje não é convertido em `message.read`. Assinatura dos eventos
(`X-Linktor-Signature`, `X-Linktor-Timestamp`) e reentrega em
[Receber eventos](../api/receber-eventos.md).

## Limites e particularidades

- **Janela de 24 horas.** A Meta só aceita mensagens até 24h depois da última mensagem do usuário.
  O Linktor **não bloqueia** o envio fora da janela neste canal: a mensagem vai e a Meta recusa,
  gerando `message.failed`.
- **Fora da janela.** Por padrão toda mensagem sai como resposta (`messaging_type: RESPONSE`). Para
  usar uma tag da Meta, envie em `metadata` as chaves `messaging_type` (`MESSAGE_TAG`) e `message_tag`
  — por exemplo `HUMAN_AGENT`, que estende a resposta humana para até 7 dias e exige permissão
  específica aprovada no app.
- **Quem inicia é o cliente.** Não há como abrir conversa por telefone, e-mail ou nome de usuário.
- **Tamanhos declarados pelo canal.** Texto de até 1000 caracteres; mídia de até 8 MB; JPEG, PNG,
  MP4 de vídeo e MP4 de áudio.
- **Sem botões.** Mensagens interativas com opções (de bots e fluxos) falham neste canal com
  `unsupported content kind`. Em fluxos que atendem o Instagram, apresente as opções em texto.
- **Erros.** Erro definitivo da Meta não é reenviado; limite de taxa (429), erro 5xx e falha de rede
  são reenviados.
- **Token.** Tokens de login expiram; prefira o de usuário do sistema.

## Solução de problemas

| Sintoma | Causa | O que fazer |
|---|---|---|
| **Testar Conexão** falha com erro `190` | Token inválido, expirado ou de outro app | Gere um token novo; confira as permissões |
| **Testar Conexão** falha com erro `100` | `instagram_id` errado (nome de usuário no lugar do id) ou token sem acesso à conta | Use o id numérico da conta |
| **Conectar** falha com `failed to connect channel` | A Meta recusou a assinatura dos eventos: falta permissão ou a conta não está vinculada | Confira `instagram_manage_messages` e `pages_manage_metadata`, e o vínculo com a Página |
| A Meta recusa **Verificar e salvar** | Token de verificação diferente ou URL com id errado | Copie os dois valores da aba **Webhook** |
| Webhook responde 400 `Invalid webhook signature` | **Segredo do App** vazio ou de outro app | Preencha com o segredo do app que entrega o webhook |
| Canal conectado, mas nenhuma mensagem chega | Conta não assinada nos eventos, ou **Permitir acesso às mensagens** desligado no aplicativo | Clique em **Conectar** de novo; ligue a opção no Instagram |
| Só chegam mensagens de quem tem papel no app | App em modo de desenvolvimento | Peça acesso avançado e publique o app |
| `message.failed` logo depois do envio | Fora da janela de 24h, ou destinatário inválido | Responda dentro da janela; use o `senderId` recebido como `to` |
| `message.failed` com `unsupported content kind` | Mensagem com botões | Envie as opções como texto |
| Contato aparece sem nome | A Meta não expôs o perfil | Normal em modo de desenvolvimento; o nome é buscado de novo enquanto o contato estiver sem nome |
| Popup de login abre e a tela não avança | `BASE_URL` em origem diferente da do painel | Use a aba **Manual** |
| `POST /messages/send` responde `channel is not connected` | Canal criado e não conectado | Clique em **Conectar** |

Logs por canal em [Observabilidade](../painel/observabilidade.md). O mesmo app e a mesma Página
atendem também o [Facebook Messenger](./facebook.md).
