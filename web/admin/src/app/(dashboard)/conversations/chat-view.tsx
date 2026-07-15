'use client'

import { Fragment, useState, useRef, useEffect, useCallback } from 'react'
import { useTranslations } from 'next-intl'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import {
  Send,
  Paperclip,
  Smile,
  MoreVertical,
  Phone,
  Video,
  User,
  Clock,
  CheckCheck,
  Check,
  AlertCircle,
  Wifi,
  WifiOff,
  Smartphone,
  History,
} from 'lucide-react'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Avatar } from '@/components/ui/avatar'
import { Badge } from '@/components/ui/badge'
import { ScrollArea } from '@/components/ui/scroll-area'
import { Skeleton } from '@/components/ui/skeleton'
import { Separator } from '@/components/ui/separator'
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { Textarea } from '@/components/ui/textarea'
import { Label } from '@/components/ui/label'
import { SimpleTooltip } from '@/components/ui/tooltip'
import { cn, formatRelativeTime } from '@/lib/utils'
import { api } from '@/lib/api'
import { queryKeys } from '@/lib/query'
import { useUser } from '@/stores/auth-store'
import { useToast } from '@/hooks/use-toast'
import {
  useWebSocketContext,
  WSEventTypes,
  type WSNewMessagePayload,
  type WSTypingPayload,
} from '@/hooks/use-websocket'
import type {
  Conversation,
  EscalationContext,
  Message,
  MessageStatus,
  User as AppUser,
} from '@/types'

const QUICK_REACTIONS = ['👍', '❤️', '👀']

/**
 * Message Status Icon
 */
function MessageStatusIcon({ status }: { status: MessageStatus }) {
  switch (status) {
    case 'pending':
      return <Clock className="h-3 w-3 text-muted-foreground" />
    case 'sent':
      return <Check className="h-3 w-3 text-muted-foreground" />
    case 'delivered':
      return <CheckCheck className="h-3 w-3 text-muted-foreground" />
    case 'read':
      return <CheckCheck className="h-3 w-3 text-primary" />
    case 'failed':
      return <AlertCircle className="h-3 w-3 text-destructive" />
    default:
      return null
  }
}

/**
 * Message Source Badge Component
 * Shows the origin of the message (API, Business App, or Imported)
 */
interface MessageSourceBadgeProps {
  source?: Message['source']
  isImported?: boolean
  isOwn: boolean
}

function MessageSourceBadge({ source, isImported, isOwn }: MessageSourceBadgeProps) {
  const t = useTranslations('conversations')
  // Only show badge for outgoing messages from non-API sources
  if (!isOwn) return null

  // Check if it's an imported message
  if (isImported || source === 'imported') {
    return (
      <SimpleTooltip content={t('importedHistory')}>
        <Badge variant="outline" className="text-[8px] px-1 py-0 h-3.5 gap-0.5">
          <History className="h-2 w-2" />
          <span>{t('history')}</span>
        </Badge>
      </SimpleTooltip>
    )
  }

  // Check if it's from Business App (echo)
  if (source === 'business_app') {
    return (
      <SimpleTooltip content={t('sentViaApp')}>
        <Badge variant="outline" className="text-[8px] px-1 py-0 h-3.5 gap-0.5">
          <Smartphone className="h-2 w-2" />
          <span>{t('app')}</span>
        </Badge>
      </SimpleTooltip>
    )
  }

  // Only show API badge when there's mixed sources in conversation
  // For now, we skip showing the API badge since most messages are from API
  // and it would add visual noise
  return null
}

/**
 * Message Bubble Component
 */
interface MessageBubbleProps {
  message: Message
  isOwn: boolean
  currentUserID?: string
  onReact: (messageID: string, emoji: string) => void
  isReacting: boolean
}

