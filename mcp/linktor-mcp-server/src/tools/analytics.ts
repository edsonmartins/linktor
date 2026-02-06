// ============================================
// Analytics Tools
// ============================================

import type { Tool } from '@modelcontextprotocol/sdk/types.js';
import type { LinktorClient } from '../api/client.js';
import type { AnalyticsParams } from '../api/types.js';

export const analyticsToolDefinitions: Tool[] = [
  {
    name: 'get_analytics_summary',
    description: 'Get analytics summary including conversation stats, message counts, and channel performance for a date range.',
    inputSchema: {
      type: 'object',
      properties: {
        start_date: {
          type: 'string',
          description: 'Start date in ISO 8601 format (YYYY-MM-DD)',
        },
        end_date: {
          type: 'string',
          description: 'End date in ISO 8601 format (YYYY-MM-DD)',
        },
        metrics: {
          type: 'array',
          items: { type: 'string' },
          description: 'Specific metrics to include (optional)',
        },
        group_by: {
          type: 'string',
          enum: ['day', 'week', 'month'],
          description: 'Group results by time period',
        },
      },
      required: ['start_date', 'end_date'],
    },
  },
  {
    name: 'get_conversation_stats',
    description: 'Get conversation statistics including total counts, response times, and resolution rates.',
    inputSchema: {
      type: 'object',
      properties: {
        period: {
          type: 'string',
          enum: ['day', 'week', 'month'],
          description: 'Time period for the statistics',
        },
      },
      required: ['period'],
    },
  },
];

export function registerAnalyticsTools(
  handlers: Map<string, (args: Record<string, unknown>) => Promise<unknown>>,
  client: LinktorClient
): void {
  handlers.set('get_analytics_summary', async (args) => {
    const params: AnalyticsParams = {
      start_date: args.start_date as string,
      end_date: args.end_date as string,
      metrics: args.metrics as string[] | undefined,
      group_by: args.group_by as 'day' | 'week' | 'month' | undefined,
    };
    return client.analytics.getSummary(params);
  });

  handlers.set('get_conversation_stats', async (args) => {
    return client.analytics.getConversationStats(args.period as 'day' | 'week' | 'month');
  });
}
