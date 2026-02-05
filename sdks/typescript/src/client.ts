/**
 * Linktor Client - Main SDK entry point
 */

import { HttpClient, HttpClientConfig } from './utils/http';
import { AuthResource } from './resources/auth';
import { ConversationsResource } from './resources/conversations';
import { ContactsResource } from './resources/contacts';
import { ChannelsResource } from './resources/channels';
import { BotsResource } from './resources/bots';
import { AIResource } from './resources/ai';
import { KnowledgeBasesResource } from './resources/knowledge-bases';
import { FlowsResource } from './resources/flows';
import { AnalyticsResource } from './resources/analytics';
import { LinktorWebSocket, WebSocketConfig } from './websocket/client';
import * as webhookUtils from './utils/webhook';

export interface LinktorClientConfig {
  /**
   * Base URL of the Linktor API
   * @default 'https://api.linktor.io'
   */
  baseUrl?: string;

  /**
   * API Key for authentication (preferred for server-side)
   */
  apiKey?: string;

  /**
   * Access token for authentication (for user sessions)
   */
  accessToken?: string;

  /**
   * Request timeout in milliseconds
   * @default 30000
   */
  timeout?: number;

  /**
   * Maximum number of retries for failed requests
   * @default 3
   */
  maxRetries?: number;

  /**
   * Delay between retries in milliseconds
   * @default 1000
   */
  retryDelay?: number;

  /**
   * Custom headers to include in all requests
   */
  headers?: Record<string, string>;

  /**
   * Callback to refresh the access token when it expires
   */
  onTokenRefresh?: () => Promise<string>;

  /**
   * WebSocket configuration
   */
  websocket?: Partial<Omit<WebSocketConfig, 'apiKey' | 'accessToken'>>;
}

export class LinktorClient {
  private http: HttpClient;
  private wsClient: LinktorWebSocket | null = null;
  private config: LinktorClientConfig;

  /**
   * Authentication resource
   */
  public readonly auth: AuthResource;

  /**
   * Conversations resource
   */
  public readonly conversations: ConversationsResource;

  /**
   * Contacts resource
   */
  public readonly contacts: ContactsResource;

  /**
   * Channels resource
   */
  public readonly channels: ChannelsResource;

  /**
   * Bots resource
   */
  public readonly bots: BotsResource;

  /**
   * AI resource (agents, completions, embeddings)
   */
  public readonly ai: AIResource;

  /**
   * Knowledge Bases resource
   */
  public readonly knowledgeBases: KnowledgeBasesResource;

  /**
   * Flows resource
   */
  public readonly flows: FlowsResource;

  /**
   * Analytics resource
   */
  public readonly analytics: AnalyticsResource;

  /**
   * Webhook utilities
   */
  public readonly webhooks = {
    /**
     * Verify webhook signature
     */
    verify: webhookUtils.verifyWebhookSignature,

    /**
     * Verify webhook with timestamp validation
     */
    verifyWithTimestamp: webhookUtils.verifyWebhook,

    /**
     * Compute HMAC-SHA256 signature
     */
    computeSignature: webhookUtils.computeSignature,

    /**
     * Parse and validate webhook event
     */
    constructEvent: webhookUtils.constructEvent,

    /**
     * Create webhook handler for Express-like frameworks
     */
    createHandler: webhookUtils.createWebhookHandler,

    /**
     * Check event type
     */
    isEventType: webhookUtils.isEventType,
  };

  constructor(config: LinktorClientConfig = {}) {
    this.config = {
      baseUrl: 'https://api.linktor.io',
      timeout: 30000,
      maxRetries: 3,
      retryDelay: 1000,
      ...config,
    };

    const httpConfig: HttpClientConfig = {
      baseUrl: this.config.baseUrl!,
      apiKey: this.config.apiKey,
      accessToken: this.config.accessToken,
      timeout: this.config.timeout,
      maxRetries: this.config.maxRetries,
      retryDelay: this.config.retryDelay,
      headers: this.config.headers,
      onTokenRefresh: this.config.onTokenRefresh,
    };

    this.http = new HttpClient(httpConfig);

    // Initialize resources
    this.auth = new AuthResource(this.http);
    this.conversations = new ConversationsResource(this.http);
    this.contacts = new ContactsResource(this.http);
    this.channels = new ChannelsResource(this.http);
    this.bots = new BotsResource(this.http);
    this.ai = new AIResource(this.http);
    this.knowledgeBases = new KnowledgeBasesResource(this.http);
    this.flows = new FlowsResource(this.http);
    this.analytics = new AnalyticsResource(this.http);
  }

  /**
   * Get WebSocket client for real-time updates
   */
  get ws(): LinktorWebSocket {
    if (!this.wsClient) {
      const wsUrl = this.config.baseUrl!.replace(/^http/, 'ws') + '/ws';
      this.wsClient = new LinktorWebSocket({
        url: wsUrl,
        apiKey: this.config.apiKey,
        accessToken: this.config.accessToken,
        ...this.config.websocket,
      });
    }
    return this.wsClient;
  }

  /**
   * Update API key
   */
  setApiKey(apiKey: string): void {
    this.config.apiKey = apiKey;
    this.http.setApiKey(apiKey);
    // Recreate WebSocket client if it exists
    if (this.wsClient) {
      this.wsClient.disconnect();
      this.wsClient = null;
    }
  }

  /**
   * Update access token
   */
  setAccessToken(accessToken: string): void {
    this.config.accessToken = accessToken;
    this.http.setAccessToken(accessToken);
    // Recreate WebSocket client if it exists
    if (this.wsClient) {
      this.wsClient.disconnect();
      this.wsClient = null;
    }
  }

  /**
   * Close all connections
   */
  close(): void {
    if (this.wsClient) {
      this.wsClient.disconnect();
      this.wsClient = null;
    }
  }
}

/**
 * Create a new Linktor client
 */
export function createClient(config?: LinktorClientConfig): LinktorClient {
  return new LinktorClient(config);
}
