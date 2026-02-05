<?php

declare(strict_types=1);

namespace Linktor\Resources;

use Linktor\LinktorClient;
use Linktor\Types\KnowledgeBase;
use Linktor\Types\Document;
use Linktor\Types\PaginatedResponse;

class KnowledgeBasesResource
{
    private LinktorClient $client;

    public function __construct(LinktorClient $client)
    {
        $this->client = $client;
    }

    public function list(array $params = []): PaginatedResponse
    {
        $query = $this->buildQuery($params);
        $data = $this->client->get("/knowledge-bases{$query}");
        $response = PaginatedResponse::fromArray($data);
        $response->data = array_map(fn($kb) => KnowledgeBase::fromArray($kb), $response->data);
        return $response;
    }

    public function get(string $id): KnowledgeBase
    {
        $data = $this->client->get("/knowledge-bases/{$id}");
        return KnowledgeBase::fromArray($data);
    }

    public function create(array $input): KnowledgeBase
    {
        $data = $this->client->post('/knowledge-bases', $input);
        return KnowledgeBase::fromArray($data);
    }

    public function update(string $id, array $input): KnowledgeBase
    {
        $data = $this->client->patch("/knowledge-bases/{$id}", $input);
        return KnowledgeBase::fromArray($data);
    }

    public function delete(string $id): void
    {
        $this->client->delete("/knowledge-bases/{$id}");
    }

    // Document operations
    public function listDocuments(string $knowledgeBaseId, array $params = []): PaginatedResponse
    {
        $query = $this->buildQuery($params);
        $data = $this->client->get("/knowledge-bases/{$knowledgeBaseId}/documents{$query}");
        $response = PaginatedResponse::fromArray($data);
        $response->data = array_map(fn($d) => Document::fromArray($d), $response->data);
        return $response;
    }

    public function getDocument(string $knowledgeBaseId, string $documentId): Document
    {
        $data = $this->client->get("/knowledge-bases/{$knowledgeBaseId}/documents/{$documentId}");
        return Document::fromArray($data);
    }

    public function addDocument(string $knowledgeBaseId, array $input): Document
    {
        $data = $this->client->post("/knowledge-bases/{$knowledgeBaseId}/documents", $input);
        return Document::fromArray($data);
    }

    public function updateDocument(string $knowledgeBaseId, string $documentId, array $input): Document
    {
        $data = $this->client->patch("/knowledge-bases/{$knowledgeBaseId}/documents/{$documentId}", $input);
        return Document::fromArray($data);
    }

    public function deleteDocument(string $knowledgeBaseId, string $documentId): void
    {
        $this->client->delete("/knowledge-bases/{$knowledgeBaseId}/documents/{$documentId}");
    }

    public function reprocessDocument(string $knowledgeBaseId, string $documentId): Document
    {
        $data = $this->client->post("/knowledge-bases/{$knowledgeBaseId}/documents/{$documentId}/reprocess", []);
        return Document::fromArray($data);
    }

    // Query operation
    public function query(string $knowledgeBaseId, array $input): array
    {
        return $this->client->post("/knowledge-bases/{$knowledgeBaseId}/query", $input);
    }

    private function buildQuery(array $params): string
    {
        if (empty($params)) {
            return '';
        }

        $parts = [];
        foreach ($params as $key => $value) {
            if ($value !== null) {
                $parts[] = urlencode($key) . '=' . urlencode((string) $value);
            }
        }

        return empty($parts) ? '' : '?' . implode('&', $parts);
    }
}
