/**
 * Conversation and Message types
 */

import type {
  ChannelType,
  ContentType,
  MessageDirection,
  MessageStatus,
  PaginationParams,
  Timestamps,
} from './common';
import type { Contact } from './contact';

export interface Conversation extends Timestamps {
  id: string;
  tenantId: string;
  channelId: string;
  channelType: ChannelType;
  contactId: string;
  contact?: Contact;
  status: ConversationStatus;
  assignedTo?: string;
  assignedAt?: string;
  lastMessage?: Message;
  lastMessageAt?: string;
  unreadCount: number;
  metadata?: Record<string, unknown>;
  tags?: string[];
}

export type ConversationStatus = 'open' | 'pending' | 'resolved' | 'snoozed';

export interface Message extends Timestamps {
  id: string;
  conversationId: string;
  direction: MessageDirection;
  contentType: ContentType;
  content: MessageContent;
  status: MessageStatus;
  externalId?: string;
  senderId?: string;
  senderType: 'contact' | 'agent' | 'bot' | 'system';
  deliveredAt?: string;
  readAt?: string;
  failedReason?: string;
  metadata?: Record<string, unknown>;
}

export interface MessageContent {
  text?: string;
  media?: MediaContent;
  location?: LocationContent;
  contact?: ContactContent;
  buttons?: ButtonContent[];
  listSections?: ListSection[];
  template?: TemplateContent;
}

export interface MediaContent {
  url: string;
  mimeType: string;
  filename?: string;
  size?: number;
  caption?: string;
  thumbnailUrl?: string;
}

export interface LocationContent {
  latitude: number;
  longitude: number;
  name?: string;
  address?: string;
}

export interface ContactContent {
  name: string;
  phones?: string[];
  emails?: string[];
}

export interface ButtonContent {
  type: 'reply' | 'url' | 'call';
  text: string;
  payload?: string;
  url?: string;
  phone?: string;
}

export interface ListSection {
  title: string;
  rows: ListRow[];
}

export interface ListRow {
  id: string;
  title: string;
  description?: string;
}

export interface TemplateContent {
  name: string;
  language: string;
  components?: TemplateComponent[];
}

export interface TemplateComponent {
  type: 'header' | 'body' | 'button';
  parameters?: TemplateParameter[];
}

export interface TemplateParameter {
  type: 'text' | 'image' | 'document' | 'video';
  text?: string;
  media?: MediaContent;
}

// Request types
export interface ListConversationsParams extends PaginationParams {
  status?: ConversationStatus;
  channelId?: string;
  channelType?: ChannelType;
  assignedTo?: string;
  contactId?: string;
  tags?: string[];
  search?: string;
  sortBy?: 'lastMessageAt' | 'createdAt' | 'updatedAt';
  sortOrder?: 'asc' | 'desc';
}

export interface SendMessageInput {
  text?: string;
  media?: {
    type: ContentType;
    url: string;
    filename?: string;
    caption?: string;
  };
  location?: LocationContent;
  buttons?: ButtonContent[];
  template?: TemplateContent;
  metadata?: Record<string, unknown>;
}

export interface UpdateConversationInput {
  status?: ConversationStatus;
  assignedTo?: string | null;
  tags?: string[];
  metadata?: Record<string, unknown>;
}

export interface ListMessagesParams extends PaginationParams {
  before?: string;
  after?: string;
}
