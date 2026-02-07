'use client'

import { useState } from 'react'
import { useMutation, useQueryClient } from '@tanstack/react-query'
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
 * Voice Provider configurations
 */
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
    description: 'Cloud-based voice with TwiML support',
    fields: [
      { key: 'account_sid', label: 'Account SID', type: 'text', placeholder: 'ACxxxxxxxx', required: true },
      { key: 'auth_token', label: 'Auth Token', type: 'password', placeholder: 'Your auth token', required: true },
      { key: 'phone_number', label: 'Phone Number', type: 'tel', placeholder: '+1234567890', required: true },
      { key: 'webhook_url', label: 'Webhook URL', type: 'url', placeholder: 'https://your-domain.com/voice/webhook', required: false },
    ],
  },
  vonage: {
    label: 'Vonage Voice',
    description: 'Nexmo/Vonage Voice API with NCCO',
    fields: [
      { key: 'api_key', label: 'API Key', type: 'text', placeholder: 'Your API key', required: true },
      { key: 'api_secret', label: 'API Secret', type: 'password', placeholder: 'Your API secret', required: true },
      { key: 'application_id', label: 'Application ID', type: 'text', placeholder: 'Application UUID', required: true },
      { key: 'private_key', label: 'Private Key', type: 'password', placeholder: 'Paste private key or path', required: true },
      { key: 'phone_number', label: 'Phone Number', type: 'tel', placeholder: '+1234567890', required: true },
    ],
  },
  amazon_connect: {
    label: 'Amazon Connect',
    description: 'AWS Contact Center service',
    fields: [
      { key: 'instance_id', label: 'Instance ID', type: 'text', placeholder: 'Connect instance ID', required: true },
      { key: 'region', label: 'AWS Region', type: 'text', placeholder: 'us-east-1', required: true },
      { key: 'access_key_id', label: 'Access Key ID', type: 'text', placeholder: 'AKIAXXXXXXXX', required: true },
      { key: 'secret_access_key', label: 'Secret Access Key', type: 'password', placeholder: 'Your secret key', required: true },
      { key: 'contact_flow_id', label: 'Contact Flow ID', type: 'text', placeholder: 'Contact flow UUID', required: true },
      { key: 'queue_id', label: 'Queue ID', type: 'text', placeholder: 'Queue UUID', required: false },
    ],
  },
  asterisk: {
    label: 'Asterisk',
    description: 'Open-source PBX with AMI/ARI',
    fields: [
      { key: 'host', label: 'Host', type: 'text', placeholder: 'asterisk.example.com', required: true },
      { key: 'ami_port', label: 'AMI Port', type: 'number', placeholder: '5038', required: true },
      { key: 'ami_username', label: 'AMI Username', type: 'text', placeholder: 'admin', required: true },
      { key: 'ami_password', label: 'AMI Password', type: 'password', placeholder: 'Password', required: true },
      { key: 'ari_port', label: 'ARI Port', type: 'number', placeholder: '8088', required: false },
      { key: 'ari_username', label: 'ARI Username', type: 'text', placeholder: 'ari_user', required: false },
      { key: 'ari_password', label: 'ARI Password', type: 'password', placeholder: 'ARI password', required: false },
    ],
  },
  freeswitch: {
    label: 'FreeSWITCH',
    description: 'Scalable open-source telephony platform',
    fields: [
      { key: 'esl_host', label: 'ESL Host', type: 'text', placeholder: 'freeswitch.example.com', required: true },
      { key: 'esl_port', label: 'ESL Port', type: 'number', placeholder: '8021', required: true },
      { key: 'esl_password', label: 'ESL Password', type: 'password', placeholder: 'ClueCon', required: true },
      { key: 'gateway', label: 'Gateway Name', type: 'text', placeholder: 'default', required: false },
      { key: 'socket_port', label: 'Socket Port', type: 'number', placeholder: '8085', required: false },
      { key: 'recordings_url', label: 'Recordings URL', type: 'url', placeholder: 'http://localhost/recordings', required: false },
    ],
  },
}

/**
 * Voice Channel Configuration Component
 */
