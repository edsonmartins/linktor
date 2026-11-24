---
sidebar_position: 9
title: Modelos e campanhas
description: Referência das rotas /templates (modelos de mensagem do WhatsApp oficial) e /campaigns (disparo em massa de um modelo para uma lista de telefones).
---

# Modelos e campanhas

Modelos de mensagem (templates) são as mensagens pré-aprovadas pela Meta que o WhatsApp oficial exige
para iniciar uma conversa fora da janela de atendimento. Campanhas disparam um modelo para uma lista de
telefones. As duas coisas ficam em `/api/v1/templates` e `/api/v1/campaigns`.

Para a operação pelo painel, veja [Modelos de mensagem](../painel/modelos-de-mensagem.md) e
[Campanhas](../painel/campanhas.md). Para enviar um modelo a **um** destinatário, veja
[Enviar mensagens](enviar-mensagens.md).

## Autenticação e permissões

| Rotas | API key (`X-API-Key`) | Usuário (JWT) |
|---|---|---|
| `GET /templates…` | scope `templates:read` | qualquer usuário autenticado |
| `POST`, `PATCH`, `DELETE /templates…` | scope `templates:write` | qualquer usuário autenticado |
| `GET /campaigns…` | qualquer chave válida (não há scope nessas rotas) | qualquer usuário autenticado |
| `POST /campaigns` | **403** por padrão (veja o aviso) | papel com permissão `campaigns` + `create` |
| `POST /campaigns/:id/start`, `/retry`, `/cancel` | **403** por padrão (veja o aviso) | papel com permissão `campaigns` + `update` |

:::warning Campanhas não são criadas com API key
As rotas de escrita de `/campaigns` não usam scopes: elas checam a **permissão do papel** de quem chama.
Uma API key autentica com o papel interno `api`, que não tem permissões de papel — o resultado é
`403 FORBIDDEN` (`insufficient permissions`) ao criar, iniciar, reprocessar ou cancelar uma campanha.
Use o token de um usuário `owner`, `admin` ou `supervisor` (`Authorization: Bearer <token>`, veja
[Autenticação](autenticacao.md)). A leitura (`GET`) funciona com API key.
:::

Os exemplos de modelos usam `X-API-Key`; os de campanha usam `Authorization: Bearer $LINKTOR_TOKEN`.

## Modelos de mensagem

### Rotas

| Método | Caminho | O que faz |
|---|---|---|
| `GET` | `/templates` | Lista os modelos do tenant (ou de um canal) |
| `POST` | `/templates` | Cria um modelo e o submete à Meta |
| `GET` | `/templates/:id` | Obtém um modelo |
| `PATCH` | `/templates/:id` | Edita categoria, componentes ou TTL (volta para `PENDING`) |
| `DELETE` | `/templates/:id` | Remove um modelo — `204` |
| `DELETE` | `/templates` | Remove vários de uma vez (mesmo canal) — `204` |
| `POST` | `/templates/:id/refresh` | Busca de novo o modelo na Meta e atualiza a cópia local |
| `POST` | `/templates/sync/:channelId` | Sincroniza todos os modelos do canal a partir da Meta (veja o aviso) |
| `GET` | `/templates/library?channel_id=…` | Lista a biblioteca de modelos prontos da Meta |
| `POST` | `/templates/library` | Cria um modelo a partir da biblioteca |
| `GET` | `/templates/namespace/:channel_id` | Devolve o `message_template_namespace` da conta (WABA) |

### O objeto modelo

```json
{
  "id": "0b6c1c0e-6f0a-4a39-9d0e-2f3a6a1f8a11",
  "tenant_id": "4f3d…",
  "channel_id": "CHANNEL_ID",
  "external_id": "1234567890123456",
  "name": "confirmacao_pedido",
  "language": "pt_BR",
  "category": "UTILITY",
  "parameter_format": "POSITIONAL",
  "status": "PENDING",
  "quality_score": "UNKNOWN",
  "components": [
    {
      "type": "BODY",
      "text": "Olá {{1}}, seu pedido {{2}} foi confirmado.",
      "example": { "body_text": [["Maria", "#1234"]] }
    }
  ],
  "created_at": "2026-11-24T14:03:11Z",
  "updated_at": "2026-11-24T14:03:11Z"
}
```

| Campo | Valores |
|---|---|
| `category` | `AUTHENTICATION`, `MARKETING`, `UTILITY` |
| `status` | `PENDING`, `APPROVED`, `REJECTED`, `PAUSED`, `DISABLED`, `IN_APPEAL`, `PENDING_DELETION`, `DELETED`, `REINSTATED`, `LIMIT_EXCEEDED`, `ARCHIVED` |
| `quality_score` | `GREEN`, `YELLOW`, `RED`, `UNKNOWN` |
| `parameter_format` | `POSITIONAL` (`{{1}}`, `{{2}}`) ou `NAMED` |
| `external_id` | Id do modelo na Meta. Vazio em modelo local (canal sem credenciais) |
| `rejection_reason` | Presente quando a Meta rejeita |
| `last_synced_at` | Última sincronização com a Meta |

