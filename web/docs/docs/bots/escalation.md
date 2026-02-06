---
sidebar_position: 3
title: Human Handoff & Escalation
---

# Human Handoff & Escalation

Bots are powerful, but sometimes customers need to speak with a human. Linktor provides flexible escalation rules to ensure smooth handoffs between bots and human agents.

## Why Escalation Matters

Even the best AI bots have limitations. Effective escalation ensures:

- **Customer satisfaction**: Frustrated customers get human help quickly
- **Complex issue resolution**: Nuanced problems get expert attention
- **Compliance**: Sensitive topics are handled by trained staff
- **Continuous improvement**: Escalation data helps improve bot training

## Escalation Triggers

### Keyword-Based Triggers

Escalate when customers use specific words or phrases:

```typescript
{
  escalation: {
    triggers: {
      keywords: {
        enabled: true,
        terms: [
          'speak to human',
          'talk to agent',
          'real person',
          'manager',
          'supervisor',
          'complaint',
          'lawyer',
          'legal'
        ],
        caseSensitive: false,
        matchType: 'contains'  // 'exact' | 'contains' | 'startsWith'
      }
    }
  }
}
```

### Intent-Based Triggers

Use AI to detect escalation intent:

```typescript
{
  escalation: {
    triggers: {
      intent: {
        enabled: true,
        intents: [
          'request_human',
          'express_frustration',
          'demand_refund',
          'report_bug'
        ],
        confidenceThreshold: 0.8
      }
    }
  }
}
```

### Sentiment-Based Triggers

Escalate based on customer sentiment:

```typescript
{
  escalation: {
    triggers: {
      sentiment: {
        enabled: true,
        threshold: -0.6,        // Escalate below this score (-1 to 1)
        consecutiveNegative: 3, // Or after N consecutive negative messages
        includeNeutral: false
      }
    }
  }
}
```

### Repetition-Based Triggers

Escalate when the bot fails repeatedly:

```typescript
{
  escalation: {
    triggers: {
      repetition: {
        enabled: true,
        maxRetries: 3,          // Same question asked N times
        maxFallbacks: 2,        // Bot falls back N times
        maxLoops: 5             // Conversation loops N times
      }
    }
  }
}
```

### Topic-Based Triggers

Automatically escalate for specific topics:

```typescript
{
  escalation: {
    triggers: {
      topics: {
        enabled: true,
        escalateTopics: [
          'billing_dispute',
          'account_cancellation',
          'security_incident',
          'harassment_report'
        ],
        useAIClassification: true
      }
    }
  }
}
```

### Time-Based Triggers

Escalate long-running conversations:

```typescript
{
  escalation: {
    triggers: {
      time: {
        enabled: true,
        maxDuration: 900,       // Conversation over 15 minutes
        maxMessages: 20,        // Over 20 messages
        maxBotResponses: 10     // Bot has responded 10+ times
      }
    }
  }
}
```

## Escalation Actions

### Route to Inbox

Send the conversation to a shared inbox for agents:

```typescript
{
  escalation: {
    action: {
      type: 'inbox',
      inboxId: 'inbox_support',
      priority: 'normal',      // 'low' | 'normal' | 'high' | 'urgent'
      tags: ['escalated', 'bot-handoff']
    }
  }
}
```

### Route to Specific Agent

Assign to a specific agent based on criteria:

```typescript
{
  escalation: {
    action: {
      type: 'agent',
      assignment: {
        strategy: 'round_robin', // 'round_robin' | 'least_busy' | 'skills' | 'specific'
        agentId: null,           // For 'specific' strategy
        teamId: 'team_billing',  // Filter by team
        skills: ['billing', 'refunds'],  // For 'skills' strategy
        fallbackInbox: 'inbox_general'
      }
    }
  }
}
```

### Route to External System

Integrate with external ticketing systems:

