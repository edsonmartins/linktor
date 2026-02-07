'use client'

import { useState, useEffect, useRef } from 'react'
import { useForm } from 'react-hook-form'
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
const whatsappConfigSchema = z.object({
  name: z.string().min(1, 'Channel name is required'),
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
    resolver: zodResolver(whatsappConfigSchema),
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
        title: isEditing ? 'Channel updated' : 'Channel created',
        description: `WhatsApp channel "${data.name}" has been ${isEditing ? 'updated' : 'created'} successfully.`,
      })

      onSuccess?.(result)
    } catch (error) {
      toast({
        title: 'Error',
        description: error instanceof Error ? error.message : 'Failed to save channel',
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
      const response = await api.post<{ qr_code: string; expires_in: number }>(
        `/channels/${channelId}/whatsapp/login`
      )

      if (response.qr_code) {
        setConnectionStatus('qr_pending')
        setQrCode(response.qr_code)
        setQrExpiry(response.expires_in || 60)
      }
    } catch (error) {
      setConnectionStatus('disconnected')
      toast({
        title: 'Error',
        description: 'Failed to start QR login',
        variant: 'error',
      })
    }
  }

  const startPairCodeLogin = async () => {
    const phoneNumber = form.getValues('phone_number')
    if (!phoneNumber) {
      toast({
        title: 'Phone number required',
        description: 'Please enter your phone number to use pair code login',
        variant: 'error',
      })
      return
    }

    setConnectionStatus('connecting')
    setPairCode(null)
    setQrCode(null)

    try {
      const response = await api.post<{ code: string; expires_in: number }>(
        `/channels/${channel?.id}/whatsapp/pair`,
        { phone_number: phoneNumber }
      )

      setPairCode(response.code)
      setQrExpiry(response.expires_in || 300)
      setConnectionStatus('qr_pending')
    } catch (error) {
      setConnectionStatus('disconnected')
      toast({
        title: 'Error',
        description: 'Failed to get pair code',
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
        title: 'Disconnected',
        description: 'WhatsApp has been disconnected',
      })
    } catch (error) {
      toast({
        title: 'Error',
        description: 'Failed to disconnect',
        variant: 'error',
      })
    }
  }

  const getStatusBadge = () => {
    switch (connectionStatus) {
      case 'connected':
        return <Badge variant="success" className="gap-1"><CheckCircle2 className="h-3 w-3" /> Connected</Badge>
      case 'connecting':
        return <Badge variant="secondary" className="gap-1"><Loader2 className="h-3 w-3 animate-spin" /> Connecting</Badge>
      case 'qr_pending':
        return <Badge variant="warning" className="gap-1"><QrCode className="h-3 w-3" /> Scan QR Code</Badge>
      case 'logged_out':
        return <Badge variant="error" className="gap-1"><AlertCircle className="h-3 w-3" /> Logged Out</Badge>
      default:
        return <Badge variant="secondary" className="gap-1">Disconnected</Badge>
    }
  }

  return (
    <Form {...form}>
      <form onSubmit={form.handleSubmit(onSubmit)} className="flex flex-col h-full">
        <div className="flex-1 space-y-6">
        <Tabs defaultValue="setup" className="w-full">
          <TabsList className="grid w-full grid-cols-2">
            <TabsTrigger value="setup">Setup</TabsTrigger>
            <TabsTrigger value="connection">Connection</TabsTrigger>
          </TabsList>

          <TabsContent value="setup" className="space-y-4 mt-4">
            {/* Channel Name */}
            <FormField
              control={form.control}
              name="name"
              render={({ field }) => (
                <FormItem>
                  <FormLabel>Channel Name</FormLabel>
                  <FormControl>
                    <Input placeholder="My WhatsApp" {...field} />
                  </FormControl>
                  <FormDescription>
                    A friendly name to identify this channel
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
                  <FormLabel>Device Name</FormLabel>
                  <FormControl>
                    <Input placeholder="Linktor" {...field} />
                  </FormControl>
                  <FormDescription>
                    Name shown in WhatsApp linked devices
                  </FormDescription>
                  <FormMessage />
                </FormItem>
              )}
            />

            <Alert>
              <Smartphone className="h-4 w-4" />
              <AlertTitle>Multi-Device Support</AlertTitle>
              <AlertDescription>
                This connection uses WhatsApp multi-device protocol. Your phone doesn't
                need to stay online after initial setup.
              </AlertDescription>
            </Alert>
          </TabsContent>

          <TabsContent value="connection" className="space-y-4 mt-4">
            {/* Connection Status */}
            <Card>
              <CardHeader>
                <div className="flex items-center justify-between">
                  <CardTitle className="text-base">Connection Status</CardTitle>
                  {getStatusBadge()}
                </div>
                <CardDescription>
                  {connectionStatus === 'connected' && deviceInfo
                    ? `Connected as ${deviceInfo.phone_number || deviceInfo.jid}`
                    : 'Connect your WhatsApp account to receive messages'}
                </CardDescription>
              </CardHeader>
              <CardContent>
                {connectionStatus === 'connected' ? (
                  <div className="space-y-4">
                    {deviceInfo && (
                      <div className="bg-muted p-4 rounded-lg space-y-2">
                        <div className="flex justify-between text-sm">
                          <span className="text-muted-foreground">Phone:</span>
                          <span>{deviceInfo.phone_number || deviceInfo.jid}</span>
                        </div>
                        <div className="flex justify-between text-sm">
                          <span className="text-muted-foreground">Device:</span>
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
                      Disconnect
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
                          Scan with WhatsApp to connect
                        </p>
                        {qrExpiry > 0 && (
                          <p className="text-xs text-muted-foreground mt-1">
                            Expires in {qrExpiry}s
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
                        Refresh QR
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
                          Enter this code in WhatsApp &gt; Linked Devices
                        </p>
                        {qrExpiry > 0 && (
                          <p className="text-xs text-muted-foreground mt-1">
                            Expires in {Math.floor(qrExpiry / 60)}:{(qrExpiry % 60).toString().padStart(2, '0')}
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
                              Connecting...
                            </>
                          ) : (
                            <>
                              <QrCode className="h-4 w-4 mr-2" />
                              Connect with QR Code
                            </>
                          )}
                        </Button>

                        <div className="relative">
                          <div className="absolute inset-0 flex items-center">
                            <span className="w-full border-t" />
                          </div>
                          <div className="relative flex justify-center text-xs uppercase">
                            <span className="bg-background px-2 text-muted-foreground">
                              Or use phone number
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
                                  placeholder="+55 11 99999-9999"
                                  {...field}
                                />
                              </FormControl>
                              <FormDescription>
                                Include country code (e.g., +55 for Brazil)
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
                          Get Pair Code
                        </Button>
                      </>
                    )}

                    {!isEditing && (
                      <Alert>
                        <AlertCircle className="h-4 w-4" />
                        <AlertTitle>Save First</AlertTitle>
                        <AlertDescription>
                          Save the channel configuration first, then connect your WhatsApp.
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
                <CardTitle className="text-base">How to Connect</CardTitle>
              </CardHeader>
              <CardContent className="space-y-3 text-sm">
                <div className="flex gap-3">
                  <div className="flex h-6 w-6 shrink-0 items-center justify-center rounded-full bg-primary text-primary-foreground text-xs font-medium">
                    1
                  </div>
                  <p>Open WhatsApp on your phone</p>
                </div>
                <div className="flex gap-3">
                  <div className="flex h-6 w-6 shrink-0 items-center justify-center rounded-full bg-primary text-primary-foreground text-xs font-medium">
                    2
                  </div>
                  <p>Go to Settings &gt; Linked Devices</p>
                </div>
                <div className="flex gap-3">
                  <div className="flex h-6 w-6 shrink-0 items-center justify-center rounded-full bg-primary text-primary-foreground text-xs font-medium">
                    3
                  </div>
                  <p>Tap "Link a Device"</p>
                </div>
                <div className="flex gap-3">
                  <div className="flex h-6 w-6 shrink-0 items-center justify-center rounded-full bg-primary text-primary-foreground text-xs font-medium">
                    4
                  </div>
                  <p>Scan the QR code or enter the pair code</p>
                </div>
              </CardContent>
            </Card>
          </TabsContent>
        </Tabs>
        </div>

        <div className="sticky bottom-0 flex justify-end gap-3 pt-4 pb-2 mt-4 border-t bg-background">
          {onCancel && (
            <Button type="button" variant="outline" onClick={onCancel}>
              Cancel
            </Button>
          )}
          <Button type="submit" disabled={isSubmitting}>
            {isSubmitting ? (
              <>
                <Loader2 className="h-4 w-4 mr-2 animate-spin" />
                Saving...
              </>
            ) : isEditing ? (
              'Update Channel'
            ) : (
              'Create Channel'
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

  return (
    <Dialog open={open} onOpenChange={setOpen}>
      <DialogTrigger asChild>{trigger}</DialogTrigger>
      <DialogContent className="max-w-2xl max-h-[90vh] overflow-auto">
        <DialogHeader>
          <DialogTitle>
            {channel ? 'Configure WhatsApp' : 'Add WhatsApp (Unofficial)'}
          </DialogTitle>
          <DialogDescription>
            Connect your WhatsApp account using QR code or phone pairing
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