export function VoiceConfig({ channel, onClose }: VoiceConfigProps) {
  const queryClient = useQueryClient()
  const isEditing = !!channel

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
    : 'Will be generated after saving'

  return (
    <form onSubmit={handleSubmit} className="flex flex-col h-full">
      <div className="flex-1 space-y-6">
      <Tabs defaultValue="credentials">
        <TabsList className="grid w-full grid-cols-3">
          <TabsTrigger value="credentials">
            <Key className="h-4 w-4 mr-2" />
            Credentials
          </TabsTrigger>
          <TabsTrigger value="settings">
            <Settings className="h-4 w-4 mr-2" />
            Settings
          </TabsTrigger>
          <TabsTrigger value="webhook">
            <Globe className="h-4 w-4 mr-2" />
            Webhook
          </TabsTrigger>
        </TabsList>

        {/* Credentials Tab */}
        <TabsContent value="credentials" className="space-y-4 mt-4">
          <div className="space-y-2">
            <Label htmlFor="name">Channel Name</Label>
            <Input
              id="name"
              placeholder="e.g., Main Voice Line"
              value={formData.name}
              onChange={(e) => setFormData({ ...formData, name: e.target.value })}
              required
            />
          </div>

          <div className="space-y-2">
            <Label htmlFor="provider">Voice Provider</Label>
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
              {currentProviderConfig.label} Configuration
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
                  Testing Connection...
                </>
              ) : testStatus === 'success' ? (
                <>
                  <CheckCircle2 className="h-4 w-4 mr-2 text-green-500" />
                  Connection Successful
                </>
              ) : testStatus === 'error' ? (
                <>
                  <XCircle className="h-4 w-4 mr-2 text-destructive" />
                  Connection Failed
                </>
              ) : (
                <>
                  <TestTube className="h-4 w-4 mr-2" />
                  Test Connection
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
                Recording Settings
              </CardTitle>
              <CardDescription>Configure call recording and transcription</CardDescription>
            </CardHeader>
            <CardContent className="space-y-4">
              <div className="flex items-center justify-between">
                <div>
                  <Label>Record Calls</Label>
                  <p className="text-xs text-muted-foreground">
                    Automatically record all calls for quality and training
                  </p>
                </div>
                <Button
                  type="button"
                  variant={formData.record_calls ? 'default' : 'outline'}
                  size="sm"
                  onClick={() => setFormData({ ...formData, record_calls: !formData.record_calls })}
                >
                  {formData.record_calls ? 'Enabled' : 'Disabled'}
                </Button>
              </div>

              <div className="flex items-center justify-between">
                <div>
                  <Label>Transcribe Calls</Label>
                  <p className="text-xs text-muted-foreground">
                    Convert speech to text for analysis and search
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
                  {formData.transcribe_calls ? 'Enabled' : 'Disabled'}
                </Button>
              </div>
            </CardContent>
          </Card>

          <Card>
            <CardHeader>
              <CardTitle className="text-base flex items-center gap-2">
                <Volume2 className="h-4 w-4" />
                IVR Features
              </CardTitle>
              <CardDescription>Interactive Voice Response capabilities</CardDescription>
            </CardHeader>
            <CardContent>
              <div className="grid grid-cols-2 gap-4 text-sm">
                <div className="flex items-center gap-2">
                  <CheckCircle2 className="h-4 w-4 text-green-500" />
                  <span>Text-to-Speech (TTS)</span>
                </div>
                <div className="flex items-center gap-2">
                  <CheckCircle2 className="h-4 w-4 text-green-500" />
                  <span>Speech-to-Text (STT)</span>
                </div>
                <div className="flex items-center gap-2">
                  <CheckCircle2 className="h-4 w-4 text-green-500" />
                  <span>DTMF Input</span>
                </div>
                <div className="flex items-center gap-2">
                  <CheckCircle2 className="h-4 w-4 text-green-500" />
                  <span>Call Transfer</span>
                </div>
                <div className="flex items-center gap-2">
                  <CheckCircle2 className="h-4 w-4 text-green-500" />
                  <span>Call Queue</span>
                </div>
                <div className="flex items-center gap-2">
                  <CheckCircle2 className="h-4 w-4 text-green-500" />
                  <span>Conference</span>
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
                Webhook Configuration
              </CardTitle>
              <CardDescription>
                Configure your voice provider to send events to this URL
              </CardDescription>
            </CardHeader>
            <CardContent className="space-y-4">
              <div className="space-y-2">
                <Label>Webhook URL</Label>
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
                <AlertTitle>Security</AlertTitle>
                <AlertDescription>
                  Configure webhook signature verification in your provider's dashboard for secure
                  communication.
                </AlertDescription>
              </Alert>
            </CardContent>
          </Card>

          <Card>
            <CardHeader>
              <CardTitle className="text-base">Setup Guide</CardTitle>
            </CardHeader>
            <CardContent className="space-y-3 text-sm">
              {formData.provider === 'twilio' && (
                <ol className="list-decimal list-inside space-y-2">
                  <li>Log in to your Twilio Console</li>
                  <li>Go to Phone Numbers → Manage → Active Numbers</li>
                  <li>Select your phone number</li>
                  <li>Under Voice & Fax, set "A call comes in" to Webhook</li>
                  <li>Paste the webhook URL above</li>
                  <li>Set HTTP method to POST</li>
                </ol>
              )}
              {formData.provider === 'vonage' && (
                <ol className="list-decimal list-inside space-y-2">
                  <li>Log in to your Vonage Dashboard</li>
                  <li>Go to Applications</li>
                  <li>Select or create your Voice application</li>
                  <li>Under Capabilities → Voice, set the Answer URL</li>
                  <li>Paste the webhook URL above</li>
                  <li>Set the Event URL for call status updates</li>
                </ol>
              )}
              {formData.provider === 'amazon_connect' && (
                <ol className="list-decimal list-inside space-y-2">
                  <li>Open Amazon Connect in AWS Console</li>
                  <li>Select your instance</li>
                  <li>Go to Contact flows</li>
                  <li>Create or edit a contact flow</li>
                  <li>Add "Invoke AWS Lambda function" block</li>
                  <li>Configure Lambda to forward to webhook URL</li>
                </ol>
              )}
              {formData.provider === 'asterisk' && (
                <ol className="list-decimal list-inside space-y-2">
                  <li>Edit your Asterisk dialplan (extensions.conf)</li>
                  <li>Add AGI or ARI application to handle calls</li>
                  <li>Configure the application to connect to webhook URL</li>
                  <li>Enable AMI for management commands</li>
                  <li>Configure manager.conf with credentials above</li>
                </ol>
              )}
              {formData.provider === 'freeswitch' && (
                <ol className="list-decimal list-inside space-y-2">
                  <li>Edit /etc/freeswitch/autoload_configs/event_socket.conf.xml</li>
                  <li>Configure ESL password to match above</li>
                  <li>Add outbound socket in dialplan for webhook</li>
                  <li>Restart FreeSWITCH to apply changes</li>
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
                  View Documentation
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
          Cancel
        </Button>
        <Button type="submit" disabled={saveMutation.isPending || !formData.name}>
          {saveMutation.isPending ? 'Saving...' : isEditing ? 'Update Channel' : 'Create Channel'}
        </Button>
      </div>
    </form>
  )
}
