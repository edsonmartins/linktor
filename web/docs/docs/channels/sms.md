---
sidebar_position: 4
title: SMS
---

# SMS Integration

Connect SMS messaging to Linktor using providers like Twilio, Vonage (Nexmo), or other SMS gateways. Reach customers on any mobile device without requiring them to install an app.

## Overview

The SMS integration enables you to:

- Send and receive text messages worldwide
- Support MMS for images and media (carrier dependent)
- Use short codes or long codes for different use cases
- Implement two-way conversational SMS
- Send bulk notifications and alerts
- Track delivery status for all messages

## Prerequisites

Before configuring SMS in Linktor, you'll need:

1. **SMS Provider Account**: An account with one of our supported providers
2. **Phone Number**: A phone number or short code from your provider
3. **API Credentials**: API keys or auth tokens from your provider

### Supported Providers

| Provider | Features | Best For |
|----------|----------|----------|
| **Twilio** | Full-featured, global reach, MMS | Enterprise, global coverage |
| **Vonage (Nexmo)** | Reliable, competitive pricing | Cost-effective messaging |
| **MessageBird** | European focus, WhatsApp bundle | EU businesses |
| **Plivo** | Developer-friendly, affordable | Startups, developers |
| **AWS SNS** | AWS integration, simple SMS | AWS ecosystem users |

## Configuration in Linktor

### Twilio Setup

#### Step 1: Get Twilio Credentials

