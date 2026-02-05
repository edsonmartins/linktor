<?php

declare(strict_types=1);

namespace Linktor\Types;

class Bot
{
    public string $id = '';
    public string $name = '';
    public ?string $description = null;
    public ?string $agentId = null;
    public ?string $flowId = null;
    public array $channelIds = [];
    public string $status = '';
    public ?BotConfig $config = null;
    public bool $enabled = true;
    public array $metadata = [];
    public ?\DateTimeImmutable $createdAt = null;
    public ?\DateTimeImmutable $updatedAt = null;

    public static function fromArray(array $data): self
    {
        $instance = new self();
        $instance->id = $data['id'] ?? '';
        $instance->name = $data['name'] ?? '';
        $instance->description = $data['description'] ?? null;
        $instance->agentId = $data['agentId'] ?? null;
        $instance->flowId = $data['flowId'] ?? null;
        $instance->channelIds = $data['channelIds'] ?? [];
        $instance->status = $data['status'] ?? '';
        $instance->config = isset($data['config']) ? BotConfig::fromArray($data['config']) : null;
        $instance->enabled = $data['enabled'] ?? true;
        $instance->metadata = $data['metadata'] ?? [];
        $instance->createdAt = isset($data['createdAt']) ? new \DateTimeImmutable($data['createdAt']) : null;
        $instance->updatedAt = isset($data['updatedAt']) ? new \DateTimeImmutable($data['updatedAt']) : null;
        return $instance;
    }
}

class BotConfig
{
    public ?string $welcomeMessage = null;
    public ?string $fallbackMessage = null;
    public int $handoffThreshold = 3;
    public bool $autoHandoff = true;
    public array $keywords = [];

    public static function fromArray(array $data): self
    {
        $instance = new self();
        $instance->welcomeMessage = $data['welcomeMessage'] ?? null;
        $instance->fallbackMessage = $data['fallbackMessage'] ?? null;
        $instance->handoffThreshold = $data['handoffThreshold'] ?? 3;
        $instance->autoHandoff = $data['autoHandoff'] ?? true;
        $instance->keywords = $data['keywords'] ?? [];
        return $instance;
    }
}

class BotStatus
{
    public string $id = '';
    public string $status = '';
    public bool $isRunning = false;
    public int $activeConversations = 0;
    public ?\DateTimeImmutable $startedAt = null;
    public ?string $errorMessage = null;

    public static function fromArray(array $data): self
    {
        $instance = new self();
        $instance->id = $data['id'] ?? '';
        $instance->status = $data['status'] ?? '';
        $instance->isRunning = $data['isRunning'] ?? false;
        $instance->activeConversations = $data['activeConversations'] ?? 0;
        $instance->startedAt = isset($data['startedAt']) ? new \DateTimeImmutable($data['startedAt']) : null;
        $instance->errorMessage = $data['errorMessage'] ?? null;
        return $instance;
    }
}
