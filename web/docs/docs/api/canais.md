---
sidebar_position: 7
title: Canais
description: Criar, testar, conectar (QR code, código de pareamento, passkey), habilitar, desconectar e remover canais pela API, com exemplos em oito linguagens.
---

# Canais

Um canal é uma conexão do seu tenant com um provedor de mensagens: um número de WhatsApp, um bot do
Telegram, uma caixa de e-mail. Tudo o que o painel faz em **Canais** pode ser feito pela API com uma
[chave de API](./autenticacao.md) — é assim que uma aplicação externa provisiona canais para os
próprios clientes.

Esta página cobre o ciclo de vida. O que cada tipo de canal exige em `config` e `credentials` está na
página do canal, em [Canais → Visão geral](../canais/visao-geral.md).

## Rotas

| Método | Caminho | Scope | O que faz |
|---|---|---|---|
| `GET` | `/channels` | `channels:read` | Lista os canais do tenant |
| `POST` | `/channels` | `channels:write` | Cria |
| `GET` | `/channels/:id` | `channels:read` | Obtém |
| `PUT` | `/channels/:id` | `channels:write` | Atualiza (parcial) |
| `DELETE` | `/channels/:id` | `channels:write` | Remove |
| `POST` | `/channels/test` | `channels:write` | Valida credenciais sem criar o canal |
| `POST` | `/channels/:id/connect` | `channels:write` | Conecta; no WhatsApp não oficial devolve o QR code |
| `POST` | `/channels/:id/pair` | `channels:write` | Código de pareamento (WhatsApp não oficial) |
| `POST` | `/channels/:id/passkey/response` | `channels:write` | Entrega a assinatura de passkey (WhatsApp não oficial) |
| `PUT` | `/channels/:id/enabled` | `channels:write` | Habilita ou desabilita |
| `PUT` | `/channels/:id/status` | `channels:write` | Forma antiga de habilitar/desabilitar (`active` / `inactive`) |
| `POST` | `/channels/:id/disconnect` | `channels:write` | Encerra a conexão |
| `GET` | `/channels/:id/contacts` | `channels:read` | Agenda do aparelho pareado (WhatsApp não oficial) |
| `GET` | `/channels/:id/contacts/:contact/avatar` | `channels:read` + `contacts:read` | Foto de um contato |
| `GET` | `/channels/:id/bridge-health` | `channels:read` | Saúde do Linktor Bridge do canal |
| `POST` | `/channels/:id/bridge-token` | papel `admin`/`owner` | Gera o token do Linktor Bridge |
| `POST` | `/channels/reencrypt` | papel `admin`/`owner` | Regrava as credenciais com a chave de criptografia atual |
| `GET` | `/channels/:id/coexistence-status` | `channels:read` | Estado da coexistência (WhatsApp oficial) |
| `POST` | `/channels/:id/subscribe-echoes` | `channels:write` | Assina os ecos do app (WhatsApp oficial) |

Também existem atalhos `POST /channels/test-whatsapp`, `test-telegram`, `test-twilio`,
`test-facebook`, `test-instagram`, `test-teams`, `test-slack`, `test-mattermost` e `test-email`: são
o mesmo `POST /channels/test` com o `type` já fixado.

`channels:write` implica `channels:read`. As rotas marcadas com papel `admin`/`owner` **não** são
alcançáveis por API key (ela opera com o papel `api`) e respondem `403`.

Enviar para um grupo (`POST /channels/:id/groups/:groupId/messages`) exige `messages:send`, não
`channels:write`, e está em [Enviar mensagens](./enviar-mensagens.md).

## O objeto canal

```json
{
  "id": "CHANNEL_ID",
  "tenant_id": "b7a1d0c4-2f6e-4f0a-9d58-3f2b7c1e9a10",
  "type": "telegram",
  "name": "Bot de atendimento",
  "identifier": "@suaempresa_bot",
  "enabled": true,
  "connection_status": "disconnected",
  "environment": "production",
  "config": { "webhook_events": "message.received,message.status" },
  "webhook_url": "https://app.suaempresa.com/webhooks/linktor",
  "created_at": "2026-11-24T13:00:00Z",
  "updated_at": "2026-11-24T13:00:00Z"
}
```

| Campo | Descrição |
|---|---|
| `type` | `whatsapp_official`, `whatsapp` (não oficial), `whatsapp_unofficial` (sinônimo aceito de `whatsapp`), `telegram`, `webchat`, `sms`, `rcs`, `instagram`, `facebook`, `email`, `voice`, `teams`, `slack`, `mattermost`, `direto`. Imutável |
| `enabled` | Se o canal está habilitado no Linktor. Novo canal nasce `true` |
| `connection_status` | `disconnected`, `connecting`, `connected`, `error`. Novo canal nasce `disconnected` |
| `environment` | `production` (padrão) ou `sandbox`. Imutável depois da criação |
| `config` | Parâmetros não secretos, mapa `string → string` |
| `webhook_url` | Endereço da **sua** aplicação que recebe os eventos deste canal ([Receber eventos](./receber-eventos.md)) |
| `public_credentials` | Só quando existem: `smtp_username`, `imap_username`, `verify_token` |
| `bridge_health` | Só em canais com Linktor Bridge conectado |

Dois estados independentes: `enabled` é uma decisão sua (ligar/desligar); `connection_status` é o que
o Linktor observa na conexão com o provedor. Não existe campo `status` no canal.

### Segredos nunca voltam

- `credentials` é **somente escrita**: é cifrado no banco e não aparece em nenhuma resposta.
- Chaves sensíveis dentro de `config` (`access_token`, `bot_token`, `app_secret`, `api_key`,
  `auth_token`, `webhook_secret`, `password`…) voltam como `"__redacted__"`.
- Em Teams, Slack e Mattermost, alguns identificadores não secretos guardados em `credentials`
  (`app_id`, `base_url`, `bot_user_id`…) são copiados para `config` na leitura.

## Configurar o cliente

Os exemplos desta página usam o SDK oficial onde ele funciona contra a API atual (TypeScript, Go,
Java, Rust e, em parte, Python) e o cliente HTTP nativo nas demais abas. Nenhum SDK acrescenta
`/api/v1` sozinho: passe sempre a base completa, `https://linktor.suaempresa.com/api/v1`.

:::warning C#, PHP e parte do Python usam HTTP nativo
- **C#** — o SDK .NET não compila nesta versão; os exemplos usam `HttpClient`.
- **PHP** — o SDK monta as requisições de um jeito que descarta o `/api/v1` da URL base, e as chamadas
  caem fora da API; os exemplos usam Guzzle com a URL completa.
