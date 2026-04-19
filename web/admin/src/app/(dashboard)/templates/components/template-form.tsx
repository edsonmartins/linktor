'use client'

import { useState, useMemo } from 'react'
import { useRouter } from 'next/navigation'
import Link from 'next/link'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { useTranslations } from 'next-intl'
import { Plus, Trash2, Info } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Textarea } from '@/components/ui/textarea'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Alert, AlertDescription } from '@/components/ui/alert'
import { Switch } from '@/components/ui/switch'
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
  EditTemplateRequest,
  Template,
  TemplateButton,
  TemplateButtonType,
  TemplateCategory,
  TemplateComponent,
  TemplateHeaderFormat,
  TemplateParameterFormat,
} from '@/types'

// Placeholder matchers — kept client-side so the admin sees required
// example counts change as they type instead of waiting for Meta's 400.
const POSITIONAL = /\{\{\s*(\d+)\s*\}\}/g
const NAMED = /\{\{\s*([a-z][a-z0-9_]*)\s*\}\}/g

function positionalCount(s: string): number {
  let max = 0
  for (const m of s.matchAll(POSITIONAL)) {
    const n = Number(m[1])
    if (Number.isFinite(n) && n > max) max = n
  }
  return max
}

function namedVars(s: string): string[] {
  const seen = new Set<string>()
  for (const m of s.matchAll(NAMED)) seen.add(m[1])
  return Array.from(seen)
}

interface ButtonDraft {
  type: TemplateButtonType
  text: string
  url?: string
  phone_number?: string
  example?: string // coupon for COPY_CODE
  otp_type?: string // COPY_CODE only in phase 2
  flow_id?: string
  flow_action?: 'navigate' | 'data_exchange'
}

type HeaderDraft =
  | { kind: 'none' }
  | { kind: 'TEXT'; text: string; headerExamples: string[] }
  | { kind: 'IMAGE' | 'VIDEO' | 'DOCUMENT'; handle: string }
  | { kind: 'LOCATION' }

interface TemplateFormProps {
  mode: 'create' | 'edit'
  template?: Template // required when mode=edit
}

