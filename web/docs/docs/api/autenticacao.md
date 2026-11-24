---
sidebar_position: 2
title: Autenticação
description: Chave de API com scopes para integrações, login de usuário com JWT e cookies, respostas 401/403/404 e limite de requisições.
---

# Autenticação

A API aceita duas credenciais:

| Quem chama | Credencial | Como enviar |
|---|---|---|
| Um sistema (servidor a servidor) | **Chave de API** do tenant | cabeçalho `X-API-Key: lk_sua_chave` |
| Uma pessoa (painel, CLI) | **JWT** obtido no login | `Authorization: Bearer <access_token>` ou cookie `access_token` |

Para integrar um sistema, use sempre a chave de API. Ela não expira por padrão, é restrita ao tenant
que a criou e carrega uma lista de permissões (scopes).

:::warning Envie só uma credencial
O servidor verifica primeiro o JWT (cabeçalho `Authorization` ou cookie `access_token`). Se houver um
token e ele for inválido ou estiver vencido, a resposta é `401` — a chave de API **não** é tentada em
seguida. Em chamadas de servidor, envie apenas `X-API-Key`.
:::

## Chave de API

### Criar pelo painel

1. Entre como usuário com papel `admin` ou `owner`.
2. Abra **Configurações → Chaves de API**.
3. Clique em **Gerar nova chave**.
4. Preencha **Nome da chave** (ex.: `Integração de produção`).
5. Em **Permissões**, deixe **Acesso total** ligado (scope `*`) ou desligue e marque exatamente o que a
   chave pode fazer.
6. Confirme e copie a chave exibida.

:::danger A chave aparece uma única vez
O Linktor guarda apenas o hash da chave e os 12 primeiros caracteres (`key_prefix`). Depois de fechar o
aviso não há como recuperá-la: se perder, apague a chave e gere outra.
:::

### Criar pela API

`POST /api/v1/api-keys` — exige usuário `admin` ou `owner` autenticado por JWT. Uma chave de API não
cria, lista nem apaga outras chaves.

| Campo | Tipo | Obrigatório | Descrição |
|---|---|---|---|
| `name` | texto | sim | Nome para identificar a chave |
| `scopes` | lista de texto | não | Permissões. Vazio ou ausente vira `["*"]` |
| `expires_at` | data RFC 3339 | não | Validade. Sem o campo, a chave não expira |

```bash
curl -X POST "$LINKTOR_URL/api/v1/api-keys" \
  -H "Authorization: Bearer $ACCESS_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Integração de produção",
    "scopes": ["messages:send", "conversations:write", "channels:read"],
    "expires_at": "2027-12-31T23:59:59Z"
  }'
```

Resposta (`201`) — `data.key` só existe nesta resposta:

```json
{
  "success": true,
  "data": {
    "id": "8d1c2b3a-4e5f-4a6b-9c7d-0e1f2a3b4c5d",
    "tenant_id": "5f2a9c0e-1111-4c2b-8d3e-9a7b6c5d4e3f",
    "user_id": "c0ffee00-1234-4abc-9def-000000000001",
    "name": "Integração de produção",
    "key_prefix": "lk_7038a6183",
    "scopes": ["messages:send", "conversations:write", "channels:read"],
    "expires_at": "2027-12-31T23:59:59Z",
    "created_at": "2026-11-24T14:03:11Z",
    "key": "lk_7038a618382e1f4b9a0c5d6e7f8091a2b3c4d5e6f7089a1b"
  }
}
```

A chave tem o formato `lk_` seguido de 48 caracteres hexadecimais.

### Listar e apagar

| Rota | Efeito | Resposta |
|---|---|---|
| `GET /api/v1/api-keys` | Lista as chaves do tenant (sem o segredo), com `last_used_at` | `200` |
| `DELETE /api/v1/api-keys/:id` | Revoga a chave imediatamente | `204` sem corpo |

### Escopo por tenant

A chave pertence ao tenant do administrador que a criou. Toda chamada feita com ela enxerga apenas os
recursos desse tenant — não existe parâmetro `tenant_id`. Pedir um recurso de outro tenant devolve
`404`, igual a um recurso que não existe.

## Scopes

Cada scope tem o formato `recurso:ação`. A tabela mostra o que cada um libera:

