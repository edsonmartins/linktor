/**
 * Webhook types
 */

import type { ChannelType, MessageDirection, MessageStatus } from './common';
import type { Message } from './conversation';

export type WebhookEventType =
  | 'message.received'
  | 'message.sent'
  | 'message.delivered'
  | 'message.read'
  | 'message.failed'
  | 'conversation.created'
  | 'conversation.updated'
  | 'conversation.assigned'
  | 'conversation.resolved'
  | 'contact.created'
  | 'contact.updated'
  | 'channel.connected'
  | 'channel.disconnected'
  | 'channel.error';

export interface WebhookEvent<T = unknown> {
  id: string;
  type: WebhookEventType;
  timestamp: string;
  tenantId: string;
  data: T;
}

export interface MessageReceivedEvent {
  message: Message;
  conversationId: string;
  contactId: string;
  channelId: string;
  channelType: ChannelType;
}

export interface MessageStatusEvent {
  messageId: string;
  conversationId: string;
  status: MessageStatus;
  direction: MessageDirection;
  timestamp: string;
  error?: string;
}

export interface ConversationEvent {
  conversationId: string;
  contactId: string;
  channelId: string;
  status: string;
  assignedTo?: string;
  previousStatus?: string;
  previousAssignedTo?: string;
}

export interface ContactEvent {
  contactId: string;
  name: string;
  email?: string;
  phone?: string;
  previousData?: {
    name?: string;
    email?: string;
    phone?: string;
  };
}

export interface ChannelEvent {
  channelId: string;
  channelType: ChannelType;
  status: string;
  error?: string;
}

// Webhook configuration
export interface WebhookConfig {
  id: string;
  tenantId: string;
  url: string;
  secret: string;
  events: WebhookEventType[];
  status: 'active' | 'inactive';
  headers?: Record<string, string>;
  retryPolicy?: RetryPolicy;
  createdAt: string;
  updatedAt: string;
}

export interface RetryPolicy {
  maxRetries: number;
  retryInterval: number; // seconds
  exponentialBackoff: boolean;
}

export interface CreateWebhookInput {
  url: string;
  events: WebhookEventType[];
  headers?: Record<string, string>;
  retryPolicy?: RetryPolicy;
}

export interface UpdateWebhookInput {
  url?: string;
  events?: WebhookEventType[];
  status?: 'active' | 'inactive';
  headers?: Record<string, string>;
  retryPolicy?: RetryPolicy;
}

// Webhook delivery
export interface WebhookDelivery {
  id: string;
  webhookId: string;
  eventId: string;
  eventType: WebhookEventType;
  status: 'pending' | 'success' | 'failed';
  attempts: number;
  lastAttemptAt?: string;
  responseStatus?: number;
  responseBody?: string;
  error?: string;
  createdAt: string;
}
