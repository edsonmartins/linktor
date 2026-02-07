/**
 * Type definitions for the Linktor Admin Panel
 * Matches the Go backend API models
 */

// Base types
export interface Timestamps {
  created_at: string
  updated_at: string
}

// User
export interface User {
  id: string
  email: string
  name: string
  role: 'admin' | 'agent' | 'supervisor'
  avatar_url?: string
  tenant_id: string
  status: 'active' | 'inactive'
  created_at: string
  updated_at: string
}

// Tenant
export interface Tenant {
  id: string
  name: string
  slug: string
  plan: 'free' | 'starter' | 'professional' | 'enterprise'
  status: 'active' | 'suspended' | 'cancelled'
  settings: Record<string, unknown>
  limits: TenantLimits
  created_at: string
  updated_at: string
}

export interface TenantLimits {
  max_channels: number
  max_users: number
  max_contacts: number
  max_messages_per_month: number
}

// Channel
export interface Channel {
  id: string
  tenant_id: string
  name: string
  type: ChannelType
  status: 'active' | 'inactive' | 'error'
  config: Record<string, unknown>
  created_at: string
  updated_at: string
}

export type ChannelType =
  | 'whatsapp'
  | 'whatsapp_official'
  | 'telegram'
  | 'webchat'
  | 'sms'
  | 'instagram'
  | 'facebook'
  | 'rcs'
  | 'email'
  | 'voice'

// Voice Provider Types
export type VoiceProvider = 'twilio' | 'vonage' | 'amazon_connect' | 'asterisk' | 'freeswitch'

// Contact
export interface Contact {
  id: string
  tenant_id: string
  name: string
  email?: string
  phone?: string
  avatar_url?: string
  metadata: Record<string, unknown>
  identities: ContactIdentity[]
  tags: string[]
  created_at: string
  updated_at: string
}

export interface ContactIdentity {
  id: string
  contact_id: string
  channel_type: ChannelType
  external_id: string
  created_at: string
}

// Conversation
export interface Conversation {
  id: string
  tenant_id: string
  channel_id: string
  contact_id: string
  assigned_user_id?: string
  status: ConversationStatus
  priority: 'low' | 'medium' | 'high' | 'urgent'
  subject?: string
  last_message_at?: string
  metadata: Record<string, unknown>
  created_at: string
  updated_at: string
  // Expanded relations
  contact?: Contact
  channel?: Channel
  assigned_user?: User
  last_message?: Message
  unread_count?: number
}

export type ConversationStatus =
  | 'open'
  | 'pending'
  | 'resolved'
  | 'snoozed'

// Message
export interface Message {
  id: string
  conversation_id: string
  sender_type: 'contact' | 'user' | 'system' | 'bot'
  sender_id?: string
  content_type: MessageContentType
  content: string
  metadata: Record<string, unknown>
  status: MessageStatus
  external_id?: string
  attachments?: MessageAttachment[]
  created_at: string
  updated_at: string
  // Expanded relations
  sender?: User | Contact
}

export type MessageContentType =
  | 'text'
  | 'image'
  | 'video'
  | 'audio'
  | 'document'
  | 'location'
  | 'contact'
  | 'sticker'
  | 'template'

export type MessageStatus =
  | 'pending'
  | 'sent'
  | 'delivered'
  | 'read'
  | 'failed'

export interface MessageAttachment {
  id: string
  message_id: string
  type: 'image' | 'video' | 'audio' | 'document'
  url: string
  filename?: string
  mime_type?: string
  size?: number
  created_at: string
}

// API Response types
export interface PaginatedResponse<T> {
  data: T[]
  total: number
  page: number
  per_page: number
  total_pages: number
}

export interface ApiErrorResponse {
  error: string
  message: string
  details?: Record<string, unknown>
}

// Dashboard Stats
export interface DashboardStats {
  total_conversations: number
  open_conversations: number
  total_messages_today: number
  total_contacts: number
  avg_response_time_seconds: number
  satisfaction_rate: number
}

// WebSocket Events
export interface WSEvent {
  type: string
  payload: unknown
}

export interface WSNewMessageEvent {
  type: 'new_message'
  payload: {
    conversation_id: string
    message: Message
  }
}

export interface WSConversationUpdatedEvent {
  type: 'conversation_updated'
  payload: {
    conversation: Conversation
  }
}

export interface WSTypingEvent {
  type: 'typing'
  payload: {
    conversation_id: string
    user_id: string
    is_typing: boolean
  }
}

