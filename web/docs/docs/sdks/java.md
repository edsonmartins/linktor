---
sidebar_position: 5
title: Java
description: Referência do SDK Java do Linktor (io.linktor:linktor-sdk) — instalação, cliente, canais, o que funciona, o que não funciona e como contornar com HTTP nativo.
---

# SDK Java

O SDK Java (`io.linktor:linktor-sdk`, Java 17+) autentica com API key no header correto (`X-API-Key`) e tem a **gestão de canais completa e alinhada ao backend**. Os demais recursos foram escritos contra um contrato antigo: boa parte dos métodos chama rotas que não existem ou envia campos em camelCase que o servidor ignora. Esta página documenta só o que foi conferido contra o backend e mostra o contorno para o resto.

| Recurso | Coberto pelo SDK | Observação |
|---|---|---|
| Canais | sim | criar, listar, obter, atualizar, remover, conectar (QR), pair code, desconectar |
| Envio direto (`POST /messages/send`) | não | use `java.net.http` ou `client.getHttpClient()` |
| Mensagem para uma conversa | não | `sendText`/`sendMessage` enviam corpo incompatível |
| Conversas | parcial | só `get`, `resolve`, `reopen`; campos compostos voltam `null` |
| Contatos | parcial | só `get`, `create`, `delete` |
| Bots | parcial | só `get`, `delete` |
| IA | não | todas as rotas do SDK (`/ai/agents`, `/ai/completions`, `/ai/embeddings`) não existem |
| Base de conhecimento | parcial | só `get`, `delete` |
| Fluxos | parcial | só `get`, `delete` |
| Analytics | não | rotas inexistentes ou resposta que não desserializa |
| VRE | parcial | `render`, `listTemplates` e os atalhos `render*` |
| Verificação de webhook | não | o verificador assina só o corpo; o servidor assina `timestamp.corpo` |
| WebSocket | não | URL, autenticação e protocolo diferentes dos do servidor |
| Login por e-mail/senha | não | `auth.login` não captura o token (camelCase) |

