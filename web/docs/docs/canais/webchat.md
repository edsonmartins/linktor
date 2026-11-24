---
sidebar_position: 7
title: WebChat
description: Instale o widget de chat do Linktor no seu site — snippet para HTML, React, Vue e Angular, opções do widget, origens permitidas, token do widget, upload de mídia e API.
---

# WebChat

O WebChat é um **widget de chat** que você instala no seu site. O visitante abre o balão, escreve, e
a conversa aparece no painel do Linktor como em qualquer outro canal. Não há provedor externo: o
widget conversa direto com o servidor do Linktor por WebSocket.

Valor de `type` na API: **`webchat`**.

| Recurso | Visitante → Linktor | Linktor → visitante |
|---|---|---|
| Texto | Sim | Sim |
| Imagem (JPEG, PNG, GIF, WebP) | Sim, com **Permitir anexos** ligado | Sim (exibida na bolha) |
| Áudio (arquivo ou gravado no navegador) | Sim, com **Permitir anexos** ligado | Sim (player na bolha) |
| Documento e vídeo | Não | Aparecem como link para o arquivo |
| Botões, modelos, localização | Não | Não |
| Grupos | Não se aplica | Não se aplica |
| Confirmação de leitura | Não | Não |

## Antes de começar

- O servidor do Linktor publicado em **HTTPS** (o navegador exige `wss://` em páginas HTTPS).
- Acesso ao HTML (ou ao código) do site onde o widget será instalado.
- O domínio do site liberado no servidor do Linktor — veja [Origens permitidas](#origens). Este é o
  passo mais esquecido: sem ele o widget aparece, mas fica em **Offline**.
- Para aceitar anexos: o armazenamento de mídia do servidor configurado (veja
  [Configuração](../implantar/configuracao.md)).

## Configurar na origem

Aqui "origem" é o **seu site**: instalar é colar o snippet (ou montar o componente) e liberar o
domínio no servidor.

:::tip Comece pelo canal
O snippet leva o id do canal. Se você ainda não criou o canal, faça primeiro a seção
[Configurar no Linktor](#configurar-no-linktor) e volte aqui com o `CHANNEL_ID` em mãos.
:::

### HTML puro

Cole o snippet da aba **Incorporar** logo antes de `</body>`. Ele é assim:

```html
<script>
  (function(l,i,n,k,t,o,r){l['LinktorObject']=t;l[t]=l[t]||function(){
    (l[t].q=l[t].q||[]).push(arguments)};o=i.createElement(n);
    r=i.getElementsByTagName(n)[0];o.async=1;o.src=k;r.parentNode.insertBefore(o,r);
  })(window,document,'script','https://linktor.suaempresa.com/widget/v1/linktor.js','linktor');
  linktor('init', {
    channelId: 'CHANNEL_ID',
    baseUrl: 'https://linktor.suaempresa.com',
    position: 'right',
    title: 'Fale com a gente',
    primaryColor: '#6366f1'
  });
</script>
```

- `baseUrl` é o endereço do servidor **sem** `/api/v1`: o widget acrescenta `/api/v1/...` e `/ws/...`
  sozinho.
- O carregador é servido pelo próprio Linktor em `/widget/v1/linktor.js`.
- Os comandos chamados antes de o script terminar de carregar ficam numa fila e são executados em
  ordem.

Comandos disponíveis depois do `init`:

```js
linktor('open');                               // abre a janela
linktor('close');
linktor('toggle');
linktor('sendMessage', 'Olá, preciso de ajuda'); // envia em nome do visitante
linktor('sendFile', { file: blob, filename: 'foto.jpg' });
linktor('updateUser', { name: 'Maria Souza', email: 'cliente@exemplo.com' });
linktor('reset');                              // descarta a sessão e começa outra
linktor('destroy');                            // remove o widget da página
```

### React, Vue e Angular

O widget também é publicado no npm, pelo repositório
[linktor-webchat-sdk](https://github.com/edsonmartins/linktor-webchat-sdk), como um núcleo sem
framework (`@linktor/webchat-core`) e três invólucros. Os componentes não desenham nada no lugar
onde são colocados: o widget anexa a própria interface flutuante ao `body`.

<Tabs groupId="webchat-framework">
<TabItem value="react" label="React">

```tsx
// npm install @linktor/react-webchat
import { LinktorChat } from '@linktor/react-webchat';

export function App() {
  return (
    <LinktorChat
      channelId="CHANNEL_ID"
      baseUrl="https://linktor.suaempresa.com"
      title="Fale com a gente"
      visitor={{ name: 'Maria Souza', email: 'cliente@exemplo.com' }}
      onMessage={(m) => console.log(m)}
    />
  );
}
```

</TabItem>
<TabItem value="vue" label="Vue 3">

```html
<!-- npm install @linktor/vue-webchat -->
<script setup lang="ts">
import { LinktorChat } from '@linktor/vue-webchat';
</script>

<template>
  <LinktorChat
    channel-id="CHANNEL_ID"
    base-url="https://linktor.suaempresa.com"
    title="Fale com a gente"
    @message="(m) => console.log(m)"
  />
</template>
```

</TabItem>
<TabItem value="angular" label="Angular">

```typescript
// npm install @linktor/angular-webchat
import { Component } from '@angular/core';
import { LinktorChatComponent } from '@linktor/angular-webchat';

@Component({
  selector: 'app-root',
  standalone: true,
  imports: [LinktorChatComponent],
  template: `
    <linktor-chat
      [channelId]="'CHANNEL_ID'"
      [baseUrl]="'https://linktor.suaempresa.com'"
      (message)="onMessage($event)"
    ></linktor-chat>
  `,
})
export class AppComponent {
  onMessage(message: unknown) {
    console.log(message);
  }
}
```

</TabItem>
<TabItem value="core" label="JavaScript (núcleo)">

```typescript
// npm install @linktor/webchat-core
import { LinktorWebChat } from '@linktor/webchat-core';

const chat = new LinktorWebChat({
  channelId: 'CHANNEL_ID',
  baseUrl: 'https://linktor.suaempresa.com',
}).mount();

chat.open();
chat.sendMessage('Olá!');
```

</TabItem>
</Tabs>

:::note Versões do pacote npm
As opções desta página foram conferidas no carregador que o servidor distribui
(`/widget/v1/linktor.js`). Os pacotes npm são versionados à parte: recursos mais recentes — em
especial anexos e gravação de áudio — podem não estar na versão publicada do seu framework. Em
módulos Angular (não standalone), importe `LinktorWebChatModule`. No Angular os eventos de abrir e
fechar se chamam `opened` e `closed`.
:::

### Opções do widget

Valem para o `init` do snippet e, com os mesmos nomes, como propriedades dos componentes.

| Opção | Tipo | Padrão | Descrição |
|---|---|---|---|
| `channelId` | string | — (obrigatória) | Id do canal WebChat. |
| `baseUrl` | string | — (obrigatória) | Endereço do servidor Linktor, sem caminho. |
| `position` | `'left'` ou `'right'` | `'right'` | Lado do botão flutuante. |
| `title` | string | título do canal | Título do cabeçalho; sobrepõe o `widget_title` do servidor. |
| `primaryColor` | string | cor do canal | Cor de destaque; sobrepõe o `widget_color` do servidor. |
| `autoOpen` | boolean | `false` | Abre a janela ao carregar. |
| `showLauncher` | boolean | `true` | Mostra o botão flutuante. Com `false`, abra pelo comando `open`. |
| `zIndex` | number | `999999` | Empilhamento do widget na página. |
| `visitor` | objeto | — | `name`, `email` e `phone` do visitante — veja [Identificar o visitante](#visitante). |
| `token` | string | — | Token do widget, quando o canal exige. |
| `allowAttachments` | boolean | o do canal | Força mostrar ou esconder os botões de anexo. O servidor continua decidindo se aceita o upload. |
| `maxFileSizeBytes` | number | `10485760` | Limite conferido no navegador antes do upload. |
| `acceptedMimeTypes` | string[] | imagens e áudios aceitos | Tipos aceitos pelo seletor de arquivo. |
| `theme` | objeto | — | Tokens visuais (abaixo). |
| `labels` | objeto | textos em inglês | Textos do widget (abaixo). |
| `hidePoweredBy` | boolean | `false` | Remove o rodapé "Powered by Linktor". |
| `launcherIcon` | string | ícone padrão | HTML do ícone do botão (um SVG, por exemplo). |
| `customCss` | string | — | CSS injetado depois da folha de estilo do widget. |
| `renderMessage` | função | — | Recebe `(mensagem, contexto)` e devolve um elemento ou HTML para substituir a bolha. |
| `onReady`, `onOpen`, `onClose`, `onMessage` | função | — | Retornos de chamada. `onMessage` recebe as mensagens vindas do servidor. |

**`theme`** — cada chave vira uma variável CSS restrita ao widget: `primaryColor`, `fontFamily`,
`fontSize`, `width`, `height`, `borderRadius`, `background`, `launcherSize`, `launcherIconColor`,
`headerBg`, `headerColor`, `agentBubbleBg`, `agentBubbleColor`, `visitorBubbleBg`,
`visitorBubbleColor`.

**`labels`** — os textos vêm em inglês; traduza no snippet:

```js
linktor('init', {
  channelId: 'CHANNEL_ID',
  baseUrl: 'https://linktor.suaempresa.com',
  labels: {
    title: 'Fale com a gente',
    online: 'Online',
    offline: 'Offline',
    connecting: 'Conectando…',
    inputPlaceholder: 'Digite uma mensagem…',
    sendAriaLabel: 'Enviar',
    launcherAriaLabel: 'Abrir chat',
    closeAriaLabel: 'Fechar chat',
    attachAriaLabel: 'Anexar imagem ou áudio',
    recordAriaLabel: 'Gravar áudio',
    recording: 'Gravando…',
    uploading: 'Enviando…',
    uploadFailed: 'Falha no envio',
    fileTooLarge: 'Arquivo muito grande (máx. {max})',
    unsupportedFile: 'Tipo não aceito: só imagens e áudios',
    micDenied: 'Acesso ao microfone negado'
  },
  theme: { borderRadius: '16px', headerBg: '#111827' },
  customCss: '.linktor-launcher { box-shadow: 0 0 0 4px rgba(99,102,241,.25); }'
});
```

As classes `.linktor-widget`, `.linktor-launcher`, `.linktor-chat`, `.linktor-header`,
`.linktor-messages`, `.linktor-message` (com `.contact`, `.user` ou `.system`), `.linktor-input` e
`.linktor-send` são estáveis e podem ser estilizadas pelo `customCss`.

### Identificar o visitante {#visitante}

- Cada aba do navegador recebe um **id de sessão** (guardado em `sessionStorage`, chave
  `linktor_session_CHANNEL_ID`). É esse id que identifica o contato no canal e que aparece como
  `senderId` nos eventos. Fechar a aba ou abrir outra cria uma sessão nova.
- Passe `visitor` no `init` quando souber quem é o usuário (por exemplo, depois do login). `name`
  vira o nome do contato; sem ele o contato se chama `Visitor`. Todo contato do WebChat recebe a
  etiqueta `webchat`.
- Se `visitor.email` coincidir com o e-mail de um contato existente, a sessão é vinculada a esse
  contato em vez de criar outro.
- `linktor('updateUser', ...)` só vale para a **próxima** conexão. Se o chat já está conectado, chame
  antes do `init` ou faça `reset`.

:::warning Os dados do visitante não são verificados
`name`, `email` e `phone` são enviados pelo navegador e aceitos como vieram. Qualquer pessoa pode
abrir o console e informar o e-mail de outro cliente. Trate-os como identificação declarada, não
como autenticação, e não exponha dados sensíveis com base neles.
:::

### Origens permitidas {#origens}

O servidor confere o header `Origin` do navegador ao abrir o WebSocket. A lista é **do servidor
inteiro**, definida por variáveis de ambiente — não por canal:

| Variável | Efeito |
|---|---|
| `LINKTOR_WS_ALLOWED_ORIGINS` | Origens aceitas, separadas por vírgula, com esquema e sem barra final: `https://www.suaempresa.com,https://loja.suaempresa.com`. A comparação é exata. |
| `LINKTOR_WS_ENFORCE_ORIGIN` | `true` aceita **somente** a lista acima. Sem ela, origens `localhost` e `127.0.0.1` também passam (para desenvolvimento). |
| `LINKTOR_WS_ALLOW_EMPTY_ORIGIN` | `true` aceita conexões sem header `Origin` (clientes que não são navegador). |
| `LINKTOR_CORS_ALLOWED_ORIGINS` | Origens liberadas para as chamadas HTTP do widget (`/config` e `/upload`). Inclua o mesmo domínio do site. |

Ou seja: um site em `https://www.suaempresa.com` **não conecta** enquanto esse endereço não estiver
em `LINKTOR_WS_ALLOWED_ORIGINS`. O campo **Domínios Permitidos** da tela não substitui essa
configuração.

### Token do widget {#token-do-widget}

Por padrão o WebSocket do canal é público: quem souber o `channelId` conecta. Para exigir um token
assinado:

1. Grave um segredo em `config.widget_secret` (ou `credentials.widget_secret`) pela API.
2. No **seu backend**, peça um token ao Linktor para cada carregamento de página:

   ```bash
   curl -X POST "$LINKTOR_URL/api/v1/webchat/$CHANNEL_ID/widget-token" \
     -H "X-API-Key: $LINKTOR_API_KEY"
   ```

   ```json
   { "token": "Q0hBTk5FTF9JRC4xNzk1NTI4NjA3.kq3...", "expires_in": 1800 }
   ```

3. Entregue o token à página e passe-o no `init`: `linktor('init', { ..., token: 'TOKEN' })`.

O token vale **30 minutos**, é preso ao canal e autentica o **widget**, não o visitante — ele não
carrega nome nem e-mail. Com o segredo configurado, o WebSocket e o upload respondem
`401 invalid or missing widget token` sem um token válido. O widget não renova o token sozinho: numa
página que fica aberta por mais de 30 minutos, uma reconexão depois do vencimento falha até você
reinicializar o widget com um token novo. Nunca coloque a chave de API na página.

## Configurar no Linktor {#configurar-no-linktor}

1. No painel, abra **Canais** e, em **Adicionar Novo Canal**, clique em **Web Chat**.
2. Preencha **Nome do Canal** e as abas **Aparência** e **Comportamento** (tabela abaixo).
3. Clique em **Criar Canal**.
4. No cartão do canal, abra o menu de ações e clique em **Conectar**. Um canal WebChat desconectado
   ou desabilitado recusa a conexão do widget com `503 channel not active`.
5. Reabra o canal e copie o snippet da aba **Incorporar**.

| Campo na tela | Chave em `config` | Obrigatório | O que faz hoje |
|---|---|---|---|
| **Nome do Canal** | `name` (fora de `config`) | Sim | Nome interno do canal. |
| **Título do Widget** | `widget_title` | Não | Título do cabeçalho. Vai no snippet (`title`) e é servido ao widget. Padrão: `Chat with us`. |
| **Cor Primária** | `widget_color` | Não | Cor do botão e do cabeçalho. Vai no snippet (`primaryColor`) e é servida ao widget. Padrão do servidor: `#007bff`. |
| **Cor do Texto** | `text_color` | Não | Só a prévia da tela usa. Para mudar cores de texto no site, use `theme` no snippet. |
| **Posição do Widget** | `position` | Não | **Inferior Direito** ou **Inferior Esquerdo**. Vira `position: "right"` ou `"left"` no snippet. |
| **Mensagem de Boas-vindas** | `welcome_message` | Não | Gravada e servida em `/config`, mas o widget atual **não a exibe** (veja o aviso abaixo). |
| **Mensagem Offline** | `offline_message` | Não | Idem: gravada e servida, não exibida pelo widget atual. |
| **Placeholder do Campo** | `placeholder_text` | Não | Vira `labels.inputPlaceholder` no snippet. |
| **Abrir Widget Automaticamente** | `auto_open` | Não | Vira `autoOpen: true` no snippet. |
| **Atraso (segundos)** | `auto_open_delay` | Não | Gravado, mas o widget atual abre imediatamente — não há atraso. |
| **Mostrar Indicador de Digitação** | `show_typing_indicator` | Não | Gravado; o widget atual não consulta essa chave. |
| **Permitir anexos** | `allow_attachments` | Não | **Efetivo no servidor:** libera o upload de imagem e áudio e faz o widget mostrar os botões de anexo e microfone. |
| **Domínios Permitidos** | `allowed_domains` | Não | Gravado, mas **não é aplicado**. A restrição de origem é feita por variável de ambiente — veja [Origens permitidas](#origens). |

Chaves que não aparecem na tela e só podem ser definidas pela API:

| Chave | Onde | Efeito |
|---|---|---|
| `widget_secret` | `config` ou `credentials` | Liga a exigência de [token do widget](#token-do-widget). |
| `avatar_url`, `require_name`, `require_email` | `config` | Servidos em `/config`; o widget atual não os usa. |

:::warning Campos que a tela grava e o widget ignora
A tela de configuração tem mais campos do que o widget consome hoje. O que **muda o comportamento**
de fato: título, cor primária, posição, placeholder, abertura automática (todos pelo snippet) e
**Permitir anexos** (pelo servidor). Se você alterar posição, placeholder ou abertura automática
depois de instalar, **copie o snippet de novo** — essas opções vivem no snippet, não no servidor.
Título e cor são lidos do servidor a cada carregamento, desde que o snippet não os fixe.
:::

## Webhook

O WebChat não tem webhook de provedor. Estes são os endereços que o widget usa no servidor do
Linktor:

| Método e caminho | Autenticação | Uso |
|---|---|---|
| `GET /widget/v1/linktor.js` | Pública | Carregador do widget. |
| `GET /api/v1/webchat/CHANNEL_ID/config` | Pública | Título, cor, mensagens e `allow_attachments` do canal. |
| `GET /ws/CHANNEL_ID` (WebSocket) | Origem + token, se exigido | Conversa em tempo real. Parâmetros: `session_id`, `name`, `email`, `phone`, `token`. |
| `POST /api/v1/webchat/CHANNEL_ID/upload` | Token, se exigido | Upload de anexo (`multipart/form-data`, campo `file`). Token em `?token=` ou no header `X-Widget-Token`. |
| `POST /api/v1/webchat/CHANNEL_ID/widget-token` | `X-API-Key` ou sessão | Emite o token do widget. |

Para quem escreve um cliente próprio, cada quadro do WebSocket é um JSON `type` + `payload`:

```json
{ "type": "message", "payload": { "id": "b7c1…", "content_type": "text", "content": "Olá", "timestamp": "2026-11-24T14:03:27Z" } }
```

O servidor envia `connect` (com `metadata.session_id`) ao abrir, `ack` (com o `id`) a cada mensagem
aceita, `message` para o que o atendente ou o bot escreve (`sender_type` `user` ou `system`),
`typing` e `error`. Cada quadro pode ter até 512 KB.

:::note Widget legado
O arquivo `/widget/widget.js` é a versão antiga do widget, que lê `window.LinktorSettings`. Ele
continua sendo servido, mas não tem anexos nem as opções de personalização. O carregador novo também
inicializa sozinho quando encontra `window.LinktorSettings` com `channelId` — use sempre
`/widget/v1/linktor.js`.
:::

## Criar o canal pela API

<Tabs groupId="linguagem">
<TabItem value="curl" label="cURL">

```bash
# 1. Cria o canal (nasce "disconnected")
curl -X POST "$LINKTOR_URL/api/v1/channels" \
  -H "X-API-Key: $LINKTOR_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "type": "webchat",
    "name": "Chat do site",
    "config": {
      "widget_title": "Fale com a gente",
      "widget_color": "#6366f1",
      "allow_attachments": "true"
    },
    "credentials": {
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
  "type": "webchat",
  "name": "Chat do site",
  "config": {
    "widget_title": "Fale com a gente",
    "widget_color": "#6366f1",
    "allow_attachments": "true"
  },
  "credentials": {
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
    name="Chat do site",
    type="webchat",
    config={
        "widget_title": "Fale com a gente",
        "widget_color": "#6366f1",
        "allow_attachments": "true"
    },
    credentials={
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
		Name: "Chat do site",
		Type: types.ChannelType("webchat"),
		Config: map[string]string{
			"widget_title": "Fale com a gente",
			"widget_color": "#6366f1",
			"allow_attachments": "true",
		},
		Credentials: map[string]string{
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
        .type(Channel.ChannelType.WEBCHAT)
        .name("Chat do site")
        .config(Map.of(
            "widget_title", "Fale com a gente",
            "widget_color", "#6366f1",
            "allow_attachments", "true"))
        .credentials(Map.of(
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
  "type": "webchat",
  "name": "Chat do site",
  "config": {
    "widget_title": "Fale com a gente",
    "widget_color": "#6366f1",
    "allow_attachments": "true"
  },
  "credentials": {
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
    'type' => 'webchat',
    'name' => 'Chat do site',
    'config' => [
        'widget_title' => 'Fale com a gente',
        'widget_color' => '#6366f1',
        'allow_attachments' => 'true',
    ],
    'credentials' => [
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

    let input = CreateChannelInput::new("Chat do site", ChannelType::Webchat)
        .config(to_map(&[
            ("widget_title", "Fale com a gente"),
            ("widget_color", "#6366f1"),
            ("allow_attachments", "true"),
        ]))
        .credentials(to_map(&[
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

Depois do `connect`, monte o snippet com o `id` devolvido. `webhook_url` e
`credentials.webhook_secret` são o endereço e o segredo da **sua aplicação** para receber os eventos;
a tela do WebChat não tem campos para eles.

:::warning PUT substitui o config
No `PUT /channels/CHANNEL_ID`, o `config` enviado substitui o anterior por inteiro (só `credentials`
é mesclado). Para ligar `widget_secret` depois, reenvie todas as chaves de `config` junto — ou grave
o segredo em `credentials.widget_secret`, que é mesclado.
:::

## Enviar e receber

### Enviar

`to` é o **id da sessão** do visitante — o `senderId` recebido em `message.received`. A entrega
acontece pelo WebSocket aberto: se o visitante fechou a página, não há para onde enviar e a mensagem
falha (`client not connected`). Quando a sessão original caiu mas o visitante reconectou na mesma
conversa, a mensagem vai para a conexão atual.

<Tabs groupId="linguagem">
<TabItem value="curl" label="cURL">

```bash
curl -X POST "$LINKTOR_URL/api/v1/messages/send" \
  -H "X-API-Key: $LINKTOR_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "channel_id": "'"$CHANNEL_ID"'",
    "to": "5b0e7c9a-41d2-4e8f-9a37-2c6d1f0b8e44",
    "text": "Olá! Em que posso ajudar?"
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
    "to": "5b0e7c9a-41d2-4e8f-9a37-2c6d1f0b8e44",
    "text": "Olá! Em que posso ajudar?"
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
        "to": "5b0e7c9a-41d2-4e8f-9a37-2c6d1f0b8e44",
        "text": "Olá! Em que posso ajudar?"
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
		"to": "5b0e7c9a-41d2-4e8f-9a37-2c6d1f0b8e44",
		"text": "Olá! Em que posso ajudar?",
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
      "to": "5b0e7c9a-41d2-4e8f-9a37-2c6d1f0b8e44",
      "text": "Olá! Em que posso ajudar?"
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
    to = "5b0e7c9a-41d2-4e8f-9a37-2c6d1f0b8e44",
    text = "Olá! Em que posso ajudar?",
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
        'to' => '5b0e7c9a-41d2-4e8f-9a37-2c6d1f0b8e44',
        'text' => 'Olá! Em que posso ajudar?',
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
            "to": "5b0e7c9a-41d2-4e8f-9a37-2c6d1f0b8e44",
            "text": "Olá! Em que posso ajudar?"
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

A resposta é `202` com `data.status = "queued"`; o desfecho chega como `message.sent` ou
`message.failed`. Para responder dentro de uma conversa já existente, veja
[Enviar mensagens](../api/enviar-mensagens.md).

### Receber

Com `webhook_url` e `credentials.webhook_secret` definidos no canal, cada mensagem do visitante
chega na sua aplicação assim (assinatura e headers em [Receber eventos](../api/receber-eventos.md)):

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
      "contentType": "image",
      "content": {
        "media": {
          "url": "https://linktor.suaempresa.com/media/webchat/CHANNEL_ID/3f6c0a52.jpg",
          "mimeType": "image/jpeg",
          "filename": "foto.jpg",
          "size": 184320
        }
      },
      "senderId": "5b0e7c9a-41d2-4e8f-9a37-2c6d1f0b8e44",
      "senderType": "contact",
      "metadata": { "senderName": "Maria Souza", "externalId": "b7c1d2e3-0a4f-4b5c-8d6e-7f8091a2b3c4" }
    },
    "conversationId": "c1a5d3e2-7b84-4a60-8e55-2f1d9b0c7a66",
    "contactId": "7e9a4b10-5c2d-4f3e-b8a1-6d0f2e3c4b5a",
    "channelId": "CHANNEL_ID",
    "channelType": "webchat"
  }
}
```

`senderId` é o id da sessão; `metadata.externalId` é o id que o widget deu à mensagem. A forma exata
da URL de mídia depende do armazenamento configurado no servidor. Em mensagens de texto, o conteúdo
vem em `content.text` e não há `media`.

## Limites e particularidades

- **Anexos:** só imagem (`image/jpeg`, `image/png`, `image/gif`, `image/webp`) e áudio
  (`audio/webm`, `audio/ogg`, `audio/mpeg`, `audio/mp4`, `audio/x-m4a`, `audio/wav`,
  `audio/x-wav`), até **10 MB** por arquivo. O tipo é decidido pelo servidor a partir do MIME
  declarado; o nome original não define a extensão gravada.
- **Gravação de áudio:** usa o microfone do navegador (exige HTTPS) e para sozinha aos 2 minutos.
- **Limite de requisições por IP:** rajada de 10 conexões (repondo 1 por segundo) e rajada de 10
  uploads (repondo 1 a cada 2 segundos). Acima disso o servidor responde `429`.
- **Reconexão:** se o WebSocket cair, o widget tenta de novo até 5 vezes, com espera crescente. As
  mensagens escritas enquanto está offline ficam na fila e saem quando reconecta.
- **Sem histórico no widget:** recarregar a página mantém a sessão (e a conversa no painel), mas o
  widget não busca as mensagens anteriores — a janela reabre vazia.
- **Conversa:** as mensagens de uma sessão caem na conversa aberta daquele contato no canal; se não
  houver, uma nova é criada.
- **Mensagens de boas-vindas e offline:** veja a tabela de campos — hoje são gravadas, mas não
  aparecem para o visitante.

## Solução de problemas

| Sintoma | Causa provável | O que fazer |
|---|---|---|
| O balão aparece, mas o status fica em **Offline** | Domínio do site fora de `LINKTOR_WS_ALLOWED_ORIGINS` | Inclua a origem exata (`https://www.suaempresa.com`) e reinicie o servidor. |
| WebSocket responde `503 channel not active` | Canal desconectado ou desabilitado | Clique em **Conectar** e confira se o canal está habilitado. |
| WebSocket responde `401 invalid or missing widget token` | Canal com `widget_secret` e página sem `token`, ou token vencido | Gere o token no backend a cada carregamento e passe no `init`. |
| O balão nem aparece | Script bloqueado, `channelId` ou `baseUrl` ausentes | Veja o console do navegador: o widget acusa `channelId is required` ou `baseUrl is required`. |
| Título e cor não acompanham o painel | O snippet fixa `title` ou `primaryColor` | Remova essas opções do snippet para valer o que está no canal, ou atualize o snippet. |
| Botão de anexo não aparece | **Permitir anexos** desligado | Ligue na aba **Comportamento** e salve. |
| Upload responde `403 attachments are disabled for this channel` | Idem, com `allowAttachments: true` forçado no snippet | A decisão é do servidor: ligue **Permitir anexos**. |
| Upload responde `415` ou `413` | Tipo fora da lista ou arquivo acima de 10 MB | Envie imagem ou áudio dentro do limite. |
| Upload responde `503 attachment storage is not configured` | Servidor sem armazenamento de mídia | Configure o armazenamento (veja [Configuração](../implantar/configuracao.md)). |
| Upload bloqueado pelo navegador (CORS) | Domínio fora de `LINKTOR_CORS_ALLOWED_ORIGINS` | Inclua a origem do site nessa variável. |
| Cada mensagem do mesmo cliente vira um contato novo | Sessões diferentes (abas, navegador anônimo) sem `visitor.email` | Informe `visitor` com o e-mail quando o usuário estiver logado. |
| Envio pela API termina em `message.failed` | Visitante não está mais conectado | Só é possível entregar com a página aberta. |
| Nenhum evento chega na sua aplicação | Canal sem `webhook_url` ou sem `credentials.webhook_secret` | Defina os dois pela API; sem segredo o Linktor descarta a entrega. |

Veja também: [Visão geral dos canais](visao-geral.md) · [Canais no painel](../painel/canais.md) ·
[Tempo real](../api/tempo-real.md).
