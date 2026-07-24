# Gerenciar canais via token (API key) — API HTTP e SDK

Guia para uma aplicação externa criar, editar, remover e conectar (QR) canais de um tenant usando uma **API key** do Linktor. Cobre a API HTTP crua (completa e verificada ponta a ponta), os **scopes** aplicados à chave e os SDKs oficiais das 7 linguagens (com paridade de gestão de canal).

> **Escopo por tenant:** a API key é *tenant-scoped por construção*. Toda operação feita com a key fica automaticamente restrita ao tenant dono da key — não se passa `tenant_id` em lugar nenhum. Uma key do tenant A recebe **404** ao tentar acessar recurso do tenant B.

---

## 1. Obter o token (API key)

A API key é criada por um **administrador do tenant** (papel `admin` ou `owner`). Duas formas:

**A) Pelo console admin** — *Settings → API Keys → Generate New Key*. A chave crua (`lk_...`) é exibida **uma única vez**; copie e guarde. Depois, só o prefixo é recuperável.

**B) Pela API** (autenticado como admin, via sessão/JWT):

```bash
curl -X POST https://<host>/api/v1/api-keys \
  -H "Authorization: Bearer <jwt-admin>" \
  -H 'Content-Type: application/json' \
  -d '{"name":"App externa - canais","scopes":["*"]}'
```

Resposta (a chave crua vem em `data.key`, só nesta resposta):

```json
{"data":{"id":"...","tenant_id":"<tenant do admin>","name":"App externa - canais",
         "key_prefix":"lk_7038a6183","scopes":["*"],"key":"lk_7038a618382e1..."}}
```

O `tenant_id` da key é o do admin que a criou — é assim que "por tenant" acontece.

> **Limitações conhecidas (hoje):**
> - **Scopes não são aplicados.** O campo `scopes` é persistido mas ainda não é verificado — uma key tem acesso total às rotas **não-admin** do seu tenant, independente do que estiver em `scopes`. (Enforcement de scopes está planejado.)
> - **Gestão de keys é admin-only.** Uma key (papel `api`) **não** cria/lista/remove outras keys, nem alcança rotas admin (usuários, papéis, auditoria). Ela alcança canais, mensagens, contatos, conversas, etc.

---

## 2. Autenticação nas chamadas

Envie a chave no header **`X-API-Key`** em toda requisição:

```
X-API-Key: lk_7038a618382e1...
```

(Alternativamente, um usuário humano usa `Authorization: Bearer <jwt>`. Para server-to-server, use `X-API-Key`.)

Base URL da API: `https://<host>/api/v1`.

### Scopes (permissões da chave)

Cada API key carrega **scopes** no formato `recurso:ação`, **agora aplicados** pelo backend:

- `channels:read` — `GET /channels…` · `channels:write` — criar/editar/remover/`connect`/`pair`/`disconnect`.
- `messages:send` — `POST /conversations/:id/messages`.
- `contacts:read|write`, `conversations:read|write` — vocabulário reservado (ainda não aplicado nessas rotas).
- Curingas: `*` (tudo) e `recurso:*`; `recurso:write` implica `recurso:read`.
- Chaves criadas **antes** dos scopes existirem operam como `["*"]` (sem quebra).

Uma chamada com key sem o scope necessário recebe **403** (`API key is missing the required scope: …`). Escolha os scopes ao criar a chave no console (Configurações → API Keys) ou no `POST /api-keys` (campo `scopes`). Requisições humanas (JWT) não usam scopes — são regidas por papel (role).

---

## 3. Via API HTTP (completo)

Todas as rotas abaixo aceitam `X-API-Key` e são escopadas ao tenant da key. Exemplos verificados ponta a ponta.

### Criar canal

```bash
curl -X POST https://<host>/api/v1/channels \
  -H "X-API-Key: $KEY" -H 'Content-Type: application/json' \
  -d '{
        "type": "telegram",
        "name": "Meu canal",
        "config": { "phone_number_id": "..." },
        "credentials": { "bot_token": "..." }
      }'
```

- `type`: `whatsapp` (não oficial), `whatsapp_official`, `telegram`, `webchat`, `sms`, `instagram`, `facebook`, `rcs`, `email`, `voice`, `teams`, `slack`, `mattermost`, `direto`.
- `config`: parâmetros não-secretos (ex.: `phone_number_id`, `verify_token`, `api_version`).
- `credentials`: segredos (ex.: `access_token`, `bot_token`, `app_secret`) — cifrados at-rest, nunca retornados.
- Resposta: o canal criado, com `id` e `tenant_id` (o do tenant da key).

