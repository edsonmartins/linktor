---
sidebar_position: 2
title: TypeScript SDK
---

# TypeScript SDK

The official TypeScript SDK for Linktor provides a fully-typed interface to the Linktor API with WebSocket support for real-time communication.

## Installation

Install the SDK using npm, yarn, or pnpm:

```bash
# npm
npm install @linktor/sdk

# yarn
yarn add @linktor/sdk

# pnpm
pnpm add @linktor/sdk
```

## Quick Start

### Initialize the Client

```typescript
import { LinktorClient } from '@linktor/sdk'

const client = new LinktorClient({
  apiKey: process.env.LINKTOR_API_KEY,
  // Optional configuration
  baseUrl: 'https://api.linktor.io',
  timeout: 30000,
  maxRetries: 3,
})
```

### Send a Message

```typescript
// Send a message to a conversation
const message = await client.conversations.sendMessage('conversation-id', {
  text: 'Hello from TypeScript!',
})

console.log('Message sent:', message.id)
```

### List Conversations

```typescript
// Get all conversations with pagination
const conversations = await client.conversations.list({
  limit: 20,
  status: 'open',
})

for (const conversation of conversations.data) {
  console.log(`${conversation.id}: ${conversation.contact.name}`)
}
```

### Work with Contacts

```typescript
// Create a new contact
const contact = await client.contacts.create({
  name: 'John Doe',
  email: 'john@example.com',
  phone: '+1234567890',
  metadata: {
    customerId: 'cust_123',
  },
})

// Search contacts
const results = await client.contacts.list({
  search: 'john',
  limit: 10,
})
```

## Real-time Updates (WebSocket)

The SDK includes built-in WebSocket support for receiving real-time updates.

### Connect and Subscribe

```typescript
import { LinktorClient } from '@linktor/sdk'

const client = new LinktorClient({
  apiKey: process.env.LINKTOR_API_KEY,
})

// Connect to WebSocket
await client.ws.connect()

// Subscribe to a conversation
client.ws.subscribe('conversation-id')

// Listen for new messages
client.ws.onMessage((event) => {
  console.log('New message:', event.message.text)
  console.log('From conversation:', event.conversationId)
})

// Listen for message status updates
client.ws.onMessageStatus((event) => {
  console.log(`Message ${event.messageId} status: ${event.status}`)
})

// Listen for typing indicators
client.ws.onTyping((event) => {
  if (event.isTyping) {
    console.log(`User ${event.userId} is typing...`)
  }
})

// Send typing indicator
client.ws.sendTyping('conversation-id', true)

// Unsubscribe when done
client.ws.unsubscribe('conversation-id')

// Disconnect
client.ws.disconnect()
```

### Event Handlers

```typescript
// Connection events
client.ws.on('connected', () => {
  console.log('WebSocket connected')
})

client.ws.on('disconnected', ({ code, reason }) => {
  console.log(`WebSocket disconnected: ${code} - ${reason}`)
})

client.ws.on('error', (error) => {
  console.error('WebSocket error:', error)
})

// Conversation updates
client.ws.onConversationUpdate((event) => {
  console.log(`Conversation ${event.conversationId} updated`)
  if (event.status) console.log(`New status: ${event.status}`)
  if (event.assignedTo) console.log(`Assigned to: ${event.assignedTo}`)
})
```

## Error Handling

The SDK provides typed error classes for different error scenarios.

```typescript
import {
  LinktorClient,
  LinktorError,
  AuthenticationError,
  NotFoundError,
  RateLimitError,
  ValidationError,
} from '@linktor/sdk'

const client = new LinktorClient({
  apiKey: process.env.LINKTOR_API_KEY,
})

try {
  const conversation = await client.conversations.get('invalid-id')
} catch (error) {
  if (error instanceof NotFoundError) {
    console.error('Conversation not found')
  } else if (error instanceof AuthenticationError) {
    console.error('Invalid API key')
  } else if (error instanceof RateLimitError) {
    console.error(`Rate limited. Retry after ${error.retryAfter} seconds`)
  } else if (error instanceof ValidationError) {
    console.error('Invalid request:', error.details)
  } else if (error instanceof LinktorError) {
    console.error(`API error [${error.code}]: ${error.message}`)
    console.error('Request ID:', error.requestId)
  }
}
```

### Error Types

| Error Class | Status Code | Description |
|-------------|-------------|-------------|
| `AuthenticationError` | 401 | Invalid or missing API key |
| `AuthorizationError` | 403 | Insufficient permissions |
| `NotFoundError` | 404 | Resource not found |
| `ValidationError` | 400 | Invalid request parameters |
| `RateLimitError` | 429 | Too many requests |
| `ConflictError` | 409 | Resource conflict |
| `ServerError` | 5xx | Server-side error |
| `NetworkError` | - | Network connectivity issue |
| `TimeoutError` | 408 | Request timeout |
| `WebSocketError` | - | WebSocket connection error |

## Webhook Verification

Verify incoming webhooks to ensure they're from Linktor.

```typescript
import { LinktorClient } from '@linktor/sdk'
import express from 'express'

const app = express()

app.post('/webhook', express.raw({ type: 'application/json' }), (req, res) => {
  const signature = req.headers['x-linktor-signature'] as string
  const timestamp = req.headers['x-linktor-timestamp'] as string

  const client = new LinktorClient({
    apiKey: process.env.LINKTOR_API_KEY,
  })

  try {
    // Verify and parse the webhook event
    const event = client.webhooks.constructEvent(
      req.body,
      signature,
      timestamp,
      process.env.WEBHOOK_SECRET!
    )

    // Handle different event types
    if (client.webhooks.isEventType(event, 'message.received')) {
      console.log('New message received:', event.data)
    } else if (client.webhooks.isEventType(event, 'conversation.created')) {
      console.log('New conversation:', event.data)
    }

    res.json({ received: true })
  } catch (error) {
    console.error('Webhook verification failed:', error)
    res.status(400).send('Invalid signature')
  }
})
```

## AI Features

### Completions

```typescript
// Simple completion
const response = await client.ai.completions.create({
  messages: [
    { role: 'system', content: 'You are a helpful assistant.' },
    { role: 'user', content: 'What is the capital of France?' },
  ],
  model: 'gpt-4',
})

console.log(response.message.content)
```

### Knowledge Bases

```typescript
// Query a knowledge base
const results = await client.knowledgeBases.query('kb-id', {
  query: 'How do I reset my password?',
  topK: 5,
})

for (const chunk of results.chunks) {
  console.log('Match:', chunk.content)
  console.log('Score:', chunk.score)
}
```

## Resources

The SDK provides access to all Linktor resources:

- `client.auth` - Authentication
- `client.conversations` - Conversations and messages
- `client.contacts` - Contact management
- `client.channels` - Channel configuration
- `client.bots` - Bot management
- `client.ai` - AI completions and embeddings
- `client.knowledgeBases` - Knowledge base operations
- `client.flows` - Conversation flows
- `client.analytics` - Analytics and metrics

## API Reference

For complete API documentation, see the [TypeScript SDK Reference](https://github.com/linktor/linktor/tree/main/sdks/typescript).
