---
sidebar_position: 8
title: Facebook Messenger
---

# Facebook Messenger Integration

Connect your Facebook Page to Linktor and engage with customers through Facebook Messenger. Reach billions of users with rich messaging features including templates, buttons, and persistent menus.

## Overview

The Facebook Messenger integration enables you to:

- Send and receive text, images, videos, and files
- Use message templates (generic, button, receipt, airline, etc.)
- Create persistent menus for quick navigation
- Implement quick replies for guided conversations
- Handle postback events from buttons
- Support message tags for notification messages
- Enable handover protocol for human-bot collaboration

## Prerequisites

Before configuring Facebook Messenger in Linktor, you'll need:

1. **Facebook Page**: A Facebook Page for your business
2. **Meta Developer Account**: Access to [Meta for Developers](https://developers.facebook.com)
3. **Meta App**: An app with Messenger product enabled
4. **Page Access Token**: Token with required permissions

### Required Permissions

Your Meta App needs these permissions:
- `pages_messaging` - Send and receive messages
- `pages_manage_metadata` - Manage page settings
- `pages_read_engagement` - Read page engagement data

## Configuration in Linktor

### Step 1: Create Meta App

1. Go to [Meta for Developers](https://developers.facebook.com)
2. Click **My Apps → Create App**
3. Select **Business** type
4. Add the **Messenger** product to your app
5. Complete app review for production use

### Step 2: Get Page Access Token

1. In your Meta App, go to **Messenger → Settings**
2. Under **Access Tokens**, click **Add or Remove Pages**
3. Select your Facebook Page and grant permissions
4. Click **Generate Token** and copy the Page Access Token

### Step 3: Add Channel in Dashboard

1. Go to **Settings → Channels** in Linktor
2. Click **Add Channel** and select **Facebook Messenger**
3. Fill in the configuration:

| Field | Description |
|-------|-------------|
| **Name** | Display name (e.g., "Facebook Support") |
| **Page ID** | Your Facebook Page ID |
| **Page Access Token** | Token generated in Step 2 |
| **App Secret** | Your Meta App secret (for webhook validation) |
| **Verify Token** | A secret string for webhook verification |

4. Click **Save** to create the channel

### Step 4: Configure Webhook

1. In Meta App Dashboard, go to **Messenger → Settings**
2. Under **Webhooks**, click **Add Callback URL**
3. Enter:
   - **Callback URL**: `https://api.your-domain.com/webhooks/facebook/{channelId}`
   - **Verify Token**: Same token configured in Linktor
4. Click **Verify and Save**
5. Subscribe to webhook fields:
   - `messages`
   - `messaging_postbacks`
   - `messaging_optins`
   - `message_deliveries`
   - `message_reads`

## API Usage

### Sending Messages

#### Text Message

```typescript
import { Linktor } from '@linktor/sdk'

const client = new Linktor({ apiKey: 'YOUR_API_KEY' })

await client.messages.send({
  channelId: 'ch_facebook_123',
  to: '1234567890', // Facebook Page-scoped User ID (PSID)
  content: {
    type: 'text',
    text: 'Hello! How can I help you today?'
  }
})
```

#### Image Message

```typescript
await client.messages.send({
  channelId: 'ch_facebook_123',
  to: '1234567890',
  content: {
    type: 'image',
    image: {
      url: 'https://example.com/product.jpg'
    }
  }
})
```

#### Button Template

```typescript
await client.messages.send({
  channelId: 'ch_facebook_123',
  to: '1234567890',
  content: {
    type: 'template',
    template: {
      type: 'button',
      text: 'What would you like to do?',
      buttons: [
        {
          type: 'postback',
          title: 'View Orders',
          payload: 'VIEW_ORDERS'
        },
        {
          type: 'postback',
          title: 'Track Shipment',
          payload: 'TRACK_SHIPMENT'
        },
        {
          type: 'web_url',
          title: 'Visit Website',
          url: 'https://example.com'
        }
      ]
    }
  }
})
```

#### Generic Template (Carousel)

```typescript
await client.messages.send({
  channelId: 'ch_facebook_123',
  to: '1234567890',
  content: {
    type: 'template',
    template: {
      type: 'generic',
      elements: [
        {
          title: 'iPhone 15 Pro',
          subtitle: '$999 - Free shipping',
          imageUrl: 'https://example.com/iphone.jpg',
          defaultAction: {
            type: 'web_url',
            url: 'https://example.com/products/iphone-15'
          },
          buttons: [
            {
              type: 'postback',
              title: 'Buy Now',
              payload: 'BUY_IPHONE_15'
            },
            {
              type: 'web_url',
              title: 'View Details',
              url: 'https://example.com/products/iphone-15'
            }
          ]
        },
        {
          title: 'MacBook Pro',
          subtitle: '$1,999 - Free shipping',
          imageUrl: 'https://example.com/macbook.jpg',
          buttons: [
            {
              type: 'postback',
              title: 'Buy Now',
              payload: 'BUY_MACBOOK'
            }
          ]
        }
      ]
    }
  }
})
```

#### Quick Replies

```typescript
await client.messages.send({
  channelId: 'ch_facebook_123',
  to: '1234567890',
  content: {
    type: 'text',
    text: 'How would you rate your experience?',
    quickReplies: [
      {
        contentType: 'text',
        title: 'Great!',
        payload: 'RATING_GREAT'
      },
      {
        contentType: 'text',
        title: 'Good',
        payload: 'RATING_GOOD'
      },
      {
        contentType: 'text',
        title: 'Could be better',
        payload: 'RATING_POOR'
      },
      {
        contentType: 'user_phone_number'
      },
      {
        contentType: 'user_email'
      }
    ]
  }
})
```

#### Receipt Template

```typescript
await client.messages.send({
  channelId: 'ch_facebook_123',
  to: '1234567890',
  content: {
    type: 'template',
    template: {
      type: 'receipt',
      recipientName: 'John Doe',
      orderNumber: 'ORD-12345',
      currency: 'USD',
      paymentMethod: 'Visa 1234',
      orderUrl: 'https://example.com/orders/12345',
      timestamp: '1704067200',
      address: {
        street1: '123 Main St',
        city: 'San Francisco',
        state: 'CA',
        postalCode: '94105',
        country: 'US'
      },
      summary: {
        subtotal: 99.99,
        shippingCost: 4.99,
        totalTax: 8.49,
        totalCost: 113.47
      },
      elements: [
        {
          title: 'iPhone Case',
          subtitle: 'Black, Standard',
          quantity: 1,
          price: 29.99,
          currency: 'USD',
          imageUrl: 'https://example.com/case.jpg'
        },
        {
          title: 'Screen Protector',
          subtitle: 'Tempered Glass',
          quantity: 2,
          price: 35.00,
          currency: 'USD'
        }
      ]
    }
  }
})
```

### Message Tags

Send messages outside the 24-hour window using message tags:

```typescript
// Confirmed event update
await client.messages.send({
  channelId: 'ch_facebook_123',
  to: '1234567890',
  content: {
    type: 'text',
    text: 'Your flight UA123 has been delayed. New departure time: 3:45 PM.'
  },
  options: {
    tag: 'CONFIRMED_EVENT_UPDATE'
  }
})

// Post-purchase update
await client.messages.send({
  channelId: 'ch_facebook_123',
  to: '1234567890',
  content: {
    type: 'text',
    text: 'Your order #12345 has shipped! Track it here: https://track.example.com/12345'
  },
  options: {
    tag: 'POST_PURCHASE_UPDATE'
  }
})

// Account update
await client.messages.send({
  channelId: 'ch_facebook_123',
  to: '1234567890',
  content: {
    type: 'text',
    text: 'Your password was changed. If this was not you, please contact support.'
  },
  options: {
    tag: 'ACCOUNT_UPDATE'
  }
})
```

### Persistent Menu

Configure a persistent menu for easy navigation:

```typescript
await client.facebook.setPersistentMenu({
  channelId: 'ch_facebook_123',
  menu: [
    {
      locale: 'default',
      composerInputDisabled: false,
      callToActions: [
        {
          type: 'postback',
          title: 'Get Started',
          payload: 'GET_STARTED'
        },
        {
          type: 'nested',
          title: 'Products',
          callToActions: [
            {
              type: 'postback',
              title: 'New Arrivals',
              payload: 'PRODUCTS_NEW'
            },
            {
              type: 'postback',
              title: 'Best Sellers',
              payload: 'PRODUCTS_BEST'
            },
            {
              type: 'web_url',
              title: 'All Products',
              url: 'https://example.com/products'
            }
          ]
        },
        {
          type: 'postback',
          title: 'Contact Support',
          payload: 'CONTACT_SUPPORT'
        }
      ]
    }
  ]
})
```

### Get Started Button

Set up the Get Started button for new conversations:

```typescript
await client.facebook.setGetStartedButton({
  channelId: 'ch_facebook_123',
  payload: 'GET_STARTED'
})
```

### Greeting Text

Set the greeting text shown before users start a conversation:

```typescript
await client.facebook.setGreeting({
  channelId: 'ch_facebook_123',
  greeting: [
    {
      locale: 'default',
      text: 'Hi {{user_first_name}}! Welcome to Acme Support. Tap Get Started to begin.'
    },
    {
      locale: 'pt_BR',
      text: 'Oi {{user_first_name}}! Bem-vindo ao suporte da Acme.'
    }
  ]
})
```

### Receiving Messages

#### Webhook Events

```typescript
// Text message received
{
  "event": "message.received",
  "data": {
    "id": "msg_abc123",
    "channelId": "ch_facebook_123",
    "channelType": "facebook",
    "direction": "inbound",
    "from": "1234567890",
    "content": {
      "type": "text",
      "text": "Hi, I need help with my order"
    },
    "timestamp": "2024-01-15T10:30:00Z",
    "metadata": {
      "mid": "m_abc123...",
      "firstName": "John",
      "lastName": "Doe",
      "profilePic": "https://..."
    }
  }
}

// Postback received (button click)
{
  "event": "facebook.postback",
  "data": {
    "channelId": "ch_facebook_123",
    "from": "1234567890",
    "payload": "VIEW_ORDERS",
    "title": "View Orders",
    "timestamp": "2024-01-15T10:30:05Z"
  }
}

// Quick reply selected
{
  "event": "facebook.quick_reply",
  "data": {
    "channelId": "ch_facebook_123",
    "from": "1234567890",
    "payload": "RATING_GREAT",
    "text": "Great!",
    "timestamp": "2024-01-15T10:30:10Z"
  }
}
```

## Webhook Setup

### Webhook URL Format

```
https://api.your-domain.com/webhooks/facebook/{channelId}
```

### Webhook Verification

Facebook sends a GET request to verify your webhook:

```
GET /webhooks/facebook/{channelId}?hub.mode=subscribe&hub.verify_token=YOUR_TOKEN&hub.challenge=CHALLENGE
```

Linktor automatically handles this when configured correctly.

### Webhook Security

Linktor validates all incoming webhooks using:
1. **Signature verification**: `X-Hub-Signature-256` header validation using App Secret
2. **Payload validation**: Ensures webhook structure is valid

## Common Issues and Troubleshooting

### "Message failed to send"

**Possible causes:**
- Invalid PSID (Page-scoped User ID)
- User has not messaged the page
- Token expired or invalid
- Outside 24-hour window without tag

**Solution:**
- Verify PSID is correct and user has interacted with page
- Regenerate Page Access Token
- Use appropriate message tag for notifications

### "Webhook verification failed"

**Possible causes:**
- Verify token mismatch
- Webhook URL not accessible
- SSL certificate issues

**Solution:**
- Ensure verify token matches exactly
- Test webhook URL is publicly accessible
- Verify SSL certificate is valid

### "App not approved for permissions"

**Possible causes:**
- App in development mode
- Required permissions not approved

**Solution:**
- Submit app for review
- Request necessary permissions
- For testing, add users as testers

### Template Errors

**Possible causes:**
- Invalid template structure
- Missing required fields
- Button limits exceeded

**Solution:**
- Maximum 3 buttons per template
- Maximum 10 elements in carousel
- Ensure all required fields are present

### Rate Limited

**Possible causes:**
- Exceeded API rate limits
- Too many messages in short time

**Solution:**
- Implement exponential backoff
- Batch messages when possible
- Monitor rate limit headers

## Best Practices

1. **Respond Quickly**: Users expect fast responses. Set up automated replies for common queries.

2. **Use Rich Templates**: Take advantage of carousels, buttons, and quick replies for better engagement.

3. **Personalize Messages**: Use `{{user_first_name}}` and profile data to personalize conversations.

4. **Respect the 24-Hour Window**: Only use message tags for legitimate notifications.

5. **Set Up Persistent Menu**: Provide easy navigation for common actions.

6. **Handle Postbacks**: Always handle postback events from buttons.

7. **Test in Development**: Use development mode and test users before going live.

8. **Monitor Feedback**: Watch for user feedback and adjust your bot accordingly.

## Policy Compliance

### Facebook Messenger Platform Policy

- Get opt-in before sending promotional messages
- Respond to user messages within 24 hours
- Use message tags only for their intended purpose
- Don't send spam or unwanted messages
- Provide easy opt-out mechanism

### Message Tags Guidelines

| Tag | Use Case |
|-----|----------|
| `CONFIRMED_EVENT_UPDATE` | Updates about confirmed events |
| `POST_PURCHASE_UPDATE` | Order/shipping updates |
| `ACCOUNT_UPDATE` | Account status changes |
| `HUMAN_AGENT` | Human agent response (7-day window) |

## Rate Limits

| Type | Limit |
|------|-------|
| Send API | 200 calls per user per hour |
| Batch messages | 50 messages per request |
| Broadcast messages | Based on subscriber count |

## Next Steps

- [Flows](/flows/overview) - Build automated Messenger flows
- [AI Bots](/bots/overview) - Add AI to your Messenger experience
- [Analytics](/api/analytics) - Track Messenger performance
- [Handover Protocol](/guides/facebook-handover) - Human-bot collaboration
