'use client'

import { useState, useRef, useEffect, useCallback } from 'react'
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
  Bot,
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
import { SimpleTooltip } from '@/components/ui/tooltip'
import { cn, formatDate, formatRelativeTime } from '@/lib/utils'
import { api } from '@/lib/api'
import { queryKeys } from '@/lib/query'
import { useUser } from '@/stores/auth-store'
import {
  useWebSocketContext,
  WSEventTypes,
  type WSNewMessagePayload,
  type WSTypingPayload,
} from '@/hooks/use-websocket'
import type { Conversation, Message, MessageStatus } from '@/types'

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
  // Only show badge for outgoing messages from non-API sources
  if (!isOwn) return null

  // Check if it's an imported message
  if (isImported || source === 'imported') {
    return (
      <SimpleTooltip content="Imported from chat history">
        <Badge variant="outline" className="text-[8px] px-1 py-0 h-3.5 gap-0.5">
          <History className="h-2 w-2" />
          <span>History</span>
        </Badge>
      </SimpleTooltip>
    )
  }

  // Check if it's from Business App (echo)
  if (source === 'business_app') {
    return (
      <SimpleTooltip content="Sent via WhatsApp Business App">
        <Badge variant="outline" className="text-[8px] px-1 py-0 h-3.5 gap-0.5">
          <Smartphone className="h-2 w-2" />
          <span>App</span>
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
}

function MessageBubble({ message, isOwn }: MessageBubbleProps) {
  return (
    <div
      className={cn(
        'flex gap-2 max-w-[70%]',
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
}

function ChatHeader({ conversation }: ChatHeaderProps) {
  return (
    <div className="flex h-16 items-center justify-between border-b border-border bg-card px-4">
      <div className="flex items-center gap-3">
        <Avatar
          fallback={conversation.contact?.name || 'U'}
          status={conversation.status === 'open' ? 'online' : 'offline'}
        />
        <div>
          <h2 className="font-medium">
            {conversation.contact?.name || 'Unknown Contact'}
          </h2>
          <div className="flex items-center gap-2 text-xs text-muted-foreground">
            <Badge
              variant={conversation.channel?.type as 'webchat' | undefined || 'secondary'}
              className="text-[10px] px-1.5 py-0"
            >
              {conversation.channel?.type}
            </Badge>
            <span>
              {conversation.contact?.phone || conversation.contact?.email || 'No contact info'}
            </span>
          </div>
        </div>
      </div>

      <div className="flex items-center gap-1">
        <SimpleTooltip content="Voice call">
          <Button variant="ghost" size="icon">
            <Phone className="h-4 w-4" />
          </Button>
        </SimpleTooltip>
        <SimpleTooltip content="Video call">
          <Button variant="ghost" size="icon">
            <Video className="h-4 w-4" />
          </Button>
        </SimpleTooltip>
        <SimpleTooltip content="Contact info">
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
            <DropdownMenuItem>Resolve conversation</DropdownMenuItem>
            <DropdownMenuItem>Assign to agent</DropdownMenuItem>
            <DropdownMenuItem>Snooze</DropdownMenuItem>
            <DropdownMenuSeparator />
            <DropdownMenuItem className="text-destructive">
              Delete conversation
            </DropdownMenuItem>
          </DropdownMenuContent>
        </DropdownMenu>
      </div>
    </div>
  )
}

/**
 * Typing Indicator Component
 */
interface TypingIndicatorProps {
  typingUsers: { user_id: string; user_name: string }[]
}

function TypingIndicator({ typingUsers }: TypingIndicatorProps) {
  if (typingUsers.length === 0) return null

  const names = typingUsers.map((u) => u.user_name || 'Someone').join(', ')
  return (
    <div className="flex items-center gap-2 px-4 py-2 text-xs text-muted-foreground animate-pulse">
      <div className="flex gap-1">
        <span className="w-1.5 h-1.5 rounded-full bg-muted-foreground animate-bounce [animation-delay:-0.3s]" />
        <span className="w-1.5 h-1.5 rounded-full bg-muted-foreground animate-bounce [animation-delay:-0.15s]" />
        <span className="w-1.5 h-1.5 rounded-full bg-muted-foreground animate-bounce" />
      </div>
      <span>{names} {typingUsers.length === 1 ? 'is' : 'are'} typing...</span>
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
        <SimpleTooltip content="Attach file">
          <Button type="button" variant="ghost" size="icon">
            <Paperclip className="h-5 w-5" />
          </Button>
        </SimpleTooltip>

        <div className="flex-1">
          <Input
            placeholder="Type a message..."
            value={content}
            onChange={handleChange}
            onKeyDown={handleKeyDown}
            variant="terminal"
            className="min-h-[40px]"
          />
        </div>

        <SimpleTooltip content="Emoji">
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
  const queryClient = useQueryClient()
  const user = useUser()
  const messagesEndRef = useRef<HTMLDivElement>(null)
  const [typingUsers, setTypingUsers] = useState<{ user_id: string; user_name: string }[]>([])

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
      api.get<{ data: Message[] }>(`/conversations/${conversationId}/messages`),
  })

  const messages = messagesData?.data || []

  // Send message mutation
  const sendMessage = useMutation({
    mutationFn: (content: string) =>
      api.post<Message>(`/conversations/${conversationId}/messages`, {
        content,
        content_type: 'text',
      }),
    onSuccess: () => {
      // Refetch messages after sending
      queryClient.invalidateQueries({
        queryKey: queryKeys.messages.list(conversationId),
      })
      queryClient.invalidateQueries({
        queryKey: queryKeys.conversations.detail(conversationId),
      })
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
          queryClient.invalidateQueries({
            queryKey: queryKeys.messages.list(conversationId),
          })
          queryClient.invalidateQueries({
            queryKey: queryKeys.conversations.detail(conversationId),
          })
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
  }, [conversationId, subscribe, queryClient])

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
        <p className="text-muted-foreground">Conversation not found</p>
      </div>
    )
  }

  return (
    <div className="flex-1 flex flex-col">
      <ChatHeader conversation={conversation} />

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
              <span>Connecting...</span>
            </>
          ) : (
            <>
              <WifiOff className="h-3 w-3" />
              <span>Disconnected - Messages may be delayed</span>
            </>
          )}
        </div>
      )}

      <ScrollArea className="flex-1 bg-background">
        <div className="p-4 space-y-4">
          {/* Date separator */}
          <div className="flex items-center gap-4">
            <Separator className="flex-1" />
            <span className="text-xs text-muted-foreground">
              {formatDate(conversation.created_at)}
            </span>
            <Separator className="flex-1" />
          </div>

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
          ) : messages.length > 0 ? (
            messages.map((message) => (
              <MessageBubble
                key={message.id}
                message={message}
                isOwn={
                  message.sender_type === 'user' &&
                  message.sender_id === user?.id
                }
              />
            ))
          ) : (
            <div className="py-8 text-center text-muted-foreground">
              <p className="text-sm">No messages yet</p>
              <p className="text-xs">Start the conversation by sending a message</p>
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
    </div>
  )
}
