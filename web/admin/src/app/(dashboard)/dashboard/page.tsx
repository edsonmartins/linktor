'use client'

import { useQuery } from '@tanstack/react-query'
import {
  MessageSquare,
  Users,
  Radio,
  Clock,
  TrendingUp,
  TrendingDown,
  Activity,
  Bot,
  Wifi,
  WifiOff,
  AlertTriangle,
} from 'lucide-react'
import { Header } from '@/components/layout/header'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import { Skeleton } from '@/components/ui/skeleton'
import { Avatar } from '@/components/ui/avatar'
import { ScrollArea } from '@/components/ui/scroll-area'
import { cn, formatRelativeTime } from '@/lib/utils'
import { api } from '@/lib/api'
import { queryKeys } from '@/lib/query'
import type { Conversation, DashboardStats, Channel, Bot as BotType, OverviewAnalytics } from '@/types'

/**
 * Stats Card Component
 */
interface StatCardProps {
  title: string
  value: string | number
  description?: string
  icon: React.ReactNode
  trend?: {
    value: number
    isPositive: boolean
  }
}

function StatCard({ title, value, description, icon, trend }: StatCardProps) {
  return (
    <Card>
      <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
        <CardTitle className="text-sm font-medium text-muted-foreground">
          {title}
        </CardTitle>
        <div className="text-primary">{icon}</div>
      </CardHeader>
      <CardContent>
        <div className="text-2xl font-bold">{value}</div>
        {(description || trend) && (
          <div className="flex items-center gap-2 text-xs text-muted-foreground">
            {trend && (
              <span
                className={cn(
                  'flex items-center',
                  trend.isPositive ? 'text-terminal-green' : 'text-terminal-coral'
                )}
              >
                {trend.isPositive ? (
                  <TrendingUp className="mr-1 h-3 w-3" />
                ) : (
                  <TrendingDown className="mr-1 h-3 w-3" />
                )}
                {Math.abs(trend.value)}%
              </span>
            )}
            {description && <span>{description}</span>}
          </div>
        )}
      </CardContent>
    </Card>
  )
}

/**
 * Recent Conversation Item
 */
/**
 * Format duration in milliseconds to human readable string
 */
function formatDuration(ms: number): string {
  if (ms < 1000) return `${ms}ms`
  if (ms < 60000) return `${(ms / 1000).toFixed(1)}s`
  if (ms < 3600000) return `${Math.floor(ms / 60000)}min`
  return `${(ms / 3600000).toFixed(1)}h`
}

/**
 * Channel Status Badge
 */
function ChannelStatusBadge({ status }: { status: Channel['status'] }) {
  const config = {
    active: { variant: 'success' as const, icon: <Wifi className="h-3 w-3" />, label: 'Online' },
    inactive: { variant: 'secondary' as const, icon: <WifiOff className="h-3 w-3" />, label: 'Offline' },
    error: { variant: 'error' as const, icon: <AlertTriangle className="h-3 w-3" />, label: 'Error' },
  }
  const { variant, icon, label } = config[status] || config.inactive
  return (
    <Badge variant={variant} className="gap-1">
      {icon}
      {label}
    </Badge>
  )
}

/**
 * Channel Type Config
 */
const channelTypeConfig: Record<string, { color: string; bgColor: string }> = {
  webchat: { color: 'text-primary', bgColor: 'bg-primary/10' },
  whatsapp: { color: 'text-green-500', bgColor: 'bg-green-500/10' },
  whatsapp_official: { color: 'text-green-600', bgColor: 'bg-green-600/10' },
  telegram: { color: 'text-blue-500', bgColor: 'bg-blue-500/10' },
  sms: { color: 'text-purple-500', bgColor: 'bg-purple-500/10' },
  instagram: { color: 'text-pink-500', bgColor: 'bg-pink-500/10' },
  facebook: { color: 'text-blue-600', bgColor: 'bg-blue-600/10' },
  email: { color: 'text-amber-500', bgColor: 'bg-amber-500/10' },
  rcs: { color: 'text-orange-500', bgColor: 'bg-orange-500/10' },
  voice: { color: 'text-cyan-500', bgColor: 'bg-cyan-500/10' },
}

