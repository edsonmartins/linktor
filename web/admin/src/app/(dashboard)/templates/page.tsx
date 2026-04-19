'use client'

import { useState } from 'react'
import Link from 'next/link'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { useTranslations } from 'next-intl'
import { Plus, Search, RefreshCw, FileText, Trash2, Sparkles } from 'lucide-react'
import { Header } from '@/components/layout/header'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Skeleton } from '@/components/ui/skeleton'
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
import { Badge } from '@/components/ui/badge'
import { useToast } from '@/hooks/use-toast'
import { api } from '@/lib/api'
import { queryKeys } from '@/lib/query'
import type {
  Channel,
  Template,
  TemplateCategory,
  TemplateStatus,
} from '@/types'

const STATUS_VARIANTS: Record<TemplateStatus, 'default' | 'secondary' | 'destructive' | 'outline'> = {
  APPROVED: 'default',
  PENDING: 'secondary',
  IN_APPEAL: 'secondary',
  REINSTATED: 'default',
  REJECTED: 'destructive',
  DISABLED: 'destructive',
  PAUSED: 'secondary',
  PENDING_DELETION: 'outline',
  DELETED: 'outline',
  LIMIT_EXCEEDED: 'destructive',
  ARCHIVED: 'outline',
}

export default function TemplatesPage() {
  const t = useTranslations('templates')
  const tCommon = useTranslations('common')
  const { toast } = useToast()
  const queryClient = useQueryClient()

  const [channelFilter, setChannelFilter] = useState<string>('all')
  const [statusFilter, setStatusFilter] = useState<TemplateStatus | 'all'>('all')
  const [categoryFilter, setCategoryFilter] = useState<TemplateCategory | 'all'>('all')
  const [nameSearch, setNameSearch] = useState('')
  const [templateToDelete, setTemplateToDelete] = useState<Template | null>(null)

  // Load channels (for the filter dropdown). WhatsApp-official only, since
  // only that channel type carries templates today.
  const { data: channelsData } = useQuery({
    queryKey: queryKeys.channels.list(),
    queryFn: () => api.get<Channel[]>('/channels'),
  })
  const whatsappChannels = (channelsData ?? []).filter(
    (c) => c.type === 'whatsapp_official',
  )

  const filters = {
    ...(channelFilter !== 'all' && { channel_id: channelFilter }),
    ...(statusFilter !== 'all' && { status: statusFilter }),
    ...(categoryFilter !== 'all' && { category: categoryFilter }),
    ...(nameSearch && { name: nameSearch }),
  }

  const { data, isLoading, refetch, isFetching } = useQuery({
    queryKey: queryKeys.templates.list(filters),
    queryFn: () => api.get<Template[]>('/templates', filters),
  })

  const templates = data ?? []

  const deleteMutation = useMutation({
    mutationFn: (id: string) => api.delete(`/templates/${id}`),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: queryKeys.templates.all })
      setTemplateToDelete(null)
      toast({ title: tCommon('success'), description: t('deleted') })
    },
    onError: (err: Error) => {
      toast({
        title: tCommon('error'),
        description: err.message,
        variant: 'error',
      })
    },
  })

  const syncMutation = useMutation({
    mutationFn: (channelId: string) =>
      api.post(`/templates/sync/${channelId}`),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: queryKeys.templates.all })
      toast({ title: tCommon('success'), description: t('synced') })
    },
    onError: (err: Error) => {
      toast({
        title: tCommon('error'),
        description: err.message,
        variant: 'error',
      })
    },
  })

  const handleSync = () => {
    // If a channel is selected, sync just that one; otherwise sync each
    // WhatsApp channel sequentially so the admin doesn't have to.
    const targets = channelFilter !== 'all' ? [channelFilter] : whatsappChannels.map((c) => c.id)
    targets.forEach((id) => syncMutation.mutate(id))
  }

  return (
    <div className="flex h-full flex-col">
      <Header title={t('title')} />

      <div className="flex flex-col gap-4 border-b p-4 md:flex-row md:items-center md:justify-between">
        <div className="flex flex-1 flex-col gap-4 md:flex-row md:items-center">
          <div className="relative flex-1">
            <Search className="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground" />
            <Input
              placeholder={t('searchPlaceholder')}
              value={nameSearch}
              onChange={(e) => setNameSearch(e.target.value)}
              className="pl-9"
            />
          </div>

            <Select value={channelFilter} onValueChange={setChannelFilter}>
            <SelectTrigger className="md:w-48">
              <SelectValue placeholder={t('allChannels')} />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="all">{t('allChannels')}</SelectItem>
              {whatsappChannels.map((channel) => (
                <SelectItem key={channel.id} value={channel.id}>
                  {channel.name}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>

          <Select value={statusFilter} onValueChange={(v) => setStatusFilter(v as TemplateStatus | 'all')}>
            <SelectTrigger className="md:w-40">
              <SelectValue placeholder={t('allStatuses')} />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="all">{t('allStatuses')}</SelectItem>
              <SelectItem value="APPROVED">{t('status.APPROVED')}</SelectItem>
              <SelectItem value="PENDING">{t('status.PENDING')}</SelectItem>
              <SelectItem value="REJECTED">{t('status.REJECTED')}</SelectItem>
              <SelectItem value="PAUSED">{t('status.PAUSED')}</SelectItem>
              <SelectItem value="DISABLED">{t('status.DISABLED')}</SelectItem>
            </SelectContent>
          </Select>

          <Select value={categoryFilter} onValueChange={(v) => setCategoryFilter(v as TemplateCategory | 'all')}>
            <SelectTrigger className="md:w-40">
              <SelectValue placeholder={t('allCategories')} />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="all">{t('allCategories')}</SelectItem>
              <SelectItem value="MARKETING">{t('category.MARKETING')}</SelectItem>
              <SelectItem value="UTILITY">{t('category.UTILITY')}</SelectItem>
              <SelectItem value="AUTHENTICATION">{t('category.AUTHENTICATION')}</SelectItem>
            </SelectContent>
          </Select>
        </div>

        <div className="flex gap-2">
          <Button
            variant="outline"
            size="sm"
            onClick={handleSync}
            disabled={isFetching || syncMutation.isPending || whatsappChannels.length === 0}
          >
            <RefreshCw
              className={
                syncMutation.isPending
                  ? 'mr-2 h-4 w-4 animate-spin'
                  : 'mr-2 h-4 w-4'
              }
            />
            {t('sync')}
          </Button>
          <Link href="/templates/library">
            <Button variant="outline" size="sm">
              <Sparkles className="mr-2 h-4 w-4" />
              {t('library.browse')}
            </Button>
          </Link>
          <Link href="/templates/new">
            <Button size="sm">
              <Plus className="mr-2 h-4 w-4" />
              {t('create')}
            </Button>
          </Link>
        </div>
      </div>

      <div className="flex-1 overflow-auto p-4">
        {isLoading ? (
          <div className="space-y-3">
            {[1, 2, 3].map((i) => (
              <Skeleton key={i} className="h-24 w-full" />
            ))}
          </div>
        ) : templates.length === 0 ? (
          <EmptyState
            hasWhatsAppChannel={whatsappChannels.length > 0}
            onRefresh={refetch}
          />
        ) : (
          <div className="space-y-2">
            {templates.map((tpl) => (
              <TemplateRow
                key={tpl.id}
                template={tpl}
                onDelete={() => setTemplateToDelete(tpl)}
              />
            ))}
          </div>
        )}
      </div>

      <AlertDialog
        open={templateToDelete !== null}
        onOpenChange={(open) => !open && setTemplateToDelete(null)}
      >
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>{t('deleteConfirm.title')}</AlertDialogTitle>
            <AlertDialogDescription>
              {t('deleteConfirm.description', {
                name: templateToDelete?.name ?? '',
                language: templateToDelete?.language ?? '',
              })}
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel>{tCommon('cancel')}</AlertDialogCancel>
            <AlertDialogAction
              onClick={() => {
                if (templateToDelete) {
                  deleteMutation.mutate(templateToDelete.id)
                }
              }}
              disabled={deleteMutation.isPending}
            >
              {tCommon('delete')}
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </div>
  )
}

interface TemplateRowProps {
  template: Template
  onDelete: () => void
}

function TemplateRow({ template, onDelete }: TemplateRowProps) {
  const t = useTranslations('templates')

  return (
    <div className="group flex items-center gap-4 rounded-lg border bg-card p-4 transition-colors hover:bg-accent/30">
      <FileText className="h-5 w-5 shrink-0 text-muted-foreground" />

      <Link href={`/templates/${template.id}`} className="min-w-0 flex-1">
        <div className="flex items-center gap-2">
          <span className="truncate font-medium">{template.name}</span>
          <span className="shrink-0 text-xs text-muted-foreground">
            {template.language}
          </span>
        </div>
        <div className="mt-1 flex items-center gap-2 text-xs text-muted-foreground">
          <span>{t(`category.${template.category}`)}</span>
          {template.sub_category && (
            <>
              <span>·</span>
              <span>{template.sub_category}</span>
            </>
          )}
          {template.quality_score && template.quality_score !== 'UNKNOWN' && (
            <>
              <span>·</span>
              <span>
                {t('qualityLabel')}: {template.quality_score}
              </span>
            </>
          )}
        </div>
      </Link>

      <Badge variant={STATUS_VARIANTS[template.status] ?? 'outline'}>
        {t(`status.${template.status}`)}
      </Badge>

      <Button
        variant="ghost"
        size="icon"
        onClick={onDelete}
        className="opacity-0 transition-opacity group-hover:opacity-100"
      >
        <Trash2 className="h-4 w-4" />
      </Button>
    </div>
  )
}

interface EmptyStateProps {
  hasWhatsAppChannel: boolean
  onRefresh: () => void
}

function EmptyState({ hasWhatsAppChannel, onRefresh }: EmptyStateProps) {
  const t = useTranslations('templates')

  return (
    <div className="flex flex-col items-center justify-center rounded-lg border border-dashed py-16 text-center">
      <FileText className="mb-4 h-12 w-12 text-muted-foreground" />
      <h3 className="text-lg font-medium">{t('empty.title')}</h3>
      <p className="mt-1 max-w-sm text-sm text-muted-foreground">
        {hasWhatsAppChannel ? t('empty.description') : t('empty.noChannel')}
      </p>
      {hasWhatsAppChannel && (
        <div className="mt-6 flex gap-2">
          <Link href="/templates/new">
            <Button size="sm">
              <Plus className="mr-2 h-4 w-4" />
              {t('create')}
            </Button>
          </Link>
          <Button variant="outline" size="sm" onClick={onRefresh}>
            <RefreshCw className="mr-2 h-4 w-4" />
            {t('refresh')}
          </Button>
        </div>
      )}
    </div>
  )
}
