---
sidebar_position: 5
title: Channels
---

# Channels

The Channels API allows you to configure and manage messaging channels for your organization.

## Overview

Channels are integrations with external messaging platforms (WhatsApp, Telegram, SMS, etc.). Each channel connects Linktor to a specific provider account and handles message routing between your organization and customers.

## Endpoints

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/channels` | List all channels |
| GET | `/channels/:id` | Get a specific channel |
| POST | `/channels` | Create a new channel |
| PATCH | `/channels/:id` | Update a channel |
| DELETE | `/channels/:id` | Delete a channel |
| POST | `/channels/:id/test` | Test channel connection |
| POST | `/channels/:id/activate` | Activate a channel |
| POST | `/channels/:id/deactivate` | Deactivate a channel |
| GET | `/channels/:id/stats` | Get channel statistics |

---

## List Channels

Retrieve all channels for your organization.

```
GET /channels
```

### Query Parameters

| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `page` | integer | 1 | Page number |
| `perPage` | integer | 20 | Items per page (max: 100) |
| `type` | string | - | Filter by channel type |
| `status` | string | - | Filter by status: `active`, `inactive`, `error` |
| `search` | string | - | Search by channel name |

### Example Request

```bash
curl "https://api.linktor.io/v1/channels?status=active" \
  -H "Authorization: Bearer YOUR_API_KEY"
```

### Response

```json
{
  "data": [
    {
      "id": "ch_abc123",
      "type": "channel",
      "attributes": {
        "name": "WhatsApp Business",
        "channelType": "whatsapp",
        "status": "active",
        "config": {
          "phoneNumber": "+5511888888888",
          "businessName": "Acme Corp"
        },
        "stats": {
          "messagesReceived24h": 150,
          "messagesSent24h": 120,
          "activeConversations": 45
        },
        "webhookUrl": "https://api.linktor.io/webhooks/ch_abc123",
        "createdAt": "2024-01-01T00:00:00Z",
        "updatedAt": "2024-01-15T10:00:00Z"
      }
    },
    {
      "id": "ch_def456",
      "type": "channel",
      "attributes": {
        "name": "Support Telegram",
        "channelType": "telegram",
        "status": "active",
        "config": {
          "botUsername": "@acme_support_bot"
        },
        "stats": {
          "messagesReceived24h": 80,
          "messagesSent24h": 75,
          "activeConversations": 20
        },
        "webhookUrl": "https://api.linktor.io/webhooks/ch_def456",
        "createdAt": "2024-01-05T00:00:00Z",
        "updatedAt": "2024-01-15T09:00:00Z"
      }
    }
  ],
  "meta": {
    "pagination": {
      "page": 1,
      "perPage": 20,
      "totalPages": 1,
      "totalCount": 2
    }
  }
}
```

---

## Get Channel

Retrieve a specific channel by ID.

```
GET /channels/:id
```

### Path Parameters

| Parameter | Type | Description |
|-----------|------|-------------|
| `id` | string | Channel ID |

### Example Request

```bash
curl https://api.linktor.io/v1/channels/ch_abc123 \
  -H "Authorization: Bearer YOUR_API_KEY"
