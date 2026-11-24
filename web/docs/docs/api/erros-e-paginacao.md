---
sidebar_position: 12
title: Erros e paginação
description: Formatos de erro, códigos e status HTTP, paginação por página e por cursor, idempotência e como repetir chamadas com segurança.
---

# Erros e paginação

## Formato de erro

A API tem dois formatos de erro, e um cliente robusto trata os dois.

**1. Erro de rota** — a requisição foi autenticada e chegou ao recurso. Vem no envelope padrão:

```json
{
  "success": false,
  "error": {
    "code": "VALIDATION_ERROR",
    "message": "metadata cannot overwrite Linktor internal fields",
    "details": { "reserved_keys": "campaign_id" }
  }
}
```

**2. Erro de acesso** — autenticação, scope, papel ou limite de requisições barraram a chamada antes da
rota. Vem sem envelope, com os campos na raiz:

```json
{
  "code": "UNAUTHORIZED",
  "message": "invalid API key",
  "details": null
}
```

| Campo | Descrição |
|---|---|
| `code` | Código estável, em maiúsculas. Programe em cima dele |
| `message` | Texto para leitura humana (em inglês na maior parte das rotas). Pode mudar entre versões |
| `details` | Mapa texto → texto com contexto adicional. Opcional |
| `retry_after` | Só no `429`: segundos até poder repetir |

A leitura que cobre os dois formatos:

```javascript
const erro = corpo.error ?? corpo;   // { code, message, details? }
```

:::note Terceiro formato, em poucas rotas
Algumas rotas respondem falhas de infraestrutura com `{"error": "texto"}`, sem `code`: o upload de
anexo (`413`, `500`, `503`), `POST /media/revoke` (`404`, `500`, `503`), o envio para grupo (`503`) e a
emissão de token do widget de webchat. Se `code` estiver ausente, use o status HTTP e trate `error`
como a mensagem.
:::

## Códigos de erro

| `code` | HTTP | Significado | Repetir? |
|---|---|---|---|
| `VALIDATION_ERROR` | 400 | Corpo inválido, campo obrigatório ausente, valor não aceito, cursor inválido | Não — corrija a requisição |
| `BAD_REQUEST` | 400 | Requisição malformada | Não |
| `UNAUTHORIZED` | 401 | Sem credencial, chave de API inválida, revogada ou vencida | Não |
| `INVALID_CREDENTIALS` | 401 | E-mail ou senha errados no login; senha atual errada | Não |
| `TOKEN_INVALID` | 401 | JWT inválido ou vencido; refresh token inválido | Renove o token e repita uma vez |
| `TOKEN_EXPIRED` | 401 | JWT vencido | Renove o token e repita uma vez |
| `FORBIDDEN` | 403 | Scope ausente na chave ou papel insuficiente | Não |
| `NOT_FOUND` | 404 | Recurso inexistente ou de outro tenant | Não |
| `CHANNEL_NOT_FOUND`, `CONTACT_NOT_FOUND`, `CONVERSATION_NOT_FOUND`, `MESSAGE_NOT_FOUND`, `USER_NOT_FOUND`, `TENANT_NOT_FOUND` | 404 | O mesmo, com o tipo do recurso | Não |
| `CONFLICT` | 409 | Estado conflitante: registro duplicado, ou envio com a mesma `idempotency_key` ainda em processamento | Sim, no caso da `idempotency_key`: espere e repita |
| `RATE_LIMITED` | 429 | Limite de requisições estourado | Sim, depois de `retry_after` segundos |
| `QUOTA_EXCEEDED` | 402 | Cota do plano do tenant atingida | Não |
| `NOT_IMPLEMENTED` | 501 | Operação não disponível nesta versão ou neste canal | Não |
| `BAD_GATEWAY` | 502 | O provedor externo (Meta, Telegram…) falhou ou recusou | Sim, com espera |
| `INTERNAL_ERROR` | 500 | Falha inesperada no servidor | Sim, com espera |
| `CHANNEL_DISCONNECTED` | 500 | Canal desabilitado ou desconectado | Não — reconecte o canal |
| `CHANNEL_ERROR` | 500 | O adaptador do canal devolveu erro | Depende da mensagem |
| `TIMEOUT` | 500 | Operação demorou demais (ex.: QR code não gerado a tempo) | Sim |

