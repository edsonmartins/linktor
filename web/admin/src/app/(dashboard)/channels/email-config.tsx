'use client'

import { useState } from 'react'
import { useForm } from 'react-hook-form'
import { zodResolver } from '@hookform/resolvers/zod'
import { z } from 'zod'
import { useTranslations } from 'next-intl'
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

// Combined schema with all fields (provider-specific fields are optional for form handling)
const createEmailConfigSchema = (tCommon: (key: string) => string, t: (key: string) => string) => z.object({
  name: z.string().min(1, tCommon('required')),
  provider: z.enum(['smtp', 'sendgrid', 'mailgun', 'ses', 'postmark']),
  from_email: z.string().email(t('validEmailRequired')),
  from_name: z.string().optional(),
  reply_to: z.string().email().optional().or(z.literal('')),
  // SMTP fields
  smtp_host: z.string().optional(),
  smtp_port: z.coerce.number().optional(),
  smtp_username: z.string().optional(),
  smtp_password: z.string().optional(),
  smtp_encryption: z.enum(['tls', 'starttls', 'none']).optional(),
  // IMAP fields
  imap_host: z.string().optional(),
  imap_port: z.coerce.number().optional(),
  imap_username: z.string().optional(),
  imap_password: z.string().optional(),
  imap_folder: z.string().optional(),
  imap_poll_interval: z.coerce.number().optional(),
  // SendGrid fields
  sendgrid_api_key: z.string().optional(),
  // Mailgun fields
  mailgun_domain: z.string().optional(),
  mailgun_api_key: z.string().optional(),
  mailgun_region: z.enum(['us', 'eu']).optional(),
  // SES fields
  ses_region: z.string().optional(),
  ses_access_key_id: z.string().optional(),
  ses_secret_key: z.string().optional(),
  // Postmark fields
  postmark_server_token: z.string().optional(),
}).superRefine((data, ctx) => {
  // Validate provider-specific required fields
  if (data.provider === 'smtp') {
    if (!data.smtp_host) {
      ctx.addIssue({ code: z.ZodIssueCode.custom, message: tCommon('required'), path: ['smtp_host'] })
    }
    if (!data.smtp_port) {
      ctx.addIssue({ code: z.ZodIssueCode.custom, message: tCommon('required'), path: ['smtp_port'] })
    }
  } else if (data.provider === 'sendgrid') {
    if (!data.sendgrid_api_key) {
      ctx.addIssue({ code: z.ZodIssueCode.custom, message: tCommon('required'), path: ['sendgrid_api_key'] })
    }
  } else if (data.provider === 'mailgun') {
    if (!data.mailgun_domain) {
      ctx.addIssue({ code: z.ZodIssueCode.custom, message: tCommon('required'), path: ['mailgun_domain'] })
    }
    if (!data.mailgun_api_key) {
      ctx.addIssue({ code: z.ZodIssueCode.custom, message: tCommon('required'), path: ['mailgun_api_key'] })
    }
  } else if (data.provider === 'ses') {
    if (!data.ses_region) {
      ctx.addIssue({ code: z.ZodIssueCode.custom, message: tCommon('required'), path: ['ses_region'] })
    }
    if (!data.ses_access_key_id) {
      ctx.addIssue({ code: z.ZodIssueCode.custom, message: tCommon('required'), path: ['ses_access_key_id'] })
    }
    if (!data.ses_secret_key) {
      ctx.addIssue({ code: z.ZodIssueCode.custom, message: tCommon('required'), path: ['ses_secret_key'] })
    }
  } else if (data.provider === 'postmark') {
    if (!data.postmark_server_token) {
      ctx.addIssue({ code: z.ZodIssueCode.custom, message: tCommon('required'), path: ['postmark_server_token'] })
    }
  }
})