```

### Response

```json
{
  "data": {
    "id": "ch_abc123",
    "type": "channel",
    "attributes": {
      "name": "WhatsApp Business",
      "channelType": "whatsapp",
      "status": "active",
      "config": {
        "phoneNumber": "+5511888888888",
        "phoneNumberId": "123456789",
        "businessAccountId": "987654321",
        "businessName": "Acme Corp",
        "verifyToken": "********"
      },
      "capabilities": {
        "text": true,
        "image": true,
        "document": true,
        "audio": true,
        "video": true,
        "location": true,
        "buttons": true,
        "quickReplies": true,
        "templates": true,
        "deliveryReceipts": true,
        "readReceipts": true,
        "typingIndicators": true
      },
      "defaultBot": {
        "id": "bot_xyz",
        "name": "Support Bot"
      },
      "webhookUrl": "https://api.linktor.io/webhooks/ch_abc123",
      "webhookSecret": "********",
      "stats": {
        "messagesReceived24h": 150,
        "messagesSent24h": 120,
        "messagesReceivedTotal": 15000,
        "messagesSentTotal": 12000,
        "activeConversations": 45,
        "totalConversations": 2500
      },
      "health": {
        "status": "healthy",
        "lastChecked": "2024-01-15T10:00:00Z",
        "lastMessageReceived": "2024-01-15T09:55:00Z",
        "lastMessageSent": "2024-01-15T09:58:00Z"
      },
      "createdAt": "2024-01-01T00:00:00Z",
      "updatedAt": "2024-01-15T10:00:00Z"
    }
  }
}
```

### Error Codes

| HTTP Status | Error Code | Description |
|-------------|------------|-------------|
| 404 | `CHANNEL_NOT_FOUND` | Channel does not exist |

---

## Create Channel

Create a new messaging channel.

```
POST /channels
```

### Request Body

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `name` | string | Yes | Display name for the channel |
| `type` | string | Yes | Channel type (see below) |
| `config` | object | Yes | Channel-specific configuration |
| `defaultBotId` | string | No | Bot to handle conversations |
| `metadata` | object | No | Custom metadata |

### Channel Types and Configuration

#### WhatsApp

```json
{
  "name": "WhatsApp Business",
  "type": "whatsapp",
  "config": {
    "phoneNumberId": "YOUR_PHONE_NUMBER_ID",
    "accessToken": "YOUR_ACCESS_TOKEN",
    "businessAccountId": "YOUR_BUSINESS_ACCOUNT_ID",
    "verifyToken": "YOUR_VERIFY_TOKEN"
  }
}
```

#### Telegram

```json
{
  "name": "Telegram Bot",
  "type": "telegram",
  "config": {
    "botToken": "YOUR_BOT_TOKEN"
  }
}
```

#### SMS (Twilio)

```json
{
  "name": "SMS Support",
  "type": "sms",
  "config": {
    "provider": "twilio",
    "accountSid": "YOUR_ACCOUNT_SID",
    "authToken": "YOUR_AUTH_TOKEN",
    "phoneNumber": "+15551234567"
  }
}
```

#### Email

```json
{
  "name": "Support Email",
  "type": "email",
  "config": {
    "provider": "smtp",
    "smtpHost": "smtp.example.com",
    "smtpPort": 587,
    "smtpUser": "support@example.com",
    "smtpPassword": "YOUR_PASSWORD",
    "imapHost": "imap.example.com",
    "imapPort": 993,
    "imapUser": "support@example.com",
    "imapPassword": "YOUR_PASSWORD",
    "fromName": "Acme Support",
    "fromEmail": "support@example.com"
  }
}
```

#### WebChat

```json
{
  "name": "Website Chat",
  "type": "webchat",
  "config": {
    "allowedOrigins": ["https://www.example.com", "https://app.example.com"],
    "theme": {
      "primaryColor": "#0066cc",
      "headerText": "Chat with us",
      "welcomeMessage": "Hi! How can we help?"
    },
    "requireEmail": false,
    "showBranding": true
  }
}
```

#### Facebook Messenger

```json
{
  "name": "Facebook Page",
  "type": "facebook",
  "config": {
    "pageId": "YOUR_PAGE_ID",
    "pageAccessToken": "YOUR_PAGE_ACCESS_TOKEN",
    "appSecret": "YOUR_APP_SECRET"
  }
}
```

#### Instagram

```json
{
  "name": "Instagram Business",
  "type": "instagram",
  "config": {
    "accountId": "YOUR_INSTAGRAM_ACCOUNT_ID",
    "accessToken": "YOUR_ACCESS_TOKEN"
  }
}
```

#### Voice (Twilio)

```json
{
  "name": "Voice Support",
  "type": "voice",
  "config": {
    "provider": "twilio",
    "accountSid": "YOUR_ACCOUNT_SID",
    "authToken": "YOUR_AUTH_TOKEN",
    "phoneNumber": "+15551234567",
    "speechToText": true,
    "textToSpeech": true,
    "recordCalls": true
  }
}
```

### Example Request

```bash
curl -X POST https://api.linktor.io/v1/channels \
  -H "Authorization: Bearer YOUR_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "WhatsApp Business",
    "type": "whatsapp",
    "config": {
      "phoneNumberId": "123456789",
      "accessToken": "EAABcd...",
      "businessAccountId": "987654321",
      "verifyToken": "my_verify_token"
    },
    "defaultBotId": "bot_abc123"
  }'
