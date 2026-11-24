---
sidebar_position: 8
title: E-mail
description: Conecte uma caixa de e-mail ao Linktor por SMTP/IMAP, SendGrid, Mailgun, Amazon SES ou Postmark — campos, webhooks de entrada por provedor, assunto, DNS e solução de problemas.
---

# E-mail

O canal de e-mail transforma cada mensagem recebida numa conversa do Linktor e envia as respostas
pelo provedor que você escolher. São cinco provedores: **SMTP** (com recebimento por **IMAP**),
**SendGrid**, **Mailgun**, **Amazon SES** e **Postmark**.

Valor de `type` na API: **`email`**. O provedor vai em `config.provider`.

| Provedor (`config.provider`) | Envio | Recebimento de e-mails | Status de entrega |
|---|---|---|---|
| `smtp` | SMTP | IMAP (consulta periódica da caixa) | Não |
| `sendgrid` | API do SendGrid | Webhook (Inbound Parse) | Webhook (Event Webhook) |
| `mailgun` | API do Mailgun | Webhook (Routes) | Webhook |
| `postmark` | API do Postmark | Webhook (Inbound) | Webhook |
| `ses` | API do Amazon SES | **Não** | Webhook (notificações SNS) |

| Recurso | Suporte |
|---|---|
| Texto | Sim. O corpo em texto é usado; se o e-mail só tiver HTML, o Linktor extrai o texto visível. |
| Anexos | Recebe (re-hospedados no armazenamento de mídia) e envia (o arquivo é baixado e anexado). |
| Assunto | Recebido e gravado com a mensagem; no envio, via `metadata.subject`. |
| Cópia, cópia oculta, responder-para, HTML | No envio, via `metadata` — veja [Enviar](#enviar). |
| Botões, modelos, grupos | Não se aplicam. |
| Confirmação de leitura | Só com provedores que informam abertura (vira `read`). |

## Antes de começar

- Uma conta no provedor escolhido e um **endereço remetente** que você controla.
- **DNS do domínio remetente** com SPF e DKIM configurados no provedor (seção abaixo). Sem isso as
  mensagens caem em spam ou são recusadas.
- Para SMTP/IMAP: host, porta, usuário e senha. Em contas com verificação em duas etapas (Gmail,
  Microsoft 365), gere uma **senha de aplicativo**; a senha normal é recusada.
- Para receber por webhook (SendGrid, Mailgun, Postmark): o servidor do Linktor acessível pela
  internet em HTTPS e um domínio ou subdomínio cujo **MX** você possa apontar para o provedor.
- Para anexos: o armazenamento de mídia do servidor configurado.

## Configurar na origem

Os nomes de menu abaixo são os atuais de cada provedor e mudam com frequência; procure pelo conceito
se o caminho não bater.

### DNS: SPF, DKIM e DMARC

Feito **no provedor e no seu DNS**, não no Linktor:

1. No provedor, adicione e verifique o domínio remetente (SendGrid: **Settings → Sender
   Authentication**; Mailgun: **Sending → Domains**; SES: **Verified identities**; Postmark:
   **Sender Signatures**).
2. O provedor mostra os registros a criar — normalmente um `TXT` de **SPF** e um ou mais `CNAME`/`TXT`
   de **DKIM**. Crie-os no seu DNS e aguarde a verificação.
3. Recomendado: publique um registro **DMARC** (`_dmarc.suaempresa.com`) começando com `p=none`.
4. Use em **E-mail Remetente** um endereço desse domínio verificado.

No SMTP de um provedor de caixa postal (Google Workspace, Microsoft 365), SPF e DKIM são os do
próprio serviço de e-mail.

### SMTP + IMAP

1. Anote host e porta de **SMTP** (`465` para TLS direto, `587` para STARTTLS).
2. Anote host de **IMAP**. O Linktor conecta por **TLS direto na porta 993**; a porta 143 com
   STARTTLS não é suportada.
3. Confirme que o IMAP está habilitado na conta e, se houver verificação em duas etapas, crie uma
   senha de aplicativo.

### SendGrid

1. **Settings → API Keys → Create API Key** com permissão **Mail Send**.
2. Para receber: **Settings → Inbound Parse → Add Host & URL**. Informe o subdomínio (cujo MX
   aponta para `mx.sendgrid.net`) e a URL do webhook do Linktor (seção [Webhook](#webhook)).
3. Para status: **Settings → Mail Settings → Event Webhook**, com a mesma URL.

### Mailgun

1. **Sending → Domains**: adicione o domínio e copie a **API key** (em **API Security** ou nas
   configurações do domínio). Anote a região (US ou EU).
2. Para receber: **Receiving → Routes → Create Route**, com a ação
   `forward("URL_DO_WEBHOOK")`. O MX do domínio precisa apontar para o Mailgun.
3. Para status: **Sending → Webhooks**, adicione a URL para os eventos desejados.
4. Copie a **HTTP webhook signing key** (em **Sending → Webhooks**) — é com ela que o Mailgun assina
   os webhooks, e ela é diferente da API key.

### Postmark

1. **Servers → (seu servidor) → API Tokens**: copie o **Server API token**.
2. Para receber: no fluxo **Inbound** do servidor, em **Settings**, informe a URL do webhook. Para
   autenticar, coloque usuário e senha na própria URL
   (`https://qualquer:SENHA@linktor.suaempresa.com/api/v1/webhooks/email/CHANNEL_ID/postmark`).
3. Para status: no fluxo transacional, **Webhooks → Add webhook**, com a mesma URL e os eventos de
   entrega, bounce e spam.

### Amazon SES

1. **Verified identities**: verifique o domínio e saia do sandbox do SES.
2. No IAM, crie um usuário com permissão `ses:SendEmail` / `ses:SendRawEmail` e gere **Access Key
   ID** e **Secret Access Key**.
3. Para status: crie um tópico no **SNS**, configure a identidade para publicar *Bounce*,
   *Complaint* e *Delivery* nele e inscreva a URL do webhook como endpoint HTTPS.

:::warning Limitações do Amazon SES no Linktor
- O Linktor **não recebe e-mails** pelo SES: o webhook entende apenas notificações de entrega,
  bounce e reclamação. Para receber respostas, use outro provedor para a entrada.
- Na confirmação de inscrição do SNS, o Linktor responde `200`, mas **não visita** o `SubscribeURL`.
  A inscrição fica pendente até você confirmá-la manualmente com esse endereço.
:::

## Configurar no Linktor

1. No painel, abra **Canais** e, em **Adicionar Novo Canal**, clique em **E-mail**.
2. Preencha **Nome do Canal**, escolha o **Provedor de E-mail** e informe o remetente.
3. Na aba **Credenciais**, preencha os campos do provedor. Com SMTP, a aba **Recebimento** traz os
   campos de IMAP. Nos demais, a aba **Webhooks** mostra a URL e o **Segredo de Assinatura**.
4. Clique em **Testar Conexão**. O teste é real: autentica no provedor de envio e, se houver IMAP,
   conecta na caixa e abre a pasta. Nada é enviado.
5. Clique em **Criar Canal** e depois, no cartão do canal, em **Conectar**. O recebimento por IMAP só
   começa com o canal conectado e habilitado.

**Campos comuns**

| Campo na tela | Chave | Obrigatório |
|---|---|---|
| **Nome do Canal** | `name` | Sim |
| **Provedor de E-mail** | `config.provider` | Sim (`smtp`, `sendgrid`, `mailgun`, `ses`, `postmark`) |
| **E-mail Remetente** | `config.from_email` | Sim |
| **Nome Remetente (Opcional)** | `config.from_name` | Não |
| **Responder Para (Opcional)** | `config.reply_to` | Não |

**SMTP** (aba **Credenciais**) **e IMAP** (aba **Recebimento**)

| Campo na tela | Chave | Obrigatório | Observação |
|---|---|---|---|
| **Host SMTP** | `config.smtp_host` | Sim | |
| **Porta SMTP** | `config.smtp_port` | Sim | `465` ou `587`. |
| **Usuário** | `credentials.smtp_username` | Não | Sem usuário e senha, o envio não autentica. |
| **Senha** | `credentials.smtp_password` | Não | |
| **Criptografia** | `config.smtp_encryption` | Não | `tls` (**TLS (Recomendado)**, porta 465), `starttls` (padrão, porta 587) ou `none`. |
| **Host IMAP** | `config.imap_host` | Não | Vazio = canal só envia. |
| **Porta IMAP** | `config.imap_port` | Não | Padrão `993` (TLS). |
| **Usuário** / **Senha** (IMAP) | `credentials.imap_username` / `credentials.imap_password` | Sim, se houver host | Sem os dois o recebimento fica desligado. |
| **Pasta** | `config.imap_folder` | Não | Padrão `INBOX`. |
| **Intervalo de Verificação (segundos)** | `config.imap_poll_interval` | Não | Padrão `30`; abaixo de `15` volta para `30`. |

**SendGrid, Mailgun, Amazon SES e Postmark**

| Provedor | Campo na tela | Chave | Obrigatório |
|---|---|---|---|
| SendGrid | **Chave de API** | `credentials.sendgrid_api_key` | Sim |
| Mailgun | **Domínio** | `config.mailgun_domain` | Sim |
| Mailgun | **Região** | `config.mailgun_region` | Não (`us` ou `eu`; padrão `us`) |
| Mailgun | **Chave de API** | `credentials.mailgun_api_key` | Sim |
| Amazon SES | **Região** | `config.ses_region` | Sim |
| Amazon SES | **Access Key ID** | `credentials.ses_access_key_id` | Sim |
| Amazon SES | **Secret Access Key** | `credentials.ses_secret_key` | Sim |
| Postmark | **Token do Servidor** | `credentials.postmark_server_token` | Sim |
| Todos estes | **Segredo de Assinatura** | `credentials.webhook_secret` | Em produção, sim — veja [Webhook](#webhook) |

Na edição, os campos de segredo abrem vazios; deixe em branco para manter o valor guardado.

## Webhook

Com **SMTP não há webhook**: o Linktor consulta a caixa por IMAP. Para os demais provedores, a URL
é:

```text
https://linktor.suaempresa.com/api/v1/webhooks/email/CHANNEL_ID/sendgrid
https://linktor.suaempresa.com/api/v1/webhooks/email/CHANNEL_ID/mailgun
https://linktor.suaempresa.com/api/v1/webhooks/email/CHANNEL_ID/postmark
https://linktor.suaempresa.com/api/v1/webhooks/email/CHANNEL_ID/ses
```

A mesma URL recebe e-mails de entrada e eventos de status. Quem decide como interpretar o corpo é o
`config.provider` do canal; o sufixo da URL só é usado quando o canal não tem provedor gravado.

Como cada provedor é autenticado:

| Provedor | Verificação | Segredo usado (primeira chave preenchida em `credentials`) |
|---|---|---|
| Mailgun | HMAC-SHA256 de `timestamp + token`, comparado com `signature` (campos enviados pelo Mailgun) | `mailgun_webhook_signing_key`, `webhook_secret`, `mailgun_api_key` |
| Postmark | Header `Authorization` (Basic — vale a senha —, Bearer ou valor puro) | `postmark_webhook_password`, `webhook_password`, `webhook_secret` |
| SendGrid e SES | HMAC-SHA256 do corpo, no header `X-Linktor-Signature` ou `X-Hub-Signature-256`, formato `sha256=<hex>` | `webhook_secret` |

Sem nenhum segredo gravado, o webhook é aceito **apenas fora de produção**. Com
`LINKTOR_SERVER_MODE=release` (ou `LINKTOR_REQUIRE_WEBHOOK_SECRETS=true`), a resposta é
`401 webhook secret not configured`.

:::warning SendGrid e SES não assinam nesse formato
O Inbound Parse do SendGrid e as notificações SNS do SES **não enviam** os headers
`X-Linktor-Signature` / `X-Hub-Signature-256`. Consequência prática:

- com **Segredo de Assinatura** preenchido, as chamadas diretas desses provedores são recusadas com
  `401 invalid signature`;
- sem segredo, em produção, são recusadas com `401 webhook secret not configured`.

Para usar SendGrid ou SES em produção é preciso um intermediário seu (uma função ou proxy) que receba
a chamada do provedor, calcule o HMAC-SHA256 do corpo com o segredo e repasse ao Linktor com o
header `X-Hub-Signature-256`. Mailgun e Postmark funcionam direto.
:::

:::warning O mesmo segredo assina os eventos para a sua aplicação
`credentials.webhook_secret` também é a chave com que o Linktor assina os eventos enviados ao
`webhook_url` do canal (`message.received` e os demais). Para não compartilhar um segredo entre o
provedor e a sua aplicação, grave pela API o segredo do provedor na chave específica —
`mailgun_webhook_signing_key` ou `postmark_webhook_password` — e reserve `webhook_secret` para a sua
aplicação. Com SendGrid e SES não há chave separada.
:::

## Criar o canal pela API

O exemplo cria um canal SMTP com recebimento por IMAP. Para outro provedor, troque `provider` e as
chaves conforme as tabelas acima.

<Tabs groupId="linguagem">
<TabItem value="curl" label="cURL">

```bash
# 1. Cria o canal (nasce "disconnected")
curl -X POST "$LINKTOR_URL/api/v1/channels" \
  -H "X-API-Key: $LINKTOR_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "type": "email",
    "name": "Atendimento por e-mail",
    "config": {
      "provider": "smtp",
      "from_email": "atendimento@suaempresa.com",
      "from_name": "Sua Empresa",
      "smtp_host": "smtp.suaempresa.com",
      "smtp_port": "587",
      "smtp_encryption": "starttls",
      "imap_host": "imap.suaempresa.com",
      "imap_port": "993",
      "imap_folder": "INBOX",
      "imap_poll_interval": "30"
    },
    "credentials": {
      "smtp_username": "atendimento@suaempresa.com",
      "smtp_password": "senha-de-aplicativo",
      "imap_username": "atendimento@suaempresa.com",
      "imap_password": "senha-de-aplicativo",
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
  "type": "email",
  "name": "Atendimento por e-mail",
  "config": {
    "provider": "smtp",
    "from_email": "atendimento@suaempresa.com",
    "from_name": "Sua Empresa",
    "smtp_host": "smtp.suaempresa.com",
    "smtp_port": "587",
    "smtp_encryption": "starttls",
    "imap_host": "imap.suaempresa.com",
    "imap_port": "993",
    "imap_folder": "INBOX",
    "imap_poll_interval": "30"
  },
  "credentials": {
    "smtp_username": "atendimento@suaempresa.com",
    "smtp_password": "senha-de-aplicativo",
    "imap_username": "atendimento@suaempresa.com",
    "imap_password": "senha-de-aplicativo",
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
from linktor import LinktorClient

linktor = LinktorClient(
    base_url=os.environ["LINKTOR_URL"] + "/api/v1",
    api_key=os.environ["LINKTOR_API_KEY"],
)

channel = linktor.channels.create(
    name="Atendimento por e-mail",
    type="email",
    config={
        "provider": "smtp",
        "from_email": "atendimento@suaempresa.com",
        "from_name": "Sua Empresa",
        "smtp_host": "smtp.suaempresa.com",
        "smtp_port": "587",
        "smtp_encryption": "starttls",
        "imap_host": "imap.suaempresa.com",
        "imap_port": "993",
        "imap_folder": "INBOX",
        "imap_poll_interval": "30"
    },
    credentials={
        "smtp_username": "atendimento@suaempresa.com",
        "smtp_password": "senha-de-aplicativo",
        "imap_username": "atendimento@suaempresa.com",
        "imap_password": "senha-de-aplicativo",
        "webhook_secret": "um-segredo-longo-e-aleatorio"
    },
    webhook_url="https://app.suaempresa.com/webhooks/linktor",
)

linktor.channels.connect(channel.id)
print(channel.id)
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
		Name: "Atendimento por e-mail",
		Type: types.ChannelType("email"),
		Config: map[string]string{
			"provider": "smtp",
			"from_email": "atendimento@suaempresa.com",
			"from_name": "Sua Empresa",
			"smtp_host": "smtp.suaempresa.com",
			"smtp_port": "587",
			"smtp_encryption": "starttls",
			"imap_host": "imap.suaempresa.com",
			"imap_port": "993",
			"imap_folder": "INBOX",
			"imap_poll_interval": "30",
		},
		Credentials: map[string]string{
			"smtp_username": "atendimento@suaempresa.com",
			"smtp_password": "senha-de-aplicativo",
			"imap_username": "atendimento@suaempresa.com",
			"imap_password": "senha-de-aplicativo",
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
        .type(Channel.ChannelType.EMAIL)
        .name("Atendimento por e-mail")
        .config(Map.of(
            "provider", "smtp",
            "from_email", "atendimento@suaempresa.com",
            "from_name", "Sua Empresa",
            "smtp_host", "smtp.suaempresa.com",
            "smtp_port", "587",
            "smtp_encryption", "starttls",
            "imap_host", "imap.suaempresa.com",
            "imap_port", "993",
            "imap_folder", "INBOX",
            "imap_poll_interval", "30"))
        .credentials(Map.of(
            "smtp_username", "atendimento@suaempresa.com",
            "smtp_password", "senha-de-aplicativo",
            "imap_username", "atendimento@suaempresa.com",
            "imap_password", "senha-de-aplicativo",
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
  "type": "email",
  "name": "Atendimento por e-mail",
  "config": {
    "provider": "smtp",
    "from_email": "atendimento@suaempresa.com",
    "from_name": "Sua Empresa",
    "smtp_host": "smtp.suaempresa.com",
    "smtp_port": "587",
    "smtp_encryption": "starttls",
    "imap_host": "imap.suaempresa.com",
    "imap_port": "993",
    "imap_folder": "INBOX",
    "imap_poll_interval": "30"
  },
  "credentials": {
    "smtp_username": "atendimento@suaempresa.com",
    "smtp_password": "senha-de-aplicativo",
    "imap_username": "atendimento@suaempresa.com",
    "imap_password": "senha-de-aplicativo",
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
    'type' => 'email',
    'name' => 'Atendimento por e-mail',
    'config' => [
        'provider' => 'smtp',
        'from_email' => 'atendimento@suaempresa.com',
        'from_name' => 'Sua Empresa',
        'smtp_host' => 'smtp.suaempresa.com',
        'smtp_port' => '587',
        'smtp_encryption' => 'starttls',
        'imap_host' => 'imap.suaempresa.com',
        'imap_port' => '993',
        'imap_folder' => 'INBOX',
        'imap_poll_interval' => '30',
    ],
    'credentials' => [
        'smtp_username' => 'atendimento@suaempresa.com',
        'smtp_password' => 'senha-de-aplicativo',
        'imap_username' => 'atendimento@suaempresa.com',
        'imap_password' => 'senha-de-aplicativo',
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

    let input = CreateChannelInput::new("Atendimento por e-mail", ChannelType::Email)
        .config(to_map(&[
            ("provider", "smtp"),
            ("from_email", "atendimento@suaempresa.com"),
            ("from_name", "Sua Empresa"),
            ("smtp_host", "smtp.suaempresa.com"),
            ("smtp_port", "587"),
            ("smtp_encryption", "starttls"),
            ("imap_host", "imap.suaempresa.com"),
            ("imap_port", "993"),
            ("imap_folder", "INBOX"),
            ("imap_poll_interval", "30"),
        ]))
        .credentials(to_map(&[
            ("smtp_username", "atendimento@suaempresa.com"),
            ("smtp_password", "senha-de-aplicativo"),
            ("imap_username", "atendimento@suaempresa.com"),
            ("imap_password", "senha-de-aplicativo"),
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

O canal nasce `disconnected`; o `connect` o marca como conectado e inicia a consulta da caixa IMAP.
`webhook_url` e `credentials.webhook_secret` são o endereço e o segredo da **sua aplicação**.

Para validar credenciais antes de criar, use `POST /api/v1/channels/test-email` com as mesmas chaves
de `config` e `credentials`: a resposta descreve o que foi verificado (por exemplo
`envio ok via smtp, recebimento ok via IMAP em imap.suaempresa.com (3 não lidas)`).

:::warning PUT substitui o config
No `PUT /channels/CHANNEL_ID`, o `config` enviado substitui o anterior por inteiro; só `credentials`
é mesclado. Para trocar apenas a pasta IMAP, reenvie todas as chaves de `config`.
:::

## Enviar e receber

### Enviar {#enviar}

`to` é o endereço de e-mail do destinatário. O texto vira o corpo em texto puro.

<Tabs groupId="linguagem">
<TabItem value="curl" label="cURL">

```bash
curl -X POST "$LINKTOR_URL/api/v1/messages/send" \
  -H "X-API-Key: $LINKTOR_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "channel_id": "'"$CHANNEL_ID"'",
    "to": "cliente@exemplo.com",
    "text": "Olá! Seu pedido 1234 foi enviado e chega em até 3 dias úteis.",
    "metadata": {
      "subject": "Seu pedido 1234 foi enviado"
    }
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
    "channel_id": process.env.CHANNEL_ID,
    "to": "cliente@exemplo.com",
    "text": "Olá! Seu pedido 1234 foi enviado e chega em até 3 dias úteis.",
    "metadata": {
      "subject": "Seu pedido 1234 foi enviado"
    }
  }),
});

