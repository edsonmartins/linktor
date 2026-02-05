<?php

declare(strict_types=1);

namespace Linktor\Types;

class Agent
{
    public string $id = '';
    public string $name = '';
    public ?string $description = null;
    public ?string $systemPrompt = null;
    public string $model = '';
    public ?AgentConfig $config = null;
    public array $knowledgeBaseIds = [];
    public array $tools = [];
    public array $metadata = [];
    public ?\DateTimeImmutable $createdAt = null;
    public ?\DateTimeImmutable $updatedAt = null;

    public static function fromArray(array $data): self
    {
        $instance = new self();
        $instance->id = $data['id'] ?? '';
        $instance->name = $data['name'] ?? '';
        $instance->description = $data['description'] ?? null;
        $instance->systemPrompt = $data['systemPrompt'] ?? null;
        $instance->model = $data['model'] ?? '';
        $instance->config = isset($data['config']) ? AgentConfig::fromArray($data['config']) : null;
        $instance->knowledgeBaseIds = $data['knowledgeBaseIds'] ?? [];
        $instance->tools = array_map(fn($t) => AgentTool::fromArray($t), $data['tools'] ?? []);
        $instance->metadata = $data['metadata'] ?? [];
        $instance->createdAt = isset($data['createdAt']) ? new \DateTimeImmutable($data['createdAt']) : null;
        $instance->updatedAt = isset($data['updatedAt']) ? new \DateTimeImmutable($data['updatedAt']) : null;
        return $instance;
    }
}

class AgentConfig
{
    public float $temperature = 0.7;
    public int $maxTokens = 1024;
    public array $stopSequences = [];
    public int $topK = 50;
    public float $topP = 0.9;

    public static function fromArray(array $data): self
    {
        $instance = new self();
        $instance->temperature = $data['temperature'] ?? 0.7;
        $instance->maxTokens = $data['maxTokens'] ?? 1024;
        $instance->stopSequences = $data['stopSequences'] ?? [];
        $instance->topK = $data['topK'] ?? 50;
        $instance->topP = $data['topP'] ?? 0.9;
        return $instance;
    }
}

class AgentTool
{
    public string $name = '';
    public string $type = '';
    public ?string $description = null;
    public array $parameters = [];

    public static function fromArray(array $data): self
    {
        $instance = new self();
        $instance->name = $data['name'] ?? '';
        $instance->type = $data['type'] ?? '';
        $instance->description = $data['description'] ?? null;
        $instance->parameters = $data['parameters'] ?? [];
        return $instance;
    }
}

class ChatMessage
{
    public string $role = '';
    public string $content = '';

    public function __construct(string $role = '', string $content = '')
    {
        $this->role = $role;
        $this->content = $content;
    }

    public static function fromArray(array $data): self
    {
        return new self($data['role'] ?? '', $data['content'] ?? '');
    }

    public function toArray(): array
    {
        return [
            'role' => $this->role,
            'content' => $this->content,
        ];
    }
}
