'use client'

import { use, useState } from 'react'
import Link from 'next/link'
import { useRouter } from 'next/navigation'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { useTranslations } from 'next-intl'
import { ArrowLeft, RefreshCw, Trash2, Pencil } from 'lucide-react'
import { Header } from '@/components/layout/header'
import { Button } from '@/components/ui/button'
import { Badge } from '@/components/ui/badge'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Skeleton } from '@/components/ui/skeleton'
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
import { useToast } from '@/hooks/use-toast'
import { api } from '@/lib/api'
import { queryKeys } from '@/lib/query'
import type { Template, TemplateComponent, TemplateStatus } from '@/types'

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

export default function TemplateDetailPage({ params }: { params: Promise<{ id: string }> }) {
  const { id } = use(params)
  const t = useTranslations('templates')
  const tCommon = useTranslations('common')
  const router = useRouter()
  const { toast } = useToast()
  const queryClient = useQueryClient()
  const [confirmDelete, setConfirmDelete] = useState(false)

  const { data: template, isLoading } = useQuery({
    queryKey: queryKeys.templates.detail(id),
    queryFn: () => api.get<Template>(`/templates/${id}`),
  })

  const refreshMutation = useMutation({
    mutationFn: () => api.post<Template>(`/templates/${id}/refresh`),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: queryKeys.templates.detail(id) })
      queryClient.invalidateQueries({ queryKey: queryKeys.templates.lists() })
      toast({ title: tCommon('success'), description: t('refreshed') })
    },
    onError: (err: Error) => {
      toast({
        title: tCommon('error'),
        description: err.message,
        variant: 'error',
      })
    },
  })

  const deleteMutation = useMutation({
    mutationFn: () => api.delete(`/templates/${id}`),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: queryKeys.templates.all })
      toast({ title: tCommon('success'), description: t('deleted') })
      router.push('/templates')
    },
    onError: (err: Error) => {
      toast({
        title: tCommon('error'),
        description: err.message,
        variant: 'error',
      })
    },
  })

  if (isLoading) {
    return (
      <div className="flex h-full flex-col">
        <Header title={t('detail.loading')} />
        <div className="space-y-4 p-6">
          <Skeleton className="h-8 w-64" />
          <Skeleton className="h-32 w-full" />
          <Skeleton className="h-48 w-full" />
        </div>
      </div>
    )
  }

  if (!template) {
    return (
      <div className="flex h-full flex-col">
        <Header title={t('detail.notFound')} />
        <div className="p-6">
          <Link href="/templates">
            <Button variant="outline">
              <ArrowLeft className="mr-2 h-4 w-4" />
              {t('backToList')}
            </Button>
          </Link>
        </div>
      </div>
    )
  }

  return (
    <div className="flex h-full flex-col">
      <Header title={template.name}>
        <Link
          href="/templates"
          className="inline-flex items-center gap-1 text-sm text-muted-foreground hover:text-foreground"
        >
          <ArrowLeft className="h-4 w-4" />
          {t('backToList')}
        </Link>
      </Header>

      <div className="flex-1 overflow-auto">
        <div className="mx-auto max-w-4xl space-y-6 p-6">
          <div className="flex items-start justify-between gap-4">
            <div className="space-y-2">
              <div className="flex items-center gap-2">
                <Badge variant={STATUS_VARIANTS[template.status] ?? 'outline'}>
                  {t(`status.${template.status}`)}
                </Badge>
                <Badge variant="outline">{t(`category.${template.category}`)}</Badge>
                {template.sub_category && (
                  <Badge variant="outline">{template.sub_category}</Badge>
                )}
                <span className="text-xs text-muted-foreground">
                  {template.language}
                </span>
              </div>
              {template.rejection_reason && (
                <p className="text-sm text-destructive">
                  <span className="font-medium">{t('detail.rejectionReason')}:</span>{' '}
                  {template.rejection_reason}
                </p>
              )}
            </div>
            <div className="flex gap-2">
              <Link href={`/templates/${id}/edit`}>
                <Button variant="outline" size="sm">
                  <Pencil className="mr-2 h-4 w-4" />
                  {t('edit')}
                </Button>
              </Link>
              <Button
                variant="outline"
                size="sm"
                onClick={() => refreshMutation.mutate()}
                disabled={refreshMutation.isPending || template.external_id === ''}
              >
                <RefreshCw
                  className={
                    refreshMutation.isPending
                      ? 'mr-2 h-4 w-4 animate-spin'
                      : 'mr-2 h-4 w-4'
                  }
                />
                {t('refresh')}
              </Button>
              <Button
                variant="destructive"
                size="sm"
                onClick={() => setConfirmDelete(true)}
              >
                <Trash2 className="mr-2 h-4 w-4" />
                {tCommon('delete')}
              </Button>
            </div>
          </div>

          <Card>
            <CardHeader>
              <CardTitle>{t('detail.preview')}</CardTitle>
            </CardHeader>
            <CardContent>
              <TemplatePreview components={template.components} />
            </CardContent>
          </Card>

          <Card>
            <CardHeader>
              <CardTitle>{t('detail.metadata')}</CardTitle>
            </CardHeader>
            <CardContent className="space-y-2 text-sm">
              <MetaRow label={t('detail.externalId')} value={template.external_id || '—'} />
              <MetaRow
                label={t('detail.quality')}
                value={template.quality_score || 'UNKNOWN'}
              />
              {template.parameter_format && (
                <MetaRow
                  label={t('detail.parameterFormat')}
                  value={template.parameter_format}
                />
              )}
              {template.message_send_ttl_seconds ? (
                <MetaRow
                  label={t('detail.ttl')}
                  value={`${template.message_send_ttl_seconds}s`}
                />
              ) : null}
              <MetaRow
                label={t('detail.createdAt')}
                value={new Date(template.created_at).toLocaleString()}
              />
              {template.last_synced_at && (
                <MetaRow
                  label={t('detail.lastSynced')}
                  value={new Date(template.last_synced_at).toLocaleString()}
                />
              )}
            </CardContent>
          </Card>
        </div>
      </div>

      <AlertDialog open={confirmDelete} onOpenChange={setConfirmDelete}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>{t('deleteConfirm.title')}</AlertDialogTitle>
            <AlertDialogDescription>
              {t('deleteConfirm.description', {
                name: template.name,
                language: template.language,
              })}
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel>{tCommon('cancel')}</AlertDialogCancel>
            <AlertDialogAction
              onClick={() => deleteMutation.mutate()}
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

function MetaRow({ label, value }: { label: string; value: string }) {
  return (
    <div className="flex items-center justify-between border-b py-1.5 last:border-0">
      <span className="text-muted-foreground">{label}</span>
      <span className="font-mono">{value}</span>
    </div>
  )
}

// Renders a simplified WhatsApp chat-bubble preview of the template.
// Intentionally narrow in scope for phase 1: BODY + FOOTER + button row.
// Richer components (carousel / LTO / media header) arrive in phase 2.
function TemplatePreview({ components }: { components: TemplateComponent[] }) {
  const header = components.find((c) => c.type === 'HEADER')
  const body = components.find((c) => c.type === 'BODY')
  const footer = components.find((c) => c.type === 'FOOTER')
  const buttonRow = components.find((c) => c.type === 'BUTTONS')

  return (
    <div className="rounded-lg bg-[#e5ddd5] p-4">
      <div className="mx-auto max-w-md space-y-3 rounded-lg bg-white p-4 shadow-sm">
        {header && (
          <div className="text-sm font-semibold">
            {header.format === 'TEXT' && header.text}
            {header.format === 'IMAGE' && '🖼️ Image header'}
            {header.format === 'VIDEO' && '🎬 Video header'}
            {header.format === 'DOCUMENT' && '📄 Document header'}
            {header.format === 'LOCATION' && '📍 Location header'}
          </div>
        )}
        {body?.text && (
          <div className="whitespace-pre-wrap text-sm">{body.text}</div>
        )}
        {footer?.text && (
          <div className="text-xs text-muted-foreground">{footer.text}</div>
        )}
        {buttonRow?.buttons && buttonRow.buttons.length > 0 && (
          <div className="space-y-1.5 border-t pt-2">
            {buttonRow.buttons.map((btn, i) => (
              <div
                key={i}
                className="flex items-center justify-center rounded border py-1.5 text-sm text-[#00a5f4]"
              >
                {btn.text}
              </div>
            ))}
          </div>
        )}
      </div>
    </div>
  )
}
