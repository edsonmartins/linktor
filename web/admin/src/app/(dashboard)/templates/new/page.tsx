'use client'

import { useState } from 'react'
import { useRouter } from 'next/navigation'
import Link from 'next/link'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { useTranslations } from 'next-intl'
import { ArrowLeft, Plus, Trash2, Info } from 'lucide-react'
import { Header } from '@/components/layout/header'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Textarea } from '@/components/ui/textarea'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Alert, AlertDescription } from '@/components/ui/alert'
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
import type {
  Channel,
  CreateTemplateRequest,
  Template,
  TemplateButton,
  TemplateButtonType,
  TemplateCategory,
} from '@/types'

// Client-side mirror of pkg/graphapi validation: count {{N}} placeholders in
// text so the admin can see at a glance how many examples are required.
const POSITIONAL = /\{\{\s*(\d+)\s*\}\}/g

function maxPositional(s: string): number {
  let max = 0
  for (const m of s.matchAll(POSITIONAL)) {
    const n = Number(m[1])
    if (Number.isFinite(n) && n > max) max = n
  }
  return max
}

interface ButtonDraft {
  type: 'QUICK_REPLY' | 'URL'
  text: string
  url?: string
}

export default function NewTemplatePage() {
  const t = useTranslations('templates')
  const tCommon = useTranslations('common')
  const router = useRouter()
  const { toast } = useToast()
  const queryClient = useQueryClient()

  const [channelId, setChannelId] = useState<string>('')
  const [name, setName] = useState('')
  const [language, setLanguage] = useState('pt_BR')
  const [category, setCategory] = useState<TemplateCategory>('UTILITY')
  const [bodyText, setBodyText] = useState('')
  const [bodyExamples, setBodyExamples] = useState<string[]>([])
  const [footerText, setFooterText] = useState('')
  const [buttons, setButtons] = useState<ButtonDraft[]>([])

  // Pull channels for the dropdown. We only show WhatsApp Official
  // because templates only live on that channel type in Meta.
  const { data: channelsData } = useQuery({
    queryKey: queryKeys.channels.list(),
    queryFn: () => api.get<Channel[]>('/channels'),
  })
  const whatsappChannels = (channelsData ?? []).filter(
    (c) => c.type === 'whatsapp_official',
  )

  // Keep the example array the same length as the placeholder count so the
  // form shows exactly one input per {{N}} — extra values are trimmed,
  // missing ones become empty strings.
  const variableCount = maxPositional(bodyText)
  const trimmedExamples = bodyExamples.slice(0, variableCount)
  while (trimmedExamples.length < variableCount) trimmedExamples.push('')

  const canSubmit =
    channelId !== '' &&
    name.trim() !== '' &&
    language.trim() !== '' &&
    bodyText.trim() !== '' &&
    trimmedExamples.every((v) => v.trim() !== '') &&
    buttons.every((b) => b.text.trim() !== '' && (b.type !== 'URL' || (b.url ?? '').trim() !== ''))

  const createMutation = useMutation({
    mutationFn: (payload: CreateTemplateRequest) =>
      api.post<Template>('/templates', payload),
    onSuccess: (template) => {
      queryClient.invalidateQueries({ queryKey: queryKeys.templates.all })
      toast({ title: tCommon('success'), description: t('created') })
      router.push(`/templates/${template.id}`)
    },
    onError: (err: Error) => {
      toast({
        title: tCommon('error'),
        description: err.message,
        variant: 'error',
      })
    },
  })

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault()
    if (!canSubmit) return

    const components: CreateTemplateRequest['components'] = [
      {
        type: 'BODY',
        text: bodyText,
        ...(variableCount > 0 && {
          example: { body_text: [trimmedExamples] },
        }),
      },
    ]
    if (footerText.trim() !== '') {
      components.push({ type: 'FOOTER', text: footerText })
    }
    if (buttons.length > 0) {
      components.push({
        type: 'BUTTONS',
        buttons: buttons.map<TemplateButton>((b) => ({
          type: b.type as TemplateButtonType,
          text: b.text,
          ...(b.type === 'URL' && { url: b.url }),
        })),
      })
    }

    createMutation.mutate({
      channel_id: channelId,
      name: name.trim(),
      language: language.trim(),
      category,
      components,
    })
  }

  const addButton = (type: ButtonDraft['type']) => {
    if (buttons.length >= 3) return // Meta caps button row at 3 for QUICK_REPLY/URL in phase 1
    setButtons([...buttons, { type, text: '', url: type === 'URL' ? '' : undefined }])
  }

  const updateButton = (index: number, patch: Partial<ButtonDraft>) => {
    setButtons(buttons.map((b, i) => (i === index ? { ...b, ...patch } : b)))
  }

  const removeButton = (index: number) => {
    setButtons(buttons.filter((_, i) => i !== index))
  }

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

      <form onSubmit={handleSubmit} className="flex-1 overflow-auto">
        <div className="mx-auto max-w-3xl space-y-6 p-6">
          {whatsappChannels.length === 0 && (
            <Alert>
              <Info className="h-4 w-4" />
              <AlertDescription>{t('empty.noChannel')}</AlertDescription>
            </Alert>
          )}

          <Card>
            <CardHeader>
              <CardTitle>{t('form.basics')}</CardTitle>
            </CardHeader>
            <CardContent className="space-y-4">
              <div className="space-y-2">
                <Label htmlFor="channel">{t('form.channel')}</Label>
                <Select value={channelId} onValueChange={setChannelId}>
                  <SelectTrigger id="channel">
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

              <div className="grid grid-cols-1 gap-4 md:grid-cols-2">
                <div className="space-y-2">
                  <Label htmlFor="name">{t('form.name')}</Label>
                  <Input
                    id="name"
                    value={name}
                    onChange={(e) => setName(e.target.value.toLowerCase().replace(/\s+/g, '_'))}
                    placeholder="order_confirmation"
                    pattern="[a-z0-9_]+"
                    required
                  />
                  <p className="text-xs text-muted-foreground">{t('form.nameHint')}</p>
                </div>
                <div className="space-y-2">
                  <Label htmlFor="language">{t('form.language')}</Label>
                  <Input
                    id="language"
                    value={language}
                    onChange={(e) => setLanguage(e.target.value)}
                    placeholder="pt_BR"
                    required
                  />
                  <p className="text-xs text-muted-foreground">{t('form.languageHint')}</p>
                </div>
              </div>

              <div className="space-y-2">
                <Label htmlFor="category">{t('form.category')}</Label>
                <Select value={category} onValueChange={(v) => setCategory(v as TemplateCategory)}>
                  <SelectTrigger id="category">
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectItem value="UTILITY">{t('category.UTILITY')}</SelectItem>
                    <SelectItem value="MARKETING">{t('category.MARKETING')}</SelectItem>
                    <SelectItem value="AUTHENTICATION">{t('category.AUTHENTICATION')}</SelectItem>
                  </SelectContent>
                </Select>
              </div>
            </CardContent>
          </Card>

          <Card>
            <CardHeader>
              <CardTitle>{t('form.body')}</CardTitle>
            </CardHeader>
            <CardContent className="space-y-4">
              <div className="space-y-2">
                <Label htmlFor="body">{t('form.bodyText')}</Label>
                <Textarea
                  id="body"
                  rows={4}
                  value={bodyText}
                  onChange={(e) => setBodyText(e.target.value)}
                  placeholder={t('form.bodyPlaceholder')}
                  required
                />
                <p className="text-xs text-muted-foreground">{t('form.bodyHint')}</p>
              </div>

              {variableCount > 0 && (
                <div className="space-y-2 rounded-md border border-dashed p-3">
                  <Label>{t('form.examples', { count: variableCount })}</Label>
                  <p className="text-xs text-muted-foreground">{t('form.examplesHint')}</p>
                  <div className="space-y-2">
                    {trimmedExamples.map((val, i) => (
                      <div key={i} className="flex items-center gap-2">
                        <span className="w-12 shrink-0 text-sm font-mono text-muted-foreground">
                          {`{{${i + 1}}}`}
                        </span>
                        <Input
                          value={val}
                          onChange={(e) => {
                            const next = [...trimmedExamples]
                            next[i] = e.target.value
                            setBodyExamples(next)
                          }}
                          placeholder={t('form.examplePlaceholder', { n: i + 1 })}
                        />
                      </div>
                    ))}
                  </div>
                </div>
              )}

              <div className="space-y-2">
                <Label htmlFor="footer">{t('form.footer')}</Label>
                <Input
                  id="footer"
                  value={footerText}
                  onChange={(e) => setFooterText(e.target.value)}
                  placeholder={t('form.footerPlaceholder')}
                />
                <p className="text-xs text-muted-foreground">{t('form.footerHint')}</p>
              </div>
            </CardContent>
          </Card>

          <Card>
            <CardHeader>
              <CardTitle className="flex items-center justify-between">
                {t('form.buttons')}
                <div className="flex gap-2">
                  <Button
                    type="button"
                    variant="outline"
                    size="sm"
                    onClick={() => addButton('QUICK_REPLY')}
                    disabled={buttons.length >= 3}
                  >
                    <Plus className="mr-2 h-3 w-3" />
                    {t('form.addQuickReply')}
                  </Button>
                  <Button
                    type="button"
                    variant="outline"
                    size="sm"
                    onClick={() => addButton('URL')}
                    disabled={buttons.length >= 3}
                  >
                    <Plus className="mr-2 h-3 w-3" />
                    {t('form.addURL')}
                  </Button>
                </div>
              </CardTitle>
            </CardHeader>
            <CardContent className="space-y-3">
              {buttons.length === 0 ? (
                <p className="text-sm text-muted-foreground">{t('form.noButtons')}</p>
              ) : (
                buttons.map((btn, i) => (
                  <div key={i} className="flex items-start gap-2 rounded-md border p-3">
                    <div className="flex-1 space-y-2">
                      <div className="flex items-center gap-2">
                        <span className="rounded bg-muted px-2 py-0.5 text-xs font-mono">
                          {btn.type}
                        </span>
                        <Input
                          value={btn.text}
                          onChange={(e) => updateButton(i, { text: e.target.value })}
                          placeholder={t('form.buttonTextPlaceholder')}
                          maxLength={25}
                        />
                      </div>
                      {btn.type === 'URL' && (
                        <Input
                          value={btn.url ?? ''}
                          onChange={(e) => updateButton(i, { url: e.target.value })}
                          placeholder="https://example.com/path"
                          type="url"
                        />
                      )}
                    </div>
                    <Button
                      type="button"
                      variant="ghost"
                      size="icon"
                      onClick={() => removeButton(i)}
                    >
                      <Trash2 className="h-4 w-4" />
                    </Button>
                  </div>
                ))
              )}
            </CardContent>
          </Card>

          <div className="flex justify-end gap-2">
            <Link href="/templates">
              <Button type="button" variant="outline">
                {tCommon('cancel')}
              </Button>
            </Link>
            <Button type="submit" disabled={!canSubmit || createMutation.isPending}>
              {createMutation.isPending ? tCommon('loading') : t('form.submit')}
            </Button>
          </div>
        </div>
      </form>
    </div>
  )
}
