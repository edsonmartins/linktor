'use client'

import { useEffect, useState } from 'react'
import { useForm } from 'react-hook-form'
import { useTranslations } from 'next-intl'
import { zodResolver } from '@hookform/resolvers/zod'
import { z } from 'zod'
import { Check, Copy, KeyRound, Loader2, MonitorSmartphone, PhoneOff, RefreshCw } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
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
import { Switch } from '@/components/ui/switch'
import { useToast } from '@/hooks/use-toast'
import { api, WEBHOOK_BASE_URL } from '@/lib/api'
import { copyText } from '@/lib/clipboard'
import type { Channel } from '@/types'
import {
  OutboundWebhookFields,
  outboundWebhookDefaults,
  outboundWebhookPayload,
  outboundWebhookSchema,
} from '@/components/channels/outbound-webhook-fields'

/**
 * WhatsApp via Linktor Bridge: a sessão do WhatsApp roda no computador do
 * atendente (app linktor-bridge) e liga ao Linktor por WebSocket de saída.
 * Aqui só se dá nome ao canal, gera-se o token do bridge e acompanha-se a
 * saúde da ligação; o pareamento por QR acontece no painel do bridge, no PC.
 */
const bridgeConfigSchema = z
  .object({
    name: z.string().min(1, 'Channel name is required'),
    // Chamadas: o WhatsApp só sinaliza; a chamada entra na conversa e pode
    // ser recusada pelo admin ou aqui, de forma automática (channel.config.*).
    reject_call: z.boolean(),
    reject_call_msg: z.string().optional(),
  })
  .merge(outboundWebhookSchema)

type BridgeConfigForm = z.infer<typeof bridgeConfigSchema>

interface BridgeHealth {
  online?: boolean
  stale?: boolean
  platform?: string
  version?: string
  connected_at?: string
  last_seen_ping?: string
  last_session?: { paired?: boolean; connected?: boolean; number?: string }
}

interface WhatsAppBridgeConfigProps {
  channel?: Channel
  onSuccess?: (channel: Channel) => void
  onCancel?: () => void
}

// A URL do gateway é a base pública da API com o esquema trocado para WebSocket.
const gatewayWsUrl = () =>
  `${WEBHOOK_BASE_URL.replace(/^http/, 'ws').replace(/\/$/, '')}/api/v1/gateways/ws`

