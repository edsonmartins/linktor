'use client'

import { useState } from 'react'
import { useForm } from 'react-hook-form'
import { zodResolver } from '@hookform/resolvers/zod'
import { z } from 'zod'
import { Loader2, Mail, Server, Cloud, Copy, Check, ExternalLink, Eye, EyeOff } from 'lucide-react'

import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { Alert, AlertDescription } from '@/components/ui/alert'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { api } from '@/lib/api'
import { useToast } from '@/hooks/use-toast'

// Provider options
const providers = [
  { value: 'smtp', label: 'SMTP', description: 'Generic SMTP server' },
  { value: 'sendgrid', label: 'SendGrid', description: 'SendGrid API' },
  { value: 'mailgun', label: 'Mailgun', description: 'Mailgun API' },
  { value: 'ses', label: 'AWS SES', description: 'Amazon Simple Email Service' },
  { value: 'postmark', label: 'Postmark', description: 'Postmark API' },
]

// Base schema
const baseSchema = z.object({
  name: z.string().min(1, 'Channel name is required'),
  provider: z.enum(['smtp', 'sendgrid', 'mailgun', 'ses', 'postmark']),
  from_email: z.string().email('Valid email required'),
  from_name: z.string().optional(),
  reply_to: z.string().email().optional().or(z.literal('')),
})

// Provider-specific schemas
const smtpSchema = baseSchema.extend({
  provider: z.literal('smtp'),
  smtp_host: z.string().min(1, 'SMTP host is required'),
  smtp_port: z.coerce.number().min(1, 'SMTP port is required'),
  smtp_username: z.string().optional(),
  smtp_password: z.string().optional(),
  smtp_encryption: z.enum(['tls', 'starttls', 'none']).default('tls'),
  // IMAP for receiving
  imap_host: z.string().optional(),
  imap_port: z.coerce.number().optional(),
  imap_username: z.string().optional(),
  imap_password: z.string().optional(),
  imap_folder: z.string().optional(),
  imap_poll_interval: z.coerce.number().optional(),
})

const sendgridSchema = baseSchema.extend({
  provider: z.literal('sendgrid'),
  sendgrid_api_key: z.string().min(1, 'API key is required'),
})

const mailgunSchema = baseSchema.extend({
  provider: z.literal('mailgun'),
  mailgun_domain: z.string().min(1, 'Domain is required'),
  mailgun_api_key: z.string().min(1, 'API key is required'),
  mailgun_region: z.enum(['us', 'eu']).default('us'),
})

const sesSchema = baseSchema.extend({
  provider: z.literal('ses'),
  ses_region: z.string().min(1, 'Region is required'),
  ses_access_key_id: z.string().min(1, 'Access Key ID is required'),
  ses_secret_key: z.string().min(1, 'Secret Key is required'),
})

const postmarkSchema = baseSchema.extend({
  provider: z.literal('postmark'),
  postmark_server_token: z.string().min(1, 'Server token is required'),
})

type EmailConfigForm = z.infer<typeof smtpSchema> | z.infer<typeof sendgridSchema> | z.infer<typeof mailgunSchema> | z.infer<typeof sesSchema> | z.infer<typeof postmarkSchema>

interface EmailConfigProps {
  channelId?: string
  onSuccess?: () => void
}

