---
sidebar_position: 6
title: Contatos
description: Criar, listar, buscar, atualizar e remover contatos pela API, e gerenciar as identidades do contato em cada tipo de canal.
---

# Contatos

Um contato é a pessoa (ou empresa) com quem você conversa. Ele tem dados cadastrais — nome, telefone,
e-mail, campos personalizados, etiquetas — e uma ou mais **identidades**: o endereço dele em cada
tipo de canal (o número no WhatsApp, o id no Telegram, o e-mail…).

O Linktor cria contato e identidade sozinho quando chega uma mensagem ou quando você usa o
[envio direto](./enviar-mensagens.md). Use as rotas desta página para sincronizar seu cadastro com o
Linktor.

## Rotas

| Método | Caminho | Scope da API key | O que faz |
|---|---|---|---|
| `GET` | `/contacts` | `contacts:read` | Lista e busca |
| `POST` | `/contacts` | `contacts:write` | Cria |
| `GET` | `/contacts/:id` | `contacts:read` | Obtém, com as identidades |
| `PUT` | `/contacts/:id` | `contacts:write` | Atualiza (substitui os dados cadastrais) |
| `DELETE` | `/contacts/:id` | `contacts:write` | Remove |
| `POST` | `/contacts/:id/identities` | `contacts:write` | Adiciona uma identidade |
| `DELETE` | `/contacts/:id/identities/:identityId` | `contacts:write` | Remove uma identidade |

`contacts:write` implica `contacts:read`. Sessões de usuário (JWT/cookie) não usam scopes. Veja
[Autenticação](./autenticacao.md).

## O objeto contato

```json
{
  "id": "0b9d5a3e-1c7f-4e2a-8d46-9f1a2b3c4d5e",
  "tenant_id": "b7a1d0c4-2f6e-4f0a-9d58-3f2b7c1e9a10",
  "name": "Maria Souza",
  "email": "cliente@exemplo.com",
  "phone": "+5511999999999",
  "avatar_url": "https://exemplo.com/fotos/maria.jpg",
  "custom_fields": { "codigo_cliente": "C-1042", "plano": "ouro" },
  "tags": ["vip", "sp"],
  "identities": [
    {
      "id": "9a8b7c6d-5e4f-4a3b-2c1d-0e9f8a7b6c5d",
      "contact_id": "0b9d5a3e-1c7f-4e2a-8d46-9f1a2b3c4d5e",
      "tenant_id": "b7a1d0c4-2f6e-4f0a-9d58-3f2b7c1e9a10",
      "channel_type": "whatsapp_official",
      "identifier": "5511999999999",
      "created_at": "2026-11-24T13:38:47Z"
    }
  ],
  "created_at": "2026-11-24T13:38:47Z",
  "updated_at": "2026-11-24T13:38:47Z"
}
```

| Campo | Observação |
|---|---|
| `custom_fields` | Mapa de texto para texto (`string → string`). Números e booleanos precisam ir como texto |
| `tags` | Lista de textos |
| `identities` | Vem em `GET /contacts/:id` e nas respostas das rotas de identidade. **Não** vem na listagem nem nas respostas de criar e atualizar |
| Campos vazios | `name`, `email`, `phone`, `avatar_url`, `custom_fields`, `tags` e `identities` são omitidos do JSON quando vazios |

## Criar um contato

| Campo | Obrigatório | Descrição |
|---|---|---|
| `name` | sim | Nome de exibição |
| `email` | não | Único por tenant |
| `phone` | não | Único por tenant. Use o formato internacional (`+5511999999999`) |
| `avatar_url` | não | URL da foto |
| `custom_fields` | não | Mapa `string → string` |
| `tags` | não | Lista de etiquetas |

<Tabs groupId="linguagem">
<TabItem value="curl" label="cURL">

```bash
curl -X POST "$LINKTOR_URL/api/v1/contacts" \
  -H "X-API-Key: $LINKTOR_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Maria Souza",
    "email": "cliente@exemplo.com",
    "phone": "+5511999999999",
    "custom_fields": { "codigo_cliente": "C-1042" },
    "tags": ["vip"]
  }'
```

</TabItem>
<TabItem value="typescript" label="TypeScript">

