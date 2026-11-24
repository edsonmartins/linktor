---
sidebar_position: 8
title: Mídia
description: Como baixar a mídia recebida, assinar URLs com validade, revogar arquivos e subir anexos para envio.
---

# Mídia

Todo arquivo que passa pelo Linktor — mídia recebida dos canais e anexo enviado pela API — é servido
por um único endereço, o **proxy de mídia**:

```
GET https://linktor.suaempresa.com/api/v1/media/<chave>
```

A `<chave>` é o caminho opaco do objeto no armazenamento. O prefixo indica a origem:

| Prefixo da chave | Conteúdo |
|---|---|
| `inbound/<tipo do canal>/<id do canal>/...` | Mídia recebida de um contato |
| `attachments/<tenant>/...` | Anexo enviado por `POST /conversations/:id/attachments` |
| `vre/<tenant>/...` | Imagem gerada pelo motor de respostas visuais |

Rotas desta página:

| Método | Caminho | Scope | Para quê |
|---|---|---|---|
| `GET` | `/media/*key` | Ver [formas de autorização](#autorizacao) | Baixar o arquivo |
| `POST` | `/media/sign` | `conversations:read` | Emitir URLs assinadas com validade |
| `POST` | `/media/revoke` | `conversations:write` | Bloquear um arquivo ou um prefixo, em definitivo |
| `POST` | `/conversations/:id/attachments` | `conversations:write` e `messages:send` | Subir um anexo para envio |

## Como a leitura é autorizada {#autorizacao}

O comportamento depende de o servidor ter um segredo de assinatura configurado
(`MEDIA_SIGNING_SECRET`; veja [Configuração](../implantar/configuracao.md)).

**Sem segredo configurado**, a URL é uma URL de capacidade: quem tem o endereço lê o arquivo, e ela
nunca expira.

**Com segredo configurado**, um `GET` é aceito quando qualquer uma destas condições vale, nesta
ordem:

1. **URL assinada** — a URL traz `?exp=<unix>&sig=<hex>` válidos. Não precisa de cabeçalho nem
   sessão; funciona em uma tag `<img>` ou em um download direto.
2. **Credencial do tenant dono** — a requisição traz `X-API-Key` (com o scope
   `conversations:read`), `Authorization: Bearer` ou o cookie de sessão de um usuário do tenant que
   é dono da chave.
3. **Janela de transição** — URL sem assinatura e sem credencial só é aceita enquanto a janela
   definida por `MEDIA_UNSIGNED_UNTIL` estiver aberta (sem data definida, fica aberta). A resposta
   vem com o cabeçalho `Deprecation: true`.

Respostas do `GET /media/*key` (sem corpo JSON nos erros):

| Status | Significado | O que fazer |
|---|---|---|
| `200` | Conteúdo do arquivo, com o `Content-Type` original | — |
| `400` | Chave inválida | Confira a URL |
| `401` | URL sem assinatura, sem credencial, e a janela de transição fechou | Assine a URL ou envie `X-API-Key` |
| `403` | Assinatura inválida, ou o arquivo foi revogado | Uma revogação é definitiva; assinatura inválida, peça outra |
| `404` | O objeto não existe no armazenamento | — |
| `410` | A assinatura expirou | O arquivo existe; peça uma nova assinatura em `POST /media/sign` |
| `503` | O servidor não tem armazenamento de mídia configurado | Veja [Configuração](../implantar/configuracao.md) |

## Onde aparecem URLs de mídia

| Origem | A URL vem assinada? |
|---|---|
| Respostas da API de mensagens (`attachments[].url`, `attachments[].thumbnail_url`, `metadata.media_url`) | Sim, quando o servidor tem assinatura configurada |
| Eventos do [WebSocket](tempo-real.md) (`new_message`, `message_updated`) | Sim, quando o servidor tem assinatura configurada |
| Webhook de saída (`data.message.content.media.url`) | **Não** — é a referência durável, sem `exp`/`sig` |
| Resposta do upload de anexo (`data.url`) | Não — referência durável |

A referência durável (sem assinatura) é o que você deve **guardar** no seu banco. A URL assinada é
para **uso imediato**: ela vence. Assine de novo na hora de ler.

## Baixar a mídia recebida {#baixar}

Ao receber um `message.received` com `data.message.content.media.url` (veja
[Receber eventos](receber-eventos.md)), há dois caminhos:

- **Servidor para servidor**: faça o `GET` da própria URL enviando `X-API-Key`. É o caminho mais
  simples e funciona com ou sem assinatura configurada.
- **Entregar a um navegador ou a um terceiro**: chame `POST /media/sign` e repasse a URL assinada,
  que funciona sem credencial até a validade.

<Tabs groupId="linguagem">
<TabItem value="curl" label="cURL">

```bash
MEDIA_URL="https://linktor.suaempresa.com/api/v1/media/inbound/whatsapp_official/CHANNEL_ID/7f3b2a.jpg"

# Caminho 1: baixar direto com a chave de API
curl -fSL "$MEDIA_URL" -H "X-API-Key: $LINKTOR_API_KEY" -o comprovante.jpg

# Caminho 2: assinar e baixar sem credencial
SIGNED=$(curl -s -X POST "$LINKTOR_URL/api/v1/media/sign" \
  -H "X-API-Key: $LINKTOR_API_KEY" \
  -H "Content-Type: application/json" \
  -d "{\"urls\": [\"$MEDIA_URL\"], \"ttl_seconds\": 3600}" | jq -r '.data[0].url')
curl -fSL "$SIGNED" -o comprovante.jpg
```

</TabItem>
<TabItem value="typescript" label="TypeScript">

```typescript
import { writeFile } from 'node:fs/promises';

const base = process.env.LINKTOR_URL!;
const apiKey = process.env.LINKTOR_API_KEY!;
const mediaUrl =
  'https://linktor.suaempresa.com/api/v1/media/inbound/whatsapp_official/CHANNEL_ID/7f3b2a.jpg';

// Caminho 1: baixar direto com a chave de API
const direct = await fetch(mediaUrl, { headers: { 'X-API-Key': apiKey } });
if (!direct.ok) throw new Error(`download falhou: ${direct.status}`);
await writeFile('comprovante.jpg', Buffer.from(await direct.arrayBuffer()));

// Caminho 2: assinar para repassar a um navegador
const res = await fetch(`${base}/api/v1/media/sign`, {
  method: 'POST',
  headers: { 'X-API-Key': apiKey, 'Content-Type': 'application/json' },
  body: JSON.stringify({ urls: [mediaUrl], ttl_seconds: 3600 }),
});
const { data } = await res.json();
if (data[0].error) throw new Error(data[0].error);
console.log(data[0].url, data[0].expires_at);
```

</TabItem>
<TabItem value="python" label="Python">

```python
import os

import requests

base = os.environ["LINKTOR_URL"]
headers = {"X-API-Key": os.environ["LINKTOR_API_KEY"]}
media_url = "https://linktor.suaempresa.com/api/v1/media/inbound/whatsapp_official/CHANNEL_ID/7f3b2a.jpg"

# Caminho 1: baixar direto com a chave de API
direct = requests.get(media_url, headers=headers, timeout=60)
direct.raise_for_status()
with open("comprovante.jpg", "wb") as f:
    f.write(direct.content)

# Caminho 2: assinar para repassar a um navegador
res = requests.post(
    f"{base}/api/v1/media/sign",
    headers=headers,
    json={"urls": [media_url], "ttl_seconds": 3600},
    timeout=30,
)
res.raise_for_status()
item = res.json()["data"][0]
if item.get("error"):
    raise RuntimeError(item["error"])
print(item["url"], item.get("expires_at"))
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
	base := os.Getenv("LINKTOR_URL")
	apiKey := os.Getenv("LINKTOR_API_KEY")
	mediaURL := "https://linktor.suaempresa.com/api/v1/media/inbound/whatsapp_official/CHANNEL_ID/7f3b2a.jpg"

	// Caminho 1: baixar direto com a chave de API
	req, _ := http.NewRequest(http.MethodGet, mediaURL, nil)
	req.Header.Set("X-API-Key", apiKey)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		panic(fmt.Sprintf("download falhou: %d", resp.StatusCode))
	}
	out, _ := os.Create("comprovante.jpg")
	defer out.Close()
	io.Copy(out, resp.Body)

	// Caminho 2: assinar para repassar a um navegador
	body, _ := json.Marshal(map[string]any{"urls": []string{mediaURL}, "ttl_seconds": 3600})
	signReq, _ := http.NewRequest(http.MethodPost, base+"/api/v1/media/sign", bytes.NewReader(body))
	signReq.Header.Set("X-API-Key", apiKey)
	signReq.Header.Set("Content-Type", "application/json")
	signResp, err := http.DefaultClient.Do(signReq)
	if err != nil {
		panic(err)
	}
	defer signResp.Body.Close()

	var envelope struct {
		Data []struct {
			URL       string `json:"url"`
			ExpiresAt string `json:"expires_at"`
			Error     string `json:"error"`
		} `json:"data"`
	}
	json.NewDecoder(signResp.Body).Decode(&envelope)
	if len(envelope.Data) == 0 || envelope.Data[0].Error != "" {
		panic("não foi possível assinar")
	}
	fmt.Println(envelope.Data[0].URL, envelope.Data[0].ExpiresAt)
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
import java.nio.file.Path;

public class BaixarMidia {
    public static void main(String[] args) throws Exception {
        String base = System.getenv("LINKTOR_URL");
        String apiKey = System.getenv("LINKTOR_API_KEY");
        String mediaUrl =
            "https://linktor.suaempresa.com/api/v1/media/inbound/whatsapp_official/CHANNEL_ID/7f3b2a.jpg";
        HttpClient http = HttpClient.newHttpClient();

        // Caminho 1: baixar direto com a chave de API
        HttpRequest download = HttpRequest.newBuilder(URI.create(mediaUrl))
            .header("X-API-Key", apiKey)
            .GET()
            .build();
        HttpResponse<Path> file =
            http.send(download, HttpResponse.BodyHandlers.ofFile(Path.of("comprovante.jpg")));
        if (file.statusCode() != 200) throw new IllegalStateException("download falhou: " + file.statusCode());

        // Caminho 2: assinar para repassar a um navegador
        String json = "{\"urls\":[\"" + mediaUrl + "\"],\"ttl_seconds\":3600}";
        HttpRequest sign = HttpRequest.newBuilder(URI.create(base + "/api/v1/media/sign"))
            .header("X-API-Key", apiKey)
            .header("Content-Type", "application/json")
            .POST(HttpRequest.BodyPublishers.ofString(json))
            .build();
        HttpResponse<String> signed = http.send(sign, HttpResponse.BodyHandlers.ofString());
        JsonNode item = new ObjectMapper().readTree(signed.body()).at("/data/0");
        if (item.hasNonNull("error")) throw new IllegalStateException(item.get("error").asText());
        System.out.println(item.get("url").asText());
    }
}
```

</TabItem>
<TabItem value="csharp" label="C#">

```csharp
using System.Net.Http.Json;
using System.Text.Json;

var baseUrl = Environment.GetEnvironmentVariable("LINKTOR_URL")!;
var mediaUrl =
    "https://linktor.suaempresa.com/api/v1/media/inbound/whatsapp_official/CHANNEL_ID/7f3b2a.jpg";

using var http = new HttpClient();
http.DefaultRequestHeaders.Add("X-API-Key", Environment.GetEnvironmentVariable("LINKTOR_API_KEY"));

// Caminho 1: baixar direto com a chave de API
var bytes = await http.GetByteArrayAsync(mediaUrl);
await File.WriteAllBytesAsync("comprovante.jpg", bytes);

// Caminho 2: assinar para repassar a um navegador
var response = await http.PostAsJsonAsync(
    $"{baseUrl}/api/v1/media/sign",
    new { urls = new[] { mediaUrl }, ttl_seconds = 3600 });
response.EnsureSuccessStatusCode();

using var doc = JsonDocument.Parse(await response.Content.ReadAsStringAsync());
var item = doc.RootElement.GetProperty("data")[0];
if (item.TryGetProperty("error", out var error)) throw new Exception(error.GetString());
Console.WriteLine(item.GetProperty("url").GetString());
```

</TabItem>
<TabItem value="php" label="PHP">

```php
<?php
// composer require guzzlehttp/guzzle
require 'vendor/autoload.php';

$base = getenv('LINKTOR_URL');
$mediaUrl = 'https://linktor.suaempresa.com/api/v1/media/inbound/whatsapp_official/CHANNEL_ID/7f3b2a.jpg';
$http = new GuzzleHttp\Client(['headers' => ['X-API-Key' => getenv('LINKTOR_API_KEY')]]);

// Caminho 1: baixar direto com a chave de API
$http->get($mediaUrl, ['sink' => 'comprovante.jpg']);

// Caminho 2: assinar para repassar a um navegador
$response = $http->post("$base/api/v1/media/sign", [
    'json' => ['urls' => [$mediaUrl], 'ttl_seconds' => 3600],
]);
$item = json_decode((string) $response->getBody(), true)['data'][0];
if (!empty($item['error'])) {
    throw new RuntimeException($item['error']);
}
echo $item['url'], PHP_EOL;
```

</TabItem>
<TabItem value="rust" label="Rust">

```rust
// Cargo.toml: reqwest = { version = "0.12", features = ["json"] },
//             tokio = { version = "1", features = ["full"] }, serde_json = "1"
use serde_json::{json, Value};

#[tokio::main]
async fn main() -> Result<(), Box<dyn std::error::Error>> {
    let base = std::env::var("LINKTOR_URL")?;
    let api_key = std::env::var("LINKTOR_API_KEY")?;
    let media_url =
        "https://linktor.suaempresa.com/api/v1/media/inbound/whatsapp_official/CHANNEL_ID/7f3b2a.jpg";
    let http = reqwest::Client::new();

    // Caminho 1: baixar direto com a chave de API
    let bytes = http
        .get(media_url)
        .header("X-API-Key", &api_key)
        .send()
        .await?
        .error_for_status()?
        .bytes()
        .await?;
    std::fs::write("comprovante.jpg", &bytes)?;

    // Caminho 2: assinar para repassar a um navegador
    let envelope: Value = http
        .post(format!("{base}/api/v1/media/sign"))
        .header("X-API-Key", &api_key)
        .json(&json!({ "urls": [media_url], "ttl_seconds": 3600 }))
        .send()
        .await?
        .error_for_status()?
        .json()
        .await?;
    let item = &envelope["data"][0];
    if let Some(error) = item["error"].as_str() {
        return Err(error.into());
    }
    println!("{}", item["url"]);
    Ok(())
}
```

</TabItem>
</Tabs>

## `POST /media/sign` — assinar URLs {#sign}

Emite URLs com validade para arquivos do seu tenant. Scope: `conversations:read`.

Corpo:

| Campo | Tipo | Descrição |
|---|---|---|
| `keys` | lista de string | Chaves de mídia (`inbound/...`, `attachments/...`). Uma barra inicial é ignorada |
| `urls` | lista de string | URLs do proxy de mídia como foram guardadas; a chave é extraída do caminho |
| `ttl_seconds` | inteiro | Validade pedida, em segundos. Omitido, zero ou acima do teto do servidor: vale o teto |

Envie de 1 a 100 itens somando `keys` e `urls`. O teto de validade é o TTL do servidor: 7 dias por
padrão, ajustável por `MEDIA_URL_TTL`.

```bash
curl -X POST "$LINKTOR_URL/api/v1/media/sign" \
  -H "X-API-Key: $LINKTOR_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "keys": ["attachments/5d0c2a6e-1f7b-4a37-9d0e-6a1b9a3f2c10/2b1c0d9e.pdf"],
    "urls": ["https://linktor.suaempresa.com/api/v1/media/inbound/whatsapp_official/CHANNEL_ID/7f3b2a.jpg"],
    "ttl_seconds": 3600
  }'
```

Resposta `200` — um item por entrada, na mesma ordem (`keys` primeiro, depois `urls`):

```json
{
  "success": true,
  "data": [
    {
      "key": "attachments/5d0c2a6e-1f7b-4a37-9d0e-6a1b9a3f2c10/2b1c0d9e.pdf",
      "source": "attachments/5d0c2a6e-1f7b-4a37-9d0e-6a1b9a3f2c10/2b1c0d9e.pdf",
      "url": "https://linktor.suaempresa.com/api/v1/media/attachments/5d0c2a6e-1f7b-4a37-9d0e-6a1b9a3f2c10/2b1c0d9e.pdf?exp=1795532400&sig=4c1f0e9a7b6d5c4e3f2a1b0c9d8e7f6a5b4c3d2e1f0a9b8c7d6e5f4a3b2c1d0e",
      "expires_at": "2026-11-24T15:00:00Z"
    },
    {
      "source": "https://linktor.suaempresa.com/api/v1/media/inbound/whatsapp_official/OUTRO_CANAL/9a.jpg",
      "error": "not found"
    }
  ]
}
```

A chamada responde `200` mesmo quando itens individuais são recusados: confira `error` em **cada**
item.

| `error` do item | Significado |
|---|---|
| `not a media key or media URL` | O valor não é uma chave válida nem uma URL do proxy de mídia |
| `not found` | A chave não pertence ao seu tenant (a resposta é a mesma para "de outro tenant" e "inexistente") |
| `revoked` | O arquivo foi revogado |

Erros da chamada inteira: `400` (`VALIDATION_ERROR`) para corpo inválido, nenhum item ou mais de
100; `401`/`403` conforme [Autenticação](autenticacao.md).

:::note Servidor sem assinatura configurada
Sem `MEDIA_SIGNING_SECRET`, a rota continua respondendo: devolve a URL sem assinatura em `url` e
omite `expires_at`. Um integrador que assina na leitura funciona igual nos dois ambientes — trate
`expires_at` ausente como "não expira".
:::

### Formato da URL assinada

```
<base>/api/v1/media/<chave>?exp=<unix em segundos>&sig=<hex>
```

A assinatura é feita com um segredo que só o servidor conhece; o integrador não a calcula nem a
valida. Trocar `MEDIA_SIGNING_SECRET` no servidor invalida todas as URLs assinadas já emitidas.

O navegador pode manter o arquivo em cache: a resposta de uma URL assinada vem com
`Cache-Control: private` e validade limitada ao tempo restante da assinatura, com teto de um dia.

## `POST /media/revoke` — revogar {#revoke}

Bloqueia em definitivo um arquivo, ou tudo sob um prefixo. Depois da revogação o `GET` responde
`403` **inclusive para URLs assinadas antes dela**. Não há rota para desfazer.

- Scope: `conversations:write`.
- Papel: usuário `admin` ou `owner`, ou chave de API.
- A chamada é registrada na trilha de auditoria.

Corpo — exatamente **um** dos dois campos:

| Campo | Descrição |
|---|---|
| `key` | Uma chave de mídia |
| `prefix` | Um diretório inteiro. Só são aceitos `attachments/<tenant>/`, `vre/<tenant>/` e `inbound/<tipo>/<id do canal>/` (ou subdiretórios deles) do seu próprio tenant. A barra final é acrescentada se faltar |

```bash
# Um arquivo
curl -X POST "$LINKTOR_URL/api/v1/media/revoke" \
  -H "X-API-Key: $LINKTOR_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{"key": "inbound/whatsapp_official/CHANNEL_ID/7f3b2a.jpg"}'

# Toda a mídia recebida por um canal
curl -X POST "$LINKTOR_URL/api/v1/media/revoke" \
  -H "X-API-Key: $LINKTOR_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{"prefix": "inbound/whatsapp_official/CHANNEL_ID/"}'
```

Resposta `200`:

```json
{
  "success": true,
  "data": { "revoked": "inbound/whatsapp_official/CHANNEL_ID/" }
}
```

| Status | Quando |
|---|---|
| `400` | Nenhum ou os dois campos informados (`VALIDATION_ERROR`) |
| `403` | Chave sem o scope, ou usuário sem papel `admin`/`owner` |
| `404` | A chave ou o prefixo não pertence ao seu tenant, ou o prefixo é mais amplo que o permitido |
| `503` | O servidor não tem a lista de revogação configurada (depende do Redis) |

:::note Formato de erro desta rota
Os erros `404`, `500` e `503` de `/media/revoke` respondem com `{"error": "..."}`, fora do envelope
padrão `{"success": false, "error": {...}}` descrito em
[Erros e paginação](erros-e-paginacao.md). Decida pelo status HTTP.
:::

A revogação bloqueia a leitura pelo proxy; ela não apaga o objeto do armazenamento.

## Subir um anexo para envio {#upload}

`POST /conversations/:id/attachments` recebe um arquivo em `multipart/form-data` (campo `file`),
guarda no armazenamento e devolve os dados que a rota de envio espera em `attachments`. Scopes:
`conversations:write` e `messages:send`. Limite: **25 MB** por arquivo.

```bash
curl -X POST "$LINKTOR_URL/api/v1/conversations/CONVERSATION_ID/attachments" \
  -H "X-API-Key: $LINKTOR_API_KEY" \
  -F "file=@proposta.pdf;type=application/pdf"
```

Resposta `201`:

```json
{
  "success": true,
  "data": {
    "url": "https://linktor.suaempresa.com/api/v1/media/attachments/5d0c2a6e-1f7b-4a37-9d0e-6a1b9a3f2c10/2b1c0d9e-7a6f-4e5d-8c3b-1a0f9e8d7c6b.pdf",
    "type": "document",
    "filename": "proposta.pdf",
    "mime_type": "application/pdf",
    "size_bytes": 482113
  }
}
```

| Campo | Descrição |
|---|---|
| `url` | Referência durável do arquivo no proxy de mídia |
| `type` | Derivado do tipo MIME: `image`, `video`, `audio` ou `document` (qualquer outro tipo) |
| `filename` | Nome original do arquivo enviado |
| `mime_type` | O `Content-Type` da parte do formulário; se ausente, é detectado pelo conteúdo |
| `size_bytes` | Tamanho em bytes |

Em seguida, passe o objeto de `data` dentro de `attachments` ao enviar a mensagem — o passo a passo
está em [Enviar mensagens](enviar-mensagens.md).

| Status | Quando |
|---|---|
| `400` | Sem o campo `file`, ou arquivo ilegível (`VALIDATION_ERROR`) |
| `413` | Arquivo acima de 25 MB |
| `503` | O servidor não tem armazenamento de anexos configurado |

Os erros `413`, `500` e `503` desta rota também respondem com `{"error": "..."}`, fora do envelope
padrão.

:::note O `:id` do caminho
A rota de upload não associa o arquivo à conversa: o anexo é gravado sob o tenant
(`attachments/<tenant>/...`) e só passa a pertencer a uma mensagem quando você o referencia no
envio. O id no caminho existe para manter a rota sob `/conversations`.
:::

## Boas práticas

- Guarde a **referência durável** (URL sem `exp`/`sig`, ou só a chave). Assine na leitura.
- Trate `410` como "assinar de novo", não como arquivo perdido.
- Não registre URLs assinadas em log: até expirarem, elas dão acesso ao arquivo sem credencial.
- Para servir mídia a usuários finais, assine com `ttl_seconds` curto em vez de repassar a sua chave
  de API ao navegador.
