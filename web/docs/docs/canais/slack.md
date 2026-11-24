---
sidebar_position: 12
title: Slack
description: Conectar um app do Slack ao Linktor pela Events API — scopes do bot, Request URL, signing secret, campos do canal, envio e recebimento.
---

# Slack

## Resumo

O canal Slack liga um **Slack App** (bot) ao Linktor. As mensagens chegam pela **Events API** do
Slack (um `POST` público, assinado, para o Linktor) e saem pela **Web API** (`chat.postMessage` e
upload de arquivos) com o token do bot. Socket Mode não é usado: o Linktor precisa estar
acessível pela internet.

- Valor de `type` na API: `slack`
- Como conecta: token do bot (`xoxb-…`) + signing secret
- Endereço do contato: o **id do canal do Slack** onde a conversa acontece (`D…` para mensagem
  direta, `C…` para canal público, `G…` para canal privado)

| Recurso | Enviar | Receber | Observação |
|---|---|---|---|
| Texto | Sim | Sim | O texto vai como está; o Slack interpreta a formatação `mrkdwn` |
| Imagem, áudio, vídeo, documento | Sim | Sim | Enviado como arquivo nativo; se o upload falhar, vai o link |
| Localização | Não | Não | |
| Botões e listas | Não | Não | Mensagem com botões falha no envio — veja [Limites](#limites) |
| Modelos (templates) | Vira texto | — | O Slack não tem modelos; os parâmetros são unidos em texto |
| Canais e grupos do Slack | Sim | Sim | Cada canal do Slack vira **um** contato no Linktor |
| Reação | Não | Não | |
| Confirmação de entrega e leitura | Não | — | Só `message.sent` e `message.failed` |

## Antes de começar

- Permissão para criar e instalar apps no workspace do Slack (ou um administrador que aprove).
- O Linktor publicado em HTTPS com endereço público — o Slack não chama `localhost`.
- No Linktor, um usuário do painel ou uma chave de API com os scopes `channels:write` (criar o
  canal) e `messages:send` (enviar). Veja [Chaves de API](../painel/chaves-de-api.md).

## Configurar na origem

O passo a passo é feito em [api.slack.com/apps](https://api.slack.com/apps). Os nomes de menu
abaixo são os do console do Slack e mudam com alguma frequência.

1. Clique em **Create New App** → **From scratch**, dê um nome e escolha o workspace.
2. Em **OAuth & Permissions** → **Scopes** → **Bot Token Scopes**, adicione:

   | Scope | Para quê |
   |---|---|
   | `chat:write` | Enviar mensagens (`chat.postMessage`) |
   | `files:write` | Enviar imagem, áudio, vídeo e documento como arquivo |
   | `files:read` | Baixar os arquivos que os usuários mandam |
   | `im:history` | Receber mensagens diretas ao bot |
   | `channels:history` | Receber mensagens de canais públicos onde o bot está |
   | `groups:history` | Receber mensagens de canais privados onde o bot está |
   | `mpim:history` | Receber mensagens de conversas diretas em grupo |

   Adicione só os `*:history` dos lugares onde o bot vai atender.
3. Em **App Home** → **Show Tabs**, ligue **Messages Tab** e marque a opção que permite aos
   usuários enviar mensagens pela aba de mensagens. Sem isso ninguém consegue escrever ao bot em
   mensagem direta.
4. Em **OAuth & Permissions**, clique em **Install to Workspace** e autorize. Copie o **Bot User
   OAuth Token** (começa com `xoxb-`).
5. Em **Basic Information** → **App Credentials**, copie o **Signing Secret** e o **App ID**.
6. Crie o canal no Linktor (próxima seção) para obter a URL do webhook. Volte ao Slack, abra
   **Event Subscriptions**, ligue **Enable Events** e cole a URL em **Request URL**. O Slack faz
   uma chamada de verificação na hora; o campo precisa ficar com **Verified**.
7. Ainda em **Event Subscriptions** → **Subscribe to bot events**, adicione os eventos
   correspondentes aos scopes: `message.im`, `message.channels`, `message.groups`, `message.mpim`.
   Clique em **Save Changes**.
8. Se o Slack avisar que os scopes mudaram, reinstale o app (**Reinstall to Workspace**). O token
   `xoxb-` continua o mesmo.
9. Para atender em um canal, convide o bot: `/invite @nome-do-bot`.

:::note Ordem dos passos
A Request URL contém o id do canal do Linktor, que só existe depois de o canal ser criado. Crie o
canal primeiro com o token e o signing secret; depois volte ao Slack para colar a URL.
:::

## Configurar no Linktor

1. Abra **Canais** e, em **Adicionar Novo Canal**, escolha **Slack**.
2. Preencha os campos da tabela abaixo.
3. Clique em **Testar Conexão**. O Linktor chama `auth.test` no Slack com o token digitado.
4. Clique em **Criar Canal**.
5. Reabra o canal e copie a **URL do Webhook**. Cole-a no Slack (passo 6 da seção anterior).
6. No cartão do canal, clique em **Conectar**. Enquanto o canal não estiver **Conectado** e
   habilitado, o envio é recusado.

| Rótulo na tela | Chave | Onde fica | Obrigatório |
|---|---|---|---|
| **Nome do Canal** | `name` | campo do canal | Sim |
| **Token do Bot** | `bot_token` | `credentials` | Sim |
| **Signing Secret** | `signing_secret` | `credentials` | Sim na tela; veja a nota |
| **App ID** | `app_id` | `credentials` | Não (referência) |
| **ID do Usuário Bot** | `bot_user_id` | `credentials` | Não (filtro de eco) |
| **URL do Consumidor** (em **Webhook de saída**) | `webhook_url` | campo do canal | Não |
| **Segredo de Assinatura** (em **Webhook de saída**) | `webhook_secret` | `credentials` | Só se houver URL do Consumidor |

Tudo o que é do Slack fica em `credentials`, cifrado no banco. A API nunca devolve `bot_token`,
`signing_secret` nem `webhook_secret`; ao editar, esses campos abrem vazios e **deixá-los em
branco mantém o valor guardado**. `app_id` e `bot_user_id` não são segredo e voltam em `config`
na consulta de um canal, só para preencher o formulário.

:::note Signing secret
O formulário exige o signing secret na criação. Pela API ele é opcional, mas sem ele o Linktor
não tem como validar a origem das chamadas: em produção (`release`, ou com
`LINKTOR_REQUIRE_WEBHOOK_SECRETS=true`) o webhook responde `401` e nenhuma mensagem entra.
:::

:::note Filtro de eventos do webhook de saída
A tela do Slack não tem a lista **Eventos a entregar**. Para limitar os eventos enviados ao seu
sistema, grave `config.webhook_events` pela API (lista separada por vírgula, por exemplo
`message.received,message.failed`). Salvar o canal pela tela depois disso apaga esse valor, porque
o formulário envia `config` vazio.
:::

### Testar pela API

`POST /api/v1/channels/test` valida o token contra o Slack antes de gravar qualquer coisa:

```bash
curl -X POST "$LINKTOR_URL/api/v1/channels/test" \
  -H "X-API-Key: $LINKTOR_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{"type": "slack", "credentials": {"bot_token": "xoxb-seu-token"}}'
```

```json
{
  "success": true,
  "data": {
    "status": "ok",
    "type": "slack",
    "valid": true,
    "message": "connected to Slack (team \"Sua Empresa\", bot \"atendimento\")"
  }
}
```

Para testar um canal já salvo sem redigitar o token, envie `"channel_id": "CHANNEL_ID"` no corpo:
o Linktor completa com as credenciais guardadas. `POST /api/v1/channels/test-slack` é o mesmo
teste com o tipo fixo. O teste verifica o token; o signing secret só é exercitado quando o Slack
chama a Request URL.

## Webhook

URL que o Slack deve chamar (campo **Request URL** em **Event Subscriptions**):

```text
https://linktor.suaempresa.com/api/v1/webhooks/slack/CHANNEL_ID
```

Só `POST`. O que o Linktor faz em cada chamada:

1. Localiza o canal pelo id da URL (`404` se não existir).
2. Valida a assinatura quando há `signing_secret`: recalcula
   `v0=HMAC-SHA256(signing_secret, "v0:" + timestamp + ":" + corpo)` e compara com o header
   `X-Slack-Signature`. O `timestamp` vem de `X-Slack-Request-Timestamp` e é recusado se estiver
   a mais de **5 minutos** do relógio do servidor. Falha → `401 invalid signature`.
3. Se o corpo for `url_verification`, responde `200` com o valor de `challenge` em texto puro.
4. Se for `event_callback` com evento `message`, descarta o que veio de bot (`bot_id` presente,
   subtipo `bot_message` ou autor igual a `bot_user_id`) e publica o restante na fila de entrada.
5. Responde `200 {"status":"ok"}` — inclusive quando o evento é ignorado, para o Slack não
   reenviar.

O Slack reenvia eventos quando não recebe `200` em poucos segundos. Reenvios com o mesmo corpo
são barrados pela [deduplicação de webhooks](./visao-geral.md#deduplicacao) e respondidos com
`200 {"status":"duplicate"}`.

## Criar o canal pela API

`POST /api/v1/channels` com o scope `channels:write`. Os SDKs não acrescentam `/api/v1` à base:
informe o endereço completo. Onde o SDK da linguagem ainda não cobre este tipo de canal, o exemplo
usa o cliente HTTP nativo.

<Tabs groupId="linguagem">
<TabItem value="curl" label="cURL">

```bash
curl -X POST "$LINKTOR_URL/api/v1/channels" \
  -H "X-API-Key: $LINKTOR_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
  "type": "slack",
  "name": "Suporte no Slack",
  "credentials": {
    "bot_token": "xoxb-seu-token",
    "signing_secret": "seu_signing_secret",
    "app_id": "A0123456789",
    "bot_user_id": "U0123456789",
    "webhook_secret": "um_segredo_longo_e_aleatorio"
  },
  "webhook_url": "https://seu-sistema.exemplo.com/webhooks/linktor"
}'
```

</TabItem>
<TabItem value="typescript" label="TypeScript">

```typescript
import { LinktorClient } from '@linktor/sdk';

// A base precisa incluir /api/v1: o SDK não acrescenta o prefixo.
const linktor = new LinktorClient({
  baseUrl: `${process.env.LINKTOR_URL}/api/v1`,
  apiKey: process.env.LINKTOR_API_KEY,
});

const channel = await linktor.channels.create({
  type: 'slack',
  name: 'Suporte no Slack',
  credentials: {
    bot_token: 'xoxb-seu-token',
    signing_secret: 'seu_signing_secret',
    app_id: 'A0123456789',
    bot_user_id: 'U0123456789',
    webhook_secret: 'um_segredo_longo_e_aleatorio',
  },
  webhook_url: 'https://seu-sistema.exemplo.com/webhooks/linktor',
});

console.log(channel.id, channel.connection_status);
```

</TabItem>
<TabItem value="python" label="Python">

```python
import os
import requests

resp = requests.post(
    f"{os.environ['LINKTOR_URL']}/api/v1/channels",
    headers={"X-API-Key": os.environ["LINKTOR_API_KEY"]},
    json={
        "type": "slack",
        "name": "Suporte no Slack",
        "credentials": {
            "bot_token": "xoxb-seu-token",
            "signing_secret": "seu_signing_secret",
            "app_id": "A0123456789",
            "bot_user_id": "U0123456789",
            "webhook_secret": "um_segredo_longo_e_aleatorio",
        },
        "webhook_url": "https://seu-sistema.exemplo.com/webhooks/linktor",
    },
    timeout=30,
)
resp.raise_for_status()
channel = resp.json()["data"]
print(channel["id"], channel["connection_status"])
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
	// A base precisa incluir /api/v1: o SDK não acrescenta o prefixo.
	client := linktor.NewClient(
		linktor.WithBaseURL(os.Getenv("LINKTOR_URL")+"/api/v1"),
		linktor.WithAPIKey(os.Getenv("LINKTOR_API_KEY")),
	)

	channel, err := client.Channels.Create(context.Background(), &types.CreateChannelInput{
		Type: types.ChannelTypeSlack,
		Name: "Suporte no Slack",
		Credentials: map[string]string{
			"bot_token": "xoxb-seu-token",
			"signing_secret": "seu_signing_secret",
			"app_id": "A0123456789",
			"bot_user_id": "U0123456789",
			"webhook_secret": "um_segredo_longo_e_aleatorio",
		},
		WebhookURL: "https://seu-sistema.exemplo.com/webhooks/linktor",
	})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(channel.ID, channel.ConnectionStatus)
}
```

</TabItem>
<TabItem value="java" label="Java">

```java
import io.linktor.LinktorClient;
import io.linktor.types.Channel;
import java.util.Map;

public class CriarCanal {
    public static void main(String[] args) throws Exception {
        // A base precisa incluir /api/v1: o SDK não acrescenta o prefixo.
        LinktorClient client = LinktorClient.builder()
            .baseUrl(System.getenv("LINKTOR_URL") + "/api/v1")
            .apiKey(System.getenv("LINKTOR_API_KEY"))
            .build();

        Channel.ChannelModel channel = client.channels.create(
            Channel.CreateChannelInput.builder()
                .type(Channel.ChannelType.SLACK)
                .name("Suporte no Slack")
                .credentials(Map.of(
                    "bot_token", "xoxb-seu-token",
                    "signing_secret", "seu_signing_secret",
                    "app_id", "A0123456789",
                    "bot_user_id", "U0123456789",
                    "webhook_secret", "um_segredo_longo_e_aleatorio"))
                .webhookUrl("https://seu-sistema.exemplo.com/webhooks/linktor")
                .build());

        System.out.println(channel.getId() + " " + channel.getConnectionStatus());
    }
}
```

</TabItem>
<TabItem value="csharp" label="C#">

```csharp
using System.Net.Http.Json;
using System.Text.Json;

var baseUrl = Environment.GetEnvironmentVariable("LINKTOR_URL");
using var http = new HttpClient();
http.DefaultRequestHeaders.Add("X-API-Key", Environment.GetEnvironmentVariable("LINKTOR_API_KEY"));

var resp = await http.PostAsJsonAsync($"{baseUrl}/api/v1/channels", new
{
    type = "slack",
    name = "Suporte no Slack",
    credentials = new Dictionary<string, string>
    {
        ["bot_token"] = "xoxb-seu-token",
        ["signing_secret"] = "seu_signing_secret",
        ["app_id"] = "A0123456789",
        ["bot_user_id"] = "U0123456789",
        ["webhook_secret"] = "um_segredo_longo_e_aleatorio",
    },
    webhook_url = "https://seu-sistema.exemplo.com/webhooks/linktor",
});
resp.EnsureSuccessStatusCode();

using var doc = JsonDocument.Parse(await resp.Content.ReadAsStringAsync());
var channel = doc.RootElement.GetProperty("data");
Console.WriteLine($"{channel.GetProperty("id")} {channel.GetProperty("connection_status")}");
```

</TabItem>
<TabItem value="php" label="PHP">

```php
<?php
require 'vendor/autoload.php';

$http = new GuzzleHttp\Client();

$resp = $http->post(getenv('LINKTOR_URL') . '/api/v1/channels', [
    'headers' => ['X-API-Key' => getenv('LINKTOR_API_KEY')],
    'json' => [
        'type' => 'slack',
        'name' => 'Suporte no Slack',
        'credentials' => [
            'bot_token' => 'xoxb-seu-token',
            'signing_secret' => 'seu_signing_secret',
            'app_id' => 'A0123456789',
            'bot_user_id' => 'U0123456789',
            'webhook_secret' => 'um_segredo_longo_e_aleatorio',
        ],
        'webhook_url' => 'https://seu-sistema.exemplo.com/webhooks/linktor',
    ],
]);

$channel = json_decode((string) $resp->getBody(), true)['data'];
echo $channel['id'], ' ', $channel['connection_status'], PHP_EOL;
```

</TabItem>
<TabItem value="rust" label="Rust">

```rust
use std::collections::HashMap;

use linktor::{ChannelType, CreateChannelInput, LinktorClient};

#[tokio::main]
async fn main() -> Result<(), Box<dyn std::error::Error>> {
    // A base precisa incluir /api/v1: o SDK não acrescenta o prefixo.
    let client = LinktorClient::builder()
        .base_url(format!("{}/api/v1", std::env::var("LINKTOR_URL")?))
        .api_key(std::env::var("LINKTOR_API_KEY")?)
        .build()?;

    let channel = client
        .channels()
        .create(
            CreateChannelInput::new("Suporte no Slack", ChannelType::Slack)
                .credentials(HashMap::from([
                    ("bot_token".to_string(), "xoxb-seu-token".to_string()),
                    ("signing_secret".to_string(), "seu_signing_secret".to_string()),
                    ("app_id".to_string(), "A0123456789".to_string()),
                    ("bot_user_id".to_string(), "U0123456789".to_string()),
                    ("webhook_secret".to_string(), "um_segredo_longo_e_aleatorio".to_string()),
                ]))
                .webhook_url("https://seu-sistema.exemplo.com/webhooks/linktor"),
        )
        .await?;

    println!("{} {:?}", channel.id, channel.connection_status);
    Ok(())
}
```

</TabItem>
</Tabs>

Resposta `201`:

```json
{
  "success": true,
  "data": {
    "id": "CHANNEL_ID",
    "tenant_id": "…",
    "type": "slack",
    "name": "Suporte no Slack",
    "enabled": true,
    "connection_status": "disconnected",
    "environment": "production",
    "webhook_url": "https://seu-sistema.exemplo.com/webhooks/linktor",
    "created_at": "2026-11-24T14:00:00Z",
    "updated_at": "2026-11-24T14:00:00Z"
  }
}
```

O canal nasce habilitado e **desconectado**. Conecte-o antes de enviar:

```bash
curl -X POST "$LINKTOR_URL/api/v1/channels/CHANNEL_ID/connect" \
  -H "X-API-Key: $LINKTOR_API_KEY"
```

Para o Slack, conectar apenas marca o canal como `connected`; não há chamada ao provedor nesse
passo — use o teste de conexão para validar o token.

Para trocar um segredo, envie `PUT /api/v1/channels/CHANNEL_ID` só com a chave nova em
`credentials`; as demais credenciais são mantidas. `config`, ao contrário, é **substituído por
inteiro** quando enviado.

## Enviar e receber

### Enviar

`POST /api/v1/messages/send` com o scope `messages:send`. Em `to` vai o **id do canal do Slack**
(não o nome nem o id do usuário). Para responder a quem escreveu, use o `senderId` que chegou no
evento `message.received`.

<Tabs groupId="linguagem">
<TabItem value="curl" label="cURL">

```bash
curl -X POST "$LINKTOR_URL/api/v1/messages/send" \
  -H "X-API-Key: $LINKTOR_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
  "channel_id": "CHANNEL_ID",
  "to": "D0123ABCDEF",
  "text": "Olá! Recebemos seu chamado e já estamos analisando."
}'
```

</TabItem>
<TabItem value="typescript" label="TypeScript">

```typescript
const resp = await fetch(`${process.env.LINKTOR_URL}/api/v1/messages/send`, {
  method: 'POST',
  headers: {
    'X-API-Key': process.env.LINKTOR_API_KEY!,
    'Content-Type': 'application/json',
  },
  body: JSON.stringify({
    channel_id: 'CHANNEL_ID',
    to: 'D0123ABCDEF',
    text: 'Olá! Recebemos seu chamado e já estamos analisando.',
  }),
});
if (!resp.ok) throw new Error(`Linktor respondeu ${resp.status}: ${await resp.text()}`);

const { data } = await resp.json();
console.log(data.id, data.conversation_id, data.status); // status: "queued"
```

</TabItem>
<TabItem value="python" label="Python">

```python
import os
import requests

resp = requests.post(
    f"{os.environ['LINKTOR_URL']}/api/v1/messages/send",
    headers={"X-API-Key": os.environ["LINKTOR_API_KEY"]},
    json={
        "channel_id": "CHANNEL_ID",
        "to": "D0123ABCDEF",
        "text": "Olá! Recebemos seu chamado e já estamos analisando.",
    },
    timeout=30,
)
resp.raise_for_status()
data = resp.json()["data"]
print(data["id"], data["conversation_id"], data["status"])  # status: "queued"
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
	payload, _ := json.Marshal(map[string]string{
		"channel_id": "CHANNEL_ID",
		"to":         "D0123ABCDEF",
		"text":       "Olá! Recebemos seu chamado e já estamos analisando.",
	})

	req, _ := http.NewRequest(http.MethodPost,
		os.Getenv("LINKTOR_URL")+"/api/v1/messages/send", bytes.NewReader(payload))
	req.Header.Set("X-API-Key", os.Getenv("LINKTOR_API_KEY"))
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusAccepted {
		log.Fatalf("Linktor respondeu %d", resp.StatusCode)
	}

	var out struct {
		Data struct {
			ID             string `json:"id"`
			ConversationID string `json:"conversation_id"`
			Status         string `json:"status"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		log.Fatal(err)
	}
	fmt.Println(out.Data.ID, out.Data.ConversationID, out.Data.Status) // status: "queued"
}
```

</TabItem>
<TabItem value="java" label="Java">

```java
import java.net.URI;
import java.net.http.HttpClient;
import java.net.http.HttpRequest;
import java.net.http.HttpResponse;

public class EnviarMensagem {
    public static void main(String[] args) throws Exception {
        String body = """
            {
              "channel_id": "CHANNEL_ID",
              "to": "D0123ABCDEF",
              "text": "Olá! Recebemos seu chamado e já estamos analisando."
            }
            """;

        HttpRequest request = HttpRequest.newBuilder()
            .uri(URI.create(System.getenv("LINKTOR_URL") + "/api/v1/messages/send"))
            .header("X-API-Key", System.getenv("LINKTOR_API_KEY"))
            .header("Content-Type", "application/json")
            .POST(HttpRequest.BodyPublishers.ofString(body))
            .build();

        HttpResponse<String> response = HttpClient.newHttpClient()
            .send(request, HttpResponse.BodyHandlers.ofString());

        // 202 + {"success":true,"data":{"id":...,"conversation_id":...,"status":"queued"}}
        System.out.println(response.statusCode() + " " + response.body());
    }
}
```

</TabItem>
<TabItem value="csharp" label="C#">

```csharp
using System.Net.Http.Json;
using System.Text.Json;

var baseUrl = Environment.GetEnvironmentVariable("LINKTOR_URL");
using var http = new HttpClient();
http.DefaultRequestHeaders.Add("X-API-Key", Environment.GetEnvironmentVariable("LINKTOR_API_KEY"));

var resp = await http.PostAsJsonAsync($"{baseUrl}/api/v1/messages/send", new
{
    channel_id = "CHANNEL_ID",
    to = "D0123ABCDEF",
    text = "Olá! Recebemos seu chamado e já estamos analisando.",
});
resp.EnsureSuccessStatusCode();

using var doc = JsonDocument.Parse(await resp.Content.ReadAsStringAsync());
var data = doc.RootElement.GetProperty("data");
Console.WriteLine($"{data.GetProperty("id")} {data.GetProperty("status")}"); // status: "queued"
```

</TabItem>
<TabItem value="php" label="PHP">

```php
<?php
require 'vendor/autoload.php';

$http = new GuzzleHttp\Client();

$resp = $http->post(getenv('LINKTOR_URL') . '/api/v1/messages/send', [
    'headers' => ['X-API-Key' => getenv('LINKTOR_API_KEY')],
    'json' => [
        'channel_id' => 'CHANNEL_ID',
        'to' => 'D0123ABCDEF',
        'text' => 'Olá! Recebemos seu chamado e já estamos analisando.',
    ],
]);

$data = json_decode((string) $resp->getBody(), true)['data'];
echo $data['id'], ' ', $data['conversation_id'], ' ', $data['status'], PHP_EOL; // status: "queued"
```

</TabItem>
<TabItem value="rust" label="Rust">

```rust
use serde_json::{json, Value};

#[tokio::main]
async fn main() -> Result<(), Box<dyn std::error::Error>> {
    let url = format!("{}/api/v1/messages/send", std::env::var("LINKTOR_URL")?);

    let resp = reqwest::Client::new()
        .post(url)
        .header("X-API-Key", std::env::var("LINKTOR_API_KEY")?)
        .json(&json!({
            "channel_id": "CHANNEL_ID",
            "to": "D0123ABCDEF",
            "text": "Olá! Recebemos seu chamado e já estamos analisando."
        }))
        .send()
        .await?
        .error_for_status()?;

    let body: Value = resp.json().await?;
    println!("{} {}", body["data"]["id"], body["data"]["status"]); // status: "queued"
    Ok(())
}
```

</TabItem>
</Tabs>

Resposta `202` — a mensagem foi gravada e enfileirada, ainda não entregue:

```json
{
  "success": true,
  "data": {
    "id": "0b6f1d0e-3a52-4b1e-9d0e-6c1f5f0f8a11",
    "conversation_id": "5d1c1a1e-8f0a-4a39-b7a0-3c2c9d7e6b10",
    "channel_id": "CHANNEL_ID",
    "status": "queued"
  }
}
```

O resultado da entrega chega depois, no webhook de saída, como `message.sent` ou
`message.failed`. O Linktor cria (ou reaproveita) o contato e a conversa daquele canal do Slack.

Para enviar um arquivo, informe `content_type` e `attachments` com uma URL que o Linktor consiga
baixar (até 64 MiB):

```bash
curl -X POST "$LINKTOR_URL/api/v1/messages/send" \
  -H "X-API-Key: $LINKTOR_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "channel_id": "CHANNEL_ID",
    "to": "D0123ABCDEF",
    "content_type": "document",
    "text": "Segue o contrato",
    "attachments": [
      {
        "url": "https://arquivos.suaempresa.com/contrato.pdf",
        "type": "document",
        "filename": "contrato.pdf",
        "mime_type": "application/pdf"
      }
    ]
  }'
