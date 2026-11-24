import * as React from 'react'
import type { LucideIcon } from 'lucide-react'
import { cn } from '@/lib/utils'
import type { Accent } from '@/components/layout/nav-config'

/**
 * Metric card: label, big number, optional icon and footnote, over a soft
 * mesh in the card's accent colour (`.lk-stat` in globals.css).
 */
export interface StatCardProps extends React.HTMLAttributes<HTMLDivElement> {
  label: React.ReactNode
  value: React.ReactNode
  icon?: LucideIcon
  accent?: Accent
  /** Small line under the value: a trend, a description. */
  footer?: React.ReactNode
}

export function StatCard({
  label,
  value,
  icon: Icon,
  accent = 'lime',
  footer,
  className,
  ...props
}: StatCardProps) {
  return (
    <div className={cn('lk-stat', className)} data-accent={accent} {...props}>
      <div className="flex items-start justify-between gap-3">
        <p className="lk-stat-label">{label}</p>
        {Icon && (
          <span className="lk-stat-icon" aria-hidden>
            <Icon className="h-[17px] w-[17px]" strokeWidth={1.8} />
          </span>
        )}
      </div>
      <div className="lk-stat-value">{value}</div>
      {footer && <div className="mt-1 text-xs text-muted-foreground">{footer}</div>}
    </div>
  )
}