Cada item de `components` tem `type` (`HEADER`, `BODY`, `FOOTER`, `BUTTONS`…, como na Graph API da
Meta) e, conforme o tipo: `format` (`TEXT`, `IMAGE`, `VIDEO`, `DOCUMENT`, `LOCATION`), `text`, `example`
(`header_text`, `body_text`, `header_handle`), `buttons` (`type` = `QUICK_REPLY`, `URL`, `PHONE_NUMBER`,
`COPY_CODE`, `OTP`, `FLOW`; `text`; `url`; `phone_number`…), `cards` (carrossel) e `limited_time_offer`.

### Listar modelos

`GET /templates` — scope `templates:read`.

| Parâmetro | Descrição |
|---|---|
| `channel_id` | Só os modelos deste canal |
| `status`, `category`, `sub_category`, `language`, `quality_score` | Filtro exato |
| `name`, `content` | Busca por nome ou por conteúdo |
| `since`, `until` | Intervalo, em segundos Unix. Valor malformado é ignorado |
| `page` | Padrão `1` |
| `page_size` | Padrão `50`, máximo `250` |

<Tabs groupId="linguagem">
<TabItem value="curl" label="cURL">

```bash
curl "$LINKTOR_URL/api/v1/templates?channel_id=CHANNEL_ID&status=APPROVED&page_size=100" \
  -H "X-API-Key: $LINKTOR_API_KEY"
```

</TabItem>
<TabItem value="typescript" label="TypeScript">

```typescript
const base = `${process.env.LINKTOR_URL}/api/v1`;

const qs = new URLSearchParams({ channel_id: 'CHANNEL_ID', status: 'APPROVED', page_size: '100' });
const res = await fetch(`${base}/templates?${qs}`, {
  headers: { 'X-API-Key': process.env.LINKTOR_API_KEY! },
});
if (!res.ok) throw new Error(`HTTP ${res.status}: ${await res.text()}`);

const { data: templates, meta } = await res.json();
console.log(meta.total_items, templates.map((t: any) => t.name));
```

</TabItem>
<TabItem value="python" label="Python">

```python
import os
import requests

base = f"{os.environ['LINKTOR_URL']}/api/v1"

res = requests.get(
    f"{base}/templates",
    headers={"X-API-Key": os.environ["LINKTOR_API_KEY"]},
    params={"channel_id": "CHANNEL_ID", "status": "APPROVED", "page_size": 100},
    timeout=30,
)
res.raise_for_status()
body = res.json()
print(body["meta"]["total_items"], [t["name"] for t in body["data"]])
```

</TabItem>
<TabItem value="go" label="Go">

```go
package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
)

func main() {
	q := url.Values{"channel_id": {"CHANNEL_ID"}, "status": {"APPROVED"}, "page_size": {"100"}}
	req, _ := http.NewRequest("GET", os.Getenv("LINKTOR_URL")+"/api/v1/templates?"+q.Encode(), nil)
	req.Header.Set("X-API-Key", os.Getenv("LINKTOR_API_KEY"))

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		panic(err)
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		panic(res.Status)
	}

	var body struct {
		Data []struct {
			ID     string `json:"id"`
			Name   string `json:"name"`
			Status string `json:"status"`
		} `json:"data"`
		Meta struct {
			TotalItems int64 `json:"total_items"`
		} `json:"meta"`
	}
	if err := json.NewDecoder(res.Body).Decode(&body); err != nil {
		panic(err)
	}
	fmt.Println(body.Meta.TotalItems, body.Data)
}
```

</TabItem>
<TabItem value="java" label="Java">

```java
import java.net.URI;
import java.net.http.HttpClient;
import java.net.http.HttpRequest;
import java.net.http.HttpResponse;

public class ListarModelos {
    public static void main(String[] args) throws Exception {
        String base = System.getenv("LINKTOR_URL") + "/api/v1";

        HttpRequest req = HttpRequest.newBuilder()
            .uri(URI.create(base + "/templates?channel_id=CHANNEL_ID&status=APPROVED&page_size=100"))
            .header("X-API-Key", System.getenv("LINKTOR_API_KEY"))
            .GET()
            .build();

        HttpResponse<String> res = HttpClient.newHttpClient()
            .send(req, HttpResponse.BodyHandlers.ofString());
        if (res.statusCode() != 200) throw new RuntimeException("HTTP " + res.statusCode() + ": " + res.body());

        System.out.println(res.body()); // {"success":true,"data":[...],"meta":{...}}
    }
}
```

</TabItem>
<TabItem value="csharp" label="C#">

```csharp
using System.Text.Json;

var baseUrl = Environment.GetEnvironmentVariable("LINKTOR_URL") + "/api/v1";
using var http = new HttpClient();
http.DefaultRequestHeaders.Add("X-API-Key", Environment.GetEnvironmentVariable("LINKTOR_API_KEY"));

var res = await http.GetAsync($"{baseUrl}/templates?channel_id=CHANNEL_ID&status=APPROVED&page_size=100");
res.EnsureSuccessStatusCode();

using var doc = JsonDocument.Parse(await res.Content.ReadAsStringAsync());
foreach (var t in doc.RootElement.GetProperty("data").EnumerateArray())
    Console.WriteLine($"{t.GetProperty("name").GetString()} — {t.GetProperty("status").GetString()}");
```

</TabItem>
<TabItem value="php" label="PHP">

```php
<?php
require 'vendor/autoload.php';

$http = new GuzzleHttp\Client([
    'base_uri' => getenv('LINKTOR_URL') . '/api/v1/',
    'headers'  => ['X-API-Key' => getenv('LINKTOR_API_KEY')],
]);

$res = $http->get('templates', [
    'query' => ['channel_id' => 'CHANNEL_ID', 'status' => 'APPROVED', 'page_size' => 100],
]);
$body = json_decode((string) $res->getBody(), true);

foreach ($body['data'] as $t) {
    echo $t['name'], ' — ', $t['status'], PHP_EOL;
}
```