```typescript
const base = `${process.env.LINKTOR_URL}/api/v1`;

const res = await fetch(`${base}/contacts`, {
  method: 'POST',
  headers: {
    'X-API-Key': process.env.LINKTOR_API_KEY!,
    'Content-Type': 'application/json',
  },
  body: JSON.stringify({
    name: 'Maria Souza',
    email: 'cliente@exemplo.com',
    phone: '+5511999999999',
    custom_fields: { codigo_cliente: 'C-1042' },
    tags: ['vip'],
  }),
});

const body = await res.json();
if (res.status === 409) {
  // Já existe contato com esse e-mail ou telefone
  console.warn(body.error.message);
} else if (!res.ok) {
  throw new Error(`${body.error?.code}: ${body.error?.message}`);
} else {
  console.log('Contato criado:', body.data.id);
}
```

</TabItem>
<TabItem value="python" label="Python">

```python
import os
import requests

BASE = f"{os.environ['LINKTOR_URL']}/api/v1"
HEADERS = {"X-API-Key": os.environ["LINKTOR_API_KEY"]}

res = requests.post(
    f"{BASE}/contacts",
    headers=HEADERS,
    json={
        "name": "Maria Souza",
        "email": "cliente@exemplo.com",
        "phone": "+5511999999999",
        "custom_fields": {"codigo_cliente": "C-1042"},
        "tags": ["vip"],
    },
    timeout=30,
)

body = res.json()
if res.status_code == 409:
    # Já existe contato com esse e-mail ou telefone
    print(body["error"]["message"])
else:
    res.raise_for_status()
    print("Contato criado:", body["data"]["id"])
```

</TabItem>
<TabItem value="go" label="Go">

```go
package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
)

func main() {
	payload, _ := json.Marshal(map[string]any{
		"name":          "Maria Souza",
		"email":         "cliente@exemplo.com",
		"phone":         "+5511999999999",
		"custom_fields": map[string]string{"codigo_cliente": "C-1042"},
		"tags":          []string{"vip"},
	})

	req, _ := http.NewRequest(http.MethodPost,
		os.Getenv("LINKTOR_URL")+"/api/v1/contacts", bytes.NewReader(payload))
	req.Header.Set("X-API-Key", os.Getenv("LINKTOR_API_KEY"))
	req.Header.Set("Content-Type", "application/json")

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		panic(err)
	}
	defer res.Body.Close()

	var body struct {
		Data struct {
			ID string `json:"id"`
		} `json:"data"`
		Error *struct {
			Code    string `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.NewDecoder(res.Body).Decode(&body); err != nil {
		panic(err)
	}

	switch {
	case res.StatusCode == http.StatusConflict:
		fmt.Println("já existe:", body.Error.Message)
	case res.StatusCode != http.StatusCreated:
		panic(fmt.Sprintf("%s: %s", body.Error.Code, body.Error.Message))
	default:
		fmt.Println("Contato criado:", body.Data.ID)
	}
}
```

</TabItem>
<TabItem value="java" label="Java">

```java
import com.fasterxml.jackson.databind.JsonNode;
import com.fasterxml.jackson.databind.ObjectMapper;
import java.net.URI;
import java.net.http.HttpClient;
import java.net.http.HttpRequest;
import java.net.http.HttpResponse;
import java.util.List;
import java.util.Map;

public class CriarContato {
    public static void main(String[] args) throws Exception {
        ObjectMapper json = new ObjectMapper();

        String payload = json.writeValueAsString(Map.of(
            "name", "Maria Souza",
            "email", "cliente@exemplo.com",
            "phone", "+5511999999999",
            "custom_fields", Map.of("codigo_cliente", "C-1042"),
            "tags", List.of("vip")
        ));

        HttpRequest req = HttpRequest.newBuilder(
                URI.create(System.getenv("LINKTOR_URL") + "/api/v1/contacts"))
            .header("X-API-Key", System.getenv("LINKTOR_API_KEY"))
            .header("Content-Type", "application/json")
            .POST(HttpRequest.BodyPublishers.ofString(payload))
            .build();

        HttpResponse<String> res = HttpClient.newHttpClient()
            .send(req, HttpResponse.BodyHandlers.ofString());
        JsonNode body = json.readTree(res.body());

        if (res.statusCode() == 409) {
            System.out.println("Já existe: " + body.path("error").path("message").asText());
        } else if (res.statusCode() != 201) {
            throw new IllegalStateException(res.body());
        } else {
            System.out.println("Contato criado: " + body.path("data").path("id").asText());
        }
    }
}
```

</TabItem>
<TabItem value="csharp" label="C#">

```csharp
using System.Net;
using System.Net.Http.Json;
using System.Text.Json;

