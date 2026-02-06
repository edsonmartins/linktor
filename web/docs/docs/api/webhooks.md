---
sidebar_position: 7
title: Webhooks
---

# Webhooks

Webhooks allow you to receive real-time notifications when events occur in your Linktor account.

## Overview

Instead of polling the API for changes, webhooks push data to your server as events happen. This enables real-time integrations and faster response times for your applications.

## Endpoints

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/webhooks` | List all webhook endpoints |
| GET | `/webhooks/:id` | Get a specific webhook |
| POST | `/webhooks` | Create a webhook endpoint |
| PATCH | `/webhooks/:id` | Update a webhook |
| DELETE | `/webhooks/:id` | Delete a webhook |
| POST | `/webhooks/:id/test` | Send a test event |
| GET | `/webhooks/:id/deliveries` | List delivery attempts |
| POST | `/webhooks/:id/deliveries/:deliveryId/retry` | Retry a failed delivery |

---

## List Webhooks

Retrieve all webhook endpoints.

```
GET /webhooks
```

### Example Request

```bash
curl https://api.linktor.io/v1/webhooks \
  -H "Authorization: Bearer YOUR_API_KEY"
```

### Response

```json
{
  "data": [
    {
      "id": "wh_abc123",
      "type": "webhook",
      "attributes": {
        "url": "https://example.com/webhooks/linktor",
        "description": "Production webhook",
        "status": "active",
        "events": ["message.received", "message.sent", "conversation.created"],
        "stats": {
          "deliveriesTotal": 15000,
          "deliveriesSucceeded": 14850,
          "deliveriesFailed": 150,
          "successRate": 0.99
        },
        "createdAt": "2024-01-01T00:00:00Z",
        "updatedAt": "2024-01-15T10:00:00Z"
      }
    }
  ]
}
```

---

## Get Webhook

Retrieve a specific webhook endpoint.

```
GET /webhooks/:id
```

### Path Parameters

| Parameter | Type | Description |
|-----------|------|-------------|
| `id` | string | Webhook ID |

### Example Request

```bash
curl https://api.linktor.io/v1/webhooks/wh_abc123 \
  -H "Authorization: Bearer YOUR_API_KEY"
```

### Response

```json
{
  "data": {
    "id": "wh_abc123",
    "type": "webhook",
    "attributes": {
      "url": "https://example.com/webhooks/linktor",
      "description": "Production webhook",
      "status": "active",
      "events": [
        "message.received",
        "message.sent",
        "message.delivered",
        "message.read",
        "message.failed",
        "conversation.created",
        "conversation.closed"
      ],
      "secret": "whsec_abc123...",
      "headers": {
        "X-Custom-Header": "custom-value"
      },
      "config": {
        "retryPolicy": "exponential",
        "maxRetries": 5,
        "timeout": 30
      },
      "stats": {
        "deliveriesTotal": 15000,
        "deliveriesSucceeded": 14850,
        "deliveriesFailed": 150,
        "successRate": 0.99,
        "avgLatency": 250
      },
      "lastDelivery": {
        "id": "del_xyz789",
        "event": "message.received",
        "status": "succeeded",
        "timestamp": "2024-01-15T10:30:00Z"
      },
      "createdAt": "2024-01-01T00:00:00Z",
      "updatedAt": "2024-01-15T10:00:00Z"
    }
  }
}
```

---

## Create Webhook

Create a new webhook endpoint.

```
POST /webhooks
```

### Request Body

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `url` | string | Yes | HTTPS URL to receive events |
| `events` | array | Yes | Events to subscribe to |
| `description` | string | No | Webhook description |
| `headers` | object | No | Custom headers to include |
| `config` | object | No | Webhook configuration |

### Example Request

```bash
curl -X POST https://api.linktor.io/v1/webhooks \
  -H "Authorization: Bearer YOUR_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "url": "https://example.com/webhooks/linktor",
    "description": "Production webhook endpoint",
    "events": [
      "message.received",
      "message.sent",
      "conversation.created",
      "conversation.closed"
    ],
    "headers": {
      "X-Custom-Header": "my-value"
    },
    "config": {
      "retryPolicy": "exponential",
      "maxRetries": 5,
      "timeout": 30
    }
  }'