</TabItem>
<TabItem value="rust" label="Rust">

```rust
// Cargo.toml: reqwest = { version = "0.12", features = ["json"] }, serde_json = "1", tokio = { version = "1", features = ["full"] }
use serde_json::Value;

#[tokio::main]
async fn main() -> Result<(), Box<dyn std::error::Error>> {
    let base = format!("{}/api/v1", std::env::var("LINKTOR_URL")?);

    let body: Value = reqwest::Client::new()
        .get(format!("{base}/templates"))
        .header("X-API-Key", std::env::var("LINKTOR_API_KEY")?)
        .query(&[("channel_id", "CHANNEL_ID"), ("status", "APPROVED"), ("page_size", "100")])
        .send()
        .await?
        .error_for_status()?
        .json()
        .await?;

    for t in body["data"].as_array().unwrap_or(&vec![]) {
        println!("{} — {}", t["name"], t["status"]);
    }
    Ok(())
}
```

</TabItem>
</Tabs>

Resposta `200`:

```json
{
  "success": true,
  "data": [ { "id": "0b6c1c0e-…", "name": "confirmacao_pedido", "language": "pt_BR", "status": "APPROVED", "…": "…" } ],
  "meta": { "page": 1, "page_size": 100, "total_pages": 0, "total_items": 12, "has_next": false, "has_previous": false }
}
```

:::note Paginação desta rota
Em `GET /templates`, só `page`, `page_size` e `total_items` são preenchidos em `meta`. `total_pages`,
`has_next` e `has_previous` vêm sempre zerados — calcule a próxima página a partir de `total_items`.
:::

### Criar um modelo

`POST /templates` — scope `templates:write`. Resposta `201` com o modelo em `data`, `status: "PENDING"`.

| Campo | Obrigatório | Descrição |
|---|---|---|
| `channel_id` | sim | Canal WhatsApp oficial dono do modelo |
| `name` | sim | Nome do modelo (minúsculas e `_`, regra da Meta) |
| `language` | sim | Código de idioma da Meta, ex. `pt_BR` |
| `category` | sim | `AUTHENTICATION`, `MARKETING` ou `UTILITY` |
| `components` | sim | Lista de componentes (veja acima) |
| `sub_category` | não | Subcategoria da Meta |
| `parameter_format` | não | `POSITIONAL` ou `NAMED` |
| `message_send_ttl_seconds` | não | Validade do envio, em segundos |
| `allow_category_change` | não | Permite que a Meta reclassifique a categoria |

O que acontece na criação:

- Os componentes são validados localmente antes de chamar a Meta. O erro mais comum é declarar variáveis
  (`{{1}}`) no `BODY` sem `example.body_text`.
- Nome + idioma já existentes no mesmo canal são recusados.
- Se o canal tem credenciais, o modelo é criado na Meta; se a Meta rejeitar, **nada é gravado** no
  Linktor. Sem credenciais, o modelo fica só local (rascunho, `external_id` vazio).
- Há um limite de 80 criações/edições por hora por conta WhatsApp Business (WABA).

<Tabs groupId="linguagem">
<TabItem value="curl" label="cURL">

```bash
curl -X POST "$LINKTOR_URL/api/v1/templates" \
  -H "X-API-Key: $LINKTOR_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "channel_id": "CHANNEL_ID",
    "name": "confirmacao_pedido",
    "language": "pt_BR",
    "category": "UTILITY",
    "components": [
      {
        "type": "BODY",
        "text": "Olá {{1}}, seu pedido {{2}} foi confirmado.",
        "example": { "body_text": [["Maria", "#1234"]] }
      },
      { "type": "FOOTER", "text": "Equipe Sua Empresa" }
    ]
  }'
```

</TabItem>
<TabItem value="typescript" label="TypeScript">

```typescript
const base = `${process.env.LINKTOR_URL}/api/v1`;

const res = await fetch(`${base}/templates`, {
  method: 'POST',
  headers: {
    'X-API-Key': process.env.LINKTOR_API_KEY!,
    'Content-Type': 'application/json',
  },
  body: JSON.stringify({
    channel_id: 'CHANNEL_ID',
    name: 'confirmacao_pedido',
    language: 'pt_BR',
    category: 'UTILITY',
    components: [
      {
        type: 'BODY',
        text: 'Olá {{1}}, seu pedido {{2}} foi confirmado.',
        example: { body_text: [['Maria', '#1234']] },
      },
      { type: 'FOOTER', text: 'Equipe Sua Empresa' },
    ],
  }),
});
if (res.status !== 201) throw new Error(`HTTP ${res.status}: ${await res.text()}`);

const { data: template } = await res.json();
console.log(template.id, template.status); // PENDING
```

</TabItem>
<TabItem value="python" label="Python">

