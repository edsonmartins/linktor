---
sidebar_position: 5
title: Conversas
description: Listar, criar, atualizar, atribuir, resolver, reabrir e escalar conversas pela API, e ler o histórico de mensagens com paginação por cursor.
---

# Conversas

Uma conversa é o fio de mensagens entre **um contato** e **um canal**. Ela nasce sozinha quando chega
a primeira mensagem do contato ou quando você usa o [envio direto](./enviar-mensagens.md); as rotas
desta página servem para consultar a caixa, mudar o estado do atendimento e ler o histórico.

Todas as rotas ficam em `/api/v1/conversations` e são restritas ao tenant da credencial.

## Rotas

| Método | Caminho | Scope da API key | O que faz |
|---|---|---|---|
| `GET` | `/conversations` | `conversations:read` | Lista a caixa, da atividade mais recente para a mais antiga |
| `POST` | `/conversations` | `conversations:write` | Cria (ou devolve a conversa aberta que já existe) |
| `GET` | `/conversations/:id` | `conversations:read` | Obtém uma conversa |
| `PUT` | `/conversations/:id` | `conversations:write` | Altera assunto, prioridade, status e etiquetas |
| `DELETE` | `/conversations/:id` | `conversations:write` | Apaga a conversa **e as mensagens** (irreversível) |
| `POST` | `/conversations/:id/assign` | `conversations:write` | Atribui a um usuário |
| `POST` | `/conversations/:id/resolve` | `conversations:write` | Marca como resolvida |
| `POST` | `/conversations/:id/reopen` | `conversations:write` | Reabre |
| `POST` | `/conversations/:id/escalate` | `conversations:write` | Passa do bot para atendimento humano |
| `GET` | `/conversations/:id/escalation-context` | `conversations:read` | Resumo da conversa para quem assume o atendimento |
| `GET` | `/conversations/:id/messages` | `conversations:read` | Histórico de mensagens |
| `POST` | `/conversations/:id/auto-assign` | — | Distribui a conversa pela regra de atribuição do tenant |

`conversations:write` implica `conversations:read`. Chamadas com sessão de usuário (JWT/cookie) não
usam scopes — valem os papéis. Veja [Autenticação](./autenticacao.md).

Enviar mensagem, reagir e subir anexo numa conversa (`POST /conversations/:id/messages`,
`.../reactions`, `.../attachments`) exigem `messages:send` **além** de `conversations:write` e estão
em [Enviar mensagens](./enviar-mensagens.md).

## O objeto conversa

```json
{
  "id": "6d1f0c2e-8a57-4c4b-9a55-0a0e3c1f6b11",
  "tenant_id": "b7a1d0c4-2f6e-4f0a-9d58-3f2b7c1e9a10",
  "contact_id": "0b9d5a3e-1c7f-4e2a-8d46-9f1a2b3c4d5e",
  "channel_id": "CHANNEL_ID",
  "environment": "production",
  "assigned_user_id": "f3c2b1a0-9e8d-4c7b-a6f5-e4d3c2b1a098",
  "status": "open",
  "priority": "normal",
  "subject": "Dúvida sobre o pedido 1234",
  "tags": ["pedido", "vip"],
  "metadata": {},
  "unread_count": 2,
  "last_message_at": "2026-11-24T14:02:11Z",
  "first_reply_at": "2026-11-24T13:40:05Z",
  "created_at": "2026-11-24T13:38:47Z",
  "updated_at": "2026-11-24T14:02:11Z",
  "contact": { "id": "0b9d5a3e-1c7f-4e2a-8d46-9f1a2b3c4d5e", "tenant_id": "…", "name": "Maria Souza", "phone": "+5511999999999", "created_at": "…", "updated_at": "…" },
  "channel": { "id": "CHANNEL_ID", "tenant_id": "…", "type": "whatsapp_official", "name": "Atendimento", "enabled": true, "connection_status": "connected", "environment": "production", "created_at": "…", "updated_at": "…" },
  "last_message": {
    "id": "a4f1c6d2-3b7e-4a19-8c5d-2e9f0b1a7c33",
    "content": "Pode seguir",
    "content_type": "text",
    "sender_type": "contact",
    "created_at": "2026-11-24T14:02:11Z"
  }
}
```

