# `linktor-channel-v1` — referral do anúncio e código do erro (campos aditivos, 2026-10-23)

**Branch:** `feat/referral-ctwa-webhook` (sobre `origin/main` `42aa4dc`). **Natureza:** mudança **aditiva e
retrocompatível** do contrato de webhook de saída (`internal/infrastructure/webhook/envelope.go`, espelhado em
`sdks/go/webhook.go`). Consumidores atuais (DeskLenz, VendaX) seguem lendo o mesmo envelope: os campos novos usam
`omitempty` e só aparecem quando há o dado — mensagem sem anúncio e status sem código saem byte a byte como antes.

## 1. `message.received` → `data.message.referral` (novo, opcional)

Presente só quando a conversa foi aberta por um anúncio (WhatsApp Click-to-WhatsApp, adaptador `whatsapp_official`).

```json
"referral": {
  "sourceType": "ad",
  "sourceId": "120200000000000",
  "sourceUrl": "https://fb.me/...",
  "headline": "…",
  "body": "…",
  "ctwaClid": "ARAk…",
  "mediaType": "image",
  "mediaUrl": "https://…"
}
```

| Campo | Origem na Meta (`messages[].referral`) | Observação |
|---|---|---|
| `sourceType` | `source_type` | `ad` ou `post` |
| `sourceId` | `source_id` | id do anúncio/post |
| `sourceUrl` | `source_url` | **novo** — antes não era extraído |
| `headline`, `body` | `headline`, `body` | |
| `ctwaClid` | `ctwa_clid` | **novo** — id do clique (atribuição/conversões); faltava em `ReferralContent` |
| `mediaType` | `media_type` | **novo** — `image` ou `video` |
| `mediaUrl` | `image_url`, senão `video_url` | **novo** — criativo do anúncio |

Todos opcionais. As chaves `referral_*` continuam na metadata da mensagem no Linktor (agora com `referral_source_url`,
`referral_ctwa_clid`, `referral_media_type`, `referral_image_url`, `referral_video_url`) e passam a viajar no evento
durável `message.received` (outbox) — antes paravam na mensagem e o webhook não as via.

## 2. `message.{sent,delivered,read,failed}` → `data.errorCode` (novo, opcional)

Código do erro do provedor ao lado do `error` (texto), quando o provedor manda — no `whatsapp_official`,
`statuses[].errors[0].code` da Meta (ex.: `"131047"`, fora da janela de 24 h). Texto, para não amarrar o contrato ao
tipo numérico de um provedor. Ausente nos demais status e quando o provedor não manda código.

Caminho: `ParsedStatus.ErrorCode` → `plugin.StatusCallback.ErrorCode` → `nats.StatusUpdate.ErrorCode`
(`error_code`) → evento `message.*` → `StatusData.ErrorCode`.

## 3. SDK Go

`sdks/go/webhook.go`: `WebhookEvent.Environment` (o campo já emitido pelo servidor), `WebhookEvent.DecodeData`, e os
tipos `ReferralData` e `StatusData` (com `ErrorCode`).

## 4. Testes

- `internal/adapters/whatsapp_official/webhook_test.go`: `TestExtractMessages_ReferralCtwaCompleto` (JSON da Meta fictício) e
  `TestExtractStatuses` com `wantErrCode`.
- `internal/application/usecase/receive_message_test.go`: `TestReceiveMessage_ReferralReachesTheDurableEvent`.
- `internal/infrastructure/webhook/referral_test.go`: JSON da Meta → adaptador → evento → envelope com `referral`
  (`ctwaClid` no fio); referral ausente fora do fio; `errorCode` em `message.failed` e ausente nos outros.
- `sdks/go/webhook_payloads_test.go`.
