'use client'

import { memo } from 'react'
import { Handle, Position } from '@xyflow/react'
import { ImageIcon, Star } from 'lucide-react'
import { cn } from '@/lib/utils'
import type { VRETemplateId } from '@/types'

interface VRENodeProps {
  data: {
    content: string
    vre_config?: {
      template_id: VRETemplateId
      data_mapping?: Record<string, string>
      caption?: string
      follow_up_text?: string
    }
    isStart?: boolean
    isSelected?: boolean
  }
}

const templateLabels: Record<VRETemplateId, string> = {
  menu_opcoes: 'Menu Options',
  card_produto: 'Product Card',
  status_pedido: 'Order Status',
  lista_produtos: 'Product List',
  confirmacao: 'Confirmation',
  cobranca_pix: 'PIX Payment',
}

export const VRENode = memo(function VRENode({ data }: VRENodeProps) {
  const { content, vre_config, isStart, isSelected } = data

  const templateLabel = vre_config?.template_id
    ? templateLabels[vre_config.template_id]
    : 'Select template...'

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
          <ImageIcon className="h-4 w-4 text-purple-600" />
        </div>
        <span className="text-sm font-medium text-purple-700">VRE Visual</span>
        {isStart && (
          <Star className="h-3 w-3 text-green-500 fill-green-500 ml-auto" />
        )}
      </div>

      <div className="text-xs text-purple-600 font-medium mb-1">
        {templateLabel}
      </div>

      <p className="text-sm text-gray-600 line-clamp-2">
        {content || 'Visual response image'}
      </p>

      <Handle
        type="source"
        position={Position.Bottom}
        className="!bg-purple-500 !w-3 !h-3"
      />
    </div>
  )
})