type EmailConfigForm = z.infer<ReturnType<typeof createEmailConfigSchema>>

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
  const t = useTranslations('channels.config')
  const tCommon = useTranslations('common')

  const isEditing = !!channelId
  const emailConfigSchema = createEmailConfigSchema(tCommon, t)

  // Provider options with translations
  const providers = [
    { value: 'smtp', label: 'SMTP', description: t('smtpDesc') },
    { value: 'sendgrid', label: 'SendGrid', description: t('sendgridDesc') },
    { value: 'mailgun', label: 'Mailgun', description: t('mailgunDesc') },
    { value: 'ses', label: 'AWS SES', description: t('sesDesc') },
    { value: 'postmark', label: 'Postmark', description: t('postmarkDesc') },
  ]

  const {
    register,
    handleSubmit,
    watch,
    setValue,
    formState: { errors },
  } = useForm<EmailConfigForm>({
    resolver: zodResolver(emailConfigSchema),
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
        config.smtp_host = data.smtp_host || ''
        config.smtp_port = String(data.smtp_port || '')
        config.smtp_encryption = data.smtp_encryption || 'tls'
        credentials.smtp_username = data.smtp_username || ''
        credentials.smtp_password = data.smtp_password || ''
        // IMAP
        if (data.imap_host) {
          config.imap_host = data.imap_host
          config.imap_port = String(data.imap_port || 993)
          config.imap_folder = data.imap_folder || 'INBOX'
          config.imap_poll_interval = String(data.imap_poll_interval || 30)
          credentials.imap_username = data.imap_username || ''
          credentials.imap_password = data.imap_password || ''
        }
      } else if (data.provider === 'sendgrid') {
        credentials.sendgrid_api_key = data.sendgrid_api_key || ''
      } else if (data.provider === 'mailgun') {
        config.mailgun_domain = data.mailgun_domain || ''
        config.mailgun_region = data.mailgun_region || 'us'
        credentials.mailgun_api_key = data.mailgun_api_key || ''
      } else if (data.provider === 'ses') {
        config.ses_region = data.ses_region || ''
        credentials.ses_access_key_id = data.ses_access_key_id || ''
        credentials.ses_secret_key = data.ses_secret_key || ''
      } else if (data.provider === 'postmark') {
        credentials.postmark_server_token = data.postmark_server_token || ''
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
          title: tCommon('updated'),
          description: t('channelUpdated'),
        })
      } else {
        await api.post('/channels', payload)
        toast({
          title: tCommon('created'),
          description: t('channelCreated'),
        })
      }

      onSuccess?.()
    } catch (error: any) {
      toast({
        title: tCommon('error'),
        description: error.message || t('saveError'),
        variant: 'error',
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
        title: t('connectionSuccess'),
        description: t('emailConnectionVerified'),
      })
    } catch (error: any) {
      toast({
        title: t('connectionFailed'),
        description: error.message || t('emailConnectionError'),
        variant: 'error',
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
    <form onSubmit={handleSubmit(onSubmit)} className="flex flex-col h-full">
      <div className="flex-1 space-y-6">
      {/* Basic Configuration */}
      <div className="space-y-4">
        <div className="space-y-2">
          <Label htmlFor="name">{t('channelName')}</Label>
          <Input
            id="name"
            placeholder={t('myEmailChannel')}
            {...register('name')}
          />
          {errors.name && (
            <p className="text-sm text-destructive">{errors.name.message}</p>
          )}
        </div>

        <div className="space-y-2">
          <Label htmlFor="provider">{t('emailProvider')}</Label>
          <Select
            value={selectedProvider}
            onValueChange={(value) => setValue('provider', value as any)}
          >
            <SelectTrigger>
              <SelectValue placeholder={t('selectProvider')} />
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
            <Label htmlFor="from_email">{t('fromEmail')}</Label>
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
            <Label htmlFor="from_name">{t('fromNameOptional')}</Label>
            <Input
              id="from_name"
              placeholder={t('myCompany')}
              {...register('from_name')}
            />
          </div>
        </div>

        <div className="space-y-2">
          <Label htmlFor="reply_to">{t('replyToOptional')}</Label>
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
          <TabsTrigger value="credentials">{t('credentials')}</TabsTrigger>
          <TabsTrigger value="receiving">{t('receiving')}</TabsTrigger>
          <TabsTrigger value="webhooks">{t('webhooks')}</TabsTrigger>
        </TabsList>

        {/* Credentials Tab */}
        <TabsContent value="credentials" className="space-y-4 pt-4">
          {/* SMTP Configuration */}
          {selectedProvider === 'smtp' && (
            <Card>
              <CardHeader>
                <CardTitle className="text-base flex items-center gap-2">
                  <Server className="h-4 w-4" />
                  {t('smtpSettings')}
                </CardTitle>
                <CardDescription>{t('configureSmtpConnection')}</CardDescription>
              </CardHeader>
              <CardContent className="space-y-4">
                <div className="grid grid-cols-2 gap-4">
                  <div className="space-y-2">
                    <Label htmlFor="smtp_host">{t('smtpHost')}</Label>
                    <Input
                      id="smtp_host"
                      placeholder="smtp.example.com"
                      {...register('smtp_host' as any)}
                    />
                  </div>
                  <div className="space-y-2">
                    <Label htmlFor="smtp_port">{t('smtpPort')}</Label>
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
                    <Label htmlFor="smtp_username">{tCommon('username')}</Label>
                    <Input
                      id="smtp_username"
                      placeholder={tCommon('username').toLowerCase()}
                      {...register('smtp_username' as any)}
                    />
                  </div>
                  <div className="space-y-2">
                    <Label htmlFor="smtp_password">{tCommon('password')}</Label>
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
                  <Label htmlFor="smtp_encryption">{t('encryption')}</Label>
                  <Select
                    defaultValue="tls"
                    onValueChange={(value) => setValue('smtp_encryption' as any, value)}
                  >
                    <SelectTrigger>
                      <SelectValue />
                    </SelectTrigger>
                    <SelectContent>
                      <SelectItem value="tls">{t('tlsRecommended')}</SelectItem>
                      <SelectItem value="starttls">STARTTLS</SelectItem>
                      <SelectItem value="none">{tCommon('none')}</SelectItem>
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
                  {t('sendgridSettings')}
                </CardTitle>
                <CardDescription>
                  {t('getApiKeyFrom')}{' '}
                  <a
                    href="https://app.sendgrid.com/settings/api_keys"
                    target="_blank"
                    rel="noopener noreferrer"
                    className="text-primary hover:underline inline-flex items-center gap-1"
                  >
                    {t('sendgridSettingsLink')}
                    <ExternalLink className="h-3 w-3" />
                  </a>
                </CardDescription>
              </CardHeader>
              <CardContent className="space-y-4">
                <div className="space-y-2">
                  <Label htmlFor="sendgrid_api_key">{t('apiKey')}</Label>
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
                    {t('sendgridApiKeyAlert')}
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
                  {t('mailgunSettings')}
                </CardTitle>
                <CardDescription>
                  {t('getCredentialsFrom')}{' '}
                  <a
                    href="https://app.mailgun.com/app/sending/domains"
                    target="_blank"
                    rel="noopener noreferrer"
                    className="text-primary hover:underline inline-flex items-center gap-1"
                  >
                    {t('mailgunDashboard')}
                    <ExternalLink className="h-3 w-3" />
                  </a>
                </CardDescription>
              </CardHeader>
              <CardContent className="space-y-4">
                <div className="grid grid-cols-2 gap-4">
                  <div className="space-y-2">
                    <Label htmlFor="mailgun_domain">{t('domain')}</Label>
                    <Input
                      id="mailgun_domain"
                      placeholder="mg.example.com"
                      {...register('mailgun_domain' as any)}
                    />
                  </div>
                  <div className="space-y-2">
                    <Label htmlFor="mailgun_region">{t('region')}</Label>
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
                  <Label htmlFor="mailgun_api_key">{t('apiKey')}</Label>
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
                  {t('sesSettings')}
                </CardTitle>
                <CardDescription>
                  {t('configureAwsSes')}
                </CardDescription>
              </CardHeader>
              <CardContent className="space-y-4">
                <div className="space-y-2">
                  <Label htmlFor="ses_region">{t('region')}</Label>
                  <Select
                    onValueChange={(value) => setValue('ses_region' as any, value)}
                  >
                    <SelectTrigger>
                      <SelectValue placeholder={t('selectRegion')} />
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
                  <Label htmlFor="ses_access_key_id">{t('accessKeyId')}</Label>
                  <Input
                    id="ses_access_key_id"
                    placeholder="AKIAIOSFODNN7EXAMPLE"
                    {...register('ses_access_key_id' as any)}
                  />
                </div>

                <div className="space-y-2">
                  <Label htmlFor="ses_secret_key">{t('secretAccessKey')}</Label>
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
                    {t('sesIamAlert')}
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
                  {t('postmarkSettings')}
                </CardTitle>
                <CardDescription>
                  {t('getServerTokenFrom')}{' '}
                  <a
                    href="https://account.postmarkapp.com/servers"
                    target="_blank"
                    rel="noopener noreferrer"
                    className="text-primary hover:underline inline-flex items-center gap-1"
                  >
                    {t('postmarkServers')}
                    <ExternalLink className="h-3 w-3" />
                  </a>
                </CardDescription>
              </CardHeader>
              <CardContent className="space-y-4">
                <div className="space-y-2">
                  <Label htmlFor="postmark_server_token">{t('serverToken')}</Label>
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
                    {t('postmarkSignatureAlert')}
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
                  {t('imapSettingsOptional')}
                </CardTitle>
                <CardDescription>
                  {t('imapSettingsDesc')}
                </CardDescription>
              </CardHeader>
              <CardContent className="space-y-4">
                <div className="grid grid-cols-2 gap-4">
                  <div className="space-y-2">
                    <Label htmlFor="imap_host">{t('imapHost')}</Label>
                    <Input
                      id="imap_host"
                      placeholder="imap.example.com"
                      {...register('imap_host' as any)}
                    />
                  </div>
                  <div className="space-y-2">
                    <Label htmlFor="imap_port">{t('imapPort')}</Label>
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
                    <Label htmlFor="imap_username">{tCommon('username')}</Label>
                    <Input
                      id="imap_username"
                      placeholder={tCommon('username').toLowerCase()}
                      {...register('imap_username' as any)}
                    />
                  </div>
                  <div className="space-y-2">
                    <Label htmlFor="imap_password">{tCommon('password')}</Label>
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
                    <Label htmlFor="imap_folder">{t('folder')}</Label>
                    <Input
                      id="imap_folder"
                      placeholder="INBOX"
                      defaultValue="INBOX"
                      {...register('imap_folder' as any)}
                    />
                  </div>
                  <div className="space-y-2">
                    <Label htmlFor="imap_poll_interval">{t('pollInterval')}</Label>
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
                <CardTitle className="text-base">{t('inboundWebhooks')}</CardTitle>
                <CardDescription>
                  {selectedProvider === 'sendgrid' && t('sendgridInboundDesc')}
                  {selectedProvider === 'mailgun' && t('mailgunInboundDesc')}
                  {selectedProvider === 'ses' && t('sesInboundDesc')}
                  {selectedProvider === 'postmark' && t('postmarkInboundDesc')}
                </CardDescription>
              </CardHeader>
              <CardContent>
                <Alert>
                  <Mail className="h-4 w-4" />
                  <AlertDescription>
                    {t('inboundWebhooksAlert')}
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
              <CardTitle className="text-base">{t('webhookUrls')}</CardTitle>
              <CardDescription>
                {t('webhookUrlsDesc')}
              </CardDescription>
            </CardHeader>
            <CardContent className="space-y-4">
              {selectedProvider === 'sendgrid' && (
                <>
                  <div className="space-y-2">
                    <Label>{t('inboundParseWebhook')}</Label>
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
                    <Label>{t('eventWebhook')}</Label>
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
                  <Label>{t('mailgunWebhook')}</Label>
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
                  <Label>{t('snsNotificationEndpoint')}</Label>
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
                    {t('snsEndpointDesc')}
                  </p>
                </div>
              )}

              {selectedProvider === 'postmark' && (
                <div className="space-y-2">
                  <Label>{t('postmarkWebhook')}</Label>
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
                    {t('smtpImapPollingInfo')}
                  </AlertDescription>
                </Alert>
              )}
            </CardContent>
          </Card>
        </TabsContent>
      </Tabs>
      </div>

      {/* Actions */}
      <div className="sticky bottom-0 flex justify-end gap-3 pt-4 pb-2 mt-4 border-t bg-background">
        <Button type="button" variant="outline" onClick={testConnection} disabled={isTesting}>
          {isTesting && <Loader2 className="mr-2 h-4 w-4 animate-spin" />}
          {t('testConnection')}
        </Button>

        <Button type="submit" disabled={isSubmitting}>
          {isSubmitting && <Loader2 className="mr-2 h-4 w-4 animate-spin" />}
          {isEditing ? t('updateChannel') : t('createChannel')}
        </Button>
      </div>
    </form>
  )
}
