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
  MessageSquare,
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
import { api } from '@/lib/api'
import type { Channel } from '@/types'

const createSchema = (t: (key: string) => string) => z.object({
  name: z.string().min(1, t('channelNameRequired')),
  // Appearance
  primary_color: z.string(),
  text_color: z.string(),
  position: z.enum(['bottom-right', 'bottom-left']),
  // Messages
  welcome_message: z.string().optional(),
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
    resolver: zodResolver(createSchema(tCommon)),
    defaultValues: {
      name: channel?.name || '',
      primary_color: (channel?.config?.primary_color as string) || '#6366f1',
      text_color: (channel?.config?.text_color as string) || '#ffffff',
      position: (channel?.config?.position as 'bottom-right' | 'bottom-left') || 'bottom-right',
      welcome_message: (channel?.config?.welcome_message as string) || '',
      placeholder_text: (channel?.config?.placeholder_text as string) || 'Type a message...',
      auto_open: (channel?.config?.auto_open as boolean) || false,
      auto_open_delay: (channel?.config?.auto_open_delay as number) || 3,
      show_typing_indicator: (channel?.config?.show_typing_indicator as boolean) ?? true,
      allowed_domains: (channel?.config?.allowed_domains as string) || '',
    },
  })

  const primaryColor = watch('primary_color')
  const position = watch('position')
  const autoOpen = watch('auto_open')

  const onSubmit = async (data: WebchatConfigForm) => {
    setIsSubmitting(true)

    try {
      const payload = {
        name: data.name,
        type: 'webchat',
        config: {
          primary_color: data.primary_color,
          text_color: data.text_color,
          position: data.position,
          welcome_message: data.welcome_message,
          placeholder_text: data.placeholder_text,
          auto_open: data.auto_open,
          auto_open_delay: data.auto_open_delay,
          show_typing_indicator: data.show_typing_indicator,
          allowed_domains: data.allowed_domains,
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

  const copyEmbedCode = () => {
    const channelId = channel?.id || '{CHANNEL_ID}'
    const embedCode = `<script>
  (function(w, d, s, o, f, js, fjs) {
    w['LinktorWidget'] = o;
    w[o] = w[o] || function() { (w[o].q = w[o].q || []).push(arguments) };
    js = d.createElement(s); fjs = d.getElementsByTagName(s)[0];
    js.id = o; js.src = f; js.async = 1; fjs.parentNode.insertBefore(js, fjs);
  }(window, document, 'script', 'linktor', '${typeof window !== 'undefined' ? window.location.origin : ''}/widget/linktor.js'));
  linktor('init', { channelId: '${channelId}' });
</script>`
    navigator.clipboard.writeText(embedCode)
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
              <div className="relative h-64 bg-muted rounded-lg overflow-hidden">
                <div
                  className={`absolute bottom-4 ${position === 'bottom-right' ? 'right-4' : 'left-4'}`}
                >
                  <div
                    className="w-14 h-14 rounded-full flex items-center justify-center shadow-lg cursor-pointer"
                    style={{ backgroundColor: primaryColor }}
                  >
                    <MessageSquare className="h-6 w-6 text-white" />
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
                  <code>{`<script>
  (function(w, d, s, o, f, js, fjs) {
    w['LinktorWidget'] = o;
    w[o] = w[o] || function() { (w[o].q = w[o].q || []).push(arguments) };
    js = d.createElement(s); fjs = d.getElementsByTagName(s)[0];
    js.id = o; js.src = f; js.async = 1; fjs.parentNode.insertBefore(js, fjs);
  }(window, document, 'script', 'linktor', '/widget/linktor.js'));
  linktor('init', { channelId: '${channel?.id || '{CHANNEL_ID}'}' });
</script>`}</code>
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
