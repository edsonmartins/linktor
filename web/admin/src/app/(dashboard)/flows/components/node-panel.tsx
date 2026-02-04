'use client'

import { useState, useEffect } from 'react'
import {
  X,
  Trash2,
  Plus,
  MessageSquare,
  HelpCircle,
  GitBranch,
  Zap,
  CircleStop,
} from 'lucide-react'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Textarea } from '@/components/ui/textarea'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { Separator } from '@/components/ui/separator'
import { ScrollArea } from '@/components/ui/scroll-area'
import type { FlowNode, FlowTransition, QuickReply, FlowAction, FlowActionType, TransitionCondition } from '@/types'

interface NodePanelProps {
  node: FlowNode
  allNodes: FlowNode[]
  onUpdate: (node: FlowNode) => void
  onDelete: () => void
  onClose: () => void
}

const nodeTypeLabels: Record<FlowNode['type'], { icon: React.ReactNode; label: string }> = {
  message: { icon: <MessageSquare className="h-4 w-4" />, label: 'Message' },
  question: { icon: <HelpCircle className="h-4 w-4" />, label: 'Question' },
  condition: { icon: <GitBranch className="h-4 w-4" />, label: 'Condition' },
  action: { icon: <Zap className="h-4 w-4" />, label: 'Action' },
  end: { icon: <CircleStop className="h-4 w-4" />, label: 'End' },
}

const actionTypes: { value: FlowActionType; label: string }[] = [
  { value: 'tag', label: 'Add Tag' },
  { value: 'assign', label: 'Assign to User' },
  { value: 'escalate', label: 'Escalate' },
  { value: 'set_entity', label: 'Set Entity' },
  { value: 'http_call', label: 'HTTP Call' },
]

const conditionTypes: { value: TransitionCondition; label: string }[] = [
  { value: 'default', label: 'Default (always)' },
  { value: 'reply_equals', label: 'Reply equals' },
  { value: 'contains', label: 'Contains' },
  { value: 'regex', label: 'Regex match' },
]

