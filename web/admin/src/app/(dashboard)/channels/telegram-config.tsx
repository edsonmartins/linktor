'use client'

import { useState } from 'react'
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
  Bot,
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
 * Telegram Configuration Schema
 */
const telegramConfigSchema = z.object({
  name: z.string().min(1, 'Channel name is required'),
  bot_token: z.string().min(1, 'Bot token is required'),
  bot_name: z.string().optional(),
})

type TelegramConfigForm = z.infer<typeof telegramConfigSchema>

interface TelegramConfigProps {
  channel?: Channel
  onSuccess?: (channel: Channel) => void
  onCancel?: () => void
}

/**
 * Telegram Channel Configuration Component
 */
export function TelegramConfig({
  channel,
  onSuccess,
  onCancel,
}: TelegramConfigProps) {
  const t = useTranslations('channels.config')
  const tCommon = useTranslations('common')
  const { toast } = useToast()
  const [isSubmitting, setIsSubmitting] = useState(false)
  const [showBotToken, setShowBotToken] = useState(false)
  const [testStatus, setTestStatus] = useState<'idle' | 'testing' | 'success' | 'error'>('idle')

  const isEditing = !!channel

  const form = useForm<TelegramConfigForm>({
    resolver: zodResolver(telegramConfigSchema),
    defaultValues: {
      name: channel?.name || '',
      bot_token: '',
      bot_name: (channel?.config?.bot_name as string) || '',
    },
  })

  const webhookUrl = channel
    ? `${window.location.origin}/api/v1/webhooks/telegram/${channel.id}`
    : t('willBeGenerated')

  const onSubmit = async (data: TelegramConfigForm) => {
    setIsSubmitting(true)
    try {
      const payload = {
        name: data.name,
        type: 'telegram',
        config: {
          bot_name: data.bot_name,
        },
        credentials: {
          bot_token: data.bot_token,
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
    if (!values.bot_token) {
      toast({
        title: t('missingCredentials'),
        description: t('enterCredentialsFirst'),
        variant: 'error',
      })
      return
    }

    setTestStatus('testing')
    try {
      await api.post('/channels/test-telegram', {
        bot_token: values.bot_token,
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

  const copyToClipboard = (text: string, label: string) => {
    navigator.clipboard.writeText(text)
    toast({
      title: t('copied'),
      description: t('copiedToClipboard', { label }),
    })
  }

  return (
    <Form {...form}>
      <form onSubmit={form.handleSubmit(onSubmit)} className="flex flex-col h-full">
        <div className="flex-1 space-y-6">
        <Tabs defaultValue="credentials" className="w-full">
          <TabsList className="grid w-full grid-cols-3">
            <TabsTrigger value="credentials">{t('credentials')}</TabsTrigger>
            <TabsTrigger value="webhook">{t('webhook')}</TabsTrigger>
            <TabsTrigger value="setup">{t('setupGuide')}</TabsTrigger>
          </TabsList>

          <TabsContent value="credentials" className="space-y-4 mt-4">
            {/* Channel Name */}
            <FormField
              control={form.control}
              name="name"
              render={({ field }) => (
                <FormItem>
                  <FormLabel>{t('channelName')}</FormLabel>
                  <FormControl>
                    <Input placeholder={t('myTelegramBot')} {...field} />
                  </FormControl>
                  <FormDescription>
                    {t('channelNameDesc')}
                  </FormDescription>
                  <FormMessage />
                </FormItem>
              )}
            />

            {/* Bot Token */}
            <FormField
              control={form.control}
              name="bot_token"
              render={({ field }) => (
                <FormItem>
                  <FormLabel>{t('botToken')}</FormLabel>
                  <FormControl>
                    <div className="relative">
                      <Bot className="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-muted-foreground" />
                      <Input
                        type={showBotToken ? 'text' : 'password'}
                        className="pl-10 pr-10"
                        placeholder={isEditing ? '••••••••••••••••' : t('botTokenPlaceholder')}
                        {...field}
                      />
                      <Button
                        type="button"
                        variant="ghost"
                        size="icon"
                        className="absolute right-0 top-0 h-full"
                        onClick={() => setShowBotToken(!showBotToken)}
                      >
                        {showBotToken ? (
                          <EyeOff className="h-4 w-4" />
                        ) : (
                          <Eye className="h-4 w-4" />
                        )}
                      </Button>
                    </div>
                  </FormControl>
                  <FormDescription>
                    {t('botTokenDesc')}
                  </FormDescription>
                  <FormMessage />
                </FormItem>
              )}
            />

            {/* Bot Name (Optional) */}
            <FormField
              control={form.control}
              name="bot_name"
              render={({ field }) => (
                <FormItem>
                  <FormLabel>
                    {t('botUsername')}
                    <Badge variant="outline" className="ml-2">{t('optional')}</Badge>
                  </FormLabel>
                  <FormControl>
                    <Input placeholder={t('botUsernamePlaceholder')} {...field} />
                  </FormControl>
                  <FormDescription>
                    {t('botUsernameDesc')}
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
                  {t('webhookAutoConfigured')}
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
                      onClick={() => copyToClipboard(webhookUrl, t('webhookUrl'))}
                    >
                      <Copy className="h-4 w-4" />
                    </Button>
                  )}
                </div>
              </CardContent>
            </Card>

            <Alert>
              <AlertCircle className="h-4 w-4" />
              <AlertTitle>{t('webhookAutoSetup')}</AlertTitle>
              <AlertDescription>
                {t('webhookAutoSetupDesc')}
              </AlertDescription>
            </Alert>
          </TabsContent>

          <TabsContent value="setup" className="space-y-4 mt-4">
            <Card>
              <CardHeader>
                <CardTitle>{t('setupGuideTitle')}</CardTitle>
                <CardDescription>
                  {t('telegramSetupGuide')}
                </CardDescription>
              </CardHeader>
              <CardContent className="space-y-4">
                <div className="space-y-4">
                  <SetupStep
                    number={1}
                    title={t('telegramStep1Title')}
                    description={t('telegramStep1Desc')}
                  >
                    <Button variant="outline" size="sm" asChild>
                      <a
                        href="https://t.me/BotFather"
                        target="_blank"
                        rel="noopener noreferrer"
                      >
                        {t('openBotFather')}
                        <ExternalLink className="h-3 w-3 ml-2" />
                      </a>
                    </Button>
                  </SetupStep>

                  <Separator />

                  <SetupStep
                    number={2}
                    title={t('telegramStep2Title')}
                    description={t('telegramStep2Desc')}
                  />

                  <Separator />

                  <SetupStep
                    number={3}
                    title={t('telegramStep3Title')}
                    description={t('telegramStep3Desc')}
                  />

                  <Separator />

                  <SetupStep
                    number={4}
                    title={t('telegramStep4Title')}
                    description={t('telegramStep4Desc')}
                  />

                  <Separator />

                  <SetupStep
                    number={5}
                    title={t('telegramStep5Title')}
                    description={t('telegramStep5Desc')}
                  />
                </div>
              </CardContent>
            </Card>

            <Alert>
              <Bot className="h-4 w-4" />
              <AlertTitle>{t('botCommands')}</AlertTitle>
              <AlertDescription>
                {t('botCommandsDesc')}
              </AlertDescription>
            </Alert>
          </TabsContent>
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

/**
 * Telegram Config Dialog
 */
export function TelegramConfigDialog({
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
              ? tChannels('configureChannel', { channel: 'Telegram' })
              : tChannels('addChannelType', { channel: 'Telegram' })}
          </DialogTitle>
          <DialogDescription>
            {channel
              ? tChannels('updateSettings', { channel: 'Telegram' })
              : tChannels('setupNewChannel', { channel: 'Telegram' })}
          </DialogDescription>
        </DialogHeader>
        <TelegramConfig
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

export default TelegramConfig
