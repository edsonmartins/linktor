'use client'

import Link from 'next/link'
import { useTranslations } from 'next-intl'
import { ArrowLeft } from 'lucide-react'
import { Header } from '@/components/layout/header'
import { TemplateForm } from '../components/template-form'

export default function NewTemplatePage() {
  const t = useTranslations('templates')

  return (
    <div className="flex h-full flex-col">
      <Header title={t('create')}>
        <Link
          href="/templates"
          className="inline-flex items-center gap-1 text-sm text-muted-foreground hover:text-foreground"
        >
          <ArrowLeft className="h-4 w-4" />
          {t('backToList')}
        </Link>
      </Header>
      <TemplateForm mode="create" />
    </div>
  )
}
