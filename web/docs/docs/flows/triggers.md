---
sidebar_position: 3
title: Flow Triggers
---

# Flow Triggers

Triggers determine when and how a flow starts. Every flow needs at least one trigger to activate it. Linktor supports multiple trigger types to handle different conversation scenarios.

## Trigger Types Overview

| Trigger | Description | Use Case |
|---------|-------------|----------|
| **Welcome** | New conversation starts | Greeting, onboarding |
| **Keyword** | Customer says specific words | Menu navigation, commands |
| **Intent** | AI detects customer intent | Natural language activation |
| **Event** | System event occurs | Status changes, external triggers |
| **Schedule** | Time-based activation | Follow-ups, reminders |
| **Manual** | Triggered via API/agent | Agent-initiated flows |
| **Webhook** | External system triggers | CRM events, app integrations |

## Welcome Trigger

Activates when a new conversation starts or when a customer returns after inactivity.

### Configuration

```typescript
{
  trigger: {
    type: 'welcome',
    conditions: {
      newConversation: true,      // First message in conversation
      returningAfter: 86400000,   // Returning after 24 hours
      channelTypes: ['whatsapp', 'telegram']  // Specific channels only
    },
    priority: 10  // Higher priority takes precedence
  }
}
```

### New Customer Welcome

```typescript
{
  trigger: {
    type: 'welcome',
    conditions: {
      newContact: true,  // Never messaged before
      channelTypes: null  // All channels
    }
  },
  nodes: [
    {
      id: 'welcome',
      type: 'message',
      content: 'Welcome to Acme! We are excited to have you. How can we help?'
    }
  ]
}
```

### Returning Customer Welcome

```typescript
{
  trigger: {
    type: 'welcome',
    conditions: {
      newContact: false,
      returningAfter: 604800000  // 7 days
    }
  },
  nodes: [
    {
      id: 'welcome_back',
      type: 'message',
      content: 'Welcome back, {{contact.name}}! Great to see you again.'
    }
  ]
}
```

### Channel-Specific Welcome

```typescript
{
  trigger: {
    type: 'welcome',
    conditions: {
      channelTypes: ['whatsapp']
    }
  },
  nodes: [
    {
      id: 'whatsapp_welcome',
      type: 'message',
      content: 'Hi! Thanks for reaching out on WhatsApp. Tap a button below to get started.',
      buttons: [
        { label: 'Browse Products', value: 'products' },
        { label: 'Track Order', value: 'track' },
        { label: 'Get Support', value: 'support' }
      ]
    }
  ]
}
```

## Keyword Trigger

Activates when a customer message contains specific words or phrases.

### Basic Keyword Trigger

```typescript
{
  trigger: {
    type: 'keyword',
    keywords: ['pricing', 'price', 'cost', 'how much'],
    matchType: 'contains',    // 'exact' | 'contains' | 'startsWith' | 'endsWith'
    caseSensitive: false
  }
}
```

### Match Types

| Type | Description | Example |
|------|-------------|---------|
| `exact` | Message equals keyword | "help" matches "help" only |
| `contains` | Message includes keyword | "I need help please" matches "help" |
| `startsWith` | Message begins with keyword | "track 12345" matches "track" |
| `endsWith` | Message ends with keyword | "please help" matches "help" |
| `word` | Keyword appears as complete word | "help me" matches, "helpful" doesn't |

### Regex Keywords

Use regular expressions for complex matching:

```typescript
{
  trigger: {
    type: 'keyword',
    patterns: [
      '^track\\s+#?\\d{5,}$',         // "track #12345" or "track 12345"
      'order\\s+(status|number)',      // "order status" or "order number"
      '(?:what|how).*(?:return|refund)'  // Questions about returns
    ],
    matchType: 'regex'
  }
}
```

### Keyword Groups

Organize keywords into logical groups:

```typescript
{
  trigger: {
    type: 'keyword',
    groups: [
      {
        name: 'order_tracking',
        keywords: ['track', 'where is', 'order status', 'delivery'],
        next: 'track_order'
      },
      {
        name: 'returns',
        keywords: ['return', 'refund', 'exchange', 'send back'],
        next: 'returns_flow'
      },
      {
        name: 'human',
        keywords: ['agent', 'human', 'person', 'speak to someone'],
        next: 'escalate'
      }
    ]
  }
}
```

### Keyword Priority

When multiple flows match, priority determines which activates:

```typescript
// High priority - specific keyword
{
  trigger: {
    type: 'keyword',
    keywords: ['cancel subscription'],
    priority: 100
  }
}

// Lower priority - general keyword
{
  trigger: {
    type: 'keyword',
    keywords: ['cancel'],
    priority: 50
  }
}
```

## Intent Trigger

Uses AI to understand customer intent and activate flows accordingly.

### Basic Intent Trigger

```typescript
{
  trigger: {
    type: 'intent',
    intents: ['order_tracking', 'shipping_inquiry'],
    confidenceThreshold: 0.75
  }
}
```

### Configuring Intents

Define what each intent means:

