// ============================================
// Bot Tools
// ============================================

import type { Tool } from '@modelcontextprotocol/sdk/types.js';
import type { LinktorClient } from '../api/client.js';
import type { BotStatus, TestBotInput } from '../api/types.js';

export const botToolDefinitions: Tool[] = [
  {
    name: 'list_bots',
    description: 'List all AI bots configured in the system with their status and configuration.',
    inputSchema: {
      type: 'object',
      properties: {
        status: {
          type: 'string',
          enum: ['active', 'inactive', 'training'],
          description: 'Filter by bot status',
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
    name: 'get_bot',
    description: 'Get detailed information about a specific bot, including its configuration, system prompt, and escalation rules.',
    inputSchema: {
      type: 'object',
      properties: {
        bot_id: {
          type: 'string',
          description: 'The bot ID',
        },
      },
      required: ['bot_id'],
    },
  },
  {
    name: 'activate_bot',
    description: 'Activate a bot to start handling conversations.',
    inputSchema: {
      type: 'object',
      properties: {
        bot_id: {
          type: 'string',
          description: 'The bot ID to activate',
        },
      },
      required: ['bot_id'],
    },
  },
  {
    name: 'deactivate_bot',
    description: 'Deactivate a bot. It will stop handling new conversations.',
    inputSchema: {
      type: 'object',
      properties: {
        bot_id: {
          type: 'string',
          description: 'The bot ID to deactivate',
        },
      },
      required: ['bot_id'],
    },
  },
  {
    name: 'test_bot',
    description: 'Test a bot with a sample message to see how it would respond.',
    inputSchema: {
      type: 'object',
      properties: {
        bot_id: {
          type: 'string',
          description: 'The bot ID to test',
        },
        message: {
          type: 'string',
          description: 'The test message to send',
        },
        context: {
          type: 'object',
          additionalProperties: { type: 'string' },
          description: 'Optional context variables for the test',
        },
      },
      required: ['bot_id', 'message'],
    },
  },
];

export function registerBotTools(
  handlers: Map<string, (args: Record<string, unknown>) => Promise<unknown>>,
  client: LinktorClient
): void {
  handlers.set('list_bots', async (args) => {
    return client.bots.list({
      status: args.status as BotStatus | undefined,
      limit: args.limit as number | undefined,
      offset: args.offset as number | undefined,
    });
  });

  handlers.set('get_bot', async (args) => {
    return client.bots.get(args.bot_id as string);
  });

  handlers.set('activate_bot', async (args) => {
    return client.bots.activate(args.bot_id as string);
  });

  handlers.set('deactivate_bot', async (args) => {
    return client.bots.deactivate(args.bot_id as string);
  });

  handlers.set('test_bot', async (args) => {
    const input: TestBotInput = {
      message: args.message as string,
      context: args.context as Record<string, string> | undefined,
    };
    return client.bots.test(args.bot_id as string, input);
  });
}
