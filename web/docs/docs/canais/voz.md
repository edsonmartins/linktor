---
sidebar_position: 11
title: Voz
description: O que o canal de voz do Linktor faz hoje — atendimento de chamadas recebidas pela Twilio com saudação e gravação —, o que ainda não está ligado, campos, webhook e API.
---

# Voz

O canal de voz registra **chamadas telefônicas recebidas** como mensagens de uma conversa. Hoje ele
funciona como uma **caixa de recados**: atende a chamada pela Twilio, fala uma saudação, grava o
recado e registra o andamento da chamada na conversa do contato.

Valor de `type` na API: **`voice`**.

:::warning Estado do canal: parcial
Leia isto antes de planejar qualquer coisa com voz. No servidor, o canal de voz hoje é **apenas o
webhook de chamadas recebidas da Twilio**. Não estão ligados:

- **chamadas de saída** e qualquer envio pelo canal (`POST /messages/send` não funciona com `voice`);
- **URA/IVR dinâmica** — menus, coleta de dígitos, texto para fala por mensagem, filas,
  transferência e conferência;
- os provedores **Vonage, Amazon Connect, Asterisk e FreeSWITCH**. Eles aparecem na tela e têm
  código no repositório, mas o webhook do servidor interpreta e valida toda requisição no formato da
  Twilio.

A tela do canal lista esses itens em **Recursos de IVR** e oferece os cinco provedores; trate isso
como intenção de produto, não como funcionalidade disponível.
:::

