'use client'

import { useEffect, useMemo, useState } from 'react'
import { useMutation, useQuery } from '@tanstack/react-query'
import {
  ImageIcon,
  RefreshCw,
  Save,
  Send,
  Sparkles,
  Trash2,
} from 'lucide-react'
import { Header } from '@/components/layout/header'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { Separator } from '@/components/ui/separator'
import { Textarea } from '@/components/ui/textarea'
import { Badge } from '@/components/ui/badge'
import { Alert, AlertDescription, AlertTitle } from '@/components/ui/alert'
import { useToast } from '@/hooks/use-toast'
import { api } from '@/lib/api'
import type { Channel } from '@/types'

type VRETemplateListResponse = {
  templates: string[]
}

type VREOutputFormat = 'png' | 'webp' | 'jpeg'

type VRERenderResponse = {
  image_url?: string
  image_base64?: string
  caption: string
  follow_up_text?: string
  width: number
  height: number
  format: VREOutputFormat
  size_bytes: number
  render_time_ms: number
  cache_hit: boolean
  delivered?: boolean
}

type VREBrandConfig = {
  tenant_id?: string
  name: string
  logo_url?: string
  primary_color: string
  secondary_color: string
  accent_color: string
  background: string
  text_color: string
  muted_color: string
  font_family: string
  border_radius: string
  icons?: Record<string, string>
}

const DEFAULT_TEMPLATE = 'menu_opcoes'

