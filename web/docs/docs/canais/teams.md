---
sidebar_position: 13
title: Microsoft Teams
description: Conectar um bot do Microsoft Teams (Azure Bot / Bot Framework) ao Linktor — registro no Azure, endpoint de mensagens, manifesto, campos do canal, envio e recebimento.
---

# Microsoft Teams

## Resumo

O canal Teams liga um **bot do Bot Framework** (recurso Azure Bot) ao Linktor. O Teams entrega
cada mensagem como uma *Activity* em um `POST` para o Linktor, autenticado por um JWT do Bot
Connector. As respostas saem pela API REST do Bot Connector
(`{serviceUrl}/v3/conversations/{id}/activities`) com um token do Microsoft Entra ID obtido por
*client credentials*.

- Valor de `type` na API: `teams`
- Como conecta: App ID + segredo do cliente (+ Tenant ID em bots de um único tenant)
- Endereço do contato: o **id da conversa** do Bot Framework (`a:…` em conversa 1:1,
  `19:…@thread…` em chat de grupo ou canal de equipe)

| Recurso | Enviar | Receber | Observação |
|---|---|---|---|
| Texto | Sim | Sim | |
| Imagem, áudio, vídeo, documento | Por link | Sim | No envio vai um anexo apontando para a URL; não há upload nativo |
| Localização | Não | Não | |
| Botões, listas e Adaptive Cards | Não | Não | Mensagem com botões falha no envio — veja [Limites](#limites) |
| Modelos (templates) | Vira texto | — | Os parâmetros são unidos em texto |
| Chats de grupo e canais de equipe | Sim | Sim | Cada conversa vira **um** contato no Linktor |
| Reação | Não | Não | |
| Confirmação de entrega e leitura | Não | — | Só `message.sent` e `message.failed` |

## Antes de começar

- Uma assinatura do Azure com permissão para criar o recurso **Azure Bot** e um registro de
  aplicativo no Microsoft Entra ID.
- Permissão para instalar apps personalizados no Teams (ou um administrador do Teams que publique
  o app para a organização).
- O Linktor publicado em HTTPS com endereço público.
- No Linktor, um usuário do painel ou uma chave de API com os scopes `channels:write` e
  `messages:send`. Veja [Chaves de API](../painel/chaves-de-api.md).

## Configurar na origem

Os passos abaixo são feitos no [portal do Azure](https://portal.azure.com) e no Developer Portal
do Teams. Os nomes de menu são os da Microsoft e mudam com frequência.

### 1. Registrar o bot no Azure

1. No portal do Azure, crie um recurso **Azure Bot**.
2. Em **Type of App**, escolha **Single Tenant** (o bot atende só a sua organização) ou
   **Multi Tenant** (um mesmo bot atende várias organizações). A Microsoft vem restringindo a
   criação de bots multi-tenant novos; se a opção não aparecer, use single-tenant.
3. Deixe o Azure criar o registro de aplicativo (**Create new Microsoft App ID**) ou informe um
   existente.
4. Depois de criado, abra o recurso → **Configuration** e anote:
   - **Microsoft App ID** — é o `app_id`;
   - **App Tenant ID** — é o `tenant_id` (só em single-tenant).
5. Ainda em **Configuration**, clique em **Manage Password** (abre **Certificates & secrets** do
   registro de aplicativo) → **New client secret**. Copie a coluna **Value** — ela só aparece uma
   vez. É o `app_password`. Não confunda com **Secret ID**.
6. Em **Channels**, adicione o canal **Microsoft Teams** e aceite os termos.

### 2. Apontar o endpoint de mensagens

Crie o canal no Linktor (próxima seção) para obter a URL. Depois, no recurso Azure Bot →
**Configuration** → **Messaging endpoint**, cole a URL do webhook e clique em **Apply**.

### 3. Publicar o app no Teams

O bot só aparece para os usuários dentro de um app do Teams.

1. No **Developer Portal** do Teams (`dev.teams.microsoft.com`), crie um app em **Apps** →
   **New app**.
2. Em **App features** → **Bot**, escolha **Enter a bot ID** e informe o Microsoft App ID. Marque
   os escopos em que o bot vai atender: **Personal** (conversa 1:1), **Team** (canais de equipe)
   e **Group chat**.
3. Preencha as informações básicas exigidas (nome, descrições, ícones, URLs do desenvolvedor).
4. Publique: **Publish** → **Publish to org** (um administrador aprova no Teams Admin Center) ou
   baixe o pacote e instale com **Upload a custom app**.

Quem prefere montar o pacote à mão usa um `manifest.json` com a seção `bots` apontando para o
mesmo App ID:

```json
{
  "$schema": "https://developer.microsoft.com/json-schemas/teams/v1.16/MicrosoftTeams.schema.json",
  "manifestVersion": "1.16",
  "version": "1.0.0",
  "id": "00000000-0000-0000-0000-000000000000",
  "packageName": "com.suaempresa.atendimento",
  "developer": {
    "name": "Sua Empresa",
    "websiteUrl": "https://suaempresa.com",
    "privacyUrl": "https://suaempresa.com/privacidade",
    "termsOfUseUrl": "https://suaempresa.com/termos"
  },
  "name": { "short": "Atendimento", "full": "Atendimento Sua Empresa" },
  "description": {
    "short": "Fale com o atendimento",
    "full": "Canal de atendimento da Sua Empresa dentro do Teams."
  },
  "icons": { "color": "color.png", "outline": "outline.png" },
  "accentColor": "#FFFFFF",
  "bots": [
    {
      "botId": "00000000-0000-0000-0000-000000000000",
      "scopes": ["personal", "team", "groupchat"],
      "supportsFiles": false,
      "isNotificationOnly": false
    }
  ],
  "validDomains": []
}
```

O pacote é um `.zip` com `manifest.json`, `color.png` (192×192) e `outline.png` (32×32).

## Configurar no Linktor

1. Abra **Canais** e, em **Adicionar Novo Canal**, escolha **Microsoft Teams**.
2. Preencha os campos da tabela abaixo.
3. Clique em **Testar Conexão**. O Linktor pede um token ao Microsoft Entra ID com o App ID, o
   segredo e o tenant informados; se o token vier, as três informações estão corretas.
4. Clique em **Criar Canal**.
5. Reabra o canal, copie a **URL do Webhook** e cole-a em **Messaging endpoint** no Azure.
6. No cartão do canal, clique em **Conectar**. Enquanto o canal não estiver **Conectado** e
   habilitado, o envio é recusado.
7. Mande uma mensagem ao bot pelo Teams. A primeira mensagem recebida ensina ao Linktor o
   endereço de resposta (`service_url`) — veja [Limites](#limites).

| Rótulo na tela | Chave | Onde fica | Obrigatório |
|---|---|---|---|
| **Nome do Canal** | `name` | campo do canal | Sim |
| **App ID** | `app_id` | `credentials` | Sim |
| **Senha do App** | `app_password` | `credentials` | Sim |
| **Tenant ID** | `tenant_id` | `credentials` | Em single-tenant, sim; em multi-tenant, vazio ou `common` |
| — (sem campo na tela) | `service_url` | `credentials` | Não; preenchido sozinho na primeira mensagem recebida |
| **URL do Consumidor** (em **Webhook de saída**) | `webhook_url` | campo do canal | Não |
| **Segredo de Assinatura** (em **Webhook de saída**) | `webhook_secret` | `credentials` | Só se houver URL do Consumidor |

Tudo fica em `credentials`, cifrado no banco. `app_password` e `webhook_secret` nunca voltam pela
API; ao editar, deixá-los em branco mantém o valor guardado. `app_id`, `tenant_id` e `service_url`
não são segredo e voltam em `config` na consulta de um canal, só para preencher o formulário.

:::warning Tenant ID e o token de envio
O Linktor pede o token de envio em
`login.microsoftonline.com/<tenant_id>/oauth2/v2.0/token`. Com **Tenant ID** vazio ou `common`,
usa o tenant `botframework.com`, que só serve a bots multi-tenant. Um bot **single-tenant** sem
Tenant ID falha no teste de conexão e em todo envio.
:::

:::note Filtro de eventos do webhook de saída
A tela do Teams não tem a lista **Eventos a entregar**. Para limitar os eventos enviados ao seu
sistema, grave `config.webhook_events` pela API (lista separada por vírgula). Salvar o canal pela
tela depois disso apaga esse valor, porque o formulário envia `config` vazio.
:::

### Testar pela API

```bash
curl -X POST "$LINKTOR_URL/api/v1/channels/test" \
  -H "X-API-Key: $LINKTOR_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "type": "teams",
    "credentials": {
      "app_id": "00000000-0000-0000-0000-000000000000",
      "app_password": "valor_do_segredo_do_cliente",
      "tenant_id": "11111111-1111-1111-1111-111111111111"
    }
  }'
```

```json
{
  "success": true,
  "data": {
    "status": "ok",
    "type": "teams",
    "valid": true,
    "message": "credentials valid (AAD token acquired)"
  }
}
```

Para testar um canal já salvo sem redigitar o segredo, envie `"channel_id": "CHANNEL_ID"` no
corpo. `POST /api/v1/channels/test-teams` é o mesmo teste com o tipo fixo. O limite é de 10
segundos.

## Webhook

Há duas URLs, uma para cada modelo de bot. Ambas aceitam só `POST`.

### Uma URL por canal (recomendado)

```text
https://linktor.suaempresa.com/api/v1/webhooks/teams/CHANNEL_ID
```

É a URL que a tela mostra. Use-a quando o registro do bot pertence a um único canal do Linktor —
o caso de um bot single-tenant.

### URL compartilhada (um bot, vários canais)

```text
https://linktor.suaempresa.com/api/v1/webhooks/teams
```

Um bot do Azure tem um único *messaging endpoint*. Quando o mesmo App ID atende várias
organizações — cada uma com seu canal no Linktor — aponte o bot para esta URL, sem id de canal. O
Linktor escolhe o canal assim:

1. lê o App ID na audiência (`aud`) do token recebido;
2. procura, entre os canais `teams` com esse `app_id`, um cujo `tenant_id` seja igual ao tenant da
   organização que mandou a mensagem (`channelData.tenant.id` da Activity);
3. se não houver, usa o canal desse `app_id` com `tenant_id` vazio, `common`, `organizations` ou
   `botframework.com`;
4. se nenhum servir, responde `404`.

:::warning Limite de isolamento da URL compartilhada
O token do Bot Connector prova o emissor e o App ID, mas **não** o tenant da organização — esse
valor vem do corpo da requisição. Canais que compartilham o mesmo registro de bot estão dentro de
uma mesma fronteira de confiança. Para isolamento estrito entre organizações, use um registro de
bot por canal e a URL com `CHANNEL_ID`.
:::

### Como a chamada é validada

Em ambas as URLs o Linktor valida o header `Authorization: Bearer <jwt>` enviado pelo Bot
Connector:

- assinatura RS256, com as chaves públicas publicadas em
  `login.botframework.com/v1/.well-known/openidconfiguration` (em cache por 24 horas);
- emissor `https://api.botframework.com`;
- audiência igual ao `app_id` do canal;
- validade.

Token inválido → `401 invalid bot connector token`. Na URL por canal, um canal sem `app_id` aceita
a chamada sem validar, exceto em produção (`release`, ou
`LINKTOR_REQUIRE_WEBHOOK_SECRETS=true`), onde responde `401`.

Só Activities do tipo `message` viram mensagem. As demais (`conversationUpdate`, `typing`,
instalação do bot) são aceitas com `200` e ignoradas.

## Criar o canal pela API

`POST /api/v1/channels` com o scope `channels:write`. Os SDKs não acrescentam `/api/v1` à base:
informe o endereço completo. Onde o SDK da linguagem ainda não cobre este tipo de canal, o exemplo
usa o cliente HTTP nativo.

<Tabs groupId="linguagem">
<TabItem value="curl" label="cURL">

```bash
curl -X POST "$LINKTOR_URL/api/v1/channels" \
  -H "X-API-Key: $LINKTOR_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
  "type": "teams",
  "name": "Suporte no Teams",
  "credentials": {
    "app_id": "00000000-0000-0000-0000-000000000000",
    "app_password": "valor_do_segredo_do_cliente",
    "tenant_id": "11111111-1111-1111-1111-111111111111",
    "webhook_secret": "um_segredo_longo_e_aleatorio"
  },
  "webhook_url": "https://seu-sistema.exemplo.com/webhooks/linktor"
}'
```

</TabItem>
<TabItem value="typescript" label="TypeScript">

```typescript
import { LinktorClient } from '@linktor/sdk';

// A base precisa incluir /api/v1: o SDK não acrescenta o prefixo.
const linktor = new LinktorClient({
  baseUrl: `${process.env.LINKTOR_URL}/api/v1`,
  apiKey: process.env.LINKTOR_API_KEY,
});

const channel = await linktor.channels.create({
  type: 'teams',
  name: 'Suporte no Teams',
  credentials: {
    app_id: '00000000-0000-0000-0000-000000000000',
    app_password: 'valor_do_segredo_do_cliente',
    tenant_id: '11111111-1111-1111-1111-111111111111',
    webhook_secret: 'um_segredo_longo_e_aleatorio',
  },
  webhook_url: 'https://seu-sistema.exemplo.com/webhooks/linktor',
});

console.log(channel.id, channel.connection_status);
```

</TabItem>
<TabItem value="python" label="Python">

```python
import os
import requests

resp = requests.post(
    f"{os.environ['LINKTOR_URL']}/api/v1/channels",
    headers={"X-API-Key": os.environ["LINKTOR_API_KEY"]},
    json={
        "type": "teams",
        "name": "Suporte no Teams",
        "credentials": {
            "app_id": "00000000-0000-0000-0000-000000000000",
            "app_password": "valor_do_segredo_do_cliente",
            "tenant_id": "11111111-1111-1111-1111-111111111111",
            "webhook_secret": "um_segredo_longo_e_aleatorio",
        },
        "webhook_url": "https://seu-sistema.exemplo.com/webhooks/linktor",
    },
    timeout=30,
)
resp.raise_for_status()
channel = resp.json()["data"]
print(channel["id"], channel["connection_status"])
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
	// A base precisa incluir /api/v1: o SDK não acrescenta o prefixo.
	client := linktor.NewClient(
		linktor.WithBaseURL(os.Getenv("LINKTOR_URL")+"/api/v1"),
		linktor.WithAPIKey(os.Getenv("LINKTOR_API_KEY")),
	)

	channel, err := client.Channels.Create(context.Background(), &types.CreateChannelInput{
		Type: types.ChannelTypeTeams,
		Name: "Suporte no Teams",
		Credentials: map[string]string{
			"app_id": "00000000-0000-0000-0000-000000000000",
			"app_password": "valor_do_segredo_do_cliente",
			"tenant_id": "11111111-1111-1111-1111-111111111111",
			"webhook_secret": "um_segredo_longo_e_aleatorio",
		},
		WebhookURL: "https://seu-sistema.exemplo.com/webhooks/linktor",
	})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(channel.ID, channel.ConnectionStatus)
}
```

</TabItem>
<TabItem value="java" label="Java">

```java
import io.linktor.LinktorClient;
import io.linktor.types.Channel;
import java.util.Map;

public class CriarCanal {
    public static void main(String[] args) throws Exception {
        // A base precisa incluir /api/v1: o SDK não acrescenta o prefixo.
        LinktorClient client = LinktorClient.builder()
            .baseUrl(System.getenv("LINKTOR_URL") + "/api/v1")
            .apiKey(System.getenv("LINKTOR_API_KEY"))
            .build();

        Channel.ChannelModel channel = client.channels.create(
            Channel.CreateChannelInput.builder()
                .type(Channel.ChannelType.TEAMS)
                .name("Suporte no Teams")
                .credentials(Map.of(
                    "app_id", "00000000-0000-0000-0000-000000000000",
                    "app_password", "valor_do_segredo_do_cliente",
                    "tenant_id", "11111111-1111-1111-1111-111111111111",
                    "webhook_secret", "um_segredo_longo_e_aleatorio"))
                .webhookUrl("https://seu-sistema.exemplo.com/webhooks/linktor")
                .build());

        System.out.println(channel.getId() + " " + channel.getConnectionStatus());
    }
}
```

</TabItem>
<TabItem value="csharp" label="C#">

```csharp
using System.Net.Http.Json;
using System.Text.Json;

var baseUrl = Environment.GetEnvironmentVariable("LINKTOR_URL");
using var http = new HttpClient();
http.DefaultRequestHeaders.Add("X-API-Key", Environment.GetEnvironmentVariable("LINKTOR_API_KEY"));

var resp = await http.PostAsJsonAsync($"{baseUrl}/api/v1/channels", new
{
    type = "teams",
    name = "Suporte no Teams",
    credentials = new Dictionary<string, string>
    {
        ["app_id"] = "00000000-0000-0000-0000-000000000000",
        ["app_password"] = "valor_do_segredo_do_cliente",
        ["tenant_id"] = "11111111-1111-1111-1111-111111111111",
        ["webhook_secret"] = "um_segredo_longo_e_aleatorio",
    },
    webhook_url = "https://seu-sistema.exemplo.com/webhooks/linktor",
});
resp.EnsureSuccessStatusCode();

using var doc = JsonDocument.Parse(await resp.Content.ReadAsStringAsync());
var channel = doc.RootElement.GetProperty("data");
Console.WriteLine($"{channel.GetProperty("id")} {channel.GetProperty("connection_status")}");
```

</TabItem>
<TabItem value="php" label="PHP">

```php
<?php
require 'vendor/autoload.php';

$http = new GuzzleHttp\Client();

$resp = $http->post(getenv('LINKTOR_URL') . '/api/v1/channels', [
    'headers' => ['X-API-Key' => getenv('LINKTOR_API_KEY')],
    'json' => [
        'type' => 'teams',
        'name' => 'Suporte no Teams',
        'credentials' => [
            'app_id' => '00000000-0000-0000-0000-000000000000',
            'app_password' => 'valor_do_segredo_do_cliente',
            'tenant_id' => '11111111-1111-1111-1111-111111111111',
            'webhook_secret' => 'um_segredo_longo_e_aleatorio',
        ],
        'webhook_url' => 'https://seu-sistema.exemplo.com/webhooks/linktor',
    ],
]);

$channel = json_decode((string) $resp->getBody(), true)['data'];
echo $channel['id'], ' ', $channel['connection_status'], PHP_EOL;
```

</TabItem>
<TabItem value="rust" label="Rust">

```rust
use std::collections::HashMap;

use linktor::{ChannelType, CreateChannelInput, LinktorClient};

#[tokio::main]
async fn main() -> Result<(), Box<dyn std::error::Error>> {
    // A base precisa incluir /api/v1: o SDK não acrescenta o prefixo.
    let client = LinktorClient::builder()
        .base_url(format!("{}/api/v1", std::env::var("LINKTOR_URL")?))
        .api_key(std::env::var("LINKTOR_API_KEY")?)
        .build()?;

    let channel = client
        .channels()
        .create(
            CreateChannelInput::new("Suporte no Teams", ChannelType::Teams)
                .credentials(HashMap::from([
                    ("app_id".to_string(), "00000000-0000-0000-0000-000000000000".to_string()),
                    ("app_password".to_string(), "valor_do_segredo_do_cliente".to_string()),
                    ("tenant_id".to_string(), "11111111-1111-1111-1111-111111111111".to_string()),
                    ("webhook_secret".to_string(), "um_segredo_longo_e_aleatorio".to_string()),
                ]))
                .webhook_url("https://seu-sistema.exemplo.com/webhooks/linktor"),
        )
        .await?;

    println!("{} {:?}", channel.id, channel.connection_status);
    Ok(())
}
```

</TabItem>
</Tabs>

Resposta `201`:

```json
{
  "success": true,
  "data": {
    "id": "CHANNEL_ID",
    "tenant_id": "…",
    "type": "teams",
    "name": "Suporte no Teams",
    "enabled": true,
    "connection_status": "disconnected",
    "environment": "production",
    "webhook_url": "https://seu-sistema.exemplo.com/webhooks/linktor",
    "created_at": "2026-11-24T14:00:00Z",
    "updated_at": "2026-11-24T14:00:00Z"
  }
}
```

O campo `tenant_id` da resposta é a empresa no Linktor; o tenant do Microsoft Entra ID é o
`credentials.tenant_id` que você enviou. O canal nasce habilitado e **desconectado**. Conecte-o
antes de enviar:

```bash
curl -X POST "$LINKTOR_URL/api/v1/channels/CHANNEL_ID/connect" \
  -H "X-API-Key: $LINKTOR_API_KEY"
```

Para o Teams, conectar apenas marca o canal como `connected`; use o teste de conexão para validar
as credenciais.

Para trocar um segredo, envie `PUT /api/v1/channels/CHANNEL_ID` só com a chave nova em
`credentials`; as demais credenciais são mantidas. `config`, ao contrário, é **substituído por
inteiro** quando enviado.

## Enviar e receber

### Enviar

`POST /api/v1/messages/send` com o scope `messages:send`. Em `to` vai o **id da conversa** do Bot
Framework — o `senderId` que chegou no evento `message.received`. Não é possível iniciar uma
conversa com alguém que nunca escreveu ao bot: o Linktor não cria conversas no Teams, só responde
nas que já existem.

<Tabs groupId="linguagem">
<TabItem value="curl" label="cURL">

```bash
curl -X POST "$LINKTOR_URL/api/v1/messages/send" \
  -H "X-API-Key: $LINKTOR_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
  "channel_id": "CHANNEL_ID",
  "to": "a:1AbCdEfGhIjKlMnOpQrStUvWxYz",
  "text": "Olá! Recebemos seu chamado e já estamos analisando."
}'
```

</TabItem>
<TabItem value="typescript" label="TypeScript">

```typescript
const resp = await fetch(`${process.env.LINKTOR_URL}/api/v1/messages/send`, {
  method: 'POST',
  headers: {
    'X-API-Key': process.env.LINKTOR_API_KEY!,
    'Content-Type': 'application/json',
  },
  body: JSON.stringify({
    channel_id: 'CHANNEL_ID',
    to: 'a:1AbCdEfGhIjKlMnOpQrStUvWxYz',
    text: 'Olá! Recebemos seu chamado e já estamos analisando.',
  }),
});
if (!resp.ok) throw new Error(`Linktor respondeu ${resp.status}: ${await resp.text()}`);

const { data } = await resp.json();
console.log(data.id, data.conversation_id, data.status); // status: "queued"
```

</TabItem>
<TabItem value="python" label="Python">

```python
import os
import requests

resp = requests.post(
    f"{os.environ['LINKTOR_URL']}/api/v1/messages/send",
    headers={"X-API-Key": os.environ["LINKTOR_API_KEY"]},
    json={
        "channel_id": "CHANNEL_ID",
        "to": "a:1AbCdEfGhIjKlMnOpQrStUvWxYz",
        "text": "Olá! Recebemos seu chamado e já estamos analisando.",
    },
    timeout=30,
)
resp.raise_for_status()
data = resp.json()["data"]
print(data["id"], data["conversation_id"], data["status"])  # status: "queued"
```

</TabItem>
<TabItem value="go" label="Go">

```go
package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
)

func main() {
	payload, _ := json.Marshal(map[string]string{
		"channel_id": "CHANNEL_ID",
		"to":         "a:1AbCdEfGhIjKlMnOpQrStUvWxYz",
		"text":       "Olá! Recebemos seu chamado e já estamos analisando.",
	})

	req, _ := http.NewRequest(http.MethodPost,
		os.Getenv("LINKTOR_URL")+"/api/v1/messages/send", bytes.NewReader(payload))
	req.Header.Set("X-API-Key", os.Getenv("LINKTOR_API_KEY"))
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusAccepted {
		log.Fatalf("Linktor respondeu %d", resp.StatusCode)
	}

	var out struct {
		Data struct {
			ID             string `json:"id"`
			ConversationID string `json:"conversation_id"`
			Status         string `json:"status"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		log.Fatal(err)
	}
	fmt.Println(out.Data.ID, out.Data.ConversationID, out.Data.Status) // status: "queued"
}
```

</TabItem>
<TabItem value="java" label="Java">

```java
import java.net.URI;
import java.net.http.HttpClient;
import java.net.http.HttpRequest;
import java.net.http.HttpResponse;

public class EnviarMensagem {
    public static void main(String[] args) throws Exception {
        String body = """
            {
              "channel_id": "CHANNEL_ID",
              "to": "a:1AbCdEfGhIjKlMnOpQrStUvWxYz",
              "text": "Olá! Recebemos seu chamado e já estamos analisando."
            }
            """;

        HttpRequest request = HttpRequest.newBuilder()
            .uri(URI.create(System.getenv("LINKTOR_URL") + "/api/v1/messages/send"))
            .header("X-API-Key", System.getenv("LINKTOR_API_KEY"))
            .header("Content-Type", "application/json")
            .POST(HttpRequest.BodyPublishers.ofString(body))
            .build();

        HttpResponse<String> response = HttpClient.newHttpClient()
            .send(request, HttpResponse.BodyHandlers.ofString());

        // 202 + {"success":true,"data":{"id":...,"conversation_id":...,"status":"queued"}}
        System.out.println(response.statusCode() + " " + response.body());
    }
}
```

</TabItem>
<TabItem value="csharp" label="C#">

```csharp
using System.Net.Http.Json;
using System.Text.Json;

var baseUrl = Environment.GetEnvironmentVariable("LINKTOR_URL");
using var http = new HttpClient();
http.DefaultRequestHeaders.Add("X-API-Key", Environment.GetEnvironmentVariable("LINKTOR_API_KEY"));

var resp = await http.PostAsJsonAsync($"{baseUrl}/api/v1/messages/send", new
{
    channel_id = "CHANNEL_ID",
    to = "a:1AbCdEfGhIjKlMnOpQrStUvWxYz",
    text = "Olá! Recebemos seu chamado e já estamos analisando.",
});
resp.EnsureSuccessStatusCode();

using var doc = JsonDocument.Parse(await resp.Content.ReadAsStringAsync());
var data = doc.RootElement.GetProperty("data");
Console.WriteLine($"{data.GetProperty("id")} {data.GetProperty("status")}"); // status: "queued"
```

</TabItem>
<TabItem value="php" label="PHP">

```php
<?php
require 'vendor/autoload.php';

$http = new GuzzleHttp\Client();

$resp = $http->post(getenv('LINKTOR_URL') . '/api/v1/messages/send', [
    'headers' => ['X-API-Key' => getenv('LINKTOR_API_KEY')],
    'json' => [
        'channel_id' => 'CHANNEL_ID',
        'to' => 'a:1AbCdEfGhIjKlMnOpQrStUvWxYz',
        'text' => 'Olá! Recebemos seu chamado e já estamos analisando.',
    ],
]);

$data = json_decode((string) $resp->getBody(), true)['data'];
echo $data['id'], ' ', $data['conversation_id'], ' ', $data['status'], PHP_EOL; // status: "queued"
```

</TabItem>
<TabItem value="rust" label="Rust">

```rust
use serde_json::{json, Value};

#[tokio::main]
async fn main() -> Result<(), Box<dyn std::error::Error>> {
    let url = format!("{}/api/v1/messages/send", std::env::var("LINKTOR_URL")?);

    let resp = reqwest::Client::new()
        .post(url)
        .header("X-API-Key", std::env::var("LINKTOR_API_KEY")?)
        .json(&json!({
            "channel_id": "CHANNEL_ID",
            "to": "a:1AbCdEfGhIjKlMnOpQrStUvWxYz",
            "text": "Olá! Recebemos seu chamado e já estamos analisando."
        }))
        .send()
        .await?
        .error_for_status()?;

    let body: Value = resp.json().await?;
    println!("{} {}", body["data"]["id"], body["data"]["status"]); // status: "queued"
    Ok(())
}
```

</TabItem>
</Tabs>

Resposta `202` — a mensagem foi gravada e enfileirada, ainda não entregue:

```json
{
  "success": true,
  "data": {
    "id": "0b6f1d0e-3a52-4b1e-9d0e-6c1f5f0f8a11",
    "conversation_id": "5d1c1a1e-8f0a-4a39-b7a0-3c2c9d7e6b10",
    "channel_id": "CHANNEL_ID",
    "status": "queued"
  }
}
```

O resultado da entrega chega depois, no webhook de saída, como `message.sent` ou
`message.failed`.

Para enviar um arquivo, informe `content_type` e `attachments`. O Teams recebe um anexo que
aponta para a URL, então ela precisa ser **pública** — o Teams busca o arquivo diretamente:

```bash
curl -X POST "$LINKTOR_URL/api/v1/messages/send" \
  -H "X-API-Key: $LINKTOR_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "channel_id": "CHANNEL_ID",
    "to": "a:1AbCdEfGhIjKlMnOpQrStUvWxYz",
    "content_type": "image",
    "text": "Comprovante em anexo",
    "attachments": [
      {
        "url": "https://arquivos.suaempresa.com/comprovante.png",
        "type": "image",
        "filename": "comprovante.png",
        "mime_type": "image/png"
      }
    ]
  }'
