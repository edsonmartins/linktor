'use client'

import { useEffect, useState } from 'react'
import { useForm } from 'react-hook-form'
import { zodResolver } from '@hookform/resolvers/zod'
import { z } from 'zod'
import { useMutation, useQueryClient } from '@tanstack/react-query'
import { X, Plus } from 'lucide-react'
import { useTranslations } from 'next-intl'
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

function createFormSchema(t: ReturnType<typeof useTranslations<'knowledgeBase.item'>>) {
  return z.object({
    question: z.string().min(1, t('questionRequired')).max(1000, t('questionMaxLength')),
    answer: z.string().min(1, t('answerRequired')).max(10000, t('answerMaxLength')),
    source: z.string().max(200).optional(),
  })
}

type FormValues = z.infer<ReturnType<typeof createFormSchema>>

interface KnowledgeItemFormProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  knowledgeBaseId: string
  item?: KnowledgeItem | null
}

export function KnowledgeItemForm({ open, onOpenChange, knowledgeBaseId, item }: KnowledgeItemFormProps) {
  const t = useTranslations('knowledgeBase.item')
  const tCommon = useTranslations('common')
  const queryClient = useQueryClient()
  const isEditing = !!item
  const formSchema = createFormSchema(t)

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
        title: t('itemCreated'),
        description: t('itemCreatedDesc'),
      })
      onOpenChange(false)
    },
    onError: (error: Error) => {
      toast({
        title: tCommon('error'),
        description: error.message || t('failedCreate'),
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
        title: t('itemUpdated'),
        description: t('itemUpdatedDesc'),
      })
      onOpenChange(false)
    },
    onError: (error: Error) => {
      toast({
        title: tCommon('error'),
        description: error.message || t('failedUpdate'),
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
            {isEditing ? t('editTitle') : t('addTitle')}
          </DialogTitle>
          <DialogDescription>
            {isEditing ? t('editDesc') : t('addDesc')}
          </DialogDescription>
        </DialogHeader>

        <Form {...form}>
          <form onSubmit={form.handleSubmit(onSubmit)} className="space-y-4">
            <FormField
              control={form.control}
              name="question"
              render={({ field }) => (
                <FormItem>
                  <FormLabel>{t('question')}</FormLabel>
                  <FormControl>
                    <Textarea
                      placeholder={t('questionPlaceholder')}
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
                  <FormLabel>{t('answer')}</FormLabel>
                  <FormControl>
                    <Textarea
                      placeholder={t('answerPlaceholder')}
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
              <FormLabel>{t('keywordsOptional')}</FormLabel>
              <div className="flex gap-2">
                <Input
                  placeholder={t('keywordPlaceholder')}
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
                {t('keywordsHint')}
              </FormDescription>
            </div>

            <FormField
              control={form.control}
              name="source"
              render={({ field }) => (
                <FormItem>
                  <FormLabel>{t('sourceOptional')}</FormLabel>
                  <FormControl>
                    <Input placeholder={t('sourcePlaceholder')} {...field} />
                  </FormControl>
                  <FormDescription>
                    {t('sourceHint')}
                  </FormDescription>
                  <FormMessage />
                </FormItem>
              )}
            />

            <DialogFooter>
              <Button type="button" variant="outline" onClick={() => onOpenChange(false)}>
                {tCommon('cancel')}
              </Button>
              <Button type="submit" disabled={isSubmitting}>
                {isSubmitting ? t('saving') : isEditing ? t('saveChanges') : t('addItem')}
              </Button>
            </DialogFooter>
          </form>
        </Form>
      </DialogContent>
    </Dialog>
  )
}
