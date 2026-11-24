'use client'

import { memo } from 'react'
import { Handle, Position } from '@xyflow/react'
import { GitBranch, Star } from 'lucide-react'
import { cn } from '@/lib/utils'
import type { FlowTransition } from '@/types'

interface ConditionNodeProps {
  data: {
    content: string
    transitions?: FlowTransition[]
    isStart?: boolean
    isSelected?: boolean
  }
}

export const ConditionNode = memo(function ConditionNode({ data }: ConditionNodeProps) {
  const { content, transitions, isStart, isSelected } = data

  return (
    <div
      className={cn(
        'rounded-lg border-2 bg-card text-card-foreground p-4 shadow-sm transition-shadow min-w-[200px] max-w-[280px]',
        isSelected ? 'border-amber-500 shadow-md' : 'border-amber-500/30',
        isStart && 'ring-2 ring-green-500 ring-offset-2 ring-offset-background'
      )}
    >
      <Handle
        type="target"
        position={Position.Top}
        className="!bg-amber-500 !w-3 !h-3"
      />

      <div className="flex items-center gap-2 mb-2">
        <div className="rounded-md bg-amber-500/15 p-1.5">
          <GitBranch className="h-4 w-4 text-amber-600 dark:text-amber-400" />
        </div>
        <span className="text-sm font-medium text-amber-700 dark:text-amber-300">Condition</span>
        {isStart && (
          <Star className="h-3 w-3 text-green-600 dark:text-green-400 fill-green-500 ml-auto" />
        )}
      </div>

      <p className="text-sm text-muted-foreground line-clamp-2 mb-2">
        {content || 'Click to edit condition...'}
      </p>

      {/* Branches Preview */}
      {transitions && transitions.length > 0 && (
        <div className="text-xs text-muted-foreground mt-2">
          {transitions.length} {transitions.length === 1 ? 'branch' : 'branches'}
        </div>
      )}

      <Handle
        type="source"
        position={Position.Bottom}
        className="!bg-amber-500 !w-3 !h-3"
      />
    </div>
  )
})
