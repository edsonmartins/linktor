---
sidebar_position: 1
title: Flows Overview
---

# Flows Overview

Flows are visual conversation workflows that let you design structured interactions without writing code. Build complex conversation logic with drag-and-drop simplicity.

## What are Flows?

A **flow** in Linktor is a visual representation of a conversation path. Flows allow you to:

- **Guide conversations**: Lead customers through structured interactions
- **Collect information**: Gather data through sequential questions
- **Make decisions**: Branch conversations based on user input
- **Trigger actions**: Execute integrations, API calls, and automations
- **Combine with AI**: Use flows as scaffolding with AI-powered responses

## Why Use Flows?

| Use Case | Benefit |
|----------|---------|
| **Lead Qualification** | Collect and score leads consistently |
| **Appointment Booking** | Guide customers through scheduling |
| **Order Status** | Look up and display order information |
| **Surveys & Feedback** | Gather structured customer feedback |
| **FAQ Navigation** | Help customers find answers |
| **Onboarding** | Welcome and orient new users |

## Flow Builder

### Visual Editor

The flow builder provides a canvas-based interface:

```
┌─────────────────────────────────────────────────────────────────┐
│  Flow: Customer Onboarding                        [Save] [Test] │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│   ┌─────────┐     ┌─────────┐     ┌─────────┐                   │
│   │ Welcome │────►│ Question│────►│Condition│                   │
│   │ Message │     │  Name   │     │  Check  │                   │
│   └─────────┘     └─────────┘     └────┬────┘                   │
│                                        │                        │
│                            ┌───────────┼───────────┐            │
│                            ▼           ▼           ▼            │
│                      ┌─────────┐ ┌─────────┐ ┌─────────┐        │
│                      │ Premium │ │ Standard│ │  Free   │        │
│                      │  Path   │ │  Path   │ │  Path   │        │
│                      └─────────┘ └─────────┘ └─────────┘        │
│                                                                 │
├─────────────────────────────────────────────────────────────────┤
│  Node Palette  │  Node Properties  │  Flow Settings            │
└─────────────────────────────────────────────────────────────────┘
```

### Key Features

- **Drag-and-drop nodes**: Add nodes from the palette to the canvas
- **Connect with edges**: Draw connections between nodes
- **Configure properties**: Click nodes to edit their settings
- **Real-time preview**: Test flows without deploying
- **Version history**: Track changes and rollback if needed
- **Collaboration**: Multiple team members can edit (with locking)

## Creating Your First Flow

### Step 1: Create a New Flow

1. Navigate to **Flows** in the sidebar
2. Click **Create Flow**
3. Enter a name and description
4. Choose a template or start from scratch

### Step 2: Add a Trigger

Every flow needs a trigger to start:

```
[Welcome Trigger] ──► [First Node]
```

Common triggers:
- **Welcome**: Activates when a new conversation starts
- **Keyword**: Activates when customer says specific words
- **Intent**: Activates when AI detects specific intent
- **Manual**: Triggered via API or by agents

### Step 3: Build the Conversation

Add nodes to create your conversation flow:

```
[Welcome] ──► [Ask Name] ──► [Ask Email] ──► [Confirm] ──► [End]
```

### Step 4: Add Conditions

Branch based on customer responses:

```
                            ┌── [High Value Path]
[Ask Budget] ──► [Check] ──┼── [Medium Value Path]
                            └── [Low Value Path]
```

### Step 5: Test and Deploy

1. Use the **Test** button to simulate conversations
2. Review the flow for edge cases
3. Click **Publish** to deploy

## Flow Example: Lead Qualification