export default function VREPage() {
  const { toast } = useToast()
  const [selectedTemplate, setSelectedTemplate] = useState(DEFAULT_TEMPLATE)
  const [channelId, setChannelId] = useState<string>('')
  const [sendTo, setSendTo] = useState('')
  const [caption, setCaption] = useState('')
  const [followUpText, setFollowUpText] = useState('')
  const [format, setFormat] = useState<VREOutputFormat>('jpeg')
  const [dataJson, setDataJson] = useState(
    JSON.stringify(getSampleTemplateData(DEFAULT_TEMPLATE), null, 2),
  )
  const [renderResult, setRenderResult] = useState<VRERenderResponse | null>(null)
  const [previewResult, setPreviewResult] = useState<VRERenderResponse | null>(null)
  const [brandConfig, setBrandConfig] = useState<VREBrandConfig>({
    name: '',
    logo_url: '',
    primary_color: '#0F3460',
    secondary_color: '#E94560',
    accent_color: '#16C79A',
    background: '#FFFFFF',
    text_color: '#1A1A2E',
    muted_color: '#8B95A2',
    font_family: "'DM Sans', sans-serif",
    border_radius: '14px',
    icons: {},
  })

  const { data: templatesData, refetch: refetchTemplates, isFetching: isRefreshingTemplates } = useQuery({
    queryKey: ['vre', 'templates'],
    queryFn: () => api.get<VRETemplateListResponse>('/vre/templates'),
  })

  const { data: channelsData } = useQuery({
    queryKey: ['vre', 'channels'],
    queryFn: () => api.get<Channel[]>('/channels'),
  })

  const { data: loadedBrandConfig } = useQuery({
    queryKey: ['vre', 'config'],
    queryFn: () => api.get<VREBrandConfig>('/vre/config'),
  })

  useEffect(() => {
    if (loadedBrandConfig) {
      setBrandConfig(loadedBrandConfig)
    }
  }, [loadedBrandConfig])

  const channels = useMemo(
    () => (channelsData ?? []).filter((channel) => channel.enabled),
    [channelsData],
  )

  useEffect(() => {
    if (!channelId && channels.length > 0) {
      setChannelId(channels[0].id)
    }
  }, [channelId, channels])

  const previewMutation = useMutation({
    mutationFn: (templateId: string) =>
      api.get<VRERenderResponse>(`/vre/templates/${templateId}/preview`),
    onSuccess: (response) => {
      setPreviewResult(response)
      toast({
        title: 'Preview ready',
        description: `${response.width}x${response.height} ${response.format}`,
      })
    },
    onError: (error) => {
      toast({
        title: 'Preview failed',
        description: error instanceof Error ? error.message : 'Unknown error',
        variant: 'error',
      })
    },
  })

  const renderMutation = useMutation({
    mutationFn: (payload: Record<string, unknown>) =>
      api.post<VRERenderResponse>('/vre/render', payload),
    onSuccess: (response) => {
      setRenderResult(response)
      toast({
        title: 'Render complete',
        description: `${response.width}x${response.height} ${response.format}`,
      })
    },
    onError: (error) => {
      toast({
        title: 'Render failed',
        description: error instanceof Error ? error.message : 'Unknown error',
        variant: 'error',
      })
    },
  })

  const renderAndSendMutation = useMutation({
    mutationFn: (payload: Record<string, unknown>) =>
      api.post<VRERenderResponse>('/vre/render-and-send', payload),
    onSuccess: (response) => {
      setRenderResult(response)
      toast({
        title: response.delivered ? 'Sent' : 'Rendered without delivery',
        description: response.caption || 'VRE response processed',
      })
    },
    onError: (error) => {
      toast({
        title: 'Send failed',
        description: error instanceof Error ? error.message : 'Unknown error',
        variant: 'error',
      })
    },
  })

  const saveBrandConfigMutation = useMutation({
    mutationFn: (payload: VREBrandConfig) => api.put<VREBrandConfig>('/vre/config', payload),
    onSuccess: () => {
      toast({
        title: 'Brand config saved',
        description: brandConfig.name || 'VRE branding updated',
      })
    },
    onError: (error) => {
      toast({
        title: 'Failed to save brand config',
        description: error instanceof Error ? error.message : 'Unknown error',
        variant: 'error',
      })
    },
  })

  const invalidateCacheMutation = useMutation({
    mutationFn: () => api.delete<{ message: string }>('/vre/cache'),
    onSuccess: (response) => {
      toast({
        title: 'Cache invalidated',
        description: response.message,
      })
    },
    onError: (error) => {
      toast({
        title: 'Failed to invalidate cache',
        description: error instanceof Error ? error.message : 'Unknown error',
        variant: 'error',
      })
    },
  })

  const handleTemplateChange = (templateId: string) => {
    setSelectedTemplate(templateId)
    setDataJson(JSON.stringify(getSampleTemplateData(templateId), null, 2))
  }

  const parseData = () => {
    return JSON.parse(dataJson) as Record<string, unknown>
  }

  const buildRenderPayload = () => {
    const selectedChannel = channels.find((item) => item.id === channelId)
    return {
      template_id: selectedTemplate,
      data: parseData(),
      channel: mapChannelTypeForVRE(selectedChannel?.type),
      channel_id: selectedChannel?.id,
      caption,
      follow_up_text: followUpText,
      format,
    }
  }

  const runRender = () => {
    try {
      renderMutation.mutate(buildRenderPayload())
    } catch (error) {
      toast({
        title: 'Invalid render data',
        description: error instanceof Error ? error.message : 'JSON parse error',
        variant: 'error',
      })
    }
  }

  const runRenderAndSend = () => {
    if (!sendTo) {
      toast({
        title: 'Recipient required',
        description: 'Fill send_to before sending',
        variant: 'error',
      })
      return
    }

    try {
      renderAndSendMutation.mutate({
        ...buildRenderPayload(),
        send_to: sendTo,
      })
    } catch (error) {
      toast({
        title: 'Invalid render data',
        description: error instanceof Error ? error.message : 'JSON parse error',
        variant: 'error',
      })
    }
  }

  return (
    <div className="flex h-full flex-col">
      <Header title="VRE" />

      <div className="flex-1 overflow-auto p-4">
        <div className="grid gap-4 xl:grid-cols-[1.2fr_0.8fr]">
          <div className="space-y-4">
            <Card>
              <CardHeader>
                <CardTitle className="flex items-center gap-2">
                  <Sparkles className="h-4 w-4" />
                  Runtime
                </CardTitle>
                <CardDescription>
                  Preview templates, render payloads, and send visual responses through a real channel.
                </CardDescription>
              </CardHeader>
              <CardContent className="space-y-4">
                <div className="grid gap-4 md:grid-cols-2">
                  <div className="space-y-2">
                    <Label htmlFor="vre-template">Template</Label>
                    <Select value={selectedTemplate} onValueChange={handleTemplateChange}>
                      <SelectTrigger id="vre-template">
                        <SelectValue placeholder="Select a template" />
                      </SelectTrigger>
                      <SelectContent>
                        {(templatesData?.templates ?? []).map((template) => (
                          <SelectItem key={template} value={template}>
                            {template}
                          </SelectItem>
                        ))}
                      </SelectContent>
                    </Select>
                  </div>

                  <div className="space-y-2">
                    <Label htmlFor="vre-channel">Channel</Label>
                    <Select value={channelId} onValueChange={setChannelId}>
                      <SelectTrigger id="vre-channel">
                        <SelectValue placeholder="Select a channel" />
                      </SelectTrigger>
                      <SelectContent>
                        {channels.map((channel) => (
                          <SelectItem key={channel.id} value={channel.id}>
                            {channel.name}
                          </SelectItem>
                        ))}
                      </SelectContent>
                    </Select>
                  </div>
                </div>

                <div className="grid gap-4 md:grid-cols-3">
                  <div className="space-y-2">
                    <Label htmlFor="vre-format">Format</Label>
                    <Select value={format} onValueChange={(value) => setFormat(value as VREOutputFormat)}>
                      <SelectTrigger id="vre-format">
                        <SelectValue />
                      </SelectTrigger>
                      <SelectContent>
                        <SelectItem value="jpeg">jpeg</SelectItem>
                        <SelectItem value="png">png</SelectItem>
                        <SelectItem value="webp">webp</SelectItem>
                      </SelectContent>
                    </Select>
                  </div>

                  <div className="space-y-2">
                    <Label htmlFor="vre-caption">Caption</Label>
                    <Input
                      id="vre-caption"
                      value={caption}
                      onChange={(event) => setCaption(event.target.value)}
                      placeholder="Optional caption"
                    />
                  </div>

                  <div className="space-y-2">
                    <Label htmlFor="vre-follow-up">Follow-up text</Label>
                    <Input
                      id="vre-follow-up"
                      value={followUpText}
                      onChange={(event) => setFollowUpText(event.target.value)}
                      placeholder="Optional follow-up text"
                    />
                  </div>
                </div>

                <div className="space-y-2">
                  <Label htmlFor="vre-data">Render data (JSON)</Label>
                  <Textarea
                    id="vre-data"
                    value={dataJson}
                    onChange={(event) => setDataJson(event.target.value)}
                    className="min-h-[220px] font-mono text-xs"
                  />
                </div>

                <div className="grid gap-4 md:grid-cols-[minmax(0,1fr)_auto_auto_auto]">
                  <div className="space-y-2">
                    <Label htmlFor="vre-send-to">Send to</Label>
                    <Input
                      id="vre-send-to"
                      value={sendTo}
                      onChange={(event) => setSendTo(event.target.value)}
                      placeholder="+5511999999999"
                    />
                  </div>

                  <div className="flex items-end">
                    <Button
                      type="button"
                      variant="outline"
                      onClick={() => previewMutation.mutate(selectedTemplate)}
                      disabled={previewMutation.isPending}
                    >
                      {previewMutation.isPending ? (
                        <RefreshCw className="mr-2 h-4 w-4 animate-spin" />
                      ) : (
                        <ImageIcon className="mr-2 h-4 w-4" />
                      )}
                      Preview
                    </Button>
                  </div>

                  <div className="flex items-end">
                    <Button
                      type="button"
                      variant="outline"
                      onClick={runRender}
                      disabled={renderMutation.isPending}
                    >
                      {renderMutation.isPending ? (
                        <RefreshCw className="mr-2 h-4 w-4 animate-spin" />
                      ) : (
                        <Sparkles className="mr-2 h-4 w-4" />
                      )}
                      Render
                    </Button>
                  </div>

                  <div className="flex items-end">
                    <Button
                      type="button"
                      onClick={runRenderAndSend}
                      disabled={renderAndSendMutation.isPending || !channelId}
                    >
                      {renderAndSendMutation.isPending ? (
                        <RefreshCw className="mr-2 h-4 w-4 animate-spin" />
                      ) : (
                        <Send className="mr-2 h-4 w-4" />
                      )}
                      Render and send
                    </Button>
                  </div>
                </div>
              </CardContent>
            </Card>

            <Card>
              <CardHeader>
                <CardTitle>Preview and output</CardTitle>
                <CardDescription>
                  Latest preview and render result from the VRE runtime.
                </CardDescription>
              </CardHeader>
              <CardContent className="space-y-4">
                <div className="grid gap-4 lg:grid-cols-2">
                  <ResultCard title="Template preview" result={previewResult} testId="vre-preview-result" />
                  <ResultCard title="Latest render" result={renderResult} testId="vre-render-result" />
                </div>
              </CardContent>
            </Card>
          </div>

          <div className="space-y-4">
            <Card>
              <CardHeader>
                <CardTitle>Templates and cache</CardTitle>
                <CardDescription>
                  Inspect the available templates and refresh runtime state.
                </CardDescription>
              </CardHeader>
              <CardContent className="space-y-4">
                <div className="flex flex-wrap gap-2">
                  {(templatesData?.templates ?? []).map((template) => (
                    <Badge key={template} variant={template === selectedTemplate ? 'default' : 'outline'}>
                      {template}
                    </Badge>
                  ))}
                </div>

                <div className="flex gap-2">
                  <Button
                    type="button"
                    variant="outline"
                    onClick={() => refetchTemplates()}
                    disabled={isRefreshingTemplates}
                  >
                    {isRefreshingTemplates ? (
                      <RefreshCw className="mr-2 h-4 w-4 animate-spin" />
                    ) : (
                      <RefreshCw className="mr-2 h-4 w-4" />
                    )}
                    Refresh templates
                  </Button>

                  <Button
                    type="button"
                    variant="outline"
                    onClick={() => invalidateCacheMutation.mutate()}
                    disabled={invalidateCacheMutation.isPending}
                  >
                    {invalidateCacheMutation.isPending ? (
                      <RefreshCw className="mr-2 h-4 w-4 animate-spin" />
                    ) : (
                      <Trash2 className="mr-2 h-4 w-4" />
                    )}
                    Invalidate cache
                  </Button>
                </div>
              </CardContent>
            </Card>

            <Card>
              <CardHeader>
                <CardTitle>Brand config</CardTitle>
                <CardDescription>
                  Edit the tenant branding used by VRE templates.
                </CardDescription>
              </CardHeader>
              <CardContent className="space-y-4">
                <div className="grid gap-4 md:grid-cols-2">
                  <BrandConfigInput
                    id="brand-name"
                    label="Brand name"
                    value={brandConfig.name}
                    onChange={(value) => setBrandConfig((current) => ({ ...current, name: value }))}
                  />
                  <BrandConfigInput
                    id="brand-logo-url"
                    label="Logo URL"
                    value={brandConfig.logo_url ?? ''}
                    onChange={(value) => setBrandConfig((current) => ({ ...current, logo_url: value }))}
                  />
                  <BrandConfigInput
                    id="brand-primary-color"
                    label="Primary color"
                    value={brandConfig.primary_color}
                    onChange={(value) => setBrandConfig((current) => ({ ...current, primary_color: value }))}
                  />
                  <BrandConfigInput
                    id="brand-secondary-color"
                    label="Secondary color"
                    value={brandConfig.secondary_color}
                    onChange={(value) => setBrandConfig((current) => ({ ...current, secondary_color: value }))}
                  />
                  <BrandConfigInput
                    id="brand-accent-color"
                    label="Accent color"
                    value={brandConfig.accent_color}
                    onChange={(value) => setBrandConfig((current) => ({ ...current, accent_color: value }))}
                  />
                  <BrandConfigInput
                    id="brand-background"
                    label="Background"
                    value={brandConfig.background}
                    onChange={(value) => setBrandConfig((current) => ({ ...current, background: value }))}
                  />
                  <BrandConfigInput
                    id="brand-text-color"
                    label="Text color"
                    value={brandConfig.text_color}
                    onChange={(value) => setBrandConfig((current) => ({ ...current, text_color: value }))}
                  />
                  <BrandConfigInput
                    id="brand-muted-color"
                    label="Muted color"
                    value={brandConfig.muted_color}
                    onChange={(value) => setBrandConfig((current) => ({ ...current, muted_color: value }))}
                  />
                  <BrandConfigInput
                    id="brand-font-family"
                    label="Font family"
                    value={brandConfig.font_family}
                    onChange={(value) => setBrandConfig((current) => ({ ...current, font_family: value }))}
                  />
                  <BrandConfigInput
                    id="brand-border-radius"
                    label="Border radius"
                    value={brandConfig.border_radius}
                    onChange={(value) => setBrandConfig((current) => ({ ...current, border_radius: value }))}
                  />
                </div>

                <Separator />

                <Button
                  type="button"
                  onClick={() => saveBrandConfigMutation.mutate(brandConfig)}
                  disabled={saveBrandConfigMutation.isPending}
                >
                  {saveBrandConfigMutation.isPending ? (
                    <RefreshCw className="mr-2 h-4 w-4 animate-spin" />
                  ) : (
                    <Save className="mr-2 h-4 w-4" />
                  )}
                  Save brand config
                </Button>
              </CardContent>
            </Card>
          </div>
        </div>
      </div>
    </div>
  )
}

