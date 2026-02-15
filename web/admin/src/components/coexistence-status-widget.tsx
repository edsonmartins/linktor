'use client'

import { useQuery } from '@tanstack/react-query'
import { useTranslations } from 'next-intl'
import {
  CheckCircle2,
  AlertTriangle,
  XCircle,
  Smartphone,
  RefreshCw,
  Clock,
} from 'lucide-react'
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { Badge } from '@/components/ui/badge'
import { Progress } from '@/components/ui/progress'
import { Skeleton } from '@/components/ui/skeleton'
import { SimpleTooltip } from '@/components/ui/tooltip'
import { api } from '@/lib/api'
import { cn, formatRelativeTime } from '@/lib/utils'

interface CoexistenceStatus {
  enabled: boolean
  status: 'active' | 'warning' | 'disconnected' | 'pending' | 'inactive'
  days_since_last_echo: number
  days_until_disconnect: number
  last_echo_at: string | null
  recommendation?: string
}

interface CoexistenceStatusWidgetProps {
  channelId: string
  className?: string
}

/**
 * CoexistenceStatusWidget displays the WhatsApp Coexistence status for a channel
 * Shows warnings when the Business App hasn't been opened recently
 */
export function CoexistenceStatusWidget({
  channelId,
  className,
}: CoexistenceStatusWidgetProps) {
  const t = useTranslations('channels')

  const { data: status, isLoading, refetch, isRefetching } = useQuery({
    queryKey: ['coexistence-status', channelId],
    queryFn: () =>
      api.get<CoexistenceStatus>(`/channels/${channelId}/coexistence-status`),
    refetchInterval: 60000, // Refresh every minute
    staleTime: 30000,
  })

  if (isLoading) {
    return (
      <Card className={className}>
        <CardHeader className="pb-2">
          <Skeleton className="h-5 w-32" />
        </CardHeader>
        <CardContent>
          <Skeleton className="h-16 w-full" />
        </CardContent>
      </Card>
    )
  }

  // Don't render if coexistence is not enabled
  if (!status?.enabled) {
    return null
  }

  const getStatusConfig = () => {
    switch (status.status) {
      case 'active':
        return {
          icon: CheckCircle2,
          color: 'text-green-500',
          bgColor: 'bg-green-500/10',
          borderColor: 'border-green-500/20',
          label: 'Active',
          description: status.last_echo_at
            ? `Last activity ${formatRelativeTime(status.last_echo_at)}`
            : 'Business App active',
        }
      case 'warning':
        return {
          icon: AlertTriangle,
          color: 'text-yellow-500',
          bgColor: 'bg-yellow-500/10',
          borderColor: 'border-yellow-500/20',
          label: 'Warning',
          description: `${status.days_until_disconnect} days until disconnect`,
        }
      case 'disconnected':
        return {
          icon: XCircle,
          color: 'text-red-500',
          bgColor: 'bg-red-500/10',
          borderColor: 'border-red-500/20',
          label: 'Disconnected',
          description: 'Coexistence has been disabled',
        }
      case 'pending':
        return {
          icon: Clock,
          color: 'text-blue-500',
          bgColor: 'bg-blue-500/10',
          borderColor: 'border-blue-500/20',
          label: 'Pending',
          description: 'Waiting for first Business App activity',
        }
      default:
        return {
          icon: Clock,
          color: 'text-muted-foreground',
          bgColor: 'bg-muted/50',
          borderColor: 'border-muted',
          label: 'Unknown',
          description: 'Status unknown',
        }
    }
  }

  const config = getStatusConfig()
  const StatusIcon = config.icon

  // Calculate progress for warning state
  const progressValue =
    status.status === 'warning' && status.days_until_disconnect > 0
      ? ((14 - status.days_until_disconnect) / 14) * 100
      : 0

  return (
    <Card className={cn('overflow-hidden', className)}>
      <CardHeader className="pb-2">
        <div className="flex items-center justify-between">
          <CardTitle className="text-sm font-medium flex items-center gap-2">
            <Smartphone className="h-4 w-4" />
            Business App Activity
          </CardTitle>
          <SimpleTooltip content="Refresh status">
            <Button
              variant="ghost"
              size="icon"
              className="h-6 w-6"
              onClick={() => refetch()}
              disabled={isRefetching}
            >
              <RefreshCw className={cn('h-3 w-3', isRefetching && 'animate-spin')} />
            </Button>
          </SimpleTooltip>
        </div>
        <CardDescription className="text-xs">
          WhatsApp Coexistence Status
        </CardDescription>
      </CardHeader>

      <CardContent className="space-y-3">
        {/* Status Display */}
        <div
          className={cn(
            'flex items-center gap-3 p-3 rounded-lg border',
            config.bgColor,
            config.borderColor
          )}
        >
          <StatusIcon className={cn('h-5 w-5 shrink-0', config.color)} />
          <div className="flex-1 min-w-0">
            <div className="flex items-center gap-2">
              <span className={cn('font-medium text-sm', config.color)}>
                {config.label}
              </span>
              {status.days_since_last_echo >= 0 && (
                <Badge variant="outline" className="text-[10px] h-4">
                  {status.days_since_last_echo}d ago
                </Badge>
              )}
            </div>
            <p className="text-xs text-muted-foreground truncate">
              {config.description}
            </p>
          </div>
        </div>

        {/* Progress Bar for Warning State */}
        {status.status === 'warning' && (
          <div className="space-y-1">
            <div className="flex justify-between text-[10px] text-muted-foreground">
              <span>Days until disconnect</span>
              <span>{status.days_until_disconnect} days remaining</span>
            </div>
            <Progress value={progressValue} className="h-1.5" />
          </div>
        )}

        {/* Recommendation */}
        {status.recommendation && (
          <p className="text-xs text-muted-foreground bg-muted/30 p-2 rounded">
            {status.recommendation}
          </p>
        )}

        {/* Action for Warning/Disconnected */}
        {(status.status === 'warning' || status.status === 'disconnected') && (
          <div className="pt-1">
            <p className="text-xs text-muted-foreground mb-2">
              To maintain coexistence, open WhatsApp Business App on your phone.
            </p>
          </div>
        )}
      </CardContent>
    </Card>
  )
}

