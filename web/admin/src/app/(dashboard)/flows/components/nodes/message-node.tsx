'use client'

import { memo } from 'react'
import { Handle, Position } from '@xyflow/react'
import { MessageSquare, Star } from 'lucide-react'
import { cn } from '@/lib/utils'

interface MessageNodeProps {
  data: {
    content: string
    isStart?: boolean
    isSelected?: boolean
  }
}

export const MessageNode = memo(function MessageNode({ data }: MessageNodeProps) {
  const { content, isStart, isSelected } = data

  return (
    <div
      className={cn(
        'rounded-lg border-2 bg-white p-4 shadow-sm transition-shadow min-w-[200px] max-w-[280px]',
        isSelected ? 'border-blue-500 shadow-md' : 'border-blue-200',
        isStart && 'ring-2 ring-green-500 ring-offset-2'
      )}
    >
      <Handle
        type="target"
        position={Position.Top}
        className="!bg-blue-500 !w-3 !h-3"
      />

      <div className="flex items-center gap-2 mb-2">
        <div className="rounded-md bg-blue-100 p-1.5">
          <MessageSquare className="h-4 w-4 text-blue-600" />
        </div>
        <span className="text-sm font-medium text-blue-700">Message</span>
        {isStart && (
          <Star className="h-3 w-3 text-green-500 fill-green-500 ml-auto" />
        )}
      </div>

      <p className="text-sm text-gray-600 line-clamp-3">
        {content || 'Click to edit message...'}
      </p>

      <Handle
        type="source"
        position={Position.Bottom}
        className="!bg-blue-500 !w-3 !h-3"
      />
    </div>
  )
})
