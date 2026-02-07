'use client'

import { useState } from 'react'
import { useForm } from 'react-hook-form'
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
const smsConfigSchema = z.object({
  name: z.string().min(1, 'Channel name is required'),
  account_sid: z.string().min(1, 'Account SID is required'),
  auth_token: z.string().min(1, 'Auth token is required'),
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
    message: 'Phone number or Messaging Service SID is required',
    path: ['phone_number'],
  }
)

type SMSConfigForm = z.infer<typeof smsConfigSchema>

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
  const [isSubmitting, setIsSubmitting] = useState(false)
  const [showAuthToken, setShowAuthToken] = useState(false)
  const [testStatus, setTestStatus] = useState<'idle' | 'testing' | 'success' | 'error'>('idle')

  const isEditing = !!channel

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
    : 'Will be generated after creation'

  const statusCallbackUrl = channel
    ? `${window.location.origin}/api/v1/webhooks/twilio/${channel.id}`
    : 'Will be generated after creation'

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
        title: isEditing ? 'Channel updated' : 'Channel created',
        description: `SMS channel "${data.name}" has been ${isEditing ? 'updated' : 'created'} successfully.`,
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

  const testConnection = async () => {
    const values = form.getValues()
    if (!values.account_sid || !values.auth_token) {
      toast({
        title: 'Missing credentials',
        description: 'Please enter Account SID and Auth Token first',
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
        title: 'Connection successful',
        description: 'Twilio credentials are valid',
      })
    } catch {
      setTestStatus('error')
      toast({
        title: 'Connection failed',
        description: 'Could not connect to Twilio API. Please check your credentials.',
        variant: 'error',
      })
    }
  }

  const copyToClipboard = (text: string, label: string) => {
    navigator.clipboard.writeText(text)
    toast({
      title: 'Copied',
      description: `${label} copied to clipboard`,
    })
  }

  return (
    <Form {...form}>
      <form onSubmit={form.handleSubmit(onSubmit)} className="flex flex-col h-full">
        <div className="flex-1 space-y-6">
        <Tabs defaultValue="credentials" className="w-full">
          <TabsList className="grid w-full grid-cols-3">
            <TabsTrigger value="credentials">Credentials</TabsTrigger>
            <TabsTrigger value="webhook">Webhook</TabsTrigger>
            <TabsTrigger value="setup">Setup Guide</TabsTrigger>
          </TabsList>

          <TabsContent value="credentials" className="space-y-4 mt-4">
            {/* Channel Name */}
            <FormField
              control={form.control}
              name="name"
              render={({ field }) => (
                <FormItem>
                  <FormLabel>Channel Name</FormLabel>
                  <FormControl>
                    <Input placeholder="My SMS Channel" {...field} />
                  </FormControl>
                  <FormDescription>
                    A friendly name to identify this channel
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
                  <FormLabel>Account SID</FormLabel>
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
                    Found on your Twilio Console dashboard
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
                  <FormLabel>Auth Token</FormLabel>
                  <FormControl>
                    <div className="relative">
                      <Input
                        type={showAuthToken ? 'text' : 'password'}
                        placeholder={isEditing ? '••••••••••••••••' : 'Enter auth token'}
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
                    Auth Token from your Twilio Console
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
                  <FormLabel>Sender Type</FormLabel>
                  <FormControl>
                    <RadioGroup
                      onValueChange={field.onChange}
                      defaultValue={field.value}
                      className="flex flex-col space-y-1"
                    >
                      <div className="flex items-center space-x-3">
                        <RadioGroupItem value="phone_number" id="phone_number" />
                        <Label htmlFor="phone_number" className="font-normal">
                          Phone Number
                        </Label>
                      </div>
                      <div className="flex items-center space-x-3">
                        <RadioGroupItem value="messaging_service" id="messaging_service" />
                        <Label htmlFor="messaging_service" className="font-normal">
                          Messaging Service
                        </Label>
                      </div>
                    </RadioGroup>
                  </FormControl>
                  <FormDescription>
                    Use a Messaging Service for better deliverability and A2P 10DLC compliance
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
                    <FormLabel>Phone Number</FormLabel>
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
                      Your Twilio phone number in E.164 format
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
                    <FormLabel>Messaging Service SID</FormLabel>
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
                      Messaging Service SID from Twilio Console
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
                    Testing...
                  </>
                ) : testStatus === 'success' ? (
                  <>
                    <CheckCircle2 className="h-4 w-4 mr-2 text-green-500" />
                    Connection Valid
                  </>
                ) : testStatus === 'error' ? (
                  <>
                    <AlertCircle className="h-4 w-4 mr-2 text-red-500" />
                    Test Failed
                  </>
                ) : (
                  'Test Connection'
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
                  Incoming Messages Webhook
                </CardTitle>
                <CardDescription>
                  Configure this URL in your Twilio phone number settings
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

            {/* Status Callback URL */}
            <Card>
              <CardHeader>
                <CardTitle className="text-base flex items-center gap-2">
                  <AlertCircle className="h-4 w-4" />
                  Status Callback URL
                </CardTitle>
                <CardDescription>
                  For delivery status updates (optional but recommended)
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
                      onClick={() => copyToClipboard(statusCallbackUrl, 'Status Callback URL')}
                    >
                      <Copy className="h-4 w-4" />
                    </Button>
                  )}
                </div>
              </CardContent>
            </Card>

            <Alert>
              <AlertCircle className="h-4 w-4" />
              <AlertTitle>Webhook Configuration</AlertTitle>
              <AlertDescription>
                Configure the webhook URL in your Twilio Console under Phone Numbers &gt;
                Manage Numbers &gt; Active Numbers &gt; [Your Number] &gt; Messaging section.
              </AlertDescription>
            </Alert>
          </TabsContent>

          <TabsContent value="setup" className="space-y-4 mt-4">
            <Card>
              <CardHeader>
                <CardTitle>Setup Guide</CardTitle>
                <CardDescription>
                  Follow these steps to configure Twilio SMS
                </CardDescription>
              </CardHeader>
              <CardContent className="space-y-4">
                <div className="space-y-4">
                  <SetupStep
                    number={1}
                    title="Create Twilio Account"
                    description="Sign up for a Twilio account if you don't have one"
                  >
                    <Button variant="outline" size="sm" asChild>
                      <a
                        href="https://www.twilio.com/console"
                        target="_blank"
                        rel="noopener noreferrer"
                      >
                        Open Twilio Console
                        <ExternalLink className="h-3 w-3 ml-2" />
                      </a>
                    </Button>
                  </SetupStep>

                  <Separator />

                  <SetupStep
                    number={2}
                    title="Get Account Credentials"
                    description="Find your Account SID and Auth Token on the Console Dashboard"
                  />

                  <Separator />

                  <SetupStep
                    number={3}
                    title="Get a Phone Number"
                    description="Purchase a phone number or use a Messaging Service for sending SMS"
                  />

                  <Separator />

                  <SetupStep
                    number={4}
                    title="Configure Webhook (After Channel Creation)"
                    description="Configure the incoming message webhook URL in your phone number settings"
                  >
                    <div className="text-sm text-muted-foreground">
                      <p>Navigate to: Phone Numbers &gt; Manage &gt; Active Numbers</p>
                      <p>Set &quot;A MESSAGE COMES IN&quot; to the Webhook URL</p>
                    </div>
                  </SetupStep>

                  <Separator />

                  <SetupStep
                    number={5}
                    title="Test the Integration"
                    description="Send a test SMS to your Twilio number to verify the setup"
                  />
                </div>
              </CardContent>
            </Card>

            <Alert>
              <Phone className="h-4 w-4" />
              <AlertTitle>A2P 10DLC Compliance (US)</AlertTitle>
              <AlertDescription>
                If you&apos;re sending SMS to US numbers, consider using a Messaging Service
                with A2P 10DLC registration for better deliverability and compliance.
              </AlertDescription>
            </Alert>
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

  return (
    <Dialog open={open} onOpenChange={setOpen}>
      <DialogTrigger asChild>{trigger}</DialogTrigger>
      <DialogContent className="max-w-2xl max-h-[90vh] overflow-auto">
        <DialogHeader>
          <DialogTitle>
            {channel ? 'Configure SMS Channel' : 'Add SMS Channel'}
          </DialogTitle>
          <DialogDescription>
            Connect your Twilio account for SMS messaging
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
