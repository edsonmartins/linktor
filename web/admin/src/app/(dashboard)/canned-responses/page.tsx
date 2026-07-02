'use client'

import { useState } from 'react'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { useTranslations } from 'next-intl'
import { Plus, Search, MessageSquareText, Pencil, Trash2 } from 'lucide-react'
import { Header } from '@/components/layout/header'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Badge } from '@/components/ui/badge'
import { Textarea } from '@/components/ui/textarea'
import { Skeleton } from '@/components/ui/skeleton'
import {
  Dialog,
  DialogContent,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
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
import type { CannedRequest, CannedResponse } from '@/types'

export default function CannedResponsesPage() {
  const t = useTranslations('canned')
  const { toast } = useToast()
  const queryClient = useQueryClient()

  const [search, setSearch] = useState('')
  const [editing, setEditing] = useState<CannedResponse | null>(null)
  const [creating, setCreating] = useState(false)
  const [deleting, setDeleting] = useState<CannedResponse | null>(null)

  const { data: items, isLoading } = useQuery({
    queryKey: queryKeys.canned.list({ search }),
    queryFn: () => api.get<CannedResponse[]>('/canned-responses', search ? { search } : undefined),
  })

  const invalidate = () => queryClient.invalidateQueries({ queryKey: queryKeys.canned.all })
  const onErr = (e: Error) =>
    toast({ title: 'Error', description: e.message, variant: 'error' })

  const deleteMutation = useMutation({
    mutationFn: (id: string) => api.delete(`/canned-responses/${id}`),
    onSuccess: () => {
      toast({ title: t('delete') })
      setDeleting(null)
      invalidate()
    },
    onError: onErr,
  })

  return (
    <>
      <Header title={t('title')}>
        <Button size="sm" onClick={() => setCreating(true)}>
          <Plus className="mr-2 h-4 w-4" />
          {t('new')}
        </Button>
      </Header>

      <div className="p-6">
        <p className="mb-4 text-sm text-muted-foreground">{t('subtitle')}</p>

        <div className="relative mb-4 max-w-sm">
          <Search className="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground" />
          <Input
            value={search}
            onChange={(e) => setSearch(e.target.value)}
            placeholder={t('search')}
            className="pl-9"
          />
        </div>

        {isLoading ? (
          <Skeleton className="h-64 w-full" />
        ) : !items || items.length === 0 ? (
          <div className="flex flex-col items-center justify-center rounded-lg border border-dashed py-16 text-center">
            <MessageSquareText className="mb-3 h-10 w-10 text-muted-foreground" />
            <p className="text-sm text-muted-foreground">{t('empty')}</p>
          </div>
        ) : (
          <div className="rounded-lg border">
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>{t('shortcut')}</TableHead>
                  <TableHead>{t('replyTitle')}</TableHead>
                  <TableHead>{t('tags')}</TableHead>
                  <TableHead className="text-right">{t('usage')}</TableHead>
                  <TableHead className="w-24" />
                </TableRow>
              </TableHeader>
              <TableBody>
                {items.map((c) => (
                  <TableRow key={c.id}>
                    <TableCell className="font-mono text-sm">/{c.shortcut}</TableCell>
                    <TableCell className="font-medium">{c.title}</TableCell>
                    <TableCell>
                      <div className="flex flex-wrap gap-1">
                        {(c.tags ?? []).map((tag) => (
                          <Badge key={tag} variant="outline" className="text-xs">
                            {tag}
                          </Badge>
                        ))}
                      </div>
                    </TableCell>
                    <TableCell className="text-right tabular-nums text-muted-foreground">
                      {c.usage_count}
                    </TableCell>
                    <TableCell>
                      <div className="flex justify-end gap-1">
                        <Button variant="ghost" size="icon" onClick={() => setEditing(c)}>
                          <Pencil className="h-4 w-4" />
                        </Button>
                        <Button variant="ghost" size="icon" onClick={() => setDeleting(c)}>
                          <Trash2 className="h-4 w-4 text-red-600" />
                        </Button>
                      </div>
                    </TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          </div>
        )}
      </div>

      {(creating || editing) && (
        <CannedDialog
          item={editing}
          onClose={() => {
            setCreating(false)
            setEditing(null)
          }}
          onSaved={() => {
            invalidate()
            setCreating(false)
            setEditing(null)
          }}
        />
      )}

      <AlertDialog open={!!deleting} onOpenChange={(o) => !o && setDeleting(null)}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>{t('deleteConfirm')}</AlertDialogTitle>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel>{t('cancel')}</AlertDialogCancel>
            <AlertDialogAction
              onClick={() => deleting && deleteMutation.mutate(deleting.id)}
              className="bg-destructive text-destructive-foreground hover:bg-destructive/90"
            >
              {t('delete')}
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </>
  )
}

function CannedDialog({
  item,
  onClose,
  onSaved,
}: {
  item: CannedResponse | null
  onClose: () => void
  onSaved: () => void
}) {
  const t = useTranslations('canned')
  const { toast } = useToast()

  const [shortcut, setShortcut] = useState(item?.shortcut ?? '')
  const [title, setTitle] = useState(item?.title ?? '')
  const [content, setContent] = useState(item?.content ?? '')
  const [tags, setTags] = useState((item?.tags ?? []).join(', '))

  const saveMutation = useMutation({
    mutationFn: (body: CannedRequest) =>
      item
        ? api.put<CannedResponse>(`/canned-responses/${item.id}`, body)
        : api.post<CannedResponse>('/canned-responses', body),
    onSuccess: onSaved,
    onError: (e: Error) => toast({ title: 'Error', description: e.message, variant: 'error' }),
  })

  const canSubmit = shortcut.trim() && content.trim()

  function submit() {
    if (!canSubmit) return
    saveMutation.mutate({
      shortcut: shortcut.trim(),
      title: title.trim(),
      content,
      tags: tags
        .split(',')
        .map((x) => x.trim())
        .filter(Boolean),
    })
  }

  return (
    <Dialog open onOpenChange={(o) => !o && onClose()}>
      <DialogContent className="max-w-xl">
        <DialogHeader>
          <DialogTitle>{item ? t('editTitle') : t('createTitle')}</DialogTitle>
        </DialogHeader>

        <div className="space-y-4">
          <div className="grid grid-cols-2 gap-4">
            <div className="space-y-2">
              <Label>{t('shortcut')}</Label>
              <Input
                value={shortcut}
                onChange={(e) => setShortcut(e.target.value)}
                placeholder="greeting"
              />
            </div>
            <div className="space-y-2">
              <Label>{t('replyTitle')}</Label>
              <Input value={title} onChange={(e) => setTitle(e.target.value)} />
            </div>
          </div>
          <div className="space-y-2">
            <Label>{t('content')}</Label>
            <Textarea rows={5} value={content} onChange={(e) => setContent(e.target.value)} />
            <p className="text-xs text-muted-foreground">{t('contentHelp')}</p>
          </div>
          <div className="space-y-2">
            <Label>{t('tags')}</Label>
            <Input value={tags} onChange={(e) => setTags(e.target.value)} placeholder="sales, support" />
            <p className="text-xs text-muted-foreground">{t('tagsHelp')}</p>
          </div>
        </div>

        <DialogFooter>
          <Button variant="outline" onClick={onClose}>
            {t('cancel')}
          </Button>
          <Button onClick={submit} disabled={!canSubmit || saveMutation.isPending}>
            {item ? t('save') : t('create')}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}
