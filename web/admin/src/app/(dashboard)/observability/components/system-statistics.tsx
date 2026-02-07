'use client'

import { useState } from 'react'
import { useQuery } from '@tanstack/react-query'
import {
  BarChart,
  Bar,
  XAxis,
  YAxis,
  CartesianGrid,
  Tooltip,
  ResponsiveContainer,
  PieChart,
  Pie,
  Cell,
  LineChart,
  Line,
} from 'recharts'
import { format } from 'date-fns'
import {
  MessageSquare,
  Clock,
  Radio,
  Users,
  AlertTriangle,
  RefreshCw,
  TrendingUp,
} from 'lucide-react'
import { cn } from '@/lib/utils'
import { queryKeys } from '@/lib/query'
import { api } from '@/lib/api'
import type { SystemStats, StatsPeriod } from '@/types'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { Badge } from '@/components/ui/badge'
import { Spinner } from '@/components/ui/spinner'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'

const COLORS = ['#10B981', '#F59E0B', '#EF4444', '#8B5CF6', '#3B82F6']

interface StatCardProps {
  title: string
  value: string | number
  icon: React.ReactNode
  description?: string
  trend?: 'up' | 'down' | 'neutral'
  trendValue?: string
}

function StatCard({ title, value, icon, description, trend, trendValue }: StatCardProps) {
  return (
    <Card>
      <CardContent className="p-4">
        <div className="flex items-start justify-between">
          <div>
            <p className="text-sm text-muted-foreground">{title}</p>
            <p className="text-2xl font-bold mt-1">{value}</p>
            {description && (
              <p className="text-xs text-muted-foreground mt-1">{description}</p>
            )}
          </div>
          <div className="p-2 rounded-lg bg-primary/10 text-primary">
            {icon}
          </div>
        </div>
        {trend && trendValue && (
          <div className={cn(
            'flex items-center gap-1 mt-2 text-xs',
            trend === 'up' && 'text-green-500',
            trend === 'down' && 'text-red-500',
            trend === 'neutral' && 'text-muted-foreground'
          )}>
            <TrendingUp className={cn('h-3 w-3', trend === 'down' && 'rotate-180')} />
            <span>{trendValue}</span>
          </div>
        )}
      </CardContent>
    </Card>
  )
}