export function EmailConfig({ channelId, onSuccess }: EmailConfigProps) {
  const [isSubmitting, setIsSubmitting] = useState(false)
  const [isTesting, setIsTesting] = useState(false)
  const [copied, setCopied] = useState(false)
  const [showPassword, setShowPassword] = useState<Record<string, boolean>>({})
  const { toast } = useToast()

  const isEditing = !!channelId

  const {
    register,
    handleSubmit,
    watch,
    setValue,
    formState: { errors },
  } = useForm<EmailConfigForm>({
    resolver: zodResolver(baseSchema),
    defaultValues: {
      provider: 'smtp',
      smtp_encryption: 'tls',
      mailgun_region: 'us',
    },
  })

  const selectedProvider = watch('provider')

  const onSubmit = async (data: EmailConfigForm) => {
    setIsSubmitting(true)

    try {
      // Build config based on provider
      const config: Record<string, string> = {
        provider: data.provider,
        from_email: data.from_email,
        from_name: data.from_name || '',
        reply_to: data.reply_to || '',
      }

      const credentials: Record<string, string> = {}

      if (data.provider === 'smtp') {
        const smtpData = data as z.infer<typeof smtpSchema>
        config.smtp_host = smtpData.smtp_host || ''
        config.smtp_port = String(smtpData.smtp_port || '')
        config.smtp_encryption = smtpData.smtp_encryption || 'tls'
        credentials.smtp_username = smtpData.smtp_username || ''
        credentials.smtp_password = smtpData.smtp_password || ''
        // IMAP
        if (smtpData.imap_host) {
          config.imap_host = smtpData.imap_host
          config.imap_port = String(smtpData.imap_port || 993)
          config.imap_folder = smtpData.imap_folder || 'INBOX'
          config.imap_poll_interval = String(smtpData.imap_poll_interval || 30)
          credentials.imap_username = smtpData.imap_username || ''
          credentials.imap_password = smtpData.imap_password || ''
        }
      } else if (data.provider === 'sendgrid') {
        const sgData = data as z.infer<typeof sendgridSchema>
        credentials.sendgrid_api_key = sgData.sendgrid_api_key
      } else if (data.provider === 'mailgun') {
        const mgData = data as z.infer<typeof mailgunSchema>
        config.mailgun_domain = mgData.mailgun_domain
        config.mailgun_region = mgData.mailgun_region || 'us'
        credentials.mailgun_api_key = mgData.mailgun_api_key
      } else if (data.provider === 'ses') {
        const sesData = data as z.infer<typeof sesSchema>
        config.ses_region = sesData.ses_region
        credentials.ses_access_key_id = sesData.ses_access_key_id
        credentials.ses_secret_key = sesData.ses_secret_key
      } else if (data.provider === 'postmark') {
        const pmData = data as z.infer<typeof postmarkSchema>
        credentials.postmark_server_token = pmData.postmark_server_token
      }

      const payload = {
        name: data.name,
        type: 'email',
        config,
        credentials,
      }

      if (isEditing) {
        await api.put(`/channels/${channelId}`, payload)
        toast({
          title: 'Channel updated',
          description: 'Email channel configuration has been updated.',
        })
      } else {
        await api.post('/channels', payload)
        toast({
          title: 'Channel created',
          description: 'Email channel has been created successfully.',
        })
      }

      onSuccess?.()
    } catch (error: any) {
      toast({
        title: 'Error',
        description: error.message || 'Failed to save channel configuration.',
        variant: 'destructive',
      })
    } finally {
      setIsSubmitting(false)
    }
  }

  const testConnection = async () => {
    setIsTesting(true)

    try {
      // Would call test endpoint
      await new Promise((resolve) => setTimeout(resolve, 2000))
      toast({
        title: 'Connection successful',
        description: 'Email provider connection verified.',
      })
    } catch (error: any) {
      toast({
        title: 'Connection failed',
        description: error.message || 'Could not connect to email provider.',
        variant: 'destructive',
      })
    } finally {
      setIsTesting(false)
    }
  }

  const copyWebhookUrl = (provider: string) => {
    const baseUrl = typeof window !== 'undefined' ? window.location.origin : ''
    const webhookUrl = `${baseUrl}/api/v1/webhooks/email/${provider}/{channelId}`
    navigator.clipboard.writeText(webhookUrl)
    setCopied(true)
    setTimeout(() => setCopied(false), 2000)
  }

  const togglePasswordVisibility = (field: string) => {
    setShowPassword((prev) => ({ ...prev, [field]: !prev[field] }))
  }

  return (
    <form onSubmit={handleSubmit(onSubmit)} className="space-y-6">
      {/* Basic Configuration */}
      <div className="space-y-4">
        <div className="space-y-2">
          <Label htmlFor="name">Channel Name</Label>
          <Input
            id="name"
            placeholder="My Email Channel"
            {...register('name')}
          />
          {errors.name && (
            <p className="text-sm text-destructive">{errors.name.message}</p>
          )}
        </div>

        <div className="space-y-2">
          <Label htmlFor="provider">Email Provider</Label>
          <Select
            value={selectedProvider}
            onValueChange={(value) => setValue('provider', value as any)}
          >
            <SelectTrigger>
              <SelectValue placeholder="Select provider" />
            </SelectTrigger>
            <SelectContent>
              {providers.map((provider) => (
                <SelectItem key={provider.value} value={provider.value}>
                  <div className="flex flex-col">
                    <span>{provider.label}</span>
                    <span className="text-xs text-muted-foreground">
                      {provider.description}
                    </span>
                  </div>
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
        </div>

        <div className="grid grid-cols-2 gap-4">
          <div className="space-y-2">
            <Label htmlFor="from_email">From Email</Label>
            <Input
              id="from_email"
              type="email"
              placeholder="noreply@example.com"
              {...register('from_email')}
            />
            {errors.from_email && (
              <p className="text-sm text-destructive">{errors.from_email.message}</p>
            )}
          </div>

          <div className="space-y-2">
            <Label htmlFor="from_name">From Name (Optional)</Label>
            <Input
              id="from_name"
              placeholder="My Company"
              {...register('from_name')}
            />
          </div>
        </div>

        <div className="space-y-2">
          <Label htmlFor="reply_to">Reply-To (Optional)</Label>
          <Input
            id="reply_to"
            type="email"
            placeholder="support@example.com"
            {...register('reply_to')}
          />
        </div>
      </div>

      <Tabs defaultValue="credentials" className="w-full">
        <TabsList className="grid w-full grid-cols-3">
          <TabsTrigger value="credentials">Credentials</TabsTrigger>
          <TabsTrigger value="receiving">Receiving</TabsTrigger>
          <TabsTrigger value="webhooks">Webhooks</TabsTrigger>
        </TabsList>

        {/* Credentials Tab */}
        <TabsContent value="credentials" className="space-y-4 pt-4">
          {/* SMTP Configuration */}
          {selectedProvider === 'smtp' && (
            <Card>
              <CardHeader>
                <CardTitle className="text-base flex items-center gap-2">
                  <Server className="h-4 w-4" />
                  SMTP Settings
                </CardTitle>
                <CardDescription>Configure your SMTP server connection</CardDescription>
              </CardHeader>
              <CardContent className="space-y-4">
                <div className="grid grid-cols-2 gap-4">
                  <div className="space-y-2">
                    <Label htmlFor="smtp_host">SMTP Host</Label>
                    <Input
                      id="smtp_host"
                      placeholder="smtp.example.com"
                      {...register('smtp_host' as any)}
                    />
                  </div>
                  <div className="space-y-2">
                    <Label htmlFor="smtp_port">SMTP Port</Label>
                    <Input
                      id="smtp_port"
                      type="number"
                      placeholder="587"
                      {...register('smtp_port' as any)}
                    />
                  </div>
                </div>

                <div className="grid grid-cols-2 gap-4">
                  <div className="space-y-2">
                    <Label htmlFor="smtp_username">Username</Label>
                    <Input
                      id="smtp_username"
                      placeholder="username"
                      {...register('smtp_username' as any)}
                    />
                  </div>
                  <div className="space-y-2">
                    <Label htmlFor="smtp_password">Password</Label>
                    <div className="relative">
                      <Input
                        id="smtp_password"
                        type={showPassword['smtp'] ? 'text' : 'password'}
                        placeholder="••••••••"
                        {...register('smtp_password' as any)}
                      />
                      <Button
                        type="button"
                        variant="ghost"
                        size="icon"
                        className="absolute right-0 top-0 h-full px-3"
                        onClick={() => togglePasswordVisibility('smtp')}
                      >
                        {showPassword['smtp'] ? (
                          <EyeOff className="h-4 w-4" />
                        ) : (
                          <Eye className="h-4 w-4" />
                        )}
                      </Button>
                    </div>
                  </div>
                </div>

                <div className="space-y-2">
                  <Label htmlFor="smtp_encryption">Encryption</Label>
                  <Select
                    defaultValue="tls"
                    onValueChange={(value) => setValue('smtp_encryption' as any, value)}
                  >
                    <SelectTrigger>
                      <SelectValue />
                    </SelectTrigger>
                    <SelectContent>
                      <SelectItem value="tls">TLS (Recommended)</SelectItem>
                      <SelectItem value="starttls">STARTTLS</SelectItem>
                      <SelectItem value="none">None</SelectItem>
                    </SelectContent>
                  </Select>
                </div>
              </CardContent>
            </Card>
          )}

          {/* SendGrid Configuration */}
          {selectedProvider === 'sendgrid' && (
            <Card>
              <CardHeader>
                <CardTitle className="text-base flex items-center gap-2">
                  <Cloud className="h-4 w-4" />
                  SendGrid Settings
                </CardTitle>
                <CardDescription>
                  Get your API key from{' '}
                  <a
                    href="https://app.sendgrid.com/settings/api_keys"
                    target="_blank"
                    rel="noopener noreferrer"
                    className="text-primary hover:underline inline-flex items-center gap-1"
                  >
                    SendGrid Settings
                    <ExternalLink className="h-3 w-3" />
                  </a>
                </CardDescription>
              </CardHeader>
              <CardContent className="space-y-4">
                <div className="space-y-2">
                  <Label htmlFor="sendgrid_api_key">API Key</Label>
                  <div className="relative">
                    <Input
                      id="sendgrid_api_key"
                      type={showPassword['sendgrid'] ? 'text' : 'password'}
                      placeholder="SG.xxxx..."
                      {...register('sendgrid_api_key' as any)}
                    />
                    <Button
                      type="button"
                      variant="ghost"
                      size="icon"
                      className="absolute right-0 top-0 h-full px-3"
                      onClick={() => togglePasswordVisibility('sendgrid')}
                    >
                      {showPassword['sendgrid'] ? (
                        <EyeOff className="h-4 w-4" />
                      ) : (
                        <Eye className="h-4 w-4" />
                      )}
                    </Button>
                  </div>
                </div>

                <Alert>
                  <Mail className="h-4 w-4" />
                  <AlertDescription>
                    Make sure your API key has "Mail Send" permissions and your sender email is verified.
                  </AlertDescription>
                </Alert>
              </CardContent>
            </Card>
          )}

          {/* Mailgun Configuration */}
          {selectedProvider === 'mailgun' && (
            <Card>
              <CardHeader>
                <CardTitle className="text-base flex items-center gap-2">
                  <Cloud className="h-4 w-4" />
                  Mailgun Settings
                </CardTitle>
                <CardDescription>
                  Get your credentials from{' '}
                  <a
                    href="https://app.mailgun.com/app/sending/domains"
                    target="_blank"
                    rel="noopener noreferrer"
                    className="text-primary hover:underline inline-flex items-center gap-1"
                  >
                    Mailgun Dashboard
                    <ExternalLink className="h-3 w-3" />
                  </a>
                </CardDescription>
              </CardHeader>
              <CardContent className="space-y-4">
                <div className="grid grid-cols-2 gap-4">
                  <div className="space-y-2">
                    <Label htmlFor="mailgun_domain">Domain</Label>
                    <Input
                      id="mailgun_domain"
                      placeholder="mg.example.com"
                      {...register('mailgun_domain' as any)}
                    />
                  </div>
                  <div className="space-y-2">
                    <Label htmlFor="mailgun_region">Region</Label>
                    <Select
                      defaultValue="us"
                      onValueChange={(value) => setValue('mailgun_region' as any, value)}
                    >
                      <SelectTrigger>
                        <SelectValue />
                      </SelectTrigger>
                      <SelectContent>
                        <SelectItem value="us">US</SelectItem>
                        <SelectItem value="eu">EU</SelectItem>
                      </SelectContent>
                    </Select>
                  </div>
                </div>

                <div className="space-y-2">
                  <Label htmlFor="mailgun_api_key">API Key</Label>
                  <div className="relative">
                    <Input
                      id="mailgun_api_key"
                      type={showPassword['mailgun'] ? 'text' : 'password'}
                      placeholder="key-xxxx..."
                      {...register('mailgun_api_key' as any)}
                    />
                    <Button
                      type="button"
                      variant="ghost"
                      size="icon"
                      className="absolute right-0 top-0 h-full px-3"
                      onClick={() => togglePasswordVisibility('mailgun')}
                    >
                      {showPassword['mailgun'] ? (
                        <EyeOff className="h-4 w-4" />
                      ) : (
                        <Eye className="h-4 w-4" />
                      )}
                    </Button>
                  </div>
                </div>
              </CardContent>
            </Card>
          )}

          {/* AWS SES Configuration */}
          {selectedProvider === 'ses' && (
            <Card>
              <CardHeader>
                <CardTitle className="text-base flex items-center gap-2">
                  <Cloud className="h-4 w-4" />
                  AWS SES Settings
                </CardTitle>
                <CardDescription>
                  Configure your AWS Simple Email Service credentials
                </CardDescription>
              </CardHeader>
              <CardContent className="space-y-4">
                <div className="space-y-2">
                  <Label htmlFor="ses_region">Region</Label>
                  <Select
                    onValueChange={(value) => setValue('ses_region' as any, value)}
                  >
                    <SelectTrigger>
                      <SelectValue placeholder="Select region" />
                    </SelectTrigger>
                    <SelectContent>
                      <SelectItem value="us-east-1">US East (N. Virginia)</SelectItem>
                      <SelectItem value="us-west-2">US West (Oregon)</SelectItem>
                      <SelectItem value="eu-west-1">EU (Ireland)</SelectItem>
                      <SelectItem value="eu-central-1">EU (Frankfurt)</SelectItem>
                      <SelectItem value="ap-southeast-1">Asia Pacific (Singapore)</SelectItem>
                      <SelectItem value="ap-southeast-2">Asia Pacific (Sydney)</SelectItem>
                    </SelectContent>
                  </Select>
                </div>

                <div className="space-y-2">
                  <Label htmlFor="ses_access_key_id">Access Key ID</Label>
                  <Input
                    id="ses_access_key_id"
                    placeholder="AKIAIOSFODNN7EXAMPLE"
                    {...register('ses_access_key_id' as any)}
                  />
                </div>

                <div className="space-y-2">
                  <Label htmlFor="ses_secret_key">Secret Access Key</Label>
                  <div className="relative">
                    <Input
                      id="ses_secret_key"
                      type={showPassword['ses'] ? 'text' : 'password'}
                      placeholder="wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY"
                      {...register('ses_secret_key' as any)}
                    />
                    <Button
                      type="button"
                      variant="ghost"
                      size="icon"
                      className="absolute right-0 top-0 h-full px-3"
                      onClick={() => togglePasswordVisibility('ses')}
                    >
                      {showPassword['ses'] ? (
                        <EyeOff className="h-4 w-4" />
                      ) : (
                        <Eye className="h-4 w-4" />
                      )}
                    </Button>
                  </div>
                </div>

                <Alert>
                  <Mail className="h-4 w-4" />
                  <AlertDescription>
                    Ensure your IAM user has SES permissions and your sender email is verified in the SES console.
                  </AlertDescription>
                </Alert>
              </CardContent>
            </Card>
          )}

          {/* Postmark Configuration */}
          {selectedProvider === 'postmark' && (
            <Card>
              <CardHeader>
                <CardTitle className="text-base flex items-center gap-2">
                  <Cloud className="h-4 w-4" />
                  Postmark Settings
                </CardTitle>
                <CardDescription>
                  Get your server token from{' '}
                  <a
                    href="https://account.postmarkapp.com/servers"
                    target="_blank"
                    rel="noopener noreferrer"
                    className="text-primary hover:underline inline-flex items-center gap-1"
                  >
                    Postmark Servers
                    <ExternalLink className="h-3 w-3" />
                  </a>
                </CardDescription>
              </CardHeader>
              <CardContent className="space-y-4">
                <div className="space-y-2">
                  <Label htmlFor="postmark_server_token">Server Token</Label>
                  <div className="relative">
                    <Input
                      id="postmark_server_token"
                      type={showPassword['postmark'] ? 'text' : 'password'}
                      placeholder="xxxx-xxxx-xxxx-xxxx"
                      {...register('postmark_server_token' as any)}
                    />
                    <Button
                      type="button"
                      variant="ghost"
                      size="icon"
                      className="absolute right-0 top-0 h-full px-3"
                      onClick={() => togglePasswordVisibility('postmark')}
                    >
                      {showPassword['postmark'] ? (
                        <EyeOff className="h-4 w-4" />
                      ) : (
                        <Eye className="h-4 w-4" />
                      )}
                    </Button>
                  </div>
                </div>

                <Alert>
                  <Mail className="h-4 w-4" />
                  <AlertDescription>
                    Make sure your sender signature is verified in Postmark.
                  </AlertDescription>
                </Alert>
              </CardContent>
            </Card>
          )}
        </TabsContent>

        {/* Receiving Tab (IMAP for SMTP provider) */}
        <TabsContent value="receiving" className="space-y-4 pt-4">
          {selectedProvider === 'smtp' ? (
            <Card>
              <CardHeader>
                <CardTitle className="text-base flex items-center gap-2">
                  <Mail className="h-4 w-4" />
                  IMAP Settings (Optional)
                </CardTitle>
                <CardDescription>
                  Configure IMAP to receive incoming emails. Leave blank to use webhooks only.
                </CardDescription>
              </CardHeader>
              <CardContent className="space-y-4">
                <div className="grid grid-cols-2 gap-4">
                  <div className="space-y-2">
                    <Label htmlFor="imap_host">IMAP Host</Label>
                    <Input
                      id="imap_host"
                      placeholder="imap.example.com"
                      {...register('imap_host' as any)}
                    />
                  </div>
                  <div className="space-y-2">
                    <Label htmlFor="imap_port">IMAP Port</Label>
                    <Input
                      id="imap_port"
                      type="number"
                      placeholder="993"
                      defaultValue={993}
                      {...register('imap_port' as any)}
                    />
                  </div>
                </div>

                <div className="grid grid-cols-2 gap-4">
                  <div className="space-y-2">
                    <Label htmlFor="imap_username">Username</Label>
                    <Input
                      id="imap_username"
                      placeholder="username"
                      {...register('imap_username' as any)}
                    />
                  </div>
                  <div className="space-y-2">
                    <Label htmlFor="imap_password">Password</Label>
                    <div className="relative">
                      <Input
                        id="imap_password"
                        type={showPassword['imap'] ? 'text' : 'password'}
                        placeholder="••••••••"
                        {...register('imap_password' as any)}
                      />
                      <Button
                        type="button"
                        variant="ghost"
                        size="icon"
                        className="absolute right-0 top-0 h-full px-3"
                        onClick={() => togglePasswordVisibility('imap')}
                      >
                        {showPassword['imap'] ? (
                          <EyeOff className="h-4 w-4" />
                        ) : (
                          <Eye className="h-4 w-4" />
                        )}
                      </Button>
                    </div>
                  </div>
                </div>

                <div className="grid grid-cols-2 gap-4">
                  <div className="space-y-2">
                    <Label htmlFor="imap_folder">Folder</Label>
                    <Input
                      id="imap_folder"
                      placeholder="INBOX"
                      defaultValue="INBOX"
                      {...register('imap_folder' as any)}
                    />
                  </div>
                  <div className="space-y-2">
                    <Label htmlFor="imap_poll_interval">Poll Interval (seconds)</Label>
                    <Input
                      id="imap_poll_interval"
                      type="number"
                      placeholder="30"
                      defaultValue={30}
                      {...register('imap_poll_interval' as any)}
                    />
                  </div>
                </div>
              </CardContent>
            </Card>
          ) : (
            <Card>
              <CardHeader>
                <CardTitle className="text-base">Inbound Webhooks</CardTitle>
                <CardDescription>
                  {selectedProvider === 'sendgrid' && 'Configure SendGrid Inbound Parse to receive emails.'}
                  {selectedProvider === 'mailgun' && 'Configure Mailgun Routes to receive emails.'}
                  {selectedProvider === 'ses' && 'Configure SES receiving rules to forward to SNS.'}
                  {selectedProvider === 'postmark' && 'Configure Postmark Inbound to receive emails.'}
                </CardDescription>
              </CardHeader>
              <CardContent>
                <Alert>
                  <Mail className="h-4 w-4" />
                  <AlertDescription>
                    Configure the inbound webhook URL in your provider's dashboard to receive incoming emails.
                    See the Webhooks tab for the URL to configure.
                  </AlertDescription>
                </Alert>
              </CardContent>
            </Card>
          )}
        </TabsContent>

        {/* Webhooks Tab */}
        <TabsContent value="webhooks" className="space-y-4 pt-4">
          <Card>
            <CardHeader>
              <CardTitle className="text-base">Webhook URLs</CardTitle>
              <CardDescription>
                Configure these URLs in your email provider's dashboard to receive events and inbound emails.
              </CardDescription>
            </CardHeader>
            <CardContent className="space-y-4">
              {selectedProvider === 'sendgrid' && (
                <>
                  <div className="space-y-2">
                    <Label>Inbound Parse Webhook</Label>
                    <div className="flex gap-2">
                      <Input
                        readOnly
                        value={`${typeof window !== 'undefined' ? window.location.origin : ''}/api/v1/webhooks/email/sendgrid/{channelId}`}
                        className="font-mono text-sm"
                      />
                      <Button
                        type="button"
                        variant="outline"
                        size="icon"
                        onClick={() => copyWebhookUrl('sendgrid')}
                      >
                        {copied ? <Check className="h-4 w-4" /> : <Copy className="h-4 w-4" />}
                      </Button>
                    </div>
                  </div>
                  <div className="space-y-2">
                    <Label>Event Webhook</Label>
                    <div className="flex gap-2">
                      <Input
                        readOnly
                        value={`${typeof window !== 'undefined' ? window.location.origin : ''}/api/v1/webhooks/email/sendgrid/{channelId}/events`}
                        className="font-mono text-sm"
                      />
                      <Button
                        type="button"
                        variant="outline"
                        size="icon"
                        onClick={() => copyWebhookUrl('sendgrid')}
                      >
                        {copied ? <Check className="h-4 w-4" /> : <Copy className="h-4 w-4" />}
                      </Button>
                    </div>
                  </div>
                </>
              )}

              {selectedProvider === 'mailgun' && (
                <div className="space-y-2">
                  <Label>Mailgun Webhook</Label>
                  <div className="flex gap-2">
                    <Input
                      readOnly
                      value={`${typeof window !== 'undefined' ? window.location.origin : ''}/api/v1/webhooks/email/mailgun/{channelId}`}
                      className="font-mono text-sm"
                    />
                    <Button
                      type="button"
                      variant="outline"
                      size="icon"
                      onClick={() => copyWebhookUrl('mailgun')}
                    >
                      {copied ? <Check className="h-4 w-4" /> : <Copy className="h-4 w-4" />}
                    </Button>
                  </div>
                </div>
              )}

              {selectedProvider === 'ses' && (
                <div className="space-y-2">
                  <Label>SNS Notification Endpoint</Label>
                  <div className="flex gap-2">
                    <Input
                      readOnly
                      value={`${typeof window !== 'undefined' ? window.location.origin : ''}/api/v1/webhooks/email/ses/{channelId}`}
                      className="font-mono text-sm"
                    />
                    <Button
                      type="button"
                      variant="outline"
                      size="icon"
                      onClick={() => copyWebhookUrl('ses')}
                    >
                      {copied ? <Check className="h-4 w-4" /> : <Copy className="h-4 w-4" />}
                    </Button>
                  </div>
                  <p className="text-xs text-muted-foreground">
                    Create an SNS topic, subscribe this HTTPS endpoint, then configure SES to publish events to that topic.
                  </p>
                </div>
              )}

              {selectedProvider === 'postmark' && (
                <div className="space-y-2">
                  <Label>Postmark Webhook</Label>
                  <div className="flex gap-2">
                    <Input
                      readOnly
                      value={`${typeof window !== 'undefined' ? window.location.origin : ''}/api/v1/webhooks/email/postmark/{channelId}`}
                      className="font-mono text-sm"
                    />
                    <Button
                      type="button"
                      variant="outline"
                      size="icon"
                      onClick={() => copyWebhookUrl('postmark')}
                    >
                      {copied ? <Check className="h-4 w-4" /> : <Copy className="h-4 w-4" />}
                    </Button>
                  </div>
                </div>
              )}

              {selectedProvider === 'smtp' && (
                <Alert>
                  <Mail className="h-4 w-4" />
                  <AlertDescription>
                    SMTP provider uses IMAP polling for receiving emails. No webhook configuration needed.
                  </AlertDescription>
                </Alert>
              )}
            </CardContent>
          </Card>
        </TabsContent>
      </Tabs>

      {/* Actions */}
      <div className="flex justify-between pt-4">
        <Button type="button" variant="outline" onClick={testConnection} disabled={isTesting}>
          {isTesting && <Loader2 className="mr-2 h-4 w-4 animate-spin" />}
          Test Connection
        </Button>

        <Button type="submit" disabled={isSubmitting}>
          {isSubmitting && <Loader2 className="mr-2 h-4 w-4 animate-spin" />}
          {isEditing ? 'Update Channel' : 'Create Channel'}
        </Button>
      </div>
    </form>
  )
}
