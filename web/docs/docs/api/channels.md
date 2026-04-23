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
| POST | `/channels` | Create a new channel |
| POST | `/channels/test` | Test channel configuration without creating it |
| POST | `/channels/test-whatsapp` | Test WhatsApp official configuration |
| POST | `/channels/test-telegram` | Test Telegram configuration |
| POST | `/channels/test-twilio` | Test SMS/Twilio configuration |
| POST | `/channels/test-facebook` | Test Facebook configuration |
| POST | `/channels/test-instagram` | Test Instagram configuration |
| GET | `/channels/:id` | Get a specific channel |
| PUT | `/channels/:id` | Update a channel |
| DELETE | `/channels/:id` | Delete a channel |
| PUT | `/channels/:id/status` | Set backwards-compatible active/inactive status |
| PUT | `/channels/:id/enabled` | Enable or disable a channel |
| POST | `/channels/:id/connect` | Connect a channel |
| POST | `/channels/:id/disconnect` | Disconnect a channel |
| POST | `/channels/:id/pair` | Request WhatsApp unofficial pair code |
| GET | `/channels/:id/coexistence-status` | Get WhatsApp official coexistence status |
| POST | `/channels/:id/subscribe-echoes` | Subscribe WhatsApp official message echoes |

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
PUT /channels/:id
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
| `identifier` | string | Channel identifier |
| `config` | object | Configuration updates |
| `credentials` | object | Sensitive credential updates |

### Example Request

```bash
curl -X PUT https://api.linktor.io/v1/channels/ch_abc123 \
  -H "Authorization: Bearer YOUR_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "WhatsApp Business - Main"
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
      "connection_status": "connected",
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

## Test Channel Configuration

Validate credentials and configuration before creating or updating a channel.

```
POST /channels/test
```

Specialized aliases are also available: `/channels/test-whatsapp`, `/channels/test-telegram`, `/channels/test-twilio`, `/channels/test-facebook`, and `/channels/test-instagram`.

### Example Request

```bash
curl -X POST https://api.linktor.io/v1/channels/test \
  -H "Authorization: Bearer YOUR_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "type": "telegram",
    "credentials": {
      "bot_token": "YOUR_BOT_TOKEN"
    }
  }'
```

### Response

```json
{
  "data": {
    "status": "ok",
    "type": "telegram",
    "valid": true,
    "message": "configuration accepted"
  }
}
```

---

## Connect Channel

Connect a channel to start receiving messages.

```
POST /channels/:id/connect
```

## Disconnect Channel

Disconnect a channel to stop receiving messages.

```
POST /channels/:id/disconnect
```

## Enable Or Disable Channel

Enable or disable a channel at system level without changing its connection session.

```
PUT /channels/:id/enabled
```

```json
{
  "enabled": true
}
```

## WhatsApp Pair Code

Request a pair code for WhatsApp unofficial authentication.

```
POST /channels/:id/pair
```

```json
{
  "phone_number": "+5511999999999"
}
```

## WhatsApp Coexistence

WhatsApp official channels expose coexistence helpers:

```
GET /channels/:id/coexistence-status
POST /channels/:id/subscribe-echoes
```

---

## Channel Statuses

| Status | Description |
|--------|-------------|
| `connected` | Channel connection is established |
| `connecting` | Channel is connecting |
| `disconnected` | Channel is disconnected |
| `error` | Channel has connection issues |
| `active` | Backwards-compatible active status |
| `inactive` | Backwards-compatible inactive status |

---

## Supported Channel Types

| Type | Description |
|------|-------------|
| `whatsapp` | WhatsApp Business API |
| `whatsapp_official` | WhatsApp Official / Meta Cloud API |
| `whatsapp_unofficial` | WhatsApp unofficial session-based channel |
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
