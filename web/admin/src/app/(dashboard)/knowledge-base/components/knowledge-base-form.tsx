'use client'

import { useEffect } from 'react'
import { useForm } from 'react-hook-form'
import { zodResolver } from '@hookform/resolvers/zod'
import { z } from 'zod'
import { useMutation, useQueryClient } from '@tanstack/react-query'
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

function createFormSchema(t: ReturnType<typeof useTranslations<'knowledgeBase.form'>>) {
  return z.object({
    name: z.string().min(1, t('nameRequired')).max(100, t('nameMaxLength')),
    description: z.string().max(500, t('descMaxLength')).optional(),
    type: z.enum(['faq', 'documents', 'website'], {
      required_error: t('selectType'),
    }),
  })
}

type FormValues = z.infer<ReturnType<typeof createFormSchema>>

interface KnowledgeBaseFormProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  knowledgeBase?: KnowledgeBase | null
}

export function KnowledgeBaseForm({ open, onOpenChange, knowledgeBase }: KnowledgeBaseFormProps) {
  const t = useTranslations('knowledgeBase.form')
  const tCommon = useTranslations('common')
  const queryClient = useQueryClient()
  const isEditing = !!knowledgeBase
  const formSchema = createFormSchema(t)

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
        title: t('created'),
        description: t('createdDesc'),
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
    mutationFn: (data: UpdateKnowledgeBaseInput) =>
      api.put<KnowledgeBase>(`/knowledge-bases/${knowledgeBase?.id}`, data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: queryKeys.knowledgeBases.all })
      toast({
        title: t('updated'),
        description: t('updatedDesc'),
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
            {isEditing ? t('editTitle') : t('createTitle')}
          </DialogTitle>
          <DialogDescription>
            {isEditing ? t('editDesc') : t('createDesc')}
          </DialogDescription>
        </DialogHeader>

        <Form {...form}>
          <form onSubmit={form.handleSubmit(onSubmit)} className="space-y-4">
            <FormField
              control={form.control}
              name="name"
              render={({ field }) => (
                <FormItem>
                  <FormLabel>{t('name')}</FormLabel>
                  <FormControl>
                    <Input placeholder={t('namePlaceholder')} {...field} />
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
                  <FormLabel>{t('description')}</FormLabel>
                  <FormControl>
                    <Textarea
                      placeholder={t('descPlaceholder')}
                      className="resize-none"
                      rows={3}
                      {...field}
                    />
                  </FormControl>
                  <FormDescription>
                    {t('descHint')}
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
                  <FormLabel>{t('type')}</FormLabel>
                  <Select
                    onValueChange={field.onChange}
                    defaultValue={field.value}
                    disabled={isEditing}
                  >
                    <FormControl>
                      <SelectTrigger>
                        <SelectValue placeholder={t('selectTypePlaceholder')} />
                      </SelectTrigger>
                    </FormControl>
                    <SelectContent>
                      <SelectItem value="faq">
                        <div className="flex flex-col">
                          <span>{t('faq')}</span>
                          <span className="text-xs text-muted-foreground">
                            {t('faqDesc')}
                          </span>
                        </div>
                      </SelectItem>
                      <SelectItem value="documents">
                        <div className="flex flex-col">
                          <span>{t('documents')}</span>
                          <span className="text-xs text-muted-foreground">
                            {t('documentsDesc')}
                          </span>
                        </div>
                      </SelectItem>
                      <SelectItem value="website">
                        <div className="flex flex-col">
                          <span>{t('website')}</span>
                          <span className="text-xs text-muted-foreground">
                            {t('websiteDesc')}
                          </span>
                        </div>
                      </SelectItem>
                    </SelectContent>
                  </Select>
                  {isEditing && (
                    <FormDescription>
                      {t('typeCannotChange')}
                    </FormDescription>
                  )}
                  <FormMessage />
                </FormItem>
              )}
            />

            <DialogFooter>
              <Button type="button" variant="outline" onClick={() => onOpenChange(false)}>
                {tCommon('cancel')}
              </Button>
              <Button type="submit" disabled={isSubmitting}>
                {isSubmitting ? t('saving') : isEditing ? t('saveChanges') : t('create')}
              </Button>
            </DialogFooter>
          </form>
        </Form>
      </DialogContent>
    </Dialog>
  )
}
