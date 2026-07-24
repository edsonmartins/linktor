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

// AuditChanges renders the non-secret change payload as compact key=value
// chips. The backend never puts credential VALUES here (INV-002 / WP-D);
// this shows what defines the sandbox boundary — environment,
// credential_environment declaration, phone_number_id, recipient — for the
// prioritized channel.* and sandbox_allowlist.* events.
function AuditChanges({ changes }: { changes?: Record<string, unknown> }) {
  if (!changes || Object.keys(changes).length === 0) return <span className="text-muted-foreground">—</span>
  return (
    <div className="flex flex-wrap gap-1">
      {Object.entries(changes).map(([k, v]) => (
        <Badge key={k} variant="outline" className="max-w-[220px] truncate font-mono text-[10px]">
          {k}={typeof v === 'object' ? JSON.stringify(v) : String(v)}
        </Badge>
      ))}
    </div>
  )
}

export default function AuditLogsPage() {
  const t = useTranslations('audit')

  const [action, setAction] = useState('')
  const [resource, setResource] = useState('')
  const [actor, setActor] = useState('')
  const [startDate, setStartDate] = useState('')
  const [endDate, setEndDate] = useState('')
  const [page, setPage] = useState(1)

  const filters = { action, resource, actor, startDate, endDate, page }
  const { data: envelope, isLoading } = useQuery({
    queryKey: queryKeys.audit.list(filters),
    queryFn: () => {
      const params: Record<string, string> = {
        page: String(page),
        page_size: String(PAGE_SIZE),
      }
      // All filters are applied by the backend (tenant-scoped, admin-only).
      if (action.trim()) params.action = action.trim()
      if (resource.trim()) params.resource_type = resource.trim()
      if (actor.trim()) params.actor = actor.trim()
      if (startDate) params.start_date = startDate
      if (endDate) params.end_date = endDate
      return api.getEnvelope<AuditLog[]>('/audit-logs', params)
    },
  })

  const resetPage = () => setPage(1)

  const logs = envelope?.data ?? []
  const meta = envelope?.meta

  return (
    <>
      <Header title={t('title')} />
      <div className="p-6">
        <p className="mb-4 text-sm text-muted-foreground">{t('subtitle')}</p>

        <div className="mb-4 flex flex-wrap gap-2">
          <Input
            value={actor}
            onChange={(e) => {
              setActor(e.target.value)
              resetPage()
            }}
            placeholder={t('filterActor')}
            className="max-w-xs"
          />
          <Input
            value={action}
            onChange={(e) => {
              setAction(e.target.value)
              resetPage()
            }}
            placeholder={t('filterAction')}
            className="max-w-xs"
          />
          <Input
            value={resource}
            onChange={(e) => {
              setResource(e.target.value)
              resetPage()
            }}
            placeholder={t('filterResource')}
            className="max-w-xs"
          />
          <Input
            type="date"
            aria-label={t('filterStart')}
            value={startDate}
            onChange={(e) => {
              setStartDate(e.target.value)
              resetPage()
            }}
            className="max-w-[160px]"
          />
          <Input
            type="date"
            aria-label={t('filterEnd')}
            value={endDate}
            onChange={(e) => {
              setEndDate(e.target.value)
              resetPage()
            }}
            className="max-w-[160px]"
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
                    <TableHead>{t('details')}</TableHead>
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
                      <TableCell className="text-xs">
                        <AuditChanges changes={l.changes} />
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