```

O Linktor baixa o arquivo e o publica no Slack como upload nativo, com o texto como comentário.
Se o download ou o upload falhar, a mensagem ainda é entregue: o texto seguido do link.

Para não enviar duas vezes em caso de repetição, informe `metadata.idempotency_key` (até 255
caracteres, único por empresa): a segunda chamada devolve a mensagem original.

### Receber

Com **URL do Consumidor** e **Segredo de Assinatura** preenchidos, cada mensagem recebida no Slack
é entregue ao seu sistema como `message.received`:

```json
{
  "id": "evt_8c7e2f0a-4b1d-4e0b-9a55-2f6d3c1b7e90_received",
  "type": "message.received",
  "timestamp": "2026-11-24T14:03:27.512Z",
  "tenantId": "3f2b7c9e-1d4a-4f6b-8e2a-9c0d1e2f3a4b",
  "environment": "production",
  "data": {
    "message": {
      "id": "8c7e2f0a-4b1d-4e0b-9a55-2f6d3c1b7e90",
      "direction": "inbound",
      "contentType": "text",
      "content": {
        "text": "Preciso de ajuda com o pedido 1234"
      },
      "senderId": "D0123ABCDEF",
      "senderType": "contact",
      "metadata": {
        "externalId": "1732456999.000200"
      }
    },
    "conversationId": "5d1c1a1e-8f0a-4a39-b7a0-3c2c9d7e6b10",
    "contactId": "a1b2c3d4-e5f6-4a7b-8c9d-0e1f2a3b4c5d",
    "channelId": "CHANNEL_ID",
    "channelType": "slack"
  }
}
```

| Campo | No Slack |
|---|---|
| `data.message.senderId` | Id do canal do Slack onde a mensagem foi escrita. É o valor a usar em `to` para responder |
| `data.message.metadata.externalId` | O `ts` da mensagem no Slack |
| `data.message.contentType` | `text`, ou `image` / `video` / `audio` / `document` quando há arquivo (pelo MIME) |
| `data.message.content.media` | Presente quando há arquivo: `url`, `mimeType`, `filename`, `size`. Só o primeiro arquivo é enviado no evento |

O id do usuário do Slack que escreveu não faz parte do evento: no Slack, o contato do Linktor é o
canal, não a pessoa.

A entrega é assinada com `X-Linktor-Signature` e `X-Linktor-Timestamp`; como validar está em
[Receber eventos](../api/receber-eventos.md). **Sem segredo de assinatura o Linktor não entrega
nada**, mesmo com a URL preenchida.

## Limites e particularidades {#limites}

- **Um canal do Slack é um contato.** A conversa no Linktor é identificada pelo id do canal do
  Slack. Em mensagem direta isso equivale a uma pessoa; em um canal com várias pessoas, todas
  falam na mesma conversa e aparecem como o mesmo contato.
- **Respostas não vão para a thread.** O Linktor recebe mensagens escritas dentro de threads, mas
  responde sempre na raiz do canal.
- **Edição e exclusão de mensagens não são tratadas.** Só o evento `message` é consumido, e apenas
  o filtro de bot é aplicado; não há atualização da mensagem original no Linktor quando alguém
  edita ou apaga no Slack.
- **Mensagens com botões falham.** Blocos, botões, menus e comandos de barra não são suportados.
  Uma mensagem de bot ou fluxo que leve respostas rápidas é recusada pelo adaptador com erro
  permanente (`unsupported content kind`) e vira `message.failed`. Em fluxos que atendem o Slack,
  use só texto.
- **Modelos viram texto.** Um envio de modelo entrega os parâmetros do corpo unidos por espaço
  (ou o nome do modelo, se não houver parâmetros).
- **Arquivos recebidos.** O `url_private` do Slack exige o token do bot. Quando o servidor tem
  armazenamento de mídia configurado, o Linktor baixa o arquivo (até 64 MiB) e entrega uma URL
  própria; sem armazenamento, o evento leva a URL do Slack, que seu sistema não consegue abrir
  sem o token. Veja [Configuração](../implantar/configuracao.md).
- **Sem recibos.** O Slack não informa entrega nem leitura; o Linktor emite `message.sent`
  quando a Web API aceita a mensagem e `message.failed` quando recusa.
- **Erros que não são repetidos:** `channel_not_found`, `not_in_channel`, `is_archived`,
  `msg_too_long`, `no_text`, `invalid_auth`, `token_revoked`, `account_inactive`, `not_authed`,
  `restricted_action` e qualquer outro `4xx`. Limite de taxa (`ratelimited`, `429`) e erros `5xx`
  são tentados de novo pela fila.
- **Sandbox não se aplica.** A proteção de canal sandbox existe só para WhatsApp; um canal Slack
  criado com `environment: "sandbox"` tem todo envio bloqueado. Veja
  [Sandbox](../painel/sandbox.md).

## Solução de problemas

| Sintoma | Causa provável | O que fazer |
|---|---|---|
| O Slack não marca a Request URL como **Verified** | Id do canal errado na URL (`404`), signing secret diferente do gravado (`401`) ou servidor sem acesso público | Copie a URL pelo botão do canal; regrave o **Signing Secret**; teste a URL de fora da rede |
| `401 slack signing_secret not configured` | Canal sem signing secret em servidor de produção | Edite o canal e preencha o **Signing Secret** |
| `401 invalid signature` em todas as chamadas | Segredo de outro app, ou relógio do servidor com mais de 5 minutos de diferença | Confira o segredo em **Basic Information**; sincronize o relógio (NTP) |
| O bot não recebe mensagens diretas | **Messages Tab** desligada ou falta `im:history` / evento `message.im` | Ligue a aba, adicione scope e evento e reinstale o app |
| O bot não recebe mensagens de um canal | Bot não foi convidado, ou falta `channels:history` / `groups:history` | `/invite @nome-do-bot`; adicione scope e evento |
| Envio volta `channel is not connected` | Canal criado mas não conectado | Clique em **Conectar** no cartão do canal |
| `message.failed` com `not_in_channel` ou `channel_not_found` | `to` não é um id de canal, ou o bot não está nele | Use o `senderId` recebido; convide o bot |
| Arquivo enviado chega como link | Download da URL ou upload ao Slack falhou (falta `files:write`, URL inacessível, mais de 64 MiB) | Adicione `files:write` e reinstale; confira a URL |
| Arquivo recebido vem com URL `files.slack.com` que não abre | Sem armazenamento de mídia no servidor, ou falta `files:read` | Configure o armazenamento; adicione o scope |
| As respostas do bot voltam como mensagens novas | Eco do próprio bot não reconhecido | Preencha **ID do Usuário Bot** |
| Nada chega ao seu sistema | **Segredo de Assinatura** vazio, ou evento fora de `config.webhook_events` | Preencha o segredo; revise o filtro |

Os erros de entrega e de webhook de cada canal aparecem em
[Observabilidade](../painel/observabilidade.md).
