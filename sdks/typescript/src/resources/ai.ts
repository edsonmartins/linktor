/**
 * AI Resource (Agents, Completions, Embeddings)
 */

import type { HttpClient } from '../utils/http';
import type { PaginatedResponse } from '../types/common';
import type {
  Agent,
  CreateAgentInput,
  UpdateAgentInput,
  ListAgentsParams,
  InvokeAgentInput,
  InvokeAgentResponse,
  CompletionRequest,
  CompletionResponse,
  CompletionChunk,
  EmbeddingRequest,
  EmbeddingResponse,
} from '../types/ai';

export class AIResource {
  public readonly agents: AgentsSubResource;
  public readonly completions: CompletionsSubResource;
  public readonly embeddings: EmbeddingsSubResource;

  constructor(private http: HttpClient) {
    this.agents = new AgentsSubResource(http);
    this.completions = new CompletionsSubResource(http);
    this.embeddings = new EmbeddingsSubResource(http);
  }
}

class AgentsSubResource {
  constructor(private http: HttpClient) {}

  /**
   * List agents
   */
  async list(params?: ListAgentsParams): Promise<PaginatedResponse<Agent>> {
    return this.http.get<PaginatedResponse<Agent>>('/ai/agents', { params });
  }

  /**
   * Get a single agent
   */
  async get(id: string): Promise<Agent> {
    return this.http.get<Agent>(`/ai/agents/${id}`);
  }

  /**
   * Create a new agent
   */
  async create(data: CreateAgentInput): Promise<Agent> {
    return this.http.post<Agent>('/ai/agents', data);
  }

  /**
   * Update an agent
   */
  async update(id: string, data: UpdateAgentInput): Promise<Agent> {
    return this.http.patch<Agent>(`/ai/agents/${id}`, data);
  }

  /**
   * Delete an agent
   */
  async delete(id: string): Promise<void> {
    await this.http.delete<void>(`/ai/agents/${id}`);
  }

  /**
   * Activate an agent
   */
  async activate(id: string): Promise<Agent> {
    return this.update(id, { status: 'active' });
  }

  /**
   * Deactivate an agent
   */
  async deactivate(id: string): Promise<Agent> {
    return this.update(id, { status: 'inactive' });
  }

  /**
   * Invoke an agent
   */
  async invoke(id: string, input: InvokeAgentInput): Promise<InvokeAgentResponse> {
    return this.http.post<InvokeAgentResponse>(`/ai/agents/${id}/invoke`, input);
  }

  /**
   * Invoke an agent with streaming
   */
  async *invokeStream(
    id: string,
    input: Omit<InvokeAgentInput, 'stream'>
  ): AsyncGenerator<string, void, unknown> {
    const response = await this.http.stream<{ delta: string }>(
      `/ai/agents/${id}/invoke`,
      { ...input, stream: true }
    );

    for await (const chunk of response) {
      if (chunk.delta) {
        yield chunk.delta;
      }
    }
  }

  /**
   * Assign knowledge bases to agent
   */
  async assignKnowledgeBases(id: string, knowledgeBaseIds: string[]): Promise<Agent> {
    return this.http.post<Agent>(`/ai/agents/${id}/knowledge-bases`, { knowledgeBaseIds });
  }

  /**
   * Remove knowledge bases from agent
   */
  async removeKnowledgeBases(id: string, knowledgeBaseIds: string[]): Promise<Agent> {
    return this.http.delete<Agent>(`/ai/agents/${id}/knowledge-bases`, {
      data: { knowledgeBaseIds },
    });
  }

  /**
   * Duplicate an agent
   */
  async duplicate(id: string, name?: string): Promise<Agent> {
    return this.http.post<Agent>(`/ai/agents/${id}/duplicate`, { name });
  }
}

class CompletionsSubResource {
  constructor(private http: HttpClient) {}

  /**
   * Create a chat completion
   */
  async create(request: CompletionRequest): Promise<CompletionResponse> {
    return this.http.post<CompletionResponse>('/ai/completions', {
      ...request,
      stream: false,
    });
  }

  /**
   * Create a streaming chat completion
   */
  async *createStream(
    request: Omit<CompletionRequest, 'stream'>
  ): AsyncGenerator<CompletionChunk, void, unknown> {
    for await (const chunk of this.http.stream<CompletionChunk>('/ai/completions', {
      ...request,
      stream: true,
    })) {
      yield chunk;
    }
  }

  /**
   * Simple text completion (single message)
   */
  async complete(
    prompt: string,
    options?: Partial<Omit<CompletionRequest, 'messages'>>
  ): Promise<string> {
    const response = await this.create({
      ...options,
      messages: [{ role: 'user', content: prompt }],
    });
    return response.message.content;
  }

  /**
   * Simple streaming text completion
   */
  async *completeStream(
    prompt: string,
    options?: Partial<Omit<CompletionRequest, 'messages' | 'stream'>>
  ): AsyncGenerator<string, void, unknown> {
    for await (const chunk of this.createStream({
      ...options,
      messages: [{ role: 'user', content: prompt }],
    })) {
      if (chunk.delta.content) {
        yield chunk.delta.content;
      }
    }
  }
}

class EmbeddingsSubResource {
  constructor(private http: HttpClient) {}

  /**
   * Create embeddings
   */
  async create(request: EmbeddingRequest): Promise<EmbeddingResponse> {
    return this.http.post<EmbeddingResponse>('/ai/embeddings', request);
  }

  /**
   * Create embedding for a single text
   */
  async embed(text: string, model?: string): Promise<number[]> {
    const response = await this.create({ input: text, model });
    return response.data[0].embedding;
  }

  /**
   * Create embeddings for multiple texts
   */
  async embedBatch(texts: string[], model?: string): Promise<number[][]> {
    const response = await this.create({ input: texts, model });
    return response.data.map((d) => d.embedding);
  }
}
