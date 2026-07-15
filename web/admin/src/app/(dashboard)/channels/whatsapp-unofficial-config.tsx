'use client'

import { useState, useEffect, useRef } from 'react'
import { useForm } from 'react-hook-form'
import { useTranslations } from 'next-intl'
import { zodResolver } from '@hookform/resolvers/zod'
import { z } from 'zod'
import {
  AlertCircle,
  CheckCircle2,
  Eye,
  EyeOff,
  KeyRound,
  Loader2,
  QrCode,
  RefreshCw,
  Send,
  Smartphone,
} from 'lucide-react'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Switch } from '@/components/ui/switch'
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
import {
  Alert,
  AlertDescription,
  AlertTitle,
} from '@/components/ui/alert'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from '@/components/ui/dialog'
import {
  Tabs,
  TabsContent,
  TabsList,
  TabsTrigger,
} from '@/components/ui/tabs'
import { Badge } from '@/components/ui/badge'
import { Separator } from '@/components/ui/separator'
import { useToast } from '@/hooks/use-toast'
import { api } from '@/lib/api'
import {
  detectPasskeyConnector,
  runPasskeyAssertion,
  type PasskeyPublicKey,
} from '@/lib/passkey-connector'
import type { Channel } from '@/types'

/**
 * Outbound webhook event types (linktor-channel-v1). Order matches the UI list.
 * Stored in channel.config.webhook_events as a comma-separated string; an empty
 * list means "deliver all events" (matches the backend default).
 */
