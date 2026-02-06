---
sidebar_position: 2
title: WhatsApp
---

# WhatsApp Business API Integration

Connect your WhatsApp Business account to Linktor and engage with over 2 billion users worldwide. This guide covers setup with the official WhatsApp Business API (Cloud API).

## Overview

The WhatsApp Business API integration allows you to:

- Send and receive text messages, images, documents, audio, and video
- Use message templates for notifications and marketing
- Send interactive messages with buttons and lists
- Track message delivery and read receipts
- Handle high message volumes with enterprise-grade reliability

## Prerequisites

Before configuring WhatsApp in Linktor, you'll need:

1. **Meta Business Account**: A verified Meta Business account at [business.facebook.com](https://business.facebook.com)
2. **WhatsApp Business Account**: Created within Meta Business Suite
3. **Phone Number**: A phone number that will be used for WhatsApp Business (can't be registered with regular WhatsApp)
4. **Meta App**: A Meta App with WhatsApp Product enabled

### Meta App Setup

1. Go to [Meta for Developers](https://developers.facebook.com)
2. Create a new App (select "Business" type)
3. Add the **WhatsApp** product to your app
4. Complete business verification (required for production)

## Configuration in Linktor

### Step 1: Get Your Credentials

From the Meta App Dashboard:

1. Navigate to **WhatsApp → API Setup**
2. Copy your **Phone Number ID**
3. Generate a **Permanent Access Token** (or use a System User token for production)
4. Note your **WhatsApp Business Account ID**

### Step 2: Add Channel in Dashboard

1. Go to **Settings → Channels** in Linktor
2. Click **Add Channel** and select **WhatsApp**
3. Fill in the configuration:

| Field | Description |
|-------|-------------|
| **Name** | Display name for this channel (e.g., "WhatsApp Support") |
| **Phone Number ID** | Your WhatsApp phone number ID from Meta |
| **Access Token** | Permanent access token with `whatsapp_business_messaging` permission |
| **Verify Token** | A secret string you create for webhook verification |
| **Business Account ID** | Your WhatsApp Business Account ID |

4. Click **Save** to create the channel

### Step 3: Configure Webhook

Linktor will provide a webhook URL. Configure it in Meta:

1. In Meta App Dashboard, go to **WhatsApp → Configuration**
2. Under **Webhook**, click **Edit**
3. Enter your Linktor webhook URL: `https://api.your-domain.com/webhooks/whatsapp/{channelId}`
4. Enter the same **Verify Token** you configured in Linktor
5. Click **Verify and Save**
6. Subscribe to webhook fields:
   - `messages`
   - `message_template_status_update` (optional)

## API Usage

### Sending Messages

#### Text Message

```typescript
import { Linktor } from '@linktor/sdk'

const client = new Linktor({ apiKey: 'YOUR_API_KEY' })

await client.messages.send({
  channelId: 'ch_whatsapp_123',
  to: '+5511999999999',
  content: {
    type: 'text',
    text: 'Hello! How can I help you today?'
  }
})
```

#### Image Message

```typescript
await client.messages.send({
  channelId: 'ch_whatsapp_123',
  to: '+5511999999999',
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
  channelId: 'ch_whatsapp_123',
  to: '+5511999999999',
  content: {
    type: 'document',
    document: {
      url: 'https://example.com/invoice.pdf',
      filename: 'Invoice-2024-001.pdf',
      caption: 'Your invoice is attached'
    }
  }
})
```

#### Interactive Buttons

```typescript
await client.messages.send({
  channelId: 'ch_whatsapp_123',
  to: '+5511999999999',
  content: {
    type: 'interactive',
    interactive: {
      type: 'button',
      body: {
        text: 'Would you like to proceed with your order?'
      },
      action: {
        buttons: [
          { id: 'confirm', title: 'Yes, confirm' },
          { id: 'cancel', title: 'No, cancel' },
          { id: 'help', title: 'Need help' }
        ]
      }
    }
  }
})
```

#### Interactive List

```typescript
await client.messages.send({
  channelId: 'ch_whatsapp_123',
  to: '+5511999999999',
  content: {
    type: 'interactive',
    interactive: {
      type: 'list',
      header: {
        type: 'text',
        text: 'Our Services'
      },
      body: {
        text: 'Please select a service:'
      },
      action: {
        button: 'View Options',
        sections: [
          {
            title: 'Support',
            rows: [
              { id: 'technical', title: 'Technical Support', description: 'Get help with technical issues' },
              { id: 'billing', title: 'Billing Support', description: 'Questions about payments' }
            ]
          },
          {
            title: 'Sales',
            rows: [
              { id: 'pricing', title: 'Pricing Info', description: 'Learn about our plans' },
              { id: 'demo', title: 'Request Demo', description: 'Schedule a product demo' }
            ]
          }
        ]
      }
    }
  }
})
```

### Message Templates

WhatsApp requires pre-approved templates for initiating conversations outside the 24-hour window.

#### Creating a Template

Templates are created in Meta Business Manager, but you can list them via API:

```typescript
const templates = await client.whatsapp.templates.list({
  channelId: 'ch_whatsapp_123'
})
```

#### Sending a Template Message

```typescript
await client.messages.send({
  channelId: 'ch_whatsapp_123',
  to: '+5511999999999',
  content: {
    type: 'template',
    template: {
      name: 'order_confirmation',
      language: { code: 'en' },
      components: [
        {
          type: 'body',
          parameters: [
            { type: 'text', text: 'John' },
            { type: 'text', text: 'ORD-12345' },
            { type: 'currency', currency: { code: 'USD', amount_1000: 99900, fallback_value: '$999.00' } }
          ]
        }
      ]
    }
  }
})
```

### Receiving Messages

#### Webhook Events

Linktor normalizes WhatsApp webhooks into standard events:

```typescript
// Message received event
{
  "event": "message.received",
  "data": {
    "id": "msg_abc123",
    "channelId": "ch_whatsapp_123",
    "channelType": "whatsapp",
    "direction": "inbound",
    "from": "+5511999999999",
    "content": {
      "type": "text",
      "text": "Hi, I need help"
    },
    "timestamp": "2024-01-15T10:30:00Z",
    "metadata": {
      "waMessageId": "wamid.abc123",
      "profileName": "John Doe"
    }
  }
}
```

#### Status Updates

```typescript
// Message status update
{
  "event": "message.status",
  "data": {
    "messageId": "msg_abc123",
    "status": "read", // sent, delivered, read, failed
    "timestamp": "2024-01-15T10:31:00Z"
  }
}
```

### Using WebSocket for Real-Time Updates

```typescript
const ws = client.realtime.connect()

ws.on('message.received', (message) => {
  if (message.channelType === 'whatsapp') {
    console.log('WhatsApp message:', message.content)
  }
})

ws.on('message.status', (status) => {
  console.log(`Message ${status.messageId} is now ${status.status}`)
})
```

## Webhook Setup

### Webhook URL Format

```
https://api.your-domain.com/webhooks/whatsapp/{channelId}
```

### Webhook Verification

When Meta verifies your webhook, it sends a GET request with:
- `hub.mode`: Should be "subscribe"
- `hub.verify_token`: Your verify token
- `hub.challenge`: Return this to confirm

Linktor handles this automatically when properly configured.

### Webhook Security

Linktor validates all incoming webhooks using:
1. **Signature verification**: X-Hub-Signature-256 header validation
2. **Timestamp validation**: Rejects old webhooks to prevent replay attacks

## Common Issues and Troubleshooting

### "Message failed to send"

**Possible causes:**
- Phone number not registered or verified
- Access token expired or invalid
- Outside 24-hour window without template

**Solution:**
- Verify phone number status in Meta Business Manager
- Regenerate access token if expired
- Use a template message to initiate conversation

### "Template rejected"

**Possible causes:**
- Template contains policy violations
- Incorrect template parameters

**Solution:**
- Review Meta's [template guidelines](https://developers.facebook.com/docs/whatsapp/message-templates/guidelines)
- Ensure all required parameters are provided

### "Webhook not receiving messages"

**Possible causes:**
- Webhook URL not configured correctly
- Verify token mismatch
- Webhook subscription fields not selected

**Solution:**
- Verify webhook URL is publicly accessible
- Confirm verify token matches exactly
- Check all required webhook fields are subscribed

### "Rate limited"

**Possible causes:**
- Exceeded messaging limits for your tier
- Too many API requests

**Solution:**
- Check your messaging tier in Meta Business Manager
- Implement exponential backoff
- Request a higher messaging limit

### Media Upload Failures

**Possible causes:**
- File too large (max 16MB for most media)
- Unsupported file format
- URL not publicly accessible

**Solution:**
- Compress files before sending
- Use supported formats (JPEG, PNG, PDF, MP4, etc.)
- Ensure media URLs are publicly accessible or use Linktor's media upload

## Best Practices

1. **24-Hour Window**: Respond within 24 hours to send free-form messages. After that, use templates.

2. **Template Approval**: Submit templates well in advance. Approval can take 24-48 hours.

3. **Opt-in Compliance**: Ensure users have opted in before sending marketing messages.

4. **Quality Rating**: Monitor your quality rating in Meta Business Manager. Low ratings can restrict your account.

5. **Message Formatting**: Use formatting sparingly. WhatsApp supports *bold*, _italic_, ~strikethrough~, and ```code```.

6. **Error Handling**: Always handle failed messages gracefully and implement retry logic.

## Rate Limits

| Tier | Messages per Day | Requirements |
|------|------------------|--------------|
| Unverified | 250 | New accounts |
| Tier 1 | 1,000 | Verified business |
| Tier 2 | 10,000 | Good quality rating |
| Tier 3 | 100,000 | Excellent quality rating |
| Tier 4 | Unlimited | Enterprise partnership |

## Next Steps

- [Message Templates](/api/whatsapp-templates) - Deep dive into templates
- [Flows](/flows/overview) - Build automated WhatsApp flows
- [Bots](/bots/overview) - Connect AI bots to WhatsApp
- [Analytics](/api/analytics) - Track WhatsApp performance
