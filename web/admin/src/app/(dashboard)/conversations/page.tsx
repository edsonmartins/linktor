'use client'

import { useState } from 'react'
import { useQuery } from '@tanstack/react-query'
import { useTranslations } from 'next-intl'
import { Search, Filter, MessageSquare, Plus } from 'lucide-react'
import { Header } from '@/components/layout/header'
import { Input } from '@/components/ui/input'
import { Button } from '@/components/ui/button'
import { Badge } from '@/components/ui/badge'
import { Avatar } from '@/components/ui/avatar'
import { ScrollArea } from '@/components/ui/scroll-area'
import { Skeleton } from '@/components/ui/skeleton'
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu'
import { cn, formatRelativeTime, truncate } from '@/lib/utils'
import { api } from '@/lib/api'
import { queryKeys } from '@/lib/query'
import { useUIStore, useActiveConversation } from '@/stores/ui-store'
import { ChatView } from './chat-view'
import type { Conversation, ConversationStatus, PaginatedResponse } from '@/types'

/**
 * Conversation List Item
 */
interface ConversationItemProps {
  conversation: Conversation
  isActive: boolean
  onClick: () => void
  t: (key: string) => string
}

function ConversationItem({ conversation, isActive, onClick, t }: ConversationItemProps) {
  const statusVariant: Record<ConversationStatus, 'success' | 'warning' | 'info' | 'secondary'> = {
    open: 'success',
    pending: 'warning',
    resolved: 'info',
    snoozed: 'secondary',
  }

  return (
    <button
      onClick={onClick}
      className={cn(
        'flex w-full items-start gap-3 rounded-lg p-3 text-left transition-colors',
        isActive
          ? 'bg-primary/10 border border-primary/30'
          : 'hover:bg-secondary/50'
      )}
    >
      <Avatar
        fallback={conversation.contact?.name || 'U'}
        size="default"
        status={conversation.status === 'open' ? 'online' : 'offline'}
      />
      <div className="flex-1 min-w-0">
        <div className="flex items-center justify-between gap-2">
          <span className="font-medium truncate">
            {conversation.contact?.name || t('unknownContact')}
          </span>
          <span className="text-xs text-muted-foreground shrink-0">
            {conversation.last_message_at
              ? formatRelativeTime(conversation.last_message_at)
              : '-'}
          </span>
        </div>
        <div className="flex items-center gap-2 mt-1">
          <Badge
            variant={conversation.channel?.type as 'webchat' | undefined || 'secondary'}
            className="text-[10px] px-1.5 py-0"
          >
            {conversation.channel?.type || 'unknown'}
          </Badge>
          <Badge
            variant={statusVariant[conversation.status]}
            className="text-[10px] px-1.5 py-0"
          >
            {t(conversation.status)}
          </Badge>
        </div>
        <p className="mt-1 text-xs text-muted-foreground truncate">
          {conversation.last_message?.content
            ? truncate(conversation.last_message.content, 50)
            : t('noMessagesYet')}
        </p>
      </div>
      {conversation.unread_count && conversation.unread_count > 0 && (
        <span className="flex h-5 w-5 items-center justify-center rounded-full bg-primary text-[10px] font-bold text-primary-foreground">
          {conversation.unread_count > 9 ? '9+' : conversation.unread_count}
        </span>
      )}
    </button>
  )
}

/**
 * Conversations Page
 */
