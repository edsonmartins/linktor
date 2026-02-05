/**
 * Channels Resource
 */

import type { HttpClient } from '../utils/http';
import type { PaginatedResponse } from '../types/common';
import type {
  Channel,
  ChannelCapabilities,
  CreateChannelInput,
  UpdateChannelInput,
  ListChannelsParams,
} from '../types/channel';

export class ChannelsResource {
  constructor(private http: HttpClient) {}

  /**
   * List channels
   */
  async list(params?: ListChannelsParams): Promise<PaginatedResponse<Channel>> {
    return this.http.get<PaginatedResponse<Channel>>('/channels', { params });
  }

  /**
   * Get a single channel
   */
  async get(id: string): Promise<Channel> {
    return this.http.get<Channel>(`/channels/${id}`);
  }

  /**
   * Create a new channel
   */
  async create(data: CreateChannelInput): Promise<Channel> {
    return this.http.post<Channel>('/channels', data);
  }

  /**
   * Update a channel
   */
  async update(id: string, data: UpdateChannelInput): Promise<Channel> {
    return this.http.patch<Channel>(`/channels/${id}`, data);
  }

  /**
   * Delete a channel
   */
  async delete(id: string): Promise<void> {
    await this.http.delete<void>(`/channels/${id}`);
  }

  /**
   * Connect a channel (activate it)
   */
  async connect(id: string): Promise<Channel> {
    return this.http.post<Channel>(`/channels/${id}/connect`);
  }

  /**
   * Disconnect a channel (deactivate it)
   */
  async disconnect(id: string): Promise<Channel> {
    return this.http.post<Channel>(`/channels/${id}/disconnect`);
  }

  /**
   * Get channel status
   */
  async getStatus(id: string): Promise<{ status: string; lastPing?: string; error?: string }> {
    return this.http.get<{ status: string; lastPing?: string; error?: string }>(
      `/channels/${id}/status`
    );
  }

  /**
   * Get channel capabilities
   */
  async getCapabilities(id: string): Promise<ChannelCapabilities> {
    return this.http.get<ChannelCapabilities>(`/channels/${id}/capabilities`);
  }

  /**
   * Test channel configuration
   */
  async test(id: string): Promise<{ success: boolean; message?: string }> {
    return this.http.post<{ success: boolean; message?: string }>(`/channels/${id}/test`);
  }

  /**
   * Get webhook URL for a channel
   */
  async getWebhookUrl(id: string): Promise<{ url: string; verifyToken?: string }> {
    return this.http.get<{ url: string; verifyToken?: string }>(`/channels/${id}/webhook`);
  }

  /**
   * Regenerate webhook URL/token for a channel
   */
  async regenerateWebhook(id: string): Promise<{ url: string; verifyToken?: string }> {
    return this.http.post<{ url: string; verifyToken?: string }>(`/channels/${id}/webhook/regenerate`);
  }

  /**
   * Get channel statistics
   */
  async getStats(
    id: string,
    period?: 'day' | 'week' | 'month'
  ): Promise<{
    messagesReceived: number;
    messagesSent: number;
    conversationsCreated: number;
    avgResponseTime: number;
  }> {
    return this.http.get<{
      messagesReceived: number;
      messagesSent: number;
      conversationsCreated: number;
      avgResponseTime: number;
    }>(`/channels/${id}/stats`, { params: { period } });
  }
}
