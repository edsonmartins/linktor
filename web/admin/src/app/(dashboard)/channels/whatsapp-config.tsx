'use client'

import { useState } from 'react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
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
  PhoneCall,
  Webhook,
  Wallet,
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
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { Switch } from '@/components/ui/switch'
import { EnvironmentBadge } from '@/components/environment-badge'
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
import { api, WEBHOOK_BASE_URL } from '@/lib/api'
import { copyText } from '@/lib/clipboard'
import type { Channel } from '@/types'

/**
 * WhatsApp Official Configuration Schema
 */
const whatsappConfigSchema = z
  .object({
    name: z.string().min(1, 'Channel name is required'),
    access_token: z.string().min(1, 'Access token is required'),
    phone_number_id: z.string().min(1, 'Phone number ID is required'),
    business_id: z.string().optional(), // Optional - not used in embedded signup flow
    verify_token: z.string().min(1, 'Verify token is required'),
    webhook_secret: z.string().optional(),
    api_version: z.string().min(1),
    // Environment (INV-016): selectable at creation only; immutable afterwards.
    environment: z.enum(['production', 'sandbox']),
    // Declarative test-credential binding (INV-002): the human declares the
    // credentials belong to the TEST number. Format-level check only — the
    // backend revalidates and the delivery allowlist is the hard barrier.
    credential_is_sandbox: z.boolean(),
    sandbox_test_phone_number_ids: z.string().optional(),
  })
  .superRefine((data, ctx) => {
    if (data.environment !== 'sandbox') return
    if (!data.credential_is_sandbox) {
      ctx.addIssue({
        code: z.ZodIssueCode.custom,
        path: ['credential_is_sandbox'],
        message: 'The test-credential declaration is required for a sandbox channel',
      })
    }
    if (!data.sandbox_test_phone_number_ids?.trim()) {
      ctx.addIssue({
        code: z.ZodIssueCode.custom,
        path: ['sandbox_test_phone_number_ids'],
        message: 'The test phone_number_ids list is required for a sandbox channel',
      })
    }
  })

type WhatsAppConfigForm = z.infer<typeof whatsappConfigSchema>

interface WhatsAppConfigProps {
  channel?: Channel
  onSuccess?: (channel: Channel) => void
  onCancel?: () => void
}

interface PaymentStats {
  total_payments: number
  successful_payments: number
  failed_payments: number
  total_amount: number
  refunded_amount: number
  currency: string
  success_rate: number
}

interface PaymentResponse {
  payment_id: string
  status: string
  payment_url?: string
  qr_code?: string
}

interface CallStats {
  total_calls: number
  inbound_calls: number
  outbound_calls: number
  completed_calls: number
  missed_calls: number
  failed_calls: number
  total_duration: number
  average_duration: number
}

interface ChannelCall {
  id: string
  to: string
  type: 'voice' | 'video'
  status: string
  duration: number
  created_at: string
}

interface RecentCallsResponse {
  calls: ChannelCall[]
  limit: number
  offset: number
}

