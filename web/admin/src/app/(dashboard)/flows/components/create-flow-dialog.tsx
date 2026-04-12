'use client'

import { useState } from 'react'
import { useMutation, useQueryClient } from '@tanstack/react-query'
import { useRouter } from 'next/navigation'
import { Loader2 } from 'lucide-react'
import { useTranslations } from 'next-intl'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
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
import { api } from '@/lib/api'
import { queryKeys } from '@/lib/query'
import type { Flow, FlowTriggerType, CreateFlowInput } from '@/types'

interface CreateFlowDialogProps {
  open: boolean
  onOpenChange: (open: boolean) => void
}

export function CreateFlowDialog({ open, onOpenChange }: CreateFlowDialogProps) {
  const t = useTranslations('flows.createDialog')
  const tCommon = useTranslations('common')
  const router = useRouter()
  const queryClient = useQueryClient()

  const triggerOptions: { value: FlowTriggerType; label: string; description: string }[] = [
    { value: 'welcome', label: t('triggers.welcome'), description: t('triggers.welcomeDesc') },
    { value: 'keyword', label: t('triggers.keyword'), description: t('triggers.keywordDesc') },
    { value: 'intent', label: t('triggers.intent'), description: t('triggers.intentDesc') },
    { value: 'manual', label: t('triggers.manual'), description: t('triggers.manualDesc') },
  ]

  const [name, setName] = useState('')
  const [description, setDescription] = useState('')
  const [trigger, setTrigger] = useState<FlowTriggerType>('keyword')
  const [triggerValue, setTriggerValue] = useState('')

  const createMutation = useMutation({
    mutationFn: (input: CreateFlowInput) => api.post<Flow>('/flows', input),
    onSuccess: (flow) => {
      queryClient.invalidateQueries({ queryKey: queryKeys.flows.all })
      onOpenChange(false)
      // Navigate to editor
      router.push(`/flows/${flow.id}`)
    },
  })

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault()

    // Create with a basic start node
    const startNodeId = 'start'
    const input: CreateFlowInput = {
      name,
      description: description || undefined,
      trigger,
      trigger_value: triggerValue || undefined,
      start_node_id: startNodeId,
      nodes: [
        {
          id: startNodeId,
          type: 'message',
          content: 'Hello! How can I help you today?',
          transitions: [],
          position: { x: 250, y: 50 },
        },
      ],
    }

    createMutation.mutate(input)
  }

  const resetForm = () => {
    setName('')
    setDescription('')
    setTrigger('keyword')
    setTriggerValue('')
  }

  const handleOpenChange = (isOpen: boolean) => {
    if (!isOpen) {
      resetForm()
    }
    onOpenChange(isOpen)
  }

  const showTriggerValue = trigger === 'keyword' || trigger === 'intent'

  return (
    <Dialog open={open} onOpenChange={handleOpenChange}>
      <DialogContent className="sm:max-w-[500px]">
        <form onSubmit={handleSubmit}>
          <DialogHeader>
            <DialogTitle>{t('title')}</DialogTitle>
            <DialogDescription>
              {t('description')}
            </DialogDescription>
          </DialogHeader>

          <div className="grid gap-4 py-4">
            {/* Name */}
            <div className="grid gap-2">
              <Label htmlFor="name">{t('name')}</Label>
              <Input
                id="name"
                placeholder={t('namePlaceholder')}
                value={name}
                onChange={(e) => setName(e.target.value)}
                required
              />
            </div>

            {/* Description */}
            <div className="grid gap-2">
              <Label htmlFor="description">{t('descriptionOptional')}</Label>
              <Textarea
                id="description"
                placeholder={t('descPlaceholder')}
                value={description}
                onChange={(e) => setDescription(e.target.value)}
                rows={2}
              />
            </div>

            {/* Trigger Type */}
            <div className="grid gap-2">
              <Label htmlFor="trigger">{t('trigger')}</Label>
              <Select value={trigger} onValueChange={(v) => setTrigger(v as FlowTriggerType)}>
                <SelectTrigger>
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  {triggerOptions.map((option) => (
                    <SelectItem key={option.value} value={option.value}>
                      <div className="flex flex-col">
                        <span>{option.label}</span>
                        <span className="text-xs text-muted-foreground">{option.description}</span>
                      </div>
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </div>

            {/* Trigger Value */}
            {showTriggerValue && (
              <div className="grid gap-2">
                <Label htmlFor="triggerValue">
                  {trigger === 'keyword' ? t('keywordLabel') : t('intentLabel')}
                </Label>
                <Input
                  id="triggerValue"
                  placeholder={trigger === 'keyword' ? t('keywordPlaceholder') : t('intentPlaceholder')}
                  value={triggerValue}
                  onChange={(e) => setTriggerValue(e.target.value)}
                />
                <p className="text-xs text-muted-foreground">
                  {trigger === 'keyword' ? t('keywordHint') : t('intentHint')}
                </p>
              </div>
            )}
          </div>

          <DialogFooter>
            <Button type="button" variant="outline" onClick={() => handleOpenChange(false)}>
              {tCommon('cancel')}
            </Button>
            <Button type="submit" disabled={!name || createMutation.isPending}>
              {createMutation.isPending && <Loader2 className="mr-2 h-4 w-4 animate-spin" />}
              {t('createAndEdit')}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  )
}
