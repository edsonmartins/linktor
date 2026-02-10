// ============================================
// Linktor API Client
// ============================================

import { config, type Config } from '../config.js';
import type {
  ApiResponse,
  PaginatedResponse,
  // Conversations
  Conversation,
  CreateConversationInput,
  ListConversationsParams,
  // Messages
  Message,
  SendMessageInput,
  ListMessagesParams,
  // Contacts
  Contact,
  CreateContactInput,
  UpdateContactInput,
  PaginationParams,
  // Channels
  Channel,
  ChannelType,
  ChannelStatus,
  // Bots
  Bot,
  BotStatus,
  ListBotsParams,
  TestBotInput,
  TestBotResponse,
  // Knowledge Base
  KnowledgeDocument,
  SearchKnowledgeParams,
  SearchResult,
  // Analytics
  AnalyticsParams,
  AnalyticsSummary,
  ConversationStats,
  // Users
  User,
} from './types.js';

export class LinktorClientError extends Error {
  constructor(
    message: string,
    public readonly code: string,
    public readonly status?: number,
    public readonly details?: Record<string, string>
  ) {
    super(message);
    this.name = 'LinktorClientError';
  }
}

interface RequestOptions {
  method: 'GET' | 'POST' | 'PUT' | 'PATCH' | 'DELETE';
  path: string;
  body?: unknown;
  params?: Record<string, string | number | boolean | undefined>;
}

export class LinktorClient {
  private readonly baseUrl: string;
  private readonly apiKey?: string;
  private readonly accessToken?: string;
  private readonly timeout: number;
  private readonly maxRetries: number;
  private readonly retryDelay: number;

  constructor(options?: Partial<Config>) {
    const cfg = { ...config, ...options };
    this.baseUrl = cfg.apiUrl.replace(/\/$/, '');
    this.apiKey = cfg.apiKey;
    this.accessToken = cfg.accessToken;
    this.timeout = cfg.timeout;
    this.maxRetries = cfg.maxRetries;
    this.retryDelay = cfg.retryDelay;
  }

  private getAuthHeaders(): Record<string, string> {
    if (this.accessToken) {
      return { Authorization: `Bearer ${this.accessToken}` };
    }
    if (this.apiKey) {
      return { 'X-API-Key': this.apiKey };
    }
    return {};
  }

  private buildUrl(path: string, params?: Record<string, string | number | boolean | undefined>): string {
    const url = new URL(`${this.baseUrl}${path}`);
    if (params) {
      Object.entries(params).forEach(([key, value]) => {
        if (value !== undefined) {
          url.searchParams.append(key, String(value));
        }
      });
    }
    return url.toString();
  }

  private async request<T>(options: RequestOptions): Promise<T> {
    const { method, path, body, params } = options;
    const url = this.buildUrl(path, params);

    let lastError: Error | undefined;

    for (let attempt = 0; attempt <= this.maxRetries; attempt++) {
      try {
        const controller = new AbortController();
        const timeoutId = setTimeout(() => controller.abort(), this.timeout);

        const response = await fetch(url, {
          method,
          headers: {
            'Content-Type': 'application/json',
            Accept: 'application/json',
            ...this.getAuthHeaders(),
          },
          body: body ? JSON.stringify(body) : undefined,
          signal: controller.signal,
        });

        clearTimeout(timeoutId);

        const data = await response.json() as ApiResponse<T>;

        if (!response.ok || !data.success) {
          throw new LinktorClientError(
            data.error?.message || `Request failed with status ${response.status}`,
            data.error?.code || 'UNKNOWN_ERROR',
            response.status,
            data.error?.details
          );
        }

        return data.data as T;
      } catch (error) {
        lastError = error as Error;

        // Don't retry on client errors (4xx)
        if (error instanceof LinktorClientError && error.status && error.status < 500) {
          throw error;
        }

        // Don't retry on abort
        if (error instanceof Error && error.name === 'AbortError') {
          throw new LinktorClientError('Request timed out', 'TIMEOUT');
        }

        // Wait before retrying
        if (attempt < this.maxRetries) {
          await new Promise((resolve) => setTimeout(resolve, this.retryDelay * (attempt + 1)));
        }
      }
    }

    throw lastError || new LinktorClientError('Request failed', 'UNKNOWN_ERROR');
  }

  // ============================================
  // Conversations
  // ============================================