:::warning Três códigos respondem 500 sem serem falha do servidor
`CHANNEL_DISCONNECTED`, `CHANNEL_ERROR` e `TIMEOUT` não têm status HTTP próprio nesta versão e saem
como `500`. Decida pelo `code`, não pelo status: repetir um `CHANNEL_DISCONNECTED` às cegas só consome
o limite de requisições.
:::

Outros status que aparecem:

| HTTP | Quando |
|---|---|
| `202` | Envio aceito e enfileirado. Não é erro, mas também não é entrega — veja [Enviar mensagens](./enviar-mensagens.md) |
| `204` | Exclusão concluída, sem corpo. Não tente ler JSON |
| `413` | Arquivo acima de 25 MB no upload de anexo |
| `503` | Dependência indisponível (armazenamento de anexos, fila de saída) ou `/ready` com falha |

### Mensagens de validação

O `message` de um `VALIDATION_ERROR` diz qual campo falhou (`to is required`,
`attachments are required for content_type "image"`). Quando a rota não consegue sequer ler o corpo —
JSON malformado, tipo errado, campo obrigatório ausente em rotas com validação declarativa —, o texto é
genérico: `Invalid request body`. Nesse caso confira o JSON, os tipos e os campos obrigatórios da rota.

### `X-Request-ID`

Toda resposta traz o cabeçalho `X-Request-ID`. Envie o seu próprio valor para correlacionar a chamada
com os seus logs; se não enviar, o servidor gera um. Registre-o junto com o erro: é a chave para achar
a requisição nos logs do servidor.

## Paginação

Existem dois modelos, conforme o recurso.

| Modelo | Parâmetros | Recursos |
|---|---|---|
| Por página, com `meta` | `page`, `page_size` | contatos, modelos, campanhas e seus destinatários, usuários |
| Por página, com `pagination` | `page`, `page_size`, `sort_by`, `sort_dir` | bots, fluxos, bases de conhecimento e seus itens |
| Por cursor | `limit`, `before` | lista de conversas, histórico de mensagens da conversa |
| Sem paginação | — | canais (`GET /channels` devolve a lista inteira) |

### Por página

```bash
curl "$LINKTOR_URL/api/v1/contacts?page=2&page_size=50" -H "X-API-Key: $LINKTOR_API_KEY"
```

| Parâmetro | Padrão | Descrição |
|---|---|---|
| `page` | `1` | Número da página, começando em 1 |
| `page_size` | `20` | Itens por página. Peça no máximo `100` |

```json
{
  "success": true,
  "data": [ ],
  "meta": {
    "page": 2,
    "page_size": 50,
    "total_pages": 7,
    "total_items": 312,
    "has_next": true,
    "has_previous": true
  }
}
```

Continue enquanto `meta.has_next` for `true`.

:::note
- Em contatos, um `page_size` acima de 100 é reduzido a 100 na consulta, mas o `meta` é calculado com o
  valor pedido. Fique em 100 ou menos para que `total_pages` e `has_next` correspondam ao que foi
  devolvido.
- Em modelos (`GET /templates`), o `meta` traz apenas `page`, `page_size` e `total_items`;
  `total_pages` e `has_next` vêm zerados. Calcule o fim por `page * page_size >= total_items`.
:::

#### Listas com `pagination` (bots, fluxos, bases de conhecimento)

Estas listas usam os mesmos parâmetros `page` e `page_size` (padrão `20`), mais `sort_by` (padrão
`created_at`) e `sort_dir` (`asc` ou `desc`, padrão `desc`), mas respondem em um formato próprio — sem
`success` e com `pagination` no lugar de `meta`:

