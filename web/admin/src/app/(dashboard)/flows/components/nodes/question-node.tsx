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
        'rounded-lg border-2 bg-white p-4 shadow-sm transition-shadow min-w-[200px] max-w-[280px]',
        isSelected ? 'border-purple-500 shadow-md' : 'border-purple-200',
        isStart && 'ring-2 ring-green-500 ring-offset-2'
      )}
    >
      <Handle
        type="target"
        position={Position.Top}
        className="!bg-purple-500 !w-3 !h-3"
      />

      <div className="flex items-center gap-2 mb-2">
        <div className="rounded-md bg-purple-100 p-1.5">
          <HelpCircle className="h-4 w-4 text-purple-600" />
        </div>
        <span className="text-sm font-medium text-purple-700">Question</span>
        {isStart && (
          <Star className="h-3 w-3 text-green-500 fill-green-500 ml-auto" />
        )}
      </div>

      <p className="text-sm text-gray-600 line-clamp-2 mb-2">
        {content || 'Click to edit question...'}
      </p>

      {/* Quick Replies Preview */}
      {quick_replies && quick_replies.length > 0 && (
        <div className="flex flex-wrap gap-1 mt-2">
          {quick_replies.slice(0, 3).map((reply) => (
            <span
              key={reply.id}
              className="inline-block px-2 py-0.5 text-xs bg-purple-50 text-purple-700 rounded-full border border-purple-200"
            >
              {reply.title}
            </span>
          ))}
          {quick_replies.length > 3 && (
            <span className="text-xs text-gray-400">
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
