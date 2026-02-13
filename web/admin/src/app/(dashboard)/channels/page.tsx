'use client'

import { useState } from 'react'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { useTranslations } from 'next-intl'
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
  RefreshCw,
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
import { toastSuccess, toastError } from '@/hooks/use-toast'
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
 * Channel type icon config
 */
const channelIcons: Record<ChannelType, { color: string; bgColor: string }> = {
  webchat: { color: 'text-primary', bgColor: 'bg-primary/10' },
  whatsapp: { color: 'text-green-500', bgColor: 'bg-green-500/10' },
  whatsapp_official: { color: 'text-green-600', bgColor: 'bg-green-600/10' },
  telegram: { color: 'text-blue-500', bgColor: 'bg-blue-500/10' },
  sms: { color: 'text-purple-500', bgColor: 'bg-purple-500/10' },
  instagram: { color: 'text-pink-500', bgColor: 'bg-pink-500/10' },
  facebook: { color: 'text-blue-600', bgColor: 'bg-blue-600/10' },
  rcs: { color: 'text-orange-500', bgColor: 'bg-orange-500/10' },
  email: { color: 'text-amber-500', bgColor: 'bg-amber-500/10' },
  voice: { color: 'text-cyan-500', bgColor: 'bg-cyan-500/10' },
}

/**
 * Status Badge
 */
function StatusBadge({ status, tCommon }: { status: Channel['status']; tCommon: (key: string) => string }) {
  const config = {
    active: {
      variant: 'success' as const,
      icon: <Wifi className="h-3 w-3" />,
      labelKey: 'active',
    },
    inactive: {
      variant: 'secondary' as const,
      icon: <WifiOff className="h-3 w-3" />,
      labelKey: 'inactive',
    },
    error: {
      variant: 'error' as const,
      icon: <AlertTriangle className="h-3 w-3" />,
      labelKey: 'error',
    },
  }

  const { variant, icon, labelKey } = config[status]

  return (
    <Badge variant={variant} className="gap-1">
      {icon}
      {tCommon(labelKey)}
    </Badge>
  )
}

/**
 * Channel Card Component
 */
