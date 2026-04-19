'use client'

import { use } from 'react'
import Link from 'next/link'
import { useQuery } from '@tanstack/react-query'
import { useTranslations } from 'next-intl'
import { ArrowLeft } from 'lucide-react'
import { Header } from '@/components/layout/header'
import { Skeleton } from '@/components/ui/skeleton'
import { Button } from '@/components/ui/button'
import { api } from '@/lib/api'
import { queryKeys } from '@/lib/query'
import type { Template } from '@/types'
import { TemplateForm } from '../../components/template-form'

export default function EditTemplatePage({
  params,
}: {
  params: Promise<{ id: string }>
}) {
  const { id } = use(params)
  const t = useTranslations('templates')

  const { data: template, isLoading } = useQuery({
    queryKey: queryKeys.templates.detail(id),
    queryFn: () => api.get<Template>(`/templates/${id}`),
  })

  if (isLoading) {
    return (
      <div className="flex h-full flex-col">
        <Header title={t('detail.loading')} />
        <div className="space-y-4 p-6">
          <Skeleton className="h-8 w-64" />
          <Skeleton className="h-48 w-full" />
        </div>
      </div>
    )
  }

  if (!template) {
    return (
      <div className="flex h-full flex-col">
        <Header title={t('detail.notFound')} />
        <div className="p-6">
          <Link href="/templates">
            <Button variant="outline">
              <ArrowLeft className="mr-2 h-4 w-4" />
              {t('backToList')}
            </Button>
          </Link>
        </div>
      </div>
    )
  }

  return (
    <div className="flex h-full flex-col">
      <Header title={t('edit')}>
        <Link
          href={`/templates/${id}`}
          className="inline-flex items-center gap-1 text-sm text-muted-foreground hover:text-foreground"
        >
          <ArrowLeft className="h-4 w-4" />
          {t('backToDetail')}
        </Link>
      </Header>
      <TemplateForm mode="edit" template={template} />
    </div>
  )
}