```python
import os
import requests

base = f"{os.environ['LINKTOR_URL']}/api/v1"

res = requests.post(
    f"{base}/templates",
    headers={"X-API-Key": os.environ["LINKTOR_API_KEY"]},
    json={
        "channel_id": "CHANNEL_ID",
        "name": "confirmacao_pedido",
        "language": "pt_BR",
        "category": "UTILITY",
        "components": [
            {
                "type": "BODY",
                "text": "Olá {{1}}, seu pedido {{2}} foi confirmado.",
                "example": {"body_text": [["Maria", "#1234"]]},
            },
            {"type": "FOOTER", "text": "Equipe Sua Empresa"},
        ],
    },
    timeout=30,
)
res.raise_for_status()
template = res.json()["data"]
print(template["id"], template["status"])  # PENDING
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

func main() {
	payload, _ := json.Marshal(map[string]any{
		"channel_id": "CHANNEL_ID",
		"name":       "confirmacao_pedido",
		"language":   "pt_BR",
		"category":   "UTILITY",
		"components": []map[string]any{
			{
				"type":    "BODY",
				"text":    "Olá {{1}}, seu pedido {{2}} foi confirmado.",
				"example": map[string]any{"body_text": [][]string{{"Maria", "#1234"}}},
			},
			{"type": "FOOTER", "text": "Equipe Sua Empresa"},
		},
	})

	req, _ := http.NewRequest("POST", os.Getenv("LINKTOR_URL")+"/api/v1/templates", bytes.NewReader(payload))
	req.Header.Set("X-API-Key", os.Getenv("LINKTOR_API_KEY"))
	req.Header.Set("Content-Type", "application/json")

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		panic(err)
	}
	defer res.Body.Close()
	raw, _ := io.ReadAll(res.Body)
	if res.StatusCode != http.StatusCreated {
		panic(fmt.Sprintf("HTTP %d: %s", res.StatusCode, raw))
	}

	var body struct {
		Data struct {
			ID     string `json:"id"`
			Status string `json:"status"`
		} `json:"data"`
	}
	_ = json.Unmarshal(raw, &body)
	fmt.Println(body.Data.ID, body.Data.Status) // PENDING
}
```

</TabItem>
<TabItem value="java" label="Java">

```java
import java.net.URI;
import java.net.http.HttpClient;
import java.net.http.HttpRequest;
import java.net.http.HttpResponse;

public class CriarModelo {
    public static void main(String[] args) throws Exception {
        String base = System.getenv("LINKTOR_URL") + "/api/v1";

        String json = """
            {
              "channel_id": "CHANNEL_ID",
              "name": "confirmacao_pedido",
              "language": "pt_BR",
              "category": "UTILITY",
              "components": [
                {
                  "type": "BODY",
                  "text": "Olá {{1}}, seu pedido {{2}} foi confirmado.",
                  "example": { "body_text": [["Maria", "#1234"]] }
                },
                { "type": "FOOTER", "text": "Equipe Sua Empresa" }
              ]
            }
            """;

        HttpRequest req = HttpRequest.newBuilder()
            .uri(URI.create(base + "/templates"))
            .header("X-API-Key", System.getenv("LINKTOR_API_KEY"))
            .header("Content-Type", "application/json")
            .POST(HttpRequest.BodyPublishers.ofString(json))
            .build();

        HttpResponse<String> res = HttpClient.newHttpClient()
            .send(req, HttpResponse.BodyHandlers.ofString());
        if (res.statusCode() != 201) throw new RuntimeException("HTTP " + res.statusCode() + ": " + res.body());

        System.out.println(res.body());
    }
}
```

</TabItem>
<TabItem value="csharp" label="C#">

```csharp
using System.Net.Http.Json;
using System.Text.Json;

var baseUrl = Environment.GetEnvironmentVariable("LINKTOR_URL") + "/api/v1";
using var http = new HttpClient();
http.DefaultRequestHeaders.Add("X-API-Key", Environment.GetEnvironmentVariable("LINKTOR_API_KEY"));

var res = await http.PostAsJsonAsync($"{baseUrl}/templates", new
{
    channel_id = "CHANNEL_ID",
    name = "confirmacao_pedido",
    language = "pt_BR",
    category = "UTILITY",
    components = new object[]
    {
        new
        {
            type = "BODY",
            text = "Olá {{1}}, seu pedido {{2}} foi confirmado.",
            example = new { body_text = new[] { new[] { "Maria", "#1234" } } },
        },
        new { type = "FOOTER", text = "Equipe Sua Empresa" },
    },
});
res.EnsureSuccessStatusCode();

using var doc = JsonDocument.Parse(await res.Content.ReadAsStringAsync());
var template = doc.RootElement.GetProperty("data");
Console.WriteLine($"{template.GetProperty("id").GetString()} {template.GetProperty("status").GetString()}");
```

</TabItem>
<TabItem value="php" label="PHP">

```php
<?php
require 'vendor/autoload.php';

$http = new GuzzleHttp\Client([
    'base_uri' => getenv('LINKTOR_URL') . '/api/v1/',
    'headers'  => ['X-API-Key' => getenv('LINKTOR_API_KEY')],
]);

$res = $http->post('templates', [
    'json' => [
        'channel_id' => 'CHANNEL_ID',
        'name'       => 'confirmacao_pedido',
        'language'   => 'pt_BR',
        'category'   => 'UTILITY',
        'components' => [
            [
                'type'    => 'BODY',
                'text'    => 'Olá {{1}}, seu pedido {{2}} foi confirmado.',
                'example' => ['body_text' => [['Maria', '#1234']]],
            ],
            ['type' => 'FOOTER', 'text' => 'Equipe Sua Empresa'],
        ],
    ],
]);

$template = json_decode((string) $res->getBody(), true)['data'];
echo $template['id'], ' ', $template['status'], PHP_EOL; // PENDING
```

