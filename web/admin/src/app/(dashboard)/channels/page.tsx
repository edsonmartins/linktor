'use client'

import { useState } from 'react'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import {
  Plus,
  Radio,
  MessageSquare,
  Settings,
  MoreVertical,
  Wifi,
  WifiOff,
  AlertTriangle,
  Trash2,
  Power,
  PowerOff,
} from 'lucide-react'
import { Header } from '@/components/layout/header'
import { Button } from '@/components/ui/button'
import { Badge } from '@/components/ui/badge'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Skeleton } from '@/components/ui/skeleton'
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu'
import {
  Sheet,
  SheetContent,
  SheetDescription,
  SheetHeader,
  SheetTitle,
} from '@/components/ui/sheet'
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
import { cn } from '@/lib/utils'
import { api } from '@/lib/api'
import { queryKeys } from '@/lib/query'
import { useToast } from '@/hooks/use-toast'
import type { Channel, ChannelType } from '@/types'

// Channel config components
import { WebchatConfig } from './webchat-config'
import { WhatsAppConfig } from './whatsapp-config'
import { WhatsAppUnofficialConfig } from './whatsapp-unofficial-config'
import { TelegramConfig } from './telegram-config'
import { SMSConfig } from './sms-config'
import { FacebookConfig } from './facebook-config'
import { InstagramConfig } from './instagram-config'
import { EmailConfig } from './email-config'
import { RCSConfig } from './rcs-config'
import { VoiceConfig } from './voice-config'

/**
 * Channel type configurations
 */
const channelConfigs: Record<
  ChannelType,
  {
    label: string
    description: string
    icon: React.ReactNode
    color: string
    bgColor: string
  }
> = {
  webchat: {
    label: 'Web Chat',
    description: 'Embeddable widget for your website',
    icon: <MessageSquare className="h-6 w-6" />,
    color: 'text-primary',
    bgColor: 'bg-primary/10',
  },
  whatsapp: {
    label: 'WhatsApp',
    description: 'Unofficial WhatsApp integration',
    icon: <MessageSquare className="h-6 w-6" />,
    color: 'text-green-500',
    bgColor: 'bg-green-500/10',
  },
  whatsapp_official: {
    label: 'WhatsApp Official',
    description: 'Meta Business Cloud API',
    icon: <MessageSquare className="h-6 w-6" />,
    color: 'text-green-600',
    bgColor: 'bg-green-600/10',
  },
  telegram: {
    label: 'Telegram',
    description: 'Bot API integration',
    icon: <MessageSquare className="h-6 w-6" />,
    color: 'text-blue-500',
    bgColor: 'bg-blue-500/10',
  },
  sms: {
    label: 'SMS',
    description: 'Twilio SMS integration',
    icon: <MessageSquare className="h-6 w-6" />,
    color: 'text-purple-500',
    bgColor: 'bg-purple-500/10',
  },
  instagram: {
    label: 'Instagram',
    description: 'Meta Graph API integration',
    icon: <MessageSquare className="h-6 w-6" />,
    color: 'text-pink-500',
    bgColor: 'bg-pink-500/10',
  },
  facebook: {
    label: 'Facebook Messenger',
    description: 'Meta Messenger Platform',
    icon: <MessageSquare className="h-6 w-6" />,
    color: 'text-blue-600',
    bgColor: 'bg-blue-600/10',
  },
  rcs: {
    label: 'RCS',
    description: 'Rich Communication Services',
    icon: <MessageSquare className="h-6 w-6" />,
    color: 'text-orange-500',
    bgColor: 'bg-orange-500/10',
  },
  email: {
    label: 'Email',
    description: 'Multi-provider email integration',
    icon: <MessageSquare className="h-6 w-6" />,
    color: 'text-amber-500',
    bgColor: 'bg-amber-500/10',
  },
  voice: {
    label: 'Voice',
    description: 'VoIP with IVR support',
    icon: <MessageSquare className="h-6 w-6" />,
    color: 'text-cyan-500',
    bgColor: 'bg-cyan-500/10',
  },
}

/**
 * Status Badge
 */
function StatusBadge({ status }: { status: Channel['status'] }) {
  const config = {
    active: {
      variant: 'success' as const,
      icon: <Wifi className="h-3 w-3" />,
      label: 'Active',
    },
    inactive: {
      variant: 'secondary' as const,
      icon: <WifiOff className="h-3 w-3" />,
      label: 'Inactive',
    },
    error: {
      variant: 'error' as const,
      icon: <AlertTriangle className="h-3 w-3" />,
      label: 'Error',
    },
  }

  const { variant, icon, label } = config[status]

  return (
    <Badge variant={variant} className="gap-1">
      {icon}
      {label}
    </Badge>
  )
}

