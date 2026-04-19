'use client'

import { useState } from 'react'
import Link from 'next/link'
import { useRouter } from 'next/navigation'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { useTranslations } from 'next-intl'
import { ArrowLeft, Search, Sparkles } from 'lucide-react'
import { Header } from '@/components/layout/header'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Badge } from '@/components/ui/badge'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Skeleton } from '@/components/ui/skeleton'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { useToast } from '@/hooks/use-toast'
import { api } from '@/lib/api'
import { queryKeys } from '@/lib/query'
import type { Channel, Template, TemplateCategory } from '@/types'

// Library catalog entry as returned by GET /templates/library.
// Kept narrow — only the fields we render in the picker. The backend
// passes the full Meta payload through in case later phases need more.
interface LibraryTemplate {
  name: string
  category: string
  language?: string
  industry?: string[]
  topic?: string
  usecase?: string
  body_text?: string
  header_text?: string
}

export default function TemplateLibraryPage() {
  const t = useTranslations('templates')
  const tLib = useTranslations('templates.library')
  const tCommon = useTranslations('common')
  const router = useRouter()
  const { toast } = useToast()
  const queryClient = useQueryClient()

  const [channelId, setChannelId] = useState<string>('')
  const [search, setSearch] = useState('')
  const [topic, setTopic] = useState('')
  const [usecase, setUsecase] = useState('')
  const [industry, setIndustry] = useState('')
  const [language, setLanguage] = useState('')

  const [selected, setSelected] = useState<LibraryTemplate | null>(null)

  const { data: channelsData } = useQuery({
    queryKey: queryKeys.channels.list(),
    queryFn: () => api.get<Channel[]>('/channels'),
  })
  const whatsappChannels = (channelsData ?? []).filter(
    (c) => c.type === 'whatsapp_official',
  )

  // Default the channel to the only candidate so the page is usable with
  // no clicks when the tenant has a single WhatsApp line.
  if (channelId === '' && whatsappChannels.length === 1) {
    setChannelId(whatsappChannels[0].id)
  }

  const filters = {
    channel_id: channelId,
    ...(search && { search }),
    ...(topic && { topic }),
    ...(usecase && { usecase }),
    ...(industry && { industry }),
    ...(language && { language }),
  }

  const {
    data: libraryData,
    isLoading,
    isFetching,
  } = useQuery({
    queryKey: queryKeys.templates.library(filters),
    queryFn: () =>
      api.get<LibraryTemplate[]>('/templates/library', filters),
    enabled: channelId !== '',
  })

  const items = libraryData ?? []

  return (
    <div className="flex h-full flex-col">
      <Header title={tLib('title')}>
        <Link
          href="/templates"
          className="inline-flex items-center gap-1 text-sm text-muted-foreground hover:text-foreground"
        >
          <ArrowLeft className="h-4 w-4" />
          {t('backToList')}
        </Link>
      </Header>

      <div className="flex flex-col gap-4 border-b p-4">
        <div className="grid grid-cols-1 gap-3 md:grid-cols-6">
          <div className="md:col-span-2 space-y-1">
            <Label className="text-xs">{t('form.channel')}</Label>
            <Select value={channelId} onValueChange={setChannelId}>
              <SelectTrigger>
                <SelectValue placeholder={t('form.channelPlaceholder')} />
              </SelectTrigger>
              <SelectContent>
                {whatsappChannels.map((c) => (
                  <SelectItem key={c.id} value={c.id}>
                    {c.name}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          </div>

          <div className="md:col-span-2 space-y-1">
            <Label className="text-xs">{tLib('search')}</Label>
            <div className="relative">
              <Search className="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground" />
              <Input
                value={search}
                onChange={(e) => setSearch(e.target.value)}
                placeholder={tLib('searchPlaceholder')}
                className="pl-9"
              />
            </div>
          </div>

          <div className="space-y-1">
            <Label className="text-xs">{tLib('language')}</Label>
            <Input
              value={language}
              onChange={(e) => setLanguage(e.target.value)}
              placeholder="pt_BR"
            />
          </div>
          <div className="space-y-1">
            <Label className="text-xs">{tLib('industry')}</Label>
            <Input
              value={industry}
              onChange={(e) => setIndustry(e.target.value)}
              placeholder={tLib('industryPlaceholder')}
            />
          </div>
        </div>

        <div className="grid grid-cols-1 gap-3 md:grid-cols-2">
          <div className="space-y-1">
            <Label className="text-xs">{tLib('topic')}</Label>
            <Input value={topic} onChange={(e) => setTopic(e.target.value)} placeholder={tLib('topicPlaceholder')} />
          </div>
          <div className="space-y-1">
            <Label className="text-xs">{tLib('usecase')}</Label>
            <Input value={usecase} onChange={(e) => setUsecase(e.target.value)} placeholder={tLib('usecasePlaceholder')} />
          </div>
        </div>
      </div>

      <div className="flex-1 overflow-auto p-4">
        {channelId === '' ? (
          <EmptyNoChannel />
        ) : isLoading || isFetching ? (
          <div className="grid grid-cols-1 gap-4 md:grid-cols-2 lg:grid-cols-3">
            {[1, 2, 3, 4, 5, 6].map((i) => (
              <Skeleton key={i} className="h-40" />
            ))}
          </div>
        ) : items.length === 0 ? (
          <EmptyNoResults />
        ) : (
          <div className="grid grid-cols-1 gap-4 md:grid-cols-2 lg:grid-cols-3">
            {items.map((item) => (
              <LibraryCard
                key={item.name}
                template={item}
                onPick={() => setSelected(item)}
              />
            ))}
          </div>
        )}
      </div>

      <InstantiateDialog
        open={selected !== null}
        libraryTemplate={selected}
        channelId={channelId}
        onClose={() => setSelected(null)}
        onCreated={(tpl) => {
          queryClient.invalidateQueries({ queryKey: queryKeys.templates.all })
          toast({ title: tCommon('success'), description: t('created') })
          router.push(`/templates/${tpl.id}`)
        }}
      />
    </div>
  )
}

function LibraryCard({
  template,
  onPick,
}: {
  template: LibraryTemplate
  onPick: () => void
}) {
  const tLib = useTranslations('templates.library')

  return (
    <Card className="flex h-full flex-col">
      <CardHeader>
        <CardTitle className="flex items-start justify-between gap-2 text-base">
          <span className="truncate">{template.name}</span>
          <Badge variant="outline">{template.category}</Badge>
        </CardTitle>
        <div className="flex flex-wrap gap-1 text-xs text-muted-foreground">
          {template.language && <span>{template.language}</span>}
          {template.topic && (
            <>
              <span>·</span>
              <span>{template.topic}</span>
            </>
          )}
          {template.usecase && (
            <>
              <span>·</span>
              <span>{template.usecase}</span>
            </>
          )}
        </div>
      </CardHeader>
      <CardContent className="flex flex-1 flex-col justify-between gap-3">
        {template.header_text && (
          <p className="text-sm font-medium">{template.header_text}</p>
        )}
        {template.body_text && (
          <p className="line-clamp-4 whitespace-pre-wrap text-sm text-muted-foreground">
            {template.body_text}
          </p>
        )}
        <Button variant="outline" className="mt-auto" onClick={onPick}>
          <Sparkles className="mr-2 h-4 w-4" />
          {tLib('pick')}
        </Button>
      </CardContent>
    </Card>
  )
}

function EmptyNoChannel() {
  const tLib = useTranslations('templates.library')

  return (
    <div className="flex flex-col items-center justify-center rounded-lg border border-dashed py-16 text-center">
      <Sparkles className="mb-4 h-12 w-12 text-muted-foreground" />
      <h3 className="text-lg font-medium">{tLib('empty.selectChannel')}</h3>
    </div>
  )
}

function EmptyNoResults() {
  const tLib = useTranslations('templates.library')

  return (
    <div className="flex flex-col items-center justify-center rounded-lg border border-dashed py-16 text-center">
      <Sparkles className="mb-4 h-12 w-12 text-muted-foreground" />
      <h3 className="text-lg font-medium">{tLib('empty.noResults')}</h3>
      <p className="mt-1 max-w-sm text-sm text-muted-foreground">
        {tLib('empty.noResultsHint')}
      </p>
    </div>
  )
}

function InstantiateDialog({
  open,
  libraryTemplate,
  channelId,
  onClose,
  onCreated,
}: {
  open: boolean
  libraryTemplate: LibraryTemplate | null
  channelId: string
  onClose: () => void
  onCreated: (t: Template) => void
}) {
  const tLib = useTranslations('templates.library')
  const tCommon = useTranslations('common')
  const { toast } = useToast()

  const [name, setName] = useState('')
  const [language, setLanguage] = useState('pt_BR')

  // Reset when the user picks a different library template.
  if (libraryTemplate && name === '' && libraryTemplate.name) {
    setName(libraryTemplate.name)
    if (libraryTemplate.language) setLanguage(libraryTemplate.language)
  }
  if (!libraryTemplate && name !== '') {
    setName('')
  }

  const mutation = useMutation({
    mutationFn: () =>
      api.post<Template>('/templates/library', {
        channel_id: channelId,
        name,
        language,
        category: libraryTemplate?.category as TemplateCategory,
        library_template_name: libraryTemplate?.name,
      }),
    onSuccess: (tpl) => {
      onCreated(tpl)
      onClose()
    },
    onError: (err: Error) => {
      toast({
        title: tCommon('error'),
        description: err.message,
        variant: 'error',
      })
    },
  })

  return (
    <Dialog open={open} onOpenChange={(o) => !o && onClose()}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>{tLib('dialog.title')}</DialogTitle>
          <DialogDescription>
            {libraryTemplate
              ? tLib('dialog.description', { name: libraryTemplate.name })
              : ''}
          </DialogDescription>
        </DialogHeader>

        <div className="space-y-4">
          <div className="space-y-2">
            <Label htmlFor="lib-name">{tLib('dialog.name')}</Label>
            <Input
              id="lib-name"
              value={name}
              onChange={(e) =>
                setName(e.target.value.toLowerCase().replace(/\s+/g, '_'))
              }
              placeholder="my_custom_name"
              pattern="[a-z0-9_]+"
            />
            <p className="text-xs text-muted-foreground">
              {tLib('dialog.nameHint')}
            </p>
          </div>

          <div className="space-y-2">
            <Label htmlFor="lib-language">{tLib('dialog.language')}</Label>
            <Input
              id="lib-language"
              value={language}
              onChange={(e) => setLanguage(e.target.value)}
              placeholder="pt_BR"
            />
          </div>
        </div>

        <DialogFooter>
          <Button variant="outline" onClick={onClose}>
            {tCommon('cancel')}
          </Button>
          <Button
            onClick={() => mutation.mutate()}
            disabled={!name.trim() || !language.trim() || mutation.isPending}
          >
            {mutation.isPending ? tCommon('loading') : tLib('dialog.create')}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}
