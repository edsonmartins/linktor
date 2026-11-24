---
sidebar_position: 10
title: Bots e IA
description: Como criar e ativar bots, alimentar a base de conhecimento, montar fluxos e chamar os serviços de IA pela API — /bots, /knowledge-bases, /flows e /ai.
---

# Bots e IA

Esta página resume, por tarefa, as quatro famílias de rotas de automação:

| Recurso | Base | Scope de leitura | Scope de escrita |
|---|---|---|---|
| Bots | `/api/v1/bots` | `bots:read` | `bots:write` |
| Base de conhecimento | `/api/v1/knowledge-bases` | `knowledge:read` | `knowledge:write` |
| Fluxos | `/api/v1/flows` | `flows:read` | `flows:write` |
| Serviços de IA | `/api/v1/ai` | `ai:use` (todas as rotas) | `ai:use` |

Em cada grupo, `GET` exige o scope de leitura e qualquer outro método exige o de escrita — inclusive
os `POST` que só consultam, como `POST /bots/:id/test` e `POST /knowledge-bases/:id/search`. Usuários
autenticados por JWT não usam scopes. Detalhes em [Autenticação](autenticacao.md).

Para configurar pelo painel, veja [Bots](../painel/bots.md), [Fluxos](../painel/fluxos.md) e
[Base de conhecimento](../painel/base-de-conhecimento.md).

:::warning Formato de resposta diferente nestas rotas
Parte destas rotas **não** usa o envelope padrão `{ "success", "data", "meta" }`:

- As listagens (`GET /bots`, `GET /flows`, `GET /knowledge-bases`, `GET /knowledge-bases/:id/items`)
  devolvem `{ "data": [...], "pagination": { "total", "page", "page_size", "pages" } }`, sem `success`.
- `GET /flows/:id` e `PUT /flows/:id` devolvem o fluxo direto na raiz do JSON; ativar, desativar e
  remover fluxo devolvem `{ "message": "…" }`; `POST /flows/:id/test` devolve `{ "flow_id", "results" }`.
- `POST /knowledge-bases/:id/search` e `POST /knowledge-bases/:id/items/bulk` também respondem sem
  envelope.

Os erros seguem o formato padrão em todas elas. Um cliente que valida `success === true` em toda
resposta vai rejeitar essas chamadas — trate pelo status HTTP.
:::

## Bots

### Rotas

| Método | Caminho | O que faz |
|---|---|---|
| `GET` | `/bots` | Lista (`page`, `page_size` — padrão 20, `sort_by`, `sort_dir`) |
| `POST` | `/bots` | Cria — `201` |
| `GET` | `/bots/:id` | Obtém |
| `PUT` | `/bots/:id` | Atualiza nome, modelo e mensagens |
| `DELETE` | `/bots/:id` | Remove — `204` |
| `POST` | `/bots/:id/activate` | Ativa |
| `POST` | `/bots/:id/deactivate` | Desativa |
| `POST` | `/bots/:id/channels` | Vincula um canal ao bot |
| `DELETE` | `/bots/:id/channels/:channelId` | Desvincula o canal |
| `PUT` | `/bots/:id/config` | Atualiza a configuração completa |
| `POST` | `/bots/:id/escalation-rules` | Acrescenta uma regra de transbordo |
| `POST` | `/bots/:id/test` | Envia uma mensagem de teste e devolve a resposta do bot |

### Criar um bot, vincular a um canal e ativar

Um bot nasce `inactive` e sem canais. São três chamadas:

