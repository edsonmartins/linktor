'use client'

import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { useTranslations } from 'next-intl'
import {
  Database,
  RefreshCw,
  RotateCcw,
  Server,
  Users,
  Clock,
  HardDrive,
} from 'lucide-react'
import { cn } from '@/lib/utils'
import { queryKeys } from '@/lib/query'
import { api } from '@/lib/api'
import type { QueueStats, StreamInfo, ResetConsumerRequest } from '@/types'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { Badge } from '@/components/ui/badge'
import { Spinner } from '@/components/ui/spinner'
import { toastSuccess, toastError } from '@/hooks/use-toast'
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
  AlertDialogTrigger,
} from '@/components/ui/alert-dialog'

function formatBytes(bytes: number): string {
  if (bytes === 0) return '0 B'
  const k = 1024
  const sizes = ['B', 'KB', 'MB', 'GB']
  const i = Math.floor(Math.log(bytes) / Math.log(k))
  return `${parseFloat((bytes / Math.pow(k, i)).toFixed(1))} ${sizes[i]}`
}

function StreamCard({ stream, t, tCommon }: { stream: StreamInfo; t: (key: string, values?: Record<string, string>) => string; tCommon: (key: string) => string }) {
  const queryClient = useQueryClient()

  const resetMutation = useMutation({
    mutationFn: (params: ResetConsumerRequest) =>
      api.post('/observability/queue/reset-consumer', params),
    onSuccess: () => {
      toastSuccess(t('resetConsumer'), t('consumerResetSuccess'))
      queryClient.invalidateQueries({ queryKey: queryKeys.observability.queue() })
    },
    onError: (error) => {
      toastError(t('resetFailed'), error instanceof Error ? error.message : t('resetFailed'))
    },
  })

  const handleResetConsumer = (consumerName: string) => {
    resetMutation.mutate({
      stream: stream.name,
      consumer: consumerName,
    })
  }

  return (
    <Card>
      <CardHeader className="pb-3">
        <div className="flex items-center justify-between">
          <CardTitle className="text-base flex items-center gap-2">
            <Server className="h-4 w-4 text-primary" />
            {stream.name}
          </CardTitle>
          <Badge variant="outline" className="font-mono">
            {stream.messages.toLocaleString()} msgs
          </Badge>
        </div>
        {stream.description && (
          <p className="text-sm text-muted-foreground">{stream.description}</p>
        )}
      </CardHeader>
      <CardContent className="space-y-4">
        {/* Stream Stats */}
        <div className="grid grid-cols-3 gap-4 text-sm">
          <div>
            <p className="text-muted-foreground">{t('size')}</p>
            <p className="font-mono font-medium">{formatBytes(stream.bytes)}</p>
          </div>
          <div>
            <p className="text-muted-foreground">{t('firstSeq')}</p>
            <p className="font-mono font-medium">{stream.first_seq}</p>
          </div>
          <div>
            <p className="text-muted-foreground">{t('lastSeq')}</p>
            <p className="font-mono font-medium">{stream.last_seq}</p>
          </div>
        </div>

        {/* Consumers */}
        {stream.consumers.length > 0 && (
          <div className="space-y-2">
            <p className="text-sm font-medium flex items-center gap-2">
              <Users className="h-4 w-4" />
              {t('consumers')} ({stream.consumers.length})
            </p>
            <div className="space-y-2">
              {stream.consumers.map((consumer) => (
                <div
                  key={consumer.name}
                  className="flex items-center justify-between p-3 rounded-lg border bg-secondary/30"
                >
                  <div className="space-y-1">
                    <p className="text-sm font-mono">{consumer.name}</p>
                    <div className="flex gap-3 text-xs text-muted-foreground">
                      <span>
                        {t('pending')}: <strong className={cn(consumer.pending > 100 && 'text-yellow-500')}>{consumer.pending}</strong>
                      </span>
                      <span>
                        {t('ackPending')}: <strong className={cn(consumer.ack_pending > 50 && 'text-orange-500')}>{consumer.ack_pending}</strong>
                      </span>
                      <span>
                        {t('redelivered')}: <strong className={cn(consumer.redelivered > 10 && 'text-red-500')}>{consumer.redelivered}</strong>
                      </span>
                    </div>
                  </div>

                  <AlertDialog>
                    <AlertDialogTrigger asChild>
                      <Button
                        variant="ghost"
                        size="sm"
                        disabled={resetMutation.isPending}
                      >
                        <RotateCcw className="h-4 w-4" />
                      </Button>
                    </AlertDialogTrigger>
                    <AlertDialogContent>
                      <AlertDialogHeader>
                        <AlertDialogTitle>{t('resetConsumerTitle')}</AlertDialogTitle>
                        <AlertDialogDescription>
                          {t('resetConsumerDescription', { name: consumer.name })}
                        </AlertDialogDescription>
                      </AlertDialogHeader>
                      <AlertDialogFooter>
                        <AlertDialogCancel>{tCommon('cancel')}</AlertDialogCancel>
                        <AlertDialogAction
                          onClick={() => handleResetConsumer(consumer.name)}
                          className="bg-destructive text-destructive-foreground hover:bg-destructive/90"
                        >
                          {t('resetConsumer')}
                        </AlertDialogAction>
                      </AlertDialogFooter>
                    </AlertDialogContent>
                  </AlertDialog>
                </div>
              ))}
            </div>
          </div>
        )}

        {stream.consumers.length === 0 && (
          <p className="text-sm text-muted-foreground text-center py-2">
            {t('noActiveConsumers')}
          </p>
        )}
      </CardContent>
    </Card>
  )
}

export function MessageQueueMonitor() {
  const t = useTranslations('observability')
  const tCommon = useTranslations('common')

  const { data, isLoading, refetch, isFetching } = useQuery({
    queryKey: queryKeys.observability.queue(),
    queryFn: () => api.get<QueueStats>('/observability/queue'),
    refetchInterval: 5000,
  })

  return (
    <div className="space-y-4">
      {/* Header Stats */}
      <div className="flex items-center justify-between">
        <div className="flex gap-6">
          <div className="flex items-center gap-2">
            <Database className="h-5 w-5 text-primary" />
            <div>
              <p className="text-sm text-muted-foreground">{t('totalMessages')}</p>
              <p className="text-xl font-bold">
                {data?.total_messages?.toLocaleString() || 0}
              </p>
            </div>
          </div>
          <div className="flex items-center gap-2">
            <Clock className="h-5 w-5 text-yellow-500" />
            <div>
              <p className="text-sm text-muted-foreground">{t('totalPending')}</p>
              <p className="text-xl font-bold">
                {data?.total_pending?.toLocaleString() || 0}
              </p>
            </div>
          </div>
        </div>

        <Button
          variant="outline"
          size="sm"
          onClick={() => refetch()}
          disabled={isFetching}
        >
          <RefreshCw className={cn('h-4 w-4 mr-2', isFetching && 'animate-spin')} />
          {tCommon('refresh')}
        </Button>
      </div>

      {/* Stream Cards */}
      {isLoading ? (
        <div className="flex items-center justify-center py-20">
          <Spinner size="lg" />
        </div>
      ) : !data?.streams?.length ? (
        <Card>
          <CardContent className="py-10 text-center text-muted-foreground">
            <HardDrive className="h-8 w-8 mx-auto mb-2" />
            <p>{t('noStreamsFound')}</p>
          </CardContent>
        </Card>
      ) : (
        <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-3">
          {data.streams.map((stream) => (
            <StreamCard key={stream.name} stream={stream} t={t} tCommon={tCommon} />
          ))}
        </div>
      )}
    </div>
  )
}
