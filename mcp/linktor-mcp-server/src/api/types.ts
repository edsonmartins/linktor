// ============================================
// Linktor API Types
// ============================================

// Common Types
export interface PaginationParams {
  limit?: number;
  offset?: number;
}

export interface PaginatedResponse<T> {
  data: T[];
  meta: {
    page: number;
    page_size: number;
    total_pages: number;
    total_items: number;
    has_next: boolean;
    has_prev: boolean;
  };
}

export interface ApiResponse<T> {
  success: boolean;
  data?: T;
  error?: {
    code: string;
    message: string;
    details?: Record<string, string>;
  };
}

// User Types
export type UserRole = 'agent' | 'supervisor' | 'admin' | 'owner';
export type UserStatus = 'active' | 'inactive' | 'suspended';

export interface User {
  id: string;
  tenant_id: string;
  email: string;
  name: string;
  role: UserRole;
  avatar_url?: string;
  status: UserStatus;
  last_login_at?: string;
  created_at: string;
  updated_at: string;
}

// Contact Types
export interface ContactIdentity {
  id: string;
  channel_type: ChannelType;
  identifier: string;
  metadata?: Record<string, string>;
}

export interface Contact {
  id: string;
  tenant_id: string;
  name: string;
  email?: string;
  phone?: string;
  avatar_url?: string;
  custom_fields?: Record<string, string>;
  tags?: string[];
  identities?: ContactIdentity[];
  created_at: string;
  updated_at: string;
}

export interface CreateContactInput {
  name: string;
  email?: string;
  phone?: string;
  tags?: string[];
  custom_fields?: Record<string, string>;
}

export interface UpdateContactInput {
  name?: string;
  email?: string;
  phone?: string;
  tags?: string[];
  custom_fields?: Record<string, string>;
}

// Channel Types
export type ChannelType =
  | 'webchat'
  | 'whatsapp'
  | 'whatsapp_official'
  | 'telegram'
  | 'sms'
  | 'rcs'
  | 'instagram'
  | 'facebook'
  | 'email'
  | 'voice';

export type ChannelStatus = 'inactive' | 'active' | 'error' | 'disconnected';

export interface Channel {
  id: string;
  tenant_id: string;
  type: ChannelType;
  name: string;
  identifier?: string;
  status: ChannelStatus;
  config?: Record<string, string>;
  webhook_url?: string;
  created_at: string;
  updated_at: string;
}

// Conversation Types
export type ConversationStatus = 'open' | 'pending' | 'resolved' | 'closed';
export type ConversationPriority = 'low' | 'normal' | 'high' | 'urgent';

export interface Conversation {
  id: string;
  tenant_id: string;
  contact_id: string;
  channel_id: string;
  assigned_user_id?: string;
  status: ConversationStatus;
  priority: ConversationPriority;
  subject?: string;
  tags?: string[];
  metadata?: Record<string, string>;
  unread_count: number;
  last_message_at?: string;
  first_reply_at?: string;
  resolved_at?: string;
  created_at: string;
  updated_at: string;
  // Expanded relations
  contact?: Contact;
  channel?: Channel;
  assigned_user?: User;
}

export interface CreateConversationInput {
  contact_id: string;
  channel_id: string;
  subject?: string;
  tags?: string[];
  metadata?: Record<string, string>;
}

export interface ListConversationsParams extends PaginationParams {
  status?: ConversationStatus;
  channel_id?: string;
  assigned_to?: string;
  contact_id?: string;
}

// Message Types
export type SenderType = 'contact' | 'user' | 'system' | 'bot';
export type ContentType =
  | 'text'
  | 'image'
  | 'video'
  | 'audio'
  | 'document'
  | 'location'
  | 'contact'
  | 'template'
  | 'interactive';
export type MessageStatus = 'pending' | 'sent' | 'delivered' | 'read' | 'failed';

export interface MessageAttachment {
  id: string;
  type: string;
  url: string;
  filename?: string;
  mime_type?: string;
  size?: number;
}

export interface Message {
  id: string;
  conversation_id: string;
  sender_type: SenderType;
  sender_id: string;
  content_type: ContentType;
  content: string;
  status: MessageStatus;
  external_id?: string;
  error_message?: string;
  attachments?: MessageAttachment[];
  metadata?: Record<string, string>;
  sent_at?: string;
  delivered_at?: string;
  read_at?: string;
  created_at: string;
}

export interface SendMessageInput {
  content: string;
  content_type?: ContentType;
  attachments?: { url: string; type: string }[];
  metadata?: Record<string, string>;
}

export interface ListMessagesParams {
  limit?: number;
  before?: string;
  after?: string;
}

// Bot Types
export type BotType = 'ai' | 'rule_based' | 'hybrid';
export type AIProviderType = 'openai' | 'anthropic' | 'ollama';
export type BotStatus = 'active' | 'inactive' | 'training';

export interface EscalationRule {
  type: 'low_confidence' | 'sentiment' | 'keyword' | 'intent' | 'user_request';
  value?: string;
  threshold?: number;
}

export interface WorkingHours {
  timezone: string;
  schedule: {
    day: number;
    start: string;
    end: string;
  }[];
}

export interface BotConfig {
  system_prompt: string;
  temperature: number;
  max_tokens: number;
  confidence_threshold: number;
  escalation_rules?: EscalationRule[];
  knowledge_base_id?: string;
  welcome_message?: string;
  fallback_message: string;
  working_hours?: WorkingHours;
  context_window_size: number;
  enabled_intents?: string[];
  max_response_length: number;
}

export interface Bot {
  id: string;
  tenant_id: string;
  name: string;
  type: BotType;
  provider: AIProviderType;
  model: string;
  config: BotConfig;
  status: BotStatus;
  channels?: string[];
  created_at: string;
  updated_at: string;
}

export interface ListBotsParams extends PaginationParams {
  status?: BotStatus;
}

export interface TestBotInput {
  message: string;
  context?: Record<string, string>;
}

export interface TestBotResponse {
  response: string;
  confidence: number;
  intent?: string;
  escalate: boolean;
  processing_time_ms: number;
}

// Knowledge Base Types
export interface KnowledgeDocument {
  id: string;
  tenant_id: string;
  knowledge_base_id: string;
  title: string;
  content: string;
  source_url?: string;
  metadata?: Record<string, string>;
  chunk_count: number;
  created_at: string;
  updated_at: string;
}

export interface SearchKnowledgeParams {
  query: string;
  limit?: number;
  threshold?: number;
}

export interface SearchResult {
  document_id: string;
  title: string;
  content: string;
  score: number;
  metadata?: Record<string, string>;
}

// Analytics Types
export interface AnalyticsParams {
  start_date: string;
  end_date: string;
  metrics?: string[];
  group_by?: 'day' | 'week' | 'month';
}

export interface ConversationStats {
  total: number;
  open: number;
  pending: number;
  resolved: number;
  closed: number;
  avg_response_time_ms: number;
  avg_resolution_time_ms: number;
}

export interface AnalyticsSummary {
  conversations: ConversationStats;
  messages: {
    total: number;
    inbound: number;
    outbound: number;
  };
  channels: {
    channel_id: string;
    channel_name: string;
    conversation_count: number;
    message_count: number;
  }[];
  period: {
    start: string;
    end: string;
  };
}