```bash
# 1) Cria
BOT_ID=$(curl -s -X POST "$LINKTOR_URL/api/v1/bots" \
  -H "X-API-Key: $LINKTOR_API_KEY" -H "Content-Type: application/json" \
  -d '{
    "name": "Atendente virtual",
    "type": "ai",
    "provider": "openai",
    "model": "gpt-4o-mini",
    "system_prompt": "Você é o atendente da Sua Empresa. Responda em português, de forma objetiva.",
    "temperature": 0.4,
    "max_tokens": 800
  }' | jq -r '.data.id')

# 2) Vincula ao canal
curl -X POST "$LINKTOR_URL/api/v1/bots/$BOT_ID/channels" \
  -H "X-API-Key: $LINKTOR_API_KEY" -H "Content-Type: application/json" \
  -d '{ "channel_id": "CHANNEL_ID" }'

# 3) Ativa
curl -X POST "$LINKTOR_URL/api/v1/bots/$BOT_ID/activate" -H "X-API-Key: $LINKTOR_API_KEY"
```

Campos de `POST /bots`:

| Campo | Obrigatório | Descrição |
|---|---|---|
| `name` | sim | Nome do bot |
| `type` | sim | `ai`, `rule_based` ou `hybrid` |
| `provider` | sim | `openai`, `anthropic` ou `ollama` |
| `model` | sim | Um modelo do provedor — consulte `GET /ai/providers` |
| `system_prompt` | não | Instruções do bot. Há um prompt padrão em inglês se ausente |
| `temperature` | não | Padrão `0.7` |
| `max_tokens` | não | Padrão `1024` |

:::note
A criação não valida `type`, `provider` nem `model` contra as listas acima — um valor incorreto é
gravado e só falha quando o bot tenta responder. Confira com `GET /ai/providers` antes.
:::

Resposta (`201`):

```json
{
  "success": true,
  "data": {
    "id": "5d2b8a3e-1c1f-4e0f-9a57-7f1c1b2d3e4f",
    "tenant_id": "4f3d…",
    "name": "Atendente virtual",
    "type": "ai",
    "provider": "openai",
    "model": "gpt-4o-mini",
    "config": {
      "system_prompt": "Você é o atendente da Sua Empresa. Responda em português, de forma objetiva.",
      "temperature": 0.4,
      "max_tokens": 800,
      "confidence_threshold": 0.7,
      "escalation_rules": [],
      "knowledge_base_id": null,
      "welcome_message": null,
      "fallback_message": "I'm sorry, I couldn't understand your request. Let me connect you with a human agent.",
      "working_hours": null,
      "context_window_size": 10,
      "enabled_intents": null,
      "max_response_length": 500,
      "enable_vre_tools": false
    },
    "status": "inactive",
    "channels": [],
    "created_at": "2026-11-24T14:00:00Z",
    "updated_at": "2026-11-24T14:00:00Z"
  }
}
```

`status` é `active`, `inactive` ou `training`. `channels` lista os ids dos canais vinculados.

### Testar um bot

`POST /bots/:id/test` — scope `bots:write`. Envia uma mensagem como se fosse o cliente e devolve o que o
bot responderia, sem tocar em nenhuma conversa real.

<Tabs groupId="linguagem">
<TabItem value="curl" label="cURL">

```bash
curl -X POST "$LINKTOR_URL/api/v1/bots/BOT_ID/test" \
  -H "X-API-Key: $LINKTOR_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{ "message": "Qual o prazo de entrega para São Paulo?" }'
```

</TabItem>
<TabItem value="typescript" label="TypeScript">

```typescript
const base = `${process.env.LINKTOR_URL}/api/v1`;

const res = await fetch(`${base}/bots/BOT_ID/test`, {
  method: 'POST',
  headers: {
    'X-API-Key': process.env.LINKTOR_API_KEY!,
    'Content-Type': 'application/json',
  },
  body: JSON.stringify({ message: 'Qual o prazo de entrega para São Paulo?' }),
});
if (!res.ok) throw new Error(`HTTP ${res.status}: ${await res.text()}`);

const { data } = await res.json();
console.log(data.content, data.confidence, data.should_escalate);
```

</TabItem>
<TabItem value="python" label="Python">

