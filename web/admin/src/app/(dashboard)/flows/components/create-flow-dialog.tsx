'use client'

import { useState } from 'react'
import { useMutation, useQueryClient } from '@tanstack/react-query'
import { useRouter } from 'next/navigation'
import { Loader2 } from 'lucide-react'
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

const triggerOptions: { value: FlowTriggerType; label: string; description: string }[] = [
  { value: 'welcome', label: 'Welcome Message', description: 'Triggered when a new conversation starts' },
  { value: 'keyword', label: 'Keyword', description: 'Triggered when a specific keyword is detected' },
  { value: 'intent', label: 'Intent', description: 'Triggered when an intent is recognized by AI' },
  { value: 'manual', label: 'Manual', description: 'Triggered manually by agents or other flows' },
]

export function CreateFlowDialog({ open, onOpenChange }: CreateFlowDialogProps) {
  const router = useRouter()
  const queryClient = useQueryClient()

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
            <DialogTitle>Create New Flow</DialogTitle>
            <DialogDescription>
              Create a conversational flow to guide customers through a decision tree.
            </DialogDescription>
          </DialogHeader>

          <div className="grid gap-4 py-4">
            {/* Name */}
            <div className="grid gap-2">
              <Label htmlFor="name">Name</Label>
              <Input
                id="name"
                placeholder="e.g., Support Ticket Flow"
                value={name}
                onChange={(e) => setName(e.target.value)}
                required
              />
            </div>

            {/* Description */}
            <div className="grid gap-2">
              <Label htmlFor="description">Description (optional)</Label>
              <Textarea
                id="description"
                placeholder="Describe what this flow does..."
                value={description}
                onChange={(e) => setDescription(e.target.value)}
                rows={2}
              />
            </div>

            {/* Trigger Type */}
            <div className="grid gap-2">
              <Label htmlFor="trigger">Trigger</Label>
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
                  {trigger === 'keyword' ? 'Keyword' : 'Intent Name'}
                </Label>
                <Input
                  id="triggerValue"
                  placeholder={trigger === 'keyword' ? 'e.g., support, help' : 'e.g., request_support'}
                  value={triggerValue}
                  onChange={(e) => setTriggerValue(e.target.value)}
                />
                <p className="text-xs text-muted-foreground">
                  {trigger === 'keyword'
                    ? 'The keyword that will trigger this flow (case-insensitive)'
                    : 'The intent name that will trigger this flow'}
                </p>
              </div>
            )}
          </div>

          <DialogFooter>
            <Button type="button" variant="outline" onClick={() => handleOpenChange(false)}>
              Cancel
            </Button>
            <Button type="submit" disabled={!name || createMutation.isPending}>
              {createMutation.isPending && <Loader2 className="mr-2 h-4 w-4 animate-spin" />}
              Create & Edit
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  )
}