function BrandConfigInput({
  id,
  label,
  value,
  onChange,
}: {
  id: string
  label: string
  value: string
  onChange: (value: string) => void
}) {
  return (
    <div className="space-y-2">
      <Label htmlFor={id}>{label}</Label>
      <Input id={id} value={value} onChange={(event) => onChange(event.target.value)} />
    </div>
  )
}

function ResultCard({
  title,
  result,
  testId,
}: {
  title: string
  result: VRERenderResponse | null
  testId?: string
}) {
  const imageSrc = result?.image_base64
    ? `data:image/${result.format};base64,${result.image_base64}`
    : result?.image_url

  return (
    <div className="rounded-md border p-4" data-testid={testId}>
      <div className="mb-3 flex items-center justify-between">
        <h3 className="font-medium">{title}</h3>
        {result && (
          <div className="flex gap-2">
            {result.cache_hit && <Badge variant="outline">cache hit</Badge>}
            {result.delivered && <Badge variant="default">delivered</Badge>}
          </div>
        )}
      </div>

      {!result ? (
        <Alert>
          <ImageIcon className="h-4 w-4" />
          <AlertTitle>No output yet</AlertTitle>
          <AlertDescription>
            Run preview or render to inspect the generated asset.
          </AlertDescription>
        </Alert>
      ) : (
        <div className="space-y-3">
          {imageSrc && (
            <img
              src={imageSrc}
              alt={title}
              className="max-h-72 w-full rounded-md border object-contain"
            />
          )}
          <div className="grid gap-2 text-sm md:grid-cols-2">
            <div>
              <span className="text-muted-foreground">Format:</span> {result.format}
            </div>
            <div>
              <span className="text-muted-foreground">Size:</span> {result.width}x{result.height}
            </div>
            <div>
              <span className="text-muted-foreground">Bytes:</span> {result.size_bytes}
            </div>
            <div>
              <span className="text-muted-foreground">Render time:</span> {String(result.render_time_ms)}
            </div>
          </div>
          {result.caption && (
            <p className="text-sm">
              <span className="text-muted-foreground">Caption:</span> {result.caption}
            </p>
          )}
          {result.follow_up_text && (
            <p className="text-sm">
              <span className="text-muted-foreground">Follow-up:</span> {result.follow_up_text}
            </p>
          )}
        </div>
      )}
    </div>
  )
}

