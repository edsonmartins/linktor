'use client'

import Link from 'next/link'
import { BookOpen, FileText, Globe, MoreVertical, Pencil, Trash2, RefreshCw, Search } from 'lucide-react'
import { useTranslations } from 'next-intl'
import { Card, CardContent, CardHeader } from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu'
import type { KnowledgeBase } from '@/types'
import { formatDistanceToNow } from '@/lib/utils'

interface KnowledgeBaseCardProps {
  knowledgeBase: KnowledgeBase
  onEdit: () => void
  onDelete: () => void
  onRegenerate: () => void
  isRegenerating?: boolean
}

const typeIcons = {
  faq: BookOpen,
  documents: FileText,
  website: Globe,
}

const statusVariants: Record<string, 'default' | 'success' | 'warning' | 'error'> = {
  active: 'success',
  syncing: 'warning',
  error: 'error',
}

export function KnowledgeBaseCard({
  knowledgeBase,
  onEdit,
  onDelete,
  onRegenerate,
  isRegenerating,
}: KnowledgeBaseCardProps) {
  const t = useTranslations('knowledgeBase.card')
  const tKb = useTranslations('knowledgeBase')
  const Icon = typeIcons[knowledgeBase.type] || BookOpen

  const typeLabels: Record<string, string> = {
    faq: tKb('faq'),
    documents: tKb('documents'),
    website: tKb('website'),
  }

  return (
    <Card className="group transition-colors hover:border-primary/30">
      <CardHeader className="pb-3">
        <div className="flex items-start justify-between">
          <div className="flex items-center gap-3">
            <div className="flex h-10 w-10 items-center justify-center rounded-lg bg-primary/10">
              <Icon className="h-5 w-5 text-primary" />
            </div>
            <div>
              <Link
                href={`/knowledge-base/${knowledgeBase.id}`}
                className="font-medium hover:text-primary hover:underline"
              >
                {knowledgeBase.name}
              </Link>
              <div className="flex items-center gap-2 mt-1">
                <Badge variant="secondary" className="text-xs">
                  {typeLabels[knowledgeBase.type]}
                </Badge>
                <Badge variant={statusVariants[knowledgeBase.status]} className="text-xs">
                  {knowledgeBase.status}
                </Badge>
              </div>
            </div>
          </div>

          <DropdownMenu>
            <DropdownMenuTrigger asChild>
              <Button
                variant="ghost"
                size="icon"
                className="h-8 w-8 opacity-0 group-hover:opacity-100"
              >
                <MoreVertical className="h-4 w-4" />
              </Button>
            </DropdownMenuTrigger>
            <DropdownMenuContent align="end">
              <DropdownMenuItem asChild>
                <Link href={`/knowledge-base/${knowledgeBase.id}`}>
                  <FileText className="mr-2 h-4 w-4" />
                  {t('viewItems')}
                </Link>
              </DropdownMenuItem>
              <DropdownMenuItem asChild>
                <Link href={`/knowledge-base/${knowledgeBase.id}/search`}>
                  <Search className="mr-2 h-4 w-4" />
                  {t('testSearch')}
                </Link>
              </DropdownMenuItem>
              <DropdownMenuSeparator />
              <DropdownMenuItem onClick={onEdit}>
                <Pencil className="mr-2 h-4 w-4" />
                {t('edit')}
              </DropdownMenuItem>
              <DropdownMenuItem onClick={onRegenerate} disabled={isRegenerating}>
                <RefreshCw className={`mr-2 h-4 w-4 ${isRegenerating ? 'animate-spin' : ''}`} />
                {t('regenerateEmbeddings')}
              </DropdownMenuItem>
              <DropdownMenuSeparator />
              <DropdownMenuItem onClick={onDelete} className="text-destructive">
                <Trash2 className="mr-2 h-4 w-4" />
                {t('delete')}
              </DropdownMenuItem>
            </DropdownMenuContent>
          </DropdownMenu>
        </div>
      </CardHeader>

      <CardContent>
        {knowledgeBase.description && (
          <p className="mb-3 text-sm text-muted-foreground line-clamp-2">
            {knowledgeBase.description}
          </p>
        )}

        <div className="flex items-center justify-between text-sm text-muted-foreground">
          <span>{t('items', { count: knowledgeBase.item_count })}</span>
          {knowledgeBase.last_sync_at ? (
            <span>{t('synced', { time: formatDistanceToNow(knowledgeBase.last_sync_at) })}</span>
          ) : (
            <span>{t('neverSynced')}</span>
          )}
        </div>
      </CardContent>
    </Card>
  )
}
