'use client'

import {
  MessageSquare,
  HelpCircle,
  GitBranch,
  Zap,
  CircleStop,
  ImageIcon,
} from 'lucide-react'
import { Button } from '@/components/ui/button'
import {
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from '@/components/ui/tooltip'
import type { FlowNodeType } from '@/types'

interface NodeToolbarProps {
  onAddNode: (type: FlowNodeType) => void
}

const nodeTypes: { type: FlowNodeType; icon: React.ReactNode; label: string; description: string }[] = [
  {
    type: 'message',
    icon: <MessageSquare className="h-5 w-5" />,
    label: 'Message',
    description: 'Send a message to the user',
  },
  {
    type: 'question',
    icon: <HelpCircle className="h-5 w-5" />,
    label: 'Question',
    description: 'Ask a question with quick reply options',
  },
  {
    type: 'condition',
    icon: <GitBranch className="h-5 w-5" />,
    label: 'Condition',
    description: 'Branch based on user response',
  },
  {
    type: 'action',
    icon: <Zap className="h-5 w-5" />,
    label: 'Action',
    description: 'Execute actions like tagging or escalation',
  },
  {
    type: 'vre',
    icon: <ImageIcon className="h-5 w-5" />,
    label: 'VRE Visual',
    description: 'Send a visual response image (menu, product card, etc.)',
  },
  {
    type: 'end',
    icon: <CircleStop className="h-5 w-5" />,
    label: 'End',
    description: 'End the flow',
  },
]

export function NodeToolbar({ onAddNode }: NodeToolbarProps) {
  return (
    <TooltipProvider>
      <div className="w-16 border-r bg-muted/30 p-2 flex flex-col gap-2">
        <div className="text-xs font-medium text-muted-foreground text-center mb-2">
          Nodes
        </div>

        {nodeTypes.map((nodeType) => (
          <Tooltip key={nodeType.type}>
            <TooltipTrigger asChild>
              <Button
                variant="outline"
                size="icon"
                className="h-10 w-10"
                onClick={() => onAddNode(nodeType.type)}
              >
                {nodeType.icon}
              </Button>
            </TooltipTrigger>
            <TooltipContent side="right">
              <div className="text-sm font-medium">{nodeType.label}</div>
              <div className="text-xs text-muted-foreground">
                {nodeType.description}
              </div>
            </TooltipContent>
          </Tooltip>
        ))}
      </div>
    </TooltipProvider>
  )
}
