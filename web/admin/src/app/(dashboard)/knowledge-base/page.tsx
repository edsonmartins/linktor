'use client'

import { useState } from 'react'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
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
import { api } from '@/lib/api'
import { queryKeys } from '@/lib/query'
import type { KnowledgeBase, KnowledgeBaseType, KnowledgeBaseStatus } from '@/types'
import { KnowledgeBaseCard } from './components/knowledge-base-card'
import { KnowledgeBaseForm } from './components/knowledge-base-form'

export default function KnowledgeBasePage() {
  const [searchQuery, setSearchQuery] = useState('')
  const [typeFilter, setTypeFilter] = useState<KnowledgeBaseType | 'all'>('all')
  const [statusFilter, setStatusFilter] = useState<KnowledgeBaseStatus | 'all'>('all')
  const [isFormOpen, setIsFormOpen] = useState(false)
  const [editingKb, setEditingKb] = useState<KnowledgeBase | null>(null)

  const queryClient = useQueryClient()

  const { data, isLoading } = useQuery({
    queryKey: queryKeys.knowledgeBases.list({ search: searchQuery, type: typeFilter, status: statusFilter }),
    queryFn: () =>
      api.get<{ data: KnowledgeBase[]; total: number }>('/knowledge-bases', {
        ...(searchQuery && { search: searchQuery }),
        ...(typeFilter !== 'all' && { type: typeFilter }),
        ...(statusFilter !== 'all' && { status: statusFilter }),
      }),
  })

  const deleteMutation = useMutation({
    mutationFn: (id: string) => api.delete(`/knowledge-bases/${id}`),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: queryKeys.knowledgeBases.all })
    },
  })

  const regenerateMutation = useMutation({
    mutationFn: (id: string) => api.post(`/knowledge-bases/${id}/regenerate-embeddings`),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: queryKeys.knowledgeBases.all })
    },
  })

  const knowledgeBases = data?.data || []

  const handleEdit = (kb: KnowledgeBase) => {
    setEditingKb(kb)
    setIsFormOpen(true)
  }

  const handleDelete = async (kb: KnowledgeBase) => {
    if (confirm(`Are you sure you want to delete "${kb.name}"? This will also delete all items in this knowledge base.`)) {
      await deleteMutation.mutateAsync(kb.id)
    }
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
      <Header title="Knowledge Base" />

      <div className="flex-1 overflow-auto p-6">
        {/* Toolbar */}
        <div className="mb-6 flex flex-col gap-4 sm:flex-row sm:items-center sm:justify-between">
          <div className="flex flex-1 items-center gap-4">
            {/* Search */}
            <div className="relative max-w-sm flex-1">
              <Search className="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground" />
              <Input
                placeholder="Search knowledge bases..."
                value={searchQuery}
                onChange={(e) => setSearchQuery(e.target.value)}
                className="pl-9"
              />
            </div>

            {/* Type Filter */}
            <Select value={typeFilter} onValueChange={(v) => setTypeFilter(v as typeof typeFilter)}>
              <SelectTrigger className="w-[140px]">
                <SelectValue placeholder="Type" />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="all">All Types</SelectItem>
                <SelectItem value="faq">FAQ</SelectItem>
                <SelectItem value="documents">Documents</SelectItem>
                <SelectItem value="website">Website</SelectItem>
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
                <SelectItem value="syncing">Syncing</SelectItem>
                <SelectItem value="error">Error</SelectItem>
              </SelectContent>
            </Select>
          </div>

          {/* Add Button */}
          <Button onClick={() => setIsFormOpen(true)}>
            <Plus className="mr-2 h-4 w-4" />
            New Knowledge Base
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
        ) : knowledgeBases.length === 0 ? (
          <div className="flex flex-col items-center justify-center py-12 text-center">
            <BookOpen className="mx-auto mb-4 h-12 w-12 text-muted-foreground/50" />
            <h3 className="mb-2 text-lg font-medium">No knowledge bases yet</h3>
            <p className="mb-4 text-sm text-muted-foreground">
              Create your first knowledge base to power AI-assisted responses.
            </p>
            <Button onClick={() => setIsFormOpen(true)}>
              <Plus className="mr-2 h-4 w-4" />
              Create Knowledge Base
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
    </div>
  )
}