function MessageBubble({ message, isOwn, currentUserID, onReact, isReacting }: MessageBubbleProps) {
  const ownReaction = message.reactions?.find((reaction) => reaction.user_id === currentUserID)?.emoji

  return (
    <div
      className={cn(
        'group flex gap-2 max-w-[70%]',
        isOwn ? 'ml-auto flex-row-reverse' : ''
      )}
    >
      {!isOwn && (
        <Avatar
          fallback={message.sender_type === 'contact' ? 'C' : 'S'}
          size="sm"
        />
      )}
      <div className={cn('space-y-1', isOwn && 'items-end')}>
        <div
          className={cn(
            'rounded-lg px-3 py-2',
            isOwn ? 'message-outgoing' : 'message-incoming'
          )}
        >
          <p className="text-sm whitespace-pre-wrap">{message.content}</p>
        </div>
        <div className={cn('flex flex-wrap items-center gap-1', isOwn && 'justify-end')}>
          {QUICK_REACTIONS.map((emoji) => (
            <button
              key={emoji}
              type="button"
              className={cn(
                'rounded-full border px-2 py-0.5 text-xs transition-colors',
                ownReaction === emoji
                  ? 'border-primary bg-primary/10 text-primary'
                  : 'border-border text-muted-foreground opacity-0 group-hover:opacity-100 hover:bg-secondary'
              )}
              disabled={isReacting}
              onClick={() => onReact(message.id, ownReaction === emoji ? '' : emoji)}
            >
              {emoji}
            </button>
          ))}
        </div>
        {message.reactions && message.reactions.length > 0 && (
          <div className={cn('flex flex-wrap items-center gap-1', isOwn && 'justify-end')}>
            {message.reactions.map((reaction) => (
              <Badge key={`${reaction.user_id}-${reaction.emoji}`} variant="outline" className="h-5 px-1.5 text-[10px]">
                {reaction.emoji}
              </Badge>
            ))}
          </div>
        )}
        <div
          className={cn(
            'flex items-center gap-1.5 text-[10px] text-muted-foreground',
            isOwn && 'justify-end'
          )}
        >
          <MessageSourceBadge
            source={message.source}
            isImported={message.is_imported}
            isOwn={isOwn}
          />
          <span>{formatRelativeTime(message.created_at)}</span>
          {isOwn && <MessageStatusIcon status={message.status} />}
        </div>
      </div>
    </div>
  )
}

/**
 * Chat Header
 */
interface ChatHeaderProps {
  conversation: Conversation
  onAssign: () => void
  onEscalate: () => void
  onResolveOrReopen: () => void
  onViewEscalationContext: () => void
  isResolving: boolean
  isEscalatingContextLoading: boolean
}

function ChatHeader({
  conversation,
  onAssign,
  onEscalate,
  onResolveOrReopen,
  onViewEscalationContext,
  isResolving,
  isEscalatingContextLoading,
}: ChatHeaderProps) {
  const t = useTranslations('conversations')
  const isResolved = conversation.status === 'resolved'
  return (
    <div className="flex h-16 items-center justify-between border-b border-border bg-card px-4">
      <div className="flex items-center gap-3">
        <Avatar
          fallback={conversation.contact?.name || 'U'}
          status={conversation.status === 'open' ? 'online' : 'offline'}
        />
        <div>
          <h2 className="font-medium">
            {conversation.contact?.name || t('unknownContact')}
          </h2>
          <div className="flex items-center gap-2 text-xs text-muted-foreground">
            <Badge
              variant={conversation.channel?.type as 'webchat' | undefined || 'secondary'}
              className="text-[10px] px-1.5 py-0"
            >
              {conversation.channel?.type}
            </Badge>
            <span>
              {conversation.contact?.phone || conversation.contact?.email || t('noContactInfo')}
            </span>
          </div>
        </div>
      </div>

      <div className="flex items-center gap-1">
        <SimpleTooltip content={t('voiceCall')}>
          <Button variant="ghost" size="icon">
            <Phone className="h-4 w-4" />
          </Button>
        </SimpleTooltip>
        <SimpleTooltip content={t('videoCall')}>
          <Button variant="ghost" size="icon">
            <Video className="h-4 w-4" />
          </Button>
        </SimpleTooltip>
        <SimpleTooltip content={t('contactInfo')}>
          <Button variant="ghost" size="icon">
            <User className="h-4 w-4" />
          </Button>
        </SimpleTooltip>
        <DropdownMenu>
          <DropdownMenuTrigger asChild>
            <Button variant="ghost" size="icon">
              <MoreVertical className="h-4 w-4" />
            </Button>
          </DropdownMenuTrigger>
          <DropdownMenuContent align="end">
            <DropdownMenuItem onClick={onResolveOrReopen} disabled={isResolving}>
              {isResolved ? 'Reopen conversation' : t('resolveConversation')}
            </DropdownMenuItem>
            <DropdownMenuItem onClick={onAssign}>
              {t('assignToAgent')}
            </DropdownMenuItem>
            <DropdownMenuItem onClick={onEscalate}>
              Escalate conversation
            </DropdownMenuItem>
            <DropdownMenuItem onClick={onViewEscalationContext} disabled={isEscalatingContextLoading}>
              View escalation context
            </DropdownMenuItem>
            <DropdownMenuSeparator />
            <DropdownMenuItem disabled>{t('snooze')}</DropdownMenuItem>
          </DropdownMenuContent>
        </DropdownMenu>
      </div>
    </div>
  )
}