/**
 * Channel Card Component
 */
function ChannelCard({
  channel,
  onConfigure,
  onToggleStatus,
  onDelete,
}: {
  channel: Channel
  onConfigure: () => void
  onToggleStatus: () => void
  onDelete: () => void
}) {
  const config = channelConfigs[channel.type] || channelConfigs.webchat

  return (
    <Card className="hover:border-primary/30 transition-colors">
      <CardHeader className="pb-3">
        <div className="flex items-start justify-between">
          <div className="flex items-center gap-3">
            <div
              className={cn(
                'flex h-12 w-12 items-center justify-center rounded-lg',
                config.bgColor,
                config.color
              )}
            >
              {config.icon}
            </div>
            <div>
              <CardTitle className="text-base">{channel.name}</CardTitle>
              <CardDescription className="text-xs">
                {config.description}
              </CardDescription>
            </div>
          </div>
          <DropdownMenu>
            <DropdownMenuTrigger asChild>
              <Button variant="ghost" size="icon" className="h-8 w-8">
                <MoreVertical className="h-4 w-4" />
              </Button>
            </DropdownMenuTrigger>
            <DropdownMenuContent align="end">
              <DropdownMenuItem onClick={onConfigure}>
                <Settings className="h-4 w-4 mr-2" />
                Configure
              </DropdownMenuItem>
              <DropdownMenuItem onClick={onToggleStatus}>
                {channel.status === 'active' ? (
                  <>
                    <PowerOff className="h-4 w-4 mr-2" />
                    Deactivate
                  </>
                ) : (
                  <>
                    <Power className="h-4 w-4 mr-2" />
                    Activate
                  </>
                )}
              </DropdownMenuItem>
              <DropdownMenuSeparator />
              <DropdownMenuItem className="text-destructive" onClick={onDelete}>
                <Trash2 className="h-4 w-4 mr-2" />
                Delete channel
              </DropdownMenuItem>
            </DropdownMenuContent>
          </DropdownMenu>
        </div>
      </CardHeader>
      <CardContent className="pt-0">
        <div className="flex items-center justify-between">
          <StatusBadge status={channel.status} />
          <Badge variant="outline" className="font-mono text-xs">
            {channel.type}
          </Badge>
        </div>
      </CardContent>
    </Card>
  )
}

/**
 * Available Channel Type Card
 */
function AvailableChannelCard({
  type,
  disabled,
  onClick,
}: {
  type: ChannelType
  disabled?: boolean
  onClick?: () => void
}) {
  const config = channelConfigs[type]

  return (
    <Card
      className={cn(
        'transition-colors',
        disabled
          ? 'opacity-50 cursor-not-allowed'
          : 'hover:border-primary/30 cursor-pointer'
      )}
      onClick={disabled ? undefined : onClick}
    >
      <CardContent className="p-4">
        <div className="flex items-center gap-3">
          <div
            className={cn(
              'flex h-10 w-10 items-center justify-center rounded-lg',
              config.bgColor,
              config.color
            )}
          >
            {config.icon}
          </div>
          <div className="flex-1">
            <h4 className="text-sm font-medium">{config.label}</h4>
            <p className="text-xs text-muted-foreground">{config.description}</p>
          </div>
          {disabled ? (
            <Badge variant="secondary" className="text-[10px]">
              Coming soon
            </Badge>
          ) : (
            <Plus className="h-4 w-4 text-muted-foreground" />
          )}
        </div>
      </CardContent>
    </Card>
  )
}

/**
 * Channel Config Sheet - renders the appropriate config component
 */