```json
{
  "data": [ ],
  "pagination": {
    "total": 42,
    "page": 1,
    "page_size": 20,
    "pages": 3
  }
}
```

Continue enquanto `pagination.page` for menor que `pagination.pages`. Os detalhes de cada rota estão em
[Bots e IA](./bots-e-ia.md).

### Por cursor {#cursor}

A lista de conversas e o histórico de mensagens crescem o tempo todo; paginar por número de página
faria itens pularem ou se repetirem quando chega mensagem nova. Por isso usam cursor.

| Parâmetro | Padrão | Máximo | Descrição |
|---|---|---|---|
| `limit` | `30` | `100` | Itens por página |
| `before` | — | — | Cursor devolvido em `meta.next_cursor` da página anterior |

Ordem dos resultados:

- `GET /conversations` — da atividade mais recente para a mais antiga (data da última mensagem; sem
  mensagem, data de criação).
- `GET /conversations/:id/messages` — da mensagem mais nova para a mais antiga.

Primeira página:

```bash
curl "$LINKTOR_URL/api/v1/conversations/$CONVERSATION_ID/messages?limit=50" \
  -H "X-API-Key: $LINKTOR_API_KEY"
```

```json
{
  "success": true,
  "data": [ ],
  "meta": {
    "page": 0,
    "page_size": 50,
    "total_pages": 0,
    "total_items": 0,
    "has_next": true,
    "has_previous": false,
    "next_cursor": "MjAyNi0xMS0yNFQxNDowMzoxMS40MTJafDlhOGI3YzZkLTVlNGYtNGEzYi05YzJkLTFlMGY5YThiN2M2ZA"
  }
}
```

Página seguinte — passe o cursor em `before`:

```bash
curl "$LINKTOR_URL/api/v1/conversations/$CONVERSATION_ID/messages?limit=50&before=$NEXT_CURSOR" \
  -H "X-API-Key: $LINKTOR_API_KEY"
```

Regras:

- **O cursor é opaco.** Devolva exatamente o que recebeu; não monte nem interprete o valor. O formato
  interno pode mudar.
- **Pare quando `next_cursor` não vier.** A ausência do campo (ou `has_next: false`) significa que não
  há mais nada. Não descubra o fim por tentativa.
- **Cursor inválido é `400`** (`VALIDATION_ERROR`, `invalid cursor`), não uma página vazia.
- Em `meta`, `page` e `total_pages` vêm sempre `0` neste modelo, e `page_size` é a quantidade de itens
  devolvida. `total_items` só é preenchido na **primeira** página da lista de conversas; no histórico
  de mensagens é sempre `0`.
- Mantenha os mesmos filtros (`status`, `channel_id`, `search`…) em todas as páginas de uma mesma
  varredura.

Laço completo:

<Tabs groupId="linguagem">
<TabItem value="curl" label="cURL">

```bash
CURSOR=""
while :; do
  URL="$LINKTOR_URL/api/v1/conversations?limit=100&status=open"
  [ -n "$CURSOR" ] && URL="$URL&before=$CURSOR"

  PAGINA=$(curl -s "$URL" -H "X-API-Key: $LINKTOR_API_KEY")
  echo "$PAGINA" | jq -r '.data[].id'

  CURSOR=$(echo "$PAGINA" | jq -r '.meta.next_cursor // empty')
  [ -z "$CURSOR" ] && break
done
```

</TabItem>
<TabItem value="typescript" label="TypeScript">

```typescript
async function* conversasAbertas() {
  let cursor: string | undefined;
  do {
    const url = new URL(`${process.env.LINKTOR_URL}/api/v1/conversations`);
    url.searchParams.set('limit', '100');
    url.searchParams.set('status', 'open');
    if (cursor) url.searchParams.set('before', cursor);

    const res = await fetch(url, { headers: { 'X-API-Key': process.env.LINKTOR_API_KEY! } });
    const body = await res.json();
    if (!res.ok) throw new Error(`${res.status} ${JSON.stringify(body)}`);

    yield* body.data as Array<{ id: string }>;
    cursor = body.meta?.next_cursor;
  } while (cursor);
}

for await (const conversa of conversasAbertas()) {
  console.log(conversa.id);
}
```

