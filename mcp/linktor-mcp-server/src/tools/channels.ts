// ============================================
// Channel Tools
// ============================================

import type { Tool } from '@modelcontextprotocol/sdk/types.js';
import type { LinktorClient } from '../api/client.js';
import type { ChannelType, ChannelStatus, CreateChannelInput, TestChannelInput, UpdateChannelInput } from '../api/types.js';

const channelTypes = [
  'webchat',
  'whatsapp',
  'whatsapp_official',
  'whatsapp_unofficial',
  'telegram',
  'sms',
  'rcs',
  'instagram',
  'facebook',
  'email',
  'voice',
];

const configProperty = {
  type: 'object',
  additionalProperties: { type: 'string' },
  description: 'Channel configuration key-value pairs.',
};

const credentialsProperty = {
  type: 'object',
  additionalProperties: { type: 'string' },
  description: 'Sensitive channel credentials. The API stores these separately and does not expose them back.',
};

export const channelToolDefinitions: Tool[] = [
  {
    name: 'list_channels',
    description: 'List all configured communication channels (WhatsApp, Telegram, SMS, etc.) with their status.',
    inputSchema: {
      type: 'object',
      properties: {
        type: {
          type: 'string',
          enum: channelTypes,
          description: 'Filter by channel type',
        },
        status: {
          type: 'string',
          enum: ['inactive', 'active', 'connected', 'connecting', 'error', 'disconnected'],
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
    name: 'create_channel',
    description: 'Create a communication channel. Supports webchat, WhatsApp official/unofficial, Telegram, SMS, RCS, Instagram, Facebook, email, and voice.',
    inputSchema: {
      type: 'object',
      properties: {
        type: {
          type: 'string',
          enum: channelTypes,
          description: 'Channel type.',
        },
        name: {
          type: 'string',
          description: 'Display name for the channel.',
        },
        identifier: {
          type: 'string',
          description: 'Optional channel identifier such as phone number, page ID, bot username, or email address.',
        },
        config: configProperty,
        credentials: credentialsProperty,
      },
      required: ['type', 'name'],
    },
  },
  {
    name: 'update_channel',
    description: 'Update channel name, identifier, configuration, or credentials.',
    inputSchema: {
      type: 'object',
      properties: {
        channel_id: {
          type: 'string',
          description: 'The channel ID.',
        },
        name: {
          type: 'string',
          description: 'Updated display name.',
        },
        identifier: {
          type: 'string',
          description: 'Updated channel identifier.',
        },
        config: configProperty,
        credentials: credentialsProperty,
      },
      required: ['channel_id'],
    },
  },
  {
    name: 'delete_channel',
    description: 'Delete a channel by ID.',
    inputSchema: {
      type: 'object',
      properties: {
        channel_id: {
          type: 'string',
          description: 'The channel ID.',
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
  {
    name: 'set_channel_enabled',
    description: 'Enable or disable a channel at system level without changing its connection session.',
    inputSchema: {
      type: 'object',
      properties: {
        channel_id: {
          type: 'string',
          description: 'The channel ID.',
        },
        enabled: {
          type: 'boolean',
          description: 'Whether the channel should be enabled.',
        },
      },
      required: ['channel_id', 'enabled'],
    },
  },
  {
    name: 'set_channel_status',
    description: 'Set channel status to active or inactive for backwards-compatible API flows.',
    inputSchema: {
      type: 'object',
      properties: {
        channel_id: {
          type: 'string',
          description: 'The channel ID.',
        },
        status: {
          type: 'string',
          enum: ['active', 'inactive'],
          description: 'Desired channel status.',
        },
      },
      required: ['channel_id', 'status'],
    },
  },
  {
    name: 'request_whatsapp_pair_code',
    description: 'Request a WhatsApp unofficial pair code for a channel using a phone number.',
    inputSchema: {
      type: 'object',
      properties: {
        channel_id: {
          type: 'string',
          description: 'The WhatsApp unofficial channel ID.',
        },
        phone_number: {
          type: 'string',
          description: 'Phone number to pair.',
        },
      },
      required: ['channel_id', 'phone_number'],
    },
  },
  {
    name: 'test_channel_config',
    description: 'Validate channel credentials/configuration without creating a channel.',
    inputSchema: {
      type: 'object',
      properties: {
        type: {
          type: 'string',
          enum: channelTypes,
          description: 'Channel type to validate.',
        },
        config: configProperty,
        credentials: credentialsProperty,
      },
      required: ['type'],
    },
  },
  {
    name: 'get_whatsapp_coexistence_status',
    description: 'Get WhatsApp official coexistence status for a channel.',
    inputSchema: {
      type: 'object',
      properties: {
        channel_id: {
          type: 'string',
          description: 'The WhatsApp official channel ID.',
        },
      },
      required: ['channel_id'],
    },
  },
  {
    name: 'subscribe_whatsapp_echoes',
    description: 'Subscribe a WhatsApp official channel to message echoes for coexistence mode.',
    inputSchema: {
      type: 'object',
      properties: {
        channel_id: {
          type: 'string',
          description: 'The WhatsApp official channel ID.',
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

  handlers.set('create_channel', async (args) => {
    return client.channels.create({
      type: args.type as ChannelType,
      name: args.name as string,
      identifier: args.identifier as string | undefined,
      config: args.config as Record<string, string> | undefined,
      credentials: args.credentials as Record<string, string> | undefined,
    } satisfies CreateChannelInput);
  });

  handlers.set('update_channel', async (args) => {
    const input: UpdateChannelInput = {
      name: args.name as string | undefined,
      identifier: args.identifier as string | undefined,
      config: args.config as Record<string, string> | undefined,
      credentials: args.credentials as Record<string, string> | undefined,
    };
    return client.channels.update(args.channel_id as string, input);
  });

  handlers.set('delete_channel', async (args) => {
    await client.channels.delete(args.channel_id as string);
    return { deleted: true, channel_id: args.channel_id };
  });

  handlers.set('connect_channel', async (args) => {
    return client.channels.connect(args.channel_id as string);
  });

  handlers.set('disconnect_channel', async (args) => {
    return client.channels.disconnect(args.channel_id as string);
  });

  handlers.set('set_channel_enabled', async (args) => {
    return client.channels.updateEnabled(args.channel_id as string, args.enabled as boolean);
  });

  handlers.set('set_channel_status', async (args) => {
    return client.channels.updateStatus(args.channel_id as string, args.status as 'active' | 'inactive');
  });

  handlers.set('request_whatsapp_pair_code', async (args) => {
    return client.channels.requestPairCode(args.channel_id as string, args.phone_number as string);
  });

  handlers.set('test_channel_config', async (args) => {
    return client.channels.test({
      type: args.type as ChannelType,
      config: args.config as Record<string, string> | undefined,
      credentials: args.credentials as Record<string, string> | undefined,
    } satisfies TestChannelInput);
  });

  handlers.set('get_whatsapp_coexistence_status', async (args) => {
    return client.channels.getCoexistenceStatus(args.channel_id as string);
  });

  handlers.set('subscribe_whatsapp_echoes', async (args) => {
    return client.channels.subscribeEchoes(args.channel_id as string);
  });
}