- **Python** — o SDK só reconhece os tipos `whatsapp`, `telegram`, `facebook`, `instagram`, `webchat`,
  `sms`, `email` e `rcs`. Qualquer resposta com um canal de outro tipo (`whatsapp_official`,
  `whatsapp_unofficial`, `voice`, `teams`, `slack`, `mattermost`, `direto`) lança erro de validação —
  inclusive em `list`, se o tenant tiver um canal desses. Os exemplos usam o SDK apenas para criar e
  remover o canal de Telegram, e `requests` no restante.

O estado de cada SDK está em [SDKs → Visão geral](../sdks/visao-geral.md).
:::

<Tabs groupId="linguagem">
<TabItem value="curl" label="cURL">

```bash
export LINKTOR_URL="https://linktor.suaempresa.com"
export LINKTOR_API_KEY="lk_sua_chave"
```

</TabItem>
<TabItem value="typescript" label="TypeScript">

```typescript
import { createClient } from '@linktor/sdk';

const linktor = createClient({
  baseUrl: `${process.env.LINKTOR_URL}/api/v1`,
  apiKey: process.env.LINKTOR_API_KEY, // enviado como X-API-Key
});
```

</TabItem>
<TabItem value="python" label="Python">

```python
import os
import requests
from linktor import LinktorClient

BASE = f"{os.environ['LINKTOR_URL']}/api/v1"

# SDK: usado nesta página só onde o tipo do canal existe no enum do SDK (veja o aviso abaixo)
linktor = LinktorClient(
    base_url=BASE,
    api_key=os.environ["LINKTOR_API_KEY"],  # enviado como X-API-Key
)

# HTTP nativo: funciona com qualquer tipo de canal
session = requests.Session()
session.headers.update({"X-API-Key": os.environ["LINKTOR_API_KEY"]})


def api(method: str, path: str, body: dict | None = None):
    res = session.request(method, f"{BASE}{path}", json=body, timeout=60)
    dados = res.json() if res.content else {}
    if not res.ok:
        err = dados.get("error") or dados
        raise RuntimeError(f"{res.status_code} {err.get('code')}: {err.get('message')}")
    return dados.get("data")
```

</TabItem>
<TabItem value="go" label="Go">

```go
import (
	"context"
	"os"

	linktor "github.com/linktor/linktor-go"
	"github.com/linktor/linktor-go/types"
)

ctx := context.Background()
client := linktor.NewClient(
	linktor.WithBaseURL(os.Getenv("LINKTOR_URL")+"/api/v1"),
	linktor.WithAPIKey(os.Getenv("LINKTOR_API_KEY")), // enviado como X-API-Key
)
```

</TabItem>
<TabItem value="java" label="Java">

```java
import io.linktor.LinktorClient;
import io.linktor.types.Channel;

LinktorClient linktor = LinktorClient.builder()
    .baseUrl(System.getenv("LINKTOR_URL") + "/api/v1")
    .apiKey(System.getenv("LINKTOR_API_KEY")) // enviado como X-API-Key
    .build();
```

</TabItem>
<TabItem value="csharp" label="C#">

```csharp
using System.Net.Http.Json;
using System.Text.Json;

var baseUrl = Environment.GetEnvironmentVariable("LINKTOR_URL") + "/api/v1";
var http = new HttpClient { Timeout = TimeSpan.FromSeconds(60) };
http.DefaultRequestHeaders.Add("X-API-Key", Environment.GetEnvironmentVariable("LINKTOR_API_KEY"));

// Devolve o conteúdo de "data" do envelope (ou um JsonElement vazio em respostas 204).
async Task<JsonElement> Api(HttpMethod method, string path, object? body = null)
{
    using var request = new HttpRequestMessage(method, baseUrl + path);
    if (body is not null) request.Content = JsonContent.Create(body);

    using var response = await http.SendAsync(request);
    var texto = await response.Content.ReadAsStringAsync();
    var json = texto.Length > 0 ? JsonDocument.Parse(texto).RootElement.Clone() : default;

    if (!response.IsSuccessStatusCode)
    {
        throw new Exception($"{(int)response.StatusCode} {texto}");
    }
    return json.ValueKind == JsonValueKind.Object && json.TryGetProperty("data", out var data)
        ? data
        : default;
}
```

</TabItem>
<TabItem value="php" label="PHP">

```php
<?php
require 'vendor/autoload.php';

// Devolve o conteúdo de "data" do envelope (ou null em respostas 204).
function api(string $method, string $path, ?array $body = null): ?array
{
    static $http = null;
    $http ??= new GuzzleHttp\Client(['http_errors' => false, 'timeout' => 60]);

    // URL completa: não use base_uri com caminho, o Guzzle descarta o /api/v1
    $res = $http->request($method, getenv('LINKTOR_URL') . '/api/v1' . $path, [
        'headers' => ['X-API-Key' => getenv('LINKTOR_API_KEY')],
    ] + ($body === null ? [] : ['json' => $body]));

    $dados = json_decode((string) $res->getBody(), true) ?? [];
    if ($res->getStatusCode() >= 400) {
        $err = $dados['error'] ?? $dados;
        throw new RuntimeException($res->getStatusCode() . ' ' . ($err['code'] ?? '') . ': ' . ($err['message'] ?? ''));
    }
    return $dados['data'] ?? null;
}
```

</TabItem>
<TabItem value="rust" label="Rust">

```rust
use linktor::{types::*, LinktorClient};

let client = LinktorClient::builder()
    .base_url(format!("{}/api/v1", std::env::var("LINKTOR_URL")?))
    .api_key(std::env::var("LINKTOR_API_KEY")?) // enviado como X-API-Key
    .build()?;
```

</TabItem>
</Tabs>

Instalação e detalhes de cada SDK: [SDKs → Visão geral](../sdks/visao-geral.md).

## Criar um canal

| Campo | Obrigatório | Descrição |
|---|---|---|
| `type` | sim | Tipo do canal (tabela acima) |
| `name` | sim | Nome de exibição |
| `identifier` | não | Rótulo do endereço do canal (número, `@usuario`, e-mail) |
| `environment` | não | `production` (padrão) ou `sandbox` |
| `config` | não | Parâmetros não secretos (`string → string`) |
| `credentials` | não | Segredos (`string → string`) |
| `webhook_url` | não | Onde entregar os eventos deste canal |

