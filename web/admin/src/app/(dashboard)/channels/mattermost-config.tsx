'use client'

import { useState } from 'react'
import { useForm } from 'react-hook-form'
import { useTranslations } from 'next-intl'
import { zodResolver } from '@hookform/resolvers/zod'
import { z } from 'zod'
import { Eye, EyeOff, Info, KeyRound, Loader2, Send } from 'lucide-react'
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
import { Alert, AlertDescription, AlertTitle } from '@/components/ui/alert'
import { Badge } from '@/components/ui/badge'
import { useToast } from '@/hooks/use-toast'
import { api } from '@/lib/api'
import type { Channel } from '@/types'

/**
 * Mattermost Configuration Schema
 */
const mattermostConfigSchema = z.object({
  name: z.string().min(1, 'Channel name is required'),
  base_url: z.string().min(1, 'Server URL is required'),
  bot_token: z.string().min(1, 'Bot token is required'),
  bot_user_id: z.string().optional(),
  webhook_url: z.string().url('Must be a valid URL').or(z.literal('')).optional(),
  webhook_secret: z.string().optional(),
})

type MattermostConfigForm = z.infer<typeof mattermostConfigSchema>

interface MattermostConfigProps {
  channel?: Channel
  onSuccess?: (channel: Channel) => void
  onCancel?: () => void
}

/**
 * Mattermost Channel Configuration Component
 */
export function MattermostConfig({ channel, onSuccess, onCancel }: MattermostConfigProps) {
  const t = useTranslations('channels.config')
  const { toast } = useToast()
  const [isSubmitting, setIsSubmitting] = useState(false)
  const [showBotToken, setShowBotToken] = useState(false)
  const [showWebhookSecret, setShowWebhookSecret] = useState(false)
  const [testStatus, setTestStatus] = useState<'idle' | 'testing' | 'success' | 'error'>('idle')

  const isEditing = !!channel

  const form = useForm<MattermostConfigForm>({
    resolver: zodResolver(mattermostConfigSchema),
    defaultValues: {
      name: channel?.name || '',
      base_url: (channel?.config?.base_url as string) || '',
      bot_token: '',
      bot_user_id: (channel?.config?.bot_user_id as string) || '',
      webhook_url: channel?.webhook_url || '',
      webhook_secret: '',
    },
  })

  const onSubmit = async (data: MattermostConfigForm) => {
    setIsSubmitting(true)
    try {
      const payload = {
        name: data.name,
        type: 'mattermost',
        config: {},
        credentials: {
          base_url: data.base_url,
          bot_token: data.bot_token,
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
    if (!values.base_url || !values.bot_token) {
      toast({
        title: t('missingCredentials'),
        description: t('enterCredentialsFirst'),
        variant: 'error',
      })
      return
    }

    setTestStatus('testing')
    try {
      await api.post('/channels/test-mattermost', {
        base_url: values.base_url,
        bot_token: values.bot_token,
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
                  <Input placeholder={t('myMattermostBot')} {...field} />
                </FormControl>
                <FormDescription>{t('channelNameDesc')}</FormDescription>
                <FormMessage />
              </FormItem>
            )}
          />

          {/* Server URL */}
          <FormField
            control={form.control}
            name="base_url"
            render={({ field }) => (
              <FormItem>
                <FormLabel>{t('mattermostBaseUrl')}</FormLabel>
                <FormControl>
                  <Input placeholder={t('mattermostBaseUrlPlaceholder')} {...field} />
                </FormControl>
                <FormDescription>{t('mattermostBaseUrlDesc')}</FormDescription>
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
                <FormLabel>{t('mattermostBotToken')}</FormLabel>
                <FormControl>
                  <div className="relative">
                    <KeyRound className="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-muted-foreground" />
                    <Input
                      type={showBotToken ? 'text' : 'password'}
                      className="pl-10 pr-10"
                      placeholder={isEditing ? '••••••••••••••••' : t('mattermostBotTokenPlaceholder')}
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
                <FormDescription>{t('mattermostBotTokenDesc')}</FormDescription>
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
                  {t('mattermostBotUserId')}
                  <Badge variant="outline" className="ml-2">{t('optional')}</Badge>
                </FormLabel>
                <FormControl>
                  <Input placeholder={t('mattermostBotUserIdPlaceholder')} {...field} />
                </FormControl>
                <FormDescription>{t('mattermostBotUserIdDesc')}</FormDescription>
                <FormMessage />
              </FormItem>
            )}
          />

          {/* Connectivity info (outbound WebSocket, no inbound webhook) */}
          <Alert>
            <Info className="h-4 w-4" />
            <AlertTitle>{t('mattermostInfoTitle')}</AlertTitle>
            <AlertDescription>{t('mattermostInfoDesc')}</AlertDescription>
          </Alert>

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

export default MattermostConfig
