'use client'

import { useState } from 'react'
import { useForm } from 'react-hook-form'
import { useTranslations } from 'next-intl'
import { zodResolver } from '@hookform/resolvers/zod'
import { z } from 'zod'
import {
  AlertCircle,
  CheckCircle2,
  Copy,
  ExternalLink,
  Eye,
  EyeOff,
  Loader2,
  Phone,
  Shield,
  Smartphone,
  Webhook,
  Zap,
} from 'lucide-react'
import { WhatsAppEmbeddedSignup } from './whatsapp-embedded-signup'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import {
  Card,
  CardContent,
  CardDescription,
  CardFooter,
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
  DialogFooter,
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
 * WhatsApp Official Configuration Schema
 */
const whatsappConfigSchema = z.object({
  name: z.string().min(1, 'Channel name is required'),
  access_token: z.string().min(1, 'Access token is required'),
  phone_number_id: z.string().min(1, 'Phone number ID is required'),
  business_id: z.string().optional(), // Optional - not used in embedded signup flow
  verify_token: z.string().min(1, 'Verify token is required'),
  webhook_secret: z.string().optional(),
  api_version: z.string().min(1),
})

type WhatsAppConfigForm = z.infer<typeof whatsappConfigSchema>

interface WhatsAppConfigProps {
  channel?: Channel
  onSuccess?: (channel: Channel) => void
  onCancel?: () => void
}

/**
 * WhatsApp Official Channel Configuration Component
 */
export function WhatsAppConfig({
  channel,
  onSuccess,
  onCancel,
}: WhatsAppConfigProps) {
  const t = useTranslations('channels.config')
  const tCommon = useTranslations('common')
  const { toast } = useToast()
  const [isSubmitting, setIsSubmitting] = useState(false)
  const [showAccessToken, setShowAccessToken] = useState(false)
  const [showWebhookSecret, setShowWebhookSecret] = useState(false)
  const [testStatus, setTestStatus] = useState<'idle' | 'testing' | 'success' | 'error'>('idle')

  const isEditing = !!channel

  const form = useForm<WhatsAppConfigForm>({
    resolver: zodResolver(whatsappConfigSchema),
    defaultValues: {
      name: channel?.name || '',
      access_token: '',
      phone_number_id: (channel?.config?.phone_number_id as string) || '',
      business_id: (channel?.config?.business_id as string) || '',
      verify_token: (channel?.config?.verify_token as string) || generateVerifyToken(),
      webhook_secret: '',
      api_version: (channel?.config?.api_version as string) || 'v21.0',
    },
  })

  const webhookUrl = channel
    ? `${window.location.origin}/api/v1/webhooks/whatsapp/${channel.id}`
    : t('willBeGenerated')

  const onSubmit = async (data: WhatsAppConfigForm) => {
    setIsSubmitting(true)
    try {
      const payload = {
        name: data.name,
        type: 'whatsapp_official',
        config: {
          phone_number_id: data.phone_number_id,
          business_id: data.business_id,
          verify_token: data.verify_token,
          api_version: data.api_version,
        },
        credentials: {
          access_token: data.access_token,
          webhook_secret: data.webhook_secret,
        },
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
    if (!values.access_token || !values.phone_number_id) {
      toast({
        title: t('missingCredentials'),
        description: t('enterCredentialsFirst'),
        variant: 'error',
      })
      return
    }

    setTestStatus('testing')
    try {
      await api.post('/channels/test-whatsapp', {
        access_token: values.access_token,
        phone_number_id: values.phone_number_id,
        api_version: values.api_version,
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

  const copyToClipboard = (text: string, label: string) => {
    navigator.clipboard.writeText(text)
    toast({
      title: t('copied'),
      description: t('copiedToClipboard', { label }),
    })
  }

  return (
    <Form {...form}>
      <form onSubmit={form.handleSubmit(onSubmit)} className="flex flex-col h-full">
        <div className="flex-1 space-y-6">
        <Tabs defaultValue={isEditing ? "credentials" : "embedded"} className="w-full">
          <TabsList className="grid w-full grid-cols-4">
            <TabsTrigger value="embedded" className="flex items-center gap-1.5">
              <Zap className="h-3.5 w-3.5" />
              {t('embeddedSignup')}
            </TabsTrigger>
            <TabsTrigger value="credentials" className="flex items-center gap-1.5">
              <Shield className="h-3.5 w-3.5" />
              {t('manualSetup')}
            </TabsTrigger>
            <TabsTrigger value="webhook">{t('webhook')}</TabsTrigger>
            <TabsTrigger value="setup">{t('setupGuide')}</TabsTrigger>
          </TabsList>

          {/* Embedded Signup Tab - Quick setup via OAuth */}
          <TabsContent value="embedded" className="space-y-4 mt-4">
            <Card>
              <CardHeader>
                <CardTitle className="flex items-center gap-2">
                  <Smartphone className="h-5 w-5" />
                  {t('connectExistingNumber')}
                </CardTitle>
                <CardDescription>
                  {t('connectExistingNumberDesc')}
                </CardDescription>
              </CardHeader>
              <CardContent>
                <WhatsAppEmbeddedSignup
                  onSuccess={(channel) => {
                    onSuccess?.(channel)
                  }}
                />
              </CardContent>
            </Card>

            <Alert>
              <Zap className="h-4 w-4" />
              <AlertTitle>{t('coexistenceMode')}</AlertTitle>
              <AlertDescription>
                {t('coexistenceModeDesc')}
              </AlertDescription>
            </Alert>
          </TabsContent>

          <TabsContent value="credentials" className="space-y-4 mt-4">
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

            {/* Access Token */}
            <FormField
              control={form.control}
              name="access_token"
              render={({ field }) => (
                <FormItem>
                  <FormLabel>{t('accessToken')}</FormLabel>
                  <FormControl>
                    <div className="relative">
                      <Input
                        type={showAccessToken ? 'text' : 'password'}
                        placeholder={isEditing ? '••••••••••••••••' : t('accessTokenPlaceholder')}
                        {...field}
                      />
                      <Button
                        type="button"
                        variant="ghost"
                        size="icon"
                        className="absolute right-0 top-0 h-full"
                        onClick={() => setShowAccessToken(!showAccessToken)}
                      >
                        {showAccessToken ? (
                          <EyeOff className="h-4 w-4" />
                        ) : (
                          <Eye className="h-4 w-4" />
                        )}
                      </Button>
                    </div>
                  </FormControl>
                  <FormDescription>
                    {t('accessTokenDesc')}
                  </FormDescription>
                  <FormMessage />
                </FormItem>
              )}
            />

            {/* Phone Number ID */}
            <FormField
              control={form.control}
              name="phone_number_id"
              render={({ field }) => (
                <FormItem>
                  <FormLabel>{t('phoneNumberId')}</FormLabel>
                  <FormControl>
                    <div className="relative">
                      <Phone className="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-muted-foreground" />
                      <Input
                        className="pl-10"
                        placeholder="123456789012345"
                        {...field}
                      />
                    </div>
                  </FormControl>
                  <FormDescription>
                    {t('phoneNumberIdDesc')}
                  </FormDescription>
                  <FormMessage />
                </FormItem>
              )}
            />

            {/* Business ID */}
            <FormField
              control={form.control}
              name="business_id"
              render={({ field }) => (
                <FormItem>
                  <FormLabel>{t('businessId')}</FormLabel>
                  <FormControl>
                    <Input placeholder="123456789012345" {...field} />
                  </FormControl>
                  <FormDescription>
                    {t('businessIdDesc')}
                  </FormDescription>
                  <FormMessage />
                </FormItem>
              )}
            />

            {/* API Version */}
            <FormField
              control={form.control}
              name="api_version"
              render={({ field }) => (
                <FormItem>
                  <FormLabel>{t('apiVersion')}</FormLabel>
                  <FormControl>
                    <Input placeholder="v21.0" {...field} />
                  </FormControl>
                  <FormDescription>
                    {t('apiVersionDesc')}
                  </FormDescription>
                  <FormMessage />
                </FormItem>
              )}
            />

            {/* Test Connection */}
            <div className="pt-2">
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
                ) : testStatus === 'success' ? (
                  <>
                    <CheckCircle2 className="h-4 w-4 mr-2 text-green-500" />
                    {t('connectionValid')}
                  </>
                ) : testStatus === 'error' ? (
                  <>
                    <AlertCircle className="h-4 w-4 mr-2 text-red-500" />
                    {t('testFailed')}
                  </>
                ) : (
                  t('testConnection')
                )}
              </Button>
            </div>
          </TabsContent>

          <TabsContent value="webhook" className="space-y-4 mt-4">
            {/* Webhook URL */}
            <Card>
              <CardHeader>
                <CardTitle className="text-base flex items-center gap-2">
                  <Webhook className="h-4 w-4" />
                  {t('webhookUrl')}
                </CardTitle>
                <CardDescription>
                  {t('webhookUrlDesc')}
                </CardDescription>
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
                      onClick={() => copyToClipboard(webhookUrl, 'Webhook URL')}
                    >
                      <Copy className="h-4 w-4" />
                    </Button>
                  )}
                </div>
              </CardContent>
            </Card>

            {/* Verify Token */}
            <FormField
              control={form.control}
              name="verify_token"
              render={({ field }) => (
                <FormItem>
                  <FormLabel>{t('verifyToken')}</FormLabel>
                  <FormControl>
                    <div className="flex items-center gap-2">
                      <Input {...field} />
                      <Button
                        type="button"
                        variant="outline"
                        size="icon"
                        onClick={() => copyToClipboard(field.value, t('verifyToken'))}
                      >
                        <Copy className="h-4 w-4" />
                      </Button>
                    </div>
                  </FormControl>
                  <FormDescription>
                    {t('verifyTokenDesc')}
                  </FormDescription>
                  <FormMessage />
                </FormItem>
              )}
            />

            {/* Webhook Secret */}
            <FormField
              control={form.control}
              name="webhook_secret"
              render={({ field }) => (
                <FormItem>
                  <FormLabel>
                    {t('appSecret')}
                    <Badge variant="outline" className="ml-2">{t('optional')}</Badge>
                  </FormLabel>
                  <FormControl>
                    <div className="relative">
                      <Shield className="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-muted-foreground" />
                      <Input
                        type={showWebhookSecret ? 'text' : 'password'}
                        className="pl-10 pr-10"
                        placeholder={isEditing ? '••••••••••••••••' : t('appSecretPlaceholder')}
                        {...field}
                      />
                      <Button
                        type="button"
                        variant="ghost"
                        size="icon"
                        className="absolute right-0 top-0 h-full"
                        onClick={() => setShowWebhookSecret(!showWebhookSecret)}
                      >
                        {showWebhookSecret ? (
                          <EyeOff className="h-4 w-4" />
                        ) : (
                          <Eye className="h-4 w-4" />
                        )}
                      </Button>
                    </div>
                  </FormControl>
                  <FormDescription>
                    {t('appSecretDesc')}
                  </FormDescription>
                  <FormMessage />
                </FormItem>
              )}
            />

            <Alert>
              <Shield className="h-4 w-4" />
              <AlertTitle>{t('securityRecommendation')}</AlertTitle>
              <AlertDescription>
                {t('securityRecommendationDesc')}
              </AlertDescription>
            </Alert>
          </TabsContent>

          <TabsContent value="setup" className="space-y-4 mt-4">
            <Card>
              <CardHeader>
                <CardTitle>{t('setupGuideTitle')}</CardTitle>
                <CardDescription>
                  {t('setupGuideDesc')}
                </CardDescription>
              </CardHeader>
              <CardContent className="space-y-4">
                <div className="space-y-4">
                  <SetupStep
                    number={1}
                    title={t('step1Title')}
                    description={t('step1Desc')}
                  >
                    <Button variant="outline" size="sm" asChild>
                      <a
                        href="https://developers.facebook.com/apps"
                        target="_blank"
                        rel="noopener noreferrer"
                      >
                        {t('openMetaPortal')}
                        <ExternalLink className="h-3 w-3 ml-2" />
                      </a>
                    </Button>
                  </SetupStep>

                  <Separator />

                  <SetupStep
                    number={2}
                    title={t('step2Title')}
                    description={t('step2Desc')}
                  />

                  <Separator />

                  <SetupStep
                    number={3}
                    title={t('step3Title')}
                    description={t('step3Desc')}
                  />

                  <Separator />

                  <SetupStep
                    number={4}
                    title={t('step4Title')}
                    description={t('step4Desc')}
                  >
                    <div className="text-sm text-muted-foreground">
                      <p>{t('step4Details1')}</p>
                      <p>{t('step4Details2')}</p>
                      <p>{t('step4Details3')}</p>
                    </div>
                  </SetupStep>

                  <Separator />

                  <SetupStep
                    number={5}
                    title={t('step5Title')}
                    description={t('step5Desc')}
                  />
                </div>
              </CardContent>
            </Card>

            <Alert>
              <AlertCircle className="h-4 w-4" />
              <AlertTitle>{t('messagingWindow')}</AlertTitle>
              <AlertDescription>
                {t('messagingWindowDesc')}
              </AlertDescription>
            </Alert>
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
 * Setup Step Component
 */
function SetupStep({
  number,
  title,
  description,
  children,
}: {
  number: number
  title: string
  description: string
  children?: React.ReactNode
}) {
  return (
    <div className="flex gap-4">
      <div className="flex h-8 w-8 shrink-0 items-center justify-center rounded-full bg-primary text-primary-foreground text-sm font-medium">
        {number}
      </div>
      <div className="space-y-1">
        <h4 className="font-medium">{title}</h4>
        <p className="text-sm text-muted-foreground">{description}</p>
        {children && <div className="pt-2">{children}</div>}
      </div>
    </div>
  )
}

/**
 * WhatsApp Config Dialog
 */
export function WhatsAppConfigDialog({
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
              ? tChannels('configureChannel', { channel: 'WhatsApp Business' })
              : tChannels('addChannelType', { channel: 'WhatsApp Business' })}
          </DialogTitle>
          <DialogDescription>
            {channel
              ? tChannels('updateSettings', { channel: 'WhatsApp Business' })
              : tChannels('setupNewChannel', { channel: 'WhatsApp Business' })}
          </DialogDescription>
        </DialogHeader>
        <WhatsAppConfig
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

/**
 * Generate a random verify token
 */
function generateVerifyToken(): string {
  const chars = 'ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789'
  let token = ''
  for (let i = 0; i < 32; i++) {
    token += chars.charAt(Math.floor(Math.random() * chars.length))
  }
  return token
}

export default WhatsAppConfig
