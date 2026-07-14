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
  Link2,
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
import type { Channel } from '@/types'

/**
 * Direto (VendaX) Configuration Schema.
 *
 * The Direto sender needs send_url + instance_id + api_token (RFC-009). send_url
 * and instance_id are non-secret and stored in config (so they prefill on edit);
 * api_token is the Bearer secret and stored in credentials (redacted on read).
 */
const diretoConfigSchema = z.object({
  name: z.string().min(1, 'Channel name is required'),
  send_url: z.string().url('Must be a valid URL'),
  instance_id: z.string().min(1, 'Instance ID is required'),
  api_token: z.string().optional(),
  webhook_url: z.string().url('Must be a valid URL').or(z.literal('')).optional(),
  webhook_secret: z.string().optional(),
})

type DiretoConfigForm = z.infer<typeof diretoConfigSchema>

interface DiretoConfigProps {
  channel?: Channel
  onSuccess?: (channel: Channel) => void
  onCancel?: () => void
}

export function DiretoConfig({ channel, onSuccess, onCancel }: DiretoConfigProps) {
  const t = useTranslations('channels.config')
  const { toast } = useToast()
  const [isSubmitting, setIsSubmitting] = useState(false)
  const [showApiToken, setShowApiToken] = useState(false)
  const [showWebhookSecret, setShowWebhookSecret] = useState(false)

  const isEditing = !!channel

  const form = useForm<DiretoConfigForm>({
    resolver: zodResolver(diretoConfigSchema),
    defaultValues: {
      name: channel?.name || '',
      send_url: (channel?.config?.send_url as string) || '',
      instance_id: (channel?.config?.instance_id as string) || '',
      api_token: '',
      webhook_url: channel?.webhook_url || '',
      webhook_secret: '',
    },
  })

  // Inbound URL where the Direto gateway POSTs messages TO Linktor.
  const inboundWebhookUrl = channel
    ? `${WEBHOOK_BASE_URL}/api/v1/webhooks/direto/${channel.id}`
    : t('willBeGenerated')

  const onSubmit = async (data: DiretoConfigForm) => {
    setIsSubmitting(true)
    try {
      const payload = {
        name: data.name,
        type: 'direto',
        // Non-secret sender settings live in config (prefill on edit). Config is
        // replaced wholesale on update, so both are always sent.
        config: {
          send_url: data.send_url,
          instance_id: data.instance_id,
        },
        // Secrets live in credentials (merged on update: blank keeps the stored
        // value, so api_token can be left empty when editing).
        credentials: {
          ...(data.api_token ? { api_token: data.api_token } : {}),
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

  const copyToClipboard = (text: string) => {
    navigator.clipboard.writeText(text)
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
                  <Input placeholder={t('channelNamePlaceholder')} {...field} />
                </FormControl>
                <FormDescription>{t('channelNameDesc')}</FormDescription>
                <FormMessage />
              </FormItem>
            )}
          />

          {/* Send URL */}
          <FormField
            control={form.control}
            name="send_url"
            render={({ field }) => (
              <FormItem>
                <FormLabel>{t('diretoSendUrl')}</FormLabel>
                <FormControl>
                  <div className="relative">
                    <Link2 className="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-muted-foreground" />
                    <Input className="pl-10" placeholder={t('diretoSendUrlPlaceholder')} {...field} />
                  </div>
                </FormControl>
                <FormDescription>{t('diretoSendUrlDesc')}</FormDescription>
                <FormMessage />
              </FormItem>
            )}
          />

          {/* Instance ID */}
          <FormField
            control={form.control}
            name="instance_id"
            render={({ field }) => (
              <FormItem>
                <FormLabel>{t('diretoInstanceId')}</FormLabel>
                <FormControl>
                  <Input placeholder={t('diretoInstanceIdPlaceholder')} {...field} />
                </FormControl>
                <FormDescription>{t('diretoInstanceIdDesc')}</FormDescription>
                <FormMessage />
              </FormItem>
            )}
          />

          {/* API Token */}
          <FormField
            control={form.control}
            name="api_token"
            render={({ field }) => (
              <FormItem>
                <FormLabel>{t('diretoApiToken')}</FormLabel>
                <FormControl>
                  <div className="relative">
                    <KeyRound className="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-muted-foreground" />
                    <Input
                      type={showApiToken ? 'text' : 'password'}
                      className="pl-10 pr-10"
                      placeholder={isEditing ? '••••••••••••••••' : t('diretoApiTokenPlaceholder')}
                      {...field}
                    />
                    <Button
                      type="button"
                      variant="ghost"
                      size="icon"
                      className="absolute right-0 top-0 h-full"
                      onClick={() => setShowApiToken(!showApiToken)}
                    >
                      {showApiToken ? <EyeOff className="h-4 w-4" /> : <Eye className="h-4 w-4" />}
                    </Button>
                  </div>
                </FormControl>
                <FormDescription>{t('diretoApiTokenDesc')}</FormDescription>
                <FormMessage />
              </FormItem>
            )}
          />

          {/* Inbound webhook URL (Direto → Linktor) */}
          <Card>
            <CardHeader>
              <CardTitle className="text-base flex items-center gap-2">
                <Webhook className="h-4 w-4" />
                {t('webhookUrl')}
              </CardTitle>
              <CardDescription>{t('diretoInboundWebhookDesc')}</CardDescription>
            </CardHeader>
            <CardContent>
              <div className="flex items-center gap-2">
                <code className="flex-1 bg-muted px-3 py-2 rounded text-sm font-mono break-all">
                  {inboundWebhookUrl}
                </code>
                {channel && (
                  <Button
                    type="button"
                    variant="outline"
                    size="icon"
                    onClick={() => copyToClipboard(inboundWebhookUrl)}
                  >
                    <Copy className="h-4 w-4" />
                  </Button>
                )}
              </div>
            </CardContent>
          </Card>

          {/* Outbound webhook (external consumer) */}
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

export default DiretoConfig
