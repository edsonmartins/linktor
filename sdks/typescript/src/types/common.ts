/**
 * Common types used across the SDK
 */

export interface PaginationParams {
  limit?: number;
  offset?: number;
  cursor?: string;
}

export interface PaginatedResponse<T> {
  data: T[];
  pagination: {
    total: number;
    limit: number;
    offset: number;
    hasMore: boolean;
    nextCursor?: string;
  };
}

export interface ApiResponse<T> {
  success: boolean;
  data?: T;
  error?: ApiError;
  requestId?: string;
}

export interface ApiError {
  code: string;
  message: string;
  details?: Record<string, unknown>;
}

export interface Timestamps {
  createdAt: string;
  updatedAt: string;
}

export interface TenantScoped {
  tenantId: string;
}

export type MessageDirection = 'inbound' | 'outbound';
export type MessageStatus = 'pending' | 'sent' | 'delivered' | 'read' | 'failed';
export type ChannelType =
  | 'whatsapp'
  | 'telegram'
  | 'facebook'
  | 'instagram'
  | 'webchat'
  | 'sms'
  | 'email'
  | 'rcs';

export type ContentType =
  | 'text'
  | 'image'
  | 'video'
  | 'audio'
  | 'document'
  | 'location'
  | 'contact'
  | 'sticker';
