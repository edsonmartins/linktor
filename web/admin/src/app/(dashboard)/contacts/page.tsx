'use client'

import { useState } from 'react'
import { useQuery } from '@tanstack/react-query'
import { useTranslations } from 'next-intl'
import { Search, Plus, Filter, Users, Mail, Phone, MoreVertical } from 'lucide-react'
import { Header } from '@/components/layout/header'
import { Input } from '@/components/ui/input'
import { Button } from '@/components/ui/button'
import { Badge } from '@/components/ui/badge'
import { Avatar } from '@/components/ui/avatar'
import { ScrollArea } from '@/components/ui/scroll-area'
import { Skeleton } from '@/components/ui/skeleton'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu'
import { cn, formatDate } from '@/lib/utils'
import { api } from '@/lib/api'
import { queryKeys } from '@/lib/query'
import type { Contact, PaginatedResponse } from '@/types'

/**
 * Contact Card Component
 */
function ContactCard({
  contact,
  t,
  tCommon,
}: {
  contact: Contact
  t: (key: string) => string
  tCommon: (key: string) => string
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
                  <DropdownMenuItem>{t('viewDetails')}</DropdownMenuItem>
                  <DropdownMenuItem>{t('startConversation')}</DropdownMenuItem>
                  <DropdownMenuItem>{t('editContact')}</DropdownMenuItem>
                  <DropdownMenuItem className="text-destructive">
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
  const [searchQuery, setSearchQuery] = useState('')

  // Fetch contacts
  const { data, isLoading } = useQuery({
    queryKey: queryKeys.contacts.list({ search: searchQuery }),
    queryFn: () =>
      api.get<PaginatedResponse<Contact>>('/contacts', {
        ...(searchQuery && { search: searchQuery }),
      }),
  })

  const contacts = data?.data ?? []

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
          <Button>
            <Plus className="h-4 w-4 mr-2" />
            {t('addContact')}
          </Button>
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
                />
              ))}
            </div>
          ) : (
            <div className="py-12 text-center text-muted-foreground">
              <Users className="mx-auto h-12 w-12 opacity-50" />
              <p className="mt-4 text-lg font-medium">{t('noContacts')}</p>
              <p className="text-sm">{t('noContactsDescription')}</p>
              <Button className="mt-4">
                <Plus className="h-4 w-4 mr-2" />
                {t('addContact')}
              </Button>
            </div>
          )}
        </ScrollArea>
      </div>
    </div>
  )
}