function ChannelConfigSheet({
  open,
  onOpenChange,
  channelType,
  channel,
  onSuccess,
}: {
  open: boolean
  onOpenChange: (open: boolean) => void
  channelType: ChannelType | null
  channel?: Channel
  onSuccess: () => void
}) {
  if (!channelType) return null

  const config = channelConfigs[channelType]
  const isEditing = !!channel

  const handleSuccess = () => {
    onSuccess()
    onOpenChange(false)
  }

  const handleCancel = () => {
    onOpenChange(false)
  }

  const renderConfigComponent = () => {
    // Common props for most configs
    const commonProps = {
      channel,
      onSuccess: handleSuccess,
      onCancel: handleCancel,
    }

    switch (channelType) {
      case 'webchat':
        return <WebchatConfig {...commonProps} />
      case 'whatsapp_official':
        return <WhatsAppConfig {...commonProps} />
      case 'whatsapp':
        return <WhatsAppUnofficialConfig {...commonProps} />
      case 'telegram':
        return <TelegramConfig {...commonProps} />
      case 'sms':
        return <SMSConfig {...commonProps} />
      case 'facebook':
        return <FacebookConfig {...commonProps} />
      case 'instagram':
        return <InstagramConfig {...commonProps} />
      case 'email':
        return <EmailConfig channelId={channel?.id} onSuccess={handleSuccess} />
      case 'rcs':
        return (
          <RCSConfig
            channelId={channel?.id}
            onSave={() => handleSuccess()}
          />
        )
      case 'voice':
        return <VoiceConfig channel={channel} onClose={handleCancel} />
      default:
        return <p>Configuration not available for this channel type.</p>
    }
  }

  return (
    <Sheet open={open} onOpenChange={onOpenChange}>
      <SheetContent className="sm:max-w-xl flex flex-col h-full p-0">
        <SheetHeader className="px-6 pt-6 pb-4 border-b">
          <SheetTitle className="flex items-center gap-2">
            <div className={cn('p-2 rounded-lg', config.bgColor, config.color)}>
              {config.icon}
            </div>
            {isEditing ? `Configure ${config.label}` : `Add ${config.label}`}
          </SheetTitle>
          <SheetDescription>
            {isEditing
              ? `Update your ${config.label} channel settings`
              : `Set up a new ${config.label} channel`}
          </SheetDescription>
        </SheetHeader>
        <div className="flex-1 overflow-y-auto px-6 py-4">
          {renderConfigComponent()}
        </div>
      </SheetContent>
    </Sheet>
  )
}

/**
 * Channels Page
 */