O exemplo cria um canal do Telegram. As chaves de `config`/`credentials` de cada tipo estão na página
do canal — por exemplo [Telegram](../canais/telegram.md), [WhatsApp
oficial](../canais/whatsapp-oficial.md), [WhatsApp não oficial](../canais/whatsapp-nao-oficial.md).

<Tabs groupId="linguagem">
<TabItem value="curl" label="cURL">

```bash
curl -X POST "$LINKTOR_URL/api/v1/channels" \
  -H "X-API-Key: $LINKTOR_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "type": "telegram",
    "name": "Bot de atendimento",
    "credentials": { "bot_token": "123456:ABC-seu-token" },
    "webhook_url": "https://app.suaempresa.com/webhooks/linktor"
  }'
```

</TabItem>
<TabItem value="typescript" label="TypeScript">

```typescript
const canal = await linktor.channels.create({
  type: 'telegram',
  name: 'Bot de atendimento',
  credentials: { bot_token: '123456:ABC-seu-token' }, // somente escrita
  webhook_url: 'https://app.suaempresa.com/webhooks/linktor',
});

console.log(canal.id, canal.connection_status); // "disconnected"
```

</TabItem>
<TabItem value="python" label="Python">

```python
canal = linktor.channels.create(
    type="telegram",
    name="Bot de atendimento",
    credentials={"bot_token": "123456:ABC-seu-token"},  # somente escrita
    webhook_url="https://app.suaempresa.com/webhooks/linktor",
)

print(canal.id, canal.connection_status.value)  # "disconnected"
```

</TabItem>
<TabItem value="go" label="Go">

```go
canal, err := client.Channels.Create(ctx, &types.CreateChannelInput{
	Type:        types.ChannelTypeTelegram,
	Name:        "Bot de atendimento",
	Credentials: map[string]string{"bot_token": "123456:ABC-seu-token"}, // somente escrita
	WebhookURL:  "https://app.suaempresa.com/webhooks/linktor",
})
if err != nil {
	log.Fatal(err)
}

fmt.Println(canal.ID, canal.ConnectionStatus) // "disconnected"
```

</TabItem>
<TabItem value="java" label="Java">

```java
Channel.ChannelModel canal = linktor.channels.create(
    Channel.CreateChannelInput.builder()
        .type(Channel.ChannelType.TELEGRAM)
        .name("Bot de atendimento")
        .credentials(Map.of("bot_token", "123456:ABC-seu-token")) // somente escrita
        .webhookUrl("https://app.suaempresa.com/webhooks/linktor")
        .build());

System.out.println(canal.getId() + " " + canal.getConnectionStatus()); // DISCONNECTED
```

</TabItem>
<TabItem value="csharp" label="C#">

```csharp
var canal = await Api(HttpMethod.Post, "/channels", new
{
    type = "telegram",
    name = "Bot de atendimento",
    credentials = new Dictionary<string, string> { ["bot_token"] = "123456:ABC-seu-token" }, // somente escrita
    webhook_url = "https://app.suaempresa.com/webhooks/linktor",
});

Console.WriteLine($"{canal.GetProperty("id")} {canal.GetProperty("connection_status")}"); // "disconnected"
```

</TabItem>
<TabItem value="php" label="PHP">

```php
$canal = api('POST', '/channels', [
    'type'        => 'telegram',
    'name'        => 'Bot de atendimento',
    'credentials' => ['bot_token' => '123456:ABC-seu-token'], // somente escrita
    'webhook_url' => 'https://app.suaempresa.com/webhooks/linktor',
]);

echo $canal['id'] . ' ' . $canal['connection_status'] . PHP_EOL; // "disconnected"
```

</TabItem>
<TabItem value="rust" label="Rust">

```rust
use std::collections::HashMap;

let canal = client
    .channels()
    .create(
        CreateChannelInput::new("Bot de atendimento", ChannelType::Telegram)
            // somente escrita
            .credentials(HashMap::from([(
                "bot_token".to_string(),
                "123456:ABC-seu-token".to_string(),
            )]))
            .webhook_url("https://app.suaempresa.com/webhooks/linktor"),
    )
    .await?;

println!("{} {:?}", canal.id, canal.connection_status); // Disconnected
```

</TabItem>
</Tabs>

Resposta `201`:

```json
{
  "success": true,
  "data": {
    "id": "CHANNEL_ID",
    "tenant_id": "b7a1d0c4-2f6e-4f0a-9d58-3f2b7c1e9a10",
    "type": "telegram",
    "name": "Bot de atendimento",
    "enabled": true,
    "connection_status": "disconnected",
    "environment": "production",
    "webhook_url": "https://app.suaempresa.com/webhooks/linktor",
    "created_at": "2026-11-24T13:00:00Z",
    "updated_at": "2026-11-24T13:00:00Z"
  }
}
```

