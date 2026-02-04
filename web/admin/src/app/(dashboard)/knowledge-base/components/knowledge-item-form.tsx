'use client'

import { useEffect, useState } from 'react'
import { useForm } from 'react-hook-form'
import { zodResolver } from '@hookform/resolvers/zod'
import { z } from 'zod'
import { useMutation, useQueryClient } from '@tanstack/react-query'
import { X, Plus } from 'lucide-react'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import {
  Form,
  FormControl,
  FormDescription,
  FormField,
  FormItem,
  FormLabel,
  FormMessage,
} from '@/components/ui/form'
import { Input } from '@/components/ui/input'
import { Button } from '@/components/ui/button'
import { Textarea } from '@/components/ui/textarea'
import { Badge } from '@/components/ui/badge'
import { api } from '@/lib/api'
import { queryKeys } from '@/lib/query'
import { toast } from '@/hooks/use-toast'
import type { KnowledgeItem, CreateKnowledgeItemInput, UpdateKnowledgeItemInput } from '@/types'

const formSchema = z.object({
  question: z.string().min(1, 'Question is required').max(1000, 'Question must be 1000 characters or less'),
  answer: z.string().min(1, 'Answer is required').max(10000, 'Answer must be 10000 characters or less'),
  source: z.string().max(200).optional(),
})

type FormValues = z.infer<typeof formSchema>

interface KnowledgeItemFormProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  knowledgeBaseId: string
  item?: KnowledgeItem | null
}

