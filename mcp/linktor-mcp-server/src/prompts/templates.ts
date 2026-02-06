// ============================================
// Prompt Templates
// ============================================

import type { Prompt, GetPromptResult } from '@modelcontextprotocol/sdk/types.js';

export const promptDefinitions: Prompt[] = [
  {
    name: 'customer_support',
    description: 'Customer support assistant prompt for handling conversations',
    arguments: [
      {
        name: 'company_name',
        description: 'The name of the company',
        required: true,
      },
      {
        name: 'tone',
        description: 'The tone of the support (friendly, professional, casual)',
        required: false,
      },
    ],
  },
  {
    name: 'conversation_summary',
    description: 'Summarize a conversation including key points and action items',
    arguments: [
      {
        name: 'conversation_id',
        description: 'The ID of the conversation to summarize',
        required: true,
      },
    ],
  },
  {
    name: 'draft_response',
    description: 'Draft a response to a customer message',
    arguments: [
      {
        name: 'message',
        description: 'The customer message to respond to',
        required: true,
      },
      {
        name: 'context',
        description: 'Additional context about the conversation',
        required: false,
      },
    ],
  },
  {
    name: 'analyze_sentiment',
    description: 'Analyze the sentiment of a conversation or message',
    arguments: [
      {
        name: 'text',
        description: 'The text to analyze',
        required: true,
      },
    ],
  },
];

const promptTemplates: Record<string, (args: Record<string, string>) => GetPromptResult> = {
  customer_support: (args) => ({
    description: 'Customer support assistant prompt',
    messages: [
      {
        role: 'user',
        content: {
          type: 'text',
          text: `You are a customer support assistant for ${args.company_name}. Your role is to help customers with their inquiries in a ${args.tone || 'professional'} manner.

Key guidelines:
1. Be helpful and empathetic
2. Provide accurate information based on the knowledge base
3. Escalate to a human agent when you cannot resolve an issue
4. Always confirm customer satisfaction before closing

Available tools:
- Use list_conversations to see active conversations
- Use send_message to respond to customers
- Use search_knowledge to find relevant information
- Use assign_conversation to escalate to a human agent

Start by reviewing the current conversation context and provide appropriate assistance.`,
        },
      },
    ],
  }),

  conversation_summary: (args) => ({
    description: 'Summarize a conversation',
    messages: [
      {
        role: 'user',
        content: {
          type: 'text',
          text: `Please summarize the conversation with ID: ${args.conversation_id}

Use the following tools to gather information:
1. get_conversation - to get conversation details
2. list_messages - to get all messages in the conversation

Provide a summary that includes:
- Main topic/issue discussed
- Key points raised by the customer
- Actions taken by the support team
- Current status and any pending items
- Suggested next steps

Format the summary in a clear, structured way.`,
        },
      },
    ],
  }),

  draft_response: (args) => ({
    description: 'Draft a response to a customer message',
    messages: [
      {
        role: 'user',
        content: {
          type: 'text',
          text: `Please draft a response to the following customer message:

"${args.message}"

${args.context ? `Additional context: ${args.context}` : ''}

Guidelines for the response:
1. Be polite and professional
2. Address the customer's concern directly
3. Provide clear and actionable information
4. Offer further assistance if needed

You can use search_knowledge to find relevant information to include in the response.

Provide the draft response and explain your reasoning.`,
        },
      },
    ],
  }),

  analyze_sentiment: (args) => ({
    description: 'Analyze the sentiment of text',
    messages: [
      {
        role: 'user',
        content: {
          type: 'text',
          text: `Please analyze the sentiment of the following text:

"${args.text}"

Provide:
1. Overall sentiment (positive, negative, neutral, mixed)
2. Confidence level (high, medium, low)
3. Key emotional indicators
4. Suggestions for response approach based on sentiment
5. Whether escalation to a human agent is recommended

If this is related to a conversation, consider using get_conversation and list_messages for more context.`,
        },
      },
    ],
  }),
};

export function handlePromptGet(
  name: string,
  args: Record<string, string>
): GetPromptResult | null {
  const template = promptTemplates[name];
  if (!template) {
    return null;
  }
  return template(args);
}