interface AssignConversationDialogProps {
  conversation: Conversation
  users: AppUser[]
  open: boolean
  onOpenChange: (open: boolean) => void
  onAssign: (userID: string) => void
  isSubmitting: boolean
}

function AssignConversationDialog({
  conversation,
  users,
  open,
  onOpenChange,
  onAssign,
  isSubmitting,
}: AssignConversationDialogProps) {
  const [selectedUserID, setSelectedUserID] = useState(conversation.assigned_user_id || '')

  useEffect(() => {
    if (open) {
      setSelectedUserID(conversation.assigned_user_id || '')
    }
  }, [conversation.assigned_user_id, open])

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>Assign conversation</DialogTitle>
          <DialogDescription>
            Route this conversation to a team member.
          </DialogDescription>
        </DialogHeader>
        <div className="space-y-2">
          <Label htmlFor="assign-user">Agent</Label>
          <Select value={selectedUserID} onValueChange={setSelectedUserID}>
            <SelectTrigger id="assign-user">
              <SelectValue placeholder="Select an agent" />
            </SelectTrigger>
            <SelectContent>
              {users.map((user) => (
                <SelectItem key={user.id} value={user.id}>
                  {user.name} ({user.email})
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
        </div>
        <DialogFooter>
          <Button variant="outline" onClick={() => onOpenChange(false)}>
            Cancel
          </Button>
          <Button
            onClick={() => onAssign(selectedUserID)}
            disabled={!selectedUserID || isSubmitting}
          >
            {isSubmitting ? 'Assigning...' : 'Assign'}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}

interface EscalateConversationDialogProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  onEscalate: (payload: { reason: string; priority: 'low' | 'normal' | 'high' | 'urgent' }) => void
  context?: EscalationContext
  isLoadingContext: boolean
  isSubmitting: boolean
}

function EscalateConversationDialog({
  open,
  onOpenChange,
  onEscalate,
  context,
  isLoadingContext,
  isSubmitting,
}: EscalateConversationDialogProps) {
  const [reason, setReason] = useState('')
  const [priority, setPriority] = useState<'low' | 'normal' | 'high' | 'urgent'>('normal')

  useEffect(() => {
    if (open) {
      setReason(context?.reason_detail || context?.summary || '')
      setPriority(context?.priority || 'normal')
    }
  }, [context, open])

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-2xl">
        <DialogHeader>
          <DialogTitle>Escalate conversation</DialogTitle>
          <DialogDescription>
            Send this conversation to a human queue with context.
          </DialogDescription>
        </DialogHeader>
        <div className="grid gap-4 md:grid-cols-[1.2fr_0.8fr]">
          <div className="space-y-3">
            <div className="space-y-2">
              <Label htmlFor="escalation-reason">Reason</Label>
              <Textarea
                id="escalation-reason"
                value={reason}
                onChange={(event) => setReason(event.target.value)}
                placeholder="Explain why this conversation needs escalation"
                className="min-h-[120px]"
              />
            </div>
            <div className="space-y-2">
              <Label htmlFor="escalation-priority">Priority</Label>
              <Select value={priority} onValueChange={(value) => setPriority(value as typeof priority)}>
                <SelectTrigger id="escalation-priority">
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="low">Low</SelectItem>
                  <SelectItem value="normal">Normal</SelectItem>
                  <SelectItem value="high">High</SelectItem>
                  <SelectItem value="urgent">Urgent</SelectItem>
                </SelectContent>
              </Select>
            </div>
          </div>
          <div className="space-y-3 rounded-md border p-3 text-sm">
            <div>
              <p className="font-medium">Escalation context</p>
              <p className="text-xs text-muted-foreground">
                {isLoadingContext ? 'Loading context...' : 'Current handoff snapshot'}
              </p>
            </div>
            {context ? (
              <>
                {context.summary && (
                  <div>
                    <p className="text-xs font-medium uppercase text-muted-foreground">Summary</p>
                    <p>{context.summary}</p>
                  </div>
                )}
                <div className="flex flex-wrap gap-2">
                  {context.detected_intent && <Badge variant="outline">{context.detected_intent}</Badge>}
                  {context.sentiment && <Badge variant="outline">{context.sentiment}</Badge>}
                  {context.channel_type && <Badge variant="outline">{context.channel_type}</Badge>}
                </div>
                {context.last_messages?.length > 0 && (
                  <div className="space-y-2">
                    <p className="text-xs font-medium uppercase text-muted-foreground">Recent messages</p>
                    {context.last_messages.slice(0, 3).map((message) => (
                      <div key={message.id} className="rounded border p-2">
                        <p className="text-xs text-muted-foreground">
                          {message.sender_name || message.sender_type}
                        </p>
                        <p>{message.content}</p>
                      </div>
                    ))}
                  </div>
                )}
              </>
            ) : (
              <p className="text-muted-foreground">No escalation context available yet.</p>
            )}
          </div>
        </div>
        <DialogFooter>
          <Button variant="outline" onClick={() => onOpenChange(false)}>
            Cancel
          </Button>
          <Button
            onClick={() => onEscalate({ reason, priority })}
            disabled={!reason.trim() || isSubmitting}
          >
            {isSubmitting ? 'Escalating...' : 'Escalate'}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}

interface EscalationContextDialogProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  context?: EscalationContext
  isLoading: boolean
}

function EscalationContextDialog({
  open,
  onOpenChange,
  context,
  isLoading,
}: EscalationContextDialogProps) {
  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-2xl">
        <DialogHeader>
          <DialogTitle>Escalation context</DialogTitle>
          <DialogDescription>
            Handoff context generated for human support.
          </DialogDescription>
        </DialogHeader>
        {isLoading ? (
          <div className="space-y-2">
            <Skeleton className="h-5 w-40" />
            <Skeleton className="h-24 w-full" />
            <Skeleton className="h-24 w-full" />
          </div>
        ) : context ? (
          <div className="space-y-4 text-sm">
            <div>
              <p className="text-xs font-medium uppercase text-muted-foreground">Summary</p>
              <p>{context.summary || 'No summary available.'}</p>
            </div>
            <div className="flex flex-wrap gap-2">
              <Badge variant="outline">{context.priority}</Badge>
              {context.escalation_reason && <Badge variant="outline">{context.escalation_reason}</Badge>}
              {context.suggested_team && <Badge variant="outline">{context.suggested_team}</Badge>}
            </div>
            <div className="grid gap-3 md:grid-cols-2">
              <div>
                <p className="text-xs font-medium uppercase text-muted-foreground">Customer</p>
                <p>{context.customer?.name || '-'}</p>
                <p className="text-muted-foreground">{context.customer?.email || context.customer?.phone || '-'}</p>
              </div>
              <div>
                <p className="text-xs font-medium uppercase text-muted-foreground">Metrics</p>
                <p>Messages: {context.message_count}</p>
                <p>Bot attempts: {context.bot_attempts}</p>
                <p>Wait time: {context.wait_time_seconds}s</p>
              </div>
            </div>
            {context.last_messages?.length > 0 && (
              <div className="space-y-2">
                <p className="text-xs font-medium uppercase text-muted-foreground">Last messages</p>
                {context.last_messages.map((message) => (
                  <div key={message.id} className="rounded-md border p-3">
                    <p className="text-xs text-muted-foreground">
                      {message.sender_name || message.sender_type} · {formatRelativeTime(message.timestamp)}
                    </p>
                    <p>{message.content}</p>
                  </div>
                ))}
              </div>
            )}
          </div>
        ) : (
          <p className="text-sm text-muted-foreground">No escalation context available.</p>
        )}
      </DialogContent>
    </Dialog>
  )
}

