'use client'

import { useTranslations } from 'next-intl'
import { TrendingUp, TrendingDown, MessageSquare, Bot, Users, Clock, type LucideIcon } from 'lucide-react'
import { StatCard } from '@/components/ui/stat-card'
import type { Accent } from '@/components/layout/nav-config'
import { cn } from '@/lib/utils'
import type { OverviewAnalytics } from '@/types'

interface StatsCardsProps {
  overview?: OverviewAnalytics
}

export function StatsCards({ overview }: StatsCardsProps) {
  const t = useTranslations('analytics')

  const stats: { label: string; value: string | number; trend?: number; icon: LucideIcon; accent: Accent }[] = [
    {
      label: t('totalConversations'),
      value: overview?.total_conversations ?? 0,
      trend: overview?.conversations_trend ?? 0,
      icon: MessageSquare,
      accent: 'blue',
    },
    {
      label: t('resolutionRate'),
      value: `${(overview?.resolution_rate ?? 0).toFixed(1)}%`,
      trend: overview?.resolution_trend ?? 0,
      icon: Bot,
      accent: 'teal',
    },
    {
      label: t('avgResponseTime'),
      value: formatDuration(overview?.avg_first_response_ms ?? 0),
      icon: Clock,
      accent: 'amber',
    },
    {
      label: t('avgConfidence'),
      value: (overview?.avg_confidence ?? 0).toFixed(2),
      icon: Users,
      accent: 'violet',
    },
  ]

  return (
    <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-4">
      {stats.map((stat) => (
        <StatCard
          key={stat.label}
          label={stat.label}
          value={stat.value}
          icon={stat.icon}
          accent={stat.accent}
          footer={stat.trend !== undefined ? <TrendIndicator value={stat.trend} /> : undefined}
        />
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
        isPositive ? 'text-emerald-600 dark:text-emerald-400' : 'text-red-600 dark:text-red-400'
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