const WEBHOOK_EVENT_TYPES = [
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

const WEBHOOK_EVENT_LABEL_KEYS: Record<(typeof WEBHOOK_EVENT_TYPES)[number], string> = {
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

/**
 * WhatsApp Unofficial Configuration Schema
 */
const whatsappConfigSchema = z.object({
  name: z.string().min(1, 'Channel name is required'),
  device_name: z.string().optional(),
  phone_number: z.string().optional(),
  webhook_url: z.string().url('Must be a valid URL').or(z.literal('')).optional(),
  webhook_secret: z.string().optional(),
  webhook_events: z.array(z.string()).optional(),
})

type WhatsAppConfigForm = z.infer<typeof whatsappConfigSchema>

/** Parse the stored comma-separated webhook_events config into a clean array. */
function parseWebhookEvents(raw: unknown): string[] {
  if (typeof raw !== 'string' || raw.trim() === '') return []
  return raw
    .split(',')
    .map((e) => e.trim())
    .filter((e) => WEBHOOK_EVENT_TYPES.includes(e as (typeof WEBHOOK_EVENT_TYPES)[number]))
}

interface WhatsAppUnofficialConfigProps {
  channel?: Channel
  onSuccess?: (channel: Channel) => void
  onCancel?: () => void
}

type ConnectionStatus =
  | 'disconnected'
  | 'connecting'
  | 'qr_pending'
  | 'passkey_pending'
  | 'connected'
  | 'logged_out'

/**
 * WhatsApp Unofficial Channel Configuration Component
 */
export function WhatsAppUnofficialConfig({
  channel,
  onSuccess,
  onCancel,
}: WhatsAppUnofficialConfigProps) {
  const t = useTranslations('channels.config')
  const tCommon = useTranslations('common')
  const { toast } = useToast()
  const [isSubmitting, setIsSubmitting] = useState(false)
  const [connectionStatus, setConnectionStatus] = useState<ConnectionStatus>('disconnected')
  const [qrCode, setQrCode] = useState<string | null>(null)
  const [qrExpiry, setQrExpiry] = useState<number>(0)
  const [pairCode, setPairCode] = useState<string | null>(null)
  const [deviceInfo, setDeviceInfo] = useState<any>(null)
  const [showWebhookSecret, setShowWebhookSecret] = useState(false)
  const [passkeyChallenge, setPasskeyChallenge] = useState<PasskeyPublicKey | null>(null)
  const [passkeyBusy, setPasskeyBusy] = useState(false)
  const [passkeyError, setPasskeyError] = useState<string | null>(null)
  const wsRef = useRef<WebSocket | null>(null)

  const isEditing = !!channel

  useEffect(() => {
    if (!channel?.connection_status) return

    if (channel.connection_status === 'connected') {
      setConnectionStatus('connected')
      return
    }

    if (channel.connection_status === 'connecting') {
      setConnectionStatus('connecting')
      return
    }

    setConnectionStatus('disconnected')
  }, [channel?.connection_status])

  const form = useForm<WhatsAppConfigForm>({
    resolver: zodResolver(whatsappConfigSchema),
    defaultValues: {
      name: channel?.name || '',
      device_name: (channel?.config?.device_name as string) || 'Linktor',
      phone_number: '',
      webhook_url: channel?.webhook_url || '',
      webhook_secret: '',
      webhook_events: parseWebhookEvents(channel?.config?.webhook_events),
    },
  })

  // Cleanup on unmount
  useEffect(() => {
    return () => {
      if (wsRef.current) {
        wsRef.current.close()
      }
    }
  }, [])

  // QR code countdown
  useEffect(() => {
    if (qrExpiry > 0) {
      const interval = setInterval(() => {
        setQrExpiry((prev) => Math.max(0, prev - 1))
      }, 1000)
      return () => clearInterval(interval)
    }
  }, [qrExpiry])

  // Poll channel status while a QR code or a passkey challenge is pending.
  useEffect(() => {
    if (
      (connectionStatus !== 'qr_pending' && connectionStatus !== 'passkey_pending') ||
      !channel?.id
    ) {
      return
    }

    const pollInterval = setInterval(async () => {
      try {
        const response = await api.get<Channel>(`/channels/${channel.id}`)
        if (response.connection_status === 'connected') {
          setConnectionStatus('connected')
          setQrCode(null)
          setPairCode(null)
          setPasskeyChallenge(null)
          toast({
            title: t('channelConnected'),
            description: t('channelConnectedDesc'),
          })
          onSuccess?.(response)
        }
      } catch (error) {
        // Ignore polling errors
        // Status poll error - silently ignored
      }
    }, 2000) // Poll every 2 seconds

    return () => clearInterval(pollInterval)
  }, [connectionStatus, channel?.id, t, toast, onSuccess])

  const onSubmit = async (data: WhatsAppConfigForm) => {
    setIsSubmitting(true)
    try {
      const payload = {
        name: data.name,
        type: 'whatsapp',
        config: {
          device_name: data.device_name,
          // Comma-separated list; empty string means "deliver all events".
          webhook_events: (data.webhook_events || []).join(','),
        },
        credentials: {
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

  const startQRLogin = async (channelId: string) => {
    setConnectionStatus('connecting')
    setQrCode(null)
    setPairCode(null)

    setPasskeyChallenge(null)
    setPasskeyError(null)

    try {
      const response = await api.post<{
        channel: Channel
        qr_code?: string
        expires_in?: number
        passkey_required?: boolean
        passkey_challenge?: PasskeyPublicKey
      }>(`/channels/${channelId}/connect`)

      if (response.passkey_required && response.passkey_challenge) {
        // Passkey-locked account: no QR. The owner must sign the challenge in
        // their browser via the Linktor passkey extension.
        setConnectionStatus('passkey_pending')
        setPasskeyChallenge(response.passkey_challenge)
      } else if (response.qr_code) {
        setConnectionStatus('qr_pending')
        setQrCode(response.qr_code)
        setQrExpiry(response.expires_in || 60)
      } else {
        // Channel connected without QR (already authenticated or different auth method)
        setConnectionStatus('connected')
        toast({
          title: t('channelConnected'),
          description: t('channelConnectedDesc'),
        })
      }
    } catch (error) {
      setConnectionStatus('disconnected')
      toast({
        title: t('error'),
        description: t('failedToStartQr'),
        variant: 'error',
      })
    }
  }

  // Drives the passkey resolution: detect the extension, run the WebAuthn
  // assertion in the owner's browser, and submit it. On success the status poll
  // flips the channel to connected.
  const resolvePasskey = async () => {
    if (!channel?.id || !passkeyChallenge) return
    setPasskeyBusy(true)
    setPasskeyError(null)
    try {
      const installed = await detectPasskeyConnector()
      if (!installed) {
        setPasskeyError(t('passkeyExtensionMissing'))
        return
      }
      const assertion = await runPasskeyAssertion(passkeyChallenge)
      // Forward the assertion verbatim; do not re-encode its base64url fields.
      await api.post(`/channels/${channel.id}/passkey/response`, assertion)
      setConnectionStatus('connecting')
      toast({
        title: t('passkeySubmitted'),
        description: t('passkeySubmittedDesc'),
      })
    } catch (error) {
      const reason = error instanceof Error ? error.message : String(error)
      setPasskeyError(reason === 'timeout' ? t('passkeyTimeout') : t('passkeyFailed'))
    } finally {
      setPasskeyBusy(false)
    }
  }

  const startPairCodeLogin = async () => {
    const phoneNumber = form.getValues('phone_number')
    if (!phoneNumber) {
      toast({
        title: t('phoneNumberRequired'),
        description: t('enterPhoneForPairCode'),
        variant: 'error',
      })
      return
    }

    setConnectionStatus('connecting')
    setPairCode(null)
    setQrCode(null)

    try {
      const response = await api.post<{ channel: Channel; code?: string; expires_in?: number }>(
        `/channels/${channel?.id}/pair`,
        { phone_number: phoneNumber }
      )

      if (response.code) {
        setPairCode(response.code)
        setQrExpiry(response.expires_in || 300)
        setConnectionStatus('qr_pending')
      } else {
        // Pair code not supported yet
        toast({
          title: t('notAvailable'),
          description: t('pairCodeNotAvailable'),
          variant: 'warning',
        })
        setConnectionStatus('disconnected')
      }
    } catch (error) {
      setConnectionStatus('disconnected')
      toast({
        title: t('error'),
        description: t('failedToGetPairCode'),
        variant: 'error',
      })
    }
  }

  const refreshQR = () => {
    if (channel?.id) {
      startQRLogin(channel.id)
    }
  }

  const disconnect = async () => {
    if (!channel?.id) return

    try {
      await api.post(`/channels/${channel.id}/disconnect`)
      setConnectionStatus('disconnected')
      setDeviceInfo(null)
      toast({
        title: t('disconnectedSuccess'),
        description: t('disconnectedDesc'),
      })
    } catch (error) {
      toast({
        title: t('error'),
        description: t('failedToDisconnect'),
        variant: 'error',
      })
    }
  }

  const getStatusBadge = () => {
    switch (connectionStatus) {
      case 'connected':
        return <Badge variant="success" className="gap-1"><CheckCircle2 className="h-3 w-3" /> {t('connected')}</Badge>
      case 'connecting':
        return <Badge variant="secondary" className="gap-1"><Loader2 className="h-3 w-3 animate-spin" /> {t('connecting')}</Badge>
      case 'qr_pending':
        return <Badge variant="warning" className="gap-1"><QrCode className="h-3 w-3" /> {t('scanQrCode')}</Badge>
      case 'logged_out':
        return <Badge variant="error" className="gap-1"><AlertCircle className="h-3 w-3" /> {t('loggedOut')}</Badge>
      default:
        return <Badge variant="secondary" className="gap-1">{t('disconnected')}</Badge>
    }
  }

  return (
    <Form {...form}>
      <form onSubmit={form.handleSubmit(onSubmit)} className="flex flex-col h-full">
        <div className="flex-1 space-y-6">
        <Tabs defaultValue="setup" className="w-full">
          <TabsList className="grid w-full grid-cols-2">
            <TabsTrigger value="setup">{t('setup')}</TabsTrigger>
            <TabsTrigger value="connection">{t('connection')}</TabsTrigger>
          </TabsList>

          <TabsContent value="setup" className="space-y-4 mt-4">
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
                  <FormDescription>
                    {t('channelNameDesc')}
                  </FormDescription>
                  <FormMessage />
                </FormItem>
              )}
            />

            {/* Device Name */}
            <FormField
              control={form.control}
              name="device_name"
              render={({ field }) => (
                <FormItem>
                  <FormLabel>{t('deviceName')}</FormLabel>
                  <FormControl>
                    <Input placeholder={t('deviceNamePlaceholder')} {...field} />
                  </FormControl>
                  <FormDescription>
                    {t('deviceNameDesc')}
                  </FormDescription>
                  <FormMessage />
                </FormItem>
              )}
            />

            <Alert>
              <Smartphone className="h-4 w-4" />
              <AlertTitle>{t('multiDeviceSupport')}</AlertTitle>
              <AlertDescription>
                {t('multiDeviceDesc')}
              </AlertDescription>
            </Alert>

            {/* Outbound webhook (DeskLenz / external consumer) */}
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
                <FormField
                  control={form.control}
                  name="webhook_events"
                  render={({ field }) => {
                    const selected = field.value || []
                    const toggle = (eventType: string, on: boolean) => {
                      if (on) {
                        field.onChange([...selected.filter((e) => e !== eventType), eventType])
                      } else {
                        field.onChange(selected.filter((e) => e !== eventType))
                      }
                    }
                    return (
                      <FormItem>
                        <FormLabel>{t('webhookEventsLabel')}</FormLabel>
                        <div className="space-y-2 rounded-lg border p-3">
                          {WEBHOOK_EVENT_TYPES.map((eventType) => (
                            <div
                              key={eventType}
                              className="flex items-center justify-between gap-3"
                            >
                              <div className="flex flex-col">
                                <span className="text-sm">{t(WEBHOOK_EVENT_LABEL_KEYS[eventType])}</span>
                                <code className="text-xs text-muted-foreground font-mono">{eventType}</code>
                              </div>
                              <Switch
                                checked={selected.includes(eventType)}
                                onCheckedChange={(on) => toggle(eventType, on)}
                              />
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
          </TabsContent>

          <TabsContent value="connection" className="space-y-4 mt-4">
            {/* Connection Status */}
            <Card>
              <CardHeader>
                <div className="flex items-center justify-between">
                  <CardTitle className="text-base">{t('connectionStatus')}</CardTitle>
                  {getStatusBadge()}
                </div>
                <CardDescription>
                  {connectionStatus === 'connected' && deviceInfo
                    ? t('connectedAs', { phone: deviceInfo.phone_number || deviceInfo.jid })
                    : t('connectAccount')}
                </CardDescription>
              </CardHeader>
              <CardContent>
                {connectionStatus === 'connected' ? (
                  <div className="space-y-4">
                    {deviceInfo && (
                      <div className="bg-muted p-4 rounded-lg space-y-2">
                        <div className="flex justify-between text-sm">
                          <span className="text-muted-foreground">{t('phone')}:</span>
                          <span>{deviceInfo.phone_number || deviceInfo.jid}</span>
                        </div>
                        <div className="flex justify-between text-sm">
                          <span className="text-muted-foreground">{t('device')}:</span>
                          <span>{deviceInfo.display_name || 'Linktor'}</span>
                        </div>
                      </div>
                    )}
                    <Button
                      type="button"
                      variant="destructive"
                      onClick={disconnect}
                      className="w-full"
                    >
                      {t('disconnect')}
                    </Button>
                  </div>
                ) : connectionStatus === 'qr_pending' && qrCode ? (
                  <div className="space-y-4">
                    {/* QR Code Display */}
                    <div className="flex flex-col items-center space-y-4">
                      <div className="bg-white p-4 rounded-lg">
                        <img
                          src={`https://api.qrserver.com/v1/create-qr-code/?size=200x200&data=${encodeURIComponent(qrCode)}`}
                          alt="WhatsApp QR Code"
                          className="w-48 h-48"
                        />
                      </div>
                      <div className="text-center">
                        <p className="text-sm text-muted-foreground">
                          {t('scanWithWhatsApp')}
                        </p>
                        {qrExpiry > 0 && (
                          <p className="text-xs text-muted-foreground mt-1">
                            {t('expiresIn', { seconds: qrExpiry })}
                          </p>
                        )}
                      </div>
                      <Button
                        type="button"
                        variant="outline"
                        size="sm"
                        onClick={refreshQR}
                      >
                        <RefreshCw className="h-4 w-4 mr-2" />
                        {t('refreshQr')}
                      </Button>
                    </div>
                  </div>
                ) : connectionStatus === 'qr_pending' && pairCode ? (
                  <div className="space-y-4">
                    {/* Pair Code Display */}
                    <div className="flex flex-col items-center space-y-4">
                      <div className="bg-muted p-6 rounded-lg">
                        <p className="text-3xl font-mono tracking-wider">{pairCode}</p>
                      </div>
                      <div className="text-center">
                        <p className="text-sm text-muted-foreground">
                          {t('enterPairCode')}
                        </p>
                        {qrExpiry > 0 && (
                          <p className="text-xs text-muted-foreground mt-1">
                            {t('expiresIn', { seconds: `${Math.floor(qrExpiry / 60)}:${(qrExpiry % 60).toString().padStart(2, '0')}` })}
                          </p>
                        )}
                      </div>
                    </div>
                  </div>
                ) : connectionStatus === 'passkey_pending' ? (
                  <div className="space-y-4">
                    {/* Passkey resolution (account is passkey-locked) */}
                    <div className="flex flex-col items-center space-y-4 text-center">
                      <div className="rounded-full bg-primary/10 p-4">
                        <KeyRound className="h-8 w-8 text-primary" />
                      </div>
                      <div>
                        <p className="text-sm font-medium">{t('passkeyRequiredTitle')}</p>
                        <p className="mt-1 text-sm text-muted-foreground">
                          {t('passkeyRequiredDesc')}
                        </p>
                      </div>
                      {passkeyError && (
                        <Alert variant="destructive" className="text-left">
                          <AlertCircle className="h-4 w-4" />
                          <AlertDescription>{passkeyError}</AlertDescription>
                        </Alert>
                      )}
                      <Button type="button" onClick={resolvePasskey} disabled={passkeyBusy} className="w-full">
                        {passkeyBusy ? (
                          <>
                            <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                            {t('passkeyWaiting')}
                          </>
                        ) : (
                          <>
                            <KeyRound className="mr-2 h-4 w-4" />
                            {t('passkeyAuthorize')}
                          </>
                        )}
                      </Button>
                      <p className="text-xs text-muted-foreground">{t('passkeyExtensionHint')}</p>
                    </div>
                  </div>
                ) : (
                  <div className="space-y-4">
                    {/* Login Options */}
                    {isEditing && (
                      <>
                        <Button
                          type="button"
                          onClick={() => startQRLogin(channel.id)}
                          disabled={connectionStatus === 'connecting'}
                          className="w-full"
                        >
                          {connectionStatus === 'connecting' ? (
                            <>
                              <Loader2 className="h-4 w-4 mr-2 animate-spin" />
                              {t('connecting')}...
                            </>
                          ) : (
                            <>
                              <QrCode className="h-4 w-4 mr-2" />
                              {t('connectWithQr')}
                            </>
                          )}
                        </Button>

                        <div className="relative">
                          <div className="absolute inset-0 flex items-center">
                            <span className="w-full border-t" />
                          </div>
                          <div className="relative flex justify-center text-xs uppercase">
                            <span className="bg-background px-2 text-muted-foreground">
                              {t('orUsePhoneNumber')}
                            </span>
                          </div>
                        </div>

                        <FormField
                          control={form.control}
                          name="phone_number"
                          render={({ field }) => (
                            <FormItem>
                              <FormControl>
                                <Input
                                  placeholder={t('phoneNumberPlaceholder')}
                                  {...field}
                                />
                              </FormControl>
                              <FormDescription>
                                {t('includeCountryCode')}
                              </FormDescription>
                            </FormItem>
                          )}
                        />

                        <Button
                          type="button"
                          variant="outline"
                          onClick={startPairCodeLogin}
                          disabled={connectionStatus === 'connecting'}
                          className="w-full"
                        >
                          <Smartphone className="h-4 w-4 mr-2" />
                          {t('getPairCode')}
                        </Button>
                      </>
                    )}

                    {!isEditing && (
                      <Alert>
                        <AlertCircle className="h-4 w-4" />
                        <AlertTitle>{t('saveFirst')}</AlertTitle>
                        <AlertDescription>
                          {t('saveFirstDesc')}
                        </AlertDescription>
                      </Alert>
                    )}
                  </div>
                )}
              </CardContent>
            </Card>

            {/* Setup Guide */}
            <Card>
              <CardHeader>
                <CardTitle className="text-base">{t('howToConnect')}</CardTitle>
              </CardHeader>
              <CardContent className="space-y-3 text-sm">
                <div className="flex gap-3">
                  <div className="flex h-6 w-6 shrink-0 items-center justify-center rounded-full bg-primary text-primary-foreground text-xs font-medium">
                    1
                  </div>
                  <p>{t('howToStep1')}</p>
                </div>
                <div className="flex gap-3">
                  <div className="flex h-6 w-6 shrink-0 items-center justify-center rounded-full bg-primary text-primary-foreground text-xs font-medium">
                    2
                  </div>
                  <p>{t('howToStep2')}</p>
                </div>
                <div className="flex gap-3">
                  <div className="flex h-6 w-6 shrink-0 items-center justify-center rounded-full bg-primary text-primary-foreground text-xs font-medium">
                    3
                  </div>
                  <p>{t('howToStep3')}</p>
                </div>
                <div className="flex gap-3">
                  <div className="flex h-6 w-6 shrink-0 items-center justify-center rounded-full bg-primary text-primary-foreground text-xs font-medium">
                    4
                  </div>
                  <p>{t('howToStep4')}</p>
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

/**
 * WhatsApp Unofficial Config Dialog
 */
export function WhatsAppUnofficialConfigDialog({
  channel,
  trigger,
  onSuccess,
}: {
  channel?: Channel
  trigger: React.ReactNode
  onSuccess?: (channel: Channel) => void
}) {
  const [open, setOpen] = useState(false)
  const tChannels = useTranslations('channels')

  return (
    <Dialog open={open} onOpenChange={setOpen}>
      <DialogTrigger asChild>{trigger}</DialogTrigger>
      <DialogContent className="max-w-2xl max-h-[90vh] overflow-auto">
        <DialogHeader>
          <DialogTitle>
            {channel
              ? tChannels('configureChannel', { channel: 'WhatsApp' })
              : tChannels('addChannelType', { channel: 'WhatsApp' })}
          </DialogTitle>
          <DialogDescription>
            {channel
              ? tChannels('updateSettings', { channel: 'WhatsApp' })
              : tChannels('setupNewChannel', { channel: 'WhatsApp' })}
          </DialogDescription>
        </DialogHeader>
        <WhatsAppUnofficialConfig
          channel={channel}
          onSuccess={(ch) => {
            setOpen(false)
            onSuccess?.(ch)
          }}
          onCancel={() => setOpen(false)}
        />
      </DialogContent>
    </Dialog>
  )
}

export default WhatsAppUnofficialConfig
