/**
 * Flows Resource
 */

import type { HttpClient } from '../utils/http';
import type { PaginatedResponse } from '../types/common';
import type {
  Flow,
  FlowExecution,
  FlowNodeType,
  CreateFlowInput,
  UpdateFlowInput,
  ListFlowsParams,
  ExecuteFlowInput,
} from '../types/flow';

export class FlowsResource {
  constructor(private http: HttpClient) {}

  /**
   * List flows
   */
  async list(params?: ListFlowsParams): Promise<PaginatedResponse<Flow>> {
    return this.http.get<PaginatedResponse<Flow>>('/flows', { params });
  }

  /**
   * Get a single flow
   */
  async get(id: string): Promise<Flow> {
    return this.http.get<Flow>(`/flows/${id}`);
  }

  /**
   * Create a new flow
   */
  async create(data: CreateFlowInput): Promise<Flow> {
    return this.http.post<Flow>('/flows', data);
  }

  /**
   * Update a flow
   */
  async update(id: string, data: UpdateFlowInput): Promise<Flow> {
    return this.http.patch<Flow>(`/flows/${id}`, data);
  }

  /**
   * Delete a flow
   */
  async delete(id: string): Promise<void> {
    await this.http.delete<void>(`/flows/${id}`);
  }

  /**
   * Activate a flow
   */
  async activate(id: string): Promise<Flow> {
    return this.update(id, { status: 'active' });
  }

  /**
   * Deactivate a flow
   */
  async deactivate(id: string): Promise<Flow> {
    return this.update(id, { status: 'inactive' });
  }

  /**
   * Duplicate a flow
   */
  async duplicate(id: string, name?: string): Promise<Flow> {
    return this.http.post<Flow>(`/flows/${id}/duplicate`, { name });
  }

  /**
   * Execute a flow manually
   */
  async execute(id: string, input: ExecuteFlowInput): Promise<FlowExecution> {
    return this.http.post<FlowExecution>(`/flows/${id}/execute`, input);
  }

  /**
   * Get flow execution history
   */
  async getExecutions(
    id: string,
    params?: { limit?: number; offset?: number; status?: string }
  ): Promise<PaginatedResponse<FlowExecution>> {
    return this.http.get<PaginatedResponse<FlowExecution>>(
      `/flows/${id}/executions`,
      { params }
    );
  }

  /**
   * Get a specific execution
   */
  async getExecution(flowId: string, executionId: string): Promise<FlowExecution> {
    return this.http.get<FlowExecution>(
      `/flows/${flowId}/executions/${executionId}`
    );
  }

  /**
   * Cancel an execution
   */
  async cancelExecution(flowId: string, executionId: string): Promise<void> {
    await this.http.post<void>(
      `/flows/${flowId}/executions/${executionId}/cancel`
    );
  }

  /**
   * Get available node types
   */
  async getNodeTypes(): Promise<
    {
      type: FlowNodeType;
      label: string;
      description: string;
      category: string;
      inputs: { name: string; type: string; required: boolean }[];
      outputs: { name: string; type: string }[];
    }[]
  > {
    return this.http.get<
      {
        type: FlowNodeType;
        label: string;
        description: string;
        category: string;
        inputs: { name: string; type: string; required: boolean }[];
        outputs: { name: string; type: string }[];
      }[]
    >('/flows/node-types');
  }

  /**
   * Validate flow configuration
   */
  async validate(id: string): Promise<{
    valid: boolean;
    errors: { nodeId: string; message: string }[];
    warnings: { nodeId: string; message: string }[];
  }> {
    return this.http.post<{
      valid: boolean;
      errors: { nodeId: string; message: string }[];
      warnings: { nodeId: string; message: string }[];
    }>(`/flows/${id}/validate`);
  }

  /**
   * Export flow as JSON
   */
  async export(id: string): Promise<Flow> {
    return this.http.get<Flow>(`/flows/${id}/export`);
  }

  /**
   * Import flow from JSON
   */
  async import(flowData: Flow): Promise<Flow> {
    return this.http.post<Flow>('/flows/import', flowData);
  }

  /**
   * Get flow statistics
   */
  async getStats(
    id: string,
    period?: 'day' | 'week' | 'month'
  ): Promise<{
    totalExecutions: number;
    successRate: number;
    avgDuration: number;
    nodeStats: { nodeId: string; executionCount: number; avgDuration: number }[];
  }> {
    return this.http.get<{
      totalExecutions: number;
      successRate: number;
      avgDuration: number;
      nodeStats: { nodeId: string; executionCount: number; avgDuration: number }[];
    }>(`/flows/${id}/stats`, { params: { period } });
  }
}