| Recurso | Situação |
|---|---|
| Atender chamada recebida (Twilio) | Sim — saudação falada seguida de gravação de até 120 segundos |
| Registrar início e fim da chamada na conversa | Sim |
| Gravação do recado como mensagem de áudio | Sim, com ressalva — veja [Limites e particularidades](#limites) |
| Transcrição do recado | Solicitada à Twilio, mas a chegada do texto não é garantida |
| Dígitos (DTMF) e fala reconhecida | O webhook entende, mas o atendimento padrão não os solicita |
| Chamadas de saída, texto para fala por mensagem | Não |
| URA com menus, filas, transferência, conferência | Não |
| Vonage, Amazon Connect, Asterisk, FreeSWITCH | Não |
| Texto, mídia, botões, modelos, grupos | Não se aplicam |

## Antes de começar

- Uma conta **Twilio** com um número habilitado para **voz**.
- O servidor do Linktor acessível pela internet em HTTPS.
- O armazenamento de mídia do servidor configurado, para guardar as gravações.
- Se a instalação restringe tipos de canal com `LINKTOR_ENABLED_CHANNEL_TYPES`, o tipo `voice`
  precisa estar na lista — o exemplo de configuração que acompanha o projeto **não** o inclui.

## Configurar na origem

Caminhos do Console da Twilio no momento em que esta página foi escrita; os menus mudam com
frequência.

1. Em [console.twilio.com](https://console.twilio.com), copie o **Account SID** e o **Auth Token**
   (painel inicial, **Account Info**).
2. Em **Phone Numbers → Manage → Active numbers**, abra o número que vai receber as chamadas (ou
   compre um com a capacidade **Voice**).
3. Crie o canal no Linktor (próxima seção) para obter o id do canal.
4. De volta ao número, em **Voice Configuration**:
   - **A call comes in:** escolha **Webhook**, cole a URL da seção [Webhook](#webhook) e deixe o
     método em **HTTP POST**.
   - **Call status changes:** cole a mesma URL, também com **HTTP POST**. É esse retorno que registra
     o fim da chamada.
5. Salve.

O método precisa ser **POST**: o Linktor lê os dados da chamada do corpo do formulário.

## Configurar no Linktor

1. No painel, abra **Canais** e, em **Adicionar Novo Canal**, clique em **Voz**.
2. Preencha **Nome do Canal** e, em **Provedor de Voz**, mantenha **Twilio Voice**.
3. Na aba **Credenciais**, preencha os campos da tabela.
4. Clique em **Criar Canal**.
5. No cartão do canal, abra o menu de ações e clique em **Conectar**. Enquanto o canal estiver
   desconectado, as chamadas são atendidas, mas nada é registrado na conversa.
6. Ainda no cartão, clique em **Copiar ID do canal** e monte a URL do webhook (seção abaixo).

| Campo na tela | Chave | Obrigatório | O que faz hoje |
|---|---|---|---|
| **Nome do Canal** | `name` | Sim | Nome interno. |
| **Provedor de Voz** | `config.provider` | Sim | Gravado. O webhook trata tudo como Twilio, seja qual for o valor. |
| **Account SID** | `credentials.account_sid` | Sim | Usado, com o Auth Token, para baixar a gravação da Twilio. |
| **Auth Token** | `credentials.auth_token` | Sim | Valida a assinatura do webhook e autentica o download da gravação. |
| **Número de Telefone** | `credentials.phone_number` | Sim (na tela) | Gravado; o servidor não o usa. |
| **URL do Webhook** | `credentials.webhook_url` e `config.webhook_url` | Não | Gravado; o servidor não o usa. Não é o webhook do Linktor. |
| **Gravar Chamadas** (aba **Configurações**) | `config.record_calls` | Não | Gravado, mas **sem efeito**: o atendimento sempre grava. |
| **Transcrever Chamadas** (aba **Configurações**) | `config.transcribe_calls` | Não | Gravado, mas **sem efeito**: a transcrição é sempre solicitada. |

Chave aceita apenas pela API:

| Chave | Efeito |
|---|---|
| `config.voice_greeting` | Texto falado ao atender. Padrão: `Hello. Please leave a message after the tone.` |

:::note Limitações da tela
- **Testar Conexão** sempre falha com `unsupported channel type: voice`: o servidor não tem teste
  para este tipo de canal. O erro não indica problema nas suas credenciais.
- A aba **Webhook** exibe o que foi digitado em **URL do Webhook** (ou **Não configurado**) — não a
  URL do Linktor. Use a URL desta página.
- Os campos dos outros provedores (Vonage, Amazon Connect, Asterisk, FreeSWITCH) são gravados em
  `credentials`, mas nada no servidor os utiliza.
:::

## Webhook

```text
https://linktor.suaempresa.com/api/v1/webhooks/voice/CHANNEL_ID
```

**Validação da assinatura.** Igual à do [SMS](sms.md): a Twilio assina a requisição no header
`X-Twilio-Signature`, e o Linktor recalcula o HMAC-SHA1 (Base64) sobre a URL completa mais os
parâmetros do formulário em ordem alfabética, com o `credentials.auth_token` do canal. Assinatura
inválida recebe `401 invalid signature`.

- A URL é reconstruída a partir de `X-Forwarded-Proto`, `Host` e do caminho. Atrás de proxy reverso,
  repasse o `Host` original e `X-Forwarded-Proto: https`.
- Sem Auth Token gravado, o webhook é aceito sem validação fora de produção; com
  `LINKTOR_SERVER_MODE=release` (ou `LINKTOR_REQUIRE_WEBHOOK_SECRETS=true`) a resposta é
  `401 webhook secret not configured`.

**O que o Linktor responde.** Na primeira requisição de uma chamada recebida (direção `inbound`,
status de início), a resposta é este TwiML:

```xml
<?xml version="1.0" encoding="UTF-8"?>
<Response>
  <Say>Hello. Please leave a message after the tone.</Say>
  <Record maxLength="120" transcribe="true" playBeep="true"/>
</Response>
```

Para todas as demais requisições (fim da gravação, mudanças de status), a resposta é um TwiML vazio.

**O que cada requisição vira na conversa.** O contato é identificado pelo número de quem ligou
(`From`); requisições sem esse número são ignoradas.

| Requisição da Twilio | Mensagem registrada | `contentType` |
|---|---|---|
| Início da chamada (`initiated`, `ringing`, `in-progress`) | `Incoming voice call` (precedido de um emoji de telefone); uma por chamada | `text` |
| Fim da chamada (`completed`) | `Voice call ended (42s)` | `text` |
| `busy`, `no-answer`, `failed`, `canceled` | `Voice call busy` (e equivalentes) | `text` |
| Gravação pronta (`RecordingUrl`) | `Voice recording`, com o áudio anexado | `audio` |
| Transcrição (`TranscriptionText`) | O texto transcrito | `text` |
| Dígitos (`Digits`) ou fala (`SpeechResult`) | O valor recebido | `text` |

Os textos são fixos, em inglês, e não são configuráveis.

## Criar o canal pela API

<Tabs groupId="linguagem">
<TabItem value="curl" label="cURL">

```bash
# 1. Cria o canal (nasce "disconnected")
curl -X POST "$LINKTOR_URL/api/v1/channels" \
  -H "X-API-Key: $LINKTOR_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "type": "voice",
    "name": "Linha de voz principal",
    "config": {
      "provider": "twilio",
      "voice_greeting": "Hello. Please leave a message after the tone."
    },
    "credentials": {
      "account_sid": "ACxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx",
      "auth_token": "seu-auth-token",
      "phone_number": "+5511999999999",
      "webhook_secret": "um-segredo-longo-e-aleatorio"
    },
    "webhook_url": "https://app.suaempresa.com/webhooks/linktor"
  }'

# 2. Conecta — use o "id" devolvido em data.id
curl -X POST "$LINKTOR_URL/api/v1/channels/$CHANNEL_ID/connect" \
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

const channel = await linktor.channels.create({
  "type": "voice",
  "name": "Linha de voz principal",
  "config": {
    "provider": "twilio",
    "voice_greeting": "Hello. Please leave a message after the tone."
  },
  "credentials": {
    "account_sid": "ACxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx",
    "auth_token": "seu-auth-token",
    "phone_number": "+5511999999999",
    "webhook_secret": "um-segredo-longo-e-aleatorio"
  },
  "webhook_url": "https://app.suaempresa.com/webhooks/linktor"
});

await linktor.channels.connect(channel.id);
console.log(channel.id);
```

</TabItem>
<TabItem value="python" label="Python">

```python
import os
import requests

base = os.environ["LINKTOR_URL"] + "/api/v1"
headers = {"X-API-Key": os.environ["LINKTOR_API_KEY"]}

# O enum de tipos do SDK Python ainda não inclui "voice": use HTTP direto.
resp = requests.post(f"{base}/channels", headers=headers, json={
  "type": "voice",
  "name": "Linha de voz principal",
  "config": {
    "provider": "twilio",
    "voice_greeting": "Hello. Please leave a message after the tone."
  },
  "credentials": {
    "account_sid": "ACxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx",
    "auth_token": "seu-auth-token",
    "phone_number": "+5511999999999",
    "webhook_secret": "um-segredo-longo-e-aleatorio"
  },
  "webhook_url": "https://app.suaempresa.com/webhooks/linktor"
})
resp.raise_for_status()
channel_id = resp.json()["data"]["id"]

requests.post(f"{base}/channels/{channel_id}/connect", headers=headers).raise_for_status()
print(channel_id)
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
	"github.com/linktor/linktor-go/types"
)

func main() {
	ctx := context.Background()
	client := linktor.NewClient(
		linktor.WithBaseURL(os.Getenv("LINKTOR_URL")+"/api/v1"),
		linktor.WithAPIKey(os.Getenv("LINKTOR_API_KEY")),
	)

	channel, err := client.Channels.Create(ctx, &types.CreateChannelInput{
		Name: "Linha de voz principal",
		Type: types.ChannelType("voice"),
		Config: map[string]string{
			"provider": "twilio",
			"voice_greeting": "Hello. Please leave a message after the tone.",
		},
		Credentials: map[string]string{
			"account_sid": "ACxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx",
			"auth_token": "seu-auth-token",
			"phone_number": "+5511999999999",
			"webhook_secret": "um-segredo-longo-e-aleatorio",
		},
		WebhookURL: "https://app.suaempresa.com/webhooks/linktor",
	})
	if err != nil {
		log.Fatal(err)
	}

	if _, err := client.Channels.Connect(ctx, channel.ID); err != nil {
		log.Fatal(err)
	}
	fmt.Println(channel.ID)
}
```

</TabItem>
<TabItem value="java" label="Java">

```java
import io.linktor.LinktorClient;
import io.linktor.types.Channel;
import java.util.Map;

LinktorClient linktor = LinktorClient.builder()
    .baseUrl(System.getenv("LINKTOR_URL") + "/api/v1")
    .apiKey(System.getenv("LINKTOR_API_KEY"))
    .build();

Channel.ChannelModel channel = linktor.channels.create(
    Channel.CreateChannelInput.builder()
        .type(Channel.ChannelType.VOICE)
        .name("Linha de voz principal")
        .config(Map.of(
            "provider", "twilio",
            "voice_greeting", "Hello. Please leave a message after the tone."))
        .credentials(Map.of(
            "account_sid", "ACxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx",
            "auth_token", "seu-auth-token",
            "phone_number", "+5511999999999",
            "webhook_secret", "um-segredo-longo-e-aleatorio"))
        .webhookUrl("https://app.suaempresa.com/webhooks/linktor")
        .build());

linktor.channels.connect(channel.getId());
System.out.println(channel.getId());
```

</TabItem>
<TabItem value="csharp" label="C#">

```csharp
using System.Net.Http;
using System.Text;
using System.Text.Json;

var baseUrl = Environment.GetEnvironmentVariable("LINKTOR_URL") + "/api/v1";
using var http = new HttpClient();
http.DefaultRequestHeaders.Add("X-API-Key", Environment.GetEnvironmentVariable("LINKTOR_API_KEY"));

var body = """
{
  "type": "voice",
  "name": "Linha de voz principal",
  "config": {
    "provider": "twilio",
    "voice_greeting": "Hello. Please leave a message after the tone."
  },
  "credentials": {
    "account_sid": "ACxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx",
    "auth_token": "seu-auth-token",
    "phone_number": "+5511999999999",
    "webhook_secret": "um-segredo-longo-e-aleatorio"
  },
  "webhook_url": "https://app.suaempresa.com/webhooks/linktor"
}
""";

var resp = await http.PostAsync($"{baseUrl}/channels",
    new StringContent(body, Encoding.UTF8, "application/json"));
resp.EnsureSuccessStatusCode();

using var doc = JsonDocument.Parse(await resp.Content.ReadAsStringAsync());
var channelId = doc.RootElement.GetProperty("data").GetProperty("id").GetString();

(await http.PostAsync($"{baseUrl}/channels/{channelId}/connect", null)).EnsureSuccessStatusCode();
Console.WriteLine(channelId);
```

</TabItem>
<TabItem value="php" label="PHP">

```php
<?php
require 'vendor/autoload.php';

$base = getenv('LINKTOR_URL') . '/api/v1';
$http = new GuzzleHttp\Client([
    'headers' => ['X-API-Key' => getenv('LINKTOR_API_KEY')],
]);

$resp = $http->post("$base/channels", ['json' => [
    'type' => 'voice',
    'name' => 'Linha de voz principal',
    'config' => [
        'provider' => 'twilio',
        'voice_greeting' => 'Hello. Please leave a message after the tone.',
    ],
    'credentials' => [
        'account_sid' => 'ACxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx',
        'auth_token' => 'seu-auth-token',
        'phone_number' => '+5511999999999',
        'webhook_secret' => 'um-segredo-longo-e-aleatorio',
    ],
    'webhook_url' => 'https://app.suaempresa.com/webhooks/linktor',
]]);
$channelId = json_decode((string) $resp->getBody(), true)['data']['id'];

$http->post("$base/channels/$channelId/connect");
echo $channelId, PHP_EOL;
```

</TabItem>
<TabItem value="rust" label="Rust">

```rust
use linktor::{ChannelType, CreateChannelInput, LinktorClient};
use std::collections::HashMap;

fn to_map(pairs: &[(&str, &str)]) -> HashMap<String, String> {
    pairs.iter().map(|(k, v)| (k.to_string(), v.to_string())).collect()
}

#[tokio::main]
async fn main() -> Result<(), Box<dyn std::error::Error>> {
    let client = LinktorClient::builder()
        .base_url(format!("{}/api/v1", std::env::var("LINKTOR_URL")?))
        .api_key(std::env::var("LINKTOR_API_KEY")?)
        .build()?;

    let input = CreateChannelInput::new("Linha de voz principal", ChannelType::Voice)
        .config(to_map(&[
            ("provider", "twilio"),
            ("voice_greeting", "Hello. Please leave a message after the tone."),
        ]))
        .credentials(to_map(&[
            ("account_sid", "ACxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx"),
            ("auth_token", "seu-auth-token"),
            ("phone_number", "+5511999999999"),
            ("webhook_secret", "um-segredo-longo-e-aleatorio"),
        ]))
        .webhook_url("https://app.suaempresa.com/webhooks/linktor");

    let channel = client.channels().create(input).await?;
    client.channels().connect(&channel.id).await?;
    println!("{}", channel.id);
    Ok(())
}
```

</TabItem>
</Tabs>

Depois de criar e conectar, cadastre a URL do webhook no número da Twilio. `webhook_url` e
`credentials.webhook_secret` são o endereço e o segredo da **sua aplicação** para os eventos do
Linktor (não confunda com `credentials.webhook_url`, um campo da tela sem uso no servidor).

:::warning PUT substitui o config
No `PUT /channels/CHANNEL_ID`, o `config` enviado substitui o anterior por inteiro; só `credentials`
é mesclado. Para mudar a saudação, reenvie `provider` e `voice_greeting` juntos.
:::

## Enviar e receber

### Enviar

**Não há envio neste canal.** O servidor não tem um emissor registrado para `voice`: uma chamada a
`POST /messages/send` (ou uma resposta escrita no painel) numa conversa de voz é aceita na fila, mas
nunca vira ligação nem áudio: o envio para em `no outbound sender for channel type voice`. Por isso esta página não traz exemplo de envio.

Para responder a quem deixou um recado, use outro canal com o mesmo telefone (o `senderId` do
evento), por exemplo [SMS](sms.md) ou [WhatsApp](whatsapp-oficial.md).

### Receber

Cada linha da tabela da seção [Webhook](#webhook) gera um `message.received` no `webhook_url` do
canal. O recado gravado chega assim:

```json
{
  "id": "evt_8d2f6f0a-93c1-4b7e-a7f5-0c9e1d2b3a4c_received",
  "type": "message.received",
  "timestamp": "2026-11-24T14:03:27Z",
  "tenantId": "0b6f7c1e-2a44-4f6b-9d0e-6a1b2c3d4e5f",
  "environment": "production",
  "data": {
    "message": {
      "id": "8d2f6f0a-93c1-4b7e-a7f5-0c9e1d2b3a4c",
      "direction": "inbound",
      "contentType": "audio",
      "content": {
        "text": "Voice recording",
        "media": {
          "url": "https://linktor.suaempresa.com/media/recado.mp3",
          "mimeType": "audio/mpeg"
        }
      },
      "senderId": "+5511999999999",
      "senderType": "contact",
      "metadata": { "externalId": "CA0123456789abcdef0123456789abcdef:recording" }
    },
    "conversationId": "c1a5d3e2-7b84-4a60-8e55-2f1d9b0c7a66",
    "contactId": "7e9a4b10-5c2d-4f3e-b8a1-6d0f2e3c4b5a",
    "channelId": "CHANNEL_ID",
    "channelType": "voice"
  }
}
```

- `senderId` é o número de quem ligou.
- `metadata.externalId` é o `CallSid` da Twilio seguido do tipo de evento (`call_started`,
  `call_ended`, `call_status`, `recording`, `transcription`, `dtmf`, `speech`). Use o `CallSid`
  (a parte antes dos dois-pontos) para agrupar os eventos de uma mesma chamada.
- A forma da URL de mídia depende do armazenamento do servidor. As gravações da Twilio exigem
  autenticação; o Linktor as baixa com o Account SID e o Auth Token do canal e as re-hospeda.
- Os eventos `call.answered`, `call.missed` e `call.rejected` do contrato de webhook pertencem às
  chamadas do WhatsApp, **não** a este canal.
- Assinatura e headers do evento em [Receber eventos](../api/receber-eventos.md).

## Limites e particularidades {#limites}

- **Saudação.** O texto de `config.voice_greeting` é falado com a voz e o idioma padrão da Twilio
  (inglês): um texto em português é lido com pronúncia inglesa. Não há como escolher voz ou idioma.
- **Recado de até 120 segundos**, com bipe. O limite não é configurável.
- **Gravação possivelmente registrada como dígito.** Ao terminar a gravação, a Twilio envia
  `RecordingUrl` junto com `Digits` (a tecla que encerrou o recado, ou `hangup` quando a pessoa
  desliga). Como o Linktor verifica `Digits` antes de `RecordingUrl`, essa requisição pode ser
  registrada como uma mensagem de texto (`hangup`, `#`) em vez do áudio. Esse caminho não foi
  validado de ponta a ponta: **teste com uma chamada real** antes de depender da gravação.
- **Transcrição.** O atendimento pede transcrição à Twilio, mas não informa um endereço de retorno
  específico para ela; na prática o texto pode não chegar. A transcrição da Twilio também só atende
  inglês.
- **Sem duplicatas de início.** Os vários retornos que significam "a chamada começou" viram uma
  única mensagem por chamada.
- **Sem roteamento de chamada.** A chamada não toca para um atendente: ela é sempre atendida pela
  saudação e pela gravação.

## Solução de problemas

| Sintoma | Causa provável | O que fazer |
|---|---|---|
| Quem liga ouve uma mensagem de erro da Twilio | Webhook não cadastrado, URL errada ou método diferente de POST | Em **Voice Configuration**, use a URL da seção [Webhook](#webhook) com **HTTP POST**. |
| Webhook responde `401 invalid signature` | Proxy alterando `Host` ou sem `X-Forwarded-Proto`; Auth Token desatualizado | Ajuste o proxy; regrave o **Auth Token** no canal. |
| Webhook responde `401 webhook secret not configured` | Produção, canal sem Auth Token | Grave o **Auth Token**. |
| Chamada atendida, mas nada aparece no painel | Requisição sem número de origem (chamada anônima) | Sem `From` o Linktor não tem como identificar o contato. |
| A conversa registra o início, mas não o fim | **Call status changes** não aponta para o webhook | Cadastre a mesma URL nesse campo. |
| Aparece uma mensagem `hangup` em vez do áudio | Comportamento descrito em [Limites e particularidades](#limites) | Limitação atual; a gravação continua disponível no Console da Twilio. |
| **Testar Conexão** falha | O servidor não testa o tipo `voice` | Ignore; valide com uma chamada. |
| Criação do canal recusada | `voice` fora de `LINKTOR_ENABLED_CHANNEL_TYPES` | Inclua o tipo (veja [Configuração](../implantar/configuracao.md)). |
| Respostas escritas no painel falham | Não há envio no canal de voz | Responda por outro canal. |
| O webhook é chamado, mas nada aparece no painel | Canal desconectado ou desabilitado: o recebimento é descartado (`channel is not active`) | Clique em **Conectar** e confira se o canal está habilitado. |
| Nenhum evento chega na sua aplicação | Canal sem `webhook_url` ou sem `credentials.webhook_secret` | Defina os dois pela API. |

Veja também: [Visão geral dos canais](visao-geral.md) · [SMS](sms.md) ·
[Canais no painel](../painel/canais.md).
