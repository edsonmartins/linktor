/**
 * WebSocket Client for real-time communication
 */

import WebSocket from 'ws';
import { WebSocketError } from '../utils/errors';
import type { Message } from '../types/conversation';

export type WebSocketEvent =
  | 'connected'
  | 'disconnected'
  | 'message'
  | 'message_status'
  | 'conversation_update'
  | 'typing'
  | 'error';

export interface WebSocketConfig {
  url: string;
  apiKey?: string;
  accessToken?: string;
  autoReconnect?: boolean;
  reconnectInterval?: number;
  maxReconnectAttempts?: number;
  pingInterval?: number;
}

export interface MessageEvent {
  type: 'message';
  conversationId: string;
  message: Message;
}

export interface MessageStatusEvent {
  type: 'message_status';
  conversationId: string;
  messageId: string;
  status: string;
  timestamp: string;
}

export interface ConversationUpdateEvent {
  type: 'conversation_update';
  conversationId: string;
  status?: string;
  assignedTo?: string;
  metadata?: Record<string, unknown>;
}

export interface TypingEvent {
  type: 'typing';
  conversationId: string;
  userId: string;
  isTyping: boolean;
}

export type WebSocketData =
  | MessageEvent
  | MessageStatusEvent
  | ConversationUpdateEvent
  | TypingEvent;

type EventHandler<T = WebSocketData> = (data: T) => void;

export class LinktorWebSocket {
  private ws: WebSocket | null = null;
  private config: Required<WebSocketConfig>;
  private reconnectAttempts = 0;
  private pingTimer?: NodeJS.Timeout;
  private reconnectTimer?: NodeJS.Timeout;
  private handlers: Map<WebSocketEvent, Set<EventHandler>> = new Map();
  private subscriptions: Set<string> = new Set();
  private isConnecting = false;
  private shouldReconnect = true;

  constructor(config: WebSocketConfig) {
    this.config = {
      autoReconnect: true,
      reconnectInterval: 5000,
      maxReconnectAttempts: 10,
      pingInterval: 30000,
      ...config,
    };
  }

  /**
   * Connect to WebSocket server
   */
  async connect(): Promise<void> {
    if (this.ws?.readyState === WebSocket.OPEN) {
      return;
    }

    if (this.isConnecting) {
      return;
    }

    this.isConnecting = true;
    this.shouldReconnect = true;

    return new Promise((resolve, reject) => {
      const wsUrl = this.buildWsUrl();

      try {
        this.ws = new WebSocket(wsUrl);

        this.ws.onopen = () => {
          this.isConnecting = false;
          this.reconnectAttempts = 0;
          this.startPingInterval();
          this.resubscribe();
          this.emit('connected', undefined);
          resolve();
        };

        this.ws.onclose = (event) => {
          this.isConnecting = false;
          this.stopPingInterval();
          this.emit('disconnected', { code: event.code, reason: event.reason });

          if (this.shouldReconnect && this.config.autoReconnect) {
            this.scheduleReconnect();
          }
        };

        this.ws.onerror = (error) => {
          this.isConnecting = false;
          this.emit('error', error);

          if (this.ws?.readyState !== WebSocket.OPEN) {
            reject(new WebSocketError('WebSocket connection failed'));
          }
        };

        this.ws.onmessage = (event) => {
          this.handleMessage(event.data);
        };
      } catch (error) {
        this.isConnecting = false;
        reject(error);
      }
    });
  }

  /**
   * Disconnect from WebSocket server
   */
  disconnect(): void {
    this.shouldReconnect = false;
    this.stopPingInterval();
    this.clearReconnectTimer();

    if (this.ws) {
      this.ws.close(1000, 'Client disconnect');
      this.ws = null;
    }
  }

  /**
   * Check if connected
   */
  isConnected(): boolean {
    return this.ws?.readyState === WebSocket.OPEN;
  }

  /**
   * Subscribe to a conversation
   */
  subscribe(conversationId: string): void {
    this.subscriptions.add(conversationId);
    this.send({
      type: 'subscribe',
      conversationId,
    });
  }

  /**
   * Unsubscribe from a conversation
   */
  unsubscribe(conversationId: string): void {
    this.subscriptions.delete(conversationId);
    this.send({
      type: 'unsubscribe',
      conversationId,
    });
  }

