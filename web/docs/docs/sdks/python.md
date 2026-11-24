---
sidebar_position: 3
title: Python
description: Referência do SDK Python do Linktor — o que funciona hoje (canais dos tipos clássicos), o que não funciona e como fazer o restante com httpx, incluindo envio direto e verificação de webhook em FastAPI.
---

# Python

O pacote `linktor` (pasta `sdks/python` do repositório) traz um cliente síncrono sobre `httpx` e
modelos `pydantic`. Hoje, só o recurso **canais** está alinhado com a API, e mesmo ele com uma
restrição de tipo. O resto desta página mostra o que usar do SDK e o que fazer com `httpx`, que já
vem instalado como dependência do SDK.

:::warning O que não funciona hoje
- **Só `client.channels` desempacota o envelope** `{"success", "data", ...}` da API. Conversas,
  contatos, bots, IA, base de conhecimento, fluxos e login passam o envelope inteiro para o modelo
  e terminam em `pydantic.ValidationError` — depois de a requisição já ter sido executada no servidor.
- **`client.channels` falha com canais `whatsapp_unofficial`**, `whatsapp_official`, `voice`,
  `teams`, `slack`, `mattermost` e `direto`: o enum `ChannelType` do SDK só conhece `whatsapp`,
  `telegram`, `facebook`, `instagram`, `webchat`, `sms`, `email` e `rcs`. `channels.list()` falha se
  o tenant tiver **qualquer** canal de outro tipo.
- **`LinktorAsyncClient` não tem recursos** (só `close()`), **não há envio direto**
  (`POST /messages/send`), **não há cliente WebSocket** e o **verificador de webhook é incompatível**
  com a assinatura que o servidor envia.

Para tudo isso, o caminho que funciona é HTTP direto com `httpx`, mostrado em cada seção.
:::

| Recurso | No SDK | Observação |
|---|---|---|
| Canais | parcial | Funciona para os 8 tipos do enum; `disconnect` sempre lança erro |
| Enviar mensagem (envio direto) | não | Use `httpx` |
| Enviar mensagem para uma conversa | não | Corpo e resposta incompatíveis; use `httpx` |
| Conversas | não | Use `httpx` |
| Contatos, bots, fluxos, base de conhecimento | parcial | Só `delete` |
| IA | não | Rotas inexistentes no servidor |
| Analytics | parcial | Só `get_conversation_metrics`, devolve o envelope cru |
| VRE | parcial | Só `render` e os atalhos `render_*` |
| Verificação de webhook | não | Assina só o corpo; o servidor assina `timestamp.corpo` |
| Tempo real (WebSocket) | não | Não há cliente; veja [Tempo real](../api/tempo-real.md) |
| Cliente assíncrono | não | Classe existe, sem recursos; use `httpx.AsyncClient` |
| Autenticação por API key | sim | Sai em `X-API-Key`, como o servidor espera |

## Instalação

O manifest (`pyproject.toml`) declara o pacote `linktor`, versão `1.0.0`.

:::note
Não foi possível confirmar que `linktor` está publicado no PyPI. O caminho garantido é instalar a
partir do repositório.
:::

```bash
pip install "git+https://github.com/edsonmartins/linktor.git#subdirectory=sdks/python"
```

Em `requirements.txt` ou nas dependências de um `pyproject.toml`, use
`linktor @ git+https://github.com/edsonmartins/linktor.git#subdirectory=sdks/python`. de um clone local do repositório: `pip install ./linktor/sdks/python`.

## Requisitos

- **Python 3.10 ou superior.** O manifest declara `>=3.9`, mas o pacote usa anotações `str | bytes`
  avaliadas na importação: em Python 3.9, `import linktor` falha com `TypeError`.
- Dependências instaladas junto: `httpx>=0.25.0`, `pydantic>=2.0`, `websockets>=12.0` (esta última
  é declarada, mas nenhum módulo do SDK a usa).
- Para o receptor de webhook desta página: `pip install fastapi uvicorn`.

## Criar o cliente

A `base_url` precisa **incluir `/api/v1`**: o SDK concatena caminhos como `/channels` direto nela. O
padrão (`https://api.linktor.io`) não serve para uma instalação própria.

```python
import os
from linktor import LinktorClient

client = LinktorClient(
    base_url=os.environ["LINKTOR_URL"].rstrip("/") + "/api/v1",
    api_key=os.environ["LINKTOR_API_KEY"],
    timeout=30.0,
    max_retries=3,
)
```

