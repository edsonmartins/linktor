---
sidebar_position: 1
title: Visão geral da API
description: Base URL, formato das respostas, convenções, mapa dos recursos e a primeira chamada à API HTTP do Linktor.
---

# Visão geral da API

A API HTTP do Linktor fica em `/api/v1` no mesmo endereço do servidor. Tudo o que o painel faz passa
por ela: enviar e receber mensagens, gerenciar canais, contatos, conversas, modelos, campanhas, bots e
bases de conhecimento.

| O quê | Valor |
|---|---|
| Base da API | `https://linktor.suaempresa.com/api/v1` |
| Formato | JSON (`Content-Type: application/json`), exceto upload de arquivo (`multipart/form-data`) |
| Autenticação servidor a servidor | cabeçalho `X-API-Key: lk_sua_chave` |
| Autenticação de usuário | `Authorization: Bearer <jwt>` ou cookie de sessão |
| Nomes de campo | `snake_case` no corpo e na resposta |
| Datas | RFC 3339 em UTC (`2026-11-24T14:03:11Z`) |
| Identificadores | UUID em texto |

Os exemplos desta seção usam duas variáveis de ambiente:

```bash
export LINKTOR_URL="https://linktor.suaempresa.com"
export LINKTOR_API_KEY="lk_sua_chave"
```

## Primeira chamada em 2 minutos

1. No painel, abra **Configurações → Chaves de API** e clique em **Gerar nova chave**. Copie a chave
   (`lk_...`): ela aparece uma única vez. Detalhes em [Autenticação](./autenticacao.md).
2. Liste os canais do seu tenant:

<Tabs groupId="linguagem">
<TabItem value="curl" label="cURL">

```bash
curl "$LINKTOR_URL/api/v1/channels" \
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

const canais = await linktor.channels.list();
for (const canal of canais) {
  console.log(canal.id, canal.type, canal.name, canal.connection_status);
}
```

</TabItem>
<TabItem value="python" label="Python">

```python
import os
import requests

res = requests.get(
    f"{os.environ['LINKTOR_URL']}/api/v1/channels",
    headers={"X-API-Key": os.environ["LINKTOR_API_KEY"]},
    timeout=30,
)
res.raise_for_status()

for canal in res.json()["data"]:
    print(canal["id"], canal["type"], canal["name"], canal["connection_status"])
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
)

func main() {
	client := linktor.NewClient(
		linktor.WithBaseURL(os.Getenv("LINKTOR_URL")+"/api/v1"),
		linktor.WithAPIKey(os.Getenv("LINKTOR_API_KEY")),
	)

	canais, err := client.Channels.List(context.Background(), nil)
	if err != nil {
		log.Fatal(err)
	}
	for _, canal := range canais {
		fmt.Println(canal.ID, canal.Type, canal.Name, canal.ConnectionStatus)
	}
}
```

</TabItem>
<TabItem value="java" label="Java">

```java
import io.linktor.LinktorClient;
import io.linktor.types.Channel;

LinktorClient linktor = LinktorClient.builder()
    .baseUrl(System.getenv("LINKTOR_URL") + "/api/v1")
    .apiKey(System.getenv("LINKTOR_API_KEY"))
    .build();

for (Channel.ChannelModel canal : linktor.channels.list()) {
    System.out.println(canal.getId() + " " + canal.getType() + " " + canal.getName());
}
```

</TabItem>
<TabItem value="csharp" label="C#">

```csharp
using System.Net.Http.Json;
using System.Text.Json;

var http = new HttpClient();
http.DefaultRequestHeaders.Add("X-API-Key", Environment.GetEnvironmentVariable("LINKTOR_API_KEY"));

var body = await http.GetFromJsonAsync<JsonElement>(
    Environment.GetEnvironmentVariable("LINKTOR_URL") + "/api/v1/channels");

foreach (var canal in body.GetProperty("data").EnumerateArray())
{
    Console.WriteLine(
        $"{canal.GetProperty("id")} {canal.GetProperty("type")} {canal.GetProperty("name")} {canal.GetProperty("connection_status")}");
}
```

</TabItem>
<TabItem value="php" label="PHP">

```php
<?php
require 'vendor/autoload.php';

$http = new GuzzleHttp\Client(['timeout' => 30]);

$res = $http->get(getenv('LINKTOR_URL') . '/api/v1/channels', [
    'headers' => ['X-API-Key' => getenv('LINKTOR_API_KEY')],
]);

foreach (json_decode((string) $res->getBody(), true)['data'] as $canal) {
    echo "{$canal['id']} {$canal['type']} {$canal['name']} {$canal['connection_status']}\n";
}
```

</TabItem>
<TabItem value="rust" label="Rust">

```rust
use linktor::LinktorClient;

#[tokio::main]
async fn main() -> Result<(), Box<dyn std::error::Error>> {
    let linktor = LinktorClient::builder()
        .base_url(format!("{}/api/v1", std::env::var("LINKTOR_URL")?))
        .api_key(std::env::var("LINKTOR_API_KEY")?)
        .build()?;

    for canal in linktor.channels().list(None).await? {
        println!("{} {} {:?}", canal.id, canal.name, canal.connection_status);
    }
    Ok(())
}
```

