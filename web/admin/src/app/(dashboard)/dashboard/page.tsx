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
import type { Conversation, DashboardStats } from '@/types'

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
  // Fetch dashboard stats
  const { data: stats, isLoading: statsLoading } = useQuery({
    queryKey: queryKeys.dashboard.stats(),
    queryFn: () => api.get<DashboardStats>('/dashboard/stats'),
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

  // Mock stats for demo
  const mockStats: DashboardStats = {
    total_conversations: 156,
    open_conversations: 23,
    total_messages_today: 487,
    total_contacts: 1247,
    avg_response_time_seconds: 180,
    satisfaction_rate: 94.5,
  }

  const displayStats = stats || mockStats

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
                  title="Open Conversations"
                  value={displayStats.open_conversations}
                  description="active now"
                  icon={<MessageSquare className="h-5 w-5" />}
                  trend={{ value: 12, isPositive: true }}
                />
                <StatCard
                  title="Messages Today"
                  value={displayStats.total_messages_today}
                  description="vs yesterday"
                  icon={<Activity className="h-5 w-5" />}
                  trend={{ value: 8, isPositive: true }}
                />
                <StatCard
                  title="Total Contacts"
                  value={displayStats.total_contacts.toLocaleString()}
                  description="this month"
                  icon={<Users className="h-5 w-5" />}
                  trend={{ value: 5, isPositive: true }}
                />
                <StatCard
                  title="Avg Response Time"
                  value={`${Math.floor(displayStats.avg_response_time_seconds / 60)}min`}
                  description="target: 5min"
                  icon={<Clock className="h-5 w-5" />}
                  trend={{ value: 15, isPositive: false }}
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
                  Active communication channels
                </CardDescription>
              </CardHeader>
              <CardContent>
                <div className="space-y-4">
                  {/* WebChat */}
                  <div className="flex items-center justify-between">
                    <div className="flex items-center gap-3">
                      <div className="flex h-10 w-10 items-center justify-center rounded-lg bg-primary/10">
                        <MessageSquare className="h-5 w-5 text-primary" />
                      </div>
                      <div>
                        <p className="text-sm font-medium">Web Chat</p>
                        <p className="text-xs text-muted-foreground">
                          Widget Integration
                        </p>
                      </div>
                    </div>
                    <Badge variant="success" dot pulse>
                      Online
                    </Badge>
                  </div>

                  {/* WhatsApp */}
                  <div className="flex items-center justify-between">
                    <div className="flex items-center gap-3">
                      <div className="flex h-10 w-10 items-center justify-center rounded-lg bg-green-500/10">
                        <MessageSquare className="h-5 w-5 text-green-500" />
                      </div>
                      <div>
                        <p className="text-sm font-medium">WhatsApp</p>
                        <p className="text-xs text-muted-foreground">
                          Meta Cloud API
                        </p>
                      </div>
                    </div>
                    <Badge variant="warning" dot>
                      Pending
                    </Badge>
                  </div>

                  {/* Telegram */}
                  <div className="flex items-center justify-between">
                    <div className="flex items-center gap-3">
                      <div className="flex h-10 w-10 items-center justify-center rounded-lg bg-blue-500/10">
                        <MessageSquare className="h-5 w-5 text-blue-500" />
                      </div>
                      <div>
                        <p className="text-sm font-medium">Telegram</p>
                        <p className="text-xs text-muted-foreground">
                          Bot API
                        </p>
                      </div>
                    </div>
                    <Badge variant="secondary">
                      Not configured
                    </Badge>
                  </div>
                </div>
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
