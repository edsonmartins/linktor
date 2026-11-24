---
sidebar_position: 8
title: Rust
description: Referência do SDK Rust do Linktor (crate linktor) — o que funciona hoje (canais e render VRE), como enviar mensagens com reqwest e como verificar a assinatura do webhook com axum.
---

# SDK Rust

O crate `linktor` (em `sdks/rust`) é um cliente assíncrono sobre `reqwest` e `tokio`. Hoje a parte
confiável dele é a **gestão de canais** (criar, atualizar, remover, conectar por QR ou código de
pareamento). Os outros recursos existem no código, mas não conversam com a API atual: use
`reqwest` direto para eles, como mostrado nesta página.

:::warning Leia antes de usar
- **Só os canais têm modelos alinhados com a API.** Conversas, contatos, bots, IA, base de
  conhecimento, fluxos e login usam `#[serde(rename_all = "camelCase")]`, enquanto a API responde
  em snake_case. Em Rust isso não resulta em campo vazio: a desserialização **falha** e o método
  devolve `Err(LinktorError::Serialization)`, mesmo quando o servidor executou a operação.
- **Não há envio direto** (`POST /messages/send`), nem WebSocket, nem analytics.
- **O verificador de webhook do SDK não valida as entregas reais** (assina só o corpo; o servidor
  assina `timestamp.corpo`). Use a verificação manual da seção de webhook.
- O crate não expõe um método genérico de requisição. Para tudo o que ele não cobre, o caminho é
  `reqwest` com o header `X-API-Key`.