| Campo | Valores / observação |
|---|---|
| `status` | `open`, `pending`, `resolved`, `closed` |
| `priority` | `low`, `normal` (padrão), `high`, `urgent` |
| `environment` | `production` ou `sandbox`; copiado do canal na criação e imutável |
| `assigned_user_id` | Ausente enquanto ninguém assumiu |
| `resolved_at` | Presente só em conversa resolvida |
| `contact`, `channel` | Relações expandidas. Vêm em `GET /conversations` e `GET /conversations/:id`; **não** vêm nas respostas de `PUT`, `assign`, `resolve`, `reopen` e na criação |
| `last_message` | Só na listagem, e só quando a conversa tem mensagem |

## Listar conversas

`GET /conversations` devolve a caixa ordenada pela atividade mais recente (`last_message_at`; conversa
sem mensagem ordena por `created_at`). A paginação é **por cursor**, não por número de página.

| Parâmetro | Descrição |
|---|---|
| `status` | Filtra por status (`open`, `pending`, `resolved`, `closed`) |
| `assigned_to` | Id do usuário responsável |
| `channel_id` | Id do canal |
| `environment` | `production` ou `sandbox`. Vazio = todos. Outro valor devolve `400` |
| `search` | Trecho do nome, telefone ou e-mail do contato |
| `limit` | Itens por página. Padrão `30`, máximo `100` |
| `before` | Cursor recebido em `meta.next_cursor` da página anterior |

<Tabs groupId="linguagem">
<TabItem value="curl" label="cURL">

```bash
# Primeira página
curl -G "$LINKTOR_URL/api/v1/conversations" \
  -H "X-API-Key: $LINKTOR_API_KEY" \
  --data-urlencode "status=open" \
  --data-urlencode "limit=50"

# Página seguinte: repita os mesmos filtros e passe o cursor
curl -G "$LINKTOR_URL/api/v1/conversations" \
  -H "X-API-Key: $LINKTOR_API_KEY" \
  --data-urlencode "status=open" \
  --data-urlencode "limit=50" \
  --data-urlencode "before=$NEXT_CURSOR"
```

</TabItem>
<TabItem value="typescript" label="TypeScript">

```typescript
const base = `${process.env.LINKTOR_URL}/api/v1`;
const headers = { 'X-API-Key': process.env.LINKTOR_API_KEY! };

async function listarAbertas() {
  const todas: any[] = [];
  let cursor: string | undefined;

  do {
    const qs = new URLSearchParams({ status: 'open', limit: '50' });
    if (cursor) qs.set('before', cursor);

    const res = await fetch(`${base}/conversations?${qs}`, { headers });
    if (!res.ok) throw new Error(`HTTP ${res.status}: ${await res.text()}`);

    const body = await res.json();
    todas.push(...(body.data ?? []));
    cursor = body.meta?.next_cursor;
  } while (cursor);

  return todas;
}
```

</TabItem>
<TabItem value="python" label="Python">

```python
import os
import requests

BASE = f"{os.environ['LINKTOR_URL']}/api/v1"
HEADERS = {"X-API-Key": os.environ["LINKTOR_API_KEY"]}


def listar_abertas():
    todas, cursor = [], None
    while True:
        params = {"status": "open", "limit": 50}
        if cursor:
            params["before"] = cursor

        res = requests.get(f"{BASE}/conversations", headers=HEADERS, params=params, timeout=30)
        res.raise_for_status()
        body = res.json()

        todas.extend(body.get("data") or [])
        cursor = (body.get("meta") or {}).get("next_cursor")
        if not cursor:
            return todas
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

type conversa struct {
	ID       string `json:"id"`
	Status   string `json:"status"`
	Priority string `json:"priority"`
}

type pagina struct {
	Data []conversa `json:"data"`
	Meta struct {
		NextCursor string `json:"next_cursor"`
		HasNext    bool   `json:"has_next"`
	} `json:"meta"`
}

func listarAbertas() ([]conversa, error) {
	base := os.Getenv("LINKTOR_URL") + "/api/v1"
	var todas []conversa
	cursor := ""

	for {
		q := url.Values{"status": {"open"}, "limit": {"50"}}
		if cursor != "" {
			q.Set("before", cursor)
		}

		req, _ := http.NewRequest(http.MethodGet, base+"/conversations?"+q.Encode(), nil)
		req.Header.Set("X-API-Key", os.Getenv("LINKTOR_API_KEY"))

		res, err := http.DefaultClient.Do(req)
		if err != nil {
			return nil, err
		}
		if res.StatusCode != http.StatusOK {
			res.Body.Close()
			return nil, fmt.Errorf("HTTP %d", res.StatusCode)
		}

		var p pagina
		err = json.NewDecoder(res.Body).Decode(&p)
		res.Body.Close()
		if err != nil {
			return nil, err
		}

		todas = append(todas, p.Data...)
		if p.Meta.NextCursor == "" {
			return todas, nil
		}
		cursor = p.Meta.NextCursor
	}
}
```