export default function ChannelsPage() {
  const queryClient = useQueryClient()
  const { toast } = useToast()

  // State for dialogs
  const [configSheetOpen, setConfigSheetOpen] = useState(false)
  const [selectedChannelType, setSelectedChannelType] = useState<ChannelType | null>(null)
  const [selectedChannel, setSelectedChannel] = useState<Channel | undefined>()
  const [deleteDialogOpen, setDeleteDialogOpen] = useState(false)
  const [channelToDelete, setChannelToDelete] = useState<Channel | null>(null)

  // Fetch channels
  const { data, isLoading } = useQuery({
    queryKey: queryKeys.channels.list(),
    queryFn: () => api.get<{ data: Channel[] }>('/channels'),
  })

  const channels = data?.data || []

  // Delete mutation
  const deleteMutation = useMutation({
    mutationFn: (id: string) => api.delete(`/channels/${id}`),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: queryKeys.channels.all })
      toast({
        title: 'Channel deleted',
        description: 'The channel has been removed.',
      })
      setDeleteDialogOpen(false)
      setChannelToDelete(null)
    },
    onError: (error: Error) => {
      toast({
        title: 'Error',
        description: error.message || 'Failed to delete channel.',
        variant: 'error',
      })
    },
  })

  // Toggle status mutation
  const toggleStatusMutation = useMutation({
    mutationFn: ({ id, status }: { id: string; status: 'active' | 'inactive' }) =>
      api.put(`/channels/${id}`, { status }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: queryKeys.channels.all })
      toast({
        title: 'Status updated',
        description: 'Channel status has been updated.',
      })
    },
    onError: (error: Error) => {
      toast({
        title: 'Error',
        description: error.message || 'Failed to update channel status.',
        variant: 'error',
      })
    },
  })

  // Handlers
  const handleAddChannel = (type: ChannelType) => {
    setSelectedChannelType(type)
    setSelectedChannel(undefined)
    setConfigSheetOpen(true)
  }

  const handleConfigureChannel = (channel: Channel) => {
    setSelectedChannelType(channel.type)
    setSelectedChannel(channel)
    setConfigSheetOpen(true)
  }

  const handleToggleStatus = (channel: Channel) => {
    const newStatus = channel.status === 'active' ? 'inactive' : 'active'
    toggleStatusMutation.mutate({ id: channel.id, status: newStatus })
  }

  const handleDeleteChannel = (channel: Channel) => {
    setChannelToDelete(channel)
    setDeleteDialogOpen(true)
  }

  const confirmDelete = () => {
    if (channelToDelete) {
      deleteMutation.mutate(channelToDelete.id)
    }
  }

  const handleConfigSuccess = () => {
    queryClient.invalidateQueries({ queryKey: queryKeys.channels.all })
  }

  return (
    <div className="flex flex-col h-full">
      <Header title="Channels" />

      <div className="p-6 space-y-6 overflow-auto">
        {/* Active Channels */}
        <section>
          <div className="flex items-center justify-between mb-4">
            <div>
              <h2 className="text-lg font-semibold">Active Channels</h2>
              <p className="text-sm text-muted-foreground">
                Manage your connected communication channels
              </p>
            </div>
          </div>

          {isLoading ? (
            <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-3">
              {Array.from({ length: 3 }).map((_, i) => (
                <Card key={i}>
                  <CardHeader className="pb-3">
                    <div className="flex items-start gap-3">
                      <Skeleton className="h-12 w-12 rounded-lg" />
                      <div className="space-y-2">
                        <Skeleton className="h-4 w-24" />
                        <Skeleton className="h-3 w-32" />
                      </div>
                    </div>
                  </CardHeader>
                  <CardContent className="pt-0">
                    <Skeleton className="h-6 w-16" />
                  </CardContent>
                </Card>
              ))}
            </div>
          ) : channels.length > 0 ? (
            <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-3">
              {channels.map((channel) => (
                <ChannelCard
                  key={channel.id}
                  channel={channel}
                  onConfigure={() => handleConfigureChannel(channel)}
                  onToggleStatus={() => handleToggleStatus(channel)}
                  onDelete={() => handleDeleteChannel(channel)}
                />
              ))}
            </div>
          ) : (
            <Card className="border-dashed">
              <CardContent className="py-8 text-center">
                <Radio className="mx-auto h-12 w-12 text-muted-foreground opacity-50" />
                <p className="mt-4 text-lg font-medium">No channels configured</p>
                <p className="text-sm text-muted-foreground">
                  Add your first channel to start receiving messages
                </p>
              </CardContent>
            </Card>
          )}
        </section>

        {/* Available Channels */}
        <section>
          <div className="mb-4">
            <h2 className="text-lg font-semibold">Add New Channel</h2>
            <p className="text-sm text-muted-foreground">
              Connect new communication channels to your account
            </p>
          </div>

          <div className="grid gap-3 md:grid-cols-2 lg:grid-cols-4">
            <AvailableChannelCard type="webchat" onClick={() => handleAddChannel('webchat')} />
            <AvailableChannelCard type="whatsapp_official" onClick={() => handleAddChannel('whatsapp_official')} />
            <AvailableChannelCard type="telegram" onClick={() => handleAddChannel('telegram')} />
            <AvailableChannelCard type="sms" onClick={() => handleAddChannel('sms')} />
            <AvailableChannelCard type="facebook" onClick={() => handleAddChannel('facebook')} />
            <AvailableChannelCard type="instagram" onClick={() => handleAddChannel('instagram')} />
            <AvailableChannelCard type="email" onClick={() => handleAddChannel('email')} />
            <AvailableChannelCard type="whatsapp" onClick={() => handleAddChannel('whatsapp')} />
            <AvailableChannelCard type="rcs" onClick={() => handleAddChannel('rcs')} />
            <AvailableChannelCard type="voice" onClick={() => handleAddChannel('voice')} />
          </div>
        </section>
      </div>

      {/* Channel Config Sheet */}
      <ChannelConfigSheet
        open={configSheetOpen}
        onOpenChange={setConfigSheetOpen}
        channelType={selectedChannelType}
        channel={selectedChannel}
        onSuccess={handleConfigSuccess}
      />

      {/* Delete Confirmation Dialog */}
      <AlertDialog open={deleteDialogOpen} onOpenChange={setDeleteDialogOpen}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>Delete Channel</AlertDialogTitle>
            <AlertDialogDescription>
              Are you sure you want to delete "{channelToDelete?.name}"? This action cannot be undone.
              All conversations and messages associated with this channel will be preserved but the channel will no longer receive new messages.
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel>Cancel</AlertDialogCancel>
            <AlertDialogAction
              onClick={confirmDelete}
              className="bg-destructive text-destructive-foreground hover:bg-destructive/90"
            >
              Delete
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </div>
  )
}
