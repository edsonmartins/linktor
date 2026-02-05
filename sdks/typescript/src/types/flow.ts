/**
 * Flow types
 */

import type { PaginationParams, Timestamps } from './common';

export interface Flow extends Timestamps {
  id: string;
  tenantId: string;
  name: string;
  description?: string;
  status: FlowStatus;
  version: number;
  nodes: FlowNode[];
  edges: FlowEdge[];
  variables?: FlowVariable[];
  metadata?: Record<string, unknown>;
}

export type FlowStatus = 'active' | 'inactive' | 'draft';

export interface FlowNode {
  id: string;
  type: FlowNodeType;
  position: { x: number; y: number };
  data: FlowNodeData;
}

export type FlowNodeType =
  | 'start'
  | 'message'
  | 'condition'
  | 'action'
  | 'input'
  | 'ai'
  | 'api'
  | 'delay'
  | 'assign'
  | 'end';

export interface FlowNodeData {
  label?: string;
  // Message node
  messageContent?: string;
  messageType?: 'text' | 'media' | 'buttons' | 'list';
  buttons?: FlowButton[];
  // Condition node
  conditions?: FlowCondition[];
  // Action node
  actionType?: string;
  actionParams?: Record<string, unknown>;
  // Input node
  inputVariable?: string;
  inputValidation?: InputValidation;
  // AI node
  agentId?: string;
  prompt?: string;
  outputVariable?: string;
  // API node
  apiUrl?: string;
  apiMethod?: 'GET' | 'POST' | 'PUT' | 'DELETE';
  apiHeaders?: Record<string, string>;
  apiBody?: string;
  // Delay node
  delaySeconds?: number;
  // Assign node
  assignments?: VariableAssignment[];
}

export interface FlowButton {
  id: string;
  text: string;
  value?: string;
}

export interface FlowCondition {
  id: string;
  variable: string;
  operator: ConditionOperator;
  value: unknown;
  targetNodeId: string;
}

export type ConditionOperator =
  | 'equals'
  | 'not_equals'
  | 'contains'
  | 'not_contains'
  | 'starts_with'
  | 'ends_with'
  | 'greater_than'
  | 'less_than'
  | 'is_empty'
  | 'is_not_empty'
  | 'matches_regex';

export interface InputValidation {
  type: 'text' | 'number' | 'email' | 'phone' | 'date' | 'regex';
  pattern?: string;
  errorMessage?: string;
  retries?: number;
}

export interface VariableAssignment {
  variable: string;
  value: unknown;
  expression?: string;
}

export interface FlowEdge {
  id: string;
  source: string;
  target: string;
  sourceHandle?: string;
  targetHandle?: string;
  label?: string;
  condition?: string;
}

export interface FlowVariable {
  name: string;
  type: 'string' | 'number' | 'boolean' | 'array' | 'object';
  defaultValue?: unknown;
  description?: string;
}

// Execution types
export interface FlowExecution {
  id: string;
  flowId: string;
  conversationId: string;
  status: FlowExecutionStatus;
  currentNodeId?: string;
  variables: Record<string, unknown>;
  history: FlowExecutionStep[];
  startedAt: string;
  completedAt?: string;
  error?: string;
}

export type FlowExecutionStatus = 'running' | 'waiting' | 'completed' | 'failed' | 'cancelled';

export interface FlowExecutionStep {
  nodeId: string;
  nodeType: FlowNodeType;
  startedAt: string;
  completedAt?: string;
  input?: Record<string, unknown>;
  output?: Record<string, unknown>;
  error?: string;
}

// Request types
export interface CreateFlowInput {
  name: string;
  description?: string;
  nodes?: FlowNode[];
  edges?: FlowEdge[];
  variables?: FlowVariable[];
  metadata?: Record<string, unknown>;
}

export interface UpdateFlowInput {
  name?: string;
  description?: string;
  status?: FlowStatus;
  nodes?: FlowNode[];
  edges?: FlowEdge[];
  variables?: FlowVariable[];
  metadata?: Record<string, unknown>;
}

export interface ListFlowsParams extends PaginationParams {
  status?: FlowStatus;
  search?: string;
}

export interface ExecuteFlowInput {
  conversationId: string;
  variables?: Record<string, unknown>;
  startNodeId?: string;
}