1. Sign up at [twilio.com](https://www.twilio.com)
2. From the Console Dashboard, note your:
   - **Account SID**
   - **Auth Token**
3. Buy or configure a phone number with SMS capability
4. Note your **Twilio Phone Number**

#### Step 2: Add Channel in Dashboard

1. Go to **Settings → Channels** in Linktor
2. Click **Add Channel** and select **SMS**
3. Choose **Twilio** as the provider
4. Fill in the configuration:

| Field | Description |
|-------|-------------|
| **Name** | Display name for this channel |
| **Account SID** | Your Twilio Account SID |
| **Auth Token** | Your Twilio Auth Token |
| **Phone Number** | Your Twilio phone number (E.164 format) |
| **Messaging Service SID** | Optional, for using Messaging Services |

5. Click **Save** to create the channel

#### Step 3: Configure Webhook in Twilio

1. Go to **Phone Numbers → Manage → Active Numbers** in Twilio Console
2. Select your phone number
3. Under **Messaging**, set the webhook URL:
   - **When a message comes in**: `https://api.your-domain.com/webhooks/sms/{channelId}`
   - **Method**: HTTP POST
4. Under **Status Callback URL** (optional):
   - `https://api.your-domain.com/webhooks/sms/{channelId}/status`

### Vonage (Nexmo) Setup

#### Step 1: Get Vonage Credentials

1. Sign up at [vonage.com](https://www.vonage.com/communications-apis/)
2. From the Dashboard, note your:
   - **API Key**
   - **API Secret**
3. Rent a virtual number with SMS capability

#### Step 2: Add Channel in Dashboard

1. Go to **Settings → Channels** in Linktor
2. Click **Add Channel** and select **SMS**
3. Choose **Vonage** as the provider
4. Fill in the configuration:

| Field | Description |
|-------|-------------|
| **Name** | Display name for this channel |
| **API Key** | Your Vonage API Key |
| **API Secret** | Your Vonage API Secret |
| **Phone Number** | Your Vonage virtual number (E.164 format) |

5. Click **Save** to create the channel

#### Step 3: Configure Webhook in Vonage

1. Go to **Numbers → Your Numbers** in Vonage Dashboard
2. Click the gear icon next to your number
3. Set the **Inbound Webhook URL**:
   - `https://api.your-domain.com/webhooks/sms/{channelId}`

## API Usage

### Sending Messages

#### Basic Text Message

```typescript
import { Linktor } from '@linktor/sdk'

const client = new Linktor({ apiKey: 'YOUR_API_KEY' })

await client.messages.send({
  channelId: 'ch_sms_123',
  to: '+5511999999999',
  content: {
    type: 'text',
    text: 'Your verification code is 123456'
  }
})
```

#### With Sender ID (Alpha Tag)

```typescript
await client.messages.send({
  channelId: 'ch_sms_123',
  to: '+5511999999999',
  from: 'MyCompany', // Alpha sender ID (country dependent)
  content: {
    type: 'text',
    text: 'Your order has shipped!'
  }
})
```

#### MMS Message (Twilio)

```typescript
await client.messages.send({
  channelId: 'ch_sms_123',
  to: '+15551234567', // US/Canada numbers support MMS
  content: {
    type: 'image',
    image: {
      url: 'https://example.com/product.jpg',
      caption: 'Check out our new product!'
    }
  }
})
```

#### Bulk Messaging

```typescript
const recipients = ['+5511999999999', '+5511888888888', '+5511777777777']

const results = await client.messages.sendBulk({
  channelId: 'ch_sms_123',
  recipients: recipients.map(to => ({
    to,
    content: {
      type: 'text',
      text: 'Flash sale! 50% off all items today only.'
    }
  }))
})

// Check results
results.forEach(result => {
  if (result.success) {
    console.log(`Sent to ${result.to}: ${result.messageId}`)
  } else {
    console.log(`Failed for ${result.to}: ${result.error}`)
  }
})
```

### Message Templates

Use templates for consistent messaging:

```typescript
// Create a template
await client.templates.create({
  name: 'order_shipped',
  channel: 'sms',
  content: 'Hi {{name}}, your order #{{orderId}} has shipped! Track it here: {{trackingUrl}}'
})

// Send using template
await client.messages.send({
  channelId: 'ch_sms_123',
  to: '+5511999999999',
  template: {
    name: 'order_shipped',
    variables: {
      name: 'John',
      orderId: 'ORD-12345',
      trackingUrl: 'https://track.example.com/ORD-12345'
    }
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
    "channelId": "ch_sms_123",
    "channelType": "sms",
    "direction": "inbound",
    "from": "+5511999999999",
    "to": "+5511888888888",
    "content": {
      "type": "text",
      "text": "STOP"
    },
    "timestamp": "2024-01-15T10:30:00Z",
    "metadata": {
      "provider": "twilio",
      "providerMessageId": "SM123abc"
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
    "status": "delivered", // queued, sent, delivered, failed, undelivered
    "timestamp": "2024-01-15T10:30:05Z",
    "metadata": {
      "errorCode": null,
      "errorMessage": null
    }
  }
}
```

### Two-Way Conversations

Handle conversational SMS with context:

```typescript
import { Linktor } from '@linktor/sdk'

const client = new Linktor({ apiKey: 'YOUR_API_KEY' })

// Set up real-time listener
const ws = client.realtime.connect()

ws.on('message.received', async (message) => {
  if (message.channelType !== 'sms') return

  const { from, content } = message
  const text = content.text.toUpperCase()

  // Handle keywords
  if (text === 'STOP') {
    await client.contacts.update(from, { optedOut: true })
    await client.messages.send({
      channelId: message.channelId,
      to: from,
      content: { type: 'text', text: 'You have been unsubscribed.' }
    })
  } else if (text === 'HELP') {
    await client.messages.send({
      channelId: message.channelId,
      to: from,
      content: {
        type: 'text',
        text: 'Reply STOP to unsubscribe, BALANCE to check balance, or type your question.'
      }
    })
  } else if (text === 'BALANCE') {
    const balance = await getCustomerBalance(from)
    await client.messages.send({
      channelId: message.channelId,
      to: from,
      content: { type: 'text', text: `Your balance is $${balance}` }
    })
  } else {
    // Route to conversation flow or agent
    await client.flows.trigger('sms_support', { message })
  }
})
```

## Webhook Setup

### Webhook URL Format

```
https://api.your-domain.com/webhooks/sms/{channelId}
```

### Webhook Validation

Linktor validates webhooks differently per provider:

**Twilio:**
- Validates `X-Twilio-Signature` header using your Auth Token

**Vonage:**
- Validates request signature if configured

### Webhook Response

For Twilio, you can respond directly to inbound messages:

```xml
<?xml version="1.0" encoding="UTF-8"?>
<Response>
  <Message>Thanks for your message! We'll get back to you soon.</Message>
</Response>
```

Linktor handles this automatically when you configure auto-replies.

## Common Issues and Troubleshooting

### "Message failed to send"

**Possible causes:**
- Invalid phone number format
- Number not SMS-capable
- Insufficient account balance
- Carrier filtering

**Solution:**
- Use E.164 format (+CountryCode Number)
- Verify number is mobile, not landline
- Check provider account balance
- Review carrier filtering guidelines

### "Undelivered" Status

**Possible causes:**
- Phone is off or out of coverage
- Number is invalid or disconnected
- Message filtered by carrier

**Solution:**
- Check error code from provider
- Verify recipient number is active
- Review message content for spam triggers

### Not Receiving Inbound Messages

**Possible causes:**
- Webhook URL not configured
- Webhook URL not accessible
- Number not configured for inbound

**Solution:**
- Verify webhook URL in provider console
- Test webhook endpoint is publicly accessible
- Ensure number has inbound SMS capability

### "30007" Error (Twilio) - Message Filtering

**Possible causes:**
- Message flagged as spam
- Recipient carrier filtering
- Violated messaging guidelines

**Solution:**
- Register for A2P 10DLC (US)
- Use approved sender IDs
- Follow carrier messaging guidelines

### Character Encoding Issues

**Possible causes:**
- Special characters outside GSM-7 encoding
- Emojis causing UCS-2 encoding (160→70 char limit)

**Solution:**
- Stick to GSM-7 characters for maximum length
- Be aware emojis reduce character limit
- Split long messages appropriately

## Best Practices

1. **Get Consent**: Always have explicit opt-in before sending SMS. Include opt-out instructions.

2. **Respect Quiet Hours**: Don't send marketing messages late at night or early morning.

3. **Keep It Short**: SMS has a 160-character limit (GSM-7). Be concise.

4. **Include Opt-Out**: Always include STOP instructions for marketing messages.

5. **Use A2P 10DLC**: In the US, register for 10DLC to avoid carrier filtering.

6. **Handle Keywords**: Process STOP, HELP, and other standard keywords.

7. **Monitor Delivery Rates**: Track delivery rates and investigate failures.

8. **Personalize When Appropriate**: Use the recipient's name when available.

## Compliance

### US Regulations (TCPA)

- Obtain prior express written consent for marketing
- Honor opt-out requests immediately
- Include business identification
- Send only during reasonable hours (8am-9pm local)

### A2P 10DLC Registration

For US messaging, register your brand and campaigns:

1. Register your business with The Campaign Registry (TCR)
2. Create and register your messaging campaigns
3. Associate registered campaigns with your phone numbers

### GDPR (EU)

- Obtain explicit consent
- Provide clear opt-out mechanism
- Don't store data longer than necessary
- Honor data deletion requests

## Rate Limits

| Provider | Limit | Notes |
|----------|-------|-------|
| Twilio (Long Code) | 1 msg/sec per number | Use messaging service for higher throughput |
| Twilio (Short Code) | 100 msg/sec | Requires approval |
| Vonage | 30 msg/sec | Default, can be increased |

## Pricing Considerations

SMS pricing varies by:
- **Direction**: Outbound vs inbound
- **Country**: International rates vary significantly
- **Number type**: Short codes vs long codes
- **Provider**: Different base rates

Use Linktor's cost estimation:

```typescript
const estimate = await client.sms.estimateCost({
  to: '+5511999999999',
  content: 'Your message here',
  channelId: 'ch_sms_123'
})

console.log(`Estimated cost: $${estimate.cost} (${estimate.segments} segments)`)
```

## Next Steps

- [Flows](/flows/overview) - Build SMS automation flows
- [Bulk Messaging](/api/bulk-messaging) - Send campaigns at scale
- [Analytics](/api/analytics) - Track SMS performance
- [Compliance Guide](/guides/sms-compliance) - Detailed compliance information
