"""
Knowledge Base types
"""

from datetime import datetime
from enum import Enum
from typing import Any, Optional
from pydantic import BaseModel, Field

from linktor.types.common import PaginationParams


class KnowledgeBaseStatus(str, Enum):
    """Knowledge base status"""

    ACTIVE = "active"
    PROCESSING = "processing"
    ERROR = "error"
    EMPTY = "empty"


class DocumentType(str, Enum):
    """Document type"""

    PDF = "pdf"
    TXT = "txt"
    DOCX = "docx"
    HTML = "html"
    MARKDOWN = "markdown"
    CSV = "csv"
    JSON = "json"


class DocumentStatus(str, Enum):
    """Document status"""

    PENDING = "pending"
    PROCESSING = "processing"
    COMPLETED = "completed"
    FAILED = "failed"


class KnowledgeBase(BaseModel):
    """Knowledge base model"""

    id: str
    tenant_id: str = Field(alias="tenantId")
    name: str
    description: Optional[str] = None
    status: KnowledgeBaseStatus
    embedding_model: str = Field(alias="embeddingModel")
    chunk_size: int = Field(alias="chunkSize")
    chunk_overlap: int = Field(alias="chunkOverlap")
    document_count: int = Field(alias="documentCount")
    total_chunks: int = Field(alias="totalChunks")
    metadata: Optional[dict[str, Any]] = None
    created_at: datetime = Field(alias="createdAt")
    updated_at: datetime = Field(alias="updatedAt")

    class Config:
        populate_by_name = True


class Document(BaseModel):
    """Document model"""

    id: str
    knowledge_base_id: str = Field(alias="knowledgeBaseId")
    name: str
    type: DocumentType
    source_url: Optional[str] = Field(None, alias="sourceUrl")
    status: DocumentStatus
    size: int
    chunk_count: int = Field(alias="chunkCount")
    metadata: Optional[dict[str, Any]] = None
    error: Optional[str] = None
    created_at: datetime = Field(alias="createdAt")
    updated_at: datetime = Field(alias="updatedAt")

    class Config:
        populate_by_name = True


class DocumentChunk(BaseModel):
    """Document chunk"""

    id: str
    document_id: str = Field(alias="documentId")
    content: str
    chunk_index: int = Field(alias="chunkIndex")
    token_count: int = Field(alias="tokenCount")
    metadata: Optional[dict[str, Any]] = None

    class Config:
        populate_by_name = True


class ScoredChunk(DocumentChunk):
    """Chunk with similarity score"""

    score: float
    document: Optional[Document] = None


class QueryKnowledgeBaseInput(BaseModel):
    """Query knowledge base input"""

    query: str
    top_k: Optional[int] = Field(None, alias="topK")
    threshold: Optional[float] = None
    filter: Optional[dict[str, Any]] = None

    class Config:
        populate_by_name = True


class QueryResult(BaseModel):
    """Query result"""

    chunks: list[ScoredChunk]
    query: str
    model: str


class CreateKnowledgeBaseInput(BaseModel):
    """Create knowledge base input"""

    name: str
    description: Optional[str] = None
    embedding_model: Optional[str] = Field(None, alias="embeddingModel")
    chunk_size: Optional[int] = Field(None, alias="chunkSize")
    chunk_overlap: Optional[int] = Field(None, alias="chunkOverlap")
    metadata: Optional[dict[str, Any]] = None

    class Config:
        populate_by_name = True


class UpdateKnowledgeBaseInput(BaseModel):
    """Update knowledge base input"""

    name: Optional[str] = None
    description: Optional[str] = None
    metadata: Optional[dict[str, Any]] = None


class ListKnowledgeBasesParams(PaginationParams):
    """List knowledge bases parameters"""

    status: Optional[KnowledgeBaseStatus] = None
    search: Optional[str] = None


class UploadDocumentInput(BaseModel):
    """Upload document input"""

    name: Optional[str] = None
    type: Optional[DocumentType] = None
    content: Optional[str] = None
    source_url: Optional[str] = Field(None, alias="sourceUrl")
    metadata: Optional[dict[str, Any]] = None

    class Config:
        populate_by_name = True


class ListDocumentsParams(PaginationParams):
    """List documents parameters"""

    status: Optional[DocumentStatus] = None
    type: Optional[DocumentType] = None
    search: Optional[str] = None


__all__ = [
    "KnowledgeBaseStatus",
    "DocumentType",
    "DocumentStatus",
    "KnowledgeBase",
    "Document",
    "DocumentChunk",
    "ScoredChunk",
    "QueryKnowledgeBaseInput",
    "QueryResult",
    "CreateKnowledgeBaseInput",
    "UpdateKnowledgeBaseInput",
    "ListKnowledgeBasesParams",
    "UploadDocumentInput",
    "ListDocumentsParams",
]
