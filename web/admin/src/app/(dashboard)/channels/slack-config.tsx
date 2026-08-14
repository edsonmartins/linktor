'use client'

import { useState } from 'react'
import { useForm } from 'react-hook-form'
import { useTranslations } from 'next-intl'
import { zodResolver } from '@hookform/resolvers/zod'
import { z } from 'zod'
import {
  Copy,
  Eye,
  EyeOff,
  KeyRound,
  Loader2,
  Send,
  Webhook,
} from 'lucide-react'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from '@/components/ui/card'
import {
  Form,
  FormControl,
  FormDescription,
  FormField,
  FormItem,
  FormLabel,
  FormMessage,
} from '@/components/ui/form'
import { Badge } from '@/components/ui/badge'
import { useToast } from '@/hooks/use-toast'
import { api, WEBHOOK_BASE_URL } from '@/lib/api'
import { copyText } from '@/lib/clipboard'
import type { Channel } from '@/types'

/**
 * Slack Configuration Schema
 */
/**
 * Segredos são exigidos apenas na CRIAÇÃO.
 *
 * A API nunca devolve credencial guardada, então ao editar esses campos abrem
 * vazios e a tela mostra "••••••••", prometendo que o valor atual será mantido.
 * Exigi-los no schema quebrava a promessa: era impossível salvar qualquer
 * alteração — até renomear o canal — sem redigitar o segredo. Em branco, o
 * backend mantém o guardado (service/channel.go: `if v == "" { continue }`).
 */
const slackConfigSchema = (isEditing = false) => z.object({
  name: z.string().min(1, 'Channel name is required'),
  bot_token: z.string().optional(),
  signing_secret: z.string().optional(),
  app_id: z.string().optional(),
  bot_user_id: z.string().optional(),
  webhook_url: z.string().url('Must be a valid URL').or(z.literal('')).optional(),
  webhook_secret: z.string().optional(),
}).superRefine((data, ctx) => {
  if (isEditing) return // em branco = manter o segredo guardado
  if (!data.bot_token) {
    ctx.addIssue({ code: z.ZodIssueCode.custom, message: 'Bot token is required', path: ['bot_token'] })
  }
  if (!data.signing_secret) {
    ctx.addIssue({ code: z.ZodIssueCode.custom, message: 'Signing secret is required', path: ['signing_secret'] })
  }
})

type SlackConfigForm = z.infer<ReturnType<typeof slackConfigSchema>>

interface SlackConfigProps {
  channel?: Channel
  onSuccess?: (channel: Channel) => void
  onCancel?: () => void
}

/**
 * Slack Channel Configuration Component
 */