```python
import os
import requests

base = f"{os.environ['LINKTOR_URL']}/api/v1"

res = requests.post(
    f"{base}/bots/BOT_ID/test",
    headers={"X-API-Key": os.environ["LINKTOR_API_KEY"]},
    json={"message": "Qual o prazo de entrega para São Paulo?"},
    timeout=60,
)
res.raise_for_status()
data = res.json()["data"]
print(data["content"], data["confidence"], data["should_escalate"])
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
	payload, _ := json.Marshal(map[string]string{"message": "Qual o prazo de entrega para São Paulo?"})

	req, _ := http.NewRequest("POST", os.Getenv("LINKTOR_URL")+"/api/v1/bots/BOT_ID/test", bytes.NewReader(payload))
	req.Header.Set("X-API-Key", os.Getenv("LINKTOR_API_KEY"))
	req.Header.Set("Content-Type", "application/json")

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		panic(err)
	}
	defer res.Body.Close()
	raw, _ := io.ReadAll(res.Body)
	if res.StatusCode != http.StatusOK {
		panic(fmt.Sprintf("HTTP %d: %s", res.StatusCode, raw))
	}

	var body struct {
		Data struct {
			Content        string  `json:"content"`
			Confidence     float64 `json:"confidence"`
			ShouldEscalate bool    `json:"should_escalate"`
		} `json:"data"`
	}
	_ = json.Unmarshal(raw, &body)
	fmt.Println(body.Data.Content, body.Data.Confidence, body.Data.ShouldEscalate)
}
```

</TabItem>
<TabItem value="java" label="Java">

```java
import java.net.URI;
import java.net.http.HttpClient;
import java.net.http.HttpRequest;
import java.net.http.HttpResponse;

public class TestarBot {
    public static void main(String[] args) throws Exception {
        String base = System.getenv("LINKTOR_URL") + "/api/v1";

        HttpRequest req = HttpRequest.newBuilder()
            .uri(URI.create(base + "/bots/BOT_ID/test"))
            .header("X-API-Key", System.getenv("LINKTOR_API_KEY"))
            .header("Content-Type", "application/json")
            .POST(HttpRequest.BodyPublishers.ofString(
                "{\"message\":\"Qual o prazo de entrega para São Paulo?\"}"))
            .build();

        HttpResponse<String> res = HttpClient.newHttpClient()
            .send(req, HttpResponse.BodyHandlers.ofString());
        if (res.statusCode() != 200) throw new RuntimeException("HTTP " + res.statusCode() + ": " + res.body());

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

var res = await http.PostAsJsonAsync($"{baseUrl}/bots/BOT_ID/test",
    new { message = "Qual o prazo de entrega para São Paulo?" });
res.EnsureSuccessStatusCode();

using var doc = JsonDocument.Parse(await res.Content.ReadAsStringAsync());
var data = doc.RootElement.GetProperty("data");
Console.WriteLine(data.GetProperty("content").GetString());
Console.WriteLine(data.GetProperty("should_escalate").GetBoolean());
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

$res = $http->post('bots/BOT_ID/test', [
    'json' => ['message' => 'Qual o prazo de entrega para São Paulo?'],
]);
$data = json_decode((string) $res->getBody(), true)['data'];

echo $data['content'], PHP_EOL;
var_dump($data['should_escalate']);
```

</TabItem>
<TabItem value="rust" label="Rust">

```rust
use serde_json::{json, Value};

#[tokio::main]
async fn main() -> Result<(), Box<dyn std::error::Error>> {
    let base = format!("{}/api/v1", std::env::var("LINKTOR_URL")?);

    let body: Value = reqwest::Client::new()
        .post(format!("{base}/bots/BOT_ID/test"))
        .header("X-API-Key", std::env::var("LINKTOR_API_KEY")?)
        .json(&json!({ "message": "Qual o prazo de entrega para São Paulo?" }))
        .send()
        .await?
        .error_for_status()?
        .json()
        .await?;

    println!("{}", body["data"]["content"]);
    println!("{}", body["data"]["should_escalate"]);
    Ok(())
}
```

