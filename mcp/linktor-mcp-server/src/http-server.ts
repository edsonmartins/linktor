// ============================================
// Linktor MCP Server - HTTP Transport
// ============================================

import express, { Request, Response } from 'express';
import cors from 'cors';
import { randomUUID } from 'crypto';
import { Server } from '@modelcontextprotocol/sdk/server/index.js';
import {
  CallToolRequestSchema,
  ListToolsRequestSchema,
  ListResourcesRequestSchema,
  ReadResourceRequestSchema,
  ListPromptsRequestSchema,
  GetPromptRequestSchema,
  InitializeRequestSchema,
  ErrorCode,
  McpError,
} from '@modelcontextprotocol/sdk/types.js';

import { LinktorClient, LinktorClientError } from './api/client.js';
import { conversationToolDefinitions, registerConversationTools } from './tools/conversations.js';
import { messageToolDefinitions, registerMessageTools } from './tools/messages.js';
import { contactToolDefinitions, registerContactTools } from './tools/contacts.js';
import { channelToolDefinitions, registerChannelTools } from './tools/channels.js';
import { botToolDefinitions, registerBotTools } from './tools/bots.js';
import { analyticsToolDefinitions, registerAnalyticsTools } from './tools/analytics.js';
import { knowledgeToolDefinitions, registerKnowledgeTools } from './tools/knowledge.js';
import { resourceDefinitions, handleResourceRead } from './resources/handlers.js';
import { promptDefinitions, handlePromptGet } from './prompts/templates.js';

// ─── Session Management ───────────────────────────────────────────
interface Session {
  id: string;
  client: LinktorClient;
  toolHandlers: Map<string, (args: Record<string, unknown>) => Promise<unknown>>;
  initialized: boolean;
  createdAt: Date;
}

const sessions = new Map<string, Session>();

// Clean up old sessions (older than 30 minutes)
setInterval(() => {
  const now = Date.now();
  const maxAge = 30 * 60 * 1000; // 30 minutes

  for (const [id, session] of sessions.entries()) {
    if (now - session.createdAt.getTime() > maxAge) {
      sessions.delete(id);
    }
  }
}, 5 * 60 * 1000); // Check every 5 minutes

function createSession(): Session {
  const id = randomUUID();
  const client = new LinktorClient();
  const toolHandlers = new Map<string, (args: Record<string, unknown>) => Promise<unknown>>();

  // Register all tool handlers
  registerConversationTools(toolHandlers, client);
  registerMessageTools(toolHandlers, client);
  registerContactTools(toolHandlers, client);
  registerChannelTools(toolHandlers, client);
  registerBotTools(toolHandlers, client);
  registerAnalyticsTools(toolHandlers, client);
  registerKnowledgeTools(toolHandlers, client);

  const session: Session = {
    id,
    client,
    toolHandlers,
    initialized: false,
    createdAt: new Date(),
  };

  sessions.set(id, session);
  return session;
}

// ─── Tool Definitions ─────────────────────────────────────────────
const allTools = [
  ...conversationToolDefinitions,
  ...messageToolDefinitions,
  ...contactToolDefinitions,
  ...channelToolDefinitions,
  ...botToolDefinitions,
  ...analyticsToolDefinitions,
  ...knowledgeToolDefinitions,
];

// ─── JSON-RPC Types ───────────────────────────────────────────────
interface JsonRpcRequest {
  jsonrpc: '2.0';
  id?: string | number;
  method: string;
  params?: Record<string, unknown>;
}

interface JsonRpcResponse {
  jsonrpc: '2.0';
  id?: string | number;
  result?: unknown;
  error?: {
    code: number;
    message: string;
    data?: unknown;
  };
}

