---
sidebar_position: 4
title: Messages
---

# Messages

The Messages API allows you to send, receive, and manage messages across all channels.

## Overview

Messages are the core of Linktor. Each message belongs to a conversation and can contain various content types including text, images, documents, audio, video, and interactive elements like buttons and quick replies.

## Endpoints

| Method | Endpoint | Description |
|--------|----------|-------------|
| POST | `/messages` | Send a new message |
| GET | `/messages` | List messages |
| GET | `/messages/:id` | Get a specific message |
| POST | `/messages/:id/read` | Mark message as read |
| POST | `/messages/bulk-read` | Mark multiple messages as read |
| GET | `/conversations/:id/messages` | List messages in a conversation |
| DELETE | `/messages/:id` | Delete a message (where supported) |

---

## Send Message

Send a message to a contact through a channel.

```
POST /messages
```

### Request Body

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `channelId` | string | Yes | Channel ID to send through |
| `conversationId` | string | No | Existing conversation ID |
| `to` | string | Conditional | Recipient identifier (required if no conversationId) |
| `content` | object | Yes | Message content |
| `metadata` | object | No | Custom metadata |
| `scheduledAt` | string | No | Schedule message for later (ISO 8601) |

### Content Types

#### Text Message

```json
{
  "content": {
    "type": "text",
    "text": "Hello! How can I help you today?"
  }
}
```

#### Image Message

```json
{
  "content": {
    "type": "image",
    "url": "https://cdn.linktor.io/images/product.jpg",
    "caption": "Our latest product"
  }
}
```

#### Document Message

```json
{
  "content": {
    "type": "document",
    "url": "https://cdn.linktor.io/docs/invoice.pdf",
    "filename": "invoice_2024.pdf",
    "caption": "Your invoice for January 2024"
  }
}
```

#### Audio Message

```json
{
  "content": {
    "type": "audio",
    "url": "https://cdn.linktor.io/audio/message.ogg"
  }
}
```

#### Video Message

```json
{
  "content": {
    "type": "video",
    "url": "https://cdn.linktor.io/videos/tutorial.mp4",
    "caption": "Quick tutorial"
  }
}
```

#### Location Message

```json
{
  "content": {
    "type": "location",
    "latitude": -23.5505,
    "longitude": -46.6333,
    "name": "Linktor Office",
    "address": "Av. Paulista, 1000, Sao Paulo"
  }
}
```

#### Buttons Message

```json
{
  "content": {
    "type": "buttons",
    "text": "How would you like to proceed?",
    "buttons": [
      {
        "type": "reply",
        "id": "btn_yes",
        "title": "Yes, continue"
      },
      {
        "type": "reply",
        "id": "btn_no",
        "title": "No, cancel"
      },
      {
        "type": "url",
        "title": "Learn more",
        "url": "https://example.com/info"
      }
    ]
  }
}
```

#### Quick Replies

```json
{
  "content": {
    "type": "text",
    "text": "What's your preferred contact method?",
    "quickReplies": [
      {
        "id": "qr_email",
        "title": "Email"
      },
      {
        "id": "qr_phone",
        "title": "Phone"
      },
      {
        "id": "qr_whatsapp",
        "title": "WhatsApp"
      }
    ]
  }
}
```

#### Template Message (WhatsApp)

```json
{
  "content": {
    "type": "template",
    "template": {
      "name": "order_confirmation",
      "language": "en",
      "components": [
        {
          "type": "header",
          "parameters": [
            {
              "type": "image",
              "image": {
                "url": "https://cdn.linktor.io/images/order.jpg"
              }
            }
          ]
        },
        {
          "type": "body",
          "parameters": [
            {
              "type": "text",
              "text": "John"
            },
            {
              "type": "text",
              "text": "ORD-12345"
            }
          ]
        }
      ]
    }
  }
}
```

### Example Request

```bash
curl -X POST https://api.linktor.io/v1/messages \
  -H "Authorization: Bearer YOUR_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "channelId": "ch_xyz789",
    "to": "+5511999999999",
    "content": {
      "type": "text",
      "text": "Hello! Your order #12345 has been shipped."
    },
    "metadata": {
      "orderId": "12345",
      "notificationType": "shipping"
    }
  }'
```

### Response

```json
{
  "data": {
    "id": "msg_abc123",
    "type": "message",
    "attributes": {
      "conversationId": "conv_xyz789",
      "channelId": "ch_xyz789",
      "direction": "outbound",
      "from": "+5511888888888",
      "to": "+5511999999999",
      "content": {
        "type": "text",
        "text": "Hello! Your order #12345 has been shipped."
      },
      "status": "sent",
      "metadata": {
        "orderId": "12345",
        "notificationType": "shipping"
      },
      "externalId": "wamid.abc123...",
      "createdAt": "2024-01-15T10:30:00Z",
      "sentAt": "2024-01-15T10:30:01Z"
    }
  }
}
```