```typescript
{
  intents: {
    order_tracking: {
      description: 'Customer wants to know the status of their order',
      examples: [
        'Where is my package?',
        "I haven't received my order yet",
        'Can you track my shipment?',
        'When will my order arrive?'
      ]
    },
    product_inquiry: {
      description: 'Customer is asking about products or services',
      examples: [
        'What products do you have?',
        'Tell me about your services',
        'Do you sell electronics?',
        'What are your bestsellers?'
      ]
    },
    pricing_question: {
      description: 'Customer wants pricing information',
      examples: [
        'How much does this cost?',
        'What are your prices?',
        'Is there a discount?',
        'Do you have any deals?'
      ]
    }
  }
}
```

### Intent with Entities

Extract entities along with intent detection:

```typescript
{
  trigger: {
    type: 'intent',
    intents: ['book_appointment'],
    extractEntities: true,
    entities: ['date', 'time', 'service_type']
  }
}
```

When triggered, entities are available as variables:
```typescript
// Customer says: "I want to book a haircut for tomorrow at 3pm"
// Variables set:
// {{intent}} = 'book_appointment'
// {{entity_date}} = '2024-01-16'
// {{entity_time}} = '15:00'
// {{entity_service_type}} = 'haircut'
```

### Fallback Intent

Handle messages that don't match any specific intent:

```typescript
{
  trigger: {
    type: 'intent',
    intents: ['fallback', 'unknown', 'other'],
    priority: -100  // Lowest priority
  },
  nodes: [
    {
      id: 'fallback',
      type: 'ai_response',
      botId: 'bot_general',
      prompt: 'Answer the customer query: {{last_message}}'
    }
  ]
}
```

## Event Trigger

Activates when specific system events occur.

### Supported Events

| Event | Description |
|-------|-------------|
| `conversation.assigned` | Conversation assigned to agent/inbox |
| `conversation.unassigned` | Conversation unassigned |
| `conversation.resolved` | Conversation marked resolved |
| `conversation.reopened` | Resolved conversation reopened |
| `agent.joined` | Agent joined conversation |
| `agent.left` | Agent left conversation |
| `contact.tagged` | Tag added to contact |
| `contact.updated` | Contact profile updated |
| `order.created` | New order created (integration) |
| `order.shipped` | Order shipped (integration) |
| `payment.received` | Payment received (integration) |

### Event Trigger Configuration

```typescript
{
  trigger: {
    type: 'event',
    event: 'conversation.resolved',
    conditions: {
      resolvedBy: 'agent',   // 'bot' | 'agent' | 'system'
      minDuration: 300       // At least 5 minutes
    }
  },
  nodes: [
    {
      id: 'satisfaction_survey',
      type: 'message',
      content: 'Thanks for chatting with us! How would you rate your experience?',
      buttons: [
        { label: 'Great', value: '5' },
        { label: 'Good', value: '4' },
        { label: 'OK', value: '3' },
        { label: 'Poor', value: '2' },
        { label: 'Bad', value: '1' }
      ]
    }
  ]
}
```

### Custom Event Trigger

Listen for custom events from your application:

```typescript
{
  trigger: {
    type: 'event',
    event: 'custom.order_delivered',
    dataMapping: {
      'orderId': 'order_id',
      'deliveryDate': 'delivered_at'
    }
  },
  nodes: [
    {
      id: 'delivery_confirmation',
      type: 'message',
      content: 'Your order #{{order_id}} has been delivered! We hope you love it.'
    }
  ]
}
```

## Schedule Trigger

Time-based triggers for automated follow-ups and reminders.

### Delay After Event

```typescript
{
  trigger: {
    type: 'schedule',
    after: {
      event: 'conversation.last_message',
      delay: 3600000  // 1 hour after last message
    },
    conditions: {
      status: 'open',
      lastMessageFrom: 'bot'  // Bot sent last message
    }
  },
  nodes: [
    {
      id: 'followup',
      type: 'message',
      content: "Hi! Just checking in. Do you need any more help with your question?"
    }
  ]
}
```

### Fixed Schedule

```typescript
{
  trigger: {
    type: 'schedule',
    schedule: {
      type: 'cron',
      expression: '0 9 * * 1',  // Every Monday at 9 AM
      timezone: 'America/New_York'
    },
    targetAudience: {
      tags: ['newsletter_subscriber'],
      lastActiveWithin: 604800000  // Active in last 7 days
    }
  },
  nodes: [
    {
      id: 'weekly_update',
      type: 'message',
      content: 'Good morning! Here are this week top deals...'
    }
  ]
}
```

### Abandoned Cart Reminder

```typescript
{
  trigger: {
    type: 'schedule',
    after: {
      event: 'custom.cart_abandoned',
      delay: 3600000  // 1 hour
    }
  },
  nodes: [
    {
      id: 'cart_reminder',
      type: 'message',
      content: 'Hey {{contact.name}}! You left some items in your cart. Ready to complete your purchase?'
    }
  ]
}
```

## Manual Trigger

Flows that are started programmatically or by agents.

### Agent-Initiated

