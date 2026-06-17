'use client'

import Link from 'next/link'
import { useQuery } from '@tanstack/react-query'
import { useTranslations } from 'next-intl'
import { Plus, Megaphone, ChevronRight } from 'lucide-react'
import { Header } from '@/components/layout/header'
import { Button } from '@/components/ui/button'
import { Badge } from '@/components/ui/badge'
import { Skeleton } from '@/components/ui/skeleton'
import { Progress } from '@/components/ui/progress'
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
import type { Campaign, CampaignStatus } from '@/types'

const STATUS_VARIANT: Record<CampaignStatus, 'default' | 'secondary' | 'destructive' | 'outline'> = {
  draft: 'outline',
  scheduled: 'secondary',
  processing: 'default',
  completed: 'default',
  failed: 'destructive',
  cancelled: 'outline',
}

function progressPct(c: Campaign): number {
  if (!c.total_recipients) return 0
  return Math.round(((c.sent_count + c.failed_count) / c.total_recipients) * 100)
}

export default function CampaignsPage() {
  const t = useTranslations('campaigns')

  const { data: campaigns, isLoading } = useQuery({
    queryKey: queryKeys.campaigns.list({}),
    queryFn: () => api.get<Campaign[]>('/campaigns'),
    // Keep statuses/counters fresh while campaigns are processing.
    refetchInterval: 5000,
  })

  return (
    <>
      <Header title={t('title')}>
        <Link href="/campaigns/new">
          <Button size="sm">
            <Plus className="mr-2 h-4 w-4" />
            {t('newCampaign')}
          </Button>
        </Link>
      </Header>

      <div className="p-6">
        <p className="mb-4 text-sm text-muted-foreground">{t('subtitle')}</p>

        {isLoading ? (
          <div className="space-y-2">
            {Array.from({ length: 4 }).map((_, i) => (
              <Skeleton key={i} className="h-14 w-full" />
            ))}
          </div>
        ) : !campaigns || campaigns.length === 0 ? (
          <div className="flex flex-col items-center justify-center rounded-lg border border-dashed py-16 text-center">
            <Megaphone className="mb-3 h-10 w-10 text-muted-foreground" />
            <p className="mb-4 text-sm text-muted-foreground">{t('empty')}</p>
            <Link href="/campaigns/new">
              <Button size="sm">
                <Plus className="mr-2 h-4 w-4" />
                {t('newCampaign')}
              </Button>
            </Link>
          </div>
        ) : (
          <div className="rounded-lg border">
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>{t('name')}</TableHead>
                  <TableHead>{t('template')}</TableHead>
                  <TableHead>{t('status')}</TableHead>
                  <TableHead className="w-[220px]">{t('progress')}</TableHead>
                  <TableHead className="text-right">{t('total')}</TableHead>
                  <TableHead className="w-10" />
                </TableRow>
              </TableHeader>
              <TableBody>
                {campaigns.map((c) => (
                  <TableRow key={c.id} className="cursor-pointer">
                    <TableCell className="font-medium">
                      <Link href={`/campaigns/${c.id}`} className="hover:underline">
                        {c.name}
                      </Link>
                    </TableCell>
                    <TableCell className="text-muted-foreground">
                      {c.template_name}
                    </TableCell>
                    <TableCell>
                      <Badge variant={STATUS_VARIANT[c.status]} className="capitalize">
                        {c.status}
                      </Badge>
                    </TableCell>
                    <TableCell>
                      <div className="flex items-center gap-2">
                        <Progress value={progressPct(c)} className="h-2" />
                        <span className="w-10 text-right text-xs text-muted-foreground">
                          {progressPct(c)}%
                        </span>
                      </div>
                    </TableCell>
                    <TableCell className="text-right tabular-nums">
                      {c.total_recipients}
                    </TableCell>
                    <TableCell>
                      <Link href={`/campaigns/${c.id}`}>
                        <ChevronRight className="h-4 w-4 text-muted-foreground" />
                      </Link>
                    </TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          </div>
        )}
      </div>
    </>
  )
}
