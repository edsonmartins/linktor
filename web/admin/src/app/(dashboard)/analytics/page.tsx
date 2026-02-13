'use client'

import { useState } from 'react'
import { useQuery } from '@tanstack/react-query'
import { useTranslations } from 'next-intl'
import { format, subDays } from 'date-fns'
import { queryKeys } from '@/lib/query'
import { api } from '@/lib/api'
import type {
  OverviewAnalytics,
  ConversationAnalytics,
  FlowAnalytics,
  EscalationAnalytics,
  ChannelAnalytics,
  AnalyticsPeriod,
} from '@/types'
import { Header } from '@/components/layout/header'
import { Spinner } from '@/components/ui/spinner'
import { StatsCards } from './components/stats-cards'
import { ConversationsChart } from './components/conversations-chart'
import { EscalationsChart } from './components/escalations-chart'
import { FlowPerformanceTable } from './components/flow-performance-table'
import { ChannelBreakdownTable } from './components/channel-breakdown-table'
import { DateRangePicker } from './components/date-range-picker'

export default function AnalyticsPage() {
  const t = useTranslations('analytics')
  const [period, setPeriod] = useState<AnalyticsPeriod>('weekly')
  const [dateRange, setDateRange] = useState({
    start: format(subDays(new Date(), 7), 'yyyy-MM-dd'),
    end: format(new Date(), 'yyyy-MM-dd'),
  })

  const queryParams: Record<string, string> = {
    period,
    start_date: dateRange.start,
    end_date: dateRange.end,
  }

  // Fetch overview data
  const { data: overview, isLoading: isLoadingOverview } = useQuery({
    queryKey: queryKeys.analytics.overview(queryParams),
    queryFn: () => api.get<OverviewAnalytics>('/analytics/overview', queryParams),
  })

  // Fetch conversation analytics
  const { data: conversationsData, isLoading: isLoadingConversations } = useQuery({
    queryKey: queryKeys.analytics.conversations(queryParams),
    queryFn: async () => {
      const response = await api.get<{ data: ConversationAnalytics[] }>(
        '/analytics/conversations',
        queryParams
      )
      return response.data
    },
  })

  // Fetch flow analytics
  const { data: flowsData, isLoading: isLoadingFlows } = useQuery({
    queryKey: queryKeys.analytics.flows(queryParams),
    queryFn: async () => {
      const response = await api.get<{ data: FlowAnalytics[] }>(
        '/analytics/flows',
        queryParams
      )
      return response.data
    },
  })

  // Fetch escalation analytics
  const { data: escalationsData, isLoading: isLoadingEscalations } = useQuery({
    queryKey: queryKeys.analytics.escalations(queryParams),
    queryFn: async () => {
      const response = await api.get<{ data: EscalationAnalytics[] }>(
        '/analytics/escalations',
        queryParams
      )
      return response.data
    },
  })

  // Fetch channel analytics
  const { data: channelsData, isLoading: isLoadingChannels } = useQuery({
    queryKey: queryKeys.analytics.channels(queryParams),
    queryFn: async () => {
      const response = await api.get<{ data: ChannelAnalytics[] }>(
        '/analytics/channels',
        queryParams
      )
      return response.data
    },
  })

  const handlePeriodChange = (newPeriod: AnalyticsPeriod) => {
    setPeriod(newPeriod)
    const now = new Date()
    let startDate: Date

    switch (newPeriod) {
      case 'daily':
        startDate = subDays(now, 1)
        break
      case 'weekly':
        startDate = subDays(now, 7)
        break
      case 'monthly':
        startDate = subDays(now, 30)
        break
      default:
        startDate = subDays(now, 7)
    }

    setDateRange({
      start: format(startDate, 'yyyy-MM-dd'),
      end: format(now, 'yyyy-MM-dd'),
    })
  }

  const isLoading =
    isLoadingOverview ||
    isLoadingConversations ||
    isLoadingFlows ||
    isLoadingEscalations ||
    isLoadingChannels

  return (
    <div className="flex flex-col h-full">
      <Header title={t('title')}>
        <DateRangePicker
          period={period}
          onPeriodChange={handlePeriodChange}
          dateRange={dateRange}
          onDateRangeChange={setDateRange}
        />
      </Header>

      <div className="p-6 space-y-6 overflow-auto">
        {isLoading ? (
          <div className="flex items-center justify-center py-20">
            <Spinner size="lg" />
          </div>
        ) : (
          <>
            {/* Stats Cards */}
            <StatsCards overview={overview} />

            {/* Charts Row */}
            <div className="grid gap-6 md:grid-cols-2">
              <ConversationsChart data={conversationsData || []} />
              <EscalationsChart data={escalationsData || []} />
            </div>

            {/* Tables Row */}
            <div className="grid gap-6 md:grid-cols-2">
              <FlowPerformanceTable data={flowsData || []} />
              <ChannelBreakdownTable data={channelsData || []} />
            </div>
          </>
        )}
      </div>
    </div>
  )
}
