# Smoke real do canal sandbox WhatsApp contra a Meta — roteiro executável

- **Objetivo:** provar ponta a ponta, contra o WhatsApp **real**, que a barreira de sandbox funciona (INV-016/017/018). Até este smoke, a capacidade inteira foi validada só contra mock e banco.
- **Executor:** operador humano com acesso ao painel da Meta e a **dois aparelhos** com números reais.
- **O item 5 é o teste que justifica a fase inteira.** A prova dele é a **ausência de mensagem no aparelho B** — a resposta da API não basta.
- Script auxiliar de verificação: `scripts/sandbox-smoke-check.sh` (roda as checagens de banco/métrica de um envio pelo `message_id`).

## Pré-requisitos

**Na Meta (App Dashboard → WhatsApp → API Setup):**

- [ ] Número de teste da Meta ativo; anote `PHONE_NUMBER_ID` e `WABA_ID`.
- [ ] **Dois números reais VERIFICADOS** como destinatários do número de teste (a Meta só entrega a até 5 números verificados):
  - **Número A** — entrará na allowlist do Linktor.
  - **Número B** — ficará **fora** da allowlist. ⚠️ **B precisa estar verificado na Meta.** Se B não estiver verificado, a própria Meta recusaria a entrega e o teste 5 passaria mesmo com a nossa guarda quebrada — falso positivo clássico. B verificado garante que, se a guarda falhar, a mensagem **chega**, e o teste acusa.
- [ ] Template `hello_world` (vem provisionado no WABA de teste) visível em Message Templates.
- [ ] Access token válido (token temporário de 24h do painel serve para o smoke).
- [ ] Para o cenário 7 (inbound): webhook do app apontando para o backend exposto publicamente (ex.: `ngrok http 8081`), URL `https://<túnel>/api/v1/webhooks/whatsapp/<CHANNEL_ID>`, com `verify_token` e `app_secret` configurados no canal. O inspector do ngrok (`http://127.0.0.1:4040`) será usado para replay no teste de dedup.

**No ambiente Linktor (descartável — não use produção):**

- [ ] Backend rodando (`scripts/run-backend.sh`), Postgres/Redis/NATS ativos, migrations aplicadas (`00012`–`00014` presentes: `SELECT version_id FROM goose_db_version ORDER BY version_id DESC LIMIT 1;` ≥ 14).
- [ ] `LINKTOR_WA_WINDOW_ENFORCEMENT` e `LINKTOR_WA_TEMPLATE_ENFORCEMENT` **não definidos** (default `dry_run`) até o cenário 9.
- [ ] Usuário com papel `admin` ou `owner` no tenant de teste.
- [ ] Ferramentas: `curl`, `jq`, `psql`, `nats` CLI (opcional, cenário 4), `tcpdump` ou `mitmproxy` (opcional, prova de rede do cenário 5).

**Variáveis de sessão** (ajuste e cole no shell):

```bash
export BASE="http://localhost:8081/api/v1"
export METRICS="http://localhost:8081/metrics"
export PSQL="psql -h localhost -p 5432 -U linktor -d linktor -tA"
export PHONE_NUMBER_ID="<phone_number_id do número de teste>"
export META_TOKEN="<access token>"
export NUM_A="+55DDDXXXXXXXX"   # verificado na Meta; ENTRARÁ na allowlist
export NUM_B="+55DDDYYYYYYYY"   # verificado na Meta; FORA da allowlist

# Login (admin) — captura o JWT
export TOKEN=$(curl -s -X POST $BASE/auth/login \
  -H 'Content-Type: application/json' \
  -d '{"email":"<admin@tenant>","password":"<senha>"}' | jq -r '.data.token // .token')
export AUTH="Authorization: Bearer $TOKEN"
```

---

## Cenário 1 — Criação do canal sandbox: validações rejeitam antes de persistir

**1a. Sem `credential_environment` → rejeitado:**

```bash
curl -s -X POST $BASE/channels -H "$AUTH" -H 'Content-Type: application/json' -d '{
  "type": "whatsapp_official", "name": "smoke-sandbox", "environment": "sandbox",
  "config": {"phone_number_id": "'$PHONE_NUMBER_ID'", "sandbox_test_phone_number_ids": "'$PHONE_NUMBER_ID'"},
  "credentials": {"access_token": "'$META_TOKEN'"}
}' | jq .
```