</TabItem>
</Tabs>

Resposta (`200`):

```json
{
  "success": true,
  "data": {
    "content": "Para São Paulo capital o prazo é de 2 dias úteis.",
    "confidence": 0.86,
    "intent": { "name": "question", "confidence": 0.9 },
    "sentiment": "neutral",
    "should_escalate": false,
    "tokens_used": 412,
    "latency_ms": 1380
  }
}
```

Quando o bot decide transbordar, `should_escalate` vem `true` com `escalate_reason`. `quick_replies`,
`actions`, `flow_id` e `flow_ended` aparecem quando há botões ou um fluxo envolvido.

### Ajustar a configuração

`PUT /bots/:id/config` altera só os campos enviados:

```bash
curl -X PUT "$LINKTOR_URL/api/v1/bots/BOT_ID/config" \
  -H "X-API-Key: $LINKTOR_API_KEY" -H "Content-Type: application/json" \
  -d '{
    "welcome_message": "Olá! Sou o assistente virtual da Sua Empresa.",
    "fallback_message": "Não entendi. Vou chamar um atendente.",
    "confidence_threshold": 0.6,
    "context_window_size": 12,
    "knowledge_base_id": "KB_ID",
    "working_hours": {
      "enabled": true,
      "timezone": "America/Sao_Paulo",
      "schedule": [
        { "day": 1, "start_time": "09:00", "end_time": "18:00" },
        { "day": 2, "start_time": "09:00", "end_time": "18:00" }
      ]
    }
  }'
```

| Campo | Descrição |
|---|---|
| `system_prompt`, `temperature`, `max_tokens` | Parâmetros do modelo |
| `context_window_size` | Quantas mensagens anteriores entram no contexto |
| `welcome_message`, `fallback_message` | Mensagem de boas-vindas e de "não entendi" |
| `confidence_threshold` | Confiança mínima para o bot responder sozinho |
| `knowledge_base_id` | Base de conhecimento consultada pelo bot |
| `working_hours` | `enabled`, `timezone`, `schedule[]` com `day` (0 = domingo … 6 = sábado), `start_time`, `end_time` |
| `escalation_rules` | Substitui a lista inteira de regras de transbordo |

`PUT /bots/:id` é a versão curta: aceita `name`, `model`, `system_prompt`, `temperature`, `max_tokens`,
`welcome_message` e `fallback_message`.

### Regras de transbordo para humano

```bash
curl -X POST "$LINKTOR_URL/api/v1/bots/BOT_ID/escalation-rules" \
  -H "X-API-Key: $LINKTOR_API_KEY" -H "Content-Type: application/json" \
  -d '{ "condition": "keyword", "value": "falar com atendente", "priority": "high" }'
```

| Campo | Valores |
|---|---|
| `condition` (obrigatório) | `low_confidence`, `sentiment`, `keyword`, `intent`, `user_request` |
| `value` | Limite, palavra-chave, sentimento ou intenção, conforme a condição |
| `priority` | `normal`, `high`, `urgent` |

Para o que acontece com a conversa depois do transbordo, veja [Conversas](conversas.md).

## Base de conhecimento

Uma base é uma coleção de pares pergunta/resposta que o bot consulta antes de responder.

| Método | Caminho | O que faz |
|---|---|---|
| `GET` | `/knowledge-bases` | Lista as bases |
| `POST` | `/knowledge-bases` | Cria — `201` |
| `GET` / `PUT` / `DELETE` | `/knowledge-bases/:id` | Obtém, atualiza (`name`, `description`, `config`), remove (`204`) |
| `GET` | `/knowledge-bases/:id/items` | Lista os itens |
| `POST` | `/knowledge-bases/:id/items` | Acrescenta um item — `201` |
| `POST` | `/knowledge-bases/:id/items/bulk` | Acrescenta até 100 itens |
| `GET` / `PUT` / `DELETE` | `/knowledge-bases/:id/items/:itemId` | Obtém, atualiza, remove (`204`) um item |
| `POST` | `/knowledge-bases/:id/search` | Busca |
| `POST` | `/knowledge-bases/:id/regenerate-embeddings` | Recalcula os vetores de todos os itens |

