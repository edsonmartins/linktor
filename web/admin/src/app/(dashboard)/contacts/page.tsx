'use client'

import { useState } from 'react'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { useTranslations } from 'next-intl'
import { Search, Plus, Filter, Users, Mail, Phone, MoreVertical, RefreshCw } from 'lucide-react'
import { Header } from '@/components/layout/header'
import { Input } from '@/components/ui/input'
import { Button } from '@/components/ui/button'
import { Badge } from '@/components/ui/badge'
import { Avatar } from '@/components/ui/avatar'
import { ScrollArea } from '@/components/ui/scroll-area'
import { Skeleton } from '@/components/ui/skeleton'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Label } from '@/components/ui/label'
import {
  Dialog,
  DialogContent,
  DialogDescription,
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
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu'
import { cn, formatDate } from '@/lib/utils'
import { api } from '@/lib/api'
import { queryKeys } from '@/lib/query'
import { useToast } from '@/hooks/use-toast'
import type { Contact } from '@/types'

/**
 * Contact Card Component
 */
function ContactCard({
  contact,
  t,
  tCommon,
  onView,
  onEdit,
  onDelete,
}: {
  contact: Contact
  t: (key: string) => string
  tCommon: (key: string) => string
  onView: () => void
  onEdit: () => void
  onDelete: () => void
}) {
  return (
    <Card className="hover:border-primary/30 transition-colors">
      <CardContent className="p-4">
        <div className="flex items-start gap-3">
          <Avatar fallback={contact.name} size="lg" />
          <div className="flex-1 min-w-0">
            <div className="flex items-center justify-between">
              <h3 className="font-medium truncate">{contact.name}</h3>
              <DropdownMenu>
                <DropdownMenuTrigger asChild>
                  <Button variant="ghost" size="icon" className="h-8 w-8">
                    <MoreVertical className="h-4 w-4" />
                  </Button>
                </DropdownMenuTrigger>
                <DropdownMenuContent align="end">
                  <DropdownMenuItem onClick={onView}>{t('viewDetails')}</DropdownMenuItem>
                  <DropdownMenuItem>{t('startConversation')}</DropdownMenuItem>
                  <DropdownMenuItem onClick={onEdit}>{t('editContact')}</DropdownMenuItem>
                  <DropdownMenuItem className="text-destructive" onClick={onDelete}>
                    {tCommon('delete')}
                  </DropdownMenuItem>
                </DropdownMenuContent>
              </DropdownMenu>
            </div>

            <div className="mt-2 space-y-1 text-sm text-muted-foreground">
              {contact.email && (
                <div className="flex items-center gap-2">
                  <Mail className="h-3 w-3" />
                  <span className="truncate">{contact.email}</span>
                </div>
              )}
              {contact.phone && (
                <div className="flex items-center gap-2">
                  <Phone className="h-3 w-3" />
                  <span>{contact.phone}</span>
                </div>
              )}
            </div>

            {contact.identities && contact.identities.length > 0 && (
              <div className="mt-3 flex flex-wrap gap-1">
                {contact.identities.map((identity) => (
                  <Badge
                    key={identity.id}
                    variant={identity.channel_type as 'webchat' | undefined || 'secondary'}
                    className="text-[10px]"
                  >
                    {identity.channel_type}
                  </Badge>
                ))}
              </div>
            )}

            {contact.tags && contact.tags.length > 0 && (
              <div className="mt-2 flex flex-wrap gap-1">
                {contact.tags.map((tag) => (
                  <Badge key={tag} variant="outline" className="text-[10px]">
                    {tag}
                  </Badge>
                ))}
              </div>
            )}

            <p className="mt-2 text-xs text-muted-foreground">
              {t('created')} {formatDate(contact.created_at)}
            </p>
          </div>
        </div>
      </CardContent>
    </Card>
  )
}

/**
 * Contacts Page
 */
export default function ContactsPage() {
  const t = useTranslations('contacts')
  const tCommon = useTranslations('common')
  const { toast } = useToast()
  const queryClient = useQueryClient()
  const [searchQuery, setSearchQuery] = useState('')
  const [isDialogOpen, setIsDialogOpen] = useState(false)
  const [selectedContact, setSelectedContact] = useState<Contact | null>(null)
  const [contactToDelete, setContactToDelete] = useState<Contact | null>(null)
  const [editingContact, setEditingContact] = useState<Contact | null>(null)
  const [formData, setFormData] = useState({
    name: '',
    email: '',
    phone: '',
    tags: '',
  })

  // Fetch contacts
  const { data, isLoading, refetch, isFetching } = useQuery({
    queryKey: queryKeys.contacts.list({ search: searchQuery }),
    queryFn: () =>
      api.getEnvelope<Contact[]>('/contacts', {
        ...(searchQuery && { search: searchQuery }),
      }),
  })

  const contacts = data?.data ?? []

  const resetForm = () => {
    setFormData({
      name: '',
      email: '',
      phone: '',
      tags: '',
    })
    setEditingContact(null)
  }

  const createMutation = useMutation({
    mutationFn: (payload: { name: string; email: string; phone: string; tags: string[] }) =>
      api.post<Contact>('/contacts', payload),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: queryKeys.contacts.all })
      setIsDialogOpen(false)
      resetForm()
      toast({
        title: tCommon('created'),
        description: t('addContact'),
      })
    },
  })

  const updateMutation = useMutation({
    mutationFn: (payload: { id: string; body: { name: string; email: string; phone: string; tags: string[] } }) =>
      api.put<Contact>(`/contacts/${payload.id}`, payload.body),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: queryKeys.contacts.all })
      setIsDialogOpen(false)
      resetForm()
      toast({
        title: tCommon('updated'),
        description: t('editContact'),
      })
    },
  })

  const deleteMutation = useMutation({
    mutationFn: (id: string) => api.delete(`/contacts/${id}`),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: queryKeys.contacts.all })
      setContactToDelete(null)
      toast({
        title: tCommon('deleted'),
        description: t('deleteContact'),
      })
    },
  })

  const openCreateDialog = () => {
    resetForm()
    setIsDialogOpen(true)
  }

  const openEditDialog = (contact: Contact) => {
    setEditingContact(contact)
    setFormData({
      name: contact.name || '',
      email: contact.email || '',
      phone: contact.phone || '',
      tags: (contact.tags || []).join(', '),
    })
    setIsDialogOpen(true)
  }

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault()

    const payload = {
      name: formData.name.trim(),
      email: formData.email.trim(),
      phone: formData.phone.trim(),
      tags: formData.tags
        .split(',')
        .map((tag) => tag.trim())
        .filter(Boolean),
    }

    if (editingContact) {
      updateMutation.mutate({ id: editingContact.id, body: payload })
      return
    }

    createMutation.mutate(payload)
  }

  return (
    <div className="flex flex-col h-full">
      <Header title={t('title')} />

      <div className="p-6 space-y-6">
        {/* Search and Actions */}
        <div className="flex items-center justify-between gap-4">
          <div className="flex items-center gap-2 flex-1 max-w-md">
            <Input
              placeholder={t('searchPlaceholder')}
              value={searchQuery}
              onChange={(e) => setSearchQuery(e.target.value)}
              leftIcon={<Search className="h-4 w-4" />}
            />
            <Button variant="outline" size="icon">
              <Filter className="h-4 w-4" />
            </Button>
          </div>
          <div className="flex items-center gap-2">
            <Button
              variant="outline"
              size="icon"
              onClick={() => refetch()}
              disabled={isFetching}
            >
              <RefreshCw className={cn("h-4 w-4", isFetching && "animate-spin")} />
            </Button>
            <Button onClick={openCreateDialog}>
              <Plus className="h-4 w-4 mr-2" />
              {t('addContact')}
            </Button>
          </div>
        </div>

        {/* Stats */}
        <div className="grid gap-4 md:grid-cols-3">
          <Card>
            <CardHeader className="pb-2">
              <CardTitle className="text-sm font-medium text-muted-foreground">
                {t('totalContacts')}
              </CardTitle>
            </CardHeader>
            <CardContent>
              <div className="text-2xl font-bold">{contacts.length}</div>
            </CardContent>
          </Card>
          <Card>
            <CardHeader className="pb-2">
              <CardTitle className="text-sm font-medium text-muted-foreground">
                {t('withPhone')}
              </CardTitle>
            </CardHeader>
            <CardContent>
              <div className="text-2xl font-bold">
                {contacts.filter((c) => c.phone).length}
              </div>
            </CardContent>
          </Card>
          <Card>
            <CardHeader className="pb-2">
              <CardTitle className="text-sm font-medium text-muted-foreground">
                {t('withEmail')}
              </CardTitle>
            </CardHeader>
            <CardContent>
              <div className="text-2xl font-bold">
                {contacts.filter((c) => c.email).length}
              </div>
            </CardContent>
          </Card>
        </div>

        {/* Contacts Grid */}
        <ScrollArea className="flex-1">
          {isLoading ? (
            <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-3">
              {Array.from({ length: 6 }).map((_, i) => (
                <Card key={i}>
                  <CardContent className="p-4">
                    <div className="flex items-start gap-3">
                      <Skeleton className="h-12 w-12 rounded-full" />
                      <div className="flex-1 space-y-2">
                        <Skeleton className="h-4 w-32" />
                        <Skeleton className="h-3 w-40" />
                        <Skeleton className="h-3 w-28" />
                      </div>
                    </div>
                  </CardContent>
                </Card>
              ))}
            </div>
          ) : contacts.length > 0 ? (
            <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-3">
              {contacts.map((contact) => (
                <ContactCard
                  key={contact.id}
                  contact={contact}
                  t={t}
                  tCommon={tCommon}
                  onView={() => setSelectedContact(contact)}
                  onEdit={() => openEditDialog(contact)}
                  onDelete={() => setContactToDelete(contact)}
                />
              ))}
            </div>
          ) : (
            <div className="py-12 text-center text-muted-foreground">
              <Users className="mx-auto h-12 w-12 opacity-50" />
              <p className="mt-4 text-lg font-medium">{t('noContacts')}</p>
              <p className="text-sm">{t('noContactsDescription')}</p>
              <Button className="mt-4" onClick={openCreateDialog}>
                <Plus className="h-4 w-4 mr-2" />
                {t('addContact')}
              </Button>
            </div>
          )}
        </ScrollArea>
      </div>

      <Dialog open={!!selectedContact} onOpenChange={(open) => !open && setSelectedContact(null)}>
        <DialogContent className="sm:max-w-[560px]">
          <DialogHeader>
            <DialogTitle>{selectedContact?.name || t('viewDetails')}</DialogTitle>
            <DialogDescription>{t('viewDetails')}</DialogDescription>
          </DialogHeader>

          {selectedContact && (
            <div className="space-y-6">
              <div className="flex items-start gap-4">
                <Avatar fallback={selectedContact.name} size="lg" />
                <div className="min-w-0 space-y-1">
                  <h3 className="truncate font-medium">{selectedContact.name}</h3>
                  <p className="text-sm text-muted-foreground">
                    {t('created')} {formatDate(selectedContact.created_at)}
                  </p>
                </div>
              </div>

              <div className="grid gap-4 sm:grid-cols-2">
                <div className="space-y-1">
                  <p className="text-xs font-medium uppercase text-muted-foreground">{t('email')}</p>
                  <p className="break-words text-sm">{selectedContact.email || '-'}</p>
                </div>
                <div className="space-y-1">
                  <p className="text-xs font-medium uppercase text-muted-foreground">{t('phone')}</p>
                  <p className="break-words text-sm">{selectedContact.phone || '-'}</p>
                </div>
              </div>

              <div className="space-y-2">
                <p className="text-xs font-medium uppercase text-muted-foreground">{t('tags')}</p>
                {selectedContact.tags?.length ? (
                  <div className="flex flex-wrap gap-2">
                    {selectedContact.tags.map((tag) => (
                      <Badge key={tag} variant="outline">{tag}</Badge>
                    ))}
                  </div>
                ) : (
                  <p className="text-sm text-muted-foreground">-</p>
                )}
              </div>

              <div className="space-y-2">
                <p className="text-xs font-medium uppercase text-muted-foreground">Identities</p>
                {selectedContact.identities?.length ? (
                  <div className="space-y-2">
                    {selectedContact.identities.map((identity) => (
                      <div key={identity.id} className="rounded-md border p-3 text-sm">
                        <div className="flex items-center justify-between gap-3">
                          <Badge variant={identity.channel_type as 'webchat' | undefined || 'secondary'}>
                            {identity.channel_type}
                          </Badge>
                          <span className="truncate text-muted-foreground">{identity.external_id}</span>
                        </div>
                      </div>
                    ))}
                  </div>
                ) : (
                  <p className="text-sm text-muted-foreground">-</p>
                )}
              </div>
            </div>
          )}

          <DialogFooter>
            <Button type="button" variant="outline" onClick={() => setSelectedContact(null)}>
              {tCommon('close')}
            </Button>
            {selectedContact && (
              <Button
                type="button"
                onClick={() => {
                  openEditDialog(selectedContact)
                  setSelectedContact(null)
                }}
              >
                {t('editContact')}
              </Button>
            )}
          </DialogFooter>
        </DialogContent>
      </Dialog>

      <Dialog
        open={isDialogOpen}
        onOpenChange={(open) => {
          setIsDialogOpen(open)
          if (!open) {
            resetForm()
          }
        }}
      >
        <DialogContent className="sm:max-w-[500px]">
          <form onSubmit={handleSubmit}>
            <DialogHeader>
              <DialogTitle>{editingContact ? t('editContact') : t('addContact')}</DialogTitle>
              <DialogDescription>
                {editingContact ? t('editContact') : t('noContactsDescription')}
              </DialogDescription>
            </DialogHeader>

            <div className="grid gap-4 py-4">
              <div className="grid gap-2">
                <Label htmlFor="contact-name">{t('name')}</Label>
                <Input
                  id="contact-name"
                  value={formData.name}
                  onChange={(e) => setFormData((current) => ({ ...current, name: e.target.value }))}
                  required
                />
              </div>

              <div className="grid gap-2">
                <Label htmlFor="contact-email">{t('email')}</Label>
                <Input
                  id="contact-email"
                  type="email"
                  value={formData.email}
                  onChange={(e) => setFormData((current) => ({ ...current, email: e.target.value }))}
                />
              </div>

              <div className="grid gap-2">
                <Label htmlFor="contact-phone">{t('phone')}</Label>
                <Input
                  id="contact-phone"
                  value={formData.phone}
                  onChange={(e) => setFormData((current) => ({ ...current, phone: e.target.value }))}
                />
              </div>

              <div className="grid gap-2">
                <Label htmlFor="contact-tags">{t('tags')}</Label>
                <Input
                  id="contact-tags"
                  value={formData.tags}
                  onChange={(e) => setFormData((current) => ({ ...current, tags: e.target.value }))}
                  placeholder="vip, lead"
                />
              </div>
            </div>

            <DialogFooter>
              <Button type="button" variant="outline" onClick={() => setIsDialogOpen(false)}>
                {tCommon('cancel')}
              </Button>
              <Button type="submit" disabled={createMutation.isPending || updateMutation.isPending || !formData.name.trim()}>
                {editingContact ? t('editContact') : t('addContact')}
              </Button>
            </DialogFooter>
          </form>
        </DialogContent>
      </Dialog>

      <AlertDialog open={!!contactToDelete} onOpenChange={(open) => !open && setContactToDelete(null)}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>{t('deleteContact')}</AlertDialogTitle>
            <AlertDialogDescription>
              {contactToDelete ? `${t('deleteContact')}: ${contactToDelete.name}` : t('deleteContact')}
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel>{tCommon('cancel')}</AlertDialogCancel>
            <AlertDialogAction
              onClick={() => contactToDelete && deleteMutation.mutate(contactToDelete.id)}
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
