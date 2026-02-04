'use client'

import { Calendar, ChevronDown } from 'lucide-react'
import { cn } from '@/lib/utils'
import type { AnalyticsPeriod } from '@/types'

interface DateRangePickerProps {
  period: AnalyticsPeriod
  onPeriodChange: (period: AnalyticsPeriod) => void
  dateRange: { start: string; end: string }
  onDateRangeChange: (range: { start: string; end: string }) => void
}

const PERIODS: { value: AnalyticsPeriod; label: string }[] = [
  { value: 'daily', label: 'Last 24 hours' },
  { value: 'weekly', label: 'Last 7 days' },
  { value: 'monthly', label: 'Last 30 days' },
]

export function DateRangePicker({
  period,
  onPeriodChange,
  dateRange,
  onDateRangeChange,
}: DateRangePickerProps) {
  return (
    <div className="flex items-center gap-4">
      {/* Period Selector */}
      <div className="relative">
        <select
          value={period}
          onChange={(e) => onPeriodChange(e.target.value as AnalyticsPeriod)}
          className="appearance-none rounded-md border bg-card px-4 py-2 pr-10 text-sm font-medium focus:outline-none focus:ring-2 focus:ring-primary"
        >
          {PERIODS.map((p) => (
            <option key={p.value} value={p.value}>
              {p.label}
            </option>
          ))}
        </select>
        <ChevronDown className="pointer-events-none absolute right-3 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground" />
      </div>

      {/* Custom Date Range */}
      <div className="flex items-center gap-2 rounded-md border bg-card px-3 py-1.5">
        <Calendar className="h-4 w-4 text-muted-foreground" />
        <input
          type="date"
          value={dateRange.start}
          onChange={(e) =>
            onDateRangeChange({ ...dateRange, start: e.target.value })
          }
          className="border-none bg-transparent text-sm focus:outline-none"
        />
        <span className="text-muted-foreground">-</span>
        <input
          type="date"
          value={dateRange.end}
          onChange={(e) =>
            onDateRangeChange({ ...dateRange, end: e.target.value })
          }
          className="border-none bg-transparent text-sm focus:outline-none"
        />
      </div>
    </div>
  )
}
