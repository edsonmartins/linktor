'use client'

import { useState } from 'react'
import type { Control, FieldValues, Path } from 'react-hook-form'
import { useTranslations } from 'next-intl'
import { z } from 'zod'
import { Eye, EyeOff, KeyRound, Send } from 'lucide-react'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { FormControl, FormDescription, FormField, FormItem, FormLabel, FormMessage } from '@/components/ui/form'
import { Input } from '@/components/ui/input'
import { Switch } from '@/components/ui/switch'
import type { Channel } from '@/types'

/**
 * Webhook de saída de um canal: o Linktor faz POST assinado (linktor-channel-v1)
 * para um consumidor externo a cada evento. O backend trata isso igual para
 * qualquer tipo de canal (channel.webhook_url, credentials.webhook_secret,
 * config.webhook_events), então o bloco é um só, usado por todas as telas.
 */

/** Tipos de evento, na ordem da lista. Vazio quer dizer "entregar todos". */
export const WEBHOOK_EVENT_TYPES = [
  'message.received',
  'message.sent',
  'message.delivered',
  'message.read',
  'message.failed',
  'contact.created',
  'conversation.created',
  'conversation.assigned',
  'conversation.resolved',
  'conversation.reopened',
  'conversation.escalated',
] as const

type WebhookEventType = (typeof WEBHOOK_EVENT_TYPES)[number]

const WEBHOOK_EVENT_LABEL_KEYS: Record<WebhookEventType, string> = {
  'message.received': 'eventMessageReceived',
  'message.sent': 'eventMessageSent',
  'message.delivered': 'eventMessageDelivered',
  'message.read': 'eventMessageRead',
  'message.failed': 'eventMessageFailed',
  'contact.created': 'eventContactCreated',
  'conversation.created': 'eventConversationCreated',
  'conversation.assigned': 'eventConversationAssigned',
  'conversation.resolved': 'eventConversationResolved',
  'conversation.reopened': 'eventConversationReopened',
  'conversation.escalated': 'eventConversationEscalated',
}

/** Campos do formulário; cada tela faz `schema.merge(outboundWebhookSchema)`. */
export const outboundWebhookSchema = z.object({
  webhook_url: z.string().url('Must be a valid URL').or(z.literal('')).optional(),
  webhook_secret: z.string().optional(),
  webhook_events: z.array(z.string()).optional(),
})

export type OutboundWebhookFormValues = z.infer<typeof outboundWebhookSchema>

/** Lê a lista separada por vírgula guardada em config.webhook_events. */
export function parseWebhookEvents(raw: unknown): string[] {
  if (typeof raw !== 'string' || raw.trim() === '') return []
  return raw
    .split(',')
    .map((e) => e.trim())
    .filter((e) => (WEBHOOK_EVENT_TYPES as readonly string[]).includes(e))
}

/** Valores iniciais a partir do canal; o segredo nunca volta da API. */
export function outboundWebhookDefaults(channel?: Channel): OutboundWebhookFormValues {
  return {
    webhook_url: channel?.webhook_url || '',
    webhook_secret: '',
    webhook_events: parseWebhookEvents(channel?.config?.webhook_events),
  }
}

/**
 * Pedaços do payload de create/update. O segredo só vai quando digitado (em
 * branco mantém o atual); a lista de eventos vai sempre, para "nenhum" valer
 * como "todos".
 */
export function outboundWebhookPayload(values: OutboundWebhookFormValues) {
  return {
    config: { webhook_events: (values.webhook_events || []).join(',') },
    credentials: values.webhook_secret ? { webhook_secret: values.webhook_secret } : {},
    ...(values.webhook_url ? { webhook_url: values.webhook_url } : {}),
  }
}

interface OutboundWebhookFieldsProps<T extends FieldValues> {
  control: Control<T>
  isEditing: boolean
}

export function OutboundWebhookFields<T extends OutboundWebhookFormValues & FieldValues>({
  control,
  isEditing,
}: OutboundWebhookFieldsProps<T>) {
  const t = useTranslations('channels.config')
  const [showSecret, setShowSecret] = useState(false)

  return (
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
          control={control}
          name={'webhook_url' as Path<T>}
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
          control={control}
          name={'webhook_secret' as Path<T>}
          render={({ field }) => (
            <FormItem>
              <FormLabel>{t('webhookSecretLabel')}</FormLabel>
              <FormControl>
                <div className="relative">
                  <KeyRound className="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-muted-foreground" />
                  <Input
                    type={showSecret ? 'text' : 'password'}
                    className="pl-10 pr-10"
                    placeholder={isEditing ? '••••••••••••••••' : t('webhookSecretPlaceholder')}
                    {...field}
                  />
                  <Button
                    type="button"
                    variant="ghost"
                    size="icon"
                    className="absolute right-0 top-0 h-full"
                    onClick={() => setShowSecret(!showSecret)}
                  >
                    {showSecret ? <EyeOff className="h-4 w-4" /> : <Eye className="h-4 w-4" />}
                  </Button>
                </div>
              </FormControl>
              <FormDescription>{t('webhookSecretDesc')}</FormDescription>
              <FormMessage />
            </FormItem>
          )}
        />
        <FormField
          control={control}
          name={'webhook_events' as Path<T>}
          render={({ field }) => {
            const selected: string[] = field.value || []
            const toggle = (eventType: string, on: boolean) => {
              field.onChange(on ? [...selected.filter((e) => e !== eventType), eventType] : selected.filter((e) => e !== eventType))
            }
            return (
              <FormItem>
                <FormLabel>{t('webhookEventsLabel')}</FormLabel>
                <div className="space-y-2 rounded-lg border p-3">
                  {WEBHOOK_EVENT_TYPES.map((eventType) => (
                    <div key={eventType} className="flex items-center justify-between gap-3">
                      <div className="flex flex-col">
                        <span className="text-sm">{t(WEBHOOK_EVENT_LABEL_KEYS[eventType])}</span>
                        <code className="text-xs text-muted-foreground font-mono">{eventType}</code>
                      </div>
                      <Switch checked={selected.includes(eventType)} onCheckedChange={(on) => toggle(eventType, on)} />
                    </div>
                  ))}
                </div>
                <FormDescription>{t('webhookEventsDesc')}</FormDescription>
                <FormMessage />
              </FormItem>
            )
          }}
        />
      </CardContent>
    </Card>
  )
}