function ChannelCard({
  channel,
  t,
  tCommon,
  onConfigure,
  onToggleStatus,
  onDelete,
}: {
  channel: Channel
  t: (key: string) => string
  tCommon: (key: string) => string
  onConfigure: () => void
  onToggleStatus: () => void
  onDelete: () => void
}) {
  const iconConfig = channelIcons[channel.type] || channelIcons.webchat

  return (
    <Card className="hover:border-primary/30 transition-colors">
      <CardHeader className="pb-3">
        <div className="flex items-start justify-between">
          <div className="flex items-center gap-3">
            <div
              className={cn(
                'flex h-12 w-12 items-center justify-center rounded-lg',
                iconConfig.bgColor,
                iconConfig.color
              )}
            >
              <MessageSquare className="h-6 w-6" />
            </div>
            <div>
              <CardTitle className="text-base">{channel.name}</CardTitle>
              <CardDescription className="text-xs">
                {t(`descriptions.${channel.type}`)}
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
                {t('configure')}
              </DropdownMenuItem>
              <DropdownMenuItem onClick={onToggleStatus}>
                {channel.status === 'active' ? (
                  <>
                    <PowerOff className="h-4 w-4 mr-2" />
                    {t('deactivate')}
                  </>
                ) : (
                  <>
                    <Power className="h-4 w-4 mr-2" />
                    {t('activate')}
                  </>
                )}
              </DropdownMenuItem>
              <DropdownMenuSeparator />
              <DropdownMenuItem className="text-destructive" onClick={onDelete}>
                <Trash2 className="h-4 w-4 mr-2" />
                {t('deleteChannel')}
              </DropdownMenuItem>
            </DropdownMenuContent>
          </DropdownMenu>
        </div>
      </CardHeader>
      <CardContent className="pt-0">
        <div className="flex items-center justify-between">
          <StatusBadge status={channel.status} tCommon={tCommon} />
          <Badge variant="outline" className="font-mono text-xs">
            {t(`types.${channel.type}`)}
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
  t,
  disabled,
  onClick,
}: {
  type: ChannelType
  t: (key: string) => string
  disabled?: boolean
  onClick?: () => void
}) {
  const iconConfig = channelIcons[type]

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
              iconConfig.bgColor,
              iconConfig.color
            )}
          >
            <MessageSquare className="h-5 w-5" />
          </div>
          <div className="flex-1">
            <h4 className="text-sm font-medium">{t(`types.${type}`)}</h4>
            <p className="text-xs text-muted-foreground">{t(`descriptions.${type}`)}</p>
          </div>
          {disabled ? (
            <Badge variant="secondary" className="text-[10px]">
              {t('comingSoon')}
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
  t,
  onSuccess,
}: {
  open: boolean
  onOpenChange: (open: boolean) => void
  channelType: ChannelType | null
  channel?: Channel
  t: (key: string, values?: Record<string, string>) => string
  onSuccess: () => void
}) {
  if (!channelType) return null

  const iconConfig = channelIcons[channelType]
  const isEditing = !!channel
  const channelLabel = t(`types.${channelType}`)

  const handleSuccess = () => {
    onSuccess()
    onOpenChange(false)
  }

  const handleCancel = () => {
    onOpenChange(false)
  }

  const renderConfigComponent = () => {
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
        return <VoiceConfig channel={channel} onSuccess={handleSuccess} onCancel={handleCancel} />
      default:
        return <p>{t('configNotAvailable')}</p>
    }
  }

  return (
    <Sheet open={open} onOpenChange={onOpenChange}>
      <SheetContent className="sm:max-w-xl flex flex-col h-full p-0">
        <SheetHeader className="px-6 pt-6 pb-4 border-b">
          <SheetTitle className="flex items-center gap-2">
            <div className={cn('p-2 rounded-lg', iconConfig.bgColor, iconConfig.color)}>
              <MessageSquare className="h-5 w-5" />
            </div>
            {isEditing ? t('configureChannel', { channel: channelLabel }) : t('addChannelType', { channel: channelLabel })}
          </SheetTitle>
          <SheetDescription>
            {isEditing
              ? t('updateSettings', { channel: channelLabel })
              : t('setupNewChannel', { channel: channelLabel })}
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
  const t = useTranslations('channels')
  const tCommon = useTranslations('common')
  const queryClient = useQueryClient()

  // State for dialogs
  const [configSheetOpen, setConfigSheetOpen] = useState(false)
  const [selectedChannelType, setSelectedChannelType] = useState<ChannelType | null>(null)
  const [selectedChannel, setSelectedChannel] = useState<Channel | undefined>()
  const [deleteDialogOpen, setDeleteDialogOpen] = useState(false)
  const [channelToDelete, setChannelToDelete] = useState<Channel | null>(null)

  // Fetch channels
  const { data, isLoading, refetch, isFetching } = useQuery({
    queryKey: queryKeys.channels.list(),
    queryFn: () => api.get<Channel[]>('/channels'),
  })

  const channels = Array.isArray(data) ? data : []

  // Delete mutation
  const deleteMutation = useMutation({
    mutationFn: (id: string) => api.delete(`/channels/${id}`),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: queryKeys.channels.all })
      toastSuccess(t('channelDeleted'), t('channelDeletedDesc'))
      setDeleteDialogOpen(false)
      setChannelToDelete(null)
    },
    onError: (error: Error) => {
      toastError(tCommon('error'), error.message)
    },
  })

  // Toggle status mutation
  const toggleStatusMutation = useMutation({
    mutationFn: ({ id, status }: { id: string; status: 'active' | 'inactive' }) =>
      api.put(`/channels/${id}`, { status }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: queryKeys.channels.all })
      toastSuccess(t('statusUpdated'), t('statusUpdatedDesc'))
    },
    onError: (error: Error) => {
      toastError(tCommon('error'), error.message)
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
      <Header title={t('title')} />

      <div className="p-6 space-y-6 overflow-auto">
        {/* Active Channels */}
        <section>
          <div className="flex items-center justify-between mb-4">
            <div>
              <h2 className="text-lg font-semibold">{t('activeChannels')}</h2>
              <p className="text-sm text-muted-foreground">
                {t('manageChannels')}
              </p>
            </div>
            <Button
              variant="outline"
              size="sm"
              onClick={() => refetch()}
              disabled={isFetching}
            >
              <RefreshCw className={cn("h-4 w-4 mr-2", isFetching && "animate-spin")} />
              {tCommon('refresh')}
            </Button>
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
                  t={t}
                  tCommon={tCommon}
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
                <p className="mt-4 text-lg font-medium">{t('noChannelsConfigured')}</p>
                <p className="text-sm text-muted-foreground">
                  {t('addFirstChannel')}
                </p>
              </CardContent>
            </Card>
          )}
        </section>

        {/* Available Channels */}
        <section>
          <div className="mb-4">
            <h2 className="text-lg font-semibold">{t('addNewChannel')}</h2>
            <p className="text-sm text-muted-foreground">
              {t('connectNewChannels')}
            </p>
          </div>

          <div className="grid gap-3 md:grid-cols-2 lg:grid-cols-4">
            <AvailableChannelCard type="webchat" t={t} onClick={() => handleAddChannel('webchat')} />
            <AvailableChannelCard type="whatsapp_official" t={t} onClick={() => handleAddChannel('whatsapp_official')} />
            <AvailableChannelCard type="telegram" t={t} onClick={() => handleAddChannel('telegram')} />
            <AvailableChannelCard type="sms" t={t} onClick={() => handleAddChannel('sms')} />
            <AvailableChannelCard type="facebook" t={t} onClick={() => handleAddChannel('facebook')} />
            <AvailableChannelCard type="instagram" t={t} onClick={() => handleAddChannel('instagram')} />
            <AvailableChannelCard type="email" t={t} onClick={() => handleAddChannel('email')} />
            <AvailableChannelCard type="whatsapp" t={t} onClick={() => handleAddChannel('whatsapp')} />
            <AvailableChannelCard type="rcs" t={t} onClick={() => handleAddChannel('rcs')} />
            <AvailableChannelCard type="voice" t={t} onClick={() => handleAddChannel('voice')} />
          </div>
        </section>
      </div>

      {/* Channel Config Sheet */}
      <ChannelConfigSheet
        open={configSheetOpen}
        onOpenChange={setConfigSheetOpen}
        channelType={selectedChannelType}
        channel={selectedChannel}
        t={t}
        onSuccess={handleConfigSuccess}
      />

      {/* Delete Confirmation Dialog */}
      <AlertDialog open={deleteDialogOpen} onOpenChange={setDeleteDialogOpen}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>{t('deleteChannelTitle')}</AlertDialogTitle>
            <AlertDialogDescription>
              {t('deleteChannelDescription', { name: channelToDelete?.name || '' })}
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel>{tCommon('cancel')}</AlertDialogCancel>
            <AlertDialogAction
              onClick={confirmDelete}
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
