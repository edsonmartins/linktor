'use client'

import { useCallback, useMemo } from 'react'
import {
  ReactFlow,
  Background,
  Controls,
  MiniMap,
  useNodesState,
  useEdgesState,
  addEdge,
  type Node,
  type Edge,
  type Connection,
  type OnNodesChange,
  type OnEdgesChange,
  MarkerType,
} from '@xyflow/react'
import '@xyflow/react/dist/style.css'

import type { FlowNode } from '@/types'
import { MessageNode } from './nodes/message-node'
import { QuestionNode } from './nodes/question-node'
import { ConditionNode } from './nodes/condition-node'
import { ActionNode } from './nodes/action-node'
import { EndNode } from './nodes/end-node'
import { VRENode } from './nodes/vre-node'

interface FlowCanvasProps {
  nodes: FlowNode[]
  startNodeId: string
  selectedNodeId: string | null
  onNodesChange: (nodes: FlowNode[]) => void
  onNodeSelect: (nodeId: string | null) => void
}

// Custom node types - using any to avoid complex type gymnastics with @xyflow/react
const nodeTypes = {
  message: MessageNode,
  question: QuestionNode,
  condition: ConditionNode,
  action: ActionNode,
  vre: VRENode,
  end: EndNode,
} as const

// Convert FlowNode to React Flow Node
function toReactFlowNode(node: FlowNode, isStart: boolean, isSelected: boolean): Node {
  return {
    id: node.id,
    type: node.type,
    position: node.position || { x: 0, y: 0 },
    data: {
      ...node,
      isStart,
      isSelected,
    },
    selected: isSelected,
  }
}

// Convert FlowNode transitions to React Flow Edges
function toReactFlowEdges(nodes: FlowNode[]): Edge[] {
  const edges: Edge[] = []

  nodes.forEach((node) => {
    node.transitions.forEach((transition, index) => {
      edges.push({
        id: `${node.id}-${transition.to_node_id}-${index}`,
        source: node.id,
        target: transition.to_node_id,
        label: transition.condition !== 'default' ? transition.value || transition.condition : undefined,
        markerEnd: {
          type: MarkerType.ArrowClosed,
          width: 20,
          height: 20,
        },
        style: {
          strokeWidth: 2,
        },
        labelStyle: {
          fontSize: 12,
          fontWeight: 500,
        },
        labelBgStyle: {
          fill: '#fff',
          fillOpacity: 0.9,
        },
      })
    })
  })

  return edges
}

// Convert React Flow position changes back to FlowNode
function updateNodePositions(nodes: FlowNode[], changes: Node[]): FlowNode[] {
  const positionMap = new Map<string, { x: number; y: number }>()

  changes.forEach((change) => {
    if (change.position) {
      positionMap.set(change.id, change.position)
    }
  })

  return nodes.map((node) => {
    const newPosition = positionMap.get(node.id)
    if (newPosition) {
      return { ...node, position: newPosition }
    }
    return node
  })
}

export function FlowCanvas({
  nodes,
  startNodeId,
  selectedNodeId,
  onNodesChange,
  onNodeSelect,
}: FlowCanvasProps) {
  // Convert to React Flow format
  const rfNodes = useMemo(
    () =>
      nodes.map((node) =>
        toReactFlowNode(node, node.id === startNodeId, node.id === selectedNodeId)
      ),
    [nodes, startNodeId, selectedNodeId]
  )

  const rfEdges = useMemo(() => toReactFlowEdges(nodes), [nodes])

  const [flowNodes, setFlowNodes, onFlowNodesChange] = useNodesState(rfNodes)
  const [flowEdges, setFlowEdges, onFlowEdgesChange] = useEdgesState(rfEdges)

  // Sync external nodes with internal state
  useMemo(() => {
    setFlowNodes(rfNodes)
  }, [rfNodes, setFlowNodes])

  useMemo(() => {
    setFlowEdges(rfEdges)
  }, [rfEdges, setFlowEdges])

  // Handle node changes (position, selection, etc.)
  const handleNodesChange: OnNodesChange = useCallback(
    (changes) => {
      onFlowNodesChange(changes)

      // Check for position changes and update parent
      const positionChanges = changes.filter(
        (change) => change.type === 'position' && change.dragging === false
      )

      if (positionChanges.length > 0) {
        // Get updated positions from the internal state after change
        setTimeout(() => {
          const updatedNodes = updateNodePositions(nodes, flowNodes)
          if (JSON.stringify(updatedNodes) !== JSON.stringify(nodes)) {
            onNodesChange(updatedNodes)
          }
        }, 0)
      }
    },
    [onFlowNodesChange, nodes, flowNodes, onNodesChange]
  )

  // Handle edge changes
  const handleEdgesChange: OnEdgesChange = useCallback(
    (changes) => {
      onFlowEdgesChange(changes)
    },
    [onFlowEdgesChange]
  )

  // Handle new connections
  const handleConnect = useCallback(
    (connection: Connection) => {
      if (!connection.source || !connection.target) return

      // Update the source node's transitions
      const updatedNodes = nodes.map((node) => {
        if (node.id === connection.source) {
          // Check if transition already exists
          const exists = node.transitions.some(
            (t) => t.to_node_id === connection.target
          )
          if (exists) return node

          return {
            ...node,
            transitions: [
              ...node.transitions,
              {
                id: `trans-${Date.now()}`,
                to_node_id: connection.target!,
                condition: 'default' as const,
              },
            ],
          }
        }
        return node
      })

      onNodesChange(updatedNodes)
    },
    [nodes, onNodesChange]
  )

  // Handle node click
  const handleNodeClick = useCallback(
    (_event: React.MouseEvent, node: Node) => {
      onNodeSelect(node.id)
    },
    [onNodeSelect]
  )

  // Handle pane click (deselect)
  const handlePaneClick = useCallback(() => {
    onNodeSelect(null)
  }, [onNodeSelect])

  return (
    <div className="h-full w-full">
      <ReactFlow
        nodes={flowNodes}
        edges={flowEdges}
        onNodesChange={handleNodesChange}
        onEdgesChange={handleEdgesChange}
        onConnect={handleConnect}
        onNodeClick={handleNodeClick}
        onPaneClick={handlePaneClick}
        nodeTypes={nodeTypes}
        fitView
        snapToGrid
        snapGrid={[15, 15]}
        defaultEdgeOptions={{
          type: 'smoothstep',
        }}
        proOptions={{ hideAttribution: true }}
      >
        <Background gap={15} size={1} />
        <Controls />
        <MiniMap
          nodeStrokeColor={(n) => {
            if (n.data?.isStart) return '#22c55e'
            if (n.type === 'end') return '#ef4444'
            return '#64748b'
          }}
          nodeColor={(n) => {
            if (n.data?.isStart) return '#dcfce7'
            if (n.type === 'end') return '#fee2e2'
            return '#f1f5f9'
          }}
          nodeBorderRadius={8}
        />
      </ReactFlow>
    </div>
  )
}
