'use client'

import { useEffect } from 'react'
import { useForm } from 'react-hook-form'
import { zodResolver } from '@hookform/resolvers/zod'
import { z } from 'zod'
import { useMutation, useQueryClient } from '@tanstack/react-query'
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
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { api } from '@/lib/api'
import { queryKeys } from '@/lib/query'
import { toast } from '@/hooks/use-toast'
import type { KnowledgeBase, CreateKnowledgeBaseInput, UpdateKnowledgeBaseInput } from '@/types'

const formSchema = z.object({
  name: z.string().min(1, 'Name is required').max(100, 'Name must be 100 characters or less'),
  description: z.string().max(500, 'Description must be 500 characters or less').optional(),
  type: z.enum(['faq', 'documents', 'website'], {
    required_error: 'Please select a type',
  }),
})

type FormValues = z.infer<typeof formSchema>

interface KnowledgeBaseFormProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  knowledgeBase?: KnowledgeBase | null
}

export function KnowledgeBaseForm({ open, onOpenChange, knowledgeBase }: KnowledgeBaseFormProps) {
  const queryClient = useQueryClient()
  const isEditing = !!knowledgeBase

  const form = useForm<FormValues>({
    resolver: zodResolver(formSchema),
    defaultValues: {
      name: '',
      description: '',
      type: 'faq',
    },
  })

  // Reset form when dialog opens/closes or when editing different KB
  useEffect(() => {
    if (open) {
      if (knowledgeBase) {
        form.reset({
          name: knowledgeBase.name,
          description: knowledgeBase.description || '',
          type: knowledgeBase.type,
        })
      } else {
        form.reset({
          name: '',
          description: '',
          type: 'faq',
        })
      }
    }
  }, [open, knowledgeBase, form])

  const createMutation = useMutation({
    mutationFn: (data: CreateKnowledgeBaseInput) =>
      api.post<KnowledgeBase>('/knowledge-bases', data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: queryKeys.knowledgeBases.all })
      toast({
        title: 'Knowledge base created',
        description: 'Your knowledge base has been created successfully.',
      })
      onOpenChange(false)
    },
    onError: (error: Error) => {
      toast({
        title: 'Error',
        description: error.message || 'Failed to create knowledge base',
        variant: 'error',
      })
    },
  })

  const updateMutation = useMutation({
    mutationFn: (data: UpdateKnowledgeBaseInput) =>
      api.put<KnowledgeBase>(`/knowledge-bases/${knowledgeBase?.id}`, data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: queryKeys.knowledgeBases.all })
      toast({
        title: 'Knowledge base updated',
        description: 'Your knowledge base has been updated successfully.',
      })
      onOpenChange(false)
    },
    onError: (error: Error) => {
      toast({
        title: 'Error',
        description: error.message || 'Failed to update knowledge base',
        variant: 'error',
      })
    },
  })

  const onSubmit = (values: FormValues) => {
    if (isEditing) {
      updateMutation.mutate({
        name: values.name,
        description: values.description,
      })
    } else {
      createMutation.mutate({
        name: values.name,
        description: values.description,
        type: values.type,
      })
    }
  }

  const isSubmitting = createMutation.isPending || updateMutation.isPending

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-[500px]">
        <DialogHeader>
          <DialogTitle>
            {isEditing ? 'Edit Knowledge Base' : 'Create Knowledge Base'}
          </DialogTitle>
          <DialogDescription>
            {isEditing
              ? 'Update the details of your knowledge base.'
              : 'Create a new knowledge base to store FAQs, documents, or website content for AI-powered responses.'}
          </DialogDescription>
        </DialogHeader>

        <Form {...form}>
          <form onSubmit={form.handleSubmit(onSubmit)} className="space-y-4">
            <FormField
              control={form.control}
              name="name"
              render={({ field }) => (
                <FormItem>
                  <FormLabel>Name</FormLabel>
                  <FormControl>
                    <Input placeholder="e.g., Product FAQ" {...field} />
                  </FormControl>
                  <FormMessage />
                </FormItem>
              )}
            />

            <FormField
              control={form.control}
              name="description"
              render={({ field }) => (
                <FormItem>
                  <FormLabel>Description</FormLabel>
                  <FormControl>
                    <Textarea
                      placeholder="Describe what this knowledge base contains..."
                      className="resize-none"
                      rows={3}
                      {...field}
                    />
                  </FormControl>
                  <FormDescription>
                    A brief description to help identify this knowledge base.
                  </FormDescription>
                  <FormMessage />
                </FormItem>
              )}
            />

            <FormField
              control={form.control}
              name="type"
              render={({ field }) => (
                <FormItem>
                  <FormLabel>Type</FormLabel>
                  <Select
                    onValueChange={field.onChange}
                    defaultValue={field.value}
                    disabled={isEditing}
                  >
                    <FormControl>
                      <SelectTrigger>
                        <SelectValue placeholder="Select a type" />
                      </SelectTrigger>
                    </FormControl>
                    <SelectContent>
                      <SelectItem value="faq">
                        <div className="flex flex-col">
                          <span>FAQ</span>
                          <span className="text-xs text-muted-foreground">
                            Question and answer pairs
                          </span>
                        </div>
                      </SelectItem>
                      <SelectItem value="documents">
                        <div className="flex flex-col">
                          <span>Documents</span>
                          <span className="text-xs text-muted-foreground">
                            PDF, Word, or text documents
                          </span>
                        </div>
                      </SelectItem>
                      <SelectItem value="website">
                        <div className="flex flex-col">
                          <span>Website</span>
                          <span className="text-xs text-muted-foreground">
                            Crawled website content
                          </span>
                        </div>
                      </SelectItem>
                    </SelectContent>
                  </Select>
                  {isEditing && (
                    <FormDescription>
                      Type cannot be changed after creation.
                    </FormDescription>
                  )}
                  <FormMessage />
                </FormItem>
              )}
            />

            <DialogFooter>
              <Button type="button" variant="outline" onClick={() => onOpenChange(false)}>
                Cancel
              </Button>
              <Button type="submit" disabled={isSubmitting}>
                {isSubmitting ? 'Saving...' : isEditing ? 'Save Changes' : 'Create'}
              </Button>
            </DialogFooter>
          </form>
        </Form>
      </DialogContent>
    </Dialog>
  )
}