```typescript
{
  escalation: {
    action: {
      type: 'webhook',
      webhook: {
        url: 'https://your-crm.com/api/tickets',
        method: 'POST',
        headers: {
          'Authorization': 'Bearer {{CRM_TOKEN}}'
        },
        bodyTemplate: {
          customerId: '{{contact.id}}',
          conversationId: '{{conversation.id}}',
          transcript: '{{conversation.transcript}}',
          reason: '{{escalation.reason}}'
        }
      }
    }
  }
}
```

### Create Callback Request

Schedule a callback for the customer:

```typescript
{
  escalation: {
    action: {
      type: 'callback',
      callback: {
        collectPhone: true,
        collectPreferredTime: true,
        availableSlots: 'business_hours',
        confirmationMessage: "We'll call you back within 2 hours."
      }
    }
  }
}
```

## Escalation Messages

### Pre-Escalation Messages

Inform the customer before escalating:

```typescript
{
  escalation: {
    messages: {
      preEscalation: "I'm going to connect you with one of our support specialists who can better assist you.",
      collectingInfo: "Before I transfer you, could you briefly describe the issue?",
      transferring: "Transferring you now. Please hold..."
    }
  }
}
```

### Queue Messages

Keep customers informed while waiting:

```typescript
{
  escalation: {
    messages: {
      inQueue: "You're now in queue. An agent will be with you shortly.",
      queuePosition: "You're number {{position}} in queue. Estimated wait: {{wait_time}}.",
      agentAssigned: "Great news! {{agent.name}} is joining the conversation."
    }
  }
}
```

### Offline Messages

Handle escalations outside business hours:

```typescript
{
  escalation: {
    messages: {
      agentsOffline: "Our team is currently offline. We'll get back to you when we reopen.",
      leaveMessage: "Please leave a message and we'll respond as soon as possible.",
      confirmationOffline: "Thanks! We've received your message and will respond within 24 hours."
    }
  }
}
```

## Handoff Context

### Conversation Summary

Automatically generate a summary for agents:

```typescript
{
  escalation: {
    handoff: {
      includeSummary: true,
      summaryPrompt: "Summarize this conversation in 2-3 sentences for the support agent, highlighting the main issue and any relevant details.",
      maxSummaryLength: 500
    }
  }
}
```

### Transcript Options

```typescript
{
  escalation: {
    handoff: {
      includeFullTranscript: true,
      transcriptLimit: 50,     // Last N messages
      highlightKeyMessages: true,
      includeMetadata: true    // Timestamps, sentiment scores, etc.
    }
  }
}
```

### Customer Context

```typescript
{
  escalation: {
    handoff: {
      includeCustomerProfile: true,
      includeConversationHistory: true,  // Previous conversations
      includePurchaseHistory: true,
      includeCustomFields: ['tier', 'account_manager']
    }
  }
}
```

## Routing Rules

### Conditional Routing

Route to different destinations based on conditions:

```typescript
{
  escalation: {
    routing: {
      rules: [
        {
          conditions: {
            topic: 'billing',
            customerTier: 'enterprise'
          },
          action: {
            type: 'agent',
            teamId: 'team_enterprise_billing',
            priority: 'high'
          }
        },
        {
          conditions: {
            topic: 'technical',
            productCategory: 'software'
          },
          action: {
            type: 'inbox',
            inboxId: 'inbox_tech_support'
          }
        },
        {
          // Default fallback
          conditions: {},
          action: {
            type: 'inbox',
            inboxId: 'inbox_general'
          }
        }
      ]
    }
  }
}
```

### Skills-Based Routing

Match agents to customer needs:

```typescript
{
  escalation: {
    routing: {
      skillsMatching: {
        enabled: true,
        requiredSkills: '{{detected_topic}}',
        preferredLanguage: '{{customer.language}}',
        fallbackBehavior: 'queue_any'  // 'queue_any' | 'wait' | 'callback'
      }
    }
  }
}
```

## Priority Management

### Priority Levels

