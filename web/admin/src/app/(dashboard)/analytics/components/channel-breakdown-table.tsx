'use client'

import {
  MessageSquare,
  MessageCircle,
  Globe,
  Phone,
  Instagram,
  Facebook,
} from 'lucide-react'
import { cn } from '@/lib/utils'
import type { ChannelAnalytics } from '@/types'

interface ChannelBreakdownTableProps {
  data: ChannelAnalytics[]
}

const CHANNEL_ICONS: Record<string, React.ReactNode> = {
  whatsapp: <MessageCircle className="h-4 w-4 text-green-600" />,
  whatsapp_official: <MessageCircle className="h-4 w-4 text-green-600" />,
  telegram: <MessageSquare className="h-4 w-4 text-blue-500" />,
  webchat: <Globe className="h-4 w-4 text-purple-600" />,
  sms: <Phone className="h-4 w-4 text-gray-600" />,
  instagram: <Instagram className="h-4 w-4 text-pink-600" />,
  facebook: <Facebook className="h-4 w-4 text-blue-600" />,
}

export function ChannelBreakdownTable({ data }: ChannelBreakdownTableProps) {
  return (
    <div className="rounded-lg border bg-card p-4 shadow-sm">
      <h3 className="mb-4 text-lg font-semibold flex items-center gap-2">
        <Globe className="h-5 w-5 text-primary" />
        Channel Breakdown
      </h3>
      <div className="overflow-x-auto">
        {data.length === 0 ? (
          <div className="flex h-[200px] items-center justify-center text-muted-foreground">
            No channel data available
          </div>
        ) : (
          <table className="w-full">
            <thead>
              <tr className="border-b text-left text-sm text-muted-foreground">
                <th className="pb-3 font-medium">Channel</th>
                <th className="pb-3 font-medium text-right">Convos</th>
                <th className="pb-3 font-medium text-right">Bot Resolved</th>
                <th className="pb-3 font-medium text-right">Rate</th>
              </tr>
            </thead>
            <tbody>
              {data.map((channel) => (
                <tr key={channel.channel_id} className="border-b last:border-0">
                  <td className="py-3">
                    <div className="flex items-center gap-2">
                      {CHANNEL_ICONS[channel.channel_type] || (
                        <MessageSquare className="h-4 w-4 text-gray-500" />
                      )}
                      <span className="font-medium">{channel.channel_name}</span>
                    </div>
                  </td>
                  <td className="py-3 text-right">{channel.total_conversations}</td>
                  <td className="py-3 text-right">{channel.resolved_by_bot}</td>
                  <td className="py-3 text-right">
                    <span
                      className={cn(
                        'inline-flex items-center rounded-full px-2 py-0.5 text-xs font-medium',
                        channel.resolution_rate >= 80
                          ? 'bg-emerald-100 text-emerald-700'
                          : channel.resolution_rate >= 50
                          ? 'bg-amber-100 text-amber-700'
                          : 'bg-red-100 text-red-700'
                      )}
                    >
                      {channel.resolution_rate.toFixed(0)}%
                    </span>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        )}
      </div>
    </div>
  )
}