</TabItem>
<TabItem value="python" label="Python">

```python
import os
import requests


def conversas_abertas():
    cursor = None
    while True:
        params = {"limit": 100, "status": "open"}
        if cursor:
            params["before"] = cursor

        res = requests.get(
            f"{os.environ['LINKTOR_URL']}/api/v1/conversations",
            headers={"X-API-Key": os.environ["LINKTOR_API_KEY"]},
            params=params,
            timeout=30,
        )
        res.raise_for_status()
        body = res.json()

        yield from body["data"]

        cursor = (body.get("meta") or {}).get("next_cursor")
        if not cursor:
            break


for conversa in conversas_abertas():
    print(conversa["id"])
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

type page struct {
	Data []struct {
		ID string `json:"id"`
	} `json:"data"`
	Meta struct {
		NextCursor string `json:"next_cursor"`
	} `json:"meta"`
}

func main() {
	cursor := ""
	for {
		q := url.Values{"limit": {"100"}, "status": {"open"}}
		if cursor != "" {
			q.Set("before", cursor)
		}
		req, _ := http.NewRequest(http.MethodGet,
			os.Getenv("LINKTOR_URL")+"/api/v1/conversations?"+q.Encode(), nil)
		req.Header.Set("X-API-Key", os.Getenv("LINKTOR_API_KEY"))

		res, err := http.DefaultClient.Do(req)
		if err != nil {
			panic(err)
		}
		var p page
		err = json.NewDecoder(res.Body).Decode(&p)
		res.Body.Close()
		if err != nil || res.StatusCode != http.StatusOK {
			panic(fmt.Sprintf("status %d: %v", res.StatusCode, err))
		}

		for _, c := range p.Data {
			fmt.Println(c.ID)
		}
		if p.Meta.NextCursor == "" {
			break
		}
		cursor = p.Meta.NextCursor
	}
}
```

</TabItem>
</Tabs>

Filtros e campos de cada lista estão em [Conversas](./conversas.md) e [Contatos](./contatos.md).

## Idempotência

Repetir uma chamada só é seguro quando o efeito não se duplica.

| Operação | Repetir é seguro? |
|---|---|
| `GET` de qualquer recurso | Sim |
| `PUT` de atualização | Sim — o resultado final é o mesmo |
| `DELETE` | Sim — a segunda chamada pode responder `404` |
| `POST /messages/send` **com** `metadata.idempotency_key` | Sim — a repetição devolve a mensagem original |
| `POST /messages/send` sem `idempotency_key` | **Não** — envia de novo |
| `POST /conversations/:id/messages` | **Não** — envia de novo |
| `POST /channels/:id/groups/:groupId/messages` | **Não** — envia de novo |
| `POST` de criação (contato, canal, modelo…) | Não — pode criar duplicata ou responder `409` |

Para envio de mensagem disparado por um evento do seu sistema, use sempre o envio direto com
`idempotency_key` derivada do evento (`pedido-1234-confirmado`), nunca um valor aleatório gerado a cada
tentativa. As regras completas — incluindo o `409` de envio em andamento — estão em
[Enviar mensagens](./enviar-mensagens.md).

Do lado do recebimento, o webhook pode entregar o mesmo evento mais de uma vez; a deduplicação pelo
`id` do evento está descrita em [Receber eventos](./receber-eventos.md).

## Repetir chamadas com segurança

O que repetir:

