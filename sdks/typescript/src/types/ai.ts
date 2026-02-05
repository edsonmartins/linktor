/**
 * AI/Agent types
 */

import type { PaginationParams, Timestamps } from './common';

export interface Agent extends Timestamps {
  id: string;
  tenantId: string;
  name: string;
  description?: string;
  status: AgentStatus;
  model: string;
  systemPrompt?: string;
  temperature: number;
  maxTokens: number;
  tools?: AgentTool[];
  knowledgeBaseIds?: string[];
  metadata?: Record<string, unknown>;
}

export type AgentStatus = 'active' | 'inactive' | 'draft';

export interface AgentTool {
  name: string;
  description: string;
  parameters: ToolParameter[];
  handler?: string; // Function/webhook URL
}

export interface ToolParameter {
  name: string;
  type: 'string' | 'number' | 'boolean' | 'array' | 'object';
  description: string;
  required: boolean;
  default?: unknown;
  enum?: string[];
}

// Completion types
export interface CompletionRequest {
  agentId?: string;
  model?: string;
  messages: ChatMessage[];
  temperature?: number;
  maxTokens?: number;
  stream?: boolean;
  tools?: AgentTool[];
  knowledgeBaseIds?: string[];
  conversationId?: string;
}

export interface ChatMessage {
  role: 'system' | 'user' | 'assistant' | 'tool';
  content: string;
  name?: string;
  toolCallId?: string;
  toolCalls?: ToolCall[];
}

export interface ToolCall {
  id: string;
  type: 'function';
  function: {
    name: string;
    arguments: string;
  };
}

export interface CompletionResponse {
  id: string;
  model: string;
  message: ChatMessage;
  usage: TokenUsage;
  finishReason: 'stop' | 'length' | 'tool_calls';
}

export interface CompletionChunk {
  id: string;
  delta: {
    role?: string;
    content?: string;
    toolCalls?: ToolCall[];
  };
  finishReason?: 'stop' | 'length' | 'tool_calls';
}

export interface TokenUsage {
  promptTokens: number;
  completionTokens: number;
  totalTokens: number;
}

// Embedding types
export interface EmbeddingRequest {
  input: string | string[];
  model?: string;
}

export interface EmbeddingResponse {
  data: Embedding[];
  model: string;
  usage: {
    promptTokens: number;
    totalTokens: number;
  };
}

export interface Embedding {
  index: number;
  embedding: number[];
}

// Request types
export interface CreateAgentInput {
  name: string;
  description?: string;
  model: string;
  systemPrompt?: string;
  temperature?: number;
  maxTokens?: number;
  tools?: AgentTool[];
  knowledgeBaseIds?: string[];
  metadata?: Record<string, unknown>;
}

export interface UpdateAgentInput {
  name?: string;
  description?: string;
  status?: AgentStatus;
  model?: string;
  systemPrompt?: string;
  temperature?: number;
  maxTokens?: number;
  tools?: AgentTool[];
  knowledgeBaseIds?: string[];
  metadata?: Record<string, unknown>;
}

export interface ListAgentsParams extends PaginationParams {
  status?: AgentStatus;
  search?: string;
}

export interface InvokeAgentInput {
  message: string;
  conversationId?: string;
  context?: Record<string, unknown>;
  stream?: boolean;
}

export interface InvokeAgentResponse {
  response: string;
  conversationId: string;
  toolResults?: ToolResult[];
  usage: TokenUsage;
}

export interface ToolResult {
  toolCallId: string;
  name: string;
  result: unknown;
}
