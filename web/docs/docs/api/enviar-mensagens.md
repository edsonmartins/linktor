---
sidebar_position: 3
title: Enviar mensagens
description: Envio direto por canal e destinatário, envio dentro de uma conversa, anexos, reações, grupos, modelos do WhatsApp e acompanhamento do status.
---

# Enviar mensagens

Há três rotas de envio. Escolha pela informação que você tem em mãos:

| Você tem | Rota | Scopes da chave |
|---|---|---|
| Um canal e o endereço do destinatário (telefone, e-mail, id) | [`POST /messages/send`](#envio-direto) | `messages:send` |
| O id de uma conversa que já existe | [`POST /conversations/:id/messages`](#envio-na-conversa) | `messages:send` + `conversations:write` |
| Um canal de WhatsApp não oficial e o id de um grupo | [`POST /channels/:id/groups/:groupId/messages`](#grupo) | `messages:send` |

As três **enfileiram** a mensagem e respondem na hora. A entrega acontece depois, em segundo plano, com
nova tentativa automática em caso de falha; o resultado chega no
[webhook da sua aplicação](./receber-eventos.md). Veja [Acompanhar o status](#status).

:::note SDKs
`POST /messages/send` existe nos SDKs TypeScript (`messages.sendDirect`) e Go (`Messages.SendDirect`),
mas nesta versão os dois devolvem a resposta sem desembrulhar o envelope `{ success, data }` — em Go os
campos do resultado chegam vazios, e em TypeScript o `id` fica em `resultado.data.id`. Os tipos deles
também só aceitam `text`. Os métodos de envio em conversa dos SDKs (`sendText`, `sendMessage`) mandam o
campo `text`, enquanto a rota espera `content` e `content_type`. Por isso os exemplos desta página
usam o cliente HTTP nativo de cada linguagem, que funciona em todas. O estado de cada SDK está em
[SDKs](../sdks/visao-geral.md).
:::

## Envio direto {#envio-direto}

`POST /api/v1/messages/send`

Envia para um destinatário em um canal sem que você precise conhecer contato ou conversa. O Linktor
localiza — ou cria — o contato, a identidade dele naquele canal e a conversa, grava a mensagem e a
coloca na fila de saída.

### Corpo

| Campo | Tipo | Obrigatório | Descrição |
|---|---|---|---|
| `channel_id` | texto | sim | Id do canal de saída |
| `to` | texto | sim | Endereço do destinatário: telefone (`+5511999999999`), e-mail no canal `email`, ou o id do destinatário no canal |
| `content_type` | texto | não | `text` (padrão), `image`, `video`, `audio`, `document` ou `sticker` |
| `text` | texto | sim para `text` | Texto da mensagem; em mídia, vira a legenda |
| `content` | texto | não | Sinônimo de `text` (mesmo nome usado na rota de conversa). Se os dois vierem, vale `text` |
| `attachments` | lista | sim para mídia | Anexos já enviados ao Linktor — veja [Anexos](#anexos) |
| `metadata` | mapa texto → texto | não | Dados seus que acompanham a mensagem — veja [metadata](#metadata) |

Regras:

- Sem `content_type`, a mensagem é `text`; se houver `attachments`, é `document`.
- `image`, `video`, `audio`, `document` e `sticker` exigem ao menos um anexo.
- Os tipos `location`, `contact`, `template`, `interactive`, `poll` e `reaction` **não** são aceitos
  nesta rota (`400`). Para eles use o [envio na conversa](#envio-na-conversa).
- Telefone pode vir com ou sem `+`, espaços ou traços: é normalizado para E.164. E-mail é convertido
  para minúsculas.
- O canal precisa ser do seu tenant, estar habilitado e conectado.

<Tabs groupId="linguagem">
<TabItem value="curl" label="cURL">

```bash
curl -X POST "$LINKTOR_URL/api/v1/messages/send" \
  -H "X-API-Key: $LINKTOR_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "channel_id": "CHANNEL_ID",
    "to": "+5511999999999",
    "content_type": "text",
    "text": "Seu pedido 1234 saiu para entrega.",
    "metadata": {
      "source": "erp",
      "idempotency_key": "pedido-1234-saiu-para-entrega"
    }
  }'
```

</TabItem>
<TabItem value="typescript" label="TypeScript">

```typescript
const res = await fetch(`${process.env.LINKTOR_URL}/api/v1/messages/send`, {
  method: 'POST',
  headers: {
    'X-API-Key': process.env.LINKTOR_API_KEY!,
    'Content-Type': 'application/json',
  },
  body: JSON.stringify({
    channel_id: 'CHANNEL_ID',
    to: '+5511999999999',
    content_type: 'text',
    text: 'Seu pedido 1234 saiu para entrega.',
    metadata: {
      source: 'erp',
      idempotency_key: 'pedido-1234-saiu-para-entrega',
    },
  }),
});

const body = await res.json();
if (res.status !== 202) {
  const err = body.error ?? body;
  throw new Error(`${res.status} ${err.code}: ${err.message}`);
}
console.log(body.data.id, body.data.conversation_id, body.data.status);
```

</TabItem>
<TabItem value="python" label="Python">

```python
import os
import requests

res = requests.post(
    f"{os.environ['LINKTOR_URL']}/api/v1/messages/send",
    headers={"X-API-Key": os.environ["LINKTOR_API_KEY"]},
    json={
        "channel_id": "CHANNEL_ID",
        "to": "+5511999999999",
        "content_type": "text",
        "text": "Seu pedido 1234 saiu para entrega.",
        "metadata": {
            "source": "erp",
            "idempotency_key": "pedido-1234-saiu-para-entrega",
        },
    },
    timeout=30,
)

body = res.json()
if res.status_code != 202:
    err = body.get("error") or body
    raise RuntimeError(f"{res.status_code} {err.get('code')}: {err.get('message')}")

data = body["data"]
print(data["id"], data["conversation_id"], data["status"])
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

type DirectSend struct {
	ChannelID   string            `json:"channel_id"`
	To          string            `json:"to"`
	ContentType string            `json:"content_type,omitempty"`
	Text        string            `json:"text,omitempty"`
	Metadata    map[string]string `json:"metadata,omitempty"`
}

type DirectSendResponse struct {
	Success bool `json:"success"`
	Data    struct {
		ID             string `json:"id"`
		ConversationID string `json:"conversation_id"`
		ChannelID      string `json:"channel_id"`
		Status         string `json:"status"`
	} `json:"data"`
}

func main() {
	payload, _ := json.Marshal(DirectSend{
		ChannelID:   "CHANNEL_ID",
		To:          "+5511999999999",
		ContentType: "text",
		Text:        "Seu pedido 1234 saiu para entrega.",
		Metadata: map[string]string{
			"source":          "erp",
			"idempotency_key": "pedido-1234-saiu-para-entrega",
		},
	})

	req, _ := http.NewRequest(http.MethodPost,
		os.Getenv("LINKTOR_URL")+"/api/v1/messages/send", bytes.NewReader(payload))
	req.Header.Set("X-API-Key", os.Getenv("LINKTOR_API_KEY"))
	req.Header.Set("Content-Type", "application/json")

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Fatal(err)
	}
	defer res.Body.Close()

	raw, _ := io.ReadAll(res.Body)
	if res.StatusCode != http.StatusAccepted {
		log.Fatalf("%d %s", res.StatusCode, raw)
	}

	var out DirectSendResponse
	if err := json.Unmarshal(raw, &out); err != nil {
		log.Fatal(err)
	}
	fmt.Println(out.Data.ID, out.Data.ConversationID, out.Data.Status)
}
```

</TabItem>
<TabItem value="java" label="Java">

```java
import java.net.URI;
import java.net.http.HttpClient;
import java.net.http.HttpRequest;
import java.net.http.HttpResponse;

public class EnvioDireto {
    public static void main(String[] args) throws Exception {
        String json = """
            {
              "channel_id": "CHANNEL_ID",
              "to": "+5511999999999",
              "content_type": "text",
              "text": "Seu pedido 1234 saiu para entrega.",
              "metadata": {
                "source": "erp",
                "idempotency_key": "pedido-1234-saiu-para-entrega"
              }
            }
            """;

        HttpRequest request = HttpRequest.newBuilder(
                URI.create(System.getenv("LINKTOR_URL") + "/api/v1/messages/send"))
            .header("X-API-Key", System.getenv("LINKTOR_API_KEY"))
            .header("Content-Type", "application/json")
            .POST(HttpRequest.BodyPublishers.ofString(json))
            .build();

        HttpResponse<String> response = HttpClient.newHttpClient()
            .send(request, HttpResponse.BodyHandlers.ofString());

        if (response.statusCode() != 202) {
            throw new RuntimeException(response.statusCode() + " " + response.body());
        }
        // {"success":true,"data":{"id":"...","conversation_id":"...","channel_id":"...","status":"queued"}}
        System.out.println(response.body());
    }
}
```

</TabItem>
<TabItem value="csharp" label="C#">

```csharp
using System.Net;
using System.Net.Http.Json;
using System.Text.Json;

var http = new HttpClient();
http.DefaultRequestHeaders.Add("X-API-Key", Environment.GetEnvironmentVariable("LINKTOR_API_KEY"));

var response = await http.PostAsJsonAsync(
    Environment.GetEnvironmentVariable("LINKTOR_URL") + "/api/v1/messages/send",
    new
    {
        channel_id = "CHANNEL_ID",
        to = "+5511999999999",
        content_type = "text",
        text = "Seu pedido 1234 saiu para entrega.",
        metadata = new Dictionary<string, string>
        {
            ["source"] = "erp",
            ["idempotency_key"] = "pedido-1234-saiu-para-entrega",
        },
    });

var body = await response.Content.ReadFromJsonAsync<JsonElement>();
if (response.StatusCode != HttpStatusCode.Accepted)
{
    throw new Exception($"{(int)response.StatusCode} {body}");
}

var data = body.GetProperty("data");
Console.WriteLine($"{data.GetProperty("id")} {data.GetProperty("conversation_id")} {data.GetProperty("status")}");
```

</TabItem>
<TabItem value="php" label="PHP">

```php
<?php
require 'vendor/autoload.php';

$http = new GuzzleHttp\Client(['http_errors' => false, 'timeout' => 30]);

$res = $http->post(getenv('LINKTOR_URL') . '/api/v1/messages/send', [
    'headers' => ['X-API-Key' => getenv('LINKTOR_API_KEY')],
    'json' => [
        'channel_id'   => 'CHANNEL_ID',
        'to'           => '+5511999999999',
        'content_type' => 'text',
        'text'         => 'Seu pedido 1234 saiu para entrega.',
        'metadata'     => [
            'source'          => 'erp',
            'idempotency_key' => 'pedido-1234-saiu-para-entrega',
        ],
    ],
]);

$body = json_decode((string) $res->getBody(), true);
if ($res->getStatusCode() !== 202) {
    $err = $body['error'] ?? $body;
    throw new RuntimeException("{$res->getStatusCode()} {$err['code']}: {$err['message']}");
}

echo "{$body['data']['id']} {$body['data']['conversation_id']} {$body['data']['status']}\n";
```

</TabItem>
<TabItem value="rust" label="Rust">

```rust
use serde_json::{json, Value};

#[tokio::main]
async fn main() -> Result<(), Box<dyn std::error::Error>> {
    let url = format!("{}/api/v1/messages/send", std::env::var("LINKTOR_URL")?);

    let res = reqwest::Client::new()
        .post(url)
        .header("X-API-Key", std::env::var("LINKTOR_API_KEY")?)
        .json(&json!({
            "channel_id": "CHANNEL_ID",
            "to": "+5511999999999",
            "content_type": "text",
            "text": "Seu pedido 1234 saiu para entrega.",
            "metadata": {
                "source": "erp",
                "idempotency_key": "pedido-1234-saiu-para-entrega"
            }
        }))
        .send()
        .await?;

    let status = res.status();
    let body: Value = res.json().await?;
    if status.as_u16() != 202 {
        return Err(format!("{status} {body}").into());
    }

    println!("{} {} {}", body["data"]["id"], body["data"]["conversation_id"], body["data"]["status"]);
    Ok(())
}
```

</TabItem>
</Tabs>

### Resposta

`202 Accepted`:

```json
{
  "success": true,
  "data": {
    "id": "9a8b7c6d-5e4f-4a3b-9c2d-1e0f9a8b7c6d",
    "conversation_id": "c1f0f5a2-7f1e-4c58-8a44-2b7f5a1d9e03",
    "channel_id": "CHANNEL_ID",
    "status": "queued"
  }
}
```

| Campo | Descrição |
|---|---|
| `id` | Id da mensagem no Linktor. É o `messageId` dos eventos de status e o id de `GET /messages/:id` |
| `conversation_id` | Conversa onde a mensagem foi gravada (encontrada ou criada) |
| `channel_id` | Canal usado |
| `status` | `queued` em um envio novo. Em uma repetição idempotente, o status atual da mensagem original: `sent`, `delivered`, `read` ou `failed` |

`202` significa **aceita e enfileirada**, não entregue. Não conclua nada sobre a entrega a partir desta
resposta.

### `metadata` {#metadata}

`metadata` é um mapa de texto para texto. Tudo o que você colocar é gravado na mensagem e segue até o
adaptador do canal. Algumas chaves, além de viajar junto, mudam o comportamento:

| Chave | Efeito |
|---|---|
| `idempotency_key` | Impede envio duplicado — veja [Idempotência](#idempotencia) |
| `subject` | Assunto do e-mail no canal `email` (há um assunto padrão quando ausente) |
| `reply_to_id` | Envia como resposta citando outra mensagem — veja [Responder citando](#citacao) |
| `quoted_text` | Texto da mensagem citada, usado pelo WhatsApp não oficial para montar a citação |
| `alcada_correlation` | Token opaco devolvido no webhook quando o contato responde citando esta mensagem |
| `source` | Rótulo da origem; volta junto com `alcada_correlation` no webhook |

As duas últimas alimentam o bloco `context` do evento `message.received`, descrito em
[Receber eventos](./receber-eventos.md).

#### Chaves reservadas

Estas chaves são escritas pelo próprio Linktor. Uma requisição que nomeie qualquer uma delas é recusada
com `400` — a chave não é removida em silêncio:

`is_reaction`, `reaction`, `reaction_emoji`, `reaction_message_id`, `reaction_target_external_id`,
`reaction_target_message_id`, `campaign_id`, `campaign_recipient_id`, `sender_id`, `sender_name`,
`sender_type`, `is_group`, `chat_jid`, `mentions`, `external_id`, `message_id`, `conversation_id`,
`contact_id`, `channel_id`, `channel_type`, `tenant_id`, `environment`.

A comparação ignora maiúsculas e espaços nas pontas. A resposta lista as chaves ofensoras:

```json
{
  "success": false,
  "error": {
    "code": "VALIDATION_ERROR",
    "message": "metadata cannot overwrite Linktor internal fields",
    "details": { "reserved_keys": "campaign_id,sender_id" }
  }
}
```

A mesma regra vale nas três rotas de envio.

### Idempotência {#idempotencia}

Coloque em `metadata.idempotency_key` uma chave que identifique o envio no **seu** sistema (por exemplo
`pedido-1234-saiu-para-entrega`). A chave é única por tenant e tem no máximo 255 caracteres.

| Situação | Resposta |
|---|---|
| Primeira chamada com a chave | `202`, `status: "queued"` |
| Repetição com a mesma chave | `202` com o **mesmo** `id` e o status atual da mensagem original. Nada é enviado de novo |
| Repetição enquanto a primeira ainda está sendo processada | `409` `CONFLICT` — `a send with this idempotency key is in flight; retry shortly` |
| A primeira chamada falhou (erro `4xx`/`5xx`) | A chave é liberada: repita com a mesma chave |
| Chave com mais de 255 caracteres | `400` `VALIDATION_ERROR` |

Ao receber `409`, espere um instante e repita a mesma chamada — a resposta seguinte será o `202` da
mensagem original. O `409` existe para a API não confirmar um envio que ainda não foi gravado.

:::note
A idempotência vale apenas para `POST /messages/send`. Na rota de conversa a chave é gravada na
mensagem, mas chamadas repetidas não são agrupadas.
:::

### Responder citando {#citacao}

Para a mensagem sair como resposta a outra, envie em `metadata.reply_to_id` o **id da mensagem no
provedor** — o campo `external_id` da mensagem na API, ou `message.metadata.externalId` no evento
`message.received`. Não é o id interno do Linktor.

```json
{
  "channel_id": "CHANNEL_ID",
  "to": "+5511999999999",
  "text": "Sim, o pedido sai amanhã.",
  "metadata": {
    "reply_to_id": "wamid.HBgNNTUxMTk5OTk5OTk5ORUCABIYFjNFQjBDN0Q4RjQ5",
    "quoted_text": "Olá, quero saber do meu pedido"
  }
}
```

A citação é aplicada nos canais `whatsapp_official`, `whatsapp` (não oficial) e `telegram`. Nos demais,
a chave é gravada na mensagem e ignorada na entrega. `quoted_text` só é usado pelo WhatsApp não oficial.

### Erros do envio direto

| HTTP | `code` | Causa |
|---|---|---|
| `400` | `VALIDATION_ERROR` | `channel_id` ou `to` ausente; `text` vazio sem anexo; `content_type` não aceito; mídia sem `attachments`; anexo sem `url`; chave reservada em `metadata`; `idempotency_key` longa demais |
| `400` | `VALIDATION_ERROR` | Canal de sandbox e destinatário fora da lista de permitidos, ou número que não é E.164 |
| `500` | `CHANNEL_DISCONNECTED` | Canal desabilitado (`channel is not enabled`) ou desconectado (`channel is not connected`) — veja o aviso abaixo |
| `401` | `UNAUTHORIZED` | Chave ausente ou inválida |
| `403` | `FORBIDDEN` | Chave sem o scope `messages:send` |
| `404` | `CHANNEL_NOT_FOUND` | Canal inexistente ou de outro tenant |
| `409` | `CONFLICT` | Mesma `idempotency_key` ainda em processamento |
| `500` | `INTERNAL_ERROR` | Falha ao gravar ou ao enfileirar — pode repetir com a mesma `idempotency_key` |

:::warning
`CHANNEL_DISCONNECTED` responde com HTTP `500` nesta versão: o código de erro não tem um status HTTP
próprio mapeado no servidor. Identifique o caso pelo campo `error.code`, não pelo status, e **não**
repita a chamada automaticamente — o canal precisa ser habilitado ou reconectado antes.
:::

Nenhum desses erros deixa contato ou conversa criados pela metade: canal e sandbox são verificados
antes de qualquer gravação.

## Envio na conversa {#envio-na-conversa}

`POST /api/v1/conversations/:id/messages`

Use quando você já tem o `conversation_id` — tipicamente para responder a um `message.received`. O
destinatário é o contato da conversa e o canal é o da conversa.

### Corpo

| Campo | Tipo | Obrigatório | Descrição |
|---|---|---|---|
| `content` | texto | sim, exceto se houver anexo | Texto, legenda, ou o JSON do conteúdo interativo |
| `content_type` | texto | não | Padrão `text`; com anexo e sem o campo, usa o `type` do primeiro anexo |
| `attachments` | lista | não | Mesmo formato do envio direto — veja [Anexos](#anexos) |
| `metadata` | mapa texto → texto | não | Mesmas regras de [metadata](#metadata) |

Aqui o campo do texto chama-se `content` (não `text`).

### Tipos de conteúdo

O servidor grava o `content_type` como recebido; quem interpreta é o adaptador do canal. Um tipo que o
canal não suporta gera `message.failed` depois, não um erro na chamada.

| `content_type` | O que enviar | Observações |
|---|---|---|
| `text` | `content` com o texto | Todos os canais. No WhatsApp oficial, `metadata.preview_url: "true"` liga a prévia de link |
| `image`, `video`, `audio`, `document` | `attachments` com um item; `content` opcional como legenda | No WhatsApp não oficial, `metadata.ptt: "true"` envia o áudio como mensagem de voz |
| `location` | `metadata.latitude`, `metadata.longitude` e, opcionais, `metadata.name` e `metadata.address`; `content` não pode ficar vazio (use o nome do local) | WhatsApp oficial e não oficial |
| `template` | `metadata.template_name`, `metadata.template_language`, `metadata.template_components` | Só WhatsApp oficial — veja [Modelo no WhatsApp oficial](#modelo) |
| `interactive` | `content` com o JSON do conteúdo interativo | Formato depende do canal — veja [Botões e listas](#interativo) |

O que cada canal aceita está na tabela de recursos de cada página em
[Canais](../canais/visao-geral.md).

<Tabs groupId="linguagem">
<TabItem value="curl" label="cURL">

```bash
curl -X POST "$LINKTOR_URL/api/v1/conversations/$CONVERSATION_ID/messages" \
  -H "X-API-Key: $LINKTOR_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "content_type": "text",
    "content": "Olá! Já estou verificando o seu pedido.",
    "metadata": { "source": "erp" }
  }'
```

</TabItem>
<TabItem value="typescript" label="TypeScript">

```typescript
async function responder(conversationId: string, texto: string) {
  const res = await fetch(
    `${process.env.LINKTOR_URL}/api/v1/conversations/${conversationId}/messages`,
    {
      method: 'POST',
      headers: {
        'X-API-Key': process.env.LINKTOR_API_KEY!,
        'Content-Type': 'application/json',
      },
      body: JSON.stringify({
        content_type: 'text',
        content: texto,
        metadata: { source: 'erp' },
      }),
    },
  );

  const body = await res.json();
  if (res.status !== 201) {
    const err = body.error ?? body;
    throw new Error(`${res.status} ${err.code}: ${err.message}`);
  }
  return body.data; // a mensagem criada, com status "pending"
}

const mensagem = await responder(conversationId, 'Olá! Já estou verificando o seu pedido.');
console.log(mensagem.id, mensagem.status);
```

</TabItem>
<TabItem value="python" label="Python">

```python
import os
import requests


def responder(conversation_id: str, texto: str) -> dict:
    res = requests.post(
        f"{os.environ['LINKTOR_URL']}/api/v1/conversations/{conversation_id}/messages",
        headers={"X-API-Key": os.environ["LINKTOR_API_KEY"]},
        json={
            "content_type": "text",
            "content": texto,
            "metadata": {"source": "erp"},
        },
        timeout=30,
    )
    body = res.json()
    if res.status_code != 201:
        err = body.get("error") or body
        raise RuntimeError(f"{res.status_code} {err.get('code')}: {err.get('message')}")
    return body["data"]  # a mensagem criada, com status "pending"


mensagem = responder(conversation_id, "Olá! Já estou verificando o seu pedido.")
print(mensagem["id"], mensagem["status"])
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
	"net/http"
	"os"
)

type Message struct {
	ID             string `json:"id"`
	ConversationID string `json:"conversation_id"`
	ContentType    string `json:"content_type"`
	Content        string `json:"content"`
	Status         string `json:"status"`
}

func Responder(conversationID, texto string) (*Message, error) {
	payload, _ := json.Marshal(map[string]any{
		"content_type": "text",
		"content":      texto,
		"metadata":     map[string]string{"source": "erp"},
	})

	url := os.Getenv("LINKTOR_URL") + "/api/v1/conversations/" + conversationID + "/messages"
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(payload))
	if err != nil {
		return nil, err
	}
	req.Header.Set("X-API-Key", os.Getenv("LINKTOR_API_KEY"))
	req.Header.Set("Content-Type", "application/json")

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	raw, _ := io.ReadAll(res.Body)
	if res.StatusCode != http.StatusCreated {
		return nil, fmt.Errorf("%d %s", res.StatusCode, raw)
	}

	var out struct {
		Data Message `json:"data"`
	}
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, err
	}
	return &out.Data, nil // status "pending"
}
```

</TabItem>
<TabItem value="java" label="Java">

```java
import java.net.URI;
import java.net.http.HttpClient;
import java.net.http.HttpRequest;
import java.net.http.HttpResponse;

public class Responder {
    static final HttpClient HTTP = HttpClient.newHttpClient();

    /** Devolve o JSON da mensagem criada (envelope completo). */
    static String responder(String conversationId, String jsonBody) throws Exception {
        HttpRequest request = HttpRequest.newBuilder(URI.create(
                System.getenv("LINKTOR_URL") + "/api/v1/conversations/" + conversationId + "/messages"))
            .header("X-API-Key", System.getenv("LINKTOR_API_KEY"))
            .header("Content-Type", "application/json")
            .POST(HttpRequest.BodyPublishers.ofString(jsonBody))
            .build();

        HttpResponse<String> response = HTTP.send(request, HttpResponse.BodyHandlers.ofString());
        if (response.statusCode() != 201) {
            throw new RuntimeException(response.statusCode() + " " + response.body());
        }
        return response.body();
    }

    public static void main(String[] args) throws Exception {
        String body = """
            {
              "content_type": "text",
              "content": "Olá! Já estou verificando o seu pedido.",
              "metadata": { "source": "erp" }
            }
            """;
        System.out.println(responder(args[0], body));
    }
}
```

</TabItem>
<TabItem value="csharp" label="C#">

```csharp
using System.Net;
using System.Net.Http.Json;
using System.Text.Json;

var http = new HttpClient();
http.DefaultRequestHeaders.Add("X-API-Key", Environment.GetEnvironmentVariable("LINKTOR_API_KEY"));

async Task<JsonElement> Responder(string conversationId, string texto)
{
    var response = await http.PostAsJsonAsync(
        $"{Environment.GetEnvironmentVariable("LINKTOR_URL")}/api/v1/conversations/{conversationId}/messages",
        new
        {
            content_type = "text",
            content = texto,
            metadata = new Dictionary<string, string> { ["source"] = "erp" },
        });

    var body = await response.Content.ReadFromJsonAsync<JsonElement>();
    if (response.StatusCode != HttpStatusCode.Created)
    {
        throw new Exception($"{(int)response.StatusCode} {body}");
    }
    return body.GetProperty("data"); // a mensagem criada, com status "pending"
}

var mensagem = await Responder(conversationId, "Olá! Já estou verificando o seu pedido.");
Console.WriteLine($"{mensagem.GetProperty("id")} {mensagem.GetProperty("status")}");
```

</TabItem>
<TabItem value="php" label="PHP">

```php
<?php
require 'vendor/autoload.php';

function responder(string $conversationId, string $texto): array
{
    $http = new GuzzleHttp\Client(['http_errors' => false, 'timeout' => 30]);

    $res = $http->post(
        getenv('LINKTOR_URL') . "/api/v1/conversations/{$conversationId}/messages",
        [
            'headers' => ['X-API-Key' => getenv('LINKTOR_API_KEY')],
            'json' => [
                'content_type' => 'text',
                'content'      => $texto,
                'metadata'     => ['source' => 'erp'],
            ],
        ]
    );

    $body = json_decode((string) $res->getBody(), true);
    if ($res->getStatusCode() !== 201) {
        $err = $body['error'] ?? $body;
        throw new RuntimeException("{$res->getStatusCode()} {$err['code']}: {$err['message']}");
    }
    return $body['data']; // a mensagem criada, com status "pending"
}

$mensagem = responder($conversationId, 'Olá! Já estou verificando o seu pedido.');
echo "{$mensagem['id']} {$mensagem['status']}\n";
```

</TabItem>
<TabItem value="rust" label="Rust">

```rust
use serde_json::{json, Value};

async fn responder(conversation_id: &str, texto: &str) -> Result<Value, Box<dyn std::error::Error>> {
    let url = format!(
        "{}/api/v1/conversations/{}/messages",
        std::env::var("LINKTOR_URL")?,
        conversation_id
    );

    let res = reqwest::Client::new()
        .post(url)
        .header("X-API-Key", std::env::var("LINKTOR_API_KEY")?)
        .json(&json!({
            "content_type": "text",
            "content": texto,
            "metadata": { "source": "erp" }
        }))
        .send()
        .await?;

    let status = res.status();
    let body: Value = res.json().await?;
    if status.as_u16() != 201 {
        return Err(format!("{status} {body}").into());
    }
    Ok(body["data"].clone()) // a mensagem criada, com status "pending"
}
```

</TabItem>
</Tabs>

### Resposta

`201 Created` com a mensagem gravada:

```json
{
  "success": true,
  "data": {
    "id": "3f1d2c4b-6a7e-4b8c-9d0e-1f2a3b4c5d6e",
    "conversation_id": "c1f0f5a2-7f1e-4c58-8a44-2b7f5a1d9e03",
    "sender_type": "user",
    "sender_id": "c0ffee00-1234-4abc-9def-000000000001",
    "content_type": "text",
    "content": "Olá! Já estou verificando o seu pedido.",
    "metadata": { "source": "erp" },
    "status": "pending",
    "created_at": "2026-11-24T14:07:33.412Z"
  }
}
```

`status: "pending"` aqui equivale ao `queued` do envio direto: gravada e enfileirada, ainda não
entregue.

| HTTP | `code` | Causa |
|---|---|---|
| `400` | `VALIDATION_ERROR` | `content` vazio sem anexo; corpo inválido; chave reservada em `metadata`; destinatário fora da lista do sandbox |
| `403` | `FORBIDDEN` | Chave sem `messages:send` ou sem `conversations:write` |
| `404` | `CONVERSATION_NOT_FOUND` | Conversa inexistente ou de outro tenant |
| `404` | `CHANNEL_NOT_FOUND`, `CONTACT_NOT_FOUND` | O canal ou o contato da conversa não existe mais |

:::note
Diferente do envio direto, esta rota não recusa canal desabilitado ou desconectado na hora: a mensagem
é enfileirada e a falha aparece depois, em `message.failed`.
:::

## Anexos {#anexos}

O envio de mídia tem dois passos: subir o arquivo e referenciá-lo em `attachments`.

### 1. Subir o arquivo

`POST /api/v1/conversations/:id/attachments` — `multipart/form-data`, campo `file`. Scopes:
`messages:send` + `conversations:write`. Limite de **25 MB** por arquivo (`413` acima disso).

O arquivo é guardado sob o seu tenant e não fica preso à conversa do caminho: a URL devolvida pode ser
usada em qualquer envio, inclusive no envio direto. Para o envio direto sem conversa conhecida, use no
caminho o id de qualquer conversa do tenant.

Resposta (`201`):

```json
{
  "success": true,
  "data": {
    "url": "https://linktor.suaempresa.com/api/v1/media/attachments/5f2a9c0e-1111-4c2b-8d3e-9a7b6c5d4e3f/7c9e6679-7425-40de-944b-e07fc1f90ae7.pdf",
    "type": "document",
    "filename": "boleto.pdf",
    "mime_type": "application/pdf",
    "size_bytes": 48213
  }
}
```

`type` é derivado do tipo MIME: `image`, `video`, `audio` ou `document`. Se o armazenamento de anexos
não estiver configurado no servidor, a rota responde `503`. Detalhes sobre armazenamento e URLs em
[Mídia](./midia.md).

### 2. Enviar referenciando o anexo

Copie o objeto devolvido para dentro de `attachments`:

| Campo do anexo | Obrigatório | Descrição |
|---|---|---|
| `url` | sim | A `url` devolvida pelo upload |
| `type` | não | `image`, `video`, `audio` ou `document` |
| `filename` | não | Nome exibido ao destinatário (documentos) |
| `mime_type` | não | Tipo MIME |
| `size_bytes` | não | Tamanho em bytes |

O exemplo sobe um PDF e o envia pelo envio direto, com legenda:

<Tabs groupId="linguagem">
<TabItem value="curl" label="cURL">

```bash
# 1) Upload
curl -X POST "$LINKTOR_URL/api/v1/conversations/$CONVERSATION_ID/attachments" \
  -H "X-API-Key: $LINKTOR_API_KEY" \
  -F "file=@boleto.pdf;type=application/pdf"

# 2) Envio (cole a url devolvida no passo 1)
curl -X POST "$LINKTOR_URL/api/v1/messages/send" \
  -H "X-API-Key: $LINKTOR_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "channel_id": "CHANNEL_ID",
    "to": "+5511999999999",
    "content_type": "document",
    "text": "Segue o boleto do pedido 1234.",
    "attachments": [{
      "url": "URL_DEVOLVIDA_PELO_UPLOAD",
      "type": "document",
      "filename": "boleto.pdf",
      "mime_type": "application/pdf",
      "size_bytes": 48213
    }]
  }'
```

</TabItem>
<TabItem value="typescript" label="TypeScript">

```typescript
import { readFile } from 'node:fs/promises';

const BASE = `${process.env.LINKTOR_URL}/api/v1`;
const auth = { 'X-API-Key': process.env.LINKTOR_API_KEY! };

// 1) Upload — não defina Content-Type: o fetch gera o boundary do multipart
const form = new FormData();
form.append('file', new Blob([await readFile('boleto.pdf')], { type: 'application/pdf' }), 'boleto.pdf');

const up = await fetch(`${BASE}/conversations/${conversationId}/attachments`, {
  method: 'POST',
  headers: auth,
  body: form,
});
if (up.status !== 201) throw new Error(`upload: ${up.status} ${await up.text()}`);
const anexo = (await up.json()).data;

// 2) Envio
const res = await fetch(`${BASE}/messages/send`, {
  method: 'POST',
  headers: { ...auth, 'Content-Type': 'application/json' },
  body: JSON.stringify({
    channel_id: 'CHANNEL_ID',
    to: '+5511999999999',
    content_type: anexo.type,
    text: 'Segue o boleto do pedido 1234.',
    attachments: [anexo],
  }),
});
if (res.status !== 202) throw new Error(`envio: ${res.status} ${await res.text()}`);
console.log((await res.json()).data.id);
```

</TabItem>
<TabItem value="python" label="Python">

```python
import os
import requests

BASE = f"{os.environ['LINKTOR_URL']}/api/v1"
auth = {"X-API-Key": os.environ["LINKTOR_API_KEY"]}

# 1) Upload
with open("boleto.pdf", "rb") as arquivo:
    up = requests.post(
        f"{BASE}/conversations/{conversation_id}/attachments",
        headers=auth,
        files={"file": ("boleto.pdf", arquivo, "application/pdf")},
        timeout=60,
    )
up.raise_for_status()
anexo = up.json()["data"]

# 2) Envio
res = requests.post(
    f"{BASE}/messages/send",
    headers=auth,
    json={
        "channel_id": "CHANNEL_ID",
        "to": "+5511999999999",
        "content_type": anexo["type"],
        "text": "Segue o boleto do pedido 1234.",
        "attachments": [anexo],
    },
    timeout=30,
)
if res.status_code != 202:
    raise RuntimeError(f"{res.status_code} {res.text}")
print(res.json()["data"]["id"])
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
	"mime/multipart"
	"net/http"
	"net/textproto"
	"os"
)

type Attachment struct {
	URL       string `json:"url"`
	Type      string `json:"type"`
	Filename  string `json:"filename"`
	MimeType  string `json:"mime_type"`
	SizeBytes int64  `json:"size_bytes"`
}

func do(req *http.Request, want int, out any) error {
	req.Header.Set("X-API-Key", os.Getenv("LINKTOR_API_KEY"))
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	raw, _ := io.ReadAll(res.Body)
	if res.StatusCode != want {
		return fmt.Errorf("%d %s", res.StatusCode, raw)
	}
	return json.Unmarshal(raw, out)
}

func EnviarPDF(conversationID, caminho string) (string, error) {
	base := os.Getenv("LINKTOR_URL") + "/api/v1"

	// 1) Upload
	data, err := os.ReadFile(caminho)
	if err != nil {
		return "", err
	}
	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	h := textproto.MIMEHeader{}
	h.Set("Content-Disposition", `form-data; name="file"; filename="boleto.pdf"`)
	h.Set("Content-Type", "application/pdf")
	part, _ := w.CreatePart(h)
	part.Write(data)
	w.Close()

	up, _ := http.NewRequest(http.MethodPost, base+"/conversations/"+conversationID+"/attachments", &buf)
	up.Header.Set("Content-Type", w.FormDataContentType())
	var uploaded struct {
		Data Attachment `json:"data"`
	}
	if err := do(up, http.StatusCreated, &uploaded); err != nil {
		return "", err
	}

	// 2) Envio
	payload, _ := json.Marshal(map[string]any{
		"channel_id":   "CHANNEL_ID",
		"to":           "+5511999999999",
		"content_type": uploaded.Data.Type,
		"text":         "Segue o boleto do pedido 1234.",
		"attachments":  []Attachment{uploaded.Data},
	})
	send, _ := http.NewRequest(http.MethodPost, base+"/messages/send", bytes.NewReader(payload))
	send.Header.Set("Content-Type", "application/json")
	var sent struct {
		Data struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	if err := do(send, http.StatusAccepted, &sent); err != nil {
		return "", err
	}
	return sent.Data.ID, nil
}
```

</TabItem>
<TabItem value="java" label="Java">

```java
import java.io.ByteArrayOutputStream;
import java.net.URI;
import java.net.http.HttpClient;
import java.net.http.HttpRequest;
import java.net.http.HttpResponse;
import java.nio.charset.StandardCharsets;
import java.nio.file.Files;
import java.nio.file.Path;

public class EnviarAnexo {
    static final String BASE = System.getenv("LINKTOR_URL") + "/api/v1";
    static final String KEY = System.getenv("LINKTOR_API_KEY");
    static final HttpClient HTTP = HttpClient.newHttpClient();

    /** 1) Upload: java.net.http não monta multipart, então o corpo é montado à mão. */
    static String upload(String conversationId, Path arquivo, String mime) throws Exception {
        String boundary = "----linktor" + System.nanoTime();
        ByteArrayOutputStream body = new ByteArrayOutputStream();
        body.write(("--" + boundary + "\r\n"
            + "Content-Disposition: form-data; name=\"file\"; filename=\"" + arquivo.getFileName() + "\"\r\n"
            + "Content-Type: " + mime + "\r\n\r\n").getBytes(StandardCharsets.UTF_8));
        body.write(Files.readAllBytes(arquivo));
        body.write(("\r\n--" + boundary + "--\r\n").getBytes(StandardCharsets.UTF_8));

        HttpRequest request = HttpRequest.newBuilder(
                URI.create(BASE + "/conversations/" + conversationId + "/attachments"))
            .header("X-API-Key", KEY)
            .header("Content-Type", "multipart/form-data; boundary=" + boundary)
            .POST(HttpRequest.BodyPublishers.ofByteArray(body.toByteArray()))
            .build();

        HttpResponse<String> response = HTTP.send(request, HttpResponse.BodyHandlers.ofString());
        if (response.statusCode() != 201) {
            throw new RuntimeException("upload: " + response.statusCode() + " " + response.body());
        }
        return response.body(); // {"success":true,"data":{"url":...,"type":...}}
    }

    /** 2) Envio: anexoJson é o objeto "data" devolvido pelo upload. */
    static String enviar(String anexoJson) throws Exception {
        String json = """
            {
              "channel_id": "CHANNEL_ID",
              "to": "+5511999999999",
              "content_type": "document",
              "text": "Segue o boleto do pedido 1234.",
              "attachments": [%s]
            }
            """.formatted(anexoJson);

        HttpRequest request = HttpRequest.newBuilder(URI.create(BASE + "/messages/send"))
            .header("X-API-Key", KEY)
            .header("Content-Type", "application/json")
            .POST(HttpRequest.BodyPublishers.ofString(json))
            .build();

        HttpResponse<String> response = HTTP.send(request, HttpResponse.BodyHandlers.ofString());
        if (response.statusCode() != 202) {
            throw new RuntimeException("envio: " + response.statusCode() + " " + response.body());
        }
        return response.body();
    }
}
```

Extraia o objeto `data` da resposta do upload com a biblioteca JSON do seu projeto (Jackson, Gson)
antes de passá-lo a `enviar`.

</TabItem>
<TabItem value="csharp" label="C#">

```csharp
using System.Net;
using System.Net.Http.Headers;
using System.Net.Http.Json;
using System.Text.Json;

var baseUrl = Environment.GetEnvironmentVariable("LINKTOR_URL") + "/api/v1";
var http = new HttpClient();
http.DefaultRequestHeaders.Add("X-API-Key", Environment.GetEnvironmentVariable("LINKTOR_API_KEY"));

// 1) Upload
using var form = new MultipartFormDataContent();
var arquivo = new ByteArrayContent(await File.ReadAllBytesAsync("boleto.pdf"));
arquivo.Headers.ContentType = new MediaTypeHeaderValue("application/pdf");
form.Add(arquivo, "file", "boleto.pdf");

var up = await http.PostAsync($"{baseUrl}/conversations/{conversationId}/attachments", form);
if (up.StatusCode != HttpStatusCode.Created)
{
    throw new Exception($"upload: {(int)up.StatusCode} {await up.Content.ReadAsStringAsync()}");
}
var anexo = (await up.Content.ReadFromJsonAsync<JsonElement>()).GetProperty("data");

// 2) Envio
var res = await http.PostAsJsonAsync($"{baseUrl}/messages/send", new
{
    channel_id = "CHANNEL_ID",
    to = "+5511999999999",
    content_type = anexo.GetProperty("type").GetString(),
    text = "Segue o boleto do pedido 1234.",
    attachments = new[] { anexo },
});
if (res.StatusCode != HttpStatusCode.Accepted)
{
    throw new Exception($"envio: {(int)res.StatusCode} {await res.Content.ReadAsStringAsync()}");
}
var enviado = await res.Content.ReadFromJsonAsync<JsonElement>();
Console.WriteLine(enviado.GetProperty("data").GetProperty("id"));
```

</TabItem>
<TabItem value="php" label="PHP">

```php
<?php
require 'vendor/autoload.php';

$base = getenv('LINKTOR_URL') . '/api/v1';
$http = new GuzzleHttp\Client([
    'http_errors' => false,
    'timeout'     => 60,
    'headers'     => ['X-API-Key' => getenv('LINKTOR_API_KEY')],
]);

// 1) Upload
$up = $http->post("{$base}/conversations/{$conversationId}/attachments", [
    'multipart' => [[
        'name'     => 'file',
        'contents' => fopen('boleto.pdf', 'r'),
        'filename' => 'boleto.pdf',
        'headers'  => ['Content-Type' => 'application/pdf'],
    ]],
]);
if ($up->getStatusCode() !== 201) {
    throw new RuntimeException("upload: {$up->getStatusCode()} {$up->getBody()}");
}
$anexo = json_decode((string) $up->getBody(), true)['data'];

// 2) Envio
$res = $http->post("{$base}/messages/send", [
    'json' => [
        'channel_id'   => 'CHANNEL_ID',
        'to'           => '+5511999999999',
        'content_type' => $anexo['type'],
        'text'         => 'Segue o boleto do pedido 1234.',
        'attachments'  => [$anexo],
    ],
]);
if ($res->getStatusCode() !== 202) {
    throw new RuntimeException("envio: {$res->getStatusCode()} {$res->getBody()}");
}
echo json_decode((string) $res->getBody(), true)['data']['id'], "\n";
```

</TabItem>
<TabItem value="rust" label="Rust">

```rust
// Cargo.toml: reqwest = { version = "0.12", features = ["json", "multipart"] }
use reqwest::multipart::{Form, Part};
use serde_json::{json, Value};

async fn enviar_pdf(conversation_id: &str) -> Result<String, Box<dyn std::error::Error>> {
    let base = format!("{}/api/v1", std::env::var("LINKTOR_URL")?);
    let key = std::env::var("LINKTOR_API_KEY")?;
    let http = reqwest::Client::new();

    // 1) Upload
    let part = Part::bytes(tokio::fs::read("boleto.pdf").await?)
        .file_name("boleto.pdf")
        .mime_str("application/pdf")?;

    let up = http
        .post(format!("{base}/conversations/{conversation_id}/attachments"))
        .header("X-API-Key", &key)
        .multipart(Form::new().part("file", part))
        .send()
        .await?;
    if up.status().as_u16() != 201 {
        return Err(format!("upload: {} {}", up.status(), up.text().await?).into());
    }
    let anexo = up.json::<Value>().await?["data"].clone();

    // 2) Envio
    let res = http
        .post(format!("{base}/messages/send"))
        .header("X-API-Key", &key)
        .json(&json!({
            "channel_id": "CHANNEL_ID",
            "to": "+5511999999999",
            "content_type": anexo["type"],
            "text": "Segue o boleto do pedido 1234.",
            "attachments": [anexo]
        }))
        .send()
        .await?;
    if res.status().as_u16() != 202 {
        return Err(format!("envio: {} {}", res.status(), res.text().await?).into());
    }
    let body: Value = res.json().await?;
    Ok(body["data"]["id"].as_str().unwrap_or_default().to_string())
}
```

</TabItem>
</Tabs>

Na rota de conversa o corpo é o mesmo, trocando `text` por `content` e retirando `channel_id` e `to`.

:::note Um anexo por mensagem
A lista aceita vários itens e todos são gravados, mas os adaptadores de WhatsApp entregam apenas o
primeiro. Para enviar várias mídias, faça um envio por arquivo.
:::

## Modelo no WhatsApp oficial {#modelo}

Fora da janela de atendimento de 24 horas, o WhatsApp oficial só aceita modelos (templates) aprovados
pela Meta. Envie pela rota de conversa com `content_type: "template"` e os dados do modelo em
`metadata`:

| Chave em `metadata` | Obrigatório | Descrição |
|---|---|---|
| `template_name` | sim | Nome do modelo aprovado |
| `template_language` | não | Código do idioma do modelo (ex.: `pt_BR`). **Padrão `en`** — informe sempre |
| `template_components` | não | Texto JSON com a lista de componentes e parâmetros, no formato da Cloud API |

```bash
curl -X POST "$LINKTOR_URL/api/v1/conversations/$CONVERSATION_ID/messages" \
  -H "X-API-Key: $LINKTOR_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "content_type": "template",
    "content": "Olá Maria, seu pedido 1234 foi confirmado.",
    "metadata": {
      "template_name": "pedido_confirmado",
      "template_language": "pt_BR",
      "template_components": "[{\"type\":\"body\",\"parameters\":[{\"type\":\"text\",\"text\":\"Maria\"},{\"type\":\"text\",\"text\":\"1234\"}]}]"
    }
  }'
```

Pontos de atenção:

- `metadata` só aceita texto, então `template_components` é um **texto contendo JSON** (com as aspas
  escapadas), não um objeto.
- `content` é obrigatório e não é enviado ao destinatário: é o que fica registrado no histórico da
  conversa. Escreva nele o texto do modelo já preenchido.
- Cada componente tem `type` (`header`, `body`, `button`), e os de botão levam também `sub_type` e
  `index`. Cada parâmetro tem `type` e o valor correspondente (`text`, `payload` para botão de resposta
  rápida, `coupon_code` para botão de copiar código).
- O envio direto não aceita `template`. Para a primeira mensagem a um número novo, crie a conversa
  (veja [Conversas](./conversas.md)) e envie o modelo nela, ou use uma
  [campanha](./modelos-e-campanhas.md).

Como criar, sincronizar e consultar modelos está em [Modelos e campanhas](./modelos-e-campanhas.md).

## Botões e listas {#interativo}

Use `content_type: "interactive"` na rota de conversa. O `content` carrega o conteúdo interativo como
**texto JSON**, e o formato depende do canal.

**WhatsApp não oficial** — `variant` é `buttons` ou `list`:

```json
{
  "content_type": "interactive",
  "content": "{\"variant\":\"buttons\",\"body\":\"Como podemos ajudar?\",\"footer\":\"Atendimento\",\"buttons\":[{\"id\":\"rastreio\",\"title\":\"Rastrear pedido\"},{\"id\":\"atendente\",\"title\":\"Falar com atendente\"}]}"
}
```

Para `list`, use `button_text` e `sections` (cada seção com `title` e `rows` de `id`, `title`,
`description`). Nesse canal a lista é entregue como texto numerado, não como lista nativa — detalhes em
[WhatsApp não oficial](../canais/whatsapp-nao-oficial.md).

**WhatsApp oficial** — `content` é o objeto `interactive` da Cloud API e `metadata.interactive_type` é
obrigatório:

```json
{
  "content_type": "interactive",
  "content": "{\"type\":\"button\",\"body\":{\"text\":\"Como podemos ajudar?\"},\"action\":{\"buttons\":[{\"type\":\"reply\",\"reply\":{\"id\":\"rastreio\",\"title\":\"Rastrear pedido\"}}]}}",
  "metadata": { "interactive_type": "button" }
}
```

O clique do contato chega como `message.received` normal.

## Reações {#reacoes}

`POST /api/v1/conversations/:id/messages/:messageId/reactions` — scopes `messages:send` +
`conversations:write`.

`:messageId` é o id da mensagem **no Linktor**. Envie `emoji` vazio para remover a reação.

```bash
# Reagir
curl -X POST "$LINKTOR_URL/api/v1/conversations/$CONVERSATION_ID/messages/$MESSAGE_ID/reactions" \
  -H "X-API-Key: $LINKTOR_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{"emoji": "👍"}'

# Remover a reação
curl -X POST "$LINKTOR_URL/api/v1/conversations/$CONVERSATION_ID/messages/$MESSAGE_ID/reactions" \
  -H "X-API-Key: $LINKTOR_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{"emoji": ""}'
```

Resposta (`200`):

```json
{
  "success": true,
  "data": {
    "message": "Reaction added successfully",
    "message_id": "0b0f4c1e-6a55-4a0e-9d3b-0c9a4f2a7e11",
    "emoji": "👍"
  }
}
```

A reação é gravada na mensagem (campo `reactions`) e enfileirada para o canal. Se a mensagem alvo não
tem `external_id` — nunca chegou ao provedor —, a reação fica só registrada no Linktor. `404`
(`MESSAGE_NOT_FOUND`) quando a mensagem não pertence à conversa informada.

## Enviar para um grupo {#grupo}

`POST /api/v1/channels/:id/groups/:groupId/messages` — scope `messages:send`.

Envia texto direto ao identificador de um grupo do **WhatsApp não oficial** (JID terminado em
`@g.us`), sem passar por contato ou conversa.

| Campo | Tipo | Obrigatório | Descrição |
|---|---|---|---|
| `text` | texto | sim | Texto da mensagem |
| `metadata` | mapa texto → texto | não | Mesmas regras de [metadata](#metadata) |

```bash
curl -X POST "$LINKTOR_URL/api/v1/channels/CHANNEL_ID/groups/120363041234567890@g.us/messages" \
  -H "X-API-Key: $LINKTOR_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{"text": "Lembrete: reunião às 15h."}'
```

Resposta (`202`):

```json
{
  "success": true,
  "data": {
    "id": "6e1f0a9b-8c7d-4e6f-a5b4-c3d2e1f0a9b8",
    "recipient": "120363041234567890@g.us",
    "status": "queued"
  }
}
```

:::warning Diferenças em relação às outras rotas
- Só texto: não há `attachments` nem `content_type`.
- A mensagem **não** é gravada em uma conversa. O `id` devolvido é o id do envio na fila, e
  `GET /messages/:id` não o encontra.
- Não há `idempotency_key`: repetir a chamada envia de novo.
- O canal é validado quanto ao tenant, mas não quanto a estar conectado; a falha de entrega aparece
  depois.
:::

O id do grupo aparece nos eventos de mensagens recebidas em grupo — veja
[WhatsApp não oficial](../canais/whatsapp-nao-oficial.md).

## Acompanhar o status {#status}

Uma mensagem passa por estes estados:

| Estado | Significado |
|---|---|
| `pending` (`queued` no envio direto) | Gravada e na fila de saída |
| `sent` | O provedor aceitou a mensagem; `external_id` passa a existir |
| `delivered` | Chegou ao aparelho do destinatário (quando o canal informa) |
| `read` | Foi lida (quando o canal informa) |
| `failed` | A entrega falhou; `error_message` traz o motivo |

### Por webhook (recomendado)

Configure o webhook de saída do canal e trate os eventos `message.sent`, `message.delivered`,
`message.read` e `message.failed`. O campo `data.messageId` é o `id` que a rota de envio devolveu.
Formato dos eventos, assinatura e política de reentrega em [Receber eventos](./receber-eventos.md).

### Por consulta

`GET /api/v1/messages/:id` devolve a mensagem com o estado atual. A rota exige autenticação, mas não
tem scope próprio.

```bash
curl "$LINKTOR_URL/api/v1/messages/$MESSAGE_ID" -H "X-API-Key: $LINKTOR_API_KEY"
```

```json
{
  "success": true,
  "data": {
    "id": "9a8b7c6d-5e4f-4a3b-9c2d-1e0f9a8b7c6d",
    "conversation_id": "c1f0f5a2-7f1e-4c58-8a44-2b7f5a1d9e03",
    "sender_type": "system",
    "content_type": "text",
    "content": "Seu pedido 1234 saiu para entrega.",
    "metadata": { "source": "erp", "idempotency_key": "pedido-1234-saiu-para-entrega" },
    "status": "delivered",
    "external_id": "wamid.HBgNNTUxMTk5OTk5OTk5ORUCABEYEjdBNUM2RkE0QjM5",
    "sent_at": "2026-11-24T14:03:13Z",
    "delivered_at": "2026-11-24T14:03:15Z",
    "created_at": "2026-11-24T14:03:11Z"
  }
}
```

Mensagens enviadas pelo envio direto têm `sender_type: "system"`; as enviadas pela rota de conversa,
`sender_type: "user"`. Use a consulta para reconciliar, não para fazer polling contínuo: o limite de
100 requisições por minuto é compartilhado por todo o tenant.

Para ler o histórico completo de uma conversa, veja [Conversas](./conversas.md).

## Canal de sandbox

Em um canal criado no ambiente de sandbox, o envio só é aceito para números cadastrados na lista de
permitidos do tenant. Fora dela, a chamada devolve `400` (`VALIDATION_ERROR`) com o número mascarado na
mensagem, e nada é gravado. Veja [Sandbox](../painel/sandbox.md).

## Resumo das diferenças entre as rotas

| | Envio direto | Na conversa | Grupo |
|---|---|---|---|
| Campo do texto | `text` (ou `content`) | `content` | `text` |
| Sucesso | `202` | `201` | `202` |
| Corpo da resposta | `id`, `conversation_id`, `channel_id`, `status` | a mensagem completa | `id`, `recipient`, `status` |
| Tipos aceitos | `text` e mídia | todos | só texto |
| Idempotência | sim | não | não |
| Recusa canal desconectado na hora | sim | não | não |
| Grava mensagem consultável | sim | sim | não |