| Situação | Ação |
|---|---|
| Erro de rede, timeout de conexão | Repita com espera crescente — só se a operação for idempotente |
| `429` | Espere `retry_after` segundos (ou até `X-RateLimit-Reset`) e repita |
| `409` com `idempotency_key` em processamento | Espere 1 a 2 segundos e repita a mesma chamada |
| `502`, `503`, `500` com `INTERNAL_ERROR` ou `TIMEOUT` | Repita com espera crescente |
| `401` com `TOKEN_INVALID` (sessão de usuário) | Renove em `/auth/refresh` e repita uma vez |
| `400`, `403`, `404`, `401` de chave de API, `500` com `CHANNEL_DISCONNECTED` | **Não repita** |

Como repetir:

- **Espera exponencial com variação aleatória**: 1 s, 2 s, 4 s, 8 s…, somando um valor aleatório para
  que vários processos não repitam no mesmo instante.
- **Limite de tentativas**: 4 ou 5. Depois disso, registre o erro e trate fora do fluxo.
- **Respeite o limite do tenant**: são 100 requisições por minuto somando todas as chaves. Um laço de
  repetição sem espera esgota o limite de todos os sistemas integrados.
- **Defina timeout no cliente**: 30 segundos cobre as chamadas normais. `POST /channels/:id/connect`
  pode levar até 30 segundos aguardando o QR code — use um timeout maior nela.

Exemplo de chamada com repetição, usando o envio direto (idempotente por causa da chave):

<Tabs groupId="linguagem">
<TabItem value="curl" label="cURL">

```bash
# --retry repete em 408, 429, 500, 502, 503 e 504, e respeita o cabeçalho Retry-After quando existe.
# O Linktor informa a espera no corpo (retry_after), então mantenha um --retry-delay conservador.
curl --retry 4 --retry-delay 5 --max-time 30 \
  -X POST "$LINKTOR_URL/api/v1/messages/send" \
  -H "X-API-Key: $LINKTOR_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "channel_id": "CHANNEL_ID",
    "to": "+5511999999999",
    "text": "Pedido 1234 confirmado.",
    "metadata": { "idempotency_key": "pedido-1234-confirmado" }
  }'
```

</TabItem>
<TabItem value="typescript" label="TypeScript">

```typescript
const NAO_REPETIR = new Set(['CHANNEL_DISCONNECTED', 'CHANNEL_ERROR']);
const dormir = (ms: number) => new Promise((r) => setTimeout(r, ms));

async function comRepeticao(path: string, body: unknown, tentativas = 5) {
  for (let n = 0; ; n++) {
    let res: Response | undefined;
    try {
      res = await fetch(`${process.env.LINKTOR_URL}/api/v1${path}`, {
        method: 'POST',
        headers: { 'X-API-Key': process.env.LINKTOR_API_KEY!, 'Content-Type': 'application/json' },
        body: JSON.stringify(body),
        signal: AbortSignal.timeout(30_000),
      });
    } catch (e) {
      if (n + 1 >= tentativas) throw e; // rede ou timeout
    }

    if (res) {
      const json = await res.json().catch(() => ({}));
      if (res.ok) return json.data;

      const err = json.error ?? json;
      const repetivel =
        res.status === 429 || res.status === 409 || (res.status >= 500 && !NAO_REPETIR.has(err.code));
      if (!repetivel || n + 1 >= tentativas) {
        throw new Error(`${res.status} ${err.code}: ${err.message}`);
      }
      if (res.status === 429 && json.retry_after) {
        await dormir(json.retry_after * 1000);
        continue;
      }
    }
    await dormir(2 ** n * 1000 + Math.random() * 500);
  }
}

await comRepeticao('/messages/send', {
  channel_id: 'CHANNEL_ID',
  to: '+5511999999999',
  text: 'Pedido 1234 confirmado.',
  metadata: { idempotency_key: 'pedido-1234-confirmado' },
});
```

</TabItem>
<TabItem value="python" label="Python">

