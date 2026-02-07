'use client'

import { useState } from 'react'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import {
  Plus,
  Bot as BotIcon,
  MoreVertical,
  Play,
  Pause,
  Settings,
  Trash2,
  Zap,
  Brain,
  MessageSquare,
  Search,
} from 'lucide-react'
import { Header } from '@/components/layout/header'
import { Button } from '@/components/ui/button'
import { Badge } from '@/components/ui/badge'
import { Input } from '@/components/ui/input'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Skeleton } from '@/components/ui/skeleton'
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { Label } from '@/components/ui/label'
import { Textarea } from '@/components/ui/textarea'
import { cn } from '@/lib/utils'
import { api } from '@/lib/api'
import { queryKeys } from '@/lib/query'
import type { Bot, BotType, AIProvider, CreateBotInput } from '@/types'
import { useRouter } from 'next/navigation'

/**
 * Bot type configurations
 */
const botTypeConfigs: Record<BotType, { label: string; description: string; icon: React.ReactNode; color: string; bgColor: string }> = {
  customer_service: {
    label: 'Customer Service',
    description: 'Handle support inquiries and FAQs',
    icon: <MessageSquare className="h-5 w-5" />,
    color: 'text-blue-500',
    bgColor: 'bg-blue-500/10',
  },
  sales: {
    label: 'Sales',
    description: 'Qualify leads and assist with purchases',
    icon: <Zap className="h-5 w-5" />,
    color: 'text-green-500',
    bgColor: 'bg-green-500/10',
  },
  faq: {
    label: 'FAQ',
    description: 'Answer frequently asked questions',
    icon: <Brain className="h-5 w-5" />,
    color: 'text-purple-500',
    bgColor: 'bg-purple-500/10',
  },
  custom: {
    label: 'Custom',
    description: 'Fully customizable bot',
    icon: <BotIcon className="h-5 w-5" />,
    color: 'text-amber-500',
    bgColor: 'bg-amber-500/10',
  },
}

/**
 * AI Provider configurations
 */
const providerConfigs: Record<AIProvider, { label: string; models: string[] }> = {
  openai: {
    label: 'OpenAI',
    models: ['gpt-4', 'gpt-4-turbo', 'gpt-3.5-turbo'],
  },
  anthropic: {
    label: 'Anthropic',
    models: ['claude-3-opus', 'claude-3-sonnet', 'claude-3-haiku'],
  },
  ollama: {
    label: 'Ollama (Local)',
    models: ['llama2', 'mistral', 'codellama'],
  },
}

/**
 * Status Badge
 */
function StatusBadge({ isActive }: { isActive: boolean }) {
  return (
    <Badge variant={isActive ? 'success' : 'secondary'} className="gap-1">
      {isActive ? (
        <>
          <Play className="h-3 w-3" />
          Active
        </>
      ) : (
        <>
          <Pause className="h-3 w-3" />
          Inactive
        </>
      )}
    </Badge>
  )
}

/**
 * Bot Card Component
 */
function BotCard({
  bot,
  onActivate,
  onDeactivate,
  onDelete,
}: {
  bot: Bot
  onActivate: () => void
  onDeactivate: () => void
  onDelete: () => void
}) {
  const router = useRouter()
  const config = botTypeConfigs[bot.type] || botTypeConfigs.custom

  return (
    <Card className="hover:border-primary/30 transition-colors">
      <CardHeader className="pb-3">
        <div className="flex items-start justify-between">
          <div className="flex items-center gap-3">
            <div
              className={cn(
                'flex h-12 w-12 items-center justify-center rounded-lg',
                config.bgColor,
                config.color
              )}
            >
              {config.icon}
            </div>
            <div>
              <CardTitle className="text-base">{bot.name}</CardTitle>
              <CardDescription className="text-xs">
                {providerConfigs[bot.provider]?.label} / {bot.model}
              </CardDescription>
            </div>
          </div>
          <DropdownMenu>
            <DropdownMenuTrigger asChild>
              <Button variant="ghost" size="icon" className="h-8 w-8">
                <MoreVertical className="h-4 w-4" />
              </Button>
            </DropdownMenuTrigger>
            <DropdownMenuContent align="end">
              <DropdownMenuItem onClick={() => router.push(`/bots/${bot.id}`)}>
                <Settings className="h-4 w-4 mr-2" />
                Configure
              </DropdownMenuItem>
              {bot.is_active ? (
                <DropdownMenuItem onClick={onDeactivate}>
                  <Pause className="h-4 w-4 mr-2" />
                  Deactivate
                </DropdownMenuItem>
              ) : (
                <DropdownMenuItem onClick={onActivate}>
                  <Play className="h-4 w-4 mr-2" />
                  Activate
                </DropdownMenuItem>
              )}
              <DropdownMenuSeparator />
              <DropdownMenuItem className="text-destructive" onClick={onDelete}>
                <Trash2 className="h-4 w-4 mr-2" />
                Delete
              </DropdownMenuItem>
            </DropdownMenuContent>
          </DropdownMenu>
        </div>
      </CardHeader>
      <CardContent className="pt-0">
        <div className="flex items-center justify-between">
          <StatusBadge isActive={bot.is_active} />
          <div className="flex items-center gap-2 text-xs text-muted-foreground">
            <Badge variant="outline" className="font-mono text-xs">
              {bot.channel_ids?.length || 0} channels
            </Badge>
          </div>
        </div>
        {bot.config?.system_prompt && (
          <p className="mt-3 text-xs text-muted-foreground line-clamp-2">
            {bot.config.system_prompt}
          </p>
        )}
      </CardContent>
    </Card>
  )
}

