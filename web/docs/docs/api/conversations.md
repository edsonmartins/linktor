---
sidebar_position: 3
title: Conversations
---

# Conversations

The Conversations API allows you to manage conversations between your organization and contacts across all messaging channels.

## Overview

A conversation represents an ongoing dialogue with a contact. Conversations can span multiple messages, sessions, and even channels (when contacts switch platforms). Each conversation has a lifecycle from creation to closure.

## Endpoints

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/conversations` | List all conversations |
| GET | `/conversations/:id` | Get a specific conversation |
| POST | `/conversations` | Create a new conversation |
| PATCH | `/conversations/:id` | Update a conversation |
| POST | `/conversations/:id/close` | Close a conversation |
| POST | `/conversations/:id/reopen` | Reopen a closed conversation |
| POST | `/conversations/:id/assign` | Assign conversation to user/bot |
| POST | `/conversations/:id/transfer` | Transfer to another assignee |
| GET | `/conversations/:id/participants` | List conversation participants |

---

## List Conversations

Retrieve a paginated list of conversations.

```
GET /conversations
```

### Query Parameters

| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `page` | integer | 1 | Page number |
| `perPage` | integer | 20 | Items per page (max: 100) |
| `status` | string | - | Filter by status: `open`, `pending`, `closed` |
| `channelId` | string | - | Filter by channel ID |
| `channelType` | string | - | Filter by channel type |
| `assigneeId` | string | - | Filter by assignee (user or bot ID) |
| `contactId` | string | - | Filter by contact ID |
| `botId` | string | - | Filter by assigned bot |
| `startDate` | string | - | Filter by created date (ISO 8601) |
| `endDate` | string | - | Filter by created date (ISO 8601) |
| `search` | string | - | Search in contact name or messages |
| `sortBy` | string | `updatedAt` | Sort field |
| `sortOrder` | string | `desc` | Sort order: `asc` or `desc` |

### Example Request

```bash
curl "https://api.linktor.io/v1/conversations?status=open&channelType=whatsapp&perPage=50" \
  -H "Authorization: Bearer YOUR_API_KEY"
```

### Response

```json
{
  "data": [
    {
      "id": "conv_abc123",
      "type": "conversation",
      "attributes": {
        "status": "open",
        "channel": {
          "id": "ch_xyz789",
          "type": "whatsapp",
          "name": "WhatsApp Business"
        },
        "contact": {
          "id": "contact_123",
          "name": "John Doe",
          "phone": "+5511999999999",
          "avatar": "https://cdn.linktor.io/avatars/contact_123.jpg"
        },
        "assignee": {
          "id": "user_456",
          "type": "user",
          "name": "Support Agent"
        },
        "lastMessage": {
          "id": "msg_789",
          "content": {
            "type": "text",
            "text": "Thanks for your help!"
          },
          "direction": "inbound",
          "timestamp": "2024-01-15T10:35:00Z"
        },
        "unreadCount": 2,
        "messageCount": 15,
        "metadata": {
          "source": "website",
          "campaign": "winter_sale"
        },
        "createdAt": "2024-01-15T10:00:00Z",
        "updatedAt": "2024-01-15T10:35:00Z"
      }
    }
  ],
  "meta": {
    "pagination": {
      "page": 1,
      "perPage": 50,
      "totalPages": 10,
      "totalCount": 500
    }
  }
}
```

---

## Get Conversation

Retrieve a specific conversation by ID.

```
GET /conversations/:id
```

### Path Parameters

| Parameter | Type | Description |
|-----------|------|-------------|
| `id` | string | Conversation ID |

### Query Parameters

| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `includeMessages` | boolean | false | Include recent messages |
| `messageLimit` | integer | 20 | Number of messages to include |

### Example Request

```bash
curl "https://api.linktor.io/v1/conversations/conv_abc123?includeMessages=true&messageLimit=10" \
  -H "Authorization: Bearer YOUR_API_KEY"
