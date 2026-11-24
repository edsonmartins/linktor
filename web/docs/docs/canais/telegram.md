---
sidebar_position: 6
title: Telegram
description: Conecte um bot do Telegram ao Linktor — criação no BotFather, registro automático do webhook, tipos de mensagem suportados, API e solução de problemas.
---

# Telegram

O canal Telegram liga um **bot** (criado no BotFather) ao Linktor. Você informa o token do bot, clica
em **Conectar** e o próprio Linktor registra o webhook no Telegram (`setWebhook`) — não há nada para
configurar manualmente do lado do Telegram além de criar o bot.

Valor de `type` na API: **`telegram`**.

| Recurso | Receber | Enviar |
|---|---|---|
| Texto (até 4096 caracteres) | Sim | Sim |
| Imagem, vídeo, áudio, documento | Sim | Sim (por URL) |
| Mensagem de voz, vídeo circular (video note) | Sim (viram `audio` / `video`) | Não |
| Figurinha (sticker) | Sim (vira `image`) | Não |
| Localização, contato | Sim | Não |
| Botões inline | Sim (o toque chega como `interactive`) | Sim, a partir de bots e fluxos |
| Modelos de mensagem (templates) | — | Não existem no Telegram |
| Grupos e supergrupos | Sim | Veja [Grupos](#grupos) |
| Canais de transmissão (broadcast) | Ignorados | Não |
| Edição de mensagem pelo contato | Sim (atualiza a mensagem já gravada) | Não |
| Confirmação de entrega e leitura | Não (a Bot API não informa) | — |

## Antes de começar

- Uma conta no Telegram para falar com o [@BotFather](https://t.me/BotFather).
- O servidor do Linktor acessível pela internet em **HTTPS**, com a variável de ambiente `BASE_URL`
  apontando para esse endereço público (por exemplo `https://linktor.suaempresa.com`). O Telegram só
  entrega atualizações em HTTPS, nas portas 443, 80, 88 ou 8443. Sem `BASE_URL` o **Conectar** falha
  com `BASE_URL is not configured: Telegram needs a public webhook URL`.
- Para integrar por API: uma chave com os scopes `channels:write` e `messages:send`
  (veja [Chaves de API](../painel/chaves-de-api.md)).

## Configurar na origem

No Telegram, tudo acontece numa conversa com o BotFather. Os comandos abaixo são estáveis, mas o
texto das respostas do BotFather muda de vez em quando.

1. Abra uma conversa com **@BotFather** e envie `/newbot`.
2. Informe o **nome de exibição** do bot (ex.: `Atendimento Sua Empresa`).
3. Informe o **nome de usuário**, que precisa terminar em `bot` (ex.: `suaempresa_atendimento_bot`).
4. O BotFather responde com o **token**, no formato `123456789:ABCdefGHIjklmNOPQrstUVwxyz`. Guarde-o:
   quem tem o token controla o bot.
5. Opcional: `/setdescription`, `/setabouttext`, `/setuserpic` e `/setcommands` para dar cara ao bot.
6. Se o bot vai atender em **grupos**, envie `/setprivacy`, escolha o bot e selecione **Disable**. Com
   o modo de privacidade ligado (o padrão), o Telegram só entrega ao bot comandos, menções e
   respostas diretas — as demais mensagens do grupo nunca chegam ao Linktor.

:::warning Um bot, um webhook
O Telegram aceita **um único** webhook por bot. Ao conectar o canal, o Linktor substitui qualquer
webhook anterior. Não use o mesmo token em dois canais, em dois ambientes ou em outro sistema ao
mesmo tempo: o último a registrar fica com as mensagens.
:::

## Configurar no Linktor

1. No painel, abra **Canais** e, em **Adicionar Novo Canal**, clique em **Telegram**.
2. Na aba **Credenciais**, preencha os campos da tabela abaixo.
3. Clique em **Criar Canal**. O canal aparece em **Canais Ativos** como desconectado.
4. No cartão do canal, abra o menu de ações e clique em **Conectar**. É neste momento que o Linktor
   valida o token (`getMe`), gera o segredo do webhook e chama `setWebhook`. Se der certo, aparece
   **Canal conectado**.

| Campo na tela | Chave | Obrigatório | Observação |
|---|---|---|---|
| **Nome do Canal** | `name` | Sim | Nome interno, só para identificar o canal no painel. |
| **Token do Bot** | `credentials.bot_token` | Sim | O token do BotFather. Na edição o campo abre vazio; deixe em branco para manter o atual. |
| **Nome de Usuário do Bot** | `config.bot_name` | Não | Sem `@`. Se ficar vazio, é preenchido no **Conectar** com o `username` devolvido pelo Telegram. |
| — (gerado pelo Linktor) | `credentials.secret_token` | — | 64 caracteres hexadecimais, criado no primeiro **Conectar** e reaproveitado nos seguintes. |

A aba **Webhook** mostra a **URL do Webhook** apenas para consulta; a aba **Guia** repete o passo a
passo do BotFather.

:::note Sobre o botão Testar Conexão
**Testar Conexão** só confere se o token foi preenchido — ele não consulta o Telegram. A validação
real do token acontece no **Conectar**; um token errado aparece ali como
`failed to connect channel: invalid bot token: Unauthorized`.
:::

O **Desconectar** (e a exclusão do canal) chama `deleteWebhook` no Telegram; as atualizações
pendentes ficam guardadas pelo Telegram até o próximo **Conectar**.

## Webhook

O Linktor registra esta URL no Telegram, junto com o `secret_token`:

```text
https://linktor.suaempresa.com/api/v1/webhooks/telegram/CHANNEL_ID
```

- **Método:** `POST`, corpo JSON (`Update` da Bot API).
- **Autenticação:** o Telegram devolve o segredo no header `X-Telegram-Bot-Api-Secret-Token`. O
  Linktor compara com `credentials.secret_token` e responde `401 invalid secret token` se for
  diferente.
- Em produção (`LINKTOR_SERVER_MODE=release`, ou `LINKTOR_REQUIRE_WEBHOOK_SECRETS=true`), um canal **sem**
  segredo gravado recusa tudo com `401 webhook secret not configured`. Na prática isso só acontece
  com canais que nunca passaram pelo **Conectar**.
- O Linktor responde `200 {"status":"ok"}` mesmo quando o processamento interno falha, para o
  Telegram não entrar em reenvio contínuo.

Para conferir o que o Telegram tem registrado (rode de uma máquina sua, nunca cole o token em
ferramentas de terceiros):

```bash
curl "https://api.telegram.org/bot<SEU_TOKEN>/getWebhookInfo"
```

O campo `url` deve ser a URL acima; `last_error_message` mostra a última falha de entrega.

## Criar o canal pela API

Criar não basta: o canal nasce `disconnected` e só passa a receber e enviar depois do
`POST /channels/CHANNEL_ID/connect`, que é quem registra o webhook.

<Tabs groupId="linguagem">
<TabItem value="curl" label="cURL">

```bash
# 1. Cria o canal (nasce "disconnected")
curl -X POST "$LINKTOR_URL/api/v1/channels" \
  -H "X-API-Key: $LINKTOR_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "type": "telegram",
    "name": "Atendimento Telegram",
    "config": {
      "bot_name": "suaempresa_atendimento_bot"
    },
    "credentials": {
      "bot_token": "123456789:ABCdefGHIjklmNOPQrstUVwxyz",
      "webhook_secret": "um-segredo-longo-e-aleatorio"
    },
    "webhook_url": "https://app.suaempresa.com/webhooks/linktor"
  }'

# 2. Conecta — use o "id" devolvido em data.id
curl -X POST "$LINKTOR_URL/api/v1/channels/$CHANNEL_ID/connect" \
  -H "X-API-Key: $LINKTOR_API_KEY"
```

</TabItem>
<TabItem value="typescript" label="TypeScript">

```typescript
import { createClient } from '@linktor/sdk';

const linktor = createClient({
  baseUrl: `${process.env.LINKTOR_URL}/api/v1`,
  apiKey: process.env.LINKTOR_API_KEY,
});

const channel = await linktor.channels.create({
  "type": "telegram",
  "name": "Atendimento Telegram",
  "config": {
    "bot_name": "suaempresa_atendimento_bot"
  },
  "credentials": {
    "bot_token": "123456789:ABCdefGHIjklmNOPQrstUVwxyz",
    "webhook_secret": "um-segredo-longo-e-aleatorio"
  },
  "webhook_url": "https://app.suaempresa.com/webhooks/linktor"
});

await linktor.channels.connect(channel.id);
console.log(channel.id);
```

</TabItem>
<TabItem value="python" label="Python">

```python
import os
from linktor import LinktorClient

linktor = LinktorClient(
    base_url=os.environ["LINKTOR_URL"] + "/api/v1",
    api_key=os.environ["LINKTOR_API_KEY"],
)

channel = linktor.channels.create(
    name="Atendimento Telegram",
    type="telegram",
    config={
        "bot_name": "suaempresa_atendimento_bot"
    },
    credentials={
        "bot_token": "123456789:ABCdefGHIjklmNOPQrstUVwxyz",
        "webhook_secret": "um-segredo-longo-e-aleatorio"
    },
    webhook_url="https://app.suaempresa.com/webhooks/linktor",
)

linktor.channels.connect(channel.id)
print(channel.id)
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

	channel, err := client.Channels.Create(ctx, &types.CreateChannelInput{
		Name: "Atendimento Telegram",
		Type: types.ChannelType("telegram"),
		Config: map[string]string{
			"bot_name": "suaempresa_atendimento_bot",
		},
		Credentials: map[string]string{
			"bot_token": "123456789:ABCdefGHIjklmNOPQrstUVwxyz",
			"webhook_secret": "um-segredo-longo-e-aleatorio",
		},
		WebhookURL: "https://app.suaempresa.com/webhooks/linktor",
	})
	if err != nil {
		log.Fatal(err)
	}

	if _, err := client.Channels.Connect(ctx, channel.ID); err != nil {
		log.Fatal(err)
	}
	fmt.Println(channel.ID)
}
```

</TabItem>
<TabItem value="java" label="Java">

```java
import io.linktor.LinktorClient;
import io.linktor.types.Channel;
import java.util.Map;

LinktorClient linktor = LinktorClient.builder()
    .baseUrl(System.getenv("LINKTOR_URL") + "/api/v1")
    .apiKey(System.getenv("LINKTOR_API_KEY"))
    .build();

Channel.ChannelModel channel = linktor.channels.create(
    Channel.CreateChannelInput.builder()
        .type(Channel.ChannelType.TELEGRAM)
        .name("Atendimento Telegram")
        .config(Map.of(
            "bot_name", "suaempresa_atendimento_bot"))
        .credentials(Map.of(
            "bot_token", "123456789:ABCdefGHIjklmNOPQrstUVwxyz",
            "webhook_secret", "um-segredo-longo-e-aleatorio"))
        .webhookUrl("https://app.suaempresa.com/webhooks/linktor")
        .build());

linktor.channels.connect(channel.getId());
System.out.println(channel.getId());
```

</TabItem>
<TabItem value="csharp" label="C#">

```csharp
using System.Net.Http;
using System.Text;
using System.Text.Json;

var baseUrl = Environment.GetEnvironmentVariable("LINKTOR_URL") + "/api/v1";
using var http = new HttpClient();
http.DefaultRequestHeaders.Add("X-API-Key", Environment.GetEnvironmentVariable("LINKTOR_API_KEY"));

var body = """
{
  "type": "telegram",
  "name": "Atendimento Telegram",
  "config": {
    "bot_name": "suaempresa_atendimento_bot"
  },
  "credentials": {
    "bot_token": "123456789:ABCdefGHIjklmNOPQrstUVwxyz",
    "webhook_secret": "um-segredo-longo-e-aleatorio"
  },
  "webhook_url": "https://app.suaempresa.com/webhooks/linktor"
}
""";

var resp = await http.PostAsync($"{baseUrl}/channels",
    new StringContent(body, Encoding.UTF8, "application/json"));
resp.EnsureSuccessStatusCode();

using var doc = JsonDocument.Parse(await resp.Content.ReadAsStringAsync());
var channelId = doc.RootElement.GetProperty("data").GetProperty("id").GetString();

(await http.PostAsync($"{baseUrl}/channels/{channelId}/connect", null)).EnsureSuccessStatusCode();
Console.WriteLine(channelId);
```

</TabItem>
<TabItem value="php" label="PHP">

```php
<?php
require 'vendor/autoload.php';

$base = getenv('LINKTOR_URL') . '/api/v1';
$http = new GuzzleHttp\Client([
    'headers' => ['X-API-Key' => getenv('LINKTOR_API_KEY')],
]);

$resp = $http->post("$base/channels", ['json' => [
    'type' => 'telegram',
    'name' => 'Atendimento Telegram',
    'config' => [
        'bot_name' => 'suaempresa_atendimento_bot',
    ],
    'credentials' => [
        'bot_token' => '123456789:ABCdefGHIjklmNOPQrstUVwxyz',
        'webhook_secret' => 'um-segredo-longo-e-aleatorio',
    ],
    'webhook_url' => 'https://app.suaempresa.com/webhooks/linktor',
]]);
$channelId = json_decode((string) $resp->getBody(), true)['data']['id'];

$http->post("$base/channels/$channelId/connect");
echo $channelId, PHP_EOL;
```

</TabItem>
<TabItem value="rust" label="Rust">

```rust
use linktor::{ChannelType, CreateChannelInput, LinktorClient};
use std::collections::HashMap;

fn to_map(pairs: &[(&str, &str)]) -> HashMap<String, String> {
    pairs.iter().map(|(k, v)| (k.to_string(), v.to_string())).collect()
}

#[tokio::main]
async fn main() -> Result<(), Box<dyn std::error::Error>> {
    let client = LinktorClient::builder()
        .base_url(format!("{}/api/v1", std::env::var("LINKTOR_URL")?))
        .api_key(std::env::var("LINKTOR_API_KEY")?)
        .build()?;

    let input = CreateChannelInput::new("Atendimento Telegram", ChannelType::Telegram)
        .config(to_map(&[
            ("bot_name", "suaempresa_atendimento_bot"),
        ]))
        .credentials(to_map(&[
            ("bot_token", "123456789:ABCdefGHIjklmNOPQrstUVwxyz"),
            ("webhook_secret", "um-segredo-longo-e-aleatorio"),
        ]))
        .webhook_url("https://app.suaempresa.com/webhooks/linktor");

    let channel = client.channels().create(input).await?;
    client.channels().connect(&channel.id).await?;
    println!("{}", channel.id);
    Ok(())
}
```

</TabItem>
</Tabs>

`webhook_url` e `credentials.webhook_secret` são o endereço e o segredo da **sua aplicação** (o
destino dos eventos do Linktor) — não têm relação com o webhook do Telegram. A tela do canal
Telegram não tem campos para eles; defina-os pela API, na criação ou depois:

```bash
curl -X PUT "$LINKTOR_URL/api/v1/channels/$CHANNEL_ID" \
  -H "X-API-Key: $LINKTOR_API_KEY" -H "Content-Type: application/json" \
  -d '{
    "webhook_url": "https://app.suaempresa.com/webhooks/linktor",
    "credentials": { "webhook_secret": "um-segredo-longo-e-aleatorio" }
  }'
```

:::warning PUT substitui o config
No `PUT /channels/CHANNEL_ID`, o objeto `config`, quando enviado, **substitui** o anterior por
inteiro; só `credentials` é mesclado (valor vazio mantém o que está guardado). Para alterar uma chave
de `config`, reenvie o `config` completo — ou omita `config`, como no exemplo acima.
:::

## Enviar e receber

### Enviar

No Telegram, `to` é o **`chat_id` numérico** da conversa. Numa conversa privada ele é igual ao id do
usuário, que chega em `data.message.senderId` no evento `message.received`. Não funciona com
`@usuario` nem com telefone.

:::warning O contato precisa falar primeiro
Um bot não pode iniciar conversa: o usuário tem de abrir o bot e tocar em **Iniciar** (`/start`) ao
menos uma vez. Antes disso o Telegram recusa o envio com `403 Forbidden: bot can't initiate
conversation with a user` e a mensagem termina em `message.failed`.
:::

<Tabs groupId="linguagem">
<TabItem value="curl" label="cURL">

```bash
curl -X POST "$LINKTOR_URL/api/v1/messages/send" \
  -H "X-API-Key: $LINKTOR_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "channel_id": "'"$CHANNEL_ID"'",
    "to": "987654321",
    "text": "Olá! Seu pedido saiu para entrega."
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
    "channel_id": process.env.CHANNEL_ID,
    "to": "987654321",
    "text": "Olá! Seu pedido saiu para entrega."
  }),
});

if (!resp.ok) throw new Error(`HTTP ${resp.status}: ${await resp.text()}`);
const { data } = await resp.json();
console.log(data.id, data.status); // status "queued"
```

</TabItem>
<TabItem value="python" label="Python">

```python
import os
import requests

resp = requests.post(
    os.environ["LINKTOR_URL"] + "/api/v1/messages/send",
    headers={"X-API-Key": os.environ["LINKTOR_API_KEY"]},
    json={
        "channel_id": os.environ["CHANNEL_ID"],
        "to": "987654321",
        "text": "Olá! Seu pedido saiu para entrega."
    },
)
resp.raise_for_status()
data = resp.json()["data"]
print(data["id"], data["status"])  # status "queued"
```

</TabItem>
<TabItem value="go" label="Go">

```go
package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
)

func main() {
	body, _ := json.Marshal(map[string]any{
		"channel_id": os.Getenv("CHANNEL_ID"),
		"to": "987654321",
		"text": "Olá! Seu pedido saiu para entrega.",
	})

	req, _ := http.NewRequest(http.MethodPost,
		os.Getenv("LINKTOR_URL")+"/api/v1/messages/send", bytes.NewReader(body))
	req.Header.Set("X-API-Key", os.Getenv("LINKTOR_API_KEY"))
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Fatal(err)
	}
	defer resp.Body.Close()

	out, _ := io.ReadAll(resp.Body)
	fmt.Println(resp.StatusCode, string(out)) // 202 + data.status "queued"
}
```

</TabItem>
<TabItem value="java" label="Java">

```java
import java.net.URI;
import java.net.http.HttpClient;
import java.net.http.HttpRequest;
import java.net.http.HttpResponse;

String body = """
    {
      "channel_id": "%s",
      "to": "987654321",
      "text": "Olá! Seu pedido saiu para entrega."
    }
    """.formatted(System.getenv("CHANNEL_ID"));

HttpRequest request = HttpRequest.newBuilder()
    .uri(URI.create(System.getenv("LINKTOR_URL") + "/api/v1/messages/send"))
    .header("X-API-Key", System.getenv("LINKTOR_API_KEY"))
    .header("Content-Type", "application/json")
    .POST(HttpRequest.BodyPublishers.ofString(body))
    .build();

HttpResponse<String> response = HttpClient.newHttpClient()
    .send(request, HttpResponse.BodyHandlers.ofString());
System.out.println(response.statusCode() + " " + response.body()); // 202
```

</TabItem>
<TabItem value="csharp" label="C#">

```csharp
using System.Net.Http;
using System.Net.Http.Json;

using var http = new HttpClient();
http.DefaultRequestHeaders.Add("X-API-Key", Environment.GetEnvironmentVariable("LINKTOR_API_KEY"));

var payload = new
{
    channel_id = Environment.GetEnvironmentVariable("CHANNEL_ID"),
    to = "987654321",
    text = "Olá! Seu pedido saiu para entrega.",
};

var resp = await http.PostAsJsonAsync(
    Environment.GetEnvironmentVariable("LINKTOR_URL") + "/api/v1/messages/send", payload);
resp.EnsureSuccessStatusCode(); // 202
Console.WriteLine(await resp.Content.ReadAsStringAsync());
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
        'channel_id' => getenv('CHANNEL_ID'),
        'to' => '987654321',
        'text' => 'Olá! Seu pedido saiu para entrega.',
    ],
]);

$data = json_decode((string) $resp->getBody(), true)['data'];
echo $data['id'], ' ', $data['status'], PHP_EOL; // status "queued"
```

</TabItem>
<TabItem value="rust" label="Rust">

```rust
// Cargo.toml: reqwest = { version = "0.12", features = ["json"] }, serde_json, tokio
use serde_json::json;

#[tokio::main]
async fn main() -> Result<(), Box<dyn std::error::Error>> {
    let resp = reqwest::Client::new()
        .post(format!("{}/api/v1/messages/send", std::env::var("LINKTOR_URL")?))
        .header("X-API-Key", std::env::var("LINKTOR_API_KEY")?)
        .json(&json!({
            "channel_id": std::env::var("CHANNEL_ID")?,
            "to": "987654321",
            "text": "Olá! Seu pedido saiu para entrega."
        }))
        .send()
        .await?
        .error_for_status()?;

    println!("{}", resp.text().await?); // 202 + data.status "queued"
    Ok(())
}
```

</TabItem>
</Tabs>

A resposta é `202` com `data.status = "queued"`: a mensagem foi aceita e enfileirada. O desfecho
chega depois no webhook da sua aplicação como `message.sent` ou `message.failed`.

Para mídia, use `content_type` `image`, `video`, `audio` ou `document` e informe a URL em
`attachments` (o Telegram baixa o arquivo dessa URL, que precisa ser pública); `text` vira a legenda:

```bash
curl -X POST "$LINKTOR_URL/api/v1/messages/send" \
  -H "X-API-Key: $LINKTOR_API_KEY" -H "Content-Type: application/json" \
  -d '{
    "channel_id": "'"$CHANNEL_ID"'",
    "to": "987654321",
    "content_type": "image",
    "text": "Seu comprovante",
    "attachments": [
      { "url": "https://app.suaempresa.com/arquivos/comprovante.jpg", "mime_type": "image/jpeg" }
    ]
  }'
```

Mais opções (idempotência, correlação, envio dentro de uma conversa) em
[Enviar mensagens](../api/enviar-mensagens.md).

### Receber

Cada mensagem do contato gera um evento `message.received`, entregue por `POST` no `webhook_url` do
canal com os headers `X-Linktor-Signature` e `X-Linktor-Timestamp` (assinatura HMAC-SHA256 de
`timestamp.corpo` com o `webhook_secret`; detalhes em [Receber eventos](../api/receber-eventos.md)).

```json
{
  "id": "evt_8d2f6f0a-93c1-4b7e-a7f5-0c9e1d2b3a4c_received",
  "type": "message.received",
  "timestamp": "2026-11-24T14:03:27Z",
  "tenantId": "0b6f7c1e-2a44-4f6b-9d0e-6a1b2c3d4e5f",
  "environment": "production",
  "data": {
    "message": {
      "id": "8d2f6f0a-93c1-4b7e-a7f5-0c9e1d2b3a4c",
      "direction": "inbound",
      "contentType": "text",
      "content": { "text": "Olá, quero saber do meu pedido" },
      "senderId": "987654321",
      "senderType": "contact",
      "metadata": { "senderName": "Maria Souza", "externalId": "4127" }
    },
    "conversationId": "c1a5d3e2-7b84-4a60-8e55-2f1d9b0c7a66",
    "contactId": "7e9a4b10-5c2d-4f3e-b8a1-6d0f2e3c4b5a",
    "channelId": "CHANNEL_ID",
    "channelType": "telegram"
  }
}
```

- `senderId` é o id numérico do usuário no Telegram; `metadata.externalId` é o `message_id` do
  Telegram; `metadata.senderName` é nome + sobrenome do perfil.
- Em mensagens com mídia, `content.media` traz `url`, `mimeType`, `filename` e `size`. O Telegram
  entrega apenas um `file_id`; o Linktor baixa o arquivo (`getFile`) e o grava no armazenamento de
  mídia do servidor, e é essa URL que chega no evento.
- **Figurinha** chega como `contentType: "image"` (`image/webp`), com o emoji correspondente no
  texto. **Voz** chega como `audio`; **vídeo circular**, como `video`.
- **Localização** chega como `contentType: "location"` com o texto `latitude,longitude`.
  **Contato** chega como `contentType: "contact"` com o telefone no texto.
- **Toque em botão inline** chega como `contentType: "interactive"`, com o identificador da opção
  (`callback_data`) em `content.text`.

## Limites e particularidades

### Grupos {#grupos}

- Mensagens de grupos e supergrupos são recebidas. O **contato** continua sendo a pessoa que
  escreveu (`senderId`); o grupo vem no bloco `data.group`, cujo `id` é o `chat_id` do grupo (um
  número negativo). Postagens de **canais de transmissão** são ignoradas.
- Quando um administrador publica como anônimo, o Telegram não informa o autor; o Linktor usa o id
  do próprio chat como remetente.
- Eventos de serviço do grupo (alguém entrou, título alterado) não geram mensagem.
- Lembre-se do modo de privacidade do BotFather (`/setprivacy`): com ele ligado, quase nada do grupo
  chega.

### Outros comportamentos

- **Edição:** quando o contato edita uma mensagem, o Linktor atualiza o conteúdo da mensagem já
  gravada (e a marca como editada) em vez de criar outra.
- **Respostas citadas:** uma mensagem recebida que responde a outra traz o id da citada nos metadados
  internos. No **envio**, o caminho de saída do Telegram hoje **não** aplica `reply_to_id`: a
  mensagem sai sem citação.
- **Botões inline:** são gerados por [bots](../painel/bots.md) e [fluxos](../painel/fluxos.md) (uma
  opção por linha). O envio direto (`POST /messages/send`) aceita apenas texto e mídia. O Linktor não
  chama `answerCallbackQuery`, então o aplicativo do Telegram pode exibir o indicador de
  carregamento no botão por alguns segundos após o toque.
- **Formatação:** o texto é enviado sem `parse_mode`; Markdown e HTML aparecem literalmente.
- **Tamanho de mídia:** os limites são os da Bot API pública — em geral, download de até 20 MB pelo
  bot (mídia recebida) e envio por URL de até 5 MB para fotos e 20 MB para os demais tipos. Quando o
  download de um arquivo recebido falha, `content.media.url` traz só o `file_id` do Telegram, não uma
  URL.
- **Status:** o Telegram não informa entrega nem leitura. Você recebe `message.sent` quando a Bot
  API aceita a mensagem e `message.failed` quando recusa; `message.delivered` e `message.read` não
  existem neste canal.
- **Erros permanentes e temporários:** respostas `4xx` da Bot API (exceto `429`) encerram a
  mensagem como falha imediatamente; `429` e `5xx` são tentados de novo pela fila.

## Solução de problemas

| Sintoma | Causa provável | O que fazer |
|---|---|---|
| **Conectar** falha com `BASE_URL is not configured` | Variável `BASE_URL` ausente no servidor | Defina `BASE_URL` com o endereço público HTTPS e reinicie. Veja [Configuração](../implantar/configuracao.md). |
| **Conectar** falha com `invalid bot token: Unauthorized` | Token errado, incompleto ou revogado no BotFather | Copie o token de novo (`/token` no BotFather) e salve o canal. |
| **Conectar** falha com `setWebhook failed: ...` | URL pública sem HTTPS válido, porta não aceita ou domínio não resolvível | Confira o certificado e a porta; a mensagem após os dois-pontos é a descrição do Telegram. |
| Canal conectado, mas nada chega | Outro sistema registrou webhook com o mesmo token | Rode `getWebhookInfo`; se a `url` não for a do Linktor, clique em **Desconectar** e **Conectar**. |
| Chega em conversa privada, mas não em grupo | Modo de privacidade do bot ligado | `/setprivacy` → **Disable** no BotFather; remova e adicione o bot ao grupo. |
| Webhook responde `401 invalid secret token` | Webhook registrado manualmente com outro segredo | Não chame `setWebhook` por fora; refaça o **Conectar**. |
| Envio termina em `message.failed` com `Forbidden` | O usuário nunca iniciou o bot ou o bloqueou | O contato precisa enviar `/start`. Não há como contornar. |
| Envio falha com `invalid telegram chat_id` | `to` não é numérico (`@usuario`, telefone) | Use o `senderId` recebido no `message.received`. |
| Mídia recebida com `content.media.url` que não é uma URL | Arquivo acima do limite de download da Bot API ou armazenamento de mídia indisponível | Verifique o armazenamento em [Observabilidade](../painel/observabilidade.md) e o tamanho do arquivo. |
| O webhook é chamado, mas nada aparece no painel | Canal desconectado ou desabilitado: o recebimento é descartado (`channel is not active`) | Clique em **Conectar** e confira se o canal está habilitado. |
| Nenhum evento chega na sua aplicação | Canal sem `webhook_url` ou sem `credentials.webhook_secret` | Sem segredo o Linktor descarta a entrega. Defina os dois pelo `PUT /channels/CHANNEL_ID`. |

Veja também: [Visão geral dos canais](visao-geral.md) · [Canais no painel](../painel/canais.md) ·
[API de canais](../api/canais.md).
