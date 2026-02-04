'use client'

import { GitBranch, Play, Pause, Edit, Trash2, TestTube, MoreVertical, Zap, MessageSquare, Hash, Hand } from 'lucide-react'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { Badge } from '@/components/ui/badge'
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu'
import type { Flow, FlowTriggerType } from '@/types'

interface FlowCardProps {
  flow: Flow
  onEdit: () => void
  onToggleActive: () => void
  onDelete: () => void
  onTest: () => void
  isToggling?: boolean
}

const triggerIcons: Record<FlowTriggerType, React.ReactNode> = {
  welcome: <MessageSquare className="h-4 w-4" />,
  keyword: <Hash className="h-4 w-4" />,
  intent: <Zap className="h-4 w-4" />,
  manual: <Hand className="h-4 w-4" />,
}

const triggerLabels: Record<FlowTriggerType, string> = {
  welcome: 'Welcome Message',
  keyword: 'Keyword Trigger',
  intent: 'Intent Match',
  manual: 'Manual Trigger',
}

export function FlowCard({
  flow,
  onEdit,
  onToggleActive,
  onDelete,
  onTest,
  isToggling,
}: FlowCardProps) {
  const nodeCount = flow.nodes?.length || 0

  return (
    <Card className="group relative hover:shadow-md transition-shadow">
      <CardHeader className="pb-3">
        <div className="flex items-start justify-between">
          <div className="flex items-center gap-3">
            <div className={`rounded-lg p-2 ${flow.is_active ? 'bg-green-100 text-green-700' : 'bg-gray-100 text-gray-500'}`}>
              <GitBranch className="h-5 w-5" />
            </div>
            <div>
              <CardTitle className="text-base font-medium">{flow.name}</CardTitle>
              {flow.description && (
                <CardDescription className="text-sm line-clamp-1">
                  {flow.description}
                </CardDescription>
              )}
            </div>
          </div>

          <DropdownMenu>
            <DropdownMenuTrigger asChild>
              <Button variant="ghost" size="icon" className="h-8 w-8 opacity-0 group-hover:opacity-100">
                <MoreVertical className="h-4 w-4" />
              </Button>
            </DropdownMenuTrigger>
            <DropdownMenuContent align="end">
              <DropdownMenuItem onClick={onEdit}>
                <Edit className="mr-2 h-4 w-4" />
                Edit
              </DropdownMenuItem>
              <DropdownMenuItem onClick={onTest}>
                <TestTube className="mr-2 h-4 w-4" />
                Test
              </DropdownMenuItem>
              <DropdownMenuItem onClick={onToggleActive} disabled={isToggling}>
                {flow.is_active ? (
                  <>
                    <Pause className="mr-2 h-4 w-4" />
                    Deactivate
                  </>
                ) : (
                  <>
                    <Play className="mr-2 h-4 w-4" />
                    Activate
                  </>
                )}
              </DropdownMenuItem>
              <DropdownMenuSeparator />
              <DropdownMenuItem onClick={onDelete} className="text-destructive">
                <Trash2 className="mr-2 h-4 w-4" />
                Delete
              </DropdownMenuItem>
            </DropdownMenuContent>
          </DropdownMenu>
        </div>
      </CardHeader>

      <CardContent>
        <div className="flex flex-wrap items-center gap-2 text-sm">
          {/* Trigger Badge */}
          <Badge variant="secondary" className="gap-1">
            {triggerIcons[flow.trigger]}
            {triggerLabels[flow.trigger]}
          </Badge>

          {/* Trigger Value */}
          {flow.trigger_value && (
            <Badge variant="outline" className="font-mono text-xs">
              {flow.trigger_value}
            </Badge>
          )}

          {/* Node Count */}
          <Badge variant="outline">
            {nodeCount} {nodeCount === 1 ? 'node' : 'nodes'}
          </Badge>

          {/* Status */}
          <Badge variant={flow.is_active ? 'default' : 'secondary'} className="ml-auto">
            {flow.is_active ? 'Active' : 'Inactive'}
          </Badge>
        </div>

        {/* Quick Actions */}
        <div className="mt-4 flex gap-2">
          <Button variant="outline" size="sm" className="flex-1" onClick={onEdit}>
            <Edit className="mr-2 h-3 w-3" />
            Edit
          </Button>
          <Button variant="outline" size="sm" className="flex-1" onClick={onTest}>
            <TestTube className="mr-2 h-3 w-3" />
            Test
          </Button>
        </div>
      </CardContent>
    </Card>
  )
}
