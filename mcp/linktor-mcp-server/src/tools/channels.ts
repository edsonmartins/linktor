// ============================================
// Channel Tools
// ============================================

import type { Tool } from '@modelcontextprotocol/sdk/types.js';
import type { LinktorClient } from '../api/client.js';
import type { ChannelType, ChannelStatus } from '../api/types.js';

export const channelToolDefinitions: Tool[] = [
  {
    name: 'list_channels',
    description: 'List all configured communication channels (WhatsApp, Telegram, SMS, etc.) with their status.',
    inputSchema: {
      type: 'object',
      properties: {
        type: {
          type: 'string',
          enum: ['webchat', 'whatsapp', 'whatsapp_official', 'telegram', 'sms', 'rcs', 'instagram', 'facebook', 'email', 'voice'],
          description: 'Filter by channel type',
        },
        status: {
          type: 'string',
          enum: ['inactive', 'active', 'error', 'disconnected'],
          description: 'Filter by channel status',
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
    name: 'get_channel',
    description: 'Get detailed information about a specific channel, including configuration and status.',
    inputSchema: {
      type: 'object',
      properties: {
        channel_id: {
          type: 'string',
          description: 'The channel ID',
        },
      },
      required: ['channel_id'],
    },
  },
  {
    name: 'connect_channel',
    description: 'Connect/activate a channel to start receiving messages.',
    inputSchema: {
      type: 'object',
      properties: {
        channel_id: {
          type: 'string',
          description: 'The channel ID to connect',
        },
      },
      required: ['channel_id'],
    },
  },
  {
    name: 'disconnect_channel',
    description: 'Disconnect a channel. Messages will no longer be received.',
    inputSchema: {
      type: 'object',
      properties: {
        channel_id: {
          type: 'string',
          description: 'The channel ID to disconnect',
        },
      },
      required: ['channel_id'],
    },
  },
];

export function registerChannelTools(
  handlers: Map<string, (args: Record<string, unknown>) => Promise<unknown>>,
  client: LinktorClient
): void {
  handlers.set('list_channels', async (args) => {
    return client.channels.list({
      type: args.type as ChannelType | undefined,
      status: args.status as ChannelStatus | undefined,
      limit: args.limit as number | undefined,
      offset: args.offset as number | undefined,
    });
  });

  handlers.set('get_channel', async (args) => {
    return client.channels.get(args.channel_id as string);
  });

  handlers.set('connect_channel', async (args) => {
    return client.channels.connect(args.channel_id as string);
  });

  handlers.set('disconnect_channel', async (args) => {
    return client.channels.disconnect(args.channel_id as string);
  });
}
