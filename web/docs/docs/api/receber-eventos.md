---
sidebar_position: 4
title: Receber eventos (webhook)
description: Como configurar o webhook de saída de um canal, os tipos de evento com payload de exemplo, verificação de assinatura HMAC, política de reentrega e um servidor receptor mínimo em cada linguagem.
---

# Receber eventos (webhook)

O Linktor avisa a sua aplicação por **HTTP POST assinado** sempre que algo acontece em um canal:
mensagem recebida, status de entrega, conversa criada, chamada perdida. O formato do corpo é o mesmo
para todos os tipos de canal (contrato `linktor-channel-v1`); só o campo `channelType` muda.

Resumo do contrato:

| Item | Valor |
|---|---|
| Método | `POST` com `Content-Type: application/json` |
| Configuração | **Por canal** (`webhook_url` do canal), não por tenant |
| Assinatura | `X-Linktor-Signature`: HMAC-SHA256 em hexadecimal de `timestamp + "." + corpo` |
| Anti-replay | `X-Linktor-Timestamp`: Unix em segundos; aceite até 300 s de diferença |
| Sucesso | Qualquer resposta `2xx` |
| Reentrega | Até 10 tentativas, depois fila de mensagens mortas (DLQ) |
| Garantia | Pelo menos uma vez — trate duplicatas pelo `id` do evento |

## Configurar {#configurar}

O webhook de saída é uma propriedade **de cada canal**. Três valores controlam a entrega:

| O quê | Onde fica no canal | Observação |
|---|---|---|
| URL do consumidor | `webhook_url` (campo de primeiro nível) | Sem URL, nenhum evento do canal é enviado |
| Segredo de assinatura | `credentials.webhook_secret` | Só escrita; nunca volta em respostas da API |
| Eventos a entregar | `config.webhook_events` | Lista separada por vírgula; vazio ou `*` = todos |

:::warning Sem segredo, não há entrega
Se o canal tem `webhook_url` mas não tem `webhook_secret`, o Linktor **descarta** as entregas em vez
de enviar um POST com assinatura forjável. O único sinal é uma linha de aviso no log do servidor.
Configure sempre os dois juntos.
:::

### Pelo painel

Na tela de configuração do canal, o bloco **Webhook de saída** tem os campos **URL do Consumidor**,
**Segredo de Assinatura** e **Eventos a entregar**. Se nenhum evento for marcado, todos são
entregues. Ao editar um canal, deixar o segredo em branco mantém o segredo atual. Veja
[Canais no painel](../painel/canais.md).

### Pela API

Envie os três valores ao criar (`POST /channels`) ou editar (`PUT /channels/:id`) o canal. Requer o
scope `channels:write`. O contrato completo dessas rotas está em [Canais](canais.md).

```bash
curl -X PUT "$LINKTOR_URL/api/v1/channels/CHANNEL_ID" \
  -H "X-API-Key: $LINKTOR_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "type": "whatsapp_official",
    "name": "WhatsApp Vendas",
    "webhook_url": "https://app.suaempresa.com/webhooks/linktor",
    "config": { "webhook_events": "message.received,message.failed" },
    "credentials": { "webhook_secret": "um-segredo-longo-e-aleatorio" }
  }'
```

Gere o segredo com um gerador aleatório (por exemplo `openssl rand -hex 32`) e guarde-o também na
sua aplicação — é com ele que você verifica a assinatura.

:::note Servidor com NATS
A entrega de webhooks roda sobre a fila durável do Linktor (NATS JetStream). Em uma instalação sem
NATS habilitado não há entrega de webhook de saída. Veja [Arquitetura](../implantar/arquitetura.md).
:::

## Cabeçalhos da requisição {#cabecalhos}

| Cabeçalho | Exemplo | Significado |
|---|---|---|
| `Content-Type` | `application/json` | O corpo é sempre JSON |
| `X-Linktor-Timestamp` | `1795528800` | Momento do envio, Unix em segundos. É renovado a cada tentativa |
| `X-Linktor-Signature` | `9f2c…e41a` (64 caracteres hex) | HMAC-SHA256 de `timestamp + "." + corpo`, sem prefixo |

## Envelope {#envelope}

Todo evento tem a mesma casca. Os nomes de campo do webhook são em **camelCase** (diferente da API
REST, que usa snake_case).

```json
{
  "id": "evt_0b0f4c1e-6a55-4a0e-9d3b-0c9a4f2a7e11_received",
  "type": "message.received",
  "timestamp": "2026-11-24T14:03:21.512345Z",
  "tenantId": "5d0c2a6e-1f7b-4a37-9d0e-6a1b9a3f2c10",
  "environment": "production",
  "data": {}
}
```