// Knowledge Base
export type KnowledgeBaseType = 'faq' | 'documents' | 'website'
export type KnowledgeBaseStatus = 'active' | 'syncing' | 'error'

export interface KnowledgeBase {
  id: string
  tenant_id: string
  name: string
  description: string
  type: KnowledgeBaseType
  status: KnowledgeBaseStatus
  config: {
    language?: string
    auto_sync?: boolean
    sync_interval?: number
  }
  item_count: number
  last_sync_at: string | null
  created_at: string
  updated_at: string
}

export interface KnowledgeItem {
  id: string
  knowledge_base_id: string
  question: string
  answer: string
  keywords: string[]
  source: string
  metadata: Record<string, string>
  has_embedding: boolean
  created_at: string
  updated_at: string
}

export interface KnowledgeSearchResult {
  item: KnowledgeItem
  score: number
}

export interface CreateKnowledgeBaseInput {
  name: string
  description?: string
  type: KnowledgeBaseType
  config?: KnowledgeBase['config']
}

export interface UpdateKnowledgeBaseInput {
  name?: string
  description?: string
  config?: KnowledgeBase['config']
}

export interface CreateKnowledgeItemInput {
  question: string
  answer: string
  keywords?: string[]
  source?: string
  metadata?: Record<string, string>
}

export interface UpdateKnowledgeItemInput {
  question?: string
  answer?: string
  keywords?: string[]
  source?: string
  metadata?: Record<string, string>
}

// Flow (Conversational Decision Trees)
export type FlowTriggerType = 'intent' | 'keyword' | 'manual' | 'welcome'
export type FlowNodeType = 'message' | 'question' | 'condition' | 'action' | 'end'
export type TransitionCondition = 'default' | 'reply_equals' | 'contains' | 'regex'
export type FlowActionType = 'tag' | 'assign' | 'escalate' | 'set_entity' | 'http_call'

export interface Flow {
  id: string
  tenant_id: string
  bot_id?: string
  name: string
  description?: string
  trigger: FlowTriggerType
  trigger_value?: string
  start_node_id: string
  nodes: FlowNode[]
  is_active: boolean
  priority: number
  created_at: string
  updated_at: string
}

export interface FlowNode {
  id: string
  type: FlowNodeType
  content: string
  quick_replies?: QuickReply[]
  transitions: FlowTransition[]
  actions?: FlowAction[]
  metadata?: Record<string, unknown>
  // UI positioning for visual editor
  position?: { x: number; y: number }
}

export interface FlowTransition {
  id: string
  to_node_id: string
  condition: TransitionCondition
  value?: string
}

export interface QuickReply {
  id: string
  title: string
  value?: string
}

export interface FlowAction {
  type: FlowActionType
  config: Record<string, unknown>
}

export interface CreateFlowInput {
  name: string
  description?: string
  bot_id?: string
  trigger: FlowTriggerType
  trigger_value?: string
  start_node_id: string
  nodes: FlowNode[]
  priority?: number
}

export interface UpdateFlowInput {
  name?: string
  description?: string
  trigger_value?: string
  start_node_id?: string
  nodes?: FlowNode[]
  priority?: number
}

export interface FlowTestResult {
  node_id: string
  content: string
  quick_replies?: QuickReply[]
  actions_executed?: string[]
  next_node_id?: string
  flow_ended: boolean
}

// Bot
export type BotType = 'customer_service' | 'sales' | 'faq' | 'custom'
export type AIProvider = 'openai' | 'anthropic' | 'ollama'
export type EscalationRuleType = 'low_confidence' | 'sentiment' | 'keyword' | 'intent' | 'user_request'

export interface Bot {
  id: string
  tenant_id: string
  name: string
  type: BotType
  provider: AIProvider
  model: string
  config: BotConfig
  is_active: boolean
  channel_ids: string[]
  created_at: string
  updated_at: string
  // Expanded relations
  channels?: Channel[]
  knowledge_bases?: KnowledgeBase[]
}

export interface BotConfig {
  system_prompt: string
  temperature: number
  max_tokens: number
  context_window: number
  confidence_threshold: number
  welcome_message: string
  fallback_message: string
  working_hours?: WorkingHours
  knowledge_base_ids: string[]
  escalation_rules: EscalationRule[]
}

export interface WorkingHours {
  enabled: boolean
  timezone: string
  schedule: WeekSchedule
  outside_hours_message: string
}

