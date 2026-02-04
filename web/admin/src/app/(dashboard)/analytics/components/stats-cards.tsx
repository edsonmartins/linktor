'use client'

import { TrendingUp, TrendingDown, MessageSquare, Bot, Users, Clock } from 'lucide-react'
import { cn } from '@/lib/utils'
import type { OverviewAnalytics } from '@/types'

interface StatsCardsProps {
  overview?: OverviewAnalytics
}

export function StatsCards({ overview }: StatsCardsProps) {
  const stats = [
    {
      label: 'Total Conversations',
      value: overview?.total_conversations ?? 0,
      trend: overview?.conversations_trend ?? 0,
      icon: MessageSquare,
      color: 'text-blue-600',
      bgColor: 'bg-blue-100',
    },
    {
      label: 'Resolution Rate',
      value: `${(overview?.resolution_rate ?? 0).toFixed(1)}%`,
      trend: overview?.resolution_trend ?? 0,
      icon: Bot,
      color: 'text-emerald-600',
      bgColor: 'bg-emerald-100',
    },
    {
      label: 'Avg Response Time',
      value: formatDuration(overview?.avg_first_response_ms ?? 0),
      icon: Clock,
      color: 'text-amber-600',
      bgColor: 'bg-amber-100',
    },
    {
      label: 'Avg Confidence',
      value: (overview?.avg_confidence ?? 0).toFixed(2),
      icon: Users,
      color: 'text-purple-600',
      bgColor: 'bg-purple-100',
    },
  ]

  return (
    <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-4">
      {stats.map((stat) => (
        <div
          key={stat.label}
          className="rounded-lg border bg-card p-4 shadow-sm"
        >
          <div className="flex items-center justify-between">
            <div className={cn('rounded-md p-2', stat.bgColor)}>
              <stat.icon className={cn('h-5 w-5', stat.color)} />
            </div>
            {stat.trend !== undefined && (
              <TrendIndicator value={stat.trend} />
            )}
          </div>
          <div className="mt-3">
            <p className="text-2xl font-semibold">{stat.value}</p>
            <p className="text-sm text-muted-foreground">{stat.label}</p>
          </div>
        </div>
      ))}
    </div>
  )
}

function TrendIndicator({ value }: { value: number }) {
  if (value === 0) return null

  const isPositive = value > 0
  const Icon = isPositive ? TrendingUp : TrendingDown

  return (
    <div
      className={cn(
        'flex items-center gap-1 text-sm font-medium',
        isPositive ? 'text-emerald-600' : 'text-red-600'
      )}
    >
      <Icon className="h-4 w-4" />
      <span>{isPositive ? '+' : ''}{value.toFixed(1)}%</span>
    </div>
  )
}

function formatDuration(ms: number): string {
  if (ms < 1000) return `${ms}ms`
  const seconds = ms / 1000
  if (seconds < 60) return `${seconds.toFixed(1)}s`
  const minutes = seconds / 60
  if (minutes < 60) return `${minutes.toFixed(1)}m`
  const hours = minutes / 60
  return `${hours.toFixed(1)}h`
}