| Scope | Libera |
|---|---|
| `*` | Tudo o que uma chave de API pode alcançar |
| `messages:send` | `POST /messages/send`, `POST /conversations/:id/messages`, `POST /conversations/:id/messages/:messageId/reactions`, `POST /conversations/:id/attachments`, `POST /channels/:id/groups/:groupId/messages` |
| `conversations:read` | `GET` em `/conversations` e sub-rotas (inclui o histórico de mensagens); `POST /media/sign` |
| `conversations:write` | Demais métodos em `/conversations` (criar, atualizar, atribuir, resolver, reabrir, escalar, apagar); `POST /media/revoke` |
| `contacts:read` | `GET` em `/contacts`; avatar de contato do canal (`GET /channels/:id/contacts/:contact/avatar`) |
| `contacts:write` | Criar, atualizar, apagar contatos e identidades |
| `channels:read` | `GET` em `/channels` |
| `channels:write` | Criar, atualizar, apagar, testar, conectar, parear, habilitar e desconectar canais |
| `templates:read` / `templates:write` | `/templates` |
| `bots:read` / `bots:write` | `/bots` |
| `flows:read` / `flows:write` | `/flows` |
| `knowledge:read` / `knowledge:write` | `/knowledge-bases` |
| `orders:read` / `orders:write` | `/orders` |
| `analytics:read` | `/analytics` |
| `ai:use` | `/ai/*` |

Regras de correspondência:

- `recurso:write` inclui `recurso:read`.
- `recurso:*` libera todas as ações do recurso (ex.: `channels:*`).
- Em um grupo de rotas, `GET` e `HEAD` pedem o scope de leitura; `POST`, `PUT`, `PATCH` e `DELETE` pedem
  o de escrita.
- Os scopes **somam**: as rotas de envio dentro de `/conversations` pedem `messages:send` **e**
  `conversations:write`. Para responder em uma conversa existente a chave precisa dos dois. O envio
  direto (`POST /messages/send`) pede só `messages:send`.
- Chaves criadas antes de os scopes existirem funcionam como `["*"]`.
- Scopes valem apenas para chave de API. Usuário autenticado por JWT é limitado pelo papel, não por
  scope.

:::note Scope é privilégio mínimo, não isolamento
Algumas rotas autenticadas não têm verificação de scope: `/me`, `GET /tenant`, `GET /messages/:id`,
`/oauth/*`, `GET /campaigns`, `GET /canned-responses`, `GET /settings`, `/whatsapp/flows`, `/vre`,
`POST /conversations/:id/auto-assign` e as sub-rotas `/channels/:id/analytics`, `/payments`, `/calls` e
`/ctwa`. Uma chave com scopes restritos ainda as alcança. Para separar de verdade dois sistemas, use
tenants diferentes.

As rotas de escrita de `/campaigns` e `/canned-responses` checam a permissão do **papel** de quem
chama; o papel de uma chave de API não tem essas permissões por padrão — veja
[Modelos e campanhas](./modelos-e-campanhas.md). O WebSocket `GET /ws` só aceita JWT — veja
[Tempo real](./tempo-real.md).
:::

### O que uma chave de API nunca alcança

A requisição por chave recebe o papel interno `api`, que não é `admin` nem `owner`. Estas rotas
respondem `403` para qualquer chave, mesmo com scope `*`:

`/users`, `/roles`, `/api-keys`, `/audit-logs`, `/observability`, `/sandbox/allowlist`, `PUT /tenant`,
`PUT /settings`, `POST /channels/reencrypt` e `POST /channels/:id/bridge-token`.

## Configurar o cliente

O exemplo monta um cliente HTTP com a chave no cabeçalho e chama `GET /me`, que devolve o usuário
vinculado à chave (o administrador que a criou). Os exemplos em HTTP nativo das demais páginas
reaproveitam este mesmo formato.

<Tabs groupId="linguagem">
<TabItem value="curl" label="cURL">

```bash
export LINKTOR_URL="https://linktor.suaempresa.com"
export LINKTOR_API_KEY="lk_sua_chave"

curl "$LINKTOR_URL/api/v1/me" -H "X-API-Key: $LINKTOR_API_KEY"
```

</TabItem>
<TabItem value="typescript" label="TypeScript">

