'use client'

import { useState } from 'react'
import { useForm } from 'react-hook-form'
import { zodResolver } from '@hookform/resolvers/zod'
import { z } from 'zod'
import {
  Copy,
  Check,
  Loader2,
  Code,
  Palette,
  MessageSquare,
} from 'lucide-react'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Textarea } from '@/components/ui/textarea'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Switch } from '@/components/ui/switch'
import { useToast } from '@/hooks/use-toast'
import { api } from '@/lib/api'
import type { Channel } from '@/types'

const webchatConfigSchema = z.object({
  name: z.string().min(1, 'Channel name is required'),
  // Appearance
  primary_color: z.string(),
  text_color: z.string(),
  position: z.enum(['bottom-right', 'bottom-left']),
  // Messages
  welcome_message: z.string().optional(),
  placeholder_text: z.string().optional(),
  // Behavior
  auto_open: z.boolean(),
  auto_open_delay: z.coerce.number().min(0),
  show_typing_indicator: z.boolean(),
  // Allowed domains
  allowed_domains: z.string().optional(),
})

type WebchatConfigForm = z.infer<typeof webchatConfigSchema>

interface WebchatConfigProps {
  channel?: Channel
  onSuccess?: (channel: Channel) => void
  onCancel?: () => void
}

export function WebchatConfig({ channel, onSuccess, onCancel }: WebchatConfigProps) {
  const { toast } = useToast()
  const [isSubmitting, setIsSubmitting] = useState(false)
  const [copied, setCopied] = useState(false)

  const isEditing = !!channel

  const {
    register,
    handleSubmit,
    watch,
    setValue,
    formState: { errors },
  } = useForm<WebchatConfigForm>({
    resolver: zodResolver(webchatConfigSchema),
    defaultValues: {
      name: channel?.name || '',
      primary_color: (channel?.config?.primary_color as string) || '#6366f1',
      text_color: (channel?.config?.text_color as string) || '#ffffff',
      position: (channel?.config?.position as 'bottom-right' | 'bottom-left') || 'bottom-right',
      welcome_message: (channel?.config?.welcome_message as string) || '',
      placeholder_text: (channel?.config?.placeholder_text as string) || 'Type a message...',
      auto_open: (channel?.config?.auto_open as boolean) || false,
      auto_open_delay: (channel?.config?.auto_open_delay as number) || 3,
      show_typing_indicator: (channel?.config?.show_typing_indicator as boolean) ?? true,
      allowed_domains: (channel?.config?.allowed_domains as string) || '',
    },
  })

  const primaryColor = watch('primary_color')
  const position = watch('position')
  const autoOpen = watch('auto_open')

  const onSubmit = async (data: WebchatConfigForm) => {
    setIsSubmitting(true)

    try {
      const payload = {
        name: data.name,
        type: 'webchat',
        config: {
          primary_color: data.primary_color,
          text_color: data.text_color,
          position: data.position,
          welcome_message: data.welcome_message,
          placeholder_text: data.placeholder_text,
          auto_open: data.auto_open,
          auto_open_delay: data.auto_open_delay,
          show_typing_indicator: data.show_typing_indicator,
          allowed_domains: data.allowed_domains,
        },
      }

      let result: Channel
      if (isEditing) {
        result = await api.put<Channel>(`/channels/${channel.id}`, payload)
        toast({
          title: 'Channel updated',
          description: 'WebChat configuration has been updated.',
        })
      } else {
        result = await api.post<Channel>('/channels', payload)
        toast({
          title: 'Channel created',
          description: 'WebChat channel has been created successfully.',
        })
      }

      onSuccess?.(result)
    } catch (error: unknown) {
      const message = error instanceof Error ? error.message : 'Failed to save channel configuration.'
      toast({
        title: 'Error',
        description: message,
        variant: 'error',
      })
    } finally {
      setIsSubmitting(false)
    }
  }

  const copyEmbedCode = () => {
    const channelId = channel?.id || '{CHANNEL_ID}'
    const embedCode = `<script>
  (function(w, d, s, o, f, js, fjs) {
    w['LinktorWidget'] = o;
    w[o] = w[o] || function() { (w[o].q = w[o].q || []).push(arguments) };
    js = d.createElement(s); fjs = d.getElementsByTagName(s)[0];
    js.id = o; js.src = f; js.async = 1; fjs.parentNode.insertBefore(js, fjs);
  }(window, document, 'script', 'linktor', '${typeof window !== 'undefined' ? window.location.origin : ''}/widget/linktor.js'));
  linktor('init', { channelId: '${channelId}' });
</script>`
    navigator.clipboard.writeText(embedCode)
    setCopied(true)
    setTimeout(() => setCopied(false), 2000)
  }

  return (
    <form onSubmit={handleSubmit(onSubmit)} className="flex flex-col h-full">
      <div className="flex-1 space-y-6">
        <div className="space-y-4">
          <div className="space-y-2">
            <Label htmlFor="name">Channel Name</Label>
            <Input
              id="name"
              placeholder="My Website Chat"
              {...register('name')}
            />
            {errors.name && (
              <p className="text-sm text-destructive">{errors.name.message}</p>
            )}
          </div>
        </div>

        <Tabs defaultValue="appearance" className="w-full">
        <TabsList className="grid w-full grid-cols-3">
          <TabsTrigger value="appearance">Appearance</TabsTrigger>
          <TabsTrigger value="behavior">Behavior</TabsTrigger>
          <TabsTrigger value="embed">Embed Code</TabsTrigger>
        </TabsList>

        <TabsContent value="appearance" className="space-y-4 pt-4">
          <Card>
            <CardHeader>
              <CardTitle className="text-base flex items-center gap-2">
                <Palette className="h-4 w-4" />
                Widget Appearance
              </CardTitle>
              <CardDescription>Customize how the chat widget looks</CardDescription>
            </CardHeader>
            <CardContent className="space-y-4">
              <div className="grid grid-cols-2 gap-4">
                <div className="space-y-2">
                  <Label htmlFor="primary_color">Primary Color</Label>
                  <div className="flex gap-2">
                    <Input
                      id="primary_color"
                      type="color"
                      className="w-12 h-10 p-1 cursor-pointer"
                      {...register('primary_color')}
                    />
                    <Input
                      value={primaryColor}
                      onChange={(e) => setValue('primary_color', e.target.value)}
                      placeholder="#6366f1"
                      className="flex-1"
                    />
                  </div>
                </div>
                <div className="space-y-2">
                  <Label htmlFor="text_color">Text Color</Label>
                  <div className="flex gap-2">
                    <Input
                      id="text_color"
                      type="color"
                      className="w-12 h-10 p-1 cursor-pointer"
                      {...register('text_color')}
                    />
                    <Input
                      {...register('text_color')}
                      placeholder="#ffffff"
                      className="flex-1"
                    />
                  </div>
                </div>
              </div>

              <div className="space-y-2">
                <Label>Widget Position</Label>
                <Select
                  value={position}
                  onValueChange={(value: 'bottom-right' | 'bottom-left') => setValue('position', value)}
                >
                  <SelectTrigger>
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectItem value="bottom-right">Bottom Right</SelectItem>
                    <SelectItem value="bottom-left">Bottom Left</SelectItem>
                  </SelectContent>
                </Select>
              </div>

              <div className="space-y-2">
                <Label htmlFor="welcome_message">Welcome Message</Label>
                <Textarea
                  id="welcome_message"
                  placeholder="Hello! How can we help you today?"
                  {...register('welcome_message')}
                />
              </div>

              <div className="space-y-2">
                <Label htmlFor="placeholder_text">Input Placeholder</Label>
                <Input
                  id="placeholder_text"
                  placeholder="Type a message..."
                  {...register('placeholder_text')}
                />
              </div>
            </CardContent>
          </Card>

          {/* Preview */}
          <Card>
            <CardHeader>
              <CardTitle className="text-base">Preview</CardTitle>
            </CardHeader>
            <CardContent>
              <div className="relative h-64 bg-muted rounded-lg overflow-hidden">
                <div
                  className={`absolute bottom-4 ${position === 'bottom-right' ? 'right-4' : 'left-4'}`}
                >
                  <div
                    className="w-14 h-14 rounded-full flex items-center justify-center shadow-lg cursor-pointer"
                    style={{ backgroundColor: primaryColor }}
                  >
                    <MessageSquare className="h-6 w-6 text-white" />
                  </div>
                </div>
              </div>
            </CardContent>
          </Card>
        </TabsContent>

        <TabsContent value="behavior" className="space-y-4 pt-4">
          <Card>
            <CardHeader>
              <CardTitle className="text-base">Widget Behavior</CardTitle>
              <CardDescription>Configure how the widget behaves</CardDescription>
            </CardHeader>
            <CardContent className="space-y-4">
              <div className="flex items-center justify-between">
                <div>
                  <Label>Auto-open Widget</Label>
                  <p className="text-sm text-muted-foreground">
                    Automatically open the chat widget after a delay
                  </p>
                </div>
                <Switch
                  checked={autoOpen}
                  onCheckedChange={(checked) => setValue('auto_open', checked)}
                />
              </div>

              {autoOpen && (
                <div className="space-y-2 pl-4 border-l-2 border-primary/20">
                  <Label htmlFor="auto_open_delay">Delay (seconds)</Label>
                  <Input
                    id="auto_open_delay"
                    type="number"
                    min={0}
                    {...register('auto_open_delay')}
                  />
                </div>
              )}

              <div className="flex items-center justify-between">
                <div>
                  <Label>Show Typing Indicator</Label>
                  <p className="text-sm text-muted-foreground">
                    Show when the agent or bot is typing
                  </p>
                </div>
                <Switch
                  checked={watch('show_typing_indicator')}
                  onCheckedChange={(checked) => setValue('show_typing_indicator', checked)}
                />
              </div>

              <div className="space-y-2">
                <Label htmlFor="allowed_domains">Allowed Domains</Label>
                <Textarea
                  id="allowed_domains"
                  placeholder="example.com&#10;app.example.com"
                  {...register('allowed_domains')}
                />
                <p className="text-xs text-muted-foreground">
                  One domain per line. Leave empty to allow all domains.
                </p>
              </div>
            </CardContent>
          </Card>
        </TabsContent>

        <TabsContent value="embed" className="space-y-4 pt-4">
          <Card>
            <CardHeader>
              <CardTitle className="text-base flex items-center gap-2">
                <Code className="h-4 w-4" />
                Embed Code
              </CardTitle>
              <CardDescription>
                Add this code to your website to display the chat widget
              </CardDescription>
            </CardHeader>
            <CardContent className="space-y-4">
              <div className="relative">
                <pre className="bg-muted p-4 rounded-lg text-xs overflow-x-auto">
                  <code>{`<script>
  (function(w, d, s, o, f, js, fjs) {
    w['LinktorWidget'] = o;
    w[o] = w[o] || function() { (w[o].q = w[o].q || []).push(arguments) };
    js = d.createElement(s); fjs = d.getElementsByTagName(s)[0];
    js.id = o; js.src = f; js.async = 1; fjs.parentNode.insertBefore(js, fjs);
  }(window, document, 'script', 'linktor', '/widget/linktor.js'));
  linktor('init', { channelId: '${channel?.id || '{CHANNEL_ID}'}' });
</script>`}</code>
                </pre>
                <Button
                  type="button"
                  variant="outline"
                  size="sm"
                  className="absolute top-2 right-2"
                  onClick={copyEmbedCode}
                >
                  {copied ? <Check className="h-4 w-4" /> : <Copy className="h-4 w-4" />}
                </Button>
              </div>

              <p className="text-sm text-muted-foreground">
                Place this code just before the closing <code className="bg-muted px-1 rounded">&lt;/body&gt;</code> tag on your website.
              </p>
            </CardContent>
          </Card>
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
          {isSubmitting && <Loader2 className="mr-2 h-4 w-4 animate-spin" />}
          {isEditing ? 'Update Channel' : 'Create Channel'}
        </Button>
      </div>
    </form>
  )
}