</TabItem>
<TabItem value="java" label="Java">

```java
import com.fasterxml.jackson.databind.JsonNode;
import com.fasterxml.jackson.databind.ObjectMapper;
import java.net.URI;
import java.net.URLEncoder;
import java.net.http.HttpClient;
import java.net.http.HttpRequest;
import java.net.http.HttpResponse;
import java.nio.charset.StandardCharsets;
import java.util.ArrayList;
import java.util.List;

public class ListarConversas {
    public static List<JsonNode> listarAbertas() throws Exception {
        String base = System.getenv("LINKTOR_URL") + "/api/v1";
        HttpClient http = HttpClient.newHttpClient();
        ObjectMapper json = new ObjectMapper();

        List<JsonNode> todas = new ArrayList<>();
        String cursor = null;

        do {
            String url = base + "/conversations?status=open&limit=50";
            if (cursor != null) {
                url += "&before=" + URLEncoder.encode(cursor, StandardCharsets.UTF_8);
            }

            HttpRequest req = HttpRequest.newBuilder(URI.create(url))
                .header("X-API-Key", System.getenv("LINKTOR_API_KEY"))
                .GET()
                .build();

            HttpResponse<String> res = http.send(req, HttpResponse.BodyHandlers.ofString());
            if (res.statusCode() != 200) {
                throw new IllegalStateException("HTTP " + res.statusCode() + ": " + res.body());
            }

            JsonNode body = json.readTree(res.body());
            body.path("data").forEach(todas::add);
            cursor = body.path("meta").path("next_cursor").asText(null);
        } while (cursor != null && !cursor.isEmpty());

        return todas;
    }
}
```

</TabItem>
<TabItem value="csharp" label="C#">

```csharp
using System.Text.Json;

var http = new HttpClient { BaseAddress = new Uri(Environment.GetEnvironmentVariable("LINKTOR_URL") + "/api/v1/") };
http.DefaultRequestHeaders.Add("X-API-Key", Environment.GetEnvironmentVariable("LINKTOR_API_KEY"));

var todas = new List<JsonElement>();
string? cursor = null;

do
{
    var url = "conversations?status=open&limit=50";
    if (cursor is not null) url += "&before=" + Uri.EscapeDataString(cursor);

    using var res = await http.GetAsync(url);
    res.EnsureSuccessStatusCode();

    using var doc = JsonDocument.Parse(await res.Content.ReadAsStringAsync());
    var root = doc.RootElement;

    if (root.TryGetProperty("data", out var data) && data.ValueKind == JsonValueKind.Array)
        foreach (var item in data.EnumerateArray()) todas.Add(item.Clone());

    cursor = root.TryGetProperty("meta", out var meta) && meta.TryGetProperty("next_cursor", out var next)
        ? next.GetString()
        : null;
} while (!string.IsNullOrEmpty(cursor));

Console.WriteLine($"{todas.Count} conversas abertas");
```

</TabItem>
<TabItem value="php" label="PHP">

