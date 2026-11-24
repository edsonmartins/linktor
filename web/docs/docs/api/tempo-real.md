---
sidebar_position: 11
title: Tempo real (WebSocket)
description: Como conectar ao WebSocket de atendentes em /api/v1/ws, autenticar com o token de usuário e tratar os eventos de mensagem, digitação e presença.
---

# Tempo real (WebSocket)

O endpoint `GET /api/v1/ws` é o canal de tempo real que o painel usa: ele empurra para os atendentes
conectados as mensagens novas, as atualizações de mensagem, a digitação e a presença dos colegas do
mesmo tenant.

```
wss://linktor.suaempresa.com/api/v1/ws
```

:::warning Para integração servidor a servidor, use o webhook
Este WebSocket é feito para **telas de atendimento**. Ele não tem reentrega, não guarda o que foi
emitido enquanto você estava desconectado, exige o **token de um usuário** (chave de API não
conecta) e mantém uma conexão por usuário. Para um sistema que precisa receber eventos com
garantia, use o [webhook de saída](receber-eventos.md).
:::

## Autenticação {#autenticacao}

A conexão é autenticada pelo **token de acesso (JWT) de um usuário**, o mesmo emitido em
`POST /auth/login` (veja [Autenticação](autenticacao.md)). O handshake passa por duas verificações,
e o token precisa satisfazer as duas:

| Cliente | Como enviar o token |
|---|---|
| Navegador, mesma origem do painel | Nada a fazer: o cookie HttpOnly `access_token` acompanha o handshake |
| Qualquer outro cliente (Node, Python, Go…) | Cabeçalho `Authorization: Bearer <token>` **e** parâmetro `?token=<token>` na URL |

O cabeçalho satisfaz a autenticação comum das rotas protegidas; o parâmetro `token` é o que o
handler do WebSocket lê para identificar o usuário. Enviar só um dos dois resulta em `401`.

:::note Chave de API não abre o WebSocket
Uma requisição com `X-API-Key` passa pela autenticação das rotas, mas o handler exige um JWT de
usuário e responde `401` com `{"error": "missing token"}`. Crie um usuário dedicado para a
integração e faça login com ele.
:::

Falhas no handshake respondem com HTTP `401` antes do upgrade:

| Corpo | Causa |
|---|---|
| `{"error": "missing token"}` | Sem `?token=` e sem cookie `access_token` |
| `{"error": "invalid token"}` | Token malformado, com assinatura inválida ou expirado |
| `{"error": "invalid token claims"}` | Token sem usuário ou tenant |
| `{"code": "UNAUTHORIZED", "message": "..."}` (ou `TOKEN_INVALID`) | Sem `Authorization`, cookie ou `X-API-Key` válidos — recusado pela autenticação comum, antes de chegar ao handler |

O token só é validado no handshake. Uma conexão aberta continua valendo depois que o token expira;
ao reconectar, use um token novo (renove com `POST /auth/refresh`).

### Origem (navegadores)

Quando o handshake traz o cabeçalho `Origin`, ele precisa ser um destes:

- a mesma origem pela qual o servidor foi acessado;
- uma origem listada em `LINKTOR_WS_ALLOWED_ORIGINS` (separadas por vírgula);
- `http://localhost:<porta>`, `http://127.0.0.1:<porta>` ou os equivalentes em `https`.

Clientes fora do navegador não enviam `Origin` e não são afetados. Veja
[Configuração](../implantar/configuracao.md).

## Formato das mensagens {#formato}

Toda mensagem, nos dois sentidos, é um quadro de texto com JSON neste formato:

```json
{ "type": "<tipo do evento>", "payload": {} }
```

Os nomes de campo seguem a API REST (snake_case).

## Eventos do servidor {#eventos-servidor}

| `type` | Quando | `payload` |
|---|---|---|
| `connected` | Logo após o handshake | Usuário, tenant e quem está online |
| `new_message` | Mensagem nova em qualquer conversa do tenant (recebida, enviada pelo painel ou pela API) | `conversation_id` e a mensagem |
| `message_updated` | Uma mensagem existente mudou: status de entrega, edição pelo contato, desfecho de chamada | `conversation_id` e a mensagem completa, já atualizada |
| `typing` | Outro atendente está digitando | Conversa, usuário e `is_typing` |
| `presence` | Um atendente conectou ou desconectou | Usuário e `status` |