### Criar a base e carregar itens

```bash
# Cria a base (type: faq, documents ou website)
KB_ID=$(curl -s -X POST "$LINKTOR_URL/api/v1/knowledge-bases" \
  -H "X-API-Key: $LINKTOR_API_KEY" -H "Content-Type: application/json" \
  -d '{ "name": "FAQ da loja", "description": "Perguntas frequentes", "type": "faq" }' \
  | jq -r '.data.id')

# Carrega itens em lote (máximo 100 por chamada)
curl -X POST "$LINKTOR_URL/api/v1/knowledge-bases/$KB_ID/items/bulk" \
  -H "X-API-Key: $LINKTOR_API_KEY" -H "Content-Type: application/json" \
  -d '{
    "items": [
      {
        "question": "Qual o prazo de entrega?",
        "answer": "2 dias úteis para capitais e até 7 dias úteis para o interior.",
        "keywords": ["prazo", "entrega", "frete"]
      },
      {
        "question": "Como troco um produto?",
        "answer": "Em até 30 dias, pelo link enviado no e-mail de confirmação.",
        "keywords": ["troca", "devolução"]
      }
    ]
  }'
```

Resposta do lote (`200`, sem envelope). Itens inválidos não derrubam os demais — confira `errors`:

```json
{
  "created": 2,
  "items": [
    {
      "id": "e7a1…",
      "knowledge_base_id": "KB_ID",
      "question": "Qual o prazo de entrega?",
      "answer": "2 dias úteis para capitais e até 7 dias úteis para o interior.",
      "keywords": ["prazo", "entrega", "frete"],
      "created_at": "2026-11-24T14:20:00Z",
      "updated_at": "2026-11-24T14:20:00Z"
    }
  ],
  "errors": []
}
```

Um item aceita `question` e `answer` (obrigatórios), `keywords`, `source` e `metadata` (mapa de texto).

### Buscar

```bash
curl -X POST "$LINKTOR_URL/api/v1/knowledge-bases/KB_ID/search" \
  -H "X-API-Key: $LINKTOR_API_KEY" -H "Content-Type: application/json" \
  -d '{ "query": "quanto tempo demora para chegar", "limit": 3 }'
```

`limit` tem padrão `5` e máximo `20`. Resposta (sem envelope):

```json
{
  "results": [
    {
      "item": {
        "id": "e7a1…",
        "knowledge_base_id": "KB_ID",
        "question": "Qual o prazo de entrega?",
        "answer": "2 dias úteis para capitais e até 7 dias úteis para o interior."
      },
      "score": 0.91
    }
  ],
  "query": "quanto tempo demora para chegar",
  "count": 1
}
```

Para o bot usar a base, grave o id em `knowledge_base_id` com `PUT /bots/:id/config`.

## Fluxos

Um fluxo é um roteiro determinístico: nós ligados por transições. Não confunda com os
*WhatsApp Flows* da Meta, que ficam em `/whatsapp/flows`.

| Método | Caminho | O que faz |
|---|---|---|
| `GET` | `/flows` | Lista (`bot_id`, `is_active`, `trigger`, `page`, `page_size`, `sort_by`, `sort_dir`) |
| `POST` | `/flows` | Cria — `201` |
| `GET` / `PUT` / `DELETE` | `/flows/:id` | Obtém, atualiza, remove |
| `POST` | `/flows/:id/activate` | Ativa |
| `POST` | `/flows/:id/deactivate` | Desativa |
| `POST` | `/flows/:id/test` | Simula o fluxo com uma sequência de respostas |

