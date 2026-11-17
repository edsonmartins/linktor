'use client'

import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { useTranslations } from 'next-intl'
import { format } from 'date-fns'
import { AlertTriangle, CheckCircle2, RefreshCw, Smartphone } from 'lucide-react'
import { api } from '@/lib/api'
import { queryKeys } from '@/lib/query'
import { cn } from '@/lib/utils'
import type { WhatsAppVersionStatus } from '@/types'
import { Alert, AlertDescription, AlertTitle } from '@/components/ui/alert'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Card, CardContent } from '@/components/ui/card'
import { toastError, toastSuccess } from '@/hooks/use-toast'

function fmt(ts?: string) {
  return ts ? format(new Date(ts), 'dd/MM/yyyy HH:mm') : undefined
}

/**
 * Saúde da versão do WhatsApp Web anunciada pelos canais não-oficiais
 * (whatsmeow). Quando o WhatsApp recusa o build (405 "client outdated") os
 * canais ficam conectando/desconectando sem enviar nada — este card explica
 * o porquê em vez de deixar o operador re-parear à toa.
 */
export function WhatsAppVersionStatusCard() {
  const t = useTranslations('observability.whatsappVersion')
  const queryClient = useQueryClient()

  const { data } = useQuery({
    queryKey: queryKeys.observability.whatsappVersion(),
    queryFn: () => api.get<WhatsAppVersionStatus>('/observability/whatsapp-version'),
    refetchInterval: 60_000,
  })

  const check = useMutation({
    mutationFn: () => api.post<WhatsAppVersionStatus>('/observability/whatsapp-version/check'),
    onSuccess: (fresh) => {
      queryClient.setQueryData(queryKeys.observability.whatsappVersion(), fresh)
      toastSuccess(t('checked'), fresh.current)
    },
    onError: (error) => toastError(t('checkNow'), error instanceof Error ? error.message : ''),
  })

  if (!data) return null

  const statusLabel = {
    ok: t('statusOk'),
    behind: t('statusBehind'),
    outdated: t('statusOutdated'),
    unknown: t('statusUnknown'),
  }[data.status]

  const checkButton = (
    <Button variant="outline" size="sm" onClick={() => check.mutate()} disabled={check.isPending}>
      <RefreshCw className={cn('mr-2 h-4 w-4', check.isPending && 'animate-spin')} />
      {t('checkNow')}
    </Button>
  )

  return (
    <div className="space-y-3">
      {data.status === 'outdated' && (
        <Alert variant="destructive">
          <AlertTriangle className="h-4 w-4" />
          <AlertTitle>{t('outdatedTitle')}</AlertTitle>
          <AlertDescription>
            {t('outdatedBody', {
              count: String(data.outdated_count),
              at: fmt(data.last_outdated_at) ?? '-',
            })}
          </AlertDescription>
        </Alert>
      )}
      {data.status === 'behind' && (
        <Alert className="border-amber-500/50 text-amber-700 dark:text-amber-400 [&>svg]:text-amber-600">
          <AlertTriangle className="h-4 w-4" />
          <AlertTitle>{t('behindTitle')}</AlertTitle>
          <AlertDescription>{t('behindBody')}</AlertDescription>
        </Alert>
      )}

      <Card>
        <CardContent className="flex flex-wrap items-center gap-x-6 gap-y-2 py-3 text-sm">
          <span className="flex items-center gap-2 font-medium">
            <Smartphone className="h-4 w-4 text-primary" />
            {t('title')}
          </span>
          <Badge
            variant={data.status === 'outdated' ? 'destructive' : 'outline'}
            className={cn(
              'gap-1',
              data.status === 'ok' && 'border-green-500/50 text-green-700 dark:text-green-400',
              data.status === 'behind' && 'border-amber-500/50 text-amber-700 dark:text-amber-400',
            )}
          >
            {data.status === 'ok' && <CheckCircle2 className="h-3 w-3" />}
            {statusLabel}
          </Badge>
          <span className="text-muted-foreground">
            {t('current')}: <span className="font-mono text-foreground">{data.current}</span>
          </span>
          {data.latest && (
            <span className="text-muted-foreground">
              {t('latest')}: <span className="font-mono text-foreground">{data.latest}</span>
            </span>
          )}
          <span className="text-muted-foreground">
            {t('lastCheck')}: {fmt(data.last_check_at) ?? t('never')}
          </span>
          <span className="ml-auto">{checkButton}</span>
          {data.last_updated_from && (
            <span className="w-full text-xs text-muted-foreground">
              {t('updatedFrom', { from: data.last_updated_from, at: fmt(data.last_updated_at) ?? '-' })}
              {' · '}
              {t('builtin')}: <span className="font-mono">{data.builtin}</span>
            </span>
          )}
          {data.last_check_error && (
            <span className="w-full text-xs text-destructive">
              {t('checkError', { error: data.last_check_error })}
            </span>
          )}
        </CardContent>
      </Card>
    </div>
  )
}