```typescript
{
  escalation: {
    priority: {
      rules: [
        { condition: { customerTier: 'enterprise' }, priority: 'high' },
        { condition: { sentiment: { lt: -0.8 } }, priority: 'urgent' },
        { condition: { waitTime: { gt: 300 } }, priority: 'high' },
        { default: true, priority: 'normal' }
      ]
    }
  }
}
```

### SLA Configuration

```typescript
{
  escalation: {
    sla: {
      urgent: {
        firstResponse: 60,    // 1 minute
        resolution: 3600      // 1 hour
      },
      high: {
        firstResponse: 300,   // 5 minutes
        resolution: 14400     // 4 hours
      },
      normal: {
        firstResponse: 900,   // 15 minutes
        resolution: 86400     // 24 hours
      },
      low: {
        firstResponse: 3600,  // 1 hour
        resolution: 172800    // 48 hours
      }
    }
  }
}
```

## Agent Experience

### Conversation Takeover

When an agent joins, configure bot behavior:

```typescript
{
  escalation: {
    agentTakeover: {
      botBehavior: 'pause',     // 'pause' | 'assist' | 'exit'
      assistMode: {
        suggestResponses: true,
        provideContext: true,
        answerInternalQueries: true  // Agent can ask bot questions
      },
      returnToBot: {
        enabled: true,
        trigger: 'agent_close',   // 'agent_close' | 'manual' | 'timeout'
        confirmWithCustomer: true
      }
    }
  }
}
```

### Agent Assist Mode

Bot helps agent during conversation:

```typescript
{
  escalation: {
    agentAssist: {
      enabled: true,
      features: {
        suggestedResponses: true,
        knowledgeSearch: true,
        sentimentIndicator: true,
        customerInsights: true,
        nextBestAction: true
      }
    }
  }
}
```

## Monitoring and Analytics

Track escalation performance:

| Metric | Description |
|--------|-------------|
| **Escalation Rate** | % of conversations escalated |
| **Escalation Reasons** | Breakdown by trigger type |
| **Time to Escalation** | Average time before escalation |
| **Resolution After Escalation** | % resolved by agents |
| **Customer Satisfaction** | CSAT for escalated conversations |
| **Return Rate** | Customers who escalate again |

### Escalation Dashboard

View escalation analytics in the dashboard:

```
Dashboard → Analytics → Escalations
```

### Webhooks for Monitoring

```typescript
{
  escalation: {
    webhooks: {
      onEscalation: 'https://your-api.com/hooks/escalation',
      onAgentAssigned: 'https://your-api.com/hooks/assigned',
      onResolved: 'https://your-api.com/hooks/resolved'
    }
  }
}
```

## Complete Example

```typescript
import { Linktor } from '@linktor/sdk'

const client = new Linktor({ apiKey: 'YOUR_API_KEY' })

await client.bots.update('bot_123', {
  escalation: {
    triggers: {
      keywords: {
        enabled: true,
        terms: ['human', 'agent', 'manager', 'complaint']
      },
      sentiment: {
        enabled: true,
        threshold: -0.6,
        consecutiveNegative: 3
      },
      repetition: {
        enabled: true,
        maxRetries: 3,
        maxFallbacks: 2
      }
    },

    action: {
      type: 'inbox',
      inboxId: 'inbox_support',
      priority: 'normal'
    },

    messages: {
      preEscalation: "I'll connect you with a specialist.",
      inQueue: "You're in queue. An agent will be with you shortly.",
      agentAssigned: "{{agent.name}} has joined the conversation."
    },

    handoff: {
      includeSummary: true,
      includeFullTranscript: true,
      includeCustomerProfile: true
    },

    agentTakeover: {
      botBehavior: 'assist',
      returnToBot: {
        enabled: true,
        trigger: 'agent_close'
      }
    }
  }
})
```

## Next Steps

- [Testing Bots](/bots/testing) - Test escalation flows
- [Knowledge Base](/knowledge-base/overview) - Reduce escalations with better answers
- [Flows](/flows/overview) - Build custom escalation flows
