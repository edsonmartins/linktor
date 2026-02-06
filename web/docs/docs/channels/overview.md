---
sidebar_position: 1
title: Channels Overview
---

# Channels Overview

Channels are the communication pathways that connect your Linktor platform to your customers. Each channel represents a different messaging platform or communication method through which users can interact with your bots and agents.

## What are Channels?

In Linktor, a **channel** is an integration with an external messaging platform. When you configure a channel, Linktor establishes a two-way connection that allows:

- **Inbound messages**: Receive messages from customers on their preferred platform
- **Outbound messages**: Send messages, notifications, and responses back to customers
- **Media handling**: Exchange images, documents, audio, and video files
- **Delivery tracking**: Monitor message delivery status and read receipts (where supported)

## Supported Channels

Linktor supports 10+ messaging channels out of the box:

| Channel | Type | Features |
|---------|------|----------|
| [WhatsApp](/channels/whatsapp) | Instant Messaging | Text, media, templates, buttons, lists |
| [Telegram](/channels/telegram) | Instant Messaging | Text, media, inline keyboards, commands |
| [SMS](/channels/sms) | Text Messaging | Text, MMS (carrier dependent) |
| [Email](/channels/email) | Email | HTML/text, attachments, threading |
| [Voice](/channels/voice) | Voice Calls | IVR, speech-to-text, call recording |
| [WebChat](/channels/webchat) | Web Widget | Text, media, typing indicators, customizable UI |
| [Facebook Messenger](/channels/facebook) | Social Media | Text, media, templates, persistent menu |
| [Instagram](/channels/instagram) | Social Media | Text, media, story mentions, ice breakers |
| [RCS](/channels/rcs) | Rich Messaging | Text, media, carousels, suggested actions |

## How Channels Work

### Architecture

```
┌─────────────────┐     ┌─────────────────┐     ┌─────────────────┐
│   Customer      │     │    Linktor      │     │   Your Bot/     │
│   Device        │◄───►│    Platform     │◄───►│   Agent         │
└─────────────────┘     └─────────────────┘     └─────────────────┘
        │                       │
        │    External API       │
        │   (WhatsApp, etc.)    │
        └───────────────────────┘
```

### Message Flow

1. **Inbound**: Customer sends a message on their platform → Platform webhook delivers to Linktor → Linktor normalizes and routes to your bot/flow
2. **Outbound**: Your bot/flow generates a response → Linktor formats for the target platform → Platform API delivers to customer

### Unified Message Format

Linktor normalizes messages from all channels into a unified format:

```json
{
  "id": "msg_abc123",
  "channelId": "ch_xyz789",
  "channelType": "whatsapp",
  "direction": "inbound",
  "from": "+5511999999999",
  "to": "+5511888888888",
  "content": {
    "type": "text",
    "text": "Hello, I need help with my order"
  },
  "timestamp": "2024-01-15T10:30:00Z",
  "metadata": {
    "profileName": "John Doe",
    "profilePicture": "https://..."
  }
}
```

## Channel Configuration

### Dashboard Setup

1. Navigate to **Settings → Channels** in the Linktor dashboard
2. Click **Add Channel** and select the channel type
3. Enter the required credentials and configuration
4. Configure webhook URLs (auto-generated or custom)
5. Assign the channel to a bot or inbox

### API Configuration

You can also configure channels programmatically:

```typescript
import { Linktor } from '@linktor/sdk'

const client = new Linktor({ apiKey: 'YOUR_API_KEY' })

// Create a WhatsApp channel
const channel = await client.channels.create({
  type: 'whatsapp',
  name: 'WhatsApp Business',
  config: {
    phoneNumberId: 'YOUR_PHONE_NUMBER_ID',
    accessToken: 'YOUR_ACCESS_TOKEN',
    verifyToken: 'YOUR_VERIFY_TOKEN'
  }
})
```

## Channel Capabilities

Different channels support different features. Here's a capability matrix:

| Feature | WhatsApp | Telegram | SMS | Email | Voice | WebChat | Facebook | Instagram | RCS |
|---------|:--------:|:--------:|:---:|:-----:|:-----:|:-------:|:--------:|:---------:|:---:|
| Text Messages | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ |
| Images | ✅ | ✅ | ⚠️ | ✅ | ❌ | ✅ | ✅ | ✅ | ✅ |
| Documents | ✅ | ✅ | ❌ | ✅ | ❌ | ✅ | ✅ | ❌ | ✅ |
| Audio | ✅ | ✅ | ❌ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ |
| Video | ✅ | ✅ | ❌ | ✅ | ❌ | ✅ | ✅ | ✅ | ✅ |
| Buttons | ✅ | ✅ | ❌ | ❌ | ❌ | ✅ | ✅ | ✅ | ✅ |
| Quick Replies | ✅ | ✅ | ❌ | ❌ | ❌ | ✅ | ✅ | ✅ | ✅ |
| Templates | ✅ | ❌ | ❌ | ✅ | ❌ | ❌ | ✅ | ❌ | ✅ |
| Delivery Receipts | ✅ | ✅ | ✅ | ⚠️ | ✅ | ✅ | ✅ | ✅ | ✅ |
| Read Receipts | ✅ | ✅ | ❌ | ⚠️ | ❌ | ✅ | ✅ | ✅ | ✅ |
| Typing Indicators | ✅ | ✅ | ❌ | ❌ | ❌ | ✅ | ✅ | ✅ | ❌ |

**Legend**: ✅ Supported | ⚠️ Partial/Provider dependent | ❌ Not supported

## Multi-Channel Strategies

### Channel Routing

Route conversations to different bots or agents based on channel:

```typescript
// In your flow configuration
{
  "triggers": [
    {
      "type": "channel",
      "channels": ["whatsapp", "telegram"],
      "action": "route_to_sales_bot"
    },
    {
      "type": "channel",
      "channels": ["email"],
      "action": "route_to_support_inbox"
    }
  ]
}
```

### Cross-Channel Conversations

Maintain conversation context when customers switch channels:

```typescript
// Get conversation across all channels for a contact
const conversations = await client.conversations.list({
  contactId: 'contact_123',
  includeAllChannels: true
})
```

## Best Practices

1. **Start with your customers**: Choose channels based on where your customers are, not just what's easiest to set up

2. **Consistent experience**: Maintain similar conversation flows across channels, adapting only for channel-specific features

3. **Graceful degradation**: When a feature isn't supported (e.g., buttons in SMS), provide a text-based alternative

4. **Channel-specific optimization**: Take advantage of unique features (WhatsApp templates, Telegram inline keyboards, etc.)

5. **Monitor channel health**: Set up alerts for webhook failures and API errors

## Next Steps

- [WhatsApp Integration](/channels/whatsapp) - Set up WhatsApp Business API
- [Telegram Integration](/channels/telegram) - Connect your Telegram bot
- [WebChat Widget](/channels/webchat) - Embed chat on your website
- [API Reference](/api/overview) - Full API documentation