export interface WeekSchedule {
  monday?: DaySchedule
  tuesday?: DaySchedule
  wednesday?: DaySchedule
  thursday?: DaySchedule
  friday?: DaySchedule
  saturday?: DaySchedule
  sunday?: DaySchedule
}

export interface DaySchedule {
  enabled: boolean
  start: string // "09:00"
  end: string   // "18:00"
}

export interface EscalationRule {
  id: string
  type: EscalationRuleType
  value?: string
  priority: number
  message?: string
}

export interface CreateBotInput {
  name: string
  type: BotType
  provider: AIProvider
  model: string
  config?: Partial<BotConfig>
}

export interface UpdateBotInput {
  name?: string
  type?: BotType
  provider?: AIProvider
  model?: string
}

export interface UpdateBotConfigInput {
  system_prompt?: string
  temperature?: number
  max_tokens?: number
  context_window?: number
  confidence_threshold?: number
  welcome_message?: string
  fallback_message?: string
  working_hours?: WorkingHours
  knowledge_base_ids?: string[]
}

export interface BotTestInput {
  message: string
  conversation_context?: Message[]
}

export interface BotTestResult {
  response: string
  confidence: number
  intent?: string
  entities?: Record<string, string>
  knowledge_used?: KnowledgeItem[]
  should_escalate: boolean
  escalation_reason?: string
}

// Analytics
export type AnalyticsPeriod = 'daily' | 'weekly' | 'monthly'

export interface OverviewAnalytics {
  period: AnalyticsPeriod
  start_date: string
  end_date: string
  total_conversations: number
  resolved_by_bot: number
  escalated_to_human: number
  resolution_rate: number
  avg_first_response_ms: number
  avg_resolution_time_ms: number
  total_bot_messages: number
  avg_confidence: number
  conversations_trend: number
  resolution_trend: number
}

export interface ConversationAnalytics {
  date: string
  total_conversations: number
  resolved_by_bot: number
  escalated: number
  pending: number
}

export interface FlowAnalytics {
  flow_id: string
  flow_name: string
  times_triggered: number
  times_completed: number
  completion_rate: number
  avg_steps_to_end: number
}

export interface EscalationAnalytics {
  reason: string
  count: number
  percentage: number
  avg_time_to_escalation_ms: number
}

export interface ChannelAnalytics {
  channel_id: string
  channel_name: string
  channel_type: string
  total_conversations: number
  resolved_by_bot: number
  resolution_rate: number
}

// Observability
export type LogLevel = 'info' | 'warn' | 'error'
export type LogSource = 'channel' | 'queue' | 'system' | 'webhook'
export type StatsPeriod = 'hour' | 'day' | 'week'

export interface MessageLog {
  id: string
  tenant_id: string
  channel_id?: string
  channel_name?: string
  level: LogLevel
  source: LogSource
  message: string
  metadata?: Record<string, unknown>
  created_at: string
}

export interface LogsResponse {
  logs: MessageLog[]
  total: number
  has_more: boolean
}

export interface ConsumerInfo {
  name: string
  pending: number
  ack_pending: number
  redelivered: number
  last_delivered?: string
}

export interface StreamInfo {
  name: string
  description?: string
  messages: number
  bytes: number
  consumers: ConsumerInfo[]
  first_seq: number
  last_seq: number
  created: string
}

export interface QueueStats {
  streams: StreamInfo[]
  total_messages: number
  total_pending: number
}

export interface HourlyCount {
  hour: string
  count: number
}

export interface ChannelCount {
  channel_id: string
  channel_name: string
  channel_type: string
  count: number
}

export interface MessageStats {
  total: number
  per_hour: HourlyCount[]
  by_channel: ChannelCount[]
}

export interface ResponseTimeStats {
  avg_ms: number
  p95_ms: number
  p99_ms: number
}

export interface ChannelStats {
  total: number
  connected: number
  disconnected: number
}

export interface ConversationStats {
  active: number
  resolved_today: number
  pending: number
}

export interface SourceErrorCount {
  source: LogSource
  count: number
}

export interface ErrorStats {
  last_hour: number
  last_24h: number
  by_source: SourceErrorCount[]
}

export interface SystemStats {
  messages: MessageStats
  response_time: ResponseTimeStats
  channels: ChannelStats
  conversations: ConversationStats
  errors: ErrorStats
}

export interface ResetConsumerRequest {
  stream: string
  consumer: string
}
