'use client'

import { useState } from 'react'
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
    : 'Will be generated after creation'

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
        title: isEditing ? 'Channel updated' : 'Channel created',
        description: `Telegram channel "${data.name}" has been ${isEditing ? 'updated' : 'created'} successfully.`,
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
    if (!values.bot_token) {
      toast({
        title: 'Missing credentials',
        description: 'Please enter bot token first',
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
        title: 'Connection successful',
        description: 'Telegram bot credentials are valid',
      })
    } catch {
      setTestStatus('error')
      toast({
        title: 'Connection failed',
        description: 'Could not connect to Telegram API. Please check your bot token.',
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

  return (
    <Form {...form}>
      <form onSubmit={form.handleSubmit(onSubmit)} className="flex flex-col h-full">
        <div className="flex-1 space-y-6">
        <Tabs defaultValue="credentials" className="w-full">
          <TabsList className="grid w-full grid-cols-3">
            <TabsTrigger value="credentials">Credentials</TabsTrigger>
            <TabsTrigger value="webhook">Webhook</TabsTrigger>
            <TabsTrigger value="setup">Setup Guide</TabsTrigger>
          </TabsList>

          <TabsContent value="credentials" className="space-y-4 mt-4">
            {/* Channel Name */}
            <FormField
              control={form.control}
              name="name"
              render={({ field }) => (
                <FormItem>
                  <FormLabel>Channel Name</FormLabel>
                  <FormControl>
                    <Input placeholder="My Telegram Bot" {...field} />
                  </FormControl>
                  <FormDescription>
                    A friendly name to identify this channel
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
                  <FormLabel>Bot Token</FormLabel>
                  <FormControl>
                    <div className="relative">
                      <Bot className="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-muted-foreground" />
                      <Input
                        type={showBotToken ? 'text' : 'password'}
                        className="pl-10 pr-10"
                        placeholder={isEditing ? '••••••••••••••••' : '123456789:ABCdef...'}
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
                    Bot token from @BotFather
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
                    Bot Username
                    <Badge variant="outline" className="ml-2">Optional</Badge>
                  </FormLabel>
                  <FormControl>
                    <Input placeholder="my_company_bot" {...field} />
                  </FormControl>
                  <FormDescription>
                    Your bot username (without @). Will be auto-detected if not provided.
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
                  This URL will be automatically configured when you save the channel
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
              <AlertTitle>Automatic Webhook Setup</AlertTitle>
              <AlertDescription>
                Linktor will automatically configure the webhook with Telegram when you save this channel.
                You don&apos;t need to manually set up the webhook.
              </AlertDescription>
            </Alert>
          </TabsContent>

          <TabsContent value="setup" className="space-y-4 mt-4">
            <Card>
              <CardHeader>
                <CardTitle>Setup Guide</CardTitle>
                <CardDescription>
                  Follow these steps to create a Telegram Bot
                </CardDescription>
              </CardHeader>
              <CardContent className="space-y-4">
                <div className="space-y-4">
                  <SetupStep
                    number={1}
                    title="Open BotFather"
                    description="Start a chat with @BotFather on Telegram"
                  >
                    <Button variant="outline" size="sm" asChild>
                      <a
                        href="https://t.me/BotFather"
                        target="_blank"
                        rel="noopener noreferrer"
                      >
                        Open BotFather
                        <ExternalLink className="h-3 w-3 ml-2" />
                      </a>
                    </Button>
                  </SetupStep>

                  <Separator />

                  <SetupStep
                    number={2}
                    title="Create New Bot"
                    description="Send /newbot command and follow the prompts to create your bot"
                  />

                  <Separator />

                  <SetupStep
                    number={3}
                    title="Copy Bot Token"
                    description="BotFather will give you a token that looks like 123456789:ABCdefGHIjklmNOPQrstUVwxyz"
                  />

                  <Separator />

                  <SetupStep
                    number={4}
                    title="Configure Bot Settings (Optional)"
                    description="Use /setcommands, /setdescription, and /setabouttext to customize your bot"
                  />

                  <Separator />

                  <SetupStep
                    number={5}
                    title="Paste Token Here"
                    description="Enter the bot token in the Credentials tab above"
                  />
                </div>
              </CardContent>
            </Card>

            <Alert>
              <Bot className="h-4 w-4" />
              <AlertTitle>Bot Commands</AlertTitle>
              <AlertDescription>
                You can configure bot commands with BotFather using /setcommands.
                Common commands include /start for greeting new users and /help for assistance.
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

  return (
    <Dialog open={open} onOpenChange={setOpen}>
      <DialogTrigger asChild>{trigger}</DialogTrigger>
      <DialogContent className="max-w-2xl max-h-[90vh] overflow-auto">
        <DialogHeader>
          <DialogTitle>
            {channel ? 'Configure Telegram Channel' : 'Add Telegram Channel'}
          </DialogTitle>
          <DialogDescription>
            Connect your Telegram Bot using the Bot API
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