if (!resp.ok) throw new Error(`HTTP ${resp.status}: ${await resp.text()}`);
const { data } = await resp.json();
console.log(data.id, data.status); // status "queued"
```

</TabItem>
<TabItem value="python" label="Python">

```python
import os
import requests

resp = requests.post(
    os.environ["LINKTOR_URL"] + "/api/v1/messages/send",
    headers={"X-API-Key": os.environ["LINKTOR_API_KEY"]},
    json={
        "channel_id": os.environ["CHANNEL_ID"],
        "to": "cliente@exemplo.com",
        "text": "Olá! Seu pedido 1234 foi enviado e chega em até 3 dias úteis.",
        "metadata": {
            "subject": "Seu pedido 1234 foi enviado"
        }
    },
)
resp.raise_for_status()
data = resp.json()["data"]
print(data["id"], data["status"])  # status "queued"
```

</TabItem>
<TabItem value="go" label="Go">

```go
package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
)

func main() {
	body, _ := json.Marshal(map[string]any{
		"channel_id": os.Getenv("CHANNEL_ID"),
		"to": "cliente@exemplo.com",
		"text": "Olá! Seu pedido 1234 foi enviado e chega em até 3 dias úteis.",
		"metadata": map[string]string{
			"subject": "Seu pedido 1234 foi enviado",
		},
	})

	req, _ := http.NewRequest(http.MethodPost,
		os.Getenv("LINKTOR_URL")+"/api/v1/messages/send", bytes.NewReader(body))
	req.Header.Set("X-API-Key", os.Getenv("LINKTOR_API_KEY"))
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Fatal(err)
	}
	defer resp.Body.Close()

	out, _ := io.ReadAll(resp.Body)
	fmt.Println(resp.StatusCode, string(out)) // 202 + data.status "queued"
}
```

</TabItem>
<TabItem value="java" label="Java">

```java
import java.net.URI;
import java.net.http.HttpClient;
import java.net.http.HttpRequest;
import java.net.http.HttpResponse;

