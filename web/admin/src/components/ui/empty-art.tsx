import type { LucideIcon } from 'lucide-react'
import { cn } from '@/lib/utils'
import type { Accent } from '@/components/layout/nav-config'

/**
 * Picture for an empty state: the icon in a gradient disc with two dashed
 * rings (`.lk-empty-art` in globals.css). Without `accent` it takes the
 * colour of the section it is in, which the dashboard layout sets on <main>.
 */
export function EmptyArt({
  icon: Icon,
  accent,
  className,
}: {
  icon: LucideIcon
  accent?: Accent
  className?: string
}) {
  return (
    <div className={cn('lk-empty-art', className)} data-accent={accent} aria-hidden>
      <span className="lk-empty-icon">
        <Icon strokeWidth={1.8} />
      </span>
    </div>
  )
}
