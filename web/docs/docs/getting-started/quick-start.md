---
sidebar_position: 2
title: Quick Start
---

# Quick Start

Get your first chatbot running in under 5 minutes.

## 1. Access the Dashboard

After [installation](/getting-started/installation), open the admin dashboard:

```
http://localhost:3000
```

Create an account or log in with the default credentials:
- Email: `admin@linktor.io`
- Password: `admin123`

## 2. Create a Channel

Navigate to **Channels** and click **Add Channel**.

For testing, the easiest option is **WebChat**:

1. Click **WebChat**
2. Enter a name: "My Website Chat"
3. Configure colors and appearance
4. Click **Create Channel**

Copy the embed code to add the widget to your website.

## 3. Create a Bot

Navigate to **Bots** and click **Create Bot**:

1. Enter a name: "Support Bot"
2. Select an AI provider (OpenAI, Anthropic, or Ollama)
3. Configure the system prompt:

```
You are a helpful customer support assistant for Acme Corp.
You help customers with product questions, orders, and general inquiries.
Be friendly, professional, and concise.
```

4. Set temperature: 0.7
5. Link the bot to your WebChat channel
6. Click **Create Bot**

## 4. Test the Bot

Open the embedded WebChat widget and send a message:

```
Hi, I need help with my order
```

The bot will respond using AI!

## 5. Create a Flow (Optional)

For more control, create a conversation flow:

1. Navigate to **Flows** → **Create Flow**
2. Drag a **Message** node to send a greeting
3. Add a **Question** node to collect information
4. Add **Condition** nodes for branching logic
5. Connect the nodes
6. Click **Save** and **Activate**

## Using the API

You can also interact programmatically:

```bash
# Get an API key from Settings → API Keys

# Send a message
curl -X POST http://localhost:8080/api/v1/messages \
  -H "Authorization: Bearer YOUR_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "channel_id": "your-channel-id",
    "to": "user@example.com",
    "content": "Hello from the API!"
  }'
```

## Using an SDK

Or use one of our SDKs:

```typescript
import { Linktor } from '@linktor/sdk'

const client = new Linktor({
  apiKey: process.env.LINKTOR_API_KEY
})

// Send a message
await client.messages.send({
  channelId: 'your-channel-id',
  to: '+1234567890',
  content: 'Hello from TypeScript!'
})
```

## Next Steps

- [Channels](/channels/overview) - Connect WhatsApp, Telegram, and more
- [Bots](/bots/overview) - Configure AI settings
- [Flows](/flows/overview) - Build conversation flows
- [SDKs](/sdks/overview) - Integrate with your application
