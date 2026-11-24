---
sidebar_position: 3
title: WhatsApp não oficial
description: Conectar um número de WhatsApp como dispositivo vinculado (QR code, código de pareamento ou passkey), enviar para contatos e grupos, e lidar com a recusa de versão do cliente (erro 405).
---

# WhatsApp não oficial

## Resumo

O canal não oficial conecta um número comum de WhatsApp como **dispositivo vinculado** — o mesmo
mecanismo do WhatsApp Web — usando a biblioteca `whatsmeow`. Não há conta na Meta, aprovação, modelo de
mensagem nem custo por mensagem. Em troca, o uso contraria os termos do WhatsApp e o número pode ser
banido; leia [Riscos](#riscos) antes de colocar em produção.

- Valor de `type` na API: **`whatsapp`**
- Nome no painel: **WhatsApp (não oficial)** ("Sessão do WhatsApp neste servidor; QR Code aqui no admin")
- A sessão fica **no servidor Linktor**, num arquivo por canal em `storages/whatsapp_<id do canal>.db`

:::note `whatsapp` ou `whatsapp_unofficial`?
O backend aceita os dois valores para conectar, parear, reconectar na subida do servidor e enviar. O
painel, porém, cria e edita apenas canais `whatsapp`: um canal criado pela API como
`whatsapp_unofficial` funciona, mas aparece no painel com "Configuração não disponível para este tipo
de canal". Use **`whatsapp`**, a menos que você nunca vá abrir o canal no painel.
:::

| Recurso | Suportado | Observação |
|---|---|---|
| Texto | Sim | Com resposta citada e reações |
| Imagem, vídeo, áudio, documento | Sim | Uma mídia por mensagem |
| Localização e contato | Sim | |
| Botões | Sim | Até 3 botões nativos; acima disso viram texto numerado |
| Lista | Como texto | Sempre enviada como texto numerado |
| Modelos de mensagem (templates) | Não | Não existem neste canal; não há janela de 24h |
| Grupos | Sim | Recebe e envia (texto) |
| Entregue / lida | Sim | `message.delivered` e `message.read` |
| Contatos do aparelho e foto de perfil | Sim | Rotas próprias, ver [Contatos e avatar](#contatos) |
| Chamadas | Parcial | Registra a chamada na conversa e pode recusá-la; o áudio não passa pelo Linktor |

Há uma variante, **WhatsApp via Bridge**, em que a sessão fica no computador do atendente — veja
[Bridge](#bridge).

## Antes de começar

1. Um número de WhatsApp ativo num celular, com o aplicativo atualizado e uma vaga livre em
   **Dispositivos conectados**.
2. Acesso ao celular na hora de conectar (para ler o QR code ou digitar o código).
3. O diretório `storages/` do servidor em **volume persistente**. Se ele se perder, todas as sessões se
   perdem e cada número precisa ser pareado de novo.
4. Saída do servidor liberada para os servidores do WhatsApp e para `web.whatsapp.com` (consulta da
   versão do cliente).
5. Se a conta usa **passkey**, um navegador Chrome ou Edge com a extensão
   [Linktor Passkey Connector](#passkey).

## Configurar na origem

Não há console de provedor. A "origem" é o celular:

1. Abra o WhatsApp no celular.
2. Vá em **Configurações** → **Dispositivos conectados** (em algumas versões, **Aparelhos conectados**).
3. Toque em **Conectar dispositivo**.
4. Deixe a câmera pronta: o QR code é gerado no passo seguinte, no Linktor.

Depois de vinculado, o celular não precisa ficar online. O WhatsApp desconecta dispositivos vinculados
quando o celular principal passa cerca de 14 dias sem abrir o aplicativo.

## Configurar no Linktor

1. No painel, abra **Canais** e clique em **WhatsApp (não oficial)**.
2. Na aba **Configuração**, preencha os campos e clique em **Criar Canal**.
3. Abra o canal de novo e vá à aba **Conexão**. Antes de salvar, ela mostra **Salve Primeiro**.
4. Clique em **Conectar com QR Code** e leia o código com o celular. O status passa de **Escanear QR
   Code** para **Conectado**, com a linha **Conectado como** e o número.

| Rótulo na tela | Chave | Onde fica | Obrigatório |
|---|---|---|---|
| **Nome do Canal** | `name` | raiz | Sim |
| **Ambiente do canal** | `environment` | raiz | Não (padrão `production`; imutável depois) |
| **Nome do Dispositivo** | `device_name` | `config` | Não (padrão `Linktor`) |
| **Receber mensagens de grupo** | `ignore_groups` (invertido: ligado grava `false`) | `config` | Não |
| **Sempre online** | `always_online` | `config` | Não |
| **Marcar como lida automaticamente** | `auto_read_messages` | `config` | Não |
| **Ignorar status/stories** | `ignore_status` | `config` | Não |
| **Rejeitar chamadas recebidas** | `reject_call` | `config` | Não |
| **Mensagem ao rejeitar a chamada** | `reject_call_msg` | `config` | Não |
| **Máx. de QR codes** | `qrcode_max_count` (`0` = ilimitado) | `config` | Não |
| **Host do proxy** / **Porta do proxy** | `proxy_host`, `proxy_port` | `config` | Não |
| **Usuário do proxy** / **Senha do proxy** | `proxy_user`, `proxy_pass` | `config` | Não |
| **URL do Consumidor** | `webhook_url` | raiz | Não |
| **Segredo de Assinatura** | `webhook_secret` | `credentials` | Sim, se houver `webhook_url` |
| **Eventos a entregar** | `webhook_events` | `config` | Não (vazio entrega todos) |

Todos os valores de `config` são **strings** (`"true"`, `"false"`, `"0"`). O canal não tem credenciais
de provedor: a "credencial" é a sessão pareada.

### Conectar por QR code {#qr}

Pela API, o fluxo é uma chamada mais consulta periódica:

```bash
curl -X POST "$LINKTOR_URL/api/v1/channels/CHANNEL_ID/connect" \
  -H "X-API-Key: $LINKTOR_API_KEY"
```

```json
{
  "success": true,
  "data": {
    "channel": { "id": "CHANNEL_ID", "type": "whatsapp", "connection_status": "connecting" },
    "qr_code": "2@AbCdEf...",
    "expires_in": 20
  }
}
```

1. `qr_code` é o **conteúdo** do QR: gere a imagem com uma biblioteca de QR code e mostre ao usuário.
2. Consulte `GET /channels/CHANNEL_ID` a cada poucos segundos até `connection_status` virar
   `connected`.
3. Quando `expires_in` (segundos) estourar, chame `connect` de novo para obter outro código.
4. Se o canal já tem sessão válida, `connect` responde direto com o canal conectado, sem `qr_code`.

Se o WhatsApp não entregar um QR em 30 segundos a chamada falha com `timeout waiting for QR code`.

### Conectar por código de pareamento {#pareamento}

Alternativa ao QR: o WhatsApp mostra um campo para digitar um código de 8 caracteres.

```bash
curl -X POST "$LINKTOR_URL/api/v1/channels/CHANNEL_ID/pair" \
  -H "X-API-Key: $LINKTOR_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{ "phone_number": "+5511999999999" }'
```

A resposta traz `data.pair_code` e `data.expires_in`. No celular, em **Conectar dispositivo**, toque em
**Conectar com número de telefone** e digite o código. Acompanhe `connection_status` como no QR.

:::warning Na tela, use o QR code
A aba **Conexão** tem a opção **Obter Código de Pareamento**, mas hoje ela lê um campo diferente do que
a API devolve e mostra "Login por código de pareamento ainda não está disponível". Pela API e pelos
SDKs (`requestPairCode`) o pareamento por código funciona.
:::

### Contas com passkey {#passkey}

Contas que o WhatsApp protege com **passkey** não podem ser vinculadas por QR a partir de um servidor:
o WhatsApp exige uma assinatura WebAuthn feita pelo dono da conta. Nesse caso `connect` responde sem
QR:

```json
{
  "success": true,
  "data": {
    "channel": { "id": "CHANNEL_ID", "connection_status": "connecting" },
    "passkey_required": true,
    "passkey_challenge": { "...": "desafio WebAuthn, repassado como veio do WhatsApp" }
  }
}
```

No painel, a aba **Conexão** mostra **Esta conta exige uma passkey** e o botão **Autorizar com
passkey**. O fluxo:

1. Instale a extensão **Linktor Passkey Connector** no navegador do dono da conta. Ela está em
   `web/passkey-extension` no repositório e não precisa de build: em `chrome://extensions`, ative o
   **Modo do desenvolvedor**, clique em **Carregar sem compactação** e escolha a pasta.
2. A extensão só conversa com as origens declaradas em `manifest.json` (`host_permissions` e
   `content_scripts[].matches`) e em `background.js` (`APP_HOST_PATTERNS`). O padrão cobre
   `https://*.linktor.dev/*`, `http://localhost/*` e `http://127.0.0.1/*`: **acrescente o domínio do
   seu painel** nos dois arquivos.
3. Clique em **Autorizar com passkey**. A extensão abre `web.whatsapp.com`, o dono aprova com biometria
   ou PIN e a assinatura volta ao painel.
4. O painel envia a assinatura em `POST /channels/CHANNEL_ID/passkey/response` (o corpo é o JSON da
   assinatura, sem alterações, até 64 KB). A resposta é `{"status": "accepted"}` e o canal passa a
   `connected` em seguida.

O resultado é um dispositivo novo vinculado; a extensão não lê nem copia a sessão do WhatsApp do dono.
Sem a extensão, a tela mostra "Extensão Linktor Passkey Connector não detectada".

### WhatsApp via Bridge {#bridge}

**WhatsApp via Bridge (não oficial)** é outro tipo de canal (`whatsapp_remote`): a sessão do WhatsApp
fica no **computador do atendente**, no aplicativo Linktor Bridge, que se liga ao servidor por uma
conexão WebSocket de saída (`/api/v1/gateways/ws`). Nada precisa ser aberto no firewall do atendente.

1. Crie o canal em **Canais** → **WhatsApp via Bridge (não oficial)** e salve.
2. Clique em **Gerar token**. O token aparece **uma única vez**; gerar outro invalida o anterior.
   Pela API: `POST /channels/CHANNEL_ID/bridge-token`, que exige usuário com papel `admin` ou `owner`
   (uma chave de API não alcança) e devolve `data.bridge_token`.
3. No computador do atendente, instale o Linktor Bridge, abra o painel local
   (`http://127.0.0.1:7788`) e informe **Servidor**, **ID do canal** e **Token do bridge** — ou rode o
   comando que a tela monta:
   `linktor-bridge --server <endereço do gateway> --channel-id CHANNEL_ID --token <token>`.
4. No painel do bridge, conecte o WhatsApp pelo QR code.

O estado aparece no cartão **Bridge** do canal (**Online**, **Sem sinal** ou **Offline**, e o número
pareado) e em `GET /channels/CHANNEL_ID/bridge-health`. O aplicativo Linktor Bridge é distribuído à
parte; esta página cobre só o lado do servidor.

## Webhook

Este canal **não tem webhook de provedor**: as mensagens chegam pela sessão aberta entre o servidor e o
WhatsApp, então não há URL para cadastrar em lugar nenhum.

O que existe é o **Webhook de saída**, do Linktor para a sua aplicação, configurado no próprio canal:

| Campo | Chave | Efeito |
|---|---|---|
| **URL do Consumidor** | `webhook_url` | Endereço HTTPS que recebe os eventos |
| **Segredo de Assinatura** | `credentials.webhook_secret` | Chave do HMAC que assina cada evento |
| **Eventos a entregar** | `config.webhook_events` | Tipos de evento, separados por vírgula |

Cada entrega leva `X-Linktor-Signature` (HMAC-SHA256 em hexadecimal de `timestamp + "." + corpo`) e
`X-Linktor-Timestamp`. Sem `webhook_secret` o Linktor descarta a entrega em vez de enviá-la sem
assinatura. Verificação e reentrega em [Receber eventos](../api/receber-eventos.md).

## Criar o canal pela API

`POST /api/v1/channels` com o scope `channels:write`, depois `POST /channels/CHANNEL_ID/connect` para
obter o QR code.

<Tabs groupId="linguagem">
<TabItem value="curl" label="cURL">

```bash
curl -X POST "$LINKTOR_URL/api/v1/channels" \
  -H "X-API-Key: $LINKTOR_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "type": "whatsapp",
    "name": "WhatsApp Vendas",
    "config": {
      "device_name": "Linktor",
      "ignore_groups": "false",
      "ignore_status": "true",
      "webhook_events": "message.received,message.delivered,message.read,message.failed"
    },
    "credentials": { "webhook_secret": "um-segredo-longo" },
    "webhook_url": "https://app.suaempresa.com/webhooks/linktor"
  }'

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
  type: 'whatsapp',
  name: 'WhatsApp Vendas',
  config: { device_name: 'Linktor', ignore_groups: 'false', ignore_status: 'true' },
  credentials: { webhook_secret: 'um-segredo-longo' },
  webhook_url: 'https://app.suaempresa.com/webhooks/linktor',
});

const res = await client.channels.connect(canal.id);
if (res.passkey_required) {
  console.log('Conta com passkey: conclua pelo painel');
} else if (res.qr_code) {
  console.log('Gere um QR com este conteúdo:', res.qr_code, `(expira em ${res.expires_in}s)`);
}
```

</TabItem>
<TabItem value="python" label="Python">

```python
import os
import requests

base = os.environ["LINKTOR_URL"].rstrip("/") + "/api/v1"
headers = {"X-API-Key": os.environ["LINKTOR_API_KEY"]}

r = requests.post(f"{base}/channels", headers=headers, timeout=30, json={
    "type": "whatsapp",
    "name": "WhatsApp Vendas",
    "config": {"device_name": "Linktor", "ignore_groups": "false", "ignore_status": "true"},
    "credentials": {"webhook_secret": "um-segredo-longo"},
    "webhook_url": "https://app.suaempresa.com/webhooks/linktor",
})
r.raise_for_status()
canal = r.json()["data"]

# O connect espera o QR do WhatsApp: use um timeout folgado
r = requests.post(f"{base}/channels/{canal['id']}/connect", headers=headers, timeout=60)
r.raise_for_status()
res = r.json()["data"]
if res.get("passkey_required"):
    print("Conta com passkey: conclua pelo painel")
elif res.get("qr_code"):
    print("Gere um QR com este conteúdo:", res["qr_code"], f"(expira em {res['expires_in']}s)")
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
		Type: types.ChannelTypeWhatsApp, // "whatsapp"
		Name: "WhatsApp Vendas",
		Config: map[string]string{
			"device_name":   "Linktor",
			"ignore_groups": "false",
			"ignore_status": "true",
		},
		Credentials: map[string]string{"webhook_secret": "um-segredo-longo"},
		WebhookURL:  "https://app.suaempresa.com/webhooks/linktor",
	})
	if err != nil {
		log.Fatal(err)
	}

	res, err := client.Channels.Connect(ctx, canal.ID)
	if err != nil {
		log.Fatal(err)
	}
	switch {
	case res.PasskeyRequired:
		fmt.Println("Conta com passkey: conclua pelo painel")
	case res.QRCode != "":
		fmt.Printf("Gere um QR com este conteúdo: %s (expira em %ds)\n", res.QRCode, res.ExpiresIn)
	}
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
                .type(Channel.ChannelType.WHATSAPP)
                .name("WhatsApp Vendas")
                .config(Map.of(
                        "device_name", "Linktor",
                        "ignore_groups", "false",
                        "ignore_status", "true"))
                .credentials(Map.of("webhook_secret", "um-segredo-longo"))
                .webhookUrl("https://app.suaempresa.com/webhooks/linktor")
                .build());

// Devolve o ConnectResult com o conteúdo do QR e a validade em segundos
var res = client.channels.connect(canal.getId());
```

</TabItem>
<TabItem value="csharp" label="C#">

```csharp
using System.Net.Http.Json;
using System.Text.Json;

var http = new HttpClient
{
    BaseAddress = new Uri(Environment.GetEnvironmentVariable("LINKTOR_URL")!.TrimEnd('/') + "/api/v1/"),
    Timeout = TimeSpan.FromSeconds(60),
};
http.DefaultRequestHeaders.Add("X-API-Key", Environment.GetEnvironmentVariable("LINKTOR_API_KEY"));

var criado = await http.PostAsJsonAsync("channels", new
{
    type = "whatsapp",
    name = "WhatsApp Vendas",
    config = new Dictionary<string, string>
    {
        ["device_name"] = "Linktor",
        ["ignore_groups"] = "false",
        ["ignore_status"] = "true",
    },
    credentials = new Dictionary<string, string> { ["webhook_secret"] = "um-segredo-longo" },
    webhook_url = "https://app.suaempresa.com/webhooks/linktor",
});
criado.EnsureSuccessStatusCode();
var id = (await criado.Content.ReadFromJsonAsync<JsonElement>())
    .GetProperty("data").GetProperty("id").GetString();

var conexao = await http.PostAsync($"channels/{id}/connect", null);
conexao.EnsureSuccessStatusCode();
var res = (await conexao.Content.ReadFromJsonAsync<JsonElement>()).GetProperty("data");

if (res.TryGetProperty("qr_code", out var qr))
{
    Console.WriteLine($"Gere um QR com este conteúdo: {qr} (expira em {res.GetProperty("expires_in")}s)");
}
```

</TabItem>
<TabItem value="php" label="PHP">

```php
<?php
require __DIR__ . '/vendor/autoload.php';

$http = new GuzzleHttp\Client([
    'base_uri' => rtrim(getenv('LINKTOR_URL'), '/') . '/api/v1/',
    'headers'  => ['X-API-Key' => getenv('LINKTOR_API_KEY')],
    'timeout'  => 60,
]);

$resposta = $http->post('channels', ['json' => [
    'type' => 'whatsapp',
    'name' => 'WhatsApp Vendas',
    'config' => [
        'device_name'   => 'Linktor',
        'ignore_groups' => 'false',
        'ignore_status' => 'true',
    ],
    'credentials' => ['webhook_secret' => 'um-segredo-longo'],
    'webhook_url' => 'https://app.suaempresa.com/webhooks/linktor',
]]);
$canal = json_decode((string) $resposta->getBody(), true)['data'];

$res = json_decode((string) $http->post("channels/{$canal['id']}/connect")->getBody(), true)['data'];
if (!empty($res['qr_code'])) {
    echo "Gere um QR com este conteúdo: {$res['qr_code']} (expira em {$res['expires_in']}s)", PHP_EOL;
}
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
        ("device_name".to_string(), "Linktor".to_string()),
        ("ignore_groups".to_string(), "false".to_string()),
        ("ignore_status".to_string(), "true".to_string()),
    ]);
    let credentials =
        HashMap::from([("webhook_secret".to_string(), "um-segredo-longo".to_string())]);

    let canal = client
        .channels()
        .create(
            CreateChannelInput::new("WhatsApp Vendas", ChannelType::Whatsapp)
                .config(config)
                .credentials(credentials)
                .webhook_url("https://app.suaempresa.com/webhooks/linktor"),
        )
        .await?;

    let res = client.channels().connect(&canal.id).await?;
    println!("{:?}", res); // ConnectResult: qr_code, expires_in, passkey_required...
    Ok(())
}
```

</TabItem>
</Tabs>

:::warning Ao atualizar, reenvie o `config` inteiro
`PUT /channels/CHANNEL_ID` **substitui** `config` por completo; só `credentials` é mesclado. Mandar
apenas `{"always_online": "true"}` apaga `device_name`, `ignore_groups`, `webhook_events` e as demais
chaves. Leia o canal, altere o que precisa e envie tudo de volta.
:::

Mais sobre as rotas de canal em [API de canais](../api/canais.md).

## Enviar e receber

### Enviar uma mensagem

`POST /api/v1/messages/send` (scope `messages:send`). O canal precisa estar habilitado e `connected`.

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
    "text": "Olá! Seu pedido saiu para entrega."
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
	if resp.StatusCode != http.StatusAccepted {
		log.Fatalf("envio recusado: %s", resp.Status)
	}

	var saida struct {
		Data struct {
			ID     string `json:"id"`
			Status string `json:"status"`
		} `json:"data"`
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
        "text", "Olá! Seu pedido saiu para entrega."));

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
            "text": "Olá! Seu pedido saiu para entrega."
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

A resposta `202` traz `data.id`, `data.conversation_id`, `data.channel_id` e `data.status: "queued"`.
Mídia (`image`, `video`, `audio`, `document`, `sticker` com `attachments`), idempotência e correlação
em [Enviar mensagens](../api/enviar-mensagens.md).

### Enviar para um grupo {#grupos}

`POST /api/v1/channels/CHANNEL_ID/groups/GROUP_ID/messages`, com o scope `messages:send` (não
`channels:write`). `GROUP_ID` é o identificador do grupo no WhatsApp, no formato
`120363012345678901@g.us` — o mesmo valor que chega em `data.group.id` nos eventos recebidos.

```bash
curl -X POST "$LINKTOR_URL/api/v1/channels/CHANNEL_ID/groups/120363012345678901@g.us/messages" \
  -H "X-API-Key: $LINKTOR_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{ "text": "Bom dia, pessoal. O relatório de ontem já está disponível." }'
```

```json
{
  "success": true,
  "data": {
    "id": "9a0d7c11-6f1b-4f6f-8c0e-1c2b3d4e5f60",
    "recipient": "120363012345678901@g.us",
    "status": "queued"
  }
}
```

- Só texto (`text`, obrigatório) e `metadata` opcional.
- O número conectado precisa ser participante do grupo.
- A mensagem vai direto para a fila de envio do canal: ela não cria conversa nem contato no Linktor.
- Não há rotas para listar, criar ou administrar grupos. Descubra o id do grupo a partir de uma
  mensagem recebida dele.

Para receber mensagens de grupo, deixe **Receber mensagens de grupo** ligado. Desligado, elas são
descartadas no canal e não geram evento.

### Botões e lista {#botoes}

Bots e fluxos enviam opções como mensagem interativa. Neste canal:

- **Até 3 opções** saem como **botões nativos** de resposta rápida. O toque volta como
  `message.received` com `contentType: "interactive"` e o título da opção em `content.text`.
- **4 opções ou mais**, e qualquer **lista**, saem como **texto numerado**:

```text
Como podemos ajudar?
[ 1 ] Vendas
[ 2 ] Suporte
[ 3 ] Financeiro
[ 4 ] Falar com atendente
```

- Se o envio dos botões nativos falhar, o Linktor reenvia a mesma mensagem como texto numerado.

A lista vira texto de propósito: o aplicativo do WhatsApp aceita a lista enviada por um dispositivo
vinculado, marca como entregue e **não mostra nada** ao cliente. Como não há erro para capturar, o
texto numerado é o único formato confiável. Quem responde digita o número ou o texto da opção; o
tratamento da resposta fica a cargo do seu bot ou fluxo.

### O que sua aplicação recebe

Mensagem recebida de um contato:

```json
{
  "id": "evt_4b2c9e10-received",
  "type": "message.received",
  "timestamp": "2026-11-24T14:03:11Z",
  "tenantId": "3f6f0c1e-9a51-4f0e-b0a5-6b6f6d2c1a11",
  "environment": "production",
  "data": {
    "message": {
      "id": "4b2c9e10-7d3a-4c55-8e21-5f6a7b8c9d01",
      "direction": "inbound",
      "contentType": "text",
      "content": { "text": "Oi, quero saber do meu pedido" },
      "senderId": "5511999999999",
      "senderType": "contact",
      "metadata": {
        "senderName": "Maria Souza",
        "externalId": "3EB0C0A5F2B1D4E6A7C8"
      }
    },
    "conversationId": "b3a9d2c4-55e1-4f0a-8a44-0f6f1f1f7a21",
    "contactId": "0c2b7f7e-3f1d-4d57-a0ad-0f9e5f1a2b33",
    "channelId": "CHANNEL_ID",
    "channelType": "whatsapp"
  }
}
```

Numa mensagem de **grupo**, `data` ganha o bloco `group` e `senderId` continua sendo quem falou:

```json
{
  "message": {
    "id": "c7e1a2b3-0f4d-4a6b-9c8d-1e2f3a4b5c6d",
    "direction": "inbound",
    "contentType": "text",
    "content": { "text": "@Linktor alguém pode ver o pedido 1042?" },
    "senderId": "5511988887777",
    "senderType": "contact",
    "mentions": ["5511999999999@s.whatsapp.net"]
  },
  "group": { "id": "120363012345678901@g.us" },
  "channelId": "CHANNEL_ID",
  "channelType": "whatsapp"
}
```

Três comportamentos próprios deste canal:

- **Mensagens digitadas no celular.** O que o operador escreve direto no aparelho pareado também chega
  como `message.received`, mas com `direction: "outbound"` e `senderType` diferente de `contact`.
  Filtre por `direction` para não tratar a fala do operador como se fosse do cliente.
- **Contato sem telefone.** O WhatsApp pode identificar um contato só por um identificador interno
  (LID). Quando o Linktor consegue resolver o número, `senderId` é o telefone; quando não consegue, é o
  identificador.
- **Chamadas.** Uma chamada recebida entra como `message.received` com `contentType: "call"` e depois
  gera `call.answered`, `call.missed` ou `call.rejected`.

Os status de envio chegam como `message.sent`, `message.delivered`, `message.read` e `message.failed`,
sem bloco `pricing` (ele só existe no canal oficial). Formato completo em
[Receber eventos](../api/receber-eventos.md).

### Contatos do aparelho e avatar {#contatos}

**Agenda do aparelho** — `GET /api/v1/channels/CHANNEL_ID/contacts` (scope `channels:read`):

```json
{
  "success": true,
  "data": {
    "contacts": [
      {
        "jid": "5511999999999@s.whatsapp.net",
        "phone": "5511999999999",
        "fullName": "Maria Souza",
        "pushName": "Maria",
        "businessName": ""
      }
    ],
    "total": 1,
    "withoutPhone": 0
  }
}
```

- Exige sessão ativa; sem ela a resposta é `409` (`no live WhatsApp session for this channel`).
- Contato identificado só por LID vem com `phone` **vazio**, não omitido da lista: `withoutPhone` diz
  quantos são. "Sem telefone" é diferente de "não encontrado".

**Foto de perfil** — `GET /api/v1/channels/CHANNEL_ID/contacts/CONTATO/avatar`, com os scopes
`channels:read` **e** `contacts:read`. `CONTATO` é o telefone só com dígitos (`5511999999999`) ou o
identificador completo (`5511999999999@s.whatsapp.net`).

- Sucesso: `200` com os bytes da imagem, e os headers `X-Picture-Id` e `ETag`.
- Envie `If-None-Match` com o `ETag` anterior para receber `304` quando a foto não mudou.
- `404` com `reason: "hidden"` (a privacidade do contato esconde a foto) ou `reason: "not_set"` (não há
  foto).
- As consultas têm ritmo limitado por canal (`LINKTOR_CONTACT_AVATAR_INTERVAL`); ao exceder, a resposta
  traz `Retry-After`.

Funciona também em canais **WhatsApp via Bridge**; o canal oficial não expõe a foto do cliente.

## Limites e particularidades

### "Conecta e desconecta": versão do cliente recusada (405) {#erro-405}

O WhatsApp só aceita clientes que anunciam uma versão recente do WhatsApp Web. A versão anunciada vem
embutida na biblioteca; quando o WhatsApp deixa de aceitá-la, **todos** os canais não oficiais passam a
falhar ao mesmo tempo com o erro `405` (`Client outdated`). O sintoma no painel é o canal alternando
entre conectado e desconectado, sem pedir novo QR code.

O Linktor trata isso sozinho na maior parte dos casos:

- consulta a versão atual em `web.whatsapp.com` a cada 6 horas e antes de conectar, e passa a usar a
  mais nova;
- registra cada recusa e força uma nova consulta (no máximo uma a cada 2 minutos);
- mostra o estado em **Observabilidade**, no cartão **Versão do cliente WhatsApp (canais
  não-oficiais)**.

| Estado no cartão | `status` | Significado |
|---|---|---|
| **Em dia** | `ok` | A última conexão autenticou |
| **Desatualizada** | `behind` | Existe versão mais nova e a atualização automática está desligada |
| **Recusada pelo WhatsApp** | `outdated` | O WhatsApp recusou a versão depois da última conexão boa |
| **Não verificada** | `unknown` | Ainda não houve consulta nem conexão |

O que fazer quando aparece **O WhatsApp está recusando a versão do cliente (erro 405)**:

1. Clique em **Verificar agora**. O Linktor busca a versão nova e os canais reconectam sozinhos.
2. **Não pareie de novo.** A sessão continua válida; ler outro QR code não resolve.
3. Se o alerta continuar, a biblioteca precisa ser atualizada: atualize o Linktor
   ([Backup e atualização](../implantar/backup-e-atualizacao.md)).

Pela API (papel `admin` ou `owner`): `GET /api/v1/observability/whatsapp-version` e
`POST /api/v1/observability/whatsapp-version/check`. A resposta traz `status`, `builtin`, `current`,
`latest`, `auto_update`, `last_check_at`, `last_check_error`, `last_outdated_at`, `outdated_count` e
`last_login_at`.

A variável `WHATSAPP_VERSION_AUTO_UPDATE=false` desliga a troca automática (a consulta e o alerta
continuam). Nos logs do servidor, procure por `Client outdated`.

### Outros limites

- **Sessão no servidor.** Um canal equivale a um arquivo em `storages/`. Faça backup do diretório e
  mantenha-o em volume persistente.
- **Reconexão automática.** Na subida do servidor, todos os canais não oficiais com sessão guardada
  reconectam sozinhos, um de cada vez e com prazo próprio.
- **Tamanhos.** O canal declara até 65.536 caracteres por mensagem e mídia de até 100 MB; na prática
  valem os limites do aplicativo do WhatsApp.
- **Sem modelos e sem janela de 24h.** Você pode iniciar conversa com qualquer número — o que é
  exatamente o comportamento que leva a banimento quando usado para disparo em massa.
- **Proxy.** `proxy_host` e `proxy_port` roteiam a conexão do canal por um proxy SOCKS5.
- **Chamadas.** Com **Rejeitar chamadas recebidas** ligado, a chamada é recusada e a **Mensagem ao
  rejeitar a chamada** é enviada na conversa.

### Riscos e quando preferir o oficial {#riscos}

O canal não oficial automatiza um cliente que o WhatsApp não autoriza. Consequências práticas:

- **Banimento.** O WhatsApp pode bloquear o número, temporária ou definitivamente, sem aviso e sem
  recurso garantido. O risco cresce com número novo, volume alto, mensagens para quem nunca falou com
  você, conteúdo repetido e denúncias de destinatários.
- **Instabilidade.** Mudanças do lado do WhatsApp quebram o canal sem aviso (o [erro 405](#erro-405) é
  o caso mais comum).
- **Sem suporte e sem garantias** da Meta: não há SLA, selo de verificação nem métricas oficiais.

Reduza o risco usando um número já em uso há algum tempo, respondendo mais do que iniciando, evitando
disparos e mantendo o celular principal ativo.

Prefira o [WhatsApp oficial](./whatsapp-oficial.md) quando:

- o número é crítico para o negócio e não pode ficar fora do ar;
- você envia notificações ou campanhas em volume;
- precisa de modelos aprovados, métricas da Meta, custo previsível ou conformidade contratual;
- o cliente final exige uso da API oficial.

O não oficial faz sentido para atendimento receptivo de baixo volume, grupos (que a Cloud API não
cobre) e ambientes de teste.

## Solução de problemas

| Sintoma | Causa | O que fazer |
|---|---|---|
| Canal alterna entre conectado e desconectado, sem pedir QR | Versão do cliente recusada (`405`, `Client outdated`) | **Observabilidade** → **Verificar agora**; se persistir, atualize o Linktor. Não pareie de novo |
| `connect` responde `timeout waiting for QR code` | O servidor não alcança o WhatsApp (rede, proxy, DNS) | Verifique a saída de rede e a configuração de proxy do canal |
| QR code expira antes da leitura | Validade curta do código | Clique em **Atualizar QR** ou chame `connect` de novo |
| Tela mostra **Esta conta exige uma passkey** | Conta protegida por passkey | Instale a extensão e clique em **Autorizar com passkey** |
| "Extensão Linktor Passkey Connector não detectada" | Extensão ausente ou domínio do painel fora das origens permitidas | Instale a extensão e acrescente o domínio em `manifest.json` e `background.js` |
| "Login por código de pareamento ainda não está disponível" | Limitação da tela | Use o QR code, ou `POST /channels/CHANNEL_ID/pair` pela API |
| Canal volta a `disconnected` depois de dias | Aparelho removido em **Dispositivos conectados** ou celular principal inativo por cerca de 14 dias | Conecte de novo com QR code |
| Todos os canais pedem QR depois de um deploy | Diretório `storages/` não persistido | Monte `storages/` em volume persistente |
| `POST /messages/send` responde `channel is not connected` | Sessão caída | Chame `connect`; se a sessão existir, ele reconecta sem QR |
| Mensagem fica em fila e é reenviada | Não há sessão ativa no momento | Confira o estado do canal e os logs em [Observabilidade](../painel/observabilidade.md) |
| Lista chega como texto numerado | Comportamento esperado | Ver [Botões e lista](#botoes) |
| Mensagens de grupo não chegam | **Receber mensagens de grupo** desligado | Ligue a opção e salve |
| `GET .../contacts` responde 409 | Canal sem sessão ativa | Conecte o canal antes |
| Número parou de enviar e o aplicativo mostra conta banida | Banimento pelo WhatsApp | Só o WhatsApp reverte; avalie migrar para o [canal oficial](./whatsapp-oficial.md) |