```php
<?php
require 'vendor/autoload.php';

use GuzzleHttp\Client;

$http = new Client([
    'base_uri' => getenv('LINKTOR_URL') . '/api/v1/',
    'headers'  => ['X-API-Key' => getenv('LINKTOR_API_KEY')],
    'timeout'  => 30,
]);

$todas = [];
$cursor = null;

do {
    $query = ['status' => 'open', 'limit' => 50];
    if ($cursor !== null) {
        $query['before'] = $cursor;
    }

    $res  = $http->get('conversations', ['query' => $query]);
    $body = json_decode((string) $res->getBody(), true);

    $todas  = array_merge($todas, $body['data'] ?? []);
    $cursor = $body['meta']['next_cursor'] ?? null;
} while (!empty($cursor));

printf("%d conversas abertas\n", count($todas));
```

</TabItem>
<TabItem value="rust" label="Rust">

```rust
use serde_json::Value;

async fn listar_abertas() -> Result<Vec<Value>, reqwest::Error> {
    let base = format!("{}/api/v1", std::env::var("LINKTOR_URL").unwrap());
    let key = std::env::var("LINKTOR_API_KEY").unwrap();
    let http = reqwest::Client::new();

    let mut todas = Vec::new();
    let mut cursor: Option<String> = None;

    loop {
        let mut query = vec![("status", "open".to_string()), ("limit", "50".to_string())];
        if let Some(c) = &cursor {
            query.push(("before", c.clone()));
        }

        let body: Value = http
            .get(format!("{base}/conversations"))
            .header("X-API-Key", &key)
            .query(&query)
            .send()
            .await?
            .error_for_status()?
            .json()
            .await?;

        if let Some(items) = body["data"].as_array() {
            todas.extend(items.iter().cloned());
        }

        match body["meta"]["next_cursor"].as_str() {
            Some(c) if !c.is_empty() => cursor = Some(c.to_string()),
            _ => return Ok(todas),
        }
    }
}
```

</TabItem>
</Tabs>

:::note Por que HTTP nativo aqui
Os SDKs têm um recurso `conversations`, mas os modelos dele ainda estão em camelCase e a listagem
espera um envelope de paginação diferente do que a API devolve. Até serem reconciliados com o
contrato, use o cliente HTTP da linguagem nas rotas desta página.
:::

Resposta:

```json
{
  "success": true,
  "data": [ { "id": "6d1f0c2e-…", "status": "open", "…": "…" } ],
  "meta": {
    "page": 0,
    "page_size": 50,
    "total_pages": 0,
    "total_items": 137,
    "has_next": true,
    "has_previous": false,
    "next_cursor": "MjAyNi0xMS0yNFQxNDowMjoxMVp8NmQxZjBjMmUtOGE1Ny00YzRiLTlhNTUtMGEwZTNjMWY2YjEx"
  }
}
```

Como ler o `meta` desta rota:

- `next_cursor` só aparece quando há mais conversas. **Ausente = fim da lista.**
- `page_size` é a quantidade de itens **nesta** resposta.
- `total_items` (total que casa com os filtros) só é calculado na **primeira página**; nas páginas
  pedidas com `before` ele vem `0`. Guarde o valor da primeira resposta.
- `page` e `total_pages` não se aplicam a esta rota e vêm `0`.
- O cursor é opaco: devolva-o como recebeu. Um cursor adulterado devolve `400` (`invalid cursor`), não
  uma lista vazia.
- Repita os mesmos filtros em todas as páginas; o cursor guarda só a posição.

Detalhes de paginação comuns à API estão em [Erros e paginação](./erros-e-paginacao.md).

## Obter uma conversa

```bash
curl "$LINKTOR_URL/api/v1/conversations/$CONVERSATION_ID" \
  -H "X-API-Key: $LINKTOR_API_KEY"
```

Devolve `200` com o objeto conversa (com `contact` e `channel` expandidos). Id inexistente ou de outro
tenant devolve `404` com `CONVERSATION_NOT_FOUND` — os dois casos são indistinguíveis de propósito.

## Criar uma conversa

Na maior parte das integrações você **não precisa** desta rota: o [envio
direto](./enviar-mensagens.md) resolve contato e conversa sozinho. Use-a quando quiser abrir a
conversa antes de enviar algo — por exemplo, para já definir assunto, prioridade e etiquetas.

