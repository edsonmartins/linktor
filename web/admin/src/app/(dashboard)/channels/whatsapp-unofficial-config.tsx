'use client'

import { useState, useEffect, useRef } from 'react'
import { useForm } from 'react-hook-form'
import { useTranslations } from 'next-intl'
import { zodResolver } from '@hookform/resolvers/zod'
import { z } from 'zod'
import {
  AlertCircle,
  CheckCircle2,
  Loader2,
  QrCode,
  RefreshCw,
  Smartphone,
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
import type { Channel } from '@/types'

/**
 * WhatsApp Unofficial Configuration Schema
 */
const createSchema = (t: (key: string) => string) => z.object({
  name: z.string().min(1, t('channelNameRequired')),
  device_name: z.string().optional(),
  phone_number: z.string().optional(),
})

type WhatsAppConfigForm = z.infer<typeof whatsappConfigSchema>

interface WhatsAppUnofficialConfigProps {
  channel?: Channel
  onSuccess?: (channel: Channel) => void
  onCancel?: () => void
}

type ConnectionStatus = 'disconnected' | 'connecting' | 'qr_pending' | 'connected' | 'logged_out'

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
  const wsRef = useRef<WebSocket | null>(null)

  const isEditing = !!channel

  const form = useForm<WhatsAppConfigForm>({
    resolver: zodResolver(createSchema(tCommon)),
    defaultValues: {
      name: channel?.name || '',
      device_name: (channel?.config?.device_name as string) || 'Linktor',
      phone_number: '',
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

  const onSubmit = async (data: WhatsAppConfigForm) => {
    setIsSubmitting(true)
    try {
      const payload = {
        name: data.name,
        type: 'whatsapp',
        config: {
          device_name: data.device_name,
        },
        credentials: {},
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

    try {
      const response = await api.post<{ channel: Channel; qr_code?: string; expires_in?: number }>(
        `/channels/${channelId}/connect`
      )

      if (response.qr_code) {
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
      // TODO: Backend needs to implement pair code support
      // For now, use connect endpoint with phone_number
      const response = await api.post<{ channel: Channel; code?: string; expires_in?: number }>(
        `/channels/${channel?.id}/connect`,
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
      await api.post(`/channels/${channel.id}/whatsapp/logout`)
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
