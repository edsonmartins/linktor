'use client'

import { useState } from 'react'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { useTranslations } from 'next-intl'
import { FlaskConical, Plus, Trash2, AlertTriangle } from 'lucide-react'
import { Header } from '@/components/layout/header'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Badge } from '@/components/ui/badge'
import { Skeleton } from '@/components/ui/skeleton'
import { Alert, AlertDescription } from '@/components/ui/alert'
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'
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
import { api } from '@/lib/api'
import { queryKeys } from '@/lib/query'
import { formatDate } from '@/lib/utils'
import { toastSuccess, toastError } from '@/hooks/use-toast'
import type { Channel, SandboxAllowlistEntry } from '@/types'

export default function SandboxAllowlistPage() {
  const t = useTranslations('sandboxAllowlist')
  const queryClient = useQueryClient()

  const [recipient, setRecipient] = useState('')
  const [note, setNote] = useState('')
  const [channelId, setChannelId] = useState('')
  const [toRemove, setToRemove] = useState<SandboxAllowlistEntry | null>(null)

  // The list read decides "no allowlist configured" vs "empty list": the
  // backend returns [] for a tenant with no entries, which is a distinct
  // operator state from a load error (both would otherwise block every send).
  const { data, isLoading, isError } = useQuery({
    queryKey: queryKeys.sandboxAllowlist.list(),
    queryFn: () => api.getEnvelope<SandboxAllowlistEntry[]>('/sandbox/allowlist'),
    meta: { skipGlobalError: true },
  })
  const entries = data?.data ?? []

  // Only the tenant's sandbox channels can be an opt-in scope target.
  const { data: sandboxChannels } = useQuery({
    queryKey: [...queryKeys.channels.all, 'sandbox'] as const,
    queryFn: async () => {
      const all = await api.get<Channel[]>('/channels')
      return all.filter((c) => c.environment === 'sandbox')
    },
  })

  const addMutation = useMutation({
    mutationFn: (payload: { recipient: string; note?: string; channel_id?: string }) =>
      api.post<SandboxAllowlistEntry>('/sandbox/allowlist', payload),
    onSuccess: (entry) => {
      // Show the backend-normalized value, not what was typed (INV-017).
      toastSuccess(t('added'), t('addedDesc', { recipient: entry.recipient }))
      setRecipient('')
      setNote('')
      setChannelId('')
      queryClient.invalidateQueries({ queryKey: queryKeys.sandboxAllowlist.all })
    },
    onError: (err: Error) => toastError(t('added'), err.message),
    meta: { skipGlobalError: true },
  })

  const removeMutation = useMutation({
    mutationFn: (id: string) => api.delete(`/sandbox/allowlist/${id}`),
    onSuccess: (_res, _id) => {
      toastSuccess(t('removed'), t('removedDesc', { recipient: toRemove?.recipient ?? '' }))
      setToRemove(null)
      queryClient.invalidateQueries({ queryKey: queryKeys.sandboxAllowlist.all })
    },
    onError: (err: Error) => toastError(t('removed'), err.message),
    meta: { skipGlobalError: true },
  })

  const onAdd = () => {
    if (!recipient.trim()) return
    addMutation.mutate({
      recipient: recipient.trim(),
      ...(note.trim() ? { note: note.trim() } : {}),
      ...(channelId ? { channel_id: channelId } : {}),
    })
  }

  const channelName = (id?: string) =>
    (id && sandboxChannels?.find((c) => c.id === id)?.name) || id

  return (
    <div className="flex h-full flex-col">
      <Header title={t('title')} />
      <div className="flex-1 space-y-4 overflow-auto p-4">
        <p className="text-sm text-muted-foreground">{t('subtitle')}</p>

        {/* Security framing: the allowlist is a safety boundary, not config. */}
        <Alert>
          <FlaskConical className="h-4 w-4" />
          <AlertDescription>{t('securityNote')}</AlertDescription>
        </Alert>

        {/* Add form */}
        <div className="flex flex-wrap items-end gap-3 rounded-lg border border-border p-4">
          <div className="flex-1 min-w-[200px]">
            <label className="mb-1 block text-xs font-medium" htmlFor="sb-recipient">
              {t('recipient')}
            </label>
            <Input
              id="sb-recipient"
              placeholder={t('recipientPlaceholder')}
              value={recipient}
              onChange={(e) => setRecipient(e.target.value)}
              onKeyDown={(e) => e.key === 'Enter' && onAdd()}
            />
            <p className="mt-1 text-xs text-muted-foreground">{t('recipientHint')}</p>
          </div>
          <div className="min-w-[200px]">
            <label className="mb-1 block text-xs font-medium" htmlFor="sb-scope">
              {t('channelScope')}
            </label>
            <select
              id="sb-scope"
              className="h-9 w-full rounded-md border border-input bg-background px-3 text-sm"
              value={channelId}
              onChange={(e) => setChannelId(e.target.value)}
            >
              <option value="">{t('channelScopeAll')}</option>
              {sandboxChannels?.map((c) => (
                <option key={c.id} value={c.id}>
                  {c.name}
                </option>
              ))}
            </select>
          </div>
          <div className="flex-1 min-w-[200px]">
            <label className="mb-1 block text-xs font-medium" htmlFor="sb-note">
              {t('note')}
            </label>
            <Input
              id="sb-note"
              placeholder={t('notePlaceholder')}
              value={note}
              onChange={(e) => setNote(e.target.value)}
              onKeyDown={(e) => e.key === 'Enter' && onAdd()}
            />
          </div>
          <Button onClick={onAdd} disabled={!recipient.trim() || addMutation.isPending}>
            <Plus className="mr-2 h-4 w-4" />
            {t('add')}
          </Button>
        </div>

        {/* List */}
        {isLoading ? (
          <div className="space-y-2">
            {Array.from({ length: 3 }).map((_, i) => (
              <Skeleton key={i} className="h-10 w-full" />
            ))}
          </div>
        ) : isError ? (
          <Alert variant="destructive">
            <AlertTriangle className="h-4 w-4" />
            <AlertDescription>{t('loadError')}</AlertDescription>
          </Alert>
        ) : entries.length === 0 ? (
          // "No allowlist configured" is distinct from a load error: the read
          // succeeded and returned an empty list. Both block every send, but
          // the operator action differs — so say it explicitly.
          <div className="rounded-lg border border-dashed border-border py-12 text-center">
            <FlaskConical className="mx-auto h-8 w-8 text-muted-foreground opacity-50" />
            <p className="mt-2 text-sm font-medium">{t('empty')}</p>
            <p className="mt-1 text-xs text-muted-foreground">{t('emptyHint')}</p>
          </div>
        ) : (
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>{t('recipient')}</TableHead>
                <TableHead>{t('channelScope')}</TableHead>
                <TableHead>{t('note')}</TableHead>
                <TableHead>{t('addedAt')}</TableHead>
                <TableHead className="text-right">{t('actions')}</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {entries.map((entry) => (
                <TableRow key={entry.id}>
                  <TableCell className="font-mono text-sm">{entry.recipient}</TableCell>
                  <TableCell>
                    {entry.channel_id ? (
                      <Badge variant="outline">{channelName(entry.channel_id)}</Badge>
                    ) : (
                      <span className="text-xs text-muted-foreground">
                        {t('channelScopeAll')}
                      </span>
                    )}
                  </TableCell>
                  <TableCell className="text-sm text-muted-foreground">
                    {entry.note || '—'}
                  </TableCell>
                  <TableCell className="text-sm text-muted-foreground">
                    {formatDate(entry.created_at)}
                  </TableCell>
                  <TableCell className="text-right">
                    <Button
                      variant="ghost"
                      size="icon"
                      className="text-destructive"
                      onClick={() => setToRemove(entry)}
                      aria-label={t('remove')}
                    >
                      <Trash2 className="h-4 w-4" />
                    </Button>
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        )}
      </div>

      {/* Removal confirmation makes the immediate-effect explicit: this is a
          security operation, not a config tweak. */}
      <AlertDialog open={!!toRemove} onOpenChange={(open) => !open && setToRemove(null)}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>{t('removeTitle')}</AlertDialogTitle>
            <AlertDialogDescription>
              {t('removeConfirm', { recipient: toRemove?.recipient ?? '' })}
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel>{t('cancel')}</AlertDialogCancel>
            <AlertDialogAction
              className="bg-destructive text-destructive-foreground hover:bg-destructive/90"
              onClick={() => toRemove && removeMutation.mutate(toRemove.id)}
            >
              {t('removeAction')}
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </div>
  )
}
