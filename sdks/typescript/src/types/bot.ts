/**
 * Bot types
 */

import type { PaginationParams, Timestamps } from './common';

export interface Bot extends Timestamps {
  id: string;
  tenantId: string;
  name: string;
  description?: string;
  status: BotStatus;
  type: BotType;
  config: BotConfig;
  channelIds: string[];
  flowId?: string;
  agentId?: string;
  knowledgeBaseIds?: string[];
  metadata?: Record<string, unknown>;
}

export type BotStatus = 'active' | 'inactive' | 'draft';
export type BotType = 'flow' | 'ai' | 'hybrid';

export interface BotConfig {
  welcomeMessage?: string;
  fallbackMessage?: string;
  handoffMessage?: string;
  handoffTriggers?: string[];
  operatingHours?: OperatingHours;
  aiConfig?: AIBotConfig;
}

export interface OperatingHours {
  enabled: boolean;
  timezone: string;
  schedule: DaySchedule[];
  outsideHoursMessage?: string;
}

export interface DaySchedule {
  day: number; // 0-6 (Sunday-Saturday)
  enabled: boolean;
  startTime: string; // HH:mm
  endTime: string; // HH:mm
}

export interface AIBotConfig {
  model: string;
  temperature?: number;
  maxTokens?: number;
  systemPrompt?: string;
  useKnowledgeBase: boolean;
  enableStreaming?: boolean;
}

// Request types
export interface CreateBotInput {
  name: string;
  description?: string;
  type: BotType;
  config?: BotConfig;
  channelIds?: string[];
  flowId?: string;
  agentId?: string;
  knowledgeBaseIds?: string[];
  metadata?: Record<string, unknown>;
}

export interface UpdateBotInput {
  name?: string;
  description?: string;
  status?: BotStatus;
  config?: Partial<BotConfig>;
  channelIds?: string[];
  flowId?: string;
  agentId?: string;
  knowledgeBaseIds?: string[];
  metadata?: Record<string, unknown>;
}

export interface ListBotsParams extends PaginationParams {
  status?: BotStatus;
  type?: BotType;
  channelId?: string;
  search?: string;
}
