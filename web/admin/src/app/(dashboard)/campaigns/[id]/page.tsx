'use client'

import { use, useState } from 'react'
import Link from 'next/link'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { useTranslations } from 'next-intl'
import { ArrowLeft, Play, RotateCcw, Ban, ChevronLeft, ChevronRight } from 'lucide-react'
import { Header } from '@/components/layout/header'
import { Button } from '@/components/ui/button'
import { Badge } from '@/components/ui/badge'
import { Progress } from '@/components/ui/progress'
import { Skeleton } from '@/components/ui/skeleton'
import { Card, CardContent } from '@/components/ui/card'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from '@/components/ui/alert-dialog'
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'
import { useToast } from '@/hooks/use-toast'
import { api } from '@/lib/api'
import { queryKeys } from '@/lib/query'
import type {
  Campaign,
  CampaignRecipient,
  CampaignStatus,
  RecipientStatus,
} from '@/types'

const STATUS_VARIANT: Record<CampaignStatus, 'default' | 'secondary' | 'destructive' | 'outline'> = {
  draft: 'outline',
  scheduled: 'secondary',
  processing: 'default',
  completed: 'default',
  failed: 'destructive',
  cancelled: 'outline',
}

const RECIPIENT_VARIANT: Record<RecipientStatus, 'default' | 'secondary' | 'destructive' | 'outline'> = {
  pending: 'outline',
  queued: 'secondary',
  sent: 'default',
  delivered: 'default',
  read: 'default',
  failed: 'destructive',
}

const RECIPIENT_STATUSES: RecipientStatus[] = [
  'pending',
  'queued',
  'sent',
  'delivered',
  'read',
  'failed',
]

const PAGE_SIZE = 50

