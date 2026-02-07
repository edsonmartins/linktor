'use client'

import { useState, useEffect } from 'react'
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
  Facebook,
  Loader2,
  MessageSquare,
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
 * Facebook Configuration Schema
 */
const facebookConfigSchema = z.object({
  name: z.string().min(1, 'Channel name is required'),
  app_id: z.string().min(1, 'App ID is required'),
  app_secret: z.string().min(1, 'App Secret is required'),
  page_id: z.string().min(1, 'Page ID is required'),
  page_access_token: z.string().min(1, 'Page Access Token is required'),
  verify_token: z.string().min(1, 'Verify Token is required'),
})

type FacebookConfigForm = z.infer<typeof facebookConfigSchema>

interface FacebookConfigProps {
  channel?: Channel
  onSuccess?: (channel: Channel) => void
  onCancel?: () => void
}

/**
 * Facebook Messenger Channel Configuration Component
 */
// OAuth Page type
interface OAuthPage {
  id: string
  name: string
  access_token: string
  category: string
  picture_url?: string
  instagram?: {
    id: string
    username: string
    name: string
    profile_picture_url?: string
    followers_count: number
  }
}

export function FacebookConfig({
  channel,
  onSuccess,
  onCancel,
}: FacebookConfigProps) {
  const { toast } = useToast()
  const [isSubmitting, setIsSubmitting] = useState(false)
  const [showAppSecret, setShowAppSecret] = useState(false)
  const [showPageToken, setShowPageToken] = useState(false)
  const [testStatus, setTestStatus] = useState<'idle' | 'testing' | 'success' | 'error'>('idle')

  // OAuth state
  const [oauthLoading, setOauthLoading] = useState(false)
  const [oauthPages, setOauthPages] = useState<OAuthPage[]>([])
  const [selectedPage, setSelectedPage] = useState<OAuthPage | null>(null)
  const [oauthAppId, setOauthAppId] = useState('')
  const [oauthAppSecret, setOauthAppSecret] = useState('')

  const isEditing = !!channel

  const form = useForm<FacebookConfigForm>({
    resolver: zodResolver(facebookConfigSchema),
    defaultValues: {
      name: channel?.name || '',
      app_id: (channel?.config?.app_id as string) || '',
      app_secret: '',
      page_id: (channel?.config?.page_id as string) || '',
      page_access_token: '',
      verify_token: (channel?.config?.verify_token as string) || generateVerifyToken(),
    },
  })

  const webhookUrl = channel
    ? `${window.location.origin}/api/v1/webhooks/facebook/${channel.id}`
    : 'Will be generated after creation'

  const onSubmit = async (data: FacebookConfigForm) => {
    setIsSubmitting(true)
    try {
      const payload = {
        name: data.name,
        type: 'facebook',
        config: {
          app_id: data.app_id,
          page_id: data.page_id,
          verify_token: data.verify_token,
        },
        credentials: {
          app_secret: data.app_secret,
          page_access_token: data.page_access_token,
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
        description: `Facebook Messenger channel "${data.name}" has been ${isEditing ? 'updated' : 'created'} successfully.`,
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
    if (!values.page_access_token) {
      toast({
        title: 'Missing credentials',
        description: 'Please enter Page Access Token first',
        variant: 'error',
      })
      return
    }

    setTestStatus('testing')
    try {
      await api.post('/channels/test-facebook', {
        page_access_token: values.page_access_token,
        page_id: values.page_id,
      })
      setTestStatus('success')
      toast({
        title: 'Connection successful',
        description: 'Facebook credentials are valid',
      })
    } catch {
      setTestStatus('error')
      toast({
        title: 'Connection failed',
        description: 'Could not connect to Facebook API. Please check your credentials.',
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

  // OAuth: Start Facebook login
  const startOAuthFlow = async () => {
    if (!oauthAppId || !oauthAppSecret) {
      toast({
        title: 'Missing credentials',
        description: 'Please enter App ID and App Secret first',
        variant: 'error',
      })
      return
    }

    setOauthLoading(true)
    try {
      const response = await api.post<{ login_url: string; state: string }>('/oauth/facebook/login', {
        app_id: oauthAppId,
        app_secret: oauthAppSecret,
      })

      // Store state for callback
      sessionStorage.setItem('fb_oauth_state', response.state)
      sessionStorage.setItem('fb_oauth_app_id', oauthAppId)
      sessionStorage.setItem('fb_oauth_app_secret', oauthAppSecret)

      // Open Facebook login in popup
      const popup = window.open(
        response.login_url,
        'facebook_oauth',
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
        title: 'OAuth Error',
        description: error instanceof Error ? error.message : 'Failed to start OAuth flow',
        variant: 'error',
      })
    }
  }

  // OAuth: Handle callback
  const handleOAuthCallback = async (code: string, state: string) => {
    setOauthLoading(true)
    try {
      const appId = sessionStorage.getItem('fb_oauth_app_id') || oauthAppId
      const appSecret = sessionStorage.getItem('fb_oauth_app_secret') || oauthAppSecret

      const response = await api.post<{
        user_access_token: string
        pages: OAuthPage[]
      }>('/oauth/facebook/callback', {
        code,
        state,
        app_id: appId,
        app_secret: appSecret,
      })

      setOauthPages(response.pages)
      if (response.pages.length === 1) {
        setSelectedPage(response.pages[0])
      }

      toast({
        title: 'Connected successfully',
        description: `Found ${response.pages.length} page(s)`,
      })

      // Clean up session storage
      sessionStorage.removeItem('fb_oauth_state')
      sessionStorage.removeItem('fb_oauth_app_id')
      sessionStorage.removeItem('fb_oauth_app_secret')
    } catch (error) {
      toast({
        title: 'OAuth Error',
        description: error instanceof Error ? error.message : 'Failed to complete OAuth',
        variant: 'error',
      })
    } finally {
      setOauthLoading(false)
    }
  }

  // OAuth: Create channel from selected page
  const createFromOAuth = async () => {
    if (!selectedPage) {
      toast({
        title: 'No page selected',
        description: 'Please select a Facebook Page',
        variant: 'error',
      })
      return
    }

    setIsSubmitting(true)
    try {
      const result = await api.post<{ channel: Channel }>('/oauth/channels', {
        name: selectedPage.name,
        type: 'facebook',
        page_id: selectedPage.id,
        access_token: selectedPage.access_token,
        app_id: oauthAppId,
        app_secret: oauthAppSecret,
      })

      toast({
        title: 'Channel created',
        description: `Facebook Page "${selectedPage.name}" connected successfully`,
      })

      onSuccess?.(result.channel)
    } catch (error) {
      toast({
        title: 'Error',
        description: error instanceof Error ? error.message : 'Failed to create channel',
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
            {!isEditing && <TabsTrigger value="oauth">Connect</TabsTrigger>}
            <TabsTrigger value="credentials">Manual</TabsTrigger>
            <TabsTrigger value="webhook">Webhook</TabsTrigger>
            <TabsTrigger value="setup">Setup Guide</TabsTrigger>
          </TabsList>

          {/* OAuth Tab - Recommended for new channels */}
          {!isEditing && (
            <TabsContent value="oauth" className="space-y-4 mt-4">
              <Alert>
                <Facebook className="h-4 w-4" />
                <AlertTitle>Connect with Facebook</AlertTitle>
                <AlertDescription>
                  The easiest way to connect your Facebook Page. Enter your App credentials and click Connect.
                </AlertDescription>
              </Alert>

              <div className="space-y-4">
                {/* App ID for OAuth */}
                <div className="space-y-2">
                  <label className="text-sm font-medium">App ID</label>
                  <Input
                    placeholder="123456789012345"
                    value={oauthAppId}
                    onChange={(e) => setOauthAppId(e.target.value)}
                  />
                  <p className="text-xs text-muted-foreground">
                    From your Facebook App Dashboard
                  </p>
                </div>

                {/* App Secret for OAuth */}
                <div className="space-y-2">
                  <label className="text-sm font-medium">App Secret</label>
                  <div className="relative">
                    <Input
                      type={showAppSecret ? 'text' : 'password'}
                      className="pr-10"
                      placeholder="Your app secret"
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
                      Connecting...
                    </>
                  ) : (
                    <>
                      <Facebook className="h-4 w-4 mr-2" />
                      Connect with Facebook
                    </>
                  )}
                </Button>

                {/* Page Selection */}
                {oauthPages.length > 0 && (
                  <Card>
                    <CardHeader>
                      <CardTitle className="text-base">Select a Page</CardTitle>
                      <CardDescription>
                        Choose which Facebook Page to connect
                      </CardDescription>
                    </CardHeader>
                    <CardContent className="space-y-2">
                      {oauthPages.map((page) => (
                        <div
                          key={page.id}
                          className={`p-3 border rounded-lg cursor-pointer transition-colors ${
                            selectedPage?.id === page.id
                              ? 'border-primary bg-primary/5'
                              : 'hover:border-primary/50'
                          }`}
                          onClick={() => setSelectedPage(page)}
                        >
                          <div className="flex items-center gap-3">
                            {page.picture_url ? (
                              <img
                                src={page.picture_url}
                                alt={page.name}
                                className="w-10 h-10 rounded-full"
                              />
                            ) : (
                              <div className="w-10 h-10 rounded-full bg-blue-500/10 flex items-center justify-center">
                                <Facebook className="h-5 w-5 text-blue-500" />
                              </div>
                            )}
                            <div className="flex-1">
                              <p className="font-medium">{page.name}</p>
                              <p className="text-xs text-muted-foreground">{page.category}</p>
                            </div>
                            {selectedPage?.id === page.id && (
                              <CheckCircle2 className="h-5 w-5 text-primary" />
                            )}
                          </div>
                          {page.instagram && (
                            <div className="mt-2 pt-2 border-t text-xs text-muted-foreground">
                              Connected Instagram: @{page.instagram.username}
                            </div>
                          )}
                        </div>
                      ))}

                      <Button
                        type="button"
                        className="w-full mt-4"
                        onClick={createFromOAuth}
                        disabled={!selectedPage || isSubmitting}
                      >
                        {isSubmitting ? (
                          <>
                            <Loader2 className="h-4 w-4 mr-2 animate-spin" />
                            Creating...
                          </>
                        ) : (
                          'Create Channel'
                        )}
                      </Button>
                    </CardContent>
                  </Card>
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
                  <FormLabel>Channel Name</FormLabel>
                  <FormControl>
                    <Input placeholder="My Facebook Page" {...field} />
                  </FormControl>
                  <FormDescription>
                    A friendly name to identify this channel
                  </FormDescription>
                  <FormMessage />
                </FormItem>
              )}
            />

            {/* App ID */}
            <FormField
              control={form.control}
              name="app_id"
              render={({ field }) => (
                <FormItem>
                  <FormLabel>App ID</FormLabel>
                  <FormControl>
                    <Input placeholder="123456789012345" {...field} />
                  </FormControl>
                  <FormDescription>
                    Your Facebook App ID from the developer dashboard
                  </FormDescription>
                  <FormMessage />
                </FormItem>
              )}
            />

            {/* App Secret */}
            <FormField
              control={form.control}
              name="app_secret"
              render={({ field }) => (
                <FormItem>
                  <FormLabel>App Secret</FormLabel>
                  <FormControl>
                    <div className="relative">
                      <Input
                        type={showAppSecret ? 'text' : 'password'}
                        className="pr-10"
                        placeholder={isEditing ? '••••••••••••••••' : 'Your app secret'}
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
                    App Secret from Settings &gt; Basic in your Facebook App
                  </FormDescription>
                  <FormMessage />
                </FormItem>
              )}
            />

            {/* Page ID */}
            <FormField
              control={form.control}
              name="page_id"
              render={({ field }) => (
                <FormItem>
                  <FormLabel>Page ID</FormLabel>
                  <FormControl>
                    <Input placeholder="123456789012345" {...field} />
                  </FormControl>
                  <FormDescription>
                    Your Facebook Page ID
                  </FormDescription>
                  <FormMessage />
                </FormItem>
              )}
            />

            {/* Page Access Token */}
            <FormField
              control={form.control}
              name="page_access_token"
              render={({ field }) => (
                <FormItem>
                  <FormLabel>Page Access Token</FormLabel>
                  <FormControl>
                    <div className="relative">
                      <MessageSquare className="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-muted-foreground" />
                      <Input
                        type={showPageToken ? 'text' : 'password'}
                        className="pl-10 pr-10"
                        placeholder={isEditing ? '••••••••••••••••' : 'EAABsbCS...'}
                        {...field}
                      />
                      <Button
                        type="button"
                        variant="ghost"
                        size="icon"
                        className="absolute right-0 top-0 h-full"
                        onClick={() => setShowPageToken(!showPageToken)}
                      >
                        {showPageToken ? (
                          <EyeOff className="h-4 w-4" />
                        ) : (
                          <Eye className="h-4 w-4" />
                        )}
                      </Button>
                    </div>
                  </FormControl>
                  <FormDescription>
                    Page Access Token with messaging permissions
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
                    Verify Token
                    <Badge variant="outline" className="ml-2">Auto-generated</Badge>
                  </FormLabel>
                  <FormControl>
                    <div className="flex gap-2">
                      <Input {...field} />
                      <Button
                        type="button"
                        variant="outline"
                        size="icon"
                        onClick={() => copyToClipboard(field.value, 'Verify Token')}
                      >
                        <Copy className="h-4 w-4" />
                      </Button>
                    </div>
                  </FormControl>
                  <FormDescription>
                    Used to verify webhook requests from Facebook
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
            {/* Webhook URL */}
            <Card>
              <CardHeader>
                <CardTitle className="text-base flex items-center gap-2">
                  <Webhook className="h-4 w-4" />
                  Webhook URL
                </CardTitle>
                <CardDescription>
                  Configure this URL in your Facebook App webhooks settings
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

            <Alert>
              <AlertCircle className="h-4 w-4" />
              <AlertTitle>Webhook Configuration</AlertTitle>
              <AlertDescription>
                In your Facebook App settings, add this webhook URL and select the following fields:
                messages, messaging_postbacks, message_deliveries, message_reads
              </AlertDescription>
            </Alert>
          </TabsContent>

          <TabsContent value="setup" className="space-y-4 mt-4">
            <Card>
              <CardHeader>
                <CardTitle>Setup Guide</CardTitle>
                <CardDescription>
                  Follow these steps to configure Facebook Messenger integration
                </CardDescription>
              </CardHeader>
              <CardContent className="space-y-4">
                <div className="space-y-4">
                  <SetupStep
                    number={1}
                    title="Create a Facebook App"
                    description="Go to Facebook for Developers and create a new app"
                  >
                    <Button variant="outline" size="sm" asChild>
                      <a
                        href="https://developers.facebook.com/apps/create/"
                        target="_blank"
                        rel="noopener noreferrer"
                      >
                        Open Developer Console
                        <ExternalLink className="h-3 w-3 ml-2" />
                      </a>
                    </Button>
                  </SetupStep>

                  <Separator />

                  <SetupStep
                    number={2}
                    title="Add Messenger Product"
                    description="In your app dashboard, add the Messenger product"
                  />

                  <Separator />

                  <SetupStep
                    number={3}
                    title="Generate Page Access Token"
                    description="In Messenger Settings, connect your Page and generate a Page Access Token"
                  />

                  <Separator />

                  <SetupStep
                    number={4}
                    title="Configure Webhook"
                    description="Add the webhook URL and verify token from the Webhook tab above"
                  />

                  <Separator />

                  <SetupStep
                    number={5}
                    title="Subscribe to Events"
                    description="Subscribe your webhook to messages, messaging_postbacks, message_deliveries, and message_reads events"
                  />
                </div>
              </CardContent>
            </Card>

            <Alert>
              <MessageSquare className="h-4 w-4" />
              <AlertTitle>Permissions Required</AlertTitle>
              <AlertDescription>
                Your app needs pages_messaging permission. You may need to complete App Review
                for production use.
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
 * Generate a random verify token
 */
function generateVerifyToken(): string {
  return Math.random().toString(36).substring(2, 15) + Math.random().toString(36).substring(2, 15)
}

/**
 * Facebook Config Dialog
 */
export function FacebookConfigDialog({
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
            {channel ? 'Configure Facebook Messenger' : 'Add Facebook Messenger'}
          </DialogTitle>
          <DialogDescription>
            Connect your Facebook Page to receive Messenger conversations
          </DialogDescription>
        </DialogHeader>
        <FacebookConfig
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

export default FacebookConfig
