# Status de e-mail com o evento original do provedor — `linktor-channel-v1` (aditivo)

**Origem:** VCR-324 (VendaX CRM). **Compatibilidade:** só campos opcionais com `omitempty`. `status` mantém os
valores e as regras de sempre; quem não lê os campos novos (DeskLenz) recebe o mesmo payload.

## Por quê

O status de e-mail junta eventos diferentes: abertura e clique viram `read`; bounce, falha e reclamação de spam
viram `failed`. O consumidor precisa do evento original para suprimir endereço que deu bounce e para registrar como
oposição quem reclamou de spam. Agora o evento original vai junto.

## Campos novos em `data` de `message.{sent,delivered,read,failed}`

| Campo | Tipo | Quando vem | Valores |
|---|---|---|---|
| `providerEvent` | string | todo status de e-mail que veio de um evento do provedor (ausente nos outros canais e no "na fila") | `sent`, `delivered`, `opened`, `clicked`, `unsubscribed`, `bounced`, `spam_report`, `failed` |
| `bounceType` | string | `bounced`, quando o provedor informa o tipo | `hard`, `soft` |
| `url` | string | `clicked`, quando o provedor manda o link | o link clicado |

**`status` para cada evento original:**

| `providerEvent` | `status` |
|---|---|
| `sent` | `sent` |
| `delivered` | `delivered` |
| `opened`, `clicked`, `unsubscribed` | `read` |
| `bounced`, `spam_report`, `failed` | `failed` |

Mudança de comportamento: `unsubscribed` ia como `pending`, que não gera evento de webhook e fazia a mensagem regredir.
Agora vai como `read`, porque cancelar a inscrição exige abrir o e-mail.

**De onde vem cada campo, por provedor:**

| Provedor | `bounced` / `bounceType` | `spam_report` | `clicked` / `url` | `unsubscribed` |
|---|---|---|---|---|
| SendGrid | `bounce` (`type: bounce` → hard, `blocked` → soft) | `spamreport` (ou `spam_report`) | `click` (`url`) | `unsubscribe`, `group_unsubscribe` |
| Mailgun | `failed` com `severity: permanent` → `bounced`/hard (a temporária não gera status) | `complained` | `clicked` (`url`) | `unsubscribed` |
| SES | `Bounce` (`Permanent` → hard, `Transient` → soft, `Undetermined` → sem tipo) | `Complaint` | — | — |
| Postmark | `Bounce` (`HardBounce`, `BadEmailAddress`, `ManuallyDeactivated` → hard; `SoftBounce`, `Transient`, `DnsError` → soft; os demais sem tipo) | `SpamComplaint` | `Click` (`OriginalLink`) | `SubscriptionChange` |

SMTP não tem webhook de status e fica de fora.

**Identidade (dedup):** um status com `providerEvent` tem id próprio por evento (`evt_<messageId>_<tipo>_<providerEvent>`).
Assim a abertura e um clique depois (os dois `message.read`) não se fundem no stream. Sem `providerEvent`, o id é o de
sempre.

## Exemplo — reclamação de spam

```json
{
  "id": "evt_9b2c…_message.failed_spam_report",
  "type": "message.failed",
  "timestamp": "2026-10-24T18:02:11Z",
  "tenantId": "lk-tenant",
  "data": {
    "messageId": "9b2c…",
    "status": "failed",
    "direction": "outbound",
    "timestamp": "2026-10-24T18:02:11Z",
    "channelId": "ch-email-1",
    "channelType": "email",
    "providerEvent": "spam_report"
  }
}
```

Um hard bounce é igual, com `"providerEvent": "bounced", "bounceType": "hard"` e o `error` do provedor. Um clique é
`message.read`, com `"providerEvent": "clicked", "url": "https://…"`.

## Caminho no código

`email.StatusCallback` (`ProviderEvent`, `BounceType`, `URL`, preenchidos nos parsers de `internal/adapters/email/webhook.go`)
→ `nats.StatusUpdate` (`provider_event`, `bounce_type`, `url`) → evento `message.<status>` (`statusWebhookPayload`)
→ `webhook.StatusData` (`providerEvent`, `bounceType`, `url`).
