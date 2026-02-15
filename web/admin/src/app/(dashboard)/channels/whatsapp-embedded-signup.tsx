'use client'

import { useState, useEffect, useCallback } from 'react'
import { useTranslations } from 'next-intl'
import {
  AlertCircle,
  CheckCircle2,
  Loader2,
  Smartphone,
  Wifi,
  WifiOff,
  ExternalLink,
} from 'lucide-react'
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
  Alert,
  AlertDescription,
  AlertTitle,
} from '@/components/ui/alert'
import { Badge } from '@/components/ui/badge'
import { Separator } from '@/components/ui/separator'
import { Progress } from '@/components/ui/progress'
import { useToast } from '@/hooks/use-toast'
import { api } from '@/lib/api'
import type { Channel } from '@/types'

interface EmbeddedSignupProps {
  onSuccess?: (channel: Channel) => void
  onCancel?: () => void
}

interface EmbeddedSignupResponse {
  access_token: string
  waba_id: string
  phone_number_id: string
  phone_number: string
  is_coexistence: boolean
  coexistence_status: string
  business_name?: string
  quality_rating?: string
  verify_token: string
  webhook_url: string
  subscribed_fields: string[]
}

/**
 * WhatsApp Embedded Signup Component for Coexistence
 * Allows connecting existing WhatsApp Business App numbers
 */
