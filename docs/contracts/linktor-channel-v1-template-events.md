# `linktor-channel-v1` — template events

Two additive event types on the canonical outbound webhook (`internal/infrastructure/webhook/envelope.go`). They ride
the same envelope as every other event, with the same signature, retry and DLQ:

```json
{ "id": "evt_…", "type": "template.status_changed", "timestamp": "2026-10-31T12:00:00Z",
  "tenantId": "<Linktor tenant>", "environment": "production", "data": { … } }
```

## Opt-in

Both types are delivered **only** to a channel that lists them **by name** in `webhook_events`, for example
`message.received,template.status_changed,template.quality_changed`. A channel with no list, or with `*`, keeps
receiving exactly the types it received before, so existing consumers never see a type they do not know.

## `template.status_changed`

Emitted by `ProcessTemplateStatusWebhook` when Meta changes a template's status, including `PAUSED`. It is emitted
only when the status actually changes. Delivered on the webhook of the template's channel.

| Field | Meaning |
|---|---|
| `templateId` | Linktor template id |
| `externalId` | Meta template id |
| `name`, `language`, `category` | as stored in Linktor |
| `previousStatus`, `status` | `PENDING`, `APPROVED`, `REJECTED`, `PAUSED`, `DISABLED`, `IN_APPEAL`, `PENDING_DELETION`, `DELETED`, `REINSTATED`, `LIMIT_EXCEEDED`, `ARCHIVED` |
| `reason` | reason sent by Meta, when there is one |
| `channelId`, `channelType` | channel of the template |

## `template.quality_changed`

Emitted by `ProcessTemplateQualityWebhook` when Meta changes a template's quality rating. It is emitted only when the
rating actually changes. The `data` mirrors `template.status_changed`, with `previousQuality` and `quality` in place of
`previousStatus` and `status`, and no `reason`.

| Field | Meaning |
|---|---|
| `templateId`, `externalId`, `name`, `language`, `category`, `channelId`, `channelType` | as in `template.status_changed` |
| `previousQuality`, `quality` | `GREEN`, `YELLOW`, `RED`, `UNKNOWN` |

## Delivery

Each occurrence gets a fresh event id, since a status or a quality rating can legitimately repeat over time.
Publishing is best-effort: if the event cannot be published, the Meta webhook still succeeds and the change is still
persisted.
