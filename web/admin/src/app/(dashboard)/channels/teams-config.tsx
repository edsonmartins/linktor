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
 * Microsoft Teams Configuration Schema
 */
const teamsConfigSchema = z.object({
  name: z.string().min(1, 'Channel name is required'),
  app_id: z.string().min(1, 'App ID is required'),
  app_password: z.string().min(1, 'App password is required'),
  tenant_id: z.string().optional(),
  webhook_url: z.string().url('Must be a valid URL').or(z.literal('')).optional(),
  webhook_secret: z.string().optional(),
})

type TeamsConfigForm = z.infer<typeof teamsConfigSchema>

interface TeamsConfigProps {
  channel?: Channel
  onSuccess?: (channel: Channel) => void
  onCancel?: () => void
}

/**
 * Microsoft Teams Channel Configuration Component
 */
export function TeamsConfig({ channel, onSuccess, onCancel }: TeamsConfigProps) {
  const t = useTranslations('channels.config')
  const { toast } = useToast()
  const [isSubmitting, setIsSubmitting] = useState(false)
  const [showPassword, setShowPassword] = useState(false)
  const [showWebhookSecret, setShowWebhookSecret] = useState(false)
  const [testStatus, setTestStatus] = useState<'idle' | 'testing' | 'success' | 'error'>('idle')

  const isEditing = !!channel

  const form = useForm<TeamsConfigForm>({
    resolver: zodResolver(teamsConfigSchema),
    defaultValues: {
      name: channel?.name || '',
      app_id: (channel?.config?.app_id as string) || '',
      app_password: '',
      tenant_id: (channel?.config?.tenant_id as string) || '',
      webhook_url: channel?.webhook_url || '',
      webhook_secret: '',
    },
  })

  const webhookUrl = channel
    ? `${WEBHOOK_BASE_URL}/api/v1/webhooks/teams/${channel.id}`
    : t('willBeGenerated')

  const onSubmit = async (data: TeamsConfigForm) => {
    setIsSubmitting(true)
    try {
      const payload = {
        name: data.name,
        type: 'teams',
        config: {},
        credentials: {
          app_id: data.app_id,
          app_password: data.app_password,
          ...(data.tenant_id ? { tenant_id: data.tenant_id } : {}),
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
    if (!values.app_id || !values.app_password) {
      toast({
        title: t('missingCredentials'),
        description: t('enterCredentialsFirst'),
        variant: 'error',
      })
      return
    }

    setTestStatus('testing')
    try {
      await api.post('/channels/test-teams', {
        app_id: values.app_id,
        app_password: values.app_password,
        ...(values.tenant_id ? { tenant_id: values.tenant_id } : {}),
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
                  <Input placeholder={t('myTeamsBot')} {...field} />
                </FormControl>
                <FormDescription>{t('channelNameDesc')}</FormDescription>
                <FormMessage />
              </FormItem>
            )}
          />

          {/* App ID */}
          <FormField
            control={form.control}
            name="app_id"
            render={({ field }) => (
              <FormItem>
                <FormLabel>{t('teamsAppId')}</FormLabel>
                <FormControl>
                  <Input placeholder={t('teamsAppIdPlaceholder')} {...field} />
                </FormControl>
                <FormDescription>{t('teamsAppIdDesc')}</FormDescription>
                <FormMessage />
              </FormItem>
            )}
          />

          {/* App Password */}
          <FormField
            control={form.control}
            name="app_password"
            render={({ field }) => (
              <FormItem>
                <FormLabel>{t('teamsAppPassword')}</FormLabel>
                <FormControl>
                  <div className="relative">
                    <KeyRound className="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-muted-foreground" />
                    <Input
                      type={showPassword ? 'text' : 'password'}
                      className="pl-10 pr-10"
                      placeholder={isEditing ? '••••••••••••••••' : t('teamsAppPasswordPlaceholder')}
                      {...field}
                    />
                    <Button
                      type="button"
                      variant="ghost"
                      size="icon"
                      className="absolute right-0 top-0 h-full"
                      onClick={() => setShowPassword(!showPassword)}
                    >
                      {showPassword ? <EyeOff className="h-4 w-4" /> : <Eye className="h-4 w-4" />}
                    </Button>
                  </div>
                </FormControl>
                <FormDescription>{t('teamsAppPasswordDesc')}</FormDescription>
                <FormMessage />
              </FormItem>
            )}
          />

          {/* Tenant ID (Optional) */}
          <FormField
            control={form.control}
            name="tenant_id"
            render={({ field }) => (
              <FormItem>
                <FormLabel>
                  {t('teamsTenantId')}
                  <Badge variant="outline" className="ml-2">{t('optional')}</Badge>
                </FormLabel>
                <FormControl>
                  <Input placeholder={t('teamsTenantIdPlaceholder')} {...field} />
                </FormControl>
                <FormDescription>{t('teamsTenantIdDesc')}</FormDescription>
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
              <CardDescription>{t('teamsWebhookDesc')}</CardDescription>
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

export default TeamsConfig
