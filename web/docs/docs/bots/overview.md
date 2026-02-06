---
sidebar_position: 1
title: Bots Overview
---

# Bots Overview

Bots are the intelligent conversational agents that power automated interactions in Linktor. They combine AI language models with your business knowledge to provide natural, helpful responses to your customers.

## What are Bots?

A **bot** in Linktor is an AI-powered conversational agent that can:

- **Understand natural language**: Process customer messages and understand intent
- **Generate contextual responses**: Provide relevant, helpful answers based on conversation history
- **Access knowledge bases**: Reference your documentation, FAQs, and business data
- **Execute actions**: Trigger workflows, update records, or integrate with external systems
- **Escalate when needed**: Hand off to human agents for complex issues

## How Bots Work

### Architecture

```
┌─────────────────┐     ┌─────────────────┐     ┌─────────────────┐
│   Customer      │     │    Linktor      │     │   AI Provider   │
│   Message       │────►│    Bot Engine   │────►│   (OpenAI, etc) │
└─────────────────┘     └────────┬────────┘     └────────┬────────┘
                                 │                       │
                        ┌────────▼────────┐              │
                        │  Knowledge Base │◄─────────────┘
                        │  (RAG Context)  │
                        └─────────────────┘
```

### Processing Flow

1. **Message Received**: Customer sends a message through any connected channel
2. **Context Gathering**: Bot retrieves conversation history and relevant knowledge base content
3. **Prompt Construction**: System prompt, context, and customer message are combined
4. **AI Processing**: The configured AI provider generates a response
5. **Response Delivery**: The response is sent back through the original channel

## AI Provider Integrations

Linktor supports multiple AI providers, allowing you to choose the best model for your needs:

### OpenAI

```typescript
{
  provider: 'openai',
  model: 'gpt-4-turbo',
  apiKey: process.env.OPENAI_API_KEY
}
```

Supported models:
- `gpt-4-turbo` - Best for complex reasoning and nuanced responses
- `gpt-4` - High-quality responses with strong capabilities
- `gpt-3.5-turbo` - Fast and cost-effective for simpler tasks

### Anthropic

```typescript
{
  provider: 'anthropic',
  model: 'claude-3-opus',
  apiKey: process.env.ANTHROPIC_API_KEY
}
```

Supported models:
- `claude-3-opus` - Highest capability for complex tasks
- `claude-3-sonnet` - Balanced performance and speed
- `claude-3-haiku` - Fastest response times

### Azure OpenAI

```typescript
{
  provider: 'azure',
  deploymentId: 'your-deployment',
  apiKey: process.env.AZURE_OPENAI_KEY,
  endpoint: 'https://your-resource.openai.azure.com'
}
```

### Self-Hosted Models

Linktor also supports self-hosted models via OpenAI-compatible APIs:

```typescript
{
  provider: 'custom',
  baseUrl: 'http://localhost:11434/v1',
  model: 'llama3',
  apiKey: 'optional'
}
```

Compatible with:
- Ollama
- vLLM
- LocalAI
- LM Studio

## Creating a Bot

### Via Dashboard

1. Navigate to **Bots** in the sidebar
2. Click **Create Bot**
3. Enter a name and description
4. Select an AI provider and model
5. Write your system prompt
6. Configure behavior settings
7. Assign channels or flows

### Via API

```typescript
import { Linktor } from '@linktor/sdk'

const client = new Linktor({ apiKey: 'YOUR_API_KEY' })

const bot = await client.bots.create({
  name: 'Support Assistant',
  description: 'Handles customer support inquiries',
  provider: {
    type: 'openai',
    model: 'gpt-4-turbo',
    apiKey: process.env.OPENAI_API_KEY
  },
  systemPrompt: `You are a helpful customer support assistant for Acme Corp.
Your goal is to help customers with their questions about our products and services.
Be friendly, professional, and concise in your responses.`,
  settings: {
    temperature: 0.7,
    maxTokens: 1024
  }
})

// Assign to a channel
await client.channels.update('ch_abc123', {
  botId: bot.id
})
```

## Bot Types

### Conversational Bots

General-purpose bots that handle open-ended conversations:

- Customer support
- Sales inquiries
- General Q&A

### Task-Oriented Bots

Bots designed to complete specific tasks:

- Order status lookups
- Appointment scheduling
- Lead qualification

### Hybrid Bots

Combine flow-based automation with AI intelligence:

- Start with structured flows
- Fall back to AI for unstructured queries
- Escalate to humans when needed

## Knowledge Integration

Bots become more powerful when connected to your knowledge base:

```typescript
await client.bots.update(bot.id, {
  knowledgeBaseIds: ['kb_docs123', 'kb_faq456']
})
```

The bot will automatically:
1. Search knowledge bases for relevant content when processing messages
2. Include relevant context in the AI prompt
3. Cite sources in responses (if configured)

## Monitoring and Analytics

Track bot performance in the dashboard:

| Metric | Description |
|--------|-------------|
| **Response Rate** | Percentage of messages the bot responded to |
| **Resolution Rate** | Conversations resolved without human intervention |
| **Avg. Response Time** | Time to generate and send responses |
| **Escalation Rate** | Percentage of conversations escalated to humans |
| **Satisfaction Score** | Customer ratings and feedback |

## Best Practices

1. **Write clear system prompts**: Be specific about the bot's personality, knowledge boundaries, and response style

2. **Set appropriate temperature**: Lower values (0.1-0.3) for factual responses, higher values (0.7-0.9) for creative conversations

3. **Configure fallback behavior**: Define what happens when the bot cannot answer

4. **Test thoroughly**: Use the built-in testing tools before deploying

5. **Monitor and iterate**: Review conversation logs and continuously improve

## Next Steps

- [Bot Configuration](/bots/configuration) - Detailed configuration options
- [Escalation Rules](/bots/escalation) - Set up human handoff
- [Testing Bots](/bots/testing) - Test before deployment
- [Knowledge Base](/knowledge-base/overview) - Enhance bot responses with your data