| Campo | Tipo | Descrição |
|---|---|---|
| `id` | string | Identificador do evento. Use para descartar duplicatas (ver [Idempotência](#idempotencia)) |
| `type` | string | Tipo do evento (lista abaixo) |
| `timestamp` | string RFC 3339 (UTC) | Quando o Linktor gerou o evento |
| `tenantId` | string | Tenant do Linktor dono do canal |
| `environment` | `production` ou `sandbox` | Ambiente do canal. Ausente equivale a `production` |
| `data` | objeto | Depende do `type` |

Campos novos podem ser acrescentados ao envelope e ao `data` sem aviso de quebra. Faça o seu parser
ignorar o que não conhece.

## Tipos de evento {#tipos}

| `type` | Quando acontece | Formato de `data` |
|---|---|---|
| `message.received` | Chegou uma mensagem (ou reação) em uma conversa do canal | [Mensagem](#message-received) |
| `message.sent` | O provedor aceitou uma mensagem enviada | [Status](#message-status) |
| `message.delivered` | A mensagem foi entregue ao destinatário | [Status](#message-status) |
| `message.read` | O destinatário leu a mensagem | [Status](#message-status) |
| `message.failed` | O envio falhou | [Status](#message-status) |
| `contact.created` | Um contato novo foi criado a partir do canal | [Contato](#contact-created) |
| `conversation.created` | Uma conversa foi aberta | [Conversa](#conversation) |
| `conversation.assigned` | A conversa foi atribuída a um atendente | [Conversa](#conversation) |
| `conversation.resolved` | A conversa foi resolvida | [Conversa](#conversation) |
| `conversation.reopened` | A conversa foi reaberta | [Conversa](#conversation) |
| `conversation.escalated` | A conversa foi escalada (do bot para atendimento humano) | [Conversa](#conversation) |
| `conversation.deleted` | A conversa foi removida | [Conversa](#conversation) |
| `call.answered` | Uma chamada anunciada foi atendida | [Chamada](#call) |
| `call.missed` | Uma chamada anunciada foi perdida | [Chamada](#call) |
| `call.rejected` | Uma chamada anunciada foi recusada | [Chamada](#call) |

:::note `conversation.deleted` e o painel
O servidor emite `conversation.deleted`, mas esse tipo não aparece na lista **Eventos a entregar**
do painel. Com a lista vazia (todos os eventos) ele é entregue normalmente. Para incluí-lo em uma
lista restrita, grave `config.webhook_events` pela API — e saiba que salvar o canal de novo pelo
painel remove o valor que a tela não conhece.
:::

### `message.received` {#message-received}

```json
{
  "id": "evt_0b0f4c1e-6a55-4a0e-9d3b-0c9a4f2a7e11_received",
  "type": "message.received",
  "timestamp": "2026-11-24T14:03:21.512345Z",
  "tenantId": "5d0c2a6e-1f7b-4a37-9d0e-6a1b9a3f2c10",
  "environment": "production",
  "data": {
    "message": {
      "id": "0b0f4c1e-6a55-4a0e-9d3b-0c9a4f2a7e11",
      "direction": "inbound",
      "contentType": "text",
      "content": { "text": "Olá, quero saber do meu pedido" },
      "senderId": "+5511999999999",
      "senderType": "contact",
      "metadata": {
        "senderName": "Maria Souza",
        "externalId": "wamid.HBgNNTUxMTk5OTk5OTk5ORUCABIYFjNFQjBDN0Q4RjQ5"
      }
    },
    "conversationId": "c1f0f5a2-7f1e-4c58-8a44-2b7f5a1d9e03",
    "contactId": "a7d3e0b4-52c1-4f0a-b6de-91c0a3f4e5d2",
    "channelId": "CHANNEL_ID",
    "channelType": "whatsapp_official"
  }
}
```

Campos de `data`:

| Campo | Descrição |
|---|---|
| `message.id` | Id da mensagem no Linktor (o mesmo de `GET /messages/:id`) |
| `message.direction` | `inbound` quando o contato escreveu; `outbound` quando foi o seu próprio lado (ver nota abaixo) |
| `message.contentType` | `text`, `image`, `video`, `audio`, `document`, `location`, `contact`, `template`, `interactive`, `sticker`, `poll`, `reaction` ou `call` |
| `message.content.text` | Texto da mensagem (ou legenda/descrição, conforme o canal). Omitido quando vazio |
| `message.content.media` | Presente quando há anexo — **somente o primeiro** anexo da mensagem |
| `message.senderId` | Endereço de quem escreveu no canal (telefone, e-mail, id do provedor) |
| `message.senderType` | `contact`, `user`, `bot` ou `system` |
| `message.metadata.senderName` | Nome do remetente informado pelo provedor, quando existe |
| `message.metadata.externalId` | Id da mensagem no provedor, quando existe |
| `message.mentions` | Lista de identificadores mencionados (grupos). Omitido quando não há menção |
| `message.reaction` | Presente quando o evento é uma reação (ver abaixo) |
| `conversationId`, `contactId` | Conversa e contato no Linktor |
| `channelId`, `channelType` | Canal de origem e o tipo dele |
| `group` | Presente só em conversa de grupo: `{ "id": "...", "name": "..." }` |
| `context` | Presente só em resposta citada correlacionada (ver [Bloco `context`](#context)) |

:::warning `message.received` nem sempre é o cliente falando
Em canais em que o aparelho pareado continua na mão do operador (WhatsApp não oficial, WhatsApp
Business App em coexistência), o que o operador digita no aparelho chega por este mesmo evento, com
`senderType` diferente de `contact` e `direction: "outbound"`. Antes de tratar o texto como fala do
cliente, confira `senderType == "contact"`.
:::

**Mensagem com mídia** — `content.media` descreve o anexo por URL:

```json
{
  "message": {
    "id": "3c9a4b1d-0e2f-4a6b-8c7d-5e4f3a2b1c0d",
    "direction": "inbound",
    "contentType": "image",
    "content": {
      "text": "Segue o comprovante",
      "media": {
        "url": "https://linktor.suaempresa.com/api/v1/media/inbound/whatsapp_official/CHANNEL_ID/7f3b2a.jpg",
        "mimeType": "image/jpeg",
        "filename": "comprovante.jpg",
        "size": 184213
      }
    },
    "senderId": "+5511999999999",
    "senderType": "contact"
  }
}
```

A URL é a referência durável da mídia e não carrega assinatura. Para baixar o arquivo, veja
[Mídia](midia.md).

**Mensagem de grupo** — o bloco `group` identifica o grupo e `message.senderId` continua sendo a
pessoa que falou:

```json
{
  "message": {
    "id": "8e1d2c3b-4a5f-4e6d-9c8b-7a6f5e4d3c2b",
    "direction": "inbound",
    "contentType": "text",
    "content": { "text": "@suporte alguém pode ajudar?" },
    "senderId": "5511999999999@s.whatsapp.net",
    "senderType": "contact",
    "mentions": ["5511888888888@s.whatsapp.net"]
  },
  "conversationId": "c1f0f5a2-7f1e-4c58-8a44-2b7f5a1d9e03",
  "contactId": "a7d3e0b4-52c1-4f0a-b6de-91c0a3f4e5d2",
  "channelId": "CHANNEL_ID",
  "channelType": "whatsapp",
  "group": { "id": "120363041234567890@g.us" }
}
```

`group.name` é preenchido quando o nome do grupo já foi resolvido; pode vir ausente.

**Reação** — quando `message.reaction` existe, aplique a reação à mensagem alvo em vez de criar uma
mensagem nova:

```json
{
  "message": {
    "id": "5a4b3c2d-1e0f-4a9b-8c7d-6e5f4a3b2c1d",
    "direction": "inbound",
    "contentType": "reaction",
    "content": { "text": "👍" },
    "senderId": "+5511999999999",
    "senderType": "contact",
    "reaction": {
      "emoji": "👍",
      "targetMessageId": "0b0f4c1e-6a55-4a0e-9d3b-0c9a4f2a7e11",
      "targetChannelMessageId": "wamid.HBgNNTUxMTk5OTk5OTk5ORUCABIYFjNFQjBDN0Q4RjQ5"
    }
  }
}
```

| Campo | Descrição |
|---|---|
| `reaction.emoji` | O emoji. **Vazio** significa que o contato removeu a reação |
| `reaction.targetMessageId` | Id da mensagem alvo no Linktor. Ausente quando o alvo não pôde ser resolvido (mensagem anterior à integração) |
| `reaction.targetChannelMessageId` | Id da mensagem alvo no provedor |

**Chamada** — uma chamada recebida é anunciada como `message.received` com `contentType: "call"`; o
desfecho chega depois em um evento `call.*` com o mesmo `messageId`.

### `message.sent`, `message.delivered`, `message.read`, `message.failed` {#message-status}

Os quatro compartilham o mesmo `data`. `messageId` é o `id` que você recebeu ao enviar (veja
[Enviar mensagens](enviar-mensagens.md)).

```json
{
  "id": "evt_9a8b7c6d-5e4f-4a3b-9c2d-1e0f9a8b7c6d_message.delivered",
  "type": "message.delivered",
  "timestamp": "2026-11-24T14:05:02.118004Z",
  "tenantId": "5d0c2a6e-1f7b-4a37-9d0e-6a1b9a3f2c10",
  "environment": "production",
  "data": {
    "messageId": "9a8b7c6d-5e4f-4a3b-9c2d-1e0f9a8b7c6d",
    "status": "delivered",
    "direction": "outbound",
    "timestamp": "2026-11-24T14:05:02.118004Z",
    "channelId": "CHANNEL_ID",
    "channelType": "whatsapp_official"
  }
}
```

Falha de envio — `error` traz o motivo informado pelo canal:

```json
{
  "id": "evt_9a8b7c6d-5e4f-4a3b-9c2d-1e0f9a8b7c6d_message.failed",
  "type": "message.failed",
  "timestamp": "2026-11-24T14:05:02.118004Z",
  "tenantId": "5d0c2a6e-1f7b-4a37-9d0e-6a1b9a3f2c10",
  "environment": "production",
  "data": {
    "messageId": "9a8b7c6d-5e4f-4a3b-9c2d-1e0f9a8b7c6d",
    "status": "failed",
    "direction": "outbound",
    "timestamp": "2026-11-24T14:05:02.118004Z",
    "error": "(#131047) Re-engagement message",
    "channelId": "CHANNEL_ID",
    "channelType": "whatsapp_official"
  }
}
```

No canal `whatsapp_official`, o status pode trazer o bloco `pricing` com o que a Meta informou sobre
a cobrança:

```json
{
  "id": "evt_9a8b7c6d-5e4f-4a3b-9c2d-1e0f9a8b7c6d_message.sent_pricing",
  "type": "message.sent",
  "timestamp": "2026-11-24T14:04:58.300120Z",
  "tenantId": "5d0c2a6e-1f7b-4a37-9d0e-6a1b9a3f2c10",
  "environment": "production",
  "data": {
    "messageId": "9a8b7c6d-5e4f-4a3b-9c2d-1e0f9a8b7c6d",
    "status": "sent",
    "direction": "outbound",
    "timestamp": "2026-11-24T14:04:58.300120Z",
    "channelId": "CHANNEL_ID",
    "channelType": "whatsapp_official",
    "pricing": { "billable": true, "pricingModel": "PMP", "category": "utility" }
  }
}
```

| Campo | Descrição |
|---|---|
| `messageId` | Id da mensagem no Linktor |
| `status` | `sent`, `delivered`, `read` ou `failed` |
| `direction` | Sempre `outbound` |
| `timestamp` | Quando o Linktor processou o status |
| `error` | Só em falha |
| `pricing` | Só `whatsapp_official`, e só quando a Meta informou. Ausência significa "não informado", não "gratuito". Os valores de `category` e `pricingModel` são os da Meta, sem tradução |

Dois pontos de atenção:

- A mesma mensagem pode gerar **dois** `message.sent`: o do aceite do envio e, depois, um com
  `pricing`. Eles têm `id` diferentes (o segundo termina em `_pricing`). `pricing` também pode
  repetir em mais de um status da mesma mensagem — contabilize uma vez por `messageId`.
- Os status podem chegar fora de ordem (`read` antes de `delivered`). Nunca regrida o estado.

### `contact.created` {#contact-created}

Emitido uma vez por contato, roteado para o webhook do canal em que o contato apareceu.

```json
{
  "id": "evt_a7d3e0b4-52c1-4f0a-b6de-91c0a3f4e5d2_contact.created",
  "type": "contact.created",
  "timestamp": "2026-11-24T14:03:21.401220Z",
  "tenantId": "5d0c2a6e-1f7b-4a37-9d0e-6a1b9a3f2c10",
  "environment": "production",
  "data": {
    "contactId": "a7d3e0b4-52c1-4f0a-b6de-91c0a3f4e5d2",
    "name": "Maria Souza",
    "phone": "+5511999999999",
    "email": "cliente@exemplo.com",
    "channelId": "CHANNEL_ID",
    "channelType": "whatsapp_official"
  }
}
```

`name`, `phone` e `email` são omitidos quando vazios.

### `conversation.*` {#conversation}

Os seis eventos de conversa têm o mesmo `data`:

```json
{
  "id": "evt_4f6a1c2e-9b3d-4e7f-8a1b-2c3d4e5f6a7b",
  "type": "conversation.assigned",
  "timestamp": "2026-11-24T14:10:45.902311Z",
  "tenantId": "5d0c2a6e-1f7b-4a37-9d0e-6a1b9a3f2c10",
  "environment": "production",
  "data": {
    "conversationId": "c1f0f5a2-7f1e-4c58-8a44-2b7f5a1d9e03",
    "channelId": "CHANNEL_ID",
    "contactId": "a7d3e0b4-52c1-4f0a-b6de-91c0a3f4e5d2",
    "status": "open",
    "assignedUserId": "2e7c9b1a-3d4f-4a5b-8c6d-7e8f9a0b1c2d",
    "channelType": "whatsapp_official"
  }
}
```

| Campo | Descrição |
|---|---|
| `conversationId`, `channelId`, `contactId` | Conversa, canal e contato |
| `status` | Estado da conversa quando informado: `open`, `pending`, `resolved` ou `closed` |
| `assignedUserId` | Atendente responsável, quando informado |

`status` e `assignedUserId` são omitidos quando o evento não os carrega. O `id` de
`conversation.created` e `conversation.deleted` é determinístico
(`evt_<conversationId>_<tipo>`); os de `assigned`, `resolved`, `reopened` e `escalated` são
aleatórios, porque essas transições podem se repetir legitimamente na mesma conversa.

Para as operações que disparam esses eventos, veja [Conversas](conversas.md).

### `call.*` {#call}

Desfecho único de uma chamada que já foi anunciada por `message.received` com
`contentType: "call"`. Não há duração: o dispositivo pareado não enxerga o fim da conversa.

```json
{
  "id": "evt_3EB0A1B2C3D4E5F6_call.rejected",
  "type": "call.rejected",
  "timestamp": "2026-11-24T15:20:11.004510Z",
  "tenantId": "5d0c2a6e-1f7b-4a37-9d0e-6a1b9a3f2c10",
  "environment": "production",
  "data": {
    "messageId": "6d5c4b3a-2f1e-4d0c-9b8a-7f6e5d4c3b2a",
    "callId": "3EB0A1B2C3D4E5F6",
    "state": "rejected",
    "from": "+5511999999999",
    "rejectedBy": "auto",
    "timestamp": "2026-11-24T15:20:11.004510Z",
    "conversationId": "c1f0f5a2-7f1e-4c58-8a44-2b7f5a1d9e03",
    "contactId": "a7d3e0b4-52c1-4f0a-b6de-91c0a3f4e5d2",
    "channelId": "CHANNEL_ID",
    "channelType": "whatsapp"
  }
}
```

| Campo | Descrição |
|---|---|
| `messageId` | A mensagem `message.received` que anunciou a chamada |
| `callId` | Id da chamada no provedor |
| `state` | `answered`, `missed` ou `rejected` |
| `from` | Quem ligou, quando conhecido |
| `rejectedBy` | Só em `call.rejected`: `auto`, `operator` ou `device` |

## Bloco `context`: correlação de resposta citada {#context}

Quando o contato **responde citando** uma mensagem que você enviou com metadata de correlação, o
`message.received` devolve essa correlação em `data.context`:

```json
{
  "type": "message.received",
  "data": {
    "message": {
      "id": "7b6a5c4d-3e2f-4a1b-9c0d-8e7f6a5b4c3d",
      "direction": "inbound",
      "contentType": "text",
      "content": { "text": "Pode seguir" },
      "senderId": "+5511999999999",
      "senderType": "contact"
    },
    "conversationId": "c1f0f5a2-7f1e-4c58-8a44-2b7f5a1d9e03",
    "contactId": "a7d3e0b4-52c1-4f0a-b6de-91c0a3f4e5d2",
    "channelId": "CHANNEL_ID",
    "channelType": "whatsapp",
    "context": { "alcada_correlation": "token-opaco", "source": "meu-sistema" }
  }
}
```

Regras:

- `context` só existe para **citação explícita** (o contato usou "responder" em cima da sua
  mensagem). O Linktor nunca infere correlação por telefone, conversa, horário ou texto.
- A chave que constitui a correlação é uma só e fixa: **`alcada_correlation`**. A mensagem citada
  precisa tê-la na `metadata` do envio; sem ela não há `context`.
- `source` é qualificador: viaja junto quando há correlação, mas sozinho nunca gera o bloco.
- Nenhuma outra chave da `metadata` do envio é devolvida. Um nome próprio (`meu_token`,
  `x-correlation`) fica gravado na mensagem enviada, mas não volta no webhook.
- A mensagem citada é procurada dentro da mesma conversa, então a correlação nunca cruza tenants.

Como anexar a `metadata` no envio está em [Enviar mensagens](enviar-mensagens.md).

## Verificar a assinatura {#assinatura}

Verifique **toda** requisição antes de confiar no conteúdo. O algoritmo:

1. Leia o **corpo cru** (os bytes exatos recebidos), antes de qualquer parser de JSON. Reserializar
   o JSON muda os bytes e a assinatura deixa de bater.
2. Leia `X-Linktor-Timestamp`. Rejeite se a diferença para o relógio atual passar de 300 segundos
   (para mais ou para menos).
3. Calcule `HMAC-SHA256(segredo, timestamp + "." + corpo)` e converta para hexadecimal minúsculo.
4. Compare com `X-Linktor-Signature` usando comparação em **tempo constante**.
5. Responda `401` quando não bater; só então interprete o JSON.

:::warning Verificadores dos SDKs
Na data desta página, apenas o SDK **Go** (`linktor.VerifyWebhook`, `linktor.ConstructEvent`)
implementa o algoritmo acima. Os verificadores dos SDKs TypeScript, Python, Java, .NET, PHP e Rust
calculam o HMAC **somente sobre o corpo**, sem o timestamp — eles **rejeitam** toda entrega real do
servidor. Nessas linguagens, use a implementação manual mostrada abaixo.
:::

## Servidor receptor mínimo {#receptor}

Cada exemplo abaixo é um servidor completo: lê o corpo cru, verifica timestamp e assinatura,
descarta duplicatas pelo `id` e responde `200` rápido. O segredo vem da variável de ambiente
`LINKTOR_WEBHOOK_SECRET`. A aba cURL simula uma entrega assinada, para você testar o seu receptor
sem depender de um canal real.

<Tabs groupId="linguagem">
<TabItem value="curl" label="cURL">

```bash
# Simula uma entrega assinada do Linktor para um receptor local.
SECRET="um-segredo-longo-e-aleatorio"
BODY='{"id":"evt_teste_1","type":"message.received","timestamp":"2026-11-24T14:03:21Z","tenantId":"t1","data":{"message":{"id":"m1","direction":"inbound","contentType":"text","content":{"text":"Olá"},"senderId":"+5511999999999","senderType":"contact"},"conversationId":"c1","contactId":"k1","channelId":"CHANNEL_ID","channelType":"whatsapp"}}'
TS=$(date +%s)
SIG=$(printf '%s.%s' "$TS" "$BODY" | openssl dgst -sha256 -hmac "$SECRET" | sed 's/^.*= //')

curl -i -X POST "http://localhost:3000/webhooks/linktor" \
  -H "Content-Type: application/json" \
  -H "X-Linktor-Timestamp: $TS" \
  -H "X-Linktor-Signature: $SIG" \
  --data-binary "$BODY"
```

</TabItem>
<TabItem value="typescript" label="TypeScript">

```typescript
// Express — npm install express
import express from 'express';
import { createHmac, timingSafeEqual } from 'node:crypto';

const SECRET = process.env.LINKTOR_WEBHOOK_SECRET!;
const TOLERANCE_SECONDS = 300;
const seen = new Set<string>(); // em produção: Redis ou tabela com chave única

function verify(rawBody: Buffer, timestamp?: string, signature?: string): boolean {
  if (!timestamp || !signature) return false;
  const ts = Number(timestamp);
  if (!Number.isInteger(ts)) return false;
  if (Math.abs(Math.floor(Date.now() / 1000) - ts) > TOLERANCE_SECONDS) return false;

  const expected = createHmac('sha256', SECRET)
    .update(timestamp)
    .update('.')
    .update(rawBody)
    .digest('hex');

  const a = Buffer.from(expected, 'utf8');
  const b = Buffer.from(signature, 'utf8');
  return a.length === b.length && timingSafeEqual(a, b);
}

const app = express();

// express.raw entrega o corpo como Buffer, sem reserializar o JSON.
app.post('/webhooks/linktor', express.raw({ type: 'application/json' }), (req, res) => {
  const raw = req.body as Buffer;
  if (!verify(raw, req.header('x-linktor-timestamp'), req.header('x-linktor-signature'))) {
    return res.status(401).send('assinatura inválida');
  }

  const event = JSON.parse(raw.toString('utf8'));
  if (seen.has(event.id)) return res.sendStatus(200); // duplicata
  seen.add(event.id);

  switch (event.type) {
    case 'message.received':
      if (event.data.message.senderType === 'contact') {
        console.log('mensagem de', event.data.message.senderId, event.data.message.content.text);
      }
      break;
    case 'message.failed':
      console.error('falha no envio', event.data.messageId, event.data.error);
      break;
  }

  res.sendStatus(200);
});

app.listen(3000);
```

</TabItem>
<TabItem value="python" label="Python">

```python
# FastAPI — pip install fastapi uvicorn
# Rodar: uvicorn receptor:app --port 3000
import hashlib
import hmac
import json
import os
import time

from fastapi import FastAPI, Request, Response

SECRET = os.environ["LINKTOR_WEBHOOK_SECRET"].encode()
TOLERANCE_SECONDS = 300
seen: set[str] = set()  # em produção: Redis ou tabela com chave única

app = FastAPI()


def verify(raw_body: bytes, timestamp: str | None, signature: str | None) -> bool:
    if not timestamp or not signature:
        return False
    try:
        ts = int(timestamp)
    except ValueError:
        return False
    if abs(int(time.time()) - ts) > TOLERANCE_SECONDS:
        return False
    expected = hmac.new(SECRET, timestamp.encode() + b"." + raw_body, hashlib.sha256).hexdigest()
    return hmac.compare_digest(expected, signature)


@app.post("/webhooks/linktor")
async def linktor_webhook(request: Request) -> Response:
    raw = await request.body()  # corpo cru, antes de qualquer parser
    if not verify(
        raw,
        request.headers.get("x-linktor-timestamp"),
        request.headers.get("x-linktor-signature"),
    ):
        return Response("assinatura inválida", status_code=401)

    event = json.loads(raw)
    if event["id"] in seen:
        return Response(status_code=200)  # duplicata
    seen.add(event["id"])

    if event["type"] == "message.received":
        message = event["data"]["message"]
        if message["senderType"] == "contact":
            print("mensagem de", message.get("senderId"), message["content"].get("text"))
    elif event["type"] == "message.failed":
        print("falha no envio", event["data"]["messageId"], event["data"].get("error"))

    return Response(status_code=200)
```

</TabItem>
<TabItem value="go" label="Go">

```go
// net/http, somente biblioteca padrão.
package main

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"sync"
	"time"
)

const toleranceSeconds = 300

var (
	secret = os.Getenv("LINKTOR_WEBHOOK_SECRET")
	seen   sync.Map // em produção: Redis ou tabela com chave única
)

func verify(rawBody []byte, timestamp, signature string) bool {
	if timestamp == "" || signature == "" || secret == "" {
		return false
	}
	ts, err := strconv.ParseInt(timestamp, 10, 64)
	if err != nil {
		return false
	}
	delta := time.Now().Unix() - ts
	if delta < 0 {
		delta = -delta
	}
	if delta > toleranceSeconds {
		return false
	}
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(timestamp))
	mac.Write([]byte("."))
	mac.Write(rawBody)
	expected := hex.EncodeToString(mac.Sum(nil))
	return hmac.Equal([]byte(expected), []byte(signature))
}

type event struct {
	ID   string          `json:"id"`
	Type string          `json:"type"`
	Data json.RawMessage `json:"data"`
}

func handler(w http.ResponseWriter, r *http.Request) {
	raw, err := io.ReadAll(io.LimitReader(r.Body, 1<<20)) // corpo cru
	if err != nil {
		http.Error(w, "corpo ilegível", http.StatusBadRequest)
		return
	}
	if !verify(raw, r.Header.Get("X-Linktor-Timestamp"), r.Header.Get("X-Linktor-Signature")) {
		http.Error(w, "assinatura inválida", http.StatusUnauthorized)
		return
	}

	var ev event
	if err := json.Unmarshal(raw, &ev); err != nil {
		http.Error(w, "JSON inválido", http.StatusBadRequest)
		return
	}
	if _, dup := seen.LoadOrStore(ev.ID, true); dup {
		w.WriteHeader(http.StatusOK) // duplicata
		return
	}

	switch ev.Type {
	case "message.received":
		var data struct {
			Message struct {
				SenderID   string `json:"senderId"`
				SenderType string `json:"senderType"`
				Content    struct {
					Text string `json:"text"`
				} `json:"content"`
			} `json:"message"`
		}
		if json.Unmarshal(ev.Data, &data) == nil && data.Message.SenderType == "contact" {
			log.Printf("mensagem de %s: %s", data.Message.SenderID, data.Message.Content.Text)
		}
	case "message.failed":
		log.Printf("falha no envio: %s", ev.Data)
	}

	w.WriteHeader(http.StatusOK)
}

func main() {
	http.HandleFunc("POST /webhooks/linktor", handler)
	log.Fatal(http.ListenAndServe(":3000", nil))
}
```

No Go, o SDK oficial faz a mesma verificação: `linktor.VerifyWebhook(raw, r.Header, secret, 0)`
(pacote `github.com/linktor/linktor-go`; `0` usa a tolerância padrão de 300 s).

</TabItem>
<TabItem value="java" label="Java">

```java
// Spring Boot (spring-boot-starter-web)
package com.suaempresa.webhooks;

import com.fasterxml.jackson.databind.JsonNode;
import com.fasterxml.jackson.databind.ObjectMapper;
import java.nio.charset.StandardCharsets;
import java.security.MessageDigest;
import java.util.HexFormat;
import java.util.Set;
import java.util.concurrent.ConcurrentHashMap;
import javax.crypto.Mac;
import javax.crypto.spec.SecretKeySpec;
import org.springframework.http.ResponseEntity;
import org.springframework.web.bind.annotation.PostMapping;
import org.springframework.web.bind.annotation.RequestBody;
import org.springframework.web.bind.annotation.RequestHeader;
import org.springframework.web.bind.annotation.RestController;

@RestController
public class LinktorWebhookController {

    private static final long TOLERANCE_SECONDS = 300;
    private final String secret = System.getenv("LINKTOR_WEBHOOK_SECRET");
    private final ObjectMapper mapper = new ObjectMapper();
    // Em produção: Redis ou tabela com chave única.
    private final Set<String> seen = ConcurrentHashMap.newKeySet();

    // byte[] entrega o corpo cru, sem passar pelo conversor de JSON.
    @PostMapping("/webhooks/linktor")
    public ResponseEntity<String> receive(
            @RequestBody byte[] rawBody,
            @RequestHeader(value = "X-Linktor-Timestamp", required = false) String timestamp,
            @RequestHeader(value = "X-Linktor-Signature", required = false) String signature)
            throws Exception {

        if (!verify(rawBody, timestamp, signature)) {
            return ResponseEntity.status(401).body("assinatura inválida");
        }

        JsonNode event = mapper.readTree(rawBody);
        if (!seen.add(event.get("id").asText())) {
            return ResponseEntity.ok().build(); // duplicata
        }

        switch (event.get("type").asText()) {
            case "message.received" -> {
                JsonNode message = event.at("/data/message");
                if ("contact".equals(message.path("senderType").asText())) {
                    System.out.println("mensagem de " + message.path("senderId").asText()
                            + ": " + message.at("/content/text").asText());
                }
            }
            case "message.failed" -> System.err.println("falha no envio: " + event.get("data"));
            default -> { }
        }
        return ResponseEntity.ok().build();
    }

    private boolean verify(byte[] rawBody, String timestamp, String signature) throws Exception {
        if (timestamp == null || signature == null || secret == null) return false;
        long ts;
        try {
            ts = Long.parseLong(timestamp);
        } catch (NumberFormatException e) {
            return false;
        }
        if (Math.abs(System.currentTimeMillis() / 1000 - ts) > TOLERANCE_SECONDS) return false;

        Mac mac = Mac.getInstance("HmacSHA256");
        mac.init(new SecretKeySpec(secret.getBytes(StandardCharsets.UTF_8), "HmacSHA256"));
        mac.update(timestamp.getBytes(StandardCharsets.UTF_8));
        mac.update((byte) '.');
        mac.update(rawBody);
        String expected = HexFormat.of().formatHex(mac.doFinal());

        return MessageDigest.isEqual(
                expected.getBytes(StandardCharsets.UTF_8),
                signature.getBytes(StandardCharsets.UTF_8));
    }
}
```

</TabItem>
<TabItem value="csharp" label="C#">

```csharp
// ASP.NET Core minimal API (.NET 8) — Program.cs
using System.Collections.Concurrent;
using System.Security.Cryptography;
using System.Text;
using System.Text.Json;

var builder = WebApplication.CreateBuilder(args);
var app = builder.Build();

var secret = Encoding.UTF8.GetBytes(
    Environment.GetEnvironmentVariable("LINKTOR_WEBHOOK_SECRET")!);
const long ToleranceSeconds = 300;
var seen = new ConcurrentDictionary<string, bool>(); // em produção: Redis ou tabela com chave única

bool Verify(byte[] rawBody, string? timestamp, string? signature)
{
    if (string.IsNullOrEmpty(timestamp) || string.IsNullOrEmpty(signature)) return false;
    if (!long.TryParse(timestamp, out var ts)) return false;
    if (Math.Abs(DateTimeOffset.UtcNow.ToUnixTimeSeconds() - ts) > ToleranceSeconds) return false;

    var prefix = Encoding.UTF8.GetBytes(timestamp + ".");
    var signed = new byte[prefix.Length + rawBody.Length];
    Buffer.BlockCopy(prefix, 0, signed, 0, prefix.Length);
    Buffer.BlockCopy(rawBody, 0, signed, prefix.Length, rawBody.Length);

    var expected = Convert.ToHexString(HMACSHA256.HashData(secret, signed)).ToLowerInvariant();
    return CryptographicOperations.FixedTimeEquals(
        Encoding.UTF8.GetBytes(expected), Encoding.UTF8.GetBytes(signature));
}

app.MapPost("/webhooks/linktor", async (HttpRequest request) =>
{
    // Corpo cru: lê o stream direto, sem model binding.
    using var buffer = new MemoryStream();
    await request.Body.CopyToAsync(buffer);
    var raw = buffer.ToArray();

    if (!Verify(raw,
            request.Headers["X-Linktor-Timestamp"].FirstOrDefault(),
            request.Headers["X-Linktor-Signature"].FirstOrDefault()))
    {
        return Results.Text("assinatura inválida", statusCode: 401);
    }

    using var doc = JsonDocument.Parse(raw);
    var root = doc.RootElement;
    var id = root.GetProperty("id").GetString()!;
    if (!seen.TryAdd(id, true)) return Results.Ok(); // duplicata

    switch (root.GetProperty("type").GetString())
    {
        case "message.received":
            var message = root.GetProperty("data").GetProperty("message");
            if (message.GetProperty("senderType").GetString() == "contact")
            {
                var text = message.GetProperty("content").TryGetProperty("text", out var t)
                    ? t.GetString() : null;
                Console.WriteLine($"mensagem: {text}");
            }
            break;
        case "message.failed":
            Console.Error.WriteLine($"falha no envio: {root.GetProperty("data")}");
            break;
    }

    return Results.Ok();
});

app.Run("http://0.0.0.0:3000");
```

</TabItem>
<TabItem value="php" label="PHP">

```php
<?php
// PHP puro — webhook.php
// Rodar para teste: php -S 0.0.0.0:3000 webhook.php

const TOLERANCE_SECONDS = 300;
$secret = getenv('LINKTOR_WEBHOOK_SECRET');

$raw = file_get_contents('php://input'); // corpo cru
$timestamp = $_SERVER['HTTP_X_LINKTOR_TIMESTAMP'] ?? '';
$signature = $_SERVER['HTTP_X_LINKTOR_SIGNATURE'] ?? '';

function verify(string $raw, string $timestamp, string $signature, string $secret): bool
{
    if ($timestamp === '' || $signature === '' || $secret === '') {
        return false;
    }
    if (!ctype_digit($timestamp) || abs(time() - (int) $timestamp) > TOLERANCE_SECONDS) {
        return false;
    }
    $expected = hash_hmac('sha256', $timestamp . '.' . $raw, $secret);
    return hash_equals($expected, $signature);
}

if (!verify($raw, $timestamp, $signature, (string) $secret)) {
    http_response_code(401);
    exit('assinatura inválida');
}

$event = json_decode($raw, true, 512, JSON_THROW_ON_ERROR);

// Idempotência: em produção use uma tabela com chave única ou Redis (SET NX).
$marker = sys_get_temp_dir() . '/linktor_evt_' . hash('sha256', $event['id']);
if (file_exists($marker)) {
    http_response_code(200);
    exit; // duplicata
}
touch($marker);

switch ($event['type']) {
    case 'message.received':
        $message = $event['data']['message'];
        if ($message['senderType'] === 'contact') {
            error_log('mensagem de ' . ($message['senderId'] ?? '?') . ': ' . ($message['content']['text'] ?? ''));
        }
        break;
    case 'message.failed':
        error_log('falha no envio: ' . json_encode($event['data']));
        break;
}

http_response_code(200);
```

</TabItem>
<TabItem value="rust" label="Rust">

```rust
// axum 0.7
// Cargo.toml: axum = "0.7", tokio = { version = "1", features = ["full"] },
//             hmac = "0.12", sha2 = "0.10", hex = "0.4", serde_json = "1"
use std::{
    collections::HashSet,
    sync::{Arc, Mutex},
    time::{SystemTime, UNIX_EPOCH},
};

use axum::{
    body::Bytes,
    extract::State,
    http::{HeaderMap, StatusCode},
    routing::post,
    Router,
};
use hmac::{Hmac, Mac};
use sha2::Sha256;

const TOLERANCE_SECONDS: i64 = 300;

#[derive(Clone)]
struct AppState {
    secret: Arc<String>,
    seen: Arc<Mutex<HashSet<String>>>, // em produção: Redis ou tabela com chave única
}

fn verify(raw: &[u8], timestamp: &str, signature: &str, secret: &str) -> bool {
    if timestamp.is_empty() || signature.is_empty() || secret.is_empty() {
        return false;
    }
    let Ok(ts) = timestamp.parse::<i64>() else { return false };
    let now = SystemTime::now().duration_since(UNIX_EPOCH).unwrap().as_secs() as i64;
    if (now - ts).abs() > TOLERANCE_SECONDS {
        return false;
    }
    let Ok(sig_bytes) = hex::decode(signature) else { return false };

    let mut mac = Hmac::<Sha256>::new_from_slice(secret.as_bytes()).unwrap();
    mac.update(timestamp.as_bytes());
    mac.update(b".");
    mac.update(raw);
    mac.verify_slice(&sig_bytes).is_ok() // comparação em tempo constante
}

fn header<'a>(headers: &'a HeaderMap, name: &str) -> &'a str {
    headers.get(name).and_then(|v| v.to_str().ok()).unwrap_or("")
}

// Bytes entrega o corpo cru, sem desserializar.
async fn receive(State(state): State<AppState>, headers: HeaderMap, raw: Bytes) -> StatusCode {
    if !verify(
        &raw,
        header(&headers, "x-linktor-timestamp"),
        header(&headers, "x-linktor-signature"),
        &state.secret,
    ) {
        return StatusCode::UNAUTHORIZED;
    }

    let Ok(event) = serde_json::from_slice::<serde_json::Value>(&raw) else {
        return StatusCode::BAD_REQUEST;
    };
    let id = event["id"].as_str().unwrap_or_default().to_string();
    if !state.seen.lock().unwrap().insert(id) {
        return StatusCode::OK; // duplicata
    }

    match event["type"].as_str() {
        Some("message.received") => {
            let message = &event["data"]["message"];
            if message["senderType"] == "contact" {
                println!("mensagem de {}: {}", message["senderId"], message["content"]["text"]);
            }
        }
        Some("message.failed") => eprintln!("falha no envio: {}", event["data"]),
        _ => {}
    }

    StatusCode::OK
}

#[tokio::main]
async fn main() {
    let state = AppState {
        secret: Arc::new(std::env::var("LINKTOR_WEBHOOK_SECRET").expect("LINKTOR_WEBHOOK_SECRET")),
        seen: Arc::new(Mutex::new(HashSet::new())),
    };
    let app = Router::new()
        .route("/webhooks/linktor", post(receive))
        .with_state(state);
    let listener = tokio::net::TcpListener::bind("0.0.0.0:3000").await.unwrap();
    axum::serve(listener, app).await.unwrap();
}
```

</TabItem>
</Tabs>

## Reentrega, timeout e DLQ {#reentrega}

| Regra | Valor |
|---|---|
| Sucesso | Resposta com status `2xx` |
| Falha | Qualquer outro status, erro de conexão, erro de TLS ou timeout |
| Timeout por tentativa | 30 segundos |
| Tentativas | Até **10** por evento |
| Intervalo | Cerca de 5 segundos entre tentativas (intervalo fixo, sem crescimento exponencial) |
| Depois da última tentativa | O evento vai para a fila de mensagens mortas (DLQ) e **não é reenviado sozinho** |
| Redirecionamentos, corpo da resposta | O corpo da sua resposta é ignorado |

O que isso significa na prática:

- A janela total de reentrega é curta — da ordem de um a poucos minutos. Um receptor que fica fora
  do ar por mais tempo **perde eventos** para a DLQ. Para se recuperar, reconcilie pela API:
  liste as conversas e as mensagens do período (veja [Conversas](conversas.md)).
- Cada tentativa é assinada de novo, com `X-Linktor-Timestamp` atual. Uma reentrega nunca cai fora
  da janela de 300 s por ser reentrega.
- A URL e o segredo são lidos do canal no momento de **cada** tentativa. Trocar o segredo vale para
  as entregas ainda na fila; remover a `webhook_url` ou apagar o canal descarta as pendentes.
- Responda rápido. Enfileire o processamento pesado do seu lado e devolva `200` em seguida: uma
  resposta lenta atrasa as próximas entregas, e uma que passe de 30 s vira reentrega.
- A ordem de chegada **não é garantida**. Use `timestamp` e as regras de estado (nunca regredir o
  status de uma mensagem) em vez de assumir sequência.
- A DLQ é operada por quem administra o servidor; o acompanhamento das filas está em
  [Observabilidade](../painel/observabilidade.md) e [Monitoramento](../implantar/monitoramento.md).

:::note Receptor na mesma máquina com certificado autoassinado
O Linktor valida o certificado TLS do seu endpoint. A única exceção é opcional e restrita: com
`LINKTOR_WEBHOOK_INSECURE_LOOPBACK=true` no servidor, a verificação é dispensada **somente** para
destinos em IP de loopback literal (`https://127.0.0.1:...`, `https://[::1]:...`). O nome
`localhost` não entra na exceção. Veja [Configuração](../implantar/configuracao.md).
:::

## Idempotência do lado do receptor {#idempotencia}

A entrega é **pelo menos uma vez**: se a sua aplicação processou o evento mas a resposta `2xx` não
chegou ao Linktor (timeout, queda de rede), o mesmo evento chega de novo. Trate assim:

1. Guarde o `id` do evento em um armazenamento com chave única (tabela com `UNIQUE`, `SET NX` no
   Redis) **na mesma transação** em que aplica o efeito.
2. Se o `id` já existe, responda `200` sem reprocessar.
3. Mantenha os ids por pelo menos alguns dias.

O `id` é estável entre reentregas do mesmo evento. Para os eventos de mensagem, contato, chamada e
para `conversation.created`/`conversation.deleted`, ele é derivado do recurso
(`evt_<id do recurso>_<tipo>`), então também colapsa uma eventual duplicação na origem.

Como segunda barreira, torne os efeitos naturalmente idempotentes: grave a mensagem recebida com
chave em `data.message.id`, e atualize status por `data.messageId` sem regredir.

## Solução de problemas {#problemas}

| Sintoma | Causa provável | O que fazer |
|---|---|---|
| Nenhum evento chega | Canal sem `webhook_url` ou sem `webhook_secret` | Configure os dois; sem segredo as entregas são descartadas |
| Chegam só alguns tipos | `config.webhook_events` restringe a lista | Deixe vazio para receber todos |
| A assinatura nunca bate | HMAC calculado só sobre o corpo, ou sobre o JSON reserializado | Assine `timestamp + "." + corpo cru`; confira o aviso sobre os SDKs |
| A assinatura bate em teste e falha em produção | Um proxy ou middleware alterou o corpo (descompressão, reformatação) | Capture os bytes antes de qualquer transformação |
| `401` por timestamp | Relógio do servidor receptor fora de sincronia | Sincronize por NTP; a tolerância é de 300 s |
| Eventos duplicados | Reentrega após timeout da sua resposta | Deduplique por `id`; responda em menos de 30 s |
| Falha de TLS, nada chega ao seu log | Certificado inválido ou autoassinado no endpoint | Use um certificado válido; em loopback, veja a nota acima |
