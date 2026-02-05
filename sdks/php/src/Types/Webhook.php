<?php

declare(strict_types=1);

namespace Linktor\Types;

class WebhookConstants
{
    public const SIGNATURE_HEADER = 'X-Linktor-Signature';
    public const TIMESTAMP_HEADER = 'X-Linktor-Timestamp';
    public const DEFAULT_TOLERANCE_SECONDS = 300;
}

enum EventType: string
{
    case MessageReceived = 'message.received';
    case MessageSent = 'message.sent';
    case MessageDelivered = 'message.delivered';
    case MessageRead = 'message.read';
    case MessageFailed = 'message.failed';
    case ConversationCreated = 'conversation.created';
    case ConversationUpdated = 'conversation.updated';
    case ConversationResolved = 'conversation.resolved';
    case ConversationAssigned = 'conversation.assigned';
    case ContactCreated = 'contact.created';
    case ContactUpdated = 'contact.updated';
    case ContactDeleted = 'contact.deleted';
    case ChannelConnected = 'channel.connected';
    case ChannelDisconnected = 'channel.disconnected';
    case ChannelError = 'channel.error';
    case BotStarted = 'bot.started';
    case BotStopped = 'bot.stopped';
    case FlowStarted = 'flow.started';
    case FlowCompleted = 'flow.completed';
    case FlowFailed = 'flow.failed';
}

class WebhookEvent
{
    public string $id = '';
    public string $type = '';
    public \DateTimeImmutable $timestamp;
    public string $tenantId = '';
    public array $data = [];

    public function __construct()
    {
        $this->timestamp = new \DateTimeImmutable();
    }

    public static function fromArray(array $data): self
    {
        $instance = new self();
        $instance->id = $data['id'] ?? '';
        $instance->type = $data['type'] ?? '';
        $instance->timestamp = isset($data['timestamp'])
            ? new \DateTimeImmutable($data['timestamp'])
            : new \DateTimeImmutable();
        $instance->tenantId = $data['tenantId'] ?? '';
        $instance->data = $data['data'] ?? [];
        return $instance;
    }

    public function getEventType(): ?EventType
    {
        return EventType::tryFrom($this->type);
    }
}

class WebhookConfig
{
    public string $url = '';
    public string $secret = '';
    /** @var string[] */
    public array $events = [];
    public bool $enabled = true;
    public array $headers = [];

    public static function fromArray(array $data): self
    {
        $instance = new self();
        $instance->url = $data['url'] ?? '';
        $instance->secret = $data['secret'] ?? '';
        $instance->events = $data['events'] ?? [];
        $instance->enabled = $data['enabled'] ?? true;
        $instance->headers = $data['headers'] ?? [];
        return $instance;
    }
}