### Criar um fluxo

```bash
curl -X POST "$LINKTOR_URL/api/v1/flows" \
  -H "X-API-Key: $LINKTOR_API_KEY" -H "Content-Type: application/json" \
  -d '{
    "name": "Triagem inicial",
    "description": "Pergunta o assunto e encaminha",
    "trigger": "keyword",
    "trigger_value": "menu",
    "start_node_id": "inicio",
    "priority": 10,
    "nodes": [
      {
        "id": "inicio",
        "type": "question",
        "content": "Sobre o que você quer falar?",
        "quick_replies": [
          { "id": "vendas", "title": "Vendas" },
          { "id": "suporte", "title": "Suporte" }
        ],
        "transitions": [
          { "id": "t1", "to_node_id": "vendas", "condition": "reply_equals", "value": "Vendas", "priority": 2 },
          { "id": "t2", "to_node_id": "suporte", "condition": "default", "priority": 1 }
        ]
      },
      {
        "id": "vendas",
        "type": "message",
        "content": "Certo! Um consultor já vai falar com você.",
        "actions": [ { "type": "escalate", "config": {} } ],
        "transitions": [ { "id": "t3", "to_node_id": "fim", "condition": "default", "priority": 1 } ]
      },
      {
        "id": "suporte",
        "type": "message",
        "content": "Descreva o problema e já retornamos.",
        "transitions": [ { "id": "t4", "to_node_id": "fim", "condition": "default", "priority": 1 } ]
      },
      { "id": "fim", "type": "end", "transitions": [] }
    ]
  }'
```

| Campo | Obrigatório | Valores |
|---|---|---|
| `name` | sim | — |
| `trigger` | sim | `intent`, `keyword`, `manual`, `welcome` (primeira mensagem da conversa) |
| `trigger_value` | não | Nome da intenção ou palavra-chave |
| `start_node_id` | sim | `id` do nó de entrada |
| `nodes` | sim (mínimo 1) | Veja abaixo |
| `bot_id` | não | Restringe o fluxo a um bot; sem ele o fluxo é global |
| `priority` | não | Fluxos de prioridade maior são avaliados primeiro |

Nó (`nodes[]`):

| Campo | Valores |
|---|---|
| `id` | Identificador único dentro do fluxo |
| `type` | `message`, `question`, `condition`, `action`, `vre`, `end` |
| `content` | Texto enviado |
| `quick_replies[]` | Botões: `id`, `title`, `value` (opcional) |
| `transitions[]` | `id`, `to_node_id`, `condition` (`default`, `reply_equals`, `contains`, `regex`, `entity`), `value`, `priority` (maior é avaliado primeiro) |
| `actions[]` | `type` (`tag`, `assign`, `escalate`, `set_entity`, `http_call`, `notify`) e `config` (objeto livre) |
| `vre_config` | Para nós `vre`: `template_id`, `data_mapping`, `caption`, `follow_up_text` |

:::note
O formato de `config` de cada tipo de ação não é validado pela API e não está documentado aqui. Monte o
fluxo no editor visual do painel e leia-o com `GET /flows/:id` para ver a estrutura exata.
:::

O fluxo nasce inativo (`is_active: false`); ative com `POST /flows/:id/activate`. `PUT /flows/:id`
aceita `name`, `description`, `trigger_value`, `start_node_id`, `nodes` e `priority` — o `trigger` não
muda depois da criação.

### Simular

```bash
curl -X POST "$LINKTOR_URL/api/v1/flows/FLOW_ID/test" \
  -H "X-API-Key: $LINKTOR_API_KEY" -H "Content-Type: application/json" \
  -d '{ "inputs": ["menu", "Vendas"] }'
```

