---
sidebar_position: 1
title: Visão geral
description: Os sete SDKs oficiais do Linktor, o que cada um cobre de fato hoje, como instalar e o que fazer quando o SDK não cobre uma rota.
---

# SDKs — visão geral

O Linktor tem SDKs em sete linguagens, todos no mesmo repositório do servidor, em `sdks/`. Eles são
clientes finos sobre a [API REST](../api/visao-geral.md): nenhum faz algo que você não consiga fazer
com um cliente HTTP.

:::warning Leia a matriz de cobertura antes de escolher
Os SDKs não estão no mesmo nível. O recurso de **canais** (criar, conectar por QR, pair code,
atualizar, remover) foi reconciliado com o servidor nos sete. O restante — conversas, contatos, bots,
IA, base de conhecimento, fluxos, analytics — foi escrito contra uma versão anterior da API e, em boa
parte, chama rotas que não existem mais ou envia campos em camelCase que o servidor ignora. A
[matriz abaixo](#matriz-de-cobertura) diz o que funciona em cada linguagem; cada página de linguagem
tem uma seção "Limitações conhecidas" com o detalhe método a método.

Para o que o SDK não cobre, use o cliente HTTP nativo da linguagem — todas as páginas trazem o
exemplo. É o caminho que funciona hoje para **enviar mensagens** em cinco das sete linguagens.
:::

## SDKs disponíveis

| Linguagem | Pacote (conforme o manifest) | Versão | Runtime mínimo | Caminho no repositório |
|---|---|---|---|---|
| [TypeScript](./typescript.md) | `@linktor/sdk` (npm) | 1.0.0 | Node.js 18 | `sdks/typescript` |
| [Python](./python.md) | `linktor` (PyPI) | 1.0.0 | 3.9 declarado; **3.10 na prática** | `sdks/python` |
| [Go](./go.md) | módulo `github.com/linktor/linktor-go` | sem tag | Go 1.21 | `sdks/go` |
| [Java](./java.md) | `io.linktor:linktor-sdk` (Maven) | 1.0.0 | Java 17 | `sdks/java` |
| [C# / .NET](./dotnet.md) | `Linktor` (NuGet) | 1.0.0 | .NET 8 (`net8.0`) | `sdks/dotnet` |
| [PHP](./php.md) | `linktor/linktor-php` (Composer) | sem versão declarada | PHP 8.1 | `sdks/php` |
| [Rust](./rust.md) | `linktor` (crate) | 1.0.0 | edição 2021 (sem `rust-version`) | `sdks/rust` |

:::note Instale a partir do repositório
Os nomes de pacote acima são os declarados em cada manifest. Não há como confirmar, a partir do
código, que eles estão publicados no npm, PyPI, Maven Central, NuGet, Packagist ou crates.io — e os
campos `repository` dos manifests apontam para repositórios `github.com/linktor/linktor-*` que não
são o repositório real. O caminho garantido é instalar a partir de um clone de
`github.com/edsonmartins/linktor` (dependência por caminho ou por git). Cada página de linguagem
mostra o comando exato.
:::

Particularidades de instalação que valem saber antes:

- **Go** — o `go.mod` declara `github.com/linktor/linktor-go`, mas o código vive em
  `sdks/go` deste repositório. Use `replace` apontando para o clone.
- **TypeScript** — `npm run build` falha na geração de tipos (quatro erros de `tsc` no fonte). A
  página mostra o contorno para gerar o pacote local.
- **C# / .NET** — o projeto **não compila como está**: `Resources/*.cs` usa oito tipos que não são
  declarados em nenhum arquivo (`MessageMedia`, `QuickReply`, `BotConfig`, `AgentConfig`,
  `AgentTool`, `TokenUsage`, `KnowledgeBaseConfig`, `FlowSettings`). A página mostra o contorno.
- **Python** — `import linktor` falha em Python 3.9 (o fonte usa `str | bytes` em tempo de execução),
  apesar de o `pyproject.toml` declarar `>=3.9`.
- **PHP** — o Guzzle descarta o `/api/v1` da base URL porque os recursos usam caminhos absolutos, e
  o autoload PSR-4 não encontra classes declaradas em arquivo de outro nome. A página traz os dois
  contornos. Este SDK não pôde ser executado na verificação desta documentação (só leitura do fonte).

## Matriz de cobertura {#matriz-de-cobertura}

Construída lendo o fonte de cada SDK e comparando rota, verbo, corpo e resposta com o servidor
(`cmd/server/main.go` e `internal/api/handlers/`).

- **Sim** — funciona contra o servidor atual.
- **Parcial** — parte dos métodos funciona; o texto diz qual.
- **Quebrado** — o SDK tem o recurso, mas ele não funciona contra o servidor atual (rota
  inexistente, corpo incompatível ou resposta que não desserializa).
- **—** — o SDK não tem o recurso.

| Recurso | TypeScript | Python | Go | Java | C# / .NET | PHP | Rust |
|---|---|---|---|---|---|---|---|
| Canais (CRUD, QR, pair code) | Sim | Parcial ¹ | Sim | Sim | Sim | Sim | Sim |
| Envio direto (`POST /messages/send`) | Sim ² | — | Sim ² | — | — | — | — |
| Enviar para uma conversa | Quebrado | Quebrado | Quebrado | Quebrado | Quebrado | Parcial ³ | Quebrado |
| Conversas | Parcial: `list`, `get`, `listMessages` | Quebrado | Parcial: `List`, `Get` | Parcial: `get`, `resolve`, `reopen` | Parcial: `DeleteAsync`, `GetMessagesAsync` | Parcial: `get`, `create`, `delete`, `resolve`, `reopen` | Quebrado |
| Contatos | Parcial: `list`, `search`, `get`, `create`, `delete` | Parcial: só `delete` | Parcial: `List`, `Get`, `Create`, `Delete` | Parcial: `get`, `create`, `delete` | Parcial: listar, obter, criar, remover | Parcial: `get`, `create`, `delete` | Parcial: só `delete` |
| Bots | Parcial: `list`, `get`, `delete` | Parcial: só `delete` | Parcial: `List`, `Get` | Parcial: `get`, `delete` | Parcial: listar, obter, remover | Parcial: `get`, `create`, `delete` | Parcial: só `delete` |
| IA | Quebrado | Quebrado | Quebrado | Quebrado | Quebrado | Quebrado | Quebrado |
| Base de conhecimento | Parcial: `list`, `get`, `delete` | Parcial: só `delete` | Parcial: `List`, `Get` | Parcial: `get`, `delete` | Parcial: listar, obter, remover | Parcial: `get`, `create`, `delete` | Parcial: só `delete` |
| Fluxos | Parcial: `list`, `get`, `delete` | Parcial: só `delete` | Parcial: `List`, `Get` | Parcial: `get`, `delete` | Parcial: listar, obter, remover | Parcial: `get`, `create`, `delete` | Parcial: só `delete` |
| Analytics | Parcial: conversas e canais | Parcial: conversas | Quebrado | Quebrado | — | — | — |
| VRE | Parcial: `render`, `listTemplates` | Parcial: `render` | Parcial: `Render` | Parcial: `render`, `listTemplates` | Parcial: `RenderAsync` | Parcial: `render` | Parcial: `render` |
| Verificação de webhook | Quebrado ⁴ | Quebrado ⁴ | **Sim** | Quebrado ⁴ | Quebrado ⁴ | Quebrado ⁴ | Quebrado ⁴ |
| WebSocket | Quebrado ⁵ | — | — | Quebrado ⁵ | — | — | — |
| API key no header `X-API-Key` | Sim | Sim | Sim | Sim | Sim | Sim | Sim |

Notas:

1. **Python, canais** — o enum `ChannelType` do SDK só conhece `whatsapp`, `telegram`, `facebook`,
   `instagram`, `webchat`, `sms`, `email` e `rcs`. Para os demais tipos (`whatsapp_unofficial`,
   `whatsapp_official`, `voice`, `teams`, `slack`, `mattermost`, `direto`) o servidor executa a
   operação e o SDK lança `ValidationError` ao ler a resposta; `channels.list()` falha se o tenant
   tiver qualquer canal desses tipos.
2. **Envio direto em TypeScript e Go** — o corpo enviado está correto e a mensagem é aceita, mas o
   SDK não desembrulha o envelope `{success, data}` desta rota. Em TypeScript o resultado está em
   `.data` do que o método devolve; em Go o resultado volta vazio com `err == nil` (a página mostra
   um adaptador). O `DirectSendInput` do Go só cobre texto; para mídia use HTTP nativo.
3. **PHP, enviar para conversa** — a mensagem é enviada, mas o objeto `Message` devolvido vem sem o
   texto e sem os campos compostos.
4. **Verificação de webhook** — o servidor assina `timestamp + "." + corpo`; estes verificadores
   assinam só o corpo, então recusam toda entrega legítima. Veja
   [Assinatura do webhook](#assinatura-do-webhook).
5. **WebSocket** — os clientes de TypeScript e Java usam autenticação e protocolo (`subscribe`,
   tipos de evento, campo `data`) que o servidor não fala. Veja [Tempo real](../api/tempo-real.md) e
   use um cliente WebSocket nativo.

Defeitos que atravessam quase todos os SDKs e que você vai notar mesmo nos métodos que funcionam:

- **Base URL** — em TypeScript, Python, Go, Java, C# e Rust, passe a base **com** `/api/v1`
  (`https://linktor.suaempresa.com/api/v1`); o SDK não acrescenta o prefixo e o padrão embutido
  (`https://api.linktor.io`) não serve para uma instalação própria. Em PHP é o contrário — veja a
  página.
- **Mensagem de erro perdida** — os SDKs leem `code` e `message` na raiz do corpo, mas o servidor
  responde `{"success": false, "error": {"code", "message", "details"}}`. A exceção chega com o
  status HTTP certo e mensagem genérica (`"Unknown error"`, `"Request failed"` ou vazia). Para ver
  o motivo real, use HTTP nativo ou os contornos de cada página.
- **Novas tentativas também em POST** — 5xx e falha de rede são repetidos para qualquer verbo. Em
  envio de mensagem isso pode duplicar; reduza as tentativas onde o SDK permite.
- **`update` com PATCH** — fora de canais, os métodos de atualização usam `PATCH`; o servidor só tem
  `PUT`. Por isso nenhum `update` de conversa, contato, bot, base ou fluxo aparece na matriz.
- **`PUT /channels/{id}`** (comportamento do servidor, não do SDK) — `credentials` é mesclado com o
  que está gravado, mas `config` **substitui** o mapa inteiro. Ao atualizar `config`, reenvie as
  chaves que quer manter.

## Enviar a primeira mensagem

O mesmo envio direto — texto para `+5511999999999` pelo canal `CHANNEL_ID` — em cada linguagem.
TypeScript e Go usam o SDK; as demais usam o cliente HTTP nativo, porque o SDK não tem o método. A
chave precisa do scope `messages:send` ([Chaves de API](../painel/chaves-de-api.md)). A resposta é
`202` com `data.status` igual a `"queued"`: a mensagem foi persistida e enfileirada, não entregue —
a entrega chega pelo [webhook](../api/receber-eventos.md).

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
    "text": "Olá! Seu pedido saiu para entrega."
  }'
```

</TabItem>
<TabItem value="typescript" label="TypeScript">

```typescript
import { LinktorClient } from '@linktor/sdk';

const client = new LinktorClient({
  baseUrl: `${process.env.LINKTOR_URL}/api/v1`,
  apiKey: process.env.LINKTOR_API_KEY,
});

const res = await client.messages.sendText(
  'CHANNEL_ID',
  '+5511999999999',
  'Olá! Seu pedido saiu para entrega.',
);

// O SDK devolve o envelope { success, data } sem desembrulhar.
const envio = (res as unknown as { data: typeof res }).data;
console.log(envio.id, envio.status); // status: "queued"
```

</TabItem>
<TabItem value="python" label="Python">

```python
import os
import httpx  # já é dependência do SDK

api = httpx.Client(
    base_url=os.environ["LINKTOR_URL"].rstrip("/") + "/api/v1",
    headers={"X-API-Key": os.environ["LINKTOR_API_KEY"]},
    timeout=30.0,
)

resp = api.post("/messages/send", json={
    "channel_id": "CHANNEL_ID",
    "to": "+5511999999999",
    "content_type": "text",
    "text": "Olá! Seu pedido saiu para entrega.",
})
resp.raise_for_status()
msg = resp.json()["data"]
print(msg["id"], msg["conversation_id"], msg["status"])  # status == "queued"
```

</TabItem>
<TabItem value="go" label="Go">

```go
package main

import (
	"context"
	"log"
	"os"
	"time"

	linktor "github.com/linktor/linktor-go"
)

func main() {
	client := linktor.NewClient(
		linktor.WithBaseURL(os.Getenv("LINKTOR_URL")+"/api/v1"),
		linktor.WithAPIKey(os.Getenv("LINKTOR_API_KEY")),
	)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// O SDK não desembrulha o envelope desta rota: o resultado vem vazio.
	// err == nil significa que a mensagem entrou na fila (HTTP 202).
	_, err := client.Messages.SendDirectText(ctx, "CHANNEL_ID", "+5511999999999",
		"Olá! Seu pedido saiu para entrega.")
	if err != nil {
		log.Fatalf("envio recusado: %v", err)
	}
	log.Println("mensagem aceita na fila")
}
```

</TabItem>
<TabItem value="java" label="Java">

```java
import com.google.gson.Gson;
import com.google.gson.JsonObject;
import java.net.URI;
import java.net.http.HttpClient;
import java.net.http.HttpRequest;
import java.net.http.HttpResponse;
import java.time.Duration;
import java.util.Map;

public class EnviarMensagem {
    public static void main(String[] args) throws Exception {
        String baseUrl = System.getenv("LINKTOR_URL") + "/api/v1";
        Gson gson = new Gson(); // já é dependência do SDK

        String corpo = gson.toJson(Map.of(
                "channel_id", "CHANNEL_ID",
                "to", "+5511999999999",
                "content_type", "text",
                "text", "Olá! Seu pedido saiu para entrega."));

        HttpRequest request = HttpRequest.newBuilder(URI.create(baseUrl + "/messages/send"))
                .timeout(Duration.ofSeconds(30))
                .header("Content-Type", "application/json")
                .header("X-API-Key", System.getenv("LINKTOR_API_KEY"))
                .POST(HttpRequest.BodyPublishers.ofString(corpo))
                .build();

        HttpResponse<String> response = HttpClient.newHttpClient()
                .send(request, HttpResponse.BodyHandlers.ofString());
        if (response.statusCode() != 202) {
            throw new IllegalStateException("Envio recusado (" + response.statusCode() + "): " + response.body());
        }
        JsonObject data = gson.fromJson(response.body(), JsonObject.class).getAsJsonObject("data");
        System.out.println(data.get("id").getAsString() + " " + data.get("status").getAsString());
    }
}
```

</TabItem>
<TabItem value="csharp" label="C#">

```csharp
using System.Net.Http.Json;
using System.Text.Json;

var http = new HttpClient
{
    BaseAddress = new Uri(Environment.GetEnvironmentVariable("LINKTOR_URL")!.TrimEnd('/') + "/api/v1/"),
};
http.DefaultRequestHeaders.Add("X-API-Key", Environment.GetEnvironmentVariable("LINKTOR_API_KEY"));

var response = await http.PostAsJsonAsync("messages/send", new
{
    channel_id = "CHANNEL_ID",
    to = "+5511999999999",
    content_type = "text",
    text = "Olá! Seu pedido saiu para entrega.",
});

var body = await response.Content.ReadFromJsonAsync<JsonElement>();
if (!response.IsSuccessStatusCode)
{
    var error = body.GetProperty("error");
    throw new Exception($"{error.GetProperty("code")}: {error.GetProperty("message")}");
}

var data = body.GetProperty("data"); // 202: { id, conversation_id, channel_id, status: "queued" }
Console.WriteLine($"{data.GetProperty("id")} -> {data.GetProperty("status")}");
```

</TabItem>
<TabItem value="php" label="PHP">

```php
<?php
declare(strict_types=1);

require __DIR__ . '/vendor/autoload.php';

$http = new GuzzleHttp\Client([
    'base_uri'    => rtrim(getenv('LINKTOR_URL'), '/') . '/api/v1/',   // barra final obrigatória
    'headers'     => ['X-API-Key' => getenv('LINKTOR_API_KEY')],
    'timeout'     => 30,
    'http_errors' => false,
]);

$resposta = $http->post('messages/send', [   // caminho relativo: sem barra inicial
    'json' => [
        'channel_id'   => 'CHANNEL_ID',
        'to'           => '+5511999999999',
        'content_type' => 'text',
        'text'         => 'Olá! Seu pedido saiu para entrega.',
    ],
]);

$corpo = json_decode((string) $resposta->getBody(), true);

if ($resposta->getStatusCode() !== 202) {
    throw new RuntimeException($corpo['error']['code'] . ': ' . $corpo['error']['message']);
}

echo $corpo['data']['id'], ' ', $corpo['data']['status'], PHP_EOL;   // "queued"
```

</TabItem>
<TabItem value="rust" label="Rust">

```rust
// Cargo.toml: tokio (features = ["full"]), reqwest (features = ["json", "rustls-tls"]), serde_json
use serde_json::{json, Value};

#[tokio::main]
async fn main() -> Result<(), Box<dyn std::error::Error>> {
    let base = format!("{}/api/v1", std::env::var("LINKTOR_URL")?);
    let api_key = std::env::var("LINKTOR_API_KEY")?;
    let http = reqwest::Client::new();

    let resp = http
        .post(format!("{base}/messages/send"))
        .header("X-API-Key", &api_key)
        .json(&json!({
            "channel_id": "CHANNEL_ID",
            "to": "+5511999999999",
            "content_type": "text",
            "text": "Olá! Seu pedido saiu para entrega."
        }))
        .send()
        .await?;

    let status = resp.status();
    let corpo: Value = resp.json().await?;
    if !status.is_success() {
        return Err(format!("{status}: {}", corpo["error"]["message"]).into());
    }

    println!("{} {}", corpo["data"]["id"], corpo["data"]["status"]); // "..." "queued"
    Ok(())
}
```

</TabItem>
</Tabs>

Campos, mídia, metadados e idempotência do envio estão em
[Enviar mensagens](../api/enviar-mensagens.md).

## Quando o SDK não cobre uma rota

Use o cliente HTTP nativo da linguagem (`fetch`, `httpx`/`requests`, `net/http`, `java.net.http`,
`HttpClient`, Guzzle, `reqwest`). Três regras bastam:

1. Base `https://linktor.suaempresa.com/api/v1`, header `X-API-Key: lk_sua_chave`. Não envie a API
   key em `Authorization: Bearer` — esse header é reservado ao JWT de usuário e, se estiver presente
   e não for um JWT válido, a requisição é recusada com `401` sem tentar a API key.
2. Corpo e resposta em JSON **snake_case** (`channel_id`, `content_type`, `created_at`).
3. A resposta vem no envelope `{"success", "data", "error", "meta"}`; leia o resultado em `data` e
   o motivo de uma falha em `error.code` / `error.message`. Exceções: as rotas `/vre/*` e
   `/analytics/*` não usam o envelope padrão, e `DELETE` de canal, contato e conversa responde `204`
   sem corpo.

A referência completa de rotas está em [API — visão geral](../api/visao-geral.md); autenticação e
scopes em [Autenticação](../api/autenticacao.md); formato de erro e paginação em
[Erros e paginação](../api/erros-e-paginacao.md).

Em Java há um meio-termo: `client.getHttpClient()` é público e expõe `get`/`post`/`put`/`delete`
com a mesma autenticação e o desembrulho do envelope — veja a [página de Java](./java.md). Em Rust os
métodos genéricos de requisição são privados ao crate; use `reqwest`.

## camelCase nos modelos, snake_case no wire

A API fala snake_case em tudo, exceto no envelope do webhook de saída (veja abaixo). Nos SDKs, só
os tipos de **canal** (e, onde existe, os do envio direto e do render do VRE) declaram os nomes em
snake_case. Os demais modelos — conversa, mensagem, contato, bot, base de conhecimento, fluxo,
login — estão em camelCase (`conversationId`, `contentType`, `createdAt`, `accessToken`) e não há
conversão na serialização. O efeito muda por linguagem:

| Linguagem | Na leitura | Na escrita |
|---|---|---|
| TypeScript | O objeto chega em snake_case em tempo de execução; os tipos declarados mentem (`conversation.contactId` é `undefined`, o valor está em `contact_id`) | Campos compostos em camelCase são ignorados pelo servidor (`avatarUrl`, `customFields`) |
| Python | Os modelos aceitam snake_case (`populate_by_name`), mas a falta de desembrulho do envelope e campos obrigatórios ausentes fazem a validação falhar | Corpos fixos em camelCase (`refreshToken`, `agentId`, `topK`) |
| Go | Só campos de uma palavra chegam preenchidos (`id`, `status`, `tags`); os demais ficam no valor zero | `avatarUrl` e `customFields` são ignorados pelo servidor |
| Java | Campos compostos chegam `null` (Gson sem política de nomes) | Idem |
| C# / .NET | Campos compostos chegam no valor padrão; um valor de enum desconhecido derruba a desserialização | Corpos em camelCase recusados com `400` onde o campo é obrigatório (`contact_id`, `user_id`) |
| PHP | `fromArray` lê chaves camelCase: só campos de uma palavra chegam | Os recursos repassam o seu array — use chaves snake_case e funciona |
| Rust | Campos são obrigatórios e `rename_all = "camelCase"`: a desserialização falha e o método devolve `Err(Serialization)` **mesmo quando o servidor executou a operação** | Campos compostos ignorados pelo servidor |

Consequência prática: `auth.login` não instala o token em nenhum SDK (o servidor devolve
`access_token`, o SDK lê `accessToken`). Para integração servidor-a-servidor isso não importa — use
API key.

## Assinatura do webhook {#assinatura-do-webhook}

O servidor entrega os eventos de cada canal na `webhook_url` do canal, assinados com o
`credentials.webhook_secret` do canal:

| Item | Valor |
|---|---|
| Header da assinatura | `X-Linktor-Signature` — HMAC-SHA256 em hexadecimal minúsculo, **sem** prefixo `sha256=` |
| Header do timestamp | `X-Linktor-Timestamp` — epoch em segundos |
| O que é assinado | `<timestamp>` + `.` + corpo cru da requisição |
| Tolerância anti-replay | 300 segundos |

Só o SDK de **Go** (`sdks/go/webhook.go`) implementa esse contrato. Os verificadores de TypeScript,
Python, Java, C#, PHP e Rust calculam o HMAC apenas sobre o corpo e devolvem `false` para toda
entrega legítima. Cada página mostra a verificação manual correta dentro de um receptor mínimo
(Express, FastAPI, `net/http`, Spring Boot, ASP.NET minimal API, PHP puro, axum). PHP e Rust têm uma
função de nível mais baixo (`verifySignature` / `verify_signature`) que funciona se você mesmo montar
`timestamp.corpo`.

O envelope do webhook é a exceção de nomenclatura: ele é **camelCase** (`tenantId`,
`conversationId`, `contentType`). Os enums de tipo de evento dos SDKs estão defasados — não conhecem
`call.*` nem `conversation.reopened`, `conversation.escalated` e `conversation.deleted`, e listam
tipos que o servidor não emite. Trate o campo `type` como string. A lista de eventos e payloads está
em [Receber eventos](../api/receber-eventos.md).

## Próximos passos

- Escolha a linguagem: [TypeScript](./typescript.md), [Python](./python.md), [Go](./go.md),
  [Java](./java.md), [C# / .NET](./dotnet.md), [PHP](./php.md), [Rust](./rust.md).
- Crie uma chave com os scopes certos em [Chaves de API](../painel/chaves-de-api.md).
- Conecte um número por QR ou pair code: [WhatsApp não-oficial](../canais/whatsapp-nao-oficial.md).