:::warning Modelos em camelCase
O SDK usa Gson **sem política de nomes**. Só os tipos de canal (`Channel.*`) e parte dos de VRE têm `@SerializedName` em snake_case. Nos demais modelos, todo campo composto (`channelId`, `contactId`, `createdAt`, `customFields`, `accessToken`…) é lido como `null`/`0` e, na escrita, é ignorado pelo servidor. Campos de uma palavra só (`id`, `name`, `email`, `phone`, `status`, `tags`, `subject`, `metadata`) funcionam. Para qualquer coisa além disso, use o [acesso HTTP do próprio SDK](#http-direto) com `JsonObject`.
:::

## Instalação

Coordenadas do `pom.xml`:

```xml
<dependency>
  <groupId>io.linktor</groupId>
  <artifactId>linktor-sdk</artifactId>
  <version>1.0.0</version>
</dependency>
```

```kotlin
// build.gradle.kts
dependencies {
    implementation("io.linktor:linktor-sdk:1.0.0")
}
```

:::note Publicação no Maven Central não confirmada
Não foi possível confirmar que `io.linktor:linktor-sdk` está publicado no Maven Central. O caminho garantido é instalar a partir do repositório no seu repositório Maven local.
:::

```bash
git clone https://github.com/edsonmartins/linktor.git
cd linktor/sdks/java
mvn install -DskipTests
```

Depois disso a dependência acima resolve pelo `~/.m2`. No Gradle, acrescente `mavenLocal()` em `repositories`.

## Requisitos

- Java 17 ou superior (`maven.compiler.source`/`target` = 17).
- Dependências transitivas: OkHttp 4.12.0, Gson 2.10.1, Java-WebSocket 1.5.4.

## Criar o cliente

```java
import io.linktor.LinktorClient;

LinktorClient client = LinktorClient.builder()
        .baseUrl(System.getenv("LINKTOR_URL") + "/api/v1")
        .apiKey(System.getenv("LINKTOR_API_KEY"))
        .timeout(30)
        .maxRetries(3)
        .build();
```

| Opção | Tipo | Padrão | O que faz |
|---|---|---|---|
| `baseUrl(String)` | `String` | `https://api.linktor.io` | Base das chamadas. O SDK **não** acrescenta `/api/v1`: ele concatena direto `baseUrl + "/channels"`. Passe `https://linktor.suaempresa.com/api/v1`. A barra final é removida. |
| `apiKey(String)` | `String` | — | Enviada em `X-API-Key`. |
| `accessToken(String)` | `String` | — | JWT de usuário, enviado em `Authorization: Bearer`. Só é usado se `apiKey` estiver vazia. |
| `timeout(int)` | segundos | `30` | Timeout de conexão, leitura e escrita. |
| `maxRetries(int)` | `int` | `3` | **Número total de tentativas**, não de repetições. Com `0` nenhuma requisição é enviada e toda chamada lança `LinktorException("Request failed after 0 attempts")`. Use no mínimo `1`. |

:::warning O padrão de `baseUrl` não serve
O valor padrão `https://api.linktor.io` não inclui `/api/v1` e não é o seu servidor. Sempre informe `baseUrl`.
:::

Não há opção de headers customizados nem callbacks. Os recursos são campos públicos: `client.channels`, `client.contacts`, `client.conversations`, `client.bots`, `client.flows`, `client.knowledgeBases`, `client.vre`, `client.ai`, `client.analytics`, `client.auth`. `client.getHttpClient()` devolve o cliente HTTP interno.

Tentativas: erro de rede e respostas 500/502/503/504 são repetidos com espera de 2 s, 4 s…; resposta 429 espera **60 s fixos** antes de repetir (o header `Retry-After` não é lido). `POST` também é repetido — em operações não idempotentes use `maxRetries(1)`.

## Autenticação

Use API key: o SDK envia `X-API-Key: lk_sua_chave`, que é o que o servidor espera para integração servidor-a-servidor. A chave precisa dos scopes das rotas chamadas (`channels:read`, `channels:write`, `messages:send`…); sem o scope a chamada lança `AuthorizationException` (403). Veja [Autenticação](../api/autenticacao.md) e [Chaves de API](../painel/chaves-de-api.md).

Se `apiKey` e `accessToken` forem informados juntos, só a API key é enviada.

:::warning `client.auth` não funciona
`auth.login(email, senha)` chama a rota certa, mas lê `accessToken` (camelCase) de uma resposta que traz `access_token`: o token volta `null` e não é guardado. `auth.refreshToken` envia `refreshToken` em vez de `refresh_token`; `getCurrentUser` e `getCurrentTenant` chamam `/auth/me` e `/auth/tenant`, que não existem. Para agir como usuário, faça `POST /auth/login` com `java.net.http` (corpo `email` e `password`), leia `data.access_token` da resposta e passe o JWT em `accessToken(...)`.
:::

## Canais

Todos os métodos abaixo foram conferidos contra `cmd/server/main.go` e os handlers; os tipos `Channel.*` serializam em snake_case.

Métodos: `list()`, `get(id)`, `create(input)`, `update(id, input)`, `delete(id)`, `connect(id)`, `requestPairCode(id, phoneNumber)` e `disconnect(id)`. `list` devolve `List<Channel.ChannelModel>`; `connect` e `requestPairCode` devolvem `Channel.ConnectResult`; `delete` é `void`; os demais devolvem `Channel.ChannelModel`.

### Criar um canal WhatsApp não oficial

```java
import io.linktor.types.Channel;

Channel.ChannelModel canal = client.channels.create(
        Channel.CreateChannelInput.builder()
                .type(Channel.ChannelType.WHATSAPP_UNOFFICIAL)
                .name("WhatsApp Vendas")
                .build());
String channelId = canal.getId();
```

O builder aceita `type`, `name`, `identifier`, `config` e `credentials` (ambos `Map<String, String>`) e `webhookUrl`. Não há como informar `environment` pelo SDK. `credentials` nunca volta nas respostas; segredos em `config` voltam como `__redacted__`. Detalhes do canal em [WhatsApp não oficial](../canais/whatsapp-nao-oficial.md).

### Conectar por QR code

`connect` devolve o conteúdo do QR em `getQrCode()` e a validade em `getExpiresIn()` (segundos). Renderize o conteúdo como imagem QR, acompanhe `connection_status` e peça um QR novo quando o atual expirar:

```java
boolean conectado = false;
while (!conectado) {
    Channel.ConnectResult r = client.channels.connect(channelId);
    if (r.getQrCode() == null) {
        break; // sem QR: sessão já pareada ou conta com passkey
    }
    System.out.println("Escaneie este QR: " + r.getQrCode());

    int validade = r.getExpiresIn() != null ? r.getExpiresIn() : 60;
    long limite = System.currentTimeMillis() + validade * 1000L;
    while (!conectado && System.currentTimeMillis() < limite) {
        Thread.sleep(3000);
        conectado = client.channels.get(channelId).getConnectionStatus()
                == Channel.ConnectionStatus.CONNECTED;
    }
}
```

`ConnectResult` também expõe `getChannel()`, `getPairCode()`, `getPasskeyRequired()` e `getPasskeyChallenge()` (`JsonElement`). O SDK não tem método para responder ao desafio de passkey (`POST /channels/{id}/passkey/response`).

### Conectar por código de pareamento

```java
Channel.ConnectResult par = client.channels.requestPairCode(channelId, "+5511999999999");
System.out.println("Digite no WhatsApp: " + par.getPairCode());
```

### Atualizar, incluindo o webhook da sua aplicação

O `update` é um `PUT` parcial: campos `null` não são enviados. A URL de destino dos eventos é `webhook_url`; o segredo do HMAC vai em `credentials.webhook_secret`; o filtro de eventos, em `config.webhook_events` (lista separada por vírgula; vazio = todos).

```java
import java.util.Map;

Channel.ChannelModel atualizado = client.channels.update(channelId,
        Channel.UpdateChannelInput.builder()
                .name("WhatsApp Vendas SP")
                .webhookUrl("https://app.suaempresa.com/webhooks/linktor")
                .credentials(Map.of("webhook_secret", "um-segredo-longo-e-aleatorio"))
                .config(Map.of("webhook_events", "message.received,message.failed"))
                .build());
```

### Listar, desconectar e remover

```java
for (Channel.ChannelModel c : client.channels.list()) {
    System.out.println(c.getId() + " " + c.getType() + " " + c.getConnectionStatus());
}
client.channels.disconnect(channelId);
client.channels.delete(channelId);
```

:::warning Retorno de `disconnect`
O servidor responde `{"message": "Channel disconnected"}`, não o canal. A chamada funciona, mas o `ChannelModel` devolvido vem com todos os campos `null`. Chame `channels.get` se precisar do estado atual.
:::

`ChannelModel` não tem getter para `environment`. Não há métodos para `PUT /channels/{id}/enabled`, `PUT /channels/{id}/status` nem `POST /channels/test`.

## Enviar mensagem

O SDK não tem método para o envio direto. Com o cliente HTTP nativo do Java:

```java
import com.google.gson.Gson;
import com.google.gson.JsonObject;
import java.net.URI;
import java.net.http.HttpClient;
import java.net.http.HttpRequest;
import java.net.http.HttpResponse;
import java.time.Duration;
import java.util.Map;

String baseUrl = System.getenv("LINKTOR_URL") + "/api/v1";
Gson gson = new Gson();

String corpo = gson.toJson(Map.of(
        "channel_id", "CHANNEL_ID",
        "to", "+5511999999999",
        "content_type", "text",
        "text", "Olá! Seu pedido saiu para entrega."));

HttpRequest request = HttpRequest.newBuilder(URI.create(baseUrl + "/messages/send"))
        .timeout(Duration.ofSeconds(30))
        .header("Content-Type", "application/json")
        .header("X-API-Key", System.getenv("LINKTOR_API_KEY"))
        .POST(HttpRequest.BodyPublishers.ofString(corpo))
        .build();

HttpResponse<String> response = HttpClient.newHttpClient()
        .send(request, HttpResponse.BodyHandlers.ofString());
if (response.statusCode() != 202) {
    throw new IllegalStateException("Envio recusado (" + response.statusCode() + "): " + response.body());
}
JsonObject data = gson.fromJson(response.body(), JsonObject.class).getAsJsonObject("data");
System.out.println(data.get("id").getAsString() + " " + data.get("status").getAsString());
```

A resposta é **202** com `data: {id, conversation_id, channel_id, status: "queued"}`. Campos, mídia e `metadata` em [Enviar mensagens](../api/enviar-mensagens.md).

### Pelo HTTP do próprio SDK {#http-direto}

`client.getHttpClient()` expõe `get`, `post`, `put` e `delete` com a mesma autenticação, as mesmas tentativas e o desempacotamento do envelope (`data`). Passe o corpo como `Map` com as chaves em snake_case e peça `com.google.gson.JsonObject` (ou `List<JsonObject>`, via `com.google.gson.reflect.TypeToken`, para listas):

```java
io.linktor.utils.HttpClient http = client.getHttpClient();

JsonObject enviado = http.post("/messages/send", Map.of(
        "channel_id", "CHANNEL_ID",
        "to", "+5511999999999",
        "content_type", "text",
        "text", "Olá! Seu pedido saiu para entrega."), JsonObject.class);
System.out.println(enviado.get("id").getAsString() + " " + enviado.get("status").getAsString());

// Responder dentro de uma conversa existente
JsonObject msg = http.post("/conversations/" + conversationId + "/messages",
        Map.of("content_type", "text", "content", "Já estamos verificando."), JsonObject.class);
```

Limites desse caminho: o `meta` de paginação é descartado (só `data` volta) e os valores de query string **não são codificados** — aplique `URLEncoder.encode` em valores com espaço, `+` ou `&`.

## Conversas

Funcionam `get(conversationId)`, `resolve(conversationId)` e `reopen(conversationId)`, todos devolvendo `Conversation.ConversationModel` parcial.

```java
import io.linktor.types.Conversation;

Conversation.ConversationModel conv = client.conversations.get(conversationId);
System.out.println(conv.getStatus() + " " + conv.getSubject() + " " + conv.getTags());
client.conversations.resolve(conversationId);
client.conversations.reopen(conversationId);
```

:::warning O que se perde em `ConversationModel`
Chegam preenchidos apenas `id`, `status`, `subject`, `tags`, `metadata`, `contact` e `channel`. `getChannelId()`, `getContactId()`, `getAssignedAgentId()`, `getUnreadCount()`, `getLastMessage()` e todas as datas voltam `null`/`0`. `getPriority()` volta `null` quando a prioridade é `normal` (o enum do SDK só conhece `low`, `medium`, `high`, `urgent`).
:::

Listar, atribuir e ler o histórico pelo HTTP do SDK (a listagem é paginada por cursor: `limit` e `before`):

```java
List<JsonObject> abertas = http.get("/conversations",
        Map.of("status", "open", "channel_id", "CHANNEL_ID", "limit", "50"),
        new TypeToken<List<JsonObject>>() {}.getType());
http.post("/conversations/" + conversationId + "/assign", Map.of("user_id", "USER_ID"), JsonObject.class);
```

## Contatos

Funcionam `get(contactId)`, `create(input)` (para `name`, `email`, `phone` e `tags`) e `delete(contactId)`.

```java
import io.linktor.types.Contact;

Contact.ContactModel novo = client.contacts.create(Contact.CreateContactInput.builder()
        .name("Maria Souza").email("cliente@exemplo.com").phone("+5511999999999")
        .tags(List.of("vip")).build());
Contact.ContactModel lido = client.contacts.get(novo.getId());
client.contacts.delete(novo.getId());
```

:::warning Campos ignorados
No `create`, `customFields`, `avatar`, `identifiers` e `metadata` são enviados com nomes que o servidor não lê (ele espera `custom_fields` e `avatar_url`). Na leitura, `getCustomFields()`, `getAvatar()` e as datas voltam `null`. Para campos personalizados use `http.post("/contacts", Map.of("name", ..., "custom_fields", Map.of(...)), JsonObject.class)`.
:::

Listagem e busca: `http.get("/contacts", Map.of("page", "1", "page_size", "50", "search", "maria"), new TypeToken<List<JsonObject>>() {}.getType())`.

## Bots, fluxos e base de conhecimento

Nos três recursos só `get(String id)` e `delete(String id)` funcionam, e o `get` devolve o modelo parcial (`id`, `name`, `status`, `config`; em fluxos, `nodes` só com `id` e `type`). Criação e ações vão pelo HTTP do SDK:

```java
JsonObject bot = http.post("/bots", Map.of(
        "name", "Atendente", "type", "customer_service",
        "provider", "openai", "model", "gpt-4o-mini",
        "system_prompt", "Você atende clientes da loja."), JsonObject.class);
http.post("/bots/" + bot.get("id").getAsString() + "/activate", null, JsonObject.class);

JsonObject base = http.post("/knowledge-bases", Map.of("name", "FAQ Loja", "type", "faq"), JsonObject.class);
String kbId = base.get("id").getAsString();
http.post("/knowledge-bases/" + kbId + "/items",
        Map.of("question", "Qual o prazo de entrega?", "answer", "Até 5 dias úteis."), JsonObject.class);
```

## IA e analytics

Nenhum método de `client.ai` ou `client.analytics` é utilizável (veja [Limitações conhecidas](#limitações-conhecidas)). As rotas reais, pelo HTTP do SDK:

```java
JsonObject resposta = http.post("/ai/complete", Map.of(
        "provider", "openai", "model", "gpt-4o-mini",
        "messages", List.of(Map.of("role", "user", "content", "Resuma: pedido atrasado."))), JsonObject.class);

JsonObject visao = http.get("/analytics/overview", Map.of("period", "weekly"), JsonObject.class);
JsonObject porDia = http.get("/analytics/conversations",
        Map.of("start_date", "2026-11-01", "end_date", "2026-11-24"), JsonObject.class);
System.out.println(visao.get("total_conversations").getAsLong() + " " + porDia.getAsJsonArray("data").size());
```

As rotas de analytics são `overview`, `conversations`, `flows`, `escalations` e `channels`, com query `period`, `start_date`, `end_date`. Elas respondem sem o envelope padrão: `overview` devolve o objeto direto e as demais devolvem `{"data": [...]}`. `/ai/*` exige o scope `ai:use`; analytics, `analytics:read`.

## VRE

As rotas `/vre/*` só existem quando o serviço VRE está habilitado no servidor e respondem JSON sem envelope — o SDK trata isso corretamente.

```java
import io.linktor.types.VRE;

VRE.RenderResponse img = client.vre.render(VRE.RenderRequest.builder()
        .templateId(VRE.TemplateType.MENU_OPCOES)
        .data(Map.of("titulo", "Como posso ajudar?",
                "opcoes", List.of(Map.of("label", "Fazer pedido"), Map.of("label", "Falar com atendente"))))
        .channel(VRE.ChannelType.WHATSAPP)
        .build());
byte[] png = java.util.Base64.getDecoder().decode(img.getImageBase64());
```

`client.vre.listTemplates().getTemplates()` devolve os modelos disponíveis (`getId()`, `getName()`, `getDescription()`, `getSchema()`). `RenderRequest` aceita `templateId` (enum ou `String`), `data`, `channel`, `format`, `width`, `quality`, `scale` e `tenantId`. Não preencha `tenantId`: se for diferente do tenant da chave, o servidor responde 403. Os atalhos `renderMenu`, `renderProductCard`, `renderOrderStatus`, `renderProductList`, `renderConfirmation` e `renderPixPayment` montam o `data` e chamam `render`; passe `null` no primeiro parâmetro (`tenantId`).

:::warning `getRenderTimeMs()`
O servidor serializa esse campo em **nanossegundos**, e o SDK o lê como `int`. O valor não está em milissegundos, e uma renderização que leve mais de ~2,1 s faz o `render` lançar `JsonSyntaxException` mesmo com a imagem gerada.
:::

## Verificar a assinatura do webhook

:::warning O verificador do SDK não é compatível
`WebhookVerifier.verify`, `WebhookVerifier.constructEvent` e `LinktorClient.verifyWebhook` leem os headers certos (`X-Linktor-Signature`, `X-Linktor-Timestamp`) e conferem a tolerância de 300 s, mas calculam o HMAC **só sobre o corpo**. O servidor assina `timestamp + "." + corpo`. Resultado: toda entrega legítima é rejeitada. Verifique manualmente.
:::

Receptor mínimo em Spring Boot, lendo o corpo cru:

```java
import com.google.gson.JsonObject;
import com.google.gson.JsonParser;
import java.nio.charset.StandardCharsets;
import java.security.MessageDigest;
import java.time.Instant;
import java.util.HexFormat;
import javax.crypto.Mac;
import javax.crypto.spec.SecretKeySpec;
import org.springframework.http.ResponseEntity;
import org.springframework.web.bind.annotation.*;

@RestController
public class LinktorWebhookController {
    private static final long TOLERANCIA_SEGUNDOS = 300;
    private final String secret = System.getenv("LINKTOR_WEBHOOK_SECRET");

    @PostMapping("/webhooks/linktor")
    public ResponseEntity<Void> receber(
            @RequestBody byte[] corpo,
            @RequestHeader(value = "X-Linktor-Signature", required = false) String assinatura,
            @RequestHeader(value = "X-Linktor-Timestamp", required = false) String timestamp) throws Exception {

        if (!assinaturaValida(corpo, assinatura, timestamp)) {
            return ResponseEntity.status(401).build();
        }

        JsonObject evento = JsonParser.parseString(new String(corpo, StandardCharsets.UTF_8)).getAsJsonObject();
        String tipo = evento.get("type").getAsString();
        JsonObject data = evento.getAsJsonObject("data");

        if ("message.received".equals(tipo)) {
            JsonObject content = data.getAsJsonObject("message").getAsJsonObject("content");
            String texto = content.has("text") ? content.get("text").getAsString() : "";
            System.out.println(data.get("conversationId").getAsString() + ": " + texto);
        }
        return ResponseEntity.ok().build();
    }

    private boolean assinaturaValida(byte[] corpo, String assinatura, String timestamp) throws Exception {
        if (assinatura == null || timestamp == null || !timestamp.matches("\\d{1,12}")) {
            return false;
        }
        if (Math.abs(Instant.now().getEpochSecond() - Long.parseLong(timestamp)) > TOLERANCIA_SEGUNDOS) {
            return false;
        }

        Mac mac = Mac.getInstance("HmacSHA256");
        mac.init(new SecretKeySpec(secret.getBytes(StandardCharsets.UTF_8), "HmacSHA256"));
        mac.update((timestamp + ".").getBytes(StandardCharsets.UTF_8));
        String esperado = HexFormat.of().formatHex(mac.doFinal(corpo));

        return MessageDigest.isEqual(
                esperado.getBytes(StandardCharsets.UTF_8),
                assinatura.getBytes(StandardCharsets.UTF_8));
    }
}
```

A assinatura é hex minúsculo, sem prefixo `sha256=`. O segredo é o `credentials.webhook_secret` do canal. O envelope do evento é camelCase (`tenantId`, `conversationId`, `contentType`). O enum `Webhook.EventType` do SDK não corresponde à lista real de eventos; compare a string `type`. Eventos e payloads em [Receber eventos](../api/receber-eventos.md).

## Tempo real (WebSocket)

:::warning `client.webSocket()` não conecta
`LinktorWebSocket` abre `baseUrl + "/ws/conversations"`, autentica só por header e envia `subscribe`/`unsubscribe`. O servidor atende em `/api/v1/ws`, exige o **JWT de usuário** em `?token=` (API key sozinha não serve), não tem inscrição por conversa e emite `{"type", "payload"}` com tipos como `new_message` — o SDK espera `data` e `message.received`.
:::

Para consumir o tempo real em Java use `java.net.http.WebSocket`: obtenha um JWT com `POST /auth/login` (`data.access_token`) e conecte em `wss://linktor.suaempresa.com/api/v1/ws?token=<JWT>`, enviando o mesmo JWT também no header `Authorization: Bearer <JWT>` do handshake (`WebSocket.Builder.header`) — a rota passa pelo middleware de autenticação antes de o handler ler `?token=`. Cada mensagem de texto é um JSON `{"type", "payload"}`; em `new_message` o payload traz `conversation_id` e `message`. O servidor transmite todos os eventos do tenant. Tipos e payloads em [Tempo real](../api/tempo-real.md). Para integração servidor-a-servidor prefira o webhook.

## Tratamento de erros

Todas as exceções são `RuntimeException`, em `io.linktor.utils.LinktorException`, com `getStatusCode()`, `getErrorCode()` e `getRequestId()` (header `X-Request-ID` da resposta).

| Exceção | Quando |
|---|---|
| `LinktorException.ValidationException` | HTTP 400 |
| `LinktorException.AuthenticationException` | HTTP 401 |
| `LinktorException.AuthorizationException` | HTTP 403 (inclui API key sem o scope) |
| `LinktorException.NotFoundException` | HTTP 404 |
| `LinktorException.RateLimitException` | HTTP 429, esgotadas as tentativas; `getRetryAfter()` é sempre `60` |
| `LinktorException.ServerException` | HTTP 500/502/503/504, esgotadas as tentativas |
| `LinktorException.NetworkException` | falha de rede ou timeout; `getStatusCode()` é `0` |
| `LinktorException` (base) | demais status (409, 422…), com `getErrorCode()` `null` |
| `LinktorException.WebhookVerificationException` | lançada por `WebhookVerifier.constructEvent` |

```java
import io.linktor.utils.LinktorException;

try {
    client.channels.get("id-inexistente");
} catch (LinktorException.NotFoundException e) {
    System.err.println("Canal não existe (request " + e.getRequestId() + ")");
} catch (LinktorException.AuthenticationException | LinktorException.AuthorizationException e) {
    System.err.println("Chave inválida ou sem scope: HTTP " + e.getStatusCode());
} catch (LinktorException e) {
    System.err.println("HTTP " + e.getStatusCode() + " " + e.getMessage());
}
```

:::warning Mensagem e código do servidor se perdem
O SDK tenta ler `code`/`message` na raiz do corpo, mas o servidor os envia dentro de `error`. Por isso `getMessage()` é sempre `"Request failed"` e `getErrorCode()` é um valor fixo por status (`VALIDATION_ERROR`, `NOT_FOUND`…), não o código real. Para diagnosticar, guarde o `getRequestId()` e procure no log do servidor, ou repita a chamada por HTTP nativo. Além disso, respostas que o SDK não consegue desserializar (todas as listagens paginadas) lançam `com.google.gson.JsonSyntaxException`, que **não** é `LinktorException`. Formato dos erros em [Erros e paginação](../api/erros-e-paginacao.md).
:::

## Limitações conhecidas

| Método | Problema | Alternativa |
|---|---|---|
| `conversations.list`, `conversations.getMessages`, `contacts.list`, `contacts.findByEmail`, `contacts.findByPhone`, `contacts.getOrCreate`, `bots.list`, `flows.list`, `knowledgeBases.list` | Esperam `data` como objeto `{data, pagination}`; o servidor devolve `data` como lista e a paginação em `meta`. Lançam `JsonSyntaxException`. | `http.get(..., List<JsonObject>)` |
| `conversations.sendText`, `conversations.sendMessage` | Enviam `{"text", "type"}`; o servidor lê `content` e `content_type` e responde 400. | `http.post("/conversations/{id}/messages", ...)` |
| `conversations.assign` | Envia `agentId`; o servidor exige `user_id` (400). | `http.post(".../assign", Map.of("user_id", ...))` |
| `conversations.update`, `conversations.setPriority`, `contacts.update`, `bots.update`, `bots.activate`, `bots.deactivate`, `flows.update`, `flows.activate`, `flows.deactivate`, `knowledgeBases.update` | Usam `PATCH`; as rotas são `PUT` (ou `POST .../activate`). 404. | `http.put(...)` / `http.post(".../activate", null, ...)` |
| `conversations.unassign`, `addTags`, `removeTags`, `contacts.merge`, `flows.execute`, `getExecution`, `cancelExecution`, `duplicate` | Rotas inexistentes. | `PUT /conversations/{id}` com `tags`; `POST /flows/{id}/test` |
| `bots.create`, `flows.create`, `knowledgeBases.create` | Corpo sem campos obrigatórios (`provider`/`model`; `trigger`/`start_node_id`; `type`). 400. | `http.post` com o corpo real |
| `knowledgeBases.listDocuments`, `addDocument`, `getDocument`, `deleteDocument`, `query` | Rotas `/documents` e `/query` não existem. | `/knowledge-bases/{id}/items` e `/search` |
| `ai.agents.*`, `ai.completions.*`, `ai.embeddings.*` | Rotas inexistentes. | `POST /ai/complete` e demais rotas `/ai/*` |
| `analytics.getDashboard`, `getRealtime`, `getAgentPerformance` | Rotas inexistentes. | `GET /analytics/overview` |
| `analytics.getConversationStats`, `getChannelStats` | Enviam `startDate`/`endDate` (ignorados) e leem um formato que o servidor não devolve: retorno sempre zerado. | `http.get("/analytics/conversations", ...)` |
| `vre.renderAndSend` | Não envia `send_to`, `channel` e `channel_id`, obrigatórios (400). | `http.post("/vre/render-and-send", ...)` |
| `vre.preview` | Usa `POST`; a rota é `GET /vre/templates/{id}/preview`. | `http.get(...)` |
| `auth.*`, `WebhookVerifier.*`, `client.webSocket()` | Veja as seções acima. | HTTP nativo / verificação manual |

## Referência: método → rota HTTP

Caminhos relativos a `/api/v1`. Só os métodos que funcionam.

| Método | Rota |
|---|---|
| `channels.list()` | `GET /channels` |
| `channels.get(id)` | `GET /channels/{id}` |
| `channels.create(input)` | `POST /channels` |
| `channels.update(id, input)` | `PUT /channels/{id}` |
| `channels.delete(id)` | `DELETE /channels/{id}` |
| `channels.connect(id)` | `POST /channels/{id}/connect` |
| `channels.requestPairCode(id, phone)` | `POST /channels/{id}/pair` |
| `channels.disconnect(id)` | `POST /channels/{id}/disconnect` |
| `conversations.get(id)` | `GET /conversations/{id}` |
| `conversations.resolve(id)` | `POST /conversations/{id}/resolve` |
| `conversations.reopen(id)` | `POST /conversations/{id}/reopen` |
| `contacts.get(id)` | `GET /contacts/{id}` |
| `contacts.create(input)` | `POST /contacts` |
| `contacts.delete(id)` | `DELETE /contacts/{id}` |
| `bots.get(id)` / `bots.delete(id)` | `GET` / `DELETE /bots/{id}` |
| `flows.get(id)` / `flows.delete(id)` | `GET` / `DELETE /flows/{id}` |
| `knowledgeBases.get(id)` / `knowledgeBases.delete(id)` | `GET` / `DELETE /knowledge-bases/{id}` |
| `vre.render(request)` e atalhos `render*` | `POST /vre/render` |
| `vre.listTemplates()` | `GET /vre/templates` |
| `getHttpClient().get/post/put/delete(path, ...)` | qualquer rota; veja a [visão geral da API](../api/visao-geral.md) |

Comparação com os outros SDKs em [Visão geral dos SDKs](./visao-geral.md).
