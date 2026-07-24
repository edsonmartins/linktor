'use client'

import { useTranslations } from 'next-intl'
import { ShieldX, ServerCrash } from 'lucide-react'
import type { Message } from '@/types'

/**
 * MessageFailureDetail — WP-K: on a failed outbound message, distinguishes a
 * LOCAL delivery-guard block from a PROVIDER rejection, in actionable language,
 * with the raw technical value available.
 *
 * The classification is NOT inferred here (INV-014): it reads
 * `metadata.blocked_by`, set by the backend when a guard stops the send. An
 * unknown reason is shown raw — never translated by guessing.
 */
const KNOWN_REASONS: Record<string, string> = {
  allowlist: 'reasonAllowlist',
  window_24h: 'reasonWindow24h',
  template_rejected: 'reasonTemplateRejected',
  invalid_recipient: 'reasonInvalidRecipient',
  unsupported_channel_type: 'reasonUnsupported',
}

export function MessageFailureDetail({ message }: { message: Message }) {
  const t = useTranslations('messageFailure')
  if (message.status !== 'failed') return null

  const blockedBy =
    typeof message.metadata?.blocked_by === 'string'
      ? (message.metadata.blocked_by as string)
      : undefined

  const isLocalBlock = !!blockedBy
  const reasonKey = blockedBy ? KNOWN_REASONS[blockedBy] : undefined

  const reasonText = isLocalBlock
    ? reasonKey
      ? t(reasonKey)
      : t('reasonUnknown', { reason: blockedBy! }) // unknown reason: show raw value, no guessing
    : undefined

  return (
    <div
      role="note"
      className="mt-1 rounded-md border border-destructive/40 bg-destructive/5 px-2 py-1.5 text-[11px] text-destructive"
    >
      <div className="flex items-center gap-1.5 font-medium">
        {isLocalBlock ? (
          <ShieldX className="h-3 w-3 shrink-0" aria-hidden="true" />
        ) : (
          <ServerCrash className="h-3 w-3 shrink-0" aria-hidden="true" />
        )}
        <span>{isLocalBlock ? t('localBlock') : t('remoteFailure')}</span>
      </div>
      {reasonText && <p className="mt-0.5 text-foreground/80">{reasonText}</p>}
      {message.error_message && (
        <p className="mt-0.5 break-words text-muted-foreground">
          <span className="font-medium">{t('technicalDetail')}:</span> {message.error_message}
        </p>
      )}
    </div>
  )
}
