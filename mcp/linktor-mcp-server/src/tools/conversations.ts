// ============================================
// Conversation Tools
// ============================================

import type { Tool } from '@modelcontextprotocol/sdk/types.js';
import type { LinktorClient } from '../api/client.js';
import type { ConversationStatus, CreateConversationInput } from '../api/types.js';

export const conversationToolDefinitions: Tool[] = [
  {
    name: 'list_conversations',
    description: 'List conversations with optional filters. Returns paginated results with conversation details including contact, channel, and assigned user information.',
    inputSchema: {
      type: 'object',
      properties: {
        status: {
          type: 'string',
          enum: ['open', 'pending', 'resolved', 'closed'],
          description: 'Filter by conversation status',
        },
        channel_id: {
          type: 'string',
          description: 'Filter by channel ID',
        },
        assigned_to: {
          type: 'string',
          description: 'Filter by assigned user ID',
        },
        contact_id: {
          type: 'string',
          description: 'Filter by contact ID',
        },
        limit: {
          type: 'number',
          description: 'Maximum number of results (default: 20)',
          default: 20,
        },
        offset: {
          type: 'number',
          description: 'Number of results to skip (default: 0)',
          default: 0,
        },
      },
    },
  },
  {
    name: 'get_conversation',
    description: 'Get detailed information about a specific conversation, including messages, contact info, and metadata.',
    inputSchema: {
      type: 'object',
      properties: {
        conversation_id: {
          type: 'string',
          description: 'The conversation ID',
        },
      },
      required: ['conversation_id'],
    },
  },
  {
    name: 'create_conversation',
    description: 'Create a new conversation with a contact through a specific channel.',
    inputSchema: {
      type: 'object',
      properties: {
        contact_id: {
          type: 'string',
          description: 'The contact ID to start a conversation with',
        },
        channel_id: {
          type: 'string',
          description: 'The channel ID to use for the conversation',
        },
        subject: {
          type: 'string',
          description: 'Optional subject for the conversation',
        },
        tags: {
          type: 'array',
          items: { type: 'string' },
          description: 'Optional tags for the conversation',
        },
        metadata: {
          type: 'object',
          additionalProperties: { type: 'string' },
          description: 'Optional metadata key-value pairs',
        },
      },
      required: ['contact_id', 'channel_id'],
    },
  },
  {
    name: 'assign_conversation',
    description: 'Assign a conversation to a specific user/agent.',
    inputSchema: {
      type: 'object',
      properties: {
        conversation_id: {
          type: 'string',
          description: 'The conversation ID',
        },
        user_id: {
          type: 'string',
          description: 'The user ID to assign the conversation to',
        },
      },
      required: ['conversation_id', 'user_id'],
    },
  },
  {
    name: 'unassign_conversation',
    description: 'Remove assignment from a conversation, making it available for other agents.',
    inputSchema: {
      type: 'object',
      properties: {
        conversation_id: {
          type: 'string',
          description: 'The conversation ID',
        },
      },
      required: ['conversation_id'],
    },
  },
  {
    name: 'resolve_conversation',
    description: 'Mark a conversation as resolved.',
    inputSchema: {
      type: 'object',
      properties: {
        conversation_id: {
          type: 'string',
          description: 'The conversation ID',
        },
      },
      required: ['conversation_id'],
    },
  },
  {
    name: 'reopen_conversation',
    description: 'Reopen a previously resolved or closed conversation.',
    inputSchema: {
      type: 'object',
      properties: {
        conversation_id: {
          type: 'string',
          description: 'The conversation ID',
        },
      },
      required: ['conversation_id'],
    },
  },
  {
    name: 'close_conversation',
    description: 'Close a conversation permanently.',
    inputSchema: {
      type: 'object',
      properties: {
        conversation_id: {
          type: 'string',
          description: 'The conversation ID',
        },
      },
      required: ['conversation_id'],
    },
  },
];

export function registerConversationTools(
  handlers: Map<string, (args: Record<string, unknown>) => Promise<unknown>>,
  client: LinktorClient
): void {
  handlers.set('list_conversations', async (args) => {
    return client.conversations.list({
      status: args.status as ConversationStatus | undefined,
      channel_id: args.channel_id as string | undefined,
      assigned_to: args.assigned_to as string | undefined,
      contact_id: args.contact_id as string | undefined,
      limit: args.limit as number | undefined,
      offset: args.offset as number | undefined,
    });
  });

  handlers.set('get_conversation', async (args) => {
    return client.conversations.get(args.conversation_id as string);
  });

  handlers.set('create_conversation', async (args) => {
    const input: CreateConversationInput = {
      contact_id: args.contact_id as string,
      channel_id: args.channel_id as string,
      subject: args.subject as string | undefined,
      tags: args.tags as string[] | undefined,
      metadata: args.metadata as Record<string, string> | undefined,
    };
    return client.conversations.create(input);
  });

  handlers.set('assign_conversation', async (args) => {
    return client.conversations.assign(
      args.conversation_id as string,
      args.user_id as string
    );
  });

  handlers.set('unassign_conversation', async (args) => {
    return client.conversations.unassign(args.conversation_id as string);
  });

  handlers.set('resolve_conversation', async (args) => {
    return client.conversations.resolve(args.conversation_id as string);
  });

  handlers.set('reopen_conversation', async (args) => {
    return client.conversations.reopen(args.conversation_id as string);
  });

  handlers.set('close_conversation', async (args) => {
    return client.conversations.close(args.conversation_id as string);
  });
}