</TabItem>
<TabItem value="rust" label="Rust">

```rust
use serde_json::{json, Value};

#[tokio::main]
async fn main() -> Result<(), Box<dyn std::error::Error>> {
    let base = format!("{}/api/v1", std::env::var("LINKTOR_URL")?);

    let body: Value = reqwest::Client::new()
        .post(format!("{base}/templates"))
        .header("X-API-Key", std::env::var("LINKTOR_API_KEY")?)
        .json(&json!({
            "channel_id": "CHANNEL_ID",
            "name": "confirmacao_pedido",
            "language": "pt_BR",
            "category": "UTILITY",
            "components": [
                {
                    "type": "BODY",
                    "text": "Olá {{1}}, seu pedido {{2}} foi confirmado.",
                    "example": { "body_text": [["Maria", "#1234"]] }
                },
                { "type": "FOOTER", "text": "Equipe Sua Empresa" }
            ]
        }))
        .send()
        .await?
        .error_for_status()?
        .json()
        .await?;

    println!("{} {}", body["data"]["id"], body["data"]["status"]); // PENDING
    Ok(())
}
```

</TabItem>
</Tabs>

:::warning Erros de criação chegam como 500
Corpo malformado ou campo obrigatório ausente devolve `400 VALIDATION_ERROR`. Já a validação dos
componentes, o nome duplicado e a **rejeição pela Meta** hoje são devolvidos como
`500 INTERNAL_ERROR` com a mensagem genérica `An unexpected error occurred` — o motivo real só aparece no
log do servidor. Ao receber 500 nesta rota, revise os componentes antes de repetir a chamada; repetir
sem alterar não resolve.
:::

### Acompanhar a aprovação

A aprovação é assíncrona. O `status` muda quando a Meta notifica o Linktor pelo webhook do canal; para
forçar a leitura, chame `POST /templates/:id/refresh` e leia `status` e `rejection_reason` na resposta.

```bash
curl -X POST "$LINKTOR_URL/api/v1/templates/TEMPLATE_ID/refresh" \
  -H "X-API-Key: $LINKTOR_API_KEY"
```

:::warning Sincronização em lote
`POST /templates/sync/:channelId` está registrada, mas no código atual o handler lê o parâmetro com outro
nome e responde sempre `400` (`Channel ID is required`). Até a correção, sincronize pelo painel ou use
`POST /templates/:id/refresh` modelo a modelo.
:::

### Editar

`PATCH /templates/:id` — aceita `category`, `components` e `message_send_ttl_seconds` (os campos que a
Meta permite editar). O modelo volta para `PENDING`.

```bash
curl -X PATCH "$LINKTOR_URL/api/v1/templates/TEMPLATE_ID" \
  -H "X-API-Key: $LINKTOR_API_KEY" -H "Content-Type: application/json" \
  -d '{
    "components": [
      { "type": "BODY", "text": "Olá {{1}}, pedido {{2}} confirmado.", "example": { "body_text": [["Maria", "#1234"]] } }
    ]
  }'
```

### Remover

```bash
# Um modelo — 204
curl -X DELETE "$LINKTOR_URL/api/v1/templates/TEMPLATE_ID" -H "X-API-Key: $LINKTOR_API_KEY"

# Vários de uma vez (todos do mesmo canal) — 204
curl -X DELETE "$LINKTOR_URL/api/v1/templates" \
  -H "X-API-Key: $LINKTOR_API_KEY" -H "Content-Type: application/json" \
  -d '{ "ids": ["TEMPLATE_ID_1", "TEMPLATE_ID_2"] }'
```

### Biblioteca de modelos da Meta

`GET /templates/library` lista modelos prontos da Meta. `channel_id` é obrigatório (define o token
usado); `search`, `topic`, `usecase`, `industry` e `language` são filtros opcionais.

```bash
curl "$LINKTOR_URL/api/v1/templates/library?channel_id=CHANNEL_ID&language=pt_BR&search=pedido" \
  -H "X-API-Key: $LINKTOR_API_KEY"
```

Cada item traz `name`, `category`, `language`, `industry`, `topic`, `usecase`, `body_text`,
`header_text` e `buttons`. Para criar um modelo a partir de um deles:

```bash
curl -X POST "$LINKTOR_URL/api/v1/templates/library" \
  -H "X-API-Key: $LINKTOR_API_KEY" -H "Content-Type: application/json" \
  -d '{
    "channel_id": "CHANNEL_ID",
    "name": "atualizacao_entrega",
    "language": "pt_BR",
    "category": "UTILITY",
    "library_template_name": "NOME_NA_BIBLIOTECA"
  }'
```

Campos opcionais: `library_template_body_inputs` (objeto) e `library_template_button_inputs` (lista de
objetos), repassados à Meta. Resposta `201` com o modelo criado.

### Namespace

```bash
curl "$LINKTOR_URL/api/v1/templates/namespace/CHANNEL_ID?refresh=true" \
  -H "X-API-Key: $LINKTOR_API_KEY"
# { "success": true, "data": { "message_template_namespace": "abcd1234_…" } }
```

Sem `refresh=true` o valor vem do cache gravado no canal.

## Campanhas

