'use client'

import { useState } from 'react'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { useTranslations } from 'next-intl'
import { Plus, Search, GitBranch, Play, Pause, Trash2, Edit, TestTube, RefreshCw } from 'lucide-react'
import { useRouter } from 'next/navigation'
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
import type { Flow, FlowTriggerType, PaginatedResponse } from '@/types'
import { FlowCard } from './components/flow-card'
import { CreateFlowDialog } from './components/create-flow-dialog'

export default function FlowsPage() {
  const t = useTranslations('flows')
  const tCommon = useTranslations('common')
  const { toast } = useToast()
  const router = useRouter()
  const [searchQuery, setSearchQuery] = useState('')
  const [triggerFilter, setTriggerFilter] = useState<FlowTriggerType | 'all'>('all')
  const [statusFilter, setStatusFilter] = useState<'all' | 'active' | 'inactive'>('all')
  const [isCreateOpen, setIsCreateOpen] = useState(false)
  const [flowToDelete, setFlowToDelete] = useState<Flow | null>(null)

  const queryClient = useQueryClient()

  const { data, isLoading, refetch, isFetching } = useQuery({
    queryKey: queryKeys.flows.list({ search: searchQuery, trigger: triggerFilter, status: statusFilter }),
    queryFn: () =>
      api.get<PaginatedResponse<Flow>>('/flows', {
        ...(searchQuery && { search: searchQuery }),
        ...(triggerFilter !== 'all' && { trigger: triggerFilter }),
        ...(statusFilter !== 'all' && { is_active: statusFilter === 'active' ? 'true' : 'false' }),
      }),
  })

  const flows = data?.data ?? []

  const activateMutation = useMutation({
    mutationFn: (id: string) => api.post(`/flows/${id}/activate`),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: queryKeys.flows.all })
      toast({
        title: tCommon('success'),
        description: t('active'),
      })
    },
  })

  const deactivateMutation = useMutation({
    mutationFn: (id: string) => api.post(`/flows/${id}/deactivate`),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: queryKeys.flows.all })
      toast({
        title: tCommon('success'),
        description: t('inactive'),
      })
    },
  })

  const deleteMutation = useMutation({
    mutationFn: (id: string) => api.delete(`/flows/${id}`),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: queryKeys.flows.all })
      setFlowToDelete(null)
      toast({
        title: tCommon('success'),
        description: t('deleteFlow'),
      })
    },
    onError: () => {
      toast({
        title: tCommon('error'),
        variant: 'destructive',
      })
    },
  })

  const handleEdit = (flow: Flow) => {
    router.push(`/flows/${flow.id}`)
  }

  const handleToggleActive = async (flow: Flow) => {
    if (flow.is_active) {
      await deactivateMutation.mutateAsync(flow.id)
    } else {
      await activateMutation.mutateAsync(flow.id)
    }
  }

  const handleDelete = (flow: Flow) => {
    setFlowToDelete(flow)
  }

  const handleTest = (flow: Flow) => {
    router.push(`/flows/${flow.id}?test=true`)
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

            {/* Trigger Filter */}
            <Select value={triggerFilter} onValueChange={(v) => setTriggerFilter(v as typeof triggerFilter)}>
              <SelectTrigger className="w-[140px]">
                <SelectValue placeholder={t('trigger')} />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="all">{t('allTriggers')}</SelectItem>
                <SelectItem value="welcome">{t('welcome')}</SelectItem>
                <SelectItem value="keyword">{t('keyword')}</SelectItem>
                <SelectItem value="intent">{t('intent')}</SelectItem>
                <SelectItem value="manual">{t('manual')}</SelectItem>
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
                <SelectItem value="inactive">{t('inactive')}</SelectItem>
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
            <Button onClick={() => setIsCreateOpen(true)}>
              <Plus className="mr-2 h-4 w-4" />
              {t('newFlow')}
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
        ) : flows.length === 0 ? (
          <div className="flex flex-col items-center justify-center py-12 text-center">
            <GitBranch className="mx-auto mb-4 h-12 w-12 text-muted-foreground/50" />
            <h3 className="mb-2 text-lg font-medium">{t('noFlows')}</h3>
            <p className="mb-4 text-sm text-muted-foreground">
              {t('noFlowsDesc')}
            </p>
            <Button onClick={() => setIsCreateOpen(true)}>
              <Plus className="mr-2 h-4 w-4" />
              {t('createFlow')}
            </Button>
          </div>
        ) : (
          <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-3">
            {flows.map((flow) => (
              <FlowCard
                key={flow.id}
                flow={flow}
                onEdit={() => handleEdit(flow)}
                onToggleActive={() => handleToggleActive(flow)}
                onDelete={() => handleDelete(flow)}
                onTest={() => handleTest(flow)}
                isToggling={
                  (activateMutation.isPending && activateMutation.variables === flow.id) ||
                  (deactivateMutation.isPending && deactivateMutation.variables === flow.id)
                }
              />
            ))}
          </div>
        )}
      </div>

      {/* Create Dialog */}
      <CreateFlowDialog
        open={isCreateOpen}
        onOpenChange={setIsCreateOpen}
      />

      {/* Delete Confirmation Dialog */}
      <AlertDialog open={!!flowToDelete} onOpenChange={(open) => !open && setFlowToDelete(null)}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>{t('deleteFlow')}</AlertDialogTitle>
            <AlertDialogDescription>
              {t('confirmDelete', { name: flowToDelete?.name || '' })}
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel>{tCommon('cancel')}</AlertDialogCancel>
            <AlertDialogAction
              onClick={() => flowToDelete && deleteMutation.mutate(flowToDelete.id)}
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