interface CTWADashboard {
  summary?: {
    total_referrals?: number
    total_conversions?: number
    conversion_rate?: number
    total_value?: number
    currency?: string
    average_value?: number
  }
  top_ads?: Array<{
    ad_id: string
    ad_name: string
    campaign_name: string
    referrals: number
    conversions: number
  }>
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
  const tEnv = useTranslations('environment')
  const { toast } = useToast()
  const [isSubmitting, setIsSubmitting] = useState(false)
  const [showAccessToken, setShowAccessToken] = useState(false)
  const [showWebhookSecret, setShowWebhookSecret] = useState(false)
  const [testStatus, setTestStatus] = useState<'idle' | 'testing' | 'success' | 'error'>('idle')
  const [paymentTo, setPaymentTo] = useState('')
  const [paymentAmount, setPaymentAmount] = useState('')
  const [paymentReferenceId, setPaymentReferenceId] = useState('')
  const [paymentDescription, setPaymentDescription] = useState('')
  const [callTo, setCallTo] = useState('')
  const [callType, setCallType] = useState<'voice' | 'video'>('voice')
  const [conversionReferralId, setConversionReferralId] = useState('')
  const [conversionType, setConversionType] = useState('purchase')
  const [conversionValue, setConversionValue] = useState('')
  const queryClient = useQueryClient()

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
      api_version: (channel?.config?.api_version as string) || 'v23.0',
      environment: channel?.environment === 'sandbox' ? 'sandbox' : 'production',
      credential_is_sandbox: channel?.environment === 'sandbox',
      sandbox_test_phone_number_ids:
        (channel?.config?.sandbox_test_phone_number_ids as string) || '',
    },
  })
  const watchEnvironment = form.watch('environment')
  const isSandbox = isEditing ? channel?.environment === 'sandbox' : watchEnvironment === 'sandbox'

  const webhookUrl = channel
    ? `${WEBHOOK_BASE_URL}/api/v1/webhooks/whatsapp_official/${channel.id}`
    : t('willBeGenerated')

  const paymentsStatsQuery = useQuery({
    queryKey: ['channel-operations', channel?.id, 'payments-stats'],
    queryFn: () => api.get<PaymentStats>(`/channels/${channel!.id}/payments/stats`),
    enabled: isEditing,
    retry: false,
  })

  const callStatsQuery = useQuery({
    queryKey: ['channel-operations', channel?.id, 'calls-stats'],
    queryFn: () => api.get<CallStats>(`/channels/${channel!.id}/calls/stats`),
    enabled: isEditing,
    retry: false,
  })

  const recentCallsQuery = useQuery({
    queryKey: ['channel-operations', channel?.id, 'calls', 'recent'],
    queryFn: () =>
      api.get<RecentCallsResponse>(`/channels/${channel!.id}/calls`, {
        limit: '5',
        offset: '0',
      }),
    enabled: isEditing,
    retry: false,
  })

  const ctwaDashboardQuery = useQuery({
    queryKey: ['channel-operations', channel?.id, 'ctwa-dashboard'],
    queryFn: () => api.get<CTWADashboard>(`/channels/${channel!.id}/ctwa/dashboard`),
    enabled: isEditing,
    retry: false,
  })

  const createPaymentMutation = useMutation({
    mutationFn: () =>
      api.post<PaymentResponse>(`/channels/${channel!.id}/payments`, {
        to: paymentTo,
        reference_id: paymentReferenceId,
        type: 'order',
        amount: Number(paymentAmount),
        currency: 'BRL',
        description: paymentDescription,
      }),
    onSuccess: (result) => {
      toast({
        title: 'Payment created',
        description: result.payment_id,
      })
      setPaymentTo('')
      setPaymentAmount('')
      setPaymentReferenceId('')
      setPaymentDescription('')
      queryClient.invalidateQueries({
        queryKey: ['channel-operations', channel?.id, 'payments-stats'],
      })
    },
    onError: (error) => {
      toast({
        title: 'Failed to create payment',
        description: error instanceof Error ? error.message : 'Unknown error',
        variant: 'error',
      })
    },
  })

  const initiateCallMutation = useMutation({
    mutationFn: () =>
      api.post(`/channels/${channel!.id}/calls`, {
        to: callTo,
        type: callType,
      }),
    onSuccess: () => {
      toast({
        title: 'Call started',
        description: callTo,
      })
      setCallTo('')
      queryClient.invalidateQueries({
        queryKey: ['channel-operations', channel?.id, 'calls-stats'],
      })
      queryClient.invalidateQueries({
        queryKey: ['channel-operations', channel?.id, 'calls', 'recent'],
      })
    },
    onError: (error) => {
      toast({
        title: 'Failed to start call',
        description: error instanceof Error ? error.message : 'Unknown error',
        variant: 'error',
      })
    },
  })

  const trackConversionMutation = useMutation({
    mutationFn: () =>
      api.post(`/channels/${channel!.id}/ctwa/conversions`, {
        referral_id: conversionReferralId,
        conversion_type: conversionType,
        value: conversionValue ? Number(conversionValue) : 0,
        currency: 'BRL',
      }),
    onSuccess: () => {
      toast({
        title: 'Conversion tracked',
        description: conversionReferralId,
      })
      setConversionReferralId('')
      setConversionValue('')
      queryClient.invalidateQueries({
        queryKey: ['channel-operations', channel?.id, 'ctwa-dashboard'],
      })
    },
    onError: (error) => {
      toast({
        title: 'Failed to track conversion',
        description: error instanceof Error ? error.message : 'Unknown error',
        variant: 'error',
      })
    },
  })

  const onSubmit = async (data: WhatsAppConfigForm) => {
    setIsSubmitting(true)
    try {
      const payload = {
        name: data.name,
        type: 'whatsapp_official',
        // environment só na criação: imutável após criado (INV-016), o campo
        // nem é submetido no update.
        ...(isEditing ? {} : { environment: data.environment }),
        config: {
          phone_number_id: data.phone_number_id,
          business_id: data.business_id,
          verify_token: data.verify_token,
          api_version: data.api_version,
          ...(isSandbox && data.sandbox_test_phone_number_ids
            ? { sandbox_test_phone_number_ids: data.sandbox_test_phone_number_ids }
            : {}),
        },
        credentials: {
          access_token: data.access_token,
          webhook_secret: data.webhook_secret,
          ...(isSandbox ? { credential_environment: 'sandbox' } : {}),
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

  const copyToClipboard = async (text: string, label: string) => {
    if (!(await copyText(text))) return
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
          <TabsList className={`grid w-full ${isEditing ? 'grid-cols-5' : 'grid-cols-4'}`}>
            <TabsTrigger value="embedded" className="flex items-center gap-1.5">
              <Zap className="h-3.5 w-3.5" />
              {t('embeddedSignupTab')}
            </TabsTrigger>
            <TabsTrigger value="credentials" className="flex items-center gap-1.5">
              <Shield className="h-3.5 w-3.5" />
              {t('manualSetup')}
            </TabsTrigger>
            <TabsTrigger value="webhook">{t('webhook')}</TabsTrigger>
            <TabsTrigger value="setup">{t('setupGuide')}</TabsTrigger>
            {isEditing && (
              <TabsTrigger value="operations">{t('operations')}</TabsTrigger>
            )}
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

            {/* Ambiente do canal (INV-016). Na criação: selecionável, default
                production. Na edição: SOMENTE LEITURA com a razão visível —
                erro pós-submit para regra conhecida seria defeito de UI. */}
            {isEditing ? (
              <FormItem>
                <FormLabel>{tEnv('formLabel')}</FormLabel>
                <div className="flex items-center gap-2">
                  <EnvironmentBadge environment={channel?.environment} showProduction />
                </div>
                <FormDescription>{tEnv('immutableReason')}</FormDescription>
              </FormItem>
            ) : (
              <FormField
                control={form.control}
                name="environment"
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>{tEnv('formLabel')}</FormLabel>
                    <Select onValueChange={field.onChange} value={field.value}>
                      <FormControl>
                        <SelectTrigger>
                          <SelectValue />
                        </SelectTrigger>
                      </FormControl>
                      <SelectContent>
                        <SelectItem value="production">{tEnv('formProduction')}</SelectItem>
                        <SelectItem value="sandbox">{tEnv('formSandbox')}</SelectItem>
                      </SelectContent>
                    </Select>
                    <FormDescription>{tEnv('formDefaultHint')}</FormDescription>
                    <FormMessage />
                  </FormItem>
                )}
              />
            )}

            {/* Campos de sandbox: obrigatórios quando environment=sandbox. */}
            {isSandbox && (
              <div className="space-y-4 rounded-md border border-yellow-500/40 bg-yellow-500/5 p-4">
                <p className="text-sm font-medium">{tEnv('sandboxSection')}</p>
                <FormField
                  control={form.control}
                  name="sandbox_test_phone_number_ids"
                  render={({ field }) => (
                    <FormItem>
                      <FormLabel>{tEnv('testPhoneIdsLabel')}</FormLabel>
                      <FormControl>
                        <Input placeholder="111222333, 444555666" {...field} />
                      </FormControl>
                      <FormDescription>{tEnv('testPhoneIdsHint')}</FormDescription>
                      <FormMessage />
                    </FormItem>
                  )}
                />
                <FormField
                  control={form.control}
                  name="credential_is_sandbox"
                  render={({ field }) => (
                    <FormItem>
                      <div className="flex items-center gap-2">
                        <FormControl>
                          <Switch
                            checked={field.value}
                            onCheckedChange={field.onChange}
                            disabled={isEditing}
                            aria-label={tEnv('credentialDeclaration')}
                          />
                        </FormControl>
                        <FormLabel className="!mt-0">{tEnv('credentialDeclaration')}</FormLabel>
                      </div>
                      <FormDescription>
                        {isEditing
                          ? tEnv('credentialDeclarationStored')
                          : tEnv('credentialDeclarationHint')}
                      </FormDescription>
                      <FormMessage />
                    </FormItem>
                  )}
                />
              </div>
            )}

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
                    <Input placeholder="v23.0" {...field} />
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

          {isEditing && channel && (
            <TabsContent value="operations" className="space-y-4 mt-4">
              <Card>
                <CardHeader>
                  <CardTitle className="flex items-center gap-2">
                    <Wallet className="h-4 w-4" />
                    Payments
                  </CardTitle>
                  <CardDescription>
                    Stats and a basic payment request flow for this channel.
                  </CardDescription>
                </CardHeader>
                <CardContent className="space-y-4">
                  <OperationsState
                    isLoading={paymentsStatsQuery.isLoading}
                    error={paymentsStatsQuery.error}
                  >
                    <div className="grid gap-3 md:grid-cols-4">
                      <MetricCard label="Total payments" value={String(paymentsStatsQuery.data?.total_payments ?? 0)} />
                      <MetricCard label="Success rate" value={formatPercent(paymentsStatsQuery.data?.success_rate)} />
                      <MetricCard
                        label="Total amount"
                        value={formatMinorCurrency(
                          paymentsStatsQuery.data?.total_amount,
                          paymentsStatsQuery.data?.currency
                        )}
                      />
                      <MetricCard
                        label="Refunded"
                        value={formatMinorCurrency(
                          paymentsStatsQuery.data?.refunded_amount,
                          paymentsStatsQuery.data?.currency
                        )}
                      />
                    </div>
                  </OperationsState>

                  <div className="grid gap-3 md:grid-cols-2">
                    <div className="space-y-2">
                      <Label htmlFor="paymentTo">Customer phone</Label>
                      <Input
                        id="paymentTo"
                        value={paymentTo}
                        onChange={(event) => setPaymentTo(event.target.value)}
                        placeholder="+5511999999999"
                      />
                    </div>
                    <div className="space-y-2">
                      <Label htmlFor="paymentAmount">Amount (cents)</Label>
                      <Input
                        id="paymentAmount"
                        value={paymentAmount}
                        onChange={(event) => setPaymentAmount(event.target.value)}
                        placeholder="1999"
                      />
                    </div>
                    <div className="space-y-2">
                      <Label htmlFor="paymentReferenceId">Reference ID</Label>
                      <Input
                        id="paymentReferenceId"
                        value={paymentReferenceId}
                        onChange={(event) => setPaymentReferenceId(event.target.value)}
                        placeholder="order-123"
                      />
                    </div>
                    <div className="space-y-2">
                      <Label htmlFor="paymentDescription">Description</Label>
                      <Input
                        id="paymentDescription"
                        value={paymentDescription}
                        onChange={(event) => setPaymentDescription(event.target.value)}
                        placeholder="Order payment"
                      />
                    </div>
                  </div>

                  <Button
                    type="button"
                    onClick={() => createPaymentMutation.mutate()}
                    disabled={
                      createPaymentMutation.isPending ||
                      !paymentTo ||
                      !paymentAmount ||
                      !paymentReferenceId
                    }
                  >
                    {createPaymentMutation.isPending && (
                      <Loader2 className="h-4 w-4 mr-2 animate-spin" />
                    )}
                    Create payment
                  </Button>
                </CardContent>
              </Card>

              <Card>
                <CardHeader>
                  <CardTitle className="flex items-center gap-2">
                    <PhoneCall className="h-4 w-4" />
                    Calls
                  </CardTitle>
                  <CardDescription>
                    Live channel call stats and recent outbound activity.
                  </CardDescription>
                </CardHeader>
                <CardContent className="space-y-4">
                  <OperationsState
                    isLoading={callStatsQuery.isLoading}
                    error={callStatsQuery.error}
                  >
                    <div className="grid gap-3 md:grid-cols-4">
                      <MetricCard label="Total calls" value={String(callStatsQuery.data?.total_calls ?? 0)} />
                      <MetricCard label="Completed" value={String(callStatsQuery.data?.completed_calls ?? 0)} />
                      <MetricCard label="Missed" value={String(callStatsQuery.data?.missed_calls ?? 0)} />
                      <MetricCard
                        label="Average duration"
                        value={formatDuration(callStatsQuery.data?.average_duration)}
                      />
                    </div>
                  </OperationsState>

                  <div className="grid gap-3 md:grid-cols-[minmax(0,1fr)_160px_auto]">
                    <div className="space-y-2">
                      <Label htmlFor="callTo">Call destination</Label>
                      <Input
                        id="callTo"
                        value={callTo}
                        onChange={(event) => setCallTo(event.target.value)}
                        placeholder="+5511888888888"
                      />
                    </div>
                    <div className="space-y-2">
                      <Label htmlFor="callType">Call type</Label>
                      <Input
                        id="callType"
                        value={callType}
                        onChange={(event) => setCallType(event.target.value === 'video' ? 'video' : 'voice')}
                        placeholder="voice"
                      />
                    </div>
                    <div className="flex items-end">
                      <Button
                        type="button"
                        onClick={() => initiateCallMutation.mutate()}
                        disabled={initiateCallMutation.isPending || !callTo}
                      >
                        {initiateCallMutation.isPending && (
                          <Loader2 className="h-4 w-4 mr-2 animate-spin" />
                        )}
                        Start call
                      </Button>
                    </div>
                  </div>

                  <OperationsState
                    isLoading={recentCallsQuery.isLoading}
                    error={recentCallsQuery.error}
                  >
                    <div className="space-y-2">
                      <Label>Recent calls</Label>
                      <div className="rounded-md border">
                        {(recentCallsQuery.data?.calls ?? []).length === 0 ? (
                          <div className="p-4 text-sm text-muted-foreground">
                            No recent calls.
                          </div>
                        ) : (
                          (recentCallsQuery.data?.calls ?? []).map((call) => (
                            <div
                              key={call.id}
                              className="flex items-center justify-between gap-3 border-b px-4 py-3 last:border-b-0"
                            >
                              <div>
                                <p className="text-sm font-medium">{call.to}</p>
                                <p className="text-xs text-muted-foreground">
                                  {call.type} · {call.status}
                                </p>
                              </div>
                              <Badge variant="outline">
                                {formatDuration(call.duration)}
                              </Badge>
                            </div>
                          ))
                        )}
                      </div>
                    </div>
                  </OperationsState>
                </CardContent>
              </Card>

              <Card>
                <CardHeader>
                  <CardTitle className="flex items-center gap-2">
                    <Zap className="h-4 w-4" />
                    CTWA
                  </CardTitle>
                  <CardDescription>
                    Ad referral metrics and manual conversion tracking.
                  </CardDescription>
                </CardHeader>
                <CardContent className="space-y-4">
                  <OperationsState
                    isLoading={ctwaDashboardQuery.isLoading}
                    error={ctwaDashboardQuery.error}
                  >
                    <div className="grid gap-3 md:grid-cols-4">
                      <MetricCard
                        label="Referrals"
                        value={String(ctwaDashboardQuery.data?.summary?.total_referrals ?? 0)}
                      />
                      <MetricCard
                        label="Conversions"
                        value={String(ctwaDashboardQuery.data?.summary?.total_conversions ?? 0)}
                      />
                      <MetricCard
                        label="Conversion rate"
                        value={formatPercent(ctwaDashboardQuery.data?.summary?.conversion_rate)}
                      />
                      <MetricCard
                        label="Total value"
                        value={formatCurrency(
                          ctwaDashboardQuery.data?.summary?.total_value,
                          ctwaDashboardQuery.data?.summary?.currency
                        )}
                      />
                    </div>
                  </OperationsState>

                  <div className="grid gap-3 md:grid-cols-3">
                    <div className="space-y-2">
                      <Label htmlFor="conversionReferralId">Referral ID</Label>
                      <Input
                        id="conversionReferralId"
                        value={conversionReferralId}
                        onChange={(event) => setConversionReferralId(event.target.value)}
                        placeholder="ref-123"
                      />
                    </div>
                    <div className="space-y-2">
                      <Label htmlFor="conversionType">Conversion type</Label>
                      <Input
                        id="conversionType"
                        value={conversionType}
                        onChange={(event) => setConversionType(event.target.value)}
                        placeholder="purchase"
                      />
                    </div>
                    <div className="space-y-2">
                      <Label htmlFor="conversionValue">Value</Label>
                      <Input
                        id="conversionValue"
                        value={conversionValue}
                        onChange={(event) => setConversionValue(event.target.value)}
                        placeholder="199.90"
                      />
                    </div>
                  </div>

                  <Button
                    type="button"
                    onClick={() => trackConversionMutation.mutate()}
                    disabled={trackConversionMutation.isPending || !conversionReferralId || !conversionType}
                  >
                    {trackConversionMutation.isPending && (
                      <Loader2 className="h-4 w-4 mr-2 animate-spin" />
                    )}
                    Track conversion
                  </Button>

                  <div className="space-y-2">
                    <Label>Top ads</Label>
                    <div className="rounded-md border">
                      {(ctwaDashboardQuery.data?.top_ads ?? []).length === 0 ? (
                        <div className="p-4 text-sm text-muted-foreground">
                          No CTWA ads recorded.
                        </div>
                      ) : (
                        (ctwaDashboardQuery.data?.top_ads ?? []).map((ad) => (
                          <div
                            key={ad.ad_id}
                            className="flex items-center justify-between gap-3 border-b px-4 py-3 last:border-b-0"
                          >
                            <div>
                              <p className="text-sm font-medium">{ad.ad_name}</p>
                              <p className="text-xs text-muted-foreground">
                                {ad.campaign_name || ad.ad_id}
                              </p>
                            </div>
                            <Badge variant="outline">
                              {ad.conversions} conversions
                            </Badge>
                          </div>
                        ))
                      )}
                    </div>
                  </div>
                </CardContent>
              </Card>
            </TabsContent>
          )}
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

function MetricCard({ label, value }: { label: string; value: string }) {
  return (
    <div className="rounded-md border p-3">
      <p className="text-xs text-muted-foreground">{label}</p>
      <p className="mt-1 text-lg font-semibold">{value}</p>
    </div>
  )
}

function OperationsState({
  isLoading,
  error,
  children,
}: {
  isLoading: boolean
  error: unknown
  children: React.ReactNode
}) {
  if (isLoading) {
    return (
      <div className="flex items-center gap-2 text-sm text-muted-foreground">
        <Loader2 className="h-4 w-4 animate-spin" />
        Loading channel operations...
      </div>
    )
  }

  if (error) {
    return (
      <Alert>
        <AlertCircle className="h-4 w-4" />
        <AlertTitle>Unavailable</AlertTitle>
        <AlertDescription>
          {error instanceof Error ? error.message : 'Failed to load data'}
        </AlertDescription>
      </Alert>
    )
  }

  return <>{children}</>
}

function formatMinorCurrency(amount?: number, currency = 'BRL') {
  return new Intl.NumberFormat('pt-BR', {
    style: 'currency',
    currency,
  }).format((amount ?? 0) / 100)
}

function formatCurrency(amount?: number, currency = 'BRL') {
  return new Intl.NumberFormat('pt-BR', {
    style: 'currency',
    currency,
  }).format(amount ?? 0)
}

function formatPercent(value?: number) {
  return `${((value ?? 0) * 100).toFixed(1)}%`
}

function formatDuration(value?: number) {
  const seconds = Math.round(value ?? 0)
  const minutes = Math.floor(seconds / 60)
  const remainder = seconds % 60
  return `${minutes}m ${remainder}s`
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
