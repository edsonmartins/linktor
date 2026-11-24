---
sidebar_position: 7
title: PHP
description: Referência do SDK PHP do Linktor (linktor/linktor-php) — instalação a partir do repositório, os contornos necessários, o que funciona, o que não funciona e como cobrir o resto com Guzzle.
---

# SDK PHP

O SDK PHP (`linktor/linktor-php`, em `sdks/php`) é um cliente sobre Guzzle com autenticação por
`X-API-Key`, novas tentativas e exceções tipadas. Hoje **só o recurso de canais está alinhado com a
API**; os demais têm rotas, corpos ou modelos defasados. Esta página mostra o que funciona, os dois
contornos obrigatórios e como fazer o restante com Guzzle.

| Recurso | Cobertura | Observação |
|---|---|---|
| Canais | sim | criar, atualizar, remover, conectar por QR, pair code |
| Envio direto (`POST /messages/send`) | não | use `$linktor->post()` ou Guzzle |
| Envio para uma conversa | parcial | envia; o objeto `Message` devolvido perde o texto e os campos compostos |
| Conversas | parcial | `get`, `create`, `delete`, `resolve`, `reopen`; listagem, `update` e `assign` não funcionam |
| Contatos | parcial | `get`, `create`, `delete`; listagem, `update`, `search`, `merge` não funcionam |
| Bots | parcial | `get`, `create`, `delete` |
| Base de conhecimento | parcial | `get`, `create`, `delete` da base; itens e busca não |
| Fluxos | parcial | `get`, `create`, `delete` |
| IA | não | todas as rotas do recurso `ai` não existem no servidor |
| Analytics | não | o SDK não tem o recurso |
| VRE | parcial | só `render` (e os atalhos `render*`) |
| Verificação de webhook | parcial | `verify`/`constructEvent` são incompatíveis; `verifySignature` serve se você montar a base assinada |
| WebSocket | não | o SDK não tem cliente de tempo real; veja [Tempo real](../api/tempo-real.md) |
| Login por e-mail e senha | não | `auth->login()` não guarda o token; faça o login com `post()` |

:::warning Três defeitos transversais — leia antes de usar
Conferidos por leitura do fonte (`sdks/php/src`); o PHP não foi executado na preparação desta página.