- **Esperado:** HTTP 400/422 com mensagem citando `credential_environment`. **Falso positivo:** erro genérico de binding — a mensagem deve ser a da validação de ambiente, não "Invalid request body".
- Confirmar que **nada** persistiu: `$PSQL -c "SELECT count(*) FROM channels WHERE name='smoke-sandbox'"` → `0`.

**1b. `phone_number_id` fora da lista declarada → rejeitado:**

```bash
curl -s -X POST $BASE/channels -H "$AUTH" -H 'Content-Type: application/json' -d '{
  "type": "whatsapp_official", "name": "smoke-sandbox", "environment": "sandbox",
  "config": {"phone_number_id": "'$PHONE_NUMBER_ID'", "sandbox_test_phone_number_ids": "000000000"},
  "credentials": {"access_token": "'$META_TOKEN'", "credential_environment": "sandbox"}
}' | jq .
```

- **Esperado:** validação citando `sandbox_test_phone_number_ids`; count no banco continua `0`.

**1c. Criação válida:**

```bash
export CHANNEL_ID=$(curl -s -X POST $BASE/channels -H "$AUTH" -H 'Content-Type: application/json' -d '{
  "type": "whatsapp_official", "name": "smoke-sandbox", "environment": "sandbox",
  "config": {"phone_number_id": "'$PHONE_NUMBER_ID'", "sandbox_test_phone_number_ids": "'$PHONE_NUMBER_ID'"},
  "credentials": {"access_token": "'$META_TOKEN'", "credential_environment": "sandbox"}
}' | jq -r '.data.id')
echo $CHANNEL_ID
$PSQL -c "SELECT environment FROM channels WHERE id='$CHANNEL_ID'"   # => sandbox
```

Habilite o canal se necessário (`connection_status` não importa para o sender stateless; `enabled=true` é o default).

## Cenário 2 — Environment é imutável (confirmar no banco, não só na resposta)

```bash
curl -s -X PUT $BASE/channels/$CHANNEL_ID -H "$AUTH" -H 'Content-Type: application/json' \
  -d '{"type":"whatsapp_official","name":"smoke-sandbox","environment":"production"}' | jq .
$PSQL -c "SELECT environment FROM channels WHERE id='$CHANNEL_ID'"
```

- **Esperado:** HTTP 400 com "environment is immutable"; a query devolve **`sandbox`**. **Falso positivo:** resposta de erro mas coluna alterada — por isso a checagem é no banco.

## Cenário 3 — Allowlist: normalização E.164 + auditoria

```bash
# Formato deliberadamente "sujo" para provar a normalização:
export ENTRY_ID=$(curl -s -X POST $BASE/sandbox/allowlist -H "$AUTH" -H 'Content-Type: application/json' \
  -d '{"recipient": "'"$(echo $NUM_A | sed 's/+55/+55 /')"'", "note": "smoke operador"}' | jq -r '.data.id')
$PSQL -c "SELECT recipient FROM sandbox_allowlist_entries WHERE id='$ENTRY_ID'"
$PSQL -c "SELECT action, actor_email, changes->>'recipient' FROM audit_logs WHERE action='sandbox_allowlist.add' ORDER BY created_at DESC LIMIT 1"
```

- **Esperado:** `recipient` gravado **exatamente** como `+55DDDXXXXXXXX` (sem espaços); linha de auditoria com autor e valor.

## Cenário 4 — Envio para número NA allowlist: a mensagem CHEGA

Preparação (contato + conversa para o número A):

```bash
export CONTACT_A=$(curl -s -X POST $BASE/contacts -H "$AUTH" -H 'Content-Type: application/json' \
  -d '{"name":"Smoke A","phone":"'$NUM_A'"}' | jq -r '.data.id')
export CONV_A=$(curl -s -X POST $BASE/conversations -H "$AUTH" -H 'Content-Type: application/json' \
  -d '{"contact_id":"'$CONTACT_A'","channel_id":"'$CHANNEL_ID'"}' | jq -r '.data.id')
```

Observadores (terminais separados, ANTES do envio):

```bash
# Envelope NATS de saída:
nats sub 'linktor.messages.outbound.whatsapp_official'
# Webhook de saída (se channel.webhook_url configurado): receptor local
python3 -m http.server 9099   # e configure webhook_url do canal para http://localhost:9099/hook
```

Envio (dentro da janela de 24h da Meta, o número de teste aceita free-form para verificados; se recusar, responda antes do aparelho A para abrir a janela):

```bash
export MSG_A=$(curl -s -X POST $BASE/conversations/$CONV_A/messages -H "$AUTH" -H 'Content-Type: application/json' \
  -d '{"sender_type":"user","content":"smoke sandbox: allowlist OK"}' | jq -r '.data.id')
scripts/sandbox-smoke-check.sh $MSG_A
```

