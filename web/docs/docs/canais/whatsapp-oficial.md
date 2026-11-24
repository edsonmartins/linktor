---
sidebar_position: 2
title: WhatsApp oficial (Cloud API)
description: Conectar um número pela WhatsApp Business Cloud API da Meta — app, WABA, token permanente, webhook, Embedded Signup, coexistência, janela de 24h, modelos, botões, mídia e status com pricing.
---

# WhatsApp oficial (Cloud API)

## Resumo

O canal oficial fala com a **WhatsApp Business Cloud API** da Meta (Graph API). O número pertence a uma
conta WhatsApp Business (WABA), o envio é cobrado pela Meta e as regras da plataforma valem: janela de
24 horas e modelos de mensagem aprovados.

- Valor de `type` na API: **`whatsapp_official`**
- Nome no painel: **WhatsApp Business** ("API do Meta Business Cloud")
- Versão padrão da Graph API usada pelo Linktor: `v23.0` (campo **Versão da API**)

| Recurso | Suportado | Observação |
|---|---|---|
| Texto | Sim | Até 4096 caracteres; `preview_url` opcional |
| Imagem, vídeo, áudio, documento | Sim | Uma mídia por mensagem; legenda em imagem, vídeo e documento |
| Figurinha, localização, contato | Recebe | Figurinha chega como imagem; localização e contato chegam como conteúdo JSON |
| Botões de resposta | Sim | Até 3 botões; acima disso vira lista |
| Lista | Sim | Gerada automaticamente a partir de 4 opções ou mais |
| Modelos de mensagem (templates) | Sim | Obrigatórios fora da janela de 24h |
| Reações | Recebe | Chegam como evento `message.received` com o bloco `reaction` |
| Grupos | Não | A Cloud API tratada aqui é 1:1 |
| Entregue / lida | Sim | `message.delivered` e `message.read`; o status traz `pricing` |
| Coexistência com o WhatsApp Business App | Sim | Via Embedded Signup; ver [Coexistência](#coexistencia) |

Se você não tem (ou não quer) uma conta na Meta e aceita os riscos de uma sessão de WhatsApp Web, veja
[WhatsApp não oficial](./whatsapp-nao-oficial.md).

## Antes de começar

Você precisa de:

1. Uma conta em [developers.facebook.com](https://developers.facebook.com) com acesso de administrador
   a um **portfólio empresarial** (Business Manager).
2. Uma **conta do WhatsApp Business (WABA)** dentro desse portfólio.
3. Um **número de telefone** registrado na WABA. Um número que hoje está no aplicativo WhatsApp comum
   precisa ser migrado; um número que está no **WhatsApp Business App** pode continuar no celular se
   você usar a [coexistência](#coexistencia).
4. Verificação do negócio na Meta, para sair dos limites de conta não verificada.
5. O servidor Linktor publicado em **HTTPS** com certificado válido — a Meta não chama webhook em HTTP.
6. Um usuário do Linktor com papel de administrador, ou uma [chave de API](../painel/chaves-de-api.md)
   com o scope `channels:write`.

:::note
Os nomes de menu do painel da Meta mudam com frequência. Os passos abaixo descrevem o que precisa ser
feito; se um rótulo estiver diferente, procure pela função equivalente.
:::

## Configurar na origem

### 1. Criar o app

1. Em **Meus apps** → **Criar app**, escolha o caso de uso de negócios (tipo **Empresa**) e vincule o
   app ao seu portfólio empresarial.
2. No painel do app, adicione o produto **WhatsApp**.
3. Em **WhatsApp** → **Configuração da API**, anote:
   - **Identificação do número de telefone** (Phone number ID) — vai no campo **ID do Número de Telefone**;
   - **Identificação da conta do WhatsApp Business** (WABA ID) — vai no campo **ID da Conta WhatsApp Business**.
4. Em **Configurações do app** → **Básico**, copie a **Chave secreta do app** (App Secret) — vai no
   campo **Segredo do App**.

### 2. Gerar um token permanente (System User)

O token temporário da tela **Configuração da API** expira em cerca de 24 horas. Para produção, use um
token de **usuário do sistema**:

1. Em **Configurações do negócio** (business.facebook.com) → **Usuários** → **Usuários do sistema**,
   crie um usuário do sistema com função de **Administrador**.
2. Em **Atribuir ativos**, dê a ele controle total sobre o **app** e sobre a **conta do WhatsApp**.
3. Clique em **Gerar novo token**, escolha o app, validade **Nunca** e marque as permissões:
   - `whatsapp_business_messaging` — enviar e receber mensagens;
   - `whatsapp_business_management` — modelos, números e assinatura de webhook;
   - `business_management` — só necessária no Embedded Signup.
4. Copie o token (começa com `EAA`). Ele só é exibido uma vez.

### 3. Assinar o app na WABA

A Meta só entrega eventos de uma WABA para apps assinados nela. Com o token do passo anterior:

```bash
curl -X POST "https://graph.facebook.com/v23.0/WABA_ID/subscribed_apps" \
  -H "Authorization: Bearer EAAxxxxxxxx"
```

No fluxo [Embedded Signup](#embedded-signup) o Linktor faz essa chamada por você.

### 4. Configurar o webhook no app

Faça este passo **depois** de criar o canal no Linktor, porque a URL contém o id do canal. Veja
[Webhook](#webhook).

## Configurar no Linktor

1. No painel, abra **Canais** e, em **Adicionar Novo Canal**, clique em **WhatsApp Business**.
2. A tela tem as abas **Configuração Rápida** (Embedded Signup), **Manual**, **Webhook** e **Guia**.
   Depois de salvo, o canal ganha também a aba **Operações**.
3. Na aba **Manual**, preencha os campos da tabela abaixo e clique em **Testar Conexão**.
4. Na aba **Webhook**, copie o **Token de Verificação** (já vem gerado) e preencha o **Segredo do App**.
5. Clique em **Criar Canal**. A **URL do Webhook** aparece na aba **Webhook** depois de salvar.
6. Na lista de canais, abra o menu do canal e clique em **Conectar**. Só um canal conectado e habilitado
   aceita envio.

| Rótulo na tela | Chave | Onde fica | Obrigatório |
|---|---|---|---|
| **Nome do Canal** | `name` | raiz | Sim |
| **Ambiente do canal** | `environment` (`production` ou `sandbox`) | raiz | Não (padrão `production`; imutável depois) |
| **Token de Acesso** | `access_token` | `credentials` | Sim |
| **ID do Número de Telefone** | `phone_number_id` | `config` | Sim |
| **ID da Conta WhatsApp Business** | `business_id` | `config` | Não para enviar; sim para modelos e analytics |
| **Versão da API** | `api_version` | `config` | Não (padrão `v23.0`) |
| **Token de Verificação** | `verify_token` | `config` | Sim |
| **Segredo do App** | `webhook_secret` | `credentials` | Sim em produção (ver abaixo) |

Outras chaves aceitas, sem campo na tela (defina pela API):

| Chave | Onde fica | Efeito |
|---|---|---|
| `webhook_url` | raiz | Endereço da **sua** aplicação que recebe os eventos do canal |
| `webhook_events` | `config` | Lista separada por vírgula dos eventos a entregar; vazio entrega todos |
| `waba_id` | `config` | Alternativa a `business_id` (é a chave que o Embedded Signup grava) |

:::warning O Segredo do App tem dois usos neste canal
A chave `credentials.webhook_secret` é usada para **validar a assinatura dos webhooks da Meta** (por
isso ela precisa ser o App Secret) e também para **assinar os eventos que o Linktor envia à sua
aplicação** em `webhook_url`. Na prática, sua aplicação verifica a assinatura dos eventos do Linktor
usando o App Secret da Meta. Trate esse valor como segredo nos dois lados.
:::

O botão **Testar Conexão** deste canal só confere se `access_token` e `phone_number_id` foram
informados — ele não chama a Meta. A validação real acontece no primeiro envio e no handshake do
webhook.

Para um canal de teste, escolha **Sandbox — canal de teste/homologação** em **Ambiente do canal**; os
campos extras e as garantias estão em [Sandbox](../painel/sandbox.md).

### Embedded Signup {#embedded-signup}

A aba **Configuração Rápida** conecta o número por login do Facebook, sem copiar ids nem tokens:

1. Informe o **Facebook App ID** e clique em **Conectar com Facebook**. Um popup abre o login da Meta
   pedindo as permissões `whatsapp_business_management`, `whatsapp_business_messaging` e
   `business_management`.
2. Ao concluir, o Linktor troca o código por um token, localiza a WABA e o número, verifica se o número
   está em coexistência e assina o app na WABA nos campos `messages`,
   `message_template_status_update`, `message_template_quality_update`, `template_category_update`,
   `account_alerts`, `account_update`, `phone_number_quality_update` e `security` (mais
   `message_echoes` quando há coexistência).
3. A tela mostra **Número de Telefone**, **Nome do Negócio**, **Coexistência** e **Classificação de
   Qualidade**. Dê um **Nome do Canal** e clique em **Criar Canal**. O canal já nasce conectado.

Pré-requisitos no servidor:

| Variável de ambiente | Uso |
|---|---|
| `WHATSAPP_EMBEDDED_SIGNUP_APP_SECRET` | App Secret usado para trocar o código por token |
| `WHATSAPP_EMBEDDED_SIGNUP_APP_SECRET_<APP_ID>` | Mesmo valor, por app (tem prioridade); o id entra em maiúsculas, com `_` no lugar de outros caracteres |
| `BASE_URL` | Endereço público do servidor; compõe a URL de retorno `<BASE_URL>/api/v1/oauth/whatsapp/embedded-signup/callback`, que precisa estar cadastrada no app em **Login do Facebook** → **URIs de redirecionamento OAuth válidos** |

Sem o App Secret no servidor a conclusão falha com
`server-side app secret is not configured for WhatsApp Embedded Signup`.

:::warning Depois do Embedded Signup, complete o canal
O canal criado por esse fluxo **não grava `credentials.webhook_secret`**. Em produção o Linktor recusa
webhooks de canais sem esse segredo (HTTP 401, `webhook secret not configured`), então nenhuma mensagem
entra. Abra o canal, vá à aba **Webhook**, preencha **Segredo do App** e salve; em seguida cadastre a
URL e o token de verificação no app da Meta, como descrito em [Webhook](#webhook).
:::

As rotas usadas pela tela são `POST /oauth/whatsapp/embedded-signup/start`, `.../callback` e
`.../create-channel`. Elas respondem fora do envelope padrão (`{"error": "..."}` em caso de falha) e
foram feitas para o painel; para automação, prefira `POST /channels`.

### Coexistência {#coexistencia}

Coexistência é usar o **mesmo número** no WhatsApp Business App (no celular) e na Cloud API. Ela só é
ativada pelo Embedded Signup, quando a Meta informa que o número está ativo no aplicativo.

- O canal guarda `is_coexistence`, `waba_id`, `coexistence_status` e `last_echo_at`.
- `GET /channels/CHANNEL_ID/coexistence-status` devolve `enabled`, `status`, `last_echo_at`,
  `days_since_last_echo`, `days_until_disconnect` e `recommendation`.
- `POST /channels/CHANNEL_ID/subscribe-echoes` assina `message_echoes` num canal oficial já existente e
  o marca como coexistência.

| `status` | Significado |
|---|---|
| `inactive` | O canal não é de coexistência |
| `pending` | Assinado, mas nenhum eco chegou ainda — abra o WhatsApp Business App |
| `active` | Último eco há menos de 10 dias |
| `warning` | Último eco entre 10 e 13 dias |
| `disconnected` | 14 dias ou mais sem eco |

A Meta exige que o aplicativo seja aberto no celular pelo menos uma vez a cada 14 dias; depois disso
ela desfaz o vínculo.

:::note Parcial
Hoje o eco (`message_echoes`) serve para manter o estado da coexistência: ele atualiza `last_echo_at`
e o `status`. O **conteúdo** das mensagens digitadas no WhatsApp Business App não é gravado na
conversa pela rota de webhook, e a importação do histórico anterior do aplicativo não está
implementada.
:::

## Webhook {#webhook}

URL que a Meta deve chamar:

```text
https://linktor.suaempresa.com/api/v1/webhooks/whatsapp/CHANNEL_ID
```

No app da Meta, em **WhatsApp** → **Configuração** → **Webhook**:

1. **URL de retorno** — a URL acima (copie da aba **Webhook** do canal).
2. **Token de verificação** — o valor do campo **Token de Verificação**.
3. Clique em **Verificar e salvar**. A Meta faz um `GET` com `hub.mode=subscribe`,
   `hub.verify_token` e `hub.challenge`; o Linktor compara o token em tempo constante e devolve o
   desafio. Token diferente, ou canal sem token, responde 403 `verification failed`.
4. Em **Campos do webhook**, assine:

| Campo | Para quê | Necessário |
|---|---|---|
| `messages` | Mensagens recebidas e status (enviada, entregue, lida, falhou) | Sim |
| `message_template_status_update` | Aprovação, rejeição e pausa de modelos | Recomendado |
| `message_template_quality_update` | Mudança de qualidade de modelo | Recomendado |
| `template_category_update` | Reclassificação de categoria de modelo | Recomendado |
| `phone_number_quality_update` | Qualidade do número e limite de envio | Recomendado |
| `message_echoes` | Coexistência | Só com coexistência |

Validação da assinatura:

- A Meta envia `X-Hub-Signature-256: sha256=<hex>`, o HMAC-SHA256 do corpo bruto com o App Secret.
- O Linktor recalcula com `credentials.webhook_secret` e compara. Assinatura ausente ou diferente
  responde 401 `invalid signature`.
- Canal **sem** `webhook_secret`: o webhook é recusado com 401 quando o servidor roda em modo
  `release` ou com `LINKTOR_REQUIRE_WEBHOOK_SECRETS=true`. Fora disso (desenvolvimento) é aceito sem
  validação.

Os webhooks passam por deduplicação (janela de 6 horas), então reentregas da Meta não duplicam
mensagens. Eventos de qualidade do número atualizam `config.quality_rating`,
`config.quality_rating_event` e `config.messaging_limit_tier` do canal.

## Criar o canal pela API

`POST /api/v1/channels` com o scope `channels:write`. A resposta é `201` com o canal em `data`
(`credentials` nunca volta; segredos em `config` voltam como `__redacted__`). O canal nasce
`disconnected`: chame `POST /channels/CHANNEL_ID/connect` em seguida.

<Tabs groupId="linguagem">
<TabItem value="curl" label="cURL">

```bash
curl -X POST "$LINKTOR_URL/api/v1/channels" \
  -H "X-API-Key: $LINKTOR_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "type": "whatsapp_official",
    "name": "WhatsApp Vendas",
    "config": {
      "phone_number_id": "123456789012345",
      "business_id": "987654321098765",
      "verify_token": "um-token-aleatorio-longo",
      "api_version": "v23.0",
      "webhook_events": "message.received,message.delivered,message.read,message.failed"
    },
    "credentials": {
      "access_token": "EAAxxxxxxxx",
      "webhook_secret": "APP_SECRET_DA_META"
    },
    "webhook_url": "https://app.suaempresa.com/webhooks/linktor"
  }'

# Marca o canal como conectado
curl -X POST "$LINKTOR_URL/api/v1/channels/CHANNEL_ID/connect" \
  -H "X-API-Key: $LINKTOR_API_KEY"
```

</TabItem>
<TabItem value="typescript" label="TypeScript">

```typescript
import { LinktorClient } from '@linktor/sdk';

const client = new LinktorClient({
  baseUrl: `${process.env.LINKTOR_URL}/api/v1`,
  apiKey: process.env.LINKTOR_API_KEY,
});

const canal = await client.channels.create({
  type: 'whatsapp_official',
  name: 'WhatsApp Vendas',
  config: {
    phone_number_id: '123456789012345',
    business_id: '987654321098765',
    verify_token: 'um-token-aleatorio-longo',
    api_version: 'v23.0',
  },
  credentials: {
    access_token: 'EAAxxxxxxxx',
    webhook_secret: 'APP_SECRET_DA_META',
  },
  webhook_url: 'https://app.suaempresa.com/webhooks/linktor',
});

await client.channels.connect(canal.id);
console.log(`${process.env.LINKTOR_URL}/api/v1/webhooks/whatsapp/${canal.id}`);
```

</TabItem>
<TabItem value="python" label="Python">

```python
import os
import requests

base = os.environ["LINKTOR_URL"].rstrip("/") + "/api/v1"
headers = {"X-API-Key": os.environ["LINKTOR_API_KEY"]}

# O enum de tipos do SDK Python não conhece whatsapp_official: use HTTP direto.
r = requests.post(f"{base}/channels", headers=headers, timeout=30, json={
    "type": "whatsapp_official",
    "name": "WhatsApp Vendas",
    "config": {
        "phone_number_id": "123456789012345",
        "business_id": "987654321098765",
        "verify_token": "um-token-aleatorio-longo",
        "api_version": "v23.0",
    },
    "credentials": {
        "access_token": "EAAxxxxxxxx",
        "webhook_secret": "APP_SECRET_DA_META",
    },
    "webhook_url": "https://app.suaempresa.com/webhooks/linktor",
})
r.raise_for_status()
canal = r.json()["data"]

requests.post(f"{base}/channels/{canal['id']}/connect", headers=headers, timeout=30).raise_for_status()
print(f"{base}/webhooks/whatsapp/{canal['id']}")
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

	canal, err := client.Channels.Create(ctx, &types.CreateChannelInput{
		Type: types.ChannelTypeWhatsAppOfficial,
		Name: "WhatsApp Vendas",
		Config: map[string]string{
			"phone_number_id": "123456789012345",
			"business_id":     "987654321098765",
			"verify_token":    "um-token-aleatorio-longo",
			"api_version":     "v23.0",
		},
		Credentials: map[string]string{
			"access_token":   "EAAxxxxxxxx",
			"webhook_secret": "APP_SECRET_DA_META",
		},
		WebhookURL: "https://app.suaempresa.com/webhooks/linktor",
	})
	if err != nil {
		log.Fatal(err)
	}
	if _, err := client.Channels.Connect(ctx, canal.ID); err != nil {
		log.Fatal(err)
	}
	fmt.Printf("%s/api/v1/webhooks/whatsapp/%s\n", os.Getenv("LINKTOR_URL"), canal.ID)
}
```

</TabItem>
<TabItem value="java" label="Java">

```java
import io.linktor.LinktorClient;
import io.linktor.types.Channel;
import java.util.Map;

LinktorClient client = LinktorClient.builder()
        .baseUrl(System.getenv("LINKTOR_URL") + "/api/v1")
        .apiKey(System.getenv("LINKTOR_API_KEY"))
        .build();

Channel.ChannelModel canal = client.channels.create(
        Channel.CreateChannelInput.builder()
                .type(Channel.ChannelType.WHATSAPP_OFFICIAL)
                .name("WhatsApp Vendas")
                .config(Map.of(
                        "phone_number_id", "123456789012345",
                        "business_id", "987654321098765",
                        "verify_token", "um-token-aleatorio-longo",
                        "api_version", "v23.0"))
                .credentials(Map.of(
                        "access_token", "EAAxxxxxxxx",
                        "webhook_secret", "APP_SECRET_DA_META"))
                .webhookUrl("https://app.suaempresa.com/webhooks/linktor")
                .build());

client.channels.connect(canal.getId());
System.out.println(System.getenv("LINKTOR_URL") + "/api/v1/webhooks/whatsapp/" + canal.getId());
```

</TabItem>
<TabItem value="csharp" label="C#">

```csharp
using System.Net.Http.Json;
using System.Text.Json;

var http = new HttpClient
{
    BaseAddress = new Uri(Environment.GetEnvironmentVariable("LINKTOR_URL")!.TrimEnd('/') + "/api/v1/"),
};
http.DefaultRequestHeaders.Add("X-API-Key", Environment.GetEnvironmentVariable("LINKTOR_API_KEY"));

var resposta = await http.PostAsJsonAsync("channels", new
{
    type = "whatsapp_official",
    name = "WhatsApp Vendas",
    config = new Dictionary<string, string>
    {
        ["phone_number_id"] = "123456789012345",
        ["business_id"] = "987654321098765",
        ["verify_token"] = "um-token-aleatorio-longo",
        ["api_version"] = "v23.0",
    },
    credentials = new Dictionary<string, string>
    {
        ["access_token"] = "EAAxxxxxxxx",
        ["webhook_secret"] = "APP_SECRET_DA_META",
    },
    webhook_url = "https://app.suaempresa.com/webhooks/linktor",
});
resposta.EnsureSuccessStatusCode();

var corpo = await resposta.Content.ReadFromJsonAsync<JsonElement>();
var id = corpo.GetProperty("data").GetProperty("id").GetString();

(await http.PostAsync($"channels/{id}/connect", null)).EnsureSuccessStatusCode();
Console.WriteLine($"{http.BaseAddress}webhooks/whatsapp/{id}");
```

</TabItem>
<TabItem value="php" label="PHP">

```php
<?php
require __DIR__ . '/vendor/autoload.php';

$http = new GuzzleHttp\Client([
    'base_uri' => rtrim(getenv('LINKTOR_URL'), '/') . '/api/v1/',
    'headers'  => ['X-API-Key' => getenv('LINKTOR_API_KEY')],
    'timeout'  => 30,
]);

$resposta = $http->post('channels', ['json' => [
    'type' => 'whatsapp_official',
    'name' => 'WhatsApp Vendas',
    'config' => [
        'phone_number_id' => '123456789012345',
        'business_id'     => '987654321098765',
        'verify_token'    => 'um-token-aleatorio-longo',
        'api_version'     => 'v23.0',
    ],
    'credentials' => [
        'access_token'   => 'EAAxxxxxxxx',
        'webhook_secret' => 'APP_SECRET_DA_META',
    ],
    'webhook_url' => 'https://app.suaempresa.com/webhooks/linktor',
]]);

$canal = json_decode((string) $resposta->getBody(), true)['data'];
$http->post("channels/{$canal['id']}/connect");

echo rtrim(getenv('LINKTOR_URL'), '/') . "/api/v1/webhooks/whatsapp/{$canal['id']}", PHP_EOL;
```

</TabItem>
<TabItem value="rust" label="Rust">

```rust
use linktor::{ChannelType, CreateChannelInput, LinktorClient};
use std::collections::HashMap;

#[tokio::main]
async fn main() -> Result<(), Box<dyn std::error::Error>> {
    let servidor = std::env::var("LINKTOR_URL")?;
    let client = LinktorClient::builder()
        .base_url(format!("{}/api/v1", servidor.trim_end_matches('/')))
        .api_key(std::env::var("LINKTOR_API_KEY")?)
        .build()?;

    let config = HashMap::from([
        ("phone_number_id".to_string(), "123456789012345".to_string()),
        ("business_id".to_string(), "987654321098765".to_string()),
        ("verify_token".to_string(), "um-token-aleatorio-longo".to_string()),
        ("api_version".to_string(), "v23.0".to_string()),
    ]);
    let credentials = HashMap::from([
        ("access_token".to_string(), "EAAxxxxxxxx".to_string()),
        ("webhook_secret".to_string(), "APP_SECRET_DA_META".to_string()),
    ]);

    let canal = client
        .channels()
        .create(
            CreateChannelInput::new("WhatsApp Vendas", ChannelType::WhatsappOfficial)
                .config(config)
                .credentials(credentials)
                .webhook_url("https://app.suaempresa.com/webhooks/linktor"),
        )
        .await?;

    client.channels().connect(&canal.id).await?;
    println!("{}/api/v1/webhooks/whatsapp/{}", servidor, canal.id);
    Ok(())
}
```

</TabItem>
</Tabs>

:::warning Ao atualizar, reenvie o `config` inteiro
`PUT /channels/CHANNEL_ID` **substitui** `config` por completo; só `credentials` é mesclado (valor
vazio ou `__redacted__` mantém o segredo guardado). Um update que mande apenas `{"api_version": "v24.0"}`
em `config` apaga `phone_number_id` e `verify_token`.
:::

Referência completa das rotas de canal em [API de canais](../api/canais.md).

## Enviar e receber

### Enviar uma mensagem

`POST /api/v1/messages/send` (scope `messages:send`) envia para um telefone sem que você precise
conhecer a conversa: o Linktor resolve ou cria o contato e a conversa. A resposta `202` significa
**enfileirada**; o resultado chega depois como `message.sent`, `message.delivered`, `message.read` ou
`message.failed`.

<Tabs groupId="linguagem">
<TabItem value="curl" label="cURL">

```bash
curl -X POST "$LINKTOR_URL/api/v1/messages/send" \
  -H "X-API-Key: $LINKTOR_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "channel_id": "CHANNEL_ID",
    "to": "+5511999999999",
    "content_type": "text",
    "text": "Olá! Seu pedido saiu para entrega.",
    "metadata": { "idempotency_key": "pedido-1042-saiu" }
  }'
```

</TabItem>
<TabItem value="typescript" label="TypeScript">

```typescript
import { LinktorClient } from '@linktor/sdk';

const client = new LinktorClient({
  baseUrl: `${process.env.LINKTOR_URL}/api/v1`,
  apiKey: process.env.LINKTOR_API_KEY,
});

const res: any = await client.messages.sendText(
  'CHANNEL_ID',
  '+5511999999999',
  'Olá! Seu pedido saiu para entrega.',
  { idempotency_key: 'pedido-1042-saiu' },
);

// O SDK devolve o envelope inteiro: o resultado está em res.data
console.log(res.data.id, res.data.status); // ... queued
```

</TabItem>
<TabItem value="python" label="Python">

```python
import os
import requests

base = os.environ["LINKTOR_URL"].rstrip("/") + "/api/v1"

r = requests.post(
    f"{base}/messages/send",
    headers={"X-API-Key": os.environ["LINKTOR_API_KEY"]},
    timeout=30,
    json={
        "channel_id": "CHANNEL_ID",
        "to": "+5511999999999",
        "content_type": "text",
        "text": "Olá! Seu pedido saiu para entrega.",
        "metadata": {"idempotency_key": "pedido-1042-saiu"},
    },
)
r.raise_for_status()
envio = r.json()["data"]
print(envio["id"], envio["status"])  # ... queued
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
	corpo, _ := json.Marshal(map[string]any{
		"channel_id":   "CHANNEL_ID",
		"to":           "+5511999999999",
		"content_type": "text",
		"text":         "Olá! Seu pedido saiu para entrega.",
		"metadata":     map[string]string{"idempotency_key": "pedido-1042-saiu"},
	})

	req, _ := http.NewRequest(http.MethodPost,
		os.Getenv("LINKTOR_URL")+"/api/v1/messages/send", bytes.NewReader(corpo))
	req.Header.Set("X-API-Key", os.Getenv("LINKTOR_API_KEY"))
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Fatal(err)
	}
	defer resp.Body.Close()

	var saida struct {
		Data struct {
			ID     string `json:"id"`
			Status string `json:"status"`
		} `json:"data"`
	}
	if resp.StatusCode != http.StatusAccepted {
		log.Fatalf("envio recusado: %s", resp.Status)
	}
	json.NewDecoder(resp.Body).Decode(&saida)
	fmt.Println(saida.Data.ID, saida.Data.Status) // ... queued
}
```

</TabItem>
<TabItem value="java" label="Java">

```java
import com.google.gson.Gson;
import com.google.gson.JsonObject;
import java.net.URI;
import java.net.http.HttpClient;
import java.net.http.HttpRequest;
import java.net.http.HttpResponse;
import java.util.Map;

Gson gson = new Gson();
String corpo = gson.toJson(Map.of(
        "channel_id", "CHANNEL_ID",
        "to", "+5511999999999",
        "content_type", "text",
        "text", "Olá! Seu pedido saiu para entrega.",
        "metadata", Map.of("idempotency_key", "pedido-1042-saiu")));

HttpRequest request = HttpRequest.newBuilder(
                URI.create(System.getenv("LINKTOR_URL") + "/api/v1/messages/send"))
        .header("Content-Type", "application/json")
        .header("X-API-Key", System.getenv("LINKTOR_API_KEY"))
        .POST(HttpRequest.BodyPublishers.ofString(corpo))
        .build();

HttpResponse<String> response = HttpClient.newHttpClient()
        .send(request, HttpResponse.BodyHandlers.ofString());
if (response.statusCode() != 202) {
    throw new IllegalStateException("Envio recusado: " + response.body());
}
JsonObject data = gson.fromJson(response.body(), JsonObject.class).getAsJsonObject("data");
System.out.println(data.get("id").getAsString() + " " + data.get("status").getAsString());
```

</TabItem>
<TabItem value="csharp" label="C#">

```csharp
using System.Net.Http.Json;
using System.Text.Json;

var http = new HttpClient
{
    BaseAddress = new Uri(Environment.GetEnvironmentVariable("LINKTOR_URL")!.TrimEnd('/') + "/api/v1/"),
};
http.DefaultRequestHeaders.Add("X-API-Key", Environment.GetEnvironmentVariable("LINKTOR_API_KEY"));

var resposta = await http.PostAsJsonAsync("messages/send", new
{
    channel_id = "CHANNEL_ID",
    to = "+5511999999999",
    content_type = "text",
    text = "Olá! Seu pedido saiu para entrega.",
    metadata = new Dictionary<string, string> { ["idempotency_key"] = "pedido-1042-saiu" },
});
resposta.EnsureSuccessStatusCode();

var data = (await resposta.Content.ReadFromJsonAsync<JsonElement>()).GetProperty("data");
Console.WriteLine($"{data.GetProperty("id")} {data.GetProperty("status")}"); // ... queued
```

</TabItem>
<TabItem value="php" label="PHP">

```php
<?php
require __DIR__ . '/vendor/autoload.php';

$http = new GuzzleHttp\Client([
    'base_uri' => rtrim(getenv('LINKTOR_URL'), '/') . '/api/v1/',
    'headers'  => ['X-API-Key' => getenv('LINKTOR_API_KEY')],
    'timeout'  => 30,
]);

$resposta = $http->post('messages/send', ['json' => [
    'channel_id'   => 'CHANNEL_ID',
    'to'           => '+5511999999999',
    'content_type' => 'text',
    'text'         => 'Olá! Seu pedido saiu para entrega.',
    'metadata'     => ['idempotency_key' => 'pedido-1042-saiu'],
]]);

$envio = json_decode((string) $resposta->getBody(), true)['data'];
echo $envio['id'], ' ', $envio['status'], PHP_EOL; // ... queued
```

</TabItem>
<TabItem value="rust" label="Rust">

```rust
use serde_json::{json, Value};

#[tokio::main]
async fn main() -> Result<(), Box<dyn std::error::Error>> {
    let base = format!("{}/api/v1", std::env::var("LINKTOR_URL")?.trim_end_matches('/'));

    let resposta = reqwest::Client::new()
        .post(format!("{base}/messages/send"))
        .header("X-API-Key", std::env::var("LINKTOR_API_KEY")?)
        .json(&json!({
            "channel_id": "CHANNEL_ID",
            "to": "+5511999999999",
            "content_type": "text",
            "text": "Olá! Seu pedido saiu para entrega.",
            "metadata": { "idempotency_key": "pedido-1042-saiu" }
        }))
        .send()
        .await?
        .error_for_status()?;

    let corpo: Value = resposta.json().await?;
    println!("{} {}", corpo["data"]["id"], corpo["data"]["status"]); // ... "queued"
    Ok(())
}
```

</TabItem>
</Tabs>

Resposta:

```json
{
  "success": true,
  "data": {
    "id": "7c1f0a52-0d1e-4c3b-9b7e-2a9c1f4e8d10",
    "conversation_id": "b3a9d2c4-55e1-4f0a-8a44-0f6f1f1f7a21",
    "channel_id": "CHANNEL_ID",
    "status": "queued"
  }
}
```

O envio direto aceita `content_type` `text`, `image`, `video`, `audio`, `document` e `sticker`; os
tipos de mídia exigem `attachments` com `url`. Corpo completo, idempotência e correlação em
[Enviar mensagens](../api/enviar-mensagens.md).

### Mídia

Envie a mídia por URL pública (ou enviada antes pelo Linktor — veja [Mídia](../api/midia.md)):

```json
{
  "channel_id": "CHANNEL_ID",
  "to": "+5511999999999",
  "content_type": "document",
  "text": "Segue a nota fiscal",
  "attachments": [
    {
      "url": "https://app.suaempresa.com/arquivos/nf-1042.pdf",
      "type": "document",
      "filename": "nf-1042.pdf",
      "mime_type": "application/pdf"
    }
  ]
}
```

O texto vira a legenda. Na entrada, a Meta entrega só um identificador de mídia: o Linktor baixa o
arquivo com o token do canal e o regrava no armazenamento de mídia, de modo que o evento já traz uma
URL utilizável em `content.media.url`.

### Modelos de mensagem (templates) {#modelos}

Fora da janela de 24 horas só passa modelo aprovado. Os modelos são criados, sincronizados e
acompanhados em [Modelos de mensagem](../painel/modelos-de-mensagem.md) (API:
[Modelos e campanhas](../api/modelos-e-campanhas.md)), e exigem o **ID da Conta WhatsApp Business** no
canal.

Há dois caminhos para disparar um modelo:

- **Campanha** — o caminho indicado para iniciar conversas, com lista de destinatários, modelo e
  variáveis.
- **Mensagem numa conversa existente** — `POST /conversations/CONVERSATION_ID/messages` com
  `content_type: "template"` e os dados do modelo em `metadata`:

```json
{
  "content_type": "template",
  "content": "pedido_enviado",
  "metadata": {
    "template_name": "pedido_enviado",
    "template_language": "pt_BR",
    "template_components": "[{\"type\":\"body\",\"parameters\":[{\"type\":\"text\",\"text\":\"Maria\"},{\"type\":\"text\",\"text\":\"1042\"}]}]"
  }
}
```

`template_components` é uma **string** com o JSON de `components` no formato da Cloud API. Sem
`template_language` o idioma enviado é `en`. `POST /messages/send` não aceita `template`.

### Botões e lista {#botoes}

Bots e fluxos enviam opções como mensagem interativa; pela API, use a rota de conversa com
`content_type: "interactive"`:

```json
{
  "content_type": "interactive",
  "content": "Como podemos ajudar?",
  "metadata": {
    "interactive_body": "Como podemos ajudar?",
    "quick_replies": "[{\"id\":\"vendas\",\"title\":\"Vendas\"},{\"id\":\"suporte\",\"title\":\"Suporte\"}]"
  }
}
```

- Até **3** opções saem como botões de resposta (título cortado em 20 caracteres).
- **4 ou mais** saem como lista, com o botão de abertura fixo em `Options` (título de cada linha
  cortado em 24 caracteres).
- `quick_replies` vazio ou inválido: a mensagem segue como texto simples.

Quando o cliente toca numa opção, chega um `message.received` com `contentType: "interactive"` e o
título da opção em `content.text`.

### O que sua aplicação recebe

Com `webhook_url` e `credentials.webhook_secret` definidos, cada evento do canal é enviado por `POST`
à sua aplicação. Mensagem recebida:

```json
{
  "id": "evt_5d0f4e6a-received",
  "type": "message.received",
  "timestamp": "2026-11-24T14:03:11Z",
  "tenantId": "3f6f0c1e-9a51-4f0e-b0a5-6b6f6d2c1a11",
  "environment": "production",
  "data": {
    "message": {
      "id": "5d0f4e6a-2b7c-4f43-9d1a-8c2e7b3a9f10",
      "direction": "inbound",
      "contentType": "text",
      "content": { "text": "Oi, quero saber do meu pedido" },
      "senderId": "5511999999999",
      "senderType": "contact",
      "metadata": {
        "senderName": "Maria Souza",
        "externalId": "wamid.HBgNNTUxMTk5OTk5OTk5ORUCABIYFjNFQjBDMEE1"
      }
    },
    "conversationId": "b3a9d2c4-55e1-4f0a-8a44-0f6f1f1f7a21",
    "contactId": "0c2b7f7e-3f1d-4d57-a0ad-0f9e5f1a2b33",
    "channelId": "CHANNEL_ID",
    "channelType": "whatsapp_official"
  }
}
```

Mídia chega em `data.message.content.media` (`url`, `mimeType`, `filename`, `size`, `caption`).

### Status de entrega e pricing {#status}

Status de uma mensagem enviada (`message.sent`, `message.delivered`, `message.read`,
`message.failed`):

```json
{
  "id": "evt_7c1f0a52-message.delivered_pricing",
  "type": "message.delivered",
  "timestamp": "2026-11-24T14:05:42Z",
  "tenantId": "3f6f0c1e-9a51-4f0e-b0a5-6b6f6d2c1a11",
  "environment": "production",
  "data": {
    "messageId": "7c1f0a52-0d1e-4c3b-9b7e-2a9c1f4e8d10",
    "status": "delivered",
    "direction": "outbound",
    "timestamp": "2026-11-24T14:05:42Z",
    "channelId": "CHANNEL_ID",
    "channelType": "whatsapp_official",
    "pricing": {
      "billable": true,
      "pricingModel": "PMP",
      "category": "utility"
    }
  }
}
```

- `pricing` é repassado como a Meta enviou (`billable`, `pricing_model`, `category`) e **só existe
  neste canal**. Use-o para saber se a mensagem foi cobrada e em qual categoria.
- O Linktor emite um `message.sent` assim que a Meta aceita o envio; a Meta manda depois o próprio
  status `sent`, já com `pricing`. Por isso você pode receber **dois** `message.sent` da mesma
  mensagem: um sem e outro com `pricing`. Trate o evento como idempotente por `messageId` + `status`.
- Em `message.failed`, `data.error` traz o erro da Meta no formato `[código] título: mensagem`.

Cada entrega leva os headers `X-Linktor-Signature` (HMAC-SHA256 em hexadecimal de
`timestamp + "." + corpo`, com o `webhook_secret` do canal) e `X-Linktor-Timestamp`. Verificação,
reentrega e lista de eventos em [Receber eventos](../api/receber-eventos.md).

## Capacidades extras da API

Rotas específicas do canal oficial. Um canal habilitado, com `access_token` e `phone_number_id`,
registra os clientes dessas rotas na subida do servidor e a cada alteração do canal.

| Recurso | Rotas | Estado |
|---|---|---|
| **WhatsApp Flows** | `POST /whatsapp/flows`, `PUT` e `DELETE /whatsapp/flows/:id`, `POST /whatsapp/flows/:id/publish`, `GET /whatsapp/flows/:id/preview`, `POST /whatsapp/flows/sync` | Implementado. O corpo leva `channel_id`, `name`, `categories`, `endpoint_uri`. Não há rota de listagem: use `sync` |
| **Analytics por canal** | `GET /channels/:id/analytics/conversations`, `/phone`, `/templates/:templateId`, `/stats`, `/export`, `/dashboard` | Implementado; consulta a Meta. Exige `business_id` (ou `waba_id`) |
| **Pagamentos** | `POST /channels/:id/payments`, `GET .../payments/stats`, `/:paymentId`, `/reference/:referenceId`, `/customer/:phone`, `POST .../:paymentId/refund`; webhook `POST /webhooks/payments/:channelId` | Parcial. Persistido em banco; depende de gateway configurado em `config` (`payment_gateway_type`, `payment_api_key`, `payment_api_secret`, `payment_merchant_id`, `payment_webhook_secret`) e da disponibilidade de pagamentos da Meta para o seu país |
| **Chamadas** | `GET` e `POST /channels/:id/calls`, `/stats`, `/phone/:phone`, `/:callId`, `/:callId/end`, `/:callId/quality`, `/:callId/recording`, `/:callId/reject`; webhook `POST /webhooks/calls/:channelId` | Parcial. O registro de chamadas fica **em memória** e se perde ao reiniciar o servidor; exige a API de chamadas habilitada no número |
| **Click-to-WhatsApp Ads (CTWA)** | `GET /channels/:id/ctwa/stats`, `/dashboard`, `/report`, `/top-ads`, `/free-window/:phone`, `/referrals/...`, `POST /channels/:id/ctwa/conversions`; webhook `POST /webhooks/ctwa/:channelId` | Parcial. Referências e conversões ficam **em memória** e se perdem ao reiniciar |

A aba **Operações** do canal no painel usa pagamentos, chamadas e CTWA. Painel de métricas gerais em
[Analytics](../painel/analytics.md).

## Limites e particularidades

- **Janela de 24 horas.** Mensagem livre só dentro de 24h desde a última mensagem do cliente. O
  Linktor tem uma guarda própria, controlada por `LINKTOR_WA_WINDOW_ENFORCEMENT`, com os valores `off`,
  `dry_run` (padrão) e `enforce`. Em `enforce`, uma mensagem livre fora da janela é bloqueada antes de
  chegar à Meta e vira `message.failed` com a razão `24h window`; em `dry_run` ela é enviada e só fica
  registrada em log e métrica. Modelos nunca são bloqueados por essa guarda. Mais em
  [Configuração](../implantar/configuracao.md).
- **Estado do modelo.** `LINKTOR_WA_TEMPLATE_ENFORCEMENT` (mesmos valores) controla uma segunda guarda, que bloqueia
  modelos com estado rejeitado, pausado, desativado ou excluído. Modelo ainda não sincronizado passa e
  a Meta decide.
- **Mídia.** Uma mídia por mensagem. Tipos declarados pelo canal: JPEG e PNG; AAC, MP4, MPEG, AMR e
  OGG de áudio; MP4 e 3GP de vídeo; PDF, documentos do Office e texto. Os limites de tamanho são os da
  Meta.
- **Erros.** Erro 4xx da Meta (exceto limite de taxa) é permanente e não é reenviado; limite de taxa,
  5xx e falha de rede são reenviados.
- **Qualidade e limite de envio.** Mudanças chegam por `phone_number_quality_update` e ficam em
  `config.quality_rating` e `config.messaging_limit_tier`.
- **Token.** O token temporário da Meta expira em 24h; use o de usuário do sistema.
- **Telefone.** Envie `to` em formato internacional (`+5511999999999`); o Linktor normaliza.

## Solução de problemas

| Sintoma | Causa | O que fazer |
|---|---|---|
| A Meta recusa **Verificar e salvar** | Token de verificação diferente, URL com id errado, ou servidor sem HTTPS válido | Copie URL e token da aba **Webhook**; teste com `curl "URL?hub.mode=subscribe&hub.verify_token=TOKEN&hub.challenge=123"` — deve devolver `123` |
| Webhook responde 401 `webhook secret not configured` | Canal sem **Segredo do App** e servidor em modo `release` | Preencha **Segredo do App** e salve. Comum em canais criados pelo Embedded Signup |
| Webhook responde 401 `invalid signature` | **Segredo do App** de outro app, ou proxy alterando o corpo | Confira o App Secret do app que está assinado na WABA; não reescreva o corpo no proxy |
| Verificação passa, mas nenhuma mensagem chega | Campo `messages` não assinado, ou app não assinado na WABA | Assine `messages` e rode o `POST /WABA_ID/subscribed_apps` |
| `message.failed` com `24h window` | Mensagem livre fora da janela | Envie um modelo aprovado |
| `message.failed` com erro `131047` | A própria Meta recusou por janela expirada | Envie um modelo aprovado |
| `message.failed` com erro `190` | Token expirado ou revogado | Gere um token de usuário do sistema e atualize **Token de Acesso** |
| `message.failed` com erro `131030` | App em modo de desenvolvimento e destinatário fora da lista de teste | Adicione o número em **Configuração da API**, ou publique o app |
| `POST /messages/send` responde `channel is not connected` | Canal criado mas não conectado | Clique em **Conectar** no menu do canal ou chame `POST /channels/CHANNEL_ID/connect` |
| Modelos não sincronizam | **ID da Conta WhatsApp Business** vazio | Preencha o campo com o WABA ID |
| Coexistência em `warning` ou `disconnected` | Aplicativo sem uso há 10 ou 14 dias | Abra o WhatsApp Business App no celular |
| Eventos não chegam à sua aplicação | `webhook_url` vazio ou canal sem `webhook_secret` (sem segredo, o Linktor descarta a entrega em vez de enviar sem assinatura) | Defina os dois pela API |

Logs por canal e fila de entrega ficam em [Observabilidade](../painel/observabilidade.md).
