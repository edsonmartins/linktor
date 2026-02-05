/**
 * Contact types
 */

import type { ChannelType, PaginationParams, Timestamps } from './common';

export interface Contact extends Timestamps {
  id: string;
  tenantId: string;
  externalId?: string;
  name: string;
  email?: string;
  phone?: string;
  avatarUrl?: string;
  identifiers: ContactIdentifier[];
  customFields?: Record<string, unknown>;
  tags?: string[];
  metadata?: Record<string, unknown>;
  lastSeenAt?: string;
  conversationCount: number;
}

export interface ContactIdentifier {
  channelType: ChannelType;
  channelId: string;
  identifier: string;
  displayName?: string;
  profilePicture?: string;
  metadata?: Record<string, unknown>;
}

// Request types
export interface CreateContactInput {
  name: string;
  email?: string;
  phone?: string;
  externalId?: string;
  avatarUrl?: string;
  customFields?: Record<string, unknown>;
  tags?: string[];
  identifiers?: Omit<ContactIdentifier, 'metadata'>[];
}

export interface UpdateContactInput {
  name?: string;
  email?: string;
  phone?: string;
  externalId?: string;
  avatarUrl?: string;
  customFields?: Record<string, unknown>;
  tags?: string[];
}

export interface ListContactsParams extends PaginationParams {
  search?: string;
  email?: string;
  phone?: string;
  tags?: string[];
  channelType?: ChannelType;
  sortBy?: 'name' | 'createdAt' | 'lastSeenAt';
  sortOrder?: 'asc' | 'desc';
}

export interface MergeContactsInput {
  primaryContactId: string;
  secondaryContactIds: string[];
}