| Campo | Obrigatório | Descrição |
|---|---|---|
| `contact_id` | sim | Id de um contato do tenant ([Contatos](./contatos.md)) |
| `channel_id` | sim | Id de um canal do tenant ([Canais](./canais.md)) |
| `subject` | não | Assunto |
| `priority` | não | `low`, `normal`, `high`, `urgent`. Padrão `normal` |
| `tags` | não | Lista de etiquetas |

```bash
curl -X POST "$LINKTOR_URL/api/v1/conversations" \
  -H "X-API-Key: $LINKTOR_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "contact_id": "0b9d5a3e-1c7f-4e2a-8d46-9f1a2b3c4d5e",
    "channel_id": "'"$CHANNEL_ID"'",
    "subject": "Dúvida sobre o pedido 1234",
    "priority": "high",
    "tags": ["pedido"]
  }'
```

Resposta `201` com a conversa (`status: "open"`).

:::warning A criação é idempotente por contato + canal
Se já existe uma conversa `open` ou `pending` para o mesmo contato no mesmo canal, a API devolve
**essa conversa** (também com `201`) e **ignora** `subject`, `priority` e `tags` do pedido. Compare o
`created_at` ou use `PUT` em seguida se precisar garantir os valores.
:::

Erros: `400` sem `contact_id`/`channel_id` ou com `priority` inválida; `404` `CONTACT_NOT_FOUND` ou
`CHANNEL_NOT_FOUND` quando o id não existe no seu tenant.

## Atualizar

`PUT /conversations/:id` é **parcial**: só os campos enviados mudam.

| Campo | Descrição |
|---|---|
| `subject` | Novo assunto |
| `priority` | `low`, `normal`, `high`, `urgent` |
| `status` | `open`, `pending`, `resolved`, `closed` |
| `tags` | Substitui a lista inteira de etiquetas |

```bash
curl -X PUT "$LINKTOR_URL/api/v1/conversations/$CONVERSATION_ID" \
  -H "X-API-Key: $LINKTOR_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{ "priority": "urgent", "tags": ["pedido", "vip"] }'
```

Resposta `200` com a conversa. Valor inválido em `priority` ou `status` devolve `400`.

:::note Prefira `resolve` e `reopen` para mudar o estado
Mudar `status` por `PUT` apenas grava o valor. São as rotas `resolve`, `reopen` e `assign` que
publicam os eventos de ciclo de vida entregues no webhook da sua aplicação
([Receber eventos](./receber-eventos.md)).
:::

## Atribuir

```bash
curl -X POST "$LINKTOR_URL/api/v1/conversations/$CONVERSATION_ID/assign" \
  -H "X-API-Key: $LINKTOR_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{ "user_id": "f3c2b1a0-9e8d-4c7b-a6f5-e4d3c2b1a098" }'
```

`user_id` é obrigatório. Resposta `200` com a conversa e `assigned_user_id` preenchido. Atribuir de
novo troca o responsável.

Para deixar o Linktor escolher o atendente pela regra de distribuição configurada no tenant:

```bash
curl -X POST "$LINKTOR_URL/api/v1/conversations/$CONVERSATION_ID/auto-assign" \
  -H "X-API-Key: $LINKTOR_API_KEY"
```

```json
{ "success": true, "data": { "conversation_id": "6d1f0c2e-…", "assigned_to": "f3c2b1a0-…" } }
```

:::note
`auto-assign` está registrada fora do grupo `/conversations`, então nenhum scope `conversations:*` é
verificado nela; a chamada é registrada no log de auditoria.
:::

## Resolver e reabrir

```bash
# Resolver
curl -X POST "$LINKTOR_URL/api/v1/conversations/$CONVERSATION_ID/resolve" \
  -H "X-API-Key: $LINKTOR_API_KEY"

# Reabrir
curl -X POST "$LINKTOR_URL/api/v1/conversations/$CONVERSATION_ID/reopen" \
  -H "X-API-Key: $LINKTOR_API_KEY"
```

Nenhuma das duas recebe corpo. Ambas devolvem `200` com a conversa.

