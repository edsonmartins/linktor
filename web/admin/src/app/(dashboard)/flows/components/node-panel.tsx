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
  ImageIcon,
} from 'lucide-react'
import { useTranslations } from 'next-intl'
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
import type { FlowNode, FlowTransition, QuickReply, FlowAction, FlowActionType, TransitionCondition, VRETemplateId, VRENodeConfig } from '@/types'

interface NodePanelProps {
  node: FlowNode
  allNodes: FlowNode[]
  onUpdate: (node: FlowNode) => void
  onDelete: () => void
  onClose: () => void
}

const nodeTypeIcons: Record<FlowNode['type'], React.ReactNode> = {
  message: <MessageSquare className="h-4 w-4" />,
  question: <HelpCircle className="h-4 w-4" />,
  condition: <GitBranch className="h-4 w-4" />,
  action: <Zap className="h-4 w-4" />,
  vre: <ImageIcon className="h-4 w-4" />,
  end: <CircleStop className="h-4 w-4" />,
}

export function NodePanel({ node, allNodes, onUpdate, onDelete, onClose }: NodePanelProps) {
  const t = useTranslations('flows.nodePanel')
  const [localNode, setLocalNode] = useState<FlowNode>(node)

  const nodeTypeLabels: Record<FlowNode['type'], string> = {
    message: t('nodeTypes.message'),
    question: t('nodeTypes.question'),
    condition: t('nodeTypes.condition'),
    action: t('nodeTypes.action'),
    vre: t('nodeTypes.vreVisual'),
    end: t('nodeTypes.end'),
  }

  const actionTypes: { value: FlowActionType; label: string }[] = [
    { value: 'tag', label: t('actionTypes.add_tag') },
    { value: 'assign', label: t('actionTypes.assign_user') },
    { value: 'escalate', label: t('actionTypes.escalate') },
    { value: 'set_entity', label: t('actionTypes.set_entity') },
    { value: 'http_call', label: t('actionTypes.http_call') },
  ]

  const conditionTypes: { value: TransitionCondition; label: string }[] = [
    { value: 'default', label: t('conditionTypes.default') },
    { value: 'reply_equals', label: t('conditionTypes.equals') },
    { value: 'contains', label: t('conditionTypes.contains') },
    { value: 'regex', label: t('conditionTypes.regex') },
  ]

  const vreTemplateTypes: { value: VRETemplateId; label: string; description: string }[] = [
    { value: 'menu_opcoes', label: t('vreTemplates.menu_options'), description: t('vreTemplates.menu_optionsDesc') },
    { value: 'card_produto', label: t('vreTemplates.product_card'), description: t('vreTemplates.product_cardDesc') },
    { value: 'status_pedido', label: t('vreTemplates.order_status'), description: t('vreTemplates.order_statusDesc') },
    { value: 'lista_produtos', label: t('vreTemplates.product_list'), description: t('vreTemplates.product_listDesc') },
    { value: 'confirmacao', label: t('vreTemplates.confirmation'), description: t('vreTemplates.confirmationDesc') },
    { value: 'cobranca_pix', label: t('vreTemplates.pix_payment'), description: t('vreTemplates.pix_paymentDesc') },
  ]

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
      title: t('newOption'),
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
  const typeLabel = nodeTypeLabels[localNode.type]
  const typeIcon = nodeTypeIcons[localNode.type]

  return (
    <div className="w-80 border-l bg-background flex flex-col">
      {/* Header */}
      <div className="flex items-center justify-between p-4 border-b">
        <div className="flex items-center gap-2">
          {typeIcon}
          <span className="font-medium">{t('nodeTitle', { type: typeLabel })}</span>
        </div>
        <Button variant="ghost" size="icon" onClick={onClose}>
          <X className="h-4 w-4" />
        </Button>
      </div>

      <ScrollArea className="flex-1">
        <div className="p-4 space-y-6">
          {/* Node ID */}
          <div className="space-y-2">
            <Label className="text-muted-foreground text-xs">{t('nodeId')}</Label>
            <code className="text-xs bg-muted px-2 py-1 rounded block">{localNode.id}</code>
          </div>

          {/* Content (for non-end nodes) */}
          {localNode.type !== 'end' && (
            <div className="space-y-2">
              <Label htmlFor="content">{t('content')}</Label>
              <Textarea
                id="content"
                value={localNode.content}
                onChange={(e) => handleChange({ content: e.target.value })}
                placeholder={t('contentPlaceholder')}
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
                  <Label>{t('quickReplies')}</Label>
                  <Button variant="outline" size="sm" onClick={handleAddQuickReply}>
                    <Plus className="h-3 w-3 mr-1" />
                    {t('add')}
                  </Button>
                </div>

                {localNode.quick_replies?.map((reply, index) => (
                  <div key={reply.id} className="flex gap-2">
                    <Input
                      value={reply.title}
                      onChange={(e) => handleUpdateQuickReply(index, { title: e.target.value })}
                      placeholder={t('buttonText')}
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
                  <Label>{t('actions')}</Label>
                  <Button variant="outline" size="sm" onClick={handleAddAction}>
                    <Plus className="h-3 w-3 mr-1" />
                    {t('add')}
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
                          <SelectValue placeholder={t('priority')} />
                        </SelectTrigger>
                        <SelectContent>
                          <SelectItem value="low">{t('priorityLow')}</SelectItem>
                          <SelectItem value="normal">{t('priorityNormal')}</SelectItem>
                          <SelectItem value="high">{t('priorityHigh')}</SelectItem>
                          <SelectItem value="urgent">{t('priorityUrgent')}</SelectItem>
                        </SelectContent>
                      </Select>
                    )}
                  </div>
                ))}
              </div>
            </>
          )}

          {/* VRE Configuration (for vre nodes) */}
          {localNode.type === 'vre' && (
            <>
              <Separator />
              <div className="space-y-3">
                <Label>{t('vreTemplate')}</Label>
                <Select
                  value={localNode.vre_config?.template_id || ''}
                  onValueChange={(v) =>
                    handleChange({
                      vre_config: {
                        ...localNode.vre_config,
                        template_id: v as VRETemplateId,
                        data_mapping: localNode.vre_config?.data_mapping || {},
                      },
                    })
                  }
                >
                  <SelectTrigger>
                    <SelectValue placeholder={t('selectTemplate')} />
                  </SelectTrigger>
                  <SelectContent>
                    {vreTemplateTypes.map((template) => (
                      <SelectItem key={template.value} value={template.value}>
                        <div>
                          <div className="font-medium">{template.label}</div>
                          <div className="text-xs text-muted-foreground">{template.description}</div>
                        </div>
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>

                {localNode.vre_config?.template_id && (
                  <>
                    <div className="space-y-2">
                      <Label htmlFor="vre-caption">{t('captionOptional')}</Label>
                      <Input
                        id="vre-caption"
                        value={localNode.vre_config?.caption || ''}
                        onChange={(e) =>
                          handleChange({
                            vre_config: {
                              ...localNode.vre_config!,
                              caption: e.target.value,
                            },
                          })
                        }
                        placeholder={t('customCaption')}
                      />
                    </div>

                    <div className="space-y-2">
                      <Label htmlFor="vre-followup">{t('followUpOptional')}</Label>
                      <Input
                        id="vre-followup"
                        value={localNode.vre_config?.follow_up_text || ''}
                        onChange={(e) =>
                          handleChange({
                            vre_config: {
                              ...localNode.vre_config!,
                              follow_up_text: e.target.value,
                            },
                          })
                        }
                        placeholder={t('followUpDesc')}
                      />
                    </div>

                    <div className="text-xs text-muted-foreground p-2 bg-muted rounded">
                      {t('variableSyntax')}
                    </div>
                  </>
                )}
              </div>
            </>
          )}

          {/* Transitions (for non-end nodes) */}
          {localNode.type !== 'end' && (
            <>
              <Separator />
              <div className="space-y-3">
                <div className="flex items-center justify-between">
                  <Label>{t('transitions')}</Label>
                  <Button variant="outline" size="sm" onClick={handleAddTransition}>
                    <Plus className="h-3 w-3 mr-1" />
                    {t('add')}
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
                          <SelectValue placeholder={t('selectTarget')} />
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
                    {t('noTransitions')}
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
          {t('deleteNode')}
        </Button>
      </div>
    </div>
  )
}