```python
import os
import random
import time

import requests

NAO_REPETIR = {"CHANNEL_DISCONNECTED", "CHANNEL_ERROR"}


def com_repeticao(path: str, body: dict, tentativas: int = 5):
    for n in range(tentativas):
        ultima = n + 1 >= tentativas
        try:
            res = requests.post(
                f"{os.environ['LINKTOR_URL']}/api/v1{path}",
                headers={"X-API-Key": os.environ["LINKTOR_API_KEY"]},
                json=body,
                timeout=30,
            )
        except requests.RequestException:
            if ultima:
                raise
            time.sleep(2**n + random.random() / 2)
            continue

        dados = res.json() if res.content else {}
        if res.ok:
            return dados.get("data")

        err = dados.get("error") or dados
        repetivel = res.status_code in (409, 429) or (
            res.status_code >= 500 and err.get("code") not in NAO_REPETIR
        )
        if not repetivel or ultima:
            raise RuntimeError(f"{res.status_code} {err.get('code')}: {err.get('message')}")

        if res.status_code == 429 and dados.get("retry_after"):
            time.sleep(dados["retry_after"])
        else:
            time.sleep(2**n + random.random() / 2)


com_repeticao("/messages/send", {
    "channel_id": "CHANNEL_ID",
    "to": "+5511999999999",
    "text": "Pedido 1234 confirmado.",
    "metadata": {"idempotency_key": "pedido-1234-confirmado"},
})
```

</TabItem>
<TabItem value="go" label="Go">

```go
package linktorhttp

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"os"
	"time"
)

var client = &http.Client{Timeout: 30 * time.Second}

type failure struct {
	Error *struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
	Code       string `json:"code"`
	Message    string `json:"message"`
	RetryAfter int    `json:"retry_after"`
}

// PostComRepeticao devolve o corpo da resposta de sucesso.
func PostComRepeticao(path string, body any, tentativas int) ([]byte, error) {
	payload, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}

	var ultimo error
	for n := 0; n < tentativas; n++ {
		espera := time.Duration(1<<n)*time.Second + time.Duration(rand.Intn(500))*time.Millisecond

		req, _ := http.NewRequest(http.MethodPost,
			os.Getenv("LINKTOR_URL")+"/api/v1"+path, bytes.NewReader(payload))
		req.Header.Set("X-API-Key", os.Getenv("LINKTOR_API_KEY"))
		req.Header.Set("Content-Type", "application/json")

		res, err := client.Do(req)
		if err != nil { // rede ou timeout
			ultimo = err
			time.Sleep(espera)
			continue
		}
		raw, _ := io.ReadAll(res.Body)
		res.Body.Close()

		if res.StatusCode < 400 {
			return raw, nil
		}

		var f failure
		_ = json.Unmarshal(raw, &f)
		code, msg := f.Code, f.Message
		if f.Error != nil {
			code, msg = f.Error.Code, f.Error.Message
		}
		ultimo = fmt.Errorf("%d %s: %s", res.StatusCode, code, msg)

		repetivel := res.StatusCode == 429 || res.StatusCode == 409 ||
			(res.StatusCode >= 500 && code != "CHANNEL_DISCONNECTED" && code != "CHANNEL_ERROR")
		if !repetivel {
			return nil, ultimo
		}
		if res.StatusCode == 429 && f.RetryAfter > 0 {
			espera = time.Duration(f.RetryAfter) * time.Second
		}
		time.Sleep(espera)
	}
	return nil, ultimo
}
```

</TabItem>
<TabItem value="java" label="Java">