Os eventos são do **tenant inteiro**: não há inscrição por conversa. Filtre no cliente por
`payload.conversation_id`.

:::note Tipos declarados e ainda não emitidos
O servidor também declara os tipos `conversation_updated`, `conversation_created` e `error`, mas
nenhum fluxo os emite hoje. Para acompanhar o ciclo de vida das conversas, use os eventos
`conversation.*` do [webhook](receber-eventos.md) ou consulte a API de [Conversas](conversas.md).
:::

### `connected`

```json
{
  "type": "connected",
  "payload": {
    "user_id": "2e7c9b1a-3d4f-4a5b-8c6d-7e8f9a0b1c2d",
    "tenant_id": "5d0c2a6e-1f7b-4a37-9d0e-6a1b9a3f2c10",
    "online_users": ["2e7c9b1a-3d4f-4a5b-8c6d-7e8f9a0b1c2d"]
  }
}
```

### `new_message` e `message_updated`

`payload.message` é o mesmo objeto de mensagem da API REST.

```json
{
  "type": "new_message",
  "payload": {
    "conversation_id": "c1f0f5a2-7f1e-4c58-8a44-2b7f5a1d9e03",
    "message": {
      "id": "0b0f4c1e-6a55-4a0e-9d3b-0c9a4f2a7e11",
      "conversation_id": "c1f0f5a2-7f1e-4c58-8a44-2b7f5a1d9e03",
      "sender_type": "contact",
      "content_type": "image",
      "content": "Segue o comprovante",
      "status": "delivered",
      "external_id": "wamid.HBgNNTUxMTk5OTk5OTk5ORUCABIYFjNFQjBDN0Q4RjQ5",
      "attachments": [
        {
          "id": "f1e2d3c4-b5a6-4978-8a9b-0c1d2e3f4a5b",
          "message_id": "0b0f4c1e-6a55-4a0e-9d3b-0c9a4f2a7e11",
          "type": "image",
          "filename": "comprovante.jpg",
          "mime_type": "image/jpeg",
          "size_bytes": 184213,
          "url": "https://linktor.suaempresa.com/api/v1/media/inbound/whatsapp_official/CHANNEL_ID/7f3b2a.jpg?exp=1796133600&sig=4c1f0e9a7b6d5c4e3f2a1b0c9d8e7f6a5b4c3d2e1f0a9b8c7d6e5f4a3b2c1d0e",
          "created_at": "2026-11-24T14:03:21.512345Z"
        }
      ],
      "created_at": "2026-11-24T14:03:21.512345Z"
    }
  }
}
```

| Campo de `message` | Valores |
|---|---|
| `sender_type` | `contact`, `user`, `bot`, `system` |
| `content_type` | `text`, `image`, `video`, `audio`, `document`, `location`, `contact`, `template`, `interactive`, `sticker`, `poll`, `reaction`, `call` |
| `status` | `pending`, `sent`, `delivered`, `read`, `failed` |
| `error_message` | Presente quando o envio falhou |
| `sent_at`, `delivered_at`, `read_at` | Presentes conforme o status avança |

As URLs em `attachments[].url` chegam **assinadas e com validade** quando o servidor tem assinatura
de mídia configurada — veja [Mídia](midia.md).

Em `message_updated` o formato é idêntico; substitua a mensagem que você tem pelo objeto recebido,
usando `message.id` como chave:

```json
{
  "type": "message_updated",
  "payload": {
    "conversation_id": "c1f0f5a2-7f1e-4c58-8a44-2b7f5a1d9e03",
    "message": {
      "id": "9a8b7c6d-5e4f-4a3b-9c2d-1e0f9a8b7c6d",
      "conversation_id": "c1f0f5a2-7f1e-4c58-8a44-2b7f5a1d9e03",
      "sender_type": "user",
      "content_type": "text",
      "content": "Seu pedido saiu para entrega",
      "status": "read",
      "sent_at": "2026-11-24T14:04:58Z",
      "delivered_at": "2026-11-24T14:05:02Z",
      "read_at": "2026-11-24T14:06:40Z",
      "created_at": "2026-11-24T14:04:57.801Z"
    }
  }
}
```

