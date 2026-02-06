// ============================================
// Knowledge Base Tools
// ============================================

import type { Tool } from '@modelcontextprotocol/sdk/types.js';
import type { LinktorClient } from '../api/client.js';
import type { SearchKnowledgeParams } from '../api/types.js';

export const knowledgeToolDefinitions: Tool[] = [
  {
    name: 'search_knowledge',
    description: 'Search the knowledge base using semantic search. Returns relevant documents and their content.',
    inputSchema: {
      type: 'object',
      properties: {
        knowledge_base_id: {
          type: 'string',
          description: 'The knowledge base ID to search in',
        },
        query: {
          type: 'string',
          description: 'The search query',
        },
        limit: {
          type: 'number',
          description: 'Maximum number of results (default: 5)',
          default: 5,
        },
        threshold: {
          type: 'number',
          description: 'Minimum relevance score threshold (0-1)',
          default: 0.7,
        },
      },
      required: ['knowledge_base_id', 'query'],
    },
  },
  {
    name: 'list_knowledge_documents',
    description: 'List all documents in a knowledge base.',
    inputSchema: {
      type: 'object',
      properties: {
        knowledge_base_id: {
          type: 'string',
          description: 'The knowledge base ID',
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
      required: ['knowledge_base_id'],
    },
  },
  {
    name: 'get_knowledge_document',
    description: 'Get a specific document from the knowledge base.',
    inputSchema: {
      type: 'object',
      properties: {
        knowledge_base_id: {
          type: 'string',
          description: 'The knowledge base ID',
        },
        document_id: {
          type: 'string',
          description: 'The document ID',
        },
      },
      required: ['knowledge_base_id', 'document_id'],
    },
  },
];

export function registerKnowledgeTools(
  handlers: Map<string, (args: Record<string, unknown>) => Promise<unknown>>,
  client: LinktorClient
): void {
  handlers.set('search_knowledge', async (args) => {
    const params: SearchKnowledgeParams = {
      query: args.query as string,
      limit: args.limit as number | undefined,
      threshold: args.threshold as number | undefined,
    };
    return client.knowledge.search(args.knowledge_base_id as string, params);
  });

  handlers.set('list_knowledge_documents', async (args) => {
    return client.knowledge.listDocuments(args.knowledge_base_id as string, {
      limit: args.limit as number | undefined,
      offset: args.offset as number | undefined,
    });
  });

  handlers.set('get_knowledge_document', async (args) => {
    return client.knowledge.getDocument(
      args.knowledge_base_id as string,
      args.document_id as string
    );
  });
}
