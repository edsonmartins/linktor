---
sidebar_position: 2
title: Bot Configuration
---

# Bot Configuration

This guide covers all configuration options available when creating and managing bots in Linktor.

## Basic Settings

### Name and Description

```typescript
{
  name: 'Support Bot',
  description: 'Handles customer support inquiries for the product team'
}
```

- **Name**: A unique identifier displayed in the dashboard (required)
- **Description**: Internal notes about the bot's purpose (optional)

### Avatar and Branding

```typescript
{
  avatar: 'https://cdn.example.com/bot-avatar.png',
  displayName: 'Acme Support',
  brandColor: '#4F46E5'
}
```

These settings affect how the bot appears in WebChat and other channels that support customization.

## AI Provider Configuration

### Provider Selection

```typescript
{
  provider: {
    type: 'openai',           // 'openai' | 'anthropic' | 'azure' | 'custom'
    model: 'gpt-4-turbo',     // Model identifier
    apiKey: 'sk-...',         // API key (stored encrypted)
    baseUrl: null             // Custom endpoint (for 'custom' type)
  }
}
```

### Azure OpenAI

```typescript
{
  provider: {
    type: 'azure',
    deploymentId: 'gpt-4-deployment',
    apiKey: process.env.AZURE_OPENAI_KEY,
    endpoint: 'https://your-resource.openai.azure.com',
    apiVersion: '2024-02-15-preview'
  }
}
```

### Custom/Self-Hosted Models

```typescript
{
  provider: {
    type: 'custom',
    baseUrl: 'http://localhost:11434/v1',
    model: 'llama3:70b',
    apiKey: 'optional-key'
  }
}
```

## System Prompt

The system prompt defines your bot's personality, knowledge, and behavior guidelines.

### Basic Structure

```typescript
{
  systemPrompt: `You are [BOT_NAME], a [ROLE] for [COMPANY].

## Your Role
- [Primary responsibility]
- [Secondary responsibility]

## Guidelines
- [Behavior guideline 1]
- [Behavior guideline 2]

## Limitations
- [What the bot should NOT do]
- [Topics to avoid]

## Response Style
- [Tone and voice guidelines]
- [Formatting preferences]`
}
```

### Example System Prompt

```typescript
{
  systemPrompt: `You are Alex, a customer support specialist for TechGadgets Inc.

## Your Role
- Help customers with product questions, orders, and technical issues
- Provide accurate information about our products and services
- Guide customers through troubleshooting steps

## Guidelines
- Be friendly, patient, and professional
- Keep responses concise but complete
- Use simple language, avoid technical jargon unless necessary
- If you don't know something, say so and offer to escalate

## Limitations
- Never share customer personal data
- Don't make promises about refunds or exchanges without verification
- Don't discuss competitor products

## Response Style
- Use a warm, conversational tone
- Include relevant links when helpful
- Format longer responses with bullet points for readability`
}
```

### Dynamic Variables

You can use variables in your system prompt that are replaced at runtime:

| Variable | Description |
|----------|-------------|
| `{{customer_name}}` | Customer's name if available |
| `{{current_date}}` | Today's date |
| `{{current_time}}` | Current time in customer's timezone |
| `{{channel_type}}` | The channel being used (whatsapp, telegram, etc.) |
| `{{conversation_id}}` | Unique conversation identifier |

```typescript
{
  systemPrompt: `Today is {{current_date}}.
You are speaking with {{customer_name}} via {{channel_type}}.`
}
```

## Generation Settings

### Temperature

Controls randomness in responses. Range: 0.0 to 2.0

```typescript
{
  settings: {
    temperature: 0.7  // Default: 0.7
  }
}
```

| Value | Use Case |
|-------|----------|
| 0.0 - 0.3 | Factual, consistent responses (FAQ, support) |
| 0.4 - 0.7 | Balanced (general conversation) |
| 0.8 - 1.2 | Creative, varied responses |
| 1.3+ | Highly creative (not recommended for support) |

### Max Tokens

Maximum length of generated responses:

```typescript
{
  settings: {
    maxTokens: 1024  // Default: 1024
  }
}
```

Considerations:
- Shorter limits (256-512) for quick answers
- Longer limits (1024-2048) for detailed explanations
- Consider channel limitations (SMS has character limits)

### Top P (Nucleus Sampling)

Alternative to temperature for controlling diversity:

```typescript
{
  settings: {
    topP: 0.9  // Default: 1.0
  }
}
```

### Frequency Penalty

Reduces repetition in responses:

```typescript
{
  settings: {
    frequencyPenalty: 0.5  // Range: -2.0 to 2.0, Default: 0
  }
}
```

### Presence Penalty

Encourages discussing new topics:

```typescript
{
  settings: {
    presencePenalty: 0.5  // Range: -2.0 to 2.0, Default: 0
  }
}
```

## Context Settings

### Conversation History

How many previous messages to include as context:

```typescript
{
  context: {
    maxMessages: 20,        // Maximum messages to include
    maxTokens: 4000,        // Maximum tokens for context
    includeSystemMessages: true  // Include bot's own messages
  }
}
```

### Knowledge Base Integration

```typescript
{
  context: {
    knowledgeBaseIds: ['kb_123', 'kb_456'],
    maxChunks: 5,           // Maximum knowledge chunks to include
    minRelevanceScore: 0.7, // Minimum similarity threshold
    citeSources: true       // Include source citations
  }
}
```

