'use client'

import { useQuery } from '@tanstack/react-query'
import {
  Plus,
  Radio,
  MessageSquare,
  Settings,
  MoreVertical,
  Wifi,
  WifiOff,
  AlertTriangle,
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
import { cn } from '@/lib/utils'
import { api } from '@/lib/api'
import { queryKeys } from '@/lib/query'
import type { Channel, ChannelType } from '@/types'

/**
 * Channel type configurations - Plugin Pattern
 * Each channel type is a "plugin" with its own config
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
    description: 'Meta Cloud API integration',
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
function ChannelCard({ channel }: { channel: Channel }) {
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
              <DropdownMenuItem>
                <Settings className="h-4 w-4 mr-2" />
                Configure
              </DropdownMenuItem>
              <DropdownMenuItem>
                View analytics
              </DropdownMenuItem>
              <DropdownMenuSeparator />
              <DropdownMenuItem className="text-destructive">
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
}: {
  type: ChannelType
  disabled?: boolean
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
 * Channels Page
 */
export default function ChannelsPage() {
  // Fetch channels
  const { data, isLoading } = useQuery({
    queryKey: queryKeys.channels.list(),
    queryFn: () => api.get<{ data: Channel[] }>('/channels'),
  })

  const channels = data?.data || []

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
                <ChannelCard key={channel.id} channel={channel} />
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
            <AvailableChannelCard type="webchat" />
            <AvailableChannelCard type="whatsapp_official" />
            <AvailableChannelCard type="whatsapp" disabled />
            <AvailableChannelCard type="telegram" disabled />
            <AvailableChannelCard type="sms" disabled />
            <AvailableChannelCard type="instagram" disabled />
            <AvailableChannelCard type="facebook" disabled />
            <AvailableChannelCard type="rcs" disabled />
          </div>
        </section>
      </div>
    </div>
  )
}