### Error Codes

| HTTP Status | Error Code | Description |
|-------------|------------|-------------|
| 400 | `VALIDATION_ERROR` | Invalid message format |
| 400 | `INVALID_RECIPIENT` | Invalid recipient identifier |
| 400 | `UNSUPPORTED_CONTENT_TYPE` | Channel doesn't support this content type |
| 404 | `CHANNEL_NOT_FOUND` | Channel does not exist |
| 404 | `CONVERSATION_NOT_FOUND` | Conversation does not exist |
| 422 | `CHANNEL_NOT_CONNECTED` | Channel is not properly configured |
| 422 | `OUTSIDE_MESSAGE_WINDOW` | Cannot message outside 24h window (WhatsApp) |
| 429 | `RATE_LIMIT_EXCEEDED` | Too many messages |

---

## List Messages

Retrieve a paginated list of messages.

```
GET /messages
```

### Query Parameters

| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `page` | integer | 1 | Page number |
| `perPage` | integer | 20 | Items per page (max: 100) |
| `conversationId` | string | - | Filter by conversation |
| `channelId` | string | - | Filter by channel |
| `direction` | string | - | `inbound` or `outbound` |
| `status` | string | - | `pending`, `sent`, `delivered`, `read`, `failed` |
| `contentType` | string | - | Filter by content type |
| `startDate` | string | - | Filter by date (ISO 8601) |
| `endDate` | string | - | Filter by date (ISO 8601) |
| `search` | string | - | Search in message text |
| `sortBy` | string | `createdAt` | Sort field |
| `sortOrder` | string | `desc` | `asc` or `desc` |

### Example Request

```bash
curl "https://api.linktor.io/v1/messages?conversationId=conv_abc123&direction=inbound&perPage=50" \
  -H "Authorization: Bearer YOUR_API_KEY"
```

### Response

```json
{
  "data": [
    {
      "id": "msg_001",
      "type": "message",
      "attributes": {
        "conversationId": "conv_abc123",
        "channelId": "ch_xyz789",
        "direction": "inbound",
        "from": "+5511999999999",
        "to": "+5511888888888",
        "content": {
          "type": "text",
          "text": "Hi, I need help with my order"
        },
        "status": "read",
        "createdAt": "2024-01-15T10:00:00Z"
      }
    },
    {
      "id": "msg_002",
      "type": "message",
      "attributes": {
        "conversationId": "conv_abc123",
        "channelId": "ch_xyz789",
        "direction": "inbound",
        "from": "+5511999999999",
        "to": "+5511888888888",
        "content": {
          "type": "image",
          "url": "https://cdn.linktor.io/uploads/receipt.jpg",
          "caption": "Here's my receipt"
        },
        "status": "read",
        "createdAt": "2024-01-15T10:05:00Z"
      }
    }
  ],
  "meta": {
    "pagination": {
      "page": 1,
      "perPage": 50,
      "totalPages": 3,
      "totalCount": 125
    }
  }
}
```

---

## Get Message

Retrieve a specific message by ID.

```
GET /messages/:id
```

### Path Parameters

| Parameter | Type | Description |
|-----------|------|-------------|
| `id` | string | Message ID |

### Example Request

```bash
curl https://api.linktor.io/v1/messages/msg_abc123 \
  -H "Authorization: Bearer YOUR_API_KEY"
```

### Response

```json
{
  "data": {
    "id": "msg_abc123",
    "type": "message",
    "attributes": {
      "conversationId": "conv_xyz789",
      "channelId": "ch_xyz789",
      "channelType": "whatsapp",
      "direction": "outbound",
      "from": "+5511888888888",
      "to": "+5511999999999",
      "content": {
        "type": "text",
        "text": "Your order has been shipped!"
      },
      "status": "delivered",
      "statusHistory": [
        {
          "status": "pending",
          "timestamp": "2024-01-15T10:30:00Z"
        },
        {
          "status": "sent",
          "timestamp": "2024-01-15T10:30:01Z"
        },
        {
          "status": "delivered",
          "timestamp": "2024-01-15T10:30:05Z"
        }
      ],
      "externalId": "wamid.abc123...",
      "metadata": {
        "orderId": "12345"
      },
      "createdAt": "2024-01-15T10:30:00Z",
      "sentAt": "2024-01-15T10:30:01Z",
      "deliveredAt": "2024-01-15T10:30:05Z"
    }
  }
}
```

### Error Codes

| HTTP Status | Error Code | Description |
|-------------|------------|-------------|
| 404 | `MESSAGE_NOT_FOUND` | Message does not exist |

---

## List Conversation Messages

Get all messages in a specific conversation.

```
GET /conversations/:id/messages
```

### Path Parameters

