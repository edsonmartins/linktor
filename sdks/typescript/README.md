# Linktor SDK for TypeScript/JavaScript

Official Linktor SDK for TypeScript and JavaScript applications. Works with Node.js and browsers.

## Installation

```bash
npm install @linktor/sdk
# or
yarn add @linktor/sdk
# or
pnpm add @linktor/sdk
```

## Quick Start

```typescript
import { LinktorClient } from '@linktor/sdk';

const client = new LinktorClient({
  baseUrl: 'https://api.linktor.io',
  apiKey: 'your-api-key',
});

// List conversations
const conversations = await client.conversations.list({ limit: 10 });

// Send a message
await client.conversations.sendMessage(conversationId, {
  text: 'Hello!',
});

// Use AI
const response = await client.ai.completions.complete('What is 2 + 2?');
console.log(response); // "4"
```

## Authentication

### API Key (Server-side)

```typescript
const client = new LinktorClient({
  apiKey: 'your-api-key',
});
```

### Access Token (Client-side)

```typescript
const client = new LinktorClient({
  accessToken: 'user-access-token',
});

// Auto-refresh token
const client = new LinktorClient({
  accessToken: 'initial-token',
  onTokenRefresh: async () => {
    const response = await fetch('/api/refresh-token');
    const { accessToken } = await response.json();
    return accessToken;
  },
});
```

### Login

```typescript
const client = new LinktorClient({ baseUrl: 'https://api.linktor.io' });

const { accessToken, user } = await client.auth.login({
  email: 'user@example.com',
  password: 'password',
});
```

## Resources

### Conversations

```typescript
// List conversations
const convs = await client.conversations.list({
  status: 'open',
  limit: 20,
});

// Get a conversation
const conv = await client.conversations.get('conv-id');

// Send text message
await client.conversations.sendText('conv-id', 'Hello!');

// Send image
await client.conversations.sendImage('conv-id', 'https://example.com/image.jpg', 'Caption');

// Assign to agent
await client.conversations.assign('conv-id', 'agent-id');

// Resolve conversation
await client.conversations.resolve('conv-id');

// Iterate all conversations
for await (const conv of client.conversations.iterate()) {
  console.log(conv.id);
}
```

### Contacts

```typescript
// Create contact
const contact = await client.contacts.create({
  name: 'John Doe',
  email: 'john@example.com',
  phone: '+1234567890',
});

// Search contacts
const results = await client.contacts.search('john');

// Find by email
const contact = await client.contacts.findByEmail('john@example.com');

// Merge contacts
await client.contacts.merge({
  primaryContactId: 'contact-1',
  secondaryContactIds: ['contact-2', 'contact-3'],
});
```

### Channels

```typescript
// List channels
const channels = await client.channels.list();

// Create channel
const channel = await client.channels.create({
  name: 'WhatsApp Business',
  type: 'whatsapp',
  config: {
    type: 'whatsapp',
    phoneNumberId: '123456789',
    businessAccountId: '987654321',
    accessToken: 'token',
    verifyToken: 'verify',
  },
});

// Connect channel
await client.channels.connect('channel-id');

// Get status
const status = await client.channels.getStatus('channel-id');
```

### Bots

```typescript
// Create AI bot
const bot = await client.bots.create({
  name: 'Support Bot',
  type: 'ai',
  config: {
    welcomeMessage: 'Hello! How can I help?',
    aiConfig: {
      model: 'gpt-4',
      systemPrompt: 'You are a helpful support agent.',
      useKnowledgeBase: true,
    },
  },
  knowledgeBaseIds: ['kb-id'],
});

// Assign to channels
await client.bots.assignToChannels('bot-id', ['channel-1', 'channel-2']);

// Activate
await client.bots.activate('bot-id');
```

### AI

```typescript
// Simple completion
const answer = await client.ai.completions.complete('What is the capital of France?');

// Streaming completion
for await (const chunk of client.ai.completions.completeStream('Tell me a story')) {
  process.stdout.write(chunk);
}

// Chat completion
const response = await client.ai.completions.create({
  model: 'gpt-4',
  messages: [
    { role: 'system', content: 'You are a helpful assistant.' },
    { role: 'user', content: 'Hello!' },
  ],
});

// Invoke agent
const result = await client.ai.agents.invoke('agent-id', {
  message: 'How do I reset my password?',
  conversationId: 'conv-id',
});

// Create embeddings
const embeddings = await client.ai.embeddings.embed('Hello world');
```

### Knowledge Bases

```typescript
// Create knowledge base
const kb = await client.knowledgeBases.create({
  name: 'Product Documentation',
  description: 'Product docs and FAQs',
});

// Upload document
await client.knowledgeBases.uploadDocument('kb-id', {
  name: 'FAQ.pdf',
  file: pdfFile, // File or Blob
});

// Query knowledge base
const results = await client.knowledgeBases.query('kb-id', {
  query: 'How to reset password?',
  topK: 5,
});

// Simple search
const texts = await client.knowledgeBases.search('kb-id', 'reset password');
```