export function SystemStatistics() {
  const [period, setPeriod] = useState<StatsPeriod>('day')

  const { data, isLoading, refetch, isFetching } = useQuery({
    queryKey: queryKeys.observability.stats(period),
    queryFn: () => api.get<SystemStats>('/observability/stats', { period }),
    refetchInterval: 30000, // Auto-refresh every 30 seconds
  })

  // Format data for charts
  const hourlyData = data?.messages?.per_hour?.map((item) => ({
    hour: format(new Date(item.hour), 'HH:mm'),
    count: item.count,
  })) || []

  const channelData = data?.messages?.by_channel?.map((item, index) => ({
    name: item.channel_name,
    value: item.count,
    color: COLORS[index % COLORS.length],
  })) || []

  const errorsBySource = data?.errors?.by_source?.map((item, index) => ({
    name: item.source,
    value: item.count,
    color: COLORS[index % COLORS.length],
  })) || []

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-4">
          <Select
            value={period}
            onValueChange={(v) => setPeriod(v as StatsPeriod)}
          >
            <SelectTrigger className="w-[150px]">
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="hour">Last Hour</SelectItem>
              <SelectItem value="day">Last 24 Hours</SelectItem>
              <SelectItem value="week">Last 7 Days</SelectItem>
            </SelectContent>
          </Select>
        </div>

        <Button
          variant="outline"
          size="sm"
          onClick={() => refetch()}
          disabled={isFetching}
        >
          <RefreshCw className={cn('h-4 w-4 mr-2', isFetching && 'animate-spin')} />
          Refresh
        </Button>
      </div>

      {isLoading ? (
        <div className="flex items-center justify-center py-20">
          <Spinner size="lg" />
        </div>
      ) : (
        <>
          {/* Stats Cards */}
          <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-4">
            <StatCard
              title="Total Messages"
              value={data?.messages?.total?.toLocaleString() || 0}
              icon={<MessageSquare className="h-5 w-5" />}
              description={`In the last ${period === 'hour' ? 'hour' : period === 'day' ? '24 hours' : '7 days'}`}
            />
            <StatCard
              title="Avg Response Time"
              value={`${data?.response_time?.avg_ms || 0}ms`}
              icon={<Clock className="h-5 w-5" />}
              description={`P95: ${data?.response_time?.p95_ms || 0}ms`}
            />
            <StatCard
              title="Connected Channels"
              value={`${data?.channels?.connected || 0}/${data?.channels?.total || 0}`}
              icon={<Radio className="h-5 w-5" />}
              description={`${data?.channels?.disconnected || 0} disconnected`}
            />
            <StatCard
              title="Errors (24h)"
              value={data?.errors?.last_24h || 0}
              icon={<AlertTriangle className="h-5 w-5" />}
              description={`${data?.errors?.last_hour || 0} in the last hour`}
            />
          </div>

          {/* Conversation Stats */}
          <div className="grid gap-4 md:grid-cols-3">
            <Card className="p-4">
              <div className="flex items-center gap-3">
                <div className="p-2 rounded-full bg-green-100 text-green-600">
                  <Users className="h-5 w-5" />
                </div>
                <div>
                  <p className="text-sm text-muted-foreground">Active Conversations</p>
                  <p className="text-xl font-bold">{data?.conversations?.active || 0}</p>
                </div>
              </div>
            </Card>
            <Card className="p-4">
              <div className="flex items-center gap-3">
                <div className="p-2 rounded-full bg-blue-100 text-blue-600">
                  <Users className="h-5 w-5" />
                </div>
                <div>
                  <p className="text-sm text-muted-foreground">Resolved Today</p>
                  <p className="text-xl font-bold">{data?.conversations?.resolved_today || 0}</p>
                </div>
              </div>
            </Card>
            <Card className="p-4">
              <div className="flex items-center gap-3">
                <div className="p-2 rounded-full bg-yellow-100 text-yellow-600">
                  <Users className="h-5 w-5" />
                </div>
                <div>
                  <p className="text-sm text-muted-foreground">Pending</p>
                  <p className="text-xl font-bold">{data?.conversations?.pending || 0}</p>
                </div>
              </div>
            </Card>
          </div>

          {/* Charts */}
          <div className="grid gap-6 md:grid-cols-2">
            {/* Messages per Hour */}
            <Card>
              <CardHeader>
                <CardTitle className="text-base">Messages per Hour</CardTitle>
              </CardHeader>
              <CardContent>
                {hourlyData.length > 0 ? (
                  <ResponsiveContainer width="100%" height={250}>
                    <LineChart data={hourlyData}>
                      <CartesianGrid strokeDasharray="3 3" className="stroke-muted" />
                      <XAxis dataKey="hour" className="text-xs" />
                      <YAxis className="text-xs" />
                      <Tooltip
                        contentStyle={{
                          backgroundColor: 'hsl(var(--card))',
                          border: '1px solid hsl(var(--border))',
                          borderRadius: '8px',
                        }}
                      />
                      <Line
                        type="monotone"
                        dataKey="count"
                        stroke="hsl(var(--primary))"
                        strokeWidth={2}
                        dot={false}
                      />
                    </LineChart>
                  </ResponsiveContainer>
                ) : (
                  <div className="flex items-center justify-center h-[250px] text-muted-foreground">
                    No data available
                  </div>
                )}
              </CardContent>
            </Card>

            {/* Messages by Channel */}
            <Card>
              <CardHeader>
                <CardTitle className="text-base">Messages by Channel</CardTitle>
              </CardHeader>
              <CardContent>
                {channelData.length > 0 ? (
                  <ResponsiveContainer width="100%" height={250}>
                    <PieChart>
                      <Pie
                        data={channelData}
                        cx="50%"
                        cy="50%"
                        outerRadius={80}
                        dataKey="value"
                        label={({ name, percent }) =>
                          `${name} (${(percent * 100).toFixed(0)}%)`
                        }
                      >
                        {channelData.map((entry, index) => (
                          <Cell key={`cell-${index}`} fill={entry.color} />
                        ))}
                      </Pie>
                      <Tooltip
                        contentStyle={{
                          backgroundColor: 'hsl(var(--card))',
                          border: '1px solid hsl(var(--border))',
                          borderRadius: '8px',
                        }}
                      />
                    </PieChart>
                  </ResponsiveContainer>
                ) : (
                  <div className="flex items-center justify-center h-[250px] text-muted-foreground">
                    No data available
                  </div>
                )}
              </CardContent>
            </Card>
          </div>

          {/* Errors by Source */}
          {errorsBySource.length > 0 && (
            <Card>
              <CardHeader>
                <CardTitle className="text-base flex items-center gap-2">
                  <AlertTriangle className="h-4 w-4 text-destructive" />
                  Errors by Source (Last 24h)
                </CardTitle>
              </CardHeader>
              <CardContent>
                <ResponsiveContainer width="100%" height={200}>
                  <BarChart data={errorsBySource} layout="vertical">
                    <CartesianGrid strokeDasharray="3 3" className="stroke-muted" />
                    <XAxis type="number" className="text-xs" />
                    <YAxis type="category" dataKey="name" className="text-xs" width={80} />
                    <Tooltip
                      contentStyle={{
                        backgroundColor: 'hsl(var(--card))',
                        border: '1px solid hsl(var(--border))',
                        borderRadius: '8px',
                      }}
                    />
                    <Bar dataKey="value" fill="hsl(var(--destructive))" radius={4} />
                  </BarChart>
                </ResponsiveContainer>
              </CardContent>
            </Card>
          )}
        </>
      )}
    </div>
  )
}
