'use client'

import { useCallback, useEffect, useMemo, useRef, useState } from 'react'
import { useInfiniteQuery, useQuery } from '@tanstack/react-query'
import { useTranslations } from 'next-intl'
import { Search, Filter, MessageSquare, Plus, RefreshCw, Loader2 } from 'lucide-react'
import { Header } from '@/components/layout/header'
import { Input } from '@/components/ui/input'
import { Button } from '@/components/ui/button'
import { Badge } from '@/components/ui/badge'
import { Avatar } from '@/components/ui/avatar'
import { EnvironmentBadge } from '@/components/environment-badge'
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
import type { Channel, Conversation, ConversationStatus } from '@/types'

/**
 * Conversation List Item
 */
interface ConversationItemProps {
  conversation: Conversation
  isActive: boolean
  onClick: () => void
  t: (key: string) => string
}

/**
 * O que a linha da conversa mostra como última fala.
 *
 * Áudio, imagem, documento e localização chegam sem texto — é a natureza
 * deles. Caindo no "ainda sem mensagens", a linha dizia que a conversa estava
 * vazia quando o que havia era um áudio recém-recebido. Sem texto, anuncia-se
 * o tipo, como em qualquer mensageiro.
 */
function previaDaConversa(
  conversation: Conversation,
  t: (key: string) => string
): string {
  const ultima = conversation.last_message
  if (!ultima) return t('noMessagesYet')

  const texto = ultima.content?.trim()
  if (texto) return truncate(texto, 50)

  switch (ultima.content_type) {
    case 'image':
      return t('previewImage')
    case 'video':
      return t('previewVideo')
    case 'audio':
      return t('previewAudio')
    case 'document':
      return t('previewDocument')
    case 'sticker':
      return t('previewSticker')
    case 'location':
      return t('previewLocation')
    case 'contact':
      return t('previewContact')
    default:
      return t('unsupportedMessage')
  }
}