| Opção | Tipo | Padrão | O que faz |
|---|---|---|---|
| `base_url` | `str` | `"https://api.linktor.io"` | Base das chamadas; a barra final é removida. Inclua `/api/v1` |
| `api_key` | `Optional[str]` | `None` | Enviada em `X-API-Key` |
| `access_token` | `Optional[str]` | `None` | JWT de usuário, enviado em `Authorization: Bearer`. Ignorado no construtor se `api_key` foi passada |
| `timeout` | `float` | `30.0` | Tempo limite por requisição, em segundos |
| `max_retries` | `int` | `3` | Novas tentativas em 429, 5xx, erro de rede e timeout |
| `retry_delay` | `float` | `1.0` | Espera base; cresce como `retry_delay * 2**tentativa` |
| `headers` | `Optional[dict[str, str]]` | `None` | Headers adicionais em toda requisição |
| `on_token_refresh` | `Optional[Callable[[], str]]` | `None` | Aceito e guardado, mas nunca chamado |

Métodos do cliente: `set_api_key(api_key)`, `set_access_token(access_token)`, `close()`; também
funciona como gerenciador de contexto (`with LinktorClient(...) as client:`). As novas tentativas
valem para qualquer verbo, inclusive `POST` e `PUT`: use `max_retries=0` se a operação não puder ser
repetida.

### Cliente HTTP para o que o SDK não cobre

Os exemplos com `httpx` desta página usam este cliente e esta função:

```python
import os
import httpx

api = httpx.Client(
    base_url=os.environ["LINKTOR_URL"].rstrip("/") + "/api/v1",
    headers={"X-API-Key": os.environ["LINKTOR_API_KEY"]},
    timeout=30.0,
)

def data(resp: httpx.Response):
    """Devolve o campo `data` do envelope; lança httpx.HTTPStatusError em 4xx/5xx."""
    resp.raise_for_status()
    return resp.json()["data"] if resp.content else None
```

Em código assíncrono, troque por `httpx.AsyncClient` com os mesmos argumentos e use `await` nas
chamadas. `LinktorAsyncClient` existe e aceita as mesmas opções (menos `on_token_refresh`), mas não
expõe nenhum recurso.

## Autenticação

- **API key** (`api_key=`): sai no header `X-API-Key`. É a forma indicada para integração entre
  sistemas; as permissões vêm dos scopes da chave — veja [Chaves de API](../painel/chaves-de-api.md)
  e [Autenticação](../api/autenticacao.md).
