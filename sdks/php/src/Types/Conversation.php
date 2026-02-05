<?php

declare(strict_types=1);

namespace Linktor\Types;

class Conversation
{
    public string $id = '';
    public string $channelId = '';
    public string $contactId = '';
    public string $status = '';
    public ?string $assignedTo = null;
    public ?string $subject = null;
    public ?Message $lastMessage = null;
    public int $messageCount = 0;
    public int $unreadCount = 0;
    public array $metadata = [];
    public ?\DateTimeImmutable $createdAt = null;
    public ?\DateTimeImmutable $updatedAt = null;
    public ?\DateTimeImmutable $resolvedAt = null;

    public static function fromArray(array $data): self
    {
        $instance = new self();
        $instance->id = $data['id'] ?? '';
        $instance->channelId = $data['channelId'] ?? '';
        $instance->contactId = $data['contactId'] ?? '';
        $instance->status = $data['status'] ?? '';
        $instance->assignedTo = $data['assignedTo'] ?? null;
        $instance->subject = $data['subject'] ?? null;
        $instance->lastMessage = isset($data['lastMessage']) ? Message::fromArray($data['lastMessage']) : null;
        $instance->messageCount = $data['messageCount'] ?? 0;
        $instance->unreadCount = $data['unreadCount'] ?? 0;
        $instance->metadata = $data['metadata'] ?? [];
        $instance->createdAt = isset($data['createdAt']) ? new \DateTimeImmutable($data['createdAt']) : null;
        $instance->updatedAt = isset($data['updatedAt']) ? new \DateTimeImmutable($data['updatedAt']) : null;
        $instance->resolvedAt = isset($data['resolvedAt']) ? new \DateTimeImmutable($data['resolvedAt']) : null;
        return $instance;
    }
}

class Message
{
    public string $id = '';
    public string $conversationId = '';
    public string $direction = '';
    public string $contentType = 'text';
    public ?string $text = null;
    public ?MessageMedia $media = null;
    public ?array $quickReplies = null;
    public string $status = '';
    public ?string $externalId = null;
    public array $metadata = [];
    public ?\DateTimeImmutable $createdAt = null;
    public ?\DateTimeImmutable $deliveredAt = null;
    public ?\DateTimeImmutable $readAt = null;

    public static function fromArray(array $data): self
    {
        $instance = new self();
        $instance->id = $data['id'] ?? '';
        $instance->conversationId = $data['conversationId'] ?? '';
        $instance->direction = $data['direction'] ?? '';
        $instance->contentType = $data['contentType'] ?? 'text';
        $instance->text = $data['text'] ?? null;
        $instance->media = isset($data['media']) ? MessageMedia::fromArray($data['media']) : null;
        $instance->quickReplies = $data['quickReplies'] ?? null;
        $instance->status = $data['status'] ?? '';
        $instance->externalId = $data['externalId'] ?? null;
        $instance->metadata = $data['metadata'] ?? [];
        $instance->createdAt = isset($data['createdAt']) ? new \DateTimeImmutable($data['createdAt']) : null;
        $instance->deliveredAt = isset($data['deliveredAt']) ? new \DateTimeImmutable($data['deliveredAt']) : null;
        $instance->readAt = isset($data['readAt']) ? new \DateTimeImmutable($data['readAt']) : null;
        return $instance;
    }
}

class MessageMedia
{
    public string $type = '';
    public string $url = '';
    public ?string $mimeType = null;
    public ?string $filename = null;
    public ?int $size = null;
    public ?int $width = null;
    public ?int $height = null;
    public ?int $duration = null;
    public ?string $thumbnailUrl = null;

    public static function fromArray(array $data): self
    {
        $instance = new self();
        $instance->type = $data['type'] ?? '';
        $instance->url = $data['url'] ?? '';
        $instance->mimeType = $data['mimeType'] ?? null;
        $instance->filename = $data['filename'] ?? null;
        $instance->size = $data['size'] ?? null;
        $instance->width = $data['width'] ?? null;
        $instance->height = $data['height'] ?? null;
        $instance->duration = $data['duration'] ?? null;
        $instance->thumbnailUrl = $data['thumbnailUrl'] ?? null;
        return $instance;
    }
}

class QuickReply
{
    public string $title = '';
    public string $payload = '';

    public static function fromArray(array $data): self
    {
        $instance = new self();
        $instance->title = $data['title'] ?? '';
        $instance->payload = $data['payload'] ?? '';
        return $instance;
    }
}