function ConversationItem({ conversation, isActive, onClick, t }: ConversationItemProps) {
  const statusVariant: Record<ConversationStatus, 'success' | 'warning' | 'info' | 'secondary'> = {
    open: 'success',
    pending: 'warning',
    resolved: 'info',
    snoozed: 'secondary',
  }

  // Treat the legacy hardcoded "Unknown" the same as an empty name so it is
  // localized, not shown as the raw English word. When no real name is known,
  // fall back to the phone number so the contact is still identifiable (the
  // user's ask) — only then to the localized "Unknown Contact".
  const rawName = conversation.contact?.name
  const contactName =
    (rawName && rawName !== 'Unknown' ? rawName : '') || conversation.contact?.phone || ''
  // Label the channel by its name (falling back to type) so two channels of the
  // same type are distinguishable.
  const channelLabel = conversation.channel?.name || conversation.channel?.type || 'unknown'

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
        fallback={contactName || 'U'}
        size="default"
        status={conversation.status === 'open' ? 'online' : 'offline'}
      />
      <div className="flex-1 min-w-0">
        <div className="flex items-center justify-between gap-2">
          <span className="font-medium truncate">
            {contactName || t('unknownContact')}
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
            title={conversation.channel?.type}
          >
            {channelLabel}
          </Badge>
          <Badge
            variant={statusVariant[conversation.status]}
            className="text-[10px] px-1.5 py-0"
          >
            {t(conversation.status)}
          </Badge>
          {/* Marcação de origem sintética (INV-018): valor vem da API. */}
          <EnvironmentBadge
            environment={conversation.environment}
            className="text-[10px] px-1.5 py-0"
          />
        </div>
        <p className="mt-1 text-xs text-muted-foreground truncate">
          {previaDaConversa(conversation, t)}
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

// Quantas conversas por página da caixa. Enche uma tela alta sem fazer a
// primeira pintura esperar pela caixa inteira.
const CONVERSATIONS_PAGE_SIZE = 30

/**
 * Conversations Page
 */
export default function ConversationsPage() {
  const t = useTranslations('conversations')
  const tCommon = useTranslations('common')
  const tEnv = useTranslations('environment')
  const [searchQuery, setSearchQuery] = useState('')
  // A busca vai ao servidor; esperar a digitação parar evita uma consulta por
  // tecla sobre a caixa inteira.
  const [debouncedSearch, setDebouncedSearch] = useState('')
  const [statusFilter, setStatusFilter] = useState<ConversationStatus | 'all'>('all')
  // Default "all": sandbox conversations stay discoverable and the mandatory
  // badge is what prevents reading them as real (decisão WP-H).
  const [environmentFilter, setEnvironmentFilter] = useState<'all' | 'production' | 'sandbox'>('all')
  const [channelFilter, setChannelFilter] = useState<string>('all')
  useEffect(() => {
    const id = setTimeout(() => setDebouncedSearch(searchQuery.trim()), 300)
    return () => clearTimeout(id)
  }, [searchQuery])

  const activeConversationId = useActiveConversation()
  const setActiveConversation = useUIStore((s) => s.setActiveConversation)

  // Resizable left panel (conversation list). Width is dragged via the splitter,
  // clamped to a sensible range and persisted so it survives reloads.
  const LEFT_MIN = 260
  const LEFT_MAX = 560
  const rootRef = useRef<HTMLDivElement>(null)
  const [leftWidth, setLeftWidth] = useState(320)
  const [isResizing, setIsResizing] = useState(false)

  useEffect(() => {
    const saved = Number(localStorage.getItem('conversations:leftWidth'))
    if (saved >= LEFT_MIN && saved <= LEFT_MAX) setLeftWidth(saved)
  }, [])

  const startResize = useCallback((e: React.MouseEvent) => {
    e.preventDefault()
    const rootLeft = rootRef.current?.getBoundingClientRect().left ?? 0
    setIsResizing(true)
    document.body.style.userSelect = 'none'
    document.body.style.cursor = 'col-resize'

    const onMove = (ev: MouseEvent) => {
      const next = Math.min(LEFT_MAX, Math.max(LEFT_MIN, ev.clientX - rootLeft))
      setLeftWidth(next)
    }
    const onUp = () => {
      setIsResizing(false)
      document.body.style.userSelect = ''
      document.body.style.cursor = ''
      window.removeEventListener('mousemove', onMove)
      window.removeEventListener('mouseup', onUp)
      setLeftWidth((w) => {
        localStorage.setItem('conversations:leftWidth', String(w))
        return w
      })
    }
    window.addEventListener('mousemove', onMove)
    window.addEventListener('mouseup', onUp)
  }, [])

  // Status filter options with translations
  const statusFilters: { label: string; value: ConversationStatus | 'all' }[] = [
    { label: t('all'), value: 'all' },
    { label: t('open'), value: 'open' },
    { label: t('pending'), value: 'pending' },
    { label: t('resolved'), value: 'resolved' },
    { label: t('snoozed'), value: 'snoozed' },
  ]

  // Channels for the per-channel filter (and to label rows by channel name).
  const { data: channels = [] } = useQuery({
    queryKey: queryKeys.channels.list(),
    queryFn: () => api.get<Channel[]>('/channels'),
  })

  // Busca e filtros são aplicados pelo BACKEND (query params), nunca em memória
  // aqui (WP-H). A caixa vem por página, da conversa mais ativa para a mais
  // parada, e a rolagem busca as seguintes — antes a tela recebia as 20
  // primeiras e não havia como chegar às demais.
  const {
    data,
    isLoading,
    refetch,
    isFetching,
    hasNextPage: hasMoreConversations,
    isFetchingNextPage: isLoadingMoreConversations,
    fetchNextPage: fetchMoreConversations,
  } = useInfiniteQuery({
    queryKey: queryKeys.conversations.list({
      search: debouncedSearch,
      status: statusFilter,
      environment: environmentFilter,
      channel_id: channelFilter,
    }),
    queryFn: ({ pageParam }) =>
      api.getEnvelope<Conversation[]>('/conversations', {
        limit: String(CONVERSATIONS_PAGE_SIZE),
        ...(pageParam ? { before: pageParam } : {}),
        ...(debouncedSearch && { search: debouncedSearch }),
        ...(statusFilter !== 'all' && { status: statusFilter }),
        ...(environmentFilter !== 'all' && { environment: environmentFilter }),
        ...(channelFilter !== 'all' && { channel_id: channelFilter }),
      }),
    initialPageParam: '',
    getNextPageParam: (lastPage) => lastPage.meta?.next_cursor || undefined,
  })

  const conversations = useMemo(
    () => (data?.pages ?? []).flatMap((page) => page.data ?? []),
    [data]
  )
  const totalConversations = data?.pages?.[0]?.meta?.total_items ?? 0

  // Chegar perto do fim da lista traz a página seguinte.
  const handleListScroll = useCallback(
    (event: React.UIEvent<HTMLDivElement>) => {
      const el = event.currentTarget
      const faltaRolar = el.scrollHeight - el.scrollTop - el.clientHeight
      if (faltaRolar < 300 && hasMoreConversations && !isLoadingMoreConversations) {
        fetchMoreConversations()
      }
    },
    [hasMoreConversations, isLoadingMoreConversations, fetchMoreConversations]
  )
  const selectedChannel = channels.find((ch) => ch.id === channelFilter)

  return (
    <div ref={rootRef} className="flex h-full overflow-hidden">
      {/* Conversation List */}
      <div
        className="flex min-h-0 shrink-0 flex-col border-r border-border bg-card"
        style={{ width: leftWidth }}
      >
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
            <Button
              variant="outline"
              size="icon"
              onClick={() => refetch()}
              disabled={isFetching}
            >
              <RefreshCw className={cn("h-4 w-4", isFetching && "animate-spin")} />
            </Button>
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
                <DropdownMenuSeparator />
                <DropdownMenuLabel>{tEnv('filterLabel')}</DropdownMenuLabel>
                <DropdownMenuSeparator />
                {([
                  { value: 'all', label: tEnv('filterAll') },
                  { value: 'production', label: tEnv('filterProduction') },
                  { value: 'sandbox', label: tEnv('filterSandbox') },
                ] as const).map((filter) => (
                  <DropdownMenuItem
                    key={filter.value}
                    onClick={() => setEnvironmentFilter(filter.value)}
                    className={cn(
                      environmentFilter === filter.value && 'bg-primary/10 text-primary'
                    )}
                  >
                    {filter.label}
                  </DropdownMenuItem>
                ))}
                {channels.length > 0 && (
                  <>
                    <DropdownMenuSeparator />
                    <DropdownMenuLabel>{t('filterByChannel')}</DropdownMenuLabel>
                    <DropdownMenuSeparator />
                    <DropdownMenuItem
                      onClick={() => setChannelFilter('all')}
                      className={cn(channelFilter === 'all' && 'bg-primary/10 text-primary')}
                    >
                      {t('allChannels')}
                    </DropdownMenuItem>
                    {channels.map((ch) => (
                      <DropdownMenuItem
                        key={ch.id}
                        onClick={() => setChannelFilter(ch.id)}
                        className={cn(channelFilter === ch.id && 'bg-primary/10 text-primary')}
                      >
                        {ch.name || ch.type}
                      </DropdownMenuItem>
                    ))}
                  </>
                )}
              </DropdownMenuContent>
            </DropdownMenu>
          </div>
          {(statusFilter !== 'all' || environmentFilter !== 'all' || channelFilter !== 'all') && (
            <div className="flex flex-wrap items-center gap-2">
              {channelFilter !== 'all' && (
                <Badge variant="outline" className="gap-1">
                  {t('filterByChannel')}: {selectedChannel?.name || selectedChannel?.type || channelFilter}
                  <button
                    onClick={() => setChannelFilter('all')}
                    className="ml-1 hover:text-foreground"
                  >
                    ×
                  </button>
                </Badge>
              )}
              {statusFilter !== 'all' && (
                <Badge variant="outline" className="gap-1">
                  {tCommon('status')}: {t(statusFilter)}
                  <button
                    onClick={() => setStatusFilter('all')}
                    className="ml-1 hover:text-foreground"
                  >
                    ×
                  </button>
                </Badge>
              )}
              {environmentFilter !== 'all' && (
                <Badge variant="outline" className="gap-1">
                  {tEnv('filterLabel')}: {environmentFilter === 'sandbox' ? tEnv('filterSandbox') : tEnv('filterProduction')}
                  <button
                    onClick={() => setEnvironmentFilter('all')}
                    className="ml-1 hover:text-foreground"
                  >
                    ×
                  </button>
                </Badge>
              )}
            </div>
          )}
        </div>

        {/* Conversation List */}
        <ScrollArea className="min-h-0 flex-1" onViewportScroll={handleListScroll}>
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

            {isLoadingMoreConversations && (
              <div className="flex justify-center py-3">
                <Loader2 className="h-4 w-4 animate-spin text-muted-foreground" />
              </div>
            )}

            {/* Quantas há ao todo, e quantas já desceram: sem isso não dá para
                saber se a caixa acabou ou se a rolagem parou. */}
            {!isLoading && conversations.length > 0 && (
              <p className="py-3 text-center text-[11px] text-muted-foreground">
                {hasMoreConversations
                  ? t('showingOfTotal', { shown: conversations.length, total: totalConversations })
                  : t('allLoaded', { total: conversations.length })}
              </p>
            )}
          </div>
        </ScrollArea>
      </div>

      {/* Splitter */}
      <div
        role="separator"
        aria-orientation="vertical"
        onMouseDown={startResize}
        onDoubleClick={() => {
          setLeftWidth(320)
          localStorage.setItem('conversations:leftWidth', '320')
        }}
        title={t('resizePanel')}
        className={cn(
          'group relative w-1 shrink-0 cursor-col-resize bg-border transition-colors hover:bg-primary/40',
          isResizing && 'bg-primary/60'
        )}
      >
        {/* Wider invisible hit area for easier grabbing */}
        <span className="absolute inset-y-0 -left-1.5 -right-1.5" />
      </div>

      {/* Chat View */}
      <div className="flex min-h-0 min-w-0 flex-1 flex-col">
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