```typescript
{
  "id": "flow_lead_qual",
  "name": "Lead Qualification",
  "trigger": {
    "type": "keyword",
    "keywords": ["pricing", "demo", "buy"]
  },
  "nodes": [
    {
      "id": "welcome",
      "type": "message",
      "content": "Hi! I'd be happy to help you learn more about our product. Let me ask a few questions.",
      "next": "ask_company"
    },
    {
      "id": "ask_company",
      "type": "question",
      "content": "What's your company name?",
      "variable": "company_name",
      "next": "ask_size"
    },
    {
      "id": "ask_size",
      "type": "question",
      "content": "How many employees does {{company_name}} have?",
      "variable": "company_size",
      "options": ["1-10", "11-50", "51-200", "201-1000", "1000+"],
      "next": "check_size"
    },
    {
      "id": "check_size",
      "type": "condition",
      "conditions": [
        { "if": "company_size in ['201-1000', '1000+']", "then": "enterprise_path" },
        { "if": "company_size in ['51-200']", "then": "mid_market_path" },
        { "else": "smb_path" }
      ]
    },
    {
      "id": "enterprise_path",
      "type": "action",
      "action": "assign_to_sales",
      "data": { "tier": "enterprise", "priority": "high" },
      "next": "book_demo"
    },
    {
      "id": "book_demo",
      "type": "message",
      "content": "Great! Let me connect you with our enterprise team. They'll reach out within 1 hour.",
      "next": "end"
    }
  ]
}
```

## Flow + AI Hybrid

Combine structured flows with AI intelligence:

```
[Flow Start] ──► [Collect Info] ──► [AI Response] ──► [Flow Continue]
```

### AI Fallback

When flow can't handle a query:

```typescript
{
  "id": "ai_fallback",
  "type": "ai_response",
  "botId": "bot_support",
  "fallbackMessage": "Let me find the answer for you...",
  "returnToFlow": true,
  "returnNode": "after_ai"
}
```

### AI within Flow

Use AI for specific responses:

```typescript
{
  "id": "personalized_recommendation",
  "type": "ai_response",
  "botId": "bot_recommender",
  "prompt": "Based on the customer's preferences ({{preferences}}), recommend 3 products.",
  "next": "show_products"
}
```

## Variables and Context

### Setting Variables

Store data throughout the flow:

```typescript
{
  "id": "ask_name",
  "type": "question",
  "content": "What's your name?",
  "variable": "customer_name"  // Stores response in {{customer_name}}
}
```

### Using Variables

Reference variables in messages:

```typescript
{
  "id": "greeting",
  "type": "message",
  "content": "Nice to meet you, {{customer_name}}! How can I help?"
}
```

### System Variables

Built-in variables available in all flows:

| Variable | Description |
|----------|-------------|
| `{{contact.name}}` | Customer's name |
| `{{contact.email}}` | Customer's email |
| `{{contact.phone}}` | Customer's phone |
| `{{channel.type}}` | Channel type (whatsapp, etc.) |
| `{{conversation.id}}` | Current conversation ID |
| `{{flow.name}}` | Current flow name |
| `{{timestamp}}` | Current timestamp |

## Flow Management

### Versioning

```typescript
// Create a new version
await client.flows.createVersion('flow_123', {
  comment: 'Added enterprise path'
})

// List versions
const versions = await client.flows.listVersions('flow_123')

// Rollback to previous version
await client.flows.rollback('flow_123', 'version_abc')
```

### Analytics

Track flow performance:

```typescript
const analytics = await client.flows.getAnalytics('flow_123', {
  dateRange: { from: '2024-01-01', to: '2024-01-31' }
})

console.log(analytics.completionRate)     // % who finished the flow
console.log(analytics.dropOffPoints)      // Where users abandon
console.log(analytics.averageDuration)    // Time to complete
console.log(analytics.conversionRate)     // Goal completion rate
```

### A/B Testing

Test different flow versions:

```typescript
await client.flows.createABTest({
  flowId: 'flow_123',
  variants: [
    { versionId: 'v1', weight: 50 },
    { versionId: 'v2', weight: 50 }
  ],
  metric: 'conversion_rate',
  duration: 7 * 24 * 60 * 60 * 1000  // 7 days
})
```

## Best Practices

1. **Keep flows focused**: One flow per goal or use case

2. **Handle all paths**: Ensure every branch has an endpoint

3. **Use meaningful names**: Name nodes descriptively for easy maintenance

4. **Test edge cases**: What if the user types unexpected input?

5. **Combine with AI wisely**: Use flows for structure, AI for flexibility

6. **Monitor and iterate**: Review analytics and improve drop-off points

7. **Document complex flows**: Add comments for team understanding

## Next Steps

- [Node Types](/flows/node-types) - Learn about all available nodes
- [Triggers](/flows/triggers) - Configure flow triggers
- [Bots Overview](/bots/overview) - Combine flows with AI bots
