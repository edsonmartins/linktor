// ============================================
// Linktor MCP Server
// ============================================

import { Server } from '@modelcontextprotocol/sdk/server/index.js';
import { StdioServerTransport } from '@modelcontextprotocol/sdk/server/stdio.js';
import {
  CallToolRequestSchema,
  ListToolsRequestSchema,
  ListResourcesRequestSchema,
  ReadResourceRequestSchema,
  ListPromptsRequestSchema,
  GetPromptRequestSchema,
  ErrorCode,
  McpError,
} from '@modelcontextprotocol/sdk/types.js';

import { LinktorClient, LinktorClientError } from './api/client.js';
import { registerConversationTools, conversationToolDefinitions } from './tools/conversations.js';
import { registerMessageTools, messageToolDefinitions } from './tools/messages.js';
import { registerContactTools, contactToolDefinitions } from './tools/contacts.js';
import { registerChannelTools, channelToolDefinitions } from './tools/channels.js';
import { registerBotTools, botToolDefinitions } from './tools/bots.js';
import { registerAnalyticsTools, analyticsToolDefinitions } from './tools/analytics.js';
import { registerKnowledgeTools, knowledgeToolDefinitions } from './tools/knowledge.js';
import { resourceDefinitions, handleResourceRead } from './resources/handlers.js';
import { promptDefinitions, handlePromptGet } from './prompts/templates.js';

export function createServer(client: LinktorClient): Server {
  const server = new Server(
    {
      name: 'linktor-mcp-server',
      version: '1.0.0',
    },
    {
      capabilities: {
        tools: {},
        resources: {},
        prompts: {},
      },
    }
  );

  // Collect all tool definitions
  const allTools = [
    ...conversationToolDefinitions,
    ...messageToolDefinitions,
    ...contactToolDefinitions,
    ...channelToolDefinitions,
    ...botToolDefinitions,
    ...analyticsToolDefinitions,
    ...knowledgeToolDefinitions,
  ];

  // Register tool handlers
  const toolHandlers = new Map<string, (args: Record<string, unknown>) => Promise<unknown>>();

  registerConversationTools(toolHandlers, client);
  registerMessageTools(toolHandlers, client);
  registerContactTools(toolHandlers, client);
  registerChannelTools(toolHandlers, client);
  registerBotTools(toolHandlers, client);
  registerAnalyticsTools(toolHandlers, client);
  registerKnowledgeTools(toolHandlers, client);

  // List Tools Handler
  server.setRequestHandler(ListToolsRequestSchema, async () => {
    return { tools: allTools };
  });

  // Call Tool Handler
  server.setRequestHandler(CallToolRequestSchema, async (request) => {
    const { name, arguments: args } = request.params;

    const handler = toolHandlers.get(name);
    if (!handler) {
      throw new McpError(ErrorCode.MethodNotFound, `Unknown tool: ${name}`);
    }

    try {
      const result = await handler(args || {});
      return {
        content: [
          {
            type: 'text',
            text: JSON.stringify(result, null, 2),
          },
        ],
      };
    } catch (error) {
      if (error instanceof LinktorClientError) {
        return {
          content: [
            {
              type: 'text',
              text: JSON.stringify(
                {
                  error: true,
                  code: error.code,
                  message: error.message,
                  details: error.details,
                },
                null,
                2
              ),
            },
          ],
          isError: true,
        };
      }
      throw error;
    }
  });

  // List Resources Handler
  server.setRequestHandler(ListResourcesRequestSchema, async () => {
    return { resources: resourceDefinitions };
  });

  // Read Resource Handler
  server.setRequestHandler(ReadResourceRequestSchema, async (request) => {
    const { uri } = request.params;

    try {
      const contents = await handleResourceRead(uri, client);
      return { contents };
    } catch (error) {
      if (error instanceof LinktorClientError) {
        throw new McpError(ErrorCode.InternalError, error.message);
      }
      throw error;
    }
  });

  // List Prompts Handler
  server.setRequestHandler(ListPromptsRequestSchema, async () => {
    return { prompts: promptDefinitions };
  });

  // Get Prompt Handler
  server.setRequestHandler(GetPromptRequestSchema, async (request) => {
    const { name, arguments: args } = request.params;

    const result = handlePromptGet(name, args || {});
    if (!result) {
      throw new McpError(ErrorCode.InvalidParams, `Unknown prompt: ${name}`);
    }

    return result;
  });

  return server;
}

export async function runServer(): Promise<void> {
  const client = new LinktorClient();
  const server = createServer(client);
  const transport = new StdioServerTransport();

  await server.connect(transport);

  // Handle graceful shutdown
  process.on('SIGINT', async () => {
    await server.close();
    process.exit(0);
  });

  process.on('SIGTERM', async () => {
    await server.close();
    process.exit(0);
  });
}