### Obter / listar

```bash
curl https://<host>/api/v1/channels/$CHANNEL_ID -H "X-API-Key: $KEY"
curl https://<host>/api/v1/channels            -H "X-API-Key: $KEY"
```

Segredos em `config` vêm redigidos (`__redacted__`); `credentials` nunca são serializados.

### Editar canal

```bash
curl -X PUT https://<host>/api/v1/channels/$CHANNEL_ID \
  -H "X-API-Key: $KEY" -H 'Content-Type: application/json' \
  -d '{ "type": "telegram", "name": "Nome novo", "config": {} }'
```

Em `credentials`, um valor vazio ou `__redacted__` significa "manter o segredo armazenado" (não sobrescreve).

### Remover canal

```bash
curl -X DELETE https://<host>/api/v1/channels/$CHANNEL_ID -H "X-API-Key: $KEY"
```

### Testar credenciais (sem criar)

```bash
curl -X POST https://<host>/api/v1/channels/test \
  -H "X-API-Key: $KEY" -H 'Content-Type: application/json' \
  -d '{ "type": "telegram", "credentials": { "bot_token": "..." } }'
```

### Conectar e ativar por **QR code** (WhatsApp não oficial)

O fluxo é **connect síncrono + polling do status**:

```bash
# 1) Dispara a conexão; a resposta traz o QR (conteúdo a renderizar como imagem)
curl -X POST https://<host>/api/v1/channels/$CHANNEL_ID/connect -H "X-API-Key: $KEY"
# → { "data": { "qr_code": "<conteúdo>", "expires_in": 60, "connection_status": "connecting" } }
```

1. Renderize `qr_code` como imagem QR e mostre ao usuário escanear no app do WhatsApp.
2. Faça **poll** em `GET /channels/:id` — `connection_status` vai de `connecting` → `connected` quando escaneado.
3. QR expirou? Chame `POST /connect` de novo para gerar um novo.
4. **Conta com passkey ("Shortcake"):** o connect devolve `passkey_required: true` + `passkey_challenge`; conclua com `POST /channels/:id/passkey/response`.

**Alternativa — código de pareamento** (em vez de QR):

```bash
curl -X POST https://<host>/api/v1/channels/$CHANNEL_ID/pair \
  -H "X-API-Key: $KEY" -H 'Content-Type: application/json' \
  -d '{ "phone_number": "+5544999999999" }'
```

### Habilitar / desabilitar / desconectar

```bash
curl -X PUT  https://<host>/api/v1/channels/$CHANNEL_ID/enabled -H "X-API-Key: $KEY" -H 'Content-Type: application/json' -d '{"enabled": true}'
curl -X POST https://<host>/api/v1/channels/$CHANNEL_ID/disconnect -H "X-API-Key: $KEY"
```

### Semântica de erros

- **404** — o canal não existe **ou** não pertence ao seu tenant (o isolamento não distingue os dois, de propósito, para não vazar existência).
- **401** — API key ausente/inválida.
- **403** — rota admin-only (ex.: `POST /channels/reencrypt`) — uma API key (papel `api`) não alcança.

---

## 4. Via SDK

Há SDKs oficiais em `sdks/` para 7 linguagens. Todos autenticam por API key. Os SDKs foram **alinhados ao contrato real do backend** (wire snake_case) e têm **paridade** de gestão de canal:

| Operação | TS | Python | Go | Java | PHP | Rust | .NET |
|---|:--:|:--:|:--:|:--:|:--:|:--:|:--:|
| `list` / `get` / `create` | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ |
| `update` (PUT) | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ |
| `delete` | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ |
| `connect` / `disconnect` | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ |
| **QR no `connect` (`ConnectResult`)** | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ |
| `requestPairCode` (pareamento) | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ |
| `credentials` no create/update | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ |

Pontos-chave do contrato (todos os SDKs agora seguem):

- **`connect()` retorna `ConnectResult`**, não `Channel`: campos `qr_code`, `expires_in` (segundos), `pair_code`, `passkey_required`, `passkey_challenge`, além do `channel`. Para ativar por QR: renderize `qr_code` e chame `connect()` de novo para renovar quando `expires_in` estourar.
- **`update()` usa PUT** (não PATCH) — reusa o mesmo corpo do create.
- **Segredos vão em `credentials`** (write-only, nunca retornado); configurações não-secretas em `config`. Ambos são mapas string→string.
- **Modelo `Channel` em snake_case**: `tenant_id`, `connection_status` (`disconnected|connecting|connected|error`), `enabled`, `config`, `webhook_url`, `created_at`, campos de coexistência. Não há campo `status`.