```json
{
  "flow_id": "FLOW_ID",
  "results": [
    {
      "message": "Sobre o que você quer falar?",
      "quick_replies": [ { "id": "vendas", "title": "Vendas" }, { "id": "suporte", "title": "Suporte" } ],
      "flow_ended": false,
      "should_wait": true
    },
    {
      "message": "Certo! Um consultor já vai falar com você.",
      "actions": [ { "type": "escalate", "config": {} } ],
      "flow_ended": false,
      "should_wait": false
    }
  ]
}
```

:::note
Em fluxo de outro tenant, as rotas de `/flows/:id` respondem `403 FORBIDDEN`
(`Flow does not belong to tenant`), e não `404` como no restante da API.
:::

## Serviços de IA

Chamadas avulsas aos provedores de IA configurados no servidor. Todas exigem o scope `ai:use`. Os
provedores disponíveis dependem das chaves configuradas na instalação — veja
[Configuração](../implantar/configuracao.md).

| Método | Caminho | O que faz |
|---|---|---|
| `GET` | `/ai/providers` | Provedores registrados, disponibilidade e modelos |
| `GET` | `/ai/providers/:provider/models` | Modelos de um provedor |
| `POST` | `/ai/complete` | Completa uma conversa (chat completion) |
| `POST` | `/ai/classify-intent` | Classifica a intenção de um texto |
| `POST` | `/ai/analyze-sentiment` | Classifica o sentimento de um texto |
| `POST` | `/ai/analyze-message` | Intenção + sentimento + decisão de transbordo para uma mensagem de uma conversa |
| `POST` | `/ai/generate-response` | Gera a resposta do bot do canal para uma mensagem |
| `POST` | `/ai/escalate` | Transborda uma conversa para atendimento humano |

### Ver o que está disponível

```bash
curl "$LINKTOR_URL/api/v1/ai/providers" -H "X-API-Key: $LINKTOR_API_KEY"
```

```json
{
  "success": true,
  "data": {
    "providers": [
      { "name": "openai", "available": true, "default_model": "gpt-4o-mini", "models": ["gpt-4o-mini", "gpt-4o"] }
    ]
  }
}
```

Os nomes de modelo acima são ilustrativos; use os que a sua instalação devolver.

### Completar

```bash
curl -X POST "$LINKTOR_URL/api/v1/ai/complete" \
  -H "X-API-Key: $LINKTOR_API_KEY" -H "Content-Type: application/json" \
  -d '{
    "provider": "openai",
    "model": "gpt-4o-mini",
    "messages": [
      { "role": "system", "content": "Resuma em uma frase." },
      { "role": "user", "content": "O cliente pediu troca do produto porque veio no tamanho errado e quer frete grátis na devolução." }
    ],
    "max_tokens": 200,
    "temperature": 0.2
  }'
```

- `provider`, `model` e `messages` são obrigatórios. `role` é `system`, `user` ou `assistant`.
- `provider` precisa ser `openai`, `anthropic` ou `ollama` e estar disponível; `model` precisa estar na
  lista do provedor. Fora disso: `400 VALIDATION_ERROR` (`Provider not allowed`, `Provider not available`
  ou `Model not allowed`).
- `max_tokens` tem padrão `1024` e teto `4096` (valores maiores são reduzidos ao teto).
  `temperature` tem padrão `0.7`.

```json
{
  "success": true,
  "data": {
    "content": "Cliente solicita troca por tamanho errado, com devolução sem custo de frete.",
    "model": "gpt-4o-mini",
    "tokens_used": 68,
    "prompt_tokens": 49,
    "completion_tokens": 19,
    "finish_reason": "stop",
    "latency_ms": 742
  }
}
```

### Classificar intenção e sentimento

