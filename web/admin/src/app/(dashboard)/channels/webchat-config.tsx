'use client'

import { useState } from 'react'
import { useForm } from 'react-hook-form'
import { useTranslations } from 'next-intl'
import { zodResolver } from '@hookform/resolvers/zod'
import { z } from 'zod'
import {
  Copy,
  Check,
  Loader2,
  Code,
  Palette,
  X,
  Send,
} from 'lucide-react'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Textarea } from '@/components/ui/textarea'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Switch } from '@/components/ui/switch'
import { useToast } from '@/hooks/use-toast'
import { api, WEBHOOK_BASE_URL } from '@/lib/api'
import { copyText } from '@/lib/clipboard'
import type { Channel } from '@/types'

const webchatConfigSchema = z.object({
  name: z.string().min(1, 'Channel name is required'),
  // Appearance
  widget_title: z.string().optional(),
  primary_color: z.string(),
  text_color: z.string(),
  position: z.enum(['bottom-right', 'bottom-left']),
  // Messages
  welcome_message: z.string().optional(),
  offline_message: z.string().optional(),
  placeholder_text: z.string().optional(),
  // Behavior
  auto_open: z.boolean(),
  auto_open_delay: z.coerce.number().min(0),
  show_typing_indicator: z.boolean(),
  // Allowed domains
  allowed_domains: z.string().optional(),
})

type WebchatConfigForm = z.infer<typeof webchatConfigSchema>

interface WebchatConfigProps {
  channel?: Channel
  onSuccess?: (channel: Channel) => void
  onCancel?: () => void
}

const toBoolean = (value: unknown, fallback: boolean): boolean => {
  if (typeof value === 'boolean') return value
  if (typeof value === 'string') {
    const normalized = value.trim().toLowerCase()
    if (normalized === 'true') return true
    if (normalized === 'false') return false
  }
  return fallback
}

const toNumber = (value: unknown, fallback: number): number => {
  if (typeof value === 'number' && Number.isFinite(value)) return value
  if (typeof value === 'string') {
    const parsed = Number(value)
    if (Number.isFinite(parsed)) return parsed
  }
  return fallback
}