  /**
   * Send typing indicator
   */
  sendTyping(conversationId: string, isTyping: boolean): void {
    this.send({
      type: 'typing',
      conversationId,
      isTyping,
    });
  }

  /**
   * Register event handler
   */
  on<T extends WebSocketEvent>(
    event: T,
    handler: EventHandler<
      T extends 'message'
        ? MessageEvent
        : T extends 'message_status'
        ? MessageStatusEvent
        : T extends 'conversation_update'
        ? ConversationUpdateEvent
        : T extends 'typing'
        ? TypingEvent
        : unknown
    >
  ): () => void {
    if (!this.handlers.has(event)) {
      this.handlers.set(event, new Set());
    }
    this.handlers.get(event)!.add(handler as EventHandler);

    // Return unsubscribe function
    return () => {
      this.handlers.get(event)?.delete(handler as EventHandler);
    };
  }

  /**
   * Register message handler
   */
  onMessage(handler: (event: MessageEvent) => void): () => void {
    return this.on('message', handler);
  }

  /**
   * Register message status handler
   */
  onMessageStatus(handler: (event: MessageStatusEvent) => void): () => void {
    return this.on('message_status', handler);
  }

  /**
   * Register conversation update handler
   */
  onConversationUpdate(handler: (event: ConversationUpdateEvent) => void): () => void {
    return this.on('conversation_update', handler);
  }

  /**
   * Register typing handler
   */
  onTyping(handler: (event: TypingEvent) => void): () => void {
    return this.on('typing', handler);
  }

  /**
   * Remove event handler
   */
  off(event: WebSocketEvent, handler: EventHandler): void {
    this.handlers.get(event)?.delete(handler);
  }

  /**
   * Remove all handlers for an event
   */
  offAll(event?: WebSocketEvent): void {
    if (event) {
      this.handlers.delete(event);
    } else {
      this.handlers.clear();
    }
  }

  private buildWsUrl(): string {
    const url = new URL(this.config.url);

    if (this.config.apiKey) {
      url.searchParams.set('api_key', this.config.apiKey);
    } else if (this.config.accessToken) {
      url.searchParams.set('token', this.config.accessToken);
    }

    return url.toString();
  }

  private send(data: unknown): void {
    if (this.ws?.readyState === WebSocket.OPEN) {
      this.ws.send(JSON.stringify(data));
    }
  }

  private handleMessage(data: WebSocket.Data): void {
    try {
      const parsed = JSON.parse(data.toString()) as WebSocketData;

      switch (parsed.type) {
        case 'message':
          this.emit('message', parsed);
          break;
        case 'message_status':
          this.emit('message_status', parsed);
          break;
        case 'conversation_update':
          this.emit('conversation_update', parsed);
          break;
        case 'typing':
          this.emit('typing', parsed);
          break;
        default:
          // Unknown message type
          break;
      }
    } catch {
      // Invalid JSON
    }
  }

  private emit<T>(event: WebSocketEvent, data: T): void {
    const handlers = this.handlers.get(event);
    if (handlers) {
      handlers.forEach((handler) => {
        try {
          handler(data as WebSocketData);
        } catch {
          // Handler error
        }
      });
    }
  }

  private startPingInterval(): void {
    this.stopPingInterval();
    this.pingTimer = setInterval(() => {
      this.send({ type: 'ping' });
    }, this.config.pingInterval);
  }

  private stopPingInterval(): void {
    if (this.pingTimer) {
      clearInterval(this.pingTimer);
      this.pingTimer = undefined;
    }
  }

  private scheduleReconnect(): void {
    if (this.reconnectAttempts >= this.config.maxReconnectAttempts) {
      this.emit('error', new WebSocketError('Max reconnection attempts reached'));
      return;
    }

    this.reconnectAttempts++;
    const delay = this.config.reconnectInterval * Math.pow(2, this.reconnectAttempts - 1);

    this.reconnectTimer = setTimeout(() => {
      this.connect().catch(() => {
        // Reconnection failed, will retry
      });
    }, delay);
  }

  private clearReconnectTimer(): void {
    if (this.reconnectTimer) {
      clearTimeout(this.reconnectTimer);
      this.reconnectTimer = undefined;
    }
  }

  private resubscribe(): void {
    for (const conversationId of this.subscriptions) {
      this.send({
        type: 'subscribe',
        conversationId,
      });
    }
  }
}
