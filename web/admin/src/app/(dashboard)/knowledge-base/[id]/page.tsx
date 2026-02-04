'use client'

import { useState } from 'react'
import { useParams, useRouter } from 'next/navigation'
import Link from 'next/link'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import {
  ArrowLeft,
  Plus,
  Upload,
  Search,
  Settings,
  RefreshCw,
  BookOpen,
  FileText,
  Globe,
  CheckCircle,
  AlertCircle,
} from 'lucide-react'
import { Header } from '@/components/layout/header'
import { Button } from '@/components/ui/button'
import { Badge } from '@/components/ui/badge'
import { Skeleton } from '@/components/ui/skeleton'
import { Card, CardContent } from '@/components/ui/card'
import { api } from '@/lib/api'
import { queryKeys } from '@/lib/query'
import { formatDistanceToNow } from '@/lib/utils'
import type { KnowledgeBase, KnowledgeItem } from '@/types'
import { KnowledgeItemTable } from '../components/knowledge-item-table'
import { KnowledgeItemForm } from '../components/knowledge-item-form'
import { BulkImportDialog } from '../components/bulk-import-dialog'

const typeIcons = {
  faq: BookOpen,
  documents: FileText,
  website: Globe,
}

const statusVariants: Record<string, 'default' | 'success' | 'warning' | 'error'> = {
  active: 'success',
  syncing: 'warning',
  error: 'error',
}

