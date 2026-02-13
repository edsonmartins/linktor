'use client'

import { useTranslations } from 'next-intl'
import {
  PieChart,
  Pie,
  Cell,
  ResponsiveContainer,
  Legend,
  Tooltip,
} from 'recharts'
import type { EscalationAnalytics } from '@/types'

interface EscalationsChartProps {
  data: EscalationAnalytics[]
}

const COLORS = ['#3b82f6', '#10b981', '#f59e0b', '#ef4444', '#8b5cf6', '#ec4899']

const REASON_KEYS: Record<string, string> = {
  low_confidence: 'lowConfidence',
  user_request: 'userRequest',
  negative_sentiment: 'negativeSentiment',
  keyword: 'keyword',
  intent: 'intent',
  unknown: 'unknown',
}

export function EscalationsChart({ data }: EscalationsChartProps) {
  const t = useTranslations('analytics')

  const chartData = data.map((item) => ({
    ...item,
    name: REASON_KEYS[item.reason] ? t(`reasons.${REASON_KEYS[item.reason]}`) : item.reason,
    value: item.count,
  }))

  return (
    <div className="rounded-lg border bg-card p-4 shadow-sm">
      <h3 className="mb-4 text-lg font-semibold">{t('escalationReasons')}</h3>
      <div className="h-[300px]">
        {data.length === 0 ? (
          <div className="flex h-full items-center justify-center text-muted-foreground">
            {t('noEscalations')}
          </div>
        ) : (
          <ResponsiveContainer width="100%" height="100%">
            <PieChart>
              <Pie
                data={chartData}
                cx="50%"
                cy="50%"
                labelLine={false}
                outerRadius={80}
                fill="#8884d8"
                dataKey="value"
                label={({ name, percent }) =>
                  `${name} (${((percent ?? 0) * 100).toFixed(0)}%)`
                }
              >
                {chartData.map((entry, index) => (
                  <Cell
                    key={`cell-${index}`}
                    fill={COLORS[index % COLORS.length]}
                  />
                ))}
              </Pie>
              <Tooltip
                contentStyle={{
                  backgroundColor: 'hsl(var(--card))',
                  border: '1px solid hsl(var(--border))',
                  borderRadius: '8px',
                }}
              />
              <Legend />
            </PieChart>
          </ResponsiveContainer>
        )}
      </div>
    </div>
  )
}
