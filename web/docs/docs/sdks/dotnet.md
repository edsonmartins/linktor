---
sidebar_position: 6
title: "C# / .NET"
description: SDK .NET do Linktor — o que compila e funciona hoje, como instalar a partir do repositório, canais por QR e pair code, envio de mensagem com HttpClient e verificação de webhook em ASP.NET Core.
---

# C# / .NET

O SDK .NET fica em `sdks/dotnet` (pacote `Linktor`, namespace raiz `Linktor`). Hoje ele é confiável
para **gerenciar canais** (criar, atualizar, conectar por QR ou pair code, remover). Para o resto —
inclusive **enviar mensagem** — use `System.Net.Http.HttpClient` direto na API; esta página mostra os
dois caminhos.

:::warning O projeto não compila como está no repositório
`src/Linktor/Resources/*.cs` referencia oito tipos que não existem em lugar nenhum do fonte:
`MessageMedia`, `QuickReply`, `BotConfig`, `AgentConfig`, `AgentTool`, `TokenUsage`,
`KnowledgeBaseConfig` e `FlowSettings`. O `dotnet build` falha com `CS0246` até que eles sejam
declarados. Você tem dois caminhos:

1. **HTTP nativo** (funciona sem o SDK): veja [Enviar mensagem](#enviar-mensagem) e
   [Verificar a assinatura do webhook](#webhook).
2. **Usar o SDK a partir de um clone**, acrescentando o arquivo de tipos faltantes descrito em
   [Instalação](#instalacao). Depois disso, o recurso `Channels` funciona por inteiro; os demais têm
   as restrições listadas em [Limitações conhecidas](#limitacoes).

Esta página foi conferida por leitura do fonte do SDK e do backend; os trechos não foram compilados.
:::

| Recurso | No SDK | Observação |
|---|---|---|
| Canais | sim | CRUD, QR, pair code, desconectar |
| Mensagens — envio direto (`POST /messages/send`) | não | Use `HttpClient` |
| Mensagens — para uma conversa | não | O método existe, mas manda o corpo errado |
| Conversas | parcial | Só apagar e listar mensagens (sem o conteúdo) |
| Contatos | parcial | Listar, obter, criar (campos simples), apagar |
| Bots | parcial | Só obter e apagar |
| IA | não | Todas as rotas chamadas não existem |
| Base de conhecimento | parcial | Só listar, obter e apagar a base |
| Fluxos | parcial | Só listar, obter e apagar |
| Analytics | não | Não há recurso |
| VRE | parcial | Só `RenderAsync` |
| Verificação de webhook | não compatível | Assina só o corpo; o servidor assina `timestamp.corpo` |
| WebSocket | não | Não há cliente |
| Autenticação por API key | sim | Header `X-API-Key` |

## Instalação {#instalacao}

Coordenadas do `Linktor.csproj`: `PackageId` **`Linktor`**, versão **`1.0.0`**.

```bash
dotnet add package Linktor --version 1.0.0
```

:::note
Não é possível confirmar daqui se o pacote está publicado no NuGet — e, como o fonte atual não
compila, um pacote gerado a partir dele não existiria. O caminho garantido é a referência de projeto
a partir do repositório.
:::

```bash
git clone https://github.com/edsonmartins/linktor.git
```

Crie, **no seu clone**, o arquivo `sdks/dotnet/src/Linktor/Types/Missing.cs` com os tipos que faltam:

```csharp
namespace Linktor.Types;

public class MessageMedia { }
public class QuickReply { }
public class BotConfig { }
public class AgentConfig { }
public class AgentTool { }
public class TokenUsage { }
public class KnowledgeBaseConfig { }
public class FlowSettings { }
```

E referencie o projeto na sua aplicação:

```bash
dotnet add reference <clone>/sdks/dotnet/src/Linktor/Linktor.csproj
```

## Requisitos

- **.NET 8** (`TargetFramework` `net8.0`, C# 12).
- Dependências: `System.Net.Http.Json` 8.0.0 e `System.Text.Json` 8.0.4. Nenhuma outra.

## Criar o cliente

```csharp
using Linktor;
using Linktor.Types;

var client = new LinktorClient(new LinktorClientOptions
{
    BaseUrl = Environment.GetEnvironmentVariable("LINKTOR_URL")!.TrimEnd('/') + "/api/v1",
    ApiKey = Environment.GetEnvironmentVariable("LINKTOR_API_KEY"),
});
```

`BaseUrl` **precisa incluir `/api/v1`**: o SDK só remove a barra final e concatena o caminho do recurso
(`/channels`, `/contacts`…). O padrão `https://api.linktor.io` não serve para uma instalação própria.

| Opção | Tipo | Padrão | O que faz |
|---|---|---|---|
| `BaseUrl` | `string` | `https://api.linktor.io` | Base da API, com `/api/v1` |
| `ApiKey` | `string?` | `null` | Enviada em `X-API-Key` |
| `AccessToken` | `string?` | `null` | JWT de usuário, enviado em `Authorization: Bearer`; só é usado se `ApiKey` estiver vazia |
| `TimeoutSeconds` | `int` | `30` | Timeout do `HttpClient` interno |
| `MaxRetries` | `int` | `3` | Número total de tentativas em 429 e 5xx |

Não há opção para headers extras nem para injetar um `HttpClient` ou `HttpMessageHandler`. O cliente
cria um `HttpClient` próprio e não implementa `IDisposable`: crie **uma** instância e reutilize.

Os tipos ficam em três namespaces: `Linktor` (cliente e opções), `Linktor.Types` (modelos) e
`Linktor.Utils` (exceções e `WebhookVerifier`). `Linktor.Resources` declara classes de entrada com o
**mesmo nome** de classes de `Linktor.Types` (`CreateContactInput`, `SendMessageInput`,
`CreateBotInput`…). Se você importar os dois namespaces, o compilador acusa `CS0104`; os métodos
esperam a versão de `Linktor.Resources`, então escreva o nome completo, como nos exemplos de
[Contatos](#contatos).

## Autenticação

- **API key** (integração servidor a servidor): preencha `ApiKey`. O SDK envia `X-API-Key: lk_...`,
  que é o header que o servidor aceita. A chave precisa dos scopes das rotas usadas — veja
  [Chaves de API](../painel/chaves-de-api.md) e [Autenticação](../api/autenticacao.md).
- **JWT de usuário**: preencha `AccessToken` com um token já obtido. `client.Auth.LoginAsync` não
  serve para isso: ele lê `accessToken` da resposta, mas o servidor devolve `access_token`, e o token
  fica vazio. Faça `POST /auth/login` com `HttpClient` (corpo `email` e `password`), leia
  `data.access_token` e passe o valor em `AccessToken`. O token não pode ser trocado depois: quando
  expirar, renove em `POST /auth/refresh` (corpo `refresh_token`) e crie outro `LinktorClient`.

## Canais

`client.Channels` é o recurso alinhado com o backend: os modelos usam `[JsonPropertyName]` em
snake_case e há testes em `tests/Linktor.Tests/ChannelsResourceTests.cs`.

```csharp
Task<List<Channel>> ListAsync(ListChannelsParams? parameters = null, CancellationToken ct = default)
Task<Channel> GetAsync(string id, CancellationToken ct = default)
Task<Channel> CreateAsync(CreateChannelInput input, CancellationToken ct = default)
Task<Channel> UpdateAsync(string id, UpdateChannelInput input, CancellationToken ct = default)
Task DeleteAsync(string id, CancellationToken ct = default)
Task<ConnectResult> ConnectAsync(string id, CancellationToken ct = default)
Task<ConnectResult> RequestPairCodeAsync(string id, string phoneNumber, CancellationToken ct = default)
Task<Channel> DisconnectAsync(string id, CancellationToken ct = default)
```

### Criar um canal WhatsApp não-oficial e conectar por QR

O QR expira: `ConnectResult.ExpiresIn` diz em quantos segundos. Enquanto o canal não conecta, chame
`ConnectAsync` de novo para obter um código novo.

```csharp
var channel = await client.Channels.CreateAsync(new CreateChannelInput
{
    Type = ChannelType.Whatsapp, // "whatsapp" (não oficial)
    Name = "WhatsApp Vendas",
});

var connected = false;
while (!connected)
{
    var result = await client.Channels.ConnectAsync(channel.Id);
    if (result.PasskeyRequired == true)
        throw new InvalidOperationException("Conta protegida por passkey: conecte pelo painel.");

    if (!string.IsNullOrEmpty(result.QrCode))
        Console.WriteLine(result.QrCode); // conteúdo do QR: desenhe com a biblioteca de sua escolha

    var deadline = DateTime.UtcNow.AddSeconds(result.ExpiresIn ?? 60);
    while (!connected && DateTime.UtcNow < deadline)
    {
        await Task.Delay(TimeSpan.FromSeconds(3));
        var current = await client.Channels.GetAsync(channel.Id);
        connected = current.ConnectionStatus == ConnectionStatus.Connected;
    }
}
```

`ConnectResult` traz `Channel`, `QrCode`, `ExpiresIn`, `PairCode`, `PasskeyRequired` e
`PasskeyChallenge` (`JsonElement?`). Detalhes do canal em
[WhatsApp não-oficial](../canais/whatsapp-nao-oficial.md).

### Conectar por pair code

```csharp
var pair = await client.Channels.RequestPairCodeAsync(channel.Id, "+5511999999999");
Console.WriteLine(pair.PairCode); // digite no WhatsApp do aparelho
```

### Atualizar, apontar o webhook e remover

`UpdateAsync` usa `PUT` e só altera o que for enviado. `Credentials` é mesclado chave a chave com o
que já está gravado, então mandar apenas `webhook_secret` não apaga os outros segredos.

```csharp
await client.Channels.UpdateAsync(channel.Id, new UpdateChannelInput
{
    Name = "WhatsApp Vendas SP",
    WebhookUrl = "https://app.suaempresa.com/webhooks/linktor",
    Credentials = new Dictionary<string, string> { ["webhook_secret"] = "um-segredo-longo" },
    Config = new Dictionary<string, string> { ["webhook_events"] = "message.received,message.failed" },
});

await client.Channels.DisconnectAsync(channel.Id);
await client.Channels.DeleteAsync(channel.Id);
```

Sem `webhook_secret` o servidor não entrega eventos para a `webhook_url`. `webhook_events` vazio
significa todos os eventos.

:::warning Particularidades de `Channels`
- `ListChannelsParams` (`Type`, `Status`, `Search`) vai na query string, mas o servidor ignora:
  `ListAsync` sempre devolve todos os canais do tenant. Filtre em memória.
- `CreateChannelInput` não tem `environment`; pelo SDK o canal nasce sempre como `production`.
- `DisconnectAsync` desconecta, mas o `Channel` devolvido vem vazio (o servidor responde só uma
  mensagem). Chame `GetAsync` se precisar do estado.
- `Channel.Config` devolve valores secretos como `__redacted__`; `credentials` nunca volta.
:::

## Enviar mensagem {#enviar-mensagem}

O SDK não tem o envio direto (`POST /messages/send`), e `Conversations.SendMessageAsync` não funciona
(veja [Limitações conhecidas](#limitacoes)). Use `HttpClient`:

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
    metadata = new Dictionary<string, string> { ["pedido"] = "1234" }, // opcional
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

A chave precisa do scope `messages:send`. A resposta `202` significa "na fila"; o resultado da entrega
chega pelo webhook (`message.sent`, `message.delivered`, `message.failed`). Mídia, `metadata` e
correlação estão em [Enviar mensagens](../api/enviar-mensagens.md).

Para responder dentro de uma conversa existente, o mesmo `http` serve:

```csharp
await http.PostAsJsonAsync("conversations/CONVERSATION_ID/messages",
    new { content_type = "text", content = "Posso ajudar em mais alguma coisa?" });
```

## Conversas

Só dois métodos de `client.Conversations` têm resultado útil:

```csharp
Task DeleteAsync(string id, CancellationToken ct = default)
Task<PaginatedResponse<Message>> GetMessagesAsync(string conversationId,
    Linktor.Resources.ListMessagesParams? parameters = null, CancellationToken ct = default)
```

```csharp
var page = await client.Conversations.GetMessagesAsync("CONVERSATION_ID",
    new Linktor.Resources.ListMessagesParams { Limit = 50 });

foreach (var message in page.Data)
    Console.WriteLine($"{message.Id} {message.Status}");
```

:::warning O que se perde
`Message` é desserializado em camelCase e o servidor responde em snake_case. Chegam `Id`, `Status` e
`Metadata`; **o texto não chega** (`Text` fica `null`, pois o campo no wire é `content`), nem
`ConversationId`, `SenderType`, `CreatedAt` ou anexos. Dos parâmetros, só `Limit` tem efeito; a
paginação do servidor é pelo cursor `before`, que o SDK não envia, e `page.Pagination` vem zerado.
Para ler mensagens com conteúdo, use `GET /conversations/{id}/messages` com `HttpClient`.
:::

## Contatos {#contatos}

```csharp
Task<PaginatedResponse<Contact>> ListAsync(Linktor.Resources.ListContactsParams? parameters = null, CancellationToken ct = default)
Task<Contact> GetAsync(string id, CancellationToken ct = default)
Task<Contact> CreateAsync(Linktor.Resources.CreateContactInput input, CancellationToken ct = default)
Task DeleteAsync(string id, CancellationToken ct = default)
```

```csharp
var contact = await client.Contacts.CreateAsync(new Linktor.Resources.CreateContactInput
{
    Name = "Maria Souza",
    Email = "cliente@exemplo.com",
    Phone = "+5511999999999",
    Tags = new List<string> { "vip" },
});

var found = await client.Contacts.ListAsync(new Linktor.Resources.ListContactsParams { Search = "maria" });
foreach (var c in found.Data ?? new List<Contact>())
    Console.WriteLine($"{c.Id} {c.Name} {c.Phone}");

await client.Contacts.DeleteAsync(contact.Id);
```

:::warning O que se perde
- Na escrita, o servidor só recebe `Name`, `Email`, `Phone` e `Tags`. `AvatarUrl` e `CustomFields`
  saem como `avatarUrl` e `customFields` e são ignorados (o servidor espera `avatar_url` e
  `custom_fields`); `ExternalId` e `Metadata` não existem na API.
- Na leitura, chegam `Id`, `Name`, `Email`, `Phone` e `Tags`. `TenantId`, `Avatar`, `CustomFields`,
  `CreatedAt` e `UpdatedAt` ficam com o valor padrão, e as identidades do contato não são mapeadas.
- Na listagem, só `Search` tem efeito: `Limit`, `Offset`, `Cursor` e `Tag` são ignorados (o servidor
  usa `page` e `page_size`), então você recebe sempre a primeira página de 20. `Pagination` vem zerado.
:::

## Bots, base de conhecimento e fluxos

Nestes três recursos só a leitura e a remoção chamam rotas reais:

```csharp
// client.Bots, client.KnowledgeBases e client.Flows têm a mesma forma:
Task<PaginatedResponse<T>> ListAsync(...)   // T = Bot, KnowledgeBase ou Flow
Task<T> GetAsync(string id, CancellationToken ct = default)
Task DeleteAsync(string id, CancellationToken ct = default)
```

```csharp
var flows = await client.Flows.ListAsync();
foreach (var flow in flows.Data ?? new List<Flow>())
    Console.WriteLine($"{flow.Id} {flow.Name}");

await client.KnowledgeBases.DeleteAsync("KB_ID");
```

:::warning Leitura frágil
- Os parâmetros de listagem (`Limit`, `Offset`, `Status`) são ignorados; vem a primeira página.
- Só `Id`, `Name`, `Description` (base e fluxo) e `Config` do bot são confiáveis. Campos compostos
  (`TenantId`, `ChannelIds`, `CreatedAt`…) ficam vazios.
- `Flow.Status` vem sempre `Active`, mesmo com o fluxo inativo: o servidor manda `is_active`, que o
  modelo não lê. `Flow.Nodes` traz só `Id` e `Type` de cada nó.
- Os enums do SDK não cobrem todos os valores do servidor. Um bot `rule_based` ou em `training`, ou
  uma base `inactive` ou `syncing`, faz `ListAsync` lançar `System.Text.Json.JsonException` e
  `GetAsync` devolver um objeto vazio (`Id == ""`).
:::

Criar e atualizar bots, bases e fluxos, e tudo de IA (`/ai/complete`, `/ai/classify-intent`…), só por
`HttpClient`.

## VRE

`client.VRE.RenderAsync` funciona: o corpo é snake_case e a resposta (sem envelope) é lida
diretamente. As rotas `/vre/*` só existem se o serviço VRE estiver habilitado no servidor.

```csharp
Task<VRERenderResponse> RenderAsync(VRERenderRequest request, CancellationToken ct = default)
```

```csharp
var image = await client.VRE.RenderAsync(new Linktor.Resources.VRERenderRequest
{
    TemplateId = "menu_opcoes",
    Channel = "whatsapp",
    Data = new Dictionary<string, object> { ["titulo"] = "Como posso ajudar?" },
});

await File.WriteAllBytesAsync($"menu.{image.Format}", Convert.FromBase64String(image.ImageBase64));
```

Deixe `TenantId` vazio: se for preenchido com um valor diferente do tenant autenticado, o servidor
responde 403. `RenderTimeMs` chega em nanossegundos, apesar do nome. Os atalhos `RenderMenuAsync`,
`RenderProductCardAsync`, `RenderOrderStatusAsync`, `RenderProductListAsync`,
`RenderConfirmationAsync` e `RenderPixPaymentAsync` apenas montam um `VRERenderRequest` com um
`TemplateId` fixo e chamam `RenderAsync`.

## Verificar a assinatura do webhook {#webhook}

:::warning O verificador do SDK não é compatível
`WebhookVerifier.Verify`, `WebhookVerifier.ConstructEvent` e os atalhos estáticos de `LinktorClient`
leem os headers certos, mas calculam o HMAC **só sobre o corpo**. O servidor assina
`<timestamp> + "." + <corpo cru>`. Toda entrega real é rejeitada. Verifique manualmente.
:::

O contrato: `X-Linktor-Signature` é o HMAC-SHA256 em hexadecimal minúsculo, sem prefixo, de
`X-Linktor-Timestamp + "." + corpo`, com o `credentials.webhook_secret` do canal como chave. O
timestamp é epoch em segundos; recuse diferenças acima de 300 s.

Receptor mínimo em ASP.NET Core (`dotnet new web`, arquivo `Program.cs`):

```csharp
using System.Security.Cryptography;
using System.Text;
using System.Text.Json;

var builder = WebApplication.CreateBuilder(args);
var app = builder.Build();
var secret = Environment.GetEnvironmentVariable("LINKTOR_WEBHOOK_SECRET")!;

app.MapPost("/webhooks/linktor", async (HttpRequest request) =>
{
    // Corpo cru: não use [FromBody] nem ReadFromJsonAsync antes de verificar.
    using var buffer = new MemoryStream();
    await request.Body.CopyToAsync(buffer);
    var raw = buffer.ToArray();

    var signature = request.Headers["X-Linktor-Signature"].ToString();
    var timestamp = request.Headers["X-Linktor-Timestamp"].ToString();

    if (!long.TryParse(timestamp, out var sentAt) ||
        Math.Abs(DateTimeOffset.UtcNow.ToUnixTimeSeconds() - sentAt) > 300)
        return Results.Unauthorized();

    var prefix = Encoding.UTF8.GetBytes(timestamp + ".");
    var signed = new byte[prefix.Length + raw.Length];
    prefix.CopyTo(signed, 0);
    raw.CopyTo(signed, prefix.Length);

    var expected = Convert.ToHexString(
        HMACSHA256.HashData(Encoding.UTF8.GetBytes(secret), signed)).ToLowerInvariant();

    if (!CryptographicOperations.FixedTimeEquals(
            Encoding.UTF8.GetBytes(expected), Encoding.UTF8.GetBytes(signature)))
        return Results.Unauthorized();

    using var doc = JsonDocument.Parse(raw);
    var root = doc.RootElement;
    var type = root.GetProperty("type").GetString();

    if (type == "message.received")
    {
        var data = root.GetProperty("data");
        var content = data.GetProperty("message").GetProperty("content");
        var text = content.TryGetProperty("text", out var t) ? t.GetString() : null;
        Console.WriteLine($"{data.GetProperty("conversationId")}: {text}");
    }

    return Results.Ok();
});

app.Run();
```

O envelope do evento é camelCase (`id`, `type`, `timestamp`, `tenantId`, `environment`, `data`). Tipos
de evento e o formato de cada `data` estão em [Receber eventos](../api/receber-eventos.md).

## Tempo real (WebSocket)

O SDK não tem cliente WebSocket. O contrato de `GET /api/v1/ws` (autenticado por JWT de usuário, não
por API key) está em [Tempo real](../api/tempo-real.md); em .NET, use
`System.Net.WebSockets.ClientWebSocket`.

## Tratamento de erros

As exceções ficam em `Linktor.Utils` e derivam de `LinktorException`:

| Exceção | Status HTTP | `ErrorCode` |
|---|---|---|
| `ValidationException` | 400 | `VALIDATION_ERROR` |
| `AuthenticationException` | 401 | `AUTHENTICATION_ERROR` |
| `AuthorizationException` | 403 | `AUTHORIZATION_ERROR` |
| `NotFoundException` | 404 | `NOT_FOUND` |
| `RateLimitException` | 429 | `RATE_LIMIT` (propriedade extra `RetryAfter`) |
| `ServerException` | 5xx | `SERVER_ERROR` |
| `LinktorException` | demais | `null` |
| `WebhookVerificationException` | — | `WEBHOOK_VERIFICATION_FAILED` |

Propriedades: `StatusCode` (`int`), `ErrorCode` (`string?`), `RequestId` (`string?`, do header
`X-Request-ID`).

```csharp
using Linktor.Utils;

try
{
    await client.Channels.GetAsync("CHANNEL_ID");
}
catch (NotFoundException)
{
    Console.WriteLine("Canal não existe.");
}
catch (LinktorException ex)
{
    Console.WriteLine($"HTTP {ex.StatusCode} ({ex.ErrorCode}), request {ex.RequestId}");
}
```

:::warning O que a exceção não traz
- `ex.Message` vem **vazio** e `ErrorCode` é fixo por classe: o SDK procura `message` na raiz do JSON,
  mas o servidor responde `{"success": false, "error": {"code", "message", "details"}}`. O código e a
  mensagem reais do servidor se perdem; guarde o `RequestId` para procurar nos logs.
- `StatusCode` de `ServerException` é sempre 500, e `RateLimitException.RetryAfter` é sempre 60.
- Falhas de rede e timeout saem como `HttpRequestException` ou `TaskCanceledException`, e resposta
  que não bate com o modelo sai como `System.Text.Json.JsonException` — nenhuma delas é
  `LinktorException`.
:::

Novas tentativas: em 429 o SDK espera o `Retry-After` (ou 60 s) e repete; em 5xx espera 2 s e depois
4 s. `MaxRetries` é o total de tentativas. A repetição vale também para `POST`, então uma criação
pode ser enviada mais de uma vez se o servidor responder 5xx depois de processar. Códigos de erro da
API em [Erros e paginação](../api/erros-e-paginacao.md).

## Limitações conhecidas {#limitacoes}

| Método | O que faz | Por que não funciona | Alternativa |
|---|---|---|---|
| `Auth.LoginAsync`, `Auth.RefreshTokenAsync` | `POST /auth/login`, `/auth/refresh` | Lê `accessToken` (o servidor devolve `access_token`); o refresh envia `refreshToken` em vez de `refresh_token` | `HttpClient` + opção `AccessToken` |
| `Auth.GetCurrentUserAsync`, `Auth.GetCurrentTenantAsync` | `GET /auth/me`, `/auth/tenant` | Rotas não existem | `GET /me` |
| `Conversations.ListAsync` | `GET /conversations` | Lança `JsonException`: a prioridade padrão `normal` não existe no enum `ConversationPriority`. Filtros saem em camelCase e são ignorados | `HttpClient` |
| `Conversations.GetAsync`, `ResolveAsync`, `ReopenAsync` | `GET`, `POST .../resolve`, `.../reopen` | A ação é executada, mas o retorno vem vazio (`Id == ""`) pelo mesmo motivo do enum | Use só pelo efeito, ou `HttpClient` |
| `Conversations.CreateAsync`, `AssignAsync` | `POST /conversations`, `.../assign` | Envia `channelId`, `contactId`, `userId`; o servidor exige `channel_id`, `contact_id`, `user_id` (400) | `HttpClient` |
| `Conversations.SendMessageAsync` | `POST /conversations/{id}/messages` | Envia `text` e `contentType`; o servidor lê `content` e `content_type` (400) | [Enviar mensagem](#enviar-mensagem) |
| `Conversations.UpdateAsync`, `Contacts.UpdateAsync`, `Bots.UpdateAsync`, `KnowledgeBases.UpdateAsync`, `Flows.UpdateAsync` | `PATCH` no recurso | O servidor só tem `PUT` | `HttpClient` com `PUT` |
| `Contacts.SearchAsync`, `MergeAsync`, `GetByExternalIdAsync` | `/contacts/search`, `/merge`, `/external/{id}` | Rotas não existem | `ListAsync` com `Search` |
| `Bots.CreateAsync` | `POST /bots` | Não envia `type`, `provider` e `model`, obrigatórios | `HttpClient` |
| `Bots.StartAsync`, `StopAsync`, `GetStatusAsync` | `/bots/{id}/start`, `/stop`, `/status` | Rotas não existem | `POST /bots/{id}/activate` e `/deactivate` |
| `AI.Agents.*`, `AI.Completions.*`, `AI.Embeddings.*` | `/ai/agents`, `/ai/completions`, `/ai/embeddings` | Rotas não existem | `POST /ai/complete` etc. |
| `KnowledgeBases.CreateAsync` | `POST /knowledge-bases` | Não envia `type`, obrigatório | `HttpClient` |
| `KnowledgeBases.*Document*`, `QueryAsync` | `/knowledge-bases/{id}/documents`, `/query` | Rotas não existem | `/knowledge-bases/{id}/items` e `/search` |
| `Flows.CreateAsync` | `POST /flows` | Não envia `trigger` e `start_node_id`, obrigatórios | `HttpClient` |
| `Flows.PublishAsync`, `UnpublishAsync`, `ExecuteAsync`, `ValidateAsync`, `DuplicateAsync`, `GetNodeTypesAsync`, `GetExecutionsAsync` | Rotas de publicação e execução | Rotas não existem | `POST /flows/{id}/activate`, `/deactivate`, `/test` |
| `VRE.RenderAndSendAsync` | `POST /vre/render-and-send` | Envia `conversation_id`; o servidor exige `send_to`, `channel` e `channel_id` | `HttpClient` |
| `VRE.ListTemplatesAsync` | `GET /vre/templates` | O servidor devolve uma lista de textos; o SDK espera objetos (`JsonException`) | `HttpClient` |
| `VRE.PreviewAsync` | `POST /vre/templates/{id}/preview` | A rota é `GET` | `HttpClient` |
| `WebhookVerifier.*` | Verifica a assinatura | Assina só o corpo | [Verificação manual](#webhook) |

## Referência: método → rota HTTP

Métodos que chamam uma rota existente com corpo e resposta utilizáveis (caminhos relativos a `/api/v1`):

| Método | Rota |
|---|---|
| `Channels.ListAsync`, `Channels.CreateAsync` | `GET /channels`, `POST /channels` |
| `Channels.GetAsync`, `UpdateAsync`, `DeleteAsync` | `GET`, `PUT`, `DELETE /channels/{id}` |
| `Channels.ConnectAsync` | `POST /channels/{id}/connect` |
| `Channels.RequestPairCodeAsync` | `POST /channels/{id}/pair` |
| `Channels.DisconnectAsync` | `POST /channels/{id}/disconnect` |
| `Conversations.GetMessagesAsync` | `GET /conversations/{id}/messages` |
| `Conversations.DeleteAsync` | `DELETE /conversations/{id}` |
| `Contacts.ListAsync`, `Contacts.CreateAsync` | `GET /contacts`, `POST /contacts` |
| `Contacts.GetAsync`, `Contacts.DeleteAsync` | `GET`, `DELETE /contacts/{id}` |
| `Bots.ListAsync`, `GetAsync`, `DeleteAsync` | `GET /bots`, `GET` e `DELETE /bots/{id}` |
| `KnowledgeBases.ListAsync`, `GetAsync`, `DeleteAsync` | `GET /knowledge-bases`, `GET` e `DELETE /knowledge-bases/{id}` |
| `Flows.ListAsync`, `GetAsync`, `DeleteAsync` | `GET /flows`, `GET` e `DELETE /flows/{id}` |
| `VRE.RenderAsync` | `POST /vre/render` |

Visão comparativa dos SDKs em [Visão geral](./visao-geral.md); a API completa em
[Visão geral da API](../api/visao-geral.md).