String body = """
    {
      "channel_id": "%s",
      "to": "cliente@exemplo.com",
      "text": "Olá! Seu pedido 1234 foi enviado e chega em até 3 dias úteis.",
      "metadata": {
        "subject": "Seu pedido 1234 foi enviado"
      }
    }
    """.formatted(System.getenv("CHANNEL_ID"));

HttpRequest request = HttpRequest.newBuilder()
    .uri(URI.create(System.getenv("LINKTOR_URL") + "/api/v1/messages/send"))
    .header("X-API-Key", System.getenv("LINKTOR_API_KEY"))
    .header("Content-Type", "application/json")
    .POST(HttpRequest.BodyPublishers.ofString(body))
    .build();

HttpResponse<String> response = HttpClient.newHttpClient()
    .send(request, HttpResponse.BodyHandlers.ofString());
System.out.println(response.statusCode() + " " + response.body()); // 202
```

</TabItem>
<TabItem value="csharp" label="C#">

```csharp
using System.Net.Http;
using System.Net.Http.Json;

using var http = new HttpClient();
http.DefaultRequestHeaders.Add("X-API-Key", Environment.GetEnvironmentVariable("LINKTOR_API_KEY"));

var payload = new
{
    channel_id = Environment.GetEnvironmentVariable("CHANNEL_ID"),
    to = "cliente@exemplo.com",
    text = "Olá! Seu pedido 1234 foi enviado e chega em até 3 dias úteis.",
    metadata = new Dictionary<string, string>
    {
        ["subject"] = "Seu pedido 1234 foi enviado",
    },
};