> **Nota (fora do escopo de canais):** o alinhamento snake_case foi aplicado ao **recurso de canal**. Outros recursos dos SDKs (contatos, conversas…) ainda modelam campos em camelCase e não foram reconciliados com o wire — tratar como dívida separada.

### TypeScript

```ts
import { createClient } from '@linktor/sdk'; // sdks/typescript

const linktor = createClient({
  baseURL: 'https://<host>/api/v1',
  apiKey: 'lk_7038a618382e1...',   // → header X-API-Key
});

const ch = await linktor.channels.create({
  name: 'Meu canal',
  type: 'whatsapp_unofficial',
  config: { phone_number_id: '...' },
  credentials: { access_token: 'segredo' },   // write-only
});

await linktor.channels.update(ch.id, { name: 'Nome novo' });  // PUT

// Ativar por QR
const res = await linktor.channels.connect(ch.id);
if (res.qr_code) render(res.qr_code); // expira em res.expires_in s → chame connect() de novo
// Alternativa: pareamento por número
const pair = await linktor.channels.requestPairCode(ch.id, '+5511999999999');

await linktor.channels.delete(ch.id);
```

### Python

```python
from linktor import LinktorClient  # sdks/python

linktor = LinktorClient(
    base_url="https://<host>/api/v1",
    api_key="lk_7038a618382e1...",   # → header X-API-Key
)

ch = linktor.channels.create(
    name="Meu canal", type="whatsapp_unofficial",
    config={"phone_number_id": "..."},
    credentials={"access_token": "segredo"},   # write-only
)
linktor.channels.update(ch.id, name="Nome novo")  # PUT

res = linktor.channels.connect(ch.id)
if res.qr_code:
    render(res.qr_code)  # expira em res.expires_in s → connect() de novo
pair = linktor.channels.request_pair_code(ch.id, "+5511999999999")

linktor.channels.delete(ch.id)
```

### Go

```go
import "github.com/msgfy/linktor/sdks/go" // pacote linktor

client := linktor.NewClient(
    linktor.WithBaseURL("https://<host>/api/v1"),
    linktor.WithAPIKey("lk_7038a618382e1..."), // → header X-API-Key
)

ch, _ := client.Channels.Create(ctx, &types.CreateChannelInput{
    Name: "Meu canal", Type: "whatsapp_unofficial",
    Config:      map[string]string{"phone_number_id": "..."},
    Credentials: map[string]string{"access_token": "segredo"}, // write-only
})
client.Channels.Update(ctx, ch.ID, &types.UpdateChannelInput{Name: "Nome novo"})

res, _ := client.Channels.Connect(ctx, ch.ID)
if res.QRCode != "" { render(res.QRCode) } // expira em res.ExpiresIn s → connect() de novo
pair, _ := client.Channels.RequestPairCode(ctx, ch.ID, "+5511999999999")

client.Channels.Delete(ctx, ch.ID)
```

> Java, PHP, Rust e .NET expõem os mesmos métodos (`connect`/`requestPairCode` → `ConnectResult`, `update` via PUT, `credentials` no create/update). Veja os READMEs em `sdks/<lang>/`.

---

## 5. Provar o isolamento por tenant (opcional)

Com a key do tenant A, tente acessar um canal do tenant B — deve retornar 404 em GET/PUT/DELETE, e o canal do B fica intacto:

```bash
curl -o /dev/null -w '%{http_code}\n' https://<host>/api/v1/channels/<id-do-tenant-B> -H "X-API-Key: $KEY_A"
# → 404
```

---

## 6. Resumo do estado / roadmap

Fechado nesta entrega:

- ✅ **Enforcement de scopes** — `channels:*` e `messages:send` aplicados; keys antigas viram `["*"]`.
- ✅ **Paridade de canal nos 7 SDKs** — `update`(PUT)/`delete`, `ConnectResult` com `qr_code`/`expires_in`/`pair_code`, `requestPairCode`, `credentials` no create/update; modelos `Channel` alinhados ao wire snake_case.
- ✅ **Console** — seleção de scopes ao criar a API key.

Dívidas conhecidas:

- **Demais recursos dos SDKs** (contatos, conversas…) ainda em camelCase, não reconciliados com o wire — alinhar em trabalho separado.
- **Scopes de `contacts`/`conversations`** definidos mas ainda não aplicados nessas rotas (rollout começou por canais + mensagens).
- **Criação de tenant** não é self-service (feita por seed/provisionamento) — a API key opera dentro de um tenant já existente.
