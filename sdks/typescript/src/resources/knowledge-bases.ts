/**
 * Knowledge Bases Resource
 */

import type { HttpClient } from '../utils/http';
import type { PaginatedResponse } from '../types/common';
import type {
  KnowledgeBase,
  Document,
  CreateKnowledgeBaseInput,
  UpdateKnowledgeBaseInput,
  ListKnowledgeBasesParams,
  UploadDocumentInput,
  ListDocumentsParams,
  QueryKnowledgeBaseInput,
  QueryResult,
} from '../types/knowledge';

export class KnowledgeBasesResource {
  constructor(private http: HttpClient) {}

  /**
   * List knowledge bases
   */
  async list(params?: ListKnowledgeBasesParams): Promise<PaginatedResponse<KnowledgeBase>> {
    return this.http.get<PaginatedResponse<KnowledgeBase>>('/knowledge-bases', { params });
  }

  /**
   * Get a single knowledge base
   */
  async get(id: string): Promise<KnowledgeBase> {
    return this.http.get<KnowledgeBase>(`/knowledge-bases/${id}`);
  }

  /**
   * Create a new knowledge base
   */
  async create(data: CreateKnowledgeBaseInput): Promise<KnowledgeBase> {
    return this.http.post<KnowledgeBase>('/knowledge-bases', data);
  }

  /**
   * Update a knowledge base
   */
  async update(id: string, data: UpdateKnowledgeBaseInput): Promise<KnowledgeBase> {
    return this.http.patch<KnowledgeBase>(`/knowledge-bases/${id}`, data);
  }

  /**
   * Delete a knowledge base
   */
  async delete(id: string): Promise<void> {
    await this.http.delete<void>(`/knowledge-bases/${id}`);
  }

  /**
   * Query a knowledge base
   */
  async query(id: string, input: QueryKnowledgeBaseInput): Promise<QueryResult> {
    return this.http.post<QueryResult>(`/knowledge-bases/${id}/query`, input);
  }

  /**
   * Simple query (returns text results)
   */
  async search(id: string, query: string, topK = 5): Promise<string[]> {
    const result = await this.query(id, { query, topK });
    return result.chunks.map((c) => c.content);
  }

  /**
   * Reprocess all documents in a knowledge base
   */
  async reprocess(id: string): Promise<void> {
    await this.http.post<void>(`/knowledge-bases/${id}/reprocess`);
  }

  /**
   * Get knowledge base statistics
   */
  async getStats(id: string): Promise<{
    documentCount: number;
    totalChunks: number;
    totalTokens: number;
    avgChunkSize: number;
    storageUsed: number;
  }> {
    return this.http.get<{
      documentCount: number;
      totalChunks: number;
      totalTokens: number;
      avgChunkSize: number;
      storageUsed: number;
    }>(`/knowledge-bases/${id}/stats`);
  }

  // Document methods
  /**
   * List documents in a knowledge base
   */
  async listDocuments(
    knowledgeBaseId: string,
    params?: ListDocumentsParams
  ): Promise<PaginatedResponse<Document>> {
    return this.http.get<PaginatedResponse<Document>>(
      `/knowledge-bases/${knowledgeBaseId}/documents`,
      { params }
    );
  }

  /**
   * Get a single document
   */
  async getDocument(knowledgeBaseId: string, documentId: string): Promise<Document> {
    return this.http.get<Document>(
      `/knowledge-bases/${knowledgeBaseId}/documents/${documentId}`
    );
  }

  /**
   * Upload a document
   */
  async uploadDocument(
    knowledgeBaseId: string,
    input: UploadDocumentInput
  ): Promise<Document> {
    if (input.file) {
      return this.http.upload<Document>(
        `/knowledge-bases/${knowledgeBaseId}/documents`,
        input.file,
        'file',
        {
          name: input.name || '',
          type: input.type || '',
          metadata: input.metadata ? JSON.stringify(input.metadata) : '',
        }
      );
    }

    return this.http.post<Document>(
      `/knowledge-bases/${knowledgeBaseId}/documents`,
      input
    );
  }

  /**
   * Upload multiple documents
   */
  async uploadDocuments(
    knowledgeBaseId: string,
    documents: UploadDocumentInput[]
  ): Promise<Document[]> {
    const results: Document[] = [];
    for (const doc of documents) {
      const result = await this.uploadDocument(knowledgeBaseId, doc);
      results.push(result);
    }
    return results;
  }

  /**
   * Delete a document
   */
  async deleteDocument(knowledgeBaseId: string, documentId: string): Promise<void> {
    await this.http.delete<void>(
      `/knowledge-bases/${knowledgeBaseId}/documents/${documentId}`
    );
  }

  /**
   * Reprocess a document
   */
  async reprocessDocument(knowledgeBaseId: string, documentId: string): Promise<void> {
    await this.http.post<void>(
      `/knowledge-bases/${knowledgeBaseId}/documents/${documentId}/reprocess`
    );
  }

  /**
   * Get document content/chunks
   */
  async getDocumentChunks(
    knowledgeBaseId: string,
    documentId: string
  ): Promise<{ chunks: { content: string; index: number }[] }> {
    return this.http.get<{ chunks: { content: string; index: number }[] }>(
      `/knowledge-bases/${knowledgeBaseId}/documents/${documentId}/chunks`
    );
  }
}