var resp = await http.PostAsJsonAsync(
    Environment.GetEnvironmentVariable("LINKTOR_URL") + "/api/v1/messages/send", payload);
resp.EnsureSuccessStatusCode(); // 202
Console.WriteLine(await resp.Content.ReadAsStringAsync());
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
        'channel_id' => getenv('CHANNEL_ID'),
        'to' => 'cliente@exemplo.com',
        'text' => 'Olá! Seu pedido 1234 foi enviado e chega em até 3 dias úteis.',
        'metadata' => [
            'subject' => 'Seu pedido 1234 foi enviado',
        ],
    ],
]);

$data = json_decode((string) $resp->getBody(), true)['data'];
echo $data['id'], ' ', $data['status'], PHP_EOL; // status "queued"
```

</TabItem>
<TabItem value="rust" label="Rust">

```rust
// Cargo.toml: reqwest = { version = "0.12", features = ["json"] }, serde_json, tokio
use serde_json::json;

#[tokio::main]
async fn main() -> Result<(), Box<dyn std::error::Error>> {
    let resp = reqwest::Client::new()
        .post(format!("{}/api/v1/messages/send", std::env::var("LINKTOR_URL")?))
        .header("X-API-Key", std::env::var("LINKTOR_API_KEY")?)
        .json(&json!({
            "channel_id": std::env::var("CHANNEL_ID")?,
            "to": "cliente@exemplo.com",
            "text": "Olá! Seu pedido 1234 foi enviado e chega em até 3 dias úteis.",
            "metadata": {
                "subject": "Seu pedido 1234 foi enviado"
            }
        }))
        .send()
        .await?
        .error_for_status()?;

    println!("{}", resp.text().await?); // 202 + data.status "queued"
    Ok(())
}
```

</TabItem>
</Tabs>

Chaves de `metadata` que o canal de e-mail interpreta:

| Chave | Efeito |
|---|---|
| `subject` | Assunto. Sem ela, o assunto sai como `Message`. |
| `html_body` | Corpo alternativo em HTML (o `text` continua indo como texto puro). |
| `cc`, `bcc` | Endereços separados por vírgula. |
| `reply_to` | Endereço de resposta desta mensagem. |
| `in_reply_to`, `references` | `Message-ID` da mensagem respondida, para manter o fio no cliente de e-mail do destinatário. |
| `idempotency_key` | Repetir a chamada com a mesma chave não envia de novo. |

Para anexar um arquivo, use `content_type` `document` (ou `image`, `audio`, `video`) com a URL em
`attachments`. O Linktor baixa o arquivo e o anexa de verdade; se o download falhar, a URL é
acrescentada ao corpo para a mídia não se perder.

A resposta é `202` com `data.status = "queued"`. O desfecho chega como `message.sent` (o provedor
aceitou) ou `message.failed`. Mais opções em [Enviar mensagens](../api/enviar-mensagens.md).

### Receber

Cada e-mail recebido gera um `message.received` no `webhook_url` do canal (assinatura em
[Receber eventos](../api/receber-eventos.md)):

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
      "contentType": "text",
      "content": {
        "text": "Boa tarde, segue o comprovante em anexo.",
        "media": {
          "url": "https://linktor.suaempresa.com/media/comprovante.pdf",
          "mimeType": "application/pdf",
          "filename": "comprovante.pdf",
          "size": 48211
        }
      },
      "senderId": "cliente@exemplo.com",
      "senderType": "contact",
      "metadata": {
        "senderName": "Maria Souza",
        "externalId": "<CAF1x9k2b7@mail.exemplo.com>"
      }
    },
    "conversationId": "c1a5d3e2-7b84-4a60-8e55-2f1d9b0c7a66",
    "contactId": "7e9a4b10-5c2d-4f3e-b8a1-6d0f2e3c4b5a",
    "channelId": "CHANNEL_ID",
    "channelType": "email"
  }
}
```