export function SlackConfig({ channel, onSuccess, onCancel }: SlackConfigProps) {
  const t = useTranslations('channels.config')
  const { toast } = useToast()
  const [isSubmitting, setIsSubmitting] = useState(false)
  const [showBotToken, setShowBotToken] = useState(false)
  const [showSigningSecret, setShowSigningSecret] = useState(false)
  const [showWebhookSecret, setShowWebhookSecret] = useState(false)
  const [testStatus, setTestStatus] = useState<'idle' | 'testing' | 'success' | 'error'>('idle')

  const isEditing = !!channel
  const form = useForm<SlackConfigForm>({
    resolver: zodResolver(slackConfigSchema(isEditing)),
    defaultValues: {
      name: channel?.name || '',
      bot_token: '',
      signing_secret: '',
      app_id: (channel?.config?.app_id as string) || '',
      bot_user_id: (channel?.config?.bot_user_id as string) || '',
      webhook_url: channel?.webhook_url || '',
      webhook_secret: '',
    },
  })

  const webhookUrl = channel
    ? `${WEBHOOK_BASE_URL}/api/v1/webhooks/slack/${channel.id}`
    : t('willBeGenerated')

  const onSubmit = async (data: SlackConfigForm) => {
    setIsSubmitting(true)
    try {
      const payload = {
        name: data.name,
        type: 'slack',
        config: {},
        credentials: {
          bot_token: data.bot_token,
          signing_secret: data.signing_secret,
          ...(data.app_id ? { app_id: data.app_id } : {}),
          ...(data.bot_user_id ? { bot_user_id: data.bot_user_id } : {}),
          ...(data.webhook_secret ? { webhook_secret: data.webhook_secret } : {}),
        },
        ...(data.webhook_url ? { webhook_url: data.webhook_url } : {}),
      }

      let result: Channel
      if (isEditing) {
        result = await api.put<Channel>(`/channels/${channel.id}`, payload)
      } else {
        result = await api.post<Channel>('/channels', payload)
      }

      toast({
        title: isEditing ? t('channelUpdated') : t('channelCreated'),
        description: isEditing
          ? t('channelUpdatedDesc', { name: data.name })
          : t('channelCreatedDesc', { name: data.name }),
      })

      onSuccess?.(result)
    } catch (error) {
      toast({
        title: t('error'),
        description: error instanceof Error ? error.message : t('failedToSave'),
        variant: 'error',
      })
    } finally {
      setIsSubmitting(false)
  }
  }

  const testConnection = async () => {
    const values = form.getValues()
    if (!values.bot_token) {
      toast({
        title: t('missingCredentials'),
        description: t('enterCredentialsFirst'),
        variant: 'error',
      })
      return
  }

    setTestStatus('testing')
    try {
      await api.post('/channels/test-slack', {
        bot_token: values.bot_token,
        ...(values.signing_secret ? { signing_secret: values.signing_secret } : {}),
      })
      setTestStatus('success')
      toast({
        title: t('connectionSuccess'),
        description: t('credentialsValid'),
      })
    } catch {
      setTestStatus('error')
      toast({
        title: t('connectionFailed'),
        description: t('checkCredentials'),
        variant: 'error',
      })
  }
  }

  const copyToClipboard = async (text: string) => {
    if (!(await copyText(text))) return
    toast({
      title: t('copied'),
      description: t('copiedToClipboard', { label: t('webhookUrl') }),
    })
  }

  return (
    <Form {...form}>
      <form onSubmit={form.handleSubmit(onSubmit)} className="flex flex-col h-full">
        <div className="flex-1 space-y-6">
          {/* Channel Name */}
          <FormField
            control={form.control}
            name="name"
            render={({ field }) => (
              <FormItem>
                <FormLabel>{t('channelName')}</FormLabel>
                <FormControl>
                  <Input placeholder={t('mySlackBot')} {...field} />
                </FormControl>
                <FormDescription>{t('channelNameDesc')}</FormDescription>
                <FormMessage />
              </FormItem>
            )}
          />

          {/* Bot Token */}
          <FormField
            control={form.control}
            name="bot_token"
            render={({ field }) => (
              <FormItem>
                <FormLabel>{t('slackBotToken')}</FormLabel>
                <FormControl>
                  <div className="relative">
                    <KeyRound className="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-muted-foreground" />
                    <Input
                      type={showBotToken ? 'text' : 'password'}
                      className="pl-10 pr-10"
                      placeholder={isEditing ? '••••••••••••••••' : t('slackBotTokenPlaceholder')}
                      {...field}
                    />
                    <Button
                      type="button"
                      variant="ghost"
                      size="icon"
                      className="absolute right-0 top-0 h-full"
                      onClick={() => setShowBotToken(!showBotToken)}
                    >
                      {showBotToken ? <EyeOff className="h-4 w-4" /> : <Eye className="h-4 w-4" />}
                    </Button>
                  </div>
                </FormControl>
                <FormDescription>{t('slackBotTokenDesc')}</FormDescription>
                <FormMessage />
              </FormItem>
            )}
          />

          {/* Signing Secret */}
          <FormField
            control={form.control}
            name="signing_secret"
            render={({ field }) => (
              <FormItem>
                <FormLabel>{t('slackSigningSecret')}</FormLabel>
                <FormControl>
                  <div className="relative">
                    <KeyRound className="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-muted-foreground" />
                    <Input
                      type={showSigningSecret ? 'text' : 'password'}
                      className="pl-10 pr-10"
                      placeholder={isEditing ? '••••••••••••••••' : t('slackSigningSecretPlaceholder')}
                      {...field}
                    />
                    <Button
                      type="button"
                      variant="ghost"
                      size="icon"
                      className="absolute right-0 top-0 h-full"
                      onClick={() => setShowSigningSecret(!showSigningSecret)}
                    >
                      {showSigningSecret ? <EyeOff className="h-4 w-4" /> : <Eye className="h-4 w-4" />}
                    </Button>
                  </div>
                </FormControl>
                <FormDescription>{t('slackSigningSecretDesc')}</FormDescription>
                <FormMessage />
              </FormItem>
            )}
          />

          {/* App ID (Optional) */}
          <FormField
            control={form.control}
            name="app_id"
            render={({ field }) => (
              <FormItem>
                <FormLabel>
                  {t('slackAppId')}
                  <Badge variant="outline" className="ml-2">{t('optional')}</Badge>
                </FormLabel>
                <FormControl>
                  <Input placeholder={t('slackAppIdPlaceholder')} {...field} />
                </FormControl>
                <FormDescription>{t('slackAppIdDesc')}</FormDescription>
                <FormMessage />
              </FormItem>
            )}
          />

          {/* Bot User ID (Optional) */}
          <FormField
            control={form.control}
            name="bot_user_id"
            render={({ field }) => (
              <FormItem>
                <FormLabel>
                  {t('slackBotUserId')}
                  <Badge variant="outline" className="ml-2">{t('optional')}</Badge>
                </FormLabel>
                <FormControl>
                  <Input placeholder={t('slackBotUserIdPlaceholder')} {...field} />
                </FormControl>
                <FormDescription>{t('slackBotUserIdDesc')}</FormDescription>
                <FormMessage />
              </FormItem>
            )}
          />

          {/* Webhook URL */}
          <Card>
            <CardHeader>
              <CardTitle className="text-base flex items-center gap-2">
                <Webhook className="h-4 w-4" />
                {t('webhookUrl')}
              </CardTitle>
              <CardDescription>{t('slackWebhookDesc')}</CardDescription>
            </CardHeader>
            <CardContent>
              <div className="flex items-center gap-2">
                <code className="flex-1 bg-muted px-3 py-2 rounded text-sm font-mono break-all">
                  {webhookUrl}
                </code>
                {channel && (
                  <Button
                    type="button"
                    variant="outline"
                    size="icon"
                    onClick={() => copyToClipboard(webhookUrl)}
                  >
                    <Copy className="h-4 w-4" />
                  </Button>
                )}
              </div>
            </CardContent>
          </Card>

          {/* Outbound webhook (DeskLenz) */}
          <Card>
            <CardHeader>
              <CardTitle className="text-base flex items-center gap-2">
                <Send className="h-4 w-4" />
                {t('outboundWebhookSection')}
                <Badge variant="outline" className="ml-1">{t('optional')}</Badge>
              </CardTitle>
              <CardDescription>{t('outboundWebhookDesc')}</CardDescription>
            </CardHeader>
            <CardContent className="space-y-4">
              <FormField
                control={form.control}
                name="webhook_url"
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>{t('consumerUrlLabel')}</FormLabel>
                    <FormControl>
                      <Input placeholder={t('consumerUrlPlaceholder')} {...field} />
                    </FormControl>
                    <FormDescription>{t('consumerUrlDesc')}</FormDescription>
                    <FormMessage />
                  </FormItem>
                )}
              />
              <FormField
                control={form.control}
                name="webhook_secret"
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>{t('webhookSecretLabel')}</FormLabel>
                    <FormControl>
                      <div className="relative">
                        <KeyRound className="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-muted-foreground" />
                        <Input
                          type={showWebhookSecret ? 'text' : 'password'}
                          className="pl-10 pr-10"
                          placeholder={isEditing ? '••••••••••••••••' : t('webhookSecretPlaceholder')}
                          {...field}
                        />
                        <Button
                          type="button"
                          variant="ghost"
                          size="icon"
                          className="absolute right-0 top-0 h-full"
                          onClick={() => setShowWebhookSecret(!showWebhookSecret)}
                        >
                          {showWebhookSecret ? <EyeOff className="h-4 w-4" /> : <Eye className="h-4 w-4" />}
                        </Button>
                      </div>
                    </FormControl>
                    <FormDescription>{t('webhookSecretDesc')}</FormDescription>
                    <FormMessage />
                  </FormItem>
                )}
              />
            </CardContent>
          </Card>
        </div>

        <div className="sticky bottom-0 flex justify-end gap-3 pt-4 pb-2 mt-4 border-t bg-background">
          {onCancel && (
            <Button type="button" variant="outline" onClick={onCancel}>
              {t('cancel')}
            </Button>
          )}
          <Button
            type="button"
            variant="outline"
            onClick={testConnection}
            disabled={testStatus === 'testing'}
          >
            {testStatus === 'testing' ? (
              <>
                <Loader2 className="h-4 w-4 mr-2 animate-spin" />
                {t('testing')}
              </>
            ) : (
              t('testConnection')
            )}
          </Button>
          <Button type="submit" disabled={isSubmitting}>
            {isSubmitting ? (
              <>
                <Loader2 className="h-4 w-4 mr-2 animate-spin" />
                {t('saving')}
              </>
            ) : isEditing ? (
              t('updateChannel')
            ) : (
              t('createChannel')
            )}
          </Button>
        </div>
      </form>
    </Form>
  )
}

export default SlackConfig
