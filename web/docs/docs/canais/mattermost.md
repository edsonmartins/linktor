---
sidebar_position: 14
title: Mattermost
description: Conectar um servidor Mattermost ao Linktor com uma conta de bot — token de acesso, conexão por WebSocket, campos do canal, envio e recebimento.
---

# Mattermost

## Resumo

O canal Mattermost liga uma **conta de bot** de um servidor Mattermost ao Linktor. Diferente dos
outros canais, **não há webhook de entrada**: é o Linktor que abre uma conexão WebSocket com o
servidor Mattermost (`/api/v4/websocket`) e fica ouvindo as mensagens. As respostas saem pela API
REST (`/api/v4/posts`) com o token do bot.

- Valor de `type` na API: `mattermost`
- Como conecta: URL do servidor + token de acesso do bot
- Endereço do contato: o **id do canal do Mattermost** onde a conversa acontece (mensagem direta
  com o bot, canal público ou privado)

| Recurso | Enviar | Receber | Observação |
|---|---|---|---|
| Texto | Sim | Sim | O texto vai como está; o Mattermost interpreta Markdown |
| Imagem, áudio, vídeo, documento | Sim | Sim | Enviado como arquivo nativo; se o upload falhar, vai o link |
| Localização | Não | Não | |
| Botões e listas | Não | Não | Mensagem com botões falha no envio — veja [Limites](#limites) |
| Modelos (templates) | Vira texto | — | Os parâmetros são unidos em texto |
| Canais com várias pessoas | Sim | Sim | Cada canal do Mattermost vira **um** contato no Linktor |
| Reação | Não | Não | |
| Confirmação de entrega e leitura | Não | — | Só `message.sent` e `message.failed` |

## Antes de começar

- Um servidor Mattermost (auto-hospedado ou em nuvem) e um usuário **System Admin** para criar a
  conta de bot.
- **Rede: o Linktor precisa alcançar o servidor Mattermost**, por HTTP(S) e por WebSocket, na URL
  informada. O sentido é Linktor → Mattermost; o Mattermost não precisa alcançar o Linktor. Em
  servidor interno, libere a saída do Linktor (firewall, VPN, DNS).
- No Linktor, um usuário do painel ou uma chave de API com os scopes `channels:write` e
  `messages:send`. Veja [Chaves de API](../painel/chaves-de-api.md).

## Configurar na origem

Feito no próprio Mattermost. Os nomes de menu variam conforme a versão.

1. Em **System Console** → **Integrations** → **Bot Accounts**, ligue **Enable Bot Account
   Creation**.
2. No menu de produtos, abra **Integrations** → **Bot Accounts** → **Add Bot Account**.
3. Informe **Username** (por exemplo `atendimento`), nome de exibição e papel **Member**. Salve.
4. O Mattermost mostra o **Token** de acesso uma única vez. Copie — é o `bot_token`. O valor a
   copiar é o token, não o **Token ID** exibido ao lado.
5. Adicione o bot à equipe e a cada canal em que ele vai atender (`/invite @atendimento` dentro do
   canal). Para conversas diretas não é preciso convite: o usuário abre uma mensagem direta com
   o bot.

O **ID do Usuário Bot** é opcional. Sem ele, o Linktor descobre o id sozinho ao conectar,
consultando `/api/v4/users/me` com o token. Se preferir informar, o mesmo endereço devolve o
valor:

```bash
curl -H "Authorization: Bearer token_de_acesso_do_bot" \
  https://mattermost.exemplo.com/api/v4/users/me
```

O campo `id` da resposta é o `bot_user_id`.

:::note Webhooks do Mattermost não são usados
O Linktor não usa *Incoming Webhooks*, *Outgoing Webhooks* nem *Slash Commands* do Mattermost.
Basta a conta de bot e o token.
:::

## Configurar no Linktor

1. Abra **Canais** e, em **Adicionar Novo Canal**, escolha **Mattermost**.
2. Preencha os campos da tabela abaixo.
3. Clique em **Testar Conexão**. O Linktor chama `/api/v4/users/me` no servidor com o token; o
   teste confirma, de uma vez, que o servidor é alcançável e que o token é válido.
4. Clique em **Criar Canal**.
5. No cartão do canal, clique em **Conectar**. É nesse momento que o Linktor abre o WebSocket e
   começa a receber. Um canal Mattermost desconectado não recebe nem envia.

| Rótulo na tela | Chave | Onde fica | Obrigatório |
|---|---|---|---|
| **Nome do Canal** | `name` | campo do canal | Sim |
| **URL do Servidor** | `base_url` | `credentials` | Sim |
| **Token do Bot** | `bot_token` | `credentials` | Sim |
| **ID do Usuário Bot** | `bot_user_id` | `credentials` | Não (descoberto ao conectar) |
| **URL do Consumidor** (em **Webhook de saída**) | `webhook_url` | campo do canal | Não |
| **Segredo de Assinatura** (em **Webhook de saída**) | `webhook_secret` | `credentials` | Só se houver URL do Consumidor |

A **URL do Servidor** é a base, sem caminho: `https://mattermost.exemplo.com`. O Linktor troca
`https://` por `wss://` (e `http://` por `ws://`) para montar o endereço do WebSocket.

Tudo fica em `credentials`, cifrado no banco — inclusive a URL do servidor, que é por canal.
`bot_token` e `webhook_secret` nunca voltam pela API; ao editar, deixá-los em branco mantém o
valor guardado. `base_url` e `bot_user_id` não são segredo e voltam em `config` na consulta de um
canal, só para preencher o formulário.

:::note Filtro de eventos do webhook de saída
A tela do Mattermost não tem a lista **Eventos a entregar**. Para limitar os eventos enviados ao
seu sistema, grave `config.webhook_events` pela API (lista separada por vírgula). Salvar o canal
pela tela depois disso apaga esse valor, porque o formulário envia `config` vazio.
:::

### Testar pela API

```bash
curl -X POST "$LINKTOR_URL/api/v1/channels/test" \
  -H "X-API-Key: $LINKTOR_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "type": "mattermost",
    "credentials": {
      "base_url": "https://mattermost.exemplo.com",
      "bot_token": "token_de_acesso_do_bot"
    }
  }'
```

```json
{
  "success": true,
  "data": {
    "status": "ok",
    "type": "mattermost",
    "valid": true,
    "message": "connected to Mattermost (bot \"atendimento\")"
  }
}
```

Para testar um canal já salvo sem redigitar o token, envie `"channel_id": "CHANNEL_ID"` no corpo.
`POST /api/v1/channels/test-mattermost` é o mesmo teste com o tipo fixo. O limite é de 10
segundos.

## Webhook

**Este canal não tem URL de webhook.** Não há nada para cadastrar no Mattermost. No lugar do
webhook, o Linktor mantém uma conexão de saída:

```text
wss://mattermost.exemplo.com/api/v4/websocket
```

Como a conexão funciona:

1. Ao **Conectar** o canal — e a cada reinício do servidor, para os canais já conectados — o
   Linktor abre o WebSocket e se autentica enviando o token do bot
   (`authentication_challenge`).
2. Se o Mattermost recusar o token, a conexão é encerrada, o erro vai para o log do servidor e o
   Linktor tenta de novo.
3. O Linktor consome apenas eventos `posted` (mensagem nova), e só depois de a autenticação ser
   confirmada.
4. Mensagens escritas pelo próprio bot são descartadas, comparando o autor com `bot_user_id`.
5. Se a conexão cair, o Linktor reconecta sozinho, com espera crescente de 1 até 30 segundos
   entre as tentativas.

Alterar as credenciais reinicia a conexão com os valores novos. Desconectar ou excluir o canal
encerra o WebSocket.

:::warning Mensagens durante uma queda de conexão
O WebSocket do Mattermost entrega eventos em tempo real e não os reenvia. O que for escrito
enquanto o Linktor estiver sem conexão (servidor reiniciando, rede fora) **não é recuperado**
depois.
:::

Como não há webhook, a [deduplicação de webhooks](./visao-geral.md#deduplicacao) não participa
deste canal.

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
  "type": "mattermost",
  "name": "Suporte no Mattermost",
  "credentials": {
    "base_url": "https://mattermost.exemplo.com",
    "bot_token": "token_de_acesso_do_bot",
    "bot_user_id": "id_do_usuario_do_bot",
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
  type: 'mattermost',
  name: 'Suporte no Mattermost',
  credentials: {
    base_url: 'https://mattermost.exemplo.com',
    bot_token: 'token_de_acesso_do_bot',
    bot_user_id: 'id_do_usuario_do_bot',
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
        "type": "mattermost",
        "name": "Suporte no Mattermost",
        "credentials": {
            "base_url": "https://mattermost.exemplo.com",
            "bot_token": "token_de_acesso_do_bot",
            "bot_user_id": "id_do_usuario_do_bot",
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
		Type: types.ChannelTypeMattermost,
		Name: "Suporte no Mattermost",
		Credentials: map[string]string{
			"base_url": "https://mattermost.exemplo.com",
			"bot_token": "token_de_acesso_do_bot",
			"bot_user_id": "id_do_usuario_do_bot",
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
                .type(Channel.ChannelType.MATTERMOST)
                .name("Suporte no Mattermost")
                .credentials(Map.of(
                    "base_url", "https://mattermost.exemplo.com",
                    "bot_token", "token_de_acesso_do_bot",
                    "bot_user_id", "id_do_usuario_do_bot",
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
    type = "mattermost",
    name = "Suporte no Mattermost",
    credentials = new Dictionary<string, string>
    {
        ["base_url"] = "https://mattermost.exemplo.com",
        ["bot_token"] = "token_de_acesso_do_bot",
        ["bot_user_id"] = "id_do_usuario_do_bot",
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
        'type' => 'mattermost',
        'name' => 'Suporte no Mattermost',
        'credentials' => [
            'base_url' => 'https://mattermost.exemplo.com',
            'bot_token' => 'token_de_acesso_do_bot',
            'bot_user_id' => 'id_do_usuario_do_bot',
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
            CreateChannelInput::new("Suporte no Mattermost", ChannelType::Mattermost)
                .credentials(HashMap::from([
                    ("base_url".to_string(), "https://mattermost.exemplo.com".to_string()),
                    ("bot_token".to_string(), "token_de_acesso_do_bot".to_string()),
                    ("bot_user_id".to_string(), "id_do_usuario_do_bot".to_string()),
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
    "type": "mattermost",
    "name": "Suporte no Mattermost",
    "enabled": true,
    "connection_status": "disconnected",
    "environment": "production",
    "webhook_url": "https://seu-sistema.exemplo.com/webhooks/linktor",
    "created_at": "2026-11-24T14:00:00Z",
    "updated_at": "2026-11-24T14:00:00Z"
  }
}
```

O canal nasce habilitado e **desconectado**, sem WebSocket aberto. Conecte-o para começar a
receber:

```bash
curl -X POST "$LINKTOR_URL/api/v1/channels/CHANNEL_ID/connect" \
  -H "X-API-Key: $LINKTOR_API_KEY"
```

:::note O que "conectado" garante
`connect` marca o canal como `connected` e dispara a abertura do WebSocket em segundo plano. A
resposta não espera a autenticação no Mattermost: um token inválido aparece no log do servidor,
não no `connection_status`. Valide o token antes com o teste de conexão.
:::

Para trocar um segredo, envie `PUT /api/v1/channels/CHANNEL_ID` só com a chave nova em
`credentials`; as demais credenciais são mantidas. `config`, ao contrário, é **substituído por
inteiro** quando enviado.

## Enviar e receber

### Enviar

`POST /api/v1/messages/send` com o scope `messages:send`. Em `to` vai o **id do canal do
Mattermost** (26 caracteres) — o `senderId` que chegou no evento `message.received`. O bot precisa
ser membro do canal.

<Tabs groupId="linguagem">
<TabItem value="curl" label="cURL">

```bash
curl -X POST "$LINKTOR_URL/api/v1/messages/send" \
  -H "X-API-Key: $LINKTOR_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
  "channel_id": "CHANNEL_ID",
  "to": "4xp9fdt77pncbef59f4k1qe83o",
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
    to: '4xp9fdt77pncbef59f4k1qe83o',
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
        "to": "4xp9fdt77pncbef59f4k1qe83o",
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
		"to":         "4xp9fdt77pncbef59f4k1qe83o",
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
              "to": "4xp9fdt77pncbef59f4k1qe83o",
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
    to = "4xp9fdt77pncbef59f4k1qe83o",
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
        'to' => '4xp9fdt77pncbef59f4k1qe83o',
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
            "to": "4xp9fdt77pncbef59f4k1qe83o",
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
`message.failed`.

Para enviar um arquivo, informe `content_type` e `attachments` com uma URL que o Linktor consiga
baixar (até 64 MiB):

```bash
curl -X POST "$LINKTOR_URL/api/v1/messages/send" \
  -H "X-API-Key: $LINKTOR_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "channel_id": "CHANNEL_ID",
    "to": "4xp9fdt77pncbef59f4k1qe83o",
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

O Linktor baixa o arquivo, envia ao Mattermost (`/api/v4/files`) e cria a mensagem com o arquivo
anexado e o texto como corpo. Se o download ou o upload falhar, a mensagem ainda é entregue: o
texto seguido do link.

Para não enviar duas vezes em caso de repetição, informe `metadata.idempotency_key` (até 255
caracteres, único por empresa): a segunda chamada devolve a mensagem original.

### Receber

Com **URL do Consumidor** e **Segredo de Assinatura** preenchidos, cada mensagem recebida no
Mattermost é entregue ao seu sistema como `message.received`:

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
      "senderId": "4xp9fdt77pncbef59f4k1qe83o",
      "senderType": "contact",
      "metadata": {
        "senderName": "maria.silva",
        "externalId": "k7qz3m9w1bnt8e5c6d2f4h0a1y"
      }
    },
    "conversationId": "5d1c1a1e-8f0a-4a39-b7a0-3c2c9d7e6b10",
    "contactId": "a1b2c3d4-e5f6-4a7b-8c9d-0e1f2a3b4c5d",
    "channelId": "CHANNEL_ID",
    "channelType": "mattermost"
  }
}
```

| Campo | No Mattermost |
|---|---|
| `data.message.senderId` | Id do canal do Mattermost onde a mensagem foi escrita. É o valor a usar em `to` para responder |
| `data.message.metadata.senderName` | Nome de usuário de quem escreveu, sem o `@` |
| `data.message.metadata.externalId` | Id do post no Mattermost |
| `data.message.contentType` | `text`, ou `image` / `video` / `audio` / `document` quando há arquivo (pelo MIME) |
| `data.message.content.media` | Presente quando há arquivo **e** o Linktor conseguiu hospedá-lo: `url`, `mimeType`, `filename`, `size`. Só o primeiro arquivo é enviado no evento |

A entrega é assinada com `X-Linktor-Signature` e `X-Linktor-Timestamp`; como validar está em
[Receber eventos](../api/receber-eventos.md). **Sem segredo de assinatura o Linktor não entrega
nada**, mesmo com a URL preenchida.

## Limites e particularidades {#limites}

- **Um canal do Mattermost é um contato.** A conversa no Linktor é identificada pelo id do canal.
  Em mensagem direta com o bot isso equivale a uma pessoa; em um canal com várias pessoas, todas
  falam na mesma conversa, e o contato leva o nome de quem escreveu primeiro.
- **O bot ouve tudo o que vê.** Toda mensagem nova em qualquer canal do qual o bot é membro vira
  mensagem no Linktor — não é preciso mencioná-lo. Adicione o bot só aos canais de atendimento.
- **Respostas não vão para a thread.** O Linktor cria sempre um post na raiz do canal.
- **Edição, exclusão e reações não são tratadas.** Só o evento `posted` é consumido.
- **Mensagens com botões falham.** Anexos interativos e botões não são suportados. Uma mensagem
  de bot ou fluxo que leve respostas rápidas é recusada pelo adaptador com erro permanente
  (`unsupported content kind`) e vira `message.failed`. Em fluxos que atendem o Mattermost, use só
  texto.
- **Arquivos recebidos dependem do armazenamento de mídia.** Os arquivos do Mattermost exigem o
  token do bot. Com armazenamento configurado no servidor, o Linktor baixa cada arquivo (até
  64 MiB) e entrega uma URL própria. Sem armazenamento, a mensagem chega com o tipo e o nome do
  arquivo, mas **sem URL**. Veja [Configuração](../implantar/configuracao.md).
- **Modelos viram texto.** Um envio de modelo entrega os parâmetros do corpo unidos por espaço
  (ou o nome do modelo, se não houver parâmetros).
- **Sem recibos.** O Linktor emite `message.sent` quando o Mattermost aceita o post e
  `message.failed` quando recusa. Erros `4xx` (exceto `429`) não são repetidos; `429` e `5xx` são
  tentados de novo pela fila.
- **Sandbox não se aplica.** A proteção de canal sandbox existe só para WhatsApp; um canal
  Mattermost criado com `environment: "sandbox"` tem todo envio bloqueado. Veja
  [Sandbox](../painel/sandbox.md).

## Solução de problemas

| Sintoma | Causa provável | O que fazer |
|---|---|---|
| **Testar Conexão** falha por tempo esgotado | O Linktor não alcança a **URL do Servidor** | Teste a URL a partir do servidor do Linktor; libere firewall, VPN ou DNS |
| **Testar Conexão** falha com `401` | Token errado, revogado, ou **Token ID** colado no lugar do token | Gere um token novo na conta de bot |
| Canal **Conectado**, mas nenhuma mensagem chega | Token recusado na autenticação do WebSocket, WebSocket bloqueado por proxy, ou bot fora do canal | Veja o log do servidor (`mattermost: … ws error`); libere o *upgrade* de WebSocket no proxy; convide o bot |
| Nenhuma mensagem chega e o canal aparece **Desconectado** | O WebSocket só é aberto para canal conectado | Clique em **Conectar** |
| As respostas do bot voltam como mensagens novas | `bot_user_id` errado | Apague o campo para o Linktor descobrir o id, ou corrija-o |
| `message.failed` com `403` | O bot não é membro do canal de destino | `/invite @atendimento` no canal |
| `message.failed` com `404` | `to` não é um id de canal | Use o `senderId` recebido |
| Arquivo enviado chega como link | Download da URL ou upload ao Mattermost falhou | Confira a URL, o limite de tamanho do servidor Mattermost e as permissões de arquivo |
| Arquivo recebido chega sem URL | Sem armazenamento de mídia no servidor | Configure o armazenamento |
| Mensagens escritas durante uma queda sumiram | O WebSocket não reenvia eventos | Não há recuperação; acompanhe a estabilidade da rede |
| Nada chega ao seu sistema | **Segredo de Assinatura** vazio, ou evento fora de `config.webhook_events` | Preencha o segredo; revise o filtro |

Os erros de entrega de cada canal aparecem em [Observabilidade](../painel/observabilidade.md).