export function WhatsAppEmbeddedSignup({
  onSuccess,
  onCancel,
}: EmbeddedSignupProps) {
  const t = useTranslations('channels.config')
  const { toast } = useToast()

  // Form state
  const [appId, setAppId] = useState('')
  const [appSecret, setAppSecret] = useState('')
  const [configId, setConfigId] = useState('')
  const [channelName, setChannelName] = useState('')

  // Flow state
  const [step, setStep] = useState<'credentials' | 'connecting' | 'connected' | 'creating'>('credentials')
  const [isLoading, setIsLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)

  // OAuth response
  const [oauthData, setOauthData] = useState<EmbeddedSignupResponse | null>(null)

  // Start OAuth flow
  const startEmbeddedSignup = async () => {
    if (!appId || !appSecret) {
      toast({
        title: t('error'),
        description: t('enterAppCredentialsFirst'),
        variant: 'error',
      })
      return
    }

    setIsLoading(true)
    setError(null)

    try {
      // Get login URL from backend
      const response = await api.post<{ login_url: string; state: string }>(
        '/oauth/whatsapp/embedded-signup/start',
        {
          app_id: appId,
          config_id: configId || undefined,
          redirect_url: window.location.href,
        }
      )

      // Store only state and app_id in session storage (NOT app_secret)
      // App secret remains only in component state
      sessionStorage.setItem('wa_embedded_signup_state', response.state)
      sessionStorage.setItem('wa_embedded_signup_app_id', appId)

      // Open OAuth popup
      const width = 600
      const height = 700
      const left = window.screenX + (window.outerWidth - width) / 2
      const top = window.screenY + (window.outerHeight - height) / 2

      const popup = window.open(
        response.login_url,
        'WhatsApp Embedded Signup',
        `width=${width},height=${height},left=${left},top=${top},scrollbars=yes`
      )

      if (!popup) {
        throw new Error('Popup blocked. Please allow popups for this site.')
      }

      setStep('connecting')

      // Poll for popup close
      const checkPopup = setInterval(() => {
        if (popup.closed) {
          clearInterval(checkPopup)
          handlePopupClosed()
        }
      }, 500)
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to start OAuth flow')
      setIsLoading(false)
    }
  }

  // Handle popup closed (check for OAuth callback)
  const handlePopupClosed = useCallback(async () => {
    // Check URL for OAuth code
    const urlParams = new URLSearchParams(window.location.search)
    const code = urlParams.get('code')
    const state = urlParams.get('state')

    const savedState = sessionStorage.getItem('wa_embedded_signup_state')
    const savedAppId = sessionStorage.getItem('wa_embedded_signup_app_id')

    // Use appSecret from component state (more secure than sessionStorage)
    if (code && state === savedState && savedAppId && appSecret) {
      try {
        // Exchange code for token
        const response = await api.post<EmbeddedSignupResponse>(
          '/oauth/whatsapp/embedded-signup/callback',
          {
            code,
            state,
            app_id: savedAppId,
            app_secret: appSecret, // From component state, not sessionStorage
          }
        )

        setOauthData(response)
        setStep('connected')
        setChannelName(response.business_name || 'WhatsApp Business')

        // Clear URL params
        window.history.replaceState({}, '', window.location.pathname)

        toast({
          title: t('connectedSuccessfully'),
          description: response.is_coexistence
            ? 'Coexistence enabled! Your Business App is connected.'
            : 'WhatsApp connected successfully.',
        })
      } catch (err) {
        setError(err instanceof Error ? err.message : 'Failed to complete OAuth')
        setStep('credentials')
      }
    } else if (code && state === savedState && savedAppId && !appSecret) {
      // App secret was lost (page reload) - need to re-enter
      setError('Session expired. Please enter your App Secret again and retry.')
      setStep('credentials')
    } else {
      // OAuth was cancelled or failed
      setStep('credentials')
    }

    setIsLoading(false)

    // Clean up session storage (no app_secret stored)
    sessionStorage.removeItem('wa_embedded_signup_state')
    sessionStorage.removeItem('wa_embedded_signup_app_id')
  }, [t, toast, appSecret])

  // Check for OAuth callback on mount
  useEffect(() => {
    const urlParams = new URLSearchParams(window.location.search)
    if (urlParams.get('code')) {
      handlePopupClosed()
    }
  }, [handlePopupClosed])

  // Create channel from OAuth data
  const createChannel = async () => {
    if (!oauthData || !channelName) return

    // Validate app credentials are still available
    if (!appId || !appSecret) {
      setError('App credentials are required. Please go back and re-enter them.')
      setStep('credentials')
      return
    }

    setIsLoading(true)
    setStep('creating')

    try {
      const response = await api.post<{ channel: Channel }>(
        '/oauth/whatsapp/embedded-signup/create-channel',
        {
          name: channelName,
          access_token: oauthData.access_token,
          waba_id: oauthData.waba_id,
          phone_number_id: oauthData.phone_number_id,
          phone_number: oauthData.phone_number,
          app_id: appId,
          app_secret: appSecret,
          verify_token: oauthData.verify_token,
          is_coexistence: oauthData.is_coexistence,
        }
      )

      toast({
        title: t('channelCreated'),
        description: t('channelCreatedDesc', { name: channelName }),
      })

      // Clear sensitive data from state after successful creation
      setAppSecret('')

      onSuccess?.(response.channel)
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to create channel')
      setStep('connected')
    } finally {
      setIsLoading(false)
    }
  }

  return (
    <div className="space-y-6">
      {/* Error Alert */}
      {error && (
        <Alert variant="destructive">
          <AlertCircle className="h-4 w-4" />
          <AlertTitle>{t('error')}</AlertTitle>
          <AlertDescription>{error}</AlertDescription>
        </Alert>
      )}

      {/* Step 1: Credentials */}
      {step === 'credentials' && (
        <Card>
          <CardHeader>
            <CardTitle className="flex items-center gap-2">
              <Smartphone className="h-5 w-5" />
              Connect Existing WhatsApp Number
            </CardTitle>
            <CardDescription>
              Connect your existing WhatsApp Business App number to use both the app and API simultaneously (Coexistence).
            </CardDescription>
          </CardHeader>
          <CardContent className="space-y-4">
            {/* Benefits */}
            <div className="bg-muted/50 p-4 rounded-lg space-y-2">
              <h4 className="font-medium text-sm">Benefits of Coexistence:</h4>
              <ul className="text-sm text-muted-foreground space-y-1">
                <li className="flex items-center gap-2">
                  <CheckCircle2 className="h-4 w-4 text-green-500" />
                  Keep using WhatsApp Business App on your phone
                </li>
                <li className="flex items-center gap-2">
                  <CheckCircle2 className="h-4 w-4 text-green-500" />
                  Import up to 6 months of chat history
                </li>
                <li className="flex items-center gap-2">
                  <CheckCircle2 className="h-4 w-4 text-green-500" />
                  Messages sync between App and API
                </li>
                <li className="flex items-center gap-2">
                  <CheckCircle2 className="h-4 w-4 text-green-500" />
                  No number migration required
                </li>
              </ul>
            </div>

            <Separator />

            {/* App ID */}
            <div className="space-y-2">
              <Label htmlFor="appId">{t('facebookAppId')}</Label>
              <Input
                id="appId"
                value={appId}
                onChange={(e) => setAppId(e.target.value)}
                placeholder="123456789012345"
              />
              <p className="text-xs text-muted-foreground">
                {t('facebookAppIdDesc')}
              </p>
            </div>

            {/* App Secret */}
            <div className="space-y-2">
              <Label htmlFor="appSecret">{t('appSecret')}</Label>
              <Input
                id="appSecret"
                type="password"
                value={appSecret}
                onChange={(e) => setAppSecret(e.target.value)}
                placeholder="••••••••••••••••"
              />
              <p className="text-xs text-muted-foreground">
                {t('appSecretDesc')}
              </p>
            </div>

            {/* Config ID (Optional) */}
            <div className="space-y-2">
              <Label htmlFor="configId">
                Embedded Signup Config ID
                <Badge variant="outline" className="ml-2">{t('optional')}</Badge>
              </Label>
              <Input
                id="configId"
                value={configId}
                onChange={(e) => setConfigId(e.target.value)}
                placeholder="Optional - from Meta dashboard"
              />
            </div>
          </CardContent>
          <CardFooter className="flex justify-between">
            {onCancel && (
              <Button variant="outline" onClick={onCancel}>
                {t('cancel')}
              </Button>
            )}
            <Button onClick={startEmbeddedSignup} disabled={!appId || isLoading}>
              {isLoading ? (
                <>
                  <Loader2 className="h-4 w-4 mr-2 animate-spin" />
                  Connecting...
                </>
              ) : (
                <>
                  <ExternalLink className="h-4 w-4 mr-2" />
                  Connect with Facebook
                </>
              )}
            </Button>
          </CardFooter>
        </Card>
      )}

      {/* Step 2: Connecting */}
      {step === 'connecting' && (
        <Card>
          <CardHeader>
            <CardTitle className="flex items-center gap-2">
              <Loader2 className="h-5 w-5 animate-spin" />
              Connecting to WhatsApp...
            </CardTitle>
            <CardDescription>
              Complete the login in the popup window. This window will update automatically.
            </CardDescription>
          </CardHeader>
          <CardContent className="space-y-4">
            <div className="flex flex-col items-center gap-4 py-8">
              <div className="animate-pulse flex space-x-2">
                <div className="h-3 w-3 bg-primary rounded-full"></div>
                <div className="h-3 w-3 bg-primary rounded-full animation-delay-200"></div>
                <div className="h-3 w-3 bg-primary rounded-full animation-delay-400"></div>
              </div>
              <p className="text-sm text-muted-foreground">
                Waiting for authentication...
              </p>
            </div>
          </CardContent>
          <CardFooter>
            <Button variant="outline" onClick={() => setStep('credentials')} className="w-full">
              Cancel
            </Button>
          </CardFooter>
        </Card>
      )}

      {/* Step 3: Connected - Create Channel */}
      {step === 'connected' && oauthData && (
        <Card>
          <CardHeader>
            <CardTitle className="flex items-center gap-2">
              <CheckCircle2 className="h-5 w-5 text-green-500" />
              WhatsApp Connected!
            </CardTitle>
            <CardDescription>
              Your WhatsApp account has been connected. Configure your channel below.
            </CardDescription>
          </CardHeader>
          <CardContent className="space-y-4">
            {/* Connection Details */}
            <div className="bg-muted/50 p-4 rounded-lg space-y-3">
              <div className="flex justify-between">
                <span className="text-sm text-muted-foreground">Phone Number:</span>
                <span className="text-sm font-medium">{oauthData.phone_number}</span>
              </div>
              {oauthData.business_name && (
                <div className="flex justify-between">
                  <span className="text-sm text-muted-foreground">Business Name:</span>
                  <span className="text-sm font-medium">{oauthData.business_name}</span>
                </div>
              )}
              <div className="flex justify-between">
                <span className="text-sm text-muted-foreground">Coexistence:</span>
                <Badge variant={oauthData.is_coexistence ? 'default' : 'secondary'}>
                  {oauthData.is_coexistence ? (
                    <>
                      <Wifi className="h-3 w-3 mr-1" />
                      Enabled
                    </>
                  ) : (
                    <>
                      <WifiOff className="h-3 w-3 mr-1" />
                      Not Available
                    </>
                  )}
                </Badge>
              </div>
              {oauthData.quality_rating && (
                <div className="flex justify-between">
                  <span className="text-sm text-muted-foreground">Quality Rating:</span>
                  <Badge variant="outline">{oauthData.quality_rating}</Badge>
                </div>
              )}
            </div>

            {oauthData.is_coexistence && (
              <Alert>
                <Smartphone className="h-4 w-4" />
                <AlertTitle>Coexistence Mode Active</AlertTitle>
                <AlertDescription>
                  Messages sent from the WhatsApp Business App will sync to this platform.
                  Remember to open the app at least once every 14 days to maintain the connection.
                </AlertDescription>
              </Alert>
            )}

            <Separator />

            {/* Channel Name */}
            <div className="space-y-2">
              <Label htmlFor="channelName">{t('channelName')}</Label>
              <Input
                id="channelName"
                value={channelName}
                onChange={(e) => setChannelName(e.target.value)}
                placeholder={t('channelNamePlaceholder')}
              />
              <p className="text-xs text-muted-foreground">
                {t('channelNameDesc')}
              </p>
            </div>
          </CardContent>
          <CardFooter className="flex justify-between">
            <Button variant="outline" onClick={() => setStep('credentials')}>
              Back
            </Button>
            <Button onClick={createChannel} disabled={!channelName || isLoading}>
              {isLoading ? (
                <>
                  <Loader2 className="h-4 w-4 mr-2 animate-spin" />
                  {t('creating')}
                </>
              ) : (
                t('createChannel')
              )}
            </Button>
          </CardFooter>
        </Card>
      )}

      {/* Step 4: Creating */}
      {step === 'creating' && (
        <Card>
          <CardContent className="py-8">
            <div className="flex flex-col items-center gap-4">
              <Loader2 className="h-8 w-8 animate-spin text-primary" />
              <p className="text-sm text-muted-foreground">
                Creating channel and configuring webhooks...
              </p>
            </div>
          </CardContent>
        </Card>
      )}
    </div>
  )
}

// Re-export CoexistenceStatusWidget from the dedicated component file
export { CoexistenceStatusWidget, CoexistenceStatusBadge } from '@/components/coexistence-status-widget'

export default WhatsAppEmbeddedSignup
