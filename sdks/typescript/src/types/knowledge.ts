/**
 * Knowledge Base types
 */

import type { PaginationParams, Timestamps } from './common';

export interface KnowledgeBase extends Timestamps {
  id: string;
  tenantId: string;
  name: string;
  description?: string;
  status: KnowledgeBaseStatus;
  embeddingModel: string;
  chunkSize: number;
  chunkOverlap: number;
  documentCount: number;
  totalChunks: number;
  metadata?: Record<string, unknown>;
}

export type KnowledgeBaseStatus = 'active' | 'processing' | 'error' | 'empty';

export interface Document extends Timestamps {
  id: string;
  knowledgeBaseId: string;
  name: string;
  type: DocumentType;
  sourceUrl?: string;
  status: DocumentStatus;
  size: number;
  chunkCount: number;
  metadata?: Record<string, unknown>;
  error?: string;
}

export type DocumentType = 'pdf' | 'txt' | 'docx' | 'html' | 'markdown' | 'csv' | 'json';
export type DocumentStatus = 'pending' | 'processing' | 'completed' | 'failed';

export interface DocumentChunk {
  id: string;
  documentId: string;
  content: string;
  chunkIndex: number;
  tokenCount: number;
  metadata?: Record<string, unknown>;
}

// Query types
export interface QueryKnowledgeBaseInput {
  query: string;
  topK?: number;
  threshold?: number;
  filter?: Record<string, unknown>;
}

export interface QueryResult {
  chunks: ScoredChunk[];
  query: string;
  model: string;
}

export interface ScoredChunk extends DocumentChunk {
  score: number;
  document?: Document;
}

// Request types
export interface CreateKnowledgeBaseInput {
  name: string;
  description?: string;
  embeddingModel?: string;
  chunkSize?: number;
  chunkOverlap?: number;
  metadata?: Record<string, unknown>;
}

export interface UpdateKnowledgeBaseInput {
  name?: string;
  description?: string;
  metadata?: Record<string, unknown>;
}

export interface ListKnowledgeBasesParams extends PaginationParams {
  status?: KnowledgeBaseStatus;
  search?: string;
}

export interface UploadDocumentInput {
  name?: string;
  type?: DocumentType;
  content?: string;
  file?: File | Blob;
  sourceUrl?: string;
  metadata?: Record<string, unknown>;
}

export interface ListDocumentsParams extends PaginationParams {
  status?: DocumentStatus;
  type?: DocumentType;
  search?: string;
}
