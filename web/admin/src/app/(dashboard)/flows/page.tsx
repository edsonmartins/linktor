'use client'

import { useState } from 'react'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { Plus, Search, GitBranch, Play, Pause, Trash2, Edit, TestTube } from 'lucide-react'
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
import { api } from '@/lib/api'
import { queryKeys } from '@/lib/query'
import type { Flow, FlowTriggerType } from '@/types'
import { FlowCard } from './components/flow-card'
import { CreateFlowDialog } from './components/create-flow-dialog'

export default function FlowsPage() {
  const router = useRouter()
  const [searchQuery, setSearchQuery] = useState('')
  const [triggerFilter, setTriggerFilter] = useState<FlowTriggerType | 'all'>('all')
  const [statusFilter, setStatusFilter] = useState<'all' | 'active' | 'inactive'>('all')
  const [isCreateOpen, setIsCreateOpen] = useState(false)

  const queryClient = useQueryClient()

  const { data, isLoading } = useQuery({
    queryKey: queryKeys.flows.list({ search: searchQuery, trigger: triggerFilter, status: statusFilter }),
    queryFn: () =>
      api.get<{ data: Flow[]; total: number }>('/flows', {
        ...(searchQuery && { search: searchQuery }),
        ...(triggerFilter !== 'all' && { trigger: triggerFilter }),
        ...(statusFilter !== 'all' && { is_active: statusFilter === 'active' ? 'true' : 'false' }),
      }),
  })

  const activateMutation = useMutation({
    mutationFn: (id: string) => api.post(`/flows/${id}/activate`),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: queryKeys.flows.all })
    },
  })

  const deactivateMutation = useMutation({
    mutationFn: (id: string) => api.post(`/flows/${id}/deactivate`),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: queryKeys.flows.all })
    },
  })

  const deleteMutation = useMutation({
    mutationFn: (id: string) => api.delete(`/flows/${id}`),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: queryKeys.flows.all })
    },
  })

  const flows = data?.data || []

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

  const handleDelete = async (flow: Flow) => {
    if (confirm(`Are you sure you want to delete "${flow.name}"?`)) {
      await deleteMutation.mutateAsync(flow.id)
    }
  }

  const handleTest = (flow: Flow) => {
    router.push(`/flows/${flow.id}?test=true`)
  }

  return (
    <div className="flex h-full flex-col">
      <Header title="Conversational Flows" />

      <div className="flex-1 overflow-auto p-6">
        {/* Toolbar */}
        <div className="mb-6 flex flex-col gap-4 sm:flex-row sm:items-center sm:justify-between">
          <div className="flex flex-1 items-center gap-4">
            {/* Search */}
            <div className="relative max-w-sm flex-1">
              <Search className="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground" />
              <Input
                placeholder="Search flows..."
                value={searchQuery}
                onChange={(e) => setSearchQuery(e.target.value)}
                className="pl-9"
              />
            </div>

            {/* Trigger Filter */}
            <Select value={triggerFilter} onValueChange={(v) => setTriggerFilter(v as typeof triggerFilter)}>
              <SelectTrigger className="w-[140px]">
                <SelectValue placeholder="Trigger" />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="all">All Triggers</SelectItem>
                <SelectItem value="welcome">Welcome</SelectItem>
                <SelectItem value="keyword">Keyword</SelectItem>
                <SelectItem value="intent">Intent</SelectItem>
                <SelectItem value="manual">Manual</SelectItem>
              </SelectContent>
            </Select>

            {/* Status Filter */}
            <Select value={statusFilter} onValueChange={(v) => setStatusFilter(v as typeof statusFilter)}>
              <SelectTrigger className="w-[140px]">
                <SelectValue placeholder="Status" />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="all">All Status</SelectItem>
                <SelectItem value="active">Active</SelectItem>
                <SelectItem value="inactive">Inactive</SelectItem>
              </SelectContent>
            </Select>
          </div>

          {/* Add Button */}
          <Button onClick={() => setIsCreateOpen(true)}>
            <Plus className="mr-2 h-4 w-4" />
            New Flow
          </Button>
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
            <h3 className="mb-2 text-lg font-medium">No flows yet</h3>
            <p className="mb-4 text-sm text-muted-foreground">
              Create conversational flows to guide your customers through decision trees.
            </p>
            <Button onClick={() => setIsCreateOpen(true)}>
              <Plus className="mr-2 h-4 w-4" />
              Create Flow
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
    </div>
  )
}