```

### Response

```json
{
  "data": {
    "id": "ch_new789",
    "type": "channel",
    "attributes": {
      "name": "WhatsApp Business",
      "channelType": "whatsapp",
      "status": "inactive",
      "config": {
        "phoneNumber": "+5511888888888",
        "phoneNumberId": "123456789",
        "businessAccountId": "987654321"
      },
      "webhookUrl": "https://api.linktor.io/webhooks/ch_new789",
      "webhookSecret": "whsec_abc123...",
      "createdAt": "2024-01-15T11:00:00Z"
    }
  },
  "meta": {
    "setupInstructions": "Configure your WhatsApp webhook to: https://api.linktor.io/webhooks/ch_new789 with verify token: my_verify_token"
  }
}
```

### Error Codes

| HTTP Status | Error Code | Description |
|-------------|------------|-------------|
| 400 | `VALIDATION_ERROR` | Invalid configuration |
| 400 | `INVALID_CHANNEL_TYPE` | Unsupported channel type |
| 409 | `CHANNEL_ALREADY_EXISTS` | Channel with same identifier exists |

---

## Update Channel

Update an existing channel.

```
PATCH /channels/:id
```

### Path Parameters

| Parameter | Type | Description |
|-----------|------|-------------|
| `id` | string | Channel ID |

### Request Body

| Field | Type | Description |
|-------|------|-------------|
| `name` | string | Display name |
| `config` | object | Configuration updates |
| `defaultBotId` | string | Default bot ID |
| `metadata` | object | Custom metadata |

### Example Request

```bash
curl -X PATCH https://api.linktor.io/v1/channels/ch_abc123 \
  -H "Authorization: Bearer YOUR_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "WhatsApp Business - Main",
    "defaultBotId": "bot_new456"
  }'
```

### Response

```json
{
  "data": {
    "id": "ch_abc123",
    "type": "channel",
    "attributes": {
      "name": "WhatsApp Business - Main",
      "channelType": "whatsapp",
      "status": "active",
      "defaultBot": {
        "id": "bot_new456",
        "name": "New Bot"
      },
      "updatedAt": "2024-01-15T12:00:00Z"
    }
  }
}
```

---

## Delete Channel

Delete a channel. This will close all active conversations on this channel.

```
DELETE /channels/:id
```

### Path Parameters

| Parameter | Type | Description |
|-----------|------|-------------|
| `id` | string | Channel ID |

### Query Parameters

| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `force` | boolean | false | Delete even if active conversations exist |

### Example Request

```bash
curl -X DELETE "https://api.linktor.io/v1/channels/ch_abc123?force=true" \
  -H "Authorization: Bearer YOUR_API_KEY"
```

### Response

```
HTTP/1.1 204 No Content
```

### Error Codes

| HTTP Status | Error Code | Description |
|-------------|------------|-------------|
| 400 | `ACTIVE_CONVERSATIONS_EXIST` | Channel has active conversations |
| 404 | `CHANNEL_NOT_FOUND` | Channel does not exist |

---

## Test Connection

Test the channel connection and verify credentials.

```
POST /channels/:id/test
```

### Path Parameters

| Parameter | Type | Description |
|-----------|------|-------------|
| `id` | string | Channel ID |

### Example Request

```bash
curl -X POST https://api.linktor.io/v1/channels/ch_abc123/test \
  -H "Authorization: Bearer YOUR_API_KEY"
```

### Response (Success)

```json
{
  "data": {
    "success": true,
    "channelId": "ch_abc123",
    "tests": [
      {
        "name": "credentials",
        "status": "passed",
        "message": "API credentials are valid"
      },
      {
        "name": "webhook",
        "status": "passed",
        "message": "Webhook is reachable"
      },
      {
        "name": "permissions",
        "status": "passed",
        "message": "All required permissions granted"
      }
    ],
    "testedAt": "2024-01-15T10:30:00Z"
  }
}
```

### Response (Failure)

```json
{
  "data": {
    "success": false,
    "channelId": "ch_abc123",
    "tests": [
      {
        "name": "credentials",
        "status": "passed",
        "message": "API credentials are valid"
      },
      {
        "name": "webhook",
        "status": "failed",
        "message": "Webhook URL returned 404",
        "details": {
          "url": "https://api.linktor.io/webhooks/ch_abc123",
          "statusCode": 404
        }
      }
    ],
    "testedAt": "2024-01-15T10:30:00Z"
  }
}
```

---

## Activate Channel

Enable a channel to start receiving and sending messages.

```
POST /channels/:id/activate
```

### Path Parameters

| Parameter | Type | Description |
|-----------|------|-------------|
| `id` | string | Channel ID |

### Example Request

```bash
curl -X POST https://api.linktor.io/v1/channels/ch_abc123/activate \
  -H "Authorization: Bearer YOUR_API_KEY"