function ConversationItem({ conversation }: { conversation: Conversation }) {
  return (
    <div className="flex items-center gap-3 rounded-md p-3 hover:bg-secondary/50 transition-colors">
      <Avatar
        fallback={conversation.contact?.name || 'U'}
        size="sm"
        status="online"
      />
      <div className="flex-1 min-w-0">
        <div className="flex items-center justify-between gap-2">
          <p className="truncate text-sm font-medium">
            {conversation.contact?.name || 'Unknown'}
          </p>
          <Badge variant={conversation.channel?.type as 'webchat' | undefined || 'info'} className="shrink-0">
            {conversation.channel?.type}
          </Badge>
        </div>
        <p className="truncate text-xs text-muted-foreground">
          {conversation.last_message?.content || 'No messages'}
        </p>
      </div>
      <span className="text-xs text-muted-foreground shrink-0">
        {conversation.last_message_at
          ? formatRelativeTime(conversation.last_message_at)
          : '-'}
      </span>
    </div>
  )
}

/**
 * Dashboard Page
 */
export default function DashboardPage() {
  // Fetch analytics overview
  const { data: analytics, isLoading: analyticsLoading } = useQuery({
    queryKey: queryKeys.analytics.overview({ period: 'daily' }),
    queryFn: () => api.get<OverviewAnalytics>('/analytics/overview', { period: 'daily' }),
  })

  // Fetch recent conversations
  const { data: conversations, isLoading: conversationsLoading } = useQuery({
    queryKey: queryKeys.conversations.list({ limit: '5', status: 'open' }),
    queryFn: () =>
      api.get<{ data: Conversation[] }>('/conversations', {
        limit: '5',
        status: 'open',
      }),
  })

  // Fetch channels
  const { data: channelsData, isLoading: channelsLoading } = useQuery({
    queryKey: queryKeys.channels.list(),
    queryFn: () => api.get<{ data: Channel[] }>('/channels'),
  })

  // Fetch active bots
  const { data: botsData } = useQuery({
    queryKey: queryKeys.bots.list({ is_active: true }),
    queryFn: () => api.get<{ data: BotType[] }>('/bots', { is_active: 'true' }),
  })

  // Fetch contacts count
  const { data: contactsData } = useQuery({
    queryKey: queryKeys.contacts.list({ limit: '1' }),
    queryFn: () => api.get<{ data: unknown[], total: number }>('/contacts', { limit: '1' }),
  })

  const channels = channelsData?.data || []
  const activeBots = botsData?.data?.length || 0
  const totalContacts = contactsData?.total || 0

  const statsLoading = analyticsLoading

  return (
    <div className="flex flex-col h-full">
      <Header title="Dashboard" />

      <ScrollArea className="flex-1">
        <div className="p-6 space-y-6">
          {/* Stats Grid */}
          <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-4">
            {statsLoading ? (
              <>
                {[1, 2, 3, 4].map((i) => (
                  <Card key={i}>
                    <CardHeader className="pb-2">
                      <Skeleton className="h-4 w-24" />
                    </CardHeader>
                    <CardContent>
                      <Skeleton className="h-8 w-16" />
                      <Skeleton className="mt-2 h-3 w-32" />
                    </CardContent>
                  </Card>
                ))}
              </>
            ) : (
              <>
                <StatCard
                  title="Total Conversations"
                  value={analytics?.total_conversations || 0}
                  description="this period"
                  icon={<MessageSquare className="h-5 w-5" />}
                  trend={analytics?.conversations_trend ? {
                    value: Math.abs(analytics.conversations_trend),
                    isPositive: analytics.conversations_trend >= 0
                  } : undefined}
                />
                <StatCard
                  title="Resolution Rate"
                  value={`${((analytics?.resolution_rate || 0) * 100).toFixed(1)}%`}
                  description="by bot"
                  icon={<Bot className="h-5 w-5" />}
                  trend={analytics?.resolution_trend ? {
                    value: Math.abs(analytics.resolution_trend),
                    isPositive: analytics.resolution_trend >= 0
                  } : undefined}
                />
                <StatCard
                  title="Total Contacts"
                  value={totalContacts.toLocaleString()}
                  description="in database"
                  icon={<Users className="h-5 w-5" />}
                />
                <StatCard
                  title="Avg Response Time"
                  value={formatDuration(analytics?.avg_first_response_ms || 0)}
                  description="first response"
                  icon={<Clock className="h-5 w-5" />}
                />
              </>
            )}
          </div>

          {/* Main Content Grid */}
          <div className="grid gap-6 lg:grid-cols-2">
            {/* Recent Conversations */}
            <Card className="col-span-1">
              <CardHeader>
                <CardTitle className="flex items-center gap-2">
                  <MessageSquare className="h-5 w-5 text-primary" />
                  Recent Conversations
                </CardTitle>
                <CardDescription>
                  Latest open conversations requiring attention
                </CardDescription>
              </CardHeader>
              <CardContent>
                {conversationsLoading ? (
                  <div className="space-y-3">
                    {[1, 2, 3, 4, 5].map((i) => (
                      <div key={i} className="flex items-center gap-3 p-3">
                        <Skeleton className="h-8 w-8 rounded-full" />
                        <div className="flex-1 space-y-2">
                          <Skeleton className="h-4 w-32" />
                          <Skeleton className="h-3 w-48" />
                        </div>
                      </div>
                    ))}
                  </div>
                ) : conversations?.data?.length ? (
                  <div className="space-y-1">
                    {conversations.data.map((conv) => (
                      <ConversationItem key={conv.id} conversation={conv} />
                    ))}
                  </div>
                ) : (
                  <div className="py-8 text-center text-muted-foreground">
                    <MessageSquare className="mx-auto h-8 w-8 opacity-50" />
                    <p className="mt-2 text-sm">No open conversations</p>
                  </div>
                )}
              </CardContent>
            </Card>

            {/* Channel Status */}
            <Card className="col-span-1">
              <CardHeader>
                <CardTitle className="flex items-center gap-2">
                  <Radio className="h-5 w-5 text-primary" />
                  Channel Status
                </CardTitle>
                <CardDescription>
                  Active communication channels ({channels.length})
                </CardDescription>
              </CardHeader>
              <CardContent>
                {channelsLoading ? (
                  <div className="space-y-4">
                    {[1, 2, 3].map((i) => (
                      <div key={i} className="flex items-center justify-between">
                        <div className="flex items-center gap-3">
                          <Skeleton className="h-10 w-10 rounded-lg" />
                          <div className="space-y-2">
                            <Skeleton className="h-4 w-24" />
                            <Skeleton className="h-3 w-32" />
                          </div>
                        </div>
                        <Skeleton className="h-6 w-16" />
                      </div>
                    ))}
                  </div>
                ) : channels.length > 0 ? (
                  <div className="space-y-4">
                    {channels.map((channel) => {
                      const typeConfig = channelTypeConfig[channel.type] || channelTypeConfig.webchat
                      return (
                        <div key={channel.id} className="flex items-center justify-between">
                          <div className="flex items-center gap-3">
                            <div className={cn(
                              'flex h-10 w-10 items-center justify-center rounded-lg',
                              typeConfig.bgColor
                            )}>
                              <MessageSquare className={cn('h-5 w-5', typeConfig.color)} />
                            </div>
                            <div>
                              <p className="text-sm font-medium">{channel.name}</p>
                              <p className="text-xs text-muted-foreground capitalize">
                                {channel.type.replace('_', ' ')}
                              </p>
                            </div>
                          </div>
                          <ChannelStatusBadge status={channel.status} />
                        </div>
                      )
                    })}
                  </div>
                ) : (
                  <div className="py-6 text-center text-muted-foreground">
                    <Radio className="mx-auto h-8 w-8 opacity-50" />
                    <p className="mt-2 text-sm">No channels configured</p>
                  </div>
                )}
              </CardContent>
            </Card>
          </div>

          {/* Terminal-style activity log */}
          <Card variant="terminal">
            <CardHeader>
              <CardTitle className="font-mono text-sm text-primary">
                {'>'} system.log
              </CardTitle>
            </CardHeader>
            <CardContent>
              <div className="font-mono text-xs text-muted-foreground space-y-1">
                <p>
                  <span className="text-terminal-green">[INFO]</span>{' '}
                  <span className="text-muted-foreground/70">{new Date().toISOString()}</span>{' '}
                  Dashboard loaded successfully
                </p>
                <p>
                  <span className="text-terminal-cyan">[CONN]</span>{' '}
                  <span className="text-muted-foreground/70">{new Date().toISOString()}</span>{' '}
                  WebSocket connection established
                </p>
                <p>
                  <span className="text-terminal-yellow">[SYNC]</span>{' '}
                  <span className="text-muted-foreground/70">{new Date().toISOString()}</span>{' '}
                  Syncing conversations...
                </p>
              </div>
            </CardContent>
          </Card>
        </div>
      </ScrollArea>
    </div>
  )
}
