'use client'

import { useState } from 'react'
import { useQuery } from '@tanstack/react-query'
import { useTranslations } from 'next-intl'
import { ScrollText, ChevronLeft, ChevronRight } from 'lucide-react'
import { Header } from '@/components/layout/header'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Badge } from '@/components/ui/badge'
import { Skeleton } from '@/components/ui/skeleton'
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'
import { api } from '@/lib/api'
import { queryKeys } from '@/lib/query'
import type { AuditLog } from '@/types'

const PAGE_SIZE = 50

export default function AuditLogsPage() {
  const t = useTranslations('audit')

  const [action, setAction] = useState('')
  const [resource, setResource] = useState('')
  const [page, setPage] = useState(1)

  const filters = { action, resource, page }
  const { data: envelope, isLoading } = useQuery({
    queryKey: queryKeys.audit.list(filters),
    queryFn: () => {
      const params: Record<string, string> = {
        page: String(page),
        page_size: String(PAGE_SIZE),
      }
      if (action.trim()) params.action = action.trim()
      if (resource.trim()) params.resource_type = resource.trim()
      return api.getEnvelope<AuditLog[]>('/audit-logs', params)
    },
  })

  const logs = envelope?.data ?? []
  const meta = envelope?.meta

  return (
    <>
      <Header title={t('title')} />
      <div className="p-6">
        <p className="mb-4 text-sm text-muted-foreground">{t('subtitle')}</p>

        <div className="mb-4 flex flex-wrap gap-2">
          <Input
            value={action}
            onChange={(e) => {
              setAction(e.target.value)
              setPage(1)
            }}
            placeholder={t('filterAction')}
            className="max-w-xs"
          />
          <Input
            value={resource}
            onChange={(e) => {
              setResource(e.target.value)
              setPage(1)
            }}
            placeholder={t('filterResource')}
            className="max-w-xs"
          />
        </div>

        {isLoading ? (
          <Skeleton className="h-64 w-full" />
        ) : logs.length === 0 ? (
          <div className="flex flex-col items-center justify-center rounded-lg border border-dashed py-16 text-center">
            <ScrollText className="mb-3 h-10 w-10 text-muted-foreground" />
            <p className="text-sm text-muted-foreground">{t('empty')}</p>
          </div>
        ) : (
          <>
            <div className="rounded-lg border">
              <Table>
                <TableHeader>
                  <TableRow>
                    <TableHead>{t('when')}</TableHead>
                    <TableHead>{t('actor')}</TableHead>
                    <TableHead>{t('action')}</TableHead>
                    <TableHead>{t('resource')}</TableHead>
                    <TableHead>{t('ip')}</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {logs.map((l) => (
                    <TableRow key={l.id}>
                      <TableCell className="whitespace-nowrap text-xs text-muted-foreground">
                        {new Date(l.created_at).toLocaleString()}
                      </TableCell>
                      <TableCell className="text-sm">
                        {l.actor_email || l.actor_name || l.actor_id || '—'}
                      </TableCell>
                      <TableCell>
                        <Badge variant="secondary" className="font-mono text-xs">
                          {l.action}
                        </Badge>
                      </TableCell>
                      <TableCell className="font-mono text-xs text-muted-foreground">
                        {l.resource_type}
                        {l.resource_id ? `:${l.resource_id.slice(0, 8)}` : ''}
                      </TableCell>
                      <TableCell className="text-xs text-muted-foreground">{l.ip_address}</TableCell>
                    </TableRow>
                  ))}
                </TableBody>
              </Table>
            </div>

            {meta && meta.total_items > PAGE_SIZE && (
              <div className="mt-3 flex items-center justify-end gap-2 text-sm">
                <span className="text-muted-foreground">
                  {meta.total_items} · {meta.page}/{meta.total_pages}
                </span>
                <Button
                  size="sm"
                  variant="outline"
                  disabled={!meta.has_previous}
                  onClick={() => setPage((p) => Math.max(1, p - 1))}
                >
                  <ChevronLeft className="h-4 w-4" />
                  {t('prev')}
                </Button>
                <Button
                  size="sm"
                  variant="outline"
                  disabled={!meta.has_next}
                  onClick={() => setPage((p) => p + 1)}
                >
                  {t('next')}
                  <ChevronRight className="h-4 w-4" />
                </Button>
              </div>
            )}
          </>
        )}
      </div>
    </>
  )
}