- `senderId` é o endereço do remetente; `metadata.externalId` é o `Message-ID` do e-mail.
- O evento traz **apenas o primeiro anexo** em `content.media`. A forma da URL depende do
  armazenamento do servidor.
- O **assunto não vai no evento**: ele fica gravado nos metadados da mensagem dentro do Linktor,
  mas `metadata` do evento só leva `senderName` e `externalId`.

## Limites e particularidades

- **IMAP marca como lido.** O Linktor busca as mensagens **não lidas** da pasta e as marca como
  lidas ao processar. Use uma caixa dedicada ao atendimento: uma mensagem aberta antes por uma
  pessoa não entra no Linktor.
- **Até 25 mensagens por rodada.** Uma caixa com muito histórico não lido é drenada aos poucos, 25
  por consulta — e cada uma vira conversa. Marque o histórico como lido antes de conectar.
- **Só TLS na porta 993** para IMAP.
- **Status de entrega** depende do provedor: `delivered` quando ele confirma a entrega, `read`
  quando informa abertura ou clique, `failed` em bounce, falha ou marcação como spam. SMTP não
  informa nada depois do aceite.
- **Erros permanentes de SMTP** (respostas `5xx`: credencial, destinatário, tamanho) encerram a
  mensagem como falha de imediato; erros de rede e `4xx` são tentados de novo.