```

### Response

```json
{
  "data": {
    "id": "wh_new456",
    "type": "webhook",
    "attributes": {
      "url": "https://example.com/webhooks/linktor",
      "description": "Production webhook endpoint",
      "status": "active",
      "events": [
        "message.received",
        "message.sent",
        "conversation.created",
        "conversation.closed"
      ],
      "secret": "whsec_def456ghi789...",
      "headers": {
        "X-Custom-Header": "my-value"
      },
      "createdAt": "2024-01-15T11:00:00Z"
    }
  }
}
```

**Important**: Store the `secret` securely. It's only shown once and is required to verify webhook signatures.

### Error Codes

| HTTP Status | Error Code | Description |
|-------------|------------|-------------|
| 400 | `VALIDATION_ERROR` | Invalid URL or events |
| 400 | `INVALID_URL` | URL must be HTTPS |
| 400 | `INVALID_EVENTS` | Unknown event types |

---

## Update Webhook

Update an existing webhook.

```
PATCH /webhooks/:id
```

### Path Parameters

| Parameter | Type | Description |
|-----------|------|-------------|
| `id` | string | Webhook ID |

### Request Body

| Field | Type | Description |
|-------|------|-------------|
| `url` | string | Webhook URL |
| `events` | array | Events to subscribe to |
| `description` | string | Description |
| `headers` | object | Custom headers |
| `status` | string | `active` or `inactive` |
| `config` | object | Configuration |

### Example Request

```bash
curl -X PATCH https://api.linktor.io/v1/webhooks/wh_abc123 \
  -H "Authorization: Bearer YOUR_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "events": [
      "message.received",
      "message.sent",
      "message.delivered",
      "message.read",
      "conversation.created",
      "conversation.closed",
      "bot.handoff"
    ]
  }'
```

---

## Delete Webhook

Delete a webhook endpoint.

```
DELETE /webhooks/:id
```

### Example Request

```bash
curl -X DELETE https://api.linktor.io/v1/webhooks/wh_abc123 \
  -H "Authorization: Bearer YOUR_API_KEY"
```

### Response

```
HTTP/1.1 204 No Content
```

---

## Test Webhook

Send a test event to your webhook endpoint.

```
POST /webhooks/:id/test
```

### Path Parameters

| Parameter | Type | Description |
|-----------|------|-------------|
| `id` | string | Webhook ID |

### Request Body

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `event` | string | No | Event type to simulate (default: `test.ping`) |

### Example Request

```bash
curl -X POST https://api.linktor.io/v1/webhooks/wh_abc123/test \
  -H "Authorization: Bearer YOUR_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "event": "message.received"
  }'
```

### Response

```json
{
  "data": {
    "deliveryId": "del_test123",
    "status": "succeeded",
    "statusCode": 200,
    "latency": 150,
    "timestamp": "2024-01-15T11:00:00Z"
  }
}
```

---

## List Deliveries

Get the delivery history for a webhook.

```
GET /webhooks/:id/deliveries
```

### Query Parameters

| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `page` | integer | 1 | Page number |
| `perPage` | integer | 20 | Items per page |
| `status` | string | - | Filter: `succeeded`, `failed`, `pending` |
| `event` | string | - | Filter by event type |

### Example Request

```bash
curl "https://api.linktor.io/v1/webhooks/wh_abc123/deliveries?status=failed&perPage=10" \
  -H "Authorization: Bearer YOUR_API_KEY"
```

### Response

```json
{
  "data": [
    {
      "id": "del_xyz789",
      "type": "delivery",
      "attributes": {
        "event": "message.received",
        "status": "failed",
        "statusCode": 500,
        "errorMessage": "Internal Server Error",
        "attempts": 3,
        "payload": {
          "event": "message.received",
          "data": {
            "id": "msg_abc123"
          }
        },
        "request": {
          "headers": {
            "Content-Type": "application/json",
            "X-Linktor-Signature": "sha256=..."
          }
        },
        "response": {
          "statusCode": 500,
          "body": "Internal Server Error"
        },
        "createdAt": "2024-01-15T10:30:00Z",
        "lastAttemptAt": "2024-01-15T10:35:00Z"
      }
    }
  ],
  "meta": {
    "pagination": {
      "page": 1,
      "perPage": 10,
      "totalCount": 25
    }
  }
}
```

---

## Retry Delivery

Manually retry a failed webhook delivery.

```
POST /webhooks/:id/deliveries/:deliveryId/retry
```

### Example Request

```bash
curl -X POST https://api.linktor.io/v1/webhooks/wh_abc123/deliveries/del_xyz789/retry \
  -H "Authorization: Bearer YOUR_API_KEY"
