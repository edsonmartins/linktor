---
sidebar_position: 3
title: Telegram
---

# Telegram Bot API Integration

Connect your Telegram Bot to Linktor and reach millions of users on one of the world's most popular messaging platforms. Telegram offers rich features including inline keyboards, commands, and group chat support.

## Overview

The Telegram integration enables you to:

- Send and receive text, images, documents, audio, video, and stickers
- Create interactive experiences with inline keyboards and reply keyboards
- Support bot commands for quick actions
- Handle group chats and channel broadcasts
- Receive location, contact, and poll responses
- Support inline mode for cross-chat functionality

## Prerequisites

Before configuring Telegram in Linktor, you'll need:

1. **Telegram Account**: A personal Telegram account to create and manage bots
2. **Bot Token**: Obtained from [@BotFather](https://t.me/BotFather)
3. **Public URL**: A publicly accessible URL for webhook delivery (HTTPS required)

### Creating a Telegram Bot

1. Open Telegram and search for [@BotFather](https://t.me/BotFather)
2. Send `/newbot` command
3. Follow the prompts to:
   - Set a display name for your bot
   - Choose a username (must end in "bot", e.g., `mycompany_support_bot`)
4. Copy the **Bot Token** provided by BotFather
5. Optionally, set bot profile picture with `/setuserpic`

## Configuration in Linktor

### Step 1: Get Your Bot Token

The Bot Token from BotFather looks like:
```
7123456789:AAHxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx
```

Keep this token secure - anyone with it can control your bot.

### Step 2: Add Channel in Dashboard

1. Go to **Settings → Channels** in Linktor
2. Click **Add Channel** and select **Telegram**
3. Fill in the configuration:

| Field | Description |
|-------|-------------|
| **Name** | Display name for this channel (e.g., "Support Bot") |
| **Bot Token** | Your bot token from BotFather |
| **Bot Username** | Your bot's username (without @) |
| **Webhook Secret** | Optional secret for webhook validation |

4. Click **Save** to create the channel

### Step 3: Webhook Registration

Linktor automatically registers the webhook with Telegram. You can verify it's working:

```bash
curl "https://api.telegram.org/bot<TOKEN>/getWebhookInfo"
```

Expected response:
```json
{
  "ok": true,
  "result": {
    "url": "https://api.your-domain.com/webhooks/telegram/ch_telegram_123",
    "has_custom_certificate": false,
    "pending_update_count": 0
  }
}
```

## API Usage

### Sending Messages

#### Text Message

```typescript
import { Linktor } from '@linktor/sdk'

const client = new Linktor({ apiKey: 'YOUR_API_KEY' })

await client.messages.send({
  channelId: 'ch_telegram_123',
  to: '123456789', // Telegram user ID or chat ID
  content: {
    type: 'text',
    text: 'Hello! How can I help you today?'
  }
})
```

#### Text with Markdown

```typescript
await client.messages.send({
  channelId: 'ch_telegram_123',
  to: '123456789',
  content: {
    type: 'text',
    text: '*Bold* _italic_ `code` [link](https://example.com)',
    parseMode: 'MarkdownV2'
  }
})
```

#### Image Message

```typescript
await client.messages.send({
  channelId: 'ch_telegram_123',
  to: '123456789',
  content: {
    type: 'image',
    image: {
      url: 'https://example.com/image.jpg',
      caption: 'Check out this product!'
    }
  }
})
```

#### Document Message

```typescript
await client.messages.send({
  channelId: 'ch_telegram_123',
  to: '123456789',
  content: {
    type: 'document',
    document: {
      url: 'https://example.com/document.pdf',
      filename: 'Report.pdf',
      caption: 'Here is your report'
    }
  }
})
```

#### Inline Keyboard

```typescript
await client.messages.send({
  channelId: 'ch_telegram_123',
  to: '123456789',
  content: {
    type: 'text',
    text: 'Choose an option:',
    replyMarkup: {
      inlineKeyboard: [
        [
          { text: 'Option A', callbackData: 'option_a' },
          { text: 'Option B', callbackData: 'option_b' }
        ],
        [
          { text: 'Visit Website', url: 'https://example.com' }
        ],
        [
          { text: 'Share', switchInlineQuery: 'Check this out!' }
        ]
      ]
    }
  }
})
```

#### Reply Keyboard

```typescript
await client.messages.send({
  channelId: 'ch_telegram_123',
  to: '123456789',
  content: {
    type: 'text',
    text: 'Please share your contact or location:',
    replyMarkup: {
      keyboard: [
        [
          { text: 'Share Phone Number', requestContact: true },
          { text: 'Share Location', requestLocation: true }
        ],
        [
          { text: 'Cancel' }
        ]
      ],
      resizeKeyboard: true,
      oneTimeKeyboard: true
    }
  }
})
```

#### Location Message

```typescript
await client.messages.send({
  channelId: 'ch_telegram_123',
  to: '123456789',
  content: {
    type: 'location',
    location: {
      latitude: -23.5505,
      longitude: -46.6333
    }
  }
})
```

#### Poll Message

```typescript
await client.messages.send({
  channelId: 'ch_telegram_123',
  to: '123456789',
  content: {
    type: 'poll',
    poll: {
      question: 'How would you rate our service?',
      options: ['Excellent', 'Good', 'Fair', 'Poor'],
      isAnonymous: false,
      allowsMultipleAnswers: false
    }
  }
})
```

### Editing Messages

```typescript
await client.messages.edit({
  channelId: 'ch_telegram_123',
  messageId: 'msg_123',
  content: {
    type: 'text',
    text: 'Updated message content'
  }
})
```

### Deleting Messages

```typescript
await client.messages.delete({
  channelId: 'ch_telegram_123',
  messageId: 'msg_123',
  chatId: '123456789'
})
```

### Handling Bot Commands

Bot commands start with `/` and are highlighted in Telegram. Set up commands with BotFather using `/setcommands`:

```
start - Start the bot
help - Get help
settings - Open settings
feedback - Send feedback
```

Then handle them in your flow or webhook:

```typescript
ws.on('message.received', (message) => {
  if (message.content.type === 'text') {
    const text = message.content.text

    if (text.startsWith('/start')) {
      // Handle /start command
      await sendWelcomeMessage(message.from)
    } else if (text.startsWith('/help')) {
      // Handle /help command
      await sendHelpMessage(message.from)
    }
  }
})
```

### Handling Callback Queries

When users click inline keyboard buttons:

```typescript
ws.on('callback_query', async (query) => {
  const { id, data, from, message } = query

  // Answer the callback query (removes loading state)
  await client.telegram.answerCallbackQuery({
    channelId: 'ch_telegram_123',
    callbackQueryId: id,
    text: 'Processing your selection...',
    showAlert: false
  })

  // Handle the callback data
  switch (data) {
    case 'option_a':
      await handleOptionA(from, message)
      break
    case 'option_b':
      await handleOptionB(from, message)
      break
  }
})
```

### Receiving Messages

#### Webhook Events

```typescript
// Message received event
{
  "event": "message.received",
  "data": {
    "id": "msg_abc123",
    "channelId": "ch_telegram_123",
    "channelType": "telegram",
    "direction": "inbound",
    "from": "123456789",
    "content": {
      "type": "text",
      "text": "/start"
    },
    "timestamp": "2024-01-15T10:30:00Z",
    "metadata": {
      "telegramMessageId": 456,
      "chatType": "private",
      "firstName": "John",
      "lastName": "Doe",
      "username": "johndoe"
    }
  }
}
```

## Webhook Setup

### Webhook URL Format

```
https://api.your-domain.com/webhooks/telegram/{channelId}
```

### Manual Webhook Setup

If you need to manually configure the webhook:

```bash
curl -X POST "https://api.telegram.org/bot<TOKEN>/setWebhook" \
  -H "Content-Type: application/json" \
  -d '{
    "url": "https://api.your-domain.com/webhooks/telegram/ch_telegram_123",
    "secret_token": "your_webhook_secret",
    "allowed_updates": ["message", "callback_query", "inline_query"]
  }'
```

### Webhook Security

Linktor validates incoming webhooks using:
1. **Secret Token**: X-Telegram-Bot-Api-Secret-Token header validation
2. **IP Whitelist**: Optional restriction to Telegram's IP ranges

## Group and Channel Support

### Group Chats

Your bot can be added to groups. Configure privacy settings with BotFather:

```
/setprivacy - Disabled (bot receives all messages)
/setprivacy - Enabled (bot only receives commands and mentions)
```

Handle group messages:

```typescript
ws.on('message.received', (message) => {
  const { chatType } = message.metadata

  if (chatType === 'group' || chatType === 'supergroup') {
    // This is a group message
    const isCommand = message.content.text?.startsWith('/')
    const isMentioned = message.content.text?.includes(`@${botUsername}`)

    if (isCommand || isMentioned) {
      // Respond to the group
    }
  }
})
```

### Channel Broadcasts

For channel posts, add your bot as an administrator:

```typescript
await client.messages.send({
  channelId: 'ch_telegram_123',
  to: '@your_channel_username', // or channel ID
  content: {
    type: 'text',
    text: 'New announcement for all subscribers!'
  }
})
```

## Common Issues and Troubleshooting

### "Unauthorized" Error

**Possible causes:**
- Bot token is invalid or revoked
- Bot was deleted

**Solution:**
- Verify token with BotFather
- Create a new bot if necessary

### "Bot was blocked by the user"

**Possible causes:**
- User blocked your bot
- User deleted their Telegram account

**Solution:**
- Handle this error gracefully
- Remove user from active recipients

### Webhook Not Receiving Updates

**Possible causes:**
- Webhook URL not HTTPS
- SSL certificate issues
- Webhook not registered properly

**Solution:**
- Ensure URL uses valid HTTPS certificate
- Check webhook status with `getWebhookInfo`
- Re-register webhook if needed

### "Message is too long"

**Possible causes:**
- Message exceeds 4096 character limit

**Solution:**
- Split long messages into multiple parts
- Use documents for very long content

### Inline Keyboard Not Showing

**Possible causes:**
- Invalid keyboard structure
- Callback data too long (max 64 bytes)

**Solution:**
- Verify keyboard JSON structure
- Shorten callback data or use IDs

### Media Upload Failures

**Possible causes:**
- File too large (max 50MB for most files, 20MB via URL)
- Unsupported format
- URL not accessible

**Solution:**
- Compress files before sending
- Use Linktor's media upload for large files
- Ensure URLs are publicly accessible

## Best Practices

1. **Respond Quickly**: Telegram users expect fast responses. Acknowledge messages within seconds.

2. **Use Commands**: Define clear bot commands for common actions.

3. **Inline Keyboards Over Reply Keyboards**: Inline keyboards provide better UX and don't clutter the chat.

4. **Handle Errors Gracefully**: Always handle blocked users and deleted messages.

5. **Respect Rate Limits**: Telegram limits broadcasts to ~30 messages per second.

6. **Group Etiquette**: In groups, only respond when explicitly triggered to avoid spam.

7. **Deep Linking**: Use start parameters for tracking: `https://t.me/yourbot?start=campaign123`

## Rate Limits

| Action | Limit |
|--------|-------|
| Messages to same chat | 1 per second |
| Bulk notifications | 30 per second |
| Group messages | 20 per minute per group |
| Inline query results | 50 per query |

## Next Steps

- [Flows](/flows/overview) - Build automated Telegram flows
- [Bots](/bots/overview) - Connect AI bots to Telegram
- [Inline Mode](/api/telegram-inline) - Enable inline queries
- [Payments](/api/telegram-payments) - Accept payments via Telegram