var http = new HttpClient { BaseAddress = new Uri(Environment.GetEnvironmentVariable("LINKTOR_URL") + "/api/v1/") };
http.DefaultRequestHeaders.Add("X-API-Key", Environment.GetEnvironmentVariable("LINKTOR_API_KEY"));

using var res = await http.PostAsJsonAsync("contacts", new
{
    name = "Maria Souza",
    email = "cliente@exemplo.com",
    phone = "+5511999999999",
    custom_fields = new Dictionary<string, string> { ["codigo_cliente"] = "C-1042" },
    tags = new[] { "vip" },
});

using var doc = JsonDocument.Parse(await res.Content.ReadAsStringAsync());
var root = doc.RootElement;

if (res.StatusCode == HttpStatusCode.Conflict)
{
    Console.WriteLine("Já existe: " + root.GetProperty("error").GetProperty("message").GetString());
}
else
{
    res.EnsureSuccessStatusCode();
    Console.WriteLine("Contato criado: " + root.GetProperty("data").GetProperty("id").GetString());
}
```

</TabItem>
<TabItem value="php" label="PHP">

```php
<?php
require 'vendor/autoload.php';

use GuzzleHttp\Client;

$http = new Client([
    'base_uri'    => getenv('LINKTOR_URL') . '/api/v1/',
    'headers'     => ['X-API-Key' => getenv('LINKTOR_API_KEY')],
    'http_errors' => false,
]);

$res = $http->post('contacts', ['json' => [
    'name'          => 'Maria Souza',
    'email'         => 'cliente@exemplo.com',
    'phone'         => '+5511999999999',
    'custom_fields' => ['codigo_cliente' => 'C-1042'],
    'tags'          => ['vip'],
]]);

$body = json_decode((string) $res->getBody(), true);

if ($res->getStatusCode() === 409) {
    echo 'Já existe: ' . $body['error']['message'] . PHP_EOL;
} elseif ($res->getStatusCode() !== 201) {
    throw new RuntimeException($body['error']['code'] . ': ' . $body['error']['message']);
} else {
    echo 'Contato criado: ' . $body['data']['id'] . PHP_EOL;
}
```

</TabItem>
<TabItem value="rust" label="Rust">

```rust
use reqwest::StatusCode;
use serde_json::{json, Value};

#[tokio::main]
async fn main() -> Result<(), Box<dyn std::error::Error>> {
    let base = format!("{}/api/v1", std::env::var("LINKTOR_URL")?);

    let res = reqwest::Client::new()
        .post(format!("{base}/contacts"))
        .header("X-API-Key", std::env::var("LINKTOR_API_KEY")?)
        .json(&json!({
            "name": "Maria Souza",
            "email": "cliente@exemplo.com",
            "phone": "+5511999999999",
            "custom_fields": { "codigo_cliente": "C-1042" },
            "tags": ["vip"]
        }))
        .send()
        .await?;

    let status = res.status();
    let body: Value = res.json().await?;

    match status {
        StatusCode::CREATED => println!("Contato criado: {}", body["data"]["id"]),
        StatusCode::CONFLICT => println!("Já existe: {}", body["error"]["message"]),
        _ => return Err(format!("{}: {}", body["error"]["code"], body["error"]["message"]).into()),
    }
    Ok(())
}
```

</TabItem>
</Tabs>

:::note Por que HTTP nativo aqui
Os SDKs têm um recurso `contacts`, mas os modelos dele ainda estão em camelCase e não foram
reconciliados com os campos reais (`custom_fields`, `avatar_url`…). Até lá, use o cliente HTTP da
linguagem nas rotas desta página.
:::

Resposta `201`:

```json
{
  "success": true,
  "data": {
    "id": "0b9d5a3e-1c7f-4e2a-8d46-9f1a2b3c4d5e",
    "tenant_id": "b7a1d0c4-2f6e-4f0a-9d58-3f2b7c1e9a10",
    "name": "Maria Souza",
    "email": "cliente@exemplo.com",
    "phone": "+5511999999999",
    "custom_fields": { "codigo_cliente": "C-1042" },
    "tags": ["vip"],
    "created_at": "2026-11-24T13:38:47Z",
    "updated_at": "2026-11-24T13:38:47Z"
  }
}
```

Erros:

| Status | `error.code` | `error.message` |
|---|---|---|
| `400` | `VALIDATION_ERROR` | `contact name is required` ou `Invalid request body` |
| `409` | `CONFLICT` | `contact with this email already exists` / `contact with this phone already exists` |

:::tip Sincronização sem duplicar
Não há rota de "criar ou atualizar". Para sincronizar um cadastro, tente criar; no `409`, localize o
contato com `GET /contacts?search=<telefone ou e-mail>` e atualize-o com `PUT`.
:::

## Listar e buscar

`GET /contacts` pagina **por número de página** e ordena do mais recente para o mais antigo.

| Parâmetro | Padrão | Descrição |
|---|---|---|
| `page` | `1` | Página, começando em 1 |
| `page_size` | `20` | Itens por página. Máximo `100` — acima disso a API devolve 100 itens, mas o `meta` ecoa o valor pedido; não passe de 100 |
| `search` | — | Trecho contido no nome, no e-mail ou no telefone (sem diferenciar maiúsculas) |

```bash
curl -G "$LINKTOR_URL/api/v1/contacts" \
  -H "X-API-Key: $LINKTOR_API_KEY" \
  --data-urlencode "search=maria" \
  --data-urlencode "page=1" \
  --data-urlencode "page_size=50"
