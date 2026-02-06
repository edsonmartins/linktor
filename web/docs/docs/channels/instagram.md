---
sidebar_position: 9
title: Instagram
---

# Instagram DM Integration

Connect your Instagram Business or Creator account to Linktor and engage with customers through Instagram Direct Messages. Respond to DMs, story mentions, and story replies from a unified inbox.

## Overview

The Instagram DM integration enables you to:

- Send and receive direct messages
- Respond to story mentions and replies
- Send images, videos, and voice messages
- Use quick replies and ice breakers
- Handle message reactions
- Support Instagram Shopping product shares
- Integrate with AI bots for automated responses

## Prerequisites

Before configuring Instagram in Linktor, you'll need:

1. **Instagram Business/Creator Account**: A professional Instagram account (not personal)
2. **Facebook Page**: Connected to your Instagram account
3. **Meta Developer Account**: Access to [Meta for Developers](https://developers.facebook.com)
4. **Meta App**: An app with Instagram Messaging enabled

### Required Permissions

Your Meta App needs these permissions:
- `instagram_basic` - Basic account info
- `instagram_manage_messages` - Send and receive messages
- `pages_manage_metadata` - Manage connected page

### Account Requirements

- Instagram account must be **Business** or **Creator** type
- Account must be connected to a Facebook Page
- Page must have message permissions enabled
- For full production access, complete Meta App Review

## Configuration in Linktor

### Step 1: Connect Instagram to Facebook Page

1. Open Instagram app and go to **Settings → Account**
2. Select **Linked Accounts** or **Sharing to Other Apps**
3. Connect to your Facebook Page
4. Enable **Allow access to messages** in Facebook Page settings

### Step 2: Create Meta App

1. Go to [Meta for Developers](https://developers.facebook.com)
2. Create a new App or use existing one
3. Add **Instagram** product
4. Add **Messenger** product (required for Instagram API)
5. Configure Instagram settings

### Step 3: Get Access Token

1. In Meta App Dashboard, go to **Instagram → Basic Display** or use **Messenger → Settings**
2. Generate a Page Access Token that includes Instagram
3. The token should have `instagram_basic` and `instagram_manage_messages` permissions

### Step 4: Add Channel in Dashboard

1. Go to **Settings → Channels** in Linktor
2. Click **Add Channel** and select **Instagram**
3. Fill in the configuration:

| Field | Description |
|-------|-------------|
| **Name** | Display name (e.g., "Instagram Support") |
| **Instagram Account ID** | Your Instagram Business Account ID |
| **Page ID** | Connected Facebook Page ID |
| **Access Token** | Page Access Token with Instagram permissions |
| **App Secret** | Your Meta App secret |
| **Verify Token** | A secret string for webhook verification |

4. Click **Save** to create the channel

### Step 5: Configure Webhook

1. In Meta App Dashboard, go to **Instagram → Webhooks** (via Messenger settings)
2. Add Callback URL: `https://api.your-domain.com/webhooks/instagram/{channelId}`
3. Enter your Verify Token
4. Subscribe to webhook fields:
   - `messages`
   - `messaging_postbacks`
   - `message_reactions`
   - `story_mentions` (requires additional permission)

## API Usage

### Sending Messages

#### Text Message

```typescript
import { Linktor } from '@linktor/sdk'

const client = new Linktor({ apiKey: 'YOUR_API_KEY' })

await client.messages.send({
  channelId: 'ch_instagram_123',
  to: '17841234567890', // Instagram-scoped User ID (IGSID)
  content: {
    type: 'text',
    text: 'Thanks for reaching out! How can I help you today?'
  }
})
```

#### Image Message

```typescript
await client.messages.send({
  channelId: 'ch_instagram_123',
  to: '17841234567890',
  content: {
    type: 'image',
    image: {
      url: 'https://example.com/product.jpg'
    }
  }
})
```

#### Video Message

```typescript
await client.messages.send({
  channelId: 'ch_instagram_123',
  to: '17841234567890',
  content: {
    type: 'video',
    video: {
      url: 'https://example.com/tutorial.mp4'
    }
  }
})
```

#### Heart/Like Sticker

```typescript
await client.messages.send({
  channelId: 'ch_instagram_123',
  to: '17841234567890',
  content: {
    type: 'sticker',
    sticker: {
      id: 'like' // Heart sticker
    }
  }
})
```

#### Generic Template

```typescript
await client.messages.send({
  channelId: 'ch_instagram_123',
  to: '17841234567890',
  content: {
    type: 'template',
    template: {
      type: 'generic',
      elements: [
        {
          title: 'Summer Collection',
          subtitle: 'Check out our latest arrivals',
          imageUrl: 'https://example.com/summer.jpg',
          buttons: [
            {
              type: 'web_url',
              title: 'Shop Now',
              url: 'https://example.com/summer-collection'
            },
            {
              type: 'postback',
              title: 'Learn More',
              payload: 'SUMMER_INFO'
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
  channelId: 'ch_instagram_123',
  to: '17841234567890',
  content: {
    type: 'text',
    text: 'What are you looking for today?',
    quickReplies: [
      {
        contentType: 'text',
        title: 'New Arrivals',
        payload: 'BROWSE_NEW'
      },
      {
        contentType: 'text',
        title: 'Best Sellers',
        payload: 'BROWSE_BEST'
      },
      {
        contentType: 'text',
        title: 'Sale Items',
        payload: 'BROWSE_SALE'
      },
      {
        contentType: 'text',
        title: 'Track Order',
        payload: 'TRACK_ORDER'
      }
    ]
  }
})
```

### Ice Breakers

Set up conversation starters that appear when users first message you:

```typescript
await client.instagram.setIceBreakers({
  channelId: 'ch_instagram_123',
  iceBreakers: [
    {
      question: 'What products do you have?',
      payload: 'GET_PRODUCTS'
    },
    {
      question: 'What are your store hours?',
      payload: 'GET_HOURS'
    },
    {
      question: 'How do I track my order?',
      payload: 'TRACK_ORDER'
    },
    {
      question: 'I need help with a return',
      payload: 'RETURNS_HELP'
    }
  ]
})
```

### Handling Story Mentions

When users mention your account in their stories:

```typescript
// Webhook event for story mention
{
  "event": "instagram.story_mention",
  "data": {
    "channelId": "ch_instagram_123",
    "from": "17841234567890",
    "storyId": "story_abc123",
    "mediaUrl": "https://...", // URL to the story media
    "timestamp": "2024-01-15T10:30:00Z",
    "metadata": {
      "username": "customer_user",
      "mentionText": "@yourbrand"
    }
  }
}

// Respond to story mention
await client.messages.send({
  channelId: 'ch_instagram_123',
  to: '17841234567890',
  content: {
    type: 'text',
    text: 'Thanks for the mention! We love seeing our products in action!'
  }
})
```

### Handling Story Replies

When users reply to your stories:

```typescript
// Webhook event for story reply
{
  "event": "instagram.story_reply",
  "data": {
    "channelId": "ch_instagram_123",
    "from": "17841234567890",
    "storyId": "story_xyz789",
    "replyText": "Love this!",
    "timestamp": "2024-01-15T10:30:00Z"
  }
}
```

### Receiving Messages

#### Webhook Events

```typescript
// Text message received
{
  "event": "message.received",
  "data": {
    "id": "msg_abc123",
    "channelId": "ch_instagram_123",
    "channelType": "instagram",
    "direction": "inbound",
    "from": "17841234567890",
    "content": {
      "type": "text",
      "text": "Hi! Do you have this in size medium?"
    },
    "timestamp": "2024-01-15T10:30:00Z",
    "metadata": {
      "mid": "aWdfZAG1...",
      "username": "customer_user"
    }
  }
}

// Image received
{
  "event": "message.received",
  "data": {
    "id": "msg_abc124",
    "channelId": "ch_instagram_123",
    "channelType": "instagram",
    "direction": "inbound",
    "from": "17841234567890",
    "content": {
      "type": "image",
      "image": {
        "url": "https://..."
      }
    },
    "timestamp": "2024-01-15T10:31:00Z"
  }
}

// Quick reply selected
{
  "event": "instagram.quick_reply",
  "data": {
    "channelId": "ch_instagram_123",
    "from": "17841234567890",
    "payload": "BROWSE_NEW",
    "text": "New Arrivals",
    "timestamp": "2024-01-15T10:32:00Z"
  }
}

// Postback from button
{
  "event": "instagram.postback",
  "data": {
    "channelId": "ch_instagram_123",
    "from": "17841234567890",
    "payload": "SUMMER_INFO",
    "timestamp": "2024-01-15T10:33:00Z"
  }
}
```

### Message Reactions

Handle when users react to messages:

```typescript
// Reaction received
{
  "event": "instagram.reaction",
  "data": {
    "channelId": "ch_instagram_123",
    "from": "17841234567890",
    "messageId": "msg_abc123",
    "reaction": "love", // love, laugh, wow, sad, angry, like
    "action": "react", // react or unreact
    "timestamp": "2024-01-15T10:34:00Z"
  }
}
```

### Product Shares (Instagram Shopping)

When users share products from your Instagram Shop:

```typescript
// Product share received
{
  "event": "instagram.product_share",
  "data": {
    "channelId": "ch_instagram_123",
    "from": "17841234567890",
    "product": {
      "id": "prod_123",
      "name": "Summer Dress",
      "price": "$79.99",
      "imageUrl": "https://...",
      "productUrl": "https://..."
    },
    "timestamp": "2024-01-15T10:35:00Z"
  }
}
```

## Webhook Setup

### Webhook URL Format

```
https://api.your-domain.com/webhooks/instagram/{channelId}
```

### Webhook Verification

Instagram uses the same verification as Facebook Messenger:

```
GET /webhooks/instagram/{channelId}?hub.mode=subscribe&hub.verify_token=YOUR_TOKEN&hub.challenge=CHALLENGE
```

### Webhook Security

Webhooks are validated using:
1. **Signature verification**: `X-Hub-Signature-256` header using App Secret
2. **Payload validation**: Ensures webhook matches expected format

## Common Issues and Troubleshooting

### "Message failed to send"

**Possible causes:**
- User hasn't messaged you first (policy requirement)
- Invalid IGSID
- Access token expired
- Outside messaging window

**Solution:**
- Ensure user initiated conversation
- Verify IGSID is correct
- Regenerate access token
- Check 24-hour messaging window

### "Permission denied"

**Possible causes:**
- Missing required permissions
- App not approved for Instagram Messaging
- Account not properly connected

**Solution:**
- Verify all required permissions are granted
- Complete Meta App Review for production
- Re-connect Instagram to Facebook Page

### "Webhook not receiving events"

**Possible causes:**
- Webhook not configured correctly
- Subscribe to wrong webhook fields
- Account level issues

**Solution:**
- Verify webhook URL and verify token
- Ensure subscribed to `messages` field
- Check Instagram account settings allow messaging

### "Can't send to user"

**Possible causes:**
- User blocked your account
- User has message controls enabled
- 24-hour window expired

**Solution:**
- Handle blocked users gracefully
- Respect user preferences
- Use ice breakers to re-engage

### Story mentions not received

**Possible causes:**
- Missing `instagram_manage_comments` permission
- Webhook not subscribed to story_mentions
- Story expired (24 hours)

**Solution:**
- Request and get approval for story mention permissions
- Add story_mentions to webhook subscriptions
- Process story events promptly

## Best Practices

1. **Respond Quickly**: Instagram users expect fast responses. Aim for under 1 hour.

2. **Use Visual Content**: Instagram is a visual platform. Include images when relevant.

3. **Personalize Responses**: Use username and conversation history to personalize.

4. **Set Up Ice Breakers**: Help users start conversations with suggested questions.

5. **Handle Story Mentions**: Respond to story mentions to build engagement and loyalty.

6. **Respect the 24-Hour Window**: Send proactive messages only within the messaging window.

7. **Keep Messages Concise**: Instagram DMs are meant for quick exchanges.

8. **Monitor Message Requests**: Check message requests folder for filtered messages.

## Policy Compliance

### Instagram Messaging Policy

- Users must initiate conversations (no cold outreach)
- Respond within 24 hours for standard messaging
- Don't send spam or promotional content unsolicited
- Respect user preferences and opt-out requests
- Use automation responsibly

### Message Window

| Type | Window |
|------|--------|
| Standard Response | 24 hours from last user message |
| Human Agent Tag | 7 days (requires human response) |

### Prohibited Content

- Spam or bulk unsolicited messages
- Adult content
- Illegal products or services
- Misleading or false information

## Rate Limits

| Action | Limit |
|--------|-------|
| Send API | 200 calls per user per hour |
| Overall API | Subject to Meta API limits |

## Limitations

Current Instagram API limitations:
- No scheduled messages
- No broadcast/bulk messaging
- Limited template types (compared to Messenger)
- No persistent menu (use ice breakers instead)
- Story mentions require additional permissions

## Next Steps

- [Flows](/flows/overview) - Build automated Instagram flows
- [AI Bots](/bots/overview) - Add AI to Instagram DMs
- [Instagram Shopping](/guides/instagram-shopping) - Handle product inquiries
- [Analytics](/api/analytics) - Track Instagram performance