### `typing`

```json
{
  "type": "typing",
  "payload": {
    "conversation_id": "c1f0f5a2-7f1e-4c58-8a44-2b7f5a1d9e03",
    "user_id": "2e7c9b1a-3d4f-4a5b-8c6d-7e8f9a0b1c2d",
    "user_name": "",
    "is_typing": true
  }
}
```

`user_name` existe no formato, mas hoje chega vazio; resolva o nome pelo `user_id`. Você não recebe
o eco da sua própria digitação.

### `presence`

```json
{
  "type": "presence",
  "payload": {
    "user_id": "2e7c9b1a-3d4f-4a5b-8c6d-7e8f9a0b1c2d",
    "status": "offline",
    "last_seen": "2026-11-24T14:30:00Z"
  }
}
```

`status` é `online` ou `offline`. `last_seen` só acompanha `offline`.

## Mensagens do cliente {#mensagens-cliente}

O servidor trata um único tipo vindo do cliente: `typing`, que é repassado aos demais atendentes do
tenant.

```json
{
  "type": "typing",
  "payload": {
    "conversation_id": "c1f0f5a2-7f1e-4c58-8a44-2b7f5a1d9e03",
    "is_typing": true
  }
}
```

Qualquer outro tipo (ou JSON inválido) é ignorado em silêncio. Não existem mensagens de
`subscribe`, `unsubscribe` ou `ping` de aplicação. Para **enviar mensagens** a um contato, use a API
REST — veja [Enviar mensagens](enviar-mensagens.md).

## Manter a conexão viva {#keepalive}

| Regra | Valor |
|---|---|
| Ping do servidor | Quadro de controle `ping` do protocolo WebSocket a cada 54 segundos |
| Prazo de resposta | A conexão é encerrada se o servidor ficar 60 segundos sem receber `pong` |
| Tamanho máximo de mensagem do cliente | 512 KB |
| Conexões por usuário | **Uma**. Abrir uma segunda conexão com o mesmo usuário faz a primeira parar de receber eventos |

As bibliotecas dos exemplos abaixo respondem ao `ping` automaticamente. Do seu lado:

- Reconecte com espera crescente (por exemplo 1 s, 2 s, 4 s… até 30 s) quando a conexão cair.
- Depois de reconectar, **recarregue o estado pela API REST** (conversas e mensagens recentes): o
  que foi emitido durante a queda não é reenviado.
- Se um cliente lento acumular eventos demais na fila de saída, o servidor descarta os excedentes em
  vez de bloquear os outros clientes. Trate o WebSocket como aviso de "algo mudou", não como fonte
  única da verdade.
- Use um usuário por processo conectado. Dois processos com o mesmo usuário disputam a conexão.

## Exemplos {#exemplos}

Os três exemplos fazem login, conectam, tratam os eventos e reconectam em caso de queda. Usam
`LINKTOR_URL`, `LINKTOR_EMAIL` e `LINKTOR_PASSWORD`.

:::warning Cliente WebSocket dos SDKs
O cliente WebSocket do SDK TypeScript (`client.ws`) não segue este protocolo: envia a chave como
`?api_key=`, não envia o cabeçalho `Authorization`, espera eventos chamados `message`,
`message_status` e `conversation_update` e manda `subscribe`/`ping` que o servidor ignora. Use uma
biblioteca WebSocket diretamente, como abaixo.
:::

<Tabs groupId="linguagem">
<TabItem value="typescript" label="TypeScript">

