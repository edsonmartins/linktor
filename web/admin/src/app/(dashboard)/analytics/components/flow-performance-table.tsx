'use client'

import { GitBranch } from 'lucide-react'
import { cn } from '@/lib/utils'
import type { FlowAnalytics } from '@/types'

interface FlowPerformanceTableProps {
  data: FlowAnalytics[]
}

export function FlowPerformanceTable({ data }: FlowPerformanceTableProps) {
  return (
    <div className="rounded-lg border bg-card p-4 shadow-sm">
      <h3 className="mb-4 text-lg font-semibold flex items-center gap-2">
        <GitBranch className="h-5 w-5 text-primary" />
        Flow Performance
      </h3>
      <div className="overflow-x-auto">
        {data.length === 0 ? (
          <div className="flex h-[200px] items-center justify-center text-muted-foreground">
            No flow data available
          </div>
        ) : (
          <table className="w-full">
            <thead>
              <tr className="border-b text-left text-sm text-muted-foreground">
                <th className="pb-3 font-medium">Flow Name</th>
                <th className="pb-3 font-medium text-right">Triggered</th>
                <th className="pb-3 font-medium text-right">Completed</th>
                <th className="pb-3 font-medium text-right">Rate</th>
              </tr>
            </thead>
            <tbody>
              {data.map((flow) => (
                <tr key={flow.flow_id} className="border-b last:border-0">
                  <td className="py-3">
                    <span className="font-medium">{flow.flow_name}</span>
                  </td>
                  <td className="py-3 text-right">{flow.times_triggered}</td>
                  <td className="py-3 text-right">{flow.times_completed}</td>
                  <td className="py-3 text-right">
                    <span
                      className={cn(
                        'inline-flex items-center rounded-full px-2 py-0.5 text-xs font-medium',
                        flow.completion_rate >= 80
                          ? 'bg-emerald-100 text-emerald-700'
                          : flow.completion_rate >= 50
                          ? 'bg-amber-100 text-amber-700'
                          : 'bg-red-100 text-red-700'
                      )}
                    >
                      {flow.completion_rate.toFixed(0)}%
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