| Parameter | Type | Description |
|-----------|------|-------------|
| `id` | string | Conversation ID |

### Query Parameters

| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `cursor` | string | - | Cursor for pagination |
| `limit` | integer | 50 | Number of messages (max: 100) |
| `direction` | string | `before` | `before` or `after` cursor |

### Example Request

```bash
curl "https://api.linktor.io/v1/conversations/conv_abc123/messages?limit=20" \
  -H "Authorization: Bearer YOUR_API_KEY"
```

### Response

```json
{
  "data": [
    {
      "id": "msg_001",
      "type": "message",
      "attributes": {
        "direction": "inbound",
        "content": {
          "type": "text",
          "text": "Hi there!"
        },
        "status": "read",
        "createdAt": "2024-01-15T10:00:00Z"
      }
    }
  ],
  "meta": {
    "cursor": {
      "next": "eyJpZCI6Im1zZ18wMDEifQ",
      "hasMore": true
    }
  }
}
```

---

## Mark as Read

Mark a message as read.

```
POST /messages/:id/read
```

### Path Parameters

| Parameter | Type | Description |
|-----------|------|-------------|
| `id` | string | Message ID |

### Example Request

```bash
curl -X POST https://api.linktor.io/v1/messages/msg_abc123/read \
  -H "Authorization: Bearer YOUR_API_KEY"
```

### Response

```json
{
  "data": {
    "id": "msg_abc123",
    "type": "message",
    "attributes": {
      "status": "read",
      "readAt": "2024-01-15T10:35:00Z"
    }
  }
}
```

---

## Bulk Mark as Read

Mark multiple messages as read at once.

```
POST /messages/bulk-read
```

### Request Body

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `messageIds` | array | Conditional | Array of message IDs |
| `conversationId` | string | Conditional | Mark all in conversation |
| `beforeTimestamp` | string | No | Mark messages before timestamp |

### Example Request

```bash
curl -X POST https://api.linktor.io/v1/messages/bulk-read \
  -H "Authorization: Bearer YOUR_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "conversationId": "conv_abc123",
    "beforeTimestamp": "2024-01-15T10:30:00Z"
  }'
```

### Response

```json
{
  "data": {
    "markedCount": 15,
    "conversationId": "conv_abc123"
  }
}
```

---

## Delete Message

Delete a message (where supported by the channel).

```
DELETE /messages/:id
```

### Path Parameters

| Parameter | Type | Description |
|-----------|------|-------------|
| `id` | string | Message ID |

### Query Parameters

| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `forEveryone` | boolean | false | Delete for all participants |

### Example Request

```bash
curl -X DELETE "https://api.linktor.io/v1/messages/msg_abc123?forEveryone=true" \
  -H "Authorization: Bearer YOUR_API_KEY"
```

### Response

```json
{
  "data": {
    "id": "msg_abc123",
    "deleted": true,
    "deletedAt": "2024-01-15T11:00:00Z"
  }
}
```

### Error Codes

| HTTP Status | Error Code | Description |
|-------------|------------|-------------|
| 400 | `DELETE_NOT_SUPPORTED` | Channel doesn't support message deletion |
| 400 | `DELETE_WINDOW_EXPIRED` | Message too old to delete |
| 404 | `MESSAGE_NOT_FOUND` | Message does not exist |

---

## Message Statuses

| Status | Description |
|--------|-------------|
| `pending` | Message queued for sending |
| `sent` | Message sent to channel provider |
| `delivered` | Message delivered to recipient device |
| `read` | Message read by recipient |
| `failed` | Message failed to send |

---

## Media Upload

For media messages, you can either provide a URL or upload the file first.

### Upload Media

```bash
curl -X POST https://api.linktor.io/v1/media/upload \
  -H "Authorization: Bearer YOUR_API_KEY" \
  -F "file=@/path/to/image.jpg" \
  -F "type=image"
```

### Response

```json
{
  "data": {
    "id": "media_abc123",
    "type": "image",
    "url": "https://cdn.linktor.io/uploads/media_abc123.jpg",
    "mimeType": "image/jpeg",
    "size": 102400,
    "expiresAt": "2024-01-22T10:30:00Z"
  }
}
```

Then use the URL in your message:

```bash
curl -X POST https://api.linktor.io/v1/messages \
  -H "Authorization: Bearer YOUR_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "channelId": "ch_xyz789",
    "to": "+5511999999999",
    "content": {
      "type": "image",
      "url": "https://cdn.linktor.io/uploads/media_abc123.jpg",
      "caption": "Check out this image!"
    }
  }'
```

---

## Webhooks

Message events can be received via webhooks:

- `message.received` - New inbound message
- `message.sent` - Outbound message sent
- `message.delivered` - Message delivered
- `message.read` - Message read by recipient
- `message.failed` - Message failed to send

See [Webhooks](/api/webhooks) for configuration details.
