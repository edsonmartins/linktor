'use client'

import { FlaskConical } from 'lucide-react'
import { useTranslations } from 'next-intl'
import { Badge } from '@/components/ui/badge'
import { cn } from '@/lib/utils'
import type { ChannelEnvironment } from '@/types'

/**
 * EnvironmentBadge — visible marking of synthetic origin (INV-018 / proposta
 * INV-025). The value ALWAYS comes from the backend (`environment` on the
 * channel/conversation); this component never infers it.
 *
 * Accessibility is a requirement, not polish: the distinction never relies on
 * color alone — there is a textual label plus an icon, and the information is
 * exposed to screen readers via role="status" + aria-label.
 *
 * Production renders nothing by default: the presentation of production
 * channels/conversations must not change (WP-H). Pass `showProduction` for the
 * few surfaces (e.g. channel detail) that want the explicit label.
 */
export function EnvironmentBadge({
  environment,
  showProduction = false,
  className,
}: {
  environment?: ChannelEnvironment
  showProduction?: boolean
  className?: string
}) {
  const t = useTranslations('environment')

  if (environment !== 'sandbox') {
    if (!showProduction) return null
    return (
      <Badge variant="outline" className={className} role="status" aria-label={t('production')}>
        {t('production')}
      </Badge>
    )
  }

  return (
    <Badge
      variant="warning"
      role="status"
      aria-label={t('sandboxAria')}
      title={t('sandboxTooltip')}
      className={cn('gap-1 uppercase tracking-wide', className)}
    >
      <FlaskConical className="h-3 w-3" aria-hidden="true" />
      {t('sandbox')}
    </Badge>
  )
}

/**
 * SandboxConversationBanner — persistent, non-dismissable indicator shown for
 * the whole session while a sandbox conversation is open (WP-H). Deliberately
 * has no close button: reading synthetic data as real is the failure mode this
 * exists to prevent.
 */
export function SandboxConversationBanner({
  environment,
}: {
  environment?: ChannelEnvironment
}) {
  const t = useTranslations('environment')
  if (environment !== 'sandbox') return null
  return (
    <div
      role="status"
      aria-label={t('sandboxAria')}
      className="flex items-center gap-2 border-b border-yellow-500/40 bg-yellow-500/10 px-4 py-1.5 text-xs font-medium text-yellow-600 dark:text-yellow-400"
    >
      <FlaskConical className="h-3.5 w-3.5 shrink-0" aria-hidden="true" />
      <span>{t('sandboxBanner')}</span>
    </div>
  )
}
