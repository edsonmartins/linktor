---
sidebar_position: 6
title: Bots
---

# Bots

The Bots API allows you to create and manage AI-powered chatbots that handle conversations automatically.

## Overview

Bots in Linktor are AI agents that can respond to customer messages, execute actions, and escalate to human agents when needed. Bots can be powered by various AI providers (OpenAI, Anthropic, custom LLMs) and can be configured with custom instructions, knowledge bases, and conversation flows.

## Endpoints

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/bots` | List all bots |
| GET | `/bots/:id` | Get a specific bot |
| POST | `/bots` | Create a new bot |
| PATCH | `/bots/:id` | Update a bot |
| DELETE | `/bots/:id` | Delete a bot |
| POST | `/bots/:id/activate` | Activate a bot |
| POST | `/bots/:id/deactivate` | Deactivate a bot |
| POST | `/bots/:id/test` | Test bot with a message |
| GET | `/bots/:id/stats` | Get bot statistics |
| POST | `/bots/:id/train` | Trigger knowledge base training |

---

## List Bots

Retrieve all bots for your organization.

```
GET /bots
```

### Query Parameters

| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `page` | integer | 1 | Page number |
| `perPage` | integer | 20 | Items per page (max: 100) |
| `status` | string | - | Filter by status: `active`, `inactive`, `training` |
| `provider` | string | - | Filter by AI provider |
| `search` | string | - | Search by bot name |

### Example Request

```bash
curl "https://api.linktor.io/v1/bots?status=active" \
  -H "Authorization: Bearer YOUR_API_KEY"
```

### Response

```json
{
  "data": [
    {
      "id": "bot_abc123",
      "type": "bot",
      "attributes": {
        "name": "Support Bot",
        "description": "Handles customer support inquiries",
        "status": "active",
        "provider": "openai",
        "model": "gpt-4",
        "stats": {
          "conversationsHandled24h": 150,
          "messagesProcessed24h": 1200,
          "avgResponseTime": 1.5,
          "handoffRate": 0.12
        },
        "channels": [
          {
            "id": "ch_xyz789",
            "name": "WhatsApp Business"
          }
        ],
        "createdAt": "2024-01-01T00:00:00Z",
        "updatedAt": "2024-01-15T10:00:00Z"
      }
    }
  ],
  "meta": {
    "pagination": {
      "page": 1,
      "perPage": 20,
      "totalPages": 1,
      "totalCount": 1
    }
  }
}
```

---

## Get Bot

Retrieve a specific bot by ID.

```
GET /bots/:id
```

### Path Parameters

| Parameter | Type | Description |
|-----------|------|-------------|
| `id` | string | Bot ID |

### Example Request

```bash
curl https://api.linktor.io/v1/bots/bot_abc123 \
  -H "Authorization: Bearer YOUR_API_KEY"
