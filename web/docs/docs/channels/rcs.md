---
sidebar_position: 10
title: RCS
---

# RCS Business Messaging Integration

Connect RCS (Rich Communication Services) to Linktor and deliver rich, app-like messaging experiences directly in users' native messaging apps. RCS is the next evolution of SMS, offering features like carousels, suggested actions, and read receipts.

## Overview

RCS Business Messaging enables you to:

- Send rich media messages (images, videos, files)
- Use carousels for product showcases
- Add suggested replies and actions
- Display branded sender information (logo, name, verification)
- Track delivery, read receipts, and typing indicators
- Fall back to SMS when RCS is unavailable
- Reach users without requiring app downloads

## Prerequisites

Before configuring RCS in Linktor, you'll need:

1. **RCS Provider Account**: An account with a supported RCS Business Messaging provider
2. **Verified Business**: Business verification with Google or your RCS provider
3. **Agent Registration**: A registered RCS Business Messaging agent

### Supported Providers

| Provider | Coverage | Features |
|----------|----------|----------|
| **Google RCS Business Messaging** | Global (carrier dependent) | Full feature set |
| **Sinch** | Global | RCS + SMS fallback |
| **Infobip** | Global | Omnichannel support |
| **Vonage** | Global | API-first approach |
| **Twilio** | US (expanding) | Unified messaging API |

### RCS Availability

RCS is available on:
- Android devices with Google Messages (or carrier RCS app)
- Support varies by carrier and country
- iOS support expected in future (Apple announced RCS adoption)

## Configuration in Linktor

### Google RCS Business Messaging Setup

#### Step 1: Register as RCS Partner

