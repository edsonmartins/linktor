'use client'

import { useMemo, useState } from 'react'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { useTranslations } from 'next-intl'
import { Plus, ShieldCheck, Pencil, Trash2 } from 'lucide-react'
import { Header } from '@/components/layout/header'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Badge } from '@/components/ui/badge'
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
  AlertDialogDescription,
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
import type { Permission, Role, RoleCatalog, RoleRequest } from '@/types'

function permKey(resource: string, action: string) {
  return `${resource}:${action}`
}

export default function RolesPage() {
  const t = useTranslations('roles')
  const { toast } = useToast()
  const queryClient = useQueryClient()

  const { data: roles, isLoading } = useQuery({
    queryKey: queryKeys.roles.lists(),
    queryFn: () => api.get<Role[]>('/roles'),
  })
  const { data: catalog } = useQuery({
    queryKey: queryKeys.roles.catalog(),
    queryFn: () => api.get<RoleCatalog>('/roles/catalog'),
  })

  const [editing, setEditing] = useState<Role | null>(null)
  const [creating, setCreating] = useState(false)
  const [deleting, setDeleting] = useState<Role | null>(null)

  const invalidate = () =>
    queryClient.invalidateQueries({ queryKey: queryKeys.roles.lists() })
  const onErr = (e: Error) =>
    toast({ title: 'Error', description: e.message, variant: 'error' })

  const deleteMutation = useMutation({
    mutationFn: (id: string) => api.delete(`/roles/${id}`),
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
          {t('newRole')}
        </Button>
      </Header>

      <div className="p-6">
        <p className="mb-4 text-sm text-muted-foreground">{t('subtitle')}</p>

        {isLoading ? (
          <Skeleton className="h-64 w-full" />
        ) : !roles || roles.length === 0 ? (
          <div className="flex flex-col items-center justify-center rounded-lg border border-dashed py-16 text-center">
            <ShieldCheck className="mb-3 h-10 w-10 text-muted-foreground" />
            <p className="text-sm text-muted-foreground">{t('empty')}</p>
          </div>
        ) : (
          <div className="rounded-lg border">
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>{t('name')}</TableHead>
                  <TableHead>{t('description')}</TableHead>
                  <TableHead className="text-right">{t('permissions')}</TableHead>
                  <TableHead className="w-24" />
                </TableRow>
              </TableHeader>
              <TableBody>
                {roles.map((r) => (
                  <TableRow key={r.id}>
                    <TableCell className="font-medium">
                      <div className="flex items-center gap-2">
                        {r.name}
                        {r.is_system && <Badge variant="secondary">{t('system')}</Badge>}
                        {r.is_default && <Badge variant="outline">{t('default')}</Badge>}
                      </div>
                    </TableCell>
                    <TableCell className="text-muted-foreground">{r.description}</TableCell>
                    <TableCell className="text-right tabular-nums text-muted-foreground">
                      {r.permissions?.length ?? 0}
                    </TableCell>
                    <TableCell>
                      <div className="flex justify-end gap-1">
                        <Button
                          variant="ghost"
                          size="icon"
                          onClick={() => setEditing(r)}
                          title={r.is_system ? t('systemReadonly') : t('save')}
                        >
                          <Pencil className="h-4 w-4" />
                        </Button>
                        {!r.is_system && (
                          <Button variant="ghost" size="icon" onClick={() => setDeleting(r)}>
                            <Trash2 className="h-4 w-4 text-red-600" />
                          </Button>
                        )}
                      </div>
                    </TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          </div>
        )}
      </div>

      {(creating || editing) && catalog && (
        <RoleDialog
          role={editing}
          catalog={catalog}
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
            <AlertDialogDescription>{t('deleteConfirmDesc')}</AlertDialogDescription>
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

function RoleDialog({
  role,
  catalog,
  onClose,
  onSaved,
}: {
  role: Role | null
  catalog: RoleCatalog
  onClose: () => void
  onSaved: () => void
}) {
  const t = useTranslations('roles')
  const { toast } = useToast()
  const readonly = !!role?.is_system

  const [name, setName] = useState(role?.name ?? '')
  const [description, setDescription] = useState(role?.description ?? '')
  const [perms, setPerms] = useState<Set<string>>(
    new Set((role?.permissions ?? []).map((p) => permKey(p.resource, p.action)))
  )

  const toggle = (resource: string, action: string) => {
    if (readonly) return
    const key = permKey(resource, action)
    setPerms((prev) => {
      const next = new Set(prev)
      if (next.has(key)) next.delete(key)
      else next.add(key)
      return next
    })
  }

  const permissions: Permission[] = useMemo(
    () =>
      Array.from(perms).map((k) => {
        const [resource, action] = k.split(':')
        return { resource, action }
      }),
    [perms]
  )

  const saveMutation = useMutation({
    mutationFn: (body: RoleRequest) =>
      role
        ? api.put<Role>(`/roles/${role.id}`, body)
        : api.post<Role>('/roles', body),
    onSuccess: onSaved,
    onError: (e: Error) => toast({ title: 'Error', description: e.message, variant: 'error' }),
  })

  return (
    <Dialog open onOpenChange={(o) => !o && onClose()}>
      <DialogContent className="max-w-3xl">
        <DialogHeader>
          <DialogTitle>{role ? t('editTitle') : t('createTitle')}</DialogTitle>
        </DialogHeader>

        <div className="space-y-4">
          {readonly && (
            <p className="rounded-md bg-muted px-3 py-2 text-xs text-muted-foreground">
              {t('systemReadonly')}
            </p>
          )}
          <div className="grid grid-cols-2 gap-4">
            <div className="space-y-2">
              <Label>{t('name')}</Label>
              <Input value={name} onChange={(e) => setName(e.target.value)} disabled={readonly} />
            </div>
            <div className="space-y-2">
              <Label>{t('description')}</Label>
              <Input
                value={description}
                onChange={(e) => setDescription(e.target.value)}
                disabled={readonly}
              />
            </div>
          </div>

          <div className="space-y-2">
            <Label>{t('permissions')}</Label>
            <div className="max-h-[340px] overflow-auto rounded-lg border">
              <Table>
                <TableHeader>
                  <TableRow>
                    <TableHead className="sticky left-0 bg-background">{t('resource')}</TableHead>
                    {catalog.actions.map((a) => (
                      <TableHead key={a} className="text-center capitalize">
                        {a}
                      </TableHead>
                    ))}
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {catalog.resources.map((resource) => (
                    <TableRow key={resource}>
                      <TableCell className="sticky left-0 bg-background font-mono text-xs">
                        {resource}
                      </TableCell>
                      {catalog.actions.map((action) => (
                        <TableCell key={action} className="text-center">
                          <input
                            type="checkbox"
                            className="h-4 w-4 accent-primary"
                            checked={perms.has(permKey(resource, action))}
                            onChange={() => toggle(resource, action)}
                            disabled={readonly}
                          />
                        </TableCell>
                      ))}
                    </TableRow>
                  ))}
                </TableBody>
              </Table>
            </div>
          </div>
        </div>

        <DialogFooter>
          <Button variant="outline" onClick={onClose}>
            {t('cancel')}
          </Button>
          {!readonly && (
            <Button
              onClick={() => saveMutation.mutate({ name, description, permissions })}
              disabled={!name.trim() || saveMutation.isPending}
            >
              {role ? t('save') : t('create')}
            </Button>
          )}
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}
