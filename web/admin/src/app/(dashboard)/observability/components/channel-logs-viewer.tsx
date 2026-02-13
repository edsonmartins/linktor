'use client'

import { useState } from 'react'
import { useQuery } from '@tanstack/react-query'
import { useTranslations } from 'next-intl'
import { format } from 'date-fns'
import {
  AlertCircle,
  AlertTriangle,
  Info,
  RefreshCw,
} from 'lucide-react'
import { cn } from '@/lib/utils'
import { queryKeys } from '@/lib/query'
import { api } from '@/lib/api'
import type { LogsResponse, LogLevel, LogSource } from '@/types'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { Badge } from '@/components/ui/badge'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { Spinner } from '@/components/ui/spinner'
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'

const levelIcons = {
  info: Info,
  warn: AlertTriangle,
  error: AlertCircle,
}

const levelColors = {
  info: 'text-blue-500',
  warn: 'text-yellow-500',
  error: 'text-red-500',
}

const levelBadgeVariants = {
  info: 'default' as const,
  warn: 'warning' as const,
  error: 'destructive' as const,
}

export function ChannelLogsViewer() {
  const t = useTranslations('observability')
  const tCommon = useTranslations('common')

  const [level, setLevel] = useState<LogLevel | ''>('')
  const [source, setSource] = useState<LogSource | ''>('')
  const [limit] = useState(50)
  const [offset, setOffset] = useState(0)

  const queryParams: Record<string, string> = {
    limit: limit.toString(),
    offset: offset.toString(),
  }

  if (level) queryParams.level = level
  if (source) queryParams.source = source

  const { data, isLoading, refetch, isFetching } = useQuery({
    queryKey: queryKeys.observability.logs(queryParams),
    queryFn: () => api.get<LogsResponse>('/observability/logs', queryParams),
    refetchInterval: 10000,
  })

  const handleRefresh = () => {
    refetch()
  }

  const handleNextPage = () => {
    setOffset((prev) => prev + limit)
  }

  const handlePrevPage = () => {
    setOffset((prev) => Math.max(0, prev - limit))
  }

  return (
    <Card>
      <CardHeader>
        <div className="flex items-center justify-between">
          <CardTitle className="text-lg flex items-center gap-2">
            <AlertTriangle className="h-5 w-5" />
            {t('channelLogs')}
          </CardTitle>
          <Button
            variant="outline"
            size="sm"
            onClick={handleRefresh}
            disabled={isFetching}
          >
            <RefreshCw className={cn('h-4 w-4 mr-2', isFetching && 'animate-spin')} />
            {tCommon('refresh')}
          </Button>
        </div>

        {/* Filters */}
        <div className="flex gap-4 mt-4">
          <Select
            value={level || 'all'}
            onValueChange={(v) => {
              setLevel(v === 'all' ? '' : (v as LogLevel))
              setOffset(0)
            }}
          >
            <SelectTrigger className="w-[150px]">
              <SelectValue placeholder={t('logLevel')} />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="all">{t('allLevels')}</SelectItem>
              <SelectItem value="info">{t('info')}</SelectItem>
              <SelectItem value="warn">{t('warning')}</SelectItem>
              <SelectItem value="error">{t('error')}</SelectItem>
            </SelectContent>
          </Select>

          <Select
            value={source || 'all'}
            onValueChange={(v) => {
              setSource(v === 'all' ? '' : (v as LogSource))
              setOffset(0)
            }}
          >
            <SelectTrigger className="w-[150px]">
              <SelectValue placeholder={t('source')} />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="all">{t('allSources')}</SelectItem>
              <SelectItem value="channel">{t('channel')}</SelectItem>
              <SelectItem value="queue">{t('queue')}</SelectItem>
              <SelectItem value="system">{t('system')}</SelectItem>
              <SelectItem value="webhook">{t('webhook')}</SelectItem>
            </SelectContent>
          </Select>
        </div>
      </CardHeader>

      <CardContent>
        {isLoading ? (
          <div className="flex items-center justify-center py-10">
            <Spinner size="lg" />
          </div>
        ) : !data?.logs?.length ? (
          <div className="text-center py-10 text-muted-foreground">
            <Info className="h-8 w-8 mx-auto mb-2" />
            <p>{t('noLogsFound')}</p>
          </div>
        ) : (
          <>
            <div className="rounded-md border">
              <Table>
                <TableHeader>
                  <TableRow>
                    <TableHead className="w-[100px]">{t('level')}</TableHead>
                    <TableHead className="w-[100px]">{t('source')}</TableHead>
                    <TableHead className="w-[150px]">{t('channel')}</TableHead>
                    <TableHead>{t('message')}</TableHead>
                    <TableHead className="w-[180px]">{t('time')}</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {data.logs.map((log) => {
                    const LevelIcon = levelIcons[log.level]
                    return (
                      <TableRow key={log.id}>
                        <TableCell>
                          <div className="flex items-center gap-2">
                            <LevelIcon
                              className={cn('h-4 w-4', levelColors[log.level])}
                            />
                            <Badge variant={levelBadgeVariants[log.level]}>
                              {log.level}
                            </Badge>
                          </div>
                        </TableCell>
                        <TableCell>
                          <Badge variant="outline">{log.source}</Badge>
                        </TableCell>
                        <TableCell className="text-muted-foreground">
                          {log.channel_name || '-'}
                        </TableCell>
                        <TableCell>
                          <span className="font-mono text-sm">{log.message}</span>
                        </TableCell>
                        <TableCell className="text-muted-foreground text-sm">
                          {format(new Date(log.created_at), 'MMM dd, HH:mm:ss')}
                        </TableCell>
                      </TableRow>
                    )
                  })}
                </TableBody>
              </Table>
            </div>

            {/* Pagination */}
            <div className="flex items-center justify-between mt-4">
              <p className="text-sm text-muted-foreground">
                {tCommon('showing')} {offset + 1}-{Math.min(offset + limit, data.total)} {tCommon('of')}{' '}
                {data.total} logs
              </p>
              <div className="flex gap-2">
                <Button
                  variant="outline"
                  size="sm"
                  onClick={handlePrevPage}
                  disabled={offset === 0}
                >
                  {tCommon('previous')}
                </Button>
                <Button
                  variant="outline"
                  size="sm"
                  onClick={handleNextPage}
                  disabled={!data.has_more}
                >
                  {tCommon('next')}
                </Button>
              </div>
            </div>
          </>
        )}
      </CardContent>
    </Card>
  )
}