export default function ConversationsPage() {
  const t = useTranslations('conversations')
  const tCommon = useTranslations('common')
  const [searchQuery, setSearchQuery] = useState('')
  const [statusFilter, setStatusFilter] = useState<ConversationStatus | 'all'>('all')
  const activeConversationId = useActiveConversation()
  const setActiveConversation = useUIStore((s) => s.setActiveConversation)

  // Status filter options with translations
  const statusFilters: { label: string; value: ConversationStatus | 'all' }[] = [
    { label: t('all'), value: 'all' },
    { label: t('open'), value: 'open' },
    { label: t('pending'), value: 'pending' },
    { label: t('resolved'), value: 'resolved' },
    { label: t('snoozed'), value: 'snoozed' },
  ]

  // Fetch conversations
  const { data, isLoading } = useQuery({
    queryKey: queryKeys.conversations.list({
      search: searchQuery,
      status: statusFilter,
    }),
    queryFn: () =>
      api.get<PaginatedResponse<Conversation>>('/conversations', {
        ...(searchQuery && { search: searchQuery }),
        ...(statusFilter !== 'all' && { status: statusFilter }),
      }),
  })

  const conversations = data?.data ?? []

  return (
    <div className="flex h-full">
      {/* Conversation List */}
      <div className="flex w-80 flex-col border-r border-border bg-card">
        <Header title={t('title')} />

        {/* Search and Filters */}
        <div className="border-b border-border p-3 space-y-2">
          <div className="flex items-center gap-2">
            <Input
              placeholder={t('searchPlaceholder')}
              value={searchQuery}
              onChange={(e) => setSearchQuery(e.target.value)}
              leftIcon={<Search className="h-4 w-4" />}
              className="flex-1"
            />
            <DropdownMenu>
              <DropdownMenuTrigger asChild>
                <Button variant="outline" size="icon">
                  <Filter className="h-4 w-4" />
                </Button>
              </DropdownMenuTrigger>
              <DropdownMenuContent align="end">
                <DropdownMenuLabel>{t('filterByStatus')}</DropdownMenuLabel>
                <DropdownMenuSeparator />
                {statusFilters.map((filter) => (
                  <DropdownMenuItem
                    key={filter.value}
                    onClick={() => setStatusFilter(filter.value)}
                    className={cn(
                      statusFilter === filter.value && 'bg-primary/10 text-primary'
                    )}
                  >
                    {filter.label}
                  </DropdownMenuItem>
                ))}
              </DropdownMenuContent>
            </DropdownMenu>
          </div>
          {statusFilter !== 'all' && (
            <div className="flex items-center gap-2">
              <Badge variant="outline" className="gap-1">
                {tCommon('status')}: {t(statusFilter)}
                <button
                  onClick={() => setStatusFilter('all')}
                  className="ml-1 hover:text-foreground"
                >
                  ×
                </button>
              </Badge>
            </div>
          )}
        </div>

        {/* Conversation List */}
        <ScrollArea className="flex-1">
          <div className="p-2 space-y-1">
            {isLoading ? (
              // Loading skeletons
              Array.from({ length: 8 }).map((_, i) => (
                <div key={i} className="flex items-start gap-3 p-3">
                  <Skeleton className="h-10 w-10 rounded-full" />
                  <div className="flex-1 space-y-2">
                    <Skeleton className="h-4 w-32" />
                    <Skeleton className="h-3 w-24" />
                    <Skeleton className="h-3 w-48" />
                  </div>
                </div>
              ))
            ) : conversations.length > 0 ? (
              conversations.map((conversation) => (
                <ConversationItem
                  key={conversation.id}
                  conversation={conversation}
                  isActive={activeConversationId === conversation.id}
                  onClick={() => setActiveConversation(conversation.id)}
                  t={t}
                />
              ))
            ) : (
              <div className="py-12 text-center text-muted-foreground">
                <MessageSquare className="mx-auto h-8 w-8 opacity-50" />
                <p className="mt-2 text-sm">{t('noConversations')}</p>
                <p className="text-xs">{t('adjustFilters')}</p>
              </div>
            )}
          </div>
        </ScrollArea>
      </div>

      {/* Chat View */}
      <div className="flex-1 flex flex-col">
        {activeConversationId ? (
          <ChatView conversationId={activeConversationId} />
        ) : (
          <div className="flex-1 flex items-center justify-center bg-background">
            <div className="text-center text-muted-foreground">
              <MessageSquare className="mx-auto h-12 w-12 opacity-50" />
              <p className="mt-4 text-lg font-medium">{t('selectConversation')}</p>
              <p className="text-sm">{t('selectConversationDescription')}</p>
            </div>
          </div>
        )}
      </div>
    </div>
  )
}
