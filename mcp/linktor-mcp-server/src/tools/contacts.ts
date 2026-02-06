// ============================================
// Contact Tools
// ============================================

import type { Tool } from '@modelcontextprotocol/sdk/types.js';
import type { LinktorClient } from '../api/client.js';
import type { CreateContactInput, UpdateContactInput } from '../api/types.js';

export const contactToolDefinitions: Tool[] = [
  {
    name: 'list_contacts',
    description: 'List contacts with optional search and filters. Returns paginated results.',
    inputSchema: {
      type: 'object',
      properties: {
        search: {
          type: 'string',
          description: 'Search by name, email, or phone',
        },
        tags: {
          type: 'array',
          items: { type: 'string' },
          description: 'Filter by tags',
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
    name: 'get_contact',
    description: 'Get detailed information about a specific contact, including their identities across different channels.',
    inputSchema: {
      type: 'object',
      properties: {
        contact_id: {
          type: 'string',
          description: 'The contact ID',
        },
      },
      required: ['contact_id'],
    },
  },
  {
    name: 'create_contact',
    description: 'Create a new contact with basic information.',
    inputSchema: {
      type: 'object',
      properties: {
        name: {
          type: 'string',
          description: 'Contact name',
        },
        email: {
          type: 'string',
          description: 'Contact email address',
        },
        phone: {
          type: 'string',
          description: 'Contact phone number',
        },
        tags: {
          type: 'array',
          items: { type: 'string' },
          description: 'Tags for categorizing the contact',
        },
        custom_fields: {
          type: 'object',
          additionalProperties: { type: 'string' },
          description: 'Custom fields as key-value pairs',
        },
      },
      required: ['name'],
    },
  },
  {
    name: 'update_contact',
    description: 'Update an existing contact\'s information.',
    inputSchema: {
      type: 'object',
      properties: {
        contact_id: {
          type: 'string',
          description: 'The contact ID',
        },
        name: {
          type: 'string',
          description: 'Updated contact name',
        },
        email: {
          type: 'string',
          description: 'Updated email address',
        },
        phone: {
          type: 'string',
          description: 'Updated phone number',
        },
        tags: {
          type: 'array',
          items: { type: 'string' },
          description: 'Updated tags',
        },
        custom_fields: {
          type: 'object',
          additionalProperties: { type: 'string' },
          description: 'Updated custom fields',
        },
      },
      required: ['contact_id'],
    },
  },
  {
    name: 'delete_contact',
    description: 'Delete a contact. This action cannot be undone.',
    inputSchema: {
      type: 'object',
      properties: {
        contact_id: {
          type: 'string',
          description: 'The contact ID to delete',
        },
      },
      required: ['contact_id'],
    },
  },
];

export function registerContactTools(
  handlers: Map<string, (args: Record<string, unknown>) => Promise<unknown>>,
  client: LinktorClient
): void {
  handlers.set('list_contacts', async (args) => {
    return client.contacts.list({
      search: args.search as string | undefined,
      tags: args.tags as string[] | undefined,
      limit: args.limit as number | undefined,
      offset: args.offset as number | undefined,
    });
  });

  handlers.set('get_contact', async (args) => {
    return client.contacts.get(args.contact_id as string);
  });

  handlers.set('create_contact', async (args) => {
    const input: CreateContactInput = {
      name: args.name as string,
      email: args.email as string | undefined,
      phone: args.phone as string | undefined,
      tags: args.tags as string[] | undefined,
      custom_fields: args.custom_fields as Record<string, string> | undefined,
    };
    return client.contacts.create(input);
  });

  handlers.set('update_contact', async (args) => {
    const input: UpdateContactInput = {
      name: args.name as string | undefined,
      email: args.email as string | undefined,
      phone: args.phone as string | undefined,
      tags: args.tags as string[] | undefined,
      custom_fields: args.custom_fields as Record<string, string> | undefined,
    };
    return client.contacts.update(args.contact_id as string, input);
  });

  handlers.set('delete_contact', async (args) => {
    await client.contacts.delete(args.contact_id as string);
    return { success: true, message: 'Contact deleted successfully' };
  });
}