- **Prova principal:** a mensagem **aparece no aparelho A**. Status `sent` na API **não** é a prova.
- `conversations.environment` = `sandbox`: `$PSQL -c "SELECT environment FROM conversations WHERE id='$CONV_A'"`.
- Envelope NATS capturado contém `"environment":"sandbox"`.
- Payload do webhook de saída contém `"environment":"sandbox"` no topo do envelope.

## Cenário 5 — Envio para número FORA da allowlist: NADA chega (o teste central)

> B está **verificado na Meta** e **fora da allowlist**. Se a guarda falhar, a mensagem chega no aparelho B.

```bash
# (Opcional, prova de rede definitiva — inicie ANTES do envio:)
sudo tcpdump -n host graph.facebook.com and port 443 &

export CONTACT_B=$(curl -s -X POST $BASE/contacts -H "$AUTH" -H 'Content-Type: application/json' \
  -d '{"name":"Smoke B","phone":"'$NUM_B'"}' | jq -r '.data.id')
export CONV_B=$(curl -s -X POST $BASE/conversations -H "$AUTH" -H 'Content-Type: application/json' \
  -d '{"contact_id":"'$CONTACT_B'","channel_id":"'$CHANNEL_ID'"}' | jq -r '.data.id')

curl -s -X POST $BASE/conversations/$CONV_B/messages -H "$AUTH" -H 'Content-Type: application/json' \
  -d '{"sender_type":"user","content":"NUNCA deve chegar"}' | jq .
```

- **Esperado na API:** erro síncrono de validação com número **mascarado** (`+55*********XX`) — fail-fast da API.
- **Aparelho B: nenhuma mensagem, aguardar ≥2 min.** Esta é a prova.
- O fail-fast impede até a criação da linha; para exercitar a **guarda autoritativa do funil** (o caminho que campanha/bot usam), injete direto no NATS o mesmo envio e verifique o bloqueio no worker:

```bash
nats pub linktor.messages.outbound.whatsapp_official '{
  "id":"'$(uuidgen | tr A-Z a-z)'","tenant_id":"<TENANT_ID>","channel_id":"'$CHANNEL_ID'",
  "channel_type":"whatsapp_official","conversation_id":"'$CONV_B'","contact_id":"'$CONTACT_B'",
  "recipient_id":"'$NUM_B'","content_type":"text","content":"NUNCA deve chegar (funil)",
  "timestamp":"'$(date -u +%Y-%m-%dT%H:%M:%SZ)'"}'
```

Verificações (o `message_id` é o `id` publicado acima):

```bash
$PSQL -c "SELECT level, message, metadata->>'blocked_by' FROM message_logs WHERE metadata->>'message_id'='<id>'"
curl -s $METRICS | grep 'linktor_outbound_guard_blocked_total'
```

- **Esperado:** log `Envio bloqueado por guarda (allowlist)` com `blocked_by=allowlist`; métrica `linktor_outbound_guard_blocked_total{channel_type="whatsapp_official",mode="enforce",reason="allowlist"}` incrementada (compare o valor de antes).
- **Nenhuma chamada saiu à Graph API:** (a) o `tcpdump` não mostra tráfego novo para `graph.facebook.com` no intervalo do teste (prova de rede, não inferência); (b) `message_logs` não tem `Mensagem entregue ao canal` para esse id; (c) `linktor_outbound_messages_total{result="sent"}` inalterado. *(Nota: o backend hoje não loga requisições HTTP de saída à Graph API — lacuna registrada; o tcpdump é o substituto direto.)*
- **Sem retry:** repetir a consulta de métrica após 1 min — o contador de bloqueio não cresce sozinho (NATS não reentrega bloqueio permanente).

## Cenário 6 — Remoção da allowlist vale imediatamente (sem esperar cache de 5 min)

```bash
curl -s -X DELETE $BASE/sandbox/allowlist/$ENTRY_ID -H "$AUTH" -w '%{http_code}\n'
# IMEDIATAMENTE (dentro dos 5 min do cache de senders do Resolver):
curl -s -X POST $BASE/conversations/$CONV_A/messages -H "$AUTH" -H 'Content-Type: application/json' \
  -d '{"sender_type":"user","content":"não deve sair após remoção"}' | jq .
```