export function WebchatConfig({ channel, onSuccess, onCancel }: WebchatConfigProps) {
  const t = useTranslations('channels.config')
  const tCommon = useTranslations('common')
  const { toast } = useToast()
  const [isSubmitting, setIsSubmitting] = useState(false)
  const [copied, setCopied] = useState(false)

  const isEditing = !!channel

  const {
    register,
    handleSubmit,
    watch,
    setValue,
    formState: { errors },
  } = useForm<WebchatConfigForm>({
    resolver: zodResolver(webchatConfigSchema),
    defaultValues: {
      name: channel?.name || '',
      widget_title: (channel?.config?.widget_title as string) || '',
      // Read the canonical `widget_color` (what the backend/widget consume),
      // falling back to the legacy `primary_color` key for older channels.
      primary_color:
        (channel?.config?.widget_color as string) ||
        (channel?.config?.primary_color as string) ||
        '#6366f1',
      text_color: (channel?.config?.text_color as string) || '#ffffff',
      position: (channel?.config?.position as 'bottom-right' | 'bottom-left') || 'bottom-right',
      welcome_message: (channel?.config?.welcome_message as string) || '',
      offline_message: (channel?.config?.offline_message as string) || '',
      placeholder_text: (channel?.config?.placeholder_text as string) || 'Type a message...',
      auto_open: toBoolean(channel?.config?.auto_open, false),
      auto_open_delay: toNumber(channel?.config?.auto_open_delay, 3),
      show_typing_indicator: toBoolean(channel?.config?.show_typing_indicator, true),
      allowed_domains: (channel?.config?.allowed_domains as string) || '',
    },
  })

  const primaryColor = watch('primary_color')
  const position = watch('position')
  const autoOpen = watch('auto_open')
  const widgetTitle = watch('widget_title')
  const placeholderText = watch('placeholder_text')
  const textColor = watch('text_color')
  const welcomeMessage = watch('welcome_message')

  const onSubmit = async (data: WebchatConfigForm) => {
    setIsSubmitting(true)

    try {
      const payload = {
        name: data.name,
        type: 'webchat',
        config: {
          // Canonical keys consumed by the backend adapter (ParseConfig) and
          // delivered to the widget via GET /webchat/:id/config.
          widget_title: data.widget_title || '',
          widget_color: data.primary_color,
          welcome_message: data.welcome_message || '',
          offline_message: data.offline_message || '',
          // Client-side widget options, carried into the embed snippet.
          text_color: data.text_color,
          position: data.position,
          placeholder_text: data.placeholder_text || '',
          auto_open: String(data.auto_open),
          auto_open_delay: String(data.auto_open_delay),
          show_typing_indicator: String(data.show_typing_indicator),
          allowed_domains: data.allowed_domains || '',
        },
      }

      let result: Channel
      if (isEditing) {
        result = await api.put<Channel>(`/channels/${channel.id}`, payload)
        toast({
          title: t('channelUpdated'),
          description: t('channelUpdatedDesc', { name: 'WebChat' }),
        })
      } else {
        result = await api.post<Channel>('/channels', payload)
        toast({
          title: t('channelCreated'),
          description: t('channelCreatedDesc', { name: 'WebChat' }),
        })
      }

      onSuccess?.(result)
    } catch (error: unknown) {
      const message = error instanceof Error ? error.message : t('failedToSave')
      toast({
        title: t('error'),
        description: message,
        variant: 'error',
      })
    } finally {
      setIsSubmitting(false)
    }
  }

  // Canonical widget embed snippet. The loader is served from the API origin at
  // /widget/v1/linktor.js and both the script src and baseUrl point there — the
  // widget appends /api/v1/... and /ws/... itself, so baseUrl carries no suffix.
  // The init options mirror the configured appearance/behavior so the snippet is
  // self-contained (JSON.stringify keeps values safely quoted).
  const buildEmbedCode = (channelId: string) => {
    const opts: string[] = [
      `channelId: ${JSON.stringify(channelId)}`,
      `baseUrl: ${JSON.stringify(WEBHOOK_BASE_URL)}`,
      `position: ${JSON.stringify(position === 'bottom-left' ? 'left' : 'right')}`,
    ]
    if (autoOpen) opts.push('autoOpen: true')
    if (widgetTitle) opts.push(`title: ${JSON.stringify(widgetTitle)}`)
    if (primaryColor) opts.push(`primaryColor: ${JSON.stringify(primaryColor)}`)
    if (placeholderText) opts.push(`labels: { inputPlaceholder: ${JSON.stringify(placeholderText)} }`)
    return `<script>
  (function(l,i,n,k,t,o,r){l['LinktorObject']=t;l[t]=l[t]||function(){
    (l[t].q=l[t].q||[]).push(arguments)};o=i.createElement(n);
    r=i.getElementsByTagName(n)[0];o.async=1;o.src=k;r.parentNode.insertBefore(o,r);
  })(window,document,'script','${WEBHOOK_BASE_URL}/widget/v1/linktor.js','linktor');
  linktor('init', { ${opts.join(', ')} });
</script>`
  }

  const copyEmbedCode = () => {
    void copyText(buildEmbedCode(channel?.id || '{CHANNEL_ID}'))
    setCopied(true)
    setTimeout(() => setCopied(false), 2000)
  }

  return (
    <form onSubmit={handleSubmit(onSubmit)} className="flex flex-col h-full">
      <div className="flex-1 space-y-6">
        <div className="space-y-4">
          <div className="space-y-2">
            <Label htmlFor="name">{t('channelName')}</Label>
            <Input
              id="name"
              placeholder={t('myWebsiteChat')}
              {...register('name')}
            />
            {errors.name && (
              <p className="text-sm text-destructive">{errors.name.message}</p>
            )}
          </div>
        </div>

        <Tabs defaultValue="appearance" className="w-full">
        <TabsList className="grid w-full grid-cols-3">
          <TabsTrigger value="appearance">{t('appearance')}</TabsTrigger>
          <TabsTrigger value="behavior">{t('behavior')}</TabsTrigger>
          <TabsTrigger value="embed">{t('embedCode')}</TabsTrigger>
        </TabsList>

        <TabsContent value="appearance" className="space-y-4 pt-4">
          <Card>
            <CardHeader>
              <CardTitle className="text-base flex items-center gap-2">
                <Palette className="h-4 w-4" />
                {t('widgetAppearance')}
              </CardTitle>
              <CardDescription>{t('customizeWidget')}</CardDescription>
            </CardHeader>
            <CardContent className="space-y-4">
              <div className="space-y-2">
                <Label htmlFor="widget_title">{t('widgetTitle')}</Label>
                <Input
                  id="widget_title"
                  placeholder={t('widgetTitlePlaceholder')}
                  {...register('widget_title')}
                />
              </div>

              <div className="grid grid-cols-2 gap-4">
                <div className="space-y-2">
                  <Label htmlFor="primary_color">{t('primaryColor')}</Label>
                  <div className="flex gap-2">
                    <Input
                      id="primary_color"
                      type="color"
                      className="w-12 h-10 p-1 cursor-pointer"
                      {...register('primary_color')}
                    />
                    <Input
                      value={primaryColor}
                      onChange={(e) => setValue('primary_color', e.target.value)}
                      placeholder="#6366f1"
                      className="flex-1"
                    />
                  </div>
                </div>
                <div className="space-y-2">
                  <Label htmlFor="text_color">{t('textColor')}</Label>
                  <div className="flex gap-2">
                    <Input
                      id="text_color"
                      type="color"
                      className="w-12 h-10 p-1 cursor-pointer"
                      {...register('text_color')}
                    />
                    <Input
                      {...register('text_color')}
                      placeholder="#ffffff"
                      className="flex-1"
                    />
                  </div>
                </div>
              </div>

              <div className="space-y-2">
                <Label>{t('widgetPosition')}</Label>
                <Select
                  value={position}
                  onValueChange={(value: 'bottom-right' | 'bottom-left') => setValue('position', value)}
                >
                  <SelectTrigger>
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectItem value="bottom-right">{t('bottomRight')}</SelectItem>
                    <SelectItem value="bottom-left">{t('bottomLeft')}</SelectItem>
                  </SelectContent>
                </Select>
              </div>

              <div className="space-y-2">
                <Label htmlFor="welcome_message">{t('welcomeMessage')}</Label>
                <Textarea
                  id="welcome_message"
                  placeholder={t('welcomeMessagePlaceholder')}
                  {...register('welcome_message')}
                />
              </div>

              <div className="space-y-2">
                <Label htmlFor="offline_message">{t('offlineMessage')}</Label>
                <Textarea
                  id="offline_message"
                  placeholder={t('offlineMessagePlaceholder')}
                  {...register('offline_message')}
                />
              </div>

              <div className="space-y-2">
                <Label htmlFor="placeholder_text">{t('inputPlaceholder')}</Label>
                <Input
                  id="placeholder_text"
                  placeholder={t('inputPlaceholderDefault')}
                  {...register('placeholder_text')}
                />
              </div>
            </CardContent>
          </Card>

          {/* Preview */}
          <Card>
            <CardHeader>
              <CardTitle className="text-base">{t('preview')}</CardTitle>
            </CardHeader>
            <CardContent>
              <div className="relative h-80 bg-muted rounded-lg overflow-hidden">
                {/* Live widget preview — reflects the configured color, title,
                    welcome message and input placeholder. */}
                <div
                  className={`absolute bottom-4 flex w-[280px] max-w-[calc(100%-2rem)] flex-col overflow-hidden rounded-xl bg-white shadow-xl ${
                    position === 'bottom-right' ? 'right-4' : 'left-4'
                  }`}
                  style={{ height: '272px' }}
                >
                  <div
                    className="flex items-center justify-between px-4 py-3"
                    style={{ backgroundColor: primaryColor, color: textColor }}
                  >
                    <div>
                      <div className="text-sm font-semibold leading-tight">
                        {widgetTitle || 'Chat'}
                      </div>
                      <div className="text-[11px] opacity-90">Online</div>
                    </div>
                    <X className="h-4 w-4 shrink-0" />
                  </div>

                  <div className="flex-1 space-y-2 overflow-hidden bg-white p-3">
                    <div className="max-w-[85%] rounded-2xl rounded-bl-sm bg-muted px-3 py-2 text-xs text-foreground">
                      {welcomeMessage || t('welcomeMessagePlaceholder')}
                    </div>
                  </div>

                  <div className="flex items-center gap-2 border-t p-2">
                    <div className="flex-1 truncate rounded-full border px-3 py-1.5 text-xs text-muted-foreground">
                      {placeholderText || t('inputPlaceholderDefault')}
                    </div>
                    <div
                      className="flex h-8 w-8 shrink-0 items-center justify-center rounded-full"
                      style={{ backgroundColor: primaryColor }}
                    >
                      <Send className="h-3.5 w-3.5 text-white" />
                    </div>
                  </div>
                </div>
              </div>
            </CardContent>
          </Card>
        </TabsContent>

        <TabsContent value="behavior" className="space-y-4 pt-4">
          <Card>
            <CardHeader>
              <CardTitle className="text-base">{t('widgetBehavior')}</CardTitle>
              <CardDescription>{t('configureBehavior')}</CardDescription>
            </CardHeader>
            <CardContent className="space-y-4">
              <div className="flex items-center justify-between">
                <div>
                  <Label>{t('autoOpenWidget')}</Label>
                  <p className="text-sm text-muted-foreground">
                    {t('autoOpenDesc')}
                  </p>
                </div>
                <Switch
                  checked={autoOpen}
                  onCheckedChange={(checked) => setValue('auto_open', checked)}
                />
              </div>

              {autoOpen && (
                <div className="space-y-2 pl-4 border-l-2 border-primary/20">
                  <Label htmlFor="auto_open_delay">{t('delaySeconds')}</Label>
                  <Input
                    id="auto_open_delay"
                    type="number"
                    min={0}
                    {...register('auto_open_delay')}
                  />
                </div>
              )}

              <div className="flex items-center justify-between">
                <div>
                  <Label>{t('showTypingIndicator')}</Label>
                  <p className="text-sm text-muted-foreground">
                    {t('typingIndicatorDesc')}
                  </p>
                </div>
                <Switch
                  checked={watch('show_typing_indicator')}
                  onCheckedChange={(checked) => setValue('show_typing_indicator', checked)}
                />
              </div>

              <div className="space-y-2">
                <Label htmlFor="allowed_domains">{t('allowedDomains')}</Label>
                <Textarea
                  id="allowed_domains"
                  placeholder={t('allowedDomainsPlaceholder')}
                  {...register('allowed_domains')}
                />
                <p className="text-xs text-muted-foreground">
                  {t('allowedDomainsDesc')}
                </p>
              </div>
            </CardContent>
          </Card>
        </TabsContent>

        <TabsContent value="embed" className="space-y-4 pt-4">
          <Card>
            <CardHeader>
              <CardTitle className="text-base flex items-center gap-2">
                <Code className="h-4 w-4" />
                {t('embedCodeTitle')}
              </CardTitle>
              <CardDescription>
                {t('embedCodeDesc')}
              </CardDescription>
            </CardHeader>
            <CardContent className="space-y-4">
              <div className="relative">
                <pre className="bg-muted p-4 rounded-lg text-xs overflow-x-auto">
                  <code>{buildEmbedCode(channel?.id || '{CHANNEL_ID}')}</code>
                </pre>
                <Button
                  type="button"
                  variant="outline"
                  size="sm"
                  className="absolute top-2 right-2"
                  onClick={copyEmbedCode}
                >
                  {copied ? <Check className="h-4 w-4" /> : <Copy className="h-4 w-4" />}
                </Button>
              </div>

              <p className="text-sm text-muted-foreground">
                {t('embedCodePlacement', { tag: '</body>' })}
              </p>

              <div className="rounded-lg border bg-muted/40 p-4 space-y-2">
                <p className="text-sm font-medium">{t('embedFrameworks')}</p>
                <p className="text-sm text-muted-foreground">{t('embedFrameworksDesc')}</p>
                <div className="flex flex-wrap gap-2 pt-1">
                  <code className="text-xs bg-background rounded px-1.5 py-0.5 border">@linktor/react-webchat</code>
                  <code className="text-xs bg-background rounded px-1.5 py-0.5 border">@linktor/vue-webchat</code>
                  <code className="text-xs bg-background rounded px-1.5 py-0.5 border">@linktor/angular-webchat</code>
                </div>
                <a
                  href="https://github.com/edsonmartins/linktor-webchat-sdk/tree/main/examples"
                  target="_blank"
                  rel="noopener noreferrer"
                  className="inline-flex items-center gap-1.5 text-sm text-primary hover:underline pt-1"
                >
                  <Code className="h-3.5 w-3.5" />
                  {t('embedViewExamples')}
                </a>
              </div>
            </CardContent>
          </Card>
        </TabsContent>
        </Tabs>
      </div>

      <div className="sticky bottom-0 flex justify-end gap-3 pt-4 pb-2 mt-4 border-t bg-background">
        {onCancel && (
          <Button type="button" variant="outline" onClick={onCancel}>
            {t('cancel')}
          </Button>
        )}
        <Button type="submit" disabled={isSubmitting}>
          {isSubmitting && <Loader2 className="mr-2 h-4 w-4 animate-spin" />}
          {isEditing ? t('updateChannel') : t('createChannel')}
        </Button>
      </div>
    </form>
  )
}
