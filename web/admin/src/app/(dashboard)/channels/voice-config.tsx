'use client'

import { useState } from 'react'
import { useMutation, useQueryClient } from '@tanstack/react-query'
import { useTranslations } from 'next-intl'
import {
  Phone,
  PhoneCall,
  PhoneOff,
  Settings,
  Mic,
  Volume2,
  Globe,
  Server,
  Key,
  Shield,
  TestTube,
  Copy,
  ExternalLink,
  CheckCircle2,
  XCircle,
  Loader2,
} from 'lucide-react'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { Alert, AlertDescription, AlertTitle } from '@/components/ui/alert'
import { Separator } from '@/components/ui/separator'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { api } from '@/lib/api'
import { queryKeys } from '@/lib/query'
import type { Channel, VoiceProvider } from '@/types'

interface VoiceConfigProps {
  channel?: Channel
  onClose: () => void
}

/**
 * Voice Channel Configuration Component
 */
export function VoiceConfig({ channel, onClose }: VoiceConfigProps) {
  const t = useTranslations('channels.config')
  const tCommon = useTranslations('common')
  const queryClient = useQueryClient()
  const isEditing = !!channel

  // Voice Provider configurations with translations
  const voiceProviderConfigs: Record<
    VoiceProvider,
    {
      label: string
      description: string
      fields: { key: string; label: string; type: string; placeholder: string; required: boolean }[]
    }
  > = {
    twilio: {
      label: 'Twilio Voice',
      description: t('twilioVoiceDesc'),
      fields: [
        { key: 'account_sid', label: t('accountSid'), type: 'text', placeholder: 'ACxxxxxxxx', required: true },
        { key: 'auth_token', label: t('authToken'), type: 'password', placeholder: t('authTokenPlaceholder'), required: true },
        { key: 'phone_number', label: t('phoneNumber'), type: 'tel', placeholder: '+1234567890', required: true },
        { key: 'webhook_url', label: t('webhookUrl'), type: 'url', placeholder: 'https://your-domain.com/voice/webhook', required: false },
      ],
    },
    vonage: {
      label: 'Vonage Voice',
      description: t('vonageVoiceDesc'),
      fields: [
        { key: 'api_key', label: t('apiKeyLabel'), type: 'text', placeholder: t('apiKeyPlaceholder'), required: true },
        { key: 'api_secret', label: t('apiSecretLabel'), type: 'password', placeholder: t('apiSecretPlaceholder'), required: true },
        { key: 'application_id', label: t('applicationId'), type: 'text', placeholder: t('applicationIdPlaceholder'), required: true },
        { key: 'private_key', label: t('privateKey'), type: 'password', placeholder: t('privateKeyPlaceholder'), required: true },
        { key: 'phone_number', label: t('phoneNumber'), type: 'tel', placeholder: '+1234567890', required: true },
      ],
    },
    amazon_connect: {
      label: 'Amazon Connect',
      description: t('amazonConnectDesc'),
      fields: [
        { key: 'instance_id', label: t('instanceId'), type: 'text', placeholder: t('instanceIdPlaceholder'), required: true },
        { key: 'region', label: t('awsRegion'), type: 'text', placeholder: 'us-east-1', required: true },
        { key: 'access_key_id', label: t('accessKeyId'), type: 'text', placeholder: 'AKIAXXXXXXXX', required: true },
        { key: 'secret_access_key', label: t('secretAccessKey'), type: 'password', placeholder: t('secretKeyPlaceholder'), required: true },
        { key: 'contact_flow_id', label: t('contactFlowId'), type: 'text', placeholder: t('contactFlowIdPlaceholder'), required: true },
        { key: 'queue_id', label: t('queueId'), type: 'text', placeholder: t('queueIdPlaceholder'), required: false },
      ],
    },
    asterisk: {
      label: 'Asterisk',
      description: t('asteriskDesc'),
      fields: [
        { key: 'host', label: t('hostLabel'), type: 'text', placeholder: 'asterisk.example.com', required: true },
        { key: 'ami_port', label: t('amiPort'), type: 'number', placeholder: '5038', required: true },
        { key: 'ami_username', label: t('amiUsername'), type: 'text', placeholder: 'admin', required: true },
        { key: 'ami_password', label: t('amiPassword'), type: 'password', placeholder: t('passwordPlaceholder'), required: true },
        { key: 'ari_port', label: t('ariPort'), type: 'number', placeholder: '8088', required: false },
        { key: 'ari_username', label: t('ariUsername'), type: 'text', placeholder: 'ari_user', required: false },
        { key: 'ari_password', label: t('ariPassword'), type: 'password', placeholder: t('ariPasswordPlaceholder'), required: false },
      ],
    },
    freeswitch: {
      label: 'FreeSWITCH',
      description: t('freeswitchDesc'),
      fields: [
        { key: 'esl_host', label: t('eslHost'), type: 'text', placeholder: 'freeswitch.example.com', required: true },
        { key: 'esl_port', label: t('eslPort'), type: 'number', placeholder: '8021', required: true },
        { key: 'esl_password', label: t('eslPassword'), type: 'password', placeholder: 'ClueCon', required: true },
        { key: 'gateway', label: t('gatewayName'), type: 'text', placeholder: 'default', required: false },
        { key: 'socket_port', label: t('socketPort'), type: 'number', placeholder: '8085', required: false },
        { key: 'recordings_url', label: t('recordingsUrl'), type: 'url', placeholder: 'http://localhost/recordings', required: false },
      ],
    },
  }

  const [formData, setFormData] = useState({
    name: channel?.name || '',
    provider: (channel?.config?.provider as VoiceProvider) || 'twilio',
    credentials: (channel?.config?.credentials as Record<string, string>) || {},
    record_calls: (channel?.config?.record_calls as boolean) || false,
    transcribe_calls: (channel?.config?.transcribe_calls as boolean) || false,
  })

  const [testStatus, setTestStatus] = useState<'idle' | 'testing' | 'success' | 'error'>('idle')
  const [testError, setTestError] = useState<string>('')

  const currentProviderConfig = voiceProviderConfigs[formData.provider]

  // Create/Update mutation
  const saveMutation = useMutation({
    mutationFn: async (data: typeof formData) => {
      const payload = {
        name: data.name,
        type: 'voice',
        config: {
          provider: data.provider,
          credentials: data.credentials,
          record_calls: data.record_calls,
          transcribe_calls: data.transcribe_calls,
        },
      }

      if (isEditing) {
        return api.put(`/channels/${channel.id}`, payload)
      }
      return api.post('/channels', payload)
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: queryKeys.channels.all })
      onClose()
    },
  })

  // Test connection mutation
  const testMutation = useMutation({
    mutationFn: async () => {
      const payload = {
        type: 'voice',
        config: {
          provider: formData.provider,
          credentials: formData.credentials,
        },
      }
      return api.post('/channels/test', payload)
    },
    onSuccess: () => {
      setTestStatus('success')
    },
    onError: (error: Error) => {
      setTestStatus('error')
      setTestError(error.message)
    },
  })

  const handleTest = () => {
    setTestStatus('testing')
    setTestError('')
    testMutation.mutate()
  }

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault()
    saveMutation.mutate(formData)
  }

  const updateCredential = (key: string, value: string) => {
    setFormData({
      ...formData,
      credentials: {
        ...formData.credentials,
        [key]: value,
      },
    })
  }

  const webhookUrl = channel?.id
    ? `${process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8081'}/api/v1/webhooks/voice/${channel.id}`
    : t('willBeGenerated')

  return (
    <form onSubmit={handleSubmit} className="flex flex-col h-full">
      <div className="flex-1 space-y-6">
      <Tabs defaultValue="credentials">
        <TabsList className="grid w-full grid-cols-3">
          <TabsTrigger value="credentials">
            <Key className="h-4 w-4 mr-2" />
            {t('credentials')}
          </TabsTrigger>
          <TabsTrigger value="settings">
            <Settings className="h-4 w-4 mr-2" />
            {tCommon('settings')}
          </TabsTrigger>
          <TabsTrigger value="webhook">
            <Globe className="h-4 w-4 mr-2" />
            {t('webhook')}
          </TabsTrigger>
        </TabsList>

        {/* Credentials Tab */}
        <TabsContent value="credentials" className="space-y-4 mt-4">
          <div className="space-y-2">
            <Label htmlFor="name">{t('channelName')}</Label>
            <Input
              id="name"
              placeholder={t('mainVoiceLine')}
              value={formData.name}
              onChange={(e) => setFormData({ ...formData, name: e.target.value })}
              required
            />
          </div>

          <div className="space-y-2">
            <Label htmlFor="provider">{t('voiceProvider')}</Label>
            <Select
              value={formData.provider}
              onValueChange={(value: VoiceProvider) =>
                setFormData({ ...formData, provider: value, credentials: {} })
              }
            >
              <SelectTrigger>
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                {Object.entries(voiceProviderConfigs).map(([key, config]) => (
                  <SelectItem key={key} value={key}>
                    <div className="flex items-center gap-2">
                      <Phone className="h-4 w-4" />
                      <div>
                        <span className="font-medium">{config.label}</span>
                        <span className="text-xs text-muted-foreground ml-2">
                          {config.description}
                        </span>
                      </div>
                    </div>
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          </div>

          <Separator />

          <div className="space-y-4">
            <h4 className="font-medium flex items-center gap-2">
              <Server className="h-4 w-4" />
              {t('providerConfiguration', { provider: currentProviderConfig.label })}
            </h4>

            {currentProviderConfig.fields.map((field) => (
              <div key={field.key} className="space-y-2">
                <Label htmlFor={field.key}>
                  {field.label}
                  {field.required && <span className="text-destructive ml-1">*</span>}
                </Label>
                <Input
                  id={field.key}
                  type={field.type}
                  placeholder={field.placeholder}
                  value={formData.credentials[field.key] || ''}
                  onChange={(e) => updateCredential(field.key, e.target.value)}
                  required={field.required}
                />
              </div>
            ))}
          </div>

          {/* Test Connection */}
          <div className="pt-4">
            <Button
              type="button"
              variant="outline"
              onClick={handleTest}
              disabled={testStatus === 'testing'}
              className="w-full"
            >
              {testStatus === 'testing' ? (
                <>
                  <Loader2 className="h-4 w-4 mr-2 animate-spin" />
                  {t('testing')}
                </>
              ) : testStatus === 'success' ? (
                <>
                  <CheckCircle2 className="h-4 w-4 mr-2 text-green-500" />
                  {t('connectionSuccess')}
                </>
              ) : testStatus === 'error' ? (
                <>
                  <XCircle className="h-4 w-4 mr-2 text-destructive" />
                  {t('testFailed')}
                </>
              ) : (
                <>
                  <TestTube className="h-4 w-4 mr-2" />
                  {t('testConnection')}
                </>
              )}
            </Button>
            {testStatus === 'error' && testError && (
              <p className="text-sm text-destructive mt-2">{testError}</p>
            )}
          </div>
        </TabsContent>

        {/* Settings Tab */}
        <TabsContent value="settings" className="space-y-4 mt-4">
          <Card>
            <CardHeader>
              <CardTitle className="text-base flex items-center gap-2">
                <Mic className="h-4 w-4" />
                {t('recordingSettings')}
              </CardTitle>
              <CardDescription>{t('configureRecording')}</CardDescription>
            </CardHeader>
            <CardContent className="space-y-4">
              <div className="flex items-center justify-between">
                <div>
                  <Label>{t('recordCalls')}</Label>
                  <p className="text-xs text-muted-foreground">
                    {t('recordCallsDesc')}
                  </p>
                </div>
                <Button
                  type="button"
                  variant={formData.record_calls ? 'default' : 'outline'}
                  size="sm"
                  onClick={() => setFormData({ ...formData, record_calls: !formData.record_calls })}
                >
                  {formData.record_calls ? tCommon('enabled') : tCommon('disabled')}
                </Button>
              </div>

              <div className="flex items-center justify-between">
                <div>
                  <Label>{t('transcribeCalls')}</Label>
                  <p className="text-xs text-muted-foreground">
                    {t('transcribeCallsDesc')}
                  </p>
                </div>
                <Button
                  type="button"
                  variant={formData.transcribe_calls ? 'default' : 'outline'}
                  size="sm"
                  onClick={() =>
                    setFormData({ ...formData, transcribe_calls: !formData.transcribe_calls })
                  }
                >
                  {formData.transcribe_calls ? tCommon('enabled') : tCommon('disabled')}
                </Button>
              </div>
            </CardContent>
          </Card>

          <Card>
            <CardHeader>
              <CardTitle className="text-base flex items-center gap-2">
                <Volume2 className="h-4 w-4" />
                {t('ivrFeatures')}
              </CardTitle>
              <CardDescription>{t('ivrFeaturesDesc')}</CardDescription>
            </CardHeader>
            <CardContent>
              <div className="grid grid-cols-2 gap-4 text-sm">
                <div className="flex items-center gap-2">
                  <CheckCircle2 className="h-4 w-4 text-green-500" />
                  <span>{t('tts')}</span>
                </div>
                <div className="flex items-center gap-2">
                  <CheckCircle2 className="h-4 w-4 text-green-500" />
                  <span>{t('stt')}</span>
                </div>
                <div className="flex items-center gap-2">
                  <CheckCircle2 className="h-4 w-4 text-green-500" />
                  <span>{t('dtmfInput')}</span>
                </div>
                <div className="flex items-center gap-2">
                  <CheckCircle2 className="h-4 w-4 text-green-500" />
                  <span>{t('callTransfer')}</span>
                </div>
                <div className="flex items-center gap-2">
                  <CheckCircle2 className="h-4 w-4 text-green-500" />
                  <span>{t('callQueue')}</span>
                </div>
                <div className="flex items-center gap-2">
                  <CheckCircle2 className="h-4 w-4 text-green-500" />
                  <span>{t('conference')}</span>
                </div>
              </div>
            </CardContent>
          </Card>
        </TabsContent>

        {/* Webhook Tab */}
        <TabsContent value="webhook" className="space-y-4 mt-4">
          <Card>
            <CardHeader>
              <CardTitle className="text-base flex items-center gap-2">
                <Globe className="h-4 w-4" />
                {t('voiceWebhookConfig')}
              </CardTitle>
              <CardDescription>
                {t('configureVoiceProvider')}
              </CardDescription>
            </CardHeader>
            <CardContent className="space-y-4">
              <div className="space-y-2">
                <Label>{t('webhookUrl')}</Label>
                <div className="flex gap-2">
                  <Input value={webhookUrl} readOnly className="font-mono text-sm" />
                  <Button
                    type="button"
                    variant="outline"
                    size="icon"
                    onClick={() => navigator.clipboard.writeText(webhookUrl)}
                  >
                    <Copy className="h-4 w-4" />
                  </Button>
                </div>
              </div>

              <Alert>
                <Shield className="h-4 w-4" />
                <AlertTitle>{t('webhookSecurity')}</AlertTitle>
                <AlertDescription>
                  {t('webhookSecurityDesc')}
                </AlertDescription>
              </Alert>
            </CardContent>
          </Card>

          <Card>
            <CardHeader>
              <CardTitle className="text-base">{t('setupGuide')}</CardTitle>
            </CardHeader>
            <CardContent className="space-y-3 text-sm">
              {formData.provider === 'twilio' && (
                <ol className="list-decimal list-inside space-y-2">
                  <li>{t('twilioVoiceSetup1')}</li>
                  <li>{t('twilioVoiceSetup2')}</li>
                  <li>{t('twilioVoiceSetup3')}</li>
                  <li>{t('twilioVoiceSetup4')}</li>
                  <li>{t('twilioVoiceSetup5')}</li>
                  <li>{t('twilioVoiceSetup6')}</li>
                </ol>
              )}
              {formData.provider === 'vonage' && (
                <ol className="list-decimal list-inside space-y-2">
                  <li>{t('vonageSetup1')}</li>
                  <li>{t('vonageSetup2')}</li>
                  <li>{t('vonageSetup3')}</li>
                  <li>{t('vonageSetup4')}</li>
                  <li>{t('vonageSetup5')}</li>
                  <li>{t('vonageSetup6')}</li>
                </ol>
              )}
              {formData.provider === 'amazon_connect' && (
                <ol className="list-decimal list-inside space-y-2">
                  <li>{t('amazonConnectSetup1')}</li>
                  <li>{t('amazonConnectSetup2')}</li>
                  <li>{t('amazonConnectSetup3')}</li>
                  <li>{t('amazonConnectSetup4')}</li>
                  <li>{t('amazonConnectSetup5')}</li>
                  <li>{t('amazonConnectSetup6')}</li>
                </ol>
              )}
              {formData.provider === 'asterisk' && (
                <ol className="list-decimal list-inside space-y-2">
                  <li>{t('asteriskSetup1')}</li>
                  <li>{t('asteriskSetup2')}</li>
                  <li>{t('asteriskSetup3')}</li>
                  <li>{t('asteriskSetup4')}</li>
                  <li>{t('asteriskSetup5')}</li>
                </ol>
              )}
              {formData.provider === 'freeswitch' && (
                <ol className="list-decimal list-inside space-y-2">
                  <li>{t('freeswitchSetup1')}</li>
                  <li>{t('freeswitchSetup2')}</li>
                  <li>{t('freeswitchSetup3')}</li>
                  <li>{t('freeswitchSetup4')}</li>
                </ol>
              )}
              <Button variant="link" className="h-auto p-0 mt-2" asChild>
                <a
                  href={
                    formData.provider === 'twilio'
                      ? 'https://www.twilio.com/docs/voice'
                      : formData.provider === 'vonage'
                        ? 'https://developer.vonage.com/voice/voice-api/overview'
                        : formData.provider === 'amazon_connect'
                          ? 'https://docs.aws.amazon.com/connect/'
                          : formData.provider === 'asterisk'
                            ? 'https://wiki.asterisk.org/'
                            : 'https://freeswitch.org/confluence/'
                  }
                  target="_blank"
                  rel="noopener noreferrer"
                >
                  <ExternalLink className="h-4 w-4 mr-1" />
                  {t('viewDocumentation')}
                </a>
              </Button>
            </CardContent>
          </Card>
        </TabsContent>
      </Tabs>
      </div>

      {/* Form Actions */}
      <div className="sticky bottom-0 flex justify-end gap-3 pt-4 pb-2 mt-4 border-t bg-background">
        <Button type="button" variant="outline" onClick={onClose}>
          {tCommon('cancel')}
        </Button>
        <Button type="submit" disabled={saveMutation.isPending || !formData.name}>
          {saveMutation.isPending ? t('saving') : isEditing ? t('updateChannel') : t('createChannel')}
        </Button>
      </div>
    </form>
  )
}