export default function KnowledgeBaseDetailPage() {
  const params = useParams()
  const router = useRouter()
  const queryClient = useQueryClient()
  const id = params.id as string

  const [isItemFormOpen, setIsItemFormOpen] = useState(false)
  const [editingItem, setEditingItem] = useState<KnowledgeItem | null>(null)
  const [isBulkImportOpen, setIsBulkImportOpen] = useState(false)
  const [page, setPage] = useState(1)
  const [searchQuery, setSearchQuery] = useState('')

  // Fetch knowledge base details
  const { data: kbData, isLoading: isLoadingKb } = useQuery({
    queryKey: queryKeys.knowledgeBases.detail(id),
    queryFn: () => api.get<KnowledgeBase>(`/knowledge-bases/${id}`),
  })

  // Fetch items
  const { data: itemsData, isLoading: isLoadingItems } = useQuery({
    queryKey: queryKeys.knowledgeItems.list(id, { page, search: searchQuery }),
    queryFn: () =>
      api.get<{ data: KnowledgeItem[]; total: number }>(`/knowledge-bases/${id}/items`, {
        page: String(page),
        page_size: '20',
        ...(searchQuery && { search: searchQuery }),
      }),
  })

  // Regenerate embeddings mutation
  const regenerateMutation = useMutation({
    mutationFn: () => api.post(`/knowledge-bases/${id}/regenerate-embeddings`),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: queryKeys.knowledgeBases.detail(id) })
    },
  })

  // Delete item mutation
  const deleteItemMutation = useMutation({
    mutationFn: (itemId: string) => api.delete(`/knowledge-bases/${id}/items/${itemId}`),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: queryKeys.knowledgeItems.list(id, {}) })
      queryClient.invalidateQueries({ queryKey: queryKeys.knowledgeBases.detail(id) })
    },
  })

  const kb = kbData
  const items = itemsData?.data || []
  const totalItems = itemsData?.total || 0
  const totalPages = Math.ceil(totalItems / 20)

  const Icon = kb ? typeIcons[kb.type] : BookOpen
  const itemsWithEmbedding = items.filter((item) => item.has_embedding).length

  const handleEditItem = (item: KnowledgeItem) => {
    setEditingItem(item)
    setIsItemFormOpen(true)
  }

  const handleDeleteItem = async (item: KnowledgeItem) => {
    if (confirm(`Are you sure you want to delete this item?`)) {
      await deleteItemMutation.mutateAsync(item.id)
    }
  }

  const handleItemFormClose = () => {
    setIsItemFormOpen(false)
    setEditingItem(null)
  }

  if (isLoadingKb) {
    return (
      <div className="flex h-full flex-col">
        <Header title="Knowledge Base" />
        <div className="flex-1 p-6">
          <Skeleton className="mb-4 h-8 w-64" />
          <Skeleton className="mb-6 h-24 w-full" />
          <Skeleton className="h-96 w-full" />
        </div>
      </div>
    )
  }

  if (!kb) {
    return (
      <div className="flex h-full flex-col">
        <Header title="Knowledge Base" />
        <div className="flex flex-1 items-center justify-center">
          <div className="text-center">
            <AlertCircle className="mx-auto mb-4 h-12 w-12 text-muted-foreground" />
            <h2 className="mb-2 text-lg font-medium">Knowledge base not found</h2>
            <p className="mb-4 text-sm text-muted-foreground">
              The knowledge base you&apos;re looking for doesn&apos;t exist.
            </p>
            <Button onClick={() => router.push('/knowledge-base')}>
              <ArrowLeft className="mr-2 h-4 w-4" />
              Back to Knowledge Bases
            </Button>
          </div>
        </div>
      </div>
    )
  }

  return (
    <div className="flex h-full flex-col">
      <Header title="Knowledge Base" />

      <div className="flex-1 overflow-auto p-6">
        {/* Breadcrumb & Title */}
        <div className="mb-6">
          <Link
            href="/knowledge-base"
            className="mb-2 inline-flex items-center text-sm text-muted-foreground hover:text-foreground"
          >
            <ArrowLeft className="mr-1 h-4 w-4" />
            Back to Knowledge Bases
          </Link>

          <div className="flex items-start justify-between">
            <div className="flex items-center gap-4">
              <div className="flex h-12 w-12 items-center justify-center rounded-lg bg-primary/10">
                <Icon className="h-6 w-6 text-primary" />
              </div>
              <div>
                <h1 className="text-2xl font-bold">{kb.name}</h1>
                <div className="mt-1 flex items-center gap-2">
                  <Badge variant="secondary">{kb.type.toUpperCase()}</Badge>
                  <Badge variant={statusVariants[kb.status]}>{kb.status}</Badge>
                </div>
              </div>
            </div>

            <div className="flex items-center gap-2">
              <Button variant="outline" asChild>
                <Link href={`/knowledge-base/${id}/search`}>
                  <Search className="mr-2 h-4 w-4" />
                  Test Search
                </Link>
              </Button>
              <Button
                variant="outline"
                onClick={() => regenerateMutation.mutate()}
                disabled={regenerateMutation.isPending}
              >
                <RefreshCw
                  className={`mr-2 h-4 w-4 ${regenerateMutation.isPending ? 'animate-spin' : ''}`}
                />
                Regenerate Embeddings
              </Button>
            </div>
          </div>

          {kb.description && (
            <p className="mt-4 text-muted-foreground">{kb.description}</p>
          )}
        </div>

        {/* Stats */}
        <div className="mb-6 grid gap-4 md:grid-cols-4">
          <Card>
            <CardContent className="p-4">
              <p className="text-sm text-muted-foreground">Total Items</p>
              <p className="text-2xl font-bold">{kb.item_count}</p>
            </CardContent>
          </Card>
          <Card>
            <CardContent className="p-4">
              <p className="text-sm text-muted-foreground">With Embeddings</p>
              <p className="text-2xl font-bold flex items-center gap-2">
                {itemsWithEmbedding}
                {itemsWithEmbedding === items.length && items.length > 0 && (
                  <CheckCircle className="h-5 w-5 text-success" />
                )}
              </p>
            </CardContent>
          </Card>
          <Card>
            <CardContent className="p-4">
              <p className="text-sm text-muted-foreground">Last Synced</p>
              <p className="text-2xl font-bold">
                {kb.last_sync_at ? formatDistanceToNow(kb.last_sync_at) : 'Never'}
              </p>
            </CardContent>
          </Card>
          <Card>
            <CardContent className="p-4">
              <p className="text-sm text-muted-foreground">Created</p>
              <p className="text-2xl font-bold">{formatDistanceToNow(kb.created_at)}</p>
            </CardContent>
          </Card>
        </div>

        {/* Actions Toolbar */}
        <div className="mb-4 flex items-center justify-between">
          <h2 className="text-lg font-semibold">Items</h2>
          <div className="flex items-center gap-2">
            <Button variant="outline" onClick={() => setIsBulkImportOpen(true)}>
              <Upload className="mr-2 h-4 w-4" />
              Bulk Import
            </Button>
            <Button onClick={() => setIsItemFormOpen(true)}>
              <Plus className="mr-2 h-4 w-4" />
              Add Item
            </Button>
          </div>
        </div>

        {/* Items Table */}
        <KnowledgeItemTable
          items={items}
          isLoading={isLoadingItems}
          searchQuery={searchQuery}
          onSearchChange={setSearchQuery}
          page={page}
          totalPages={totalPages}
          onPageChange={setPage}
          onEdit={handleEditItem}
          onDelete={handleDeleteItem}
        />
      </div>

      {/* Item Form Dialog */}
      <KnowledgeItemForm
        open={isItemFormOpen}
        onOpenChange={handleItemFormClose}
        knowledgeBaseId={id}
        item={editingItem}
      />

      {/* Bulk Import Dialog */}
      <BulkImportDialog
        open={isBulkImportOpen}
        onOpenChange={setIsBulkImportOpen}
        knowledgeBaseId={id}
      />
    </div>
  )
}
