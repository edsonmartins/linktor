'use client'

import { useState, useRef, useEffect, useCallback } from 'react'
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
  X,
  Loader2,
  FileText,
  Download,
} from 'lucide-react'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Avatar } from '@/components/ui/avatar'
import { Badge } from '@/components/ui/badge'
import { EnvironmentBadge, SandboxConversationBanner } from '@/components/environment-badge'
import { MessageFailureDetail } from '@/components/message-failure-detail'
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
import { Popover, PopoverContent, PopoverTrigger } from '@/components/ui/popover'
import { EmojiPicker } from '@/components/emoji-picker'
import { cn, formatDate, formatRelativeTime } from '@/lib/utils'
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
  MessageAttachment,
  MessageStatus,
  User as AppUser,
} from '@/types'

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
 * Renders a single message attachment: an inline preview for images, an inline
 * player for video/audio, and a downloadable chip for everything else.
 */
function MessageAttachmentView({ attachment }: { attachment: MessageAttachment }) {
  if (attachment.type === 'image') {
    return (
      <a href={attachment.url} target="_blank" rel="noopener noreferrer" className="block">
        {/* eslint-disable-next-line @next/next/no-img-element */}
        <img
          src={attachment.url}
          alt={attachment.filename || 'image'}
          className="max-h-60 max-w-full rounded-md object-contain"
        />
      </a>
    )
  }

  if (attachment.type === 'video') {
    return (
      <video
        src={attachment.url}
        controls
        preload="metadata"
        className="max-h-60 max-w-full rounded-md"
      />
    )
  }

  if (attachment.type === 'audio') {
    return <audio src={attachment.url} controls preload="metadata" className="max-w-full" />
  }

  return (
    <a
      href={attachment.url}
      target="_blank"
      rel="noopener noreferrer"
      className="flex items-center gap-2 rounded-md border border-border/60 bg-background/40 px-2 py-1.5 text-xs hover:bg-background/70"
    >
      <FileText className="h-4 w-4 shrink-0" />
      <span className="max-w-[180px] truncate">{attachment.filename || attachment.url}</span>
      <Download className="ml-auto h-3.5 w-3.5 shrink-0 opacity-70" />
    </a>
  )
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
      <div className={cn('space-y-0.5', isOwn && 'items-end')}>
        <div
          className={cn(
            'space-y-2 rounded-lg px-3 py-1.5',
            isOwn ? 'message-outgoing' : 'message-incoming'
          )}
        >
          {message.attachments && message.attachments.length > 0 && (
            <div className="space-y-2">
              {message.attachments.map((attachment) => (
                <MessageAttachmentView key={attachment.id} attachment={attachment} />
              ))}
            </div>
          )}
          {message.content && (
            <p className="text-sm whitespace-pre-wrap">{message.content}</p>
          )}
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
        {/* WP-K: on failure, distinguish local guard block from provider
            rejection with an actionable reason. */}
        {isOwn && <MessageFailureDetail message={message} />}
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
          fallback={(conversation.contact?.name && conversation.contact.name !== 'Unknown' ? conversation.contact.name : '') || 'U'}
          status={conversation.status === 'open' ? 'online' : 'offline'}
        />
        <div>
          <h2 className="font-medium">
            {(conversation.contact?.name && conversation.contact.name !== 'Unknown'
              ? conversation.contact.name
              : '') || t('unknownContact')}
          </h2>
          <div className="flex items-center gap-2 text-xs text-muted-foreground">
            <Badge
              variant={conversation.channel?.type as 'webchat' | undefined || 'secondary'}
              className="text-[10px] px-1.5 py-0"
              title={conversation.channel?.type}
            >
              {conversation.channel?.name || conversation.channel?.type}
            </Badge>
            <EnvironmentBadge
              environment={conversation.environment}
              className="text-[10px] px-1.5 py-0"
            />
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
interface UploadedAttachment {
  url: string
  type: string
  filename: string
  mime_type: string
  size_bytes: number
}

interface ComposerProps {
  conversationId: string
  onSend: (content: string, attachments: UploadedAttachment[]) => void
  onTyping: (isTyping: boolean) => void
  isSending: boolean
}

function Composer({ conversationId, onSend, onTyping, isSending }: ComposerProps) {
  const t = useTranslations('conversations')
  const { toast } = useToast()
  const [content, setContent] = useState('')
  const [attachments, setAttachments] = useState<UploadedAttachment[]>([])
  const [uploading, setUploading] = useState(false)
  const typingTimeoutRef = useRef<NodeJS.Timeout | null>(null)
  const fileInputRef = useRef<HTMLInputElement | null>(null)

  const canSend = (content.trim().length > 0 || attachments.length > 0) && !isSending && !uploading

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault()
    if (!canSend) return
    onTyping(false)
    onSend(content.trim(), attachments)
    setContent('')
    setAttachments([])
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

  const handleFiles = async (e: React.ChangeEvent<HTMLInputElement>) => {
    const files = e.target.files
    if (!files || files.length === 0) return
    setUploading(true)
    try {
      for (const file of Array.from(files)) {
        const formData = new FormData()
        formData.append('file', file)
        const uploaded = await api.upload<UploadedAttachment>(
          `/conversations/${conversationId}/attachments`,
          formData
        )
        setAttachments((prev) => [...prev, uploaded])
      }
    } catch (error) {
      toast({
        title: t('attachFailed'),
        description: error instanceof Error ? error.message : undefined,
        variant: 'error',
      })
    } finally {
      setUploading(false)
      if (fileInputRef.current) fileInputRef.current.value = ''
    }
  }

  const removeAttachment = (index: number) => {
    setAttachments((prev) => prev.filter((_, i) => i !== index))
  }

  return (
    <form
      onSubmit={handleSubmit}
      className="border-t border-border bg-card p-4"
    >
      {attachments.length > 0 && (
        <div className="mb-2 flex flex-wrap gap-2">
          {attachments.map((att, index) => (
            <div
              key={att.url}
              className="flex items-center gap-2 rounded-md border border-border bg-secondary/50 px-2 py-1 text-xs"
            >
              {att.type === 'image' ? (
                // eslint-disable-next-line @next/next/no-img-element
                <img
                  src={att.url}
                  alt={att.filename}
                  className="h-8 w-8 rounded object-cover"
                />
              ) : (
                <FileText className="h-4 w-4 shrink-0 text-muted-foreground" />
              )}
              <span className="max-w-[140px] truncate">{att.filename}</span>
              <button
                type="button"
                onClick={() => removeAttachment(index)}
                className="text-muted-foreground hover:text-foreground"
                aria-label={t('removeAttachment')}
              >
                <X className="h-3.5 w-3.5" />
              </button>
            </div>
          ))}
        </div>
      )}
      <div className="flex items-end gap-2">
        <input
          ref={fileInputRef}
          type="file"
          multiple
          className="hidden"
          onChange={handleFiles}
        />
        <SimpleTooltip content={t('attachFile')}>
          <Button
            type="button"
            variant="ghost"
            size="icon"
            onClick={() => fileInputRef.current?.click()}
            disabled={uploading}
            aria-label={t('attachFile')}
          >
            {uploading ? (
              <Loader2 className="h-5 w-5 animate-spin" />
            ) : (
              <Paperclip className="h-5 w-5" />
            )}
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

        <Popover>
          <PopoverTrigger asChild>
            <Button
              type="button"
              variant="ghost"
              size="icon"
              aria-label={t('emoji')}
            >
              <Smile className="h-5 w-5" />
            </Button>
          </PopoverTrigger>
          <PopoverContent
            side="top"
            align="end"
            className="w-auto border-0 bg-transparent p-0 shadow-none"
          >
            <EmojiPicker onSelect={(emoji) => setContent((c) => c + emoji)} />
          </PopoverContent>
        </Popover>

        <Button
          type="submit"
          size="icon"
          disabled={!canSend}
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

  // The API returns the latest N messages newest-first (created_at desc, id desc)
  // so pagination fetches the most recent window. Reverse for display so the
  // thread reads chronologically — oldest at top, newest at the bottom. Reverse
  // (not a created_at sort) preserves the backend's stable id tiebreaker for
  // messages sharing a second.
  const messages = [...(messagesData?.data || [])].reverse()

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
    mutationFn: (payload: { content: string; attachments: UploadedAttachment[] }) =>
      api.post<Message>(`/conversations/${conversationId}/messages`, {
        content: payload.content,
        content_type:
          payload.attachments.length > 0 ? payload.attachments[0].type : 'text',
        attachments: payload.attachments,
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

  // Scroll to bottom when the conversation changes or a message arrives.
  // Scroll only the ScrollArea viewport — NOT scrollIntoView, which also scrolls
  // every scrollable ancestor (the dashboard <main>), shifting the whole page up
  // and leaving a blank gap. Key on stable primitives so it doesn't refire on the
  // messages array identity churn during refetch.
  useEffect(() => {
    const viewport = messagesEndRef.current?.closest(
      '[data-radix-scroll-area-viewport]'
    ) as HTMLElement | null | undefined
    if (viewport) {
      viewport.scrollTop = viewport.scrollHeight
    }
  }, [conversationId, messages.length])

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
      {/* Persistent, non-dismissable sandbox marking (WP-H / INV-018): stays
          visible for the whole session while the conversation is open. */}
      <SandboxConversationBanner environment={conversation.environment} />
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
        <div className="p-4 space-y-1">
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
        onSend={(content, attachments) => sendMessage.mutate({ content, attachments })}
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
