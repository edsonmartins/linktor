// ============================================
// Resource Handlers
// ============================================

import type { Resource, TextResourceContents } from '@modelcontextprotocol/sdk/types.js';
import type { LinktorClient } from '../api/client.js';

export const resourceDefinitions: Resource[] = [
  {
    uri: 'linktor://conversations',
    name: 'Active Conversations',
    description: 'List of all active conversations in the system',
    mimeType: 'application/json',
  },
  {
    uri: 'linktor://contacts',
    name: 'Contacts',
    description: 'List of all contacts',
    mimeType: 'application/json',
  },
  {
    uri: 'linktor://channels',
    name: 'Channels',
    description: 'List of configured communication channels',
    mimeType: 'application/json',
  },
  {
    uri: 'linktor://bots',
    name: 'Bots',
    description: 'List of AI bots',
    mimeType: 'application/json',
  },
  {
    uri: 'linktor://analytics/summary',
    name: 'Analytics Summary',
    description: 'Current analytics summary (last 30 days)',
    mimeType: 'application/json',
  },
  {
    uri: 'linktor://users',
    name: 'Users',
    description: 'List of team members/agents',
    mimeType: 'application/json',
  },
];

// Dynamic resource templates (for parameterized resources)
export const resourceTemplates = [
  {
    uriTemplate: 'linktor://conversations/{id}',
    name: 'Conversation Details',
    description: 'Detailed information about a specific conversation',
    mimeType: 'application/json',
  },
  {
    uriTemplate: 'linktor://contacts/{id}',
    name: 'Contact Details',
    description: 'Detailed information about a specific contact',
    mimeType: 'application/json',
  },
  {
    uriTemplate: 'linktor://channels/{id}',
    name: 'Channel Details',
    description: 'Detailed information about a specific channel',
    mimeType: 'application/json',
  },
  {
    uriTemplate: 'linktor://bots/{id}',
    name: 'Bot Details',
    description: 'Detailed information about a specific bot',
    mimeType: 'application/json',
  },
];

export async function handleResourceRead(
  uri: string,
  client: LinktorClient
): Promise<TextResourceContents[]> {
  const url = new URL(uri);
  const path = url.pathname.replace(/^\/\//, '');
  const segments = path.split('/').filter(Boolean);

  // Handle static resources
  if (segments.length === 1) {
    switch (segments[0]) {
      case 'conversations': {
        const data = await client.conversations.list({ status: 'open', limit: 50 });
        return [
          {
            uri,
            mimeType: 'application/json',
            text: JSON.stringify(data, null, 2),
          },
        ];
      }

      case 'contacts': {
        const data = await client.contacts.list({ limit: 50 });
        return [
          {
            uri,
            mimeType: 'application/json',
            text: JSON.stringify(data, null, 2),
          },
        ];
      }

      case 'channels': {
        const data = await client.channels.list({ limit: 50 });
        return [
          {
            uri,
            mimeType: 'application/json',
            text: JSON.stringify(data, null, 2),
          },
        ];
      }

      case 'bots': {
        const data = await client.bots.list({ limit: 50 });
        return [
          {
            uri,
            mimeType: 'application/json',
            text: JSON.stringify(data, null, 2),
          },
        ];
      }

      case 'users': {
        const data = await client.users.list({ limit: 50 });
        return [
          {
            uri,
            mimeType: 'application/json',
            text: JSON.stringify(data, null, 2),
          },
        ];
      }

      default:
        throw new Error(`Unknown resource: ${uri}`);
    }
  }

  // Handle analytics/summary
  if (segments[0] === 'analytics' && segments[1] === 'summary') {
    const now = new Date();
    const thirtyDaysAgo = new Date(now.getTime() - 30 * 24 * 60 * 60 * 1000);
    const data = await client.analytics.getSummary({
      start_date: thirtyDaysAgo.toISOString().split('T')[0],
      end_date: now.toISOString().split('T')[0],
    });
    return [
      {
        uri,
        mimeType: 'application/json',
        text: JSON.stringify(data, null, 2),
      },
    ];
  }

  // Handle parameterized resources (e.g., linktor://conversations/{id})
  if (segments.length === 2) {
    const [resourceType, id] = segments;

    switch (resourceType) {
      case 'conversations': {
        const data = await client.conversations.get(id);
        return [
          {
            uri,
            mimeType: 'application/json',
            text: JSON.stringify(data, null, 2),
          },
        ];
      }

      case 'contacts': {
        const data = await client.contacts.get(id);
        return [
          {
            uri,
            mimeType: 'application/json',
            text: JSON.stringify(data, null, 2),
          },
        ];
      }

      case 'channels': {
        const data = await client.channels.get(id);
        return [
          {
            uri,
            mimeType: 'application/json',
            text: JSON.stringify(data, null, 2),
          },
        ];
      }

      case 'bots': {
        const data = await client.bots.get(id);
        return [
          {
            uri,
            mimeType: 'application/json',
            text: JSON.stringify(data, null, 2),
          },
        ];
      }

      case 'users': {
        const data = await client.users.get(id);
        return [
          {
            uri,
            mimeType: 'application/json',
            text: JSON.stringify(data, null, 2),
          },
        ];
      }

      default:
        throw new Error(`Unknown resource: ${uri}`);
    }
  }

  throw new Error(`Invalid resource URI: ${uri}`);
}