1. **O prefixo `/api/v1` se perde.** O cliente configura o Guzzle com `base_uri` e chama caminhos
   absolutos (`/channels`). Pela regra de resolução de URI, `https://linktor.suaempresa.com/api/v1` +
   `/channels` vira `https://linktor.suaempresa.com/channels`, que não existe (404). Contorno: a
   subclasse de [Criar o cliente](#criar-o-cliente).
2. **Várias classes por arquivo, com autoload PSR-4.** `ConnectResult` está em `Types/Channel.php`,
   `Message` em `Types/Conversation.php`, `PaginatedResponse` em `Types/Common.php`,
   `KnowledgeBase` em `Types/Knowledge.php`, as respostas de VRE em `Types/VRE.php`,
   `WebhookConstants` em `Types/Webhook.php`. O autoloader não acha essas classes e o PHP lança
   `Error: Class "..." not found` — **depois** de a requisição já ter sido feita. Contorno: o
   `classmap` de [Instalação](#instalacao).
3. **Listagens paginadas voltam vazias e as mensagens de erro são genéricas.** O cliente devolve só o
   `data` do envelope; `list()` de conversas, contatos, bots, bases e fluxos procura um `data` dentro
   dele e entrega `[]`. Nos erros, o cliente lê `message` na raiz, mas o servidor manda
   `error.message` — a exceção sempre diz `Request failed`.
:::

## Instalação {#instalacao}

O pacote se chama `linktor/linktor-php`. O `composer.json` do SDK não declara versão.

```bash
composer require linktor/linktor-php
```

:::note
Não foi possível confirmar que o pacote está publicado no Packagist. O caminho garantido é instalar a
partir de um clone do repositório `github.com/edsonmartins/linktor`. Um repositório Composer do tipo
`vcs` não serve aqui, porque o `composer.json` do SDK fica na subpasta `sdks/php`, e não na raiz.
:::

Clone o repositório e aponte um repositório do tipo `path` para a subpasta. O bloco `classmap` é o
contorno do defeito 2: ele faz o Composer indexar todas as classes do SDK, arquivo por arquivo.

```bash
git clone https://github.com/edsonmartins/linktor.git ../linktor
```

```json
{
  "repositories": [
    { "type": "path", "url": "../linktor/sdks/php" }
  ],
  "require": {
    "linktor/linktor-php": "@dev"
  },
  "autoload": {
    "classmap": ["vendor/linktor/linktor-php/src/"]
  }
}
```

```bash
composer update linktor/linktor-php
composer dump-autoload
```

## Requisitos

PHP 8.1 ou superior. Dependências instaladas pelo Composer: `guzzlehttp/guzzle` `^7.8` e
`psr/http-message` `^2.0`.

## Criar o cliente {#criar-o-cliente}

`Linktor\LinktorClient` recebe um array de opções. Como os caminhos internos não levam `/api/v1`
(defeito 1), crie uma subclasse que acrescenta o prefixo. Os recursos (`$linktor->channels` etc.)
chamam esses cinco métodos, então a subclasse corrige todos de uma vez.

```php
<?php
declare(strict_types=1);

require __DIR__ . '/vendor/autoload.php';

use Linktor\LinktorClient;

final class Linktor extends LinktorClient
{
    private const PREFIXO = '/api/v1';

    public function get(string $path): array { return parent::get(self::PREFIXO . $path); }
    public function post(string $path, ?array $body = null): array { return parent::post(self::PREFIXO . $path, $body); }
    public function put(string $path, array $body): array { return parent::put(self::PREFIXO . $path, $body); }
    public function patch(string $path, array $body): array { return parent::patch(self::PREFIXO . $path, $body); }
    public function delete(string $path): void { parent::delete(self::PREFIXO . $path); }
}

$linktor = new Linktor([
    'baseUrl' => getenv('LINKTOR_URL'),      // https://linktor.suaempresa.com — SEM /api/v1
    'apiKey'  => getenv('LINKTOR_API_KEY'),  // lk_sua_chave
]);
```

| Opção (alias) | Tipo | Padrão | Efeito |
|---|---|---|---|
| `baseUrl` (`base_url`) | `string` | `https://api.linktor.io` | Endereço do servidor. Só esquema e host são aproveitados; qualquer caminho é descartado. |
| `apiKey` (`api_key`) | `?string` | `null` | Enviada em `X-API-Key`. |
| `accessToken` (`access_token`) | `?string` | `null` | JWT de usuário, enviado em `Authorization: Bearer`. Ignorado se houver `apiKey`. |
| `timeoutSeconds` (`timeout`) | `int` | `30` | Tempo limite por requisição. Precisa ser inteiro. |
| `maxRetries` (`max_retries`) | `int` | `3` | Número total de tentativas. `1` desliga as novas tentativas. |

Não há opção para headers extras nem para injetar um cliente Guzzle próprio.

O cliente tenta de novo em falha de conexão e em resposta 5xx (espera de 2 s, depois 4 s) e em 429
(espera o `Retry-After`, ou 60 s se o header não vier — bloqueando o processo). A nova tentativa vale
também para `POST`: em rotinas de envio, use `'maxRetries' => 1` para não duplicar mensagem.

Os métodos `get`, `post`, `put` e `delete` do cliente são públicos e devolvem o conteúdo de `data` do
envelope como array, com as chaves originais em snake_case. É por eles que você alcança o que os
recursos não cobrem. Para chamar sem corpo, passe `null`: um array PHP vazio vira `[]` em JSON, e as
rotas que leem um objeto respondem 400. O mesmo vale para mapas aninhados (`'config' => []`) — omita
a chave quando o mapa estiver vazio.

## Autenticação

Com `apiKey`, toda requisição sai com `X-API-Key: lk_...`, que é o header que o servidor aceita para
chave de API. Crie a chave e escolha os escopos em [Chaves de API](../painel/chaves-de-api.md); os
detalhes estão em [Autenticação](../api/autenticacao.md).

Para agir como um usuário (JWT), não use `$linktor->auth->login()`: ele lê `accessToken` da
resposta, mas o servidor devolve `access_token`, e o cliente fica sem token. Faça o login direto:

```php
$linktor = new Linktor(['baseUrl' => getenv('LINKTOR_URL')]);

$sessao = $linktor->post('/auth/login', [
    'email'    => 'voce@suaempresa.com',
    'password' => getenv('LINKTOR_PASSWORD'),
]);
$linktor->setAccessToken($sessao['access_token']);   // passa a enviar Authorization: Bearer

// Renovar: expira em $sessao['expires_in'] segundos
$nova = $linktor->post('/auth/refresh', ['refresh_token' => $sessao['refresh_token']]);
$linktor->setAccessToken($nova['access_token']);
```

## Canais

`$linktor->channels` é o recurso alinhado com a API. As entradas são arrays com as chaves da API
(snake_case); as saídas são objetos `Linktor\Types\Channel` e `Linktor\Types\ConnectResult`.

| Método | Retorno |
|---|---|
| `list(array $params = [])` | `Channel[]` (o servidor não filtra; `$params` é ignorado por ele) |
| `get(string $id)` | `Channel` |
| `create(array $input)` | `Channel` |
| `update(string $id, array $input)` | `Channel` |
| `delete(string $id)` | `void` |
| `connect(string $id)` | `ConnectResult` |
| `requestPairCode(string $id, string $phoneNumber)` | `ConnectResult` |
| `disconnect(string $id)` | `Channel` **vazio** — veja o aviso abaixo |

`Channel` expõe `id`, `tenantId`, `type`, `name`, `identifier`, `enabled`, `connectionStatus`
(`disconnected`, `connecting`, `connected`, `error`), `config`, `webhookUrl`, `createdAt`,
`updatedAt`. As credenciais nunca voltam. `ConnectResult` expõe `channel`, `qrCode`, `expiresIn`,
`pairCode`, `passkeyRequired`, `passkeyChallenge`.

### Criar e conectar por QR

```php
$canal = $linktor->channels->create([
    'type' => 'whatsapp', // não oficial
    'name' => 'WhatsApp Vendas',
]);

while (true) {
    $r = $linktor->channels->connect($canal->id);

    if ($r->channel?->connectionStatus === 'connected') {
        break;
    }
    if ($r->passkeyRequired) {
        throw new RuntimeException('Conta protegida por passkey: conecte pelo painel.');
    }
    if ($r->qrCode !== null) {
        mostrarQr($r->qrCode);   // sua função: desenha o QR a partir desse conteúdo
    }

    // O QR vale expires_in segundos. Acompanhe o estado e, se expirar, peça outro.
    $expira = time() + ($r->expiresIn ?? 20);
    while (time() < $expira) {
        sleep(3);
        if ($linktor->channels->get($canal->id)->connectionStatus === 'connected') {
            break 2;
        }
    }
}
```

Alternativa ao QR, por código de pareamento:

```php
$r = $linktor->channels->requestPairCode($canal->id, '+5511999999999');
echo "Digite no WhatsApp do aparelho: {$r->pairCode} (vale {$r->expiresIn} s)\n";
```

O fluxo completo de vínculo está em [WhatsApp não-oficial](../canais/whatsapp-nao-oficial.md).

### Atualizar, apontar o webhook e remover

`update` usa `PUT`. Campos omitidos ficam como estão. `credentials` é mesclado com o que está
guardado; `config`, quando enviado, **substitui** o mapa inteiro — parta do `config` atual (valores
secretos chegam como `__redacted__` e são preservados quando devolvidos assim).

```php
$canal = $linktor->channels->get('CHANNEL_ID');

$linktor->channels->update($canal->id, [
    'name'        => 'WhatsApp Vendas SP',
    'webhook_url' => 'https://app.suaempresa.com/webhooks/linktor',
    'credentials' => ['webhook_secret' => getenv('LINKTOR_WEBHOOK_SECRET')],
    'config'      => array_merge($canal->config, [
        'webhook_events' => 'message.received,message.delivered,message.read,message.failed',
    ]),
]);

$linktor->channels->delete($canal->id);   // 204, sem retorno
```

:::warning
`disconnect()` desconecta o canal, mas o servidor responde só `{"message": "Channel disconnected"}`
e o SDK devolve um `Channel` com todos os campos vazios. Ignore o retorno e chame `get()` em seguida.
:::

## Enviar mensagem

O SDK não tem método para o envio direto. Use Guzzle, que já é dependência do SDK:

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

echo $corpo['data']['id'], ' ', $corpo['data']['status'], PHP_EOL;   // status "queued"
```

Com o cliente da seção [Criar o cliente](#criar-o-cliente) a mesma chamada cabe em uma linha e
devolve o `data` (`id`, `conversation_id`, `channel_id`, `status`):

```php
$envio = $linktor->post('/messages/send', [
    'channel_id' => 'CHANNEL_ID', 'to' => '+5511999999999',
    'content_type' => 'text', 'text' => 'Olá! Seu pedido saiu para entrega.',
]);
```

A chave precisa do escopo `messages:send`. Mídia, `metadata` e correlação estão em
[Enviar mensagens](../api/enviar-mensagens.md).

### Para uma conversa existente

`sendMessage(string $conversationId, array $input): Message` chama
`POST /conversations/{id}/messages` com o array como está:

```php
$msg = $linktor->conversations->sendMessage($conversaId, [
    'content_type' => 'text',
    'content'      => 'Posso ajudar em mais alguma coisa?',
]);
echo $msg->id, ' ', $msg->status;
```

:::warning
`Message` lê as chaves em camelCase e com outros nomes. Só `id`, `status` e `metadata` vêm
preenchidos; `text` fica `null` (o servidor manda `content`), `conversationId`, `direction`,
`externalId` e as datas ficam vazios, e `contentType` é sempre `text`. Se precisar desses campos, use
`$linktor->post("/conversations/{$conversaId}/messages", [...])`, que devolve o array original.
:::

## Conversas, contatos, bots, bases e fluxos

Nesses recursos funcionam os métodos abaixo. Todos recebem arrays com as chaves da API e devolvem
objetos cujo `fromArray` espera camelCase — então os campos de uma palavra chegam e os compostos não.

| Recurso | Métodos que funcionam | Campos preenchidos no retorno | Campos perdidos |
|---|---|---|---|
| `conversations` | `get`, `create`, `delete`, `resolve`, `reopen` | `id`, `status`, `subject`, `metadata` | `channelId`, `contactId`, `assignedTo`, `unreadCount`, `lastMessage`, datas |
| `contacts` | `get`, `create`, `delete` | `id`, `name`, `email`, `phone`, `tags` | `avatarUrl`, `customFields`, datas |
| `bots` | `get`, `create`, `delete` | `id`, `name`, `status` | `type`, `provider`, `model`, `channelIds`, conteúdo de `config`, datas |
| `knowledgeBases` | `get`, `create`, `delete` | `id`, `name`, `description`, `status` | `type`, contagem de itens, `config`, datas |
| `flows` | `get`, `create`, `delete` | `id`, `name`, `description` | `trigger`, `start_node_id`, `is_active`, conteúdo dos nós, datas |

```php
$contato = $linktor->contacts->create([
    'name'          => 'Maria Souza',
    'phone'         => '+5511999999999',
    'email'         => 'cliente@exemplo.com',
    'custom_fields' => ['codigo_erp' => '10482'],
    'tags'          => ['vip'],
]);

$conversa = $linktor->conversations->create([
    'contact_id' => $contato->id,
    'channel_id' => 'CHANNEL_ID',
    'subject'    => 'Pedido 10482',
]);

$linktor->conversations->resolve($conversa->id);
```

Para ler todos os campos, listar ou atualizar, use os métodos crus do cliente:

```php
// Lista: devolve o array de itens. O bloco meta (total, next_cursor) é descartado pelo cliente.
$abertas  = $linktor->get('/conversations?status=open&limit=50');
$contatos = $linktor->get('/contacts?page=1&page_size=50&search=maria');

// Atualização é PUT (o SDK manda PATCH, que não existe)
$linktor->put("/conversations/{$conversa->id}", ['priority' => 'high', 'tags' => ['entrega']]);
$linktor->post("/conversations/{$conversa->id}/assign", ['user_id' => $usuarioId]);

// Bots, base de conhecimento, fluxos, IA
$linktor->post("/bots/{$botId}/activate", null);
$linktor->post("/knowledge-bases/{$baseId}/items", ['question' => 'Qual o prazo?', 'answer' => 'Até 3 dias úteis.']);
$achados = $linktor->post("/knowledge-bases/{$baseId}/search", ['query' => 'prazo de entrega', 'limit' => 3]);
$texto   = $linktor->post('/ai/complete', [
    'provider' => 'openai', 'model' => 'gpt-4o-mini',
    'messages' => [['role' => 'user', 'content' => 'Resuma: pedido saiu para entrega.']],
]);
```

:::note
`PUT /contacts/{id}` usa o mesmo corpo da criação e grava como vazio o que você omitir: envie o
contato completo. Conversas paginam por cursor (`limit`, `before`); como o cliente descarta o
`meta.next_cursor`, para percorrer páginas use Guzzle e leia o envelope inteiro. Rotas e corpos:
[Erros e paginação](../api/erros-e-paginacao.md) e [Visão geral da API](../api/visao-geral.md).
:::

## VRE

As rotas `/vre/*` só existem quando o serviço VRE está habilitado no servidor. Do recurso
`$linktor->vre`, só a renderização funciona:

`render(string $tenantId, string $templateId, array $data, ?string $channel = null, ?string $format = null, ?int $width = null, ?int $quality = null, ?float $scale = null): VRERenderResponse`.
O servidor usa o tenant da credencial e responde 403 se `tenant_id` vier preenchido com outro valor.
Passe string vazia:

```php
$img = $linktor->vre->render('', 'menu_opcoes', [
    'titulo' => 'Como podemos ajudar?',
    'opcoes' => [['label' => 'Segunda via'], ['label' => 'Falar com atendente']],
], 'whatsapp', 'png');

file_put_contents('menu.png', base64_decode($img->imageBase64));
```

`VRERenderResponse` expõe `imageBase64`, `caption`, `width`, `height`, `format`, `renderTimeMs`,
`sizeBytes`, `cacheHit`. Os atalhos `renderMenu`, `renderProductCard`, `renderOrderStatus`,
`renderProductList`, `renderConfirmation` e `renderPixPayment` chamam `render` com um
`template_id` fixo e têm o mesmo primeiro parâmetro `$tenantId`.

## Verificar a assinatura do webhook

O Linktor assina cada entrega com HMAC-SHA256 do texto `<timestamp>.<corpo cru>`, usando o
`credentials.webhook_secret` do canal. O resultado vai em hexadecimal, sem prefixo, no header
`X-Linktor-Signature`; o timestamp (epoch em segundos) vai em `X-Linktor-Timestamp`.

:::warning
`WebhookVerifier::verify()` e `LinktorClient::constructWebhookEvent()` assinam **só o corpo**, sem
o timestamp: a comparação nunca bate com o que o servidor envia. Não use os dois. A função de baixo
nível `LinktorClient::verifyWebhookSignature($base, $assinatura, $segredo)` calcula o HMAC do texto
que receber e compara em tempo constante — serve, desde que você monte a base
`$timestamp . '.' . $corpo` e confira a tolerância por conta própria.
:::

Receptor em PHP puro, com a verificação completa:

```php
<?php
declare(strict_types=1);

$segredo    = (string) getenv('LINKTOR_WEBHOOK_SECRET');
$corpo      = file_get_contents('php://input');           // corpo cru, antes de qualquer json_decode
$assinatura = $_SERVER['HTTP_X_LINKTOR_SIGNATURE'] ?? '';
$timestamp  = $_SERVER['HTTP_X_LINKTOR_TIMESTAMP'] ?? '';

function assinaturaValida(string $corpo, string $assinatura, string $timestamp, string $segredo): bool
{
    if ($segredo === '' || $assinatura === '' || !ctype_digit($timestamp)) {
        return false;
    }
    if (abs(time() - (int) $timestamp) > 300) {            // tolerância contra repetição
        return false;
    }
    $esperada = hash_hmac('sha256', $timestamp . '.' . $corpo, $segredo);
    return hash_equals($esperada, strtolower($assinatura));
}

if (!assinaturaValida($corpo, $assinatura, $timestamp, $segredo)) {
    http_response_code(401);
    exit;
}

$evento = json_decode($corpo, true, 512, JSON_THROW_ON_ERROR);

if ($evento['type'] === 'message.received') {
    $texto = $evento['data']['message']['content']['text'] ?? '';
    error_log("[{$evento['data']['conversationId']}] {$texto}");
}

http_response_code(200);   // responda rápido; processe o trabalho pesado em fila
```

O envelope do evento é camelCase (`id`, `type`, `timestamp`, `tenantId`, `environment`, `data`).
Depois de validar, `Linktor\Types\WebhookEvent::fromArray($evento)` devolve um objeto com `id`,
`type`, `timestamp`, `tenantId` e `data`; `getEventType()` devolve `null` para os tipos que o enum
do SDK não conhece (`call.*`, `conversation.reopened`, `conversation.escalated`,
`conversation.deleted`). A lista de eventos e de campos está em
[Receber eventos](../api/receber-eventos.md).

No Laravel, a mesma função com o corpo cru da requisição (rota em `routes/api.php`):

```php
Route::post('/webhooks/linktor', function (Illuminate\Http\Request $request) {
    $ok = assinaturaValida(
        $request->getContent(),
        (string) $request->header('X-Linktor-Signature'),
        (string) $request->header('X-Linktor-Timestamp'),
        (string) config('services.linktor.webhook_secret'),
    );
    abort_unless($ok, 401);

    ProcessarEventoLinktor::dispatch($request->json()->all());   // seu job
    return response()->noContent();
});
```

## Tratamento de erros

Toda resposta fora de 2xx vira uma exceção do namespace `Linktor\Utils`:

| Classe | Quando | `getStatusCode()` | `getErrorCode()` |
|---|---|---|---|
| `ValidationException` | 400 | 400 | `VALIDATION_ERROR` |
| `AuthenticationException` | 401 | 401 | `AUTHENTICATION_ERROR` |
| `AuthorizationException` | 403 | 403 | `AUTHORIZATION_ERROR` |
| `NotFoundException` | 404 | 404 | `NOT_FOUND` |
| `RateLimitException` | 429, esgotadas as tentativas | 429 | `RATE_LIMIT` |
| `ServerException` | 5xx, esgotadas as tentativas | sempre 500 | `SERVER_ERROR` |
| `LinktorException` (base) | demais status (409, 422…) e falha de conexão | o status, ou 0 na falha de conexão | `null` |

Todas herdam de `LinktorException` e têm `getMessage()`, `getStatusCode()`, `getErrorCode()` e
`getRequestId()` (valor do header `X-Request-ID` da resposta). `RateLimitException::getRetryAfter()`
devolve sempre 60.

```php
use Linktor\Utils\LinktorException;
use Linktor\Utils\NotFoundException;

try {
    $canal = $linktor->channels->get('CHANNEL_ID');
} catch (NotFoundException $e) {
    echo "Canal não existe (request {$e->getRequestId()})\n";
} catch (LinktorException $e) {
    echo "Falha HTTP {$e->getStatusCode()}\n";
}
```

:::warning
`getErrorCode()` é fixo por status, não o `error.code` do servidor, e `getMessage()` é sempre
`Request failed` (defeito 3). Quando precisar do código e da mensagem reais — por exemplo, para
saber qual campo falhou na validação — faça a chamada com Guzzle e `'http_errors' => false`, como em
[Enviar mensagem](#enviar-mensagem), e leia `error.code`, `error.message` e `error.details`. O
formato está em [Erros e paginação](../api/erros-e-paginacao.md).
:::

## Limitações conhecidas

| Método | O que faz | Por que não funciona | Alternativa |
|---|---|---|---|
| `list()` de `conversations`, `contacts`, `bots`, `knowledgeBases`, `flows`; `conversations->getMessages()` | `GET` da coleção | procura `data` dentro do `data` já extraído: devolve `PaginatedResponse` vazio | `$linktor->get('/conversations?...')` |
| `update()` de `conversations`, `contacts`, `bots`, `knowledgeBases`, `flows` | `PATCH /recurso/{id}` | o servidor só tem `PUT` | `$linktor->put(...)` |
| `conversations->assign()` | `POST .../assign` com `userId` | o servidor exige `user_id` (400) | `$linktor->post(".../assign", ['user_id' => ...])` |
| `contacts->search()`, `merge()`, `getByExternalId()` | `/contacts/search`, `/contacts/merge`, `/contacts/external/{id}` | rotas não existem | `GET /contacts?search=` |
| `bots->start()`, `stop()`, `getStatus()` | `/bots/{id}/start`, `/stop`, `/status` | rotas não existem | `POST /bots/{id}/activate` e `/deactivate` |
| `ai->agents->*`, `ai->completions->*`, `ai->embeddings->*` | `/ai/agents`, `/ai/completions`, `/ai/embeddings` | rotas não existem | `POST /ai/complete`, `/ai/classify-intent`, `/ai/analyze-sentiment` |
| `knowledgeBases->*Document()`, `query()` | `/knowledge-bases/{id}/documents`, `/query` | rotas não existem | `/knowledge-bases/{id}/items` e `/search` |
| `flows->publish()`, `unpublish()`, `execute()`, `validate()`, `duplicate()`, `getNodeTypes()`, `getExecutions()` | rotas homônimas | rotas não existem | `POST /flows/{id}/activate`, `/deactivate`, `/test` |
| `auth->login()` | `POST /auth/login` | lê `accessToken`; o servidor manda `access_token`, e o token não é guardado | `post('/auth/login')` + `setAccessToken()` |
| `auth->refreshToken()` | `POST /auth/refresh` com `refreshToken` | o servidor exige `refresh_token` | `post('/auth/refresh', ['refresh_token' => ...])` |
| `auth->getCurrentUser()`, `getCurrentTenant()` | `/auth/me`, `/auth/tenant` | rotas não existem | `GET /me`, `GET /tenant` |
| `vre->renderAndSend()` | `POST /vre/render-and-send` com `conversation_id` | o servidor exige `send_to`, `channel` e `channel_id` | Guzzle com esses campos |
| `vre->listTemplates()` | `GET /vre/templates` | o servidor devolve lista de strings; o SDK espera objetos (`TypeError`) | `$linktor->get('/vre/templates')` |
| `vre->preview()` | `POST /vre/templates/{id}/preview` | a rota é `GET` | `$linktor->get("/vre/templates/{$id}/preview")` |
| `WebhookVerifier::verify()`, `constructEvent()` | valida header e assinatura | assina só o corpo, sem o timestamp | verificação manual desta página |

## Referência: método → rota HTTP

Métodos que funcionam com os dois contornos aplicados. Todos os caminhos são relativos a `/api/v1`.

| Método | Rota |
|---|---|
| `channels->list()` / `create($input)` | `GET /channels` / `POST /channels` |
| `channels->get($id)` / `update($id, $input)` / `delete($id)` | `GET` / `PUT` / `DELETE /channels/{id}` |
| `channels->connect($id)` | `POST /channels/{id}/connect` |
| `channels->requestPairCode($id, $phoneNumber)` | `POST /channels/{id}/pair` |
| `channels->disconnect($id)` | `POST /channels/{id}/disconnect` |
| `conversations->get($id)` / `create($input)` / `delete($id)` | `GET /conversations/{id}` / `POST /conversations` / `DELETE /conversations/{id}` |
| `conversations->sendMessage($id, $input)` | `POST /conversations/{id}/messages` |
| `conversations->resolve($id)` / `reopen($id)` | `POST /conversations/{id}/resolve` / `/reopen` |
| `contacts->get($id)` / `create($input)` / `delete($id)` | `GET /contacts/{id}` / `POST /contacts` / `DELETE /contacts/{id}` |
| `bots->get($id)` / `create($input)` / `delete($id)` | `GET /bots/{id}` / `POST /bots` / `DELETE /bots/{id}` |
| `knowledgeBases->get($id)` / `create($input)` / `delete($id)` | `GET /knowledge-bases/{id}` / `POST /knowledge-bases` / `DELETE /knowledge-bases/{id}` |
| `flows->get($id)` / `create($input)` / `delete($id)` | `GET /flows/{id}` / `POST /flows` / `DELETE /flows/{id}` |
| `auth->logout()` | `POST /auth/logout` |
| `vre->render(...)` e atalhos `render*` | `POST /vre/render` |
| `get($path)`, `post($path, $body)`, `put($path, $body)`, `delete($path)` | qualquer rota da API |

Veja também a [visão geral dos SDKs](./visao-geral.md).
