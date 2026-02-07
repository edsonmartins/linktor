'use client'

import { useState } from 'react'
import { useParams, useRouter } from 'next/navigation'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import {
  ArrowLeft,
  Save,
  Play,
  Pause,
  Trash2,
  Plus,
  X,
  TestTube,
  Send,
  Bot as BotIcon,
  Settings,
  Clock,
  Zap,
  BookOpen,
  Radio,
} from 'lucide-react'
import { Header } from '@/components/layout/header'
import { Button } from '@/components/ui/button'
import { Badge } from '@/components/ui/badge'
import { Input } from '@/components/ui/input'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Skeleton } from '@/components/ui/skeleton'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { Label } from '@/components/ui/label'
import { Textarea } from '@/components/ui/textarea'
import { Slider } from '@/components/ui/slider'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import { ScrollArea } from '@/components/ui/scroll-area'
import { cn } from '@/lib/utils'
import { api } from '@/lib/api'
import { queryKeys } from '@/lib/query'
import type {
  Bot,
  BotConfig,
  BotTestInput,
  BotTestResult,
  Channel,
  KnowledgeBase,
  EscalationRule,
  EscalationRuleType,
  AIProvider,
} from '@/types'

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

const escalationRuleTypes: { value: EscalationRuleType; label: string; description: string }[] = [
  { value: 'low_confidence', label: 'Low Confidence', description: 'Escalate when confidence is below threshold' },
  { value: 'sentiment', label: 'Negative Sentiment', description: 'Escalate on negative customer sentiment' },
  { value: 'keyword', label: 'Keyword Match', description: 'Escalate when specific keywords are detected' },
  { value: 'intent', label: 'Intent Match', description: 'Escalate for specific detected intents' },
  { value: 'user_request', label: 'User Request', description: 'Escalate when user asks for human' },
]

/**
 * Bot Detail Page
 */
