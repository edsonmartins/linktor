<?php

declare(strict_types=1);

namespace Linktor\Types;

class Flow
{
    public string $id = '';
    public string $name = '';
    public ?string $description = null;
    /** @var FlowNode[] */
    public array $nodes = [];
    /** @var FlowEdge[] */
    public array $edges = [];
    public ?FlowSettings $settings = null;
    public string $status = '';
    public ?string $publishedVersion = null;
    public array $metadata = [];
    public ?\DateTimeImmutable $createdAt = null;
    public ?\DateTimeImmutable $updatedAt = null;
    public ?\DateTimeImmutable $publishedAt = null;

    public static function fromArray(array $data): self
    {
        $instance = new self();
        $instance->id = $data['id'] ?? '';
        $instance->name = $data['name'] ?? '';
        $instance->description = $data['description'] ?? null;
        $instance->nodes = array_map(fn($n) => FlowNode::fromArray($n), $data['nodes'] ?? []);
        $instance->edges = array_map(fn($e) => FlowEdge::fromArray($e), $data['edges'] ?? []);
        $instance->settings = isset($data['settings']) ? FlowSettings::fromArray($data['settings']) : null;
        $instance->status = $data['status'] ?? '';
        $instance->publishedVersion = $data['publishedVersion'] ?? null;
        $instance->metadata = $data['metadata'] ?? [];
        $instance->createdAt = isset($data['createdAt']) ? new \DateTimeImmutable($data['createdAt']) : null;
        $instance->updatedAt = isset($data['updatedAt']) ? new \DateTimeImmutable($data['updatedAt']) : null;
        $instance->publishedAt = isset($data['publishedAt']) ? new \DateTimeImmutable($data['publishedAt']) : null;
        return $instance;
    }
}

class FlowNode
{
    public string $id = '';
    public string $type = '';
    public ?FlowNodePosition $position = null;
    public array $data = [];

    public static function fromArray(array $data): self
    {
        $instance = new self();
        $instance->id = $data['id'] ?? '';
        $instance->type = $data['type'] ?? '';
        $instance->position = isset($data['position']) ? FlowNodePosition::fromArray($data['position']) : null;
        $instance->data = $data['data'] ?? [];
        return $instance;
    }

    public function toArray(): array
    {
        return [
            'id' => $this->id,
            'type' => $this->type,
            'position' => $this->position?->toArray(),
            'data' => $this->data,
        ];
    }
}

class FlowNodePosition
{
    public float $x = 0;
    public float $y = 0;

    public static function fromArray(array $data): self
    {
        $instance = new self();
        $instance->x = $data['x'] ?? 0;
        $instance->y = $data['y'] ?? 0;
        return $instance;
    }

    public function toArray(): array
    {
        return ['x' => $this->x, 'y' => $this->y];
    }
}

class FlowEdge
{
    public string $id = '';
    public string $source = '';
    public string $target = '';
    public ?string $sourceHandle = null;
    public ?string $targetHandle = null;
    public ?string $label = null;

    public static function fromArray(array $data): self
    {
        $instance = new self();
        $instance->id = $data['id'] ?? '';
        $instance->source = $data['source'] ?? '';
        $instance->target = $data['target'] ?? '';
        $instance->sourceHandle = $data['sourceHandle'] ?? null;
        $instance->targetHandle = $data['targetHandle'] ?? null;
        $instance->label = $data['label'] ?? null;
        return $instance;
    }

    public function toArray(): array
    {
        return [
            'id' => $this->id,
            'source' => $this->source,
            'target' => $this->target,
            'sourceHandle' => $this->sourceHandle,
            'targetHandle' => $this->targetHandle,
            'label' => $this->label,
        ];
    }
}

class FlowSettings
{
    public ?int $timeout = null;
    public int $maxRetries = 3;
    public bool $enableLogging = true;

    public static function fromArray(array $data): self
    {
        $instance = new self();
        $instance->timeout = $data['timeout'] ?? null;
        $instance->maxRetries = $data['maxRetries'] ?? 3;
        $instance->enableLogging = $data['enableLogging'] ?? true;
        return $instance;
    }
}

class FlowExecution
{
    public string $id = '';
    public string $flowId = '';
    public string $status = '';
    public ?string $conversationId = null;
    public ?string $contactId = null;
    public array $variables = [];
    public ?string $currentNodeId = null;
    public array $executionLog = [];
    public ?string $errorMessage = null;
    public ?\DateTimeImmutable $startedAt = null;
    public ?\DateTimeImmutable $completedAt = null;

    public static function fromArray(array $data): self
    {
        $instance = new self();
        $instance->id = $data['id'] ?? '';
        $instance->flowId = $data['flowId'] ?? '';
        $instance->status = $data['status'] ?? '';
        $instance->conversationId = $data['conversationId'] ?? null;
        $instance->contactId = $data['contactId'] ?? null;
        $instance->variables = $data['variables'] ?? [];
        $instance->currentNodeId = $data['currentNodeId'] ?? null;
        $instance->executionLog = $data['executionLog'] ?? [];
        $instance->errorMessage = $data['errorMessage'] ?? null;
        $instance->startedAt = isset($data['startedAt']) ? new \DateTimeImmutable($data['startedAt']) : null;
        $instance->completedAt = isset($data['completedAt']) ? new \DateTimeImmutable($data['completedAt']) : null;
        return $instance;
    }
}
