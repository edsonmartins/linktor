/**
 * Analytics Resource
 */

import type { HttpClient } from '../utils/http';

export interface DateRange {
  startDate: string;
  endDate: string;
}

export interface DashboardMetrics {
  conversations: {
    total: number;
    open: number;
    resolved: number;
    avgResolutionTime: number;
  };
  messages: {
    total: number;
    inbound: number;
    outbound: number;
  };
  contacts: {
    total: number;
    new: number;
  };
  agents: {
    avgResponseTime: number;
    avgHandlingTime: number;
    satisfactionScore?: number;
  };
}

export interface ConversationMetrics {
  totalConversations: number;
  byStatus: { status: string; count: number }[];
  byChannel: { channel: string; count: number }[];
  avgResolutionTime: number;
  avgFirstResponseTime: number;
  timeline: { date: string; count: number }[];
}

export interface MessageMetrics {
  totalMessages: number;
  byDirection: { direction: string; count: number }[];
  byChannel: { channel: string; count: number }[];
  avgPerConversation: number;
  timeline: { date: string; inbound: number; outbound: number }[];
}

export interface AgentMetrics {
  agents: {
    agentId: string;
    name: string;
    conversationsHandled: number;
    avgResponseTime: number;
    avgHandlingTime: number;
    satisfactionScore?: number;
  }[];
  overall: {
    avgResponseTime: number;
    avgHandlingTime: number;
    satisfactionScore?: number;
  };
}

export interface BotMetrics {
  totalInteractions: number;
  handoffRate: number;
  avgResponseTime: number;
  topIntents: { intent: string; count: number }[];
  satisfactionScore?: number;
  byBot: {
    botId: string;
    name: string;
    interactions: number;
    handoffRate: number;
  }[];
}

export interface ChannelMetrics {
  channels: {
    channelId: string;
    name: string;
    type: string;
    messagesReceived: number;
    messagesSent: number;
    conversationsCreated: number;
    avgResponseTime: number;
  }[];
  overall: {
    messagesReceived: number;
    messagesSent: number;
    conversationsCreated: number;
  };
}

export class AnalyticsResource {
  constructor(private http: HttpClient) {}

  /**
   * Get dashboard overview metrics
   */
  async getDashboard(range?: DateRange): Promise<DashboardMetrics> {
    return this.http.get<DashboardMetrics>('/analytics/dashboard', {
      params: range,
    });
  }

  /**
   * Get conversation metrics
   */
  async getConversationMetrics(range?: DateRange): Promise<ConversationMetrics> {
    return this.http.get<ConversationMetrics>('/analytics/conversations', {
      params: range,
    });
  }

  /**
   * Get message metrics
   */
  async getMessageMetrics(range?: DateRange): Promise<MessageMetrics> {
    return this.http.get<MessageMetrics>('/analytics/messages', {
      params: range,
    });
  }

  /**
   * Get agent performance metrics
   */
  async getAgentMetrics(range?: DateRange): Promise<AgentMetrics> {
    return this.http.get<AgentMetrics>('/analytics/agents', {
      params: range,
    });
  }

  /**
   * Get bot performance metrics
   */
  async getBotMetrics(range?: DateRange): Promise<BotMetrics> {
    return this.http.get<BotMetrics>('/analytics/bots', {
      params: range,
    });
  }

  /**
   * Get channel metrics
   */
  async getChannelMetrics(range?: DateRange): Promise<ChannelMetrics> {
    return this.http.get<ChannelMetrics>('/analytics/channels', {
      params: range,
    });
  }

  /**
   * Generate a custom report
   */
  async generateReport(config: {
    type: 'conversations' | 'messages' | 'agents' | 'bots' | 'channels';
    range: DateRange;
    groupBy?: 'day' | 'week' | 'month';
    filters?: Record<string, unknown>;
    format?: 'json' | 'csv';
  }): Promise<{ url?: string; data?: unknown }> {
    return this.http.post<{ url?: string; data?: unknown }>('/analytics/reports', config);
  }

  /**
   * Export analytics data
   */
  async export(config: {
    type: string;
    range: DateRange;
    format: 'csv' | 'xlsx' | 'json';
  }): Promise<{ downloadUrl: string; expiresAt: string }> {
    return this.http.post<{ downloadUrl: string; expiresAt: string }>(
      '/analytics/export',
      config
    );
  }

  /**
   * Get real-time metrics (current active data)
   */
  async getRealtime(): Promise<{
    activeConversations: number;
    activeAgents: number;
    queueSize: number;
    avgWaitTime: number;
    messagesPerMinute: number;
  }> {
    return this.http.get<{
      activeConversations: number;
      activeAgents: number;
      queueSize: number;
      avgWaitTime: number;
      messagesPerMinute: number;
    }>('/analytics/realtime');
  }
}