Uma campanha é: um canal, um modelo (pelo **nome** e idioma) e uma lista de destinatários, cada um com
seus parâmetros. Ela nasce em `draft` e só envia depois de `start`.

### Rotas

| Método | Caminho | O que faz |
|---|---|---|
| `GET` | `/campaigns` | Lista (`page`, `page_size` — padrão 20) |
| `POST` | `/campaigns` | Cria em `draft` — `201` |
| `GET` | `/campaigns/:id` | Obtém, com contadores atualizados |
| `GET` | `/campaigns/:id/recipients` | Lista destinatários (`status`, `page`, `page_size` — padrão 100) |
| `POST` | `/campaigns/:id/start` | Inicia o disparo |
| `POST` | `/campaigns/:id/retry` | Reenfileira os destinatários com falha |
| `POST` | `/campaigns/:id/cancel` | Cancela — `204` |

### Criar e iniciar

Corpo de `POST /campaigns`:

| Campo | Obrigatório | Descrição |
|---|---|---|
| `name` | sim | Nome interno da campanha |
| `channel_id` | sim | Canal do tenant que fará o envio |
| `template_name` | sim | Nome do modelo aprovado |
| `template_language` | não | Padrão `pt_BR` |
| `recipients` | sim | Lista com ao menos um item |
| `recipients[].phone` | sim | Telefone do destinatário. Itens com telefone vazio são descartados |
| `recipients[].contact_id` | não | Contato já existente a associar |
| `recipients[].params` | não | Valores das variáveis do **corpo** do modelo, em ordem (`{{1}}`, `{{2}}`…) |

:::note Só variáveis de texto do corpo
`params` preenche apenas as variáveis de texto do componente `BODY`. Variáveis de cabeçalho, mídia de
cabeçalho e botões dinâmicos não são parametrizáveis por campanha.
:::

<Tabs groupId="linguagem">
<TabItem value="curl" label="cURL">

```bash
# 1) Cria (draft)
CAMPAIGN_ID=$(curl -s -X POST "$LINKTOR_URL/api/v1/campaigns" \
  -H "Authorization: Bearer $LINKTOR_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Confirmação de pedidos — novembro",
    "channel_id": "CHANNEL_ID",
    "template_name": "confirmacao_pedido",
    "template_language": "pt_BR",
    "recipients": [
      { "phone": "+5511999999999", "params": ["Maria", "#1234"] },
      { "phone": "+5511988888888", "params": ["João", "#1235"] }
    ]
  }' | jq -r '.data.id')

# 2) Inicia
curl -X POST "$LINKTOR_URL/api/v1/campaigns/$CAMPAIGN_ID/start" \
  -H "Authorization: Bearer $LINKTOR_TOKEN"
```

</TabItem>
<TabItem value="typescript" label="TypeScript">

```typescript
const base = `${process.env.LINKTOR_URL}/api/v1`;
const headers = {
  Authorization: `Bearer ${process.env.LINKTOR_TOKEN}`,
  'Content-Type': 'application/json',
};

async function call(method: string, path: string, body?: unknown) {
  const res = await fetch(`${base}${path}`, {
    method,
    headers,
    body: body ? JSON.stringify(body) : undefined,
  });
  if (!res.ok) throw new Error(`HTTP ${res.status}: ${await res.text()}`);
  return res.status === 204 ? null : (await res.json()).data;
}

const campaign = await call('POST', '/campaigns', {
  name: 'Confirmação de pedidos — novembro',
  channel_id: 'CHANNEL_ID',
  template_name: 'confirmacao_pedido',
  template_language: 'pt_BR',
  recipients: [
    { phone: '+5511999999999', params: ['Maria', '#1234'] },
    { phone: '+5511988888888', params: ['João', '#1235'] },
  ],
});

const started = await call('POST', `/campaigns/${campaign.id}/start`);
console.log(started.status); // processing
```

</TabItem>
<TabItem value="python" label="Python">

```python
import os
import requests

base = f"{os.environ['LINKTOR_URL']}/api/v1"
session = requests.Session()
session.headers["Authorization"] = f"Bearer {os.environ['LINKTOR_TOKEN']}"

res = session.post(
    f"{base}/campaigns",
    json={
        "name": "Confirmação de pedidos — novembro",
        "channel_id": "CHANNEL_ID",
        "template_name": "confirmacao_pedido",
        "template_language": "pt_BR",
        "recipients": [
            {"phone": "+5511999999999", "params": ["Maria", "#1234"]},
            {"phone": "+5511988888888", "params": ["João", "#1235"]},
        ],
    },
    timeout=30,
)
res.raise_for_status()
campaign = res.json()["data"]

res = session.post(f"{base}/campaigns/{campaign['id']}/start", timeout=30)
res.raise_for_status()
print(res.json()["data"]["status"])  # processing
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

type campaign struct {
	ID     string `json:"id"`
	Status string `json:"status"`
}

func call(method, path string, payload any) (*campaign, error) {
	var body io.Reader
	if payload != nil {
		raw, _ := json.Marshal(payload)
		body = bytes.NewReader(raw)
	}
	req, _ := http.NewRequest(method, os.Getenv("LINKTOR_URL")+"/api/v1"+path, body)
	req.Header.Set("Authorization", "Bearer "+os.Getenv("LINKTOR_TOKEN"))
	req.Header.Set("Content-Type", "application/json")

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	raw, _ := io.ReadAll(res.Body)
	if res.StatusCode >= 300 {
		return nil, fmt.Errorf("HTTP %d: %s", res.StatusCode, raw)
	}
	var env struct {
		Data campaign `json:"data"`
	}
	err = json.Unmarshal(raw, &env)
	return &env.Data, err
}

func main() {
	created, err := call("POST", "/campaigns", map[string]any{
		"name":              "Confirmação de pedidos — novembro",
		"channel_id":        "CHANNEL_ID",
		"template_name":     "confirmacao_pedido",
		"template_language": "pt_BR",
		"recipients": []map[string]any{
			{"phone": "+5511999999999", "params": []string{"Maria", "#1234"}},
			{"phone": "+5511988888888", "params": []string{"João", "#1235"}},
		},
	})
	if err != nil {
		panic(err)
	}

	started, err := call("POST", "/campaigns/"+created.ID+"/start", nil)
	if err != nil {
		panic(err)
	}
	fmt.Println(started.Status) // processing
}
```