function mapChannelTypeForVRE(channelType?: Channel['type']) {
  switch (channelType) {
    case 'telegram':
      return 'telegram'
    case 'email':
      return 'email'
    case 'webchat':
      return 'web'
    default:
      return 'whatsapp'
  }
}

function getSampleTemplateData(templateId: string) {
  switch (templateId) {
    case 'card_produto':
      return {
        nome: 'Cesta premium',
        preco: 199.9,
        unidade: 'un',
        estoque: 18,
        destaque: 'mais vendido',
        mensagem: 'Entrega em 24h',
      }
    case 'status_pedido':
      return {
        numero_pedido: 'PED-1024',
        status_atual: 'transporte',
        itens_resumo: '2 itens',
        valor_total: 249.9,
        previsao_entrega: 'Hoje, 18:30',
      }
    case 'lista_produtos':
      return {
        titulo: 'Ofertas da semana',
        produtos: [
          { nome: 'Kit cafe da manha', preco: 39.9 },
          { nome: 'Cesta gourmet', preco: 149.9 },
        ],
      }
    case 'confirmacao':
      return {
        titulo: 'Pedido confirmado',
        descricao: 'Seu pagamento foi aprovado.',
      }
    case 'cobranca_pix':
      return {
        titulo: 'Pix pendente',
        valor: 89.9,
        vencimento: '2026-04-22 18:00',
      }
    default:
      return {
        titulo: 'Como podemos ajudar?',
        subtitulo: 'Escolha uma opcao abaixo',
        opcoes: [
          { label: 'Novo pedido', descricao: 'Comprar agora', icone: 'pedido' },
          { label: 'Falar com suporte', descricao: 'Atendimento humano', icone: 'atendente' },
        ],
      }
  }
}