```

```json
{
  "success": true,
  "data": [
    {
      "id": "0b9d5a3e-1c7f-4e2a-8d46-9f1a2b3c4d5e",
      "tenant_id": "b7a1d0c4-2f6e-4f0a-9d58-3f2b7c1e9a10",
      "name": "Maria Souza",
      "email": "cliente@exemplo.com",
      "phone": "+5511999999999",
      "tags": ["vip"],
      "created_at": "2026-11-24T13:38:47Z",
      "updated_at": "2026-11-24T13:38:47Z"
    }
  ],
  "meta": {
    "page": 1,
    "page_size": 50,
    "total_pages": 1,
    "total_items": 1,
    "has_next": false,
    "has_previous": false
  }
}
```

Avance incrementando `page` enquanto `meta.has_next` for `true`. A busca por telefone compara o texto
como foi gravado: procure pelos dígitos finais (`999999999`) se não tiver certeza do formato usado no
cadastro.

A listagem não traz `identities`; para vê-las, obtenha o contato pelo id.

## Obter

```bash
curl "$LINKTOR_URL/api/v1/contacts/$CONTACT_ID" \
  -H "X-API-Key: $LINKTOR_API_KEY"
```

Resposta `200` com o [objeto contato](#o-objeto-contato) completo, incluindo `identities`. Id
inexistente ou de outro tenant devolve `404` `CONTACT_NOT_FOUND`.

## Atualizar

`PUT /contacts/:id` recebe o mesmo corpo da criação.

```bash
curl -X PUT "$LINKTOR_URL/api/v1/contacts/$CONTACT_ID" \
  -H "X-API-Key: $LINKTOR_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Maria Souza Lima",
    "email": "cliente@exemplo.com",
    "phone": "+5511999999999",
    "custom_fields": { "codigo_cliente": "C-1042", "plano": "ouro" },
    "tags": ["vip", "sp"]
  }'
```

:::warning O `PUT` substitui os dados cadastrais
`name`, `email`, `phone` e `avatar_url` são **sempre regravados**: um campo que você não enviar fica
**vazio**. Envie o contato inteiro — leia com `GET`, altere e devolva.

`custom_fields` e `tags` só mudam quando enviados, e nesse caso substituem o valor inteiro (não há
mesclagem de chaves nem acréscimo de etiqueta).
:::

Resposta `200` com o contato atualizado (sem `identities`). As identidades não são alteradas por
esta rota.

:::note
A verificação de e-mail e telefone duplicados acontece na **criação**. A atualização não a repete.
:::

## Remover

```bash
curl -X DELETE "$LINKTOR_URL/api/v1/contacts/$CONTACT_ID" \
  -H "X-API-Key: $LINKTOR_API_KEY"