/**
 * Compact version of the coexistence status for inline display
 */
export function CoexistenceStatusBadge({
  channelId,
  className,
}: CoexistenceStatusWidgetProps) {
  const { data: status, isLoading } = useQuery({
    queryKey: ['coexistence-status', channelId],
    queryFn: () =>
      api.get<CoexistenceStatus>(`/channels/${channelId}/coexistence-status`),
    staleTime: 60000,
  })

  if (isLoading || !status?.enabled) {
    return null
  }

  const getStatusStyle = () => {
    switch (status.status) {
      case 'active':
        return 'bg-green-500/10 text-green-500 border-green-500/20'
      case 'warning':
        return 'bg-yellow-500/10 text-yellow-500 border-yellow-500/20'
      case 'disconnected':
        return 'bg-red-500/10 text-red-500 border-red-500/20'
      default:
        return 'bg-muted text-muted-foreground'
    }
  }

  const getIcon = () => {
    switch (status.status) {
      case 'active':
        return <CheckCircle2 className="h-3 w-3" />
      case 'warning':
        return <AlertTriangle className="h-3 w-3" />
      case 'disconnected':
        return <XCircle className="h-3 w-3" />
      default:
        return <Smartphone className="h-3 w-3" />
    }
  }

  return (
    <SimpleTooltip
      content={
        status.status === 'warning'
          ? `${status.days_until_disconnect} days until coexistence disconnect`
          : status.status === 'disconnected'
          ? 'Coexistence disconnected - open Business App'
          : 'Coexistence active'
      }
    >
      <Badge
        variant="outline"
        className={cn('gap-1 text-[10px] cursor-help', getStatusStyle(), className)}
      >
        {getIcon()}
        <span className="capitalize">{status.status}</span>
      </Badge>
    </SimpleTooltip>
  )
}

export default CoexistenceStatusWidget