```typescript
// Node.js — npm install ws
import WebSocket from 'ws';

const base = process.env.LINKTOR_URL!; // https://linktor.suaempresa.com

async function login(): Promise<string> {
  const res = await fetch(`${base}/api/v1/auth/login`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({
      email: process.env.LINKTOR_EMAIL,
      password: process.env.LINKTOR_PASSWORD,
    }),
  });
  if (!res.ok) throw new Error(`login falhou: ${res.status}`);
  const { data } = await res.json();
  return data.access_token;
}

async function connect(attempt = 0): Promise<void> {
  const token = await login();
  const url = `${base.replace(/^http/, 'ws')}/api/v1/ws?token=${encodeURIComponent(token)}`;
  const ws = new WebSocket(url, { headers: { Authorization: `Bearer ${token}` } });

  ws.on('open', () => {
    attempt = 0;
  });

  ws.on('message', (raw) => {
    const event = JSON.parse(raw.toString());
    switch (event.type) {
      case 'connected':
        console.log('online:', event.payload.online_users);
        break;
      case 'new_message':
        console.log('nova mensagem em', event.payload.conversation_id, event.payload.message.content);
        break;
      case 'message_updated':
        console.log('mensagem', event.payload.message.id, '->', event.payload.message.status);
        break;
      case 'typing':
        console.log(event.payload.user_id, event.payload.is_typing ? 'digitando' : 'parou');
        break;
      case 'presence':
        console.log(event.payload.user_id, event.payload.status);
        break;
    }
  });

  ws.on('close', () => {
    const delay = Math.min(30_000, 1000 * 2 ** attempt);
    setTimeout(() => connect(attempt + 1).catch(console.error), delay);
  });
  ws.on('error', (err) => console.error('ws:', err.message));

  // Avisar que está digitando em uma conversa:
  // ws.send(JSON.stringify({ type: 'typing', payload: { conversation_id: 'ID', is_typing: true } }));
}

connect().catch(console.error);
```

No navegador, na mesma origem do painel e com a sessão aberta, basta
`new WebSocket('wss://linktor.suaempresa.com/api/v1/ws')`: o cookie de sessão faz a autenticação. A
API de WebSocket do navegador não permite definir cabeçalhos, então não há como conectar a partir de
uma página sem esse cookie.

</TabItem>
<TabItem value="python" label="Python">

```python
# pip install websockets requests   (websockets >= 14)
import asyncio
import json
import os
from urllib.parse import quote

import requests
import websockets

BASE = os.environ["LINKTOR_URL"]  # https://linktor.suaempresa.com


def login() -> str:
    res = requests.post(
        f"{BASE}/api/v1/auth/login",
        json={"email": os.environ["LINKTOR_EMAIL"], "password": os.environ["LINKTOR_PASSWORD"]},
        timeout=30,
    )
    res.raise_for_status()
    return res.json()["data"]["access_token"]


async def listen() -> None:
    attempt = 0
    while True:
        try:
            token = login()
            url = BASE.replace("http", "ws", 1) + f"/api/v1/ws?token={quote(token)}"
            async with websockets.connect(
                url,
                additional_headers={"Authorization": f"Bearer {token}"},
            ) as ws:
                attempt = 0
                async for raw in ws:
                    event = json.loads(raw)
                    payload = event.get("payload") or {}
                    if event["type"] == "connected":
                        print("online:", payload["online_users"])
                    elif event["type"] == "new_message":
                        print("nova mensagem em", payload["conversation_id"], payload["message"].get("content"))
                    elif event["type"] == "message_updated":
                        print("mensagem", payload["message"]["id"], "->", payload["message"]["status"])
                    elif event["type"] == "typing":
                        print(payload["user_id"], "digitando" if payload["is_typing"] else "parou")
                    elif event["type"] == "presence":
                        print(payload["user_id"], payload["status"])
        except Exception as exc:  # queda de conexão, login recusado etc.
            print("ws:", exc)
        await asyncio.sleep(min(30, 2**attempt))
        attempt += 1


asyncio.run(listen())
```

Em versões do `websockets` anteriores à 14, o parâmetro se chama `extra_headers` em vez de
`additional_headers`.

</TabItem>
<TabItem value="go" label="Go">