```java
import java.io.IOException;
import java.net.URI;
import java.net.http.HttpClient;
import java.net.http.HttpRequest;
import java.net.http.HttpResponse;
import java.time.Duration;
import java.util.concurrent.ThreadLocalRandom;

public class LinktorRetry {
    static final HttpClient HTTP = HttpClient.newHttpClient();

    /** Devolve o corpo da resposta de sucesso. */
    static String postComRepeticao(String path, String json, int tentativas) throws Exception {
        Exception ultimo = null;
        for (int n = 0; n < tentativas; n++) {
            long espera = (1L << n) * 1000 + ThreadLocalRandom.current().nextInt(500);
            try {
                HttpRequest request = HttpRequest.newBuilder(
                        URI.create(System.getenv("LINKTOR_URL") + "/api/v1" + path))
                    .timeout(Duration.ofSeconds(30))
                    .header("X-API-Key", System.getenv("LINKTOR_API_KEY"))
                    .header("Content-Type", "application/json")
                    .POST(HttpRequest.BodyPublishers.ofString(json))
                    .build();

                HttpResponse<String> response = HTTP.send(request, HttpResponse.BodyHandlers.ofString());
                int status = response.statusCode();
                String body = response.body();
                if (status < 400) {
                    return body;
                }

                ultimo = new RuntimeException(status + " " + body);
                boolean canalFora = body.contains("\"CHANNEL_DISCONNECTED\"") || body.contains("\"CHANNEL_ERROR\"");
                boolean repetivel = status == 429 || status == 409 || (status >= 500 && !canalFora);
                if (!repetivel) {
                    throw ultimo;
                }
                if (status == 429) {
                    espera = 60_000; // a janela do limite é de um minuto
                }
            } catch (IOException e) { // rede ou timeout
                ultimo = e;
            }
            Thread.sleep(espera);
        }
        throw ultimo;
    }
}
```

</TabItem>
<TabItem value="csharp" label="C#">

```csharp
using System.Net.Http.Json;
using System.Text.Json;

static class LinktorRetry
{
    static readonly HttpClient Http = new() { Timeout = TimeSpan.FromSeconds(30) };
    static readonly string[] NaoRepetir = { "CHANNEL_DISCONNECTED", "CHANNEL_ERROR" };

    public static async Task<JsonElement> PostComRepeticao(string path, object body, int tentativas = 5)
    {
        Exception? ultimo = null;
        for (var n = 0; n < tentativas; n++)
        {
            var espera = TimeSpan.FromMilliseconds(Math.Pow(2, n) * 1000 + Random.Shared.Next(500));
            try
            {
                using var request = new HttpRequestMessage(HttpMethod.Post,
                    Environment.GetEnvironmentVariable("LINKTOR_URL") + "/api/v1" + path)
                {
                    Content = JsonContent.Create(body),
                };
                request.Headers.Add("X-API-Key", Environment.GetEnvironmentVariable("LINKTOR_API_KEY"));

                using var response = await Http.SendAsync(request);
                var json = await response.Content.ReadFromJsonAsync<JsonElement>();
                if (response.IsSuccessStatusCode)
                {
                    return json.GetProperty("data");
                }

                var err = json.TryGetProperty("error", out var e) ? e : json;
                var code = err.TryGetProperty("code", out var c) ? c.GetString() : null;
                var status = (int)response.StatusCode;
                ultimo = new Exception($"{status} {code}");

                var repetivel = status == 429 || status == 409 || (status >= 500 && !NaoRepetir.Contains(code));
                if (!repetivel) throw ultimo;

                if (status == 429 && json.TryGetProperty("retry_after", out var ra))
                {
                    espera = TimeSpan.FromSeconds(ra.GetInt32());
                }
            }
            catch (Exception ex) when (ex is HttpRequestException or TaskCanceledException)
            {
                ultimo = ex; // rede ou timeout
            }
            await Task.Delay(espera);
        }
        throw ultimo!;
    }
}
```

</TabItem>
<TabItem value="php" label="PHP">