```typescript
const BASE = `${process.env.LINKTOR_URL}/api/v1`;

export async function linktor<T>(path: string, init: RequestInit = {}): Promise<T> {
  const res = await fetch(`${BASE}${path}`, {
    ...init,
    headers: {
      'X-API-Key': process.env.LINKTOR_API_KEY!,
      'Content-Type': 'application/json',
      ...init.headers,
    },
  });
  const body = res.status === 204 ? {} : await res.json();
  if (!res.ok) {
    // Erro de rota: { success:false, error:{code,message} } — erro de autenticação: { code, message }
    const err = body.error ?? body;
    throw new Error(`${res.status} ${err.code}: ${err.message}`);
  }
  return body.data as T;
}

const eu = await linktor<{ id: string; email: string; role: string }>('/me');
console.log(eu.email, eu.role);
```

Com o SDK: `createClient({ baseUrl: BASE, apiKey: process.env.LINKTOR_API_KEY })`.

</TabItem>
<TabItem value="python" label="Python">

```python
import os
import requests

BASE = f"{os.environ['LINKTOR_URL']}/api/v1"

session = requests.Session()
session.headers.update({"X-API-Key": os.environ["LINKTOR_API_KEY"]})


def linktor(method: str, path: str, **kwargs):
    res = session.request(method, f"{BASE}{path}", timeout=30, **kwargs)
    body = res.json() if res.content else {}
    if not res.ok:
        # Erro de rota: {"success": false, "error": {...}} — erro de autenticação: {"code", "message"}
        err = body.get("error") or body
        raise RuntimeError(f"{res.status_code} {err.get('code')}: {err.get('message')}")
    return body.get("data")


eu = linktor("GET", "/me")
print(eu["email"], eu["role"])
```

Com o SDK: `LinktorClient(base_url=BASE, api_key=os.environ["LINKTOR_API_KEY"])`.

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
	"time"
)

var httpClient = &http.Client{Timeout: 30 * time.Second}

// Envelope cobre os dois formatos de resposta: o de rota e o de autenticação.
type Envelope struct {
	Success bool            `json:"success"`
	Data    json.RawMessage `json:"data"`
	Error   *APIError       `json:"error"`
	Code    string          `json:"code"`    // erro de autenticação / limite
	Message string          `json:"message"` // idem
}

type APIError struct {
	Code    string            `json:"code"`
	Message string            `json:"message"`
	Details map[string]string `json:"details"`
}

func Linktor(method, path string, in, out any) error {
	var body io.Reader
	if in != nil {
		raw, err := json.Marshal(in)
		if err != nil {
			return err
		}
		body = bytes.NewReader(raw)
	}
	req, err := http.NewRequest(method, os.Getenv("LINKTOR_URL")+"/api/v1"+path, body)
	if err != nil {
		return err
	}
	req.Header.Set("X-API-Key", os.Getenv("LINKTOR_API_KEY"))
	req.Header.Set("Content-Type", "application/json")

	res, err := httpClient.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	raw, _ := io.ReadAll(res.Body)
	var env Envelope
	_ = json.Unmarshal(raw, &env)
	if res.StatusCode >= 400 {
		if env.Error != nil {
			return fmt.Errorf("%d %s: %s", res.StatusCode, env.Error.Code, env.Error.Message)
		}
		return fmt.Errorf("%d %s: %s", res.StatusCode, env.Code, env.Message)
	}
	if out != nil && len(env.Data) > 0 {
		return json.Unmarshal(env.Data, out)
	}
	return nil
}

func main() {
	var eu struct {
		Email string `json:"email"`
		Role  string `json:"role"`
	}
	if err := Linktor(http.MethodGet, "/me", nil, &eu); err != nil {
		panic(err)
	}
	fmt.Println(eu.Email, eu.Role)
}
```

Com o SDK: `linktor.NewClient(linktor.WithBaseURL(base), linktor.WithAPIKey(key))`.

</TabItem>
<TabItem value="java" label="Java">

```java
import java.net.URI;
import java.net.http.HttpClient;
import java.net.http.HttpRequest;
import java.net.http.HttpResponse;
import java.time.Duration;

public class Linktor {
    static final String BASE = System.getenv("LINKTOR_URL") + "/api/v1";
    static final HttpClient HTTP = HttpClient.newBuilder()
        .connectTimeout(Duration.ofSeconds(10))
        .build();