```

Para não enviar duas vezes em caso de repetição, informe `metadata.idempotency_key` (até 255
caracteres, único por empresa): a segunda chamada devolve a mensagem original.

### Receber

Com **URL do Consumidor** e **Segredo de Assinatura** preenchidos, cada mensagem recebida no Teams
é entregue ao seu sistema como `message.received`:

```json
{
  "id": "evt_8c7e2f0a-4b1d-4e0b-9a55-2f6d3c1b7e90_received",
  "type": "message.received",
  "timestamp": "2026-11-24T14:03:27.512Z",
  "tenantId": "3f2b7c9e-1d4a-4f6b-8e2a-9c0d1e2f3a4b",
  "environment": "production",
  "data": {
    "message": {
      "id": "8c7e2f0a-4b1d-4e0b-9a55-2f6d3c1b7e90",
      "direction": "inbound",
      "contentType": "text",
      "content": {
        "text": "Preciso de ajuda com o pedido 1234"
      },
      "senderId": "a:1AbCdEfGhIjKlMnOpQrStUvWxYz",
      "senderType": "contact",
      "metadata": {
        "senderName": "Maria Silva",
        "externalId": "1732456999123"
      }
    },
    "conversationId": "5d1c1a1e-8f0a-4a39-b7a0-3c2c9d7e6b10",
    "contactId": "a1b2c3d4-e5f6-4a7b-8c9d-0e1f2a3b4c5d",
    "channelId": "CHANNEL_ID",
    "channelType": "teams"
  }
}
```

| Campo | No Teams |
|---|---|
| `tenantId` | A empresa no Linktor — não o tenant do Microsoft Entra ID |
| `data.message.senderId` | Id da conversa do Bot Framework. É o valor a usar em `to` para responder |
| `data.message.metadata.senderName` | Nome de quem escreveu (`from.name` da Activity) |
| `data.message.metadata.externalId` | Id da Activity |
| `data.message.contentType` | `text`, ou `image` / `video` / `audio` / `document` quando há anexo com URL |
| `data.message.content.media` | Presente quando há anexo: `url`, `mimeType`, `filename`. Só o primeiro anexo é enviado no evento |

A entrega é assinada com `X-Linktor-Signature` e `X-Linktor-Timestamp`; como validar está em
[Receber eventos](../api/receber-eventos.md). **Sem segredo de assinatura o Linktor não entrega
nada**, mesmo com a URL preenchida.

## Limites e particularidades {#limites}

- **O bot só responde depois da primeira mensagem recebida.** O endereço do Bot Connector
  (`serviceUrl`) vem dentro de cada Activity. O Linktor grava esse valor em
  `credentials.service_url` na primeira mensagem e o usa nos envios. Antes disso, todo envio falha
  (depois das tentativas da fila) com `no service_url configured for channel`. É possível semear o valor ao criar o canal, mas ele
  varia por região e o caminho seguro é mandar uma mensagem ao bot.
- **`serviceUrl` só é aceito de hosts da Microsoft.** O Linktor grava o endereço apenas quando é
  HTTPS e termina em `botframework.com`, `trafficmanager.net` ou `skype.com`. Isso impede que uma
  requisição forjada desvie o token do bot para outro servidor.
- **Uma conversa do Teams é um contato.** A conversa no Linktor é identificada pelo id da conversa
  do Bot Framework. Em chat 1:1 isso equivale a uma pessoa; em chat de grupo ou canal de equipe,
  todos falam na mesma conversa, e o contato leva o nome de quem escreveu primeiro.
- **Em canais de equipe o Teams só entrega menções.** O bot recebe apenas mensagens em que foi
  mencionado, e o texto chega com a marcação da menção (`<at>Nome do Bot</at>`), que o Linktor
  não remove.
- **Mensagens com botões falham.** Adaptive Cards, botões e ações não são suportados. Uma mensagem
  de bot ou fluxo que leve respostas rápidas é recusada pelo adaptador com erro permanente
  (`unsupported content kind`) e vira `message.failed`. Em fluxos que atendem o Teams, use só
  texto.
- **Arquivos enviados vão por link.** O Linktor não faz upload ao Teams: envia um anexo com
  `contentUrl` e um tipo genérico (`image/*`, `video/*`, `audio/*` ou
  `application/octet-stream`). Imagens costumam aparecer na conversa; para os demais tipos a
  exibição depende do cliente do Teams. Se o arquivo precisa chegar com certeza, mande também o
  link no texto.
- **Arquivos recebidos.** As URLs de anexo do Bot Connector exigem o token do bot. Quando o
  servidor tem armazenamento de mídia configurado, o Linktor baixa o arquivo (até 64 MiB) e
  entrega uma URL própria; sem armazenamento, o evento leva a URL da Microsoft. Veja
  [Configuração](../implantar/configuracao.md).
- **Modelos viram texto.** Um envio de modelo entrega os parâmetros do corpo unidos por espaço
  (ou o nome do modelo, se não houver parâmetros).
- **Sem recibos.** O Linktor emite `message.sent` quando o Bot Connector aceita a mensagem e
  `message.failed` quando recusa. Erros `4xx` (exceto `429`) não são repetidos; `429` e `5xx` são
  tentados de novo pela fila.
- **Sandbox não se aplica.** A proteção de canal sandbox existe só para WhatsApp; um canal Teams
  criado com `environment: "sandbox"` tem todo envio bloqueado. Veja
  [Sandbox](../painel/sandbox.md).

## Solução de problemas

| Sintoma | Causa provável | O que fazer |
|---|---|---|
| **Testar Conexão** falha | Segredo expirado, **Secret ID** colado no lugar do **Value**, ou bot single-tenant sem **Tenant ID** | Gere um novo segredo e copie o **Value**; preencha o Tenant ID |
| O bot não responde e o Azure mostra erro no canal Teams | **Messaging endpoint** errado ou servidor sem acesso público | Cole a URL copiada do canal; teste de fora da rede |
| `401 invalid bot connector token` | `app_id` do canal diferente do bot que está chamando | Confira o **App ID** no Azure e no canal |
| `401 teams app_id not configured` | Canal sem `app_id` em servidor de produção | Edite o canal e preencha o **App ID** |
| `404 no teams channel for this bot/tenant` na URL compartilhada | Nenhum canal com esse `app_id` atende o tenant da organização | Crie um canal com o `tenant_id` da organização, ou um canal com `tenant_id` vazio |
| `message.failed` com `no service_url configured for channel` | O bot ainda não recebeu nenhuma mensagem nesse canal | Mande uma mensagem ao bot pelo Teams e reenvie |
| `message.failed` com `bot connector returned 401` ou `403` | Segredo trocado ou expirado, ou Tenant ID errado | Atualize **Senha do App** e teste a conexão |
| `message.failed` com `bot connector returned 404` | `to` não é um id de conversa válido para esse bot | Use o `senderId` recebido |
| Envio volta `channel is not connected` | Canal criado mas não conectado | Clique em **Conectar** no cartão do canal |
| O bot não vê as mensagens do canal de equipe | No Teams, bots só recebem mensagens com menção em canais | Mencione o bot ou use chat 1:1 |
| Nada chega ao seu sistema | **Segredo de Assinatura** vazio, ou evento fora de `config.webhook_events` | Preencha o segredo; revise o filtro |

Os erros de entrega e de webhook de cada canal aparecem em
[Observabilidade](../painel/observabilidade.md).