  readonly conversations = {
    list: async (params?: ListConversationsParams): Promise<PaginatedResponse<Conversation>> => {
      return this.request({
        method: 'GET',
        path: '/conversations',
        params: params as Record<string, string | number | boolean | undefined>,
      });
    },

    get: async (id: string): Promise<Conversation> => {
      return this.request({
        method: 'GET',
        path: `/conversations/${id}`,
      });
    },

    create: async (input: CreateConversationInput): Promise<Conversation> => {
      return this.request({
        method: 'POST',
        path: '/conversations',
        body: input,
      });
    },

    assign: async (id: string, userId: string): Promise<Conversation> => {
      return this.request({
        method: 'POST',
        path: `/conversations/${id}/assign`,
        body: { user_id: userId },
      });
    },

    unassign: async (id: string): Promise<Conversation> => {
      return this.request({
        method: 'POST',
        path: `/conversations/${id}/unassign`,
      });
    },

    resolve: async (id: string): Promise<Conversation> => {
      return this.request({
        method: 'POST',
        path: `/conversations/${id}/resolve`,
      });
    },

    reopen: async (id: string): Promise<Conversation> => {
      return this.request({
        method: 'POST',
        path: `/conversations/${id}/reopen`,
      });
    },

    close: async (id: string): Promise<Conversation> => {
      return this.request({
        method: 'POST',
        path: `/conversations/${id}/close`,
      });
    },
  };

  // ============================================
  // Messages
  // ============================================

  readonly messages = {
    list: async (conversationId: string, params?: ListMessagesParams): Promise<PaginatedResponse<Message>> => {
      return this.request({
        method: 'GET',
        path: `/conversations/${conversationId}/messages`,
        params: params as Record<string, string | number | boolean | undefined>,
      });
    },

    get: async (conversationId: string, messageId: string): Promise<Message> => {
      return this.request({
        method: 'GET',
        path: `/conversations/${conversationId}/messages/${messageId}`,
      });
    },

    send: async (conversationId: string, input: SendMessageInput): Promise<Message> => {
      return this.request({
        method: 'POST',
        path: `/conversations/${conversationId}/messages`,
        body: input,
      });
    },
  };

  // ============================================
  // Contacts
  // ============================================

  readonly contacts = {
    list: async (params?: PaginationParams & { search?: string; tags?: string[] }): Promise<PaginatedResponse<Contact>> => {
      const queryParams: Record<string, string | number | boolean | undefined> = {
        limit: params?.limit,
        offset: params?.offset,
        search: (params as { search?: string })?.search,
      };
      if ((params as { tags?: string[] })?.tags?.length) {
        queryParams.tags = (params as { tags: string[] }).tags.join(',');
      }
      return this.request({
        method: 'GET',
        path: '/contacts',
        params: queryParams,
      });
    },

    get: async (id: string): Promise<Contact> => {
      return this.request({
        method: 'GET',
        path: `/contacts/${id}`,
      });
    },

    create: async (input: CreateContactInput): Promise<Contact> => {
      return this.request({
        method: 'POST',
        path: '/contacts',
        body: input,
      });
    },

    update: async (id: string, input: UpdateContactInput): Promise<Contact> => {
      return this.request({
        method: 'PUT',
        path: `/contacts/${id}`,
        body: input,
      });
    },

    delete: async (id: string): Promise<void> => {
      return this.request({
        method: 'DELETE',
        path: `/contacts/${id}`,
      });
    },
  };

  // ============================================
  // Channels
  // ============================================

  readonly channels = {
    list: async (params?: { type?: ChannelType; status?: ChannelStatus } & PaginationParams): Promise<PaginatedResponse<Channel>> => {
      return this.request({
        method: 'GET',
        path: '/channels',
        params: params as Record<string, string | number | boolean | undefined>,
      });
    },

    get: async (id: string): Promise<Channel> => {
      return this.request({
        method: 'GET',
        path: `/channels/${id}`,
      });
    },

    connect: async (id: string): Promise<Channel> => {
      return this.request({
        method: 'POST',
        path: `/channels/${id}/connect`,
      });
    },

    disconnect: async (id: string): Promise<Channel> => {
      return this.request({
        method: 'POST',
        path: `/channels/${id}/disconnect`,
      });
    },
  };

  // ============================================
  // Bots
  // ============================================

