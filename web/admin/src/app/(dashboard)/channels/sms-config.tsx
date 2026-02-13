'use client'

import { useState } from 'react'
import { useForm } from 'react-hook-form'
import { zodResolver } from '@hookform/resolvers/zod'
import { z } from 'zod'
import { useTranslations } from 'next-intl'
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
  Webhook,
  MessageSquare,
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
import { RadioGroup, RadioGroupItem } from '@/components/ui/radio-group'
import { Label } from '@/components/ui/label'
import { useToast } from '@/hooks/use-toast'
import { api } from '@/lib/api'
import type { Channel } from '@/types'

/**
 * SMS/Twilio Configuration Schema
 */
const createSmsConfigSchema = (tCommon: (key: string) => string) => z.object({
  name: z.string().min(1, tCommon('required')),
  account_sid: z.string().min(1, tCommon('required')),
  auth_token: z.string().min(1, tCommon('required')),
  sender_type: z.enum(['phone_number', 'messaging_service']),
  phone_number: z.string().optional(),
  messaging_service_sid: z.string().optional(),
}).refine(
  (data) => {
    if (data.sender_type === 'phone_number') {
      return !!data.phone_number
    }
    if (data.sender_type === 'messaging_service') {
      return !!data.messaging_service_sid
    }
    return false
  },
  {
    message: tCommon('required'),
    path: ['phone_number'],
  }
)

type SMSConfigForm = z.infer<ReturnType<typeof createSmsConfigSchema>>

interface SMSConfigProps {
  channel?: Channel
  onSuccess?: (channel: Channel) => void
  onCancel?: () => void
}

/**
 * SMS/Twilio Channel Configuration Component
 */