export function NodePanel({ node, allNodes, onUpdate, onDelete, onClose }: NodePanelProps) {
  const [localNode, setLocalNode] = useState<FlowNode>(node)

  // Sync local state when external node changes
  useEffect(() => {
    setLocalNode(node)
  }, [node])

  // Debounced update
  const handleChange = (updates: Partial<FlowNode>) => {
    const updated = { ...localNode, ...updates }
    setLocalNode(updated)
    onUpdate(updated)
  }

  // Quick Replies
  const handleAddQuickReply = () => {
    const newReply: QuickReply = {
      id: `reply-${Date.now()}`,
      title: 'New option',
    }
    handleChange({
      quick_replies: [...(localNode.quick_replies || []), newReply],
    })
  }

  const handleUpdateQuickReply = (index: number, updates: Partial<QuickReply>) => {
    const replies = [...(localNode.quick_replies || [])]
    replies[index] = { ...replies[index], ...updates }
    handleChange({ quick_replies: replies })
  }

  const handleDeleteQuickReply = (index: number) => {
    const replies = [...(localNode.quick_replies || [])]
    replies.splice(index, 1)
    handleChange({ quick_replies: replies })
  }

  // Transitions
  const handleAddTransition = () => {
    const newTransition: FlowTransition = {
      id: `trans-${Date.now()}`,
      to_node_id: '',
      condition: 'default',
    }
    handleChange({
      transitions: [...localNode.transitions, newTransition],
    })
  }

  const handleUpdateTransition = (index: number, updates: Partial<FlowTransition>) => {
    const transitions = [...localNode.transitions]
    transitions[index] = { ...transitions[index], ...updates }
    handleChange({ transitions })
  }

  const handleDeleteTransition = (index: number) => {
    const transitions = [...localNode.transitions]
    transitions.splice(index, 1)
    handleChange({ transitions })
  }

  // Actions
  const handleAddAction = () => {
    const newAction: FlowAction = {
      type: 'tag',
      config: {},
    }
    handleChange({
      actions: [...(localNode.actions || []), newAction],
    })
  }

  const handleUpdateAction = (index: number, updates: Partial<FlowAction>) => {
    const actions = [...(localNode.actions || [])]
    actions[index] = { ...actions[index], ...updates }
    handleChange({ actions })
  }

  const handleDeleteAction = (index: number) => {
    const actions = [...(localNode.actions || [])]
    actions.splice(index, 1)
    handleChange({ actions })
  }

  const otherNodes = allNodes.filter((n) => n.id !== node.id)
  const typeInfo = nodeTypeLabels[localNode.type]

  return (
    <div className="w-80 border-l bg-background flex flex-col">
      {/* Header */}
      <div className="flex items-center justify-between p-4 border-b">
        <div className="flex items-center gap-2">
          {typeInfo.icon}
          <span className="font-medium">{typeInfo.label} Node</span>
        </div>
        <Button variant="ghost" size="icon" onClick={onClose}>
          <X className="h-4 w-4" />
        </Button>
      </div>

      <ScrollArea className="flex-1">
        <div className="p-4 space-y-6">
          {/* Node ID */}
          <div className="space-y-2">
            <Label className="text-muted-foreground text-xs">Node ID</Label>
            <code className="text-xs bg-muted px-2 py-1 rounded block">{localNode.id}</code>
          </div>

          {/* Content (for non-end nodes) */}
          {localNode.type !== 'end' && (
            <div className="space-y-2">
              <Label htmlFor="content">Content</Label>
              <Textarea
                id="content"
                value={localNode.content}
                onChange={(e) => handleChange({ content: e.target.value })}
                placeholder="Enter message content..."
                rows={3}
              />
            </div>
          )}

          {/* Quick Replies (for question nodes) */}
          {localNode.type === 'question' && (
            <>
              <Separator />
              <div className="space-y-3">
                <div className="flex items-center justify-between">
                  <Label>Quick Replies</Label>
                  <Button variant="outline" size="sm" onClick={handleAddQuickReply}>
                    <Plus className="h-3 w-3 mr-1" />
                    Add
                  </Button>
                </div>

                {localNode.quick_replies?.map((reply, index) => (
                  <div key={reply.id} className="flex gap-2">
                    <Input
                      value={reply.title}
                      onChange={(e) => handleUpdateQuickReply(index, { title: e.target.value })}
                      placeholder="Button text"
                      className="flex-1"
                    />
                    <Button
                      variant="ghost"
                      size="icon"
                      onClick={() => handleDeleteQuickReply(index)}
                    >
                      <Trash2 className="h-3 w-3 text-destructive" />
                    </Button>
                  </div>
                ))}
              </div>
            </>
          )}

          {/* Actions (for action nodes) */}
          {localNode.type === 'action' && (
            <>
              <Separator />
              <div className="space-y-3">
                <div className="flex items-center justify-between">
                  <Label>Actions</Label>
                  <Button variant="outline" size="sm" onClick={handleAddAction}>
                    <Plus className="h-3 w-3 mr-1" />
                    Add
                  </Button>
                </div>

                {localNode.actions?.map((action, index) => (
                  <div key={index} className="space-y-2 p-3 border rounded-lg">
                    <div className="flex items-center gap-2">
                      <Select
                        value={action.type}
                        onValueChange={(v) => handleUpdateAction(index, { type: v as FlowActionType })}
                      >
                        <SelectTrigger className="flex-1">
                          <SelectValue />
                        </SelectTrigger>
                        <SelectContent>
                          {actionTypes.map((type) => (
                            <SelectItem key={type.value} value={type.value}>
                              {type.label}
                            </SelectItem>
                          ))}
                        </SelectContent>
                      </Select>
                      <Button
                        variant="ghost"
                        size="icon"
                        onClick={() => handleDeleteAction(index)}
                      >
                        <Trash2 className="h-3 w-3 text-destructive" />
                      </Button>
                    </div>

                    {/* Action config based on type */}
                    {action.type === 'tag' && (
                      <Input
                        value={(action.config.tag as string) || ''}
                        onChange={(e) =>
                          handleUpdateAction(index, {
                            config: { ...action.config, tag: e.target.value },
                          })
                        }
                        placeholder="Tag name"
                      />
                    )}
                    {action.type === 'escalate' && (
                      <Select
                        value={(action.config.priority as string) || 'normal'}
                        onValueChange={(v) =>
                          handleUpdateAction(index, {
                            config: { ...action.config, priority: v },
                          })
                        }
                      >
                        <SelectTrigger>
                          <SelectValue placeholder="Priority" />
                        </SelectTrigger>
                        <SelectContent>
                          <SelectItem value="low">Low</SelectItem>
                          <SelectItem value="normal">Normal</SelectItem>
                          <SelectItem value="high">High</SelectItem>
                          <SelectItem value="urgent">Urgent</SelectItem>
                        </SelectContent>
                      </Select>
                    )}
                  </div>
                ))}
              </div>
            </>
          )}

          {/* Transitions (for non-end nodes) */}
          {localNode.type !== 'end' && (
            <>
              <Separator />
              <div className="space-y-3">
                <div className="flex items-center justify-between">
                  <Label>Transitions</Label>
                  <Button variant="outline" size="sm" onClick={handleAddTransition}>
                    <Plus className="h-3 w-3 mr-1" />
                    Add
                  </Button>
                </div>

                {localNode.transitions.map((transition, index) => (
                  <div key={transition.id} className="space-y-2 p-3 border rounded-lg">
                    <div className="flex items-center gap-2">
                      <Select
                        value={transition.to_node_id || ''}
                        onValueChange={(v) => handleUpdateTransition(index, { to_node_id: v })}
                      >
                        <SelectTrigger className="flex-1">
                          <SelectValue placeholder="Select target node" />
                        </SelectTrigger>
                        <SelectContent>
                          {otherNodes.map((n) => (
                            <SelectItem key={n.id} value={n.id}>
                              {n.type}: {n.content?.substring(0, 20) || n.id}
                            </SelectItem>
                          ))}
                        </SelectContent>
                      </Select>
                      <Button
                        variant="ghost"
                        size="icon"
                        onClick={() => handleDeleteTransition(index)}
                      >
                        <Trash2 className="h-3 w-3 text-destructive" />
                      </Button>
                    </div>

                    <Select
                      value={transition.condition}
                      onValueChange={(v) =>
                        handleUpdateTransition(index, { condition: v as TransitionCondition })
                      }
                    >
                      <SelectTrigger>
                        <SelectValue />
                      </SelectTrigger>
                      <SelectContent>
                        {conditionTypes.map((type) => (
                          <SelectItem key={type.value} value={type.value}>
                            {type.label}
                          </SelectItem>
                        ))}
                      </SelectContent>
                    </Select>

                    {transition.condition !== 'default' && (
                      <Input
                        value={transition.value || ''}
                        onChange={(e) =>
                          handleUpdateTransition(index, { value: e.target.value })
                        }
                        placeholder="Match value"
                      />
                    )}
                  </div>
                ))}

                {localNode.transitions.length === 0 && (
                  <p className="text-sm text-muted-foreground text-center py-2">
                    No transitions. Add one or connect nodes on the canvas.
                  </p>
                )}
              </div>
            </>
          )}
        </div>
      </ScrollArea>

      {/* Footer */}
      <div className="p-4 border-t">
        <Button
          variant="destructive"
          className="w-full"
          onClick={onDelete}
        >
          <Trash2 className="h-4 w-4 mr-2" />
          Delete Node
        </Button>
      </div>
    </div>
  )
}