| Chamada | Efeito | Erro `400` quando |
|---|---|---|
| `resolve` | `status` vira `resolved`, `resolved_at` é preenchido | A conversa já está `resolved` (`conversation is already resolved`) |
| `reopen` | `status` vira `open`, `resolved_at` é limpo | A conversa já está `open` ou `pending` (`conversation is already open`) |

Resolver não apaga nada: o histórico continua disponível e a próxima mensagem do contato abre uma
conversa nova.

## Escalar para atendimento humano

`POST /conversations/:id/escalate` tira a conversa do bot e a entrega a um atendente. Se houver
atendente disponível para o canal, o de menor carga é atribuído na hora; se não, a conversa vai para
a fila com `status: "pending"`.

| Campo | Descrição |
|---|---|
| `reason` | Motivo, texto livre. Fica em `metadata.escalation_reason` da conversa |
| `priority` | `low`, `normal`, `high`, `urgent`. Valor ausente ou desconhecido vira `normal` |

```bash
curl -X POST "$LINKTOR_URL/api/v1/conversations/$CONVERSATION_ID/escalate" \
  -H "X-API-Key: $LINKTOR_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{ "reason": "Cliente pediu para falar com uma pessoa", "priority": "high" }'
```

```json
{
  "success": true,
  "data": {
    "conversation_id": "6d1f0c2e-8a57-4c4b-9a55-0a0e3c1f6b11",
    "status": "queued",
    "queue_position": 3,
    "estimated_wait_seconds": 360,
    "escalated_at": "2026-11-24T14:10:00Z"
  }
}
```

| `data.status` | Significado |
|---|---|
| `assigned` | Um atendente foi atribuído; `assigned_user_id` vem na resposta |
| `queued` | Ninguém disponível; `queue_position` e `estimated_wait_seconds` (estimativa simples: 2 minutos por posição) |
| `already_assigned` | A conversa já tinha responsável; nada foi alterado |

:::warning `assign_to` é ignorado
O corpo aceita um campo `assign_to`, mas a rota não o usa: o responsável é sempre escolhido pela
regra automática. Para direcionar a um usuário específico, chame `assign` depois.
:::

### Contexto da escalação

`GET /conversations/:id/escalation-context` devolve o material que o atendente precisa para assumir
sem reler tudo: `summary` (resumo gerado por IA), `detected_intent`, `sentiment`,
`escalation_reason`, `priority`, `collected_entities`, `flow_data`, `tags`, `last_messages`,
`message_count`, `bot_attempts` e `bot_success_rate`.

```bash
curl "$LINKTOR_URL/api/v1/conversations/$CONVERSATION_ID/escalation-context" \
  -H "X-API-Key: $LINKTOR_API_KEY"
```

:::note
O conteúdo de `summary`, `detected_intent` e `sentiment` depende de haver um provedor de IA
configurado no servidor. Veja [Bots e IA](./bots-e-ia.md).
:::

## Mensagens da conversa

`GET /conversations/:id/messages` devolve o histórico **da mais nova para a mais antiga**, também por
cursor.

| Parâmetro | Descrição |
|---|---|
| `limit` | Mensagens por página. Padrão `30`, máximo `100` |
| `before` | Cursor de `meta.next_cursor`; traz as mensagens **anteriores** à última da página |

```bash
curl -G "$LINKTOR_URL/api/v1/conversations/$CONVERSATION_ID/messages" \
  -H "X-API-Key: $LINKTOR_API_KEY" \
  --data-urlencode "limit=50"
```