</TabItem>
<TabItem value="java" label="Java">

```java
import java.net.URI;
import java.net.http.HttpClient;
import java.net.http.HttpRequest;
import java.net.http.HttpResponse;
import java.util.regex.Matcher;
import java.util.regex.Pattern;

public class CriarCampanha {
    public static void main(String[] args) throws Exception {
        String base = System.getenv("LINKTOR_URL") + "/api/v1";
        String auth = "Bearer " + System.getenv("LINKTOR_TOKEN");
        HttpClient http = HttpClient.newHttpClient();

        String json = """
            {
              "name": "Confirmação de pedidos — novembro",
              "channel_id": "CHANNEL_ID",
              "template_name": "confirmacao_pedido",
              "template_language": "pt_BR",
              "recipients": [
                { "phone": "+5511999999999", "params": ["Maria", "#1234"] },
                { "phone": "+5511988888888", "params": ["João", "#1235"] }
              ]
            }
            """;

        HttpResponse<String> created = http.send(
            HttpRequest.newBuilder()
                .uri(URI.create(base + "/campaigns"))
                .header("Authorization", auth)
                .header("Content-Type", "application/json")
                .POST(HttpRequest.BodyPublishers.ofString(json))
                .build(),
            HttpResponse.BodyHandlers.ofString());
        if (created.statusCode() != 201) throw new RuntimeException(created.body());

        // Use a biblioteca JSON do seu projeto (Jackson, Gson); aqui, só o id.
        Matcher m = Pattern.compile("\"id\":\"([^\"]+)\"").matcher(created.body());
        if (!m.find()) throw new RuntimeException("id ausente");
        String campaignId = m.group(1);

        HttpResponse<String> started = http.send(
            HttpRequest.newBuilder()
                .uri(URI.create(base + "/campaigns/" + campaignId + "/start"))
                .header("Authorization", auth)
                .POST(HttpRequest.BodyPublishers.noBody())
                .build(),
            HttpResponse.BodyHandlers.ofString());
        System.out.println(started.statusCode() + " " + started.body());
    }
}
```

</TabItem>
<TabItem value="csharp" label="C#">

```csharp
using System.Net.Http.Headers;
using System.Net.Http.Json;
using System.Text.Json;

var baseUrl = Environment.GetEnvironmentVariable("LINKTOR_URL") + "/api/v1";
using var http = new HttpClient();
http.DefaultRequestHeaders.Authorization =
    new AuthenticationHeaderValue("Bearer", Environment.GetEnvironmentVariable("LINKTOR_TOKEN"));

var created = await http.PostAsJsonAsync($"{baseUrl}/campaigns", new
{
    name = "Confirmação de pedidos — novembro",
    channel_id = "CHANNEL_ID",
    template_name = "confirmacao_pedido",
    template_language = "pt_BR",
    recipients = new[]
    {
        new { phone = "+5511999999999", @params = new[] { "Maria", "#1234" } },
        new { phone = "+5511988888888", @params = new[] { "João", "#1235" } },
    },
});
created.EnsureSuccessStatusCode();

using var doc = JsonDocument.Parse(await created.Content.ReadAsStringAsync());
var campaignId = doc.RootElement.GetProperty("data").GetProperty("id").GetString();

var started = await http.PostAsync($"{baseUrl}/campaigns/{campaignId}/start", null);
started.EnsureSuccessStatusCode();
Console.WriteLine(await started.Content.ReadAsStringAsync());
```

</TabItem>
<TabItem value="php" label="PHP">

```php
<?php
require 'vendor/autoload.php';

$http = new GuzzleHttp\Client([
    'base_uri' => getenv('LINKTOR_URL') . '/api/v1/',
    'headers'  => ['Authorization' => 'Bearer ' . getenv('LINKTOR_TOKEN')],
]);

$res = $http->post('campaigns', [
    'json' => [
        'name'              => 'Confirmação de pedidos — novembro',
        'channel_id'        => 'CHANNEL_ID',
        'template_name'     => 'confirmacao_pedido',
        'template_language' => 'pt_BR',
        'recipients'        => [
            ['phone' => '+5511999999999', 'params' => ['Maria', '#1234']],
            ['phone' => '+5511988888888', 'params' => ['João', '#1235']],
        ],
    ],
]);
$campaign = json_decode((string) $res->getBody(), true)['data'];

$res = $http->post("campaigns/{$campaign['id']}/start");
echo json_decode((string) $res->getBody(), true)['data']['status'], PHP_EOL; // processing
```

