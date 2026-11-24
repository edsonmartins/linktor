---
sidebar_position: 10
title: RCS
description: Conecte um agente RCS Business Messaging ao Linktor por Zenvia, Infobip, Pontaltech ou Google — o que cada provedor suporta de fato, campos, webhook, API e limitações.
---

# RCS

O canal RCS (Rich Communication Services) envia e recebe mensagens pelo aplicativo de mensagens
nativo do celular, em nome de um **agente** de marca verificado. O Linktor fala com o agente por
meio de um provedor.

Valor de `type` na API: **`rcs`**. O provedor vai em `config.provider`.

:::warning Estado do canal
O código aceita quatro provedores, mas eles não estão no mesmo nível. Leia a tabela antes de
escolher: hoje, **Zenvia é o único com envio e recebimento de texto e mídia**. Cards ricos,
carrosséis e respostas sugeridas — o que diferencia o RCS do SMS — **não são enviados por nenhum
provedor**, embora a tela do canal os liste em **Recursos do RCS**.
:::

| Provedor (`config.provider`) | Enviar texto | Enviar mídia | Receber | Status de entrega | Observação |
|---|---|---|---|---|---|
| `zenvia` (padrão) | Sim | Sim | Texto e arquivo | `sent`, `delivered`, `read`, `failed` | Caminho mais completo. |
| `infobip` | Sim | Não (só o texto é enviado) | Só texto | Sim | Exige a URL base da sua conta. |
| `pontaltech` | Sim | Não (só o texto é enviado) | Só texto | Sim | |
| `google` | Parcial | Não | Só texto | Não | Veja o aviso sobre autenticação abaixo. |

| Recurso | Suporte |
|---|---|
| Texto | Sim |
| Mídia | Só Zenvia, no envio e no recebimento |
| Localização e toque em resposta sugerida (recebimento) | Não — nenhum provedor entrega esses dados ao Linktor hoje |
| Cards ricos, carrosséis, respostas sugeridas (envio) | **Não** |
| Modelos | Não (um envio de modelo é achatado em texto) |
| Grupos | Não |
| Confirmação de leitura | Sim, quando o provedor informa |

:::warning Google RBM direto
Com `provider: "google"`, o Linktor envia o valor de `api_key` como `Authorization: Bearer`. A API
do Google RCS Business Messaging exige um **token de acesso OAuth 2.0** obtido a partir da conta de
serviço — colar o JSON da conta de serviço, como a tela sugere (**Service Account JSON**), não gera
esse token. Trate a integração direta com o Google como **não funcional** no estado atual e prefira
um provedor intermediário.
:::

## Antes de começar

- Um **agente RCS** aprovado: marca verificada e agente lançado nas operadoras onde você vai enviar.
  A aprovação é feita pelo provedor junto ao Google e às operadoras, e leva de dias a semanas.
- Contrato e credenciais de API no provedor.
- O servidor do Linktor acessível pela internet em HTTPS, para o webhook.
- RCS não alcança todo mundo: o destinatário precisa de um aparelho e de uma operadora com RCS
  ativo. O Linktor **não faz fallback automático para SMS** — a mensagem falha, e a decisão de
  reenviar por outro canal é da sua aplicação.

## Configurar na origem

Cada provedor tem o seu console, e os nomes de menu mudam; use esta lista como roteiro do que obter.

**Zenvia**

1. No console da Zenvia, confirme que o canal RCS está contratado e anote o **identificador do
   agente/remetente** RCS da sua conta.
