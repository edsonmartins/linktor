---
sidebar_position: 15
title: Direto (VendaX)
description: O canal direto — o aplicativo de chat VendaX Direto ligado ao Linktor por uma API REST de envio e um webhook assinado. Campos, protocolo, envio e recebimento.
---

# Direto (VendaX)

## Resumo

`direto` é o canal do **VendaX Direto**, um aplicativo de chat próprio da VendaX. Não é um
provedor público como WhatsApp ou Telegram: quem conversa do outro lado é um usuário do app
Direto, identificado pelo número de telefone.

A integração segue um protocolo próprio (referido no código como RFC-009), com formato parecido
com o da API do WhatsApp, em duas direções:

- **Linktor → Direto:** o Linktor chama a API REST de envio do runtime do Direto, com um token
  Bearer.
- **Direto → Linktor:** o runtime do Direto chama um webhook do Linktor, assinado com HMAC-SHA256.

Dados do canal:

- Valor de `type` na API: `direto`
- Como conecta: URL do runtime + id da instância + token da API + segredo do webhook
- Endereço do contato: o **telefone** do usuário do Direto, em formato internacional

| Recurso | Enviar | Receber | Observação |
|---|---|---|---|
| Texto | Sim | Sim | |
| Imagem, áudio, vídeo, documento | Sim | Sim | Enviado por URL; o runtime do Direto busca o arquivo |
| Localização | Não | Não | |
| Botões e listas | Vira texto | Não | Só o corpo da mensagem é entregue; as opções são descartadas |
| Modelos (templates) | Não | — | Envio de modelo falha — veja [Limites](#limites) |
| Grupos | Não | Não | Conversas 1:1 |
| Reação | Não | Não | |
| Confirmação de entrega e leitura | — | Sim | `message.delivered` e `message.read` |
| "Digitando…" do cliente | — | Sim | Aparece para o atendente na conversa aberta |

:::note O que esta página cobre
O lado do Linktor está descrito por inteiro, a partir do código. A configuração dentro do Direto
(onde gerar o token, onde cadastrar o webhook) pertence à VendaX e não está neste repositório:
peça esses dados a quem opera a sua instância do Direto.
:::

## Antes de começar

Você precisa de uma instância do Direto e, de quem a opera, quatro informações:

| Informação | Exemplo | Vai em |
|---|---|---|
| Endereço base do runtime do Direto | `https://direto.vendax.ai` | **URL de envio** |
| Identificador da instância | `inst_123` | **ID da instância** |
| Token da API de envio | — | **Token da API** |
| Segredo compartilhado do webhook | — | **Segredo do webhook de entrada** |

Também é preciso que:

- o Linktor esteja publicado em HTTPS e acessível pelo runtime do Direto (para o webhook);
- o Linktor alcance o runtime do Direto (para o envio e para baixar mídia);
- você tenha um usuário do painel ou uma chave de API com os scopes `channels:write` e
  `messages:send`. Veja [Chaves de API](../painel/chaves-de-api.md).

## Configurar na origem

Na instância do Direto, quem a opera precisa cadastrar dois valores:

1. **A URL do webhook do Linktor** — `https://linktor.suaempresa.com/api/v1/webhooks/direto/CHANNEL_ID`.
   Ela só existe depois de o canal ser criado no Linktor.
2. **O segredo do webhook** — o mesmo valor digitado em **Segredo do webhook de entrada**. O
   runtime usa esse segredo para assinar cada chamada.

Como o id do canal faz parte da URL, a ordem prática é: criar o canal no Linktor, copiar a URL,
cadastrá-la na instância do Direto.

## Configurar no Linktor

1. Abra **Canais** e, em **Adicionar Novo Canal**, escolha **Direto (VendaX)**.
2. Preencha os campos da tabela abaixo.
3. Clique em **Criar Canal**.
4. Reabra o canal, copie a **URL do Webhook** e passe-a a quem opera a instância do Direto.
5. No cartão do canal, clique em **Conectar**. Enquanto o canal não estiver **Conectado** e
   habilitado, o envio é recusado.

| Rótulo na tela | Chave | Onde fica | Obrigatório |
|---|---|---|---|
| **Nome do Canal** | `name` | campo do canal | Sim |
| **URL de envio** | `send_url` | `config` | Sim |
| **ID da instância** | `instance_id` | `config` | Sim |
| **Token da API** | `api_token` | `credentials` | Sim para enviar |
| **Segredo do webhook de entrada** | `webhook_secret` | `credentials` | Sim em produção |
| **URL do Consumidor** (em **Webhook de saída**) | `webhook_url` | campo do canal | Não |

`send_url` e `instance_id` não são segredo: ficam em `config` e voltam na consulta do canal.
`api_token` e `webhook_secret` ficam em `credentials`, cifrados, e nunca voltam pela API; ao
editar, deixá-los em branco mantém o valor guardado.

A **URL de envio** é só a base. O Linktor completa o caminho:
`<send_url>/channel/v1/<instance_id>/messages`.

:::warning Um segredo, dois usos
`webhook_secret` é usado nas duas pontas: confere a assinatura do que o **Direto envia ao
Linktor** e assina os eventos que o **Linktor envia ao seu sistema** (webhook de saída). Por isso
a tela do Direto não tem um campo **Segredo de Assinatura** separado. Se você usa a **URL do
Consumidor**, o seu sistema valida os eventos com o mesmo segredo configurado na instância do
Direto — e trocar o segredo afeta os dois lados ao mesmo tempo.
:::

:::note Sem botão de teste
A tela do Direto não tem **Testar Conexão**, e `POST /api/v1/channels/test` responde
`unsupported channel type: direto`. O teste é enviar uma mensagem de verdade e acompanhar o
resultado.
:::

:::note Filtro de eventos do webhook de saída
A tela do Direto não tem a lista **Eventos a entregar**. Para limitar os eventos enviados ao seu
sistema, grave `config.webhook_events` pela API junto com `send_url` e `instance_id`. Salvar o
canal pela tela depois disso apaga esse valor, porque o formulário reenvia `config` só com os
dois campos que conhece.
:::

## Webhook

URL que o runtime do Direto deve chamar:

```text
https://linktor.suaempresa.com/api/v1/webhooks/direto/CHANNEL_ID
```

Só `POST`, com corpo JSON.

### Assinatura

O runtime envia o header `X-Direto-Signature-256` no formato `sha256=<hex>`, onde o valor é o
HMAC-SHA256 do **corpo bruto** da requisição com o `webhook_secret` como chave — o mesmo esquema
do `X-Hub-Signature-256` da Meta.

```bash
CORPO='{"instanceId":"inst_123","channel":"direto","messages":[]}'
ASSINATURA="sha256=$(printf '%s' "$CORPO" | openssl dgst -sha256 -hmac "$SEGREDO" | sed 's/^.* //')"

curl -X POST "$LINKTOR_URL/api/v1/webhooks/direto/CHANNEL_ID" \
  -H "Content-Type: application/json" \
  -H "X-Direto-Signature-256: $ASSINATURA" \
  -d "$CORPO"
```

| Situação | Resposta |
|---|---|
| Canal não existe | `404 channel not found` |
| Canal com `webhook_secret` e assinatura ausente ou errada | `401 invalid signature` |
| Canal sem `webhook_secret`, servidor em produção (`release` ou `LINKTOR_REQUIRE_WEBHOOK_SECRETS=true`) | `401 webhook secret not configured` |
| Canal sem `webhook_secret`, servidor de desenvolvimento | Aceita sem validar |
| Corpo não é JSON válido | `400 invalid payload` |
| Aceito | `200 {"status":"ok"}` |

### Corpo

Uma chamada pode trazer mensagens, recibos e sinais efêmeros, em três listas independentes:

```json
{
  "instanceId": "inst_123",
  "channel": "direto",
  "messages": [
    {
      "id": "msg_01HXYZ",
      "from": "+5511999999999",
      "type": "text",
      "text": { "body": "Oi, meu pedido já saiu?" }
    },
    {
      "id": "msg_01HXZA",
      "from": "+5511999999999",
      "type": "image",
      "caption": "Foto do comprovante",
      "media": {
        "mime": "image/jpeg",
        "name": "comprovante.jpg",
        "url": "https://direto.vendax.ai/media/abc123",
        "ref": "/v0/file/s/abc123.jpg",
        "size": 184320
      }
    }
  ],
  "statuses": [
    { "id": "msg_01HXY0", "status": "delivered", "recipient": "+5511999999999", "timestamp": 1795528800 },
    { "id": "msg_01HXY0", "status": "read", "timestamp": 1795528860 }
  ],
  "events": [
    { "type": "typing", "from": "+5511999999999", "state": "on" }
  ]
}
```

| Campo | Significado |
|---|---|
| `messages[].id` | Id da mensagem no Direto; vira o `externalId` no Linktor |
| `messages[].from` | Telefone do cliente em E.164. O Linktor guarda sem o `+` |
| `messages[].text.body` | Texto, quando não há `media` |
| `messages[].media` | Anexo. `mime` define o tipo no Linktor (imagem, áudio, vídeo, documento); sem `mime`, vale `type` |
| `messages[].media.url` | Endereço de onde o Linktor baixa o arquivo. Se faltar, é usado `ref` |
| `messages[].caption` | Legenda do anexo |
| `statuses[].id` | Id que a API de envio do Direto devolveu para uma mensagem enviada pelo Linktor |
| `statuses[].status` | `delivered` ou `read`. Outros valores são ignorados |
| `statuses[].timestamp` | Segundos Unix; ausente ou zero vale "agora" |
| `events[]` | Sinal efêmero. Só `type: "typing"` é usado: `state: "on"` liga o "digitando…", qualquer outro valor desliga. `presence` é aceito e ignorado |

Uma mensagem com problema não derruba o lote: o Linktor processa as demais e responde `200`.

Reenvios com o mesmo corpo são barrados pela
[deduplicação de webhooks](./visao-geral.md#deduplicacao) e respondidos com
`200 {"status":"duplicate"}`.

### O que o Linktor envia ao Direto

Para cada mensagem de saída, o Linktor faz:

```http
POST <send_url>/channel/v1/<instance_id>/messages
Authorization: Bearer <api_token>
Content-Type: application/json
```

```json
{
  "to": "+5511999999999",
  "type": "text",
  "text": "Olá! Seu pedido saiu para entrega."
}
```

Com anexo:

```json
{
  "to": "+5511999999999",
  "type": "file",
  "mediaUrl": "https://linktor.suaempresa.com/…/contrato.pdf",
  "caption": "Segue o contrato",
  "filename": "contrato.pdf",
  "mimeType": "application/pdf"
}
```

`type` é `text`, `image`, `audio`, `video` ou `file` (o Direto chama documento de `file`). O
Linktor espera uma resposta `2xx` com o id da mensagem, que passa a ser o `externalId` usado nos
recibos:

```json
{ "messages": [ { "id": "msg_01HXY0" } ] }
```

Resposta `2xx` sem id é tratada como falha. O tempo limite da chamada é de 15 segundos.

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
  "type": "direto",
  "name": "Direto VendaX",
  "config": {
    "send_url": "https://direto.vendax.ai",
    "instance_id": "inst_123"
  },
  "credentials": {
    "api_token": "token_da_api_do_direto",
    "webhook_secret": "mesmo_segredo_configurado_na_instancia"
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
  type: 'direto',
  name: 'Direto VendaX',
  config: {
    send_url: 'https://direto.vendax.ai',
    instance_id: 'inst_123',
  },
  credentials: {
    api_token: 'token_da_api_do_direto',
    webhook_secret: 'mesmo_segredo_configurado_na_instancia',
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
        "type": "direto",
        "name": "Direto VendaX",
        "config": {
            "send_url": "https://direto.vendax.ai",
            "instance_id": "inst_123",
        },
        "credentials": {
            "api_token": "token_da_api_do_direto",
            "webhook_secret": "mesmo_segredo_configurado_na_instancia",
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
		Type: types.ChannelTypeDireto,
		Name: "Direto VendaX",
		Config: map[string]string{
			"send_url": "https://direto.vendax.ai",
			"instance_id": "inst_123",
		},
		Credentials: map[string]string{
			"api_token": "token_da_api_do_direto",
			"webhook_secret": "mesmo_segredo_configurado_na_instancia",
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
                .type(Channel.ChannelType.DIRETO)
                .name("Direto VendaX")
                .config(Map.of(
                    "send_url", "https://direto.vendax.ai",
                    "instance_id", "inst_123"))
                .credentials(Map.of(
                    "api_token", "token_da_api_do_direto",
                    "webhook_secret", "mesmo_segredo_configurado_na_instancia"))
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
    type = "direto",
    name = "Direto VendaX",
    config = new Dictionary<string, string>
    {
        ["send_url"] = "https://direto.vendax.ai",
        ["instance_id"] = "inst_123",
    },
    credentials = new Dictionary<string, string>
    {
        ["api_token"] = "token_da_api_do_direto",
        ["webhook_secret"] = "mesmo_segredo_configurado_na_instancia",
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
        'type' => 'direto',
        'name' => 'Direto VendaX',
        'config' => [
            'send_url' => 'https://direto.vendax.ai',
            'instance_id' => 'inst_123',
        ],
        'credentials' => [
            'api_token' => 'token_da_api_do_direto',
            'webhook_secret' => 'mesmo_segredo_configurado_na_instancia',
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
            CreateChannelInput::new("Direto VendaX", ChannelType::Direto)
                .config(HashMap::from([
                    ("send_url".to_string(), "https://direto.vendax.ai".to_string()),
                    ("instance_id".to_string(), "inst_123".to_string()),
                ]))
                .credentials(HashMap::from([
                    ("api_token".to_string(), "token_da_api_do_direto".to_string()),
                    ("webhook_secret".to_string(), "mesmo_segredo_configurado_na_instancia".to_string()),
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
    "type": "direto",
    "name": "Direto VendaX",
    "enabled": true,
    "connection_status": "disconnected",
    "environment": "production",
    "config": {
      "send_url": "https://direto.vendax.ai",
      "instance_id": "inst_123"
    },
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

Para o Direto, conectar apenas marca o canal como `connected`; nenhuma chamada é feita ao runtime
nesse passo.

:::warning Ao atualizar, reenvie o `config` inteiro
`PUT /api/v1/channels/CHANNEL_ID` **substitui** `config` pelo que for enviado. Para mudar só o
`instance_id`, envie também `send_url` (e `webhook_events`, se usar) — o que faltar é apagado.
`credentials` funciona ao contrário: é mesclado, e uma chave ausente ou vazia mantém o valor
guardado.

```bash
curl -X PUT "$LINKTOR_URL/api/v1/channels/CHANNEL_ID" \
  -H "X-API-Key: $LINKTOR_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "config": {
      "send_url": "https://direto.vendax.ai",
      "instance_id": "inst_456"
    }
  }'
```

:::

## Enviar e receber

### Enviar

`POST /api/v1/messages/send` com o scope `messages:send`. Em `to` vai o telefone do usuário do
Direto. Com ou sem `+`, o Linktor normaliza: guarda sem o sinal e envia ao Direto com ele.

<Tabs groupId="linguagem">
<TabItem value="curl" label="cURL">

```bash
curl -X POST "$LINKTOR_URL/api/v1/messages/send" \
  -H "X-API-Key: $LINKTOR_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
  "channel_id": "CHANNEL_ID",
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
    channel_id: 'CHANNEL_ID',
    to: '+5511999999999',
    text: 'Olá! Seu pedido saiu para entrega.',
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
        "to": "+5511999999999",
        "text": "Olá! Seu pedido saiu para entrega.",
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
		"to":         "+5511999999999",
		"text":       "Olá! Seu pedido saiu para entrega.",
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
              "to": "+5511999999999",
              "text": "Olá! Seu pedido saiu para entrega."
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
    to = "+5511999999999",
    text = "Olá! Seu pedido saiu para entrega.",
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
        'to' => '+5511999999999',
        'text' => 'Olá! Seu pedido saiu para entrega.',
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
            "to": "+5511999999999",
            "text": "Olá! Seu pedido saiu para entrega."
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

Depois chegam, no webhook de saída, `message.sent` (o Direto aceitou), `message.delivered` e
`message.read` (recibos do Direto) ou `message.failed`.

Para enviar um arquivo, informe `content_type` e `attachments`. O Linktor repassa a URL ao Direto,
que busca o arquivo — ela precisa ser alcançável pelo runtime:

```bash
curl -X POST "$LINKTOR_URL/api/v1/messages/send" \
  -H "X-API-Key: $LINKTOR_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "channel_id": "CHANNEL_ID",
    "to": "+5511999999999",
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

Para não enviar duas vezes em caso de repetição, informe `metadata.idempotency_key` (até 255
caracteres, único por empresa): a segunda chamada devolve a mensagem original.

### Receber

Com **URL do Consumidor** preenchida (e o `webhook_secret` gravado), cada mensagem recebida do
Direto é entregue ao seu sistema como `message.received`:

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
        "text": "Oi, meu pedido já saiu?"
      },
      "senderId": "5511999999999",
      "senderType": "contact",
      "metadata": {
        "externalId": "msg_01HXYZ"
      }
    },
    "conversationId": "5d1c1a1e-8f0a-4a39-b7a0-3c2c9d7e6b10",
    "contactId": "a1b2c3d4-e5f6-4a7b-8c9d-0e1f2a3b4c5d",
    "channelId": "CHANNEL_ID",
    "channelType": "direto"
  }
}
```

| Campo | No Direto |
|---|---|
| `data.message.senderId` | Telefone do cliente, **sem** o `+` |
| `data.message.metadata.externalId` | Id da mensagem no Direto |
| `data.message.contentType` | `text`, ou `image` / `video` / `audio` / `document` conforme o MIME do anexo |
| `data.message.content.text` | Texto da mensagem, ou a legenda quando há anexo |
| `data.message.content.media` | Presente quando há anexo: `url`, `mimeType`, `filename`, `size` |

O Direto não informa o nome do cliente; o contato é criado pelo telefone. Se já existir um contato
com esse telefone (vindo de outro canal), a conversa do Direto é ligada a ele.

A entrega é assinada com `X-Linktor-Signature` e `X-Linktor-Timestamp`, usando o `webhook_secret`
do canal; como validar está em [Receber eventos](../api/receber-eventos.md). Sem `webhook_secret`
o Linktor não entrega nada ao seu sistema.

## Limites e particularidades {#limites}

- **O telefone é a identidade.** O Linktor guarda telefones sem `+` e envia ao Direto em E.164
  com `+`. O Direto compara o número exato: um destinatário que não é usuário do app é recusado
  (por exemplo `404 channel_unknown_recipient`) e a mensagem vira `message.failed`, sem nova
  tentativa.
- **Erros `4xx` do Direto não são repetidos** (destinatário desconhecido, token inválido). Erros
  `5xx` e falhas de rede são tentados de novo pela fila.
- **Botões viram texto.** Uma mensagem de bot ou fluxo com respostas rápidas é entregue só com o
  corpo; as opções não aparecem. Escreva as opções no próprio texto.
- **Modelos não são suportados.** Um envio de modelo é recusado pelo adaptador com erro permanente
  (`unsupported content kind`), assim como uma reação.
- **Mídia enviada é por URL.** O Linktor não faz upload ao Direto: informa `mediaUrl`, `filename`
  e `mimeType`, e o runtime busca o arquivo.
- **Mídia recebida é copiada para o Linktor.** Quando o servidor tem armazenamento de mídia
  configurado, o Linktor baixa o arquivo e entrega uma URL própria. O token da API só é enviado
  no download quando o endereço do arquivo está no mesmo host da **URL de envio** — um anexo
  hospedado em outro domínio é baixado sem autenticação. Sem armazenamento, o evento leva a URL
  original do Direto. Veja [Configuração](../implantar/configuracao.md).
- **"Digitando…" só em conversa aberta.** O sinal aparece para o atendente quando já existe um
  contato com aquele telefone e uma conversa aberta no canal. Presença (`active` / `away`) é
  recebida e ainda não é exibida.
- **Sandbox não se aplica.** A proteção de canal sandbox existe só para WhatsApp; um canal Direto
  criado com `environment: "sandbox"` tem todo envio bloqueado. Veja
  [Sandbox](../painel/sandbox.md).

## Solução de problemas

| Sintoma | Causa provável | O que fazer |
|---|---|---|
| O Direto recebe `401 invalid signature` | Segredo diferente nos dois lados, ou assinatura calculada sobre um corpo reformatado | Regrave o **Segredo do webhook de entrada**; assine o corpo bruto, byte a byte |
| O Direto recebe `401 webhook secret not configured` | Canal sem segredo em servidor de produção | Preencha o **Segredo do webhook de entrada** |
| O Direto recebe `404 channel not found` | Id do canal errado na URL | Copie a URL pelo botão do canal |
| `message.failed` com `send_url is required`, `instance_id is required` ou `api_token is required` | Campo em branco — comum depois de um `PUT` que enviou `config` incompleto | Regrave os três valores |
| `message.failed` com `direto send API returned 401` | Token da API errado ou revogado | Atualize o **Token da API** |
| `message.failed` com `direto send API returned 404` | Destinatário não é usuário do Direto, ou **ID da instância** errado | Confira o telefone e a instância |
| `message.failed` com `no message id in response` | O runtime respondeu `2xx` sem `messages[0].id` | Verifique a versão do runtime do Direto |
| Envio volta `channel is not connected` | Canal criado mas não conectado | Clique em **Conectar** no cartão do canal |
| Mensagem fica em **enviada** e nunca **entregue** | O runtime não está mandando `statuses`, ou manda um `id` diferente do devolvido no envio | Confira o webhook do lado do Direto |
| Nada chega ao seu sistema | `webhook_secret` vazio, ou evento fora de `config.webhook_events` | Preencha o segredo; revise o filtro |

Os erros de entrega e de webhook de cada canal aparecem em
[Observabilidade](../painel/observabilidade.md).
