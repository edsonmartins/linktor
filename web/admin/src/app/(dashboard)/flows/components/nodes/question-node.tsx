'use client'

import { memo } from 'react'
import { Handle, Position } from '@xyflow/react'
import { HelpCircle, Star } from 'lucide-react'
import { cn } from '@/lib/utils'
import type { QuickReply } from '@/types'

interface QuestionNodeProps {
  data: {
    content: string
    quick_replies?: QuickReply[]
    isStart?: boolean
    isSelected?: boolean
  }
}

export const QuestionNode = memo(function QuestionNode({ data }: QuestionNodeProps) {
  const { content, quick_replies, isStart, isSelected } = data

  return (
    <div
      className={cn(
        'rounded-lg border-2 bg-card text-card-foreground p-4 shadow-sm transition-shadow min-w-[200px] max-w-[280px]',
        isSelected ? 'border-purple-500 shadow-md' : 'border-purple-500/30',
        isStart && 'ring-2 ring-green-500 ring-offset-2 ring-offset-background'
      )}
    >
      <Handle
        type="target"
        position={Position.Top}
        className="!bg-purple-500 !w-3 !h-3"
      />

      <div className="flex items-center gap-2 mb-2">
        <div className="rounded-md bg-purple-500/15 p-1.5">
          <HelpCircle className="h-4 w-4 text-purple-600 dark:text-purple-400" />
        </div>
        <span className="text-sm font-medium text-purple-700 dark:text-purple-300">Question</span>
        {isStart && (
          <Star className="h-3 w-3 text-green-600 dark:text-green-400 fill-green-500 ml-auto" />
        )}
      </div>

      <p className="text-sm text-muted-foreground line-clamp-2 mb-2">
        {content || 'Click to edit question...'}
      </p>

      {/* Quick Replies Preview */}
      {quick_replies && quick_replies.length > 0 && (
        <div className="flex flex-wrap gap-1 mt-2">
          {quick_replies.slice(0, 3).map((reply) => (
            <span
              key={reply.id}
              className="inline-block px-2 py-0.5 text-xs bg-purple-500/10 text-purple-700 dark:text-purple-300 rounded-full border border-purple-500/30"
            >
              {reply.title}
            </span>
          ))}
          {quick_replies.length > 3 && (
            <span className="text-xs text-muted-foreground">
              +{quick_replies.length - 3} more
            </span>
          )}
        </div>
      )}

      <Handle
        type="source"
        position={Position.Bottom}
        className="!bg-purple-500 !w-3 !h-3"
      />
    </div>
  )
})
