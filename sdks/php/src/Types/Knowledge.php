<?php

declare(strict_types=1);

namespace Linktor\Types;

class KnowledgeBase
{
    public string $id = '';
    public string $name = '';
    public ?string $description = null;
    public string $embeddingModel = '';
    public ?KnowledgeBaseConfig $config = null;
    public int $documentCount = 0;
    public int $chunkCount = 0;
    public string $status = '';
    public array $metadata = [];
    public ?\DateTimeImmutable $createdAt = null;
    public ?\DateTimeImmutable $updatedAt = null;

    public static function fromArray(array $data): self
    {
        $instance = new self();
        $instance->id = $data['id'] ?? '';
        $instance->name = $data['name'] ?? '';
        $instance->description = $data['description'] ?? null;
        $instance->embeddingModel = $data['embeddingModel'] ?? '';
        $instance->config = isset($data['config']) ? KnowledgeBaseConfig::fromArray($data['config']) : null;
        $instance->documentCount = $data['documentCount'] ?? 0;
        $instance->chunkCount = $data['chunkCount'] ?? 0;
        $instance->status = $data['status'] ?? '';
        $instance->metadata = $data['metadata'] ?? [];
        $instance->createdAt = isset($data['createdAt']) ? new \DateTimeImmutable($data['createdAt']) : null;
        $instance->updatedAt = isset($data['updatedAt']) ? new \DateTimeImmutable($data['updatedAt']) : null;
        return $instance;
    }
}

class KnowledgeBaseConfig
{
    public int $chunkSize = 512;
    public int $chunkOverlap = 50;
    public string $splitter = 'recursive';

    public static function fromArray(array $data): self
    {
        $instance = new self();
        $instance->chunkSize = $data['chunkSize'] ?? 512;
        $instance->chunkOverlap = $data['chunkOverlap'] ?? 50;
        $instance->splitter = $data['splitter'] ?? 'recursive';
        return $instance;
    }
}

class Document
{
    public string $id = '';
    public string $knowledgeBaseId = '';
    public ?string $title = null;
    public ?string $content = null;
    public ?string $url = null;
    public ?string $fileType = null;
    public string $status = '';
    public int $chunkCount = 0;
    public array $metadata = [];
    public ?\DateTimeImmutable $createdAt = null;
    public ?\DateTimeImmutable $updatedAt = null;
    public ?\DateTimeImmutable $processedAt = null;

    public static function fromArray(array $data): self
    {
        $instance = new self();
        $instance->id = $data['id'] ?? '';
        $instance->knowledgeBaseId = $data['knowledgeBaseId'] ?? '';
        $instance->title = $data['title'] ?? null;
        $instance->content = $data['content'] ?? null;
        $instance->url = $data['url'] ?? null;
        $instance->fileType = $data['fileType'] ?? null;
        $instance->status = $data['status'] ?? '';
        $instance->chunkCount = $data['chunkCount'] ?? 0;
        $instance->metadata = $data['metadata'] ?? [];
        $instance->createdAt = isset($data['createdAt']) ? new \DateTimeImmutable($data['createdAt']) : null;
        $instance->updatedAt = isset($data['updatedAt']) ? new \DateTimeImmutable($data['updatedAt']) : null;
        $instance->processedAt = isset($data['processedAt']) ? new \DateTimeImmutable($data['processedAt']) : null;
        return $instance;
    }
}
