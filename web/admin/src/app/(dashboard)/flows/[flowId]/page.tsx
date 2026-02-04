'use client'

import { useState, useCallback, useEffect } from 'react'
import { useParams, useRouter, useSearchParams } from 'next/navigation'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import {
  ArrowLeft,
  Save,
  Play,
  Pause,
  TestTube,
  Loader2,
  Plus,
} from 'lucide-react'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Badge } from '@/components/ui/badge'
import { Skeleton } from '@/components/ui/skeleton'
import { api } from '@/lib/api'
import { queryKeys } from '@/lib/query'
import type { Flow, FlowNode, UpdateFlowInput } from '@/types'
import { FlowCanvas } from '../components/flow-canvas'
import { NodePanel } from '../components/node-panel'
import { FlowTestPanel } from '../components/flow-test-panel'
import { NodeToolbar } from '../components/node-toolbar'

export default function FlowEditorPage() {
  const params = useParams()
  const router = useRouter()
  const searchParams = useSearchParams()
  const queryClient = useQueryClient()

  const flowId = params.flowId as string
  const showTest = searchParams.get('test') === 'true'

  const [selectedNodeId, setSelectedNodeId] = useState<string | null>(null)
  const [nodes, setNodes] = useState<FlowNode[]>([])
  const [hasChanges, setHasChanges] = useState(false)
  const [isTestOpen, setIsTestOpen] = useState(showTest)

  // Fetch flow
  const { data: flow, isLoading } = useQuery({
    queryKey: queryKeys.flows.detail(flowId),
    queryFn: () => api.get<Flow>(`/flows/${flowId}`),
    enabled: !!flowId,
  })

  // Initialize nodes when flow loads
  useEffect(() => {
    if (flow?.nodes) {
      setNodes(flow.nodes)
      setHasChanges(false)
    }
  }, [flow])

  // Save mutation
  const saveMutation = useMutation({
    mutationFn: (input: UpdateFlowInput) => api.put<Flow>(`/flows/${flowId}`, input),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: queryKeys.flows.detail(flowId) })
      setHasChanges(false)
    },
  })

  // Toggle active mutation
  const toggleActiveMutation = useMutation({
    mutationFn: () => {
      if (flow?.is_active) {
        return api.post(`/flows/${flowId}/deactivate`)
      }
      return api.post(`/flows/${flowId}/activate`)
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: queryKeys.flows.detail(flowId) })
      queryClient.invalidateQueries({ queryKey: queryKeys.flows.all })
    },
  })

  // Handle node changes
  const handleNodesChange = useCallback((updatedNodes: FlowNode[]) => {
    setNodes(updatedNodes)
    setHasChanges(true)
  }, [])

  // Handle save
  const handleSave = () => {
    if (!flow) return

    const input: UpdateFlowInput = {
      nodes,
      start_node_id: flow.start_node_id,
    }

    saveMutation.mutate(input)
  }

  // Handle add node
  const handleAddNode = (type: FlowNode['type']) => {
    const newNode: FlowNode = {
      id: `node-${Date.now()}`,
      type,
      content: type === 'end' ? '' : 'New node content...',
      transitions: [],
      position: { x: 250, y: (nodes.length + 1) * 150 },
    }

    if (type === 'question') {
      newNode.quick_replies = [
        { id: 'option-1', title: 'Option 1' },
        { id: 'option-2', title: 'Option 2' },
      ]
    }

    if (type === 'action') {
      newNode.actions = [{ type: 'tag', config: { tag: 'new-tag' } }]
    }

    setNodes([...nodes, newNode])
    setSelectedNodeId(newNode.id)
    setHasChanges(true)
  }

  // Handle node selection
  const handleNodeSelect = (nodeId: string | null) => {
    setSelectedNodeId(nodeId)
  }

  // Handle node update from panel
  const handleNodeUpdate = (updatedNode: FlowNode) => {
    setNodes(nodes.map((n) => (n.id === updatedNode.id ? updatedNode : n)))
    setHasChanges(true)
  }

  // Handle node delete
  const handleNodeDelete = (nodeId: string) => {
    // Remove node and update transitions that point to it
    const updatedNodes = nodes
      .filter((n) => n.id !== nodeId)
      .map((n) => ({
        ...n,
        transitions: n.transitions.filter((t) => t.to_node_id !== nodeId),
      }))

    setNodes(updatedNodes)
    setSelectedNodeId(null)
    setHasChanges(true)
  }

  const selectedNode = nodes.find((n) => n.id === selectedNodeId) || null

  if (isLoading) {
    return (
      <div className="flex h-full flex-col">
        {/* Header skeleton */}
        <div className="flex items-center justify-between border-b px-4 py-3">
          <div className="flex items-center gap-4">
            <Skeleton className="h-8 w-8" />
            <Skeleton className="h-6 w-48" />
          </div>
          <div className="flex gap-2">
            <Skeleton className="h-9 w-20" />
            <Skeleton className="h-9 w-20" />
          </div>
        </div>
        {/* Content skeleton */}
        <div className="flex-1 p-4">
          <Skeleton className="h-full w-full" />
        </div>
      </div>
    )
  }

  if (!flow) {
    return (
      <div className="flex h-full items-center justify-center">
        <div className="text-center">
          <h2 className="text-lg font-medium">Flow not found</h2>
          <Button variant="link" onClick={() => router.push('/flows')}>
            Back to flows
          </Button>
        </div>
      </div>
    )
  }

  return (
    <div className="flex h-full flex-col">
      {/* Header */}
      <div className="flex items-center justify-between border-b px-4 py-3">
        <div className="flex items-center gap-4">
          <Button variant="ghost" size="icon" onClick={() => router.push('/flows')}>
            <ArrowLeft className="h-4 w-4" />
          </Button>
          <div>
            <h1 className="text-lg font-medium">{flow.name}</h1>
            <div className="flex items-center gap-2 text-sm text-muted-foreground">
              <Badge variant={flow.is_active ? 'default' : 'secondary'} className="text-xs">
                {flow.is_active ? 'Active' : 'Inactive'}
              </Badge>
              {hasChanges && (
                <Badge variant="outline" className="text-xs">
                  Unsaved changes
                </Badge>
              )}
            </div>
          </div>
        </div>

        <div className="flex items-center gap-2">
          <Button
            variant="outline"
            size="sm"
            onClick={() => setIsTestOpen(true)}
          >
            <TestTube className="mr-2 h-4 w-4" />
            Test
          </Button>

          <Button
            variant="outline"
            size="sm"
            onClick={() => toggleActiveMutation.mutate()}
            disabled={toggleActiveMutation.isPending}
          >
            {toggleActiveMutation.isPending ? (
              <Loader2 className="mr-2 h-4 w-4 animate-spin" />
            ) : flow.is_active ? (
              <Pause className="mr-2 h-4 w-4" />
            ) : (
              <Play className="mr-2 h-4 w-4" />
            )}
            {flow.is_active ? 'Deactivate' : 'Activate'}
          </Button>

          <Button
            size="sm"
            onClick={handleSave}
            disabled={!hasChanges || saveMutation.isPending}
          >
            {saveMutation.isPending ? (
              <Loader2 className="mr-2 h-4 w-4 animate-spin" />
            ) : (
              <Save className="mr-2 h-4 w-4" />
            )}
            Save
          </Button>
        </div>
      </div>

      {/* Main Content */}
      <div className="flex flex-1 overflow-hidden">
        {/* Node Toolbar */}
        <NodeToolbar onAddNode={handleAddNode} />

        {/* Canvas */}
        <div className="flex-1">
          <FlowCanvas
            nodes={nodes}
            startNodeId={flow.start_node_id}
            selectedNodeId={selectedNodeId}
            onNodesChange={handleNodesChange}
            onNodeSelect={handleNodeSelect}
          />
        </div>

        {/* Node Panel */}
        {selectedNode && (
          <NodePanel
            node={selectedNode}
            allNodes={nodes}
            onUpdate={handleNodeUpdate}
            onDelete={() => handleNodeDelete(selectedNode.id)}
            onClose={() => setSelectedNodeId(null)}
          />
        )}
      </div>

      {/* Test Panel */}
      <FlowTestPanel
        flowId={flowId}
        open={isTestOpen}
        onOpenChange={setIsTestOpen}
      />
    </div>
  )
}