```json
{
  "success": true,
  "data": [
    {
      "id": "a4f1c6d2-3b7e-4a19-8c5d-2e9f0b1a7c33",
      "conversation_id": "6d1f0c2e-8a57-4c4b-9a55-0a0e3c1f6b11",
      "sender_type": "contact",
      "content_type": "image",
      "content": "Segue o comprovante",
      "metadata": {},
      "status": "delivered",
      "external_id": "wamid.HBgNNTUxMTk5OTk5OTk5ORUCABIYFDNBQjY…",
      "attachments": [
        {
          "id": "c1d2e3f4-…",
          "message_id": "a4f1c6d2-…",
          "type": "image",
          "filename": "comprovante.jpg",
          "mime_type": "image/jpeg",
          "size_bytes": 184223,
          "url": "https://linktor.suaempresa.com/api/v1/media/…?exp=…&sig=…",
          "created_at": "2026-11-24T14:02:11Z"
        }
      ],
      "created_at": "2026-11-24T14:02:11Z",
      "source": "api"
    }
  ],
  "meta": {
    "page": 0,
    "page_size": 50,
    "total_pages": 0,
    "total_items": 0,
    "has_next": true,
    "has_previous": false,
    "next_cursor": "MjAyNi0xMS0yNFQxMzo0MDowNVp8ZjBlMWQyYzMtLi4u"
  }
}
```

| Campo da mensagem | Valores |
|---|---|
| `sender_type` | `contact`, `user`, `bot`, `system` |
| `content_type` | `text`, `image`, `video`, `audio`, `document`, `location`, `contact`, `template`, `interactive`, `sticker`, `poll`, `reaction`, `call` |
| `status` | `pending`, `sent`, `delivered`, `read`, `failed` (com `error_message` quando falha) |
| `source` | `api`, `business_app` (digitada no app do WhatsApp Business), `imported` |
| `reply_to_id` | Id da mensagem citada, quando é resposta |
| `reactions` | Lista de `{ "user_id", "emoji", "timestamp" }` |
| `is_edited`, `is_deleted` | Presentes quando a mensagem foi editada ou apagada |

Particularidades:

- Esta rota **não** informa o total: `total_items` vem sempre `0`. Pare quando `next_cursor` não vier.
- Com a assinatura de mídia ativa no servidor, as URLs em `attachments[].url` vêm **assinadas e com
  validade**. Não as guarde como link permanente; veja [Mídia](./midia.md).
- Para consultar uma única mensagem, `GET /messages/:id` — descrita em
  [Enviar mensagens](./enviar-mensagens.md), junto com o acompanhamento de status.

Percorrer o histórico inteiro, da mais nova para a mais antiga:

```bash
CURSOR=""
while :; do
  RESP=$(curl -s -G "$LINKTOR_URL/api/v1/conversations/$CONVERSATION_ID/messages" \
    -H "X-API-Key: $LINKTOR_API_KEY" \
    --data-urlencode "limit=100" \
    ${CURSOR:+--data-urlencode "before=$CURSOR"})

  echo "$RESP" | jq -c '.data[] | {id, sender_type, content}'

  CURSOR=$(echo "$RESP" | jq -r '.meta.next_cursor // empty')
  [ -z "$CURSOR" ] && break
done
```

## Apagar

```bash
curl -X DELETE "$LINKTOR_URL/api/v1/conversations/$CONVERSATION_ID" \
  -H "X-API-Key: $LINKTOR_API_KEY"
```

Resposta `204` sem corpo.

:::danger Irreversível
Apagar remove a conversa e, em cascata, **todas as mensagens**. Não há desfazer. Para tirar a conversa
da fila preservando o histórico, use `resolve`.
:::

Além do scope `conversations:write`, a rota exige papel `admin`, `owner` ou `supervisor` quando
chamada por um usuário — um atendente (`agent`) recebe `403`. Chamadas por API key passam por essa
verificação. Toda exclusão entra no log de auditoria.

## Erros desta página

| Status | `error.code` | Quando |
|---|---|---|
| `400` | `VALIDATION_ERROR` | Corpo inválido, `priority`/`status`/`environment` fora dos valores aceitos, cursor ilegível, conversa já resolvida/aberta |
| `401` | `UNAUTHORIZED` | Credencial ausente ou inválida |
| `403` | `FORBIDDEN` | API key sem o scope da rota, ou papel insuficiente no `DELETE` |
| `404` | `CONVERSATION_NOT_FOUND`, `CONTACT_NOT_FOUND`, `CHANNEL_NOT_FOUND` | O recurso não existe **ou** é de outro tenant |

O formato completo do erro está em [Erros e paginação](./erros-e-paginacao.md).