- **Esperado:** bloqueado **no primeiro envio** após a remoção; nada chega no aparelho A. O ponto do teste: o sender decorado continua cacheado, mas a allowlist é consultada por envio.
- Recoloque A na allowlist antes de prosseguir (repita o cenário 3).

## Cenário 7 — Inbound real + dedup

1. **Responda do aparelho A** para o número de teste.
2. Verifique a chegada e a marcação:

```bash
$PSQL -c "SELECT m.id, c.environment FROM messages m JOIN conversations c ON c.id=m.conversation_id WHERE c.id='$CONV_A' AND m.sender_type='contact' ORDER BY m.created_at DESC LIMIT 1"
```

- **Esperado:** mensagem presente; `environment=sandbox`.

3. **Dedup:** no inspector do ngrok (`http://127.0.0.1:4040`), localize o POST do webhook e clique **Replay** (corpo e assinatura idênticos). Depois:

```bash
$PSQL -c "SELECT count(*) FROM messages WHERE conversation_id='$CONV_A' AND sender_type='contact'"
```

- **Esperado:** o count **não** muda (replay respondido como duplicado pelo middleware Redis e/ou pelo unique de `external_id`). **Falso positivo:** replay com corpo alterado geraria assinatura inválida (401) — isso testaria a assinatura, não o dedup; use o Replay exato.

## Cenário 8 — Janela de 24h expirada em dry_run: a mensagem SAI

Force a expiração da janela na conversa A (ambiente descartável — manipulação aceitável):

```bash
$PSQL -c "UPDATE messages SET created_at = created_at - interval '25 hours' WHERE conversation_id='$CONV_A' AND sender_type='contact'"
BEFORE=$(curl -s $METRICS | grep 'guard_blocked_total.*window_24h.*dry_run' || echo 0)
curl -s -X POST $BASE/conversations/$CONV_A/messages -H "$AUTH" -H 'Content-Type: application/json' \
  -d '{"sender_type":"user","content":"free-form fora da janela (dry-run)"}' | jq -r '.data.id'
curl -s $METRICS | grep 'guard_blocked_total.*window_24h'
```

- **Esperado:** a mensagem **sai** (chega no aparelho A **ou** a Meta a recusa por janela — ambos provam que o Linktor não bloqueou); métrica `reason="window_24h",mode="dry_run"` incrementou; log `[dry-run]` no stdout do backend.
- ⚠️ A allowlist continua valendo: A precisa estar na allowlist para este teste.

## Cenário 9 — Janela de 24h com enforcement (ambiente descartável)

Reinicie o backend com `LINKTOR_WA_WINDOW_ENFORCEMENT=enforce` e repita o envio do cenário 8:

- **Esperado:** API aceita (o fail-fast síncrono não avalia janela), mas o worker bloqueia **antes** da chamada à Meta: status `failed`, `message_logs` com `blocked_by=window_24h` e mensagem citando "24h window" — **distinguível** do bloqueio de allowlist (`blocked_by=allowlist`); nada chega no aparelho A; métrica `reason="window_24h",mode="enforce"`.
- Ao final, **volte para dry_run** (remova a env var e reinicie) — rollback por configuração, sem deploy.

## Cenário 10 — Canal production preexistente: comportamento idêntico

Use um canal `whatsapp_official` de **produção** já existente no ambiente (ou crie um sem `environment`, que nasce `production`), com uma conversa ativa:

```bash
$PSQL -c "SELECT environment FROM channels WHERE id='<PROD_CHANNEL_ID>'"   # => production
# envie uma mensagem normal pela API para uma conversa desse canal
```

- **Esperado:** entrega normal ("Mensagem entregue ao canal" em `message_logs`); **nenhum** incremento em `linktor_outbound_guard_blocked_total` para o envio; a mensagem chega no destinatário. Envelope de webhook de saída **sem** a chave `environment` (omitempty) — o wire format de produção não mudou.

---

## Reversão ao final

```bash
curl -s -X DELETE $BASE/sandbox/allowlist/<ids> -H "$AUTH"        # limpar allowlist
curl -s -X DELETE $BASE/channels/$CHANNEL_ID -H "$AUTH"           # remove canal (cascade: conversas/mensagens de smoke)
unset LINKTOR_WA_WINDOW_ENFORCEMENT LINKTOR_WA_TEMPLATE_ENFORCEMENT  # e reinicie o backend
```

- Revogue o access token temporário no painel da Meta; desligue o túnel (`ngrok`); remova o webhook do app se ele apontava para este ambiente.
- Registre no relatório do smoke: data, commit do backend, resultado por cenário e qualquer divergência.