A lista completa está em [Limitações conhecidas](#limitacoes).
:::

| Recurso | Coberto pelo SDK? |
|---|---|
| Canais (CRUD, conectar, pair code) | Sim (`disconnect` é parcial) |
| Enviar mensagem — envio direto | Não — use `reqwest` |
| Enviar mensagem — para uma conversa | Não (corpo incompatível) — use `reqwest` |
| Conversas | Não (modelos camelCase, `PATCH` inexistente) |
| Contatos | Parcial — só `delete` |
| Bots | Parcial — só `delete` |
| IA | Não (rotas inexistentes) |
| Base de conhecimento | Parcial — só `delete` |
| Fluxos | Parcial — só `delete` |
| Analytics | Não existe no SDK |
| VRE | Parcial — `render` e os atalhos `render_*` |
| Verificação de webhook | Incompatível com o servidor — verificação manual |
| Tempo real (WebSocket) | Não existe no SDK |
| Autenticação por API key | Sim (`X-API-Key`) |
| Login por e-mail e senha | Não (`login` falha ao ler a resposta) |

## Instalação {#instalacao}

O manifest declara o pacote `linktor`, versão `1.0.0`.

```bash
cargo add linktor
```

:::note Publicação no crates.io não confirmada
Não foi possível confirmar que `linktor` 1.0.0 está publicado no crates.io. O caminho garantido é
instalar a partir do repositório.
:::

Dependência git — o Cargo procura o pacote pelo `name` dentro do repositório, então não é preciso
informar a subpasta `sdks/rust`:

```toml
[dependencies]
linktor = { git = "https://github.com/edsonmartins/linktor" }
tokio = { version = "1", features = ["full"] }
```

Fixe um commit com `rev = "<sha>"` (ou `tag`/`branch`) para builds reproduzíveis. Com o repositório
clonado ao lado do seu projeto, use `path`:

```toml
[dependencies]
linktor = { path = "../linktor/sdks/rust" }
```

Os exemplos desta página também usam `reqwest`, `serde_json` e, no receptor de webhook, `axum`,
`hmac`, `sha2` e `hex`:

```toml
reqwest = { version = "0.12", features = ["json", "rustls-tls"] }
serde_json = "1"
axum = "0.8"
hmac = "0.12"
sha2 = "0.10"
hex = "0.4"
```

## Requisitos

- Rust edição 2021. O manifest não declara `rust-version`; use um toolchain estável recente (os
  exemplos desta página foram compilados com Rust 1.97).
- Runtime assíncrono **tokio** — todos os métodos de rede são `async`.
- Dependências do crate: `reqwest` 0.12 (com `rustls-tls`), `tokio` 1, `serde`/`serde_json` 1,
  `serde_urlencoded` 0.7, `chrono` 0.4, `thiserror` 2, `hmac` 0.12, `sha2` 0.10, `hex` 0.4,
  `tokio-tungstenite` 0.24, `futures-util` 0.3 e `url` 2 (as três últimas são declaradas, mas
  nenhum código do crate as usa).

## Criar o cliente

```rust
use linktor::LinktorClient;

fn criar_cliente() -> Result<LinktorClient, Box<dyn std::error::Error>> {
    let servidor = std::env::var("LINKTOR_URL")?; // https://linktor.suaempresa.com

    let client = LinktorClient::builder()
        .base_url(format!("{}/api/v1", servidor.trim_end_matches('/')))
        .api_key(std::env::var("LINKTOR_API_KEY")?)
        .timeout(30)
        .max_retries(3)
        .build()?;
    Ok(client)
}
```

:::warning A base URL precisa terminar em `/api/v1`
O SDK concatena o caminho do recurso (`/channels`, `/vre/render`…) diretamente em `base_url`, sem
acrescentar `/api/v1`. Passe `https://linktor.suaempresa.com/api/v1`. O valor padrão
(`https://api.linktor.io`) não serve para uma instalação própria.
:::

| Método do builder | Tipo | Padrão | Observação |
|---|---|---|---|
| `base_url(url)` | `impl Into<String>` | `https://api.linktor.io` | Inclua `/api/v1`. A barra final é removida. |
| `api_key(key)` | `impl Into<String>` | nenhum | Enviada em `X-API-Key`. |
| `access_token(token)` | `impl Into<String>` | nenhum | JWT de usuário, enviado em `Authorization: Bearer`. Ignorado se houver `api_key`. |
| `timeout(secs)` | `u64` | `30` | Timeout total de cada requisição, em segundos. |
| `max_retries(n)` | `u32` | `3` | Número máximo de **tentativas** (não de repetições). `1` desliga as repetições. |
| `build()` | `Result<LinktorClient>` | — | Só falha se o cliente HTTP não puder ser criado. |

Não há opção para headers extras, cliente `reqwest` próprio ou callbacks. `LinktorClient` implementa
`Clone` (o clone compartilha conexões e token); os recursos são obtidos por `client.channels()`,
`client.vre()` etc.

**Repetições.** Em `429` o cliente espera o valor de `Retry-After` (ou 60 s) e tenta de novo; em
`5xx` espera 2 s, depois 4 s. Isso vale para qualquer verbo, inclusive `POST` — um `create` que
recebeu `5xx` pode ser reenviado. Erros de rede e timeout não são repetidos.

## Autenticação

| Credencial | Como configurar | Header enviado |
|---|---|---|
| API key (`lk_...`) | `.api_key(...)` | `X-API-Key: lk_sua_chave` |
| JWT de usuário | `.access_token(...)` ou `client.set_access_token(Some(token)).await` | `Authorization: Bearer <token>` |

A API key é o caminho para integração entre servidores e é o header que o servidor espera. Crie a
chave e escolha os scopes em [Chaves de API](../painel/chaves-de-api.md); os detalhes estão em
[Autenticação](../api/autenticacao.md). Se as duas credenciais forem configuradas, só a API key é
enviada.

`client.auth().login(email, senha)` **não funciona** (o servidor autentica, mas o SDK não consegue
ler a resposta). Para usar JWT, faça o login com `reqwest` e entregue o token ao cliente:

```rust
let login: serde_json::Value = reqwest::Client::new()
    .post(format!("{base}/auth/login")) // base = https://linktor.suaempresa.com/api/v1
    .json(&serde_json::json!({ "email": "voce@suaempresa.com", "password": "sua-senha" }))
    .send()
    .await?
    .error_for_status()?
    .json()
    .await?;
let token = login["data"]["access_token"].as_str().ok_or("login sem access_token")?;

let client = LinktorClient::builder().base_url(&base).access_token(token).build()?;
```

O token expira (`data.expires_in`, em segundos). Renove com `POST /auth/refresh` (corpo
`{"refresh_token": "..."}`) e chame `client.set_access_token(Some(novo)).await`.

## Canais

`client.channels()` devolve um `ChannelsResource`. Os tipos `Channel`, `ConnectResult`,
`CreateChannelInput` e `UpdateChannelInput` seguem o formato real da API.

```rust
pub async fn list(&self, params: Option<ListChannelsParams>) -> Result<Vec<Channel>>
pub async fn get(&self, id: &str) -> Result<Channel>
pub async fn create(&self, input: CreateChannelInput) -> Result<Channel>
pub async fn update(&self, id: &str, input: UpdateChannelInput) -> Result<Channel>
pub async fn delete(&self, id: &str) -> Result<()>
pub async fn connect(&self, id: &str) -> Result<ConnectResult>
pub async fn request_pair_code(&self, id: &str, phone_number: &str) -> Result<ConnectResult>
pub async fn disconnect(&self, id: &str) -> Result<Channel>
```

### Criar um canal WhatsApp não-oficial

```rust
use linktor::{ChannelType, CreateChannelInput, LinktorClient};

async fn criar_canal(client: &LinktorClient) -> Result<String, linktor::Error> {
    let canal = client
        .channels()
        .create(CreateChannelInput::new("WhatsApp Vendas", ChannelType::WhatsappUnofficial))
        .await?;

    println!("{} {:?}", canal.id, canal.connection_status); // ... Disconnected
    Ok(canal.id)
}
```

`CreateChannelInput::new(name, channel_type)` aceita em cadeia `.identifier(...)`, `.config(map)`,
`.credentials(map)` e `.webhook_url(...)`; `config` e `credentials` são `HashMap<String, String>`.
Segredos vão em `credentials` e nunca voltam nas respostas. O input não tem o campo `environment`,
então o canal é sempre criado como `production`. Os valores de `ChannelType` são serializados em
snake_case (`WhatsappUnofficial` → `whatsapp_unofficial`); o que cada canal exige em `config` e
`credentials` está nas páginas de canal, por exemplo
[WhatsApp não-oficial](../canais/whatsapp-nao-oficial.md).

### Conectar por QR code

`connect` devolve um `ConnectResult` com `qr_code` (o **conteúdo** do QR, não uma imagem) e
`expires_in` (validade em segundos). Enquanto o QR vale, consulte o canal; quando expirar, chame
`connect` de novo para receber outro.

```rust
use linktor::{ConnectionStatus, LinktorClient};
use std::time::Duration;

async fn conectar_por_qr(client: &LinktorClient, channel_id: &str) -> Result<(), linktor::Error> {
    loop {
        let res = client.channels().connect(channel_id).await?;

        if res.channel.as_ref().map(|c| c.connection_status) == Some(ConnectionStatus::Connected) {
            return Ok(()); // já havia sessão salva: conectou sem QR
        }
        if res.passkey_required == Some(true) {
            // Conta protegida por passkey: não conecta por QR (veja a página do canal).
            return Ok(());
        }
        let Some(qr) = res.qr_code else {
            tokio::time::sleep(Duration::from_secs(3)).await;
            continue;
        };

        // `qr` é o CONTEÚDO do QR code: gere a imagem com a biblioteca de sua
        // preferência e mostre para a pessoa escanear no WhatsApp.
        println!("Escaneie: {qr}");

        // Enquanto o QR vale, consulta o canal; quando expira, pede outro.
        let validade = Duration::from_secs(res.expires_in.unwrap_or(60) as u64);
        let inicio = tokio::time::Instant::now();
        while inicio.elapsed() < validade {
            tokio::time::sleep(Duration::from_secs(3)).await;
            let canal = client.channels().get(channel_id).await?;
            if canal.connection_status == ConnectionStatus::Connected {
                return Ok(());
            }
        }
    }
}
```

Campos de `ConnectResult`: `channel: Option<Channel>`, `qr_code`, `expires_in`, `pair_code`,
`passkey_required` e `passkey_challenge` — todos `Option`.

### Conectar por código de pareamento

```rust
let res = client
    .channels()
    .request_pair_code("CHANNEL_ID", "+5511999999999")
    .await?;

if let Some(codigo) = res.pair_code {
    println!("Digite no WhatsApp: {codigo}");
}
```

Depois de a pessoa digitar o código no aparelho, acompanhe `connection_status` com
`channels().get(...)`, como no fluxo do QR.

### Atualizar: webhook de saída e segredo

`update` faz `PUT /channels/{id}` e só envia os campos preenchidos (`None` = não mexe). A URL que
recebe os eventos é `webhook_url`; o segredo usado para assinar as entregas é
`credentials.webhook_secret`.

```rust
use linktor::UpdateChannelInput;
use std::collections::HashMap;

let canal = client
    .channels()
    .update(
        "CHANNEL_ID",
        UpdateChannelInput {
            name: Some("WhatsApp Vendas".into()),
            webhook_url: Some("https://app.suaempresa.com/webhooks/linktor".into()),
            credentials: Some(HashMap::from([(
                "webhook_secret".to_string(),
                "um-segredo-longo-e-aleatorio".to_string(),
            )])),
            ..Default::default()
        },
    )
    .await?;
```

Para receber só alguns eventos, envie em `config` a chave `webhook_events` com os tipos separados
por vírgula (vazio = todos). `config` substitui as chaves não secretas do canal: leia o `config`
atual com `get` e reenvie o mapa completo com a alteração.

### Listar, consultar, remover e desconectar

```rust
use linktor::ChannelType;

let canais = client.channels().list(None).await?; // Vec<Channel>
for c in canais.iter().filter(|c| c.channel_type == ChannelType::WhatsappUnofficial) {
    println!("{} {} {:?} enabled={}", c.id, c.name, c.connection_status, c.enabled);
}

let canal = client.channels().get("CHANNEL_ID").await?;
client.channels().delete("CHANNEL_ID").await?; // 204 sem corpo -> Ok(())
```

:::warning Filtros de `list` e retorno de `disconnect`
- Os filtros de `ListChannelsParams` (`type`, `status`, `search`) são enviados na query, mas o
  servidor os ignora: a lista vem sempre completa. Filtre no cliente, como acima.
- `disconnect` chama `POST /channels/{id}/disconnect`, e o servidor desconecta o canal. Porém a
  resposta é `{"message": "Channel disconnected"}`, não um canal, e o método devolve
  `Err(LinktorError::Serialization)`. Trate esse erro como sucesso:

```rust
match client.channels().disconnect("CHANNEL_ID").await {
    Ok(_) | Err(linktor::Error::Serialization(_)) => {} // desconectado
    Err(e) => return Err(e),
}
```
:::

O struct `Channel` traz `id`, `tenant_id`, `channel_type` (campo `type` da API), `name`,
`identifier`, `enabled`, `connection_status`, `config`, `webhook_url`, `created_at`, `updated_at` e
os campos de coexistência do WhatsApp oficial. O campo `environment` da API não está no struct.

## Enviar mensagem

O SDK não tem método para `POST /messages/send`. Use `reqwest`:

```rust
use serde_json::{json, Value};

#[tokio::main]
async fn main() -> Result<(), Box<dyn std::error::Error>> {
    let base = format!("{}/api/v1", std::env::var("LINKTOR_URL")?);
    let api_key = std::env::var("LINKTOR_API_KEY")?;
    let http = reqwest::Client::new();

    let resp = http
        .post(format!("{base}/messages/send"))
        .header("X-API-Key", &api_key)
        .json(&json!({
            "channel_id": "CHANNEL_ID",
            "to": "+5511999999999",
            "content_type": "text",
            "text": "Olá! Seu pedido saiu para entrega.",
            "metadata": { "pedido": "12345" }
        }))
        .send()
        .await?;

    let status = resp.status();
    let corpo: Value = resp.json().await?;
    if !status.is_success() {
        return Err(format!("{status}: {}", corpo["error"]["message"]).into());
    }

    // 202 Accepted: a mensagem entrou na fila de envio.
    println!("{} {}", corpo["data"]["id"], corpo["data"]["status"]); // "..." "queued"
    Ok(())
}
```

A chave precisa do scope `messages:send`. `metadata` é um mapa de string para string; campos,
anexos e chaves reservadas estão em [Enviar mensagens](../api/enviar-mensagens.md).

## Conversas

`client.conversations()` existe, mas nenhum método dele é utilizável hoje (veja
[Limitações conhecidas](#limitacoes)). Em particular, **não use** `send_text`/`send_message`: o
corpo enviado é `{"text": ..., "type": ...}`, e a rota espera `content_type` e `content`.

Responder em uma conversa existente com `reqwest`:

```rust
use serde_json::{json, Value};

async fn responder(http: &reqwest::Client, base: &str, api_key: &str) -> Result<(), Box<dyn std::error::Error>> {
    let conversation_id = "CONVERSATION_ID";

    let corpo: Value = http
        .post(format!("{base}/conversations/{conversation_id}/messages"))
        .header("X-API-Key", api_key)
        .json(&json!({ "content_type": "text", "content": "Já estou verificando para você." }))
        .send()
        .await?
        .error_for_status()?
        .json()
        .await?;
    println!("{}", corpo["data"]["id"]);
    Ok(())
}
```

Contatos, bots, IA, base de conhecimento, fluxos e analytics seguem o mesmo padrão: `reqwest` com
`X-API-Key` nas rotas descritas em [Visão geral da API](../api/visao-geral.md). Do SDK, só os
métodos `delete` desses recursos funcionam (`contacts().delete(id)`, `bots().delete(id)`,
`knowledge_bases().delete(id)`, `flows().delete(id)`).

## VRE

`client.vre().render(...)` funciona: o corpo é snake_case e o cliente aceita a resposta sem
envelope das rotas `/vre/*`. Essas rotas só existem quando o serviço VRE está habilitado no
servidor.

```rust
use linktor::{LinktorClient, VREChannelType, VREOutputFormat, VRERenderRequest};
use std::collections::HashMap;

async fn renderizar(client: &LinktorClient) -> Result<(), linktor::Error> {
    let dados = HashMap::from([
        ("titulo".to_string(), serde_json::json!("Como posso ajudar?")),
        (
            "opcoes".to_string(),
            serde_json::json!([{ "label": "Segunda via" }, { "label": "Falar com atendente" }]),
        ),
    ]);

    // Primeiro argumento (tenant_id): deixe vazio; o servidor usa o tenant da credencial.
    let req = VRERenderRequest::new("", "menu_opcoes", dados)
        .channel(VREChannelType::Whatsapp)
        .format(VREOutputFormat::Png);

    let img = client.vre().render(req).await?;
    println!("{}x{} {}", img.width, img.height, img.caption);
    // img.image_base64 contém a imagem.
    Ok(())
}
```

`VRERenderRequest` também aceita `.width(i32)`, `.quality(i32)` e `.scale(f64)`. Os atalhos
`render_menu`, `render_product_card`, `render_order_status`, `render_product_list`,
`render_confirmation` e `render_pix_payment` montam o `data` dos modelos `menu_opcoes`,
`card_produto`, `status_pedido`, `lista_produtos`, `confirmacao` e `cobranca_pix` e chamam `render`;
o primeiro argumento de todos é o `tenant_id` (passe `""`).

:::warning O que observar no `render`
- Um `tenant_id` preenchido e diferente do tenant da credencial resulta em `403`.
- `render_time_ms` chega em **nanossegundos** (o servidor serializa uma duração) e o SDK o lê como
  `i32`: um render que leve mais de ~2,1 s devolve `Err(LinktorError::Serialization)`, embora a
  imagem tenha sido gerada.
- `render_and_send`, `list_templates` e `preview` não funcionam (veja as limitações). Para
  renderizar e enviar, chame `POST /vre/render-and-send` com `reqwest`, informando `send_to`,
  `channel` e `channel_id`.
:::

## Verificar a assinatura do webhook {#webhook}

O Linktor envia cada evento para o `webhook_url` do canal com dois headers:
`X-Linktor-Timestamp` (epoch em segundos) e `X-Linktor-Signature` (HMAC-SHA256 em hex minúsculo,
sem prefixo, de `timestamp + "." + corpo cru`, com o `credentials.webhook_secret` do canal como
chave).

:::warning Os verificadores do SDK rejeitam entregas legítimas
`linktor::webhook::verify` e `linktor::webhook::construct_event` calculam o HMAC **só sobre o corpo** e usam o timestamp apenas para a janela de
tolerância. Como o servidor assina `timestamp.corpo`, o resultado é sempre `false` (ou
`Err(WebhookVerification)`) para uma entrega real. Faça a verificação como abaixo.
:::

Receptor mínimo com axum, extraindo `HeaderMap` e `Bytes` para ter o corpo exatamente como chegou:

```rust
use axum::{body::Bytes, extract::State, http::{HeaderMap, StatusCode}, routing::post, Router};
use hmac::{Hmac, Mac};
use sha2::Sha256;
use std::sync::Arc;
use std::time::{SystemTime, UNIX_EPOCH};

const TOLERANCIA_SEGUNDOS: i64 = 300;

fn assinatura_valida(headers: &HeaderMap, corpo: &[u8], segredo: &str) -> bool {
    let header = |nome: &str| headers.get(nome).and_then(|v| v.to_str().ok());
    let (Some(assinatura), Some(timestamp)) = (header("X-Linktor-Signature"), header("X-Linktor-Timestamp")) else {
        return false;
    };

    // Anti-replay: o timestamp (epoch em segundos) precisa estar a no máximo 300 s de agora.
    let Ok(ts) = timestamp.parse::<i64>() else { return false };
    let agora = SystemTime::now().duration_since(UNIX_EPOCH).map(|d| d.as_secs() as i64).unwrap_or(0);
    if (agora - ts).abs() > TOLERANCIA_SEGUNDOS {
        return false;
    }

    // HMAC-SHA256 de "<timestamp>.<corpo cru>", em hex minúsculo, sem prefixo.
    let Ok(recebida) = hex::decode(assinatura) else { return false };
    let Ok(mut mac) = Hmac::<Sha256>::new_from_slice(segredo.as_bytes()) else { return false };
    mac.update(timestamp.as_bytes());
    mac.update(b".");
    mac.update(corpo);
    mac.verify_slice(&recebida).is_ok() // comparação em tempo constante
}

async fn receber(State(segredo): State<Arc<String>>, headers: HeaderMap, corpo: Bytes) -> StatusCode {
    if !assinatura_valida(&headers, &corpo, &segredo) {
        return StatusCode::UNAUTHORIZED;
    }

    // O envelope bate com o tipo do SDK (id, type, timestamp, tenantId, data).
    let Ok(evento) = serde_json::from_slice::<linktor::WebhookEvent>(&corpo) else {
        return StatusCode::BAD_REQUEST;
    };

    let dados = evento.data.unwrap_or_default();
    match evento.event_type.as_str() {
        "message.received" => {
            let texto = dados["message"]["content"]["text"].as_str().unwrap_or("");
            println!("{} escreveu: {texto}", dados["contactId"]);
        }
        "message.delivered" | "message.read" | "message.failed" => {
            println!("{} -> {}", dados["messageId"], dados["status"]);
        }
        outro => println!("evento {outro} ({})", evento.id),
    }
    StatusCode::OK
}

#[tokio::main]
async fn main() -> Result<(), Box<dyn std::error::Error>> {
    let segredo = Arc::new(std::env::var("LINKTOR_WEBHOOK_SECRET")?);
    let app = Router::new().route("/webhooks/linktor", post(receber)).with_state(segredo);

    let listener = tokio::net::TcpListener::bind("0.0.0.0:3000").await?;
    axum::serve(listener, app).await?;
    Ok(())
}
```

O que dá para aproveitar do SDK:

- `linktor::WebhookEvent` desserializa o envelope real (`id`, `event_type`, `timestamp`,
  `tenant_id`, `data` como `HashMap<String, serde_json::Value>`). O campo `environment` do envelope
  é descartado.
- Compare o tipo pela string `event_type`. `get_event_type()` devolve `None` para tipos que o enum
  `EventType` não conhece (`call.*`, `conversation.reopened`, `conversation.escalated`,
  `conversation.deleted`).
- `linktor::webhook::verify_signature(bytes, assinatura, segredo)` compara em tempo constante e
  valida corretamente **se** você passar em `bytes` a concatenação `timestamp + "." + corpo`; a
  checagem dos 300 s continua por sua conta.

Os eventos e o formato de `data` estão em [Receber eventos](../api/receber-eventos.md).

## Tratamento de erros

Todos os métodos devolvem `linktor::Result<T>`, isto é, `Result<T, LinktorError>` (também exportado
como `linktor::Error`).

| Variante | Quando | Campos |
|---|---|---|
| `Validation` | HTTP 400 | `message`, `request_id` |
| `Authentication` | HTTP 401 | `message`, `request_id` |
| `Authorization` | HTTP 403 (scope ou papel insuficiente) | `message`, `request_id` |
| `NotFound` | HTTP 404 | `message`, `request_id` |
| `RateLimit` | HTTP 429 depois de esgotar as tentativas | `retry_after` (sempre `60`), `message`, `request_id` |
| `Server` | HTTP 5xx depois de esgotar as tentativas | `message`, `request_id` |
| `Unknown` | Qualquer outro status (409, 405, 422…) | `message`, `status_code` |
| `Network(reqwest::Error)` | Falha de conexão, DNS, TLS ou timeout | — |
| `Serialization(serde_json::Error)` | A resposta de sucesso não tem o formato que o SDK espera | — |
| `WebhookVerification` | `webhook::construct_event` | `message` |
| `WebSocket` | Nunca é produzido (não há cliente WebSocket) | `message` |

`request_id()` devolve o `X-Request-ID` da resposta nas variantes que o guardam.

:::warning `message` é o JSON inteiro do erro
O SDK tenta ler o erro como `{"code", "message"}` na raiz, mas a API responde
`{"success": false, "error": {"code", "message", "details"}}`. Por isso `message` contém o corpo
bruto da resposta. Extraia o código e a mensagem com `serde_json`:
:::

```rust
use linktor::{LinktorClient, LinktorError};

async fn consultar(client: &LinktorClient) {
    match client.channels().get("CHANNEL_ID").await {
        Ok(canal) => println!("{}", canal.name),
        Err(LinktorError::NotFound { message, request_id }) => {
            // `message` é o corpo JSON inteiro da resposta de erro.
            let corpo: serde_json::Value = serde_json::from_str(&message).unwrap_or_default();
            eprintln!(
                "não encontrado: {} ({}) request_id={:?}",
                corpo["error"]["message"], corpo["error"]["code"], request_id
            );
        }
        Err(LinktorError::Authentication { .. }) => eprintln!("chave ausente, inválida ou revogada"),
        Err(LinktorError::Authorization { .. }) => eprintln!("a chave não tem o scope necessário"),
        Err(LinktorError::Network(e)) => eprintln!("rede/timeout: {e}"),
        Err(LinktorError::Serialization(e)) => eprintln!("resposta em formato inesperado: {e}"),
        Err(e) => eprintln!("{e}"),
    }
}
```

Códigos de erro e paginação da API: [Erros e paginação](../api/erros-e-paginacao.md).

## Limitações conhecidas {#limitacoes}

Um `Err(Serialization)` nesta tabela significa que a requisição **chegou ao servidor**; se a rota e
o corpo eram válidos, a operação foi executada e só a leitura da resposta falhou. Não repita a
chamada às cegas.

| Método | O que faz | Por que não funciona | Alternativa |
|---|---|---|---|
| `auth().login` | `POST /auth/login` | O servidor autentica, mas `LoginResponse` espera `accessToken`, `refreshToken` e `tenant`; a resposta é snake_case e não traz `tenant` | Login com `reqwest` + `access_token(...)` |
| `auth().refresh_token` | `POST /auth/refresh` | Envia `refreshToken`; a rota espera `refresh_token` | `reqwest` |
| `auth().get_current_user`, `get_current_tenant` | `GET /auth/me`, `GET /auth/tenant` | Rotas inexistentes (a que existe é `GET /me`) | `reqwest` em `GET /me` |
| `conversations().list`, `contacts().list`, `bots().list`, `knowledge_bases().list`, `flows().list` | `GET` da coleção | Esperam `{data, pagination}`; a API devolve a lista em `data` e a paginação em `meta` | `reqwest` |
| `conversations().get`, `resolve` | `GET /conversations/{id}`, `POST .../resolve` | `Conversation` é camelCase (`tenantId`, `channelId`…); `resolve` é executado no servidor, mas devolve erro | `reqwest` |
| `conversations().assign` | `POST .../assign` | Envia `agentId`; a rota exige `user_id` | `reqwest` |
| `conversations().send_text`, `send_message` | `POST /conversations/{id}/messages` | Envia `text` e `type`; a rota espera `content` e `content_type`. `Message` também é camelCase | `reqwest` (seção Conversas) |
| `conversations().get_messages` | `GET /conversations/{id}/messages` | Espera `{data, pagination}` e envia `cursor`; a rota pagina por `before` | `reqwest` |
| `conversations().update`, `contacts().update`, `bots().update`, `flows().update` | `PATCH` | A API usa `PUT`; não há rota `PATCH` | `reqwest` com `PUT` |
| `contacts().create`, `get` | `POST /contacts`, `GET /contacts/{id}` | `Contact` é camelCase; o `create` **cria o contato** e devolve erro. `avatar`/`customFields` não são os nomes da API (`avatar_url`, `custom_fields`) | `reqwest` |
| `bots().create`, `get` | `POST /bots`, `GET /bots/{id}` | O input não tem `provider` nem `model`, obrigatórios; `Bot` é camelCase | `reqwest` |
| `ai().completions()`, `embeddings()`, `agents()` | `/ai/completions`, `/ai/embeddings`, `/ai/agents` | Rotas inexistentes | `reqwest` em `POST /ai/complete` |
| `knowledge_bases().create`, `get` | `POST`/`GET /knowledge-bases` | O input não tem `type`, obrigatório; `KnowledgeBase` exige campos que a API não devolve | `reqwest` |
| `knowledge_bases().query`, `add_document` | `/query`, `/documents` | Rotas inexistentes | `reqwest` em `POST .../search` e `POST .../items` |
| `flows().create`, `get`, `execute` | `POST`/`GET /flows`, `POST .../execute` | Input sem `trigger` e `start_node_id`; `Flow` exige `status` e `version`; `/execute` não existe | `reqwest` |
| `channels().disconnect` | `POST /channels/{id}/disconnect` | Desconecta, mas a resposta não é um canal | Tratar `Serialization` como sucesso |
| `vre().render_and_send` | `POST /vre/render-and-send` | Envia `conversation_id`; a rota exige `send_to`, `channel` e `channel_id` | `reqwest` |
| `vre().list_templates` | `GET /vre/templates` | A API devolve uma lista de ids (strings); o SDK espera objetos | `reqwest` |
| `vre().preview` | `POST /vre/templates/{id}/preview` | A rota é `GET` | `reqwest` com `GET` |
| `webhook::verify`, `construct_event` | Verificam a assinatura | Assinam só o corpo, sem o timestamp | [Verificação manual](#webhook) |

Também não existem no SDK: envio direto, analytics, WebSocket, upload de mídia, modelos de mensagem
e campanhas. Para acompanhar eventos em tempo real a partir de um servidor, prefira o webhook; o
WebSocket da API aceita apenas JWT de usuário ([Tempo real](../api/tempo-real.md)).

## Referência: método → rota HTTP

Os caminhos são relativos à base `/api/v1`.

| Método | Rota |
|---|---|
| `channels().list(params)` | `GET /channels` |
| `channels().get(id)` | `GET /channels/{id}` |
| `channels().create(input)` | `POST /channels` |
| `channels().update(id, input)` | `PUT /channels/{id}` |
| `channels().delete(id)` | `DELETE /channels/{id}` |
| `channels().connect(id)` | `POST /channels/{id}/connect` |
| `channels().request_pair_code(id, phone_number)` | `POST /channels/{id}/pair` |
| `channels().disconnect(id)` (retorno com erro, veja acima) | `POST /channels/{id}/disconnect` |
| `contacts().delete(id)` | `DELETE /contacts/{id}` |
| `bots().delete(id)` | `DELETE /bots/{id}` |
| `knowledge_bases().delete(id)` | `DELETE /knowledge-bases/{id}` |
| `flows().delete(id)` | `DELETE /flows/{id}` |
| `vre().render(request)` e atalhos `render_*` | `POST /vre/render` |
| `auth().logout()` | `POST /auth/logout` |
| `set_access_token(token)` | nenhuma (troca o token em memória) |

Comparação com os outros SDKs: [Visão geral dos SDKs](./visao-geral.md).