```

### Response

```json
{
  "data": {
    "id": "del_xyz789",
    "status": "succeeded",
    "statusCode": 200,
    "attempts": 4,
    "retriedAt": "2024-01-15T11:00:00Z"
  }
}
```

---

## Event Types

### Message Events

| Event | Description |
|-------|-------------|
| `message.received` | New inbound message received |
| `message.sent` | Outbound message sent |
| `message.delivered` | Message delivered to recipient |
| `message.read` | Message read by recipient |
| `message.failed` | Message failed to send |

### Conversation Events

| Event | Description |
|-------|-------------|
| `conversation.created` | New conversation started |
| `conversation.updated` | Conversation properties changed |
| `conversation.assigned` | Conversation assigned to user/bot |
| `conversation.transferred` | Conversation transferred |
| `conversation.closed` | Conversation closed |
| `conversation.reopened` | Conversation reopened |

### Channel Events

| Event | Description |
|-------|-------------|
| `channel.created` | New channel created |
| `channel.updated` | Channel configuration changed |
| `channel.activated` | Channel activated |
| `channel.deactivated` | Channel deactivated |
| `channel.error` | Channel encountered an error |
| `channel.deleted` | Channel deleted |

### Bot Events

| Event | Description |
|-------|-------------|
| `bot.created` | New bot created |
| `bot.updated` | Bot configuration changed |
| `bot.activated` | Bot activated |
| `bot.deactivated` | Bot deactivated |
| `bot.handoff` | Bot handed off to human agent |
| `bot.training.started` | Knowledge base training started |
| `bot.training.completed` | Knowledge base training completed |
| `bot.training.failed` | Knowledge base training failed |

### Contact Events

| Event | Description |
|-------|-------------|
| `contact.created` | New contact created |
| `contact.updated` | Contact updated |
| `contact.merged` | Contacts merged |

### System Events

| Event | Description |
|-------|-------------|
| `test.ping` | Test event for webhook verification |

---

## Webhook Payload

All webhook events are delivered with the following structure:

```json
{
  "id": "evt_abc123",
  "event": "message.received",
  "timestamp": "2024-01-15T10:30:00Z",
  "data": {
    "id": "msg_xyz789",
    "type": "message",
    "attributes": {
      "conversationId": "conv_123",
      "channelId": "ch_456",
      "direction": "inbound",
      "content": {
        "type": "text",
        "text": "Hello, I need help!"
      },
      "createdAt": "2024-01-15T10:30:00Z"
    }
  },
  "meta": {
    "organizationId": "org_abc",
    "webhookId": "wh_xyz"
  }
}
```

---

## Signature Verification

All webhook payloads are signed using HMAC-SHA256. Always verify signatures to ensure payloads are from Linktor.

### Headers

| Header | Description |
|--------|-------------|
| `X-Linktor-Signature` | HMAC-SHA256 signature |
| `X-Linktor-Timestamp` | Unix timestamp of the request |
| `X-Linktor-Event` | Event type |
| `X-Linktor-Delivery-Id` | Unique delivery ID |

### Verification Process

1. Extract the timestamp and signature from headers
2. Construct the signed payload: `{timestamp}.{body}`
3. Compute HMAC-SHA256 using your webhook secret
4. Compare with the provided signature

### Example (Node.js)

```javascript
const crypto = require('crypto');

function verifyWebhookSignature(payload, signature, timestamp, secret) {
  // Protect against replay attacks (5 minute tolerance)
  const tolerance = 5 * 60 * 1000;
  const now = Date.now();
  const requestTime = parseInt(timestamp) * 1000;

  if (Math.abs(now - requestTime) > tolerance) {
    throw new Error('Timestamp outside tolerance');
  }

  // Compute expected signature
  const signedPayload = `${timestamp}.${payload}`;
  const expectedSignature = crypto
    .createHmac('sha256', secret)
    .update(signedPayload)
    .digest('hex');

  // Compare signatures
  const providedSig = signature.replace('sha256=', '');

  if (!crypto.timingSafeEqual(
    Buffer.from(expectedSignature),
    Buffer.from(providedSig)
  )) {
    throw new Error('Invalid signature');
  }

  return true;
}