1. Apply at [Google RCS Business Messaging](https://developers.google.com/business-communications/rcs-business-messaging)
2. Complete business verification
3. Create an RCS agent (your business identity)
4. Get your agent approved

#### Step 2: Get Credentials

From the Business Communications Console:
1. Create a service account
2. Download the JSON key file
3. Note your Agent ID

#### Step 3: Add Channel in Dashboard

1. Go to **Settings → Channels** in Linktor
2. Click **Add Channel** and select **RCS**
3. Choose **Google** as the provider
4. Fill in the configuration:

| Field | Description |
|-------|-------------|
| **Name** | Display name (e.g., "RCS Messaging") |
| **Agent ID** | Your RCS agent ID |
| **Service Account JSON** | Upload or paste service account credentials |
| **Fallback SMS Channel** | Optional SMS channel for fallback |

5. Click **Save** to create the channel

### Sinch RCS Setup

#### Step 1: Create Sinch Account

1. Sign up at [sinch.com](https://www.sinch.com)
2. Apply for RCS access
3. Complete brand verification
4. Create an RCS agent

#### Step 2: Get Credentials

1. Note your **Project ID**
2. Generate an **API Token**
3. Get your **Agent/Bot ID**

#### Step 3: Add Channel in Dashboard

| Field | Description |
|-------|-------------|
| **Name** | Display name for this channel |
| **Project ID** | Your Sinch Project ID |
| **API Token** | Your Sinch API token |
| **Bot ID** | Your RCS bot/agent ID |

## API Usage

### Sending Messages

#### Text Message

```typescript
import { Linktor } from '@linktor/sdk'

const client = new Linktor({ apiKey: 'YOUR_API_KEY' })

await client.messages.send({
  channelId: 'ch_rcs_123',
  to: '+5511999999999',
  content: {
    type: 'text',
    text: 'Hello! Welcome to our RCS experience. How can we help you today?'
  }
})
```

#### Rich Card

```typescript
await client.messages.send({
  channelId: 'ch_rcs_123',
  to: '+5511999999999',
  content: {
    type: 'richCard',
    richCard: {
      standaloneCard: {
        cardOrientation: 'VERTICAL',
        thumbnailImageAlignment: 'LEFT',
        cardContent: {
          title: 'iPhone 15 Pro',
          description: 'The most advanced iPhone ever. Starting at $999.',
          media: {
            height: 'MEDIUM',
            contentInfo: {
              fileUrl: 'https://example.com/iphone.jpg',
              forceRefresh: false
            }
          },
          suggestions: [
            {
              action: {
                text: 'Buy Now',
                postbackData: 'buy_iphone_15',
                openUrlAction: {
                  url: 'https://example.com/buy/iphone-15'
                }
              }
            },
            {
              reply: {
                text: 'Learn More',
                postbackData: 'learn_more_iphone_15'
              }
            }
          ]
        }
      }
    }
  }
})
```

#### Carousel (Rich Card Carousel)

```typescript
await client.messages.send({
  channelId: 'ch_rcs_123',
  to: '+5511999999999',
  content: {
    type: 'richCard',
    richCard: {
      carouselCard: {
        cardWidth: 'MEDIUM',
        cardContents: [
          {
            title: 'iPhone 15 Pro',
            description: 'Starting at $999',
            media: {
              height: 'MEDIUM',
              contentInfo: {
                fileUrl: 'https://example.com/iphone.jpg'
              }
            },
            suggestions: [
              {
                action: {
                  text: 'View',
                  postbackData: 'view_iphone',
                  openUrlAction: { url: 'https://example.com/iphone' }
                }
              }
            ]
          },
          {
            title: 'MacBook Pro',
            description: 'Starting at $1,999',
            media: {
              height: 'MEDIUM',
              contentInfo: {
                fileUrl: 'https://example.com/macbook.jpg'
              }
            },
            suggestions: [
              {
                action: {
                  text: 'View',
                  postbackData: 'view_macbook',
                  openUrlAction: { url: 'https://example.com/macbook' }
                }
              }
            ]
          },
          {
            title: 'AirPods Pro',
            description: 'Starting at $249',
            media: {
              height: 'MEDIUM',
              contentInfo: {
                fileUrl: 'https://example.com/airpods.jpg'
              }
            },
            suggestions: [
              {
                action: {
                  text: 'View',
                  postbackData: 'view_airpods',
                  openUrlAction: { url: 'https://example.com/airpods' }
                }
              }
            ]
          }
        ]
      }
    }
  }
})
```

#### Suggested Replies

```typescript
await client.messages.send({
  channelId: 'ch_rcs_123',
  to: '+5511999999999',
  content: {
    type: 'text',
    text: 'How would you like to proceed?',
    suggestions: [
      {
        reply: {
          text: 'Check Order Status',
          postbackData: 'check_order'
        }
      },
      {
        reply: {
          text: 'Browse Products',
          postbackData: 'browse_products'
        }
      },
      {
        reply: {
          text: 'Contact Support',
          postbackData: 'contact_support'
        }
      }
    ]
  }
})
```

#### Suggested Actions

```typescript
await client.messages.send({
  channelId: 'ch_rcs_123',
  to: '+5511999999999',
  content: {
    type: 'text',
    text: 'Here are some quick actions:',
    suggestions: [
      {
        action: {
          text: 'Call Us',
          postbackData: 'call_action',
          dialAction: {
            phoneNumber: '+15551234567'
          }
        }
      },
      {
        action: {
          text: 'Get Directions',
          postbackData: 'directions_action',
          openUrlAction: {
            url: 'https://maps.google.com/?q=123+Main+St'
          }
        }
      },
      {
        action: {
          text: 'Share Location',
          postbackData: 'location_action',
          shareLocationAction: {}
        }
      },
      {
        action: {
          text: 'Add to Calendar',
          postbackData: 'calendar_action',
          createCalendarEventAction: {
            startTime: '2024-01-20T14:00:00Z',
            endTime: '2024-01-20T15:00:00Z',
            title: 'Appointment with Acme Inc',
            description: 'Product demo session'
          }
        }
      }
    ]
  }
})
```

#### Image Message

```typescript
await client.messages.send({
  channelId: 'ch_rcs_123',
  to: '+5511999999999',
  content: {
    type: 'image',
    image: {
      url: 'https://example.com/product.jpg',
      thumbnailUrl: 'https://example.com/product-thumb.jpg'
    }
  }
})
```

#### Video Message

```typescript
await client.messages.send({
  channelId: 'ch_rcs_123',
  to: '+5511999999999',
  content: {
    type: 'video',
    video: {
      url: 'https://example.com/demo.mp4',
      thumbnailUrl: 'https://example.com/demo-thumb.jpg'
    }
  }
})
```

### SMS Fallback

Configure automatic SMS fallback when RCS is unavailable:

```typescript
await client.messages.send({
  channelId: 'ch_rcs_123',
  to: '+5511999999999',
  content: {
    type: 'text',
    text: 'Your order has shipped! Track it here: https://track.example.com/12345'
  },
  options: {
    fallback: {
      enabled: true,
      channel: 'ch_sms_123', // Your SMS channel
      timeout: 30000, // Wait 30s for RCS delivery before fallback
      content: {
        type: 'text',
        text: 'Your order has shipped! Track: https://track.example.com/12345'
      }
    }
  }
})
```

### Checking RCS Capability

```typescript
// Check if a number supports RCS
const capability = await client.rcs.checkCapability({
  channelId: 'ch_rcs_123',
  phoneNumber: '+5511999999999'
})

console.log(capability)
// {
//   phoneNumber: '+5511999999999',
//   rcsEnabled: true,
//   features: ['RICHCARD_STANDALONE', 'RICHCARD_CAROUSEL', 'ACTION_DIAL', 'ACTION_URL'],
//   timestamp: '2024-01-15T10:30:00Z'
// }

// Batch capability check
const capabilities = await client.rcs.checkCapabilityBatch({
  channelId: 'ch_rcs_123',
  phoneNumbers: ['+5511999999999', '+5511888888888', '+5511777777777']
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
    "channelId": "ch_rcs_123",
    "channelType": "rcs",
    "direction": "inbound",
    "from": "+5511999999999",
    "content": {
      "type": "text",
      "text": "I want to check my order status"
    },
    "timestamp": "2024-01-15T10:30:00Z",
    "metadata": {
      "messageId": "rcs_msg_xyz",
      "agentId": "brands/abc/agents/123"
    }
  }
}

// Suggested reply selected
{
  "event": "rcs.suggestion_response",
  "data": {
    "channelId": "ch_rcs_123",
    "from": "+5511999999999",
    "text": "Check Order Status",
    "postbackData": "check_order",
    "timestamp": "2024-01-15T10:30:05Z"
  }
}

// Location shared
{
  "event": "rcs.location",
  "data": {
    "channelId": "ch_rcs_123",
    "from": "+5511999999999",
    "location": {
      "latitude": -23.5505,
      "longitude": -46.6333
    },
    "timestamp": "2024-01-15T10:30:10Z"
  }
}
```

#### Delivery and Read Receipts

```typescript
// Delivery receipt
{
  "event": "message.status",
  "data": {
    "messageId": "msg_abc123",
    "status": "delivered",
    "timestamp": "2024-01-15T10:30:02Z"
  }
}

// Read receipt
{
  "event": "message.status",
  "data": {
    "messageId": "msg_abc123",
    "status": "read",
    "timestamp": "2024-01-15T10:30:10Z"
  }
}
```

#### User Events

```typescript
// User started typing
{
  "event": "rcs.typing_started",
  "data": {
    "channelId": "ch_rcs_123",
    "from": "+5511999999999",
    "timestamp": "2024-01-15T10:30:15Z"
  }
}

// User stopped typing
{
  "event": "rcs.typing_stopped",
  "data": {
    "channelId": "ch_rcs_123",
    "from": "+5511999999999",
    "timestamp": "2024-01-15T10:30:20Z"
  }
}
```

## Webhook Setup

### Webhook URL Format

```
https://api.your-domain.com/webhooks/rcs/{channelId}
```

### Google RCS Webhook

Google RCS uses a push subscription model:

```typescript
// Configure in Google Business Communications Console
// Or programmatically:
await client.rcs.configureWebhook({
  channelId: 'ch_rcs_123',
  webhookUrl: 'https://api.your-domain.com/webhooks/rcs/ch_rcs_123'
})
```

### Webhook Security

RCS webhooks are validated using:
1. **Service account authentication**: Google-signed JWTs
2. **Signature verification**: Provider-specific signatures

## RCS Agent Branding

### Set Up Agent Information

```typescript
// Agent information is configured during registration
// Key branding elements:
{
  "displayName": "Acme Store",
  "logoUrl": "https://example.com/logo.png",
  "heroImageUrl": "https://example.com/hero.jpg",
  "color": "#0066FF",
  "description": "Your one-stop shop for electronics",
  "privacyPolicy": "https://example.com/privacy",
  "termsOfService": "https://example.com/terms"
}
```

## Common Issues and Troubleshooting

### "Message not delivered"

**Possible causes:**
- Recipient doesn't support RCS
- Device offline
- Carrier doesn't support RCS

**Solution:**
- Check RCS capability before sending
- Enable SMS fallback
- Verify carrier support in target region

### "Agent not verified"

**Possible causes:**
- Business verification incomplete
- Agent approval pending
- Documentation issues

**Solution:**
- Complete business verification process
- Submit required documentation
- Contact RCS provider support

### "Rich card not displaying"

**Possible causes:**
- Image URL not accessible
- Image format not supported
- Card structure invalid

**Solution:**
- Use publicly accessible HTTPS URLs
- Use supported formats (JPEG, PNG)
- Validate card structure against schema

### "Suggestions not showing"

**Possible causes:**
- Too many suggestions (max varies by type)
- Invalid postback data
- Text too long

**Solution:**
- Limit suggestions (typically max 11)
- Keep postback data under limits
- Shorten suggestion text (max 25 chars for replies)

### "Capability check failed"

**Possible causes:**
- Invalid phone number format
- Rate limited
- Service unavailable

**Solution:**
- Use E.164 format
- Implement rate limiting
- Cache capability results

## Best Practices

1. **Check Capabilities First**: Always verify RCS support before sending rich messages.

2. **Enable SMS Fallback**: Configure fallback for users without RCS support.

3. **Use Rich Cards Appropriately**: Use carousels for multiple items, standalone cards for single items.

4. **Keep Suggestions Concise**: Suggestion text should be short and action-oriented.

5. **Optimize Media**: Use appropriate image sizes and compress media files.

6. **Brand Consistently**: Maintain consistent branding across your agent.

7. **Test on Multiple Devices**: RCS rendering varies by device and app version.

8. **Monitor Delivery Rates**: Track RCS vs SMS fallback usage.

## Supported Features by Platform

| Feature | Google Messages | Samsung Messages | Carrier Apps |
|---------|:---------------:|:----------------:|:------------:|
| Text Messages | Yes | Yes | Yes |
| Rich Cards | Yes | Yes | Varies |
| Carousels | Yes | Yes | Varies |
| Suggested Replies | Yes | Yes | Yes |
| Suggested Actions | Yes | Yes | Varies |
| Read Receipts | Yes | Yes | Yes |
| Typing Indicators | Yes | Yes | Varies |

## Rate Limits

| Provider | Limit | Notes |
|----------|-------|-------|
| Google RCS | 60 msgs/min per agent | Can request increase |
| Sinch | Based on plan | Enterprise plans higher |
| Infobip | Based on plan | Contact for limits |

## Pricing

RCS pricing varies by:
- **Provider**: Different base rates
- **Country**: International rates vary
- **Volume**: Bulk discounts available
- **Fallback**: SMS fallback adds cost

## Next Steps

- [Flows](/flows/overview) - Build automated RCS flows
- [Rich Card Designer](/guides/rcs-rich-cards) - Design rich cards visually
- [SMS Integration](/channels/sms) - Configure SMS fallback
- [Analytics](/api/analytics) - Track RCS performance