O canal nasce **habilitado e desconectado**. O próximo passo é [conectar](#conectar).

Para a aplicação validar a assinatura dos eventos recebidos, envie o segredo em
`credentials.webhook_secret`; para limitar os tipos de evento, use `config.webhook_events` (lista
separada por vírgula). Detalhes em [Receber eventos](./receber-eventos.md).

Erros de criação (`400 VALIDATION_ERROR`):

| Mensagem | Causa |
|---|---|
| `Invalid request body` | Falta `type` ou `name`, ou `config`/`credentials` não são mapas de texto |
| `channel type "…" is not enabled for this deployment` | O servidor restringe os tipos permitidos (`LINKTOR_ENABLED_CHANNEL_TYPES`) |
| `invalid channel environment "…"` | `environment` diferente de `production`/`sandbox` |
| `sandbox channel requires credentials["credential_environment"] = "sandbox"` | Canal `sandbox` sem a declaração explícita de credencial de teste |
| `credentials declared as sandbox cannot be bound to a production channel` | O inverso |

Canais de homologação têm regras próprias; veja [Sandbox](../painel/sandbox.md).

:::note
A criação **não** valida as credenciais no provedor. Use [`POST /channels/test`](#testar-credenciais)
antes, ou observe o resultado do `connect`.
:::

## Testar credenciais {#testar-credenciais}

`POST /channels/test` confere a configuração **sem criar nada**. Para Teams, Slack, Mattermost,
e-mail, Facebook e Instagram o Linktor vai de fato ao provedor; nos demais tipos a verificação é só
de campos obrigatórios.

```bash
curl -X POST "$LINKTOR_URL/api/v1/channels/test" \
  -H "X-API-Key: $LINKTOR_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "type": "slack",
    "credentials": { "bot_token": "xoxb-seu-token" }
  }'
```

```json
{
  "success": true,
  "data": {
    "status": "ok",
    "type": "slack",
    "valid": true,
    "message": "connected to Slack (team \"Sua Empresa\", bot \"linktor\")"
  }
}
```

Quando não há verificação ao vivo, `message` é `configuration accepted`. Falha devolve `400` com a
causa em `error.message` (`bot_token is required`, `connection test failed: …`).

| Tipo | O que é verificado |
|---|---|
| `teams`, `slack`, `mattermost` | Autenticação real no provedor |
| `email` | Envio (credencial do provedor ou login SMTP) e, se houver IMAP configurado, login e abertura da pasta. Nada é enviado |
| `facebook`, `instagram` | Leitura da Página / conta com o token |
| `whatsapp_official`, `telegram`, `sms`, `rcs` | Presença e formato dos campos obrigatórios |
| `webchat`, `voice`, `direto`, `whatsapp` (não oficial) | Não suportado: `400 unsupported channel type` |

Para testar um canal **já salvo** sem redigitar os segredos, envie `channel_id`: o que estiver em
branco é completado com o valor guardado, e o que você enviar prevalece.

```bash
curl -X POST "$LINKTOR_URL/api/v1/channels/test" \
  -H "X-API-Key: $LINKTOR_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{ "type": "email", "channel_id": "'"$CHANNEL_ID"'" }'
```

Esta rota não existe nos SDKs; use HTTP.

## Listar e obter

<Tabs groupId="linguagem">
<TabItem value="curl" label="cURL">

```bash
curl "$LINKTOR_URL/api/v1/channels" -H "X-API-Key: $LINKTOR_API_KEY"
curl "$LINKTOR_URL/api/v1/channels/$CHANNEL_ID" -H "X-API-Key: $LINKTOR_API_KEY"
```

</TabItem>
<TabItem value="typescript" label="TypeScript">

```typescript
const canais = await linktor.channels.list();
const canal = await linktor.channels.get(channelId);
```

</TabItem>
<TabItem value="python" label="Python">

```python
# requests: channels.list() do SDK falha se o tenant tiver um canal de tipo fora do enum do SDK
canais = api("GET", "/channels")
canal = api("GET", f"/channels/{channel_id}")
print(canal["type"], canal["connection_status"])
```

</TabItem>
<TabItem value="go" label="Go">

```go
canais, err := client.Channels.List(ctx, nil)
canal, err := client.Channels.Get(ctx, channelID)
```

</TabItem>
<TabItem value="java" label="Java">

```java
List<Channel.ChannelModel> canais = linktor.channels.list();
Channel.ChannelModel canal = linktor.channels.get(channelId);
```

</TabItem>
<TabItem value="csharp" label="C#">

```csharp
var canais = await Api(HttpMethod.Get, "/channels");            // array JSON
var canal = await Api(HttpMethod.Get, $"/channels/{channelId}");

foreach (var c in canais.EnumerateArray())
{
    Console.WriteLine($"{c.GetProperty("id")} {c.GetProperty("type")} {c.GetProperty("connection_status")}");
}
```

</TabItem>
<TabItem value="php" label="PHP">

```php
$canais = api('GET', '/channels');
$canal  = api('GET', "/channels/{$channelId}");

foreach ($canais as $c) {
    echo "{$c['id']} {$c['type']} {$c['connection_status']}" . PHP_EOL;
}
```

</TabItem>
<TabItem value="rust" label="Rust">

```rust
let canais = client.channels().list(None).await?;
let canal = client.channels().get(&channel_id).await?;
```

</TabItem>
</Tabs>

`GET /channels` devolve um **array simples** em `data`, sem `meta`, do mais recente para o mais
antigo:

```json
{
  "success": true,
  "data": [
    { "id": "CHANNEL_ID", "type": "telegram", "name": "Bot de atendimento", "enabled": true, "connection_status": "connected", "environment": "production", "…": "…" }
  ]
}
```

:::note Limites da listagem
A rota não tem paginação nem filtros: devolve até **100** canais e **ignora qualquer parâmetro de
query**. Os SDKs aceitam `type`, `status` e `search` em `list`, mas o servidor não os aplica — filtre
no seu código.
:::

Canal inexistente ou de outro tenant devolve `404` `CHANNEL_NOT_FOUND`; os dois casos não são
distinguíveis, de propósito.

## Atualizar

`PUT /channels/:id` é **parcial**: campo ausente não é tocado. `type` e `environment` são imutáveis.

| Campo | Comportamento |
|---|---|
| `name`, `identifier`, `webhook_url` | Substituem o valor quando presentes |
| `config` | **Substitui o mapa inteiro** das chaves não secretas — envie todas as que quer manter. Chaves sensíveis em branco ou `"__redacted__"` preservam o valor guardado |
| `credentials` | **Mescla**: valor não vazio sobrescreve, chave nova é adicionada, valor vazio mantém o guardado |
| `environment` | Só aceito se igual ao atual; valor diferente devolve `400` (`channel environment is immutable`) |

:::warning `config` é substituído por inteiro
Enviar `config` com uma chave só **apaga as demais** chaves não secretas. Apenas `credentials` é
mesclado. Para mudar um item de `config`, leia o canal, altere o `config` devolvido e mande de volta
como veio — os `"__redacted__"` não apagam nada. Os comentários dos SDKs dizem que `update` "reusa o
corpo do create"; o comportamento real é o descrito aqui.
:::

<Tabs groupId="linguagem">
<TabItem value="curl" label="cURL">

```bash
curl -X PUT "$LINKTOR_URL/api/v1/channels/$CHANNEL_ID" \
  -H "X-API-Key: $LINKTOR_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Bot de vendas",
    "webhook_url": "https://app.suaempresa.com/webhooks/linktor-v2",
    "credentials": { "webhook_secret": "novo-segredo" }
  }'
```

</TabItem>
<TabItem value="typescript" label="TypeScript">

```typescript
const canal = await linktor.channels.update(channelId, {
  name: 'Bot de vendas',
  webhook_url: 'https://app.suaempresa.com/webhooks/linktor-v2',
  credentials: { webhook_secret: 'novo-segredo' },
});
```

</TabItem>
<TabItem value="python" label="Python">

```python
canal = api("PUT", f"/channels/{channel_id}", {
    "name": "Bot de vendas",
    "webhook_url": "https://app.suaempresa.com/webhooks/linktor-v2",
    "credentials": {"webhook_secret": "novo-segredo"},
})
```

</TabItem>
<TabItem value="go" label="Go">

```go
canal, err := client.Channels.Update(ctx, channelID, &types.UpdateChannelInput{
	Name:        "Bot de vendas",
	WebhookURL:  "https://app.suaempresa.com/webhooks/linktor-v2",
	Credentials: map[string]string{"webhook_secret": "novo-segredo"},
})
```

</TabItem>
<TabItem value="java" label="Java">

```java
Channel.ChannelModel canal = linktor.channels.update(channelId,
    Channel.UpdateChannelInput.builder()
        .name("Bot de vendas")
        .webhookUrl("https://app.suaempresa.com/webhooks/linktor-v2")
        .credentials(Map.of("webhook_secret", "novo-segredo"))
        .build());
```

</TabItem>
<TabItem value="csharp" label="C#">

```csharp
var canal = await Api(HttpMethod.Put, $"/channels/{channelId}", new
{
    name = "Bot de vendas",
    webhook_url = "https://app.suaempresa.com/webhooks/linktor-v2",
    credentials = new Dictionary<string, string> { ["webhook_secret"] = "novo-segredo" },
});
```

</TabItem>
<TabItem value="php" label="PHP">

```php
$canal = api('PUT', "/channels/{$channelId}", [
    'name'        => 'Bot de vendas',
    'webhook_url' => 'https://app.suaempresa.com/webhooks/linktor-v2',
    'credentials' => ['webhook_secret' => 'novo-segredo'],
]);
```

</TabItem>
<TabItem value="rust" label="Rust">

```rust
let canal = client
    .channels()
    .update(
        &channel_id,
        UpdateChannelInput {
            name: Some("Bot de vendas".into()),
            webhook_url: Some("https://app.suaempresa.com/webhooks/linktor-v2".into()),
            credentials: Some(HashMap::from([(
                "webhook_secret".to_string(),
                "novo-segredo".to_string(),
            )])),
            ..Default::default()
        },
    )
    .await?;
```

</TabItem>
</Tabs>

Resposta `200` com o canal atualizado.

## Conectar {#conectar}

`POST /channels/:id/connect` não recebe corpo. O efeito depende do tipo:

- **WhatsApp não oficial** (`whatsapp`): inicia o pareamento e devolve um QR code — ou restabelece a
  sessão já pareada, sem QR.
- **Demais tipos**: executa a preparação do canal quando existe (por exemplo, registrar o webhook no
  provedor) e marca `connection_status: "connected"`. Se a preparação falha, `400` com
  `failed to connect channel: …`.

A resposta é um **resultado de conexão**, não um canal:

| Campo | Descrição |
|---|---|
| `channel` | O canal, com o `connection_status` atualizado |
| `qr_code` | Conteúdo do QR a renderizar como imagem (não é uma imagem pronta) |
| `expires_in` | Validade do QR ou do código de pareamento, em segundos |
| `pair_code` | Código de pareamento (resposta de `/pair`) |
| `passkey_required` | `true` quando a conta exige passkey em vez de QR |
| `passkey_challenge` | Desafio WebAuthn a assinar |

### Por QR code, com polling

1. Chame `connect`. Se vier `qr_code`, gere a imagem do QR com esse texto e mostre ao usuário.
2. O usuário escaneia em **WhatsApp → Dispositivos conectados → Conectar um dispositivo**.
3. Consulte `GET /channels/:id` a cada poucos segundos: `connection_status` passa de `connecting`
   para `connected`.
4. Passou de `expires_in` segundos sem conectar? Chame `connect` de novo para obter um QR novo.

Se `connect` voltar **sem** `qr_code` e com `channel.connection_status: "connected"`, a sessão já
estava pareada e foi retomada.

```json
{
  "success": true,
  "data": {
    "channel": { "id": "CHANNEL_ID", "type": "whatsapp", "connection_status": "connecting", "…": "…" },
    "qr_code": "2@Hk3…,Zx9…,Qw1…",
    "expires_in": 60
  }
}
```

<Tabs groupId="linguagem">
<TabItem value="curl" label="cURL">

```bash
# 1) Pede o QR
curl -s -X POST "$LINKTOR_URL/api/v1/channels/$CHANNEL_ID/connect" \
  -H "X-API-Key: $LINKTOR_API_KEY" | jq -r '.data.qr_code' | qrencode -t ANSIUTF8

# 2) Aguarda a leitura
until [ "$(curl -s "$LINKTOR_URL/api/v1/channels/$CHANNEL_ID" \
  -H "X-API-Key: $LINKTOR_API_KEY" | jq -r '.data.connection_status')" = "connected" ]; do
  sleep 3
done
echo "Canal conectado"
```

</TabItem>
<TabItem value="typescript" label="TypeScript">

```typescript
import qrcode from 'qrcode-terminal';

const dormir = (ms: number) => new Promise((r) => setTimeout(r, ms));

async function conectarPorQR(channelId: string) {
  for (let tentativa = 0; tentativa < 5; tentativa++) {
    const res = await linktor.channels.connect(channelId);

    if (res.passkey_required) throw new Error('Conta exige passkey');
    if (!res.qr_code) return res.channel; // sessão já pareada

    qrcode.generate(res.qr_code, { small: true });

    const limite = Date.now() + (res.expires_in ?? 60) * 1000;
    while (Date.now() < limite) {
      await dormir(3000);
      const canal = await linktor.channels.get(channelId);
      if (canal.connection_status === 'connected') return canal;
    }
    // QR expirou: o laço pede outro
  }
  throw new Error('QR não foi lido a tempo');
}
```

</TabItem>
<TabItem value="python" label="Python">

```python
import time
import qrcode  # pip install qrcode


def conectar_por_qr(channel_id: str) -> dict:
    for _ in range(5):
        res = api("POST", f"/channels/{channel_id}/connect")

        if res.get("passkey_required"):
            raise RuntimeError("Conta exige passkey")
        if not res.get("qr_code"):
            return res["channel"]  # sessão já pareada

        qr = qrcode.QRCode()
        qr.add_data(res["qr_code"])
        qr.print_ascii()

        limite = time.time() + (res.get("expires_in") or 60)
        while time.time() < limite:
            time.sleep(3)
            canal = api("GET", f"/channels/{channel_id}")
            if canal["connection_status"] == "connected":
                return canal
        # QR expirou: o laço pede outro

    raise TimeoutError("QR não foi lido a tempo")
```

</TabItem>
<TabItem value="go" label="Go">

```go
func conectarPorQR(ctx context.Context, client *linktor.Client, channelID string) (*types.Channel, error) {
	for tentativa := 0; tentativa < 5; tentativa++ {
		res, err := client.Channels.Connect(ctx, channelID)
		if err != nil {
			return nil, err
		}
		if res.PasskeyRequired {
			return nil, errors.New("conta exige passkey")
		}
		if res.QRCode == "" {
			return res.Channel, nil // sessão já pareada
		}

		// github.com/mdp/qrterminal/v3
		qrterminal.Generate(res.QRCode, qrterminal.L, os.Stdout)

		validade := res.ExpiresIn
		if validade <= 0 {
			validade = 60
		}
		limite := time.Now().Add(time.Duration(validade) * time.Second)
		for time.Now().Before(limite) {
			time.Sleep(3 * time.Second)
			canal, err := client.Channels.Get(ctx, channelID)
			if err != nil {
				return nil, err
			}
			if canal.ConnectionStatus == types.ConnectionStatusConnected {
				return canal, nil
			}
		}
		// QR expirou: o laço pede outro
	}
	return nil, errors.New("QR não foi lido a tempo")
}
```

</TabItem>
<TabItem value="java" label="Java">

```java
static Channel.ChannelModel conectarPorQR(LinktorClient linktor, String channelId) throws Exception {
    for (int tentativa = 0; tentativa < 5; tentativa++) {
        Channel.ConnectResult res = linktor.channels.connect(channelId);

        if (Boolean.TRUE.equals(res.getPasskeyRequired())) {
            throw new IllegalStateException("Conta exige passkey");
        }
        if (res.getQrCode() == null) {
            return res.getChannel(); // sessão já pareada
        }

        // Renderize res.getQrCode() como imagem (por exemplo com ZXing) e mostre ao usuário
        mostrarQr(res.getQrCode());

        int validade = res.getExpiresIn() != null ? res.getExpiresIn() : 60;
        long limite = System.currentTimeMillis() + validade * 1000L;
        while (System.currentTimeMillis() < limite) {
            Thread.sleep(3000);
            Channel.ChannelModel canal = linktor.channels.get(channelId);
            if (canal.getConnectionStatus() == Channel.ConnectionStatus.CONNECTED) {
                return canal;
            }
        }
        // QR expirou: o laço pede outro
    }
    throw new IllegalStateException("QR não foi lido a tempo");
}
```

</TabItem>
<TabItem value="csharp" label="C#">

```csharp
async Task<JsonElement> ConectarPorQrAsync(string channelId)
{
    for (var tentativa = 0; tentativa < 5; tentativa++)
    {
        var res = await Api(HttpMethod.Post, $"/channels/{channelId}/connect");

        if (res.TryGetProperty("passkey_required", out var pk) && pk.ValueKind == JsonValueKind.True)
        {
            throw new InvalidOperationException("Conta exige passkey");
        }
        if (!res.TryGetProperty("qr_code", out var qr) || string.IsNullOrEmpty(qr.GetString()))
        {
            return res.GetProperty("channel"); // sessão já pareada
        }

        // Renderize o texto do QR como imagem (por exemplo com QRCoder) e mostre ao usuário
        MostrarQr(qr.GetString()!);

        var validade = res.TryGetProperty("expires_in", out var exp) ? exp.GetInt32() : 60;
        var limite = DateTime.UtcNow.AddSeconds(validade);
        while (DateTime.UtcNow < limite)
        {
            await Task.Delay(3000);
            var canal = await Api(HttpMethod.Get, $"/channels/{channelId}");
            if (canal.GetProperty("connection_status").GetString() == "connected") return canal;
        }
        // QR expirou: o laço pede outro
    }
    throw new TimeoutException("QR não foi lido a tempo");
}
```

</TabItem>
<TabItem value="php" label="PHP">

```php
function conectarPorQr(string $channelId): array
{
    for ($tentativa = 0; $tentativa < 5; $tentativa++) {
        $res = api('POST', "/channels/{$channelId}/connect");

        if (!empty($res['passkey_required'])) {
            throw new RuntimeException('Conta exige passkey');
        }
        if (empty($res['qr_code'])) {
            return $res['channel']; // sessão já pareada
        }

        // Renderize $res['qr_code'] como imagem (por exemplo com endroid/qr-code) e mostre ao usuário
        mostrarQr($res['qr_code']);

        $limite = time() + ($res['expires_in'] ?? 60);
        while (time() < $limite) {
            sleep(3);
            $canal = api('GET', "/channels/{$channelId}");
            if ($canal['connection_status'] === 'connected') {
                return $canal;
            }
        }
        // QR expirou: o laço pede outro
    }
    throw new RuntimeException('QR não foi lido a tempo');
}
```

</TabItem>
<TabItem value="rust" label="Rust">

```rust
use std::time::{Duration, Instant};

async fn conectar_por_qr(
    client: &LinktorClient,
    channel_id: &str,
) -> Result<Option<Channel>, Box<dyn std::error::Error>> {
    for _ in 0..5 {
        let res = client.channels().connect(channel_id).await?;

        if res.passkey_required == Some(true) {
            return Err("conta exige passkey".into());
        }
        let Some(qr) = res.qr_code else {
            return Ok(res.channel); // sessão já pareada
        };

        // crate qr2term
        qr2term::print_qr(&qr)?;

        let limite = Instant::now() + Duration::from_secs(res.expires_in.unwrap_or(60) as u64);
        while Instant::now() < limite {
            tokio::time::sleep(Duration::from_secs(3)).await;
            let canal = client.channels().get(channel_id).await?;
            if canal.connection_status == ConnectionStatus::Connected {
                return Ok(Some(canal));
            }
        }
        // QR expirou: o laço pede outro
    }
    Err("QR não foi lido a tempo".into())
}
```

</TabItem>
</Tabs>

O que pode dar errado no `connect` do WhatsApp não oficial:

| Sintoma | Causa |
|---|---|
| Erro `timeout waiting for QR code` | O WhatsApp não entregou o QR em 30 s. Tente de novo |
| Erro `CHANNEL_DISCONNECTED` | A sessão está pareada, mas o WhatsApp recusou a autenticação (por exemplo, versão do cliente desatualizada). Um novo QR não resolve; veja [WhatsApp não oficial](../canais/whatsapp-nao-oficial.md) |
| `passkey_required: true` | A conta só pareia com passkey — veja [abaixo](#passkey) |

:::note
O polling de `GET /channels/:id` é a forma suportada de saber que o pareamento terminou: a mudança
de estado do canal não é entregue como evento no webhook da sua aplicação.
:::

### Por código de pareamento

Alternativa ao QR para quando o usuário não consegue apontar a câmera: `POST /channels/:id/pair`
devolve um código que ele digita em **Dispositivos conectados → Conectar com número de telefone**.
Só vale para WhatsApp não oficial; em outro tipo, `400` (`pair code is only available for WhatsApp
channels`).

<Tabs groupId="linguagem">
<TabItem value="curl" label="cURL">

```bash
curl -X POST "$LINKTOR_URL/api/v1/channels/$CHANNEL_ID/pair" \
  -H "X-API-Key: $LINKTOR_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{ "phone_number": "+5511999999999" }'
```

</TabItem>
<TabItem value="typescript" label="TypeScript">

```typescript
const res = await linktor.channels.requestPairCode(channelId, '+5511999999999');
console.log(`Digite ${res.pair_code} no WhatsApp (vale ${res.expires_in}s)`);
```

</TabItem>
<TabItem value="python" label="Python">

```python
res = api("POST", f"/channels/{channel_id}/pair", {"phone_number": "+5511999999999"})
print(f"Digite {res['pair_code']} no WhatsApp (vale {res['expires_in']}s)")
```

</TabItem>
<TabItem value="go" label="Go">

```go
res, err := client.Channels.RequestPairCode(ctx, channelID, "+5511999999999")
if err != nil {
	log.Fatal(err)
}
fmt.Printf("Digite %s no WhatsApp (vale %ds)\n", res.PairCode, res.ExpiresIn)
```

</TabItem>
<TabItem value="java" label="Java">

```java
Channel.ConnectResult res = linktor.channels.requestPairCode(channelId, "+5511999999999");
System.out.printf("Digite %s no WhatsApp (vale %ds)%n", res.getPairCode(), res.getExpiresIn());
```

</TabItem>
<TabItem value="csharp" label="C#">

```csharp
var res = await Api(HttpMethod.Post, $"/channels/{channelId}/pair", new { phone_number = "+5511999999999" });
Console.WriteLine($"Digite {res.GetProperty("pair_code")} no WhatsApp (vale {res.GetProperty("expires_in")}s)");
```

</TabItem>
<TabItem value="php" label="PHP">

```php
$res = api('POST', "/channels/{$channelId}/pair", ['phone_number' => '+5511999999999']);
echo "Digite {$res['pair_code']} no WhatsApp (vale {$res['expires_in']}s)" . PHP_EOL;
```

</TabItem>
<TabItem value="rust" label="Rust">

```rust
let res = client
    .channels()
    .request_pair_code(&channel_id, "+5511999999999")
    .await?;
println!(
    "Digite {} no WhatsApp (vale {}s)",
    res.pair_code.unwrap_or_default(),
    res.expires_in.unwrap_or_default()
);
```

</TabItem>
</Tabs>

```json
{
  "success": true,
  "data": {
    "channel": { "id": "CHANNEL_ID", "type": "whatsapp", "connection_status": "connecting", "…": "…" },
    "pair_code": "ABCD-1234",
    "expires_in": 300
  }
}
```

`phone_number` é o número da conta de WhatsApp que será pareada. Depois de exibir o código, faça o
mesmo polling de `GET /channels/:id` até `connected`.

### Conta com passkey {#passkey}

Algumas contas de WhatsApp não aceitam QR: o `connect` devolve `passkey_required: true` e um
`passkey_challenge` (desafio WebAuthn). O **dono da conta** precisa assinar esse desafio no navegador
dele, e a assinatura é entregue assim:

```bash
curl -X POST "$LINKTOR_URL/api/v1/channels/$CHANNEL_ID/passkey/response" \
  -H "X-API-Key: $LINKTOR_API_KEY" \
  -H "Content-Type: application/json" \
  --data-binary @assertion.json
```

```json
{ "success": true, "data": { "status": "accepted" } }
```

- O corpo é a asserção WebAuthn **exatamente como o navegador a produziu**. Não reserialize o JSON:
  os campos em base64url são conferidos byte a byte. Tamanho máximo de 64 KB.
- Só funciona enquanto a sessão aberta pelo `connect` está viva; fora disso, `409`
  (`no live WhatsApp session for this channel; start the connection again`).
- `accepted` significa que a asserção foi entregue. A confirmação é, como no QR,
  `connection_status: "connected"` em `GET /channels/:id`.

:::note
Assinar o desafio depende de um componente no navegador do dono da conta (a extensão de passkey do
Linktor, usada pelo painel). Não há suporte a esse passo nos SDKs; se você está embutindo o
pareamento na sua aplicação, o caminho suportado de ponta a ponta é o QR ou o código de pareamento.
:::

## Habilitar e desabilitar

```bash
curl -X PUT "$LINKTOR_URL/api/v1/channels/$CHANNEL_ID/enabled" \
  -H "X-API-Key: $LINKTOR_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{ "enabled": false }'
```

Resposta `200` com o canal. Um canal desabilitado recusa envios e interrompe processos de
recebimento que o Linktor mantém por canal (como a leitura da caixa IMAP); a conexão com o provedor
não é desfeita — para isso use `disconnect`.

:::warning Envie `enabled` explicitamente
Um corpo sem o campo (`{}`) é lido como `false` e **desabilita** o canal.
:::

`PUT /channels/:id/status` com `{"status": "active"}` ou `{"status": "inactive"}` faz exatamente o
mesmo e existe por compatibilidade; prefira `/enabled`.

Esta rota não existe nos SDKs; use HTTP.

## Desconectar

```bash
curl -X POST "$LINKTOR_URL/api/v1/channels/$CHANNEL_ID/disconnect" \
  -H "X-API-Key: $LINKTOR_API_KEY"
```

```json
{ "success": true, "data": { "message": "Channel disconnected" } }
```

Encerra a conexão ativa e grava `connection_status: "disconnected"`. O canal continua existindo, com
configuração e histórico; chame `connect` para voltar.

:::warning `disconnect` nos SDKs
Os SDKs declaram que `channels.disconnect` devolve um canal, mas a API devolve só a mensagem acima.
O resultado depende da linguagem: em Python e Rust a chamada **lança erro de desserialização depois
de o canal já ter sido desconectado**; nas demais, o objeto devolvido vem vazio. Em qualquer
linguagem, chame esta rota por HTTP e leia o estado depois em `GET /channels/:id`.
:::

## Remover

<Tabs groupId="linguagem">
<TabItem value="curl" label="cURL">

```bash
curl -X DELETE "$LINKTOR_URL/api/v1/channels/$CHANNEL_ID" \
  -H "X-API-Key: $LINKTOR_API_KEY"
```

</TabItem>
<TabItem value="typescript" label="TypeScript">

```typescript
await linktor.channels.delete(channelId);
```

</TabItem>
<TabItem value="python" label="Python">

```python
linktor.channels.delete(channel_id)
```

</TabItem>
<TabItem value="go" label="Go">

```go
err := client.Channels.Delete(ctx, channelID)
```

</TabItem>
<TabItem value="java" label="Java">

```java
linktor.channels.delete(channelId);
```

</TabItem>
<TabItem value="csharp" label="C#">

```csharp
await Api(HttpMethod.Delete, $"/channels/{channelId}"); // 204, sem corpo
```

</TabItem>
<TabItem value="php" label="PHP">

```php
api('DELETE', "/channels/{$channelId}"); // 204, sem corpo
```

</TabItem>
<TabItem value="rust" label="Rust">

```rust
client.channels().delete(&channel_id).await?;
```

</TabItem>
</Tabs>

Resposta `204` sem corpo. A remoção é definitiva. Desconecte antes de remover um canal de WhatsApp
não oficial, para encerrar a sessão no aparelho.

## Dados do aparelho pareado (WhatsApp não oficial)

### Agenda

`GET /channels/:id/contacts` lê a agenda do aparelho pareado. Exige sessão ativa.

```bash
curl "$LINKTOR_URL/api/v1/channels/$CHANNEL_ID/contacts" \
  -H "X-API-Key: $LINKTOR_API_KEY"
```

```json
{
  "success": true,
  "data": {
    "contacts": [
      { "jid": "5511999999999@s.whatsapp.net", "phone": "5511999999999", "fullName": "Maria Souza", "pushName": "Maria" },
      { "jid": "123456789012345@lid", "pushName": "João" }
    ],
    "total": 2,
    "withoutPhone": 1
  }
}
```

- Os campos desta resposta são **camelCase** (`fullName`, `pushName`, `businessName`,
  `withoutPhone`), diferente do resto da API.
- `phone` ausente significa que o WhatsApp identificou o contato só por LID e o número não é
  conhecido — é diferente de "contato não encontrado". `withoutPhone` conta esses casos.
- Canal sem sessão ativa, ou que não é WhatsApp não oficial: `409 CONFLICT`.

### Foto do contato

`GET /channels/:id/contacts/:contact/avatar` devolve **os bytes da imagem** (não JSON). `:contact` é
o telefone só com dígitos, um `<numero>@s.whatsapp.net` ou um `<id>@lid`. A API key precisa de
`channels:read` **e** `contacts:read`.

```bash
curl "$LINKTOR_URL/api/v1/channels/$CHANNEL_ID/contacts/5511999999999/avatar" \
  -H "X-API-Key: $LINKTOR_API_KEY" \
  -o foto.jpg -D -
```

| Resposta | Significado |
|---|---|
| `200` | Imagem no corpo; cabeçalhos `X-Picture-Id` e `ETag` trazem o identificador da foto |
| `304` | Você enviou `If-None-Match: "<picture id>"` e a foto não mudou |
| `404` | Sem foto (`reason: "not_set"`), foto oculta pela privacidade do contato (`reason: "hidden"`), ou canal inexistente |
| `409` | Canal sem sessão ativa |
| `429` | Limite por canal; respeite o `Retry-After` |
| `501` | O tipo de canal não oferece foto de contato |

Guarde o `X-Picture-Id` e mande-o em `If-None-Match` nas consultas seguintes, para não baixar a mesma
imagem de novo.

## Rotas de operação

| Rota | Uso |
|---|---|
| `GET /channels/:id/bridge-health` | Estado do Linktor Bridge do canal. O mesmo dado vem em `bridge_health` no `GET /channels/:id` |
| `POST /channels/:id/bridge-token` | Gera o token com que o Bridge se autentica. Devolve `{"bridge_token": "…"}` **uma única vez**. Só `admin`/`owner` |
| `POST /channels/reencrypt` | Depois de uma rotação da chave de criptografia, regrava as credenciais de todos os canais do tenant. Devolve `{"reencrypted": <n>}`. Só `admin`/`owner` |
| `GET /channels/:id/coexistence-status`, `POST /channels/:id/subscribe-echoes` | Coexistência do WhatsApp oficial com o app WhatsApp Business; veja [WhatsApp oficial](../canais/whatsapp-oficial.md) |

## O que os SDKs cobrem

| Operação | Rota | SDK |
|---|---|---|
| Listar, obter, criar, atualizar, remover | `/channels`, `/channels/:id` | Métodos existem nos 7 SDKs; utilizáveis hoje em TypeScript, Go, Java e Rust (Python só para tipos do enum; C# e PHP: use HTTP) |
| Conectar (QR) | `POST /channels/:id/connect` | Idem |
| Código de pareamento | `POST /channels/:id/pair` | Idem |
| Desconectar | `POST /channels/:id/disconnect` | Existe, mas o tipo de retorno não bate com a API (ver aviso acima) |
| Testar credenciais, habilitar/desabilitar, passkey, agenda, foto | — | Não coberto: use HTTP |

:::note
O SDK TypeScript expõe `channels.getCapabilities(id)`, que chama `/channels/:id/capabilities`. Essa
rota não existe no servidor; a chamada devolve `404`.
:::

## Erros desta página

| Status | `error.code` | Quando |
|---|---|---|
| `400` | `VALIDATION_ERROR` | Corpo inválido, tipo não habilitado, regra de ambiente, falha na preparação do `connect`, teste de credencial reprovado |
| `401` | `UNAUTHORIZED` | Credencial ausente ou inválida |
| `403` | `FORBIDDEN` | API key sem `channels:read`/`channels:write`, ou rota restrita a `admin`/`owner` |
| `404` | `CHANNEL_NOT_FOUND` | O canal não existe **ou** é de outro tenant |
| `409` | `CONFLICT` | Operação exige sessão de WhatsApp ativa e não há |

O formato completo do erro está em [Erros e paginação](./erros-e-paginacao.md).
