---
sidebar_position: 9
title: SMS
description: Conecte um número de SMS da Twilio ao Linktor — credenciais, webhook de mensagens recebidas, validação da assinatura, status callback, API e solução de problemas.
---

# SMS

O canal de SMS envia e recebe mensagens de texto por um número (ou Messaging Service) da **Twilio**.
A Twilio é o único provedor de SMS implementado.

Valor de `type` na API: **`sms`**.

| Recurso | Receber | Enviar |
|---|---|---|
| Texto | Sim | Sim |
| Mídia (MMS) | Sim — toda mídia chega classificada como `image` | Sim, por URL pública (onde a Twilio oferece MMS) |
| Botões, modelos, localização | Não | Não (um modelo é achatado em texto) |
| Grupos | Não | Não |
| Status de entrega | — | `sent`, `delivered`, `failed` (exige status callback) |
| Confirmação de leitura | — | Não existe em SMS |

## Antes de começar

- Uma conta Twilio com um **número habilitado para SMS** ou um **Messaging Service**.
- O servidor do Linktor acessível pela internet em HTTPS.
- Atenção à cobertura: a Twilio não vende número local em todos os países, e o MMS só existe em
  alguns (essencialmente EUA e Canadá). Para o Brasil, confirme com a Twilio que tipo de remetente
  está disponível para a sua conta antes de planejar o canal.
- Nos EUA, o envio de SMS por número local exige registro **A2P 10DLC**, feito num Messaging
  Service.

## Configurar na origem

Os caminhos são os do Console da Twilio no momento em que esta página foi escrita; a Twilio
reorganiza os menus com frequência.