- **HTML recebido** é convertido em texto: formatação, imagens embutidas e assinaturas em HTML não
  são preservadas.
- **Um provedor por canal.** Para enviar por um e receber por outro, crie dois canais.

## Solução de problemas

| Sintoma | Causa provável | O que fazer |
|---|---|---|
| **Testar Conexão** falha em `envio (smtp)` | Host, porta ou criptografia incompatíveis; senha normal em conta com duas etapas | `465` pede **TLS**, `587` pede **STARTTLS**. Gere uma senha de aplicativo. |
| **Testar Conexão** falha em `recebimento (IMAP ...)` | IMAP desabilitado, pasta inexistente ou porta diferente de 993 | Habilite o IMAP na conta e confira o nome exato da pasta. |
| O canal envia, mas nenhuma resposta chega | Sem IMAP configurado (SMTP), webhook de entrada não cadastrado, ou canal não conectado | Preencha a aba **Recebimento** ou cadastre a URL no provedor; clique em **Conectar**. |
| E-mails antigos não aparecem | Já estavam marcados como lidos | Só mensagens não lidas são importadas. |
| Dezenas de conversas antigas surgem ao conectar | Caixa com histórico não lido | Marque o histórico como lido antes de conectar. |
| Webhook responde `401 invalid signature` (Mailgun) | Segredo gravado é a API key, não a signing key | Grave a **HTTP webhook signing key** em **Segredo de Assinatura**. |
| Webhook responde `401 invalid authorization` (Postmark) | URL do webhook sem usuário e senha, ou senha diferente | Use `https://qualquer:SENHA@...` no Postmark, com a mesma senha gravada no canal. |
| Webhook responde `401` com SendGrid ou SES | O provedor não assina no formato esperado | Veja o aviso em [Webhook](#webhook). |
| Inscrição SNS fica em "Pending confirmation" | O Linktor não confirma a inscrição | Confirme manualmente com o `SubscribeURL` da mensagem do SNS. |
| Mensagens chegam em spam | SPF/DKIM do domínio remetente ausentes ou remetente não verificado | Conclua a autenticação do domínio no provedor. |
| Todo e-mail sai com assunto `Message` | Envio sem `metadata.subject` | Informe `metadata.subject`. |
| A resposta abre um fio novo no cliente do destinatário | Envio pela API sem `in_reply_to` / `references` | Informe o `Message-ID` da mensagem respondida (o `metadata.externalId` do evento). |
| O webhook é chamado, mas nada aparece no painel | Canal desconectado ou desabilitado: o recebimento é descartado (`channel is not active`) | Clique em **Conectar** e confira se o canal está habilitado. |
| Nenhum evento chega na sua aplicação | Canal sem `webhook_url` ou sem `credentials.webhook_secret` | Defina os dois pela API. |

Veja também: [Visão geral dos canais](visao-geral.md) · [Canais no painel](../painel/canais.md) ·
[API de canais](../api/canais.md).
