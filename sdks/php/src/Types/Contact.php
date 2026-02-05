<?php

declare(strict_types=1);

namespace Linktor\Types;

class Contact
{
    public string $id = '';
    public ?string $name = null;
    public ?string $email = null;
    public ?string $phone = null;
    public ?string $externalId = null;
    public ?string $avatarUrl = null;
    public array $tags = [];
    public array $customFields = [];
    public array $metadata = [];
    public ?\DateTimeImmutable $createdAt = null;
    public ?\DateTimeImmutable $updatedAt = null;
    public ?\DateTimeImmutable $lastSeenAt = null;

    public static function fromArray(array $data): self
    {
        $instance = new self();
        $instance->id = $data['id'] ?? '';
        $instance->name = $data['name'] ?? null;
        $instance->email = $data['email'] ?? null;
        $instance->phone = $data['phone'] ?? null;
        $instance->externalId = $data['externalId'] ?? null;
        $instance->avatarUrl = $data['avatarUrl'] ?? null;
        $instance->tags = $data['tags'] ?? [];
        $instance->customFields = $data['customFields'] ?? [];
        $instance->metadata = $data['metadata'] ?? [];
        $instance->createdAt = isset($data['createdAt']) ? new \DateTimeImmutable($data['createdAt']) : null;
        $instance->updatedAt = isset($data['updatedAt']) ? new \DateTimeImmutable($data['updatedAt']) : null;
        $instance->lastSeenAt = isset($data['lastSeenAt']) ? new \DateTimeImmutable($data['lastSeenAt']) : null;
        return $instance;
    }
}