    /** Devolve o corpo JSON; lança exceção em status >= 400. */
    static String call(String method, String path, String json) throws Exception {
        HttpRequest.BodyPublisher body = json == null
            ? HttpRequest.BodyPublishers.noBody()
            : HttpRequest.BodyPublishers.ofString(json);

        HttpRequest request = HttpRequest.newBuilder(URI.create(BASE + path))
            .timeout(Duration.ofSeconds(30))
            .header("X-API-Key", System.getenv("LINKTOR_API_KEY"))
            .header("Content-Type", "application/json")
            .method(method, body)
            .build();

        HttpResponse<String> response = HTTP.send(request, HttpResponse.BodyHandlers.ofString());
        if (response.statusCode() >= 400) {
            throw new RuntimeException(response.statusCode() + " " + response.body());
        }
        return response.body();
    }

    public static void main(String[] args) throws Exception {
        System.out.println(call("GET", "/me", null));
    }
}
```

Com o SDK: `LinktorClient.builder().baseUrl(BASE).apiKey(key).build()`.

</TabItem>
<TabItem value="csharp" label="C#">

```csharp
using System.Net.Http.Json;
using System.Text.Json;

var http = new HttpClient
{
    BaseAddress = new Uri(Environment.GetEnvironmentVariable("LINKTOR_URL") + "/api/v1/"),
    Timeout = TimeSpan.FromSeconds(30),
};
http.DefaultRequestHeaders.Add("X-API-Key", Environment.GetEnvironmentVariable("LINKTOR_API_KEY"));

// Caminhos relativos, sem barra inicial, para respeitar o BaseAddress.
var response = await http.GetAsync("me");
var body = await response.Content.ReadFromJsonAsync<JsonElement>();

if (!response.IsSuccessStatusCode)
{
    // Erro de rota: { success:false, error:{...} } — erro de autenticação: { code, message }
    var err = body.TryGetProperty("error", out var e) ? e : body;
    throw new Exception($"{(int)response.StatusCode} {err.GetProperty("code")}: {err.GetProperty("message")}");
}

var eu = body.GetProperty("data");
Console.WriteLine($"{eu.GetProperty("email")} {eu.GetProperty("role")}");
```

</TabItem>
<TabItem value="php" label="PHP">

```php
<?php
require 'vendor/autoload.php';

use GuzzleHttp\Client;

$http = new Client([
    'base_uri'    => getenv('LINKTOR_URL') . '/api/v1/',
    'timeout'     => 30,
    'http_errors' => false,
    'headers'     => ['X-API-Key' => getenv('LINKTOR_API_KEY')],
]);

function linktor(Client $http, string $method, string $path, ?array $json = null): ?array
{
    $res  = $http->request($method, ltrim($path, '/'), $json === null ? [] : ['json' => $json]);
    $body = json_decode((string) $res->getBody(), true) ?? [];

    if ($res->getStatusCode() >= 400) {
        // Erro de rota: ['success' => false, 'error' => [...]] — erro de autenticação: ['code', 'message']
        $err = $body['error'] ?? $body;
        throw new RuntimeException("{$res->getStatusCode()} {$err['code']}: {$err['message']}");
    }
    return $body['data'] ?? null;
}

$eu = linktor($http, 'GET', '/me');
echo "{$eu['email']} {$eu['role']}\n";
```

</TabItem>
<TabItem value="rust" label="Rust">

```rust
use reqwest::header::{HeaderMap, HeaderValue};
use serde_json::Value;

fn client() -> reqwest::Result<reqwest::Client> {
    let mut headers = HeaderMap::new();
    let key = std::env::var("LINKTOR_API_KEY").expect("LINKTOR_API_KEY");
    headers.insert("X-API-Key", HeaderValue::from_str(&key).expect("chave inválida"));

    reqwest::Client::builder()
        .default_headers(headers)
        .timeout(std::time::Duration::from_secs(30))
        .build()
}

