<?php

declare(strict_types=1);

namespace Linktor\Types;

class PaginatedResponse
{
    /** @var array<mixed> */
    public array $data = [];
    public ?string $nextCursor = null;
    public ?string $prevCursor = null;
    public ?int $total = null;
    public bool $hasMore = false;

    public static function fromArray(array $data): self
    {
        $instance = new self();
        $instance->data = $data['data'] ?? [];
        $instance->nextCursor = $data['nextCursor'] ?? null;
        $instance->prevCursor = $data['prevCursor'] ?? null;
        $instance->total = $data['total'] ?? null;
        $instance->hasMore = $data['hasMore'] ?? false;
        return $instance;
    }
}

class ApiResponse
{
    public bool $success = false;
    public mixed $data = null;
    public ?string $message = null;

    public static function fromArray(array $data): self
    {
        $instance = new self();
        $instance->success = $data['success'] ?? false;
        $instance->data = $data['data'] ?? null;
        $instance->message = $data['message'] ?? null;
        return $instance;
    }
}

class ListParams
{
    public ?int $limit = null;
    public ?int $offset = null;
    public ?string $cursor = null;

    public function toQuery(): array
    {
        $query = [];
        if ($this->limit !== null) $query['limit'] = $this->limit;
        if ($this->offset !== null) $query['offset'] = $this->offset;
        if ($this->cursor !== null) $query['cursor'] = $this->cursor;
        return $query;
    }
}

class TokenUsage
{
    public int $promptTokens = 0;
    public int $completionTokens = 0;
    public int $totalTokens = 0;

    public static function fromArray(array $data): self
    {
        $instance = new self();
        $instance->promptTokens = $data['promptTokens'] ?? 0;
        $instance->completionTokens = $data['completionTokens'] ?? 0;
        $instance->totalTokens = $data['totalTokens'] ?? 0;
        return $instance;
    }
}