export default function BotDetailPage() {
  const params = useParams()
  const router = useRouter()
  const queryClient = useQueryClient()
  const botId = params.id as string

  const [activeTab, setActiveTab] = useState('general')
  const [testDialogOpen, setTestDialogOpen] = useState(false)
  const [testMessage, setTestMessage] = useState('')
  const [testResult, setTestResult] = useState<BotTestResult | null>(null)
  const [hasChanges, setHasChanges] = useState(false)

  // Fetch bot
  const { data: bot, isLoading } = useQuery({
    queryKey: queryKeys.bots.detail(botId),
    queryFn: () => api.get<Bot>(`/bots/${botId}`),
  })

  // Fetch channels for assignment
  const { data: channelsData } = useQuery({
    queryKey: queryKeys.channels.list(),
    queryFn: () => api.get<{ data: Channel[] }>('/channels'),
  })

  // Fetch knowledge bases
  const { data: kbData } = useQuery({
    queryKey: queryKeys.knowledgeBases.list({}),
    queryFn: () => api.get<{ data: KnowledgeBase[] }>('/knowledge-bases'),
  })

  const channels = channelsData?.data || []
  const knowledgeBases = kbData?.data || []

  // Local state for editing
  const [config, setConfig] = useState<Partial<BotConfig>>({})
  const [selectedChannels, setSelectedChannels] = useState<string[]>([])
  const [escalationRules, setEscalationRules] = useState<EscalationRule[]>([])

  // Initialize state when bot loads
  useState(() => {
    if (bot) {
      setConfig(bot.config || {})
      setSelectedChannels(bot.channel_ids || [])
      setEscalationRules(bot.config?.escalation_rules || [])
    }
  })

  // Update config mutation
  const updateConfigMutation = useMutation({
    mutationFn: (data: Partial<BotConfig>) => api.put(`/bots/${botId}/config`, data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: queryKeys.bots.detail(botId) })
      setHasChanges(false)
    },
  })

  // Activate/Deactivate mutations
  const activateMutation = useMutation({
    mutationFn: () => api.post(`/bots/${botId}/activate`),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: queryKeys.bots.detail(botId) })
    },
  })

  const deactivateMutation = useMutation({
    mutationFn: () => api.post(`/bots/${botId}/deactivate`),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: queryKeys.bots.detail(botId) })
    },
  })

  // Assign channel mutation
  const assignChannelMutation = useMutation({
    mutationFn: (channelId: string) => api.post(`/bots/${botId}/channels`, { channel_id: channelId }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: queryKeys.bots.detail(botId) })
    },
  })

  // Unassign channel mutation
  const unassignChannelMutation = useMutation({
    mutationFn: (channelId: string) => api.delete(`/bots/${botId}/channels/${channelId}`),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: queryKeys.bots.detail(botId) })
    },
  })

  // Test bot mutation
  const testMutation = useMutation({
    mutationFn: (input: BotTestInput) => api.post<BotTestResult>(`/bots/${botId}/test`, input),
    onSuccess: (result) => {
      setTestResult(result)
    },
  })

  // Delete mutation
  const deleteMutation = useMutation({
    mutationFn: () => api.delete(`/bots/${botId}`),
    onSuccess: () => {
      router.push('/bots')
    },
  })

  const handleSave = () => {
    updateConfigMutation.mutate({
      ...config,
      escalation_rules: escalationRules,
    })
  }

  const handleTest = () => {
    if (testMessage.trim()) {
      testMutation.mutate({ message: testMessage })
    }
  }

  const addEscalationRule = () => {
    const newRule: EscalationRule = {
      id: `rule_${Date.now()}`,
      type: 'keyword',
      value: '',
      priority: escalationRules.length + 1,
    }
    setEscalationRules([...escalationRules, newRule])
    setHasChanges(true)
  }

  const updateEscalationRule = (index: number, updates: Partial<EscalationRule>) => {
    const updated = [...escalationRules]
    updated[index] = { ...updated[index], ...updates }
    setEscalationRules(updated)
    setHasChanges(true)
  }

  const removeEscalationRule = (index: number) => {
    setEscalationRules(escalationRules.filter((_, i) => i !== index))
    setHasChanges(true)
  }

  if (isLoading) {
    return (
      <div className="flex flex-col h-full">
        <Header title="Bot Configuration" />
        <div className="p-6 space-y-4">
          <Skeleton className="h-8 w-64" />
          <Skeleton className="h-[400px] w-full" />
        </div>
      </div>
    )
  }

  if (!bot) {
    return (
      <div className="flex flex-col h-full">
        <Header title="Bot Not Found" />
        <div className="p-6">
          <Card>
            <CardContent className="py-12 text-center">
              <BotIcon className="mx-auto h-12 w-12 text-muted-foreground opacity-50" />
              <p className="mt-4 text-lg font-medium">Bot not found</p>
              <Button className="mt-4" onClick={() => router.push('/bots')}>
                <ArrowLeft className="h-4 w-4 mr-2" />
                Back to Bots
              </Button>
            </CardContent>
          </Card>
        </div>
      </div>
    )
  }

  return (
    <div className="flex flex-col h-full">
      <Header title={bot.name} />

      <div className="p-6 space-y-6 overflow-auto">
        {/* Header Actions */}
        <div className="flex items-center justify-between">
          <Button variant="ghost" onClick={() => router.push('/bots')}>
            <ArrowLeft className="h-4 w-4 mr-2" />
            Back to Bots
          </Button>
          <div className="flex items-center gap-2">
            <Button variant="outline" onClick={() => setTestDialogOpen(true)}>
              <TestTube className="h-4 w-4 mr-2" />
              Test Bot
            </Button>
            {bot.is_active ? (
              <Button variant="outline" onClick={() => deactivateMutation.mutate()}>
                <Pause className="h-4 w-4 mr-2" />
                Deactivate
              </Button>
            ) : (
              <Button variant="outline" onClick={() => activateMutation.mutate()}>
                <Play className="h-4 w-4 mr-2" />
                Activate
              </Button>
            )}
            <Button onClick={handleSave} disabled={!hasChanges || updateConfigMutation.isPending}>
              <Save className="h-4 w-4 mr-2" />
              {updateConfigMutation.isPending ? 'Saving...' : 'Save Changes'}
            </Button>
          </div>
        </div>

        {/* Status Card */}
        <Card>
          <CardContent className="py-4">
            <div className="flex items-center justify-between">
              <div className="flex items-center gap-4">
                <div className="flex h-12 w-12 items-center justify-center rounded-lg bg-primary/10">
                  <BotIcon className="h-6 w-6 text-primary" />
                </div>
                <div>
                  <h2 className="text-lg font-semibold">{bot.name}</h2>
                  <p className="text-sm text-muted-foreground">
                    {providerConfigs[bot.provider]?.label} / {bot.model}
                  </p>
                </div>
              </div>
              <Badge variant={bot.is_active ? 'success' : 'secondary'} className="text-sm">
                {bot.is_active ? 'Active' : 'Inactive'}
              </Badge>
            </div>
          </CardContent>
        </Card>

        {/* Configuration Tabs */}
        <Tabs value={activeTab} onValueChange={setActiveTab}>
          <TabsList className="grid w-full grid-cols-5">
            <TabsTrigger value="general" className="gap-2">
              <Settings className="h-4 w-4" />
              General
            </TabsTrigger>
            <TabsTrigger value="prompts" className="gap-2">
              <BotIcon className="h-4 w-4" />
              Prompts
            </TabsTrigger>
            <TabsTrigger value="channels" className="gap-2">
              <Radio className="h-4 w-4" />
              Channels
            </TabsTrigger>
            <TabsTrigger value="knowledge" className="gap-2">
              <BookOpen className="h-4 w-4" />
              Knowledge
            </TabsTrigger>
            <TabsTrigger value="escalation" className="gap-2">
              <Zap className="h-4 w-4" />
              Escalation
            </TabsTrigger>
          </TabsList>

          {/* General Tab */}
          <TabsContent value="general" className="space-y-4">
            <Card>
              <CardHeader>
                <CardTitle>Model Settings</CardTitle>
                <CardDescription>Configure the AI model parameters</CardDescription>
              </CardHeader>
              <CardContent className="space-y-6">
                <div className="grid grid-cols-2 gap-4">
                  <div className="space-y-2">
                    <Label>AI Provider</Label>
                    <Select value={bot.provider} disabled>
                      <SelectTrigger>
                        <SelectValue />
                      </SelectTrigger>
                      <SelectContent>
                        {Object.entries(providerConfigs).map(([key, cfg]) => (
                          <SelectItem key={key} value={key}>{cfg.label}</SelectItem>
                        ))}
                      </SelectContent>
                    </Select>
                  </div>
                  <div className="space-y-2">
                    <Label>Model</Label>
                    <Select value={bot.model} disabled>
                      <SelectTrigger>
                        <SelectValue />
                      </SelectTrigger>
                      <SelectContent>
                        {providerConfigs[bot.provider]?.models.map((m) => (
                          <SelectItem key={m} value={m}>{m}</SelectItem>
                        ))}
                      </SelectContent>
                    </Select>
                  </div>
                </div>

                <div className="space-y-2">
                  <Label>Temperature: {config.temperature ?? bot.config?.temperature ?? 0.7}</Label>
                  <Slider
                    value={[config.temperature ?? bot.config?.temperature ?? 0.7]}
                    onValueChange={([value]) => {
                      setConfig({ ...config, temperature: value })
                      setHasChanges(true)
                    }}
                    min={0}
                    max={1}
                    step={0.1}
                  />
                  <p className="text-xs text-muted-foreground">
                    Lower values make responses more focused, higher values more creative
                  </p>
                </div>

                <div className="grid grid-cols-2 gap-4">
                  <div className="space-y-2">
                    <Label>Max Tokens</Label>
                    <Input
                      type="number"
                      value={config.max_tokens ?? bot.config?.max_tokens ?? 1024}
                      onChange={(e) => {
                        setConfig({ ...config, max_tokens: parseInt(e.target.value) })
                        setHasChanges(true)
                      }}
                    />
                  </div>
                  <div className="space-y-2">
                    <Label>Confidence Threshold</Label>
                    <Input
                      type="number"
                      step="0.1"
                      min="0"
                      max="1"
                      value={config.confidence_threshold ?? bot.config?.confidence_threshold ?? 0.7}
                      onChange={(e) => {
                        setConfig({ ...config, confidence_threshold: parseFloat(e.target.value) })
                        setHasChanges(true)
                      }}
                    />
                  </div>
                </div>
              </CardContent>
            </Card>
          </TabsContent>

          {/* Prompts Tab */}
          <TabsContent value="prompts" className="space-y-4">
            <Card>
              <CardHeader>
                <CardTitle>System Prompt</CardTitle>
                <CardDescription>Define the bot's personality and behavior</CardDescription>
              </CardHeader>
              <CardContent>
                <Textarea
                  rows={8}
                  placeholder="You are a helpful customer service assistant..."
                  value={config.system_prompt ?? bot.config?.system_prompt ?? ''}
                  onChange={(e) => {
                    setConfig({ ...config, system_prompt: e.target.value })
                    setHasChanges(true)
                  }}
                />
              </CardContent>
            </Card>

            <div className="grid grid-cols-2 gap-4">
              <Card>
                <CardHeader>
                  <CardTitle>Welcome Message</CardTitle>
                  <CardDescription>First message sent to new conversations</CardDescription>
                </CardHeader>
                <CardContent>
                  <Textarea
                    rows={4}
                    placeholder="Hello! How can I help you today?"
                    value={config.welcome_message ?? bot.config?.welcome_message ?? ''}
                    onChange={(e) => {
                      setConfig({ ...config, welcome_message: e.target.value })
                      setHasChanges(true)
                    }}
                  />
                </CardContent>
              </Card>

              <Card>
                <CardHeader>
                  <CardTitle>Fallback Message</CardTitle>
                  <CardDescription>Sent when the bot doesn't understand</CardDescription>
                </CardHeader>
                <CardContent>
                  <Textarea
                    rows={4}
                    placeholder="I'm sorry, I didn't understand that..."
                    value={config.fallback_message ?? bot.config?.fallback_message ?? ''}
                    onChange={(e) => {
                      setConfig({ ...config, fallback_message: e.target.value })
                      setHasChanges(true)
                    }}
                  />
                </CardContent>
              </Card>
            </div>
          </TabsContent>

          {/* Channels Tab */}
          <TabsContent value="channels" className="space-y-4">
            <Card>
              <CardHeader>
                <CardTitle>Assigned Channels</CardTitle>
                <CardDescription>Select which channels this bot should handle</CardDescription>
              </CardHeader>
              <CardContent>
                <div className="space-y-3">
                  {channels.map((channel) => {
                    const isAssigned = bot.channel_ids?.includes(channel.id)
                    return (
                      <div
                        key={channel.id}
                        className={cn(
                          'flex items-center justify-between p-3 rounded-lg border',
                          isAssigned ? 'border-primary bg-primary/5' : 'border-border'
                        )}
                      >
                        <div className="flex items-center gap-3">
                          <Radio className="h-5 w-5 text-muted-foreground" />
                          <div>
                            <p className="font-medium">{channel.name}</p>
                            <p className="text-xs text-muted-foreground">{channel.type}</p>
                          </div>
                        </div>
                        {isAssigned ? (
                          <Button
                            variant="outline"
                            size="sm"
                            onClick={() => unassignChannelMutation.mutate(channel.id)}
                          >
                            <X className="h-4 w-4 mr-1" />
                            Remove
                          </Button>
                        ) : (
                          <Button
                            variant="outline"
                            size="sm"
                            onClick={() => assignChannelMutation.mutate(channel.id)}
                          >
                            <Plus className="h-4 w-4 mr-1" />
                            Assign
                          </Button>
                        )}
                      </div>
                    )
                  })}
                  {channels.length === 0 && (
                    <p className="text-center text-muted-foreground py-4">
                      No channels available. Create a channel first.
                    </p>
                  )}
                </div>
              </CardContent>
            </Card>
          </TabsContent>

          {/* Knowledge Tab */}
          <TabsContent value="knowledge" className="space-y-4">
            <Card>
              <CardHeader>
                <CardTitle>Knowledge Bases</CardTitle>
                <CardDescription>Link knowledge bases for RAG-powered responses</CardDescription>
              </CardHeader>
              <CardContent>
                <div className="space-y-3">
                  {knowledgeBases.map((kb) => {
                    const isLinked = (config.knowledge_base_ids ?? bot.config?.knowledge_base_ids ?? []).includes(kb.id)
                    return (
                      <div
                        key={kb.id}
                        className={cn(
                          'flex items-center justify-between p-3 rounded-lg border',
                          isLinked ? 'border-primary bg-primary/5' : 'border-border'
                        )}
                      >
                        <div className="flex items-center gap-3">
                          <BookOpen className="h-5 w-5 text-muted-foreground" />
                          <div>
                            <p className="font-medium">{kb.name}</p>
                            <p className="text-xs text-muted-foreground">
                              {kb.item_count} items · {kb.type}
                            </p>
                          </div>
                        </div>
                        <Button
                          variant="outline"
                          size="sm"
                          onClick={() => {
                            const currentIds = config.knowledge_base_ids ?? bot.config?.knowledge_base_ids ?? []
                            const newIds = isLinked
                              ? currentIds.filter((id) => id !== kb.id)
                              : [...currentIds, kb.id]
                            setConfig({ ...config, knowledge_base_ids: newIds })
                            setHasChanges(true)
                          }}
                        >
                          {isLinked ? (
                            <>
                              <X className="h-4 w-4 mr-1" />
                              Unlink
                            </>
                          ) : (
                            <>
                              <Plus className="h-4 w-4 mr-1" />
                              Link
                            </>
                          )}
                        </Button>
                      </div>
                    )
                  })}
                  {knowledgeBases.length === 0 && (
                    <p className="text-center text-muted-foreground py-4">
                      No knowledge bases available. Create one first.
                    </p>
                  )}
                </div>
              </CardContent>
            </Card>
          </TabsContent>

          {/* Escalation Tab */}
          <TabsContent value="escalation" className="space-y-4">
            <Card>
              <CardHeader>
                <div className="flex items-center justify-between">
                  <div>
                    <CardTitle>Escalation Rules</CardTitle>
                    <CardDescription>Define when to escalate to human agents</CardDescription>
                  </div>
                  <Button onClick={addEscalationRule}>
                    <Plus className="h-4 w-4 mr-2" />
                    Add Rule
                  </Button>
                </div>
              </CardHeader>
              <CardContent>
                <div className="space-y-4">
                  {escalationRules.map((rule, index) => (
                    <div key={rule.id} className="flex items-start gap-4 p-4 rounded-lg border">
                      <div className="flex-1 grid grid-cols-3 gap-4">
                        <div className="space-y-2">
                          <Label>Rule Type</Label>
                          <Select
                            value={rule.type}
                            onValueChange={(value: EscalationRuleType) =>
                              updateEscalationRule(index, { type: value })
                            }
                          >
                            <SelectTrigger>
                              <SelectValue />
                            </SelectTrigger>
                            <SelectContent>
                              {escalationRuleTypes.map((type) => (
                                <SelectItem key={type.value} value={type.value}>
                                  {type.label}
                                </SelectItem>
                              ))}
                            </SelectContent>
                          </Select>
                        </div>
                        {(rule.type === 'keyword' || rule.type === 'intent') && (
                          <div className="space-y-2">
                            <Label>Value</Label>
                            <Input
                              placeholder={rule.type === 'keyword' ? 'angry, refund, cancel' : 'complaint, cancellation'}
                              value={rule.value || ''}
                              onChange={(e) => updateEscalationRule(index, { value: e.target.value })}
                            />
                          </div>
                        )}
                        <div className="space-y-2">
                          <Label>Message (optional)</Label>
                          <Input
                            placeholder="Let me connect you with an agent..."
                            value={rule.message || ''}
                            onChange={(e) => updateEscalationRule(index, { message: e.target.value })}
                          />
                        </div>
                      </div>
                      <Button
                        variant="ghost"
                        size="icon"
                        className="text-destructive"
                        onClick={() => removeEscalationRule(index)}
                      >
                        <Trash2 className="h-4 w-4" />
                      </Button>
                    </div>
                  ))}
                  {escalationRules.length === 0 && (
                    <p className="text-center text-muted-foreground py-8">
                      No escalation rules defined. Add rules to automatically escalate conversations.
                    </p>
                  )}
                </div>
              </CardContent>
            </Card>
          </TabsContent>
        </Tabs>

        {/* Danger Zone */}
        <Card className="border-destructive/50">
          <CardHeader>
            <CardTitle className="text-destructive">Danger Zone</CardTitle>
            <CardDescription>Irreversible actions</CardDescription>
          </CardHeader>
          <CardContent>
            <Button
              variant="destructive"
              onClick={() => {
                if (confirm('Are you sure you want to delete this bot? This action cannot be undone.')) {
                  deleteMutation.mutate()
                }
              }}
            >
              <Trash2 className="h-4 w-4 mr-2" />
              Delete Bot
            </Button>
          </CardContent>
        </Card>
      </div>

      {/* Test Dialog */}
      <Dialog open={testDialogOpen} onOpenChange={setTestDialogOpen}>
        <DialogContent className="sm:max-w-[600px]">
          <DialogHeader>
            <DialogTitle>Test Bot</DialogTitle>
            <DialogDescription>
              Send a test message to see how the bot responds
            </DialogDescription>
          </DialogHeader>
          <div className="space-y-4">
            <div className="flex gap-2">
              <Input
                placeholder="Type a message..."
                value={testMessage}
                onChange={(e) => setTestMessage(e.target.value)}
                onKeyDown={(e) => e.key === 'Enter' && handleTest()}
              />
              <Button onClick={handleTest} disabled={testMutation.isPending || !testMessage.trim()}>
                <Send className="h-4 w-4" />
              </Button>
            </div>
            {testResult && (
              <Card>
                <CardContent className="pt-4 space-y-3">
                  <div>
                    <Label className="text-xs text-muted-foreground">Response</Label>
                    <p className="mt-1">{testResult.response}</p>
                  </div>
                  <div className="flex gap-4 text-sm">
                    <div>
                      <Label className="text-xs text-muted-foreground">Confidence</Label>
                      <p className="font-mono">{(testResult.confidence * 100).toFixed(1)}%</p>
                    </div>
                    {testResult.intent && (
                      <div>
                        <Label className="text-xs text-muted-foreground">Intent</Label>
                        <p>{testResult.intent}</p>
                      </div>
                    )}
                    <div>
                      <Label className="text-xs text-muted-foreground">Escalate?</Label>
                      <p>{testResult.should_escalate ? 'Yes' : 'No'}</p>
                    </div>
                  </div>
                </CardContent>
              </Card>
            )}
          </div>
          <DialogFooter>
            <Button variant="outline" onClick={() => setTestDialogOpen(false)}>
              Close
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  )
}
