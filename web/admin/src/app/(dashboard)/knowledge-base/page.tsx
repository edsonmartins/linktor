'use client'

import { useState } from 'react'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { useTranslations } from 'next-intl'
import { Plus, Search, BookOpen, FileText, Globe, RefreshCw } from 'lucide-react'
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
import { useToast } from '@/hooks/use-toast'
import { cn } from '@/lib/utils'
import { api } from '@/lib/api'
import { queryKeys } from '@/lib/query'
import type { KnowledgeBase, KnowledgeBaseType, KnowledgeBaseStatus, PaginatedResponse } from '@/types'
import { KnowledgeBaseCard } from './components/knowledge-base-card'
import { KnowledgeBaseForm } from './components/knowledge-base-form'

export default function KnowledgeBasePage() {
  const t = useTranslations('knowledgeBase')
  const tCommon = useTranslations('common')
  const { toast } = useToast()
  const [searchQuery, setSearchQuery] = useState('')
  const [typeFilter, setTypeFilter] = useState<KnowledgeBaseType | 'all'>('all')
  const [statusFilter, setStatusFilter] = useState<KnowledgeBaseStatus | 'all'>('all')
  const [isFormOpen, setIsFormOpen] = useState(false)
  const [editingKb, setEditingKb] = useState<KnowledgeBase | null>(null)
  const [kbToDelete, setKbToDelete] = useState<KnowledgeBase | null>(null)

  const queryClient = useQueryClient()

  const { data, isLoading, refetch, isFetching } = useQuery({
    queryKey: queryKeys.knowledgeBases.list({ search: searchQuery, type: typeFilter, status: statusFilter }),
    queryFn: () =>
      api.get<PaginatedResponse<KnowledgeBase>>('/knowledge-bases', {
        ...(searchQuery && { search: searchQuery }),
        ...(typeFilter !== 'all' && { type: typeFilter }),
        ...(statusFilter !== 'all' && { status: statusFilter }),
      }),
  })

  const knowledgeBases = data?.data ?? []

  const deleteMutation = useMutation({
    mutationFn: (id: string) => api.delete(`/knowledge-bases/${id}`),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: queryKeys.knowledgeBases.all })
      setKbToDelete(null)
      toast({
        title: tCommon('success'),
        description: t('deleteKnowledge'),
      })
    },
    onError: () => {
      toast({
        title: tCommon('error'),
        variant: 'destructive',
      })
    },
  })

  const regenerateMutation = useMutation({
    mutationFn: (id: string) => api.post(`/knowledge-bases/${id}/regenerate-embeddings`),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: queryKeys.knowledgeBases.all })
    },
  })

  const handleEdit = (kb: KnowledgeBase) => {
    setEditingKb(kb)
    setIsFormOpen(true)
  }

  const handleDelete = (kb: KnowledgeBase) => {
    setKbToDelete(kb)
  }

  const handleRegenerate = async (kb: KnowledgeBase) => {
    await regenerateMutation.mutateAsync(kb.id)
  }

  const handleFormClose = () => {
    setIsFormOpen(false)
    setEditingKb(null)
  }

  return (
    <div className="flex h-full flex-col">
      <Header title={t('title')} />

      <div className="flex-1 overflow-auto p-6">
        {/* Toolbar */}
        <div className="mb-6 flex flex-col gap-4 sm:flex-row sm:items-center sm:justify-between">
          <div className="flex flex-1 items-center gap-4">
            {/* Search */}
            <div className="relative max-w-sm flex-1">
              <Search className="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground" />
              <Input
                placeholder={t('searchPlaceholder')}
                value={searchQuery}
                onChange={(e) => setSearchQuery(e.target.value)}
                className="pl-9"
              />
            </div>

            {/* Type Filter */}
            <Select value={typeFilter} onValueChange={(v) => setTypeFilter(v as typeof typeFilter)}>
              <SelectTrigger className="w-[140px]">
                <SelectValue placeholder={tCommon('type')} />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="all">{t('allTypes')}</SelectItem>
                <SelectItem value="faq">{t('faq')}</SelectItem>
                <SelectItem value="documents">{t('documents')}</SelectItem>
                <SelectItem value="website">{t('website')}</SelectItem>
              </SelectContent>
            </Select>

            {/* Status Filter */}
            <Select value={statusFilter} onValueChange={(v) => setStatusFilter(v as typeof statusFilter)}>
              <SelectTrigger className="w-[140px]">
                <SelectValue placeholder={tCommon('status')} />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="all">{t('allStatus')}</SelectItem>
                <SelectItem value="active">{t('active')}</SelectItem>
                <SelectItem value="syncing">{t('syncing')}</SelectItem>
                <SelectItem value="error">{t('error')}</SelectItem>
              </SelectContent>
            </Select>
          </div>

          {/* Actions */}
          <div className="flex items-center gap-2">
            <Button
              variant="outline"
              size="icon"
              onClick={() => refetch()}
              disabled={isFetching}
            >
              <RefreshCw className={cn("h-4 w-4", isFetching && "animate-spin")} />
            </Button>
            <Button onClick={() => setIsFormOpen(true)}>
              <Plus className="mr-2 h-4 w-4" />
              {t('newKnowledgeBase')}
            </Button>
          </div>
        </div>

        {/* Content */}
        {isLoading ? (
          <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-3">
            {Array.from({ length: 6 }).map((_, i) => (
              <div key={i} className="rounded-lg border border-border bg-card p-4">
                <Skeleton className="mb-4 h-10 w-10 rounded-lg" />
                <Skeleton className="mb-2 h-5 w-3/4" />
                <Skeleton className="mb-4 h-4 w-full" />
                <div className="flex gap-2">
                  <Skeleton className="h-5 w-16" />
                  <Skeleton className="h-5 w-20" />
                </div>
              </div>
            ))}
          </div>
        ) : knowledgeBases.length === 0 ? (
          <div className="flex flex-col items-center justify-center py-12 text-center">
            <BookOpen className="mx-auto mb-4 h-12 w-12 text-muted-foreground/50" />
            <h3 className="mb-2 text-lg font-medium">{t('noKnowledgeBases')}</h3>
            <p className="mb-4 text-sm text-muted-foreground">
              {t('noKnowledgeBasesDesc')}
            </p>
            <Button onClick={() => setIsFormOpen(true)}>
              <Plus className="mr-2 h-4 w-4" />
              {t('createKnowledgeBase')}
            </Button>
          </div>
        ) : (
          <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-3">
            {knowledgeBases.map((kb) => (
              <KnowledgeBaseCard
                key={kb.id}
                knowledgeBase={kb}
                onEdit={() => handleEdit(kb)}
                onDelete={() => handleDelete(kb)}
                onRegenerate={() => handleRegenerate(kb)}
                isRegenerating={regenerateMutation.isPending && regenerateMutation.variables === kb.id}
              />
            ))}
          </div>
        )}
      </div>

      {/* Form Dialog */}
      <KnowledgeBaseForm
        open={isFormOpen}
        onOpenChange={handleFormClose}
        knowledgeBase={editingKb}
      />

      {/* Delete Confirmation Dialog */}
      <AlertDialog open={!!kbToDelete} onOpenChange={(open) => !open && setKbToDelete(null)}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>{t('deleteKnowledge')}</AlertDialogTitle>
            <AlertDialogDescription>
              {t('confirmDelete', { name: kbToDelete?.name || '' })}
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel>{tCommon('cancel')}</AlertDialogCancel>
            <AlertDialogAction
              onClick={() => kbToDelete && deleteMutation.mutate(kbToDelete.id)}
              className="bg-destructive text-destructive-foreground hover:bg-destructive/90"
            >
              {tCommon('delete')}
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </div>
  )
}