1. Em [console.twilio.com](https://console.twilio.com), no painel inicial (**Account Info**), copie
   o **Account SID** e o **Auth Token**.
2. Em **Phone Numbers → Manage → Buy a number**, compre um número com a capacidade **SMS** — ou use
   um que já tenha.
3. Opcional: em **Messaging → Services**, crie um **Messaging Service**, adicione o número ao
   *Sender Pool* e copie o **Messaging Service SID** (começa com `MG`).
4. Crie o canal no Linktor (próxima seção) para obter a URL do webhook e volte aqui.
5. Mensagens recebidas — escolha **um** dos dois lugares:
   - **No número:** **Phone Numbers → Manage → Active numbers →** (seu número) **→ Messaging
     Configuration**. Em **A message comes in**, escolha **Webhook**, cole a URL e deixe o método
     em **HTTP POST**.
   - **No Messaging Service:** **Integration → Incoming Messages → Send a webhook**, com a mesma URL.
     Quando o número pertence a um Messaging Service, é a configuração do serviço que vale.
6. Status de entrega: no Messaging Service, em **Integration → Delivery Status Callback URL**, cole a
   mesma URL. Sem Messaging Service, veja [Status callback](#status-callback).

## Configurar no Linktor

1. No painel, abra **Canais** e, em **Adicionar Novo Canal**, clique em **SMS**.
2. Na aba **Credenciais**, preencha os campos da tabela.
3. Clique em **Criar Canal**.
4. Reabra o canal: a aba **Webhook** mostra **Webhook de Mensagens Recebidas** e **URL de Status
   Callback** (é o mesmo endereço). Copie e cadastre na Twilio.
5. No cartão do canal, abra o menu de ações e clique em **Conectar**. Para SMS, conectar apenas
   marca o canal como conectado — condição para enviar; as credenciais são exercitadas no primeiro
   envio.

| Campo na tela | Chave | Obrigatório | Observação |
|---|---|---|---|
| **Nome do Canal** | `name` | Sim | |
| **Account SID** | `config.account_sid` | Sim | Começa com `AC`. Nas respostas da API aparece mascarado. |
| **Auth Token** | `credentials.auth_token` | Sim na criação | Também é a chave que valida a assinatura dos webhooks. Na edição, em branco mantém o atual. |
| **Tipo de Remetente** | — | Sim | Só escolhe qual dos dois campos abaixo a tela mostra. |
| **Número de Telefone** | `config.phone_number` | Um dos dois | Formato E.164: `+5511999999999`. |
| **Messaging Service SID** | `config.messaging_service_sid` | Um dos dois | Se os dois existirem, o Messaging Service tem prioridade. |

Chaves aceitas apenas pela API:

| Chave | Efeito |
|---|---|
| `config.status_callback_url` | URL enviada à Twilio como `StatusCallback` em **cada** mensagem. Veja [Status callback](#status-callback). |
| `credentials.api_key_sid` + `credentials.api_key_secret` | API Key da Twilio para autenticar o envio, no lugar do Auth Token. O Auth Token continua necessário para validar os webhooks. |

:::note Sobre o botão Testar Conexão
Hoje **Testar Conexão** envia ao servidor apenas o Account SID e o Auth Token, e a validação exige
também o número ou o Messaging Service — por isso o teste acusa falha mesmo com credenciais corretas.
Não trate esse resultado como diagnóstico: salve o canal, conecte e faça um envio de teste.
:::

## Webhook

```text
https://linktor.suaempresa.com/api/v1/webhooks/twilio/CHANNEL_ID
```

`/api/v1/webhooks/sms/CHANNEL_ID` é um apelido para o mesmo tratamento. A mesma URL recebe mensagens
e callbacks de status: o Linktor trata como **mensagem recebida** tudo o que vem sem status ou com
`SmsStatus=received`, e como **status** o restante.

**Validação da assinatura.** A Twilio assina cada requisição no header `X-Twilio-Signature`. O
Linktor recalcula o HMAC-SHA1 (em Base64) sobre a URL completa da requisição seguida dos parâmetros
do formulário em ordem alfabética, usando o **Auth Token** do canal, e responde
`401 invalid signature` se não bater.

- A URL usada na conta é montada a partir de `X-Forwarded-Proto`, do header `Host` e do caminho.
  Atrás de um proxy reverso, ele precisa repassar o `Host` original e `X-Forwarded-Proto: https` —
  senão a URL recalculada difere da que a Twilio assinou.
- Cadastre na Twilio exatamente a URL exibida no painel. Se cadastrar o apelido `/sms/`, a
  assinatura continua válida, desde que seja a mesma URL nos dois lados.
- Sem `credentials.auth_token` gravado, o webhook é aceito sem validação fora de produção; com
  `LINKTOR_SERVER_MODE=release` (ou `LINKTOR_REQUIRE_WEBHOOK_SECRETS=true`) a resposta é
  `401 webhook secret not configured`.

O Linktor responde com um TwiML vazio (`text/xml`): nenhuma resposta automática é devolvida à
Twilio.

### Status callback {#status-callback}

A Twilio só avisa sobre a entrega se souber para onde enviar. Há duas formas:

- **No Messaging Service** (recomendado): preencha **Delivery Status Callback URL** com a URL do
  webhook. Vale para todas as mensagens do serviço.
- **Por mensagem:** grave `config.status_callback_url` no canal, com a mesma URL do webhook. O
  Linktor passa esse valor como `StatusCallback` em cada envio. A tela do canal mostra a URL de
  status, mas **não grava** essa chave — use a API.

Mapeamento dos status da Twilio:

| Twilio | Linktor | Evento para a sua aplicação |
|---|---|---|
| `sent` | `sent` | `message.sent` |
| `delivered` | `delivered` | `message.delivered` |
| `failed`, `undelivered` | `failed` | `message.failed` (com a mensagem de erro da Twilio) |
| `queued`, `sending`, `accepted` | `pending` | — |

## Criar o canal pela API

<Tabs groupId="linguagem">
<TabItem value="curl" label="cURL">

```bash
# 1. Cria o canal (nasce "disconnected")
curl -X POST "$LINKTOR_URL/api/v1/channels" \
  -H "X-API-Key: $LINKTOR_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "type": "sms",
    "name": "SMS Twilio",
    "config": {
      "account_sid": "ACxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx",
      "phone_number": "+5511999999999"
    },
    "credentials": {
      "auth_token": "seu-auth-token",
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
  "type": "sms",
  "name": "SMS Twilio",
  "config": {
    "account_sid": "ACxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx",
    "phone_number": "+5511999999999"
  },
  "credentials": {
    "auth_token": "seu-auth-token",
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
    name="SMS Twilio",
    type="sms",
    config={
        "account_sid": "ACxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx",
        "phone_number": "+5511999999999"
    },
    credentials={
        "auth_token": "seu-auth-token",
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
		Name: "SMS Twilio",
		Type: types.ChannelType("sms"),
		Config: map[string]string{
			"account_sid": "ACxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx",
			"phone_number": "+5511999999999",
		},
		Credentials: map[string]string{
			"auth_token": "seu-auth-token",
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
        .type(Channel.ChannelType.SMS)
        .name("SMS Twilio")
        .config(Map.of(
            "account_sid", "ACxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx",
            "phone_number", "+5511999999999"))
        .credentials(Map.of(
            "auth_token", "seu-auth-token",
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
  "type": "sms",
  "name": "SMS Twilio",
  "config": {
    "account_sid": "ACxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx",
    "phone_number": "+5511999999999"
  },
  "credentials": {
    "auth_token": "seu-auth-token",
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
    'type' => 'sms',
    'name' => 'SMS Twilio',
    'config' => [
        'account_sid' => 'ACxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx',
        'phone_number' => '+5511999999999',
    ],
    'credentials' => [
        'auth_token' => 'seu-auth-token',
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

    let input = CreateChannelInput::new("SMS Twilio", ChannelType::Sms)
        .config(to_map(&[
            ("account_sid", "ACxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx"),
            ("phone_number", "+5511999999999"),
        ]))
        .credentials(to_map(&[
            ("auth_token", "seu-auth-token"),
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

O canal nasce `disconnected`; sem o `connect`, o envio é recusado com `channel is not connected`.
`webhook_url` e `credentials.webhook_secret` são o endereço e o segredo da **sua aplicação** — a
tela do canal SMS não tem campos para eles.

:::warning PUT substitui o config
No `PUT /channels/CHANNEL_ID`, o `config` enviado substitui o anterior por inteiro; só `credentials`
é mesclado. Para acrescentar `status_callback_url` a um canal existente, reenvie `account_sid`,
`phone_number` (ou `messaging_service_sid`) e a nova chave juntos.
:::

## Enviar e receber

### Enviar

`to` é o telefone do destinatário em formato E.164.

<Tabs groupId="linguagem">
<TabItem value="curl" label="cURL">

```bash
curl -X POST "$LINKTOR_URL/api/v1/messages/send" \
  -H "X-API-Key: $LINKTOR_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "channel_id": "'"$CHANNEL_ID"'",
    "to": "+5511999999999",
    "text": "Sua Empresa: seu pedido saiu para entrega."
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
    "to": "+5511999999999",
    "text": "Sua Empresa: seu pedido saiu para entrega."
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
        "to": "+5511999999999",
        "text": "Sua Empresa: seu pedido saiu para entrega."
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
		"to": "+5511999999999",
		"text": "Sua Empresa: seu pedido saiu para entrega.",
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
      "to": "+5511999999999",
      "text": "Sua Empresa: seu pedido saiu para entrega."
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
    to = "+5511999999999",
    text = "Sua Empresa: seu pedido saiu para entrega.",
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
        'to' => '+5511999999999',
        'text' => 'Sua Empresa: seu pedido saiu para entrega.',
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
            "to": "+5511999999999",
            "text": "Sua Empresa: seu pedido saiu para entrega."
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

A resposta é `202` com `data.status = "queued"`. Depois chegam `message.sent` e, com o status
callback configurado, `message.delivered` ou `message.failed`.

Para MMS, use `content_type: "image"` com a URL em `attachments`; `text` vira o corpo da mensagem. A
Twilio baixa a mídia dessa URL, que precisa ser pública. Mais opções em
[Enviar mensagens](../api/enviar-mensagens.md).

### Receber

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
      "content": { "text": "Quero remarcar minha entrega" },
      "senderId": "+5511999999999",
      "senderType": "contact",
      "metadata": { "externalId": "SM0123456789abcdef0123456789abcdef" }
    },
    "conversationId": "c1a5d3e2-7b84-4a60-8e55-2f1d9b0c7a66",
    "contactId": "7e9a4b10-5c2d-4f3e-b8a1-6d0f2e3c4b5a",
    "channelId": "CHANNEL_ID",
    "channelType": "sms"
  }
}
```

- `senderId` é o telefone de quem enviou; `metadata.externalId` é o `MessageSid` da Twilio. SMS não
  traz nome do remetente, então não há `senderName`.
- Num MMS, `contentType` é `image` e `content.media` descreve a **primeira** mídia. As mídias da
  Twilio ficam atrás de autenticação; o Linktor as baixa com o Account SID e o Auth Token e as
  re-hospeda no armazenamento do servidor.
- O evento é entregue por `POST` no `webhook_url`, assinado com `credentials.webhook_secret`
  (detalhes em [Receber eventos](../api/receber-eventos.md)).

## Limites e particularidades

- **Tamanho:** a Twilio aceita até 1600 caracteres por mensagem e a divide em segmentos — 160
  caracteres no alfabeto GSM-7, ou 70 quando há acentos fora dele ou emoji (UCS-2). Cada segmento é
  cobrado.
- **Sem janela de atendimento:** não existe regra de 24 horas como no WhatsApp. Valem as regras de
  consentimento e de opt-out do país de destino; a Twilio trata palavras como `STOP` por conta
  própria e passa a recusar envios para quem saiu.
- **MMS recebido vira `image`**, qualquer que seja o tipo real do arquivo; o MIME verdadeiro vai em
  `content.media.mimeType`.
- **Modelos e botões** não existem em SMS. Um envio de modelo sai como texto simples com os
  parâmetros concatenados.
- **Erros da Twilio** no envio (número inválido, sem saldo, destino bloqueado) voltam como
  `message.failed` com a descrição da Twilio.
- **Trial:** uma conta de avaliação da Twilio só envia para números verificados e prefixa as
  mensagens com um aviso.

## Solução de problemas

| Sintoma | Causa provável | O que fazer |
|---|---|---|
| Webhook responde `401 invalid signature` | Proxy reverso alterando `Host` ou sem `X-Forwarded-Proto`; URL cadastrada diferente da acessada; Auth Token trocado na Twilio | Repasse `Host` e `X-Forwarded-Proto: https` no proxy; cadastre a URL exata; atualize o **Auth Token** no canal. |
| Webhook responde `401 webhook secret not configured` | Canal sem Auth Token em produção | Grave o **Auth Token** no canal. |
| Webhook responde `404 channel not found` | `CHANNEL_ID` errado na URL | Copie a URL da aba **Webhook**. |
| SMS chega na Twilio, mas não no Linktor | Webhook cadastrado no número, e o número pertence a um Messaging Service (ou o contrário) | Cadastre a URL no lugar que vale: o Messaging Service, quando existe. |
| Envio recusado com `channel is not connected` | Faltou o **Conectar** | Conecte o canal. |
| `message.failed` com erro de autenticação | Account SID ou Auth Token incorretos | Corrija as credenciais e salve. |
| `message.failed` logo após `message.sent` | Operadora recusou, número inválido ou destinatário em opt-out | Consulte o código de erro no **Monitor → Logs → Messaging** da Twilio. |
| Mensagens ficam em "enviada" e nunca "entregue" | Status callback não configurado | Veja [Status callback](#status-callback). |
| **Testar Conexão** sempre falha | Limitação atual da tela | Veja a nota em [Configurar no Linktor](#configurar-no-linktor). |
| O webhook é chamado, mas nada aparece no painel | Canal desconectado ou desabilitado: o recebimento é descartado (`channel is not active`) | Clique em **Conectar** e confira se o canal está habilitado. |
| Nenhum evento chega na sua aplicação | Canal sem `webhook_url` ou sem `credentials.webhook_secret` | Defina os dois pela API. |

Veja também: [Visão geral dos canais](visao-geral.md) · [Canais no painel](../painel/canais.md) ·
[Voz](voz.md), que usa a mesma conta Twilio.
