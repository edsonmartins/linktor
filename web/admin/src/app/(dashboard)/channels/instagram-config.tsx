'use client'

import { useState, useEffect } from 'react'
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
  Facebook,
  Loader2,
  Instagram,
  Webhook,
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
 * Instagram Configuration Schema
 */
const createInstagramConfigSchema = (tCommon: (key: string) => string) => z.object({
  name: z.string().min(1, tCommon('required')),
  instagram_id: z.string().min(1, tCommon('required')),
  access_token: z.string().min(1, tCommon('required')),
  app_id: z.string().optional(),
  app_secret: z.string().optional(),
  verify_token: z.string().min(1, tCommon('required')),
})

type InstagramConfigForm = z.infer<ReturnType<typeof createInstagramConfigSchema>>

interface InstagramConfigProps {
  channel?: Channel
  onSuccess?: (channel: Channel) => void
  onCancel?: () => void
}

// OAuth Account type
interface OAuthInstagramAccount {
  id: string
  username: string
  name: string
  profile_picture_url?: string
  followers_count: number
  page_id: string
  page_name: string
  page_access_token: string
}

/**
 * Instagram DM Channel Configuration Component
 */
export function InstagramConfig({
  channel,
  onSuccess,
  onCancel,
}: InstagramConfigProps) {
  const { toast } = useToast()
  const t = useTranslations('channels.config')
  const tCommon = useTranslations('common')
  const [isSubmitting, setIsSubmitting] = useState(false)
  const [showAccessToken, setShowAccessToken] = useState(false)
  const [showAppSecret, setShowAppSecret] = useState(false)
  const [testStatus, setTestStatus] = useState<'idle' | 'testing' | 'success' | 'error'>('idle')

  // OAuth state
  const [oauthLoading, setOauthLoading] = useState(false)
  const [oauthAccounts, setOauthAccounts] = useState<OAuthInstagramAccount[]>([])
  const [selectedAccount, setSelectedAccount] = useState<OAuthInstagramAccount | null>(null)
  const [oauthAppId, setOauthAppId] = useState('')
  const [oauthAppSecret, setOauthAppSecret] = useState('')

  const isEditing = !!channel
  const instagramConfigSchema = createInstagramConfigSchema(tCommon)

  const form = useForm<InstagramConfigForm>({
    resolver: zodResolver(instagramConfigSchema),
    defaultValues: {
      name: channel?.name || '',
      instagram_id: (channel?.config?.instagram_id as string) || '',
      access_token: '',
      app_id: (channel?.config?.app_id as string) || '',
      app_secret: '',
      verify_token: (channel?.config?.verify_token as string) || generateVerifyToken(),
    },
  })

  const webhookUrl = channel
    ? `${window.location.origin}/api/v1/webhooks/instagram/${channel.id}`
    : t('webhookPending')

  const onSubmit = async (data: InstagramConfigForm) => {
    setIsSubmitting(true)
    try {
      const payload = {
        name: data.name,
        type: 'instagram',
        config: {
          instagram_id: data.instagram_id,
          app_id: data.app_id,
          verify_token: data.verify_token,
        },
        credentials: {
          access_token: data.access_token,
          app_secret: data.app_secret,
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
    if (!values.access_token) {
      toast({
        title: t('missingCredentials'),
        description: t('enterAccessTokenFirst'),
        variant: 'error',
      })
      return
    }

    setTestStatus('testing')
    try {
      await api.post('/channels/test-instagram', {
        access_token: values.access_token,
        instagram_id: values.instagram_id,
      })
      setTestStatus('success')
      toast({
        title: t('connectionSuccess'),
        description: t('instagramCredentialsValid'),
      })
    } catch {
      setTestStatus('error')
      toast({
        title: t('connectionFailed'),
        description: t('instagramConnectionError'),
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

  // OAuth: Start Instagram login (via Facebook)
  const startOAuthFlow = async () => {
    if (!oauthAppId || !oauthAppSecret) {
      toast({
        title: t('missingCredentials'),
        description: t('enterAppCredentialsFirst'),
        variant: 'error',
      })
      return
    }

    setOauthLoading(true)
    try {
      const response = await api.post<{ login_url: string; state: string }>('/oauth/instagram/login', {
        app_id: oauthAppId,
        app_secret: oauthAppSecret,
      })

      // Store state for callback
      sessionStorage.setItem('ig_oauth_state', response.state)
      sessionStorage.setItem('ig_oauth_app_id', oauthAppId)
      sessionStorage.setItem('ig_oauth_app_secret', oauthAppSecret)

      // Open Facebook login in popup (Instagram uses Facebook OAuth)
      const popup = window.open(
        response.login_url,
        'instagram_oauth',
        'width=600,height=700,scrollbars=yes'
      )

      // Listen for popup close/redirect
      const checkPopup = setInterval(() => {
        try {
          if (popup?.closed) {
            clearInterval(checkPopup)
            setOauthLoading(false)
          }
          // Check if redirected back with code
          if (popup?.location?.href?.includes('code=')) {
            const url = new URL(popup.location.href)
            const code = url.searchParams.get('code')
            const state = url.searchParams.get('state')
            popup.close()
            clearInterval(checkPopup)
            if (code && state) {
              handleOAuthCallback(code, state)
            }
          }
        } catch {
          // Cross-origin error expected until redirect
        }
      }, 500)
    } catch (error) {
      setOauthLoading(false)
      toast({
        title: t('oauthError'),
        description: error instanceof Error ? error.message : t('oauthStartError'),
        variant: 'error',
      })
    }
  }

  // OAuth: Handle callback
  const handleOAuthCallback = async (code: string, state: string) => {
    setOauthLoading(true)
    try {
      const appId = sessionStorage.getItem('ig_oauth_app_id') || oauthAppId
      const appSecret = sessionStorage.getItem('ig_oauth_app_secret') || oauthAppSecret

      const response = await api.post<{
        user_access_token: string
        accounts: OAuthInstagramAccount[]
      }>('/oauth/instagram/callback', {
        code,
        state,
        app_id: appId,
        app_secret: appSecret,
      })

      setOauthAccounts(response.accounts)
      if (response.accounts.length === 1) {
        setSelectedAccount(response.accounts[0])
      }

      toast({
        title: t('connectedSuccessfully'),
        description: t('foundInstagramAccounts', { count: response.accounts.length }),
      })

      // Clean up session storage
      sessionStorage.removeItem('ig_oauth_state')
      sessionStorage.removeItem('ig_oauth_app_id')
      sessionStorage.removeItem('ig_oauth_app_secret')
    } catch (error) {
      toast({
        title: t('oauthError'),
        description: error instanceof Error ? error.message : t('oauthCompleteError'),
        variant: 'error',
      })
    } finally {
      setOauthLoading(false)
    }
  }

  // OAuth: Create channel from selected account
  const createFromOAuth = async () => {
    if (!selectedAccount) {
      toast({
        title: t('noAccountSelected'),
        description: t('selectInstagramAccount'),
        variant: 'error',
      })
      return
    }

    setIsSubmitting(true)
    try {
      const result = await api.post<{ channel: Channel }>('/oauth/channels', {
        name: `@${selectedAccount.username}`,
        type: 'instagram',
        instagram_id: selectedAccount.id,
        page_id: selectedAccount.page_id,
        access_token: selectedAccount.page_access_token,
        app_id: oauthAppId,
        app_secret: oauthAppSecret,
      })

      toast({
        title: tCommon('created'),
        description: t('instagramConnectedSuccessfully', { username: selectedAccount.username }),
      })

      onSuccess?.(result.channel)
    } catch (error) {
      toast({
        title: tCommon('error'),
        description: error instanceof Error ? error.message : t('createChannelError'),
        variant: 'error',
      })
    } finally {
      setIsSubmitting(false)
    }
  }

  return (
    <Form {...form}>
      <form onSubmit={form.handleSubmit(onSubmit)} className="flex flex-col h-full">
        <div className="flex-1 space-y-6">
        <Tabs defaultValue={isEditing ? "credentials" : "oauth"} className="w-full">
          <TabsList className="grid w-full grid-cols-4">
            {!isEditing && <TabsTrigger value="oauth">{t('connect')}</TabsTrigger>}
            <TabsTrigger value="credentials">{t('manual')}</TabsTrigger>
            <TabsTrigger value="webhook">{t('webhook')}</TabsTrigger>
            <TabsTrigger value="setup">{t('setupGuide')}</TabsTrigger>
          </TabsList>

          {/* OAuth Tab - Recommended for new channels */}
          {!isEditing && (
            <TabsContent value="oauth" className="space-y-4 mt-4">
              <Alert>
                <Instagram className="h-4 w-4" />
                <AlertTitle>{t('connectWithInstagram')}</AlertTitle>
                <AlertDescription>
                  {t('instagramOauthDesc')}
                </AlertDescription>
              </Alert>

              <div className="space-y-4">
                {/* App ID for OAuth */}
                <div className="space-y-2">
                  <label className="text-sm font-medium">{t('facebookAppId')}</label>
                  <Input
                    placeholder="123456789012345"
                    value={oauthAppId}
                    onChange={(e) => setOauthAppId(e.target.value)}
                  />
                  <p className="text-xs text-muted-foreground">
                    {t('fromFacebookDashboard')}
                  </p>
                </div>

                {/* App Secret for OAuth */}
                <div className="space-y-2">
                  <label className="text-sm font-medium">{t('facebookAppSecret')}</label>
                  <div className="relative">
                    <Input
                      type={showAppSecret ? 'text' : 'password'}
                      className="pr-10"
                      placeholder={t('yourAppSecret')}
                      value={oauthAppSecret}
                      onChange={(e) => setOauthAppSecret(e.target.value)}
                    />
                    <Button
                      type="button"
                      variant="ghost"
                      size="icon"
                      className="absolute right-0 top-0 h-full"
                      onClick={() => setShowAppSecret(!showAppSecret)}
                    >
                      {showAppSecret ? <EyeOff className="h-4 w-4" /> : <Eye className="h-4 w-4" />}
                    </Button>
                  </div>
                </div>

                {/* Connect Button */}
                <Button
                  type="button"
                  className="w-full"
                  onClick={startOAuthFlow}
                  disabled={oauthLoading || !oauthAppId || !oauthAppSecret}
                >
                  {oauthLoading ? (
                    <>
                      <Loader2 className="h-4 w-4 mr-2 animate-spin" />
                      {t('connecting')}
                    </>
                  ) : (
                    <>
                      <Facebook className="h-4 w-4 mr-2" />
                      {t('connectViaFacebook')}
                    </>
                  )}
                </Button>

                {/* Account Selection */}
                {oauthAccounts.length > 0 && (
                  <Card>
                    <CardHeader>
                      <CardTitle className="text-base">{t('selectAnAccount')}</CardTitle>
                      <CardDescription>
                        {t('selectInstagramAccountDesc')}
                      </CardDescription>
                    </CardHeader>
                    <CardContent className="space-y-2">
                      {oauthAccounts.map((account) => (
                        <div
                          key={account.id}
                          className={`p-3 border rounded-lg cursor-pointer transition-colors ${
                            selectedAccount?.id === account.id
                              ? 'border-primary bg-primary/5'
                              : 'hover:border-primary/50'
                          }`}
                          onClick={() => setSelectedAccount(account)}
                        >
                          <div className="flex items-center gap-3">
                            {account.profile_picture_url ? (
                              <img
                                src={account.profile_picture_url}
                                alt={account.username}
                                className="w-10 h-10 rounded-full"
                              />
                            ) : (
                              <div className="w-10 h-10 rounded-full bg-pink-500/10 flex items-center justify-center">
                                <Instagram className="h-5 w-5 text-pink-500" />
                              </div>
                            )}
                            <div className="flex-1">
                              <p className="font-medium">@{account.username}</p>
                              <p className="text-xs text-muted-foreground">
                                {account.followers_count.toLocaleString()} {t('followers')}
                              </p>
                            </div>
                            {selectedAccount?.id === account.id && (
                              <CheckCircle2 className="h-5 w-5 text-primary" />
                            )}
                          </div>
                          <div className="mt-2 pt-2 border-t text-xs text-muted-foreground">
                            {t('connectedVia')}: {account.page_name}
                          </div>
                        </div>
                      ))}

                      <Button
                        type="button"
                        className="w-full mt-4"
                        onClick={createFromOAuth}
                        disabled={!selectedAccount || isSubmitting}
                      >
                        {isSubmitting ? (
                          <>
                            <Loader2 className="h-4 w-4 mr-2 animate-spin" />
                            {t('creating')}
                          </>
                        ) : (
                          t('createChannel')
                        )}
                      </Button>
                    </CardContent>
                  </Card>
                )}

                {oauthAccounts.length === 0 && !oauthLoading && (
                  <Alert variant="default">
                    <AlertCircle className="h-4 w-4" />
                    <AlertTitle>{t('noInstagramAccounts')}</AlertTitle>
                    <AlertDescription>
                      {t('noInstagramAccountsDesc')}
                    </AlertDescription>
                  </Alert>
                )}
              </div>
            </TabsContent>
          )}

          <TabsContent value="credentials" className="space-y-4 mt-4">
            {/* Channel Name */}
            <FormField
              control={form.control}
              name="name"
              render={({ field }) => (
                <FormItem>
                  <FormLabel>{t('channelName')}</FormLabel>
                  <FormControl>
                    <Input placeholder={t('myInstagramBusiness')} {...field} />
                  </FormControl>
                  <FormDescription>
                    {t('channelNameDesc')}
                  </FormDescription>
                  <FormMessage />
                </FormItem>
              )}
            />

            {/* Instagram ID */}
            <FormField
              control={form.control}
              name="instagram_id"
              render={({ field }) => (
                <FormItem>
                  <FormLabel>{t('instagramBusinessAccountId')}</FormLabel>
                  <FormControl>
                    <div className="relative">
                      <Instagram className="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-muted-foreground" />
                      <Input className="pl-10" placeholder="17841234567890123" {...field} />
                    </div>
                  </FormControl>
                  <FormDescription>
                    {t('instagramBusinessAccountIdDesc')}
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
                        className="pr-10"
                        placeholder={isEditing ? '••••••••••••••••' : 'IGAAI...'}
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
                    {t('instagramAccessTokenDesc')}
                  </FormDescription>
                  <FormMessage />
                </FormItem>
              )}
            />

            {/* App ID (Optional) */}
            <FormField
              control={form.control}
              name="app_id"
              render={({ field }) => (
                <FormItem>
                  <FormLabel>
                    {t('appId')}
                    <Badge variant="outline" className="ml-2">{tCommon('optional')}</Badge>
                  </FormLabel>
                  <FormControl>
                    <Input placeholder="123456789012345" {...field} />
                  </FormControl>
                  <FormDescription>
                    {t('appIdDesc')}
                  </FormDescription>
                  <FormMessage />
                </FormItem>
              )}
            />

            {/* App Secret (Optional) */}
            <FormField
              control={form.control}
              name="app_secret"
              render={({ field }) => (
                <FormItem>
                  <FormLabel>
                    {t('appSecret')}
                    <Badge variant="outline" className="ml-2">{tCommon('optional')}</Badge>
                  </FormLabel>
                  <FormControl>
                    <div className="relative">
                      <Input
                        type={showAppSecret ? 'text' : 'password'}
                        className="pr-10"
                        placeholder={t('yourAppSecret')}
                        {...field}
                      />
                      <Button
                        type="button"
                        variant="ghost"
                        size="icon"
                        className="absolute right-0 top-0 h-full"
                        onClick={() => setShowAppSecret(!showAppSecret)}
                      >
                        {showAppSecret ? (
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

            {/* Verify Token */}
            <FormField
              control={form.control}
              name="verify_token"
              render={({ field }) => (
                <FormItem>
                  <FormLabel>
                    {t('verifyToken')}
                    <Badge variant="outline" className="ml-2">{t('autoGenerated')}</Badge>
                  </FormLabel>
                  <FormControl>
                    <div className="flex gap-2">
                      <Input {...field} />
                      <Button
                        type="button"
                        variant="outline"
                        size="icon"
                        onClick={() => copyToClipboard(field.value)}
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
                  {t('configureInstagramWebhook')}
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

            <Alert>
              <AlertCircle className="h-4 w-4" />
              <AlertTitle>{t('webhookConfiguration')}</AlertTitle>
              <AlertDescription>
                {t('instagramWebhookInstructions')}
              </AlertDescription>
            </Alert>
          </TabsContent>

          <TabsContent value="setup" className="space-y-4 mt-4">
            <Card>
              <CardHeader>
                <CardTitle>{t('setupGuide')}</CardTitle>
                <CardDescription>
                  {t('instagramSetupDesc')}
                </CardDescription>
              </CardHeader>
              <CardContent className="space-y-4">
                <div className="space-y-4">
                  <SetupStep
                    number={1}
                    title={t('createFacebookApp')}
                    description={t('createFacebookAppDesc')}
                  >
                    <Button variant="outline" size="sm" asChild>
                      <a
                        href="https://developers.facebook.com/apps/create/"
                        target="_blank"
                        rel="noopener noreferrer"
                      >
                        {t('openDeveloperConsole')}
                        <ExternalLink className="h-3 w-3 ml-2" />
                      </a>
                    </Button>
                  </SetupStep>

                  <Separator />

                  <SetupStep
                    number={2}
                    title={t('addInstagramMessaging')}
                    description={t('addInstagramMessagingDesc')}
                  />

                  <Separator />

                  <SetupStep
                    number={3}
                    title={t('connectInstagramBusiness')}
                    description={t('connectInstagramBusinessDesc')}
                  />

                  <Separator />

                  <SetupStep
                    number={4}
                    title={t('generateAccessToken')}
                    description={t('generateAccessTokenDesc')}
                  />

                  <Separator />

                  <SetupStep
                    number={5}
                    title={t('configureWebhookStep')}
                    description={t('configureInstagramWebhookStepDesc')}
                  />
                </div>
              </CardContent>
            </Card>

            <Alert>
              <Instagram className="h-4 w-4" />
              <AlertTitle>{t('requirements')}</AlertTitle>
              <AlertDescription>
                {t('instagramRequirementsDesc')}
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
 * Generate a random verify token
 */
function generateVerifyToken(): string {
  return Math.random().toString(36).substring(2, 15) + Math.random().toString(36).substring(2, 15)
}

/**
 * Instagram Config Dialog
 */
export function InstagramConfigDialog({
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
            {channel ? t('configureInstagramDm') : t('addInstagramDm')}
          </DialogTitle>
          <DialogDescription>
            {t('connectInstagramBusinessAccount')}
          </DialogDescription>
        </DialogHeader>
        <InstagramConfig
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

export default InstagramConfig