```php
<?php
require 'vendor/autoload.php';

use GuzzleHttp\Client;
use GuzzleHttp\Exception\ConnectException;

function postComRepeticao(string $path, array $body, int $tentativas = 5): ?array
{
    $http = new Client(['http_errors' => false, 'timeout' => 30]);
    $naoRepetir = ['CHANNEL_DISCONNECTED', 'CHANNEL_ERROR'];
    $ultimo = null;

    for ($n = 0; $n < $tentativas; $n++) {
        $espera = (2 ** $n) * 1_000_000 + random_int(0, 500_000); // microssegundos
        try {
            $res = $http->post(getenv('LINKTOR_URL') . '/api/v1' . $path, [
                'headers' => ['X-API-Key' => getenv('LINKTOR_API_KEY')],
                'json'    => $body,
            ]);
            $json   = json_decode((string) $res->getBody(), true) ?? [];
            $status = $res->getStatusCode();
            if ($status < 400) {
                return $json['data'] ?? null;
            }

            $err    = $json['error'] ?? $json;
            $code   = $err['code'] ?? '';
            $ultimo = new RuntimeException("{$status} {$code}: " . ($err['message'] ?? ''));

            $repetivel = $status === 429 || $status === 409
                || ($status >= 500 && !in_array($code, $naoRepetir, true));
            if (!$repetivel) {
                throw $ultimo;
            }
            if ($status === 429 && !empty($json['retry_after'])) {
                $espera = $json['retry_after'] * 1_000_000;
            }
        } catch (ConnectException $e) { // rede ou timeout
            $ultimo = $e;
        }
        usleep((int) $espera);
    }
    throw $ultimo;
}
```

</TabItem>
<TabItem value="rust" label="Rust">

```rust
use rand::Rng;
use serde_json::Value;
use std::time::Duration;

pub async fn post_com_repeticao(
    path: &str,
    body: &Value,
    tentativas: u32,
) -> Result<Value, Box<dyn std::error::Error>> {
    let http = reqwest::Client::builder().timeout(Duration::from_secs(30)).build()?;
    let url = format!("{}/api/v1{}", std::env::var("LINKTOR_URL")?, path);
    let key = std::env::var("LINKTOR_API_KEY")?;
    let mut ultimo: Box<dyn std::error::Error> = "nenhuma tentativa".into();

    for n in 0..tentativas {
        let mut espera =
            Duration::from_millis(2u64.pow(n) * 1000 + rand::thread_rng().gen_range(0..500));

        match http.post(&url).header("X-API-Key", &key).json(body).send().await {
            Err(e) => ultimo = e.into(), // rede ou timeout
            Ok(res) => {
                let status = res.status().as_u16();
                let json: Value = res.json().await.unwrap_or(Value::Null);
                if status < 400 {
                    return Ok(json["data"].clone());
                }

                let err = json.get("error").unwrap_or(&json);
                let code = err["code"].as_str().unwrap_or("");
                ultimo = format!("{status} {code}: {}", err["message"]).into();

                let canal_fora = code == "CHANNEL_DISCONNECTED" || code == "CHANNEL_ERROR";
                let repetivel = status == 429 || status == 409 || (status >= 500 && !canal_fora);
                if !repetivel {
                    return Err(ultimo);
                }
                if status == 429 {
                    if let Some(s) = json["retry_after"].as_u64() {
                        espera = Duration::from_secs(s);
                    }
                }
            }
        }
        tokio::time::sleep(espera).await;
    }
    Err(ultimo)
}
```

</TabItem>
</Tabs>

:::warning Só repita `POST` com proteção contra duplicidade
A função acima é segura para `POST /messages/send` **com** `idempotency_key`. Aplicada ao envio na
conversa ou ao envio para grupo, uma falha de rede depois de o servidor já ter aceitado a mensagem
resulta em mensagem duplicada para o cliente.
:::

## Limite de requisições

O limite é de 100 requisições por minuto por tenant. Os cabeçalhos `X-RateLimit-Limit`,
`X-RateLimit-Remaining` e `X-RateLimit-Reset` permitem reduzir o ritmo antes de receber um `429`.
Detalhes em [Autenticação](./autenticacao.md).

Para volume alto de envios, não dispare chamadas em paralelo sem controle: enfileire no seu lado e
envie em ritmo constante, ou use uma [campanha](./modelos-e-campanhas.md) quando o conteúdo for um
modelo do WhatsApp.
