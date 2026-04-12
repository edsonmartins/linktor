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
  const tSignup = useTranslations('channels.config.embeddedSignup')
  const { toast } = useToast()

  // Form state
  const [appId, setAppId] = useState('')
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
    if (!appId) {
      toast({
        title: t('error'),
        description: tSignup('enterAppIdFirst'),
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
        throw new Error(tSignup('popupBlocked'))
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
      setError(err instanceof Error ? err.message : tSignup('failedStartOauth'))
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

    if (code && state === savedState && savedAppId) {
      try {
        const response = await api.post<EmbeddedSignupResponse>(
          '/oauth/whatsapp/embedded-signup/callback',
          {
            code,
            state,
            app_id: savedAppId,
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
        setError(err instanceof Error ? err.message : tSignup('failedCompleteOauth'))
        setStep('credentials')
      }
    } else {
      setStep('credentials')
    }

    setIsLoading(false)

    // Clean up session storage (no app_secret stored)
    sessionStorage.removeItem('wa_embedded_signup_state')
    sessionStorage.removeItem('wa_embedded_signup_app_id')
  }, [t, tSignup, toast])

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

    if (!appId) {
      setError(tSignup('appIdRequired'))
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
          verify_token: oauthData.verify_token,
          is_coexistence: oauthData.is_coexistence,
          quality_rating: oauthData.quality_rating,
        }
      )

      toast({
        title: t('channelCreated'),
        description: t('channelCreatedDesc', { name: channelName }),
      })
      onSuccess?.(response.channel)
    } catch (err) {
      setError(err instanceof Error ? err.message : tSignup('failedCreateChannel'))
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
              {tSignup('connectExistingNumber')}
            </CardTitle>
            <CardDescription>
              {tSignup('connectExistingDesc')}
            </CardDescription>
          </CardHeader>
          <CardContent className="space-y-4">
            {/* Benefits */}
            <div className="bg-muted/50 p-4 rounded-lg space-y-2">
              <h4 className="font-medium text-sm">{tSignup('benefitsTitle')}</h4>
              <ul className="text-sm text-muted-foreground space-y-1">
                <li className="flex items-center gap-2">
                  <CheckCircle2 className="h-4 w-4 text-green-500" />
                  {tSignup('benefitKeepUsing')}
                </li>
                <li className="flex items-center gap-2">
                  <CheckCircle2 className="h-4 w-4 text-green-500" />
                  {tSignup('benefitKeepNumber')}
                </li>
                <li className="flex items-center gap-2">
                  <CheckCircle2 className="h-4 w-4 text-green-500" />
                  {tSignup('benefitSync')}
                </li>
                <li className="flex items-center gap-2">
                  <CheckCircle2 className="h-4 w-4 text-green-500" />
                  {tSignup('benefitNoMigration')}
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

            <Alert>
              <AlertCircle className="h-4 w-4" />
              <AlertTitle>{tSignup('appSecretServerSide')}</AlertTitle>
              <AlertDescription>
                {tSignup('appSecretServerSideDesc')}
              </AlertDescription>
            </Alert>

            <div className="space-y-2">
              <Label htmlFor="configId">
                Embedded Signup Config ID
                <Badge variant="outline" className="ml-2">{t('optional')}</Badge>
              </Label>
              <Input
                id="configId"
                value={configId}
                onChange={(e) => setConfigId(e.target.value)}
                placeholder={tSignup('optionalFromMeta')}
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
                  {tSignup('connecting')}
                </>
              ) : (
                <>
                  <ExternalLink className="h-4 w-4 mr-2" />
                  {tSignup('connectWithFacebook')}
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
              {tSignup('connectingToWhatsApp')}
            </CardTitle>
            <CardDescription>
              {tSignup('completeLoginPopup')}
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
                {tSignup('waitingForAuth')}
              </p>
            </div>
          </CardContent>
          <CardFooter>
            <Button variant="outline" onClick={() => setStep('credentials')} className="w-full">
              {t('cancel')}
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
              {tSignup('whatsappConnected')}
            </CardTitle>
            <CardDescription>
              {tSignup('accountConnectedDesc')}
            </CardDescription>
          </CardHeader>
          <CardContent className="space-y-4">
            {/* Connection Details */}
            <div className="bg-muted/50 p-4 rounded-lg space-y-3">
              <div className="flex justify-between">
                <span className="text-sm text-muted-foreground">{tSignup('phoneNumberLabel')}</span>
                <span className="text-sm font-medium">{oauthData.phone_number}</span>
              </div>
              {oauthData.business_name && (
                <div className="flex justify-between">
                  <span className="text-sm text-muted-foreground">{tSignup('businessNameLabel')}</span>
                  <span className="text-sm font-medium">{oauthData.business_name}</span>
                </div>
              )}
              <div className="flex justify-between">
                <span className="text-sm text-muted-foreground">{tSignup('coexistenceLabel')}</span>
                <Badge variant={oauthData.is_coexistence ? 'default' : 'secondary'}>
                  {oauthData.is_coexistence ? (
                    <>
                      <Wifi className="h-3 w-3 mr-1" />
                      {tSignup('enabled')}
                    </>
                  ) : (
                    <>
                      <WifiOff className="h-3 w-3 mr-1" />
                      {tSignup('notAvailable')}
                    </>
                  )}
                </Badge>
              </div>
              {oauthData.quality_rating && (
                <div className="flex justify-between">
                  <span className="text-sm text-muted-foreground">{tSignup('qualityRatingLabel')}</span>
                  <Badge variant="outline">{oauthData.quality_rating}</Badge>
                </div>
              )}
            </div>

            {oauthData.is_coexistence && (
              <Alert>
                <Smartphone className="h-4 w-4" />
                <AlertTitle>{tSignup('coexistenceModeActive')}</AlertTitle>
                <AlertDescription>
                  {tSignup('coexistenceModeActiveDesc')}
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
              {tSignup('back')}
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
                {tSignup('creatingChannelWebhooks')}
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
