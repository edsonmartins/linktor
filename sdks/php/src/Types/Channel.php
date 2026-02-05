<?php

declare(strict_types=1);

namespace Linktor\Types;

class Channel
{
    public string $id = '';
    public string $name = '';
    public string $type = '';
    public string $status = '';
    public array $config = [];
    public bool $enabled = true;
    public array $metadata = [];
    public ?\DateTimeImmutable $createdAt = null;
    public ?\DateTimeImmutable $updatedAt = null;
    public ?\DateTimeImmutable $connectedAt = null;

    public static function fromArray(array $data): self
    {
        $instance = new self();
        $instance->id = $data['id'] ?? '';
        $instance->name = $data['name'] ?? '';
        $instance->type = $data['type'] ?? '';
        $instance->status = $data['status'] ?? '';
        $instance->config = $data['config'] ?? [];
        $instance->enabled = $data['enabled'] ?? true;
        $instance->metadata = $data['metadata'] ?? [];
        $instance->createdAt = isset($data['createdAt']) ? new \DateTimeImmutable($data['createdAt']) : null;
        $instance->updatedAt = isset($data['updatedAt']) ? new \DateTimeImmutable($data['updatedAt']) : null;
        $instance->connectedAt = isset($data['connectedAt']) ? new \DateTimeImmutable($data['connectedAt']) : null;
        return $instance;
    }
}

class ChannelStatus
{
    public string $id = '';
    public string $status = '';
    public bool $isConnected = false;
    public ?\DateTimeImmutable $lastActivityAt = null;
    public ?string $errorMessage = null;
    public array $details = [];

    public static function fromArray(array $data): self
    {
        $instance = new self();
        $instance->id = $data['id'] ?? '';
        $instance->status = $data['status'] ?? '';
        $instance->isConnected = $data['isConnected'] ?? false;
        $instance->lastActivityAt = isset($data['lastActivityAt']) ? new \DateTimeImmutable($data['lastActivityAt']) : null;
        $instance->errorMessage = $data['errorMessage'] ?? null;
        $instance->details = $data['details'] ?? [];
        return $instance;
    }
}
