'use client'

import { memo } from 'react'
import { Handle, Position } from '@xyflow/react'
import { Zap, Star, Tag, UserPlus, AlertTriangle, Globe } from 'lucide-react'
import { cn } from '@/lib/utils'
import type { FlowAction, FlowActionType } from '@/types'

interface ActionNodeProps {
  data: {
    content: string
    actions?: FlowAction[]
    isStart?: boolean
    isSelected?: boolean
  }
}

const actionIcons: Record<FlowActionType, React.ReactNode> = {
  tag: <Tag className="h-3 w-3" />,
  assign: <UserPlus className="h-3 w-3" />,
  escalate: <AlertTriangle className="h-3 w-3" />,
  set_entity: <Zap className="h-3 w-3" />,
  http_call: <Globe className="h-3 w-3" />,
}

const actionLabels: Record<FlowActionType, string> = {
  tag: 'Add Tag',
  assign: 'Assign',
  escalate: 'Escalate',
  set_entity: 'Set Entity',
  http_call: 'HTTP Call',
}

export const ActionNode = memo(function ActionNode({ data }: ActionNodeProps) {
  const { content, actions, isStart, isSelected } = data

  return (
    <div
      className={cn(
        'rounded-lg border-2 bg-white p-4 shadow-sm transition-shadow min-w-[200px] max-w-[280px]',
        isSelected ? 'border-emerald-500 shadow-md' : 'border-emerald-200',
        isStart && 'ring-2 ring-green-500 ring-offset-2'
      )}
    >
      <Handle
        type="target"
        position={Position.Top}
        className="!bg-emerald-500 !w-3 !h-3"
      />

      <div className="flex items-center gap-2 mb-2">
        <div className="rounded-md bg-emerald-100 p-1.5">
          <Zap className="h-4 w-4 text-emerald-600" />
        </div>
        <span className="text-sm font-medium text-emerald-700">Action</span>
        {isStart && (
          <Star className="h-3 w-3 text-green-500 fill-green-500 ml-auto" />
        )}
      </div>

      {content && (
        <p className="text-sm text-gray-600 line-clamp-2 mb-2">
          {content}
        </p>
      )}

      {/* Actions Preview */}
      {actions && actions.length > 0 && (
        <div className="flex flex-wrap gap-1 mt-2">
          {actions.map((action, index) => (
            <span
              key={index}
              className="inline-flex items-center gap-1 px-2 py-0.5 text-xs bg-emerald-50 text-emerald-700 rounded-full border border-emerald-200"
            >
              {actionIcons[action.type]}
              {actionLabels[action.type]}
            </span>
          ))}
        </div>
      )}

      <Handle
        type="source"
        position={Position.Bottom}
        className="!bg-emerald-500 !w-3 !h-3"
      />
    </div>
  )
})