  readonly bots = {
    list: async (params?: ListBotsParams): Promise<PaginatedResponse<Bot>> => {
      return this.request({
        method: 'GET',
        path: '/bots',
        params: params as Record<string, string | number | boolean | undefined>,
      });
    },

    get: async (id: string): Promise<Bot> => {
      return this.request({
        method: 'GET',
        path: `/bots/${id}`,
      });
    },

    activate: async (id: string): Promise<Bot> => {
      return this.request({
        method: 'POST',
        path: `/bots/${id}/activate`,
      });
    },

    deactivate: async (id: string): Promise<Bot> => {
      return this.request({
        method: 'POST',
        path: `/bots/${id}/deactivate`,
      });
    },

    test: async (id: string, input: TestBotInput): Promise<TestBotResponse> => {
      return this.request({
        method: 'POST',
        path: `/bots/${id}/test`,
        body: input,
      });
    },
  };

  // ============================================
  // Knowledge Base
  // ============================================

  readonly knowledge = {
    listDocuments: async (knowledgeBaseId: string, params?: PaginationParams): Promise<PaginatedResponse<KnowledgeDocument>> => {
      return this.request({
        method: 'GET',
        path: `/knowledge-bases/${knowledgeBaseId}/documents`,
        params: params as Record<string, string | number | boolean | undefined>,
      });
    },

    getDocument: async (knowledgeBaseId: string, documentId: string): Promise<KnowledgeDocument> => {
      return this.request({
        method: 'GET',
        path: `/knowledge-bases/${knowledgeBaseId}/documents/${documentId}`,
      });
    },

    search: async (knowledgeBaseId: string, params: SearchKnowledgeParams): Promise<SearchResult[]> => {
      return this.request({
        method: 'POST',
        path: `/knowledge-bases/${knowledgeBaseId}/search`,
        body: params,
      });
    },
  };

  // ============================================
  // Analytics
  // ============================================

  readonly analytics = {
    getSummary: async (params: AnalyticsParams): Promise<AnalyticsSummary> => {
      const queryParams: Record<string, string | number | boolean | undefined> = {
        start_date: params.start_date,
        end_date: params.end_date,
        group_by: params.group_by,
      };
      if (params.metrics?.length) {
        queryParams.metrics = params.metrics.join(',');
      }
      return this.request({
        method: 'GET',
        path: '/analytics/summary',
        params: queryParams,
      });
    },

    getConversationStats: async (period: 'day' | 'week' | 'month'): Promise<ConversationStats> => {
      return this.request({
        method: 'GET',
        path: '/analytics/conversations',
        params: { period },
      });
    },
  };

  // ============================================
  // VRE (Visual Response Engine)
  // ============================================

  readonly vre = {
    render: async (input: {
      tenant_id: string;
      template_id: string;
      data: Record<string, unknown>;
      channel?: string;
      format?: string;
    }): Promise<{
      image_base64: string;
      caption: string;
      width: number;
      height: number;
      format: string;
      render_time_ms: number;
    }> => {
      return this.request({
        method: 'POST',
        path: '/vre/render',
        body: input,
      });
    },

    renderAndSend: async (input: {
      conversation_id: string;
      template_id: string;
      data: Record<string, unknown>;
      caption?: string;
      follow_up_text?: string;
    }): Promise<{
      message_id: string;
      image_url: string;
      caption: string;
    }> => {
      return this.request({
        method: 'POST',
        path: '/vre/render-and-send',
        body: input,
      });
    },

    listTemplates: async (tenantId?: string): Promise<{
      templates: Array<{
        id: string;
        name: string;
        description: string;
        schema: Record<string, unknown>;
      }>;
    }> => {
      return this.request({
        method: 'GET',
        path: '/vre/templates',
        params: tenantId ? { tenant_id: tenantId } : undefined,
      });
    },

    preview: async (input: {
      template_id: string;
      data?: Record<string, unknown>;
    }): Promise<{
      image_base64: string;
      width: number;
      height: number;
    }> => {
      return this.request({
        method: 'POST',
        path: `/vre/templates/${input.template_id}/preview`,
        body: input.data ? { data: input.data } : undefined,
      });
    },
  };

  // ============================================
  // Users (for reference/lookup)
  // ============================================

  readonly users = {
    list: async (params?: PaginationParams): Promise<PaginatedResponse<User>> => {
      return this.request({
        method: 'GET',
        path: '/users',
        params: params as Record<string, string | number | boolean | undefined>,
      });
    },

    get: async (id: string): Promise<User> => {
      return this.request({
        method: 'GET',
        path: `/users/${id}`,
      });
    },

    me: async (): Promise<User> => {
      return this.request({
        method: 'GET',
        path: '/users/me',
      });
    },
  };
}

// Export a default instance
export const linktorClient = new LinktorClient();