export function TemplateForm({ mode, template }: TemplateFormProps) {
  const t = useTranslations('templates')
  const tCommon = useTranslations('common')
  const router = useRouter()
  const { toast } = useToast()
  const queryClient = useQueryClient()

  // ---- Field state ---------------------------------------------------------
  const [channelId, setChannelId] = useState<string>(template?.channel_id ?? '')
  const [name, setName] = useState(template?.name ?? '')
  const [language, setLanguage] = useState(template?.language ?? 'pt_BR')
  const [category, setCategory] = useState<TemplateCategory>(
    template?.category ?? 'UTILITY',
  )
  const [subCategory, setSubCategory] = useState<string>(
    template?.sub_category ?? '',
  )
  const [parameterFormat, setParameterFormat] = useState<TemplateParameterFormat>(
    template?.parameter_format ?? 'POSITIONAL',
  )
  const [ttlSeconds, setTtlSeconds] = useState<number | ''>(
    template?.message_send_ttl_seconds ?? '',
  )
  const [allowCategoryChange, setAllowCategoryChange] = useState<boolean>(
    template?.allow_category_change ?? false,
  )

  // Components are stored as an intermediate shape that mirrors the UI
  // semantics (header has a `kind`, body has its own examples, buttons
  // carry drafts). We serialize to the canonical entity shape at submit
  // time — that keeps the state manageable and the payload correct.
  const initialHeader = useMemo<HeaderDraft>(() => {
    const h = template?.components.find((c) => c.type === 'HEADER')
    if (!h) return { kind: 'none' }
    if (h.format === 'TEXT') {
      return {
        kind: 'TEXT',
        text: h.text ?? '',
        headerExamples: h.example?.header_text ?? [],
      }
    }
    if (h.format === 'IMAGE' || h.format === 'VIDEO' || h.format === 'DOCUMENT') {
      return { kind: h.format, handle: h.example?.header_handle?.[0] ?? '' }
    }
    if (h.format === 'LOCATION') return { kind: 'LOCATION' }
    return { kind: 'none' }
  }, [template])
  const [header, setHeader] = useState<HeaderDraft>(initialHeader)

  const initialBody = useMemo(() => {
    const b = template?.components.find((c) => c.type === 'BODY')
    return {
      text: b?.text ?? '',
      examples: b?.example?.body_text?.[0] ?? [],
    }
  }, [template])
  const [bodyText, setBodyText] = useState(initialBody.text)
  const [bodyExamples, setBodyExamples] = useState<string[]>(initialBody.examples)

  const [footerText, setFooterText] = useState<string>(
    template?.components.find((c) => c.type === 'FOOTER')?.text ?? '',
  )

  const initialButtons = useMemo<ButtonDraft[]>(() => {
    const row = template?.components.find((c) => c.type === 'BUTTONS')
    return (
      row?.buttons?.map((b) => ({
        type: b.type,
        text: b.text,
        url: b.url,
        phone_number: b.phone_number,
        example: b.example,
        otp_type: b.otp_type,
        flow_id: b.flow_id,
        flow_action: (b.flow_action as 'navigate' | 'data_exchange' | undefined) ?? undefined,
      })) ?? []
    )
  }, [template])
  const [buttons, setButtons] = useState<ButtonDraft[]>(initialButtons)

  // ---- Channel list --------------------------------------------------------
  const { data: channelsData } = useQuery({
    queryKey: queryKeys.channels.list(),
    queryFn: () => api.get<Channel[]>('/channels'),
    enabled: mode === 'create',
  })
  const whatsappChannels = (channelsData ?? []).filter(
    (c) => c.type === 'whatsapp_official',
  )

  // ---- Derived placeholder counts -----------------------------------------
  // Body placeholder count drives how many example inputs we show. For
  // POSITIONAL we use {{N}} max index; for NAMED we enumerate distinct
  // identifiers so the admin can pair each name with its example.
  const bodyVarCount = parameterFormat === 'NAMED'
    ? namedVars(bodyText).length
    : positionalCount(bodyText)
  const trimmedBodyExamples = useMemo(() => {
    const out = bodyExamples.slice(0, bodyVarCount)
    while (out.length < bodyVarCount) out.push('')
    return out
  }, [bodyExamples, bodyVarCount])
  const namedList = useMemo(
    () => (parameterFormat === 'NAMED' ? namedVars(bodyText) : []),
    [bodyText, parameterFormat],
  )

  const headerVarCount = header.kind === 'TEXT'
    ? (parameterFormat === 'NAMED'
      ? namedVars(header.text).length
      : positionalCount(header.text))
    : 0
  const trimmedHeaderExamples = useMemo(() => {
    if (header.kind !== 'TEXT') return []
    const arr = header.headerExamples.slice(0, headerVarCount)
    while (arr.length < headerVarCount) arr.push('')
    return arr
  }, [header, headerVarCount])

  // ---- Validation ----------------------------------------------------------
  const canSubmit = useMemo(() => {
    if (mode === 'create') {
      if (channelId === '') return false
      if (!name.trim()) return false
      if (!language.trim()) return false
    }
    if (!bodyText.trim()) return false
    if (trimmedBodyExamples.some((v) => !v.trim())) return false
    if (header.kind === 'TEXT' && trimmedHeaderExamples.some((v) => !v.trim())) {
      return false
    }
    if (
      (header.kind === 'IMAGE' || header.kind === 'VIDEO' || header.kind === 'DOCUMENT') &&
      !header.handle.trim()
    ) {
      return false
    }
    for (const b of buttons) {
      if (!b.text.trim()) return false
      if (b.type === 'URL' && !b.url?.trim()) return false
      if (b.type === 'PHONE_NUMBER' && !b.phone_number?.trim()) return false
      if (b.type === 'COPY_CODE' && !b.example?.trim()) return false
      if (b.type === 'FLOW' && (!b.flow_id?.trim() || !b.flow_action)) return false
    }
    return true
  }, [
    mode,
    channelId,
    name,
    language,
    bodyText,
    trimmedBodyExamples,
    header,
    trimmedHeaderExamples,
    buttons,
  ])

  // ---- Submit --------------------------------------------------------------
  const buildComponents = (): TemplateComponent[] => {
    const components: TemplateComponent[] = []

    if (header.kind === 'TEXT') {
      components.push({
        type: 'HEADER',
        format: 'TEXT',
        text: header.text,
        ...(headerVarCount > 0 && {
          example: { header_text: trimmedHeaderExamples },
        }),
      })
    } else if (header.kind === 'IMAGE' || header.kind === 'VIDEO' || header.kind === 'DOCUMENT') {
      components.push({
        type: 'HEADER',
        format: header.kind as TemplateHeaderFormat,
        example: { header_handle: [header.handle] },
      })
    } else if (header.kind === 'LOCATION') {
      components.push({ type: 'HEADER', format: 'LOCATION' })
    }

    components.push({
      type: 'BODY',
      text: bodyText,
      ...(trimmedBodyExamples.length > 0 && {
        example: { body_text: [trimmedBodyExamples] },
      }),
    })

    if (footerText.trim() !== '') {
      components.push({ type: 'FOOTER', text: footerText })
    }

    if (buttons.length > 0) {
      components.push({
        type: 'BUTTONS',
        buttons: buttons.map<TemplateButton>((b) => ({
          type: b.type,
          text: b.text,
          ...(b.type === 'URL' && { url: b.url }),
          ...(b.type === 'PHONE_NUMBER' && { phone_number: b.phone_number }),
          ...(b.type === 'COPY_CODE' && { example: b.example }),
          ...(b.type === 'OTP' && { otp_type: b.otp_type ?? 'COPY_CODE' }),
          ...(b.type === 'FLOW' && {
            flow_id: b.flow_id,
            flow_action: b.flow_action,
          }),
        })),
      })
    }

    return components
  }

  const createMutation = useMutation({
    mutationFn: (payload: CreateTemplateRequest) =>
      api.post<Template>('/templates', payload),
    onSuccess: (created) => {
      queryClient.invalidateQueries({ queryKey: queryKeys.templates.all })
      toast({ title: tCommon('success'), description: t('created') })
      router.push(`/templates/${created.id}`)
    },
    onError: (err: Error) => {
      toast({
        title: tCommon('error'),
        description: err.message,
        variant: 'error',
      })
    },
  })

  const editMutation = useMutation({
    mutationFn: (payload: EditTemplateRequest) =>
      api.patch<Template>(`/templates/${template!.id}`, payload),
    onSuccess: (updated) => {
      queryClient.invalidateQueries({ queryKey: queryKeys.templates.all })
      queryClient.invalidateQueries({
        queryKey: queryKeys.templates.detail(updated.id),
      })
      toast({ title: tCommon('success'), description: t('edited') })
      router.push(`/templates/${updated.id}`)
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

    const components = buildComponents()
    if (mode === 'create') {
      createMutation.mutate({
        channel_id: channelId,
        name: name.trim(),
        language: language.trim(),
        category,
        ...(category === 'UTILITY' && subCategory && { sub_category: subCategory }),
        ...(parameterFormat && { parameter_format: parameterFormat }),
        ...(typeof ttlSeconds === 'number' && ttlSeconds > 0 && {
          message_send_ttl_seconds: ttlSeconds,
        }),
        ...(allowCategoryChange && { allow_category_change: true }),
        components,
      })
    } else {
      // Edit: Meta only accepts category / components / TTL. Name, language,
      // and parameter_format are immutable — the form reflects that by
      // disabling those inputs below.
      editMutation.mutate({
        category,
        components,
        ...(typeof ttlSeconds === 'number' && ttlSeconds > 0 && {
          message_send_ttl_seconds: ttlSeconds,
        }),
      })
    }
  }

  const pending = createMutation.isPending || editMutation.isPending
  const isEdit = mode === 'edit'

  // ---- Button helpers ------------------------------------------------------
  const addButton = (type: TemplateButtonType) => {
    if (buttons.length >= 10) return
    const draft: ButtonDraft = { type, text: '' }
    if (type === 'URL') draft.url = ''
    if (type === 'PHONE_NUMBER') draft.phone_number = ''
    if (type === 'COPY_CODE') draft.example = ''
    if (type === 'OTP') draft.otp_type = 'COPY_CODE'
    if (type === 'FLOW') draft.flow_action = 'navigate'
    setButtons([...buttons, draft])
  }

  const updateButton = (i: number, patch: Partial<ButtonDraft>) => {
    setButtons(buttons.map((b, idx) => (idx === i ? { ...b, ...patch } : b)))
  }

  const removeButton = (i: number) => {
    setButtons(buttons.filter((_, idx) => idx !== i))
  }

  return (
    <form onSubmit={handleSubmit} className="flex-1 overflow-auto">
      <div className="mx-auto max-w-3xl space-y-6 p-6">
        {mode === 'create' && whatsappChannels.length === 0 && (
          <Alert>
            <Info className="h-4 w-4" />
            <AlertDescription>{t('empty.noChannel')}</AlertDescription>
          </Alert>
        )}

        {/* ================= Identity ================= */}
        <Card>
          <CardHeader>
            <CardTitle>{t('form.basics')}</CardTitle>
          </CardHeader>
          <CardContent className="space-y-4">
            {mode === 'create' && (
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
            )}

            <div className="grid grid-cols-1 gap-4 md:grid-cols-2">
              <div className="space-y-2">
                <Label htmlFor="name">{t('form.name')}</Label>
                <Input
                  id="name"
                  value={name}
                  onChange={(e) =>
                    setName(e.target.value.toLowerCase().replace(/\s+/g, '_'))
                  }
                  placeholder="order_confirmation"
                  pattern="[a-z0-9_]+"
                  required={mode === 'create'}
                  disabled={isEdit}
                />
                <p className="text-xs text-muted-foreground">
                  {isEdit ? t('form.nameImmutable') : t('form.nameHint')}
                </p>
              </div>
              <div className="space-y-2">
                <Label htmlFor="language">{t('form.language')}</Label>
                <Input
                  id="language"
                  value={language}
                  onChange={(e) => setLanguage(e.target.value)}
                  placeholder="pt_BR"
                  required={mode === 'create'}
                  disabled={isEdit}
                />
                <p className="text-xs text-muted-foreground">
                  {isEdit ? t('form.languageImmutable') : t('form.languageHint')}
                </p>
              </div>
            </div>

            <div className="grid grid-cols-1 gap-4 md:grid-cols-2">
              <div className="space-y-2">
                <Label htmlFor="category">{t('form.category')}</Label>
                <Select
                  value={category}
                  onValueChange={(v) => setCategory(v as TemplateCategory)}
                >
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

              {category === 'UTILITY' && (
                <div className="space-y-2">
                  <Label htmlFor="sub_category">{t('form.subCategory')}</Label>
                  <Select value={subCategory || 'none'} onValueChange={(v) => setSubCategory(v === 'none' ? '' : v)}>
                    <SelectTrigger id="sub_category">
                      <SelectValue placeholder={t('form.subCategoryPlaceholder')} />
                    </SelectTrigger>
                    <SelectContent>
                      <SelectItem value="none">{t('form.subCategoryNone')}</SelectItem>
                      <SelectItem value="ORDER_DETAILS">ORDER_DETAILS</SelectItem>
                      <SelectItem value="ORDER_STATUS">ORDER_STATUS</SelectItem>
                      <SelectItem value="RICH_ORDER_STATUS">RICH_ORDER_STATUS</SelectItem>
                    </SelectContent>
                  </Select>
                </div>
              )}
            </div>

            {!isEdit && (
              <div className="space-y-2">
                <Label htmlFor="parameter_format">{t('form.parameterFormat')}</Label>
                <Select
                  value={parameterFormat}
                  onValueChange={(v) => setParameterFormat(v as TemplateParameterFormat)}
                >
                  <SelectTrigger id="parameter_format">
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectItem value="POSITIONAL">
                      {t('form.parameterFormatPositional')}
                    </SelectItem>
                    <SelectItem value="NAMED">
                      {t('form.parameterFormatNamed')}
                    </SelectItem>
                  </SelectContent>
                </Select>
                <p className="text-xs text-muted-foreground">
                  {parameterFormat === 'NAMED'
                    ? t('form.parameterFormatNamedHint')
                    : t('form.parameterFormatPositionalHint')}
                </p>
              </div>
            )}

            <div className="grid grid-cols-1 gap-4 md:grid-cols-2">
              <div className="space-y-2">
                <Label htmlFor="ttl">{t('form.ttl')}</Label>
                <Input
                  id="ttl"
                  type="number"
                  min={0}
                  value={ttlSeconds === '' ? '' : ttlSeconds}
                  onChange={(e) => {
                    const v = e.target.value
                    setTtlSeconds(v === '' ? '' : Math.max(0, parseInt(v, 10) || 0))
                  }}
                  placeholder="0"
                />
                <p className="text-xs text-muted-foreground">{t('form.ttlHint')}</p>
              </div>

              {!isEdit && (
                <div className="flex items-center justify-between space-y-2">
                  <div>
                    <Label htmlFor="allow_category_change">{t('form.allowCategoryChange')}</Label>
                    <p className="text-xs text-muted-foreground">
                      {t('form.allowCategoryChangeHint')}
                    </p>
                  </div>
                  <Switch
                    id="allow_category_change"
                    checked={allowCategoryChange}
                    onCheckedChange={setAllowCategoryChange}
                  />
                </div>
              )}
            </div>
          </CardContent>
        </Card>

        {/* ================= Header ================= */}
        <Card>
          <CardHeader>
            <CardTitle>{t('form.header')}</CardTitle>
          </CardHeader>
          <CardContent className="space-y-4">
            <div className="space-y-2">
              <Label htmlFor="header_format">{t('form.headerFormat')}</Label>
              <Select
                value={header.kind}
                onValueChange={(v) => {
                  if (v === 'none') setHeader({ kind: 'none' })
                  else if (v === 'TEXT') setHeader({ kind: 'TEXT', text: '', headerExamples: [] })
                  else if (v === 'LOCATION') setHeader({ kind: 'LOCATION' })
                  else setHeader({ kind: v as 'IMAGE' | 'VIDEO' | 'DOCUMENT', handle: '' })
                }}
              >
                <SelectTrigger id="header_format">
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="none">{t('form.headerNone')}</SelectItem>
                  <SelectItem value="TEXT">{t('form.headerText')}</SelectItem>
                  <SelectItem value="IMAGE">{t('form.headerImage')}</SelectItem>
                  <SelectItem value="VIDEO">{t('form.headerVideo')}</SelectItem>
                  <SelectItem value="DOCUMENT">{t('form.headerDocument')}</SelectItem>
                  <SelectItem value="LOCATION">{t('form.headerLocation')}</SelectItem>
                </SelectContent>
              </Select>
            </div>

            {header.kind === 'TEXT' && (
              <>
                <div className="space-y-2">
                  <Label htmlFor="header_text">{t('form.headerTextLabel')}</Label>
                  <Input
                    id="header_text"
                    value={header.text}
                    onChange={(e) => setHeader({ ...header, text: e.target.value })}
                    placeholder={t('form.headerTextPlaceholder')}
                    maxLength={60}
                  />
                </div>
                {headerVarCount > 0 && (
                  <div className="space-y-2 rounded-md border border-dashed p-3">
                    <Label>
                      {parameterFormat === 'NAMED'
                        ? t('form.headerExamplesNamed')
                        : t('form.headerExamples', { count: headerVarCount })}
                    </Label>
                    {trimmedHeaderExamples.map((val, i) => (
                      <div key={i} className="flex items-center gap-2">
                        <span className="w-24 shrink-0 font-mono text-sm text-muted-foreground">
                          {parameterFormat === 'NAMED'
                            ? `{{${namedVars(header.text)[i] ?? i + 1}}}`
                            : `{{${i + 1}}}`}
                        </span>
                        <Input
                          value={val}
                          onChange={(e) => {
                            const next = [...trimmedHeaderExamples]
                            next[i] = e.target.value
                            setHeader({ ...header, headerExamples: next })
                          }}
                        />
                      </div>
                    ))}
                  </div>
                )}
              </>
            )}

            {(header.kind === 'IMAGE' || header.kind === 'VIDEO' || header.kind === 'DOCUMENT') && (
              <div className="space-y-2">
                <Label htmlFor="handle">{t('form.headerHandle')}</Label>
                <Input
                  id="handle"
                  value={header.handle}
                  onChange={(e) => setHeader({ ...header, handle: e.target.value })}
                  placeholder="4:AAAa..."
                />
                <p className="text-xs text-muted-foreground">{t('form.headerHandleHint')}</p>
              </div>
            )}

            {header.kind === 'LOCATION' && (
              <Alert>
                <Info className="h-4 w-4" />
                <AlertDescription>{t('form.headerLocationHint')}</AlertDescription>
              </Alert>
            )}
          </CardContent>
        </Card>

        {/* ================= Body ================= */}
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
                placeholder={
                  parameterFormat === 'NAMED'
                    ? t('form.bodyPlaceholderNamed')
                    : t('form.bodyPlaceholder')
                }
                required
              />
              <p className="text-xs text-muted-foreground">
                {parameterFormat === 'NAMED'
                  ? t('form.bodyHintNamed')
                  : t('form.bodyHint')}
              </p>
            </div>

            {bodyVarCount > 0 && (
              <div className="space-y-2 rounded-md border border-dashed p-3">
                <Label>
                  {parameterFormat === 'NAMED'
                    ? t('form.bodyExamplesNamed')
                    : t('form.examples', { count: bodyVarCount })}
                </Label>
                <p className="text-xs text-muted-foreground">
                  {t('form.examplesHint')}
                </p>
                {trimmedBodyExamples.map((val, i) => (
                  <div key={i} className="flex items-center gap-2">
                    <span className="w-24 shrink-0 font-mono text-sm text-muted-foreground">
                      {parameterFormat === 'NAMED'
                        ? `{{${namedList[i] ?? i + 1}}}`
                        : `{{${i + 1}}}`}
                    </span>
                    <Input
                      value={val}
                      onChange={(e) => {
                        const next = [...trimmedBodyExamples]
                        next[i] = e.target.value
                        setBodyExamples(next)
                      }}
                    />
                  </div>
                ))}
              </div>
            )}

            <div className="space-y-2">
              <Label htmlFor="footer">{t('form.footer')}</Label>
              <Input
                id="footer"
                value={footerText}
                onChange={(e) => setFooterText(e.target.value)}
                placeholder={t('form.footerPlaceholder')}
                maxLength={60}
              />
              <p className="text-xs text-muted-foreground">{t('form.footerHint')}</p>
            </div>
          </CardContent>
        </Card>

        {/* ================= Buttons ================= */}
        <Card>
          <CardHeader>
            <CardTitle className="flex flex-wrap items-center justify-between gap-2">
              {t('form.buttons')}
              <ButtonTypeMenu onAdd={addButton} disabled={buttons.length >= 10} />
            </CardTitle>
          </CardHeader>
          <CardContent className="space-y-3">
            {buttons.length === 0 ? (
              <p className="text-sm text-muted-foreground">{t('form.noButtons')}</p>
            ) : (
              buttons.map((btn, i) => (
                <ButtonEditor
                  key={i}
                  button={btn}
                  onChange={(patch) => updateButton(i, patch)}
                  onRemove={() => removeButton(i)}
                />
              ))
            )}
          </CardContent>
        </Card>

        <div className="flex justify-end gap-2">
          <Link href={isEdit ? `/templates/${template!.id}` : '/templates'}>
            <Button type="button" variant="outline">
              {tCommon('cancel')}
            </Button>
          </Link>
          <Button type="submit" disabled={!canSubmit || pending}>
            {pending ? tCommon('loading') : isEdit ? t('form.save') : t('form.submit')}
          </Button>
        </div>
      </div>
    </form>
  )
}

// -----------------------------------------------------------------------------
// ButtonTypeMenu — the "add button" dropdown
// -----------------------------------------------------------------------------
function ButtonTypeMenu({
  onAdd,
  disabled,
}: {
  onAdd: (type: TemplateButtonType) => void
  disabled: boolean
}) {
  const t = useTranslations('templates.form')

  return (
    <div className="flex flex-wrap gap-2">
      {(['QUICK_REPLY', 'URL', 'PHONE_NUMBER', 'COPY_CODE', 'OTP', 'FLOW'] as const).map((type) => (
        <Button
          key={type}
          type="button"
          variant="outline"
          size="sm"
          onClick={() => onAdd(type)}
          disabled={disabled}
        >
          <Plus className="mr-1 h-3 w-3" />
          {t(`button.${type}`)}
        </Button>
      ))}
    </div>
  )
}

// -----------------------------------------------------------------------------
// ButtonEditor — inline editor for each button type
// -----------------------------------------------------------------------------
function ButtonEditor({
  button,
  onChange,
  onRemove,
}: {
  button: ButtonDraft
  onChange: (patch: Partial<ButtonDraft>) => void
  onRemove: () => void
}) {
  const t = useTranslations('templates.form')

  return (
    <div className="flex items-start gap-2 rounded-md border p-3">
      <div className="flex-1 space-y-2">
        <div className="flex items-center gap-2">
          <span className="rounded bg-muted px-2 py-0.5 font-mono text-xs">
            {button.type}
          </span>
          <Input
            value={button.text}
            onChange={(e) => onChange({ text: e.target.value })}
            placeholder={t('buttonTextPlaceholder')}
            maxLength={25}
          />
        </div>

        {button.type === 'URL' && (
          <Input
            value={button.url ?? ''}
            onChange={(e) => onChange({ url: e.target.value })}
            placeholder="https://example.com/path/{{1}}"
            type="url"
          />
        )}

        {button.type === 'PHONE_NUMBER' && (
          <Input
            value={button.phone_number ?? ''}
            onChange={(e) => onChange({ phone_number: e.target.value })}
            placeholder="+5511999999999"
            type="tel"
          />
        )}

        {button.type === 'COPY_CODE' && (
          <Input
            value={button.example ?? ''}
            onChange={(e) => onChange({ example: e.target.value })}
            placeholder={t('copyCodeExamplePlaceholder')}
          />
        )}

        {button.type === 'OTP' && (
          <div className="grid grid-cols-1 gap-2 md:grid-cols-2">
            <div className="space-y-1">
              <Label>{t('otpType')}</Label>
              <Select
                value={button.otp_type ?? 'COPY_CODE'}
                onValueChange={(v) => onChange({ otp_type: v })}
              >
                <SelectTrigger>
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="COPY_CODE">COPY_CODE</SelectItem>
                  <SelectItem value="ONE_TAP" disabled>
                    ONE_TAP ({t('phase3')})
                  </SelectItem>
                  <SelectItem value="ZERO_TAP" disabled>
                    ZERO_TAP ({t('phase3')})
                  </SelectItem>
                </SelectContent>
              </Select>
            </div>
          </div>
        )}

        {button.type === 'FLOW' && (
          <div className="grid grid-cols-1 gap-2 md:grid-cols-2">
            <Input
              value={button.flow_id ?? ''}
              onChange={(e) => onChange({ flow_id: e.target.value })}
              placeholder={t('flowIdPlaceholder')}
            />
            <Select
              value={button.flow_action ?? 'navigate'}
              onValueChange={(v) =>
                onChange({ flow_action: v as 'navigate' | 'data_exchange' })
              }
            >
              <SelectTrigger>
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="navigate">navigate</SelectItem>
                <SelectItem value="data_exchange">data_exchange</SelectItem>
              </SelectContent>
            </Select>
          </div>
        )}
      </div>

      <Button type="button" variant="ghost" size="icon" onClick={onRemove}>
        <Trash2 className="h-4 w-4" />
      </Button>
    </div>
  )
}