export function KnowledgeItemForm({ open, onOpenChange, knowledgeBaseId, item }: KnowledgeItemFormProps) {
  const queryClient = useQueryClient()
  const isEditing = !!item

  const [keywords, setKeywords] = useState<string[]>([])
  const [keywordInput, setKeywordInput] = useState('')

  const form = useForm<FormValues>({
    resolver: zodResolver(formSchema),
    defaultValues: {
      question: '',
      answer: '',
      source: '',
    },
  })

  // Reset form when dialog opens/closes or when editing different item
  useEffect(() => {
    if (open) {
      if (item) {
        form.reset({
          question: item.question,
          answer: item.answer,
          source: item.source || '',
        })
        setKeywords(item.keywords || [])
      } else {
        form.reset({
          question: '',
          answer: '',
          source: '',
        })
        setKeywords([])
      }
      setKeywordInput('')
    }
  }, [open, item, form])

  const createMutation = useMutation({
    mutationFn: (data: CreateKnowledgeItemInput) =>
      api.post<KnowledgeItem>(`/knowledge-bases/${knowledgeBaseId}/items`, data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: queryKeys.knowledgeItems.list(knowledgeBaseId, {}) })
      queryClient.invalidateQueries({ queryKey: queryKeys.knowledgeBases.detail(knowledgeBaseId) })
      toast({
        title: 'Item created',
        description: 'The item has been added to the knowledge base.',
      })
      onOpenChange(false)
    },
    onError: (error: Error) => {
      toast({
        title: 'Error',
        description: error.message || 'Failed to create item',
        variant: 'error',
      })
    },
  })

  const updateMutation = useMutation({
    mutationFn: (data: UpdateKnowledgeItemInput) =>
      api.put<KnowledgeItem>(`/knowledge-bases/${knowledgeBaseId}/items/${item?.id}`, data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: queryKeys.knowledgeItems.list(knowledgeBaseId, {}) })
      toast({
        title: 'Item updated',
        description: 'The item has been updated successfully.',
      })
      onOpenChange(false)
    },
    onError: (error: Error) => {
      toast({
        title: 'Error',
        description: error.message || 'Failed to update item',
        variant: 'error',
      })
    },
  })

  const onSubmit = (values: FormValues) => {
    const data = {
      question: values.question,
      answer: values.answer,
      keywords: keywords.length > 0 ? keywords : undefined,
      source: values.source || undefined,
    }

    if (isEditing) {
      updateMutation.mutate(data)
    } else {
      createMutation.mutate(data)
    }
  }

  const addKeyword = () => {
    const keyword = keywordInput.trim().toLowerCase()
    if (keyword && !keywords.includes(keyword)) {
      setKeywords([...keywords, keyword])
      setKeywordInput('')
    }
  }

  const removeKeyword = (keywordToRemove: string) => {
    setKeywords(keywords.filter((k) => k !== keywordToRemove))
  }

  const handleKeywordKeyDown = (e: React.KeyboardEvent) => {
    if (e.key === 'Enter') {
      e.preventDefault()
      addKeyword()
    }
  }

  const isSubmitting = createMutation.isPending || updateMutation.isPending

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-[600px]">
        <DialogHeader>
          <DialogTitle>
            {isEditing ? 'Edit Item' : 'Add Item'}
          </DialogTitle>
          <DialogDescription>
            {isEditing
              ? 'Update the question and answer for this knowledge item.'
              : 'Add a new question-answer pair to your knowledge base.'}
          </DialogDescription>
        </DialogHeader>

        <Form {...form}>
          <form onSubmit={form.handleSubmit(onSubmit)} className="space-y-4">
            <FormField
              control={form.control}
              name="question"
              render={({ field }) => (
                <FormItem>
                  <FormLabel>Question</FormLabel>
                  <FormControl>
                    <Textarea
                      placeholder="e.g., How do I reset my password?"
                      className="resize-none"
                      rows={2}
                      {...field}
                    />
                  </FormControl>
                  <FormMessage />
                </FormItem>
              )}
            />

            <FormField
              control={form.control}
              name="answer"
              render={({ field }) => (
                <FormItem>
                  <FormLabel>Answer</FormLabel>
                  <FormControl>
                    <Textarea
                      placeholder="Provide a detailed answer..."
                      className="resize-none"
                      rows={4}
                      {...field}
                    />
                  </FormControl>
                  <FormMessage />
                </FormItem>
              )}
            />

            {/* Keywords */}
            <div className="space-y-2">
              <FormLabel>Keywords (optional)</FormLabel>
              <div className="flex gap-2">
                <Input
                  placeholder="Add a keyword..."
                  value={keywordInput}
                  onChange={(e) => setKeywordInput(e.target.value)}
                  onKeyDown={handleKeywordKeyDown}
                />
                <Button type="button" variant="outline" onClick={addKeyword}>
                  <Plus className="h-4 w-4" />
                </Button>
              </div>
              {keywords.length > 0 && (
                <div className="flex flex-wrap gap-1 mt-2">
                  {keywords.map((keyword) => (
                    <Badge key={keyword} variant="secondary" className="gap-1">
                      {keyword}
                      <button
                        type="button"
                        onClick={() => removeKeyword(keyword)}
                        className="ml-1 hover:text-destructive"
                      >
                        <X className="h-3 w-3" />
                      </button>
                    </Badge>
                  ))}
                </div>
              )}
              <FormDescription>
                Keywords help with search relevance. Press Enter to add.
              </FormDescription>
            </div>

            <FormField
              control={form.control}
              name="source"
              render={({ field }) => (
                <FormItem>
                  <FormLabel>Source (optional)</FormLabel>
                  <FormControl>
                    <Input placeholder="e.g., FAQ v1.0, Help Center" {...field} />
                  </FormControl>
                  <FormDescription>
                    Track where this information came from.
                  </FormDescription>
                  <FormMessage />
                </FormItem>
              )}
            />

            <DialogFooter>
              <Button type="button" variant="outline" onClick={() => onOpenChange(false)}>
                Cancel
              </Button>
              <Button type="submit" disabled={isSubmitting}>
                {isSubmitting ? 'Saving...' : isEditing ? 'Save Changes' : 'Add Item'}
              </Button>
            </DialogFooter>
          </form>
        </Form>
      </DialogContent>
    </Dialog>
  )
}