- **JWT de usuário** (`access_token=`): sai em `Authorization: Bearer`. `client.auth.login()` não
  funciona (veja [Limitações conhecidas](#limitacoes)); obtenha o token com `httpx` e entregue ao
  cliente:

```python
import httpx
from linktor import LinktorClient

base = "https://linktor.suaempresa.com/api/v1"
login = httpx.post(f"{base}/auth/login", json={"email": "voce@suaempresa.com", "password": "sua-senha"})
login.raise_for_status()
token = login.json()["data"]["access_token"]

client = LinktorClient(base_url=base, access_token=token)
```

Não combine os dois: se o cliente foi criado com `api_key` e você chamar `set_access_token()`, os
dois headers passam a ser enviados e o servidor considera só o `Bearer`.

## Canais

### Com o SDK (tipos `whatsapp`, `telegram`, `facebook`, `instagram`, `webchat`, `sms`, `email`, `rcs`)

| Método | Retorno |
|---|---|
| `client.channels.list(**params)` | `list[Channel]` |
| `client.channels.get(id)` | `Channel` |
| `client.channels.create(**kwargs)` | `Channel` |
| `client.channels.update(id, **kwargs)` | `Channel` |
| `client.channels.delete(id)` | `None` |
| `client.channels.connect(id)` | `ConnectResult` |
| `client.channels.request_pair_code(id, phone_number)` | `ConnectResult` |

Os `kwargs` vão para o corpo JSON como estão; use os nomes da API: `type`, `name`, `identifier`,
`environment`, `config`, `credentials`, `webhook_url`. `config` e `credentials` são dicionários de
string para string. `Channel` tem `id`, `tenant_id`, `type`, `name`, `identifier`, `enabled`,
`connection_status`, `config`, `webhook_url`, `created_at`, `updated_at`; `credentials` nunca volta.

```python
canal = client.channels.create(
    type="telegram",
    name="Telegram Suporte",
    credentials={"bot_token": "123456:ABC-DEF"},
)

client.channels.update(
    canal.id,
    webhook_url="https://app.suaempresa.com/webhooks/linktor",
    credentials={"bot_token": "123456:ABC-DEF", "webhook_secret": "um-segredo-longo"},
)

client.channels.delete(canal.id)
```

### WhatsApp não oficial (`whatsapp`) com httpx

O SDK envia a requisição, mas não consegue ler a resposta de um canal deste tipo. Faça o ciclo
inteiro com o cliente `api` definido acima. Detalhes do canal em
[WhatsApp não-oficial](../canais/whatsapp-nao-oficial.md).

```python
import time

canal = data(api.post("/channels", json={"type": "whatsapp", "name": "WhatsApp Vendas"}))
channel_id = canal["id"]

def conectar_por_qr(channel_id: str) -> None:
    while True:
        r = data(api.post(f"/channels/{channel_id}/connect"))
        if r["channel"]["connection_status"] == "connected":
            return
        if r.get("passkey_required"):
            raise RuntimeError("Conta protegida por passkey: conecte pelo painel")
        # qr_code é o conteúdo do QR: gere a imagem com a biblioteca de sua preferência.
        print("Leia este QR no WhatsApp:", r["qr_code"])
        # O QR vale expires_in segundos; consulte o canal enquanto isso e renove ao expirar.
        expira = time.monotonic() + (r.get("expires_in") or 20)
        while time.monotonic() < expira:
            time.sleep(3)
            atual = data(api.get(f"/channels/{channel_id}"))
            if atual["connection_status"] == "connected":
                return

conectar_por_qr(channel_id)
```

Código de pareamento, em vez do QR:

```python
r = data(api.post(f"/channels/{channel_id}/pair", json={"phone_number": "+5511999999999"}))
print("Digite no WhatsApp:", r["pair_code"], "- vale", r.get("expires_in"), "segundos")
```

Atualizar o canal para receber eventos na sua aplicação — a URL vai em `webhook_url`, o segredo da
assinatura em `credentials.webhook_secret` e o filtro opcional em `config.webhook_events` (lista
separada por vírgula; vazio envia todos):

```python
data(api.put(f"/channels/{channel_id}", json={
    "webhook_url": "https://app.suaempresa.com/webhooks/linktor",
    "credentials": {"webhook_secret": "um-segredo-longo"},
    "config": {"webhook_events": "message.received,message.delivered,message.failed"},
}))
```

Desconectar é `data(api.post(f"/channels/{channel_id}/disconnect"))`; remover é
`api.delete(f"/channels/{channel_id}")`, que responde 204 sem corpo.

## Enviar mensagem

O SDK não tem método para `POST /messages/send`, e `conversations.send_message` / `send_text` não
funcionam (`send_text` envia `text`, a rota espera `content`, e a resposta não é lida). Use `httpx`. A chave precisa do scope
`messages:send`.

Envio direto, sem conversa prévia — responde **202** com a mensagem enfileirada:

```python
msg = data(api.post("/messages/send", json={
    "channel_id": "CHANNEL_ID",
    "to": "+5511999999999",
    "content_type": "text",
    "text": "Olá! Seu pedido saiu para entrega.",
    "metadata": {"pedido": "12345"},
}))
print(msg["id"], msg["conversation_id"], msg["status"])   # status == "queued"
```

Para uma conversa existente:

```python
msg = data(api.post("/conversations/CONVERSATION_ID/messages", json={
    "content_type": "text",
    "content": "Posso ajudar em mais alguma coisa?",
}))
```

Mídia, `metadata` e acompanhamento da entrega estão em [Enviar mensagens](../api/enviar-mensagens.md).

## Conversas e contatos

Nenhum método de leitura ou escrita desses recursos funciona no SDK; `client.contacts.delete(id)` é
a exceção. Com `httpx`:

```python
# Conversas: paginação por cursor (limit + before)
resp = api.get("/conversations", params={"status": "open", "channel_id": "CHANNEL_ID", "limit": 20})
resp.raise_for_status()
conversas = resp.json()["data"]
proximo = resp.json().get("meta", {}).get("next_cursor")

mensagens = data(api.get("/conversations/CONVERSATION_ID/messages", params={"limit": 50}))
data(api.post("/conversations/CONVERSATION_ID/assign", json={"user_id": "USER_ID"}))
data(api.post("/conversations/CONVERSATION_ID/resolve"))

# Contatos: paginação por página (page + page_size)
contato = data(api.post("/contacts", json={
    "name": "Maria Souza",
    "phone": "+5511999999999",
    "email": "cliente@exemplo.com",
    "tags": ["vip"],
}))
encontrados = data(api.get("/contacts", params={"search": "Maria", "page": 1, "page_size": 20}))

client.contacts.delete(contato["id"])   # SDK: DELETE /contacts/{id}
```

`PUT /contacts/{id}` substitui o contato: campo omitido vira string vazia. Envie o contato completo.

## Bots, fluxos, base de conhecimento e IA

No SDK, só as remoções funcionam: `client.bots.delete(id)`, `client.flows.delete(id)` e
`client.knowledge_bases.delete(id)`. Os demais métodos estão em
[Limitações conhecidas](#limitacoes). As rotas reais (`/bots`, `/flows`, `/knowledge-bases`, `/ai/*`)
estão em [Visão geral da API](../api/visao-geral.md); chame-as com o cliente `api`:

```python
resposta = data(api.post("/knowledge-bases/KB_ID/search", json={"query": "prazo de entrega", "limit": 3}))
```

## Analytics

`client.analytics.get_conversation_metrics(**params)` chama `GET /analytics/conversations` e devolve
o **envelope inteiro** como `dict`; os dados estão em `["data"]`. Parâmetros: `period`, `start_date`,
`end_date`. Exige o scope `analytics:read`.

```python
metricas = client.analytics.get_conversation_metrics(period="weekly")["data"]

# overview, flows, escalations e channels: só por HTTP
visao_geral = data(api.get("/analytics/overview", params={"period": "weekly"}))
```

## VRE

As rotas `/vre/*` só existem quando o VRE está habilitado no servidor e respondem JSON sem envelope —
por isso `client.vre.render` consegue ler a resposta.

```text
client.vre.render(tenant_id, template_id, data, channel=None, format=None,
                  width=None, quality=None, scale=None) -> VRERenderResponse
```

`tenant_id` é posicional e obrigatório na assinatura, mas o servidor usa o tenant da chave: passe
`""` (um valor diferente do tenant autenticado é recusado com 403).

```python
img = client.vre.render("", "menu_opcoes", {
    "titulo": "Como podemos ajudar?",
    "opcoes": [{"label": "Acompanhar pedido"}, {"label": "Falar com atendente"}],
}, channel="whatsapp")
print(img.width, img.height, img.format.value, len(img.image_base64))
```

Os atalhos `render_menu`, `render_product_card`, `render_order_status`, `render_product_list`,
`render_confirmation` e `render_pix_payment` montam o `data` e chamam `render` com o mesmo
`tenant_id` na primeira posição. `render_and_send`, `list_templates` e `preview` não funcionam.

## Verificar a assinatura do webhook

:::warning O verificador do SDK não é compatível
`verify_webhook`, `verify_webhook_signature`, `construct_event` e `create_webhook_handler` calculam
o HMAC **só sobre o corpo**. O servidor assina **`timestamp + "." + corpo`**. Com um evento real,
`verify_webhook` devolve `False` e `construct_event` lança `ValueError`. Além disso, o enum
`WebhookEventType` não conhece `call.*`, `conversation.reopened`, `conversation.escalated` nem
`conversation.deleted`. Verifique manualmente, como abaixo.
:::

O servidor envia `X-Linktor-Signature` (HMAC-SHA256 em hexadecimal minúsculo, sem prefixo) e
`X-Linktor-Timestamp` (epoch em segundos). O segredo é o `credentials.webhook_secret` do canal.

```python
import hashlib, hmac, json, os, time

from fastapi import FastAPI, HTTPException, Request

app = FastAPI()
SECRET = os.environ["LINKTOR_WEBHOOK_SECRET"].encode()
TOLERANCIA = 300  # segundos

def assinatura_valida(body: bytes, signature: str, timestamp: str) -> bool:
    if not signature or not timestamp.isdigit():
        return False
    if abs(time.time() - int(timestamp)) > TOLERANCIA:
        return False
    esperado = hmac.new(SECRET, timestamp.encode() + b"." + body, hashlib.sha256).hexdigest()
    return hmac.compare_digest(esperado.encode(), signature.encode())

@app.post("/webhooks/linktor")
async def receber(request: Request):
    body = await request.body()   # corpo cru: não use request.json() antes de verificar
    if not assinatura_valida(
        body,
        request.headers.get("X-Linktor-Signature", ""),
        request.headers.get("X-Linktor-Timestamp", ""),
    ):
        raise HTTPException(status_code=401, detail="assinatura inválida")

    evento = json.loads(body)
    if evento["type"] == "message.received":
        d = evento["data"]
        print(d["conversationId"], d["message"]["content"].get("text"))
    elif evento["type"] in ("message.delivered", "message.read", "message.failed"):
        print(evento["data"]["messageId"], evento["data"]["status"])

    return {"ok": True}
```

Para rodar: `LINKTOR_WEBHOOK_SECRET=um-segredo-longo uvicorn app:app --port 8000`. O envelope do
evento usa camelCase (`tenantId`, `conversationId`, `contentType`), diferente da API REST. Tipos e
campos de cada evento estão em [Receber eventos](../api/receber-eventos.md).

## Tratamento de erros

As exceções do SDK ficam em `linktor` e herdam de `LinktorError`, com os atributos `message`,
`code`, `status_code`, `request_id` (header `X-Request-ID` da resposta) e `details`.

| Exceção | Quando |
|---|---|
| `ValidationError` | HTTP 400 |
| `AuthenticationError` | HTTP 401 |
| `AuthorizationError` | HTTP 403 (inclui scope ausente na chave) |
| `NotFoundError` | HTTP 404 — recurso ou rota inexistente |
| `ConflictError` | HTTP 409 |
| `RateLimitError` | HTTP 429; tem `retry_after` em segundos |
| `ServerError` | HTTP 5xx |
| `NetworkError` | Falha de conexão |
| `TimeoutError` | Tempo limite (é `linktor.TimeoutError`, não a embutida do Python) |
| `LinktorError` | Demais status (`WebSocketError` é exportada, mas nunca lançada) |

:::warning O que se perde
- O SDK procura `message` e `code` na raiz do corpo, e a API os envia dentro de `error`. Em 400,
  401, 403, 409 e 5xx a mensagem vira `"Unknown error"`; em 404 é sempre `"Resource not found"`.
  `code` é o código fixo da classe (`VALIDATION_ERROR`, `NOT_FOUND`…), não o do servidor. Só o 429
  traz a mensagem real.
- Uma resposta que o SDK não consegue ler lança `pydantic.ValidationError`, que **não** herda de
  `LinktorError` — e a operação já foi executada no servidor.
:::

```python
import pydantic
from linktor import LinktorError, NotFoundError, RateLimitError

try:
    canal = client.channels.get("CHANNEL_ID")
except NotFoundError:
    print("canal não existe")
except RateLimitError as e:
    print("limite atingido; tente de novo em", e.retry_after, "s")
except LinktorError as e:
    print(e.status_code, e.code, e.request_id)
except pydantic.ValidationError:
    print("resposta fora do modelo do SDK (ex.: canal whatsapp_unofficial)")
```

Com `httpx`, a mensagem e o código reais ficam acessíveis:

```python
try:
    data(api.post("/messages/send", json={"channel_id": "CHANNEL_ID", "to": "+5511999999999",
                                           "content_type": "text", "text": "Olá"}))
except httpx.HTTPStatusError as e:
    corpo = e.response.json()
    erro = corpo.get("error") or corpo   # o 429 vem sem envelope: {"code", "message", "retry_after"}
    print(e.response.status_code, erro.get("code"), erro.get("message"))
```

Códigos e formato completo em [Erros e paginação](../api/erros-e-paginacao.md).

## Limitações conhecidas {#limitacoes}

| Método | O que faz | Por que não funciona | Alternativa |
|---|---|---|---|
| `channels.*` com tipo fora do enum | Chama a rota correta | `ChannelType` não tem `whatsapp_unofficial`, `whatsapp_official`, `voice`, `teams`, `slack`, `mattermost`, `direto` → `pydantic.ValidationError` na resposta | `httpx` |
| `channels.disconnect` | `POST /channels/{id}/disconnect` | Desconecta, mas a resposta é `{"message": ...}` e o SDK tenta montar um `Channel` | `api.post(".../disconnect")` |
| `auth.login` | `POST /auth/login` | Não desempacota o envelope; modelo exige `token_type`, que a API não envia | `httpx` + `access_token=` |
| `auth.refresh_token` | `POST /auth/refresh` | Envia `refreshToken`; a API lê `refresh_token` | `httpx` |
| `auth.get_current_user` | `GET /auth/me` | Rota não existe (a real é `GET /me`) | `api.get("/me")` |
| `list`, `get`, `create` de `conversations`, `contacts`, `bots`, `knowledge_bases`, `flows`; `contacts.search`; `conversations.list_messages` | Chamam as rotas corretas | Não desempacotam o envelope; os modelos esperam `pagination` e campos camelCase que a API não envia. Um `create` chega a criar o registro | `httpx` |
| `update` de `conversations`, `contacts`, `bots`, `knowledge_bases`, `flows`; `conversations.resolve`; `activate` e `deactivate` de `bots` e `flows` | `PATCH` no recurso | A API não tem `PATCH` nesses recursos: são `PUT` no recurso, `POST .../resolve`, `POST .../activate` e `POST .../deactivate` | `httpx` |
| `conversations.send_message`, `send_text` | `POST /conversations/{id}/messages` | `send_text` envia `text` (a rota espera `content`); a resposta não é lida | `httpx` |
| `conversations.assign` | `POST /conversations/{id}/assign` | Envia `agentId`; a API exige `user_id` | `httpx` |
| `ai.agents.*`, `ai.completions.*`, `ai.embeddings.*` | `/ai/agents`, `/ai/completions`, `/ai/embeddings` | Rotas não existem | `POST /ai/complete` e demais `/ai/*` com `httpx` |
| `knowledge_bases.query`, `search` | `POST .../query` | Rota não existe; a real é `POST .../search` com `query` e `limit` | `httpx` |
| `knowledge_bases.list_documents`, `upload_document`, `delete_document` | `.../documents` | Rotas não existem; a base usa `.../items` (pergunta e resposta) | `httpx` |
| `flows.execute` | `POST /flows/{id}/execute` | Rota não existe (há `POST /flows/{id}/test`) | `httpx` |
| `analytics.get_dashboard`, `get_message_metrics`, `get_realtime` | `/analytics/dashboard`, `/messages`, `/realtime` | Rotas não existem | `GET /analytics/overview` e demais |
| `vre.render_and_send` | `POST /vre/render-and-send` | Envia `conversation_id`; a rota exige `send_to`, `channel` e `channel_id` | `httpx` |
| `vre.list_templates` | `GET /vre/templates` | A API devolve lista de strings; o modelo espera objetos | `httpx` |
| `vre.preview` | `POST /vre/templates/{id}/preview` | A rota é `GET` | `httpx` |
| `verify_webhook`, `construct_event` e afins | Verificam a assinatura | Assinam só o corpo | Verificação manual acima |
| `LinktorAsyncClient` | Cria um `httpx.AsyncClient` | Não expõe nenhum recurso | `httpx.AsyncClient` |

Os modelos de conversa, mensagem, contato, bot e fluxo declaram aliases camelCase (`conversationId`,
`contentType`) e a API REST responde em snake_case; `PaginatedResponse` não corresponde ao `meta`.

## Referência: método → rota HTTP

Somente os métodos que funcionam; caminhos relativos a `/api/v1`. Comparação com os outros SDKs na
[visão geral dos SDKs](./visao-geral.md).

| Método | Rota | Observação |
|---|---|---|
| `channels.list()` | `GET /channels` | Falha se houver canal de tipo fora do enum |
| `channels.get(id)` | `GET /channels/{id}` | Tipos do enum |
| `channels.create(**kwargs)` | `POST /channels` | Tipos do enum |
| `channels.update(id, **kwargs)` | `PUT /channels/{id}` | Tipos do enum |
| `channels.delete(id)` | `DELETE /channels/{id}` | Qualquer tipo |
| `channels.connect(id)` | `POST /channels/{id}/connect` | Tipos do enum |
| `channels.request_pair_code(id, phone_number)` | `POST /channels/{id}/pair` | Só faz sentido em canais WhatsApp |
| `contacts.delete(id)`, `bots.delete(id)`, `flows.delete(id)` | `DELETE /contacts/{id}`, `/bots/{id}`, `/flows/{id}` | |
| `knowledge_bases.delete(id)` | `DELETE /knowledge-bases/{id}` | |
| `analytics.get_conversation_metrics(**params)` | `GET /analytics/conversations` | Devolve o envelope cru |
| `vre.render(...)` e atalhos `render_*` | `POST /vre/render` | Só com VRE habilitado |
| `auth.logout()` | `POST /auth/logout` | Só tem efeito com sessão de usuário |