export function SMSConfig({
  channel,
  onSuccess,
  onCancel,
}: SMSConfigProps) {
  const { toast } = useToast()
  const t = useTranslations('channels.config')
  const tCommon = useTranslations('common')
  const [isSubmitting, setIsSubmitting] = useState(false)
  const [showAuthToken, setShowAuthToken] = useState(false)
  const [testStatus, setTestStatus] = useState<'idle' | 'testing' | 'success' | 'error'>('idle')

  const isEditing = !!channel
  const smsConfigSchema = createSmsConfigSchema(tCommon)

  const form = useForm<SMSConfigForm>({
    resolver: zodResolver(smsConfigSchema),
    defaultValues: {
      name: channel?.name || '',
      account_sid: (channel?.config?.account_sid as string) || '',
      auth_token: '',
      sender_type: channel?.config?.messaging_service_sid ? 'messaging_service' : 'phone_number',
      phone_number: (channel?.config?.phone_number as string) || '',
      messaging_service_sid: (channel?.config?.messaging_service_sid as string) || '',
    },
  })

  const senderType = form.watch('sender_type')

  const webhookUrl = channel
    ? `${window.location.origin}/api/v1/webhooks/twilio/${channel.id}`
    : t('webhookPending')

  const statusCallbackUrl = channel
    ? `${window.location.origin}/api/v1/webhooks/twilio/${channel.id}`
    : t('webhookPending')

  const onSubmit = async (data: SMSConfigForm) => {
    setIsSubmitting(true)
    try {
      const payload = {
        name: data.name,
        type: 'sms',
        config: {
          account_sid: data.account_sid,
          phone_number: data.sender_type === 'phone_number' ? data.phone_number : undefined,
          messaging_service_sid: data.sender_type === 'messaging_service' ? data.messaging_service_sid : undefined,
        },
        credentials: {
          auth_token: data.auth_token,
        },
      }

      let result: Channel
      if (isEditing) {
        result = await api.put<Channel>(`/channels/${channel.id}`, payload)
      } else {
        result = await api.post<Channel>('/channels', payload)
      }

      toast({
        title: isEditing ? tCommon('updated') : tCommon('created'),
        description: isEditing ? t('channelUpdated') : t('channelCreated'),
      })

      onSuccess?.(result)
    } catch (error) {
      toast({
        title: tCommon('error'),
        description: error instanceof Error ? error.message : t('saveError'),
        variant: 'error',
      })
    } finally {
      setIsSubmitting(false)
    }
  }

  const testConnection = async () => {
    const values = form.getValues()
    if (!values.account_sid || !values.auth_token) {
      toast({
        title: t('missingCredentials'),
        description: t('missingCredentialsDesc'),
        variant: 'error',
      })
      return
    }

    setTestStatus('testing')
    try {
      await api.post('/channels/test-twilio', {
        account_sid: values.account_sid,
        auth_token: values.auth_token,
      })
      setTestStatus('success')
      toast({
        title: t('connectionSuccess'),
        description: t('twilioCredentialsValid'),
      })
    } catch {
      setTestStatus('error')
      toast({
        title: t('connectionFailed'),
        description: t('twilioConnectionError'),
        variant: 'error',
      })
    }
  }

  const copyToClipboard = (text: string) => {
    navigator.clipboard.writeText(text)
    toast({
      title: tCommon('copied'),
      description: t('copiedToClipboard'),
    })
  }

  return (
    <Form {...form}>
      <form onSubmit={form.handleSubmit(onSubmit)} className="flex flex-col h-full">
        <div className="flex-1 space-y-6">
        <Tabs defaultValue="credentials" className="w-full">
          <TabsList className="grid w-full grid-cols-3">
            <TabsTrigger value="credentials">{t('credentials')}</TabsTrigger>
            <TabsTrigger value="webhook">{t('webhook')}</TabsTrigger>
            <TabsTrigger value="setup">{t('setupGuide')}</TabsTrigger>
          </TabsList>

          <TabsContent value="credentials" className="space-y-4 mt-4">
            {/* Channel Name */}
            <FormField
              control={form.control}
              name="name"
              render={({ field }) => (
                <FormItem>
                  <FormLabel>{t('channelName')}</FormLabel>
                  <FormControl>
                    <Input placeholder={t('mySmsChannel')} {...field} />
                  </FormControl>
                  <FormDescription>
                    {t('channelNameDesc')}
                  </FormDescription>
                  <FormMessage />
                </FormItem>
              )}
            />

            {/* Account SID */}
            <FormField
              control={form.control}
              name="account_sid"
              render={({ field }) => (
                <FormItem>
                  <FormLabel>{t('accountSid')}</FormLabel>
                  <FormControl>
                    <div className="relative">
                      <Shield className="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-muted-foreground" />
                      <Input
                        className="pl-10"
                        placeholder="ACxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx"
                        {...field}
                      />
                    </div>
                  </FormControl>
                  <FormDescription>
                    {t('accountSidDesc')}
                  </FormDescription>
                  <FormMessage />
                </FormItem>
              )}
            />

            {/* Auth Token */}
            <FormField
              control={form.control}
              name="auth_token"
              render={({ field }) => (
                <FormItem>
                  <FormLabel>{t('authToken')}</FormLabel>
                  <FormControl>
                    <div className="relative">
                      <Input
                        type={showAuthToken ? 'text' : 'password'}
                        placeholder={isEditing ? '••••••••••••••••' : t('enterAuthToken')}
                        {...field}
                      />
                      <Button
                        type="button"
                        variant="ghost"
                        size="icon"
                        className="absolute right-0 top-0 h-full"
                        onClick={() => setShowAuthToken(!showAuthToken)}
                      >
                        {showAuthToken ? (
                          <EyeOff className="h-4 w-4" />
                        ) : (
                          <Eye className="h-4 w-4" />
                        )}
                      </Button>
                    </div>
                  </FormControl>
                  <FormDescription>
                    {t('authTokenDesc')}
                  </FormDescription>
                  <FormMessage />
                </FormItem>
              )}
            />

            {/* Sender Type */}
            <FormField
              control={form.control}
              name="sender_type"
              render={({ field }) => (
                <FormItem className="space-y-3">
                  <FormLabel>{t('senderType')}</FormLabel>
                  <FormControl>
                    <RadioGroup
                      onValueChange={field.onChange}
                      defaultValue={field.value}
                      className="flex flex-col space-y-1"
                    >
                      <div className="flex items-center space-x-3">
                        <RadioGroupItem value="phone_number" id="phone_number" />
                        <Label htmlFor="phone_number" className="font-normal">
                          {t('phoneNumber')}
                        </Label>
                      </div>
                      <div className="flex items-center space-x-3">
                        <RadioGroupItem value="messaging_service" id="messaging_service" />
                        <Label htmlFor="messaging_service" className="font-normal">
                          {t('messagingService')}
                        </Label>
                      </div>
                    </RadioGroup>
                  </FormControl>
                  <FormDescription>
                    {t('senderTypeDesc')}
                  </FormDescription>
                  <FormMessage />
                </FormItem>
              )}
            />

            {/* Phone Number (conditional) */}
            {senderType === 'phone_number' && (
              <FormField
                control={form.control}
                name="phone_number"
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>{t('phoneNumber')}</FormLabel>
                    <FormControl>
                      <div className="relative">
                        <Phone className="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-muted-foreground" />
                        <Input
                          className="pl-10"
                          placeholder="+1234567890"
                          {...field}
                        />
                      </div>
                    </FormControl>
                    <FormDescription>
                      {t('phoneNumberDesc')}
                    </FormDescription>
                    <FormMessage />
                  </FormItem>
                )}
              />
            )}

            {/* Messaging Service SID (conditional) */}
            {senderType === 'messaging_service' && (
              <FormField
                control={form.control}
                name="messaging_service_sid"
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>{t('messagingServiceSid')}</FormLabel>
                    <FormControl>
                      <div className="relative">
                        <MessageSquare className="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-muted-foreground" />
                        <Input
                          className="pl-10"
                          placeholder="MGxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx"
                          {...field}
                        />
                      </div>
                    </FormControl>
                    <FormDescription>
                      {t('messagingServiceSidDesc')}
                    </FormDescription>
                    <FormMessage />
                  </FormItem>
                )}
              />
            )}

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
            {/* Webhook URL for Incoming Messages */}
            <Card>
              <CardHeader>
                <CardTitle className="text-base flex items-center gap-2">
                  <Webhook className="h-4 w-4" />
                  {t('incomingMessagesWebhook')}
                </CardTitle>
                <CardDescription>
                  {t('configureWebhookInTwilio')}
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
                      onClick={() => copyToClipboard(webhookUrl)}
                    >
                      <Copy className="h-4 w-4" />
                    </Button>
                  )}
                </div>
              </CardContent>
            </Card>

            {/* Status Callback URL */}
            <Card>
              <CardHeader>
                <CardTitle className="text-base flex items-center gap-2">
                  <AlertCircle className="h-4 w-4" />
                  {t('statusCallbackUrl')}
                </CardTitle>
                <CardDescription>
                  {t('statusCallbackDesc')}
                </CardDescription>
              </CardHeader>
              <CardContent>
                <div className="flex items-center gap-2">
                  <code className="flex-1 bg-muted px-3 py-2 rounded text-sm font-mono break-all">
                    {statusCallbackUrl}
                  </code>
                  {channel && (
                    <Button
                      type="button"
                      variant="outline"
                      size="icon"
                      onClick={() => copyToClipboard(statusCallbackUrl)}
                    >
                      <Copy className="h-4 w-4" />
                    </Button>
                  )}
                </div>
              </CardContent>
            </Card>

            <Alert>
              <AlertCircle className="h-4 w-4" />
              <AlertTitle>{t('webhookConfiguration')}</AlertTitle>
              <AlertDescription>
                {t('twilioWebhookInstructions')}
              </AlertDescription>
            </Alert>
          </TabsContent>

          <TabsContent value="setup" className="space-y-4 mt-4">
            <Card>
              <CardHeader>
                <CardTitle>{t('setupGuide')}</CardTitle>
                <CardDescription>
                  {t('twilioSetupDesc')}
                </CardDescription>
              </CardHeader>
              <CardContent className="space-y-4">
                <div className="space-y-4">
                  <SetupStep
                    number={1}
                    title={t('createTwilioAccount')}
                    description={t('createTwilioAccountDesc')}
                  >
                    <Button variant="outline" size="sm" asChild>
                      <a
                        href="https://www.twilio.com/console"
                        target="_blank"
                        rel="noopener noreferrer"
                      >
                        {t('openTwilioConsole')}
                        <ExternalLink className="h-3 w-3 ml-2" />
                      </a>
                    </Button>
                  </SetupStep>

                  <Separator />

                  <SetupStep
                    number={2}
                    title={t('getAccountCredentials')}
                    description={t('getAccountCredentialsDesc')}
                  />

                  <Separator />

                  <SetupStep
                    number={3}
                    title={t('getPhoneNumber')}
                    description={t('getPhoneNumberDesc')}
                  />

                  <Separator />

                  <SetupStep
                    number={4}
                    title={t('configureWebhookStep')}
                    description={t('configureWebhookStepDesc')}
                  >
                    <div className="text-sm text-muted-foreground">
                      <p>{t('webhookNavigation')}</p>
                      <p>{t('webhookSetMessage')}</p>
                    </div>
                  </SetupStep>

                  <Separator />

                  <SetupStep
                    number={5}
                    title={t('testIntegration')}
                    description={t('testIntegrationDesc')}
                  />
                </div>
              </CardContent>
            </Card>

            <Alert>
              <Phone className="h-4 w-4" />
              <AlertTitle>{t('a2p10dlcCompliance')}</AlertTitle>
              <AlertDescription>
                {t('a2p10dlcComplianceDesc')}
              </AlertDescription>
            </Alert>
          </TabsContent>
        </Tabs>
        </div>

        <div className="sticky bottom-0 flex justify-end gap-3 pt-4 pb-2 mt-4 border-t bg-background">
          {onCancel && (
            <Button type="button" variant="outline" onClick={onCancel}>
              {tCommon('cancel')}
            </Button>
          )}
          <Button type="submit" disabled={isSubmitting}>
            {isSubmitting ? (
              <>
                <Loader2 className="h-4 w-4 mr-2 animate-spin" />
                {tCommon('saving')}
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
 * SMS Config Dialog
 */
export function SMSConfigDialog({
  channel,
  trigger,
  onSuccess,
}: {
  channel?: Channel
  trigger: React.ReactNode
  onSuccess?: (channel: Channel) => void
}) {
  const [open, setOpen] = useState(false)
  const t = useTranslations('channels.config')

  return (
    <Dialog open={open} onOpenChange={setOpen}>
      <DialogTrigger asChild>{trigger}</DialogTrigger>
      <DialogContent className="max-w-2xl max-h-[90vh] overflow-auto">
        <DialogHeader>
          <DialogTitle>
            {channel ? t('configureSmsChannel') : t('addSmsChannel')}
          </DialogTitle>
          <DialogDescription>
            {t('connectTwilioAccount')}
          </DialogDescription>
        </DialogHeader>
        <SMSConfig
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

export default SMSConfig