/**
 * Typing Indicator Component
 */
interface TypingIndicatorProps {
  typingUsers: { user_id: string; user_name: string }[]
}

function TypingIndicator({ typingUsers }: TypingIndicatorProps) {
  const t = useTranslations('conversations')
  if (typingUsers.length === 0) return null

  const names = typingUsers.map((u) => u.user_name || 'Someone').join(', ')
  return (
    <div className="flex items-center gap-2 px-4 py-2 text-xs text-muted-foreground animate-pulse">
      <div className="flex gap-1">
        <span className="w-1.5 h-1.5 rounded-full bg-muted-foreground animate-bounce [animation-delay:-0.3s]" />
        <span className="w-1.5 h-1.5 rounded-full bg-muted-foreground animate-bounce [animation-delay:-0.15s]" />
        <span className="w-1.5 h-1.5 rounded-full bg-muted-foreground animate-bounce" />
      </div>
      <span>{names} {typingUsers.length === 1 ? t('isTyping') : t('areTyping')}</span>
    </div>
  )
}

/**
 * Message Composer
 */
interface ComposerProps {
  conversationId: string
  onSend: (content: string) => void
  onTyping: (isTyping: boolean) => void
  isSending: boolean
}

function Composer({ conversationId, onSend, onTyping, isSending }: ComposerProps) {
  const t = useTranslations('conversations')
  const [content, setContent] = useState('')
  const typingTimeoutRef = useRef<NodeJS.Timeout | null>(null)

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault()
    if (!content.trim() || isSending) return
    onTyping(false)
    onSend(content.trim())
    setContent('')
  }

  const handleKeyDown = (e: React.KeyboardEvent) => {
    if (e.key === 'Enter' && !e.shiftKey) {
      e.preventDefault()
      handleSubmit(e)
    }
  }

  const handleChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    setContent(e.target.value)

    // Send typing indicator
    onTyping(true)

    // Clear previous timeout
    if (typingTimeoutRef.current) {
      clearTimeout(typingTimeoutRef.current)
    }

    // Stop typing after 2 seconds of no input
    typingTimeoutRef.current = setTimeout(() => {
      onTyping(false)
    }, 2000)
  }

  return (
    <form
      onSubmit={handleSubmit}
      className="border-t border-border bg-card p-4"
    >
      <div className="flex items-end gap-2">
        <SimpleTooltip content={t('attachFile')}>
          <Button type="button" variant="ghost" size="icon">
            <Paperclip className="h-5 w-5" />
          </Button>
        </SimpleTooltip>

        <div className="flex-1">
          <Input
            placeholder={t('typeMessage')}
            value={content}
            onChange={handleChange}
            onKeyDown={handleKeyDown}
            variant="terminal"
            className="min-h-[40px]"
          />
        </div>

        <SimpleTooltip content={t('emoji')}>
          <Button type="button" variant="ghost" size="icon">
            <Smile className="h-5 w-5" />
          </Button>
        </SimpleTooltip>

        <Button
          type="submit"
          size="icon"
          disabled={!content.trim() || isSending}
          loading={isSending}
        >
          <Send className="h-5 w-5" />
        </Button>
      </div>
    </form>
  )
}

