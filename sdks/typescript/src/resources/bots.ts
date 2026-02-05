/**
 * Bots Resource
 */

import type { HttpClient } from '../utils/http';
import type { PaginatedResponse } from '../types/common';
import type {
  Bot,
  CreateBotInput,
  UpdateBotInput,
  ListBotsParams,
} from '../types/bot';

export class BotsResource {
  constructor(private http: HttpClient) {}

  /**
   * List bots
   */
  async list(params?: ListBotsParams): Promise<PaginatedResponse<Bot>> {
    return this.http.get<PaginatedResponse<Bot>>('/bots', { params });
  }

  /**
   * Get a single bot
   */
  async get(id: string): Promise<Bot> {
    return this.http.get<Bot>(`/bots/${id}`);
  }

  /**
   * Create a new bot
   */
  async create(data: CreateBotInput): Promise<Bot> {
    return this.http.post<Bot>('/bots', data);
  }

  /**
   * Update a bot
   */
  async update(id: string, data: UpdateBotInput): Promise<Bot> {
    return this.http.patch<Bot>(`/bots/${id}`, data);
  }

  /**
   * Delete a bot
   */
  async delete(id: string): Promise<void> {
    await this.http.delete<void>(`/bots/${id}`);
  }

  /**
   * Activate a bot
   */
  async activate(id: string): Promise<Bot> {
    return this.update(id, { status: 'active' });
  }

  /**
   * Deactivate a bot
   */
  async deactivate(id: string): Promise<Bot> {
    return this.update(id, { status: 'inactive' });
  }

  /**
   * Assign bot to channels
   */
  async assignToChannels(id: string, channelIds: string[]): Promise<Bot> {
    return this.http.post<Bot>(`/bots/${id}/channels`, { channelIds });
  }

  /**
   * Remove bot from channels
   */
  async removeFromChannels(id: string, channelIds: string[]): Promise<Bot> {
    return this.http.delete<Bot>(`/bots/${id}/channels`, {
      data: { channelIds },
    });
  }

  /**
   * Assign knowledge bases to bot
   */
  async assignKnowledgeBases(id: string, knowledgeBaseIds: string[]): Promise<Bot> {
    return this.http.post<Bot>(`/bots/${id}/knowledge-bases`, { knowledgeBaseIds });
  }

  /**
   * Remove knowledge bases from bot
   */
  async removeKnowledgeBases(id: string, knowledgeBaseIds: string[]): Promise<Bot> {
    return this.http.delete<Bot>(`/bots/${id}/knowledge-bases`, {
      data: { knowledgeBaseIds },
    });
  }

  /**
   * Get bot statistics
   */
  async getStats(
    id: string,
    period?: 'day' | 'week' | 'month'
  ): Promise<{
    conversationsHandled: number;
    messagesProcessed: number;
    handoffRate: number;
    avgResponseTime: number;
    satisfactionScore?: number;
  }> {
    return this.http.get<{
      conversationsHandled: number;
      messagesProcessed: number;
      handoffRate: number;
      avgResponseTime: number;
      satisfactionScore?: number;
    }>(`/bots/${id}/stats`, { params: { period } });
  }

  /**
   * Duplicate a bot
   */
  async duplicate(id: string, name?: string): Promise<Bot> {
    return this.http.post<Bot>(`/bots/${id}/duplicate`, { name });
  }
}