```

### Response

```json
{
  "data": {
    "id": "conv_abc123",
    "type": "conversation",
    "attributes": {
      "status": "open",
      "channel": {
        "id": "ch_xyz789",
        "type": "whatsapp",
        "name": "WhatsApp Business"
      },
      "contact": {
        "id": "contact_123",
        "name": "John Doe",
        "phone": "+5511999999999",
        "email": "john@example.com",
        "avatar": "https://cdn.linktor.io/avatars/contact_123.jpg",
        "metadata": {
          "customerId": "cust_789",
          "tier": "premium"
        }
      },
      "assignee": {
        "id": "user_456",
        "type": "user",
        "name": "Support Agent"
      },
      "bot": {
        "id": "bot_abc",
        "name": "Support Bot",
        "active": true
      },
      "unreadCount": 2,
      "messageCount": 15,
      "firstMessageAt": "2024-01-15T10:00:00Z",
      "lastMessageAt": "2024-01-15T10:35:00Z",
      "metadata": {
        "source": "website",
        "campaign": "winter_sale",
        "intent": "order_status"
      },
      "tags": ["vip", "order-issue"],
      "createdAt": "2024-01-15T10:00:00Z",
      "updatedAt": "2024-01-15T10:35:00Z"
    },
    "relationships": {
      "messages": [
        {
          "id": "msg_001",
          "direction": "inbound",
          "content": {
            "type": "text",
            "text": "Hi, I need help with my order"
          },
          "timestamp": "2024-01-15T10:00:00Z"
        },
        {
          "id": "msg_002",
          "direction": "outbound",
          "content": {
            "type": "text",
            "text": "Hello! I'd be happy to help. What's your order number?"
          },
          "timestamp": "2024-01-15T10:01:00Z"
        }
      ]
    }
  }
}
```

### Error Codes

| HTTP Status | Error Code | Description |
|-------------|------------|-------------|
| 404 | `CONVERSATION_NOT_FOUND` | Conversation does not exist |

---

## Create Conversation

Create a new conversation with a contact.

```
POST /conversations
```

### Request Body

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `channelId` | string | Yes | Channel ID to use |
| `contactId` | string | No | Existing contact ID |
| `contact` | object | No | New contact details (if contactId not provided) |
| `contact.phone` | string | Conditional | Contact phone (E.164 format) |
| `contact.email` | string | Conditional | Contact email |
| `contact.name` | string | No | Contact display name |
| `assigneeId` | string | No | User or bot ID to assign |
| `botId` | string | No | Bot to handle conversation |
| `metadata` | object | No | Custom metadata |
| `tags` | array | No | Array of tag strings |

### Example Request

```bash
curl -X POST https://api.linktor.io/v1/conversations \
  -H "Authorization: Bearer YOUR_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "channelId": "ch_xyz789",
    "contact": {
      "phone": "+5511999999999",
      "name": "Jane Smith"
    },
    "botId": "bot_abc",
    "metadata": {
      "source": "api",
      "orderId": "order_12345"
    },
    "tags": ["api-created", "order-inquiry"]
  }'
```

### Response

```json
{
  "data": {
    "id": "conv_new123",
    "type": "conversation",
    "attributes": {
      "status": "open",
      "channel": {
        "id": "ch_xyz789",
        "type": "whatsapp",
        "name": "WhatsApp Business"
      },
      "contact": {
        "id": "contact_new456",
        "name": "Jane Smith",
        "phone": "+5511999999999"
      },
      "bot": {
        "id": "bot_abc",
        "name": "Support Bot",
        "active": true
      },
      "unreadCount": 0,
      "messageCount": 0,
      "metadata": {
        "source": "api",
        "orderId": "order_12345"
      },
      "tags": ["api-created", "order-inquiry"],
      "createdAt": "2024-01-15T11:00:00Z",
      "updatedAt": "2024-01-15T11:00:00Z"
    }
  }
}
```

### Error Codes

| HTTP Status | Error Code | Description |
|-------------|------------|-------------|
| 400 | `VALIDATION_ERROR` | Invalid request parameters |
| 404 | `CHANNEL_NOT_FOUND` | Channel does not exist |
| 404 | `CONTACT_NOT_FOUND` | Contact ID not found |
| 409 | `CONVERSATION_EXISTS` | Active conversation already exists |

---

## Update Conversation

Update conversation properties.

```
PATCH /conversations/:id
```

### Path Parameters

| Parameter | Type | Description |
|-----------|------|-------------|
| `id` | string | Conversation ID |

### Request Body

| Field | Type | Description |
|-------|------|-------------|
| `status` | string | Conversation status |
| `metadata` | object | Custom metadata (merged with existing) |
| `tags` | array | Replace tags array |
| `priority` | string | Priority: `low`, `normal`, `high`, `urgent` |

### Example Request

```bash
curl -X PATCH https://api.linktor.io/v1/conversations/conv_abc123 \
  -H "Authorization: Bearer YOUR_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "priority": "high",
    "metadata": {
      "escalationReason": "customer_request"
    },
    "tags": ["vip", "escalated"]
  }'
```

### Response

```json
{
  "data": {
    "id": "conv_abc123",
    "type": "conversation",
    "attributes": {
      "status": "open",
      "priority": "high",
      "metadata": {
        "source": "website",
        "escalationReason": "customer_request"
      },
      "tags": ["vip", "escalated"],
      "updatedAt": "2024-01-15T11:30:00Z"
    }
  }
}
```

---

## Close Conversation

Mark a conversation as closed.

```
POST /conversations/:id/close
```

### Path Parameters

| Parameter | Type | Description |
|-----------|------|-------------|
| `id` | string | Conversation ID |

### Request Body

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `reason` | string | No | Closure reason |
| `resolution` | string | No | Resolution status: `resolved`, `unresolved`, `spam` |
| `notes` | string | No | Internal notes |

### Example Request

```bash
curl -X POST https://api.linktor.io/v1/conversations/conv_abc123/close \
  -H "Authorization: Bearer YOUR_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "reason": "Issue resolved",
    "resolution": "resolved",
    "notes": "Customer was happy with the refund"
  }'
