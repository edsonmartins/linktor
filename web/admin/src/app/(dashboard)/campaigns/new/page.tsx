'use client'

import { useState } from 'react'
import Link from 'next/link'
import { useRouter } from 'next/navigation'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { useTranslations } from 'next-intl'
import { ArrowLeft } from 'lucide-react'
import { Header } from '@/components/layout/header'
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
import { Card, CardContent } from '@/components/ui/card'
import { useToast } from '@/hooks/use-toast'
import { api } from '@/lib/api'
import { queryKeys } from '@/lib/query'
import type { Campaign, Channel, CreateCampaignRequest } from '@/types'

// Channels that have an outbound sender wired.
const SUPPORTED_TYPES = new Set(['whatsapp_official', 'telegram'])

// Parse the recipients textarea: one per line, "phone,param1,param2".
function parseRecipients(raw: string): CreateCampaignRequest['recipients'] {
  return raw
    .split('\n')
    .map((line) => line.trim())
    .filter(Boolean)
    .map((line) => {
      const [phone, ...params] = line.split(',').map((p) => p.trim())
      return { phone, params: params.filter(Boolean) }
    })
    .filter((r) => r.phone)
}

export default function NewCampaignPage() {
  const t = useTranslations('campaigns')
  const router = useRouter()
  const { toast } = useToast()
  const queryClient = useQueryClient()

  const [name, setName] = useState('')
  const [channelId, setChannelId] = useState('')
  const [templateName, setTemplateName] = useState('')
  const [language, setLanguage] = useState('pt_BR')
  const [recipientsRaw, setRecipientsRaw] = useState('')

  const { data: channels } = useQuery({
    queryKey: queryKeys.channels.list(),
    queryFn: () => api.get<Channel[]>('/channels'),
  })

  const usableChannels = (channels ?? []).filter((c) => SUPPORTED_TYPES.has(c.type))

  const createMutation = useMutation({
    mutationFn: (body: CreateCampaignRequest) => api.post<Campaign>('/campaigns', body),
    onSuccess: (campaign) => {
      queryClient.invalidateQueries({ queryKey: queryKeys.campaigns.lists() })
      toast({ title: t('createCta'), description: campaign.name })
      router.push(`/campaigns/${campaign.id}`)
    },
    onError: (err: Error) => {
      toast({ title: 'Error', description: err.message, variant: 'destructive' })
    },
  })

  const recipients = parseRecipients(recipientsRaw)
  const canSubmit =
    name.trim() && channelId && templateName.trim() && recipients.length > 0

  function handleSubmit() {
    if (!canSubmit) return
    createMutation.mutate({
      name: name.trim(),
      channel_id: channelId,
      template_name: templateName.trim(),
      template_language: language.trim() || 'pt_BR',
      recipients,
    })
  }

  return (
    <>
      <Header title={t('createTitle')}>
        <Link href="/campaigns">
          <Button variant="ghost" size="sm">
            <ArrowLeft className="mr-2 h-4 w-4" />
            {t('backToList')}
          </Button>
        </Link>
      </Header>

      <div className="p-6">
        <Card className="max-w-2xl">
          <CardContent className="space-y-5 pt-6">
            <div className="space-y-2">
              <Label htmlFor="name">{t('name')}</Label>
              <Input
                id="name"
                value={name}
                onChange={(e) => setName(e.target.value)}
                placeholder={t('name')}
              />
            </div>

            <div className="space-y-2">
              <Label>{t('channel')}</Label>
              <Select value={channelId} onValueChange={setChannelId}>
                <SelectTrigger>
                  <SelectValue placeholder={t('channel')} />
                </SelectTrigger>
                <SelectContent>
                  {usableChannels.map((c) => (
                    <SelectItem key={c.id} value={c.id}>
                      {c.name} ({c.type})
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </div>

            <div className="grid grid-cols-2 gap-4">
              <div className="space-y-2">
                <Label htmlFor="template">{t('template')}</Label>
                <Input
                  id="template"
                  value={templateName}
                  onChange={(e) => setTemplateName(e.target.value)}
                  placeholder="welcome_v1"
                />
              </div>
              <div className="space-y-2">
                <Label htmlFor="language">{t('language')}</Label>
                <Input
                  id="language"
                  value={language}
                  onChange={(e) => setLanguage(e.target.value)}
                  placeholder="pt_BR"
                />
              </div>
            </div>

            <div className="space-y-2">
              <Label htmlFor="recipients">
                {t('recipients')}
                {recipients.length > 0 && (
                  <span className="ml-2 text-xs text-muted-foreground">
                    ({recipients.length})
                  </span>
                )}
              </Label>
              <Textarea
                id="recipients"
                rows={8}
                value={recipientsRaw}
                onChange={(e) => setRecipientsRaw(e.target.value)}
                placeholder={'+5511999999999,John,Order #42\n+5511888888888,Mary,Order #43'}
                className="font-mono text-sm"
              />
              <p className="text-xs text-muted-foreground">{t('recipientsHelp')}</p>
            </div>

            <div className="flex justify-end gap-2 pt-2">
              <Link href="/campaigns">
                <Button variant="outline">{t('cancel')}</Button>
              </Link>
              <Button onClick={handleSubmit} disabled={!canSubmit || createMutation.isPending}>
                {t('createCta')}
              </Button>
            </div>
          </CardContent>
        </Card>
      </div>
    </>
  )
}