/**
 * Create Bot Dialog
 */
function CreateBotDialog({
  open,
  onOpenChange,
  onSuccess,
}: {
  open: boolean
  onOpenChange: (open: boolean) => void
  onSuccess: () => void
}) {
  const [formData, setFormData] = useState<CreateBotInput>({
    name: '',
    type: 'customer_service',
    provider: 'openai',
    model: 'gpt-4',
  })

  const queryClient = useQueryClient()

  const createMutation = useMutation({
    mutationFn: (data: CreateBotInput) => api.post<Bot>('/bots', data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: queryKeys.bots.all })
      onOpenChange(false)
      setFormData({
        name: '',
        type: 'customer_service',
        provider: 'openai',
        model: 'gpt-4',
      })
      onSuccess()
    },
  })

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault()
    createMutation.mutate(formData)
  }

  const availableModels = providerConfigs[formData.provider]?.models || []

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-[500px]">
        <form onSubmit={handleSubmit}>
          <DialogHeader>
            <DialogTitle>Create New Bot</DialogTitle>
            <DialogDescription>
              Configure a new AI bot for your conversations
            </DialogDescription>
          </DialogHeader>

          <div className="grid gap-4 py-4">
            <div className="grid gap-2">
              <Label htmlFor="name">Bot Name</Label>
              <Input
                id="name"
                placeholder="e.g., Support Bot"
                value={formData.name}
                onChange={(e) => setFormData({ ...formData, name: e.target.value })}
                required
              />
            </div>

            <div className="grid gap-2">
              <Label htmlFor="type">Bot Type</Label>
              <Select
                value={formData.type}
                onValueChange={(value: BotType) => setFormData({ ...formData, type: value })}
              >
                <SelectTrigger>
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  {Object.entries(botTypeConfigs).map(([key, config]) => (
                    <SelectItem key={key} value={key}>
                      <div className="flex items-center gap-2">
                        <span className={config.color}>{config.icon}</span>
                        <span>{config.label}</span>
                      </div>
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
              <p className="text-xs text-muted-foreground">
                {botTypeConfigs[formData.type].description}
              </p>
            </div>

            <div className="grid grid-cols-2 gap-4">
              <div className="grid gap-2">
                <Label htmlFor="provider">AI Provider</Label>
                <Select
                  value={formData.provider}
                  onValueChange={(value: AIProvider) => {
                    const newModels = providerConfigs[value]?.models || []
                    setFormData({
                      ...formData,
                      provider: value,
                      model: newModels[0] || '',
                    })
                  }}
                >
                  <SelectTrigger>
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent>
                    {Object.entries(providerConfigs).map(([key, config]) => (
                      <SelectItem key={key} value={key}>
                        {config.label}
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
              </div>

              <div className="grid gap-2">
                <Label htmlFor="model">Model</Label>
                <Select
                  value={formData.model}
                  onValueChange={(value) => setFormData({ ...formData, model: value })}
                >
                  <SelectTrigger>
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent>
                    {availableModels.map((model) => (
                      <SelectItem key={model} value={model}>
                        {model}
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
              </div>
            </div>
          </div>

          <DialogFooter>
            <Button type="button" variant="outline" onClick={() => onOpenChange(false)}>
              Cancel
            </Button>
            <Button type="submit" disabled={createMutation.isPending || !formData.name}>
              {createMutation.isPending ? 'Creating...' : 'Create Bot'}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  )
}

/**
 * Bots Page
 */
export default function BotsPage() {
  const router = useRouter()
  const queryClient = useQueryClient()
  const [search, setSearch] = useState('')
  const [createDialogOpen, setCreateDialogOpen] = useState(false)

  // Fetch bots
  const { data, isLoading } = useQuery({
    queryKey: queryKeys.bots.list({ search }),
    queryFn: () => api.get<{ data: Bot[] }>('/bots', search ? { search } : undefined),
  })

  const bots = data?.data || []

  // Activate mutation
  const activateMutation = useMutation({
    mutationFn: (id: string) => api.post(`/bots/${id}/activate`),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: queryKeys.bots.all })
    },
  })

  // Deactivate mutation
  const deactivateMutation = useMutation({
    mutationFn: (id: string) => api.post(`/bots/${id}/deactivate`),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: queryKeys.bots.all })
    },
  })

  // Delete mutation
  const deleteMutation = useMutation({
    mutationFn: (id: string) => api.delete(`/bots/${id}`),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: queryKeys.bots.all })
    },
  })

  const filteredBots = bots.filter(
    (bot) =>
      bot.name.toLowerCase().includes(search.toLowerCase()) ||
      bot.type.toLowerCase().includes(search.toLowerCase())
  )

  return (
    <div className="flex flex-col h-full">
      <Header title="Bots" />

      <div className="p-6 space-y-6 overflow-auto">
        {/* Header with search and create */}
        <div className="flex items-center justify-between gap-4">
          <div className="relative flex-1 max-w-sm">
            <Search className="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground" />
            <Input
              placeholder="Search bots..."
              value={search}
              onChange={(e) => setSearch(e.target.value)}
              className="pl-9"
            />
          </div>
          <Button onClick={() => setCreateDialogOpen(true)}>
            <Plus className="h-4 w-4 mr-2" />
            Create Bot
          </Button>
        </div>

        {/* Bots Grid */}
        {isLoading ? (
          <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-3">
            {Array.from({ length: 3 }).map((_, i) => (
              <Card key={i}>
                <CardHeader className="pb-3">
                  <div className="flex items-start gap-3">
                    <Skeleton className="h-12 w-12 rounded-lg" />
                    <div className="space-y-2">
                      <Skeleton className="h-4 w-24" />
                      <Skeleton className="h-3 w-32" />
                    </div>
                  </div>
                </CardHeader>
                <CardContent className="pt-0">
                  <Skeleton className="h-6 w-16" />
                </CardContent>
              </Card>
            ))}
          </div>
        ) : filteredBots.length > 0 ? (
          <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-3">
            {filteredBots.map((bot) => (
              <BotCard
                key={bot.id}
                bot={bot}
                onActivate={() => activateMutation.mutate(bot.id)}
                onDeactivate={() => deactivateMutation.mutate(bot.id)}
                onDelete={() => {
                  if (confirm('Are you sure you want to delete this bot?')) {
                    deleteMutation.mutate(bot.id)
                  }
                }}
              />
            ))}
          </div>
        ) : (
          <Card className="border-dashed">
            <CardContent className="py-12 text-center">
              <BotIcon className="mx-auto h-12 w-12 text-muted-foreground opacity-50" />
              <p className="mt-4 text-lg font-medium">No bots configured</p>
              <p className="text-sm text-muted-foreground">
                Create your first AI bot to automate conversations
              </p>
              <Button className="mt-4" onClick={() => setCreateDialogOpen(true)}>
                <Plus className="h-4 w-4 mr-2" />
                Create Bot
              </Button>
            </CardContent>
          </Card>
        )}

        {/* Bot Types Reference */}
        <section>
          <h3 className="text-lg font-semibold mb-4">Bot Types</h3>
          <div className="grid gap-3 md:grid-cols-2 lg:grid-cols-4">
            {Object.entries(botTypeConfigs).map(([key, config]) => (
              <Card key={key} className="hover:border-primary/30 transition-colors">
                <CardContent className="p-4">
                  <div className="flex items-center gap-3">
                    <div
                      className={cn(
                        'flex h-10 w-10 items-center justify-center rounded-lg',
                        config.bgColor,
                        config.color
                      )}
                    >
                      {config.icon}
                    </div>
                    <div>
                      <h4 className="text-sm font-medium">{config.label}</h4>
                      <p className="text-xs text-muted-foreground">{config.description}</p>
                    </div>
                  </div>
                </CardContent>
              </Card>
            ))}
          </div>
        </section>
      </div>

      {/* Create Dialog */}
      <CreateBotDialog
        open={createDialogOpen}
        onOpenChange={setCreateDialogOpen}
        onSuccess={() => {}}
      />
    </div>
  )
}