export function WhatsAppBridgeConfig({ channel, onSuccess, onCancel }: WhatsAppBridgeConfigProps) {
  const t = useTranslations('channels.config')
  const tb = useTranslations('channels.config.bridge')
  const { toast } = useToast()
  const [isSubmitting, setIsSubmitting] = useState(false)
  const [health, setHealth] = useState<BridgeHealth | null>(
    (channel?.bridge_health as BridgeHealth | undefined) ?? null,
  )
  const [token, setToken] = useState<string | null>(null)
  const [generating, setGenerating] = useState(false)
  const [copied, setCopied] = useState<string | null>(null)

  const isEditing = !!channel

  const form = useForm<BridgeConfigForm>({
    resolver: zodResolver(bridgeConfigSchema),
    defaultValues: {
      name: channel?.name || '',
      reject_call: channel?.config?.reject_call === 'true',
      reject_call_msg: (channel?.config?.reject_call_msg as string) || '',
      ...outboundWebhookDefaults(channel),
    },
  })

  // Saúde do bridge a cada 10s enquanto a tela está aberta: o ping do bridge
  // chega a cada 20s, então "visto há" muda devagar mesmo.
  useEffect(() => {
    if (!channel?.id) return
    let cancelled = false
    const load = async () => {
      try {
        const h = await api.get<BridgeHealth>(`/channels/${channel.id}/bridge-health`)
        if (!cancelled) setHealth(h)
      } catch {
        /* sem bridge ligado o endpoint ainda responde; erro de rede só mantém o último */
      }
    }
    void load()
    const timer = setInterval(load, 10_000)
    return () => {
      cancelled = true
      clearInterval(timer)
    }
  }, [channel?.id])

  const onSubmit = async (data: BridgeConfigForm) => {
    setIsSubmitting(true)
    try {
      // O update troca o config inteiro: tudo que esta tela gerencia vai junto.
      const webhook = outboundWebhookPayload(data)
      const body = {
        name: data.name,
        ...webhook,
        config: {
          ...webhook.config,
          reject_call: String(data.reject_call),
          reject_call_msg: data.reject_call_msg || '',
        },
      }
      const payload = { type: 'whatsapp_remote', ...body }
      let result: Channel
      if (isEditing) {
        result = await api.put<Channel>(`/channels/${channel.id}`, body)
        toast({ title: t('channelUpdated'), description: t('channelUpdatedDesc', { name: data.name }) })
      } else {
        result = await api.post<Channel>('/channels', payload)
        toast({ title: t('channelCreated'), description: t('channelCreatedDesc', { name: data.name }) })
      }
      onSuccess?.(result)
    } catch (error: unknown) {
      const message = error instanceof Error ? error.message : t('failedToSave')
      toast({ title: t('error'), description: message, variant: 'error' })
    } finally {
      setIsSubmitting(false)
    }
  }

  const generateToken = async () => {
    if (!channel?.id) return
    setGenerating(true)
    try {
      const res = await api.post<{ bridge_token: string }>(`/channels/${channel.id}/bridge-token`)
      setToken(res.bridge_token)
      toast({ title: tb('tokenGenerated'), description: tb('tokenShownOnce') })
    } catch (error: unknown) {
      const message = error instanceof Error ? error.message : t('failedToSave')
      toast({ title: t('error'), description: message, variant: 'error' })
    } finally {
      setGenerating(false)
    }
  }

  const copy = async (key: string, value: string) => {
    if (await copyText(value)) {
      setCopied(key)
      setTimeout(() => setCopied(null), 2000)
    }
  }

  const command = channel
    ? `linktor-bridge --server ${gatewayWsUrl()} --channel-id ${channel.id} --token ${token ?? '<token>'}`
    : ''

  const online = !!health?.online && !health?.stale
  const session = health?.last_session

  return (
    <Form {...form}>
      <form onSubmit={form.handleSubmit(onSubmit)} className="flex flex-col h-full">
        <div className="flex-1 space-y-6">
          <FormField
            control={form.control}
            name="name"
            render={({ field }) => (
              <FormItem>
                <FormLabel>{t('channelName')}</FormLabel>
                <FormControl>
                  <Input placeholder={tb('namePlaceholder')} {...field} />
                </FormControl>
                <FormDescription>{t('channelNameDesc')}</FormDescription>
                <FormMessage />
              </FormItem>
            )}
          />

          <Card>
            <CardHeader className="pb-3">
              <CardTitle className="text-base flex items-center gap-2">
                <MonitorSmartphone className="h-4 w-4" />
                {tb('howItWorksTitle')}
              </CardTitle>
              <CardDescription>{tb('howItWorksDesc')}</CardDescription>
            </CardHeader>
            <CardContent>
              <ol className="list-decimal pl-5 space-y-1 text-sm text-muted-foreground">
                <li>{tb('step1')}</li>
                <li>{tb('step2')}</li>
                <li>{tb('step3')}</li>
              </ol>
            </CardContent>
          </Card>

          <Card>
            <CardHeader className="pb-3">
              <CardTitle className="text-base flex items-center gap-2">
                <PhoneOff className="h-4 w-4" />
                {tb('callsTitle')}
              </CardTitle>
              <CardDescription>{tb('callsDesc')}</CardDescription>
            </CardHeader>
            <CardContent className="space-y-4">
              <FormField
                control={form.control}
                name="reject_call"
                render={({ field }) => (
                  <FormItem>
                    <div className="flex items-center gap-2">
                      <FormControl>
                        <Switch checked={field.value} onCheckedChange={field.onChange} aria-label={t('rejectCall')} />
                      </FormControl>
                      <FormLabel className="!mt-0">{t('rejectCall')}</FormLabel>
                    </div>
                    <FormDescription>{t('rejectCallDesc')}</FormDescription>
                  </FormItem>
                )}
              />
              <FormField
                control={form.control}
                name="reject_call_msg"
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>{t('rejectCallMsg')}</FormLabel>
                    <FormControl>
                      <Input placeholder={t('rejectCallMsgPlaceholder')} {...field} />
                    </FormControl>
                    <FormDescription>{tb('rejectCallMsgDesc')}</FormDescription>
                    <FormMessage />
                  </FormItem>
                )}
              />
            </CardContent>
          </Card>

          <OutboundWebhookFields control={form.control} isEditing={isEditing} />

          {isEditing ? (
            <>
              <Card>
                <CardHeader className="pb-3">
                  <CardTitle className="text-base flex items-center justify-between">
                    <span>{tb('healthTitle')}</span>
                    <Badge variant={online ? 'default' : 'secondary'}>
                      {online ? tb('online') : health?.stale ? tb('stale') : tb('offline')}
                    </Badge>
                  </CardTitle>
                </CardHeader>
                <CardContent className="space-y-2 text-sm">
                  {health?.online ? (
                    <>
                      <p className="text-muted-foreground">
                        {health.platform} {health.version}
                        {health.last_seen_ping && (
                          <> · {tb('lastSeen', { at: new Date(health.last_seen_ping).toLocaleTimeString() })}</>
                        )}
                      </p>
                      <p>
                        {session?.paired
                          ? tb('whatsappPaired', { number: session.number ?? '' })
                          : tb('whatsappNotPaired')}
                        {session?.paired && (
                          <Badge variant="outline" className="ml-2">
                            {session.connected ? t('connected') : t('disconnected')}
                          </Badge>
                        )}
                      </p>
                    </>
                  ) : (
                    <p className="text-muted-foreground">{tb('notConnected')}</p>
                  )}
                </CardContent>
              </Card>

              <Card>
                <CardHeader className="pb-3">
                  <CardTitle className="text-base flex items-center gap-2">
                    <KeyRound className="h-4 w-4" />
                    {tb('valuesTitle')}
                  </CardTitle>
                  <CardDescription>{tb('valuesDesc')}</CardDescription>
                </CardHeader>
                <CardContent className="space-y-4">
                  <CopyRow label={tb('serverLabel')} value={gatewayWsUrl()} copied={copied === 'server'} onCopy={() => copy('server', gatewayWsUrl())} />
                  <CopyRow label={tb('channelIdLabel')} value={channel.id} copied={copied === 'channel'} onCopy={() => copy('channel', channel.id)} />

                  <div className="space-y-1">
                    <p className="text-xs font-medium">{tb('tokenLabel')}</p>
                    {token ? (
                      <>
                        <div className="flex gap-2">
                          <Input readOnly value={token} className="font-mono text-xs" />
                          <Button type="button" variant="outline" size="icon" onClick={() => copy('token', token)}>
                            {copied === 'token' ? <Check className="h-4 w-4" /> : <Copy className="h-4 w-4" />}
                          </Button>
                        </div>
                        <p className="text-xs text-amber-600">{tb('tokenShownOnce')}</p>
                      </>
                    ) : (
                      <p className="text-xs text-muted-foreground">{tb('tokenPending')}</p>
                    )}
                    <Button type="button" variant="outline" size="sm" onClick={generateToken} disabled={generating}>
                      {generating ? <Loader2 className="h-4 w-4 mr-2 animate-spin" /> : <RefreshCw className="h-4 w-4 mr-2" />}
                      {token ? tb('regenerateToken') : tb('generateToken')}
                    </Button>
                    <p className="text-xs text-muted-foreground">{tb('tokenDesc')}</p>
                  </div>

                  <div className="space-y-1 pt-2 border-t">
                    <p className="text-xs font-medium">{tb('commandTitle')}</p>
                    <div className="flex gap-2">
                      <Input readOnly value={command} className="font-mono text-xs" />
                      <Button type="button" variant="outline" size="icon" onClick={() => copy('cmd', command)}>
                        {copied === 'cmd' ? <Check className="h-4 w-4" /> : <Copy className="h-4 w-4" />}
                      </Button>
                    </div>
                    <p className="text-xs text-muted-foreground">{tb('commandDesc')}</p>
                  </div>
                </CardContent>
              </Card>
            </>
          ) : (
            <p className="text-sm text-muted-foreground">{tb('saveFirst')}</p>
          )}
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

/** Linha "rótulo + valor somente leitura + copiar", para a pessoa colar no painel do bridge. */
function CopyRow({ label, value, copied, onCopy }: { label: string; value: string; copied: boolean; onCopy: () => void }) {
  return (
    <div className="space-y-1">
      <p className="text-xs font-medium">{label}</p>
      <div className="flex gap-2">
        <Input readOnly value={value} className="font-mono text-xs" />
        <Button type="button" variant="outline" size="icon" onClick={onCopy}>
          {copied ? <Check className="h-4 w-4" /> : <Copy className="h-4 w-4" />}
        </Button>
      </div>
    </div>
  )
}

export default WhatsAppBridgeConfig