</TabItem>
</Tabs>

Resposta (`200`):

```json
{
  "success": true,
  "data": [
    {
      "id": "0b6f3c1e-8a55-4d0e-9a5b-2f6f0f8f7c11",
      "tenant_id": "5f2a9c0e-1111-4c2b-8d3e-9a7b6c5d4e3f",
      "type": "whatsapp_official",
      "name": "WhatsApp Vendas",
      "identifier": "5511999999999",
      "enabled": true,
      "connection_status": "connected",
      "environment": "production",
      "config": { "phone_number_id": "123456789012345", "verify_token": "__redacted__" },
      "created_at": "2026-11-20T12:00:00Z",
      "updated_at": "2026-11-24T09:15:42Z"
    }
  ]
}
```

Guarde o `id` do canal: é o `CHANNEL_ID` usado em [Enviar mensagens](./enviar-mensagens.md).

:::warning SDKs: passe a base completa e confira a cobertura
- Nenhum SDK acrescenta `/api/v1` sozinho e o endereço padrão deles é `https://api.linktor.io`. Passe
  sempre a base completa, como nos exemplos acima.
- As abas Python, C# e PHP usam o cliente HTTP nativo de propósito: nesta versão o SDK Python rejeita
  canais de tipos que o seu enum não conhece (`whatsapp_official`, `teams`, `slack`…), o SDK PHP perde
  o caminho `/api/v1` da base ao montar a URL e o SDK .NET não compila. O estado de cada SDK está em
  [SDKs](../sdks/visao-geral.md).
:::

## Formato das respostas {#envelope}

Toda resposta de sucesso vem dentro do mesmo envelope:

```json
{
  "success": true,
  "data": { },
  "meta": { }
}
```

| Campo | Quando aparece | Conteúdo |
|---|---|---|
| `success` | sempre | `true` em sucesso, `false` em erro |
| `data` | sucesso | o recurso, uma lista ou um objeto de confirmação |
| `meta` | listas paginadas | `page`, `page_size`, `total_pages`, `total_items`, `has_next`, `has_previous`, `next_cursor` |
| `error` | erro | `code`, `message` e, às vezes, `details` |

Erro gerado por uma rota:

```json
{
  "success": false,
  "error": {
    "code": "VALIDATION_ERROR",
    "message": "to is required"
  }
}
```

Códigos HTTP usados: `200` (ok), `201` (criado), `202` (aceito e enfileirado — envio de mensagem),
`204` (sem corpo — algumas exclusões), `400`, `401`, `403`, `404`, `409`, `429`, `500`, `503`.

:::note Rotas fora do envelope
Algumas rotas de `/bots`, `/flows` e `/knowledge-bases` respondem em formato próprio — as listas
devolvem `{ "data": [...], "pagination": {...} }` sem `success`, e parte das rotas de fluxo devolve o
objeto direto. As exceções estão indicadas em [Bots e IA](./bots-e-ia.md). Em um cliente genérico, não
dependa de `success`: decida pelo status HTTP.
:::

:::warning Erros de autenticação e de limite de requisições não usam o envelope
As respostas `401`, `403` e `429` geradas **antes** de a requisição chegar à rota (chave inválida, scope
ausente, papel insuficiente, limite estourado) trazem `code` e `message` na raiz do JSON, sem `success`
nem `error`. Trate os dois formatos — veja [Erros e paginação](./erros-e-paginacao.md).
:::

## Convenções

- **Isolamento por tenant.** A chave de API ou o usuário autenticado define o tenant. Nenhuma rota
  recebe `tenant_id`; um recurso de outro tenant responde `404`, como se não existisse.
- **`snake_case` em tudo.** `channel_id`, `content_type`, `connection_status`, `created_at`.
- **`config` e `credentials` são mapas texto → texto.** Valores numéricos e booleanos viajam como texto
  (`"true"`, `"587"`).
- **Segredos nunca voltam.** `credentials` não é serializado; valores sensíveis dentro de `config`
  voltam como `__redacted__`.
- **Envio é assíncrono.** A API confirma que a mensagem foi gravada e enfileirada; o resultado da
  entrega chega pelo [webhook](./receber-eventos.md).
- **`X-Request-ID`.** Toda resposta traz esse cabeçalho. Se você enviar o seu, ele é mantido; senão o
  servidor gera um UUID. Informe-o ao abrir um chamado.

## Mapa dos recursos