export default function CampaignDetailPage({
  params,
}: {
  params: Promise<{ id: string }>
}) {
  const { id } = use(params)
  const t = useTranslations('campaigns')
  const { toast } = useToast()
  const queryClient = useQueryClient()

  const [statusFilter, setStatusFilter] = useState<string>('all')
  const [page, setPage] = useState(1)
  const [confirmCancel, setConfirmCancel] = useState(false)

  const { data: campaign, isLoading } = useQuery({
    queryKey: queryKeys.campaigns.detail(id),
    queryFn: () => api.get<Campaign>(`/campaigns/${id}`),
    refetchInterval: (query) =>
      query.state.data?.status === 'processing' ? 2000 : false,
  })

  const { data: recipientsEnvelope } = useQuery({
    queryKey: [...queryKeys.campaigns.recipients(id), statusFilter, page],
    queryFn: () => {
      const params: Record<string, string> = {
        page: String(page),
        page_size: String(PAGE_SIZE),
      }
      if (statusFilter !== 'all') params.status = statusFilter
      return api.getEnvelope<CampaignRecipient[]>(`/campaigns/${id}/recipients`, params)
    },
    refetchInterval: campaign?.status === 'processing' ? 3000 : false,
  })

  const recipients = recipientsEnvelope?.data ?? []
  const meta = recipientsEnvelope?.meta

  function refresh() {
    queryClient.invalidateQueries({ queryKey: queryKeys.campaigns.detail(id) })
    queryClient.invalidateQueries({ queryKey: queryKeys.campaigns.recipients(id) })
  }

  const onErr = (e: Error) =>
    toast({ title: 'Error', description: e.message, variant: 'error' })

  const startMutation = useMutation({
    mutationFn: () => api.post(`/campaigns/${id}/start`),
    onSuccess: () => {
      toast({ title: t('start') })
      refresh()
    },
    onError: onErr,
  })
  const retryMutation = useMutation({
    mutationFn: () => api.post(`/campaigns/${id}/retry`),
    onSuccess: () => {
      toast({ title: t('retry') })
      refresh()
    },
    onError: onErr,
  })
  const cancelMutation = useMutation({
    mutationFn: () => api.post(`/campaigns/${id}/cancel`),
    onSuccess: () => {
      toast({ title: t('cancel') })
      setConfirmCancel(false)
      refresh()
    },
    onError: onErr,
  })

  if (isLoading || !campaign) {
    return (
      <>
        <Header title={t('title')} />
        <div className="space-y-3 p-6">
          <Skeleton className="h-8 w-64" />
          <Skeleton className="h-32 w-full" />
        </div>
      </>
    )
  }

  const pct = campaign.total_recipients
    ? Math.round(((campaign.sent_count + campaign.failed_count) / campaign.total_recipients) * 100)
    : 0
  const canStart = campaign.status === 'draft' || campaign.status === 'scheduled'
  const canRetry = campaign.failed_count > 0 && campaign.status !== 'processing'
  const canCancel = campaign.status === 'processing'

  const stats: { label: string; value: number; tone: string }[] = [
    { label: t('total'), value: campaign.total_recipients, tone: 'text-foreground' },
    { label: t('sent'), value: campaign.sent_count, tone: 'text-blue-600' },
    { label: t('delivered'), value: campaign.delivered_count, tone: 'text-emerald-600' },
    { label: t('read'), value: campaign.read_count, tone: 'text-violet-600' },
    { label: t('failed'), value: campaign.failed_count, tone: 'text-red-600' },
  ]

  return (
    <>
      <Header title={campaign.name}>
        <Link href="/campaigns">
          <Button variant="ghost" size="sm">
            <ArrowLeft className="mr-2 h-4 w-4" />
            {t('backToList')}
          </Button>
        </Link>
      </Header>

      <div className="space-y-6 p-6">
        <div className="flex flex-wrap items-center justify-between gap-3">
          <div className="flex items-center gap-3">
            <Badge variant={STATUS_VARIANT[campaign.status]} className="capitalize">
              {campaign.status}
            </Badge>
            <span className="text-sm text-muted-foreground">
              {campaign.template_name} · {campaign.template_language}
            </span>
          </div>
          <div className="flex gap-2">
            {canStart && (
              <Button size="sm" onClick={() => startMutation.mutate()} disabled={startMutation.isPending}>
                <Play className="mr-2 h-4 w-4" />
                {t('start')}
              </Button>
            )}
            {canRetry && (
              <Button size="sm" variant="outline" onClick={() => retryMutation.mutate()} disabled={retryMutation.isPending}>
                <RotateCcw className="mr-2 h-4 w-4" />
                {t('retry')}
              </Button>
            )}
            {canCancel && (
              <Button size="sm" variant="destructive" onClick={() => setConfirmCancel(true)}>
                <Ban className="mr-2 h-4 w-4" />
                {t('cancel')}
              </Button>
            )}
          </div>
        </div>

        <Card>
          <CardContent className="space-y-4 pt-6">
            <div className="flex items-center gap-3">
              <Progress value={pct} className="h-3" />
              <span className="w-12 text-right text-sm font-medium tabular-nums">{pct}%</span>
            </div>
            <div className="grid grid-cols-2 gap-4 sm:grid-cols-5">
              {stats.map((s) => (
                <div key={s.label} className="rounded-lg border p-3 text-center">
                  <div className={`text-2xl font-semibold tabular-nums ${s.tone}`}>{s.value}</div>
                  <div className="text-xs text-muted-foreground">{s.label}</div>
                </div>
              ))}
            </div>
          </CardContent>
        </Card>

        <div>
          <div className="mb-3 flex items-center justify-between gap-3">
            <h2 className="text-sm font-semibold">{t('recipientsList')}</h2>
            <Select
              value={statusFilter}
              onValueChange={(v) => {
                setStatusFilter(v)
                setPage(1)
              }}
            >
              <SelectTrigger className="w-40">
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="all">{t('filterAll')}</SelectItem>
                {RECIPIENT_STATUSES.map((s) => (
                  <SelectItem key={s} value={s} className="capitalize">
                    {s}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          </div>

          <div className="rounded-lg border">
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>{t('phone')}</TableHead>
                  <TableHead>{t('status')}</TableHead>
                  <TableHead className="text-right">#</TableHead>
                  <TableHead>{t('errorReason')}</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {recipients.map((r) => (
                  <TableRow key={r.id}>
                    <TableCell className="font-mono text-sm">{r.phone}</TableCell>
                    <TableCell>
                      <Badge variant={RECIPIENT_VARIANT[r.status]} className="capitalize">
                        {r.status}
                      </Badge>
                    </TableCell>
                    <TableCell className="text-right tabular-nums text-muted-foreground">
                      {r.attempts}
                    </TableCell>
                    <TableCell className="max-w-[280px] truncate text-xs text-red-600">
                      {r.error_reason}
                    </TableCell>
                  </TableRow>
                ))}
                {recipients.length === 0 && (
                  <TableRow>
                    <TableCell colSpan={4} className="py-8 text-center text-sm text-muted-foreground">
                      —
                    </TableCell>
                  </TableRow>
                )}
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
        </div>
      </div>

      <AlertDialog open={confirmCancel} onOpenChange={setConfirmCancel}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>{t('confirmCancelTitle')}</AlertDialogTitle>
            <AlertDialogDescription>{t('confirmCancelDesc')}</AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel>{t('keepRunning')}</AlertDialogCancel>
            <AlertDialogAction
              onClick={() => cancelMutation.mutate()}
              className="bg-destructive text-destructive-foreground hover:bg-destructive/90"
            >
              {t('confirmCancel')}
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </>
  )
}