#[tokio::main]
async fn main() -> Result<(), Box<dyn std::error::Error>> {
    let base = format!("{}/api/v1", std::env::var("LINKTOR_URL")?);
    let res = client()?.get(format!("{base}/me")).send().await?;

    let status = res.status();
    let body: Value = res.json().await?;
    if !status.is_success() {
        // Erro de rota: {"success":false,"error":{...}} — erro de autenticação: {"code","message"}
        let err = body.get("error").unwrap_or(&body);
        return Err(format!("{status} {}: {}", err["code"], err["message"]).into());
    }

    println!("{} {}", body["data"]["email"], body["data"]["role"]);
    Ok(())
}
```

Com o SDK: `LinktorClient::builder().base_url(base).api_key(key).build()?`.

</TabItem>
</Tabs>

Resposta de `GET /me` (`200`):

```json
{
  "success": true,
  "data": {
    "id": "c0ffee00-1234-4abc-9def-000000000001",
    "tenant_id": "5f2a9c0e-1111-4c2b-8d3e-9a7b6c5d4e3f",
    "email": "admin@exemplo.com",
    "name": "Administrador",
    "role": "admin",
    "status": "active",
    "created_at": "2026-10-01T12:00:00Z",
    "updated_at": "2026-11-20T08:30:00Z"
  }
}
```

:::warning Chave de API não funciona direto do navegador
O CORS do servidor não inclui `X-API-Key` em `Access-Control-Allow-Headers`. Uma página em outra origem
que tente chamar a API com a chave falha no preflight — e, de todo modo, a chave não deve ser exposta
no navegador. Chame a API a partir do seu servidor.
:::

:::note SDKs
Onde aparece "Com o SDK", `BASE` já inclui `/api/v1`: nenhum SDK acrescenta esse sufixo. As abas C# e
PHP não trazem a linha do SDK porque, nesta versão, o SDK .NET não compila e o SDK PHP perde o caminho
`/api/v1` ao montar a URL. O `auth.login` dos SDKs não deixa o cliente autenticado de forma confiável
(a resposta vem no envelope `{ success, data }` com campos `snake_case`, que eles não leem): faça o
login por HTTP e passe o `access_token` ao construir o cliente. A consulta do usuário atual deles chama
`/auth/me`, que não existe — a rota é `/me`. Veja
[SDKs](../sdks/visao-geral.md).
:::

## Respostas 401, 403 e 404 {#erros-de-acesso}

| Código | Quando | `code` | `message` |
|---|---|---|---|
| `401` | Nenhuma credencial enviada | `UNAUTHORIZED` | `missing authentication credentials` |
| `401` | Chave inexistente, revogada ou vencida | `UNAUTHORIZED` | `invalid API key` |
| `401` | JWT inválido ou vencido | `TOKEN_INVALID` | `invalid or expired token` |
| `403` | A chave não tem o scope da rota | `FORBIDDEN` | `API key is missing the required scope: messages:send` |
| `403` | A rota exige papel `admin`/`owner` | `FORBIDDEN` | `insufficient permissions` |
| `404` | O recurso não existe **ou** é de outro tenant | `NOT_FOUND`, `CHANNEL_NOT_FOUND`… | varia |

O `404` para recurso de outro tenant é proposital: a API não revela se um id existe fora do seu tenant.

Os erros `401` e `403` desta tabela são emitidos antes da rota e vêm sem o envelope:

```json
{
  "code": "FORBIDDEN",
  "message": "API key is missing the required scope: messages:send",
  "details": null
}
```

## Login de usuário (JWT)

Use o login quando a chamada representa uma pessoa — por exemplo, um script administrativo que cria
chaves de API ou usuários. As rotas `/auth/*` não exigem autenticação.

### Entrar

`POST /api/v1/auth/login`

| Campo | Tipo | Obrigatório | Regra |
|---|---|---|---|
| `email` | texto | sim | e-mail válido |
| `password` | texto | sim | mínimo de 6 caracteres |

```bash
curl -X POST "$LINKTOR_URL/api/v1/auth/login" \
  -H "Content-Type: application/json" \
  -c cookies.txt \
  -d '{"email": "admin@exemplo.com", "password": "sua-senha"}'
```

Resposta (`200`):

```json
{
  "success": true,
  "data": {
    "user": {
      "id": "c0ffee00-1234-4abc-9def-000000000001",
      "tenant_id": "5f2a9c0e-1111-4c2b-8d3e-9a7b6c5d4e3f",
      "email": "admin@exemplo.com",
      "name": "Administrador",
      "role": "admin",
      "status": "active",
      "created_at": "2026-10-01T12:00:00Z",
      "updated_at": "2026-11-20T08:30:00Z"
    },
    "access_token": "eyJhbGciOiJIUzI1NiIs...",
    "refresh_token": "eyJhbGciOiJIUzI1NiIs...",
    "expires_in": 900
  }
}
```

Credenciais erradas devolvem `401` com `code: "INVALID_CREDENTIALS"`.

Os tokens chegam de duas formas ao mesmo tempo:

- **No corpo**, para clientes de API e CLI: envie `Authorization: Bearer <access_token>` nas chamadas
  seguintes.
- **Em cookies `HttpOnly`**, para navegadores: `access_token` e `refresh_token`, com `Path=/`. O painel
  usa este caminho — o JavaScript da página nunca lê o token.

| Item | Padrão | Observação |
|---|---|---|
| Validade do access token | 15 minutos | `expires_in` vem em segundos |
| Validade do refresh token | 168 horas (7 dias) | |
| Cookie `Secure` | ligado | exige HTTPS; desligue apenas em desenvolvimento local |
| Cookie `SameSite` | `lax` | aceita `lax`, `strict` ou `none` |
| Domínio do cookie | vazio | o do próprio servidor |

Esses valores são ajustáveis na configuração do servidor (seção `jwt`) — veja
[Configuração](../implantar/configuracao.md).

### Renovar

`POST /api/v1/auth/refresh` troca o refresh token por um **novo par** de tokens. O refresh token pode
ir no corpo ou no cookie `refresh_token`; o corpo tem prioridade.

```bash
curl -X POST "$LINKTOR_URL/api/v1/auth/refresh" \
  -H "Content-Type: application/json" \
  -d '{"refresh_token": "eyJhbGciOiJIUzI1NiIs..."}'
```

```json
{
  "success": true,
  "data": {
    "access_token": "eyJhbGciOiJIUzI1NiIs...",
    "refresh_token": "eyJhbGciOiJIUzI1NiIs...",
    "expires_in": 900
  }
}
```

Sem refresh token a resposta é `401` (`UNAUTHORIZED`, `missing refresh token`). Com um token inválido,
vencido ou de usuário inativo, é `401` (`TOKEN_INVALID`) e os cookies de sessão são apagados — faça
login de novo.

### Sair

`POST /api/v1/auth/logout` apaga os dois cookies e responde `200`:

```json
{ "success": true, "data": { "message": "Logged out" } }
```

:::note
O logout apenas remove os cookies do navegador. Um access token que já estava em mãos continua válido
até expirar (15 minutos por padrão). Em um cliente de API, descarte os tokens localmente.
:::

### Perfil do usuário autenticado

| Rota | Corpo | Efeito |
|---|---|---|
| `GET /api/v1/me` | — | Dados do usuário |
| `PUT /api/v1/me` | `name`, `avatar_url` (ambos opcionais) | Atualiza o próprio perfil |
| `PUT /api/v1/me/password` | `current_password`, `new_password` (mínimo de 8 caracteres) | Troca a senha |

## Limite de requisições

O limite é de **100 requisições por minuto**, em janela fixa (o contador zera na virada do minuto).

| Rotas | Contado por |
|---|---|
| Rotas autenticadas de `/api/v1` | tenant (todas as chaves e usuários do tenant somam no mesmo contador) |
| `/api/v1/auth/*` | endereço IP |
| `/api/v1/webhooks/*` (entrada dos provedores) | endereço IP |

Toda resposta dessas rotas traz:

| Cabeçalho | Conteúdo |
|---|---|
| `X-RateLimit-Limit` | limite da janela (`100`) |
| `X-RateLimit-Remaining` | quanto resta na janela atual |
| `X-RateLimit-Reset` | instante Unix (segundos) em que a janela termina |

Ao estourar, a resposta é `429`, sem envelope:

```json
{
  "code": "RATE_LIMITED",
  "message": "Too many requests. Please try again later.",
  "retry_after": 37
}
```

`retry_after` é o número de segundos até a janela zerar. Espere esse tempo antes de tentar de novo —
veja as recomendações em [Erros e paginação](./erros-e-paginacao.md).

:::note
O limite depende do Redis. Se o servidor subir sem conexão com o Redis, o limite fica desligado e os
cabeçalhos `X-RateLimit-*` não aparecem. O valor de 100 por minuto é fixo no código nesta versão, sem
variável de configuração.
:::

## Boas práticas

- Uma chave por sistema integrado, com o nome do sistema — revogar uma não derruba as outras.
- Marque só os scopes necessários. Para um sistema que apenas envia notificações: `messages:send`.
- Guarde a chave em cofre de segredos ou variável de ambiente; nunca no código-fonte nem no navegador.
- Defina `expires_at` para chaves temporárias (homologação, parceiros).
- Para trocar uma chave sem parar a integração: crie a nova, atualize o sistema, apague a antiga.