### Flows

```typescript
// Create flow
const flow = await client.flows.create({
  name: 'Welcome Flow',
  nodes: [
    { id: 'start', type: 'start', position: { x: 0, y: 0 }, data: {} },
    {
      id: 'welcome',
      type: 'message',
      position: { x: 200, y: 0 },
      data: { messageContent: 'Welcome!' },
    },
  ],
  edges: [{ id: 'e1', source: 'start', target: 'welcome' }],
});

// Execute flow
const execution = await client.flows.execute('flow-id', {
  conversationId: 'conv-id',
  variables: { userName: 'John' },
});

// Get execution status
const status = await client.flows.getExecution('flow-id', execution.id);
```

### Analytics

```typescript
// Dashboard metrics
const dashboard = await client.analytics.getDashboard({
  startDate: '2024-01-01',
  endDate: '2024-01-31',
});

// Conversation metrics
const convMetrics = await client.analytics.getConversationMetrics();

// Real-time metrics
const realtime = await client.analytics.getRealtime();

// Export data
const { downloadUrl } = await client.analytics.export({
  type: 'conversations',
  range: { startDate: '2024-01-01', endDate: '2024-01-31' },
  format: 'csv',
});
```

## WebSocket

Real-time updates via WebSocket.

```typescript
// Connect
await client.ws.connect();

// Subscribe to conversation
client.ws.subscribe('conv-id');

// Handle messages
client.ws.onMessage((event) => {
  console.log('New message:', event.message);
});

// Handle status updates
client.ws.onMessageStatus((event) => {
  console.log('Message status:', event.status);
});

// Handle typing
client.ws.onTyping((event) => {
  console.log(`${event.userId} is ${event.isTyping ? 'typing' : 'not typing'}`);
});

// Send typing indicator
client.ws.sendTyping('conv-id', true);

// Disconnect
client.ws.disconnect();
```

## Webhooks

Verify and handle incoming webhooks.

```typescript
import { verifyWebhookSignature, constructEvent } from '@linktor/sdk';

// Express middleware
app.post('/webhook', express.raw({ type: 'application/json' }), (req, res) => {
  const signature = req.headers['x-linktor-signature'];
  const secret = process.env.WEBHOOK_SECRET;

  try {
    const event = constructEvent(req.body, req.headers, secret);

    switch (event.type) {
      case 'message.received':
        console.log('New message:', event.data);
        break;
      case 'conversation.created':
        console.log('New conversation:', event.data);
        break;
    }

    res.json({ received: true });
  } catch (err) {
    res.status(400).send(`Webhook Error: ${err.message}`);
  }
});

// Or use the handler factory
import { createWebhookHandler } from '@linktor/sdk';

const handler = createWebhookHandler(process.env.WEBHOOK_SECRET, {
  'message.received': (event) => {
    console.log('New message:', event.data);
  },
  'conversation.created': (event) => {
    console.log('New conversation:', event.data);
  },
});

app.post('/webhook', express.raw({ type: 'application/json' }), async (req, res) => {
  const result = await handler(req);
  res.status(result.status).send(result.body);
});
```

## Error Handling

```typescript
import {
  LinktorError,
  AuthenticationError,
  RateLimitError,
  ValidationError,
} from '@linktor/sdk';

try {
  await client.conversations.get('invalid-id');
} catch (error) {
  if (error instanceof AuthenticationError) {
    console.log('Please login again');
  } else if (error instanceof RateLimitError) {
    console.log(`Rate limited. Retry after ${error.retryAfter}s`);
  } else if (error instanceof ValidationError) {
    console.log('Validation error:', error.details);
  } else if (error instanceof LinktorError) {
    console.log(`Error [${error.code}]: ${error.message}`);
  }
}
```

## Configuration

```typescript
const client = new LinktorClient({
  // Base URL
  baseUrl: 'https://api.linktor.io',

  // Authentication
  apiKey: 'your-api-key',
  // or
  accessToken: 'user-token',

  // Request options
  timeout: 30000, // 30 seconds
  maxRetries: 3,
  retryDelay: 1000, // 1 second

  // Custom headers
  headers: {
    'X-Custom-Header': 'value',
  },

  // Token refresh callback
  onTokenRefresh: async () => {
    const { accessToken } = await refreshToken();
    return accessToken;
  },

  // WebSocket options
  websocket: {
    autoReconnect: true,
    reconnectInterval: 5000,
    maxReconnectAttempts: 10,
    pingInterval: 30000,
  },
});
```

## TypeScript

The SDK is written in TypeScript and provides full type definitions.

```typescript
import type {
  Conversation,
  Message,
  Contact,
  Channel,
  Bot,
  Agent,
  KnowledgeBase,
  Flow,
} from '@linktor/sdk';

const conversation: Conversation = await client.conversations.get('id');
```

## License

MIT