</TabItem>
<TabItem value="rust" label="Rust">

```rust
use serde_json::{json, Value};

#[tokio::main]
async fn main() -> Result<(), Box<dyn std::error::Error>> {
    let base = format!("{}/api/v1", std::env::var("LINKTOR_URL")?);
    let token = std::env::var("LINKTOR_TOKEN")?;
    let http = reqwest::Client::new();

    let created: Value = http
        .post(format!("{base}/campaigns"))
        .bearer_auth(&token)
        .json(&json!({
            "name": "Confirmação de pedidos — novembro",
            "channel_id": "CHANNEL_ID",
            "template_name": "confirmacao_pedido",
            "template_language": "pt_BR",
            "recipients": [
                { "phone": "+5511999999999", "params": ["Maria", "#1234"] },
                { "phone": "+5511988888888", "params": ["João", "#1235"] }
            ]
        }))
        .send()
        .await?
        .error_for_status()?
        .json()
        .await?;
    let id = created["data"]["id"].as_str().ok_or("id ausente")?;

    let started: Value = http
        .post(format!("{base}/campaigns/{id}/start"))
        .bearer_auth(&token)
        .send()
        .await?
        .error_for_status()?
        .json()
        .await?;
    println!("{}", started["data"]["status"]); // processing
    Ok(())
}
```

</TabItem>
</Tabs>

Resposta da criação (`201`):

```json
{
  "success": true,
  "data": {
    "id": "8a1f0c52-3d0b-4c58-9a51-0c2f6f1f7b21",
    "tenant_id": "4f3d…",
    "channel_id": "CHANNEL_ID",
    "name": "Confirmação de pedidos — novembro",
    "template_name": "confirmacao_pedido",
    "template_language": "pt_BR",
    "status": "draft",
    "total_recipients": 2,
    "sent_count": 0,
    "delivered_count": 0,
    "read_count": 0,
    "failed_count": 0,
    "created_by": "USER_ID",
    "created_at": "2026-11-24T14:10:00Z",
    "updated_at": "2026-11-24T14:10:00Z"
  }
}
```

`start` devolve a mesma campanha com `status: "processing"` e `started_at`. A resposta volta
imediatamente: cada destinatário pendente vira um envio na fila de saída, a mesma usada pelas mensagens
comuns (com retry e controle de falha).

### Acompanhar

```bash
# Contadores
curl "$LINKTOR_URL/api/v1/campaigns/CAMPAIGN_ID" -H "Authorization: Bearer $LINKTOR_TOKEN"

# Quem falhou e por quê
curl "$LINKTOR_URL/api/v1/campaigns/CAMPAIGN_ID/recipients?status=failed&page_size=100" \
  -H "Authorization: Bearer $LINKTOR_TOKEN"
```

Destinatário:

```json
{
  "id": "c2c9…",
  "campaign_id": "8a1f0c52-…",
  "phone": "+5511999999999",
  "params": ["Maria", "#1234"],
  "status": "failed",
  "message_id": "MESSAGE_ID",
  "error_reason": "…",
  "attempts": 1,
  "sent_at": "2026-11-24T14:10:05Z",
  "created_at": "2026-11-24T14:10:00Z",
  "updated_at": "2026-11-24T14:10:06Z"
}
```

| Status da campanha | Significado |
|---|---|
| `draft` | Criada, ainda não iniciada |
| `scheduled` | Reservado (a API de criação não aceita data de agendamento) |
| `processing` | Envio em andamento |
| `completed` | Todos os destinatários chegaram a um estado final |
| `failed` | A campanha falhou |
| `cancelled` | Cancelada |

| Status do destinatário | Significado |
|---|---|
| `pending` | Ainda não enfileirado |
| `queued` | Publicado na fila de saída, ainda não enviado |
| `sent`, `delivered`, `read` | Progresso informado pelo provedor |
| `failed` | Falhou — veja `error_reason` |

As listagens de campanha e de destinatários devolvem `meta` completo (`page`, `page_size`,
`total_pages`, `total_items`, `has_next`, `has_previous`).

### Reprocessar falhas e cancelar

```bash
# Devolve os "failed" para a fila e retoma o disparo
curl -X POST "$LINKTOR_URL/api/v1/campaigns/CAMPAIGN_ID/retry" -H "Authorization: Bearer $LINKTOR_TOKEN"
# { "success": true, "data": { "requeued": 3 } }

# Cancela — 204
curl -X POST "$LINKTOR_URL/api/v1/campaigns/CAMPAIGN_ID/cancel" -H "Authorization: Bearer $LINKTOR_TOKEN"
```

### Erros

| Status | `error.code` | Quando |
|---|---|---|
| `400` | `VALIDATION_ERROR` | Corpo inválido, `name`/`template_name` vazio, nenhum destinatário |
| `401` | `UNAUTHORIZED` | Sem credencial ou credencial inválida |
| `403` | `FORBIDDEN` | Papel sem permissão em `campaigns` (inclui API key) |
| `404` | `CHANNEL_NOT_FOUND` | Canal inexistente ou de outro tenant |
| `404` | `NOT_FOUND` | Campanha inexistente ou de outro tenant |
| `409` | `CONFLICT` | `start` em campanha já em `processing`; `cancel` em campanha `completed` ou `cancelled` |

O formato do envelope de erro está em [Erros e paginação](erros-e-paginacao.md).