```

Resposta `204` sem corpo; `404` `CONTACT_NOT_FOUND` se o id não existe no seu tenant.

## Identidades

A identidade diz **como alcançar o contato em um tipo de canal**. É por ela que uma mensagem recebida
é ligada ao contato certo, e é ela que o Linktor usa como destinatário ao responder.

| Campo | Descrição |
|---|---|
| `channel_type` | Tipo do canal: `whatsapp`, `whatsapp_official`, `telegram`, `webchat`, `sms`, `rcs`, `instagram`, `facebook`, `email`, `voice`, `teams`, `slack`, `mattermost`, `direto` |
| `identifier` | O endereço do contato naquele canal: telefone, e-mail, id do usuário no provedor |
| `metadata` | Mapa `string → string` opcional |

Um contato pode ter várias identidades — de tipos de canal diferentes ou do mesmo tipo (cada sessão
nova do WebChat, por exemplo, gera uma). Quando há mais de uma do mesmo tipo, vale a **mais recente**.

### Adicionar

```bash
curl -X POST "$LINKTOR_URL/api/v1/contacts/$CONTACT_ID/identities" \
  -H "X-API-Key: $LINKTOR_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "channel_type": "telegram",
    "identifier": "123456789",
    "metadata": { "username": "mariasouza" }
  }'
```

`channel_type` e `identifier` são obrigatórios. Resposta `200` com o contato e a lista de
`identities` atualizada:

```json
{
  "success": true,
  "data": {
    "id": "0b9d5a3e-1c7f-4e2a-8d46-9f1a2b3c4d5e",
    "tenant_id": "b7a1d0c4-2f6e-4f0a-9d58-3f2b7c1e9a10",
    "name": "Maria Souza",
    "phone": "+5511999999999",
    "identities": [
      {
        "id": "9a8b7c6d-5e4f-4a3b-2c1d-0e9f8a7b6c5d",
        "contact_id": "0b9d5a3e-1c7f-4e2a-8d46-9f1a2b3c4d5e",
        "channel_type": "whatsapp_official",
        "identifier": "5511999999999",
        "created_at": "2026-11-24T13:38:47Z"
      },
      {
        "id": "1f2e3d4c-5b6a-4978-8695-a4b3c2d1e0f9",
        "contact_id": "0b9d5a3e-1c7f-4e2a-8d46-9f1a2b3c4d5e",
        "channel_type": "telegram",
        "identifier": "123456789",
        "metadata": { "username": "mariasouza" },
        "created_at": "2026-11-24T15:01:20Z"
      }
    ],
    "created_at": "2026-11-24T13:38:47Z",
    "updated_at": "2026-11-24T13:38:47Z"
  }
}
```

:::note
`channel_type` não é validado contra a lista de tipos: um valor digitado errado é gravado e
simplesmente nunca casa com um canal. Repetir o mesmo par tipo + identificador no **mesmo** contato
apenas atualiza o `metadata`. A rota **não impede** que o mesmo par seja adicionado a dois contatos
diferentes — confira antes com `GET /contacts?search=`, ou a mensagem recebida pode ser ligada ao
contato errado.

O formato do `identifier` é o que o provedor entrega — no WhatsApp, por exemplo, os dígitos sem `+`.
Na dúvida, veja como ele aparece em `identities` de um contato criado por uma mensagem recebida
naquele canal.
:::

### Remover

```bash
curl -X DELETE "$LINKTOR_URL/api/v1/contacts/$CONTACT_ID/identities/$IDENTITY_ID" \
  -H "X-API-Key: $LINKTOR_API_KEY"
```

Diferente do `DELETE` do contato, esta rota devolve `200` **com o contato** e as identidades que
restaram.

## Agenda do aparelho e foto (WhatsApp não oficial)

Duas rotas de `/channels` trazem dados de contato direto da sessão do WhatsApp pareada. Elas estão em
[Canais](./canais.md): `GET /channels/:id/contacts` (agenda do aparelho) e
`GET /channels/:id/contacts/:contact/avatar` (foto, exige também `contacts:read`).

## Erros desta página

| Status | `error.code` | Quando |
|---|---|---|
| `400` | `VALIDATION_ERROR` | Corpo inválido, `name` ausente na criação, `channel_type`/`identifier` ausentes |
| `401` | `UNAUTHORIZED` | Credencial ausente ou inválida |
| `403` | `FORBIDDEN` | API key sem `contacts:read` / `contacts:write` |
| `404` | `CONTACT_NOT_FOUND` | O contato não existe **ou** é de outro tenant |
| `409` | `CONFLICT` | E-mail ou telefone já usado por outro contato (na criação) |

O formato completo do erro está em [Erros e paginação](./erros-e-paginacao.md).
