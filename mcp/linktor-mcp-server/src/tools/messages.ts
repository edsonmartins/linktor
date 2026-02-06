// ============================================
// Message Tools
// ============================================

import type { Tool } from '@modelcontextprotocol/sdk/types.js';
import type { LinktorClient } from '../api/client.js';
import type { ContentType, SendMessageInput } from '../api/types.js';

export const messageToolDefinitions: Tool[] = [
  {
    name: 'list_messages',
    description: 'List messages in a conversation. Messages are returned in chronological order with pagination support.',
    inputSchema: {
      type: 'object',
      properties: {
        conversation_id: {
          type: 'string',
          description: 'The conversation ID',
        },
        limit: {
          type: 'number',
          description: 'Maximum number of messages to return (default: 50)',
          default: 50,
        },
        before: {
          type: 'string',
          description: 'Return messages before this message ID (for pagination)',
        },
        after: {
          type: 'string',
          description: 'Return messages after this message ID (for pagination)',
        },
      },
      required: ['conversation_id'],
    },
  },
  {
    name: 'get_message',
    description: 'Get detailed information about a specific message.',
    inputSchema: {
      type: 'object',
      properties: {
        conversation_id: {
          type: 'string',
          description: 'The conversation ID',
        },
        message_id: {
          type: 'string',
          description: 'The message ID',
        },
      },
      required: ['conversation_id', 'message_id'],
    },
  },
  {
    name: 'send_message',
    description: 'Send a message in a conversation. Supports text, images, documents, and other content types.',
    inputSchema: {
      type: 'object',
      properties: {
        conversation_id: {
          type: 'string',
          description: 'The conversation ID',
        },
        content: {
          type: 'string',
          description: 'The message content (text for text messages, URL for media)',
        },
        content_type: {
          type: 'string',
          enum: ['text', 'image', 'video', 'audio', 'document', 'location', 'contact', 'template', 'interactive'],
          description: 'The type of content (default: text)',
          default: 'text',
        },
        attachments: {
          type: 'array',
          items: {
            type: 'object',
            properties: {
              url: { type: 'string', description: 'URL of the attachment' },
              type: { type: 'string', description: 'MIME type of the attachment' },
            },
            required: ['url', 'type'],
          },
          description: 'Optional file attachments',
        },
        metadata: {
          type: 'object',
          additionalProperties: { type: 'string' },
          description: 'Optional metadata key-value pairs',
        },
      },
      required: ['conversation_id', 'content'],
    },
  },
];

export function registerMessageTools(
  handlers: Map<string, (args: Record<string, unknown>) => Promise<unknown>>,
  client: LinktorClient
): void {
  handlers.set('list_messages', async (args) => {
    return client.messages.list(args.conversation_id as string, {
      limit: args.limit as number | undefined,
      before: args.before as string | undefined,
      after: args.after as string | undefined,
    });
  });

  handlers.set('get_message', async (args) => {
    return client.messages.get(
      args.conversation_id as string,
      args.message_id as string
    );
  });

  handlers.set('send_message', async (args) => {
    const input: SendMessageInput = {
      content: args.content as string,
      content_type: args.content_type as ContentType | undefined,
      attachments: args.attachments as { url: string; type: string }[] | undefined,
      metadata: args.metadata as Record<string, string> | undefined,
    };
    return client.messages.send(args.conversation_id as string, input);
  });
}