/**
 * Chat View Component
 */
interface ChatViewProps {
  conversationId: string
}

export function ChatView({ conversationId }: ChatViewProps) {
  const t = useTranslations('conversations')
  const queryClient = useQueryClient()
  const user = useUser()
  const { toast } = useToast()
  const messagesEndRef = useRef<HTMLDivElement>(null)
  const [typingUsers, setTypingUsers] = useState<{ user_id: string; user_name: string }[]>([])
  const [assignDialogOpen, setAssignDialogOpen] = useState(false)
  const [escalateDialogOpen, setEscalateDialogOpen] = useState(false)
  const [escalationContextOpen, setEscalationContextOpen] = useState(false)

  // WebSocket integration
  const { connectionState, subscribe, sendTyping } = useWebSocketContext()

  // Fetch conversation
  const { data: conversation, isLoading: conversationLoading } = useQuery({
    queryKey: queryKeys.conversations.detail(conversationId),
    queryFn: () => api.get<Conversation>(`/conversations/${conversationId}`),
  })

  // Fetch messages
  const { data: messagesData, isLoading: messagesLoading } = useQuery({
    queryKey: queryKeys.messages.list(conversationId),
    queryFn: () =>
      api.getEnvelope<Message[]>(`/conversations/${conversationId}/messages`),
  })

  // Render messages oldest→newest (newest at the bottom, matching the auto
  // scroll-to-bottom) with a deterministic id tiebreaker for same-second
  // messages, mirroring the backend order. The API may return newest-first, so
  // we sort here rather than rely on the transport order.
  const sortedMessages = [...(messagesData?.data || [])].sort((a, b) => {
    const ta = new Date(a.created_at).getTime()
    const tb = new Date(b.created_at).getTime()
    if (ta !== tb) return ta - tb
    return a.id.localeCompare(b.id)
  })

  // Local-day key + human label for the per-day date separators. Grouping is by
  // the viewer's local calendar day so a message at 23:45Z shows under the right
  // day for a UTC-3 user (not the UTC day).
  const dayKeyOf = (iso: string) => {
    const d = new Date(iso)
    return `${d.getFullYear()}-${d.getMonth()}-${d.getDate()}`
  }
  const labelForDay = (iso: string) => {
    const now = new Date()
    const yesterday = new Date()
    yesterday.setDate(now.getDate() - 1)
    const key = dayKeyOf(iso)
    if (key === dayKeyOf(now.toISOString())) return t('today')
    if (key === dayKeyOf(yesterday.toISOString())) return t('yesterday')
    return new Intl.DateTimeFormat('pt-BR', {
      day: '2-digit',
      month: '2-digit',
      year: 'numeric',
    }).format(new Date(iso))
  }

  const { data: users = [] } = useQuery({
    queryKey: queryKeys.users.list({ status: 'active' }),
    queryFn: () => api.get<AppUser[]>('/users'),
  })

  const {
    data: escalationContext,
    isFetching: isEscalationContextLoading,
    refetch: refetchEscalationContext,
  } = useQuery({
    queryKey: [...queryKeys.conversations.detail(conversationId), 'escalation-context'],
    queryFn: () => api.get<EscalationContext>(`/conversations/${conversationId}/escalation-context`),
    enabled: false,
    retry: false,
  })

  const invalidateConversation = useCallback(() => {
    queryClient.invalidateQueries({
      queryKey: queryKeys.messages.list(conversationId),
    })
    queryClient.invalidateQueries({
      queryKey: queryKeys.conversations.detail(conversationId),
    })
    queryClient.invalidateQueries({
      queryKey: queryKeys.conversations.all,
    })
  }, [conversationId, queryClient])

  // Send message mutation
  const sendMessage = useMutation({
    mutationFn: (content: string) =>
      api.post<Message>(`/conversations/${conversationId}/messages`, {
        content,
        content_type: 'text',
      }),
    onSuccess: () => {
      invalidateConversation()
    },
  })

  const assignConversation = useMutation({
    mutationFn: (userID: string) =>
      api.post<Conversation>(`/conversations/${conversationId}/assign`, { user_id: userID }),
    onSuccess: () => {
      invalidateConversation()
      setAssignDialogOpen(false)
      toast({ title: 'Conversation assigned' })
    },
    onError: (error: Error) => {
      toast({ title: 'Failed to assign conversation', description: error.message, variant: 'error' })
    },
  })

  const resolveConversation = useMutation({
    mutationFn: () => api.post<Conversation>(`/conversations/${conversationId}/resolve`),
    onSuccess: () => {
      invalidateConversation()
      toast({ title: 'Conversation resolved' })
    },
    onError: (error: Error) => {
      toast({ title: 'Failed to update conversation', description: error.message, variant: 'error' })
    },
  })

  const reopenConversation = useMutation({
    mutationFn: () => api.post<Conversation>(`/conversations/${conversationId}/reopen`),
    onSuccess: () => {
      invalidateConversation()
      toast({ title: 'Conversation reopened' })
    },
    onError: (error: Error) => {
      toast({ title: 'Failed to update conversation', description: error.message, variant: 'error' })
    },
  })

  const escalateConversation = useMutation({
    mutationFn: (payload: { reason: string; priority: 'low' | 'normal' | 'high' | 'urgent' }) =>
      api.post(`/conversations/${conversationId}/escalate`, payload),
    onSuccess: () => {
      invalidateConversation()
      setEscalateDialogOpen(false)
      refetchEscalationContext()
      toast({ title: 'Conversation escalated' })
    },
    onError: (error: Error) => {
      toast({ title: 'Failed to escalate conversation', description: error.message, variant: 'error' })
    },
  })

  const reactToMessage = useMutation({
    mutationFn: ({ messageID, emoji }: { messageID: string; emoji: string }) =>
      api.post(`/conversations/${conversationId}/messages/${messageID}/reactions`, { emoji }),
    onSuccess: () => {
      invalidateConversation()
    },
    onError: (error: Error) => {
      toast({ title: 'Failed to update reaction', description: error.message, variant: 'error' })
    },
  })

  // Handle typing indicator
  const handleTyping = useCallback(
    (isTyping: boolean) => {
      sendTyping(conversationId, isTyping)
    },
    [conversationId, sendTyping]
  )

  // Subscribe to WebSocket events
  useEffect(() => {
    // Subscribe to new messages for this conversation
    const unsubNewMessage = subscribe<WSNewMessagePayload>(
      WSEventTypes.NEW_MESSAGE,
      (payload) => {
        if (payload.conversation_id === conversationId) {
          // Invalidate queries to fetch new message
          invalidateConversation()
        }
      }
    )

    // Subscribe to typing events for this conversation
    const unsubTyping = subscribe<WSTypingPayload>(
      WSEventTypes.TYPING,
      (payload) => {
        if (payload.conversation_id === conversationId) {
          setTypingUsers((prev) => {
            if (payload.is_typing) {
              // Add user if not already typing
              if (!prev.find((u) => u.user_id === payload.user_id)) {
                return [...prev, { user_id: payload.user_id, user_name: payload.user_name }]
              }
              return prev
            } else {
              // Remove user from typing
              return prev.filter((u) => u.user_id !== payload.user_id)
            }
          })
        }
      }
    )

    return () => {
      unsubNewMessage()
      unsubTyping()
    }
  }, [conversationId, subscribe, invalidateConversation])

  // Scroll to bottom when messages change
  useEffect(() => {
    messagesEndRef.current?.scrollIntoView({ behavior: 'smooth' })
  }, [messages])

  if (conversationLoading) {
    return (
      <div className="flex-1 flex flex-col">
        <div className="flex h-16 items-center gap-3 border-b border-border bg-card px-4">
          <Skeleton className="h-10 w-10 rounded-full" />
          <div className="space-y-2">
            <Skeleton className="h-4 w-32" />
            <Skeleton className="h-3 w-24" />
          </div>
        </div>
        <div className="flex-1 p-4 space-y-4">
          {Array.from({ length: 5 }).map((_, i) => (
            <div
              key={i}
              className={cn('flex gap-2', i % 2 === 0 ? '' : 'justify-end')}
            >
              {i % 2 === 0 && <Skeleton className="h-8 w-8 rounded-full" />}
              <Skeleton
                className={cn('h-16 rounded-lg', i % 2 === 0 ? 'w-48' : 'w-56')}
              />
            </div>
          ))}
        </div>
      </div>
    )
  }

  if (!conversation) {
    return (
      <div className="flex-1 flex items-center justify-center">
        <p className="text-muted-foreground">{t('conversationNotFound')}</p>
      </div>
    )
  }

  return (
    <div className="flex-1 flex flex-col min-h-0">
      <ChatHeader
        conversation={conversation}
        onAssign={() => setAssignDialogOpen(true)}
        onEscalate={async () => {
          await refetchEscalationContext()
          setEscalateDialogOpen(true)
        }}
        onResolveOrReopen={() => {
          if (conversation.status === 'resolved') {
            reopenConversation.mutate()
            return
          }
          resolveConversation.mutate()
        }}
        onViewEscalationContext={async () => {
          await refetchEscalationContext()
          setEscalationContextOpen(true)
        }}
        isResolving={resolveConversation.isPending || reopenConversation.isPending}
        isEscalatingContextLoading={isEscalationContextLoading}
      />

      {/* Connection status indicator */}
      {connectionState !== 'connected' && (
        <div
          className={cn(
            'flex items-center justify-center gap-2 py-1 text-xs',
            connectionState === 'connecting' || connectionState === 'reconnecting'
              ? 'bg-yellow-500/10 text-yellow-500'
              : 'bg-destructive/10 text-destructive'
          )}
        >
          {connectionState === 'connecting' || connectionState === 'reconnecting' ? (
            <>
              <Wifi className="h-3 w-3 animate-pulse" />
              <span>{t('connectingStatus')}</span>
            </>
          ) : (
            <>
              <WifiOff className="h-3 w-3" />
              <span>{t('disconnectedStatus')}</span>
            </>
          )}
        </div>
      )}

      <ScrollArea className="flex-1 min-h-0 bg-background">
        <div className="p-4 space-y-4">
          {messagesLoading ? (
            Array.from({ length: 5 }).map((_, i) => (
              <div
                key={i}
                className={cn('flex gap-2', i % 2 === 0 ? '' : 'justify-end')}
              >
                {i % 2 === 0 && <Skeleton className="h-8 w-8 rounded-full" />}
                <Skeleton
                  className={cn('h-16 rounded-lg', i % 2 === 0 ? 'w-48' : 'w-56')}
                />
              </div>
            ))
          ) : sortedMessages.length > 0 ? (
            sortedMessages.map((message, i) => {
              // A date separator precedes the first message of each local day.
              const showDaySeparator =
                i === 0 ||
                dayKeyOf(sortedMessages[i - 1].created_at) !== dayKeyOf(message.created_at)
              return (
                <Fragment key={message.id}>
                  {showDaySeparator && (
                    <div className="flex items-center gap-4">
                      <Separator className="flex-1" />
                      <span className="text-xs text-muted-foreground">
                        {labelForDay(message.created_at)}
                      </span>
                      <Separator className="flex-1" />
                    </div>
                  )}
                  <MessageBubble
                    message={message}
                    isOwn={
                      message.sender_type === 'user' &&
                      message.sender_id === user?.id
                    }
                    currentUserID={user?.id}
                    onReact={(messageID, emoji) => reactToMessage.mutate({ messageID, emoji })}
                    isReacting={reactToMessage.isPending}
                  />
                </Fragment>
              )
            })
          ) : (
            <div className="py-8 text-center text-muted-foreground">
              <p className="text-sm">{t('noMessagesYet')}</p>
              <p className="text-xs">{t('startByMessage')}</p>
            </div>
          )}

          <div ref={messagesEndRef} />
        </div>
      </ScrollArea>

      {/* Typing indicator */}
      <TypingIndicator typingUsers={typingUsers} />

      <Composer
        conversationId={conversationId}
        onSend={(content) => sendMessage.mutate(content)}
        onTyping={handleTyping}
        isSending={sendMessage.isPending}
      />

      <AssignConversationDialog
        conversation={conversation}
        users={users.filter((candidate) => candidate.status === 'active')}
        open={assignDialogOpen}
        onOpenChange={setAssignDialogOpen}
        onAssign={(userID) => assignConversation.mutate(userID)}
        isSubmitting={assignConversation.isPending}
      />

      <EscalateConversationDialog
        open={escalateDialogOpen}
        onOpenChange={setEscalateDialogOpen}
        onEscalate={(payload) => escalateConversation.mutate(payload)}
        context={escalationContext}
        isLoadingContext={isEscalationContextLoading}
        isSubmitting={escalateConversation.isPending}
      />

      <EscalationContextDialog
        open={escalationContextOpen}
        onOpenChange={setEscalationContextOpen}
        context={escalationContext}
        isLoading={isEscalationContextLoading}
      />
    </div>
  )
}