```go
// go get github.com/gorilla/websocket
package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/gorilla/websocket"
)

type wsEvent struct {
	Type    string          `json:"type"`
	Payload json.RawMessage `json:"payload"`
}

type messagePayload struct {
	ConversationID string `json:"conversation_id"`
	Message        struct {
		ID      string `json:"id"`
		Content string `json:"content"`
		Status  string `json:"status"`
	} `json:"message"`
}

func login(base string) (string, error) {
	body, _ := json.Marshal(map[string]string{
		"email":    os.Getenv("LINKTOR_EMAIL"),
		"password": os.Getenv("LINKTOR_PASSWORD"),
	})
	resp, err := http.Post(base+"/api/v1/auth/login", "application/json", bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("login falhou: %d", resp.StatusCode)
	}
	var envelope struct {
		Data struct {
			AccessToken string `json:"access_token"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&envelope); err != nil {
		return "", err
	}
	return envelope.Data.AccessToken, nil
}

func listen(base string) error {
	token, err := login(base)
	if err != nil {
		return err
	}
	wsURL := strings.Replace(base, "http", "ws", 1) + "/api/v1/ws?token=" + url.QueryEscape(token)
	header := http.Header{"Authorization": {"Bearer " + token}}

	conn, _, err := websocket.DefaultDialer.Dial(wsURL, header)
	if err != nil {
		return err
	}
	defer conn.Close()

	// gorilla/websocket responde aos pings do servidor automaticamente
	// enquanto houver uma leitura em andamento.
	for {
		var ev wsEvent
		if err := conn.ReadJSON(&ev); err != nil {
			return err
		}
		switch ev.Type {
		case "new_message":
			var p messagePayload
			json.Unmarshal(ev.Payload, &p)
			log.Printf("nova mensagem em %s: %s", p.ConversationID, p.Message.Content)
		case "message_updated":
			var p messagePayload
			json.Unmarshal(ev.Payload, &p)
			log.Printf("mensagem %s -> %s", p.Message.ID, p.Message.Status)
		case "connected", "typing", "presence":
			log.Printf("%s: %s", ev.Type, ev.Payload)
		}
	}
}

func main() {
	base := os.Getenv("LINKTOR_URL") // https://linktor.suaempresa.com
	for attempt := 0; ; attempt++ {
		start := time.Now()
		err := listen(base)
		log.Println("ws:", err)
		if time.Since(start) > time.Minute {
			attempt = 0 // a conexão durou; recomeça a espera do zero
		}
		delay := time.Duration(1<<min(attempt, 5)) * time.Second
		time.Sleep(min(delay, 30*time.Second))
	}
}
```

</TabItem>
</Tabs>

## Outros WebSockets do servidor

Dois endpoints WebSocket do mesmo servidor têm outro propósito e outro protocolo, e não são
cobertos por esta página:

| Endpoint | Para quê |
|---|---|
| `/ws/:channelId` | Conexão do widget de chat no navegador do visitante. Veja [WebChat](../canais/webchat.md) |
| `/api/v1/gateways/ws` | Conexão dos dispositivos Linktor Bridge, autenticada pelo token do bridge |

## Solução de problemas {#problemas}

| Sintoma | Causa provável | O que fazer |
|---|---|---|
| `401` com `missing token` | Faltou o `?token=` (cliente fora do navegador) | Envie o token na URL **e** no cabeçalho `Authorization` |
| `401` com envelope de erro | Faltou o cabeçalho `Authorization` | Idem |
| `401` usando chave de API | O WebSocket exige JWT de usuário | Faça login com um usuário dedicado |
| `403` no handshake a partir de uma página | Origem não permitida | Inclua a origem em `LINKTOR_WS_ALLOWED_ORIGINS` |
| A conexão cai a cada minuto | O cliente não responde aos `ping` do protocolo | Use uma biblioteca que responda `pong` automaticamente; mantenha a leitura ativa |
| Um cliente para de receber eventos quando outro conecta | Mesmo usuário em duas conexões | Use um usuário por processo |
| Eventos perdidos após reconectar | Não há reenvio | Recarregue o estado pela API REST ao reconectar |