```bash
curl -X POST "$LINKTOR_URL/api/v1/ai/classify-intent" \
  -H "X-API-Key: $LINKTOR_API_KEY" -H "Content-Type: application/json" \
  -d '{ "message": "quero cancelar meu pedido", "intents": ["cancelamento", "troca", "duvida"] }'
# data: { "intent": { "name": "cancelamento", "confidence": 0.93 }, "candidates": [ … ] }

curl -X POST "$LINKTOR_URL/api/v1/ai/analyze-sentiment" \
  -H "X-API-Key: $LINKTOR_API_KEY" -H "Content-Type: application/json" \
  -d '{ "message": "já é a terceira vez que reclamo e ninguém resolve" }'
# data: { "sentiment": "negative", "score": -0.8, "confidence": 0.9 }
```

- `provider` é opcional nas duas rotas; o padrão é `openai`.
- Sem `intents`, a classificação usa a lista padrão: `greeting`, `farewell`, `question`, `complaint`,
  `support_request`, `sales_inquiry`, `feedback`, `escalation_request`, `general`.
- `sentiment` é `positive`, `neutral` ou `negative`.

### Analisar ou responder uma mensagem de uma conversa

`POST /ai/analyze-message` recebe `conversation_id`, `channel_id` e `content` (todos obrigatórios) e
devolve `intent`, `sentiment`, `should_escalate`, `escalate_reason`, `keywords` e o `bot` do canal.

`POST /ai/generate-response` gera a resposta que o bot vinculado ao canal daria:

```bash
curl -X POST "$LINKTOR_URL/api/v1/ai/generate-response" \
  -H "X-API-Key: $LINKTOR_API_KEY" -H "Content-Type: application/json" \
  -d '{
    "message_id": "MESSAGE_ID",
    "conversation_id": "CONVERSATION_ID",
    "channel_id": "CHANNEL_ID",
    "content": "Qual o prazo de entrega?"
  }'
```

```json
{
  "success": true,
  "data": {
    "response": "2 dias úteis para capitais e até 7 dias úteis para o interior.",
    "confidence": 0.88,
    "tokens_used": 391,
    "latency_ms": 1210,
    "model": "gpt-4o-mini",
    "should_escalate": false
  }
}
```

:::note
O campo `bot_id` é aceito no corpo de `/ai/generate-response`, mas hoje é ignorado: o bot usado é
sempre o vinculado ao `channel_id`.
:::

### Transbordar para humano

```bash
curl -X POST "$LINKTOR_URL/api/v1/ai/escalate" \
  -H "X-API-Key: $LINKTOR_API_KEY" -H "Content-Type: application/json" \
  -d '{
    "conversation_id": "CONVERSATION_ID",
    "channel_id": "CHANNEL_ID",
    "contact_id": "CONTACT_ID",
    "reason": "Cliente pediu atendente",
    "priority": "high"
  }'
```

`conversation_id`, `channel_id` e `contact_id` são obrigatórios. `priority` aceita `low`, `normal`
(padrão), `high` e `urgent`; `reason` tem um texto padrão se ausente.

```json
{
  "success": true,
  "data": {
    "conversation_id": "CONVERSATION_ID",
    "status": "pending",
    "queue_position": 2,
    "escalated_at": "2026-11-24T14:31:00Z"
  }
}
```

O valor de `status` e a presença de `queue_position`, `estimated_wait_seconds` e `assigned_user_id`
dependem da fila de atendimento no momento. Para transbordar a partir da própria conversa, existe
também `POST /conversations/:id/escalate` — veja [Conversas](conversas.md).

## Erros

| Status | `error.code` | Quando |
|---|---|---|
| `400` | `VALIDATION_ERROR` | Corpo inválido, campo obrigatório ausente, provedor ou modelo não permitido |
| `401` | `UNAUTHORIZED` | Credencial ausente ou inválida |
| `403` | `FORBIDDEN` | API key sem o scope da rota; fluxo de outro tenant |
| `404` | `NOT_FOUND` | Bot, base ou item inexistente ou de outro tenant |

Formato completo em [Erros e paginação](erros-e-paginacao.md).