```typescript
{
  trigger: {
    type: 'manual',
    allowedInitiators: ['agent'],
    requiredRole: 'support',  // Agent must have this role
    showInMenu: true,
    menuLabel: 'Send Product Catalog'
  }
}
```

### API-Initiated

```typescript
{
  trigger: {
    type: 'manual',
    allowedInitiators: ['api'],
    webhookSecret: 'your-secret-key'  // For authentication
  }
}
```

Trigger via API:

```typescript
await client.flows.trigger('flow_123', {
  conversationId: 'conv_abc',
  variables: {
    product_id: 'prod_xyz',
    discount_code: 'SAVE20'
  }
})
```

## Webhook Trigger

Start flows from external systems.

### Configuration

```typescript
{
  trigger: {
    type: 'webhook',
    authentication: {
      type: 'bearer',     // 'bearer' | 'basic' | 'hmac' | 'none'
      token: 'your-token'
    },
    mapping: {
      'body.customer_id': 'contact_id',
      'body.event_data.order': 'order_details'
    }
  }
}
```

### Webhook Endpoint

Each flow with a webhook trigger gets a unique URL:

```
POST https://api.linktor.io/v1/webhooks/flows/{flow_id}/trigger
```

Example request:

```bash
curl -X POST \
  https://api.linktor.io/v1/webhooks/flows/flow_123/trigger \
  -H "Authorization: Bearer your-token" \
  -H "Content-Type: application/json" \
  -d '{
    "customer_id": "cust_123",
    "event_data": {
      "order": {
        "id": "order_456",
        "status": "shipped"
      }
    }
  }'
```

## Trigger Priority and Conflicts

When multiple flows could be triggered by the same message:

### Priority Levels

```typescript
{
  trigger: {
    type: 'keyword',
    keywords: ['help'],
    priority: 50  // Default is 0
  }
}
```

Higher priority flows take precedence.

### Conflict Resolution

| Scenario | Resolution |
|----------|------------|
| Same priority | Most specific trigger wins |
| Exact vs. contains | Exact match wins |
| Keyword vs. intent | Keyword wins (faster) |
| Multiple intents | Highest confidence wins |

### Exclusive Triggers

Prevent other flows from activating:

```typescript
{
  trigger: {
    type: 'keyword',
    keywords: ['stop', 'unsubscribe'],
    exclusive: true,  // No other flows can process this message
    priority: 1000    // Highest priority
  }
}
```

## Trigger Conditions

Add conditions to any trigger type.

### Channel Conditions

```typescript
{
  trigger: {
    type: 'keyword',
    keywords: ['menu'],
    conditions: {
      channelTypes: ['whatsapp', 'telegram'],
      excludeChannels: ['sms']  // SMS doesn't support buttons
    }
  }
}
```

### Contact Conditions

```typescript
{
  trigger: {
    type: 'intent',
    intents: ['pricing'],
    conditions: {
      contact: {
        tags: { includes: 'lead' },
        customField: { 'tier': 'enterprise' }
      }
    }
  }
}
```

### Time Conditions

```typescript
{
  trigger: {
    type: 'welcome',
    conditions: {
      time: {
        days: ['monday', 'tuesday', 'wednesday', 'thursday', 'friday'],
        hours: { start: '09:00', end: '18:00' },
        timezone: 'America/New_York'
      }
    }
  }
}
```

### Conversation Conditions

```typescript
{
  trigger: {
    type: 'keyword',
    keywords: ['escalate'],
    conditions: {
      conversation: {
        status: 'open',
        assignedTo: { type: 'bot' },  // Currently with bot
        messageCount: { min: 3 }       // At least 3 messages exchanged
      }
    }
  }
}
```

## Testing Triggers

### Trigger Tester

Test which flow would activate for a given message:

```typescript
const result = await client.flows.testTrigger({
  message: 'I want to track my order #12345',
  channelType: 'whatsapp',
  contactId: 'contact_abc'
})

console.log(result.matchedFlow)      // Flow that would trigger
console.log(result.trigger)          // Which trigger matched
console.log(result.extractedData)    // Entities or variables extracted
console.log(result.allMatches)       // All flows that matched (by priority)
```

### Debug Mode

Enable verbose logging for trigger evaluation:

```typescript
{
  trigger: {
    type: 'intent',
    intents: ['order_status'],
    debug: true  // Log detailed trigger evaluation
  }
}
```

## Best Practices

1. **Be specific with keywords**: Use phrases rather than single words when possible

2. **Set appropriate priorities**: Important flows should have higher priority

3. **Use intents for natural language**: Keywords for commands, intents for questions

4. **Test thoroughly**: Use the trigger tester to verify expected behavior

5. **Document triggers**: Add descriptions to explain when flows should activate

6. **Handle conflicts**: Define clear priority rules for overlapping triggers

7. **Consider channel differences**: Some triggers may not work on all channels

## Next Steps

- [Node Types](/flows/node-types) - Build your flow logic
- [Flows Overview](/flows/overview) - Flow concepts and best practices
- [Bots Overview](/bots/overview) - Combine triggers with AI