```

### Response

```json
{
  "data": {
    "id": "bot_abc123",
    "type": "bot",
    "attributes": {
      "name": "Support Bot",
      "description": "Handles customer support inquiries",
      "status": "active",
      "provider": "openai",
      "model": "gpt-4",
      "config": {
        "systemPrompt": "You are a helpful customer support agent for Acme Corp. Be friendly, professional, and concise.",
        "temperature": 0.7,
        "maxTokens": 500,
        "topP": 1.0,
        "frequencyPenalty": 0,
        "presencePenalty": 0
      },
      "behaviors": {
        "greeting": {
          "enabled": true,
          "message": "Hello! I'm the Acme support assistant. How can I help you today?"
        },
        "fallback": {
          "enabled": true,
          "message": "I'm not sure I understand. Could you rephrase that?",
          "maxRetries": 2
        },
        "handoff": {
          "enabled": true,
          "triggers": ["speak to human", "agent", "representative"],
          "message": "I'll connect you with a human agent. Please wait a moment."
        },
        "offlineMessage": {
          "enabled": true,
          "message": "Our support team is currently offline. Please leave a message and we'll get back to you."
        }
      },
      "knowledgeBase": {
        "id": "kb_xyz789",
        "name": "Product Knowledge Base",
        "documentCount": 150,
        "lastTrainedAt": "2024-01-14T00:00:00Z"
      },
      "flow": {
        "id": "flow_abc",
        "name": "Support Flow"
      },
      "capabilities": {
        "textResponses": true,
        "imageGeneration": false,
        "codeExecution": false,
        "webSearch": false,
        "functionCalling": true
      },
      "functions": [
        {
          "name": "get_order_status",
          "description": "Get the status of a customer order",
          "parameters": {
            "type": "object",
            "properties": {
              "orderId": {
                "type": "string",
                "description": "The order ID"
              }
            },
            "required": ["orderId"]
          }
        },
        {
          "name": "create_ticket",
          "description": "Create a support ticket",
          "parameters": {
            "type": "object",
            "properties": {
              "subject": {"type": "string"},
              "priority": {"type": "string", "enum": ["low", "normal", "high"]}
            },
            "required": ["subject"]
          }
        }
      ],
      "channels": [
        {
          "id": "ch_xyz789",
          "name": "WhatsApp Business",
          "type": "whatsapp"
        }
      ],
      "stats": {
        "conversationsHandled24h": 150,
        "conversationsHandledTotal": 15000,
        "messagesProcessed24h": 1200,
        "messagesProcessedTotal": 120000,
        "avgResponseTime": 1.5,
        "avgConversationLength": 8,
        "handoffRate": 0.12,
        "satisfactionScore": 4.5
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
| 404 | `BOT_NOT_FOUND` | Bot does not exist |

---

## Create Bot

Create a new bot.

```
POST /bots
```

### Request Body

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `name` | string | Yes | Bot display name |
| `description` | string | No | Bot description |
| `provider` | string | Yes | AI provider: `openai`, `anthropic`, `custom` |
| `model` | string | Yes | Model identifier |
| `config` | object | No | Model configuration |
| `behaviors` | object | No | Bot behavior settings |
| `knowledgeBaseId` | string | No | Knowledge base to use |
| `flowId` | string | No | Conversation flow to use |
| `functions` | array | No | Custom functions |
| `channelIds` | array | No | Channels to assign bot to |

### Example Request

```bash
curl -X POST https://api.linktor.io/v1/bots \
  -H "Authorization: Bearer YOUR_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Sales Bot",
    "description": "Handles product inquiries and sales",
    "provider": "openai",
    "model": "gpt-4",
    "config": {
      "systemPrompt": "You are a helpful sales assistant for Acme Corp. Help customers find the right products and answer questions about pricing and features. Be friendly and persuasive but not pushy.",
      "temperature": 0.8,
      "maxTokens": 600
    },
    "behaviors": {
      "greeting": {
        "enabled": true,
        "message": "Hi there! Welcome to Acme. What are you looking for today?"
      },
      "handoff": {
        "enabled": true,
        "triggers": ["buy", "purchase", "pricing for enterprise"],
        "message": "Great choice! Let me connect you with our sales team."
      }
    },
    "knowledgeBaseId": "kb_products",
    "channelIds": ["ch_webchat"]
  }'
```

### Response

```json
{
  "data": {
    "id": "bot_new456",
    "type": "bot",
    "attributes": {
      "name": "Sales Bot",
      "description": "Handles product inquiries and sales",
      "status": "inactive",
      "provider": "openai",
      "model": "gpt-4",
      "config": {
        "systemPrompt": "You are a helpful sales assistant...",
        "temperature": 0.8,
        "maxTokens": 600
      },
      "behaviors": {
        "greeting": {
          "enabled": true,
          "message": "Hi there! Welcome to Acme. What are you looking for today?"
        },
        "handoff": {
          "enabled": true,
          "triggers": ["buy", "purchase", "pricing for enterprise"],
          "message": "Great choice! Let me connect you with our sales team."
        }
      },
      "knowledgeBase": {
        "id": "kb_products",
        "name": "Product Catalog"
      },
      "channels": [
        {
          "id": "ch_webchat",
          "name": "Website Chat"
        }
      ],
      "createdAt": "2024-01-15T11:00:00Z"
    }
  }
}
```

### Error Codes

| HTTP Status | Error Code | Description |
|-------------|------------|-------------|
| 400 | `VALIDATION_ERROR` | Invalid configuration |
| 400 | `INVALID_PROVIDER` | Unsupported AI provider |
| 400 | `INVALID_MODEL` | Model not available for provider |
| 404 | `KNOWLEDGE_BASE_NOT_FOUND` | Knowledge base not found |
| 404 | `FLOW_NOT_FOUND` | Flow not found |

---

## Update Bot

Update an existing bot.

```
PATCH /bots/:id
```

### Path Parameters

| Parameter | Type | Description |
|-----------|------|-------------|
| `id` | string | Bot ID |

### Request Body

| Field | Type | Description |
|-------|------|-------------|
| `name` | string | Bot display name |
| `description` | string | Bot description |
| `model` | string | Model identifier |
| `config` | object | Model configuration |
| `behaviors` | object | Behavior settings |
| `knowledgeBaseId` | string | Knowledge base ID |
| `flowId` | string | Flow ID |
| `functions` | array | Custom functions |
| `channelIds` | array | Channel assignments |

### Example Request

```bash
curl -X PATCH https://api.linktor.io/v1/bots/bot_abc123 \
  -H "Authorization: Bearer YOUR_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "config": {
      "systemPrompt": "Updated system prompt with new instructions...",
      "temperature": 0.5
    },
    "behaviors": {
      "handoff": {
        "enabled": true,
        "triggers": ["agent", "human", "help me please"],
        "message": "Connecting you to a human agent now."
      }
    }
  }'
```

### Response

```json
{
  "data": {
    "id": "bot_abc123",
    "type": "bot",
    "attributes": {
      "name": "Support Bot",
      "config": {
        "systemPrompt": "Updated system prompt with new instructions...",
        "temperature": 0.5,
        "maxTokens": 500
      },
      "behaviors": {
        "handoff": {
          "enabled": true,
          "triggers": ["agent", "human", "help me please"],
          "message": "Connecting you to a human agent now."
        }
      },
      "updatedAt": "2024-01-15T12:00:00Z"
    }
  }
}
```

---

## Delete Bot

Delete a bot. The bot must be deactivated first.

```
DELETE /bots/:id
```

### Path Parameters

| Parameter | Type | Description |
|-----------|------|-------------|
| `id` | string | Bot ID |

### Example Request

```bash
curl -X DELETE https://api.linktor.io/v1/bots/bot_abc123 \
  -H "Authorization: Bearer YOUR_API_KEY"
```

### Response

```
HTTP/1.1 204 No Content
```

### Error Codes

| HTTP Status | Error Code | Description |
|-------------|------------|-------------|
| 400 | `BOT_IS_ACTIVE` | Bot must be deactivated first |
| 404 | `BOT_NOT_FOUND` | Bot does not exist |

---

## Activate Bot

Enable a bot to start handling conversations.

```
POST /bots/:id/activate
```

### Path Parameters

| Parameter | Type | Description |
|-----------|------|-------------|
| `id` | string | Bot ID |

### Example Request

```bash
curl -X POST https://api.linktor.io/v1/bots/bot_abc123/activate \
  -H "Authorization: Bearer YOUR_API_KEY"
```

### Response

```json
{
  "data": {
    "id": "bot_abc123",
    "type": "bot",
    "attributes": {
      "status": "active",
      "activatedAt": "2024-01-15T12:00:00Z"
    }
  }
}
```

---

## Deactivate Bot

Disable a bot. Active conversations will be transferred to fallback handling.

```
POST /bots/:id/deactivate
```

### Path Parameters

| Parameter | Type | Description |
|-----------|------|-------------|
| `id` | string | Bot ID |

### Request Body

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `transferTo` | string | No | User/bot ID to transfer conversations |
| `reason` | string | No | Deactivation reason |

### Example Request

```bash
curl -X POST https://api.linktor.io/v1/bots/bot_abc123/deactivate \
  -H "Authorization: Bearer YOUR_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "transferTo": "bot_fallback",
    "reason": "Updating knowledge base"
  }'
```

### Response

```json
{
  "data": {
    "id": "bot_abc123",
    "type": "bot",
    "attributes": {
      "status": "inactive",
      "deactivatedAt": "2024-01-15T12:00:00Z",
      "conversationsTransferred": 5
    }
  }
}
```

---

## Test Bot

Send a test message to the bot and get a response.

```
POST /bots/:id/test
```

### Path Parameters

| Parameter | Type | Description |
|-----------|------|-------------|
| `id` | string | Bot ID |

### Request Body

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `message` | string | Yes | Test message |
| `context` | object | No | Conversation context |
| `context.history` | array | No | Previous messages |
| `context.metadata` | object | No | Contact/conversation metadata |

### Example Request

```bash
curl -X POST https://api.linktor.io/v1/bots/bot_abc123/test \
  -H "Authorization: Bearer YOUR_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "message": "What are your business hours?",
    "context": {
      "history": [
        {"role": "user", "content": "Hi there"},
        {"role": "assistant", "content": "Hello! How can I help you today?"}
      ],
      "metadata": {
        "customerTier": "premium"
      }
    }
  }'
```

### Response

```json
{
  "data": {
    "response": {
      "content": "Our business hours are Monday to Friday, 9 AM to 6 PM EST. On weekends, we have limited support from 10 AM to 2 PM. Is there anything specific you need help with during these hours?",
      "confidence": 0.95,
      "intent": "business_hours_inquiry",
      "sources": [
        {
          "type": "knowledge_base",
          "documentId": "doc_hours",
          "excerpt": "Business hours: Mon-Fri 9AM-6PM EST..."
        }
      ]
    },
    "usage": {
      "promptTokens": 150,
      "completionTokens": 45,
      "totalTokens": 195
    },
    "latency": 1.2
  }
}
```

---

## Get Bot Statistics

Retrieve detailed statistics for a bot.

```
GET /bots/:id/stats
```

### Path Parameters

| Parameter | Type | Description |
|-----------|------|-------------|
| `id` | string | Bot ID |

### Query Parameters

| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `period` | string | `24h` | Time period: `1h`, `24h`, `7d`, `30d` |

### Example Request

```bash
curl "https://api.linktor.io/v1/bots/bot_abc123/stats?period=7d" \
  -H "Authorization: Bearer YOUR_API_KEY"
```

### Response

```json
{
  "data": {
    "botId": "bot_abc123",
    "period": "7d",
    "conversations": {
      "total": 1050,
      "resolved": 920,
      "handedOff": 130,
      "avgLength": 8.5,
      "avgDuration": 420
    },
    "messages": {
      "received": 8500,
      "sent": 7800,
      "avgResponseTime": 1.3
    },
    "performance": {
      "resolutionRate": 0.876,
      "handoffRate": 0.124,
      "satisfactionScore": 4.6,
      "feedbackCount": 250
    },
    "usage": {
      "totalTokens": 1500000,
      "promptTokens": 1000000,
      "completionTokens": 500000,
      "estimatedCost": 45.00
    },
    "topIntents": [
      {"intent": "order_status", "count": 320},
      {"intent": "product_inquiry", "count": 280},
      {"intent": "pricing", "count": 150}
    ],
    "topHandoffReasons": [
      {"reason": "complex_issue", "count": 50},
      {"reason": "customer_request", "count": 45},
      {"reason": "out_of_scope", "count": 35}
    ]
  }
}
```

---

## Train Knowledge Base

Trigger a knowledge base training/sync for the bot.

```
POST /bots/:id/train
```

### Path Parameters

| Parameter | Type | Description |
|-----------|------|-------------|
| `id` | string | Bot ID |

### Request Body

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `fullRetrain` | boolean | No | Force full retrain (default: incremental) |

### Example Request

```bash
curl -X POST https://api.linktor.io/v1/bots/bot_abc123/train \
  -H "Authorization: Bearer YOUR_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "fullRetrain": false
  }'
```

### Response

```json
{
  "data": {
    "trainingId": "train_xyz789",
    "botId": "bot_abc123",
    "status": "in_progress",
    "type": "incremental",
    "documentsToProcess": 15,
    "startedAt": "2024-01-15T12:00:00Z",
    "estimatedCompletion": "2024-01-15T12:05:00Z"
  }
}
```

---

## Supported Providers

| Provider | Models |
|----------|--------|
| `openai` | `gpt-4`, `gpt-4-turbo`, `gpt-3.5-turbo` |
| `anthropic` | `claude-3-opus`, `claude-3-sonnet`, `claude-3-haiku` |
| `custom` | Self-hosted or custom API endpoint |

---

## Bot Statuses

| Status | Description |
|--------|-------------|
| `active` | Bot is handling conversations |
| `inactive` | Bot is disabled |
| `training` | Knowledge base is being updated |
| `error` | Bot has configuration issues |

---

## Webhooks

Bot events can be received via webhooks:

- `bot.created` - New bot created
- `bot.updated` - Bot configuration changed
- `bot.activated` - Bot activated
- `bot.deactivated` - Bot deactivated
- `bot.handoff` - Bot handed off to human
- `bot.training.started` - Training started
- `bot.training.completed` - Training completed
- `bot.training.failed` - Training failed

See [Webhooks](/api/webhooks) for configuration details.