```

### Response

```json
{
  "data": {
    "id": "ch_abc123",
    "type": "channel",
    "attributes": {
      "status": "active",
      "activatedAt": "2024-01-15T10:30:00Z"
    }
  }
}
```

---

## Deactivate Channel

Disable a channel. New messages will not be received or sent.

```
POST /channels/:id/deactivate
```

### Path Parameters

| Parameter | Type | Description |
|-----------|------|-------------|
| `id` | string | Channel ID |

### Request Body

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `reason` | string | No | Reason for deactivation |
| `closeConversations` | boolean | No | Close active conversations |

### Example Request

```bash
curl -X POST https://api.linktor.io/v1/channels/ch_abc123/deactivate \
  -H "Authorization: Bearer YOUR_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "reason": "Maintenance",
    "closeConversations": false
  }'
```

### Response

```json
{
  "data": {
    "id": "ch_abc123",
    "type": "channel",
    "attributes": {
      "status": "inactive",
      "deactivatedAt": "2024-01-15T10:30:00Z",
      "deactivationReason": "Maintenance"
    }
  }
}
```

---

## Get Channel Statistics

Retrieve detailed statistics for a channel.

```
GET /channels/:id/stats
```

### Path Parameters

| Parameter | Type | Description |
|-----------|------|-------------|
| `id` | string | Channel ID |

### Query Parameters

| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `period` | string | `24h` | Time period: `1h`, `24h`, `7d`, `30d` |
| `granularity` | string | `hour` | Data granularity: `minute`, `hour`, `day` |

### Example Request

```bash
curl "https://api.linktor.io/v1/channels/ch_abc123/stats?period=7d&granularity=day" \
  -H "Authorization: Bearer YOUR_API_KEY"
```

### Response

```json
{
  "data": {
    "channelId": "ch_abc123",
    "period": "7d",
    "summary": {
      "messagesReceived": 1050,
      "messagesSent": 980,
      "conversationsStarted": 150,
      "conversationsClosed": 120,
      "avgResponseTime": 45,
      "avgResolutionTime": 1800
    },
    "timeSeries": [
      {
        "timestamp": "2024-01-09T00:00:00Z",
        "messagesReceived": 140,
        "messagesSent": 130,
        "conversationsStarted": 20
      },
      {
        "timestamp": "2024-01-10T00:00:00Z",
        "messagesReceived": 155,
        "messagesSent": 145,
        "conversationsStarted": 22
      }
    ]
  }
}
```

---

## Channel Statuses

| Status | Description |
|--------|-------------|
| `active` | Channel is operational |
| `inactive` | Channel is disabled |
| `error` | Channel has connection issues |
| `pending` | Channel setup incomplete |

---

## Supported Channel Types

| Type | Description |
|------|-------------|
| `whatsapp` | WhatsApp Business API |
| `telegram` | Telegram Bot API |
| `sms` | SMS via Twilio, Vonage, etc. |
| `email` | Email via SMTP/IMAP |
| `voice` | Voice calls via Twilio, etc. |
| `webchat` | Website chat widget |
| `facebook` | Facebook Messenger |
| `instagram` | Instagram Direct Messages |
| `rcs` | RCS Business Messaging |

---

## Webhooks

Channel events can be received via webhooks:

- `channel.created` - New channel created
- `channel.updated` - Channel configuration changed
- `channel.activated` - Channel activated
- `channel.deactivated` - Channel deactivated
- `channel.error` - Channel encountered an error
- `channel.deleted` - Channel deleted

See [Webhooks](/api/webhooks) for configuration details.