```

### Response

```json
{
  "data": {
    "id": "conv_abc123",
    "type": "conversation",
    "attributes": {
      "status": "closed",
      "closedAt": "2024-01-15T12:00:00Z",
      "closedBy": {
        "id": "user_456",
        "name": "Support Agent"
      },
      "resolution": "resolved",
      "closureReason": "Issue resolved"
    }
  }
}
```

---

## Reopen Conversation

Reopen a closed conversation.

```
POST /conversations/:id/reopen
```

### Path Parameters

| Parameter | Type | Description |
|-----------|------|-------------|
| `id` | string | Conversation ID |

### Example Request

```bash
curl -X POST https://api.linktor.io/v1/conversations/conv_abc123/reopen \
  -H "Authorization: Bearer YOUR_API_KEY"
```

### Response

```json
{
  "data": {
    "id": "conv_abc123",
    "type": "conversation",
    "attributes": {
      "status": "open",
      "reopenedAt": "2024-01-15T14:00:00Z",
      "reopenedBy": {
        "id": "user_456",
        "name": "Support Agent"
      }
    }
  }
}
```

---

## Assign Conversation

Assign a conversation to a user or bot.

```
POST /conversations/:id/assign
```

### Path Parameters

| Parameter | Type | Description |
|-----------|------|-------------|
| `id` | string | Conversation ID |

### Request Body

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `assigneeId` | string | Yes | User ID or Bot ID |
| `assigneeType` | string | No | `user` or `bot` (auto-detected) |
| `note` | string | No | Assignment note |

### Example Request

```bash
curl -X POST https://api.linktor.io/v1/conversations/conv_abc123/assign \
  -H "Authorization: Bearer YOUR_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "assigneeId": "user_789",
    "note": "Escalated to senior support"
  }'
```

### Response

```json
{
  "data": {
    "id": "conv_abc123",
    "type": "conversation",
    "attributes": {
      "assignee": {
        "id": "user_789",
        "type": "user",
        "name": "Senior Support Agent"
      },
      "assignedAt": "2024-01-15T12:30:00Z",
      "assignedBy": {
        "id": "user_456",
        "name": "Support Agent"
      }
    }
  }
}
```

---

## Transfer Conversation

Transfer a conversation to another assignee with handoff context.

```
POST /conversations/:id/transfer
```

### Request Body

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `toAssigneeId` | string | Yes | New assignee ID |
| `reason` | string | No | Transfer reason |
| `summary` | string | No | Conversation summary for new assignee |
| `notifyContact` | boolean | No | Send transfer notification to contact |

### Example Request

```bash
curl -X POST https://api.linktor.io/v1/conversations/conv_abc123/transfer \
  -H "Authorization: Bearer YOUR_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "toAssigneeId": "user_expert",
    "reason": "Technical issue",
    "summary": "Customer has a billing dispute from last month. Needs access to invoice #12345.",
    "notifyContact": true
  }'
```

---

## Get Participants

List all participants in a conversation.

```
GET /conversations/:id/participants
```

### Example Request

```bash
curl https://api.linktor.io/v1/conversations/conv_abc123/participants \
  -H "Authorization: Bearer YOUR_API_KEY"
```

### Response

```json
{
  "data": [
    {
      "id": "contact_123",
      "type": "contact",
      "attributes": {
        "name": "John Doe",
        "phone": "+5511999999999",
        "role": "customer"
      }
    },
    {
      "id": "user_456",
      "type": "user",
      "attributes": {
        "name": "Support Agent",
        "email": "agent@company.com",
        "role": "agent"
      }
    },
    {
      "id": "bot_abc",
      "type": "bot",
      "attributes": {
        "name": "Support Bot",
        "role": "bot"
      }
    }
  ]
}
```

---

## Conversation Statuses

| Status | Description |
|--------|-------------|
| `open` | Active conversation awaiting response |
| `pending` | Waiting for customer response |
| `snoozed` | Temporarily hidden, will reopen at scheduled time |
| `closed` | Conversation resolved and closed |

---

## Webhooks

Conversation events can be received via webhooks:

- `conversation.created` - New conversation started
- `conversation.updated` - Conversation properties changed
- `conversation.assigned` - Conversation assigned to user/bot
- `conversation.transferred` - Conversation transferred
- `conversation.closed` - Conversation closed
- `conversation.reopened` - Conversation reopened

See [Webhooks](/api/webhooks) for configuration details.