2. Gere um **token de API** (área de tokens e webhooks do console).
3. Crie uma assinatura de webhook para o canal RCS com os eventos de **mensagem** (`MESSAGE`) e de
   **status** (`MESSAGE_STATUS`), apontando para a URL da seção [Webhook](#webhook).

**Infobip**

1. No portal, crie uma **API key** e anote a **URL base** da sua conta (algo como
   `https://xxxxx.api.infobip.com`).
2. Anote o identificador do remetente RCS.
3. Cadastre a URL do webhook para mensagens recebidas e para relatórios de entrega.

**Pontaltech**

1. Solicite ao suporte a **chave de API** e o **segredo**, e o identificador do agente.
2. Peça o cadastro da URL do webhook.

## Configurar no Linktor

1. No painel, abra **Canais** e, em **Adicionar Novo Canal**, clique em **RCS**.
2. Em **Provedor RCS**, escolha o provedor.
3. Na aba **Credenciais**, preencha os campos (os rótulos dos campos de provedor estão em inglês na
   tela).
4. Clique em **Salvar Configuração**. O canal é criado com o nome `RCS` seguido do nome do provedor
   (por exemplo `RCS Zenvia`); a tela não tem campo de nome.
5. No cartão do canal, abra o menu de ações e clique em **Conectar**.
6. Cadastre no provedor a URL da seção [Webhook](#webhook).

| Campo na tela | Chave | Provedores | Obrigatório |
|---|---|---|---|
| **Provedor RCS** | `config.provider` | todos | Sim (`zenvia`, `infobip`, `pontaltech`, `google`) |
| **Agent ID** | `config.agent_id` | todos | Sim |
| **API Token** (Zenvia) / **API Key** | `credentials.api_key` | todos | Sim |
| **Sender ID (Optional)** | `config.sender_id` | Zenvia | Não — gravado, mas o envio usa o **Agent ID** como remetente |
| **Base URL** | `config.base_url` | Infobip | Sim para Infobip |
| **API Secret** | `credentials.api_secret` | Pontaltech | Exigido pela tela; o envio autentica só com a chave |
| **Service Account JSON** | `credentials.api_key` | Google | Veja o aviso acima |

A tela grava `api_key` e `api_secret` também dentro de `config`; lá eles são cifrados e voltam
mascarados nas respostas da API.

Chave aceita apenas pela API:

| Chave | Efeito |
|---|---|
| `credentials.webhook_secret` | Liga a validação de assinatura do webhook (seção abaixo). Em produção, é obrigatória. |

:::note Limitações da tela
- **Testar Conexão** confere apenas se provedor, **Agent ID** e chave foram preenchidos — não
  consulta o provedor.
- A aba **Webhook** deveria exibir a URL depois de salvar, mas hoje fica vazia. Use a URL desta
  página, com o id do canal (copie-o no cartão do canal, em **Copiar ID do canal**).
:::

## Webhook

```text
https://linktor.suaempresa.com/api/v1/webhooks/rcs/CHANNEL_ID
```

A mesma URL recebe mensagens e status. O corpo é interpretado conforme o `config.provider` do canal
(sem provedor gravado, assume Zenvia).

**Assinatura.** Quando o canal tem `credentials.webhook_secret`, o Linktor exige o HMAC-SHA256 do
corpo, em hexadecimal, no header `X-Signature` ou `X-Hub-Signature-256` (com ou sem o prefixo
`sha256=`), e responde `401 invalid signature` se não bater.

- Sem segredo, o webhook é aceito sem validação fora de produção.
- Com `LINKTOR_SERVER_MODE=release` (ou `LINKTOR_REQUIRE_WEBHOOK_SECRETS=true`), um canal sem
  segredo recusa tudo com `401 webhook secret not configured`.

:::warning Confirme o formato com o provedor
O formato de assinatura acima é o que o Linktor espera; **não foi confirmado** que cada provedor
assina os webhooks dessa maneira. Se o seu provedor não enviar esse header, em produção será preciso
um intermediário seu que calcule o HMAC e repasse a chamada ao Linktor. Teste o recebimento em
homologação antes de contar com o canal.
:::

Com Zenvia, uma falha ao processar a mensagem recebida devolve `500`, para a Zenvia tentar de novo.

## Criar o canal pela API

<Tabs groupId="linguagem">
<TabItem value="curl" label="cURL">

```bash
# 1. Cria o canal (nasce "disconnected")
curl -X POST "$LINKTOR_URL/api/v1/channels" \
  -H "X-API-Key: $LINKTOR_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "type": "rcs",
    "name": "RCS Zenvia",
    "config": {
      "provider": "zenvia",
      "agent_id": "seu-agente-rcs"
    },
    "credentials": {
      "api_key": "seu-token-zenvia",
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
  "type": "rcs",
  "name": "RCS Zenvia",
  "config": {
    "provider": "zenvia",
    "agent_id": "seu-agente-rcs"
  },
  "credentials": {
    "api_key": "seu-token-zenvia",
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
    name="RCS Zenvia",
    type="rcs",
    config={
        "provider": "zenvia",
        "agent_id": "seu-agente-rcs"
    },
    credentials={
        "api_key": "seu-token-zenvia",
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
		Name: "RCS Zenvia",
		Type: types.ChannelType("rcs"),
		Config: map[string]string{
			"provider": "zenvia",
			"agent_id": "seu-agente-rcs",
		},
		Credentials: map[string]string{
			"api_key": "seu-token-zenvia",
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
        .type(Channel.ChannelType.RCS)
        .name("RCS Zenvia")
        .config(Map.of(
            "provider", "zenvia",
            "agent_id", "seu-agente-rcs"))
        .credentials(Map.of(
            "api_key", "seu-token-zenvia",
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
  "type": "rcs",
  "name": "RCS Zenvia",
  "config": {
    "provider": "zenvia",
    "agent_id": "seu-agente-rcs"
  },
  "credentials": {
    "api_key": "seu-token-zenvia",
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
    'type' => 'rcs',
    'name' => 'RCS Zenvia',
    'config' => [
        'provider' => 'zenvia',
        'agent_id' => 'seu-agente-rcs',
    ],
    'credentials' => [
        'api_key' => 'seu-token-zenvia',
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

    let input = CreateChannelInput::new("RCS Zenvia", ChannelType::Rcs)
        .config(to_map(&[
            ("provider", "zenvia"),
            ("agent_id", "seu-agente-rcs"),
        ]))
        .credentials(to_map(&[
            ("api_key", "seu-token-zenvia"),
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

O canal nasce `disconnected`; sem o `connect` o envio é recusado. `webhook_url` é o endereço da
**sua aplicação** para os eventos do Linktor.

:::warning Um segredo, dois usos
Neste canal, `credentials.webhook_secret` valida o webhook **do provedor** e também assina os
eventos enviados à **sua aplicação**. Não há chave separada: o provedor e a sua aplicação precisam
conhecer o mesmo valor.
:::

:::warning PUT substitui o config
No `PUT /channels/CHANNEL_ID`, o `config` enviado substitui o anterior por inteiro; só `credentials`
é mesclado. Ao alterar uma chave de `config`, reenvie `provider` e `agent_id` junto.
:::

## Enviar e receber

### Enviar

`to` é o telefone do destinatário, no formato que o provedor espera (em geral E.164:
`+5511999999999`).

<Tabs groupId="linguagem">
<TabItem value="curl" label="cURL">

```bash
curl -X POST "$LINKTOR_URL/api/v1/messages/send" \
  -H "X-API-Key: $LINKTOR_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "channel_id": "'"$CHANNEL_ID"'",
    "to": "+5511999999999",
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
    "to": "+5511999999999",
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
        "to": "+5511999999999",
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
		"to": "+5511999999999",
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
      "to": "+5511999999999",
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
    to = "+5511999999999",
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
        'to' => '+5511999999999',
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
            "to": "+5511999999999",
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

A resposta é `202` com `data.status = "queued"`. O desfecho chega como `message.sent`,
`message.delivered`, `message.read` ou `message.failed`, conforme o provedor informar.

Com Zenvia, para enviar mídia use `content_type` `image`, `video`, `audio` ou `document` com a URL
em `attachments`; texto e arquivo vão como partes da mesma mensagem. Nos demais provedores só o
texto é transmitido — um envio de mídia sem `text` chega vazio ou é recusado pelo provedor. Mais
opções em [Enviar mensagens](../api/enviar-mensagens.md).

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
      "content": { "text": "Sim, pode confirmar" },
      "senderId": "+5511999999999",
      "senderType": "contact",
      "metadata": { "externalId": "a3f1c2d4-5b6e-47f8-9a0b-1c2d3e4f5a6b" }
    },
    "conversationId": "c1a5d3e2-7b84-4a60-8e55-2f1d9b0c7a66",
    "contactId": "7e9a4b10-5c2d-4f3e-b8a1-6d0f2e3c4b5a",
    "channelId": "CHANNEL_ID",
    "channelType": "rcs"
  }
}
```

- `senderId` é o telefone do contato; `metadata.externalId` é o id da mensagem no provedor.
- Mídia recebida (Zenvia) vem em `content.media`, com `contentType` `image`, `video`, `audio` ou `document`
  conforme o MIME; o Linktor baixa o arquivo e o re-hospeda no armazenamento do servidor.
- O evento é entregue por `POST` no `webhook_url` do canal, assinado com
  `credentials.webhook_secret` (detalhes em [Receber eventos](../api/receber-eventos.md)).

## Limites e particularidades

- **Sem conteúdo rico no envio.** Cards, carrosséis e respostas sugeridas não são transmitidos. Com
  Zenvia, uma mensagem que só tenha esse tipo de conteúdo falha com um erro explícito em vez de sair
  vazia.
- **Regras da plataforma.** Janelas de conversa, categorias de mensagem e cobrança do RCS são
  definidas pelo Google, pelas operadoras e pelo contrato com o provedor. O Linktor não aplica
  nenhuma janela neste canal: o que o provedor recusar volta como `message.failed`.
- **Contato identificado pelo telefone.** Cada telefone é um contato; RCS não traz nome de perfil.
- **Sem fallback para SMS.** Destinatário sem RCS resulta em falha.
- **Tipo de mídia enviado à Zenvia.** O campo de tipo do arquivo é preenchido com a categoria
  (`image`, `video`...) e não com o MIME completo; se a Zenvia recusar a mídia, este é o primeiro
  ponto a investigar.
- **Limitar o canal por instalação.** A variável `LINKTOR_ENABLED_CHANNEL_TYPES` define quais tipos
  de canal podem ser criados e conectados. Se `rcs` não estiver nela, a criação é recusada.

## Solução de problemas

| Sintoma | Causa provável | O que fazer |
|---|---|---|
| Criação recusada pelo servidor | `rcs` fora de `LINKTOR_ENABLED_CHANNEL_TYPES` | Inclua o tipo na variável (veja [Configuração](../implantar/configuracao.md)). |
| A aba **Webhook** não mostra a URL | Limitação atual da tela | Monte a URL com o id do canal, como na seção [Webhook](#webhook). |
| Webhook responde `401 webhook secret not configured` | Produção, canal sem `credentials.webhook_secret` | Grave o segredo pela API e alinhe a assinatura com o provedor. |
| Webhook responde `401 invalid signature` | Provedor não assina no formato esperado, ou segredo diferente | Veja o aviso em [Webhook](#webhook). |
| Webhook responde `400 invalid payload` | `config.provider` não corresponde ao provedor que está chamando | Corrija o provedor do canal. |
| `message.failed` em todo envio com Google | Autenticação por conta de serviço não implementada | Use um provedor intermediário. |
| `message.failed` com erro `401`/`403` do provedor | Chave de API inválida ou sem permissão para RCS | Gere outra chave no provedor e atualize o canal. |
| Mídia não chega (Infobip, Pontaltech, Google) | Só Zenvia envia mídia | Envie o link no texto ou use Zenvia. |
| Envio recusado com `channel is not connected` | Faltou o **Conectar** | Conecte o canal. |
| O webhook é chamado, mas nada aparece no painel | Canal desconectado ou desabilitado: o recebimento é descartado (`channel is not active`) | Clique em **Conectar** e confira se o canal está habilitado. |
| Nenhum evento chega na sua aplicação | Canal sem `webhook_url` ou sem `credentials.webhook_secret` | Defina os dois pela API. |

Veja também: [Visão geral dos canais](visao-geral.md) · [SMS](sms.md) ·
[Canais no painel](../painel/canais.md).
