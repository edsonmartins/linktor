// ============================================
// Message Tools
// ============================================

import type { Tool } from '@modelcontextprotocol/sdk/types.js';
import type { LinktorClient } from '../api/client.js';
import type { ContentType, Conversation, ConversationStatus, Message, SendMessageInput } from '../api/types.js';

type AttachmentInput = { url: string; type: string };

function buildSendMessageInput(args: Record<string, unknown>): SendMessageInput {
  return {
    content: args.content as string,
    content_type: args.content_type as ContentType | undefined,
    attachments: args.attachments as AttachmentInput[] | undefined,
    metadata: args.metadata as Record<string, string> | undefined,
  };
}

function requiredString(args: Record<string, unknown>, key: string): string {
  const value = args[key];
  if (typeof value !== 'string' || value.trim() === '') {
    throw new Error(`${key} is required`);
  }
  return value;
}

function listItems<T>(response: T[] | { data: T[] }): T[] {
  return Array.isArray(response) ? response : response.data;
}

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
  {
    name: 'send_channel_message',
    description: 'Send a message through a channel. Use conversation_id for an existing conversation, or provide channel_id and contact_id to create a conversation first.',
    inputSchema: {
      type: 'object',
      properties: {
        conversation_id: {
          type: 'string',
          description: 'Existing conversation ID. If provided, channel_id and contact_id are not required.',
        },
        channel_id: {
          type: 'string',
          description: 'Channel ID to send through when creating a new conversation.',
        },
        contact_id: {
          type: 'string',
          description: 'Contact ID to message when creating a new conversation.',
        },
        subject: {
          type: 'string',
          description: 'Optional subject when creating a new conversation.',
        },
        tags: {
          type: 'array',
          items: { type: 'string' },
          description: 'Optional tags when creating a new conversation.',
        },
        content: {
          type: 'string',
          description: 'The message content (text for text messages, URL for media).',
        },
        content_type: {
          type: 'string',
          enum: ['text', 'image', 'video', 'audio', 'document', 'location', 'contact', 'template', 'interactive'],
          description: 'The type of content (default: text).',
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
          description: 'Optional file attachments.',
        },
        metadata: {
          type: 'object',
          additionalProperties: { type: 'string' },
          description: 'Optional metadata key-value pairs.',
        },
      },
      required: ['content'],
    },
  },
  {
    name: 'receive_channel_messages',
    description: 'Read recent conversations and messages received through a channel. This polls stored Linktor conversations populated by channel webhooks.',
    inputSchema: {
      type: 'object',
      properties: {
        channel_id: {
          type: 'string',
          description: 'Channel ID to read messages from.',
        },
        status: {
          type: 'string',
          enum: ['open', 'pending', 'resolved', 'closed'],
          description: 'Optional conversation status filter.',
        },
        conversation_limit: {
          type: 'number',
          description: 'Maximum number of conversations to inspect (default: 10).',
          default: 10,
        },
        offset: {
          type: 'number',
          description: 'Number of conversations to skip (default: 0).',
          default: 0,
        },
        messages_per_conversation: {
          type: 'number',
          description: 'Maximum number of messages per conversation (default: 20).',
          default: 20,
        },
        after: {
          type: 'string',
          description: 'Return messages after this message ID, when supported by the API.',
        },
      },
      required: ['channel_id'],
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
    return client.messages.send(args.conversation_id as string, buildSendMessageInput(args));
  });

  handlers.set('send_channel_message', async (args) => {
    let conversationID = args.conversation_id as string | undefined;
    let conversation: Conversation | undefined;

    if (!conversationID) {
      conversation = await client.conversations.create({
        channel_id: requiredString(args, 'channel_id'),
        contact_id: requiredString(args, 'contact_id'),
        subject: args.subject as string | undefined,
        tags: args.tags as string[] | undefined,
        metadata: args.metadata as Record<string, string> | undefined,
      });
      conversationID = conversation.id;
    }

    const message = await client.messages.send(conversationID, buildSendMessageInput(args));
    return {
      conversation_id: conversationID,
      conversation,
      message,
    };
  });

  handlers.set('receive_channel_messages', async (args) => {
    const channelID = requiredString(args, 'channel_id');
    const conversationsResponse = await client.conversations.list({
      channel_id: channelID,
      status: args.status as ConversationStatus | undefined,
      limit: (args.conversation_limit as number | undefined) ?? 10,
      offset: args.offset as number | undefined,
    });
    const conversations = listItems<Conversation>(conversationsResponse);

    const conversationsWithMessages = await Promise.all(
      conversations.map(async (conversation) => {
        const messagesResponse = await client.messages.list(conversation.id, {
          limit: (args.messages_per_conversation as number | undefined) ?? 20,
          after: args.after as string | undefined,
        });
        return {
          conversation,
          messages: listItems<Message>(messagesResponse),
        };
      })
    );

    return {
      channel_id: channelID,
      conversations: conversationsWithMessages,
    };
  });
}
