<?php

declare(strict_types=1);

namespace Linktor\Resources;

use Linktor\LinktorClient;
use Linktor\Types\Conversation;
use Linktor\Types\Message;
use Linktor\Types\PaginatedResponse;

class ConversationsResource
{
    private LinktorClient $client;

    public function __construct(LinktorClient $client)
    {
        $this->client = $client;
    }

    public function list(array $params = []): PaginatedResponse
    {
        $query = $this->buildQuery($params);
        $data = $this->client->get("/conversations{$query}");
        $response = PaginatedResponse::fromArray($data);
        $response->data = array_map(fn($c) => Conversation::fromArray($c), $response->data);
        return $response;
    }

    public function get(string $id): Conversation
    {
        $data = $this->client->get("/conversations/{$id}");
        return Conversation::fromArray($data);
    }

    public function create(array $input): Conversation
    {
        $data = $this->client->post('/conversations', $input);
        return Conversation::fromArray($data);
    }

    public function update(string $id, array $input): Conversation
    {
        $data = $this->client->patch("/conversations/{$id}", $input);
        return Conversation::fromArray($data);
    }

    public function delete(string $id): void
    {
        $this->client->delete("/conversations/{$id}");
    }

    public function getMessages(string $conversationId, array $params = []): PaginatedResponse
    {
        $query = $this->buildQuery($params);
        $data = $this->client->get("/conversations/{$conversationId}/messages{$query}");
        $response = PaginatedResponse::fromArray($data);
        $response->data = array_map(fn($m) => Message::fromArray($m), $response->data);
        return $response;
    }

    public function sendMessage(string $conversationId, array $input): Message
    {
        $data = $this->client->post("/conversations/{$conversationId}/messages", $input);
        return Message::fromArray($data);
    }

    public function assign(string $id, string $userId): Conversation
    {
        $data = $this->client->post("/conversations/{$id}/assign", ['userId' => $userId]);
        return Conversation::fromArray($data);
    }

    public function resolve(string $id): Conversation
    {
        $data = $this->client->post("/conversations/{$id}/resolve", []);
        return Conversation::fromArray($data);
    }

    public function reopen(string $id): Conversation
    {
        $data = $this->client->post("/conversations/{$id}/reopen", []);
        return Conversation::fromArray($data);
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