### Context Window Strategy

```typescript
{
  context: {
    strategy: 'sliding',    // 'sliding' | 'summary' | 'hybrid'
    summaryThreshold: 30    // Messages before summarization (for 'summary' and 'hybrid')
  }
}
```

| Strategy | Description |
|----------|-------------|
| `sliding` | Keep most recent N messages |
| `summary` | Summarize older messages to preserve context |
| `hybrid` | Combine sliding window with summarized history |

## Response Behavior

### Typing Simulation

Make responses feel more natural:

```typescript
{
  behavior: {
    typingIndicator: true,
    typingDelay: {
      min: 500,   // Minimum delay in ms
      max: 2000   // Maximum delay in ms
    }
  }
}
```

### Message Chunking

Split long responses for channels with limits:

```typescript
{
  behavior: {
    chunkMessages: true,
    maxChunkLength: 1600,  // Characters per chunk
    chunkDelay: 1000       // Delay between chunks in ms
  }
}
```

### Fallback Responses

When the bot cannot generate a response:

```typescript
{
  behavior: {
    fallbackMessages: [
      "I'm sorry, I didn't understand that. Could you rephrase?",
      "I'm having trouble understanding. Let me connect you with a human agent."
    ],
    fallbackAction: 'escalate'  // 'retry' | 'escalate' | 'message'
  }
}
```

## Rate Limiting

Protect against abuse and control costs:

```typescript
{
  rateLimits: {
    messagesPerMinute: 10,
    messagesPerHour: 100,
    messagesPerDay: 1000,
    cooldownMessage: "Please wait a moment before sending another message."
  }
}
```

## Working Hours

Configure when the bot is active:

```typescript
{
  schedule: {
    enabled: true,
    timezone: 'America/New_York',
    hours: {
      monday: { start: '09:00', end: '18:00' },
      tuesday: { start: '09:00', end: '18:00' },
      wednesday: { start: '09:00', end: '18:00' },
      thursday: { start: '09:00', end: '18:00' },
      friday: { start: '09:00', end: '17:00' },
      saturday: null,  // Closed
      sunday: null     // Closed
    },
    offlineMessage: "We're currently offline. Leave a message and we'll get back to you.",
    offlineAction: 'collect_info'  // 'message' | 'collect_info' | 'redirect'
  }
}
```

## Language Settings

```typescript
{
  language: {
    primary: 'en',
    supported: ['en', 'es', 'pt', 'fr'],
    autoDetect: true,
    translateResponses: false  // Translate AI responses to customer's language
  }
}
```

## Moderation

### Content Filtering

```typescript
{
  moderation: {
    enabled: true,
    blockProfanity: true,
    blockPII: true,        // Block personal identifiable information
    blockMalicious: true,  // Block potential prompt injection
    customBlocklist: ['competitor-name', 'inappropriate-term']
  }
}
```

### Sentiment Analysis

```typescript
{
  moderation: {
    sentimentAnalysis: true,
    escalateOnNegativeSentiment: true,
    negativeThreshold: -0.5  // Range: -1.0 to 1.0
  }
}
```

## Complete Configuration Example

```typescript
import { Linktor } from '@linktor/sdk'

const client = new Linktor({ apiKey: 'YOUR_API_KEY' })

const bot = await client.bots.create({
  name: 'Support Assistant',
  description: 'Main customer support bot',
  avatar: 'https://cdn.example.com/bot.png',
  displayName: 'Acme Support',

  provider: {
    type: 'openai',
    model: 'gpt-4-turbo',
    apiKey: process.env.OPENAI_API_KEY
  },

  systemPrompt: `You are a helpful customer support assistant for Acme Corp...`,

  settings: {
    temperature: 0.7,
    maxTokens: 1024,
    topP: 1.0,
    frequencyPenalty: 0.3,
    presencePenalty: 0.3
  },

  context: {
    maxMessages: 20,
    maxTokens: 4000,
    knowledgeBaseIds: ['kb_docs', 'kb_faq'],
    maxChunks: 5,
    minRelevanceScore: 0.7,
    citeSources: true,
    strategy: 'hybrid'
  },

  behavior: {
    typingIndicator: true,
    typingDelay: { min: 500, max: 2000 },
    chunkMessages: true,
    maxChunkLength: 1600,
    fallbackAction: 'escalate'
  },

  rateLimits: {
    messagesPerMinute: 10,
    messagesPerHour: 100
  },

  schedule: {
    enabled: true,
    timezone: 'America/New_York',
    hours: {
      monday: { start: '09:00', end: '18:00' },
      // ... other days
    }
  },

  language: {
    primary: 'en',
    supported: ['en', 'es'],
    autoDetect: true
  },

  moderation: {
    enabled: true,
    blockProfanity: true,
    blockPII: true,
    sentimentAnalysis: true
  }
})
```

## Environment Variables

For security, use environment variables for sensitive configuration:

| Variable | Description |
|----------|-------------|
| `OPENAI_API_KEY` | OpenAI API key |
| `ANTHROPIC_API_KEY` | Anthropic API key |
| `AZURE_OPENAI_KEY` | Azure OpenAI key |
| `AZURE_OPENAI_ENDPOINT` | Azure OpenAI endpoint |

## Next Steps

- [Escalation Rules](/bots/escalation) - Configure human handoff
- [Testing Bots](/bots/testing) - Test your configuration
- [Knowledge Base](/knowledge-base/overview) - Enhance bot responses