// Express middleware example
app.post('/webhooks/linktor', (req, res) => {
  const signature = req.headers['x-linktor-signature'];
  const timestamp = req.headers['x-linktor-timestamp'];
  const payload = JSON.stringify(req.body);

  try {
    verifyWebhookSignature(payload, signature, timestamp, process.env.WEBHOOK_SECRET);

    // Process the event
    const event = req.body;
    console.log(`Received event: ${event.event}`);

    res.status(200).send('OK');
  } catch (error) {
    console.error('Webhook verification failed:', error);
    res.status(401).send('Invalid signature');
  }
});
```

### Example (Python)

```python
import hmac
import hashlib
import time

def verify_webhook_signature(payload: bytes, signature: str, timestamp: str, secret: str) -> bool:
    # Protect against replay attacks (5 minute tolerance)
    tolerance = 5 * 60
    request_time = int(timestamp)

    if abs(time.time() - request_time) > tolerance:
        raise ValueError('Timestamp outside tolerance')

    # Compute expected signature
    signed_payload = f"{timestamp}.{payload.decode('utf-8')}"
    expected_signature = hmac.new(
        secret.encode('utf-8'),
        signed_payload.encode('utf-8'),
        hashlib.sha256
    ).hexdigest()

    # Compare signatures
    provided_sig = signature.replace('sha256=', '')

    if not hmac.compare_digest(expected_signature, provided_sig):
        raise ValueError('Invalid signature')

    return True

# Flask example
@app.route('/webhooks/linktor', methods=['POST'])
def handle_webhook():
    signature = request.headers.get('X-Linktor-Signature')
    timestamp = request.headers.get('X-Linktor-Timestamp')
    payload = request.get_data()

    try:
        verify_webhook_signature(payload, signature, timestamp, WEBHOOK_SECRET)

        event = request.json
        print(f"Received event: {event['event']}")

        return 'OK', 200
    except ValueError as e:
        print(f'Webhook verification failed: {e}')
        return 'Invalid signature', 401
```

---

## Retry Policy

When a webhook delivery fails, Linktor automatically retries with exponential backoff:

| Attempt | Delay |
|---------|-------|
| 1 | Immediate |
| 2 | 1 minute |
| 3 | 5 minutes |
| 4 | 30 minutes |
| 5 | 2 hours |

A delivery is considered failed if:
- The endpoint returns a non-2xx status code
- The request times out (default: 30 seconds)
- A network error occurs

After all retries are exhausted, the delivery is marked as failed. You can manually retry failed deliveries via the API.

---

## Best Practices

### Respond Quickly

Return a 2xx response as quickly as possible. Process events asynchronously:

```javascript
app.post('/webhooks/linktor', async (req, res) => {
  // Acknowledge receipt immediately
  res.status(200).send('OK');

  // Process asynchronously
  processEvent(req.body).catch(console.error);
});
```

### Handle Duplicates

Events may be delivered more than once. Use the event `id` for idempotency:

```javascript
async function processEvent(event) {
  const processed = await db.checkProcessed(event.id);
  if (processed) {
    console.log(`Event ${event.id} already processed`);
    return;
  }

  await handleEvent(event);
  await db.markProcessed(event.id);
}
```

### Verify Signatures

Always verify webhook signatures to ensure requests are from Linktor.

### Use HTTPS

Webhook endpoints must use HTTPS with a valid SSL certificate.

### Monitor Failures

Set up alerts for webhook delivery failures to catch issues quickly.

### Handle All Events

Subscribe only to events you need, but handle unknown events gracefully:

```javascript
switch (event.event) {
  case 'message.received':
    await handleMessageReceived(event.data);
    break;
  case 'conversation.created':
    await handleConversationCreated(event.data);
    break;
  default:
    console.log(`Unhandled event: ${event.event}`);
}
```