| Recurso | Rotas | Scope da chave | Página |
|---|---|---|---|
| Envio direto | `POST /messages/send` | `messages:send` | [Enviar mensagens](./enviar-mensagens.md) |
| Mensagens da conversa | `/conversations/:id/messages`, `/attachments`, `/reactions` | `messages:send` + `conversations:write` | [Enviar mensagens](./enviar-mensagens.md) |
| Envio para grupo | `POST /channels/:id/groups/:groupId/messages` | `messages:send` | [Enviar mensagens](./enviar-mensagens.md) |
| Conversas | `/conversations` | `conversations:read` / `conversations:write` | [Conversas](./conversas.md) |
| Contatos | `/contacts` | `contacts:read` / `contacts:write` | [Contatos](./contatos.md) |
| Canais | `/channels` | `channels:read` / `channels:write` | [Canais](./canais.md) |
| Mídia | `/media/*`, `/media/sign`, `/media/revoke` | `conversations:read` / `conversations:write` | [Mídia](./midia.md) |
| Modelos de mensagem | `/templates` | `templates:read` / `templates:write` | [Modelos e campanhas](./modelos-e-campanhas.md) |
| Campanhas | `/campaigns` | sem scope; criar e iniciar exigem usuário com permissão (chave de API só lê) | [Modelos e campanhas](./modelos-e-campanhas.md) |
| Bots | `/bots` | `bots:read` / `bots:write` | [Bots e IA](./bots-e-ia.md) |
| IA | `/ai/*` | `ai:use` | [Bots e IA](./bots-e-ia.md) |
| Bases de conhecimento | `/knowledge-bases` | `knowledge:read` / `knowledge:write` | [Bots e IA](./bots-e-ia.md) |
| Fluxos | `/flows` | `flows:read` / `flows:write` | [Bots e IA](./bots-e-ia.md) |
| Tempo real | `GET /ws` (WebSocket) | — | [Tempo real](./tempo-real.md) |
| Eventos para a sua aplicação | webhook de saída | — | [Receber eventos](./receber-eventos.md) |
| Sessão de usuário | `/auth/login`, `/auth/refresh`, `/auth/logout`, `/me` | — | [Autenticação](./autenticacao.md) |
| Chaves de API | `/api-keys` | só usuário `admin`/`owner` | [Autenticação](./autenticacao.md) |

Outros grupos existem na API e são usados pelo painel — `/users`, `/roles`, `/settings`, `/tenant`,
`/analytics`, `/orders`, `/canned-responses`, `/audit-logs`, `/observability`, `/sandbox/allowlist`,
`/whatsapp/flows`, `/vre` e as sub-rotas de canal `/channels/:id/analytics`, `/payments`, `/calls` e
`/ctwa`. Eles usam a mesma autenticação; `/vre/*` e `/analytics/*` respondem em formato próprio, fora
do envelope padrão. Vários exigem papel `admin` ou `owner`
e, por isso, não são alcançados por chave de API.

As rotas `/api/v1/webhooks/<tipo>/:channelId` são a **entrada** de eventos dos provedores (Meta,
Telegram, Twilio…) e não fazem parte da API de integração: quem as chama é o provedor. Cada página em
[Canais](../canais/visao-geral.md) informa a URL exata.

## Swagger

O servidor publica uma interface Swagger em `/swagger/index.html`, gerada a partir das anotações do
código:

```
https://linktor.suaempresa.com/swagger/index.html
```

:::note
A especificação Swagger é gerada por anotação e pode estar atrás do código: ela declara apenas o
esquema `BearerAuth` (JWT) e nem toda rota está anotada. Use-a para explorar; em caso de divergência,
vale o que está nestas páginas, conferido contra as rotas reais.
:::

## Saúde do servidor

Três rotas sem autenticação, fora de `/api/v1`:

| Rota | Uso | Resposta |
|---|---|---|
| `GET /health` | o processo está de pé | sempre `200` |
| `GET /ready` | as dependências respondem (PostgreSQL, Redis e NATS, quando habilitado) | `200` ou `503` |
| `GET /metrics` | métricas no formato Prometheus | texto |

```bash
curl "$LINKTOR_URL/health"
```

```json
{ "status": "ok", "service": "linktor", "version": "1.4.0", "timestamp": "2026-11-24T14:03:11Z" }
```

```bash
curl "$LINKTOR_URL/ready"
```

```json
{
  "status": "ready",
  "service": "linktor",
  "timestamp": "2026-11-24T14:03:11Z",
  "checks": {
    "postgres": { "status": "healthy" },
    "redis": { "status": "healthy" },
    "nats": { "status": "healthy" }
  }
}
```

Com alguma dependência fora do ar, `/ready` responde `503`, `status` vira `"not ready"` e o item com
problema traz `"status": "unhealthy"` e o campo `error`. O campo `version` de `/health` reflete a
variável `LINKTOR_VERSION` do servidor e fica vazio se ela não estiver definida.

## Próximos passos

- [Autenticação](./autenticacao.md) — criar a chave, escolher scopes, configurar o cliente.
- [Enviar mensagens](./enviar-mensagens.md) — o envio direto e o envio dentro de uma conversa.
- [Receber eventos](./receber-eventos.md) — o webhook que avisa a sua aplicação.
- [SDKs](../sdks/visao-geral.md) — bibliotecas oficiais em sete linguagens.
