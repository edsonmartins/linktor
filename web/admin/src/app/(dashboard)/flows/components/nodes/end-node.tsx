'use client'

import { memo } from 'react'
import { Handle, Position } from '@xyflow/react'
import { CircleStop } from 'lucide-react'
import { cn } from '@/lib/utils'

interface EndNodeProps {
  data: {
    isSelected?: boolean
  }
}

export const EndNode = memo(function EndNode({ data }: EndNodeProps) {
  const { isSelected } = data

  return (
    <div
      className={cn(
        'rounded-lg border-2 bg-white p-4 shadow-sm transition-shadow min-w-[120px]',
        isSelected ? 'border-red-500 shadow-md' : 'border-red-200'
      )}
    >
      <Handle
        type="target"
        position={Position.Top}
        className="!bg-red-500 !w-3 !h-3"
      />

      <div className="flex items-center justify-center gap-2">
        <div className="rounded-md bg-red-100 p-1.5">
          <CircleStop className="h-4 w-4 text-red-600" />
        </div>
        <span className="text-sm font-medium text-red-700">End</span>
      </div>
    </div>
  )
})