// ─── Request Handler ──────────────────────────────────────────────
async function handleMcpRequest(
  session: Session,
  request: JsonRpcRequest
): Promise<JsonRpcResponse | null> {
  const { id, method, params = {} } = request;

  // Handle notifications (no id = notification, no response needed)
  if (id === undefined) {
    if (method === 'notifications/initialized') {
      session.initialized = true;
    }
    return null;
  }

  try {
    switch (method) {
      case 'initialize': {
        return {
          jsonrpc: '2.0',
          id,
          result: {
            protocolVersion: '2025-03-26',
            capabilities: {
              tools: {},
              resources: {},
              prompts: {},
            },
            serverInfo: {
              name: 'linktor-mcp-server',
              version: '1.0.0',
            },
          },
        };
      }

      case 'tools/list': {
        return {
          jsonrpc: '2.0',
          id,
          result: { tools: allTools },
        };
      }

      case 'tools/call': {
        const { name, arguments: args } = params as { name: string; arguments?: Record<string, unknown> };
        const handler = session.toolHandlers.get(name);

        if (!handler) {
          return {
            jsonrpc: '2.0',
            id,
            error: {
              code: -32601,
              message: `Unknown tool: ${name}`,
            },
          };
        }

        try {
          const result = await handler(args || {});
          return {
            jsonrpc: '2.0',
            id,
            result: {
              content: [
                {
                  type: 'text',
                  text: JSON.stringify(result, null, 2),
                },
              ],
            },
          };
        } catch (error) {
          if (error instanceof LinktorClientError) {
            return {
              jsonrpc: '2.0',
              id,
              result: {
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
              },
            };
          }
          throw error;
        }
      }

      case 'resources/list': {
        return {
          jsonrpc: '2.0',
          id,
          result: { resources: resourceDefinitions },
        };
      }

      case 'resources/read': {
        const { uri } = params as { uri: string };
        try {
          const contents = await handleResourceRead(uri, session.client);
          return {
            jsonrpc: '2.0',
            id,
            result: { contents },
          };
        } catch (error) {
          if (error instanceof LinktorClientError) {
            return {
              jsonrpc: '2.0',
              id,
              error: {
                code: -32603,
                message: error.message,
              },
            };
          }
          throw error;
        }
      }

      case 'prompts/list': {
        return {
          jsonrpc: '2.0',
          id,
          result: { prompts: promptDefinitions },
        };
      }

      case 'prompts/get': {
        const { name, arguments: args } = params as { name: string; arguments?: Record<string, string> };
        const result = handlePromptGet(name, args || {});

        if (!result) {
          return {
            jsonrpc: '2.0',
            id,
            error: {
              code: -32602,
              message: `Unknown prompt: ${name}`,
            },
          };
        }

        return {
          jsonrpc: '2.0',
          id,
          result,
        };
      }

      default: {
        return {
          jsonrpc: '2.0',
          id,
          error: {
            code: -32601,
            message: `Method not found: ${method}`,
          },
        };
      }
    }
  } catch (error) {
    return {
      jsonrpc: '2.0',
      id,
      error: {
        code: -32603,
        message: error instanceof Error ? error.message : 'Internal error',
      },
    };
  }
}

// ─── Express App ──────────────────────────────────────────────────
export function createHttpServer() {
  const app = express();

  // CORS configuration
  app.use(cors({
    origin: true,
    credentials: true,
    exposedHeaders: ['Mcp-Session-Id'],
  }));

  app.use(express.json());

  // Health check
  app.get('/health', (_req, res) => {
    res.json({ status: 'ok', service: 'linktor-mcp-server' });
  });

  // MCP endpoint
  app.post('/mcp', async (req: Request, res: Response) => {
    try {
      const sessionId = req.headers['mcp-session-id'] as string | undefined;

      // Get or create session
      let session: Session;
      if (sessionId && sessions.has(sessionId)) {
        session = sessions.get(sessionId)!;
      } else {
        session = createSession();
      }

      // Set session ID header
      res.setHeader('Mcp-Session-Id', session.id);

      // Handle request
      const request = req.body as JsonRpcRequest;
      const response = await handleMcpRequest(session, request);

      // Notifications don't get a response
      if (response === null) {
        res.status(204).send();
        return;
      }

      res.json(response);
    } catch (error) {
      console.error('MCP request error:', error);
      res.status(500).json({
        jsonrpc: '2.0',
        error: {
          code: -32603,
          message: 'Internal server error',
        },
      });
    }
  });

  return app;
}

// ─── Main ─────────────────────────────────────────────────────────
export function startHttpServer(port: number = 3001): void {
  const app = createHttpServer();

  app.listen(port, () => {
    console.log(`🔌 Linktor MCP HTTP server running on http://localhost:${port}`);
    console.log(`   Endpoint: POST http://localhost:${port}/mcp`);
    console.log(`   Health:   GET  http://localhost:${port}/health`);
    console.log('');
    console.log('Available capabilities:');
    console.log(`   Tools:     ${allTools.length}`);
    console.log(`   Resources: ${resourceDefinitions.length}`);
    console.log(`   Prompts:   ${promptDefinitions.length}`);
  });
}

// Run if called directly
const isMain = import.meta.url === `file://${process.argv[1]}`;
if (isMain) {
  const port = parseInt(process.env.MCP_HTTP_PORT || '3001', 10);
  startHttpServer(port);
}
